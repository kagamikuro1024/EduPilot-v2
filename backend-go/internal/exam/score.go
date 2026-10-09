// Package exam: nghiệp vụ bài thi hằng tuần (SRS FEAT-weekly-exam). Tính điểm là code thuần (`shopspring/decimal`, cấm float);
// chỉ `suggest.go` được import `internal/llm` (gợi ý nháp — US-PE-03), kiểm bằng TestOnlySuggestImportsLLM.
package exam

import (
	"errors"

	"github.com/shopspring/decimal"
)

// ScoreItem là một mục của bài đã có điểm đạt được. `Void` = câu bị vô hiệu (đổi đáp án / bỏ câu): mọi lượt nhận đủ điểm kể cả bỏ trống.
type ScoreItem struct {
	Points decimal.Decimal
	Earned decimal.Decimal
	Void   bool
}

// ErrNoPoints: tổng điểm các mục bằng 0 (không chia được).
var ErrNoPoints = errors.New("exam: tổng điểm các mục bằng 0")

// precision là số chữ số thập phân của phép chia trung gian (DivRound).
const precision = 16

// CodeEarned là điểm câu code: `points × Σweight_đạt ÷ Σweight_tất_cả` (DivRound 16). Tổng trọng số 0 → 0 (service chặn từ lúc duyệt: TOTAL_WEIGHT_ZERO).
func CodeEarned(points decimal.Decimal, passedWeight, totalWeight int64) decimal.Decimal {
	if totalWeight <= 0 || passedWeight <= 0 {
		return decimal.Zero
	}
	if passedWeight > totalWeight {
		passedWeight = totalWeight
	}
	return points.Mul(decimal.NewFromInt(passedWeight)).DivRound(decimal.NewFromInt(totalWeight), precision)
}

// Score: `raw = Σ earned`, `total = Σ points` (câu void tính earned = points), `score = RoundToStep(raw × maxScore ÷ total, step)`, kẹp [0, maxScore].
// Làm tròn đúng MỘT lần ở cuối: tử số `raw × maxScore` nhân trước, rồi chia THẲNG cho `total × step` (hoặc cho `total` khi không có bước) — không có
// phép chia trung gian nào bị cắt, nên điểm nằm đúng giữa hai bước (ví dụ max 4.5, bước 0.25) luôn làm tròn lên đúng nửa.
func Score(items []ScoreItem, maxScore, step decimal.Decimal) (decimal.Decimal, error) {
	raw, total := decimal.Zero, decimal.Zero
	for _, it := range items {
		total = total.Add(it.Points)
		if it.Void {
			raw = raw.Add(it.Points)
		} else {
			raw = raw.Add(it.Earned)
		}
	}
	if total.Sign() <= 0 {
		return decimal.Zero, ErrNoPoints
	}
	num := raw.Mul(maxScore)
	var v decimal.Decimal
	if step.Sign() > 0 {
		v = num.DivRound(total.Mul(step), 0).Mul(step) // số bước nguyên gần nhất, nửa lên (raw ≥ 0)
	} else {
		v = num.DivRound(total, precision)
	}
	return clamp(v, maxScore), nil
}

// RoundToStep làm tròn `v` về bội của `step` (nửa lên) rồi kẹp vào [0, maxScore]. Bước ≤ 0 → chỉ kẹp. Dùng cho điểm đã tính sẵn (điều chỉnh tay);
// điểm bài thi đi qua Score (không chia trung gian).
func RoundToStep(v, step, maxScore decimal.Decimal) decimal.Decimal {
	if step.Sign() > 0 {
		v = v.DivRound(step, 0).Mul(step)
	}
	return clamp(v, maxScore)
}

func clamp(v, maxScore decimal.Decimal) decimal.Decimal {
	if v.Sign() < 0 {
		return decimal.Zero
	}
	if v.GreaterThan(maxScore) {
		return maxScore
	}
	return v
}
