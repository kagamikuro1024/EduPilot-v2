package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TestRegressCLIRefusesExport — `export` mặc định TỪ CHỐI (REGRESSION_EXPORT_ALLOWED vắng / false / sai): thoát ≠ 0, không tạo tệp, không chạm DB / mạng (getenv không có DATABASE_URL vẫn dừng đúng chỗ).
func TestRegressCLIRefusesExport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "ca.jsonl")
	for _, env := range []map[string]string{{}, {"REGRESSION_EXPORT_ALLOWED": "false"}, {"REGRESSION_EXPORT_ALLOWED": "1"}, {"REGRESSION_EXPORT_ALLOWED": "yes"}} {
		var so, se bytes.Buffer
		code := run([]string{"export", "--course", "00000000-0000-7000-8000-000000000001", "--out", out}, func(k string) string { return env[k] }, &so, &se)
		require.Equal(t, 3, code, "%v", env)
		require.Contains(t, se.String(), "REGRESSION_EXPORT_ALLOWED")
		require.NoFileExists(t, out)
	}
	// cho phép nhưng --out nằm trong repo → từ chối trước khi mở DB
	wd, _ := os.Getwd()
	var so, se bytes.Buffer
	code := run([]string{"export", "--course", "00000000-0000-7000-8000-000000000001", "--out", filepath.Join(wd, "ca.jsonl")}, func(k string) string {
		if k == "REGRESSION_EXPORT_ALLOWED" {
			return "true"
		}
		return ""
	}, &so, &se)
	require.Equal(t, 2, code)
	require.True(t, strings.Contains(se.String(), "NGOÀI"))
	require.NoFileExists(t, filepath.Join(wd, "ca.jsonl"))
}

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// TestRegressCompare — MAE / lệch có dấu / ca xấu đi bằng decimal; «Có lùi» khi MAE tăng quá MaxRel hoặc |lệch TB| xấu thêm > 0,20.
func TestRegressCompare(t *testing.T) {
	// teacher 1,0: AI cũ 2,0 (lệch +1), AI mới 1,5 (+0,5); ca hai: teacher 2,0, AI cũ 2,0, AI mới 1,0 (xấu đi)
	r := compare([]pair{{old: d("2"), now: d("1.5"), teacher: d("1")}, {old: d("2"), now: d("1"), teacher: d("2")}}, 1)
	require.Equal(t, Report{N: 2, MAE: "0.75", Bias: "-0.25", OldMAE: "0.50", Worse: 1, Better: 1, Flipped: 1, Skipped: 1}, r)
	base := Report{N: 2, MAE: "0.70", Bias: "0.00"}
	bad, _ := regressed(r, base, 0.10)
	require.True(t, bad, "0,75 > 0,70 × 1,10 = 0,77? không: 0,75 ≤ 0,77 nhưng |lệch TB| 0,25 > 0,20")
	bad, _ = regressed(Report{MAE: "0.77", Bias: "0.10"}, base, 0.10)
	require.False(t, bad, "đúng ngưỡng không phải lùi")
	bad, _ = regressed(Report{MAE: "0.78", Bias: "0.00"}, base, 0.10)
	require.True(t, bad)
	require.Equal(t, Report{MAE: "0.00", Bias: "0.00", OldMAE: "0.00"}, compare(nil, 0))
}

// TestRegressFlippedBetter — US-GD-07 AC3/AC4: «tốt lên» = AI mới gần giảng viên hơn lúc đó; «vốn đúng nay sai» = |Δ cũ| ≤ 0,25 và |Δ mới| > 0,5 (đúng 0,5 chưa tính) → «Có lùi» dù MAE còn trong ngưỡng.
func TestRegressFlippedBetter(t *testing.T) {
	r := compare([]pair{
		{old: d("1.1"), now: d("1.6"), teacher: d("1")}, // 0,1 → 0,6: vốn đúng nay sai
		{old: d("1.0"), now: d("1.5"), teacher: d("1")}, // 0 → 0,5: chưa tính
		{old: d("3.0"), now: d("2.0"), teacher: d("1")}, // 2 → 1: tốt lên
		{old: d("1.0"), now: d("1.0"), teacher: d("1")}, // không đổi
		{old: d("2.0"), now: d("2.6"), teacher: d("1")}, // 1 → 1,6: xấu đi, nhưng vốn đã sai
	}, 0)
	require.Equal(t, 1, r.Flipped)
	require.Equal(t, 1, r.Better)
	require.Equal(t, 3, r.Worse)
	base := Report{MAE: "9.00", Bias: "0.00"} // MAE rất rộng: chỉ ca «vốn đúng nay sai» bắt được
	bad, why := regressed(Report{MAE: "0.50", Bias: "0.00", Flipped: 1}, base, 0.10)
	require.True(t, bad)
	require.Contains(t, why, "vốn đúng")
	bad, _ = regressed(Report{MAE: "0.50", Bias: "0.00", Flipped: 0}, base, 0.10)
	require.False(t, bad)
}

// TestRegressRunNeedsMeasureEnv — US-GD-07 AC3: `run` chỉ chạy ở môi trường đo (REGRESSION_RUN_ALLOWED=true); thiếu / sai → thoát 3, không chạm DB.
func TestRegressRunNeedsMeasureEnv(t *testing.T) {
	for _, env := range []map[string]string{{}, {"REGRESSION_RUN_ALLOWED": "false"}, {"REGRESSION_RUN_ALLOWED": "1"}} {
		var so, se bytes.Buffer
		code := run([]string{"run", "--assignment", "00000000-0000-7000-8000-000000000001"}, func(k string) string { return env[k] }, &so, &se)
		require.Equal(t, 3, code, "%v", env)
		require.Contains(t, se.String(), "REGRESSION_RUN_ALLOWED")
	}
	var so, se bytes.Buffer
	code := run([]string{"run"}, func(k string) string { return map[string]string{"REGRESSION_RUN_ALLOWED": "true"}[k] }, &so, &se)
	require.Equal(t, 2, code)
	require.Contains(t, se.String(), "--assignment")
}

// TestBaselineFileParses — benchmarks/guidance/baseline.json (US-GD-07 AC4) đọc được thành Report với MAE / lệch là decimal hợp lệ: `regress run --baseline` không chết vì tệp hỏng.
func TestBaselineFileParses(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "benchmarks", "guidance", "baseline.json"))
	require.NoError(t, err)
	var b Report
	require.NoError(t, json.Unmarshal(raw, &b))
	require.Positive(t, b.N)
	require.NotPanics(t, func() { regressed(Report{MAE: "0.00", Bias: "0.00"}, b, 0.10) })
}
