package assignment_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/regress"
)

const canaryText = "CANARY_NHAN_XET_SINH_VIEN_Nguyễn_Văn_A_B21DCAT123_" + "rất_dài_không_được_lọt_vào_ca_hồi_quy_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

func (e *env) regressOn() {
	flags := grading.NewGuidanceFlags(e.pool, nil)
	flags.Set(map[uuid.UUID][]string{e.course: {grading.FlagRegressionCapture}})
	e.svc.Guidance = flags
	e.svc.RegressMinDelta = decimal.RequireFromString("1.0")
}

func (e *env) cases(kind string) (n int, expected string, crit string) {
	e.t.Helper()
	e.one(&n, `select count(*)::int from grading_regression_cases where course_id = $1 and kind = $2`, e.course, kind)
	if n > 0 {
		e.one(&expected, `select string_agg(expected::text, '|' order by criterion_id) from grading_regression_cases where course_id = $1 and kind = $2`, e.course, kind)
		e.one(&crit, `select string_agg(coalesce(criterion_id, ''), ',' order by criterion_id) from grading_regression_cases where course_id = $1 and kind = $2`, e.course, kind)
	}
	return n, expected, crit
}

// TestRegressionCaseNoText — M2 (FEAT-grading-guidance 4.3): ca GRADING_EDIT chỉ có tham chiếu + số / enum (chuỗi ≤ 64 ký tự); nhận xét của giảng viên (canary) không vào; tiêu chí lệch < MIN_DELTA không có ca; cờ lớp tắt → không ghi gì.
func TestRegressionCaseNoText(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	crit := `[{"id":"c1","score":"2","comment":"` + canaryText + `","edited":true},{"id":"c2","score":"5.5","comment":"` + canaryText + `","edited":true}]`
	off := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit, nil)
	_, err := e.review(off.res, 1)
	require.NoError(t, err)
	n, _, _ := e.cases(regress.KindGradingEdit)
	require.Zero(t, n, "cờ REGRESSION_CAPTURE tắt")

	e.regressOn()
	r := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", crit, nil)
	_, err = e.review(r.res, 1)
	require.NoError(t, err)
	n, exp, cid := e.cases(regress.KindGradingEdit)
	require.Equal(t, 1, n, "c1 lệch 2,0 ≥ 1,0; c2 lệch 0,5 < 1,0")
	require.Equal(t, "c1", cid)
	require.NotContains(t, exp, "CANARY")
	require.NotContains(t, exp, "B21DCAT123")
	var rubric []byte
	e.one(&rubric, `select rubric from assignments where id = $1`, a.ID)
	require.JSONEq(t, `{"ai_score":"4.00","teacher_score":"2.00","direction":"STRICTER","rubric_hash":"`+regress.RubricHash(rubric)+`"}`, exp, "rubric_hash khớp rubric của bài tập")
}

// TestRegressionCaseIdempotent — duyệt lại (sửa rồi duyệt lần nữa) không sinh ca thứ hai: UPSERT theo (lớp, loại, nguồn, tiêu chí) cập nhật `expected`.
func TestRegressionCaseIdempotent(t *testing.T) {
	e := newEnv(t)
	e.regressOn()
	a := e.create("BT03")
	r := e.rrow(a, e.student(), 1, time.Now().UTC().Add(-time.Hour), "DONE", "DRAFT", `[{"id":"c1","score":"2","comment":"x","edited":true},{"id":"c2","score":"6"}]`, nil)
	_, err := e.review(r.res, 1)
	require.NoError(t, err)
	n, _, _ := e.cases(regress.KindGradingEdit)
	require.Equal(t, 1, n)
	var ver int32
	e.exec(`update grading_results set status = 'DRAFT', criteria = '[{"id":"c1","score":"1.5","edited":true},{"id":"c2","score":"6"}]'::jsonb, version = version + 1 where id = $1`, r.res)
	e.one(&ver, `select version from grading_results where id = $1`, r.res)
	_, err = e.review(r.res, ver)
	require.NoError(t, err)
	n, exp, _ := e.cases(regress.KindGradingEdit)
	require.Equal(t, 1, n, "không thêm ca")
	require.True(t, strings.Contains(exp, `"teacher_score": "1.50"`), exp)
}

// TestRegressionCaseAppeal — phúc khảo ADJUSTED ghi ca GRADING_APPEAL cho tiêu chí đã đổi, so với điểm AI gốc.
func TestRegressionCaseAppeal(t *testing.T) {
	e := newEnv(t)
	e.regressOn()
	a := e.create("BT03")
	id, _, _ := e.filed(a)
	_, err := e.decide(id, adj("3"))
	require.NoError(t, err)
	n, exp, cid := e.cases(regress.KindGradingAppeal)
	require.Equal(t, 1, n)
	require.Equal(t, "c2", cid)
	require.Contains(t, exp, `"ai_score": "6.00"`)
	require.Contains(t, exp, `"teacher_score": "3.00"`)
}
