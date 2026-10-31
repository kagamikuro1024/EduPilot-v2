package grade

import (
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// Status của một kết quả: COMPLETE (có HP), INCOMPLETE (thiếu điểm, không bịa số 0), INVALID (công thức thiếu luật bắt buộc: làm tròn / thang chữ).
type Status string

const (
	StatusComplete   Status = "COMPLETE"
	StatusIncomplete Status = "INCOMPLETE"
	StatusInvalid    Status = "INVALID"
)

// Lý do thiếu điểm của một item (giải trình hiển thị chữ tương ứng).
const (
	ReasonNoScore          = "NO_SCORE"
	ReasonExamUnpublished  = "EXAM_UNPUBLISHED"
	ReasonColumnUnpublish  = "COLUMN_UNPUBLISHED"
	WarnBonusNoEffect      = "BONUS_NO_EFFECT"
	WarnPenaltyNotInScheme = "PENALTY_NOT_IN_SCHEME"
	WarnBonusNotInScheme   = "BONUS_NOT_IN_SCHEME"
)

// MissingItem: item của thành phần chưa có điểm (và nguồn chưa chốt, hoặc thành phần không nằm trong zero_components).
type MissingItem struct {
	Component string
	Item      string
	Reason    string
}

// ComponentResult là điểm một thành phần (thang 10).
type ComponentResult struct {
	Key, Name string
	Group     Group
	Weight    decimal.Decimal
	Score     decimal.Decimal
	Computed  bool
}

// Line là một dòng giải trình. Value là chuỗi (dấu phẩy, đủ chữ số, bỏ số 0 thừa — TLR-5), Expr chỉ gồm các số đã hiển thị
// (cú pháp: + − × ÷ ( ) min(a; b) max(a; b) round(x)) nên tính lại bằng decimal ra đúng Value. Quote nil ⇔ mục giảng viên tự điền / không có căn cứ.
type Line struct {
	Label        string
	Expr         string
	Value        string
	Quote        *string
	Page         int
	FilledByUser bool
}

// Result là kết quả Compute cho một sinh viên.
type Result struct {
	Status        Status
	Missing       []string // key thành phần còn thiếu
	MissingItems  []MissingItem
	Components    []ComponentResult
	QT0           decimal.Decimal // chỉ khi HasQT
	Bonus         decimal.Decimal // Σ điểm SPEAK + BONUS (chưa trần)
	BonusApplied  decimal.Decimal // min(Bonus, cap)
	Penalty       decimal.Decimal
	QTBeforeClamp decimal.Decimal
	QT            decimal.Decimal
	CK            decimal.Decimal // chỉ khi HasCK
	HPRaw         decimal.Decimal // đã chuẩn hoá Round(NormalizeDigits)
	HP            decimal.Decimal // đã làm tròn theo bước
	HasQT, HasCK  bool
	Letter        string
	Verdict       string // "OK" | "LIET:final<3.0" | "LIET:absent>=7" | "INCOMPLETE:weekly+final" | "INVALID:ROUNDING_MISSING"
	Warnings      []string
	Lines         []Line
}

func fmtNum(d decimal.Decimal) string { return strings.Replace(d.String(), ".", ",", 1) }

// quote lấy căn cứ quy chế của một luật; mục tự điền (hoặc không có trích dẫn) → nil.
func (m RuleMeta) quote() (*string, int, bool) {
	if m.FilledByUser || m.Quote == "" {
		return nil, 0, m.FilledByUser
	}
	q := m.Quote
	return &q, m.Page, false
}

func (c Component) quote() (*string, int, bool) {
	if c.FilledByUser || len(c.Citations) == 0 {
		return nil, 0, c.FilledByUser
	}
	q := c.Citations[0].Quote
	return &q, c.Citations[0].Page, false
}

func (r Rules) zeroSet() map[string]bool {
	out := map[string]bool{}
	if r.MissingScore != nil {
		for _, k := range r.MissingScore.ZeroComponents {
			out[k] = true
		}
	}
	return out
}

// Compute là hàm THUẦN: không context, không I/O, không đồng hồ (SRS 4.3). Công thức ở 4.3.2.
func Compute(s Scheme, in Input) Result { return compute(s, in, nil) }

// WhatIf = Compute với điểm cuối kỳ CK = ck (0–10), không ghi gì; điều kiện liệt vẫn áp.
func WhatIf(s Scheme, in Input, ck decimal.Decimal) Result { return compute(s, in, &ck) }

// MinFinalFor trả điểm cuối kỳ nhỏ nhất (bước 0,01) ∈ [0; 10] để điểm chữ ≥ target và không liệt; false nếu không đạt được
// (kể cả khi còn thiếu điểm thành phần khác). Dùng cùng Compute (không có đường tính thứ hai).
func MinFinalFor(s Scheme, in Input, target string) (decimal.Decimal, bool) {
	if s.Rules.Letters == nil {
		return decimal.Zero, false
	}
	rank := map[string]int{}
	for i, b := range s.Rules.Letters.Bands {
		rank[b.Letter] = i
	}
	want, ok := rank[target]
	if !ok {
		return decimal.Zero, false
	}
	step := decimal.New(1, -2)
	for ck := decimal.Zero; ck.LessThanOrEqual(MaxScale); ck = ck.Add(step) {
		r := WhatIf(s, in, ck)
		if r.Status != StatusComplete {
			return decimal.Zero, false
		}
		if r.Verdict == "OK" && rank[r.Letter] <= want {
			return ck, true
		}
	}
	return decimal.Zero, false
}

func compute(s Scheme, in Input, ckOverride *decimal.Decimal) Result {
	r := Result{Status: StatusComplete, Verdict: "OK"}
	switch {
	case s.Rules.Rounding == nil:
		r.Status, r.Verdict = StatusInvalid, "INVALID:ROUNDING_MISSING"
		return r
	case s.Rules.Letters == nil:
		r.Status, r.Verdict = StatusInvalid, "INVALID:LETTER_SCALE_MISSING"
		return r
	}
	zero := s.Rules.zeroSet()

	// Trọng số từng nhóm.
	var wQT, wCK decimal.Decimal
	for _, c := range s.Components {
		if c.Group == GroupQT {
			wQT = wQT.Add(c.Weight)
		} else {
			wCK = wCK.Add(c.Weight)
		}
	}
	hasQT, hasCK := wQT.Sign() > 0, wCK.Sign() > 0 // nhóm có w = 0: bỏ số hạng, không chia (TLR-9)

	// Điểm thành phần (4.3.2 bước 1–2, 4.3.3).
	for _, c := range s.Components {
		cr := ComponentResult{Key: c.Key, Name: c.Name, Group: c.Group, Weight: c.Weight}
		if c.Group == GroupCK && ckOverride != nil {
			cr.Score, cr.Computed = *ckOverride, true
			r.Components = append(r.Components, cr)
			continue
		}
		var scores []decimal.Decimal
		var exprs []string
		missing := false
		for _, it := range c.Items {
			raw, ok := in.Scores[it.ID]
			if ok {
				si := raw.Mul(MaxScale).DivRound(it.Max, DivPrecision)
				scores = append(scores, si)
				if it.Max.Equal(MaxScale) {
					exprs = append(exprs, fmtNum(raw))
				} else {
					exprs = append(exprs, fmtNum(raw)+" × 10 ÷ "+fmtNum(it.Max))
				}
				continue
			}
			if it.Closed && zero[c.Key] { // nguồn đã chốt sổ cho item → 0 theo quy chế (TLR-2)
				scores = append(scores, decimal.Zero)
				exprs = append(exprs, "0")
				continue
			}
			reason := ReasonNoScore
			switch {
			case it.Source == SourceExam && !it.Closed:
				reason = ReasonExamUnpublished
			case it.Source != SourceExam && !it.Closed:
				reason = ReasonColumnUnpublish
			}
			if (c.Group == GroupQT && !hasQT) || (c.Group == GroupCK && !hasCK) {
				continue // nhóm bị bỏ số hạng: thiếu điểm không chặn
			}
			missing = true
			r.MissingItems = append(r.MissingItems, MissingItem{Component: c.Key, Item: it.Key, Reason: reason})
		}
		if missing {
			r.Missing = append(r.Missing, c.Key)
		} else if len(scores) > 0 {
			sum := decimal.Zero
			for _, v := range scores {
				sum = sum.Add(v)
			}
			cr.Score = sum.DivRound(decimal.NewFromInt(int64(len(scores))), DivPrecision)
			cr.Computed = true
			expr := exprs[0]
			if len(scores) > 1 {
				parts := make([]string, len(scores))
				for i, v := range scores {
					parts[i] = fmtNum(v)
				}
				expr = "(" + strings.Join(parts, " + ") + ") ÷ " + strconv.Itoa(len(scores))
			}
			q, pg, fu := c.quote()
			r.Lines = append(r.Lines, Line{Label: c.Name + " (" + fmtNum(c.Weight) + "%)", Expr: expr, Value: fmtNum(cr.Score), Quote: q, Page: pg, FilledByUser: fu})
		}
		r.Components = append(r.Components, cr)
	}
	// Item không có điểm vì không dự → 0: giữ dòng riêng cho thành phần (đã có ở trên). Thiếu → INCOMPLETE.
	if len(r.Missing) > 0 {
		r.Status = StatusIncomplete
		r.Verdict = "INCOMPLETE:" + strings.Join(r.Missing, "+")
		for _, m := range r.MissingItems {
			txt := "Chưa có điểm"
			switch m.Reason {
			case ReasonExamUnpublished:
				txt = "Bài thi chưa công bố"
			case ReasonColumnUnpublish:
				txt = "Cột chưa công bố"
			}
			r.Lines = append(r.Lines, Line{Label: m.Item, Value: txt})
		}
		return r
	}

	// Điểm cộng, điểm trừ (bước 4).
	r.Bonus = in.BonusPoints
	bonusOn := s.Rules.Bonus != nil && s.Rules.Bonus.Applicable
	penOn := s.Rules.Penalty != nil && s.Rules.Penalty.Applicable
	if bonusOn {
		r.BonusApplied = decimal.Min(in.BonusPoints, s.Rules.Bonus.Cap)
		q, pg, fu := s.Rules.Bonus.quote()
		r.Lines = append(r.Lines, Line{Label: "Điểm cộng", Expr: "min(" + fmtNum(in.BonusPoints) + "; " + fmtNum(s.Rules.Bonus.Cap) + ")", Value: fmtNum(r.BonusApplied), Quote: q, Page: pg, FilledByUser: fu})
		if !hasQT {
			r.Warnings = append(r.Warnings, WarnBonusNoEffect)
		}
	} else if s.Rules.Bonus != nil && in.BonusPoints.Sign() != 0 {
		r.Warnings = append(r.Warnings, WarnBonusNotInScheme)
	}
	if penOn {
		p := s.Rules.Penalty
		over := decimal.Max(decimal.Zero, decimal.NewFromInt(int64(in.Absent-p.Free)))
		r.Penalty = over.Mul(p.Per).Add(in.PenaltyEventPoints)
		q, pg, fu := p.quote()
		r.Lines = append(r.Lines, Line{Label: "Điểm trừ", Expr: "max(0; " + strconv.Itoa(in.Absent) + " − " + strconv.Itoa(p.Free) + ") × " + fmtNum(p.Per) + " + " + fmtNum(in.PenaltyEventPoints), Value: fmtNum(r.Penalty), Quote: q, Page: pg, FilledByUser: fu})
	} else if s.Rules.Penalty != nil && in.PenaltyEventPoints.Sign() != 0 {
		r.Warnings = append(r.Warnings, WarnPenaltyNotInScheme)
	}

	// Điểm quá trình (bước 3, 5).
	weighted := func(g Group, w decimal.Decimal) (decimal.Decimal, string) {
		sum := decimal.Zero
		var terms []string
		for _, c := range r.Components {
			if c.Group == g && c.Computed {
				sum = sum.Add(c.Weight.Mul(c.Score))
				terms = append(terms, fmtNum(c.Weight)+" × "+fmtNum(c.Score))
			}
		}
		return sum.DivRound(w, DivPrecision), "(" + strings.Join(terms, " + ") + ") ÷ " + fmtNum(w)
	}
	hpNum := decimal.Zero
	var hpTerms []string
	if hasQT {
		var expr string
		r.QT0, expr = weighted(GroupQT, wQT)
		r.HasQT = true
		r.Lines = append(r.Lines, Line{Label: "Điểm quá trình trước điểm cộng / trừ", Expr: expr, Value: fmtNum(r.QT0)})
		r.QTBeforeClamp = r.QT0.Add(r.BonusApplied).Sub(r.Penalty)
		r.QT = decimal.Min(MaxScale, decimal.Max(decimal.Zero, r.QTBeforeClamp))
		r.Lines = append(r.Lines, Line{Label: "Điểm quá trình", Expr: "min(10; max(0; " + fmtNum(r.QT0) + " + " + fmtNum(r.BonusApplied) + " − " + fmtNum(r.Penalty) + "))", Value: fmtNum(r.QT)})
		hpNum = hpNum.Add(wQT.Mul(r.QT))
		hpTerms = append(hpTerms, fmtNum(wQT)+" × "+fmtNum(r.QT))
	}
	if hasCK {
		if ckOverride != nil {
			r.CK = *ckOverride
			r.Lines = append(r.Lines, Line{Label: "Điểm cuối kỳ (giả định)", Expr: fmtNum(r.CK), Value: fmtNum(r.CK)})
		} else {
			var expr string
			r.CK, expr = weighted(GroupCK, wCK)
			r.Lines = append(r.Lines, Line{Label: "Điểm cuối kỳ", Expr: expr, Value: fmtNum(r.CK)})
		}
		r.HasCK = true
		hpNum = hpNum.Add(wCK.Mul(r.CK))
		hpTerms = append(hpTerms, fmtNum(wCK)+" × "+fmtNum(r.CK))
	}

	// Điểm học phần thô → chuẩn hoá độ chính xác máy → làm tròn một lần ở cuối (bước 6–7, TLR-1, TLR-13).
	r.HPRaw = hpNum.DivRound(TotalWeight, DivPrecision).Round(NormalizeDigits)
	r.Lines = append(r.Lines, Line{Label: "Điểm học phần thô", Expr: "(" + strings.Join(hpTerms, " + ") + ") ÷ " + fmtNum(TotalWeight), Value: fmtNum(r.HPRaw)})
	step := s.Rules.Rounding.Step
	r.HP = r.HPRaw.DivRound(step, DivPrecision).Round(0).Mul(step)
	q, pg, fu := s.Rules.Rounding.quote()
	r.Lines = append(r.Lines, Line{Label: "Làm tròn đến " + fmtNum(step) + " (nửa lên)", Expr: "round(" + fmtNum(r.HPRaw) + " ÷ " + fmtNum(step) + ") × " + fmtNum(step), Value: fmtNum(r.HP), Quote: q, Page: pg, FilledByUser: fu})

	// Điểm chữ (bước 8) và điều kiện liệt (bước 9).
	for _, b := range s.Rules.Letters.Bands {
		if r.HP.GreaterThanOrEqual(b.Min) {
			r.Letter = b.Letter
			break
		}
	}
	q, pg, fu = s.Rules.Letters.quote()
	r.Lines = append(r.Lines, Line{Label: "Điểm chữ", Value: r.Letter, Quote: q, Page: pg, FilledByUser: fu})
	for _, f := range s.Rules.Fails {
		var hit bool
		var verdict string
		switch f.Kind {
		case FailFinalBelow:
			hit, verdict = r.HasCK && r.CK.LessThan(f.Value), "LIET:final<"+f.Text
		case FailAbsentAtLeast:
			hit, verdict = decimal.NewFromInt(int64(in.Absent)).GreaterThanOrEqual(f.Value), "LIET:absent>="+f.Text
		}
		if hit {
			r.Verdict, r.Letter = verdict, "F"
			q, pg, fu := f.quote()
			r.Lines = append(r.Lines, Line{Label: "Không đạt", Value: "F", Quote: q, Page: pg, FilledByUser: fu})
			break
		}
	}
	return r
}
