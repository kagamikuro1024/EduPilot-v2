package scheduler_test

import (
	"errors"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
)

// TestDecideBucketAndNoWait: bucket RPM riêng của provider decision; hết giấy phép → ErrOverloaded NGAY (NoWait) để cascade đi fallback,
// còn provider chat vẫn được cấp (không dùng chung bucket).
func TestDecideBucketAndNoWait(t *testing.T) {
	t.Parallel()
	s := scheduler.New(baseCfg(), nil, discard())
	dec := llm.Work{Lane: llm.LaneInteractive, Task: llm.TaskDecide, ProviderID: "dec", RPM: 20, TPM: 1_000_000, EstTokens: 10, NoWait: true} // burst = RPM/10 = 2
	for i := range 2 {
		p, err := s.Admit(t.Context(), dec)
		if err != nil {
			t.Fatalf("lần %d: %v", i, err)
		}
		p.Done(10)
	}
	start := time.Now()
	_, err := s.Admit(t.Context(), dec)
	var ov *llm.ErrOverloaded
	if !errors.As(err, &ov) || time.Since(start) > 100*time.Millisecond {
		t.Fatalf("muốn ErrOverloaded tức thì, được %v sau %v", err, time.Since(start))
	}
	chat := llm.Work{Lane: llm.LaneInteractive, Task: llm.TaskChat, ProviderID: "chat", RPM: 60, TPM: 1_000_000, EstTokens: 10}
	p, err := s.Admit(t.Context(), chat)
	if err != nil {
		t.Fatalf("chat bị ảnh hưởng bởi bucket decision: %v", err)
	}
	p.Done(10)
}
