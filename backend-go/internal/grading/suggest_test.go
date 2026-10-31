package grading_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/llm"
)

var refAnswer = strings.Repeat("Đáp án chuẩn của bài tập. ", 6)

func sugg(weights ...int) json.RawMessage {
	var cs []map[string]any
	for i, w := range weights {
		cs = append(cs, map[string]any{"name": "Tiêu chí " + string(rune('A'+i)), "guidance": "Hướng dẫn", "weight_pct": w})
	}
	b, _ := json.Marshal(map[string]any{"criteria": cs})
	return b
}

func code(err error) string {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestSuggestRubricOneCall(t *testing.T) {
	f := constLLM(sugg(40, 60))
	out, err := grading.SuggestRubric(t.Context(), f, refAnswer, decimal.NewFromInt(10))
	require.NoError(t, err)
	require.Equal(t, 1, f.count())
	require.Equal(t, llm.TaskUtility, f.reqs[0].Task)
	require.NotNil(t, f.reqs[0].Params.Temperature)
	require.Zero(t, *f.reqs[0].Params.Temperature)
	require.Equal(t, []string{"4", "6"}, []string{out[0].MaxScore, out[1].MaxScore})
}

func TestSuggestRubricLargestRemainder(t *testing.T) {
	cases := []struct {
		max  string
		w    []int
		want []string
	}{
		{"10", []int{33, 33, 34}, []string{"3.25", "3.25", "3.5"}}, // 3,3 / 3,3 / 3,4 → bước 0,25, Σ = 10
		{"10", []int{25, 25, 25, 25}, []string{"2.5", "2.5", "2.5", "2.5"}},
		{"7", []int{50, 50}, []string{"3.5", "3.5"}},
		{"1", []int{70, 20, 10}, []string{"0.5", "0.25", "0.25"}}, // 4 bước: 2,8 / 0,8 / 0,4 → 3 + 1 phần dư lớn nhất; tiêu chí nhỏ nhất vẫn ≥ 1 bước
	}
	for _, c := range cases {
		max := decimal.RequireFromString(c.max)
		out, err := grading.SuggestRubric(t.Context(), constLLM(sugg(c.w...)), refAnswer, max)
		require.NoError(t, err, c.max)
		sum := decimal.Zero
		var got []string
		for _, o := range out {
			d := decimal.RequireFromString(o.MaxScore)
			require.True(t, d.Mod(decimal.RequireFromString("0.25")).IsZero(), o.MaxScore)
			sum = sum.Add(d)
			got = append(got, o.MaxScore)
		}
		require.True(t, sum.Equal(max), "Σ %s ≠ %s", sum, c.max)
		require.Equal(t, c.want, got)
	}
}

func TestSuggestRubricNoWrite(t *testing.T) {
	// SuggestRubric là hàm thuần trên Structurer: không nhận pool nên không có đường ghi DB; kiểm bằng chữ ký — và đầu ra chưa có id.
	out, err := grading.SuggestRubric(t.Context(), constLLM(sugg(100)), refAnswer, decimal.NewFromInt(10))
	require.NoError(t, err)
	raw, _ := json.Marshal(out)
	require.NotContains(t, string(raw), `"id"`)
}

func TestSuggestRubricInvalid(t *testing.T) {
	for name, raw := range map[string]json.RawMessage{
		"Σ ≠ 100":         sugg(40, 50),
		"không phải JSON": json.RawMessage(`xin chào`),
		"rỗng":            json.RawMessage(`{"criteria":[]}`),
		"trọng số 0":      sugg(0, 100),
		"11 tiêu chí":     sugg(10, 10, 10, 10, 10, 10, 10, 10, 10, 5, 5),
	} {
		_, err := grading.SuggestRubric(t.Context(), constLLM(raw), refAnswer, decimal.NewFromInt(10))
		require.Equal(t, "RUBRIC_SUGGEST_INVALID", code(err), name)
	}
	// max quá nhỏ để chia mỗi tiêu chí ≥ 1 bước
	_, err := grading.SuggestRubric(t.Context(), constLLM(sugg(25, 25, 25, 25)), refAnswer, decimal.RequireFromString("0.75"))
	require.Equal(t, "RUBRIC_SUGGEST_INVALID", code(err))
}

func TestSuggestRubricNeedsAnswer(t *testing.T) {
	f := constLLM(sugg(100))
	for _, ref := range []string{"", "ngắn", strings.Repeat(" ", 80)} {
		_, err := grading.SuggestRubric(t.Context(), f, ref, decimal.NewFromInt(10))
		require.Equal(t, "REFERENCE_ANSWER_REQUIRED", code(err))
	}
	require.Zero(t, f.count(), "không gọi LLM khi thiếu đáp án")
}

func TestSuggestRubricLLMDown(t *testing.T) {
	f := &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) {
		return nil, &llm.ErrUnavailable{Reason: llm.ReasonBudgetExhausted}
	}}
	_, err := grading.SuggestRubric(t.Context(), f, refAnswer, decimal.NewFromInt(10))
	require.Equal(t, "LLM_UNAVAILABLE", code(err))
	f = &fakeLLM{fn: func(int, context.Context) (json.RawMessage, error) { return nil, llm.ErrBadRequest }}
	_, err = grading.SuggestRubric(t.Context(), f, refAnswer, decimal.NewFromInt(10))
	require.Equal(t, "RUBRIC_SUGGEST_INVALID", code(err))
}
