package grading_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

func unavailable(reason string) func(int) error {
	return func(int) error { return &llm.ErrUnavailable{Reason: reason} }
}

func (r *qrig) attempts(res uuid.UUID) (n int) {
	r.one(&n, `select attempts::int from grading_results where id = $1`, res)
	return n
}

func (r *qrig) onlyResult() (id uuid.UUID) {
	r.one(&id, `select id from grading_results where course_id = $1`, r.course)
	return id
}

// TestRetryThreeThenDead — AC4: lỗi tạm 3 lần rồi FAILED ở DB; `.dead` chỉ để đếm; nhãn lỗi bằng mã.
func TestRetryThreeThenDead(t *testing.T) {
	r := newQRig(t)
	r.sub(uuid.Nil, 1, "MATCHED")
	r.llmErr.Store(unavailable(llm.ReasonAllFailed))
	e := r.gradeAll(false)
	q := r.queue(nil)
	r.start(q)
	r.deliver(*e.JobID)
	r.eventually(func() bool { return r.states()["FAILED"] == 1 }, "hết lượt → FAILED")
	res := r.onlyResult()
	require.Equal(t, 3, r.attempts(res))
	var code string
	r.one(&code, `select error_code from grading_results where id = $1`, res)
	require.Equal(t, grading.ErrLLMUnavailable, code)
	// FAILED ghi ở DB TRƯỚC khi consumer XADD `.dead` và tăng bộ đếm (hai bước tuần tự, không cùng giao dịch): chờ cho tới khi quan sát được, không đọc ngay (BUG-P7-07-5)
	r.eventually(func() bool {
		n, err := r.rdb.XLen(r.ctx, r.prefix+grading.StreamGradingDead).Result()
		return err == nil && n == 1 && q.M.Snapshot().DeadTotal == 1
	}, ".dead có đúng 1 mục và DeadTotal = 1")
	require.Equal(t, 6, int(r.calls.Load()), "ba lần chạy, mỗi lần hai lượt")
}

// TestBackoffUsesNotBefore — AC4: thử lại giãn cách bằng not_before (consumer không ngủ): sau lần lỗi đầu hàng QUEUED, not_before ở tương lai, không có lời gọi LLM mới.
func TestBackoffUsesNotBefore(t *testing.T) {
	r := newQRig(t)
	r.sub(uuid.Nil, 1, "MATCHED")
	r.llmErr.Store(unavailable(llm.ReasonAllFailed))
	e := r.gradeAll(false)
	r.start(r.queue(func(c *grading.QueueConfig) { c.Backoff = []time.Duration{time.Hour} }))
	r.deliver(*e.JobID)
	res := r.onlyResult()
	r.eventually(func() bool { return r.states()["QUEUED"] == 1 && r.attempts(res) == 1 }, "QUEUED lại sau lần lỗi đầu")
	var future bool
	r.one(&future, `select not_before > now() + interval '50 minutes' from grading_results where id = $1`, res)
	require.True(t, future)
	time.Sleep(600 * time.Millisecond) // máy quét chạy vài lần: chưa tới giờ nên không nạp lại
	require.Equal(t, 2, int(r.calls.Load()))
	require.Equal(t, 1, r.attempts(res))
}

// TestPermanentErrorNoRetry — AC4: hết ngân sách → FAILED ngay (attempts = 1), không thử lại.
func TestPermanentErrorNoRetry(t *testing.T) {
	r := newQRig(t)
	r.sub(uuid.Nil, 1, "MATCHED")
	r.llmErr.Store(unavailable(llm.ReasonBudgetExhausted))
	e := r.gradeAll(false)
	r.start(r.queue(nil))
	r.deliver(*e.JobID)
	r.eventually(func() bool { return r.states()["FAILED"] == 1 }, "FAILED")
	res := r.onlyResult()
	var code string
	r.one(&code, `select error_code from grading_results where id = $1`, res)
	require.Equal(t, grading.ErrLLMBudgetExhausted, code)
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, r.attempts(res))
	require.Equal(t, 2, int(r.calls.Load()))
}

// TestDeadLetterVisible — AC4: bài hết lượt hiện ở DB (FAILED + mã) và bộ đếm dead tăng; bài khác trong lô không bị chặn.
func TestDeadLetterVisible(t *testing.T) {
	r := newQRig(t)
	r.sub(uuid.Nil, 1, "MATCHED")
	r.llmErr.Store(unavailable(llm.ReasonAllFailed))
	e := r.gradeAll(false)
	q := r.queue(nil)
	r.start(q)
	r.deliver(*e.JobID)
	r.eventually(func() bool { s, _, _ := r.jobRow(*e.JobID); return s == "SUCCEEDED" }, "lô xong dù bài lỗi")
	_, _, res := r.jobRow(*e.JobID)
	require.JSONEq(t, `{"graded":0,"failed":1,"flagged":0}`, res)
	snap := q.M.Snapshot()
	require.EqualValues(t, 1, snap.DeadTotal)
	require.EqualValues(t, 1, snap.JobsTotal["failed"])
	require.EqualValues(t, 2, snap.JobsTotal["retry"])
}

// TestRetryEndpointRequeues — AC4: retry đưa FAILED về QUEUED (attempts 0, cùng round); không phải FAILED → VERSION_CONFLICT; chấm lại thành công.
func TestRetryEndpointRequeues(t *testing.T) {
	r := newQRig(t)
	sub := r.sub(uuid.Nil, 1, "MATCHED")
	var res uuid.UUID
	r.one(&res, `insert into grading_results (course_id, submission_id, run_state, error_code, attempts) values ($1, $2, 'FAILED', 'LLM_UNAVAILABLE', 3) returning id`, r.course, sub)
	e, err := r.batches.Retry(r.ctx, r.course, res, r.teacher)
	require.NoError(t, err)
	require.Equal(t, 1, e.Queued)
	require.Equal(t, 0, r.attempts(res))
	require.Equal(t, 1, r.count(`select round::int from grading_results where id = $1`, res))
	_, err = r.batches.Retry(r.ctx, r.course, res, r.teacher)
	_, code := apiCode(t, err)
	require.Equal(t, "VERSION_CONFLICT", code)
	r.start(r.queue(nil))
	r.deliver(*e.JobID)
	r.eventually(func() bool { return r.states()["DONE"] == 1 }, "chấm lại xong")
}

// TestOneFailureDoesNotBlockBatch — AC4: một bài lỗi vĩnh viễn không chặn các bài khác; jobs không kẹt.
func TestOneFailureDoesNotBlockBatch(t *testing.T) {
	r := newQRig(t)
	for range 4 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	r.llmErr.Store(func(n int) error {
		if n <= 2 { // bài đầu tiên (hai lượt)
			return &llm.ErrUnavailable{Reason: llm.ReasonBudgetExhausted}
		}
		return nil
	})
	e := r.gradeAll(false)
	r.start(r.queue(func(c *grading.QueueConfig) { c.Workers = 1 }))
	r.deliver(*e.JobID)
	r.eventually(func() bool { s, _, _ := r.jobRow(*e.JobID); return s == "SUCCEEDED" }, "lô xong")
	_, _, res := r.jobRow(*e.JobID)
	require.JSONEq(t, `{"graded":3,"failed":1,"flagged":0}`, res)
}

// TestSupersededSkippedNoLLM — AC7: bản cũ còn QUEUED khi có bản mới → SUPERSEDED, không gọi LLM.
func TestSupersededSkippedNoLLM(t *testing.T) {
	r := newQRig(t)
	first := r.sub(uuid.Nil, 1, "MATCHED")
	e := r.gradeAll(false)
	r.sub(r.student(first), 2, "MATCHED")
	r.start(r.queue(nil))
	r.deliver(*e.JobID)
	r.eventually(func() bool { return r.states()["FAILED"] == 1 }, "bản cũ kết thúc")
	var code string
	r.one(&code, `select error_code from grading_results where submission_id = $1`, first)
	require.Equal(t, grading.ErrSuperseded, code)
	require.Zero(t, r.calls.Load(), "không gọi LLM")
	r.eventually(func() bool { s, _, _ := r.jobRow(*e.JobID); return s == "SUCCEEDED" }, "lô vẫn kết thúc")
}

// TestWorkerConcurrencyBound — AC6: không quá Workers bài RUNNING cùng lúc.
func TestWorkerConcurrencyBound(t *testing.T) {
	r := newQRig(t)
	for range 8 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	r.delay.Store(int64(200 * time.Millisecond))
	e := r.gradeAll(false)
	r.start(r.queue(func(c *grading.QueueConfig) { c.Workers = 2 }))
	r.deliver(*e.JobID)
	peak := 0
	r.eventually(func() bool {
		peak = max(peak, r.states()["RUNNING"])
		return r.states()["DONE"] == 8
	}, "8 bài xong")
	require.LessOrEqual(t, peak, 2)
	require.GreaterOrEqual(t, peak, 2, "hai worker thật sự chạy song song")
}

// TestLeaseRenewedWhileRunning — AC6: bài chậm giữ thuê nhờ nhịp gia hạn; chỉ một lần chạy.
func TestLeaseRenewedWhileRunning(t *testing.T) {
	r := newQRig(t)
	r.sub(uuid.Nil, 1, "MATCHED")
	r.delay.Store(int64(1800 * time.Millisecond))
	e := r.gradeAll(false)
	cfg := func(c *grading.QueueConfig) {
		c.Lease = 500 * time.Millisecond
		c.Heartbeat = 120 * time.Millisecond
		c.Workers = 1
	}
	r.start(r.queue(cfg))
	r.start(r.queue(cfg)) // consumer thứ hai thử nhận lại: thuê còn hạn nên bị từ chối
	r.deliver(*e.JobID)
	res := r.onlyResult()
	var first time.Time
	r.eventually(func() bool {
		return r.states()["RUNNING"] == 1 && r.pool.QueryRow(r.ctx, `select lease_until from grading_results where id = $1`, res).Scan(&first) == nil
	}, "đang chạy")
	var later time.Time
	r.eventually(func() bool {
		return r.pool.QueryRow(r.ctx, `select lease_until from grading_results where id = $1`, res).Scan(&later) == nil && later.After(first)
	}, "lease_until tăng")
	r.eventually(func() bool { return r.states()["DONE"] == 1 }, "xong")
	require.Equal(t, 1, r.attempts(res))
	require.Equal(t, 2, int(r.calls.Load()), "không trả tiền LLM hai lần")
}

// TestGradingNotInOutboxConsumer — AC6: handler outbox của grading.batch chỉ XADD rồi trả ngay (ErrDeferred) — một bài chậm không chiếm consumer outbox.
func TestGradingNotInOutboxConsumer(t *testing.T) {
	r := newQRig(t)
	r.sub(uuid.Nil, 1, "MATCHED")
	r.delay.Store(int64(3 * time.Second))
	e := r.gradeAll(false)
	r.queue(nil) // đăng ký handler, KHÔNG chạy consumer
	start := time.Now()
	r.deliver(*e.JobID)
	require.Less(t, time.Since(start), time.Second)
	n, err := r.rdb.XLen(r.ctx, r.prefix+grading.StreamGrading).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	require.Zero(t, r.calls.Load())
	s, _, _ := r.jobRow(*e.JobID)
	require.Equal(t, "RUNNING", s, "việc chờ consumer ep:grading, không xong trong outbox")
}

// TestBudgetExhaustedBatchSingleNotice — AC8: cả lô hết ngân sách → mỗi bài FAILED nhưng nhân viên nhận MỘT thông báo gộp.
func TestBudgetExhaustedBatchSingleNotice(t *testing.T) {
	r := newQRig(t)
	for range 5 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	r.llmErr.Store(unavailable(llm.ReasonBudgetExhausted))
	e := r.gradeAll(false)
	r.start(r.queue(nil))
	r.deliver(*e.JobID)
	r.eventually(func() bool { s, _, _ := r.jobRow(*e.JobID); return s == "SUCCEEDED" }, "lô xong")
	require.Equal(t, 5, r.states()["FAILED"])
	require.Equal(t, 1, r.count(`select count(*)::int from notifications where user_id = $1 and type = $2`, r.teacher, grading.TypeBudget))
}

// TestRedisOutageRequeuesQueued — AC8: hàng QUEUED ở DB mà tin Redis mất → máy quét nạp lại, không mất bài.
func TestRedisOutageRequeuesQueued(t *testing.T) {
	r := newQRig(t)
	for range 3 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	r.gradeAll(false) // không giao outbox: tin XADD "mất"
	require.Equal(t, 3, r.states()["QUEUED"])
	r.start(r.queue(func(c *grading.QueueConfig) { c.RequeueAfter = 300 * time.Millisecond }))
	r.eventually(func() bool { return r.states()["DONE"] == 3 }, "bài bị mất tin vẫn được chấm")
}

func (r *qrig) router(t *testing.T) http.Handler {
	t.Helper()
	store, err := blob.New(t.Context(), blob.Config{Endpoint: "127.0.0.1:9", Bucket: "x", AccessKey: "x", SecretKey: "x"})
	require.NoError(t, err)
	return httpapi.NewRouter(httpapi.Deps{Cfg: r.cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: r.pool, Redis: r.rdb, Clock: clock.Real{}, State: httpapi.NewState(), Jobs: r.jobs, Blob: store})
}

func (r *qrig) call(h http.Handler, method, path string, uid uuid.UUID, role auth.Role) (int, map[string]any) {
	r.t.Helper()
	tok, err := auth.NewIssuer(rbacSecret, time.Hour, clock.Real{}).Issue(uid.String(), role, "")
	require.NoError(r.t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	if method != http.MethodGet {
		req.Header.Set("Idempotency-Key", uuid.NewString())
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

type actors struct{ ta, student, other, admin uuid.UUID }

func (r *qrig) actors() actors {
	a := actors{ta: r.user("TA", "Trợ Giảng"), student: r.user("STUDENT", "Sinh Viên"), other: r.user("TEACHER", "GV Khác"), admin: r.user("ADMIN", "Quản Trị")}
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN'), ($1, $3, 'STUDENT', 'ACTIVE', 'ADMIN')`, r.course, a.ta, a.student)
	return a
}

// TestGradeAllRBAC — AC9: grade-all / grade / retry: TA, TEACHER qua; STUDENT, ADMIN, giảng viên lớp khác 403.
func TestGradeAllRBAC(t *testing.T) {
	r := newQRig(t)
	h := r.router(t)
	a := r.actors()
	sub := r.sub(uuid.Nil, 1, "MATCHED")
	var failed uuid.UUID
	r.one(&failed, `insert into grading_results (course_id, submission_id, run_state, error_code) values ($1, $2, 'FAILED', 'LLM_UNAVAILABLE') returning id`, r.course, r.sub(uuid.Nil, 1, "MATCHED"))
	c := "/api/v1/courses/" + r.course.String()
	paths := []struct{ name, path string }{
		{"grade-all", c + "/assignments/" + r.asg.String() + "/grade-all"},
		{"grade", c + "/submissions/" + sub.String() + "/grade"},
		{"retry", c + "/grading-results/" + failed.String() + "/retry"},
	}
	for _, p := range paths {
		for _, who := range []struct {
			name string
			id   uuid.UUID
			role auth.Role
			want int
		}{{"STUDENT", a.student, auth.RoleStudent, 403}, {"ADMIN", a.admin, auth.RoleAdmin, 403}, {"GV lớp khác", a.other, auth.RoleTeacher, 403}} {
			code, _ := r.call(h, http.MethodPost, p.path, who.id, who.role)
			require.Equal(t, who.want, code, p.name+" "+who.name)
		}
	}
	require.Zero(t, r.count(`select count(*)::int from jobs where kind = 'grading.batch'`), "ca bị từ chối không tạo việc")
	code, body := r.call(h, http.MethodPost, paths[0].path, a.ta, auth.RoleTA)
	require.Equal(t, 202, code)
	require.NotEmpty(t, body["job_id"])
	code, _ = r.call(h, http.MethodPost, paths[1].path, r.teacher, auth.RoleTeacher)
	require.Equal(t, 409, code, "bài đã xếp hàng ở lô trước: VERSION_CONFLICT, không phải 403")
	code, _ = r.call(h, http.MethodPost, paths[2].path, r.teacher, auth.RoleTeacher)
	require.Equal(t, 202, code)
}

// TestJobReadRBAC — AC9: việc chấm chỉ người tạo và nhân viên cùng lớp đọc được; STUDENT, ADMIN, giảng viên lớp khác không.
func TestJobReadRBAC(t *testing.T) {
	r := newQRig(t)
	h := r.router(t)
	a := r.actors()
	r.sub(uuid.Nil, 1, "MATCHED")
	e := r.gradeAll(false)
	q := r.queue(nil)
	q.Prefix = ""
	r.first.Prefix = "" // JobScope của gateway đọc lô ở khoá không tiền tố
	r.deliver(*e.JobID)
	res := r.onlyResult()
	t.Cleanup(func() {
		r.rdb.Del(context.Background(), "ep:gradebatch:"+e.JobID.String(), "ep:gradejob:"+res.String(), grading.StreamGrading)
	})
	path := "/api/v1/jobs/" + e.JobID.String()
	for _, who := range []struct {
		name string
		id   uuid.UUID
		role auth.Role
		want int
	}{{"người tạo", r.teacher, auth.RoleTeacher, 200}, {"TA cùng lớp", a.ta, auth.RoleTA, 200}, {"STUDENT", a.student, auth.RoleStudent, 404}, {"ADMIN", a.admin, auth.RoleAdmin, 404}, {"GV lớp khác", a.other, auth.RoleTeacher, 404}} {
		code, _ := r.call(h, http.MethodGet, path, who.id, who.role)
		require.Equal(t, who.want, code, who.name)
	}
}

// TestGradingMetricsNoPII — AC10: bộ đếm chỉ có số đếm theo trạng thái / dead / độ sâu / p50-p95; không nhãn tên, MSSV, văn bản bài.
func TestGradingMetricsNoPII(t *testing.T) {
	r := newQRig(t)
	for range 3 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	e := r.gradeAll(false)
	q := r.queue(nil)
	r.start(q)
	r.deliver(*e.JobID)
	r.eventually(func() bool { return r.states()["DONE"] == 3 }, "xong")
	q.Tick(r.ctx)
	snap := q.M.Snapshot()
	require.EqualValues(t, 3, snap.JobsTotal["done"])
	require.Greater(t, snap.P95, 0.0)
	raw, _ := json.Marshal(snap)
	for _, bad := range []string{"SV ", r.text[:30], "@", "student"} {
		require.NotContains(t, string(raw), bad)
	}
	for k := range snap.JobsTotal {
		require.Contains(t, []string{"done", "failed", "retry", "superseded", "lease_lost"}, k)
	}
}

// TestGradingLogsHaveTraceNoPII — AC10: log của hàng đợi không chứa văn bản bài / tên / MSSV (kể cả khi lỗi).
func TestGradingLogsHaveTraceNoPII(t *testing.T) {
	r := newQRig(t)
	var buf bytes.Buffer
	r.sub(uuid.Nil, 1, "MATCHED")
	r.llmErr.Store(unavailable(llm.ReasonAllFailed))
	e := r.gradeAll(false)
	q := r.queue(nil)
	q.Log = slog.New(slog.NewTextHandler(&buf, nil))
	r.grader.Log = q.Log
	r.start(q)
	r.deliver(*e.JobID)
	r.eventually(func() bool { return r.states()["FAILED"] == 1 }, "FAILED")
	for _, bad := range []string{r.text[:40], "SV 1", "Nguyễn"} {
		require.NotContains(t, buf.String(), bad)
	}
	_ = goredis.Nil
	_ = atomic.Int64{}
}

// TestGradeAllIdempotentKey — AC2: cùng Idempotency-Key → cùng job_id, không xếp thêm.
func TestGradeAllIdempotentKey(t *testing.T) {
	r := newQRig(t)
	h := r.router(t)
	for range 3 {
		r.sub(uuid.Nil, 1, "MATCHED")
	}
	tok, err := auth.NewIssuer(rbacSecret, time.Hour, clock.Real{}).Issue(r.teacher.String(), auth.RoleTeacher, "")
	require.NoError(t, err)
	key := uuid.NewString()
	var jobIDs []any
	for range 3 {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+r.course.String()+"/assignments/"+r.asg.String()+"/grade-all", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		require.Equal(t, 202, w.Code, w.Body.String())
		var out map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		jobIDs = append(jobIDs, out["job_id"])
	}
	require.Equal(t, jobIDs[0], jobIDs[1])
	require.Equal(t, jobIDs[0], jobIDs[2])
	require.Equal(t, 1, r.count(`select count(*)::int from jobs where kind = 'grading.batch'`))
	require.Equal(t, 3, r.count(`select count(*)::int from grading_results where course_id = $1`, r.course))
}
