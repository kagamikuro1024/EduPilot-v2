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
	Knowledge  bool      `json:"knowledge,omitempty"` // nguồn là tri thức đã xác nhận: giao diện không liên kết tới tài liệu
	// ThreadID: đoạn tri thức nguồn POST → liên kết tới thread đã xác nhận (`/threads/{id}`).
	ThreadID *uuid.UUID `json:"thread_id,omitempty"`
}

// reMarker: `[1]` hoặc danh sách `[1, 2]` / `[2, 5, 7]` (mô hình hay gộp nhiều nguồn vào một cặp ngoặc; trước đây chỉ [1] được hiểu).
var reMarker = regexp.MustCompile(`\s?\[(\d{1,3}(?:\s*,\s*\d{1,3})*)\]`)

// markerNums: các số trong một dấu trích dẫn (nhóm 1 của reMarker).
func markerNums(group string) []int {
	var out []int
	for _, f := range strings.Split(group, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// ExtractCitations giữ các [n] có trong danh sách hits (n = vị trí + 1), bỏ [n] lạ khỏi văn bản; citations theo thứ tự xuất hiện, không trùng, snippet ≤ 200 rune.
// Danh sách `[1, 2]` được tách thành `[1] [2]` (giao diện chỉ hiểu [n]); số lạ trong danh sách bị bỏ.
func ExtractCitations(text string, hits []rag.Hit) (string, []Citation) {
	cites := []Citation{}
	seen := map[int]bool{}
	out := reMarker.ReplaceAllStringFunc(text, func(m string) string {
		lead := ""
		if strings.HasPrefix(m, " ") || strings.HasPrefix(m, "\n") || strings.HasPrefix(m, "\t") {
			lead = m[:1]
		}
		var kept []string
		for _, n := range markerNums(reMarker.FindStringSubmatch(m)[1]) {
			if n < 1 || n > len(hits) {
				continue
			}
			if !seen[n] {
				seen[n] = true
				cites = append(cites, CitationOf(n, hits[n-1]))
			}
			kept = append(kept, "["+strconv.Itoa(n)+"]")
		}
		if len(kept) == 0 {
			return ""
		}
		return lead + strings.Join(kept, " ")
	})
	return out, cites
}

// CitationOf dựng mục trích dẫn thứ n từ một đoạn.
func CitationOf(n int, h rag.Hit) Citation {
	return Citation{N: n, DocumentID: h.DocumentID, Title: h.Title, PageNo: h.PageNo, Snippet: Cut(h.Text, 200), Knowledge: h.Knowledge, ThreadID: h.ThreadID}
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
