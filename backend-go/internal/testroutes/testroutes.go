//go:build testroutes

// Package testroutes: endpoint thử `/api/v1/_test/…` dùng để kiểm các helper HTTP (SRS 6.3).
// MỌI file của gói có `//go:build testroutes`, nên binary / image mặc định không liên kết gói này
// (`go tool nm` không có symbol, chuỗi `/api/v1/_test/` không nằm trong file thực thi — 03-AC18).
package testroutes

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaTimeout là hạn tạo bảng thử lúc khởi động.
const schemaTimeout = 10 * time.Second

// Deps là phụ thuộc của route thử; `internal/httpapi` dựng từ `httpapi.Deps` ở bản dựng có build tag.
type Deps struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	Log       *slog.Logger
	Clock     clock.Clock
	Verifier  *auth.Verifier
	Publisher sse.Publisher
	Jobs      *jobs.Service
	LLM       *llmrt.Runtime
	// RequireIdempotencyKey là middleware `Idempotency-Key` bắt buộc của httpapi (SRS 6.6).
	RequireIdempotencyKey func(http.Handler) http.Handler
}

// Register tạo bảng thử rồi đăng ký 15 thao tác của SRS 6.3 dưới `/api/v1/_test`.
// r là router của nhóm nghiệp vụ (đã có deadline, giới hạn thân, rate limit).
func Register(r chi.Router, d Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), schemaTimeout)
	defer cancel()
	if err := ensureSchema(ctx, d); err != nil && d.Log != nil {
		d.Log.ErrorContext(ctx, "không tạo được bảng thử _test_items", "error", err.Error())
	}

	r.Route("/_test", func(r chi.Router) {
		// Ẩn danh (SRS 6.3 mục 5–9).
		r.Get("/slow", slowHandler)
		r.Get("/db-sleep", dbSleepHandler(d))
		r.Get("/redis-block", redisBlockHandler(d))
		r.Get("/panic", panicHandler)
		r.Get("/error/{status}", errorHandler)

		// Cần đăng nhập: xác thực chạy TRƯỚC RBAC / guard / Idempotency-Key (SRS 3.1).
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(d.Verifier))
			r.Get("/items", listItems(d))
			r.With(d.RequireIdempotencyKey).Post("/items", createItem(d))
			r.Get("/items/{id}", getItem(d))
			r.Put("/items/{id}", putItem(d))
			r.Get("/whoami", whoami)
			r.With(auth.RequireRole(auth.RoleAdmin)).Get("/rbac/admin", pingHandler)
			r.With(auth.RequireRole(auth.RoleTeacher, auth.RoleTA)).Get("/rbac/staff", pingHandler)
			r.With(auth.CourseAccessGuard(nil, auth.Member)).Get("/courses/{courseId}/ping", pingHandler)
			r.Post("/jobs", createJob(d))
			r.Post("/events", publishEvent(d))
			registerLLM(r, d)
		})
	})
}

// schemaSQL dựng bảng thử của SRS 6.3 (không nằm trong migration: chỉ có ở bản dựng có build tag).
const schemaSQL = `
create table if not exists _test_items (
	id         uuid        primary key default uuidv7(),
	name       text        not null,
	version    integer     not null default 1,
	owner_id   uuid        not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create index if not exists _test_items_created_at_id_idx on _test_items (created_at, id);`

func ensureSchema(ctx context.Context, d Deps) error {
	if d.DB == nil {
		return nil
	}
	_, err := d.DB.Exec(ctx, schemaSQL)
	return err
}

// pingHandler là thân phản hồi chung của các route chỉ kiểm phân quyền.
func pingHandler(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
