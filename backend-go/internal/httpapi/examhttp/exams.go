package examhttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// MountExams đăng ký thao tác 17–25, 27, 28 của SRS 6.2 (bài thi; #26 hoãn công bố thuộc US-PE-08). Quyền theo `exam.Routes()`.
func (h *Handler) MountExams(r chi.Router) {
	member, staff, teacher := h.Guard(auth.MemberRole), h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole)
	const e = "/courses/{id}/exams"
	r.With(member).Get(e, h.listExams)
	r.With(staff, h.Idem).Post(e, h.createExam)
	r.With(member).Get(e+"/{eid}", h.getExam)
	r.With(staff).Put(e+"/{eid}", h.updateExam)
	r.With(staff).Put(e+"/{eid}/items", h.putItems)
	r.With(staff).Get(e+"/{eid}/preview", h.previewExam)
	r.With(teacher, h.OptIdem).Post(e+"/{eid}/schedule", h.scheduleExam)
	r.With(teacher).Post(e+"/{eid}/unschedule", h.unscheduleExam)
	r.With(teacher).Post(e+"/{eid}/extend", h.extendExam)
	r.With(teacher).Delete(e+"/{eid}", h.deleteExam)
	r.With(staff).Post(e+"/{eid}/clone", h.cloneExam)
}

// examIDs đọc id lớp + id bài từ đường dẫn (sai định dạng → 404).
func (h *Handler) examIDs(w http.ResponseWriter, r *http.Request) (reqCtx, uuid.UUID, bool) {
	c, ok := h.ids(w, r, false)
	if !ok {
		return c, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "eid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return c, uuid.Nil, false
	}
	return c, id, true
}

func isStaff(r *http.Request) bool {
	a, _ := auth.CourseFromContext(r.Context())
	return a.CourseRole == auth.RoleTeacher || a.CourseRole == auth.RoleTA
}

func (h *Handler) listExams(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "", exam.StatusDraft, exam.StatusScheduled, exam.StatusOpen, exam.StatusClosed, exam.StatusPublished:
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "status", Code: "enum", Message: "Trạng thái lọc không hợp lệ."}))
		return
	}
	var cur *exam.Cursor
	if pp.Cursor != nil {
		cur = &exam.Cursor{At: pp.Cursor.CreatedAt, ID: pp.Cursor.ID}
	}
	if isStaff(r) {
		rows, err := h.Svc.ListExams(r.Context(), c.course, status, cur, pp.Fetch())
		if err != nil {
			h.fail(w, r, err)
			return
		}
		httpx.WriteJSONETag(w, r, httpx.Paginate(rows, pp, func(x exam.ExamListItem) (time.Time, string) { return x.CreatedAt, x.ID.String() }), "")
		return
	}
	// Sinh viên: DTO riêng, không bao giờ có DRAFT (lọc ở SQL) — STUDENT là vai duy nhất còn lại sau guard Member.
	rows, err := h.Svc.ListExamsForStudent(r.Context(), c.course, c.actor, status, cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSONETag(w, r, httpx.Paginate(rows, pp, func(x exam.StudentListRow) (time.Time, string) { return x.CreatedAt, x.ID.String() }), "")
}

func (h *Handler) createExam(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	var in exam.ExamIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.Svc.CreateExam(r.Context(), c.actor, c.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusCreated, d)
}

func (h *Handler) getExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	if !isStaff(r) {
		v, err := h.Svc.GetExamForStudent(r.Context(), c.course, c.actor, id)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		httpx.WriteJSONETag(w, r, v, "")
		return
	}
	d, err := h.Svc.GetExam(r.Context(), c.course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSONETag(w, r, d, httpx.ETagVersion(d.Version))
}

func (h *Handler) updateExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	var in exam.ExamIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	ver, aerr := httpx.WantedVersion(r, in.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	d, err := h.Svc.UpdateExam(r.Context(), c.actor, c.course, id, in, ver)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) putItems(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	var in exam.ItemsIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	ver, aerr := httpx.WantedVersion(r, in.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	d, err := h.Svc.PutItems(r.Context(), c.actor, c.course, id, in, ver)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) previewExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.PreviewExam(r.Context(), c.course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store") // mỗi lần xem một hạt giống mới
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) scheduleExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.ScheduleExam(r.Context(), c.actor, c.course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) unscheduleExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.UnscheduleExam(r.Context(), c.actor, c.course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

type extendIn struct {
	ClosesAt time.Time `json:"closes_at" validate:"required"`
}

func (h *Handler) extendExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	var in extendIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.Svc.ExtendExam(r.Context(), c.actor, c.course, id, in.ClosesAt)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) deleteExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	if err := h.Svc.DeleteExam(r.Context(), c.actor, c.course, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cloneExam(w http.ResponseWriter, r *http.Request) {
	c, id, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.CloneExam(r.Context(), c.actor, c.course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}
