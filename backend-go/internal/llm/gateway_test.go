package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

func TestClientSurface(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf((*llm.Client)(nil)).Elem()
	want := map[string]reflect.Type{
		"Chat":       reflect.TypeOf(func(context.Context, llm.Request) (llm.Response, error) { return llm.Response{}, nil }),
		"Stream":     reflect.TypeOf(func(context.Context, llm.Request) (<-chan llm.Chunk, error) { return nil, nil }),
		"Structured": reflect.TypeOf(func(context.Context, llm.Request, json.RawMessage) (json.RawMessage, error) { return nil, nil }),
		"Embed":      reflect.TypeOf(func(context.Context, llm.EmbedRequest) ([][]float32, error) { return nil, nil }),
	}
	if typ.NumMethod() != len(want) {
		t.Fatalf("Client có %d hàm, muốn %d", typ.NumMethod(), len(want))
	}
	for i := range typ.NumMethod() {
		m := typ.Method(i)
		if m.Type != want[m.Name] {
			t.Errorf("%s: %s, muốn %s", m.Name, m.Type, want[m.Name])
		}
	}
	for _, f := range []string{"UserID", "CourseID", "StudentCode"} {
		if _, ok := reflect.TypeOf(llm.Request{}).FieldByName(f); ok {
			t.Errorf("Request không được có trường %s", f)
		}
	}
}

func TestChatBasicAndCost(t *testing.T) {
	t.Parallel()
	g, cap := newGateway(t, &stub{name: "A"})
	r, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("xin chào")})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "ok từ A" || r.FallbackIndex != 0 || r.Provider != "A" || r.Model != "m-A" {
		t.Fatalf("response = %+v", r)
	}
	// 10 vào × 1000 + 5 ra × 2000 = 20000 đ / 1e6
	if r.CostEst.String() != "0.02" {
		t.Errorf("cost = %s, muốn 0.02", r.CostEst)
	}
	rows := cap.all(t, g)
	if len(rows) != 1 || rows[0].Status != "ok" || rows[0].Lane != "INTERACTIVE" || rows[0].TokensIn != 10 {
		t.Fatalf("audit = %+v", rows)
	}
}

func TestRetryBackoff(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		lane      llm.Lane
		failKind  provider.Kind
		failTimes int
		wantCalls int
		wantSleep []time.Duration // Rand=1 → đúng trần: 500 ms rồi 1 s
		wantErr   error
	}{
		{"INTERACTIVE: chỉ 1 lần thử lại", llm.LaneInteractive, provider.KindServer, 5, 2, []time.Duration{500 * time.Millisecond}, llm.ErrAllProvidersFailed},
		{"NEAR_REALTIME: 2 lần, trễ nhân đôi", llm.LaneNearRealtime, provider.KindServer, 5, 3, []time.Duration{500 * time.Millisecond, time.Second}, llm.ErrAllProvidersFailed},
		{"BATCH: 2 lần", llm.LaneBatch, provider.KindNetwork, 5, 3, []time.Duration{500 * time.Millisecond, time.Second}, llm.ErrAllProvidersFailed},
		{"RATE_LIMIT được thử lại", llm.LaneNearRealtime, provider.KindRateLimit, 1, 2, []time.Duration{500 * time.Millisecond}, nil},
		{"TIMEOUT được thử lại", llm.LaneNearRealtime, provider.KindTimeout, 1, 2, []time.Duration{500 * time.Millisecond}, nil},
		{"AUTH không thử lại", llm.LaneNearRealtime, provider.KindAuth, 5, 1, nil, llm.ErrAllProvidersFailed},
		{"MODEL_NOT_FOUND không thử lại", llm.LaneNearRealtime, provider.KindModelNotFound, 5, 1, nil, llm.ErrAllProvidersFailed},
		{"BAD_REQUEST không thử lại", llm.LaneNearRealtime, provider.KindBadRequest, 5, 1, nil, llm.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &stub{name: "A", chat: func(n int, _ provider.ChatOpts) (provider.Result, error) {
				if n <= tc.failTimes {
					return provider.Result{}, perr(tc.failKind, 500)
				}
				return provider.Result{Text: "ok", TokensIn: 1, TokensOut: 1}, nil
			}}
			g, cap := newGateway(t, p)
			lane := tc.lane
			task := llm.TaskUtility
			if tc.lane == llm.LaneInteractive {
				task = llm.TaskChat
			}
			_, err := g.Chat(t.Context(), llm.Request{Task: task, Lane: &lane, Messages: userMsg("x")})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, muốn %v", err, tc.wantErr)
			}
			if got := int(p.calls.Load()); got != tc.wantCalls {
				t.Errorf("số lần gọi = %d, muốn %d", got, tc.wantCalls)
			}
			if len(cap.sleeps) != len(tc.wantSleep) {
				t.Fatalf("sleeps = %v, muốn %v", cap.sleeps, tc.wantSleep)
			}
			for i, d := range tc.wantSleep {
				if cap.sleeps[i] != d {
					t.Errorf("sleep[%d] = %v, muốn %v", i, cap.sleeps[i], d)
				}
			}
		})
	}
}

func TestRetryAfterHeader(t *testing.T) {
	t.Parallel()
	mk := func(ra time.Duration) (*stub, *llm.Gateway, *capture) {
		p := &stub{name: "A", chat: func(n int, _ provider.ChatOpts) (provider.Result, error) {
			if n == 1 {
				return provider.Result{}, &provider.Error{Kind: provider.KindRateLimit, Status: 429, RetryAfter: ra}
			}
			return provider.Result{Text: "ok"}, nil
		}}
		g, cap := newGateway(t, p)
		return p, g, cap
	}
	lane := llm.LaneBatch
	req := llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: userMsg("x")}

	// Retry-After 3 s (≤ 5 s) lớn hơn jitter → chờ đúng 3 s
	p, g, cap := mk(3 * time.Second)
	if _, err := g.Chat(t.Context(), req); err != nil || p.calls.Load() != 2 || len(cap.sleeps) != 1 || cap.sleeps[0] != 3*time.Second {
		t.Fatalf("Retry-After 3 s: err=%v calls=%d sleeps=%v", err, p.calls.Load(), cap.sleeps)
	}
	// Retry-After 30 s (> 5 s) → không chờ, chuyển nhà kế (ở đây hết chuỗi)
	p, g, cap = mk(30 * time.Second)
	if _, err := g.Chat(t.Context(), req); !errors.Is(err, llm.ErrAllProvidersFailed) || p.calls.Load() != 1 || len(cap.sleeps) != 0 {
		t.Fatalf("Retry-After 30 s: err=%v calls=%d sleeps=%v", err, p.calls.Load(), cap.sleeps)
	}
}

func TestRetryRespectsDeadline(t *testing.T) {
	t.Parallel()
	p := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindServer, 503)
	}}
	g, cap := newGateway(t, p)
	ctx, cancel := context.WithTimeout(t.Context(), 800*time.Millisecond) // còn < 1 s
	defer cancel()
	lane := llm.LaneBatch
	_, err := g.Chat(ctx, llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: userMsg("x")})
	if !errors.Is(err, llm.ErrAllProvidersFailed) || p.calls.Load() != 1 || len(cap.sleeps) != 0 {
		t.Fatalf("err=%v calls=%d sleeps=%v: không được thử lại khi còn < 1 s", err, p.calls.Load(), cap.sleeps)
	}
}

func TestFallbackOrder(t *testing.T) {
	t.Parallel()
	a := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindAuth, 401)
	}}
	b := &stub{name: "B", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindModelNotFound, 404)
	}}
	c := &stub{name: "C"}
	g, cap := newGateway(t, a, b, c)
	r, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")})
	if err != nil || r.FallbackIndex != 2 || r.Provider != "C" {
		t.Fatalf("r=%+v err=%v, muốn nhà C (index 2)", r, err)
	}
	if a.calls.Load() != 1 || b.calls.Load() != 1 || c.calls.Load() != 1 {
		t.Errorf("gọi A/B/C = %d/%d/%d, muốn 1/1/1", a.calls.Load(), b.calls.Load(), c.calls.Load())
	}
	if rows := cap.all(t, g); len(rows) != 1 || rows[0].FallbackIndex != 2 || rows[0].Provider != "C" {
		t.Errorf("audit = %+v", rows)
	}
}

func TestNoFallbackOnBadRequest(t *testing.T) {
	t.Parallel()
	a := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindBadRequest, 400)
	}}
	b := &stub{name: "B"}
	g, cap := newGateway(t, a, b)
	_, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")})
	if !errors.Is(err, llm.ErrBadRequest) || b.calls.Load() != 0 {
		t.Fatalf("err=%v B=%d: BAD_REQUEST không được chuyển tiếp", err, b.calls.Load())
	}
	if rows := cap.all(t, g); len(rows) != 1 || rows[0].ErrorKind != "BAD_REQUEST" || rows[0].Status != "error" {
		t.Errorf("audit = %+v", rows)
	}
}

func TestAllFailed(t *testing.T) {
	t.Parallel()
	bad := func(n string) *stub {
		return &stub{name: n, chat: func(int, provider.ChatOpts) (provider.Result, error) {
			return provider.Result{}, perr(provider.KindServer, 500)
		}}
	}
	g, cap := newGateway(t, bad("A"), bad("B"))
	lane := llm.LaneBatch
	_, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: userMsg("x")})
	if !errors.Is(err, llm.ErrAllProvidersFailed) {
		t.Fatalf("err = %v", err)
	}
	rows := cap.all(t, g)
	if len(rows) != 1 || rows[0].Attempts != 6 || rows[0].ErrorKind != "SERVER" {
		t.Errorf("audit = %+v (muốn 1 dòng, 6 lần thử = 3×2 nhà)", rows)
	}
}

func TestLaneRules(t *testing.T) {
	t.Parallel()
	ptr := func(l llm.Lane) *llm.Lane { return &l }
	cases := []struct {
		task llm.Task
		lane *llm.Lane
		want llm.Lane
		err  error
	}{
		{llm.TaskChat, nil, llm.LaneInteractive, nil},
		{llm.TaskChat, ptr(llm.LaneNearRealtime), llm.LaneNearRealtime, nil},
		{llm.TaskClassify, nil, llm.LaneNearRealtime, nil},
		{llm.TaskClassify, ptr(llm.LaneBatch), llm.LaneBatch, nil},
		{llm.TaskClassify, ptr(llm.LaneInteractive), 0, llm.ErrBadLane},
		{llm.TaskUtility, ptr(llm.LaneInteractive), 0, llm.ErrBadLane},
		{llm.TaskGrading, nil, llm.LaneBatch, nil},
		{llm.TaskGrading, ptr(llm.LaneInteractive), 0, llm.ErrBadLane},
		{llm.TaskQuestionGen, ptr(llm.LaneNearRealtime), 0, llm.ErrBadLane},
		{llm.TaskInsight, ptr(llm.LaneInteractive), 0, llm.ErrBadLane},
		{llm.TaskEmbedding, nil, llm.LaneBatch, nil},
		{llm.TaskEmbedding, ptr(llm.LaneInteractive), llm.LaneInteractive, nil},
		{llm.TaskChat, ptr(llm.Lane(7)), 0, llm.ErrBadLane},
	}
	for _, tc := range cases {
		got, err := llm.ResolveLane(tc.task, tc.lane)
		if !errors.Is(err, tc.err) || (err == nil && got != tc.want) {
			t.Errorf("%s/%v: (%v, %v), muốn (%v, %v)", tc.task, tc.lane, got, err, tc.want, tc.err)
		}
	}
	// Gateway từ chối làn sai và vẫn ghi audit
	g, cap := newGateway(t, &stub{name: "A"})
	_, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskGrading, Lane: ptr(llm.LaneInteractive), Messages: userMsg("x")})
	if !errors.Is(err, llm.ErrBadLane) || len(cap.all(t, g)) != 1 {
		t.Errorf("err=%v", err)
	}
}

func TestNotConfigured(t *testing.T) {
	t.Parallel()
	reg := llm.NewRegistry(nil, llm.EnvConfig{}, nil, nil)
	if err := reg.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	g := llm.New(llm.Options{Registry: reg})
	for _, call := range []func() error{
		func() error {
			_, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("x")})
			return err
		},
		func() error {
			_, err := g.Stream(t.Context(), llm.Request{Task: llm.TaskChat, Messages: userMsg("x")})
			return err
		},
		func() error { _, err := g.Embed(t.Context(), llm.EmbedRequest{Inputs: []string{"x"}}); return err },
		func() error {
			_, err := g.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")}, json.RawMessage(`{"type":"object"}`))
			return err
		},
	} {
		if err := call(); !errors.Is(err, llm.ErrNotConfigured) {
			t.Errorf("err = %v, muốn ErrNotConfigured", err)
		}
	}
}

func TestEnvFallback(t *testing.T) {
	t.Parallel()
	log := newDiscardLog()
	t.Run("LLM_PROVIDER=fake", func(t *testing.T) {
		reg := llm.NewRegistry(nil, llm.EnvConfig{Provider: "fake"}, nil, log)
		if err := reg.Load(t.Context()); err != nil {
			t.Fatal(err)
		}
		g := llm.New(llm.Options{Registry: reg, Log: log})
		r, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskUtility, Messages: userMsg("tóm tắt giúp")})
		if err != nil || r.Model != llm.FakeChatModel || !strings.Contains(r.Text, "tóm tắt giúp") {
			t.Fatalf("r=%+v err=%v", r, err)
		}
		vecs, err := g.Embed(t.Context(), llm.EmbedRequest{Inputs: []string{"a", "b"}})
		if err != nil || len(vecs) != 2 || len(vecs[0]) != llm.EmbedDims {
			t.Fatalf("embed: %d vectơ, err=%v", len(vecs), err)
		}
		if !reg.EnvActive() {
			t.Error("EnvActive phải true")
		}
	})
	t.Run("khoá OpenAI → chuỗi chat + nhúng", func(t *testing.T) {
		reg := llm.NewRegistry(nil, llm.EnvConfig{OpenAIKey: "k1", AnthropicKey: "k2", GeminiKey: "k3"}, nil, log)
		if err := reg.Load(t.Context()); err != nil {
			t.Fatal(err)
		}
		rt, err := reg.Route(llm.TaskChat)
		if err != nil || len(rt.Targets) != 3 {
			t.Fatalf("chuỗi chat = %d, err=%v", len(rt.Targets), err)
		}
		want := []string{llm.DefaultOpenAIChat, llm.DefaultAnthropicChat, llm.DefaultGeminiChat}
		for i, tg := range rt.Targets {
			if tg.Model != want[i] {
				t.Errorf("chuỗi[%d] = %s, muốn %s", i, tg.Model, want[i])
			}
		}
		emb, err := reg.Route(llm.TaskEmbedding)
		if err != nil || len(emb.Targets) != 1 || emb.Targets[0].Model != llm.DefaultOpenAIEmbedding {
			t.Fatalf("embedding = %+v err=%v", emb, err)
		}
	})
	t.Run("chỉ khoá Anthropic → không có nhúng", func(t *testing.T) {
		reg := llm.NewRegistry(nil, llm.EnvConfig{AnthropicKey: "k2"}, nil, log)
		if err := reg.Load(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := reg.Route(llm.TaskEmbedding); !errors.Is(err, llm.ErrNotConfigured) {
			t.Errorf("err = %v", err)
		}
		if _, err := reg.Route(llm.TaskChat); err != nil {
			t.Errorf("chat: %v", err)
		}
	})
}

func TestTraceIDPropagation(t *testing.T) {
	t.Parallel()
	buf, log := jsonBuf()
	a := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{}, perr(provider.KindAuth, 401)
	}}
	g, cap := newGatewayLog(t, log, a, &stub{name: "B"})
	const want = "0af7651916cd43dd8448eb211c80319c"
	tid, _ := trace.TraceIDFromHex(want)
	sid, _ := trace.SpanIDFromHex("b7ad6b7169203331")
	ctx := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled}))
	if _, err := g.Chat(ctx, llm.Request{Task: llm.TaskClassify, Messages: userMsg("x")}); err != nil {
		t.Fatal(err)
	}
	rows := cap.all(t, g)
	if len(rows) != 1 || rows[0].TraceID != want {
		t.Fatalf("audit = %+v", rows)
	}
	// dòng warn chuyển dự phòng cùng trace_id, không chứa nội dung / khoá
	var warn map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && m["level"] == "WARN" {
			warn = m
		}
	}
	if warn == nil || warn["trace_id"] != want || warn["task"] != "CLASSIFY" || warn["from"] != "A" || warn["to"] != "B" || warn["error_kind"] != "AUTH" {
		t.Fatalf("log fallback = %v", warn)
	}
}
