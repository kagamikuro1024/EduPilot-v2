//go:build !testroutes

package main

import "github.com/edupilot/backend-go/internal/platform/config"

// refuseTestBuild: bản dựng mặc định không có route thử nên chạy được ở mọi APP_ENV.
func refuseTestBuild(config.Config) error { return nil }
