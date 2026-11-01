package grade_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

func (d *tdb) policy(course uuid.UUID) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into documents (course_id, title, type, status) values ($1, 'Quy chế', 'COURSE_POLICY', 'READY') returning id`, course)
	return id
}

// readyDraft: bản nháp qua được bộ kiểm (tổng trọng số 100, đủ bốn luật).
func (d *tdb) readyDraft(course, teacher uuid.UUID, no int) uuid.UUID {
	d.t.Helper()
	sc, _ := d.scheme(course, teacher, no)
	d.exec(`update grade_components set weight = 40 where scheme_id = $1`, sc)
	d.exec(`insert into grade_components (course_id, scheme_id, key, name, grp, weight, position) values ($1, $2, 'final', 'Cuối kỳ', 'CK', 60, 1)`, course, sc)
	d.exec(`insert into grade_rules (course_id, scheme_id, kind, applicable) values ($1, $2, 'BONUS', false), ($1, $2, 'PENALTY', false)`, course, sc)
	d.exec(`insert into grade_rules (course_id, scheme_id, kind, params) values ($1, $2, 'ROUNDING', '{"step":"0.5","mode":"HALF_UP"}'),
		($1, $2, 'LETTER_SCALE', '{"bands":[{"min":"8.5","letter":"A"},{"min":"7","letter":"B"},{"min":"5.5","letter":"C"},{"min":"4","letter":"D"},{"min":"0","letter":"F"}]}')`, course, sc)
	return sc
}

// recompute ghi `unclear_count` như đường thật (worker `document.changed`); đọc trạng thái KHÔNG chạy bộ kiểm nữa (#16).
func (d *tdb) recompute(course uuid.UUID) {
	d.t.Helper()
	require.NoError(d.t, (&grade.Service{Pool: d.pool}).RecomputeUnclear(d.ctx, course))
}

func (d *tdb) confirm(sc, teacher uuid.UUID) {
	d.t.Helper()
	d.exec(`update grade_schemes set status = 'CONFIRMED', confirmed_by = $2, confirmed_at = now() where id = $1`, sc, teacher)
}

func (d *tdb) status(course uuid.UUID) grade.SchemeStatus {
	d.t.Helper()
	st, err := (&grade.Service{Pool: d.pool}).Status(d.ctx, course)
	require.NoError(d.t, err)
	return st
}

// AC1: năm trạng thái theo đúng dữ kiện, tính một lần cho một lô.
func TestStatusStates(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	require.Equal(t, grade.StateNoPolicy, d.status(c).State)
	require.False(t, d.status(c).HasPolicy)

	d.policy(c)
	require.Equal(t, grade.StateNotExtracted, d.status(c).State)
	require.True(t, d.status(c).HasPolicy)

	sc, _ := d.scheme(c, gv, 1) // một thành phần 20% → chưa đủ 100
	d.recompute(c)
	st := d.status(c)
	require.Equal(t, grade.StateDraftUnclear, st.State)
	require.Positive(t, st.UnclearCount)
	d.exec(`delete from grade_components where scheme_id = $1`, sc)
	d.exec(`delete from grade_schemes where id = $1`, sc)

	ready := d.readyDraft(c, gv, 1)
	d.recompute(c)
	st = d.status(c)
	require.Equal(t, grade.StateDraftReady, st.State)
	require.Zero(t, st.UnclearCount)

	d.confirm(ready, gv)
	st = d.status(c)
	require.Equal(t, grade.StateConfirmed, st.State)
	require.NotNil(t, st.ConfirmedAt)
	require.False(t, st.PendingDraft)
	require.Empty(t, st.FinalizeBlockedReason)
}

// AC6: bản nháp mới cạnh bản đang dùng không đổi trạng thái CONFIRMED; chỉ bật cờ `pending_draft`.
func TestStatusPendingDraft(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.policy(c)
	d.confirm(d.readyDraft(c, gv, 1), gv)
	d.readyDraft(c, gv, 2)
	st := d.status(c)
	require.Equal(t, grade.StateConfirmed, st.State)
	require.True(t, st.PendingDraft)
}

// Chưa CONFIRMED → `Chốt điểm` bị khoá kèm lý do.
func TestStatusBlocksFinalizeWithoutScheme(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	require.Equal(t, grade.BlockedNoScheme, d.status(c).FinalizeBlockedReason)
}

// AC8: gỡ tài liệu quy chế không làm mất công thức đã xác nhận.
func TestSourceDocumentDeletedKeepsConfirmed(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.policy(c)
	sc := d.readyDraft(c, gv, 1)
	d.exec(`update grade_schemes set source_document_id = $2, source_title = 'Quy chế' where id = $1`, sc, doc)
	d.confirm(sc, gv)
	d.exec(`delete from documents where id = $1`, doc)
	st := d.status(c)
	require.Equal(t, grade.StateConfirmed, st.State)
	var src *uuid.UUID
	d.one(&src, `select source_document_id from grade_schemes where id = $1`, sc)
	require.Nil(t, src)
	var title string
	d.one(&title, `select source_title from grade_schemes where id = $1`, sc)
	require.Equal(t, "Quy chế", title, "tên tài liệu còn để hiển thị")
}

type nopPub struct{}

func (nopPub) Publish(context.Context, string, string, any) (string, error) { return "1", nil }

type remindRig struct {
	*tdb
	job *grade.ReminderJob
	clk *clock.Fake
	gv  uuid.UUID
	c   uuid.UUID
}

// Thứ Hai 2026-11-02 00:00 ICT = 2026-11-01T17:00Z.
var mon = time.Date(2026, 11, 1, 17, 0, 0, 0, time.UTC)

func newRemindRig(t *testing.T) *remindRig {
	t.Helper()
	d := newTDB(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	require.NoError(t, rdb.Del(t.Context(), grade.ReminderLeaderKey).Err())
	for _, wk := range []string{"202645", "202646", "202647"} {
		require.NoError(t, rdb.Del(t.Context(), appredis.Key("grade.remind", wk)).Err())
	}
	clk := clock.NewFake(mon.Add(8*time.Hour + time.Minute)) // thứ Hai 08:01
	svc := &grade.Service{Pool: d.pool, Clock: clk, Notify: notify.New(d.pool, nopPub{}, nil)}
	gv := d.user("TEACHER", "GV")
	return &remindRig{tdb: d, clk: clk, gv: gv, c: d.course(gv, "L1"),
		job: &grade.ReminderJob{Svc: svc, Redis: rdb, Name: "t", Every: time.Minute}}
}

func (r *remindRig) count(user uuid.UUID) (n int) {
	r.one(&n, `select count(*) from notifications where user_id = $1 and type = 'GRADE_SCHEME_REMINDER'`, user)
	return n
}

func (r *remindRig) once() bool {
	r.t.Helper()
	ok, err := r.job.Once(r.ctx)
	require.NoError(r.t, err)
	return ok
}

// AC3: mỗi tuần đúng một lần; trước 08:00 thứ Hai chưa nhắc.
func TestReminderWeeklyOncePerWeek(t *testing.T) {
	r := newRemindRig(t)
	r.clk.Set(mon.Add(7*time.Hour + 59*time.Minute))
	require.False(t, r.once())
	require.Zero(t, r.count(r.gv))

	r.clk.Set(mon.Add(8 * time.Hour))
	require.True(t, r.once())
	require.False(t, r.once(), "tick sau cùng tuần không nhắc lại")
	r.clk.Advance(48 * time.Hour)
	require.False(t, r.once())
	require.Equal(t, 1, r.count(r.gv))

	r.clk.Set(mon.Add(7*24*time.Hour + 8*time.Hour))
	require.True(t, r.once())
	require.Equal(t, 2, r.count(r.gv))
}

// Worker lên muộn giữa tuần vẫn nhắc bù một lần.
func TestReminderCatchUp(t *testing.T) {
	r := newRemindRig(t)
	r.clk.Set(mon.Add(2*24*time.Hour + 14*time.Hour)) // thứ Tư 14:00
	require.True(t, r.once())
	require.Equal(t, 1, r.count(r.gv))
}

// Hết nhắc khi lớp đã có công thức CONFIRMED.
func TestReminderStopsWhenConfirmed(t *testing.T) {
	r := newRemindRig(t)
	r.policy(r.c)
	r.confirm(r.readyDraft(r.c, r.gv, 1), r.gv)
	require.True(t, r.once())
	require.Zero(t, r.count(r.gv))
}

func TestReminderSkipsArchived(t *testing.T) {
	r := newRemindRig(t)
	r.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, r.c)
	require.True(t, r.once())
	require.Zero(t, r.count(r.gv))
}

// Chỉ TEACHER của lớp nhận; TA, sinh viên, admin không.
func TestReminderRecipientsTeacherOnly(t *testing.T) {
	r := newRemindRig(t)
	ta := r.user("TA", "TA")
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, r.c, ta)
	sv := r.student(r.c, "SV", "20220001")
	admin := r.user("ADMIN", "AD")
	require.True(t, r.once())
	require.Equal(t, 1, r.count(r.gv))
	for _, u := range []uuid.UUID{ta, sv, admin} {
		require.Zero(t, r.count(u))
	}
}

// #16: số mục chưa rõ nằm ở cột, ghi lúc lưu / khi tài liệu đổi; đọc không tính lại. Gỡ tài liệu nguồn → trích dẫn thôi bị kiểm → tính lại về 0.
func TestRecomputeOnDocumentChanged(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.policy(c)
	sc := d.readyDraft(c, gv, 1)
	d.exec(`update grade_schemes set source_document_id = $2 where id = $1`, sc, doc)
	d.exec(`update grade_components set citations = '[{"page":1,"quote":"điều không có trong tài liệu"}]' where scheme_id = $1 and key = 'final'`, sc)
	d.recompute(c)
	require.Equal(t, 1, d.status(c).UnclearCount, "CITATION_NOT_FOUND")
	d.exec(`update grade_schemes set unclear_count = 7 where id = $1`, sc)
	require.Equal(t, 7, d.status(c).UnclearCount, "đọc lấy đúng cột, không chạy bộ kiểm")

	d.exec(`delete from documents where id = $1`, doc)
	payload := []byte(`{"course_id":"` + c.String() + `","document_id":"` + doc.String() + `"}`)
	require.NoError(t, (&grade.Service{Pool: d.pool}).HandleDocumentChanged(d.ctx, outbox.Message{Topic: "document.changed", Payload: payload}))
	require.Zero(t, d.status(c).UnclearCount)
	require.Equal(t, grade.StateDraftReady, d.status(c).State)
}
