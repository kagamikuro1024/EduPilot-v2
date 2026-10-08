package exam_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/testutil"
)

// rig: Postgres thật đã migrate, một lớp (ACTIVE) có Giảng viên + TA, service dựng trên đó. Mỗi test tạo lớp riêng nên chạy song song được.
type rig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	svc     *exam.Service
	course  uuid.UUID
	teacher uuid.UUID
	ta      uuid.UUID
}

func newRig(t *testing.T) *rig {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	r := &rig{t: t, pool: pool, svc: &exam.Service{Pool: pool, Jobs: jobs.NewService(pool)}}
	sfx := strings.ToLower(uuid.NewString()[:8])
	user := func(role string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'Người Thử', $2::user_role) returning id`, strings.ToLower(role)+"."+sfx+"@example.test", role).Scan(&id))
		return id
	}
	r.teacher, r.ta = user("TEACHER"), user("TA")
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	id := uuid.New()
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, "EX-"+strings.ToUpper(sfx), string(jc), r.teacher).Scan(&r.course))
	return r
}

func (r *rig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

// mcq tạo một câu MCQ_SINGLE hợp lệ.
func (r *rig) mcq(title string) exam.QuestionDetail {
	r.t.Helper()
	d, err := r.svc.Create(r.t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "MCQ_SINGLE", Title: title, Topic: "Số học", Difficulty: "EASY", Stem: "2+2=?",
		Options: []exam.OptionIn{{Body: "3"}, {Body: "4"}}, Correct: []int{1}})
	require.NoError(r.t, err)
	return d
}

// code tạo câu CODE với lời giải mẫu (chưa có test).
func (r *rig) code(title string) exam.QuestionDetail {
	r.t.Helper()
	d, err := r.svc.Create(r.t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "CODE", Title: title, Topic: "Cơ bản", Stem: "Đọc a b, in a+b."})
	require.NoError(r.t, err)
	d, err = r.svc.PutCode(r.t.Context(), r.course, d.ID, exam.CodeIn{Languages: []string{"cpp17"}, Reference: &exam.Reference{Language: "cpp17", Source: "int main(){}"}}, d.Version)
	require.NoError(r.t, err)
	return d
}

// test thêm một test tạo tay.
func (r *rig) test(qid uuid.UUID, name, in, want string, sample bool, weight int) exam.Testcase {
	r.t.Helper()
	tc, err := r.svc.AddTest(r.t.Context(), r.teacher, r.course, qid, exam.TestcaseIn{Name: new(name), Input: new(in), Expected: new(want), IsSample: new(sample), Weight: new(weight)}, true)
	require.NoError(r.t, err)
	return tc
}

// useIn gắn câu vào một bài thi (status `st`) để thử khoá sửa.
func (r *rig) useIn(qid uuid.UUID, st string) uuid.UUID {
	r.t.Helper()
	var eid uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into exams (course_id, title, status, opens_at, closes_at, duration_minutes, published_at, created_by)
		select $1, 'Bài thi ' || $2, $2::exam_status, case when $2 <> 'DRAFT' then now() - interval '2 hours' end, case when $2 <> 'DRAFT' then now() + interval '2 hours' end,
		       case when $2 <> 'DRAFT' then 30 end, case when $2 = 'PUBLISHED' then now() end, $3 returning id`, r.course, st, r.teacher).Scan(&eid))
	r.exec(`insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 1)`, r.course, eid, qid)
	return eid
}
