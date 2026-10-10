package document_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

type fx struct {
	pool     *pgxpool.Pool
	rdb      *appredis.Client
	blob     *blob.Store
	svc      *document.Service
	teacher  uuid.UUID
	ta       uuid.UUID
	course   uuid.UUID
	archived uuid.UUID
}

func newFx(t *testing.T) *fx {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	store, err := blob.New(ctx, blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "doc-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	f := &fx{pool: pool, rdb: rdb, blob: store}
	f.svc = &document.Service{Pool: pool, Redis: rdb, Blob: store, Jobs: jobs.NewService(pool), Clock: clock.Real{}, PresignPerMin: 1000}
	f.teacher = f.user(t, "GV", "TEACHER")
	f.ta = f.user(t, "TA", "TA")
	f.course = f.newCourse(t, f.teacher, "ACTIVE")
	f.archived = f.newCourse(t, f.teacher, "ARCHIVED")
	return f
}

func (f *fx) user(t *testing.T, name, role string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, uuid.NewString()+"@example.test", name, role).Scan(&id))
	return id
}

func (f *fx) newCourse(t *testing.T, owner uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	var c uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by, status, archived_at, join_enabled) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3, $4::course_status, case when $4::text = 'ARCHIVED' then now() end, $4::text = 'ACTIVE') returning id`,
		"DC"+id.String()[:8], string(jc), owner, status).Scan(&c))
	_, err := f.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, c, owner)
	require.NoError(t, err)
	return c
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

const pdfMime = "application/pdf"

func pdf() []byte { return []byte("%PDF-1.4 " + uuid.NewString()) }

func in(name, mime string, b []byte) document.PresignIn {
	return document.PresignIn{Purpose: "document", Filename: name, MimeType: mime, SizeBytes: int64(len(b)), SHA256: sum(b)}
}

func (f *fx) put(t *testing.T, u string, mime string, b []byte) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, u, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", mime)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(t, resp.Body.Close())
	return resp.StatusCode
}

// upload: presign + PUT (nếu put) và trả upload_id.
func (f *fx) upload(t *testing.T, actor, course uuid.UUID, name, mime string, b []byte, put bool) uuid.UUID {
	t.Helper()
	out, err := f.svc.Presign(t.Context(), actor, course, in(name, mime, b))
	require.NoError(t, err)
	if put {
		require.Equal(t, http.StatusOK, f.put(t, out.URL, mime, b))
	}
	return out.UploadID
}

func apiCode(t *testing.T, err error) (int, string) {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "cần *apierr.Error, nhận %v", err)
	return ae.Status, ae.Code
}

func completeIn(id uuid.UUID, typ string) document.CompleteIn {
	return document.CompleteIn{UploadID: id, Title: "Bài giảng tuần 1", Type: typ}
}

// TestPresignValidation — AC1.
func TestPresignValidation(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	b := pdf()
	good := in("bai.pdf", pdfMime, b)
	ok, err := f.svc.Presign(t.Context(), f.teacher, f.course, good)
	require.NoError(t, err)
	require.Equal(t, "PUT", ok.Method)
	require.Equal(t, 600, ok.ExpiresIn)
	require.Equal(t, pdfMime, ok.Headers["Content-Type"])
	require.Equal(t, strconv.Itoa(len(b)), ok.Headers["Content-Length"]) // = size_bytes
	require.True(t, strings.HasPrefix(ok.BlobKey, "courses/"+f.course.String()+"/documents/"+ok.UploadID.String()+"/"))

	mut := func(fn func(*document.PresignIn)) document.PresignIn { c := good; fn(&c); return c }
	for name, c := range map[string]struct {
		in     document.PresignIn
		status int
		code   string
	}{
		"đuôi lạ":              {in("x.exe", "application/octet-stream", b), 422, apierr.FileTypeNotAllowed},
		"mime không khớp đuôi": {mut(func(p *document.PresignIn) { p.MimeType = "text/plain" }), 422, apierr.FileTypeNotAllowed},
		"không đuôi":           {mut(func(p *document.PresignIn) { p.Filename = "baigiang" }), 422, apierr.FileTypeNotAllowed},
		"quá lớn":              {mut(func(p *document.PresignIn) { p.SizeBytes = 50<<20 + 1 }), 422, apierr.FileTooLarge},
		"rỗng":                 {mut(func(p *document.PresignIn) { p.SizeBytes = 0 }), 422, apierr.FileTooLarge},
		"âm":                   {mut(func(p *document.PresignIn) { p.SizeBytes = -5 }), 422, apierr.FileTooLarge},
		"băm sai":              {mut(func(p *document.PresignIn) { p.SHA256 = "abc" }), 422, apierr.ValidationFailed},
		"tên có /":             {mut(func(p *document.PresignIn) { p.Filename = "a/b.pdf" }), 422, apierr.ValidationFailed},
		"tên có \\":            {mut(func(p *document.PresignIn) { p.Filename = `a\b.pdf` }), 422, apierr.ValidationFailed},
		"tên có ..":            {mut(func(p *document.PresignIn) { p.Filename = "a..b.pdf" }), 422, apierr.ValidationFailed},
		"tên dài":              {mut(func(p *document.PresignIn) { p.Filename = strings.Repeat("a", 252) + ".pdf" }), 422, apierr.ValidationFailed},
		"mục đích lạ":          {mut(func(p *document.PresignIn) { p.Purpose = "avatar" }), 422, apierr.ValidationFailed},
	} {
		_, err := f.svc.Presign(t.Context(), f.teacher, f.course, c.in)
		st, code := apiCode(t, err)
		require.Equal(t, c.status, st, name)
		require.Equal(t, c.code, code, name)
	}
	// 3 đuôi hợp lệ, kể cả HOA
	for _, c := range [][2]string{{"a.PDF", pdfMime}, {"a.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}, {"a.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"}} {
		_, err := f.svc.Presign(t.Context(), f.teacher, f.course, in(c[0], c[1], b))
		require.NoError(t, err, c[0])
	}
}

// TestPresignExpiry — AC1: URL ký hết hạn 10 phút, ký kèm Content-Length (PUT sai kích thước bị MinIO từ chối).
func TestPresignExpiry(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	b := pdf()
	out, err := f.svc.Presign(t.Context(), f.teacher, f.course, in("bai.pdf", pdfMime, b))
	require.NoError(t, err)
	u, err := url.Parse(out.URL)
	require.NoError(t, err)
	require.Equal(t, "600", u.Query().Get("X-Amz-Expires"))
	require.Contains(t, u.Query().Get("X-Amz-SignedHeaders"), "content-length")
	require.GreaterOrEqual(t, f.put(t, out.URL, pdfMime, append(b, "thừa"...)), 400)
	n, err := f.rdb.TTL(t.Context(), appredis.Key("upload", out.UploadID.String())).Result()
	require.NoError(t, err)
	require.True(t, n > 14*time.Minute && n <= 15*time.Minute, n)
}

func TestPresignRateLimit(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.svc.PresignPerMin = 10
	b := pdf()
	var got429 bool
	for i := range 11 {
		_, err := f.svc.Presign(t.Context(), f.teacher, f.course, in("bai.pdf", pdfMime, b))
		if i < 10 {
			if err != nil { // rơi sang phút kế tiếp giữa chừng thì bộ đếm đã đổi: chấp nhận chạy lại
				t.Skip("qua ranh giới phút giữa lúc đếm")
			}
			continue
		}
		st, code := apiCode(t, err)
		require.Equal(t, 429, st)
		require.Equal(t, apierr.RateLimited, code)
		got429 = true
	}
	require.True(t, got429)
	other := f.user(t, "GV2", "TEACHER") // hạn mức theo người, không theo lớp
	_, err := f.svc.Presign(t.Context(), other, f.course, in("bai.pdf", pdfMime, b))
	require.NoError(t, err)
}

func TestPresignArchivedCourse(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, err := f.svc.Presign(t.Context(), f.teacher, f.archived, in("bai.pdf", pdfMime, pdf()))
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, apierr.CourseArchived, code)
}

// TestCompleteCreatesDocAndJob — AC2: tài liệu QUEUED + việc document.ingest trong MỘT giao dịch; cờ mặc định theo loại.
func TestCompleteCreatesDocAndJob(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for _, c := range []struct {
		typ         string
		visible, rg bool
	}{{"LECTURE", true, true}, {"COURSE_POLICY", true, true}, {"EXAM_PAPER", true, false}, {"ANSWER_KEY", false, true}, {"OTHER", true, true}} {
		b := pdf()
		id := f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, true)
		out, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(id, c.typ))
		require.NoError(t, err, c.typ)
		require.Equal(t, "QUEUED", out.Document.Status)
		require.Equal(t, c.visible, out.Document.VisibleToStudents, c.typ)
		require.Equal(t, c.rg, out.Document.UseForRAG, c.typ)
		require.Equal(t, f.course, out.Document.CourseID)
		var kind, st string
		require.NoError(t, f.pool.QueryRow(t.Context(), `select kind, status::text from jobs where id=$1`, out.JobID).Scan(&kind, &st))
		require.Equal(t, "document.ingest", kind)
		require.Equal(t, "QUEUED", st)
		var n int
		require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from outbox where topic='job.enqueue' and payload->>'job_id'=$1`, out.JobID.String()).Scan(&n))
		require.Equal(t, 1, n, "việc đi kèm một dòng outbox cùng giao dịch")
		var key string
		require.NoError(t, f.pool.QueryRow(t.Context(), `select blob_key from documents where id=$1`, out.Document.ID).Scan(&key))
		_, err = f.blob.Stat(t.Context(), key)
		require.NoError(t, err, "tệp còn nguyên ở kho")
	}
	// cờ do người gọi đặt ghi đè mặc định; đáp án không được hiện cho sinh viên
	id := f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, pdf(), true)
	no := false
	out, err := f.svc.Complete(t.Context(), f.teacher, f.course, document.CompleteIn{UploadID: id, Title: "x", Type: "LECTURE", UseForRAG: &no, VisibleToStudents: &no})
	require.NoError(t, err)
	require.False(t, out.Document.UseForRAG)
	require.False(t, out.Document.VisibleToStudents)
	yes := true
	id = f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, pdf(), true)
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, document.CompleteIn{UploadID: id, Title: "x", Type: "ANSWER_KEY", VisibleToStudents: &yes})
	st, _ := apiCode(t, err)
	require.Equal(t, 422, st)
	// vé đã dùng: lần hai → 404
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(uuid.New(), "LECTURE"))
	st, code := apiCode(t, err)
	require.Equal(t, 404, st)
	require.Equal(t, apierr.UploadNotFound, code)
}

// TestCompleteRejectsFakePDF / MagicBytesMismatch / SizeMismatch — AC3: không tạo documents / jobs, xoá object.
func TestCompleteRejectsFakePDF(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	fake := []byte("MZ\x90\x00 đây là chương trình chạy được, không phải PDF")
	out, err := f.svc.Presign(t.Context(), f.teacher, f.course, in("bai.pdf", pdfMime, fake))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, f.put(t, out.URL, pdfMime, fake))
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(out.UploadID, "LECTURE"))
	st, code := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, apierr.FileTypeMismatch, code)
	_, err = f.blob.Stat(t.Context(), out.BlobKey)
	require.ErrorIs(t, err, blob.ErrNotFound, "object bị xoá")
	f.expectNoDocs(t)
}

func TestCompleteMagicBytesMismatch(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for _, c := range []struct {
		name, mime string
		body       []byte
		ok         bool
	}{
		{"a.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("%PDF-1.4 là PDF đổi đuôi"), false},
		{"a.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation", []byte("PK\x03\x04 zip " + uuid.NewString()), true},
		{"a.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("PK\x03\x04 zip " + uuid.NewString()), true},
	} {
		id := f.upload(t, f.teacher, f.course, c.name, c.mime, c.body, true)
		_, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(id, "LECTURE"))
		if c.ok {
			require.NoError(t, err, c.name)
			continue
		}
		_, code := apiCode(t, err)
		require.Equal(t, apierr.FileTypeMismatch, code, c.name)
	}
}

func TestCompleteSizeMismatch(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	// khai 100 byte nhưng đưa lên đúng 100 byte rồi đổi vé? Dùng Stat: tạo vé với size A, ghi object kích thước B bằng đường khác (Put trực tiếp).
	b := pdf()
	out, err := f.svc.Presign(t.Context(), f.teacher, f.course, in("bai.pdf", pdfMime, b))
	require.NoError(t, err)
	require.NoError(t, f.blob.Put(t.Context(), out.BlobKey, bytes.NewReader(append(b, "dài hơn khai báo"...)), int64(len(b)+len("dài hơn khai báo")), pdfMime))
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(out.UploadID, "LECTURE"))
	st, code := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, apierr.UploadIncomplete, code)
	_, err = f.blob.Stat(t.Context(), out.BlobKey)
	require.ErrorIs(t, err, blob.ErrNotFound)
	f.expectNoDocs(t)
}

func TestCompleteUploadNotFound(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	b := pdf()
	id := f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, false) // chưa PUT
	_, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(id, "LECTURE"))
	st, code := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, apierr.UploadIncomplete, code)
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(uuid.New(), "LECTURE"))
	st, code = apiCode(t, err)
	require.Equal(t, 404, st)
	require.Equal(t, apierr.UploadNotFound, code)
	require.NoError(t, f.rdb.Del(t.Context(), appredis.Key("upload", id.String())).Err()) // vé hết hạn
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(id, "LECTURE"))
	_, code = apiCode(t, err)
	require.Equal(t, apierr.UploadNotFound, code)
	f.expectNoDocs(t)
}

func TestCompleteOtherUsersUpload404(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	id := f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, pdf(), true)
	_, err := f.svc.Complete(t.Context(), f.ta, f.course, completeIn(id, "LECTURE"))
	st, code := apiCode(t, err)
	require.Equal(t, 404, st, "vé của người khác: không lộ sự tồn tại")
	require.Equal(t, apierr.UploadNotFound, code)
	other := f.newCourse(t, f.teacher, "ACTIVE")
	_, err = f.svc.Complete(t.Context(), f.teacher, other, completeIn(id, "LECTURE"))
	_, code = apiCode(t, err)
	require.Equal(t, apierr.UploadNotFound, code, "vé của lớp khác")
	f.expectNoDocs(t)
}

func (f *fx) expectNoDocs(t *testing.T) {
	t.Helper()
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select (select count(*) from documents) + (select count(*) from jobs)`).Scan(&n))
	require.Zero(t, n, "không tạo documents / jobs")
}

// TestDuplicateSameCourse / ParallelDifferentKeys / ElsewhereHint / NoLeakOtherTeacher — AC4.
func TestDuplicateSameCourse(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	b := pdf()
	first, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, true), "LECTURE"))
	require.NoError(t, err)
	id2, err := f.svc.Presign(t.Context(), f.teacher, f.course, in("khac-ten.pdf", pdfMime, b))
	require.NoError(t, err)
	require.Equal(t, 200, f.put(t, id2.URL, pdfMime, b))
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(id2.UploadID, "LECTURE"))
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, 409, ae.Status)
	require.Equal(t, apierr.DocumentDuplicate, ae.Code)
	require.Equal(t, map[string]string{"existing_id": first.Document.ID.String()}, ae.Details)
	_, err = f.blob.Stat(t.Context(), id2.BlobKey)
	require.ErrorIs(t, err, blob.ErrNotFound, "object mới bị xoá")
}

func TestDuplicateParallelDifferentKeys(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	b := pdf()
	var ids [4]uuid.UUID
	for i := range ids {
		ids[i] = f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, true)
	}
	var wg sync.WaitGroup
	errs := make([]error, len(ids))
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = f.svc.Complete(context.Background(), f.teacher, f.course, completeIn(ids[i], "LECTURE"))
		}()
	}
	wg.Wait()
	ok := 0
	for _, e := range errs {
		if e == nil {
			ok++
			continue
		}
		_, code := apiCode(t, e)
		require.Equal(t, apierr.DocumentDuplicate, code)
	}
	require.Equal(t, 1, ok, "đúng một tài liệu")
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from documents where course_id=$1`, f.course).Scan(&n))
	require.Equal(t, 1, n)
}

func TestDuplicateElsewhereHint(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	other := f.newCourse(t, f.teacher, "ACTIVE")
	b := pdf()
	first, err := f.svc.Complete(t.Context(), f.teacher, other, completeIn(f.upload(t, f.teacher, other, "bai.pdf", pdfMime, b, true), "LECTURE"))
	require.NoError(t, err)
	_, err = f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, true), "LECTURE"))
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, 409, ae.Status)
	require.Equal(t, apierr.DocumentDuplicateElsewhere, ae.Code)
	d, _ := ae.Details.(map[string]string)
	require.Equal(t, first.Document.ID.String(), d["document_id"])
	require.NotEmpty(t, d["class_code"])
}

func TestDuplicateNoLeakOtherTeacher(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	stranger := f.user(t, "GV khác", "TEACHER")
	theirs := f.newCourse(t, stranger, "ACTIVE")
	b := pdf()
	_, err := f.svc.Complete(t.Context(), stranger, theirs, completeIn(f.upload(t, stranger, theirs, "bai.pdf", pdfMime, b, true), "LECTURE"))
	require.NoError(t, err)
	// lớp của người gọi không có tệp này, người gọi không thuộc lớp kia: coi như không trùng
	out, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, b, true), "LECTURE"))
	require.NoError(t, err)
	require.Equal(t, "QUEUED", out.Document.Status)
}

// TestRetry / TestReindex (nhánh lỗi 409) — US-P8-01 AC15/AC16.
func TestRetryAndReindexStates(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	out, err := f.svc.Complete(t.Context(), f.teacher, f.course, completeIn(f.upload(t, f.teacher, f.course, "bai.pdf", pdfMime, pdf(), true), "LECTURE"))
	require.NoError(t, err)
	id := out.Document.ID
	_, err = f.svc.Retry(t.Context(), f.teacher, f.course, id) // QUEUED
	_, code := apiCode(t, err)
	require.Equal(t, apierr.DocumentNotFailed, code)
	_, err = f.svc.Reindex(t.Context(), f.teacher, f.course, id) // chưa READY
	_, code = apiCode(t, err)
	require.Equal(t, apierr.DocumentNotReady, code)
	_, err = f.pool.Exec(t.Context(), `update documents set status='FAILED', error='x' where id=$1`, id)
	require.NoError(t, err)
	r, err := f.svc.Retry(t.Context(), f.teacher, f.course, id)
	require.NoError(t, err)
	var st string
	var e *string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select status::text, error from documents where id=$1`, id).Scan(&st, &e))
	require.Equal(t, "QUEUED", st)
	require.Nil(t, e)
	var kind string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select kind from jobs where id=$1`, r.JobID).Scan(&kind))
	require.Equal(t, "document.ingest", kind)
	_, err = f.svc.Retry(t.Context(), f.teacher, f.course, uuid.New())
	st2, _ := apiCode(t, err)
	require.Equal(t, 404, st2)
	_, err = f.svc.Retry(t.Context(), f.teacher, f.archived, id)
	_, code = apiCode(t, err)
	require.Equal(t, apierr.CourseArchived, code)

	// tài liệu chia sẻ từ lớp khác: chỉ lớp gốc được lập chỉ mục lại
	other := f.newCourse(t, f.teacher, "ACTIVE")
	_, err = f.pool.Exec(t.Context(), `update documents set status='READY' where id=$1`, id)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `insert into document_courses (document_id, course_id, shared_by) values ($1, $2, $3)`, id, other, f.teacher)
	require.NoError(t, err)
	_, err = f.svc.Reindex(t.Context(), f.teacher, other, id)
	_, code = apiCode(t, err)
	require.Equal(t, apierr.DocumentSharedReadonly, code)
	ro, err := f.svc.Reindex(t.Context(), f.teacher, f.course, id)
	require.NoError(t, err)
	require.NoError(t, f.pool.QueryRow(t.Context(), `select kind from jobs where id=$1`, ro.JobID).Scan(&kind))
	require.Equal(t, "document.reindex", kind)
	all, err := f.svc.ReindexAll(t.Context(), f.teacher, f.course)
	require.NoError(t, err)
	require.NoError(t, f.pool.QueryRow(t.Context(), `select kind from jobs where id=$1`, all.JobID).Scan(&kind))
	require.Equal(t, "document.reindex_all", kind)
}
