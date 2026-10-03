package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

// flightGroup hợp nhất các yêu cầu GIỐNG HỆT trong MỘT tiến trình (single-flight, SRS 4.3, US-P1-03 AC10).
// ponytail: hợp nhất giữa tiến trình cần khoá Redis + pub/sub — chỉ làm khi đo thấy trùng lặp đáng kể.
type flightGroup struct {
	mu sync.Mutex
	m  map[string]*flight
}

type flight struct {
	done   chan struct{}
	resp   Response
	err    error
	refs   int
	cancel context.CancelFunc
}

// shareable: chỉ Shareable=true, làn ≠ INTERACTIVE, tác vụ ≠ GRADING.
func shareable(r Request, lane Lane) bool {
	return r.Shareable && lane != LaneInteractive && r.Task != TaskGrading
}

func flightKey(r Request, model string) string {
	canon := struct {
		Task     Task
		Model    string
		Messages []Message
		Temp     *float64
		Max      int
	}{r.Task, model, r.Messages, r.Params.Temperature, r.Params.MaxTokens}
	b, _ := json.Marshal(canon)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// do chạy fn đúng một lần cho mọi người gọi cùng key đang chờ. Lời gọi chạy trên ctx tách khỏi người gọi đầu tiên (giữ giá trị,
// bỏ huỷ) và chỉ bị huỷ khi KHÔNG còn người chờ nào. Người chờ huỷ ctx của mình thì rời đi, lời gọi tiếp tục nếu còn người khác.
func (g *flightGroup) do(ctx context.Context, key string, fn func(ctx context.Context) (Response, error)) (Response, error, bool) {
	g.mu.Lock()
	if g.m == nil {
		g.m = map[string]*flight{}
	}
	f, shared := g.m[key]
	if !shared {
		callCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		if dl, ok := ctx.Deadline(); ok {
			var c2 context.CancelFunc
			callCtx, c2 = context.WithDeadline(callCtx, dl)
			prev := cancel
			cancel = func() { c2(); prev() }
		}
		f = &flight{done: make(chan struct{}), cancel: cancel}
		g.m[key] = f
		go func() {
			f.resp, f.err = fn(callCtx)
			g.mu.Lock()
			delete(g.m, key) // nhóm mới sau đó được thử lại độc lập
			g.mu.Unlock()
			cancel()
			close(f.done)
		}()
	}
	f.refs++
	g.mu.Unlock()

	select {
	case <-f.done:
		return f.resp, f.err, shared
	case <-ctx.Done():
		g.mu.Lock()
		f.refs--
		last := f.refs == 0
		g.mu.Unlock()
		if last {
			f.cancel()
		}
		return Response{}, ctx.Err(), shared
	}
}
