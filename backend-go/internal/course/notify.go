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

// Loại thông báo vào lớp.
const (
	TypeJoinRequest  = "JOIN_REQUEST"
	TypeJoinApproved = "JOIN_APPROVED"
	TypeJoinRejected = "JOIN_REJECTED"
)

func (n *Notifier) put(ctx context.Context, q *store.Queries, user uuid.UUID, courseID uuid.UUID, typ, title string, body, link *string, key string) error {
	if _, err := q.InsertNotification(ctx, store.InsertNotificationParams{UserID: user, CourseID: &courseID, Type: typ, Title: title, Body: body, Link: link, DedupeKey: &key}); err != nil {
		return fmt.Errorf("ghi thông báo: %w", err)
	}
	return nil
}

// HandleJoinRequested (`course.join_requested`): JOIN_REQUEST cho giảng viên VÀ các TA của lớp. Thân của GIẢNG VIÊN thêm "— email chưa khớp MSSV"
// khi có cảnh báo; TA nhận thân thường, không nêu lý do (TA không duyệt được hàng này). Bỏ qua nếu yêu cầu không còn chờ.
func (n *Notifier) HandleJoinRequested(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID uuid.UUID `json:"course_id"`
		UserID   uuid.UUID `json:"user_id"`
		Mismatch bool      `json:"mismatch"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil || p.UserID == uuid.Nil {
		return errors.New("payload course.join_requested không hợp lệ")
	}
	q := store.New(n.Pool)
	e, err := q.GetMembership(ctx, store.GetMembershipParams{CourseID: p.CourseID, UserID: p.UserID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && e.Status != store.EnrollmentStatusPENDING) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("đọc ghi danh: %w", err)
	}
	c, err := q.GetCourse(ctx, p.CourseID)
	if err != nil {
		return fmt.Errorf("đọc lớp: %w", err)
	}
	u, err := q.GetUser(ctx, p.UserID)
	if err != nil {
		return fmt.Errorf("đọc người xin vào: %w", err)
	}
	staff, err := q.ListStaffEnrollments(ctx, p.CourseID)
	if err != nil {
		return fmt.Errorf("đọc đội ngũ: %w", err)
	}
	title := fmt.Sprintf("%s xin vào lớp %s", u.FullName, c.ClassCode)
	link := fmt.Sprintf("/class/members?course=%s&tab=pending", c.ID)
	key := TopicJoinRequested + ":" + m.ID.String()
	for _, r := range staff {
		body := "Mở hàng chờ duyệt để xem."
		if p.Mismatch && r.RoleInCourse == store.EnrollmentRoleTEACHER {
			body += " — email chưa khớp MSSV"
		}
		if err := n.put(ctx, q, r.UserID, c.ID, TypeJoinRequest, title, &body, &link, key); err != nil {
			return err
		}
	}
	return nil
}

// HandleJoinDecided (`course.join_decided`): JOIN_APPROVED / JOIN_REJECTED cho sinh viên.
func (n *Notifier) HandleJoinDecided(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID uuid.UUID `json:"course_id"`
		UserID   uuid.UUID `json:"user_id"`
		Decision string    `json:"decision"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil || p.UserID == uuid.Nil {
		return errors.New("payload course.join_decided không hợp lệ")
	}
	q := store.New(n.Pool)
	c, err := q.GetCourse(ctx, p.CourseID)
	if err != nil {
		return fmt.Errorf("đọc lớp: %w", err)
	}
	key := TopicJoinDecided + ":" + m.ID.String()
	switch p.Decision {
	case "APPROVED":
		// link NULL = "Hôm nay": CHECK `notifications_link_chk` (^/[^/\\]) của 00003 không nhận "/" (đề xuất #9); chuông mở "/" khi không có link.
		return n.put(ctx, q, p.UserID, c.ID, TypeJoinApproved, fmt.Sprintf("Bạn đã được duyệt vào lớp %s – %s", c.Name, c.ClassCode), nil, nil, key)
	case "REJECTED":
		link := "/join"
		return n.put(ctx, q, p.UserID, c.ID, TypeJoinRejected, fmt.Sprintf("Yêu cầu vào lớp %s chưa được chấp nhận", c.ClassCode), nil, &link, key)
	}
	return errors.New("quyết định course.join_decided không hợp lệ")
}
