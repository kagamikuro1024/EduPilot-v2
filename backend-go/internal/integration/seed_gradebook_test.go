package integration_test

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/store"
)

// Lớp 1 của seed sprint 8 dựng trực tiếp trong CSDL từ đúng các tệp của BA (extraction-lop1.json, inputs-lop1.csv) — cùng dữ liệu `scripts/seed.mjs` đưa vào qua API;
// phần HTTP (xem trước, giải trình, phân quyền) đi qua router thật. Đường nhập XLSX có test riêng ở `internal/grade` (TestImport*).
const (
	gbFix  = "../../../docs/specs/FEAT-gradebook/fixtures/"
	gbSeed = "../../../seed/"
	canary = "7.77" // điểm cuối kỳ CHƯA công bố của B: không được lọt vào bất kỳ phản hồi nào dành cho sinh viên
)

func readCSV(t *testing.T, path string) []map[string]string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	recs, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	var out []map[string]string
	for _, r := range recs[1:] {
		m := map[string]string{}
		for i, h := range recs[0] {
			m[h] = r[i]
		}
		out = append(out, m)
	}
	return out
}

type gbWorld struct {
	r                    *rig
	c1, c2               uuid.UUID
	gv, ta, admin, dd    session
	sv                   map[string]session // account → phiên
	byCode               map[string]string  // MSSV → account
	userOf               map[string]uuid.UUID
	rows                 []map[string]string
	items                map[string]uuid.UUID
	midterm, final, exam uuid.UUID
}

func (r *rig) must(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

func newGBWorld(t *testing.T) *gbWorld {
	t.Helper()
	r := newRig(t)
	ctx := t.Context()
	w := &gbWorld{r: r, sv: map[string]session{}, byCode: map[string]string{}, userOf: map[string]uuid.UUID{}}
	gv := r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	ta := r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE)
	adm := r.addUser(uniq("admin"), store.UserRoleADMIN, store.UserStatusACTIVE)
	d := r.addUser(uniq("d"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	w.c1, w.c2 = r.course(gv.ID, "761987-"+uuid.NewString()[:4]), r.course(gv.ID, "761988-"+uuid.NewString()[:4])
	for _, c := range []uuid.UUID{w.c1, w.c2} {
		r.enroll(c, gv.ID, "TEACHER", "ACTIVE")
	}
	r.enroll(w.c1, ta.ID, "TA", "ACTIVE")
	w.gv, w.ta, w.admin, w.dd = r.mustLogin(gv.Email), r.mustLogin(ta.Email), r.mustLogin(adm.Email), r.mustLogin(d.Email)

	var exam uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into exams (course_id, title, created_by, status, max_score, opens_at, closes_at, duration_minutes, published_at)
		values ($1, 'Kiểm tra tuần 9 — Mật mã', $2, 'PUBLISHED', 10, now() - interval '3 hours', now() - interval '1 hour', 30, now()) returning id`, w.c1, gv.ID).Scan(&exam))
	w.exam = exam
	r.must(`insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 1, $2)`, w.c1, gv.ID)
	b, err := os.ReadFile(gbFix + "extraction-lop1.json")
	require.NoError(t, err)
	var ex struct {
		Components []struct {
			Key      string `json:"key"`
			Name     string `json:"name"`
			Group    string `json:"group"`
			Weight   string `json:"weight"`
			KindHint string `json:"kind_hint"`
		} `json:"components"`
		Rules []struct {
			Kind       string          `json:"kind"`
			Applicable bool            `json:"applicable"`
			Params     json.RawMessage `json:"params"`
		} `json:"rules"`
	}
	require.NoError(t, json.Unmarshal(b, &ex))
	in := grade.PutInput{Version: 1}
	for _, c := range ex.Components {
		pc := grade.PutComponent{Key: c.Key, Name: c.Name, Grp: c.Group, Weight: c.Weight}
		if c.KindHint == "EXAM" {
			pc.Items = []grade.PutItem{{Key: c.Key, Name: c.Name, MaxScore: "10", Source: "EXAM", ExamID: &exam}}
		}
		in.Components = append(in.Components, pc)
	}
	for _, ru := range ex.Rules {
		ap := ru.Applicable
		in.Rules = append(in.Rules, grade.PutRule{Kind: ru.Kind, Applicable: &ap, Params: ru.Params})
	}
	svc := &grade.Service{Pool: r.pool}
	doc, err := svc.PutDraft(ctx, gv.ID, w.c1, in)
	require.NoError(t, err)
	_, err = svc.ConfirmScheme(ctx, gv.ID, w.c1, doc.Version)
	require.NoError(t, err)
	w.items = map[string]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `select key, id from grade_items where course_id = $1`, w.c1)
	require.NoError(t, err)
	for rows.Next() {
		var k string
		var id uuid.UUID
		require.NoError(t, rows.Scan(&k, &id))
		w.items[k] = id
	}
	rows.Close()
	w.midterm, w.final = w.items["midterm"], w.items["final"]

	var sessions []uuid.UUID
	for i := 1; i <= 10; i++ {
		var id uuid.UUID
		require.NoError(t, r.pool.QueryRow(ctx, `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2::int, now() + $2::int * interval '1 day', now() + $2::int * interval '1 day' + interval '2 hours') returning id`, w.c1, i).Scan(&id))
		sessions = append(sessions, id)
	}
	w.rows = readCSV(t, gbFix+"inputs-lop1.csv")
	for _, row := range w.rows {
		acc := row["account"]
		u := r.addUser(uniq(strings.ReplaceAll(acc, ".", "")), store.UserRoleSTUDENT, store.UserStatusACTIVE)
		w.userOf[acc], w.byCode[row["student_code"]] = u.ID, acc
		r.must(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ROSTER', $3)`, w.c1, u.ID, row["student_code"])
		for _, k := range []struct {
			item uuid.UUID
			col  string
		}{{w.midterm, "midterm"}, {w.final, "final_exam"}} {
			r.must(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ($1, $2, $3, $4::numeric, $5)`, w.c1, k.item, u.ID, row[k.col], gv.ID)
		}
		if v := row["weekly_exam_t9"]; v != "" {
			r.must(`insert into exam_attempts (course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason, auto_score, graded_at)
				values ($1, $2, $3, 'GRADED', now() - interval '2 hours', now() - interval '90 minutes', now() - interval '100 minutes', 'MANUAL', $4::numeric, now() - interval '80 minutes')`, w.c1, exam, u.ID, v)
		}
		if v := row["speak_bonus_points"]; v != "" && v != "0.00" {
			r.must(`insert into participation_events (course_id, student_id, type, points, client_event_id, created_by) values ($1, $2, 'SPEAK', $3::numeric, $4, $5)`, w.c1, u.ID, v, uuid.New(), gv.ID)
		}
		if v := row["penalty_event_points"]; v != "" && v != "0.00" {
			r.must(`insert into participation_events (course_id, student_id, type, points, client_event_id, created_by) values ($1, $2, 'PENALTY', $3::numeric, $4, $5)`, w.c1, u.ID, "-"+v, uuid.New(), gv.ID)
		}
		n, err := strconv.Atoi(row["absent"])
		require.NoError(t, err)
		for i := range n {
			r.must(`insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, 'ABSENT', $4)`, w.c1, sessions[i], u.ID, gv.ID)
		}
		if acc == "sv.gioi" || acc == "sv.kha" || acc == "sv.nguyco" {
			w.sv[acc] = r.mustLogin(u.Email)
		}
	}
	// hai cột đã công bố như seed
	r.must(`update grade_items set published_at = now() where id = any($1)`, []uuid.UUID{w.midterm, w.final})
	return w
}

func (w *gbWorld) get(who session, path string) resp {
	return w.r.do(req{method: http.MethodGet, path: path, bearer: who.access})
}

func (w *gbWorld) gradebook() []map[string]any {
	w.r.t.Helper()
	var out []map[string]any
	cursor := ""
	for {
		res := w.get(w.gv, "/courses/"+w.c1.String()+"/gradebook?limit=100"+cursor)
		require.Equal(w.r.t, http.StatusOK, res.code, string(res.body))
		j := res.json()
		for _, x := range j["rows"].([]any) {
			out = append(out, x.(map[string]any))
		}
		next, _ := j["next_cursor"].(string)
		if next == "" {
			return out
		}
		cursor = "&cursor=" + next
	}
}

// TestSeedGradebookExpected — US-S8-SEED AC2: sổ lớp 1 dựng từ đúng tệp của BA ra 30/30 dòng khớp seed/expected_final_grades.csv (HP, điểm chữ), MSSV khớp theo dòng ROSTER (#20).
func TestSeedGradebookExpected(t *testing.T) {
	w := newGBWorld(t)
	want := map[string]map[string]string{}
	for _, e := range readCSV(t, gbSeed+"expected_final_grades.csv") {
		want[e["student_code"]] = e
	}
	rows := w.gradebook()
	require.Len(t, rows, 30)
	ok := 0
	for _, row := range rows {
		code := row["student"].(map[string]any)["student_code"].(string)
		res := row["result"].(map[string]any)
		e := want[code]
		require.NotNil(t, e, code)
		require.True(t, decimal.RequireFromString(res["hp"].(string)).Equal(decimal.RequireFromString(e["hp"])), "%s: %v ≠ %s", e["account"], res["hp"], e["hp"])
		require.Equal(t, e["letter"], res["letter"], e["account"])
		ok++
	}
	require.Equal(t, 30, ok)
	// sổ lớp 1 chưa chốt; lớp 2 không có cột
	require.Equal(t, "CONFIRMED", w.get(w.gv, "/courses/"+w.c1.String()+"/grade-scheme/status").json()["state"])
	require.NotEqual(t, "LOCKED", w.get(w.gv, "/courses/"+w.c1.String()+"/gradebook?limit=1").json()["state"].(map[string]any)["status"])
	res2 := w.get(w.gv, "/courses/"+w.c2.String()+"/gradebook").json()
	require.Empty(t, res2["columns"])
}

// TestSeedGradebookIsolation — AC8: A, B, C chỉ đọc điểm của chính mình; D (chưa vào lớp) không thấy lớp nào; TA thấy sổ lớp 1 (chỉ đọc) và không thấy lớp 2; canary không lọt.
func TestSeedGradebookIsolation(t *testing.T) {
	w := newGBWorld(t)
	b := w.userOf["sv.kha"]
	w.r.must(`update grade_entries set score = $3::numeric where item_id = $1 and student_id = $2`, w.final, b, canary)
	w.r.must(`update grade_items set published_at = null where id = $1`, w.final)
	for acc, mid := range map[string]string{"sv.gioi": "9", "sv.kha": "7.5", "sv.nguyco": "5"} {
		path := "/courses/" + w.c1.String() + "/me/grades"
		res := w.get(w.sv[acc], path)
		require.Equal(t, http.StatusOK, res.code, acc)
		var own struct {
			Components []struct{ Key, Score string } `json:"components"`
		}
		require.NoError(t, json.Unmarshal(res.body, &own))
		var got string
		for _, c := range own.Components {
			if c.Key == "midterm" {
				got = c.Score
			}
		}
		require.True(t, decimal.RequireFromString(got).Equal(decimal.RequireFromString(mid)), "%s: giữa kỳ %q ≠ %s", acc, got, mid)
		// tham số tự khai không đổi người xem
		other := w.userOf["sv.gioi"]
		if acc == "sv.gioi" {
			other = w.userOf["sv.kha"]
		}
		res2 := w.get(w.sv[acc], path+"?student_id="+other.String()+"&user_id="+other.String())
		require.Equal(t, res.body, res2.body, acc)
		for _, s := range []string{canary, strings.Replace(canary, ".", ",", 1)} {
			require.NotContains(t, string(res.body), s, acc)
		}
		require.Equal(t, http.StatusForbidden, w.get(w.sv[acc], "/courses/"+w.c1.String()+"/gradebook").code, acc)
		require.Equal(t, http.StatusForbidden, w.get(w.sv[acc], "/courses/"+w.c2.String()+"/me/grades").code, acc+" lớp 2")
	}
	// D chưa vào lớp nào
	require.Equal(t, http.StatusForbidden, w.get(w.dd, "/courses/"+w.c1.String()+"/me/grades").code)
	mine := w.get(w.dd, "/me/courses").json()
	require.Empty(t, mine["items"])
	// TA: sổ lớp 1 đọc được, ghi không; lớp 2 không thấy
	require.Equal(t, http.StatusOK, w.get(w.ta, "/courses/"+w.c1.String()+"/gradebook").code)
	put := w.r.do(req{method: http.MethodPut, path: "/courses/" + w.c1.String() + "/grade-entries", bearer: w.ta.access, body: map[string]any{"item_id": w.midterm, "student_id": b, "score": "1", "version": 0}})
	require.Equal(t, http.StatusForbidden, put.code)
	require.Equal(t, http.StatusForbidden, w.get(w.ta, "/courses/"+w.c2.String()+"/gradebook").code)
}

// TestSeedGradebookAdminForbidden — AC8: ADMIN 403 ở mọi route sổ điểm của lớp.
func TestSeedGradebookAdminForbidden(t *testing.T) {
	w := newGBWorld(t)
	c := w.c1.String()
	b := w.userOf["sv.kha"].String()
	for _, p := range []string{"/gradebook", "/gradebook/preview", "/grade-scheme", "/grade-scheme/status", "/gradebook/snapshots", "/gradebook/export.xlsx", "/gradebook/students/" + b + "/explain", "/gradebook/students/" + b + "/history", "/me/grades"} {
		require.Equal(t, http.StatusForbidden, w.get(w.admin, "/courses/"+c+p).code, p)
	}
	for _, p := range []string{"/gradebook/finalize", "/gradebook/unlock", "/grade-scheme/confirm", "/grade-scheme/extract", "/grade-scheme/impact", "/grade-entries/import", "/me/grades/what-if"} {
		res := w.r.do(req{path: "/courses/" + c + p, bearer: w.admin.access, body: map[string]any{}, hdr: map[string]string{"Idempotency-Key": "chk-" + uuid.NewString()}})
		require.Equal(t, http.StatusForbidden, res.code, p)
	}
}

// TestSeedRiskAfterGradebook — US-P6-09 AC8: điểm quá trình của seed ra đúng cột `qt_current` / `grade_signal` của seed/expected_risk.csv, và các hàng có cột hoạt động
// phân loại đúng (sv26: tuần 9 không hoạt động, tuần 8 có → WATCH; sv24, sv25, sv27, sv29: có hoạt động cả hai tuần → không WATCH; sv30 đúng 5,0 không thấp).
func TestSeedRiskAfterGradebook(t *testing.T) {
	w := newGBWorld(t)
	qt, err := grade.RiskQT(t.Context(), store.New(w.r.pool), w.c1)
	require.NoError(t, err)
	th := crm.DefaultThresholds()
	for _, e := range readCSV(t, gbSeed+"expected_risk.csv") {
		id := w.userOf[e["account"]]
		got, ok := qt[id]
		require.True(t, ok, e["account"])
		require.True(t, got.Equal(decimal.RequireFromString(e["qt_current"])), "%s: qt %s ≠ %s", e["account"], got, e["qt_current"])
		require.Equal(t, e["grade_signal"] == "LOW", got.LessThan(th.GradeLowBelow), e["account"])
		if e["activity_week9"] == "" {
			continue
		}
		act := func(s string) int {
			if s == "0" {
				return 0
			}
			return 3
		}
		weeks := []crm.Week{{Activity: act(e["activity_week9"]), Grade: &got}, {Activity: act(e["activity_week8"]), Grade: &got}}
		level := crm.Evaluate(weeks, th).Level
		if e["risk_level"] == "WATCH" {
			require.Equal(t, crm.LevelWatch, level, e["account"])
		} else {
			require.Equal(t, crm.LevelNone, level, e["account"])
		}
	}
}
