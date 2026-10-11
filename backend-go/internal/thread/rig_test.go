package thread_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/edupilot/backend-go/internal/thread"
)

type fakeLock struct {
	mu      sync.Mutex
	lock    *exam.Lock
	err     error
	blocked []uuid.UUID
}

func (l *fakeLock) IsLocked(context.Context, uuid.UUID) (exam.Lock, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return exam.Lock{}, false, l.err
	}
	if l.lock == nil {
		return exam.Lock{}, false, nil
	}
	return *l.lock, true, nil
}

func (l *fakeLock) RecordChatBlocked(_ context.Context, _, att uuid.UUID) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.blocked = append(l.blocked, att)
	return nil
}

type fakeRag struct {
	hits []rag.Hit
	err  error
}

func (f *fakeRag) SearchStudent(context.Context, rag.Query) ([]rag.Hit, error) { return f.hits, f.err }

// fakeLLM là llm.Client giả: ghi lại làn và số lần gọi Chat.
type fakeLLM struct {
	llm.Client
	calls atomic.Int32
	lanes []llm.Lane
	text  string
	err   error
	deg   bool
	mu    sync.Mutex
}

func (f *fakeLLM) Chat(_ context.Context, r llm.Request) (llm.Response, error) {
	f.calls.Add(1)
	f.mu.Lock()
	if r.Lane != nil {
		f.lanes = append(f.lanes, *r.Lane)
	} else {
		f.lanes = append(f.lanes, llm.LaneInteractive)
	}
	f.mu.Unlock()
	return llm.Response{Text: f.text, Degraded: f.deg}, f.err
}

type rig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	svc     *thread.Service
	lock    *fakeLock
	rag     *fakeRag
	llm     *fakeLLM
	course  uuid.UUID
	teacher thread.Actor
	ta      thread.Actor
	sv      thread.Actor // Vũ Hoàng Giang, MSSV 20229001
	other   thread.Actor // Ngô Ngọc Cẩm
}

const (
	svName  = "Vũ Hoàng Giang"
	svCode  = "20229001"
	svEmail = "giang.vh@sv.edu.vn"
)

func newRig(t *testing.T) *rig {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool := testutil.RuntimePool(t) // đúng cấu hình runtime (QueryExecModeExec)
	var err error
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := &rig{t: t, pool: pool, rdb: rdb, lock: &fakeLock{}, rag: &fakeRag{}, llm: &fakeLLM{text: "Theo quy chế [1], sinh viên được thi lại một lần."}}
	gv := r.user("Phạm Quang Huy", "TEACHER", "")
	r.course = r.newCourse(gv)
	r.teacher = thread.Actor{UserID: gv, Role: auth.RoleTeacher}
	r.enroll(gv, "TEACHER", "")
	r.ta = thread.Actor{UserID: r.user("Lý Thu Trang", "TA", ""), Role: auth.RoleTA}
	r.enroll(r.ta.UserID, "TA", "")
	r.sv = thread.Actor{UserID: r.user(svName, "STUDENT", svEmail), Role: auth.RoleStudent}
	r.enroll(r.sv.UserID, "STUDENT", svCode)
	r.other = thread.Actor{UserID: r.user("Ngô Ngọc Cẩm", "STUDENT", "cam.nn@sv.edu.vn"), Role: auth.RoleStudent}
	r.enroll(r.other.UserID, "STUDENT", "20229002")
	log := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: pool}, Redis: rdb, Log: log}}
	cl := &privacy.Classifier{Detector: det}
	r.svc = &thread.Service{Pool: pool, Redis: rdb, FW: &thread.Firewall{Detector: det, Classifier: cl}, Lock: r.lock, Jobs: jobs.NewService(pool),
		Rag: r.rag, LLM: r.llm, Clock: clock.Real{}, Log: log}
	return r
}

func (r *rig) user(name, role, email string) uuid.UUID {
	r.t.Helper()
	if email == "" {
		email = uuid.NewString() + "@example.test"
	}
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, email, name, role).Scan(&id))
	return id
}

func (r *rig) newCourse(owner uuid.UUID) uuid.UUID {
	r.t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "TH"+id.String()[:8], string(jc), owner).Scan(&c))
	return c
}

func (r *rig) enroll(user uuid.UUID, role, code string) {
	r.t.Helper()
	var snap *string
	if code != "" {
		snap = &code
	}
	_, err := r.pool.Exec(r.t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, $3::enrollment_role, 'ACTIVE', 'ADMIN', $4)`, r.course, user, role, snap)
	require.NoError(r.t, err)
}

func (r *rig) create(a thread.Actor, title, body string, redact bool) (thread.ThreadView, error) {
	return r.svc.Create(r.t.Context(), a, r.course, thread.CreateIn{Title: title, Body: body, Redact: redact})
}

func (r *rig) mustCreate(a thread.Actor, title, body string) uuid.UUID {
	r.t.Helper()
	v, err := r.create(a, title, body, false)
	require.NoError(r.t, err)
	return v.Thread.ID
}

func (r *rig) count(q string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), q, args...).Scan(&n))
	return n
}

func hit(cos float64) rag.Hit {
	p := 4
	return rag.Hit{ChunkID: uuid.New(), DocumentID: uuid.New(), Title: "Quy chế học vụ", PageNo: &p, Text: "Sinh viên được thi lại một lần.", Cosine: cos}
}

func code(err error) string {
	var ae *apierr.Error
	if asAPI(err, &ae) {
		return ae.Code
	}
	return ""
}

func status(err error) int {
	var ae *apierr.Error
	if asAPI(err, &ae) {
		return ae.Status
	}
	if err == nil {
		return 0
	}
	return -1
}

var _ = agent.RagSimFloor

func asAPI(err error, target **apierr.Error) bool { return errors.As(err, target) }
