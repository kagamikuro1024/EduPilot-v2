package llm_test

// Throwaway QC (qc3) — P-11-110 structured_invalid. Đặt tạm vào backend-go/internal/llm.
import (
	"encoding/json"
	"strings"
	"testing"
	"log/slog"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

func siRun(t *testing.T, text string, tokens, cap int) string {
	t.Helper()
	var buf syncBuf
	log := slog.New(slog.NewTextHandler(&buf, nil))
	wrong := &stub{name: "A", chat: func(int, provider.ChatOpts) (provider.Result, error) { return provider.Result{Text: text, TokensOut: tokens}, nil }}
	g, _ := newGatewayLog(t, log, wrong)
	_, _ = g.Structured(t.Context(), llm.Request{Task: llm.TaskClassify, Messages: userMsg("hỏi"), Params: llm.Params{MaxTokens: cap}}, json.RawMessage(classifySchema))
	return buf.String()
}

func TestQC3_OC70_structuredInvalidKinds(t *testing.T) {
	const secret = "BIMAT123"
	cases := []struct{ name, text string; tokens, cap int }{
		{"không phải JSON (chữ thường)", secret + " không phải json", 50, 1024},
		{"JSON cụt ở trần", `{"label":"hoi_dap","confidence":0.`, 1024, 1024},
		{"sai enum", `{"label":"` + secret + `","confidence":0.5}`, 20, 1024},
		{"sai kiểu", `{"label":"hoi_dap","confidence":"` + secret + `"}`, 20, 1024},
		{"thừa thuộc tính", `{"label":"hoi_dap","confidence":0.5,"` + secret + `":1}`, 20, 1024},
		{"thiếu thuộc tính", `{"label":"hoi_dap"}`, 8, 1024},
		{"rỗng", ``, 0, 1024},
	}
	for _, c := range cases {
		out := siRun(t, c.text, c.tokens, c.cap)
		lines := strings.Count(out, "structured_invalid")
		hasSecret := strings.Contains(out, secret)
		first := ""
		if i := strings.Index(c.text, " "); strings.HasPrefix(c.text, secret) && i > 0 {
			first = c.text[:1]
		}
		t.Logf("[%s] dòng structured_invalid=%d | lộ %q: %v | log=%s", c.name, lines, secret, hasSecret, strings.TrimSpace(out))
		if lines != 1 {
			t.Errorf("[%s] số dòng WARN = %d, muốn 1", c.name, lines)
		}
		if hasSecret {
			t.Errorf("[%s] log chứa nội dung đầu ra", c.name)
		}
		_ = first
	}
}

func TestQC3_OC71_validNoLog(t *testing.T) {
	out := siRun(t, `{"label":"hoi_dap","confidence":0.5}`, 12, 1024)
	if strings.Contains(out, "structured_invalid") {
		t.Fatalf("đầu ra hợp lệ mà vẫn ghi WARN: %s", out)
	}
	t.Logf("OK đầu ra hợp lệ → không có dòng structured_invalid (log=%q)", strings.TrimSpace(out))
}
