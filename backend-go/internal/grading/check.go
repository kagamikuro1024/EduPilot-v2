package grading

import (
	"encoding/json"
	"slices"
	"sort"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/textfold"
)

// Cờ của kết quả chấm (SRS FEAT-assignments 4.6.8). Mười một giá trị này là toàn bộ giá trị cho phép của `grading_results.flags` (CHECK ở 00017).
const (
	FlagEmptySubmission    = "EMPTY_SUBMISSION"
	FlagOffTopic           = "OFF_TOPIC"
	FlagInjectionSuspected = "INJECTION_SUSPECTED"
	FlagNeedsReview        = "NEEDS_REVIEW"
	FlagTotalMismatch      = "TOTAL_MISMATCH"
	FlagEvidenceNotFound   = "EVIDENCE_NOT_FOUND"
	FlagScoreAdjusted      = "SCORE_ADJUSTED"
	FlagLowConfidence      = "LOW_CONFIDENCE"
	FlagExtractFailed      = "EXTRACT_FAILED"
	FlagLateAfterCutoff    = "LATE_AFTER_CUTOFF"
	FlagAnswerEcho         = "ANSWER_ECHO"
)

// ErrOutputInvalid là mã lỗi vĩnh viễn khi đầu ra của LLM sai hình (thiếu / lạ / trùng id tiêu chí, điểm không phải số).
const ErrOutputInvalid = "GRADING_OUTPUT_INVALID"

// scoreStep là bước điểm tiêu chí (0,25).
var scoreStep = decimal.RequireFromString("0.25") //nolint:gochecknoglobals // hằng

// RubricItem là một tiêu chí của rubric (`assignments.rubric`). `Max` là chuỗi decimal ở JSON; Check đọc qua decimal.
type RubricItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MaxScore string `json:"max_score"`
	Guidance string `json:"guidance"`
}

// Raw* là đầu ra của llm.Structured sau khi giải mã. Điểm và tổng là CHUỖI; `confidence` đọc dạng json.Number rồi so bằng decimal (không float).
type (
	RawCriterion struct {
		ID             string   `json:"id"`
		Score          string   `json:"score"`
		EvidenceQuotes []string `json:"evidence_quotes"`
		Comment        string   `json:"comment"`
	}
	RawOutput struct {
		Criteria        []RawCriterion `json:"criteria"`
		OverallFeedback string         `json:"overall_feedback"`
		Confidence      json.Number    `json:"confidence"`
		Flags           []string       `json:"flags"`
		Total           *string        `json:"total"`
	}
)

// Evidence là một trích dẫn có thật trong bài; Start/End theo UTF-16 code unit của `extracted_text` (màn duyệt dùng thẳng).
type Evidence struct {
	Quote string `json:"quote"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// Dropped là trích dẫn bị loại vì không có trong bài.
type Dropped struct {
	Criterion string `json:"criterion"`
	Quote     string `json:"quote"`
}

// Adjusted ghi một điểm bị kẹp / làm tròn.
type Adjusted struct {
	Criterion string `json:"criterion"`
	From      string `json:"from"`
	To        string `json:"to"`
}

// PassResult là kết quả kiểm của MỘT lượt. Valid = false: `Err` = ErrOutputInvalid, mọi trường khác bỏ.
type PassResult struct {
	Valid    bool
	Err      string
	Scores   map[string]decimal.Decimal
	Comments map[string]string
	Evidence map[string][]Evidence
	Total    decimal.Decimal
	Flags    []string
	Dropped  []Dropped
	Adjusted []Adjusted
	// Reported là tổng do LLM báo (nil nếu không báo); chỉ để lưu `llm_reported_total`.
	Reported *decimal.Decimal
	Feedback string
	// Confidence là độ chắc chắn LLM báo (0–1).
	Confidence decimal.Decimal
	// LLMInjection: LLM báo INJECTION_SUSPECTED (chỉ có khi schema cho phép, tức cổng INJECTION_FLAG có hiệu lực). Không vào Flags / Merge: Run quyết OR với các tầng khác.
	LLMInjection bool
}

func maxOf(rubric []RubricItem, id string) (decimal.Decimal, bool) {
	for _, r := range rubric {
		if r.ID == id {
			m, err := decimal.NewFromString(r.MaxScore)
			return m, err == nil
		}
	}
	return decimal.Zero, false
}

func uniqSorted(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := slices.Clone(in)
	sort.Strings(out)
	return slices.Compact(out)
}

// Check kiểm bằng code một lượt chấm (SRS 4.6.6), không phụ thuộc LLM. `text` là extracted_text (đầu ra đã khôi phục danh tính). `lowConf` là ngưỡng `GRADING_LOW_CONFIDENCE`.
func Check(out RawOutput, text string, rubric []RubricItem, lowConf decimal.Decimal) PassResult {
	bad := PassResult{Err: ErrOutputInvalid}
	if len(out.Criteria) != len(rubric) {
		return bad
	}
	seen := map[string]bool{}
	for _, c := range out.Criteria {
		if _, ok := maxOf(rubric, c.ID); !ok || seen[c.ID] {
			return bad
		}
		seen[c.ID] = true
	}
	res := PassResult{Valid: true, Scores: map[string]decimal.Decimal{}, Comments: map[string]string{}, Evidence: map[string][]Evidence{}, Feedback: out.OverallFeedback}
	var flags []string
	for _, c := range out.Criteria {
		raw, err := decimal.NewFromString(c.Score)
		if err != nil {
			return bad
		}
		max, _ := maxOf(rubric, c.ID)
		s := decimal.Min(decimal.Max(raw, decimal.Zero), max)
		s = decimal.Min(s.Div(scoreStep).Round(0).Mul(scoreStep), max) // nửa lên về bội 0,25 (s ≥ 0)
		if !s.Equal(raw) {
			flags = append(flags, FlagScoreAdjusted)
			res.Adjusted = append(res.Adjusted, Adjusted{Criterion: c.ID, From: raw.StringFixed(2), To: s.StringFixed(2)})
		}
		res.Scores[c.ID] = s
		res.Comments[c.ID] = c.Comment
		res.Total = res.Total.Add(s)
		ev := []Evidence{}
		for _, q := range c.EvidenceQuotes {
			if st, en, ok := textfold.Index(text, q); ok {
				ev = append(ev, Evidence{Quote: q, Start: st, End: en})
			} else {
				res.Dropped = append(res.Dropped, Dropped{Criterion: c.ID, Quote: q})
				flags = append(flags, FlagEvidenceNotFound)
			}
		}
		res.Evidence[c.ID] = ev
	}
	if out.Total != nil {
		t, err := decimal.NewFromString(*out.Total)
		if err != nil {
			return bad
		}
		res.Reported = &t
		if !t.Equal(res.Total) {
			flags = append(flags, FlagTotalMismatch)
		}
	}
	conf, err := decimal.NewFromString(out.Confidence.String())
	if err != nil {
		return bad
	}
	res.Confidence = conf
	if conf.LessThan(lowConf) {
		flags = append(flags, FlagLowConfidence)
	}
	if slices.Contains(out.Flags, FlagOffTopic) { // LLM chỉ được báo lạc đề; mọi cờ khác do code
		flags = append(flags, FlagOffTopic)
	}
	res.LLMInjection = slices.Contains(out.Flags, FlagInjectionSuspected)
	res.Flags = uniqSorted(flags)
	return res
}

// Disagreement là tiêu chí lệch giữa hai lượt.
type Disagreement struct {
	Criterion string `json:"criterion"`
	Delta     string `json:"delta"`
}

// Merged là kết quả hợp nhất: điểm nháp = lượt 1; lượt 2 chỉ để phát hiện lệch (SRS 4.6.7).
type Merged struct {
	Pass1      PassResult
	Flags      []string
	Disagree   []Disagreement
	TotalDelta decimal.Decimal
	// Pass2OK = false: lượt 2 lỗi / hỏng (khi lượt 1 tốt) → giữ lượt 1, cờ NEEDS_REVIEW.
	Pass2OK bool
}

// Merge hợp nhất hai lượt. `p2` có thể không hợp lệ (Valid = false). `disagree` là ngưỡng `GRADING_DISAGREE`; lệch > ngưỡng (tổng hoặc một tiêu chí) → NEEDS_REVIEW.
// Cờ = hợp cờ của hai lượt (lượt hỏng không có cờ). `rubric` giữ thứ tự tiêu chí cho danh sách lệch.
func Merge(p1, p2 PassResult, rubric []RubricItem, disagree decimal.Decimal) Merged {
	m := Merged{Pass1: p1, Pass2OK: p2.Valid, Disagree: []Disagreement{}}
	flags := slices.Clone(p1.Flags)
	if !p2.Valid {
		m.Flags = uniqSorted(append(flags, FlagNeedsReview))
		return m
	}
	flags = append(flags, p2.Flags...)
	for _, r := range rubric {
		d := p1.Scores[r.ID].Sub(p2.Scores[r.ID]).Abs()
		if d.GreaterThan(disagree) {
			m.Disagree = append(m.Disagree, Disagreement{Criterion: r.ID, Delta: d.StringFixed(2)})
		}
	}
	m.TotalDelta = p1.Total.Sub(p2.Total).Abs()
	if len(m.Disagree) > 0 || m.TotalDelta.GreaterThan(disagree) {
		flags = append(flags, FlagNeedsReview)
	}
	m.Flags = uniqSorted(flags)
	return m
}

// AnswerEcho: trong các nhận xét có ≥ n từ liên tiếp (sau fold, `\w+`) trùng đáp án chuẩn → true (SRS 4.6.6 (7); chỉ nhân viên thấy).
func AnswerEcho(comments []string, reference string, n int) bool {
	ref := textfold.Words(reference)
	if n < 1 || len(ref) < n {
		return false
	}
	grams := map[string]bool{}
	for i := 0; i+n <= len(ref); i++ {
		grams[joinWords(ref[i:i+n])] = true
	}
	for _, c := range comments {
		w := textfold.Words(c)
		for i := 0; i+n <= len(w); i++ {
			if grams[joinWords(w[i:i+n])] {
				return true
			}
		}
	}
	return false
}

func joinWords(w []string) string {
	n := 0
	for _, s := range w {
		n += len(s) + 1
	}
	b := make([]byte, 0, n)
	for i, s := range w {
		if i > 0 {
			b = append(b, ' ')
		}
		b = append(b, s...)
	}
	return string(b)
}
