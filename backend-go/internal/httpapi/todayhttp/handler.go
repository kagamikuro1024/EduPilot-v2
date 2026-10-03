// Package todayhttp: `GET /me/today`, `GET /courses/{id}/today` (US-P2-11). Handler mỏng; nghiệp vụ ở internal/today.
package todayhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/today"
)

// Handler là nhóm đường "Hôm nay".
type Handler struct {
	Today *today.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/me/today", h.me)
	r.With(h.Guard(auth.Member)).Get("/courses/{id}/today", h.course)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, today.Scope{})
}

func (h *Handler) course(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	h.serve(w, r, today.Scope{CourseID: id})
}

// serve: dạng phản hồi do VAI TRONG JWT quyết định; không tham số nào của client đổi được nó.
func (h *Handler) serve(w http.ResponseWriter, r *http.Request, scope today.Scope) {
	p := auth.MustFromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	body, err := h.Today.Get(r.Context(), uid, today.Role(p.Role), scope)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		apierr.Write(w, r, apierr.New(http.StatusGatewayTimeout, apierr.DeadlineExceeded))
		return
	case errors.Is(err, context.Canceled):
		return // client đã bỏ đi
	case errors.Is(err, today.ErrUserGone):
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	case err != nil:
		h.Log.ErrorContext(r.Context(), "today lỗi", "error", err.Error())
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
		return
	}
	httpx.WriteJSONETag(w, r, jsonRaw(body), "")
}

// jsonRaw cho phép ghi thân đã dựng (và lưu cache) mà không mã hoá lại.
type jsonRaw []byte

// MarshalJSON implements json.Marshaler.
func (j jsonRaw) MarshalJSON() ([]byte, error) { return j, nil }
