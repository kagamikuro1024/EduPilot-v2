package library_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/library"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const (
	dim       = 1536
	jwtSecret = "0123456789abcdef0123456789abcdef"
)

type fx struct {
	t       *testing.T
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	blob    *blob.Store
	svc     *library.Service
	course  uuid.UUID
	teacher uuid.UUID
	ta      uuid.UUID
	sv      uuid.UUID
	srv     *httptest.Server
	iss     *auth.Issuer
}

func newFx(t *testing.T) *fx {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	dburl := testutil.MigratedPostgresURL(t)
	pool := testutil.RuntimePoolAt(t, dburl) // đúng cấu hình runtime
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	store, err := blob.New(ctx, blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "lib-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	f := &fx{t: t, pool: pool, rdb: rdb, blob: store, iss: auth.NewIssuer(jwtSecret, time.Hour, clock.Real{})}
	f.svc = &library.Service{Pool: pool, Blob: store}
	f.teacher = f.user("GV", "TEACHER")
	f.ta = f.user("TA", "TA")
	f.sv = f.user("SV", "STUDENT")
	f.course = f.newCourse("ACTIVE")
	f.enroll(f.course, f.teacher, "TEACHER", "ACTIVE")
	f.enroll(f.course, f.ta, "TA", "ACTIVE")
	f.enroll(f.course, f.sv, "STUDENT", "ACTIVE")
	env := map[string]string{
		"DATABASE_URL": dburl, "REDIS_URL": testutil.RedisURL(t), "JWT_SECRET_KEY": jwtSecret,
		"BLOB_ENDPOINT": testutil.MinIOEndpoint(t), "BLOB_BUCKET": "lib-" + strings.ToLower(testutil.TestPrefix(t)), "BLOB_ACCESS_KEY": testutil.MinIOAccessKey, "BLOB_SECRET_KEY": testutil.MinIOSecretKey,
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "BCRYPT_COST": "4",
		"RATE_LIMIT_IP_PER_MIN": "1000000", "RATE_LIMIT_USER_PER_MIN": "1000000", "REQUEST_TIMEOUT": "5s",
	}
	cfg, err := config.Load(func(k string) string { return env[k] }, config.Gateway)
	require.NoError(t, err)
	h := httpapi.NewRouter(httpapi.Deps{Cfg: cfg, Log: nil, DB: pool, Redis: rdb, Clock: clock.Real{}, State: httpapi.NewState(), Blob: store})
	f.srv = httptest.NewServer(h)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fx) user(name, role string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, uuid.NewString()+"@example.test", name, role).Scan(&id))
	return id
}

func (f *fx) newCourse(status string) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by, status, archived_at, join_enabled) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3, $4::course_status, case when $4::text = 'ARCHIVED' then now() end, $4::text = 'ACTIVE') returning id`,
		"LB"+id.String()[:8], string(jc), f.teacher, status).Scan(&c))
	return c
}

func (f *fx) enroll(course, user uuid.UUID, role, status string) {
	f.t.Helper()
	_, err := f.pool.Exec(f.t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at) values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'ADMIN', case when $4::text = 'REMOVED' then now() end)`, course, user, role, status)
	require.NoError(f.t, err)
}

func sum(s string) string { return fmt.Sprintf("%064x", []byte(s))[:64] }

func vecLit(axis int) string {
	p := make([]string, dim)
	for i := range p {
		p[i] = "0"
	}
	p[axis] = "1"
	return "[" + strings.Join(p, ",") + "]"
}

type docOpt struct {
	course  uuid.UUID
	title   string
	typ     string
	hidden  bool
	status  string
	file    string
	chunks  []string
	noEmbed bool
	cat     string
	week    int
	putBlob []byte
}

// doc chèn một tài liệu; mặc định READY, hiện với sinh viên, dùng cho AI, PDF, có blob thật nếu putBlob != nil.
func (f *fx) doc(o docOpt) uuid.UUID {
	f.t.Helper()
	if o.course == uuid.Nil {
		o.course = f.course
	}
	if o.typ == "" {
		o.typ = "LECTURE"
	}
	if o.status == "" {
		o.status = "READY"
	}
	if o.file == "" {
		o.file = o.title + ".pdf"
	}
	key := "courses/" + o.course.String() + "/documents/" + uuid.NewString() + "/" + url.PathEscape(o.file)
	if o.putBlob != nil {
		require.NoError(f.t, f.blob.Put(f.t.Context(), key, bytes.NewReader(o.putBlob), int64(len(o.putBlob)), "application/pdf"))
	}
	var cat *string
	if o.cat != "" {
		cat = &o.cat
	}
	var wk *int
	if o.week > 0 {
		wk = &o.week
	}
	var id uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into documents (course_id, title, type, filename, mime_type, sha256, blob_key, status, visible_to_students, use_for_rag, category, week_no, uploaded_by)
		values ($1, $2, $3::document_type, $4, 'application/pdf', $5, $6, $7::document_status, $8, true, $9, $10, $11) returning id`,
		o.course, o.title, o.typ, o.file, sum(uuid.NewString()), key, o.status, !o.hidden && o.typ != "ANSWER_KEY", cat, wk, f.teacher).Scan(&id))
	for i, text := range o.chunks {
		var emb any
		if !o.noEmbed {
			emb = vecLit(i % dim)
		}
		_, err := f.pool.Exec(f.t.Context(), `insert into content_chunks (document_id, course_ids, audience, ord, page_no, text, embedding) values ($1, array[$2::uuid], 'ALL', $3, $4, $5, $6::vector)`, id, o.course, i, i+1, text, emb)
		require.NoError(f.t, err)
	}
	return id
}

func (f *fx) list(q library.Filter) []library.Item {
	f.t.Helper()
	p, err := f.svc.List(f.t.Context(), f.course, q, httpx.PageParams{Limit: 50})
	require.NoError(f.t, err)
	return p.Items
}

func titles(items []library.Item) []string {
	var out []string
	for _, i := range items {
		out = append(out, i.Title)
	}
	return out
}

// TestLibrarySearchDiacriticInsensitive — AC9: tên / chủ đề không dấu, `q` dưới 2 ký tự bị bỏ qua (không 422).
func TestLibrarySearchDiacriticInsensitive(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.doc(docOpt{title: "Cảnh báo học vụ", cat: "Quy chế"})
	f.doc(docOpt{title: "Mật mã đối xứng"})
	require.Equal(t, []string{"Cảnh báo học vụ"}, titles(f.list(library.Filter{Q: "canh bao hoc vu"})))
	require.Equal(t, []string{"Cảnh báo học vụ"}, titles(f.list(library.Filter{Q: "QUY CHE"})), "chủ đề")
	require.Equal(t, []string{"Mật mã đối xứng"}, titles(f.list(library.Filter{Q: "mat ma"})))
	require.Len(t, f.list(library.Filter{Q: "a"}), 2, "dưới 2 ký tự: như không có q")
	require.Empty(t, f.list(library.Filter{Q: "100%"}))
}

// TestLibrarySearchByChunkText — AC9: khớp từ khoá trong nội dung đoạn (kể cả đoạn không nhúng), kèm đoạn trích.
func TestLibrarySearchByChunkText(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.doc(docOpt{title: "Slide 1", chunks: []string{"Giới thiệu chung.", "Thuật toán Dijkstra tìm đường đi ngắn nhất."}, noEmbed: true})
	f.doc(docOpt{title: "Slide 2", chunks: []string{"Nội dung khác hẳn."}})
	got := f.list(library.Filter{Q: "dijkstra duong di"})
	require.Equal(t, []string{"Slide 1"}, titles(got))
	require.Contains(t, got[0].Snippet, "Dijkstra")
	require.False(t, got[0].CanAskAI, "chưa nhúng đoạn nào")
}

// TestLibraryOnlyVisibleReady — AC9: chỉ READY + visible_to_students.
func TestLibraryOnlyVisibleReady(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.doc(docOpt{title: "Hiện"})
	f.doc(docOpt{title: "Ẩn", hidden: true})
	f.doc(docOpt{title: "Đang xử lý", status: "PROCESSING"})
	f.doc(docOpt{title: "Lỗi", status: "FAILED"})
	require.Equal(t, []string{"Hiện"}, titles(f.list(library.Filter{})))
}

// TestLibraryNoAnswerKey — AC9: không bao giờ ANSWER_KEY, kể cả khi ai đó cố tình bật cờ hiện (DB chặn) hay tìm đúng tên.
func TestLibraryNoAnswerKey(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	key := f.doc(docOpt{title: "Đáp án tuần 1", typ: "ANSWER_KEY", chunks: []string{"CANARY-7Q2X đáp án"}})
	f.doc(docOpt{title: "Bài giảng"})
	require.Equal(t, []string{"Bài giảng"}, titles(f.list(library.Filter{})))
	require.Empty(t, f.list(library.Filter{Q: "dap an"}))
	require.Empty(t, f.list(library.Filter{Q: "CANARY-7Q2X"}))
	_, err := f.svc.Get(t.Context(), f.course, key)
	require.Equal(t, "NOT_FOUND", codeOf(t, err))
	_, err = f.svc.Download(t.Context(), f.course, key)
	require.Equal(t, "NOT_FOUND", codeOf(t, err))
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "%v", err)
	return ae.Code
}

// TestLibraryOtherCourseHidden — AC9: tài liệu lớp khác ẩn; chia sẻ vào lớp thì hiện.
func TestLibraryOtherCourseHidden(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	other := f.newCourse("ACTIVE")
	foreign := f.doc(docOpt{course: other, title: "Của lớp khác", chunks: []string{"nội dung"}})
	shared := f.doc(docOpt{course: other, title: "Chia sẻ vào", chunks: []string{"nội dung"}})
	_, err := f.pool.Exec(t.Context(), `insert into document_courses (document_id, course_id) values ($1, $2)`, shared, f.course)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `update content_chunks set course_ids = course_ids || $2::uuid where document_id = $1`, shared, f.course)
	require.NoError(t, err)
	require.Equal(t, []string{"Chia sẻ vào"}, titles(f.list(library.Filter{})))
	_, err = f.svc.Get(t.Context(), f.course, foreign)
	require.Equal(t, "NOT_FOUND", codeOf(t, err))
	_, err = f.svc.Get(t.Context(), f.course, shared)
	require.NoError(t, err)
}

// TestLibraryCursor — AC9 / luật 13: con trỏ; lọc tuần.
func TestLibraryCursor(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for i := range 5 {
		f.doc(docOpt{title: fmt.Sprintf("Tài liệu %d", i), week: 1 + i%2})
	}
	seen := map[uuid.UUID]bool{}
	var cur *httpx.Cursor
	for range 5 {
		p, err := f.svc.List(t.Context(), f.course, library.Filter{}, httpx.PageParams{Limit: 2, Cursor: cur})
		require.NoError(t, err)
		for _, it := range p.Items {
			require.False(t, seen[it.ID])
			seen[it.ID] = true
		}
		if p.NextCursor == nil {
			break
		}
		c, err := httpx.ParseCursor(*p.NextCursor)
		require.NoError(t, err)
		cur = &c
	}
	require.Len(t, seen, 5)
	w := 1
	require.Len(t, f.list(library.Filter{Week: &w}), 3)
}

// TestPreviewURLInline — AC10: PDF có preview_url ký sẵn, Content-Disposition inline; tệp mất → không có preview, vẫn đọc được thông tin.
func TestPreviewURLInline(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(docOpt{title: "Bài", putBlob: []byte("%PDF-1.4 x")})
	d, err := f.svc.Get(t.Context(), f.course, id)
	require.NoError(t, err)
	require.NotNil(t, d.PreviewURL)
	u, err := url.Parse(*d.PreviewURL)
	require.NoError(t, err)
	require.Equal(t, "inline", u.Query().Get("response-content-disposition"))
	res, err := http.Get(*d.PreviewURL) //nolint:gosec,noctx // URL ký sẵn tới MinIO của test
	require.NoError(t, err)
	_ = res.Body.Close()
	require.Equal(t, 200, res.StatusCode)
	require.Contains(t, res.Header.Get("Content-Disposition"), "inline")
	gone := f.doc(docOpt{title: "Mất tệp"})
	d, err = f.svc.Get(t.Context(), f.course, gone)
	require.NoError(t, err)
	require.Nil(t, d.PreviewURL)
}

// TestDownloadCountAtomic — AC10: 20 yêu cầu song song → +20; URL là attachment.
func TestDownloadCountAtomic(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(docOpt{title: "Bài", putBlob: []byte("%PDF-1.4 x")})
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := f.svc.Download(context.Background(), f.course, id)
			if err != nil || !strings.Contains(u, "attachment") {
				t.Errorf("download: %v %s", err, u)
			}
		}()
	}
	wg.Wait()
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select download_count from documents where id=$1`, id).Scan(&n))
	require.Equal(t, 20, n)
}

// TestDownloadNotAllowed404 — AC10: tài liệu ẩn / chưa sẵn sàng / lớp khác → 404 (không lộ tồn tại), không đếm.
func TestDownloadNotAllowed404(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	hidden := f.doc(docOpt{title: "Ẩn", hidden: true, putBlob: []byte("%PDF-1.4 x")})
	proc := f.doc(docOpt{title: "Chưa xong", status: "PROCESSING", putBlob: []byte("%PDF-1.4 y")})
	for _, id := range []uuid.UUID{hidden, proc, uuid.New()} {
		_, err := f.svc.Download(t.Context(), f.course, id)
		require.Equal(t, "NOT_FOUND", codeOf(t, err))
	}
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select coalesce(sum(download_count), 0) from documents`).Scan(&n))
	require.Zero(t, n)
}

// TestDownloadFileGone — AC10: object đã mất → 404 FILE_GONE, không đếm.
func TestDownloadFileGone(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(docOpt{title: "Mất tệp"})
	_, err := f.svc.Download(t.Context(), f.course, id)
	require.Equal(t, apierr.FileGone, codeOf(t, err))
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select download_count from documents where id=$1`, id).Scan(&n))
	require.Zero(t, n)
}

// --- qua router thật ---------------------------------------------------------------------------------------------------------------------------

func (f *fx) do(user uuid.UUID, role auth.Role, method, path string, hdr ...string) (*http.Response, []byte) {
	f.t.Helper()
	req, err := http.NewRequestWithContext(f.t.Context(), method, f.srv.URL+"/api/v1"+path, strings.NewReader(""))
	require.NoError(f.t, err)
	if user != uuid.Nil {
		tok, err := f.iss.Issue(user.String(), role, "x@example.test")
		require.NoError(f.t, err)
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(f.t, err)
	b, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	return res, b
}

// TestLibraryETag — AC10: ETag ổn định khi không đổi; If-None-Match → 304.
func TestLibraryETag(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.doc(docOpt{title: "Bài", chunks: []string{"nội dung"}})
	path := "/courses/" + f.course.String() + "/library"
	res, _ := f.do(f.sv, auth.RoleStudent, http.MethodGet, path)
	require.Equal(t, 200, res.StatusCode)
	tag := res.Header.Get("ETag")
	require.NotEmpty(t, tag)
	res2, _ := f.do(f.sv, auth.RoleStudent, http.MethodGet, path)
	require.Equal(t, tag, res2.Header.Get("ETag"))
	res3, body := f.do(f.sv, auth.RoleStudent, http.MethodGet, path, "If-None-Match", tag)
	require.Equal(t, 304, res3.StatusCode)
	require.Empty(t, body)
}

// TestLibraryETagChangesOnChunkEdit — AC10, #12: sửa đoạn đổi `updated_at` của tài liệu nên ETag của danh sách đổi.
func TestLibraryETagChangesOnChunkEdit(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(docOpt{title: "Bài", chunks: []string{"nội dung"}})
	path := "/courses/" + f.course.String() + "/library"
	res, _ := f.do(f.sv, auth.RoleStudent, http.MethodGet, path)
	before := res.Header.Get("ETag")
	time.Sleep(20 * time.Millisecond)
	_, err := f.pool.Exec(t.Context(), `update documents set updated_at = now(), version = version + 1 where id=$1`, id) // đúng việc document.EditChunk làm (DocTouch)
	require.NoError(t, err)
	res, _ = f.do(f.sv, auth.RoleStudent, http.MethodGet, path)
	require.NotEqual(t, before, res.Header.Get("ETag"))
}

// TestLibraryMatrix — AC14: chỉ STUDENT ACTIVE của lớp; TA / TEACHER / ADMIN / ngoài lớp / PENDING / REMOVED → 403; chưa đăng nhập 401.
func TestLibraryMatrix(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.doc(docOpt{title: "Bài", putBlob: []byte("%PDF-1.4 x")})
	pending, removed, outsider, admin := f.user("P", "STUDENT"), f.user("R", "STUDENT"), f.user("O", "STUDENT"), f.user("A", "ADMIN")
	f.enroll(f.course, pending, "STUDENT", "PENDING")
	f.enroll(f.course, removed, "STUDENT", "REMOVED")
	base := "/courses/" + f.course.String() + "/library"
	for _, c := range []struct {
		name string
		u    uuid.UUID
		r    auth.Role
		want int
	}{{"student", f.sv, auth.RoleStudent, 200}, {"ta", f.ta, auth.RoleTA, 403}, {"teacher", f.teacher, auth.RoleTeacher, 403}, {"admin", admin, auth.RoleAdmin, 403},
		{"outsider", outsider, auth.RoleStudent, 403}, {"pending", pending, auth.RoleStudent, 403}, {"removed", removed, auth.RoleStudent, 403}, {"anonymous", uuid.Nil, "", 401}} {
		for _, p := range []string{base, base + "/" + id.String(), base + "/" + id.String() + "/download"} {
			res, _ := f.do(c.u, c.r, http.MethodGet, p)
			require.Equal(t, c.want, res.StatusCode, c.name+" "+p)
		}
	}
	_, body := f.do(f.sv, auth.RoleStudent, http.MethodGet, base+"/"+id.String()+"/download")
	var out struct{ URL string }
	require.NoError(t, json.Unmarshal(body, &out))
	require.NotEmpty(t, out.URL)
}

// --- tool search_library -------------------------------------------------------------------------------------------------------------------

func (f *fx) tc() agent.TrustedContext {
	return agent.TrustedContext{UserID: f.sv, CourseID: f.course, Role: "STUDENT", TraceID: "t"}
}

// TestSearchLibraryTool — AC12: tối đa 5 tài liệu {document_id, title, type, week_no, page_no?, snippet, href}; chỉ tài liệu sinh viên được thấy.
func TestSearchLibraryTool(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for i := range 7 {
		f.doc(docOpt{title: fmt.Sprintf("Bài giảng mã hóa %d", i), week: 3, chunks: []string{"Giới thiệu.", "AES là mã hóa đối xứng khối."}})
	}
	f.doc(docOpt{title: "Mã hóa ẩn", hidden: true})
	facts, srcs, ok, err := f.svc.Search(t.Context(), f.tc(), "mã hóa")
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, srcs, 5)
	rs := facts["results"].([]map[string]any)
	require.Len(t, rs, 5)
	for _, r := range rs {
		require.Equal(t, "/library/"+r["document_id"].(uuid.UUID).String(), r["href"])
		require.NotContains(t, r["title"], "ẩn")
		for _, k := range []string{"document_id", "title", "type", "week_no", "snippet", "href"} {
			require.Contains(t, r, k)
		}
	}
	_, _, ok, err = f.svc.Search(t.Context(), f.tc(), "zzzzzz")
	require.NoError(t, err)
	require.False(t, ok)
	_, _, ok, _ = f.svc.Search(t.Context(), f.tc(), "a")
	require.False(t, ok, "q ngắn")
}

// TestSearchLibraryNoIdentityParam — AC12: tham số của tool chỉ có `query` (không user_id / student_code / course_id); lớp lấy từ TrustedContext.
func TestSearchLibraryNoIdentityParam(t *testing.T) {
	t.Parallel()
	tool := agent.NewLibraryTool(nil)
	typ := tool.ArgsType()
	require.Equal(t, 1, typ.NumField())
	require.Equal(t, "query", typ.Field(0).Tag.Get("json"))
}

// TestSearchLibraryExcludesAnswerKey — AC12: tool không bao giờ trả ANSWER_KEY (kể cả khớp nội dung đoạn), tài liệu ẩn, tài liệu lớp khác.
func TestSearchLibraryExcludesAnswerKey(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.doc(docOpt{title: "Đáp án đề thi", typ: "ANSWER_KEY", chunks: []string{"CANARY-7Q2X đáp án đề thi"}})
	other := f.newCourse("ACTIVE")
	f.doc(docOpt{course: other, title: "Đáp án đề thi lớp khác", chunks: []string{"đáp án đề thi"}})
	f.doc(docOpt{title: "Hướng dẫn ôn tập", chunks: []string{"Ôn tập cho kỳ thi"}})
	for _, q := range []string{"đáp án đề thi", "CANARY-7Q2X", "dap an"} {
		_, srcs, ok, err := f.svc.Search(t.Context(), f.tc(), q)
		require.NoError(t, err)
		require.False(t, ok, q)
		require.Empty(t, srcs)
	}
}
