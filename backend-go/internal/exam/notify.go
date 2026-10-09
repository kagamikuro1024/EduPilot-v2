package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Loại thông báo của bài thi (`notifications.type`).
const (
	TypeExamScheduled   = "EXAM_SCHEDULED"
	TypeExamUnscheduled = "EXAM_UNSCHEDULED"
)

// Notifier xử lý các topic outbox của bài thi ở worker (SRS 4.10). Idempotent: khử trùng theo (bài, người) hoặc (outbox id, người).
type Notifier struct {
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

type examEvent struct {
	ExamID   uuid.UUID `json:"exam_id"`
	CourseID uuid.UUID `json:"course_id"`
}

func (n *Notifier) load(ctx context.Context, m outbox.Message) (examEvent, *store.Exam, error) {
	var p examEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.ExamID == uuid.Nil || p.CourseID == uuid.Nil {
		return p, nil, fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	e, err := store.New(n.Pool).ExamGet(ctx, store.ExamGetParams{CourseID: p.CourseID, ID: p.ExamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil, nil // bài đã bị xoá: không còn gì để báo
	}
	if err != nil {
		return p, nil, fmt.Errorf("đọc bài thi: %w", err)
	}
	return p, &e, nil
}

// ViTime là "Thứ Ba, 09:00 01/12" theo Asia/Ho_Chi_Minh (không phụ thuộc tzdata của máy chạy).
func ViTime(t time.Time) string {
	t = t.In(time.FixedZone("Asia/Ho_Chi_Minh", 7*3600))
	weekday := [...]string{"Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"}
	return fmt.Sprintf("%s, %s", weekday[t.Weekday()], t.Format("15:04 02/01"))
}

// HandleScheduled (`exam.scheduled`): EXAM_SCHEDULED cho mọi sinh viên ACTIVE của lớp — MỘT câu lệnh cho cả lớp. Bỏ qua nếu bài không còn SCHEDULED
// (bỏ lịch ngay sau khi lên lịch): không báo một lịch đã bị thu hồi.
func (n *Notifier) HandleScheduled(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil {
		return err
	}
	if e.Status == store.ExamStatusDRAFT || e.OpensAt == nil || e.DurationMinutes == nil {
		return nil
	}
	link := fmt.Sprintf("/exams/%s/take", e.ID)
	title := fmt.Sprintf("Bài thi %s mở lúc %s, làm trong %d phút", e.Title, ViTime(*e.OpensAt), *e.DurationMinutes)
	if _, err := store.New(n.Pool).ExamNotifyScheduled(ctx, store.ExamNotifyScheduledParams{CourseID: p.CourseID, ExamID: p.ExamID.String(), Title: title, Body: nil, Link: &link}); err != nil {
		return fmt.Errorf("ghi thông báo: %w", err)
	}
	return nil
}

// HandleUnscheduled (`exam.unscheduled`): thu hồi thông báo lịch cũ (xoá, để lần lên lịch sau báo lại được) và báo "hoãn" cho những người đã nhận. Bỏ qua nếu bài đã
// được lên lịch lại trước khi worker chạy (không thu hồi lịch mới).
func (n *Notifier) HandleUnscheduled(ctx context.Context, m outbox.Message) error {
	p, e, err := n.load(ctx, m)
	if err != nil || e == nil || e.Status != store.ExamStatusDRAFT {
		return err
	}
	q := store.New(n.Pool)
	users, err := q.ExamRecallScheduled(ctx, store.ExamRecallScheduledParams{CourseID: &p.CourseID, ExamID: p.ExamID.String()})
	if err != nil {
		return fmt.Errorf("thu hồi thông báo lịch: %w", err)
	}
	if len(users) == 0 {
		return nil
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.String()
	}
	link := fmt.Sprintf("/exams/%s/take", e.ID)
	if _, err := q.ExamNotifyUnscheduled(ctx, store.ExamNotifyUnscheduledParams{CourseID: &p.CourseID, Title: fmt.Sprintf("Bài thi %s đã bị hoãn, chờ lịch mới", e.Title), Link: &link, OutboxID: m.ID.String(), UserIds: ids}); err != nil {
		return fmt.Errorf("ghi thông báo hoãn: %w", err)
	}
	return nil
}
