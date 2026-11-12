package practice

import "github.com/edupilot/backend-go/internal/auth"

// Idem là yêu cầu Idempotency-Key của một thao tác.
type Idem int

const (
	IdemNone Idem = iota
	IdemOptional
	IdemRequired
)

// Route là một thao tác của SRS FEAT-practice mục 6 (số thao tác theo bảng ở đó). `Path` tính từ `/api/v1/courses/{courseId}`.
// Nguồn DUY NHẤT về quyền của luyện đề: router và test ma trận quyền / openapi đều đọc từ đây.
type Route struct {
	No     int
	Method string
	Path   string
	Mode   auth.GuardMode
	Idem   Idem
}

// Routes trả các thao tác của sinh viên (#5–#14, trừ #11–#12 SSE / feedback ở file riêng) và của staff / giảng viên ở các file khác.
func Routes() []Route {
	const a = "/practice/attempts/{aid}"
	student := auth.StudentRole
	return []Route{
		{5, "GET", "/practice/overview", student, IdemNone},
		{6, "POST", "/practice/start", student, IdemRequired},
		{7, "GET", a, student, IdemNone},
		{8, "PUT", a + "/answers/{itemId}", student, IdemNone},
		{9, "PATCH", a, student, IdemNone},
		{10, "POST", a + "/submit", student, IdemRequired},
		{11, "POST", a + "/items/{itemId}/explain", student, IdemNone},
		{12, "POST", a + "/items/{itemId}/feedback", student, IdemRequired},
		{13, "GET", "/practice/history", student, IdemNone},
		{14, "GET", "/practice/weakness", student, IdemNone},
	}
}
