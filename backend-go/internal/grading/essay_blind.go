package grading

import (
	"regexp"
	"strings"

	"github.com/edupilot/backend-go/internal/textfold"
)

var identityLine = regexp.MustCompile(`^\s*(ho ten|mssv|ma so|email)\b`) //nolint:gochecknoglobals // hằng

// StripIdentityLines (a′, TLR-6) xoá NGUYÊN dòng tiêu đề danh tính (`Họ tên: …`, `MSSV: …`, `Mã số: …`, `Email: …`; nhãn không phân biệt hoa thường / dấu) rồi gộp khoảng trắng thừa:
// bớt token và chấm mù. Mã KHÔNG tự đặt placeholder — che tên / MSSV / email còn lại là việc của privacy.Masker ở gateway. Mọi câu khác giữ nguyên (kể cả câu nghi chèn lệnh).
func StripIdentityLines(text string) string {
	var keep []string
	for _, l := range strings.Split(text, "\n") {
		if identityLine.MatchString(textfold.Fold(l)) {
			continue
		}
		keep = append(keep, strings.Join(strings.Fields(l), " "))
	}
	// gộp dòng trống liên tiếp
	var out []string
	blank := false
	for _, l := range keep {
		if l == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
