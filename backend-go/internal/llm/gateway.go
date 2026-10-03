package llm

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm/cost"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llmconfig"
)

// Hằng số thử lại (SRS 4.2).
const (
	retryBase         = 500 * time.Millisecond
	retryCap          = 4 * time.Second
	retryAfterMax     = 5 * time.Second
	retryMinRemaining = time.Second
	defaultMaxTokens  = 1024
)

// Options dựng Gateway.
type Options struct {
	Registry *Registry
	Gate     Gate     // nil = không giới hạn (nopGate)
	Auditor  *Auditor // nil = không ghi llm_audit
	Log      *slog.Logger

	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error
	Rand  func() float64 // [0,1) cho jitter

	RequestTimeout time.Duration // INTERACTIVE, NEAR_REALTIME khi ctx không có hạn (mặc định 30 s)
	BatchTimeout   time.Duration // BATCH (mặc định 120 s)
}

// Gateway cài Client.
type Gateway struct {
	o       Options
	gate    Gate
	flights flightGroup
}

var _ Client = (*Gateway)(nil)

// New dựng Gateway.
func New(o Options) *Gateway {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Sleep == nil {
		o.Sleep = sleepCtx
	}
	if o.Rand == nil {
		o.Rand = rand.Float64 //nolint:gosec // jitter, không cần CSPRNG
	}
	if o.RequestTimeout <= 0 {
		o.RequestTimeout = 30 * time.Second
	}
	if o.BatchTimeout <= 0 {
		o.BatchTimeout = 120 * time.Second
	}
	if o.Log == nil {
		o.Log = slog.Default()
	}
	g := o.Gate
	if g == nil {
		g = nopGate{}
	}
	return &Gateway{o: o, gate: g}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// call giữ trạng thái của MỘT lời gọi logic (một dòng llm_audit).
type call struct {
	g      *Gateway
	ctx    context.Context
	cancel context.CancelFunc
	task   Task
	lane   Lane
	start  time.Time
	route  Route
	row    AuditRow
	maxRet int // số lần thử lại tối đa cùng nhà (làn + params.retries)
	logged bool
}

// begin giải làn, tuyến, hạn chót, danh tính; mọi lỗi sớm vẫn ghi MỘT dòng audit.
func (g *Gateway) begin(ctx context.Context, task Task, want *Lane, pii int) (*call, error) {
	now := g.o.Now()
	c := &call{g: g, task: task, start: now}
	id := IdentityFrom(ctx)
	tid := TraceID(ctx)
	c.row = AuditRow{Task: string(task), Lane: LaneBatch.String(), Attempts: 0, PIIMaskedCount: pii, UserID: id.UserID, CourseID: id.CourseID, TraceID: tid, At: now}
	lane, err := ResolveLane(task, want)
	if err != nil {
		return c.fail(ctx, "error", "BAD_LANE", err)
	}
	c.lane = lane
	c.row.Lane = lane.String()
	rt, err := g.o.Registry.Route(task)
	if err != nil {
		return c.fail(ctx, "not_configured", "", err)
	}
	c.route = rt
	// Ngân sách (US-P1-03 AC11): cạn → BATCH dừng; INTERACTIVE / NEAR_REALTIME đổi sang mô hình chat rẻ nhất, KHÔNG BAO GIỜ bị từ chối.
	if task != TaskEmbedding && g.gate.BudgetState(ctx, id.CourseID) == BudgetExhausted {
		if lane == LaneBatch {
			return c.fail(ctx, "budget_blocked", "BUDGET", &ErrUnavailable{Reason: ReasonBudgetExhausted})
		}
		if cheap, ok := g.o.Registry.Cheapest(); ok {
			c.route = Route{Targets: []Target{cheap}, Params: rt.Params}
		}
	}
	c.maxRet = 2
	if lane == LaneInteractive {
		c.maxRet = 1
	}
	if v, ok := paramInt(rt.Params, "retries"); ok && v < c.maxRet { // chỉ thu hẹp
		c.maxRet = v
	}
	// Hạn chót: ctx gốc nếu có, không thì mặc định theo làn; timeout_s của tuyến chỉ THU HẸP.
	deadline, has := ctx.Deadline()
	if !has {
		d := g.o.RequestTimeout
		if lane == LaneBatch {
			d = g.o.BatchTimeout
		}
		deadline = now.Add(d)
	}
	if v, ok := paramInt(rt.Params, "timeout_s"); ok {
		if narrow := now.Add(time.Duration(v) * time.Second); narrow.Before(deadline) {
			deadline = narrow
		}
	}
	c.ctx, c.cancel = context.WithDeadline(ctx, deadline)
	if tid == "" {
		c.row.TraceID = "0"
	}
	return c, nil
}

// fail ghi audit cho lỗi sớm (trước khi có c.ctx) và trả (nil, err) để begin trả ra.
func (c *call) fail(ctx context.Context, status, kind string, err error) (*call, error) {
	c.ctx = ctx
	c.row.Status, c.row.ErrorKind = status, kind
	c.row.LatencyMS = int(c.g.o.Now().Sub(c.start) / time.Millisecond)
	c.g.o.Auditor.Record(c.row)
	c.logged = true
	return nil, err
}

func paramInt(p llmconfig.Params, k string) (int, bool) {
	v, ok := p[k]
	if !ok {
		return 0, false
	}
	n, err := v.Int64()
	return int(n), err == nil
}

func paramFloat(p llmconfig.Params, k string) *float64 {
	v, ok := p[k]
	if !ok {
		return nil
	}
	f, err := v.Float64()
	if err != nil {
		return nil
	}
	return &f
}

// opts hợp nhất tham số: Request ghi đè tuyến.
func (c *call) opts(t Target, r Request, schema json.RawMessage) provider.ChatOpts {
	o := provider.ChatOpts{Model: t.Model, Schema: schema, Fast: c.lane == LaneInteractive}
	for _, m := range r.Messages {
		o.Messages = append(o.Messages, provider.Message{Role: m.Role, Content: m.Content})
	}
	o.Temperature = paramFloat(c.route.Params, "temperature")
	if r.Params.Temperature != nil {
		o.Temperature = r.Params.Temperature
	}
	o.MaxTokens = defaultMaxTokens
	if v, ok := paramInt(c.route.Params, "max_tokens"); ok {
		o.MaxTokens = v
	}
	if r.Params.MaxTokens > 0 {
		o.MaxTokens = r.Params.MaxTokens
	}
	return o
}

func estTokens(o provider.ChatOpts) int {
	n := 0
	for _, m := range o.Messages {
		n += len(m.Content)
	}
	return (n+3)/4 + o.MaxTokens
}

// kindOf chuẩn hoá lỗi và phân biệt huỷ của client với hết hạn.
func (c *call) kindOf(err error) (provider.Kind, *provider.Error) {
	var pe *provider.Error
	if errors.As(err, &pe) {
		k := pe.Kind
		if k == provider.KindTimeout || k == provider.KindNetwork {
			if cerr := c.ctx.Err(); errors.Is(cerr, context.Canceled) {
				return provider.KindCancelled, pe
			}
		}
		return k, pe
	}
	switch {
	case errors.Is(err, context.Canceled):
		return provider.KindCancelled, &provider.Error{Kind: provider.KindCancelled}
	case errors.Is(err, context.DeadlineExceeded):
		return provider.KindTimeout, &provider.Error{Kind: provider.KindTimeout}
	}
	return provider.KindNetwork, &provider.Error{Kind: provider.KindNetwork}
}

func (c *call) remaining() time.Duration {
	if d, ok := c.ctx.Deadline(); ok {
		return d.Sub(c.g.o.Now())
	}
	return time.Hour
}

// attemptResult là kết quả cuối của một nhà cung cấp sau các lần thử.
type attemptResult[T any] struct {
	val      T
	tokens   int // tổng token thật (nếu biết) để đối soát
	attempts int
	kind     provider.Kind // rỗng = thành công
	perr     *provider.Error
}

// withRetry gọi op trên một nhà cung cấp, thử lại có jitter (SRS 4.2).
func withRetry[T any](c *call, op func(ctx context.Context) (T, int, error)) attemptResult[T] {
	var res attemptResult[T]
	for n := 0; ; n++ {
		res.attempts++
		v, tokens, err := op(c.ctx)
		if err == nil {
			res.val, res.tokens, res.kind, res.perr = v, tokens, "", nil
			return res
		}
		res.kind, res.perr = c.kindOf(err)
		if !res.kind.Retryable() || n >= c.maxRet {
			return res
		}
		base := min(retryBase<<n, retryCap)
		d := time.Duration(c.g.o.Rand() * float64(base)) // jitter đầy đủ: đều trong [0, base]
		if ra := res.perr.RetryAfter; ra > 0 {
			if ra > retryAfterMax {
				return res // chờ quá lâu: chuyển nhà kế tiếp
			}
			d = max(d, ra)
		}
		if c.remaining() < retryMinRemaining || c.remaining() < d {
			return res
		}
		if err := c.g.o.Sleep(c.ctx, d); err != nil {
			res.kind, res.perr = provider.KindCancelled, &provider.Error{Kind: provider.KindCancelled}
			if errors.Is(c.ctx.Err(), context.DeadlineExceeded) {
				res.kind, res.perr = provider.KindTimeout, &provider.Error{Kind: provider.KindTimeout}
			}
			return res
		}
	}
}

// chosen là nhà cung cấp đã trả lời.
type chosen[T any] struct {
	val    T
	target Target
	idx    int
	permit Permit
	tokens int
}

// runChain duyệt chuỗi: mạch → xin chỗ → thử lại → chuyển nhà kế. Thành công trả permit CHƯA trả (người gọi Done).
// Lỗi trả error đã chuẩn hoá của gói (ErrOverloaded, ErrDeadline, ErrBadRequest, ErrAllProvidersFailed, context.Canceled, ErrDimsMismatch).
func runChain[T any](c *call, est func(Target) int, op func(ctx context.Context, t Target) (T, int, error)) (chosen[T], error) {
	var zero chosen[T]
	targets := c.route.Targets
	lastKind := provider.Kind("")
	for idx, t := range targets {
		if err := c.ctx.Err(); err != nil {
			return zero, c.ctxErr(err)
		}
		c.row.Provider, c.row.Model, c.row.FallbackIndex = t.ProviderName, t.Model, idx
		if !c.g.gate.BreakerAllow(c.ctx, t.ProviderID) {
			lastKind = "circuit_open"
			c.row.ErrorKind = "circuit_open"
			c.warnFallback(targets, idx, "circuit_open")
			continue
		}
		permit, err := c.g.gate.Admit(c.ctx, Work{Lane: c.lane, Task: c.task, ProviderID: t.ProviderID, RPM: t.RPM, TPM: t.TPM, EstTokens: est(t)})
		if err != nil {
			return zero, c.admitErr(err)
		}
		c.row.QueueWaitMS += int(permit.Wait() / time.Millisecond)
		ar := withRetry(c, func(ctx context.Context) (T, int, error) { return op(ctx, t) })
		c.row.Attempts += ar.attempts
		if ar.kind == "" {
			c.g.gate.BreakerReport(c.ctx, t.ProviderID, "")
			return chosen[T]{val: ar.val, target: t, idx: idx, permit: permit, tokens: ar.tokens}, nil
		}
		permit.Done(-1)
		lastKind = ar.kind
		c.row.ErrorKind = string(ar.kind)
		if ar.perr != nil && ar.perr.Detail != "" {
			c.g.o.Log.DebugContext(c.ctx, "lỗi nhà cung cấp LLM", "provider", t.ProviderName, "kind", string(ar.kind), "detail", ar.perr.Detail)
		}
		if ar.kind.CountsToBreaker() {
			c.g.gate.BreakerReport(c.ctx, t.ProviderID, ar.kind)
		}
		switch {
		case ar.kind == provider.KindCancelled:
			return zero, c.ctxErr(context.Canceled)
		case ar.kind == provider.KindDimsMismatch:
			return zero, &ErrDimsMismatch{Expected: EmbedDims, Actual: dimsOf(ar.perr)}
		case !ar.kind.NextProvider():
			return zero, ErrBadRequest
		}
		c.warnFallback(targets, idx, string(ar.kind))
	}
	_ = lastKind
	if err := c.ctx.Err(); err != nil {
		return zero, c.ctxErr(err)
	}
	return zero, ErrAllProvidersFailed
}

// dimsOf lấy số chiều thật đã gắn vào Detail của lỗi DIMS_MISMATCH ("n").
func dimsOf(pe *provider.Error) int {
	if pe == nil {
		return 0
	}
	n := 0
	for _, ch := range pe.Detail {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func (c *call) warnFallback(targets []Target, idx int, kind string) {
	if idx+1 >= len(targets) {
		return
	}
	c.g.o.Log.WarnContext(c.ctx, "chuyển nhà cung cấp dự phòng",
		"task", string(c.task), "from", targets[idx].ProviderName, "to", targets[idx+1].ProviderName, "error_kind", kind)
}

// ctxErr đổi lỗi ctx thành lỗi của gói: hết hạn → ErrDeadline; huỷ → context.Canceled.
func (c *call) ctxErr(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrDeadline
	}
	return context.Canceled
}

func (c *call) admitErr(err error) error {
	var ov *ErrOverloaded
	switch {
	case errors.As(err, &ov):
		return err
	case errors.Is(err, context.DeadlineExceeded):
		return ErrDeadline
	case errors.Is(err, context.Canceled):
		return context.Canceled
	}
	return err
}

// finish ghi dòng audit cuối cùng (đúng một dòng mỗi lời gọi) và huỷ ctx con.
func (c *call) finish(resp *Response, err error) {
	if c.cancel != nil {
		defer c.cancel()
	}
	if c.logged {
		return
	}
	c.logged = true
	c.row.LatencyMS = int(c.g.o.Now().Sub(c.start) / time.Millisecond)
	if resp != nil {
		c.row.TokensIn, c.row.TokensOut, c.row.CostEst = resp.TokensIn, resp.TokensOut, resp.CostEst
		c.row.Provider, c.row.Model, c.row.FallbackIndex, c.row.Degraded = resp.Provider, resp.Model, resp.FallbackIndex, resp.Degraded
	}
	switch {
	case err == nil && resp != nil && resp.Degraded:
		c.row.Status = "degraded"
	case err == nil:
		c.row.Status, c.row.ErrorKind = "ok", ""
	default:
		c.row.Status = statusOf(err, c.row.ErrorKind)
	}
	c.g.o.Auditor.Record(c.row)
}

func statusOf(err error, kind string) string {
	var ov *ErrOverloaded
	var un *ErrUnavailable
	switch {
	case errors.As(err, &ov):
		return "overloaded"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, ErrDeadline):
		return "timeout"
	case errors.Is(err, ErrNotConfigured):
		return "not_configured"
	case errors.As(err, &un) && un.Reason == ReasonBudgetExhausted:
		return "budget_blocked"
	}
	switch provider.Kind(kind) {
	case provider.KindRateLimit:
		return "rate_limited"
	case provider.KindTimeout:
		return "timeout"
	case "circuit_open":
		return "circuit_open"
	}
	return "error"
}

func (c *call) response(t Target, idx int, text string, in, out int, permit Permit) Response {
	r := Response{Text: text, TokensIn: in, TokensOut: out, Provider: t.ProviderName, Model: t.Model, FallbackIndex: idx,
		QueueWait: permit.Wait(), CostEst: cost.Of(in, out, t.PriceIn, t.PriceOut)}
	permit.Done(in + out)
	c.g.gate.BudgetCharge(c.ctx, c.row.CourseID, r.CostEst)
	return r
}

func costOf(t Target, in, out int) decimal.Decimal { return cost.Of(in, out, t.PriceIn, t.PriceOut) }

func validRequest(r Request) error {
	if len(r.Messages) == 0 {
		return ErrBadRequest
	}
	for _, m := range r.Messages {
		if strings.TrimSpace(m.Content) == "" {
			return ErrBadRequest
		}
	}
	return nil
}

// Chat sinh văn bản một lần. Yêu cầu Shareable ở làn NEAR_REALTIME / BATCH (không GRADING) giống hệt nhau được hợp nhất trong tiến trình.
func (g *Gateway) Chat(ctx context.Context, r Request) (Response, error) {
	if lane, err := ResolveLane(r.Task, r.Lane); err == nil && shareable(r, lane) {
		if rt, rerr := g.o.Registry.Route(r.Task); rerr == nil {
			resp, ferr, _ := g.flights.do(ctx, flightKey(r, rt.Targets[0].Model), func(c context.Context) (Response, error) { return g.chat1(c, r) })
			return resp, ferr
		}
	}
	return g.chat1(ctx, r)
}

func (g *Gateway) chat1(ctx context.Context, r Request) (Response, error) {
	c, err := g.begin(ctx, r.Task, r.Lane, r.PIIMaskedCount)
	if err != nil {
		return Response{}, err
	}
	if err := validRequest(r); err != nil {
		c.row.ErrorKind = string(provider.KindBadRequest)
		c.finish(nil, err)
		return Response{}, err
	}
	resp, err := g.chat(c, r, nil)
	if errors.Is(err, ErrAllProvidersFailed) {
		if c.lane == LaneInteractive { // suy giảm có kiểm soát: trích nguyên văn, không sinh (SRS 4.3)
			resp = degraded(r)
			c.finish(&resp, nil)
			return resp, nil
		}
		err = &ErrUnavailable{Reason: ReasonAllFailed}
	}
	c.finish(optional(resp, err), err)
	return resp, err
}

func optional(r Response, err error) *Response {
	if err != nil {
		return nil
	}
	return &r
}

func (g *Gateway) chat(c *call, r Request, schema json.RawMessage) (Response, error) {
	type chatVal = provider.Result
	ch, err := runChain(c,
		func(t Target) int { return estTokens(c.opts(t, r, schema)) },
		func(ctx context.Context, t Target) (chatVal, int, error) {
			res, err := t.P.Chat(ctx, c.opts(t, r, schema))
			if err != nil {
				return res, 0, err
			}
			if len(schema) > 0 {
				res.Text = stripFence(res.Text)
				if verr := ValidateJSON(schema, json.RawMessage(res.Text)); verr != nil {
					return res, 0, &provider.Error{Kind: provider.KindBadResponse}
				}
			}
			return res, res.TokensIn + res.TokensOut, nil
		})
	if err != nil {
		return Response{}, err
	}
	return c.response(ch.target, ch.idx, ch.val.Text, ch.val.TokensIn, ch.val.TokensOut, ch.permit), nil
}

// stripFence bỏ hàng rào ```json ... ``` mà vài mô hình thêm quanh JSON.
func stripFence(s string) string {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "```") {
		t = strings.TrimPrefix(t, "```json")
		t = strings.TrimPrefix(t, "```")
		t = strings.TrimSuffix(strings.TrimSpace(t), "```")
	}
	return strings.TrimSpace(t)
}

// Structured trả JSON hợp lệ theo schema hoặc lỗi — không bao giờ trả JSON sai schema. Cách ép theo loại nhà cung cấp (provider.applySchema);
// kết quả LUÔN được kiểm bằng schema ở Go; sai schema = lỗi nhà cung cấp (tính vào mạch, chuyển dự phòng).
func (g *Gateway) Structured(ctx context.Context, r Request, schema json.RawMessage) (json.RawMessage, error) {
	c, err := g.begin(ctx, r.Task, r.Lane, r.PIIMaskedCount)
	if err != nil {
		return nil, err
	}
	var probe map[string]any
	if verr := validRequest(r); verr != nil || json.Unmarshal(schema, &probe) != nil {
		c.row.ErrorKind = string(provider.KindBadRequest)
		c.finish(nil, ErrBadRequest)
		return nil, ErrBadRequest
	}
	resp, err := g.chat(c, r, schema)
	if errors.Is(err, ErrAllProvidersFailed) {
		err = &ErrUnavailable{Reason: ReasonAllFailed}
	}
	c.finish(optional(resp, err), err)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(resp.Text), nil
}

// Embed nhúng các chuỗi; chia lô 100; vectơ ≠ 1536 chiều → ErrDimsMismatch (không thử lại, không trả vectơ).
func (g *Gateway) Embed(ctx context.Context, r EmbedRequest) ([][]float32, error) {
	c, err := g.begin(ctx, TaskEmbedding, r.Lane, r.PIIMaskedCount)
	if err != nil {
		return nil, err
	}
	bad := len(r.Inputs) == 0
	for _, s := range r.Inputs {
		bad = bad || strings.TrimSpace(s) == ""
	}
	if bad { // chuỗi rỗng: lỗi cục bộ, không gọi mạng
		c.row.ErrorKind = string(provider.KindBadRequest)
		c.finish(nil, ErrBadRequest)
		return nil, ErrBadRequest
	}
	type embVal struct {
		vecs   [][]float32
		tokens int
	}
	ch, err := runChain(c,
		func(Target) int {
			n := 0
			for _, s := range r.Inputs {
				n += len(s)
			}
			return (n + 3) / 4
		},
		func(ctx context.Context, t Target) (embVal, int, error) {
			var out [][]float32
			total := 0
			for i := 0; i < len(r.Inputs); i += EmbedBatch {
				batch := r.Inputs[i:min(i+EmbedBatch, len(r.Inputs))]
				vecs, tok, err := t.P.Embed(ctx, provider.EmbedOpts{Model: t.Model, Inputs: batch, Dims: EmbedDims})
				if err != nil {
					return embVal{}, 0, err
				}
				for _, v := range vecs {
					if len(v) != EmbedDims {
						return embVal{}, 0, &provider.Error{Kind: provider.KindDimsMismatch, Detail: itoa(len(v))}
					}
				}
				out = append(out, vecs...)
				total += tok
			}
			return embVal{out, total}, total, nil
		})
	if err != nil {
		c.finish(nil, err)
		return nil, err
	}
	resp := c.response(ch.target, ch.idx, "", ch.val.tokens, 0, ch.permit)
	c.finish(&resp, nil)
	return ch.val.vecs, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// FlushAudit đẩy ngay các dòng llm_audit đang đệm (test, và bước tắt gateway trước khi đóng pool).
func (g *Gateway) FlushAudit(ctx context.Context) {
	if g.o.Auditor != nil {
		g.o.Auditor.flush(ctx)
	}
}
