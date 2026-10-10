package chat_test

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/rag"
)

var numbers = regexp.MustCompile(`"(confidence|retrieval_score|groundedness)"`)

func (h *httpRig) qaWith(cos float64, answer string) {
	h.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		hits := []rag.Hit{{ChunkID: uuid.New(), DocumentID: uuid.New(), Title: "Quy chế học vụ", Text: "Sinh viên được thi lại tối đa một lần đối với mỗi học phần.", Cosine: cos}}
		return agent.Outcome{Plan: agent.IntentCourseQA, Hits: hits, Stream: h.ag.stream(ctx, nil, time.Millisecond, llm.Response{}, answer)}, nil
	}
}

func (h *httpRig) sendHTTP(text string) (sse string) {
	h.t.Helper()
	res := h.do(http.MethodPost, "/chat/sessions/"+h.sess.String()+"/messages", h.tok(h.student.UserID, auth.RoleStudent), `{"content":"`+text+`"}`, "Idempotency-Key", uuid.NewString())
	defer func() { _ = res.Body.Close() }()
	require.Equal(h.t, 200, res.StatusCode)
	b, err := io.ReadAll(res.Body)
	require.NoError(h.t, err)
	h.svc.Wait()
	return string(b)
}

// TestStudentNeverSeesConfidence — US-P3-07 AC3: khung SSE và danh sách tin của sinh viên không có confidence / retrieval_score / groundedness; chỉ `low_confidence` (bool).
// Độ tin cậy vẫn được tính và LƯU (cột confidence) cho Staff ở Threads — nhưng không có đường đọc nào của chat.
func TestStudentNeverSeesConfidence(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	h.qaWith(0.30, "Cổng mạng hai nghìn bốn mươi tám bit mã hóa không đối xứng.") // retr 0,125, ground 0 → 0,075 < ngưỡng lớp
	sse := h.sendHTTP("hỏi 1")
	require.NotRegexp(t, numbers, sse)
	require.Contains(t, sse, `"low_confidence":true`)
	h.qaWith(0.65, "Sinh viên được thi lại tối đa một lần đối với mỗi học phần.") // 1,000
	sse2 := h.sendHTTP("hỏi 2")
	require.NotRegexp(t, numbers, sse2)
	require.Contains(t, sse2, `"low_confidence":false`)

	res := h.do(http.MethodGet, "/chat/sessions/"+h.sess.String()+"/messages", h.tok(h.student.UserID, auth.RoleStudent), "")
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	require.Equal(t, 200, res.StatusCode)
	require.NotRegexp(t, numbers, string(b))
	require.Contains(t, string(b), `"low_confidence":true`)
	require.Contains(t, string(b), `"low_confidence":false`)
	// nhưng được tính và lưu
	require.Equal(t, 1, h.count(`select count(*) from chat_messages where role='ASSISTANT' and low_confidence and confidence = 0.075`))
	require.Equal(t, 1, h.count(`select count(*) from chat_messages where role='ASSISTANT' and not low_confidence and confidence = 1.000`))
}

// TestStaffCannotReadPrivateConfidence — AC5: TA / TEACHER / ADMIN không có đường đọc tin chat riêng (kể cả `confidence`): mọi route đọc trả 403 / 404, thân không chứa số.
func TestStaffCannotReadPrivateConfidence(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	h.qaWith(0.30, "Cổng mạng hai nghìn bốn mươi tám bit mã hóa không đối xứng.")
	_ = h.sendHTTP("hỏi")
	mid := h.msgID()
	for role, g := range map[string]string{"TA": "TA", "TEACHER": "TEACHER", "ADMIN": "ADMIN"} {
		id := h.user(role, g)
		if g != "ADMIN" {
			h.enroll(h.course, id, role, "ACTIVE")
		}
		for _, p := range []string{"/chat/sessions/" + h.sess.String() + "/messages", "/chat/messages/" + mid + "/stream", "/chat/sessions?course_id=" + h.course.String()} {
			res := h.do(http.MethodGet, p, h.tok(id, auth.Role(g)), "")
			b, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if !strings.HasPrefix(p, "/chat/sessions?") { // danh sách phiên là của CHÍNH người gọi (rỗng) nên có thể 200
				require.Contains(t, []int{403, 404}, res.StatusCode, role+" "+p)
			}
			require.NotRegexp(t, numbers, string(b), role+" "+p)
			require.NotContains(t, string(b), "hỏi", role+" "+p+": lộ nội dung")
		}
	}
}

func (h *httpRig) msgID() string {
	var id uuid.UUID
	require.NoError(h.t, h.pool.QueryRow(h.t.Context(), `select id from chat_messages where role='ASSISTANT' order by created_at desc limit 1`).Scan(&id))
	return id.String()
}
