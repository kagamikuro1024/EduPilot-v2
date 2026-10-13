package thread_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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
