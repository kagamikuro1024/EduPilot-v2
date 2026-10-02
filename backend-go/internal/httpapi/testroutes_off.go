//go:build !testroutes

package httpapi

import "github.com/go-chi/chi/v5"

// registerTestRoutes (bản mặc định): KHÔNG có route thử — `/api/v1/_test/…` trả 404 như mọi đường dẫn lạ,
// và gói `internal/testroutes` không được liên kết vào binary (03-AC18).
func registerTestRoutes(chi.Router, Deps) {}
