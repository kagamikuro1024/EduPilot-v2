package grade_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
)

var numTok = regexp.MustCompile(`\d+(?:,\d+)?`)

func numSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, n := range numTok.FindAllString(s, -1) {
		m[n] = true
	}
	return m
}

// TestLineTextSameNumbersNoCode — P-11-15 (SRS 4.8.2a): câu cho sinh viên dựng từ CÙNG tập số với `expr`/`value`/nhãn, không có hàm / ÷ / ngoặc; `expr` của Staff không đổi.
// Chạy trên cả 30 sinh viên lớp 1 (có kẹp, điểm cộng vượt trần, điểm trừ, liệt) và các ca tổng hợp.
func TestLineTextSameNumbersNoCode(t *testing.T) {
	s, ids, rows := lop1(t)
	var lines []grade.Line
	for _, row := range rows {
		lines = append(lines, grade.Compute(s, inputOf(row, ids)).Lines...)
	}
	for _, c := range readCSV(t, fixDir+"cases-tong-hop.csv") {
		cs, in := caseScheme(t, c)
		lines = append(lines, grade.Compute(cs, in).Lines...)
	}
	withText := 0
	for _, l := range lines {
		if l.Expr == "" {
			continue // dòng «thiếu điểm» nội bộ: không ra sinh viên
		}
		withText++
		require.NotEmpty(t, l.Text, "dòng %q thiếu text", l.Label)
		all := append([]string{l.Text}, l.Detail...)
		for _, txt := range all {
			for _, bad := range []string{"min(", "max(", "round(", "÷", "(", ")", ";"} {
				require.NotContains(t, txt, bad, "text lộ biểu thức: %q", txt)
			}
		}
		allowed := numSet(l.Expr + " " + l.Value + " " + l.Label)
		for _, txt := range all {
			for n := range numSet(txt) {
				require.True(t, allowed[n] || n == "10" || n == "0", "số %s trong %q không có ở expr/value/nhãn (%q)", n, txt, l.Expr)
			}
		}
		if strings.HasPrefix(l.Label, "Điểm cuối kỳ") || l.Label == "Điểm quá trình" {
			require.Contains(t, l.Text, l.Value)
		}
	}
	require.Greater(t, withText, 100)
}
