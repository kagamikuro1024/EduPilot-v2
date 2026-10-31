package assignment_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/testutil"
)

// lastQuery ghi câu SQL + tham số của hàng chờ (tracer pgx) để chạy EXPLAIN đúng câu thật.
type lastQuery struct {
	mu   sync.Mutex
	sql  string
	args []any
}

func (l *lastQuery) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if strings.Contains(d.SQL, "from submissions s") && strings.Contains(d.SQL, "rank") {
		l.mu.Lock()
		l.sql, l.args = d.SQL, append([]any(nil), d.Args...)
		l.mu.Unlock()
	}
	return ctx
}
func (l *lastQuery) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// seedCourses dựng `courses` lớp, mỗi lớp `perCourse` bài nộp MATCHED (mỗi bài một sinh viên) + kết quả DRAFT, 1/10 có cờ.
func (e *env) seedCourses(courses, perCourse int) {
	e.t.Helper()
	e.exec(`
	with cs as (select gen_random_uuid() as id from generate_series(1, $1::int)),
	ins_c as (
	  insert into courses (id, subject_code, class_code, name, semester, join_code, created_by)
	  select id, 'INT1006', 'P-' || substr(id::text, 1, 8), 'ANM', '2026-2027-HK1', translate(upper(substr(md5(id::text), 1, 7)), '01', 'XY'), $2 from cs returning id),
	ins_a as (
	  insert into assignments (course_id, title, max_score, due_at, rubric, submission_code, created_by, published_at)
	  select id, 'BT', 10, now() + interval '3 days', '[{"id":"c1","name":"Một","max_score":"10","guidance":""}]'::jsonb, 'BT03', $2, now() from ins_c returning id, course_id),
	pairs as (select a.id as aid, a.course_id as cid, gen_random_uuid() as uid, g from ins_a a cross join generate_series(1, $3::int) g),
	stu as (
	  insert into users (id, email, full_name, role)
	  select uid, 'perf.' || uid || '@example.test', 'Sinh Viên ' || g, 'STUDENT'::user_role from pairs returning id),
	enr as (
	  insert into enrollments (course_id, user_id, role_in_course, status, joined_via)
	  select cid, uid, 'STUDENT', 'ACTIVE', 'ADMIN' from pairs returning course_id),
	sub as (
	  insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at)
	  select cid, aid, uid, 'IN_APP', 1, 'k', 'a.pdf', 1, 'application/pdf', 'x', 'MATCHED', 'IN_APP', now() - (random() * interval '2 days'), now() - (random() * interval '2 days')
	  from pairs returning id, course_id)
	insert into grading_results (course_id, submission_id, run_state, status, criteria, total, flags)
	select course_id, id, 'DONE', 'DRAFT', '[{"id":"c1","score":"5"}]'::jsonb, 5, case when random() < 0.1 then array['NEEDS_REVIEW'] else '{}'::text[] end from sub`,
		courses, e.teacher, perCourse)
}

// TestQueueP95Under300ms — AC1: p95 ≤ 300 ms ở lớp 1.000 bài (kiểm trên nhiều lớp).
func TestQueueP95Under300ms(t *testing.T) {
	testutil.SkipTiming(t)
	e := newEnv(t)
	e.seedCourses(1, 1000)
	var course uuid.UUID
	e.one(&course, `select course_id from submissions limit 1`)
	e.exec(`analyze submissions; analyze grading_results; analyze enrollments`)
	var ds []time.Duration
	for range 30 {
		start := time.Now()
		p, err := e.svc.Queue(e.ctx, course, assignment.QueueFilter{Limit: 100})
		require.NoError(t, err)
		require.Len(t, p.Items, 100)
		ds = append(ds, time.Since(start))
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	require.Less(t, ds[len(ds)*95/100], 300*time.Millisecond)
}

// TestQueueExplainNoSeqScan — AC1: EXPLAIN của hàng chờ (mặc định và lọc cờ) không Seq Scan trên submissions / grading_results khi mỗi lớp chỉ chiếm ≤ 5 % dòng.
func TestQueueExplainNoSeqScan(t *testing.T) {
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	cfg, err := pgxpool.ParseConfig(url)
	require.NoError(t, err)
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	lq := &lastQuery{}
	cfg.ConnConfig.Tracer = lq
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	e := newEnvPool(t, pool)
	e.seedCourses(40, 100)
	e.exec(`analyze submissions; analyze grading_results; analyze enrollments; analyze assignments; analyze users`)
	var course uuid.UUID
	e.one(&course, `select course_id from submissions limit 1`)
	needs := true
	for name, f := range map[string]assignment.QueueFilter{"mặc định": {}, "cần xem kỹ": {NeedsReview: &needs}, "trang sau": {Limit: 20}} {
		_, err := e.svc.Queue(e.ctx, course, f)
		require.NoError(t, err, name)
		lq.mu.Lock()
		sql, args := lq.sql, lq.args
		lq.mu.Unlock()
		require.NotEmpty(t, sql)
		var plan []byte
		require.NoError(t, pool.QueryRow(e.ctx, "explain (format json) "+sql, args...).Scan(&plan), name)
		var parsed []struct {
			Plan json.RawMessage `json:"Plan"`
		}
		require.NoError(t, json.Unmarshal(plan, &parsed))
		var walk func(raw json.RawMessage)
		walk = func(raw json.RawMessage) {
			var n struct {
				NodeType string            `json:"Node Type"`
				Relation string            `json:"Relation Name"`
				Plans    []json.RawMessage `json:"Plans"`
			}
			require.NoError(t, json.Unmarshal(raw, &n))
			if n.NodeType == "Seq Scan" {
				require.NotContains(t, []string{"submissions", "grading_results"}, n.Relation, fmt.Sprintf("%s: Seq Scan trên %s", name, n.Relation))
			}
			for _, c := range n.Plans {
				walk(c)
			}
		}
		walk(parsed[0].Plan)
	}
}
