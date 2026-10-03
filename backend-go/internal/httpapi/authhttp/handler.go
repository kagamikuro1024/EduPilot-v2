// Package authhttp: các đường /auth/* (US-P2-02…). Handler mỏng: logic ở internal/auth.
package authhttp

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/config"
)

// CookieName là tên cookie refresh; Path hẹp để trình duyệt chỉ gửi nó tới /auth/* (SRS 6.3).
const (
	CookieName = "ep_rt"
	CookiePath = "/api/v1/auth"
)

// Handler là nhóm đường /auth/*.
type Handler struct {
	Sessions *auth.Sessions
	Cfg      config.Config
	Log      *slog.Logger
	// ClientIP trả IP đã kiểm chứng của request (theo TRUSTED_PROXY_CIDRS); nil → không ghi IP.
	ClientIP func(*http.Request) string
}

// Mount đăng ký các đường công khai (không cần Bearer) dưới /auth. Mọi phản hồi: no-store.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Use(noStore)
		r.Post("/login", h.login)
		r.With(h.cookieGuard).Post("/refresh", h.refresh)
		r.With(h.cookieGuard).Post("/logout", h.logout)
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}

type loginBody struct {
	Email    string `json:"email" validate:"required,max=320"`
	Password string `json:"password" validate:"required,max=1024"`
}

type sessionBody struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresIn   int           `json:"expires_in"`
	User        auth.UserInfo `json:"user"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var b loginBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	res, err := h.Sessions.Login(r.Context(), auth.LoginInput{
		Email: b.Email, Password: b.Password, UserAgent: r.UserAgent(), IP: h.ip(r),
	})
	switch {
	case err == nil:
		h.writeSession(w, res)
	case errors.Is(err, auth.ErrInvalidCredentials):
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.InvalidCredentials))
	case errors.Is(err, auth.ErrAccountDisabled):
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.AccountDisabled))
	default:
		h.internal(w, r, "login", err)
	}
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		h.clearCookie(w)
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return
	}
	res, err := h.Sessions.Refresh(r.Context(), c.Value, r.UserAgent(), h.ip(r))
	var revoked *auth.SessionRevokedError
	switch {
	case err == nil:
		h.writeSession(w, res)
	case errors.As(err, &revoked):
		h.clearCookie(w)
		e := apierr.New(http.StatusUnauthorized, apierr.SessionRevoked)
		if revoked.Reason != "" {
			e = e.WithDetails(map[string]string{"reason": revoked.Reason})
		}
		apierr.Write(w, r, e)
	case errors.Is(err, auth.ErrRefreshInvalid):
		h.clearCookie(w)
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.TokenInvalid))
	default:
		h.internal(w, r, "refresh", err)
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		if err := h.Sessions.Logout(r.Context(), c.Value); err != nil {
			h.internal(w, r, "logout", err)
			return
		}
	}
	h.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeSession(w http.ResponseWriter, res auth.Result) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: res.RefreshToken, Path: CookiePath, Domain: h.Cfg.CookieDomain,
		MaxAge: int(res.RefreshTTL.Seconds()), HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
	httpx.WriteJSON(w, http.StatusOK, sessionBody{AccessToken: res.AccessToken, TokenType: "Bearer", ExpiresIn: res.ExpiresIn, User: res.User})
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: CookiePath, Domain: h.Cfg.CookieDomain,
		MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) ip(r *http.Request) string {
	if h.ClientIP == nil {
		return ""
	}
	return h.ClientIP(r)
}

func (h *Handler) internal(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.Log.ErrorContext(r.Context(), "auth: lỗi nội bộ", "op", op, "error", err.Error())
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

// cookieGuard là lớp CSRF của endpoint dùng cookie (SRS 4.1): kiểm nguồn gốc rồi kiểu nội dung.
//   - Origin có mặt → phải thuộc CORS_ORIGINS hoặc cùng origin với APP_PUBLIC_URL;
//   - không có Origin mà Sec-Fetch-Site là cross-site / same-site → 403;
//   - không có cả hai → client không phải trình duyệt → cho qua;
//   - Content-Type nếu có phải là application/json (form / text/plain đơn giản bị chặn); thân rỗng không cần header.
//
// Cookie SameSite=Lax là lớp thứ hai.
func (h *Handler) cookieGuard(next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, o := range h.Cfg.CORSOrigins {
		allowed[o] = struct{}{}
	}
	if u, err := url.Parse(h.Cfg.AppPublicURL); err == nil && u.Host != "" {
		allowed[u.Scheme+"://"+u.Host] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" {
			if _, ok := allowed[o]; !ok {
				forbiddenOrigin(w, r)
				return
			}
		} else if sfs := r.Header.Get("Sec-Fetch-Site"); sfs == "cross-site" || sfs == "same-site" {
			forbiddenOrigin(w, r)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			if mt, _, err := mime.ParseMediaType(ct); err != nil || !strings.EqualFold(mt, "application/json") {
				apierr.Write(w, r, apierr.New(http.StatusUnsupportedMediaType, apierr.UnsupportedMediaType))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func forbiddenOrigin(w http.ResponseWriter, r *http.Request) {
	apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "origin"}))
}
