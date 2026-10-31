package grade_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/testutil"
)

// ---- hạ tầng ---------------------------------------------------------------------------------------------------------------------------------------

type finRig struct {
	*tdb
	course, gv uuid.UUID
	items      map[string]uuid.UUID
	students   []uuid.UUID
}

// newFinRig: lớp weekly 40 (QT) + final 60 (CK), `n` sinh viên đủ điểm (weekly 8, final 6 + i % 3).
func newFinRig(t *testing.T, n int) *finRig {
	t.Helper()
	d := newTDB(t)
	r := &finRig{tdb: d}
	r.gv = d.user("TEACHER", "GV")
	r.course = d.course(r.gv, "L1")
	doc := d.newDraft(r.course, r.gv, 1, nil)
	d.confirmDoc(r.course, r.gv, doc)
	r.items = d.itemIDs(r.course, doc.ID)
	for i := range n {
		st := d.student(r.course, fmt.Sprintf("SV %02d", i+1), fmt.Sprintf("2022%04d", i+1))
		r.students = append(r.students, st)
		d.entry(r.course, r.items["weekly"], st, r.gv, "8")
		d.entry(r.course, r.items["final"], st, r.gv, fmt.Sprint(6+i%3))
	}
	return r
}

func (r *finRig) finalize() (grade.FinalizeOut, error) {
	return r.svc().Finalize(r.ctx, r.gv, r.course)
}

func (r *finRig) snapCount(version int) int {
	return r.countRows(`select count(*) from final_grade_snapshots where course_id = $1 and version = $2`, r.course, version)
}

func (r *finRig) status() (st string, v int) {
	r.one(&st, `select status::text from gradebooks where course_id = $1`, r.course)
	r.one(&v, `select snapshot_version from gradebooks where course_id = $1`, r.course)
	return
}

func (r *finRig) unlock(reason string) error {
	_, err := r.svc().Unlock(r.ctx, r.gv, r.course, reason)
	return err
}

func (r *finRig) snapHash(version int) string {
	var h string
	r.one(&h, `select coalesce(md5(string_agg(student_id::text || numeric_grade::text || letter || coalesce(failed_reason, '') || breakdown::text, '|' order by student_id)), '') from final_grade_snapshots where course_id = $1 and version = $2`, r.course, version)
	return h
}

func incompleteDetails(t *testing.T, err error) (total int, students []struct {
	ID      uuid.UUID `json:"id"`
	Missing []string  `json:"missing"`
}) {
	t.Helper()
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	raw, _ := json.Marshal(ae.Details)
	var d struct {
		Total    int `json:"total"`
		Students []struct {
			ID      uuid.UUID `json:"id"`
			Missing []string  `json:"missing"`
		} `json:"students"`
	}
	require.NoError(t, json.Unmarshal(raw, &d))
	return d.Total, d.Students
}

func incompleteTotal(t *testing.T, err error) int { n, _ := incompleteDetails(t, err); return n }
func incompleteListLen(t *testing.T, err error) int {
	_, l := incompleteDetails(t, err)
	return len(l)
}

// ---- AC1 -------------------------------------------------------------------------------------------------------------------------------------------

func TestFinalizeSnapshotsAll(t *testing.T) {
	r := newFinRig(t, 5)
	out, err := r.finalize()
	require.NoError(t, err)
	require.Equal(t, 1, out.Version)
	require.Equal(t, 5, out.Students)
	require.False(t, out.LockedAt.IsZero())
	require.Equal(t, 5, r.snapCount(1))
	st, v := r.status()
	require.Equal(t, "LOCKED", st)
	require.Equal(t, 1, v)
	var bd string
	var by uuid.UUID
	var unlockReason *string
	r.one(&bd, `select breakdown::text from final_grade_snapshots where course_id = $1 and version = 1 order by student_id limit 1`, r.course)
	r.one(&by, `select locked_by from final_grade_snapshots where course_id = $1 and version = 1 limit 1`, r.course)
	r.one(&unlockReason, `select unlock_reason from final_grade_snapshots where course_id = $1 and version = 1 limit 1`, r.course)
	require.Equal(t, r.gv, by)
	require.Nil(t, unlockReason, "bản đầu không mang lý do mở khoá")
	var b map[string]any
	require.NoError(t, json.Unmarshal([]byte(bd), &b))
	for _, k := range []string{"scheme", "components", "items", "bonus", "penalty", "qt", "ck", "hp", "letter", "lines"} {
		require.Contains(t, b, k)
	}
	require.NotEmpty(t, b["lines"])
}

func TestFinalizeEqualsPreview(t *testing.T) {
	r := newFinRig(t, 7)
	pre, err := r.svc().Preview(r.ctx, r.course, "", 100)
	require.NoError(t, err)
	_, err = r.finalize()
	require.NoError(t, err)
	snap, err := r.svc().Snapshot(r.ctx, r.course, 1, "", 100)
	require.NoError(t, err)
	want := map[uuid.UUID][2]string{}
	for _, it := range pre.Items {
		want[it.StudentID] = [2]string{it.HP, it.Letter}
	}
	require.Len(t, snap.Items, len(want))
	for _, it := range snap.Items {
		require.Equal(t, want[it.Student.ID], [2]string{it.NumericGrade, it.Letter})
	}
}

// TestFinalizeIdempotent: cùng Idempotency-Key → cùng phản hồi, không snapshot thứ hai.
func TestFinalizeIdempotent(t *testing.T) {
	h := newHTTPRig(t)
	course, gv, st, weekly, final := h.sheet()
	h.entry(course, weekly, st, gv, "8")
	h.entry(course, final, st, gv, "7")
	tok := h.token(gv, auth.RoleTeacher)
	p := "/api/v1/courses/" + course.String() + "/gradebook/finalize"
	s1, o1 := h.do(http.MethodPost, p, tok, "key-final-0001", map[string]any{})
	require.Equal(t, 200, s1)
	s2, o2 := h.do(http.MethodPost, p, tok, "key-final-0001", map[string]any{})
	require.Equal(t, 200, s2)
	require.Equal(t, o1, o2)
	require.Equal(t, 1, h.countRows(`select count(*) from final_grade_snapshots where course_id = $1`, course))
	s3, _ := h.do(http.MethodPost, p, tok, "key-final-0002", map[string]any{})
	require.Equal(t, 409, s3, "khoá mới: sổ đã chốt")
	s4, _ := h.do(http.MethodPost, p, tok, "", map[string]any{})
	require.GreaterOrEqual(t, s4, 400)
	require.Less(t, s4, 500)
}

// TestFinalizeOneTxAndAudit: có dòng audit `finalize`, có outbox `grade.finalized`; audit lỗi → không snapshot, sổ vẫn OPEN (cùng một giao dịch).
func TestFinalizeOneTxAndAudit(t *testing.T) {
	r := newFinRig(t, 3)
	r.exec(`create function audit_boom() returns trigger language plpgsql as $$ begin raise exception 'audit down'; end $$`)
	r.exec(`create trigger audit_boom before insert on audit_log for each row when (new.action = 'finalize') execute function audit_boom()`)
	_, err := r.finalize()
	require.Error(t, err)
	require.Zero(t, r.snapCount(1))
	st, v := r.status()
	require.Equal(t, "OPEN", st)
	require.Zero(t, v)
	require.Zero(t, r.countRows(`select count(*) from outbox where topic = $1`, grade.TopicFinalized))
	r.exec(`drop trigger audit_boom on audit_log`)
	_, err = r.finalize()
	require.NoError(t, err)
	require.Equal(t, 1, r.countRows(`select count(*) from audit_log where course_id = $1 and action = 'finalize' and (after ->> 'students')::int = 3`, r.course))
	require.Equal(t, 1, r.countRows(`select count(*) from outbox where topic = $1`, grade.TopicFinalized))
}

// TestFinalizeLop1MatchesExpected: lớp 1 dựng trong CSDL từ extraction-lop1.json + inputs-lop1.csv; snapshot khớp seed/expected_final_grades.csv 30/30.
func TestFinalizeLop1MatchesExpected(t *testing.T) {
	d := newTDB(t)
	ex := loadExtraction(t, "extraction-lop1.json")
	gv := d.user("TEACHER", "GV")
	course := d.course(gv, "L1")
	d.scheme(course, gv, 1)
	exam := d.exam(course, gv, "PUBLISHED", "10")
	in := grade.PutInput{Version: 1}
	for _, c := range ex.Components {
		pc := grade.PutComponent{Key: c.Key, Name: c.Name, Grp: c.Group, Weight: c.Weight}
		if c.KindHint == "EXAM" {
			pc.Items = []grade.PutItem{{Key: c.Key, Name: c.Name, MaxScore: "10", Source: "EXAM", ExamID: &exam}}
		}
		in.Components = append(in.Components, pc)
	}
	for _, r := range ex.Rules {
		ap := r.Applicable
		in.Rules = append(in.Rules, grade.PutRule{Kind: r.Kind, Applicable: &ap, Params: r.Params})
	}
	doc, err := d.svc().PutDraft(d.ctx, gv, course, in)
	require.NoError(t, err)
	d.confirmDoc(course, gv, doc)
	items := d.itemIDs(course, doc.ID)

	var sessions []uuid.UUID
	for i := 1; i <= 10; i++ {
		sessions = append(sessions, d.session(course, i))
	}
	rows := readCSV(t, fixDir+"inputs-lop1.csv")
	want := expected30(t)
	byCode := map[string]string{}
	for _, row := range rows {
		st := d.student(course, row["account"], row["student_code"])
		byCode[st.String()] = row["account"]
		if row["midterm"] != "" {
			d.entry(course, items[midKey], st, gv, row["midterm"])
		}
		if row["final_exam"] != "" {
			d.entry(course, items[finalKey], st, gv, row["final_exam"])
		}
		if row["weekly_exam_t9"] != "" {
			d.attempt(course, exam, st, row["weekly_exam_t9"], "")
		}
		if row["speak_bonus_points"] != "" && row["speak_bonus_points"] != "0.00" {
			d.part(course, st, gv, "SPEAK", row["speak_bonus_points"])
		}
		if p := row["penalty_event_points"]; p != "" && p != "0.00" {
			d.part(course, st, gv, "PENALTY", "-"+p)
		}
		for i := range atoiMust(row["absent"]) {
			d.attend(course, sessions[i], st, gv, "ABSENT")
		}
	}
	out, err := d.svc().Finalize(d.ctx, gv, course)
	require.NoError(t, err)
	require.Equal(t, 30, out.Students)
	snap, err := d.svc().Snapshot(d.ctx, course, 1, "", 100)
	require.NoError(t, err)
	require.Len(t, snap.Items, 30)
	for _, it := range snap.Items {
		w := want[byCode[it.Student.ID.String()]]
		require.True(t, decimal.RequireFromString(w["hp"]).Equal(decimal.RequireFromString(it.NumericGrade)), "%s: %s ≠ %s", w["account"], w["hp"], it.NumericGrade)
		require.Equal(t, w["letter"], it.Letter, w["account"])
		if strings.HasPrefix(w["verdict"], "LIET") {
			require.NotNil(t, it.FailedReason, w["account"])
			require.Equal(t, w["verdict"], *it.FailedReason)
		} else {
			require.Nil(t, it.FailedReason, w["account"])
		}
	}
}

// ---- AC2 -------------------------------------------------------------------------------------------------------------------------------------------

func TestFinalizeNotConfirmed(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L2")
	d.student(c, "An", "20220001")
	_, err := d.svc().Finalize(d.ctx, gv, c)
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "SCHEME_NOT_CONFIRMED", code)
	// có bản nháp nhưng chưa xác nhận: vẫn 409 SCHEME_NOT_CONFIRMED (chưa có hàng sổ điểm)
	d.newDraft(c, gv, 1, nil)
	_, err = d.svc().Finalize(d.ctx, gv, c)
	_, code = apiCode(t, err)
	require.Equal(t, "SCHEME_NOT_CONFIRMED", code)
}

func TestFinalizeIncompleteAllOrNothing(t *testing.T) {
	r := newFinRig(t, 4)
	for _, st := range r.students[:2] {
		r.exec(`delete from grade_entries where item_id = $1 and student_id = $2`, r.items["final"], st)
	}
	_, err := r.finalize()
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "GRADES_INCOMPLETE", code)
	require.Zero(t, r.snapCount(1))
	s, _ := r.status()
	require.Equal(t, "OPEN", s)
	require.Equal(t, 2, incompleteTotal(t, err))
	require.Equal(t, 2, incompleteListLen(t, err))
}

// TestFinalizeIncomplete40 (TC-08): 40 sinh viên thiếu → danh sách 30, tổng 40.
func TestFinalizeIncomplete40(t *testing.T) {
	r := newFinRig(t, 0)
	r.bulk(r.course, 40)
	_, err := r.finalize()
	require.Equal(t, 40, incompleteTotal(t, err))
	require.Equal(t, 30, incompleteListLen(t, err))
}

func TestFinalizeAlreadyLocked(t *testing.T) {
	r := newFinRig(t, 2)
	_, err := r.finalize()
	require.NoError(t, err)
	_, err = r.finalize()
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "GRADEBOOK_LOCKED", code)
	require.Equal(t, 2, r.snapCount(1))
}

// TC-45: bài thi chưa công bố → thiếu điểm (không phải 0 cho cả lớp).
func TestFinalizeUnpublishedExamIsIncomplete(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	exam := d.exam(c, gv, "CLOSED", "10")
	doc := d.newDraft(c, gv, 1, func(in *grade.PutInput) {
		in.Components[0].Items = []grade.PutItem{{Key: "weekly", Name: "Tuần", MaxScore: "10", Source: "EXAM", ExamID: &exam}}
	})
	d.confirmDoc(c, gv, doc)
	st := d.student(c, "An", "20220001")
	d.attempt(c, exam, st, "8", "")
	d.entry(c, d.itemIDs(c, doc.ID)["final"], st, gv, "7")
	_, err := d.svc().Finalize(d.ctx, gv, c)
	_, code := apiCode(t, err)
	require.Equal(t, "GRADES_INCOMPLETE", code)
}

// ---- AC3 -------------------------------------------------------------------------------------------------------------------------------------------

func TestLockedBlocksWrites(t *testing.T) {
	r := newFinRig(t, 2)
	_, err := r.finalize()
	require.NoError(t, err)
	_, err = put(r.tdb, r.gv, r.course, r.students[0], r.items["weekly"], s("5"), n(1))
	_, code := apiCode(t, err)
	require.Equal(t, "GRADEBOOK_LOCKED", code)
	_, err = r.svc().PublishItem(r.ctx, r.gv, r.course, r.items["weekly"], true)
	_, code = apiCode(t, err)
	require.Equal(t, "GRADEBOOK_LOCKED", code)
	_, err = r.svc().ConfirmScheme(r.ctx, r.gv, r.course, 1)
	_, code = apiCode(t, err)
	require.Contains(t, []string{"GRADEBOOK_LOCKED", "NO_DRAFT", "SCHEME_NOT_DRAFT"}, code)
	page, err := r.svc().Gradebook(r.ctx, r.course, "", 30, true)
	require.NoError(t, err)
	require.Equal(t, "LOCKED", page.State.Status)
	require.Equal(t, 1, page.State.SnapshotVersion)
	require.NotNil(t, page.Rows[0].Locked)
}

// TestSnapshotUnaffectedBySourceChange + TestDriftFlag: sửa nguồn sống không đổi bản chốt; hàng khác bản chốt có cờ drift (chỉ ở Gradebook của Staff).
func TestSnapshotUnaffectedBySourceChange(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	h1 := r.snapHash(1)
	r.exec(`update grade_entries set score = 2, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[0])
	r.part(r.course, r.students[1], r.gv, "SPEAK", "0.25")
	require.Equal(t, h1, r.snapHash(1))
}

func TestDriftFlag(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	page, err := r.svc().Gradebook(r.ctx, r.course, "", 30, true)
	require.NoError(t, err)
	for _, row := range page.Rows {
		require.False(t, row.Drift)
		require.Equal(t, "COMPLETE", row.Result.Status)
	}
	r.exec(`update grade_entries set score = 2, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[0])
	page, err = r.svc().Gradebook(r.ctx, r.course, "", 30, true)
	require.NoError(t, err)
	var drift int
	for _, row := range page.Rows {
		if row.Drift {
			drift++
			require.Equal(t, r.students[0], row.Student.ID)
			require.NotEqual(t, "2", row.Result.HP, "kết quả hiển thị là của bản chốt")
		}
	}
	require.Equal(t, 1, drift)
}

// ---- AC4 -------------------------------------------------------------------------------------------------------------------------------------------

func TestUnlockRequiresReason(t *testing.T) {
	r := newFinRig(t, 2)
	_, err := r.finalize()
	require.NoError(t, err)
	for _, bad := range []string{"", "ngắn", "123456789", strings.Repeat("a", 501), strings.Repeat(" ", 12)} {
		st, code := apiCode(t, r.unlock(bad))
		require.Equal(t, 422, st, bad)
		require.Equal(t, "VALIDATION_FAILED", code, bad)
	}
	s0, _ := r.status()
	require.Equal(t, "LOCKED", s0)
	require.NoError(t, r.unlock(strings.Repeat("a", 10)))
	_, err = r.finalize()
	require.NoError(t, err)
	require.NoError(t, r.unlock(strings.Repeat("b", 500)))
}

func TestUnlockWhenOpen(t *testing.T) {
	r := newFinRig(t, 2)
	st, code := apiCode(t, r.unlock("Phúc khảo sv04: sửa điểm"))
	require.Equal(t, 409, st)
	require.Equal(t, "NOT_LOCKED", code)
}

func TestUnlockAudit(t *testing.T) {
	r := newFinRig(t, 2)
	_, err := r.finalize()
	require.NoError(t, err)
	h1 := r.snapHash(1)
	require.NoError(t, r.unlock("Phúc khảo sv04: sửa điểm giữa kỳ"))
	st, v := r.status()
	require.Equal(t, "OPEN", st)
	require.Equal(t, 1, v)
	var reason string
	var by uuid.UUID
	r.one(&reason, `select unlock_reason from gradebooks where course_id = $1`, r.course)
	r.one(&by, `select unlocked_by from gradebooks where course_id = $1`, r.course)
	require.Equal(t, "Phúc khảo sv04: sửa điểm giữa kỳ", reason)
	require.Equal(t, r.gv, by)
	require.Equal(t, 1, r.countRows(`select count(*) from audit_log where course_id = $1 and action = 'unlock'`, r.course))
	require.Equal(t, h1, r.snapHash(1))
}

func TestUnlockRBAC(t *testing.T) {
	r := newRBACRig(t)
	base := "/api/v1/courses/" + r.course.String()
	for _, p := range []string{"/gradebook/finalize", "/gradebook/unlock"} {
		for _, who := range []string{"TA", "SV", "ADMIN", "GV khác"} {
			code, _ := r.do(http.MethodPost, base+p, r.tok[who], "k-"+who+p, map[string]any{"reason": "Phúc khảo sinh viên A"})
			require.Equal(t, http.StatusForbidden, code, who+p)
		}
		code, _ := r.do(http.MethodPost, base+p, "", "k", map[string]any{})
		require.Equal(t, http.StatusUnauthorized, code)
	}
	code, _ := r.do(http.MethodPost, "/api/v1/courses/khong-phai-uuid/gradebook/finalize", r.tok["GV"], "k-x", map[string]any{})
	require.Equal(t, http.StatusNotFound, code)
}

// TestFinalizeRBAC: giảng viên của lớp chốt được; Staff đọc bản chốt, sinh viên không.
func TestFinalizeRBAC(t *testing.T) {
	r := newRBACRig(t)
	base := "/api/v1/courses/" + r.course.String()
	r.exec(`insert into grade_entries (course_id, item_id, student_id, score, entered_by)
		select e.course_id, i.id, e.user_id, 7, (select created_by from courses where id = $1) from enrollments e join grade_items i on i.course_id = e.course_id
		where e.course_id = $1 and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'`, r.course)
	code, _ := r.do(http.MethodPost, base+"/gradebook/finalize", r.tok["GV"], "key-fin-0001", map[string]any{})
	require.Equal(t, http.StatusOK, code)
	for _, who := range []string{"GV", "TA"} {
		for _, p := range []string{"/gradebook/snapshots", "/gradebook/snapshots/1"} {
			code, _ := r.do(http.MethodGet, base+p, r.tok[who], "", nil)
			require.Equal(t, http.StatusOK, code, who+p)
		}
	}
	for _, who := range []string{"SV", "ADMIN", "GV khác"} {
		code, _ := r.do(http.MethodGet, base+"/gradebook/snapshots/1", r.tok[who], "", nil)
		require.Equal(t, http.StatusForbidden, code, who)
	}
	code, _ = r.do(http.MethodGet, base+"/gradebook/snapshots/9", r.tok["GV"], "", nil)
	require.Equal(t, http.StatusNotFound, code)
}

// ---- AC5 -------------------------------------------------------------------------------------------------------------------------------------------

func TestRefinalizeNewVersion(t *testing.T) {
	r := newFinRig(t, 4)
	_, err := r.finalize()
	require.NoError(t, err)
	require.NoError(t, r.unlock("Phúc khảo sv02: sửa điểm cuối kỳ"))
	r.exec(`update grade_entries set score = 10, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[1])
	out, err := r.finalize()
	require.NoError(t, err)
	require.Equal(t, 2, out.Version)
	require.Equal(t, 4, r.snapCount(2))
	st, v := r.status()
	require.Equal(t, "LOCKED", st)
	require.Equal(t, 2, v)
	require.Equal(t, 4, r.countRows(`select count(*) from final_grade_snapshots where course_id = $1 and version = 2 and unlock_reason = 'Phúc khảo sv02: sửa điểm cuối kỳ'`, r.course))
	require.Zero(t, r.countRows(`select count(*) from final_grade_snapshots where course_id = $1 and version = 1 and unlock_reason is not null`, r.course))
	require.Zero(t, r.countRows(`select count(*) from gradebooks where course_id = $1 and unlock_reason is not null`, r.course), "chốt lại xoá thông tin mở khoá")
}

func TestOldSnapshotHashUnchanged(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	h1 := r.snapHash(1)
	require.NoError(t, r.unlock("Phúc khảo sv01: sửa điểm giữa kỳ"))
	r.exec(`update grade_entries set score = 3, version = version + 1 where item_id = $1 and student_id = $2`, r.items["weekly"], r.students[0])
	_, err = r.finalize()
	require.NoError(t, err)
	require.Equal(t, h1, r.snapHash(1))
	for _, sql := range []string{
		`update final_grade_snapshots set letter = 'A' where version = 1`,
		`delete from final_grade_snapshots where version = 1`,
		`truncate final_grade_snapshots`,
	} {
		_, err := r.pool.Exec(r.ctx, sql)
		require.Error(t, err, sql)
		require.Contains(t, err.Error(), "snapshot is immutable", sql)
	}
	require.Equal(t, h1, r.snapHash(1))
	// ba lần chốt – mở – chốt: version 1, 2, 3 không nhảy / không trùng
	require.NoError(t, r.unlock("Sửa lần hai để thử phiên bản ba"))
	out, err := r.finalize()
	require.NoError(t, err)
	require.Equal(t, 3, out.Version)
	require.Equal(t, h1, r.snapHash(1))
}

func TestSnapshotHistoryReadable(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	require.NoError(t, r.unlock("Phúc khảo sv01: sửa điểm giữa kỳ"))
	r.exec(`update grade_entries set score = 10, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[0])
	_, err = r.finalize()
	require.NoError(t, err)
	vs, err := r.svc().SnapshotVersions(r.ctx, r.course)
	require.NoError(t, err)
	require.Len(t, vs, 2)
	require.Equal(t, 2, vs[0].Version)
	require.NotNil(t, vs[0].UnlockReason)
	require.Nil(t, vs[1].UnlockReason)
	require.Equal(t, 3, vs[1].Students)
	old, err := r.svc().Snapshot(r.ctx, r.course, 1, "", 2)
	require.NoError(t, err)
	require.Len(t, old.Items, 2)
	require.NotNil(t, old.NextCursor)
	next, err := r.svc().Snapshot(r.ctx, r.course, 1, *old.NextCursor, 2)
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	_, err = r.svc().Snapshot(r.ctx, r.course, 7, "", 10)
	st, _ := apiCode(t, err)
	require.Equal(t, 404, st)
	// bản 1 giữ điểm cũ của sv01; bản 2 điểm mới
	var g1, g2 string
	r.one(&g1, `select numeric_grade::text from final_grade_snapshots where course_id = $1 and version = 1 and student_id = $2`, r.course, r.students[0])
	r.one(&g2, `select numeric_grade::text from final_grade_snapshots where course_id = $1 and version = 2 and student_id = $2`, r.course, r.students[0])
	require.NotEqual(t, g1, g2)
}

// TC-44: sinh viên rời lớp trước lần chốt lại → không có bản v2 của họ; bản v1 còn nguyên.
func TestRefinalizeSkipsLeftStudent(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	require.NoError(t, r.unlock("Sinh viên 2 rời lớp trước khi chốt lại"))
	r.exec(`update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and user_id = $2`, r.course, r.students[1])
	out, err := r.finalize()
	require.NoError(t, err)
	require.Equal(t, 2, out.Students)
	require.Zero(t, r.countRows(`select count(*) from final_grade_snapshots where course_id = $1 and version = 2 and student_id = $2`, r.course, r.students[1]))
	require.Equal(t, 1, r.countRows(`select count(*) from final_grade_snapshots where course_id = $1 and version = 1 and student_id = $2`, r.course, r.students[1]))
}

// ---- AC6: thông báo -------------------------------------------------------------------------------------------------------------------------------

func (r *finRig) handle(version int) {
	svc := &grade.Service{Pool: r.pool, Notify: notify.New(r.pool, nil, nil)}
	payload, _ := json.Marshal(map[string]any{"course_id": r.course.String(), "version": version})
	require.NoError(r.t, svc.HandleFinalized(r.ctx, outbox.Message{Topic: grade.TopicFinalized, Payload: payload}))
}

func (r *finRig) notes(typ string) int {
	return r.countRows(`select count(*) from notifications where course_id = $1 and type = $2`, r.course, typ)
}

func TestFinalizeNotifiesAll(t *testing.T) {
	r := newFinRig(t, 5)
	_, err := r.finalize()
	require.NoError(t, err)
	r.handle(1)
	require.Equal(t, 5, r.notes("GRADE_FINALIZED"))
	var title string
	r.one(&title, `select title from notifications where course_id = $1 and type = 'GRADE_FINALIZED' limit 1`, r.course)
	require.Equal(t, "Điểm học phần lớp "+r.classCode()+" đã được chốt.", title)
	r.handle(1) // khử trùng
	require.Equal(t, 5, r.notes("GRADE_FINALIZED"))
	require.Zero(t, r.notes("GRADE_CHANGED"))
}

func (r *finRig) classCode() (c string) {
	r.one(&c, `select class_code from courses where id = $1`, r.course)
	return
}

func TestRefinalizeNotifiesOnlyChanged(t *testing.T) {
	r := newFinRig(t, 4)
	_, err := r.finalize()
	require.NoError(t, err)
	r.handle(1)
	require.NoError(t, r.unlock("Phúc khảo: đổi điểm một sinh viên"))
	// sv0: điểm đổi; sv1: điểm thành phần đổi nhưng HP + chữ không đổi (final 7 → 7,2 vẫn làm tròn như cũ)
	r.exec(`update grade_entries set score = 10, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[0])
	r.exec(`update grade_entries set score = 7.2, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[1])
	_, err = r.finalize()
	require.NoError(t, err)
	r.handle(2)
	require.Equal(t, 1, r.notes("GRADE_CHANGED"))
	var u uuid.UUID
	r.one(&u, `select user_id from notifications where course_id = $1 and type = 'GRADE_CHANGED'`, r.course)
	require.Equal(t, r.students[0], u)
	r.handle(2)
	require.Equal(t, 1, r.notes("GRADE_CHANGED"))
}

func TestNotifyNoNumbers(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	r.handle(1)
	require.NoError(t, r.unlock("Phúc khảo: đổi điểm một sinh viên"))
	r.exec(`update grade_entries set score = 10, version = version + 1 where item_id = $1 and student_id = $2`, r.items["final"], r.students[0])
	_, err = r.finalize()
	require.NoError(t, err)
	r.handle(2)
	rows, err := r.pool.Query(r.ctx, `select title, coalesce(body, ''), coalesce(link, '') from notifications where course_id = $1 and type in ('GRADE_FINALIZED', 'GRADE_CHANGED')`, r.course)
	require.NoError(t, err)
	defer rows.Close()
	n := 0
	for rows.Next() {
		var title, body, link string
		require.NoError(t, rows.Scan(&title, &body, &link))
		n++
		for _, txt := range []string{title, body} {
			stripped := strings.ReplaceAll(txt, r.classCode(), "")
			require.NotRegexp(t, `\d`, stripped, "thông báo không có con số")
		}
	}
	require.Positive(t, n)
}

func TestNotifyDedupe(t *testing.T) {
	r := newFinRig(t, 3)
	_, err := r.finalize()
	require.NoError(t, err)
	r.handle(1)
	r.handle(1)
	r.handle(1)
	require.Equal(t, 3, r.notes("GRADE_FINALIZED"))
}

// ---- AC7: đua --------------------------------------------------------------------------------------------------------------------------------------

func TestFinalizeConcurrent(t *testing.T) {
	r := newFinRig(t, 5)
	var wg sync.WaitGroup
	results := make([]error, 6)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, results[i] = r.finalize()
		}()
	}
	wg.Wait()
	ok := 0
	for _, err := range results {
		if err == nil {
			ok++
			continue
		}
		_, code := apiCode(t, err)
		require.Equal(t, "GRADEBOOK_LOCKED", code)
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, r.countRows(`select count(distinct version) from final_grade_snapshots where course_id = $1`, r.course))
	require.Equal(t, 5, r.snapCount(1))
}

// TestFinalizeVsPutRace: PUT chạy xong trước chốt thì có trong snapshot; sau chốt thì 409; không bao giờ "nửa mới nửa cũ".
func TestFinalizeVsPutRace(t *testing.T) {
	for range 20 {
		r := newFinRig(t, 3)
		var wg sync.WaitGroup
		var putErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := r.finalize()
			require.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			_, putErr = put(r.tdb, r.gv, r.course, r.students[0], r.items["final"], s("10"), n(1))
		}()
		wg.Wait()
		var live, snap string
		r.one(&live, `select score::text from grade_entries where item_id = $1 and student_id = $2`, r.items["final"], r.students[0])
		var bd string
		r.one(&bd, `select breakdown::text from final_grade_snapshots where course_id = $1 and student_id = $2 and version = 1`, r.course, r.students[0])
		var b grade.SnapBreakdown
		require.NoError(t, json.Unmarshal([]byte(bd), &b))
		snap = b.Items["final"]
		if putErr == nil {
			require.Equal(t, "10", snap, "PUT thành công trước chốt phải có trong snapshot")
			require.Equal(t, "10.00", live)
		} else {
			_, code := apiCode(t, putErr)
			require.Equal(t, "GRADEBOOK_LOCKED", code)
			require.Equal(t, "6", snap)
			require.Equal(t, "6.00", live)
		}
	}
}

// TestWriteTxLocksGradebookShared: mọi giao dịch ghi mở đầu bằng `FOR SHARE` trên hàng `gradebooks` (nguồn: mã Go).
func TestWriteTxLocksGradebookShared(t *testing.T) {
	for _, f := range []string{"gradebook.go", "import.go", "confirm.go"} {
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		require.True(t, strings.Contains(string(src), "GradebookStatusShare") || strings.Contains(string(src), "FOR SHARE") || strings.Contains(string(src), "for share"), f)
	}
}

// ---- AC8: hiệu năng, không N+1 ---------------------------------------------------------------------------------------------------------------------

func TestFinalize1000Under3s(t *testing.T) {
	testutil.SkipTiming(t)
	r := newFinRig(t, 0)
	r.bulk(r.course, 1000)
	r.exec(`insert into grade_entries (course_id, item_id, student_id, score, entered_by)
		select e.course_id, i.id, e.user_id, 7, $2 from enrollments e join grade_items i on i.course_id = e.course_id where e.course_id = $1 and e.role_in_course = 'STUDENT'`, r.course, r.gv)
	t0 := time.Now()
	out, err := r.finalize()
	require.NoError(t, err)
	require.Equal(t, 1000, out.Students)
	require.Less(t, time.Since(t0), 3*time.Second)
	t0 = time.Now()
	page, err := r.svc().Snapshot(r.ctx, r.course, 1, "", 100)
	require.NoError(t, err)
	require.Len(t, page.Items, 100)
	require.Less(t, time.Since(t0), 500*time.Millisecond)
	// TLR-15: đọc bản chốt dùng chỉ mục (không Seq Scan trên final_grade_snapshots)
	rows, err := r.pool.Query(r.ctx, `explain select student_id from final_grade_snapshots where course_id = $1 and version = 1 order by student_id limit 100`, r.course)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		require.NotContains(t, line, "Seq Scan on final_grade_snapshots")
	}
}

func TestFinalizeNoNPlusOne(t *testing.T) {
	d, c := countingDB(t)
	mk := func(n int) *finRig {
		gv := d.user("TEACHER", "GV")
		course := d.course(gv, "L1")
		doc := d.newDraft(course, gv, 1, nil)
		d.confirmDoc(course, gv, doc)
		d.bulk(course, n)
		d.exec(`insert into grade_entries (course_id, item_id, student_id, score, entered_by)
			select e.course_id, i.id, e.user_id, 7, $2 from enrollments e join grade_items i on i.course_id = e.course_id where e.course_id = $1 and e.role_in_course = 'STUDENT'`, course, gv)
		return &finRig{tdb: d, course: course, gv: gv}
	}
	count := func(n int) int64 {
		r := mk(n)
		before := c.n.Load()
		_, err := r.finalize()
		require.NoError(t, err)
		return c.n.Load() - before
	}
	small, big := count(30), count(400)
	require.Equal(t, small, big, "cùng số câu SQL cho 30 và 400 sinh viên")
}
