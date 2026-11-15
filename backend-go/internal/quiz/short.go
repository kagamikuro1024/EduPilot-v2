package quiz

import (
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

// NormalizeShort chuẩn hoá câu trả lời ngắn để so khớp (SRS FEAT-practice FR-1): Unicode NFC; bỏ khoảng trắng đầu / cuối, gộp khoảng trắng giữa;
// chữ thường; bỏ dấu câu ở CUỐI. Dấu tiếng Việt được GIỮ ("Hà Nội" ≠ "ha noi").
func NormalizeShort(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(norm.NFC.String(s)), " "))
	return strings.TrimRightFunc(s, func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSpace(r) })
}

// gradeShort: đúng nếu câu trả lời chuẩn hoá trùng MỘT đáp án chấp nhận đã chuẩn hoá; sai hoặc trống → 0. Không đáp án chấp nhận nào dùng được → ErrBadKey.
func gradeShort(points decimal.Decimal, key Key, a *Answer) (decimal.Decimal, error) {
	accepted := make(map[string]struct{}, len(key.Accepted))
	for _, k := range key.Accepted {
		if n := NormalizeShort(k); n != "" {
			accepted[n] = struct{}{}
		}
	}
	if len(accepted) == 0 {
		return decimal.Zero, ErrBadKey
	}
	if a == nil || a.Text == nil {
		return decimal.Zero, nil
	}
	if _, ok := accepted[NormalizeShort(*a.Text)]; ok {
		return points, nil
	}
	return decimal.Zero, nil
}
