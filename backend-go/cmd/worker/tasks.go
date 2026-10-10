package main

import (
	"context"
	"os"
	"time"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

// Task là một việc nền chạy tới khi ctx bị huỷ (relay outbox, consumer Stream…).
// Task tự quản vòng lặp và span gốc của mình; trả về khi ctx huỷ để worker tắt trong ≤ 10 s.
type Task interface {
	Name() string
	Run(ctx context.Context) error
}

// newTasks trả relay + consumer outbox (US-PG-02): relay xếp hàng dòng tới hạn vào Stream
// `outbox.dispatch`, consumer chạy handler đăng ký trong newRegistry.
func newTasks(d Deps) []Task {
	od := outbox.Deps{Pool: d.DB, Redis: d.Redis, Log: d.Log, Clock: clock.Real{}, Cfg: d.Cfg}
	tasks := []Task{outbox.NewRelay(od), outbox.NewConsumer(od, newRegistry(d))}
	name := "worker"
	if d.Judge != nil {
		name = d.Judge.Name // cùng tên với bộ lập lịch của hàng chấm: hai bộ cùng tiến trình chung một khoá leader
	}
	tasks = append(tasks, examTickTask{&exam.Ticker{Svc: &exam.Service{Pool: d.DB, Clock: clock.Real{}, Jobs: jobs.NewService(d.DB), Attempt: exam.AttemptConfig{Grace: time.Duration(d.Cfg.ExamGraceSeconds) * time.Second}, Redis: d.Redis}, Redis: d.Redis, Log: d.Log, Name: name, Every: d.Cfg.ExamTickInterval}})
	if d.JudgeConsumer {
		tasks = append(tasks, judgeTask{d.Judge})
	}
	if d.LLM != nil && d.Blob != nil {
		set := ingest.LoadSettings(os.Getenv)
		pub := sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL)
		proc := &ingest.Processor{Pool: d.DB, Blob: d.Blob, Docling: &ingest.Docling{BaseURL: set.DoclingURL, Poll: set.PollInterval}, LLM: d.LLM,
			Jobs: jobs.NewRunner(d.DB, pub, clock.Real{}, d.Log), Log: d.Log, Set: set}
		tasks = append(tasks, ingestTask{&ingest.Queue{P: proc, Redis: d.Redis, Consumer: d.Cfg.InstanceID, Log: d.Log}})
	}
	return tasks
}

// ingestTask chạy consumer hàng ep:ingest (đọc tài liệu, lập chỉ mục lại) — tách khỏi consumer outbox để việc dài không chặn việc ngắn.
type ingestTask struct{ q *ingest.Queue }

func (ingestTask) Name() string                    { return "ingest.consumer" }
func (t ingestTask) Run(ctx context.Context) error { return t.q.Run(ctx) }

// examTickTask chạy bộ lập lịch bài thi (mở / đóng đúng giờ, US-PE-04) — một bản nhờ khoá leader Redis.
type examTickTask struct{ t *exam.Ticker }

func (examTickTask) Name() string                    { return "exam.tick" }
func (t examTickTask) Run(ctx context.Context) error { return t.t.Run(ctx) }

// judgeTask chạy consumer chấm code (chỉ ở bản worker JUDGE_CONSUMER=true).
type judgeTask struct{ q *judge.Queue }

func (judgeTask) Name() string                    { return "judge.consumer" }
func (t judgeTask) Run(ctx context.Context) error { return t.q.Run(ctx) }
