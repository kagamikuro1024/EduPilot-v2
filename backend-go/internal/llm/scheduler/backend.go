package scheduler

import (
	"context"
	"time"
)

// laneChar mã hoá làn trong phần tử ZSET: I / N / B (để đếm BATCH đang chạy mà không cần khoá phụ).
const (
	laneI = "I"
	laneN = "N"
	laneB = "B"
)

// leaseReq là yêu cầu một chỗ gọi nhà cung cấp.
type leaseReq struct {
	provider string
	lane     string // laneI | laneN | laneB
	member   string // laneChar + ":" + req_id
	max      int
	batchCap int
	lease    time.Duration
}

// bucketReq là yêu cầu token RPM + TPM. cost = ước tính token.
type bucketReq struct {
	provider string
	rpm, tpm int
	cost     int
}

// backend là phần trạng thái dùng chung: chỗ đồng thời, token bucket, mạch. Hai bản cài: Redis (toàn cục) và cục bộ (Redis mất).
type backend interface {
	tryLease(ctx context.Context, r leaseReq) (bool, error)
	release(ctx context.Context, provider, member string) error
	// tryBucket trừ token nếu đủ cả hai bucket; không đủ → (false, thời gian chờ gợi ý).
	tryBucket(ctx context.Context, r bucketReq) (bool, time.Duration, error)
	// reconcile cộng (diff > 0: hoàn) hoặc trừ (diff < 0) token TPM sau khi biết số thật.
	reconcile(ctx context.Context, provider string, tpm, diff int) error
	cbAllow(ctx context.Context, provider string) (bool, error)
	cbReport(ctx context.Context, provider string, failure bool) error
	cbState(ctx context.Context, provider string) (string, error)
	counts(ctx context.Context, provider string) (total int, batch int, err error)
}

// burst = max(1, 10 % hạn mức) (SRS 4.3).
func burst(limit int) int { return max(1, limit/10) }
