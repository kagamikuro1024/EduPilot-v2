package exam_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// ---- helper -----------------------------------------------------------------------------------------------------------------

// goodIn là thân tạo bài hợp lệ: mở sau 2 giờ, đóng sau 4 giờ, làm 45 phút.
func goodIn(title string) exam.ExamIn {
	opens, closes := time.Now().UTC().Add(2*time.Hour), time.Now().UTC().Add(4*time.Hour)
	return exam.ExamIn{Title: &title, OpensAt: &opens, ClosesAt: &closes, DurationMinutes: new(45)}
}

func (r *rig) newExam(in exam.ExamIn) exam.ExamDetail {
	r.t.Helper()
	d, err := r.svc.CreateExam(r.t.Context(), r.teacher, r.course, in)
	require.NoError(r.t, err)
	return d
}

// approvedMCQ tạo một câu MCQ rồi duyệt.
func (r *rig) approvedMCQ(title string) uuid.UUID {
	r.t.Helper()
	q := r.mcq(title)
	_, err := r.svc.Review(r.t.Context(), r.teacher, r.course, q.ID, "APPROVE", q.Version)
	require.NoError(r.t, err)
	return q.ID
}

// approvedCode tạo câu code đủ test mẫu + ẩn, đã kiểm lời giải mẫu, đã duyệt.
func (r *rig) approvedCode(title string) uuid.UUID {
	r.t.Helper()
	d, _, _ := r.codeWithTests(title)
	r.verify(d.ID)
	_, err := r.svc.Review(r.t.Context(), r.teacher, r.course, d.ID, "APPROVE", r.version(d.ID))
	require.NoError(r.t, err)
	return d.ID
}

func (r *rig) putItems(e exam.ExamDetail, qs ...uuid.UUID) (exam.ExamDetail, error) {
	r.t.Helper()
	in := exam.ItemsIn{}
	for _, q := range qs {
		in.Items = append(in.Items, exam.ItemIn{QuestionID: q, Points: decimal.RequireFromString("1.00")})
	}
	return r.svc.PutItems(r.t.Context(), r.teacher, r.course, e.ID, in, e.Version)
}

func (r *rig) mustItems(e exam.ExamDetail, qs ...uuid.UUID) exam.ExamDetail {
	r.t.Helper()
	d, err := r.putItems(e, qs...)
	require.NoError(r.t, err)
	return d
}

func (r *rig) count(sql string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&n))
	return n
}

// student thêm một sinh viên ACTIVE vào lớp.
func (r *rig) student(status string) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into users (email, full_name, role) values ($1, 'Sinh Viên', 'STUDENT') returning id`, "sv."+uuid.NewString()[:8]+"@example.test").Scan(&id))
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at) values ($1, $2, 'STUDENT', $3::enrollment_status, 'ADMIN', case when $3 = 'REMOVED' then now() end)`, r.course, id, status)
	return id
}

func (r *rig) otherCourse() uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	u := uuid.New()
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(u[i])%len(alpha)]
	}
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp khác', '2026-2027-HK1', $2, $3) returning id`,
		"OT-"+strings.ToUpper(uuid.NewString()[:6]), string(jc), r.teacher).Scan(&id))
	return id
}

func (r *rig) at(clk clock.Clock) *exam.Service {
	return &exam.Service{Pool: r.pool, Jobs: jobs.NewService(r.pool), Clock: clk}
}

// ---- AC1 -------------------------------------------------------------------------------------------------------------------

// TestCreateExamDefaults — AC1: mặc định shuffle_* = true, max_score 10,00, bước 0,01, PARTIAL, hiện đáp án, 7 ngày phúc khảo; DRAFT; kind MCQ; version 1; `audit_log`.
func TestCreateExamDefaults(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.newExam(exam.ExamIn{Title: new("Kiểm tra tuần 9")})
	require.Equal(t, "DRAFT", d.Status)
	require.Equal(t, "DRAFT", d.EffectiveStatus)
	require.Equal(t, "MCQ", d.Kind)
	require.True(t, d.ShuffleQuestions && d.ShuffleOptions && d.RevealAnswers)
	require.Equal(t, "10.00", d.MaxScore)
	require.Equal(t, "0.01", d.RoundingStep)
	require.Equal(t, "PARTIAL", d.MultiScoring)
	require.Equal(t, 7, d.AppealDays)
	require.Equal(t, 1, d.Version)
	require.Empty(t, d.Items)
	require.Nil(t, d.OpensAt)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity='exam' and entity_id=$1 and action='exam.create'`, d.ID.String()))
	// chọn tay các giá trị khác mặc định
	in := goodIn("Có cài đặt")
	in.MaxScore, in.RoundingStep = new(decimal.RequireFromString("20")), new(decimal.RequireFromString("0.25"))
	in.MultiScoring, in.AppealDays, in.ShuffleOptions, in.RevealAnswers = new("ALL_OR_NOTHING"), new(0), new(false), new(false)
	d = r.newExam(in)
	require.Equal(t, "20.00", d.MaxScore)
	require.Equal(t, "0.25", d.RoundingStep)
	require.Equal(t, "ALL_OR_NOTHING", d.MultiScoring)
	require.False(t, d.ShuffleOptions || d.RevealAnswers)
	require.Zero(t, d.AppealDays)
	require.Equal(t, 45, *d.DurationMinutes)
}

// TestExamValidation — AC1: bảng ≥ 20 ca (biên hợp lệ và không hợp lệ); mọi lỗi trả cùng lúc.
func TestExamValidation(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	base := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	at := func(d time.Duration) *time.Time { v := base.Add(d); return &v }
	dec := func(s string) *decimal.Decimal { v := decimal.RequireFromString(s); return &v }
	mod := func(f func(*exam.ExamIn)) exam.ExamIn {
		in := exam.ExamIn{Title: new("Bài"), OpensAt: at(0), ClosesAt: at(3 * time.Hour), DurationMinutes: new(45)}
		f(&in)
		return in
	}
	cases := []struct {
		name string
		in   exam.ExamIn
		want []string // mã lỗi mong đợi (rỗng = hợp lệ)
	}{
		{"hợp lệ", mod(func(*exam.ExamIn) {}), nil},
		{"tiêu đề rỗng", mod(func(in *exam.ExamIn) { in.Title = new("") }), []string{"TITLE_LENGTH"}},
		{"tiêu đề 120 ký tự", mod(func(in *exam.ExamIn) { in.Title = new(strings.Repeat("a", 120)) }), nil},
		{"tiêu đề 121 ký tự", mod(func(in *exam.ExamIn) { in.Title = new(strings.Repeat("a", 121)) }), []string{"TITLE_LENGTH"}},
		{"tiêu đề 120 chữ có dấu", mod(func(in *exam.ExamIn) { in.Title = new(strings.Repeat("ế", 120)) }), nil},
		{"hướng dẫn 4000", mod(func(in *exam.ExamIn) { in.Instructions = new(strings.Repeat("x", 4000)) }), nil},
		{"hướng dẫn 4001", mod(func(in *exam.ExamIn) { in.Instructions = new(strings.Repeat("x", 4001)) }), []string{"INSTRUCTIONS_TOO_LONG"}},
		{"đóng bằng mở", mod(func(in *exam.ExamIn) { in.ClosesAt = at(0) }), []string{"CLOSES_BEFORE_OPENS"}},
		{"đóng trước mở", mod(func(in *exam.ExamIn) { in.ClosesAt = at(-time.Minute) }), []string{"CLOSES_BEFORE_OPENS"}},
		{"thời lượng 4 phút", mod(func(in *exam.ExamIn) { in.DurationMinutes = new(4) }), []string{"DURATION_TOO_SHORT"}},
		{"thời lượng 5 phút", mod(func(in *exam.ExamIn) { in.DurationMinutes = new(5) }), nil},
		{"thời lượng 0", mod(func(in *exam.ExamIn) { in.DurationMinutes = new(0) }), []string{"DURATION_TOO_SHORT"}},
		{"thời lượng 301", mod(func(in *exam.ExamIn) { in.ClosesAt, in.DurationMinutes = at(10*time.Hour), new(301) }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"thời lượng 300 trong khung 300", mod(func(in *exam.ExamIn) { in.ClosesAt, in.DurationMinutes = at(300*time.Minute), new(300) }), nil},
		{"thời lượng vượt khung 1 phút", mod(func(in *exam.ExamIn) { in.ClosesAt, in.DurationMinutes = at(44*time.Minute), new(45) }), []string{"DURATION_EXCEEDS_WINDOW"}},
		{"thời lượng đúng khung", mod(func(in *exam.ExamIn) { in.ClosesAt, in.DurationMinutes = at(45*time.Minute), new(45) }), nil},
		{"điểm tối đa 0", mod(func(in *exam.ExamIn) { in.MaxScore = dec("0") }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"điểm tối đa 100", mod(func(in *exam.ExamIn) { in.MaxScore = dec("100") }), nil},
		{"điểm tối đa 100,01", mod(func(in *exam.ExamIn) { in.MaxScore = dec("100.01") }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"điểm tối đa 3 chữ số thập phân", mod(func(in *exam.ExamIn) { in.MaxScore = dec("10.001") }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"bước 0,25", mod(func(in *exam.ExamIn) { in.RoundingStep = dec("0.25") }), nil},
		{"bước 1", mod(func(in *exam.ExamIn) { in.RoundingStep = dec("1") }), nil},
		{"bước 0,2", mod(func(in *exam.ExamIn) { in.RoundingStep = dec("0.2") }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"cách tính lạ", mod(func(in *exam.ExamIn) { in.MultiScoring = new("HALF") }), []string{"INVALID_TYPE"}},
		{"phúc khảo -1", mod(func(in *exam.ExamIn) { in.AppealDays = new(-1) }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"phúc khảo 30", mod(func(in *exam.ExamIn) { in.AppealDays = new(30) }), nil},
		{"phúc khảo 31", mod(func(in *exam.ExamIn) { in.AppealDays = new(31) }), []string{"LIMIT_OUT_OF_RANGE"}},
		{"nhiều lỗi cùng lúc", mod(func(in *exam.ExamIn) { in.Title, in.ClosesAt, in.AppealDays = new(""), at(-time.Hour), new(99) }), []string{"TITLE_LENGTH", "CLOSES_BEFORE_OPENS", "LIMIT_OUT_OF_RANGE"}},
	}
	require.GreaterOrEqual(t, len(cases), 20)
	for _, c := range cases {
		_, err := r.svc.CreateExam(t.Context(), r.teacher, r.course, c.in)
		if len(c.want) == 0 {
			require.NoError(t, err, c.name)
			continue
		}
		require.ElementsMatch(t, c.want, fieldCodes(t, err), c.name)
		st, code := apiStatus(t, err)
		require.Equal(t, 422, st, c.name)
		require.Equal(t, "VALIDATION_FAILED", code, c.name)
	}
	// thiếu tiêu đề khi tạo
	_, err := r.svc.CreateExam(t.Context(), r.teacher, r.course, exam.ExamIn{})
	require.Contains(t, fieldCodes(t, err), "VALUE_REQUIRED")
	// lớp lưu trữ → 409
	r.exec(`update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, r.course)
	_, err = r.svc.CreateExam(t.Context(), r.teacher, r.course, goodIn("lưu trữ"))
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "COURSE_ARCHIVED", code)
}

// TestExamKindDerived — AC1: kind suy ra từ mục (chỉ MCQ → MCQ; chỉ code → CODE; cả hai → MIXED; bỏ hết → MCQ), không nhận từ client.
func TestExamKindDerived(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.newExam(goodIn("kind"))
	require.Equal(t, "MCQ", e.Kind)
	m, c := r.approvedMCQ("m"), r.approvedCode("c")
	e = r.mustItems(e, m)
	require.Equal(t, "MCQ", e.Kind)
	e = r.mustItems(e, c)
	require.Equal(t, "CODE", e.Kind)
	e = r.mustItems(e, m, c)
	require.Equal(t, "MIXED", e.Kind)
	// `kind` không có trong thân tạo / sửa: ExamIn không có trường này (bộ giải mã từ chối khoá lạ → 422: kịch bản hợp đồng `POST /exams` có `kind`)
	var in exam.ExamIn
	require.NoError(t, json.Unmarshal([]byte(`{"title":"x"}`), &in))
}

// ---- AC2 -------------------------------------------------------------------------------------------------------------------

// TestPutItems — AC2: thay TOÀN BỘ danh sách theo thứ tự gửi (position 1…n), điểm 2 chữ số, version +1, audit.
func TestPutItems(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.newExam(goodIn("items"))
	a, b, c := r.approvedMCQ("a"), r.approvedMCQ("b"), r.approvedMCQ("c")
	d, err := r.svc.PutItems(t.Context(), r.teacher, r.course, e.ID, exam.ItemsIn{Items: []exam.ItemIn{
		{QuestionID: c, Points: decimal.RequireFromString("2.5")}, {QuestionID: a, Points: decimal.RequireFromString("1")}, {QuestionID: b, Points: decimal.RequireFromString("0.01")},
	}}, e.Version)
	require.NoError(t, err)
	require.Equal(t, e.Version+1, d.Version)
	require.Len(t, d.Items, 3)
	for i, want := range []struct {
		q   uuid.UUID
		pts string
	}{{c, "2.50"}, {a, "1.00"}, {b, "0.01"}} {
		require.Equal(t, i+1, d.Items[i].Position)
		require.Equal(t, want.q, d.Items[i].QuestionID)
		require.Equal(t, want.pts, d.Items[i].Points)
	}
	require.Equal(t, 3, d.ItemsCount)
	// thay toàn bộ: danh sách mới còn một câu
	d = r.mustItems(d, b)
	require.Len(t, d.Items, 1)
	require.Equal(t, 1, d.Items[0].Position)
	require.Equal(t, 1, r.count(`select count(*) from exam_items where exam_id=$1`, e.ID))
	require.Equal(t, 2, r.count(`select count(*) from audit_log where entity_id=$1 and action='exam.items'`, e.ID.String()))
	// 100 mục là biên hợp lệ
	var many []uuid.UUID
	for i := range 100 {
		many = append(many, r.approvedMCQ(fmt.Sprintf("q%d", i)))
	}
	d = r.mustItems(d, many...)
	require.Len(t, d.Items, 100)
}

// TestPutItemsRejects — AC2: mọi lỗi trả cùng lúc: NO_ITEMS, DUPLICATE_ITEM, ITEM_NOT_APPROVED (chưa duyệt, lưu trữ), QUESTION_NOT_IN_COURSE (lớp khác, id lạ), điểm sai, > 100 mục.
func TestPutItemsRejects(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.newExam(goodIn("reject"))
	ok := r.approvedMCQ("ok")
	draft := r.mcq("nháp").ID
	archived := r.approvedMCQ("lưu trữ")
	_, err := r.svc.Archive(t.Context(), r.teacher, r.course, archived)
	require.NoError(t, err)
	other := r.otherCourse()
	foreign, err := r.svc.Create(t.Context(), r.teacher, other, exam.QuestionIn{Type: "MCQ_SINGLE", Title: "lớp khác", Topic: "t", Difficulty: "EASY", Stem: "s", Options: []exam.OptionIn{{Body: "a"}, {Body: "b"}}, Correct: []int{0}})
	require.NoError(t, err)
	_, err = r.svc.Review(t.Context(), r.teacher, other, foreign.ID, "APPROVE", foreign.Version)
	require.NoError(t, err)

	_, err = r.putItems(e)
	require.Equal(t, []string{"NO_ITEMS"}, fieldCodes(t, err))
	_, err = r.putItems(e, ok, ok)
	require.Equal(t, []string{"DUPLICATE_ITEM"}, fieldCodes(t, err))
	_, err = r.putItems(e, draft, archived, foreign.ID, uuid.New(), ok)
	require.Equal(t, []string{"ITEM_NOT_APPROVED", "ITEM_NOT_APPROVED", "QUESTION_NOT_IN_COURSE", "QUESTION_NOT_IN_COURSE"}, fieldCodes(t, err))
	for _, pts := range []string{"0", "-1", "100.01", "2.555"} {
		_, err = r.svc.PutItems(t.Context(), r.teacher, r.course, e.ID, exam.ItemsIn{Items: []exam.ItemIn{{QuestionID: ok, Points: decimal.RequireFromString(pts)}}}, e.Version)
		require.Equal(t, []string{"LIMIT_OUT_OF_RANGE"}, fieldCodes(t, err), pts)
	}
	_, err = r.svc.PutItems(t.Context(), r.teacher, r.course, e.ID, exam.ItemsIn{Items: []exam.ItemIn{{QuestionID: ok, Points: decimal.NewFromInt(100)}}}, e.Version)
	require.NoError(t, err, "100 điểm là biên hợp lệ")
	var tooMany []exam.ItemIn
	for range 101 {
		tooMany = append(tooMany, exam.ItemIn{QuestionID: ok, Points: decimal.NewFromInt(1)})
	}
	_, err = r.svc.PutItems(t.Context(), r.teacher, r.course, e.ID, exam.ItemsIn{Items: tooMany}, e.Version+1)
	require.Equal(t, []string{"LIMIT_OUT_OF_RANGE"}, fieldCodes(t, err))
	require.Zero(t, r.count(`select count(*) from exam_items where exam_id=$1 and question_id in ($2, $3)`, e.ID, draft, archived), "lỗi thì không ghi gì")
}

// TestPutItemsLockedWhenScheduled — AC2: đổi mục khi không còn DRAFT → 409 EXAM_LOCKED.
func TestPutItemsLockedWhenScheduled(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, st := range []string{"SCHEDULED", "OPEN", "CLOSED", "PUBLISHED"} {
		q := r.approvedMCQ("khoá " + st)
		eid := r.useIn(q, st)
		cur, err := r.svc.GetExam(t.Context(), r.course, eid)
		require.NoError(t, err)
		_, err = r.putItems(cur, q)
		status, code := apiStatus(t, err)
		require.Equal(t, 409, status, st)
		require.Equal(t, "EXAM_LOCKED", code, st)
	}
}

// ---- AC3 -------------------------------------------------------------------------------------------------------------------

// TestPreviewNoAttemptCreated — AC3: xem trước không ghi gì (không lượt làm, không sự kiện, không audit, không outbox) và xem được cả bài DRAFT.
func TestPreviewNoAttemptCreated(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.mustItems(r.newExam(goodIn("preview")), r.approvedMCQ("a"), r.approvedCode("c"))
	snap := func() [5]int {
		return [5]int{r.count(`select count(*) from exam_attempts where exam_id=$1`, e.ID), r.count(`select count(*) from exam_events where exam_id=$1`, e.ID),
			r.count(`select count(*) from audit_log where course_id=$1`, r.course), r.count(`select count(*) from outbox`), r.count(`select count(*) from exams where course_id=$1`, r.course)}
	}
	before := snap()
	for range 3 {
		v, err := r.svc.PreviewExam(t.Context(), r.course, e.ID)
		require.NoError(t, err)
		require.True(t, v.Preview)
		require.Len(t, v.Items, 2)
		require.Equal(t, "DRAFT", v.Exam.Status)
	}
	require.Equal(t, before, snap())
	_, err := r.svc.PreviewExam(t.Context(), r.course, uuid.New())
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestPreviewShapeEqualsStudent — AC3: cùng DTO sinh viên (khoá JSON của mục = thẻ json của ItemView); không có đáp án đúng, giải thích, test ẩn, lời giải mẫu;
// xáo trộn theo hạt giống (đáp án ghim đứng cuối; tắt xáo thì giữ thứ tự).
func TestPreviewShapeEqualsStudent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	const canary = "CANARY-KHÔNG-ĐƯỢC-LỘ"
	var qs []uuid.UUID
	for i := range 8 {
		q, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "MCQ_SINGLE", Title: fmt.Sprintf("q%d", i), Topic: "t", Difficulty: "EASY", Stem: fmt.Sprintf("Đề %d", i), Explanation: new(canary),
			Options: []exam.OptionIn{{Body: "A"}, {Body: "B"}, {Body: "C"}, {Body: "Tất cả các đáp án trên", PinnedLast: true}}, Correct: []int{1}})
		require.NoError(t, err)
		_, err = r.svc.Review(t.Context(), r.teacher, r.course, q.ID, "APPROVE", q.Version)
		require.NoError(t, err)
		qs = append(qs, q.ID)
	}
	cq, _, _ := r.codeWithTests("code")
	r.exec(`update code_testcases set name = $2 where problem_id = $1 and not is_sample`, cq.ID, "CANARYhidden")
	r.exec(`update code_problems set reference_source = $2 where question_id = $1`, cq.ID, "// "+canary)
	r.verify(cq.ID)
	_, err := r.svc.Review(t.Context(), r.teacher, r.course, cq.ID, "APPROVE", r.version(cq.ID))
	require.NoError(t, err)
	e := r.mustItems(r.newExam(goodIn("shape")), append(qs, cq.ID)...)

	order := func(shuffleQ bool) []string {
		var seen []string
		for range 12 {
			v, err := r.svc.PreviewExam(t.Context(), r.course, e.ID)
			require.NoError(t, err)
			raw, err := json.Marshal(v)
			require.NoError(t, err)
			require.NotContains(t, string(raw), canary, "không lộ giải thích / test ẩn / lời giải mẫu")
			for _, bad := range []string{"answer_key", "correct", "explanation", "pinned_last", "override", "hidden", "reference", "weight", "tests_version"} {
				require.NotContains(t, string(raw), bad)
			}
			var generic struct {
				Items []map[string]json.RawMessage `json:"items"`
			}
			require.NoError(t, json.Unmarshal(raw, &generic))
			for _, it := range generic.Items {
				require.ElementsMatch(t, jsonKeys(exam.ItemView{}), keysOf(it), "khoá của mục xem trước = khoá của ItemView")
			}
			ids := make([]string, len(v.Items))
			for i, it := range v.Items {
				require.Equal(t, i+1, it.Position)
				ids[i] = it.ItemID.String()
				if it.Type == "MCQ_SINGLE" {
					require.Len(t, it.Options, 4)
					require.Equal(t, "Tất cả các đáp án trên", it.Options[3].Body, "đáp án ghim đứng cuối")
				} else {
					require.NotNil(t, it.Code)
					require.Len(t, it.Code.Samples, 1)
					require.Equal(t, "sample1", it.Code.Samples[0].Name)
				}
			}
			seen = append(seen, strings.Join(ids, ","))
		}
		return seen
	}
	distinct := func(s []string) int {
		m := map[string]bool{}
		for _, x := range s {
			m[x] = true
		}
		return len(m)
	}
	require.Greater(t, distinct(order(true)), 1, "bật xáo: mỗi lần xem một thứ tự")
	cur, err := r.svc.GetExam(t.Context(), r.course, e.ID)
	require.NoError(t, err)
	_, err = r.svc.UpdateExam(t.Context(), r.teacher, r.course, e.ID, exam.ExamIn{ShuffleQuestions: new(false), ShuffleOptions: new(false)}, cur.Version)
	require.NoError(t, err)
	require.Equal(t, 1, distinct(order(false)), "tắt xáo: giữ thứ tự của giảng viên")
}

// ---- AC8 -------------------------------------------------------------------------------------------------------------------

// TestDeleteOnlyDraft — AC8: chỉ DRAFT xoá được (cascade mục); khác DRAFT → 409 EXAM_LOCKED.
func TestDeleteOnlyDraft(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.mustItems(r.newExam(goodIn("xoá")), r.approvedMCQ("a"))
	require.NoError(t, r.svc.DeleteExam(t.Context(), r.teacher, r.course, e.ID))
	require.Zero(t, r.count(`select count(*) from exam_items where exam_id=$1`, e.ID))
	require.Zero(t, r.count(`select count(*) from exams where id=$1`, e.ID))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id=$1 and action='exam.delete'`, e.ID.String()))
	for _, st := range []string{"SCHEDULED", "OPEN", "CLOSED", "PUBLISHED"} {
		eid := r.useIn(r.approvedMCQ("giữ "+st), st)
		st2, code := apiStatus(t, r.svc.DeleteExam(t.Context(), r.teacher, r.course, eid))
		require.Equal(t, 409, st2, st)
		require.Equal(t, "EXAM_LOCKED", code, st)
		require.Equal(t, 1, r.count(`select count(*) from exams where id=$1`, eid))
	}
	st, _ := apiStatus(t, r.svc.DeleteExam(t.Context(), r.teacher, r.course, uuid.New()))
	require.Equal(t, 404, st)
}

// TestCloneExam — AC8: bản sao DRAFT cùng mục / điểm / cài đặt, không giờ, tiêu đề + " (bản sao)" (cắt để ≤ 120).
func TestCloneExam(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	in := goodIn("Tuần 9")
	in.MaxScore, in.RoundingStep, in.MultiScoring, in.AppealDays = new(decimal.RequireFromString("20")), new(decimal.RequireFromString("0.5")), new("ALL_OR_NOTHING"), new(3)
	in.ShuffleQuestions, in.Instructions = new(false), new("**Đọc kỹ** đề")
	a, b := r.approvedMCQ("a"), r.approvedCode("b")
	e := r.newExam(in)
	e, err := r.svc.PutItems(t.Context(), r.teacher, r.course, e.ID, exam.ItemsIn{Items: []exam.ItemIn{{QuestionID: b, Points: decimal.RequireFromString("7")}, {QuestionID: a, Points: decimal.RequireFromString("3.5")}}}, e.Version)
	require.NoError(t, err)
	c, err := r.svc.CloneExam(t.Context(), r.ta, r.course, e.ID)
	require.NoError(t, err)
	require.NotEqual(t, e.ID, c.ID)
	require.Equal(t, "Tuần 9 (bản sao)", c.Title)
	require.Equal(t, "DRAFT", c.Status)
	require.Nil(t, c.OpensAt)
	require.Nil(t, c.ClosesAt)
	require.Nil(t, c.DurationMinutes)
	require.Equal(t, "MIXED", c.Kind)
	require.Equal(t, "20.00", c.MaxScore)
	require.Equal(t, "0.50", c.RoundingStep)
	require.Equal(t, "ALL_OR_NOTHING", c.MultiScoring)
	require.Equal(t, 3, c.AppealDays)
	require.False(t, c.ShuffleQuestions)
	require.Equal(t, "**Đọc kỹ** đề", *c.Instructions)
	require.Equal(t, r.ta, c.CreatedBy)
	require.Len(t, c.Items, 2)
	require.Equal(t, b, c.Items[0].QuestionID)
	require.Equal(t, "7.00", c.Items[0].Points)
	require.Equal(t, "3.50", c.Items[1].Points)
	require.Equal(t, 0, c.Attempts.Started)
	// bản gốc không đổi
	o, err := r.svc.GetExam(t.Context(), r.course, e.ID)
	require.NoError(t, err)
	require.Equal(t, e.Version, o.Version)
	// tiêu đề dài 120 ký tự vẫn ≤ 120 sau khi thêm hậu tố
	long := r.newExam(exam.ExamIn{Title: new(strings.Repeat("d", 120))})
	c, err = r.svc.CloneExam(t.Context(), r.teacher, r.course, long.ID)
	require.NoError(t, err)
	require.Len(t, []rune(c.Title), 120)
	require.True(t, strings.HasSuffix(c.Title, " (bản sao)"))
	st, _ := apiStatus(t, errOf(r.svc.CloneExam(t.Context(), r.teacher, r.course, uuid.New())))
	require.Equal(t, 404, st)
}

func errOf[T any](_ T, err error) error { return err }

// ---- AC9 -------------------------------------------------------------------------------------------------------------------

// TestExamListStaffVsStudent — AC9: Staff thấy mọi bài kể cả DRAFT với số liệu; sinh viên chỉ SCHEDULED / OPEN / CLOSED / PUBLISHED với trường giới hạn; lọc theo trạng thái hiệu lực; con trỏ.
func TestExamListStaffVsStudent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	draft := r.newExam(goodIn("nháp"))
	var ids = map[string]uuid.UUID{"DRAFT": draft.ID}
	for _, st := range []string{"SCHEDULED", "OPEN", "CLOSED", "PUBLISHED"} {
		ids[st] = r.useIn(r.approvedMCQ("q"+st), st)
	}
	// SCHEDULED nhưng đã quá giờ mở → hiệu lực OPEN (không phụ thuộc bộ lập lịch)
	rows, err := r.svc.ListExams(t.Context(), r.course, "", nil, 101)
	require.NoError(t, err)
	require.Len(t, rows, 5)
	byID := map[uuid.UUID]exam.ExamListItem{}
	for _, x := range rows {
		byID[x.ID] = x
	}
	require.Equal(t, "OPEN", byID[ids["SCHEDULED"]].EffectiveStatus, "SCHEDULED + now ≥ opens_at → OPEN")
	require.Equal(t, "SCHEDULED", byID[ids["SCHEDULED"]].Status)
	require.Equal(t, 1, byID[ids["SCHEDULED"]].ItemsCount)
	open, err := r.svc.ListExams(t.Context(), r.course, "OPEN", nil, 101)
	require.NoError(t, err)
	require.Len(t, open, 2, "lọc OPEN theo trạng thái hiệu lực: SCHEDULED đã tới giờ + OPEN")
	dr, err := r.svc.ListExams(t.Context(), r.course, "DRAFT", nil, 101)
	require.NoError(t, err)
	require.Len(t, dr, 1)

	stu, err := r.svc.ListExamsForStudent(t.Context(), r.course, sv, "", nil, 101)
	require.NoError(t, err)
	require.Len(t, stu, 4)
	for _, x := range stu {
		require.NotEqual(t, "DRAFT", x.Status)
		require.NotEqual(t, draft.ID, x.ID)
		require.Nil(t, x.MyAttempt)
		require.Nil(t, x.MyScore)
	}
	raw, err := json.Marshal(stu[0])
	require.NoError(t, err)
	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &keys))
	require.ElementsMatch(t, jsonKeys(exam.ExamStudentView{}), keysOf(keys))
	for _, bad := range []string{"items", "override", "items_count", "attempts", "version", "publish_hold", "created_by"} {
		require.NotContains(t, keysOf(keys), bad)
	}
	// con trỏ: hai trang
	p1, err := r.svc.ListExams(t.Context(), r.course, "", nil, 3)
	require.NoError(t, err)
	require.Len(t, p1, 3)
	p2, err := r.svc.ListExams(t.Context(), r.course, "", &exam.Cursor{At: p1[1].CreatedAt, ID: p1[1].ID}, 10)
	require.NoError(t, err)
	require.Len(t, p2, 3, "sau hàng thứ hai của trang một còn 3 bài")
	require.Equal(t, p1[2].ID, p2[0].ID)
}

// TestExamDetailStudentProjection — AC9: chi tiết của sinh viên có `my_attempt` của CHÍNH họ và `my_score` chỉ khi PUBLISHED; lượt của người khác không lộ.
func TestExamDetailStudentProjection(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	me, other := r.student("ACTIVE"), r.student("ACTIVE")
	pub := r.useIn(r.approvedMCQ("pub"), "PUBLISHED")
	closed := r.useIn(r.approvedMCQ("closed"), "CLOSED")
	attempt := func(eid, who uuid.UUID, score string) {
		r.exec(`insert into exam_attempts (course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason, auto_score, graded_at)
			values ($1, $2, $3, 'GRADED', now() - interval '3 hours', now() - interval '2 hours 30 minutes', now() - interval '2 hours 40 minutes', 'MANUAL', $4::numeric, now())`, r.course, eid, who, score)
	}
	attempt(pub, me, "8.50")
	attempt(pub, other, "3.00")
	attempt(closed, me, "9.00")
	v, err := r.svc.GetExamForStudent(t.Context(), r.course, me, pub)
	require.NoError(t, err)
	require.NotNil(t, v.MyAttempt)
	require.Equal(t, "GRADED", v.MyAttempt.Status)
	require.Equal(t, "8.50", *v.MyScore)
	v, err = r.svc.GetExamForStudent(t.Context(), r.course, me, closed)
	require.NoError(t, err)
	require.NotNil(t, v.MyAttempt)
	require.Nil(t, v.MyScore, "chưa công bố → không có điểm")
	v, err = r.svc.GetExamForStudent(t.Context(), r.course, other, closed)
	require.NoError(t, err)
	require.Nil(t, v.MyAttempt, "lượt của người khác không lộ")
	// điều chỉnh của giảng viên thay điểm máy chấm sau khi công bố
	r.exec(`update exam_attempts set adjusted_score = 9.25, adjusted_reason = 'x', adjusted_by = $2, adjusted_at = now() where exam_id = $1 and student_id = $3`, pub, r.teacher, me)
	v, err = r.svc.GetExamForStudent(t.Context(), r.course, me, pub)
	require.NoError(t, err)
	require.Equal(t, "9.25", *v.MyScore)
	// bài lớp khác → 404
	o := r.otherCourse()
	_, err = r.svc.GetExamForStudent(t.Context(), o, me, pub)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestStudentNeverSeesDraft — AC9: DRAFT không bao giờ xuất hiện cho sinh viên (danh sách, chi tiết, lọc ?status=DRAFT).
func TestStudentNeverSeesDraft(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	d := r.newExam(goodIn("nháp"))
	rows, err := r.svc.ListExamsForStudent(t.Context(), r.course, sv, "", nil, 101)
	require.NoError(t, err)
	require.Empty(t, rows)
	rows, err = r.svc.ListExamsForStudent(t.Context(), r.course, sv, "DRAFT", nil, 101)
	require.NoError(t, err)
	require.Empty(t, rows)
	_, err = r.svc.GetExamForStudent(t.Context(), r.course, sv, d.ID)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}

// TestExamEditVersionConflict — AC10: hai người sửa nháp cùng lúc → đúng một thành công, người kia VersionConflict kèm bản hiện tại.
func TestExamEditVersionConflict(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e := r.newExam(goodIn("đồng thời"))
	var wg sync.WaitGroup
	errs := make([]error, 6)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = r.svc.UpdateExam(t.Context(), r.teacher, r.course, e.ID, exam.ExamIn{Title: new(fmt.Sprintf("lần %d", i))}, e.Version)
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		var vc *exam.VersionConflict
		switch {
		case err == nil:
			ok++
		case errors.As(err, &vc):
			require.Equal(t, e.Version+1, vc.Version)
			cur, isDetail := vc.Current.(exam.ExamDetail)
			require.True(t, isDetail)
			require.Equal(t, e.Version+1, cur.Version)
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 1, ok)
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// detailsOf trả `details` của một *apierr.Error.
func detailsOf(t *testing.T, err error) any {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "cần *apierr.Error, có %v", err)
	return ae.Details
}
