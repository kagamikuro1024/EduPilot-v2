package course

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// TypeAssigned là `notifications.type` của thông báo nhận lớp.
const TypeAssigned = "COURSE_ASSIGNED"

// Notifier xử lý các topic outbox của lớp ở worker (SRS 4.9). Idempotent theo outbox id.
type Notifier struct {
	Pool         *pgxpool.Pool
	AppPublicURL string
	Log          *slog.Logger
}

// HandleAssigned (`course.assigned`): tạo ĐÚNG MỘT dòng `notifications` cho người mới được gán (dedupe `course.assigned:<outbox_id>`).
// Người đã bị gỡ trước khi worker chạy thì không nhận gì — nhất là mã tham gia nằm trong thân thông báo của giảng viên.
func (n *Notifier) HandleAssigned(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID uuid.UUID `json:"course_id"`
		UserID   uuid.UUID `json:"user_id"`
		Role     string    `json:"role"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil || p.UserID == uuid.Nil {
		return errors.New("payload course.assigned không hợp lệ")
	}
	q := store.New(n.Pool)
	c, err := q.GetCourse(ctx, p.CourseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("đọc lớp: %w", err)
	}
	if _, err := q.GetMembership(ctx, store.GetMembershipParams{CourseID: p.CourseID, UserID: p.UserID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("đọc ghi danh: %w", err)
	}
	staff, err := q.ListStaffEnrollments(ctx, p.CourseID)
	if err != nil {
		return fmt.Errorf("đọc đội ngũ: %w", err)
	}
	var role store.EnrollmentRole
	var teacherName string
	active := false
	for _, r := range staff {
		if r.UserID == p.UserID {
			active, role = true, r.RoleInCourse
		}
		if r.RoleInCourse == store.EnrollmentRoleTEACHER {
			teacherName = r.FullName
		}
	}
	if !active {
		return nil
	}

	link := fmt.Sprintf("/class/members?course=%s", c.ID)
	title := fmt.Sprintf("Bạn được phân công làm trợ giảng lớp %s – %s", c.Name, c.ClassCode)
	body := "Lớp này chưa có giảng viên phụ trách."
	if teacherName != "" {
		body = fmt.Sprintf("Giảng viên phụ trách: %s.", teacherName)
	}
	if role == store.EnrollmentRoleTEACHER {
		link = fmt.Sprintf("/class/settings?course=%s", c.ID)
		title = fmt.Sprintf("Bạn được phân công lớp %s – %s", c.Name, c.ClassCode)
		body = fmt.Sprintf("Mã tham gia: %s. Chia sẻ mã hoặc đường dẫn %s/join/%s cho sinh viên.", c.JoinCode, n.AppPublicURL, c.JoinCode)
	}
	key := TopicAssigned + ":" + m.ID.String()
	if _, err := q.InsertNotification(ctx, store.InsertNotificationParams{UserID: p.UserID, CourseID: &c.ID, Type: TypeAssigned, Title: title, Body: &body, Link: &link, DedupeKey: &key}); err != nil {
		return fmt.Errorf("ghi thông báo: %w", err)
	}
	return nil
}
