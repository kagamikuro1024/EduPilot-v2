package document_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

const dim = 1536

func vecLit(axis int) string {
	p := make([]string, dim)
	for i := range p {
		p[i] = "0"
	}
	p[axis] = "1"
	return "[" + strings.Join(p, ",") + "]"
}

// doc chèn tài liệu READY của `course` kèm `n` đoạn đã nhúng. Trả id tài liệu.
func (f *fx) doc(t *testing.T, course uuid.UUID, title, typ string, visible, rag bool, n int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into documents (course_id, title, type, filename, mime_type, size_bytes, sha256, blob_key, status, page_count, visible_to_students, use_for_rag, uploaded_by)
		values ($1, $2, $3::document_type, $4, 'application/pdf', 100, $5, $6, 'READY', 3, $7, $8, $9) returning id`,
		course, title, typ, title+".pdf", sum([]byte(uuid.NewString())), "courses/"+course.String()+"/documents/"+uuid.NewString()+"/x.pdf", visible, rag, f.teacher).Scan(&id))
	aud := "ALL"
	switch {
	case typ == "ANSWER_KEY":
		aud = "GRADING"
	case !visible:
		aud = "STAFF"
	}
	for i := range n {
		_, err := f.pool.Exec(t.Context(), `insert into content_chunks (document_id, course_ids, audience, ord, page_no, text, embedding) values ($1, array[$2::uuid], $3::chunk_audience, $4, $5, $6, $7::vector)`,
			id, course, aud, i, i+1, fmt.Sprintf("Đoạn %d của %s: cảnh báo học vụ", i, title), vecLit(i%dim))
		require.NoError(t, err)
	}
	return id
}

func (f *fx) share(t *testing.T, doc, to uuid.UUID) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `insert into document_courses (document_id, course_id, shared_by) values ($1, $2, $3)`, doc, to, f.teacher)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `update content_chunks set course_ids = course_ids || $2::uuid where document_id = $1`, doc, to)
	require.NoError(t, err)
}

func pg(n int) httpx.PageParams { return httpx.PageParams{Limit: n} }

func (f *fx) get(t *testing.T, course, id uuid.UUID) document.Row {
	t.Helper()
	r, err := f.svc.Get(t.Context(), course, id)
	require.NoError(t, err)
	return r
}

func (f *fx) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), q, args...).Scan(&n))
	return n
}

func (f *fx) list(t *testing.T, flt document.ListFilter) []document.Row {
	t.Helper()
	p, err := f.svc.List(t.Context(), f.course, flt, pg(50))
	require.NoError(t, err)
	return p.Items
}

func code(t *testing.T, err error) string {
	t.Helper()
	require.Error(t, err)
	_, c := apiCode(t, err)
	return c
}

func (f *fx) patch(t *testing.T, id uuid.UUID, in document.PatchIn) (document.Doc, error) {
	t.Helper()
	return f.svc.Patch(t.Context(), f.teacher, f.course, id, "", in)
}

func ptr[T any](v T) *T { return &v }

// TestListDocuments — AC1: lọc loại / trạng thái; tìm tên không dấu; ký tự đặc biệt không thành mẫu.
func TestListDocuments(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.doc(t, f.course, "Bài giảng An ninh mạng", "LECTURE", true, true, 1)
	f.doc(t, f.course, "Quy chế môn học", "COURSE_POLICY", true, true, 1)
	_, err := f.pool.Exec(t.Context(), `insert into documents (course_id, title, type, status, error) values ($1, 'Hỏng', 'OTHER', 'FAILED', 'Không đọc được')`, f.course)
	require.NoError(t, err)
	require.Len(t, f.list(t, document.ListFilter{}), 3)
	for flt, want := range map[document.ListFilter]int{{Type: "LECTURE"}: 1, {Status: "FAILED"}: 1, {Q: "quy che mon hoc"}: 1, {Q: "AN NINH"}: 1, {Q: "100%"}: 0, {Q: "pdf"}: 2} {
		require.Len(t, f.list(t, flt), want, "%+v", flt)
	}
}

// TestListCursor — AC1 / luật 13: con trỏ, không trùng không sót.
func TestListCursor(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for i := range 5 {
		f.doc(t, f.course, fmt.Sprintf("Tài liệu %d", i), "LECTURE", true, true, 0)
	}
	seen := map[uuid.UUID]bool{}
	var cur *httpx.Cursor
	for range 5 {
		p, err := f.svc.List(t.Context(), f.course, document.ListFilter{}, httpx.PageParams{Limit: 2, Cursor: cur})
		require.NoError(t, err)
		for _, r := range p.Items {
			require.False(t, seen[r.ID])
			seen[r.ID] = true
		}
		if p.NextCursor == nil {
			break
		}
		c, err := httpx.ParseCursor(*p.NextCursor)
		require.NoError(t, err)
		cur = &c
	}
	require.Len(t, seen, 5)
}

// TestListSharedMarkedReadonly — AC1: tài liệu chia sẻ từ lớp khác có `shared_from` = mã lớp gốc.
func TestListSharedMarkedReadonly(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	other := f.newCourse(t, f.teacher, "ACTIVE")
	own := f.doc(t, f.course, "Của lớp", "LECTURE", true, true, 1)
	shared := f.doc(t, other, "Chia sẻ", "LECTURE", true, true, 1)
	f.share(t, shared, f.course)
	var code string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select class_code from courses where id=$1`, other).Scan(&code))
	byID := map[uuid.UUID]document.Row{}
	for _, r := range f.list(t, document.ListFilter{}) {
		byID[r.ID] = r
	}
	require.Empty(t, byID[own].SharedFrom)
	require.Equal(t, code, byID[shared].SharedFrom)
}

// TestAnswerKeyCannotBeVisible — AC3: đáp án không bật được `visible_to_students` (422 ANSWER_KEY_NOT_VISIBLE); đổi loại sang đáp án thì tự ẩn và audience → GRADING.
func TestAnswerKeyCannotBeVisible(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	key := f.doc(t, f.course, "Đáp án", "ANSWER_KEY", false, true, 2)
	_, err := f.patch(t, key, document.PatchIn{VisibleToStudents: ptr(true), Version: 1})
	st, c := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, apierr.AnswerKeyNotVisible, c)
	lec := f.doc(t, f.course, "Bài", "LECTURE", true, true, 2)
	d, err := f.patch(t, lec, document.PatchIn{Type: ptr("ANSWER_KEY"), Version: 1})
	require.NoError(t, err)
	require.False(t, d.VisibleToStudents)
	require.Equal(t, 2, f.count(t, `select count(*) from content_chunks where document_id=$1 and audience='GRADING'`, lec))
}

// TestPatchFlagsTakeEffectImmediately — AC4: đổi cờ có hiệu lực ngay ở dữ liệu (audience đoạn, truy xuất, thư viện) và phát document.changed.
func TestPatchFlagsTakeEffectImmediately(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 3)
	d, err := f.patch(t, id, document.PatchIn{VisibleToStudents: ptr(false), Version: 1})
	require.NoError(t, err)
	require.False(t, d.VisibleToStudents)
	require.Equal(t, int32(2), d.Version)
	require.Equal(t, 3, f.count(t, `select count(*) from content_chunks where document_id=$1 and audience='STAFF'`, id))
	require.Zero(t, f.count(t, `select count(*) from documents where id=$1 and visible_to_students`, id))
	require.Equal(t, 1, f.count(t, `select count(*) from outbox where topic='document.changed' and payload->>'document_id'=$1`, id.String()))
	d, err = f.patch(t, id, document.PatchIn{VisibleToStudents: ptr(true), Title: ptr("Tên mới"), WeekNo: ptr(int16(4)), Category: ptr("Mạng"), Version: 2})
	require.NoError(t, err)
	require.Equal(t, "Tên mới", d.Title)
	require.Equal(t, 3, f.count(t, `select count(*) from content_chunks where document_id=$1 and audience='ALL'`, id))
	// bật Dùng cho AI trên tài liệu READY → xếp reindex
	off := f.doc(t, f.course, "Đề", "EXAM_PAPER", true, false, 1)
	_, err = f.patch(t, off, document.PatchIn{UseForRAG: ptr(true), Version: 1})
	require.NoError(t, err)
	require.Equal(t, 1, f.count(t, `select count(*) from outbox where topic='job.enqueue' and payload::text like '%'||$1||'%'`, off.String()))
	// kiểm tra đầu vào
	_, err = f.patch(t, id, document.PatchIn{Title: ptr("  "), Version: 3})
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
	_, err = f.patch(t, id, document.PatchIn{WeekNo: ptr(int16(21)), Version: 3})
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
	_, err = f.patch(t, id, document.PatchIn{Type: ptr("LẠ"), Version: 3})
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
}

// TestPatchVersionConflict — AC4: version cũ → 409 VERSION_CONFLICT kèm bản hiện tại; không đổi gì.
func TestPatchVersionConflict(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	_, err := f.patch(t, id, document.PatchIn{Title: ptr("A"), Version: 1})
	require.NoError(t, err)
	_, err = f.patch(t, id, document.PatchIn{Title: ptr("B"), Version: 1})
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, 409, ae.Status)
	require.Equal(t, apierr.VersionConflict, ae.Code)
	raw, _ := json.Marshal(ae.Details)
	require.Contains(t, string(raw), `"title":"A"`)
	require.Equal(t, "A", f.get(t, f.course, id).Title)
}

// TestPatchAudited — AC4: mỗi PATCH có audit_log trước / sau.
func TestPatchAudited(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	_, err := f.patch(t, id, document.PatchIn{Title: ptr("Mới"), Version: 1})
	require.NoError(t, err)
	require.Equal(t, 1, f.count(t, `select count(*) from audit_log where entity='document' and entity_id=$1 and action='document.patch' and before->>'title'='Bài' and after->>'title'='Mới'`, id.String()))
}

// TestPatchSharedReadonly — AC4, AC6, AC8: tài liệu chia sẻ từ lớp khác → 409 DOCUMENT_SHARED_READONLY (sửa, sửa đoạn, xoá).
func TestPatchSharedReadonly(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	other := f.newCourse(t, f.teacher, "ACTIVE")
	shared := f.doc(t, other, "Chia sẻ", "LECTURE", true, true, 1)
	f.share(t, shared, f.course)
	_, err := f.patch(t, shared, document.PatchIn{Title: ptr("X"), Version: 1})
	require.Equal(t, apierr.DocumentSharedReadonly, code(t, err))
	var chunk uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `select id from content_chunks where document_id=$1`, shared).Scan(&chunk))
	f.svc.Embed = func(context.Context, string) ([]float32, error) { return make([]float32, dim), nil }
	_, err = f.svc.EditChunk(t.Context(), f.teacher, f.course, shared, chunk, "", "sửa")
	require.Equal(t, apierr.DocumentSharedReadonly, code(t, err))
	require.Equal(t, apierr.DocumentSharedReadonly, code(t, f.svc.Delete(t.Context(), f.teacher, f.course, shared, "")))
	require.Equal(t, 1, f.count(t, `select count(*) from documents where id=$1`, shared))
}

// TestStatsHasCoursePolicy — AC5: has_course_policy chỉ khi có COURSE_POLICY READY (của lớp hoặc chia sẻ vào).
func TestStatsHasCoursePolicy(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	s, err := f.svc.Stats(t.Context(), f.course)
	require.NoError(t, err)
	require.False(t, s.HasCoursePolicy)
	_, err = f.pool.Exec(t.Context(), `insert into documents (course_id, title, type, status) values ($1, 'Quy chế', 'COURSE_POLICY', 'QUEUED')`, f.course)
	require.NoError(t, err)
	s, _ = f.svc.Stats(t.Context(), f.course)
	require.False(t, s.HasCoursePolicy, "chưa READY")
	other := f.newCourse(t, f.teacher, "ACTIVE")
	f.share(t, f.doc(t, other, "Quy chế khác", "COURSE_POLICY", true, true, 1), f.course)
	s, _ = f.svc.Stats(t.Context(), f.course)
	require.True(t, s.HasCoursePolicy)
}

// TestStats — AC7: tổng, theo loại / trạng thái, đoạn, đoạn đã nhúng, trang, byte, lần tải gần nhất.
func TestStats(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	empty, err := f.svc.Stats(t.Context(), f.course)
	require.NoError(t, err)
	require.Zero(t, empty.Total)
	require.Nil(t, empty.LastUploadAt)
	f.doc(t, f.course, "A", "LECTURE", true, true, 3)
	f.doc(t, f.course, "B", "COURSE_POLICY", true, true, 2)
	_, err = f.pool.Exec(t.Context(), `insert into content_chunks (document_id, course_ids, ord, text) select id, array[course_id], 99, 'chưa nhúng' from documents where title='A'`)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `insert into documents (course_id, title, type, status) values ($1, 'Đang chạy', 'OTHER', 'PROCESSING')`, f.course)
	require.NoError(t, err)
	s, err := f.svc.Stats(t.Context(), f.course)
	require.NoError(t, err)
	require.Equal(t, int64(3), s.Total)
	require.Equal(t, map[string]int64{"LECTURE": 1, "COURSE_POLICY": 1, "OTHER": 1}, s.ByType)
	require.Equal(t, map[string]int64{"READY": 2, "PROCESSING": 1}, s.ByStatus)
	require.Equal(t, int64(6), s.Chunks)
	require.Equal(t, int64(5), s.EmbeddedChunks)
	require.Equal(t, int64(6), s.Pages)
	require.Equal(t, int64(200), s.Bytes)
	require.NotNil(t, s.LastUploadAt)
	require.True(t, s.HasCoursePolicy)
}

func (f *fx) chunkOf(t *testing.T, doc uuid.UUID, ord int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `select id from content_chunks where document_id=$1 and ord=$2`, doc, ord).Scan(&id))
	return id
}

// TestEditChunkReEmbeds — AC6: sửa đoạn → một lần nhúng, lưu chữ + vectơ mới, document.changed, version tăng (ETag thư viện đổi).
func TestEditChunkReEmbeds(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 2)
	calls := 0
	f.svc.Embed = func(_ context.Context, text string) ([]float32, error) {
		calls++
		require.Equal(t, "Nội dung đã sửa.", text)
		v := make([]float32, dim)
		v[7] = 1
		return v, nil
	}
	chunk := f.chunkOf(t, id, 0)
	out, err := f.svc.EditChunk(t.Context(), f.teacher, f.course, id, chunk, "", "  Nội dung đã sửa.  ")
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, "Nội dung đã sửa.", out.Text)
	require.Equal(t, 1, f.count(t, `select count(*) from content_chunks where id=$1 and text='Nội dung đã sửa.' and embedding <=> $2::vector < 0.01`, chunk, vecLit(7)))
	require.Equal(t, int32(2), f.get(t, f.course, id).Version)
	require.Equal(t, 1, f.count(t, `select count(*) from outbox where topic='document.changed'`))
}

// TestEditChunkEmbedFailureKeepsOld — AC6: nhúng lỗi → 503, giữ nguyên chữ + vectơ cũ, không audit.
func TestEditChunkEmbedFailureKeepsOld(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	f.svc.Embed = func(context.Context, string) ([]float32, error) { return nil, errors.New("provider chết") }
	chunk := f.chunkOf(t, id, 0)
	_, err := f.svc.EditChunk(t.Context(), f.teacher, f.course, id, chunk, "", "mới")
	st, _ := apiCode(t, err)
	require.Equal(t, 503, st)
	require.Equal(t, 1, f.count(t, `select count(*) from content_chunks where id=$1 and text like 'Đoạn 0%' and embedding <=> $2::vector < 0.01`, chunk, vecLit(0)))
	require.Zero(t, f.count(t, `select count(*) from audit_log where entity='document_chunk'`))
}

// TestEditChunkValidation — AC6: rỗng hoặc > 4.000 ký tự → 422; đoạn của tài liệu khác → 404.
func TestEditChunkValidation(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.svc.Embed = func(context.Context, string) ([]float32, error) { return make([]float32, dim), nil }
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	other := f.doc(t, f.course, "Khác", "LECTURE", true, true, 1)
	chunk := f.chunkOf(t, id, 0)
	for _, text := range []string{"", "   ", strings.Repeat("a", 4001)} {
		_, err := f.svc.EditChunk(t.Context(), f.teacher, f.course, id, chunk, "", text)
		require.Equal(t, "VALIDATION_FAILED", code(t, err), len(text))
	}
	_, err := f.svc.EditChunk(t.Context(), f.teacher, f.course, id, chunk, "", strings.Repeat("a", 4000))
	require.NoError(t, err)
	_, err = f.svc.EditChunk(t.Context(), f.teacher, f.course, other, chunk, "", "x")
	require.Equal(t, "NOT_FOUND", code(t, err))
}

// TestEditChunkAudited — AC6: audit_log ghi chữ trước / sau.
func TestEditChunkAudited(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.svc.Embed = func(context.Context, string) ([]float32, error) { return make([]float32, dim), nil }
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	chunk := f.chunkOf(t, id, 0)
	_, err := f.svc.EditChunk(t.Context(), f.teacher, f.course, id, chunk, "", "Chữ sau")
	require.NoError(t, err)
	require.Equal(t, 1, f.count(t, `select count(*) from audit_log where entity='document_chunk' and entity_id=$1 and before->>'text' like 'Đoạn 0%' and after->>'text'='Chữ sau'`, chunk.String()))
	page, err := f.svc.Chunks(t.Context(), f.course, id, -1, 10)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "Chữ sau", page.Items[0].Text)
}

// TestDeleteDocument — AC8: xoá dòng, đoạn và document_courses theo CASCADE, document.changed, audit; xoá lần hai → 404.
func TestDeleteDocument(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	other := f.newCourse(t, f.teacher, "ACTIVE")
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 3)
	f.share(t, id, other)
	imp, err := f.svc.Impact(t.Context(), f.course, id)
	require.NoError(t, err)
	require.Equal(t, document.Impact{Chunks: 3, Courses: 1}, imp)
	require.NoError(t, f.svc.Delete(t.Context(), f.teacher, f.course, id, ""))
	require.Zero(t, f.count(t, `select count(*) from documents where id=$1`, id))
	require.Zero(t, f.count(t, `select count(*) from content_chunks where document_id=$1`, id)+f.count(t, `select count(*) from document_courses where document_id=$1`, id))
	require.Equal(t, 1, f.count(t, `select count(*) from outbox where topic='document.changed' and payload->>'document_id'=$1`, id.String()))
	require.Equal(t, 1, f.count(t, `select count(*) from audit_log where entity='document' and entity_id=$1 and action='document.delete'`, id.String()))
	require.Equal(t, "NOT_FOUND", code(t, f.svc.Delete(t.Context(), f.teacher, f.course, id, "")))
}

// TestDeleteCleansBlob — AC8: object trong MinIO bị xoá sau commit.
func TestDeleteCleansBlob(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	b := pdf()
	up := f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, true)
	out, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(up, "LECTURE"))
	require.NoError(t, err)
	var key string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select blob_key from documents where id=$1`, out.Document.ID).Scan(&key))
	_, err = f.blob.Stat(t.Context(), key)
	require.NoError(t, err)
	require.NoError(t, f.svc.Delete(t.Context(), f.teacher, f.course, out.Document.ID, ""))
	_, err = f.blob.Stat(t.Context(), key)
	require.Error(t, err)
}

// TestDeleteLeavesCitationsDangling — AC8: câu trả lời cũ giữ trích dẫn (JSON) dù tài liệu đã xoá — không FK, không lỗi.
func TestDeleteLeavesCitationsDangling(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	sv := f.user(t, "SV", "STUDENT")
	_, err := f.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, f.course, sv)
	require.NoError(t, err)
	var sid uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id`, f.course, sv).Scan(&sid))
	cites, _ := json.Marshal([]map[string]any{{"n": 1, "document_id": id, "title": "Bài"}})
	_, err = f.pool.Exec(t.Context(), `insert into chat_messages (session_id, course_id, user_id, role, content, stream_status, citations, client_msg_id) values ($1, $2, $3, 'ASSISTANT', 'Trả lời [1]', 'DONE', $4, gen_random_uuid())`, sid, f.course, sv, cites)
	require.NoError(t, err)
	require.NoError(t, f.svc.Delete(t.Context(), f.teacher, f.course, id, ""))
	require.Equal(t, 1, f.count(t, `select count(*) from chat_messages where citations->0->>'document_id'=$1`, id.String()))
}

// TestDeleteDocumentNullsChatSessions — AC8: phiên chat có document_id = tài liệu bị xoá chuyển NULL (ON DELETE SET NULL), không lỗi 23503.
func TestDeleteDocumentNullsChatSessions(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(t, f.course, "Bài", "LECTURE", true, true, 1)
	sv := f.user(t, "SV", "STUDENT")
	_, err := f.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, f.course, sv)
	require.NoError(t, err)
	var sid uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into chat_sessions (course_id, user_id, document_id) values ($1, $2, $3) returning id`, f.course, sv, id).Scan(&sid))
	require.NoError(t, f.svc.Delete(t.Context(), f.teacher, f.course, id, ""))
	require.Equal(t, 1, f.count(t, `select count(*) from chat_sessions where id=$1 and document_id is null`, sid))
}
