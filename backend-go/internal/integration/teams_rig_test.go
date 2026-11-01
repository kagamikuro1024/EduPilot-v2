package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/cmd/mock-graph/server"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/submission"
	"github.com/edupilot/backend-go/internal/submission/source"
	"github.com/edupilot/backend-go/internal/testutil"
)

// tmSleeper ghi lại các lần chờ mà không ngủ thật.
type tmSleeper struct {
	mu    sync.Mutex
	waits []time.Duration
}

func (s *tmSleeper) Sleep(_ context.Context, d time.Duration) error {
	s.mu.Lock()
	s.waits = append(s.waits, d)
	s.mu.Unlock()
	return nil
}

// tmRig: Postgres + Redis + MinIO thật, lớp có đủ sinh viên của teams-plan.csv, mock-graph (thư viện `cmd/mock-graph/server`) chạy qua httptest.
type tmRig struct {
	t      *testing.T
	pool   *pgxpool.Pool
	rdb    *appredis.Client
	blob   *blob.Store
	cipher *crypto.Cipher
	w      *testutil.TeamsWorld
	meta   server.Meta
	mock   *server.Server
	ts     *httptest.Server
	sl     *tmSleeper
	logs   *bytes.Buffer
	sync   *submission.TeamsSync
}

func newTM(t *testing.T) *tmRig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	st, err := blob.New(t.Context(), blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "tm-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	cipher, err := crypto.ParseKey("ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=")
	require.NoError(t, err)
	r := &tmRig{t: t, pool: pool, rdb: rdb, blob: st, cipher: cipher, sl: &tmSleeper{}, logs: &bytes.Buffer{}}
	r.w = testutil.NewTeamsWorld(t, pool)
	r.mock, r.ts = r.startMock(testutil.MockGraphDataDir())
	r.meta = r.mock.Meta()
	testutil.PutTeamsConfig(t, pool, cipher.Encrypt, r.w.Course, r.meta.TenantID, r.meta.ClientID, r.meta.ClientSecret, true)
	r.rebuild(r.ts.URL)
	return r
}

func (r *tmRig) startMock(dir string) (*server.Server, *httptest.Server) {
	r.t.Helper()
	srv, err := server.New(server.Options{DataDir: dir})
	require.NoError(r.t, err)
	ts := httptest.NewServer(srv)
	r.t.Cleanup(ts.Close)
	return srv, ts
}

func (r *tmRig) rebuild(base string) {
	r.sync = &submission.TeamsSync{Pool: r.pool, Redis: r.rdb, Blob: r.blob, Cipher: r.cipher, Jobs: jobs.NewService(r.pool), Log: slog.New(slog.NewTextHandler(r.logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Enabled: true, Timeout: 120 * time.Second, MinInterval: 30 * time.Second, Sleep: r.sl,
		Source: source.Config{BaseURL: base + "/v1.0", AuthURL: base, DownloadParallel: 4, FileRetry: 2, RetryMax: 3, FileRetryWait: time.Millisecond}}
}

func (r *tmRig) scenario(name string) {
	r.t.Helper()
	b, _ := json.Marshal(map[string]string{"name": name})
	resp, err := http.Post(r.ts.URL+"/__mock/scenario", "application/json", bytes.NewReader(b))
	require.NoError(r.t, err)
	_ = resp.Body.Close()
	require.Equal(r.t, 200, resp.StatusCode)
}

type mockLog struct {
	Scenario      string `json:"scenario"`
	TokenRequests int    `json:"token_requests"`
	Throttled     int    `json:"throttled"`
	Requests      []struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Status int    `json:"status"`
	} `json:"requests"`
}

func (r *tmRig) requests() (l mockLog) {
	r.t.Helper()
	resp, err := http.Get(r.ts.URL + "/__mock/requests")
	require.NoError(r.t, err)
	defer func() { _ = resp.Body.Close() }()
	require.NoError(r.t, json.NewDecoder(resp.Body).Decode(&l))
	return l
}

func (l mockLog) count(suffix string) (n int) {
	for _, q := range l.Requests {
		if strings.HasSuffix(q.Path, suffix) {
			n++
		}
	}
	return n
}

func (r *tmRig) run() (submission.TeamsResult, error) {
	return r.sync.Run(r.t.Context(), r.w.Course, r.w.Asg, nil)
}

func (r *tmRig) mustRun() submission.TeamsResult {
	r.t.Helper()
	res, err := r.run()
	require.NoError(r.t, err)
	return res
}

func (r *tmRig) subs() (n int) {
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select count(*)::int from submissions where course_id = $1 and source = 'TEAMS'`, r.w.Course).Scan(&n))
	return n
}

func (r *tmRig) count(sql string, args ...any) (n int) {
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&n), sql)
	return n
}

// submissionID của tài khoản trong dữ liệu mock-graph.
func submissionIDOf(t *testing.T, dir, account string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "submissions.json"))
	require.NoError(t, err)
	var all []map[string]any
	require.NoError(t, json.Unmarshal(b, &all))
	for _, s := range all {
		if s["account"] == account {
			return s["id"].(string)
		}
	}
	t.Fatalf("không có tài khoản %s", account)
	return ""
}

func sourceCode(err error) string {
	var se *source.Error
	if errors.As(err, &se) {
		return se.Code
	}
	return ""
}
