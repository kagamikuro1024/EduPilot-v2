package privacy

import (
	"context"
	"strings"
	"unicode"
)

// maxHeld: số rune tối đa giữ lại khi gặp `[` (AC10).
const maxHeld = 32

// Unknown là chữ thay cho placeholder không có trong ánh xạ (hết hạn, mô hình bịa) — người dùng không bao giờ thấy placeholder.
const Unknown = "bạn"

func kinds() []string { return []string{"SV", "MSSV", "EMAIL", "SDT", "CCCD"} }

type pstate int

const (
	pInvalid pstate = iota
	pPrefix         // tiền tố hợp lệ, chờ thêm
	pComplete
)

// parsePlaceholder khớp held với `[[ \s* KIND _ \d+ \s* ]]` (không phân biệt hoa-thường) theo tiền tố.
// Trả pComplete kèm placeholder chuẩn (`[[SV_1]]`).
func parsePlaceholder(h []rune) (pstate, string) {
	i := 0
	if len(h) < 1 || h[0] != '[' {
		return pInvalid, ""
	}
	if len(h) == 1 {
		return pPrefix, ""
	}
	if h[1] != '[' {
		return pInvalid, ""
	}
	i = 2
	for i < len(h) && unicode.IsSpace(h[i]) {
		i++
	}
	// loại
	j := i
	for j < len(h) && h[j] != '_' && unicode.IsLetter(h[j]) {
		j++
	}
	word := strings.ToUpper(string(h[i:j]))
	if j == len(h) { // hết giữa chừng ở phần chữ
		if word == "" {
			return pPrefix, ""
		}
		for _, k := range kinds() {
			if strings.HasPrefix(k, word) {
				return pPrefix, ""
			}
		}
		return pInvalid, ""
	}
	known := false
	for _, k := range kinds() {
		if k == word {
			known = true
		}
	}
	if !known {
		return pInvalid, ""
	}
	if h[j] == ']' { // `[[CCCD]]`: dạng hỏng không có số — vẫn là placeholder sót, nuốt cả `]]`
		switch {
		case j+1 == len(h):
			return pPrefix, ""
		case h[j+1] == ']':
			return pComplete, "[[" + word + "]]"
		}
		return pInvalid, ""
	}
	if h[j] != '_' {
		return pInvalid, ""
	}
	j++
	d0 := j
	for j < len(h) && h[j] >= '0' && h[j] <= '9' {
		j++
	}
	digits := string(h[d0:j])
	if j == len(h) {
		return pPrefix, ""
	}
	if digits == "" {
		return pInvalid, ""
	}
	for j < len(h) && unicode.IsSpace(h[j]) {
		j++
	}
	if j == len(h) {
		return pPrefix, ""
	}
	if h[j] != ']' {
		return pInvalid, ""
	}
	if j+1 == len(h) {
		return pPrefix, ""
	}
	if h[j+1] != ']' {
		return pInvalid, ""
	}
	if j+2 != len(h) { // không xảy ra: ta dừng ngay khi đủ `]]`
		return pInvalid, ""
	}
	return pComplete, "[[" + word + "_" + strings.TrimLeft(digits, "0") + "]]"
}

// leftover: dạng placeholder (kể cả mở dở) mà người dùng không được thấy: `\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_\d*)?`.
func leftoverEnd(h []rune) int {
	if len(h) < 2 || h[0] != '[' || h[1] != '[' {
		return -1
	}
	i := 2
	for i < len(h) && unicode.IsSpace(h[i]) {
		i++
	}
	rest := strings.ToUpper(string(h[i:]))
	for _, k := range []string{"MSSV", "EMAIL", "SDT", "CCCD", "SV"} { // MSSV trước SV không cần (khác tiền tố), thứ tự cho rõ
		if strings.HasPrefix(rest, k) {
			j := i + len([]rune(k))
			if j < len(h) && h[j] == '_' {
				j++
				for j < len(h) && h[j] >= '0' && h[j] <= '9' {
					j++
				}
			}
			return j
		}
	}
	return -1
}

// StreamUnmasker khôi phục placeholder trên luồng token cắt ở vị trí bất kỳ. Không giữ gì tới khi gặp `[`; giữ tối đa maxHeld rune.
type StreamUnmasker struct {
	m      *Masker
	ctx    context.Context
	s      Session
	held   []rune
	unseen int
}

// NewStreamUnmasker dựng bộ khôi phục cho một lượt sinh.
func (m *Masker) NewStreamUnmasker(ctx context.Context, s Session) *StreamUnmasker {
	if s == nil {
		s = NewSession("")
	}
	return &StreamUnmasker{m: m, ctx: ctx, s: s}
}

// Write nhận một đoạn token, trả phần chữ đã chắc chắn phát được.
func (u *StreamUnmasker) Write(chunk string) string {
	if len(u.held) == 0 && !strings.ContainsRune(chunk, '[') {
		return chunk // đường nhanh: không có `[` thì không giữ gì
	}
	var out strings.Builder
	for _, r := range chunk {
		u.step(r, &out)
	}
	u.warnUnknown()
	return out.String()
}

func (u *StreamUnmasker) step(r rune, out *strings.Builder) {
	if len(u.held) == 0 {
		if r == '[' {
			u.held = append(u.held, r)
		} else {
			out.WriteRune(r)
		}
		return
	}
	u.held = append(u.held, r)
	switch st, ph := parsePlaceholder(u.held); {
	case st == pComplete:
		if orig, ok := u.m.original(u.ctx, u.s, ph); ok {
			out.WriteString(orig)
		} else {
			out.WriteString(Unknown)
			u.unseen++
		}
		u.held = u.held[:0]
	case st == pPrefix && len(u.held) < maxHeld:
		// chờ thêm
	default: // tiền tố hỏng hoặc đầy bộ đệm
		u.abandon(out)
	}
}

// abandon bỏ phần đang giữ: nếu mở đầu giống placeholder (`[[SV…`) thì thay phần đó bằng Unknown; không thì phát đúng một `[` rồi
// xử lý lại phần còn lại (một `[` sau đó có thể mở placeholder thật: `[[[SV_1]]`).
func (u *StreamUnmasker) abandon(out *strings.Builder) {
	h := append([]rune(nil), u.held...)
	u.held = u.held[:0]
	if end := leftoverEnd(h); end >= 0 {
		out.WriteString(Unknown)
		u.unseen++
		h = h[end:]
	} else {
		out.WriteRune(h[0])
		h = h[1:]
	}
	for _, r := range h {
		u.step(r, out)
	}
}

// Flush gọi ở cuối luồng: phần giữ lại là tiền tố dở dạng placeholder → Unknown; còn lại phát nguyên.
func (u *StreamUnmasker) Flush() string {
	var out strings.Builder
	for len(u.held) > 0 {
		u.abandon(&out)
	}
	u.warnUnknown()
	return out.String()
}

// warnUnknown ghi MỘT dòng warn chỉ có số lượng (không kèm ánh xạ hay nội dung).
func (u *StreamUnmasker) warnUnknown() {
	if u.unseen > 0 {
		u.m.warn("placeholder sót", "count", u.unseen)
		u.unseen = 0
	}
}
