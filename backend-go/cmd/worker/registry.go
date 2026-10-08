package main

import (
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/today"
)

// newRegistry dựng bảng topic → handler của worker. Thêm việc nghiệp vụ = thêm MỘT dòng
// `reg.Register("<topic>", <handler>)` ở đây; handler phải idempotent theo Message.ID.
// Loại việc dài (`kind` của bảng jobs) thì đăng ký vào runner, không phải topic mới.
func newRegistry(d Deps) *outbox.Registry {
	reg := outbox.NewRegistry()
	runner := jobs.NewRunner(d.DB, sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL), clock.Real{}, d.Log)
	reg.Register(jobs.TopicEnqueue, runner.HandleMessage)
	ew := &exam.Worker{Pool: d.DB, Svc: &exam.Service{Pool: d.DB, Blob: d.Blob}, Sandbox: d.Sandbox, LLM: d.LLM, Log: d.Log}
	ew.Register(runner) // code.verify_reference, question.suggest (US-PE-03)
	mh := &mail.Handler{Pool: d.DB, Clock: clock.Real{}, Sender: mail.SMTP{Cfg: d.Cfg}, Cfg: d.Cfg, Log: d.Log}
	reg.Register(mail.Topic, mh.Handle)
	cn := &course.Notifier{Pool: d.DB, AppPublicURL: d.Cfg.AppPublicURL, Log: d.Log}
	// "Hôm nay": mỗi sự kiện xoá cache của người bị ảnh hưởng (US-P2-11). Ba topic đã có thông báo được nối chuỗi.
	inv := today.Invalidator{Pool: d.DB, Redis: d.Redis, Log: d.Log}
	reg.Register(course.TopicAssigned, outbox.Chain(cn.HandleAssigned, inv.Handle))
	reg.Register(course.TopicJoinRequested, outbox.Chain(cn.HandleJoinRequested, inv.Handle))
	reg.Register(course.TopicJoinDecided, outbox.Chain(cn.HandleJoinDecided, inv.Handle))
	en := &exam.Notifier{Pool: d.DB, Log: d.Log}
	reg.Register(exam.TopicExamScheduled, outbox.Chain(en.HandleScheduled, inv.Handle)) // US-PE-04: thông báo lịch + xoá cache "Hôm nay"
	reg.Register(exam.TopicExamUnscheduled, outbox.Chain(en.HandleUnscheduled, inv.Handle))
	for _, t := range []string{exam.TopicExamOpened, exam.TopicExamClosed, exam.TopicAttemptStarted, exam.TopicAttemptSubmitted, exam.TopicQuestionReviewed, course.TopicMemberChanged, course.TopicChanged, course.TopicRosterImport, auth.TopicUserVerified} {
		reg.Register(t, inv.Handle)
	}
	if d.Judge != nil {
		reg.Register(judge.TopicEnqueue, d.Judge.HandleEnqueue) // XADD tín hiệu chấm rồi đặt enqueued_at (US-PE-02)
	}
	registerTestKinds(runner)
	return reg
}
