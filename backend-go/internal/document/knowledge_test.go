package document_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/knowledge"
)

// knowledgeDoc ghi một mục tri thức (đã nhúng) vào lớp và trả id tài liệu ảo.
func (f *fx) knowledgeDoc(t *testing.T, course uuid.UUID) uuid.UUID {
	t.Helper()
	tx, err := f.pool.Begin(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(t.Context()) }()
	chunk, err := knowledge.Upsert(t.Context(), tx, knowledge.Source{CourseID: course, Type: knowledge.SourceTicket, ID: uuid.New()}, "Hạn nộp bài 3?", "Thứ Sáu.")
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
	_, err = f.pool.Exec(t.Context(), `update content_chunks set embedding = $2::vector where id = $1`, chunk, vecLit(7))
	require.NoError(t, err)
	var doc uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `select document_id from content_chunks where id = $1`, chunk).Scan(&doc))
	return doc
}

// TestKnowledgeDocHiddenFromLibraryAndDocuments — AC11: tài liệu ảo của kho tri thức không lọt vào danh sách / chi tiết / thống kê của giảng viên.
func TestKnowledgeDocHiddenFromLibraryAndDocuments(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	real := f.doc(t, f.course, "Bài giảng", "LECTURE", true, true, 2)
	kd := f.knowledgeDoc(t, f.course)

	items := f.list(t, document.ListFilter{})
	require.Len(t, items, 1)
	require.Equal(t, real, items[0].ID)
	_, err := f.svc.Get(t.Context(), f.course, kd)
	require.Equal(t, "NOT_FOUND", code(t, err))
	s, err := f.svc.Stats(t.Context(), f.course)
	require.NoError(t, err)
	require.EqualValues(t, 1, s.Total)
	require.Equal(t, map[string]int64{"LECTURE": 1}, s.ByType)
	require.Equal(t, map[string]int64{"READY": 1}, s.ByStatus)
	_, err = f.patch(t, kd, document.PatchIn{Title: ptr("Đổi tên"), Version: 1})
	require.Equal(t, "NOT_FOUND", code(t, err), "không sửa được tài liệu ảo")
	require.Equal(t, "NOT_FOUND", code(t, f.svc.Delete(t.Context(), f.teacher, f.course, kd, "")), "không xoá được tài liệu ảo")
}

// TestDocumentRejectsReservedCategory — AC11: giáo viên không tạo / sửa tài liệu có category `THREAD_KNOWLEDGE` (422).
func TestDocumentRejectsReservedCategory(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài giảng", "LECTURE", true, true, 1)
	reserved := func(err error) {
		t.Helper()
		st, c := apiCode(t, err)
		require.Equal(t, 422, st)
		require.Equal(t, "VALIDATION_FAILED", c)
		var ae *apierr.Error
		require.ErrorAs(t, err, &ae)
		require.Contains(t, fmt.Sprint(ae.Details), "reserved", "lỗi đúng trường category chứ không phải lỗi khác")
	}
	_, err := f.patch(t, id, document.PatchIn{Category: ptr(knowledge.Category), Version: 1})
	reserved(err)
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, document.CompleteIn{UploadID: uuid.New(), Title: "x", Type: "LECTURE", Category: ptr(knowledge.Category)})
	reserved(err)
	require.Zero(t, f.count(t, `select count(*) from documents where category = 'THREAD_KNOWLEDGE'`))
}
