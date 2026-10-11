package llm_test

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

var contractSchemas = []string{"classify", "formula", "rubric"}

// shape trải JSON thành tập "đường dẫn:kiểu"; số nguyên và số thực cùng là "number"; phần tử mảng lấy theo phần tử đầu.
func shape(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("không phải JSON: %v", err)
	}
	out := map[string]string{}
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			out[path] = "object"
			for k, e := range x {
				walk(path+"."+k, e)
			}
		case []any:
			out[path] = "array"
			if len(x) > 0 {
				walk(path+"[]", x[0])
			}
		case string:
			out[path] = "string"
		case float64:
			out[path] = "number"
		case bool:
			out[path] = "boolean"
		default:
			out[path] = "null"
		}
	}
	walk("$", v)
	return out
}

func diffShapes(want, got map[string]string) []string {
	var d []string
	for k, v := range want {
		if g, ok := got[k]; !ok {
			d = append(d, "thiếu khoá "+k)
		} else if g != v {
			d = append(d, k+": "+g+", muốn "+v)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			d = append(d, "khoá thừa "+k)
		}
	}
	sort.Strings(d)
	return d
}

// TestProviderContract — US-P1-02 AC13: hình dạng của Structured() ở mỗi provider có ghi sẵn trong testdata/replay/<provider>/
// phải khớp hình dạng của `fake` cho cùng 3 schema mẫu. Chạy được không cần khoá; thiếu ghi sẵn của nhà cung cấp thật → BLOCKED.
func TestProviderContract(t *testing.T) {
	g, _, _ := fakeGateway(t, fake.Settings{})
	reference := map[string][]byte{}
	for _, name := range contractSchemas {
		schema, err := os.ReadFile(filepath.Join("testdata", "schemas", name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		out, err := g.Structured(context.Background(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("mẫu " + name)}, schema)
		if err != nil {
			t.Fatalf("fake Structured(%s): %v", name, err)
		}
		if verr := llm.ValidateJSON(schema, out); verr != nil {
			t.Fatalf("fake sinh JSON sai schema %s: %v", name, verr)
		}
		reference[name] = out
	}

	for _, prov := range []string{"fake", "fake-replay", "openai", "anthropic", "gemini"} {
		t.Run(prov, func(t *testing.T) {
			if prov == "fake" {
				return // chính là tham chiếu; đã kiểm hợp lệ theo schema ở trên
			}
			dir := filepath.Join("testdata", "replay", prov)
			if _, err := os.Stat(dir); err != nil {
				t.Skip("BLOCKED: cần khoá thật để ghi (LLM_RECORD=1 OPENAI_API_KEY=… go test ./internal/llm -run TestRecordReplay)")
			}
			for _, name := range contractSchemas {
				rec, err := os.ReadFile(filepath.Join(dir, name+".json"))
				if err != nil {
					t.Fatalf("thiếu bản ghi %s/%s.json", prov, name)
				}
				schema, _ := os.ReadFile(filepath.Join("testdata", "schemas", name+".json"))
				if verr := llm.ValidateJSON(schema, rec); verr != nil {
					t.Errorf("%s/%s: bản ghi sai schema: %v", prov, name, verr)
				}
				if d := diffShapes(shape(t, reference[name]), shape(t, rec)); len(d) > 0 {
					t.Errorf("%s/%s lệch hình dạng so với fake: %s", prov, name, strings.Join(d, "; "))
				}
			}
		})
	}
}

// maxRecordCalls: trần số lời gọi thật mỗi nhà cung cấp khi ghi (3 schema + 1 nhúng = 4; còn dư cho thử lại).
const maxRecordCalls = 10

// secretShapes: bản ghi tuyệt đối không được chứa khoá, header xác thực hay mã tổ chức.
var secretShapes = []string{"Authorization", "authorization", "Bearer ", "sk-", "AIza", "org-", "x-api-key"}

// TestRecordReplay ghi phản hồi THẬT vào testdata/replay/<provider>/ — chỉ chạy tay có khoá:
// LLM_RECORD=1 OPENAI_API_KEY=… GEMINI_API_KEY=… go test ./internal/llm -run TestRecordReplay -v
// Mỗi nhà: 3 lời gọi Structured (max_tokens ≤ 300) + 1 lời gọi nhúng 1536 chiều (openai, gemini). Chỉ lưu NỘI DUNG JSON đã kiểm schema và
// số đo của vectơ — không lưu header, URL hay thân yêu cầu; vẫn quét lại tệp để chắc không lẫn khoá. Số lời gọi và token in ra qua t.Logf.
func TestRecordReplay(t *testing.T) {
	if os.Getenv("LLM_RECORD") != "1" {
		t.Skip("BLOCKED: cần khoá thật — LLM_RECORD=1 OPENAI_API_KEY=… ANTHROPIC_API_KEY=… GEMINI_API_KEY=…")
	}
	keys := map[string]string{"openai": os.Getenv("OPENAI_API_KEY"), "anthropic": os.Getenv("ANTHROPIC_API_KEY"), "gemini": os.Getenv("GEMINI_API_KEY")}
	embedModel := map[string]string{"openai": llm.DefaultOpenAIEmbedding, "gemini": "gemini-embedding-001"}
	ctx := context.Background()
	for _, prov := range []string{"openai", "anthropic", "gemini"} {
		key := keys[prov]
		if key == "" {
			continue
		}
		env := llm.EnvConfig{}
		switch prov {
		case "openai":
			env.OpenAIKey = key
		case "anthropic":
			env.AnthropicKey = key
		case "gemini":
			env.GeminiKey = key
		}
		var mu sync.Mutex
		var rows []llm.AuditRow
		aud := llm.NewAuditor(ctx, func(_ context.Context, r []llm.AuditRow) error {
			mu.Lock()
			defer mu.Unlock()
			rows = append(rows, r...)
			return nil
		}, newDiscardLog())
		reg := llm.NewRegistry(nil, env, nil, newDiscardLog())
		if err := reg.Load(ctx); err != nil {
			t.Fatal(err)
		}
		gw := llm.New(llm.Options{NoMask: true, Registry: reg, Auditor: aud, Log: newDiscardLog()})
		dir := filepath.Join("testdata", "replay", prov)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		calls, tokensIn, tokensOut := 0, 0, 0
		// Task CHAT chạy làn INTERACTIVE ⇒ bật suy nghĩ nhẹ (Fast) cho Gemini 3.x; làn khác thì 300 token bị "suy nghĩ" ăn hết ⇒ phản hồi rỗng.
		write := func(name string, data []byte) {
			for _, bad := range append([]string{key}, secretShapes...) {
				if strings.Contains(string(data), bad) {
					t.Fatalf("%s/%s chứa chuỗi cấm (%d ký tự đầu %q) — không ghi", prov, name, 3, bad[:min(3, len(bad))])
				}
			}
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		for _, name := range contractSchemas {
			if calls++; calls > maxRecordCalls {
				t.Fatalf("%s: vượt %d lời gọi", prov, maxRecordCalls)
			}
			schema, _ := os.ReadFile(filepath.Join("testdata", "schemas", name+".json"))
			out, err := gw.Structured(ctx, llm.Request{Task: llm.TaskChat, Params: llm.Params{MaxTokens: 300}, Messages: userMsg("Mẫu " + name + ": hãy điền dữ liệu minh hoạ hợp lý.")}, schema)
			if err != nil {
				t.Fatalf("%s/%s: %v", prov, name, err)
			}
			if verr := llm.ValidateJSON(schema, out); verr != nil {
				t.Fatalf("%s/%s: sai schema: %v", prov, name, verr)
			}
			write(name+".json", out)
		}
		if m, ok := embedModel[prov]; ok {
			calls++
			p := provider.NewOpenAI(provider.OpenAIConfig{Type: prov, APIKey: key})
			vecs, tok, err := p.Embed(ctx, provider.EmbedOpts{Model: m, Inputs: []string{"Xin chào, đây là câu thử nhúng."}, Dims: llm.EmbedDims})
			if err != nil {
				t.Fatalf("%s embed: %v", prov, err)
			}
			tokensIn += tok
			var sq float64
			for _, x := range vecs[0] {
				sq += float64(x) * float64(x)
			}
			meta, _ := json.Marshal(map[string]any{"model": m, "dims": len(vecs[0]), "l2_norm": math.Sqrt(sq)})
			write("embed.json", meta)
		}
		aud.Close(ctx)
		mu.Lock()
		attempts := 0
		for _, r := range rows {
			tokensIn += r.TokensIn
			tokensOut += r.TokensOut
			attempts += max(r.Attempts, 1)
		}
		mu.Unlock()
		t.Logf("replay %s: %d lời gọi (%d lần thử HTTP cho Structured), token vào %d, token ra %d", prov, calls, attempts, tokensIn, tokensOut)
	}
}

// TestEmbedReplay — US-P1-02 AC11: số đo ghi sẵn của lời gọi nhúng thật có đúng 1536 chiều và đã chuẩn hoá L2 (1 ± 1e-3; gemini sau Normalize 1 ± 1e-6).
func TestEmbedReplay(t *testing.T) {
	for _, prov := range []string{"openai", "gemini"} {
		t.Run(prov, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "replay", prov, "embed.json"))
			if err != nil {
				t.Skip("BLOCKED: chưa có bản ghi nhúng (LLM_RECORD=1 … go test ./internal/llm -run TestRecordReplay)")
			}
			var m struct {
				Model  string  `json:"model"`
				Dims   int     `json:"dims"`
				L2Norm float64 `json:"l2_norm"`
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			if m.Dims != llm.EmbedDims {
				t.Errorf("%s: %d chiều, muốn %d", prov, m.Dims, llm.EmbedDims)
			}
			tol := 1e-3
			if prov == "gemini" {
				tol = 1e-6
			}
			if math.Abs(m.L2Norm-1) > tol {
				t.Errorf("%s: chuẩn L2 = %.9f, muốn 1 ± %g", prov, m.L2Norm, tol)
			}
		})
	}
}

// TestReplayHasNoSecrets — mọi bản ghi đã commit không được chứa khoá, header xác thực hay mã tổ chức.
func TestReplayHasNoSecrets(t *testing.T) {
	err := filepath.WalkDir(filepath.Join("testdata", "replay"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		for _, bad := range secretShapes {
			if strings.Contains(string(raw), bad) {
				t.Errorf("%s chứa %q", path, bad)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
