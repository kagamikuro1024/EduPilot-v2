package grade_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// reviewedSub chèn bài nộp MATCHED (phiên bản v) + kết quả REVIEWED với MỘT tiêu chí điểm `total` (đủ để công bố qua assignment.PublishResults).
func (r *asgRig) reviewedSub(student uuid.UUID, version int, total string) uuid.UUID {
	r.t.Helper()
	var sub, res uuid.UUID
	r.one(&sub, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ($1, $2, $3, 'IN_APP', $4, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP') returning id`, r.course, r.asg, student, version)
	r.one(&res, `insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, ai_scores)
		values ($1, $2, 'DONE', 'REVIEWED', jsonb_build_array(jsonb_build_object('id', 'c1', 'score', $3::text)), $3::numeric, $3::numeric, jsonb_build_object('c1', $3::text)) returning id`, r.course, sub, total)
	return res
}

func (r *asgRig) publish(ids ...uuid.UUID) error {
	r.t.Helper()
	_, err := (&assignment.Service{Pool: r.pool}).PublishResults(r.ctx, assignment.Actor{UserID: r.gv}, r.course, assignment.PublishIn{ResultIDs: ids})
	return err
}

func (r *asgRig) entries() (n int) {
	r.one(&n, `select count(*)::int from grade_entries where course_id = $1 and item_id = $2`, r.course, r.item)
	return n
}

// TestAssignmentItemReadsPublishedDirect — AC2: item ASSIGNMENT đọc thẳng grading_results PUBLISHED; không có hàng grade_entries; bài chưa công bố (REVIEWED) không vào điểm.
func TestAssignmentItemReadsPublishedDirect(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	a, b := r.reviewedSub(r.students[0], 1, "8.5"), r.reviewedSub(r.students[1], 1, "9")
	_, early := r.inputs()[r.students[0]].Scores[r.item]
	require.False(t, early, "REVIEWED chưa công bố: không có điểm")
	require.NoError(t, r.publish(a))
	in := r.inputs()
	require.True(t, in[r.students[0]].Scores[r.item].Equal(decimal.RequireFromString("8.5")))
	_, has := in[r.students[1]].Scores[r.item]
	require.False(t, has)
	require.Zero(t, r.entries(), "không chép sang grade_entries (TLR-14 A)")
	_ = b
}

// TestPublishWritesNoGradeEntries — AC2: công bố cả lô không ghi grade_entries nào của lớp.
func TestPublishWritesNoGradeEntries(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	var before, after int
	r.one(&before, `select count(*)::int from grade_entries where course_id = $1`, r.course)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 1, "7"), r.reviewedSub(r.students[1], 1, "6.5")))
	r.one(&after, `select count(*)::int from grade_entries where course_id = $1`, r.course)
	require.Equal(t, before, after)
}

// TestAssignmentItemBatchNoNPlusOne — AC2: số truy vấn LoadInputs không phụ thuộc số sinh viên (đúng một lô cho item ASSIGNMENT).
func TestAssignmentItemBatchNoNPlusOne(t *testing.T) {
	count := func(n int) int64 {
		d, c := countingDB(t)
		r := newAsgRigOn(t, d, n, "-1 day", false)
		for i, st := range r.students {
			r.sub(st, 1, "PUBLISHED", decimal.NewFromInt(int64(5+i%5)).String())
		}
		before := c.n.Load()
		in, err := grade.LoadInputs(r.ctx, r.q, r.course, r.scheme, r.students)
		require.NoError(t, err)
		require.Len(t, in, n)
		return c.n.Load() - before
	}
	require.Equal(t, count(3), count(60), "số truy vấn không tăng theo số sinh viên")
}

// TestAssignmentItemEqualsStudentView — AC2/AC6: điểm trong sổ bằng điểm sinh viên thấy ở /me/grades (đã trừ nộp muộn).
func TestAssignmentItemEqualsStudentView(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	res := r.reviewedSub(r.students[0], 1, "8")
	require.NoError(t, r.publish(res))
	page, err := r.svc().Gradebook(r.ctx, r.course, "", 50, true)
	require.NoError(t, err)
	var cell string
	for _, row := range page.Rows {
		if row.Student.ID == r.students[0] {
			cell = row.Entries[r.item.String()].Score
			require.Equal(t, "ASSIGNMENT", row.Entries[r.item.String()].Source)
		}
	}
	me, err := r.svc().MeGrades(r.ctx, r.course, r.students[0])
	require.NoError(t, err)
	var got string
	for _, c := range me.Components {
		if c.Key == "weekly" {
			require.Equal(t, "PUBLISHED", c.Status)
			got = c.Score
		}
	}
	require.True(t, decimal.RequireFromString(cell).Equal(decimal.RequireFromString(got)), "sổ %s ≠ sinh viên thấy %s", cell, got)
	other, err := r.svc().MeGrades(r.ctx, r.course, r.students[1])
	require.NoError(t, err)
	for _, c := range other.Components {
		if c.Key == "weekly" {
			require.Equal(t, "PENDING", c.Status, "sinh viên chưa có kết quả công bố: PENDING, không điểm")
			require.Empty(t, c.Score)
		}
	}
}

// TestAssignmentItemLateLinkAfterPublish — AC2: gắn bài vào công thức SAU khi đã công bố vẫn có điểm ngay (không cần mang điểm).
func TestAssignmentItemLateLinkAfterPublish(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.exec(`update grade_items set source = 'MANUAL', assignment_id = null where id = $1`, r.item)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 1, "9")))
	r.exec(`update grade_items set source = 'ASSIGNMENT', assignment_id = $2 where id = $1`, r.item, r.asg)
	require.True(t, r.inputs()[r.students[0]].Scores[r.item].Equal(decimal.NewFromInt(9)))
}

// TestAssignmentItemNotEditable — AC2: ô item ASSIGNMENT không sửa tay / không nhập CSV (409 ITEM_NOT_EDITABLE).
func TestAssignmentItemNotEditable(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	sc := "5"
	_, err := r.svc().PutEntry(r.ctx, r.gv, r.course, grade.PutEntryInput{ItemID: r.item, StudentID: r.students[0], Score: &sc})
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apierr.ItemNotEditable, ae.Code)
	require.Equal(t, http.StatusConflict, ae.Status)
	require.Zero(t, r.entries())
}

// TestAssignmentItemNotPublishable — AC2: cột ASSIGNMENT không công bố bằng route cột (409 ITEM_NOT_PUBLISHABLE).
func TestAssignmentItemNotPublishable(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	_, err := r.svc().PublishItem(r.ctx, r.gv, r.course, r.item, true)
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apierr.ItemNotPublishable, ae.Code)
}

// TestExpectedFinalGradesUnchanged — AC2: công bố bài tập không đổi 30 dòng điểm mẫu: băm tệp khớp `.sha256` (lớp 1 không có item bài tập).
func TestExpectedFinalGradesUnchanged(t *testing.T) {
	raw, err := os.ReadFile(seedDir + "expected_final_grades.csv")
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	want, err := os.ReadFile(seedDir + "expected_final_grades.csv.sha256")
	require.NoError(t, err)
	require.Equal(t, strings.Fields(string(want))[0], hex.EncodeToString(sum[:]))
	require.NotContains(t, string(raw), "ASSIGNMENT")
}

// lockedWithAllPublished chốt khi mọi sinh viên có điểm bài tập đã công bố.
func (r *asgRig) lockedWithAllPublished() {
	r.t.Helper()
	var ids []uuid.UUID
	for i, st := range r.students {
		ids = append(ids, r.reviewedSub(st, 1, decimal.NewFromInt(int64(6+i)).String()))
	}
	require.NoError(r.t, r.publish(ids...))
	_, err := r.finalize()
	require.NoError(r.t, err)
}

// TestPublishWhenLockedKeepsSnapshot — AC3: sổ LOCKED vẫn công bố được; snapshot không đổi.
func TestPublishWhenLockedKeepsSnapshot(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.lockedWithAllPublished()
	before := r.snapHash(1)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 2, "10")), "không GRADEBOOK_LOCKED ở đây")
	require.Equal(t, before, r.snapHash(1), "bản chốt giữ nguyên")
	st, v := r.status()
	require.Equal(t, "LOCKED", st)
	require.Equal(t, 1, v)
}

// TestPublishShowsDriftWhenLocked — AC3: sau khi công bố điểm mới, hàng sinh viên hiện drift ("Khác bản chốt"), hàng khác không.
func TestPublishShowsDriftWhenLocked(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.lockedWithAllPublished()
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 2, "10")))
	page, err := r.svc().Gradebook(r.ctx, r.course, "", 50, true)
	require.NoError(t, err)
	for _, row := range page.Rows {
		if row.Student.ID == r.students[0] {
			require.True(t, row.Drift, "sinh viên có điểm bài tập mới phải hiện drift")
		} else {
			require.False(t, row.Drift)
		}
	}
}

// TestPublishThenUnlockRefinalizeNotifies — AC3: Mở khoá + Chốt lại → chỉ sinh viên đổi điểm học phần nhận GRADE_CHANGED.
func TestPublishThenUnlockRefinalizeNotifies(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.lockedWithAllPublished()
	r.handle(1)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 2, "10")))
	require.NoError(t, r.unlock("Công bố bài tập mới, chốt lại"))
	_, err := r.finalize()
	require.NoError(t, err)
	r.handle(2)
	require.Equal(t, 1, r.notes("GRADE_CHANGED"))
	var u uuid.UUID
	r.one(&u, `select user_id from notifications where course_id = $1 and type = 'GRADE_CHANGED'`, r.course)
	require.Equal(t, r.students[0], u)
}

// TestPublishSerializesWithFinalize — AC3: công bố mở đầu bằng FOR SHARE hàng gradebooks nên đợi `Chốt điểm` (FOR UPDATE) xong mới chạy.
func TestPublishSerializesWithFinalize(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	res := r.reviewedSub(r.students[0], 1, "8")
	r.exec(`insert into gradebooks (course_id) values ($1) on conflict (course_id) do nothing`, r.course)
	tx, err := r.pool.Begin(r.ctx)
	require.NoError(t, err)
	_, err = tx.Exec(r.ctx, `select status from gradebooks where course_id = $1 for update`, r.course)
	require.NoError(t, err)
	var done atomic.Bool
	errc := make(chan error, 1)
	go func() {
		errc <- r.publish(res)
		done.Store(true)
	}()
	time.Sleep(500 * time.Millisecond)
	require.False(t, done.Load(), "công bố phải đợi khi sổ đang bị khoá để chốt")
	require.NoError(t, tx.Commit(r.ctx))
	require.NoError(t, <-errc)
}

func (d *tdb) assignmentFor(course, gv uuid.UUID, max string) uuid.UUID {
	d.t.Helper()
	var id uuid.UUID
	d.one(&id, `insert into assignments (course_id, title, max_score, due_at, submission_code, created_by) values ($1, 'BT', $2::numeric, now() + interval '3 days', $3, $4) returning id`,
		course, max, "BT"+strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:6], "-", "")), gv)
	return id
}

func linkInput(items ...grade.PutItem) func(*grade.PutInput) {
	return func(in *grade.PutInput) {
		in.Components = append(in.Components, grade.PutComponent{Key: "bai_tap", Name: "Bài tập", Grp: "QT", Weight: "0", Items: items})
	}
}

// TestAssignmentItemLinkValidation — AC2 / SRS 4.9 (1): item ASSIGNMENT cần bài của lớp, thang bằng thang bài (422 ASSIGNMENT_SCALE_MISMATCH), mỗi bài một item (422 ASSIGNMENT_ALREADY_LINKED); bài chưa công bố vẫn gắn được.
func TestAssignmentItemLinkValidation(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	course := d.course(gv, "L1")
	other := d.course(gv, "L2")
	d.scheme(course, gv, 1)
	asg := d.assignmentFor(course, gv, "10")
	foreign := d.assignmentFor(other, gv, "10")
	put := func(items ...grade.PutItem) error {
		in := validInput(t, 1)
		linkInput(items...)(&in)
		_, err := d.svc().PutDraft(d.ctx, gv, course, in)
		return err
	}
	it := func(key string, max string, a *uuid.UUID) grade.PutItem {
		return grade.PutItem{Key: key, Name: key, MaxScore: max, Source: "ASSIGNMENT", AssignmentID: a}
	}
	_, code := apiCode(t, put(it("a", "20", &asg)))
	require.Equal(t, apierr.AssignmentScaleMismatch, code)
	_, code = apiCode(t, put(it("a", "10", &asg), it("b", "10", &asg)))
	require.Equal(t, apierr.AssignmentAlreadyLinked, code)
	_, code = apiCode(t, put(it("a", "10", &foreign)))
	require.Equal(t, apierr.ValidationFailed, code, "bài của lớp khác")
	_, code = apiCode(t, put(it("a", "10", nil)))
	require.Equal(t, apierr.ValidationFailed, code, "thiếu assignment_id")
	require.NoError(t, put(it("a", "10", &asg)), "bài chưa công bố vẫn gắn được")
	var linked uuid.UUID
	d.one(&linked, `select assignment_id from grade_items where course_id = $1 and key = 'a'`, course)
	require.Equal(t, asg, linked)
}

// TestStudentViewSharedByToolAndMeGrades — AC6/AC7: tool `get_my_grade_summary` và `/me/grades` đi cùng MỘT hàm (grade.MeGrades): hai bản JSON giống hệt, kể cả thành phần Bài tập; không có điểm nháp.
func TestStudentViewSharedByToolAndMeGrades(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 1, "8")))
	r.sub(r.students[1], 1, "DRAFT", "9.9")
	for _, st := range r.students[:2] {
		me, err := r.svc().MeGrades(r.ctx, r.course, st)
		require.NoError(t, err)
		want, _ := json.Marshal(me)
		facts, ok, err := agent.GradeAdapter{Grade: r.svc()}.Summary(r.ctx, agent.TrustedContext{UserID: st, CourseID: r.course})
		require.NoError(t, err)
		require.True(t, ok)
		got, _ := json.Marshal(facts)
		require.JSONEq(t, string(want), string(got), "tool và /me/grades phải cùng một DTO")
		require.NotContains(t, string(got), "9.9")
	}
}

// TestMeGradesShowsAssignmentComponent — AC6: `/me/grades` hiện thành phần có item bài tập: PUBLISHED + điểm khi kết quả của chính mình đã công bố, PENDING (không khoá score) khi chưa.
func TestMeGradesShowsAssignmentComponent(t *testing.T) {
	r := newAsgRig(t, "-1 day", false)
	r.sub(r.students[1], 1, "DRAFT", "9.9") // nháp CANARY-AISCORE: không bao giờ lộ
	require.NoError(t, r.publish(r.reviewedSub(r.students[0], 1, "8")))
	pub, err := r.svc().MeGrades(r.ctx, r.course, r.students[0])
	require.NoError(t, err)
	pend, err := r.svc().MeGrades(r.ctx, r.course, r.students[1])
	require.NoError(t, err)
	find := func(m grade.MeGrades) grade.MeComponent {
		for _, c := range m.Components {
			if c.Key == "weekly" {
				return c
			}
		}
		t.Fatal("thiếu thành phần weekly")
		return grade.MeComponent{}
	}
	require.Equal(t, "PUBLISHED", find(pub).Status)
	require.Equal(t, "8", find(pub).Score)
	require.Equal(t, "PENDING", find(pend).Status)
	raw, _ := json.Marshal(pend)
	require.NotContains(t, string(raw), "9.9")
}
