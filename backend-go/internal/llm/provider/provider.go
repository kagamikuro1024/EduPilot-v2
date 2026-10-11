// Package provider: mọi lời gọi tới nhà cung cấp LLM. ĐÂY là nơi DUY NHẤT import SDK (`openai-go`) — luật 3 của AGENTS.md,
// kiểm bằng grep ở US-P1-02 AC1. Anthropic, Gemini, OpenAI và máy chủ tương thích đều đi qua lớp tương thích OpenAI (D46).
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Kind là loại lỗi đã chuẩn hoá (SRS 4.2).
type Kind string

// Các loại lỗi.
const (
	KindAuth          Kind = "AUTH"
	KindModelNotFound Kind = "MODEL_NOT_FOUND"
	KindRateLimit     Kind = "RATE_LIMIT"
	KindServer        Kind = "SERVER"
	KindTimeout       Kind = "TIMEOUT"
	KindNetwork       Kind = "NETWORK"
	KindBadRequest    Kind = "BAD_REQUEST"
	KindBadResponse   Kind = "BAD_RESPONSE" // trả về sai định dạng / sai schema
	KindDimsMismatch  Kind = "DIMS_MISMATCH"
	KindCancelled     Kind = "CANCELLED"
	KindReplayMiss    Kind = "REPLAY_MISS"
)

// Retryable: thử lại cùng nhà cung cấp.
func (k Kind) Retryable() bool {
	switch k {
	case KindRateLimit, KindServer, KindTimeout, KindNetwork:
		return true
	}
	return false
}

// NextProvider: chuyển sang nhà kế tiếp trong chuỗi. BAD_REQUEST là lỗi của ta nên không chuyển; huỷ / vectơ sai chiều cũng không.
func (k Kind) NextProvider() bool {
	switch k {
	case KindAuth, KindModelNotFound, KindRateLimit, KindServer, KindTimeout, KindNetwork, KindBadResponse:
		return true
	}
	return false
}

// CountsToBreaker: lỗi tính vào mạch ngắt (SRS 3.2). MODEL_NOT_FOUND không tính (lỗi cấu hình mô hình, nhà vẫn sống).
func (k Kind) CountsToBreaker() bool {
	switch k {
	case KindAuth, KindRateLimit, KindServer, KindTimeout, KindNetwork, KindBadResponse:
		return true
	}
	return false
}

// Error là lỗi nhà cung cấp đã chuẩn hoá. Error() KHÔNG chứa thân lỗi của nhà cung cấp; Detail là thân đã bôi khoá và cắt ≤ 200 ký tự,
// chỉ dùng cho log mức debug.
type Error struct {
	Kind       Kind
	Status     int
	RetryAfter time.Duration
	Detail     string
}

func (e *Error) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("nhà cung cấp LLM: %s (HTTP %d)", e.Kind, e.Status)
	}
	return "nhà cung cấp LLM: " + string(e.Kind)
}

// Message là một lượt hội thoại.
type Message struct{ Role, Content string }

// ChatOpts là một lời gọi sinh văn bản tới MỘT mô hình của MỘT nhà cung cấp.
type ChatOpts struct {
	Model       string
	Messages    []Message
	Temperature *float64
	MaxTokens   int
	// Schema khác rỗng = đầu ra có cấu trúc; provider chọn cách ép theo loại nhà cung cấp (xem openai.go).
	Schema json.RawMessage
	// Fast: làn INTERACTIVE — bật chế độ suy nghĩ nhẹ cho nhà cung cấp có "thinking" mặc định (Gemini 3.x).
	Fast bool
	// Task là tác vụ LLM của lời gọi (CHAT, CLASSIFY…); chỉ để provider giả ghi lại cho route thử `_test/llm/payloads`.
	Task string
}

// Result là kết quả một lời gọi sinh văn bản.
type Result struct {
	Text                string
	TokensIn, TokensOut int
	Estimated           bool // nhà cung cấp không trả usage: token ra là ước tính
}

// Delta là một mẩu của luồng. Err != nil là phần tử cuối.
type Delta struct {
	Text      string
	Usage     *Result // chỉ ở phần tử cuối của luồng thành công: TokensIn/TokensOut thật
	Err       error
	Estimated bool
}

// EmbedOpts là một lời gọi nhúng.
type EmbedOpts struct {
	Model  string
	Inputs []string
	Dims   int
}

// Provider là hợp đồng của một nhà cung cấp đã dựng sẵn (khoá, địa chỉ gốc nằm trong đối tượng).
type Provider interface {
	Chat(ctx context.Context, o ChatOpts) (Result, error)
	// Stream trả kênh đóng khi xong; ctx huỷ → dừng lời gọi mạng.
	Stream(ctx context.Context, o ChatOpts) (<-chan Delta, error)
	Embed(ctx context.Context, o EmbedOpts) ([][]float32, int, error)
}

var (
	skKey  = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)
	bearer = regexp.MustCompile(`(?i)bearer\s+\S+`)
)

// MaxDetail là độ dài tối đa của Error.Detail.
const MaxDetail = 200

// Redact bôi mọi mẫu khoá (`sk-…`, `Bearer …`, và chính các chuỗi bí mật truyền vào) rồi cắt ≤ MaxDetail ký tự.
func Redact(s string, secrets ...string) string {
	for _, sec := range secrets {
		if len(sec) >= 4 {
			s = strings.ReplaceAll(s, sec, "[REDACTED]")
		}
	}
	s = skKey.ReplaceAllString(s, "[REDACTED]")
	s = bearer.ReplaceAllString(s, "Bearer [REDACTED]")
	if r := []rune(s); len(r) > MaxDetail {
		s = string(r[:MaxDetail])
	}
	return s
}
