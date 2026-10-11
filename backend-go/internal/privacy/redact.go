package privacy

import "strings"

// RedactMark là dấu thay cho mỗi khoảng bị ẩn (không lộ độ dài gốc).
const RedactMark = "[đã ẩn]"

// Redact thay mỗi khoảng bằng RedactMark (hai khoảng chạm / chồng nhau gộp thành một). Không có khoảng nào thì trả CÙNG chuỗi. Idempotent.
func Redact(text string, fs []Finding) string {
	if len(fs) == 0 {
		return text
	}
	runes := []rune(text)
	var b strings.Builder
	b.Grow(len(text))
	pos := 0
	for i := 0; i < len(fs); i++ {
		s, e := fs[i].Start, fs[i].End
		for i+1 < len(fs) && fs[i+1].Start <= e {
			i++
			e = max(e, fs[i].End)
		}
		b.WriteString(string(runes[pos:s]))
		b.WriteString(RedactMark)
		pos = e
	}
	b.WriteString(string(runes[pos:]))
	return b.String()
}
