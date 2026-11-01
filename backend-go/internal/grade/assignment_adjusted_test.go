package grade_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
)

// appealed gửi phúc khảo cho sinh viên s (đã có kết quả công bố, tiêu chí c1) và trả id phúc khảo.
func (r *asgRig) appealed(s uuid.UUID) uuid.UUID {
	r.t.Helper()
	r.exec(`update assignments set rubric = '[{"id":"c1","name":"Nội dung","max_score":"10","guidance":""}]'::jsonb, max_score = 10 where id = $1`, r.asg)
	out, err := (&assignment.Service{Pool: r.pool}).FileAppeal(r.ctx, assignment.Actor{UserID: s, Student: true}, r.course, r.asg, assignment.AppealIn{CriteriaIDs: []string{"c1"}, Reason: "Em nghĩ phần này đã đủ ý rồi ạ."})
	require.NoError(r.t, err)
	return out.Appeal.ID
}

func (r *asgRig) decideAdjusted(id uuid.UUID, score string) error {
	_, err := (&assignment.Service{Pool: r.pool}).DecideAppeal(r.ctx, assignment.Actor{UserID: r.gv}, r.course, id, assignment.DecideIn{
		Decision: "ADJUSTED", Response: "Thầy đã xem lại bài của em.", Version: 1, CriteriaScores: []assignment.ScoreEntry{{ID: "c1", Score: assignment.Num(score)}}})
	return err
}

// TestAdjustedUpdatesResultInPlace — AC4: item ASSIGNMENT đọc điểm đã sửa NGAY (không ghi grade_entries).
func TestAdjustedUpdatesResultInPlace(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 1, "6")))
	id := r.appealed(r.students[0])
	require.NoError(t, r.decideAdjusted(id, "9"))
	require.True(t, r.inputs()[r.students[0]].Scores[r.item].Equal(decimal.NewFromInt(9)))
	require.Zero(t, r.entries())
}

// TestAdjustedWhenLockedKeepsSnapshot — AC4: sổ LOCKED không chặn; bản chốt và trạng thái sổ không đổi.
func TestAdjustedWhenLockedKeepsSnapshot(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.lockedWithAllPublished()
	before := r.snapHash(1)
	require.NoError(t, r.decideAdjusted(r.appealed(r.students[0]), "10"), "không GRADEBOOK_LOCKED ở đây")
	require.Equal(t, before, r.snapHash(1))
	st, v := r.status()
	require.Equal(t, "LOCKED", st)
	require.Equal(t, 1, v)
}

// TestAdjustedShowsDriftWhenLocked — AC4: sau khi sửa điểm, hàng sinh viên hiện "Khác bản chốt", hàng khác không.
func TestAdjustedShowsDriftWhenLocked(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.lockedWithAllPublished()
	require.NoError(t, r.decideAdjusted(r.appealed(r.students[0]), "10"))
	page, err := r.svc().Gradebook(r.ctx, r.course, "", 50, true)
	require.NoError(t, err)
	for _, row := range page.Rows {
		require.Equal(t, row.Student.ID == r.students[0], row.Drift)
	}
}

// TestAdjustedAfterUnlockThenRefinalizeNotifies — AC4: Mở khoá + Chốt lại → chỉ sinh viên đổi điểm học phần nhận GRADE_CHANGED.
func TestAdjustedAfterUnlockThenRefinalizeNotifies(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.lockedWithAllPublished()
	r.handle(1)
	require.NoError(t, r.decideAdjusted(r.appealed(r.students[0]), "10"))
	require.NoError(t, r.unlock("Sửa điểm sau phúc khảo, chốt lại"))
	_, err := r.finalize()
	require.NoError(t, err)
	r.handle(2)
	require.Equal(t, 1, r.notes("GRADE_CHANGED"))
	var u uuid.UUID
	r.one(&u, `select user_id from notifications where course_id = $1 and type = 'GRADE_CHANGED'`, r.course)
	require.Equal(t, r.students[0], u)
}

// TestAdjustedSerializesWithFinalize — AC4: quyết định ADJUSTED đợi `Chốt điểm` (FOR UPDATE gradebooks) xong mới chạy.
func TestAdjustedSerializesWithFinalize(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 1, "6")))
	id := r.appealed(r.students[0])
	r.exec(`insert into gradebooks (course_id) values ($1) on conflict (course_id) do nothing`, r.course)
	tx, err := r.pool.Begin(r.ctx)
	require.NoError(t, err)
	_, err = tx.Exec(r.ctx, `select status from gradebooks where course_id = $1 for update`, r.course)
	require.NoError(t, err)
	var done atomic.Bool
	errc := make(chan error, 1)
	go func() {
		errc <- r.decideAdjusted(id, "9")
		done.Store(true)
	}()
	time.Sleep(500 * time.Millisecond)
	require.False(t, done.Load(), "quyết định phải đợi khi sổ đang bị khoá để chốt")
	require.NoError(t, tx.Commit(r.ctx))
	require.NoError(t, <-errc)
}
