package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// CourseURLParam là tên tham số đường dẫn chứa id lớp học.
const CourseURLParam = "courseId"

// forbiddenDetails là `details` của 403 (SRS 6.1): reason = "role" hoặc "course".
type forbiddenDetails struct {
	Reason string `json:"reason"`
}

// RevocationChecker cho biết một access token (đã qua chữ ký) thuộc phiên / người dùng đã bị thu hồi chưa.
// reason (chữ thường, "logout", "password_changed"…) chỉ có khi biết.
type RevocationChecker interface {
	Revoked(ctx context.Context, p Principal) (revoked bool, reason string)
}

type mwConfig struct {
	rev         RevocationChecker
	needSession bool
}

// MiddlewareOption tinh chỉnh Middleware.
type MiddlewareOption func(*mwConfig)

// WithRevocation bật kiểm thu hồi (US-P2-02 AC8): phiên / người dùng bị thu hồi → 401 SESSION_REVOKED.
func WithRevocation(c RevocationChecker) MiddlewareOption { return func(m *mwConfig) { m.rev = c } }

// RequireSession từ chối token không có `sid` (token dev của `gateway token`) — dùng ở production.
func RequireSession() MiddlewareOption { return func(m *mwConfig) { m.needSession = true } }

// Middleware xác thực Bearer token rồi gắn Principal vào context.
// Phân loại 401 (SRS 6.1): thiếu/sai kiểu header → UNAUTHENTICATED; hết hạn → TOKEN_EXPIRED; thu hồi → SESSION_REVOKED;
// còn lại → TOKEN_INVALID.
func Middleware(v *Verifier, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	var cfg mwConfig
	for _, o := range opts {
		o(&cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w, r, apierr.Unauthenticated)
				return
			}
			p, err := v.Verify(raw)
			if err != nil {
				code := apierr.TokenInvalid
				if errors.Is(err, ErrTokenExpired) {
					code = apierr.TokenExpired
				}
				writeUnauthorized(w, r, code)
				return
			}
			if cfg.needSession && p.SessionID == "" {
				writeUnauthorized(w, r, apierr.TokenInvalid)
				return
			}
			if cfg.rev != nil {
				if revoked, reason := cfg.rev.Revoked(r.Context(), p); revoked {
					w.Header().Set("WWW-Authenticate", `Bearer realm="edupilot", error="invalid_token"`)
					e := apierr.New(http.StatusUnauthorized, apierr.SessionRevoked)
					if reason != "" {
						e = e.WithDetails(map[string]string{"reason": reason})
					}
					apierr.Write(w, r, e)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

// bearerToken tách token khỏi header Authorization; tên scheme không phân biệt hoa thường (RFC 7235).
func bearerToken(h string) (string, bool) {
	const scheme = "bearer"
	if len(h) <= len(scheme) || h[len(scheme)] != ' ' || !strings.EqualFold(h[:len(scheme)], scheme) {
		return "", false
	}
	t := strings.TrimSpace(h[len(scheme)+1:])
	return t, t != ""
}

// writeUnauthorized trả 401 kèm WWW-Authenticate (SRS 6.5); message là chuỗi cố định của mã (SRS 6.1).
func writeUnauthorized(w http.ResponseWriter, r *http.Request, code string) {
	challenge := `Bearer realm="edupilot"`
	if code != apierr.Unauthenticated {
		challenge += `, error="invalid_token"`
	}
	w.Header().Set("WWW-Authenticate", challenge)
	apierr.Write(w, r, apierr.New(http.StatusUnauthorized, code))
}

// RequireRole chỉ cho qua các vai liệt kê; vai lấy từ claim (claim thắng DB — 04-AC6).
// Chưa xác thực → 401; sai vai → 403 FORBIDDEN với details.reason = "role".
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	allowed := make(map[Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok {
				writeUnauthorized(w, r, apierr.Unauthenticated)
				return
			}
			if _, ok := allowed[p.Role]; !ok {
				writeForbidden(w, r, "role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeForbidden(w http.ResponseWriter, r *http.Request, reason string) {
	apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).
		WithDetails(forbiddenDetails{Reason: reason}))
}

// CourseResolver quyết định một Principal có được vào một lớp hay không.
// P2 cài bản tra bảng `enrollments`; PG chỉ có khung.
type CourseResolver interface {
	CanAccess(ctx context.Context, p Principal, courseID string) (bool, error)
}

// DenyAll là resolver mặc định của PG: từ chối tất cả, KỂ CẢ ADMIN (FR-41, PRD §3 — ADMIN không thấy nội dung lớp).
type DenyAll struct{}

// CanAccess luôn từ chối.
func (DenyAll) CanAccess(context.Context, Principal, string) (bool, error) { return false, nil }

// CourseAccessGuard chặn route có `{courseId}`: id không phải uuid → 404, resolver lỗi → 503,
// từ chối → 403 (details.reason = "course"). Gọi resolver đúng 1 lần mỗi request và KHÔNG cache kết quả (FR-41).
func CourseAccessGuard(resolver CourseResolver) func(http.Handler) http.Handler {
	if resolver == nil {
		resolver = DenyAll{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok {
				writeUnauthorized(w, r, apierr.Unauthenticated)
				return
			}
			id := chi.URLParam(r, CourseURLParam)
			if _, err := uuid.Parse(id); err != nil {
				apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
				return
			}
			allowed, err := resolver.CanAccess(r.Context(), p, id)
			if err != nil {
				apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
				return
			}
			if !allowed {
				writeForbidden(w, r, "course")
				return
			}
			// ponytail: CourseRole = vai JWT vì PG chưa có `enrollments`; P2 cho resolver trả vai trong lớp.
			ctx := WithCourseAccess(r.Context(), CourseAccess{CourseID: id, CourseRole: p.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
