//go:build testroutes

package main

import "github.com/edupilot/backend-go/internal/platform/outbox"

// registerTestHandlers gắn handler của tuyến thử (`internal/testroutes`, US-PG-03/06) —
// chỉ có trong bản dựng `-tags testroutes`.
func registerTestHandlers(_ *outbox.Registry, _ Deps) {}
