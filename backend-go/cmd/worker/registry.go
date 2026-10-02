package main

import (
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

// newRegistry dựng bảng topic → handler của worker. Thêm việc nghiệp vụ = thêm MỘT dòng
// `reg.Register("<topic>", <handler>)` ở đây; handler phải idempotent theo Message.ID.
// Loại việc dài (`kind` của bảng jobs) thì đăng ký vào runner, không phải topic mới.
func newRegistry(d Deps) *outbox.Registry {
	reg := outbox.NewRegistry()
	runner := jobs.NewRunner(d.DB, sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL), clock.Real{}, d.Log)
	reg.Register(jobs.TopicEnqueue, runner.HandleMessage)
	registerTestKinds(runner)
	return reg
}
