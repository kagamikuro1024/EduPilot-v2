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

// Score: `raw = Σ earned`, `total = Σ points` (câu void tính earned = points), `score = RoundToStep(raw ÷ total × maxScore, step)`, kẹp [0, maxScore].
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
	return RoundToStep(raw.DivRound(total, precision).Mul(maxScore), step, maxScore), nil
}

// RoundToStep làm tròn `v` về bội của `step` (nửa lên) rồi kẹp vào [0, maxScore]. Bước ≤ 0 → chỉ kẹp.
func RoundToStep(v, step, maxScore decimal.Decimal) decimal.Decimal {
	if step.Sign() > 0 {
		v = v.DivRound(step, precision).Round(0).Mul(step)
	}
	if v.Sign() < 0 {
		return decimal.Zero
	}
	if v.GreaterThan(maxScore) {
		return maxScore
	}
	return v
}
