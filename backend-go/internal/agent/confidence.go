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
	NoFormula      bool // GRADE_FORMULA mà lớp chưa có công thức điểm chính thức
	LowConfidence  bool
	Degraded       bool
	MaskedCount    int
}

// Nguồn gốc ticket tự mở (khớp enum `ticket_origin`).
const (
	OriginAutoLowConfidence = "AUTO_LOW_CONFIDENCE"
	OriginAutoNoFormula     = "AUTO_NO_FORMULA"
)

// ShouldEscalate cho biết câu trả lời có tự mở ticket không (SRS escalation 4.2.4). Thuần, không I/O; origin rỗng khi false.
func (m ResponseMetadata) ShouldEscalate(intent Intent) (bool, string) {
	switch {
	case intent == IntentCrisis || intent == IntentOtherPerson || intent == IntentSmalltalk:
		return false, ""
	case m.Degraded: // nút vẫn hiện: sinh viên tự nhờ
		return false, ""
	case intent == IntentGradeFormula && m.NoFormula:
		return true, OriginAutoNoFormula
	case m.LowConfidence:
		return true, OriginAutoLowConfidence
	}
	return false, ""
}

// refusalCore: lõi của câu mẫu ReplyNoContext, đã bỏ dấu + chữ thường. Câu trả lời của LLM chứa lõi này (ở bất kỳ đâu, không phân biệt dấu) là LỜI TỪ CHỐI: ngữ cảnh
// không trả lời được câu hỏi (P-11-18, D-R-07). SystemPrompt bảo mô hình mở đầu đúng bằng câu này khi không đủ ngữ cảnh.
var refusalCore = auth.Fold(strings.ToLower("chưa tìm thấy nội dung này trong tài liệu của lớp")) //nolint:gochecknoglobals // hằng dẫn xuất từ câu mẫu

// IsRefusal cho biết văn bản là lời từ chối "chưa tìm thấy trong tài liệu" (thuần, không LLM).
func IsRefusal(s string) bool { return strings.Contains(auth.Fold(strings.ToLower(s)), refusalCore) }

// hasCitation: câu trả lời có ít nhất một dấu [n] trỏ tới một đoạn trong hits (n ∈ [1, nHits]); [n] lạ không tính (như ExtractCitations).
func hasCitation(answer string, nHits int) bool {
	for _, m := range reMarker.FindAllStringSubmatch(answer, -1) {
		for _, n := range markerNums(m[1]) {
			if n >= 1 && n <= nHits {
				return true
			}
		}
	}
	return false
}

var (
	reCiteMark = regexp.MustCompile(`\[\d{1,3}(?:\s*,\s*\d{1,3})*\]`)
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

// Groundedness là tỷ lệ "nhận định" (câu ≥ 4 từ nội dung, bỏ stop-words) có căn cứ. Một nhận định có căn cứ khi:
//
//	(a) ≥ 40 % từ nội dung của nó có trong hợp các đoạn đưa vào lời nhắc (cùng ngôn ngữ), HOẶC
//	(b) nó mang dấu [n] hợp lệ và chia sẻ ≥ 1 mỏ neo (xem `anchors`) với CHÍNH đoạn n — đường cho câu trả lời tiếng Việt dựa trên nguồn tiếng Anh, nơi (a) ≈ 0
//	    dù trả lời đúng (P-11-21). (b) tắt khi câu trả lời tự thú thiếu thông tin (`hedged`): một câu «chưa đủ thông tin» vẫn có thể dẫn [n] và trùng thuật ngữ.
//
// Không câu nào đủ 4 từ nội dung (ví dụ "Có.") → 0: không có căn cứ để chống đỡ (nghiêng về low_confidence, #11). Thuần, không LLM.
func Groundedness(answer string, hits []rag.Hit) decimal.Decimal {
	ctx := map[string]bool{}
	chunk := make([]map[string]bool, len(hits)) // từ thô (chữ thường) của từng đoạn, cho mỏ neo (b)
	for i, h := range hits {
		head := ""
		if h.Heading != nil {
			head = *h.Heading
		}
		for _, w := range contentWords(h.Title + " " + head + " " + h.Text) {
			ctx[w] = true
		}
		chunk[i] = map[string]bool{}
		for _, w := range reWord.FindAllString(strings.ToLower(h.Title+" "+head+" "+h.Text), -1) {
			chunk[i][w] = true
		}
	}
	allowAnchor := !hedged(answer)
	claims, ok := 0, 0
	for _, raw := range reSentence.Split(answer, -1) {
		s := reCiteMark.ReplaceAllString(raw, " ")
		ws := contentWords(s)
		if len(ws) < groundMinWords || IsRefusal(s) { // câu từ chối không phải một nhận định cần căn cứ (P-11-18 c)
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
			continue
		}
		if allowAnchor && anchoredToCited(raw, s, chunk) {
			ok++
		}
	}
	if claims == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(int64(ok)).Div(decimal.NewFromInt(int64(claims))).Round(3)
}

// anchoredToCited: câu `raw` (có dấu [n]) và bản `plain` (đã bỏ dấu) — có [n] ∈ [1, len(chunk)] mà đoạn n chứa một mỏ neo của câu.
func anchoredToCited(raw, plain string, chunk []map[string]bool) bool {
	an := anchors(plain)
	if len(an) == 0 {
		return false
	}
	for _, m := range reMarker.FindAllStringSubmatch(raw, -1) {
		for _, n := range markerNums(m[1]) {
			if n < 1 || n > len(chunk) {
				continue
			}
			for _, a := range an {
				if chunk[n-1][a] {
					return true
				}
			}
		}
	}
	return false
}

// Mỏ neo: token không phụ thuộc ngôn ngữ nên còn nguyên khi câu trả lời tiếng Việt dịch nguồn tiếng Anh: số (≥ 2 chữ số), chữ + số trộn (wpa3, ipv6; ≥ 3 ký tự),
// từ Latin chỉ gồm chữ không dấu ≥ 4 ký tự (exploits, internet, trojan). Loại âm tiết tiếng Việt thanh ngang không dấu (trong, theo, khi, cho, sau, thanh, chung…):
// chúng cũng là ASCII thuần nhưng không mang nghĩa kỹ thuật nào, và một câu Việt bất kỳ đầy chúng.
var (
	reASCIIWord = regexp.MustCompile(`^[a-z0-9]+$`)
	reDigits    = regexp.MustCompile(`^[0-9]+$`)
	reHasDigit  = regexp.MustCompile(`[0-9]`)
	reHasLetter = regexp.MustCompile(`[a-z]`)
	// reVNSyllable: [phụ âm đầu] + vần chỉ gồm nguyên âm không mũ/móc + [phụ âm cuối]: dạng của mọi âm tiết tiếng Việt viết không dấu.
	reVNSyllable = regexp.MustCompile(`^(?:ngh|ng|gh|gi|kh|nh|ph|th|tr|ch|qu|[bcdghklmnpqrstvx])?(?:oai|oay|uay|uai|oa|oe|uy|ai|ao|au|ay|eo|ia|iu|oi|ua|ui|a|e|i|o|u|y)(?:ch|ng|nh|c|m|n|p|t)?$`)
)

func anchors(s string) []string {
	var out []string
	for _, w := range reWord.FindAllString(strings.ToLower(s), -1) {
		switch {
		case !reASCIIWord.MatchString(w):
		case reDigits.MatchString(w):
			if len(w) >= 2 {
				out = append(out, w)
			}
		case reHasDigit.MatchString(w) && reHasLetter.MatchString(w):
			if len(w) >= 3 {
				out = append(out, w)
			}
		case len(w) >= 4 && !reVNSyllable.MatchString(w):
			out = append(out, w)
		}
	}
	return out
}

// hedgeCores: cụm tự thú thiếu thông tin (đã bỏ dấu, chữ thường) mà mô hình dùng khi KHÔNG theo câu mẫu ReplyNoContext («Chào bạn, hiện tại ngữ cảnh được cung cấp chưa đủ thông tin…»).
var hedgeCores = []string{"chua du thong tin", "khong du thong tin", "chua co thong tin", "khong co thong tin", "khong tim thay", "chua tim thay", "khong duoc de cap", "chua duoc de cap"} //nolint:gochecknoglobals // bảng tra cứu bất biến

func hedged(answer string) bool {
	f := auth.Fold(strings.ToLower(answer))
	for _, c := range hedgeCores {
		if strings.Contains(f, c) {
			return true
		}
	}
	return false
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
	m := ResponseMetadata{NoContext: o.NoContext, NoFormula: o.NoFormula, Degraded: degraded, MaskedCount: masked}
	one := decimal.NewFromInt(1)
	switch {
	case o.NoContext:
		m.RetrievalScore, m.Groundedness = Retrieval(BestCosine(o.Hits)), decimal.Zero
		m.Confidence = decimal.NullDecimal{Decimal: Score(m.RetrievalScore, m.Groundedness), Valid: true}
		m.LowConfidence = true
		return m
	case o.Canned != "":
		return m
	case o.ToolBacked() && IsRefusal(answer):
		// D-R-09: tool trả khối dữ liệu nhưng mô hình mở bằng câu «chưa tìm thấy…» (vd. LIBRARY_SEARCH trả thẻ tài liệu không liên quan): dữ liệu không trả lời được câu hỏi ⇒ chưa chắc,
		// KHÔNG phải 1,000. Cùng nguyên tắc P-11-18 (b) cho đường CourseQA.
		m.RetrievalScore, m.Groundedness = decimal.Zero, decimal.Zero
		m.Confidence = decimal.NullDecimal{Decimal: decimal.Zero.Round(3), Valid: true}
		m.LowConfidence = true
		return m
	case o.ToolBacked():
		m.RetrievalScore, m.Groundedness = one, one
		m.Confidence = decimal.NullDecimal{Decimal: one.Round(3), Valid: true}
	case o.Plan == IntentCourseQA && len(o.Hits) > 0:
		refused := IsRefusal(answer)
		m.RetrievalScore, m.Groundedness = Retrieval(BestCosine(o.Hits)), Groundedness(answer, o.Hits)
		if refused || !hasCitation(answer, len(o.Hits)) { // không dẫn nguồn nào = không có căn cứ (P-11-18 d); từ chối = chưa trả lời được (b)
			m.Groundedness = decimal.Zero
		}
		m.Confidence = decimal.NullDecimal{Decimal: Score(m.RetrievalScore, m.Groundedness), Valid: true}
		if refused { // lời từ chối luôn là "chưa chắc", kể cả khi ngưỡng của lớp thấp
			m.LowConfidence = true
			return m
		}
	default:
		return m
	}
	m.LowConfidence = m.Confidence.Decimal.LessThan(threshold)
	return m
}
