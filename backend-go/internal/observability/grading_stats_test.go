package observability_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/observability"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/testutil"
)

// US-P7-07 AC10 / TC-P7-07-13 (DEF-R9S-06, PM quyết 2026-11-19): `/admin/observability/summary` có số liệu hàng chờ chấm bài — jobs_total theo run_state, queue_depth,
// dead_total (đã hết lượt thử) và thời lượng p50 / p95 — tính từ grading_results lúc đọc, không PII.
func TestSummaryGradingStats(t *testing.T) {
	testutil.RequireContainers(t)
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	svc := &observability.Service{Pool: pool, Clock: clock.NewFake(now)}

	var teacher, course, asg uuid.UUID
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ('gs.teacher@example.test', 'GV', 'TEACHER') returning id`).Scan(&teacher))
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', 'GS-1', 'ANM', '2026-2027-HK1', 'GSTATSM', $1) returning id`, teacher).Scan(&course))
	require.NoError(t, pool.QueryRow(t.Context(), `insert into assignments (course_id, title, max_score, due_at, rubric, submission_code, created_by, published_at)
		values ($1, 'BT', 10, now() + interval '3 days', '[{"id":"c1","name":"Một","max_score":"10","guidance":""}]'::jsonb, 'BT03', $2, now()) returning id`, course, teacher).Scan(&asg))

	// (run_state, error_code, giây chạy, đã duyệt?): 2 QUEUED, 1 RUNNING, 3 DONE chưa duyệt (10 s, 20 s, 30 s), 1 DONE đã duyệt (loại khỏi p50 / p95), 2 FAILED.
	rows := []struct {
		state string
		code  *string
		secs  int
		rev   bool
	}{{"QUEUED", nil, 0, false}, {"QUEUED", nil, 0, false}, {"RUNNING", nil, 0, false}, {"DONE", nil, 10, false}, {"DONE", nil, 20, false}, {"DONE", nil, 30, false}, {"DONE", nil, 900, true}, {"FAILED", ptr("INTERNAL"), 0, false}, {"FAILED", ptr("LLM_TIMEOUT"), 0, false}}
	for i, r := range rows {
		var stu, sub uuid.UUID
		require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, "gs."+uuid.NewString()+"@example.test").Scan(&stu))
		_, err := pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, course, stu)
		require.NoError(t, err)
		require.NoError(t, pool.QueryRow(t.Context(), `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at)
			values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.pdf', 1, 'application/pdf', 'x', 'MATCHED', 'IN_APP', now(), now()) returning id`, course, asg, stu).Scan(&sub))
		created := now.Add(-time.Hour)
		_, err = pool.Exec(t.Context(), `insert into grading_results (course_id, submission_id, run_state, status, error_code, created_at, updated_at, reviewed_at)
			values ($1, $2, $3, 'DRAFT', $4, $5::timestamptz, $5::timestamptz + make_interval(secs => $6), case when $7::boolean then $5::timestamptz end)`, course, sub, r.state, r.code, created, r.secs, r.rev)
		require.NoError(t, err, "hàng %d", i)
	}

	got, err := svc.Summary(t.Context(), "24h")
	require.NoError(t, err)
	g := got.Grading
	require.Equal(t, map[string]int64{"QUEUED": 2, "RUNNING": 1, "DONE": 4, "FAILED": 2}, g.JobsTotal)
	require.EqualValues(t, 2, g.QueueDepth)
	require.EqualValues(t, 2, g.DeadTotal)
	require.InDelta(t, 20, g.DurationP50S, 0.001)
	require.InDelta(t, 29, g.DurationP95S, 0.001) // nội suy tuyến tính giữa 20 s và 30 s
}

func ptr[T any](v T) *T { return &v }
