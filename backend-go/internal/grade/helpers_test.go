package grade_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

// tdb là cơ sở dữ liệu thật (Postgres container, migration đầy đủ) kèm vài hàm dựng dữ liệu bằng SQL thẳng.
type tdb struct {
	t    *testing.T
	ctx  context.Context
	pool *pgxpool.Pool
	q    *store.Queries
}

func newTDB(t *testing.T) *tdb {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	return &tdb{t: t, ctx: t.Context(), pool: pool, q: store.New(pool)}
}

func (d *tdb) exec(sql string, args ...any) {
	d.t.Helper()
	_, err := d.pool.Exec(d.ctx, sql, args...)
	require.NoError(d.t, err, sql)
}

func (d *tdb) one(dst any, sql string, args ...any) {
	d.t.Helper()
	require.NoError(d.t, d.pool.QueryRow(d.ctx, sql, args...).Scan(dst), sql)
}

func (d *tdb) user(role, name string) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, strings.ToLower(name)+"."+uuid.NewString()[:8]+"@example.test", name, role)
	return id
}

const joinAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func (d *tdb) course(teacher uuid.UUID, class string) uuid.UUID {
	d.t.Helper()
	id := uuid.New()
	b := make([]byte, 7)
	for i := range b {
		b[i] = joinAlphabet[int(id[i])%len(joinAlphabet)]
	}
	var c uuid.UUID
	d.one(&c, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, $1, '2026-2027-HK1', $2, $3) returning id`, class+"-"+id.String()[:6], string(b), teacher)
	d.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, c, teacher)
	return c
}

// student tạo sinh viên và ghi danh ACTIVE vào lớp với MSSV `code` (ảnh chụp do giảng viên xác lập).
func (d *tdb) student(course uuid.UUID, name, code string) uuid.UUID {
	d.t.Helper()
	u := d.user("STUDENT", name)
	d.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN', $3)`, course, u, nullIfEmpty(code))
	return u
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (d *tdb) session(course uuid.UUID, no int) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2::int, now() + $2::int * interval '1 day', now() + $2::int * interval '1 day' + interval '2 hours') returning id`, course, no)
	return id
}

func (d *tdb) attend(course, session, student, teacher uuid.UUID, status string) {
	d.t.Helper()
	d.exec(`insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, $4::attendance_status, $5)`, course, session, student, status, teacher)
}

func (d *tdb) part(course, student, teacher uuid.UUID, typ, points string) {
	d.t.Helper()
	d.exec(`insert into participation_events (course_id, student_id, type, points, client_event_id, created_by) values ($1, $2, $3::participation_type, $4::numeric, $5, $6)`, course, student, typ, points, uuid.New(), teacher)
}

// exam tạo bài thi lớp `course`; status PUBLISHED / OPEN; max là thang của bài thi.
func (d *tdb) exam(course, teacher uuid.UUID, status, max string) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into exams (course_id, title, created_by, status, max_score, opens_at, closes_at, duration_minutes, published_at)
		values ($1, 'Kiểm tra tuần', $2, $3::exam_status, $4::numeric, now() - interval '3 hours', now() - interval '1 hour', 30, case when $3 = 'PUBLISHED' then now() end) returning id`, course, teacher, status, max)
	return id
}

// attempt thêm lượt GRADED (auto, adjusted tuỳ chọn: "" = không sửa).
func (d *tdb) attempt(course, exam, student uuid.UUID, auto, adjusted string) {
	d.t.Helper()
	d.exec(`insert into exam_attempts (course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason, auto_score, graded_at, adjusted_score, adjusted_reason)
		values ($1, $2, $3, 'GRADED', now() - interval '2 hours', now() - interval '90 minutes', now() - interval '100 minutes', 'MANUAL', $4::numeric, now() - interval '80 minutes', $5::numeric, case when $5::numeric is null then null else 'Chấm lại' end)`,
		course, exam, student, auto, nullIfEmpty(adjusted))
}

// scheme tạo công thức (DRAFT) với MỘT thành phần `key` và các item; trả id công thức + id thành phần.
func (d *tdb) scheme(course, teacher uuid.UUID, no int) (scheme, comp uuid.UUID) {
	d.t.Helper()
	d.one(&scheme, `insert into grade_schemes (course_id, scheme_no, created_by) values ($1, $2, $3) returning id`, course, no, teacher)
	d.one(&comp, `insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ($1, $2, 'weekly', 'Kiểm tra hằng tuần', 'QT', 20) returning id`, course, scheme)
	return scheme, comp
}

func (d *tdb) examItem(course, scheme, comp, exam uuid.UUID, key, max string) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into grade_items (course_id, scheme_id, component_id, key, name, max_score, source, exam_id) values ($1, $2, $3, $4, $4, $5::numeric, 'EXAM', $6) returning id`, course, scheme, comp, key, max, exam)
	return id
}

func (d *tdb) manualItem(course, scheme, comp uuid.UUID, key, max string, published bool) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into grade_items (course_id, scheme_id, component_id, key, name, max_score, published_at) values ($1, $2, $3, $4, $4, $5::numeric, case when $6 then now() end) returning id`, course, scheme, comp, key, max, published)
	return id
}

func (d *tdb) entry(course, item, student, teacher uuid.UUID, score string) {
	d.t.Helper()
	d.exec(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ($1, $2, $3, $4::numeric, $5)`, course, item, student, score, teacher)
}
