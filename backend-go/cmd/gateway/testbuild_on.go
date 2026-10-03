//go:build testroutes

package main

import (
	"errors"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// refuseTestBuild: binary dựng bằng build tag `testroutes` có route ghi dữ liệu thử, nên KHÔNG được
// chạy ở môi trường thật — `APP_ENV=production` thì thoát 1 ngay lúc khởi động (SRS 6.3, 03-AC18).
func refuseTestBuild(cfg config.Config) error {
	if cfg.AppEnv == "production" {
		return errors.New("binary dựng với build tag testroutes không được chạy khi APP_ENV=production")
	}
	return nil
}
