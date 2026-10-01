// Package db: pool pgx dùng chung (SRS 4.1 FR-10, FR-11).
// QueryExecModeExec để chạy được qua PgBouncer transaction mode; span otelpgx; log `slow query`;
// AfterConnect đăng ký kiểu vector/halfvec khi extension đã có.
package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Giới hạn vòng đời kết nối (SRS 4.1 FR-10).
const (
	maxConnLifetime   = 30 * time.Minute
	maxConnIdleTime   = 5 * time.Minute
	healthCheckPeriod = 30 * time.Second
)

// NewPool dựng pool pgx theo cấu hình. Không nối ngay: lần Acquire đầu mới mở kết nối,
// nên lỗi "DB chưa lên" do bước chờ phụ thuộc lúc khởi động phát hiện.
func NewPool(ctx context.Context, cfg config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.RuntimeDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL không hợp lệ: %w", err)
	}
	pc.MaxConns = cfg.DBMaxConns
	pc.MaxConnLifetime = maxConnLifetime
	pc.MaxConnIdleTime = maxConnIdleTime
	pc.HealthCheckPeriod = healthCheckPeriod
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.ConnConfig.RuntimeParams["application_name"] = "edupilot-" + cfg.Service()
	pc.ConnConfig.Tracer = &tracer{
		otel: otelpgx.NewTracer(),
		log:  logger,
		slow: time.Duration(cfg.DBSlowQueryMS) * time.Millisecond,
	}
	pc.AfterConnect = registerVectorTypes

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("dựng pool Postgres: %w", err)
	}
	return pool, nil
}

// registerVectorTypes đăng ký kiểu của extension vector cho kết nối mới.
// Extension chưa cài (DB trống trước migration) thì bỏ qua, không làm hỏng kết nối.
func registerVectorTypes(ctx context.Context, conn *pgx.Conn) error {
	for _, name := range []string{"vector", "halfvec", "sparsevec"} {
		t, err := conn.LoadType(ctx, name)
		if err != nil {
			continue
		}
		conn.TypeMap().RegisterType(t)
	}
	return nil
}

// tracer ghép span otelpgx với log truy vấn chậm (không log giá trị tham số).
type tracer struct {
	otel *otelpgx.Tracer
	log  *slog.Logger
	slow time.Duration
}

type startKey struct{}

type started struct {
	at   time.Time
	name string
}

func (t *tracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.otel.TraceQueryStart(ctx, conn, data)
	return context.WithValue(ctx, startKey{}, started{at: time.Now(), name: QueryName(data.SQL)})
}

func (t *tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	t.otel.TraceQueryEnd(ctx, conn, data)
	s, ok := ctx.Value(startKey{}).(started)
	if !ok || t.log == nil {
		return
	}
	if d := time.Since(s.at); d >= t.slow {
		t.log.WarnContext(ctx, "slow query", "query_name", s.name, "duration_ms", d.Milliseconds())
	}
}

// QueryName lấy tên truy vấn sqlc từ dòng `-- name: <Tên> :one`; không có thì dùng từ khoá đầu câu lệnh.
// Không bao giờ trả về giá trị tham số (tham số luôn là $1, $2… trong SQL).
func QueryName(sql string) string {
	for line := range strings.Lines(sql) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "-- name:"); ok {
			if name, _, _ := strings.Cut(strings.TrimSpace(rest), " "); name != "" {
				return name
			}
		}
		break
	}
	if word, _, _ := strings.Cut(strings.TrimSpace(sql), " "); word != "" {
		return strings.ToLower(word)
	}
	return "unnamed"
}
