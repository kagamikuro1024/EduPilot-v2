package llm_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
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

// TestRecordReplay ghi phản hồi THẬT vào testdata/replay/<provider>/ — chỉ chạy tay có khoá (BLOCKED tới khi chủ dự án có khoá, Q11).
func TestRecordReplay(t *testing.T) {
	if os.Getenv("LLM_RECORD") != "1" {
		t.Skip("BLOCKED: cần khoá thật — LLM_RECORD=1 OPENAI_API_KEY=… ANTHROPIC_API_KEY=… GEMINI_API_KEY=…")
	}
	keys := map[string]string{"openai": os.Getenv("OPENAI_API_KEY"), "anthropic": os.Getenv("ANTHROPIC_API_KEY"), "gemini": os.Getenv("GEMINI_API_KEY")}
	for prov, key := range keys {
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
		reg := llm.NewRegistry(nil, env, nil, newDiscardLog())
		if err := reg.Load(context.Background()); err != nil {
			t.Fatal(err)
		}
		gw := llm.New(llm.Options{Registry: reg, Log: newDiscardLog()})
		for _, name := range contractSchemas {
			schema, _ := os.ReadFile(filepath.Join("testdata", "schemas", name+".json"))
			out, err := gw.Structured(context.Background(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("Mẫu " + name + ": hãy điền dữ liệu minh hoạ hợp lý.")}, schema)
			if err != nil {
				t.Fatalf("%s/%s: %v", prov, name, err)
			}
			dir := filepath.Join("testdata", "replay", prov)
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name+".json"), out, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
