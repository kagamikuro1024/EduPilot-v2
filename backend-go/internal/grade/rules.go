package grade

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/shopspring/decimal"
)

// RuleKind trùng enum `grade_rule_kind`.
type RuleKind string

const (
	KindBonus        RuleKind = "BONUS"
	KindPenalty      RuleKind = "PENALTY"
	KindRounding     RuleKind = "ROUNDING"
	KindFail         RuleKind = "FAIL_CONDITION"
	KindLetterScale  RuleKind = "LETTER_SCALE"
	KindMissingScore RuleKind = "MISSING_SCORE"
)

// RuleRow là một dòng `grade_rules` ở dạng thuần (không phụ thuộc store) để dựng Scheme.
type RuleRow struct {
	Kind         RuleKind
	Applicable   bool
	Params       json.RawMessage
	SourceQuote  string
	SourcePage   int
	FilledByUser bool
}

// ParamError: tham số của một luật sai (thiếu, lạ, sai kiểu, ngoài miền). Missing = khoá bắt buộc không có.
// Không có giá trị mặc định thầm lặng (AC6): thiếu / lạ là lỗi.
type ParamError struct {
	Kind    RuleKind
	Field   string
	Missing bool
	Msg     string
}

func (e *ParamError) Error() string { return fmt.Sprintf("%s.%s: %s", e.Kind, e.Field, e.Msg) }

func perr(k RuleKind, field, msg string) *ParamError {
	return &ParamError{Kind: k, Field: field, Msg: msg}
}
func pmiss(k RuleKind, field string) *ParamError {
	return &ParamError{Kind: k, Field: field, Missing: true, Msg: "thiếu"}
}

var numRe = regexp.MustCompile(`^\d+(\.\d+)?$`) //nolint:gochecknoglobals // biểu thức chính quy bất biến

// parseNum đọc số decimal dạng chuỗi `^\d+(\.\d+)?$` (số điểm / ngưỡng LUÔN là chuỗi; không số JSON).
func parseNum(k RuleKind, field string, raw *string) (decimal.Decimal, *ParamError) {
	if raw == nil {
		return decimal.Zero, pmiss(k, field)
	}
	if !numRe.MatchString(*raw) {
		return decimal.Zero, perr(k, field, "phải là số decimal dạng chuỗi, không âm")
	}
	d, err := decimal.NewFromString(*raw)
	if err != nil {
		return decimal.Zero, perr(k, field, "không đọc được số")
	}
	return d, nil
}

// strictDecode giải mã JSON object, từ chối khoá lạ và dữ liệu thừa.
func strictDecode(k RuleKind, raw json.RawMessage, dst any) *ParamError {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return perr(k, "params", "tham số lạ hoặc sai kiểu: "+err.Error())
	}
	if dec.More() {
		return perr(k, "params", "dữ liệu thừa sau JSON")
	}
	return nil
}

// ParseBonus: {source:"PARTICIPATION", per_speak:"0.25", cap:"1.0"}.
func ParseBonus(raw json.RawMessage) (BonusRule, *ParamError) {
	var p struct {
		Source   *string `json:"source"`
		PerSpeak *string `json:"per_speak"`
		Cap      *string `json:"cap"`
	}
	if e := strictDecode(KindBonus, raw, &p); e != nil {
		return BonusRule{}, e
	}
	if p.Source == nil {
		return BonusRule{}, pmiss(KindBonus, "source")
	}
	if *p.Source != "PARTICIPATION" {
		return BonusRule{}, perr(KindBonus, "source", "chỉ nhận PARTICIPATION")
	}
	per, e := parseNum(KindBonus, "per_speak", p.PerSpeak)
	if e != nil {
		return BonusRule{}, e
	}
	cp, e := parseNum(KindBonus, "cap", p.Cap)
	if e != nil {
		return BonusRule{}, e
	}
	return BonusRule{Applicable: true, PerSpeak: per, Cap: cp}, nil
}

// ParsePenalty: {source:"ABSENCE", free:2, per:"0.5"}.
func ParsePenalty(raw json.RawMessage) (PenaltyRule, *ParamError) {
	var p struct {
		Source *string `json:"source"`
		Free   *int    `json:"free"`
		Per    *string `json:"per"`
	}
	if e := strictDecode(KindPenalty, raw, &p); e != nil {
		return PenaltyRule{}, e
	}
	if p.Source == nil {
		return PenaltyRule{}, pmiss(KindPenalty, "source")
	}
	if *p.Source != "ABSENCE" {
		return PenaltyRule{}, perr(KindPenalty, "source", "chỉ nhận ABSENCE")
	}
	if p.Free == nil {
		return PenaltyRule{}, pmiss(KindPenalty, "free")
	}
	if *p.Free < 0 || *p.Free > 1000 {
		return PenaltyRule{}, perr(KindPenalty, "free", "số buổi miễn phải nguyên, không âm")
	}
	per, e := parseNum(KindPenalty, "per", p.Per)
	if e != nil {
		return PenaltyRule{}, e
	}
	return PenaltyRule{Applicable: true, Free: *p.Free, Per: per}, nil
}

// roundingSteps là tập bước làm tròn cho phép (SRS 4.2.4).
var roundingSteps = []decimal.Decimal{decimal.RequireFromString("0.01"), decimal.RequireFromString("0.1"), decimal.RequireFromString("0.25"), decimal.RequireFromString("0.5"), decimal.NewFromInt(1)} //nolint:gochecknoglobals // tập bất biến

// ParseRounding: {step:"0.5", mode:"HALF_UP"}.
func ParseRounding(raw json.RawMessage) (RoundingRule, *ParamError) {
	var p struct {
		Step *string `json:"step"`
		Mode *string `json:"mode"`
	}
	if e := strictDecode(KindRounding, raw, &p); e != nil {
		return RoundingRule{}, e
	}
	if p.Mode == nil {
		return RoundingRule{}, pmiss(KindRounding, "mode")
	}
	if *p.Mode != "HALF_UP" {
		return RoundingRule{}, perr(KindRounding, "mode", "chỉ nhận HALF_UP")
	}
	step, e := parseNum(KindRounding, "step", p.Step)
	if e != nil {
		return RoundingRule{}, e
	}
	for _, ok := range roundingSteps {
		if step.Equal(ok) {
			return RoundingRule{Step: step}, nil
		}
	}
	return RoundingRule{}, perr(KindRounding, "step", "bước phải thuộc {0,01; 0,1; 0,25; 0,5; 1}")
}

// ParseLetterScale: {bands:[{min:"9.0", letter:"A+"}, …, {min:"0", letter:"F"}]}; giảm dần theo min, mức cuối min = 0.
func ParseLetterScale(raw json.RawMessage) (LetterScale, *ParamError) {
	var p struct {
		Bands *[]struct {
			Min    *string `json:"min"`
			Letter *string `json:"letter"`
		} `json:"bands"`
	}
	if e := strictDecode(KindLetterScale, raw, &p); e != nil {
		return LetterScale{}, e
	}
	if p.Bands == nil {
		return LetterScale{}, pmiss(KindLetterScale, "bands")
	}
	if len(*p.Bands) == 0 || len(*p.Bands) > 20 {
		return LetterScale{}, perr(KindLetterScale, "bands", "cần 1–20 mức")
	}
	out := LetterScale{}
	seen := map[string]bool{}
	for i, b := range *p.Bands {
		m, e := parseNum(KindLetterScale, "bands["+strconv.Itoa(i)+"].min", b.Min)
		if e != nil {
			return LetterScale{}, e
		}
		if b.Letter == nil || *b.Letter == "" || len(*b.Letter) > 4 {
			return LetterScale{}, perr(KindLetterScale, "bands["+strconv.Itoa(i)+"].letter", "điểm chữ 1–4 ký tự")
		}
		if seen[*b.Letter] {
			return LetterScale{}, perr(KindLetterScale, "bands["+strconv.Itoa(i)+"].letter", "điểm chữ trùng")
		}
		seen[*b.Letter] = true
		if m.GreaterThan(MaxScale) {
			return LetterScale{}, perr(KindLetterScale, "bands["+strconv.Itoa(i)+"].min", "vượt thang")
		}
		if i > 0 && !m.LessThan(out.Bands[i-1].Min) {
			return LetterScale{}, perr(KindLetterScale, "bands["+strconv.Itoa(i)+"].min", "phải giảm dần")
		}
		out.Bands = append(out.Bands, Band{Min: m, Letter: *b.Letter})
	}
	if !out.Bands[len(out.Bands)-1].Min.IsZero() {
		return LetterScale{}, perr(KindLetterScale, "bands", "mức cuối phải có min = 0")
	}
	return out, nil
}

// ParseFail: {kind:"FINAL_BELOW", value:"3.0"} hoặc {kind:"ABSENT_AT_LEAST", value:"7"}. Giữ nguyên chữ của value để in lại ở `verdict`.
func ParseFail(raw json.RawMessage) (FailCondition, string, *ParamError) {
	var p struct {
		Kind  *string `json:"kind"`
		Value *string `json:"value"`
	}
	if e := strictDecode(KindFail, raw, &p); e != nil {
		return FailCondition{}, "", e
	}
	if p.Kind == nil {
		return FailCondition{}, "", pmiss(KindFail, "kind")
	}
	v, e := parseNum(KindFail, "value", p.Value)
	if e != nil {
		return FailCondition{}, "", e
	}
	switch FailKind(*p.Kind) {
	case FailFinalBelow:
		if v.Sign() <= 0 || v.GreaterThan(MaxScale) {
			return FailCondition{}, "", perr(KindFail, "value", "ngưỡng điểm phải trong (0; 10]")
		}
	case FailAbsentAtLeast:
		if v.Sign() <= 0 || !v.Equal(v.Truncate(0)) || v.GreaterThan(decimal.NewFromInt(1000)) {
			return FailCondition{}, "", perr(KindFail, "value", "số buổi phải nguyên dương")
		}
	default:
		return FailCondition{}, "", perr(KindFail, "kind", "chỉ nhận FINAL_BELOW hoặc ABSENT_AT_LEAST")
	}
	return FailCondition{Kind: FailKind(*p.Kind), Value: v, Text: *p.Value}, *p.Value, nil
}

// ParseMissingScore: {zero_components:["weekly"]}.
func ParseMissingScore(raw json.RawMessage) (MissingScoreRule, *ParamError) {
	var p struct {
		Zero *[]string `json:"zero_components"`
	}
	if e := strictDecode(KindMissingScore, raw, &p); e != nil {
		return MissingScoreRule{}, e
	}
	if p.Zero == nil {
		return MissingScoreRule{}, pmiss(KindMissingScore, "zero_components")
	}
	return MissingScoreRule{ZeroComponents: *p.Zero}, nil
}

// BuildRules dựng Rules từ các dòng `grade_rules`. Luật `applicable = false` không đọc tham số (giảng viên bấm "Không áp dụng").
// Lỗi tham số trả về *ParamError (gom ở errors.Join); luật trùng (trừ FAIL_CONDITION) là lỗi.
func BuildRules(rows []RuleRow) (Rules, error) {
	var r Rules
	var errs []error
	seen := map[RuleKind]bool{}
	for _, row := range rows {
		if row.Kind != KindFail {
			if seen[row.Kind] {
				errs = append(errs, perr(row.Kind, "kind", "luật trùng"))
				continue
			}
			seen[row.Kind] = true
		}
		meta := RuleMeta{Quote: row.SourceQuote, Page: row.SourcePage, FilledByUser: row.FilledByUser}
		switch row.Kind {
		case KindBonus:
			if !row.Applicable {
				r.Bonus = &BonusRule{RuleMeta: meta}
				continue
			}
			b, e := ParseBonus(row.Params)
			if e != nil {
				errs = append(errs, e)
				continue
			}
			b.RuleMeta = meta
			r.Bonus = &b
		case KindPenalty:
			if !row.Applicable {
				r.Penalty = &PenaltyRule{RuleMeta: meta}
				continue
			}
			p, e := ParsePenalty(row.Params)
			if e != nil {
				errs = append(errs, e)
				continue
			}
			p.RuleMeta = meta
			r.Penalty = &p
		case KindRounding:
			x, e := ParseRounding(row.Params)
			if e != nil {
				errs = append(errs, e)
				continue
			}
			x.RuleMeta = meta
			r.Rounding = &x
		case KindLetterScale:
			x, e := ParseLetterScale(row.Params)
			if e != nil {
				errs = append(errs, e)
				continue
			}
			x.RuleMeta = meta
			r.Letters = &x
		case KindFail:
			f, _, e := ParseFail(row.Params)
			if e != nil {
				errs = append(errs, e)
				continue
			}
			f.RuleMeta = meta
			r.Fails = append(r.Fails, f)
		case KindMissingScore:
			x, e := ParseMissingScore(row.Params)
			if e != nil {
				errs = append(errs, e)
				continue
			}
			x.RuleMeta = meta
			r.MissingScore = &x
		default:
			errs = append(errs, perr(row.Kind, "kind", "loại luật lạ"))
		}
	}
	return r, errors.Join(errs...)
}
