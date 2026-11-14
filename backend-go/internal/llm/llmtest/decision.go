// Package llmtest dựng Gateway có provider decision giả cho test của các gói dùng cổng cascade (thread, grading, agent).
package llmtest

import (
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
)

// Defaults là ngưỡng / thời hạn mặc định của ba cổng (SRS FEAT-decision-model 4.3).
var Defaults = map[llm.DecideGate]llm.GateCfg{ //nolint:gochecknoglobals // bảng bất biến cho test
	llm.GateThreads:   {Enabled: true, Threshold: 0.05, Timeout: 800 * time.Millisecond},
	llm.GateInjection: {Enabled: true, Threshold: 0.5, Timeout: 3 * time.Second},
	llm.GateRouting:   {Enabled: true, Threshold: 0.95, Timeout: 800 * time.Millisecond},
}

// Gateway dựng Gateway (không che, không audit, không Scheduler) với provider decision giả của `ctl`; các cổng trong `on` bật cho `course`.
// Cổng không nằm trong `on` vẫn bật ở mức hệ thống nhưng lớp chưa bật → không hiệu lực.
func Gateway(ctl *fake.Controller, course uuid.UUID, on ...llm.DecideGate) *llm.Gateway {
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	reg.SetDecide(&llm.Target{ProviderID: "dec", ProviderName: "Quyết định", Type: "decision", Model: llm.FakeDecisionModel,
		RPM: 100_000, TPM: 100_000_000, P: fake.NewNamed(ctl, "dec", "")})
	snap := llm.NewDecisionSnapshot(nil, nil)
	cfg := llm.DecisionConfig{Gates: map[llm.DecideGate]llm.GateCfg{}, Courses: map[llm.DecideGate]map[uuid.UUID]struct{}{}}
	for g, c := range Defaults {
		cfg.Gates[g] = c
		cfg.Courses[g] = map[uuid.UUID]struct{}{}
	}
	for _, g := range on {
		cfg.Courses[g][course] = struct{}{}
	}
	snap.Set(cfg)
	return llm.New(llm.Options{NoMask: true, Registry: reg, Decision: snap})
}
