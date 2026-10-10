package main

import (
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/today"
)

// newRegistry dựng bảng topic → handler của worker. Thêm việc nghiệp vụ = thêm MỘT dòng
// `reg.Register("<topic>", <handler>)` ở đây; handler phải idempotent theo Message.ID.
// Loại việc dài (`kind` của bảng jobs) thì đăng ký vào runner, không phải topic mới.
func newRegistry(d Deps) *outbox.Registry {
	reg := outbox.NewRegistry()
	pub := sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL)
	runner := jobs.NewRunner(d.DB, pub, clock.Real{}, d.Log)
	reg.Register(jobs.TopicEnqueue, runner.HandleMessage)
	ew := &exam.Worker{Pool: d.DB, Svc: &exam.Service{Pool: d.DB, Blob: d.Blob, Integrity: exam.IntegrityConfig{SimilarityMinPermille: d.Cfg.SimilarityMinPermille, SimilarityCapPermille: d.Cfg.SimilarityCapPermille}}, Sandbox: d.Sandbox, LLM: d.LLM, Log: d.Log}
	ew.Register(runner)                   // code.verify_reference, question.suggest (US-PE-03)
	ingest.RegisterKinds(runner, d.Redis) // document.ingest / reindex / reindex_all: chỉ XADD ep:ingest (US-P8-01), không gọi docling trong consumer outbox
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
	for _, t := range []string{exam.TopicExamOpened, exam.TopicExamClosed, exam.TopicAttemptStarted, exam.TopicAttemptSubmitted, exam.TopicSimilarityDone, exam.TopicSimilarityReviewed, exam.TopicQuestionReviewed, course.TopicChanged, auth.TopicUserVerified} {
		reg.Register(t, inv.Handle)
	}
	// Từ điển PII (US-P3-02): thành viên đổi / nhập danh sách → xoá ep:roster:{course} ngay sau xoá cache "Hôm nay". Cùng topic nên phải Chain.
	roster := &privacy.Roster{Redis: d.Redis, Log: d.Log}
	for _, t := range []string{course.TopicMemberChanged, course.TopicRosterImport} {
		reg.Register(t, outbox.Chain(inv.Handle, roster.Invalidate))
	}
	// Tài liệu (US-P8-01): đổi / xoá → tăng ep:rag:ver:{course}; việc đọc dài chạy ở consumer ep:ingest riêng nên không làm trễ việc này.
	bump := rag.BumpVersion(d.Redis)
	reg.Register(ingest.TopicDocumentChanged, outbox.Chain(inv.Handle, bump))
	reg.Register("document.deleted", bump)
	if d.Judge != nil {
		reg.Register(judge.TopicEnqueue, d.Judge.HandleEnqueue) // XADD tín hiệu chấm rồi đặt enqueued_at (US-PE-02)
	}
	rn := &exam.ResultNotifier{Pool: d.DB, SSE: pub, Log: d.Log}
	reg.Register(exam.TopicAttemptGraded, outbox.Chain(rn.HandleGraded, inv.Handle)) // US-PE-08
	reg.Register(exam.TopicPublished, outbox.Chain(rn.HandlePublished, inv.Handle))
	reg.Register(exam.TopicRegraded, outbox.Chain(rn.HandleRegraded, inv.Handle))
	reg.Register(exam.TopicAppealCreated, outbox.Chain(rn.HandleAppealCreated, inv.Handle))
	reg.Register(exam.TopicAppealAnswered, outbox.Chain(rn.HandleAppealAnswered, inv.Handle))
	reg.Register(exam.TopicHold, inv.Handle)
	cnf := &exam.CodeNotifier{Pool: d.DB, SSE: pub, Log: d.Log, Svc: ew.Svc}
	reg.Register(judge.TopicDone, cnf.HandleDone) // US-PE-06: kết quả chạy thử / nộp → SSE `exam.run` / `exam.submission` tới chủ bản nộp
	registerTestKinds(runner)
	return reg
}
