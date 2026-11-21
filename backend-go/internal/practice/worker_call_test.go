package practice_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/practice"
)

// seqLLM trả lần lượt từng lỗi trong errs (hết thì thành công) và đếm số lời gọi.
type seqLLM struct {
	errs  []error
	calls int
}

func (s *seqLLM) Structured(context.Context, llm.Request, json.RawMessage) (json.RawMessage, error) {
	i := s.calls
	s.calls++
	if i < len(s.errs) {
		return nil, s.errs[i]
	}
	return json.RawMessage(`{"questions":[]}`), nil
}

// TestWorkerCallOverloaded — US-P9-02 AC / TC-02-14, TC-03-08 (DEF-R10S-12): quá tải → thử lại đúng 2 lần (tổng 3 lời gọi, chờ 2 s rồi 5 s) → «Chưa trích được. Thử lại.» (OVERLOADED); lỗi khác không thử lại.
func TestWorkerCallOverloaded(t *testing.T) {
	over := &llm.ErrOverloaded{RetryAfter: time.Second}
	require.Equal(t, []time.Duration{2 * time.Second, 5 * time.Second}, practice.DefaultRetryWait)
	fast := []time.Duration{time.Millisecond, time.Millisecond}

	for _, tc := range []struct {
		name  string
		errs  []error
		calls int
		code  string // "" = thành công
	}{
		{"quá tải mãi → 3 lời gọi rồi OVERLOADED", []error{over, over, over, over, over}, 3, "OVERLOADED"},
		{"quá tải hai lần rồi thành công → 3 lời gọi", []error{over, over}, 3, ""},
		{"quá tải một lần rồi thành công → 2 lời gọi", []error{over}, 2, ""},
		{"lỗi khác → 1 lời gọi, không thử lại", []error{errors.New("boom")}, 1, "LLM_UNAVAILABLE"},
		{"chưa cấu hình AI → 1 lời gọi", []error{llm.ErrNotConfigured}, 1, "LLM_NOT_CONFIGURED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &seqLLM{errs: tc.errs}
			_, err := practice.CallWorker(&practice.Worker{LLM: l, RetryWait: fast}, t.Context(), "s", "u", `{}`)
			require.Equal(t, tc.calls, l.calls)
			if tc.code == "" {
				require.NoError(t, err)
				return
			}
			var ue *jobs.UserError
			require.ErrorAs(t, err, &ue)
			require.Equal(t, tc.code, ue.Code)
			if tc.code == "OVERLOADED" {
				require.Equal(t, "Chưa trích được. Thử lại.", ue.Message)
			}
		})
	}

	t.Run("huỷ việc trong lúc chờ thử lại", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		l := &seqLLM{errs: []error{over, over, over}}
		go func() { time.Sleep(20 * time.Millisecond); cancel() }()
		_, err := practice.CallWorker(&practice.Worker{LLM: l, RetryWait: []time.Duration{time.Hour}}, ctx, "s", "u", `{}`)
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, l.calls)
	})
}
