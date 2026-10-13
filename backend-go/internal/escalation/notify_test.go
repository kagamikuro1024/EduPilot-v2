package escalation_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

type capture struct {
	mu     sync.Mutex
	frames []string
}

func (c *capture) Publish(_ context.Context, userID, _ string, data any) (string, error) {
	b, _ := json.Marshal(data)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames = append(c.frames, userID+" "+string(b))
	return "1", nil
}

// TestNotificationNoPrivateText — US-P4-02 AC6: canary trong câu hỏi / trả lời của sinh viên không lọt ra chuông, khung SSE hay outbox.payload;
// người nhận là TA + TEACHER (không phải sinh viên) và dòng gộp chỉ có số lượng + mã lớp + liên kết.
func TestNotificationNoPrivateText(t *testing.T) {
	r := newRig(t, escalation.Config{})
	_, m := r.pair("CANARY-Q-9K3Z câu hỏi riêng tư của SV A", "CANARY-A-9K3Z trả lời của AI")
	c, err := r.svc.Escalate(t.Context(), r.a, m)
	require.NoError(t, err)

	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload from outbox where topic = 'escalation.created'`).Scan(&payload))
	require.NotContains(t, string(payload), "CANARY")
	require.NotContains(t, string(payload), "SV A")

	cap := &capture{}
	n := &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, cap, nil)}
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{Topic: escalation.TopicCreated, Payload: payload}))
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{Topic: escalation.TopicCreated, Payload: payload}), "giao lại: idempotent")

	require.Equal(t, 2, r.count(`select count(*) from notifications where type = 'TICKET_NEW'`), "một dòng cho mỗi người nhận (TEACHER, TA)")
	require.Zero(t, r.count(`select count(*) from notifications where user_id = $1`, r.a.UserID), "sinh viên không nhận thông báo ticket mới")
	require.Zero(t, r.count(`select count(*) from notifications where coalesce(title,'') || coalesce(body,'') ilike '%canary%' or coalesce(title,'') || coalesce(body,'') like '%SV A%'`))
	require.Len(t, cap.frames, 2, "một khung mỗi người; giao lại không phát lặp")
	for _, f := range cap.frames {
		require.NotContains(t, f, "CANARY")
		require.NotContains(t, f, "SV A")
		require.Contains(t, f, "1 câu hỏi đang chờ giảng viên")
	}
	_ = c
}

// TestAnswerNotifiesStudentOnce: `escalation.answered` → đúng MỘT thông báo cho sinh viên chủ ticket (giao lại không nhân đôi), liên kết tới phiên chứa tin trả lời,
// không chứa nội dung trả lời; dòng gộp TICKET_NEW của nhân viên được đếm lại (hết OPEN → đã đọc).
func TestAnswerNotifiesStudentOnce(t *testing.T) {
	r := newRig(t, escalation.Config{})
	id := r.ticket("Hỏi")
	cap := &capture{}
	n := &escalation.Notifier{Pool: r.pool, Notify: notify.New(r.pool, cap, nil)}
	var created []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload from outbox where topic = 'escalation.created'`).Scan(&created))
	require.NoError(t, n.HandleCreated(t.Context(), outbox.Message{Topic: escalation.TopicCreated, Payload: created}))
	require.Equal(t, 2, r.count(`select count(*) from notifications where type = 'TICKET_NEW'`))

	_, msg, err := r.svc.Answer(t.Context(), r.gv(), r.course, id, "BÍ-MẬT-TRẢ-LỜI cho em", nil)
	require.NoError(t, err)
	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload from outbox where topic = 'escalation.answered'`).Scan(&payload))
	require.NotContains(t, string(payload), "BÍ-MẬT")
	for range 2 {
		require.NoError(t, n.HandleAnswered(t.Context(), outbox.Message{Topic: escalation.TopicAnswered, Payload: payload}))
	}
	require.Equal(t, 1, r.count(`select count(*) from notifications where type = 'TICKET_ANSWERED' and user_id = $1 and link = '/chat?session=' || $2::text`, r.a.UserID, r.sessA))
	require.Zero(t, r.count(`select count(*) from notifications where type = 'TICKET_ANSWERED' and (coalesce(title,'') || coalesce(body,'')) like '%BÍ-MẬT%'`))
	require.Equal(t, 0, r.count(`select count(*) from notifications where type = 'TICKET_NEW' and user_id = $1 and read_at is null`, r.teacher), "hết ticket OPEN: dòng gộp thành đã đọc")
	_ = msg
}
