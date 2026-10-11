package ingest

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func longText(words int, seed string) string {
	var b strings.Builder
	for i := range words {
		fmt.Fprintf(&b, "%s%d ", seed, i)
		if i%12 == 11 {
			b.WriteString("kết thúc câu. ")
		}
	}
	return strings.TrimSpace(b.String())
}

func TestChunkSizesNoOverlap(t *testing.T) {
	t.Parallel()
	md := "# Chương 1\n\n" + longText(600, "alpha") + "\n\n## Mục 1.1\n\n" + longText(300, "beta") + "\n\n" + strings.Repeat("x", 2500)
	cs := ChunkMarkdown(Clean(md), ChunkChars, MinChunkChars)
	require.Greater(t, len(cs), 5)
	seen := map[string]bool{}
	for i, c := range cs {
		n := utf8.RuneCountInString(c.Text)
		require.LessOrEqual(t, n, ChunkChars, "đoạn %d dài %d", i, n)
		require.Equal(t, i, c.Ord, "ord liên tục từ 0")
		require.False(t, seen[c.Text], "không trùng văn bản")
		seen[c.Text] = true
	}
	// không gối: nối lại các đoạn alpha theo thứ tự cho ra đúng dãy từ gốc, mỗi từ đúng một lần
	var all strings.Builder
	for _, c := range cs {
		if strings.Contains(c.Text, "alpha") {
			all.WriteString(c.Text + " ")
		}
	}
	require.Equal(t, 1, strings.Count(all.String(), "alpha0 "), "từ đầu chỉ xuất hiện một lần (không gối)")
	require.Equal(t, 1, strings.Count(all.String(), "alpha599 "))
}

func TestChunkMinMerge(t *testing.T) {
	t.Parallel()
	md := "# A\n\n" + strings.Repeat("a", 500) + "\n\n# B\n\nngắn\n\n# C\n\n" + strings.Repeat("c", 500)
	cs := ChunkMarkdown(Clean(md), ChunkChars, MinChunkChars)
	for _, c := range cs {
		require.GreaterOrEqual(t, utf8.RuneCountInString(c.Text), MinChunkChars, "đoạn ngắn phải được gộp: %q", c.Text)
		require.LessOrEqual(t, utf8.RuneCountInString(c.Text), ChunkChars)
	}
	require.Contains(t, cs[0].Text+cs[len(cs)-1].Text, "ngắn")
	// tài liệu ngắn hơn 200 rune: vẫn có đúng một đoạn
	one := ChunkMarkdown(Clean("# Tiêu đề\n\nChỉ một câu."), ChunkChars, MinChunkChars)
	require.Len(t, one, 1)
	require.Equal(t, "Chỉ một câu.", one[0].Text)
}

func TestChunkDedup(t *testing.T) {
	t.Parallel()
	para := strings.Repeat("Đoạn lặp lại y hệt. ", 20)
	md := para + "\n\n# Mục\n\n" + para + "\n\n# Mục khác\n\n" + strings.Repeat("Khác hẳn. ", 30)
	cs := ChunkMarkdown(Clean(md), ChunkChars, MinChunkChars)
	n := 0
	for _, c := range cs {
		if strings.TrimSpace(c.Text) == strings.TrimSpace(para) {
			n++
		}
	}
	require.LessOrEqual(t, n, 1)
}

func TestChunkKeepsPageAndHeading(t *testing.T) {
	t.Parallel()
	md := "# Quy chế\n\n" + strings.Repeat("Trang một. ", 40) + PageMarker + "\n\n## Điều 5. Cảnh báo học vụ\n\n" + strings.Repeat("Điều kiện bị cảnh báo. ", 30) + "\n" + PageMarker + "\n" + strings.Repeat("Trang ba. ", 50)
	cs := ChunkMarkdown(Clean(md), ChunkChars, MinChunkChars)
	require.Equal(t, 3, PageCount(md))
	var gotP2, gotP3 bool
	for _, c := range cs {
		switch {
		case strings.HasPrefix(c.Text, "Điều kiện"):
			require.Equal(t, 2, c.PageNo)
			require.Equal(t, "Điều 5. Cảnh báo học vụ", c.Heading)
			gotP2 = true
		case strings.HasPrefix(c.Text, "Trang ba"):
			require.Equal(t, 3, c.PageNo)
			require.Equal(t, "Điều 5. Cảnh báo học vụ", c.Heading, "tiêu đề gần nhất giữ qua trang")
			gotP3 = true
		case strings.HasPrefix(c.Text, "Trang một"):
			require.Equal(t, 1, c.PageNo)
			require.Equal(t, "Quy chế", c.Heading)
		}
	}
	require.True(t, gotP2 && gotP3)
	long := "# " + strings.Repeat("T", 300) + "\n\nnội dung đủ dài " + strings.Repeat("x", 250)
	require.LessOrEqual(t, utf8.RuneCountInString(ChunkMarkdown(Clean(long), ChunkChars, MinChunkChars)[0].Heading), 200)
}

func TestChunkVietnameseDiacritics(t *testing.T) {
	t.Parallel()
	// 1.000 chữ "ệ" (3 byte): cắt cứng không được rơi giữa một rune; mỗi đoạn là UTF-8 hợp lệ và đếm theo rune
	md := strings.Repeat("ệ", 1000) + " " + strings.Repeat("Điều kiện tốt nghiệp. ", 80)
	for _, c := range ChunkMarkdown(Clean(md), ChunkChars, MinChunkChars) {
		require.True(t, utf8.ValidString(c.Text))
		require.LessOrEqual(t, utf8.RuneCountInString(c.Text), ChunkChars)
	}
}

func TestCleanStripsControlAndImages(t *testing.T) {
	t.Parallel()
	got := Clean("a\x00b\x07 \t c\r\n<!-- image -->\n\n\n\nd   e")
	require.Equal(t, "ab c\n\nd e", got)
}

func TestEmbedInputIsHeadingPlusText(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Điều 5\nnội dung", EmbedInput(Chunk{Heading: "Điều 5", Text: "nội dung"}))
	require.Equal(t, "nội dung", EmbedInput(Chunk{Text: "nội dung"}))
}
