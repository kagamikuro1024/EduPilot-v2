package exam_test

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
)

// jsonKeys liệt kê khoá JSON (thẻ `json`) của một struct, bỏ trường `json:"-"`.
func jsonKeys(v any) []string {
	t := reflect.TypeOf(v)
	var out []string
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous { // struct nhúng (ExamStudentView trong StudentListRow): lấy khoá của phần nhúng
			out = append(out, jsonKeys(reflect.New(f.Type).Elem().Interface())...)
			continue
		}
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "-" {
			continue
		}
		out = append(out, tag)
	}
	return out
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestStudentDTOAllowlist — SRS 6.4: khoá JSON của mọi DTO sinh viên khớp tệp danh sách cho phép; thêm trường mới bắt buộc sửa `testdata/student_dto_allowlist.json`
// (để người rà thấy). Không DTO nào có trường nhạy cảm — kể cả khi tệp bị sửa nhầm.
func TestStudentDTOAllowlist(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/student_dto_allowlist.json")
	require.NoError(t, err)
	var allow map[string][]string
	require.NoError(t, json.Unmarshal(raw, &allow))
	types := map[string]any{
		"ExamStudentView": exam.ExamStudentView{}, "MyAttemptView": exam.MyAttemptView{}, "ItemView": exam.ItemView{}, "OptionView": exam.OptionView{},
		"CodeItemView": exam.CodeItemView{}, "SampleView": exam.SampleView{}, "PreviewView": exam.PreviewView{},
		"AttemptView": exam.AttemptView{}, "WriterView": exam.WriterView{}, "ExamBlockView": exam.ExamBlockView{}, "AttemptStartView": exam.AttemptStartView{},
		"SubmittedAttemptView": exam.SubmittedAttemptView{}, "SummaryExamView": exam.SummaryExamView{}, "AttemptSummaryView": exam.AttemptSummaryView{},
		"NoAttemptView": exam.NoAttemptView{}, "SubmitView": exam.SubmitView{}, "SaveView": exam.SaveView{}, "ResultView": exam.ResultView{}, "ResultExamView": exam.ResultExamView{},
	}
	require.Len(t, allow, len(types), "mỗi DTO có một dòng trong tệp")
	forbidden := []string{"answer_key", "correct", "is_correct", "explanation", "override", "original_position", "pinned_last", "hidden", "weight", "tests_version", "reference", "items_count", "attempts", "created_by"}
	for name, v := range types {
		got := jsonKeys(v)
		want, ok := allow[name]
		require.True(t, ok, name)
		slices.Sort(got)
		slices.Sort(want)
		require.Equal(t, want, got, name+": thêm / bớt trường phải sửa tệp allowlist")
		for _, f := range got {
			for _, bad := range forbidden {
				require.False(t, f == bad || strings.HasPrefix(f, "hidden"), "%s có trường nhạy cảm %q", name, f)
			}
		}
	}
}
