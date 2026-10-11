// Package libraryhttp: route thư viện của sinh viên (SRS FEAT-docs-calendar 6, #13–#15). Chỉ STUDENT ACTIVE của lớp (guard `StudentRole`).
package libraryhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/library"
)

// Handler là nhóm route thư viện.
type Handler struct {
	Svc   *library.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	student := h.Guard(auth.StudentRole)
	const c = "/courses/{id}/library"
	r.With(student).Get(c, h.list)
	r.With(student).Get(c+"/{docId}", h.get)
	r.With(student).Get(c+"/{docId}/download", h.download)
}

func (h *Handler) course(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return id, false
	}
	return id, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "library: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(w, r)
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	f := library.Filter{Q: q.Get("q"), Type: q.Get("type"), Category: q.Get("category")}
	if raw := q.Get("week"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 20 {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "week", Code: "range", Message: "week là số từ 1 đến 20."}))
			return
		}
		f.Week = &n
	}
	out, err := h.Svc.List(r.Context(), course, f, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSONETag(w, r, out, "") // ETag = băm thân đã tuần tự hoá: đổi cả khi sửa đoạn (#12); If-None-Match → 304
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(w, r)
	if !ok {
		return
	}
	doc, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	out, err := h.Svc.Get(r.Context(), course, doc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	course, ok := h.course(w, r)
	if !ok {
		return
	}
	doc, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	u, err := h.Svc.Download(r.Context(), course, doc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": u})
}
