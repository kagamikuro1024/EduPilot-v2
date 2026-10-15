package crm_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (w world) outreach(s session, i int, body string, key string) resp {
	return w.r.do(req{method: http.MethodPost, path: w.sp(i, "/outreach"), bearer: s.access, body: map[string]any{"body": body}, hdr: map[string]string{"Idempotency-Key": key}})
}

// TestOutreachCreatesTicketAndMessage: ticket TEACHER_OUTREACH ANSWERED (người gửi = claimed_by = answered_by), tin STAFF_ANSWER trong phiên "Tin nhắn từ giảng viên", outbox `escalation.answered`, audit không nội dung.
func TestOutreachCreatesTicketAndMessage(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	s := w.students[0].ID
	const text = "Em ghé gặp thầy sau giờ học nhé."
	res := w.outreach(w.T, 0, text, uuid.NewString())
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	tk := res.json()["ticket"].(map[string]any)
	require.Equal(t, "TEACHER_OUTREACH", tk["kind"])
	require.Equal(t, "ANSWERED", tk["status"])
	id := tk["id"].(string)
	require.Equal(t, w.teacher.ID.String()+"|"+w.teacher.ID.String()+"|STAFF_OUTREACH|STUDENT|"+s.String(), r.scalar(`select claimed_by::text||'|'||answered_by::text||'|'||origin::text||'|'||source_type||'|'||source_id::text from escalation_tickets where id = $1`, id))
	require.Equal(t, text, r.scalar(`select m.content from chat_messages m join chat_sessions cs on cs.id = m.session_id where cs.user_id = $1 and cs.title = 'Tin nhắn từ giảng viên' and m.intent = 'STAFF_ANSWER'`, s))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic = 'escalation.answered' and payload->>'ticket_id' = $1`, id))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'outreach' and entity_id = $1`, id))
	require.Equal(t, 0, r.count(`select count(*) from audit_log where course_id = $1 and (before::text like '%giờ học%' or after::text like '%giờ học%')`, w.course), "audit không có nội dung tin")
}

func TestOutreachIdempotent(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	key := uuid.NewString()
	a := w.outreach(w.T, 0, "Tin một", key)
	b := w.outreach(w.T, 0, "Tin một", key)
	require.Equal(t, http.StatusCreated, a.code)
	require.Equal(t, a.json()["ticket"].(map[string]any)["id"], b.json()["ticket"].(map[string]any)["id"])
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where kind = 'TEACHER_OUTREACH' and student_id = $1`, w.students[0].ID))
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where user_id = $1 and intent = 'STAFF_ANSWER'`, w.students[0].ID))
	require.Equal(t, http.StatusUnprocessableEntity, r.do(req{method: http.MethodPost, path: w.sp(0, "/outreach"), bearer: w.T.access, body: map[string]any{"body": "x"}}).code, "thiếu Idempotency-Key")
}

// TestOutreachBlocksOtherStudentPII: tên sinh viên KHÁC trong roster → 422 PII_IN_ANSWER, không ticket, không tin.
func TestOutreachBlocksOtherStudentPII(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.rename(1, "Phạm Thanh Sơn")
	res := w.outreach(w.T, 0, "Em xem bài của Phạm Thanh Sơn để tham khảo.", uuid.NewString())
	require.Equal(t, http.StatusUnprocessableEntity, res.code, string(res.body))
	require.Equal(t, "PII_IN_ANSWER", res.errCode())
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets where kind = 'TEACHER_OUTREACH'`))
	require.Equal(t, http.StatusUnprocessableEntity, w.outreach(w.T, 0, strings.Repeat("a", 2001), uuid.NewString()).code)
	require.Equal(t, http.StatusCreated, w.outreach(w.T, 0, "Chúc em học tốt.", uuid.NewString()).code, "tên của chính sinh viên thì được")
}

// TestOutreachDailyLimit: ≤ 3 tin / ngày tới một sinh viên (mọi nhân viên cộng lại); sinh viên khác không bị ảnh hưởng.
func TestOutreachDailyLimit(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	for i := range 3 {
		s := w.T
		if i == 1 {
			s = w.A
		}
		require.Equal(t, http.StatusCreated, w.outreach(s, 0, "Tin số "+string(rune('a'+i)), uuid.NewString()).code)
	}
	res := w.outreach(w.T, 0, "Tin thứ tư", uuid.NewString())
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "OUTREACH_LIMIT", res.errCode())
	require.NotEmpty(t, res.hdr.Get("Retry-After"))
	require.Equal(t, http.StatusCreated, w.outreach(w.T, 1, "Tin cho bạn khác", uuid.NewString()).code)
}

func TestOutreachStudentNotActive(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	gone := r.addUserRemoved(w)
	for _, id := range []uuid.UUID{w.pending.ID, gone.ID} {
		res := r.do(req{method: http.MethodPost, path: "/courses/" + w.course.String() + "/students/" + id.String() + "/outreach", bearer: w.T.access, body: map[string]any{"body": "x"}, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}})
		require.Equal(t, http.StatusConflict, res.code, string(res.body))
		require.Equal(t, "STUDENT_NOT_ACTIVE", res.errCode())
	}
	require.Equal(t, 0, r.count(`select count(*) from escalation_tickets where kind = 'TEACHER_OUTREACH'`))
}

// TestOutreachNoRiskTextToStudent: sinh viên AT_RISK nhận đúng thân tin nhân viên gõ — mọi thứ sinh viên đọc ở chat / thông báo không có nhãn hay lý do rủi ro.
func TestOutreachNoRiskTextToStudent(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.setRisk(w.students[0].ID, "AT_RISK", 0, 0)
	require.Equal(t, http.StatusCreated, w.outreach(w.T, 0, "Em ổn không? Thầy muốn trao đổi.", uuid.NewString()).code)
	// những gì sinh viên đọc được: tin chat riêng, thông báo, thư (trong cơ sở dữ liệu — rig này không có route chat)
	seen := r.scalar(`select coalesce(string_agg(m.content || ' ' || m.blocks::text, ' '), '') from chat_messages m where m.user_id = $1`, w.students[0].ID) + " " +
		r.scalar(`select coalesce(string_agg(n::text, ' '), '') from notifications n where n.user_id = $1`, w.students[0].ID) + " " +
		r.scalar(`select coalesce(string_agg(o::text, ' '), '') from outbox o where o.payload::text like '%'||$1||'%'`, w.students[0].ID.String())
	for _, bad := range []string{"AT_RISK", "rủi ro", "Rủi ro", "cần chú ý", "Cần chú ý", "WATCH", "risk"} {
		require.NotContains(t, seen, bad)
	}
	require.Contains(t, seen, "Em ổn không?")
}
