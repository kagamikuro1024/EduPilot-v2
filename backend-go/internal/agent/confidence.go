package agent

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/rag"
)

// Hằng số độ tin cậy (SRS 4.8, bản tạm — hiệu chỉnh ở E2, Q9). Không có lời gọi LLM nào trong tính toán.
const (
	RagSimCeil      = 0.65 // RAG_SIM_CEIL
	groundMinWords  = 4    // câu cần ≥ 4 từ nội dung mới được tính là một "nhận định"
	groundSupported = 0.4  // ≥ 40 % từ nội dung của câu phải có trong ngữ cảnh
)

// ResponseMetadata là siêu dữ liệu của một câu trả lời. Chỉ `LowConfidence` (và Degraded, MaskedCount) đi tới sinh viên; số còn lại chỉ Staff đọc được ở Threads.
type ResponseMetadata struct {
	Confidence     decimal.NullDecimal // NULL = không áp dụng (câu mẫu, chào hỏi)
	RetrievalScore decimal.Decimal
	Groundedness   decimal.Decimal
	NoContext      bool
	LowConfidence  bool
	Degraded       bool
	MaskedCount    int
}

var (
	reCiteMark = regexp.MustCompile(`\[\d+\]`)
	reSentence = regexp.MustCompile(`[.!?;\n]+(?:\s|$)`)
	reWord     = regexp.MustCompile(`[\p{L}\p{M}\p{N}]+`)
)

// stopWords là từ chức năng (viết đúng dấu: "thì" là từ chức năng còn "thi" là nội dung nên không thể so sau khi bỏ dấu).
var stopWords = func() map[string]bool { //nolint:gochecknoglobals // bảng tra cứu bất biến, dựng một lần
	m := map[string]bool{}
	for _, w := range strings.Fields(`là và của có không được trong cho các một những này đó để khi với thì sẽ đã đang như từ đến về ở ra vào theo hay hoặc nếu cũng rất nhiều
		thêm nữa mà ấy nào gì sao thế bị trên dưới sau trước cần phải nên tại bởi vì quá lại lên xuống mỗi từng đây kia thôi rồi còn chỉ nhé`) {
		m[w] = true
	}
	return m
}()

// contentWords: từ nội dung (không phải stop-word, ≥ 2 ký tự), đã vn_fold để so khớp không phân biệt dấu.
func contentWords(s string) []string {
	var out []string
	for _, w := range reWord.FindAllString(strings.ToLower(s), -1) {
		if utf8.RuneCountInString(w) > 1 && !stopWords[w] {
			out = append(out, auth.Fold(w))
		}
	}
	return out
}

// Retrieval chuẩn hoá cos_top1 giữa sàn và trần: clamp((cos − floor) / (ceil − floor), 0, 1), 3 chữ số.
func Retrieval(cosTop1 float64) decimal.Decimal {
	floor, ceil := decimal.NewFromFloat(RagSimFloor), decimal.NewFromFloat(RagSimCeil)
	r := decimal.NewFromFloat(cosTop1).Sub(floor).Div(ceil.Sub(floor))
	return decimal.Min(decimal.NewFromInt(1), decimal.Max(decimal.Zero, r)).Round(3)
}

// Groundedness là tỷ lệ "nhận định" (câu ≥ 4 từ nội dung, bỏ stop-words) có ≥ 40 % từ xuất hiện trong hợp các đoạn đưa vào lời nhắc.
// Không câu nào đủ 4 từ nội dung (ví dụ "Có.") → 0: không có căn cứ để chống đỡ (nghiêng về low_confidence, #11).
func Groundedness(answer string, hits []rag.Hit) decimal.Decimal {
	ctx := map[string]bool{}
	for _, h := range hits {
		head := ""
		if h.Heading != nil {
			head = *h.Heading
		}
		for _, w := range contentWords(h.Title + " " + head + " " + h.Text) {
			ctx[w] = true
		}
	}
	claims, ok := 0, 0
	for _, s := range reSentence.Split(reCiteMark.ReplaceAllString(answer, " "), -1) {
		ws := contentWords(s)
		if len(ws) < groundMinWords {
			continue
		}
		claims++
		in := 0
		for _, w := range ws {
			if ctx[w] {
				in++
			}
		}
		if float64(in) >= groundSupported*float64(len(ws)) {
			ok++
		}
	}
	if claims == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(int64(ok)).Div(decimal.NewFromInt(int64(claims))).Round(3)
}

// Score = round(0,6·retr + 0,4·ground, 3).
func Score(retr, ground decimal.Decimal) decimal.Decimal {
	return decimal.New(6, -1).Mul(retr).Add(decimal.New(4, -1).Mul(ground)).Round(3) // 0,6·retr + 0,4·ground
}

// BestCosine là cos_top1 = max(cosine) của các đoạn (không theo thứ hạng RRF).
func BestCosine(hits []rag.Hit) float64 {
	best := 0.0
	for _, h := range hits {
		best = max(best, h.Cosine)
	}
	return best
}

// ToolBacked cho biết intent đã trả dữ liệu từ tool Go (độ tin cậy 1,000: số liệu lấy từ cơ sở dữ liệu, không phải suy đoán).
func (o Outcome) ToolBacked() bool {
	switch o.Plan {
	case IntentGradeFormula, IntentAttendance, IntentParticipation, IntentGrade, IntentWhatIf, IntentExamSchedule, IntentUpcoming, IntentLibrary:
		return len(o.Blocks) > 0 || len(o.Sources) > 0
	}
	return false
}

// Metadata tính siêu dữ liệu của câu trả lời đã sinh xong. Thuần tính toán (không LLM). threshold = courses.escalation_threshold.
// `no_context` luôn kéo theo low_confidence; câu mẫu / chào hỏi không có độ tin cậy (NULL) và không bị coi là "chưa chắc".
func (o Outcome) Metadata(answer string, threshold decimal.Decimal, degraded bool, masked int) ResponseMetadata {
	m := ResponseMetadata{NoContext: o.NoContext, Degraded: degraded, MaskedCount: masked}
	one := decimal.NewFromInt(1)
	switch {
	case o.NoContext:
		m.RetrievalScore, m.Groundedness = Retrieval(BestCosine(o.Hits)), decimal.Zero
		m.Confidence = decimal.NullDecimal{Decimal: Score(m.RetrievalScore, m.Groundedness), Valid: true}
		m.LowConfidence = true
		return m
	case o.Canned != "":
		return m
	case o.ToolBacked():
		m.RetrievalScore, m.Groundedness = one, one
		m.Confidence = decimal.NullDecimal{Decimal: one.Round(3), Valid: true}
	case o.Plan == IntentCourseQA && len(o.Hits) > 0:
		m.RetrievalScore, m.Groundedness = Retrieval(BestCosine(o.Hits)), Groundedness(answer, o.Hits)
		m.Confidence = decimal.NullDecimal{Decimal: Score(m.RetrievalScore, m.Groundedness), Valid: true}
	default:
		return m
	}
	m.LowConfidence = m.Confidence.Decimal.LessThan(threshold)
	return m
}
