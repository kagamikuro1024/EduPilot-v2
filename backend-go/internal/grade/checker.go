package grade

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// Mã `unclear_items` do bộ kiểm bằng code thêm (SRS 4.2.5). Bộ kiểm chạy sau khi nhận đầu ra LLM và MỖI lần giảng viên sửa bản nháp;
// mục chưa rõ tự hết khi dữ liệu thoả điều kiện (không có trạng thái "đã giải quyết" riêng).
const (
	CodeNoComponents       = "NO_COMPONENTS"
	CodeWeightSum          = "WEIGHT_SUM"
	CodeBonusMissing       = "BONUS_MISSING"
	CodeBonusCapMissing    = "BONUS_CAP_MISSING"
	CodeAbsenceRuleMissing = "ABSENCE_RULE_MISSING"
	CodeRoundingMissing    = "ROUNDING_MISSING"
	CodeLetterScaleMissing = "LETTER_SCALE_MISSING"
	CodeLetterScaleInvalid = "LETTER_SCALE_INVALID"
	CodeParamInvalid       = "PARAM_INVALID"
	CodeCitationNotFound   = "CITATION_NOT_FOUND"

	SourceCode = "CODE"
	SourceLLM  = "LLM"
)

// Unclear là một mục chưa rõ của bản nháp công thức.
type Unclear struct {
	Source  string         `json:"source"` // CODE | LLM
	Code    string         `json:"code,omitempty"`
	Target  string         `json:"target"` // "components", "components.<key>", "rules.<KIND>"
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// DraftComponent / DraftRule / Draft là bản nháp ở dạng thuần (từ DB hoặc từ đầu ra LLM). Weight là chuỗi decimal chưa kiểm.
type DraftComponent struct {
	Key, Name    string
	Group        Group
	Weight       string
	Citations    []Citation
	FilledByUser bool
}

type DraftRule struct {
	Kind         RuleKind
	Applicable   bool
	Params       json.RawMessage
	Quote        string
	Page         int
	FilledByUser bool
}

type Draft struct {
	Components []DraftComponent
	Rules      []DraftRule
}

var keyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`) //nolint:gochecknoglobals // biểu thức chính quy bất biến

func rtarget(k RuleKind) string { return "rules." + string(k) }

// Check chạy bộ kiểm 4.2.5 (không LLM, không I/O). `cited` trả true khi trích dẫn có thật ở đúng trang (đã tra SQL ở nơi gọi); nil = không kiểm trích dẫn.
// Trả mục chưa rõ (chặn xác nhận) và cảnh báo không chặn (`BONUS_NO_EFFECT`).
func Check(d Draft, cited func(Citation) bool) (unclear []Unclear, warnings []string) {
	add := func(code, target, msg string, details map[string]any) {
		unclear = append(unclear, Unclear{Source: SourceCode, Code: code, Target: target, Message: msg, Details: details})
	}
	if len(d.Components) == 0 {
		add(CodeNoComponents, "components", "Chưa có thành phần điểm nào.", nil)
	}

	// Thành phần: khoá, nhóm, trọng số; tổng trọng số.
	sum, wCK := decimal.Zero, decimal.Zero
	keys := map[string]bool{}
	for _, c := range d.Components {
		t := "components." + c.Key
		if !keyRe.MatchString(c.Key) || keys[c.Key] {
			add(CodeParamInvalid, t, "Khoá thành phần không hợp lệ hoặc trùng.", nil)
		}
		keys[c.Key] = true
		if c.Group != GroupQT && c.Group != GroupCK {
			add(CodeParamInvalid, t, "Nhóm thành phần phải là quá trình hoặc cuối kỳ.", nil)
		}
		w, e := parseNum(KindBonus, "weight", &c.Weight)
		if e != nil || w.GreaterThan(TotalWeight) {
			add(CodeParamInvalid, t, "Trọng số phải là số trong [0; 100].", nil)
			continue
		}
		sum = sum.Add(w)
		if c.Group == GroupCK {
			wCK = wCK.Add(w)
		}
	}
	if len(d.Components) > 0 && !sum.Equal(TotalWeight) {
		add(CodeWeightSum, "components", fmt.Sprintf("Tổng trọng số là %s, cần 100.", sum.String()), map[string]any{"sum": sum.String()})
	}
	var wQT decimal.Decimal
	for _, c := range d.Components {
		if c.Group == GroupQT {
			if w, e := parseNum(KindBonus, "weight", &c.Weight); e == nil {
				wQT = wQT.Add(w)
			}
		}
	}

	// Luật: mỗi loại một dòng (trừ FAIL_CONDITION).
	byKind := map[RuleKind]DraftRule{}
	var fails []DraftRule
	for _, r := range d.Rules {
		if r.Kind == KindFail {
			fails = append(fails, r)
			continue
		}
		if _, dup := byKind[r.Kind]; dup {
			add(CodeParamInvalid, rtarget(r.Kind), "Luật bị khai hai lần.", nil)
			continue
		}
		byKind[r.Kind] = r
	}
	missingOr := func(err *ParamError, missingCode, target, msg string) {
		if err.Missing {
			add(missingCode, target, msg, map[string]any{"field": err.Field})
			return
		}
		add(CodeParamInvalid, target, err.Error(), map[string]any{"field": err.Field})
	}

	if r, ok := byKind[KindBonus]; !ok {
		add(CodeBonusMissing, rtarget(KindBonus), "Quy chế chưa nêu điểm cộng: điền hoặc chọn Không áp dụng.", nil)
	} else if r.Applicable {
		if _, e := ParseBonus(r.Params); e != nil {
			if e.Missing && e.Field == "cap" {
				add(CodeBonusCapMissing, rtarget(KindBonus), "Điểm cộng chưa có mức tối đa.", nil)
			} else {
				missingOr(e, CodeBonusMissing, rtarget(KindBonus), "Điểm cộng thiếu tham số.")
			}
		} else if wQT.IsZero() {
			warnings = append(warnings, WarnBonusNoEffect)
		}
	}
	if r, ok := byKind[KindPenalty]; !ok {
		add(CodeAbsenceRuleMissing, rtarget(KindPenalty), "Quy chế chưa nêu điểm trừ vắng: điền hoặc chọn Không áp dụng.", nil)
	} else if r.Applicable {
		if _, e := ParsePenalty(r.Params); e != nil {
			missingOr(e, CodeAbsenceRuleMissing, rtarget(KindPenalty), "Điểm trừ vắng thiếu tham số.")
		}
	}
	if r, ok := byKind[KindRounding]; !ok {
		add(CodeRoundingMissing, rtarget(KindRounding), "Quy chế chưa nêu cách làm tròn: điền bước làm tròn.", nil)
	} else if _, e := ParseRounding(r.Params); e != nil {
		missingOr(e, CodeRoundingMissing, rtarget(KindRounding), "Làm tròn thiếu tham số.")
	}
	if r, ok := byKind[KindLetterScale]; !ok {
		add(CodeLetterScaleMissing, rtarget(KindLetterScale), "Chưa có thang điểm chữ.", nil)
	} else if _, e := ParseLetterScale(r.Params); e != nil {
		if e.Missing {
			add(CodeLetterScaleMissing, rtarget(KindLetterScale), "Chưa có thang điểm chữ.", nil)
		} else {
			add(CodeLetterScaleInvalid, rtarget(KindLetterScale), "Thang điểm chữ không hợp lệ: "+e.Msg+".", map[string]any{"field": e.Field})
		}
	}
	for _, r := range fails {
		f, _, e := ParseFail(r.Params)
		switch {
		case e != nil:
			add(CodeParamInvalid, rtarget(KindFail), e.Error(), map[string]any{"field": e.Field})
		case f.Kind == FailFinalBelow && wCK.IsZero() && len(d.Components) > 0:
			add(CodeParamInvalid, rtarget(KindFail), "Điều kiện điểm cuối kỳ không dùng được khi không có điểm cuối kỳ trong công thức.", nil)
		}
	}
	if r, ok := byKind[KindMissingScore]; ok {
		if m, e := ParseMissingScore(r.Params); e != nil {
			add(CodeParamInvalid, rtarget(KindMissingScore), e.Error(), map[string]any{"field": e.Field})
		} else {
			for _, k := range m.ZeroComponents {
				if !keys[k] && len(d.Components) > 0 {
					add(CodeParamInvalid, rtarget(KindMissingScore), "Thành phần "+k+" không có trong công thức.", nil)
				}
			}
		}
	}

	// Trích dẫn phải có thật (4.2.6): mục nào dùng trích dẫn không khớp → CITATION_NOT_FOUND.
	if cited != nil {
		for _, c := range d.Components {
			for _, ct := range c.Citations {
				if !cited(ct) {
					add(CodeCitationNotFound, "components."+c.Key, "Trích dẫn không có trong tài liệu quy chế.", map[string]any{"page": ct.Page})
					break
				}
			}
		}
		for _, r := range d.Rules {
			if r.Quote != "" && !cited(Citation{Page: r.Page, Quote: r.Quote}) {
				add(CodeCitationNotFound, rtarget(r.Kind), "Trích dẫn không có trong tài liệu quy chế.", map[string]any{"page": r.Page})
			}
		}
	}
	return unclear, warnings
}

// ParamErrors trả danh sách *ParamError của lỗi gộp (errors.Join) từ BuildRules.
func ParamErrors(err error) []*ParamError {
	var out []*ParamError
	var j interface{ Unwrap() []error }
	if errors.As(err, &j) {
		for _, e := range j.Unwrap() {
			var pe *ParamError
			if errors.As(e, &pe) {
				out = append(out, pe)
			}
		}
	}
	var pe *ParamError
	if len(out) == 0 && errors.As(err, &pe) {
		out = append(out, pe)
	}
	return out
}

// CodesOf liệt kê mã (không trùng, theo thứ tự xuất hiện) của các mục chưa rõ.
func CodesOf(us []Unclear) string {
	seen := map[string]bool{}
	var out []string
	for _, u := range us {
		if u.Code != "" && !seen[u.Code] {
			seen[u.Code] = true
			out = append(out, u.Code)
		}
	}
	return strings.Join(out, ",")
}
