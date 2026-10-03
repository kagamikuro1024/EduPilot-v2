package course

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// UndoWindow: thời gian cho phép hoàn tác một đổi trạng thái thành viên (SRS 4.3, COURSE_UNDO_WINDOW).
const UndoWindow = 60 * time.Second

// Lỗi thành viên (handler ánh xạ sang mã API).
var (
	ErrMismatchNeedsTeacher = errors.New("course: duyệt email chưa khớp MSSV cần giảng viên")
	ErrUndoExpired          = errors.New("course: không còn gì để hoàn tác")
	ErrNotPending           = errors.New("course: thành viên không ở trạng thái chờ duyệt")
	ErrNotActive            = errors.New("course: thành viên không ở trạng thái đang học")
	ErrStaffMember          = errors.New("course: giảng viên và trợ giảng đổi bằng gán lớp, không bằng đường thành viên")
)

// ConfirmError: thiếu `confirm_mismatch` khi duyệt hàng EMAIL_MISMATCH (422).
type ConfirmError struct{}

func (ConfirmError) Error() string { return "course: cần xác nhận email chưa khớp" }

// Actor là người thao tác thành viên: Teacher = giảng viên của lớp hoặc ADMIN (chế độ Manage).
type Actor struct {
	ID      uuid.UUID
	Teacher bool
}

// Member là một dòng danh sách thành viên.
type Member struct {
	UserID          uuid.UUID
	FullName        string
	Email           string
	StudentCode     string
	RoleInCourse    string
	Status          string
	JoinedVia       string
	Warning         *string
	StatusChangedAt time.Time
}

// MemberFilter là query của GET …/members.
type MemberFilter struct {
	Status, Role, Q string
	// ByStudentCode cho `Q` khớp cả MSSV: chỉ giảng viên / TA ĐANG HOẠT ĐỘNG của lớp (đề xuất #10). Người khác chỉ tìm được theo tên / email.
	ByStudentCode bool
	Cursor        *Cursor
	Limit         int
}

// MemberPage là một trang thành viên kèm số đếm sinh viên của cả lớp.
type MemberPage struct {
	Rows    []Member // tối đa Limit+1
	Active  int
	Pending int
}

// Members liệt kê thành viên bằng MỘT truy vấn. Không có password_hash, ics_token, điểm, chuyên cần.
func (s Service) Members(ctx context.Context, courseID uuid.UUID, f MemberFilter) (MemberPage, error) {
	p := store.ListMembersParams{CourseID: courseID, Lim: int32(f.Limit + 1)}
	if f.Status != "" {
		st := store.EnrollmentStatus(f.Status)
		p.Status = &st
	}
	if f.Role != "" {
		r := store.EnrollmentRole(f.Role)
		p.Role = &r
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		name := "%" + likeEscape(auth.Fold(q)) + "%"
		code := likeEscape(strings.ToUpper(q)) + "%"
		email := likeEscape(strings.ToLower(q)) + "%"
		p.NameLike, p.EmailLike = &name, &email
		if f.ByStudentCode {
			p.CodeLike = &code
		}
	}
	if f.Cursor != nil {
		p.CurAt, p.CurID = &f.Cursor.At, &f.Cursor.ID
	}
	rows, err := store.New(s.Pool).ListMembers(ctx, p)
	if err != nil {
		return MemberPage{}, fmt.Errorf("course: liệt kê thành viên: %w", err)
	}
	out := MemberPage{}
	for _, r := range rows {
		out.Active, out.Pending = int(r.Active), int(r.Pending)
		if r.UserID == nil { // dòng mang số đếm của trang rỗng
			continue
		}
		m := Member{UserID: *r.UserID, FullName: *r.FullName, Email: *r.Email, RoleInCourse: string(*r.RoleInCourse), Status: string(*r.Status), JoinedVia: string(*r.JoinedVia), Warning: r.Warning, StatusChangedAt: *r.StatusChangedAt}
		if r.StudentCodeSnapshot != nil {
			m.StudentCode = *r.StudentCodeSnapshot
		}
		out.Rows = append(out.Rows, m)
	}
	return out, nil
}

// Transition là kết quả một đổi trạng thái thành viên.
type Transition struct {
	UserID         uuid.UUID
	Status         string
	PreviousStatus *string
	Warning        *string
}

func transitionOf(e store.Enrollment) Transition {
	t := Transition{UserID: e.UserID, Status: string(e.Status), Warning: e.Warning}
	if e.PreviousStatus != nil {
		p := string(*e.PreviousStatus)
		t.PreviousStatus = &p
	}
	return t
}

// member mở transaction, khoá lớp (từ chối nếu đã lưu trữ) rồi khoá dòng ghi danh của uid.
func (s Service) member(ctx context.Context, courseID, uid uuid.UUID, fn func(tx pgx.Tx, q *store.Queries, c store.Course, e store.Enrollment) (store.Enrollment, error)) (Transition, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Transition{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Transition{}, ErrNotFound
	}
	if err != nil {
		return Transition{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if c.Status == store.CourseStatusARCHIVED {
		return Transition{}, ErrArchived
	}
	e, err := q.LockEnrollment(ctx, store.LockEnrollmentParams{CourseID: courseID, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return Transition{}, ErrNotFound
	}
	if err != nil {
		return Transition{}, fmt.Errorf("course: khoá ghi danh: %w", err)
	}
	if e.RoleInCourse != store.EnrollmentRoleSTUDENT {
		return Transition{}, ErrStaffMember
	}
	out, err := fn(tx, q, c, e)
	if err != nil {
		return Transition{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Transition{}, fmt.Errorf("course: commit thành viên: %w", err)
	}
	return transitionOf(out), nil
}

func (s Service) setStatus(ctx context.Context, q *store.Queries, c store.Course, e store.Enrollment, to store.EnrollmentStatus, previous *store.EnrollmentStatus, by uuid.UUID, action string) (store.Enrollment, error) {
	row, err := q.SetEnrollmentStatus(ctx, store.SetEnrollmentStatusParams{ID: e.ID, Status: to, Previous: previous, At: s.now(), Actor: &by})
	if err != nil {
		return store.Enrollment{}, fmt.Errorf("course: đổi trạng thái: %w", err)
	}
	if err := s.auditEnrollment(ctx, q, c.ID, &by, e.ID, action, map[string]any{"status": e.Status}, map[string]any{"status": row.Status}); err != nil {
		return store.Enrollment{}, err
	}
	return row, nil
}

// roomLeft kiểm sĩ số khi thêm một sinh viên ACTIVE (duyệt, hoàn tác mời ra).
func roomLeft(ctx context.Context, q *store.Queries, c store.Course) error {
	if c.Capacity == nil {
		return nil
	}
	n, err := q.CountCourseStudents(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("course: đếm sinh viên: %w", err)
	}
	if int(n.Active) >= int(*c.Capacity) {
		return ErrCourseFull
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

// Approve duyệt PENDING → ACTIVE. Hàng EMAIL_MISMATCH chỉ giảng viên / Admin duyệt và phải gửi confirm_mismatch (TA → ErrMismatchNeedsTeacher).
func (s Service) Approve(ctx context.Context, actor Actor, courseID, uid uuid.UUID, confirmMismatch bool) (Transition, error) {
	return s.member(ctx, courseID, uid, func(tx pgx.Tx, q *store.Queries, c store.Course, e store.Enrollment) (store.Enrollment, error) {
		if e.Status != store.EnrollmentStatusPENDING {
			return store.Enrollment{}, ErrNotPending
		}
		if e.Warning != nil && *e.Warning == "EMAIL_MISMATCH" {
			if !actor.Teacher {
				return store.Enrollment{}, ErrMismatchNeedsTeacher
			}
			if !confirmMismatch {
				return store.Enrollment{}, ConfirmError{}
			}
		}
		if err := roomLeft(ctx, q, c); err != nil {
			return store.Enrollment{}, err
		}
		row, err := s.setStatus(ctx, q, c, e, store.EnrollmentStatusACTIVE, ptr(store.EnrollmentStatusPENDING), actor.ID, "member_approved")
		if err != nil {
			return store.Enrollment{}, err
		}
		_, err = outbox.Write(ctx, tx, TopicJoinDecided, map[string]string{"course_id": c.ID.String(), "user_id": uid.String(), "decision": "APPROVED"})
		return row, err
	})
}

// Reject từ chối PENDING → REMOVED.
func (s Service) Reject(ctx context.Context, actor Actor, courseID, uid uuid.UUID) (Transition, error) {
	return s.member(ctx, courseID, uid, func(tx pgx.Tx, q *store.Queries, c store.Course, e store.Enrollment) (store.Enrollment, error) {
		if e.Status != store.EnrollmentStatusPENDING {
			return store.Enrollment{}, ErrNotPending
		}
		row, err := s.setStatus(ctx, q, c, e, store.EnrollmentStatusREMOVED, ptr(store.EnrollmentStatusPENDING), actor.ID, "member_rejected")
		if err != nil {
			return store.Enrollment{}, err
		}
		_, err = outbox.Write(ctx, tx, TopicJoinDecided, map[string]string{"course_id": c.ID.String(), "user_id": uid.String(), "decision": "REJECTED"})
		return row, err
	})
}

// Remove mời ACTIVE → REMOVED (chỉ giảng viên / Admin — route Manage). Mất quyền ngay: guard không cache; dữ liệu học tập giữ nguyên.
func (s Service) Remove(ctx context.Context, actor Actor, courseID, uid uuid.UUID) (Transition, error) {
	return s.member(ctx, courseID, uid, func(_ pgx.Tx, q *store.Queries, c store.Course, e store.Enrollment) (store.Enrollment, error) {
		if e.Status != store.EnrollmentStatusACTIVE {
			return store.Enrollment{}, ErrNotActive
		}
		return s.setStatus(ctx, q, c, e, store.EnrollmentStatusREMOVED, ptr(store.EnrollmentStatusACTIVE), actor.ID, "member_removed")
	})
}

// Undo đưa về previous_status nếu cùng quyền với hành động gốc và trong UndoWindow. Mời ra cần giảng viên / Admin.
func (s Service) Undo(ctx context.Context, actor Actor, courseID, uid uuid.UUID) (Transition, error) {
	return s.member(ctx, courseID, uid, func(_ pgx.Tx, q *store.Queries, c store.Course, e store.Enrollment) (store.Enrollment, error) {
		if e.PreviousStatus == nil || s.now().Sub(e.StatusChangedAt) > UndoWindow {
			return store.Enrollment{}, ErrUndoExpired
		}
		prev := *e.PreviousStatus
		if e.Status == store.EnrollmentStatusREMOVED && prev == store.EnrollmentStatusACTIVE && !actor.Teacher {
			return store.Enrollment{}, ErrForbiddenAction // hành động gốc (mời ra) chỉ giảng viên
		}
		if prev == store.EnrollmentStatusACTIVE {
			if err := roomLeft(ctx, q, c); err != nil {
				return store.Enrollment{}, err
			}
		}
		return s.setStatus(ctx, q, c, e, prev, nil, actor.ID, "member_undo")
	})
}

// ErrForbiddenAction: người gọi không đủ quyền cho hành động này (handler trả 403 reason=role).
var ErrForbiddenAction = errors.New("course: không đủ quyền cho thao tác này")
