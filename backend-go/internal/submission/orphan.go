package submission

import (
	"context"
	"log/slog"
	"time"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// OrphanLeaderKey là khoá leader của job `submission.orphan_cleanup` (một bản chạy mỗi giờ).
const OrphanLeaderKey = "ep:leader:submission.orphan_cleanup"

// OrphanJob chạy OrphanCleanup mỗi `Every` (mặc định 1 giờ); một bản nhờ khoá leader Redis.
type OrphanJob struct {
	Svc   *Service
	Redis *appredis.Client
	Name  string
	Every time.Duration
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *OrphanJob) Run(ctx context.Context) error {
	every := j.Every
	if every <= 0 {
		every = time.Hour
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			if j.Redis != nil && !j.Redis.Leader(ctx, OrphanLeaderKey, j.Name, 2*every) {
				continue
			}
			n, err := j.Svc.OrphanCleanup(ctx)
			switch {
			case err != nil && j.Log != nil:
				j.Log.ErrorContext(ctx, "submission.orphan_cleanup lỗi", "error", err.Error())
			case n > 0 && j.Log != nil:
				j.Log.InfoContext(ctx, "submission.orphan_cleanup", "deleted", n)
			}
		}
	}
}
