package llm_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/platform/config"
	applog "github.com/edupilot/backend-go/internal/platform/log"
)

// stub là provider kịch bản hoá: mỗi lần gọi đi qua fn(lần thứ n).
type stub struct {
	name   string
	calls  atomic.Int64
	chat   func(n int, o provider.ChatOpts) (provider.Result, error)
	embed  func(n int, o provider.EmbedOpts) ([][]float32, int, error)
	stream func(ctx context.Context, n int, o provider.ChatOpts) (<-chan provider.Delta, error)
	seen   []provider.ChatOpts
	mu     sync.Mutex
}

func (s *stub) Chat(_ context.Context, o provider.ChatOpts) (provider.Result, error) {
	n := int(s.calls.Add(1))
	s.mu.Lock()
	s.seen = append(s.seen, o)
	s.mu.Unlock()
	if s.chat == nil {
		return provider.Result{Text: "ok từ " + s.name, TokensIn: 10, TokensOut: 5}, nil
	}
	return s.chat(n, o)
}

func (s *stub) Stream(ctx context.Context, o provider.ChatOpts) (<-chan provider.Delta, error) {
	n := int(s.calls.Add(1))
	if s.stream != nil {
		return s.stream(ctx, n, o)
	}
	ch := make(chan provider.Delta, 3)
	ch <- provider.Delta{Text: "xin "}
	ch <- provider.Delta{Text: "chào"}
	ch <- provider.Delta{Usage: &provider.Result{TokensIn: 7, TokensOut: 2}}
	close(ch)
	return ch, nil
}

func (s *stub) Embed(_ context.Context, o provider.EmbedOpts) ([][]float32, int, error) {
	n := int(s.calls.Add(1))
	if s.embed != nil {
		return s.embed(n, o)
	}
	out := make([][]float32, len(o.Inputs))
	for i := range out {
		out[i] = make([]float32, llm.EmbedDims)
		out[i][0] = 1
	}
	return out, len(o.Inputs), nil
}

func perr(k provider.Kind, status int) *provider.Error {
	return &provider.Error{Kind: k, Status: status}
}

// newGateway dựng Gateway với mọi tác vụ trỏ cùng chuỗi ps; jitter = trần (Rand=1), Sleep ghi lại độ trễ thay vì ngủ.
func newGateway(t *testing.T, ps ...*stub) (*llm.Gateway, *capture) {
	t.Helper()
	return newGatewayLog(t, newDiscardLog(), ps...)
}

func newDiscardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newGatewayLog(t *testing.T, log *slog.Logger, ps ...*stub) (*llm.Gateway, *capture) {
	t.Helper()
	return newGatewayFull(t, log, nil, ps...)
}

// newGatewayWithGate như newGateway nhưng gắn Gate tuỳ ý (ghi mạch ngắt, giả lập hàng đợi…).
func newGatewayWithGate(t *testing.T, gate llm.Gate, ps ...*stub) (*llm.Gateway, *capture) {
	t.Helper()
	return newGatewayFull(t, newDiscardLog(), gate, ps...)
}

func newGatewayFull(t *testing.T, log *slog.Logger, gate llm.Gate, ps ...*stub) (*llm.Gateway, *capture) {
	t.Helper()
	cap := &capture{}
	aud := llm.NewAuditor(context.Background(), cap.write, log)
	t.Cleanup(func() { aud.Close(context.Background()) })
	targets := make([]llm.Target, 0, len(ps))
	for _, p := range ps {
		targets = append(targets, llm.Target{ProviderID: p.name, ProviderName: p.name, Type: "fake", Model: "m-" + p.name,
			PriceIn: decimal.RequireFromString("1000"), PriceOut: decimal.RequireFromString("2000"), RPM: 60, TPM: 100000, P: p})
	}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	for _, task := range append(llm.ChatTasks(), llm.TaskEmbedding) {
		reg.SetRoute(task, llm.Route{Targets: targets})
	}
	g := llm.New(llm.Options{Registry: reg, Auditor: aud, Log: log, Gate: gate,
		Sleep: func(ctx context.Context, d time.Duration) error { cap.sleeps = append(cap.sleeps, d); return ctx.Err() },
		Rand:  func() float64 { return 1 }}) // jitter = đúng trần
	return g, cap
}

type capture struct {
	mu     sync.Mutex
	rows   []llm.AuditRow
	sleeps []time.Duration
	fail   atomic.Bool
	batch  []int
}

func (c *capture) write(_ context.Context, rows []llm.AuditRow) error {
	if c.fail.Load() {
		return io.ErrClosedPipe
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = append(c.rows, rows...)
	c.batch = append(c.batch, len(rows))
	return nil
}

func (c *capture) all(t *testing.T, g *llm.Gateway) []llm.AuditRow {
	t.Helper()
	g.FlushAudit(context.Background())
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]llm.AuditRow(nil), c.rows...)
}

func userMsg(s string) []llm.Message { return []llm.Message{{Role: "user", Content: s}} }

func jsonBuf() (*bytes.Buffer, *slog.Logger) {
	b := &bytes.Buffer{}
	return b, applog.NewTo(b, config.Config{LogLevel: "debug"}, "test")
}
