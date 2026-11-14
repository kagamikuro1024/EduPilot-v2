package agent

import (
	"context"

	"github.com/edupilot/backend-go/internal/llm"
)

// Decider là phần của cổng LLM mà cổng CHAT_ROUTING dùng (D60).
type Decider interface {
	Cascade(ctx context.Context, gate llm.DecideGate, req llm.DecideRequest, pol llm.Policy, fb llm.Fallback, failClosed llm.Verdict) (llm.Verdict, llm.Path)
	GateEffective(ctx context.Context, gate llm.DecideGate) bool
}

const (
	routeQID = "intent"
	// Chữ cố định trong mã (SRS FEAT-decision-model 4.3). KHÔNG có nhãn CRISIS: khủng hoảng chỉ do luật quyết.
	routeQuestion = "Tin nhắn của sinh viên gửi trợ lý học tập của một môn học. Tin nhắn thuộc loại nào?"
)

// routeOptions là 11 nhãn của CHAT_ROUTING kèm mô tả ngắn.
var routeOptions = []llm.Option{ //nolint:gochecknoglobals // bảng bất biến
	{Key: string(IntentWhatIf), Text: "Giả định: nếu đạt X điểm ở một thành phần thì tổng kết / qua môn thế nào"},
	{Key: string(IntentGradeFormula), Text: "Cách tính điểm, trọng số, công thức điểm của môn (chung cho cả lớp)"},
	{Key: string(IntentAttendance), Text: "Số buổi vắng / nghỉ / điểm danh của chính người gửi"},
	{Key: string(IntentParticipation), Text: "Điểm cộng / điểm phát biểu của chính người gửi"},
	{Key: string(IntentGrade), Text: "Điểm số / kết quả của chính người gửi"},
	{Key: string(IntentExamSchedule), Text: "Lịch thi, ngày giờ phòng thi"},
	{Key: string(IntentUpcoming), Text: "Việc sắp tới: hạn nộp, lịch học, sự kiện trong khoảng thời gian tới"},
	{Key: string(IntentLibrary), Text: "Tìm tài liệu, slide, giáo trình"},
	{Key: string(IntentCourseQA), Text: "Hỏi nội dung kiến thức môn học hoặc quy chế"},
	{Key: string(IntentOtherPerson), Text: "Hỏi điểm, chuyên cần hay thông tin cá nhân của một người khác"},
	{Key: string(IntentSmalltalk), Text: "Chào hỏi, cảm ơn, xã giao không có câu hỏi"},
}

// routeDecision là NƠI DUY NHẤT của chat được gọi Decide (US-DM-05; TestDecideCallersAllowlist). Chỉ gọi khi luật rơi về COURSE_QA. Trả nhãn khi
// decision đủ chắc (`confidence ≥ τ`, Symmetric); ngược lại (lỗi, hết hạn, dưới τ, cầu dao mở, hết giấy phép) "" — giữ kết quả của luật, KHÔNG gọi LLM.
// Nhãn không thể là CRISIS (không nằm trong danh sách) và không thể ghi đè ý định mà luật đã khớp (người gọi chỉ áp khi ý định luật là COURSE_QA).
func (a *Analyzer) routeDecision(ctx context.Context, text string) Intent {
	req := llm.DecideRequest{Lane: llm.LaneInteractive, Input: text, Questions: []llm.Question{llm.ChoiceQ{ID: routeQID, Text: routeQuestion, Options: routeOptions}}}
	v, path := a.Decider.Cascade(ctx, llm.GateRouting, req, llm.Symmetric{QID: routeQID},
		func(context.Context) (llm.Verdict, error) { return llm.Verdict{}, nil }, llm.Verdict{})
	if path != llm.PathFast {
		return ""
	}
	for _, o := range routeOptions {
		if o.Key == v.Label {
			return Intent(v.Label)
		}
	}
	return ""
}
