// Package userhttp: /admin/users (US-P2-06). Chỉ ADMIN — kể cả GET. Handler mỏng, nghiệp vụ ở internal/user.
package userhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/user"
)

// Handler là nhóm /admin/users.
type Handler struct {
	Users *user.Service
	Log   *slog.Logger
}

// Mount đăng ký 4 thao tác trong nhóm đã qua auth.Middleware. RBAC (ADMIN) chạy TRƯỚC Idempotency-Key để 403 đến trước 422 thiếu khoá.
func (h *Handler) Mount(r chi.Router, idem func(http.Handler) http.Handler) {
	admin := auth.RequireRole(auth.RoleAdmin)
	r.Route("/admin/users", func(r chi.Router) {
		r.With(admin).Get("/", h.list)
		r.With(admin, idem).Post("/", h.invite)
		r.With(admin).Patch("/{id}", h.patch)
		r.With(admin).Post("/{id}/resend-invite", h.resend)
	})
}

type item struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	Version     int        `json:"version"`
}

func itemOf(v user.View) item {
	return item{ID: v.ID, Email: v.Email, FullName: v.FullName, Role: v.Role, Status: v.Status, LastLoginAt: v.LastLoginAt, Version: v.Version}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	f := user.ListFilter{Role: q.Get("role"), Status: q.Get("status"), Q: q.Get("q"), Limit: p.Limit}
	if f.Role != "" && f.Role != "ADMIN" && f.Role != "TEACHER" && f.Role != "TA" && f.Role != "STUDENT" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "role", Code: "INVALID_ROLE", Message: "Vai không hợp lệ."}))
		return
	}
	switch f.Status {
	case "", "PENDING_VERIFICATION", "INVITED", "ACTIVE", "DISABLED":
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "status", Code: "INVALID_STATUS", Message: "Trạng thái không hợp lệ."}))
		return
	}
	if len(f.Q) > 100 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "q", Code: "TOO_LONG", Message: "Từ khoá tối đa 100 ký tự."}))
		return
	}
	if p.Cursor != nil {
		f.Cursor = &user.Cursor{CreatedAt: p.Cursor.CreatedAt, ID: p.Cursor.ID}
	}
	rows, err := h.Users.List(r.Context(), f)
	if err != nil {
		h.internal(w, r, "list", err)
		return
	}
	page := httpx.Paginate(rows, p, func(v user.View) (time.Time, string) { return v.CreatedAt, v.ID.String() })
	out := make([]item, len(page.Items))
	for i, v := range page.Items {
		out[i] = itemOf(v)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[item]{Items: out, NextCursor: page.NextCursor})
}

type inviteBody struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

func (h *Handler) invite(w http.ResponseWriter, r *http.Request) {
	var b inviteBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	v, err := h.Users.Invite(r.Context(), actor, user.InviteInput{Email: b.Email, FullName: b.FullName, Role: b.Role})
	if h.fail(w, r, "invite", err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, itemOf(v))
}

type patchBody struct {
	Version  *int    `json:"version"`
	FullName *string `json:"full_name"`
	Role     *string `json:"role"`
	Status   *string `json:"status"`
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var b patchBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	version, aerr := httpx.WantedVersion(r, b.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	v, err := h.Users.Patch(r.Context(), actor, id, user.PatchInput{Version: version, FullName: b.FullName, Role: b.Role, Status: b.Status})
	if h.fail(w, r, "patch", err) {
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(v.Version))
	httpx.WriteJSON(w, http.StatusOK, itemOf(v))
}

func (h *Handler) resend(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	exp, err := h.Users.ResendInvite(r.Context(), actor, id)
	if h.fail(w, r, "resend-invite", err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]time.Time{"expires_at": exp})
}

func (h *Handler) actor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	p, ok := auth.FromContext(r.Context())
	id, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return uuid.Nil, false
	}
	return id, true
}

// fail ánh xạ lỗi nghiệp vụ sang mã API; true = đã ghi phản hồi lỗi.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, op string, err error) bool {
	if err == nil {
		return false
	}
	var inv *user.InvalidError
	var vc *user.VersionConflictError
	var th *auth.ThrottledError
	switch {
	case errors.As(err, &inv):
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: inv.Field, Code: inv.Code, Message: inv.Message}))
	case errors.Is(err, user.ErrEmailTaken):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"field": "email"}).WithMessage("Email này đã có tài khoản."))
	case errors.Is(err, user.ErrNotFound):
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
	case errors.Is(err, user.ErrNotInvited):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"reason": "not_invited"}).WithMessage("Chỉ gửi lại được lời mời cho tài khoản chưa nhận."))
	case errors.Is(err, user.ErrSelf):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"reason": "self"}).WithMessage("Bạn không thể tự đổi hay tự khoá chính mình."))
	case errors.Is(err, user.ErrLastAdmin):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"reason": "last_admin"}).WithMessage("Không thể khoá quản trị viên cuối cùng."))
	case errors.As(err, &vc):
		httpx.WriteVersionConflict(w, r, vc.Current.Version, itemOf(vc.Current))
	case errors.As(err, &th):
		apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(th.RetryAfter))
	default:
		h.internal(w, r, op, err)
	}
	return true
}

func (h *Handler) internal(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.Log.ErrorContext(r.Context(), "admin/users lỗi", "op", op, "error", err.Error(), "status_hint", strconv.Itoa(http.StatusInternalServerError))
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}
