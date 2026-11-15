package insight

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
)

// TestLeaksRules — FR-29: ≥ 6 từ liên tiếp (bỏ dấu, chữ thường, tách theo ký tự không phải chữ / số); câu dưới ngưỡng 4–5 từ trùng khi bị chép nguyên; câu < 4 từ bỏ; why cũng được kiểm.
func TestLeaksRules(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	items := []item{
		{student: a, text: "Em bị ốm tuần sau xin hoãn bài được không"}, // 1 SV, 9 từ
		{student: a, text: "Phòng 302 khu B KTX"},                       // 1 SV, 5 từ (ngắn)
		{student: a, text: "Xin chào thầy ạ"},                           // 4 từ
		{student: a, text: "Chào thầy"},                                 // 2 từ: bỏ
		{student: a, text: "Chủ đề chung cả lớp đều hỏi nhiều lần nhé"}, // đạt ngưỡng (3 SV) → không vào below
		{student: b, text: "Chủ đề chung cả lớp đều hỏi nhiều lần nhé"},
		{student: c, text: "Chủ đề chung cả lớp đều hỏi nhiều lần nhé"},
	}
	below := belowGrams(items)
	cases := []struct {
		name, text string
		want       bool
	}{
		{"6 từ liên tiếp", "Gợi ý: em bị ốm tuần sau xin", true},
		{"6 từ, khác dấu / hoa thường", "EM BI OM TUAN SAU XIN", true},
		{"5 từ liên tiếp", "em bị ốm tuần sau", false},
		{"câu ngắn 5 từ bị chép nguyên", "Nên xem lại: phòng 302 khu b ktx nhé", true},
		{"câu ngắn 5 từ chép thiếu", "phòng 302 khu b", false},
		{"câu 4 từ chép nguyên", "Xin chào thầy ạ, ôn lại chương 2", true},
		{"câu 2 từ bị bỏ", "Chào thầy, ôn lại chương 2", false},
		{"câu đạt ngưỡng được dùng", "chủ đề chung cả lớp đều hỏi nhiều lần nhé", false},
		{"rỗng", "", false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, leaks(c.text, below), c.name)
	}
	pl := &Payload{below: below, Topics: []Topic{
		{Rank: 1, Name: "Ổn", Why: "3 sinh viên hỏi", Recommendation: "Ôn lại", NamedBy: namedByLLM},
		{Rank: 2, Name: "Tên", Why: "em bị ốm tuần sau xin hoãn", Recommendation: "x", NamedBy: namedByLLM}, // why chép
		{Rank: 3, Name: "Tên", Why: "ok", Recommendation: "phòng 302 khu b ktx", NamedBy: namedByLLM},       // khuyến nghị chép
	}}
	guardLeaks(pl)
	require.Equal(t, "Ổn", pl.Topics[0].Name)
	for _, i := range []int{1, 2} {
		tp := pl.Topics[i]
		require.Equal(t, "Chủ đề "+string(rune('0'+tp.Rank)), tp.Name)
		require.Equal(t, namedByFallback, tp.NamedBy)
		require.Empty(t, tp.Why)
		require.Empty(t, tp.Recommendation)
	}
}

// TestDemoNameRecording — DEMO_SCRIPT tầng 2: bản ghi demo cho bước NAME trả danh sách rỗng hợp schema → báo cáo vẫn xong, tên chủ đề là tên tạm theo từ khoá (không «mẫu»).
func TestDemoNameRecording(t *testing.T) {
	raw, err := os.ReadFile("../../../seed/demo/fake_llm.json")
	require.NoError(t, err)
	var f struct {
		Recordings []struct {
			Name     string
			Match    []string
			Response json.RawMessage
		}
	}
	require.NoError(t, json.Unmarshal(raw, &f))
	found := false
	for _, r := range f.Recordings {
		if r.Name != "insight-name-fallback" {
			continue
		}
		found = true
		for _, m := range r.Match {
			require.Contains(t, nameSystem, m)
		}
		require.NoError(t, llm.ValidateJSON(json.RawMessage(nameSchema), r.Response))
	}
	require.True(t, found)
}
