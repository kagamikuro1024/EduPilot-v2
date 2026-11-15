package agent

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/rag"
)

var ctxHit = rag.Hit{Title: "Quy chế học vụ", Text: "Sinh viên bị cảnh báo học vụ khi điểm trung bình tích lũy dưới 1,2 ở học kỳ đầu. Sinh viên được thi lại tối đa một lần đối với mỗi học phần."}

const (
	supported   = "Sinh viên được thi lại tối đa một lần đối với mỗi học phần."
	unsupported = "Cổng mạng hai nghìn bốn mươi tám bit mã hóa không đối xứng."
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func withCos(c float64) []rag.Hit {
	h := ctxHit
	h.Cosine = c
	return []rag.Hit{h}
}

// TestConfidenceFormula — AC1: bảng đầu vào cố định → giá trị chính xác đến 3 chữ số (confidence = 0,6·retr + 0,4·ground).
func TestConfidenceFormula(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		cos          float64
		answer       string
		retr, ground string
		want         string
	}{
		"cos đúng sàn, có căn cứ":      {0.25, supported, "0.000", "1.000", "0.400"},
		"giữa sàn và trần":             {0.45, supported, "0.500", "1.000", "0.700"},
		"đúng trần":                    {0.65, supported, "1.000", "1.000", "1.000"},
		"trên trần bị kẹp":             {0.90, supported, "1.000", "1.000", "1.000"},
		"retr 0,8":                     {0.57, supported, "0.800", "1.000", "0.880"},
		"retr 0,25":                    {0.35, supported, "0.250", "1.000", "0.550"},
		"trả lời không đủ 4 từ":        {0.65, "Có.", "1.000", "0.000", "0.600"},
		"trả lời rỗng":                 {0.65, "", "1.000", "0.000", "0.600"},
		"nhận định không có căn cứ":    {0.45, unsupported, "0.500", "0.000", "0.300"},
		"một đúng một sai":             {0.45, supported + " " + unsupported, "0.500", "0.500", "0.500"},
		"dấu [n] không tính là từ":     {0.65, "Sinh viên được thi lại tối đa một lần [1] đối với mỗi học phần [2].", "1.000", "1.000", "1.000"},
		"đúng ngưỡng 40 % (2/5 từ)":    {0.55, "Học kỳ ôn tập đều.", "0.750", "1.000", "0.850"},
		"dưới ngưỡng 40 % (1/5 từ)":    {0.55, "Học nhóm ôn tập đều.", "0.750", "0.000", "0.450"},
		"ngữ cảnh không dấu vẫn khớp":  {0.65, "SINH VIEN DUOC THI LAI TOI DA MOT LAN", "1.000", "1.000", "1.000"},
		"xuống dòng tách nhận định":    {0.65, supported + "\n" + unsupported, "1.000", "0.500", "0.800"},
		"cos dưới sàn vẫn chuẩn hoá 0": {0.10, supported, "0.000", "1.000", "0.400"},
	} {
		retr := Retrieval(tc.cos)
		ground := Groundedness(tc.answer, withCos(tc.cos))
		require.Equal(t, tc.retr, retr.StringFixed(3), name+" retr")
		require.Equal(t, tc.ground, ground.StringFixed(3), name+" ground")
		require.Equal(t, tc.want, Score(retr, ground).StringFixed(3), name+" confidence")
	}
}

// TestGroundZeroWhenNoClaim — AC1 (#11): không câu nào đủ 4 từ nội dung → ground = 0 (không có căn cứ để chống đỡ).
func TestGroundZeroWhenNoClaim(t *testing.T) {
	t.Parallel()
	for _, a := range []string{"", "Có.", "Đúng vậy.", "Được rồi nhé!", "Không. Có. Vâng.", "[1]", "   "} {
		require.True(t, Groundedness(a, withCos(0.9)).IsZero(), "%q", a)
	}
}

// TestConfidenceToolIntent — AC1: intent có tool trả dữ liệu → 1,000 (không phụ thuộc truy xuất); câu mẫu / không dữ liệu → NULL, không "chưa chắc".
func TestConfidenceToolIntent(t *testing.T) {
	t.Parallel()
	hi := dec("0.95")
	m := Outcome{Plan: IntentGrade, Blocks: []Block{{Kind: "grade_summary"}}}.Metadata("Điểm của bạn là 8,5.", hi, false, 2)
	require.Equal(t, "1.000", m.Confidence.Decimal.StringFixed(3))
	require.True(t, m.Confidence.Valid)
	require.False(t, m.LowConfidence, "1,000 không dưới bất kỳ ngưỡng nào")
	require.Equal(t, 2, m.MaskedCount)
	for _, o := range []Outcome{{Plan: IntentGrade, Canned: ReplyNoData("điểm")}, {Plan: IntentSmalltalk}, {Plan: IntentCrisis, Canned: "x"}, {Plan: IntentOtherPerson, Canned: ReplyOtherPerson}} {
		m := o.Metadata("x", hi, false, 0)
		require.False(t, m.Confidence.Valid, o.Plan)
		require.False(t, m.LowConfidence, o.Plan)
	}
}

// TestLowConfidenceUsesCourseThreshold — AC2: low_confidence = confidence < ngưỡng của lớp; đúng ngưỡng thì chưa "thấp"; đổi ngưỡng có hiệu lực ngay.
func TestLowConfidenceUsesCourseThreshold(t *testing.T) {
	t.Parallel()
	o := Outcome{Plan: IntentCourseQA, Hits: withCos(0.45)} // confidence 0,700
	for thr, want := range map[string]bool{"0.50": false, "0.70": false, "0.71": true, "0.80": true, "0.95": true} {
		m := o.Metadata(supported+" [1]", dec(thr), false, 0) // có dấu [1]: P-11-18 (d) hạ ground về 0 nếu không dẫn nguồn
		require.Equal(t, "0.700", m.Confidence.Decimal.StringFixed(3))
		require.Equal(t, want, m.LowConfidence, "ngưỡng "+thr)
	}
}

// TestNoContextCannedReply — AC6: COURSE_QA mà truy xuất không có đoạn nào trên sàn → câu mẫu, 0 lời gọi LLM, no_context + low_confidence.
func TestNoContextCannedReply(t *testing.T) {
	t.Parallel()
	for name, hits := range map[string][]rag.Hit{"không có đoạn": nil, "dưới sàn": withCos(0.10)} {
		r := newRig(t, false, 1)
		r.r.hits = hits
		out, err := r.a.Respond(context.Background(), r.tc, Input{Text: "Giao thức Dijkstra tối ưu như thế nào trong định tuyến"})
		require.NoError(t, err, name)
		require.Equal(t, "Mình chưa tìm thấy nội dung này trong tài liệu của lớp.", out.Canned, name)
		require.True(t, out.NoContext, name)
		require.Zero(t, r.g.calls.Load(), name+": 0 lời gọi LLM")
		m := out.Metadata(out.Canned, dec("0.80"), false, 0)
		require.True(t, m.NoContext && m.LowConfidence, name)
		require.True(t, m.Confidence.Valid, name)
	}
}

// TestConfidenceNoExtraLLMCall — AC1: một câu hỏi COURSE_QA = đúng MỘT lần sinh chữ; tính siêu dữ liệu không gọi LLM / nhúng thêm.
func TestConfidenceNoExtraLLMCall(t *testing.T) {
	t.Parallel()
	r := newRig(t, false, 1)
	out, err := r.a.Respond(context.Background(), r.tc, Input{Text: "Giải thích thuật toán Dijkstra giúp mình"})
	require.NoError(t, err)
	require.NotNil(t, out.Stream)
	for range out.Stream {
	}
	embeds, gens := r.embeds.Load(), r.g.calls.Load()
	m := out.Metadata("ok", dec("0.80"), false, 0)
	require.True(t, m.Confidence.Valid)
	require.Equal(t, embeds, r.embeds.Load(), "không nhúng thêm")
	require.Equal(t, gens, r.g.calls.Load())
	require.Equal(t, int64(1), gens, "đúng một lần sinh chữ")
}

// TestRefusalAndNoCitationAreLowConfidence — P-11-18 (D-R-07): lời từ chối và câu trả lời không dẫn nguồn nào KHÔNG được coi là tin cậy,
// dù truy xuất có hit cao (ep-qc: gemini-embedding-001 cho cos_top1 ≥ 0,55 với câu ngoài chủ đề, confidence 0,855, low_confidence=false).
func TestRefusalAndNoCitationAreLowConfidence(t *testing.T) {
	t.Parallel()
	o := Outcome{Plan: IntentCourseQA, Hits: withCos(0.57)} // retr 0,800: chỉ cần ground = 1 là 0,880 ≥ 0,80
	for name, tc := range map[string]struct {
		answer     string
		wantLow    bool
		wantGround string
	}{
		"từ chối đúng câu mẫu mở đầu":       {"Mình chưa tìm thấy nội dung này trong tài liệu của lớp. Bạn thử hỏi cách khác nhé.", true, "0.000"},
		"từ chối lẫn trong câu, không dấu":  {"Chao ban, minh chua tim thay noi dung nay trong tai lieu cua lop.", true, "0.000"},
		"từ chối tự do, không dẫn nguồn":    {"Hiện tại trong ngữ cảnh được cung cấp không có thông tin về cách nấu phở bò Hà Nội nên mình chưa đủ thông tin để trả lời.", true, "0.000"},
		"trả lời thật nhưng quên dẫn nguồn": {supported, true, "0.000"},
		"trả lời thật có dẫn nguồn":         {supported + " [1]", false, "1.000"},
		"[n] lạ không tính là dẫn nguồn":    {supported + " [7]", true, "0.000"},
	} {
		m := o.Metadata(tc.answer, dec("0.80"), false, 0)
		require.Equal(t, tc.wantLow, m.LowConfidence, name)
		require.Equal(t, tc.wantGround, m.Groundedness.StringFixed(3), name)
		ok, origin := m.ShouldEscalate(IntentCourseQA)
		require.Equal(t, tc.wantLow, ok, name)
		if ok {
			require.Equal(t, OriginAutoLowConfidence, origin, name)
		}
	}
	// Lời từ chối luôn thấp, kể cả khi giảng viên hạ ngưỡng xuống 0,50 và truy xuất cao.
	m := Outcome{Plan: IntentCourseQA, Hits: withCos(0.90)}.Metadata(ReplyNoContext, dec("0.50"), false, 0)
	require.True(t, m.LowConfidence)
	// Câu từ chối không làm hỏng nhận định có căn cứ đi kèm.
	require.Equal(t, "1.000", Groundedness(ReplyNoContext+" "+supported, withCos(0.9)).StringFixed(3))
	require.True(t, IsRefusal("MÌNH CHƯA TÌM THẤY NỘI DUNG NÀY TRONG TÀI LIỆU CỦA LỚP"))
	require.False(t, IsRefusal(supported))
}

// SystemPrompt phải chứa nguyên văn câu mẫu: IsRefusal chỉ bắt được lời từ chối nếu mô hình được dặn dùng đúng câu đó.
func TestSystemPromptCarriesRefusalSentence(t *testing.T) {
	t.Parallel()
	require.Contains(t, SystemPrompt, ReplyNoContext)
}

// D-R-09 (QC đợt R): câu trả lời THẬT trong ep-qc — intent LIBRARY_SEARCH có khối dữ liệu, mô hình mở bằng câu mẫu «Mình chưa tìm thấy…» rồi nói thêm.
// Trước: tool-backed ⇒ confidence 1,000, low_confidence=false. Phải là «chưa chắc».
func TestMetadataToolBackedRefusalIsLowConfidence(t *testing.T) {
	hi := decimal.New(75, -2)
	real := `Mình chưa tìm thấy nội dung này trong tài liệu của lớp. Trong tài liệu hiện chỉ liệt kê "Access Attacks" là một trong bốn loại tấn công mạng mà chưa cung cấp ví dụ chi tiết.`
	m := Outcome{Plan: IntentLibrary, Blocks: []Block{{Kind: "library_results"}}}.Metadata(real, hi, false, 0)
	if !m.LowConfidence || !m.Confidence.Valid || !m.Confidence.Decimal.IsZero() {
		t.Fatalf("lời từ chối của intent tool-backed phải thấp: %+v", m)
	}
	// biến thể: dấu hỏi / không dấu / viết hoa vẫn nhận ra
	for _, a := range []string{"MÌNH CHƯA TÌM THẤY NỘI DUNG NÀY TRONG TÀI LIỆU CỦA LỚP.", "Minh chua tim thay noi dung nay trong tai lieu cua lop."} {
		if !(Outcome{Plan: IntentLibrary, Blocks: []Block{{Kind: "x"}}}).Metadata(a, hi, false, 0).LowConfidence {
			t.Errorf("không nhận ra lời từ chối: %q", a)
		}
	}
	// không từ chối ⇒ vẫn 1,000 (dữ liệu lấy từ DB)
	ok := Outcome{Plan: IntentLibrary, Blocks: []Block{{Kind: "library_results"}}}.Metadata("Đây là các tài liệu liên quan.", hi, false, 0)
	if ok.LowConfidence || ok.Confidence.Decimal.String() != "1" {
		t.Errorf("tool-backed bình thường phải 1,000: %+v", ok)
	}
}
