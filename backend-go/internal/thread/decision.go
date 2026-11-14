package thread

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/llm"
)

// Decider là phần của cổng LLM mà cổng THREADS_PRECHECK dùng: Structured (fallback) + Cascade (decision model, D60).
type Decider interface {
	llm.Client
	Cascade(ctx context.Context, gate llm.DecideGate, req llm.DecideRequest, pol llm.Policy, fb llm.Fallback, failClosed llm.Verdict) (llm.Verdict, llm.Path)
}

// DefaultStructuredTimeout là THREADS_STRUCTURED_TIMEOUT mặc định (SRS FEAT-decision-model 4.3).
const DefaultStructuredTimeout = 6 * time.Second

const (
	privateQID = "private_or_named"
	// Chữ cố định trong mã (SRS 4.3): người dùng không sửa được; cùng câu hỏi cho decision và cho Structured.
	privateQuestion = "Nội dung có hỏi về thông tin riêng tư của một sinh viên cụ thể (điểm, chuyên cần, hoàn cảnh) hoặc nêu tên một người cụ thể không?"
	// answer là chuỗi "no" | "yes" (không phải boolean): giá trị đầu của enum là "no" nên provider giả tất định mặc định "không" (đường fallback không chặn nhầm ở dev / demo).
	privateSchema = `{"type":"object","properties":{"answer":{"type":"string","enum":["no","yes"]}},"required":["answer"],"additionalProperties":false}`
)

// CheckPublish là tường lửa của MỌI đường LƯU bài công khai: luật (CheckPost) rồi cổng decision.
// Luật đã chặn → trả ngay, KHÔNG gọi decision (AC1/AC3: chỉ thêm chặn, không đường nào gỡ chặn của luật).
// Cổng tắt / chưa có provider → đúng kết quả của luật (như trước 10.5). `precheck` (đang gõ) và RecordSwitched vẫn dùng CheckPost: không gọi decision (AC5).
func (f *Firewall) CheckPublish(ctx context.Context, courseID uuid.UUID, title, body string) (Verdict, error) {
	v, err := f.CheckPost(ctx, courseID, title, body)
	if err != nil || !v.Allowed || f.Decider == nil {
		return v, err
	}
	ctx = llm.WithIdentity(ctx, llm.Identity{CourseID: &courseID})
	input := title + "\n" + body
	req := llm.DecideRequest{Lane: llm.LaneInteractive, Input: input, Questions: []llm.Question{llm.NoulQ{ID: privateQID, Text: privateQuestion}}}
	timeout := f.StructuredTimeout
	if timeout <= 0 {
		timeout = DefaultStructuredTimeout
	}
	fb := func(ctx context.Context) (llm.Verdict, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		raw, err := f.Decider.Structured(ctx, llm.Request{Task: llm.TaskClassify, Messages: []llm.Message{
			{Role: "system", Content: privateQuestion + " Trả lời answer=yes nếu có, answer=no nếu không."},
			{Role: "user", Content: input}}}, json.RawMessage(privateSchema))
		if err != nil {
			return llm.Verdict{}, err
		}
		var out struct {
			Answer string `json:"answer"`
		}
		if err := json.Unmarshal(raw, &out); err != nil || (out.Answer != "yes" && out.Answer != "no") {
			return llm.Verdict{}, llm.ErrBadRequest
		}
		return llm.Verdict{Yes: out.Answer == "yes"}, nil
	}
	dv, path := f.Decider.Cascade(ctx, llm.GateThreads, req, llm.AddOnly{QID: privateQID}, fb, llm.Verdict{Yes: true}) // fail-closed = chặn
	if path == llm.PathNotUsed || !dv.Yes {
		return v, nil
	}
	v.Allowed, v.Personal = false, true
	v.Reasons = []Reason{{Type: "PERSONAL_QUESTION", Count: 1}}
	v.RedactedTitle, v.RedactedBody = title, body // câu hỏi riêng tư không có khoảng để thay
	return v, nil
}
