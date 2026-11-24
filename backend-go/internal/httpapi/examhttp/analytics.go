package examhttp

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// MountAnalytics đăng ký thống kê điểm thi (US-P10-06, SRS FEAT-p10-eval 6 #8–#12). Chỉ bài PUBLISHED; ADMIN chỉ thấy số tổng (không danh tính).
func (h *Handler) MountAnalytics(r chi.Router) {
	staff, staffOrAdmin := h.Guard(auth.StaffRole), h.Guard(auth.StaffOrAdmin)
	const a = "/courses/{id}/analytics/exams"
	r.With(staffOrAdmin).Get(a+"/overview", h.anOverview)
	r.With(staffOrAdmin).Get(a+"/trend", h.anTrend)
	r.With(staff).Get(a+"/hard-items", h.anHardItems)
	r.With(staff).Get(a+"/declining", h.anDeclining)
	r.With(auth.RequireRole(auth.RoleTeacher)).Get("/analytics/exams/compare", h.anCompare)
}

// analyticsCache: điểm không cache phía máy chủ (AGENTS #15) — trình duyệt luôn hỏi lại, số liệu đổi ngay khi công bố / đổi điểm.
func analyticsCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "private, no-cache") }

func isAdmin(r *http.Request) bool { return auth.MustFromContext(r.Context()).Role == auth.RoleAdmin }

func (h *Handler) anCourse(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return uuid.Nil, false
	}
	return id, true
}

func anExamID(w http.ResponseWriter, r *http.Request) (*uuid.UUID, bool) {
	raw := r.URL.Query().Get("exam_id")
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "exam_id", Code: "invalid", Message: "exam_id không hợp lệ."}))
		return nil, false
	}
	return &id, true
}

func anLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return httpx.DefaultLimit, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > httpx.MaxLimit {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "limit", Code: "range", Message: "limit phải là số nguyên từ 1 đến 100."}))
		return 0, false
	}
	return n, true
}

func (h *Handler) anOverview(w http.ResponseWriter, r *http.Request) {
	course, ok := h.anCourse(w, r)
	if !ok {
		return
	}
	eid, ok := anExamID(w, r)
	if !ok {
		return
	}
	out, err := h.Svc.AnalyticsOverview(r.Context(), course, eid, r.URL.Query().Get("week"), !isAdmin(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	analyticsCache(w)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) anTrend(w http.ResponseWriter, r *http.Request) {
	course, ok := h.anCourse(w, r)
	if !ok {
		return
	}
	out, err := h.Svc.AnalyticsTrend(r.Context(), course, !isAdmin(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	analyticsCache(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) anHardItems(w http.ResponseWriter, r *http.Request) {
	course, ok := h.anCourse(w, r)
	if !ok {
		return
	}
	eid, ok := anExamID(w, r)
	if !ok {
		return
	}
	limit, ok := anLimit(w, r)
	if !ok {
		return
	}
	items, next, err := h.Svc.AnalyticsHardItems(r.Context(), course, eid, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	analyticsCache(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[exam.HardItem]{Items: items, NextCursor: next})
}

func (h *Handler) anDeclining(w http.ResponseWriter, r *http.Request) {
	course, ok := h.anCourse(w, r)
	if !ok {
		return
	}
	limit, ok := anLimit(w, r)
	if !ok {
		return
	}
	rows, next, err := h.Svc.AnalyticsDeclining(r.Context(), course, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	analyticsCache(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[exam.DecliningRow]{Items: rows, NextCursor: next})
}

func (h *Handler) anCompare(w http.ResponseWriter, r *http.Request) {
	user, err := uuid.Parse(auth.MustFromContext(r.Context()).Sub)
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	rows, err := h.Svc.AnalyticsCompare(r.Context(), user)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	analyticsCache(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}
