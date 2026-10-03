package scheduler_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
)

func work(lane llm.Lane) llm.Work {
	return llm.Work{Lane: lane, Task: llm.TaskChat, ProviderID: "p", RPM: 1_000_000, TPM: 1_000_000_000, EstTokens: 10}
}

// occupy giữ n chỗ của nhà "p" tới khi gọi release.
func occupy(t *testing.T, s *scheduler.Scheduler, n int) (release func()) {
	t.Helper()
	var ps []llm.Permit
	for range n {
		p, err := s.Admit(t.Context(), work(llm.LaneBatch))
		if err != nil {
			t.Fatal(err)
		}
		ps = append(ps, p)
	}
	return func() {
		for _, p := range ps {
			p.Done(-1)
		}
	}
}

// grantOrder xếp hàng lần lượt các làn (cách nhau 30 ms để chắc thứ tự đến) rồi trả thứ tự được cấp.
func grantOrder(t *testing.T, s *scheduler.Scheduler, lanes []llm.Lane) []int {
	t.Helper()
	release := occupy(t, s, 1)
	var mu sync.Mutex
	var order []int
	var wg sync.WaitGroup
	for i, lane := range lanes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := s.Admit(t.Context(), work(lane))
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			p.Done(-1)
		}()
		time.Sleep(30 * time.Millisecond)
	}
	release()
	wg.Wait()
	return order
}

func TestLaneOrder(t *testing.T) {
	t.Parallel()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 1, BatchShare: 1}, nil, discard())
	// đến theo thứ tự BATCH, NEAR_REALTIME, INTERACTIVE → cấp theo INTERACTIVE, NEAR_REALTIME, BATCH
	got := grantOrder(t, s, []llm.Lane{llm.LaneBatch, llm.LaneNearRealtime, llm.LaneInteractive})
	want := []int{2, 1, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("thứ tự cấp = %v, muốn %v", got, want)
		}
	}
}

func TestFIFOWithinLane(t *testing.T) {
	t.Parallel()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 1, BatchShare: 1}, nil, discard())
	got := grantOrder(t, s, []llm.Lane{llm.LaneBatch, llm.LaneBatch, llm.LaneBatch, llm.LaneBatch})
	for i, v := range got {
		if v != i {
			t.Fatalf("FIFO sai: %v", got)
		}
	}
}

func TestDefaultLaneTable(t *testing.T) {
	t.Parallel()
	want := map[llm.Task]llm.Lane{
		llm.TaskChat: llm.LaneInteractive, llm.TaskClassify: llm.LaneNearRealtime, llm.TaskUtility: llm.LaneNearRealtime,
		llm.TaskGrading: llm.LaneBatch, llm.TaskQuestionGen: llm.LaneBatch, llm.TaskInsight: llm.LaneBatch, llm.TaskEmbedding: llm.LaneBatch,
	}
	for task, lane := range want {
		if got := llm.DefaultLane(task); got != lane {
			t.Errorf("%s: %v, muốn %v", task, got, lane)
		}
	}
	// EMBEDDING được nâng lên INTERACTIVE (vectơ hoá câu hỏi lúc chat)
	if l, err := llm.ResolveLane(llm.TaskEmbedding, ptr(llm.LaneInteractive)); err != nil || l != llm.LaneInteractive {
		t.Errorf("EMBEDDING nâng lên INTERACTIVE: %v %v", l, err)
	}
}

func TestBadLane(t *testing.T) {
	t.Parallel()
	r := newRig(t, baseCfg(), fakeSettings(), nil)
	for _, task := range []llm.Task{llm.TaskGrading, llm.TaskQuestionGen, llm.TaskInsight} {
		_, err := r.g.Chat(t.Context(), llm.Request{Task: task, Lane: ptr(llm.LaneInteractive), Messages: msg("x")})
		if !errors.Is(err, llm.ErrBadLane) {
			t.Errorf("%s nâng lên INTERACTIVE: err = %v", task, err)
		}
	}
	if r.ctl.Calls() != 0 {
		t.Error("không được gọi nhà cung cấp khi làn sai")
	}
}

func TestBatchShareCapsBatchWhenInteractiveWaits(t *testing.T) {
	t.Parallel()
	s := scheduler.New(scheduler.Config{MaxConcurrency: 10, BatchShare: 0.5}, nil, discard())
	var held []llm.Permit
	for range 5 { // BATCH chiếm tới trần ceil(10×0,5)=5 khi không có INTERACTIVE
		p, err := s.Admit(t.Context(), work(llm.LaneBatch))
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, p)
	}
	// một INTERACTIVE đang chạy → BATCH thứ 6 không được cấp (trần 5), dù còn chỗ trống
	pi, err := s.Admit(t.Context(), work(llm.LaneInteractive))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	if _, err := s.Admit(ctx, work(llm.LaneBatch)); !errors.Is(err, llm.ErrDeadline) {
		t.Fatalf("BATCH thứ 6 khi có INTERACTIVE chạy: err = %v, muốn bị giữ lại", err)
	}
	pi.Done(-1)
	time.Sleep(2100 * time.Millisecond) // quá cửa sổ 2 s kể từ INTERACTIVE gần nhất → BATCH được dùng hết công suất
	ctx2, cancel2 := context.WithTimeout(t.Context(), time.Second)
	defer cancel2()
	p, err := s.Admit(ctx2, work(llm.LaneBatch))
	if err != nil {
		t.Fatalf("hết INTERACTIVE: BATCH phải được cấp: %v", err)
	}
	p.Done(-1)
	for _, h := range held {
		h.Done(-1)
	}
}
