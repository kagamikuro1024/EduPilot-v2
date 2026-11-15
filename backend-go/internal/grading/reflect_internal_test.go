package grading

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	xnorm "golang.org/x/text/unicode/norm"

	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

type namePII struct{ names []string }

func (n namePII) Detect(_ context.Context, _ uuid.UUID, text string) ([]privacy.Finding, error) {
	for _, s := range n.names {
		if strings.Contains(text, s) {
			return []privacy.Finding{{}}, nil
		}
	}
	return nil, nil
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func caseRows(t *testing.T, comment string, ai, fin map[string]string) store.GuidanceCasesRow {
	t.Helper()
	var cs []map[string]any
	for id, s := range fin {
		cs = append(cs, map[string]any{"id": id, "score": s, "comment": "SỬA TAY " + comment})
	}
	c, _ := json.Marshal(cs)
	a, _ := json.Marshal(ai)
	return store.GuidanceCasesRow{ID: uuid.New(), Criteria: c, AiScores: a, AiComments: json.RawMessage(`{"criteria":{"c1":"AI nói c1","c2":"AI nói c2"},"overall":"x"}`)}
}

// TestReflectEligibility — SRS 4.2.5: đếm kết quả và điểm tiêu chí lệch ≥ MIN_DELTA bằng decimal (0,5 đúng ngưỡng vẫn tính; 0,49 thì không).
func TestReflectEligibility(t *testing.T) {
	rows := []store.GuidanceCasesRow{
		caseRows(t, "", map[string]string{"c1": "2.00", "c2": "1.00"}, map[string]string{"c1": "1.50", "c2": "1.00"}), // c1 lệch 0,50
		caseRows(t, "", map[string]string{"c1": "2.00", "c2": "1.00"}, map[string]string{"c1": "1.51", "c2": "1.00"}), // c1 lệch 0,49
		caseRows(t, "", map[string]string{"c1": "2.00", "c2": "1.00"}, map[string]string{"c1": "2.00", "c2": "2.00"}), // c2 lệch 1,00
	}
	res, edits := Eligibility(rows, dec("0.5"))
	require.Equal(t, 3, res)
	require.Equal(t, 2, edits)
}

// TestReflectInputNoSubmissionText — SRS 4.2.6 / Q-G-10: đầu vào chỉ có điểm + nhận xét GỐC của AI; nhận xét sửa tay của giảng viên (criteria[].comment) không lọt vào.
func TestReflectInputNoSubmissionText(t *testing.T) {
	rubric := []RubricItem{{ID: "c1", Name: "Một", MaxScore: "2.5", Guidance: "g1"}, {ID: "c2", Name: "Hai", MaxScore: "2.5", Guidance: "g2"}}
	rows := []store.GuidanceCasesRow{caseRows(t, "CANARY_SUA_TAY", map[string]string{"c1": "2.00", "c2": "1.00"}, map[string]string{"c1": "1.00", "c2": "1.00"})}
	in, refs := buildReflectInput(rows, rubric, nil, dec("0.5"), 60)
	raw, _ := json.Marshal(in)
	require.NotContains(t, string(raw), "CANARY_SUA_TAY")
	require.NotContains(t, string(raw), rows[0].ID.String(), "id kết quả không ra khỏi máy chủ; case_id là ngẫu nhiên theo lần chạy")
	require.Len(t, in.Cases, 1)
	require.Equal(t, "AI nói c1", in.Cases[0].AIComment)
	require.Equal(t, rows[0].ID, refs[in.Cases[0].CaseID].ResultID)
	in2, _ := buildReflectInput(rows, rubric, nil, dec("0.5"), 60)
	require.NotEqual(t, in.Cases[0].CaseID, in2.Cases[0].CaseID, "case_id đổi mỗi lần chạy")
	in3, _ := buildReflectInput(rows, rubric, nil, dec("0.5"), 0)
	require.Empty(t, in3.Cases, "trần số ca")
}

// TestReflectValidate — SRS 4.2.8: bảng theo từng bước; support do code đếm số ca hợp lệ cùng chiều, không tin số LLM khai.
func TestReflectValidate(t *testing.T) {
	rubric := []RubricItem{{ID: "c1"}, {ID: "c2"}}
	r1, r2, r3 := uuid.New(), uuid.New(), uuid.New()
	refs := map[string]caseRef{
		"a": {ResultID: r1, CriterionID: "c1", AI: dec("2.00"), Teacher: dec("1.00")}, // AI cao hơn: STRICTER
		"b": {ResultID: r2, CriterionID: "c1", AI: dec("2.50"), Teacher: dec("1.50")}, // STRICTER
		"c": {ResultID: r3, CriterionID: "c1", AI: dec("1.00"), Teacher: dec("2.00")}, // LENIENT
		"d": {ResultID: r3, CriterionID: "c2", AI: dec("2.00"), Teacher: dec("1.00")},
	}
	existing := []GuidanceLine{{CriterionID: "c1", Text: "Chỉ cho điểm tối đa khi có ví dụ thực tế"}}
	cfg := DefaultGuidanceConfig()
	pii := namePII{names: []string{"Nguyễn Văn A"}}
	cases := []struct {
		name string
		it   rawItem
		drop string // trường ReflectDropped được tăng; "" = giữ
		sup  int
	}{
		{"hợp lệ", rawItem{"c1", "Chấm chặt hơn khi chỉ nêu tên mà không phân tích", "STRICTER", []string{"a", "b"}}, "", 2},
		{"tiêu chí lạ", rawItem{"c9", "Lưu ý", "STRICTER", []string{"a", "b"}}, "criterion", 0},
		{"case_id lạ", rawItem{"c1", "Lưu ý", "STRICTER", []string{"a", "zzz"}}, "case", 0},
		{"case khác tiêu chí", rawItem{"c1", "Lưu ý", "STRICTER", []string{"a", "d"}}, "case", 0},
		{"sai chiều: 1 ca đúng 1 sai → support 1", rawItem{"c1", "Chấm chặt hơn khi thiếu ví dụ", "STRICTER", []string{"a", "c"}}, "support", 0},
		{"CLARIFY không ràng buộc chiều", rawItem{"c1", "Làm rõ: phân tích là nêu nguyên nhân", "CLARIFY", []string{"a", "c"}}, "", 2},
		{"case_id trùng đếm một lần", rawItem{"c1", "Chấm chặt hơn khi thiếu ví dụ", "STRICTER", []string{"a", "a"}}, "support", 0},
		{"từ khoá: trừ điểm muộn", rawItem{"c1", "Nhớ trừ điểm nếu nộp muộn", "STRICTER", []string{"a", "b"}}, "keyword", 0},
		{"từ khoá: tổng điểm", rawItem{"c1", "Tổng điểm không quá 10", "STRICTER", []string{"a", "b"}}, "keyword", 0},
		{"cụm vô hại: tổng quát / ngoại trừ / loại trừ", rawItem{"c1", "Nhận xét tổng quát không đủ; ngoại trừ ví dụ cụ thể, loại trừ chung chung", "STRICTER", []string{"a", "b"}}, "", 2},
		{"cụm vô hại nhưng còn từ khoá thật", rawItem{"c1", "Nhận xét tổng quát nhưng nhớ trừ điểm", "STRICTER", []string{"a", "b"}}, "keyword", 0},
		{"'trừu tượng' không khớp 'trừ'", rawItem{"c1", "Tránh nhận xét trừu tượng, nêu bằng chứng cụ thể", "STRICTER", []string{"a", "b"}}, "", 2},
		{"PII: tên trong roster", rawItem{"c1", "Bài của Nguyễn Văn A hay bị chấm cao", "STRICTER", []string{"a", "b"}}, "pii", 0},
		{"trùng (Jaccard ≥ 0,8)", rawItem{"c1", "Chỉ cho điểm tối đa khi có ví dụ thực tế rõ", "STRICTER", []string{"a", "b"}}, "duplicate", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, drop := validateItems(t.Context(), []rawItem{tc.it}, refs, rubric, existing, pii, uuid.New(), cfg)
			var want ReflectDropped
			if tc.drop == "criterion" {
				want.add(tc.drop, "") // tiêu chí lạ: không thuộc rubric nên criterion_id = null
				require.Empty(t, got)
			} else if tc.drop != "" {
				want.add(tc.drop, tc.it.CriterionID)
				require.Empty(t, got)
			} else {
				require.Len(t, got, 1)
				require.Equal(t, tc.sup, got[0].Support, "support do code đếm")
			}
			require.Equal(t, want, drop)
		})
	}
	t.Run("overflow cắt còn MAX_ITEMS", func(t *testing.T) {
		c2 := cfg
		c2.MaxItems = 1
		got, drop := validateItems(t.Context(), []rawItem{
			{"c1", "Chấm chặt hơn khi thiếu ví dụ", "STRICTER", []string{"a", "b"}},
			{"c1", "Phân biệt nêu tên với phân tích sâu", "STRICTER", []string{"a", "b"}},
		}, refs, rubric, nil, pii, uuid.New(), c2)
		require.Len(t, got, 1)
		require.Equal(t, 1, drop.Count("overflow"))
	})
	t.Run("hai mục trùng nhau trong cùng lượt", func(t *testing.T) {
		got, drop := validateItems(t.Context(), []rawItem{
			{"c1", "Chấm chặt hơn khi thiếu ví dụ cụ thể", "STRICTER", []string{"a", "b"}},
			{"c1", "Chấm chặt hơn khi thiếu ví dụ cụ thể", "STRICTER", []string{"a", "b"}},
		}, refs, rubric, nil, pii, uuid.New(), cfg)
		require.Len(t, got, 1)
		require.Equal(t, 1, drop.Count("duplicate"))
	})
}

// TestReflectDroppedByCriterion — P-106-12: job result ghi số mục bị bỏ THEO tiêu chí và lý do (không chữ), để giảng viên thấy lưu ý c2 bị loại vì từ khoá.
func TestReflectDroppedByCriterion(t *testing.T) {
	rubric := []RubricItem{{ID: "c1"}, {ID: "c2"}}
	refs := map[string]caseRef{
		"a": {ResultID: uuid.New(), CriterionID: "c1", AI: dec("2.00"), Teacher: dec("1.00")},
		"b": {ResultID: uuid.New(), CriterionID: "c1", AI: dec("2.50"), Teacher: dec("1.50")},
		"c": {ResultID: uuid.New(), CriterionID: "c2", AI: dec("2.00"), Teacher: dec("1.00")},
		"d": {ResultID: uuid.New(), CriterionID: "c2", AI: dec("2.00"), Teacher: dec("1.25")},
	}
	got, drop := validateItems(t.Context(), []rawItem{
		{"c1", "Chấm chặt hơn khi thiếu ví dụ cụ thể", "STRICTER", []string{"a", "b"}},
		{"c2", "Tính tổng điểm của các ý trước", "STRICTER", []string{"c", "d"}},
		{"c9", "Lưu ý", "STRICTER", []string{"a", "b"}},
	}, refs, rubric, nil, namePII{}, uuid.New(), DefaultGuidanceConfig())
	require.Len(t, got, 1)
	c2 := "c2"
	require.ElementsMatch(t, ReflectDropped{{CriterionID: &c2, Reason: "keyword", Count: 1}, {Reason: "criterion", Count: 1}}, drop, "criterion_id = null khi reason = criterion")
	require.Equal(t, ReflectDropped{{CriterionID: &c2, Reason: "keyword", Count: 1}, {Reason: "criterion", Count: 1}}, drop.sorted(), "tiêu chí trước, null cuối")
	require.Equal(t, 2, drop.Total())
	b, err := json.Marshal(ReflectResult{Drafts: 1, Dropped: drop.sorted()})
	require.NoError(t, err)
	require.JSONEq(t, `{"drafts":1,"dropped":[{"criterion_id":"c2","reason":"keyword","count":1},{"criterion_id":null,"reason":"criterion","count":1}]}`, string(b))
	b, err = json.Marshal(ReflectResult{Dropped: ReflectDropped(nil).sorted()})
	require.NoError(t, err)
	require.JSONEq(t, `{"drafts":0,"dropped":[]}`, string(b), "không có mục nào bị bỏ = mảng rỗng, không null")
}
func TestReflectKeywordExempt(t *testing.T) {
	t.Parallel()
	exempt := []string{ // KHÔNG bị loại
		"Tổng quát hoá luận điểm trước khi nêu ví dụ", "Mở bài nêu tổng quan hiện trạng", "Trình bày tổng thể kiến trúc phòng thủ",
		"Lập luận tổng hợp nhiều nguồn", "Tổng kết rồi mới kiến nghị", "Chỉ cho điểm tối đa khi có số liệu, trừ khi bài nêu lý do",
		"Ngoại trừ phần mở bài", "Loại trừ các giả định không có căn cứ",
		"TỔNG QUÁT", "tong quat", "(tổng quan)", "tổng quát,", "tổng quát.", xnorm.NFD.String("tổng quát"),
		"tổng quát tổng hợp", "tổng quát, tổng kết", "tổng quan tổng thể", // hai cụm liền nhau dùng chung ranh giới
		"Ý trừu tượng nhưng có ví dụ", // «trừu» không phải từ khoá «trừ»
	}
	for _, s := range exempt {
		require.False(t, ForbiddenText(s), "phải được miễn: %q", s)
	}
	forbiddenCases := []string{ // VẪN bị loại
		"Tính tổng điểm của các ý", "Trừ điểm khi thiếu ví dụ", "Bài nộp muộn thì hạ bậc", "Làm tròn lên 0,5", "Phạt 10 %",
		"tổng", "trừ", "Tổng của các ý", "tổng quá", "trừ đi 0,5", "tổng đài viên",
		"Tổng quát nhưng đừng tính tổng điểm", "tổng quan và trừ 0,5", "trừ khi nộp muộn", "ngoại trừ phần phạt",
		"tổng kết rồi làm tròn", "loại trừ điểm thưởng và phạt", "tổng quát tổng điểm",
	}
	for _, s := range forbiddenCases {
		require.True(t, ForbiddenText(s), "phải bị loại: %q", s)
	}
}
