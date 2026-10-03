package llm_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

const classifySchema = `{"type":"object","properties":{"label":{"type":"string","enum":["hoi_dap","khieu_nai"]},"confidence":{"type":"number","minimum":0,"maximum":1}},"required":["label","confidence"],"additionalProperties":false}`

func TestValidateJSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, data string
		ok         bool
	}{
		{"hợp lệ", `{"label":"hoi_dap","confidence":0.5}`, true},
		{"thiếu trường bắt buộc", `{"label":"hoi_dap"}`, false},
		{"trường thừa (additionalProperties=false)", `{"label":"hoi_dap","confidence":1,"x":1}`, false},
		{"enum sai", `{"label":"khac","confidence":1}`, false},
		{"kiểu sai", `{"label":"hoi_dap","confidence":"cao"}`, false},
		{"vượt maximum", `{"label":"hoi_dap","confidence":1.5}`, false},
		{"không phải JSON", `xin chào`, false},
		{"mảng thay vì object", `[1]`, false},
	}
	for _, tc := range cases {
		err := llm.ValidateJSON(json.RawMessage(classifySchema), json.RawMessage(tc.data))
		if (err == nil) != tc.ok {
			t.Errorf("%s: err=%v", tc.name, err)
		}
	}
}

func TestStructuredJSONSchema(t *testing.T) {
	t.Parallel()
	a := &stub{name: "A", chat: func(_ int, o provider.ChatOpts) (provider.Result, error) {
		if len(o.Schema) == 0 {
			return provider.Result{}, perr(provider.KindBadRequest, 400)
		}
		return provider.Result{Text: "```json\n{\"label\":\"hoi_dap\",\"confidence\":0.9}\n```", TokensIn: 5, TokensOut: 5}, nil
	}}
	g, _ := newGateway(t, a)
	out, err := g.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("hỏi")}, json.RawMessage(classifySchema))
	if err != nil {
		t.Fatal(err)
	}
	if llm.ValidateJSON(json.RawMessage(classifySchema), out) != nil || strings.Contains(string(out), "```") {
		t.Errorf("out = %s", out)
	}
	if a.calls.Load() != 1 {
		t.Errorf("Structured phải là MỘT lời gọi (D47), có %d", a.calls.Load())
	}
}

func TestStructuredInvalidFallsBack(t *testing.T) {
	t.Parallel()
	wrong := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{Text: `{"label":"khac","confidence":2}`}, nil // sai schema ở mọi lần
	}}
	right := &stub{name: "B", chat: func(int, provider.ChatOpts) (provider.Result, error) {
		return provider.Result{Text: `{"label":"khieu_nai","confidence":0.1}`}, nil
	}}
	g, cap := newGateway(t, wrong, right)
	out, err := g.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("hỏi")}, json.RawMessage(classifySchema))
	if err != nil || !strings.Contains(string(out), "khieu_nai") {
		t.Fatalf("out=%s err=%v", out, err)
	}
	if wrong.calls.Load() != 1 {
		t.Errorf("BAD_RESPONSE không thử lại cùng nhà: %d lần", wrong.calls.Load())
	}
	if rows := cap.all(t, g); len(rows) != 1 || rows[0].FallbackIndex != 1 {
		t.Errorf("audit = %+v", rows)
	}

	// mọi nhà đều sai schema → lỗi, KHÔNG BAO GIỜ trả JSON sai schema
	g2, _ := newGateway(t, wrong)
	if out, err := g2.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("hỏi")}, json.RawMessage(classifySchema)); !errors.Is(err, llm.ErrAllProvidersFailed) || out != nil {
		t.Errorf("out=%s err=%v", out, err)
	}
	// schema hỏng → lỗi cục bộ, không gọi mạng
	fresh := &stub{name: "F"}
	g3, _ := newGateway(t, fresh)
	if _, err := g3.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("hỏi")}, json.RawMessage(`{không phải json`)); !errors.Is(err, llm.ErrBadRequest) || fresh.calls.Load() != 0 {
		t.Errorf("schema hỏng: err=%v calls=%d", err, fresh.calls.Load())
	}
}

func TestEmbedDims(t *testing.T) {
	t.Parallel()
	wrong := &stub{name: "A", embed: func(_ int, o provider.EmbedOpts) ([][]float32, int, error) {
		out := make([][]float32, len(o.Inputs))
		for i := range out {
			out[i] = make([]float32, 768)
		}
		return out, 1, nil
	}}
	next := &stub{name: "B"}
	g, cap := newGateway(t, wrong, next)
	vecs, err := g.Embed(t.Context(), llm.EmbedRequest{Inputs: []string{"x"}})
	var dm *llm.ErrDimsMismatch
	if !errors.As(err, &dm) || dm.Actual != 768 || dm.Expected != 1536 || vecs != nil {
		t.Fatalf("vecs=%d err=%v", len(vecs), err)
	}
	if wrong.calls.Load() != 1 || next.calls.Load() != 0 {
		t.Errorf("không thử lại / không dự phòng: A=%d B=%d", wrong.calls.Load(), next.calls.Load())
	}
	if rows := cap.all(t, g); len(rows) != 1 || rows[0].ErrorKind != "DIMS_MISMATCH" {
		t.Errorf("audit = %+v", rows)
	}
}

func TestEmbedBatchSplit(t *testing.T) {
	t.Parallel()
	var sizes []int
	p := &stub{name: "A", embed: func(_ int, o provider.EmbedOpts) ([][]float32, int, error) {
		sizes = append(sizes, len(o.Inputs))
		if o.Dims != 1536 {
			t.Errorf("dimensions = %d, muốn 1536", o.Dims)
		}
		out := make([][]float32, len(o.Inputs))
		for i := range out {
			out[i] = make([]float32, 1536)
			out[i][0] = float32(len(sizes)) // đánh dấu lô
		}
		return out, len(o.Inputs), nil
	}}
	g, _ := newGateway(t, p)
	in := make([]string, 250)
	for i := range in {
		in[i] = "chuỗi"
	}
	vecs, err := g.Embed(t.Context(), llm.EmbedRequest{Inputs: in})
	if err != nil || len(vecs) != 250 {
		t.Fatalf("len=%d err=%v", len(vecs), err)
	}
	if len(sizes) != 3 || sizes[0] != 100 || sizes[1] != 100 || sizes[2] != 50 {
		t.Errorf("lô = %v, muốn [100 100 50]", sizes)
	}
	if vecs[0][0] != 1 || vecs[100][0] != 2 || vecs[249][0] != 3 {
		t.Errorf("thứ tự vectơ sai")
	}
}

func TestEmbedEmptyInput(t *testing.T) {
	t.Parallel()
	p := &stub{name: "A"}
	g, cap := newGateway(t, p)
	for _, in := range [][]string{nil, {}, {"ok", ""}, {"   "}} {
		if _, err := g.Embed(t.Context(), llm.EmbedRequest{Inputs: in}); !errors.Is(err, llm.ErrBadRequest) {
			t.Errorf("%q: err = %v", in, err)
		}
	}
	if p.calls.Load() != 0 {
		t.Error("không được gọi mạng khi đầu vào rỗng")
	}
	if rows := cap.all(t, g); len(rows) != 4 || rows[0].ErrorKind != "BAD_REQUEST" {
		t.Errorf("audit = %d dòng", len(rows))
	}
}
