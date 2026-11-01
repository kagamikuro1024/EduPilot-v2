package grading

import (
	"time"

	"github.com/shopspring/decimal"
)

// LatePolicy là quy tắc nộp muộn HIỆN TẠI của bài tập (đọc lại mỗi lần chấm / sửa / công bố — TLR-8).
type LatePolicy struct {
	AllowLate bool
	PerDay    decimal.Decimal
	MaxDays   int
}

// LateDays: 0 nếu không có `submitted_at` (UPLOAD) hoặc nộp trước / đúng hạn; ngược lại ⌈(nộp − hạn) ÷ 24 giờ⌉.
func LateDays(submittedAt *time.Time, due time.Time) int {
	if submittedAt == nil || !submittedAt.After(due) {
		return 0
	}
	d := submittedAt.Sub(due)
	days := int(d / (24 * time.Hour))
	if d%(24*time.Hour) != 0 {
		days++
	}
	return days
}

// LatePenalty = min(perDay × min(lateDays, maxDays), sum); `allow_late = false` hoặc perDay = 0 → 0. Decimal, không làm tròn thêm (bước 0,05 × số nguyên ngày).
func LatePenalty(lateDays int, p LatePolicy, sum decimal.Decimal) decimal.Decimal {
	if !p.AllowLate || lateDays <= 0 || !p.PerDay.IsPositive() {
		return decimal.Zero
	}
	n := min(lateDays, p.MaxDays)
	return decimal.Min(p.PerDay.Mul(decimal.NewFromInt(int64(n))), sum)
}

// TotalAfterLate = max(0, sum − penalty).
func TotalAfterLate(sum, penalty decimal.Decimal) decimal.Decimal {
	return decimal.Max(decimal.Zero, sum.Sub(penalty))
}
