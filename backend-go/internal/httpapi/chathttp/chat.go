// Package chathttp: route chat riêng (SRS FEAT-private-chat-pii 6, #1–#10). Ba route SSE (#6, #7, #9) nằm NGOÀI nhóm nghiệp vụ (SRS 4.7.0):
// không timeout chung, không RequireIdempotencyKey (middleware đó đệm cả phản hồi), chống trùng bằng UNIQUE ở DB.
package chathttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/store"
)

const (
	heartbeat    = 25 * time.Second
	writeTimeout = 10 * time.Second
	sseBodyMax   = 16 << 10
)

// Handler là nhóm route chat.
type Handler struct {
	Svc   *chat.Service
	Idem  func(http.Handler) http.Handler
	Drain <-chan struct{}
	Log   *slog.Logger
}

// Mount gắn các route JSON vào nhóm nghiệp vụ (đã qua auth.Middleware).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/chat/sessions", h.list)
	r.With(h.Idem).Post("/chat/sessions", h.create)
	r.With(h.Idem).Post("/chat/sessions/from-draft", h.fromDraft)
	r.Delete("/chat/sessions/{sid}", h.del)
	r.Post("/chat/sessions/{sid}/restore", h.restore)
	r.Get("/chat/sessions/{sid}/messages", h.messages)
	r.Post("/chat/messages/{mid}/cancel", h.cancel)
	r.Put("/chat/messages/{mid}/feedback", h.feedback)
}

// MountSSE gắn ba route SSE vào router đã qua auth.Middleware nhưng KHÔNG qua nhóm nghiệp vụ.
func (h *Handler) MountSSE(r chi.Router) {
	r.Post("/chat/sessions/{sid}/messages", h.send)
	r.Get("/chat/messages/{mid}/stream", h.stream)
	r.Post("/chat/messages/{mid}/retry", h.retry)
}

func actor(r *http.Request) (chat.Actor, bool) {
	p, ok := auth.FromContext(r.Context())
	if !ok {
		return chat.Actor{}, false
	}
	uid, err := uuid.Parse(p.Sub)
	if err != nil {
		return chat.Actor{}, false
	}
	return chat.Actor{UserID: uid, Role: p.Role, TraceID: llm.TraceID(r.Context())}, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "chat: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

// ids đọc actor + uuid trong đường dẫn; sai → ghi lỗi và trả false.
func (h *Handler) ids(w http.ResponseWriter, r *http.Request, param string) (chat.Actor, uuid.UUID, bool) {
	a, ok := actor(r)
	if !ok {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return a, uuid.Nil, false
	}
	var id uuid.UUID
	if param != "" {
		var err error
		if id, err = uuid.Parse(chi.URLParam(r, param)); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return a, id, false
		}
	}
	return a, id, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	a, _, ok := h.ids(w, r, "")
	if !ok {
		return
	}
	cid, err := uuid.Parse(r.URL.Query().Get("course_id"))
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "course_id", Code: "required", Message: "Thiếu mã lớp."}))
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	out, err := h.Svc.ListSessions(r.Context(), a, cid, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	a, _, ok := h.ids(w, r, "")
	if !ok {
		return
	}
	var in chat.CreateSessionIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.CreateSession(r.Context(), a, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) del(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "sid")
	if !ok {
		return
	}
	if err := h.Svc.DeleteSession(r.Context(), a, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "sid")
	if !ok {
		return
	}
	out, err := h.Svc.RestoreSession(r.Context(), a, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) messages(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "sid")
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	out, err := h.Svc.Messages(r.Context(), a, id, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "mid")
	if !ok {
		return
	}
	if err := h.Svc.Cancel(r.Context(), a, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) feedback(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "mid")
	if !ok {
		return
	}
	var in struct {
		Value *string `json:"value"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if err := h.Svc.Feedback(r.Context(), a, id, in.Value); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- SSE -------------------------------------------------------------------------------------------------------------------------

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	a, sid, ok := h.ids(w, r, "sid")
	if !ok {
		return
	}
	cmid, err := uuid.Parse(strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "Idempotency-Key", Code: "uuid", Message: "Idempotency-Key phải là UUID."}))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, sseBodyMax)
	var in struct {
		Content string `json:"content" validate:"required"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	m, err := h.Svc.Send(r.Context(), a, sid, cmid, in.Content)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.serve(w, r, m, "")
}

func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "mid")
	if !ok {
		return
	}
	m, err := h.Svc.Retry(r.Context(), a, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.serve(w, r, m, "")
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	a, id, ok := h.ids(w, r, "mid")
	if !ok {
		return
	}
	m, err := h.Svc.Message(r.Context(), a, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.serve(w, r, m, r.Header.Get("Last-Event-ID"))
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, m store.ChatMessage, last string) {
	hd := w.Header()
	hd.Set("Content-Type", "text/event-stream")
	hd.Set("Cache-Control", "no-cache, no-transform")
	hd.Set("X-Accel-Buffering", "no")
	if r.ProtoMajor == 1 {
		hd.Set("Connection", "keep-alive")
	}
	w.WriteHeader(http.StatusOK)
	sw := &sseWriter{w: w, rc: http.NewResponseController(w)}
	if err := sw.raw(": ok\n\n"); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		t := time.NewTicker(heartbeat)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if sw.raw(": ping\n\n") != nil {
					cancel()
					return
				}
			case <-h.Drain:
				cancel()
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	if err := h.Svc.Tail(ctx, sw, m, last); err != nil && ctx.Err() == nil {
		h.Log.WarnContext(ctx, "chat: phát SSE kết thúc lỗi", "error", err)
	}
}

type sseWriter struct {
	mu sync.Mutex
	w  http.ResponseWriter
	rc *http.ResponseController
}

func (s *sseWriter) raw(str string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.rc.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	if _, err := io.WriteString(s.w, str); err != nil {
		return err
	}
	if err := s.rc.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return err
	}
	return nil
}

// Frame ghi một khung `id/event/data` (data là JSON một dòng).
func (s *sseWriter) Frame(id, event string, data []byte) error {
	var b strings.Builder
	if id != "" {
		fmt.Fprintf(&b, "id: %s\n", id)
	}
	fmt.Fprintf(&b, "event: %s\ndata: %s\n\n", event, data)
	return s.raw(b.String())
}

func (h *Handler) fromDraft(w http.ResponseWriter, r *http.Request) {
	a, _, ok := h.ids(w, r, "")
	if !ok {
		return
	}
	var in chat.FromDraftIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.FromDraft(r.Context(), a, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}
