package attendance

import (
	"context"
	"log/slog"
	"time"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

var ict = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) //nolint:gochecknoglobals // múi giờ cố định, bất biến

// PurgeLeaderKey: khoá leader của `attendance.purge` (chỉ chống hai bản chạy cùng lúc; «đã chạy hôm nay» do khoá ngày).
var PurgeLeaderKey = appredis.Key("lock", "attendance.purge") //nolint:gochecknoglobals // tên khoá cố định

// purgeAt: giờ chạy `attendance.purge` mỗi ngày (03:00 ICT, ARCHITECTURE 11.6).
const purgeAt = 3 * time.Hour

// SweepJob (`attendance.sweep`, FR-7): đóng cửa sổ đã quá giờ mỗi CHECKIN_SWEEP_INTERVAL_S. Không leader: Close idempotent và tự khoá theo thứ tự
// buổi → cửa sổ, nên N worker cùng chạy cũng an toàn.
type SweepJob struct {
	Svc *Service
	Log *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *SweepJob) Run(ctx context.Context) error {
	every := j.Svc.Cfg.SweepInterval
	if every <= 0 {
		every = time.Minute
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if _, err := j.Svc.Sweep(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "attendance.sweep lỗi", "error", err.Error())
			}
		}
	}
}

// PurgeJob (`attendance.purge`, FR-21): mỗi phút kiểm, chạy MỘT lần mỗi ngày sau 03:00 ICT, một bản nhờ khoá leader + dấu ngày Redis (như crm.RiskJob).
type PurgeJob struct {
	Svc   *Service
	Redis *appredis.Client
	Name  string
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *PurgeJob) Run(ctx context.Context) error {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if _, err := j.Once(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "attendance.purge lỗi", "error", err.Error())
			}
		}
	}
}

// Once là một tick: true khi lượt chạy của hôm nay được thực hiện ở tick này.
func (j *PurgeJob) Once(ctx context.Context) (bool, error) {
	now := j.Svc.now().In(ict)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ict)
	if now.Before(day.Add(purgeAt)) {
		return false, nil
	}
	if !j.Redis.Leader(ctx, PurgeLeaderKey, j.Name, 2*time.Minute) {
		return false, nil
	}
	done := appredis.Key("attendance.purge", "done", day.Format("20060102"))
	ok, err := j.Redis.SetNX(ctx, done, 1, 48*time.Hour).Result()
	if err != nil || !ok {
		return false, err
	}
	if _, _, err := j.Svc.PurgeCheckins(ctx); err != nil {
		_ = j.Redis.Del(context.WithoutCancel(ctx), done).Err() // tick sau chạy lại
		return false, err
	}
	return true, nil
}
