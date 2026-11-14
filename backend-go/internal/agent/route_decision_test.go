package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/llmtest"
	"github.com/edupilot/backend-go/internal/privacy"
)

// Câu diễn đạt lại mà luật không nhận: rơi về COURSE_QA (kiểm bằng DetectIntent trong test).
const paraphrase = "Lần trước thầy dặn nộp cái gì đó mà em quên mất rồi"

func routedAnalyzer(ctl *fake.Controller, tc agent.TrustedContext, gateOn bool) *agent.Analyzer {
	det := &privacy.Detector{Roster: &privacy.Roster{Src: qcRoster{}}}
	a := &agent.Analyzer{Classifier: &privacy.Classifier{Detector: det}, Detector: det, Self: qcSelf{}}
	if gateOn {
		a.Decider = llmtest.Gateway(ctl, tc.CourseID, llm.GateRouting)
	} else {
		a.Decider = llmtest.Gateway(ctl, tc.CourseID)
	}
	return a
}

func choice(label string, conf float64) fake.DecisionRecord {
	other := (1 - conf)
	return fake.DecisionRecord{Choice: label, Probabilities: map[string]float64{label: conf, "COURSE_QA": other}}
}

// TestRouteDecisionTable — AC1/AC3/AC5: chỉ khi luật rơi về COURSE_QA mới hỏi decision; ≥ τ nhận nhãn, còn lại giữ luật; OTHER_PERSON đi đường từ chối.
func TestRouteDecisionTable(t *testing.T) {
	require.Equal(t, agent.IntentCourseQA, agent.DetectIntent(paraphrase), "tiền đề: luật không nhận câu này")
	tc := agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	for _, tc2 := range []struct {
		name      string
		on        bool
		rec       *fake.DecisionRecord
		mode      string
		want      agent.Intent
		wantOther bool
		wantCalls int64
	}{
		{"≥ τ → nhận nhãn", true, ptrRec(choice("UPCOMING_EVENTS", 0.97)), "", agent.IntentUpcoming, false, 1},
		{"= τ → nhận nhãn", true, ptrRec(choice("PERSONAL_GRADE", 0.95)), "", agent.IntentGrade, false, 1},
		{"dưới τ → giữ luật", true, ptrRec(choice("UPCOMING_EVENTS", 0.94)), "", agent.IntentCourseQA, false, 1},
		{"đầu vào lạ (chia đều) → giữ luật", true, nil, "", agent.IntentCourseQA, false, 1},
		{"decision lỗi → giữ luật", true, ptrRec(choice("UPCOMING_EVENTS", 0.99)), "error", agent.IntentCourseQA, false, 1},
		{"decision treo → giữ luật", true, ptrRec(choice("UPCOMING_EVENTS", 0.99)), "timeout", agent.IntentCourseQA, false, 1},
		{"OTHER_PERSON → đường từ chối", true, ptrRec(choice("OTHER_PERSON", 0.99)), "", agent.IntentOtherPerson, true, 1},
		{"nhãn lạ (không trong 11 nhãn) → giữ luật", true, ptrRec(choice("CRISIS", 0.99)), "", agent.IntentCourseQA, false, 1},
		{"cổng tắt ở lớp → không gọi", false, ptrRec(choice("UPCOMING_EVENTS", 0.99)), "", agent.IntentCourseQA, false, 0},
	} {
		t.Run(tc2.name, func(t *testing.T) {
			ctl := fake.NewController(fake.Settings{DecisionMode: tc2.mode})
			if tc2.rec != nil {
				ctl.AddDecision(paraphrase, "intent", *tc2.rec)
			}
			got, err := routedAnalyzer(ctl, tc, tc2.on).Analyze(context.Background(), tc, paraphrase)
			require.NoError(t, err)
			require.Equal(t, tc2.want, got.Intent)
			require.Equal(t, tc2.wantOther, got.OtherPerson)
			require.Equal(t, tc2.wantCalls, ctl.Decisions())
		})
	}
}

func ptrRec(r fake.DecisionRecord) *fake.DecisionRecord { return &r }

// TestRouteDecisionNoOverride — AC4: luật đã khớp ý định cụ thể thì decision KHÔNG được gọi và không ghi đè (dù decision "muốn" nhãn khác).
func TestRouteDecisionNoOverride(t *testing.T) {
	tc := agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	for q, want := range map[string]agent.Intent{
		"Em đã vắng mấy buổi rồi":             agent.IntentAttendance,
		"Điểm giữa kỳ của em là bao nhiêu":    agent.IntentGrade,
		"Khi nào em thi cuối kỳ?":             agent.IntentExamSchedule,
		"cách tính điểm của lớp mình thế nào": agent.IntentGradeFormula,
		"Tìm slide về chữ ký số":              agent.IntentLibrary,
		"Xin chào":                            agent.IntentSmalltalk,
		"Bùi Thanh Khải được mấy điểm lab?":   agent.IntentOtherPerson,
	} {
		ctl := fake.NewController(fake.Settings{DecisionMode: "always_yes"})
		ctl.AddDecision(q, "intent", choice("LIBRARY_SEARCH", 0.99))
		got, err := routedAnalyzer(ctl, tc, true).Analyze(context.Background(), tc, q)
		require.NoError(t, err, q)
		require.Equal(t, want, got.Intent, q)
		require.Zero(t, ctl.Decisions(), "luật đã khớp: không gọi decision: "+q)
	}
}

// TestCrisisNeverReachesDecide — AC4 (chat): khủng hoảng do luật; 0 lời gọi decision.
func TestCrisisNeverReachesDecide(t *testing.T) {
	tc := agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	for _, q := range []string{"Em không muốn sống nữa", "em muốn chết", "tôi nghĩ tới chuyện tự tử", "em khong muon song nua"} {
		ctl := fake.NewController(fake.Settings{DecisionMode: "always_no"})
		got, err := routedAnalyzer(ctl, tc, true).Analyze(context.Background(), tc, q)
		require.NoError(t, err)
		require.Equal(t, agent.IntentCrisis, got.Intent)
		require.Zero(t, ctl.Decisions(), q)
	}
}

// TestRouteDecisionParallelWithEmbed — AC2: decision chạy SONG SONG với lần nhúng (tổng ≈ max, không ≈ cộng) và mỗi tin chỉ một lần nhúng + một decision.
func TestRouteDecisionParallelWithEmbed(t *testing.T) {
	tc := agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	ctl := fake.NewController(fake.Settings{DecisionP50: 250 * time.Millisecond, DecisionP95: 250 * time.Millisecond})
	a := routedAnalyzer(ctl, tc, true)
	proto, other := make([]float32, 8), make([]float32, 8)
	proto[0], other[1] = 1, 1 // trực giao: cosine 0, không phải câu cá nhân
	a.Classifier = &privacy.Classifier{Detector: a.Detector, Protos: privacy.NewPrototypesFromVectors([][]float32{proto})}
	embeds := 0
	a.Embed = func(ctx context.Context, _ string) ([]float32, error) {
		embeds++
		select {
		case <-time.After(250 * time.Millisecond):
		case <-ctx.Done():
		}
		return other, nil
	}
	// luật PII / mẫu câu chưa quyết → Classify nhúng → decision chạy song song với lần nhúng đó.
	start := time.Now()
	_, err := a.Analyze(context.Background(), tc, paraphrase)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 420*time.Millisecond, "nối tiếp sẽ ≥ 500 ms")
	require.Equal(t, 1, embeds)
	require.EqualValues(t, 1, ctl.Decisions())
}
