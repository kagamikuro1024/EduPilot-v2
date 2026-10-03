package main

import (
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/mail"
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
	mh := &mail.Handler{Pool: d.DB, Clock: clock.Real{}, Sender: mail.SMTP{Cfg: d.Cfg}, Cfg: d.Cfg, Log: d.Log}
	reg.Register(mail.Topic, mh.Handle)
	cn := &course.Notifier{Pool: d.DB, AppPublicURL: d.Cfg.AppPublicURL, Log: d.Log}
	reg.Register(course.TopicAssigned, cn.HandleAssigned)
	reg.Register(course.TopicJoinRequested, cn.HandleJoinRequested)
	reg.Register(course.TopicJoinDecided, cn.HandleJoinDecided)
	registerTestKinds(runner)
	return reg
}
