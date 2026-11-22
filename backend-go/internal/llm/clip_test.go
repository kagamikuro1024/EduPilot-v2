package llm

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Phần thân INJECTION gửi provider ≤ trần 16.000 ký tự TÍNH CẢ dấu nối (DEF-R105S-04: trước đây 16.001).
func TestClipHeadTailStaysWithinCap(t *testing.T) {
	for _, n := range []int{16000, 16001, 16002, 40000} {
		s := strings.Repeat("a", n/2) + strings.Repeat("é", n-n/2) // đa byte: đếm theo rune
		got := clip(s, inputCap(GateInjection), true)
		if c := utf8.RuneCountInString(got); c > 16000 {
			t.Fatalf("n=%d: %d ký tự > 16000", n, c)
		}
	}
	if s := strings.Repeat("x", 16000); clip(s, 16000, true) != s {
		t.Fatal("đúng 16.000 ký tự phải giữ nguyên")
	}
	got := clip(strings.Repeat("h", 9000)+strings.Repeat("t", 9000), 16000, true)
	if !strings.HasPrefix(got, strings.Repeat("h", 8000)+"\n") || !strings.HasSuffix(got, "\n"+strings.Repeat("t", 7999)) {
		t.Fatal("giữ 8.000 đầu + dấu nối + 7.999 cuối")
	}
}
