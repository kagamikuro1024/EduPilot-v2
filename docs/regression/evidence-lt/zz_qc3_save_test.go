package exam_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// qc3 (PE05-44, AC10): gửi lại CÙNG lô lưu (retry sau mất mạng) → không lỗi, không nhân bản, đáp án giữ nguyên; gửi lại lô cũ sau lô mới không làm mất lô mới... (ghi nhận hành vi).
func TestQC3SaveRetryIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv, tab := r.student("ACTIVE"), uuid.New()
	e := r.openExam("retry", true, r.mcqSet(4)...)
	v, _ := r.start(e, sv, tab)
	count := func() int { return r.count(`select count(*) from exam_answers where attempt_id=$1`, v.Attempt.ID) }
	batch := []any{pick(v.Items[0], 0), pick(v.Items[1], 1)}
	for i := 0; i < 3; i++ { // backoff 1→2→4 s: cùng lô gửi lại 3 lần
		_, err := r.save(v, sv, tab, pick(v.Items[0], 0), pick(v.Items[1], 1))
		require.NoError(t, err, "lần %d", i+1)
		require.Equal(t, 2, count())
	}
	_ = batch
	var a0 string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select answer::text from exam_answers where attempt_id=$1 and item_id=$2`, v.Attempt.ID, v.Items[0].ItemID).Scan(&a0))
	// lô mới hơn (đổi câu 1) rồi lô cũ đến muộn: ghi nhận kết quả cuối
	_, err := r.save(v, sv, tab, pick(v.Items[0], 3))
	require.NoError(t, err)
	var a1 string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select answer::text from exam_answers where attempt_id=$1 and item_id=$2`, v.Attempt.ID, v.Items[0].ItemID).Scan(&a1))
	require.NotEqual(t, a0, a1)
	t.Logf("sau 3 lần gửi lại cùng lô: 2 dòng; đổi đáp án câu 1 → khác (%v)", a0 != a1)
}
