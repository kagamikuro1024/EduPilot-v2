package httpapi

import (
	"net/http"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/document"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/authhttp"
	"github.com/edupilot/backend-go/internal/httpapi/chathttp"
	"github.com/edupilot/backend-go/internal/httpapi/coursehttp"
	"github.com/edupilot/backend-go/internal/httpapi/documenthttp"
	"github.com/edupilot/backend-go/internal/httpapi/examhttp"
	"github.com/edupilot/backend-go/internal/httpapi/llmhttp"
	"github.com/edupilot/backend-go/internal/httpapi/threadhttp"
	"github.com/edupilot/backend-go/internal/httpapi/todayhttp"
	"github.com/edupilot/backend-go/internal/httpapi/userhttp"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/today"
	"github.com/go-chi/chi/v5"
)

// registerAPIRoutes đăng ký nhóm route nghiệp vụ dưới `/api/v1` (nhóm này đã có deadline + giới hạn thân).
// Phần còn lại của thứ tự SRS 3.1: rate limit → auth (→ RBAC / CourseAccessGuard của từng route)
// → `Idempotency-Key` (chỉ handler khai báo RequireIdempotencyKey) → handler.
func registerAPIRoutes(r chi.Router, d Deps) {
	r.Use(rateLimitMiddleware(d))

	mw := authOptions(d)
	var authAPI *authhttp.Handler
	if d.Sessions != nil {
		// US-P2-02: các đường /auth/* công khai (không Bearer) và kiểm thu hồi cho mọi API có Bearer.
		authAPI = &authhttp.Handler{
			Sessions: d.Sessions, Accounts: d.Accounts, Verify: d.Verifier.Verify, Limiter: auth.NewLimiter(d.Redis, d.now, d.Log), Limits: authLimits(d.Cfg), Cfg: d.Cfg, Log: d.Log, ClientIP: func(r *http.Request) string { return clientIP(r, d) },
		}
		authAPI.Mount(r)
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
		if d.DB != nil {
			// US-PE-07 — GET /me/exam-lock (thao tác 43): chat AI có đang bị khoá vì lượt làm bài không (chỉ chính mình, không lộ tên bài).
			r.Get("/me/exam-lock", examhttp.MyLock(&exam.Locker{Pool: d.DB, Redis: d.Redis, Clock: d.Clock, Grace: time.Duration(d.Cfg.ExamGraceSeconds) * time.Second}))
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
			svc := course.NewService(course.Service{Pool: d.DB, Production: d.Cfg.AppEnv == "production", Clock: d.Clock, Redis: d.Redis, Log: d.Log, PublicURL: d.Cfg.AppPublicURL})
			var sig today.LLMSignals
			if d.LLM != nil {
				sig = todayLLM{rt: d.LLM}
			}
			(&todayhttp.Handler{Today: today.NewService(d.DB, d.Redis, d.Clock, d.Log, sig), Guard: courseGuard, Log: d.Log}).Mount(r)
			(&coursehttp.Handler{Courses: svc, Guard: courseGuard, Idem: RequireIdempotencyKey(d), OptIdem: OptionalIdempotencyKey(d), ClientIP: func(r *http.Request) string { return clientIP(r, d) }, Log: d.Log}).Mount(r)
		}
		if d.DB != nil && d.Jobs != nil {
			// US-PE-03 / PE-04 — ngân hàng câu hỏi và bài thi (thao tác 1–25, 27, 28 của SRS FEAT-weekly-exam 6.2).
			svc := &exam.Service{Pool: d.DB, Clock: d.Clock, Jobs: d.Jobs, ZipMaxUncompressed: d.Cfg.ExamTestZipMaxUncompressed,
				Limits:    exam.Limits{MinDurationMinutes: d.Cfg.ExamMinDurationMinutes, MinLeadSeconds: d.Cfg.ExamMinLeadSeconds, MaxTotalSeconds: d.Cfg.JudgeMaxTotalSeconds},
				Attempt:   exam.AttemptConfig{Grace: time.Duration(d.Cfg.ExamGraceSeconds) * time.Second, TabStale: d.Cfg.ExamTabStale, SaveRate: d.Cfg.ExamSaveRatePerMin},
				Code:      exam.CodeConfig{RunLimit: d.Cfg.ExamRunLimit, RunWindow: d.Cfg.ExamRunWindow, SubmitCooldown: d.Cfg.ExamSubmitCooldown, SubmissionCap: d.Cfg.ExamSubmissionCap},
				Integrity: exam.IntegrityConfig{EventsMax: d.Cfg.ExamEventsMax, SimilarityMinPermille: d.Cfg.SimilarityMinPermille, SimilarityCapPermille: d.Cfg.SimilarityCapPermille}, Log: d.Log, Redis: d.Redis}
			if d.Blob != nil {
				svc.Blob = d.Blob
			}
			(&examhttp.Handler{Svc: svc, Guard: courseGuard, Idem: RequireIdempotencyKey(d), OptIdem: OptionalIdempotencyKey(d), ZipMaxBytes: d.Cfg.ExamTestZipMaxBytes, Log: d.Log}).Mount(r)
		}
		if d.DB != nil && d.Jobs != nil && d.Blob != nil && d.Redis != nil {
			// US-P8-01 — tải tài liệu lên (URL ký sẵn), hoàn tất → 202 + việc nền, thử lại, lập chỉ mục lại (SRS FEAT-docs-calendar 6, #1–#2, #9–#11).
			clk := d.Clock
			if clk == nil {
				clk = clock.Real{}
			}
			ds := &document.Service{Pool: d.DB, Redis: d.Redis, Blob: d.Blob, Jobs: d.Jobs, Clock: clk, Log: d.Log}
			(&documenthttp.Handler{Svc: ds, Guard: courseGuard, Idem: RequireIdempotencyKey(d), OptIdem: OptionalIdempotencyKey(d), Log: d.Log}).Mount(r)
		}
		if d.Thread != nil {
			// US-P3-06 — Threads: đọc / precheck / đăng / bình luận / quyết định của Staff / thread tương tự (SRS FEAT-private-chat-pii 6, #12–#20).
			(&threadhttp.Handler{Svc: d.Thread, Guard: courseGuard, Idem: RequireIdempotencyKey(d), Log: d.Log}).Mount(r)
		}
		if d.Chat != nil {
			// US-P3-05 — chat riêng: 7 route JSON ở đây; 3 route SSE nằm ngoài nhóm này (newRouterWith, SRS 4.7.0).
			(&chathttp.Handler{Svc: d.Chat, Idem: RequireIdempotencyKey(d), Drain: drainC(d), Log: d.Log}).Mount(r)
		}
		if d.LLM != nil && d.Redis != nil {
			// US-P1-04 — API cấu hình LLM: 8 đường dẫn / 13 thao tác; RBAC từng route, Idempotency-Key cho POST providers.
			llmhttp.NewAdmin(d.LLM, d.Redis.Client, d.Log, d.Clock).Mount(r, RequireIdempotencyKey(d))
		}
	})

	registerTestRoutes(r, d)
}

// authOptions là tuỳ chọn của auth.Middleware dùng chung cho nhóm nghiệp vụ và các route SSE nằm ngoài nhóm.
func authOptions(d Deps) []auth.MiddlewareOption {
	mw := []auth.MiddlewareOption{}
	if d.Sessions != nil {
		mw = append(mw, auth.WithRevocation(d.Sessions))
	}
	if d.Cfg.AppEnv == "production" {
		mw = append(mw, auth.RequireSession()) // token dev (không sid) bị từ chối ở production
	}
	return mw
}
