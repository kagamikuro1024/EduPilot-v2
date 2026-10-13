package escalationhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/escalation"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/notify"
)

// StaffHandler là route của TA / TEACHER: hàng chờ, chi tiết, nhận / bỏ nhận / trả lời / đóng (SRS 6.2 #4–#10) và đọc thông báo theo liên kết (6.3 #15a).
type StaffHandler struct {
	Svc    *escalation.Service
	Notify *notify.Service
	Guard  func(auth.GuardMode) func(http.Handler) http.Handler
	Idem   func(http.Handler) http.Handler
	Log    *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware. Guard StaffRole: ADMIN không bao giờ qua (D39), sinh viên → 403.
func (h *StaffHandler) Mount(r chi.Router) {
	staff := h.Guard(auth.StaffRole)
	const c = "/courses/{id}/tickets"
	r.With(staff).Get(c, h.list)
	r.With(staff).Get(c+"/{tid}", h.get)
	r.With(staff).Post(c+"/{tid}/claim", h.claim)
	r.With(staff).Post(c+"/{tid}/release", h.release)
	r.With(staff, h.Idem).Post(c+"/{tid}/answer", h.answer)
	r.With(staff).Post(c+"/{tid}/dismiss", h.dismiss)
	r.Get("/me/tickets", h.mine)
	r.Post("/notifications/read", h.readByLink)
}

func (h *StaffHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "escalation: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

type reqCtx struct {
	a      escalation.Actor
	course uuid.UUID
	ticket uuid.UUID
}

// ctxOf đọc người gọi (vai TRONG LỚP từ guard), lớp và ticket (nếu có) từ đường dẫn.
func (h *StaffHandler) ctxOf(w http.ResponseWriter, r *http.Request, withTicket bool) (reqCtx, bool) {
	var x reqCtx
	p, ok := auth.FromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return x, false
	}
	ca, _ := auth.CourseFromContext(r.Context())
	x.a = escalation.Actor{UserID: uid, Role: ca.CourseRole}
	if x.course, err = uuid.Parse(chi.URLParam(r, "id")); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return x, false
	}
	if withTicket {
		if x.ticket, err = uuid.Parse(chi.URLParam(r, "tid")); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return x, false
		}
	}
	return x, true
}

func (h *StaffHandler) list(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ctxOf(w, r, false)
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	out, err := h.Svc.ListStaff(r.Context(), []uuid.UUID{x.course}, r.URL.Query().Get("status"), pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// mine là `GET /me/tickets`: mọi lớp mình đang ACTIVE là TEACHER / TA. STUDENT, ADMIN → 403 (không có đường vào hàng chờ).
func (h *StaffHandler) mine(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.FromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	if p.Role != auth.RoleTeacher && p.Role != auth.RoleTA {
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden))
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	courses, err := h.Svc.MyCourseIDs(r.Context(), uid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.Svc.ListStaff(r.Context(), courses, r.URL.Query().Get("status"), pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *StaffHandler) get(w http.ResponseWriter, r *http.Request) {
	x, ok := h.ctxOf(w, r, true)
	if !ok {
		return
	}
	d, err := h.Svc.GetDetail(r.Context(), x.course, x.ticket)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

type versionBody struct {
	Version *int32 `json:"version"`
	Body    string `json:"body"`
	Reason  string `json:"reason"`
}

// optBody đọc thân JSON tuỳ chọn (claim / release có thể không có thân).
func optBody(w http.ResponseWriter, r *http.Request) (versionBody, bool) {
	var b versionBody
	raw, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		return b, false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return b, true
	}
	if json.Unmarshal(raw, &b) != nil {
		apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		return b, false
	}
	return b, true
}

func (h *StaffHandler) action(w http.ResponseWriter, r *http.Request, do func(x reqCtx, b versionBody) (any, error)) {
	x, ok := h.ctxOf(w, r, true)
	if !ok {
		return
	}
	b, ok := optBody(w, r)
	if !ok {
		return
	}
	out, err := do(x, b)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *StaffHandler) claim(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, func(x reqCtx, b versionBody) (any, error) {
		t, err := h.Svc.Claim(r.Context(), x.a, x.course, x.ticket, b.Version)
		return map[string]any{"ticket": t}, err
	})
}

func (h *StaffHandler) release(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, func(x reqCtx, b versionBody) (any, error) {
		t, err := h.Svc.Release(r.Context(), x.a, x.course, x.ticket, b.Version)
		return map[string]any{"ticket": t}, err
	})
}

func (h *StaffHandler) answer(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, func(x reqCtx, b versionBody) (any, error) {
		t, m, err := h.Svc.Answer(r.Context(), x.a, x.course, x.ticket, b.Body, b.Version)
		return map[string]any{"ticket": t, "message": m}, err
	})
}

func (h *StaffHandler) dismiss(w http.ResponseWriter, r *http.Request) {
	h.action(w, r, func(x reqCtx, b versionBody) (any, error) {
		t, err := h.Svc.Dismiss(r.Context(), x.a, x.course, x.ticket, b.Reason, b.Version)
		return map[string]any{"ticket": t}, err
	})
}

// readByLink là `POST /notifications/read {link}` (6.3 #15a): đánh dấu đã đọc thông báo CỦA CHÍNH MÌNH có `link` trùng; link rỗng → 422.
func (h *StaffHandler) readByLink(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.FromContext(r.Context())
	uid, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	var in struct {
		Link string `json:"link"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Link) == "" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "link", Code: "required", Message: "Thiếu liên kết."}))
		return
	}
	n, err := h.Notify.MarkReadByLink(r.Context(), uid, in.Link)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": n})
}
