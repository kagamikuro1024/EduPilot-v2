package grading_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/llmtest"
)

// withInjectionFlag thêm INJECTION_SUSPECTED vào flags của một lượt chấm (như LLM báo).
func withInjectionFlag(t *testing.T, raw json.RawMessage) json.RawMessage {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	m["flags"] = append(m["flags"].([]any), "INJECTION_SUSPECTED")
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

// TestInjectionOR — US-DM-04: cờ = regex OR Structured (gộp vào lượt chấm) OR decision; chỉ cờ, điểm không đổi; bài ngắn / cổng tắt không có hai tầng sau.
func TestInjectionOR(t *testing.T) {
	ss := samples(t)
	var s1 sample
	for _, s := range ss {
		if s.ID == "S1" {
			s1 = s
		}
	}
	b, err := os.ReadFile(asgFix + s1.File)
	require.NoError(t, err)
	text := string(b)
	yes := 0.9

	for _, tc := range []struct {
		name      string
		gateOn    bool
		decision  *float64 // noul trong bản phát lại; nil = lạ (0,25 → lưỡng lự)
		mode      string
		llmFlag   bool
		short     bool
		wantFlag  bool
		wantDec   int64
		wantInSch bool // prompt / schema có chỗ cho INJECTION_SUSPECTED
	}{
		{"cổng tắt: không tầng nào thêm", false, &yes, "", true, false, false, 0, false},
		{"decision có → cờ", true, &yes, "", false, false, true, 1, true},
		{"Structured báo → cờ (decision lưỡng lự)", true, nil, "", true, false, true, 1, true},
		{"decision lỗi, Structured báo → cờ (tầng vắng)", true, nil, "error", true, false, true, 1, true},
		{"decision lỗi, Structured không → không cờ, chấm vẫn xong", true, nil, "error", false, false, false, 1, true},
		{"không tầng nào → không cờ", true, nil, "", false, false, false, 1, true},
		{"bài < 20 từ: không decision, không Structured gắn cờ", true, &yes, "", true, true, false, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := text
			if tc.short {
				content = "Họ tên: Nguyễn Văn A\nMSSV: 20229999\nEmail: a@x.test\n" + "Kiến trúc phân lớp giúp tách biệt trách nhiệm và dễ bảo trì hơn nhiều."
			}
			cl := grading.StripIdentityLines(content)
			if tc.short {
				require.Less(t, len(strings.Fields(cl)), 20)
			}
			g := newGradeEnv(t, func(g *gradeEnv, r llm.Request) (json.RawMessage, error) {
				second := strings.Contains(strings.SplitN(r.Messages[1].Content, "\n", 2)[0], "c4, c3, c2, c1")
				raw := s1.P1
				if second {
					raw = s1.P2
				}
				if tc.llmFlag && !tc.short && tc.gateOn {
					raw = withInjectionFlag(t, raw)
				}
				return raw, nil
			})
			ctl := fake.NewController(fake.Settings{DecisionMode: tc.mode})
			if tc.decision != nil {
				ctl.AddDecision(cl, "grader_instruction", fake.DecisionRecord{Noul: tc.decision})
			}
			if tc.gateOn {
				g.grader.Decide = llmtest.Gateway(ctl, g.course, llm.GateInjection)
			} else {
				g.grader.Decide = llmtest.Gateway(ctl, g.course) // có provider nhưng lớp chưa bật
			}
			res := g.submit("SV", "bai.md", content, nil)
			out, err := g.grader.Run(t.Context(), g.course, res)
			require.NoError(t, err)
			require.Equal(t, "DONE", out.State)
			_, _, flags, total, _, _, _ := g.row(res)
			require.Equal(t, tc.wantFlag, containsStr(flags, "INJECTION_SUSPECTED"), "flags=%v", flags)
			require.Equal(t, tc.wantDec, ctl.Decisions())
			prompts := strings.Join(g.prompts, "\n")
			require.Equal(t, tc.wantInSch, strings.Contains(prompts, "INJECTION_SUSPECTED"), "prompt chấm chỉ có luật cờ khi cổng hiệu lực và bài ≥ 20 từ")
			if !tc.short {
				require.Equal(t, s1.Expected.Total, *total, "cờ không đổi điểm (TLR-4)")
			}
		})
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
