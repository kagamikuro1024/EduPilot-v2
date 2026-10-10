package agent

// Câu mẫu (0 lời gọi LLM) — SRS FEAT-private-chat-pii 4.6. Lời văn dành cho sinh viên: không từ kỹ thuật (AI, RAG, PII…).
const (
	ReplyOtherPerson = "Mình chỉ xem được dữ liệu của chính bạn."
	ReplyNoFormula   = "Lớp chưa có công thức điểm chính thức do giảng viên xác nhận."
	ReplyNoContext   = "Mình chưa tìm thấy nội dung này trong tài liệu của lớp."
	// ReplyCrisis: lời ân cần ngắn; không tư vấn chuyên môn, không tự báo cho ai.
	ReplyCrisis = "Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên của bạn nhé."
	// DefaultSupport thay SUPPORT_RESOURCES_VI khi cấu hình rỗng.
	DefaultSupport = "Bạn hãy trao đổi với giảng viên hoặc phòng công tác sinh viên của trường."
)

// ReplyNoData là câu "chưa có dữ liệu {điểm danh | điểm cộng | điểm | lịch}".
func ReplyNoData(what string) string {
	return "Hệ thống chưa có dữ liệu " + what + " của bạn."
}
