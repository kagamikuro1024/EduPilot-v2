package grade_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grade"
)

type meRig struct {
	*httpRig
	course, gv, a, b uuid.UUID
	items            map[string]uuid.UUID
	tok              map[string]string
}

// newMeRig: lớp weekly 40 (QT) + final 60 (CK); sinh viên A (weekly 8, final 6) và B (weekly 5, final 9,87 — canary); cả hai cột chưa công bố.
func newMeRig(t *testing.T) *meRig {
	t.Helper()
	h := newHTTPRig(t)
	r := &meRig{httpRig: h}
	r.gv = h.user("TEACHER", "GV")
	r.course = h.course(r.gv, "L1")
	doc := h.newDraft(r.course, r.gv, 1, nil)
	h.confirmDoc(r.course, r.gv, doc)
	r.items = h.itemIDs(r.course, doc.ID)
	r.a = h.student(r.course, "An", "20220001")
	r.b = h.student(r.course, "Bình", "20220002")
	h.entry(r.course, r.items["weekly"], r.a, r.gv, "8")
	h.entry(r.course, r.items["final"], r.a, r.gv, "6")
	h.entry(r.course, r.items["weekly"], r.b, r.gv, "5")
	h.entry(r.course, r.items["final"], r.b, r.gv, "9.87")
	ta := h.user("TA", "TA")
	h.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN')`, r.course, ta)
	r.tok = map[string]string{
		"A": h.token(r.a, auth.RoleStudent), "B": h.token(r.b, auth.RoleStudent), "GV": h.token(r.gv, auth.RoleTeacher), "TA": h.token(ta, auth.RoleTA),
		"ADMIN": h.token(h.user("ADMIN", "AD"), auth.RoleAdmin), "ngoài": h.token(h.user("STUDENT", "Ngoài"), auth.RoleStudent),
	}
	return r
}

func (r *meRig) publish(item string) {
	r.exec(`update grade_items set published_at = now() where id = $1`, r.items[item])
}

func (r *meRig) me(student uuid.UUID) grade.MeGrades {
	out, err := r.svc().MeGrades(r.ctx, r.course, student)
	require.NoError(r.t, err)
	return out
}

func (r *meRig) get(path, who string) (int, string, http.Header) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+r.course.String()+path, nil)
	if tok := r.tok[who]; tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, req)
	return w.Code, w.Body.String(), w.Header()
}

func (r *meRig) post(path, who string, body any) (int, string) {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+r.course.String()+path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	if tok := r.tok[who]; tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func comp(t *testing.T, m grade.MeGrades, key string) grade.MeComponent {
	t.Helper()
	for _, c := range m.Components {
		if c.Key == key {
			return c
		}
	}
	t.Fatalf("không có thành phần %s", key)
	return grade.MeComponent{}
}

func TestMeGradesNoSchemeStatus(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L2")
	st := d.student(c, "An", "20220001")
	d.newDraft(c, gv, 1, nil) // bản nháp không bao giờ lộ cho sinh viên
	out, err := d.svc().MeGrades(d.ctx, c, st)
	require.NoError(t, err)
	raw, _ := json.Marshal(out)
	require.JSONEq(t, `{"status":"NO_SCHEME"}`, string(raw))
}

func TestMeGradesOwnOnly(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	a, b := r.me(r.a), r.me(r.b)
	require.Equal(t, "IN_PROGRESS", a.Status)
	require.Equal(t, "8", comp(t, a, "weekly").Score)
	require.Equal(t, "5", comp(t, b, "weekly").Score)
	require.Equal(t, "8", a.QT)
	require.Equal(t, "5", b.QT)
	require.Equal(t, grade.OfficialNote, a.OfficialNote)
}

// TestMeGradesIgnoresStudentParam: id của người khác trong query / thân bị bỏ qua — vẫn trả của mình.
func TestMeGradesIgnoresStudentParam(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	_, own, _ := r.get("/me/grades", "A")
	code, body, _ := r.get("/me/grades?student_id="+r.b.String()+"&user_id="+r.b.String()+"&student_code=20220002", "A")
	require.Equal(t, 200, code)
	require.JSONEq(t, own, body)
	require.Contains(t, body, `"score":"8"`)
	require.NotContains(t, body, `"score":"5"`)
}

func TestMeGradesNoStore(t *testing.T) {
	r := newMeRig(t)
	code, _, h := r.get("/me/grades", "A")
	require.Equal(t, 200, code)
	require.Equal(t, "no-store", h.Get("Cache-Control"))
}

// TestPendingItemHasNoScore: cột chưa công bố → PENDING, khoá `score` không tồn tại, không `qt`.
func TestPendingItemHasNoScore(t *testing.T) {
	r := newMeRig(t)
	m := r.me(r.a)
	for _, c := range m.Components {
		require.Equal(t, "PENDING", c.Status)
		require.Empty(t, c.Score)
	}
	require.Empty(t, m.QT)
	raw, _ := json.Marshal(m)
	var generic struct {
		Components []map[string]any `json:"components"`
	}
	require.NoError(t, json.Unmarshal(raw, &generic))
	for _, c := range generic.Components {
		require.NotContains(t, c, "score")
	}
	require.Nil(t, m.Final)
	r.publish("weekly")
	require.Equal(t, "PENDING", comp(t, r.me(r.a), "final").Status)
	require.Empty(t, r.me(r.a).Final)
}

// TestNoDraftLeak: canary đặt vào điểm chưa công bố (9,87), lý do mở khoá, mã nội bộ; không có trong JSON của sinh viên ở bất kỳ trạng thái nào.
func TestNoDraftLeak(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	check := func(m grade.MeGrades) {
		raw, _ := json.Marshal(m)
		for _, bad := range []string{"9.87", "9,87", "CANARY-UNLOCK", "hp_raw", "unlock_reason", "unclear", "drift", "updated_by", "breakdown", "LIET:"} {
			require.NotContains(t, string(raw), bad)
		}
	}
	check(r.me(r.a))
	check(r.me(r.b))
	// chốt rồi mở khoá với lý do canary rồi chốt lại: bản của sinh viên vẫn không lộ lý do
	r.publish("final")
	_, err := r.svc().Finalize(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	_, err = r.svc().Unlock(r.ctx, r.gv, r.course, "CANARY-UNLOCK phúc khảo lần một")
	require.NoError(t, err)
	_, err = r.svc().Finalize(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	check(r.me(r.a))
}

var meAllowed = map[string]bool{
	"status": true, "scheme": true, "scheme_no": true, "confirmed_at": true, "components": true, "key": true, "name": true, "grp": true, "weight": true,
	"bonus": true, "applied": true, "raw": true, "cap": true, "penalty": true, "value": true, "absent": true, "free": true, "per": true,
	"qt": true, "final": true, "numeric_grade": true, "letter": true, "failed_reason": true, "version": true, "locked_at": true,
	"calculation": true, "label": true, "expr": true, "source_quote": true, "page": true, "official_note": true, "score": true,
}

func walkKeys(t *testing.T, v any, path string) {
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			require.True(t, meAllowed[k], "khoá ngoài allowlist: %s.%s", path, k)
			walkKeys(t, c, path+"."+k)
		}
	case []any:
		for _, c := range x {
			walkKeys(t, c, path+"[]")
		}
	}
}

// TestStudentDTOAllowlist: mọi khoá JSON của DTO sinh viên (IN_PROGRESS và FINAL) nằm trong allowlist.
func TestStudentDTOAllowlist(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	r.publish("final")
	for range 2 {
		raw, _ := json.Marshal(r.me(r.a))
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		walkKeys(t, m, "$")
		_, err := r.svc().Finalize(r.ctx, r.gv, r.course)
		if err != nil {
			break
		}
	}
	m := r.me(r.a)
	require.Equal(t, "FINAL", m.Status)
	raw, _ := json.Marshal(m)
	var g map[string]any
	require.NoError(t, json.Unmarshal(raw, &g))
	walkKeys(t, g, "$")
}

func TestMeGradesFinalFromSnapshot(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	r.publish("final")
	_, err := r.svc().Finalize(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	m := r.me(r.a)
	require.Equal(t, "FINAL", m.Status)
	require.NotNil(t, m.Final)
	require.Equal(t, 1, m.Final.Version)
	hp := m.Final.NumericGrade
	// điểm sống đổi sau chốt không đổi `final`
	r.exec(`update grade_entries set score = 1 where item_id = $1 and student_id = $2`, r.items["final"], r.a)
	require.Equal(t, hp, r.me(r.a).Final.NumericGrade)
	require.NotEmpty(t, m.Calculation)
}

func TestMeGradesIsolation(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	_, a, _ := r.get("/me/grades", "A")
	_, b, _ := r.get("/me/grades", "B")
	require.Contains(t, a, `"score":"8"`)
	require.Contains(t, b, `"score":"5"`)
	for _, who := range []string{"ngoài", "TA", "GV", "ADMIN"} {
		code, _, _ := r.get("/me/grades", who)
		require.Equal(t, http.StatusForbidden, code, who)
		code, _ = r.post("/me/grades/what-if", who, map[string]any{"final_exam_score": "8"})
		require.Equal(t, http.StatusForbidden, code, who)
	}
	code, _, _ := r.get("/me/grades", "")
	require.Equal(t, http.StatusUnauthorized, code)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/khong-phai-uuid/me/grades", nil)
	req.Header.Set("Authorization", "Bearer "+r.tok["A"])
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestStudentCannotCallStaffGradeRoutes(t *testing.T) {
	r := newMeRig(t)
	for _, p := range []string{"/gradebook", "/gradebook/preview", "/gradebook/snapshots", "/gradebook/snapshots/1", "/gradebook/export.xlsx", "/grade-scheme", "/grade-scheme/status",
		"/gradebook/students/" + r.a.String() + "/explain", "/gradebook/students/" + r.a.String() + "/history"} {
		code, _, _ := r.get(p, "A")
		require.Equal(t, http.StatusForbidden, code, p)
	}
	for _, p := range []string{"/gradebook/finalize", "/gradebook/unlock", "/grade-entries/import", "/grade-scheme/confirm", "/grade-scheme/impact", "/grade-scheme/extract"} {
		code, _ := r.post(p, "A", map[string]any{})
		require.GreaterOrEqual(t, code, 400, p)
		require.Contains(t, []int{http.StatusForbidden, http.StatusUnauthorized}, code, p)
	}
}

// TestSnapshotIDOR: sinh viên không đọc được bản chốt (của mình hay của người khác) qua route Staff.
func TestSnapshotIDOR(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	r.publish("final")
	_, err := r.svc().Finalize(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	for _, who := range []string{"A", "B", "ngoài"} {
		code, body, _ := r.get("/gradebook/snapshots/1", who)
		require.Equal(t, http.StatusForbidden, code, who)
		require.NotContains(t, body, "20220001")
		require.NotContains(t, body, "numeric_grade")
	}
}

// ---- what-if -----------------------------------------------------------------------------------------------------------------------------------

func TestMeWhatIfNoWrite(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	counts := func() (n int) {
		for _, tbl := range []string{"grade_entries", "audit_log", "outbox", "final_grade_snapshots", "notifications", "jobs"} {
			var c int
			r.one(&c, "select count(*) from "+tbl)
			n = n*1000 + c
		}
		return n
	}
	before := counts()
	code, body := r.post("/me/grades/what-if", "A", map[string]any{"final_exam_score": "8,0"})
	require.Equal(t, 200, code, body)
	var out grade.WhatIfOut
	require.NoError(t, json.Unmarshal([]byte(body), &out))
	require.Equal(t, "8", out.HP, "QT 8, CK 8 → 8")
	require.NotEmpty(t, out.Letter)
	require.NotEmpty(t, out.Calculation)
	require.Equal(t, before, counts())
	require.NotContains(t, body, "9.87")
}

func TestMeWhatIfTargetLetter(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	code, body := r.post("/me/grades/what-if", "A", map[string]any{"target_letter": "A"})
	require.Equal(t, 200, code, body)
	var out grade.WhatIfOut
	require.NoError(t, json.Unmarshal([]byte(body), &out))
	require.NotNil(t, out.Reachable)
	if *out.Reachable {
		require.NotNil(t, out.MinFinalExam)
	}
	code, body = r.post("/me/grades/what-if", "A", map[string]any{"target_letter": "Z"})
	require.Equal(t, 200, code)
	require.Contains(t, body, `"reachable":false`)
	require.NotContains(t, body, "min_final_exam_score")
	code, _ = r.post("/me/grades/what-if", "A", map[string]any{"target_letter": "A", "final_exam_score": "8"})
	require.Equal(t, 422, code, "đúng một trong hai trường")
}

func TestMeWhatIfIncomplete(t *testing.T) {
	r := newMeRig(t) // chưa công bố gì → thiếu thành phần quá trình
	code, body := r.post("/me/grades/what-if", "A", map[string]any{"final_exam_score": "8"})
	require.Equal(t, 409, code)
	require.Contains(t, body, "NOT_ENOUGH_GRADES")
	r.publish("weekly")
	r.publish("final")
	_, err := r.svc().Finalize(r.ctx, r.gv, r.course)
	require.NoError(t, err)
	code, body = r.post("/me/grades/what-if", "A", map[string]any{"final_exam_score": "8"})
	require.Equal(t, 409, code)
	require.Contains(t, body, "FINAL_ALREADY_LOCKED")
}

func TestMeWhatIfOutOfRange(t *testing.T) {
	r := newMeRig(t)
	r.publish("weekly")
	for _, bad := range []string{"11", "-1", "abc", "8.333", ""} {
		code, _ := r.post("/me/grades/what-if", "A", map[string]any{"final_exam_score": bad})
		require.Equal(t, 422, code, bad)
	}
	for _, ok := range []string{"0", "10", "8,5", "7.25"} {
		code, _ := r.post("/me/grades/what-if", "A", map[string]any{"final_exam_score": ok})
		require.Equal(t, 200, code, ok)
	}
}

// TestRBACMatrixAll: ma trận vai × MỌI route của sổ điểm (SRS 7.5). Staff đọc: TEACHER + TA; ghi / chốt / nhập / xuất: chỉ TEACHER;
// route của sinh viên: chỉ STUDENT của lớp. Vai không được phép → 403 (chưa đăng nhập 401); vai được phép không bao giờ 401 / 403.
func TestRBACMatrixAll(t *testing.T) {
	r := newRBACRig(t)
	base := "/api/v1/courses/" + r.course.String()
	staff, teacher, student := []string{"GV", "TA"}, []string{"GV"}, []string{"SV"}
	uid := r.student.String()
	routes := []struct {
		name, method, path string
		body               any
		allowed            []string
	}{
		{"gradebook", "GET", "/gradebook", nil, staff},
		{"preview", "GET", "/gradebook/preview", nil, staff},
		{"explain", "GET", "/gradebook/students/" + uid + "/explain", nil, staff},
		{"history", "GET", "/gradebook/students/" + uid + "/history", nil, staff},
		{"snapshots", "GET", "/gradebook/snapshots", nil, staff},
		{"snapshot", "GET", "/gradebook/snapshots/1", nil, staff},
		{"scheme", "GET", "/grade-scheme", nil, staff},
		{"status", "GET", "/grade-scheme/status", nil, staff},
		{"impact", "POST", "/grade-scheme/impact", map[string]any{"components": []any{}, "rules": []any{}}, staff},
		{"put entry", "PUT", "/grade-entries", map[string]any{"item_id": r.weekly, "student_id": r.student, "score": "7", "version": nil}, teacher},
		{"publish", "POST", "/grade-items/" + r.weekly.String() + "/publish", map[string]any{"published": true}, teacher},
		{"import", "POST", "/grade-entries/import", map[string]any{"upload_id": uuid.New(), "item_id": r.weekly, "mode": "preview", "mapping": map[string]any{"header_row": 1}}, teacher},
		{"export", "GET", "/gradebook/export.xlsx", nil, teacher},
		{"finalize", "POST", "/gradebook/finalize", map[string]any{}, teacher},
		{"unlock", "POST", "/gradebook/unlock", map[string]any{"reason": "Phúc khảo sinh viên A"}, teacher},
		{"put scheme", "PUT", "/grade-scheme", map[string]any{"version": 1, "components": []any{}, "rules": []any{}}, teacher},
		{"confirm", "POST", "/grade-scheme/confirm", map[string]any{"version": 1}, teacher},
		{"extract", "POST", "/grade-scheme/extract", map[string]any{"document_id": uuid.New()}, teacher},
		{"me grades", "GET", "/me/grades", nil, student},
		{"what-if", "POST", "/me/grades/what-if", map[string]any{"final_exam_score": "8"}, student},
	}
	require.Len(t, routes, 20)
	for _, rt := range routes {
		for who, tok := range r.tok {
			key := ""
			if rt.method == "POST" || rt.method == "PUT" {
				key = "key-rbac-" + rt.name + who
			}
			st, _ := r.do(rt.method, base+rt.path, tok, key, rt.body)
			allowed := false
			for _, a := range rt.allowed {
				allowed = allowed || a == who
			}
			switch {
			case who == "":
				require.Equal(t, http.StatusUnauthorized, st, rt.name)
			case allowed:
				require.NotContains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, st, "%s / %s", rt.name, who)
			default:
				require.Equal(t, http.StatusForbidden, st, "%s / %s", rt.name, who)
			}
		}
	}
}
