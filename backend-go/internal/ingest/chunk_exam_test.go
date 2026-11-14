package ingest_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/ingest"
)

// L1 (US-P9-02): Markdown docling THẬT của đề mẫu (24 câu, 4 trang). Mỗi dòng "## Câu N." là tiêu đề ngắn nên bị gộp (mergeShort):
// bản cũ bỏ tiêu đề của mẩu bị gộp (mất 13/24 câu và tên đề). `heading` + `text` của các đoạn phải chứa đủ chữ, và trích dẫn trang không lệch.
func TestChunkExamPaperKeepsEveryQuestion(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/de-mau-trac-nghiem.md")
	require.NoError(t, err)
	chunks := ingest.ChunkMarkdown(ingest.Clean(string(raw)), ingest.ChunkChars, ingest.MinChunkChars)

	var all strings.Builder
	pageOf := map[string]int{}
	for _, c := range chunks {
		body := c.Heading + "\n" + c.Text
		all.WriteString(body + "\n")
		for n := 1; n <= 24; n++ {
			if strings.Contains(body, fmt.Sprintf("Câu %d.", n)) {
				if _, ok := pageOf[fmt.Sprint(n)]; !ok {
					pageOf[fmt.Sprint(n)] = c.PageNo
				}
			}
		}
	}
	require.Contains(t, all.String(), "ĐỀ MẪU LUYỆN TẬP — MẠNG MÁY TÍNH")
	for n := 1; n <= 24; n++ {
		require.Contains(t, all.String(), fmt.Sprintf("Câu %d.", n), "mất đề bài câu %d", n)
	}
	require.Equal(t, 4, pageOf["19"], "câu 19 nằm ở trang 4, không bị gộp sang đoạn trang 3")
	require.Equal(t, 1, pageOf["1"])
}
