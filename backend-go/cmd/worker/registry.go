package main

import "github.com/edupilot/backend-go/internal/platform/outbox"

// newRegistry dựng bảng topic → handler của worker. Thêm việc nghiệp vụ = thêm MỘT dòng
// `reg.Register("<topic>", <handler>)` ở đây; handler phải idempotent theo Message.ID.
func newRegistry(d Deps) *outbox.Registry {
	reg := outbox.NewRegistry()
	registerTestHandlers(reg, d)
	return reg
}
