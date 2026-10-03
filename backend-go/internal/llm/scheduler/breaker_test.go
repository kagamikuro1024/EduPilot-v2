package scheduler_test

import (
	"context"
	"errors"
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
	start := time.Now()
	resp, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg("x")})
	if err != nil || resp.Provider != "B" || resp.FallbackIndex != 1 {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if a.calls.Load() != callsA {
		t.Errorf("mạch mở nhưng vẫn gọi A (%d → %d)", callsA, a.calls.Load())
	}
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("bỏ qua nhà mạch mở mất %v", d)
	}
	rows := r.cap.all(r.g)
	if last := rows[len(rows)-1]; last.FallbackIndex != 1 || last.Status != "ok" {
		t.Errorf("audit = %+v", last)
	}
}
