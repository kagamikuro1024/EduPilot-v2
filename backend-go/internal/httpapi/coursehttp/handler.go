// Package coursehttp: `GET /me/courses` và `GET /courses/{id}` (US-P2-07). Handler mỏng, nghiệp vụ ở internal/course.
package coursehttp

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler là nhóm đường đọc lớp.
type Handler struct {
	Courses course.Service
	Guard   func(auth.GuardMode) func(http.Handler) http.Handler
	Log     *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware. Đường tĩnh /courses/join… (US-P2-09) PHẢI đăng ký trước `/courses/{id}`.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/me/courses", h.myCourses)
	r.With(h.Guard(auth.MemberOrAdmin)).Get("/courses/{id}", h.get)
}

type brief struct {
	ID          uuid.UUID `json:"id"`
	ClassCode   string    `json:"class_code"`
	SubjectCode string    `json:"subject_code"`
	Name        string    `json:"name"`
	Semester    string    `json:"semester"`
	Status      string    `json:"status"`
}

type myCourseItem struct {
	Course           brief  `json:"course"`
	RoleInCourse     string `json:"role_in_course"`
	EnrollmentStatus string `json:"enrollment_status"`
}

func (h *Handler) myCourses(w http.ResponseWriter, r *http.Request) {
	p := auth.MustFromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var cur *course.Cursor
	if pp.Cursor != nil {
		cur = &course.Cursor{At: pp.Cursor.CreatedAt, ID: pp.Cursor.ID}
	}
	rows, err := h.Courses.MyCourses(r.Context(), uid, p.Role, cur, pp.Limit)
	if err != nil {
		h.Log.ErrorContext(r.Context(), "me/courses lỗi", "error", err.Error())
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
		return
	}
	page := httpx.Paginate(rows, pp, func(m course.MyCourse) (time.Time, string) { return m.EnrolledAt, m.EnrollmentID.String() })
	items := make([]myCourseItem, len(page.Items))
	for i, m := range page.Items {
		b := m.Course
		items[i] = myCourseItem{Course: brief{ID: b.ID, ClassCode: b.ClassCode, SubjectCode: b.SubjectCode, Name: b.Name, Semester: b.Semester, Status: b.Status}, RoleInCourse: m.RoleInCourse, EnrollmentStatus: m.EnrollmentStatus}
	}
	httpx.WriteJSONETag(w, r, httpx.Page[myCourseItem]{Items: items, NextCursor: page.NextCursor}, "")
}

type teacher struct {
	FullName string `json:"full_name"`
}

type counts struct {
	StudentsActive  int `json:"students_active"`
	StudentsPending int `json:"students_pending"`
}

type detail struct {
	brief
	Teachers []teacher `json:"teachers"`
	MyRole   string    `json:"my_role"`
	Counts   *counts   `json:"counts,omitempty"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	access, _ := auth.CourseFromContext(r.Context())
	role := string(access.CourseRole)
	staff := access.Status == "ACTIVE" && (access.CourseRole == auth.RoleTeacher || access.CourseRole == auth.RoleTA)
	d, err := h.Courses.Get(r.Context(), id, role, staff)
	switch {
	case errors.Is(err, course.ErrNotFound):
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound)) // chỉ ADMIN tới được đây với lớp không có thật
		return
	case err != nil:
		h.Log.ErrorContext(r.Context(), "courses/{id} lỗi", "error", err.Error())
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
		return
	}
	out := detail{brief: brief{ID: d.ID, ClassCode: d.ClassCode, SubjectCode: d.SubjectCode, Name: d.Name, Semester: d.Semester, Status: d.Status}, MyRole: d.MyRole, Teachers: make([]teacher, len(d.Teachers))}
	for i, n := range d.Teachers {
		out.Teachers[i] = teacher{FullName: n}
	}
	if d.Counts != nil {
		out.Counts = &counts{StudentsActive: d.Counts.Active, StudentsPending: d.Counts.Pending}
	}
	httpx.WriteJSONETag(w, r, out, "")
}
