package provider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

func TestQC3_OC52_embedReal(t *testing.T) {
	k := realKey(t)
	p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: "https://opencode.ai/inference/go/openai/v1", APIKey: k})
	ctx, c := context.WithTimeout(t.Context(), 40*time.Second)
	defer c()
	t0 := time.Now()
	v, tok, err := p.Embed(ctx, provider.EmbedOpts{Model: "text-embedding-3-small", Inputs: []string{"chỉ mục B-tree", "cầu dao nhà cung cấp LLM"}, Dims: 1536})
	if err != nil || len(v) != 2 || len(v[0]) != 1536 || len(v[1]) != 1536 {
		t.Fatalf("err=%v len=%d", err, len(v))
	}
	var nz int
	for _, x := range v[0] {
		if x != 0 {
			nz++
		}
	}
	t.Logf("OK %v: 2 vectơ × 1536 chiều, ô khác 0 ở vectơ 1 = %d, tokens=%d", time.Since(t0).Round(time.Millisecond), nz, tok)
}

func TestQC3_OC53_embedWrongBase(t *testing.T) {
	k := realKey(t)
	p := provider.NewOpenAI(provider.OpenAIConfig{Type: "openai_compatible", BaseURL: ocBase, APIKey: k})
	ctx, c := context.WithTimeout(t.Context(), 30*time.Second)
	defer c()
	t0 := time.Now()
	_, _, err := p.Embed(ctx, provider.EmbedOpts{Model: "text-embedding-3-small", Inputs: []string{"x"}, Dims: 1536})
	var pe *provider.Error
	if !errors.As(err, &pe) || time.Since(t0) > 30*time.Second {
		t.Fatalf("err=%v", err)
	}
	t.Logf("OK /inference/go/v1 + Embed → Kind=%s status=%d sau %v, detail=%.120q", pe.Kind, pe.Status, time.Since(t0).Round(time.Millisecond), pe.Detail)
}
