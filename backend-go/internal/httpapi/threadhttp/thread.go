// Package threadhttp: route Threads (SRS FEAT-private-chat-pii 6, #12–#20). Quyền theo guard của từng route (Member không gồm ADMIN; Staff cho quyết định).
package threadhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/thread"
)

// Handler là nhóm route Threads.
type Handler struct {
	Svc   *thread.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Idem  func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	member, staff := h.Guard(auth.MemberRole), h.Guard(auth.StaffRole)
	const c = "/courses/{id}"
	r.With(member).Get(c+"/threads", h.list)
	r.With(member).Post(c+"/threads/precheck", h.precheck)
	r.With(member, h.Idem).Post(c+"/threads", h.create)
	r.With(member).Get(c+"/threads/{tid}", h.get)
	r.With(member).Get(c+"/threads/{tid}/similar", h.similar)
	r.With(member, h.Idem).Post(c+"/threads/{tid}/posts", h.comment)
	r.With(staff).Post(c+"/posts/{pid}/verify", h.decide(thread.Verify))
	r.With(staff).Put(c+"/posts/{pid}/correct", h.decide(thread.Correct))
	r.With(staff).Post(c+"/posts/{pid}/reject", h.decide(thread.Reject))
}

type ctxIDs struct {
	a      thread.Actor
	course uuid.UUID
}

func (h *Handler) ids(w http.ResponseWriter, r *http.Request) (ctxIDs, bool) {
	var x ctxIDs
	p, ok := auth.FromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return x, false
	}
	ca, _ := auth.CourseFromContext(r.Context())
	if x.course, err = uuid.Parse(chi.URLParam(r, "id")); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return x, false
	}
	x.a = thread.Actor{UserID: uid, Role: ca.CourseRole, TraceID: llm.TraceID(r.Context())}
	return x, true
}

func param(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return id, false
	}
	return id, true
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "thread: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r)
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	f := thread.Filter{Tag: q.Get("tag"), State: q.Get("state"), Q: q.Get("q")}
	if f.State != "" && f.State != "pending" && f.State != "verified" && f.State != "none" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "state", Code: "enum", Message: "state phải là pending, verified hoặc none."}))
		return
	}
	if v := q.Get("week"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 20 {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "week", Code: "range", Message: "week từ 1 đến 20."}))
			return
		}
		f.Week = &n
	}
	out, err := h.Svc.List(r.Context(), x.a, x.course, f, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r)
	if !ok {
		return
	}
	tid, ok := param(w, r, "tid")
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	out, err := h.Svc.Get(r.Context(), x.a, x.course, tid, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) similar(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r)
	if !ok {
		return
	}
	tid, ok := param(w, r, "tid")
	if !ok {
		return
	}
	items, err := h.Svc.Similar(r.Context(), x.course, tid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) precheck(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r)
	if !ok {
		return
	}
	var in struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Precheck(r.Context(), x.a, x.course, in.Title, in.Body)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r)
	if !ok {
		return
	}
	var in thread.CreateIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Create(r.Context(), x.a, x.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) comment(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ids(w, r)
	if !ok {
		return
	}
	tid, ok := param(w, r, "tid")
	if !ok {
		return
	}
	var in thread.CommentIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.Comment(r.Context(), x.a, x.course, tid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) decide(d thread.Decision) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		x, ok := h.ids(w, r)
		if !ok {
			return
		}
		pid, ok := param(w, r, "pid")
		if !ok {
			return
		}
		var in struct {
			Body    string `json:"body"`
			Version int32  `json:"version"`
		}
		if d == thread.Correct {
			if !httpx.DecodeJSON(w, r, &in) {
				return
			}
		}
		out, err := h.Svc.Decide(r.Context(), x.a, x.course, pid, d, in.Body, in.Version)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}
}
