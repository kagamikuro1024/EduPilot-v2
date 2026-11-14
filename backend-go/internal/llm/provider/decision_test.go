package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

const canaryKey = "AIza-CANARY-0123456789"

func decider(t *testing.T, h http.HandlerFunc) provider.Decider {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return provider.NewDecision(provider.DecisionConfig{BaseURL: srv.URL + "/v1beta", APIKey: canaryKey}).(provider.Decider)
}

var qNoul = provider.DecideQuestion{Type: provider.QuestionNoul, Instructions: "Có tên người cụ thể không?"}
var qChoice = provider.DecideQuestion{Type: provider.QuestionChoice, Instructions: "Chọn ý định", Criteria: map[string]string{"A": "điểm danh", "B": "tài liệu"}}
var both = map[string]provider.DecideQuestion{"personal": qNoul, "intent": qChoice}

// TestDecisionProviderContract: đúng URL, header khoá (không ở URL), thân {model, input, questions}; đọc noul + choice; token vào.
func TestDecisionProviderContract(t *testing.T) {
	t.Parallel()
	var got map[string]any
	var path, query, key string
	d := decider(t, func(w http.ResponseWriter, r *http.Request) {
		path, query, key = r.URL.Path, r.URL.RawQuery, r.Header.Get("x-goog-api-key")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = io.WriteString(w, `{"answers":{"personal":{"type":"noul","noul":0.93,"confidence":0.93},"intent":{"type":"choice","choice":"B","probabilities":{"A":0.04,"B":0.96},"confidence":0.96}},"usage":{"total_input_tokens":120,"total_output_tokens":0}}`)
	})
	res, err := d.Decide(t.Context(), provider.DecideOpts{Model: "gemini-decision-1", Input: "xin chào [STUDENT_1]", Questions: both})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1beta/decisions" || key != canaryKey || strings.Contains(query, "key") {
		t.Fatalf("path=%q query=%q key=%q", path, query, key)
	}
	if got["model"] != "gemini-decision-1" || got["input"] != "xin chào [STUDENT_1]" {
		t.Fatalf("thân: %v", got)
	}
	qs := got["questions"].(map[string]any)
	if qs["personal"].(map[string]any)["type"] != "noul" || qs["intent"].(map[string]any)["criteria"].(map[string]any)["B"] != "tài liệu" {
		t.Fatalf("questions: %v", qs)
	}
	if a := res.Answers["personal"]; a.Noul != 0.93 || a.Confidence != 0.93 || a.Type != "noul" {
		t.Fatalf("noul: %+v", a)
	}
	if a := res.Answers["intent"]; a.Choice != "B" || a.Confidence != 0.96 || a.Probabilities["A"] != 0.04 {
		t.Fatalf("choice: %+v", a)
	}
	if res.TokensIn != 120 {
		t.Fatalf("token: %+v", res)
	}
}

// noul thấp (P(có) = 0,02) → confidence = 1 − P: độ tin cậy của câu "không".
func TestDecisionNoulConfidenceIsMax(t *testing.T) {
	t.Parallel()
	d := decider(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"answers":{"personal":{"type":"noul","noul":0.02}},"usage":{"total_input_tokens":1}}`)
	})
	res, err := d.Decide(t.Context(), provider.DecideOpts{Model: "m", Input: "x", Questions: map[string]provider.DecideQuestion{"personal": qNoul}})
	if err != nil || res.Answers["personal"].Confidence != 0.98 {
		t.Fatalf("confidence = %v, err = %v", res.Answers["personal"].Confidence, err)
	}
}

// TestDecideErrorMap (tầng provider): bảng mã HTTP / thân → Kind; Detail không chứa khoá.
func TestDecideErrorMap(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		headers map[string]string
		kind    provider.Kind
	}{
		{"401", 401, `{"error":{"message":"bad key AIza-CANARY-0123456789"}}`, nil, provider.KindAuth},
		{"400 khoá sai", 400, `{"error":{"message":"API key not valid"}}`, nil, provider.KindAuth},
		{"404", 404, `{}`, nil, provider.KindModelNotFound},
		{"429 + Retry-After", 429, `{}`, map[string]string{"Retry-After": "2"}, provider.KindRateLimit},
		{"503", 503, `{}`, nil, provider.KindServer},
		{"400 safety", 400, `{"error":{"code":"safety","message":"blocked"}}`, nil, provider.KindBlocked},
		{"400 prohibited_content", 400, `{"error":{"code":"prohibited_content"}}`, nil, provider.KindBlocked},
		{"400 blocklist", 400, `{"error":{"code":"blocklist"}}`, nil, provider.KindBlocked},
		{"400 content_blocked", 400, `{"error":{"status":"content_blocked"}}`, nil, provider.KindBlocked},
		{"400 thường", 400, `{"error":{"message":"bad question"}}`, nil, provider.KindBadRequest},
		{"200 sai định dạng", 200, `không phải JSON`, nil, provider.KindBadResponse},
		{"200 thiếu câu trả lời", 200, `{"answers":{},"usage":{}}`, nil, provider.KindBadResponse},
		{"200 nhãn lạ", 200, `{"answers":{"personal":{"noul":0.5},"intent":{"choice":"Z","confidence":0.9}},"usage":{}}`, nil, provider.KindBadResponse},
		{"200 noul ngoài khoảng", 200, `{"answers":{"personal":{"noul":1.7},"intent":{"choice":"A","confidence":0.9}},"usage":{}}`, nil, provider.KindBadResponse},
		{"200 sai kiểu", 200, `{"answers":{"personal":{"type":"choice","noul":0.5},"intent":{"choice":"A","confidence":0.9}},"usage":{}}`, nil, provider.KindBadResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := decider(t, func(w http.ResponseWriter, _ *http.Request) {
				for k, v := range tc.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			_, err := d.Decide(t.Context(), provider.DecideOpts{Model: "m", Input: "x", Questions: both})
			var pe *provider.Error
			if !errors.As(err, &pe) || pe.Kind != tc.kind {
				t.Fatalf("err = %v, muốn %s", err, tc.kind)
			}
			if tc.status == 429 && pe.RetryAfter != 2*time.Second {
				t.Fatalf("RetryAfter = %v", pe.RetryAfter)
			}
			if strings.Contains(pe.Detail, canaryKey) || len(pe.Detail) > provider.MaxDetail {
				t.Fatalf("Detail rò khoá hoặc quá dài: %q", pe.Detail)
			}
		})
	}
}

// SAFETY_BLOCK không phải lỗi của nhà cung cấp: không thử lại, không chuyển nhà, không tính vào cầu dao.
func TestBlockedKindIsNotAFault(t *testing.T) {
	t.Parallel()
	k := provider.KindBlocked
	if string(k) != "SAFETY_BLOCK" || k.Retryable() || k.NextProvider() || k.CountsToBreaker() {
		t.Fatalf("%s: retry=%v next=%v breaker=%v", k, k.Retryable(), k.NextProvider(), k.CountsToBreaker())
	}
}

func TestDecideRejectsBadQuestions(t *testing.T) {
	t.Parallel()
	d := decider(t, func(http.ResponseWriter, *http.Request) { t.Error("không được gọi mạng") })
	for name, o := range map[string]provider.DecideOpts{
		"không câu hỏi":     {Model: "m", Input: "x"},
		"đầu vào rỗng":      {Model: "m", Input: " ", Questions: map[string]provider.DecideQuestion{"q": qNoul}},
		"choice thiếu nhãn": {Model: "m", Input: "x", Questions: map[string]provider.DecideQuestion{"q": {Type: provider.QuestionChoice, Criteria: map[string]string{"A": "a"}}}},
		"score không dùng":  {Model: "m", Input: "x", Questions: map[string]provider.DecideQuestion{"q": {Type: provider.QuestionScore}}},
	} {
		var pe *provider.Error
		if _, err := d.Decide(t.Context(), o); !errors.As(err, &pe) || pe.Kind != provider.KindBadRequest {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// Hết giờ của ctx → TIMEOUT (không treo); chuyển hướng không được theo; ba hàm sinh chữ bị từ chối.
func TestDecideTimeoutRedirectAndNoGeneration(t *testing.T) {
	t.Parallel()
	d := decider(t, func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body) // đọc hết thân để máy chủ phát hiện client đóng kết nối
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var pe *provider.Error
	if _, err := d.Decide(ctx, provider.DecideOpts{Model: "m", Input: "x", Questions: map[string]provider.DecideQuestion{"q": qNoul}}); !errors.As(err, &pe) || pe.Kind != provider.KindTimeout {
		t.Fatalf("err = %v, muốn TIMEOUT", err)
	}
	hit := false
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }))
	defer other.Close()
	red := decider(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL, http.StatusFound) })
	if _, err := red.Decide(t.Context(), provider.DecideOpts{Model: "m", Input: "x", Questions: map[string]provider.DecideQuestion{"q": qNoul}}); err == nil || hit {
		t.Fatalf("chuyển hướng bị theo: err=%v hit=%v", err, hit)
	}
	p := provider.NewDecision(provider.DecisionConfig{APIKey: "k"})
	if _, err := p.Chat(t.Context(), provider.ChatOpts{}); err == nil {
		t.Fatal("Chat phải bị từ chối")
	}
}
