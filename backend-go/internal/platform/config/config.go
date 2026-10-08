// Package config: nguồn duy nhất đọc biến môi trường (SRS 8.1).
// Thiếu biến bắt buộc hoặc giá trị sai → lỗi nêu TÊN biến và lý do, không bao giờ in giá trị.
package config

import (
	"fmt"
	"math"
	"net/mail"
	"net/netip"
	"net/url"
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

	CORSOrigins        []string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	SessionAbsoluteTTL time.Duration
	CookieDomain       string
	AuthResendWindow   time.Duration
	// Giới hạn đăng nhập và hành động công khai (FEAT-account-security SRS 4.2.2 / 8.1).
	AuthLoginIPPerMin      int
	AuthLoginIPFailPer15m  int
	AuthRegisterIPPerHour  int
	AuthForgotIPPerHour    int
	AuthForgotEmailPerHour int
	AuthLinkIPPerMin       int
	AuthRefreshIPPerMin    int
	AuthChangePWFailPer10m int
	LockoutBackoffFrom     int
	LockoutLockAt          int
	LockoutDuration        time.Duration
	BcryptCost             int
	RateLimitIPPerMin      int
	RateLimitUserPerMin    int
	TrustedProxyCIDRs      []netip.Prefix

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
	// Nhập test zip (US-PE-03, SRS 4.1.5): giới hạn nén / giải nén (đếm khi đọc).
	ExamTestZipMaxBytes        int
	ExamTestZipMaxUncompressed int
	// Bài thi (US-PE-04, SRS 4.2): thời lượng tối thiểu, độ trễ tối thiểu khi lên lịch, nhịp bộ lập lịch, tổng thời gian chấm một bài code.
	ExamMinDurationMinutes int
	ExamMinLeadSeconds     int
	ExamTickInterval       time.Duration
	JudgeMaxTotalSeconds   int // JUDGE_MAX_TOTAL_SECONDS; worker đọc riêng cùng biến này ở judge.Settings
	// Lượt làm (US-PE-05, SRS 4.3): độ trễ chấp nhận ghi sau hạn, giới hạn lưu theo lượt, thời gian một tab ghi bị coi là bỏ.
	ExamGraceSeconds   int
	ExamSaveRatePerMin int
	ExamTabStale       time.Duration
	// Bài code trong lượt làm (US-PE-06, SRS 4.4): Chạy thử N lần / cửa sổ / sinh viên, giãn cách giữa hai lần nộp, số lần nộp tối đa mỗi bài.
	ExamRunLimit       int
	ExamRunWindow      time.Duration
	ExamSubmitCooldown time.Duration
	ExamSubmissionCap  int
	// Liêm chính (US-PE-07): sự kiện tối đa mỗi lượt; ngưỡng độ giống tối thiểu (‰, từ SIMILARITY_MIN = 0,60).
	ExamEventsMax         int
	SimilarityMinPermille int

	// Cổng LLM (SRS FEAT-llm-gateway 4.3, 8.1).
	LLMMaxConcurrency int
	LLMBatchShare     float64
	LLMQueueMax       int
	LLMQueueWaitMax   time.Duration
	LLMRequestTimeout time.Duration
	LLMBreakerFails   int
	LLMBreakerOpen    time.Duration
	LLMDefaultRPM     int
	LLMDefaultTPM     int
	LLMEmbedDims      int
	LLMProvider       string // "" | "fake"
	LLMReplayDir      string
	OpenAIKey         string
	AnthropicKey      string
	GeminiKey         string
	FakeLatencyMin    time.Duration
	FakeLatencyMax    time.Duration
	FakeErrorRate     float64
	FakeValidKey      string

	// Tài khoản an toàn (SRS FEAT-account-security 8.1): thư và hạn token một lần.
	AppPublicURL    string // gốc dựng liên kết trong thư (không có dấu / cuối)
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPass        string // bí mật: không log
	SMTPTLS         string // none | starttls | tls
	MailFrom        string
	MailSendTimeout time.Duration
	VerifyTokenTTL  time.Duration
	ResetTokenTTL   time.Duration
	InviteTokenTTL  time.Duration
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
	c.AppEncryptionKey = l.raw("APP_ENCRYPTION_KEY")
	if len(l.missing) > 0 {
		return Config{}, &ErrMissingEnv{Names: l.missing}
	}

	if c.JWTSecretKey != "" && len(c.JWTSecretKey) < 32 {
		l.bad("JWT_SECRET_KEY", "cần ≥ 32 byte")
	}
	// Thiếu hay hỏng đều là cùng một lỗi, cùng một thông điệp (US-P1-01 AC6); không bao giờ in giá trị khoá.
	if c.AppEncryptionKey != "" || gw {
		if _, err := crypto.ParseKey(c.AppEncryptionKey); err != nil {
			l.invalid = append(l.invalid, "APP_ENCRYPTION_KEY")
			l.problems = append(l.problems, err.Error())
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
	c.AccessTokenTTL = l.durRange("ACCESS_TOKEN_TTL", 15*time.Minute, time.Minute, time.Hour)
	c.RefreshTokenTTL = l.durRange("REFRESH_TOKEN_TTL", 336*time.Hour, time.Hour, 0)
	c.SessionAbsoluteTTL = l.durRange("SESSION_ABSOLUTE_TTL", 720*time.Hour, time.Hour, 0)
	if c.SessionAbsoluteTTL < c.RefreshTokenTTL {
		l.bad("SESSION_ABSOLUTE_TTL", "phải ≥ REFRESH_TOKEN_TTL")
	}
	c.CookieDomain = l.str("COOKIE_DOMAIN", "")
	c.AuthResendWindow = time.Duration(l.num("AUTH_RESEND_SECONDS", 60, 1, 3600)) * time.Second
	c.AuthLoginIPPerMin = l.num("AUTH_LOGIN_IP_PER_MIN", 10, 1, 1_000_000)
	c.AuthLoginIPFailPer15m = l.num("AUTH_LOGIN_IP_FAIL_PER_15M", 30, 1, 1_000_000)
	c.AuthRegisterIPPerHour = l.num("AUTH_REGISTER_IP_PER_HOUR", 5, 1, 1_000_000)
	c.AuthForgotIPPerHour = l.num("AUTH_FORGOT_IP_PER_HOUR", 5, 1, 1_000_000)
	c.AuthForgotEmailPerHour = l.num("AUTH_FORGOT_EMAIL_PER_HOUR", 3, 1, 1_000_000)
	c.AuthLinkIPPerMin = l.num("AUTH_TOKEN_IP_PER_MIN", 20, 1, 1_000_000)
	c.AuthRefreshIPPerMin = l.num("AUTH_REFRESH_IP_PER_MIN", 60, 1, 1_000_000)
	c.AuthChangePWFailPer10m = l.num("AUTH_CHANGE_PW_FAIL_PER_10M", 5, 1, 1_000)
	c.LockoutBackoffFrom = l.num("LOCKOUT_BACKOFF_FROM", 5, 1, 1_000)
	c.LockoutLockAt = l.num("LOCKOUT_LOCK_AT", 10, 2, 1_000)
	if c.LockoutLockAt <= c.LockoutBackoffFrom {
		l.bad("LOCKOUT_LOCK_AT", "phải > LOCKOUT_BACKOFF_FROM")
	}
	c.LockoutDuration = l.durRange("LOCKOUT_DURATION", 15*time.Minute, time.Second, 24*time.Hour)
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
	c.ExamTestZipMaxBytes = l.num("EXAM_TESTZIP_MAX_BYTES", 10<<20, 1024, 100<<20)
	c.ExamTestZipMaxUncompressed = l.num("EXAM_TESTZIP_MAX_UNCOMPRESSED", 50<<20, 1024, 500<<20)
	c.ExamMinDurationMinutes = l.num("EXAM_MIN_DURATION_MINUTES", 5, 1, 300)
	c.ExamMinLeadSeconds = l.num("EXAM_MIN_LEAD_SECONDS", 60, 0, 86400)
	c.ExamTickInterval = l.dur("EXAM_TICK_INTERVAL", 5*time.Second)
	c.JudgeMaxTotalSeconds = l.num("JUDGE_MAX_TOTAL_SECONDS", 300, 20, 100000)
	c.ExamGraceSeconds = l.num("EXAM_GRACE_SECONDS", 10, 0, 300)
	c.ExamSaveRatePerMin = l.num("EXAM_SAVE_RATE_PER_MIN", 240, 1, 100000)
	c.ExamTabStale = l.dur("EXAM_TAB_STALE", 20*time.Second)
	c.ExamRunLimit = l.num("EXAM_RUN_LIMIT", 10, 1, 1000)
	c.ExamRunWindow = l.durRange("EXAM_RUN_WINDOW", 10*time.Minute, time.Minute, 24*time.Hour)
	c.ExamSubmitCooldown = l.durRange("EXAM_SUBMIT_COOLDOWN", 15*time.Second, time.Second, time.Hour)
	c.ExamSubmissionCap = l.num("EXAM_SUBMISSION_CAP", 30, 1, 1000)
	c.ExamEventsMax = l.num("EXAM_EVENTS_MAX", 500, 1, 100000)
	c.SimilarityMinPermille = int(math.Round(l.fraction("SIMILARITY_MIN", 0.60) * 1000))

	c.LLMMaxConcurrency = l.num("LLM_MAX_CONCURRENCY", 10, 1, 1000)
	c.LLMBatchShare = l.fraction("LLM_BATCH_SHARE", 0.5)
	c.LLMQueueMax = l.num("LLM_QUEUE_MAX", 200, 1, 100000)
	c.LLMQueueWaitMax = l.dur("LLM_QUEUE_WAIT_MAX", 10*time.Second)
	c.LLMRequestTimeout = l.dur("LLM_REQUEST_TIMEOUT", 30*time.Second)
	c.LLMBreakerFails = l.num("LLM_BREAKER_FAILS", 5, 1, 1000)
	c.LLMBreakerOpen = l.dur("LLM_BREAKER_OPEN", 30*time.Second)
	c.LLMDefaultRPM = l.num("LLM_DEFAULT_RPM", 60, 1, 10_000_000)
	c.LLMDefaultTPM = l.num("LLM_DEFAULT_TPM", 100000, 1, 1_000_000_000)
	c.LLMEmbedDims = l.num("LLM_EMBED_DIMS", 1536, 1, 100000)
	if c.LLMEmbedDims != 1536 {
		l.bad("LLM_EMBED_DIMS", "khoá cố định 1536 (pgvector vector(1536)); đổi số chiều cần migration và dựng lại chỉ mục")
	}
	c.LLMProvider = l.enum("LLM_PROVIDER", "", "", "fake")
	c.LLMReplayDir = l.str("LLM_REPLAY_DIR", "")
	c.OpenAIKey = l.raw("OPENAI_API_KEY")
	c.AnthropicKey = l.raw("ANTHROPIC_API_KEY")
	c.GeminiKey = l.raw("GEMINI_API_KEY")
	c.FakeLatencyMin, c.FakeLatencyMax = l.latencyRange("FAKE_LLM_LATENCY", "0-0")
	c.FakeErrorRate = l.rate("FAKE_LLM_ERROR_RATE", 0)
	c.FakeValidKey = l.raw("FAKE_LLM_VALID_KEY")

	// Thư chỉ do worker gửi: production ở worker bắt buộc APP_PUBLIC_URL https và SMTP_TLS ≠ none; gateway đọc nhẹ tay (không dùng).
	mailStrict := role == Worker && c.AppEnv == "production"
	c.AppPublicURL = l.publicURL("APP_PUBLIC_URL", mailStrict)
	c.SMTPHost = l.str("SMTP_HOST", "mailpit")
	c.SMTPPort = l.num("SMTP_PORT", 1025, 1, 65535)
	c.SMTPUser = l.raw("SMTP_USER")
	c.SMTPPass = l.raw("SMTP_PASS")
	c.SMTPTLS = l.enum("SMTP_TLS", "none", "none", "starttls", "tls")
	if mailStrict && c.SMTPTLS == "none" {
		l.bad("SMTP_TLS", "production không được dùng none")
	}
	c.MailFrom = l.str("MAIL_FROM", "EduPilot <no-reply@edupilot.local>")
	if _, err := mail.ParseAddress(c.MailFrom); err != nil {
		l.bad("MAIL_FROM", "không phải địa chỉ thư hợp lệ")
	}
	c.MailSendTimeout = l.dur("MAIL_SEND_TIMEOUT", 10*time.Second)
	c.VerifyTokenTTL = l.dur("VERIFY_TOKEN_TTL", 24*time.Hour)
	c.ResetTokenTTL = l.dur("RESET_TOKEN_TTL", 30*time.Minute)
	c.InviteTokenTTL = l.dur("INVITE_TOKEN_TTL", 72*time.Hour)

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
		"access_token_ttl", c.AccessTokenTTL.String(),
		"refresh_token_ttl", c.RefreshTokenTTL.String(),
		"session_absolute_ttl", c.SessionAbsoluteTTL.String(),
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
		"llm_max_concurrency", c.LLMMaxConcurrency,
		"llm_batch_share", c.LLMBatchShare,
		"llm_queue_max", c.LLMQueueMax,
		"llm_queue_wait_max", c.LLMQueueWaitMax.String(),
		"llm_request_timeout", c.LLMRequestTimeout.String(),
		"llm_breaker_fails", c.LLMBreakerFails,
		"llm_breaker_open", c.LLMBreakerOpen.String(),
		"llm_default_rpm", c.LLMDefaultRPM,
		"llm_default_tpm", c.LLMDefaultTPM,
		"llm_embed_dims", c.LLMEmbedDims,
		"llm_provider", c.LLMProvider,
		"app_public_url", c.AppPublicURL,
		"smtp_host", c.SMTPHost,
		"smtp_port", c.SMTPPort,
		"smtp_tls", c.SMTPTLS,
		"mail_send_timeout", c.MailSendTimeout.String(),
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

// publicURL đọc gốc URL công khai: tuyệt đối, http(s), không đường dẫn; strict (worker production) bắt buộc có và phải https.
// Dev/test mặc định https://localhost (Caddy). Bỏ dấu / cuối để nối `/verify-email?...` không bị đôi.
func (l *loader) publicURL(name string, strict bool) string {
	v := l.raw(name)
	if v == "" {
		if strict {
			l.bad(name, "bắt buộc ở worker production")
			return ""
		}
		return "https://localhost"
	}
	u, err := url.Parse(v)
	switch {
	case err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http"):
		l.bad(name, "cần URL tuyệt đối http(s)")
	case strict && u.Scheme != "https":
		l.bad(name, "production cần https")
	case u.Path != "" && u.Path != "/", u.RawQuery != "", u.Fragment != "":
		l.bad(name, "chỉ gốc (scheme://host[:port]), không đường dẫn")
	}
	return strings.TrimRight(v, "/")
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

// durRange như dur nhưng giới hạn [min, max]; max = 0 là không chặn trên.
func (l *loader) durRange(name string, def, lo, hi time.Duration) time.Duration {
	d := l.dur(name, def)
	if d < lo || (hi > 0 && d > hi) {
		l.bad(name, "ngoài khoảng cho phép")
		return def
	}
	return d
}

// fraction đọc số thực trong (0, 1].
func (l *loader) fraction(name string, def float64) float64 {
	v := l.raw(name)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 || f > 1 {
		l.bad(name, "cần số thực trong (0, 1]")
		return def
	}
	return f
}

// rate đọc số thực trong [0, 1].
func (l *loader) rate(name string, def float64) float64 {
	v := l.raw(name)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f > 1 {
		l.bad(name, "cần số thực trong [0, 1]")
		return def
	}
	return f
}

// latencyRange đọc "min-max" (ms), ví dụ 0-0 hoặc 5000-15000.
func (l *loader) latencyRange(name, def string) (time.Duration, time.Duration) {
	v := l.str(name, def)
	lo, hi, ok := strings.Cut(v, "-")
	a, err1 := strconv.Atoi(lo)
	b, err2 := strconv.Atoi(hi)
	if !ok || err1 != nil || err2 != nil || a < 0 || b < a || b > 600_000 {
		l.bad(name, "cần dạng min-max theo ms, ví dụ 0-0 hoặc 5000-15000")
		return 0, 0
	}
	return time.Duration(a) * time.Millisecond, time.Duration(b) * time.Millisecond
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
