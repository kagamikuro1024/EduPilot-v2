package exam

import (
	"regexp"
	"strings"
)

// citeMarker khớp mã đoạn nguồn nội bộ của prompt sinh câu («[C18]», «[C1, C2]») kèm MỘT khoảng trắng đứng trước.
var citeMarker = regexp.MustCompile(`\s?\[C\d+(?:\s*,\s*C\d+)*\]`)

// StripCiteMarkers bỏ mã đoạn nguồn nội bộ («[C18]») khỏi chữ người học đọc: mã chỉ có nghĩa với prompt (`sources`), không bao giờ là nội dung câu hỏi / lời giải.
// Dùng lúc LƯU câu AI sinh và lúc ĐỌC (câu đã lưu từ trước).
func StripCiteMarkers(s string) string {
	out := citeMarker.ReplaceAllString(s, "")
	if loc := citeMarker.FindStringIndex(s); loc != nil && loc[0] == 0 {
		out = strings.TrimLeft(out, " ") // mã đứng đầu: khoảng trắng SAU mã cũng bỏ
	}
	return out
}

// StripCiteMarkersPtr như StripCiteMarkers cho trường tuỳ chọn; nil giữ nil.
func StripCiteMarkersPtr(s *string) *string {
	if s == nil {
		return nil
	}
	c := StripCiteMarkers(*s)
	return &c
}
