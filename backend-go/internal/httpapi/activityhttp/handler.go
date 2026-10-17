// Package activityhttp: heartbeat, xác nhận minh bạch, đọc thời gian học (FEAT-crm-360 SRS 6.3). Handler mỏng, nghiệp vụ ở internal/activity.
package activityhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/activity"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler: `POST /activity/heartbeat` và `POST /me/tracking-notice/ack` chỉ cần JWT; hai route đọc đi qua CourseAccessGuard (TLR-11).
type Handler struct {
	Svc   *activity.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	r.Post("/activity/heartbeat", h.heartbeat)
	r.Post("/me/tracking-notice/ack", h.ack)
	r.With(h.Guard(auth.Member)).Get("/courses/{id}/me/study-time", h.mine)
	r.With(h.Guard(auth.Staff)).Get("/courses/{id}/students/{uid}/study-time", h.ofStudent)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "activity: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func actorOf(w http.ResponseWriter, r *http.Request) (auth.Principal, uuid.UUID, bool) {
	p, ok := auth.FromContext(r.Context())
	id, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return p, uuid.Nil, false
	}
	return p, id, true
}

type beatBody struct {
	CourseID string `json:"course_id"`
	Area     string `json:"area"`
	TabID    string `json:"tab_id"`
}

func invalid(field, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: field, Code: "invalid", Message: msg})
}

// heartbeat → 204 (kể cả khi nhịp bị bỏ: chưa ack, tắt đo, vượt tốc độ / trần, Redis lỗi); 403 lớp không ACTIVE của sinh viên; 422 thân / area sai. KHÔNG ghi Postgres.
func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	var b beatBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	p, uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, err := uuid.Parse(b.CourseID)
	if err != nil {
		apierr.Write(w, r, invalid("course_id", "Mã lớp không hợp lệ."))
		return
	}
	if _, err := uuid.Parse(b.TabID); err != nil {
		apierr.Write(w, r, invalid("tab_id", "Mã tab không hợp lệ."))
		return
	}
	if err := h.Svc.Heartbeat(r.Context(), uid, string(p.Role), activity.Beat{CourseID: cid, Area: b.Area, TabID: b.TabID}); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ack: idempotent; chỉ STUDENT có ý nghĩa nhưng vai khác nhận 204 im lặng (không lộ gì).
func (h *Handler) ack(w http.ResponseWriter, r *http.Request) {
	_, uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	if err := h.Svc.Ack(r.Context(), uid); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func weeksOf(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("weeks")
	if raw == "" {
		return activity.DefaultWeeks, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		apierr.Write(w, r, invalid("weeks", "weeks phải là số nguyên từ 1 đến 20."))
		return 0, false
	}
	return n, true
}

// mine: STUDENT xem của CHÍNH mình (theo user_id JWT, không có tham số student).
func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	_, uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	if ca, _ := auth.CourseFromContext(r.Context()); ca.CourseRole != auth.RoleStudent {
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden))
		return
	}
	weeks, ok := weeksOf(w, r)
	if !ok {
		return
	}
	out, err := h.Svc.Read(r.Context(), cid, uid, weeks, false)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ofStudent: TA / TEACHER của lớp; STUDENT (kể cả uid của mình) và ADMIN bị guard `Staff` chặn 403.
func (h *Handler) ofStudent(w http.ResponseWriter, r *http.Request) {
	cid, err1 := uuid.Parse(chi.URLParam(r, "id"))
	sid, err2 := uuid.Parse(chi.URLParam(r, "uid"))
	if err1 != nil || err2 != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	weeks, ok := weeksOf(w, r)
	if !ok {
		return
	}
	out, err := h.Svc.Read(r.Context(), cid, sid, weeks, true)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
