package scheduler_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
)

func sleepMs(n int) { time.Sleep(time.Duration(n) * time.Millisecond) }

func shareable(task llm.Task, lane llm.Lane) llm.Request {
	return llm.Request{Task: task, Lane: &lane, Messages: msg("tóm tắt chương 3"), Shareable: true}
}

func TestSingleFlightShareable(t *testing.T) {
	t.Parallel()
	r := newRig(t, baseCfg(), fake.Settings{LatencyMin: 200 * time.Millisecond, LatencyMax: 200 * time.Millisecond}, nil)
	var wg sync.WaitGroup
	texts := make([]string, 50)
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := r.g.Chat(t.Context(), shareable(llm.TaskUtility, llm.LaneNearRealtime))
			if err != nil {
				t.Error(err)
			}
			texts[i] = resp.Text
		}()
	}
	wg.Wait()
	if r.ctl.Calls() != 1 {
		t.Fatalf("fake.Calls = %d, muốn 1", r.ctl.Calls())
	}
	for _, s := range texts {
		if s == "" || s != texts[0] {
			t.Fatalf("cả nhóm phải nhận cùng kết quả: %q vs %q", s, texts[0])
		}
	}
	// nhóm MỚI sau đó chạy lại độc lập
	if _, err := r.g.Chat(t.Context(), shareable(llm.TaskUtility, llm.LaneNearRealtime)); err != nil || r.ctl.Calls() != 2 {
		t.Errorf("nhóm mới: calls=%d err=%v", r.ctl.Calls(), err)
	}
	// khác nội dung → không gộp
	var wg2 sync.WaitGroup
	for i := range 3 {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			req := shareable(llm.TaskUtility, llm.LaneBatch)
			req.Messages = msg("câu khác " + string(rune('a'+i)))
			_, _ = r.g.Chat(t.Context(), req)
		}()
	}
	wg2.Wait()
	if r.ctl.Calls() != 5 {
		t.Errorf("3 yêu cầu khác nhau phải là 3 lời gọi: calls=%d", r.ctl.Calls())
	}
}

func TestNoSingleFlightInteractive(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  llm.Request
	}{
		{"INTERACTIVE", shareable(llm.TaskChat, llm.LaneInteractive)},
		{"GRADING", shareable(llm.TaskGrading, llm.LaneBatch)},
		{"không Shareable", func() llm.Request { r := shareable(llm.TaskUtility, llm.LaneBatch); r.Shareable = false; return r }()},
	} {
		r := newRig(t, baseCfg(), fake.Settings{LatencyMin: 100 * time.Millisecond, LatencyMax: 100 * time.Millisecond}, nil)
		var wg sync.WaitGroup
		for range 5 {
			wg.Add(1)
			go func() { defer wg.Done(); _, _ = r.g.Chat(t.Context(), tc.req) }()
		}
		wg.Wait()
		if r.ctl.Calls() != 5 {
			t.Errorf("%s: calls=%d, muốn 5 (không hợp nhất)", tc.name, r.ctl.Calls())
		}
	}
}

func TestSingleFlightCancelOneWaiter(t *testing.T) {
	t.Parallel()
	r := newRig(t, baseCfg(), fake.Settings{LatencyMin: 400 * time.Millisecond, LatencyMax: 400 * time.Millisecond}, nil)
	ctxA, cancelA := context.WithCancel(t.Context())
	results := make(chan error, 3)
	for i, ctx := range []context.Context{ctxA, t.Context(), t.Context()} {
		go func() {
			_, err := r.g.Chat(ctx, shareable(llm.TaskUtility, llm.LaneNearRealtime))
			results <- err
			_ = i
		}()
	}
	time.Sleep(100 * time.Millisecond)
	cancelA() // một người chờ huỷ
	var canceled, ok int
	for range 3 {
		if err := <-results; err == nil {
			ok++
		} else if err == context.Canceled {
			canceled++
		}
	}
	if canceled != 1 || ok != 2 || r.ctl.Calls() != 1 {
		t.Fatalf("canceled=%d ok=%d calls=%d: huỷ MỘT người chờ không được huỷ lời gọi", canceled, ok, r.ctl.Calls())
	}

	// huỷ TẤT CẢ → lời gọi bị huỷ, nhà cung cấp dừng
	r2 := newRig(t, baseCfg(), fake.Settings{LatencyMin: 5 * time.Second, LatencyMax: 5 * time.Second}, nil)
	ctx1, c1 := context.WithCancel(t.Context())
	ctx2, c2 := context.WithCancel(t.Context())
	done := make(chan struct{}, 2)
	for _, ctx := range []context.Context{ctx1, ctx2} {
		go func() { _, _ = r2.g.Chat(ctx, shareable(llm.TaskUtility, llm.LaneNearRealtime)); done <- struct{}{} }()
	}
	time.Sleep(100 * time.Millisecond)
	c1()
	c2()
	<-done
	<-done
	for i := 0; i < 100 && r2.ctl.Active() != 0; i++ {
		sleepMs(10)
	}
	if r2.ctl.Active() != 0 {
		t.Errorf("hết người chờ mà nhà cung cấp còn chạy: %d", r2.ctl.Active())
	}
}
