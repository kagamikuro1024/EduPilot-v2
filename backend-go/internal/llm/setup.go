package llm

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/crypto"
)

// Runtime gom mọi thành phần cổng LLM của một tiến trình gateway.
type Runtime struct {
	Registry *Registry
	Gateway  *Gateway
	Auditor  *Auditor
	Config   *llmconfig.Service
	Resolver *llmconfig.Resolver
	cancel   context.CancelFunc
}

// EnvFrom đổi cấu hình tiến trình thành EnvConfig của Registry.
func EnvFrom(cfg config.Config) EnvConfig {
	return EnvConfig{
		Provider: cfg.LLMProvider, OpenAIKey: cfg.OpenAIKey, AnthropicKey: cfg.AnthropicKey, GeminiKey: cfg.GeminiKey,
		DefaultRPM: cfg.LLMDefaultRPM, DefaultTPM: cfg.LLMDefaultTPM,
		Fake: fake.Settings{LatencyMin: cfg.FakeLatencyMin, LatencyMax: cfg.FakeLatencyMax, ErrorRate: cfg.FakeErrorRate,
			ValidKey: cfg.FakeValidKey, ReplayDir: cfg.LLMReplayDir},
	}
}

// NewRuntime dựng và chạy cổng LLM: nạp cấu hình, nghe thông báo nạp lại (pub/sub + thăm dò 60 s), ghi llm_audit bất đồng bộ.
// Nạp lần đầu lỗi → log error và chạy tiếp với cấu hình rỗng (mọi lời gọi trả ErrNotConfigured tới khi nạp lại thành công).
func NewRuntime(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, rdb *goredis.Client, log *slog.Logger) (*Runtime, error) {
	cipher, err := crypto.ParseKey(cfg.AppEncryptionKey)
	if err != nil {
		return nil, err
	}
	rt := &Runtime{}
	rt.Resolver = llmconfig.NewResolver(pool, cipher)
	rt.Config = llmconfig.New(pool, cipher, llmconfig.WithOnChange(func(c context.Context) {
		if rdb == nil {
			_ = rt.Registry.Reload(c)
			return
		}
		if err := PublishReload(context.WithoutCancel(c), rdb); err != nil { // Redis lỗi: ít nhất nạp lại cục bộ
			log.WarnContext(c, "không báo được nạp lại cấu hình LLM qua Redis", "error", err.Error())
			_ = rt.Registry.Reload(c)
		}
	}))
	rt.Registry = NewRegistry(rt.Resolver, EnvFrom(cfg), nil, log)
	if err := rt.Registry.Load(ctx); err != nil {
		log.ErrorContext(ctx, "nạp cấu hình LLM lần đầu thất bại", "error", err.Error())
	}
	rt.Auditor = NewAuditor(ctx, PGWriter(pool), log)
	rt.Gateway = New(Options{Registry: rt.Registry, Auditor: rt.Auditor, Log: log,
		RequestTimeout: cfg.LLMRequestTimeout})
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

// Stats trả chỉ số đếm cho route thử `_test/llm/stats`: KHÔNG có nội dung prompt hay câu trả lời.
func (r *Runtime) Stats(_ context.Context) map[string]any {
	ctl := r.Registry.Fake()
	return map[string]any{
		"queue_depth":       map[string]int{"INTERACTIVE": 0, "NEAR_REALTIME": 0, "BATCH": 0},
		"inflight":          map[string]int64{"fake": ctl.Active()},
		"provider_inflight": ctl.Active(),
		"provider_calls":    ctl.Calls(),
		"audit_dropped":     r.Auditor.Dropped(),
		"circuit":           map[string]string{},
	}
}
