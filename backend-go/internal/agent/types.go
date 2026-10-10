// Package agent: định tuyến TẤT ĐỊNH tới tool Go rồi đúng MỘT lần sinh chữ (D47, SRS FEAT-private-chat-pii 4.4–4.6). Không vòng lặp agent mở;
// LLM không gọi tool; danh tính chỉ đến từ TrustedContext (nguyên tắc bất biến 2). Gói này KHÔNG gọi privacy.Mask*: che là việc của internal/llm.
package agent

import (
	"github.com/google/uuid"
)

// Intent là ý định đã phân loại bằng luật (không LLM).
type Intent string

// 12 intent (SRS 4.5).
const (
	IntentCrisis        Intent = "CRISIS"
	IntentOtherPerson   Intent = "OTHER_PERSON"
	IntentWhatIf        Intent = "WHAT_IF_GRADE"
	IntentGradeFormula  Intent = "GRADE_FORMULA"
	IntentAttendance    Intent = "PERSONAL_ATTENDANCE"
	IntentParticipation Intent = "PERSONAL_PARTICIPATION"
	IntentGrade         Intent = "PERSONAL_GRADE"
	IntentExamSchedule  Intent = "EXAM_SCHEDULE"
	IntentUpcoming      Intent = "UPCOMING_EVENTS"
	IntentLibrary       Intent = "LIBRARY_SEARCH"
	IntentCourseQA      Intent = "COURSE_QA"
	IntentSmalltalk     Intent = "SMALLTALK"
)

// Personal cho biết intent đọc dữ liệu cá nhân (không cache, không đi kênh công khai).
func (i Intent) Personal() bool {
	switch i {
	case IntentAttendance, IntentParticipation, IntentGrade, IntentWhatIf, IntentGradeFormula, IntentExamSchedule, IntentUpcoming, IntentOtherPerson, IntentCrisis:
		return true
	}
	return false
}

// TrustedContext là danh tính DUY NHẤT mà tool được dùng, do handler dựng từ JWT + CourseAccess + phiên đã nạp theo user_id (SRS 4.5).
// Không có đường nào khác để tool nhận user_id: tham số tool không có trường danh tính (TestPersonalToolsHaveNoIdentityParam).
type TrustedContext struct {
	UserID    uuid.UUID
	CourseID  uuid.UUID
	Role      string
	SessionID uuid.UUID
	TraceID   string
}
