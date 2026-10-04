package httpapi

import (
	"context"

	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
)

// todayLLM cấp cho "Hôm nay" của Admin dữ liệu cổng AI (mạch mở, % ngân sách). Gói today không import internal/llm.
type todayLLM struct{ rt *llmrt.Runtime }

// BudgetPercent là % ngân sách AI hệ thống đã dùng hôm nay.
func (t todayLLM) BudgetPercent(ctx context.Context) (int, bool) {
	lim, err := t.rt.Resolver.Budget(ctx, "system", nil)
	if err != nil || lim.Daily == nil {
		return 0, false
	}
	st, err := t.rt.Budget.Status(ctx, "system", nil, budget.Limit{Daily: lim.Daily, Monthly: lim.Monthly})
	if err != nil || st.PctToday == nil {
		return 0, false
	}
	return int(st.PctToday.IntPart()), true
}

// OpenCircuit: mạch của nhà cung cấp đang mở.
func (t todayLLM) OpenCircuit(ctx context.Context, providerID string) bool {
	return t.rt.Scheduler.CircuitState(ctx, providerID) == "open"
}
