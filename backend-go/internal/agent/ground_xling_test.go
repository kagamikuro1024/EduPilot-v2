package agent

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/rag"
)

type xlingRow struct {
	Q      string  `json:"q"`
	Answer string  `json:"answer"`
	Conf   float64 `json:"conf"`
	Hits   []struct {
		N       int     `json:"n"`
		Title   string  `json:"title"`
		Text    string  `json:"text"`
		Heading *string `json:"heading"`
	} `json:"hits"`
}

func (r xlingRow) hits() []rag.Hit {
	max := 0
	for _, h := range r.Hits {
		max = max2(max, h.N)
	}
	out := make([]rag.Hit, max)
	for _, h := range r.Hits {
		out[h.N-1] = rag.Hit{Title: h.Title, Heading: h.Heading, Text: h.Text}
	}
	return out
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 9 câu trả lời THẬT của ep-qc (QC đợt R, P-11-18): nguồn tiếng Anh, trả lời tiếng Việt đúng, bị cờ thấp vì ground ≈ 0 (retr đã = 1,0).
// P-11-21: 8 câu đúng phải đạt ≥ 0,75 với retr = 1; câu «Spear phishing» là câu tự thú «ngữ cảnh chưa đủ thông tin» phải GIỮ cờ thấp.
func TestGroundednessCrossLingualRealAnswers(t *testing.T) {
	b, err := os.ReadFile("testdata/ground_xling.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []xlingRow
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 9 {
		t.Fatalf("fixture: %d dòng, cần 9", len(rows))
	}
	threshold := decimal.New(75, -2)
	for _, r := range rows {
		g := Groundedness(r.Answer, r.hits())
		score := Score(decimal.NewFromInt(1), g)
		t.Logf("%.3f→%s  ground=%s  %.40s", r.Conf, score, g, r.Q)
		hedgedAnswer := hedged(r.Answer)
		if hedgedAnswer {
			if score.GreaterThanOrEqual(threshold) {
				t.Errorf("%q: câu tự thú thiếu thông tin mà score %s ≥ 0,75 (ground %s)", r.Q, score, g)
			}
			continue
		}
		if score.LessThan(threshold) {
			t.Errorf("%q: câu đúng bị cờ thấp: ground %s score %s (đã lưu %v)", r.Q, g, score, r.Conf)
		}
	}
}

func hitsOf(texts ...string) []rag.Hit {
	out := make([]rag.Hit, len(texts))
	for i, tx := range texts {
		out[i] = rag.Hit{Title: "Network Security Threats", Text: tx}
	}
	return out
}

func TestGroundednessAnchorFabrications(t *testing.T) {
	src := "A DDoS attack sends extremely large numbers of requests over a network. Exploits crash applications and processes on 2018 servers."
	other := "Phishing emails impersonate banks to steal credentials."
	cases := []struct {
		name   string
		answer string
		hits   []rag.Hit
		want   string
	}{
		{"đúng: thuật ngữ + số trùng đoạn được trích", "Tấn công khiến ứng dụng và tiến trình bị sập do exploits trên các máy chủ năm 2018 [1].", hitsOf(src), "1"},
		{"bịa: không mỏ neo nào trùng đoạn", "Tấn công này luôn do chính phủ nước ngoài thực hiện nhằm đánh cắp mật khẩu ngân hàng của sinh viên [1].", hitsOf(src), "0"},
		{"bịa: mỏ neo trùng đoạn KHÁC đoạn được trích", "Tấn công khiến ứng dụng và tiến trình bị sập do exploits trên các máy chủ năm 2018 [2].", hitsOf(src, other), "0"},
		{"bịa: không có dấu [n]", "Tấn công khiến ứng dụng và tiến trình bị sập do exploits trên các máy chủ năm 2018.", hitsOf(src), "0"},
		{"bịa: [n] ngoài danh sách đoạn", "Tấn công khiến ứng dụng và tiến trình bị sập do exploits trên các máy chủ năm 2018 [7].", hitsOf(src), "0"},
		{"tự thú thiếu thông tin: tắt mỏ neo dù trùng thuật ngữ", "Ngữ cảnh chưa đủ thông tin, nhưng tấn công khiến ứng dụng bị sập do exploits trên máy chủ năm 2018 [1].", hitsOf(src), "0"},
		{"danh sách [1, 2]: mỏ neo ở đoạn thứ hai", "Tấn công khiến ứng dụng và tiến trình bị sập do exploits trên các máy chủ năm 2018 [2, 1].", hitsOf(other, src), "1"},
		{"chỉ âm tiết Việt không dấu trùng đoạn: không phải mỏ neo", "Trong khi đó theo sau thanh chung nhanh trong cho theo khi sau [1].", hitsOf("trong theo khi cho sau thanh chung nhanh trung tinh"), "0"},
	}
	for _, c := range cases {
		if got := Groundedness(c.answer, c.hits).String(); got != c.want {
			t.Errorf("%s: ground %s, muốn %s", c.name, got, c.want)
		}
	}
}

func TestAnchors(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"exploits DoS 2018 wpa3 ipv6 Internet", []string{"exploits", "2018", "wpa3", "ipv6", "internet"}},
		{"trong theo khi cho sau thanh chung nhanh trung tinh minh quan nghe nghi khoa hoang", nil}, // âm tiết Việt thanh ngang, không dấu
		{"Tấn công từ chối dịch vụ 7 lớp", nil},                                                   // chữ có dấu không phải mỏ neo; số 1 chữ số quá ngắn
	}
	for _, c := range cases {
		if got := anchors(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("anchors(%q) = %v, muốn %v", c.in, got, c.want)
		}
	}
}

func TestExtractCitationsList(t *testing.T) {
	hits := hitsOf("a", "b")
	out, cites := ExtractCitations("Một [1, 2] hai [2, 9] ba [9] bốn.", hits)
	if out != "Một [1] [2] hai [2] ba bốn." {
		t.Errorf("văn bản: %q", out)
	}
	if len(cites) != 2 || cites[0].N != 1 || cites[1].N != 2 {
		t.Errorf("citations: %+v", cites)
	}
	if !hasCitation("Có bốn nhóm [2, 5, 7]:", 2) || hasCitation("Có bốn nhóm [5, 7]:", 2) {
		t.Error("hasCitation không đọc danh sách")
	}
}
