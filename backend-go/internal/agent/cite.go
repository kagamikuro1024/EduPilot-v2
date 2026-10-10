package agent

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/rag"
)

// Citation là một nguồn được trích [n] trong câu trả lời (chat riêng và Threads dùng chung).
type Citation struct {
	N          int       `json:"n"`
	DocumentID uuid.UUID `json:"document_id"`
	Title      string    `json:"title"`
	PageNo     *int      `json:"page_no"`
	Snippet    string    `json:"snippet"`
}

var reMarker = regexp.MustCompile(`\s?\[(\d{1,3})\]`)

// ExtractCitations giữ các [n] có trong danh sách hits (n = vị trí + 1), bỏ [n] lạ khỏi văn bản; citations theo thứ tự xuất hiện, không trùng, snippet ≤ 200 rune.
func ExtractCitations(text string, hits []rag.Hit) (string, []Citation) {
	cites := []Citation{}
	seen := map[int]bool{}
	out := reMarker.ReplaceAllStringFunc(text, func(m string) string {
		sub := reMarker.FindStringSubmatch(m)
		n, _ := strconv.Atoi(sub[1])
		if n < 1 || n > len(hits) {
			return ""
		}
		if !seen[n] {
			seen[n] = true
			cites = append(cites, CitationOf(n, hits[n-1]))
		}
		return m
	})
	return out, cites
}

// CitationOf dựng mục trích dẫn thứ n từ một đoạn.
func CitationOf(n int, h rag.Hit) Citation {
	return Citation{N: n, DocumentID: h.DocumentID, Title: h.Title, PageNo: h.PageNo, Snippet: Cut(h.Text, 200)}
}

// Cut cắt tối đa n rune ở ranh giới câu / từ gần nhất.
func Cut(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	head := string(r[:n])
	if i := strings.LastIndexAny(head, ".!?"); i > n/3 {
		return head[:i+1]
	}
	if i := strings.LastIndex(head, " "); i > 0 {
		return head[:i]
	}
	return head
}
