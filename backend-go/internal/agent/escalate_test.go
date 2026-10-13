package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestShouldEscalateTable — SRS escalation 4.2.4: mỗi intent × {tin cậy cao, thấp, degraded}; câu mẫu "chưa có công thức" tự mở bằng cờ NoFormula (không so chuỗi).
func TestShouldEscalateTable(t *testing.T) {
	low := ResponseMetadata{LowConfidence: true}
	high := ResponseMetadata{}
	degraded := ResponseMetadata{LowConfidence: true, Degraded: true}
	noCtx := ResponseMetadata{LowConfidence: true, NoContext: true}
	noFormula := ResponseMetadata{NoFormula: true}
	cases := []struct {
		name   string
		intent Intent
		meta   ResponseMetadata
		want   bool
		origin string
	}{
		{"COURSE_QA tin cậy cao", IntentCourseQA, high, false, ""},
		{"COURSE_QA tin cậy thấp", IntentCourseQA, low, true, OriginAutoLowConfidence},
		{"COURSE_QA degraded", IntentCourseQA, degraded, false, ""},
		{"COURSE_QA no_context", IntentCourseQA, noCtx, true, OriginAutoLowConfidence},
		{"LIBRARY_SEARCH thấp", IntentLibrary, low, true, OriginAutoLowConfidence},
		{"PERSONAL_GRADE thấp", IntentGrade, low, true, OriginAutoLowConfidence},
		{"PERSONAL_ATTENDANCE cao", IntentAttendance, high, false, ""},
		{"WHAT_IF_GRADE thấp", IntentWhatIf, low, true, OriginAutoLowConfidence},
		{"EXAM_SCHEDULE thấp", IntentExamSchedule, low, true, OriginAutoLowConfidence},
		{"UPCOMING_EVENTS thấp", IntentUpcoming, low, true, OriginAutoLowConfidence},
		{"GRADE_FORMULA chưa có công thức", IntentGradeFormula, noFormula, true, OriginAutoNoFormula},
		{"GRADE_FORMULA có công thức, tin cậy cao", IntentGradeFormula, high, false, ""},
		{"GRADE_FORMULA chưa có công thức nhưng degraded", IntentGradeFormula, ResponseMetadata{NoFormula: true, Degraded: true}, false, ""},
		{"CRISIS không bao giờ", IntentCrisis, low, false, ""},
		{"CRISIS cả khi NoFormula", IntentCrisis, ResponseMetadata{LowConfidence: true, NoFormula: true}, false, ""},
		{"OTHER_PERSON không bao giờ", IntentOtherPerson, low, false, ""},
		{"SMALLTALK không bao giờ", IntentSmalltalk, low, false, ""},
		{"NoFormula ở intent khác không tự mở", IntentCourseQA, noFormula, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, origin := c.meta.ShouldEscalate(c.intent)
			require.Equal(t, c.want, got)
			require.Equal(t, c.origin, origin)
		})
	}
}
