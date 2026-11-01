package grading_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
)

var due = time.Date(2026, 10, 30, 23, 59, 0, 0, time.UTC)

func TestLateDays(t *testing.T) {
	t.Parallel()
	at := func(d time.Duration) *time.Time { x := due.Add(d); return &x }
	require.Zero(t, grading.LateDays(nil, due), "UPLOAD không có submitted_at")
	require.Zero(t, grading.LateDays(at(-time.Hour), due))
	require.Zero(t, grading.LateDays(at(0), due))
	require.Equal(t, 1, grading.LateDays(at(time.Second), due))
	require.Equal(t, 1, grading.LateDays(at(24*time.Hour), due))
	require.Equal(t, 2, grading.LateDays(at(24*time.Hour+time.Second), due))
}

func TestLatePenaltyDecimal(t *testing.T) {
	t.Parallel()
	p := grading.LatePolicy{AllowLate: true, PerDay: decimal.RequireFromString("0.5"), MaxDays: 2}
	sum := decimal.RequireFromString("7")
	pen := grading.LatePenalty(1, p, sum)
	require.Equal(t, "0.5", pen.String(), "sinh viên B: 1 ngày × 0,5")
	require.Equal(t, "6.5", grading.TotalAfterLate(sum, pen).String())
}

func TestLatePenaltyCappedByMaxDays(t *testing.T) {
	t.Parallel()
	p := grading.LatePolicy{AllowLate: true, PerDay: decimal.RequireFromString("0.5"), MaxDays: 2}
	require.Equal(t, "1", grading.LatePenalty(5, p, decimal.NewFromInt(7)).String())
}

func TestLatePenaltyFloorZero(t *testing.T) {
	t.Parallel()
	p := grading.LatePolicy{AllowLate: true, PerDay: decimal.NewFromInt(5), MaxDays: 3}
	pen := grading.LatePenalty(3, p, decimal.RequireFromString("2"))
	require.Equal(t, "2", pen.String(), "khoản trừ không vượt tổng")
	require.True(t, grading.TotalAfterLate(decimal.RequireFromString("2"), pen).IsZero())
	require.True(t, grading.TotalAfterLate(decimal.NewFromInt(1), decimal.NewFromInt(3)).IsZero(), "điểm không âm")
}

func TestLatePenaltyZeroCases(t *testing.T) {
	t.Parallel()
	require.True(t, grading.LatePenalty(2, grading.LatePolicy{AllowLate: false, PerDay: decimal.NewFromInt(1), MaxDays: 2}, decimal.NewFromInt(7)).IsZero())
	require.True(t, grading.LatePenalty(2, grading.LatePolicy{AllowLate: true, MaxDays: 2}, decimal.NewFromInt(7)).IsZero())
	require.True(t, grading.LatePenalty(0, grading.LatePolicy{AllowLate: true, PerDay: decimal.NewFromInt(1), MaxDays: 2}, decimal.NewFromInt(7)).IsZero())
}
