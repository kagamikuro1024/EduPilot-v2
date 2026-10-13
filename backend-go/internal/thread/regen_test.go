package thread_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/thread"
)

// US-P4-06 AC5 (#25): sửa thread khi bài AI còn PENDING → bài cũ REJECTED, AI sinh lại đúng MỘT bài mới; chỉ mục cho phép bài REJECTED cũ cạnh bài mới.
func TestEditRegeneratesAIAnswer(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	old := r.aiPost(id)
	_, err := r.edit(r.sv, id, "Hỏi lại", "Điều 6 quy chế nói gì?", 1, false)
	require.NoError(t, err)
	r.rag.hits = []rag.Hit{hit(0.8)}
	require.NoError(t, r.answer(id), "việc xếp lại chạy được")
	require.NoError(t, r.answer(id), "giao lại lần nữa: không thêm bài")
	var fresh uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id from forum_posts where thread_id=$1 and kind='AI' and verification_state='PENDING'`, id).Scan(&fresh))
	require.NotEqual(t, old, fresh)
	require.Equal(t, 2, r.count(`select count(*) from forum_posts where thread_id=$1 and kind='AI'`, id))
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where id=$1 and verification_state='REJECTED'`, old))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='ANSWERED'`, id))
	// danh sách: một dòng, trạng thái của bài MỚI; chi tiết: bài cũ chỉ còn dấu "đã loại"
	p, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, pg(10))
	require.NoError(t, err)
	require.Len(t, p.Items, 1)
	require.NotNil(t, p.Items[0].AnswerState)
	require.Equal(t, "PENDING", *p.Items[0].AnswerState)
	// loại bài mới rồi xác nhận được: gỡ tri thức đúng bài
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, fresh, thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from knowledge_items where source_type='POST' and source_id=$1`, fresh))
}

// TestKnowledgeCitationLinksThread — US-P4-06 AC12 / QC TC-42: đoạn tri thức nguồn POST trả kèm `thread_id` để giao diện liên kết `/threads/{id}`; tài liệu thường thì không.
func TestKnowledgeCitationLinksThread(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	r.syncKnowledge()
	vec := make([]float32, rag.Dims)
	vec[0] = 1
	_, err = r.pool.Exec(t.Context(), `update content_chunks set embedding = $1::vector where id in (select chunk_id from knowledge_items where source_type = 'POST')`, vecLit(vec))
	require.NoError(t, err)
	hits, err := (&rag.Service{DB: r.pool}).SearchStudent(t.Context(), rag.Query{CourseID: r.course, Vec: vec, Text: "Câu hỏi", K: 5})
	require.NoError(t, err)
	var seen int
	for _, h := range hits {
		if h.Knowledge {
			seen++
			require.NotNil(t, h.ThreadID)
			require.Equal(t, id, *h.ThreadID)
			c := agent.CitationOf(1, h)
			raw, _ := json.Marshal(c)
			require.Contains(t, string(raw), `"thread_id":"`+id.String()+`"`)
		}
	}
	require.Equal(t, 1, seen, "đoạn tri thức có mặt trong kết quả tìm")
	// bài bị ẩn: tri thức gỡ, không còn liên kết nào
	_, err = r.svc.Hide(t.Context(), r.teacher, r.course, thread.KindThread, id, "Lạc đề", int32(r.count(`select version from forum_threads where id=$1`, id)))
	require.NoError(t, err)
	hits, err = (&rag.Service{DB: r.pool}).SearchStudent(t.Context(), rag.Query{CourseID: r.course, Vec: vec, Text: "Câu hỏi", K: 5})
	require.NoError(t, err)
	for _, h := range hits {
		require.False(t, h.Knowledge)
	}
}
