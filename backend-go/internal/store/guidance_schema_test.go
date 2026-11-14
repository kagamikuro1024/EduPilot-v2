package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGuidanceTriggerImmutable — SRS FEAT-grading-guidance 5.3: sau run_state='DONE' cấm đổi ai_comments / guidance_version; khi QUEUED / RUNNING được (thử lại ghim lại).
func TestGuidanceTriggerImmutable(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	_, err := conn.Exec(ctx, f.q(`update grading_results set run_state = 'RUNNING', guidance_version = 1, ai_comments = '{"criteria":{"c1":"a"}}' where id = '{res1}'`))
	require.NoError(t, err, "trước DONE được ghim lại")
	_, err = conn.Exec(ctx, f.q(`update grading_results set guidance_version = 2 where id = '{res1}'`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.q(`update grading_results set run_state = 'DONE' where id = '{res1}'`))
	require.NoError(t, err)
	for _, u := range []string{`guidance_version = 3`, `ai_comments = NULL`, `ai_comments = '{"criteria":{}}'`} {
		_, err = conn.Exec(ctx, f.q(`update grading_results set `+u+` where id = '{res1}'`))
		require.Equal(t, "P0001", sqlState(err), u)
	}
}

// TestFirstReviewedImmutable — P-106-5: first_reviewed_* ghi một lần; ReviewApplyEdit xoá reviewed_at nhưng giữ first_reviewed_at; cặp cột cùng NULL / cùng có.
func TestFirstReviewedImmutable(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	_, err := conn.Exec(ctx, f.q(`update grading_results set first_reviewed_at = now() where id = '{res1}'`))
	require.Equal(t, "23514", sqlState(err), "first_reviewed_at mà thiếu first_reviewed_by")
	_, err = conn.Exec(ctx, f.q(`update grading_results set run_state = 'DONE', status = 'REVIEWED', reviewed_by = '{t}', reviewed_at = now(), first_reviewed_by = '{t}', first_reviewed_at = now() where id = '{res1}'`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.q(`update grading_results set status = 'DRAFT', reviewed_by = NULL, reviewed_at = NULL where id = '{res1}'`))
	require.NoError(t, err, "sửa lại (REVIEWED → DRAFT) xoá reviewed_at")
	var first bool
	require.NoError(t, conn.QueryRow(ctx, f.q(`select first_reviewed_at is not null from grading_results where id = '{res1}'`)).Scan(&first))
	require.True(t, first, "first_reviewed_at còn nguyên")
	for _, u := range []string{`first_reviewed_at = now() + interval '1 day'`, `first_reviewed_at = NULL, first_reviewed_by = NULL`, `first_reviewed_by = '{s1}'`} {
		_, err = conn.Exec(ctx, f.q(`update grading_results set `+u+` where id = '{res1}'`))
		require.Equal(t, "P0001", sqlState(err), u)
	}
}

// TestGuidanceItemChecks — trạng thái ↔ decided_by, phiên bản nghỉ > phiên bản duyệt; xoá bài tập kéo theo lưu ý (CASCADE).
func TestGuidanceItemChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	ins := func(cols, vals string) error {
		_, err := conn.Exec(ctx, f.q(`insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support`+cols+`) values ('{c1}', '{a1}', 'c1', 'Lưu ý', 'STRICTER', 2`+vals+`)`))
		return err
	}
	require.NoError(t, ins("", ""))
	require.Equal(t, "23514", sqlState(ins(", status", ", 'APPROVED'")), "APPROVED thiếu decided_by")
	require.Equal(t, "23514", sqlState(ins(", decided_by", ", '{t}'")), "DRAFT mà có decided_by")
	require.Equal(t, "23514", sqlState(ins(", status, decided_by, approved_in_version, retired_in_version", ", 'RETIRED', '{t}', 2, 2")), "nghỉ phải sau duyệt")
	require.NoError(t, ins(", status, decided_by, approved_in_version, retired_in_version", ", 'RETIRED', '{t}', 1, 2"))
	_, err := conn.Exec(ctx, f.q(`insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support) values ('{c1}', '{a1}', 'c1', '', 'STRICTER', 2)`))
	require.Equal(t, "23514", sqlState(err), "chữ rỗng")
	_, err = conn.Exec(ctx, f.q(`insert into grading_guidance_versions (course_id, assignment_id, version, item_count, created_by) values ('{c1}', '{a1}', 1, 1, '{t}')`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.q(`insert into grading_guidance_versions (course_id, assignment_id, version, item_count, created_by) values ('{c1}', '{a1}', 1, 1, '{t}')`))
	require.Equal(t, "23505", sqlState(err), "phiên bản duy nhất theo bài tập")
}

// TestRegressionCaseUnique — (lớp, loại, nguồn, tiêu chí) duy nhất kể cả criterion_id NULL (coalesce).
func TestRegressionCaseUnique(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	ins := func(crit string) error {
		_, err := conn.Exec(ctx, f.q(`insert into grading_regression_cases (course_id, kind, source_id, criterion_id, expected) values ('{c1}', 'THREAD_CORRECTED', '{res1}', `+crit+`, '{"state":"CORRECTED"}')`))
		return err
	}
	require.NoError(t, ins("NULL"))
	require.Equal(t, "23505", sqlState(ins("NULL")))
	require.NoError(t, ins("'c1'"))
	_, err := conn.Exec(ctx, f.q(`insert into grading_regression_cases (course_id, kind, source_id, expected) values ('{c1}', 'GRADING_EDIT', '{res1}', '[]')`))
	require.Equal(t, "23514", sqlState(err), "expected phải là object")
}
