package httpapi

import (
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/llmhttp"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/go-chi/chi/v5"
)

// registerAPIRoutes đăng ký nhóm route nghiệp vụ dưới `/api/v1` (nhóm này đã có deadline + giới hạn thân).
// Phần còn lại của thứ tự SRS 3.1: rate limit → auth (→ RBAC / CourseAccessGuard của từng route)
// → `Idempotency-Key` (chỉ handler khai báo RequireIdempotencyKey) → handler.
func registerAPIRoutes(r chi.Router, d Deps) {
	r.Use(rateLimitMiddleware(d))

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(d.Verifier))
		if d.Jobs != nil {
			// US-PG-03 FR-35/36 — handler việc dài của internal/jobs (chủ job hoặc ADMIN, người khác 404).
			r.Get("/jobs/{id}", jobs.Handler(d.Jobs))
		}
		if d.LLM != nil && d.Redis != nil {
			// US-P1-04 — API cấu hình LLM: 8 đường dẫn / 13 thao tác; RBAC từng route, Idempotency-Key cho POST providers.
			llmhttp.NewAdmin(d.LLM, d.Redis.Client, d.Log, d.Clock).Mount(r, RequireIdempotencyKey(d))
		}
	})

	registerTestRoutes(r, d)
}
