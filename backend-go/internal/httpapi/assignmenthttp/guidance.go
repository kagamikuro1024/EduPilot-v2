package assignmenthttp

import (
	"time"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
)

// mountGuidance đăng ký route «Lưu ý chấm» (SRS FEAT-grading-guidance 6 #1–#8). Sinh viên: không mount (DTO sinh viên không đổi). ADMIN chỉ qua ở cờ (#1, #2).
func (h *Handler) mountGuidance(r chi.Router) {
	if h.Guidance == nil {
		return
	}
	manage, staff, teacher := h.Guard(auth.Manage), h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole)
	r.With(manage).Get("/courses/{id}/guidance-flags", h.guidanceFlags)
	r.With(manage).Put("/courses/{id}/guidance-flags/{feature}", h.putGuidanceFlag)
	r.With(staff).Get("/courses/{id}/assignments/{aid}/guidance", h.guidanceOverview)
	r.With(teacher, h.Idem).Post("/courses/{id}/assignments/{aid}/guidance/reflect", h.guidanceReflect)
}

func guidanceActor(w http.ResponseWriter, r *http.Request) (course, aid uuid.UUID, a grading.GuidanceActor, ok bool) {
	course, e1 := uuid.Parse(chi.URLParam(r, "id"))
	p, hasP := auth.FromContext(r.Context())
	uid, e2 := uuid.Parse(p.Sub)
	if e1 != nil || !hasP || e2 != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return course, aid, a, false
	}
	if raw := chi.URLParam(r, "aid"); raw != "" {
		var err error
		if aid, err = uuid.Parse(raw); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return course, aid, a, false
		}
	}
	return course, aid, grading.GuidanceActor{UserID: uid, Admin: p.Role == auth.RoleAdmin, TraceID: llm.TraceID(r.Context())}, true
}

func (h *Handler) guidanceFlags(w http.ResponseWriter, r *http.Request) {
	course, _, a, ok := guidanceActor(w, r)
	if !ok {
		return
	}
	fl, err := h.Guidance.ListFlags(r.Context(), a, course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": fl})
}

func (h *Handler) putGuidanceFlag(w http.ResponseWriter, r *http.Request) {
	course, _, a, ok := guidanceActor(w, r)
	if !ok {
		return
	}
	var in struct {
		Enabled *bool `json:"enabled"`
		Version *int  `json:"version"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.Enabled == nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "enabled", Code: "required", Message: "Thiếu giá trị."}))
		return
	}
	want, e := httpx.WantedVersion(r, in.Version)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	v, err := h.Guidance.SetFlag(r.Context(), a, course, chi.URLParam(r, "feature"), *in.Enabled, want)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) guidanceOverview(w http.ResponseWriter, r *http.Request) {
	course, aid, _, ok := guidanceActor(w, r)
	if !ok {
		return
	}
	pp, perr := httpx.ParsePageParams(r)
	if perr != nil {
		apierr.Write(w, r, perr)
		return
	}
	var at *time.Time
	var cid *uuid.UUID
	if pp.Cursor != nil {
		at, cid = &pp.Cursor.CreatedAt, &pp.Cursor.ID
	}
	v, err := h.Guidance.Overview(r.Context(), course, aid, at, cid, pp.Limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) guidanceReflect(w http.ResponseWriter, r *http.Request) {
	course, aid, a, ok := guidanceActor(w, r)
	if !ok {
		return
	}
	e, err := h.Guidance.Reflect(r.Context(), a, course, aid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, e)
}
