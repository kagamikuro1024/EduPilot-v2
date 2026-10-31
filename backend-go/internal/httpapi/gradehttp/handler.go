// Package gradehttp: route công thức điểm và sổ điểm (FEAT-gradebook SRS 6). Handler mỏng, nghiệp vụ ở internal/grade và internal/grading.
package gradehttp

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	r.With(staff).Get("/courses/{id}/gradebook", h.gradebook)
	r.With(staff).Get("/courses/{id}/gradebook/preview", h.preview)
	r.With(teacher).Put("/courses/{id}/grade-entries", h.putEntry)
	r.With(teacher).Post("/courses/{id}/grade-items/{itemId}/publish", h.publish)
	r.With(teacher).Post("/courses/{id}/grade-entries/import", h.importXLSX)
	r.With(teacher).Get("/courses/{id}/gradebook/export.xlsx", h.export)
	r.With(staff).Get("/courses/{id}/gradebook/students/{uid}/explain", h.explain)
	r.With(staff).Get("/courses/{id}/gradebook/students/{uid}/history", h.history)
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

// limitOf đọc `limit` (1..100, mặc định 30); sai → 422 VALIDATION_FAILED. Con trỏ do gói grade giải mã (khoá sắp xếp là chữ, không phải thời gian).
func limitOf(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 30, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 100 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "limit", Code: "range", Message: "limit phải là số nguyên từ 1 đến 100."}))
		return 0, false
	}
	return n, true
}

// gradebook: GET …/gradebook?cursor=&limit= (Staff) — không cache (điểm không cache phía máy chủ).
func (h *Handler) gradebook(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	limit, ok := limitOf(w, r)
	if !ok {
		return
	}
	cursor := r.URL.Query().Get("cursor")
	out, err := h.Grade.Gradebook(r.Context(), cid, cursor, limit, cursor == "")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

// preview: GET …/gradebook/preview?cursor=&limit= (Staff).
func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	limit, ok := limitOf(w, r)
	if !ok {
		return
	}
	out, err := h.Grade.Preview(r.Context(), cid, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

// putEntry: PUT …/grade-entries {item_id, student_id, score, version} (TEACHER).
func (h *Handler) putEntry(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var in grade.PutEntryInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.ItemID == uuid.Nil || in.StudentID == uuid.Nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "item_id", Code: "required", Message: "Thiếu cột điểm hoặc sinh viên."}))
		return
	}
	out, err := h.Grade.PutEntry(r.Context(), actor, cid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// importXLSX: POST …/grade-entries/import {upload_id, item_id, mode, mapping, skip_errors?} (TEACHER). `commit` đòi `Idempotency-Key`; `preview` thì không.
func (h *Handler) importXLSX(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<16))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge))
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	var in grade.ImportInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	run := func(w http.ResponseWriter, r *http.Request) {
		out, err := h.Grade.Import(r.Context(), actor, cid, in)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}
	if in.Mode == "commit" { // thân đã đọc: trả lại cho middleware để băm đúng nội dung yêu cầu
		r.Body = io.NopCloser(bytes.NewReader(body))
		h.Idem(http.HandlerFunc(run)).ServeHTTP(w, r)
		return
	}
	run(w, r)
}

// export: GET …/gradebook/export.xlsx (TEACHER; no-store, attachment, có audit).
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	f, err := h.Grade.Export(r.Context(), actor, cid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="so-diem-`+f.ClassCode+`.xlsx"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(f.Bytes)
}

// publish: POST …/grade-items/{itemId}/publish {published} (TEACHER; idempotent).
func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	item, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var b struct {
		Published *bool `json:"published"`
	}
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	if b.Published == nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "published", Code: "required", Message: "Thiếu published."}))
		return
	}
	pub, err := h.Grade.PublishItem(r.Context(), actor, cid, item, *b.Published)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"item_id": item, "published": pub})
}

func studentOf(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "uid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return uuid.Nil, false
	}
	return id, true
}

// explain: GET …/gradebook/students/{uid}/explain (Staff).
func (h *Handler) explain(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	uid, ok := studentOf(w, r)
	if !ok {
		return
	}
	out, err := h.Grade.Explain(r.Context(), cid, uid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

// history: GET …/gradebook/students/{uid}/history?cursor= (Staff; ≤ 50 mục mới nhất từ audit_log).
func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	cid, ok := courseOf(w, r)
	if !ok {
		return
	}
	uid, ok := studentOf(w, r)
	if !ok {
		return
	}
	var at *time.Time
	var id *uuid.UUID
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, i, err := grade.ParseHistoryCursor(c)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		at, id = &t, &i
	}
	out, err := h.Grade.History(r.Context(), cid, uid, at, id, 50)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}
