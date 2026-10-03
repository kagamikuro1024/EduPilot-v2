//go:build testroutes

package httpapi

import (
	"github.com/edupilot/backend-go/internal/testroutes"
	"github.com/go-chi/chi/v5"
)

// registerTestRoutes (bản dựng `-tags testroutes`): đăng ký 15 thao tác thử dưới `/api/v1/_test`
// và tạo bảng `_test_items` (SRS 6.3). Bản `!testroutes` ở testroutes_off.go không làm gì,
// nên binary mặc định không liên kết gói `internal/testroutes`.
func registerTestRoutes(r chi.Router, d Deps) {
	testroutes.Register(r, testroutes.Deps{
		DB:                    d.DB,
		Redis:                 d.Redis,
		Log:                   d.Log,
		Clock:                 d.Clock,
		Verifier:              d.Verifier,
		Publisher:             d.Publisher,
		Jobs:                  d.Jobs,
		RequireIdempotencyKey: RequireIdempotencyKey(d),
	})
}
