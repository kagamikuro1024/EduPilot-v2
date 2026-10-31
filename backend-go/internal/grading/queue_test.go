package grading_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/testutil"
)

// qrig: hàng đợi chấm trên Postgres + Redis thật; LLM giả trả bản ghi của mẫu S1 cho mọi bài.
type qrig struct {
	*gradeEnv
	rdb     *appredis.Client
	prefix  string
	runner  *jobs.Runner
	batches *grading.Batches
	jobs    *jobs.Service
	text    string
	llmErr  atomic.Value // func(n int) error
	calls   atomic.Int64
	delay   atomic.Int64 // ns
	nstu    int
	cfg     config.Config
	reg     bool
	first   *grading.Queue
}

func newQRig(t *testing.T) *qrig {
	t.Helper()
	testutil.RequireContainers(t)
	ss := samples(t)
	b, err := os.ReadFile(asgFix + ss[0].File)
	require.NoError(t, err)
	r := &qrig{text: string(b)}
	r.gradeEnv = newGradeEnv(t, func(g *gradeEnv, req llm.Request) (json.RawMessage, error) {
		n := int(r.calls.Add(1))
		if d := time.Duration(r.delay.Load()); d > 0 {
			time.Sleep(d)
		}
		if f, ok := r.llmErr.Load().(func(int) error); ok && f != nil {
			if err := f(n); err != nil {
				return nil, err
			}
		}
		second := strings.Contains(strings.SplitN(req.Messages[1].Content, "\n", 2)[0], "c4, c3, c2, c1")
		if second {
			return ss[0].P2, nil
		}
		return ss[0].P1, nil
	})
	cfg, err := config.Load(func(k string) string {
		return map[string]string{"DATABASE_URL": "postgres://x", "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": rbacSecret, "BLOB_ENDPOINT": "127.0.0.1:9", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
			"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test"}[k]
	}, config.Gateway)
	require.NoError(t, err)
	r.cfg = cfg
	r.rdb, err = appredis.New(t.Context(), cfg.RedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.rdb.Close() })
	r.prefix = testutil.TestPrefix(t) + ":"
	r.runner = jobs.NewRunner(r.pool, sse.NewPublisher(r.rdb, 1000, time.Hour), clock.Real{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.jobs = jobs.NewService(r.pool)
	r.batches = &grading.Batches{Pool: r.pool, Jobs: r.jobs}
	return r
}

func (r *qrig) queue(mut func(*grading.QueueConfig)) *grading.Queue {
	cfg := grading.QueueConfig{Workers: 2, Lease: 2 * time.Second, Heartbeat: 400 * time.Millisecond, RequeueAfter: time.Second, Attempts: 3,
		Backoff: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}, Poll: 50 * time.Millisecond, Tick: 100 * time.Millisecond}
	if mut != nil {
		mut(&cfg)
	}
	pub := sse.NewPublisher(r.rdb, 1000, time.Hour)
	q := grading.NewQueue(&grading.Queue{Pool: r.pool, Redis: r.rdb, Grader: r.grader, Jobs: r.runner, Notify: notify.New(r.pool, pub, slog.New(slog.NewTextHandler(io.Discard, nil))),
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Cfg: cfg, Prefix: r.prefix, Name: "w-" + uuid.NewString()[:4]})
	if !r.reg {
		q.RegisterBatch(r.runner)
		r.reg = true
		r.first = q
	}
	return q
}

func (r *qrig) start(q *grading.Queue) {
	ctx, stop := context.WithCancel(r.ctx)
	done := make(chan struct{})
	go func() { defer close(done); _ = q.Run(ctx) }()
	r.t.Cleanup(func() { stop(); <-done })
}

// sub tạo sinh viên + bài nộp phiên bản `version` (MATCHED hoặc UNMATCHED); chưa có kết quả chấm.
func (r *qrig) sub(student uuid.UUID, version int, match string) uuid.UUID {
	r.t.Helper()
	if student == uuid.Nil {
		r.nstu++
		student = r.user("STUDENT", fmt.Sprintf("SV %d", r.nstu))
		r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, r.course, student)
	}
	key := "courses/" + r.course.String() + "/" + uuid.NewString() + "/bai.md"
	r.blob.m[key] = []byte(r.text)
	var id uuid.UUID
	if match != "MATCHED" {
		r.one(&id, `insert into submissions (course_id, source, blob_key, file_name, file_size, file_mime, sha256, match_status, unmatched_reason, submitted_at)
			values ($1, 'IMAP', $2, 'bai.md', $3, 'text/plain', 'x', 'UNMATCHED', 'NO_STUDENT_SIGNAL', now()) returning id`, r.course, key, len(r.text))
		return id
	}
	r.one(&id, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at)
		values ($1, $2, $3, 'IN_APP', $4, $5, 'bai.md', $6, 'text/plain', 'x', 'MATCHED', 'IN_APP', now()) returning id`, r.course, r.asg, student, version, key, len(r.text))
	return id
}

func (r *qrig) student(sub uuid.UUID) (s uuid.UUID) {
	r.one(&s, `select student_id from submissions where id = $1`, sub)
	return s
}

// deliver giao tin outbox `job.enqueue` của việc cho runner (thay relay + consumer outbox của worker).
func (r *qrig) deliver(job uuid.UUID) {
	r.t.Helper()
	var payload []byte
	r.one(&payload, `select payload from outbox where topic = $1 and payload->>'job_id' = $2`, jobs.TopicEnqueue, job.String())
	require.NoError(r.t, r.runner.HandleMessage(r.ctx, outbox.Message{ID: uuid.New(), Topic: jobs.TopicEnqueue, Payload: payload}))
}

func (r *qrig) count(sql string, args ...any) (n int) {
	r.one(&n, sql, args...)
	return n
}

func (r *qrig) states() map[string]int {
	rows, err := r.pool.Query(r.ctx, `select run_state, count(*)::int from grading_results where course_id = $1 group by 1`, r.course)
	require.NoError(r.t, err)
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		require.NoError(r.t, rows.Scan(&s, &n))
		m[s] = n
	}
	return m
}

func (r *qrig) eventually(cond func() bool, msg string) {
	r.t.Helper()
	require.Eventually(r.t, cond, 20*time.Second, 50*time.Millisecond, msg)
}

func (r *qrig) gradeAll(retry bool) grading.Enqueued {
	r.t.Helper()
	e, err := r.batches.GradeAll(r.ctx, r.course, r.asg, r.teacher, retry)
	require.NoError(r.t, err)
	return e
}

func (r *qrig) jobRow(id uuid.UUID) (status string, progress int, result string) {
	var res []byte
	r.one(&status, `select status::text from jobs where id = $1`, id)
	r.one(&progress, `select progress::int from jobs where id = $1`, id)
	r.one(&res, `select coalesce(result, '{}'::jsonb) from jobs where id = $1`, id)
	return status, progress, string(res)
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	_, c := apiCode(t, err)
	return c
}

// TestGradeAllEnqueuesOnlyUngraded — AC1: chỉ bài MATCHED mới nhất, chưa có kết quả; hàng QUEUED ghi trong giao dịch (không cần Redis); UNMATCHED / bản cũ / đã có kết quả không xếp.
func TestGradeAllEnqueuesOnlyUngraded(t *testing.T) {
	r := newQRig(t)
	fresh := r.sub(uuid.Nil, 1, "MATCHED")
	graded := r.sub(uuid.Nil, 1, "MATCHED")
	r.exec(`insert into grading_results (course_id, submission_id, run_state) values ($1, $2, 'DONE')`, r.course, graded)
	running := r.sub(uuid.Nil, 1, "MATCHED")
	r.exec(`insert into grading_results (course_id, submission_id, run_state, lease_owner, lease_until) values ($1, $2, 'RUNNING', 'x', now() + interval '1 hour')`, r.course, running)
	r.sub(uuid.Nil, 1, "UNMATCHED")
	e := r.gradeAll(false)
	require.Equal(t, 1, e.Queued)
	require.NotNil(t, e.JobID)
	var subID uuid.UUID
	var round int
	var state string
	r.one(&subID, `select submission_id from grading_results where course_id = $1 and run_state = 'QUEUED'`, r.course)
	r.one(&round, `select round::int from grading_results where course_id = $1 and run_state = 'QUEUED'`, r.course)
	r.one(&state, `select run_state from grading_results where submission_id = $1`, fresh)
	require.Equal(t, fresh, subID)
	require.Equal(t, 1, round)
	require.Equal(t, "QUEUED", state)
	require.Equal(t, 0, r.count(`select count(*)::int from outbox where topic = 'grading.batch'`), "XADD không đi qua outbox; chỉ jobs + job.enqueue")
	n, err := r.rdb.XLen(r.ctx, r.prefix+grading.StreamGrading).Result()
	require.NoError(t, err)
	require.Zero(t, n, "chưa XADD trước khi handler sau COMMIT chạy")
}

// TestGradeAllOnlyLatestVersion — AC7: sinh viên nộp lại thì chỉ phiên bản mới nhất được xếp.
func TestGradeAllOnlyLatestVersion(t *testing.T) {
	r := newQRig(t)
	first := r.sub(uuid.Nil, 1, "MATCHED")
	stu := r.student(first)
	second := r.sub(stu, 2, "MATCHED")
	e := r.gradeAll(false)
	require.Equal(t, 1, e.Queued)
	require.Equal(t, 0, r.count(`select count(*)::int from grading_results where submission_id = $1`, first))
	require.Equal(t, 1, r.count(`select count(*)::int from grading_results where submission_id = $1`, second))
}

// TestGradeAllNothingToGrade — AC1: không có bài mới → 200 {queued:0}, không tạo việc.
func TestGradeAllNothingToGrade(t *testing.T) {
	r := newQRig(t)
	e := r.gradeAll(false)
	require.Nil(t, e.JobID)
	require.Zero(t, e.Queued)
	require.Zero(t, r.count(`select count(*)::int from jobs where kind = 'grading.batch'`))
	r.sub(uuid.Nil, 1, "MATCHED")
	require.Equal(t, 1, r.gradeAll(false).Queued)
	require.Zero(t, r.gradeAll(false).Queued, "lần hai: bài đã QUEUED, không xếp lại")
}

// TestGradeAllProgressSSE — AC1: chấm cả lô từ đầu tới cuối; tiến độ ghi vào jobs.progress, xong → result {graded, failed}.
func TestGradeAllProgressSSE(t *testing.T) {
	r := newQRig(t)
	const n = 6
	for range n {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	e := r.gradeAll(false)
	require.Equal(t, n, e.Queued)
	r.start(r.queue(nil))
	r.deliver(*e.JobID)
	r.eventually(func() bool { s, _, _ := r.jobRow(*e.JobID); return s == "SUCCEEDED" }, "lô xong")
	_, p, res := r.jobRow(*e.JobID)
	require.Equal(t, 100, p)
	require.JSONEq(t, `{"graded":6,"failed":0,"flagged":0}`, res)
	require.Equal(t, map[string]int{"DONE": n}, r.states())
	require.Equal(t, 2*n, int(r.calls.Load()), "hai lượt mỗi bài, không chấm đôi")
}

// TestGradeAllConcurrentNoDuplicate — AC2: mười request song song + hai giảng viên: mỗi (bài nộp, round) đúng một dòng.
func TestGradeAllConcurrentNoDuplicate(t *testing.T) {
	r := newQRig(t)
	for range 8 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	var wg sync.WaitGroup
	var total atomic.Int64
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, err := r.batches.GradeAll(r.ctx, r.course, r.asg, r.teacher, false)
			if err == nil {
				total.Add(int64(e.Queued))
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 8, total.Load(), "tổng bài được xếp đúng bằng số bài")
	require.Equal(t, 8, r.count(`select count(*)::int from grading_results where course_id = $1`, r.course))
	require.Zero(t, r.count(`select count(*)::int from (select submission_id, round from grading_results group by 1, 2 having count(*) > 1) x`))
}

// TestRegradeOnlyDraftOrFailed — AC2: chấm lại round+1 chỉ khi DRAFT (xong) / FAILED; REVIEWED / PUBLISHED → ALREADY_REVIEWED; đang chạy → VERSION_CONFLICT.
func TestRegradeOnlyDraftOrFailed(t *testing.T) {
	r := newQRig(t)
	for _, c := range []struct{ state, status, want string }{
		{"DONE", "DRAFT", ""}, {"FAILED", "DRAFT", ""},
		{"DONE", "REVIEWED", apierr.AlreadyReviewed}, {"DONE", "PUBLISHED", apierr.AlreadyReviewed},
		{"RUNNING", "DRAFT", apierr.VersionConflict}, {"QUEUED", "DRAFT", apierr.VersionConflict},
	} {
		sub := r.sub(uuid.Nil, 1, "MATCHED")
		r.exec(`insert into grading_results (course_id, submission_id, run_state, status, published_at) values ($1, $2, $3, $4::text, case when $4::text = 'PUBLISHED' then now() end)`, r.course, sub, c.state, c.status)
		e, err := r.batches.GradeOne(r.ctx, r.course, sub, r.teacher)
		if c.want != "" {
			require.Equal(t, c.want, codeOf(t, err), c.state+"/"+c.status)
			continue
		}
		require.NoError(t, err, c.state)
		require.Equal(t, 1, e.Queued)
		require.Equal(t, 2, r.count(`select max(round)::int from grading_results where submission_id = $1`, sub))
	}
}

// TestRegradeConflict — AC8: hai nhân viên cùng bấm Chấm lại một bài → một thắng, người sau VERSION_CONFLICT.
func TestRegradeConflict(t *testing.T) {
	r := newQRig(t)
	sub := r.sub(uuid.Nil, 1, "MATCHED")
	r.exec(`insert into grading_results (course_id, submission_id, run_state) values ($1, $2, 'DONE')`, r.course, sub)
	var wg sync.WaitGroup
	var ok, conflict atomic.Int64
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.batches.GradeOne(r.ctx, r.course, sub, r.teacher)
			switch {
			case err == nil:
				ok.Add(1)
			case codeOf(t, err) == apierr.VersionConflict:
				conflict.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, ok.Load())
	require.EqualValues(t, 1, conflict.Load())
	require.Equal(t, 2, r.count(`select count(*)::int from grading_results where submission_id = $1`, sub))
}
