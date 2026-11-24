package provider_test

// Throwaway QC (qc3): TC-OC-02..08, 24, 25, 32..34, 37, 39, 40, 43, 44. Đặt tạm vào backend-go/internal/llm/provider, xoá sau khi chạy.
import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

const ocBase = "https://opencode.ai/inference/go/v1"

func realKey(t *testing.T) string {
	b, err := os.ReadFile(os.Getenv("HOME") + "/.config/edupilot-team/secrets/opencode-go.key")
	if err != nil {
		t.Skip("không có khoá")
	}
	return strings.TrimSpace(string(b))
}

// rewrite: giữ nguyên URL (host thật trong base_url) nhưng gửi tới máy chủ giả.
func rewrite(srv *httptest.Server) *http.Client {
	u, _ := url.Parse(srv.URL)
	return &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme, r.URL.Host = u.Scheme, u.Host
		return http.DefaultTransport.RoundTrip(r)
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func doCall(t *testing.T, base, task string, stream bool, schema string) (http.Header, string) {
	t.Helper()
	h := &hop{}
	resp := okChat
	if stream {
		resp = sse(`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"a"}}]}`)
	}
	p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: base, APIKey: "k", HTTP: h.client(t, resp, stream)})
	o := chatOpts()
	o.Task = task
	if schema != "" {
		o.Schema = json.RawMessage(schema)
		resp = `{"id":"1","object":"chat.completion","model":"m","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"{\"score\":5}"}}]}`
		h = &hop{}
		p = provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: base, APIKey: "k", HTTP: h.client(t, resp, false)})
	}
	if stream {
		ch, err := p.Stream(t.Context(), o)
		if err != nil {
			t.Fatal(err)
		}
		for range ch {
		}
	} else if _, err := p.Chat(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	if len(h.headers) != 1 {
		t.Fatalf("%d request", len(h.headers))
	}
	return h.headers[0], h.bodies[0]
}

func TestQC3_OC02_03_04_headers(t *testing.T) {
	for _, c := range []struct {
		task, sess string
		stream     bool
	}{{"GRADING", "edupilot-grading", false}, {"CHAT", "edupilot-chat", true}, {"", "edupilot", false}, {"", "edupilot", true}} {
		hd, body := doCall(t, ocBase, c.task, c.stream, "")
		if hd.Get("User-Agent") != "edupilot/2.0" || hd.Get("x-opencode-session") != c.sess || !strings.Contains(body, `"reasoning_effort":"low"`) {
			t.Errorf("%+v: UA=%q sess=%q body=%s", c, hd.Get("User-Agent"), hd.Get("x-opencode-session"), body)
		}
		t.Logf("OK task=%q stream=%v UA=%s session=%s effort=low", c.task, c.stream, hd.Get("User-Agent"), hd.Get("x-opencode-session"))
	}
}

func TestQC3_OC05_structured(t *testing.T) {
	hd, body := doCall(t, ocBase, "GRADING", false, `{"type":"object","properties":{"score":{"type":"number"}},"required":["score"]}`)
	if hd.Get("User-Agent") != "edupilot/2.0" || hd.Get("x-opencode-session") != "edupilot-grading" || !strings.Contains(body, `"reasoning_effort":"low"`) {
		t.Errorf("thiếu header/effort: %q %q %s", hd.Get("User-Agent"), hd.Get("x-opencode-session"), body)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(body), &m)
	rf, _ := m["response_format"].(map[string]any)
	if rf["type"] != "json_object" {
		t.Errorf("response_format=%v", m["response_format"])
	}
	if !strings.Contains(body, `\"required\"`) && !strings.Contains(body, "required") {
		t.Errorf("schema không nằm trong thân")
	}
	msgs, _ := m["messages"].([]any)
	sys, _ := msgs[0].(map[string]any)
	if sys["role"] != "system" || !strings.Contains(sys["content"].(string), "score") {
		t.Errorf("schema không nằm trong lời nhắc hệ thống: %v", msgs)
	}
	t.Logf("OK response_format=json_object; system chứa schema; thân=%.200s", body)
}

func TestQC3_OC06_07_08_hosts(t *testing.T) {
	for _, c := range []struct {
		base string
		want bool
	}{
		{"http://127.0.0.1:20128/v1", false},
		{"https://opencode.ai.evil.example/v1", false},
		{"https://api.opencode.ai/v1", false},
		{"https://opencode.ai@evil.example/v1", false},
		{"https://evil.example/opencode.ai/v1", false},
		{"http://localhost:8080/v1", false},
		{"https://opencode.ai:443/inference/go/v1", true},
	} {
		for _, stream := range []bool{false, true} {
			hd, body := doCall(t, c.base, "CHAT", stream, "")
			got := hd.Get("x-opencode-session") != "" && hd.Get("User-Agent") == "edupilot/2.0" && strings.Contains(body, "reasoning_effort")
			any := hd.Get("x-opencode-session") != "" || hd.Get("User-Agent") == "edupilot/2.0" || strings.Contains(body, "reasoning_effort")
			if c.want && !got || !c.want && any {
				t.Errorf("%s stream=%v: UA=%q sess=%q body=%s", c.base, stream, hd.Get("User-Agent"), hd.Get("x-opencode-session"), body)
			}
		}
		t.Logf("OK %s → khớp OpenCode=%v", c.base, c.want)
	}
}

func TestQC3_OC24_redact(t *testing.T) {
	const k = "oc_sk_QC3FAKEKEY1234567890abcdef"
	for _, st := range []int{401, 500} {
		srv, _, _ := server(t, func(int, string, map[string]any) reply {
			return reply{st, `{"error":{"message":"bad key ` + k + ` Authorization: Bearer ` + k + `"}}`, nil}
		})
		p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: ocBase, APIKey: k, HTTP: rewrite(srv)})
		_, err := p.Chat(t.Context(), chatOpts())
		var pe *provider.Error
		if !errors.As(err, &pe) {
			t.Fatalf("%d: %v", st, err)
		}
		if strings.Contains(pe.Detail, "oc_sk_") || strings.Contains(pe.Error(), k) || strings.Contains(pe.Detail, k) {
			t.Errorf("%d: Detail lộ khoá: %q", st, pe.Detail)
		}
		t.Logf("OK HTTP %d Kind=%s Detail=%q", st, pe.Kind, pe.Detail)
	}
}

func TestQC3_OC39_statusMap(t *testing.T) {
	for _, c := range []struct {
		st   int
		kind provider.Kind
	}{{401, provider.KindAuth}, {403, provider.KindAuth}, {404, provider.KindModelNotFound}, {400, provider.KindBadRequest}, {429, provider.KindRateLimit}, {500, provider.KindServer}} {
		srv, n, _ := server(t, func(int, string, map[string]any) reply {
			return reply{c.st, `{"error":{"message":"x"}}`, map[string]string{"Retry-After": "2"}}
		})
		p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: ocBase, APIKey: "k", HTTP: rewrite(srv)})
		t0 := time.Now()
		_, err := p.Chat(t.Context(), chatOpts())
		var pe *provider.Error
		if !errors.As(err, &pe) || pe.Kind != c.kind || pe.Status != c.st {
			t.Errorf("%d: %v", c.st, err)
			continue
		}
		if n.Load() != 1 || time.Since(t0) > time.Second {
			t.Errorf("%d: %d request, %v", c.st, n.Load(), time.Since(t0))
		}
		if c.st == 429 && pe.RetryAfter != 2*time.Second {
			t.Errorf("RetryAfter=%v", pe.RetryAfter)
		}
		t.Logf("OK HTTP %d → %s retry_after=%v next=%v breaker=%v requests=%d %v", c.st, pe.Kind, pe.RetryAfter, pe.Kind.NextProvider(), pe.Kind.CountsToBreaker(), n.Load(), time.Since(t0).Round(time.Millisecond))
	}
}

func TestQC3_OC43_timeout(t *testing.T) {
	hold := make(chan struct{})
	srv, _, _ := server(t, func(int, string, map[string]any) reply { <-hold; return reply{200, okChat, nil} })
	t.Cleanup(func() { close(hold) })
	p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: ocBase, APIKey: "k", HTTP: rewrite(srv)})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	t0 := time.Now()
	_, err := p.Chat(ctx, chatOpts())
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindTimeout || time.Since(t0) > 3*time.Second {
		t.Fatalf("%v %v", err, time.Since(t0))
	}
	t.Logf("OK Kind=%s sau %v", pe.Kind, time.Since(t0).Round(time.Millisecond))
}

func TestQC3_OC44_redirect(t *testing.T) {
	var hit atomic.Int64
	steal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit.Add(1); _, _ = io.WriteString(w, okChat) }))
	defer steal.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, steal.URL+"/steal", 302) }))
	defer srv.Close()
	// HTTP client mặc định của NewOpenAI (HTTP=nil) với base_url trỏ thẳng máy chủ chuyển hướng (host khác opencode.ai: cùng mã dựng client)
	p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: srv.URL + "/v1", APIKey: "k"})
	_, err := p.Chat(t.Context(), chatOpts())
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindBadResponse || hit.Load() != 0 {
		t.Fatalf("err=%v hit=%d", err, hit.Load())
	}
	t.Logf("OK Kind=%s, máy chủ đích nhận %d request", pe.Kind, hit.Load())
}

// ---- gọi thật (khoá thật; không in khoá) ----
func realP(t *testing.T, key string) provider.Provider {
	return provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: ocBase, APIKey: key})
}

func TestQC3_OC32_chat(t *testing.T) {
	k := realKey(t)
	o := chatOpts()
	o.Model, o.Task, o.MaxTokens = "deepseek-v4.1-flash", "CHAT", 256
	o.Messages = []provider.Message{{Role: "user", Content: "Trả lời đúng một từ: xin chào"}}
	ctx, c := context.WithTimeout(t.Context(), 30*time.Second)
	defer c()
	t0 := time.Now()
	r, err := realP(t, k).Chat(ctx, o)
	if err != nil || strings.TrimSpace(r.Text) == "" || r.TokensIn <= 0 || r.TokensOut <= 0 {
		t.Fatalf("err=%v r=%+v", err, r)
	}
	t.Logf("OK %v text=%q in=%d out=%d estimated=%v", time.Since(t0).Round(time.Millisecond), r.Text, r.TokensIn, r.TokensOut, r.Estimated)
}

func TestQC3_OC33_stream(t *testing.T) {
	k := realKey(t)
	o := chatOpts()
	o.Model, o.Task, o.MaxTokens = "deepseek-v4.1-flash", "CHAT", 256
	o.Messages = []provider.Message{{Role: "user", Content: "Đếm từ 1 đến 5, mỗi số một dòng"}}
	ctx, c := context.WithTimeout(t.Context(), 40*time.Second)
	defer c()
	ch, err := realP(t, k).Stream(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	n, txt := 0, ""
	var usage *provider.Result
	for d := range ch {
		if d.Err != nil {
			t.Fatal(d.Err)
		}
		if d.Text != "" {
			n++
			txt += d.Text
		}
		if d.Usage != nil {
			usage = d.Usage
		}
	}
	if n < 1 || usage == nil || usage.TokensIn <= 0 || usage.TokensOut <= 0 {
		t.Fatalf("delta=%d usage=%+v", n, usage)
	}
	t.Logf("OK delta=%d text=%q usage in=%d out=%d", n, txt, usage.TokensIn, usage.TokensOut)
}

func TestQC3_OC34_structured(t *testing.T) {
	k := realKey(t)
	schema := `{"type":"object","properties":{"score":{"type":"number","minimum":0,"maximum":10},"comment":{"type":"string","minLength":1}},"required":["score","comment"],"additionalProperties":false}`
	for i := 0; i < 3; i++ {
		o := chatOpts()
		o.Model, o.Task, o.MaxTokens, o.Schema = "deepseek-v4.1-flash", "GRADING", 512, json.RawMessage(schema)
		o.Messages = []provider.Message{{Role: "system", Content: "Bạn chấm bài. Chỉ trả JSON."}, {Role: "user", Content: "Đề: Nêu 2 ưu điểm của chỉ mục B-tree. Bài làm: B-tree giữ cây cân bằng nên tra cứu O(log n) và hỗ trợ truy vấn khoảng. Chấm thang 10."}}
		ctx, c := context.WithTimeout(t.Context(), 60*time.Second)
		r, err := realP(t, k).Chat(ctx, o)
		c()
		if err != nil {
			t.Fatalf("lần %d: %v", i+1, err)
		}
		var v struct {
			Score   *float64 `json:"score"`
			Comment string   `json:"comment"`
		}
		dec := json.NewDecoder(strings.NewReader(r.Text))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&v); err != nil || v.Score == nil || *v.Score < 0 || *v.Score > 10 || strings.TrimSpace(v.Comment) == "" {
			t.Fatalf("lần %d: JSON không hợp lệ: err=%v text=%q", i+1, err, r.Text)
		}
		t.Logf("OK lần %d: score=%v comment=%.80q", i+1, *v.Score, v.Comment)
	}
}

func TestQC3_OC37_wrongKey(t *testing.T) {
	_ = realKey(t)
	bad := "oc_sk_" + strings.Repeat("Zx9Qw3", 8)
	o := chatOpts()
	o.Model, o.Task = "deepseek-v4.1-flash", "CHAT"
	ctx, c := context.WithTimeout(t.Context(), 20*time.Second)
	defer c()
	t0 := time.Now()
	_, err := realP(t, bad).Chat(ctx, o)
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindAuth || time.Since(t0) > 20*time.Second || strings.Contains(pe.Detail, bad) {
		t.Fatalf("err=%v %v", err, time.Since(t0))
	}
	t.Logf("OK Kind=%s status=%d sau %v detail=%q", pe.Kind, pe.Status, time.Since(t0).Round(time.Millisecond), pe.Detail)
}

func TestQC3_OC40_badModel(t *testing.T) {
	k := realKey(t)
	o := chatOpts()
	o.Model, o.Task = "deepseek-v9-khong-co", "CHAT"
	ctx, c := context.WithTimeout(t.Context(), 30*time.Second)
	defer c()
	t0 := time.Now()
	_, err := realP(t, k).Chat(ctx, o)
	var pe *provider.Error
	if !errors.As(err, &pe) || time.Since(t0) > 30*time.Second {
		t.Fatalf("err=%v %v", err, time.Since(t0))
	}
	t.Logf("OK Kind=%s status=%d sau %v detail=%q", pe.Kind, pe.Status, time.Since(t0).Round(time.Millisecond), pe.Detail)
}

var _ = net.Listen

// ---- P-11-107 (3600257a): max_tokens = gốc + 4096 chỉ cho host opencode.ai ----
func maxTok(t *testing.T, base string, mt int, stream bool, schema string) (any, bool) {
	t.Helper()
	h := &hop{}
	resp := okChat
	if stream {
		resp = sse(`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"a"}}]}`)
	}
	p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: base, APIKey: "k", HTTP: h.client(t, resp, stream)})
	o := chatOpts()
	o.MaxTokens, o.Task = mt, "UTILITY"
	if schema != "" {
		o.Schema = json.RawMessage(schema)
	}
	if stream {
		ch, err := p.Stream(t.Context(), o)
		if err != nil {
			t.Fatal(err)
		}
		for range ch {
		}
	} else if _, err := p.Chat(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(h.bodies[0]), &m)
	for _, k := range []string{"max_tokens", "max_completion_tokens"} {
		if v, ok := m[k]; ok {
			return v, true
		}
	}
	return nil, false
}

func TestQC3_OC46_maxTokensPlus4096(t *testing.T) {
	for _, c := range []struct {
		mt     int
		stream bool
		schema string
		want   float64
	}{{256, false, "", 4352}, {256, true, "", 4352}, {1024, false, "", 5120}, {512, false, `{"type":"object","properties":{"a":{"type":"string"}}}`, 4608}, {1, false, "", 4097}} {
		v, ok := maxTok(t, ocBase, c.mt, c.stream, c.schema)
		if !ok || v.(float64) != c.want {
			t.Errorf("MaxTokens=%d stream=%v schema=%v: gửi đi %v (có=%v), muốn %v", c.mt, c.stream, c.schema != "", v, ok, c.want)
		}
		t.Logf("OK opencode.ai MaxTokens=%d stream=%v structured=%v → thân max_tokens=%v", c.mt, c.stream, c.schema != "", v)
	}
}

func TestQC3_OC47_maxTokensZero(t *testing.T) {
	for _, stream := range []bool{false, true} {
		v, ok := maxTok(t, ocBase, 0, stream, "")
		if ok {
			t.Errorf("stream=%v: MaxTokens gốc 0 mà vẫn gửi max_tokens=%v", stream, v)
		}
		t.Logf("OK MaxTokens=0 stream=%v → thân có max_tokens=%v", stream, ok)
	}
}

func TestQC3_OC48_maxTokensOtherHosts(t *testing.T) {
	for _, base := range []string{"http://127.0.0.1:20128/v1", "https://opencode.ai.evil.example/v1", "https://api.opencode.ai/v1", "https://opencode.ai@evil.example/v1", "http://localhost:8080/v1"} {
		for _, stream := range []bool{false, true} {
			v, ok := maxTok(t, base, 256, stream, "")
			if !ok || v.(float64) != 256 {
				t.Errorf("%s stream=%v: max_tokens=%v (có=%v), muốn 256", base, stream, v, ok)
			}
		}
		t.Logf("OK %s → max_tokens giữ 256 (Chat+Stream)", base)
	}
}

func TestQC3_OC50_realSmallCap(t *testing.T) {
	k := realKey(t)
	for _, mt := range []int{16, 256} {
		o := chatOpts()
		o.Model, o.Task, o.MaxTokens = "deepseek-v4.1-flash", "UTILITY", mt
		o.Messages = []provider.Message{{Role: "user", Content: "Tóm tắt trong một câu: B-tree là cấu trúc cây cân bằng dùng cho chỉ mục."}}
		ctx, c := context.WithTimeout(t.Context(), 60*time.Second)
		t0 := time.Now()
		r, err := realP(t, k).Chat(ctx, o)
		c()
		if err != nil || strings.TrimSpace(r.Text) == "" {
			t.Fatalf("MaxTokens=%d: err=%v text=%q", mt, err, r.Text)
		}
		t.Logf("OK MaxTokens=%d: %v, text=%.80q, tokens_out=%d", mt, time.Since(t0).Round(time.Millisecond), r.Text, r.TokensOut)
	}
}

func TestQC3_OC49_realStructuredNoTruncate(t *testing.T) {
	k := realKey(t)
	schema := `{"type":"object","properties":{"summary":{"type":"string","minLength":1},"points":{"type":"array","items":{"type":"string"},"minItems":1}},"required":["summary","points"]}`
	for i := 0; i < 3; i++ {
		o := chatOpts()
		o.Model, o.Task, o.MaxTokens, o.Schema = "deepseek-v4.1-flash", "UTILITY", 1024, json.RawMessage(schema)
		o.Messages = []provider.Message{{Role: "user", Content: "Tóm tắt tình hình học tập: đi học 8/10 buổi, điểm giữa kỳ 6,5, nộp bài đúng hạn 3/4. Trả JSON {summary, points[]}."}}
		ctx, c := context.WithTimeout(t.Context(), 60*time.Second)
		t0 := time.Now()
		r, err := realP(t, k).Chat(ctx, o)
		c()
		var v struct {
			Summary string   `json:"summary"`
			Points  []string `json:"points"`
		}
		if err != nil || json.Unmarshal([]byte(r.Text), &v) != nil || v.Summary == "" || len(v.Points) == 0 {
			t.Fatalf("lần %d: err=%v text=%q", i+1, err, r.Text)
		}
		t.Logf("OK lần %d: %v, JSON đủ (summary + %d điểm), tokens_out=%d", i+1, time.Since(t0).Round(time.Millisecond), len(v.Points), r.TokensOut)
	}
}
