// Package assignmenthttp: route bài tập, nộp bài, chấm, phúc khảo (FEAT-assignments SRS 6). Handler mỏng; nghiệp vụ ở internal/assignment, internal/submission, internal/grading.
package assignmenthttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/submission"
)

// Handler gắn các route. ADMIN không bao giờ qua (các chế độ *Role): quản trị viên không đọc bài / điểm.
type Handler struct {
	Assignments *assignment.Service
	Submit      *submission.Service
	Guard       func(auth.GuardMode) func(http.Handler) http.Handler
	Idem        func(http.Handler) http.Handler // `Idempotency-Key` bắt buộc
	Log         *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	member, staff, teacher, student := h.Guard(auth.MemberRole), h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole), h.Guard(auth.StudentRole)
	r.With(member).Get("/courses/{id}/assignments", h.list)
	r.With(teacher, h.Idem).Post("/courses/{id}/assignments", h.create)
	r.With(member).Get("/courses/{id}/assignments/{aid}", h.get)
	r.With(teacher).Put("/courses/{id}/assignments/{aid}", h.update)
	r.With(teacher).Delete("/courses/{id}/assignments/{aid}", h.remove)
	r.With(teacher, h.Idem).Post("/courses/{id}/assignments/{aid}/publish", h.publish)
	r.With(teacher).Put("/courses/{id}/assignments/{aid}/rubric", h.putRubric)
	r.With(staff, h.Idem).Post("/courses/{id}/assignments/{aid}/rubric/suggest", h.suggest)
	// Route của sinh viên (US-P7-03): chỉ STUDENT của lớp (TA / TEACHER / ADMIN 403), mọi phản hồi `Cache-Control: no-store`.
	r.With(student).Get("/courses/{id}/assignments/{aid}/mine", h.mine)
	r.With(student).Post("/courses/{id}/assignments/{aid}/submit/presign", h.presign)
	r.With(student, h.Idem).Post("/courses/{id}/assignments/{aid}/submit", h.submit)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	if h.Log != nil {
		h.Log.ErrorContext(r.Context(), "assignment: lỗi nội bộ", "error", err.Error())
	}
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func ids(w http.ResponseWriter, r *http.Request) (course, id uuid.UUID, a assignment.Actor, ok bool) {
	course, e1 := uuid.Parse(chi.URLParam(r, "id"))
	p, hasP := auth.FromContext(r.Context())
	uid, e3 := uuid.Parse(p.Sub)
	if e1 != nil || !hasP || e3 != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return course, id, a, false
	}
	if raw := chi.URLParam(r, "aid"); raw != "" {
		var err error
		if id, err = uuid.Parse(raw); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return course, id, a, false
		}
	}
	ca, _ := auth.CourseFromContext(r.Context())
	a = assignment.Actor{UserID: uid, Student: ca.CourseRole == auth.RoleStudent, TraceID: llm.TraceID(r.Context())}
	return course, id, a, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	course, _, a, ok := ids(w, r)
	if !ok {
		return
	}
	pp, perr := httpx.ParsePageParams(r)
	if perr != nil {
		apierr.Write(w, r, perr)
		return
	}
	page, err := h.Assignments.List(r.Context(), a, course, pp)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if a.Student {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": page.StudentItems, "next_cursor": page.NextCursor})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	course, _, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in assignment.Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Assignments.Create(r.Context(), a, course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.Get(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if a.Student {
		httpx.WriteJSON(w, http.StatusOK, assignment.StudentOf(out))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in assignment.Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.Version < 1 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "version", Code: "required", Message: "Thiếu version."}))
		return
	}
	out, err := h.Assignments.Update(r.Context(), a, course, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	if err := h.Assignments.Delete(r.Context(), a, course, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.Publish(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) putRubric(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in struct {
		Version  int                   `json:"version"`
		Criteria []assignment.RubricIn `json:"criteria"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.Version < 1 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "version", Code: "required", Message: "Thiếu version."}))
		return
	}
	out, err := h.Assignments.SetRubric(r.Context(), a, course, id, in.Criteria, in.Version)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) suggest(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.SuggestRubric(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"criteria": out})
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store") // mọi phản hồi (kể cả lỗi) của route sinh viên: TLR-17
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.Mine(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store") // mọi phản hồi (kể cả lỗi) của route sinh viên: TLR-17
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in submission.PresignIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Submit.Presign(r.Context(), a.UserID, course, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store") // mọi phản hồi (kể cả lỗi) của route sinh viên: TLR-17
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in struct {
		UploadID uuid.UUID `json:"upload_id"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.UploadID == uuid.Nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "upload_id", Code: "required", Message: "Thiếu mã tải lên."}))
		return
	}
	out, err := h.Submit.Complete(r.Context(), a.UserID, course, id, in.UploadID, a.TraceID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"submission": out})
}
