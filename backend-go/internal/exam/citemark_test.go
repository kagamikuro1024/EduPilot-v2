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
