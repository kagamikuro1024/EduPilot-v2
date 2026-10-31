// Package textfold bỏ dấu tiếng Việt để so khớp trích dẫn. `Fold` dựng y hệt `vn_fold` của SQL (`translate(lower($1), …)` 1:1 theo rune —
// `TestFoldParitySQL` so Go với SQL); `Index` thêm gộp khoảng trắng và trả vị trí theo UTF-16 code unit trong văn bản gốc (màn duyệt tô sáng đoạn trích).
package textfold

import (
	"strings"
	"unicode"
	"unicode/utf16"
)

const (
	from = "àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ"
	to   = "aaaaaaaaaaaaaaaaaeeeeeeeeeeeiiiiiooooooooooooooooouuuuuuuuuuuyyyyyd"
)

var table = func() map[rune]rune { //nolint:gochecknoglobals // bảng bất biến, dựng một lần
	m := make(map[rune]rune, 67)
	f, t := []rune(from), []rune(to)
	for i, r := range f {
		m[r] = t[i]
	}
	return m
}()

func fold(r rune) rune {
	r = unicode.ToLower(r)
	if x, ok := table[r]; ok {
		return x
	}
	return r
}

// Fold = `vn_fold`: chữ thường rồi đổi 67 ký tự có dấu; giữ nguyên mọi ký tự khác kể cả khoảng trắng.
func Fold(s string) string { return strings.Map(fold, s) }

// Collapse bỏ dấu, chữ thường, gộp mọi chuỗi khoảng trắng thành một dấu cách và cắt hai đầu (dạng so sánh của trích dẫn).
func Collapse(s string) string {
	f, _ := collapse(s)
	return string(f)
}

// collapse trả các rune đã gập và, với mỗi rune, [bắt đầu, kết thúc) theo UTF-16 code unit của rune gốc (chuỗi khoảng trắng gộp: cả đoạn).
func collapse(s string) ([]rune, [][2]int) {
	var out []rune
	var pos [][2]int
	u, spaceStart := 0, -1
	for _, r := range s {
		w := len(utf16.Encode([]rune{r}))
		if unicode.IsSpace(r) {
			if spaceStart < 0 {
				spaceStart = u
			}
		} else {
			if spaceStart >= 0 {
				if len(out) > 0 {
					out = append(out, ' ')
					pos = append(pos, [2]int{spaceStart, u})
				}
				spaceStart = -1
			}
			out = append(out, fold(r))
			pos = append(pos, [2]int{u, u + w})
		}
		u += w
	}
	return out, pos
}

// Index tìm `quote` trong `text` (cả hai gập như Collapse). Trả [start, end) theo UTF-16 code unit của `text` gốc; quote rỗng → không khớp.
func Index(text, quote string) (start, end int, ok bool) {
	q := []rune(Collapse(quote))
	if len(q) == 0 {
		return 0, 0, false
	}
	t, pos := collapse(text)
	for i := 0; i+len(q) <= len(t); i++ {
		match := true
		for j := range q {
			if t[i+j] != q[j] {
				match = false
				break
			}
		}
		if match {
			return pos[i][0], pos[i+len(q)-1][1], true
		}
	}
	return 0, 0, false
}

// Words tách `\w+` (chữ, số, `_`) sau Fold — dùng cho ANSWER_ECHO.
func Words(s string) []string {
	return strings.FieldsFunc(Fold(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' })
}
