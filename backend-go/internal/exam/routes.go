package exam

import "github.com/edupilot/backend-go/internal/auth"

// Idem là yêu cầu Idempotency-Key của một thao tác (SRS 6.2; thiếu khi bắt buộc → 422 IDEMPOTENCY_KEY_REQUIRED).
type Idem int

const (
	IdemNone Idem = iota
	IdemOptional
	IdemRequired
)

// Route là một thao tác của SRS FEAT-weekly-exam 6.2. `Path` tính từ `/api/v1/courses/{courseId}` (trừ #43 `/me/exam-lock`);
// `Mode` là chế độ CourseAccessGuard của route. Đây là NGUỒN DUY NHẤT về quyền của thi hằng tuần: router, test ma trận quyền và
// kiểm `openapi.yaml` đều đọc từ đây.
type Route struct {
	No     int
	Method string
	Path   string
	Mode   auth.GuardMode
	Idem   Idem
}

// MeExamLockNo là số thao tác của `GET /me/exam-lock` (không thuộc lớp, không qua CourseAccessGuard).
const MeExamLockNo = 43

// Routes trả 56 thao tác của SRS 6.2 + #57 (chi tiết một cặp độ giống — đề xuất #17) (không gồm route thử T1). ADMIN bị chặn ở mọi route có Mode (MemberRole / StaffRole / TeacherRole / StudentRole).
func Routes() []Route {
	const (
		q  = "/questions"
		qi = "/questions/{qid}"
		e  = "/exams"
		ei = "/exams/{eid}"
		at = "/exams/{eid}/attempts/{aid}"
	)
	staff, teacher, student, member := auth.StaffRole, auth.TeacherRole, auth.StudentRole, auth.MemberRole
	return []Route{
		{1, "GET", q, staff, IdemNone},
		{2, "POST", q, staff, IdemOptional},
		{3, "GET", qi, staff, IdemNone},
		{4, "PUT", qi, staff, IdemNone},
		{5, "POST", qi + "/archive", staff, IdemNone},
		{6, "POST", qi + "/duplicate", staff, IdemNone},
		{7, "PUT", qi + "/review", staff, IdemNone},
		{8, "PUT", qi + "/code", staff, IdemNone},
		{9, "GET", qi + "/testcases", staff, IdemNone},
		{10, "POST", qi + "/testcases", staff, IdemNone},
		{11, "PUT", qi + "/testcases/{tid}", staff, IdemNone}, // sau khi bài đóng: chỉ Teacher (kiểm ở service)
		{12, "DELETE", qi + "/testcases/{tid}", staff, IdemNone},
		{13, "POST", qi + "/testcases/import", staff, IdemNone},
		{14, "POST", qi + "/testcases/approve", staff, IdemNone},
		{15, "POST", qi + "/reference/verify", staff, IdemRequired},
		{16, "POST", q + "/suggest", staff, IdemRequired},
		{17, "GET", e, member, IdemNone},
		{18, "POST", e, staff, IdemRequired},
		{19, "GET", ei, member, IdemNone},
		{20, "PUT", ei, staff, IdemNone},
		{21, "PUT", ei + "/items", staff, IdemNone},
		{22, "GET", ei + "/preview", staff, IdemNone},
		{23, "POST", ei + "/schedule", teacher, IdemOptional},
		{24, "POST", ei + "/unschedule", teacher, IdemNone},
		{25, "POST", ei + "/extend", teacher, IdemNone},
		{26, "PUT", ei + "/publish-hold", teacher, IdemNone},
		{27, "DELETE", ei, teacher, IdemNone},
		{28, "POST", ei + "/clone", staff, IdemNone},
		{29, "POST", ei + "/attempts", student, IdemRequired},
		{30, "GET", ei + "/attempts/mine", student, IdemNone},
		{31, "PUT", at + "/answers", student, IdemNone},
		{32, "PUT", at + "/code/{itemId}/draft", student, IdemNone},
		{33, "POST", at + "/code/{itemId}/run", student, IdemRequired},
		{34, "GET", at + "/runs/{runId}", student, IdemNone},
		{35, "POST", at + "/code/{itemId}/submit", student, IdemRequired},
		{36, "GET", at + "/code/{itemId}/submissions", student, IdemNone},
		{37, "GET", at + "/submissions/{sid}", student, IdemNone},
		{38, "POST", at + "/events", student, IdemNone},
		{39, "POST", at + "/takeover", student, IdemNone},
		{40, "POST", at + "/submit", student, IdemRequired},
		{41, "GET", at + "/result", student, IdemNone},
		{42, "POST", at + "/appeal", student, IdemRequired},
		{MeExamLockNo, "GET", "/me/exam-lock", 0, IdemNone}, // không thuộc lớp: Mode không dùng
		{44, "GET", ei + "/results", staff, IdemNone},
		{45, "GET", ei + "/results/{aid}", staff, IdemNone},
		{46, "PUT", ei + "/results/{aid}/score", teacher, IdemNone},
		{47, "PUT", ei + "/items/{itemId}/override", teacher, IdemNone},
		{48, "POST", ei + "/regrade", teacher, IdemRequired},
		{49, "GET", ei + "/stats", staff, IdemNone},
		{50, "GET", ei + "/results.csv", staff, IdemNone},
		{51, "GET", ei + "/events", teacher, IdemNone},
		{52, "GET", ei + "/similarity", teacher, IdemNone},
		{53, "POST", ei + "/similarity/run", teacher, IdemOptional},
		{54, "PUT", ei + "/similarity/{id}/review", teacher, IdemNone},
		{55, "GET", ei + "/appeals", staff, IdemNone},
		{56, "POST", ei + "/appeals/{id}/answer", teacher, IdemOptional},
		{57, "GET", ei + "/similarity/{id}", teacher, IdemNone}, // thêm: hai mã cạnh nhau của một cặp (đề xuất #17)
	}
}
