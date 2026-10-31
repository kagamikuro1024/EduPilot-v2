// Package gradehttp: route công thức điểm và sổ điểm (FEAT-gradebook SRS 6). Handler mỏng, nghiệp vụ ở internal/grade và internal/grading.
package gradehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler gắn các route của sổ điểm. Guard `Teacher`: chỉ TEACHER đang dạy lớp (TA, STUDENT, ADMIN → 403; người ngoài lớp → 404).
type Handler struct {
	Grade   *grade.Service
	Extract *grading.Service
	Guard   func(auth.GuardMode) func(http.Handler) http.Handler
	Idem    func(http.Handler) http.Handler // `Idempotency-Key` bắt buộc
	Log     *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	teacher, staff := h.Guard(auth.Teacher), h.Guard(auth.Staff)
	r.With(teacher, h.Idem).Post("/courses/{id}/grade-scheme/extract", h.extract)
	r.With(staff).Get("/courses/{id}/grade-scheme/status", h.status)
	r.With(staff).Get("/courses/{id}/grade-scheme", h.getScheme)
	r.With(teacher).Put("/courses/{id}/grade-scheme", h.putScheme)
	r.With(staff).Post("/courses/{id}/grade-scheme/impact", h.impact)
	r.With(teacher, h.Idem).Post("/courses/{id}/grade-scheme/confirm", h.confirm)
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

// status: GET …/grade-scheme/status (Staff) — đọc mỗi lần, không cache (điểm không cache phía máy chủ).
func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	st, err := h.Grade.Status(r.Context(), cid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, st)
}

// getScheme: GET …/grade-scheme[?diff=confirmed] (Staff) → {confirmed, draft, diff?}. Không cache.
func (h *Handler) getScheme(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	v, err := h.Grade.Schemes(r.Context(), cid, r.URL.Query().Get("diff") == "confirmed")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, v)
}

// putScheme: PUT …/grade-scheme {version, components, rules} (TEACHER) → 200 bản nháp + unclear_items.
func (h *Handler) putScheme(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var in grade.PutInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.Version < 1 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "version", Code: "required", Message: "Thiếu version."}))
		return
	}
	doc, err := h.Grade.PutDraft(r.Context(), actor, cid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, doc)
}

// impact: POST …/grade-scheme/impact {components, rules} (Staff; chỉ đọc).
func (h *Handler) impact(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	var in grade.PutInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Grade.Impact(r.Context(), cid, in.Components, in.Rules)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// confirm: POST …/grade-scheme/confirm {version} (TEACHER, Idempotency-Key) — đường DUY NHẤT gọi ConfirmScheme.
func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var b struct {
		Version int `json:"version" validate:"required,min=1"`
	}
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	out, err := h.Grade.ConfirmScheme(r.Context(), actor, cid, b.Version)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
