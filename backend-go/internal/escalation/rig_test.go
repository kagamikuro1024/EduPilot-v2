package escalation_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/testutil"
)

type fakeLock struct {
	mu   sync.Mutex
	lock *exam.Lock
}

func (l *fakeLock) IsLocked(context.Context, uuid.UUID) (exam.Lock, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lock == nil {
		return exam.Lock{}, false, nil
	}
	return *l.lock, true, nil
}

// rig: hai lớp, hai sinh viên (A, B), TA, giảng viên, admin; svc đã nối Postgres thật (testcontainers).
type rig struct {
	t                  *testing.T
	pool               *pgxpool.Pool
	svc                *escalation.Service
	lock               *fakeLock
	course, course2    uuid.UUID
	teacher, ta, admin uuid.UUID
	a, b               escalation.Actor
	sessA              uuid.UUID
}

func newRig(t *testing.T, cfg escalation.Config) *rig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	r := &rig{t: t, pool: pool, lock: &fakeLock{}}
	r.teacher, r.ta, r.admin = r.user("GV", "TEACHER"), r.user("TA", "TA"), r.user("AD", "ADMIN")
	r.course, r.course2 = r.newCourse("ACTIVE"), r.newCourse("ACTIVE")
	r.a = escalation.Actor{UserID: r.user("SV A", "STUDENT"), Role: auth.RoleStudent}
	r.b = escalation.Actor{UserID: r.user("SV B", "STUDENT"), Role: auth.RoleStudent}
	for _, c := range []uuid.UUID{r.course, r.course2} {
		r.enroll(c, r.a.UserID, "STUDENT", "ACTIVE")
		r.enroll(c, r.b.UserID, "STUDENT", "ACTIVE")
		r.enroll(c, r.teacher, "TEACHER", "ACTIVE")
		r.enroll(c, r.ta, "TA", "ACTIVE")
	}
	r.svc = &escalation.Service{Pool: pool, Members: course.Resolver{Pool: pool}, Lock: r.lock, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Cfg: cfg}
	r.sessA = r.session(r.course, r.a.UserID)
	return r
}

func (r *rig) exec(q string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), q, args...)
	require.NoError(r.t, err, q)
}

func (r *rig) count(q string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), q, args...).Scan(&n), q)
	return n
}

func (r *rig) user(name, role string) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, uuid.NewString()+"@example.test", name, role).Scan(&id))
	return id
}

func (r *rig) newCourse(status string) uuid.UUID {
	r.t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by, status, archived_at, join_enabled) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3, $4::course_status, case when $4::text = 'ARCHIVED' then now() end, $4::text = 'ACTIVE') returning id`,
		"EC"+id.String()[:8], string(jc), r.teacher, status).Scan(&c))
	return c
}

func (r *rig) enroll(course, user uuid.UUID, role, status string) {
	r.t.Helper()
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at) values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'ADMIN', case when $4 = 'REMOVED' then now() end)`, course, user, role, status)
}

func (r *rig) session(course, user uuid.UUID) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id`, course, user).Scan(&id))
	return id
}

// msg chèn một tin vào phiên; role USER hoặc ASSISTANT (ASSISTANT: DONE, trả lời `reply`).
func (r *rig) msg(course, sess, user uuid.UUID, role, content, intent string, reply *uuid.UUID) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, intent, reply_to, confidence)
		values ($1, $2, $3, $4::chat_role, $5, 'DONE', nullif($6, ''), $7, case when $4 = 'ASSISTANT' then 0.52 end) returning id`, course, sess, user, role, content, intent, reply).Scan(&id))
	return id
}

// pair chèn câu hỏi + câu trả lời của AI ở phiên của A; trả (hỏi, trả lời).
func (r *rig) pair(q, a string) (uuid.UUID, uuid.UUID) {
	r.t.Helper()
	u := r.msg(r.course, r.sessA, r.a.UserID, "USER", q, "", nil)
	return u, r.msg(r.course, r.sessA, r.a.UserID, "ASSISTANT", a, "COURSE_QA", &u)
}

func status(err error) int {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return ae.Status
	}
	if err == nil {
		return 0
	}
	return -1
}

func code(err error) string {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}
