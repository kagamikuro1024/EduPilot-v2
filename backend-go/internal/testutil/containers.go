// Package testutil: container thật (Postgres 18 + pgvector, Redis 8, MinIO) dùng chung trong MỘT tiến trình test.
// Mỗi container khởi động tối đa một lần (sync.Once); mỗi test tự lấy dữ liệu riêng (database mới, tiền tố khoá Redis riêng).
// Không Docker thì test FAIL (SRS 9.1), trừ `go test -short` (test gọi RequireContainers sẽ bỏ qua).
package testutil

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

const (
	postgresImage = "pgvector/pgvector:pg18"
	redisImage    = "redis:8"
	minioImage    = "pgsty/minio:RELEASE.2026-08-04T00-00-00Z"

	// MinIO credentials dùng cho container thử.
	MinIOAccessKey = "edupilot-test"
	MinIOSecretKey = "edupilot-test-secret"
)

// RequireContainers bỏ qua test khi chạy `go test -short`; ngược lại để test chạy và tự fail nếu thiếu Docker.
func RequireContainers(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("cần container (bỏ qua với -short)")
	}
}

var (
	pgOnce  sync.Once
	pgBase  string // URL tới database `postgres` của container
	pgErr   error
	dbSeq   int64
	dbSeqMu sync.Mutex

	redisOnce sync.Once
	redisURL  string
	redisErr  error

	minioOnce     sync.Once
	minioEndpoint string
	minioErr      error
)

func startCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Minute)
}

// PostgresURL trả URL (sslmode=disable) tới MỘT database mới, trống, trong container Postgres dùng chung.
func PostgresURL(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	pgOnce.Do(func() {
		ctx, cancel := startCtx()
		defer cancel()
		c, err := tcpostgres.Run(ctx, postgresImage,
			tcpostgres.WithDatabase("postgres"), tcpostgres.WithUsername("edupilot"), tcpostgres.WithPassword("edupilot-dev"),
			testcontainers.WithCmdArgs("-c", "max_connections=300"),
			tcpostgres.BasicWaitStrategies())
		if err != nil {
			pgErr = fmt.Errorf("start postgres: %w", err)
			return
		}
		pgBase, pgErr = c.ConnectionString(ctx, "sslmode=disable")
	})
	if pgErr != nil {
		t.Fatalf("postgres container: %v", pgErr)
	}
	dbSeqMu.Lock()
	dbSeq++
	name := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), dbSeq)
	dbSeqMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, pgBase)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	return strings.Replace(pgBase, "/postgres?", "/"+name+"?", 1)
}

// PostgresHostPort trả địa chỉ host:port của container Postgres dùng chung (cho PgBouncer container nối vào).
func PostgresHostPort(t testing.TB) string {
	t.Helper()
	u := PostgresURL(t)
	rest := strings.TrimPrefix(u, "postgres://")
	at := strings.Index(rest, "@")
	slash := strings.Index(rest[at:], "/")
	return rest[at+1 : at+slash]
}

// RedisURL trả `redis://host:port/0` của container Redis dùng chung. Test PHẢI dùng tiền tố khoá riêng (TestPrefix) hoặc
// DB số khác nhau để song song được; không FLUSHALL.
func RedisURL(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	redisOnce.Do(func() {
		ctx, cancel := startCtx()
		defer cancel()
		c, err := tcredis.Run(ctx, redisImage)
		if err != nil {
			redisErr = fmt.Errorf("start redis: %w", err)
			return
		}
		redisURL, redisErr = c.ConnectionString(ctx)
	})
	if redisErr != nil {
		t.Fatalf("redis container: %v", redisErr)
	}
	return redisURL
}

// MinIOEndpoint trả `host:port` (không scheme) của container MinIO dùng chung, kèm khoá truy cập hằng ở trên.
func MinIOEndpoint(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	minioOnce.Do(func() {
		ctx, cancel := startCtx()
		defer cancel()
		c, err := tcminio.Run(ctx, minioImage, tcminio.WithUsername(MinIOAccessKey), tcminio.WithPassword(MinIOSecretKey))
		if err != nil {
			minioErr = fmt.Errorf("start minio: %w", err)
			return
		}
		minioEndpoint, minioErr = c.ConnectionString(ctx)
	})
	if minioErr != nil {
		t.Fatalf("minio container: %v", minioErr)
	}
	return minioEndpoint
}

// TestPrefix sinh tiền tố duy nhất cho một test (dùng làm user id / tên stream để các test không đạp nhau).
func TestPrefix(t testing.TB) string {
	t.Helper()
	dbSeqMu.Lock()
	defer dbSeqMu.Unlock()
	dbSeq++
	return fmt.Sprintf("t%d%d", time.Now().UnixNano()%1e9, dbSeq)
}
