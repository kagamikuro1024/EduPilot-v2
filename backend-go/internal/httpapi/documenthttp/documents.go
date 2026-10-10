// Package documenthttp: route tài liệu (SRS FEAT-docs-calendar 6, #1–#2, #9–#11 của US-P8-01). Quyền theo guard của từng route.
package documenthttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler là nhóm route tài liệu.
type Handler struct {
	Svc   *document.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Idem  func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	staff, teacher := h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole)
	const c = "/courses/{id}"
	r.With(staff).Post(c+"/uploads/presign", h.presign)
	r.With(staff, h.Idem).Post(c+"/uploads/complete", h.complete)
	r.With(teacher).Post(c+"/documents/reindex", h.reindexAll)
	r.With(staff).Post(c+"/documents/{docId}/retry", h.retry)
	r.With(staff).Post(c+"/documents/{docId}/reindex", h.reindex)
}

type ids struct{ actor, course, doc uuid.UUID }

func (h *Handler) ids(w http.ResponseWriter, r *http.Request, withDoc bool) (ids, bool) {
	var x ids
	var err error
	if x.actor, err = uuid.Parse(auth.MustFromContext(r.Context()).Sub); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return x, false
	}
	if x.course, err = uuid.Parse(chi.URLParam(r, "id")); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return x, false
	}
	if withDoc {
		if x.doc, err = uuid.Parse(chi.URLParam(r, "docId")); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return x, false
		}
	}
	return x, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "document: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	var in document.PresignIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Presign(r.Context(), x.actor, x.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	var in document.CompleteIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Complete(r.Context(), x.actor, x.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, out)
}

func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	out, err := h.Svc.Retry(r.Context(), x.actor, x.course, x.doc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, out)
}

func (h *Handler) reindex(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	out, err := h.Svc.Reindex(r.Context(), x.actor, x.course, x.doc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, out)
}

func (h *Handler) reindexAll(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	out, err := h.Svc.ReindexAll(r.Context(), x.actor, x.course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, out)
}
