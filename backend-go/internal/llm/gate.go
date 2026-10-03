package llm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

// Work mô tả một lần xin chỗ gọi nhà cung cấp.
type Work struct {
	Lane       Lane
	Task       Task
	ProviderID string
	RPM, TPM   int // hạn mức hiệu lực của nhà cung cấp (đã áp mặc định)
	EstTokens  int
}

// Permit là chỗ đã được cấp; phải gọi Done đúng một lần.
type Permit interface {
	// Wait là thời gian đã chờ trong hàng.
	Wait() time.Duration
	// Done trả chỗ; actualTokens < 0 = không biết (giữ ước tính).
	Done(actualTokens int)
}

// BudgetState là trạng thái ngân sách của phạm vi tệ nhất.
type BudgetState int

// Ba trạng thái.
const (
	BudgetOK BudgetState = iota
	BudgetWarn
	BudgetExhausted
)

// Gate là điểm nối của Scheduler (US-P1-03): hàng đợi theo làn + token bucket + đồng thời toàn cục, mạch ngắt, ngân sách.
// Gateway chạy được không cần Gate (nopGate) — mọi thứ luôn được phép.
type Gate interface {
	// Admit chờ trong hàng rồi cấp chỗ; lỗi: *ErrOverloaded, ErrDeadline, hoặc lỗi ctx.
	Admit(ctx context.Context, w Work) (Permit, error)
	// BreakerAllow cho biết có được gọi nhà cung cấp (mạch đóng / bán mở lấy được lượt thử).
	BreakerAllow(ctx context.Context, providerID string) bool
	// BreakerReport báo kết quả: k rỗng = thành công.
	BreakerReport(ctx context.Context, providerID string, k provider.Kind)
	// BudgetState đọc trạng thái ngân sách (hệ thống + lớp).
	BudgetState(ctx context.Context, courseID *uuid.UUID) BudgetState
	// BudgetCharge cộng chi phí đã dùng.
	BudgetCharge(ctx context.Context, courseID *uuid.UUID, cost decimal.Decimal)
}

type nopGate struct{}

type nopPermit struct{}

func (nopPermit) Wait() time.Duration { return 0 }
func (nopPermit) Done(int)            {}

func (nopGate) Admit(context.Context, Work) (Permit, error)               { return nopPermit{}, nil }
func (nopGate) BreakerAllow(context.Context, string) bool                 { return true }
func (nopGate) BreakerReport(context.Context, string, provider.Kind)      {}
func (nopGate) BudgetState(context.Context, *uuid.UUID) BudgetState       { return BudgetOK }
func (nopGate) BudgetCharge(context.Context, *uuid.UUID, decimal.Decimal) {}
