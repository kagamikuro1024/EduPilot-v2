package provider_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

// hop ghi header + thân của mọi request rồi đáp bằng `resp` (không gọi mạng thật: Transport đổi đích sang máy chủ giả).
type hop struct {
	mu      sync.Mutex
	headers []http.Header
	bodies  []string
}

func (h *hop) client(t *testing.T, resp string, sse bool) *http.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		h.mu.Lock()
		h.headers, h.bodies = append(h.headers, r.Header.Clone()), append(h.bodies, string(raw))
		h.mu.Unlock()
		if sse {
			w.Header().Set("Content-Type", "text/event-stream")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		_, _ = io.WriteString(w, resp)
	}))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	return &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme, r.URL.Host = target.Scheme, target.Host
		return http.DefaultTransport.RoundTrip(r)
	})}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenCodeGoHeadersAndEffort(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, base, task string
		want             bool
		session          string
	}{
		{"opencode CLASSIFY", "https://opencode.ai/inference/go/v1", "CLASSIFY", true, "edupilot-classify"},
		{"opencode không tác vụ", "https://opencode.ai/inference/go/v1", "", true, "edupilot"},
		{"9router", "http://127.0.0.1:20128/v1", "CLASSIFY", false, ""},
		{"host giống tên", "https://opencode.ai.evil.example/v1", "CLASSIFY", false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, stream := range []bool{false, true} {
				h := &hop{}
				resp := okChat
				if stream {
					resp = sse(`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"content":"a"}}]}`)
				}
				p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: tc.base, APIKey: "k", HTTP: h.client(t, resp, stream)})
				o := chatOpts()
				o.Task, o.MaxTokens = tc.task, 1024
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
					t.Fatalf("stream=%v: %d request", stream, len(h.headers))
				}
				hd, body := h.headers[0], h.bodies[0]
				var sent struct {
					MaxTokens int `json:"max_tokens"`
				}
				_ = json.Unmarshal([]byte(body), &sent)
				hasEffort := strings.Contains(body, `"reasoning_effort":"low"`)
				if tc.want {
					if hd.Get("User-Agent") != "edupilot/2.0" || hd.Get("x-opencode-session") != tc.session || !hasEffort {
						t.Errorf("stream=%v: UA=%q session=%q effort=%v", stream, hd.Get("User-Agent"), hd.Get("x-opencode-session"), hasEffort)
					}
					// reasoning tính vào max_tokens: 1024 cắt cụt JSON (BAD_RESPONSE ở UTILITY trên ep-r), nên mô hình reasoning cần thêm chỗ
					if sent.MaxTokens <= 1024+2048 {
						t.Errorf("stream=%v: max_tokens=%d, thiếu chỗ cho reasoning", stream, sent.MaxTokens)
					}
				} else if sent.MaxTokens != 1024 || hd.Get("User-Agent") == "edupilot/2.0" || hd.Get("x-opencode-session") != "" || strings.Contains(body, "reasoning_effort") {
					t.Errorf("stream=%v: nhà cung cấp khác bị gắn header/effort của OpenCode Go: UA=%q session=%q body=%s", stream, hd.Get("User-Agent"), hd.Get("x-opencode-session"), body)
				}
			}
		})
	}
}
