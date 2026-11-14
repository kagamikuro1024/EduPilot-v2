package fake_test

import (
	"context"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

func qs() map[string]provider.DecideQuestion {
	return map[string]provider.DecideQuestion{
		"p": {Type: provider.QuestionNoul, Instructions: "?"},
		"i": {Type: provider.QuestionChoice, Instructions: "?", Criteria: map[string]string{"A": "a", "B": "b"}},
	}
}

func decide(t *testing.T, s fake.Settings, in string) (provider.DecideResult, *fake.Controller, error) {
	t.Helper()
	c := fake.NewController(s)
	res, err := fake.NewNamed(c, "d", "").Decide(t.Context(), provider.DecideOpts{Model: "decision-fake", Input: in, Questions: qs()})
	return res, c, err
}

// TestDecideFakeDeterministic: cùng đầu vào → cùng kết quả; đầu vào lạ → noul 0,25 và nhãn chia đều (ép fallback); replay thắng; đếm riêng với Calls.
func TestDecideFakeDeterministic(t *testing.T) {
	t.Parallel()
	one := 0.9
	c := fake.NewController(fake.Settings{})
	c.AddDecision("tên người", "p", fake.DecisionRecord{Noul: &one})
	c.AddDecision("tên người", "i", fake.DecisionRecord{Choice: "B", Probabilities: map[string]float64{"A": 0.03, "B": 0.97}})
	p := fake.NewNamed(c, "d", "")
	for range 3 {
		r, err := p.Decide(t.Context(), provider.DecideOpts{Model: "m", Input: "tên người", Questions: qs()})
		if err != nil || r.Answers["p"].Noul != 0.9 || r.Answers["i"].Choice != "B" || r.Answers["i"].Confidence != 0.97 {
			t.Fatalf("replay: %+v %v", r, err)
		}
	}
	r, _ := p.Decide(t.Context(), provider.DecideOpts{Model: "m", Input: "lạ", Questions: qs()})
	if r.Answers["p"].Noul != 0.25 || r.Answers["i"].Confidence != 0.5 {
		t.Fatalf("đầu vào lạ: %+v", r.Answers)
	}
	if c.Decisions() != 4 || c.Calls() != 0 {
		t.Fatalf("decisions=%d calls=%d", c.Decisions(), c.Calls())
	}
}

// TestDecideFakeOverrides: FAKE_DECISION_PROB ghi đè; các chế độ lỗi; độ trễ theo p50.
func TestDecideFakeOverrides(t *testing.T) {
	t.Parallel()
	prob := 0.97
	c0 := fake.NewController(fake.Settings{DecisionProb: &prob})
	hit := 0.1
	c0.AddDecision("có trong replay", "p", fake.DecisionRecord{Noul: &hit})
	if r, _ := fake.NewNamed(c0, "d", "").Decide(t.Context(), provider.DecideOpts{Model: "m", Input: "có trong replay", Questions: qs()}); r.Answers["p"].Noul != 0.1 {
		t.Fatalf("replay phải thắng FAKE_DECISION_PROB: %+v", r.Answers["p"])
	}
	r, _, err := decide(t, fake.Settings{DecisionProb: &prob}, "bất kỳ")
	if err != nil || r.Answers["p"].Noul != 0.97 || r.Answers["i"].Confidence != 0.97 {
		t.Fatalf("prob: %+v %v", r.Answers, err)
	}
	if r, _, _ := decide(t, fake.Settings{DecisionMode: "always_yes"}, "x"); r.Answers["p"].Noul != 0.99 {
		t.Fatalf("always_yes: %+v", r.Answers["p"])
	}
	if r, _, _ := decide(t, fake.Settings{DecisionMode: "always_no"}, "x"); r.Answers["p"].Noul != 0.01 {
		t.Fatalf("always_no: %+v", r.Answers["p"])
	}
	if _, _, err := decide(t, fake.Settings{DecisionMode: "error"}, "x"); err == nil {
		t.Fatal("error mode")
	}
	if _, _, err := decide(t, fake.Settings{DecisionErrorRate: 1}, "x"); err == nil {
		t.Fatal("error rate 1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	c := fake.NewController(fake.Settings{DecisionMode: "timeout"})
	if _, err := fake.NewNamed(c, "d", "").Decide(ctx, provider.DecideOpts{Model: "m", Input: "x", Questions: qs()}); err == nil {
		t.Fatal("timeout mode phải trả lỗi khi ctx hết hạn")
	}
	start := time.Now()
	_, _, _ = decide(t, fake.Settings{DecisionP50: 60 * time.Millisecond, DecisionP95: 60 * time.Millisecond}, "x")
	if time.Since(start) < 50*time.Millisecond {
		t.Fatal("độ trễ không áp dụng")
	}
}
