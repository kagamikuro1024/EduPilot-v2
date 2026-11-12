// Package practicehttp: REST của luyện đề và thi thử (SRS FEAT-practice mục 6, #5–#14). Handler mỏng; nghiệp vụ ở internal/practice. Chỉ STUDENT ACTIVE của lớp
// (staff / ADMIN → 403 ở guard); lượt của người khác → 404 từ service.
package practicehttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/practice"
)

// Handler là nhóm đường luyện đề của sinh viên.
type Handler struct {
	Svc   *practice.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Idem  func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware. Đường tĩnh (`overview`, `start`, `history`, `weakness`) không đụng `attempts/{aid}`.
func (h *Handler) Mount(r chi.Router) {
	student := h.Guard(auth.StudentRole)
	const p = "/courses/{id}/practice"
	r.With(student).Get(p+"/overview", h.overview)
	r.With(student, h.Idem).Post(p+"/start", h.start)
	r.With(student).Get(p+"/history", h.history)
	r.With(student).Get(p+"/weakness", h.weakness)
	r.With(student).Get(p+"/attempts/{aid}", h.get)
	r.With(student).Patch(p+"/attempts/{aid}", h.patch)
	r.With(student).Put(p+"/attempts/{aid}/answers/{itemId}", h.saveAnswer)
	r.With(student, h.Idem).Post(p+"/attempts/{aid}/submit", h.submit)
	r.With(student).Post(p+"/attempts/{aid}/items/{itemId}/explain", h.explain)
	r.With(student, h.Idem).Post(p+"/attempts/{aid}/items/{itemId}/feedback", h.feedback)
}

type ids struct {
	user, course, attempt, item uuid.UUID
	tab                         uuid.UUID
}

// parse đọc người gọi, lớp, lượt, mục (nếu có trong đường dẫn) và `X-Practice-Tab`. Id sai định dạng → 404 (không lộ tồn tại).
func (h *Handler) parse(w http.ResponseWriter, r *http.Request) (ids, bool) {
	var c ids
	p := auth.MustFromContext(r.Context())
	var err error
	if c.user, err = uuid.Parse(p.Sub); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return c, false
	}
	nf := func() (ids, bool) {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return c, false
	}
	if c.course, err = uuid.Parse(chi.URLParam(r, "id")); err != nil {
		return nf()
	}
	if s := chi.URLParam(r, "aid"); s != "" {
		if c.attempt, err = uuid.Parse(s); err != nil {
			return nf()
		}
	}
	if s := chi.URLParam(r, "itemId"); s != "" {
		if c.item, err = uuid.Parse(s); err != nil {
			return nf()
		}
	}
	if raw := r.Header.Get("X-Practice-Tab"); raw != "" {
		if c.tab, err = uuid.Parse(raw); err != nil {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "X-Practice-Tab", Code: "VALUE_REQUIRED", Message: "Mã tab (X-Practice-Tab) phải là uuid."}))
			return c, false
		}
	}
	return c, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "practice: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.Overview(r.Context(), c.user, c.course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	var in practice.StartIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, created, err := h.Svc.Start(r.Context(), c.user, c.course, in, c.tab)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, v)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.Get(r.Context(), c.user, c.course, c.attempt, c.tab)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

// patchIn là thân `PATCH …/attempts/{aid}`.
type patchIn struct {
	CurrentIndex *int `json:"current_index" validate:"required"`
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	var in patchIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.SetIndex(r.Context(), c.user, c.course, c.attempt, c.tab, *in.CurrentIndex)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

// answersMaxBytes: thân `PUT …/answers/{itemId}` ≤ 32 KiB (tự luận tối đa 4.000 ký tự).
const answersMaxBytes = 32 << 10

func (h *Handler) saveAnswer(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, answersMaxBytes)
	var in practice.AnswerIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.SaveAnswer(r.Context(), c.user, c.course, c.attempt, c.item, c.tab, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.Submit(r.Context(), c.user, c.course, c.attempt)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var cur *practice.Cursor
	if pp.Cursor != nil {
		cur = &practice.Cursor{At: pp.Cursor.CreatedAt, ID: pp.Cursor.ID}
	}
	rows, err := h.Svc.History(r.Context(), c.user, c.course, cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pp, func(x practice.HistoryRow) (time.Time, string) { return x.StartedAt, x.ID.String() }))
}

func (h *Handler) weakness(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.Weakness(r.Context(), c.user, c.course)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

// explain là SSE (#11): 409 NOT_ANSWERED / NOT_SUBMITTED trả JSON TRƯỚC khi mở luồng; từ lúc mở luồng, lỗi LLM chỉ là `event: error` (phản hồi đúng / sai không bị ảnh hưởng).
func (h *Handler) explain(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	job, err := h.Svc.PrepareExplain(r.Context(), c.user, c.course, c.attempt, c.item)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", "text/event-stream")
	hd.Set("Cache-Control", "no-cache, no-transform")
	hd.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	frame := func(event string, v any) error {
		b, _ := json.Marshal(v)
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b); err != nil {
			return err
		}
		if err := rc.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return err
		}
		return nil
	}
	if err := frame("ready", map[string]bool{"cached": job.Cached != ""}); err != nil {
		return
	}
	if err := job.Run(r.Context(), func(s string) error { return frame("token", map[string]string{"text": s}) }); err != nil {
		if r.Context().Err() == nil {
			h.Log.WarnContext(r.Context(), "practice: giải thích lỗi", "error", err.Error())
			_ = frame("error", map[string]string{"code": "EXPLAIN_FAILED"})
		}
		return
	}
	_ = frame("done", map[string]bool{"ok": true})
}

func (h *Handler) feedback(w http.ResponseWriter, r *http.Request) {
	c, ok := h.parse(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.Feedback(r.Context(), c.user, c.course, c.attempt, c.item)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}
