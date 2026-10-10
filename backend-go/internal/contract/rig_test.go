package contract

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	appotel "github.com/edupilot/backend-go/internal/platform/otel"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const rigSecret = "contract-test-secret-0123456789abcdef-dev"

// rig là gateway thật (router đầy đủ) trên Postgres đã migrate + Redis thật, dùng chung cho cả gói test.
type rig struct {
	deps   httpapi.Deps
	handle http.Handler
	srv    *httptest.Server
	cfg    config.Config
}

var (
	rigOnce sync.Once
	rigVal  *rig
	rigErr  error
)

func getRig(t *testing.T) *rig {
	t.Helper()
	testutil.RequireContainers(t)
	rigOnce.Do(func() { rigVal, rigErr = buildRig(t) })
	if rigErr != nil {
		t.Fatalf("dựng gateway thử: %v", rigErr)
	}
	return rigVal
}

func buildRig(t *testing.T) (*rig, error) {
	env := map[string]string{
		"DATABASE_URL":            testutil.MigratedPostgresURL(t),
		"REDIS_URL":               testutil.RedisURL(t),
		"JWT_SECRET_KEY":          rigSecret,
		"BLOB_ENDPOINT":           "127.0.0.1:9",
		"BLOB_BUCKET":             "contract",
		"BLOB_ACCESS_KEY":         "contract-dev",
		"BLOB_SECRET_KEY":         "contract-dev-secret",
		"APP_ENCRYPTION_KEY":      "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=",
		"APP_ENV":                 "test",
		"REQUEST_TIMEOUT":         "1s",
		"MAX_BODY_BYTES":          "4096",
		"EXAM_TESTZIP_MAX_BYTES":  "65536", // US-PE-03: zip lớn hơn 64 KiB → 413 (kịch bản hợp đồng không phải gửi 10 MiB)
		"RATE_LIMIT_IP_PER_MIN":   "1000000",
		"RATE_LIMIT_USER_PER_MIN": "1000000",
		"SSE_MAX_PER_USER":        "2",
		"SSE_HEARTBEAT":           "1s",
		"SSE_MAX_DURATION":        "30s",
		"INSTANCE_ID":             "contract-gw",
		"DB_MAX_CONNS":            "8",
		"LOG_LEVEL":               "error",
		"LLM_PROVIDER":            "fake",
		"BCRYPT_COST":             "4", // REQUEST_TIMEOUT=1s: bcrypt cost 12 dưới -race vượt hạn
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	if _, err := appotel.Setup(ctx, cfg); err != nil {
		return nil, err
	}
	log := applog.NewTo(io.Discard, cfg, "gateway")
	pool, err := appdb.NewPool(ctx, cfg, log)
	if err != nil {
		return nil, err
	}
	rdb, err := appredis.New(ctx, cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	rt, err := llmrt.New(ctx, cfg, pool, rdb.Client, log)
	if err != nil {
		return nil, err
	}
	// Kho đối tượng thật (MinIO dùng chung) cho tải tài liệu lên bằng URL ký sẵn (US-P8-01).
	store, err := blob.New(ctx, blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "contract", AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	if err != nil {
		return nil, err
	}
	deps := httpapi.Deps{Cfg: cfg, Log: log, DB: pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState(), Jobs: jobs.NewService(pool), LLM: rt, Blob: store}
	h := httpapi.NewRouter(deps)
	return &rig{deps: deps, handle: h, srv: httptest.NewServer(h), cfg: cfg}, nil
}

// token ký JWT cho một người; sub rỗng → ngẫu nhiên do caller cấp.
func (r *rig) token(t *testing.T, sub string, role auth.Role) string {
	t.Helper()
	tok, err := auth.NewIssuer(rigSecret, time.Hour, clock.Real{}).Issue(sub, role, "")
	if err != nil {
		t.Fatalf("ký token: %v", err)
	}
	return tok
}
