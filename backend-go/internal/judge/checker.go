package judge

import (
	"math"
	"strconv"
	"strings"
)

// CheckerKind là `checker_kind` của bài.
type CheckerKind string

const (
	Exact    CheckerKind = "EXACT"
	Tokens   CheckerKind = "TOKENS"
	FloatEps CheckerKind = "FLOAT_EPS"
)

// Check so đầu ra của chương trình với `expected` (SRS 4.5.5) — Go thuần, chạy trong worker. `eps` chỉ dùng cho FLOAT_EPS.
func Check(kind CheckerKind, eps float64, got, want string) bool {
	switch kind {
	case Tokens:
		return tokensEqual(strings.Fields(got), strings.Fields(want), nil)
	case FloatEps:
		return tokensEqual(strings.Fields(got), strings.Fields(want), &eps)
	default:
		return normalize(got) == normalize(want)
	}
}

// normalize: `\r\n` → `\n`, bỏ khoảng trắng / tab cuối MỖI dòng, bỏ dòng trống cuối tệp.
func normalize(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func tokensEqual(a, b []string, eps *float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		if eps == nil {
			return false
		}
		x, okx := finite(a[i])
		y, oky := finite(b[i])
		if !okx || !oky {
			return false
		}
		if math.Abs(x-y) > *eps*math.Max(1, math.Abs(y)) {
			return false
		}
	}
	return true
}

// finite chỉ nhận số thập phân HỮU HẠN (NaN / Inf / "infinity" so như chuỗi).
func finite(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}
