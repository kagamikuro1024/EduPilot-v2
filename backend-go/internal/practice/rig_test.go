package practice_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/practice"
	"github.com/edupilot/backend-go/internal/testutil"
)

// rig: Postgres thật đã migrate, một lớp ACTIVE có Giảng viên; sinh viên và câu hỏi tạo theo nhu cầu. Mỗi test lớp riêng nên chạy song song được.
type rig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	svc     *practice.Service
	ex      *exam.Service
	course  uuid.UUID
	teacher uuid.UUID
	sfx     string
}

func newRig(t *testing.T) *rig {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	r := &rig{t: t, pool: pool, svc: &practice.Service{Pool: pool}, ex: &exam.Service{Pool: pool}, sfx: strings.ToLower(uuid.NewString()[:8])}
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'Giảng Viên Thử', 'TEACHER') returning id`, "gv."+r.sfx+"@example.test").Scan(&r.teacher))
	r.course = r.newCourse("PR-" + strings.ToUpper(r.sfx))
	return r
}

func (r *rig) newCourse(code string) uuid.UUID {
	r.t.Helper()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	id := uuid.New()
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, code, string(jc), r.teacher).Scan(&c))
	return c
}

func (r *rig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

// student tạo sinh viên ACTIVE của lớp `course` (mặc định lớp chính).
func (r *rig) student(course ...uuid.UUID) uuid.UUID {
	r.t.Helper()
	c := r.course
	if len(course) > 0 {
		c = course[0]
	}
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into users (email, full_name, role) values ($1, 'Sinh Viên Thử', 'STUDENT') returning id`, "sv."+strings.ToLower(uuid.NewString()[:8])+"@example.test").Scan(&id))
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, c, id)
	return id
}

// approved tạo n câu MCQ_SINGLE đã duyệt cho chủ đề `topic` (đáp án đúng là phương án thứ hai, "B"); trả id câu.
func (r *rig) approved(topic string, n int, course ...uuid.UUID) []uuid.UUID {
	r.t.Helper()
	c := r.course
	if len(course) > 0 {
		c = course[0]
	}
	var ids []uuid.UUID
	for i := range n {
		d, err := r.ex.Create(r.t.Context(), r.teacher, c, exam.QuestionIn{Type: "MCQ_SINGLE", Title: fmt.Sprintf("%s %d", topic, i), Topic: topic, Difficulty: []string{"EASY", "MEDIUM", "HARD"}[i%3],
			Stem: fmt.Sprintf("Câu %s số %d?", topic, i), Options: []exam.OptionIn{{Body: "A"}, {Body: "B"}, {Body: "C"}}, Correct: []int{1}, Explanation: new("Vì B đúng.")})
		require.NoError(r.t, err)
		_, err = r.ex.Review(r.t.Context(), r.teacher, c, d.ID, "APPROVE", d.Version)
		require.NoError(r.t, err)
		ids = append(ids, d.ID)
	}
	return ids
}

// correctOption đọc id phương án ĐÚNG của một mục từ bản chụp (chỉ test mới được đọc).
func (r *rig) correctOption(item uuid.UUID) string {
	r.t.Helper()
	var id string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select answer_key->'option_ids'->>0 from practice_items where id = $1`, item).Scan(&id))
	return id
}

func (r *rig) wrongOption(item uuid.UUID) string {
	r.t.Helper()
	var id string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select o->>'id' from practice_items i, jsonb_array_elements(i.question->'options') o where i.id = $1 and o->>'id' <> i.answer_key->'option_ids'->>0 limit 1`, item).Scan(&id))
	return id
}

func opt(id string) practice.AnswerIn {
	return practice.AnswerIn{Answer: json.RawMessage(`{"option_ids":["` + id + `"]}`)}
}

func code(t *testing.T, err error) string {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "%v", err)
	return ae.Code
}

func status(t *testing.T, err error) int {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "%v", err)
	return ae.Status
}

func (r *rig) deadline(attempt uuid.UUID) (t0 time.Time) {
	r.t.Helper()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select deadline_at from practice_attempts where id = $1`, attempt).Scan(&t0))
	return t0
}

func outboxMsg(payload []byte) outbox.Message { return outbox.Message{Payload: payload} }
