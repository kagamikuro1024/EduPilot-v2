package chat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

// fakeAgent là chat.Responder giả: kịch bản do test đặt; đếm lời gọi; thấy ctx của "provider" bị huỷ.
type fakeAgent struct {
	fn        func(ctx context.Context, tc agent.TrustedContext, in agent.Input) (agent.Outcome, error)
	calls     atomic.Int32
	provAbort atomic.Bool
}

func (f *fakeAgent) Respond(ctx context.Context, tc agent.TrustedContext, in agent.Input) (agent.Outcome, error) {
	f.calls.Add(1)
	return f.fn(ctx, tc, in)
}

// stream dựng luồng sinh chữ giả: chờ gate (nếu có), rồi phát từng mảnh cách nhau gap; ctx huỷ → đánh dấu provAbort.
func (f *fakeAgent) stream(ctx context.Context, gate <-chan struct{}, gap time.Duration, resp llm.Response, pieces ...string) <-chan llm.Chunk {
	ch := make(chan llm.Chunk)
	go func() {
		defer close(ch)
		abort := func() { f.provAbort.Store(true) }
		if gate != nil {
			select {
			case <-gate:
			case <-ctx.Done():
				abort()
				return
			}
		}
		for _, p := range pieces {
			select {
			case <-time.After(gap):
			case <-ctx.Done():
				abort()
				return
			}
			select {
			case ch <- llm.Chunk{Text: p}:
			case <-ctx.Done():
				abort()
				return
			}
		}
		select {
		case ch <- llm.Chunk{Done: true, Response: &resp}:
		case <-ctx.Done():
			abort()
		}
	}()
	return ch
}

// say: tác tử trả lời đúng `pieces`.
func (f *fakeAgent) say(gap time.Duration, pieces ...string) {
	f.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: f.stream(ctx, nil, gap, llm.Response{}, pieces...)}, nil
	}
}

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

func (l *fakeLock) RecordChatBlocked(_ context.Context, _, attempt uuid.UUID) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.blocked = append(l.blocked, attempt)
	return nil
}

type rig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	svc     *chat.Service
	ag      *fakeAgent
	lock    *fakeLock
	course  uuid.UUID
	student chat.Actor
	other   chat.Actor
	sess    uuid.UUID
	logs    *bytes.Buffer
}

func newRig(t *testing.T, tweak ...func(*chat.Config)) *rig {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := &rig{t: t, pool: pool, rdb: rdb, ag: &fakeAgent{}, lock: &fakeLock{}, logs: &bytes.Buffer{}}
	teacher := r.user("GV", "TEACHER")
	r.course = r.newCourse(teacher, "ACTIVE")
	r.student = chat.Actor{UserID: r.user("SV A", "STUDENT"), Role: auth.RoleStudent, TraceID: "trace-a"}
	r.other = chat.Actor{UserID: r.user("SV B", "STUDENT"), Role: auth.RoleStudent}
	r.enroll(r.course, r.student.UserID, "STUDENT", "ACTIVE")
	r.enroll(r.course, r.other.UserID, "STUDENT", "ACTIVE")
	cfg := chat.Config{FlushEvery: 100 * time.Millisecond, StreamMax: 20 * time.Second}
	for _, tw := range tweak {
		tw(&cfg)
	}
	r.svc = r.service(cfg)
	r.ag.say(5*time.Millisecond, "Xin ", "chào ", "bạn.")
	r.sess = r.newSession(r.student, r.course)
	t.Cleanup(r.svc.Wait)
	return r
}

func (r *rig) service(cfg chat.Config) *chat.Service {
	return &chat.Service{
		Pool: r.pool, Redis: r.rdb, Agent: r.ag, Lock: r.lock, Members: course.Resolver{Pool: r.pool}, Clock: clock.Real{},
		Log: slog.New(slog.NewTextHandler(r.logs, &slog.HandlerOptions{Level: slog.LevelDebug})), Cfg: cfg,
		PII: &privacy.Detector{},
	}
}

func (r *rig) user(name, role string) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, uuid.NewString()+"@example.test", name, role).Scan(&id))
	return id
}

func (r *rig) newCourse(owner uuid.UUID, status string) uuid.UUID {
	r.t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by, status, archived_at, join_enabled) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3, $4::course_status, case when $4::text = 'ARCHIVED' then now() end, $4::text = 'ACTIVE') returning id`,
		"DC"+id.String()[:8], string(jc), owner, status).Scan(&c))
	return c
}

func (r *rig) enroll(course, user uuid.UUID, role, status string) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at) values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'ADMIN', case when $4 = 'REMOVED' then now() end)`, course, user, role, status)
	require.NoError(r.t, err)
}

func (r *rig) newSession(a chat.Actor, course uuid.UUID) uuid.UUID {
	r.t.Helper()
	s, err := r.svc.CreateSession(r.t.Context(), a, chat.CreateSessionIn{CourseID: course})
	require.NoError(r.t, err)
	return s.ID
}

// ---- thu khung SSE ----------------------------------------------------------------------------------------------------------------

type frame struct {
	ID, Ev string
	D      map[string]any
}

type col struct {
	mu sync.Mutex
	fs []frame
	ch chan struct{}
}

func newCol() *col { return &col{ch: make(chan struct{}, 4096)} }

func (c *col) Frame(id, ev string, data []byte) error {
	var d map[string]any
	_ = json.Unmarshal(data, &d)
	c.mu.Lock()
	c.fs = append(c.fs, frame{id, ev, d})
	c.mu.Unlock()
	c.ch <- struct{}{}
	return nil
}

func (c *col) all() []frame {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]frame(nil), c.fs...)
}

func (c *col) events() []string {
	var out []string
	for _, f := range c.all() {
		out = append(out, f.Ev)
	}
	return out
}

func (c *col) text() string {
	var b strings.Builder
	for _, f := range c.all() {
		if f.Ev == chat.EvToken || f.Ev == chat.EvSnapshot {
			b.WriteString(f.D["t"].(string))
		}
	}
	return b.String()
}

func (c *col) last() frame {
	fs := c.all()
	if len(fs) == 0 {
		return frame{}
	}
	return fs[len(fs)-1]
}

// send gửi một tin rồi đọc tới sự kiện cuối.
func (r *rig) send(a chat.Actor, sid uuid.UUID, text string) (*col, uuid.UUID, error) {
	r.t.Helper()
	m, err := r.svc.Send(r.t.Context(), a, sid, uuid.New(), text)
	if err != nil {
		return nil, uuid.Nil, err
	}
	c := newCol()
	require.NoError(r.t, r.svc.Tail(r.t.Context(), c, m, ""))
	return c, m.ID, nil
}

func (r *rig) row(mid uuid.UUID) (status string, content string, partial *string, errCode *string) {
	r.t.Helper()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select stream_status::text, content, partial_content, error_code from chat_messages where id=$1`, mid).Scan(&status, &content, &partial, &errCode))
	return
}

func (r *rig) count(q string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), q, args...).Scan(&n))
	return n
}

func hit(title, text string) rag.Hit {
	p := 3
	return rag.Hit{ChunkID: uuid.New(), DocumentID: uuid.New(), Title: title, PageNo: &p, Text: text}
}

type hitT = rag.Hit

// orphan chèn một cặp tin với ASSISTANT đang STREAMING mà KHÔNG có G sống (gateway đã chết).
func (r *rig) orphan(a chat.Actor) (user, asst uuid.UUID) {
	r.t.Helper()
	cm := uuid.New()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, client_msg_id) values ($1,$2,$3,'USER','mồ côi','DONE',$4) returning id`, r.course, r.sess, a.UserID, cm).Scan(&user))
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, stream_status, reply_to, partial_content) values ($1,$2,$3,'ASSISTANT','STREAMING',$4,'phần đầu') returning id`, r.course, r.sess, a.UserID, user).Scan(&asst))
	return
}

func httpStatus(err error) int {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return ae.Status
	}
	if err == nil {
		return 0
	}
	return -1
}

func apiCode(err error) string {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func pageOf(n int) httpx.PageParams { return httpx.PageParams{Limit: n} }

func cursorOf(t *testing.T, s string) httpx.Cursor {
	t.Helper()
	c, err := httpx.ParseCursor(s)
	require.NoError(t, err)
	return c
}
