package practice_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/practice"
	"github.com/edupilot/backend-go/internal/testutil"
)

type memBlob map[string][]byte

func (m memBlob) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := m[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// formsRig: lớp có công thức ĐÃ XÁC NHẬN + một cột điểm MANUAL (thang 10), Redis + "object storage" bộ nhớ.
type formsRig struct {
	*rig
	blob memBlob
	rdb  *appredis.Client
	item uuid.UUID
	qs   []uuid.UUID // ba câu MCQ đã duyệt, đáp án đúng "B"
}

func newFormsRig(t *testing.T) *formsRig {
	t.Helper()
	r := newRig(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	f := &formsRig{rig: r, blob: memBlob{}, rdb: rdb}
	r.svc.Redis, r.svc.Blob = rdb, f.blob
	var scheme, comp uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 1, $2) returning id`, r.course, r.teacher).Scan(&scheme))
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ($1, $2, 'weekly', 'Kiểm tra hằng tuần', 'QT', 20) returning id`, r.course, scheme).Scan(&comp))
	r.exec(`update grade_schemes set status = 'CONFIRMED', confirmed_by = $2, confirmed_at = now() where id = $1`, scheme, r.teacher)
	r.exec(`insert into gradebooks (course_id) values ($1)`, r.course)
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into grade_items (course_id, scheme_id, component_id, key, name, max_score) values ($1, $2, $3, 'forms1', 'Forms 1', 10) returning id`, r.course, scheme, comp).Scan(&f.item))
	f.qs = r.approved("Mật mã", 3)
	return f
}

// who ghi danh một sinh viên: email (đã / chưa xác minh), MSSV chụp, cách vào lớp.
func (f *formsRig) who(name, email string, verified bool, code, via string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `insert into users (email, full_name, role, email_verified_at) values ($1, $2, 'STUDENT', case when $3 then now() end) returning id`, email, name, verified).Scan(&id))
	f.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', $3::enrollment_joined_via, nullif($4, ''))`, f.course, id, via, code)
	return id
}

// upload đặt tệp vào kho và ghi vé như `presign`.
func (f *formsRig) upload(user uuid.UUID, name string, data []byte) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	key := fmt.Sprintf("courses/%s/forms-imports/%s/%s", f.course, id, name)
	f.blob[key] = data
	sum := sha256.Sum256(data)
	require.NoError(f.t, f.rdb.HSet(f.t.Context(), appredis.Key("upload", id.String()), map[string]any{"user_id": user.String(), "course_id": f.course.String(), "filename": name, "blob_key": key, "sha256": hex.EncodeToString(sum[:]), "purpose": "forms_import"}).Err())
	return id
}

func csvOf(rows ...string) []byte { return []byte(strings.Join(rows, "\n") + "\n") }

func (f *formsRig) in(up uuid.UUID, qs ...string) practice.FormsIn {
	m := practice.FormsMapping{HeaderRow: 1, EmailCol: "A", CodeCol: "B", TimeCol: "C"}
	for i, id := range f.qs {
		col, _ := excelize.ColumnNumberToName(4 + i)
		m.Questions = append(m.Questions, practice.FormsQuestion{Col: col, QuestionID: &id})
	}
	item := f.item
	return practice.FormsIn{UploadID: up, ItemID: &item, AnswerSource: "BANK", Mapping: &m}
}

func (f *formsRig) entry(st uuid.UUID) (score, source string, published bool, ok bool) {
	f.t.Helper()
	err := f.pool.QueryRow(f.t.Context(), `select e.score::text, e.source::text, i.published_at is not null from grade_entries e join grade_items i on i.id = e.item_id where e.item_id = $1 and e.student_id = $2`, f.item, st).Scan(&score, &source, &published)
	return score, source, published, err == nil
}

const formsHeader = "Email,MSSV,Completion time,Q1,Q2,Q3"

func TestFormsImportFlow(t *testing.T) {
	t.Parallel()
	f := newFormsRig(t)
	s1 := f.who("An", "an@example.test", true, "20220001", "ROSTER")
	s2 := f.who("Bình", "binh@example.test", false, "20220002", "ROSTER") // email chưa xác minh → khớp bằng MSSV
	data := csvOf(formsHeader,
		"an@example.test,20220001,2026-11-01 10:00:00,B,B,A",     // 2/3
		"an@example.test,20220001,2026-11-01 11:00:00,B,B,B",     // mới hơn: 3/3 thắng
		"binh@example.test,20220002,2026-11-01 10:30:00,B,B,B",   // 3/3
		"nobody@example.test,99999999,2026-11-01 10:40:00,A,A,A", // chưa khớp
	)
	up := f.upload(f.teacher, "forms.csv", data)

	// chưa xem trước → 409
	_, err := f.svc.Import(t.Context(), f.teacher, f.course, f.in(up))
	require.Equal(t, apierr.PreviewRequired, code(t, err))

	pv, err := f.svc.Preview(t.Context(), f.teacher, f.course, practice.FormsIn{UploadID: up})
	require.NoError(t, err)
	require.Len(t, pv.Columns, 6)
	require.Equal(t, 4, pv.RowsTotal)
	require.Equal(t, "A", pv.Suggested.EmailCol)
	require.Equal(t, "B", pv.Suggested.CodeCol)
	require.Equal(t, "C", pv.Suggested.TimeCol)
	_, _, _, written := f.entry(s1)
	require.False(t, written, "xem trước không ghi")

	pv, err = f.svc.Preview(t.Context(), f.teacher, f.course, f.in(up))
	require.NoError(t, err)
	require.Equal(t, 2, pv.Result.Matched)
	require.Equal(t, 1, pv.Result.Unmatched)
	require.Equal(t, 1, pv.Result.Duplicates)
	_, _, _, written = f.entry(s1)
	require.False(t, written, "xem trước có bảng điểm vẫn không ghi")

	res, err := f.svc.Import(t.Context(), f.teacher, f.course, f.in(up))
	require.NoError(t, err)
	require.False(t, res.Replayed)
	require.Equal(t, 2, res.Written)
	require.Len(t, res.Rows, 1)
	require.Equal(t, "NOT_FOUND", res.Rows[0].Reason)
	require.Contains(t, res.Rows[0].Identifier, "***") // định danh chưa khớp được che
	require.NotContains(t, res.Rows[0].Identifier, "nobody")
	for _, st := range []uuid.UUID{s1, s2} {
		score, source, published, ok := f.entry(st)
		require.True(t, ok)
		require.Equal(t, "10.00", score) // 3/3 × 10
		require.Equal(t, "QUIZ", source)
		require.False(t, published, "không tự công bố")
	}

	// cùng tệp (cùng hash), vé mới, khoá idempotency mới → lô cũ, không đổi điểm, không audit thừa
	var before, after int
	f.exec(`update grade_entries set score = 9 where item_id = $1 and student_id = $2`, f.item, s1) // giảng viên sửa tay sau khi nhập
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from audit_log where entity = 'forms_imports'`).Scan(&before))
	again, err := f.svc.Import(t.Context(), f.teacher, f.course, f.in(f.preview(t, data)))
	require.NoError(t, err)
	require.True(t, again.Replayed)
	require.Equal(t, 2, again.Matched)
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from audit_log where entity = 'forms_imports'`).Scan(&after))
	require.Equal(t, before, after)
	score, _, _, _ := f.entry(s1)
	require.Equal(t, "9.00", score, "nhập lại cùng tệp không ghi đè điểm đã sửa")
}

func (f *formsRig) preview(t *testing.T, data []byte) uuid.UUID {
	t.Helper()
	up := f.upload(f.teacher, "forms.csv", data)
	_, err := f.svc.Preview(t.Context(), f.teacher, f.course, practice.FormsIn{UploadID: up})
	require.NoError(t, err)
	return up
}

func TestFormsMatchRules(t *testing.T) {
	t.Parallel()
	f := newFormsRig(t)
	f.who("Có xác minh", "ok@example.test", true, "", "CODE")
	f.who("Chưa xác minh", "unv@example.test", false, "", "CODE")
	f.who("MSSV tự khai", "self@example.test", true, "20220003", "CODE") // email đã xác minh nên vẫn khớp bằng email
	f.who("Chỉ MSSV tự khai", "self2@example.test", false, "20220004", "CODE")
	a := f.who("Người A", "a@example.test", true, "20220005", "ROSTER")
	f.who("Người B", "b@example.test", true, "20220006", "ROSTER")
	_ = a
	data := csvOf(formsHeader,
		"ok@example.test,,2026-11-01 10:00:00,B,B,B",          // khớp (email xác minh)
		"unv@example.test,,2026-11-01 10:00:00,B,B,B",         // email chưa xác minh, không MSSV → chưa khớp
		"self@example.test,,2026-11-01 10:00:00,B,B,B",        // khớp bằng email xác minh
		",20220004,2026-11-01 10:00:00,B,B,B",                 // chỉ MSSV tự khai → chưa khớp (không mở dữ liệu)
		"a@example.test,20220006,2026-11-01 10:00:00,B,B,B",   // email của A, MSSV của B → MISMATCH
		"b@example.test,20220006.0,2026-11-01 10:00:00,B,B,B", // MSSV số thực "….0" vẫn khớp B
	)
	pv, err := f.svc.Preview(t.Context(), f.teacher, f.course, f.in(f.upload(f.teacher, "x.csv", data)))
	require.NoError(t, err)
	require.Equal(t, 3, pv.Result.Matched)
	reasons := map[int]string{}
	for _, u := range pv.Result.Rows {
		reasons[u.Row] = u.Reason
	}
	require.Equal(t, map[int]string{3: "NOT_FOUND", 5: "NOT_FOUND", 6: "MISMATCH"}, reasons)
}

// TestFormsImportScoreDecimal: điểm = đúng × thang ÷ tổng bằng decimal, làm tròn 2 chữ số; đáp án lạ → 0 và báo "không khớp", không đoán.
func TestFormsImportScoreDecimal(t *testing.T) {
	t.Parallel()
	f := newFormsRig(t)
	f.who("An", "an@example.test", true, "", "ROSTER")
	f.who("Bình", "binh@example.test", true, "", "ROSTER")
	data := csvOf(formsHeader,
		"an@example.test,,2026-11-01 10:00:00,B,A,A",          // 1/3
		"binh@example.test,,2026-11-01 10:00:00,B,B,Không có", // 2/3, một ô lạ
	)
	pv, err := f.svc.Preview(t.Context(), f.teacher, f.course, f.in(f.upload(f.teacher, "x.csv", data)))
	require.NoError(t, err)
	got := map[string]string{}
	mismatch := 0
	for _, it := range pv.Result.Items {
		got[it.Student.FullName] = it.Score
		mismatch += it.Mismatch
	}
	require.Equal(t, map[string]string{"An": "3.33", "Bình": "6.67"}, got)
	require.Equal(t, 1, mismatch)
}

func TestFormsManualAnswerAndSemicolonCSV(t *testing.T) {
	t.Parallel()
	f := newFormsRig(t)
	f.who("An", "an@example.test", true, "", "ROSTER")
	// Excel tiếng Việt: dấu chấm phẩy + BOM
	data := append([]byte("\xef\xbb\xbf"), csvOf("Email;Q1;Q2", "an@example.test;Hà Nội.;sai")...)
	m := practice.FormsMapping{HeaderRow: 1, EmailCol: "A", Questions: []practice.FormsQuestion{
		{Col: "B", Accepted: []string{"hà nội", "Thủ đô"}}, {Col: "C", Accepted: []string{"đúng"}}}}
	item := f.item
	pv, err := f.svc.Preview(t.Context(), f.teacher, f.course, practice.FormsIn{UploadID: f.upload(f.teacher, "x.csv", data), ItemID: &item, AnswerSource: "MANUAL", Mapping: &m})
	require.NoError(t, err)
	require.Equal(t, "5", pv.Result.Items[0].Score) // 1/2 × 10
	require.Equal(t, 1, pv.Result.Items[0].Correct)
}

func TestFormsImportGuards(t *testing.T) {
	t.Parallel()
	f := newFormsRig(t)
	f.who("An", "an@example.test", true, "", "ROSTER")
	data := csvOf(formsHeader, "an@example.test,,2026-11-01 10:00:00,B,B,B")

	// vé của người khác / lớp khác / mục đích khác → 404
	other := f.upload(uuid.New(), "x.csv", data)
	_, err := f.svc.Preview(t.Context(), f.teacher, f.course, practice.FormsIn{UploadID: other})
	require.Equal(t, apierr.UploadNotFound, code(t, err))
	require.NoError(t, f.rdb.HSet(t.Context(), appredis.Key("upload", other.String()), "user_id", f.teacher.String(), "purpose", "grade_import").Err())
	_, err = f.svc.Preview(t.Context(), f.teacher, f.course, practice.FormsIn{UploadID: other})
	require.Equal(t, apierr.UploadNotFound, code(t, err))

	// tệp hỏng → 422 FILE_INVALID; thiếu cột định danh → 422
	_, err = f.svc.Preview(t.Context(), f.teacher, f.course, practice.FormsIn{UploadID: f.upload(f.teacher, "x.xlsx", []byte("không phải xlsx"))})
	require.Equal(t, apierr.FileInvalid, code(t, err))
	up := f.upload(f.teacher, "ok.csv", data)
	in := f.in(up)
	in.Mapping.EmailCol, in.Mapping.CodeCol = "", ""
	_, err = f.svc.Preview(t.Context(), f.teacher, f.course, in)
	require.Equal(t, 422, status(t, err))

	// câu tự luận không chấm được → 422
	var essay uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `select id from question_bank where course_id = $1 limit 1`, f.course).Scan(&essay))
	f.exec(`update question_bank set type = 'ESSAY' where id = $1`, essay)
	in = f.in(up)
	in.Mapping.Questions[0].QuestionID = &essay
	_, err = f.svc.Preview(t.Context(), f.teacher, f.course, in)
	require.Equal(t, 422, status(t, err))
	f.exec(`update question_bank set type = 'MCQ_SINGLE' where id = $1`, essay)

	// sổ điểm LOCKED → 409; cột không phải MANUAL → 409
	up = f.preview(t, data)
	f.exec(`update gradebooks set status = 'LOCKED', snapshot_version = 1 where course_id = $1`, f.course)
	_, err = f.svc.Import(t.Context(), f.teacher, f.course, f.in(up))
	require.Equal(t, apierr.GradebookLocked, code(t, err))
	f.exec(`update gradebooks set status = 'OPEN' where course_id = $1`, f.course)
	gone := uuid.New() // cột điểm không thuộc lớp → 404
	in = f.in(up)
	in.ItemID = &gone
	_, err = f.svc.Import(t.Context(), f.teacher, f.course, in)
	require.Equal(t, 404, status(t, err))
}

// TestFormsImportNoLLM (US-P9-11 AC12): Service không có LLM nào được cấu hình (LLM nil) và cả luồng xem trước + ghi vẫn chạy; `llm_audit` không có dòng nào.
func TestFormsImportNoLLM(t *testing.T) {
	t.Parallel()
	f := newFormsRig(t)
	require.Nil(t, f.svc.LLM)
	f.who("An", "an@example.test", true, "", "ROSTER")
	up := f.preview(t, csvOf(formsHeader, "an@example.test,,2026-11-01 10:00:00,B,B,B"))
	_, err := f.svc.Import(t.Context(), f.teacher, f.course, f.in(up))
	require.NoError(t, err)
	var n int
	require.NoError(t, f.pool.QueryRow(t.Context(), `select count(*) from llm_audit`).Scan(&n))
	require.Zero(t, n)
}
