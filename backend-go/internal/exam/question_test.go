package exam_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

func opts(bodies ...string) []exam.OptionIn {
	out := make([]exam.OptionIn, len(bodies))
	for i, b := range bodies {
		out[i] = exam.OptionIn{Body: b}
	}
	return out
}

// codes trả mã lỗi trường (details[].code) của một câu.
func codes(in exam.QuestionIn) []string {
	_, errs, _ := exam.Validate(in)
	var out []string
	for _, e := range errs {
		out = append(out, e.Code)
	}
	return out
}

func valid(over func(*exam.QuestionIn)) exam.QuestionIn {
	in := exam.QuestionIn{Type: "MCQ_SINGLE", Title: "t", Topic: "Mật mã", Difficulty: "MEDIUM", Stem: "Đề?", Options: opts("a", "b", "c"), Correct: []int{1}}
	if over != nil {
		over(&in)
	}
	return in
}

// TestCreateMCQInvalid — AC1: bảng 30 ca (hợp lệ và sai) cho quy tắc 4.1.2; mỗi ca sai ra ĐÚNG mã lỗi trường của SRS 6.1.
func TestCreateMCQInvalid(t *testing.T) {
	t.Parallel()
	long := func(n int) string { return strings.Repeat("a", n) }
	cases := []struct {
		name string
		in   exam.QuestionIn
		want []string // rỗng = hợp lệ
	}{
		{"hợp lệ 3 đáp án", valid(nil), nil},
		{"hợp lệ 2 đáp án (tối thiểu)", valid(func(q *exam.QuestionIn) { q.Options = opts("a", "b") }), nil},
		{"hợp lệ 8 đáp án (tối đa)", valid(func(q *exam.QuestionIn) { q.Options = opts("1", "2", "3", "4", "5", "6", "7", "8") }), nil},
		{"hợp lệ MCQ_MULTI 2/3", valid(func(q *exam.QuestionIn) { q.Type = "MCQ_MULTI"; q.Correct = []int{0, 2} }), nil},
		{"hợp lệ MCQ_MULTI 1/3", valid(func(q *exam.QuestionIn) { q.Type = "MCQ_MULTI"; q.Correct = []int{2} }), nil},
		{"đáp án 1.000 ký tự", valid(func(q *exam.QuestionIn) { q.Options = opts(long(1000), "b") }), nil},
		{"đề 8.000 ký tự", valid(func(q *exam.QuestionIn) { q.Stem = long(8000) }), nil},
		{"tiêu đề 120 ký tự", valid(func(q *exam.QuestionIn) { q.Title = long(120) }), nil},
		{"chủ đề 80 ký tự", valid(func(q *exam.QuestionIn) { q.Topic = long(80) }), nil},
		{"1 đáp án", valid(func(q *exam.QuestionIn) { q.Options = opts("a"); q.Correct = []int{0} }), []string{"OPTION_COUNT"}},
		{"9 đáp án", valid(func(q *exam.QuestionIn) { q.Options = opts("1", "2", "3", "4", "5", "6", "7", "8", "9") }), []string{"OPTION_COUNT"}},
		{"không đáp án", valid(func(q *exam.QuestionIn) { q.Options = nil; q.Correct = nil }), []string{"OPTION_COUNT", "NO_CORRECT_OPTION"}},
		{"đáp án trùng", valid(func(q *exam.QuestionIn) { q.Options = opts("A", "b", "a") }), []string{"DUPLICATE_OPTION"}},
		{"đáp án trùng sau bỏ khoảng trắng", valid(func(q *exam.QuestionIn) { q.Options = opts("x  y", "b", " X Y ") }), []string{"DUPLICATE_OPTION"}},
		{"đáp án rỗng", valid(func(q *exam.QuestionIn) { q.Options = opts("a", "  ", "c") }), []string{"OPTION_BODY_INVALID"}},
		{"đáp án 1.001 ký tự", valid(func(q *exam.QuestionIn) { q.Options = opts(long(1001), "b") }), []string{"OPTION_BODY_INVALID"}},
		{"không có đáp án đúng", valid(func(q *exam.QuestionIn) { q.Correct = nil }), []string{"NO_CORRECT_OPTION"}},
		{"SINGLE hai đáp án đúng", valid(func(q *exam.QuestionIn) { q.Correct = []int{0, 1} }), []string{"SINGLE_MULTIPLE_CORRECT"}},
		{"MULTI mọi đáp án đều đúng", valid(func(q *exam.QuestionIn) { q.Type = "MCQ_MULTI"; q.Correct = []int{0, 1, 2} }), []string{"NO_CORRECT_OPTION"}},
		{"chỉ số đáp án đúng ngoài khoảng", valid(func(q *exam.QuestionIn) { q.Correct = []int{5} }), []string{"INVALID_OPTION_ID"}},
		{"chỉ số âm", valid(func(q *exam.QuestionIn) { q.Correct = []int{-1} }), []string{"INVALID_OPTION_ID"}},
		{"chọn đúng hai lần một đáp án", valid(func(q *exam.QuestionIn) { q.Type = "MCQ_MULTI"; q.Correct = []int{1, 1} }), []string{"DUPLICATE_OPTION"}},
		{"đề quá dài", valid(func(q *exam.QuestionIn) { q.Stem = long(8001) }), []string{"STEM_TOO_LONG"}},
		{"đề toàn khoảng trắng", valid(func(q *exam.QuestionIn) { q.Stem = "   " }), []string{"STEM_EMPTY"}},
		{"tiêu đề 121 ký tự", valid(func(q *exam.QuestionIn) { q.Title = long(121) }), []string{"TITLE_LENGTH"}},
		{"chủ đề rỗng", valid(func(q *exam.QuestionIn) { q.Topic = " " }), []string{"TOPIC_LENGTH"}},
		{"độ khó lạ", valid(func(q *exam.QuestionIn) { q.Difficulty = "INSANE" }), []string{"INVALID_DIFFICULTY"}},
		{"giải thích 4.001 ký tự", valid(func(q *exam.QuestionIn) { q.Explanation = new(long(4001)) }), []string{"EXPLANATION_TOO_LONG"}},
		{"loại lạ", valid(func(q *exam.QuestionIn) { q.Type = "ORAL" }), []string{"INVALID_TYPE"}},
		{"nhiều lỗi cùng lúc trả đủ", valid(func(q *exam.QuestionIn) { q.Title = ""; q.Stem = long(9000); q.Options = opts("a") }), []string{"TITLE_LENGTH", "STEM_TOO_LONG", "OPTION_COUNT", "INVALID_OPTION_ID"}},
	}
	require.GreaterOrEqual(t, len(cases), 24)
	for _, c := range cases {
		got := codes(c.in)
		if len(c.want) == 0 {
			require.Empty(t, got, c.name)
			continue
		}
		for _, w := range c.want {
			require.Contains(t, got, w, c.name)
		}
	}
}

// TestCreateMCQValid — AC1: tạo câu hợp lệ → DRAFT, version 1, id đáp án do máy chủ sinh, `answer_key` lưu theo id (không theo chỉ số).
func TestCreateMCQValid(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "MCQ_MULTI", Title: "Mã hoá đối xứng", Topic: "Mật mã", Difficulty: "MEDIUM", Stem: "Chọn thuật toán đối xứng", Options: opts("AES", "RSA", "3DES"), Correct: []int{0, 2}, Explanation: new("AES và 3DES")})
	require.NoError(t, err)
	require.Equal(t, "DRAFT", d.ReviewStatus)
	require.Equal(t, "MANUAL", d.Origin)
	require.Equal(t, 1, d.Version)
	require.Len(t, d.Options, 3)
	var key struct {
		OptionIDs []uuid.UUID `json:"option_ids"`
	}
	require.NoError(t, json.Unmarshal(d.AnswerKey, &key))
	require.ElementsMatch(t, []uuid.UUID{d.Options[0].ID, d.Options[2].ID}, key.OptionIDs, "đáp án đúng lưu theo id")
	require.Equal(t, []int{1, 2, 3}, []int{d.Options[0].Position, d.Options[1].Position, d.Options[2].Position})
}

// TestStemStoredRaw — AC1: `stem` lưu và trả NGUYÊN VĂN (HTML thô không bị cắt / thoát lúc lưu — an toàn nằm ở bộ render).
func TestStemStoredRaw(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	raw := "Đề có <script>alert(1)</script> và <img src=x onerror=alert(2)> và **đậm** & \"trích\""
	in := valid(func(q *exam.QuestionIn) { q.Stem = raw })
	d, err := r.svc.Create(t.Context(), r.teacher, r.course, in)
	require.NoError(t, err)
	got, err := r.svc.Get(t.Context(), r.course, d.ID)
	require.NoError(t, err)
	require.Equal(t, raw, got.Stem)
	var stored string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select stem from question_bank where id=$1`, d.ID).Scan(&stored))
	require.Equal(t, raw, stored)
}

// TestPinnedWarning — AC1: đáp án kiểu "Tất cả các đáp án trên" chưa ghim cuối → cảnh báo (không lỗi); đã ghim thì không cảnh báo.
func TestPinnedWarning(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	in := valid(func(q *exam.QuestionIn) {
		q.Options = opts("A", "B", "Tất cả các đáp án trên")
		q.Correct = []int{2}
	})
	d, err := r.svc.Create(t.Context(), r.teacher, r.course, in)
	require.NoError(t, err)
	require.Len(t, d.Warnings, 1)
	require.Equal(t, "options[2].body", d.Warnings[0].Field)
	for _, body := range []string{"Cả A và B", "Không có đáp án nào đúng", "A và C", "Cả hai"} {
		_, _, w := exam.Validate(valid(func(q *exam.QuestionIn) { q.Options = opts("x", "y", body) }))
		require.Len(t, w, 1, body)
	}
	in.Options[2].PinnedLast = true
	d, err = r.svc.Create(t.Context(), r.teacher, r.course, in)
	require.NoError(t, err)
	require.Empty(t, d.Warnings)
	_, _, w := exam.Validate(valid(func(q *exam.QuestionIn) { q.Options = opts("Tất cả", "Cả A", "Không có") })) // không khớp đầu câu: "Tất cả" khớp
	require.NotEmpty(t, w)
}

// TestCreateTrueFalse — AC2: TRUE_FALSE lưu `answer_key={"value":…}`, không có đáp án lựa chọn; thiếu `value` → 422.
func TestCreateTrueFalse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, v := range []bool{true, false} {
		d, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "TRUE_FALSE", Title: "tf", Topic: "t", Stem: "1<2", Value: new(v)})
		require.NoError(t, err)
		require.Empty(t, d.Options)
		require.JSONEq(t, `{"value":`+map[bool]string{true: "true", false: "false"}[v]+`}`, string(d.AnswerKey))
	}
	require.Contains(t, codes(exam.QuestionIn{Type: "TRUE_FALSE", Title: "tf", Topic: "t", Stem: "x"}), "VALUE_REQUIRED")
	require.Contains(t, codes(exam.QuestionIn{Type: "TRUE_FALSE", Title: "tf", Topic: "t", Stem: "x", Value: new(true), Options: opts("a", "b")}), "OPTION_COUNT")
}

// TestTypeNotSupportedYet — AC2: SHORT / ESSAY → 422 TYPE_NOT_SUPPORTED (P9); loại lạ → 422.
func TestTypeNotSupportedYet(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, ty := range []string{"SHORT", "ESSAY", "ORAL"} {
		_, err := r.svc.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: ty, Title: "x", Topic: "t", Stem: "s"})
		var ae *apierr.Error
		require.True(t, errors.As(err, &ae), ty)
		require.Equal(t, 422, ae.Status)
		fe := ae.Details.([]apierr.FieldError)
		require.Equal(t, map[string]string{"SHORT": "TYPE_NOT_SUPPORTED", "ESSAY": "TYPE_NOT_SUPPORTED", "ORAL": "INVALID_TYPE"}[ty], fe[0].Code)
	}
}
