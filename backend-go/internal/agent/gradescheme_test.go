package agent

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type noScheme struct{}

func (noScheme) Scheme(context.Context, TrustedContext) (Facts, bool, error) { return nil, false, nil }
func (noScheme) Summary(context.Context, TrustedContext) (Facts, bool, error) {
	return nil, false, ErrNoScheme
}
func (noScheme) WhatIf(context.Context, TrustedContext, map[string]decimal.Decimal) (Facts, bool, error) {
	return nil, false, ErrNoScheme
}

func noSchemeRig(t *testing.T) *rig {
	t.Helper()
	r := newRig(t, true, 3)
	r.a.Schemes = noScheme{}
	r.a.Private = DefaultPrivateRegistry(r.spy, r.spy, noScheme{}, r.spy, r.spy)
	return r
}

// AC7: hỏi cách tính điểm khi lớp chưa có công thức đã xác nhận → câu mẫu, cờ NoFormula (để tự mở ticket), không gọi LLM.
func TestGradeFormulaNoScheme(t *testing.T) {
	t.Parallel()
	r := noSchemeRig(t)
	out := r.respond(t, "Cách tính điểm môn này ra sao")
	require.Equal(t, ReplyNoFormula, out.Canned)
	require.True(t, out.NoFormula)
	require.Nil(t, out.Stream)
	require.Empty(t, out.Blocks)
	require.Zero(t, r.g.calls.Load())
}

// AC7: điểm cá nhân khi chưa có công thức → cùng câu mẫu, KHÔNG cờ NoFormula (không tự mở ticket), không số điểm.
func TestPersonalGradeNoScheme(t *testing.T) {
	t.Parallel()
	r := noSchemeRig(t)
	out := r.respond(t, "Điểm giữa kỳ của em là bao nhiêu")
	require.Equal(t, ReplyNoFormula, out.Canned)
	require.False(t, out.NoFormula)
	require.Empty(t, out.Blocks)
	require.Zero(t, r.g.calls.Load())
}

func TestWhatIfNoScheme(t *testing.T) {
	t.Parallel()
	r := noSchemeRig(t)
	out := r.respond(t, "Nếu cuối kỳ em được 8 thì tổng kết")
	require.Equal(t, ReplyNoFormula, out.Canned)
	require.False(t, out.NoFormula)
	require.Empty(t, out.Blocks)
	require.Zero(t, r.g.calls.Load())
}
