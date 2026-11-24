package scheduler_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

func breakerRig() (*scheduler.Scheduler, *clock.Fake) {
	clk := clock.NewFake(time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC))
	return scheduler.New(baseCfg(), nil, discard(), scheduler.WithClock(clk)), clk
}

func TestBreakerOpensAfter5(t *testing.T) {
	t.Parallel()
	s, clk := breakerRig()
	ctx := context.Background()
	for i := range 4 {
		if !s.BreakerAllow(ctx, "p") {
			t.Fatalf("mạch phải đóng sau %d lỗi", i)
		}
		s.BreakerReport(ctx, "p", provider.KindServer)
	}
	if got := s.CircuitState(ctx, "p"); got != "closed" {
		t.Fatalf("sau 4 lỗi: %s", got)
	}
	s.BreakerReport(ctx, "p", provider.KindRateLimit) // lỗi thứ 5
	if s.BreakerAllow(ctx, "p") || s.CircuitState(ctx, "p") != "open" {
		t.Fatal("sau 5 lỗi liên tiếp mạch phải mở")
	}
	clk.Advance(29 * time.Second)
	if s.BreakerAllow(ctx, "p") {
		t.Error("còn trong 30 s mở")
	}
	// nhà khác không bị ảnh hưởng
	if !s.BreakerAllow(ctx, "q") {
		t.Error("mạch là theo nhà cung cấp")
	}
}

func TestBreakerHalfOpen(t *testing.T) {
	t.Parallel()
	s, clk := breakerRig()
	ctx := context.Background()
	trip := func() {
		for range 5 {
			s.BreakerReport(ctx, "p", provider.KindTimeout)
		}
	}
	trip()
	clk.Advance(30 * time.Second)
	if st := s.CircuitState(ctx, "p"); st != "half_open" {
		t.Fatalf("sau 30 s: %s", st)
	}
	if !s.BreakerAllow(ctx, "p") {
		t.Fatal("bán mở phải cho đúng 1 lời gọi thử")
	}
	if s.BreakerAllow(ctx, "p") {
		t.Fatal("chỉ 1 lời gọi thử, lượt thứ hai phải bị chặn")
	}
	s.BreakerReport(ctx, "p", provider.KindServer) // thử thất bại → mở lại 30 s
	if s.BreakerAllow(ctx, "p") {
		t.Fatal("thử thất bại phải mở lại")
	}
	clk.Advance(29 * time.Second)
	if s.BreakerAllow(ctx, "p") {
		t.Fatal("mở lại phải đủ 30 s")
	}
	clk.Advance(time.Second)
	if !s.BreakerAllow(ctx, "p") {
		t.Fatal("hết 30 s: bán mở lại")
	}
	s.BreakerReport(ctx, "p", "") // thử thành công → đóng
	if st := s.CircuitState(ctx, "p"); st != "closed" || !s.BreakerAllow(ctx, "p") || !s.BreakerAllow(ctx, "p") {
		t.Fatalf("thử thành công phải đóng mạch: %s", st)
	}
	trip()
	if s.BreakerAllow(ctx, "p") {
		t.Error("đóng rồi lại phải đếm từ 0")
	}
}

func TestBreakerResetOnSuccess(t *testing.T) {
	t.Parallel()
	s, _ := breakerRig()
	ctx := context.Background()
	for range 4 {
		s.BreakerReport(ctx, "p", provider.KindNetwork)
	}
	s.BreakerReport(ctx, "p", "") // thành công xen giữa đặt lại bộ đếm
	for range 4 {
		s.BreakerReport(ctx, "p", provider.KindNetwork)
	}
	if !s.BreakerAllow(ctx, "p") || s.CircuitState(ctx, "p") != "closed" {
		t.Fatal("4 + thành công + 4 lỗi không được mở mạch")
	}
}

func TestBreakerIgnoresBadRequest(t *testing.T) {
	t.Parallel()
	bad := &stubProv{name: "A", chat: func(context.Context, int) (resultT, error) {
		return resultT{}, &provider.Error{Kind: provider.KindBadRequest, Status: 400}
	}}
	r := stubRig(t, baseCfg(), bad)
	for range 10 {
		if _, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskUtility, Messages: msg("x")}); !errors.Is(err, llm.ErrBadRequest) {
			t.Fatalf("err = %v", err)
		}
	}
	if st := r.s.CircuitState(t.Context(), "A"); st != "closed" {
		t.Fatalf("BAD_REQUEST không tính vào mạch nhưng trạng thái = %s", st)
	}
	if bad.calls.Load() != 10 {
		t.Errorf("số lần gọi = %d, muốn 10 (mạch vẫn đóng)", bad.calls.Load())
	}
}

// Mạch mở thì bỏ qua nhà đó ngay (không gọi), chuyển dự phòng, trong < 5 ms thêm.
func TestBreakerOpenSkipsProvider(t *testing.T) {
	t.Parallel()
	a := &stubProv{name: "A", chat: func(context.Context, int) (resultT, error) {
		return resultT{}, &provider.Error{Kind: provider.KindServer, Status: 503}
	}}
	b := &stubProv{name: "B"}
	cfg := baseCfg()
	cfg.BreakerFails = 2
	r := stubRig(t, cfg, a, b)
	lane := llm.LaneBatch
	for range 2 { // mỗi lời gọi: A lỗi 3 lần (1+2 thử lại) rồi sang B; 2 lời gọi → 2 lỗi liên tiếp báo mạch? mỗi nhà báo 1 lần/lời gọi
		if _, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg("x")}); err != nil {
			t.Fatal(err)
		}
	}
	callsA := a.calls.Load()
	resp, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg("x")})
	if err != nil || resp.Provider != "B" || resp.FallbackIndex != 1 {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if a.calls.Load() != callsA {
		t.Errorf("mạch mở nhưng vẫn gọi A (%d → %d)", callsA, a.calls.Load())
	}
	rows := r.cap.all(r.g)
	if last := rows[len(rows)-1]; last.FallbackIndex != 1 || last.Status != "ok" {
		t.Errorf("audit = %+v", last)
	}
}

// DEF-R3-05 (TC-P102-14, TC-P103-19/20): một nhà lỗi 503 qua Gateway + Scheduler thật, làn BATCH (GRADING):
// (1) một lời gọi = 3 request (1 + 2 thử lại); (2) mạch mở → lời gọi kế KHÔNG gọi nhà và trả lỗi ngay; (3) hết 30 s, 3 lời gọi đồng thời = đúng MỘT thăm dò (3 request), không hơn.
func TestBatchOutageEndToEnd(t *testing.T) {
	t.Parallel()
	clk := clock.NewFake(time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC))
	down := &stubProv{name: "S4", chat: func(context.Context, int) (resultT, error) {
		return resultT{}, &provider.Error{Kind: provider.KindServer, Status: 503}
	}}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	reg.SetRoute(llm.TaskGrading, llm.Route{Targets: []llm.Target{{ProviderID: "S4", ProviderName: "S4", Type: "fake", Model: "m-503", RPM: 1_000_000, TPM: 1_000_000_000, P: down}}})
	r := finish(t, reg, scheduler.New(baseCfg(), nil, discard(), scheduler.WithClock(clk)), func(o *llm.Options) {
		o.Sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() } // không ngủ thật giữa các lần thử lại
	})
	lane := llm.LaneBatch
	call := func() error {
		_, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskGrading, Lane: &lane, Messages: msg("x")})
		return err
	}
	if err := call(); err == nil {
		t.Fatal("nhà 503 phải lỗi")
	}
	if n := down.calls.Load(); n != 3 {
		t.Fatalf("BATCH gọi nhà %d lần, muốn 3 (1 + 2 thử lại)", n)
	}
	for range 4 { // tới 5 lỗi liên tiếp
		_ = call()
	}
	if st := r.s.CircuitState(t.Context(), "S4"); st != "open" {
		t.Fatalf("sau 5 lỗi: %s", st)
	}
	before, t0 := down.calls.Load(), time.Now()
	if err := call(); err == nil {
		t.Fatal("mạch mở phải lỗi")
	}
	if down.calls.Load() != before || time.Since(t0) > 500*time.Millisecond {
		t.Fatalf("mạch mở: gọi nhà %d lần, mất %v (muốn 0 và < 500 ms)", down.calls.Load()-before, time.Since(t0))
	}
	clk.Advance(30 * time.Second)
	before = down.calls.Load()
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() { defer wg.Done(); _ = call() }()
	}
	wg.Wait()
	if n := down.calls.Load() - before; n != 3 {
		t.Fatalf("bán mở: 3 lời gọi đồng thời gọi nhà %d lần, muốn đúng 1 thăm dò = 3 request", n)
	}
}
