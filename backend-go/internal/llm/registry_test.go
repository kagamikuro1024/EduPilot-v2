package llm_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/testutil"
)

type cfgRig struct {
	pool *pgxpool.Pool
	svc  *llmconfig.Service
	res  *llmconfig.Resolver
	ctx  context.Context
}

func newCfgRig(t *testing.T) *cfgRig {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	key := make([]byte, 32)
	_, err = rand.Read(key)
	require.NoError(t, err)
	c, err := crypto.NewCipher(key)
	require.NoError(t, err)
	ctx := auth.WithPrincipal(t.Context(), auth.Principal{Sub: "00000000-0000-7000-8000-0000000000a0", Role: auth.RoleAdmin})
	return &cfgRig{pool: pool, svc: llmconfig.New(pool, c), res: llmconfig.NewResolver(pool, c), ctx: ctx}
}

func (r *cfgRig) add(t *testing.T, typ, name, key, baseURL string, models ...llmconfig.ModelInput) llmconfig.Provider {
	t.Helper()
	in := llmconfig.ProviderInput{Type: typ, Name: name, Models: models}
	if key != "" {
		k := llmconfig.NewSecret(key)
		in.APIKey = &k
	}
	if baseURL != "" {
		in.BaseURL = &baseURL
	}
	p, err := r.svc.CreateProvider(r.ctx, in)
	require.NoError(t, err)
	return p
}

func cm(name string) llmconfig.ModelInput {
	return llmconfig.ModelInput{Model: name, Kind: "chat", PriceIn: decimal.RequireFromString("4000"), PriceOut: decimal.RequireFromString("16000")}
}

func TestRegistry(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	oa := r.add(t, "openai", "OA", "sk-oa", "", cm("gpt-x"))
	an := r.add(t, "anthropic", "AN", "sk-an", "", cm("claude-x"))
	ge := r.add(t, "gemini", "GE", "sk-ge", "", cm("gem-x"))
	oc := r.add(t, "openai_compatible", "OC", "", "http://vllm.local:8000/v1", cm("llama"))
	fk := r.add(t, "fake", "FK", "", "", cm("fake-chat"))
	off := r.add(t, "fake", "OFF", "", "", cm("off-chat"))
	bad := r.add(t, "openai", "BAD", "sk-bad", "", cm("bad-chat"))

	chain := []uuid.UUID{oa.Models[0].ID, an.Models[0].ID, ge.Models[0].ID, oc.Models[0].ID}
	_, err := r.svc.SetRoute(r.ctx, "CHAT", chain, nil, 0)
	require.NoError(t, err)
	_, err = r.svc.SetRoute(r.ctx, "CLASSIFY", []uuid.UUID{off.Models[0].ID, bad.Models[0].ID, fk.Models[0].ID}, nil, 0)
	require.NoError(t, err)
	// tắt OFF; làm hỏng khoá BAD (AAD sai) — registry phải bỏ qua, không sập
	disabled := false
	_, err = r.svc.UpdateProvider(r.ctx, off.ID, llmconfig.ProviderInput{Type: "fake", Name: "OFF", Enabled: &disabled, Version: off.Version})
	require.NoError(t, err)
	_, err = r.pool.Exec(t.Context(), `update llm_providers set api_key_enc = $2 where id = $1`, bad.ID, append([]byte{1}, make([]byte, 40)...))
	require.NoError(t, err)

	seen := map[string]llm.ProviderSpec{}
	var mu sync.Mutex
	reg := llm.NewRegistry(r.res, llm.EnvConfig{}, func(s llm.ProviderSpec) (provider.Provider, error) {
		mu.Lock()
		seen[s.Name] = s
		mu.Unlock()
		return &stub{name: s.Name}, nil
	}, newDiscardLog())
	require.NoError(t, reg.Load(t.Context()))

	wantBase := map[string]string{
		"OA": "https://api.openai.com/v1",
		"AN": "https://api.anthropic.com/v1/",
		"GE": "https://generativelanguage.googleapis.com/v1beta/openai/",
		"OC": "http://vllm.local:8000/v1",
	}
	for name, want := range wantBase {
		s, ok := seen[name]
		require.True(t, ok, name)
		require.Equal(t, want, provider.BaseURLFor(s.Type, s.BaseURL), name)
	}
	require.Equal(t, "sk-oa", seen["OA"].Key, "khoá được giải mã trước khi dựng client")
	require.Contains(t, seen, "FK")
	require.NotContains(t, seen, "OFF", "nhà tắt bị bỏ qua")
	require.NotContains(t, seen, "BAD", "nhà unreadable bị bỏ qua")

	rt, err := reg.Route(llm.TaskChat)
	require.NoError(t, err)
	names := []string{}
	for _, tg := range rt.Targets {
		names = append(names, tg.ProviderName+"/"+tg.Model)
	}
	require.Equal(t, []string{"OA/gpt-x", "AN/claude-x", "GE/gem-x", "OC/llama"}, names)
	require.True(t, rt.Targets[0].PriceIn.Equal(decimal.RequireFromString("4000")))
	require.Equal(t, 60, rt.Targets[0].RPM)

	rt2, err := reg.Route(llm.TaskClassify)
	require.NoError(t, err)
	require.Len(t, rt2.Targets, 1, "chỉ còn FK: OFF tắt, BAD không đọc được")
	require.Equal(t, "FK", rt2.Targets[0].ProviderName)
	require.False(t, reg.EnvActive())
	_, err = reg.Route(llm.TaskGrading)
	require.ErrorIs(t, err, llm.ErrNotConfigured)
}

func TestReloadAtomic(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	pa := r.add(t, "fake", "A", "", "", cm("ma"))
	pb := r.add(t, "fake", "B", "", "", cm("mb"))
	_, err := r.svc.SetRoute(r.ctx, "CHAT", []uuid.UUID{pa.Models[0].ID}, nil, 0)
	require.NoError(t, err)

	entered, release := make(chan struct{}), make(chan struct{})
	a := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		close(entered)
		<-release
		return provider.Result{Text: "từ A", TokensIn: 1, TokensOut: 1}, nil
	}}
	b := &stub{name: "B"}
	stubs := map[string]*stub{"A": a, "B": b}
	reg := llm.NewRegistry(r.res, llm.EnvConfig{}, func(s llm.ProviderSpec) (provider.Provider, error) { return stubs[s.Name], nil }, newDiscardLog())
	require.NoError(t, reg.Load(t.Context()))
	g := llm.New(llm.Options{NoMask: true, Registry: reg, Log: newDiscardLog()})

	old := make(chan string, 1)
	go func() {
		resp, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("cũ")})
		if err != nil {
			old <- "lỗi: " + err.Error()
			return
		}
		old <- resp.Text
	}()
	<-entered // lời gọi cũ đang chạy bên trong A

	_, err = r.svc.SetRoute(r.ctx, "CHAT", []uuid.UUID{pb.Models[0].ID}, nil, 1)
	require.NoError(t, err)
	require.NoError(t, reg.Reload(t.Context()))
	resp, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("mới")})
	require.NoError(t, err)
	require.Equal(t, "ok từ B", resp.Text, "lời gọi MỚI dùng cấu hình mới")

	close(release)
	select {
	case got := <-old:
		require.Equal(t, "từ A", got, "lời gọi ĐANG chạy hoàn tất với cấu hình cũ")
	case <-time.After(30 * time.Second):
		t.Fatal("lời gọi cũ không hoàn tất")
	}
}

func TestReloadKeepsOldOnError(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	pa := r.add(t, "fake", "A", "", "", cm("ma"))
	_, err := r.svc.SetRoute(r.ctx, "CHAT", []uuid.UUID{pa.Models[0].ID}, nil, 0)
	require.NoError(t, err)
	var logBuf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logBuf, nil))
	reg := llm.NewRegistry(r.res, llm.EnvConfig{}, func(s llm.ProviderSpec) (provider.Provider, error) { return &stub{name: s.Name}, nil }, log)
	require.NoError(t, reg.Load(t.Context()))

	_, err = r.pool.Exec(t.Context(), `update llm_task_routes set params = '"hỏng"'::jsonb`)
	require.NoError(t, err)
	require.Error(t, reg.Reload(t.Context()))
	rt, err := reg.Route(llm.TaskChat)
	require.NoError(t, err, "giữ cấu hình cũ")
	require.Equal(t, "A", rt.Targets[0].ProviderName)
	require.Contains(t, logBuf.String(), `"level":"ERROR"`)
}

// TestKeyNeverLeaksViaErrors — US-P1-02 AC16 ở tầng Gateway: máy chủ phản chiếu Authorization; lỗi, log, audit không chứa khoá.
func TestKeyNeverLeaksViaErrors(t *testing.T) {
	t.Parallel()
	const canary = "sk-CANARY-9a8b7c6d5e4f3a2b1c0d"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"bad key; you sent %s and %s"}}`, r.Header.Get("Authorization"), canary)
	}))
	defer srv.Close()
	buf, log := jsonBuf()
	reg := llm.NewRegistry(nil, llm.EnvConfig{}, nil, log)
	reg.SetStaticForTest(llm.TaskClassify, llm.Route{Targets: []llm.Target{{ProviderID: "x", ProviderName: "X", Type: "openai_compatible", Model: "m",
		P: provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: srv.URL, APIKey: canary})}}})
	cap := &capture{}
	aud := llm.NewAuditor(context.Background(), cap.write, log)
	defer aud.Close(context.Background())
	g := llm.New(llm.Options{NoMask: true, Registry: reg, Auditor: aud, Log: log})
	_, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")})
	require.ErrorIs(t, err, llm.ErrAllProvidersFailed)
	require.NotContains(t, err.Error(), canary)
	rows := cap.all(t, g)
	require.Len(t, rows, 1)
	require.Equal(t, "AUTH", rows[0].ErrorKind)
	all := buf.String() + fmt.Sprintf("%+v", rows)
	require.NotContains(t, all, canary)
	require.False(t, strings.Contains(all, "Bearer "+canary))
	require.NotRegexp(t, `sk-[A-Za-z0-9_-]{8,}`, all)
	require.NotRegexp(t, `(?i)bearer\s+[A-Za-z0-9._-]{8,}`, all)
}

// DEMO_SCRIPT tầng 2: tắt nhà cung cấp thật ở /settings/llm → mọi tác vụ rơi về nhà cung cấp giả của env (LLM_PROVIDER=fake + LLM_DEMO_FILE); bật lại → trở về nhà cung cấp thật.
func TestRegistryDisableProviderFallsBackToEnvFake(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	real := r.add(t, "openai_compatible", "9router", "", "http://127.0.0.1:20128/v1", cm("ag/gemini"))
	_, err := r.svc.SetRoute(r.ctx, "CHAT", []uuid.UUID{real.Models[0].ID}, nil, 0)
	require.NoError(t, err)
	reg := llm.NewRegistry(r.res, llm.EnvConfig{Provider: "fake"}, func(s llm.ProviderSpec) (provider.Provider, error) { return &stub{name: s.Name}, nil }, newDiscardLog())
	require.NoError(t, reg.Load(t.Context()))
	first := func(task llm.Task) string {
		rt, err := reg.Route(task)
		require.NoError(t, err)
		return rt.Targets[0].ProviderName
	}
	require.Equal(t, "9router", first(llm.TaskChat), "DB thắng env")
	require.Equal(t, "env:fake", first(llm.TaskGrading), "tác vụ chưa có tuyến dùng env")

	off := false
	real, err = r.svc.UpdateProvider(r.ctx, real.ID, llmconfig.ProviderInput{Type: "openai_compatible", Name: "9router", BaseURL: real.BaseURL, Enabled: &off, Version: real.Version})
	require.NoError(t, err)
	require.NoError(t, reg.Reload(t.Context()))
	require.Equal(t, "env:fake", first(llm.TaskChat), "tắt nhà cung cấp thật → nhà cung cấp giả")
	require.True(t, reg.EnvActive())

	on := true
	_, err = r.svc.UpdateProvider(r.ctx, real.ID, llmconfig.ProviderInput{Type: "openai_compatible", Name: "9router", BaseURL: real.BaseURL, Enabled: &on, Version: real.Version})
	require.NoError(t, err)
	require.NoError(t, reg.Reload(t.Context()))
	require.Equal(t, "9router", first(llm.TaskChat), "bật lại → trở về nhà cung cấp thật")
}

// DEF-R105S-03 (TC-D02-10, TC-D01-25; SRS FEAT-decision-model 4.4): provider «Quyết định» có trong DB nhưng TẮT → hiệu lực = tắt cho MỌI cổng (chat, chèn lệnh, Threads),
// kể cả khi env là fake (dev / test / ep-qc). Bật lại thì dùng bản ghi DB.
func TestRegistryDisabledDecisionProviderTurnsEveryGateOff(t *testing.T) {
	t.Parallel()
	r := newCfgRig(t)
	factory := func(s llm.ProviderSpec) (provider.Provider, error) { return &stub{name: s.Name}, nil }
	newGW := func(reg *llm.Registry) (*llm.Gateway, *capture, context.Context) {
		cap := &capture{}
		aud := llm.NewAuditor(context.Background(), cap.write, newDiscardLog())
		t.Cleanup(func() { aud.Close(context.Background()) })
		snap := llm.NewDecisionSnapshot(nil, nil)
		gates, courses := map[llm.DecideGate]llm.GateCfg{}, map[llm.DecideGate]map[uuid.UUID]struct{}{}
		for _, g := range []llm.DecideGate{llm.GateThreads, llm.GateInjection, llm.GateRouting} {
			gates[g] = llm.GateCfg{Enabled: true, Threshold: 0.5, Timeout: time.Second}
			courses[g] = map[uuid.UUID]struct{}{decCourse: {}}
		}
		snap.Set(llm.DecisionConfig{Gates: gates, Courses: courses})
		ctx := llm.WithIdentity(context.Background(), llm.Identity{CourseID: &decCourse})
		return llm.New(llm.Options{NoMask: true, Registry: reg, Auditor: aud, Log: newDiscardLog(), Decision: snap}), cap, ctx
	}
	gates := []llm.DecideGate{llm.GateThreads, llm.GateInjection, llm.GateRouting}

	// đối chứng: migration 00020 gieo provider «Quyết định» (bật, decision-fake) → mọi cổng hiệu lực
	reg := llm.NewRegistry(r.res, llm.EnvConfig{Provider: "fake"}, factory, newDiscardLog())
	require.NoError(t, reg.Load(t.Context()))
	g, _, ctx := newGW(reg)
	for _, gate := range gates {
		require.True(t, g.GateEffective(ctx, gate), gate)
	}

	// provider decision có trong DB nhưng TẮT (PUT /admin/llm/providers/{id} enabled=false)
	provs, err := r.svc.ListProviders(r.ctx)
	require.NoError(t, err)
	var dec llmconfig.Provider
	for _, p := range provs {
		if p.Type == "decision" {
			dec = p
		}
	}
	require.NotEqual(t, uuid.Nil, dec.ID)
	off := false
	dm := llmconfig.ModelInput{Model: "decision-fake", Kind: "decision", PriceIn: decimal.RequireFromString("2600")}
	dec, err = r.svc.UpdateProvider(r.ctx, dec.ID, llmconfig.ProviderInput{Type: "decision", Name: dec.Name, Enabled: &off, Version: dec.Version, Models: []llmconfig.ModelInput{dm}})
	require.NoError(t, err)
	require.NoError(t, reg.Reload(t.Context()))
	g, c, ctx := newGW(reg)
	_, ok := reg.DecideTarget()
	require.False(t, ok, "provider decision tắt: không còn mô hình decision, kể cả khi env là fake")
	for _, gate := range gates {
		require.False(t, g.GateEffective(ctx, gate), gate)
		ran := false
		_, path := g.Cascade(ctx, gate, llm.DecideRequest{Gate: gate, Input: "câu hỏi mẫu", Questions: noulQ}, llm.AddOnly{QID: "q"},
			func(context.Context) (llm.Verdict, error) { ran = true; return llm.Verdict{}, nil }, llm.Verdict{})
		require.Equal(t, llm.PathNotUsed, path, gate)
		require.False(t, ran, "%s: không chạy cả đường dự phòng của cổng — người gọi chạy đường cũ", gate)
	}
	for _, row := range c.all(t, g) {
		require.NotEqual(t, "DECIDE", row.Task, "provider tắt: 0 dòng llm_audit DECIDE")
	}

	// bật lại → dùng bản ghi DB
	on := true
	_, err = r.svc.UpdateProvider(r.ctx, dec.ID, llmconfig.ProviderInput{Type: "decision", Name: "Quyết định", Enabled: &on, Version: dec.Version, Models: []llmconfig.ModelInput{dm}})
	require.NoError(t, err)
	require.NoError(t, reg.Reload(t.Context()))
	tg, ok := reg.DecideTarget()
	require.True(t, ok)
	require.Equal(t, dec.ID.String(), tg.ProviderID)
}
