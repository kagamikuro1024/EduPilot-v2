package practice_test

// QC smoke S10 (11-18): TC-09-11/12/13 — WrongAnswerSignals k-ẩn danh (bản sao tạm, không commit vào nhánh sprint).

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/practice"
)

func TestQC9sSignals(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	r.approved("P", 5)
	r.approved("Q", 5)
	wrong := func(sv uuid.UUID, topic string) {
		v, _, err := r.svc.Start(ctx, sv, r.course, practice.StartIn{Mode: "TOPIC", Topic: topic, Count: new(5)}, uuid.Nil)
		require.NoError(t, err)
		_, err = r.svc.SaveAnswer(ctx, sv, r.course, v.ID, v.Items[0].ID, uuid.Nil, opt(r.wrongOption(v.Items[0].ID))) //nolint:contextcheck // helper của rig
		require.NoError(t, err)
	}
	var svs []uuid.UUID
	for range 3 {
		sv := r.student()
		svs = append(svs, sv)
		wrong(sv, "P")
	}
	wrong(svs[0], "Q")
	wrong(svs[1], "Q") // Q: 2 SV
	from, to := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	sig, err := r.svc.WrongAnswerSignals(ctx, r.course, from, to)
	require.NoError(t, err)
	// TC-09-11: P có (≥3), Q không (k-ẩn danh)
	require.Len(t, sig, 1)
	require.Equal(t, "P", sig[0].Topic)
	require.GreaterOrEqual(t, sig[0].StudentsAffected, 3)
	raw, _ := json.Marshal(sig)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(raw, &rows))
	keys := []string{}
	for k := range rows[0] {
		keys = append(keys, k)
	}
	require.ElementsMatch(t, []string{"topic", "wrong_answers", "students_affected", "sample_question_ids"}, keys)
	for _, sv := range svs {
		require.NotContains(t, string(raw), sv.String())
	}
	// TC-09-12: ngoài dữ liệu / from > to → rỗng hoặc lỗi rõ, không panic
	none, err := r.svc.WrongAnswerSignals(ctx, r.course, time.Now().Add(-72*time.Hour), time.Now().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Empty(t, none)
	require.NotPanics(t, func() {
		rev, err2 := r.svc.WrongAnswerSignals(ctx, r.course, to, from)
		if err2 == nil {
			require.Empty(t, rev)
		}
		t.Logf("from>to: len=%d err=%v", len(rev), err2)
	})
	// TC-09-13: SV thứ 3 sai chủ đề Q → Q xuất hiện với students_affected=3
	wrong(svs[2], "Q")
	sig2, err := r.svc.WrongAnswerSignals(ctx, r.course, from, to)
	require.NoError(t, err)
	got := map[string]int{}
	for _, s := range sig2 {
		got[s.Topic] = s.StudentsAffected
	}
	require.Equal(t, 3, got["Q"])
	require.Equal(t, 3, got["P"])
}
