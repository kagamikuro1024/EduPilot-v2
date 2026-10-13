package notify_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

type pubFrame struct {
	user  string
	typ   string
	frame notify.Frame
}

// fakePub ghi lại khung SSE; fail → mọi Publish lỗi.
type fakePub struct {
	mu     sync.Mutex
	frames []pubFrame
	fail   bool
}

func (p *fakePub) Publish(_ context.Context, userID, eventType string, data any) (string, error) {
	if p.fail {
		return "", errors.New("redis chết")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frames = append(p.frames, pubFrame{user: userID, typ: eventType, frame: data.(notify.Frame)})
	return fmt.Sprint(len(p.frames)), nil
}

func (p *fakePub) all() []pubFrame {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]pubFrame(nil), p.frames...)
}

type rig struct {
	t      *testing.T
	pool   *pgxpool.Pool
	pub    *fakePub
	svc    *notify.Service
	course uuid.UUID
}

func newRig(t *testing.T) *rig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	return newRigAt(t, pool)
}

func newRigAt(t *testing.T, pool *pgxpool.Pool) *rig {
	t.Helper()
	pub := &fakePub{}
	r := &rig{t: t, pool: pool, pub: pub, svc: notify.New(pool, pub, nil)}
	teacher := r.user("GV", "TEACHER")
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`,
		"NT"+id.String()[:8], string(jc), teacher).Scan(&r.course))
	return r
}

func (r *rig) user(name, role string) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, uuid.NewString()+"@example.test", name, role).Scan(&id))
	return id
}

func (r *rig) count(q string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), q, args...).Scan(&n), q)
	return n
}

func (r *rig) exec(q string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), q, args...)
	require.NoError(r.t, err, q)
}

func spec(u uuid.UUID, course uuid.UUID, key string) notify.Spec {
	link := "/inbox"
	return notify.Spec{UserID: u, CourseID: &course, Type: "TICKET_ANSWERED", Title: "Có phản hồi", Link: &link, DedupeKey: key}
}

// TestNotificationsOnlyViaNotify — AC1 (kiến trúc): ngoài gói notify không có lời gọi NotifyInsert*/NotifyAgg* và không có câu `insert into notifications`
// (cả mã Go lẫn store/queries/*.sql); `insert into notifications` chỉ nằm ở notify.sql.
func TestNotificationsOnlyViaNotify(t *testing.T) {
	insert := regexp.MustCompile(`(?i)insert\s+into\s+notifications`)
	call := regexp.MustCompile(`\.(NotifyInsert|NotifyInsertMany|NotifyAggInsert|InsertNotification|ExamNotify\w+)\(`)
	var seen int
	err := filepath.WalkDir("../..", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "testdata" || n == "node_modules" || n == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		isGo, isSQL := strings.HasSuffix(p, ".go"), strings.HasSuffix(p, ".sql")
		if !isGo && !isSQL || strings.HasSuffix(p, "_test.go") || strings.HasSuffix(p, ".sql.go") || strings.Contains(p, "db/migrations/") {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		seen++
		inNotify := strings.Contains(filepath.ToSlash(p), "internal/notify/") || filepath.Base(p) == "notify.sql"
		if !inNotify && insert.Match(raw) {
			t.Errorf("%s có câu `insert into notifications` ngoài notify.sql", p)
		}
		if !inNotify && call.Match(raw) {
			t.Errorf("%s gọi truy vấn ghi thông báo trực tiếp (phải qua notify.Service)", p)
		}
		return nil
	})
	require.NoError(t, err)
	require.Greater(t, seen, 100, "quét thiếu tệp")
	raw, err := os.ReadFile("../store/queries/notify.sql")
	require.NoError(t, err)
	require.True(t, insert.Match(raw))
}

// queryCounter đếm câu INSERT vào notifications ở tầng pool.
type queryCounter struct{ n atomic.Int64 }

func (c *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if regexp.MustCompile(`(?i)insert\s+into\s+notifications`).MatchString(d.SQL) {
		c.n.Add(1)
	}
	return ctx
}
func (c *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TestSendManyOneStatement — AC1: thông báo cho 1.000 người = ĐÚNG MỘT câu INSERT; giao lại không tạo dòng thứ hai.
func TestSendManyOneStatement(t *testing.T) {
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	cfg, err := pgxpool.ParseConfig(url)
	require.NoError(t, err)
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	qc := &queryCounter{}
	cfg.ConnConfig.Tracer = qc
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	r := newRigAt(t, pool)
	r.exec(`insert into users (email, full_name, role) select 'bulk' || g || '.' || $1 || '@example.test', 'SV', 'STUDENT' from generate_series(1, 1000) g`, uuid.NewString()[:6])
	var ids []uuid.UUID
	rows, err := pool.Query(t.Context(), `select id from users where email like 'bulk%'`)
	require.NoError(t, err)
	for rows.Next() {
		var id uuid.UUID
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	require.Len(t, ids, 1000)

	qc.n.Store(0)
	sp := spec(uuid.Nil, r.course, "exam.scheduled:x")
	n, err := r.svc.SendMany(t.Context(), ids, sp)
	require.NoError(t, err)
	require.Equal(t, 1000, n)
	require.EqualValues(t, 1, qc.n.Load(), "1.000 người = 1 câu INSERT")
	require.Len(t, r.pub.all(), 1000, "một khung SSE cho mỗi dòng mới")
	again, err := r.svc.SendMany(t.Context(), ids, sp)
	require.NoError(t, err)
	require.Zero(t, again, "giao lại: không dòng mới")
	require.Equal(t, 1000, r.count(`select count(*) from notifications where dedupe_key like 'exam.scheduled:x:%'`))
	require.Len(t, r.pub.all(), 1000, "không phát lại khi giao lại")
}

func TestNotifyIdempotentRedelivery(t *testing.T) {
	r := newRig(t)
	u := r.user("SV", "STUDENT")
	for range 3 {
		_, err := r.svc.Send(t.Context(), spec(u, r.course, "escalation.answered:m1"))
		require.NoError(t, err)
	}
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id = $1`, u))
	fs := r.pub.all()
	require.Len(t, fs, 1, "đúng một khung SSE cho thông báo mới")
	require.Equal(t, "notification", fs[0].typ)
	require.False(t, fs[0].frame.Read)
	require.EqualValues(t, 1, fs[0].frame.UnreadCount)
}

func TestPersonalNotNotCoalesced(t *testing.T) {
	r := newRig(t)
	u := r.user("SV", "STUDENT")
	_, err := r.svc.Send(t.Context(), spec(u, r.course, "escalation.answered:a"))
	require.NoError(t, err)
	_, err = r.svc.Send(t.Context(), spec(u, r.course, "escalation.answered:b"))
	require.NoError(t, err)
	require.Equal(t, 2, r.count(`select count(*) from notifications where user_id = $1 and type = 'TICKET_ANSWERED'`, u))
}

func TestPublishFailureKeepsRow(t *testing.T) {
	r := newRig(t)
	r.pub.fail = true
	u := r.user("SV", "STUDENT")
	created, err := r.svc.Send(t.Context(), spec(u, r.course, "k"))
	require.NoError(t, err, "Redis lỗi lúc phát không làm hỏng luồng nghiệp vụ")
	require.True(t, created)
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id = $1`, u))
	require.EqualValues(t, 1, r.svc.PublishErrors.Load())
}

// ---- gộp (SRS 4.4.3) --------------------------------------------------------------------------------------------------------------

type counter struct{ n atomic.Int64 }

func (r *rig) agg(c *counter, rcpt ...uuid.UUID) notify.Agg {
	return notify.Agg{
		CourseID: r.course, Type: notify.TypeTicketNew, Recipients: rcpt, Link: "/inbox",
		Title: func(n int) string { return fmt.Sprintf("%d câu hỏi đang chờ giảng viên", n) },
		Count: func(context.Context, *store.Queries) (int, error) { return int(c.n.Load()), nil },
	}
}

func (r *rig) row(u uuid.UUID) (id uuid.UUID, title string, read bool) {
	r.t.Helper()
	var at *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select id, title, read_at::text from notifications where user_id = $1 and type = 'TICKET_NEW'`, u).Scan(&id, &title, &at))
	return id, title, at != nil
}

func TestCoalesceTicketNew(t *testing.T) {
	r := newRig(t)
	gv := r.user("GV", "TEACHER")
	var c counter
	var first uuid.UUID
	for i := 1; i <= 5; i++ {
		c.n.Store(int64(i))
		require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
		id, title, read := r.row(gv)
		if i == 1 {
			first = id
		}
		require.Equal(t, first, id, "dòng cập nhật tại chỗ, id giữ nguyên")
		require.Equal(t, fmt.Sprintf("%d câu hỏi đang chờ giảng viên", i), title)
		require.False(t, read)
	}
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id = $1`, gv), "một dòng chưa đọc")
	require.Len(t, r.pub.all(), 5, "mỗi lần số tăng phát một khung")
	require.Equal(t, "5 câu hỏi đang chờ giảng viên", r.pub.all()[4].frame.Title)
	// cùng số: không đổi, không phát
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	require.Len(t, r.pub.all(), 5)
}

func TestCoalesceReadThenNew(t *testing.T) {
	r := newRig(t)
	gv := r.user("GV", "TEACHER")
	var c counter
	c.n.Store(2)
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	id, _, _ := r.row(gv)
	r.exec(`update notifications set read_at = now() where id = $1`, id)
	c.n.Store(3)
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	_, title, read := r.row(gv)
	require.False(t, read, "ticket mới sau khi đã đọc → chưa đọc trở lại")
	require.Equal(t, "3 câu hỏi đang chờ giảng viên", title)
	require.EqualValues(t, 1, r.pub.all()[len(r.pub.all())-1].frame.UnreadCount)
}

func TestCoalesceDecreaseKeepsRead(t *testing.T) {
	r := newRig(t)
	gv := r.user("GV", "TEACHER")
	var c counter
	c.n.Store(3)
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	id, _, _ := r.row(gv)
	r.exec(`update notifications set read_at = now() where id = $1`, id)
	before := len(r.pub.all())
	var readAt string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select read_at::text from notifications where id = $1`, id).Scan(&readAt))
	c.n.Store(2)
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	_, title, read := r.row(gv)
	require.True(t, read, "số giảm không đặt lại read_at")
	require.Equal(t, "2 câu hỏi đang chờ giảng viên", title)
	var after string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select read_at::text from notifications where id = $1`, id).Scan(&after))
	require.Equal(t, readAt, after)
	require.Len(t, r.pub.all(), before, "không khung SSE mới khi số không tăng")
}

func TestCoalesceZeroMarksRead(t *testing.T) {
	r := newRig(t)
	gv := r.user("GV", "TEACHER")
	var c counter
	c.n.Store(2)
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	c.n.Store(0)
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, gv)))
	_, _, read := r.row(gv)
	require.True(t, read, "số = 0 → đã đọc")
	// số = 0 mà chưa có dòng: không tạo gì
	other := r.user("TA", "TA")
	require.NoError(t, r.svc.Aggregate(t.Context(), r.agg(&c, other)))
	require.Zero(t, r.count(`select count(*) from notifications where user_id = $1`, other))
}

// TestCoalesceParallelHandlers — hai handler song song, 50 lượt: số cuối = số ticket OPEN thật (đếm dưới khoá advisory).
func TestCoalesceParallelHandlers(t *testing.T) {
	r := newRig(t)
	gv := r.user("GV", "TEACHER")
	stu := r.user("SV", "STUDENT")
	agg := r.agg(&counter{}, gv)
	agg.Count = func(ctx context.Context, q *store.Queries) (int, error) {
		n, err := q.TicketCountOpenInCourse(ctx, r.course)
		return int(n), err
	}
	var wg sync.WaitGroup
	for w := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				_, err := r.pool.Exec(t.Context(), `insert into escalation_tickets (course_id, kind, origin, source_type, source_id, student_id) values ($1, 'TEACHER_OUTREACH', 'STAFF_OUTREACH', 'STUDENT', $2, $2)`, r.course, stu)
				require.NoError(t, err)
				require.NoError(t, r.svc.Aggregate(t.Context(), agg), "handler %d", w)
			}
		}()
	}
	wg.Wait()
	_, title, _ := r.row(gv)
	require.Equal(t, "50 câu hỏi đang chờ giảng viên", title)
}
