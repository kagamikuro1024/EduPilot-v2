package calendar

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// TestICSFolding — AC8: dòng dài gập ở 75 octet (CRLF + khoảng trắng), không cắt giữa ký tự UTF-8, ghép lại đúng nguyên văn.
func TestICSFolding(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"SUMMARY:" + strings.Repeat("a", 200), "SUMMARY:" + strings.Repeat("Bài thi giữa kỳ ", 20), "SUMMARY:" + strings.Repeat("😀", 60)} {
		out := fold(in)
		for i, l := range strings.Split(out, "\r\n") {
			require.LessOrEqual(t, len(l), 75, "dòng %d: %q", i, l)
			require.True(t, utf8.ValidString(l), "cắt giữa ký tự: %q", l)
			if i > 0 {
				require.True(t, strings.HasPrefix(l, " "))
			}
		}
		require.Equal(t, in, strings.ReplaceAll(out, "\r\n ", ""))
	}
	require.Equal(t, "SHORT:x", fold("SHORT:x"))
}

// TestICSEscaping — AC8: `\`, `;`, `,` và xuống dòng được thoát (RFC 5545 §3.3.11).
func TestICSEscaping(t *testing.T) {
	t.Parallel()
	require.Equal(t, `a\\b\;c\,d\ne\nf\ng`, escape("a\\b;c,d\ne\r\nf\rg"))
}

// TestCalendarToolsVietnamTime — AC10: giờ hiển thị theo Asia/Ho_Chi_Minh, kiểu Việt.
func TestCalendarToolsVietnamTime(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Thứ Hai, 21/09 · 14:00", Format(time.Date(2026, 9, 21, 7, 0, 0, 0, time.UTC)))
	require.Equal(t, "Chủ Nhật, 28/06 · 00:30", Format(time.Date(2026, 6, 27, 17, 30, 0, 0, time.UTC))) // qua nửa đêm UTC
}
