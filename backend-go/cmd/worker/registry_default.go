//go:build !testroutes

package main

import "github.com/edupilot/backend-go/internal/platform/outbox"

// registerTestHandlers không làm gì trong bản dựng thường (không có tuyến thử).
func registerTestHandlers(_ *outbox.Registry, _ Deps) {}
