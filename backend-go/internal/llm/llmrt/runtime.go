// Package llmrt dựng và chạy cổng LLM của một tiến trình gateway: cấu hình, Registry, Scheduler, ngân sách, llm_audit.
package llmrt

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/crypto"
)

// Runtime gom mọi thành phần cổng LLM của một tiến trình.
type Runtime struct {
	Registry  *llm.Registry
	Scheduler *scheduler.Scheduler
	Budget    *budget.Manager
	Gateway   *llm.Gateway
	Auditor   *llm.Auditor
	Config    *llmconfig.Service
	Resolver  *llmconfig.Resolver
	cancel    context.CancelFunc
}

// EnvFrom đổi cấu hình tiến trình thành EnvConfig của Registry.
func EnvFrom(cfg config.Config) llm.EnvConfig {
	return llm.EnvConfig{
		Provider: cfg.LLMProvider, OpenAIKey: cfg.OpenAIKey, AnthropicKey: cfg.AnthropicKey, GeminiKey: cfg.GeminiKey,
		DefaultRPM: cfg.LLMDefaultRPM, DefaultTPM: cfg.LLMDefaultTPM,
		Fake: fake.Settings{LatencyMin: cfg.FakeLatencyMin, LatencyMax: cfg.FakeLatencyMax, ErrorRate: cfg.FakeErrorRate,
			ValidKey: cfg.FakeValidKey, ReplayDir: cfg.LLMReplayDir},
	}
}

// SchedulerFrom đổi cấu hình tiến trình thành Config của Scheduler.
func SchedulerFrom(cfg config.Config) scheduler.Config {
	return scheduler.Config{MaxConcurrency: cfg.LLMMaxConcurrency, BatchShare: cfg.LLMBatchShare, QueueMax: cfg.LLMQueueMax,
		QueueWaitMax: cfg.LLMQueueWaitMax, BreakerFails: cfg.LLMBreakerFails, BreakerOpen: cfg.LLMBreakerOpen}
}

// New dựng và chạy cổng LLM: nạp cấu hình, nghe thông báo nạp lại (pub/sub + thăm dò 60 s), ghi llm_audit bất đồng bộ.
// Nạp lần đầu lỗi → log error và chạy tiếp với cấu hình rỗng (mọi lời gọi trả ErrNotConfigured tới khi nạp lại thành công).
func New(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, rdb *goredis.Client, log *slog.Logger) (*Runtime, error) {
	cipher, err := crypto.ParseKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, err
	}
	rt := &Runtime{}
	rt.Resolver = llmconfig.NewResolver(pool, cipher)
	rt.Config = llmconfig.New(pool, cipher, llmconfig.WithOnChange(func(c context.Context) {
		if rdb != nil {
			err := llm.PublishReload(context.WithoutCancel(c), rdb)
			if err == nil {
				return
			}
			log.WarnContext(c, "không báo được nạp lại cấu hình LLM qua Redis", "error", err.Error()) // vẫn nạp lại cục bộ
		}
		_ = rt.Registry.Reload(c)
	}))
	rt.Registry = llm.NewRegistry(rt.Resolver, EnvFrom(cfg), nil, log)
	if err := rt.Registry.Load(ctx); err != nil {
		log.ErrorContext(ctx, "nạp cấu hình LLM lần đầu thất bại", "error", err.Error())
	}
	rt.Budget = budget.New(rdb, rt.Resolver, pool, clock.Real{}, log)
	rt.Registry.OnReload(rt.Budget.Invalidate)
	rt.Scheduler = scheduler.New(SchedulerFrom(cfg), rdb, log, scheduler.WithBudget(rt.Budget))
	rt.Scheduler.OnRedisRecover(func(c context.Context) {
		if err := rt.Budget.Reconcile(c); err != nil {
			log.ErrorContext(c, "đối soát ngân sách LLM thất bại", "error", err.Error())
		}
	})
	rt.Auditor = llm.NewAuditor(ctx, llm.PGWriter(pool), log)
	rt.Gateway = llm.New(llm.Options{Registry: rt.Registry, Gate: rt.Scheduler, Auditor: rt.Auditor, Log: log, RequestTimeout: cfg.LLMRequestTimeout})
	wctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	rt.cancel = cancel
	go rt.Registry.Watch(wctx, rdb)
	return rt, nil
}

// Close dừng nghe nạp lại và đẩy hết llm_audit còn đệm.
func (r *Runtime) Close(ctx context.Context) {
	if r == nil {
		return
	}
	r.cancel()
	r.Auditor.Close(ctx)
}

// Stats trả chỉ số đếm cho route thử `_test/llm/stats` đúng SRS 6.4: KHÔNG có nội dung prompt hay câu trả lời.
// lanes=true thêm `inflight_batch` (công cụ đo `cmd/llmload`).
func (r *Runtime) Stats(ctx context.Context, lanes bool) map[string]any {
	st := r.Scheduler.Stats(ctx)
	out := map[string]any{
		"queue_depth":       st.QueueDepth,
		"inflight":          st.Inflight,
		"provider_inflight": st.ProviderInflight,
		"circuit":           st.Circuit,
		"fake_calls":        r.Registry.Fake().CallsByProvider(),
		"audit":             map[string]int64{"buffer_len": int64(r.Auditor.Pending()), "flushed": r.Auditor.Flushed(), "dropped": r.Auditor.Dropped()},
	}
	if lanes {
		out["inflight_batch"] = st.InflightBatch
	}
	return out
}
