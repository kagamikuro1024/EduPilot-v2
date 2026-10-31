package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const asgFixDir = "../../../docs/specs/FEAT-assignments/fixtures/"

// gqRig: Postgres + Redis thật, một bài nộp MATCHED + hàng chấm QUEUED, LLM giả (mẫu S1) có độ trễ / đếm lời gọi.
type gqRig struct {
	t      *testing.T
	rdb    *appredis.Client
	calls  atomic.Int64
	delay  atomic.Int64
	tasks  []llm.Task
	mu     sync.Mutex
	result uuid.UUID
	grader *grading.Grader
	prefix string
	q      func(lease, hb time.Duration, name string) *grading.Queue
	one    func(dst any, sql string, args ...any)
	exec   func(sql string, args ...any)
}

type blobMem map[string][]byte

func (b blobMem) Get(_ context.Context, k string) (io.ReadCloser, error) {
	v, ok := b[k]
	if !ok {
		return nil, errors.New("không có object")
	}
	return io.NopCloser(bytes.NewReader(v)), nil
}

type noExtract struct{}

func (noExtract) Extract(context.Context, string, io.Reader) (string, error) {
	return "", errors.New("không dùng")
}

func newGQ(t *testing.T) *gqRig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	ctx := t.Context()
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := &gqRig{t: t, rdb: rdb, prefix: testutil.TestPrefix(t) + ":"}
	r.exec = func(sql string, args ...any) {
		_, err := pool.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	r.one = func(dst any, sql string, args ...any) {
		require.NoError(t, pool.QueryRow(ctx, sql, args...).Scan(dst), sql)
	}

	var samples []struct {
		File string          `json:"file"`
		P1   json.RawMessage `json:"llm_pass1"`
		P2   json.RawMessage `json:"llm_pass2"`
	}
	raw, err := os.ReadFile(asgFixDir + "expected_grading.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &samples))
	text, err := os.ReadFile(asgFixDir + samples[0].File)
	require.NoError(t, err)
	var rb struct {
		Criteria json.RawMessage `json:"criteria"`
	}
	rubricRaw, err := os.ReadFile(asgFixDir + "rubric-bt03.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rubricRaw, &rb))
	ref, err := os.ReadFile(asgFixDir + "dap-an-chuan-bt03.md")
	require.NoError(t, err)

	var teacher, student, course, asg, sub uuid.UUID
	r.one(&teacher, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test")
	r.one(&student, `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, uuid.NewString()+"@example.test")
	id := uuid.New()
	r.one(&course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', 'ABCDEFG', $2) returning id`, "GQ"+id.String()[:8], teacher)
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, course, student)
	r.one(&asg, `insert into assignments (course_id, title, description, max_score, due_at, allow_late, late_penalty_per_day, late_max_days, rubric, reference_answer, submission_code, created_by)
		values ($1, 'BT', 'x', 10, now() + interval '1 day', true, 0.5, 2, $2::jsonb, $3, 'BT03', $4) returning id`, course, string(rb.Criteria), string(ref), teacher)
	key := "courses/" + course.String() + "/" + uuid.NewString() + "/bai.md"
	r.one(&sub, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at)
		values ($1, $2, $3, 'IN_APP', 1, $4, 'bai.md', $5, 'text/plain', 'x', 'MATCHED', 'IN_APP', now()) returning id`, course, asg, student, key, len(text))
	r.one(&r.result, `insert into grading_results (course_id, submission_id) values ($1, $2) returning id`, course, sub)

	r.grader = &grading.Grader{Pool: pool, Blob: blobMem{key: text}, Extract: noExtract{}, Cfg: grading.DefaultEssayConfig(),
		LLM: structFn(func(_ context.Context, req llm.Request, _ json.RawMessage) (json.RawMessage, error) {
			r.calls.Add(1)
			r.mu.Lock()
			r.tasks = append(r.tasks, req.Task)
			r.mu.Unlock()
			if d := time.Duration(r.delay.Load()); d > 0 {
				time.Sleep(d)
			}
			if strings.Contains(strings.SplitN(req.Messages[1].Content, "\n", 2)[0], "c4, c3, c2, c1") {
				return samples[0].P2, nil
			}
			return samples[0].P1, nil
		})}
	r.q = func(lease, hb time.Duration, name string) *grading.Queue {
		return grading.NewQueue(&grading.Queue{Pool: pool, Redis: rdb, Grader: r.grader, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Prefix: r.prefix, Name: name,
			Cfg: grading.QueueConfig{Workers: 1, Lease: lease, Heartbeat: hb, RequeueAfter: time.Hour, Attempts: 3, Poll: 30 * time.Millisecond, Tick: time.Hour}})
	}
	return r
}

type structFn func(context.Context, llm.Request, json.RawMessage) (json.RawMessage, error)

func (f structFn) Structured(ctx context.Context, r llm.Request, s json.RawMessage) (json.RawMessage, error) {
	return f(ctx, r, s)
}

func (r *gqRig) run(q *grading.Queue) {
	ctx, stop := context.WithCancel(r.t.Context())
	done := make(chan struct{})
	go func() { defer close(done); _ = q.Run(ctx) }()
	r.t.Cleanup(func() { stop(); <-done })
}

func (r *gqRig) state() (s string) {
	r.one(&s, `select run_state from grading_results where id = $1`, r.result)
	return s
}

// TestGradingJobSurvivesWorkerKill — AC3: worker chết ngay sau khi nhận việc (thuê hết hạn, tin treo trong PEL) → worker khác nhận lại, đúng một kết quả, ≤ 2 lời gọi LLM, không mất bài; worker chậm quay lại bị bỏ kết quả.
func TestGradingJobSurvivesWorkerKill(t *testing.T) {
	r := newGQ(t)
	ctx := t.Context()
	stream := r.prefix + grading.StreamGrading
	var course, sub uuid.UUID
	r.one(&course, `select course_id from grading_results where id = $1`, r.result)
	r.one(&sub, `select submission_id from grading_results where id = $1`, r.result)
	// worker "chết": đã nhận (RUNNING, thuê đã hết) và đã đọc tin nhưng không bao giờ XACK
	require.NoError(t, r.rdb.XGroupCreateMkStream(ctx, stream, grading.GroupGrading, "0").Err())
	require.NoError(t, r.rdb.XAdd(ctx, &goredis.XAddArgs{Stream: stream, Values: map[string]any{"result_id": r.result.String(), "course_id": course.String(), "submission_id": sub.String(), "round": "1"}}).Err())
	require.NoError(t, r.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: grading.GroupGrading, Consumer: "dead-worker", Streams: []string{stream, ">"}, Count: 1, Block: -1}).Err())
	r.exec(`update grading_results set run_state = 'RUNNING', lease_owner = 'dead-worker', lease_until = now() - interval '1 second', attempts = 1 where id = $1`, r.result)

	r.run(r.q(300*time.Millisecond, 100*time.Millisecond, "survivor"))
	require.Eventually(t, func() bool { return r.state() == "DONE" }, 20*time.Second, 50*time.Millisecond, "bài được nhận lại")
	var n, attempts int
	r.one(&n, `select count(*)::int from grading_results where submission_id = $1`, sub)
	r.one(&attempts, `select attempts::int from grading_results where id = $1`, r.result)
	require.Equal(t, 1, n, "đúng một kết quả")
	require.Equal(t, 2, attempts)
	require.EqualValues(t, 2, r.calls.Load(), "hai lượt chấm, không chấm đôi")

	// worker chậm quay lại sau khi bài xong → ghi bị bỏ
	var before string
	r.one(&before, `select version::text || total::text from grading_results where id = $1`, r.result)
	_, err := r.grader.Run(grading.WithLease(ctx, "dead-worker"), course, r.result)
	require.ErrorIs(t, err, grading.ErrLeaseLost)
	var after string
	r.one(&after, `select version::text || total::text from grading_results where id = $1`, r.result)
	require.Equal(t, before, after, "kết quả của worker cũ không đè")
}

// TestSlowGradeKeepsLeaseByHeartbeat — AC3: bài chậm hơn thuê vẫn giữ được thuê nhờ nhịp gia hạn; consumer thứ hai không nhận lại oan; chỉ một lần chạy (không trả tiền LLM hai lần).
func TestSlowGradeKeepsLeaseByHeartbeat(t *testing.T) {
	r := newGQ(t)
	r.delay.Store(int64(1500 * time.Millisecond))
	stream := r.prefix + grading.StreamGrading
	var course, sub uuid.UUID
	r.one(&course, `select course_id from grading_results where id = $1`, r.result)
	r.one(&sub, `select submission_id from grading_results where id = $1`, r.result)
	r.run(r.q(500*time.Millisecond, 120*time.Millisecond, "slow-a"))
	r.run(r.q(500*time.Millisecond, 120*time.Millisecond, "slow-b"))
	time.Sleep(200 * time.Millisecond) // nhóm đã tạo
	require.NoError(t, r.rdb.XAdd(t.Context(), &goredis.XAddArgs{Stream: stream, Values: map[string]any{"result_id": r.result.String(), "course_id": course.String(), "submission_id": sub.String(), "round": "1"}}).Err())
	require.Eventually(t, func() bool { return r.state() == "DONE" }, 20*time.Second, 50*time.Millisecond)
	var attempts int
	r.one(&attempts, `select attempts::int from grading_results where id = $1`, r.result)
	require.Equal(t, 1, attempts, "không bị nhận lại oan")
	require.EqualValues(t, 2, r.calls.Load())
}

// TestGradingUsesBatchLane — AC5: mọi lời gọi chấm là task GRADING → làn BATCH; xin INTERACTIVE → ErrBadLane; SuggestRubric dùng NEAR_REALTIME.
func TestGradingUsesBatchLane(t *testing.T) {
	r := newGQ(t)
	var course uuid.UUID
	r.one(&course, `select course_id from grading_results where id = $1`, r.result)
	_, err := r.grader.Run(t.Context(), course, r.result)
	require.NoError(t, err)
	require.NotEmpty(t, r.tasks)
	for _, task := range r.tasks {
		require.Equal(t, llm.TaskGrading, task)
	}
	lane, err := llm.ResolveLane(llm.TaskGrading, nil)
	require.NoError(t, err)
	require.Equal(t, llm.LaneBatch, lane)
	inter := llm.LaneInteractive
	_, err = llm.ResolveLane(llm.TaskGrading, &inter)
	require.ErrorIs(t, err, llm.ErrBadLane)
	// SuggestRubric dùng task UTILITY (suggest_test.go khẳng định) → làn NEAR_REALTIME, không tranh chỗ BATCH của chấm.
	require.Equal(t, llm.LaneNearRealtime, llm.DefaultLane(llm.TaskUtility))
}

// TestChatUnaffectedDuringBatchGrading — AC5: 200 việc chấm (BATCH) đang xếp hàng, chat (INTERACTIVE) vẫn vào ngay: chờ hàng p95 ≤ 500 ms.
func TestChatUnaffectedDuringBatchGrading(t *testing.T) {
	s := scheduler.New(scheduler.Config{MaxConcurrency: 4, BatchShare: 0.5, QueueMax: 1000}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	work := func(lane llm.Lane, task llm.Task) llm.Work {
		return llm.Work{Lane: lane, Task: task, ProviderID: "p", RPM: 1_000_000, TPM: 1_000_000_000, EstTokens: 10}
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				ctx, cancel := context.WithTimeout(t.Context(), time.Second)
				p, err := s.Admit(ctx, work(llm.LaneBatch, llm.TaskGrading))
				cancel()
				if err != nil {
					continue
				}
				time.Sleep(40 * time.Millisecond)
				p.Done(-1)
			}
		}()
	}
	t.Cleanup(func() { close(stop); wg.Wait() })
	time.Sleep(300 * time.Millisecond)
	var waits []time.Duration
	for range 20 {
		start := time.Now()
		p, err := s.Admit(t.Context(), work(llm.LaneInteractive, llm.TaskChat))
		require.NoError(t, err)
		waits = append(waits, time.Since(start))
		p.Done(-1)
		time.Sleep(20 * time.Millisecond)
	}
	var worst time.Duration
	for _, w := range waits {
		worst = max(worst, w)
	}
	require.Less(t, worst, 500*time.Millisecond, "chat bị chấm làm chậm")
}
