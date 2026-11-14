// Package observabilityhttp: quan sát AI (US-P10-01…03). Handler mỏng; nghiệp vụ ở internal/observability.
package observabilityhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/observability"
)

// Handler là nhóm đường quan sát AI.
type Handler struct {
	Svc   *observability.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware. Ba đường `/admin/observability/*` chỉ ADMIN; bản lớp qua CourseAccessGuard(Manage): TEACHER của lớp hoặc ADMIN
// (giảng viên lớp khác, TA, sinh viên → 403 như mọi route lớp; xem P-11-11).
func (h *Handler) Mount(r chi.Router) {
	admin := r.With(auth.RequireRole(auth.RoleAdmin))
	admin.Get("/admin/observability/summary", h.summary)
	admin.Get("/admin/observability/requests", h.requests)
	admin.Post("/admin/observability/requests/{id}/open", h.open)
	r.With(h.Guard(auth.Manage)).Get("/courses/{id}/observability/summary", h.course)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, what string, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return // client đã bỏ đi
	case errors.Is(err, context.DeadlineExceeded):
		apierr.Write(w, r, apierr.New(http.StatusGatewayTimeout, apierr.DeadlineExceeded))
		return
	}
	h.Log.ErrorContext(r.Context(), "observability: "+what, "error", err.Error())
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func badField(w http.ResponseWriter, r *http.Request, field, msg string) {
	apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: field, Code: "invalid", Message: msg}))
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	out, err := h.Svc.Summary(r.Context(), r.URL.Query().Get("window"))
	if errors.Is(err, observability.ErrBadFilter) {
		badField(w, r, "window", "window là 24h hoặc 7d.")
		return
	}
	if err != nil {
		h.fail(w, r, "tổng hợp", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) requests(w http.ResponseWriter, r *http.Request) {
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	rq := observability.RequestsQuery{Filter: q.Get("filter"), Task: q.Get("task"), Limit: pg.Limit}
	if raw := q.Get("course_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			badField(w, r, "course_id", "course_id không hợp lệ.")
			return
		}
		rq.CourseID = &id
	}
	for _, f := range []struct {
		name string
		dst  **time.Time
	}{{"from", &rq.From}, {"to", &rq.To}} {
		if raw := q.Get(f.name); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				badField(w, r, f.name, f.name+" phải theo RFC 3339.")
				return
			}
			*f.dst = &t
		}
	}
	if pg.Cursor != nil {
		rq.CursorAt, rq.CursorID = &pg.Cursor.CreatedAt, &pg.Cursor.ID
	}
	rows, err := h.Svc.Requests(r.Context(), rq)
	if errors.Is(err, observability.ErrBadFilter) {
		badField(w, r, "filter", "filter là all, problem hoặc privacy.")
		return
	}
	if err != nil {
		h.fail(w, r, "bảng yêu cầu", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pg, func(x observability.RequestRow) (time.Time, string) { return x.At, x.ID.String() }))
}

type openInput struct {
	Reason string `json:"reason"`
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var in openInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	p := auth.MustFromContext(r.Context())
	actor, err := uuid.Parse(p.Sub)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	d, err := h.Svc.Open(r.Context(), actor, id, in.Reason, trace.SpanContextFromContext(r.Context()).TraceID().String())
	switch {
	case errors.Is(err, observability.ErrReasonRequired):
		apierr.Write(w, r, apierr.New(http.StatusUnprocessableEntity, apierr.ReasonRequired))
	case errors.Is(err, observability.ErrNotFound):
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
	case errors.Is(err, observability.ErrExpired):
		apierr.Write(w, r, apierr.New(http.StatusGone, apierr.PayloadExpired))
	case err != nil:
		h.fail(w, r, "mở chi tiết", err)
	default:
		httpx.WriteJSON(w, http.StatusOK, d)
	}
}

func (h *Handler) course(w http.ResponseWriter, r *http.Request) {
	cid, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	out, err := h.Svc.CourseSummary(r.Context(), cid)
	if err != nil {
		h.fail(w, r, "tổng hợp lớp", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
