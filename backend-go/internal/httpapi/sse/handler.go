package sse

import (
	"log/slog"
	"net/http"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/redis"
)

// HandlerDeps là phụ thuộc của handler `GET /api/v1/events`. Handler tự xác thực bằng Verifier (route nằm NGOÀI nhóm timeout
// của router vì stream sống lâu hơn REQUEST_TIMEOUT).
type HandlerDeps struct {
	Cfg      config.Config
	Log      *slog.Logger
	Redis    *redis.Client
	Verifier *auth.Verifier
	Clock    clock.Clock
	// Drain đóng lại khi gateway bắt đầu tắt (httpapi.State.DrainC()): handler gửi `event: shutdown` rồi đóng.
	Drain <-chan struct{}
}

// NewHandler dựng handler SSE. STUB do lead tạo — agent US-PG-05 thay bằng bản thật (SRS 6.8).
func NewHandler(HandlerDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "chưa cài đặt", http.StatusNotImplemented) })
}
