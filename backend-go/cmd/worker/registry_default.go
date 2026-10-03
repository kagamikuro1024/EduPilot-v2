//go:build !testroutes

package main

import "github.com/edupilot/backend-go/internal/jobs"

// registerTestKinds không làm gì trong bản dựng thường (không có loại việc thử).
func registerTestKinds(_ *jobs.Runner) {}
