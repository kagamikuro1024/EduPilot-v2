package escalation_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/knowledge"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/privacy"
)

// kAnswered: ticket của A đã trả lời; hỏi và đáp đều nêu tên A (để thử che).
func (r *rig) kAnswered(q string) uuid.UUID {
	r.t.Helper()
	id := r.ticket(q)
	_, _, err := r.svc.Answer(r.t.Context(), r.gv(), r.course, id, "Chào SV A, hạn nộp bài 3 là thứ Sáu tuần 5.", nil)
	require.NoError(r.t, err)
	return id
}

// TestKnowledgePreviewRedacts — AC10: bản xem trước đã che tên người hỏi; không che thì chuỗi gốc có tên.
func TestKnowledgePreviewRedacts(t *testing.T) {
	r := piiRig(t)
	id := r.kAnswered("SV A hỏi hạn nộp bài 3 là khi nào?")
	p, err := r.svc.KnowledgePreview(t.Context(), r.course, id)
	require.NoError(t, err)
	require.NotContains(t, p.Question, "SV A")
	require.NotContains(t, p.Answer, "SV A")
	require.Contains(t, p.Question, privacy.RedactMark)
	require.Contains(t, p.Answer, privacy.RedactMark)
	require.Contains(t, p.Answer, "thứ Sáu tuần 5")
	require.False(t, p.Personal)

	open := r.ticket("Hỏi")
	_, err = r.svc.KnowledgePreview(t.Context(), r.course, open)
	require.Equal(t, apierr.TicketStateInvalid, code(err), "OPEN: chưa có gì để lưu")
}

// TestKnowledgeRejectsPII — AC10: bản đã sửa còn PII → 422 PII_DETECTED, không ghi gì (canary: tên người hỏi).
func TestKnowledgeRejectsPII(t *testing.T) {
	r := piiRig(t)
	id := r.kAnswered("Hỏi hạn nộp bài 3")
	_, err := r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, id, "SV A hỏi hạn nộp bài 3?", "Thứ Sáu.")
	require.Equal(t, apierr.PIIDetected, code(err))
	require.Equal(t, 422, status(err))
	_, err = r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, id, "Hạn nộp bài 3?", "Gửi tới a@example.com nhé.")
	require.Equal(t, apierr.PIIDetected, code(err), "email trong trả lời")
	require.Zero(t, r.count(`select count(*) from content_chunks where $1 = any(course_ids)`, r.course))
	require.Zero(t, r.count(`select count(*) from knowledge_items where course_id = $1`, r.course))
	require.Zero(t, r.count(`select count(*) from outbox where topic = 'knowledge.sync'`))
}

// TestKnowledgeRejectsPersonalQuestion — AC10: câu hỏi gốc là câu hỏi riêng (điểm của chính mình) → 422 KNOWLEDGE_PERSONAL.
func TestKnowledgeRejectsPersonalQuestion(t *testing.T) {
	r := piiRig(t)
	id := r.kAnswered("Điểm giữa kỳ của em là bao nhiêu?")
	p, err := r.svc.KnowledgePreview(t.Context(), r.course, id)
	require.NoError(t, err)
	require.True(t, p.Personal)
	_, err = r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, id, "Điểm giữa kỳ tính thế nào?", "Theo quy chế.")
	require.Equal(t, apierr.KnowledgePersonal, code(err))
	require.Equal(t, 422, status(err))
	require.Zero(t, r.count(`select count(*) from knowledge_items where course_id = $1`, r.course))
}

// TestKnowledgeSaveOnce — AC10: một giao dịch ghi đủ (đoạn embedding NULL, mục, outbox, cờ, audit); lưu hai lần = cập nhật; OPEN → 409.
func TestKnowledgeSaveOnce(t *testing.T) {
	r := piiRig(t)
	id := r.kAnswered("Hỏi hạn nộp bài 3")
	st, err := r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, id, "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 5.")
	require.NoError(t, err)
	require.True(t, st.Saved)
	require.False(t, st.Loaded, "chưa nạp: embedding NULL")
	_, err = r.svc.KnowledgeSave(t.Context(), r.taA(), r.course, id, "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 6.")
	require.NoError(t, err)

	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where course_id = $1 and source_type = 'TICKET' and source_id = $2`, r.course, id))
	require.Equal(t, 1, r.count(`select count(*) from content_chunks where $1 = any(course_ids) and embedding is null and text like '%tuần 6%'`, r.course))
	require.Equal(t, 2, r.count(`select count(*) from outbox where topic = 'knowledge.sync' and payload->>'course_id' = $1`, r.course.String()))
	require.Equal(t, 1, r.count(`select count(*) from escalation_tickets where id = $1 and save_as_knowledge`, id))
	require.Equal(t, 2, r.count(`select count(*) from audit_log where action = 'escalation_ticket.knowledge_saved' and entity_id = $1`, id.String()))
	d, err := r.svc.KnowledgeOf(t.Context(), r.course, id)
	require.NoError(t, err)
	require.True(t, d.Saved)
	require.Equal(t, "Hạn nộp bài 3 là khi nào?", d.Question)

	open := r.ticket("Hỏi khác")
	_, err = r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, open, "Q?", "A.")
	require.Equal(t, apierr.TicketStateInvalid, code(err))

	_, err = r.svc.KnowledgeRemove(t.Context(), r.gv(), r.course, id)
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from knowledge_items where course_id = $1`, r.course))
	require.Zero(t, r.count(`select count(*) from escalation_tickets where id = $1 and save_as_knowledge`, id))
	_, err = r.svc.KnowledgeRemove(t.Context(), r.gv(), r.course, id)
	require.NoError(t, err, "gỡ lần hai: idempotent")
}

// TestKnowledgeRBAC — AC10 / AC14: STUDENT, ADMIN, chưa đăng nhập, TA của lớp khác → 403 / 401 ở cả ba route; TA của lớp có quyền.
func TestKnowledgeRBAC(t *testing.T) {
	r := piiRig(t)
	h := r.api(t)
	id := r.kAnswered("Hỏi hạn nộp bài 3")
	other := r.user("TA2", "TA")
	r.enroll(r.course2, other, "TA", "ACTIVE")
	base := "/courses/" + r.course.String() + "/tickets/" + id.String()
	body := `{"question":"Hạn nộp bài 3?","answer":"Thứ Sáu."}`
	js := []string{"Content-Type", "application/json"}
	routes := []struct{ method, path, body string }{
		{"GET", base + "/knowledge-preview", ""}, {"POST", base + "/knowledge", body}, {"DELETE", base + "/knowledge", ""},
	}
	for _, rt := range routes {
		for _, w := range []who{r.whoAll()["sv"], r.whoAll()["admin"], {other, "TA"}} {
			c, _ := call(h, w, rt.method, rt.path, rt.body, js...)
			require.Equal(t, 403, c, rt.method+" "+rt.path+" "+w.role)
		}
		c, _ := call(h, who{}, rt.method, rt.path, rt.body, js...)
		require.Equal(t, 401, c, rt.method+" không đăng nhập")
	}
	c, _ := call(h, r.whoAll()["ta"], "POST", base+"/knowledge", body, js...)
	require.Equal(t, 201, c)
	c, _ = call(h, r.whoAll()["gv"], "POST", base+"/knowledge", body, js...)
	require.Equal(t, 200, c, "lưu lại = cập nhật")
	c, _ = call(h, r.whoAll()["gv"], "DELETE", base+"/knowledge", "")
	require.Equal(t, 204, c)
}

// TestKnowledgeEditedTextIsWhatIsEmbedded — AC10: chuỗi được Embed là bản giảng viên đã sửa, không phải câu hỏi gốc của sinh viên.
func TestKnowledgeEditedTextIsWhatIsEmbedded(t *testing.T) {
	r := piiRig(t)
	id := r.kAnswered("Mình quên mất hạn nộp bài 3 rồi huhu")
	_, err := r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, id, "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 5.")
	require.NoError(t, err)
	var chunk uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select chunk_id from knowledge_items where source_id = $1`, id).Scan(&chunk))
	var embedded []string
	s := &knowledge.Syncer{Pool: r.pool, Embed: func(_ context.Context, text string) ([]float32, error) {
		embedded = append(embedded, text)
		return make([]float32, 1536), nil
	}}
	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload from outbox where topic = 'knowledge.sync' and payload->>'chunk_id' = $1 order by created_at desc limit 1`, chunk.String()).Scan(&payload))
	p := payload
	require.NoError(t, s.Handle(t.Context(), outbox.Message{Topic: knowledge.TopicSync, Payload: p}))
	require.Len(t, embedded, 1)
	require.Contains(t, embedded[0], "Hạn nộp bài 3 là khi nào?")
	require.False(t, strings.Contains(embedded[0], "huhu"), "câu gốc của sinh viên không vào embedding")
	d, err := r.svc.KnowledgeOf(t.Context(), r.course, id)
	require.NoError(t, err)
	require.True(t, d.Loaded, "đã nạp")
}

// QC US-P4-05 BUG-1: handler `knowledge.sync` chạy ở worker không có Principal, nên ctx của lời gọi Embed phải mang lớp (llm.WithIdentity) — nếu không tường lửa PII trả MASK_FAILED.
func TestKnowledgeSyncEmbedCarriesCourseIdentity(t *testing.T) {
	r := piiRig(t)
	id := r.kAnswered("Hạn nộp bài 3?")
	_, err := r.svc.KnowledgeSave(t.Context(), r.gv(), r.course, id, "Hạn nộp bài 3 là khi nào?", "Hết thứ Sáu tuần 5.")
	require.NoError(t, err)
	var chunk uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select chunk_id from knowledge_items where source_id = $1`, id).Scan(&chunk))
	var got *uuid.UUID
	s := &knowledge.Syncer{Pool: r.pool, Embed: func(ctx context.Context, _ string) ([]float32, error) {
		got = llm.IdentityFrom(ctx).CourseID
		return make([]float32, 1536), nil
	}}
	var payload []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload from outbox where topic = 'knowledge.sync' and payload->>'chunk_id' = $1 order by created_at desc limit 1`, chunk.String()).Scan(&payload))
	require.NoError(t, s.Handle(t.Context(), outbox.Message{Topic: knowledge.TopicSync, Payload: payload}))
	require.NotNil(t, got, "ctx của Embed không có lớp")
	require.Equal(t, r.course, *got)
}
