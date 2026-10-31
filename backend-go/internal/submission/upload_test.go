package submission_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/submission"
	"github.com/edupilot/backend-go/internal/testutil"
)

type uenv struct {
	t       *testing.T
	ctx     context.Context
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	blob    *blob.Store
	svc     *submission.Service
	course  uuid.UUID
	teacher uuid.UUID
	ta      uuid.UUID
	asg     uuid.UUID
	byCode  map[string]uuid.UUID // MSSV → sinh viên (dòng ROSTER)
}

func (e *uenv) exec(sql string, args ...any) {
	e.t.Helper()
	_, err := e.pool.Exec(e.ctx, sql, args...)
	require.NoError(e.t, err, sql)
}

func (e *uenv) one(dst any, sql string, args ...any) {
	e.t.Helper()
	require.NoError(e.t, e.pool.QueryRow(e.ctx, sql, args...).Scan(dst), sql)
}

func (e *uenv) user(role, email string, verified bool) uuid.UUID {
	var id uuid.UUID
	e.one(&id, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', case when $3 then now() end) returning id`, email, role, verified)
	return id
}

func (e *uenv) enroll(user uuid.UUID, role, code, via string) {
	var snap any
	if code != "" {
		snap = code
	}
	e.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, $3::enrollment_role, 'ACTIVE', $4::enrollment_joined_via, $5)`, e.course, user, role, via, snap)
}

func newUenv(t *testing.T) *uenv {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	st, err := blob.New(t.Context(), blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "up-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	e := &uenv{t: t, ctx: t.Context(), pool: pool, rdb: rdb, blob: st, byCode: map[string]uuid.UUID{}}
	e.svc = &submission.Service{Pool: pool, Redis: rdb, Blob: st, Clock: clock.Real{}, Log: slog.New(slog.DiscardHandler)}
	e.teacher = e.user("TEACHER", "gv."+uuid.NewString()[:6]+"@example.test", true)
	e.ta = e.user("TA", "ta."+uuid.NewString()[:6]+"@example.test", true)
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i])%len(alpha)]
	}
	e.one(&e.course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'ANM', '2026-2027-HK1', $2, $3) returning id`, "G-"+id.String()[:8], string(b), e.teacher)
	e.enroll(e.teacher, "TEACHER", "", "ADMIN")
	e.enroll(e.ta, "TA", "", "ADMIN")
	for _, c := range []string{"20229001", "20229002", "20229003"} {
		u := e.user("STUDENT", "sv"+c+"@example.test", true)
		e.enroll(u, "STUDENT", c, "ROSTER")
		e.byCode[c] = u
	}
	e.one(&e.asg, `insert into assignments (course_id, title, due_at, submission_code, created_by, channels, published_at, rubric) values ($1, 'BT03', now() + interval '3 days', 'BT03', $2, ARRAY['IN_APP','UPLOAD'], now(), '[]') returning id`, e.course, e.teacher)
	return e
}

// ticket đặt một đối tượng + vé tải lên (như `uploads/presign` purpose=submission_upload) và trả upload_id.
func (e *uenv) ticket(actor uuid.UUID, name string, data []byte) uuid.UUID {
	e.t.Helper()
	id, _ := e.ticketKey(actor, name, data)
	return id
}

func (e *uenv) ticketKey(actor uuid.UUID, name string, data []byte) (uuid.UUID, string) {
	e.t.Helper()
	id := uuid.New()
	key := fmt.Sprintf("courses/%s/submissions/%s/%s", e.course, id, name)
	require.NoError(e.t, e.blob.Put(e.ctx, key, bytes.NewReader(data), int64(len(data)), "application/octet-stream"))
	rk := appredis.Key("upload", id.String())
	require.NoError(e.t, e.rdb.HSet(e.ctx, rk, map[string]any{"user_id": actor.String(), "course_id": e.course.String(), "filename": name, "size": len(data), "blob_key": key, "purpose": "submission_upload"}).Err())
	return id, key
}

func (e *uenv) upload(name string, data []byte) (submission.UploadOut, *uuid.UUID, error) {
	return e.svc.UploadComplete(e.ctx, e.ta, e.course, e.asg, e.ticket(e.ta, name, data))
}

func zipOf(t *testing.T, entries map[string][]byte, order ...string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	names := order
	if len(names) == 0 {
		for n := range entries {
			names = append(names, n)
		}
	}
	for _, n := range names {
		f, err := w.Create(n)
		require.NoError(t, err)
		_, err = f.Write(entries[n])
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return b.Bytes()
}

func ecode(err error) string {
	if ae, ok := err.(*apierr.Error); ok { //nolint:errorlint // apierr trả thẳng
		return ae.Code
	}
	return ""
}

func (e *uenv) count(sql string, args ...any) int {
	var n int
	e.one(&n, sql, args...)
	return n
}

func TestUploadSingleFile(t *testing.T) {
	e := newUenv(t)
	out, job, err := e.upload("bai-20229001.txt", []byte("Bài của sinh viên 1\n"))
	require.NoError(t, err)
	require.Nil(t, job)
	require.Equal(t, submission.UploadOut{Created: 1, Matched: 1, Deduped: []string{}, Rejected: []submission.Rejection{}}, out)
	var src, method string
	var ver int
	var at *time.Time
	require.NoError(t, e.pool.QueryRow(e.ctx, `select source, match_method, version, submitted_at from submissions where course_id = $1`, e.course).Scan(&src, &method, &ver, &at))
	require.Equal(t, []any{"UPLOAD", "CODE", 1, (*time.Time)(nil)}, []any{src, method, ver, at}, "UPLOAD không bao giờ tính muộn (submitted_at NULL)")
	out, _, err = e.upload("khong-mssv.txt", []byte("Không ghi MSSV\n"))
	require.NoError(t, err)
	require.Equal(t, 1, out.Unmatched)
	require.Equal(t, 1, e.count(`select count(*) from submissions where match_status = 'UNMATCHED' and unmatched_reason = 'NO_STUDENT_SIGNAL' and assignment_id = $1`, e.asg))
}

func TestUploadZipEntries(t *testing.T) {
	e := newUenv(t)
	z := zipOf(t, map[string][]byte{
		"20229001.txt": []byte("Một\n"), "20229002/bai.md": []byte("# Hai\n"), "khong-ma.txt": []byte("Ba\n"),
		"ghi-chu.exe": []byte("MZ"), "thu-muc/": nil, "__MACOSX/._x.txt": []byte("rác"), ".an.txt": []byte("ẩn"),
	}, "20229001.txt", "20229002/bai.md", "khong-ma.txt", "ghi-chu.exe", "thu-muc/", "__MACOSX/._x.txt", ".an.txt")
	out, job, err := e.upload("lop.zip", z)
	require.NoError(t, err)
	require.Nil(t, job, "4 mục hợp lệ → đồng bộ")
	require.Equal(t, 3, out.Created)
	require.Equal(t, 2, out.Matched, "MSSV lấy cả từ tên thư mục trong ZIP")
	require.Equal(t, 1, out.Unmatched)
	require.Equal(t, []submission.Rejection{{Entry: "ghi-chu.exe", Code: "ENTRY_TYPE_NOT_ALLOWED"}}, out.Rejected, "thư mục / __MACOSX / tệp ẩn bị bỏ lặng lẽ")
	// > 5 mục → việc nền
	many := map[string][]byte{}
	for i := range 6 {
		many[fmt.Sprintf("n%d.txt", i)] = []byte(fmt.Sprintf("bài %d\n", i))
	}
	_, job, err = e.svc.UploadComplete(e.ctx, e.ta, e.course, e.asg, e.ticket(e.ta, "nhieu.zip", zipOf(t, many)))
	require.Equal(t, "SERVICE_UNAVAILABLE", ecode(err), "chưa cấu hình hàng việc")
	require.Nil(t, job)
	out, err = e.svc.ProcessZip(e.ctx, submission.ZipPayload{CourseID: e.course, AssignmentID: e.asg, Actor: e.ta, BlobKey: e.keyOfLast("nhieu.zip"), FileName: "nhieu.zip"})
	require.NoError(t, err)
	require.Equal(t, 6, out.Created)
}

func (e *uenv) keyOfLast(name string) string {
	infos, err := e.blob.List(e.ctx, "courses/"+e.course.String()+"/submissions/")
	require.NoError(e.t, err)
	for _, i := range infos {
		if strings.HasSuffix(i.Key, "/"+name) {
			return i.Key
		}
	}
	e.t.Fatalf("không thấy %s", name)
	return ""
}

func TestZipSlipRejected(t *testing.T) {
	e := newUenv(t)
	z := zipOf(t, map[string][]byte{"../evil.txt": []byte("x"), "/abs.txt": []byte("x"), "a/../../b.txt": []byte("x"), "ok.txt": []byte("tốt\n"), "c\\d.txt": []byte("x"), "ctrl\x01.txt": []byte("x")},
		"../evil.txt", "/abs.txt", "a/../../b.txt", "ok.txt", "c\\d.txt", "ctrl\x01.txt")
	_, key := e.ticketKey(e.ta, "slip.zip", z)
	out, err := e.svc.ProcessZip(e.ctx, submission.ZipPayload{CourseID: e.course, AssignmentID: e.asg, Actor: e.ta, BlobKey: key, FileName: "slip.zip"})
	require.NoError(t, err)
	require.Equal(t, 1, out.Created)
	require.Len(t, out.Rejected, 5)
	for _, r := range out.Rejected {
		require.Equal(t, "ENTRY_UNSAFE", r.Code, r.Entry)
	}
	infos, err := e.blob.List(e.ctx, "")
	require.NoError(t, err)
	for _, i := range infos {
		require.NotContains(t, i.Key, "evil", "không có đối tượng nào ghi ra ngoài tiền tố của lớp")
		require.True(t, strings.HasPrefix(i.Key, "courses/"+e.course.String()+"/submissions/"), i.Key)
	}
}

func TestZipNestedRejected(t *testing.T) {
	e := newUenv(t)
	inner := zipOf(t, map[string][]byte{"x.txt": []byte("x")})
	out, _, err := e.upload("long.zip", zipOf(t, map[string][]byte{"inner.zip": inner, "ok.txt": []byte("ổn\n")}, "inner.zip", "ok.txt"))
	require.NoError(t, err)
	require.Equal(t, []submission.Rejection{{Entry: "inner.zip", Code: "ENTRY_UNSAFE"}}, out.Rejected)
	require.Equal(t, 1, out.Created)
}

// TestZipBombCountedOnRead — giới hạn tổng giải nén đếm KHI ĐỌC: 14 mục ~15 MiB toàn số 0 (nén rất nhỏ) vượt 200 MiB → dừng ở ZIP_LIMIT, các mục trước đó vẫn được nhập.
func TestZipBombCountedOnRead(t *testing.T) {
	e := newUenv(t)
	zeros := bytes.Repeat([]byte("a"), 15<<20)
	entries := map[string][]byte{}
	var order []string
	for i := range 14 {
		n := fmt.Sprintf("%08d.txt", 20229000+i)
		entries[n] = zeros
		order = append(order, n)
	}
	z := zipOf(t, entries, order...)
	require.Less(t, len(z), 2<<20, "ZIP nén rất nhỏ")
	_, key := e.ticketKey(e.ta, "bom.zip", z)
	out, err := e.svc.ProcessZip(e.ctx, submission.ZipPayload{CourseID: e.course, AssignmentID: e.asg, Actor: e.ta, BlobKey: key, FileName: "bom.zip"})
	require.NoError(t, err)
	require.Contains(t, out.Rejected, submission.Rejection{Entry: "*", Code: "ZIP_LIMIT"})
	require.LessOrEqual(t, out.Created, 13)
	require.Positive(t, out.Created)
	// mục lớn hơn 20 MiB bị từ chối riêng, không kéo cả lô
	big := zipOf(t, map[string][]byte{"lon.txt": bytes.Repeat([]byte("b"), 21<<20), "nho.txt": []byte("nhỏ\n")}, "lon.txt", "nho.txt")
	out, _, err = e.upload("lon.zip", big)
	require.NoError(t, err)
	require.Equal(t, []submission.Rejection{{Entry: "lon.txt", Code: "FILE_TOO_LARGE"}}, out.Rejected)
	require.Equal(t, 1, out.Created)
}

func TestZipBadEntryDoesNotFailBatch(t *testing.T) {
	e := newUenv(t)
	out, _, err := e.upload("hon-hop.zip", zipOf(t, map[string][]byte{
		"rong.txt": {}, "pdf-gia.pdf": []byte("không phải pdf"), "nul.txt": []byte("a\x00b"), "20229003.md": []byte("# Tốt\n"),
	}, "rong.txt", "pdf-gia.pdf", "nul.txt", "20229003.md"))
	require.NoError(t, err)
	require.Equal(t, 1, out.Created)
	got := map[string]string{}
	for _, r := range out.Rejected {
		got[r.Entry] = r.Code
	}
	require.Equal(t, map[string]string{"rong.txt": "FILE_EMPTY", "pdf-gia.pdf": "FILE_INVALID", "nul.txt": "FILE_INVALID"}, got)
}

func TestUploadIdempotentBySha(t *testing.T) {
	e := newUenv(t)
	data := []byte("Bài 20229001\n")
	out, _, err := e.upload("20229001.txt", data)
	require.NoError(t, err)
	require.Equal(t, 1, out.Created)
	before := e.count(`select count(*) from submissions where course_id = $1`, e.course)
	z := zipOf(t, map[string][]byte{"20229001.txt": data})
	for range 2 {
		out, _, err = e.upload("lai.zip", z)
		require.NoError(t, err)
		require.Zero(t, out.Created)
	}
	out, _, err = e.upload("20229001.txt", data)
	require.NoError(t, err)
	require.Equal(t, []string{"20229001.txt"}, out.Deduped)
	require.Equal(t, before, e.count(`select count(*) from submissions where course_id = $1`, e.course), "không phiên bản mới")
}

func TestUploadDifferentContentNewVersion(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("20229001.txt", []byte("Bản một\n"))
	require.NoError(t, err)
	out, _, err := e.upload("20229001.txt", []byte("Bản hai khác nội dung\n"))
	require.NoError(t, err)
	require.Equal(t, 1, out.Matched)
	require.Equal(t, 2, e.count(`select max(version) from submissions where student_id = $1`, e.byCode["20229001"]))
}

func TestUploadUnpublished(t *testing.T) {
	e := newUenv(t)
	e.exec(`update assignments set published_at = null where id = $1`, e.asg)
	_, _, err := e.upload("a.txt", []byte("x\n"))
	require.Equal(t, "ASSIGNMENT_NOT_PUBLISHED", ecode(err))
}

func TestUploadArchivedCourse(t *testing.T) {
	e := newUenv(t)
	e.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, e.course)
	_, _, err := e.upload("a.txt", []byte("x\n"))
	require.Equal(t, "COURSE_ARCHIVED", ecode(err))
}

func TestUploadChannelOff(t *testing.T) {
	e := newUenv(t)
	e.exec(`update assignments set channels = ARRAY['IN_APP'] where id = $1`, e.asg)
	_, _, err := e.upload("a.txt", []byte("x\n"))
	require.Equal(t, "CHANNEL_NOT_ALLOWED", ecode(err))
}

func (e *uenv) result(sub uuid.UUID, status string) {
	e.t.Helper()
	pub := "null"
	if status == "PUBLISHED" {
		pub = "now()"
	}
	e.exec(`insert into grading_results (course_id, submission_id, run_state, status, published_at) values ($1, $2, 'DONE', $3, `+pub+`)`, e.course, sub, status)
}

func (e *uenv) latestSub(student uuid.UUID) uuid.UUID {
	var id uuid.UUID
	e.one(&id, `select id from submissions where student_id = $1 and match_status = 'MATCHED' order by version desc limit 1`, student)
	return id
}

// TestUploadAlreadyReviewed — bài mới cho sinh viên đã REVIEWED / PUBLISHED không đè: vào UNMATCHED ALREADY_REVIEWED kèm gợi ý.
func TestUploadAlreadyReviewed(t *testing.T) {
	for _, st := range []string{"REVIEWED", "PUBLISHED"} {
		e := newUenv(t)
		_, _, err := e.upload("20229001.txt", []byte("Bản một\n"))
		require.NoError(t, err)
		e.result(e.latestSub(e.byCode["20229001"]), st)
		out, _, err := e.upload("20229001-v2.txt", []byte("Bản hai\n"))
		require.NoError(t, err)
		require.Equal(t, 1, out.Unmatched, st)
		var reason string
		var cand uuid.UUID
		require.NoError(t, e.pool.QueryRow(e.ctx, `select unmatched_reason, candidate_student_id from submissions where match_status = 'UNMATCHED'`).Scan(&reason, &cand))
		require.Equal(t, "ALREADY_REVIEWED", reason)
		require.Equal(t, e.byCode["20229001"], cand)
		require.Equal(t, 1, e.count(`select max(version) from submissions where student_id = $1`, e.byCode["20229001"]), "không phiên bản mới, điểm không bị đè")
	}
}

// TestIngestAlreadyReviewedAllSources — kiểm tra ở Ingest chung: UPLOAD / IMAP / TEAMS đều không đè bài đã duyệt.
func TestIngestAlreadyReviewedAllSources(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("20229001.txt", []byte("Bản một\n"))
	require.NoError(t, err)
	e.result(e.latestSub(e.byCode["20229001"]), "REVIEWED")
	env, err := submission.LoadEnv(e.ctx, store.New(e.pool), e.course, "UPLOAD")
	require.NoError(t, err)
	for i, src := range []string{"UPLOAD", "IMAP", "TEAMS"} {
		raw := submission.Raw{Source: src, ExternalRef: fmt.Sprintf("ref-%d", i), ExternalTS: time.Unix(int64(i+1), 0), SenderVerified: true, FromEmail: "sv20229001@example.test", Subject: "[BT03]",
			FileName: "a.txt", BlobKey: "k", FileMime: "text/plain", SHA256: "x", FileSize: 1}
		tx, err := e.pool.Begin(e.ctx)
		require.NoError(t, err)
		o, err := submission.Ingest(e.ctx, tx, env, raw, nil)
		require.NoError(t, err, src)
		require.NoError(t, tx.Commit(e.ctx))
		require.False(t, o.Matched, src)
		require.Equal(t, "ALREADY_REVIEWED", o.Reason, src)
	}
	require.Equal(t, 1, e.count(`select max(version) from submissions where student_id = $1`, e.byCode["20229001"]))
}

func (e *uenv) firstUnmatched() (uuid.UUID, int64) {
	pg, err := e.svc.ListUnmatched(e.ctx, e.course, httpx.PageParams{Limit: 10})
	require.NoError(e.t, err)
	require.NotEmpty(e.t, pg.Items)
	return pg.Items[0].ID, pg.Items[0].Version
}

func (e *uenv) staff() submission.Staff { return submission.Staff{UserID: e.teacher} }

func TestManualMatch(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("Bài không MSSV\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	out, err := e.svc.Match(e.ctx, e.staff(), e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229003"], Version: v})
	require.NoError(t, err)
	require.Equal(t, 1, out.Version)
	var method, status string
	require.NoError(t, e.pool.QueryRow(e.ctx, `select match_method, match_status from submissions where id = $1`, id).Scan(&method, &status))
	require.Equal(t, []string{"MANUAL", "MATCHED"}, []string{method, status})
	pg, err := e.svc.ListUnmatched(e.ctx, e.course, httpx.PageParams{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, pg.Items, "biến khỏi tab")
}

func TestManualMatchVersionBump(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("20229001.txt", []byte("Bản một\n"))
	require.NoError(t, err)
	_, _, err = e.upload("khong-ma.txt", []byte("Bản hai không MSSV\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	out, err := e.svc.Match(e.ctx, e.staff(), e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229001"], Version: v})
	require.NoError(t, err)
	require.Equal(t, 2, out.Version, "đã có bản 1 → bản kế tiếp")
}

func TestManualMatchAlreadyReviewedRejected(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("20229001.txt", []byte("Bản một\n"))
	require.NoError(t, err)
	e.result(e.latestSub(e.byCode["20229001"]), "PUBLISHED")
	_, _, err = e.upload("khong-ma.txt", []byte("Bản hai\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229001"], Version: v})
	require.Equal(t, "ALREADY_REVIEWED", ecode(err))
	// TA không gán được cho bài đã có điểm công bố (Q5)
	_, err = e.svc.Match(e.ctx, submission.Staff{UserID: e.ta, IsTA: true}, e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229002"], Version: v})
	require.Equal(t, "FORBIDDEN", ecode(err))
}

func TestManualMatchAudit(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("x\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	_, err = e.svc.Match(e.ctx, submission.Staff{UserID: e.ta, IsTA: true}, e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229002"], Version: v})
	require.NoError(t, err)
	var actor uuid.UUID
	var after string
	require.NoError(t, e.pool.QueryRow(e.ctx, `select actor_id, after::text from audit_log where entity = 'submissions' and entity_id = $1 and action = 'match'`, id.String()).Scan(&actor, &after))
	require.Equal(t, e.ta, actor)
	require.Contains(t, after, e.byCode["20229002"].String())
}

func TestManualMatchOutsideRoster(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("x\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	outsider := e.user("STUDENT", "ngoai@example.test", true)
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: outsider, Version: v})
	require.Equal(t, "STUDENT_NOT_IN_COURSE", ecode(err))
	e.exec(`update enrollments set status = 'REMOVED', removed_at = now() where user_id = $1`, e.byCode["20229003"])
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229003"], Version: v})
	require.Equal(t, "STUDENT_NOT_IN_COURSE", ecode(err), "sinh viên đã rời lớp")
}

func TestManualMatchConflict(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("x\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	in := submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229001"], Version: v + 1}
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, in)
	require.Equal(t, "VERSION_CONFLICT", ecode(err))
	in.Version = v
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, in)
	require.NoError(t, err)
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, in)
	require.Equal(t, "VERSION_CONFLICT", ecode(err), "đã gán rồi")
}

func TestReuploadAfterManualMatch(t *testing.T) {
	e := newUenv(t)
	data := []byte("Bài không MSSV\n")
	z := zipOf(t, map[string][]byte{"khong-ma.txt": data})
	_, _, err := e.upload("a.zip", z)
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	_, err = e.svc.Match(e.ctx, e.staff(), e.course, id, submission.MatchIn{AssignmentID: e.asg, StudentID: e.byCode["20229001"], Version: v})
	require.NoError(t, err)
	out, _, err := e.upload("a.zip", z)
	require.NoError(t, err)
	require.Zero(t, out.Created, "tải lại ZIP sau khi gán tay không tạo bản trùng")
	require.Equal(t, []string{"khong-ma.txt"}, out.Deduped)
	require.Equal(t, 1, e.count(`select count(*) from submissions where course_id = $1`, e.course))
}

func TestDismissUnmatched(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("x\n"))
	require.NoError(t, err)
	id, v := e.firstUnmatched()
	require.Equal(t, "VALIDATION_FAILED", ecode(e.svc.Dismiss(e.ctx, e.staff(), e.course, id, " ", v)))
	require.Equal(t, "VERSION_CONFLICT", ecode(e.svc.Dismiss(e.ctx, e.staff(), e.course, id, "Nộp nhầm lớp", v+1)))
	require.NoError(t, e.svc.Dismiss(e.ctx, e.staff(), e.course, id, "Nộp nhầm lớp", v))
	pg, err := e.svc.ListUnmatched(e.ctx, e.course, httpx.PageParams{Limit: 10})
	require.NoError(t, err)
	require.Empty(t, pg.Items)
	require.Equal(t, 1, e.count(`select count(*) from submissions where id = $1 and dismissed_at is not null and dismissed_reason = 'Nộp nhầm lớp'`, id), "không xoá bản ghi")
	require.NoError(t, e.svc.Undismiss(e.ctx, e.staff(), e.course, id))
	pg, err = e.svc.ListUnmatched(e.ctx, e.course, httpx.PageParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, pg.Items, 1, "hoàn tác")
	require.Equal(t, "CONFLICT", ecode(e.svc.Undismiss(e.ctx, e.staff(), e.course, id)))
	require.Equal(t, 2, e.count(`select count(*) from audit_log where entity_id = $1 and action in ('dismiss', 'undismiss')`, id.String()))
}

func TestDismissedNotCountedAsPending(t *testing.T) {
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("x\n"))
	require.NoError(t, err)
	n := func() int {
		rows, err := store.New(e.pool).TodayStaffAssignmentBatch(e.ctx, []string{e.course.String()})
		require.NoError(t, err)
		if len(rows) == 0 {
			return 0
		}
		return int(rows[0].Unmatched)
	}
	require.Equal(t, 1, n())
	id, v := e.firstUnmatched()
	require.NoError(t, e.svc.Dismiss(e.ctx, e.staff(), e.course, id, "Nộp nhầm lớp", v))
	require.Zero(t, n())
}

func TestSignedGetChecksRole(t *testing.T) {
	// URL ký chỉ phát qua route của nhân viên (guard StaffRole); dịch vụ trả URL attachment 5 phút cho bài của lớp, 404 cho id lạ.
	e := newUenv(t)
	_, _, err := e.upload("khong-ma.txt", []byte("x\n"))
	require.NoError(t, err)
	id, _ := e.firstUnmatched()
	u, err := e.svc.FileURL(e.ctx, e.course, id)
	require.NoError(t, err)
	require.Contains(t, u, "response-content-disposition=attachment")
	_, err = e.svc.FileURL(e.ctx, e.course, uuid.New())
	require.Equal(t, "NOT_FOUND", ecode(err))
	_, err = e.svc.FileURL(e.ctx, uuid.New(), id)
	require.Equal(t, "NOT_FOUND", ecode(err), "id của lớp khác")
}

func TestMatcherUsesEnrollmentSnapshotOnly(t *testing.T) {
	e := newUenv(t)
	// users.student_code (tự khai) trùng MSSV của một sinh viên khác nhưng KHÔNG phải snapshot ghi danh → không là tín hiệu
	declarer := e.user("STUDENT", "tukhai@example.test", true)
	e.exec(`update users set student_code = '20229001' where id = $1`, declarer)
	e.enroll(declarer, "STUDENT", "", "CODE")
	out, _, err := e.upload("20229001.txt", []byte("Bài\n"))
	require.NoError(t, err)
	require.Equal(t, 1, out.Matched)
	var student uuid.UUID
	e.one(&student, `select student_id from submissions where course_id = $1`, e.course)
	require.Equal(t, e.byCode["20229001"], student)
	require.NotEqual(t, declarer, student)
}

func TestUploadRBACService(t *testing.T) { // vai được chặn ở guard của route (xem internal/httpapi); ở đây: vé của người khác không dùng được
	e := newUenv(t)
	tk := e.ticket(e.ta, "a.txt", []byte("x\n"))
	_, _, err := e.svc.UploadComplete(e.ctx, e.teacher, e.course, e.asg, tk)
	require.Equal(t, "UPLOAD_NOT_FOUND", ecode(err))
}
