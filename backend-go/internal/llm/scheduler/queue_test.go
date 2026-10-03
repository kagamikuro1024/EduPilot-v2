package scheduler_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
)

func TestQueueFullOverloaded(t *testing.T) {
	t.Parallel()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 1, QueueMax: 3}, nil, discard())
	release := occupy(t, s, 1)
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var wg sync.WaitGroup
	for range 3 { // lấp đầy hàng INTERACTIVE
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = s.Admit(ctx, work(llm.LaneInteractive)) }()
	}
	deadline := time.Now().Add(time.Second)
	for s.Stats(ctx).QueueDepth["INTERACTIVE"] < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	start := time.Now()
	_, err := s.Admit(ctx, work(llm.LaneInteractive))
	var ov *llm.ErrOverloaded
	if !errors.As(err, &ov) {
		t.Fatalf("err = %v, muốn OVERLOADED", err)
	}
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("từ chối sau %v, muốn ≤ 50 ms", d)
	}
	if sec := int(ov.RetryAfter.Seconds()); sec < 1 || sec > 30 {
		t.Errorf("retry_after = %d, muốn 1–30", sec)
	}
	// yêu cầu bị từ chối không chiếm chỗ: vẫn đúng 1 chỗ đang giữ, hàng vẫn 3
	st := s.Stats(ctx)
	if st.Inflight["p"] != 1 || st.QueueDepth["INTERACTIVE"] != 3 {
		t.Errorf("stats = %+v", st)
	}
	// làn khác vẫn nhận (hàng đầy theo từng làn)
	cctx, ccancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer ccancel()
	if _, err := s.Admit(cctx, work(llm.LaneBatch)); errors.As(err, &ov) {
		t.Errorf("làn BATCH chưa đầy mà bị OVERLOADED")
	}
	cancel()
	wg.Wait()
}

func TestQueueWaitMax(t *testing.T) {
	t.Parallel()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 1, QueueWaitMax: 150 * time.Millisecond}, nil, discard())
	release := occupy(t, s, 1)
	defer release()
	start := time.Now()
	_, err := s.Admit(t.Context(), work(llm.LaneInteractive))
	var ov *llm.ErrOverloaded
	if !errors.As(err, &ov) {
		t.Fatalf("err = %v, muốn OVERLOADED sau LLM_QUEUE_WAIT_MAX", err)
	}
	if d := time.Since(start); d < 140*time.Millisecond || d > 600*time.Millisecond {
		t.Errorf("chờ %v, muốn ≈ 150 ms", d)
	}
	// NEAR_REALTIME không áp WAIT_MAX: chỉ bị hạn chót cắt
	ctx, cancel := context.WithTimeout(t.Context(), 400*time.Millisecond)
	defer cancel()
	start = time.Now()
	_, err = s.Admit(ctx, work(llm.LaneNearRealtime))
	if !errors.Is(err, llm.ErrDeadline) || time.Since(start) < 350*time.Millisecond {
		t.Errorf("NEAR_REALTIME: err=%v sau %v", err, time.Since(start))
	}
}

func TestRetryAfterFormula(t *testing.T) {
	t.Parallel()
	cases := []struct {
		qlen, max int
		avg       time.Duration
		want      int
	}{
		{0, 10, 5 * time.Second, 1},
		{1, 10, 100 * time.Millisecond, 1},
		{10, 10, 5 * time.Second, 5},
		{25, 10, 4 * time.Second, 10},
		{26, 10, 4 * time.Second, 11},  // ceil(10,4)
		{200, 10, 5 * time.Second, 30}, // 100 → kẹp 30
		{300, 1, 10 * time.Second, 30},
		{3, 0, 2 * time.Second, 6}, // max < 1 coi như 1
	}
	for _, tc := range cases {
		if got := int(scheduler.RetryAfter(tc.qlen, tc.max, tc.avg).Seconds()); got != tc.want {
			t.Errorf("RetryAfter(%d, %d, %v) = %d, muốn %d", tc.qlen, tc.max, tc.avg, got, tc.want)
		}
	}
}

func TestDeadlineIncludesQueueWait(t *testing.T) {
	t.Parallel()
	cfg := baseCfg()
	cfg.MaxConcurrency = 1
	r := newRig(t, cfg, fake.Settings{LatencyMin: 1500 * time.Millisecond, LatencyMax: 1500 * time.Millisecond}, nil)
	lane := llm.LaneNearRealtime
	first := make(chan error, 1)
	go func() {
		_, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskUtility, Lane: &lane, Messages: msg("giữ chỗ")})
		first <- err
	}()
	time.Sleep(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := r.g.Chat(ctx, llm.Request{Task: llm.TaskUtility, Lane: &lane, Messages: msg("bị trừ hạn")})
	if !errors.Is(err, llm.ErrDeadline) {
		t.Fatalf("err = %v, muốn ErrDeadline", err)
	}
	if d := time.Since(start); d > 600*time.Millisecond {
		t.Errorf("trả sau %v, muốn ≈ 300 ms (chờ hàng được trừ vào hạn)", d)
	}
	if r.ctl.Calls() != 1 {
		t.Errorf("lời gọi thứ hai không được tới nhà cung cấp: Calls=%d", r.ctl.Calls())
	}
	<-first
}

func TestDeadlineDefaults(t *testing.T) {
	t.Parallel()
	r := newRig(t, baseCfg(), fake.Settings{LatencyMin: 2 * time.Second, LatencyMax: 2 * time.Second}, func(o *llm.Options) {
		o.RequestTimeout, o.BatchTimeout = 200*time.Millisecond, 500*time.Millisecond
	})
	for _, tc := range []struct {
		task     llm.Task
		min, max time.Duration
	}{
		{llm.TaskChat, 150 * time.Millisecond, 450 * time.Millisecond},    // INTERACTIVE: LLM_REQUEST_TIMEOUT
		{llm.TaskUtility, 150 * time.Millisecond, 450 * time.Millisecond}, // NEAR_REALTIME: như trên
		{llm.TaskInsight, 450 * time.Millisecond, 900 * time.Millisecond}, // BATCH: mặc định riêng
	} {
		start := time.Now()
		_, err := r.g.Chat(context.Background(), llm.Request{Task: tc.task, Messages: msg("x")})
		d := time.Since(start)
		if !errors.Is(err, llm.ErrDeadline) || d < tc.min || d > tc.max {
			t.Errorf("%s: err=%v sau %v, muốn ErrDeadline trong [%v, %v]", tc.task, err, d, tc.min, tc.max)
		}
	}
}

func TestTimeoutParamOnlyNarrows(t *testing.T) {
	t.Parallel()
	slow := &stubProv{name: "S", chat: func(ctx context.Context, _ int) (resultT, error) {
		<-ctx.Done()
		return resultT{}, ctx.Err()
	}}
	mk := func(timeoutS string) *rig {
		reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
		reg.SetRoute(llm.TaskUtility, llm.Route{Targets: []llm.Target{{ProviderID: "S", ProviderName: "S", Model: "m", RPM: 1e6, TPM: 1e9, P: slow}},
			Params: map[string]json.Number{"timeout_s": json.Number(timeoutS)}})
		return finish(t, reg, scheduler.New(baseCfg(), nil, discard()), nil)
	}
	// timeout_s=1 thu hẹp hạn mặc định 30 s xuống 1 s
	r := mk("1")
	start := time.Now()
	_, err := r.g.Chat(context.Background(), llm.Request{Task: llm.TaskUtility, Messages: msg("x")})
	if !errors.Is(err, llm.ErrDeadline) || time.Since(start) < 900*time.Millisecond || time.Since(start) > 1800*time.Millisecond {
		t.Errorf("timeout_s=1: err=%v sau %v", err, time.Since(start))
	}
	// timeout_s=60 KHÔNG nới hạn 250 ms của ctx
	r = mk("60")
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	start = time.Now()
	_, err = r.g.Chat(ctx, llm.Request{Task: llm.TaskUtility, Messages: msg("x")})
	if !errors.Is(err, llm.ErrDeadline) || time.Since(start) > 700*time.Millisecond {
		t.Errorf("timeout_s=60: err=%v sau %v, hạn ctx 250 ms phải thắng", err, time.Since(start))
	}
}

func TestClientCancelStopsProvider(t *testing.T) {
	t.Parallel()
	cfg := baseCfg()
	r := newRig(t, cfg, fake.Settings{LatencyMin: 5 * time.Second, LatencyMax: 5 * time.Second}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := r.g.Chat(ctx, llm.Request{Task: llm.TaskChat, Messages: msg("hỏi")}); done <- err }()
	for i := 0; i < 100 && r.ctl.Active() == 0; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	start := time.Now()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, muốn context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("không dừng ≤ 1 s sau khi huỷ")
	}
	if time.Since(start) > time.Second {
		t.Errorf("dừng sau %v", time.Since(start))
	}
	st := r.s.Stats(t.Context())
	if r.ctl.Active() != 0 || st.ProviderInflight != 0 {
		t.Errorf("còn chạy: fake=%d inflight=%d", r.ctl.Active(), st.ProviderInflight)
	}
	rows := r.cap.all(r.g)
	if len(rows) != 1 || rows[0].Status != "cancelled" {
		t.Errorf("audit = %+v", rows)
	}
}

func TestCancelNoRetryNoFallback(t *testing.T) {
	t.Parallel()
	a := &stubProv{name: "A", chat: func(ctx context.Context, _ int) (resultT, error) {
		<-ctx.Done()
		return resultT{}, ctx.Err()
	}}
	b := &stubProv{name: "B"}
	r := stubRig(t, baseCfg(), a, b)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	lane := llm.LaneNearRealtime
	go func() {
		_, err := r.g.Chat(ctx, llm.Request{Task: llm.TaskUtility, Lane: &lane, Messages: msg("x")})
		done <- err
	}()
	time.Sleep(80 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if a.calls.Load() != 1 || b.calls.Load() != 0 {
		t.Errorf("A=%d B=%d: huỷ rồi thì không thử lại, không chuyển dự phòng", a.calls.Load(), b.calls.Load())
	}
}
