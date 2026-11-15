package practice

import (
	"context"
	"log/slog"
	"time"
)

// CloseJob chạy `practice.close` (FR-16): mỗi `Every` đóng các lượt MOCK quá hạn + dung sai. Không cần khoá leader: `PracticeDue` dùng FOR UPDATE SKIP LOCKED
// nên nhiều worker không đóng trùng, và `CloseDue` idempotent.
type CloseJob struct {
	Svc   *Service
	Every time.Duration
	Log   *slog.Logger
}

// Run chạy tới khi ctx bị huỷ.
func (j *CloseJob) Run(ctx context.Context) error {
	every := j.Every
	if every <= 0 {
		every = time.Minute
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			n, err := j.Svc.CloseDue(ctx)
			if err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "practice.close lỗi", "error", err.Error())
			} else if n > 0 && j.Log != nil {
				j.Log.InfoContext(ctx, "practice.close", "closed", n)
			}
		}
	}
}
