package grade_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

const goodRules = `[{"kind":"BONUS","applicable":false},{"kind":"PENALTY","applicable":false},{"kind":"ROUNDING","params":{"step":"0.5","mode":"HALF_UP"}},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"5","letter":"A"},{"min":"0","letter":"F"}]}}]`

func rulesOf(t *testing.T, raw string) []grade.PutRule {
	t.Helper()
	var r []grade.PutRule
	require.NoError(t, json.Unmarshal([]byte(raw), &r))
	return r
}

// validInput: weekly 40 (QT) + final 60 (CK), đủ bốn luật, làm tròn 0,5.
func validInput(t *testing.T, version int) grade.PutInput {
	return grade.PutInput{Version: version, Rules: rulesOf(t, goodRules), Components: []grade.PutComponent{
		{Key: "weekly", Name: "Hằng tuần", Grp: "QT", Weight: "40"},
		{Key: "final", Name: "Cuối kỳ", Grp: "CK", Weight: "60"},
	}}
}

func (d *tdb) svc() *grade.Service { return &grade.Service{Pool: d.pool} }

// newDraft tạo bản nháp rỗng (như sau `extract`) rồi thay bằng `in`; trả bản nháp sau khi lưu.
func (d *tdb) newDraft(course, gv uuid.UUID, no int, mod func(*grade.PutInput)) grade.SchemeDoc {
	d.t.Helper()
	d.scheme(course, gv, no)
	in := validInput(d.t, 1)
	if mod != nil {
		mod(&in)
	}
	doc, err := d.svc().PutDraft(d.ctx, gv, course, in)
	require.NoError(d.t, err)
	return doc
}

func (d *tdb) confirmDoc(course, gv uuid.UUID, doc grade.SchemeDoc) grade.ConfirmResult {
	d.t.Helper()
	res, err := d.svc().ConfirmScheme(d.ctx, gv, course, doc.Version)
	require.NoError(d.t, err)
	return res
}

func apiCode(t *testing.T, err error) (int, string) {
	t.Helper()
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae), "không phải apierr: %v", err)
	return ae.Status, ae.Code
}

func (d *tdb) itemIDs(course uuid.UUID, scheme uuid.UUID) map[string]uuid.UUID {
	d.t.Helper()
	l, err := grade.LoadScheme(d.ctx, d.q, course, scheme)
	require.NoError(d.t, err)
	out := map[string]uuid.UUID{}
	for _, c := range l.Scheme.Components {
		for _, it := range c.Items {
			out[it.Key] = it.ID
		}
	}
	return out
}

func (d *tdb) countRows(sql string, args ...any) (n int) {
	d.t.Helper()
	d.one(&n, sql, args...)
	return n
}

// AC3: PUT thay toàn bộ, version + 1, bộ kiểm chạy lại và `unclear_count` ghi trong cùng giao dịch (#16).
func TestSchemePutRecheck(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.scheme(c, gv, 1)
	in := validInput(t, 1)
	in.Rules = rulesOf(t, `[{"kind":"BONUS","applicable":false}]`)
	doc, err := d.svc().PutDraft(d.ctx, gv, c, in)
	require.NoError(t, err)
	require.Equal(t, 2, doc.Version)
	var codes []string
	for _, u := range doc.UnclearItems {
		codes = append(codes, u.Code)
	}
	require.ElementsMatch(t, []string{"ABSENCE_RULE_MISSING", "ROUNDING_MISSING", "LETTER_SCALE_MISSING"}, codes)
	require.Equal(t, len(doc.UnclearItems), d.status(c).UnclearCount, "cột unclear_count khớp bộ kiểm vừa chạy")

	doc, err = d.svc().PutDraft(d.ctx, gv, c, validInput(t, 2))
	require.NoError(t, err)
	require.Equal(t, 3, doc.Version)
	require.Empty(t, doc.UnclearItems)
	require.Zero(t, d.status(c).UnclearCount)
	require.Equal(t, grade.StateDraftReady, d.status(c).State)
}

func TestSchemePutVersionConflict(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.scheme(c, gv, 1)
	_, err := d.svc().PutDraft(d.ctx, gv, c, validInput(t, 1))
	require.NoError(t, err)
	_, err = d.svc().PutDraft(d.ctx, gv, c, validInput(t, 1)) // version cũ
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.VersionConflict, code)
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	cur := ae.Details.(map[string]any)["current"].(grade.SchemeDoc)
	require.Equal(t, 2, cur.Version, "kèm bản hiện tại để hoàn lại")
}

func TestSchemePutOnlyDraft(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	_, err := d.svc().PutDraft(d.ctx, gv, c, validInput(t, 1))
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.NoDraft, code)

	doc := d.newDraft(c, gv, 1, nil)
	d.confirmDoc(c, gv, doc)
	_, err = d.svc().PutDraft(d.ctx, gv, c, validInput(t, 3))
	st, code = apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.SchemeNotDraft, code)
}

// Đầu vào sai hình dạng → 422 kèm lỗi theo ô; không đổi gì.
func TestSchemePutValidation(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.scheme(c, gv, 1)
	for name, mod := range map[string]func(*grade.PutInput){
		"weight chữ":  func(in *grade.PutInput) { in.Components[0].Weight = "abc" },
		"weight âm":   func(in *grade.PutInput) { in.Components[0].Weight = "-5" },
		"weight 101":  func(in *grade.PutInput) { in.Components[0].Weight = "101" },
		"weight 3 số": func(in *grade.PutInput) { in.Components[0].Weight = "12.345" },
		"key trùng":   func(in *grade.PutInput) { in.Components[1].Key = "weekly" },
		"nhóm sai":    func(in *grade.PutInput) { in.Components[0].Grp = "XX" },
		"luật lạ":     func(in *grade.PutInput) { in.Rules = rulesOf(t, `[{"kind":"NOPE"}]`) },
		"exam lạ lớp": func(in *grade.PutInput) {
			in.Components[0].Items = []grade.PutItem{{Key: "a", Name: "a", MaxScore: "10", Source: "EXAM", ExamID: new(uuid.New())}}
		},
		"max_score 0": func(in *grade.PutInput) {
			in.Components[0].Items = []grade.PutItem{{Key: "a", Name: "a", MaxScore: "0", Source: "MANUAL"}}
		},
		"nguồn lạ": func(in *grade.PutInput) {
			in.Components[0].Items = []grade.PutItem{{Key: "a", Name: "a", MaxScore: "10", Source: "X"}}
		},
		"exam thiếu id": func(in *grade.PutInput) {
			in.Components[0].Items = []grade.PutItem{{Key: "a", Name: "a", MaxScore: "10", Source: "EXAM"}}
		},
	} {
		in := validInput(t, 1)
		mod(&in)
		_, err := d.svc().PutDraft(d.ctx, gv, c, in)
		st, code := apiCode(t, err)
		require.Equal(t, http.StatusUnprocessableEntity, st, name)
		require.Equal(t, apierr.ValidationFailed, code, name)
	}
	var v int
	d.one(&v, `select version from grade_schemes where course_id = $1`, c)
	require.Equal(t, 1, v, "lỗi đầu vào không tăng version")
	// Bài thi nháp chưa lên lịch không gắn được.
	ex := d.exam(c, gv, "DRAFT", "10")
	in := validInput(t, 1)
	in.Components[0].Items = []grade.PutItem{{Key: "a", Name: "a", MaxScore: "10", Source: "EXAM", ExamID: &ex}}
	_, err := d.svc().PutDraft(d.ctx, gv, c, in)
	st, _ := apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, st)
}

// AC3: mục không đổi giữ trích dẫn; mục giảng viên sửa / điền → filled_by_user = true và không còn trích dẫn.
func TestFilledByUserNoCitation(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.newDraft(c, gv, 1, nil)
	// Như bản trích từ quy chế: có trích dẫn, chưa do người điền.
	d.exec(`update grade_components set citations = '[{"page":1,"quote":"Cuối kỳ 60%"}]', filled_by_user = false where scheme_id = $1`, doc.ID)
	d.exec(`update grade_rules set source_quote = 'Làm tròn 0,5', source_page = 2, filled_by_user = false where scheme_id = $1`, doc.ID)

	in := validInput(t, doc.Version)
	in.Components[0].Weight = "35"
	in.Components[1].Weight = "65" // cả hai đổi
	in.Rules = rulesOf(t, `[{"kind":"BONUS","applicable":false},{"kind":"PENALTY","applicable":false},{"kind":"ROUNDING","params":{"step":"0.5","mode":"HALF_UP"}},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"6","letter":"A"},{"min":"0","letter":"F"}]}}]`)
	out, err := d.svc().PutDraft(d.ctx, gv, c, in)
	require.NoError(t, err)
	for _, comp := range out.Components {
		require.True(t, comp.FilledByUser, comp.Key)
		require.Empty(t, comp.Citations, comp.Key)
	}
	for _, r := range out.Rules {
		switch r.Kind {
		case "ROUNDING": // không đổi → giữ trích dẫn
			require.False(t, r.FilledByUser)
			require.Len(t, r.Citations, 1)
		case "LETTER_SCALE": // đổi
			require.True(t, r.FilledByUser)
			require.Empty(t, r.Citations)
		}
	}
	// Lưu lại đúng như cũ: mục đã do người đặt giữ nguyên cờ, mục không đổi giữ trích dẫn (không phát sinh thêm).
	same := validInput(t, out.Version)
	same.Components[0].Weight, same.Components[1].Weight = "35", "65"
	same.Rules = in.Rules
	again, err := d.svc().PutDraft(d.ctx, gv, c, same)
	require.NoError(t, err)
	require.True(t, again.Components[0].FilledByUser)
}

// AC4: ba sinh viên mẫu (cao nhất / giữa / thấp nhất) trước → sau.
func (d *tdb) gradedClass() (course, gv uuid.UUID, confirmed grade.SchemeDoc, students []uuid.UUID) {
	d.t.Helper()
	gv = d.user("TEACHER", "GV")
	course = d.course(gv, "L1")
	confirmed = d.newDraft(course, gv, 1, nil)
	d.confirmDoc(course, gv, confirmed)
	items := d.itemIDs(course, confirmed.ID)
	for i, sc := range [][2]string{{"10", "10"}, {"8", "6"}, {"5", "5"}} { // HP thô: 10 · 6,8 · 5
		st := d.student(course, []string{"An", "Binh", "Chi"}[i], "")
		students = append(students, st)
		d.entry(course, items["weekly"], st, gv, sc[0])
		d.entry(course, items["final"], st, gv, sc[1])
	}
	return course, gv, confirmed, students
}

func step01(t *testing.T) []grade.PutRule {
	return rulesOf(t, `[{"kind":"BONUS","applicable":false},{"kind":"PENALTY","applicable":false},{"kind":"ROUNDING","params":{"step":"0.1","mode":"HALF_UP"}},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"5","letter":"A"},{"min":"0","letter":"F"}]}}]`)
}

func TestImpactThreeSamples(t *testing.T) {
	d := newTDB(t)
	c, _, _, students := d.gradedClass()
	in := validInput(t, 1)
	imp, err := d.svc().Impact(d.ctx, c, in.Components, step01(t))
	require.NoError(t, err)
	require.Equal(t, 3, imp.StudentsTotal)
	require.Equal(t, 3, imp.StudentsComputable)
	require.Equal(t, 1, imp.ChangedVsConfirmed, "chỉ HP 6,8 đổi: 7,0 → 6,8")
	require.Len(t, imp.Samples, 3)
	hi, mid, lo := imp.Samples[0], imp.Samples[1], imp.Samples[2]
	require.Equal(t, students[0], hi.Student.ID)
	require.Equal(t, "10", hi.After.HP)
	require.Equal(t, "10", hi.Before.HP)
	require.Equal(t, students[1], mid.Student.ID)
	require.Equal(t, "6.8", mid.After.HP)
	require.Equal(t, "7", mid.Before.HP)
	require.Equal(t, students[2], lo.Student.ID)
	require.Equal(t, "5", lo.After.HP)
	require.Equal(t, 6, imp.EntriesCarried, "6 điểm nhập (3 sinh viên × 2 cột) cùng key / nguồn / thang")
	require.Empty(t, imp.EntriesDropped)
}

func TestImpactNoWrite(t *testing.T) {
	d := newTDB(t)
	c, _, _, _ := d.gradedClass()
	count := func() [4]int {
		return [4]int{d.countRows(`select count(*) from audit_log`), d.countRows(`select count(*) from grade_entries`), d.countRows(`select count(*) from grade_items`), d.countRows(`select coalesce(sum(version),0) from grade_schemes`)}
	}
	before := count()
	in := validInput(t, 1)
	for range 2 {
		_, err := d.svc().Impact(d.ctx, c, in.Components, step01(t))
		require.NoError(t, err)
	}
	require.Equal(t, before, count())
}

func TestImpactNoCompleteStudents(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.student(c, "An", "")
	doc := d.newDraft(c, gv, 1, nil)
	d.confirmDoc(c, gv, doc)
	in := validInput(t, 1)
	imp, err := d.svc().Impact(d.ctx, c, in.Components, in.Rules)
	require.NoError(t, err)
	require.Equal(t, 1, imp.StudentsTotal)
	require.Zero(t, imp.StudentsComputable)
	require.NotNil(t, imp.Samples)
	require.Empty(t, imp.Samples)
}

func TestConfirmFlow(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.newDraft(c, gv, 1, nil)
	res := d.confirmDoc(c, gv, doc)
	require.Equal(t, 1, res.SchemeNo)
	st := d.status(c)
	require.Equal(t, grade.StateConfirmed, st.State)
	require.False(t, st.PendingDraft)
	var by uuid.UUID
	d.one(&by, `select confirmed_by from grade_schemes where id = $1`, doc.ID)
	require.Equal(t, gv, by)
}

// AC5: bản CONFIRMED cũ thành SUPERSEDED trước, bản mới thành CONFIRMED (chỉ mục duy nhất một phần không DEFERRABLE).
func TestConfirmSupersedeBeforeConfirm(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	first := d.newDraft(c, gv, 1, nil)
	d.confirmDoc(c, gv, first)
	second := d.newDraft(c, gv, 2, func(in *grade.PutInput) { in.Components[0].Weight, in.Components[1].Weight = "30", "70" })
	res := d.confirmDoc(c, gv, second)
	require.Equal(t, 2, res.SchemeNo)
	var s1, s2 string
	d.one(&s1, `select status from grade_schemes where id = $1`, first.ID)
	d.one(&s2, `select status from grade_schemes where id = $1`, second.ID)
	require.Equal(t, "SUPERSEDED", s1)
	require.Equal(t, "CONFIRMED", s2)
}

func TestConfirmCreatesGradebookRow(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	require.Zero(t, d.countRows(`select count(*) from gradebooks where course_id = $1`, c))
	d.confirmDoc(c, gv, d.newDraft(c, gv, 1, nil))
	var status string
	d.one(&status, `select status from gradebooks where course_id = $1`, c)
	require.Equal(t, "OPEN", status)
}

func TestConfirmBlockedWhenLocked(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.newDraft(c, gv, 1, nil)
	d.exec(`insert into gradebooks (course_id, status, snapshot_version) values ($1, 'LOCKED', 1)`, c)
	_, err := d.svc().ConfirmScheme(d.ctx, gv, c, doc.Version)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.GradebookLocked, code)
	var s string
	d.one(&s, `select status from grade_schemes where id = $1`, doc.ID)
	require.Equal(t, "DRAFT", s)
}

func TestConfirmBlockedByUnclear(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.newDraft(c, gv, 1, func(in *grade.PutInput) {
		in.Rules = rulesOf(t, `[{"kind":"BONUS","applicable":false},{"kind":"PENALTY","applicable":false},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"0","letter":"F"}]}}]`)
	})
	_, err := d.svc().ConfirmScheme(d.ctx, gv, c, doc.Version)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, st)
	require.Equal(t, apierr.SchemeHasUnclear, code)
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, []string{"ROUNDING_MISSING"}, ae.Details.(map[string]any)["codes"])
	var s string
	d.one(&s, `select status from grade_schemes where id = $1`, doc.ID)
	require.Equal(t, "DRAFT", s)
}

func TestConfirmStaleVersion(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	doc := d.newDraft(c, gv, 1, nil)
	_, err := d.svc().PutDraft(d.ctx, gv, c, validInput(t, doc.Version)) // người khác vừa sửa
	require.NoError(t, err)
	_, err = d.svc().ConfirmScheme(d.ctx, gv, c, doc.Version)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, apierr.VersionConflict, code)
	var s string
	d.one(&s, `select status from grade_schemes where id = $1`, doc.ID)
	require.Equal(t, "DRAFT", s, "không xác nhận bản đã đổi")
}

// AC8: hai người cùng sửa một version → đúng một thành công, người kia 409.
func TestSchemeEditRace(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.scheme(c, gv, 1)
	var wg sync.WaitGroup
	errs := make([]error, 6)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = d.svc().PutDraft(context.Background(), gv, c, validInput(t, 1))
		}()
	}
	wg.Wait()
	ok, conflict := 0, 0
	for _, e := range errs {
		if e == nil {
			ok++
		} else if _, code := apiCode(t, e); code == apierr.VersionConflict {
			conflict++
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 5, conflict)
}

// AC6: điểm nhập tay cùng key + nguồn + max_score chuyển sang item mới trong cùng giao dịch.
func TestConfirmCarryOverEntries(t *testing.T) {
	d := newTDB(t)
	c, gv, first, students := d.gradedClass()
	oldItems := d.itemIDs(c, first.ID)
	second := d.newDraft(c, gv, 2, func(in *grade.PutInput) { in.Rules = step01(t) })
	res := d.confirmDoc(c, gv, second)
	require.Equal(t, 6, res.Impact.EntriesCarried)
	newItems := d.itemIDs(c, second.ID)
	require.Equal(t, 6, d.countRows(`select count(*) from grade_entries where item_id = any($1::text[]::uuid[])`, []string{newItems["weekly"].String(), newItems["final"].String()}))
	require.Zero(t, d.countRows(`select count(*) from grade_entries where item_id = any($1::text[]::uuid[])`, []string{oldItems["weekly"].String(), oldItems["final"].String()}))
	// công thức mới tính đúng từ điểm đã mang
	l, ok, err := grade.LoadConfirmed(d.ctx, d.q, c)
	require.NoError(t, err)
	require.True(t, ok)
	in, err := grade.LoadInputs(d.ctx, d.q, c, l.ID, students)
	require.NoError(t, err)
	require.Equal(t, "6.8", grade.Compute(l.Scheme, in[students[1]]).HP.String())
}

func TestConfirmNoCarryOnMaxScoreChange(t *testing.T) {
	d := newTDB(t)
	c, gv, first, _ := d.gradedClass()
	oldItems := d.itemIDs(c, first.ID)
	second := d.newDraft(c, gv, 2, func(in *grade.PutInput) {
		in.Components[1].Items = []grade.PutItem{{Key: "final", Name: "Cuối kỳ", MaxScore: "100", Source: "MANUAL"}}
	})
	imp, err := d.svc().Impact(d.ctx, c, componentsOf(second), rulesFromDoc(t, second))
	require.NoError(t, err)
	require.Equal(t, []grade.Dropped{{ItemKey: "final", Count: 3, Reason: grade.DropMaxChanged}}, imp.EntriesDropped)
	res := d.confirmDoc(c, gv, second)
	require.Equal(t, 3, res.Impact.EntriesCarried)
	require.Equal(t, 3, d.countRows(`select count(*) from grade_entries where item_id = $1`, oldItems["final"]), "điểm không mang vẫn còn trong DB")
	require.Zero(t, d.countRows(`select count(*) from grade_entries where item_id = $1`, d.itemIDs(c, second.ID)["final"]))
}

func TestConfirmNoCarryOnSourceChange(t *testing.T) {
	d := newTDB(t)
	c, gv, first, _ := d.gradedClass()
	oldItems := d.itemIDs(c, first.ID)
	ex := d.exam(c, gv, "PUBLISHED", "10")
	second := d.newDraft(c, gv, 2, func(in *grade.PutInput) {
		in.Components[0].Items = []grade.PutItem{{Key: "weekly", Name: "Hằng tuần", MaxScore: "10", Source: "EXAM", ExamID: &ex}}
	})
	res := d.confirmDoc(c, gv, second)
	require.Equal(t, 3, res.Impact.EntriesCarried, "chỉ cột cuối kỳ được mang")
	require.Contains(t, res.Impact.EntriesDropped, grade.Dropped{ItemKey: "weekly", Count: 3, Reason: grade.DropSourceChanged})
	require.Equal(t, 3, d.countRows(`select count(*) from grade_entries where item_id = $1`, oldItems["weekly"]))
}

func componentsOf(doc grade.SchemeDoc) []grade.PutComponent {
	var out []grade.PutComponent
	for _, c := range doc.Components {
		pc := grade.PutComponent{Key: c.Key, Name: c.Name, Grp: c.Grp, Weight: c.Weight}
		for _, it := range c.Items {
			pc.Items = append(pc.Items, grade.PutItem(it))
		}
		out = append(out, pc)
	}
	return out
}

func rulesFromDoc(t *testing.T, doc grade.SchemeDoc) []grade.PutRule {
	t.Helper()
	var out []grade.PutRule
	for _, r := range doc.Rules {
		out = append(out, grade.PutRule{Kind: r.Kind, Applicable: new(r.Applicable), Params: r.Params})
	}
	return out
}

// AC5: audit_log kèm số liệu hậu quả; outbox grade.scheme_confirmed.
func TestConfirmAudit(t *testing.T) {
	d := newTDB(t)
	c, gv, _, _ := d.gradedClass()
	second := d.newDraft(c, gv, 2, func(in *grade.PutInput) { in.Rules = step01(t) })
	d.confirmDoc(c, gv, second)
	var after []byte
	var actor uuid.UUID
	d.one(&after, `select after from audit_log where entity = 'grade_schemes' and action = 'confirm' and entity_id = $1`, second.ID.String())
	d.one(&actor, `select actor_id from audit_log where entity = 'grade_schemes' and action = 'confirm' and entity_id = $1`, second.ID.String())
	require.Equal(t, gv, actor)
	var m map[string]any
	require.NoError(t, json.Unmarshal(after, &m))
	require.EqualValues(t, 3, m["students_total"])
	require.EqualValues(t, 1, m["changed_vs_confirmed"])
	require.EqualValues(t, 6, m["entries_carried"])
	require.Equal(t, 2, d.countRows(`select count(*) from outbox where topic = $1 and payload ->> 'course_id' = $2`, grade.TopicSchemeConfirmed, c.String()), "hai lần xác nhận → hai sự kiện")
}

// AC7: bản nháp mới cạnh bản đã xác nhận không đổi bản đang dùng; so sánh do code tính.
func TestNewPolicyMakesNewDraft(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	first := d.newDraft(c, gv, 1, nil)
	d.confirmDoc(c, gv, first)
	d.newDraft(c, gv, 2, func(in *grade.PutInput) { in.Components[0].Weight, in.Components[1].Weight = "30", "70" })
	v, err := d.svc().Schemes(d.ctx, c, false)
	require.NoError(t, err)
	require.NotNil(t, v.Confirmed)
	require.NotNil(t, v.Draft)
	require.Equal(t, first.ID, v.Confirmed.ID)
	require.Equal(t, 2, v.Draft.SchemeNo)
	require.True(t, d.status(c).PendingDraft)
}

func TestConfirmedUnchangedUntilConfirm(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	first := d.newDraft(c, gv, 1, nil)
	d.confirmDoc(c, gv, first)
	before, err := d.svc().Schemes(d.ctx, c, false)
	require.NoError(t, err)
	second := d.newDraft(c, gv, 2, func(in *grade.PutInput) { in.Components[0].Weight, in.Components[1].Weight = "10", "90" })
	_, err = d.svc().PutDraft(d.ctx, gv, c, validInput(t, second.Version))
	require.NoError(t, err)
	after, err := d.svc().Schemes(d.ctx, c, false)
	require.NoError(t, err)
	require.Equal(t, before.Confirmed, after.Confirmed)
}

func TestDiffAgainstConfirmed(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.confirmDoc(c, gv, d.newDraft(c, gv, 1, nil))
	d.newDraft(c, gv, 2, func(in *grade.PutInput) {
		in.Components[0].Weight, in.Components[1].Weight = "30", "60"
		in.Components = append(in.Components, grade.PutComponent{Key: "lab", Name: "Thực hành", Grp: "QT", Weight: "10"})
		in.Rules = step01(t)
	})
	v, err := d.svc().Schemes(d.ctx, c, true)
	require.NoError(t, err)
	require.NotNil(t, v.Diff)
	require.Equal(t, []string{"lab"}, v.Diff.ComponentsAdded)
	require.Equal(t, []string{}, v.Diff.ComponentsRemoved)
	require.Equal(t, []grade.WeightChange{{Key: "weekly", From: "40", To: "30"}}, v.Diff.WeightChanged)
	require.Len(t, v.Diff.RulesChanged, 1)
	require.Equal(t, "ROUNDING", v.Diff.RulesChanged[0].Kind)
	require.Equal(t, "100", v.Diff.WeightSumFrom)
	require.Equal(t, "100", v.Diff.WeightSumTo)
	plain, err := d.svc().Schemes(d.ctx, c, false)
	require.NoError(t, err)
	require.Nil(t, plain.Diff)
}
