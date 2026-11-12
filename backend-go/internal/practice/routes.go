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

// Routes trả các thao tác #1–#14 của SRS FEAT-practice 6 (#15–#16 nhập Forms: Teacher, thêm ở US-P9-11).
func Routes() []Route {
	const a = "/practice/attempts/{aid}"
	student, staff := auth.StudentRole, auth.StaffRole
	return []Route{
		{1, "POST", "/questions/extract", staff, IdemRequired},
		{2, "POST", "/questions/generate", staff, IdemRequired},
		{3, "POST", "/questions/bulk", staff, IdemOptional},
		{4, "GET", "/questions/gen-stats", staff, IdemNone},
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
