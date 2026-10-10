package httpapi

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
)

// chatService dựng chat.Service thật từ Deps (nil nếu thiếu DB / Redis / LLM → route chat không được gắn).
func chatService(d Deps) *chat.Service {
	if d.DB == nil || d.Redis == nil || d.LLM == nil {
		return nil
	}
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: d.DB}, Redis: d.Redis, Log: d.Log}}
	cl := &privacy.Classifier{Detector: det, Log: d.Log}
	gw := d.LLM.Gateway
	embed := agent.NewEmbedder(gw, d.Redis)
	var lastTry atomic.Int64
	warm := func(ctx context.Context) { // mẫu câu cá nhân nạp muộn: nhà cung cấp nhúng có thể chưa cấu hình lúc khởi động; thử lại ≤ 1 lần / 60 s
		if cl.HasPrototypes() || time.Now().Unix()-lastTry.Load() < 60 {
			return
		}
		lastTry.Store(time.Now().Unix())
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		p, err := privacy.NewPrototypes(wctx, func(c context.Context, in []string) ([][]float32, error) {
			return gw.Embed(c, llm.EmbedRequest{Inputs: in, System: true})
		})
		if err != nil {
			d.Log.WarnContext(ctx, "chat: nạp mẫu câu cá nhân lỗi, chỉ dùng luật", "error", err)
			return
		}
		cl.SetPrototypes(p)
	}
	an := &agent.Analyzer{Classifier: cl, Detector: det, Self: agent.StoreSelf{Pool: d.DB}, Log: d.Log,
		Embed: func(ctx context.Context, text string) ([]float32, error) { warm(ctx); return embed(ctx, text) }}
	ag := &agent.Agent{
		An: an, Private: agent.DefaultPrivateRegistry(nil, nil, nil, nil, nil), Rag: &rag.Service{DB: d.DB}, Gen: gw, Events: agent.StoreEvents{Pool: d.DB}, Log: d.Log,
		Cache: &agent.AnswerCache{Redis: agent.NewRedisKV(d.Redis), Version: func(ctx context.Context, c uuid.UUID) (int64, error) { return rag.Version(ctx, d.Redis, c) }},
	}
	return &chat.Service{
		Pool: d.DB, Redis: d.Redis, Agent: ag, PII: det, Clock: d.Clock, Log: d.Log, Drain: drainC(d),
		Lock:    &exam.Locker{Pool: d.DB, Redis: d.Redis, Clock: d.Clock, Grace: time.Duration(d.Cfg.ExamGraceSeconds) * time.Second},
		Members: course.Resolver{Pool: d.DB},
		Cfg:     chat.Config{MaxInput: d.Cfg.ChatMaxInputChars, RatePerMin: d.Cfg.ChatRatePerMin, StreamMax: d.Cfg.ChatStreamMax},
	}
}
