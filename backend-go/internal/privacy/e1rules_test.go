package privacy

import "testing"

// Luật bổ sung từ E1 (US-P3-07): tên đảo tuỳ ý, MSSV gõ tách giữa, "tối/tới" không phải đại từ, câu cá nhân về lịch thi / điểm cộng / điều kiện.
func TestE1Rules(t *testing.T) {
	t.Parallel()
	idx := buildIndex([]Member{{Name: "Dương Quốc Hương", Code: "20224786"}})
	kinds := func(text string) []Kind {
		var ks []Kind
		for _, m := range scan(text, idx) {
			ks = append(ks, m.Kind)
		}
		return ks
	}
	for name, tc := range map[string]struct {
		text string
		want []Kind
	}{
		"tên đảo hoàn toàn":     {"Thầy ơi, Hương Quốc Dương chưa hiểu AES", []Kind{KindName}},
		"tên đảo giữa":          {"bạn quoc huong duong hỏi về RSA", []Kind{KindName}},
		"MSSV tách bằng dấu .":  {"em 2022.4786 hỏi", []Kind{KindMSSV}},
		"MSSV tách bằng cách":   {"em 2022 4786 hỏi", []Kind{KindMSSV}},
		"số tách không phải mã": {"năm học 2022 2023 có gì", nil},
		"MSSV tách sau từ khoá": {"MSSV 2022.9999 hỏi", []Kind{KindMSSV}},
		"ms sv + mã tách":       {"ms sv 2099 1234 xin hỏi", []Kind{KindMSSV}},
	} {
		got := kinds(tc.text)
		if len(got) != len(tc.want) || (len(got) > 0 && got[0] != tc.want[0]) {
			t.Errorf("%s: %v, muốn %v", name, got, tc.want)
		}
	}
	for text, want := range map[string]bool{
		"Quy tắc nguyên tắc đặc quyền tối thiểu nghĩa là gì?": false, // "tối thiểu" ≠ "tôi thiếu"
		"toi thieu diem danh":                     true, // gõ không dấu: đại từ
		"Em thi cuối kỳ phòng nào?":               true,
		"Mình thi cuối kỳ lúc mấy giờ vậy":        true,
		"em có phải thi lại không và thi khi nào": true,
		"Khi nào thi cuối kỳ?":                    false,
		"Thi lại được mấy lần theo quy chế":       false,
		"em phát biểu 3 lần được cộng bao nhiêu":  true,
		"em có đủ điều kiện dự thi không":         true,
		"Điều kiện dự thi cuối kỳ là gì":          false,
		"Trường hợp của em có được thi lại không": true,
	} {
		if got := IsPersonalPattern(text); got != want {
			t.Errorf("IsPersonalPattern(%q) = %v, muốn %v", text, got, want)
		}
	}
}
