package main

import (
	"bytes"
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
	require.Equal(t, Report{N: 2, MAE: "0.75", Bias: "-0.25", OldMAE: "0.50", Worse: 1, Skipped: 1}, r)
	base := Report{N: 2, MAE: "0.70", Bias: "0.00"}
	bad, _ := regressed(r, base, 0.10)
	require.True(t, bad, "0,75 > 0,70 × 1,10 = 0,77? không: 0,75 ≤ 0,77 nhưng |lệch TB| 0,25 > 0,20")
	bad, _ = regressed(Report{MAE: "0.77", Bias: "0.10"}, base, 0.10)
	require.False(t, bad, "đúng ngưỡng không phải lùi")
	bad, _ = regressed(Report{MAE: "0.78", Bias: "0.00"}, base, 0.10)
	require.True(t, bad)
	require.Equal(t, Report{MAE: "0.00", Bias: "0.00", OldMAE: "0.00"}, compare(nil, 0))
}
