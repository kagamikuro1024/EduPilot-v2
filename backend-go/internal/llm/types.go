// Package llm: cổng DUY NHẤT của backend tới LLM / embedding (luật 3 và 11 của AGENTS.md). Danh tính (user, lớp) và trace_id lấy từ ctx,
// không từ người gọi. SDK chỉ nằm ở internal/llm/provider.
package llm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// Lane là làn ưu tiên; số nhỏ = ưu tiên cao.
type Lane int

// Ba làn (SRS 4.1).
const (
	LaneInteractive Lane = iota
	LaneNearRealtime
	LaneBatch
)

func (l Lane) String() string {
	switch l {
	case LaneInteractive:
		return "INTERACTIVE"
	case LaneNearRealtime:
		return "NEAR_REALTIME"
	}
	return "BATCH"
}

// Task là tác vụ LLM.
type Task string

// Bảy tác vụ.
const (
	TaskChat        Task = "CHAT"
	TaskClassify    Task = "CLASSIFY"
	TaskUtility     Task = "UTILITY"
	TaskGrading     Task = "GRADING"
	TaskQuestionGen Task = "QUESTION_GEN"
	TaskInsight     Task = "INSIGHT"
	TaskEmbedding   Task = "EMBEDDING"
)

// DefaultLane là làn mặc định của tác vụ (SRS 4.3).
func DefaultLane(t Task) Lane {
	switch t {
	case TaskChat:
		return LaneInteractive
	case TaskClassify, TaskUtility:
		return LaneNearRealtime
	}
	return LaneBatch
}

// ResolveLane áp quy tắc: chỉ được HẠ làn so với mặc định; riêng EMBEDDING được nâng lên INTERACTIVE (vectơ hoá câu hỏi lúc chat).
func ResolveLane(t Task, want *Lane) (Lane, error) {
	def := DefaultLane(t)
	if want == nil {
		return def, nil
	}
	if *want < LaneInteractive || *want > LaneBatch {
		return 0, ErrBadLane
	}
	if *want >= def || t == TaskEmbedding {
		return *want, nil
	}
	return 0, ErrBadLane
}

// Message là một lượt hội thoại (role: system | user | assistant).
type Message struct{ Role, Content string }

// Params là tham số sinh; giá trị không đặt lấy từ tuyến của tác vụ.
type Params struct {
	Temperature *float64
	MaxTokens   int
}

// Passage là đoạn tài liệu cho đường suy giảm (INTERACTIVE khi mọi nhà cung cấp chết).
type Passage struct {
	Text   string
	Source string // tên tài liệu
	Page   int
	Score  float64
}

// Request là một yêu cầu sinh văn bản. KHÔNG có trường người dùng / lớp: hai giá trị đó lấy từ ctx (identity.go).
type Request struct {
	Task           Task
	Lane           *Lane // nil = mặc định theo bảng 4.3
	Messages       []Message
	Params         Params
	Shareable      bool // chỉ true cho việc không chứa dữ liệu cá nhân
	PIIMaskedCount int
	Passages       []Passage
}

// Response là kết quả sinh văn bản.
type Response struct {
	Text          string
	TokensIn      int
	TokensOut     int
	Provider      string
	Model         string
	FallbackIndex int
	Degraded      bool
	// MaskedCurrent: số lần thay trong tin hiện tại + kết quả tool của lượt (tin cuối của Request); KHÔNG tính lịch sử đã báo ở lượt trước (proposals #11d).
	MaskedCurrent int
	QueueWait     time.Duration
	CostEst       decimal.Decimal
}

// Chunk là một phần tử của luồng. Phần tử cuối: Done=true (kèm Response) hoặc Err != nil.
type Chunk struct {
	Text     string
	Done     bool
	Degraded bool // chunk chữ là câu suy giảm của gói llm: người gọi có thể bỏ và tự dựng câu (chat)
	Err      error
	Response *Response
}

// EmbedRequest là yêu cầu nhúng; Lane nil = BATCH (nạp tài liệu), INTERACTIVE cho câu hỏi lúc chat.
type EmbedRequest struct {
	Inputs         []string
	Lane           *Lane
	PIIMaskedCount int
	// System: chữ tĩnh của hệ thống (vd mẫu câu cá nhân của bộ phân loại), KHÔNG có dữ liệu người dùng → bỏ qua bước che. Không dùng cho chữ người dùng nhập.
	System bool
}

// Client là cổng LLM: đúng bốn hàm.
type Client interface {
	Chat(ctx context.Context, r Request) (Response, error)
	Stream(ctx context.Context, r Request) (<-chan Chunk, error)
	Structured(ctx context.Context, r Request, schema json.RawMessage) (json.RawMessage, error)
	Embed(ctx context.Context, r EmbedRequest) ([][]float32, error)
}

// EmbedDims là số chiều vectơ cố định của hệ thống.
const EmbedDims = 1536

// EmbedBatch là số chuỗi tối đa mỗi lời gọi nhúng; lô lớn hơn tự chia.
const EmbedBatch = 100
