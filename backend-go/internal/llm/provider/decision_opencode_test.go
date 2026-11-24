package provider_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

const ocSystemOneBase = "https://opencode.ai/inference/go/systemone/v1"

// ocDecider: Transport đổi đích sang httptest (không mạng thật); `h` thấy đường dẫn ĐẦY ĐỦ như gửi tới opencode.ai.
func ocDecider(t *testing.T, base string, h http.HandlerFunc) provider.Decider {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	cl := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme, r.URL.Host = target.Scheme, target.Host
		return http.DefaultTransport.RoundTrip(r)
	})}
	return provider.NewDecision(provider.DecisionConfig{BaseURL: base, APIKey: canaryKey, HTTP: cl}).(provider.Decider)
}

// OpenCode Go /systemone: POST {base}/systemone, `Authorization: Bearer`, trường `state` (không `input`), usage.input_tokens; câu trả lời giữ dạng Google.
func TestDecisionOpenCodeSystemOne(t *testing.T) {
	t.Parallel()
	var got map[string]any
	var path, auth, goog string
	d := ocDecider(t, ocSystemOneBase, func(w http.ResponseWriter, r *http.Request) {
		path, auth, goog = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("x-goog-api-key")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = io.WriteString(w, `{"model":"gemini-decision-1","answers":{"personal":{"type":"noul","noul":0.93},"intent":{"type":"choice","choice":"B","probabilities":{"A":0.04,"B":0.96},"confidence":0.96}},"usage":{"input_tokens":57,"output_tokens":9}}`)
	})
	res, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "xin chào [STUDENT_1]", Questions: both})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/inference/go/systemone/v1/systemone" || auth != "Bearer "+canaryKey || goog != "" {
		t.Fatalf("path=%q auth=%q goog=%q", path, auth, goog)
	}
	if _, has := got["input"]; has || got["state"] != "xin chào [STUDENT_1]" || got["model"] != "gemini-decision-1" {
		t.Fatalf("thân: %v", got)
	}
	if a := res.Answers["personal"]; a.Noul != 0.93 || a.Confidence != 0.93 {
		t.Fatalf("noul: %+v", a)
	}
	if a := res.Answers["intent"]; a.Choice != "B" || a.Confidence != 0.96 {
		t.Fatalf("choice: %+v", a)
	}
	if res.TokensIn != 57 {
		t.Fatalf("token vào = %d, cần 57 (usage.input_tokens)", res.TokensIn)
	}
}

// 529 (quá tải, System One) → SERVER; Google/9router không bị ảnh hưởng (decision_test.go).
func TestDecisionOpenCodeOverloaded(t *testing.T) {
	t.Parallel()
	d := ocDecider(t, ocSystemOneBase, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(529)
		_, _ = io.WriteString(w, `{"detail":"overloaded"}`)
	})
	_, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "x", Questions: both})
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindServer || pe.Status != 529 {
		t.Fatalf("err=%v", err)
	}
}

// Host khác opencode.ai (kể cả tên giống) vẫn dùng Google: /decisions, x-goog-api-key, `input`.
func TestDecisionNonOpenCodeKeepsGoogleWire(t *testing.T) {
	t.Parallel()
	var got map[string]any
	var path, auth, goog string
	d := ocDecider(t, "https://opencode.ai.evil.example/v1beta", func(w http.ResponseWriter, r *http.Request) {
		path, auth, goog = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("x-goog-api-key")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = io.WriteString(w, `{"answers":{"personal":{"type":"noul","noul":0.5},"intent":{"type":"choice","choice":"A","confidence":0.6}},"usage":{"total_input_tokens":3}}`)
	})
	res, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "xin chào", Questions: both})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1beta/decisions" || auth != "" || goog != canaryKey || got["input"] != "xin chào" || got["state"] != nil || res.TokensIn != 3 {
		t.Fatalf("path=%q auth=%q goog=%q body=%v tokens=%d", path, auth, goog, got, res.TokensIn)
	}
}
