package llm

import (
	"errors"
	"fmt"
	"time"
)

// Lỗi của gói; handler ánh xạ sang mã HTTP ở SRS 6.1.
var (
	ErrNotConfigured      = errors.New("llm: chưa cấu hình nhà cung cấp")
	ErrDeadline           = errors.New("llm: quá hạn")
	ErrBadRequest         = errors.New("llm: yêu cầu không hợp lệ")
	ErrAllProvidersFailed = errors.New("llm: mọi nhà cung cấp đều lỗi")
	ErrBadLane            = errors.New("llm: làn không hợp lệ cho tác vụ này")
	ErrStream             = errors.New("llm: luồng bị ngắt giữa chừng")
)

// ErrOverloaded: hàng chờ đầy hoặc chờ quá LLM_QUEUE_WAIT_MAX (503 OVERLOADED).
type ErrOverloaded struct{ RetryAfter time.Duration }

func (e *ErrOverloaded) Error() string {
	return fmt.Sprintf("llm: hệ thống đang quá tải, thử lại sau %d giây", int(e.RetryAfter.Seconds()))
}

// Lý do của ErrUnavailable.
const (
	ReasonAllFailed       = "all_providers_failed"
	ReasonBudgetExhausted = "budget_exhausted"
)

// ErrUnavailable: không có nhà cung cấp dùng được (503 LLM_UNAVAILABLE).
type ErrUnavailable struct {
	Reason string
	// Outage: lý do là hạ tầng nhà cung cấp (mạch mở, mạng, quá hạn, 5xx, 429), không phải đầu ra sai schema / sai khoá. Việc nền hoãn rồi thử lại thay vì tính lượt (P-11-23).
	Outage bool
	// BadOutput: mọi nhà cung cấp đã thử đều trả đầu ra sai định dạng / sai schema (không lỗi hạ tầng, không mạch mở): thử lại không đổi được kết quả, người gọi KHÔNG thử lại (DEF-R8-A1).
	BadOutput bool
}

func (e *ErrUnavailable) Error() string { return "llm: không khả dụng: " + e.Reason }

// Is: ErrUnavailable{all_providers_failed} cũng là ErrAllProvidersFailed (người gọi có thể kiểm bằng errors.Is).
func (e *ErrUnavailable) Is(target error) bool {
	return target == ErrAllProvidersFailed && e.Reason == ReasonAllFailed
}

// allFailed là ErrAllProvidersFailed kèm cờ outage / badOutput; errors.Is(err, ErrAllProvidersFailed) vẫn đúng.
type allFailed struct{ outage, badOutput bool }

func (e *allFailed) Error() string        { return ErrAllProvidersFailed.Error() }
func (e *allFailed) Is(target error) bool { return target == ErrAllProvidersFailed }

// ErrDimsMismatch: vectơ trả về khác 1536 chiều (422 MODEL_DIMS_MISMATCH).
type ErrDimsMismatch struct{ Expected, Actual int }

func (e *ErrDimsMismatch) Error() string {
	return fmt.Sprintf("llm: vectơ %d chiều, cần %d", e.Actual, e.Expected)
}
