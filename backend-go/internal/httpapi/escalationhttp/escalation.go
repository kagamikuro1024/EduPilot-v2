// Package escalationhttp: route ticket "Nhờ giảng viên" (SRS FEAT-escalation-notify 6). US-P4-01 chỉ có route của sinh viên (#1).
package escalationhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// Handler là nhóm route ticket.
type Handler struct {
	Svc *escalation.Service
	Log *slog.Logger
}

// Mount gắn route vào nhóm nghiệp vụ (đã qua auth.Middleware). Vai và quyền sở hữu do service kiểm (404 không lộ sự tồn tại).
func (h *Handler) Mount(r chi.Router) {
	r.Post("/chat/messages/{mid}/escalate", h.escalate)
}

func (h *Handler) actor(r *http.Request) (escalation.Actor, bool) {
	p, ok := auth.FromContext(r.Context())
	if !ok {
		return escalation.Actor{}, false
	}
	uid, err := uuid.Parse(p.Sub)
	if err != nil {
		return escalation.Actor{}, false
	}
	return escalation.Actor{UserID: uid, Role: p.Role}, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "escalation: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) escalate(w http.ResponseWriter, r *http.Request) {
	a, ok := h.actor(r)
	if !ok {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	mid, err := uuid.Parse(chi.URLParam(r, "mid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	res, err := h.Svc.Escalate(r.Context(), a, mid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	status := http.StatusCreated
	if res.Already {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{"ticket": escalation.ViewOf(res.Ticket), "already": res.Already})
}
