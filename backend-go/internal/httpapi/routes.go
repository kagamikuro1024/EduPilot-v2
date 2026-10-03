package httpapi

import (
	"net/http"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/httpapi/authhttp"
	"github.com/edupilot/backend-go/internal/httpapi/coursehttp"
	"github.com/edupilot/backend-go/internal/httpapi/llmhttp"
	"github.com/edupilot/backend-go/internal/httpapi/userhttp"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/go-chi/chi/v5"
)

// registerAPIRoutes đăng ký nhóm route nghiệp vụ dưới `/api/v1` (nhóm này đã có deadline + giới hạn thân).
// Phần còn lại của thứ tự SRS 3.1: rate limit → auth (→ RBAC / CourseAccessGuard của từng route)
// → `Idempotency-Key` (chỉ handler khai báo RequireIdempotencyKey) → handler.
func registerAPIRoutes(r chi.Router, d Deps) {
	r.Use(rateLimitMiddleware(d))

	mw := []auth.MiddlewareOption{}
	var authAPI *authhttp.Handler
	if d.Sessions != nil {
		// US-P2-02: các đường /auth/* công khai (không Bearer) và kiểm thu hồi cho mọi API có Bearer.
		authAPI = &authhttp.Handler{
			Sessions: d.Sessions, Accounts: d.Accounts, Verify: d.Verifier.Verify, Limiter: auth.NewLimiter(d.Redis, d.now, d.Log), Limits: authLimits(d.Cfg), Cfg: d.Cfg, Log: d.Log, ClientIP: func(r *http.Request) string { return clientIP(r, d) },
		}
		authAPI.Mount(r)
		mw = append(mw, auth.WithRevocation(d.Sessions))
	}
	if d.Cfg.AppEnv == "production" {
		mw = append(mw, auth.RequireSession()) // token dev (không sid) bị từ chối ở production
	}

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(d.Verifier, mw...))
		if authAPI != nil {
			authAPI.MountMe(r) // US-P2-04: /me/password, /me/sessions*
		}
		if d.Jobs != nil {
			// US-PG-03 FR-35/36 — handler việc dài của internal/jobs (chủ job hoặc ADMIN, người khác 404).
			r.Get("/jobs/{id}", jobs.Handler(d.Jobs))
		}
		if d.Users != nil {
			// US-P2-06 — /admin/users: chỉ ADMIN, POST cần Idempotency-Key.
			uh := &userhttp.Handler{Users: d.Users, Log: d.Log}
			uh.Mount(r, RequireIdempotencyKey(d))
			uh.MountMe(r) // US-P2-07: /me/profile, /me/settings
		}
		// CourseAccessGuard thật tra `enrollments` (không cache); không có DB ⇒ DenyAll.
		var resolver auth.CourseResolver = auth.DenyAll{}
		if d.DB != nil {
			resolver = course.Resolver{Pool: d.DB}
		}
		courseGuard := func(m auth.GuardMode) func(http.Handler) http.Handler { return auth.CourseAccessGuard(resolver, m) }
		if d.DB != nil {
			// US-P2-07 — lớp của tôi và chi tiết lớp.
			(&coursehttp.Handler{Courses: course.Service{Pool: d.DB, Production: d.Cfg.AppEnv == "production"}, Guard: courseGuard, Idem: RequireIdempotencyKey(d), Log: d.Log}).Mount(r)
		}
		if d.LLM != nil && d.Redis != nil {
			// US-P1-04 — API cấu hình LLM: 8 đường dẫn / 13 thao tác; RBAC từng route, Idempotency-Key cho POST providers.
			llmhttp.NewAdmin(d.LLM, d.Redis.Client, d.Log, d.Clock).Mount(r, RequireIdempotencyKey(d))
		}
	})

	registerTestRoutes(r, d)
}
