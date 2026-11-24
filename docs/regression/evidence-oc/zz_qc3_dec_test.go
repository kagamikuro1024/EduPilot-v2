package provider_test

// Throwaway QC (qc3) — P-11-110 decision /systemone. Đặt tạm vào backend-go/internal/llm/provider.
import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

type capDec struct {
	mu                   sync.Mutex
	path, auth, goog, ct string
	body                 map[string]any
	n                    int
}

func decWith(t *testing.T, base string, status int, resp string) (provider.Decider, *capDec) {
	t.Helper()
	c := &capDec{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.n++
		c.path, c.auth, c.goog, c.ct = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("x-goog-api-key"), r.Header.Get("Content-Type")
		_ = json.Unmarshal(b, &c.body)
		c.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, resp)
	}))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)
	cl := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme, r.URL.Host = target.Scheme, target.Host
		return http.DefaultTransport.RoundTrip(r)
	}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return provider.NewDecision(provider.DecisionConfig{BaseURL: base, APIKey: canaryKey, HTTP: cl}).(provider.Decider), c
}

const sysOK = `{"model":"gemini-decision-1","answers":{"personal":{"type":"noul","noul":0.9},"intent":{"type":"choice","choice":"B","probabilities":{"A":0.1,"B":0.9},"confidence":0.9}},"usage":{"input_tokens":41}}`
const googOK = `{"answers":{"personal":{"type":"noul","noul":0.8},"intent":{"type":"choice","choice":"A","confidence":0.7}},"usage":{"total_input_tokens":9}}`

func TestQC3_OC64_systemOneWire(t *testing.T) {
	for _, base := range []string{"https://opencode.ai/inference/go/systemone/v1", "https://opencode.ai:443/inference/go/systemone/v1"} {
		d, c := decWith(t, base, 200, sysOK)
		res, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "Em là [[SV_1]] hỏi về hạn nộp bài", Questions: both})
		if err != nil {
			t.Fatal(err)
		}
		_, hasInput := c.body["input"]
		keys := []string{}
		for k := range c.body {
			keys = append(keys, k)
		}
		if c.path != "/inference/go/systemone/v1/systemone" || c.auth != "Bearer "+canaryKey || c.goog != "" || hasInput || c.body["state"] != "Em là [[SV_1]] hỏi về hạn nộp bài" || c.body["model"] != "gemini-decision-1" || c.body["questions"] == nil || res.TokensIn != 41 || c.ct != "application/json" {
			t.Errorf("%s: path=%q auth=%q goog=%q ct=%q keys=%v tokens=%d", base, c.path, c.auth, c.goog, c.ct, keys, res.TokensIn)
		}
		t.Logf("OK %s → POST %s; Authorization Bearer; không x-goog-api-key; thân khoá=%v (state, không input); tokens_in=%d; personal.noul=%.2f intent=%s/%.2f", base, c.path, keys, res.TokensIn, res.Answers["personal"].Noul, res.Answers["intent"].Choice, res.Answers["intent"].Confidence)
	}
}

func TestQC3_OC66_googleWire(t *testing.T) {
	d, c := decWith(t, "https://generativelanguage.googleapis.com/v1beta", 200, googOK)
	res, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "xin chào", Questions: both})
	if err != nil {
		t.Fatal(err)
	}
	_, hasState := c.body["state"]
	if c.path != "/v1beta/decisions" || c.goog != canaryKey || c.auth != "" || c.body["input"] != "xin chào" || hasState || res.TokensIn != 9 {
		t.Fatalf("path=%q goog=%q auth=%q body=%v tokens=%d", c.path, c.goog, c.auth, c.body, res.TokensIn)
	}
	t.Logf("OK Google: POST %s, x-goog-api-key, không Authorization, thân input (không state), tokens_in=%d", c.path, res.TokensIn)
}

func TestQC3_OC67_fakeHosts(t *testing.T) {
	for _, base := range []string{"https://opencode.ai.evil.example/v1beta", "https://api.opencode.ai/v1beta", "https://opencode.ai@evil.example/v1beta", "https://evil.example/opencode.ai/v1beta", "http://localhost:9999/v1beta", "https://OpenCode.ai.example.com/v1beta"} {
		d, c := decWith(t, base, 200, googOK)
		if _, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "x", Questions: both}); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		u, _ := url.Parse(base)
		if !strings.HasSuffix(c.path, "/decisions") || strings.Contains(c.path, "systemone") || c.auth != "" || c.goog != canaryKey || c.body["input"] != "x" || c.body["state"] != nil {
			t.Errorf("%s (host %q): path=%q auth=%q goog=%q body=%v", base, u.Hostname(), c.path, c.auth, c.goog, c.body)
		}
		t.Logf("OK %s → đường Google: %s, x-goog-api-key, không Bearer", base, c.path)
	}
}

func TestQC3_OC68_errorMap(t *testing.T) {
	for _, c := range []struct {
		st   int
		kind provider.Kind
	}{{529, provider.KindServer}, {500, provider.KindServer}, {502, provider.KindServer}, {503, provider.KindServer}, {401, provider.KindAuth}, {403, provider.KindAuth}, {429, provider.KindRateLimit}, {400, provider.KindBadRequest}, {422, provider.KindBadRequest}, {404, provider.KindModelNotFound}} {
		for _, base := range []string{"https://opencode.ai/inference/go/systemone/v1", "https://generativelanguage.googleapis.com/v1beta"} {
			d, _ := decWith(t, base, c.st, `{"detail":"x `+canaryKey+` Bearer `+canaryKey+`"}`)
			_, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "x", Questions: both})
			var pe *provider.Error
			if !errors.As(err, &pe) || pe.Kind != c.kind || pe.Status != c.st {
				t.Errorf("%s HTTP %d: %v", base, c.st, err)
				continue
			}
			if strings.Contains(pe.Error(), canaryKey) || strings.Contains(pe.Detail, canaryKey) {
				t.Errorf("%s HTTP %d: lộ khoá: %q", base, c.st, pe.Detail)
			}
		}
		t.Logf("OK HTTP %d → %s (System One và Google giống nhau; khoá không lộ trong Error/Detail)", c.st, c.kind)
	}
}

func TestQC3_OC68b_badBodyAndRedirect(t *testing.T) {
	d, _ := decWith(t, "https://opencode.ai/inference/go/systemone/v1", 200, `<html>không phải JSON</html>`)
	_, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "x", Questions: both})
	var pe *provider.Error
	if !errors.As(err, &pe) || pe.Kind != provider.KindBadResponse {
		t.Fatalf("thân hỏng: %v", err)
	}
	t.Logf("OK thân 200 không phải JSON → %s", pe.Kind)
	d, _ = decWith(t, "https://opencode.ai/inference/go/systemone/v1", 302, ``)
	_, err = d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "x", Questions: both})
	if !errors.As(err, &pe) {
		t.Fatalf("302: %v", err)
	}
	t.Logf("OK 302 → %s (không theo chuyển hướng)", pe.Kind)
}
