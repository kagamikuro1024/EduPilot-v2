package course

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Lỗi vào lớp bằng mã.
var (
	// ErrJoinInvalid: MỘT lỗi cho mọi nguyên nhân (mã sai / cũ / tắt / hết hạn / lớp lưu trữ / sai tên miền) — không lộ lý do (SRS 4.2).
	ErrJoinInvalid = errors.New("course: mã tham gia không hợp lệ")
	ErrCourseFull  = errors.New("course: lớp đã đủ sĩ số")
	ErrNotVerified = errors.New("course: email chưa xác minh")
)

// RateLimitedError: đoán mã quá giới hạn; RetryAfter tính bằng giây.
type RateLimitedError struct{ RetryAfter int }

func (e *RateLimitedError) Error() string { return "course: quá nhiều lần thử mã" }

// JoinState là trạng thái hiển thị ở bước xem trước.
type JoinState string

// Các trạng thái xem trước (SRS 6.3).
const (
	StateOpen             JoinState = "OPEN"
	StateRequiresApproval JoinState = "REQUIRES_APPROVAL"
	StateFull             JoinState = "FULL"
	StateAlreadyMember    JoinState = "ALREADY_MEMBER"
	StatePending          JoinState = "PENDING"
)

// Preview là thân `POST /courses/join/preview`: không `join_code`, không id lớp, không danh sách sinh viên.
type Preview struct {
	Name      string
	ClassCode string
	Semester  string
	Teachers  []string
	State     JoinState
}

// JoinResult là thân `POST /courses/join`.
type JoinResult struct {
	CourseID      uuid.UUID
	Status        string // ACTIVE | PENDING
	AlreadyMember bool
}

// normCode: không phân biệt hoa thường, bỏ mọi khoảng trắng.
func normCode(in string) string {
	return strings.ToUpper(strings.Join(strings.Fields(in), ""))
}

// deadCode không bao giờ khớp (có chữ 0, ngoài bảng ký tự) nên mã sai định dạng vẫn đi qua ĐÚNG MỘT truy vấn như mã đúng định dạng.
const deadCode = "0000000"

type joinUser struct {
	ID          uuid.UUID
	Email       string
	StudentCode string
}

// lookupByCode là bước tra mã CHUNG của preview và join. Mọi nhánh thất bại làm cùng một truy vấn, cùng một phép so sánh hằng thời gian và
// cùng một quyết định gộp (không rẽ nhánh sớm) — thời gian xử lý không phân biệt được các nguyên nhân. Thành công ⇒ lớp; thất bại ⇒ ErrJoinInvalid.
func (s Service) lookupByCode(ctx context.Context, q *store.Queries, code string, u joinUser) (store.Course, error) {
	code = normCode(code)
	lookup := code
	formatOK := joinCodeRE.MatchString(code)
	if !formatOK {
		lookup = deadCode
	}
	c, err := q.GetCourseByJoinCode(ctx, lookup)
	found := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return store.Course{}, fmt.Errorf("course: tra mã: %w", err)
	}
	stored := strings.TrimSpace(c.JoinCode)
	if !found {
		stored = deadCode
	}
	same := subtle.ConstantTimeCompare([]byte(stored), []byte(code)) == 1
	notExpired := c.JoinExpiresAt == nil || c.JoinExpiresAt.After(s.now())
	domainOK := c.AllowedEmailDomain == nil || strings.HasSuffix(strings.ToLower(u.Email), "@"+*c.AllowedEmailDomain)
	ok := found && formatOK && same && c.JoinEnabled && c.Status == store.CourseStatusACTIVE && notExpired && domainOK
	if !ok {
		return store.Course{}, ErrJoinInvalid
	}
	return c, nil
}

// guard chạy trước tra mã: vượt giới hạn ⇒ 429 kể cả mã đúng. Trả hàm ghi thất bại cho nhánh ErrJoinInvalid.
func (s Service) guardGuess(ctx context.Context, userID uuid.UUID, ip string) (func(), error) {
	g := newGuessLimiter(s.Redis, s.log())
	if secs := g.Blocked(ctx, userID, ip, s.now()); secs > 0 {
		return nil, &RateLimitedError{RetryAfter: secs}
	}
	return func() { g.Fail(ctx, userID, ip, s.now()) }, nil
}

func (s Service) joinUser(ctx context.Context, q *store.Queries, id uuid.UUID) (joinUser, error) {
	u, err := q.GetUser(ctx, id)
	if err != nil {
		return joinUser{}, fmt.Errorf("course: đọc người dùng: %w", err)
	}
	if u.EmailVerifiedAt == nil {
		return joinUser{}, ErrNotVerified
	}
	ju := joinUser{ID: u.ID, Email: u.Email}
	if u.StudentCode != nil {
		ju.StudentCode = *u.StudentCode
	}
	return ju, nil
}

// Preview xem trước lớp theo mã (không ghi gì ngoài bộ đếm thất bại).
func (s Service) Preview(ctx context.Context, userID uuid.UUID, ip, code string) (Preview, error) {
	q := store.New(s.Pool)
	u, err := s.joinUser(ctx, q, userID)
	if err != nil {
		return Preview{}, err
	}
	fail, err := s.guardGuess(ctx, userID, ip)
	if err != nil {
		return Preview{}, err
	}
	c, err := s.lookupByCode(ctx, q, code, u)
	if errors.Is(err, ErrJoinInvalid) {
		fail()
		return Preview{}, err
	}
	if err != nil {
		return Preview{}, err
	}
	teachers, err := q.ListCourseTeacherNames(ctx, c.ID)
	if err != nil {
		return Preview{}, fmt.Errorf("course: đọc giảng viên: %w", err)
	}
	out := Preview{Name: c.Name, ClassCode: c.ClassCode, Semester: c.Semester, Teachers: teachers, State: StateOpen}
	if teachers == nil {
		out.Teachers = []string{}
	}
	if c.JoinRequireApproval {
		out.State = StateRequiresApproval
	}
	counts, err := q.CountCourseStudents(ctx, c.ID)
	if err != nil {
		return Preview{}, fmt.Errorf("course: đếm sinh viên: %w", err)
	}
	if c.Capacity != nil && int(counts.Active) >= int(*c.Capacity) {
		out.State = StateFull
	}
	if m, err := q.GetMembership(ctx, store.GetMembershipParams{CourseID: c.ID, UserID: userID}); err == nil {
		switch m.Status {
		case store.EnrollmentStatusACTIVE:
			out.State = StateAlreadyMember
		case store.EnrollmentStatusPENDING:
			out.State = StatePending
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Preview{}, fmt.Errorf("course: đọc ghi danh: %w", err)
	}
	return out, nil
}

// TopicJoinRequested / TopicJoinDecided: sự kiện outbox của vào lớp (SRS 4.9).
const (
	TopicJoinRequested = "course.join_requested"
	TopicJoinDecided   = "course.join_decided"
)

// Join ghi danh bằng mã. Idempotent: khoá dòng `courses` rồi `UNIQUE (course_id, user_id)` — 50 yêu cầu song song vẫn một dòng, sĩ số không đếm hai lần.
//   - ACTIVE / PENDING sẵn có ⇒ trả nguyên trạng (AlreadyMember);
//   - `require_approval` hoặc MSSV tự khai trùng ảnh chụp của người khác ⇒ PENDING (+ EMAIL_MISMATCH), kể cả khi lớp không cần duyệt;
//   - REMOVED ⇒ dùng lại CÙNG dòng, về PENDING.
func (s Service) Join(ctx context.Context, userID uuid.UUID, ip, code string) (JoinResult, error) {
	q := store.New(s.Pool)
	u, err := s.joinUser(ctx, q, userID)
	if err != nil {
		return JoinResult{}, err
	}
	fail, err := s.guardGuess(ctx, userID, ip)
	if err != nil {
		return JoinResult{}, err
	}
	c, err := s.lookupByCode(ctx, q, code, u)
	if errors.Is(err, ErrJoinInvalid) {
		fail()
		return JoinResult{}, err
	}
	if err != nil {
		return JoinResult{}, err
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return JoinResult{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := store.New(tx)
	// Khoá lớp: tuần tự hoá ghi danh của CÙNG một lớp (kiểm sĩ số đúng) và đọc lại trạng thái lớp sau khi đã giữ khoá.
	cur, err := tq.LockCourse(ctx, c.ID)
	if err != nil {
		return JoinResult{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if cur.Status != store.CourseStatusACTIVE || !cur.JoinEnabled || cur.JoinCode != c.JoinCode {
		fail()
		return JoinResult{}, ErrJoinInvalid // mã vừa bị tạo lại / tắt / lưu trữ giữa chừng
	}
	en, err := tq.LockEnrollment(ctx, store.LockEnrollmentParams{CourseID: c.ID, UserID: userID})
	have := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return JoinResult{}, fmt.Errorf("course: đọc ghi danh: %w", err)
	}
	if have && (en.Status == store.EnrollmentStatusACTIVE || en.Status == store.EnrollmentStatusPENDING) {
		return JoinResult{CourseID: c.ID, Status: string(en.Status), AlreadyMember: true}, nil
	}

	counts, err := tq.CountCourseStudents(ctx, c.ID)
	if err != nil {
		return JoinResult{}, fmt.Errorf("course: đếm sinh viên: %w", err)
	}
	if cur.Capacity != nil && int(counts.Active) >= int(*cur.Capacity) {
		return JoinResult{}, ErrCourseFull
	}

	var warning *string
	if u.StudentCode != "" {
		held, err := tq.EnrollmentConflictByStudentCode(ctx, store.EnrollmentConflictByStudentCodeParams{CourseID: c.ID, Code: &u.StudentCode, UserID: userID})
		if err != nil {
			return JoinResult{}, fmt.Errorf("course: kiểm MSSV: %w", err)
		}
		if held {
			w := "EMAIL_MISMATCH"
			warning = &w
		}
	}
	status := store.EnrollmentStatusACTIVE
	if cur.JoinRequireApproval || warning != nil {
		status = store.EnrollmentStatusPENDING
	}
	at := s.now()
	var row store.Enrollment
	if have { // REMOVED ⇒ cùng dòng, luôn PENDING (mặc định an toàn, Q5), bất kể lớp có bật duyệt
		row, err = tq.RejoinEnrollment(ctx, store.RejoinEnrollmentParams{ID: en.ID, Warning: warning, At: at})
	} else {
		var snap *string
		if u.StudentCode != "" {
			snap = &u.StudentCode
		}
		row, err = tq.InsertCodeEnrollment(ctx, store.InsertCodeEnrollmentParams{CourseID: c.ID, UserID: userID, Status: status, Snapshot: snap, Warning: warning, At: at})
	}
	if err != nil {
		return JoinResult{}, fmt.Errorf("course: ghi danh: %w", err)
	}
	if err := s.auditEnrollment(ctx, tq, c.ID, &userID, row.ID, "member_joined", nil, map[string]any{"status": row.Status, "warning": row.Warning, "via": "CODE"}); err != nil {
		return JoinResult{}, err
	}
	if row.Status == store.EnrollmentStatusPENDING {
		if _, err := outbox.Write(ctx, tx, TopicJoinRequested, map[string]any{"course_id": c.ID.String(), "user_id": userID.String(), "mismatch": warning != nil}); err != nil {
			return JoinResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return JoinResult{}, fmt.Errorf("course: commit ghi danh: %w", err)
	}
	return JoinResult{CourseID: c.ID, Status: string(row.Status)}, nil
}

func (s Service) auditEnrollment(ctx context.Context, q *store.Queries, courseID uuid.UUID, actor *uuid.UUID, enrollmentID uuid.UUID, action string, before, after any) error {
	enc := func(v any) []byte {
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v) // chỉ map/chuỗi/số: không lỗi
		return b
	}
	// before/after chỉ chứa trạng thái — không email, không MSSV (SRS 4.3).
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: actor, Entity: "enrollment", EntityID: enrollmentID.String(), Action: action, Before: enc(before), After: enc(after)}); err != nil {
		return fmt.Errorf("course: ghi audit_log: %w", err)
	}
	return nil
}
