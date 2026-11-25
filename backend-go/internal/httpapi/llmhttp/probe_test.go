package llmhttp

import (
	"context"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// slowChat trả lời sau d (tôn trọng ctx).
type slowChat struct{ d time.Duration }

func (s slowChat) Chat(ctx context.Context, _ provider.ChatOpts) (provider.Result, error) {
	select {
	case <-time.After(s.d):
		return provider.Result{Text: "ok"}, nil
	case <-ctx.Done():
		return provider.Result{}, ctx.Err()
	}
}
func (slowChat) Stream(context.Context, provider.ChatOpts) (<-chan provider.Delta, error) {
	return nil, nil
}
func (slowChat) Embed(context.Context, provider.EmbedOpts) ([][]float32, int, error) {
	return nil, 0, nil
}

// P-11-108: hạn của một lần Test là probeTimeout (30 s), KHÔNG bị cắt sớm bởi REQUEST_TIMEOUT (cũng 30 s, đã chạy từ trước lúc probe bắt đầu):
// nếu thừa kế hạn của request thì nhà cung cấp chậm 29 s vẫn bị báo TIMEOUT / 504.
func TestProbeNotCutByRequestDeadline(t *testing.T) {
	t.Parallel()
	reg := llm.NewRegistry(nil, llm.EnvConfig{}, func(llm.ProviderSpec) (provider.Provider, error) { return slowChat{d: 300 * time.Millisecond}, nil }, nil)
	a := &Admin{RT: &llmrt.Runtime{Registry: reg}, Clock: clock.Real{}}
	reqCtx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond) // hạn của request ngắn hơn lúc nhà cung cấp trả lời
	defer cancel()
	out := a.probe(reqCtx, "openai_compatible", nil, "k", candidate{model: "m", kind: "chat"})
	if !out.OK {
		t.Fatalf("probe bị cắt theo hạn của request: %+v", out)
	}
}
