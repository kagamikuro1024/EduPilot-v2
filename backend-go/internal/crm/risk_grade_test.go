package crm_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/store"
)

const gradeRules = `[{"kind":"BONUS","applicable":false},{"kind":"PENALTY","applicable":false},{"kind":"ROUNDING","params":{"step":"0.5","mode":"HALF_UP"}},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"5","letter":"A"},{"min":"0","letter":"F"}]}}]`

// confirmScheme dựng công thức weekly 40 (QT) + final 60 (CK) đã xác nhận cho lớp của world; trả id item `weekly`.
func (w world) confirmScheme() uuid.UUID {
	w.r.t.Helper()
	w.r.exec(`insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 1, $2)`, w.course, w.teacher.ID)
	svc := &grade.Service{Pool: w.r.pool}
	var rules []grade.PutRule
	require.NoError(w.r.t, json.Unmarshal([]byte(gradeRules), &rules))
	doc, err := svc.PutDraft(w.r.t.Context(), w.teacher.ID, w.course, grade.PutInput{Version: 1, Rules: rules, Components: []grade.PutComponent{{Key: "weekly", Name: "Hằng tuần", Grp: "QT", Weight: "40"}, {Key: "final", Name: "Cuối kỳ", Grp: "CK", Weight: "60"}}})
	require.NoError(w.r.t, err)
	_, err = svc.ConfirmScheme(w.r.t.Context(), w.teacher.ID, w.course, doc.Version)
	require.NoError(w.r.t, err)
	var item uuid.UUID
	require.NoError(w.r.t, w.r.pool.QueryRow(w.r.t.Context(), `select id from grade_items where course_id = $1 and key = 'weekly'`, w.course).Scan(&item))
	return item
}

func (w world) weekly(item, student uuid.UUID, score string) {
	w.r.exec(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ($1, $2, $3, $4::numeric, $5)`, w.course, item, student, score, w.teacher.ID)
}

// TestRiskGradeSignalDB: lớp CONFIRMED: điểm quá trình thấp + không hoạt động → AT_RISK kèm lý do GRADE_LOW; đúng ngưỡng 5,0 không thấp; chưa đủ điểm → không có tín hiệu.
func TestRiskGradeSignalDB(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	item := w.confirmScheme()
	low, edge, none := w.students[0].ID, w.students[1].ID, w.students[2].ID
	w.weekly(item, low, "2")
	w.weekly(item, edge, "5")
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	x, _ := w.risk(low)
	require.Equal(t, "AT_RISK", x.level)
	reasons := r.scalar(`select reasons::text from student_risk where course_id = $1 and student_id = $2`, w.course, low)
	require.Contains(t, reasons, "GRADE_LOW")
	require.Contains(t, reasons, `"qt_milli": 2000`)
	for name, s := range map[string]uuid.UUID{"đúng 5,0": edge, "chưa có điểm": none} {
		y, _ := w.risk(s)
		require.Equal(t, "NONE", y.level, name)
	}
	// ngưỡng lớp đổi được: 6,0 → 5,0 thành thấp
	require.Equal(t, http.StatusOK, w.putRisk(w.T, w.version(), map[string]any{"grade_low_below": "6.0"}).code)
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	y, _ := w.risk(edge)
	require.Equal(t, "AT_RISK", y.level)
	for _, bad := range []string{"5.3", "10.5", "-1", "abc"} {
		res := w.putRisk(w.T, w.version(), map[string]any{"grade_low_below": bad})
		require.Equal(t, http.StatusUnprocessableEntity, res.code, bad)
		require.Contains(t, string(res.body), "risk.grade_low_below")
	}
}

// TestRiskGradeNullWithoutScheme: lớp chưa CONFIRMED → GRADE = null như cũ (không tín hiệu, không lý do GRADE_LOW).
func TestRiskGradeNullWithoutScheme(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.absentTwoWeeks(w.students[0].ID)
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	snap := r.scalar(`select level::text || reasons::text || weeks::text from student_risk where course_id = $1 and student_id = $2`, w.course, w.students[0].ID)
	require.Contains(t, snap, "AT_RISK")
	require.NotContains(t, snap, "GRADE")
	qt, err := grade.RiskQT(t.Context(), store.New(r.pool), w.course)
	require.NoError(t, err)
	require.Empty(t, qt)
}

// TestRiskGradeBatchInput (TLR-16): số câu SQL của `RiskQT` không phụ thuộc số sinh viên.
func TestRiskGradeBatchInput(t *testing.T) {
	r := newRig(t)
	count := func(n int) int64 {
		w := r.world(n)
		item := w.confirmScheme()
		for _, s := range w.students {
			w.weekly(item, s.ID, "4")
		}
		before := r.qc.n.Load()
		qt, err := grade.RiskQT(t.Context(), store.New(r.pool), w.course)
		require.NoError(t, err)
		require.Len(t, qt, n)
		return r.qc.n.Load() - before
	}
	require.Equal(t, count(3), count(40), "cùng số câu SQL cho 3 và 40 sinh viên")
}
