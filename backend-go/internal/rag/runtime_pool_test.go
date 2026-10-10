package rag_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestSearchWorksOnRuntimePool — QC BUG-1 của US-P3-05: truy xuất phải chạy được trên pool đúng cấu hình runtime (QueryExecModeExec, qua PgBouncer):
// `document_ids` rỗng và có giá trị đều mã hoá được (không "unable to encode []uuid.UUID … OID 0").
func TestSearchWorksOnRuntimePool(t *testing.T) {
	t.Parallel()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	var teacher, course, doc uuid.UUID
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test").Scan(&teacher))
	id := uuid.New()
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', 'ABCDEFG', $2) returning id`, "RP"+id.String()[:8], teacher).Scan(&course))
	require.NoError(t, pool.QueryRow(t.Context(), `insert into documents (course_id, title, status) values ($1, 'Quy chế', 'READY') returning id`, course).Scan(&doc))
	_, err := pool.Exec(t.Context(), `insert into content_chunks (document_id, course_ids, audience, ord, text, embedding) values ($1, array[$2::uuid], 'ALL', 0, 'Cảnh báo học vụ khi điểm trung bình dưới 1,2.', array_fill(0.25::real, array[1536])::vector)`, doc, course)
	require.NoError(t, err)
	vec := make([]float32, rag.Dims)
	for i := range vec {
		vec[i] = 0.25
	}
	svc := &rag.Service{DB: pool}
	for name, ids := range map[string][]uuid.UUID{"nil": nil, "rỗng": {}, "một tài liệu": {doc}, "tài liệu khác": {uuid.New()}} {
		hits, err := svc.SearchStudent(t.Context(), rag.Query{CourseID: course, Vec: vec, Text: "cảnh báo học vụ", DocumentIDs: ids})
		require.NoError(t, err, name)
		if name == "tài liệu khác" {
			require.Empty(t, hits)
		} else {
			require.Len(t, hits, 1, name)
		}
	}
}
