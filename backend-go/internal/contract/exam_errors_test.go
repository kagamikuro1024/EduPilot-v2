package contract

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// TestExamErrorCodes — US-PE-01 AC13: 12 mã lỗi mới của SRS FEAT-weekly-exam 6.1 có trong enum `Error.code` của openapi.yaml,
// mỗi mã có lời văn tiếng Việt riêng (không lộ tên bảng / SQL).
func TestExamErrorCodes(t *testing.T) {
	prod, _ := loadBoth(t)
	enum := map[string]bool{}
	for _, v := range prod.Doc.Components.Schemas["Error"].Value.Properties["code"].Value.Enum {
		enum[v.(string)] = true
	}
	codes := []string{apierr.ExamNotOpen, apierr.AttemptAlreadySubmit, apierr.AttemptClosed, apierr.AttemptOtherTab, apierr.DraftConflict, apierr.ExamLocked,
		apierr.QuestionInUse, apierr.ResultNotPublished, apierr.SubmissionLimitReached, apierr.JudgeUnavailable, apierr.AppealWindowClosed, apierr.AppealExists}
	require.Len(t, codes, 12)
	seen := map[string]bool{}
	for _, c := range codes {
		require.True(t, enum[c], "%s thiếu trong enum Error.code", c)
		msg := apierr.DefaultMessage(c)
		require.NotEmpty(t, msg, c)
		require.False(t, seen[msg], "lời văn trùng: %s", msg)
		seen[msg] = true
		require.NotContains(t, msg, "SQL")
	}
	require.GreaterOrEqual(t, len(enum), 49, "37 mã trước PE + 12 mã mới (SRS 6.1); các sprint sau thêm mã nên chỉ còn đòi tối thiểu (proposals.md sprint 6 #8)")
}

// TestExamRoutesNotAheadOfSpec — mọi thao tác PE trong openapi.yaml phải có trong bảng `exam.Routes()` (nguồn quyền duy nhất):
// spec không được mô tả thao tác lạ. (Chiều ngược lại — đủ 56 — chỉ đạt ở cổng PE khi mọi story đã thêm thao tác của mình.)
func TestExamRoutesNotAheadOfSpec(t *testing.T) {
	prod, _ := loadBoth(t)
	known := map[string]bool{}
	for _, r := range exam.Routes() {
		p := "/api/v1/courses/{id}" + r.Path
		if r.No == exam.MeExamLockNo {
			p = "/api/v1" + r.Path
		}
		known[r.Method+" "+normalizeParams(p)] = true
	}
	for path, item := range prod.Doc.Paths.Map() {
		isExam := containsAny(path, "/questions", "/exams", "/exam-lock")
		if !isExam {
			continue
		}
		for method := range item.Operations() {
			require.True(t, known[method+" "+normalizeParams(path)], "%s %s không có trong exam.Routes()", method, path)
		}
	}
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}

// normalizeParams thay tên tham số đường dẫn bằng `{}` để so sánh bất kể đặt tên ({courseId} / {id}).
func normalizeParams(p string) string {
	out := make([]byte, 0, len(p))
	in := false
	for i := 0; i < len(p); i++ {
		switch {
		case p[i] == '{':
			in = true
			out = append(out, '{', '}')
		case p[i] == '}':
			in = false
		case !in:
			out = append(out, p[i])
		}
	}
	return string(out)
}
