package exam_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/clock"
)

// TestAutoSubmitPoisonedAttempt — DEF-R7S-EXAMTICK (hướng B): lượt có computeGrade lỗi (bài 0 mục → ErrNoPoints, chỉ chèn được bằng SQL) bị tick chọn lại mãi.
// Sau 5 lần lỗi: không chọn lại (AutoSubmitDue trả nil), lượt vẫn IN_PROGRESS, đúng MỘT dòng log (id, không PII), lượt lành cùng lô không bị chặn.
func TestAutoSubmitPoisonedAttempt(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	bad := r.openExam("bài 0 mục", false, r.mcqSet(1)...)
	good := r.openExam("bài lành", false, r.mcqSet(1)...)
	sv1, sv2 := r.student("ACTIVE"), r.student("ACTIVE")
	clk := clock.NewFake(time.Now().UTC())
	var buf bytes.Buffer
	svc := r.at(clk)
	svc.Log = newLogger(&buf)
	vb, _, err := svc.StartAttempt(t.Context(), sv1, r.course, bad.ID, uuid.New())
	require.NoError(t, err)
	vg, _, err := svc.StartAttempt(t.Context(), sv2, r.course, good.ID, uuid.New())
	require.NoError(t, err)
	r.exec(`delete from exam_items where exam_id=$1`, bad.ID) // dữ liệu hỏng: lượt đang làm mà bài không còn mục
	clk.Advance(45*time.Minute + 11*time.Second)

	var failed int
	for i := range 10 {
		n, err := svc.AutoSubmitDue(t.Context())
		if err != nil {
			failed++
		}
		if i == 0 {
			require.Equal(t, 1, n, "lượt lành cùng lô vẫn được nộp ở lần quét đầu")
		}
	}
	require.Equal(t, 5, failed, "lỗi đúng 5 lần rồi thôi chọn lại")
	st, _, _, _ := r.attemptRow(vb.Attempt.ID)
	require.Equal(t, "IN_PROGRESS", st, "không đổi trạng thái, không đổi điểm")
	st, _, _, _ = r.attemptRow(vg.Attempt.ID)
	require.Equal(t, "GRADED", st)
	require.Equal(t, 1, strings.Count(buf.String(), "\n"), "đúng MỘT dòng log")
	require.Contains(t, buf.String(), vb.Attempt.ID.String())
}
