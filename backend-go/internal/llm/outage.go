package llm

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// Mất nhà cung cấp (mạch mở / mọi nhà cung cấp lỗi) KHÔNG được tiêu lượt thử của việc nền (P-11-23, luật 12): consumer hoãn qua cửa sổ mở mạch rồi gọi lại.
const OutageMax = 20 // số lần hoãn tối đa mỗi việc (≈ 10 phút) trước khi tính lỗi thật

// OutageDelay > LLM_BREAKER_OPEN (30 s) để lần gọi lại rơi vào mạch bán mở. Biến để test rút ngắn.
var OutageDelay = 35 * time.Second //nolint:gochecknoglobals // chỉ test đổi

// IsOutage: không còn nhà cung cấp dùng được vì lỗi của họ (ngân sách hết KHÔNG tính).
func IsOutage(err error) bool {
	var af *allFailed
	var un *ErrUnavailable
	return (errors.As(err, &af) && af.outage) || (errors.As(err, &un) && un.Reason == ReasonAllFailed && un.Outage)
}

// IsBadOutput: mọi nhà cung cấp đã thử trả đầu ra sai định dạng / sai schema. Việc nền coi là lỗi vĩnh viễn, không thử lại (DEF-R8-A1).
func IsBadOutput(err error) bool {
	var af *allFailed
	var un *ErrUnavailable
	return (errors.As(err, &af) && af.badOutput) || (errors.As(err, &un) && un.Reason == ReasonAllFailed && un.BadOutput)
}

// unavailable chuyển ErrAllProvidersFailed của runChain thành ErrUnavailable giữ nguyên các cờ.
func unavailable(err error) *ErrUnavailable {
	return &ErrUnavailable{Reason: ReasonAllFailed, Outage: IsOutage(err), BadOutput: IsBadOutput(err)}
}

// OutageBackoff: OutageDelay + nhiễu 0–⅓ để cả lô không dồn vào cùng một giây.
func OutageBackoff(base time.Duration) time.Duration {
	return base + time.Duration(rand.Int64N(int64(base)/3+1)) //nolint:gosec // nhiễu giãn cách, không phải bí mật
}

// AwaitOutage: nếu err là mất nhà cung cấp và `*n` < OutageMax thì tăng *n, chờ (huỷ theo ctx) và trả true — người gọi gọi lại, KHÔNG tính lượt thử.
func AwaitOutage(ctx context.Context, err error, n *int) bool {
	if !IsOutage(err) || *n >= OutageMax || ctx.Err() != nil {
		return false
	}
	*n++
	t := time.NewTimer(OutageBackoff(OutageDelay))
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
