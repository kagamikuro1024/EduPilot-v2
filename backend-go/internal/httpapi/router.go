// Package httpapi: router chi, middleware và handler HTTP của gateway.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// errNoDependency: Deps chưa có DB/Redis (chỉ xảy ra trong test) — coi như phụ thuộc chết.
var errNoDependency = errors.New("phụ thuộc chưa được cấu hình")

// readyProbeTimeout giới hạn thời gian ping DB/Redis của /readyz để endpoint này không bao giờ treo.
const readyProbeTimeout = 2 * time.Second

// Deps là mọi thứ handler cần. Không có biến toàn cục: tất cả đi qua đây.
// Verifier / Publisher / Jobs để trống thì NewRouter tự dựng từ Cfg + DB + Redis.
type Deps struct {
	Cfg       config.Config
	Log       *slog.Logger
	DB        *pgxpool.Pool
	Redis     *redis.Client
	Clock     clock.Clock
	State     *State
	Verifier  *auth.Verifier
	Publisher sse.Publisher
	Jobs      *jobs.Service
	LLM       *llm.Runtime
}

// now là đồng hồ của request (Clock trống → đồng hồ hệ thống).
func (d Deps) now() time.Time {
	if d.Clock == nil {
		return clock.Real{}.Now()
	}
	return d.Clock.Now()
}

// State giữ trạng thái vòng đời của tiến trình (đang tắt, số request đang chạy).
type State struct {
	draining atomic.Bool
	inflight atomic.Int64
	drainC   chan struct{}
	drainOne sync.Once
}

// NewState tạo trạng thái cho một tiến trình gateway.
func NewState() *State { return &State{drainC: make(chan struct{})} }

// Drain đánh dấu bắt đầu tắt: readyz trả 503 ngay, SSE nhận tín hiệu qua Draining().
func (s *State) Drain() {
	s.draining.Store(true)
	s.drainOne.Do(func() { close(s.drainC) })
}

// Draining cho biết tiến trình đang tắt.
func (s *State) Draining() bool { return s.draining.Load() }

// DrainC đóng khi bắt đầu tắt (handler dài như SSE select trên kênh này để gửi `event: shutdown`).
func (s *State) DrainC() <-chan struct{} { return s.drainC }

// InFlight là số request đang chạy (dùng để báo `cancelled_requests` khi phải cắt lúc tắt).
func (s *State) InFlight() int64 { return s.inflight.Load() }

func (s *State) enter() { s.inflight.Add(1) }
func (s *State) leave() { s.inflight.Add(-1) }

// NewRouter dựng router gateway. Thứ tự middleware cố định theo SRS 3.1:
// otelhttp (trace + request-id) → recover → access log → CORS → timeout + giới hạn thân →
// rate limit → auth/RBAC/CourseAccessGuard → Idempotency-Key → handler.
// Health đi qua recover + access log nhưng KHÔNG qua rate limit và không có deadline của nhóm nghiệp vụ.
func NewRouter(d Deps) http.Handler {
	d = withDefaults(d)
	return newRouterWith(d, func(r chi.Router) { registerAPIRoutes(r, d) })
}

// withDefaults dựng các phụ thuộc suy ra được từ Cfg/DB/Redis khi người gọi chưa đặt.
func withDefaults(d Deps) Deps {
	if d.Clock == nil {
		d.Clock = clock.Real{}
	}
	if d.Verifier == nil {
		d.Verifier = auth.NewVerifier(d.Cfg.JWTSecretKey, d.Clock)
	}
	if d.Publisher == nil && d.Redis != nil {
		d.Publisher = sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL)
	}
	if d.Jobs == nil && d.DB != nil {
		d.Jobs = jobs.NewService(d.DB)
	}
	return d
}

// newRouterWith dựng router với một hàm đăng ký route của nhóm nghiệp vụ (test mount handler tạm).
func newRouterWith(d Deps, mount func(chi.Router)) http.Handler {
	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.ByStatus(http.StatusNotFound))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		apierr.Write(w, r, apierr.ByStatus(http.StatusMethodNotAllowed))
	})

	r.Use(traceMiddleware(d))
	r.Use(requestIDMiddleware(d))
	r.Use(recoverMiddleware(d))
	r.Use(accessLogMiddleware(d))
	r.Use(drainingMiddleware(d))
	r.Use(corsMiddleware(d))

	r.Get("/healthz", healthzHandler)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", healthzHandler)
		r.Get("/readyz", readyzHandler(d))

		// SSE nằm NGOÀI nhóm nghiệp vụ: stream sống lâu hơn REQUEST_TIMEOUT nên không qua
		// timeout/body limit; handler tự xác thực bằng Verifier (SRS 6.8).
		r.Method(http.MethodGet, "/events", sse.NewHandler(sse.HandlerDeps{
			Cfg: d.Cfg, Log: d.Log, Redis: d.Redis, Verifier: d.Verifier, Clock: d.Clock, Drain: drainC(d),
		}))

		// Nhóm nghiệp vụ: deadline mỗi request + giới hạn thân, rồi rate limit → auth → Idempotency-Key
		// (SRS 3.1: 413 chạy trước kiểm Idempotency-Key, xác thực chạy trước RBAC/guard).
		r.Group(func(r chi.Router) {
			r.Use(timeoutMiddleware(d))
			r.Use(bodyLimitMiddleware(d))
			mount(r)
		})
	})
	return r
}

// drainC là kênh "bắt đầu tắt" cho handler dài (SSE); State trống → kênh nil (không bao giờ bắn).
func drainC(d Deps) <-chan struct{} {
	if d.State == nil {
		return nil
	}
	return d.State.DrainC()
}

// healthzHandler chỉ kiểm sống: không gọi DB/Redis.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyzHandler ping DB + Redis với hạn ngắn; đang tắt hoặc phụ thuộc chết → 503 NOT_READY (SRS 6.1).
func readyzHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyProbeTimeout)
		defer cancel()

		dbState, redisState := upDown(pingDB(ctx, d)), upDown(pingRedis(ctx, d))
		draining := d.State != nil && d.State.Draining()
		if draining || dbState != "up" || redisState != "up" {
			apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.NotReady).
				WithDetails(map[string]any{"db": dbState, "redis": redisState, "draining": draining}))
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "db": dbState, "redis": redisState})
	}
}

func upDown(err error) string {
	if err != nil {
		return "down"
	}
	return "up"
}

func pingDB(ctx context.Context, d Deps) error {
	if d.DB == nil {
		return errNoDependency
	}
	return d.DB.Ping(ctx)
}

func pingRedis(ctx context.Context, d Deps) error {
	if d.Redis == nil {
		return errNoDependency
	}
	return d.Redis.Ping(ctx).Err()
}
