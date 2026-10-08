package exam_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// ---- helper -----------------------------------------------------------------------------------------------------------------

const canary = "CANARY_ZZ9"

// codeEnv là một lượt làm bài code đang mở: một câu CODE (c11 + cpp17), test mẫu `sample1` và một test ẩn mang canary.
type codeEnv struct {
	r      *rig
	svc    *exam.Service
	e      exam.ExamDetail
	sv     uuid.UUID
	tab    uuid.UUID
	qid    uuid.UUID
	v      exam.AttemptStartView
	item   uuid.UUID
	sample uuid.UUID
	hidden uuid.UUID
}

func (r *rig) codeQuestion(languages string) (qid, sample, hidden uuid.UUID) {
	r.t.Helper()
	q := r.code("code " + uuid.NewString()[:6])
	s := r.test(q.ID, "sample1", "1 2", "3", true, 1)
	h := r.test(q.ID, canary+"_ten", canary+"_vao", canary+"_ra", false, 2)
	r.exec(`update code_problems set languages = $2::text[], starter_code = '{"cpp17":"// khởi đầu"}'::jsonb, reference_source = $3 where question_id=$1`, q.ID, languages, "int main(){/*"+canary+"_loigiai*/}")
	r.verify(q.ID)
	_, err := r.svc.Review(r.t.Context(), r.teacher, r.course, q.ID, "APPROVE", r.version(q.ID))
	require.NoError(r.t, err)
	return q.ID, s.ID, h.ID
}

// codeAttempt dựng bài ĐANG MỞ có đúng một câu code rồi cho một sinh viên bắt đầu (đồng hồ thật; Redis không có = không giới hạn).
func (r *rig) codeAttempt(languages string) *codeEnv {
	r.t.Helper()
	qid, s, h := r.codeQuestion(languages)
	e := r.openExam("bài code", false, qid)
	env := &codeEnv{r: r, svc: r.svc, e: e, sv: r.student("ACTIVE"), tab: uuid.New(), qid: qid, sample: s, hidden: h}
	env.v, _ = r.start(e, env.sv, env.tab)
	require.Len(r.t, env.v.Items, 1)
	env.item = env.v.Items[0].ItemID
	return env
}

func (c *codeEnv) draft(lang, src string, base int) (exam.DraftSaved, error) {
	return c.svc.SaveDraft(c.r.t.Context(), c.sv, c.r.course, c.e.ID, c.v.Attempt.ID, c.item, c.tab, exam.DraftIn{Language: lang, Source: src, BaseRev: base})
}

func (c *codeEnv) run(lang, src string) (exam.RunQueued, error) {
	return c.svc.RunCode(c.r.t.Context(), c.sv, c.r.course, c.e.ID, c.v.Attempt.ID, c.item, c.tab, exam.CodeRunIn{Language: lang, Source: src})
}

func (c *codeEnv) submit(lang, src string) (exam.SubmitQueued, error) {
	return c.svc.SubmitCode(c.r.t.Context(), c.sv, c.r.course, c.e.ID, c.v.Attempt.ID, c.item, c.tab, exam.CodeRunIn{Language: lang, Source: src})
}

func (c *codeEnv) subs(kind string) []string {
	c.r.t.Helper()
	rows, err := c.r.pool.Query(c.r.t.Context(), `select source from code_submissions where attempt_id=$1 and kind=$2::submission_kind order by created_at, id`, c.v.Attempt.ID, kind)
	require.NoError(c.r.t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(c.r.t, rows.Scan(&s))
		out = append(out, s)
	}
	return out
}

// done giả lập máy chấm ghi kết quả (kết quả thật do `internal/judge` ghi; ở đây kiểm cách sinh viên ĐỌC kết quả).
func (c *codeEnv) done(id uuid.UUID, verdict string, compileOK bool, log *string, results any) {
	c.r.t.Helper()
	raw, err := json.Marshal(results)
	require.NoError(c.r.t, err)
	c.r.exec(`update code_submissions set status='DONE', verdict=$2::judge_verdict, compile_ok=$3, compile_log=$4, results=$5::jsonb, passed_weight=1, total_weight=3, tests_version=1, judged_at=now(), lease_until=null where id=$1`,
		id, verdict, compileOK, log, string(raw))
}

func (c *codeEnv) results() []map[string]any {
	return []map[string]any{
		{"test_id": c.sample, "position": 1, "is_sample": true, "verdict": "WA", "time_ms": 12, "memory_kb": 900, "weight": 1, "stdout_excerpt": "4"},
		{"test_id": c.hidden, "position": 2, "is_sample": false, "verdict": "AC", "time_ms": 7, "memory_kb": 800, "weight": 2},
	}
}

func apiDetails(t *testing.T, err error) any {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "cần *apierr.Error, có %v", err)
	return ae.Details
}

func apiRetryAfter(t *testing.T, err error) int {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "cần *apierr.Error, có %v", err)
	return ae.RetryAfter
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func redisSvc(t *testing.T, r *rig, cfg exam.CodeConfig) *exam.Service {
	t.Helper()
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	svc := *r.svc
	svc.Redis, svc.Code = rdb, cfg
	return &svc
}

func judgeUp(t *testing.T, svc *exam.Service) {
	t.Helper()
	require.NoError(t, svc.Redis.Set(t.Context(), appredis.Key("judge", "up"), "1", time.Minute).Err())
}

// ---- AC1: đề bài code phía sinh viên -------------------------------------------------------------------------------------------

// TestCodeItemPayloadWhitelist — AC1: mục code chỉ có ngôn ngữ, giới hạn, mã khởi đầu, test MẪU và bản nháp của chính mình.
func TestCodeItemPayloadWhitelist(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	code := c.v.Items[0].Code
	require.NotNil(t, code)
	require.Equal(t, []string{"c11", "cpp17"}, code.Languages)
	require.Equal(t, "// khởi đầu", code.StarterCode["cpp17"])
	require.Equal(t, []exam.SampleView{{Name: "sample1", Input: "1 2", Expected: "3"}}, code.Samples)
	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(jsonOf(t, code)), &keys))
	require.ElementsMatch(t, []string{"languages", "time_limit_ms", "memory_limit_mb", "starter_code", "samples", "language", "drafts"}, mapKeys(keys))
}

func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestHiddenTestsNeverInPayload — AC1: canary trong tên / đầu vào / đầu ra của test ẩn và lời giải mẫu không bao giờ có trong phản hồi mở lượt (lần đầu, làm tiếp, sau tải lại).
func TestHiddenTestsNeverInPayload(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	require.NotContains(t, jsonOf(t, c.v), canary)
	again, err := r.svc.MyAttempt(t.Context(), c.sv, r.course, c.e.ID, c.tab)
	require.NoError(t, err)
	require.NotContains(t, jsonOf(t, again), canary)
	for _, bad := range []string{"weight", "tests_version", "reference", "checker", "hidden"} {
		require.NotContains(t, jsonOf(t, again), bad)
	}
}

// ---- AC4, AC5: bản nháp ----------------------------------------------------------------------------------------------------------

// TestDraftPerLanguage — AC4: mỗi ngôn ngữ một bản nháp riêng, đổi qua lại không mất chữ; tải lại trả ngôn ngữ của bản lưu gần nhất.
func TestDraftPerLanguage(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	clk := clock.NewFake(time.Now().UTC())
	c.svc = r.at(clk)
	_, err := c.draft("cpp17", "// cpp v1", 0)
	require.NoError(t, err)
	clk.Advance(time.Second)
	_, err = c.draft("c11", "/* c v1 */", 0) // ngôn ngữ khác: base_rev của nó là 0
	require.NoError(t, err)
	clk.Advance(time.Second)
	_, err = c.draft("cpp17", "// cpp v2", 1)
	require.NoError(t, err)
	clk.Advance(time.Second)
	_, err = c.draft("c11", "/* c v2 */", 1)
	require.NoError(t, err)
	got, err := c.svc.MyAttempt(t.Context(), c.sv, r.course, c.e.ID, c.tab)
	require.NoError(t, err)
	code := got.(exam.AttemptStartView).Items[0].Code
	require.Equal(t, "c11", *code.Language, "ngôn ngữ của bản lưu gần nhất")
	require.Equal(t, "// cpp v2", code.Drafts["cpp17"].Source)
	require.Equal(t, 2, code.Drafts["cpp17"].Rev)
	require.Equal(t, "/* c v2 */", code.Drafts["c11"].Source)
}

// TestLanguageNotAllowed — AC4: ngôn ngữ ngoài `languages` của bài → 422 LANGUAGE_NOT_ALLOWED ở bản nháp, chạy thử và nộp; không ghi gì.
func TestLanguageNotAllowed(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{cpp17}")
	_, err := c.draft("c11", "x", 0)
	require.Equal(t, []string{"LANGUAGE_NOT_ALLOWED"}, fieldCodes(t, err))
	_, err = c.run("c11", "x")
	require.Equal(t, []string{"LANGUAGE_NOT_ALLOWED"}, fieldCodes(t, err))
	_, err = c.submit("c11", "x")
	require.Equal(t, []string{"LANGUAGE_NOT_ALLOWED"}, fieldCodes(t, err))
	require.Zero(t, r.count(`select count(*) from code_drafts where attempt_id=$1`, c.v.Attempt.ID)+r.count(`select count(*) from code_submissions where attempt_id=$1`, c.v.Attempt.ID))
}

// TestDraftRevIncrements — AC5: `rev` tăng 1 mỗi lần ghi, lần đầu base_rev = 0.
func TestDraftRevIncrements(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	for i := range 3 {
		out, err := c.draft("cpp17", strings.Repeat("a", i+1), i)
		require.NoError(t, err)
		require.Equal(t, i+1, out.Rev)
	}
	var src string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select source from code_drafts where attempt_id=$1`, c.v.Attempt.ID).Scan(&src))
	require.Equal(t, "aaa", src)
}

// TestDraftConflict409 — AC5: base_rev khác rev hiện tại → 409 DRAFT_CONFLICT kèm `current_rev`; bản đã lưu KHÔNG bị ghi đè.
func TestDraftConflict409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	_, err := c.draft("cpp17", "bản A", 0)
	require.NoError(t, err)
	_, err = c.draft("cpp17", "bản B", 0) // base cũ (0) trong khi rev đã là 1
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "DRAFT_CONFLICT", code)
	require.Contains(t, jsonOf(t, apiDetails(t, err)), `"current_rev":1`)
	var src string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select source from code_drafts where attempt_id=$1`, c.v.Attempt.ID).Scan(&src))
	require.Equal(t, "bản A", src)
	_, err = c.draft("cpp17", "bản B", 5) // base tương lai cũng không hợp lệ
	st, _ = apiStatus(t, err)
	require.Equal(t, 409, st)
}

// TestDraftAfterDeadline — AC5 / AC12: lưu nháp tới deadline + grace vẫn nhận (bản lưu giây cuối); quá ngưỡng → 409 ATTEMPT_CLOSED.
func TestDraftAfterDeadline(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	clk := clock.NewFake(time.Now().UTC())
	c.svc = r.at(clk)
	// lượt đã được mở bằng đồng hồ thật ở codeAttempt; dời hạn để dùng đồng hồ giả
	r.exec(`update exam_attempts set deadline_at = $2 where id=$1`, c.v.Attempt.ID, clk.Now().Add(time.Minute))
	_, err := c.draft("cpp17", "gõ dở", 0)
	require.NoError(t, err)
	clk.Advance(time.Minute + 9*time.Second)
	_, err = c.draft("cpp17", "gõ dở xong", 1)
	require.NoError(t, err, "bản lưu cuối trong grace vẫn được nhận")
	clk.Advance(2 * time.Second)
	_, err = c.draft("cpp17", "gõ sau giờ", 2)
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "ATTEMPT_CLOSED", code)
}

// TestDraftTooLarge — AC3 / SRS 4.4.1: nháp ≤ 65.536 byte; vượt → 422.
func TestDraftTooLarge(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	_, err := c.draft("cpp17", strings.Repeat("a", 65536), 0)
	require.NoError(t, err)
	_, err = c.draft("cpp17", strings.Repeat("a", 65537), 1)
	require.Equal(t, []string{"max"}, fieldCodes(t, err))
}

// ---- AC6: chạy thử ---------------------------------------------------------------------------------------------------------------

// TestRunQueuesRunRow — AC6: `Chạy thử` ghi MỘT dòng RUN (QUEUED) + outbox `judge.enqueue` (kind RUN) cùng giao dịch; không ảnh hưởng điểm / bản nộp.
func TestRunQueuesRunRow(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	out, err := c.run("cpp17", "int main(){}")
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from code_submissions where id=$1 and kind='RUN' and status='QUEUED' and not auto`, out.RunID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='judge.enqueue' and payload->>'submission_id'=$1 and payload->>'kind'='RUN'`, out.RunID.String()))
	require.Empty(t, c.subs("SUBMIT"))
	var score *string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select auto_score::text from exam_attempts where id=$1`, c.v.Attempt.ID).Scan(&score))
	require.Nil(t, score)
	_, err = c.run("cpp17", "   \n") // rỗng
	require.Equal(t, []string{"required"}, fieldCodes(t, err))
	_, err = c.run("cpp17", strings.Repeat("a", 65537))
	require.Equal(t, []string{"max"}, fieldCodes(t, err))
}

// TestRunJudgeDown503 — AC6: không có `ep:judge:up` → 503 JUDGE_UNAVAILABLE + retry_after 10…30 s, không ghi gì; có khoá → 202.
func TestRunJudgeDown503(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	c.svc = redisSvc(t, r, exam.CodeConfig{})
	require.NoError(t, c.svc.Redis.Del(t.Context(), appredis.Key("judge", "up")).Err())
	_, err := c.run("cpp17", "int main(){}")
	st, code := apiStatus(t, err)
	require.Equal(t, 503, st)
	require.Equal(t, "JUDGE_UNAVAILABLE", code)
	require.Contains(t, []int{10, 30}, clampRetry(t, err))
	require.Zero(t, r.count(`select count(*) from code_submissions where attempt_id=$1`, c.v.Attempt.ID))
	judgeUp(t, c.svc)
	_, err = c.run("cpp17", "int main(){}")
	require.NoError(t, err)
}

// clampRetry trả 10 nếu retry_after ∈ [10,30] (để so với tập {10,30} không phụ thuộc giá trị ngẫu nhiên), ngược lại trả giá trị gốc.
func clampRetry(t *testing.T, err error) int {
	t.Helper()
	ra := apiRetryAfter(t, err)
	if ra >= 10 && ra <= 30 {
		return 10
	}
	return ra
}

// TestRunRateLimit10Per10Min — AC6: cửa sổ trượt theo SINH VIÊN (không theo bài): vượt hạn mức → 429 RATE_LIMITED + retry_after = giây tới khi phần tử cũ nhất hết hạn; hết cửa sổ thì chạy được lại.
func TestRunRateLimit10Per10Min(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	clk := clock.NewFake(time.Now().UTC())
	c := r.codeAttempt("{c11,cpp17}")
	c.svc = redisSvc(t, r, exam.CodeConfig{RunLimit: 3, RunWindow: 10 * time.Minute})
	c.svc.Clock = clk
	r.exec(`update exam_attempts set deadline_at = $2 where id=$1`, c.v.Attempt.ID, clk.Now().Add(2*time.Hour))
	judgeUp(t, c.svc)
	for range 3 {
		_, err := c.run("cpp17", "int main(){}")
		require.NoError(t, err)
		clk.Advance(30 * time.Second)
	}
	_, err := c.run("cpp17", "int main(){}")
	st, code := apiStatus(t, err)
	require.Equal(t, 429, st)
	require.Equal(t, "RATE_LIMITED", code)
	require.Equal(t, 600-90, apiRetryAfter(t, err), "lần chạy đầu cách 90 s: còn 510 s tới khi nó hết hạn")
	require.Equal(t, 3, r.count(`select count(*) from code_submissions where attempt_id=$1`, c.v.Attempt.ID), "lần bị chặn không ghi dòng nào")
	clk.Advance(510 * time.Second)
	_, err = c.run("cpp17", "int main(){}")
	require.NoError(t, err)
}

// ---- AC8: nộp lời giải ---------------------------------------------------------------------------------------------------------

// TestSubmitCodeEnqueues — AC8: ghi SUBMIT (QUEUED, source_sha256 đúng) + outbox `judge.enqueue` cùng giao dịch.
func TestSubmitCodeEnqueues(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	out, err := c.submit("cpp17", "int main(){return 0;}")
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from code_submissions where id=$1 and kind='SUBMIT' and status='QUEUED' and not auto and source_sha256=encode(sha256(convert_to(source,'UTF8')),'hex')`, out.SubmissionID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='judge.enqueue' and payload->>'submission_id'=$1 and payload->>'kind'='SUBMIT'`, out.SubmissionID.String()))
	_, err = c.submit("cpp17", "")
	require.Equal(t, []string{"required"}, fieldCodes(t, err))
}

// TestSubmitCooldown — AC8: nộp lần hai trước hết giãn cách → 429 RATE_LIMITED + retry_after ≤ giãn cách; sau giãn cách nộp được.
func TestSubmitCooldown(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	c.svc = redisSvc(t, r, exam.CodeConfig{SubmitCooldown: 2 * time.Second})
	_, err := c.submit("cpp17", "int main(){}")
	require.NoError(t, err)
	_, err = c.submit("cpp17", "int main(){return 1;}")
	st, code := apiStatus(t, err)
	require.Equal(t, 429, st)
	require.Equal(t, "RATE_LIMITED", code)
	require.LessOrEqual(t, apiRetryAfter(t, err), 2)
	require.Equal(t, 1, r.count(`select count(*) from code_submissions where attempt_id=$1`, c.v.Attempt.ID))
	time.Sleep(2200 * time.Millisecond)
	_, err = c.submit("cpp17", "int main(){return 1;}")
	require.NoError(t, err)
}

// TestSubmitCap30 — AC8: vượt số lần nộp tối đa của một bài trong một lượt → 409 SUBMISSION_LIMIT_REACHED (cấu hình 3 cho nhanh; mặc định 30).
func TestSubmitCap30(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	c.svc = &exam.Service{Pool: r.pool, Code: exam.CodeConfig{SubmissionCap: 3}}
	for i := range 3 {
		_, err := c.submit("cpp17", strings.Repeat("a", i+1))
		require.NoError(t, err)
	}
	_, err := c.submit("cpp17", "bản thứ tư")
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "SUBMISSION_LIMIT_REACHED", code)
}

// TestLastSubmissionCounts — AC8: bản tính điểm = SUBMIT mới nhất chưa bị thay (SUPERSEDED).
func TestLastSubmissionCounts(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	var ids []uuid.UUID
	for i := range 3 {
		out, err := c.submit("cpp17", strings.Repeat("a", i+1))
		require.NoError(t, err)
		ids = append(ids, out.SubmissionID)
	}
	final := func() []string {
		page, err := r.svc.ListSubmissions(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, c.item, nil, 10)
		require.NoError(t, err)
		var out []string
		for _, s := range page {
			if s.IsFinal {
				out = append(out, s.ID.String())
			}
		}
		return out
	}
	require.Equal(t, []string{ids[2].String()}, final())
	r.exec(`update code_submissions set status='SUPERSEDED' where id=$1`, ids[2])
	require.Equal(t, []string{ids[1].String()}, final(), "bản mới nhất bị thay → bản liền trước là bản tính điểm")
}

// ---- AC9, AC10: sinh viên thấy gì ------------------------------------------------------------------------------------------------

// TestSubmissionStudentViewHidesHidden — AC9: trong giờ thi chỉ thấy biên dịch + test MẪU; không `results` của test ẩn, trọng số, verdict chung, tên test ẩn.
func TestSubmissionStudentViewHidesHidden(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	out, err := c.submit("cpp17", "int main(){}")
	require.NoError(t, err)
	c.done(out.SubmissionID, "WA", true, nil, c.results())
	v, err := r.svc.GetSubmission(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, out.SubmissionID)
	require.NoError(t, err)
	require.Equal(t, "DONE", v.Status)
	require.Equal(t, []exam.SampleResultView{{Name: "sample1", Verdict: "WA", TimeMS: 12, MemoryKB: 900}}, v.Samples, "chỉ test mẫu; không input / expected / got ở bản nộp")
	require.True(t, v.IsFinal)
	body := jsonOf(t, v)
	for _, bad := range []string{canary, "weight", "passed_weight", "total_weight", "tests_version", "hidden"} {
		require.NotContains(t, body, bad)
	}
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(body), &top))
	require.NotContains(t, top, "verdict", "không có verdict chung của bài")
}

// TestSubmissionViewCE — AC9: bản nộp lỗi biên dịch hiện `compile_ok:false` + `compile_log`, không có test nào.
func TestSubmissionViewCE(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	out, err := c.submit("cpp17", "int main(){")
	require.NoError(t, err)
	log := "main.cpp:1:12: error: expected '}'"
	c.done(out.SubmissionID, "CE", false, &log, []any{})
	v, err := r.svc.GetSubmission(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, out.SubmissionID)
	require.NoError(t, err)
	require.False(t, *v.CompileOK)
	require.Equal(t, log, *v.CompileLog)
	require.Empty(t, v.Samples)
}

// TestSubmissionViewIE — AC9: bản nộp hết lần thử (ERROR / IE) → trạng thái ERROR, không verdict, không điểm; mã vẫn xem lại được.
func TestSubmissionViewIE(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	out, err := c.submit("cpp17", "int main(){}")
	require.NoError(t, err)
	r.exec(`update code_submissions set status='ERROR', verdict='IE', fail_count=4, judged_at=now() where id=$1`, out.SubmissionID)
	v, err := r.svc.GetSubmission(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, out.SubmissionID)
	require.NoError(t, err)
	require.Equal(t, "ERROR", v.Status)
	require.Equal(t, "int main(){}", *v.Source)
	require.NotContains(t, jsonOf(t, v), "verdict")
}

// TestRunViewShowsWADetail — AC7: `Chạy thử` sai kết quả trả đầu vào / mong đợi / kết quả của bạn (cắt ≤ 2 KiB); test ẩn không bao giờ.
func TestRunViewShowsWADetail(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	out, err := c.run("cpp17", "int main(){}")
	require.NoError(t, err)
	c.done(out.RunID, "WA", true, nil, c.results())
	v, err := r.svc.GetRun(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, out.RunID)
	require.NoError(t, err)
	require.Equal(t, []exam.SampleResultView{{Name: "sample1", Verdict: "WA", TimeMS: 12, MemoryKB: 900, Input: "1 2", Expected: "3", Got: "4"}}, v.Samples)
	require.NotContains(t, jsonOf(t, v), canary)
	// khối dài bị cắt ở 2 KiB
	r.exec(`update code_testcases set expected = $2 where id=$1`, c.sample, strings.Repeat("x", 5000))
	v, err = r.svc.GetRun(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, out.RunID)
	require.NoError(t, err)
	require.Len(t, v.Samples[0].Expected, 2048)
	// một bản SUBMIT không đọc được bằng đường `runs/{id}` và ngược lại
	sub, err := c.submit("cpp17", "int main(){}")
	require.NoError(t, err)
	_, err = r.svc.GetRun(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, sub.SubmissionID)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
	_, err = r.svc.GetSubmission(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, out.RunID)
	st, _ = apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestSubmissionsListOwnOnly — AC10: chỉ lịch sử của chính lượt này; người khác đọc danh sách / một bản → 404.
func TestSubmissionsListOwnOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a, b := r.codeAttempt("{c11,cpp17}"), r.codeAttempt("{c11,cpp17}")
	sa, err := a.submit("cpp17", "int main(){/*của A*/}")
	require.NoError(t, err)
	_, err = b.submit("cpp17", "int main(){/*của B*/}")
	require.NoError(t, err)
	page, err := r.svc.ListSubmissions(t.Context(), a.sv, r.course, a.e.ID, a.v.Attempt.ID, a.item, nil, 10)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, sa.SubmissionID, page[0].ID)
	require.Nil(t, page[0].Source, "danh sách không kèm mã")
	_, err = r.svc.ListSubmissions(t.Context(), b.sv, r.course, a.e.ID, a.v.Attempt.ID, a.item, nil, 10) // lượt của A
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
	_, err = r.svc.GetSubmission(t.Context(), b.sv, r.course, a.e.ID, a.v.Attempt.ID, sa.SubmissionID)
	st, _ = apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestSubmissionsListCursor — AC10: mới nhất trước, phân trang con trỏ, không trùng không sót.
func TestSubmissionsListCursor(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	var want []string
	for i := range 5 {
		out, err := c.submit("cpp17", strings.Repeat("a", i+1))
		require.NoError(t, err)
		want = append([]string{out.SubmissionID.String()}, want...)
	}
	var got []string
	var cur *exam.Cursor
	for range 5 {
		rows, err := r.svc.ListSubmissions(t.Context(), c.sv, r.course, c.e.ID, c.v.Attempt.ID, c.item, cur, 3) // fetch = limit + 1
		require.NoError(t, err)
		for _, s := range rows[:min(2, len(rows))] {
			got = append(got, s.ID.String())
		}
		if len(rows) <= 2 {
			break
		}
		cur = &exam.Cursor{At: rows[1].CreatedAt, ID: rows[1].ID}
	}
	require.Equal(t, want, got)
}

// ---- AC11, AC12: chốt lượt ---------------------------------------------------------------------------------------------------

func (c *codeEnv) finishManual() {
	c.r.t.Helper()
	_, err := c.svc.SubmitAttempt(c.r.t.Context(), c.sv, c.r.course, c.e.ID, c.v.Attempt.ID, c.tab)
	require.NoError(c.r.t, err)
}

// TestAutoSubmitDraftWhenNone — AC11: chưa nộp lần nào mà có nháp không rỗng → một SUBMIT tự động (auto) từ đúng bản nháp; lượt ở GRADING (đợi máy chấm).
func TestAutoSubmitDraftWhenNone(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	_, err := c.draft("cpp17", "int main(){/*nháp*/}", 0)
	require.NoError(t, err)
	c.finishManual()
	require.Equal(t, 1, r.count(`select count(*) from code_submissions where attempt_id=$1 and kind='SUBMIT' and auto and language='cpp17' and source='int main(){/*nháp*/}' and status='QUEUED'`, c.v.Attempt.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='judge.enqueue' and payload->>'kind'='SUBMIT' and payload->>'submission_id' in (select id::text from code_submissions where attempt_id=$1 and auto)`, c.v.Attempt.ID))
	st, _, score, _ := r.attemptRow(c.v.Attempt.ID)
	require.Equal(t, "GRADING", st)
	require.Nil(t, score, "điểm do US-PE-08 tính khi máy chấm xong")
}

// TestNoAutoSubmitWhenSubmitExists — AC11: đã có SUBMIT thì KHÔNG dùng nháp mới hơn (tránh biến lời giải đúng thành sai chỉ vì gõ dở).
func TestNoAutoSubmitWhenSubmitExists(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	_, err := c.submit("cpp17", "int main(){/*đã nộp*/}")
	require.NoError(t, err)
	_, err = c.draft("cpp17", "int main(){/*gõ dở", 0)
	require.NoError(t, err)
	c.finishManual()
	require.Equal(t, []string{"int main(){/*đã nộp*/}"}, c.subs("SUBMIT"))
}

// TestEmptyDraftNoSubmit — AC11 (góp ý #2 b): bản nháp MỚI NHẤT rỗng → không tạo, không quay sang nháp ngôn ngữ khác.
func TestEmptyDraftNoSubmit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	clk := clock.NewFake(time.Now().UTC())
	c.svc = r.at(clk)
	_, err := c.draft("c11", "int main(){}", 0)
	require.NoError(t, err)
	clk.Advance(time.Second)
	_, err = c.draft("cpp17", "", 0) // mới nhất, rỗng
	require.NoError(t, err)
	c.finishManual()
	require.Empty(t, c.subs("SUBMIT"))
	st, _, _, _ := r.attemptRow(c.v.Attempt.ID)
	require.Equal(t, "GRADING", st, "không có bản nộp không phải lỗi; điểm 0 do PE-08 tính")
}

// TestAutoSubmitUsesLatestDraftOnly — AC11: nhiều ngôn ngữ có nháp → dùng ĐÚNG bản có updated_at lớn nhất.
func TestAutoSubmitUsesLatestDraftOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	clk := clock.NewFake(time.Now().UTC())
	c.svc = r.at(clk)
	_, err := c.draft("cpp17", "// cũ", 0)
	require.NoError(t, err)
	clk.Advance(time.Second)
	_, err = c.draft("c11", "/* mới nhất */", 0)
	require.NoError(t, err)
	c.finishManual()
	require.Equal(t, []string{"/* mới nhất */"}, c.subs("SUBMIT"))
}

// TestAutoSubmitOnTimeoutUsesGraceDraft — AC12: hết giờ khi đang gõ: bản nháp tới trong grace là bản dùng cho SUBMIT tự động khi tick nộp lượt.
func TestAutoSubmitOnTimeoutUsesGraceDraft(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	clk := clock.NewFake(time.Now().UTC())
	c.svc = r.at(clk)
	r.exec(`update exam_attempts set deadline_at = $2 where id=$1`, c.v.Attempt.ID, clk.Now().Add(time.Minute))
	_, err := c.draft("cpp17", "int main(){/*gõ tới giây cuối*/}", 0)
	require.NoError(t, err)
	clk.Advance(time.Minute + 5*time.Second)
	_, err = c.draft("cpp17", "int main(){/*gõ tới giây cuối*/ return 0;}", 1) // trong grace 10 s
	require.NoError(t, err)
	clk.Advance(10 * time.Second)
	n, err := c.svc.AutoSubmitDue(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, []string{"int main(){/*gõ tới giây cuối*/ return 0;}"}, c.subs("SUBMIT"))
	var auto bool
	require.NoError(t, r.pool.QueryRow(t.Context(), `select auto from code_submissions where attempt_id=$1 and kind='SUBMIT'`, c.v.Attempt.ID).Scan(&auto))
	require.True(t, auto)
}

// ---- AC14: không rò --------------------------------------------------------------------------------------------------------------

// TestNoAnswerLeakCodeEndpoints — AC14: canary (test ẩn, lời giải mẫu, mã của sinh viên khác) không xuất hiện trong BẤT KỲ phản hồi hay lỗi nào của đường code; đọc của người khác → 404.
func TestNoAnswerLeakCodeEndpoints(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a := r.codeAttempt("{c11,cpp17}")
	// một sinh viên khác trong CÙNG bài làm xong một lượt mang canary trong mã của họ
	other := r.student("ACTIVE")
	ot := uuid.New()
	ov, _ := r.start(a.e, other, ot)
	oc := &codeEnv{r: r, svc: r.svc, e: a.e, sv: other, tab: ot, v: ov, item: ov.Items[0].ItemID, sample: a.sample, hidden: a.hidden}
	osub, err := oc.submit("cpp17", "int main(){/*"+canary+"_cua_nguoi_khac*/}")
	require.NoError(t, err)
	orun, err := oc.run("cpp17", "int main(){/*"+canary+"_run_nguoi_khac*/}")
	require.NoError(t, err)

	var bodies []string
	add := func(v any, err error) {
		t.Helper()
		if err != nil {
			bodies = append(bodies, err.Error()+jsonOf(t, apiDetails(t, err)))
			return
		}
		bodies = append(bodies, jsonOf(t, v))
	}
	add(r.svc.MyAttempt(t.Context(), a.sv, r.course, a.e.ID, a.tab))
	add(a.draft("cpp17", "int main(){}", 0))
	add(a.draft("cpp17", "x", 0)) // 409 conflict: details không lộ nội dung
	run, err := a.run("cpp17", "int main(){}")
	add(run, err)
	sub, err := a.submit("cpp17", "int main(){}")
	add(sub, err)
	a.done(run.RunID, "WA", true, nil, a.results())
	a.done(sub.SubmissionID, "WA", true, nil, a.results())
	add(r.svc.GetRun(t.Context(), a.sv, r.course, a.e.ID, a.v.Attempt.ID, run.RunID))
	add(r.svc.GetSubmission(t.Context(), a.sv, r.course, a.e.ID, a.v.Attempt.ID, sub.SubmissionID))
	add(r.svc.ListSubmissions(t.Context(), a.sv, r.course, a.e.ID, a.v.Attempt.ID, a.item, nil, 10))
	// đọc của người khác: 404, không dữ liệu
	for _, f := range []func() error{
		func() error {
			_, err := r.svc.GetSubmission(t.Context(), a.sv, r.course, a.e.ID, ov.Attempt.ID, osub.SubmissionID)
			return err
		},
		func() error {
			_, err := r.svc.GetRun(t.Context(), a.sv, r.course, a.e.ID, ov.Attempt.ID, orun.RunID)
			return err
		},
		func() error {
			_, err := r.svc.GetSubmission(t.Context(), a.sv, r.course, a.e.ID, a.v.Attempt.ID, osub.SubmissionID) // đúng lượt của mình, id của người khác
			return err
		},
		func() error {
			_, err := r.svc.ListSubmissions(t.Context(), a.sv, r.course, a.e.ID, ov.Attempt.ID, ov.Items[0].ItemID, nil, 10)
			return err
		},
	} {
		st, _ := apiStatus(t, f())
		require.Equal(t, 404, st)
	}
	// sau khi nộp xong (lượt GRADING) màn kết quả vẫn không lộ
	a.finishManual()
	add(r.svc.MyAttempt(t.Context(), a.sv, r.course, a.e.ID, a.tab))
	add(r.svc.ListSubmissions(t.Context(), a.sv, r.course, a.e.ID, a.v.Attempt.ID, a.item, nil, 10))
	for i, b := range bodies {
		require.NotContains(t, b, canary, "phản hồi #%d", i)
	}
}

// ---- SSE ---------------------------------------------------------------------------------------------------------------------------

// TestCodeNotifierPublishes — SRS 4.4.2: máy chấm xong → SSE `exam.run` / `exam.submission` tới ĐÚNG chủ bản nộp, chỉ id + trạng thái (không mã, không kết quả).
func TestCodeNotifierPublishes(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c := r.codeAttempt("{c11,cpp17}")
	other := r.student("ACTIVE")
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	n := &exam.CodeNotifier{Pool: r.pool, SSE: sse.NewPublisher(rdb, 100, time.Minute)}
	run, err := c.run("cpp17", "int main(){/*bí mật*/}")
	require.NoError(t, err)
	sub, err := c.submit("cpp17", "int main(){/*bí mật*/}")
	require.NoError(t, err)
	c.done(run.RunID, "AC", true, nil, c.results())
	c.done(sub.SubmissionID, "AC", true, nil, c.results())
	for _, id := range []uuid.UUID{run.RunID, sub.SubmissionID} {
		payload, _ := json.Marshal(map[string]any{"submission_id": id, "kind": "X", "status": "DONE"})
		require.NoError(t, n.HandleDone(t.Context(), outbox.Message{Topic: "exam.submission_done", Payload: payload}))
	}
	evs, err := rdb.XRange(t.Context(), sse.BufKey(c.sv.String()), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, evs, 2)
	require.Equal(t, exam.EventRun, evs[0].Values["type"])
	require.Contains(t, evs[0].Values["data"], run.RunID.String())
	require.Equal(t, exam.EventSubmission, evs[1].Values["type"])
	require.Contains(t, evs[1].Values["data"], sub.SubmissionID.String())
	for _, e := range evs {
		require.NotContains(t, e.Values["data"], "bí mật")
	}
	none, err := rdb.XRange(t.Context(), sse.BufKey(other.String()), "-", "+").Result()
	require.NoError(t, err)
	require.Empty(t, none, "người khác không nhận sự kiện")
}
