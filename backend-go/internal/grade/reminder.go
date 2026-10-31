package grade

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/edupilot/backend-go/internal/notify"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Loại chuông của sổ điểm (SRS 4.5.4, 4.7.5).
const (
	TypeSchemeReminder = "GRADE_SCHEME_REMINDER"
	TypeFinalized      = "GRADE_FINALIZED"
	TypeChanged        = "GRADE_CHANGED"
)

// ReminderLeaderKey: một bản worker chạy `grade.scheme_reminder`.
const ReminderLeaderKey = "ep:lock:grade.remind"

// RemindAt là GRADE_REMIND_AT: 08:00 thứ Hai Asia/Ho_Chi_Minh.
const RemindAt = 8 * time.Hour

var ict = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) //nolint:gochecknoglobals // múi giờ cố định, bất biến

// ReminderJob nhắc giảng viên MỖI TUẦN khi lớp chưa có công thức điểm CONFIRMED (SRS 4.5.4): tick 1 phút; từ 08:00 thứ Hai trở đi trong tuần (chạy bù nếu worker lên lại muộn)
// chạy đúng một lần mỗi tuần nhờ `SET ep:grade.remind:{yyyyww} NX EX 8d` + khoá leader.
type ReminderJob struct {
	Svc   *Service
	Redis *appredis.Client
	Name  string
	Every time.Duration
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *ReminderJob) Run(ctx context.Context) error {
	tk := time.NewTicker(j.Every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			if _, err := j.Once(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "grade.scheme_reminder lỗi", "error", err.Error())
			}
		}
	}
}

// weekKey là `yyyyww` theo tuần ISO của giờ Asia/Ho_Chi_Minh.
func weekKey(t time.Time) string {
	y, w := t.In(ict).ISOWeek()
	return fmt.Sprintf("%04d%02d", y, w)
}

// weekStartICT là 00:00 thứ Hai của tuần chứa t.
func weekStartICT(t time.Time) time.Time {
	l := t.In(ict)
	d := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, ict)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// Once là một tick: true khi lượt nhắc của tuần này được thực hiện ở tick này.
func (j *ReminderJob) Once(ctx context.Context) (bool, error) {
	now := j.Svc.now()
	if now.Before(weekStartICT(now).Add(RemindAt)) {
		return false, nil // chưa tới 08:00 thứ Hai của tuần này
	}
	if !j.Redis.Leader(ctx, ReminderLeaderKey, j.Name, 2*j.Every) {
		return false, nil
	}
	done := appredis.Key("grade.remind", weekKey(now))
	ok, err := j.Redis.SetNX(ctx, done, 1, 8*24*time.Hour).Result()
	if err != nil || !ok {
		return false, err
	}
	if err := j.Svc.RemindUnconfirmed(ctx, now); err != nil {
		_ = j.Redis.Del(context.WithoutCancel(ctx), done).Err() // tick sau thử lại
		return false, err
	}
	return true, nil
}

// RemindUnconfirmed gửi chuông GRADE_SCHEME_REMINDER cho TEACHER của mọi lớp ACTIVE chưa có công thức CONFIRMED. Khử trùng theo (lớp, tuần, người):
// chạy lại cùng tuần không thêm. Chỉ TEACHER nhận (TA, STUDENT, ADMIN không bao giờ).
func (s *Service) RemindUnconfirmed(ctx context.Context, now time.Time) error {
	if s.Notify == nil {
		return fmt.Errorf("grade: chưa cấu hình thông báo")
	}
	rows, err := store.New(s.Pool).GradeReminderTargets(ctx)
	if err != nil {
		return fmt.Errorf("grade: lớp chưa có công thức: %w", err)
	}
	wk := weekKey(now)
	for _, r := range rows {
		course := r.CourseID
		link := "/gradebook/scheme?course=" + course.String()
		if _, err := s.Notify.Send(ctx, notify.Spec{UserID: r.TeacherID, CourseID: &course, Type: TypeSchemeReminder,
			Title: "Lớp " + r.ClassCode + " chưa có công thức điểm đã xác nhận.", Link: &link,
			DedupeKey: "grade.scheme_reminder:" + course.String() + ":" + wk + ":" + r.TeacherID.String()}); err != nil {
			return err
		}
	}
	return nil
}
