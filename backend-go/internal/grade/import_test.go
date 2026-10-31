package grade_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

// ---- hạ tầng thử ----------------------------------------------------------------------------------------------------------------------------------

type memBlob map[string][]byte

func (m memBlob) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := m[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

type impRig struct {
	*tdb
	svc    *grade.Service
	blob   memBlob
	rdb    *appredis.Client
	gv     uuid.UUID
	course uuid.UUID
	items  map[string]uuid.UUID
}

func newImpRig(t *testing.T, mod func(*grade.PutInput)) *impRig {
	t.Helper()
	d := newTDB(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := &impRig{tdb: d, blob: memBlob{}, rdb: rdb}
	r.gv = d.user("TEACHER", "GV")
	r.course = d.course(r.gv, "L1")
	doc := d.newDraft(r.course, r.gv, 1, mod)
	d.confirmDoc(r.course, r.gv, doc)
	r.items = d.itemIDs(r.course, doc.ID)
	r.svc = &grade.Service{Pool: d.pool, Redis: rdb, Blob: r.blob}
	return r
}

// rows dựng tệp XLSX từ các dòng (ô `string` giữ nguyên chuỗi, số → ô số).
func xlsx(t *testing.T, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	for i, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		require.NoError(t, f.SetSheetRow("Sheet1", cell, &row))
	}
	var b bytes.Buffer
	require.NoError(t, f.Write(&b))
	return b.Bytes()
}

// upload đặt tệp vào "object storage" và ghi vé như `presign` (user, lớp, khoá, sha256, mục đích).
func (r *impRig) upload(user, course uuid.UUID, data []byte) uuid.UUID {
	return r.uploadHash(user, course, data, "")
}

func (r *impRig) uploadHash(user, course uuid.UUID, data []byte, hash string) uuid.UUID {
	r.t.Helper()
	id := uuid.New()
	key := fmt.Sprintf("courses/%s/grade-imports/%s/f.xlsx", course, id)
	r.blob[key] = data
	if hash == "" {
		sum := sha256.Sum256(data)
		hash = hex.EncodeToString(sum[:])
	}
	require.NoError(r.t, r.rdb.HSet(r.ctx, appredis.Key("upload", id.String()), map[string]any{"user_id": user.String(), "course_id": course.String(), "blob_key": key, "sha256": hash, "purpose": "grade_import"}).Err())
	return id
}

func (r *impRig) imp(mode string, up uuid.UUID, item string, skip bool) (grade.ImportOut, error) {
	return r.svc.Import(r.ctx, r.gv, r.course, grade.ImportInput{UploadID: up, ItemID: r.items[item], Mode: mode, Mapping: grade.ImportMapping{HeaderRow: 1, StudentCodeCol: "A", ScoreCol: "C"}, SkipErrors: skip})
}

func (r *impRig) coded(name, code string) uuid.UUID { return r.student(r.course, name, code) }

// codedVia ghi danh ACTIVE bằng đường vào `via` (CODE = MSSV tự khai).
func (r *impRig) codedVia(name, code, via string) uuid.UUID {
	u := r.user("STUDENT", name)
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', $3::enrollment_joined_via, $4)`, r.course, u, via, code)
	return u
}

func (r *impRig) score(item string, st uuid.UUID) (score string, version int, ok bool) {
	n := r.countRows(`select count(*) from grade_entries where item_id = $1 and student_id = $2`, r.items[item], st)
	if n == 0 {
		return "", 0, false
	}
	r.one(&score, `select score::text from grade_entries where item_id = $1 and student_id = $2`, r.items[item], st)
	r.one(&version, `select version from grade_entries where item_id = $1 and student_id = $2`, r.items[item], st)
	return score, version, true
}

func (r *impRig) counts() (entries, audits int) {
	r.one(&entries, `select count(*) from grade_entries where course_id = $1`, r.course)
	r.one(&audits, `select count(*) from audit_log where course_id = $1`, r.course)
	return
}

func errCodes(out grade.ImportOut) map[int]string {
	m := map[int]string{}
	for _, e := range out.Errors {
		m[e.Row] = e.Code
	}
	return m
}

// ---- AC2, AC3: xem trước, dòng lỗi -----------------------------------------------------------------------------------------------------------------

func TestImportPreviewNoWrite(t *testing.T) {
	r := newImpRig(t, nil)
	a, b := r.coded("An", "20229001"), r.coded("Bình", "20229002")
	_ = b
	r.entry(r.course, r.items["weekly"], a, r.gv, "5")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "Họ tên", "Điểm"}, {"20229001", "An", "8"}, {"20229002", "Bình", "9"}}))
	e0, a0 := r.counts()
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, 2, out.RowsTotal)
	require.Equal(t, 2, out.RowsOK)
	require.Equal(t, 1, out.Overwrites)
	require.Equal(t, 0, out.Unchanged)
	require.Empty(t, out.Errors)
	require.Len(t, out.Sample, 2)
	require.Equal(t, "5", *out.Sample[0].Before)
	e1, a1 := r.counts()
	require.Equal(t, e0, e1)
	require.Equal(t, a0, a1)
}

func TestImportMappingSuggest(t *testing.T) {
	m := grade.SuggestMapping(1, []string{"STT", "Họ tên", "MSSV", "Điểm giữa kỳ"})
	require.Equal(t, grade.ImportMapping{HeaderRow: 1, StudentCodeCol: "C", ScoreCol: "D"}, m)
	m = grade.SuggestMapping(2, []string{"Mã sinh viên", "Score"})
	require.Equal(t, grade.ImportMapping{HeaderRow: 2, StudentCodeCol: "A", ScoreCol: "B"}, m)
	require.Equal(t, "", grade.SuggestMapping(1, []string{"x", "y"}).StudentCodeCol)

	r := newImpRig(t, nil)
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "Họ tên", "Điểm"}, {"20229001", "An", "8"}}))
	out, err := r.svc.Import(r.ctx, r.gv, r.course, grade.ImportInput{UploadID: up, ItemID: r.items["weekly"], Mode: "preview", Mapping: grade.ImportMapping{HeaderRow: 1}})
	require.NoError(t, err)
	require.Equal(t, []string{"MSSV", "Họ tên", "Điểm"}, out.Headers)
	require.Equal(t, grade.ImportMapping{HeaderRow: 1, StudentCodeCol: "A", ScoreCol: "C"}, out.Suggested)
	require.Zero(t, out.RowsTotal)
}

// TestImportParseErrors: mỗi lỗi đúng dòng, đúng mã; dòng trống bỏ qua; MSSV chỉ có ở sinh viên `CODE` báo UNVERIFIED.
func TestImportParseErrors(t *testing.T) {
	r := newImpRig(t, nil)
	r.coded("An", "20229001")
	r.coded("Bình", "20229002")
	r.coded("Cúc", "20229003")
	r.coded("Dũng", "20229004")
	r.coded("Em", "20229005")
	r.codedVia("Phúc", "20229777", "CODE")
	r.coded("Hai1", "20229888")
	r.coded("Hai2", "20229888")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{
		{"MSSV", "Họ tên", "Điểm"},
		{"99999999", "x", "5"},       // 2: không có trong lớp
		{"20229001", "An", "7"},      // 3: hợp lệ
		{"20229001", "An", "8"},      // 4: lặp trong tệp
		{"20229002", "Bình", "11"},   // 5: ngoài thang
		{"20229003", "Cúc", "8.333"}, // 6: quá 2 chữ số thập phân
		{"20229004", "Dũng", "abc"},  // 7: không phải số
		{nil, nil, nil},              // 8: trống
		{"20229777", "Phúc", "9"},    // 9: chỉ có ở sinh viên CODE
		{"20229888", "Hai", "6"},     // 10: hai sinh viên cùng MSSV
		{"20229005", "Em", "8,5"},    // 11: hợp lệ, dấu phẩy
	}))
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, map[int]string{2: "STUDENT_NOT_IN_COURSE", 4: "DUPLICATE_IN_FILE", 5: "SCORE_OUT_OF_RANGE", 6: "SCORE_PRECISION", 7: "SCORE_NOT_NUMBER", 9: "STUDENT_CODE_UNVERIFIED", 10: "AMBIGUOUS_STUDENT_CODE"}, errCodes(out))
	require.Equal(t, 9, out.RowsTotal) // dòng trống không tính
	require.Equal(t, 2, out.RowsOK)
	for _, e := range out.Errors {
		if e.Code == "STUDENT_CODE_UNVERIFIED" {
			require.Equal(t, "MSSV chưa có trong danh sách lớp", e.Message)
		}
		if e.Code == "SCORE_OUT_OF_RANGE" {
			require.Equal(t, "Điểm ngoài thang 0–10", e.Message)
		}
	}
}

func TestImportDecimalComma(t *testing.T) {
	r := newImpRig(t, nil)
	st := r.coded("An", "20229001")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8,5"}}))
	out, err := r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, 1, out.Written)
	sc, _, _ := r.score("weekly", st)
	require.Equal(t, "8.50", sc)
}

func TestImportPrecisionRejected(t *testing.T) {
	r := newImpRig(t, nil)
	st := r.coded("An", "20229001")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8.333"}}))
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, "SCORE_PRECISION", out.Errors[0].Code)
	_, _, ok := r.score("weekly", st)
	require.False(t, ok, "không tự làm tròn 8,333 → 8,33")
}

// TestImportFormattedCellValue: ô số định dạng `0.00` đọc theo giá trị ĐÃ ĐỊNH DẠNG (7.33); ô chung 7.333… bị từ chối (TLR-7).
func TestImportFormattedCellValue(t *testing.T) {
	r := newImpRig(t, nil)
	r.coded("An", "20229001")
	r.coded("Bình", "20229002")
	f := excelize.NewFile()
	require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]any{"MSSV", "", "Điểm"}))
	require.NoError(t, f.SetSheetRow("Sheet1", "A2", &[]any{"20229001", "", 7.3333333333}))
	require.NoError(t, f.SetSheetRow("Sheet1", "A3", &[]any{"20229002", "", 7.3333333333}))
	st, err := f.NewStyle(&excelize.Style{NumFmt: 2})
	require.NoError(t, err)
	require.NoError(t, f.SetCellStyle("Sheet1", "C2", "C2", st))
	var b bytes.Buffer
	require.NoError(t, f.Write(&b))
	out, err := r.imp("preview", r.upload(r.gv, r.course, b.Bytes()), "weekly", false)
	require.NoError(t, err)
	require.Equal(t, map[int]string{3: "SCORE_PRECISION"}, errCodes(out))
	require.Equal(t, "7.33", out.Sample[0].After)
}

// TestImportStudentCodeAsText: MSSV so CHUỖI — số 0 đầu được giữ; ô số thực "20229002.0" khớp 20229002.
func TestImportStudentCodeAsText(t *testing.T) {
	r := newImpRig(t, nil)
	zero := r.coded("Không", "00123456")
	real := r.coded("Thực", "20229002")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{
		{"MSSV", "", "Điểm"},
		{"00123456", "", "7"},
		{"123456", "", "6"}, // mất số 0 đầu → không khớp ai
		{"20229002.0", "", "9"},
	}))
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, map[int]string{3: "STUDENT_NOT_IN_COURSE"}, errCodes(out))
	require.Equal(t, 2, out.RowsOK)
	_, err = r.imp("commit", up, "weekly", true)
	require.NoError(t, err)
	a, _, _ := r.score("weekly", zero)
	b, _, _ := r.score("weekly", real)
	require.Equal(t, "7.00", a)
	require.Equal(t, "9.00", b)
}

// TestImportMatchesEnrollmentSnapshot: khớp theo `enrollments.student_code_snapshot` của dòng ROSTER / ADMIN, không theo `users.student_code`.
func TestImportMatchesEnrollmentSnapshot(t *testing.T) {
	r := newImpRig(t, nil)
	roster := r.codedVia("Roster", "20221111", "ROSTER")
	admin := r.codedVia("Admin", "20222222", "ADMIN")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20221111", "", "5"}, {"20222222", "", "6"}}))
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.Equal(t, 2, out.RowsOK)
	_, err = r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	_, _, ok1 := r.score("weekly", roster)
	_, _, ok2 := r.score("weekly", admin)
	require.True(t, ok1 && ok2)
}

// TestImportSelfDeclaredCodeIgnored (#20): sinh viên X khai MSSV của bạn Y ở hồ sơ → điểm của Y KHÔNG vào X.
func TestImportSelfDeclaredCodeIgnored(t *testing.T) {
	r := newImpRig(t, nil)
	y := r.codedVia("Y", "20225555", "ROSTER")
	x := r.coded("X", "")
	r.exec(`update users set student_code = '20225555' where id = $1`, x)
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20225555", "", "9"}}))
	_, err := r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	_, _, gotX := r.score("weekly", x)
	require.False(t, gotX)
	sy, _, gotY := r.score("weekly", y)
	require.True(t, gotY)
	require.Equal(t, "9.00", sy)
}

// TestImportCodeJoinedRowUnverified (#20): dòng chỉ khớp sinh viên vào lớp bằng mã → STUDENT_CODE_UNVERIFIED, không ghi.
func TestImportCodeJoinedRowUnverified(t *testing.T) {
	r := newImpRig(t, nil)
	z := r.codedVia("Z", "20229777", "CODE")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229777", "", "9"}}))
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, map[int]string{2: "STUDENT_CODE_UNVERIFIED"}, errCodes(out))
	_, err = r.imp("commit", up, "weekly", false)
	require.Error(t, err)
	_, err = r.imp("commit", up, "weekly", true)
	require.NoError(t, err)
	_, _, ok := r.score("weekly", z)
	require.False(t, ok)
	// nhập tay (không qua MSSV) vẫn được
	_, err = put(r.tdb, r.gv, r.course, z, r.items["weekly"], s("7"), nil)
	require.NoError(t, err)
}

// TestImportRosterWinsOverCodeRow: cùng MSSV ở dòng ROSTER và dòng CODE → chỉ dòng ROSTER được tính (không AMBIGUOUS).
func TestImportRosterWinsOverCodeRow(t *testing.T) {
	r := newImpRig(t, nil)
	rs := r.codedVia("R", "20229888", "ROSTER")
	z := r.codedVia("Z2", "20229888", "CODE")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229888", "", "8"}}))
	out, err := r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	_, _, gotR := r.score("weekly", rs)
	_, _, gotZ := r.score("weekly", z)
	require.True(t, gotR)
	require.False(t, gotZ)
}

// TestImportBlankKeepsOld: ô điểm trống không xoá điểm cũ và không phải lỗi.
func TestImportBlankKeepsOld(t *testing.T) {
	r := newImpRig(t, nil)
	st := r.coded("An", "20229001")
	r.entry(r.course, r.items["weekly"], st, r.gv, "6")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", nil}}))
	out, err := r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.Equal(t, 1, out.Unchanged)
	sc, v, _ := r.score("weekly", st)
	require.Equal(t, "6.00", sc)
	require.Equal(t, 1, v)
}

// ---- AC4: ghi ------------------------------------------------------------------------------------------------------------------------------------

func TestImportCommitOneTx(t *testing.T) {
	r := newImpRig(t, nil)
	a, b := r.coded("An", "20229001"), r.coded("Bình", "20229002")
	r.coded("Cúc", "20229003")
	r.entry(r.course, r.items["weekly"], a, r.gv, "5")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}, {"20229002", "", "9"}, {"20229003", "", "x"}}))
	// còn lỗi → tất cả hoặc không
	_, err := r.imp("commit", up, "weekly", false)
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "IMPORT_HAS_ERRORS", code)
	sa, _, _ := r.score("weekly", a)
	require.Equal(t, "5.00", sa)
	_, _, hasB := r.score("weekly", b)
	require.False(t, hasB, "không ghi dòng hợp lệ khi còn lỗi")

	up2 := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}, {"20229002", "", "9"}}))
	out, err := r.imp("commit", up2, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, 2, out.Written)
	sa, va, _ := r.score("weekly", a)
	require.Equal(t, "8.00", sa)
	require.Equal(t, 2, va, "ghi đè: version + 1")
	_, vb, _ := r.score("weekly", b)
	require.Equal(t, 1, vb)
	var src string
	r.one(&src, `select source::text from grade_entries where item_id = $1 and student_id = $2`, r.items["weekly"], b)
	require.Equal(t, "IMPORT", src)
}

func TestImportIdempotentSameFile(t *testing.T) {
	r := newImpRig(t, nil)
	a := r.coded("An", "20229001")
	data := xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}})
	out, err := r.imp("commit", r.upload(r.gv, r.course, data), "weekly", false)
	require.NoError(t, err)
	require.Equal(t, 1, out.Written)
	e0, a0 := r.counts()
	out, err = r.imp("commit", r.upload(r.gv, r.course, data), "weekly", false)
	require.NoError(t, err)
	require.Equal(t, out.RowsOK, out.Unchanged)
	require.Zero(t, out.Written)
	e1, a1 := r.counts()
	require.Equal(t, e0, e1)
	require.Equal(t, a0, a1, "không audit thừa")
	_, v, _ := r.score("weekly", a)
	require.Equal(t, 1, v, "không version mới")
}

func TestImportHasErrorsBlocks(t *testing.T) {
	r := newImpRig(t, nil)
	r.coded("An", "20229001")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}, {"11111111", "", "5"}}))
	e0, a0 := r.counts()
	_, err := r.imp("commit", up, "weekly", false)
	_, code := apiCode(t, err)
	require.Equal(t, "IMPORT_HAS_ERRORS", code)
	e1, a1 := r.counts()
	require.Equal(t, e0, e1)
	require.Equal(t, a0, a1)
}

func TestImportSkipErrors(t *testing.T) {
	r := newImpRig(t, nil)
	a := r.coded("An", "20229001")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}, {"11111111", "", "5"}}))
	out, err := r.imp("commit", up, "weekly", true)
	require.NoError(t, err)
	require.Equal(t, 1, out.Written)
	require.Equal(t, 1, out.Skipped)
	sc, _, _ := r.score("weekly", a)
	require.Equal(t, "8.00", sc)
}

func TestImportAuditPerChange(t *testing.T) {
	r := newImpRig(t, nil)
	a, b := r.coded("An", "20229001"), r.coded("Bình", "20229002")
	r.entry(r.course, r.items["weekly"], a, r.gv, "5")
	_, a0 := r.counts()
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}, {"20229002", "", "9"}}))
	_, err := r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	_, a1 := r.counts()
	require.Equal(t, a0+3, a1, "hai điểm đổi + một dòng tổng của lô")
	var before, after string
	r.one(&before, `select coalesce(before ->> 'score', '') from audit_log where entity = 'grade_entries' and after ->> 'student_id' = $1 order by created_at desc limit 1`, a.String())
	r.one(&after, `select after ->> 'score' from audit_log where entity = 'grade_entries' and after ->> 'student_id' = $1 order by created_at desc limit 1`, a.String())
	require.Equal(t, "5.00", before)
	require.Equal(t, "8.00", after)
	require.Equal(t, 1, r.countRows(`select count(*) from audit_log where entity = 'grade_imports' and action = 'import' and course_id = $1`, r.course))
	hist, err := r.svc.History(r.ctx, r.course, b, nil, nil, 10)
	require.NoError(t, err)
	require.Len(t, hist.Items, 1, "lịch sử của sinh viên có dòng nhập")
}

func TestImportLockedOrExam(t *testing.T) {
	r := newImpRig(t, nil)
	r.coded("An", "20229001")
	up := r.upload(r.gv, r.course, xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}}))
	r.exec(`update gradebooks set status = 'LOCKED', snapshot_version = 1 where course_id = $1`, r.course)
	_, err := r.imp("commit", up, "weekly", false)
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "GRADEBOOK_LOCKED", code)
	r.exec(`update gradebooks set status = 'OPEN' where course_id = $1`, r.course)
	var scheme, comp uuid.UUID
	r.one(&scheme, `select scheme_id from grade_items where id = $1`, r.items["weekly"])
	r.one(&comp, `select component_id from grade_items where id = $1`, r.items["weekly"])
	r.items["exam"] = r.examItem(r.course, scheme, comp, r.exam(r.course, r.gv, "PUBLISHED", "10"), "ex", "10")
	_, err = r.imp("preview", up, "exam", false)
	st, code = apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "ITEM_NOT_EDITABLE", code)
	// item / tệp của lớp khác
	_, err = r.svc.Import(r.ctx, r.gv, r.course, grade.ImportInput{UploadID: up, ItemID: uuid.New(), Mode: "preview", Mapping: grade.ImportMapping{HeaderRow: 1, StudentCodeCol: "A", ScoreCol: "C"}})
	st, _ = apiCode(t, err)
	require.Equal(t, 404, st)
	other := uuid.New()
	_, err = r.svc.Import(r.ctx, r.gv, r.course, grade.ImportInput{UploadID: r.upload(r.gv, other, nil), ItemID: r.items["final"], Mode: "preview", Mapping: grade.ImportMapping{HeaderRow: 1, StudentCodeCol: "A", ScoreCol: "C"}})
	st, code = apiCode(t, err)
	require.Equal(t, 404, st)
	require.Equal(t, "UPLOAD_NOT_FOUND", code)
}

// TestImportLop1MatchesExpected (AC5): inputs-lop1.csv → hai tệp XLSX (giữa kỳ, cuối kỳ) → 60 điểm đúng giá trị.
func TestImportLop1MatchesExpected(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/specs/FEAT-gradebook/fixtures/inputs-lop1.csv")
	require.NoError(t, err)
	r := newImpRig(t, func(in *grade.PutInput) {
		in.Components = []grade.PutComponent{{Key: "midterm", Name: "Giữa kỳ", Grp: "QT", Weight: "40"}, {Key: "final", Name: "Cuối kỳ", Grp: "CK", Weight: "60"}}
	})
	mid, fin := [][]any{{"MSSV", "Họ tên", "Điểm"}}, [][]any{{"MSSV", "Họ tên", "Điểm"}}
	want := map[string][2]string{}
	for i, line := range strings.Split(strings.TrimSpace(string(raw)), "\n")[1:] {
		f := strings.Split(strings.TrimSpace(line), ",")
		code := f[1]
		r.coded(fmt.Sprintf("SV %02d", i+1), code)
		mid = append(mid, []any{code, "", f[2]})
		fin = append(fin, []any{code, "", f[4]})
		want[code] = [2]string{f[2], f[4]}
	}
	for _, c := range []struct {
		item string
		rows [][]any
	}{{"midterm", mid}, {"final", fin}} {
		out, err := r.imp("commit", r.upload(r.gv, r.course, xlsx(t, c.rows)), c.item, false)
		require.NoError(t, err)
		require.Empty(t, out.Errors)
		require.Equal(t, 30, out.Written)
	}
	require.Equal(t, 60, r.countRows(`select count(*) from grade_entries where course_id = $1`, r.course))
	rows, err := r.pool.Query(r.ctx, `select e.student_code_snapshot, i.key, g.score::text from grade_entries g join grade_items i on i.id = g.item_id join enrollments e on e.course_id = g.course_id and e.user_id = g.student_id where g.course_id = $1`, r.course)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var code, key, score string
		require.NoError(t, rows.Scan(&code, &key, &score))
		exp := want[code][0]
		if key == "final" {
			exp = want[code][1]
		}
		trim := func(v string) string { return strings.TrimRight(strings.TrimRight(v, "0"), ".") }
		require.Equal(t, trim(exp), trim(score), code+" "+key)
	}
}

// ---- AC6: an toàn tệp ----------------------------------------------------------------------------------------------------------------------------

func TestImportZipBomb(t *testing.T) {
	r := newImpRig(t, nil)
	good := xlsx(t, [][]any{{"MSSV", "", "Điểm"}})
	zr, err := zip.NewReader(bytes.NewReader(good), int64(len(good)))
	require.NoError(t, err)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate})
		require.NoError(t, err)
		if f.Name == "xl/worksheets/sheet1.xml" {
			_, _ = w.Write([]byte(`<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>`))
			chunk := bytes.Repeat([]byte("a"), 1<<20)
			for range 64 {
				_, _ = w.Write(chunk)
			}
			_, _ = w.Write([]byte(`</t></is></c></row></sheetData></worksheet>`))
			continue
		}
		rc, _ := f.Open()
		_, _ = io.Copy(w, rc)
		_ = rc.Close()
	}
	require.NoError(t, zw.Close())
	require.Less(t, out.Len(), 1<<20, "tệp nén nhỏ, XML ~64 MiB")
	_, err = r.imp("preview", r.upload(r.gv, r.course, out.Bytes()), "weekly", false)
	st, code := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, "FILE_INVALID", code)
}

func TestImportTooManyRows(t *testing.T) {
	r := newImpRig(t, nil)
	many := func(n int) [][]any {
		rows := [][]any{{"MSSV", "", "Điểm"}}
		for i := range n {
			rows = append(rows, []any{fmt.Sprintf("X%07d", i), "", "5"})
		}
		return rows
	}
	out, err := r.imp("preview", r.upload(r.gv, r.course, xlsx(t, many(grade.ImportMaxRows))), "weekly", false)
	require.NoError(t, err)
	require.Equal(t, grade.ImportMaxRows, out.RowsTotal)
	_, err = r.imp("preview", r.upload(r.gv, r.course, xlsx(t, many(grade.ImportMaxRows+1))), "weekly", false)
	st, code := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, "TOO_MANY_ROWS", code)
}

func TestImportCorruptFile(t *testing.T) {
	r := newImpRig(t, nil)
	good := xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"1", "", "2"}})
	for name, data := range map[string][]byte{"cắt cụt": good[:len(good)/2], "không phải zip": []byte("khong phai xlsx"), "rỗng": {}, "zip rỗng": {0x50, 0x4b, 0x05, 0x06, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}} {
		_, err := r.imp("preview", r.upload(r.gv, r.course, data), "weekly", false)
		st, code := apiCode(t, err)
		require.Equal(t, 422, st, name)
		require.Equal(t, "FILE_INVALID", code, name)
	}
}

func TestImportSha256Mismatch(t *testing.T) {
	r := newImpRig(t, nil)
	r.coded("An", "20229001")
	data := xlsx(t, [][]any{{"MSSV", "", "Điểm"}, {"20229001", "", "8"}})
	up := r.uploadHash(r.gv, r.course, data, strings.Repeat("0", 64))
	for _, mode := range []string{"preview", "commit"} {
		_, err := r.imp(mode, up, "weekly", false)
		_, code := apiCode(t, err)
		require.Equal(t, "FILE_INVALID", code, mode)
	}
	require.Zero(t, r.countRows(`select count(*) from grade_entries where course_id = $1`, r.course))
}

// TestImportFormulaCellNotExecuted: ô công thức không bao giờ được tính; không có giá trị lưu sẵn thì coi là ô trống.
func TestImportFormulaCellNotExecuted(t *testing.T) {
	r := newImpRig(t, nil)
	st := r.coded("An", "20229001")
	f := excelize.NewFile()
	require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]any{"MSSV", "", "Điểm"}))
	require.NoError(t, f.SetSheetRow("Sheet1", "A2", &[]any{"20229001", ""}))
	require.NoError(t, f.SetCellFormula("Sheet1", "C2", "9+1"))
	var b bytes.Buffer
	require.NoError(t, f.Write(&b))
	out, err := r.imp("commit", r.upload(r.gv, r.course, b.Bytes()), "weekly", false)
	require.NoError(t, err)
	require.Zero(t, out.Written)
	_, _, ok := r.score("weekly", st)
	require.False(t, ok, "công thức 9+1 không được thực thi")
}

// TestImportNoLocalDisk (luật 10): đọc từ object storage vào bộ nhớ; mã nhập / xuất không ghi tệp.
func TestImportNoLocalDisk(t *testing.T) {
	for _, f := range []string{"import.go", "export.go"} {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, bad := range []string{"os.Create", "os.WriteFile", "os.OpenFile", "os.MkdirTemp", "os.CreateTemp", "SaveAs("} {
			require.NotContains(t, string(src), bad, f)
		}
	}
}

// ---- AC7: xuất -----------------------------------------------------------------------------------------------------------------------------------

func readSheet(t *testing.T, data []byte) [][]string {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows(f.GetSheetList()[0])
	require.NoError(t, err)
	return rows
}

func TestExportLayoutAndNote(t *testing.T) {
	r := newImpRig(t, nil)
	a := r.coded("An", "20229001")
	b := r.coded("Bình", "20229002")
	r.entry(r.course, r.items["weekly"], a, r.gv, "8")
	r.entry(r.course, r.items["final"], a, r.gv, "7")
	r.entry(r.course, r.items["weekly"], b, r.gv, "6")
	file, err := r.svc.Export(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	rows := readSheet(t, file.Bytes)
	require.Equal(t, "Bản xem trước — chưa chốt", rows[0][3])
	require.Contains(t, rows[0][1], "L1")
	require.Equal(t, []string{"MSSV", "Họ tên", "Hằng tuần", "Cuối kỳ", "Điểm cộng", "Điểm trừ", "Điểm quá trình", "Điểm thi cuối kỳ", "Điểm học phần", "Điểm chữ", "Ghi chú"}, rows[1])
	require.Equal(t, "20229001", rows[2][0])
	require.Equal(t, "8.00", rows[2][2])
	require.Contains(t, rows[3][10], "Thiếu điểm")
	require.Equal(t, "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.", rows[len(rows)-1][0])
	require.Equal(t, 2, file.Rows)
}

// TestExportFromSnapshotWhenLocked: LOCKED → số lấy từ bản chốt, không theo điểm sống đã đổi sau đó.
func TestExportFromSnapshotWhenLocked(t *testing.T) {
	r := newImpRig(t, nil)
	a := r.coded("An", "20229001")
	r.entry(r.course, r.items["weekly"], a, r.gv, "9")
	var scheme uuid.UUID
	r.one(&scheme, `select id from grade_schemes where course_id = $1 and status = 'CONFIRMED'`, r.course)
	r.exec(`insert into final_grade_snapshots (course_id, student_id, version, scheme_id, breakdown, numeric_grade, letter, locked_by)
		values ($1, $2, 1, $3, '{"items":{"weekly":"6.00","final":"5.00"},"bonus":"0.00","penalty":"0.00","qt":"6.00","ck":"5.00"}', 5.5, 'C', $4)`, r.course, a, scheme, r.gv)
	r.exec(`update gradebooks set status = 'LOCKED', snapshot_version = 1 where course_id = $1`, r.course)
	file, err := r.svc.Export(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	rows := readSheet(t, file.Bytes)
	require.Equal(t, "Bản chốt v1", rows[0][3])
	require.Equal(t, "6.00", rows[2][2], "điểm lấy từ bản chốt (6), không phải điểm sống (9)")
	require.Equal(t, "5.50", rows[2][8])
	require.Equal(t, "C", rows[2][9])
}

// TestExportFormulaInjection: ô văn bản bắt đầu `=`, `+`, `-`, `@` là chuỗi, không có thẻ <f>.
func TestExportFormulaInjection(t *testing.T) {
	r := newImpRig(t, nil)
	for i, n := range []string{"=1+1", "+84", "-2+3", "@SUM(A1)"} {
		r.coded(n, fmt.Sprintf("2022000%d", i))
	}
	file, err := r.svc.Export(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(file.Bytes), int64(len(file.Bytes)))
	require.NoError(t, err)
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") {
			rc, _ := f.Open()
			xml, _ := io.ReadAll(rc)
			_ = rc.Close()
			require.NotContains(t, string(xml), "<f>")
			require.NotContains(t, string(xml), "<f ")
		}
	}
	rows := readSheet(t, file.Bytes)
	var names []string
	for _, row := range rows[2 : len(rows)-1] {
		if len(row) > 1 {
			names = append(names, row[1])
		}
	}
	require.ElementsMatch(t, []string{"=1+1", "+84", "-2+3", "@SUM(A1)"}, names)
}

func TestExportHeadersAndAudit(t *testing.T) {
	h := newHTTPRig(t)
	var gv uuid.UUID
	course, gv, _, _, _ := h.sheet()
	req := "/api/v1/courses/" + course.String() + "/gradebook/export.xlsx"
	rec := httptest.NewRecorder()
	rq := httptest.NewRequest(http.MethodGet, req, nil)
	rq.Header.Set("Authorization", "Bearer "+h.token(gv, auth.RoleTeacher))
	h.h.ServeHTTP(rec, rq)
	w := rec
	require.Equal(t, 200, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	require.NotEmpty(t, w.Body.Bytes())
	var rowsAfter string
	h.one(&rowsAfter, `select after ->> 'rows' from audit_log where course_id = $1 and action = 'export' order by created_at desc limit 1`, course)
	require.Equal(t, "1", rowsAfter)
}

// ---- AC8: hiệu năng, không N+1 ---------------------------------------------------------------------------------------------------------------------

func (r *impRig) bulkCoded(n int) {
	r.exec(`with u as (insert into users (email, full_name, role) select 'b' || g || '.' || $2 || '@example.test', 'Sv ' || lpad(g::text, 4, '0'), 'STUDENT' from generate_series(1, $3::int) g returning id, email),
		num as (select id, row_number() over (order by email) as n from u)
		insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) select $1, id, 'STUDENT', 'ACTIVE', 'ROSTER', 'B22' || lpad(n::text, 6, '0') from num`, r.course, uuid.NewString()[:8], n)
}

func bigSheet(n int, score string) [][]any {
	rows := [][]any{{"MSSV", "", "Điểm"}}
	for i := 1; i <= n; i++ {
		rows = append(rows, []any{fmt.Sprintf("B22%06d", i), "", score})
	}
	return rows
}

func TestImport1000Under3s(t *testing.T) {
	testutil.SkipTiming(t)
	r := newImpRig(t, nil)
	r.bulkCoded(1000)
	up := r.upload(r.gv, r.course, xlsx(t, bigSheet(1000, "7.5")))
	t0 := time.Now()
	out, err := r.imp("preview", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, 1000, out.RowsOK)
	require.Less(t, time.Since(t0), 2*time.Second)
	t0 = time.Now()
	out, err = r.imp("commit", up, "weekly", false)
	require.NoError(t, err)
	require.Equal(t, 1000, out.Written)
	require.Less(t, time.Since(t0), 3*time.Second)
	require.Equal(t, 1000, r.countRows(`select count(*) from grade_entries where course_id = $1 and source = 'IMPORT'`, r.course))
}

func TestExport1000Under3s(t *testing.T) {
	testutil.SkipTiming(t)
	r := newImpRig(t, nil)
	r.bulkCoded(1000)
	t0 := time.Now()
	file, err := r.svc.Export(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	require.Equal(t, 1000, file.Rows)
	require.Less(t, time.Since(t0), 3*time.Second)
}

func TestImportNoNPlusOne(t *testing.T) {
	d, c := countingDB(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r := &impRig{tdb: d, blob: memBlob{}, rdb: rdb}
	r.gv = d.user("TEACHER", "GV")
	r.course = d.course(r.gv, "L1")
	doc := d.newDraft(r.course, r.gv, 1, nil)
	d.confirmDoc(r.course, r.gv, doc)
	r.items = d.itemIDs(r.course, doc.ID)
	r.svc = &grade.Service{Pool: d.pool, Redis: rdb, Blob: r.blob}
	r.bulkCoded(300)
	count := func(n int, score string) int64 {
		up := r.upload(r.gv, r.course, xlsx(t, bigSheet(n, score)))
		before := c.n.Load()
		_, err := r.imp("commit", up, "weekly", false)
		require.NoError(t, err)
		return c.n.Load() - before
	}
	small, big := count(10, "5"), count(300, "6")
	require.Equal(t, small, big, "cùng số câu SQL cho 10 và 300 dòng")
}

// TestImportRBAC / TestExportRBAC (AC1, AC7): chỉ TEACHER của lớp; TA / STUDENT / ADMIN / giảng viên lớp khác 403 ở cả preview và commit.
func TestImportRBAC(t *testing.T) {
	r := newRBACRig(t)
	base := "/api/v1/courses/" + r.course.String()
	body := map[string]any{"upload_id": uuid.New(), "item_id": r.weekly, "mapping": map[string]any{"header_row": 1, "student_code_col": "A", "score_col": "C"}}
	for _, mode := range []string{"preview", "commit"} {
		body["mode"] = mode
		for _, who := range []string{"TA", "SV", "ADMIN", "GV khác"} {
			st, _ := r.do(http.MethodPost, base+"/grade-entries/import", r.tok[who], "k-"+mode+who, body)
			require.Equal(t, http.StatusForbidden, st, who+" "+mode)
		}
		st, _ := r.do(http.MethodPost, base+"/grade-entries/import", "", "k", body)
		require.Equal(t, http.StatusUnauthorized, st)
	}
	// giảng viên của lớp qua được guard: vé không tồn tại → 404; `commit` thiếu Idempotency-Key bị từ chối trước khi làm gì
	body["mode"] = "preview"
	st, out := r.do(http.MethodPost, base+"/grade-entries/import", r.tok["GV"], "", body)
	require.Equal(t, http.StatusNotFound, st)
	require.Equal(t, "UPLOAD_NOT_FOUND", out["code"])
	body["mode"] = "commit"
	st, _ = r.do(http.MethodPost, base+"/grade-entries/import", r.tok["GV"], "", body)
	require.GreaterOrEqual(t, st, 400)
	require.Less(t, st, 500)
	require.NotEqual(t, http.StatusNotFound, st, "bị chặn ở Idempotency-Key, chưa tới nghiệp vụ")
}

func TestExportRBAC(t *testing.T) {
	r := newRBACRig(t)
	p := "/api/v1/courses/" + r.course.String() + "/gradebook/export.xlsx"
	for _, who := range []string{"TA", "SV", "ADMIN", "GV khác"} {
		st, _ := r.do(http.MethodGet, p, r.tok[who], "", nil)
		require.Equal(t, http.StatusForbidden, st, who)
	}
	st, _ := r.do(http.MethodGet, p, "", "", nil)
	require.Equal(t, http.StatusUnauthorized, st)
}

type presignBlob struct{ memBlob }

func (presignBlob) Stat(context.Context, string) (blob.Info, error) { return blob.Info{}, nil }
func (presignBlob) Delete(context.Context, string) error            { return nil }
func (presignBlob) PresignPutSized(_ context.Context, key, _ string, _ int64) (string, error) {
	return "http://blob.test/" + key, nil
}
func (presignBlob) PresignGetInline(context.Context, string) (string, error) { return "", nil }

// TestImportPresignValidation (AC1): chỉ .xlsx đúng MIME ≤ 2 MiB; ≤ 10 lượt / phút / người; vé của nhập điểm không hoàn tất được thành tài liệu.
func TestImportPresignValidation(t *testing.T) {
	r := newImpRig(t, nil)
	svc := &document.Service{Pool: r.pool, Redis: r.rdb, Blob: presignBlob{}, Clock: clock.Real{}}
	const mime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	hash := strings.Repeat("a", 64)
	in := func(name, mt string, size int64) document.PresignIn {
		return document.PresignIn{Purpose: "grade_import", Filename: name, MimeType: mt, SizeBytes: size, SHA256: hash}
	}
	out, err := svc.Presign(r.ctx, r.gv, r.course, in("diem.xlsx", mime, 1000))
	require.NoError(t, err)
	require.Contains(t, out.BlobKey, "/grade-imports/")
	tk, err := r.rdb.HGetAll(r.ctx, appredis.Key("upload", out.UploadID.String())).Result()
	require.NoError(t, err)
	require.Equal(t, "grade_import", tk["purpose"])

	for name, bad := range map[string]document.PresignIn{
		"đuôi pdf":    in("diem.pdf", "application/pdf", 1000),
		"đuôi csv":    in("diem.csv", "text/csv", 1000),
		"sai MIME":    in("diem.xlsx", "application/pdf", 1000),
		"đuôi xls cũ": in("diem.xls", mime, 1000),
	} {
		_, err := svc.Presign(r.ctx, r.gv, r.course, bad)
		st, code := apiCode(t, err)
		require.Equal(t, 422, st, name)
		require.Equal(t, "FILE_TYPE_NOT_ALLOWED", code, name)
	}
	_, err = svc.Presign(r.ctx, r.gv, r.course, in("diem.xlsx", mime, grade.ImportMaxBytes+1))
	st, code := apiCode(t, err)
	require.Equal(t, 422, st)
	require.Equal(t, "FILE_TOO_LARGE", code)

	// vé nhập điểm không dùng được cho `complete` (tài liệu)
	_, err = svc.Complete(r.ctx, r.gv, r.course, document.CompleteIn{UploadID: out.UploadID, Title: "x", Type: "OTHER"})
	st, code = apiCode(t, err)
	require.Equal(t, 404, st)
	require.Equal(t, "UPLOAD_NOT_FOUND", code)

	// 10 lượt / phút / người; người khác không bị ảnh hưởng
	other := r.user("TEACHER", "GV2")
	var limited bool
	for range 12 {
		if _, err := svc.Presign(r.ctx, r.gv, r.course, in("diem.xlsx", mime, 1000)); err != nil {
			st, _ = apiCode(t, err)
			limited = st == 429
		}
	}
	require.True(t, limited)
	_, err = svc.Presign(r.ctx, other, r.course, in("diem.xlsx", mime, 1000))
	require.NoError(t, err)
}

// TestImportMauFixture: `testdata/import-mau.xlsx` = giữa kỳ của 30 sinh viên lớp 1 (`inputs-lop1.csv`) + dòng lỗi cố ý. Sinh lại: UPDATE_FIXTURES=1 go test -run TestImportMauFixture.
func TestImportMauFixture(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/specs/FEAT-gradebook/fixtures/inputs-lop1.csv")
	require.NoError(t, err)
	const path = "testdata/import-mau.xlsx"
	if os.Getenv("UPDATE_FIXTURES") == "1" {
		rows := [][]any{{"MSSV", "Họ tên", "Điểm giữa kỳ"}}
		for i, line := range strings.Split(strings.TrimSpace(string(raw)), "\n")[1:] {
			f := strings.Split(strings.TrimSpace(line), ",")
			rows = append(rows, []any{f[1], fmt.Sprintf("SV %02d", i+1), f[2]})
		}
		rows = append(rows, []any{"99999999", "Không có trong lớp", "5"}, []any{"20229777", "Vào lớp bằng mã", "9"}, []any{"20229111", "Ngoài thang", "11"},
			[]any{"20229112", "Quá 2 chữ số", "8.333"}, []any{"20229113", "Chữ", "abc"}, []any{"20229001", "Lặp", "8"}, []any{nil, nil, nil})
		require.NoError(t, os.WriteFile(path, xlsx(t, rows), 0o644)) //nolint:gosec // tệp thử
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	r := newImpRig(t, func(in *grade.PutInput) {
		in.Components = []grade.PutComponent{{Key: "midterm", Name: "Giữa kỳ", Grp: "QT", Weight: "40"}, {Key: "final", Name: "Cuối kỳ", Grp: "CK", Weight: "60"}}
	})
	for i, line := range strings.Split(strings.TrimSpace(string(raw)), "\n")[1:] {
		r.codedVia(fmt.Sprintf("SV %02d", i+1), strings.Split(line, ",")[1], "ROSTER")
	}
	r.codedVia("Ngoài thang", "20229111", "ROSTER")
	r.codedVia("Quá 2 chữ số", "20229112", "ROSTER")
	r.codedVia("Chữ", "20229113", "ROSTER")
	r.codedVia("Vào lớp bằng mã", "20229777", "CODE")
	out, err := r.imp("preview", r.upload(r.gv, r.course, data), "midterm", false)
	require.NoError(t, err)
	require.Equal(t, 36, out.RowsTotal)
	require.Equal(t, 30, out.RowsOK)
	require.Equal(t, map[int]string{32: "STUDENT_NOT_IN_COURSE", 33: "STUDENT_CODE_UNVERIFIED", 34: "SCORE_OUT_OF_RANGE", 35: "SCORE_PRECISION", 36: "SCORE_NOT_NUMBER", 37: "DUPLICATE_IN_FILE"}, errCodes(out))
}
