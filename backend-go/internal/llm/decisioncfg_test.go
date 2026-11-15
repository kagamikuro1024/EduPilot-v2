package llm_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/llm"
)

type srcStub struct {
	n   atomic.Int64
	cfg llm.DecisionConfig
	err error
}

func (s *srcStub) LoadDecision(context.Context) (llm.DecisionConfig, error) {
	s.n.Add(1)
	return s.cfg, s.err
}

func cfgOf(systemOn bool, courses ...uuid.UUID) llm.DecisionConfig {
	c := llm.DecisionConfig{Gates: map[llm.DecideGate]llm.GateCfg{llm.GateThreads: {Enabled: systemOn, Threshold: 0.05, Timeout: 800 * time.Millisecond, Version: 3}},
		Courses: map[llm.DecideGate]map[uuid.UUID]struct{}{llm.GateThreads: {}}}
	for _, id := range courses {
		c.Courses[llm.GateThreads][id] = struct{}{}
	}
	return c
}

// TestConfigEffective: hiệu lực = công tắc hệ thống ∧ công tắc lớp; chưa nạp / không lớp → tắt (US-DM-02 AC3, AC4).
func TestConfigEffective(t *testing.T) {
	t.Parallel()
	a, b := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name   string
		cfg    *llm.DecisionConfig
		course *uuid.UUID
		want   bool
	}{
		{"chưa nạp", nil, &a, false},
		{"hệ thống bật + lớp bật", ptr(cfgOf(true, a)), &a, true},
		{"hệ thống bật + lớp khác", ptr(cfgOf(true, a)), &b, false},
		{"hệ thống tắt + lớp bật", ptr(cfgOf(false, a)), &a, false},
		{"không có lớp", ptr(cfgOf(true, a)), nil, false},
	} {
		s := llm.NewDecisionSnapshot(nil, nil)
		if tc.cfg != nil {
			s.Set(*tc.cfg)
		}
		got, ok := s.Effective(llm.GateThreads, tc.course)
		if ok != tc.want || (ok && (got.Threshold != 0.05 || got.Timeout != 800*time.Millisecond)) {
			t.Errorf("%s: ok=%v cfg=%+v", tc.name, ok, got)
		}
	}
	// cổng khác không bị ảnh hưởng
	s := llm.NewDecisionSnapshot(nil, nil)
	s.Set(cfgOf(true, a))
	if _, ok := s.Effective(llm.GateRouting, &a); ok {
		t.Error("cổng chưa cấu hình phải tắt")
	}
}

func ptr[T any](v T) *T { return &v }

// TestConfigSnapshotRefresh: đọc cấu hình là bộ nhớ thuần (không chạm nguồn); Load làm mới; Load lỗi giữ bản cũ.
func TestConfigSnapshotRefresh(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	src := &srcStub{cfg: cfgOf(false, a)}
	s := llm.NewDecisionSnapshot(src, nil)
	if err := s.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	for range 10_000 {
		s.Effective(llm.GateThreads, &a)
	}
	if src.n.Load() != 1 {
		t.Fatalf("đường nóng chạm nguồn %d lần", src.n.Load())
	}
	src.cfg = cfgOf(true, a)
	if _, ok := s.Effective(llm.GateThreads, &a); ok {
		t.Fatal("chưa làm mới thì vẫn là bản cũ")
	}
	if err := s.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Effective(llm.GateThreads, &a); !ok {
		t.Fatal("sau Load phải bật")
	}
	src.err = errors.New("db chết")
	if err := s.Load(t.Context()); err == nil {
		t.Fatal("muốn lỗi")
	}
	if _, ok := s.Effective(llm.GateThreads, &a); !ok {
		t.Fatal("lỗi nạp phải giữ bản cũ")
	}
}
