package escalation_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/store"
)

func TestEscalateIdempotent(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi gì đó?", "Trả lời của AI")
	first, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	require.False(t, first.Already)
	require.Equal(t, store.TicketStatusOPEN, first.Ticket.Status)
	require.Equal(t, store.TicketOriginSTUDENTREQUEST, first.Ticket.Origin)
	again, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	require.True(t, again.Already)
	require.Equal(t, first.Ticket.ID, again.Ticket.ID)
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where message_id = $1`, m))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.created'`), "một ticket ⇒ một dòng outbox")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'escalation_ticket' and action = 'escalation_ticket.created'`))

	// ticket đã CLOSED không bị mở lại
	r.exec(`update escalation_tickets set status = 'CLOSED', closed_at = now(), closed_reason = 'STUDENT_CONFIRMED' where id = $1`, first.Ticket.ID)
	closed, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	require.True(t, closed.Already)
	require.Equal(t, store.TicketStatusCLOSED, closed.Ticket.Status)
}

func TestEscalateConcurrent10(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi?", "Trả lời")
	var wg sync.WaitGroup
	res := make([]escalation.Created, 10)
	errs := make([]error, 10)
	for i := range res {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res[i], errs[i] = r.svc.Escalate(t.Context(), r.a, m)
		}()
	}
	wg.Wait()
	created := 0
	for i := range res {
		require.NoError(t, errs[i])
		require.Equal(t, res[0].Ticket.ID, res[i].Ticket.ID)
		if !res[i].Already {
			created++
		}
	}
	require.Equal(t, 1, created, "đúng một lời gọi tạo mới")
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where message_id = $1`, m))
}

// fill tạo n ticket mở của A ở lớp `course` (mỗi ticket một tin trợ lý riêng).
func (r *rig) fill(course uuid.UUID, n int) {
	r.t.Helper()
	sess := r.sessA
	if course != r.course {
		sess = r.session(course, r.a.UserID)
	}
	for range n {
		u := r.msg(course, sess, r.a.UserID, "USER", "q", "", nil)
		m := r.msg(course, sess, r.a.UserID, "ASSISTANT", "a", "COURSE_QA", &u)
		r.exec(`insert into escalation_tickets (course_id, origin, source_type, source_id, message_id, session_id, student_id) values ($1, 'STUDENT_REQUEST', 'CHAT_MESSAGE', $2, $2, $3, $4)`, course, m, sess, r.a.UserID)
	}
}

func TestTicketCap(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.fill(r.course, 5)
	_, m := r.pair("Câu thứ sáu?", "Trả lời")
	_, err := r.svc.Escalate(t.Context(), r.a, m)
	require.Equal(t, 409, status(err))
	require.Equal(t, "TICKET_LIMIT_REACHED", code(err))
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets where message_id = $1`, m))
	// ANSWERED / CLOSED không tính vào trần
	r.exec(`update escalation_tickets set status = 'CLOSED', closed_at = now(), closed_reason = 'AUTO_7D' where id = (select id from escalation_tickets where student_id = $1 limit 1)`, r.a.UserID)
	_, err = r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
}

func TestEscalateAlreadyBeatsCap(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Câu đã nhờ", "Trả lời")
	first, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	r.fill(r.course, 4) // tổng 5 ticket mở
	again, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err, "tin đã nhờ: 200 already, không 409")
	require.True(t, again.Already)
	require.Equal(t, first.Ticket.ID, again.Ticket.ID)
}

func TestAutoTicketSuppressedByCap(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.fill(r.course, 5)
	_, m := r.pair("Câu thấp tin cậy", "AI chưa chắc")
	tx, err := r.pool.Begin(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(t.Context()) }()
	c, err := r.svc.CreateTx(t.Context(), tx, escalation.CreateInput{CourseID: r.course, StudentID: r.a.UserID, MessageID: m, SessionID: r.sessA, Origin: store.TicketOriginAUTOLOWCONFIDENCE, Auto: true})
	require.NoError(t, err, "tự mở bị chặn thì bỏ qua, không lỗi")
	require.True(t, c.Suppressed)
	require.EqualValues(t, 1, r.svc.Suppressed.Load())
	require.NoError(t, tx.Commit(t.Context()))
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets where message_id = $1`, m))
}

func TestCapPerCourse(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.fill(r.course, 5)
	sess2 := r.session(r.course2, r.a.UserID)
	u := r.msg(r.course2, sess2, r.a.UserID, "USER", "q", "", nil)
	m2 := r.msg(r.course2, sess2, r.a.UserID, "ASSISTANT", "a", "COURSE_QA", &u)
	_, err := r.svc.Escalate(t.Context(), r.a, m2)
	require.NoError(t, err, "trần tính theo lớp")
	// và theo sinh viên: B ở lớp 1 vẫn nhờ được
	sB := r.session(r.course, r.b.UserID)
	ub := r.msg(r.course, sB, r.b.UserID, "USER", "q", "", nil)
	mb := r.msg(r.course, sB, r.b.UserID, "ASSISTANT", "a", "COURSE_QA", &ub)
	_, err = r.svc.Escalate(t.Context(), r.b, mb)
	require.NoError(t, err)
}

func TestEscalateAuthMatrix(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi?", "Trả lời")
	for name, who := range map[string]escalation.Actor{
		"TA":      {UserID: r.ta, Role: auth.RoleTA},
		"TEACHER": {UserID: r.teacher, Role: auth.RoleTeacher},
		"ADMIN":   {UserID: r.admin, Role: auth.RoleAdmin},
	} {
		_, err := r.svc.Escalate(t.Context(), who, m)
		require.Equal(t, 403, status(err), name)
	}
	_, err := r.svc.Escalate(t.Context(), r.b, m)
	require.Equal(t, 404, status(err), "tin của sinh viên khác")
	_, err = r.svc.Escalate(t.Context(), r.a, uuid.New())
	require.Equal(t, 404, status(err), "tin không tồn tại")
	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, r.course, r.a.UserID)
	_, err = r.svc.Escalate(t.Context(), r.a, m)
	require.Equal(t, 403, status(err), "sinh viên đã rời lớp")
	r.exec(`update enrollments set status = 'ACTIVE', removed_at = null where course_id = $1 and user_id = $2`, r.course, r.a.UserID)
	r.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, r.course)
	_, err = r.svc.Escalate(t.Context(), r.a, m)
	require.Equal(t, 409, status(err))
	require.Equal(t, "COURSE_ARCHIVED", code(err))
}

func TestEscalateExamLocked(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi?", "Trả lời")
	r.lock.lock = &exam.Lock{AttemptID: uuid.New()}
	_, err := r.svc.Escalate(t.Context(), r.a, m)
	require.Equal(t, 409, status(err))
	require.Equal(t, "EXAM_IN_PROGRESS", code(err), "cùng mã với chat riêng khi đang thi")
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets`))
}

func TestEscalateNotEligible(t *testing.T) {
	r := newRig(t, escalation.Config{})
	u, a := r.pair("Hỏi?", "Trả lời")
	_, err := r.svc.Escalate(t.Context(), r.a, u)
	require.Equal(t, 422, status(err), "tin của người dùng")
	require.Equal(t, "MESSAGE_NOT_ESCALATABLE", code(err))
	for _, intent := range []string{"CRISIS", "OTHER_PERSON", "SMALLTALK"} {
		m := r.msg(r.course, r.sessA, r.a.UserID, "ASSISTANT", "câu mẫu", intent, &u)
		_, err = r.svc.Escalate(t.Context(), r.a, m)
		require.Equal(t, "MESSAGE_NOT_ESCALATABLE", code(err), intent)
	}
	r.exec(`update chat_messages set stream_status = 'STREAMING', partial_content = 'dở' where id = $1`, a)
	_, err = r.svc.Escalate(t.Context(), r.a, a)
	require.Equal(t, "MESSAGE_NOT_ESCALATABLE", code(err), "đang STREAMING")
	r.exec(`update chat_messages set stream_status = 'DONE', partial_content = null where id = $1`, a)
	r.exec(`update chat_sessions set deleted_at = now() where id = $1`, r.sessA)
	_, err = r.svc.Escalate(t.Context(), r.a, a)
	require.Equal(t, "MESSAGE_NOT_ESCALATABLE", code(err), "phiên đã xoá mềm")
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets`))
}

func TestEscalateDBDown(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi?", "Trả lời")
	r.pool.Close()
	_, err := r.svc.Escalate(t.Context(), r.a, m)
	require.Equal(t, 503, status(err))
}

// TestStudentNeverSeesTicketInternals — AC11e: phần ticket sinh viên thấy chỉ có bốn khoá.
func TestStudentNeverSeesTicketInternals(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi?", "Trả lời")
	c, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	b, err := json.Marshal(escalation.ViewOf(c.Ticket))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.ElementsMatch(t, []string{"id", "status", "created_at"}, keys(got), "OPEN: chưa có answered_at")
	r.exec(`update escalation_tickets set status = 'ANSWERED', answered_by = $2, answered_at = now(), claimed_by = $2, confidence = 0.5 where id = $1`, c.Ticket.ID, r.teacher)
	v, err := escalation.ViewsForMessages(t.Context(), r.pool, r.a.UserID, []uuid.UUID{m})
	require.NoError(t, err)
	b, _ = json.Marshal(v[m])
	require.NoError(t, json.Unmarshal(b, &got))
	require.ElementsMatch(t, []string{"id", "status", "created_at", "answered_at"}, keys(got))
	other, err := escalation.ViewsForMessages(t.Context(), r.pool, r.b.UserID, []uuid.UUID{m})
	require.NoError(t, err)
	require.Empty(t, other, "ticket của A không hiện cho B")
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
