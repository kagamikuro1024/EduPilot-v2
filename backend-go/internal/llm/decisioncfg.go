package llm

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/store"
)

// DecisionChannel là kênh pub/sub báo cấu hình decision đã đổi (SRS FEAT-decision-model 4.4).
const DecisionChannel = "ep:decision:cfg"

// DecisionPoll là chu kỳ làm mới dự phòng khi mất thông báo pub/sub (TTL 60 s).
const DecisionPoll = 60 * time.Second

// GateCfg là cấu hình MỘT cổng toàn hệ thống. Threshold là ε (THREADS_PRECHECK) hoặc τ (INJECTION_FLAG, CHAT_ROUTING).
type GateCfg struct {
	Enabled   bool
	Threshold float64
	Timeout   time.Duration
	Version   int
}

// DecisionConfig là ảnh chụp cấu hình: cổng + tập (cổng, lớp) đang bật.
type DecisionConfig struct {
	Gates   map[DecideGate]GateCfg
	Courses map[DecideGate]map[uuid.UUID]struct{}
}

// DecisionSource nạp cấu hình (PG ở chạy thật, bản giả ở test).
type DecisionSource interface {
	LoadDecision(ctx context.Context) (DecisionConfig, error)
}

// DecisionSnapshot giữ ảnh chụp cấu hình trong bộ nhớ: đường nóng KHÔNG chạm DB (US-DM-02 AC5). Chưa nạp được → coi như tắt.
type DecisionSnapshot struct {
	src DecisionSource
	log *slog.Logger
	cur atomic.Pointer[DecisionConfig]
}

// NewDecisionSnapshot dựng ảnh chụp rỗng (mọi cổng tắt) rồi gọi Load để nạp.
func NewDecisionSnapshot(src DecisionSource, log *slog.Logger) *DecisionSnapshot {
	if log == nil {
		log = slog.Default()
	}
	return &DecisionSnapshot{src: src, log: log}
}

// Load nạp lại; lỗi → giữ ảnh cũ (hoặc rỗng = tắt) và trả lỗi.
func (s *DecisionSnapshot) Load(ctx context.Context) error {
	cfg, err := s.src.LoadDecision(ctx)
	if err != nil {
		return err
	}
	s.cur.Store(&cfg)
	return nil
}

// Set thay ảnh chụp (test).
func (s *DecisionSnapshot) Set(cfg DecisionConfig) { s.cur.Store(&cfg) }

// Gate trả cấu hình hệ thống của cổng (không xét lớp).
func (s *DecisionSnapshot) Gate(g DecideGate) (GateCfg, bool) {
	if s == nil {
		return GateCfg{}, false
	}
	c := s.cur.Load()
	if c == nil {
		return GateCfg{}, false
	}
	gc, ok := c.Gates[g]
	return gc, ok
}

// Effective trả cấu hình hiệu lực của cổng cho lớp: công tắc toàn hệ thống VÀ công tắc của lớp đều bật; ngược lại ok = false (cổng chạy như trước 10.5).
// Không có lớp (nil) → tắt: mọi cổng đều gắn với một lớp.
func (s *DecisionSnapshot) Effective(g DecideGate, course *uuid.UUID) (GateCfg, bool) {
	gc, ok := s.Gate(g)
	if !ok || !gc.Enabled || course == nil {
		return GateCfg{}, false
	}
	if _, on := s.cur.Load().Courses[g][*course]; !on {
		return GateCfg{}, false
	}
	return gc, true
}

// Watch làm mới theo sự kiện Redis `ep:decision:cfg` và mỗi 60 s; chạy tới khi ctx huỷ. rdb nil → chỉ thăm dò định kỳ.
func (s *DecisionSnapshot) Watch(ctx context.Context, rdb *goredis.Client) {
	var msgs <-chan *goredis.Message
	if rdb != nil {
		ps := rdb.Subscribe(ctx, DecisionChannel)
		defer func() { _ = ps.Close() }()
		msgs = ps.Channel()
	}
	t := time.NewTicker(DecisionPoll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case _, ok := <-msgs:
			if !ok {
				msgs = nil
				continue
			}
		}
		if err := s.Load(ctx); err != nil {
			s.log.WarnContext(ctx, "không nạp được cấu hình decision, giữ bản cũ", "error", err.Error())
		}
	}
}

// PublishDecisionCfg báo mọi gateway nạp lại cấu hình decision (ADMIN vừa đổi).
func PublishDecisionCfg(ctx context.Context, rdb *goredis.Client) error {
	return rdb.Publish(ctx, DecisionChannel, "1").Err()
}

// PGDecisionSource đọc cấu hình từ `decision_gates` + `course_decision_gates`.
func PGDecisionSource(db store.DBTX) DecisionSource { return pgDecisionSource{q: store.New(db)} }

type pgDecisionSource struct{ q *store.Queries }

func (p pgDecisionSource) LoadDecision(ctx context.Context) (DecisionConfig, error) {
	gs, err := p.q.DecisionGatesAll(ctx)
	if err != nil {
		return DecisionConfig{}, err
	}
	cs, err := p.q.DecisionCourseGatesEnabled(ctx)
	if err != nil {
		return DecisionConfig{}, err
	}
	cfg := DecisionConfig{Gates: map[DecideGate]GateCfg{}, Courses: map[DecideGate]map[uuid.UUID]struct{}{}}
	for _, g := range gs {
		th, _ := g.Threshold.Float64()
		cfg.Gates[DecideGate(g.Gate)] = GateCfg{Enabled: g.Enabled, Threshold: th, Timeout: time.Duration(g.TimeoutMs) * time.Millisecond, Version: int(g.Version)}
	}
	for _, c := range cs {
		gate := DecideGate(c.Gate)
		if cfg.Courses[gate] == nil {
			cfg.Courses[gate] = map[uuid.UUID]struct{}{}
		}
		cfg.Courses[gate][c.CourseID] = struct{}{}
	}
	return cfg, nil
}
