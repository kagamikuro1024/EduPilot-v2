package httpapi

import "net/http"

// Giá trị cố định của CORS (SRS 6.7). `Allow-Origin` không bao giờ là `*`.
const (
	corsAllowMethods    = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders    = "Authorization, Content-Type, Idempotency-Key, If-Match, If-None-Match, Last-Event-ID, X-Request-Id"
	corsExposeHeaders   = "ETag, X-Request-Id, Retry-After, Idempotent-Replayed"
	corsPreflightMaxAge = "600"
)

// corsMiddleware (M4): chỉ origin nằm trong CORS_ORIGINS mới nhận `Allow-Origin` + `Allow-Credentials`;
// preflight trả 204 ngay (không đi tiếp tới auth). `Vary: Origin` có trên mọi phản hồi.
func corsMiddleware(d Deps) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(d.Cfg.CORSOrigins))
	for _, o := range d.Cfg.CORSOrigins {
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			_, ok := allowed[origin]
			ok = ok && origin != ""
			if ok {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
			}
			if r.Method != http.MethodOptions || r.Header.Get("Access-Control-Request-Method") == "" {
				next.ServeHTTP(w, r)
				return
			}
			if ok {
				h.Set("Access-Control-Allow-Methods", corsAllowMethods)
				h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
				h.Set("Access-Control-Max-Age", corsPreflightMaxAge)
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
