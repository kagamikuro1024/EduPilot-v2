package llm

// Tên mô hình mặc định của đường dự phòng env (SRS 4.2). Đổi bằng DB, không cần sửa mã. Hai mô hình dự phòng cũ
// (claude-3-5-haiku-latest, gemini-2.0-flash) đã bị nhà cung cấp gỡ — docs/research/2026-10-03-openai-go-compat.md.
// Theo dõi: claude-haiku-4-5-20251001 có thể bị gỡ từ 2026-10-15.
const (
	DefaultOpenAIChat      = "gpt-4o-mini"
	DefaultOpenAIEmbedding = "text-embedding-3-small"
	DefaultAnthropicChat   = "claude-haiku-4-5-20251001"
	DefaultGeminiChat      = "gemini-3.6-flash"
	FakeChatModel          = "fake-chat"
	FakeEmbedModel         = "fake-embed"
)

// ChatTasks là sáu tác vụ dùng mô hình chat.
func ChatTasks() []Task {
	return []Task{TaskChat, TaskClassify, TaskUtility, TaskGrading, TaskQuestionGen, TaskInsight}
}
