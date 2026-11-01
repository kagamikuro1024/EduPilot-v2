package submission_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/cmd/mock-graph/server"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/submission"
	"github.com/edupilot/backend-go/internal/submission/source"
	"github.com/edupilot/backend-go/internal/testutil"
)

type noSleep struct{}

func (noSleep) Sleep(context.Context, time.Duration) error { return nil }

type tjRig struct {
	t      *testing.T
	w      *testutil.TeamsWorld
	svc    *submission.TeamsSync
	runner *jobs.Runner
	ts     *httptest.Server
	rdb    *appredis.Client
	q      func(sql string, dst any, args ...any)
}

func newTJ(t *testing.T) *tjRig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	st, err := blob.New(t.Context(), blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "tj-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	cipher, err := crypto.ParseKey("ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=")
	require.NoError(t, err)
	w := testutil.NewTeamsWorld(t, pool)
	mock, err := server.New(server.Options{DataDir: testutil.MockGraphDataDir()})
	require.NoError(t, err)
	ts := httptest.NewServer(mock)
	t.Cleanup(ts.Close)
	meta := mock.Meta()
	testutil.PutTeamsConfig(t, pool, cipher.Encrypt, w.Course, meta.TenantID, meta.ClientID, meta.ClientSecret, true)
	r := &tjRig{t: t, w: w, ts: ts, rdb: rdb}
	r.svc = &submission.TeamsSync{Pool: pool, Redis: rdb, Blob: st, Cipher: cipher, Jobs: jobs.NewService(pool), Log: slog.New(slog.DiscardHandler), Enabled: true, Timeout: time.Minute, MinInterval: 30 * time.Second, Sleep: noSleep{},
		Source: source.Config{BaseURL: ts.URL + "/v1.0", AuthURL: ts.URL, DownloadParallel: 4, FileRetry: 2, RetryMax: 3, FileRetryWait: time.Millisecond}}
	r.runner = jobs.NewRunner(pool, nil, clock.Real{}, slog.New(slog.DiscardHandler))
	r.svc.Register(r.runner)
	r.q = func(sql string, dst any, args ...any) {
		t.Helper()
		require.NoError(t, pool.QueryRow(t.Context(), sql, args...).Scan(dst), sql)
	}
	return r
}

func (r *tjRig) scenario(name string) {
	r.t.Helper()
	b, _ := json.Marshal(map[string]string{"name": name})
	resp, err := http.Post(r.ts.URL+"/__mock/scenario", "application/json", bytes.NewReader(b))
	require.NoError(r.t, err)
	_ = resp.Body.Close()
}

func (r *tjRig) enqueue() submission.TeamsJob {
	r.t.Helper()
	j, err := r.svc.Enqueue(r.t.Context(), r.w.Teacher, r.w.Course, r.w.Asg)
	require.NoError(r.t, err)
	return j
}

func (r *tjRig) runJob(id uuid.UUID) {
	r.t.Helper()
	var payload []byte
	r.q(`select payload from outbox where topic = $1 and payload->>'job_id' = $2`, &payload, jobs.TopicEnqueue, id.String())
	require.NoError(r.t, r.runner.HandleMessage(r.t.Context(), outbox.Message{ID: uuid.New(), Topic: jobs.TopicEnqueue, Payload: payload}))
}

func (r *tjRig) job(id uuid.UUID) (status string, progress int, result, errj string) {
	r.t.Helper()
	r.q(`select status::text from jobs where id = $1`, &status, id)
	r.q(`select progress::int from jobs where id = $1`, &progress, id)
	r.q(`select coalesce(result::text, '') from jobs where id = $1`, &result, id)
	r.q(`select coalesce(error::text, '') from jobs where id = $1`, &errj, id)
	return
}

// TestTeamsSyncJobProgress — AC7: 202 + job_id; xong có kết quả {fetched, created, deduped, unmatched, failed[]} và tiến độ 100; dấu "đang chạy" được dọn.
func TestTeamsSyncJobProgress(t *testing.T) {
	r := newTJ(t)
	j := r.enqueue()
	r.runJob(j.JobID)
	status, progress, result, _ := r.job(j.JobID)
	require.Equal(t, "SUCCEEDED", status)
	require.Equal(t, 100, progress)
	var got submission.TeamsResult
	require.NoError(t, json.Unmarshal([]byte(result), &got))
	require.Equal(t, 25, got.Fetched)
	require.Equal(t, 25, got.Created)
	require.Equal(t, 1, got.Unmatched)
	require.Empty(t, got.Failed)
	exists, err := r.rdb.Exists(t.Context(), "ep:teamssync:"+r.w.Asg.String()+":job").Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

// TestTeamsSyncNoOverlap — AC7: đang có một đợt cho bài → trả lại job_id cũ, không xếp việc mới.
func TestTeamsSyncNoOverlap(t *testing.T) {
	r := newTJ(t)
	a := r.enqueue()
	b := r.enqueue()
	require.Equal(t, a.JobID, b.JobID)
	var n int
	r.q(`select count(*)::int from jobs where kind = $1`, &n, submission.KindTeamsSync)
	require.Equal(t, 1, n)
}

// TestTeamsSyncRateLimit — AC7: ≤ 1 lần / 30 giây / bài (429 + retry_after) khi đợt trước đã xong; kênh / cấu hình / TEAMS_ENABLED được kiểm.
func TestTeamsSyncRateLimit(t *testing.T) {
	r := newTJ(t)
	j := r.enqueue()
	r.runJob(j.JobID)
	_, err := r.svc.Enqueue(t.Context(), r.w.Teacher, r.w.Course, r.w.Asg)
	ae, ok := err.(*apierr.Error) //nolint:errorlint // trả thẳng
	require.True(t, ok)
	require.Equal(t, http.StatusTooManyRequests, ae.Status)
	require.Positive(t, ae.RetryAfter)
	require.LessOrEqual(t, ae.RetryAfter, 30)
	r.svc.Enabled = false
	_, err = r.svc.Enqueue(t.Context(), r.w.Teacher, r.w.Course, r.w.Asg)
	require.Equal(t, apierr.TeamsDisabled, err.(*apierr.Error).Code) //nolint:errorlint // trả thẳng
}

// TestTeamsSyncChannelNotAllowed — AC7: bài không bật kênh TEAMS → 409 CHANNEL_NOT_ALLOWED.
func TestTeamsSyncChannelNotAllowed(t *testing.T) {
	r := newTJ(t)
	_, err := r.svc.Pool.Exec(t.Context(), `update assignments set channels = ARRAY['IN_APP'], teams_class_id = null, teams_assignment_id = null where id = $1`, r.w.Asg)
	require.NoError(t, err)
	_, err = r.svc.Enqueue(t.Context(), r.w.Teacher, r.w.Course, r.w.Asg)
	require.Equal(t, apierr.ChannelNotAllowed, err.(*apierr.Error).Code) //nolint:errorlint // trả thẳng
}

// TestTeamsSyncPartialListed — AC7: lỗi từng bài hiện riêng trong kết quả (graph_submission_id + code) mà không làm hỏng cả đợt (việc vẫn SUCCEEDED).
func TestTeamsSyncPartialListed(t *testing.T) {
	r := newTJ(t)
	r.scenario("partial_failure")
	j := r.enqueue()
	r.runJob(j.JobID)
	status, _, result, _ := r.job(j.JobID)
	require.Equal(t, "SUCCEEDED", status)
	var got submission.TeamsResult
	require.NoError(t, json.Unmarshal([]byte(result), &got))
	require.Equal(t, 24, got.Created)
	require.Len(t, got.Failed, 1)
	require.Equal(t, source.CodeFileDownload, got.Failed[0].Code)
	require.NotEmpty(t, got.Failed[0].SubmissionID)
}

// TestTeamsSyncConsentJobError — AC7: TEAMS_CONSENT_REQUIRED hiện lỗi đọc hiểu được ở việc (mã + câu tiếng thường), không trang trắng.
func TestTeamsSyncConsentJobError(t *testing.T) {
	r := newTJ(t)
	r.scenario("no_consent")
	j := r.enqueue()
	r.runJob(j.JobID)
	status, _, _, errj := r.job(j.JobID)
	require.Equal(t, "FAILED", status)
	require.Contains(t, errj, "TEAMS_CONSENT_REQUIRED")
	require.Contains(t, errj, "quyền đọc bài tập")
}
