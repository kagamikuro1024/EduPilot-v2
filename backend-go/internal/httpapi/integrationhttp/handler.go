// Package integrationhttp gắn route cấu hình tích hợp theo lớp (US-P7-05): IMAP, SMTP (chỉ đọc), `Đồng bộ ngay`, thư không xử lý được.
package integrationhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/mailin"
)

// Handler gắn các route. GET = `Manage` (TEACHER của lớp hoặc ADMIN); PUT / test = `Manage` + vai ADMIN; sync-now = nhân viên lớp (ADMIN 403, D39).
type Handler struct {
	Svc   *mailin.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Idem  func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	manage, staff := h.Guard(auth.Manage), h.Guard(auth.StaffRole)
	admin := auth.RequireRole(auth.RoleAdmin)
	r.With(manage).Get("/courses/{id}/integrations/imap", h.get(mailin.KindIMAP))
	r.With(manage, admin).Put("/courses/{id}/integrations/imap", h.put)
	r.With(manage, admin).Post("/courses/{id}/integrations/imap/test", h.test)
	r.With(staff, h.Idem).Post("/courses/{id}/integrations/imap/sync-now", h.syncNow)
	r.With(manage).Get("/courses/{id}/integrations/smtp", h.get(mailin.KindSMTP))
	r.With(manage, admin).Post("/courses/{id}/integrations/smtp/test", h.smtpTest)
	r.With(staff).Get("/courses/{id}/mail-inbound", h.inbound)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	if h.Log != nil {
		h.Log.ErrorContext(r.Context(), "integration: lỗi nội bộ", "error", err.Error())
	}
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func ids(w http.ResponseWriter, r *http.Request) (course, user uuid.UUID, ok bool) {
	course, e1 := uuid.Parse(chi.URLParam(r, "id"))
	p, hasP := auth.FromContext(r.Context())
	user, e2 := uuid.Parse(p.Sub)
	if e1 != nil || !hasP || e2 != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return course, user, false
	}
	return course, user, true
}

func (h *Handler) get(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		course, _, ok := ids(w, r)
		if !ok {
			return
		}
		v, err := h.Svc.Get(r.Context(), course, kind)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		httpx.WriteJSON(w, http.StatusOK, v)
	}
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	course, user, ok := ids(w, r)
	if !ok {
		return
	}
	var in mailin.PutIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.Put(r.Context(), user, course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) test(w http.ResponseWriter, r *http.Request) {
	course, _, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Svc.Test(r.Context(), course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) syncNow(w http.ResponseWriter, r *http.Request) {
	course, user, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Svc.SyncNow(r.Context(), user, course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, out)
}

func (h *Handler) smtpTest(w http.ResponseWriter, r *http.Request) {
	_, user, ok := ids(w, r)
	if !ok {
		return
	}
	if err := h.Svc.SMTPTest(r.Context(), user); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]bool{"queued": true})
}

func (h *Handler) inbound(w http.ResponseWriter, r *http.Request) {
	course, _, ok := ids(w, r)
	if !ok {
		return
	}
	pp, perr := httpx.ParsePageParams(r)
	if perr != nil {
		apierr.Write(w, r, perr)
		return
	}
	out, err := h.Svc.ListInbound(r.Context(), course, r.URL.Query().Get("status"), pp)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}
