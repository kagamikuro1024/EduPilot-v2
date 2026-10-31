// Package gradehttp: route công thức điểm và sổ điểm (FEAT-gradebook SRS 6). Handler mỏng, nghiệp vụ ở internal/grade và internal/grading.
package gradehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler gắn các route của sổ điểm. Guard `Teacher`: chỉ TEACHER đang dạy lớp (TA, STUDENT, ADMIN → 403; người ngoài lớp → 404).
type Handler struct {
	Extract *grading.Service
	Guard   func(auth.GuardMode) func(http.Handler) http.Handler
	Idem    func(http.Handler) http.Handler // `Idempotency-Key` bắt buộc
	Log     *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	teacher := h.Guard(auth.Teacher)
	r.With(teacher, h.Idem).Post("/courses/{id}/grade-scheme/extract", h.extract)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "grade: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func actorOf(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	p, ok := auth.FromContext(r.Context())
	id, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return uuid.Nil, false
	}
	return id, true
}

func courseOf(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return uuid.Nil, false
	}
	return id, true
}

// extract: POST …/grade-scheme/extract {document_id, replace_draft?} → 202 {job_id}.
func (h *Handler) extract(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var b struct {
		DocumentID   string `json:"document_id" validate:"required,uuid"`
		ReplaceDraft bool   `json:"replace_draft"`
	}
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	doc, _ := uuid.Parse(b.DocumentID)
	id, err := h.Extract.Request(r.Context(), actor, cid, doc, b.ReplaceDraft)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"job_id": id})
}
