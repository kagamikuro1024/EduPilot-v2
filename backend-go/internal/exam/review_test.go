package exam_test

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
)

func (r *rig) review(qid uuid.UUID, decision string, by uuid.UUID) (exam.QuestionDetail, error) {
	return r.svc.Review(r.t.Context(), by, r.course, qid, decision, r.version(qid))
}

// TestReviewApprove — AC7: DRAFT → REQUEST → PENDING → APPROVE (TA hoặc Giảng viên đều duyệt được) / REJECT; chuyển trạng thái sai → 409; sai version → VersionConflict.
func TestReviewApprove(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.mcq("duyệt")
	d, err := r.review(q.ID, "REQUEST", r.teacher)
	require.NoError(t, err)
	require.Equal(t, "PENDING", d.ReviewStatus)
	require.Nil(t, d.ReviewedBy)
	_, err = r.review(q.ID, "REQUEST", r.teacher)
	st, _ := apiStatus(t, err)
	require.Equal(t, 409, st, "PENDING không gửi duyệt lại")
	d, err = r.review(q.ID, "APPROVE", r.ta)
	require.NoError(t, err)
	require.Equal(t, "APPROVED", d.ReviewStatus)
	require.Equal(t, r.ta, *d.ReviewedBy)
	require.NotNil(t, d.ReviewedAt)
	_, err = r.review(q.ID, "APPROVE", r.teacher)
	st, _ = apiStatus(t, err)
	require.Equal(t, 409, st, "đã APPROVED")
	// từ chối
	q2 := r.mcq("từ chối")
	d, err = r.review(q2.ID, "REJECT", r.teacher)
	require.NoError(t, err)
	require.Equal(t, "REJECTED", d.ReviewStatus)
	require.Equal(t, r.teacher, *d.ReviewedBy)
	// duyệt thẳng từ DRAFT được (Giảng viên tự soạn tự duyệt)
	q3 := r.mcq("thẳng")
	d, err = r.review(q3.ID, "APPROVE", r.teacher)
	require.NoError(t, err)
	require.Equal(t, "APPROVED", d.ReviewStatus)
	// decision lạ → 422; sai version → conflict kèm bản hiện tại
	_, err = r.svc.Review(t.Context(), r.teacher, r.course, q3.ID, "BAN", 1)
	require.Contains(t, fieldCodes(t, err), "decision")
	_, err = r.svc.Review(t.Context(), r.teacher, r.course, q3.ID, "REJECT", 1)
	var vc *exam.VersionConflict
	require.True(t, errors.As(err, &vc))
	require.Equal(t, 2, vc.Version)
	// sửa nội dung câu đã duyệt → về DRAFT, gỡ người duyệt
	cur, _ := r.svc.Get(t.Context(), r.course, q3.ID)
	in := exam.QuestionIn{Type: "MCQ_SINGLE", Title: "thẳng 2", Topic: "Số học", Difficulty: "EASY", Stem: "2+2=?", Options: opts("3", "4"), Correct: []int{1}}
	d, err = r.svc.Update(t.Context(), r.course, q3.ID, in, cur.Version)
	require.NoError(t, err)
	require.Equal(t, "DRAFT", d.ReviewStatus)
	require.Nil(t, d.ReviewedBy)
}

// TestReviewCodeNeedsVerifiedReference — AC7: câu CODE chỉ duyệt khi ≥ 1 test mẫu + ≥ 1 test ẩn đã duyệt, Σweight > 0 và lời giải mẫu đã kiểm đúng phiên bản test hiện hành.
func TestReviewCodeNeedsVerifiedReference(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("code duyệt")
	approve := func() error { _, err := r.review(q.ID, "APPROVE", r.teacher); return err }
	has := func(err error, code string) { t.Helper(); require.Contains(t, fieldCodes(t, err), code) }
	err := approve()
	has(err, "CODE_TESTS_MISSING")
	has(err, "REFERENCE_NOT_VERIFIED")
	smp := r.test(q.ID, "s", "1", "1", true, 0)
	has(approve(), "CODE_TESTS_MISSING") // thiếu test ẩn
	hid := r.test(q.ID, "h", "2", "2", false, 0)
	err = approve()
	has(err, "TOTAL_WEIGHT_ZERO")
	_, err = r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, hid.ID, exam.TestcaseIn{Weight: new(2)}, true)
	require.NoError(t, err)
	has(approve(), "REFERENCE_NOT_VERIFIED")
	r.verify(q.ID)
	require.NoError(t, approve())
	// test AI chưa duyệt không tính vào "test ẩn"
	r.exec(`update code_testcases set approved=false where id=$1`, hid.ID)
	r.exec(`update question_bank set review_status='DRAFT', reviewed_by=null, reviewed_at=null where id=$1`, q.ID)
	has(approve(), "CODE_TESTS_MISSING")
	r.exec(`update code_testcases set approved=true where id=$1`, hid.ID)
	// đổi test sau khi duyệt → câu về DRAFT và cờ lời giải mẫu mất
	require.NoError(t, approve())
	_, err = r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, smp.ID, exam.TestcaseIn{Input: new("9")}, true)
	require.NoError(t, err)
	d, _ := r.svc.Get(t.Context(), r.course, q.ID)
	require.Equal(t, "DRAFT", d.ReviewStatus)
	require.Nil(t, d.Code.ReferenceVerifiedVersion)
	has(approve(), "REFERENCE_NOT_VERIFIED")
}

// TestAIDraftNeverAutoApproved — AC7: DB chặn APPROVED / REJECTED mà không có reviewed_by (kể cả câu AI_DRAFT); duyệt qua API luôn ghi reviewed_by.
func TestAIDraftNeverAutoApproved(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.pool.Exec(t.Context(), `insert into question_bank (course_id, type, title, topic, stem, answer_key, origin, review_status, created_by)
		values ($1, 'TRUE_FALSE', 'ai', 't', 's', '{"value":true}', 'AI_DRAFT', 'APPROVED', $2)`, r.course, r.teacher)
	var pe *pgconn.PgError
	require.True(t, errors.As(err, &pe))
	require.Equal(t, "23514", pe.Code)
	r.exec(`insert into question_bank (course_id, type, title, topic, stem, answer_key, origin, review_status, created_by)
		values ($1, 'TRUE_FALSE', 'ai', 't', 's', '{"value":true}', 'AI_DRAFT', 'PENDING', $2)`, r.course, r.teacher)
	var id uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id from question_bank where course_id=$1 and origin='AI_DRAFT'`, r.course).Scan(&id))
	d, err := r.review(id, "APPROVE", r.ta)
	require.NoError(t, err)
	require.Equal(t, r.ta, *d.ReviewedBy)
}

// TestReviewAudit — AC7 + PE-01 AC11: mỗi thao tác quan trọng đúng MỘT dòng audit_log (question.review / question.archive / testcases.change), cùng transaction, before / after chỉ có trường nhỏ — không đề, đáp án, test, mã.
func TestReviewAudit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("audit")
	secret := "BÍ-MẬT-ĐỀ-BÀI"
	_, err := r.svc.PutCode(t.Context(), r.course, q.ID, exam.CodeIn{Languages: []string{"cpp17"}, Reference: &exam.Reference{Language: "cpp17", Source: "// " + secret}}, q.Version)
	require.NoError(t, err)
	tc := r.test(q.ID, "t", secret, secret, true, 1)
	_, err = r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, tc.ID, exam.TestcaseIn{Weight: new(2)}, true)
	require.NoError(t, err)
	_, err = r.review(q.ID, "REJECT", r.teacher)
	require.NoError(t, err)
	_, err = r.svc.Archive(t.Context(), r.teacher, r.course, q.ID)
	require.NoError(t, err)
	rows, err := r.pool.Query(t.Context(), `select action, actor_id, before::text, after::text from audit_log where course_id=$1 and entity_id=$2 order by id`, r.course, q.ID.String())
	require.NoError(t, err)
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var a, b, af string
		var actor uuid.UUID
		require.NoError(t, rows.Scan(&a, &actor, &b, &af))
		actions = append(actions, a)
		require.Equal(t, r.teacher, actor)
		for _, txt := range []string{b, af} {
			require.NotContains(t, txt, secret)
			var m map[string]any
			require.NoError(t, json.Unmarshal([]byte(txt), &m))
			for k := range m {
				require.Contains(t, []string{"review_status", "version", "tests_version", "changed", "archived"}, k, "trường nhỏ")
			}
		}
	}
	// thêm test, sửa test, duyệt, lưu trữ: 2 dòng testcases.change (thêm + sửa), 1 review, 1 archive
	require.Equal(t, []string{"testcases.change", "testcases.change", "question.review", "question.archive"}, actions)
}

// TestDuplicateQuestion — AC12: bản sao DRAFT cùng nội dung (kể cả test và lời giải mẫu), tests_version 1, chưa kiểm, origin MANUAL, tiêu đề + " (bản sao)", id đáp án mới nhưng đáp án đúng vẫn đúng.
func TestDuplicateQuestion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mcq("gốc")
	_, err := r.review(m.ID, "APPROVE", r.teacher)
	require.NoError(t, err)
	d, err := r.svc.Duplicate(t.Context(), r.teacher, r.course, m.ID)
	require.NoError(t, err)
	require.Equal(t, "gốc (bản sao)", d.Title)
	require.Equal(t, "DRAFT", d.ReviewStatus)
	require.Equal(t, "MANUAL", d.Origin)
	require.Nil(t, d.ReviewedBy)
	require.NotEqual(t, m.ID, d.ID)
	require.Len(t, d.Options, 2)
	require.NotEqual(t, m.Options[0].ID, d.Options[0].ID)
	var key struct {
		OptionIDs []uuid.UUID `json:"option_ids"`
	}
	require.NoError(t, json.Unmarshal(d.AnswerKey, &key))
	require.Equal(t, []uuid.UUID{d.Options[1].ID}, key.OptionIDs, "đáp án đúng trỏ đúng id mới")
	// tiêu đề dài bị cắt cho vừa 120
	long := r.mcq(strings.Repeat("x", 120))
	d, err = r.svc.Duplicate(t.Context(), r.teacher, r.course, long.ID)
	require.NoError(t, err)
	require.Len(t, []rune(d.Title), 120)
	// bài code: sao cả test + lời giải mẫu, đặt lại phiên bản
	c := r.code("code gốc")
	r.test(c.ID, "s", "1", "1", true, 1)
	r.test(c.ID, "h", "2", "2", false, 2)
	r.verify(c.ID)
	d, err = r.svc.Duplicate(t.Context(), r.teacher, r.course, c.ID)
	require.NoError(t, err)
	require.Equal(t, 1, d.Code.TestsVersion)
	require.Nil(t, d.Code.ReferenceVerifiedVersion)
	require.NotNil(t, d.Code.Reference)
	require.Equal(t, 2, d.Code.Tests.Total)
	require.Equal(t, 3, d.Code.Tests.TotalWeight)
	require.Equal(t, "h", r.tests(d.ID)[1].Name)
}

// TestVersionConflict — PE-01 AC9: hai yêu cầu sửa đồng thời cùng version → đúng MỘT thành công, bên kia nhận VersionConflict kèm bản hiện tại.
func TestVersionConflict(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.mcq("đồng thời")
	var wg sync.WaitGroup
	errs := make([]error, 6)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := exam.QuestionIn{Type: "MCQ_SINGLE", Title: "đồng thời " + strings.Repeat("x", i), Topic: "Số học", Difficulty: "EASY", Stem: "2+2=?", Options: opts("3", "4"), Correct: []int{1}}
			_, errs[i] = r.svc.Update(t.Context(), r.course, q.ID, in, q.Version)
		}()
	}
	wg.Wait()
	ok := 0
	for _, e := range errs {
		var vc *exam.VersionConflict
		switch {
		case e == nil:
			ok++
		case errors.As(e, &vc):
			require.Equal(t, 2, vc.Version)
			require.NotNil(t, vc.Current)
		default:
			t.Fatalf("lỗi không mong đợi: %v", e)
		}
	}
	require.Equal(t, 1, ok)
}

// TestListCursor — PE-01 AC10: danh sách câu phân trang con trỏ xác định (created_at, id), không trùng không sót, không OFFSET.
func TestListCursor(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for i := 0; i < 7; i++ {
		r.mcq("câu " + string(rune('A'+i)))
	}
	var seen []uuid.UUID
	var cur *exam.Cursor
	for page := 0; page < 10; page++ {
		rows, err := r.svc.List(t.Context(), r.course, exam.ListFilter{}, cur, 4) // limit 3 + 1
		require.NoError(t, err)
		n := min(len(rows), 3)
		for _, x := range rows[:n] {
			seen = append(seen, x.ID)
		}
		if len(rows) <= 3 {
			break
		}
		last := rows[2]
		cur = &exam.Cursor{At: last.CreatedAt, ID: last.ID}
	}
	require.Len(t, seen, 7)
	uniq := map[uuid.UUID]bool{}
	for _, id := range seen {
		uniq[id] = true
	}
	require.Len(t, uniq, 7, "không trùng")
	all, err := r.svc.List(t.Context(), r.course, exam.ListFilter{}, nil, 100)
	require.NoError(t, err)
	for i, x := range all {
		require.Equal(t, x.ID, seen[i], "thứ tự giống nhau giữa các lần gọi")
	}
}

// TestQuestionListFilters — AC11: lọc theo trạng thái, chủ đề, độ khó, loại, nguồn, từ khoá (không phân biệt hoa thường) và `archived`.
func TestQuestionListFilters(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a := r.mcq("Mật mã AES")
	r.mcq("Đường đi ngắn nhất")
	c := r.code("Cộng hai số")
	_, err := r.review(a.ID, "REQUEST", r.teacher)
	require.NoError(t, err)
	r.exec(`update question_bank set topic='Đồ thị', difficulty='HARD', origin='AI_DRAFT', review_status='PENDING' where id=$1`, c.ID)
	r.exec(`update question_bank set review_status='PENDING' where id=$1 and false`, c.ID)
	_, err = r.svc.Archive(t.Context(), r.teacher, r.course, r.mcq("đã lưu trữ").ID)
	require.NoError(t, err)
	n := func(f exam.ListFilter) int {
		rows, err := r.svc.List(t.Context(), r.course, f, nil, 100)
		require.NoError(t, err)
		return len(rows)
	}
	require.Equal(t, 3, n(exam.ListFilter{}), "mặc định ẩn câu đã lưu trữ")
	require.Equal(t, 4, n(exam.ListFilter{Archived: true}))
	require.Equal(t, 2, n(exam.ListFilter{ReviewStatus: "PENDING"}))
	require.Equal(t, 1, n(exam.ListFilter{Topic: "Đồ thị"}))
	require.Equal(t, 1, n(exam.ListFilter{Difficulty: "HARD"}))
	require.Equal(t, 1, n(exam.ListFilter{Type: "CODE"}))
	require.Equal(t, 1, n(exam.ListFilter{Origin: "AI_DRAFT"}))
	require.Equal(t, 1, n(exam.ListFilter{Q: "mật MÃ"}))
	require.Equal(t, 0, n(exam.ListFilter{Q: "không có"}))
	require.Equal(t, 1, n(exam.ListFilter{Type: "CODE", Origin: "AI_DRAFT", ReviewStatus: "PENDING"}))
}

// TestQuestionDetailStaffOnly — AC11: chi tiết đầy đủ (đáp án, giải thích, test ẩn, lời giải mẫu) có ở service cho Staff; câu của LỚP KHÁC → 404 ở mọi thao tác (cách ly lớp, PE-01 AC5).
func TestQuestionDetailStaffOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	other := newRig(t) // lớp khác
	q := r.code("lớp A")
	r.test(q.ID, "h", "5", "5", false, 1)
	d, err := r.svc.Get(t.Context(), r.course, q.ID)
	require.NoError(t, err)
	require.NotNil(t, d.Code.Reference)
	require.Equal(t, 1, d.Code.Tests.Hidden)
	// lớp khác dùng id của lớp này
	nf := func(err error) {
		t.Helper()
		st, code := apiStatus(t, err)
		require.Equal(t, 404, st)
		require.Equal(t, "NOT_FOUND", code)
	}
	_, err = other.svc.Get(t.Context(), other.course, q.ID)
	nf(err)
	_, err = other.svc.Update(t.Context(), other.course, q.ID, exam.QuestionIn{Type: "CODE", Title: "x", Topic: "t", Stem: "s"}, 1)
	nf(err)
	_, err = other.svc.Review(t.Context(), other.teacher, other.course, q.ID, "APPROVE", 1)
	nf(err)
	_, err = other.svc.Archive(t.Context(), other.teacher, other.course, q.ID)
	nf(err)
	_, err = other.svc.Duplicate(t.Context(), other.teacher, other.course, q.ID)
	nf(err)
	_, err = other.svc.PutCode(t.Context(), other.course, q.ID, exam.CodeIn{Languages: []string{"c11"}}, 1)
	nf(err)
	_, err = other.svc.ListTests(t.Context(), other.course, q.ID, nil, 10)
	nf(err)
	_, err = other.svc.AddTest(t.Context(), other.teacher, other.course, q.ID, exam.TestcaseIn{Name: new("x"), Input: new("1"), Expected: new("1")}, true)
	nf(err)
	_, err = other.svc.EnqueueVerify(t.Context(), other.teacher, other.course, q.ID)
	nf(err)
	rows, err := other.svc.List(t.Context(), other.course, exam.ListFilter{}, nil, 100)
	require.NoError(t, err)
	require.Empty(t, rows)
}

// TestArchivedCourseWrites — PE-01 AC12: lớp ARCHIVED → mọi thao tác ghi 409 COURSE_ARCHIVED; đọc vẫn được.
func TestArchivedCourseWrites(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.code("lưu trữ lớp")
	tc := r.test(q.ID, "t", "1", "1", true, 1)
	r.exec(`update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, r.course)
	ar := func(err error) {
		t.Helper()
		st, code := apiStatus(t, err)
		require.Equal(t, 409, st)
		require.Equal(t, "COURSE_ARCHIVED", code)
	}
	_, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "TRUE_FALSE", Title: "t", Topic: "t", Stem: "s", Value: new(true)})
	ar(err)
	_, err = r.svc.Update(t.Context(), r.course, q.ID, exam.QuestionIn{Type: "CODE", Title: "x", Topic: "t", Stem: "s"}, 1)
	ar(err)
	_, err = r.svc.Review(t.Context(), r.teacher, r.course, q.ID, "REQUEST", 1)
	ar(err)
	_, err = r.svc.Archive(t.Context(), r.teacher, r.course, q.ID)
	ar(err)
	_, err = r.svc.Duplicate(t.Context(), r.teacher, r.course, q.ID)
	ar(err)
	_, err = r.svc.PutCode(t.Context(), r.course, q.ID, exam.CodeIn{Languages: []string{"c11"}}, 1)
	ar(err)
	_, err = r.svc.UpdateTest(t.Context(), r.teacher, r.course, q.ID, tc.ID, exam.TestcaseIn{Weight: new(2)}, true)
	ar(err)
	ar(r.svc.DeleteTest(t.Context(), r.teacher, r.course, q.ID, tc.ID, true))
	_, err = r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
	ar(err)
	_, err = r.svc.EnqueueSuggest(t.Context(), r.teacher, r.course, exam.SuggestIn{Kind: "MCQ", Topic: "x", Count: 1})
	ar(err)
	_, err = r.svc.Get(t.Context(), r.course, q.ID)
	require.NoError(t, err, "đọc vẫn được")
	l, err := r.svc.ListTests(t.Context(), r.course, q.ID, nil, 10)
	require.NoError(t, err)
	require.Len(t, l, 1)
}
