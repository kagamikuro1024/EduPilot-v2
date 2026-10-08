//go:build integration

package judge_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// qrig: Postgres + Redis thật, judge giả (httptest), một đề code có 3 test (trọng số 1/1/2), một lượt làm.
type qrig struct {
	t      *testing.T
	pool   *pgxpool.Pool
	rdb    *appredis.Client
	fake   *fake
	prefix string
	course uuid.UUID
	exam   uuid.UUID
	item   uuid.UUID
	prob   uuid.UUID
	att    uuid.UUID
	stud   uuid.UUID
	logs   *bytes.Buffer
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func newQRig(t *testing.T) *qrig {
	t.Helper()
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := &qrig{t: t, pool: pool, rdb: rdb, fake: newFake(t), prefix: testutil.TestPrefix(t) + ":", logs: &bytes.Buffer{}}
	r.fake.out = "3\n"
	exec := func(sql string, args ...any) {
		_, err := pool.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	id := uuid.New()
	sfx := strings.ToLower(id.String()[:8])
	var teacher uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, "t."+sfx+"@example.test").Scan(&teacher))
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, "s."+sfx+"@example.test").Scan(&r.stud))
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	require.NoError(t, pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "J-"+strings.ToUpper(sfx), string(jc), teacher).Scan(&r.course))
	require.NoError(t, pool.QueryRow(ctx, `insert into question_bank (course_id, type, title, topic, stem, created_by) values ($1, 'CODE', 'a+b', 't', 's', $2) returning id`, r.course, teacher).Scan(&r.prob))
	exec(`insert into code_problems (question_id, course_id) values ($1, $2)`, r.prob, r.course)
	for i, tc := range []struct {
		w        int
		in, want string
	}{{1, "1 2", "3"}, {1, "2 2", "4"}, {2, "5 5", "10"}} {
		exec(`insert into code_testcases (course_id, problem_id, position, name, is_sample, weight, input, expected, input_bytes, expected_bytes) values ($1, $2, $3, $4, $5, $6, $7, $8, 3, 2)`,
			r.course, r.prob, i+1, "t"+string(rune('1'+i)), i == 0, tc.w, tc.in, tc.want)
	}
	require.NoError(t, pool.QueryRow(ctx, `insert into exams (course_id, title, created_by) values ($1, 'B', $2) returning id`, r.course, teacher).Scan(&r.exam))
	require.NoError(t, pool.QueryRow(ctx, `insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 1) returning id`, r.course, r.exam, r.prob).Scan(&r.item))
	require.NoError(t, pool.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ($1, $2, $3, now() + interval '1 hour') returning id`, r.course, r.exam, r.stud).Scan(&r.att))
	return r
}

// sub tạo một bản nộp QUEUED (chưa XADD) và trả id.
func (r *qrig) sub(kind string, source string) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256)
		values ($1, $2, $3, $4, $5, $6, $7::submission_kind, 'cpp17', $8, repeat('a', 64)) returning id`, r.course, r.exam, r.att, r.item, r.prob, r.stud, kind, source).Scan(&id))
	return id
}

func (r *qrig) queue(name string) (*judge.Queue, *syncBuf) {
	c, err := judge.NewClient(judge.Config{URL: r.fake.srv.URL, Token: testutil.JudgeToken, Slack: time.Second})
	require.NoError(r.t, err)
	buf := &syncBuf{}
	r.t.Cleanup(func() {
		if r.t.Failed() {
			r.t.Logf("log của consumer %s:\n%s", name, buf.String())
		}
	})
	return &judge.Queue{
		Pool: r.pool, Redis: r.rdb, Judge: c, Log: slog.New(slog.NewJSONHandler(buf, nil)), P: 2, Prefix: r.prefix, Name: name,
		Lease: 600 * time.Millisecond, Renew: 150 * time.Millisecond, Idle: time.Minute, TickEvery: 100 * time.Millisecond, Probe: 100 * time.Millisecond, Poll: 50 * time.Millisecond,
		Backoff: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond},
	}, buf
}

// start chạy queue ở nền; trả hàm dừng.
func (r *qrig) start(q *judge.Queue) context.CancelFunc {
	ctx, cancel := context.WithCancel(r.t.Context())
	done := make(chan struct{})
	go func() { defer close(done); _ = q.Run(ctx) }()
	r.t.Cleanup(func() { cancel(); <-done })
	return cancel
}

func (r *qrig) enqueue(q *judge.Queue, id uuid.UUID, kind string) {
	r.t.Helper()
	require.NoError(r.t, q.HandleEnqueue(r.t.Context(), outbox.Message{ID: uuid.New(), Topic: judge.TopicEnqueue, Payload: []byte(`{"submission_id":"` + id.String() + `","kind":"` + kind + `"}`)}))
}

type subRow struct {
	Status, Verdict         string
	Attempts, FailCount     int
	Passed, Total           *int
	Lease, NextAttempt, Enq *time.Time
	CompileOK               *bool
}

func (r *qrig) row(id uuid.UUID) subRow {
	r.t.Helper()
	var s subRow
	var v *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text, verdict::text, attempts, fail_count, passed_weight, total_weight, lease_until, next_attempt_at, enqueued_at, compile_ok from code_submissions where id=$1`, id).
		Scan(&s.Status, &v, &s.Attempts, &s.FailCount, &s.Passed, &s.Total, &s.Lease, &s.NextAttempt, &s.Enq, &s.CompileOK))
	if v != nil {
		s.Verdict = *v
	}
	return s
}

func (r *qrig) waitStatus(id uuid.UUID, want string, within time.Duration) subRow {
	r.t.Helper()
	var s subRow
	require.Eventually(r.t, func() bool { s = r.row(id); return s.Status == want }, within, 25*time.Millisecond, "bản nộp %s chưa tới %s (đang %s)", id, want, s.Status)
	return s
}

// TestJudgeConsumerIdempotent — tín hiệu trùng (XADD hai lần) chỉ chấm MỘT lần; kết quả đúng (AC, 4/4 trọng số), tests_version ghi lại.
func TestJudgeConsumerIdempotent(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.start(q)
	id := r.sub("SUBMIT", "int main(){}")
	r.fake.runs = []string{okRun("3\n"), okRun("4\n"), okRun("10\n")}
	r.enqueue(q, id, "SUBMIT")
	r.enqueue(q, id, "SUBMIT")
	r.waitStatus(id, "DONE", 10*time.Second)
	time.Sleep(400 * time.Millisecond)
	s := r.row(id)
	require.Equal(t, "AC", s.Verdict)
	require.Equal(t, 1, s.Attempts, "chỉ nhận việc một lần")
	require.Equal(t, 4, *s.Passed)
	require.Equal(t, 4, *s.Total)
	require.Nil(t, s.Lease)
	r.fake.mu.Lock()
	require.Equal(t, 1, r.fake.compiles)
	r.fake.mu.Unlock()
}

// TestRunSamplesOnly — `Chạy thử` chỉ chạy test MẪU (SRS 4.4.2): đề có 3 test (1 mẫu, 2 ẩn) → RUN chạy đúng 1 test và `results` không có test ẩn; SUBMIT cùng đề chạy đủ 3.
func TestRunSamplesOnly(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.start(q)
	run := r.sub("RUN", "int main(){}")
	r.fake.runs = []string{okRun("3\n")}
	r.enqueue(q, run, "RUN")
	r.waitStatus(run, "DONE", 10*time.Second)
	r.fake.mu.Lock()
	ran := r.fake.runIdx
	r.fake.mu.Unlock()
	require.Equal(t, 1, ran, "chỉ test mẫu được chạy")
	var n, hidden int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select jsonb_array_length(results), (select count(*) from jsonb_array_elements(results) e where not (e->>'is_sample')::boolean) from code_submissions where id=$1`, run).Scan(&n, &hidden))
	require.Equal(t, 1, n)
	require.Zero(t, hidden)
	require.Equal(t, "AC", r.row(run).Verdict)
	// cùng đề, bản SUBMIT chạy đủ ba test
	sub := r.sub("SUBMIT", "int main(){}")
	r.fake.mu.Lock()
	r.fake.runs, r.fake.runIdx = []string{okRun("3\n"), okRun("4\n"), okRun("10\n")}, 0
	r.fake.mu.Unlock()
	r.enqueue(q, sub, "SUBMIT")
	r.waitStatus(sub, "DONE", 10*time.Second)
	require.NoError(t, r.pool.QueryRow(t.Context(), `select jsonb_array_length(results) from code_submissions where id=$1`, sub).Scan(&n))
	require.Equal(t, 3, n)
}

// TestRunZeroWeightSample — test mẫu trọng số 0 vẫn chạy thử được (không bị coi là "tổng trọng số bằng 0").
func TestRunZeroWeightSample(t *testing.T) {
	r := newQRig(t)
	_, err := r.pool.Exec(t.Context(), `update code_testcases set weight = 0 where problem_id=$1 and is_sample`, r.prob)
	require.NoError(t, err)
	q, _ := r.queue("c1")
	r.start(q)
	run := r.sub("RUN", "int main(){}")
	r.fake.runs = []string{okRun("3\n")}
	r.enqueue(q, run, "RUN")
	require.Equal(t, "AC", r.waitStatus(run, "DONE", 10*time.Second).Verdict)
}

// TestJudgeLeaseNoDoubleRun — hai consumer cùng nhận một tín hiệu: đúng một bên chạy sandbox.
func TestJudgeLeaseNoDoubleRun(t *testing.T) {
	r := newQRig(t)
	q1, _ := r.queue("c1")
	q2, _ := r.queue("c2")
	r.fake.runDelay = 200 * time.Millisecond
	r.start(q1)
	r.start(q2)
	id := r.sub("SUBMIT", "x")
	for i := 0; i < 3; i++ {
		r.enqueue(q1, id, "SUBMIT") // nhiều tín hiệu → cả hai consumer đều đọc được
	}
	r.waitStatus(id, "DONE", 10*time.Second)
	time.Sleep(500 * time.Millisecond)
	r.fake.mu.Lock()
	defer r.fake.mu.Unlock()
	require.Equal(t, 1, r.fake.compiles, "không chạy sandbox hai lần")
	require.Equal(t, 1, r.row(id).Attempts)
}

// TestJudgeLongRunNotReclaimed — chạy lâu hơn thuê nhưng còn gia hạn: không bị nhận lại, fail_count = 0 (chạy lâu không phải lỗi).
func TestJudgeLongRunNotReclaimed(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.fake.runDelay = 900 * time.Millisecond // 3 test ≈ 2,7 s ≫ thuê 600 ms
	r.start(q)
	id := r.sub("SUBMIT", "x")
	r.enqueue(q, id, "SUBMIT")
	r.waitStatus(id, "DONE", 15*time.Second)
	s := r.row(id)
	require.Equal(t, 0, s.FailCount)
	require.Equal(t, 1, s.Attempts)
	r.fake.mu.Lock()
	defer r.fake.mu.Unlock()
	require.Equal(t, 1, r.fake.compiles)
}

// TestJudgeLeaseExpiryRequeue — worker chết giữa chừng (RUNNING, thuê hết hạn): tick đưa về QUEUED (KHÔNG tăng fail_count) và consumer chấm xong.
func TestJudgeLeaseExpiryRequeue(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	id := r.sub("SUBMIT", "x")
	_, err := r.pool.Exec(t.Context(), `update code_submissions set status='RUNNING', lease_until = now() - interval '1 minute', attempts = 1, enqueued_at = now() where id=$1`, id)
	require.NoError(t, err)
	require.NoError(t, r.rdb.XGroupCreateMkStream(t.Context(), r.prefix+judge.StreamSubmit, judge.Group, "0").Err())
	q.Tick(t.Context())
	s := r.row(id)
	require.Equal(t, "QUEUED", s.Status)
	require.Equal(t, 0, s.FailCount, "thuê hết hạn không phải lỗi của bản nộp")
	require.NotNil(t, s.Enq, "tick đã XADD lại")
	r.start(q)
	s = r.waitStatus(id, "DONE", 10*time.Second)
	require.Equal(t, 2, s.Attempts)
}

// TestJudgeBackoffNextAttemptAt — judge 5xx: QUEUED, fail_count + 1, next_attempt_at lùi theo backoff, enqueued_at xoá (tick đưa lại khi đến hạn).
func TestJudgeBackoffNextAttemptAt(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.fake.status5 = true
	r.start(q)
	id := r.sub("SUBMIT", "x")
	r.enqueue(q, id, "SUBMIT")
	require.Eventually(t, func() bool { return r.row(id).FailCount >= 1 }, 10*time.Second, 25*time.Millisecond)
	// ngay sau lần lỗi đầu: QUEUED hoặc đã được thử lại
	s := r.row(id)
	require.Contains(t, []string{"QUEUED", "RUNNING", "ERROR"}, s.Status)
	require.NotNil(t, s.NextAttempt)
	require.Nil(t, s.Passed, "lỗi hệ thống không ghi điểm")
}

// TestJudgeRetryThenDead — lỗi hệ thống 4 lần ⇒ ERROR + IE + XADD judge.submit.dead; KHÔNG điểm 0 (không có kết quả test).
func TestJudgeRetryThenDead(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.fake.status5 = true
	r.start(q)
	id := r.sub("SUBMIT", "x")
	r.enqueue(q, id, "SUBMIT")
	s := r.waitStatus(id, "ERROR", 20*time.Second)
	require.Equal(t, "IE", s.Verdict)
	require.Equal(t, 4, s.FailCount)
	require.Nil(t, s.Passed, "IE không bao giờ là điểm 0")
	n, err := r.rdb.XLen(t.Context(), r.prefix+judge.StreamSubmitDead).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	// judge sống lại + chấm lại (đặt QUEUED như regrade 4.8.4) → xong bình thường
	r.fake.mu.Lock()
	r.fake.status5 = false
	r.fake.mu.Unlock()
	_, err = r.pool.Exec(t.Context(), `update code_submissions set status='QUEUED', fail_count=0, next_attempt_at=now(), enqueued_at=null, verdict=null where id=$1`, id)
	require.NoError(t, err)
	r.waitStatus(id, "DONE", 15*time.Second)
}

// TestJudgeKillWorkerMidRun — consumer A bị "giết" giữa lúc chạy (thuê hết hạn), consumer B chấm xong; kết quả muộn của A bị bỏ (rào chắn thuê).
func TestJudgeKillWorkerMidRun(t *testing.T) {
	r := newQRig(t)
	qa, _ := r.queue("a")
	r.fake.runDelay = 700 * time.Millisecond
	stopA := r.start(qa)
	id := r.sub("SUBMIT", "x")
	r.enqueue(qa, id, "SUBMIT")
	r.waitStatus(id, "RUNNING", 10*time.Second)
	stopA() // A dừng gia hạn
	_, err := r.pool.Exec(t.Context(), `update code_submissions set lease_until = now() - interval '1 second' where id=$1`, id)
	require.NoError(t, err)
	qb, _ := r.queue("b")
	r.start(qb)
	s := r.waitStatus(id, "DONE", 20*time.Second)
	require.NotNil(t, s.Passed, "B ghi kết quả")
	require.Equal(t, 2, s.Attempts, "A nhận lần 1, B nhận lại lần 2")
	time.Sleep(time.Second)
	require.Equal(t, "DONE", r.row(id).Status)
	require.Equal(t, 0, r.row(id).FailCount)
}

// TestSupersedeQueued — nộp lần 2 khi lần 1 còn QUEUED: lần 1 → SUPERSEDED, không chạy sandbox; lần 2 chấm đủ; `Chạy thử` không bị thay.
func TestSupersedeQueued(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	first := r.sub("SUBMIT", "v1")
	run := r.sub("RUN", "run")
	second := r.sub("SUBMIT", "v2")
	r.enqueue(q, first, "SUBMIT")
	r.enqueue(q, run, "RUN")
	r.enqueue(q, second, "SUBMIT")
	r.start(q)
	r.waitStatus(first, "SUPERSEDED", 10*time.Second)
	r.waitStatus(second, "DONE", 10*time.Second)
	r.waitStatus(run, "DONE", 10*time.Second)
	require.Equal(t, 0, r.row(first).Attempts)
	r.fake.mu.Lock()
	defer r.fake.mu.Unlock()
	require.Equal(t, 2, r.fake.compiles, "chỉ bản nộp cuối và bản chạy thử chạy sandbox")
}

// TestRunningNotCancelled — bản 1 đang RUNNING khi bản 2 vào: chạy nốt, KHÔNG bị thay.
func TestRunningNotCancelled(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.fake.runDelay = 400 * time.Millisecond
	r.start(q)
	first := r.sub("SUBMIT", "v1")
	r.enqueue(q, first, "SUBMIT")
	r.waitStatus(first, "RUNNING", 10*time.Second)
	second := r.sub("SUBMIT", "v2")
	r.enqueue(q, second, "SUBMIT")
	r.waitStatus(first, "DONE", 15*time.Second)
	r.waitStatus(second, "DONE", 15*time.Second)
}

// TestJudgeRunLaneReserved — P = 2, 6 bản SUBMIT chậm xếp hàng: một `Chạy thử` mới vào bắt đầu chạy sớm (chỗ dành cho RUN),
// không phải chờ hết hàng SUBMIT.
func TestJudgeRunLaneReserved(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	r.fake.runDelay = 500 * time.Millisecond // mỗi bản ≈ 1,5 s (3 test)
	r.start(q)
	subs := make([]uuid.UUID, 6)
	for i := range subs {
		var id uuid.UUID
		require.NoError(t, r.pool.QueryRow(t.Context(), `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256)
			values ($1, $2, $3, $4, $5, $6, 'SUBMIT', 'cpp17', 'x', repeat('a', 64)) returning id`, r.course, r.exam, r.att, r.item, r.prob, r.stud).Scan(&id))
		subs[i] = id
	}
	_ = subs
	// SUPERSEDED sẽ gạt các bản cũ cùng (lượt, mục): dùng lượt / mục khác cho từng bản để chúng thật sự xếp hàng.
	for i, id := range subs {
		var att uuid.UUID
		var stud uuid.UUID
		require.NoError(t, r.pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, "lane"+string(rune('a'+i))+"."+strings.ToLower(uuid.NewString()[:8])+"@example.test").Scan(&stud))
		require.NoError(t, r.pool.QueryRow(t.Context(), `insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ($1, $2, $3, now() + interval '1 hour') returning id`, r.course, r.exam, stud).Scan(&att))
		_, err := r.pool.Exec(t.Context(), `update code_submissions set attempt_id=$2, student_id=$3 where id=$1`, id, att, stud)
		require.NoError(t, err)
		r.enqueue(q, id, "SUBMIT")
	}
	time.Sleep(700 * time.Millisecond) // đã có bản đang chạy
	run := r.sub("RUN", "run")
	start := time.Now()
	r.enqueue(q, run, "RUN")
	r.waitStatus(run, "DONE", 15*time.Second)
	took := time.Since(start)
	pending := 0
	for _, id := range subs {
		if r.row(id).Status != "DONE" {
			pending++
		}
	}
	require.Less(t, took, 4*time.Second, "Chạy thử không chờ hết hàng SUBMIT (6 bản ≈ 4,5 s)")
	require.GreaterOrEqual(t, pending, 2, "lúc Chạy thử xong hàng SUBMIT vẫn còn việc")
}

// TestJudgeUpKeyLifecycle — worker đặt `ep:judge:up` (TTL ≤ 30 s) khi thăm dò thành công và XOÁ khi judge không phản hồi.
func TestJudgeUpKeyLifecycle(t *testing.T) {
	r := newQRig(t)
	q, _ := r.queue("c1")
	key := appredis.Key("judge", "up")
	_ = r.rdb.Del(t.Context(), key).Err()
	r.start(q)
	require.Eventually(t, func() bool { v, _ := r.rdb.Get(t.Context(), key).Result(); return v == "1" }, 5*time.Second, 25*time.Millisecond)
	ttl, err := r.rdb.TTL(t.Context(), key).Result()
	require.NoError(t, err)
	require.LessOrEqual(t, ttl, 30*time.Second)
	r.fake.mu.Lock()
	r.fake.down = true
	r.fake.mu.Unlock()
	require.Eventually(t, func() bool { n, _ := r.rdb.Exists(t.Context(), key).Result(); return n == 0 }, 5*time.Second, 25*time.Millisecond)
	require.False(t, q.Up())
	r.fake.mu.Lock()
	r.fake.down = false
	r.fake.mu.Unlock()
	require.Eventually(t, func() bool { return q.Up() }, 5*time.Second, 25*time.Millisecond)
}

// TestNoSourceInLogs — canary trong mã nguồn / input / expected không xuất hiện ở log của consumer (kể cả đường lỗi).
func TestNoSourceInLogs(t *testing.T) {
	r := newQRig(t)
	q, buf := r.queue("c1")
	_, err := r.pool.Exec(t.Context(), `update code_testcases set input='CANARY-IN', expected='CANARY-OUT' where problem_id=$1 and position=1`, r.prob)
	require.NoError(t, err)
	r.start(q)
	ok := r.sub("SUBMIT", "CANARY-SRC int main(){}")
	r.enqueue(q, ok, "SUBMIT")
	r.waitStatus(ok, "DONE", 10*time.Second)
	r.fake.mu.Lock()
	r.fake.status5 = true
	r.fake.mu.Unlock()
	var bad uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256)
		values ($1, $2, $3, $4, $5, $6, 'RUN', 'cpp17', 'CANARY-SRC-2', repeat('a', 64)) returning id`, r.course, r.exam, r.att, r.item, r.prob, r.stud).Scan(&bad))
	r.enqueue(q, bad, "RUN")
	r.waitStatus(bad, "ERROR", 20*time.Second)
	out := buf.String()
	require.NotEmpty(t, out)
	require.NotContains(t, out, "CANARY")
	require.NotContains(t, out, testutil.JudgeToken)
}

// TestJudgeProbe — judge chết > 60 s (đồng hồ giả): log `warn` ĐÚNG MỘT lần dù thăm dò nhiều lần; sống lại thì xoá cờ và đặt khoá `ep:judge:up`.
func TestJudgeProbe(t *testing.T) {
	r := newQRig(t)
	q, buf := r.queue("c1")
	clk := clock.NewFake(time.Now())
	q.Clock = clk
	key := appredis.Key("judge", "up")
	ctx := t.Context()
	q.ProbeNow(ctx)
	require.True(t, q.Up())
	require.Equal(t, "1", r.rdb.Get(ctx, key).Val())

	r.fake.mu.Lock()
	r.fake.down = true
	r.fake.mu.Unlock()
	q.ProbeNow(ctx)
	require.False(t, q.Up())
	require.EqualValues(t, 0, r.rdb.Exists(ctx, key).Val(), "thăm dò lỗi xoá khoá ngay")
	clk.Advance(59 * time.Second)
	q.ProbeNow(ctx)
	require.NotContains(t, buf.String(), "quá 60 giây")
	clk.Advance(2 * time.Second)
	q.ProbeNow(ctx)
	q.ProbeNow(ctx)
	require.Equal(t, 1, strings.Count(buf.String(), "quá 60 giây"), "warn đúng một lần")

	r.fake.mu.Lock()
	r.fake.down = false
	r.fake.mu.Unlock()
	q.ProbeNow(ctx)
	require.True(t, q.Up())
	require.Equal(t, "1", r.rdb.Get(ctx, key).Val())
	clk.Advance(2 * time.Minute)
	r.fake.mu.Lock()
	r.fake.down = true
	r.fake.mu.Unlock()
	q.ProbeNow(ctx)
	clk.Advance(61 * time.Second)
	q.ProbeNow(ctx)
	require.Equal(t, 2, strings.Count(buf.String(), "quá 60 giây"), "đợt sập mới cảnh báo lại")
}
