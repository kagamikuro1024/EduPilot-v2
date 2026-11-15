package thread_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/thread"
)

// testDecider: Cascade chạy qua Gateway thật (provider decision giả + ảnh chụp cấu hình); Structured là kịch bản (fallback của cổng).
type testDecider struct {
	llm.Client
	gw         *llm.Gateway
	ctl        *fake.Controller
	structured atomic.Int32
	answer     func(ctx context.Context) (json.RawMessage, error)
}

func (d *testDecider) Structured(ctx context.Context, _ llm.Request, _ json.RawMessage) (json.RawMessage, error) {
	d.structured.Add(1)
	return d.answer(ctx)
}

func (d *testDecider) Cascade(ctx context.Context, g llm.DecideGate, req llm.DecideRequest, p llm.Policy, fb llm.Fallback, fc llm.Verdict) (llm.Verdict, llm.Path) {
	return d.gw.Cascade(ctx, g, req, p, fb, fc)
}

func yes(v bool) func(context.Context) (json.RawMessage, error) {
	return func(context.Context) (json.RawMessage, error) {
		if v {
			return json.RawMessage(`{"answer":"yes"}`), nil
		}
		return json.RawMessage(`{"answer":"no"}`), nil
	}
}

// withDecision gắn cổng THREADS_PRECHECK vào tường lửa của rig (bật cho lớp của rig nếu `on`).
func (r *rig) withDecision(on bool, st fake.Settings) *testDecider {
	r.t.Helper()
	ctl := fake.NewController(st)
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	reg.SetDecide(&llm.Target{ProviderID: "dec", ProviderName: "Quyết định", Type: "decision", Model: llm.FakeDecisionModel, RPM: 100000, TPM: 100000000, P: fake.NewNamed(ctl, "dec", "")})
	snap := llm.NewDecisionSnapshot(nil, nil)
	cfg := llm.DecisionConfig{Gates: map[llm.DecideGate]llm.GateCfg{llm.GateThreads: {Enabled: true, Threshold: 0.05, Timeout: 500 * time.Millisecond}},
		Courses: map[llm.DecideGate]map[uuid.UUID]struct{}{llm.GateThreads: {}}}
	if on {
		cfg.Courses[llm.GateThreads][r.course] = struct{}{}
	}
	snap.Set(cfg)
	d := &testDecider{gw: llm.New(llm.Options{NoMask: true, Registry: reg, Decision: snap}), ctl: ctl, answer: yes(false)}
	r.svc.FW.Decider = d
	r.svc.FW.StructuredTimeout = 300 * time.Millisecond
	return d
}

const askBody = "Bạn Giang lớp mình bị cảnh cáo học vụ phải không?"

// TestThreadsDecisionFlow: bảng AC2 / AC11 / AC10: decision và Structured quyết định chặn hay cho qua; cổng tắt → như trước 10.5.
func TestThreadsDecisionFlow(t *testing.T) {
	t.Parallel()
	one, no := 0.93, 0.03
	for _, tc := range []struct {
		name       string
		on         bool
		rec        *float64 // noul trong bản phát lại; nil = đầu vào lạ (0,25 → lưỡng lự)
		mode       string
		answer     func(context.Context) (json.RawMessage, error)
		wantBlock  bool
		wantDec    int64
		wantStruct int32
	}{
		{"decision có → chặn FAST", true, &one, "", yes(false), true, 1, 0},
		{"decision không → cho qua FAST", true, &no, "", yes(true), false, 1, 0},
		{"lưỡng lự → Structured không → qua", true, nil, "", yes(false), false, 1, 1},
		{"lưỡng lự → Structured có → chặn", true, nil, "", yes(true), true, 1, 1},
		{"decision lỗi → Structured không → qua", true, nil, "error", yes(false), false, 1, 1},
		{"decision lỗi → Structured có → chặn", true, nil, "error", yes(true), true, 1, 1},
		{"decision lỗi + Structured lỗi → FAIL_CLOSED chặn", true, nil, "error", func(context.Context) (json.RawMessage, error) { return nil, errors.New("hết hạn mức") }, true, 1, 1},
		{"decision treo + Structured quá hạn → chặn", true, nil, "timeout", func(ctx context.Context) (json.RawMessage, error) { <-ctx.Done(); return nil, ctx.Err() }, true, 1, 1},
		{"Structured trả sai định dạng → chặn", true, nil, "error", func(context.Context) (json.RawMessage, error) { return json.RawMessage(`{}`), nil }, true, 1, 1},
		{"cổng tắt ở lớp → không gọi gì, qua", false, &one, "", yes(true), false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRig(t)
			d := r.withDecision(tc.on, fake.Settings{DecisionMode: tc.mode})
			d.answer = tc.answer
			if tc.rec != nil {
				d.ctl.AddDecision("Hỏi\n"+askBody, "private_or_named", fake.DecisionRecord{Noul: tc.rec})
			}
			_, err := r.create(r.sv, "Hỏi", askBody, false)
			if tc.wantBlock {
				require.Equal(t, apierr.PIIDetected, code(err))
				var ae *apierr.Error
				require.ErrorAs(t, err, &ae)
				require.Equal(t, true, ae.Details.(map[string]any)["personal_question"])
				require.Zero(t, r.count(`select count(*) from forum_threads`))
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantDec, d.ctl.Decisions())
			require.Equal(t, tc.wantStruct, d.structured.Load())
		})
	}
}

// TestPrecheckAddOnly (AC3, AC5): luật đã chặn → decision không được gọi và không đường nào gỡ chặn; precheck (đang gõ) không gọi decision.
func TestPrecheckAddOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.withDecision(true, fake.Settings{DecisionMode: "always_no"}) // decision "không" cho mọi thứ
	d.answer = yes(false)
	_, err := r.create(r.sv, "Hỏi", "Em là "+svName+" MSSV "+svCode+" xin hỏi điểm", false)
	require.Equal(t, apierr.PIIDetected, code(err), "luật chặn thì vẫn chặn dù decision nói 'không'")
	require.Zero(t, d.ctl.Decisions(), "luật đã chặn: không gọi decision")
	require.Zero(t, d.structured.Load())
	// precheck: chỉ luật + embedding
	_, err = r.svc.Precheck(t.Context(), r.sv, r.course, "Hỏi", askBody)
	require.NoError(t, err)
	require.Zero(t, d.ctl.Decisions(), "precheck không gọi decision")
	// "Ẩn thông tin rồi đăng": bản đã ẩn sạch luật → mới tới decision; decision 'có' vẫn chặn
	d.ctl.Set(fake.Settings{DecisionMode: "always_yes"})
	_, err = r.create(r.sv, "Hỏi", "Em là "+svName+" MSSV "+svCode+" xin hỏi điểm", true)
	require.Equal(t, apierr.PIIDetected, code(err))
}

// TestThreadsDecisionComment: bình luận đi cùng cổng (một tường lửa cho mọi đường ghi).
func TestThreadsDecisionComment(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	d := r.withDecision(true, fake.Settings{DecisionMode: "always_yes"})
	_, err := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: askBody})
	require.Equal(t, apierr.PIIDetected, code(err))
	require.Positive(t, d.ctl.Decisions())
}

// TestThreadsCrisisNeverReachesDecide (AC7): khủng hoảng do luật xử lý trước; decision / Structured không bao giờ được gọi.
func TestThreadsCrisisNeverReachesDecide(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.withDecision(true, fake.Settings{DecisionMode: "always_no"})
	_, err := r.create(r.sv, "Hỏi gì đó", crisisText, false)
	require.Equal(t, apierr.CrisisSupport, code(err))
	require.Zero(t, d.ctl.Decisions())
	require.Zero(t, d.structured.Load())
}

// TestPlaceholderOnlyBlockedByRules (AC4): nội dung chỉ có placeholder bị luật chặn, không tới decision.
func TestPlaceholderOnlyBlockedByRules(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.withDecision(true, fake.Settings{DecisionMode: "always_no"})
	for _, body := range []string{"[[MSSV_1]]", "[[SV_1]] [[MSSV_2]]"} {
		_, err := r.create(r.sv, "Hỏi", body, false)
		require.Equal(t, apierr.PIIDetected, code(err), "%s: %v", body, err)
	}
	require.Zero(t, d.ctl.Decisions())
}
