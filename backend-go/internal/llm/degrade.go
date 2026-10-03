package llm

import (
	"sort"
	"strconv"
	"strings"
)

// Câu cố định của đường suy giảm (SRS 4.3). Không chứa từ kỹ thuật (provider, fallback, trace, RAG, PII).
const (
	DegradedOpening     = "AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:"
	DegradedNoPassages  = "AI tạm thời không khả dụng. Câu hỏi của bạn đã được ghi lại, giảng viên sẽ xem."
	degradedMaxPassages = 3
	degradedMaxChars    = 600
)

// DegradedText dựng câu trả lời trích nguyên văn: câu mở đầu + ≤ 3 đoạn điểm cao nhất, mỗi đoạn «trích» — tên tài liệu, tr. N,
// cắt ≤ 600 ký tự. Không sinh gì.
func DegradedText(ps []Passage) string {
	if len(ps) == 0 {
		return DegradedNoPassages
	}
	sorted := append([]Passage(nil), ps...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })
	if len(sorted) > degradedMaxPassages {
		sorted = sorted[:degradedMaxPassages]
	}
	var b strings.Builder
	b.WriteString(DegradedOpening)
	for _, p := range sorted {
		text := strings.TrimSpace(p.Text)
		if r := []rune(text); len(r) > degradedMaxChars {
			text = string(r[:degradedMaxChars])
		}
		b.WriteString("\n\n«" + text + "»")
		if p.Source != "" {
			b.WriteString(" — " + p.Source)
			if p.Page > 0 {
				b.WriteString(", tr. " + strconv.Itoa(p.Page))
			}
		}
	}
	return b.String()
}

// degraded dựng Response suy giảm: không token, không chi phí.
func degraded(r Request) Response {
	return Response{Text: DegradedText(r.Passages), Degraded: true, Provider: "", Model: ""}
}
