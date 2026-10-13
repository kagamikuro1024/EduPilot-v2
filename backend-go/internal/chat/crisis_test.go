package chat_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/exam"
)

// US-P4-07 (SRS FEAT-escalation-notify 4.9): tin khủng hoảng ở chat riêng — câu mẫu, không báo ai, không dấu vết ngoài chat_messages, không bị khoá giờ thi.
const crisisCanary = "CANARY-CRISIS-5X"

// TestCrisisNoSideEffects / TestCrisisNeverTicketed / TestCrisisCanaryOnlyInMessages — AC3.
func TestCrisisNoSideEffects(t *testing.T) {
	r := newRig(t)
	r.withTickets()
	r.answer(agent.IntentCrisis, false, false, true, agent.CrisisText(""))
	c, mid, err := r.send(r.student, r.sess, crisisCanary+" em không muốn sống nữa")
	require.NoError(t, err)
	require.Equal(t, chat.EvDone, c.last().Ev)
	require.Nil(t, c.last().D["ticket"], "không bao giờ có ticket, kể cả tự động")
	require.Zero(t, r.count(`select count(*) from escalation_tickets`), "TestCrisisNeverTicketed")
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id = $1 and intent = 'CRISIS'`, mid))
	// không thông báo / thư cho ai
	require.Zero(t, r.count(`select count(*) from notifications`))
	require.Zero(t, r.count(`select count(*) from mail_outbox`))
	// canary chỉ ở chat_messages.content: mọi bảng khác (và log ứng dụng) sạch — TestCrisisCanaryOnlyInMessages
	var tables []string
	rows, err := r.pool.Query(t.Context(), `select table_name from information_schema.tables where table_schema = 'public' and table_type = 'BASE TABLE' and table_name <> 'chat_messages'`)
	require.NoError(t, err)
	for rows.Next() {
		var n string
		require.NoError(t, rows.Scan(&n))
		tables = append(tables, n)
	}
	rows.Close()
	for _, tb := range tables {
		require.Zero(t, r.count(fmt.Sprintf(`select count(*) from %q x where x::text like '%%%s%%'`, tb, crisisCanary)), "bảng %s", tb)
	}
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where role = 'USER' and content like '%`+crisisCanary+`%'`))
	require.NotContains(t, r.logs.String(), crisisCanary)
}

// TestCrisisAllowedDuringExamLock — AC5: đường hỗ trợ không bị khoá giờ thi, không tính CHAT_BLOCKED; tin thường vẫn bị khoá.
func TestCrisisAllowedDuringExamLock(t *testing.T) {
	r := newRig(t)
	r.lock.lock = &exam.Lock{AttemptID: uuid.New(), Until: time.Now().Add(30 * time.Minute).UTC()}
	r.answer(agent.IntentCrisis, false, false, true, agent.CrisisText(""))
	c, _, err := r.send(r.student, r.sess, "em muốn chết quá")
	require.NoError(t, err)
	require.Equal(t, chat.EvDone, c.last().Ev)
	require.Empty(t, r.lock.blocked, "không đếm RecordChatBlocked")
	_, err = r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "TCP là gì?")
	require.Equal(t, 409, httpStatus(err), "tin thường vẫn bị khoá")
	require.Len(t, r.lock.blocked, 1)
}

// TestStaffCannotReadCrisisMessages — AC8: chỉ chủ phiên đọc được tin; TA, TEACHER, sinh viên khác bị từ chối (403 / 404).
func TestStaffCannotReadCrisisMessages(t *testing.T) {
	r := newRig(t)
	r.answer(agent.IntentCrisis, false, false, true, agent.CrisisText(""))
	_, mid, err := r.send(r.student, r.sess, "em muốn chết quá")
	require.NoError(t, err)
	teacher := r.user("GV 2", "TEACHER")
	r.enroll(r.course, teacher, "TEACHER", "ACTIVE")
	ta := chat.Actor{UserID: r.user("TA", "TA"), Role: auth.RoleTA}
	r.enroll(r.course, ta.UserID, "TA", "ACTIVE")
	for name, a := range map[string]chat.Actor{"giảng viên": {UserID: teacher, Role: auth.RoleTeacher}, "trợ giảng": ta, "sinh viên khác": r.other} {
		_, err := r.svc.Messages(t.Context(), a, r.sess, pageOf(10))
		require.Contains(t, []int{403, 404}, httpStatus(err), name) // Staff: 403 (không có chat riêng); sinh viên khác: 404 (không lộ phiên)
		_, err = r.svc.Message(t.Context(), a, mid)
		require.Contains(t, []int{403, 404}, httpStatus(err), name)
	}
	page, err := r.svc.Messages(t.Context(), r.student, r.sess, pageOf(10))
	require.NoError(t, err)
	require.NotEmpty(t, page.Items)
}
