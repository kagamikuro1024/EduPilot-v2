package course_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// document chèn tài liệu + n chunk (audience cho trước, embedding thật ở chunk đầu) của lớp; trả id.
func (r *rig) document(courseID, typ, status string, visible bool, audience string, n int) string {
	r.t.Helper()
	id := r.scalar(`insert into documents (course_id, title, type, status, visible_to_students) values ($1, 'Tài liệu', $2::document_type, $3::document_status, $4) returning id::text`, courseID, typ, status, visible)
	for i := range n {
		_, err := r.pool.Exec(r.t.Context(), `insert into content_chunks (document_id, course_ids, audience, ord, text, embedding)
			values ($1, array[$2::uuid], $3::chunk_audience, $4::int, 'nội dung '||$4::int, array_fill(0.25::real, array[1536])::vector)`, id, courseID, audience, i)
		require.NoError(r.t, err)
	}
	return id
}

// twoClasses: hai lớp cùng học phần INT1006 với CÙNG giảng viên (a.gv) + lớp thứ ba cùng học phần của giảng viên khác + lớp khác học phần.
type twoClasses struct {
	k                        klass
	src, third, otherSubject string
}

func (r *rig) twoClasses() twoClasses {
	r.t.Helper()
	k := r.klass(nil)
	return twoClasses{
		k:            k,
		src:          r.mustOpen(k.a.A, map[string]any{"teacher_id": k.a.gv.ID}),
		third:        r.mustOpen(k.a.A, map[string]any{"teacher_id": k.a.gv2.ID}),
		otherSubject: r.mustOpen(k.a.A, map[string]any{"teacher_id": k.a.gv.ID, "subject_code": "INT2000"}),
	}
}

func (r *rig) share(s session, target, source string, what ...string) resp {
	return r.post(s, "/courses/"+target+"/share-from", map[string]any{"source_course_id": source, "what": what}, idemKey())
}

func TestShareDocuments(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	a := r.document(c.src, "LECTURE", "READY", true, "ALL", 3)
	b := r.document(c.src, "EXAM_PAPER", "READY", false, "STAFF", 2)
	queued := r.document(c.src, "LECTURE", "QUEUED", true, "ALL", 1)
	policy := r.document(c.src, "COURSE_POLICY", "READY", true, "ALL", 2)
	res := r.share(c.k.a.G, c.k.id, c.src, "documents")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, float64(2), res.json()["shared"].(map[string]any)["documents"])
	for _, d := range []string{a, b} {
		require.Equal(t, 1, r.count(`select count(*) from document_courses where document_id = $1 and course_id = $2`, d, c.k.id))
	}
	for _, d := range []string{queued, policy} {
		require.Equal(t, 0, r.count(`select count(*) from document_courses where document_id = $1 and course_id = $2`, d, c.k.id), "chưa READY / quy chế không chia sẻ")
		require.Equal(t, 0, r.count(`select count(*) from content_chunks where document_id = $1 and $2::uuid = any(course_ids)`, d, c.k.id))
	}
	require.Equal(t, 5, r.count(`select count(*) from content_chunks where $1::uuid = any(course_ids)`, c.k.id), "chunk của hai tài liệu được chia sẻ")
	require.Equal(t, 5, r.count(`select count(*) from content_chunks where $1::uuid = any(course_ids) and $2::uuid = any(course_ids)`, c.k.id, c.src), "lớp nguồn vẫn giữ")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where course_id = $1 and action = 'content_shared'`, c.k.id))
}

func TestShareExcludesCoursePolicy(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	r.document(c.src, "COURSE_POLICY", "READY", true, "ALL", 1)
	r.document(c.src, "LECTURE", "READY", true, "ALL", 1)
	body := r.share(c.k.a.G, c.k.id, c.src, "documents").json()
	require.Equal(t, float64(1), body["shared"].(map[string]any)["documents"])
	sk := body["skipped"].([]any)
	require.Len(t, sk, 1)
	require.Equal(t, "documents", sk[0].(map[string]any)["kind"])
	require.Contains(t, sk[0].(map[string]any)["reason"], "Quy chế")
}

func TestShareNoReembed(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	d := r.document(c.src, "LECTURE", "READY", true, "ALL", 4)
	snap := func() string {
		return r.scalar(`select count(*)||'|'||md5(string_agg(embedding::text||audience::text||ord::text||text, ',' order by ord)) from content_chunks where document_id = $1`, d)
	}
	before := snap()
	require.Equal(t, http.StatusOK, r.share(c.k.a.G, c.k.id, c.src, "documents").code)
	require.Equal(t, before, snap(), "embedding, audience, số dòng chunk không đổi")
}

func TestShareIdempotent(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	d := r.document(c.src, "LECTURE", "READY", true, "ALL", 2)
	first := r.share(c.k.a.G, c.k.id, c.src, "documents")
	second := r.share(c.k.a.G, c.k.id, c.src, "documents")
	require.Equal(t, http.StatusOK, second.code)
	require.JSONEq(t, string(first.body), string(second.body))
	require.Equal(t, 1, r.count(`select count(*) from document_courses where document_id = $1`, d), "đúng một dòng cho lớp đích (lớp nguồn sở hữu qua documents.course_id)")
	require.Equal(t, 2, r.count(`select count(*) from content_chunks where document_id = $1 and cardinality(course_ids) = 2`, d), "không thêm trùng vào course_ids")
}

func TestShareSubjectMismatch(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	r.document(c.otherSubject, "LECTURE", "READY", true, "ALL", 1)
	res := r.share(c.k.a.G, c.k.id, c.otherSubject, "documents")
	require.Equal(t, http.StatusUnprocessableEntity, res.code, string(res.body))
	require.Contains(t, string(res.body), "SUBJECT_MISMATCH")
	require.Equal(t, 0, r.count(`select count(*) from document_courses`))
}

func TestShareRequiresSourceTeacher(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	r.document(c.third, "LECTURE", "READY", true, "ALL", 1)
	for name, src := range map[string]string{"lớp của giảng viên khác": c.third, "lớp không có": uuid.NewString()} {
		res := r.share(c.k.a.G, c.k.id, src, "documents")
		require.Equal(t, http.StatusForbidden, res.code, name)
		require.Equal(t, "source_course", res.details()["reason"], name)
	}
	// Từng là giảng viên của lớp nguồn nhưng đã bị mời ra: không còn quyền.
	r.enroll(uuid.MustParse(c.third), c.k.a.gv.ID, "TEACHER", "REMOVED")
	res := r.share(c.k.a.G, c.k.id, c.third, "documents")
	require.Equal(t, http.StatusForbidden, res.code)
	require.Equal(t, "source_course", res.details()["reason"])
	require.Equal(t, 0, r.count(`select count(*) from document_courses`))
}

func TestShareNotAvailableKinds(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	d := r.document(c.src, "LECTURE", "READY", true, "ALL", 1)
	for what, phase := range map[string]string{"questions": "P9", "grade_scheme": "P6"} {
		res := r.share(c.k.a.G, c.k.id, c.src, what)
		require.Equal(t, http.StatusUnprocessableEntity, res.code)
		require.Contains(t, string(res.body), "NOT_AVAILABLE")
		require.Contains(t, string(res.body), phase)
	}
	// Một loại không có ⇒ không ghi gì kể cả phần documents hợp lệ.
	require.Equal(t, http.StatusUnprocessableEntity, r.share(c.k.a.G, c.k.id, c.src, "documents", "questions").code)
	require.Equal(t, 0, r.count(`select count(*) from document_courses where document_id = $1`, d))
	require.Contains(t, string(r.share(c.k.a.G, c.k.id, c.src, "bogus").body), "INVALID_KIND")
	require.Equal(t, http.StatusUnprocessableEntity, r.share(c.k.a.G, c.k.id, c.src).code, "what rỗng")
	require.Contains(t, string(r.share(c.k.a.G, c.k.id, c.k.id, "documents").body), "SAME_COURSE")
}

func TestShareRBACAndArchive(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	for name, s := range map[string]session{"ADMIN": c.k.a.A, "TA": c.k.a.T, "sinh viên": c.k.a.S, "giảng viên lớp khác": c.k.a.G2} {
		require.Equal(t, http.StatusForbidden, r.share(s, c.k.id, c.src, "documents").code, name)
		require.Equal(t, http.StatusForbidden, r.get(s, "/courses/"+c.k.id+"/share-sources").code, name)
	}
	require.Equal(t, http.StatusUnauthorized, r.do(req{method: http.MethodGet, path: "/courses/" + c.k.id + "/share-sources"}).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.do(req{method: http.MethodPost, path: "/courses/" + c.k.id + "/share-from", bearer: c.k.a.G.access, body: map[string]any{"source_course_id": c.src, "what": []string{"documents"}}}).code, "thiếu Idempotency-Key")
	require.Equal(t, http.StatusOK, r.post(c.k.a.A, "/admin/courses/"+c.src+"/archive", nil, nil).code)
	res := r.share(c.k.a.G, c.k.id, c.src, "documents")
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_ARCHIVED", res.errCode())
}

func TestShareSourcesEligibleOnly(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	r.document(c.src, "LECTURE", "READY", true, "ALL", 1)
	r.document(c.src, "LECTURE", "READY", true, "ALL", 1)
	r.document(c.src, "COURSE_POLICY", "READY", true, "ALL", 1)
	r.document(c.src, "LECTURE", "FAILED", true, "ALL", 1)
	r.enroll(uuid.MustParse(c.otherSubject), c.k.a.gv2.ID, "TA", "ACTIVE") // không ảnh hưởng
	items := r.get(c.k.a.G, "/courses/"+c.k.id+"/share-sources").json()["items"].([]any)
	require.Len(t, items, 1, "chỉ lớp cùng học phần mà giảng viên cũng là TEACHER: không lấy lớp của người khác, lớp khác học phần, chính lớp đích")
	it := items[0].(map[string]any)
	require.Equal(t, c.src, it["id"])
	require.Equal(t, float64(2), it["documents"], "đếm tài liệu READY trừ quy chế")
	require.Equal(t, http.StatusOK, r.post(c.k.a.A, "/admin/courses/"+c.src+"/archive", nil, nil).code)
	require.Empty(t, r.get(c.k.a.G, "/courses/"+c.k.id+"/share-sources").json()["items"], "lớp đã lưu trữ không còn là nguồn")
}

func TestShareSourcesTeacherOnly(t *testing.T) {
	r := newRig(t)
	c := r.twoClasses()
	for name, s := range map[string]session{"TA": c.k.a.T, "ADMIN": c.k.a.A, "sinh viên": c.k.a.S} {
		res := r.get(s, "/courses/"+c.k.id+"/share-sources")
		require.Equal(t, http.StatusForbidden, res.code, name)
		require.NotContains(t, string(res.body), c.src, name)
	}
}
