// Package config: nguồn duy nhất đọc biến môi trường (SRS 8.1).
// Thiếu biến bắt buộc hoặc giá trị sai → lỗi nêu TÊN biến và lý do, không bao giờ in giá trị.
package config

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/edupilot/backend-go/internal/platform/crypto"
)

// Role chọn tập biến bắt buộc (SRS 8.1 cột "Dùng bởi").
type Role string

// Hai vai của PG: gateway cần đủ 7 biến, worker chỉ cần DATABASE_URL + REDIS_URL.
const (
	Gateway Role = "gateway"
	Worker  Role = "worker"
)

// Config là cấu hình hiệu lực của một tiến trình.
type Config struct {
	Role Role

	// Bí mật / URL có mật khẩu: KHÔNG được log (kể cả dạng che).
	DatabaseURL   string
	PgBouncerURL  string
	RedisURL      string
	JWTSecretKey  string
	BlobAccessKey string
	BlobSecretKey string
	// AppEncryptionKey: 32 byte base64 cho AES-GCM (khoá API của nhà cung cấp LLM). Bắt buộc ở gateway; worker kiểm nếu có.
	AppEncryptionKey string

	AppEnv       string
	HTTPAddr     string
	InstanceID   string
	LogLevel     string
	OTLPEndpoint string

	DBMaxConns    int32
	DBSlowQueryMS int

	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	StartupTimeout  time.Duration
	MaxBodyBytes    int64

	WorkerHealthAddr string

	CORSOrigins         []string
	JWTExpiration       time.Duration
	BcryptCost          int
	RateLimitIPPerMin   int
	RateLimitUserPerMin int
	TrustedProxyCIDRs   []netip.Prefix

	SSEHeartbeat    time.Duration
	SSEMaxDuration  time.Duration
	SSEMaxPerUser   int
	SSEConnTTL      time.Duration
	SSEBufferMaxLen int64
	SSEBufferTTL    time.Duration

	OutboxPollInterval time.Duration
	OutboxBatch        int
	OutboxRetryBackoff []time.Duration
	OutboxClaimIdle    time.Duration
	OutboxStaleAfter   time.Duration

	BlobEndpoint       string
	BlobBucket         string
	BlobUseSSL         bool
	BlobPublicEndpoint string
	BlobRegion         string
}

// ErrMissingEnv liệt kê MỌI biến bắt buộc bị thiếu (chỉ tên).
type ErrMissingEnv struct{ Names []string }

func (e *ErrMissingEnv) Error() string {
	return "thiếu biến môi trường bắt buộc: " + strings.Join(e.Names, ", ")
}

// ErrInvalidEnv liệt kê biến có giá trị sai: mỗi phần tử là "TÊN: lý do" (không chứa giá trị).
type ErrInvalidEnv struct {
	Names    []string
	Problems []string
}

func (e *ErrInvalidEnv) Error() string {
	return "giá trị biến môi trường không hợp lệ: " + strings.Join(e.Problems, "; ")
}

// Load đọc toàn bộ cấu hình qua getenv theo vai. Rỗng sau TrimSpace = thiếu.
// Trả *ErrMissingEnv (đủ mọi tên thiếu) hoặc *ErrInvalidEnv (đủ mọi biến sai).
func Load(getenv func(string) string, role Role) (Config, error) {
	l := &loader{getenv: getenv}
	c := Config{Role: role}

	gw := role == Gateway
	c.DatabaseURL = l.need("DATABASE_URL", true)
	c.RedisURL = l.need("REDIS_URL", true)
	c.JWTSecretKey = l.need("JWT_SECRET_KEY", gw)
	c.BlobEndpoint = l.need("BLOB_ENDPOINT", gw)
	c.BlobBucket = l.need("BLOB_BUCKET", gw)
	c.BlobAccessKey = l.need("BLOB_ACCESS_KEY", gw)
	c.BlobSecretKey = l.need("BLOB_SECRET_KEY", gw)
	c.AppEncryptionKey = l.need("APP_ENCRYPTION_KEY", gw)
	if len(l.missing) > 0 {
		return Config{}, &ErrMissingEnv{Names: l.missing}
	}

	if c.JWTSecretKey != "" && len(c.JWTSecretKey) < 32 {
		l.bad("JWT_SECRET_KEY", "cần ≥ 32 byte")
	}
	if c.AppEncryptionKey != "" {
		if _, err := crypto.ParseKey(c.AppEncryptionKey); err != nil {
			l.invalid = append(l.invalid, "APP_ENCRYPTION_KEY")
			l.problems = append(l.problems, err.Error()) // "APP_ENCRYPTION_KEY không hợp lệ: cần 32 byte (base64)" — không in giá trị
		}
	}

	c.PgBouncerURL = l.str("PGBOUNCER_URL", "")
	c.AppEnv = l.enum("APP_ENV", "dev", "dev", "test", "production")
	c.HTTPAddr = l.str("HTTP_ADDR", ":8080")
	c.InstanceID = l.str("INSTANCE_ID", hostname())
	c.LogLevel = l.enum("LOG_LEVEL", "info", "debug", "info", "warn", "error")
	c.OTLPEndpoint = l.str("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	c.DBMaxConns = int32(l.num("DB_MAX_CONNS", 10, 1, 100))
	c.DBSlowQueryMS = l.num("DB_SLOW_QUERY_MS", 200, 1, 3_600_000)

	c.RequestTimeout = l.dur("REQUEST_TIMEOUT", 30*time.Second)
	c.ShutdownTimeout = l.dur("SHUTDOWN_TIMEOUT", 25*time.Second)
	c.StartupTimeout = l.dur("STARTUP_TIMEOUT", 30*time.Second)
	c.MaxBodyBytes = int64(l.num("MAX_BODY_BYTES", 1048576, 1, 1<<30))

	c.WorkerHealthAddr = l.str("WORKER_HEALTH_ADDR", ":8081")

	c.CORSOrigins = l.origins("CORS_ORIGINS", "http://localhost:3000,https://localhost")
	c.JWTExpiration = l.dur("JWT_EXPIRATION", 15*time.Minute)
	c.BcryptCost = l.num("BCRYPT_COST", 12, 4, 14)
	c.RateLimitIPPerMin = l.num("RATE_LIMIT_IP_PER_MIN", 300, 1, 1_000_000)
	c.RateLimitUserPerMin = l.num("RATE_LIMIT_USER_PER_MIN", 600, 1, 1_000_000)
	c.TrustedProxyCIDRs = l.cidrs("TRUSTED_PROXY_CIDRS", "127.0.0.0/8,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")

	c.SSEHeartbeat = l.dur("SSE_HEARTBEAT", 25*time.Second)
	c.SSEMaxDuration = l.dur("SSE_MAX_DURATION", 120*time.Second)
	c.SSEMaxPerUser = l.num("SSE_MAX_PER_USER", 2, 1, 100)
	c.SSEConnTTL = l.dur("SSE_CONN_TTL", 150*time.Second)
	c.SSEBufferMaxLen = int64(l.num("SSE_BUFFER_MAXLEN", 1000, 1, 1_000_000))
	c.SSEBufferTTL = l.dur("SSE_BUFFER_TTL", time.Hour)

	c.OutboxPollInterval = l.dur("OUTBOX_POLL_INTERVAL", 500*time.Millisecond)
	c.OutboxBatch = l.num("OUTBOX_BATCH", 100, 1, 10000)
	c.OutboxRetryBackoff = l.backoff("OUTBOX_RETRY_BACKOFF", "1s,5s,30s")
	c.OutboxClaimIdle = l.dur("OUTBOX_CLAIM_IDLE", 60*time.Second)
	c.OutboxStaleAfter = l.dur("OUTBOX_STALE_AFTER", 120*time.Second)
	if c.OutboxStaleAfter <= c.OutboxClaimIdle {
		l.bad("OUTBOX_STALE_AFTER", "phải lớn hơn OUTBOX_CLAIM_IDLE")
	}

	c.BlobUseSSL = l.boolean("BLOB_USE_SSL", false)
	c.BlobPublicEndpoint = l.str("BLOB_PUBLIC_ENDPOINT", c.BlobEndpoint)
	c.BlobRegion = l.str("BLOB_REGION", "us-east-1")

	if len(l.problems) > 0 {
		return Config{}, &ErrInvalidEnv{Names: l.invalid, Problems: l.problems}
	}
	return c, nil
}

// DBVia cho biết runtime nối thẳng Postgres hay qua PgBouncer (trường log `db_via`).
func (c Config) DBVia() string {
	if c.PgBouncerURL != "" {
		return "pgbouncer"
	}
	return "direct"
}

// RuntimeDatabaseURL là URL pool dùng lúc chạy (PgBouncer nếu có); migrate luôn dùng DatabaseURL.
func (c Config) RuntimeDatabaseURL() string {
	if c.PgBouncerURL != "" {
		return c.PgBouncerURL
	}
	return c.DatabaseURL
}

// Service là tên dịch vụ cho log, span và `application_name` của pool.
func (c Config) Service() string { return string(c.Role) }

// LogAttrs trả cặp khoá/giá trị cho dòng `config loaded`: chỉ biến KHÔNG bí mật,
// khoá = tên biến viết thường, thời lượng là chuỗi, kèm `db_via` và `secrets:"[redacted]"` (SRS 8.1, #Q-QC-01-2).
func (c Config) LogAttrs() []any {
	return []any{
		"app_env", c.AppEnv,
		"http_addr", c.HTTPAddr,
		"instance_id", c.InstanceID,
		"log_level", c.LogLevel,
		"otel_exporter_otlp_endpoint", c.OTLPEndpoint,
		"db_max_conns", c.DBMaxConns,
		"db_slow_query_ms", c.DBSlowQueryMS,
		"request_timeout", c.RequestTimeout.String(),
		"shutdown_timeout", c.ShutdownTimeout.String(),
		"startup_timeout", c.StartupTimeout.String(),
		"max_body_bytes", c.MaxBodyBytes,
		"worker_health_addr", c.WorkerHealthAddr,
		"cors_origins", strings.Join(c.CORSOrigins, ","),
		"jwt_expiration", c.JWTExpiration.String(),
		"bcrypt_cost", c.BcryptCost,
		"rate_limit_ip_per_min", c.RateLimitIPPerMin,
		"rate_limit_user_per_min", c.RateLimitUserPerMin,
		"trusted_proxy_cidrs", joinPrefixes(c.TrustedProxyCIDRs),
		"sse_heartbeat", c.SSEHeartbeat.String(),
		"sse_max_duration", c.SSEMaxDuration.String(),
		"sse_max_per_user", c.SSEMaxPerUser,
		"sse_conn_ttl", c.SSEConnTTL.String(),
		"sse_buffer_maxlen", c.SSEBufferMaxLen,
		"sse_buffer_ttl", c.SSEBufferTTL.String(),
		"outbox_poll_interval", c.OutboxPollInterval.String(),
		"outbox_batch", c.OutboxBatch,
		"outbox_retry_backoff", joinDurations(c.OutboxRetryBackoff),
		"outbox_claim_idle", c.OutboxClaimIdle.String(),
		"outbox_stale_after", c.OutboxStaleAfter.String(),
		"blob_endpoint", c.BlobEndpoint,
		"blob_bucket", c.BlobBucket,
		"blob_use_ssl", c.BlobUseSSL,
		"blob_public_endpoint", c.BlobPublicEndpoint,
		"blob_region", c.BlobRegion,
		"db_via", c.DBVia(),
		"secrets", "[redacted]",
	}
}

func joinPrefixes(ps []netip.Prefix) string {
	s := make([]string, len(ps))
	for i, p := range ps {
		s[i] = p.String()
	}
	return strings.Join(s, ",")
}

func joinDurations(ds []time.Duration) string {
	s := make([]string, len(ds))
	for i, d := range ds {
		s[i] = d.String()
	}
	return strings.Join(s, ",")
}

// loader gom mọi biến thiếu / sai để báo một lần.
type loader struct {
	getenv   func(string) string
	missing  []string
	invalid  []string
	problems []string
}

func (l *loader) bad(name, reason string) {
	l.invalid = append(l.invalid, name)
	l.problems = append(l.problems, name+": "+reason)
}

func (l *loader) raw(name string) string { return strings.TrimSpace(l.getenv(name)) }

func (l *loader) need(name string, required bool) string {
	v := l.raw(name)
	if v == "" && required {
		l.missing = append(l.missing, name)
	}
	return v
}

func (l *loader) str(name, def string) string {
	if v := l.raw(name); v != "" {
		return v
	}
	return def
}

func (l *loader) enum(name, def string, allowed ...string) string {
	v := l.str(name, def)
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	l.bad(name, "chỉ nhận "+strings.Join(allowed, " | "))
	return def
}

func (l *loader) num(name string, def, lo, hi int) int {
	v := l.raw(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < lo || n > hi {
		l.bad(name, fmt.Sprintf("cần số nguyên trong %d–%d", lo, hi))
		return def
	}
	return n
}

func (l *loader) dur(name string, def time.Duration) time.Duration {
	v := l.raw(name)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		l.bad(name, "cần thời lượng dương kiểu Go (ví dụ 30s, 15m, 1h)")
		return def
	}
	return d
}

func (l *loader) boolean(name string, def bool) bool {
	v := l.raw(name)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		l.bad(name, "cần true hoặc false")
		return def
	}
	return b
}

func (l *loader) origins(name, def string) []string {
	v := l.str(name, def)
	parts := splitList(v)
	if len(parts) == 0 {
		l.bad(name, "cần ít nhất một origin")
		return nil
	}
	for _, p := range parts {
		if p == "*" || strings.Contains(p, "*") {
			l.bad(name, "cấm ký tự đại diện, phải liệt kê từng origin")
			return nil
		}
	}
	return parts
}

func (l *loader) cidrs(name, def string) []netip.Prefix {
	parts := splitList(l.str(name, def))
	out := make([]netip.Prefix, 0, len(parts))
	for _, p := range parts {
		pre, err := netip.ParsePrefix(p)
		if err != nil {
			l.bad(name, "cần danh sách CIDR ngăn cách bằng dấu phẩy")
			return nil
		}
		out = append(out, pre)
	}
	return out
}

func (l *loader) backoff(name, def string) []time.Duration {
	parts := splitList(l.str(name, def))
	if len(parts) != 3 {
		l.bad(name, "cần đúng 3 thời lượng ngăn cách bằng dấu phẩy")
		return nil
	}
	out := make([]time.Duration, 0, 3)
	for _, p := range parts {
		d, err := time.ParseDuration(p)
		if err != nil || d <= 0 {
			l.bad(name, "cần 3 thời lượng dương kiểu Go (ví dụ 1s,5s,30s)")
			return nil
		}
		out = append(out, d)
	}
	return out
}

// hostname là mặc định của INSTANCE_ID (SRS 8.1).
func hostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown"
}

func splitList(v string) []string {
	out := []string{}
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
