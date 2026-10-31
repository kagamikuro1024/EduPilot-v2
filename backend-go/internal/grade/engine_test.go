package grade_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
)

// lop1 dựng công thức lớp 1 và đầu vào từng sinh viên từ inputs-lop1.csv. Tuần trống = bài thi đã công bố, không có lượt (→ 0 theo quy chế).
func lop1(t *testing.T) (grade.Scheme, map[string]uuid.UUID, []map[string]string) {
	t.Helper()
	s, ids := buildScheme(t, loadExtraction(t, "extraction-lop1.json"), defaultTweak())
	return s, ids, readCSV(t, fixDir+"inputs-lop1.csv")
}

func expected30(t *testing.T) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	for _, r := range readCSV(t, seedDir+"expected_final_grades.csv") {
		out[r["account"]] = r
	}
	require.Len(t, out, 30)
	return out
}

// TestEngineExpected30 — US-P6-02 AC1: 30/30 khớp đáp án tính tay (sửa mã, không sửa CSV).
func TestEngineExpected30(t *testing.T) {
	s, ids, rows := lop1(t)
	want := expected30(t)
	require.Len(t, rows, 30)
	for _, row := range rows {
		w := want[row["account"]]
		t.Run(row["account"], func(t *testing.T) {
			r := grade.Compute(s, inputOf(row, ids))
			require.Equal(t, grade.StatusComplete, r.Status)
			require.True(t, r.HP.Equal(dec(w["hp"])), "hp %s ≠ %s", r.HP, w["hp"])
			require.Equal(t, w["letter"], r.Letter)
			require.Equal(t, w["verdict"], r.Verdict)
			require.True(t, r.QT.Equal(dec(w["qt"])), "qt %s ≠ %s", r.QT, w["qt"])
			require.True(t, r.QTBeforeClamp.Equal(dec(w["qt_before_clamp"])), "qt_before_clamp %s ≠ %s", r.QTBeforeClamp, w["qt_before_clamp"])
			require.True(t, r.HPRaw.Equal(dec(w["hp_raw"])), "hp_raw %s ≠ %s", r.HPRaw, w["hp_raw"])
			require.True(t, r.BonusApplied.Equal(dec(w["bonus_applied"])))
			require.True(t, r.Penalty.Equal(dec(w["penalty"])))
		})
	}
}

// caseScheme dựng công thức và đầu vào của một ca ở cases-tong-hop.csv.
func caseScheme(t *testing.T, c map[string]string) (grade.Scheme, grade.Input) {
	t.Helper()
	tw := defaultTweak()
	tw.weights = map[string]string{midKey: c["w_midterm"], oneItem: c["w_weekly"], finalKey: c["w_final"]}
	tw.midMax = c["midterm_max"]
	switch c["weekly_state"] {
	case "SCORE", "EXAM_PUBLISHED_NO_ATTEMPT":
	case "EXAM_UNPUBLISHED":
		tw.weeklyClose = false
	case "MANUAL_BLANK_UNPUBLISHED":
		tw.weeklyExam, tw.weeklyClose = false, false
	case "MANUAL_BLANK_PUBLISHED":
		tw.weeklyExam = false
	default:
		require.Failf(t, "weekly_state lạ", c["weekly_state"])
	}
	tw.rules = func(rows []grade.RuleRow) []grade.RuleRow {
		for i := range rows {
			switch rows[i].Kind {
			case grade.KindRounding:
				rows[i].Params = json.RawMessage(fmt.Sprintf(`{"step":%q,"mode":"HALF_UP"}`, c["round_step"]))
			case grade.KindPenalty:
				rows[i].Applicable = c["penalty_rule"] == "ON"
			}
		}
		return rows
	}
	s, ids := buildScheme(t, loadExtraction(t, "extraction-lop1.json"), tw)
	return s, inputOf(c, ids)
}

// TestEngineCases — AC2: 18 ca biên.
func TestEngineCases(t *testing.T) {
	cases := readCSV(t, fixDir+"cases-tong-hop.csv")
	require.Len(t, cases, 18)
	for _, c := range cases {
		t.Run(c["case"], func(t *testing.T) {
			s, in := caseScheme(t, c)
			r := grade.Compute(s, in)
			require.Equal(t, c["verdict"], r.Verdict)
			if strings.HasPrefix(c["verdict"], "INCOMPLETE") {
				require.Equal(t, grade.StatusIncomplete, r.Status)
				require.Empty(t, r.Letter, "không có điểm chữ khi thiếu điểm")
				require.True(t, r.HP.IsZero())
				return
			}
			require.Equal(t, grade.StatusComplete, r.Status)
			require.True(t, r.HPRaw.Equal(dec(c["hp_raw"])), "hp_raw %s ≠ %s", r.HPRaw, c["hp_raw"])
			require.True(t, r.HP.Equal(dec(c["hp"])), "hp %s ≠ %s", r.HP, c["hp"])
			require.Equal(t, c["letter"], r.Letter)
			if w := c["scheme_warnings"]; w != "" {
				require.Contains(t, r.Warnings, w)
			} else {
				require.Empty(t, r.Warnings)
			}
		})
	}
}

// onlyFinal: công thức chỉ có điểm cuối kỳ (100 %), bước làm tròn `step`, để kiểm riêng bảng làm tròn.
func onlyFinal(t *testing.T, step string) (grade.Scheme, uuid.UUID) {
	t.Helper()
	s, ids := buildScheme(t, loadExtraction(t, "extraction-lop1.json"), tweak{
		weights: map[string]string{midKey: "0", oneItem: "0", finalKey: "100"}, weeklyExam: true, weeklyClose: true,
		rules: func(rows []grade.RuleRow) []grade.RuleRow {
			for i := range rows {
				if rows[i].Kind == grade.KindRounding {
					rows[i].Params = json.RawMessage(fmt.Sprintf(`{"step":%q,"mode":"HALF_UP"}`, step))
				}
			}
			return rows
		}})
	return s, ids[finalKey]
}

// TestRoundHalfUpTable — AC3: nửa lên đúng ở bước 0,5 / 0,1 / 1; không "chẵn gần nhất".
func TestRoundHalfUpTable(t *testing.T) {
	for _, c := range []struct{ step, raw, want string }{
		{"0.5", "6.25", "6.5"}, {"0.5", "6.75", "7"}, {"0.5", "6.5", "6.5"}, {"0.5", "6.249", "6"}, {"0.5", "6.251", "6.5"},
		{"0.5", "8.25", "8.5"}, {"0.5", "5.75", "6"}, {"0.5", "0.25", "0.5"},
		{"0.1", "7.25", "7.3"}, {"0.1", "7.35", "7.4"}, {"0.1", "7.24", "7.2"},
		{"1", "6.5", "7"}, {"1", "6.49", "6"}, {"0.25", "6.125", "6.25"},
	} {
		t.Run(c.step+"/"+c.raw, func(t *testing.T) {
			s, item := onlyFinal(t, c.step)
			r := grade.Compute(s, grade.Input{Scores: map[uuid.UUID]decimal.Decimal{item: dec(c.raw)}})
			require.Equal(t, grade.StatusComplete, r.Status)
			require.True(t, r.HP.Equal(dec(c.want)), "%s: %s ≠ %s", c.raw, r.HP, c.want)
		})
	}
}

// TestNoIntermediateRounding — điểm thành phần, QT giữ nguyên độ chính xác; chỉ HP làm tròn.
func TestNoIntermediateRounding(t *testing.T) {
	s, ids, rows := lop1(t)
	for _, row := range rows {
		if row["account"] != "sv04" {
			continue
		}
		r := grade.Compute(s, inputOf(row, ids))
		require.True(t, r.QT.Equal(dec("6.625")), r.QT.String())
		require.True(t, r.HPRaw.Equal(dec("6.25")))
		require.True(t, r.HP.Equal(dec("6.5")))
	}
}

// TestDivRoundNoDrift — S16 (TLR-1): item 10/30 không làm HP thô 6,25 thành 6,2499… (làm tròn xuống 6,0).
func TestDivRoundNoDrift(t *testing.T) {
	var c map[string]string
	for _, x := range readCSV(t, fixDir+"cases-tong-hop.csv") {
		if strings.HasPrefix(x["case"], "S16") {
			c = x
		}
	}
	s, in := caseScheme(t, c)
	r := grade.Compute(s, in)
	require.True(t, r.HPRaw.Equal(dec("6.25")), r.HPRaw.String())
	require.True(t, r.HP.Equal(dec("6.5")))
}

// TestMissingScoreIncomplete — thiếu điểm → INCOMPLETE, không bịa 0.
func TestMissingScoreIncomplete(t *testing.T) {
	s, ids, rows := lop1(t)
	in := inputOf(rows[0], ids)
	delete(in.Scores, ids[midKey])
	r := grade.Compute(s, in)
	require.Equal(t, grade.StatusIncomplete, r.Status)
	require.Equal(t, []string{midKey}, r.Missing)
	require.Equal(t, grade.ReasonNoScore, r.MissingItems[0].Reason)
	require.Equal(t, "INCOMPLETE:midterm", r.Verdict)
	delete(in.Scores, ids[finalKey])
	r = grade.Compute(s, in)
	require.Equal(t, []string{midKey, finalKey}, r.Missing)
}

// TestMissingScoreZeroRule — thành phần trong zero_components, nguồn đã chốt → 0 và có dòng giải trình.
func TestMissingScoreZeroRule(t *testing.T) {
	s, ids, rows := lop1(t)
	in := inputOf(rows[0], ids)
	delete(in.Scores, ids[oneItem])
	r := grade.Compute(s, in)
	require.Equal(t, grade.StatusComplete, r.Status)
	for _, c := range r.Components {
		if c.Key == oneItem {
			require.True(t, c.Score.IsZero())
		}
	}
	found := false
	for _, l := range r.Lines {
		if strings.HasPrefix(l.Label, "Kiểm tra hằng tuần") && l.Expr == "0" {
			found = true
		}
	}
	require.True(t, found, "dòng 'không dự → 0'")
}

// TestMissingScoreZeroOnlyWhenSourceClosed — TLR-2: nguồn chưa chốt → INCOMPLETE với lý do tương ứng; thành phần ngoài zero_components không được 0.
func TestMissingScoreZeroOnlyWhenSourceClosed(t *testing.T) {
	for _, c := range []struct {
		state, reason string
	}{{"EXAM_UNPUBLISHED", grade.ReasonExamUnpublished}, {"MANUAL_BLANK_UNPUBLISHED", grade.ReasonColumnUnpublish}} {
		var row map[string]string
		for _, x := range readCSV(t, fixDir+"cases-tong-hop.csv") {
			if x["weekly_state"] == c.state {
				row = x
			}
		}
		s, in := caseScheme(t, row)
		r := grade.Compute(s, in)
		require.Equal(t, grade.StatusIncomplete, r.Status, c.state)
		require.Equal(t, c.reason, r.MissingItems[0].Reason)
	}
	// midterm trống dù nguồn đã chốt: không nằm trong zero_components → thiếu
	s, ids, rows := lop1(t)
	in := inputOf(rows[0], ids)
	delete(in.Scores, ids[midKey])
	require.Equal(t, grade.StatusIncomplete, grade.Compute(s, in).Status)
}

// TestExpectedStudentC — sinh viên C (không có lượt, bài đã công bố): qt = 1,25, hp = 3,0.
func TestExpectedStudentC(t *testing.T) {
	s, ids, rows := lop1(t)
	for _, row := range rows {
		if row["account"] == "sv.nguyco" {
			r := grade.Compute(s, inputOf(row, ids))
			require.True(t, r.QT.Equal(dec("1.25")) && r.HP.Equal(dec("3")), "%s %s", r.QT, r.HP)
			require.Equal(t, "F", r.Letter)
			require.Equal(t, "OK", r.Verdict)
		}
	}
}

// TestFailConditions — liệt vì cuối kỳ < 3,0 hoặc vắng ≥ 7; HP giữ số đã tính.
func TestFailConditions(t *testing.T) {
	s, ids, rows := lop1(t)
	for _, row := range rows {
		if row["account"] != "sv20" {
			continue
		}
		r := grade.Compute(s, inputOf(row, ids))
		require.Equal(t, "LIET:final<3.0", r.Verdict)
		require.Equal(t, "F", r.Letter)
		require.True(t, r.HP.Equal(dec("5")), "hp giữ nguyên, không về 0")
		last := r.Lines[len(r.Lines)-1]
		require.Equal(t, "Không đạt", last.Label)
		require.NotNil(t, last.Quote)
		require.Contains(t, *last.Quote, "dưới 3,0")
		in := inputOf(row, ids)
		in.Scores[ids[finalKey]] = dec("8")
		in.Absent = 7
		r = grade.Compute(s, in)
		require.Equal(t, "LIET:absent>=7", r.Verdict)
		in.Absent = 6
		require.Equal(t, "OK", grade.Compute(s, in).Verdict)
	}
}

// TestFinalExactlyThreeNotFail — cuối kỳ đúng 3,0 không liệt (sv23, sv25).
func TestFinalExactlyThreeNotFail(t *testing.T) {
	s, ids, rows := lop1(t)
	n := 0
	for _, row := range rows {
		if row["account"] == "sv23" || row["account"] == "sv25" {
			require.Equal(t, "3.00", row["final_exam"])
			require.Equal(t, "OK", grade.Compute(s, inputOf(row, ids)).Verdict)
			n++
		}
	}
	require.Equal(t, 2, n)
}

// TestZeroWeightGroupNoDivide — TLR-9: nhóm trọng số 0 không chia cho 0; w_CK = 0 thì thiếu cuối kỳ không làm INCOMPLETE.
func TestZeroWeightGroupNoDivide(t *testing.T) {
	s, ids := buildScheme(t, loadExtraction(t, "extraction-lop1.json"), tweak{weights: map[string]string{midKey: "50", oneItem: "50", finalKey: "0"}, weeklyExam: true, weeklyClose: true,
		rules: func(rows []grade.RuleRow) []grade.RuleRow {
			var out []grade.RuleRow
			for _, r := range rows {
				if r.Kind != grade.KindFail { // FINAL_BELOW với w_CK = 0 là PARAM_INVALID của bộ kiểm (US-P6-03), không phải của engine
					out = append(out, r)
				}
			}
			return out
		}})
	in := grade.Input{Scores: map[uuid.UUID]decimal.Decimal{ids[midKey]: dec("6"), ids[oneItem]: dec("8")}}
	r := grade.Compute(s, in)
	require.Equal(t, grade.StatusComplete, r.Status)
	require.False(t, r.HasCK)
	require.True(t, r.HPRaw.Equal(dec("7")), r.HPRaw.String())
	require.True(t, r.HP.Equal(dec("7")))
}

func TestBonusNoEffectWarning(t *testing.T) {
	var c map[string]string
	for _, x := range readCSV(t, fixDir+"cases-tong-hop.csv") {
		if strings.HasPrefix(x["case"], "S17") {
			c = x
		}
	}
	s, in := caseScheme(t, c)
	r := grade.Compute(s, in)
	require.Equal(t, []string{grade.WarnBonusNoEffect}, r.Warnings)
	require.False(t, r.HasQT)
	require.True(t, r.HP.Equal(dec("7.5")))
}

func TestPenaltyNotApplicable(t *testing.T) {
	s, ids, rows := lop1(t)
	for i := range s.Rules.Fails {
		s.Rules.Fails[i].Value = dec("99") // chỉ kiểm điểm trừ, không liệt
	}
	s.Rules.Penalty.Applicable = false
	in := inputOf(rows[2], ids) // sv.nguyco: vắng 5
	in.PenaltyEventPoints = dec("0.5")
	r := grade.Compute(s, in)
	require.True(t, r.Penalty.IsZero(), "P = 0 dù vắng 5 và có sự kiện PENALTY (TLR-17)")
	require.Contains(t, r.Warnings, grade.WarnPenaltyNotInScheme)
}

func TestPenaltyNotInSchemeWarning(t *testing.T) {
	s, ids, rows := lop1(t)
	in := inputOf(rows[0], ids)
	require.Empty(t, grade.Compute(s, in).Warnings)
	s.Rules.Penalty.Applicable = false
	in.PenaltyEventPoints = dec("0")
	require.Empty(t, grade.Compute(s, in).Warnings, "không có sự kiện PENALTY thì không cảnh báo")
}

func TestBonusNotApplicable(t *testing.T) {
	s, ids, rows := lop1(t)
	s.Rules.Bonus.Applicable = false
	r := grade.Compute(s, inputOf(rows[0], ids)) // sv.gioi: 3,00 điểm cộng
	require.True(t, r.BonusApplied.IsZero())
	require.True(t, r.QT.Equal(dec("9.5")), r.QT.String())
	require.Contains(t, r.Warnings, grade.WarnBonusNotInScheme)
}

// TestParamsDriveResult — AC6: đổi params đổi kết quả mà không sửa mã.
func TestParamsDriveResult(t *testing.T) {
	base, ids1, rows := lop1(t)
	row := rows[1] // sv.kha: vắng 2, phát biểu 0,50
	in := func(ids map[string]uuid.UUID) grade.Input { return inputOf(row, ids) }
	s2, ids2 := buildScheme(t, loadExtraction(t, "extraction-lop1.json"), tweak{weeklyExam: true, weeklyClose: true, rules: func(rows []grade.RuleRow) []grade.RuleRow {
		for i := range rows {
			switch rows[i].Kind {
			case grade.KindBonus:
				rows[i].Params = json.RawMessage(`{"source":"PARTICIPATION","per_speak":"0.25","cap":"1.5"}`)
			case grade.KindPenalty:
				rows[i].Params = json.RawMessage(`{"source":"ABSENCE","free":1,"per":"0.5"}`)
			case grade.KindRounding:
				rows[i].Params = json.RawMessage(`{"step":"0.1","mode":"HALF_UP"}`)
			case grade.KindLetterScale:
				rows[i].Params = json.RawMessage(`{"bands":[{"min":"8","letter":"A"},{"min":"6.5","letter":"B"},{"min":"5","letter":"C"},{"min":"4","letter":"D"},{"min":"0","letter":"F"}]}`)
			}
		}
		return rows
	}})
	r1 := grade.Compute(base, in(ids1))
	r2 := grade.Compute(s2, in(ids2))
	require.True(t, r1.Penalty.IsZero())
	require.True(t, r2.Penalty.Equal(dec("0.5")), "miễn 1 buổi: vắng 2 → trừ 0,5")
	require.True(t, r1.QT.Equal(dec("8.25")) && r2.QT.Equal(dec("7.75")), "%s %s", r1.QT, r2.QT)
	require.True(t, r2.HP.Equal(dec("7")), r2.HP.String()) // HP thô = (40×7,75 + 60×6,5)/100 = 7,00
}

// TestMissingParamIsError — thiếu / lạ / sai kiểu tham số → lỗi rõ ràng, không mặc định thầm lặng.
func TestMissingParamIsError(t *testing.T) {
	for _, c := range []struct {
		name    string
		row     grade.RuleRow
		field   string
		missing bool
	}{
		{"thiếu cap", grade.RuleRow{Kind: grade.KindBonus, Applicable: true, Params: json.RawMessage(`{"source":"PARTICIPATION","per_speak":"0.25"}`)}, "cap", true},
		{"khoá lạ", grade.RuleRow{Kind: grade.KindBonus, Applicable: true, Params: json.RawMessage(`{"source":"PARTICIPATION","per_speak":"0.25","cap":"1","x":1}`)}, "params", false},
		{"cap là số JSON", grade.RuleRow{Kind: grade.KindBonus, Applicable: true, Params: json.RawMessage(`{"source":"PARTICIPATION","per_speak":"0.25","cap":1}`)}, "params", false},
		{"cap âm", grade.RuleRow{Kind: grade.KindBonus, Applicable: true, Params: json.RawMessage(`{"source":"PARTICIPATION","per_speak":"0.25","cap":"-1"}`)}, "cap", false},
		{"bước lạ", grade.RuleRow{Kind: grade.KindRounding, Applicable: true, Params: json.RawMessage(`{"step":"0.3","mode":"HALF_UP"}`)}, "step", false},
		{"thang hở", grade.RuleRow{Kind: grade.KindLetterScale, Applicable: true, Params: json.RawMessage(`{"bands":[{"min":"5","letter":"A"}]}`)}, "bands", false},
		{"thang không giảm", grade.RuleRow{Kind: grade.KindLetterScale, Applicable: true, Params: json.RawMessage(`{"bands":[{"min":"5","letter":"A"},{"min":"6","letter":"B"},{"min":"0","letter":"F"}]}`)}, "bands[1].min", false},
		{"liệt loại lạ", grade.RuleRow{Kind: grade.KindFail, Applicable: true, Params: json.RawMessage(`{"kind":"X","value":"1"}`)}, "kind", false},
		{"free không nguyên", grade.RuleRow{Kind: grade.KindPenalty, Applicable: true, Params: json.RawMessage(`{"source":"ABSENCE","free":"2","per":"0.5"}`)}, "params", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := grade.BuildRules([]grade.RuleRow{c.row})
			var pe *grade.ParamError
			require.ErrorAs(t, err, &pe)
			require.Equal(t, c.field, pe.Field)
			require.Equal(t, c.missing, pe.Missing)
		})
	}
	// luật trùng
	_, err := grade.BuildRules([]grade.RuleRow{
		{Kind: grade.KindRounding, Applicable: true, Params: json.RawMessage(`{"step":"0.5","mode":"HALF_UP"}`)},
		{Kind: grade.KindRounding, Applicable: true, Params: json.RawMessage(`{"step":"0.5","mode":"HALF_UP"}`)}})
	require.Error(t, err)
	// công thức thiếu làm tròn / thang chữ: engine trả INVALID, không đoán
	s, ids, rows := lop1(t)
	s.Rules.Rounding = nil
	r := grade.Compute(s, inputOf(rows[0], ids))
	require.Equal(t, grade.StatusInvalid, r.Status)
	require.Equal(t, "INVALID:ROUNDING_MISSING", r.Verdict)
}

// TestExplainOrderAndQuotes — AC9: thứ tự dòng; mỗi dòng có căn cứ quy chế (trừ mục tự điền).
func TestExplainOrderAndQuotes(t *testing.T) {
	s, ids, rows := lop1(t)
	r := grade.Compute(s, inputOf(rows[0], ids)) // sv.gioi
	var labels []string
	for _, l := range r.Lines {
		labels = append(labels, l.Label)
	}
	require.Equal(t, []string{
		"Giữa kỳ (20%)", "Kiểm tra hằng tuần (20%)", "Thi cuối kỳ (60%)", "Điểm cộng", "Điểm trừ",
		"Điểm quá trình trước điểm cộng / trừ", "Điểm quá trình", "Điểm cuối kỳ", "Điểm học phần thô", "Làm tròn đến 0,5 (nửa lên)", "Điểm chữ"}, labels)
	for _, l := range r.Lines {
		switch l.Label {
		case "Điểm quá trình trước điểm cộng / trừ", "Điểm quá trình", "Điểm cuối kỳ", "Điểm học phần thô":
			continue // phép tính trung gian, không phải mục quy chế
		}
		require.NotNil(t, l.Quote, l.Label)
		require.Positive(t, l.Page, l.Label)
		require.False(t, l.FilledByUser)
	}
}

// TestExplainUserFilledHasNoQuote — mục giảng viên tự điền: quote = nil, filled_by_user = true.
func TestExplainUserFilledHasNoQuote(t *testing.T) {
	s, ids, rows := lop1(t)
	s.Rules.Rounding.FilledByUser = true
	s.Rules.Rounding.Quote, s.Rules.Rounding.Page = "đã bị gỡ khi sửa", 2
	r := grade.Compute(s, inputOf(rows[0], ids))
	for _, l := range r.Lines {
		if strings.HasPrefix(l.Label, "Làm tròn") {
			require.Nil(t, l.Quote)
			require.True(t, l.FilledByUser)
		}
	}
}

// TestExplainExprRecomputesValue — AC9: tính lại expr bằng decimal ra đúng value (HP thô: sau chuẩn hoá Round(10)); dòng HP thô = Result.HPRaw. 30 sinh viên + 18 ca.
func TestExplainExprRecomputesValue(t *testing.T) {
	check := func(t *testing.T, r grade.Result) {
		t.Helper()
		sawRaw := false
		for _, l := range r.Lines {
			if l.Expr == "" {
				continue
			}
			got, want := evalExpr(t, l.Expr), dec(strings.Replace(l.Value, ",", ".", 1))
			if l.Label == "Điểm học phần thô" {
				got = got.Round(grade.NormalizeDigits)
				require.True(t, want.Equal(r.HPRaw), "dòng HP thô ≠ Result.HPRaw")
				sawRaw = true
			}
			require.True(t, got.Equal(want), "%s: %q = %s ≠ %s", l.Label, l.Expr, got, want)
		}
		require.True(t, sawRaw)
	}
	s, ids, rows := lop1(t)
	for _, row := range rows {
		check(t, grade.Compute(s, inputOf(row, ids)))
	}
	for _, c := range readCSV(t, fixDir+"cases-tong-hop.csv") {
		s, in := caseScheme(t, c)
		if r := grade.Compute(s, in); r.Status == grade.StatusComplete {
			check(t, r)
		}
	}
}

var explainHP = regexp.MustCompile(`HP=(\([^=]*\)/100)=([0-9.]+)->`)

// TestExplainFullPrecision — AC9: dòng giải trình đủ chữ số (không làm tròn kiểu chẵn-gần-nhất) và cột `explanation` của đáp án tính lại ra hp_raw ở 30/30 dòng.
func TestExplainFullPrecision(t *testing.T) {
	s, ids, rows := lop1(t)
	want := expected30(t)
	for _, row := range rows {
		w := want[row["account"]]
		m := explainHP.FindStringSubmatch(w["explanation"])
		require.NotNil(t, m, row["account"])
		require.True(t, evalExpr(t, m[1]).Round(grade.NormalizeDigits).Equal(dec(w["hp_raw"])), "%s: explanation %q không ra hp_raw %s", row["account"], m[1], w["hp_raw"])
		r := grade.Compute(s, inputOf(row, ids))
		for _, l := range r.Lines {
			if l.Label == "Điểm quá trình" {
				require.Equal(t, strings.Replace(dec(w["qt"]).String(), ".", ",", 1), l.Value, row["account"])
			}
		}
	}
	// sv04: QT = 6,625 (không phải 6,62 / 6,63)
	for _, row := range rows {
		if row["account"] == "sv04" {
			for _, l := range grade.Compute(s, inputOf(row, ids)).Lines {
				if l.Label == "Điểm quá trình" {
					require.Equal(t, "6,625", l.Value)
				}
			}
		}
	}
}

// TestWhatIfNoWrite — WhatIf không đổi đầu vào, dùng cùng Compute.
func TestWhatIfNoWrite(t *testing.T) {
	s, ids, rows := lop1(t)
	in := inputOf(rows[1], ids) // sv.kha
	before := in.Scores[ids[finalKey]]
	r := grade.WhatIf(s, in, dec("9"))
	require.True(t, in.Scores[ids[finalKey]].Equal(before), "đầu vào không đổi")
	// QT 8,25; HP thô = 0,4×8,25 + 0,6×9 = 8,7 → 8,5
	require.True(t, r.HPRaw.Equal(dec("8.7")) && r.HP.Equal(dec("8.5")), "%s %s", r.HPRaw, r.HP)
	same := grade.WhatIf(s, in, before)
	require.True(t, same.HP.Equal(grade.Compute(s, in).HP))
	require.Equal(t, "LIET:final<3.0", grade.WhatIf(s, in, dec("2.99")).Verdict, "liệt vẫn áp")
}

// TestMinFinalFor — điểm cuối kỳ nhỏ nhất (bước 0,01) để đạt điểm chữ mục tiêu; không đạt được → false.
func TestMinFinalFor(t *testing.T) {
	s, ids, rows := lop1(t)
	b := inputOf(rows[1], ids) // sv.kha: QT 8,25
	got, ok := grade.MinFinalFor(s, b, "A")
	require.True(t, ok)
	require.True(t, got.Equal(dec("8.25")), got.String())
	got, ok = grade.MinFinalFor(s, b, "A+")
	require.True(t, ok)
	require.True(t, got.Equal(dec("9.09")), got.String())
	c := inputOf(rows[2], ids) // sv.nguyco: QT 1,25 → không với tới A+
	_, ok = grade.MinFinalFor(s, c, "A+")
	require.False(t, ok)
	_, ok = grade.MinFinalFor(s, b, "Z")
	require.False(t, ok, "điểm chữ lạ")
	f, ok := grade.MinFinalFor(s, b, "F")
	require.True(t, ok)
	require.True(t, f.Equal(dec("3")), "F: cuối kỳ nhỏ nhất không liệt = 3,00")
	vang := inputOf(rows[1], ids)
	vang.Absent = 7
	_, ok = grade.MinFinalFor(s, vang, "D")
	require.False(t, ok, "bị liệt vì vắng")
}

func BenchmarkCompute1000(b *testing.B) {
	t := &testing.T{}
	s, ids := buildScheme(t, loadExtraction(t, "extraction-lop1.json"), defaultTweak())
	rows := readCSV(t, fixDir+"inputs-lop1.csv")
	ins := make([]grade.Input, 1000)
	for i := range ins {
		ins[i] = inputOf(rows[i%len(rows)], ids)
	}
	b.ResetTimer()
	for range b.N {
		for _, in := range ins {
			_ = grade.Compute(s, in)
		}
	}
}
