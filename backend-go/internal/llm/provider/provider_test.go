package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

const okChat = `{"id":"1","object":"chat.completion","model":"m","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"xin chào"}}],"usage":{"prompt_tokens":11,"completion_tokens":2,"total_tokens":13}}`

type reply struct {
	status  int
	body    string
	headers map[string]string
}

// server dựng máy chủ giả kiểu OpenAI; mỗi request chạy fn(n, thân yêu cầu).
func server(t *testing.T, fn func(n int, path string, body map[string]any) reply) (*httptest.Server, *atomic.Int64, *[]map[string]any) {
	t.Helper()
	var n atomic.Int64
	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		rp := fn(int(n.Add(1)), r.URL.Path, body)
		for k, v := range rp.headers {
			w.Header().Set(k, v)
		}
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(rp.status)
		_, _ = io.WriteString(w, rp.body)
	}))
	t.Cleanup(srv.Close)
	return srv, &n, &bodies
}

func newP(typ, url, key string) provider.Provider {
	return provider.NewOpenAI(provider.OpenAIConfig{Type: typ, BaseURL: url, APIKey: key})
}

func chatOpts() provider.ChatOpts {
	return provider.ChatOpts{Model: "m", Messages: []provider.Message{{Role: "user", Content: "xin chào"}}, MaxTokens: 256}
}

func TestErrorMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		status    int
		body      string
		headers   map[string]string
		kind      provider.Kind
		retryable bool
		next      bool
		breaker   bool
	}{
		{"401 OpenAI", 401, `{"error":{"message":"Incorrect API key","type":"invalid_request_error","code":"invalid_api_key"}}`, nil, provider.KindAuth, false, true, true},
		{"403", 403, `{"error":{"message":"forbidden"}}`, nil, provider.KindAuth, false, true, true},
		{"401 Anthropic (code rỗng)", 401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`, nil, provider.KindAuth, false, true, true},
		{"400 Gemini nói về API key", 400, `[{"error":{"code":400,"message":"API key not valid. Please pass a valid API key.","status":"INVALID_ARGUMENT"}}]`, nil, provider.KindAuth, false, true, true},
		{"404", 404, `{"error":{"message":"The model does not exist"}}`, nil, provider.KindModelNotFound, false, true, false},
		{"404 Anthropic (code rỗng)", 404, `{"type":"error","error":{"type":"not_found_error","message":"model: x"}}`, nil, provider.KindModelNotFound, false, true, false},
		{"404 Gemini mảng", 404, `[{"error":{"code":404,"message":"models/x is not found"}}]`, nil, provider.KindModelNotFound, false, true, false},
		{"429", 429, `{"error":{"message":"rate limit","type":"rate_limit_error"}}`, map[string]string{"Retry-After": "2"}, provider.KindRateLimit, true, true, true},
		{"408", 408, `{"error":{"message":"timeout"}}`, nil, provider.KindTimeout, true, true, true},
		{"500", 500, `{"error":{"message":"boom"}}`, nil, provider.KindServer, true, true, true},
		{"502 HTML", 502, `<html>Bad Gateway</html>`, map[string]string{"Content-Type": "text/html"}, provider.KindServer, true, true, true},
		{"503", 503, `{"error":{"message":"overloaded"}}`, nil, provider.KindServer, true, true, true},
		{"504 là SERVER (không phải TIMEOUT)", 504, `{}`, nil, provider.KindServer, true, true, true},
		{"400 thường", 400, `{"error":{"message":"bad param","type":"invalid_request_error"}}`, nil, provider.KindBadRequest, false, false, false},
		{"422", 422, `{"error":{"message":"unprocessable"}}`, nil, provider.KindBadRequest, false, false, false},
		{"400 nội dung bị lọc", 400, `{"error":{"message":"blocked by content filter","code":"content_filter"}}`, nil, provider.KindBadRequest, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv, calls, _ := server(t, func(int, string, map[string]any) reply { return reply{tc.status, tc.body, tc.headers} })
			_, err := newP("openai_compatible", srv.URL, "sk-CANARY-abcdef123456").Chat(t.Context(), chatOpts())
			var pe *provider.Error
			if !errors.As(err, &pe) {
				t.Fatalf("err = %v", err)
			}
			if pe.Kind != tc.kind || pe.Kind.Retryable() != tc.retryable || pe.Kind.NextProvider() != tc.next || pe.Kind.CountsToBreaker() != tc.breaker {
				t.Errorf("kind=%s retry=%v next=%v breaker=%v; muốn %s %v %v %v", pe.Kind, pe.Kind.Retryable(), pe.Kind.NextProvider(), pe.Kind.CountsToBreaker(),
					tc.kind, tc.retryable, tc.next, tc.breaker)
			}
			if pe.Status != tc.status {
				t.Errorf("status = %d", pe.Status)
			}
			// SDK KHÔNG được tự thử lại (WithMaxRetries(0)): đúng 1 lần gọi dù là 429 / 5xx
			if calls.Load() != 1 {
				t.Errorf("số lần gọi = %d, muốn 1 (Gateway mới là nơi thử lại)", calls.Load())
			}
			if strings.Contains(pe.Error(), "Incorrect") || strings.Contains(pe.Error(), "boom") {
				t.Errorf("Error() lộ thân lỗi của nhà cung cấp: %s", pe.Error())
			}
			if len([]rune(pe.Detail)) > provider.MaxDetail {
				t.Errorf("Detail dài %d", len([]rune(pe.Detail)))
			}
		})
	}
}

func TestRetryAfterParsed(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		h    map[string]string
		want time.Duration
	}{
		{map[string]string{"Retry-After": "3"}, 3 * time.Second},
		{map[string]string{"Retry-After-Ms": "1500", "Retry-After": "9"}, 1500 * time.Millisecond},
		{map[string]string{}, 0},
		{map[string]string{"Retry-After": "Wed, 21 Oct 2026 07:28:00 GMT"}, 0},
	} {
		srv, _, _ := server(t, func(int, string, map[string]any) reply { return reply{429, `{"error":{"message":"x"}}`, tc.h} })
		_, err := newP("openai_compatible", srv.URL, "k").Chat(t.Context(), chatOpts())
		var pe *provider.Error
		if !errors.As(err, &pe) || pe.RetryAfter != tc.want {
			t.Errorf("%v → %v, muốn %v", tc.h, pe, tc.want)
		}
	}
}

func TestNetworkAndTimeout(t *testing.T) {
	t.Parallel()
	// cổng đóng → NETWORK
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()
	_, err := newP("openai_compatible", "http://"+addr, "k").Chat(t.Context(), chatOpts())
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindNetwork {
		t.Errorf("cổng đóng: %v", err)
	}
	// quá hạn ctx → TIMEOUT
	slow, _, _ := server(t, func(int, string, map[string]any) reply {
		time.Sleep(300 * time.Millisecond)
		return reply{200, okChat, nil}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err = newP("openai_compatible", slow.URL, "k").Chat(ctx, chatOpts())
	if !errors.As(err, &pe) || pe.Kind != provider.KindTimeout {
		t.Errorf("quá hạn: %v", err)
	}
	// huỷ → CANCELLED
	ctx2, cancel2 := context.WithCancel(t.Context())
	go func() { time.Sleep(30 * time.Millisecond); cancel2() }()
	_, err = newP("openai_compatible", slow.URL, "k").Chat(ctx2, chatOpts())
	if !errors.As(err, &pe) || pe.Kind != provider.KindCancelled {
		t.Errorf("huỷ: %v", err)
	}
}

func TestChatSuccessAndUsage(t *testing.T) {
	t.Parallel()
	srv, _, bodies := server(t, func(int, string, map[string]any) reply { return reply{200, okChat, nil} })
	temp := 1.7
	for _, tc := range []struct {
		typ        string
		tokenKey   string
		otherKey   string
		wantTemp   float64
		wantEffort bool
	}{
		{"openai", "max_completion_tokens", "max_tokens", 1.7, false},
		{"anthropic", "max_completion_tokens", "max_tokens", 1, false},
		{"gemini", "max_tokens", "max_completion_tokens", 1, true},
		{"openai_compatible", "max_tokens", "max_completion_tokens", 1.7, false},
	} {
		o := chatOpts()
		o.Temperature, o.Fast = &temp, true
		res, err := newP(tc.typ, srv.URL, "k").Chat(t.Context(), o)
		if err != nil || res.Text != "xin chào" || res.TokensIn != 11 || res.TokensOut != 2 {
			t.Fatalf("%s: res=%+v err=%v", tc.typ, res, err)
		}
		b := (*bodies)[len(*bodies)-1]
		if b[tc.tokenKey] == nil || b[tc.otherKey] != nil {
			t.Errorf("%s: khoá token sai: %v", tc.typ, b)
		}
		if b["temperature"] != tc.wantTemp {
			t.Errorf("%s: temperature = %v, muốn %v", tc.typ, b["temperature"], tc.wantTemp)
		}
		if (b["reasoning_effort"] == "low") != tc.wantEffort {
			t.Errorf("%s: reasoning_effort = %v", tc.typ, b["reasoning_effort"])
		}
	}
}

func TestStructuredRequestShape(t *testing.T) {
	t.Parallel()
	schema := json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`)
	toolResp := `{"id":"1","object":"chat.completion","model":"m","choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"emit_result","arguments":"{\"a\":\"x\"}"}}]}}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`
	for _, tc := range []struct {
		typ   string
		reply string
		check func(t *testing.T, b map[string]any)
		text  string
	}{
		{"openai", okChat, func(t *testing.T, b map[string]any) {
			rf := b["response_format"].(map[string]any)
			js := rf["json_schema"].(map[string]any)
			if rf["type"] != "json_schema" || js["strict"] != true || js["schema"] == nil {
				t.Errorf("response_format = %v", rf)
			}
		}, "xin chào"},
		{"gemini", okChat, func(t *testing.T, b map[string]any) {
			if rf := b["response_format"].(map[string]any); rf["type"] != "json_schema" {
				t.Errorf("response_format = %v", rf)
			}
		}, "xin chào"},
		{"anthropic", toolResp, func(t *testing.T, b map[string]any) {
			if b["response_format"] != nil || b["tools"] == nil {
				t.Errorf("Anthropic bỏ qua response_format: phải dùng tool: %v", b)
			}
			tc := b["tool_choice"].(map[string]any)
			if tc["type"] != "function" || tc["function"].(map[string]any)["name"] != "emit_result" {
				t.Errorf("tool_choice = %v", tc)
			}
		}, `{"a":"x"}`},
		{"openai_compatible", okChat, func(t *testing.T, b map[string]any) {
			rf := b["response_format"].(map[string]any)
			msgs := b["messages"].([]any)
			first := msgs[0].(map[string]any)
			if rf["type"] != "json_object" || first["role"] != "system" || !strings.Contains(first["content"].(string), `"additionalProperties":false`) {
				t.Errorf("json_object + schema trong lời nhắc: %v", b)
			}
		}, "xin chào"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			srv, calls, bodies := server(t, func(int, string, map[string]any) reply { return reply{200, tc.reply, nil} })
			o := chatOpts()
			o.Schema = schema
			res, err := newP(tc.typ, srv.URL, "k").Chat(t.Context(), o)
			if err != nil || res.Text != tc.text {
				t.Fatalf("res=%+v err=%v", res, err)
			}
			if calls.Load() != 1 {
				t.Errorf("Structured phải là MỘT lời gọi (D47), có %d", calls.Load())
			}
			tc.check(t, (*bodies)[0])
		})
	}
}

func sse(chunks ...string) string {
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString("data: " + c + "\n\n")
	}
	b.WriteString("data: [DONE]\n\n")
	return b.String()
}

func TestStreamUsageAndEmptyChoices(t *testing.T) {
	t.Parallel()
	body := sse(
		`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"Xin "}}]}`,
		`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"chào"}}]}`,
		`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[],"usage":{"prompt_tokens":11,"completion_tokens":2,"total_tokens":13}}`,
	)
	srv, _, bodies := server(t, func(int, string, map[string]any) reply {
		return reply{200, body, map[string]string{"Content-Type": "text/event-stream"}}
	})
	ch, err := newP("openai", srv.URL, "k").Stream(t.Context(), chatOpts())
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	var usage *provider.Result
	for d := range ch {
		if d.Err != nil {
			t.Fatal(d.Err)
		}
		text.WriteString(d.Text)
		if d.Usage != nil {
			usage = d.Usage
		}
	}
	if text.String() != "Xin chào" || usage == nil || usage.TokensIn != 11 || usage.TokensOut != 2 {
		t.Fatalf("text=%q usage=%+v (chunk cuối có choices rỗng không được làm panic)", text.String(), usage)
	}
	so := (*bodies)[0]["stream_options"].(map[string]any)
	if so["include_usage"] != true {
		t.Errorf("stream_options = %v", so)
	}
}

func TestStreamErrors(t *testing.T) {
	t.Parallel()
	// lỗi HTTP trước luồng
	srv, _, _ := server(t, func(int, string, map[string]any) reply { return reply{503, `{"error":{"message":"x"}}`, nil} })
	ch, err := newP("openai", srv.URL, "k").Stream(t.Context(), chatOpts())
	if err != nil {
		t.Fatal(err)
	}
	d := <-ch
	var pe *provider.Error
	if !errors.As(d.Err, &pe) || pe.Kind != provider.KindServer {
		t.Errorf("lỗi trước luồng: %v", d.Err)
	}
	// lỗi giữa luồng (không có status) → SERVER
	mid := sse(
		`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"Xin"}}]}`,
		`{"error":{"type":"overloaded_error","message":"Overloaded"}}`,
	)
	srv2, _, _ := server(t, func(int, string, map[string]any) reply {
		return reply{200, mid, map[string]string{"Content-Type": "text/event-stream"}}
	})
	ch, _ = newP("anthropic", srv2.URL, "k").Stream(t.Context(), chatOpts())
	var got string
	var last provider.Delta
	for d := range ch {
		got += d.Text
		last = d
	}
	if got != "Xin" || !errors.As(last.Err, &pe) || pe.Kind != provider.KindServer {
		t.Errorf("giữa luồng: text=%q last=%+v", got, last)
	}
}

func TestEmbed(t *testing.T) {
	t.Parallel()
	vec := make([]float64, 4)
	vec[0], vec[1] = 3, 4
	emb := func(n int) string {
		var items []string
		for i := range n {
			items = append(items, fmt.Sprintf(`{"object":"embedding","index":%d,"embedding":[3,4,0,0]}`, i))
		}
		return `{"object":"list","model":"e","data":[` + strings.Join(items, ",") + `],"usage":{"prompt_tokens":6,"total_tokens":6}}`
	}
	srv, _, bodies := server(t, func(_ int, _ string, body map[string]any) reply {
		return reply{200, emb(len(body["input"].([]any))), nil}
	})
	for _, typ := range []string{"openai", "gemini"} {
		vecs, tok, err := newP(typ, srv.URL, "k").Embed(t.Context(), provider.EmbedOpts{Model: "e", Inputs: []string{"a", "b"}, Dims: 1536})
		if err != nil || len(vecs) != 2 || tok != 6 {
			t.Fatalf("%s: %v %d %v", typ, len(vecs), tok, err)
		}
		b := (*bodies)[len(*bodies)-1]
		if b["dimensions"] != float64(1536) {
			t.Errorf("%s: dimensions = %v", typ, b["dimensions"])
		}
		norm := vecs[0][0]*vecs[0][0] + vecs[0][1]*vecs[0][1]
		if typ == "gemini" && (norm < 0.999 || norm > 1.001) {
			t.Errorf("Gemini phải chuẩn hoá L2: |v|² = %v", norm)
		}
		if typ == "openai" && vecs[0][0] != 3 {
			t.Errorf("OpenAI giữ nguyên vectơ: %v", vecs[0])
		}
	}
	// số vectơ không khớp số đầu vào → BAD_RESPONSE
	bad, _, _ := server(t, func(int, string, map[string]any) reply { return reply{200, emb(1), nil} })
	_, _, err := newP("openai", bad.URL, "k").Embed(t.Context(), provider.EmbedOpts{Model: "e", Inputs: []string{"a", "b"}})
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindBadResponse {
		t.Errorf("err = %v", err)
	}
}

func TestBaseURLs(t *testing.T) {
	t.Parallel()
	for typ, want := range map[string]string{
		"openai":    "https://api.openai.com/v1",
		"anthropic": "https://api.anthropic.com/v1/",
		"gemini":    "https://generativelanguage.googleapis.com/v1beta/openai/",
	} {
		if got := provider.BaseURLFor(typ, ""); got != want {
			t.Errorf("%s: %s, muốn %s", typ, got, want)
		}
	}
	if got := provider.BaseURLFor("openai_compatible", "http://vllm:8000/v1"); got != "http://vllm:8000/v1" {
		t.Errorf("openai_compatible: %s", got)
	}
	if got := provider.BaseURLFor("openai", "http://override"); got != "http://override" {
		t.Errorf("ghi đè: %s", got)
	}
}

// TestKeyNeverLeaksViaErrors — US-P1-02 AC16: máy chủ phản chiếu Authorization + khoá trong thân lỗi.
func TestKeyNeverLeaksViaErrors(t *testing.T) {
	t.Parallel()
	const canary = "sk-CANARY-0123456789abcdefXYZ"
	srv, _, _ := server(t, func(int, string, map[string]any) reply {
		return reply{401, `{"error":{"message":"bad key ` + canary + ` header Authorization: Bearer ` + canary + ` " ,"param":"` + strings.Repeat("z", 400) + `"}}`,
			map[string]string{"X-Echo-Authorization": "Bearer " + canary}}
	})
	_, err := newP("openai_compatible", srv.URL, canary).Chat(t.Context(), chatOpts())
	var pe *provider.Error
	if !errors.As(err, &pe) {
		t.Fatal(err)
	}
	for _, s := range []string{err.Error(), pe.Detail, fmt.Sprintf("%v %+v %#v", err, pe, pe)} {
		if strings.Contains(s, canary) || strings.Contains(s, "Bearer "+canary) {
			t.Errorf("lộ khoá: %s", s)
		}
	}
	if len([]rune(pe.Detail)) > 200 || !strings.Contains(pe.Detail, "[REDACTED]") {
		t.Errorf("Detail = %q (%d ký tự)", pe.Detail, len([]rune(pe.Detail)))
	}
	if got := provider.Redact("x sk-ABCDEFGH12345 y Bearer abc.def z"); strings.Contains(got, "ABCDEFGH") || strings.Contains(got, "abc.def") {
		t.Errorf("Redact = %q", got)
	}
}

// SDK không tự thử lại (WithMaxRetries(0)): 429 + Retry-After 3 → đúng 1 lần gọi, không chờ 6 s như mặc định (research 2026-10-03).
func TestSDKNoInternalRetry(t *testing.T) {
	t.Parallel()
	srv, calls, _ := server(t, func(int, string, map[string]any) reply {
		return reply{429, `{"error":{"message":"slow down"}}`, map[string]string{"Retry-After": "3"}}
	})
	start := time.Now()
	_, err := newP("openai", srv.URL, "k").Chat(t.Context(), chatOpts())
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindRateLimit || pe.RetryAfter != 3*time.Second {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 1 || time.Since(start) > time.Second {
		t.Errorf("calls=%d sau %v, muốn 1 lần gọi và không chờ", calls.Load(), time.Since(start))
	}
}

// 429 với Retry-After 30 mà ctx chỉ còn 2 s vẫn là RATE_LIMIT (SDK mặc định sẽ trả DeadlineExceeded và mất status).
func TestRateLimitKeepsStatusOnDeadline(t *testing.T) {
	t.Parallel()
	srv, _, _ := server(t, func(int, string, map[string]any) reply {
		return reply{429, `{"error":{"message":"x"}}`, map[string]string{"Retry-After": "30"}}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	start := time.Now()
	_, err := newP("openai", srv.URL, "k").Chat(ctx, chatOpts())
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindRateLimit || pe.Status != 429 {
		t.Fatalf("err = %v, muốn RATE_LIMIT 429", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("mất %v", time.Since(start))
	}
}

// Structured chọn cách gọi CỐ ĐỊNH theo type, một lời gọi (D47): nhà cung cấp trả 400 cho json_schema → KHÔNG có yêu cầu thứ hai, BAD_REQUEST thẳng.
func TestStructuredDowngrade(t *testing.T) {
	t.Parallel()
	schema := json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`)
	for _, typ := range []string{"openai", "gemini", "anthropic", "openai_compatible"} {
		srv, calls, bodies := server(t, func(int, string, map[string]any) reply {
			return reply{400, `{"error":{"message":"response_format json_schema is not supported","type":"invalid_request_error"}}`, nil}
		})
		o := chatOpts()
		o.Schema = schema
		_, err := newP(typ, srv.URL, "k").Chat(t.Context(), o)
		var pe *provider.Error
		if !errors.As(err, &pe) || pe.Kind != provider.KindBadRequest {
			t.Errorf("%s: err = %v, muốn BAD_REQUEST thẳng", typ, err)
		}
		if calls.Load() != 1 || len(*bodies) != 1 {
			t.Errorf("%s: %d yêu cầu, muốn đúng 1 (không lùi sang json_object)", typ, calls.Load())
		}
	}
}

// Gemini: gửi dimensions 1536 và chuẩn hoá L2 (|v| = 1 ± 1e-6).
func TestEmbedGeminiNormalizesL2(t *testing.T) {
	t.Parallel()
	srv, _, bodies := server(t, func(int, string, map[string]any) reply {
		return reply{200, `{"object":"list","model":"e","data":[{"object":"embedding","index":0,"embedding":[0.5,1.5,-2,0.25]}],"usage":{"prompt_tokens":1,"total_tokens":1}}`, nil}
	})
	vecs, _, err := newP("gemini", srv.URL, "k").Embed(t.Context(), provider.EmbedOpts{Model: "gemini-embedding-001", Inputs: []string{"a"}, Dims: 1536})
	if err != nil {
		t.Fatal(err)
	}
	var s float64
	for _, x := range vecs[0] {
		s += float64(x) * float64(x)
	}
	if math.Abs(math.Sqrt(s)-1) > 1e-6 {
		t.Errorf("|v| = %v, muốn 1 ± 1e-6", math.Sqrt(s))
	}
	if (*bodies)[0]["dimensions"] != float64(1536) {
		t.Errorf("dimensions = %v", (*bodies)[0]["dimensions"])
	}
}

// Không theo chuyển hướng (SSRF qua base_url của ADMIN): máy chủ trả 302 → đúng 1 yêu cầu, không tới địa chỉ đích.
func TestProviderNoRedirectFollow(t *testing.T) {
	t.Parallel()
	var hits atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	defer target.Close()
	srv, calls, _ := server(t, func(int, string, map[string]any) reply {
		return reply{302, ``, map[string]string{"Location": target.URL + "/steal"}}
	})
	_, err := newP("openai_compatible", srv.URL, "k").Chat(t.Context(), chatOpts())
	if err == nil {
		t.Fatal("302 phải là lỗi")
	}
	if calls.Load() != 1 || hits.Load() != 0 {
		t.Errorf("calls=%d hits=%d: không được theo chuyển hướng", calls.Load(), hits.Load())
	}
}
