package thread_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/library"
)

// TestThreadLibraryQuestionUsesSearchLibrary — DEF-R6-P4 (TC-P8-02-33, SRS 213, US-P8-02 AC12): search_library chạy ở kênh Threads. Thread hỏi «có tài liệu nào về …»
// khi truy xuất ngữ nghĩa không có gì: bài AI vẫn trả lời từ kết quả tool, trích đúng tài liệu sinh viên được thấy; tài liệu ẩn không vào trích dẫn.
func TestThreadLibraryQuestionUsesSearchLibrary(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.hits = nil
	r.svc.Public = agent.NewPublicRegistry(agent.NewLibraryTool(&library.Service{Pool: r.pool}))
	var shown, hidden uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into documents (course_id, title, type, status, visible_to_students, use_for_rag) values ($1, 'Giáo trình an ninh mạng', 'LECTURE', 'READY', true, true) returning id`, r.course).Scan(&shown))
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into documents (course_id, title, type, status, visible_to_students, use_for_rag) values ($1, 'Đề an ninh mạng nội bộ', 'LECTURE', 'READY', false, true) returning id`, r.course).Scan(&hidden))
	for _, d := range []uuid.UUID{shown, hidden} {
		_, err := r.pool.Exec(t.Context(), `insert into content_chunks (document_id, course_ids, ord, text) values ($1, array[$2::uuid], 0, 'Chương một giới thiệu an ninh mạng và các mối đe doạ phổ biến.')`, d, r.course)
		require.NoError(t, err)
	}

	id := r.mustCreate(r.sv, "Tài liệu", "Có tài liệu nào về an ninh mạng không?")
	require.NoError(t, r.answer(id))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='ANSWERED'`, id))
	require.Equal(t, int32(1), r.llm.calls.Load(), "một lời gọi Chat")
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where thread_id=$1 and kind='AI' and citations @> jsonb_build_array(jsonb_build_object('document_id', $2::text))`, id, shown.String()))
	require.Equal(t, 0, r.count(`select count(*) from forum_posts where thread_id=$1 and citations::text like '%' || $2::text || '%'`, id, hidden.String()), "tài liệu ẩn không được trích")
}
