package escalation_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/store"
)

const canary = "CANARY-9K3Z"

// TestTicketDetailScopedToExchange + TestStaffCannotReadOtherChat — AC10: nhân viên chỉ đọc cặp hỏi–đáp đã nhờ (và tin của chính ticket);
// canary ở tin liền trước / sau, ở phiên khác và ở phiên đã xoá mềm không xuất hiện ở bất kỳ phản hồi nào.
func TestTicketDetailScopedToExchange(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.msg(r.course, r.sessA, r.a.UserID, "USER", canary+" trước", "", nil)
	u, m := r.pair("Câu hỏi được nhờ", "Trả lời của AI")
	r.msg(r.course, r.sessA, r.a.UserID, "USER", canary+" sau", "", nil)
	other := r.session(r.course, r.a.UserID)
	r.msg(r.course, other, r.a.UserID, "USER", canary+" phiên khác", "", nil)
	deleted := r.session(r.course, r.a.UserID)
	r.msg(r.course, deleted, r.a.UserID, "USER", canary+" phiên đã xoá", "", nil)
	r.exec(`update chat_sessions set deleted_at = now() where id = $1`, deleted)

	c, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	ex, err := r.svc.ReadExchange(t.Context(), r.course, c.Ticket.ID)
	require.NoError(t, err)
	require.NotNil(t, ex.Question)
	require.NotNil(t, ex.AIAnswer)
	require.Equal(t, u, ex.Question.ID)
	require.Equal(t, m, ex.AIAnswer.ID)
	require.Equal(t, "Câu hỏi được nhờ", ex.Question.Content)
	require.Empty(t, ex.Transcript)
	require.True(t, ex.AIAnswer.Confidence.Valid, "độ tin cậy có số cho giảng viên")
	raw, err := json.Marshal(ex)
	require.NoError(t, err)
	require.NotContains(t, string(raw), canary)

	// phiên đã xoá mềm sau khi nhờ: giảng viên vẫn thấy cặp hỏi–đáp (Q7)
	r.exec(`update chat_sessions set deleted_at = now() where id = $1`, r.sessA)
	ex, err = r.svc.ReadExchange(t.Context(), r.course, c.Ticket.ID)
	require.NoError(t, err)
	require.Equal(t, "Câu hỏi được nhờ", ex.Question.Content)
}

func TestStaffCannotReadOtherChat(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("Hỏi?", "Trả lời")
	c, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)
	// ticket của lớp này không đọc được qua lớp khác (course_id bắt buộc)
	_, err = r.svc.ReadExchange(t.Context(), r.course2, c.Ticket.ID)
	require.Equal(t, 404, status(err))
	// tin của chính ticket (STAFF_ANSWER mang khối ticket) có trong transcript; tin STAFF_ANSWER của ticket khác thì không
	blk := `[{"kind":"ticket","data":{"ticket_id":"` + c.Ticket.ID.String() + `"}}]`
	r.exec(`insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, intent, blocks) values ($1, $2, $3, 'ASSISTANT', 'Trả lời của giảng viên', 'DONE', 'STAFF_ANSWER', $4::jsonb)`, r.course, r.sessA, r.a.UserID, blk)
	r.exec(`insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, intent, blocks) values ($1, $2, $3, 'ASSISTANT', $4, 'DONE', 'STAFF_ANSWER', '[{"kind":"ticket","data":{"ticket_id":"00000000-0000-0000-0000-000000000000"}}]')`, r.course, r.sessA, r.a.UserID, canary)
	ex, err := r.svc.ReadExchange(t.Context(), r.course, c.Ticket.ID)
	require.NoError(t, err)
	require.Len(t, ex.Transcript, 1)
	require.Equal(t, "Trả lời của giảng viên", ex.Transcript[0].Content)
	raw, _ := json.Marshal(ex)
	require.NotContains(t, string(raw), canary)
}

// TestEscalationReadsChatOnlyViaTicketQueries — kiến trúc (SRS 4.3): trong escalation.sql chỉ GetEscalatedExchange / ListTicketTranscript chạm chat_messages / chat_sessions;
// mã Go của gói chỉ gọi các truy vấn chat đã duyệt (đọc tin CỦA CHÍNH sinh viên gọi, trạng thái lớp). Thêm truy vấn chat mới phải sửa danh sách này có chủ đích.
func TestEscalationReadsChatOnlyViaTicketQueries(t *testing.T) {
	allowedSQL := map[string]bool{"GetEscalatedExchange": true, "ListTicketTranscript": true}
	// Đọc tin CỦA CHÍNH sinh viên gọi (Escalate), trạng thái lớp, phiên của ticket; ghi tin STAFF_ANSWER; tin trả lời theo id trong payload outbox; phiên "Tin nhắn từ giảng viên".
	allowedChat := map[string]bool{"GetChatMessageOwned": true, "GetChatSession": true, "ChatCourseStatus": true, "ChatInsertStaffMessage": true, "ChatStaffSession": true,
		"InsertChatSession": true, "TouchChatSession": true, "GetChatMessageByID": true,
		"ChatMessageByClientID": true} // hỏi tiếp: khử trùng theo client_msg_id (US-P4-05)
	allowedSQL["TicketListStaff"] = true // proposals #13: đoạn cắt 140 ký tự của đúng câu hỏi đã nhờ cho danh sách

	body := func(path string) map[string]string {
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		out := map[string]string{}
		for _, part := range strings.Split(string(raw), "-- name: ")[1:] {
			name, rest, _ := strings.Cut(part, " ")
			out[name] = strings.ToLower(rest)
		}
		return out
	}
	touchesChat := func(s string) bool {
		return strings.Contains(s, "chat_messages") || strings.Contains(s, "chat_sessions")
	}

	for name, q := range body("../store/queries/escalation.sql") {
		if touchesChat(q) {
			require.True(t, allowedSQL[name], "escalation.sql: truy vấn %s chạm chat_messages / chat_sessions", name)
		}
	}
	chatQueries := map[string]bool{}
	files, err := filepath.Glob("../store/queries/chat.sql")
	require.NoError(t, err)
	for _, f := range files {
		for name, q := range body(f) {
			if touchesChat(q) {
				chatQueries[name] = true
			}
		}
	}
	src, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, f := range src {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for name := range chatQueries {
			if regexp.MustCompile(`\.` + name + `\(`).Match(raw) {
				require.True(t, allowedChat[name], "%s gọi truy vấn chat %s ngoài danh sách duyệt", f, name)
			}
		}
	}
	var _ = store.New // giữ import để rõ phạm vi: truy vấn sinh ở package store
}
