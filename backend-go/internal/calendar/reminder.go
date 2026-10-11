package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/store"
)

// ReminderLeaderKey: một bản worker chạy reminder.tick (SET NX PX 120000).
const ReminderLeaderKey = "ep:reminder:tick:leader"

const reminderBatch = 200

// Reminder là bộ nhắc 24 giờ (SRS 4.9).
type Reminder struct {
	Svc   *Service
	Name  string        // định danh của bản chạy, giữ khoá leader
	Every time.Duration // REMINDER_TICK
	Lead  time.Duration // REMINDER_LEAD
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (r *Reminder) Run(ctx context.Context) error {
	tk := time.NewTicker(r.Every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			if r.Svc.Redis != nil && !r.Svc.Redis.Leader(ctx, ReminderLeaderKey, r.Name, 120*time.Second) {
				continue
			}
			if _, err := r.Tick(ctx); err != nil && r.Log != nil {
				r.Log.ErrorContext(ctx, "reminder.tick lỗi", "error", err.Error())
			}
		}
	}
}

// prefKey ánh xạ loại nguồn → khoá `preferences.reminders`.
func prefKey(typ string) string {
	switch typ {
	case "EXAM":
		return "exam"
	case "CLASS_SESSION":
		return "class_session"
	}
	return "other"
}

func wants(prefs []byte, key string) bool {
	var p struct {
		Reminders map[string]bool `json:"reminders"`
	}
	_ = json.Unmarshal(prefs, &p)
	if v, ok := p.Reminders[key]; ok {
		return v
	}
	return key != "class_session" // mặc định: bài thi + sự kiện khác bật, buổi học tắt
}

// Tick chạy một vòng; trả số nhắc đã tạo. Idempotent: reminder_log UNIQUE, chỉ dòng chèn được mới sinh chuông / thư.
func (r *Reminder) Tick(ctx context.Context) (int, error) {
	now := r.Svc.now()
	q := store.New(r.Svc.Pool)
	due, err := q.CalDue(ctx, store.CalDueParams{NowAt: now, UntilAt: now.Add(r.Lead)})
	if err != nil {
		return 0, fmt.Errorf("calendar: nguồn tới hạn nhắc: %w", err)
	}
	total := 0
	for _, d := range due {
		after := uuid.Nil
		for {
			rcpt, err := q.CalRecipients(ctx, store.CalRecipientsParams{CourseID: d.CourseID, AfterUser: after, PageLimit: reminderBatch})
			if err != nil {
				return total, fmt.Errorf("calendar: người nhận: %w", err)
			}
			if len(rcpt) == 0 {
				break
			}
			n, err := r.batch(ctx, d, rcpt)
			total += n
			if err != nil {
				return total, err
			}
			after = rcpt[len(rcpt)-1].UserID
		}
	}
	return total, nil
}

func (r *Reminder) batch(ctx context.Context, d store.CalDueRow, rcpt []store.CalRecipientsRow) (int, error) {
	tx, err := r.Svc.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("calendar: mở giao dịch nhắc: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	link := "/calendar"
	if d.Src == "WEEKLY_EXAM" {
		link = "/exams/" + d.ID.String() + "/take"
	}
	dedupe := "remind:" + d.Src + ":" + d.ID.String() + ":" + strconv.FormatInt(d.StartsAt.Unix(), 10)
	title := "Sắp đến giờ: " + d.Title
	body := Format(d.StartsAt)
	n := 0
	for _, u := range rcpt {
		if !wants(u.Preferences, prefKey(d.Typ)) {
			continue
		}
		ins, err := q.CalInsertReminder(ctx, store.CalInsertReminderParams{UserID: u.UserID, CourseID: d.CourseID, SourceType: store.ReminderSource(d.Src), SourceID: d.ID, StartsAt: d.StartsAt})
		if err != nil {
			return 0, fmt.Errorf("calendar: ghi reminder_log: %w", err)
		}
		if ins == 0 {
			continue
		}
		n++
		course := d.CourseID
		if _, err := q.InsertNotification(ctx, store.InsertNotificationParams{UserID: u.UserID, CourseID: &course, Type: "REMINDER", Title: title, Body: &body, Link: &link, DedupeKey: &dedupe}); err != nil {
			return 0, fmt.Errorf("calendar: thông báo nhắc: %w", err)
		}
		if u.RemindMail && u.Verified {
			if _, _, err := mail.Enqueue(ctx, tx, mail.Message{To: u.Email, Template: "reminder", DedupeKey: "reminder:" + u.UserID.String() + ":" + dedupe,
				Payload: map[string]any{"event_title": d.Title, "at": d.StartsAt.UTC().Format(time.RFC3339)}}); err != nil {
				return 0, fmt.Errorf("calendar: thư nhắc: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("calendar: commit nhắc: %w", err)
	}
	return n, nil
}
