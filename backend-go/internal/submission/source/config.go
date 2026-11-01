package source

import (
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// ConfigFrom đọc GRAPH_* / TEAMS_* từ cấu hình tiến trình (chỉ địa chỉ và giới hạn — không có cờ đổi hành vi).
func ConfigFrom(c config.Config, maxFileBytes int64) Config {
	return Config{BaseURL: c.GraphBaseURL, AuthURL: c.GraphAuthURL, FileHosts: c.GraphFileHosts, DownloadParallel: c.TeamsDownloadParallel, FileRetry: c.TeamsFileRetry,
		RetryMax: c.TeamsRetryMax, MaxFileBytes: maxFileBytes, FileRetryWait: time.Second}
}
