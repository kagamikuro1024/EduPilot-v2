package course

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Lỗi nghiệp vụ (handler ánh xạ sang mã API).
var (
	ErrClassCodeTaken = errors.New("course: mã lớp đã có")
	ErrArchived       = errors.New("course: lớp đã lưu trữ")
	ErrCodeExhausted  = errors.New("course: không sinh được mã tham gia không trùng")
)

// InvalidError: một trường sai (422 VALIDATION_FAILED).
type InvalidError struct{ Field, Code, Message string }

func (e *InvalidError) Error() string { return "course: " + e.Field + ": " + e.Code }

// VersionConflictError: sai version; Current là bản hiện hành (409 VERSION_CONFLICT).
type VersionConflictError struct{ Current AdminView }

func (e *VersionConflictError) Error() string { return "course: sai version" }

// maxCodeAttempts: số lần thử sinh mã khi trùng UNIQUE trước khi báo lỗi (SRS 4.2).
const maxCodeAttempts = 5

// TopicAssigned là topic outbox khi một giảng viên / TA MỚI được gán vào lớp (SRS 4.9).
const TopicAssigned = "course.assigned"

// Các sự kiện xoá cache "Hôm nay" (US-P2-11, SRS 4.9): payload nêu course_id (+ user_id / user_ids của người liên quan).
const (
	TopicChanged       = "course.changed"
	TopicMemberChanged = "course.member_changed"
	TopicRosterImport  = "roster.imported"
)

// emitChanged ghi `course.changed` trong transaction của thay đổi.
func emitChanged(ctx context.Context, tx pgx.Tx, courseID uuid.UUID) error {
	_, err := outbox.Write(ctx, tx, TopicChanged, map[string]string{"course_id": courseID.String()})
	return err
}

// Person là người trong cột "Giảng viên" / "Trợ giảng".
type Person struct {
	ID       uuid.UUID
	FullName string
}

// AdminView là một lớp ở danh sách của Admin. KHÔNG có join_code, KHÔNG có danh sách sinh viên.
type AdminView struct {
	ID              uuid.UUID
	ClassCode       string
	SubjectCode     string
	Name            string
	Semester        string
	Status          string
	Teacher         *Person
	AssistantsCount int
	StudentsActive  int
	StudentsPending int
	Capacity        *int
	Version         int
	CreatedAt       time.Time
}

var (
	subjectRE  = regexp.MustCompile(`^[A-Z0-9._-]{2,20}$`)
	classRE    = regexp.MustCompile(`^[A-Za-z0-9._-]{3,20}$`)
	semesterRE = regexp.MustCompile(`^[0-9]{4}-[0-9]{4}-HK[123]$`)
)

// CreateInput là thân POST /admin/courses. JoinCode chỉ được đặt ngoài production (seed).
type CreateInput struct {
	SubjectCode, ClassCode, Name, Semester string
	Capacity                               *int
	TeacherID                              *uuid.UUID
	TAIDs                                  []uuid.UUID
	JoinCode                               string
}

func invalid(field, code, msg string) *InvalidError {
	return &InvalidError{Field: field, Code: code, Message: msg}
}

func checkCapacity(c *int) error {
	if c != nil && (*c < 1 || *c > 1000) {
		return invalid("capacity", "OUT_OF_RANGE", "Sĩ số từ 1 đến 1.000.")
	}
	return nil
}

func normSubject(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !subjectRE.MatchString(s) {
		return "", invalid("subject_code", "FORMAT", "Mã học phần gồm 2–20 chữ, số hoặc . _ -.")
	}
	return s, nil
}

func normClass(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !classRE.MatchString(s) {
		return "", invalid("class_code", "FORMAT", "Mã lớp gồm 3–20 chữ, số hoặc . _ -.")
	}
	return s, nil
}

func normName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if n := len([]rune(s)); n < 1 || n > 120 {
		return "", invalid("name", "LENGTH", "Tên lớp từ 1 đến 120 ký tự.")
	}
	return s, nil
}

func normSemester(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !semesterRE.MatchString(s) {
		return "", invalid("semester", "FORMAT", "Học kỳ có dạng 2026-2027-HK1.")
	}
	return s, nil
}

func (s Service) audit(ctx context.Context, q *store.Queries, courseID, actor uuid.UUID, entityID, action string, before, after any) error {
	enc := func(v any) json.RawMessage {
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v) // chỉ chứa chuỗi / số / mảng id: không lỗi
		return b
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "course", EntityID: entityID, Action: action, Before: enc(before), After: enc(after)}); err != nil {
		return fmt.Errorf("course: ghi audit_log: %w", err)
	}
	return nil
}

func isUnique(err error, constraint string) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == constraint
}

// Create mở lớp: một transaction = lớp + mã tham gia + (nếu có) gán giảng viên / TA + outbox `course.assigned` + audit_log.
func (s Service) Create(ctx context.Context, actor uuid.UUID, in CreateInput) (AdminView, error) {
	subject, err := normSubject(in.SubjectCode)
	if err != nil {
		return AdminView{}, err
	}
	class, err := normClass(in.ClassCode)
	if err != nil {
		return AdminView{}, err
	}
	name, err := normName(in.Name)
	if err != nil {
		return AdminView{}, err
	}
	sem, err := normSemester(in.Semester)
	if err != nil {
		return AdminView{}, err
	}
	if err := checkCapacity(in.Capacity); err != nil {
		return AdminView{}, err
	}
	fixed := ""
	if in.JoinCode != "" {
		if s.Production {
			return AdminView{}, invalid("join_code", "NOT_ALLOWED", "Không đặt được mã tham gia ở môi trường chạy thật.")
		}
		if !joinCodeRE.MatchString(in.JoinCode) {
			return AdminView{}, invalid("join_code", "FORMAT", "Mã tham gia gồm 7 ký tự hợp lệ.")
		}
		fixed = in.JoinCode
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AdminView{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	var c store.Course
	for attempt := 0; ; attempt++ {
		if attempt >= maxCodeAttempts {
			return AdminView{}, ErrCodeExhausted
		}
		code := fixed
		if code == "" {
			if code, err = s.newCode(); err != nil {
				return AdminView{}, err
			}
		}
		var cap32 *int32
		if in.Capacity != nil {
			v := int32(*in.Capacity)
			cap32 = &v
		}
		c, err = q.InsertCourse(ctx, store.InsertCourseParams{SubjectCode: subject, ClassCode: class, Name: name, Semester: sem, Capacity: cap32, JoinCode: code, CreatedBy: actor})
		if err == nil {
			break
		}
		if errors.Is(err, pgx.ErrNoRows) { // trùng join_code
			if fixed != "" {
				return AdminView{}, invalid("join_code", "TAKEN", "Mã tham gia này đã có.")
			}
			continue
		}
		if isUnique(err, "courses_class_code_key") {
			return AdminView{}, ErrClassCodeTaken
		}
		return AdminView{}, fmt.Errorf("course: tạo lớp: %w", err)
	}

	if in.TeacherID != nil || len(in.TAIDs) > 0 {
		ta := in.TAIDs
		if ta == nil {
			ta = []uuid.UUID{}
		}
		if _, err := s.assignTx(ctx, tx, c, actor, in.TeacherID, &ta); err != nil {
			return AdminView{}, err
		}
	}
	if err := s.audit(ctx, q, c.ID, actor, c.ID.String(), "course_created", nil, map[string]any{"class_code": c.ClassCode, "subject_code": c.SubjectCode, "semester": c.Semester}); err != nil {
		return AdminView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminView{}, fmt.Errorf("course: commit tạo lớp: %w", err)
	}
	return s.AdminGet(ctx, c.ID)
}

func (s Service) newCode() (string, error) {
	if s.genCode != nil {
		return s.genCode()
	}
	return NewJoinCode()
}

// UpdateInput là thân PUT /admin/courses/{id}; con trỏ nil = không đổi. Capacity: SetCapacity=true để đặt (nil = bỏ giới hạn).
type UpdateInput struct {
	Version                                int
	SubjectCode, ClassCode, Name, Semester *string
	SetCapacity                            bool
	Capacity                               *int
}

// Update sửa tên / mã / học kỳ / sĩ số. Lớp đã lưu trữ ⇒ ErrArchived; sai version ⇒ *VersionConflictError.
func (s Service) Update(ctx context.Context, actor, id uuid.UUID, in UpdateInput) (AdminView, error) {
	p := store.UpdateCourseParams{ID: id, Version: int32(in.Version), SetCapacity: in.SetCapacity}
	var err error
	norm := func(v *string, f func(string) (string, error), dst **string) error {
		if v == nil {
			return nil
		}
		n, e := f(*v)
		if e != nil {
			return e
		}
		*dst = &n
		return nil
	}
	for _, step := range []error{
		norm(in.SubjectCode, normSubject, &p.SubjectCode), norm(in.ClassCode, normClass, &p.ClassCode),
		norm(in.Name, normName, &p.Name), norm(in.Semester, normSemester, &p.Semester), checkCapacity(in.Capacity),
	} {
		if step != nil {
			return AdminView{}, step
		}
	}
	if in.SetCapacity && in.Capacity != nil {
		v := int32(*in.Capacity)
		p.Capacity = &v
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AdminView{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockCourse(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminView{}, ErrNotFound
	}
	if err != nil {
		return AdminView{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if cur.Status == store.CourseStatusARCHIVED {
		return AdminView{}, ErrArchived
	}
	if int(cur.Version) != in.Version {
		now, gerr := s.adminGetWith(ctx, q, id)
		if gerr != nil {
			return AdminView{}, gerr
		}
		return AdminView{}, &VersionConflictError{Current: now}
	}
	if in.SetCapacity && in.Capacity != nil {
		n, err := q.CountActiveStudents(ctx, id)
		if err != nil {
			return AdminView{}, fmt.Errorf("course: đếm sinh viên: %w", err)
		}
		if int(n) > *in.Capacity {
			return AdminView{}, invalid("capacity", "CAPACITY_BELOW_ACTIVE", "Sĩ số không được nhỏ hơn số sinh viên đang học.")
		}
	}
	upd, err := q.UpdateCourse(ctx, p)
	if isUnique(err, "courses_class_code_key") {
		return AdminView{}, ErrClassCodeTaken
	}
	if err != nil {
		return AdminView{}, fmt.Errorf("course: cập nhật lớp: %w", err)
	}
	if err := s.audit(ctx, q, id, actor, id.String(), "course_updated", courseFacts(cur), courseFacts(upd)); err != nil {
		return AdminView{}, err
	}
	if err := emitChanged(ctx, tx, id); err != nil {
		return AdminView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminView{}, fmt.Errorf("course: commit cập nhật: %w", err)
	}
	return s.AdminGet(ctx, id)
}

func courseFacts(c store.Course) map[string]any {
	return map[string]any{"class_code": c.ClassCode, "subject_code": c.SubjectCode, "name": c.Name, "semester": c.Semester, "status": c.Status, "capacity": c.Capacity, "version": c.Version}
}

// Archive lưu trữ lớp: tắt mã tham gia ngay, thành viên giữ nguyên (chỉ còn đọc). Lần hai ⇒ trả nguyên trạng, không ghi gì.
func (s Service) Archive(ctx context.Context, actor, id uuid.UUID) (AdminView, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AdminView{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockCourse(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminView{}, ErrNotFound
	}
	if err != nil {
		return AdminView{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if cur.Status == store.CourseStatusACTIVE {
		upd, err := q.ArchiveCourse(ctx, id)
		if err != nil {
			return AdminView{}, fmt.Errorf("course: lưu trữ lớp: %w", err)
		}
		if err := s.audit(ctx, q, id, actor, id.String(), "course_archived", courseFacts(cur), courseFacts(upd)); err != nil {
			return AdminView{}, err
		}
		if err := emitChanged(ctx, tx, id); err != nil {
			return AdminView{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return AdminView{}, fmt.Errorf("course: commit lưu trữ: %w", err)
		}
	}
	return s.AdminGet(ctx, id)
}

// AdminFilter là query của GET /admin/courses (đã kiểm ở handler).
type AdminFilter struct {
	Status, Semester, Q string
	Cursor              *Cursor
	Limit               int
}

// AdminList trả tối đa Limit+1 lớp bằng MỘT truy vấn tổng hợp.
func (s Service) AdminList(ctx context.Context, f AdminFilter) ([]AdminView, error) {
	p := store.ListAdminCoursesParams{Lim: int32(f.Limit + 1)}
	if f.Status != "" {
		st := store.CourseStatus(f.Status)
		p.Status = &st
	}
	if f.Semester != "" {
		p.Semester = &f.Semester
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		name := "%" + likeEscape(auth.Fold(q)) + "%"
		code := likeEscape(strings.ToLower(q)) + "%"
		p.NameLike, p.CodeLike = &name, &code
	}
	if f.Cursor != nil {
		p.CurAt, p.CurID = &f.Cursor.At, &f.Cursor.ID
	}
	return s.adminRows(ctx, store.New(s.Pool), p)
}

// AdminGet đọc một lớp ở dạng danh sách của Admin.
func (s Service) AdminGet(ctx context.Context, id uuid.UUID) (AdminView, error) {
	return s.adminGetWith(ctx, store.New(s.Pool), id)
}

func (s Service) adminGetWith(ctx context.Context, q *store.Queries, id uuid.UUID) (AdminView, error) {
	rows, err := s.adminRows(ctx, q, store.ListAdminCoursesParams{ID: &id, Lim: 1})
	if err != nil {
		return AdminView{}, err
	}
	if len(rows) == 0 {
		return AdminView{}, ErrNotFound
	}
	return rows[0], nil
}

func (s Service) adminRows(ctx context.Context, q *store.Queries, p store.ListAdminCoursesParams) ([]AdminView, error) {
	rows, err := q.ListAdminCourses(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("course: liệt kê lớp: %w", err)
	}
	out := make([]AdminView, len(rows))
	for i, r := range rows {
		v := AdminView{ID: r.ID, ClassCode: r.ClassCode, SubjectCode: r.SubjectCode, Name: r.Name, Semester: r.Semester, Status: string(r.Status),
			AssistantsCount: int(r.AssistantsCount), StudentsActive: int(r.StudentsActive), StudentsPending: int(r.StudentsPending), Version: int(r.Version), CreatedAt: r.CreatedAt}
		if r.Capacity != nil {
			c := int(*r.Capacity)
			v.Capacity = &c
		}
		if r.TeacherID != nil && r.TeacherName != nil {
			v.Teacher = &Person{ID: *r.TeacherID, FullName: *r.TeacherName}
		}
		out[i] = v
	}
	return out, nil
}

// likeEscape thoát % _ \ để q của người dùng chỉ là chữ, không phải mẫu.
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Candidate là một trợ giảng có thể thêm vào lớp.
type Candidate struct {
	ID       uuid.UUID
	FullName string
	Email    string
}

// AssistantCandidates: người dùng vai TA đang ACTIVE / INVITED (≤ 20), tìm theo tên không dấu hoặc tiền tố email.
func (s Service) AssistantCandidates(ctx context.Context, q string) ([]Candidate, error) {
	p := store.ListAssistantCandidatesParams{}
	if q = strings.TrimSpace(q); q != "" {
		name := "%" + likeEscape(auth.Fold(q)) + "%"
		email := likeEscape(strings.ToLower(q)) + "%"
		p.NameLike, p.EmailLike = &name, &email
	}
	rows, err := store.New(s.Pool).ListAssistantCandidates(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("course: tìm trợ giảng: %w", err)
	}
	out := make([]Candidate, len(rows))
	for i, r := range rows {
		out[i] = Candidate{ID: r.ID, FullName: r.FullName, Email: r.Email}
	}
	return out, nil
}
