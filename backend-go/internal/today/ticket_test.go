package today_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) ticketRow(c cls, student uuid.UUID, status string, claimedBy *uuid.UUID, waited time.Duration) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), `insert into escalation_tickets (course_id, kind, origin, source_type, source_id, student_id, status, claimed_by, claimed_at, waiting_since)
		values ($1, 'TEACHER_OUTREACH', 'STAFF_OUTREACH', 'STUDENT', $2, $2, $3::ticket_status, $4, case when $4::uuid is null then null else now() end, $5::timestamptz)`,
		c.id, student, status, claimedBy, r.clk.Now().Add(-waited))
	require.NoError(r.t, err)
}

// ticketRig: lớp có giảng viên, TA, sinh viên; trả (lớp, giảng viên, phiên GV, phiên TA, người SV, phiên SV).
func ticketRig(r *rig, code, mssv string) (cls, store.User, session, session, store.User, session) {
	c := r.klass(code)
	gvUser, gv := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	svUser, sv := r.student(mssv)
	r.enrollAt(c, svUser, "STUDENT", "ACTIVE", "", 24*time.Hour)
	return c, gvUser, gv, taSess, svUser, sv
}

func TestTicketProvider(t *testing.T) {
	r := newRig(t)
	c, gvUser, gv, taSess, svUser, _ := ticketRig(r, "761990", "B20DC000010")
	require.NotContains(t, kindsOf(r.tget(gv, "/me/today")), "TICKET", "không có ticket thì không có việc")
	r.ticketRow(c, svUser.ID, "OPEN", nil, 5*time.Hour)
	r.ticketRow(c, svUser.ID, "OPEN", nil, time.Hour)
	r.ticketRow(c, svUser.ID, "CLAIMED", &gvUser.ID, 2*time.Hour)

	k := kindsOf(r.tget(gv, "/me/today"))
	require.Contains(t, k, "TICKET")
	require.Equal(t, "2 câu hỏi đang chờ · 1 câu bạn đã nhận · lâu nhất 5 giờ", k["TICKET"]["reason"])
	require.Equal(t, "/inbox?course="+c.id.String(), k["TICKET"]["href"])
	require.Equal(t, "2 câu hỏi đang chờ · 0 câu bạn đã nhận · lâu nhất 5 giờ", kindsOf(r.tget(taSess, "/me/today"))["TICKET"]["reason"], "số 'đã nhận' là của riêng từng người")
}

func TestTicketProviderStaffOnly(t *testing.T) {
	r := newRig(t)
	c, _, _, _, svUser, sv := ticketRig(r, "761992", "B20DC000012")
	r.ticketRow(c, svUser.ID, "OPEN", nil, time.Hour)
	require.NotContains(t, kindsOf(r.tget(sv, "/me/today")), "TICKET", "sinh viên không thấy việc của nhân viên")
}

func TestTicketProviderClears(t *testing.T) {
	r := newRig(t)
	c, gvUser, gv, _, svUser, _ := ticketRig(r, "761993", "B20DC000013")
	r.ticketRow(c, svUser.ID, "OPEN", nil, time.Hour)
	require.Contains(t, kindsOf(r.tget(gv, "/me/today")), "TICKET")
	r.exec(`update escalation_tickets set status = 'ANSWERED', answered_by = $1, answered_at = now() where course_id = $2`, gvUser.ID, c.id)
	require.NotContains(t, kindsOf(r.tget(gv, "/me/today")), "TICKET", "hết OPEN và hết CLAIMED thì biến mất")
}

func TestStaffAnswerUnreadItem(t *testing.T) {
	r := newRig(t)
	c := r.klass("761991")
	r.teaching(c)
	svUser, sv := r.student("B20DC000011")
	r.enrollAt(c, svUser, "STUDENT", "ACTIVE", "", 24*time.Hour)

	link := "/chat?session=" + uuid.NewString()
	r.exec(`insert into notifications (user_id, course_id, type, title, link, created_at) values ($1, $2, 'TICKET_ANSWERED', 'Giảng viên đã trả lời câu hỏi của bạn', $3, now() - interval '2 hours')`, svUser.ID, c.id, link)
	k := kindsOf(r.tget(sv, "/me/today"))
	require.Contains(t, k, "STAFF_ANSWER_UNREAD")
	require.Equal(t, link, k["STAFF_ANSWER_UNREAD"]["href"])

	r.exec(`update notifications set read_at = now() where user_id = $1`, svUser.ID)
	require.NotContains(t, kindsOf(r.tget(sv, "/me/today")), "STAFF_ANSWER_UNREAD", "đã đọc thì biến mất")
}

func (r *rig) exec(q string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), q, args...)
	require.NoError(r.t, err, q)
}
