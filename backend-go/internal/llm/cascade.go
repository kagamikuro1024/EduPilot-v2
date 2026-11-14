package llm

import (
	"context"
)

// Path là đường đi của một cổng cascade; ba giá trị đầu là nhãn `llm_audit.cascade_path`.
type Path string

// Các đường đi. PathNotUsed: cổng tắt hoặc provider chưa sẵn sàng — không gọi decision, không ghi dòng audit (US-DM-02 AC3).
const (
	PathFast       Path = "FAST"
	PathFallback   Path = "FALLBACK"
	PathFailClosed Path = "FAIL_CLOSED"
	PathNotUsed    Path = "NOT_USED"
)

// Verdict là quyết định của cổng. Yes dùng cho cổng có/không (Threads, Injection); Label cho cổng chọn nhãn (Routing).
type Verdict struct {
	Yes        bool
	Label      string
	Confidence float64
}

// Policy biến câu trả lời của decision thành Verdict. ok = false: không đủ chắc → fallback. thr là ε / τ hiệu lực của cổng (Tau/Eps = 0 lấy thr).
type Policy interface {
	judge(DecideResponse, float64) (Verdict, bool)
}

// Symmetric: nhận kết quả khi confidence ≥ τ (cổng CHAT_ROUTING, choice QID).
type Symmetric struct {
	QID string
	Tau float64
}

func (p Symmetric) judge(r DecideResponse, thr float64) (Verdict, bool) {
	tau := p.Tau
	if tau <= 0 {
		tau = thr
	}
	a, ok := r.Choice(p.QID)
	if !ok || a.Confidence < tau {
		return Verdict{}, false
	}
	return Verdict{Label: a.Choice, Confidence: a.Confidence}, true
}

// AddOnly: chỉ THÊM chặn — noul ≥ 0,5 là chắc chắn "có" (FAST); noul ≤ ε là chắc chắn "không" (FAST); ở giữa → fallback (cổng THREADS_PRECHECK).
// Với INJECTION_FLAG dùng OrFlag (cờ khi decision ≥ τ, ngược lại fallback không đổi kết quả của lớp trước).
type AddOnly struct {
	QID string
	Eps float64
}

func (p AddOnly) judge(r DecideResponse, thr float64) (Verdict, bool) {
	eps := p.Eps
	if eps <= 0 {
		eps = thr
	}
	a, ok := r.Noul(p.QID)
	if !ok {
		return Verdict{}, false
	}
	switch {
	case a.Noul >= 0.5:
		return Verdict{Yes: true, Confidence: a.Noul}, true
	case a.Noul <= eps:
		return Verdict{Yes: false, Confidence: 1 - a.Noul}, true
	}
	return Verdict{}, false
}

// OrFlag: noul ≥ τ → cờ "có" (FAST); dưới τ → fallback (INJECTION_FLAG: kết quả cuối = OR của decision và các lớp còn lại do người gọi ghép).
type OrFlag struct {
	QID string
	Tau float64
}

func (p OrFlag) judge(r DecideResponse, thr float64) (Verdict, bool) {
	tau := p.Tau
	if tau <= 0 {
		tau = thr
	}
	a, ok := r.Noul(p.QID)
	if !ok || a.Noul < tau {
		return Verdict{}, false
	}
	return Verdict{Yes: true, Confidence: a.Noul}, true
}

// Fallback chạy đường cũ (luật / Structured). Trả lỗi → FAIL_CLOSED.
type Fallback func(ctx context.Context) (Verdict, error)

// Cascade chạy cổng: decision trước; đủ chắc → FAST; ngược lại (lỗi, timeout, hết giấy phép, vùng lưỡng lự) → fallback → FALLBACK; fallback cũng lỗi →
// trả failClosed + FAIL_CLOSED (người gọi quyết định chặn). Cổng tắt / provider chưa sẵn sàng → PathNotUsed, KHÔNG gọi gì: người gọi chạy đường cũ.
// Ghi đúng một dòng llm_audit cho lời gọi decision (cascade_gate + cascade_path); fallback ghi dòng riêng của task gốc.
func (g *Gateway) Cascade(ctx context.Context, gate DecideGate, req DecideRequest, pol Policy, fb Fallback, failClosed Verdict) (Verdict, Path) {
	cfg, ok := g.o.Decision.Effective(gate, IdentityFrom(ctx).CourseID)
	if !ok {
		return Verdict{}, PathNotUsed
	}
	if _, ready := g.o.Registry.DecideTarget(); !ready {
		return Verdict{}, PathNotUsed
	}
	req.Gate = gate
	if req.Lane != LaneBatch && gate == GateInjection {
		req.Lane = LaneBatch
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = g.o.DecisionTimeout
	}
	resp, p, err := g.decide(ctx, req, timeout)
	if err == nil {
		if v, ok := pol.judge(resp, cfg.Threshold); ok {
			p.Close(PathFast)
			return v, PathFast
		}
	}
	if fb != nil {
		if v, ferr := fb(ctx); ferr == nil {
			p.Close(PathFallback)
			return v, PathFallback
		}
	}
	p.Close(PathFailClosed)
	return failClosed, PathFailClosed
}
