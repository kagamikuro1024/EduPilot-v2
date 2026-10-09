// Package testutil: container thật (Postgres 18 + pgvector, Redis 8, MinIO) dùng chung cho MỌI gói test trên máy.
//
// Mỗi gói `go test` là một tiến trình riêng; nếu mỗi tiến trình tự dựng container thì `go test ./...` mở hàng chục
// Postgres/Redis cùng lúc và đói CPU (khởi động quá hạn). Vì vậy container có TÊN CỐ ĐỊNH và được dùng lại giữa các tiến trình
// (khoá tệp `flock` để chỉ một tiến trình tạo). Mỗi test lấy dữ liệu riêng: database mới (xoá khi test xong), id người dùng
// ngẫu nhiên trên Redis, bucket/khoá riêng trên MinIO. Không test nào được FLUSHALL hay giết container dùng chung.
//
// Dọn container: `make -C backend-go test-clean`. Không Docker thì test FAIL (SRS 9.1), trừ `go test -short`.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage = "pgvector/pgvector:pg18"
	redisImage    = "redis:8"
	minioImage    = "pgsty/minio:RELEASE.2026-08-04T00-00-00Z"

	pgName    = "edupilot-test-postgres"
	redisName = "edupilot-test-redis"
	minioName = "edupilot-test-minio"

	mailpitImage = "axllent/mailpit:v1.27"
	mailpitName  = "edupilot-test-mailpit"

	// MinIO credentials dùng cho container thử.
	MinIOAccessKey = "edupilot-test"
	MinIOSecretKey = "edupilot-test-secret"

	pgUser = "edupilot"
	pgPass = "edupilot-dev"
)

// RequireContainers bỏ qua test khi chạy `go test -short`; ngược lại để test chạy và tự fail nếu thiếu Docker.
func RequireContainers(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("cần container (bỏ qua với -short)")
	}
}

// SkipTiming bỏ qua test đo thời gian (so trung vị mili-giây giữa các nhánh) khi EP_SKIP_TIMING=1. CI đặt biến này ở bước `go test ./...` (nhiều gói chạy song song
// nên nhiễu vượt ngưỡng) và chạy riêng các test đó, tuần tự, ở bước kế tiếp (TL-3). Không dùng `-short`: cờ đó bỏ qua MỌI test cần container.
func SkipTiming(t testing.TB) {
	t.Helper()
	if os.Getenv("EP_SKIP_TIMING") == "1" {
		t.Skip("đo thời gian: chạy riêng, tuần tự (EP_SKIP_TIMING=1)")
	}
}

var (
	pgOnce   sync.Once
	pgHost   string // host:port trực tiếp tới Postgres
	pgErr    error
	seqMu    sync.Mutex
	seq      int64
	redisOne sync.Once
	redisURL string
	redisErr error
	minioOne sync.Once
	minioEP  string
	minioErr error
	mpOnce   sync.Once
	mpSMTP   string
	mpAPI    string
	mpErr    error
)

// withLock chạy fn khi giữ khoá tệp toàn máy (nhiều tiến trình test không cùng tạo một container).
// Dùng syscall thuần: gói này là hạ tầng test, không phải mã sản xuất ghi đĩa (US-PG-07 AC11b).
func withLock(fn func() error) error {
	fd, err := syscall.Open(filepath.Join(os.TempDir(), "edupilot-testutil.lock"), syscall.O_CREAT|syscall.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	defer func() { _ = syscall.Close(fd) }()
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer func() { _ = syscall.Flock(fd, syscall.LOCK_UN) }()
	return fn()
}

// ensure tạo (hoặc dùng lại) container tên cố định và trả host:port của cổng port.
func ensure(name string, req testcontainers.ContainerRequest, port string, ready func(hostPort string) error) (string, error) {
	var hostPort string
	err := withLock(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		req.Name = name
		c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true, Reuse: true})
		if err != nil {
			return fmt.Errorf("start %s: %w", name, err)
		}
		host, err := c.Host(ctx)
		if err != nil {
			return fmt.Errorf("host %s: %w", name, err)
		}
		mp, err := c.MappedPort(ctx, port)
		if err != nil {
			return fmt.Errorf("port %s: %w", name, err)
		}
		hostPort = host + ":" + mp.Port()
		// Sẵn sàng thật (không dựa vào dòng log): thử nối tới khi được.
		deadline := time.Now().Add(2 * time.Minute)
		var last error
		for time.Now().Before(deadline) {
			if last = ready(hostPort); last == nil {
				return nil
			}
			time.Sleep(300 * time.Millisecond)
		}
		return fmt.Errorf("%s chưa sẵn sàng: %w", name, last)
	})
	return hostPort, err
}

func pgDSN(hostPort, db string) string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", pgUser, pgPass, hostPort, db)
}

// purgeOldDatabases xoá database test cũ hơn 30 phút (tên t_<unix>_<pid>_<seq>).
func purgeOldDatabases(hostPort string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := pgx.Connect(ctx, pgDSN(hostPort, "postgres"))
	if err != nil {
		return
	}
	defer func() { _ = c.Close(ctx) }()
	rows, err := c.Query(ctx, "SELECT datname FROM pg_database WHERE datname LIKE 't\\_%' ")
	if err != nil {
		return
	}
	var old []string
	cutoff := time.Now().Add(-30 * time.Minute).Unix()
	for rows.Next() {
		var n string
		var ts int64
		if rows.Scan(&n) == nil {
			if _, err := fmt.Sscanf(n, "t_%d_", &ts); err == nil && ts < cutoff {
				old = append(old, n)
			}
		}
	}
	rows.Close()
	for _, n := range old {
		_, _ = c.Exec(ctx, "DROP DATABASE IF EXISTS "+n+" WITH (FORCE)")
	}
}

func startPG() {
	pgHost, pgErr = ensure(pgName, testcontainers.ContainerRequest{
		Image:        postgresImage,
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_USER": pgUser, "POSTGRES_PASSWORD": pgPass, "POSTGRES_DB": "postgres"},
		Cmd:          []string{"-c", "max_connections=400", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"},
		ShmSize:      256 << 20, // mặc định 64 MB: truy vấn song song của vài gói test cùng lúc làm "could not resize shared memory segment"
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(3 * time.Minute),
	}, "5432/tcp", func(hp string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		c, err := pgx.Connect(ctx, pgDSN(hp, "postgres"))
		if err != nil {
			return err
		}
		defer func() { _ = c.Close(ctx) }()
		return c.Ping(ctx)
	})
	if pgErr == nil {
		purgeOldDatabases(pgHost)
	}
}

// PostgresURL trả URL (sslmode=disable) tới MỘT database mới, trống, trong container Postgres dùng chung. Database KHÔNG bị xoá khi
// test xong (nhiều test dùng chung một database dựng một lần cho cả gói); database cũ hơn 30 phút bị dọn ở lần dùng đầu của mỗi tiến trình.
func PostgresURL(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	pgOnce.Do(startPG)
	if pgErr != nil {
		t.Fatalf("postgres container: %v", pgErr)
	}
	seqMu.Lock()
	seq++
	name := fmt.Sprintf("t_%d_%d_%d", time.Now().Unix(), os.Getpid(), seq)
	seqMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, err := pgx.Connect(ctx, pgDSN(pgHost, "postgres"))
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	return pgDSN(pgHost, name)
}

// PostgresHostPort trả địa chỉ host:port (trên máy chạy test) của container Postgres dùng chung.
func PostgresHostPort(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	pgOnce.Do(startPG)
	if pgErr != nil {
		t.Fatalf("postgres container: %v", pgErr)
	}
	return pgHost
}

// RedisURL trả `redis://host:port/0` của container Redis dùng chung. Test PHẢI dùng id/tiền tố riêng (TestPrefix, uuid ngẫu nhiên);
// không FLUSHALL, không giết container.
func RedisURL(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	redisOne.Do(func() {
		var hp string
		hp, redisErr = ensure(redisName, testcontainers.ContainerRequest{
			Image:        redisImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(2 * time.Minute),
		}, "6379/tcp", func(hp string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			rc := redis.NewClient(&redis.Options{Addr: hp})
			defer func() { _ = rc.Close() }()
			return rc.Ping(ctx).Err()
		})
		redisURL = "redis://" + hp + "/0"
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
	minioOne.Do(func() {
		minioEP, minioErr = ensure(minioName, testcontainers.ContainerRequest{
			Image:        minioImage,
			ExposedPorts: []string{"9000/tcp"},
			Cmd:          []string{"server", "/data"},
			Env:          map[string]string{"MINIO_ROOT_USER": MinIOAccessKey, "MINIO_ROOT_PASSWORD": MinIOSecretKey},
			WaitingFor:   wait.ForHTTP("/minio/health/live").WithPort("9000/tcp").WithStartupTimeout(2 * time.Minute),
		}, "9000/tcp", func(string) error { return nil })
	})
	if minioErr != nil {
		t.Fatalf("minio container: %v", minioErr)
	}
	return minioEP
}

// TestPrefix sinh tiền tố duy nhất cho một test (dùng làm user id / tên stream để các test không đạp nhau).
func TestPrefix(t testing.TB) string {
	t.Helper()
	seqMu.Lock()
	defer seqMu.Unlock()
	seq++
	return fmt.Sprintf("t%d%d%d", os.Getpid(), time.Now().UnixNano()%1e9, seq)
}

// Mailpit trả (host:port SMTP, URL gốc REST API) của container Mailpit dùng chung. Test PHẢI dùng địa chỉ người nhận
// duy nhất (tìm bằng `to:`), không xoá hộp thư chung.
func Mailpit(t testing.TB) (smtpAddr, apiURL string) {
	t.Helper()
	RequireContainers(t)
	mpOnce.Do(func() {
		req := testcontainers.ContainerRequest{
			Image:        mailpitImage,
			ExposedPorts: []string{"1025/tcp", "8025/tcp"},
			WaitingFor:   wait.ForHTTP("/readyz").WithPort("8025/tcp").WithStartupTimeout(2 * time.Minute),
		}
		ready := func(string) error { return nil }
		if mpSMTP, mpErr = ensure(mailpitName, req, "1025/tcp", ready); mpErr != nil {
			return
		}
		var api string
		if api, mpErr = ensure(mailpitName, req, "8025/tcp", ready); mpErr != nil {
			return
		}
		mpAPI = "http://" + api
	})
	if mpErr != nil {
		t.Fatalf("mailpit container: %v", mpErr)
	}
	return mpSMTP, mpAPI
}
