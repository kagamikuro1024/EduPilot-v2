// Package calendarhttp: lịch gộp, sự kiện của Staff, token và feed ICS (SRS FEAT-docs-calendar 6, #16–#23).
package calendarhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/calendar"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
)

// Handler là nhóm route lịch.
type Handler struct {
	Svc      *calendar.Service
	Guard    func(auth.GuardMode) func(http.Handler) http.Handler
	ClientIP func(*http.Request) string
	Log      *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware: #16–#19, #21–#23.
func (h *Handler) Mount(r chi.Router) {
	member, staff := h.Guard(auth.MemberRole), h.Guard(auth.StaffRole)
	const c = "/courses/{id}/calendar"
	r.With(member).Get(c, h.list)
	r.With(staff).Post(c+"/events", h.create)
	r.With(staff).Put(c+"/events/{eventId}", h.update)
	r.With(staff).Delete(c+"/events/{eventId}", h.remove)
	r.Post("/me/calendar/ics-token", h.issue)
	r.Delete("/me/calendar/ics-token", h.revoke)
	r.Get("/me/calendar/ics-token", h.exists)
}

// MountPublic đăng ký feed ICS (#20) NGOÀI auth.Middleware: danh tính là token.
func (h *Handler) MountPublic(r chi.Router) { r.Get("/calendar/feed.ics", h.feed) }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "calendar: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

type req struct {
	course uuid.UUID
	a      calendar.Actor
}

func param(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return id, false
	}
	return id, true
}

func (h *Handler) actor(w http.ResponseWriter, r *http.Request) (req, bool) {
	var x req
	p, okp := auth.FromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if !okp || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return x, false
	}
	x.a = calendar.Actor{UserID: uid, TraceID: llm.TraceID(r.Context())}
	if ca, ok := auth.CourseFromContext(r.Context()); ok {
		admin := p.Role == auth.RoleAdmin
		x.a.Student = ca.CourseRole == auth.RoleStudent && !admin
		x.a.Staff = ca.CourseRole == auth.RoleTeacher || ca.CourseRole == auth.RoleTA || admin
		if x.course, err = uuid.Parse(chi.URLParam(r, "id")); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return x, false
		}
	}
	return x, true
}

func parseTime(w http.ResponseWriter, r *http.Request, name string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, r.URL.Query().Get(name))
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: name, Code: "format", Message: name + " là giờ RFC 3339."}))
		return t, false
	}
	return t, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	from, ok := parseTime(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseTime(w, r, "to")
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	out, err := h.Svc.List(r.Context(), x.a, x.course, from, to, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSONETag(w, r, out, "")
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	var in calendar.EventIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Create(r.Context(), x.a, x.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, ok := param(w, r, "eventId")
	if !ok {
		return
	}
	var in calendar.EventIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, aerr := httpx.WantedVersion(r, in.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	out, err := h.Svc.Update(r.Context(), x.a, x.course, id, v, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	id, ok := param(w, r, "eventId")
	if !ok {
		return
	}
	if err := h.Svc.Delete(r.Context(), x.a, x.course, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) issue(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	url, err := h.Svc.IssueToken(r.Context(), x.a.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"url": url})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	if err := h.Svc.RevokeToken(r.Context(), x.a.UserID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) exists(w http.ResponseWriter, r *http.Request) {
	x, ok := h.actor(w, r)
	if !ok {
		return
	}
	e, err := h.Svc.TokenExists(r.Context(), x.a.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"exists": e})
}

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	if !h.Svc.RateOK(r.Context(), h.ClientIP(r)) {
		apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(60))
		return
	}
	body, err := h.Svc.Feed(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	etag := httpx.ETagBody(body)
	hd := w.Header()
	hd.Set("ETag", etag)
	hd.Set("Cache-Control", "private, max-age=300")
	if httpx.MatchIfNoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	hd.Set("Content-Type", "text/calendar; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
