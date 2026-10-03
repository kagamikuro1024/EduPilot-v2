package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llmconfig"
)

// ReloadChannel là kênh pub/sub báo cấu hình đã đổi (SRS 5.5).
const ReloadChannel = "ep:llm:reload"

// PollEvery là chu kỳ thăm dò dự phòng khi mất thông báo pub/sub.
const PollEvery = 60 * time.Second

// EnvConfig là cấu hình dự phòng từ biến môi trường (SRS 4.2, 8.1).
type EnvConfig struct {
	Provider                           string // "fake" hoặc rỗng
	OpenAIKey, AnthropicKey, GeminiKey string
	DefaultRPM, DefaultTPM             int
	Fake                               fake.Settings
}

// Target là một mô hình đã dựng sẵn trong chuỗi của tác vụ.
type Target struct {
	ProviderID   string
	ProviderName string
	Type         string
	Model        string
	PriceIn      decimal.Decimal
	PriceOut     decimal.Decimal
	RPM, TPM     int
	P            provider.Provider
}

// Route là chuỗi mô hình của một tác vụ (phần tử 0 = chính) và tham số của tuyến.
type Route struct {
	Targets []Target
	Params  llmconfig.Params
}

type regState struct {
	routes   map[Task]Route
	envOnly  bool
	cheapest *Target // mô hình chat rẻ nhất đang bật (ngân sách cạn → chat chuyển sang đây)
}

// ProviderSpec là đầu vào của Factory.
type ProviderSpec struct {
	ID, Name, Type, BaseURL, Key string
}

// Factory dựng provider từ bản ghi; mặc định: fake → fake.New, còn lại → provider.NewOpenAI.
type Factory func(ProviderSpec) (provider.Provider, error)

// Registry giữ bộ client hiện hành và hoán đổi nguyên tử khi cấu hình đổi (US-P1-02 AC4, AC14).
type Registry struct {
	res     *llmconfig.Resolver
	env     EnvConfig
	factory Factory
	ctl     *fake.Controller
	log     *slog.Logger
	cur     atomic.Pointer[regState]
	mu      sync.Mutex // một lần nạp tại một thời điểm
	hooksMu sync.Mutex
	hooks   []func()
}

// OnReload đăng ký hàm gọi SAU mỗi lần nạp thành công (Load/Reload) — dùng để vô hiệu cache giới hạn ngân sách.
func (r *Registry) OnReload(f func()) {
	r.hooksMu.Lock()
	r.hooks = append(r.hooks, f)
	r.hooksMu.Unlock()
}

func (r *Registry) fire() {
	r.hooksMu.Lock()
	hs := append([]func(){}, r.hooks...)
	r.hooksMu.Unlock()
	for _, f := range hs {
		f()
	}
}

// Cheapest trả mô hình chat rẻ nhất đang bật (price_in + price_out nhỏ nhất, hoà thì theo thứ tự tạo).
func (r *Registry) Cheapest() (Target, bool) {
	st := r.cur.Load()
	if st == nil || st.cheapest == nil {
		return Target{}, false
	}
	return *st.cheapest, true
}

// NewRegistry dựng Registry. res có thể nil (chỉ dùng env — test). factory nil = mặc định.
func NewRegistry(res *llmconfig.Resolver, env EnvConfig, factory Factory, log *slog.Logger) *Registry {
	r := &Registry{res: res, env: env, factory: factory, log: log, ctl: fake.NewController(env.Fake)}
	if r.env.DefaultRPM == 0 {
		r.env.DefaultRPM = 60
	}
	if r.env.DefaultTPM == 0 {
		r.env.DefaultTPM = 100000
	}
	if r.factory == nil {
		r.factory = r.defaultFactory
	}
	return r
}

// NewStaticRegistry dựng Registry chỉ-bộ-nhớ từ tuyến cho sẵn (không DB, không env) — dùng cho test và công cụ đo.
func NewStaticRegistry(routes map[Task]Route) *Registry {
	r := &Registry{ctl: fake.NewController(fake.Settings{})}
	r.cur.Store(&regState{routes: routes})
	return r
}

// SetRoute đặt tuyến của một tác vụ (hoán đổi nguyên tử).
func (r *Registry) SetRoute(t Task, rt Route) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.cur.Load()
	next := &regState{routes: map[Task]Route{}, envOnly: st.envOnly, cheapest: st.cheapest}
	for k, v := range st.routes {
		next.routes[k] = v
	}
	next.routes[t] = rt
	if next.cheapest == nil && t == TaskChat && len(rt.Targets) > 0 {
		c := rt.Targets[0]
		next.cheapest = &c
	}
	r.cur.Store(next)
}

// Fake là Controller của các provider fake (route thử `_test/llm/fake`, bộ đếm của test).
func (r *Registry) Fake() *fake.Controller { return r.ctl }

func (r *Registry) defaultFactory(s ProviderSpec) (provider.Provider, error) {
	switch s.Type {
	case "fake":
		return fake.NewNamed(r.ctl, s.ID, s.Key), nil
	case "openai", "anthropic", "gemini", "openai_compatible":
		return provider.NewOpenAI(provider.OpenAIConfig{Type: s.Type, BaseURL: s.BaseURL, APIKey: s.Key}), nil
	}
	return nil, fmt.Errorf("loại nhà cung cấp không hỗ trợ: %q", s.Type)
}

// Build dựng provider (dùng cho Test kết nối ở US-P1-04).
func (r *Registry) Build(s ProviderSpec) (provider.Provider, error) { return r.factory(s) }

// Load nạp lần đầu; lỗi → trả lỗi (gateway quyết định). Reload giữ cấu hình cũ khi lỗi.
func (r *Registry) Load(ctx context.Context) error {
	st, err := r.build(ctx)
	if err != nil {
		return err
	}
	r.cur.Store(st)
	r.fire()
	return nil
}

// Reload nạp lại và hoán đổi nguyên tử; lỗi → giữ cấu hình cũ, log error, trả lỗi.
func (r *Registry) Reload(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.build(ctx)
	if err != nil {
		r.log.ErrorContext(ctx, "nạp lại cấu hình LLM thất bại, giữ cấu hình cũ", "error", err.Error())
		return err
	}
	r.cur.Store(st)
	r.fire()
	return nil
}

// Route trả chuỗi của tác vụ; ErrNotConfigured nếu không có.
func (r *Registry) Route(t Task) (Route, error) {
	st := r.cur.Load()
	if st == nil {
		return Route{}, ErrNotConfigured
	}
	rt, ok := st.routes[t]
	if !ok || len(rt.Targets) == 0 {
		return Route{}, ErrNotConfigured
	}
	return rt, nil
}

// EnvActive cho biết cấu hình hiện hành hoàn toàn từ env (DB không có nhà cung cấp bật).
func (r *Registry) EnvActive() bool {
	st := r.cur.Load()
	return st != nil && st.envOnly
}

// EnvProviders là tên các nhà cung cấp đang chạy từ env dự phòng (rỗng khi cấu hình đến từ DB), thứ tự ổn định.
func (r *Registry) EnvProviders() []string {
	st := r.cur.Load()
	if st == nil || !st.envOnly {
		return []string{}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, t := range append(ChatTasks(), TaskEmbedding) {
		for _, tg := range st.routes[t].Targets {
			if !seen[tg.Type] {
				seen[tg.Type] = true
				out = append(out, tg.Type)
			}
		}
	}
	return out
}

func (r *Registry) build(ctx context.Context) (*regState, error) {
	st := &regState{routes: map[Task]Route{}}
	dbTargets := 0
	if r.res != nil {
		snap, err := r.res.Snapshot(ctx)
		if err != nil {
			return nil, fmt.Errorf("đọc cấu hình: %w", err)
		}
		built := map[uuid.UUID]provider.Provider{}
		for _, p := range snap.Providers {
			if !p.Enabled {
				continue
			}
			key := ""
			if p.HasKey {
				k, err := r.res.DecryptKey(ctx, p.ID)
				if err != nil {
					if errors.Is(err, llmconfig.ErrKeyUnreadable) {
						r.log.WarnContext(ctx, "khoá nhà cung cấp không đọc được, bỏ qua", "provider", p.Name)
					}
					continue
				}
				key = k
			} else if p.Type != "fake" && p.Type != "openai_compatible" {
				continue // không có khoá thì không gọi được
			}
			base := ""
			if p.BaseURL != nil {
				base = *p.BaseURL
			}
			pv, err := r.factory(ProviderSpec{ID: p.ID.String(), Name: p.Name, Type: p.Type, BaseURL: base, Key: key})
			if err != nil {
				r.log.WarnContext(ctx, "không dựng được nhà cung cấp, bỏ qua", "provider", p.Name, "error", err.Error())
				continue
			}
			built[p.ID] = pv
		}
		limits := map[uuid.UUID][2]int{}
		for _, p := range snap.Providers {
			rpm, tpm := r.env.DefaultRPM, r.env.DefaultTPM
			if p.RPMLimit != nil {
				rpm = *p.RPMLimit
			}
			if p.TPMLimit != nil {
				tpm = *p.TPMLimit
			}
			limits[p.ID] = [2]int{rpm, tpm}
		}
		for _, p := range snap.Providers {
			pv, ok := built[p.ID]
			if !ok {
				continue
			}
			rpm, tpm := limits[p.ID][0], limits[p.ID][1]
			for _, m := range p.Models {
				if m.Kind != "chat" || !m.Enabled {
					continue
				}
				if st.cheapest == nil || m.PriceIn.Add(m.PriceOut).LessThan(st.cheapest.PriceIn.Add(st.cheapest.PriceOut)) {
					st.cheapest = &Target{ProviderID: p.ID.String(), ProviderName: p.Name, Type: p.Type, Model: m.Model,
						PriceIn: m.PriceIn, PriceOut: m.PriceOut, RPM: rpm, TPM: tpm, P: pv}
				}
			}
		}
		for taskName, rt := range snap.Routes {
			var targets []Target
			for _, e := range rt.Chain {
				pv, ok := built[e.ProviderID]
				if !ok || !e.ProviderEnabled || !e.ModelEnabled {
					continue
				}
				typ := ""
				for _, p := range snap.Providers {
					if p.ID == e.ProviderID {
						typ = p.Type
					}
				}
				lim := limits[e.ProviderID]
				targets = append(targets, Target{ProviderID: e.ProviderID.String(), ProviderName: e.ProviderName, Type: typ, Model: e.Model,
					PriceIn: e.PriceIn, PriceOut: e.PriceOut, RPM: lim[0], TPM: lim[1], P: pv})
			}
			if len(targets) > 0 {
				st.routes[Task(taskName)] = Route{Targets: targets, Params: rt.Params}
				dbTargets += len(targets)
			}
		}
	}
	envRoutes, err := r.envRoutes()
	if err != nil {
		return nil, err
	}
	for t, rt := range envRoutes {
		if _, ok := st.routes[t]; !ok { // DB thắng; env chỉ lấp tác vụ chưa có tuyến
			st.routes[t] = rt
		}
	}
	st.envOnly = dbTargets == 0
	if st.cheapest == nil { // không có mô hình chat trong DB: lấy mô hình chính của CHAT (env)
		if rt, ok := st.routes[TaskChat]; ok && len(rt.Targets) > 0 {
			t := rt.Targets[0]
			st.cheapest = &t
		}
	}
	return st, nil
}

// envRoutes dựng tuyến dự phòng từ env (SRS 4.2).
func (r *Registry) envRoutes() (map[Task]Route, error) {
	out := map[Task]Route{}
	lim := func() (int, int) { return r.env.DefaultRPM, r.env.DefaultTPM }
	rpm, tpm := lim()
	mk := func(id, typ, key, model string) (Target, error) {
		pv, err := r.factory(ProviderSpec{ID: id, Name: id, Type: typ, Key: key})
		if err != nil {
			return Target{}, err
		}
		return Target{ProviderID: id, ProviderName: id, Type: typ, Model: model, RPM: rpm, TPM: tpm, P: pv}, nil
	}
	if r.env.Provider == "fake" {
		chat, err := mk("env:fake", "fake", "", FakeChatModel)
		if err != nil {
			return nil, err
		}
		emb := chat
		emb.Model = FakeEmbedModel
		for _, t := range ChatTasks() {
			out[t] = Route{Targets: []Target{chat}}
		}
		out[TaskEmbedding] = Route{Targets: []Target{emb}}
		return out, nil
	}
	var chain []Target
	add := func(id, typ, key, model string) error {
		if key == "" {
			return nil
		}
		t, err := mk(id, typ, key, model)
		if err != nil {
			return err
		}
		chain = append(chain, t)
		return nil
	}
	if err := add("env:openai", "openai", r.env.OpenAIKey, DefaultOpenAIChat); err != nil {
		return nil, err
	}
	if err := add("env:anthropic", "anthropic", r.env.AnthropicKey, DefaultAnthropicChat); err != nil {
		return nil, err
	}
	if err := add("env:gemini", "gemini", r.env.GeminiKey, DefaultGeminiChat); err != nil {
		return nil, err
	}
	if len(chain) > 0 {
		for _, t := range ChatTasks() {
			out[t] = Route{Targets: append([]Target(nil), chain...)}
		}
	}
	if r.env.OpenAIKey != "" { // nhúng: chỉ OpenAI (Anthropic không có endpoint, Gemini khác không gian vectơ)
		e, err := mk("env:openai", "openai", r.env.OpenAIKey, DefaultOpenAIEmbedding)
		if err != nil {
			return nil, err
		}
		out[TaskEmbedding] = Route{Targets: []Target{e}}
	}
	return out, nil
}

// Watch nghe thông báo nạp lại (pub/sub) và thăm dò mỗi PollEvery cho tới khi ctx kết thúc.
func (r *Registry) Watch(ctx context.Context, rdb *goredis.Client) {
	var msgs <-chan *goredis.Message
	if rdb != nil {
		ps := rdb.Subscribe(ctx, ReloadChannel)
		defer func() { _ = ps.Close() }()
		msgs = ps.Channel()
	}
	t := time.NewTicker(PollEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case _, ok := <-msgs:
			if !ok {
				msgs = nil
				continue
			}
		}
		_ = r.Reload(ctx) // lỗi đã được log; giữ cấu hình cũ
	}
}

// PublishReload báo mọi tiến trình gateway nạp lại cấu hình.
func PublishReload(ctx context.Context, rdb *goredis.Client) error {
	return rdb.Publish(ctx, ReloadChannel, "1").Err()
}
