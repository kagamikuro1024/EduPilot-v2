package exam_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
)

// E5: 34/100 lời giải AI sinh mang mã đoạn nội bộ «[C18]»; sinh viên không được thấy.
func TestStripCiteMarkers(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ in, want string }{
		"cuối câu":                {"Theo quy chế, sinh viên được thi lại [C18].", "Theo quy chế, sinh viên được thi lại."},
		"giữa câu":                {"Điều 5 [C2] quy định thi lại.", "Điều 5 quy định thi lại."},
		"đầu câu":                 {"[C3] Điều 5 quy định thi lại.", "Điều 5 quy định thi lại."},
		"nhiều mã một ngoặc":      {"Đúng vì tài liệu nêu rõ [C1, C2].", "Đúng vì tài liệu nêu rõ."},
		"nhiều mã không dấu cách": {"Xem [C1,C22,C3].", "Xem."},
		"hai ngoặc liền":          {"Xem [C1][C2].", "Xem."},
		"xuống dòng trước mã":     {"Dòng một\n[C4]\nDòng hai", "Dòng một\nDòng hai"},
		"không có mã":             {"Không có mã nào.", "Không có mã nào."},
		"trích dẫn số [n] của chat giữ nguyên": {"Theo quy chế [1].", "Theo quy chế [1]."},
		"ngoặc không phải mã":                  {"Mảng a[C] và [Ci] và [C1a]", "Mảng a[C] và [Ci] và [C1a]"},
		"rỗng":                                 {"", ""},
	} {
		require.Equal(t, tc.want, exam.StripCiteMarkers(tc.in), name)
	}
	require.Nil(t, exam.StripCiteMarkersPtr(nil))
	s := "Vì [C7] đúng."
	require.Equal(t, "Vì đúng.", *exam.StripCiteMarkersPtr(&s))
	require.Equal(t, "Vì [C7] đúng.", s, "không sửa chuỗi gốc")
}

// E5: câu AI sinh đã lưu còn mã «[C18]» ⇒ sinh viên không thấy ở BẤT KỲ đâu của đề tuần: lúc làm bài (stem, phương án) và ở kết quả (stem, phương án, lời giải).
func TestExamStripsCiteMarkersOnRead(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	m := r.mixExam()
	r.exec(`update question_bank set stem = stem || ' [C3]', explanation = 'Vì [C18, C2] đúng.' where course_id = $1`, r.course)
	r.exec(`update question_options set body = body || ' [C1]' where course_id = $1`, r.course)

	v, _ := r.start(m.e, m.sv, m.tab) // mở lại lượt đang làm: dựng DTO từ ngân hàng hiện tại
	require.NotEmpty(t, v.Items)
	for _, it := range v.Items {
		require.NotContains(t, it.Stem, "[C")
		for _, o := range it.Options {
			require.NotContains(t, o.Body, "[C")
		}
	}
	m.finish(true, false)
	m.closeExam()
	r.exec(`update exams set status='CLOSED' where id=$1`, m.e.ID)
	_, err := r.svc.PublishDue(t.Context())
	require.NoError(t, err)
	res, err := r.svc.AttemptResult(t.Context(), m.sv, r.course, m.e.ID, m.v.Attempt.ID)
	require.NoError(t, err)
	var explained int
	for _, it := range res.Items {
		require.NotContains(t, it.Stem, "[C")
		for _, o := range it.Options {
			require.NotContains(t, o.Body, "[C")
		}
		if it.Explanation != nil {
			explained++
			require.Equal(t, "Vì đúng.", *it.Explanation)
		}
	}
	require.NotZero(t, explained, "kết quả phải có lời giải để kiểm")
}
