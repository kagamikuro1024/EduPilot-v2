package practice_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/practice"
)

type hintGate bool

func (g hintGate) On(uuid.UUID, string) bool { return bool(g) }

// TestRejectReasonHints — M6 (FEAT-grading-guidance 4.5): lý do loại gom thành khối «Tránh…» CỐ ĐỊNH theo nhãn khi ≥ MIN câu có lý do; < MIN, cờ tắt hoặc chỉ UNSPECIFIED → không khối;
// lý do lạ → 422; khối chỉ chứa câu cố định (không chữ người dùng); lý do đã dùng lưu vào kết quả lô.
func TestRejectReasonHints(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.staff()
	ctx := t.Context()
	reject := func(reason *string, n int) {
		for range n {
			id := r.approvedPending(t, "GENERATED") //nolint:contextcheck // rig dùng t.Context()
			out, err := r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "REJECT", IDs: []uuid.UUID{id}, Reason: reason})
			require.NoError(t, err)
			require.True(t, out[0].OK, "%+v", out)
		}
	}
	mo, trung, unspec := "MO_HO", "TRUNG_LAP", "UNSPECIFIED"
	lecture := r.doc("LECTURE", "Bài giảng", "Đoạn một về mạng máy tính.")
	in := practice.GenerateIn{DocumentIDs: []uuid.UUID{lecture}, Topic: "Mạng", Difficulty: "MEDIUM", Count: 1}
	run := func(gate hintGate) (prompt string, res practice.ExtractResult) {
		stub := &fakeStructured{reply: func(_, _ string) string { return `{"questions":[]}` }}
		w := &practice.Worker{Pool: r.pool, Exam: r.ex, LLM: stub, Hints: &exam.Hints{Gate: gate, MinRejects: 5, WindowDays: 90}}
		status, raw, ec := r.runJob(w, r.teacher, practice.KindGenerate, practice.GeneratePayload{CourseID: r.course, GenerateIn: in})
		require.Equal(t, "SUCCEEDED", status, ec)
		require.NoError(t, json.Unmarshal(raw, &res))
		return strings.Join(stub.prompts, "\n"), res
	}

	reject(&mo, 3)
	reject(&unspec, 3) // UNSPECIFIED không tính
	reject(nil, 2)     // không lý do: không tính
	p, res := run(true)
	require.NotContains(t, p, "TRÁNH", "mới 3 câu có lý do < 5")
	require.Empty(t, res.Hints)

	reject(&trung, 2)
	p, res = run(true)
	require.Contains(t, p, "TRÁNH")
	require.Contains(t, p, exam.RejectHintText["MO_HO"])
	require.Contains(t, p, exam.RejectHintText["TRUNG_LAP"])
	require.NotContains(t, p, exam.RejectHintText["LAC_DE"])
	require.Equal(t, []string{"MO_HO", "TRUNG_LAP"}, res.Hints, "nhiều nhất trước; lưu vào kết quả lô")
	p, res = run(false)
	require.NotContains(t, p, "TRÁNH", "cờ lớp tắt: prompt như trước")
	require.Empty(t, res.Hints)

	// lý do lạ / lý do kèm APPROVE → 422
	id := r.approvedPending(t, "GENERATED")
	bad := "TUY_THICH"
	_, err := r.ex.ReviewWithReason(ctx, r.teacher, r.course, id, exam.DecisionReject, 1, &bad)
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
	_, err = r.ex.ReviewWithReason(ctx, r.teacher, r.course, id, exam.DecisionApprove, 1, &mo)
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
}
