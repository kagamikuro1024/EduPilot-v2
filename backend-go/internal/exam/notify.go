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

	"github.com/edupilot/backend-go/internal/notify"
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
	// Notify: một cửa tạo thông báo (ghi hàng loạt + đẩy SSE); nil → chỉ ghi.
	Notify *notify.Service
}

func (n *Notifier) svc() *notify.Service {
	if n.Notify != nil {
		return n.Notify
	}
	return &notify.Service{Pool: n.Pool, Log: n.Log}
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
	students, err := store.New(n.Pool).ExamActiveStudentIDs(ctx, p.CourseID)
	if err != nil {
		return fmt.Errorf("đọc sinh viên của lớp: %w", err)
	}
	if _, err := n.svc().SendMany(ctx, students, notify.Spec{CourseID: &p.CourseID, Type: TypeExamScheduled, Title: title, Link: &link, DedupeKey: "exam.scheduled:" + p.ExamID.String()}); err != nil {
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
	link := fmt.Sprintf("/exams/%s/take", e.ID)
	if _, err := n.svc().SendMany(ctx, users, notify.Spec{CourseID: &p.CourseID, Type: TypeExamUnscheduled, Title: fmt.Sprintf("Bài thi %s đã bị hoãn, chờ lịch mới", e.Title), Link: &link, DedupeKey: "exam.unscheduled:" + m.ID.String()}); err != nil {
		return fmt.Errorf("ghi thông báo hoãn: %w", err)
	}
	return nil
}
