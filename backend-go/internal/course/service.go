// Package course: lớp học của người đăng nhập (FEAT-course-foundation US-P2-07) — tra quyền theo `enrollments`, danh sách lớp của tôi,
// chi tiết lớp. Tạo / sửa lớp, mã tham gia, thành viên là các story sau (US-P2-08…10).
package course

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Resolver cài auth.CourseResolver: MỘT truy vấn có chỉ mục `enrollments (course_id, user_id)`, không cache.
// Lớp không tồn tại và người ngoài lớp cho cùng kết quả (Found=false) nên guard trả cùng 403 — không lộ lớp có hay không.
type Resolver struct{ Pool *pgxpool.Pool }

// Resolve tra ghi danh. ADMIN không có ghi danh ⇒ Found=false (guard cho qua nhờ chế độ "OrAdmin").
func (r Resolver) Resolve(ctx context.Context, p auth.Principal, courseID uuid.UUID) (auth.Membership, error) {
	uid, err := uuid.Parse(p.Sub)
	if err != nil {
		return auth.Membership{}, nil // token không mang uuid người dùng: không thể có ghi danh
	}
	m, err := store.New(r.Pool).GetMembership(ctx, store.GetMembershipParams{CourseID: courseID, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Membership{}, nil
	}
	if err != nil {
		return auth.Membership{}, fmt.Errorf("course: tra ghi danh: %w", err)
	}
	return auth.Membership{Found: true, Role: auth.Role(m.RoleInCourse), Status: string(m.Status)}, nil
}

// Service là nghiệp vụ đọc lớp.
type Service struct {
	Pool *pgxpool.Pool
	// Production chặn việc đặt join_code cố định qua API (chỉ seed ở dev / test).
	Production bool
	// Clock là đồng hồ của mọi mốc nghiệp vụ (hết hạn mã, cửa sổ hoàn tác, giới hạn đoán mã); nil = đồng hồ thật.
	Clock clock.Clock
	// Redis giữ bộ đếm đoán mã dùng chung giữa các bản gateway; nil = không giới hạn (chỉ test).
	Redis *appredis.Client
	Log   *slog.Logger
	// PublicURL dựng `join_url` (APP_PUBLIC_URL, không có dấu / cuối).
	PublicURL string

	genCode func() (string, error) // thay được trong test để tiêm mã trùng; nil = NewJoinCode
	// warned: mốc log "Redis lỗi" gần nhất của giới hạn đoán mã, chung cho mọi yêu cầu (≤ 1 dòng / 30 s). NewService cấp; nil = theo từng yêu cầu.
	warned *atomic.Int64
}

// NewService là cách dựng Service cho gateway (cấp trạng thái dùng chung giữa các yêu cầu). Service value-type nên mọi bản sao chung con trỏ.
func NewService(s Service) Service {
	s.warned = new(atomic.Int64)
	return s
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}

// Brief là phần lớp trong danh sách "lớp của tôi". Không có join_code.
type Brief struct {
	ID          uuid.UUID
	ClassCode   string
	SubjectCode string
	Name        string
	Semester    string
	Status      string
}

// MyCourse là một mục của `GET /me/courses`.
type MyCourse struct {
	Course           Brief
	RoleInCourse     string
	EnrollmentStatus string
	EnrolledAt       time.Time
	EnrollmentID     uuid.UUID
}

// Cursor là dòng cuối của trang trước (created_at DESC, id DESC của enrollments).
type Cursor struct {
	At time.Time
	ID uuid.UUID
}

// MyCourses trả tối đa limit+1 dòng bằng MỘT truy vấn. Admin: rỗng (quản trị lớp ở /admin/courses).
func (s Service) MyCourses(ctx context.Context, userID uuid.UUID, role auth.Role, cur *Cursor, limit int) ([]MyCourse, error) {
	if role == auth.RoleAdmin {
		return nil, nil
	}
	p := store.ListMyCoursesParams{UserID: userID, Lim: int32(limit + 1)}
	if cur != nil {
		p.CurAt, p.CurID = &cur.At, &cur.ID
	}
	rows, err := store.New(s.Pool).ListMyCourses(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("course: liệt kê lớp của tôi: %w", err)
	}
	out := make([]MyCourse, len(rows))
	for i, r := range rows {
		out[i] = MyCourse{
			Course:       Brief{ID: r.ID, ClassCode: r.ClassCode, SubjectCode: r.SubjectCode, Name: r.Name, Semester: r.Semester, Status: string(r.CourseStatus)},
			RoleInCourse: string(r.RoleInCourse), EnrollmentStatus: string(r.EnrollmentStatus), EnrolledAt: r.EnrolledAt, EnrollmentID: r.EnrollmentID,
		}
	}
	return out, nil
}

// Detail là `GET /courses/{id}`.
type Detail struct {
	Brief
	Teachers []string
	MyRole   string
	Counts   *Counts // chỉ TA / Giảng viên
}

// Counts là số sinh viên của lớp.
type Counts struct{ Active, Pending int }

// ErrNotFound: lớp không tồn tại (sau khi guard đã cho qua — chỉ xảy ra với ADMIN).
var ErrNotFound = errors.New("course: không có lớp này")

// Get trả chi tiết lớp cho người đã qua guard. Sinh viên KHÔNG thấy danh sách thành viên hay mã tham gia; TA / Giảng viên thêm số đếm.
func (s Service) Get(ctx context.Context, id uuid.UUID, myRole string, staff bool) (Detail, error) {
	q := store.New(s.Pool)
	c, err := q.GetCourseBasic(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("course: đọc lớp: %w", err)
	}
	names, err := q.ListCourseTeacherNames(ctx, id)
	if err != nil {
		return Detail{}, fmt.Errorf("course: đọc giảng viên: %w", err)
	}
	d := Detail{Brief: Brief{ID: c.ID, ClassCode: c.ClassCode, SubjectCode: c.SubjectCode, Name: c.Name, Semester: c.Semester, Status: string(c.Status)}, Teachers: names, MyRole: myRole}
	if staff {
		n, err := q.CountCourseStudents(ctx, id)
		if err != nil {
			return Detail{}, fmt.Errorf("course: đếm sinh viên: %w", err)
		}
		d.Counts = &Counts{Active: int(n.Active), Pending: int(n.Pending)}
	}
	return d, nil
}
