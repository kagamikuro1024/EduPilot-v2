package seedtool_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/seedtool"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestSeedGuidance — SEED_GUIDANCE_DEMO: bật cờ, 8 kết quả ĐÃ DUYỆT với tiêu chí đầu thấp hơn AI 1,0 («sở thích gắt»), điểm tổng đúng, chạy lại không thêm gì.
func TestSeedGuidance(t *testing.T) {
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	sfx := strings.ToLower(uuid.NewString()[:8])
	var gv, course, asg uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, "gv."+sfx+"@example.test").Scan(&gv))
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	id := uuid.New()
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	require.NoError(t, pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "G-"+sfx, string(jc), gv).Scan(&course))
	rubric := `[{"id":"c1","name":"Một","max_score":"2.5","guidance":""},{"id":"c2","name":"Hai","max_score":"2.5","guidance":""},{"id":"c3","name":"Ba","max_score":"5","guidance":""}]`
	require.NoError(t, pool.QueryRow(ctx, `insert into assignments (course_id, title, description, type, max_score, due_at, rubric, created_by, submission_code, reference_answer)
		values ($1, 'BT03', 'x', 'ESSAY', 10, now(), $2::jsonb, $3, 'BT03', 'đáp án') returning id`, course, rubric, gv).Scan(&asg))
	for i := range 10 {
		var stu uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, "sv"+string(rune('a'+i))+"."+sfx+"@example.test").Scan(&stu))
		_, err := pool.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, course, stu)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
			values ($1, $2, $3, 'IN_APP', 1, 'k/'||gen_random_uuid()::text, 'a.txt', 10, 'text/plain', 'x', 'MATCHED', 'IN_APP')`, course, asg, stu)
		require.NoError(t, err)
	}
	n, err := seedtool.SeedGuidance(ctx, pool, course, "BT03")
	require.NoError(t, err)
	require.EqualValues(t, seedtool.GuidanceDemoResults, n)
	var on bool
	require.NoError(t, pool.QueryRow(ctx, `select enabled from course_guidance_flags where course_id = $1 and feature = 'GRADING_GUIDANCE'`, course).Scan(&on))
	require.True(t, on)
	var reviewed, strict int
	var ai, total string
	require.NoError(t, pool.QueryRow(ctx, `select count(*)::int, count(*) filter (where (criteria -> 0 ->> 'score')::numeric = (ai_scores ->> 'c1')::numeric - 1)::int, max(ai_total_before_late::text), max(total::text)
		from grading_results where course_id = $1 and status = 'REVIEWED'`, course).Scan(&reviewed, &strict, &ai, &total))
	require.Equal(t, 8, reviewed)
	require.Equal(t, 8, strict, "tiêu chí đầu luôn thấp hơn AI đúng 1,0")
	require.Equal(t, "8.00", ai) // 2,0 + 2,0 + 4,0
	require.Equal(t, "7.00", total)
	n, err = seedtool.SeedGuidance(ctx, pool, course, "BT03")
	require.NoError(t, err)
	require.Zero(t, n, "chạy lại không thêm gì")
	_, err = seedtool.SeedGuidance(ctx, pool, course, "KHONG-CO")
	require.Error(t, err)
}
