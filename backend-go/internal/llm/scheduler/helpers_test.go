package scheduler_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
)

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

type capture struct {
	mu   sync.Mutex
	rows []llm.AuditRow
}

func (c *capture) write(_ context.Context, rows []llm.AuditRow) error {
	c.mu.Lock()
	c.rows = append(c.rows, rows...)
	c.mu.Unlock()
	return nil
}

func (c *capture) all(g *llm.Gateway) []llm.AuditRow {
	g.FlushAudit(context.Background())
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]llm.AuditRow(nil), c.rows...)
}

type rig struct {
	g   *llm.Gateway
	s   *scheduler.Scheduler
	ctl *fake.Controller
	cap *capture
	reg *llm.Registry
}

func baseCfg() scheduler.Config {
	return scheduler.Config{MaxConcurrency: 10, BatchShare: 0.5, QueueMax: 200, QueueWaitMax: 10 * time.Second, BreakerFails: 5, BreakerOpen: 30 * time.Second}
}

// newRig dựng Gateway + Scheduler (chỉ cục bộ trừ khi truyền opts có Redis) trên provider fake của env.
func newRig(t *testing.T, cfg scheduler.Config, fs fake.Settings, gwOpts func(*llm.Options), schedOpts ...scheduler.Option) *rig {
	t.Helper()
	reg := llm.NewRegistry(nil, llm.EnvConfig{Provider: "fake", Fake: fs, DefaultRPM: 1_000_000, DefaultTPM: 1_000_000_000}, nil, discard())
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	s := scheduler.New(cfg, nil, discard(), schedOpts...)
	return finish(t, reg, s, gwOpts)
}

func finish(t *testing.T, reg *llm.Registry, s *scheduler.Scheduler, gwOpts func(*llm.Options)) *rig {
	t.Helper()
	cap := &capture{}
	aud := llm.NewAuditor(context.Background(), cap.write, discard())
	t.Cleanup(func() { aud.Close(context.Background()) })
	o := llm.Options{Registry: reg, Gate: s, Auditor: aud, Log: discard()}
	if gwOpts != nil {
		gwOpts(&o)
	}
	return &rig{g: llm.New(o), s: s, ctl: reg.Fake(), cap: cap, reg: reg}
}

func msg(s string) []llm.Message { return []llm.Message{{Role: "user", Content: s}} }

// stubProv là provider kịch bản hoá dùng cho test đa nhà cung cấp.
type stubProv struct {
	name  string
	calls atomic.Int64
	chat  func(ctx context.Context, n int) (provider.Result, error)
	// stream (nếu có) thay thế Stream mặc định.
	stream func(ctx context.Context) (<-chan provider.Delta, error)
}

func (p *stubProv) Chat(ctx context.Context, _ provider.ChatOpts) (provider.Result, error) {
	n := int(p.calls.Add(1))
	if p.chat != nil {
		return p.chat(ctx, n)
	}
	return provider.Result{Text: "ok từ " + p.name, TokensIn: 1, TokensOut: 1}, nil
}

func (p *stubProv) Stream(ctx context.Context, o provider.ChatOpts) (<-chan provider.Delta, error) {
	if p.stream != nil {
		p.calls.Add(1)
		return p.stream(ctx)
	}
	res, err := p.Chat(ctx, o)
	if err != nil {
		return nil, err
	}
	ch := make(chan provider.Delta, 2)
	ch <- provider.Delta{Text: res.Text}
	ch <- provider.Delta{Usage: &res}
	close(ch)
	return ch, nil
}

func (p *stubProv) Embed(context.Context, provider.EmbedOpts) ([][]float32, int, error) {
	return nil, 0, &provider.Error{Kind: provider.KindBadRequest}
}

func stubRig(t *testing.T, cfg scheduler.Config, ps ...*stubProv) *rig {
	t.Helper()
	var targets []llm.Target
	for _, p := range ps {
		targets = append(targets, llm.Target{ProviderID: p.name, ProviderName: p.name, Type: "fake", Model: "m-" + p.name, RPM: 1_000_000, TPM: 1_000_000_000, P: p})
	}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	for _, task := range llm.ChatTasks() {
		reg.SetRoute(task, llm.Route{Targets: targets})
	}
	return finish(t, reg, scheduler.New(cfg, nil, discard()), nil)
}

func ptr[T any](v T) *T { return &v }

func fakeSettings() fake.Settings { return fake.Settings{} }

type resultT = provider.Result
