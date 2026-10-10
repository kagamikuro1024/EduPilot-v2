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
		m := o.Metadata(supported, dec(thr), false, 0)
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
