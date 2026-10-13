package activity

import (
	"context"
	"log/slog"
	"time"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// RollupLeaderKey là khoá leader của `heartbeat.rollup`: mỗi tick chỉ MỘT worker cuộn (bản kia bỏ qua), TTL = 2 × tick.
var RollupLeaderKey = appredis.Key("lock", "heartbeat.rollup") //nolint:gochecknoglobals // tên khoá cố định

// RollupJob chạy `Service.Rollup` mỗi ROLLUP_TICK. Một bản nhờ khoá leader Redis.
type RollupJob struct {
	Svc   *Service
	Redis *appredis.Client
	Name  string
	Every time.Duration // ROLLUP_TICK
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *RollupJob) Run(ctx context.Context) error {
	tick := time.NewTicker(j.Every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if j.Redis != nil && !j.Redis.Leader(ctx, RollupLeaderKey, j.Name, 2*j.Every) {
				continue
			}
			if n, err := j.Svc.Rollup(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "heartbeat.rollup lỗi", "error", err.Error())
			} else if n > 0 && j.Log != nil {
				j.Log.DebugContext(ctx, "heartbeat.rollup", "rows", n)
			}
		}
	}
}
