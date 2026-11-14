package thread_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/thread"
)

// TestRegressionCaseThread — M2: bài AI bị REJECT / CORRECT ghi ca THREAD_REJECTED / THREAD_CORRECTED (nguồn = id bài, expected chỉ {state}); VERIFY không ghi; cờ tắt không ghi.
func TestRegressionCaseThread(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	count := func(kind string) int {
		return r.count(`select count(*) from grading_regression_cases where course_id = $1 and kind = $2`, r.course, kind)
	}
	a, b, c := r.answered(), r.answered(), r.answered()
	_, err := r.svc.Decide(ctx, r.ta, r.course, r.aiPost(a), thread.Reject, "", 0)
	require.NoError(t, err)
	require.Zero(t, count("THREAD_REJECTED"), "cờ tắt")

	r.svc.Regress = suggestGate(true)
	_, err = r.svc.Decide(ctx, r.ta, r.course, r.aiPost(b), thread.Reject, "", 0)
	require.NoError(t, err)
	var ver int32
	require.NoError(t, r.pool.QueryRow(ctx, `select version from forum_posts where id = $1`, r.aiPost(c)).Scan(&ver))
	_, err = r.svc.Decide(ctx, r.ta, r.course, r.aiPost(c), thread.Correct, "Đáp án đúng là Điều 5.", ver)
	require.NoError(t, err)
	require.Equal(t, 1, count("THREAD_REJECTED"))
	require.Equal(t, 1, count("THREAD_CORRECTED"))
	var exp, text string
	require.NoError(t, r.pool.QueryRow(ctx, `select expected::text from grading_regression_cases where course_id = $1 and kind = 'THREAD_CORRECTED'`, r.course).Scan(&exp))
	require.JSONEq(t, `{"state":"CORRECTED"}`, exp)
	require.NoError(t, r.pool.QueryRow(ctx, `select source_id::text from grading_regression_cases where course_id = $1 and kind = 'THREAD_CORRECTED'`, r.course).Scan(&text))
	require.Equal(t, r.aiPost(c).String(), text, "nguồn là id bài, không chữ")
	v := r.answered()
	_, err = r.svc.Decide(ctx, r.ta, r.course, r.aiPost(v), thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, 1, count("THREAD_REJECTED")+count("THREAD_CORRECTED")-1, "VERIFY không ghi ca")
}
