package config_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
)

const (
	secretKey = "0123456789abcdef0123456789abcdef"
	dbURL     = "postgres://u:supersecretpw@127.0.0.1:5432/db"
	redisURL  = "redis://:redispw@127.0.0.1:6379/0"
)

func full() map[string]string {
	return map[string]string{
		"DATABASE_URL":       dbURL,
		"REDIS_URL":          redisURL,
		"JWT_SECRET_KEY":     secretKey,
		"BLOB_ENDPOINT":      "minio:9000",
		"BLOB_BUCKET":        "edupilot",
		"BLOB_ACCESS_KEY":    "ak-dev",
		"BLOB_SECRET_KEY":    "sk-dev",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=",
	}
}

func getenv(env map[string]string) func(string) string {
	return func(k string) string { return env[k] }
}

func TestLoad_MissingEnv(t *testing.T) {
	t.Parallel()
	required := []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET_KEY", "BLOB_ENDPOINT", "BLOB_BUCKET", "BLOB_ACCESS_KEY", "BLOB_SECRET_KEY"}

	for _, name := range required {
		t.Run("thiếu "+name, func(t *testing.T) {
			t.Parallel()
			env := full()
			delete(env, name)
			_, err := config.Load(getenv(env), config.Gateway)
			var me *config.ErrMissingEnv
			if !errors.As(err, &me) {
				t.Fatalf("err = %v, muốn *ErrMissingEnv", err)
			}
			if !slices.Equal(me.Names, []string{name}) {
				t.Fatalf("missing = %v, muốn [%s]", me.Names, name)
			}
			if strings.Contains(err.Error(), "supersecretpw") || strings.Contains(err.Error(), secretKey) {
				t.Fatalf("thông điệp lỗi lộ giá trị: %s", err)
			}
		})
	}

	t.Run("thiếu tất cả thì liệt kê đủ một lần", func(t *testing.T) {
		t.Parallel()
		_, err := config.Load(getenv(map[string]string{}), config.Gateway)
		var me *config.ErrMissingEnv
		if !errors.As(err, &me) {
			t.Fatalf("err = %v", err)
		}
		got := slices.Clone(me.Names)
		slices.Sort(got)
		want := slices.Clone(required)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("missing = %v, muốn %v", got, want)
		}
	})

	t.Run("chỉ khoảng trắng coi như thiếu", func(t *testing.T) {
		t.Parallel()
		env := full()
		env["BLOB_BUCKET"] = "  \t "
		_, err := config.Load(getenv(env), config.Gateway)
		var me *config.ErrMissingEnv
		if !errors.As(err, &me) || !slices.Equal(me.Names, []string{"BLOB_BUCKET"}) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestLoad_Worker(t *testing.T) {
	t.Parallel()

	t.Run("worker chỉ cần DATABASE_URL và REDIS_URL", func(t *testing.T) {
		t.Parallel()
		cfg, err := config.Load(getenv(map[string]string{"DATABASE_URL": dbURL, "REDIS_URL": redisURL}), config.Worker)
		if err != nil {
			t.Fatalf("err = %v, muốn nil", err)
		}
		if cfg.Service() != "worker" || cfg.WorkerHealthAddr != ":8081" {
			t.Fatalf("cfg = %+v", cfg)
		}
	})

	for _, name := range []string{"DATABASE_URL", "REDIS_URL"} {
		t.Run("worker thiếu "+name, func(t *testing.T) {
			t.Parallel()
			env := map[string]string{"DATABASE_URL": dbURL, "REDIS_URL": redisURL}
			delete(env, name)
			_, err := config.Load(getenv(env), config.Worker)
			var me *config.ErrMissingEnv
			if !errors.As(err, &me) || !slices.Equal(me.Names, []string{name}) {
				t.Fatalf("err = %v, muốn missing [%s]", err, name)
			}
		})
	}
}

func TestLoad_Invalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, key, value string
		noLeak           string // giá trị phải KHÔNG xuất hiện trong thông điệp (bỏ trống với số: "1–100" có chứa chữ số)
	}{
		{"khoá JWT 31 byte", "JWT_SECRET_KEY", "0123456789abcdef0123456789abcde", "0123456789abcdef0123456789abcde"},
		{"DB_MAX_CONNS=0", "DB_MAX_CONNS", "0", ""},
		{"DB_MAX_CONNS=101", "DB_MAX_CONNS", "101", ""},
		{"JWT_EXPIRATION không phải thời lượng", "JWT_EXPIRATION", "abc", "abc"},
		{"CORS_ORIGINS có dấu sao", "CORS_ORIGINS", "*", ""},
		{"APP_ENV lạ", "APP_ENV", "staging", "staging"},
		{"BCRYPT_COST dưới 4", "BCRYPT_COST", "3", ""},
		{"BCRYPT_COST trên 14", "BCRYPT_COST", "15", ""},
		{"BCRYPT_COST không phải số", "BCRYPT_COST", "abc", "abc"},
		{"LOG_LEVEL lạ", "LOG_LEVEL", "verbose", "verbose"},
		{"TRUSTED_PROXY_CIDRS sai", "TRUSTED_PROXY_CIDRS", "10.0.0.1", ""},
		{"OUTBOX_RETRY_BACKOFF chỉ 2 giá trị", "OUTBOX_RETRY_BACKOFF", "1s,5s", ""},
		{"APP_ENCRYPTION_KEY không phải base64", "APP_ENCRYPTION_KEY", "abc$%^-not-base64", "abc$%^-not-base64"},
		{"APP_ENCRYPTION_KEY 16 byte", "APP_ENCRYPTION_KEY", "AAAAAAAAAAAAAAAAAAAAAA==", "AAAAAAAAAAAAAAAAAAAAAA=="},
		{"APP_ENCRYPTION_KEY rỗng ở gateway", "APP_ENCRYPTION_KEY", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := full()
			env[tc.key] = tc.value
			_, err := config.Load(getenv(env), config.Gateway)
			var ie *config.ErrInvalidEnv
			if !errors.As(err, &ie) {
				t.Fatalf("err = %v, muốn *ErrInvalidEnv", err)
			}
			if !slices.Contains(ie.Names, tc.key) {
				t.Fatalf("Names = %v, muốn chứa %s", ie.Names, tc.key)
			}
			if !strings.Contains(err.Error(), tc.key) {
				t.Fatalf("thông điệp %q không nêu tên biến %s", err, tc.key)
			}
			if tc.noLeak != "" && strings.Contains(err.Error(), tc.noLeak) {
				t.Fatalf("thông điệp %q lộ giá trị %q", err, tc.noLeak)
			}
		})
	}

	t.Run("giá trị biên hợp lệ", func(t *testing.T) {
		t.Parallel()
		env := full()
		env["DB_MAX_CONNS"] = "1"
		env["BCRYPT_COST"] = "4"
		env["APP_ENV"] = "production"
		if _, err := config.Load(getenv(env), config.Gateway); err != nil {
			t.Fatalf("err = %v, muốn nil", err)
		}
		env["DB_MAX_CONNS"] = "100"
		env["BCRYPT_COST"] = "14"
		env["JWT_SECRET_KEY"] = secretKey // đúng 32 byte
		if _, err := config.Load(getenv(env), config.Gateway); err != nil {
			t.Fatalf("err = %v, muốn nil", err)
		}
	})

	t.Run("liệt kê đủ mọi biến sai", func(t *testing.T) {
		t.Parallel()
		env := full()
		env["DB_MAX_CONNS"] = "0"
		env["APP_ENV"] = "staging"
		_, err := config.Load(getenv(env), config.Gateway)
		var ie *config.ErrInvalidEnv
		if !errors.As(err, &ie) || len(ie.Names) != 2 {
			t.Fatalf("Names = %v, muốn đủ 2 biến", err)
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(getenv(full()), config.Gateway)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"DB_MAX_CONNS", cfg.DBMaxConns, int32(10)},
		{"DB_SLOW_QUERY_MS", cfg.DBSlowQueryMS, 200},
		{"REQUEST_TIMEOUT", cfg.RequestTimeout, 30 * time.Second},
		{"SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout, 25 * time.Second},
		{"STARTUP_TIMEOUT", cfg.StartupTimeout, 30 * time.Second},
		{"MAX_BODY_BYTES", cfg.MaxBodyBytes, int64(1048576)},
		{"HTTP_ADDR", cfg.HTTPAddr, ":8080"},
		{"WORKER_HEALTH_ADDR", cfg.WorkerHealthAddr, ":8081"},
		{"APP_ENV", cfg.AppEnv, "dev"},
		{"LOG_LEVEL", cfg.LogLevel, "info"},
		{"JWT_EXPIRATION", cfg.JWTExpiration, 15 * time.Minute},
		{"BCRYPT_COST", cfg.BcryptCost, 12},
		{"RATE_LIMIT_IP_PER_MIN", cfg.RateLimitIPPerMin, 300},
		{"RATE_LIMIT_USER_PER_MIN", cfg.RateLimitUserPerMin, 600},
		{"SSE_HEARTBEAT", cfg.SSEHeartbeat, 25 * time.Second},
		{"SSE_MAX_DURATION", cfg.SSEMaxDuration, 120 * time.Second},
		{"SSE_MAX_PER_USER", cfg.SSEMaxPerUser, 2},
		{"SSE_CONN_TTL", cfg.SSEConnTTL, 150 * time.Second},
		{"SSE_BUFFER_MAXLEN", cfg.SSEBufferMaxLen, int64(1000)},
		{"SSE_BUFFER_TTL", cfg.SSEBufferTTL, time.Hour},
		{"OUTBOX_POLL_INTERVAL", cfg.OutboxPollInterval, 500 * time.Millisecond},
		{"OUTBOX_BATCH", cfg.OutboxBatch, 100},
		{"OUTBOX_CLAIM_IDLE", cfg.OutboxClaimIdle, 60 * time.Second},
		{"OUTBOX_STALE_AFTER", cfg.OutboxStaleAfter, 120 * time.Second},
		{"BLOB_REGION", cfg.BlobRegion, "us-east-1"},
		{"BLOB_USE_SSL", cfg.BlobUseSSL, false},
		{"BLOB_PUBLIC_ENDPOINT", cfg.BlobPublicEndpoint, "minio:9000"},
		{"db_via", cfg.DBVia(), "direct"},
		{"runtime URL", cfg.RuntimeDatabaseURL(), dbURL},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, muốn %v", c.name, c.got, c.want)
		}
	}
	if want := []string{"http://localhost:3000", "https://localhost"}; !slices.Equal(cfg.CORSOrigins, want) {
		t.Errorf("CORS_ORIGINS = %v, muốn %v", cfg.CORSOrigins, want)
	}
	if len(cfg.TrustedProxyCIDRs) != 4 {
		t.Errorf("TRUSTED_PROXY_CIDRS = %v, muốn 4 dải", cfg.TrustedProxyCIDRs)
	}
	if !slices.Equal(cfg.OutboxRetryBackoff, []time.Duration{time.Second, 5 * time.Second, 30 * time.Second}) {
		t.Errorf("OUTBOX_RETRY_BACKOFF = %v", cfg.OutboxRetryBackoff)
	}

	env := full()
	env["PGBOUNCER_URL"] = "postgres://u:p@pgbouncer:6432/db"
	viaCfg, err := config.Load(getenv(env), config.Gateway)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if viaCfg.DBVia() != "pgbouncer" || viaCfg.RuntimeDatabaseURL() != env["PGBOUNCER_URL"] {
		t.Errorf("db_via = %s, runtime = %s", viaCfg.DBVia(), viaCfg.RuntimeDatabaseURL())
	}
}

func TestConfig_NoSecretInLog(t *testing.T) {
	t.Parallel()
	env := full()
	env["PGBOUNCER_URL"] = "postgres://u:bouncerpw@pgbouncer:6432/db"
	cfg, err := config.Load(getenv(env), config.Gateway)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("config loaded", cfg.LogAttrs()...)
	line := buf.String()

	for _, secret := range []string{secretKey, "supersecretpw", "redispw", "bouncerpw", "ak-dev", "sk-dev", dbURL, redisURL} {
		if strings.Contains(line, secret) {
			t.Fatalf("dòng log chứa bí mật %q: %s", secret, line)
		}
	}
	for _, key := range []string{"database_url", "redis_url", "pgbouncer_url", "jwt_secret_key", "blob_access_key", "blob_secret_key"} {
		if strings.Contains(line, `"`+key+`"`) {
			t.Fatalf("dòng log có khoá bí mật %q: %s", key, line)
		}
	}
	for _, want := range []string{`"secrets":"[redacted]"`, `"db_via":"pgbouncer"`, `"db_max_conns":10`, `"request_timeout":"30s"`, `"sse_heartbeat":"25s"`, `"app_env":"dev"`, `"cors_origins":"http://localhost:3000,https://localhost"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("dòng log thiếu %s: %s", want, line)
		}
	}
}

// TestLLMEnv — FEAT-llm-gateway US-P1-03 AC15 và US-P1-02 AC11: mặc định đúng SRS 8.1; giá trị sai → từ chối, nêu tên biến, không in giá trị.
func TestLLMEnv(t *testing.T) {
	t.Parallel()
	c, err := config.Load(getenv(full()), config.Gateway)
	if err != nil {
		t.Fatal(err)
	}
	if c.LLMMaxConcurrency != 10 || c.LLMBatchShare != 0.5 || c.LLMQueueMax != 200 || c.LLMQueueWaitMax != 10*time.Second ||
		c.LLMRequestTimeout != 30*time.Second || c.LLMBreakerFails != 5 || c.LLMBreakerOpen != 30*time.Second ||
		c.LLMDefaultRPM != 60 || c.LLMDefaultTPM != 100000 || c.LLMEmbedDims != 1536 || c.LLMProvider != "" ||
		c.FakeLatencyMin != 0 || c.FakeLatencyMax != 0 || c.FakeErrorRate != 0 {
		t.Fatalf("mặc định sai: %+v", c)
	}

	bad := []struct{ key, value string }{
		{"LLM_MAX_CONCURRENCY", "0"}, {"LLM_MAX_CONCURRENCY", "-3"}, {"LLM_MAX_CONCURRENCY", "abc"},
		{"LLM_BATCH_SHARE", "0"}, {"LLM_BATCH_SHARE", "1.5"}, {"LLM_BATCH_SHARE", "-0.1"}, {"LLM_BATCH_SHARE", "nửa"},
		{"LLM_QUEUE_MAX", "0"}, {"LLM_QUEUE_WAIT_MAX", "-1s"}, {"LLM_QUEUE_WAIT_MAX", "mười"}, {"LLM_REQUEST_TIMEOUT", "0s"},
		{"LLM_BREAKER_FAILS", "0"}, {"LLM_BREAKER_OPEN", "0"}, {"LLM_DEFAULT_RPM", "-1"}, {"LLM_DEFAULT_TPM", "0"},
		{"LLM_EMBED_DIMS", "768"}, {"LLM_PROVIDER", "openai"},
		{"FAKE_LLM_LATENCY", "5000"}, {"FAKE_LLM_LATENCY", "10-5"}, {"FAKE_LLM_ERROR_RATE", "1.2"},
	}
	for _, tc := range bad {
		env := full()
		env[tc.key] = tc.value
		_, err := config.Load(getenv(env), config.Gateway)
		var ie *config.ErrInvalidEnv
		if !errors.As(err, &ie) || !slices.Contains(ie.Names, tc.key) {
			t.Errorf("%s=%s: err = %v, muốn ErrInvalidEnv nêu %s", tc.key, tc.value, err, tc.key)
			continue
		}
		if !strings.Contains(err.Error(), tc.key) {
			t.Errorf("%s: thông điệp không nêu tên biến", tc.key)
		}
	}

	env := full()
	env["LLM_BATCH_SHARE"], env["LLM_PROVIDER"], env["FAKE_LLM_LATENCY"], env["FAKE_LLM_ERROR_RATE"] = "1", "fake", "5000-15000", "0.25"
	env["OPENAI_API_KEY"] = "sk-secret-do-not-log"
	c, err = config.Load(getenv(env), config.Gateway)
	if err != nil || c.LLMBatchShare != 1 || c.LLMProvider != "fake" || c.FakeLatencyMin != 5*time.Second || c.FakeLatencyMax != 15*time.Second || c.FakeErrorRate != 0.25 {
		t.Fatalf("giá trị hợp lệ: %+v err=%v", c, err)
	}
	if strings.Contains(fmt.Sprint(c.LogAttrs()...), "sk-secret") {
		t.Error("LogAttrs lộ khoá OPENAI_API_KEY")
	}
}

// US-P2-01: cấu hình thư — mặc định dev, giá trị sai nêu tên biến, production ở worker bắt buộc https + TLS.
func TestLoad_Mail(t *testing.T) {
	t.Parallel()
	c, err := config.Load(getenv(full()), config.Worker)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if c.AppPublicURL != "https://localhost" || c.SMTPHost != "mailpit" || c.SMTPPort != 1025 || c.SMTPTLS != "none" ||
		c.VerifyTokenTTL != 24*time.Hour || c.ResetTokenTTL != 30*time.Minute || c.InviteTokenTTL != 72*time.Hour {
		t.Fatalf("mặc định sai: %+v", c)
	}

	bad := []struct{ name, key, val string }{
		{"URL tương đối", "APP_PUBLIC_URL", "localhost"},
		{"URL có đường dẫn", "APP_PUBLIC_URL", "https://localhost/app"},
		{"TLS lạ", "SMTP_TLS", "ssl"},
		{"cổng SMTP 0", "SMTP_PORT", "0"},
		{"MAIL_FROM sai", "MAIL_FROM", "không phải địa chỉ"},
	}
	for _, tc := range bad {
		env := full()
		env[tc.key] = tc.val
		_, err := config.Load(getenv(env), config.Worker)
		var ie *config.ErrInvalidEnv
		if !errors.As(err, &ie) || !slices.Contains(ie.Names, tc.key) {
			t.Errorf("%s: err = %v, muốn *ErrInvalidEnv nêu %s", tc.name, err, tc.key)
		}
	}

	prod := full()
	prod["APP_ENV"] = "production"
	_, err = config.Load(getenv(prod), config.Worker)
	var pe *config.ErrInvalidEnv
	if !errors.As(err, &pe) || !slices.Contains(pe.Names, "APP_PUBLIC_URL") {
		t.Errorf("worker production thiếu APP_PUBLIC_URL: err = %v", err)
	}
	prod["APP_PUBLIC_URL"] = "http://edupilot.example"
	prod["SMTP_TLS"] = "none"
	_, err = config.Load(getenv(prod), config.Worker)
	var ie *config.ErrInvalidEnv
	if !errors.As(err, &ie) || !slices.Contains(ie.Names, "APP_PUBLIC_URL") || !slices.Contains(ie.Names, "SMTP_TLS") {
		t.Errorf("worker production http + TLS none: err = %v", err)
	}
	prod["APP_PUBLIC_URL"] = "https://edupilot.example/"
	prod["SMTP_TLS"] = "starttls"
	c, err = config.Load(getenv(prod), config.Worker)
	if err != nil || c.AppPublicURL != "https://edupilot.example" {
		t.Errorf("production hợp lệ: c.AppPublicURL = %q, err = %v", c.AppPublicURL, err)
	}
}
