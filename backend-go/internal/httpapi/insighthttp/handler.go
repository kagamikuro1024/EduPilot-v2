// Package insighthttp: báo cáo lỗ hổng kiến thức (US-P10-04/05). Handler mỏng; nghiệp vụ ở internal/insight. Chỉ TEACHER / TA của lớp (ADMIN không qua).
package insighthttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/insight"
)

// Handler là nhóm đường báo cáo lỗ hổng kiến thức.
type Handler struct {
	Svc   *insight.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Idem  func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	staff := h.Guard(auth.StaffRole)
	const q = "/courses/{id}/insights/knowledge-gap"
	r.With(staff, h.Idem).Post(q+"/generate", h.generate)
	r.With(staff).Get(q+"/latest", h.latest)
	r.With(staff).Get(q+"/{rid}/export", h.export)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "insight: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	user, err := uuid.Parse(auth.MustFromContext(r.Context()).Sub)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	course, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	var in struct {
		PeriodDays int `json:"period_days"`
	}
	if r.ContentLength != 0 && !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Generate(r.Context(), user, course, in.PeriodDays)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if out.Reused {
		httpx.WriteJSON(w, http.StatusOK, out)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, out)
}

func (h *Handler) latest(w http.ResponseWriter, r *http.Request) {
	course, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	out, err := h.Svc.Latest(r.Context(), course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	course, e1 := uuid.Parse(chi.URLParam(r, "id"))
	rid, e2 := uuid.Parse(chi.URLParam(r, "rid"))
	if e1 != nil || e2 != nil {
		notFound(w, r)
		return
	}
	if f := r.URL.Query().Get("format"); f != "" && f != "md" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "format", Code: "invalid", Message: "format chỉ nhận md."}))
		return
	}
	md, err := h.Svc.Export(r.Context(), course, rid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="lo-hong-kien-thuc.md"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(md))
}
