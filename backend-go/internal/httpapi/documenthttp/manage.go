package documenthttp

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
)

// MountManage đăng ký quản lý tài liệu của Staff (SRS FEAT-docs-calendar 6, #3–#8, #12; `impact` cho hộp xác nhận xoá).
func (h *Handler) MountManage(r chi.Router) {
	staff, teacher := h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole)
	const c = "/courses/{id}/documents"
	r.With(staff).Get(c, h.list)
	r.With(staff).Get(c+"/stats", h.stats)
	r.With(staff).Get(c+"/{docId}", h.get)
	r.With(staff).Patch(c+"/{docId}", h.patch)
	r.With(teacher).Delete(c+"/{docId}", h.remove)
	r.With(teacher).Get(c+"/{docId}/impact", h.impact)
	r.With(staff).Get(c+"/{docId}/chunks", h.chunks)
	r.With(staff).Patch(c+"/{docId}/chunks/{chunkId}", h.editChunk)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	out, err := h.Svc.List(r.Context(), x.course, document.ListFilter{Type: q.Get("type"), Status: q.Get("status"), Q: q.Get("q")}, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	out, err := h.Svc.Stats(r.Context(), x.course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	out, err := h.Svc.Get(r.Context(), x.course, x.doc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	var in document.PatchIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Patch(r.Context(), x.actor, x.course, x.doc, llm.TraceID(r.Context()), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	if err := h.Svc.Delete(r.Context(), x.actor, x.course, x.doc, llm.TraceID(r.Context())); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) impact(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	out, err := h.Svc.Impact(r.Context(), x.course, x.doc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) chunks(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > httpx.MaxLimit {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "limit", Code: "range", Message: "limit phải là số nguyên từ 1 đến " + strconv.Itoa(httpx.MaxLimit) + "."}))
			return
		}
		limit = n
	}
	after := int32(-1)
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 0 {
			apierr.Write(w, r, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor))
			return
		}
		after = int32(n)
	}
	out, err := h.Svc.Chunks(r.Context(), x.course, x.doc, after, limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) editChunk(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	chunk, err := uuid.Parse(chi.URLParam(r, "chunkId"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.EditChunk(r.Context(), x.actor, x.course, x.doc, chunk, llm.TraceID(r.Context()), in.Text)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
