package grading_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/grading"
)

// baseDraft dựng bản nháp hợp lệ từ bản ghi lớp 1 (không có mục chưa rõ).
func baseDraft(t *testing.T) grade.Draft {
	t.Helper()
	var ex grading.Extraction
	require.NoError(t, json.Unmarshal(lop1(t), &ex))
	cit := map[string]grade.Citation{}
	for _, c := range ex.Citations {
		cit[c.ID] = grade.Citation{Page: c.Page, Quote: c.Quote}
	}
	var d grade.Draft
	for _, c := range ex.Components {
		dc := grade.DraftComponent{Key: c.Key, Name: c.Name, Group: grade.Group(c.Group), Weight: c.Weight}
		for _, id := range c.CitationIDs {
			dc.Citations = append(dc.Citations, cit[id])
		}
		d.Components = append(d.Components, dc)
	}
	for _, r := range ex.Rules {
		dr := grade.DraftRule{Kind: grade.RuleKind(r.Kind), Applicable: r.Applicable, Params: r.Params}
		if len(r.CitationIDs) > 0 {
			dr.Quote, dr.Page = cit[r.CitationIDs[0]].Quote, cit[r.CitationIDs[0]].Page
		}
		d.Rules = append(d.Rules, dr)
	}
	return d
}

func (d *dropper) rules(kind grade.RuleKind) {
	var out []grade.DraftRule
	for _, r := range d.d.Rules {
		if r.Kind != kind {
			out = append(out, r)
		}
	}
	d.d.Rules = out
}

type dropper struct{ d *grade.Draft }

func codesOfCheck(d grade.Draft) []string {
	us, _ := grade.Check(d, nil)
	var out []string
	for _, u := range us {
		out = append(out, u.Code)
	}
	return out
}

func setParams(d *grade.Draft, kind grade.RuleKind, params string) {
	for i := range d.Rules {
		if d.Rules[i].Kind == kind {
			d.Rules[i].Params = json.RawMessage(params)
		}
	}
}

func TestCheckerCleanDraft(t *testing.T) {
	us, w := grade.Check(baseDraft(t), func(grade.Citation) bool { return true })
	require.Empty(t, us)
	require.Empty(t, w)
}

func TestCheckerWeightSum(t *testing.T) {
	d := baseDraft(t)
	d.Components[2].Weight = "50" // 20 + 20 + 50
	us, _ := grade.Check(d, nil)
	require.Len(t, us, 1)
	require.Equal(t, grade.CodeWeightSum, us[0].Code)
	require.Equal(t, grade.SourceCode, us[0].Source)
	require.Equal(t, "90", us[0].Details["sum"], "kèm tổng thật")
	d.Components[2].Weight = "70"
	us, _ = grade.Check(d, nil)
	require.Equal(t, "110", us[0].Details["sum"])
}

func TestCheckerNoComponents(t *testing.T) {
	d := baseDraft(t)
	d.Components = nil
	require.Equal(t, []string{grade.CodeNoComponents}, codesOfCheck(d))
}

func TestCheckerBonus(t *testing.T) {
	d := baseDraft(t)
	(&dropper{&d}).rules(grade.KindBonus)
	require.Equal(t, []string{grade.CodeBonusMissing}, codesOfCheck(d))
	// "Không áp dụng" thì hết mục chưa rõ
	d.Rules = append(d.Rules, grade.DraftRule{Kind: grade.KindBonus, Applicable: false, Params: json.RawMessage(`{}`)})
	require.Empty(t, codesOfCheck(d))
}

func TestCheckerBonusCap(t *testing.T) {
	d := baseDraft(t)
	setParams(&d, grade.KindBonus, `{"source":"PARTICIPATION","per_speak":"0.25"}`)
	us, _ := grade.Check(d, nil)
	require.Len(t, us, 1)
	require.Equal(t, grade.CodeBonusCapMissing, us[0].Code)
	require.Equal(t, "rules.BONUS", us[0].Target)
}

func TestCheckerAbsence(t *testing.T) {
	d := baseDraft(t)
	(&dropper{&d}).rules(grade.KindPenalty)
	require.Equal(t, []string{grade.CodeAbsenceRuleMissing}, codesOfCheck(d))
	d.Rules = append(d.Rules, grade.DraftRule{Kind: grade.KindPenalty, Applicable: false, Params: json.RawMessage(`{}`)})
	require.Empty(t, codesOfCheck(d))
}

func TestCheckerRounding(t *testing.T) {
	d := baseDraft(t)
	(&dropper{&d}).rules(grade.KindRounding)
	us, _ := grade.Check(d, nil)
	require.Len(t, us, 1)
	require.Equal(t, grade.CodeRoundingMissing, us[0].Code)
	require.Equal(t, "rules.ROUNDING", us[0].Target)
	// giảng viên điền bước 0,1 → hết (không có "không áp dụng" cho làm tròn)
	d.Rules = append(d.Rules, grade.DraftRule{Kind: grade.KindRounding, Applicable: true, Params: json.RawMessage(`{"step":"0.1","mode":"HALF_UP"}`), FilledByUser: true})
	require.Empty(t, codesOfCheck(d))
}

func TestCheckerLetterScale(t *testing.T) {
	d := baseDraft(t)
	(&dropper{&d}).rules(grade.KindLetterScale)
	require.Equal(t, []string{grade.CodeLetterScaleMissing}, codesOfCheck(d))
	for name, bands := range map[string]string{
		"không tăng dần":      `{"bands":[{"min":"9","letter":"A+"},{"min":"9","letter":"A"},{"min":"7","letter":"B"},{"min":"0","letter":"F"}]}`,
		"không bắt đầu từ 0":  `{"bands":[{"min":"9","letter":"A"},{"min":"1","letter":"F"}]}`,
		"hở (thiếu mức cuối)": `{"bands":[{"min":"9","letter":"A"},{"min":"5","letter":"C"}]}`,
		"điểm chữ trùng":      `{"bands":[{"min":"9","letter":"A"},{"min":"5","letter":"A"},{"min":"0","letter":"F"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			d := baseDraft(t)
			setParams(&d, grade.KindLetterScale, bands)
			require.Equal(t, []string{grade.CodeLetterScaleInvalid}, codesOfCheck(d))
		})
	}
	d = baseDraft(t)
	setParams(&d, grade.KindLetterScale, `{}`)
	require.Equal(t, []string{grade.CodeLetterScaleMissing}, codesOfCheck(d), "luật có nhưng thiếu bands")
}

func TestCheckerParamInvalid(t *testing.T) {
	t.Run("trọng số không phải số", func(t *testing.T) {
		d := baseDraft(t)
		d.Components[0].Weight = "abc"
		us, _ := grade.Check(d, nil)
		require.Equal(t, grade.CodeParamInvalid, us[0].Code)
		require.Equal(t, "components.midterm", us[0].Target)
	})
	t.Run("trọng số ngoài [0;100]", func(t *testing.T) {
		d := baseDraft(t)
		d.Components[0].Weight = "120"
		require.Contains(t, codesOfCheck(d), grade.CodeParamInvalid)
	})
	t.Run("bước làm tròn lạ", func(t *testing.T) {
		d := baseDraft(t)
		setParams(&d, grade.KindRounding, `{"step":"0.3","mode":"HALF_UP"}`)
		require.Equal(t, []string{grade.CodeParamInvalid}, codesOfCheck(d))
	})
	t.Run("điều kiện cuối kỳ khi nhóm CK trọng số 0", func(t *testing.T) {
		d := baseDraft(t)
		d.Components[0].Weight, d.Components[1].Weight, d.Components[2].Weight = "50", "50", "0"
		us, _ := grade.Check(d, nil)
		require.Len(t, us, 1)
		require.Equal(t, grade.CodeParamInvalid, us[0].Code)
		require.Equal(t, "rules.FAIL_CONDITION", us[0].Target)
	})
	t.Run("tham số số JSON thay vì chuỗi", func(t *testing.T) {
		d := baseDraft(t)
		setParams(&d, grade.KindBonus, `{"source":"PARTICIPATION","per_speak":0.25,"cap":"1.0"}`)
		require.Equal(t, []string{grade.CodeParamInvalid}, codesOfCheck(d))
	})
	t.Run("thành phần không có trong MISSING_SCORE", func(t *testing.T) {
		d := baseDraft(t)
		setParams(&d, grade.KindMissingScore, `{"zero_components":["khong-co"]}`)
		require.Equal(t, []string{grade.CodeParamInvalid}, codesOfCheck(d))
	})
	t.Run("khoá thành phần trùng", func(t *testing.T) {
		d := baseDraft(t)
		d.Components[1].Key = "midterm"
		require.Contains(t, codesOfCheck(d), grade.CodeParamInvalid)
	})
}

// TestCheckerBonusNoEffectWarning — w_QT = 0 → cảnh báo không chặn BONUS_NO_EFFECT.
func TestCheckerBonusNoEffectWarning(t *testing.T) {
	d := baseDraft(t)
	d.Components[0].Weight, d.Components[1].Weight, d.Components[2].Weight = "0", "0", "100"
	us, w := grade.Check(d, nil)
	require.Empty(t, us)
	require.Equal(t, []string{grade.WarnBonusNoEffect}, w)
}
