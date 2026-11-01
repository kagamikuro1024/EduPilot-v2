package assignment_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/testutil"
)

type env struct {
	t        *testing.T
	ctx      context.Context
	pool     *pgxpool.Pool
	svc      *assignment.Service
	course   uuid.UUID
	teacher  uuid.UUID
	students []uuid.UUID
	act      assignment.Actor
}

func newEnv(t *testing.T) *env {
	t.Helper()
	testutil.RequireContainers(t)
	return newEnvPool(t, testutil.RuntimePool(t))
}

func newEnvPool(t *testing.T, pool *pgxpool.Pool) *env {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	e := &env{t: t, ctx: t.Context(), pool: pool}
	e.svc = &assignment.Service{Pool: pool, Log: log, Notify: &notify.Service{Pool: pool, Log: log}}
	e.teacher = e.user("TEACHER")
	e.act = assignment.Actor{UserID: e.teacher}
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i])%len(alpha)]
	}
	e.one(&e.course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'ANM', '2026-2027-HK1', $2, $3) returning id`, "G-"+id.String()[:8], string(b), e.teacher)
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, e.course, e.teacher)
	for range 2 {
		s := e.user("STUDENT")
		e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, e.course, s)
		e.students = append(e.students, s)
	}
	return e
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	_, err := e.pool.Exec(e.ctx, sql, args...)
	require.NoError(e.t, err, sql)
}

func (e *env) one(dst any, sql string, args ...any) {
	e.t.Helper()
	require.NoError(e.t, e.pool.QueryRow(e.ctx, sql, args...).Scan(dst), sql)
}

func (e *env) user(role string) uuid.UUID {
	var id uuid.UUID
	e.one(&id, `insert into users (email, full_name, role) values ($1, 'Người Thử', $2::user_role) returning id`, strings.ToLower(role)+"."+uuid.NewString()[:8]+"@example.test", role)
	return id
}

var due = time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)

func in(code string) assignment.Input {
	return assignment.Input{Title: "BT03", SubmissionCode: code, DueAt: due, MaxScore: "10", Channels: []string{"IN_APP"}, LateMaxDays: 0,
		Rubric: []assignment.RubricIn{{Name: "Khái niệm", MaxScore: "4", Guidance: "Đúng định nghĩa"}, {Name: "Ví dụ", MaxScore: "6"}}}
}

func (e *env) create(code string) assignment.Assignment {
	e.t.Helper()
	a, err := e.svc.Create(e.ctx, e.act, e.course, in(code))
	require.NoError(e.t, err)
	return a
}

func (e *env) publish(a assignment.Assignment) {
	e.t.Helper()
	_, err := e.svc.Publish(e.ctx, e.act, e.course, a.ID)
	require.NoError(e.t, err)
}

// code trả mã lỗi apierr (rỗng nếu không phải).
func code(err error) string {
	if ae, ok := err.(*apierr.Error); ok { //nolint:errorlint // apierr.Error trả thẳng
		return ae.Code
	}
	return ""
}

func fieldOf(err error) string {
	if ae, ok := err.(*apierr.Error); ok { //nolint:errorlint // như trên
		if fs, ok := ae.Details.([]apierr.FieldError); ok && len(fs) > 0 {
			return fs[0].Field
		}
	}
	return ""
}

// result chèn một bài nộp MATCHED + kết quả cho sinh viên s.
func (e *env) result(a assignment.Assignment, s uuid.UUID, status string, submitted time.Time, criteria string, manual bool) uuid.UUID {
	e.t.Helper()
	var sub, res uuid.UUID
	e.one(&sub, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at)
		values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.pdf', 1, 'application/pdf', 'x', 'MATCHED', 'IN_APP', $4) returning id`, e.course, a.ID, s, submitted)
	pub := "null"
	if status == "PUBLISHED" {
		pub = "now()"
	}
	e.one(&res, `insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, late_penalty_manual, published_at)
		values ($1, $2, 'DONE', $3, $4::jsonb, 10, 10, $5, `+pub+`) returning id`, e.course, sub, status, criteria, manual)
	return res
}

const crit10 = `[{"id":"c1","score":"4"},{"id":"c2","score":"6"}]`

func TestCreateAssignment(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	require.Equal(t, "ESSAY", a.Type)
	require.Equal(t, "10", a.MaxScore)
	require.Equal(t, 7, a.AppealDays)
	require.Nil(t, a.PublishedAt)
	require.Equal(t, []string{"c1", "c2"}, []string{a.Rubric[0].ID, a.Rubric[1].ID})
	var n int
	e.one(&n, `select count(*) from audit_log where entity = 'assignments' and entity_id = $1 and action = 'create'`, a.ID.String())
	require.Equal(t, 1, n)
}

func TestSubmissionCodeUnique(t *testing.T) {
	e := newEnv(t)
	e.create("BT03")
	_, err := e.svc.Create(e.ctx, e.act, e.course, in("BT03"))
	require.Equal(t, "VALIDATION_FAILED", code(err))
	require.Equal(t, "submission_code", fieldOf(err))
}

func TestCreateValidation(t *testing.T) {
	e := newEnv(t)
	cases := map[string]struct {
		mut  func(*assignment.Input)
		want string
		fld  string
	}{
		"tiêu đề rỗng":        {func(i *assignment.Input) { i.Title = "  " }, "VALIDATION_FAILED", "title"},
		"mã nộp thư thường":   {func(i *assignment.Input) { i.SubmissionCode = "bt03" }, "VALIDATION_FAILED", "submission_code"},
		"mã nộp thư 1 ký tự":  {func(i *assignment.Input) { i.SubmissionCode = "A" }, "VALIDATION_FAILED", "submission_code"},
		"điểm tối đa 0":       {func(i *assignment.Input) { i.MaxScore = "0" }, "VALIDATION_FAILED", "max_score"},
		"điểm tối đa 101":     {func(i *assignment.Input) { i.MaxScore = "101" }, "VALIDATION_FAILED", "max_score"},
		"nộp muộn thiếu ngày": {func(i *assignment.Input) { i.AllowLate = true }, "VALIDATION_FAILED", "late_max_days"},
		"trừ muộn sai bước":   {func(i *assignment.Input) { i.LatePenaltyPerDay = "0.07" }, "VALIDATION_FAILED", "late_penalty_per_day"},
		"phúc khảo 31 ngày":   {func(i *assignment.Input) { d := 31; i.AppealDays = &d }, "VALIDATION_FAILED", "appeal_days"},
		"kênh lạ":             {func(i *assignment.Input) { i.Channels = []string{"FAX"} }, "VALIDATION_FAILED", "channels"},
		"Teams thiếu mã":      {func(i *assignment.Input) { i.Channels = []string{"TEAMS"} }, "VALIDATION_FAILED", "teams_class_id"},
		"thiếu hạn":           {func(i *assignment.Input) { i.DueAt = time.Time{} }, "VALIDATION_FAILED", "due_at"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			i := in("BT" + uuid.NewString()[:4])
			i.SubmissionCode = strings.ToUpper(i.SubmissionCode)
			c.mut(&i)
			_, err := e.svc.Create(e.ctx, e.act, e.course, i)
			require.Equal(t, c.want, code(err), name)
			require.Equal(t, c.fld, fieldOf(err), name)
		})
	}
}

func TestQuizTypeRejected(t *testing.T) {
	e := newEnv(t)
	i := in("BT03")
	i.Type = "QUIZ"
	_, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.Equal(t, "TYPE_NOT_SUPPORTED", code(err))
}

func TestMaxScoreMustBeQuarterStep(t *testing.T) {
	e := newEnv(t)
	i := in("BT03")
	i.MaxScore = "10.1"
	_, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.Equal(t, "SCORE_STEP", code(err))
}

func TestRubricSumEqualsMax(t *testing.T) {
	_, err := assignment.ValidateRubric([]assignment.RubricIn{{Name: "A", MaxScore: "4"}, {Name: "B", MaxScore: "5.75"}}, decimal.NewFromInt(10))
	require.Equal(t, "RUBRIC_INVALID", code(err))
	ae := err.(*apierr.Error)                                       //nolint:errorlint,forcetypeassert // kiểm chi tiết tổng thật
	require.Equal(t, "9.75", ae.Details.(map[string]string)["sum"]) //nolint:errorlint,forcetypeassert // chi tiết là map[string]string
	// 0,1 + 0,2 + … không trôi như float: 0.25 × 40 = 10 đúng.
	many := make([]assignment.RubricIn, 0, 10)
	for range 10 {
		many = append(many, assignment.RubricIn{Name: "x", MaxScore: "1"})
	}
	_, err = assignment.ValidateRubric(many, decimal.NewFromInt(10))
	require.NoError(t, err)
}

func TestRubricStepQuarter(t *testing.T) {
	for _, bad := range []string{"0", "-1", "1.1", "abc", ""} {
		_, err := assignment.ValidateRubric([]assignment.RubricIn{{Name: "A", MaxScore: bad}}, decimal.NewFromInt(10))
		require.Equal(t, "VALIDATION_FAILED", code(err), bad)
	}
	_, err := assignment.ValidateRubric([]assignment.RubricIn{{Name: "A", MaxScore: "10.00"}}, decimal.NewFromInt(10))
	require.NoError(t, err)
}

func TestRubricCountBounds(t *testing.T) {
	_, err := assignment.ValidateRubric(nil, decimal.NewFromInt(10))
	require.Equal(t, "VALIDATION_FAILED", code(err))
	eleven := make([]assignment.RubricIn, 11)
	for i := range eleven {
		eleven[i] = assignment.RubricIn{Name: "x", MaxScore: "1"}
	}
	_, err = assignment.ValidateRubric(eleven, decimal.NewFromInt(11))
	require.Equal(t, "VALIDATION_FAILED", code(err))
}

func TestRubricIDsStable(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	// đổi thứ tự + thêm tiêu chí mới: id cũ giữ, id mới cấp số kế tiếp không trùng.
	out, err := e.svc.SetRubric(e.ctx, e.act, e.course, a.ID, []assignment.RubricIn{
		{ID: "c2", Name: "Ví dụ", MaxScore: "5"}, {Name: "Mới", MaxScore: "2"}, {ID: "c1", Name: "Khái niệm", MaxScore: "3"}}, a.Version)
	require.NoError(t, err)
	var ids []string
	for _, r := range out.Rubric {
		ids = append(ids, r.ID)
	}
	require.Equal(t, []string{"c2", "c3", "c1"}, ids)
}

func TestPublishAssignment(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	out, err := e.svc.Publish(e.ctx, e.act, e.course, a.ID)
	require.NoError(t, err)
	require.NotNil(t, out.Assignment.PublishedAt)
	require.Equal(t, 2, out.Students)
	var n int
	e.one(&n, `select count(*) from outbox where topic = $1 and payload->>'assignment_id' = $2`, assignment.TopicPublished, a.ID.String())
	require.Equal(t, 1, n)
	e.one(&n, `select count(*) from audit_log where entity_id = $1 and action = 'publish'`, a.ID.String())
	require.Equal(t, 1, n)
}

func TestPublishNeedsRubric(t *testing.T) {
	e := newEnv(t)
	i := in("BT03")
	i.Rubric = nil
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(t, err)
	_, err = e.svc.Publish(e.ctx, e.act, e.course, a.ID)
	require.Equal(t, "RUBRIC_REQUIRED", code(err))
	// đổi điểm tối đa sau khi có rubric (Σ lệch) cũng không công bố được
	b := e.create("BT04")
	i2 := in("BT04")
	i2.MaxScore, i2.Version = "8", b.Version
	_, err = e.svc.Update(e.ctx, e.act, e.course, b.ID, i2)
	require.NoError(t, err)
	_, err = e.svc.Publish(e.ctx, e.act, e.course, b.ID)
	require.Equal(t, "RUBRIC_REQUIRED", code(err))
}

func TestPublishTwice(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	_, err := e.svc.Publish(e.ctx, e.act, e.course, a.ID)
	require.Equal(t, "ALREADY_PUBLISHED", code(err))
	var n int
	e.one(&n, `select count(*) from outbox where topic = $1 and payload->>'assignment_id' = $2`, assignment.TopicPublished, a.ID.String())
	require.Equal(t, 1, n, "lần hai không sinh sự kiện")
}

func TestPublishNotifiesStudents(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	var payload []byte
	e.one(&payload, `select payload from outbox where topic = $1 and payload->>'assignment_id' = $2`, assignment.TopicPublished, a.ID.String())
	msg := outbox.Message{Topic: assignment.TopicPublished, Payload: payload}
	for range 2 { // chạy lại: khử trùng, không thêm chuông
		require.NoError(t, e.svc.HandlePublished(e.ctx, msg))
	}
	var n int
	e.one(&n, `select count(*) from notifications where type = $1 and course_id = $2`, assignment.TypePublished, e.course)
	require.Equal(t, 2, n)
	e.one(&n, `select count(*) from notifications where type = $1 and user_id = $2`, assignment.TypePublished, e.teacher)
	require.Zero(t, n, "giảng viên không nhận")
}

func TestUpdateAfterPublish(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	cur, err := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	require.NoError(t, err)
	i := in("BT03")
	i.Title, i.Version, i.DueAt = "BT03 (sửa)", cur.Version, due.Add(48*time.Hour)
	out, err := e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.NoError(t, err)
	require.Equal(t, "BT03 (sửa)", out.Title)
	require.True(t, out.DueAt.Equal(due.Add(48*time.Hour)))
	var n int
	e.one(&n, `select count(*) from outbox where topic = $1 and payload->>'late_changed' = 'true'`, assignment.TopicUpdated)
	require.Equal(t, 1, n)
}

func TestUpdateLockedFields(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	cur, _ := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	i := in("BT03")
	i.Version, i.MaxScore = cur.Version, "20"
	_, err := e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.Equal(t, "VALIDATION_FAILED", code(err))
	require.Equal(t, "max_score", fieldOf(err))
	i.MaxScore, i.SubmissionCode = "10", "KHAC"
	_, err = e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.Equal(t, "submission_code", fieldOf(err))
}

func TestRubricLockedAfterGrading(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	e.result(a, e.students[0], "DRAFT", due.Add(-time.Hour), crit10, false)
	cur, _ := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	require.True(t, cur.RubricLocked)
	_, err := e.svc.SetRubric(e.ctx, e.act, e.course, a.ID, []assignment.RubricIn{{Name: "Một", MaxScore: "10"}}, cur.Version)
	require.Equal(t, "RUBRIC_LOCKED", code(err))
}

func TestLateRuleLockedAfterPublish(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.publish(a)
	e.result(a, e.students[0], "PUBLISHED", due.Add(-time.Hour), crit10, false)
	cur, _ := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	require.True(t, cur.LateRuleLocked)
	i := in("BT03")
	i.Version, i.DueAt = cur.Version, due.Add(24*time.Hour)
	_, err := e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.Equal(t, "LATE_RULE_LOCKED", code(err))
	i.DueAt, i.Title = due, "Chỉ đổi tiêu đề"
	_, err = e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.NoError(t, err, "trường khác vẫn sửa được")
}

func TestUpdateVersionConflict(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	i := in("BT03")
	i.Version = a.Version + 5
	_, err := e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.Equal(t, "VERSION_CONFLICT", code(err))
	require.NotNil(t, err.(*apierr.Error).Details.(map[string]any)["current"]) //nolint:errorlint,forcetypeassert // bản hiện tại đi kèm
}

// lateSetup: bài cho nộp muộn (trừ 1 điểm/ngày, tối đa 3 ngày); sv0 nộp muộn 1,5 ngày.
func (e *env) lateSetup(status string, manual bool) (assignment.Assignment, uuid.UUID) {
	e.t.Helper()
	i := in("BT03")
	i.AllowLate, i.LatePenaltyPerDay, i.LateMaxDays = true, "1", 3
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(e.t, err)
	e.publish(a)
	res := e.result(a, e.students[0], status, due.Add(36*time.Hour), crit10, manual)
	return a, res
}

func (e *env) recompute(a assignment.Assignment, newDue time.Time) {
	e.t.Helper()
	cur, _ := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	i := in("BT03")
	i.AllowLate, i.LatePenaltyPerDay, i.LateMaxDays, i.Version, i.DueAt = true, "1", 3, cur.Version, newDue
	_, err := e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.NoError(e.t, err)
	require.NoError(e.t, e.svc.RecomputeLate(e.ctx, e.course, a.ID))
}

func (e *env) late(res uuid.UUID) (days int, penalty, total decimal.Decimal, status string) {
	e.one(&days, `select late_days from grading_results where id = $1`, res)
	e.one(&penalty, `select late_penalty from grading_results where id = $1`, res)
	e.one(&total, `select total from grading_results where id = $1`, res)
	e.one(&status, `select status from grading_results where id = $1`, res)
	return
}

func TestDueChangeRecomputesLateBeforePublish(t *testing.T) {
	e := newEnv(t)
	a, res := e.lateSetup("REVIEWED", false)
	e.recompute(a, due.Add(-24*time.Hour)) // lùi hạn → nộp muộn 2,5 ngày → 3 ngày, trừ 3
	days, pen, total, status := e.late(res)
	require.Equal(t, 3, days)
	require.True(t, pen.Equal(decimal.NewFromInt(3)), pen.String())
	require.True(t, total.Equal(decimal.NewFromInt(7)), total.String())
	require.Equal(t, "DRAFT", status, "REVIEWED có tổng đổi quay về DRAFT")
}

func TestDueExtensionClearsLatePenalty(t *testing.T) {
	e := newEnv(t)
	a, res := e.lateSetup("DRAFT", false)
	e.recompute(a, due.Add(7*24*time.Hour)) // gia hạn: nộp trước hạn mới
	days, pen, total, status := e.late(res)
	require.Zero(t, days)
	require.True(t, pen.IsZero())
	require.True(t, total.Equal(decimal.NewFromInt(10)))
	require.Equal(t, "DRAFT", status)
}

func TestLatePenaltyManualKept(t *testing.T) {
	e := newEnv(t)
	a, res := e.lateSetup("DRAFT", true)
	e.recompute(a, due.Add(7*24*time.Hour))
	days, _, total, _ := e.late(res)
	require.Zero(t, days, "khoản trừ sửa tay không bị đụng (late_days giữ giá trị ban đầu 0 của dòng chèn)")
	require.True(t, total.Equal(decimal.NewFromInt(10)))
}

func TestDeleteClean(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	require.NoError(t, e.svc.Delete(e.ctx, e.act, e.course, a.ID))
	_, err := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	require.Equal(t, "NOT_FOUND", code(err))
	var n int
	e.one(&n, `select count(*) from audit_log where entity_id = $1 and action = 'delete'`, a.ID.String())
	require.Equal(t, 1, n)
}

func TestDeleteHasSubmissions(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.result(a, e.students[0], "DRAFT", due.Add(-time.Hour), crit10, false)
	require.Equal(t, "HAS_SUBMISSIONS", code(e.svc.Delete(e.ctx, e.act, e.course, a.ID)))
}

// gradeLink chèn công thức + một item trỏ tới bài.
func (e *env) gradeLink(a assignment.Assignment, status string) {
	var scheme, comp uuid.UUID
	e.one(&scheme, `insert into grade_schemes (course_id, scheme_no, status, created_by, confirmed_by, confirmed_at) values ($1, 1, $2::grade_scheme_status, $3, case when $2 = 'CONFIRMED' then $3::uuid end, case when $2 = 'CONFIRMED' then now() end) returning id`, e.course, status, e.teacher)
	e.one(&comp, `insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ($1, $2, 'bt', 'Bài tập', 'QT', 100) returning id`, e.course, scheme)
	e.exec(`insert into grade_items (course_id, scheme_id, component_id, key, name, source, assignment_id) values ($1, $2, $3, 'bt', 'BT', 'ASSIGNMENT', $4)`, e.course, scheme, comp, a.ID)
}

func TestDeleteLinkedToGradebook(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.gradeLink(a, "DRAFT")
	require.Equal(t, "LINKED_TO_GRADEBOOK", code(e.svc.Delete(e.ctx, e.act, e.course, a.ID)))
}

func TestGradeItemLinkDisplay(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	got, err := e.svc.Get(e.ctx, e.act, e.course, a.ID)
	require.NoError(t, err)
	require.Nil(t, got.GradeLink, `"Chưa tính vào điểm"`)
	e.gradeLink(a, "CONFIRMED")
	got, err = e.svc.Get(e.ctx, e.act, e.course, a.ID)
	require.NoError(t, err)
	require.NotNil(t, got.GradeLink)
	require.Equal(t, "Bài tập", got.GradeLink.ComponentName)
	require.Equal(t, "bt", got.GradeLink.ComponentKey)
}

func TestNoGradeItemColumn(t *testing.T) {
	e := newEnv(t)
	var n int
	e.one(&n, `select count(*) from information_schema.columns where table_name = 'assignments' and column_name = 'grade_item_id'`)
	require.Zero(t, n)
}

func TestReferenceAnswerNeverToStudent(t *testing.T) {
	e := newEnv(t)
	i := in("BT03")
	canary := "CANARY-BT03-ANSWER"
	i.ReferenceAnswer = &canary
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(t, err)
	e.publish(a)
	got, err := e.svc.Get(e.ctx, assignment.Actor{UserID: e.students[0], Student: true}, e.course, a.ID)
	require.NoError(t, err)
	raw, _ := json.Marshal(assignment.StudentOf(got))
	require.NotContains(t, string(raw), "CANARY")
	require.NotContains(t, string(raw), "guidance")
	require.NotContains(t, string(raw), "reference_answer")
	require.NotContains(t, string(raw), "submission_code")
	page, err := e.svc.List(e.ctx, assignment.Actor{UserID: e.students[0], Student: true}, e.course, httpx.PageParams{Limit: 10})
	require.NoError(t, err)
	raw, _ = json.Marshal(page.StudentItems)
	require.NotContains(t, string(raw), "CANARY")
	require.Len(t, page.StudentItems, 1)
}

func TestStudentListPublishedOnly(t *testing.T) {
	e := newEnv(t)
	draft := e.create("BT03")
	pub := e.create("BT04")
	e.publish(pub)
	stu := assignment.Actor{UserID: e.students[0], Student: true}
	_, err := e.svc.Get(e.ctx, stu, e.course, draft.ID)
	require.Equal(t, "NOT_FOUND", code(err))
	page, err := e.svc.List(e.ctx, stu, e.course, httpx.PageParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.StudentItems, 1)
	require.Equal(t, pub.ID, page.StudentItems[0].ID)
}

func TestListCursorPagination(t *testing.T) {
	e := newEnv(t)
	var want []uuid.UUID
	for _, c := range []string{"AA1", "AA2", "AA3"} {
		want = append([]uuid.UUID{e.create(c).ID}, want...) // mới nhất trước
	}
	var got []uuid.UUID
	p := httpx.PageParams{Limit: 2}
	for {
		pg, err := e.svc.List(e.ctx, e.act, e.course, p)
		require.NoError(t, err)
		for _, it := range pg.Items {
			got = append(got, it.ID)
			require.NotNil(t, it.Counts)
			require.Equal(t, 2, it.Counts.Total)
		}
		if pg.NextCursor == nil {
			break
		}
		c, perr := httpx.ParseCursor(*pg.NextCursor)
		require.Nil(t, perr)
		p.Cursor = &c
	}
	require.Equal(t, want, got)
}

func TestArchivedCourseBlocksWrites(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	e.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, e.course)
	_, err := e.svc.Create(e.ctx, e.act, e.course, in("BT09"))
	require.Equal(t, "COURSE_ARCHIVED", code(err))
	_, err = e.svc.Publish(e.ctx, e.act, e.course, a.ID)
	require.Equal(t, "COURSE_ARCHIVED", code(err))
}

var _ = grading.RubricItem{}

func httpxPage(limit int) httpx.PageParams { return httpx.PageParams{Limit: limit} }

// TestTeamsIdsFollowChannel — QC Q-01-b: kênh TEAMS ⇔ cả hai id Teams; bỏ kênh TEAMS thì hai id về NULL (chuỗi rỗng coi như không có), không đụng `assignments_teams_chk` (23514).
func TestTeamsIdsFollowChannel(t *testing.T) {
	e := newEnv(t)
	i := in("TM1")
	cls, asg := "class-1", "asg-1"
	i.Channels, i.TeamsClassID, i.TeamsAssignmentID = []string{"IN_APP", "TEAMS"}, &cls, &asg
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(t, err)
	require.NotNil(t, a.TeamsClassID)
	empty := ""
	i.Version, i.Channels, i.TeamsClassID, i.TeamsAssignmentID = a.Version, []string{"IN_APP"}, &empty, &empty
	out, err := e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.NoError(t, err)
	require.Nil(t, out.TeamsClassID)
	require.Nil(t, out.TeamsAssignmentID)
	i.Version, i.Channels, i.TeamsClassID = out.Version, []string{"IN_APP", "TEAMS"}, nil
	_, err = e.svc.Update(e.ctx, e.act, e.course, a.ID, i)
	require.Equal(t, "VALIDATION_FAILED", code(err), "TEAMS thiếu id")
}
