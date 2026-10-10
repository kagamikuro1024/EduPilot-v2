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
	"github.com/edupilot/backend-go/internal/library"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/thread"
)

// aiStack là phần dùng chung của chat riêng và Threads: bộ phát hiện PII, bộ phân loại, hàm nhúng, cổng LLM.
type aiStack struct {
	det   *privacy.Detector
	cl    *privacy.Classifier
	embed func(ctx context.Context, text string) ([]float32, error) // đã nạp mẫu câu cá nhân muộn (warm) trước khi nhúng
	raw   func(ctx context.Context, text string) ([]float32, error)
	gw    *llm.Gateway
}

// newAI dựng phần dùng chung; nil nếu thiếu DB / Redis / LLM (khi đó không có route chat / Threads).
func newAI(d Deps) *aiStack {
	if d.DB == nil || d.Redis == nil || d.LLM == nil {
		return nil
	}
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: d.DB}, Redis: d.Redis, Log: d.Log}}
	cl := &privacy.Classifier{Detector: det, Log: d.Log}
	gw := d.LLM.Gateway
	raw := agent.NewEmbedder(gw, d.Redis)
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
	return &aiStack{det: det, cl: cl, gw: gw, raw: raw, embed: func(ctx context.Context, text string) ([]float32, error) { warm(ctx); return raw(ctx, text) }}
}

func (a *aiStack) locker(d Deps) *exam.Locker {
	return &exam.Locker{Pool: d.DB, Redis: d.Redis, Clock: d.Clock, Grace: time.Duration(d.Cfg.ExamGraceSeconds) * time.Second}
}

// threadService dựng thread.Service cho gateway (đăng / đọc / quyết định); việc AI trả lời chạy ở worker.
func threadService(d Deps, a *aiStack) *thread.Service {
	if a == nil || d.Jobs == nil {
		return nil
	}
	return &thread.Service{Pool: d.DB, Redis: d.Redis, FW: &thread.Firewall{Detector: a.det, Classifier: a.cl, Embed: a.embed}, Lock: a.locker(d), Jobs: d.Jobs,
		Rag: &rag.Service{DB: d.DB}, LLM: a.gw, Embed: a.raw, Clock: d.Clock, Log: d.Log}
}

// chatService dựng chat.Service thật từ Deps (nil nếu thiếu DB / Redis / LLM → route chat không được gắn).
func chatService(d Deps, a *aiStack) *chat.Service {
	if a == nil {
		return nil
	}
	an := &agent.Analyzer{Classifier: a.cl, Detector: a.det, Self: agent.StoreSelf{Pool: d.DB}, Log: d.Log, Embed: a.embed}
	ag := &agent.Agent{
		An: an, Private: agent.DefaultPrivateRegistry(nil, nil, nil, calendarService(d), &library.Service{Pool: d.DB, Log: d.Log}), Rag: &rag.Service{DB: d.DB}, Gen: a.gw, Events: agent.StoreEvents{Pool: d.DB}, Log: d.Log,
		Cache: &agent.AnswerCache{Redis: agent.NewRedisKV(d.Redis), Version: func(ctx context.Context, c uuid.UUID) (int64, error) { return rag.Version(ctx, d.Redis, c) }},
	}
	return &chat.Service{
		Pool: d.DB, Redis: d.Redis, Agent: ag, PII: a.det, Clock: d.Clock, Log: d.Log, Drain: drainC(d),
		Lock:    a.locker(d),
		Members: course.Resolver{Pool: d.DB},
		Cfg:     chat.Config{MaxInput: d.Cfg.ChatMaxInputChars, RatePerMin: d.Cfg.ChatRatePerMin, StreamMax: d.Cfg.ChatStreamMax},
	}
}
