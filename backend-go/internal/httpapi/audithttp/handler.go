// Package audithttp: GET /admin/audit (US-P10-14). Chỉ ADMIN; không route sửa / xoá / xuất.
package audithttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/edupilot/backend-go/internal/audit"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler là nhóm đường nhật ký kiểm toán.
type Handler struct {
	Svc *audit.Service
	Log *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/admin/audit", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	admin, err := uuid.Parse(auth.MustFromContext(r.Context()).Sub)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	q := r.URL.Query()
	qy := audit.Query{Entity: q.Get("entity"), ActorID: q.Get("actor_id"), From: q.Get("from"), To: q.Get("to"), Cursor: q.Get("cursor")}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			n = -1 // ngoài 1–100 → 422 ở service
		}
		qy.Limit = n
	}
	page, err := h.Svc.List(r.Context(), admin, qy, trace.SpanContextFromContext(r.Context()).TraceID().String())
	if err != nil {
		var ae *apierr.Error
		if errors.As(err, &ae) {
			apierr.Write(w, r, ae)
			return
		}
		h.Log.ErrorContext(r.Context(), "audit: đọc nhật ký", "error", err.Error())
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, page)
}
