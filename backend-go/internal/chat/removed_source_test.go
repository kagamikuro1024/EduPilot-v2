package chat_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// DEF-R6-D1 (TC-P3-05-42 b, SRS FEAT-docs-calendar 105): chat cũ còn trích dẫn tài liệu đã xoá → mục đó `removed:true`, không còn tên / đoạn trích; mục còn tài liệu giữ nguyên.
func TestMessagesMarkRemovedSource(t *testing.T) {
	r := newRig(t)
	live := r.docIn(r.course, "READY", true, true, "LECTURE", false)
	gone := r.docIn(r.course, "READY", true, true, "LECTURE", false)
	cites, err := json.Marshal([]map[string]any{
		{"n": 1, "document_id": live, "title": "Còn", "page_no": 1, "snippet": "a"},
		{"n": 2, "document_id": gone, "title": "Đã xoá", "page_no": 2, "snippet": "b"},
	})
	require.NoError(t, err)
	_, err = r.pool.Exec(t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, citations) values ($1,$2,$3,'ASSISTANT','Trả lời [1][2]','DONE',$4)`, r.course, r.sess, r.student.UserID, cites)
	require.NoError(t, err)
	_, err = r.pool.Exec(t.Context(), `delete from documents where id = $1`, gone)
	require.NoError(t, err)

	page, err := r.svc.Messages(t.Context(), r.student, r.sess, pageOf(10))
	require.NoError(t, err)
	var got []map[string]any
	for _, m := range page.Items {
		if len(m.Citations) > 2 {
			require.NoError(t, json.Unmarshal(m.Citations, &got))
		}
	}
	require.Len(t, got, 2)
	require.Equal(t, "Còn", got[0]["title"])
	require.Nil(t, got[0]["removed"])
	require.Equal(t, true, got[1]["removed"])
	require.Empty(t, got[1]["title"])
	require.Empty(t, got[1]["snippet"])
}
