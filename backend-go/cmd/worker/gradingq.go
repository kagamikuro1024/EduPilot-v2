package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/submission"
)

// newGradingQueue dựng consumer chấm bài (US-P7-07). nil khi thiếu LLM / object storage (worker thử nghiệm): việc `grading.batch` vẫn đăng ký ở registry nhưng không ai chấm.
func newGradingQueue(d Deps) *grading.Queue {
	bs, ok := d.Blob.(submission.BlobAPI)
	if !ok || d.llmRT == nil || d.llmRT.Gateway == nil {
		return nil
	}
	set := ingest.LoadSettings(os.Getenv)
	pub := sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL)
	cfg := grading.DefaultQueueConfig()
	cfg.Workers = envInt("GRADING_WORKERS", cfg.Workers)
	cfg.Lease = envSec("GRADING_LEASE", cfg.Lease)
	cfg.Heartbeat = envSec("GRADING_HEARTBEAT", cfg.Heartbeat)
	cfg.RequeueAfter = envSec("GRADING_REQUEUE_AFTER", cfg.RequeueAfter)
	cfg.Attempts = envInt("GRADING_ATTEMPTS", cfg.Attempts)
	ec := grading.DefaultEssayConfig()
	ec.AttemptTimeout = envSec("GRADING_ATTEMPT_TIMEOUT", ec.AttemptTimeout)
	ec.GuidanceEnabled = d.Cfg.GuidanceEnabled
	return grading.NewQueue(&grading.Queue{
		Pool: d.DB, Redis: d.Redis, Log: d.Log, Cfg: cfg, Name: d.Cfg.InstanceID,
		Grader: &grading.Grader{Pool: d.DB, LLM: d.llmRT.Gateway, Flags: d.llmRT.Guidance, Decide: d.llmRT.Gateway, Blob: bs, Extract: grading.DoclingExtractor{D: &ingest.Docling{BaseURL: set.DoclingURL, Poll: set.PollInterval}, MinText: ec.MinChars}, Cfg: ec, Log: d.Log},
		Jobs:   jobs.NewRunner(d.DB, pub, clock.Real{}, d.Log),
		Notify: notify.New(d.DB, pub, d.Log),
	})
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}

func envSec(k string, def time.Duration) time.Duration {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}
	return def
}

// gradingTask chạy consumer ep:grading — tách khỏi consumer outbox để một bài chậm không chặn việc ngắn (AC6).
type gradingTask struct{ q *grading.Queue }

func (gradingTask) Name() string                    { return "grading.consumer" }
func (t gradingTask) Run(ctx context.Context) error { return t.q.Run(ctx) }

// newReflectQueue dựng consumer rút lưu ý chấm (10.6, SRS FEAT-grading-guidance 3.2). nil khi thiếu LLM (worker thử nghiệm): việc vẫn đăng ký nhưng không ai chạy.
func newReflectQueue(d Deps) *grading.ReflectQueue {
	if d.llmRT == nil || d.llmRT.Gateway == nil || d.Redis == nil {
		return nil
	}
	pub := sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL)
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: d.DB}, Redis: d.Redis, Log: d.Log}}
	return &grading.ReflectQueue{
		R:        &grading.Reflector{Pool: d.DB, LLM: d.llmRT.Gateway, Flags: d.llmRT.Guidance, Det: det, Cfg: grading.GuidanceConfigFrom(d.Cfg), Log: d.Log},
		Runner:   jobs.NewRunner(d.DB, pub, clock.Real{}, d.Log),
		Redis:    d.Redis.Client,
		Consumer: d.Cfg.InstanceID,
		Log:      d.Log,
	}
}

type reflectTask struct{ q *grading.ReflectQueue }

func (reflectTask) Name() string                    { return "grading.reflect.consumer" }
func (t reflectTask) Run(ctx context.Context) error { return t.q.Run(ctx) }
