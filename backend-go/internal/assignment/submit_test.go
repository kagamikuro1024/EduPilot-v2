package assignment_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/submission"
	"github.com/edupilot/backend-go/internal/testutil"
)

// subEnv: env + Redis + MinIO + đồng hồ giả cho InAppSource.
type subEnv struct {
	*env
	clk  *clock.Fake
	blob *blob.Store
	sub  *submission.Service
	a    assignment.Assignment // bài đã công bố, hạn +72 giờ, kênh IN_APP
}

func newSubEnv(t *testing.T) *subEnv {
	t.Helper()
	e := newEnv(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	st, err := blob.New(t.Context(), blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "sub-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	clk := clock.NewFake(time.Now().UTC())
	se := &subEnv{env: e, clk: clk, blob: st}
	se.sub = &submission.Service{Pool: e.pool, Redis: rdb, Blob: st, Clock: clk, Log: e.svc.Log}
	i := in("BT03")
	i.DueAt = clk.Now().Add(72 * time.Hour)
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(t, err)
	e.publish(a)
	se.a = a
	return se
}

func (e *subEnv) sv() uuid.UUID { return e.students[0] }

func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func (e *subEnv) presign(student uuid.UUID, a uuid.UUID, name string, data []byte) (submission.PresignOut, error) {
	return e.sub.Presign(e.ctx, student, e.course, a, submission.PresignIn{Filename: name, SizeBytes: int64(len(data)), SHA256: sha(data)})
}

func (e *subEnv) put(p submission.PresignOut, data []byte) int {
	e.t.Helper()
	req, err := http.NewRequestWithContext(e.ctx, http.MethodPut, p.URL, bytes.NewReader(data))
	require.NoError(e.t, err)
	for k, v := range p.Headers {
		if k != "Content-Length" {
			req.Header.Set(k, v)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(e.t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	require.NoError(e.t, resp.Body.Close())
	return resp.StatusCode
}

// submit: presign → PUT → complete.
func (e *subEnv) submit(student uuid.UUID, a uuid.UUID, name string, data []byte) (submission.Submitted, error) {
	e.t.Helper()
	p, err := e.presign(student, a, name, data)
	require.NoError(e.t, err)
	require.Equal(e.t, http.StatusOK, e.put(p, data))
	return e.sub.Complete(e.ctx, student, e.course, a, p.UploadID, "")
}

var txt = []byte("Bài làm của tôi.\n")

func TestSubmitPresignValidation(t *testing.T) {
	e := newSubEnv(t)
	cases := map[string]struct {
		name string
		size int64
		hash string
		want string
	}{
		"đuôi lạ":    {"bai.exe", 5, sha(txt), "FILE_TYPE_NOT_ALLOWED"},
		"không đuôi": {"bai", 5, sha(txt), "FILE_TYPE_NOT_ALLOWED"},
		"rỗng":       {"bai.txt", 0, sha(txt), "FILE_EMPTY"},
		"quá lớn":    {"bai.pdf", 20<<20 + 1, sha(txt), "FILE_TOO_LARGE"},
		"băm sai":    {"bai.txt", 5, "xyz", "VALIDATION_FAILED"},
		"tên rỗng":   {"///", 5, sha(txt), "FILE_TYPE_NOT_ALLOWED"},
	}
	for n, c := range cases {
		_, err := e.sub.Presign(e.ctx, e.sv(), e.course, e.a.ID, submission.PresignIn{Filename: c.name, SizeBytes: c.size, SHA256: c.hash})
		require.Equal(t, c.want, code(err), n)
	}
	require.Equal(t, "._etc_passwd.txt", submission.SafeName("../etc/passwd.txt"), "gạch chéo thành _, .. rút gọn: không còn đường dẫn lên thư mục cha")
	require.Equal(t, "a_b.txt", submission.SafeName("a\\b.txt"))
	require.Equal(t, "a.txt", submission.SafeName("a..txt"))
	long := strings.Repeat("a", 300) + ".pdf"
	require.Len(t, []rune(submission.SafeName(long)), 255)
	require.True(t, strings.HasSuffix(submission.SafeName(long), ".pdf"))
}

func TestSubmitContentTypeFromExt(t *testing.T) {
	e := newSubEnv(t)
	for name, want := range map[string]string{"a.pdf": "application/pdf", "a.docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "a.txt": "text/plain; charset=utf-8", "a.MD": "text/plain; charset=utf-8"} {
		p, err := e.presign(e.sv(), e.a.ID, name, txt)
		require.NoError(t, err, name)
		require.Equal(t, want, p.Headers["Content-Type"], name)
	}
}

func TestPresignChecksumSigned(t *testing.T) {
	e := newSubEnv(t)
	p, err := e.presign(e.sv(), e.a.ID, "a.txt", txt)
	require.NoError(t, err)
	u, err := url.Parse(p.URL)
	require.NoError(t, err)
	signed := u.Query().Get("X-Amz-SignedHeaders")
	require.Contains(t, signed, "x-amz-checksum-sha256")
	require.Contains(t, signed, "content-length")
	want, _ := blob.ChecksumHeader(sha(txt))
	require.Equal(t, want, p.Headers["x-amz-checksum-sha256"])
	require.Equal(t, http.StatusOK, e.put(p, txt))
	// nội dung khác (cùng cỡ) bị object storage từ chối
	q, err := e.presign(e.sv(), e.a.ID, "b.txt", txt)
	require.NoError(t, err)
	require.GreaterOrEqual(t, e.put(q, bytes.ToUpper(txt)), 400)
}

func TestSubmitFlow(t *testing.T) {
	e := newSubEnv(t)
	out, err := e.submit(e.sv(), e.a.ID, "bai.txt", txt)
	require.NoError(t, err)
	require.Equal(t, 1, out.Version)
	require.Zero(t, out.LateDays)
	require.Equal(t, "bai.txt", out.FileName)
	var src, ms, mm string
	var ver int
	e.one(&src, `select source from submissions where id = $1`, out.ID)
	e.one(&ms, `select match_status from submissions where id = $1`, out.ID)
	e.one(&mm, `select match_method from submissions where id = $1`, out.ID)
	e.one(&ver, `select version from submissions where id = $1`, out.ID)
	require.Equal(t, []string{"IN_APP", "MATCHED", "IN_APP"}, []string{src, ms, mm})
	var n int
	e.one(&n, `select count(*) from outbox where topic = 'submission.created' and payload->>'submission_id' = $1`, out.ID.String())
	require.Equal(t, 1, n)
	e.one(&n, `select count(*) from audit_log where entity = 'submissions' and entity_id = $1`, out.ID.String())
	require.Equal(t, 1, n)
	got, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: e.sv(), Student: true}, e.course, e.a.ID)
	require.NoError(t, err)
	require.Equal(t, "SUBMITTED", got.State)
	require.Equal(t, 1, got.Submission.Version)
}

func TestSubmitIdentityFromJWT(t *testing.T) {
	e := newSubEnv(t)
	p, err := e.presign(e.sv(), e.a.ID, "a.txt", txt)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, e.put(p, txt))
	// vé của sinh viên A không dùng được cho B (cùng một câu trả lời với vé không tồn tại)
	_, err = e.sub.Complete(e.ctx, e.students[1], e.course, e.a.ID, p.UploadID, "")
	require.Equal(t, "UPLOAD_NOT_FOUND", code(err))
	out, err := e.sub.Complete(e.ctx, e.sv(), e.course, e.a.ID, p.UploadID, "")
	require.NoError(t, err)
	var sid uuid.UUID
	e.one(&sid, `select student_id from submissions where id = $1`, out.ID)
	require.Equal(t, e.sv(), sid)
}

func TestSubmitNoLocalDisk(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	e := newSubEnv(t)
	_, err := e.submit(e.sv(), e.a.ID, "a.txt", txt)
	require.NoError(t, err)
	ents, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, ents, "gateway không ghi tệp tạm cục bộ")
}

func TestPresignRateLimit(t *testing.T) {
	e := newSubEnv(t)
	for range 10 {
		_, err := e.presign(e.sv(), e.a.ID, "a.txt", txt)
		require.NoError(t, err)
	}
	_, err := e.presign(e.sv(), e.a.ID, "a.txt", txt)
	require.Equal(t, "RATE_LIMITED", code(err))
	require.Positive(t, err.(*apierr.Error).RetryAfter) //nolint:errorlint,forcetypeassert // retry_after
	_, err = e.presign(e.students[1], e.a.ID, "a.txt", txt)
	require.NoError(t, err, "hạn mức theo từng người")
}

func TestSubmitSniff(t *testing.T) {
	e := newSubEnv(t)
	bad := map[string][]byte{
		"a.pdf":  []byte("không phải pdf"),
		"b.docx": []byte("PK\x03\x04 giả zip"),
		"c.txt":  []byte("có byte NUL\x00ở giữa"),
		"d.md":   {0xff, 0xfe, 'x'},
	}
	for name, data := range bad {
		_, err := e.submit(e.sv(), e.a.ID, name, data)
		require.Equal(t, "FILE_CONTENT_MISMATCH", code(err), name)
	}
	var n int
	e.one(&n, `select count(*) from submissions where assignment_id = $1`, e.a.ID)
	require.Zero(t, n, "tệp sai nội dung không thành bài nộp")
	infos, err := e.blob.List(e.ctx, "courses/")
	require.NoError(t, err)
	require.Empty(t, infos, "tệp bị từ chối bị xoá khỏi object storage")
	good := map[string][]byte{"a.pdf": []byte("%PDF-1.4 ..."), "b.txt": []byte("Tiếng Việt có dấu\n"), "c.md": []byte("# Tiêu đề\n")}
	for name, data := range good {
		_, err := e.submit(e.sv(), e.a.ID, name, data)
		require.NoError(t, err, name)
	}
	_, err = e.submit(e.sv(), e.a.ID, "e.docx", docx(t, 0, true))
	require.NoError(t, err)
}

func TestSubmitContentSniff(t *testing.T) { TestSubmitSniff(t) }

func TestSubmitEmptyFile(t *testing.T) {
	e := newSubEnv(t)
	_, err := e.sub.Presign(e.ctx, e.sv(), e.course, e.a.ID, submission.PresignIn{Filename: "a.txt", SizeBytes: 0, SHA256: sha(nil)})
	require.Equal(t, "FILE_EMPTY", code(err))
}

// docx dựng zip: `pad` byte rác (lưu thẳng) ĐỨNG TRƯỚC để thư mục trung tâm nằm xa đầu tệp; full=false thì thiếu word/document.xml.
func docx(t *testing.T, pad int, full bool) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	add := func(name string, data []byte) {
		f, err := w.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		require.NoError(t, err)
		_, err = f.Write(data)
		require.NoError(t, err)
	}
	if pad > 0 {
		add("rac.bin", bytes.Repeat([]byte("x"), pad))
	}
	add("[Content_Types].xml", []byte("<Types/>"))
	if full {
		add("word/document.xml", []byte("<w:document/>"))
	}
	require.NoError(t, w.Close())
	return b.Bytes()
}

func TestDocxCentralDirectoryAtEnd(t *testing.T) {
	e := newSubEnv(t)
	_, err := e.submit(e.sv(), e.a.ID, "lon.docx", docx(t, 300<<10, true)) // thư mục trung tâm cách đầu tệp > 300 KiB
	require.NoError(t, err)
	_, err = e.submit(e.sv(), e.a.ID, "thieu.docx", docx(t, 1024, false))
	require.Equal(t, "FILE_CONTENT_MISMATCH", code(err))
}

func TestSubmitUnpublished(t *testing.T) {
	e := newSubEnv(t)
	d := e.create("NHAP")
	_, err := e.presign(e.sv(), d.ID, "a.txt", txt)
	require.Equal(t, "ASSIGNMENT_NOT_FOUND", code(err))
	_, err = e.svc.Mine(e.ctx, assignment.Actor{UserID: e.sv(), Student: true}, e.course, d.ID)
	require.Equal(t, "ASSIGNMENT_NOT_FOUND", code(err))
}

func TestSubmitChannelNotAllowed(t *testing.T) {
	e := newSubEnv(t)
	i := in("UPONLY")
	i.Channels = []string{"UPLOAD"}
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(t, err)
	e.publish(a)
	_, err = e.presign(e.sv(), a.ID, "a.txt", txt)
	require.Equal(t, "CHANNEL_NOT_ALLOWED", code(err))
}

func TestSubmitOtherCourseForbidden(t *testing.T) {
	// guard của route chặn người ngoài lớp (403) — ở tầng dịch vụ, bài của lớp khác đơn giản không tồn tại trong lớp đang xét.
	e := newSubEnv(t)
	_, err := e.sub.Presign(e.ctx, e.sv(), uuid.New(), e.a.ID, submission.PresignIn{Filename: "a.txt", SizeBytes: 5, SHA256: sha(txt)})
	require.Equal(t, "NOT_FOUND", code(err))
	other := newEnv(t)
	_, err = e.sub.Presign(e.ctx, e.sv(), other.course, e.a.ID, submission.PresignIn{Filename: "a.txt", SizeBytes: 5, SHA256: sha(txt)})
	require.Equal(t, "NOT_FOUND", code(err), "pool khác: lớp không có trong DB này")
}

func TestLateWindowTable(t *testing.T) {
	due := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	a := store.Assignment{DueAt: due, AllowLate: true, LateMaxDays: 3}
	cases := []struct {
		name string
		at   time.Duration
		days int
		ok   bool
	}{
		{"−1 giờ", -time.Hour, 0, true},
		{"đúng hạn", 0, 0, true},
		{"+1 phút", time.Minute, 1, true},
		{"+8 giờ 11 phút", 8*time.Hour + 11*time.Minute, 1, true},
		{"+24 giờ đúng", 24 * time.Hour, 1, true},
		{"+24 giờ 1 phút", 24*time.Hour + time.Minute, 2, true},
		{"+40 giờ", 40 * time.Hour, 2, true},
		{"+m ngày đúng", 72 * time.Hour, 3, true},
		{"+m ngày 1 phút", 72*time.Hour + time.Minute, 0, false},
	}
	for _, c := range cases {
		days, err := submission.Gate(a, due.Add(c.at))
		if c.ok {
			require.NoError(t, err, c.name)
			require.Equal(t, c.days, days, c.name)
		} else {
			require.Equal(t, "SUBMISSION_CLOSED", code(err), c.name)
		}
	}
}

func TestLateDaysCeil(t *testing.T) {
	due := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { x := due.Add(d); return &x }
	require.Equal(t, 0, grading.LateDays(nil, due))
	require.Equal(t, 0, grading.LateDays(at(-time.Hour), due))
	require.Equal(t, 1, grading.LateDays(at(8*time.Hour+11*time.Minute), due))
	require.Equal(t, 2, grading.LateDays(at(40*time.Hour), due))
	require.Equal(t, 1, grading.LateDays(at(24*time.Hour), due))
}

func TestClosedAfterCutoff(t *testing.T) {
	e := newSubEnv(t)
	e.clk.Advance(72*time.Hour + time.Minute) // qua hạn nhưng chưa bật nộp muộn
	_, err := e.presign(e.sv(), e.a.ID, "a.txt", txt)
	require.Equal(t, "SUBMISSION_CLOSED", code(err))
}

func TestNoLateWhenNotAllowed(t *testing.T) {
	e := newSubEnv(t)
	p, err := e.presign(e.sv(), e.a.ID, "a.txt", txt) // vé xin lúc còn hạn
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, e.put(p, txt))
	e.clk.Advance(73 * time.Hour) // hạn qua trước khi hoàn tất: kiểm lại ở complete
	_, err = e.sub.Complete(e.ctx, e.sv(), e.course, e.a.ID, p.UploadID, "")
	require.Equal(t, "SUBMISSION_CLOSED", code(err))
	var n int
	e.one(&n, `select count(*) from submissions where assignment_id = $1`, e.a.ID)
	require.Zero(t, n, "không tạo bản ghi")
}

func TestLateDaysComputedNotStored(t *testing.T) {
	e := newSubEnv(t)
	var n int
	e.one(&n, `select count(*) from information_schema.columns where table_name = 'submissions' and column_name in ('late_days', 'is_late')`)
	require.Zero(t, n)
}

// lateAssignment: cho nộp muộn tối đa 3 ngày.
func (e *subEnv) lateAssignment() assignment.Assignment {
	i := in("LATE1")
	i.AllowLate, i.LatePenaltyPerDay, i.LateMaxDays, i.DueAt = true, "1", 3, e.clk.Now().Add(24*time.Hour)
	a, err := e.svc.Create(e.ctx, e.act, e.course, i)
	require.NoError(e.t, err)
	e.publish(a)
	return a
}

func TestResubmitCreatesVersion(t *testing.T) {
	e := newSubEnv(t)
	v1, err := e.submit(e.sv(), e.a.ID, "v1.txt", txt)
	require.NoError(t, err)
	v2, err := e.submit(e.sv(), e.a.ID, "v2.txt", []byte("Bản hai.\n"))
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, []int{v1.Version, v2.Version})
	var n int
	e.one(&n, `select count(*) from submissions where assignment_id = $1 and student_id = $2`, e.a.ID, e.sv())
	require.Equal(t, 2, n, "bản cũ giữ nguyên")
	got, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: e.sv(), Student: true}, e.course, e.a.ID)
	require.NoError(t, err)
	require.Equal(t, "v2.txt", got.Submission.FileName, "chỉ bản mới nhất hiện hành")
	// sinh viên khác không bị ảnh hưởng
	o, err := e.submit(e.students[1], e.a.ID, "o.txt", txt)
	require.NoError(t, err)
	require.Equal(t, 1, o.Version)
}

func TestOnlyLatestVersionGraded(t *testing.T) {
	e := newSubEnv(t)
	_, err := e.submit(e.sv(), e.a.ID, "v1.txt", txt)
	require.NoError(t, err)
	v2, err := e.submit(e.sv(), e.a.ID, "v2.txt", []byte("Bản hai.\n"))
	require.NoError(t, err)
	// "hiện hành" suy từ bản nộp mới nhất: truy vấn trạng thái / số liệu chỉ nhìn bản có version lớn nhất
	row, err := store.New(e.pool).SubmissionLatestMatched(e.ctx, store.SubmissionLatestMatchedParams{CourseID: e.course, AssignmentID: e.a.ID, StudentID: e.sv()})
	require.NoError(t, err)
	require.Equal(t, v2.ID, row.ID)
	cs, err := store.New(e.pool).AssignmentCounts(e.ctx, store.AssignmentCountsParams{CourseID: e.course, Ids: []string{e.a.ID.String()}})
	require.NoError(t, err)
	require.Equal(t, int32(1), cs[0].Submitted, "hai phiên bản của một sinh viên = một người đã nộp")
}

func (e *subEnv) result(sub uuid.UUID, status string) {
	e.t.Helper()
	pub := "null"
	if status == "PUBLISHED" {
		pub = "now()"
	}
	e.exec(`insert into grading_results (course_id, submission_id, run_state, status, published_at) values ($1, $2, 'DONE', $3, `+pub+`)`, e.course, sub, status)
}

func TestResubmitAfterReviewedRejected(t *testing.T) {
	for _, st := range []string{"REVIEWED", "PUBLISHED"} {
		e := newSubEnv(t)
		v1, err := e.submit(e.sv(), e.a.ID, "v1.txt", txt)
		require.NoError(t, err)
		e.result(v1.ID, st)
		_, err = e.submit(e.sv(), e.a.ID, "v2.txt", []byte("Bản hai.\n"))
		require.Equal(t, "SUBMISSION_LOCKED", code(err), st)
	}
	// kết quả DRAFT thì vẫn nộp lại được
	e := newSubEnv(t)
	v1, err := e.submit(e.sv(), e.a.ID, "v1.txt", txt)
	require.NoError(t, err)
	e.result(v1.ID, "DRAFT")
	_, err = e.submit(e.sv(), e.a.ID, "v2.txt", []byte("Bản hai.\n"))
	require.NoError(t, err)
}

func TestSubmissionLockedMessageGeneric(t *testing.T) {
	e := newSubEnv(t)
	v1, err := e.submit(e.sv(), e.a.ID, "v1.txt", txt)
	require.NoError(t, err)
	e.result(v1.ID, "REVIEWED")
	_, err = e.submit(e.sv(), e.a.ID, "v2.txt", []byte("Bản hai.\n"))
	ae := err.(*apierr.Error) //nolint:errorlint,forcetypeassert // kiểm lời văn
	msg := strings.ToLower(ae.Message + " " + ae.Code)
	for _, leak := range []string{"duyệt", "reviewed", "published", "công bố", "điểm"} {
		require.NotContains(t, msg, leak)
	}
	require.Nil(t, ae.Details)
}

func TestResubmitLateByOwnTime(t *testing.T) {
	e := newSubEnv(t)
	a := e.lateAssignment()
	v1, err := e.submit(e.sv(), a.ID, "v1.txt", txt) // đúng hạn
	require.NoError(t, err)
	require.Zero(t, v1.LateDays)
	e.clk.Advance(30 * time.Hour) // trễ 6 giờ → 1 ngày
	v2, err := e.submit(e.sv(), a.ID, "v2.txt", []byte("Bản hai.\n"))
	require.NoError(t, err)
	require.Equal(t, 1, v2.LateDays)
	var at1, at2 time.Time
	e.one(&at1, `select submitted_at from submissions where id = $1`, v1.ID)
	e.one(&at2, `select submitted_at from submissions where id = $1`, v2.ID)
	require.True(t, at2.After(at1))
	require.Zero(t, grading.LateDays(&at1, a.DueAt), "bản 1 vẫn đúng hạn")
}

func TestSubmitVersionAdvisoryLock(t *testing.T) {
	e := newSubEnv(t)
	const n = 6
	type up struct {
		p submission.PresignOut
		d []byte
	}
	var ups []up
	for i := range n {
		d := []byte(strings.Repeat("x", i+1) + "\n")
		p, err := e.presign(e.sv(), e.a.ID, "p.txt", d)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, e.put(p, d))
		ups = append(ups, up{p, d})
	}
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i, u := range ups {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = e.sub.Complete(context.Background(), e.sv(), e.course, e.a.ID, u.p.UploadID, "")
		}()
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}
	rows, err := e.pool.Query(e.ctx, `select version from submissions where assignment_id = $1 and student_id = $2`, e.a.ID, e.sv())
	require.NoError(t, err)
	defer rows.Close()
	var vs []int
	for rows.Next() {
		var v int
		require.NoError(t, rows.Scan(&v))
		vs = append(vs, v)
	}
	sort.Ints(vs)
	require.Equal(t, []int{1, 2, 3, 4, 5, 6}, vs, "không va 23505, version liên tục")
}

func TestOrphanUploadCleanup(t *testing.T) {
	e := newSubEnv(t)
	used, err := e.submit(e.sv(), e.a.ID, "dung.txt", txt)
	require.NoError(t, err)
	p, err := e.presign(e.sv(), e.a.ID, "mocoi.txt", txt)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, e.put(p, txt)) // tải lên nhưng không complete
	n, err := e.sub.OrphanCleanup(e.ctx)
	require.NoError(t, err)
	require.Zero(t, n, "chưa quá hạn: giữ")
	e.clk.Advance(25 * time.Hour)
	n, err = e.sub.OrphanCleanup(e.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	_, err = e.blob.Stat(e.ctx, p.BlobKey)
	require.ErrorIs(t, err, blob.ErrNotFound)
	var key string
	e.one(&key, `select blob_key from submissions where id = $1`, used.ID)
	_, err = e.blob.Stat(e.ctx, key)
	require.NoError(t, err, "tệp của bài nộp được giữ")
}

func keysOf(t *testing.T, v any, prefix string, out map[string]bool) {
	t.Helper()
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			out[p] = true
			keysOf(t, c, p, out)
		}
	case []any:
		for _, c := range x {
			keysOf(t, c, prefix+"[]", out)
		}
	}
}

func allowlist(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile("testdata/student_dto_allowlist.json")
	require.NoError(t, err)
	var m map[string][]string
	require.NoError(t, json.Unmarshal(raw, &m))
	return m[name]
}

func keysIn(t *testing.T, v any) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var anyV any
	require.NoError(t, json.Unmarshal(raw, &anyV))
	set := map[string]bool{}
	keysOf(t, anyV, "", set)
	var ks []string
	for k := range set {
		ks = append(ks, strings.ReplaceAll(k, "[].", "[]."))
	}
	sort.Strings(ks)
	return ks
}

// TestMineDTOAllowlist — DTO sinh viên là struct riêng: tập khoá JSON phải đúng allowlist (thêm khoá nhạy cảm → test đỏ cho tới khi sửa allowlist có chủ đích).
func TestMineDTOAllowlist(t *testing.T) {
	now := time.Now()
	full := assignment.MineOut{State: "SUBMITTED", Assignment: assignment.MineAssignment{Criteria: []assignment.StudentCriterion{{ID: "c1"}}},
		Submission: &assignment.MineSubmission{ReceivedAt: now}}
	got := keysIn(t, full)
	// khoá của phần tử mảng gộp dạng `criteria[].x`; allowlist ghi `criteria[].x` và `criteria`
	want := allowlist(t, "mine")
	sort.Strings(want)
	require.Equal(t, want, got)
	item := assignment.StudentListItem{My: assignment.MyState{SubmittedAt: &now}}
	wantItem := allowlist(t, "list_item")
	sort.Strings(wantItem)
	require.Equal(t, wantItem, keysIn(t, item))
	for _, k := range append(got, keysIn(t, item)...) {
		for _, bad := range []string{"ai_scores", "flags", "confidence", "run_state", "reference_answer", "guidance", "submission_code", "student_id"} {
			require.NotContains(t, k, bad)
		}
	}
}

func TestMineOwnOnly(t *testing.T) {
	e := newSubEnv(t)
	_, err := e.submit(e.students[1], e.a.ID, "cua-b.txt", txt)
	require.NoError(t, err)
	a, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: e.sv(), Student: true}, e.course, e.a.ID)
	require.NoError(t, err)
	require.Equal(t, "NOT_SUBMITTED", a.State)
	require.Nil(t, a.Submission, "không thấy bài của người khác")
	b, err := e.svc.Mine(e.ctx, assignment.Actor{UserID: e.students[1], Student: true}, e.course, e.a.ID)
	require.NoError(t, err)
	require.Equal(t, "cua-b.txt", b.Submission.FileName)
}

func TestStudentListMyState(t *testing.T) {
	e := newSubEnv(t)
	_, err := e.submit(e.sv(), e.a.ID, "a.txt", txt)
	require.NoError(t, err)
	pg, err := e.svc.List(e.ctx, assignment.Actor{UserID: e.sv(), Student: true}, e.course, httpxPage(10))
	require.NoError(t, err)
	require.Len(t, pg.StudentItems, 1)
	require.Equal(t, "SUBMITTED", pg.StudentItems[0].My.State)
	pg, err = e.svc.List(e.ctx, assignment.Actor{UserID: e.students[1], Student: true}, e.course, httpxPage(10))
	require.NoError(t, err)
	require.Equal(t, "NOT_SUBMITTED", pg.StudentItems[0].My.State)
}
