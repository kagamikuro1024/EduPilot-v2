package httpapi

import "github.com/go-chi/chi/v5"

// registerAPIRoutes là ĐIỂM CẮM cho các story sau của nhóm route nghiệp vụ dưới `/api/v1`
// (đã có deadline + giới hạn thân; US-PG-03 thêm rate limit / auth / Idempotency-Key vào nhóm này
// rồi gọi `registerTestRoutes(r, d)` ở đây, US-PG-05 đăng ký `/events` NGOÀI nhóm này vì stream
// sống lâu hơn REQUEST_TIMEOUT).
// PG US-PG-01 chưa có endpoint nghiệp vụ nào ngoài health.
func registerAPIRoutes(_ chi.Router, _ Deps) {}
