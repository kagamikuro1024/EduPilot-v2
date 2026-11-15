package llm_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

// TestPayloadStoresMaskedOnly — US-P10-02 AC10: dòng audit mang bản prompt ĐÃ CHE (cả ba đường Chat / Stream / Structured), kèm tên công cụ;
// không còn tên / MSSV / email gốc; che tắt hoặc embedding thì KHÔNG lưu gì (không bao giờ lưu bản gốc).
func TestPayloadStoresMaskedOnly(t *testing.T) {
	t.Parallel()
	p := &stub{name: "p", chat: func(_ int, o provider.ChatOpts) (provider.Result, error) {
		if len(o.Schema) > 0 {
			return provider.Result{Text: `{"answer":"ok"}`, TokensIn: 1, TokensOut: 1}, nil
		}
		return provider.Result{Text: "ok", TokensIn: 1, TokensOut: 1}, nil
	}}
	g, cap := maskGateway(t, newDiscardLog(), p)
	ctx := withCourse(t.Context())
	msgs := []llm.Message{
		{Role: "system", Content: "Bạn là trợ giảng."},
		{Role: "user", Content: "Em là " + leakName + " mssv " + leakMSSV + " mail " + leakEmail},
	}
	tools := []string{"attendance", "grade"}
	_, err := g.Chat(ctx, llm.Request{Task: llm.TaskChat, Messages: msgs, Tools: tools})
	require.NoError(t, err)
	ch, err := g.Stream(ctx, llm.Request{Task: llm.TaskChat, Messages: msgs, Tools: tools})
	require.NoError(t, err)
	_, _, _ = drain(t, ch)
	_, err = g.Structured(ctx, llm.Request{Task: llm.TaskUtility, Messages: msgs, Tools: tools}, json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"]}`))
	require.NoError(t, err)
	_, err = g.Embed(ctx, llm.EmbedRequest{Inputs: []string{"câu hỏi của " + leakName}})
	require.NoError(t, err)

	rows := cap.all(t, g)
	require.Len(t, rows, 4)
	stored := 0
	for _, r := range rows {
		if r.Task == string(llm.TaskEmbedding) {
			require.Empty(t, r.Prompt, "embedding không lưu payload")
			continue
		}
		stored++
		require.NotEmpty(t, r.Prompt)
		require.False(t, leaks(r.Prompt), "payload còn dữ liệu gốc: %q", r.Prompt)
		require.Contains(t, r.Prompt, "[user]")
		require.Equal(t, tools, r.Tools)
	}
	require.Equal(t, 3, stored)

	// che tắt (test / nomask): không lưu bản gốc
	g2, cap2 := newGateway(t, p)
	_, err = g2.Chat(withCourse(t.Context()), llm.Request{Task: llm.TaskChat, Messages: msgs})
	require.NoError(t, err)
	for _, r := range cap2.all(t, g2) {
		require.Empty(t, r.Prompt, "che tắt thì không được lưu bản gốc")
	}
}
