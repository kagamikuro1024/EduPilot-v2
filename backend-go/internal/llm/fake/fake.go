// Package fake: nhà cung cấp giả cho dev/test/tải (SRS 8.3). KHÔNG gọi mạng. Xác định theo đầu vào; cấu hình được độ trễ, tỉ lệ lỗi,
// khoá hợp lệ; chế độ phát lại đọc tệp ghi sẵn khoá theo sha256 của yêu cầu chuẩn hoá.
package fake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

// Settings là tham số chỉnh được lúc chạy (route thử `_test/llm/fake`).
type Settings struct {
	LatencyMin, LatencyMax time.Duration
	ErrorRate              float64       // 0–1
	ErrorKind              provider.Kind // mặc định SERVER
	ErrorStatus            int
	RetryAfter             time.Duration
	ValidKey               string        // khác rỗng: khoá khác → AUTH
	StreamDelay            time.Duration // cách nhau giữa các từ; mặc định 20 ms
	ReplayDir              string        // khác rỗng: chế độ phát lại
}

// Controller chia sẻ Settings và bộ đếm giữa mọi provider fake của một Registry.
type Controller struct {
	mu     sync.RWMutex
	s      Settings
	byName sync.Map     // tên provider → *atomic.Int64: số lời gọi sinh văn bản theo từng provider fake
	calls  atomic.Int64 // số lời gọi sinh văn bản tới provider (Chat/Stream/Structured) — D47
	embeds atomic.Int64
	active atomic.Int64
	peak   atomic.Int64
}

// NewController dựng Controller với Settings ban đầu.
func NewController(s Settings) *Controller {
	c := &Controller{}
	c.s = withDefaults(s)
	return c
}

func withDefaults(s Settings) Settings {
	if s.ErrorKind == "" {
		s.ErrorKind = provider.KindServer
	}
	if s.StreamDelay == 0 {
		s.StreamDelay = 20 * time.Millisecond
	}
	return s
}

// Set thay Settings.
func (c *Controller) Set(s Settings) { c.mu.Lock(); c.s = withDefaults(s); c.mu.Unlock() }

// Get đọc Settings hiện hành.
func (c *Controller) Get() Settings { c.mu.RLock(); defer c.mu.RUnlock(); return c.s }

// Calls là số lời gọi sinh văn bản đã tới provider.
func (c *Controller) Calls() int64 { return c.calls.Load() }

// CallsByProvider trả số lời gọi sinh văn bản theo từng provider fake (khoá = tên / id đã đặt ở NewNamed).
func (c *Controller) CallsByProvider() map[string]int64 {
	out := map[string]int64{}
	c.byName.Range(func(k, v any) bool { out[k.(string)] = v.(*atomic.Int64).Load(); return true }) //nolint:forcetypeassert // sync.Map nội bộ
	return out
}

// Embeds là số lời gọi nhúng đã tới provider.
func (c *Controller) Embeds() int64 { return c.embeds.Load() }

// Active là số lời gọi đang chạy.
func (c *Controller) Active() int64 { return c.active.Load() }

// Peak là đỉnh số lời gọi đang chạy đồng thời kể từ ResetPeak.
func (c *Controller) Peak() int64 { return c.peak.Load() }

// ResetPeak đặt lại đỉnh về 0.
func (c *Controller) ResetPeak() { c.peak.Store(0) }

func (c *Controller) enter() {
	n := c.active.Add(1)
	for {
		p := c.peak.Load()
		if n <= p || c.peak.CompareAndSwap(p, n) {
			return
		}
	}
}

// Provider là một provider fake gắn với khoá của bản ghi.
type Provider struct {
	c    *Controller
	name string
	key  string
	n    *atomic.Int64
}

// New dựng provider fake dùng chung Controller c (không đếm riêng theo tên).
func New(c *Controller, key string) *Provider { return NewNamed(c, "fake", key) }

// NewNamed như New nhưng đếm số lời gọi sinh văn bản theo `name` (hiện ở `_test/llm/stats` → fake_calls).
func NewNamed(c *Controller, name, key string) *Provider {
	v, _ := c.byName.LoadOrStore(name, new(atomic.Int64))
	return &Provider{c: c, name: name, key: key, n: v.(*atomic.Int64)} //nolint:forcetypeassert // sync.Map nội bộ
}

func (p *Provider) count() { p.c.calls.Add(1); p.n.Add(1) }

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func ctxKind(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &provider.Error{Kind: provider.KindTimeout}
	}
	return &provider.Error{Kind: provider.KindCancelled}
}

// gate áp khoá, độ trễ và lỗi ngẫu nhiên; trả Settings đã chụp.
func (p *Provider) gate(ctx context.Context) (Settings, error) {
	s := p.c.Get()
	if s.ValidKey != "" && p.key != s.ValidKey {
		return s, &provider.Error{Kind: provider.KindAuth, Status: 401}
	}
	lat := s.LatencyMin
	if s.LatencyMax > s.LatencyMin {
		lat += time.Duration(rand.Int64N(int64(s.LatencyMax - s.LatencyMin))) //nolint:gosec // độ trễ giả, không cần CSPRNG
	}
	if err := sleep(ctx, lat); err != nil {
		return s, ctxKind(err)
	}
	if s.ErrorRate > 0 && rand.Float64() < s.ErrorRate { //nolint:gosec // tỉ lệ lỗi giả
		st := s.ErrorStatus
		if st == 0 {
			st = statusOf(s.ErrorKind)
		}
		return s, &provider.Error{Kind: s.ErrorKind, Status: st, RetryAfter: s.RetryAfter}
	}
	return s, nil
}

func statusOf(k provider.Kind) int {
	switch k {
	case provider.KindAuth:
		return 401
	case provider.KindRateLimit:
		return 429
	case provider.KindServer:
		return 503
	case provider.KindModelNotFound:
		return 404
	case provider.KindBadRequest:
		return 400
	}
	return 0
}

func tokens(s string) int { return (len(s) + 3) / 4 }

func promptText(o provider.ChatOpts) string {
	var b strings.Builder
	for _, m := range o.Messages {
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

func lastUser(o provider.ChatOpts) string {
	for i := len(o.Messages) - 1; i >= 0; i-- {
		if o.Messages[i].Role != "system" && o.Messages[i].Role != "assistant" {
			return o.Messages[i].Content
		}
	}
	return ""
}

func answer(o provider.ChatOpts) string {
	q := []rune(strings.TrimSpace(lastUser(o)))
	if len(q) > 60 {
		q = q[:60]
	}
	return "Đây là câu trả lời giả lập cho: " + string(q)
}

// Chat trả văn bản xác định; có Schema → JSON sinh theo schema.
func (p *Provider) Chat(ctx context.Context, o provider.ChatOpts) (provider.Result, error) {
	p.count()
	p.c.enter()
	defer p.c.active.Add(-1)
	s, err := p.gate(ctx)
	if err != nil {
		return provider.Result{}, err
	}
	if s.ReplayDir != "" {
		return replayChat(s.ReplayDir, o)
	}
	text := answer(o)
	if len(o.Schema) > 0 {
		js, gerr := Generate(o.Schema)
		if gerr != nil {
			return provider.Result{}, &provider.Error{Kind: provider.KindBadRequest, Detail: "schema không hợp lệ"}
		}
		text = string(js)
	}
	return provider.Result{Text: text, TokensIn: tokens(promptText(o)), TokensOut: tokens(text)}, nil
}

// Stream phát từng từ cách nhau StreamDelay; ctx huỷ → dừng ngay.
func (p *Provider) Stream(ctx context.Context, o provider.ChatOpts) (<-chan provider.Delta, error) {
	p.count()
	p.c.enter()
	s, err := p.gate(ctx)
	if err != nil {
		p.c.active.Add(-1)
		return nil, err
	}
	text := answer(o)
	if s.ReplayDir != "" {
		r, rerr := replayChat(s.ReplayDir, o)
		if rerr != nil {
			p.c.active.Add(-1)
			return nil, rerr
		}
		text = r.Text
	}
	out := make(chan provider.Delta, 8)
	go func() {
		defer close(out)
		defer p.c.active.Add(-1)
		words := strings.SplitAfter(text, " ")
		for i, w := range words {
			if i > 0 {
				if err := sleep(ctx, s.StreamDelay); err != nil {
					return
				}
			}
			select {
			case out <- provider.Delta{Text: w}:
			case <-ctx.Done():
				return
			}
		}
		select {
		case out <- provider.Delta{Usage: &provider.Result{TokensIn: tokens(promptText(o)), TokensOut: tokens(text)}}:
		case <-ctx.Done():
		}
	}()
	return out, nil
}

// Embed trả vectơ chuẩn hoá L2, xác định theo chuỗi vào, đúng Dims chiều (mặc định 1536).
func (p *Provider) Embed(ctx context.Context, o provider.EmbedOpts) ([][]float32, int, error) {
	p.c.embeds.Add(1)
	p.c.enter()
	defer p.c.active.Add(-1)
	if _, err := p.gate(ctx); err != nil {
		return nil, 0, err
	}
	dims := o.Dims
	if dims <= 0 {
		dims = 1536
	}
	out := make([][]float32, len(o.Inputs))
	n := 0
	for i, in := range o.Inputs {
		out[i] = Vector(in, dims)
		n += tokens(in)
	}
	return out, n, nil
}

// Vector sinh vectơ giả xác định, chuẩn hoá L2.
func Vector(s string, dims int) []float32 {
	seed := sha256.Sum256([]byte(s))
	r := rand.New(rand.NewPCG(leUint64(seed[:8]), leUint64(seed[8:16]))) //nolint:gosec // vectơ giả xác định
	v := make([]float32, dims)
	for i := range v {
		v[i] = float32(r.Float64()*2 - 1)
	}
	provider.Normalize(v)
	return v
}

func leUint64(b []byte) uint64 {
	var x uint64
	for i := 7; i >= 0; i-- {
		x = x<<8 | uint64(b[i])
	}
	return x
}

// ---- phát lại ----

type recorded struct {
	Text      string `json:"text"`
	TokensIn  int    `json:"tokens_in"`
	TokensOut int    `json:"tokens_out"`
}

// ReplayKey là sha256 (hex) của yêu cầu chuẩn hoá: model, messages, tham số, schema.
func ReplayKey(o provider.ChatOpts) string {
	canon := struct {
		Model       string             `json:"model"`
		Messages    []provider.Message `json:"messages"`
		Temperature *float64           `json:"temperature,omitempty"`
		MaxTokens   int                `json:"max_tokens,omitempty"`
		Schema      json.RawMessage    `json:"schema,omitempty"`
	}{o.Model, o.Messages, o.Temperature, o.MaxTokens, compact(o.Schema)}
	b, _ := json.Marshal(canon)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func compact(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return raw
	}
	b, _ := json.Marshal(v) // khoá map được sắp xếp → chuẩn hoá
	return b
}

func replayChat(dir string, o provider.ChatOpts) (provider.Result, error) {
	b, err := os.ReadFile(filepath.Join(dir, ReplayKey(o)+".json")) //nolint:gosec // đường dẫn thử nghiệm do cấu hình
	if err != nil {
		return provider.Result{}, &provider.Error{Kind: provider.KindReplayMiss, Detail: "không có bản ghi cho yêu cầu này"}
	}
	var r recorded
	if err := json.Unmarshal(b, &r); err != nil {
		return provider.Result{}, &provider.Error{Kind: provider.KindBadResponse}
	}
	return provider.Result{Text: r.Text, TokensIn: r.TokensIn, TokensOut: r.TokensOut}, nil
}

// Record ghi một bản ghi phát lại (dùng bởi công cụ ghi âm có khoá thật).
func Record(dir string, o provider.ChatOpts, r provider.Result) error {
	b, err := json.MarshalIndent(recorded{r.Text, r.TokensIn, r.TokensOut}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ReplayKey(o)+".json"), b, 0o600)
}

// ---- sinh JSON theo schema ----

// Generate sinh một giá trị JSON xác định thoả schema (tập con: type, properties, required, items, enum, const, min/max, minItems).
func Generate(schema json.RawMessage) (json.RawMessage, error) {
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		return nil, err
	}
	v := gen(s, 0)
	return json.Marshal(v)
}

func gen(s map[string]any, depth int) any {
	if depth > 8 {
		return nil
	}
	if c, ok := s["const"]; ok {
		return c
	}
	if e, ok := s["enum"].([]any); ok && len(e) > 0 {
		return e[0]
	}
	typ, _ := s["type"].(string)
	if typ == "" {
		if _, ok := s["properties"]; ok {
			typ = "object"
		}
	}
	switch typ {
	case "object":
		out := map[string]any{}
		props, _ := s["properties"].(map[string]any)
		for k, v := range props {
			if sub, ok := v.(map[string]any); ok {
				out[k] = gen(sub, depth+1)
			}
		}
		return out
	case "array":
		items, _ := s["items"].(map[string]any)
		n := 1
		if m, ok := s["minItems"].(float64); ok && int(m) > n {
			n = int(m)
		}
		arr := make([]any, 0, n)
		for range n {
			arr = append(arr, gen(items, depth+1))
		}
		return arr
	case "integer":
		if m, ok := s["minimum"].(float64); ok {
			return int64(math.Ceil(m))
		}
		return 1
	case "number":
		if m, ok := s["minimum"].(float64); ok {
			return m
		}
		return 1
	case "boolean":
		return true
	case "null":
		return nil
	default:
		str := "mẫu"
		if m, ok := s["minLength"].(float64); ok && int(m) > len([]rune(str)) {
			str = strings.Repeat("a", int(m))
		}
		return str
	}
}
