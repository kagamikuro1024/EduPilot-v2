package escalation_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/privacy"
)

func (r *rig) gv() escalation.Actor {
	return escalation.Actor{UserID: r.teacher, Role: auth.RoleTeacher}
}
func (r *rig) taA() escalation.Actor { return escalation.Actor{UserID: r.ta, Role: auth.RoleTA} }

// ticket tạo ticket OPEN của A từ một cặp hỏi–đáp mới (trong phiên sessA).
func (r *rig) ticket(q string) uuid.UUID {
	r.t.Helper()
	_, m := r.pair(q, "Trả lời của AI")
	c, err := r.svc.Escalate(r.t.Context(), r.a, m)
	require.NoError(r.t, err)
	return c.Ticket.ID
}

func TestListTicketsFilterAndCursor(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 500})
	var ids []uuid.UUID
	for i := range 125 {
		ids = append(ids, r.ticket(fmt.Sprintf("Câu %d", i+1)))
	}
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, ids[0], nil)
	require.NoError(t, err)

	var seen []uuid.UUID
	pg := httpx.PageParams{Limit: 50}
	for {
		page, err := r.svc.ListStaff(t.Context(), []uuid.UUID{r.course}, "OPEN", pg)
		require.NoError(t, err)
		for _, it := range page.Items {
			seen = append(seen, it.ID)
			require.LessOrEqual(t, len([]rune(it.QuestionPreview)), 140)
		}
		if page.NextCursor == nil {
			break
		}
		c, cerr := httpx.ParseCursor(*page.NextCursor)
		require.NoError(t, cerr)
		pg.Cursor = &c
	}
	require.Equal(t, ids[1:], seen, "OPEN: cũ trước, đi hết các trang, không lặp, không sót, bỏ ticket đã nhận")

	all, err := r.svc.ListStaff(t.Context(), []uuid.UUID{r.course}, "ALL", httpx.PageParams{Limit: 50})
	require.NoError(t, err)
	require.Len(t, all.Items, 50, "một trang ≤ limit")
	claimed, err := r.svc.ListStaff(t.Context(), []uuid.UUID{r.course}, "CLAIMED", httpx.PageParams{Limit: 50})
	require.NoError(t, err)
	require.Len(t, claimed.Items, 1)
	require.Equal(t, "GV", claimed.Items[0].ClaimedBy.FullName)

	long := strings.Repeat("ạ", 300)
	_, m := r.pair(long, "AI")
	_, err = r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	last, err := r.svc.ListStaff(t.Context(), []uuid.UUID{r.course}, "OPEN", httpx.PageParams{Limit: 200})
	require.NoError(t, err)
	require.Len(t, []rune(last.Items[len(last.Items)-1].QuestionPreview), 140)

	_, err = r.svc.ListStaff(t.Context(), []uuid.UUID{r.course}, "NOPE", httpx.PageParams{Limit: 5})
	require.Equal(t, 422, status(err))
}

func TestMyTicketsAcrossCourses(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.ticket("Lớp 1")
	s2 := r.session(r.course2, r.a.UserID)
	u := r.msg(r.course2, s2, r.a.UserID, "USER", "Lớp 2", "", nil)
	m := r.msg(r.course2, s2, r.a.UserID, "ASSISTANT", "AI", "COURSE_QA", &u)
	_, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)

	mine, err := r.svc.MyCourseIDs(t.Context(), r.teacher)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{r.course, r.course2}, mine)
	page, err := r.svc.ListStaff(t.Context(), mine, "OPEN", httpx.PageParams{Limit: 50})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)

	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, r.course2, r.teacher)
	mine, err = r.svc.MyCourseIDs(t.Context(), r.teacher)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{r.course}, mine, "lớp đã rời không còn trong hàng chờ")
	studentCourses, err := r.svc.MyCourseIDs(t.Context(), r.a.UserID)
	require.NoError(t, err)
	require.Empty(t, studentCourses)
}

func TestTicketDetailOnlyExchange(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.msg(r.course, r.sessA, r.a.UserID, "USER", "Tin KHÁC không liên quan", "", nil)
	id := r.ticket("Câu hỏi được nhờ")
	d, err := r.svc.GetDetail(t.Context(), r.course, id)
	require.NoError(t, err)
	require.Equal(t, "Câu hỏi được nhờ", d.Question.Content)
	require.Equal(t, "Trả lời của AI", d.AIAnswer.Content)
	require.Empty(t, d.Transcript, "chưa có tin của ticket")
	_, err = r.svc.GetDetail(t.Context(), r.course2, id)
	require.Equal(t, 404, status(err), "ticket của lớp khác")

	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, id, "Em xem bài 3.", nil)
	require.NoError(t, err)
	d, err = r.svc.GetDetail(t.Context(), r.course, id)
	require.NoError(t, err)
	require.Len(t, d.Transcript, 1)
	require.Equal(t, "Em xem bài 3.", d.Transcript[0].Content)
}

func TestClaimRace2(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, a := range []escalation.Actor{r.gv(), r.taA()} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = r.svc.Claim(t.Context(), a, r.course, id, nil)
		}()
	}
	wg.Wait()
	ok, lost := 0, 0
	for _, e := range errs {
		switch {
		case e == nil:
			ok++
		case code(e) == apierr.TicketAlreadyClaimed:
			lost++
		}
	}
	require.Equal(t, [2]int{1, 1}, [2]int{ok, lost}, "đúng một người nhận được, người kia 409")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'escalation_ticket.claimed'`))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.claimed'`))
}

func TestClaimIdempotentSameUser(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	first, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	again, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	require.Equal(t, first.Version, again.Version, "bấm lại không đổi gì")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.claimed'`))
}

func TestClaimConflictShowsOwner(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	_, err = r.svc.Claim(t.Context(), r.taA(), r.course, id, nil)
	require.Equal(t, apierr.TicketAlreadyClaimed, code(err))
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Contains(t, ae.Details, "claimed_by")
	require.Contains(t, ae.Details, "claimed_at")
}

func TestReleaseOnlyByOwner(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	_, err := r.svc.Claim(t.Context(), r.taA(), r.course, id, nil)
	require.NoError(t, err)
	_, err = r.svc.Release(t.Context(), r.gv(), r.course, id, nil)
	require.Equal(t, 409, status(err), "giảng viên không bỏ nhận thay TA")
	got, err := r.svc.Release(t.Context(), r.taA(), r.course, id, nil)
	require.NoError(t, err)
	require.Equal(t, "OPEN", got.Status)
}

func TestReleaseKeepsWaitingSince(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	r.exec(`update escalation_tickets set waiting_since = now() - interval '3 hours' where id = $1`, id)
	var before, after time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select waiting_since from escalation_tickets where id = $1`, id).Scan(&before))
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	_, err = r.svc.Release(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	require.NoError(t, r.pool.QueryRow(t.Context(), `select waiting_since from escalation_tickets where id = $1`, id).Scan(&after))
	require.True(t, before.Equal(after), "waiting_since không đổi sau bỏ nhận")
}

func TestAnswerCreatesChatMessage(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	tk, msg, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "  Em xem lại bài 3.  ", nil)
	require.NoError(t, err)
	require.Equal(t, "ANSWERED", tk.Status)
	require.Equal(t, "Em xem lại bài 3.", msg.Content)
	var intent, role string
	var user uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select intent, role::text, user_id from chat_messages where id = $1`, msg.ID).Scan(&intent, &role, &user))
	require.Equal(t, [3]any{"STAFF_ANSWER", "ASSISTANT", r.a.UserID}, [3]any{intent, role, user}, "tin nằm trong chat riêng của sinh viên A")
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.answered'`))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'escalation_ticket.answered' and actor_id = $1`, r.teacher))

}

func TestSecondAnswer(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời đầu", nil)
	require.NoError(t, err)
	r.exec(`update escalation_tickets set answered_at = now() - interval '3 days' where id = $1`, id)
	_, _, err = r.svc.Answer(t.Context(), r.taA(), r.course, id, "Thêm", nil)
	require.Equal(t, 409, status(err), "người khác không được thêm")
	tk, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Bổ sung một ý.", nil)
	require.NoError(t, err)
	require.Equal(t, "ANSWERED", tk.Status, "ticket giữ ANSWERED")
	require.Equal(t, 2, r.count(`select count(*) from chat_messages where intent = 'STAFF_ANSWER' and user_id = $1`, r.a.UserID))
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where id = $1 and answered_at > now() - interval '1 minute'`, id), "đặt lại đồng hồ tự đóng")
}

func TestAnswerByOtherBlocked(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.NoError(t, err)
	_, _, err = r.svc.Answer(t.Context(), r.taA(), r.course, id, "Tôi trả lời thay", nil)
	require.Equal(t, apierr.TicketAlreadyClaimed, code(err))
	require.Zero(t, r.count(`select count(*) from chat_messages where intent = 'STAFF_ANSWER'`))
}

func TestTicketActionsAudited(t *testing.T) {
	r := newRig(t, escalation.Config{OpenCap: 10})
	a, b := r.ticket("Hỏi một"), r.ticket("Hỏi hai")
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, a, nil)
	require.NoError(t, err)
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, a, "BÍ-MẬT-NỘI-DUNG", nil)
	require.NoError(t, err)
	_, err = r.svc.Dismiss(t.Context(), r.taA(), r.course, b, "OUT_OF_SCOPE", nil)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'escalation_ticket' and action = 'escalation_ticket.claimed' and actor_id = $1 and entity_id = $2`, r.teacher, a.String()))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'escalation_ticket' and action = 'escalation_ticket.answered' and actor_id = $1 and entity_id = $2`, r.teacher, a.String()))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'escalation_ticket' and action = 'escalation_ticket.dismissed' and actor_id = $1 and entity_id = $2`, r.ta, b.String()))
	require.Zero(t, r.count(`select count(*) from audit_log where entity = 'escalation_ticket' and (coalesce(before::text, '') || coalesce(after::text, '')) like '%BÍ-MẬT%'`), "audit không chứa nội dung")
}

func TestAnswerValidation(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	for name, body := range map[string]string{"rỗng": "", "trắng": "  \n ", "quá dài": strings.Repeat("a", escalation.AnswerMaxChars+1)} {
		_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, body, nil)
		require.Equal(t, 422, status(err), name)
	}
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, strings.Repeat("a", escalation.AnswerMaxChars), nil)
	require.NoError(t, err, "đúng 4000 ký tự vẫn nhận")
	require.Equal(t, 0, r.count(`select count(*) from chat_messages where intent = 'STAFF_ANSWER' and content = ''`))
}

func TestAnswerToDeletedSessionGoesToStaffSession(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	r.exec(`update chat_sessions set deleted_at = now() where id = $1`, r.sessA)
	_, msg, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Trả lời", nil)
	require.NoError(t, err)
	var sess uuid.UUID
	var title string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select s.id, s.title from chat_messages m join chat_sessions s on s.id = m.session_id where m.id = $1`, msg.ID).Scan(&sess, &title))
	require.NotEqual(t, r.sessA, sess)
	require.Equal(t, escalation.StaffSessionTitle, title)
}

func TestDismissAndClosedAreFinal(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	_, err := r.svc.Dismiss(t.Context(), r.gv(), r.course, id, "KHONG_CO", nil)
	require.Equal(t, 422, status(err))
	tk, err := r.svc.Dismiss(t.Context(), r.gv(), r.course, id, "DUPLICATE", nil)
	require.NoError(t, err)
	require.Equal(t, "CLOSED", tk.Status)
	_, err = r.svc.Claim(t.Context(), r.gv(), r.course, id, nil)
	require.Equal(t, apierr.TicketStateInvalid, code(err))
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, id, "x", nil)
	require.Equal(t, apierr.TicketStateInvalid, code(err))
}

// TC-16 (QC vòng 3): người thua gửi kèm version cũ (phiếu vừa được nhận nên version đã đổi) vẫn nhận TICKET_ALREADY_CLAIMED kèm claimed_by + claimed_at, không phải VERSION_CONFLICT.
func TestClaimLoserWithStaleVersionSeesOwnerAndTime(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	d, err := r.svc.GetDetail(t.Context(), r.course, id)
	require.NoError(t, err)
	seen := d.Ticket.Version
	_, err = r.svc.Claim(t.Context(), r.gv(), r.course, id, &seen)
	require.NoError(t, err)
	_, err = r.svc.Claim(t.Context(), r.taA(), r.course, id, &seen) // bản cũ
	require.Equal(t, apierr.TicketAlreadyClaimed, code(err))
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Contains(t, ae.Details, "claimed_by")
	require.NotNil(t, ae.Details.(map[string]any)["claimed_at"])
}

func TestVersionConflict(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	stale := int32(99)
	_, err := r.svc.Claim(t.Context(), r.gv(), r.course, id, &stale)
	require.Equal(t, apierr.VersionConflict, code(err))
}

func piiRig(t *testing.T) *rig {
	r := newRig(t, escalation.Config{})
	r.svc.PII = &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: r.pool}}}
	return r
}

func TestAnswerBlocksOtherStudentPII(t *testing.T) {
	r := piiRig(t)
	id := r.ticket("Hỏi")
	// tên đầy đủ của sinh viên B (cùng lớp) trong câu trả lời cho A
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Bạn SV B cũng hỏi câu này.", nil)
	require.Equal(t, apierr.PIIInAnswer, code(err))
	require.Equal(t, 0, r.count(`select count(*) from chat_messages where intent = 'STAFF_ANSWER'`), "chặn cứng: không có tin nào")
	require.Equal(t, 0, r.count(`select count(*) from outbox where topic = 'escalation.answered'`))
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.NotContains(t, strings.ToLower(strings.Join(strings.Fields(strings.ToLower(ae.Message)), " ")), "sv b", "lỗi không lặp lại giá trị")

	// CCCD luôn chặn
	_, _, err = r.svc.Answer(t.Context(), r.gv(), r.course, id, "Số 012345678901 là của em.", nil)
	require.Equal(t, apierr.PIIInAnswer, code(err))
}

func TestAnswerAllowsOwnName(t *testing.T) {
	r := piiRig(t)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Chào SV A, em xem bài 3 nhé.", nil)
	require.NoError(t, err, "tên của chính người hỏi không bị chặn")
}

func TestAnswerBlocksCCCD(t *testing.T) {
	r := piiRig(t)
	id := r.ticket("Hỏi")
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Số 012345678901 là của em.", nil)
	require.Equal(t, apierr.PIIInAnswer, code(err))
}

func TestAnswerBlocksOtherStudentEmail(t *testing.T) {
	r := piiRig(t)
	id := r.ticket("Hỏi")
	var email string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select email from users where id = $1`, r.b.UserID).Scan(&email))
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Hãy hỏi bạn qua "+email+" nhé.", nil)
	require.Equal(t, apierr.PIIInAnswer, code(err))
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.NotContains(t, ae.Message+" "+toJSON(ae.Details), email, "không lặp giá trị")
}

func TestAnswerAllowsStaffEmail(t *testing.T) {
	r := piiRig(t)
	id := r.ticket("Hỏi")
	var email string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select email from users where id = $1`, r.teacher).Scan(&email))
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Liên hệ "+email+" hoặc phongdaotao@ptit.edu.vn nhé.", nil)
	require.NoError(t, err, "email của giảng viên / phòng đào tạo không bị chặn")
}

func TestAnswerCommonWordNotBlocked(t *testing.T) {
	r := piiRig(t)
	id := r.ticket("Hỏi")
	r.user("Bình", "STUDENT") // chưa vào lớp: không thuộc roster
	_, _, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "Em bình tĩnh, mọi việc sẽ ổn.", nil)
	require.NoError(t, err, "từ thường trùng tên không bị chặn")
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
