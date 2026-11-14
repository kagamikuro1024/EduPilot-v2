package llm_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

// dstub là provider decision kịch bản hoá.
type dstub struct {
	calls atomic.Int64
	fn    func(ctx context.Context, o provider.DecideOpts) (provider.DecideResult, error)
	mu    sync.Mutex
	seen  []provider.DecideOpts
}

func (*dstub) Chat(context.Context, provider.ChatOpts) (provider.Result, error) {
	return provider.Result{}, errors.New("không sinh chữ")
}
func (*dstub) Stream(context.Context, provider.ChatOpts) (<-chan provider.Delta, error) {
	return nil, errors.New("không sinh chữ")
}
func (*dstub) Embed(context.Context, provider.EmbedOpts) ([][]float32, int, error) {
	return nil, 0, errors.New("không nhúng")
}
func (d *dstub) Decide(ctx context.Context, o provider.DecideOpts) (provider.DecideResult, error) {
	d.calls.Add(1)
	d.mu.Lock()
	d.seen = append(d.seen, o)
	d.mu.Unlock()
	if d.fn != nil {
		return d.fn(ctx, o)
	}
	return noulRes(0.9), nil
}

func noulRes(v float64) provider.DecideResult {
	return provider.DecideResult{Answers: map[string]provider.DecideAnswer{"q": {Type: provider.QuestionNoul, Noul: v, Confidence: max(v, 1-v)}}, TokensIn: 100}
}

func choiceRes(label string, conf float64) provider.DecideResult {
	return provider.DecideResult{Answers: map[string]provider.DecideAnswer{"q": {Type: provider.QuestionChoice, Choice: label, Confidence: conf,
		Probabilities: map[string]float64{label: conf}}}, TokensIn: 100}
}

var (
	decCourse = uuid.MustParse("00000000-0000-0000-0000-0000000000d1")
	noulQ     = []llm.Question{llm.NoulQ{ID: "q", Text: "Có không?"}}
	choiceQ   = []llm.Question{llm.ChoiceQ{ID: "q", Text: "Chọn", Options: []llm.Option{{Key: "A", Text: "a"}, {Key: "B", Text: "b"}}}}
)

type decOpt func(*llm.Options)

// decideGW dựng Gateway với provider decision stub và cổng `gate` bật cho decCourse.
func decideGW(t *testing.T, d *dstub, model string, opts ...decOpt) (*llm.Gateway, *capture, context.Context) {
	t.Helper()
	cap := &capture{}
	aud := llm.NewAuditor(context.Background(), cap.write, newDiscardLog())
	t.Cleanup(func() { aud.Close(context.Background()) })
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	if d != nil {
		reg.SetDecide(&llm.Target{ProviderID: "dec", ProviderName: "Quyết định", Type: "decision", Model: model,
			PriceIn: decimal.RequireFromString("2600"), RPM: 150, TPM: 100000, P: d})
	}
	snap := llm.NewDecisionSnapshot(nil, nil)
	snap.Set(llm.DecisionConfig{
		Gates: map[llm.DecideGate]llm.GateCfg{
			llm.GateThreads:   {Enabled: true, Threshold: 0.05, Timeout: 800 * time.Millisecond},
			llm.GateInjection: {Enabled: true, Threshold: 0.5, Timeout: 3 * time.Second},
			llm.GateRouting:   {Enabled: true, Threshold: 0.95, Timeout: 800 * time.Millisecond},
		},
		Courses: map[llm.DecideGate]map[uuid.UUID]struct{}{
			llm.GateThreads: {decCourse: {}}, llm.GateInjection: {decCourse: {}}, llm.GateRouting: {decCourse: {}},
		},
	})
	o := llm.Options{NoMask: true, Registry: reg, Auditor: aud, Log: newDiscardLog(), Decision: snap,
		Sleep: func(ctx context.Context, _ time.Duration) error { return ctx.Err() }, Rand: func() float64 { return 1 }}
	for _, f := range opts {
		f(&o)
	}
	ctx := llm.WithIdentity(context.Background(), llm.Identity{CourseID: &decCourse})
	return llm.New(o), cap, ctx
}

func req(gate llm.DecideGate, qs []llm.Question) llm.DecideRequest {
	return llm.DecideRequest{Gate: gate, Input: "câu hỏi mẫu", Questions: qs}
}

func decRows(t *testing.T, g *llm.Gateway, c *capture) []llm.AuditRow {
	t.Helper()
	var out []llm.AuditRow
	for _, r := range c.all(t, g) {
		if r.Task == "DECIDE" {
			out = append(out, r)
		}
	}
	return out
}

// TestDecideAudit: đúng MỘT dòng llm_audit; task DECIDE, làn INTERACTIVE, token ra 0, chi phí = token vào × giá vào, nhãn cổng; không nội dung.
func TestDecideAudit(t *testing.T) {
	t.Parallel()
	g, c, ctx := decideGW(t, &dstub{}, llm.FakeDecisionModel)
	resp, err := g.Decide(ctx, req(llm.GateThreads, noulQ))
	if err != nil {
		t.Fatal(err)
	}
	n, ok := resp.Noul("q")
	if !ok || n.Noul != 0.9 || resp.Usage.TokensIn != 100 || resp.Usage.TokensOut != 0 {
		t.Fatalf("resp = %+v", resp)
	}
	rows := decRows(t, g, c)
	if len(rows) != 1 {
		t.Fatalf("%d dòng audit, muốn 1", len(rows))
	}
	r := rows[0]
	if r.Lane != "INTERACTIVE" || r.Status != "ok" || r.TokensIn != 100 || r.TokensOut != 0 || r.CascadeGate != "THREADS_PRECHECK" || r.CascadePath != "" ||
		r.CourseID == nil || !r.CostEst.Equal(decimal.RequireFromString("0.26")) {
		t.Fatalf("row = %+v", r)
	}
}

// TestDecideMasked: Masker che tên + MSSV trước khi tới provider decision (D47(4)).
func TestDecideMasked(t *testing.T) {
	t.Parallel()
	d := &dstub{}
	g, _, _ := decideGW(t, d, llm.FakeDecisionModel, func(o *llm.Options) { o.NoMask = false; o.Masker = realMasker(newDiscardLog()) })
	ctx := llm.WithIdentity(context.Background(), llm.Identity{CourseID: &maskCourse})
	r := req(llm.GateThreads, noulQ)
	r.Input = "Nguyễn Văn An (ZZ99887766) hỏi điểm"
	if _, err := g.Decide(ctx, r); err != nil {
		t.Fatal(err)
	}
	got := d.seen[0].Input
	if strings.Contains(got, "Nguyễn Văn An") || strings.Contains(got, "ZZ99887766") || !strings.Contains(got, "[") {
		t.Fatalf("đầu vào tới provider chưa che: %q", got)
	}
}

// TestDecideErrorMap: lỗi nhà cung cấp → error_kind trên dòng audit; KHÔNG thử lại (đúng một lời gọi); SAFETY_BLOCK không tính vào mạch ngắt.
func TestDecideErrorMap(t *testing.T) {
	t.Parallel()
	for _, k := range []provider.Kind{provider.KindServer, provider.KindRateLimit, provider.KindAuth, provider.KindBlocked, provider.KindBadResponse, provider.KindTimeout} {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			d := &dstub{fn: func(context.Context, provider.DecideOpts) (provider.DecideResult, error) {
				return provider.DecideResult{}, &provider.Error{Kind: k}
			}}
			g, c, ctx := decideGW(t, d, llm.FakeDecisionModel)
			if _, err := g.Decide(ctx, req(llm.GateThreads, noulQ)); err == nil {
				t.Fatal("muốn lỗi")
			}
			rows := decRows(t, g, c)
			if d.calls.Load() != 1 || len(rows) != 1 || rows[0].ErrorKind != string(k) || rows[0].Status == "ok" {
				t.Fatalf("calls=%d rows=%+v", d.calls.Load(), rows)
			}
		})
	}
}

// TestDecideNotConfigured: chưa có provider → lỗi tức thì (< 50 ms), không gọi mạng, đúng một dòng status not_configured.
func TestDecideNotConfigured(t *testing.T) {
	t.Parallel()
	g, c, ctx := decideGW(t, nil, "")
	start := time.Now()
	_, err := g.Decide(ctx, req(llm.GateThreads, noulQ))
	if !errors.Is(err, llm.ErrNotConfigured) || time.Since(start) > 50*time.Millisecond {
		t.Fatalf("err = %v sau %v", err, time.Since(start))
	}
	if rows := decRows(t, g, c); len(rows) != 1 || rows[0].Status != "not_configured" {
		t.Fatalf("rows = %+v", rows)
	}
}

// TestDecideRealGuard: mô hình THẬT bị chặn sau trần ngày: lời gọi thứ cap+1 trả budget_blocked, provider không bị gọi.
func TestDecideRealGuard(t *testing.T) {
	t.Parallel()
	d := &dstub{}
	g, c, ctx := decideGW(t, d, llm.RealDecisionModel, func(o *llm.Options) { o.RealDailyCap = 2 })
	for i := range 3 {
		_, err := g.Decide(ctx, req(llm.GateThreads, noulQ))
		var un *llm.ErrUnavailable
		if (i < 2) != (err == nil) || (i == 2 && !errors.As(err, &un)) {
			t.Fatalf("lần %d: err = %v", i, err)
		}
	}
	rows := decRows(t, g, c)
	if d.calls.Load() != 2 || len(rows) != 3 || rows[2].Status != "budget_blocked" {
		t.Fatalf("calls=%d rows=%+v", d.calls.Load(), rows)
	}
	// mô hình giả không bị đếm
	d2 := &dstub{}
	g2, _, ctx2 := decideGW(t, d2, llm.FakeDecisionModel, func(o *llm.Options) { o.RealDailyCap = 1 })
	for range 3 {
		if _, err := g2.Decide(ctx2, req(llm.GateThreads, noulQ)); err != nil {
			t.Fatal(err)
		}
	}
}

// exhausted: ngân sách lớp cạn.
type exhausted struct{ breakerGate }

func (*exhausted) BudgetState(context.Context, *uuid.UUID) llm.BudgetState {
	return llm.BudgetExhausted
}

// TestDecideBudgetNoCheapest: hết ngân sách → budget_blocked, KHÔNG đổi sang mô hình chat rẻ nhất (Decide không sinh chữ).
func TestDecideBudgetNoCheapest(t *testing.T) {
	t.Parallel()
	d := &dstub{}
	g, c, ctx := decideGW(t, d, llm.FakeDecisionModel, func(o *llm.Options) { o.Gate = &exhausted{} })
	_, err := g.Decide(ctx, req(llm.GateThreads, noulQ))
	var un *llm.ErrUnavailable
	if !errors.As(err, &un) || un.Reason != llm.ReasonBudgetExhausted || d.calls.Load() != 0 {
		t.Fatalf("err = %v, calls = %d", err, d.calls.Load())
	}
	if rows := decRows(t, g, c); len(rows) != 1 || rows[0].Status != "budget_blocked" {
		t.Fatalf("rows = %+v", rows)
	}
}

// TestDecideValidate: câu hỏi / đầu vào sai → ErrBadRequest, không gọi provider.
func TestDecideValidate(t *testing.T) {
	t.Parallel()
	d := &dstub{}
	g, _, ctx := decideGW(t, d, llm.FakeDecisionModel)
	bad := map[string]llm.DecideRequest{
		"rỗng":        {Gate: llm.GateThreads, Input: " ", Questions: noulQ},
		"cổng lạ":     {Gate: "X", Input: "a", Questions: noulQ},
		"không hỏi":   {Gate: llm.GateThreads, Input: "a"},
		"trùng id":    {Gate: llm.GateThreads, Input: "a", Questions: []llm.Question{llm.NoulQ{ID: "q", Text: "a"}, llm.NoulQ{ID: "q", Text: "b"}}},
		"một nhãn":    {Gate: llm.GateRouting, Input: "a", Questions: []llm.Question{llm.ChoiceQ{ID: "q", Text: "a", Options: []llm.Option{{Key: "A"}}}}},
		"score":       {Gate: llm.GateRouting, Input: "a", Questions: []llm.Question{llm.ScoreQ{ID: "q", Text: "a"}}},
		"id có dấu #": {Gate: llm.GateThreads, Input: "a", Questions: []llm.Question{llm.NoulQ{ID: "q#", Text: "a"}}},
	}
	for name, r := range bad {
		if _, err := g.Decide(ctx, r); !errors.Is(err, llm.ErrBadRequest) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if d.calls.Load() != 0 {
		t.Fatal("không được gọi provider")
	}
}

// TestDecideInputCeiling: đầu vào bị cắt theo trần cổng sau khi che (Threads 4.000 ký tự).
func TestDecideInputCeiling(t *testing.T) {
	t.Parallel()
	d := &dstub{}
	g, _, ctx := decideGW(t, d, llm.FakeDecisionModel)
	r := req(llm.GateThreads, noulQ)
	r.Input = strings.Repeat("ạ", 9000)
	if _, err := g.Decide(ctx, r); err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(d.seen[0].Input)); n != 4000 {
		t.Fatalf("%d ký tự, muốn 4000", n)
	}
}

// TestCascadeTable: bảng Policy × kết quả decision × fallback → (Verdict, Path).
func TestCascadeTable(t *testing.T) {
	t.Parallel()
	ok := func(v llm.Verdict) llm.Fallback { return func(context.Context) (llm.Verdict, error) { return v, nil } }
	bad := func(context.Context) (llm.Verdict, error) { return llm.Verdict{}, errors.New("fallback lỗi") }
	fbYes, fbNo := ok(llm.Verdict{Yes: true}), ok(llm.Verdict{Yes: false})
	block := llm.Verdict{Yes: true}
	for _, tc := range []struct {
		name     string
		gate     llm.DecideGate
		qs       []llm.Question
		res      provider.DecideResult
		derr     error
		pol      llm.Policy
		fb       llm.Fallback
		wantPath llm.Path
		wantYes  bool
		wantLbl  string
	}{
		{"AddOnly có ≥0,5 → FAST", llm.GateThreads, noulQ, noulRes(0.93), nil, llm.AddOnly{QID: "q"}, bad, llm.PathFast, true, ""},
		{"AddOnly không ≤ε → FAST", llm.GateThreads, noulQ, noulRes(0.03), nil, llm.AddOnly{QID: "q"}, bad, llm.PathFast, false, ""},
		{"AddOnly 0,06 (>ε) → fallback có", llm.GateThreads, noulQ, noulRes(0.06), nil, llm.AddOnly{QID: "q"}, fbYes, llm.PathFallback, true, ""},
		{"AddOnly 0,25 → fallback không", llm.GateThreads, noulQ, noulRes(0.25), nil, llm.AddOnly{QID: "q"}, fbNo, llm.PathFallback, false, ""},
		{"AddOnly lỗi + fallback lỗi → FAIL_CLOSED", llm.GateThreads, noulQ, provider.DecideResult{}, &provider.Error{Kind: provider.KindServer}, llm.AddOnly{QID: "q"}, bad, llm.PathFailClosed, true, ""},
		{"AddOnly safety → fallback (không phải 'có')", llm.GateThreads, noulQ, provider.DecideResult{}, &provider.Error{Kind: provider.KindBlocked}, llm.AddOnly{QID: "q"}, fbNo, llm.PathFallback, false, ""},
		{"Symmetric 0,96 ≥ τ → FAST", llm.GateRouting, choiceQ, choiceRes("B", 0.96), nil, llm.Symmetric{QID: "q"}, bad, llm.PathFast, false, "B"},
		{"Symmetric 0,95 = τ → FAST", llm.GateRouting, choiceQ, choiceRes("A", 0.95), nil, llm.Symmetric{QID: "q"}, bad, llm.PathFast, false, "A"},
		{"Symmetric 0,94 → fallback luật", llm.GateRouting, choiceQ, choiceRes("B", 0.94), nil, llm.Symmetric{QID: "q"}, ok(llm.Verdict{Label: "RULE"}), llm.PathFallback, false, "RULE"},
		{"Symmetric lỗi → fallback luật", llm.GateRouting, choiceQ, provider.DecideResult{}, &provider.Error{Kind: provider.KindTimeout}, llm.Symmetric{QID: "q"}, ok(llm.Verdict{Label: "RULE"}), llm.PathFallback, false, "RULE"},
		{"OrFlag 0,5 ≥ τ → FAST cờ", llm.GateInjection, noulQ, noulRes(0.5), nil, llm.OrFlag{QID: "q"}, bad, llm.PathFast, true, ""},
		{"OrFlag 0,4 → fallback", llm.GateInjection, noulQ, noulRes(0.4), nil, llm.OrFlag{QID: "q"}, fbNo, llm.PathFallback, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &dstub{fn: func(context.Context, provider.DecideOpts) (provider.DecideResult, error) { return tc.res, tc.derr }}
			g, c, ctx := decideGW(t, d, llm.FakeDecisionModel)
			v, path := g.Cascade(ctx, tc.gate, req(tc.gate, tc.qs), tc.pol, tc.fb, block)
			if path != tc.wantPath || v.Yes != tc.wantYes || v.Label != tc.wantLbl {
				t.Fatalf("path=%s verdict=%+v", path, v)
			}
			rows := decRows(t, g, c)
			if len(rows) != 1 || rows[0].CascadePath != string(tc.wantPath) || rows[0].CascadeGate != string(tc.gate) {
				t.Fatalf("rows = %+v", rows)
			}
		})
	}
}

// TestCascadeFailClosed: decision lỗi VÀ fallback lỗi → giá trị fail-closed của cổng; vẫn đúng MỘT dòng DECIDE có path FAIL_CLOSED.
func TestCascadeFailClosed(t *testing.T) {
	t.Parallel()
	d := &dstub{fn: func(context.Context, provider.DecideOpts) (provider.DecideResult, error) {
		return provider.DecideResult{}, &provider.Error{Kind: provider.KindServer}
	}}
	g, c, ctx := decideGW(t, d, llm.FakeDecisionModel)
	want := llm.Verdict{Yes: true, Label: "chặn"}
	v, path := g.Cascade(ctx, llm.GateThreads, req(llm.GateThreads, noulQ), llm.AddOnly{QID: "q"},
		func(context.Context) (llm.Verdict, error) { return llm.Verdict{}, errors.New("x") }, want)
	rows := decRows(t, g, c)
	if path != llm.PathFailClosed || v != want || len(rows) != 1 || rows[0].CascadePath != "FAIL_CLOSED" || rows[0].ErrorKind != "SERVER" {
		t.Fatalf("path=%s v=%+v rows=%+v", path, v, rows)
	}
}

// TestCascadeNotUsed: cổng tắt (hệ thống hoặc lớp) / chưa có provider → NOT_USED, không gọi provider, không ghi dòng, không chạy fallback.
func TestCascadeNotUsed(t *testing.T) {
	t.Parallel()
	for name, mut := range map[string]func(*llm.DecisionSnapshot){
		"hệ thống tắt": func(s *llm.DecisionSnapshot) {
			s.Set(llm.DecisionConfig{Gates: map[llm.DecideGate]llm.GateCfg{llm.GateThreads: {Enabled: false, Threshold: 0.05}}, Courses: map[llm.DecideGate]map[uuid.UUID]struct{}{llm.GateThreads: {decCourse: {}}}})
		},
		"lớp tắt": func(s *llm.DecisionSnapshot) {
			s.Set(llm.DecisionConfig{Gates: map[llm.DecideGate]llm.GateCfg{llm.GateThreads: {Enabled: true, Threshold: 0.05}}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := &dstub{}
			var snap *llm.DecisionSnapshot
			g, c, ctx := decideGW(t, d, llm.FakeDecisionModel, func(o *llm.Options) { snap = o.Decision })
			mut(snap)
			ran := false
			_, path := g.Cascade(ctx, llm.GateThreads, req(llm.GateThreads, noulQ), llm.AddOnly{QID: "q"},
				func(context.Context) (llm.Verdict, error) { ran = true; return llm.Verdict{}, nil }, llm.Verdict{})
			if path != llm.PathNotUsed || ran || d.calls.Load() != 0 || len(decRows(t, g, c)) != 0 {
				t.Fatalf("path=%s ran=%v calls=%d", path, ran, d.calls.Load())
			}
		})
	}
	g, _, ctx := decideGW(t, nil, "") // không có provider
	if _, path := g.Cascade(ctx, llm.GateThreads, req(llm.GateThreads, noulQ), llm.AddOnly{QID: "q"}, nil, llm.Verdict{}); path != llm.PathNotUsed {
		t.Fatalf("không provider: path = %s", path)
	}
}

// TestCascadeTimeout: decision treo → hết giờ theo cấu hình cổng → fallback; dòng audit TIMEOUT / FALLBACK, trễ ≈ timeout (không cộng fallback).
func TestCascadeTimeout(t *testing.T) {
	t.Parallel()
	d := &dstub{fn: func(ctx context.Context, _ provider.DecideOpts) (provider.DecideResult, error) {
		<-ctx.Done()
		return provider.DecideResult{}, &provider.Error{Kind: provider.KindTimeout}
	}}
	var snap *llm.DecisionSnapshot
	g, c, ctx := decideGW(t, d, llm.FakeDecisionModel, func(o *llm.Options) { snap = o.Decision })
	snap.Set(llm.DecisionConfig{Gates: map[llm.DecideGate]llm.GateCfg{llm.GateRouting: {Enabled: true, Threshold: 0.95, Timeout: 200 * time.Millisecond}},
		Courses: map[llm.DecideGate]map[uuid.UUID]struct{}{llm.GateRouting: {decCourse: {}}}})
	start := time.Now()
	v, path := g.Cascade(ctx, llm.GateRouting, req(llm.GateRouting, choiceQ), llm.Symmetric{QID: "q"},
		func(context.Context) (llm.Verdict, error) {
			time.Sleep(100 * time.Millisecond)
			return llm.Verdict{Label: "RULE"}, nil
		}, llm.Verdict{})
	rows := decRows(t, g, c)
	if path != llm.PathFallback || v.Label != "RULE" || time.Since(start) > time.Second || len(rows) != 1 ||
		rows[0].ErrorKind != "TIMEOUT" || rows[0].CascadePath != "FALLBACK" || rows[0].LatencyMS > 250 {
		t.Fatalf("path=%s rows=%+v sau %v", path, rows, time.Since(start))
	}
}
