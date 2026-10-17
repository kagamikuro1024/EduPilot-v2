package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// US-P5-02 AC14 (QC vòng 1, TC-38): câu ghép nhiều ý cá nhân → chạy mọi tool SQL tương ứng, đúng MỘT lần sinh chữ, phân loại vẫn một lần.
func TestCompoundPersonalQuestion(t *testing.T) {
	t.Parallel()
	r := newRig(t, true, 3)
	out := r.respond(t, "Em đã nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu rồi ạ?")
	require.NotNil(t, out.Stream)
	require.EqualValues(t, 1, r.g.calls.Load(), "một lời gọi sinh chữ")
	require.EqualValues(t, 1, r.spy.attendance.Load())
	require.EqualValues(t, 1, r.spy.participation.Load())
	kinds := []string{}
	for _, b := range out.Blocks {
		kinds = append(kinds, b.Kind)
	}
	require.Equal(t, []string{"attendance", "participation"}, kinds, "cả hai khối, theo thứ tự cố định")
	require.ElementsMatch(t, []Intent{IntentAttendance, IntentParticipation}, append([]Intent{out.Class.Intent}, out.Class.Extra...))
	require.Equal(t, me, r.spy.lastUser.Load())
}

func TestCompoundIntents(t *testing.T) {
	t.Parallel()
	for text, want := range map[string][]Intent{
		"Em nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu": {IntentAttendance}, // ý chính là điểm cộng (luật SRS 4.5 xét trước), ý phụ là điểm danh
		"Em vắng mấy buổi và điểm tổng kết của em bao nhiêu":   {IntentGrade},
		"Em vắng mấy buổi rồi":                                 nil,
		"Điểm cộng của em được bao nhiêu":                      nil,
		"Điểm chuyên cần của em thế nào":                       nil,
		"Điểm giữa kỳ của em là bao nhiêu":                     nil,
	} {
		require.Equal(t, want, extraPersonal(text, DetectIntent(text)), text)
	}
}

func TestCompoundNoData(t *testing.T) {
	t.Parallel()
	// không có dữ liệu cho cả hai ý → câu mẫu, 0 sinh
	empty := newRig(t, false, 3)
	out := empty.respond(t, "Em đã nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu rồi ạ?")
	require.NotEmpty(t, out.Canned)
	require.Zero(t, empty.g.calls.Load())
}
