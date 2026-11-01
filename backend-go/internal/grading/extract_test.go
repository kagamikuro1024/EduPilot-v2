package grading_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
)

func lop1(t *testing.T) json.RawMessage { return recorded(t, "extraction-lop1.json", nil) }

func asUserErr(t *testing.T, err error) *jobs.UserError {
	t.Helper()
	var ue *jobs.UserError
	require.ErrorAs(t, err, &ue)
	return ue
}

// TestExtractOneStructuredCall — AC1: mỗi lần chạy đúng MỘT llm.Structured; task UTILITY, temperature 0; văn bản theo trang `[Trang N]`.
func TestExtractOneStructuredCall(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.run(doc, false, 0)
	require.NoError(t, err)
	require.Equal(t, 1, e.fake.count())
	r := e.fake.reqs[0]
	require.Equal(t, llm.TaskUtility, r.Task)
	require.NotNil(t, r.Params.Temperature)
	require.Zero(t, *r.Params.Temperature)
	id := llm.IdentityFrom(e.fake.ctxs[0])
	require.NotNil(t, id.CourseID)
	require.Equal(t, e.course, *id.CourseID)
	require.Equal(t, e.teacher, *id.UserID)
}

// TestExtractTaskAndLane — AC1: tác vụ UTILITY → làn mặc định NEAR_REALTIME (Lane = nil: không tự đổi làn).
func TestExtractTaskAndLane(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	_, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	r := e.fake.reqs[0]
	require.Nil(t, r.Lane)
	require.Equal(t, llm.LaneNearRealtime, llm.DefaultLane(r.Task))
}

// TestExtractPromptHasPages — AC1: mỗi trang mở đầu bằng `[Trang N]`, theo thứ tự trang; có đủ nội dung điều 1…5.
func TestExtractPromptHasPages(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	_, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	user := e.fake.reqs[0].Messages[1].Content
	p1, p2 := strings.Index(user, "[Trang 1]"), strings.Index(user, "[Trang 2]")
	require.True(t, p1 >= 0 && p2 > p1)
	require.Contains(t, user[p1:p2], "Điều 1. Điểm học phần")
	require.Contains(t, user[p2:], "Điều 3. Làm tròn và điểm chữ")
}

// TestExtractSchemaStrictCompatible — TLR-4: mọi object có required = mọi khoá của properties và additionalProperties:false; số là chuỗi; params là anyOf sáu nhánh.
func TestExtractSchemaStrictCompatible(t *testing.T) {
	var root any
	require.NoError(t, json.Unmarshal(grading.ExtractSchema, &root))
	objects, branches := 0, 0
	var walk func(n any, path string)
	walk = func(n any, path string) {
		switch v := n.(type) {
		case map[string]any:
			if props, ok := v["properties"].(map[string]any); ok {
				objects++
				req, _ := v["required"].([]any)
				got := map[string]bool{}
				for _, r := range req {
					got[r.(string)] = true
				}
				require.Len(t, got, len(props), "%s: required phải liệt kê đủ properties", path)
				for k := range props {
					require.True(t, got[k], "%s.%s phải required", path, k)
				}
				require.Equal(t, false, v["additionalProperties"], "%s phải additionalProperties:false", path)
			}
			if any0, ok := v["anyOf"].([]any); ok && path == "$.properties.rules.items.properties.params" {
				branches = len(any0)
			}
			require.NotEqual(t, "number", v["type"], "%s: số điểm phải là chuỗi decimal", path)
			for k, c := range v {
				walk(c, path+"."+k)
			}
		case []any:
			for i, c := range v {
				walk(c, path)
				_ = i
			}
		}
	}
	walk(root, "$")
	require.Equal(t, 6, branches, "params là anyOf sáu nhánh")
	require.Greater(t, objects, 8)
	// bản ghi lop1 hợp schema (gateway sẽ kiểm bằng đúng hàm này)
	require.NoError(t, llm.ValidateJSON(grading.ExtractSchema, lop1(t)))
	require.NoError(t, llm.ValidateJSON(grading.ExtractSchema, recorded(t, "extraction-lop2.json", nil)))
}

// TestExtractAttemptTimeout — TLR-18: mỗi lần thử có ctx con ≤ AttemptTimeout; cả việc ≤ Timeout.
func TestExtractAttemptTimeout(t *testing.T) {
	var deadlines []time.Duration
	f := &fakeLLM{fn: func(_ int, ctx context.Context) (json.RawMessage, error) {
		d, ok := ctx.Deadline()
		require.True(t, ok)
		deadlines = append(deadlines, time.Until(d))
		return lop1(t), nil
	}}
	e := newEnv(t, f)
	e.svc.Cfg.AttemptTimeout, e.svc.Cfg.Timeout = 45*time.Second, 90*time.Second
	_, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	require.Len(t, deadlines, 1)
	require.LessOrEqual(t, deadlines[0], 45*time.Second)
	require.Greater(t, deadlines[0], 40*time.Second)
	// lần thử treo quá hạn của riêng nó → thử lại; cả việc không vượt Timeout
	slow := &fakeLLM{fn: func(n int, ctx context.Context) (json.RawMessage, error) {
		if n == 1 {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return lop1(t), nil
	}}
	e2 := newEnv(t, slow)
	e2.svc.Cfg.AttemptTimeout = 100 * time.Millisecond
	_, err = e2.run(e2.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	require.Equal(t, 2, slow.count())
}

// TestExtractGradeScheme — AC2: quy chế seed lớp 1 → DRAFT đúng 3 thành phần, tổng 100, đủ luật, trích dẫn đúng trang; AI không xác nhận.
func TestExtractGradeScheme(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	res, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	r := res.(grading.Result)
	require.Equal(t, 1, r.SchemeNo)
	require.Empty(t, r.UnclearCode)

	var fx struct {
		Components []struct {
			Key, Name, Group, Weight string
			CitationIDs              []string `json:"citation_ids"`
		}
		Rules []struct {
			Kind        string
			Applicable  bool
			Params      json.RawMessage
			CitationIDs []string `json:"citation_ids"`
		}
		Citations []struct {
			ID    string
			Page  int
			Quote string
		}
	}
	require.NoError(t, json.Unmarshal(lop1(t), &fx))
	cit := map[string]struct {
		page  int
		quote string
	}{}
	for _, c := range fx.Citations {
		cit[c.ID] = struct {
			page  int
			quote string
		}{c.Page, c.Quote}
	}

	var status string
	var confirmedBy *uuid.UUID
	e.one(&status, `select status::text from grade_schemes where course_id = $1`, e.course)
	e.one(&confirmedBy, `select confirmed_by from grade_schemes where course_id = $1`, e.course)
	require.Equal(t, "DRAFT", status)
	require.Nil(t, confirmedBy, "AI chỉ nháp")

	rows, err := e.pool.Query(e.ctx, `select key, name, grp::text, weight::text, citations::text from grade_components where course_id = $1 order by position`, e.course)
	require.NoError(t, err)
	defer rows.Close()
	i := 0
	total := 0
	for rows.Next() {
		var key, name, grp, w, cites string
		require.NoError(t, rows.Scan(&key, &name, &grp, &w, &cites))
		want := fx.Components[i]
		require.Equal(t, want.Key, key)
		require.Equal(t, want.Name, name)
		require.Equal(t, want.Group, grp)
		require.Equal(t, want.Weight+".00", w)
		var got []struct {
			Page  int    `json:"page"`
			Quote string `json:"quote"`
		}
		require.NoError(t, json.Unmarshal([]byte(cites), &got))
		require.Len(t, got, len(want.CitationIDs))
		for j, id := range want.CitationIDs {
			require.Equal(t, cit[id].quote, got[j].Quote)
			require.Equal(t, cit[id].page, got[j].Page)
		}
		total += atoi(want.Weight)
		i++
	}
	require.Equal(t, 3, i)
	require.Equal(t, 100, total)

	var nRules int
	e.one(&nRules, `select count(*) from grade_rules where course_id = $1`, e.course)
	require.Equal(t, len(fx.Rules), nRules)
	for _, rule := range fx.Rules {
		var params json.RawMessage
		var quote *string
		var page *int
		var applicable bool
		err := e.pool.QueryRow(e.ctx, `select params, source_quote, source_page, applicable from grade_rules where course_id = $1 and kind = $2::grade_rule_kind and params @> $3::jsonb limit 1`, e.course, rule.Kind, string(rule.Params)).Scan(&params, &quote, &page, &applicable)
		require.NoError(t, err, rule.Kind)
		require.Equal(t, rule.Applicable, applicable)
		first := cit[rule.CitationIDs[0]]
		require.NotNil(t, quote)
		require.Equal(t, first.quote, *quote)
		require.Equal(t, first.page, *page)
	}
	// mỗi thành phần có MỘT item mặc định MANUAL thang 10
	var items int
	e.one(&items, `select count(*) from grade_items where course_id = $1 and max_score = 10 and source = 'MANUAL'`, e.course)
	require.Equal(t, 3, items)
	var audits int
	e.one(&audits, `select count(*) from audit_log where course_id = $1 and entity = 'grade_schemes' and action = 'extract'`, e.course)
	require.Equal(t, 1, audits)
}

// TestExtractLop2OneUnclear — AC3: quy chế lớp 2 thiếu làm tròn → đúng MỘT mục chưa rõ ROUNDING_MISSING (source CODE), không mục nào khác.
func TestExtractLop2OneUnclear(t *testing.T) {
	e := newEnv(t, constLLM(recorded(t, "extraction-lop2.json", nil)))
	res, err := e.run(e.policyDoc("quy-che-lop2.md"), false, 0)
	require.NoError(t, err)
	require.Equal(t, grade.CodeRoundingMissing, res.(grading.Result).UnclearCode)
	us := e.draftUnclear()
	require.Len(t, us, 1)
	require.Equal(t, unclearRow{Source: "CODE", Code: grade.CodeRoundingMissing, Target: "rules.ROUNDING"}, us[0])
	var ws []string
	rows, err := e.pool.Query(e.ctx, `select weight::text from grade_components where course_id = $1 order by position`, e.course)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var w string
		require.NoError(t, rows.Scan(&w))
		ws = append(ws, w)
	}
	require.Equal(t, []string{"25.00", "25.00", "50.00"}, ws)
}

// TestExtractNeverConfirms — AC6: sau việc trích bản nháp luôn DRAFT, không confirmed_*; hàm xác nhận không tồn tại trong gói.
func TestExtractNeverConfirms(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	_, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	var n int
	e.one(&n, `select count(*) from grade_schemes where course_id = $1 and (status <> 'DRAFT' or confirmed_by is not null or confirmed_at is not null)`, e.course)
	require.Zero(t, n)
}

// ---- nhánh lỗi ----

func apiCode(t *testing.T, err error) (int, string) {
	t.Helper()
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	return ae.Status, ae.Code
}

func TestExtractNotPolicy(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	var doc uuid.UUID
	e.one(&doc, `insert into documents (course_id, title, type, status) values ($1, 'Bài giảng', 'LECTURE', 'READY') returning id`, e.course)
	_, err := e.svc.Request(e.ctx, e.teacher, e.course, doc, false)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, st)
	require.Equal(t, apierr.DocumentNotPolicy, code)
	var jobsN int
	e.one(&jobsN, `select count(*) from jobs where kind = $1`, grading.KindExtract)
	require.Zero(t, jobsN, "không xếp việc")
	require.Zero(t, e.fake.count())
}

func TestExtractNotReady(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	var doc uuid.UUID
	e.one(&doc, `insert into documents (course_id, title, type, status) values ($1, 'Quy chế', 'COURSE_POLICY', 'PROCESSING') returning id`, e.course)
	_, err := e.svc.Request(e.ctx, e.teacher, e.course, doc, false)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.DocumentNotReady, code)
	// tài liệu của lớp khác → 404
	other := e.newCourse()
	var od uuid.UUID
	e.one(&od, `insert into documents (course_id, title, type, status) values ($1, 'Quy chế', 'COURSE_POLICY', 'READY') returning id`, other)
	_, err = e.svc.Request(e.ctx, e.teacher, e.course, od, false)
	st, _ = apiCode(t, err)
	require.Equal(t, http.StatusNotFound, st)
}

func TestExtractTooLong(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	e.svc.Cfg.MaxChars = 50
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.svc.Request(e.ctx, e.teacher, e.course, doc, false)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, st)
	require.Equal(t, apierr.PolicyTooLong, code)
}

// TestExtractLLMDown — AC7(c): lỗi TẠM (ErrUnavailable) thử lại ≤ 3 lần rồi FAILED LLM_UNAVAILABLE; không bản nháp rác.
func TestExtractLLMDown(t *testing.T) {
	f := &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) {
		return nil, &llm.ErrUnavailable{Reason: llm.ReasonAllFailed}
	}}
	e := newEnv(t, f)
	_, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
	require.Equal(t, "LLM_UNAVAILABLE", asUserErr(t, err).Code)
	require.Equal(t, 3, f.count())
	var n int
	e.one(&n, `select count(*) from grade_schemes where course_id = $1`, e.course)
	require.Zero(t, n)
	// hết ngân sách không phải lỗi tạm: không thử lại
	f2 := &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) {
		return nil, &llm.ErrUnavailable{Reason: llm.ReasonBudgetExhausted}
	}}
	e2 := newEnv(t, f2)
	_, err = e2.run(e2.policyDoc("quy-che-lop1.md"), false, 0)
	require.Equal(t, "LLM_UNAVAILABLE", asUserErr(t, err).Code)
	require.Equal(t, 1, f2.count())
	// lỗi tạm rồi thành công ở lần 2
	f3 := &fakeLLM{fn: func(n int, _ context.Context) (json.RawMessage, error) {
		if n == 1 {
			return nil, &llm.ErrOverloaded{RetryAfter: time.Second}
		}
		return lop1(t), nil
	}}
	e3 := newEnv(t, f3)
	_, err = e3.run(e3.policyDoc("quy-che-lop1.md"), false, 0)
	require.NoError(t, err)
	require.Equal(t, 2, f3.count())
}

// TestExtractInvalidSchema — AC7(d), TLR-11: đầu ra không đọc được → FAILED EXTRACTION_INVALID, đúng 1 lời gọi (không thử lại, không "nhắc schema").
func TestExtractInvalidSchema(t *testing.T) {
	for name, out := range map[string]json.RawMessage{
		"không phải JSON":     json.RawMessage(`xin chào`),
		"khoá lạ":             json.RawMessage(`{"components":[],"rules":[],"citations":[],"unclear_items":[],"extra":1}`),
		"khoá thành phần xấu": recorded(t, "extraction-lop1.json", func(m map[string]any) { m["components"].([]any)[0].(map[string]any)["key"] = "Giữa Kỳ" }),
		"loại luật lạ":        recorded(t, "extraction-lop1.json", func(m map[string]any) { m["rules"].([]any)[0].(map[string]any)["kind"] = "CURVE" }),
	} {
		t.Run(name, func(t *testing.T) {
			f := constLLM(out)
			e := newEnv(t, f)
			_, err := e.run(e.policyDoc("quy-che-lop1.md"), false, 0)
			require.Equal(t, "EXTRACTION_INVALID", asUserErr(t, err).Code)
			require.Equal(t, 1, f.count())
			var n int
			e.one(&n, `select count(*) from grade_schemes where course_id = $1`, e.course)
			require.Zero(t, n)
		})
	}
}

// TestExtractFailureKeepsConfirmed — AC7: việc thất bại không đụng bản CONFIRMED.
func TestExtractFailureKeepsConfirmed(t *testing.T) {
	f := constLLM(lop1(t))
	e := newEnv(t, f)
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.run(doc, false, 0)
	require.NoError(t, err)
	e.exec(`update grade_schemes set status = 'CONFIRMED', confirmed_by = $2, confirmed_at = now() where course_id = $1`, e.course, e.teacher)
	f.fn = func(int, context.Context) (json.RawMessage, error) {
		return nil, &llm.ErrUnavailable{Reason: llm.ReasonAllFailed}
	}
	_, err = e.run(doc, false, 0)
	require.Error(t, err)
	var status string
	var no int
	e.one(&status, `select status::text from grade_schemes where course_id = $1`, e.course)
	e.one(&no, `select scheme_no from grade_schemes where course_id = $1`, e.course)
	require.Equal(t, "CONFIRMED", status)
	require.Equal(t, 1, no)
	// trích thành công khi đã có CONFIRMED: bản nháp mới nằm cạnh, CONFIRMED không đổi
	f.fn = func(int, context.Context) (json.RawMessage, error) { return lop1(t), nil }
	_, err = e.run(doc, false, 0)
	require.NoError(t, err)
	var n int
	e.one(&n, `select count(*) from grade_schemes where course_id = $1 and status = 'CONFIRMED' and scheme_no = 1`, e.course)
	require.Equal(t, 1, n)
	e.one(&n, `select count(*) from grade_schemes where course_id = $1 and status = 'DRAFT' and scheme_no = 2`, e.course)
	require.Equal(t, 1, n)
}

// TestExtractIdempotentKey — AC8: cùng việc chạy hai lần (nhận lại) → đúng một bản nháp, cùng scheme_id; extract_job_id duy nhất.
func TestExtractIdempotentKey(t *testing.T) {
	f := constLLM(lop1(t))
	e := newEnv(t, f)
	doc := e.policyDoc("quy-che-lop1.md")
	id, raw := e.enqueue(grading.Payload{CourseID: e.course, DocumentID: doc})
	r1, err := e.svc.Run(e.ctx, e.teacher, id, raw, func(int) {})
	require.NoError(t, err)
	r2, err := e.svc.Run(e.ctx, e.teacher, id, raw, func(int) {})
	require.NoError(t, err)
	require.Equal(t, r1.(grading.Result).SchemeID, r2.(grading.Result).SchemeID)
	require.Equal(t, 1, f.count(), "việc đã lưu thì không gọi LLM lần nữa")
	var n int
	e.one(&n, `select count(*) from grade_schemes where course_id = $1`, e.course)
	require.Equal(t, 1, n)
	e.one(&n, `select count(*) from audit_log where entity = 'grade_schemes' and action = 'extract'`)
	require.Equal(t, 1, n)
}

// TestExtractKillWorkerMidRun — AC8: worker chết sau khi gọi LLM, trước khi lưu → việc nhận lại tạo ĐÚNG MỘT bản nháp.
func TestExtractKillWorkerMidRun(t *testing.T) {
	f := constLLM(lop1(t))
	e := newEnv(t, f)
	doc := e.policyDoc("quy-che-lop1.md")
	id, raw := e.enqueue(grading.Payload{CourseID: e.course, DocumentID: doc})
	killed := &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) {
		return nil, errors.New("tiến trình bị giết")
	}}
	dead := *e.svc
	dead.LLM = killed
	_, err := dead.Run(e.ctx, e.teacher, id, raw, func(int) {})
	require.Error(t, err)
	var n int
	e.one(&n, `select count(*) from grade_schemes where course_id = $1`, e.course)
	require.Zero(t, n)
	_, err = e.svc.Run(e.ctx, e.teacher, id, raw, func(int) {}) // worker khác nhận lại
	require.NoError(t, err)
	e.one(&n, `select count(*) from grade_schemes where course_id = $1`, e.course)
	require.Equal(t, 1, n)
}

// editDraft mô phỏng giảng viên sửa tay bản nháp: version + 1 và một thành phần filled_by_user.
func (e *env) editDraft() {
	e.exec(`update grade_components set filled_by_user = true where id = (select id from grade_components where course_id = $1 order by position limit 1)`, e.course)
	e.exec(`update grade_schemes set version = version + 1 where course_id = $1 and status = 'DRAFT'`, e.course)
}

// TestExtractDraftExists — AC8: có DRAFT đã sửa tay mà không replace_draft → 409 DRAFT_EXISTS kèm số mục đã sửa.
func TestExtractDraftExists(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.run(doc, false, 0)
	require.NoError(t, err)
	// bản nháp chưa sửa: trích lại được (thay bản máy)
	_, err = e.svc.Request(e.ctx, e.teacher, e.course, doc, false)
	require.NoError(t, err)
	e.editDraft()
	_, err = e.svc.Request(e.ctx, e.teacher, e.course, doc, false)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.DraftExists, code)
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, map[string]any{"edited_items": 1}, ae.Details)
}

// TestExtractReplaceDraft — AC8: replace_draft thay bản nháp cũ (audit ghi).
func TestExtractReplaceDraft(t *testing.T) {
	e := newEnv(t, constLLM(lop1(t)))
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.run(doc, false, 0)
	require.NoError(t, err)
	e.editDraft()
	var ver int
	e.one(&ver, `select version from grade_schemes where course_id = $1 and status = 'DRAFT'`, e.course)
	_, err = e.svc.Request(e.ctx, e.teacher, e.course, doc, true)
	require.NoError(t, err)
	res, err := e.run(doc, true, ver)
	require.NoError(t, err)
	require.Equal(t, 1, res.(grading.Result).SchemeNo, "bản nháp chưa từng xác nhận bị thay thì số công thức dùng lại")
	var n int
	e.one(&n, `select count(*) from grade_schemes where course_id = $1`, e.course)
	require.Equal(t, 1, n, "bản nháp cũ bị thay")
	e.one(&n, `select count(*) from grade_components where course_id = $1 and filled_by_user`, e.course)
	require.Zero(t, n)
	e.one(&n, `select count(*) from audit_log where course_id = $1 and action = 'extract_replace_draft'`, e.course)
	require.Equal(t, 1, n)
}

// TestExtractDraftEditedDuringJob — TLR-10: giảng viên sửa bản nháp trong lúc việc chạy (không replace_draft) → FAILED DRAFT_EXISTS, bản nháp nguyên vẹn.
func TestExtractDraftEditedDuringJob(t *testing.T) {
	f := constLLM(lop1(t))
	e := newEnv(t, f)
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.run(doc, false, 0) // bản nháp máy
	require.NoError(t, err)
	f.fn = func(int, context.Context) (json.RawMessage, error) {
		e.editDraft() // sửa chen ngang trong lúc LLM chạy
		return lop1(t), nil
	}
	_, err = e.run(doc, false, 1)
	require.Equal(t, "DRAFT_EXISTS", asUserErr(t, err).Code)
	var filled, ver int
	e.one(&filled, `select count(*) from grade_components where course_id = $1 and filled_by_user`, e.course)
	e.one(&ver, `select version from grade_schemes where course_id = $1 and status = 'DRAFT'`, e.course)
	require.Equal(t, 1, filled)
	require.Equal(t, 2, ver)
}

// TestExtractReplaceDraftVersionChanged — replace_draft nhưng version đổi giữa lúc gọi và lúc lưu → FAILED DRAFT_EXISTS.
func TestExtractReplaceDraftVersionChanged(t *testing.T) {
	f := constLLM(lop1(t))
	e := newEnv(t, f)
	doc := e.policyDoc("quy-che-lop1.md")
	_, err := e.run(doc, false, 0)
	require.NoError(t, err)
	f.fn = func(int, context.Context) (json.RawMessage, error) {
		e.editDraft()
		return lop1(t), nil
	}
	_, err = e.run(doc, true, 1) // gọi lúc version = 1
	require.Equal(t, "DRAFT_EXISTS", asUserErr(t, err).Code)
	var filled int
	e.one(&filled, `select count(*) from grade_components where course_id = $1 and filled_by_user`, e.course)
	require.Equal(t, 1, filled, "chữ người dùng vừa gõ còn nguyên")
}
