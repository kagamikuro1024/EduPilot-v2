package llmconfig_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llmconfig"
)

func decModel(name string) llmconfig.ModelInput {
	return llmconfig.ModelInput{Model: name, Kind: "decision", PriceIn: decimal.RequireFromString("2600")}
}

// Migration 00020 seed đúng SRS FEAT-decision-model 5: một nhà `decision` bật, hai mô hình (giả bật, thật tắt), ba cổng tắt với ngưỡng / timeout mặc định.
func TestDecisionSeed(t *testing.T) {
	r := newRig(t)
	var provs, enabledModels int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from llm_providers where type='decision' and enabled`).Scan(&provs))
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from llm_models where kind='decision' and enabled and model='decision-fake'`).Scan(&enabledModels))
	require.Equal(t, 1, provs)
	require.Equal(t, 1, enabledModels)
	rows, err := r.pool.Query(t.Context(), `select gate, enabled, threshold::text, timeout_ms from decision_gates order by gate`)
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var g, th string
		var en bool
		var to int
		require.NoError(t, rows.Scan(&g, &en, &th, &to))
		require.False(t, en)
		got = append(got, g+" "+th)
		_ = to
	}
	require.Equal(t, []string{"CHAT_ROUTING 0.950", "INJECTION_FLAG 0.500", "THREADS_PRECHECK 0.050"}, got)
}

// TestDecisionProviderRules: một nhà decision bật; mô hình thật bị chặn khi ALLOW_REAL tắt (409); loại mô hình phải khớp loại nhà.
func TestDecisionProviderRules(t *testing.T) {
	r := newRig(t)
	_, err := r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "decision", Name: "Thứ hai", Models: []llmconfig.ModelInput{decModel("decision-fake")}})
	var inv *llmconfig.ErrInvalid
	require.True(t, errors.As(err, &inv), "%v", err)
	require.Equal(t, "DECISION_ALREADY_ENABLED", inv.Code)

	off := false
	on := true
	real := decModel(llmconfig.RealDecisionModel)
	real.Enabled = &on
	_, err = r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "decision", Name: "Thật", Enabled: &off, Models: []llmconfig.ModelInput{real}})
	require.ErrorIs(t, err, llmconfig.ErrDecisionRealDisabled)
	// mô hình thật đang TẮT thì lưu được (seed có sẵn; UI «Thêm» gửi cả hai mô hình)
	real.Enabled = &off
	_, err = r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "decision", Name: "Thật tắt", Enabled: &off, Models: []llmconfig.ModelInput{decModel("decision-fake"), real}})
	require.NoError(t, err)

	_, err = r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "fake", Name: "Chat sai", Models: []llmconfig.ModelInput{decModel("m")}})
	require.True(t, errors.As(err, &inv), "%v", err)
	require.Equal(t, "INVALID_KIND", inv.Code)
	_, err = r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "decision", Name: "Kind sai", Enabled: &off, Models: []llmconfig.ModelInput{chatModel("m")}})
	require.True(t, errors.As(err, &inv), "%v", err)
}
