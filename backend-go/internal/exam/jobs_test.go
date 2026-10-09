package exam_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

// fakeSandbox là máy chấm giả: ghi lại test được chấm, trả kết quả theo kịch bản.
type fakeSandbox struct {
	mu       sync.Mutex
	judgeErr error
	verdict  func(t judge.Test) judge.Verdict
	runOut   map[string]string // input → stdout
	runStat  map[string]string // input → status của go-judge (mặc định Accepted)
	gotTests []judge.Test
	deleted  []string
	onJudge  func()
}

func (f *fakeSandbox) Judge(_ context.Context, _ judge.Language, _ string, p judge.Problem, tests []judge.Test) (judge.Result, error) {
	f.mu.Lock()
	f.gotTests = append([]judge.Test(nil), tests...)
	f.mu.Unlock()
	if f.onJudge != nil {
		f.onJudge()
	}
	if f.judgeErr != nil {
		return judge.Result{}, f.judgeErr
	}
	res := judge.Result{Verdict: judge.AC, CompileOK: true, TestsVersion: p.TestsVersion}
	for _, tc := range tests {
		v := judge.AC
		if f.verdict != nil {
			v = f.verdict(tc)
		}
		if v != judge.AC && res.Verdict == judge.AC {
			res.Verdict = v
		}
		res.Results = append(res.Results, judge.TestResult{TestID: tc.ID, Position: tc.Position, IsSample: tc.IsSample, Verdict: v, TimeMS: 3, Weight: tc.Weight})
	}
	return res, nil
}

func (f *fakeSandbox) Compile(context.Context, judge.Language, string) (judge.Compiled, error) {
	if f.judgeErr != nil {
		return judge.Compiled{}, f.judgeErr
	}
	return judge.Compiled{OK: true, FileID: "FID", Verdict: judge.AC}, nil
}

func (f *fakeSandbox) Run(_ context.Context, _ string, input string, _ judge.Limits) (judge.RunResult, error) {
	st := "Accepted"
	if s, ok := f.runStat[input]; ok {
		st = s
	}
	return judge.RunResult{Status: st, Stdout: f.runOut[input]}, nil
}

func (f *fakeSandbox) DeleteFile(_ context.Context, id string) error {
	f.mu.Lock()
	f.deleted = append(f.deleted, id)
	f.mu.Unlock()
	return nil
}

// fakeLLM ghi lại mọi yêu cầu và trả kịch bản.
type fakeLLM struct {
	mu    sync.Mutex
	reqs  []llm.Request
	outs  []json.RawMessage
	errs  []error // lỗi của lần gọi thứ i (nil = dùng outs[i])
	calls int
}

func (f *fakeLLM) Structured(_ context.Context, r llm.Request, _ json.RawMessage) (json.RawMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.calls
	f.calls++
	f.reqs = append(f.reqs, r)
	if i < len(f.errs) && f.errs[i] != nil {
		return nil, f.errs[i]
	}
	if i < len(f.outs) {
		return f.outs[i], nil
	}
	return nil, errors.New("fake: hết kịch bản")
}

func newRunner(t *testing.T, r *rig, w *exam.Worker) *jobs.Runner {
	t.Helper()
	w.Pool, w.Svc, w.Log = r.pool, r.svc, slog.New(slog.NewTextHandler(io.Discard, nil))
	run := jobs.NewRunner(r.pool, nil, clock.Real{}, w.Log)
	w.Register(run)
	return run
}

type jobRow struct {
	Status string
	Result json.RawMessage
	Error  struct{ Code, Message string }
}

// runJob chạy việc `id` qua jobs.Runner thật (lấy tin outbox đã ghi cùng transaction với dòng jobs).
func runJob(t *testing.T, r *rig, run *jobs.Runner, id uuid.UUID) jobRow {
	t.Helper()
	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload from outbox where topic='job.enqueue' and payload->>'job_id'=$1`, id.String()).Scan(&payload))
	require.NoError(t, run.HandleMessage(t.Context(), outbox.Message{ID: uuid.New(), Topic: jobs.TopicEnqueue, Payload: payload}))
	var j jobRow
	var errRaw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select status::text, coalesce(result, 'null'), coalesce(error, '{}') from jobs where id=$1`, id).Scan(&j.Status, &j.Result, &errRaw))
	require.NoError(t, json.Unmarshal(errRaw, &struct {
		Code    *string `json:"code"`
		Message *string `json:"message"`
	}{}))
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(errRaw, &e)
	j.Error.Code, j.Error.Message = e.Code, e.Message
	return j
}

// codeWithTests dựng bài code có test mẫu + ẩn (và lời giải mẫu).
func (r *rig) codeWithTests(title string) (exam.QuestionDetail, exam.Testcase, exam.Testcase) {
	q := r.code(title)
	s := r.test(q.ID, "sample1", "1 2", "3", true, 1)
	h := r.test(q.ID, "hidden1", "5 5", "10", false, 2)
	d, err := r.svc.Get(r.t.Context(), r.course, q.ID)
	require.NoError(r.t, err)
	return d, s, h
}

// ---- AC6: chạy lời giải mẫu ----

func TestVerifyReferenceAllPass(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, _, _ := r.codeWithTests("verify ok")
	sb := &fakeSandbox{}
	run := newRunner(t, r, &exam.Worker{Sandbox: sb})
	id, err := r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
	require.NoError(t, err)
	j := runJob(t, r, run, id)
	require.Equal(t, "SUCCEEDED", j.Status)
	var res exam.VerifyResult
	require.NoError(t, json.Unmarshal(j.Result, &res))
	require.True(t, res.OK)
	require.Len(t, res.PerTest, 2)
	require.Equal(t, "sample1", res.PerTest[0].Name)
	p := r.problem(q.ID)
	require.NotNil(t, p.Verified)
	require.Equal(t, p.TestsVersion, *p.Verified)
	// sau khi kiểm xong thì duyệt được
	_, err = r.review(q.ID, "APPROVE", r.teacher)
	require.NoError(t, err)
}

func TestVerifyReferenceMismatch(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, _, h := r.codeWithTests("verify lệch")
	sb := &fakeSandbox{verdict: func(tc judge.Test) judge.Verdict {
		if tc.ID == h.ID.String() {
			return judge.WA
		}
		return judge.AC
	}}
	run := newRunner(t, r, &exam.Worker{Sandbox: sb})
	id, err := r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
	require.NoError(t, err)
	j := runJob(t, r, run, id)
	require.Equal(t, "SUCCEEDED", j.Status, "lệch là kết quả hợp lệ của việc, không phải lỗi việc")
	var res exam.VerifyResult
	require.NoError(t, json.Unmarshal(j.Result, &res))
	require.False(t, res.OK)
	require.Equal(t, judge.WA, res.PerTest[1].Verdict)
	require.Equal(t, "hidden1", res.PerTest[1].Name)
	require.Nil(t, r.problem(q.ID).Verified, "không ghi cờ khi có test lệch")
}

func TestVerifyReferenceJudgeDown(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, _, _ := r.codeWithTests("verify judge chết")
	for name, sb := range map[string]exam.Sandbox{"judge lỗi hệ thống": &fakeSandbox{judgeErr: fmt.Errorf("%w: connection refused", judge.ErrSandbox)}, "chưa có judge": nil} {
		w := &exam.Worker{}
		if sb != nil {
			w.Sandbox = sb
		}
		run := newRunner(t, r, w)
		id, err := r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
		require.NoError(t, err)
		j := runJob(t, r, run, id)
		require.Equal(t, "FAILED", j.Status, name)
		require.Equal(t, "Chưa chạy được lời giải mẫu, thử lại sau.", j.Error.Message, name)
		require.Nil(t, r.problem(q.ID).Verified, name)
	}
}

// TestVerifyReferenceStaleVersion — job chạy trên bản test cũ KHÔNG được "xác minh" bản test mới (tests_version đổi giữa chừng → không ghi cờ).
func TestVerifyReferenceStaleVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, s, _ := r.codeWithTests("verify cũ")
	sb := &fakeSandbox{onJudge: func() {
		_, err := r.svc.UpdateTest(context.Background(), r.teacher, r.course, q.ID, s.ID, exam.TestcaseIn{Weight: new(9)}, true)
		require.NoError(t, err)
	}}
	run := newRunner(t, r, &exam.Worker{Sandbox: sb})
	id, err := r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", runJob(t, r, run, id).Status)
	require.Nil(t, r.problem(q.ID).Verified)
}

// TestVerifyReferencePreconditions — 422 REFERENCE_REQUIRED khi thiếu lời giải mẫu, CODE_TESTS_MISSING khi chưa có test đã duyệt; câu không CODE → 404.
func TestVerifyReferencePreconditions(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	bare, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "CODE", Title: "trần", Topic: "t", Stem: "s"})
	require.NoError(t, err)
	_, err = r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, bare.ID)
	require.Contains(t, fieldCodes(t, err), "REFERENCE_REQUIRED")
	q := r.code("có tham chiếu")
	_, err = r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
	require.Contains(t, fieldCodes(t, err), "CODE_TESTS_MISSING")
	_, err = r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, r.mcq("mcq").ID)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}

// ---- AC9: gợi ý câu trắc nghiệm ----

func mcqJSON(items ...string) json.RawMessage {
	return json.RawMessage(`{"questions":[` + strings.Join(items, ",") + `]}`)
}

func aiMCQ(title string, correct ...int) string {
	opts := make([]string, 4)
	for i := range opts {
		opts[i] = fmt.Sprintf(`{"body":"Đáp án %d","correct":%t}`, i+1, contains(correct, i))
	}
	return fmt.Sprintf(`{"type":"MCQ_SINGLE","title":%q,"stem":"Nội dung %s","options":[%s],"value":null,"explanation":"Vì vậy","difficulty":"MEDIUM"}`, title, title, strings.Join(opts, ","))
}

func contains(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func (r *rig) suggest(in exam.SuggestIn) uuid.UUID {
	r.t.Helper()
	id, err := r.svc.EnqueueSuggest(r.t.Context(), r.teacher, r.course, in)
	require.NoError(r.t, err)
	return id
}

func TestSuggestMCQ(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ai := &fakeLLM{outs: []json.RawMessage{mcqJSON(aiMCQ("Một", 0), aiMCQ("Hai", 1), aiMCQ("Ba", 2))}}
	run := newRunner(t, r, &exam.Worker{LLM: ai})
	id := r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "Mật mã đối xứng", Difficulty: "MEDIUM", Count: 3})
	j := runJob(t, r, run, id)
	require.Equal(t, "SUCCEEDED", j.Status)
	var res exam.SuggestResult
	require.NoError(t, json.Unmarshal(j.Result, &res))
	require.Len(t, res.Created, 3)
	require.Empty(t, res.Dropped)
	rows, err := r.pool.Query(t.Context(), `select origin::text, review_status::text, created_by, ai_job_id, topic from question_bank where course_id=$1`, r.course)
	require.NoError(t, err)
	defer rows.Close()
	n := 0
	for rows.Next() {
		var origin, status, topic string
		var by uuid.UUID
		var job *uuid.UUID
		require.NoError(t, rows.Scan(&origin, &status, &by, &job, &topic))
		require.Equal(t, "AI_DRAFT", origin)
		require.Equal(t, "PENDING", status, "câu AI vào ở PENDING, không bao giờ tự APPROVED")
		require.Equal(t, r.teacher, by)
		require.Equal(t, id, *job)
		require.Equal(t, "Mật mã đối xứng", topic)
		n++
	}
	require.Equal(t, 3, n)
}

func TestSuggestDropsInvalid(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	tooLong := strings.Repeat("x", 130)
	oneOption := `{"type":"MCQ_SINGLE","title":"một đáp án","stem":"s","options":[{"body":"a","correct":true}],"value":null,"explanation":"","difficulty":"EASY"}`
	twoCorrect := aiMCQ("hai đúng", 0, 1)
	dup := `{"type":"MCQ_SINGLE","title":"trùng","stem":"s","options":[{"body":"a","correct":true},{"body":"A","correct":false}],"value":null,"explanation":"","difficulty":"EASY"}`
	short := `{"type":"SHORT","title":"tự luận","stem":"s","options":[],"value":null,"explanation":"","difficulty":"EASY"}`
	tf := `{"type":"TRUE_FALSE","title":"đúng sai","stem":"1<2","options":[],"value":true,"explanation":"","difficulty":"EASY"}`
	ai := &fakeLLM{outs: []json.RawMessage{mcqJSON(aiMCQ("hợp lệ", 3), oneOption, twoCorrect, dup, short, aiMCQ(tooLong, 0), tf, aiMCQ("thừa", 0))}}
	run := newRunner(t, r, &exam.Worker{LLM: ai})
	j := runJob(t, r, run, r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 7}))
	require.Equal(t, "SUCCEEDED", j.Status)
	var res exam.SuggestResult
	require.NoError(t, json.Unmarshal(j.Result, &res))
	require.Len(t, res.Created, 2, "chỉ câu hợp lệ + câu đúng-sai")
	reasons := map[int]string{}
	for _, d := range res.Dropped {
		reasons[d.Index] = d.Reason
	}
	require.Equal(t, map[int]string{1: "OPTION_COUNT", 2: "SINGLE_MULTIPLE_CORRECT", 3: "DUPLICATE_OPTION", 4: "TYPE_NOT_SUPPORTED", 5: "TITLE_LENGTH", 7: "EXTRA"}, reasons)
}

func TestSuggestNotConfigured(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	want := "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."
	for name, w := range map[string]*exam.Worker{"gateway báo": {LLM: &fakeLLM{errs: []error{llm.ErrNotConfigured}}}, "không có gateway": {}} {
		run := newRunner(t, r, w)
		j := runJob(t, r, run, r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 1}))
		require.Equal(t, "FAILED", j.Status, name)
		require.Equal(t, want, j.Error.Message, name)
	}
}

// TestSuggestPromptHasNoStudentData — AC9: prompt chỉ có chủ đề / độ khó / số câu / văn bản nguồn do giảng viên nhập; canary tên, email, MSSV của sinh viên và tên giảng viên không bao giờ xuất hiện.
func TestSuggestPromptHasNoStudentData(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.exec(`update users set full_name='CANARY-TÊN-GV' where id=$1`, r.teacher)
	var sv uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ('canary-sv@example.test', 'CANARY-TÊN-SV 20231234', 'STUDENT') returning id`).Scan(&sv))
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, r.course, sv)
	ai := &fakeLLM{outs: []json.RawMessage{mcqJSON(aiMCQ("Một", 0))}}
	run := newRunner(t, r, &exam.Worker{LLM: ai})
	runJob(t, r, run, r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "Mật mã", Difficulty: "HARD", Count: 1, SourceText: "Tài liệu: AES dùng khoá 128/192/256 bit."}))
	require.Len(t, ai.reqs, 1)
	var all strings.Builder
	for _, m := range ai.reqs[0].Messages {
		all.WriteString(m.Role + ":" + m.Content + "\n")
	}
	txt := all.String()
	for _, banned := range []string{"CANARY", "canary-sv", "20231234", r.teacher.String(), sv.String(), r.course.String(), "@example.test"} {
		require.NotContains(t, txt, banned)
	}
	require.Contains(t, txt, "Mật mã")
	require.Contains(t, txt, "HARD")
	require.Contains(t, txt, "AES dùng khoá")
}

// TestSuggestUsesBatchLane — AC9: đúng MỘT lời gọi `Structured`, tác vụ QUESTION_GEN ở làn BATCH (không chạm làn INTERACTIVE của chat); quá tải thì thử lại theo việc.
func TestSuggestUsesBatchLane(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ai := &fakeLLM{outs: []json.RawMessage{mcqJSON(aiMCQ("Một", 0))}}
	run := newRunner(t, r, &exam.Worker{LLM: ai})
	runJob(t, r, run, r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 1}))
	require.Equal(t, 1, ai.calls)
	req := ai.reqs[0]
	require.Equal(t, llm.TaskQuestionGen, req.Task)
	lane, err := llm.ResolveLane(req.Task, req.Lane)
	require.NoError(t, err)
	require.Equal(t, llm.LaneBatch, lane)
	require.NotNil(t, req.Params.Temperature)
	require.InDelta(t, 0.4, *req.Params.Temperature, 1e-9)
	// quá tải 2 lần rồi được: 3 lời gọi, việc vẫn thành công
	ai2 := &fakeLLM{errs: []error{&llm.ErrOverloaded{RetryAfter: time.Second}, &llm.ErrOverloaded{RetryAfter: time.Second}}, outs: []json.RawMessage{nil, nil, mcqJSON(aiMCQ("Hai", 1))}}
	run2 := newRunner(t, r, &exam.Worker{LLM: ai2, RetryWait: []time.Duration{time.Millisecond, time.Millisecond}})
	j := runJob(t, r, run2, r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 1}))
	require.Equal(t, "SUCCEEDED", j.Status)
	require.Equal(t, 3, ai2.calls)
	// quá tải mãi → FAILED với câu tiếng Việt
	ai3 := &fakeLLM{errs: []error{&llm.ErrOverloaded{}, &llm.ErrOverloaded{}, &llm.ErrOverloaded{}}}
	run3 := newRunner(t, r, &exam.Worker{LLM: ai3, RetryWait: []time.Duration{time.Millisecond, time.Millisecond}})
	j = runJob(t, r, run3, r.suggest(exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 1}))
	require.Equal(t, "FAILED", j.Status)
	require.Contains(t, j.Error.Message, "bận")
}

// ---- AC10: gợi ý đầu vào cho test ----

func inputsJSON(inputs ...string) json.RawMessage {
	items := make([]string, len(inputs))
	for i, in := range inputs {
		items[i] = fmt.Sprintf(`{"name":"ai%d","input":%q,"note":"ca %d","expected":"AI-TỰ-BỊA"}`, i+1, in, i+1)
	}
	return json.RawMessage(`{"inputs":[` + strings.Join(items, ",") + `]}`)
}

// TestSuggestTestsExpectedFromReference — AC10: AI chỉ đề xuất ĐẦU VÀO; `expected` lấy từ stdout của lời giải mẫu (chuẩn hoá cuối dòng); đầu vào làm lời giải mẫu TLE / RE bị bỏ kèm verdict; test AI `approved=false`.
func TestSuggestTestsExpectedFromReference(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, _, _ := r.codeWithTests("gợi ý test")
	sb := &fakeSandbox{runOut: map[string]string{"2 3": "5  \n\n\n", "10 20": "30"}, runStat: map[string]string{"1000000 1": "Time Limit Exceeded", "0 0": "Nonzero Exit Status"}}
	ai := &fakeLLM{outs: []json.RawMessage{inputsJSON("2 3", "1000000 1", "10 20", "0 0")}}
	run := newRunner(t, r, &exam.Worker{LLM: ai, Sandbox: sb})
	j := runJob(t, r, run, r.suggest(exam.SuggestIn{Kind: "CODE_TESTS", QuestionID: &q.ID, Count: 4}))
	require.Equal(t, "SUCCEEDED", j.Status)
	var res exam.SuggestResult
	require.NoError(t, json.Unmarshal(j.Result, &res))
	require.Len(t, res.Created, 2)
	verdicts := map[string]string{}
	for _, d := range res.Dropped {
		verdicts[d.Name] = d.Verdict
	}
	require.Equal(t, map[string]string{"ai2": "TLE", "ai4": "RE"}, verdicts)
	got := r.tests(q.ID)
	require.Len(t, got, 4)
	ai1 := got[2]
	require.Equal(t, "AI_DRAFT", ai1.Source)
	require.False(t, ai1.Approved)
	require.False(t, ai1.IsSample)
	require.Equal(t, 1, ai1.Weight)
	require.Equal(t, "2 3", ai1.Input)
	require.Equal(t, "5\n", ai1.Expected, "expected = stdout của lời giải mẫu, chuẩn hoá")
	require.NotContains(t, ai1.Expected, "AI-TỰ-BỊA")
	require.Equal(t, []string{"FID"}, sb.deleted, "dọn tệp chạy đã biên dịch")
}

// TestSuggestedTestsNotJudgedUntilApproved — AC10: test AI chưa duyệt KHÔNG được chấm, không tính vào test ẩn, không đổi tests_version; duyệt rồi mới tính (tests_version + 1, gỡ cờ lời giải mẫu).
func TestSuggestedTestsNotJudgedUntilApproved(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, _, _ := r.codeWithTests("chưa duyệt")
	r.verify(q.ID)
	tv := r.problem(q.ID).TestsVersion
	sb := &fakeSandbox{runOut: map[string]string{"7 7": "14\n"}}
	ai := &fakeLLM{outs: []json.RawMessage{inputsJSON("7 7")}}
	run := newRunner(t, r, &exam.Worker{LLM: ai, Sandbox: sb})
	runJob(t, r, run, r.suggest(exam.SuggestIn{Kind: "CODE_TESTS", QuestionID: &q.ID, Count: 1}))
	require.Equal(t, tv, r.problem(q.ID).TestsVersion, "thêm test AI chưa duyệt không đổi phiên bản")
	require.NotNil(t, r.problem(q.ID).Verified)
	d, err := r.svc.Get(t.Context(), r.course, q.ID)
	require.NoError(t, err)
	require.Equal(t, 2, d.Code.Tests.Total, "chỉ đếm test đã duyệt")
	require.Equal(t, 1, d.Code.Tests.Hidden)
	// việc kiểm lời giải mẫu chỉ đưa test đã duyệt vào sandbox
	vid, err := r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", runJob(t, r, run, vid).Status)
	require.Len(t, sb.gotTests, 2)
	for _, tc := range sb.gotTests {
		require.NotEqual(t, "7 7", tc.Input)
	}
	// duyệt → được tính
	var aiID uuid.UUID
	for _, tc := range r.tests(q.ID) {
		if !tc.Approved {
			aiID = tc.ID
		}
	}
	n, err := r.svc.ApproveTests(t.Context(), r.teacher, r.course, q.ID, []uuid.UUID{aiID}, true)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, tv+1, r.problem(q.ID).TestsVersion)
	require.Nil(t, r.problem(q.ID).Verified)
	d, _ = r.svc.Get(t.Context(), r.course, q.ID)
	require.Equal(t, 3, d.Code.Tests.Total)
}

func TestSuggestTestsNeedsReference(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "CODE", Title: "không tham chiếu", Topic: "t", Stem: "s"})
	require.NoError(t, err)
	_, err = r.svc.EnqueueSuggest(t.Context(), r.teacher, r.course, exam.SuggestIn{Kind: "CODE_TESTS", QuestionID: &q.ID, Count: 3})
	require.Contains(t, fieldCodes(t, err), "REFERENCE_REQUIRED")
	_, err = r.svc.EnqueueSuggest(t.Context(), r.teacher, r.course, exam.SuggestIn{Kind: "CODE_TESTS", Count: 3})
	require.Error(t, err)
	_, err = r.svc.EnqueueSuggest(t.Context(), r.teacher, r.course, exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 11})
	require.Contains(t, fieldCodes(t, err), "LIMIT_OUT_OF_RANGE")
	_, err = r.svc.EnqueueSuggest(t.Context(), r.teacher, r.course, exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 1, SourceText: strings.Repeat("a", 6001)})
	require.Contains(t, fieldCodes(t, err), "SOURCE_TOO_LARGE")
}
