package llm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
)

// P-11-23: hoãn không tính lượt tới OutageMax; lỗi khác / ngân sách hết / ctx huỷ ⇒ không hoãn.
func TestAwaitOutage(t *testing.T) {
	old := llm.OutageDelay
	llm.OutageDelay = time.Millisecond
	t.Cleanup(func() { llm.OutageDelay = old })
	out := &llm.ErrUnavailable{Reason: llm.ReasonAllFailed, Outage: true}
	n := 0
	for i := range llm.OutageMax {
		if !llm.AwaitOutage(t.Context(), out, &n) {
			t.Fatalf("lần hoãn %d bị từ chối", i+1)
		}
	}
	if llm.AwaitOutage(t.Context(), out, &n) {
		t.Fatal("quá trần OutageMax vẫn hoãn")
	}
	for _, err := range []error{nil, errors.New("x"), &llm.ErrUnavailable{Reason: llm.ReasonAllFailed}, &llm.ErrUnavailable{Reason: llm.ReasonBudgetExhausted, Outage: true}} {
		m := 0
		if llm.AwaitOutage(t.Context(), err, &m) || m != 0 {
			t.Errorf("%v không phải outage", err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	m := 0
	if llm.AwaitOutage(ctx, out, &m) {
		t.Error("ctx đã huỷ vẫn hoãn")
	}
}
