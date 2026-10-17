package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStudentAsksAboutOwnNotes — US-P5-05 AC8: sinh viên hỏi về ghi chú / nhãn của chính mình → không tool nào chạy, không khối dữ liệu cá nhân, không placeholder `[[…]]`.
func TestStudentAsksAboutOwnNotes(t *testing.T) {
	t.Parallel()
	for _, q := range []string{"Giảng viên ghi gì về em?", "Em có bị coi là cần chú ý không?", "Thầy có ghi chú gì về em không ạ?", "Em có nằm trong diện rủi ro không?"} {
		r := newRig(t, true, 3)
		out := r.respond(t, q)
		require.Empty(t, out.Blocks, q)
		for name, c := range map[string]int64{"attendance": r.spy.attendance.Load(), "participation": r.spy.participation.Load(), "grade": r.spy.grade.Load(), "whatif": r.spy.whatif.Load(), "exam": r.spy.exam.Load(), "upcoming": r.spy.upcoming.Load(), "library": r.spy.library.Load()} {
			require.Zero(t, c, "%s: tool %s không được chạy", q, name)
		}
	}
}

// TestNoToolReadsObservations — không nguồn nào của Agent / bộ tool riêng có đường đọc ghi chú quan sát hay nhãn rủi ro (kiểm cấu trúc, bổ sung cho kiểm mã nguồn ở risk_access_test.go).
func TestNoToolReadsObservations(t *testing.T) {
	t.Parallel()
	check := func(typ reflect.Type) {
		for i := range typ.NumField() {
			f := typ.Field(i)
			name := strings.ToLower(f.Name + " " + f.Type.String())
			for _, bad := range []string{"risk", "observation", "crm."} {
				require.NotContains(t, name, bad, "%s.%s", typ.Name(), f.Name)
			}
		}
	}
	check(reflect.TypeOf(Agent{}))
	check(reflect.TypeOf(TrustedContext{}))
}
