// Package cost: chi phí ước tính của một lời gọi LLM (VND), tính bằng decimal — cấm float (luật 5, US-P1-01 AC12).
package cost

import "github.com/shopspring/decimal"

// Of trả tokensIn × priceIn ÷ 1.000.000 + tokensOut × priceOut ÷ 1.000.000 (giá là đ / 1 triệu token).
func Of(tokensIn, tokensOut int, priceIn, priceOut decimal.Decimal) decimal.Decimal {
	in := decimal.NewFromInt(int64(tokensIn)).Mul(priceIn)
	out := decimal.NewFromInt(int64(tokensOut)).Mul(priceOut)
	return in.Add(out).Div(decimal.NewFromInt(1_000_000))
}

// Units đổi VND sang số nguyên 1/10.000 đ (đơn vị bộ đếm ngân sách Redis — INCRBY), làm tròn lên để không bao giờ đếm thiếu.
func Units(vnd decimal.Decimal) int64 {
	return vnd.Mul(decimal.NewFromInt(10_000)).Ceil().IntPart()
}

// FromUnits đổi ngược số nguyên 1/10.000 đ về VND.
func FromUnits(u int64) decimal.Decimal {
	return decimal.NewFromInt(u).Div(decimal.NewFromInt(10_000))
}
