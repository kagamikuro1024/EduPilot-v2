package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm/fake"
)

// Golden của 13 thao tác cấu hình LLM (FEAT-llm-gateway US-P1-04 AC10): thân phản hồi, trường động được che, so BẰNG NHAU
// (khoá lạ hay thiếu khoá đều đỏ). Golden của PG không bị đụng. Ghi lại: UPDATE_GOLDEN=1 go test ./internal/contract -run TestGolden_LLM.
// Quy tắc dự án: KHÔNG sửa golden để test xanh — chỉ ghi lại khi hợp đồng đổi có chủ ý (đổi spec + PM duyệt).

var (
	reUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)
)

func mask(k string, v any) any {
	switch x := v.(type) {
	case map[string]any:
		for kk, vv := range x {
			x[kk] = mask(kk, vv)
		}
		return x
	case []any:
		for i := range x {
			x[i] = mask(k, x[i])
		}
		return x
	case string:
		switch {
		case k == "trace_id":
			return "<trace_id>"
		case k == "access_token":
			return "<jwt>"
		case reUUID.MatchString(x):
			return "<uuid>"
		case reTime.MatchString(x):
			return "<time>"
		}
	case float64:
		if k == "latency_ms" {
			return "<ms>"
		}
	}
	return v
}

func TestGolden_LLM(t *testing.T) {
	r := getRig(t)
	ctx := context.Background()
	reset := func() {
		if _, err := r.deps.DB.Exec(ctx, `truncate llm_task_routes, llm_models, llm_providers, llm_audit; delete from llm_budgets`); err != nil {
			t.Fatal(err)
		}
		keys, _ := r.deps.Redis.Keys(ctx, "ep:llm:budget*").Result()
		if len(keys) > 0 {
			r.deps.Redis.Del(ctx, keys...)
		}
		r.deps.LLM.Budget.Invalidate()
		_ = r.deps.LLM.Registry.Reload(ctx)
	}
	reset()
	ctl := r.deps.LLM.Registry.Fake()
	old := ctl.Get()
	ctl.Set(fake.Settings{ValidKey: "good-key"})
	defer func() { ctl.Set(old); reset() }()

	admin := r.token(t, uuid.NewString(), auth.RoleAdmin)
	course := "11111111-1111-7111-8111-111111111111"
	do := func(method, path, body, name string, want int, hdr ...string) []byte {
		t.Helper()
		var rd io.Reader
		if body != "" {
			rd = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, r.srv.URL+"/api/v1"+path, rd)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+admin)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for i := 0; i+1 < len(hdr); i += 2 {
			req.Header.Set(hdr[i], hdr[i+1])
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s → %d (cần %d): %s", method, path, resp.StatusCode, want, b)
		}
		if name != "" {
			checkGolden(t, name, b)
		}
		return b
	}
	idem := func() []string { return []string{"Idempotency-Key", "golden-" + uuid.NewString()} }
	prov := func(name, key string) string {
		return `{"type":"fake","name":"` + name + `","api_key":"` + key + `","models":[` +
			`{"model":"fake-chat","kind":"chat","price_in":"4000","price_out":"16000"},` +
			`{"model":"fake-embed","kind":"embedding","dims":1536,"price_in":"20","price_out":"0"}]}`
	}

	do("POST", "/admin/llm/providers", prov("G", "bad-key"), "createLLMProvider.422", 422, idem()...)
	made := do("POST", "/admin/llm/providers", prov("G", "good-key"), "createLLMProvider", 201, idem()...)
	var p struct {
		ID     string `json:"id"`
		Models []struct{ ID, Kind string }
	}
	if err := json.Unmarshal(made, &p); err != nil {
		t.Fatal(err)
	}
	var chat, emb string
	for _, m := range p.Models {
		if m.Kind == "chat" {
			chat = m.ID
		} else {
			emb = m.ID
		}
	}
	do("GET", "/admin/llm/providers", "", "listLLMProviders", 200)
	do("PUT", "/admin/llm/providers/"+p.ID, `{"type":"fake","name":"G2","rpm_limit":60,"version":1}`, "updateLLMProvider", 200)
	do("POST", "/admin/llm/providers/"+p.ID+"/test", "", "testLLMProvider", 200)
	do("POST", "/admin/llm/providers/test", `{"type":"fake","api_key":"bad-key","model":"fake-chat"}`, "testLLMCandidate", 200)
	do("GET", "/admin/llm/routes", "", "listLLMRoutes", 200)
	do("PUT", "/admin/llm/routes", `{"task":"CHAT","chain":["`+chat+`"],"params":{"temperature":0.2},"version":0}`, "putLLMRoute", 200)
	do("PUT", "/admin/llm/routes", `{"task":"EMBEDDING","chain":["`+emb+`"],"version":0}`, "putLLMRoute.embedding", 200)
	do("GET", "/admin/llm/routes", "", "listLLMRoutes.configured", 200)
	do("GET", "/admin/llm/usage?from=2026-01-01&to=2026-01-08", "", "getLLMUsage", 200)
	do("GET", "/admin/llm/budget", "", "getLLMBudget", 200)
	do("PUT", "/admin/llm/budget", `{"daily_limit":"100000","monthly_limit":"2000000","version":0}`, "putLLMBudget", 200)
	do("GET", "/courses/"+course+"/llm-budget", "", "getLLMCourseBudget", 200)
	do("PUT", "/courses/"+course+"/llm-budget", `{"daily_limit":"5000","monthly_limit":null,"version":0}`, "putLLMCourseBudget", 200)
	do("DELETE", "/admin/llm/providers/"+uuid.NewString(), "", "deleteLLMProvider.404", 404)
}

func checkGolden(t *testing.T, name string, body []byte) { checkGoldenIn(t, "llm", name, body) }

func checkGoldenIn(t *testing.T, dir, name string, body []byte) {
	t.Helper()
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("%s: thân không phải JSON: %v (%s)", name, err, body)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(mask("", v)); err != nil { // map → khoá được sắp xếp, đầu ra ổn định
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "golden", dir, name+".json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // tệp golden trong repo
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("thiếu golden %s (UPDATE_GOLDEN=1 để tạo lần đầu): %v", path, err)
	}
	if !bytes.Equal(want, buf.Bytes()) {
		t.Errorf("%s lệch golden %s:\n--- cần\n%s--- thực tế\n%s", name, path, want, buf.Bytes())
	}
}
