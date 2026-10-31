package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/testutil"
)

func joinCode() string {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	id := uuid.New()
	b := make([]byte, 7)
	for i := range b {
		b[i] = alphabet[int(id[i])%len(alphabet)]
	}
	return string(b)
}

// gradeRig: lớp có công thức weekly 40 (QT) + final 60 (CK) đã xác nhận, hai sinh viên A (weekly 8, final 6) và B (weekly 5, final 9); cột weekly đã công bố, final chưa.
type gradeRig struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	svc    *grade.Service
	course uuid.UUID
	gv     uuid.UUID
	a, b   uuid.UUID
	ad     GradeAdapter
}

func newGradeRig(t *testing.T) *gradeRig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	r := &gradeRig{ctx: t.Context(), pool: pool, svc: &grade.Service{Pool: pool}}
	r.ad = GradeAdapter{Grade: r.svc}
	exec := func(sql string, args ...any) {
		_, err := pool.Exec(r.ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	one := func(dst any, sql string, args ...any) {
		require.NoError(t, pool.QueryRow(r.ctx, sql, args...).Scan(dst), sql)
	}
	user := func(role, name string) (id uuid.UUID) {
		one(&id, `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, strings.ToLower(name)+"."+uuid.NewString()[:8]+"@example.test", name, role)
		return id
	}
	r.gv = user("TEACHER", "GV")
	one(&r.course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp thử', '2026-2027-HK1', $2, $3) returning id`, "L-"+uuid.NewString()[:6], joinCode(), r.gv)
	exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, r.course, r.gv)
	exec(`insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 1, $2)`, r.course, r.gv)
	rules := `[{"kind":"BONUS","applicable":true,"params":{"source":"PARTICIPATION","per_speak":"0.25","cap":"1.0"}},{"kind":"PENALTY","applicable":true,"params":{"source":"ABSENCE","free":2,"per":"0.5"}},{"kind":"ROUNDING","params":{"step":"0.5","mode":"HALF_UP"}},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"8.5","letter":"A"},{"min":"5","letter":"B"},{"min":"0","letter":"F"}]}}]`
	var rs []grade.PutRule
	require.NoError(t, json.Unmarshal([]byte(rules), &rs))
	doc, err := r.svc.PutDraft(r.ctx, r.gv, r.course, grade.PutInput{Version: 1, Rules: rs, Components: []grade.PutComponent{{Key: "weekly", Name: "Hằng tuần", Grp: "QT", Weight: "40"}, {Key: "final", Name: "Cuối kỳ", Grp: "CK", Weight: "60"}}})
	require.NoError(t, err)
	_, err = r.svc.ConfirmScheme(r.ctx, r.gv, r.course, doc.Version)
	require.NoError(t, err)
	items := map[string]uuid.UUID{}
	rows, err := pool.Query(r.ctx, `select key, id from grade_items where course_id = $1`, r.course)
	require.NoError(t, err)
	for rows.Next() {
		var k string
		var id uuid.UUID
		require.NoError(t, rows.Scan(&k, &id))
		items[k] = id
	}
	rows.Close()
	student := func(name, code string) uuid.UUID {
		u := user("STUDENT", name)
		exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN', $3)`, r.course, u, code)
		return u
	}
	r.a, r.b = student("An", "20220001"), student("Bình", "20220002")
	for _, e := range []struct {
		st       uuid.UUID
		item, sc string
	}{{r.a, "weekly", "8"}, {r.a, "final", "6"}, {r.b, "weekly", "5"}, {r.b, "final", "9"}} {
		exec(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ($1, $2, $3, $4::numeric, $5)`, r.course, items[e.item], e.st, e.sc, r.gv)
	}
	exec(`update grade_items set published_at = now() where id = $1`, items["weekly"])
	return r
}

func (r *gradeRig) tc(user uuid.UUID) TrustedContext {
	return TrustedContext{UserID: user, CourseID: r.course, Role: "STUDENT"}
}

func runTool(t *testing.T, reg *PrivateRegistry, name string, tc TrustedContext, args string) Result {
	t.Helper()
	res, err := reg.Run(t.Context(), name, tc, json.RawMessage(args))
	require.NoError(t, err)
	return res
}

func (r *gradeRig) registry() *PrivateRegistry {
	return DefaultPrivateRegistry(nil, nil, r.ad, nil, nil)
}

// TestToolGradeSummaryTrustedContext: danh tính chỉ từ trusted_context; id trong tham số của LLM bị bỏ qua — sinh viên A chỉ nhận số của A.
func TestToolGradeSummaryTrustedContext(t *testing.T) {
	r := newGradeRig(t)
	res := runTool(t, r.registry(), "get_my_grade_summary", r.tc(r.a), `{"user_id":"`+r.b.String()+`","student_code":"20220002"}`)
	require.False(t, res.NoData)
	raw, _ := json.Marshal(res.Block)
	require.Contains(t, string(raw), `"score":"8"`)
	require.NotContains(t, string(raw), `"score":"5"`)
	other := runTool(t, r.registry(), "get_my_grade_summary", r.tc(r.b), `{}`)
	rawB, _ := json.Marshal(other.Block)
	require.Contains(t, string(rawB), `"score":"5"`)
	require.NotContains(t, string(rawB), `"score":"8"`)
	// điểm chưa công bố (final của B = 9) không có trong dữ liệu của chính B
	require.NotContains(t, string(rawB), `"score":"9"`)
}

func TestToolWhatIfNoIdentityParam(t *testing.T) {
	typ := reflect.TypeFor[WhatIfArgs]()
	for i := range typ.NumField() {
		tag := strings.ToLower(typ.Field(i).Tag.Get("json"))
		for _, bad := range []string{"user", "student", "course", "id"} {
			require.NotContains(t, tag, bad, "tham số của tool không nhận danh tính")
		}
	}
	for _, name := range []string{"get_my_grade_summary", "what_if_final_grade"} {
		tl, ok := DefaultPrivateRegistry(nil, nil, nil, nil, nil).tools[name]
		require.True(t, ok, name)
		if name == "get_my_grade_summary" {
			require.Zero(t, tl.ArgsType().NumField(), "get_my_grade_summary không có tham số")
		}
	}
	r := newGradeRig(t)
	res := runTool(t, r.registry(), "what_if_final_grade", r.tc(r.a), `{"assumed":{"final":"8"},"user_id":"`+r.b.String()+`","course_id":"`+uuid.NewString()+`"}`)
	require.False(t, res.NoData)
	raw, _ := json.Marshal(res.Block)
	require.Contains(t, string(raw), `"hp"`)
}

// TestToolGradeNumbersMatchMeGrades: số của tool bằng số của `GET /me/grades` và của tính thử (cùng engine, không đường thứ hai).
func TestToolGradeNumbersMatchMeGrades(t *testing.T) {
	r := newGradeRig(t)
	me, err := r.svc.MeGrades(r.ctx, r.course, r.a)
	require.NoError(t, err)
	f, ok, err := r.ad.Summary(r.ctx, r.tc(r.a))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, me.QT, f["qt"])
	require.Equal(t, me.Components[0].Score, f["components"].([]any)[0].(map[string]any)["score"])
	w, err := r.svc.MeWhatIf(r.ctx, r.course, r.a, grade.WhatIfInput{FinalExamScore: new("8")})
	require.NoError(t, err)
	wf, ok, err := r.ad.WhatIf(r.ctx, r.tc(r.a), map[string]decimal.Decimal{"final": decimal.NewFromInt(8)})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, w.HP, wf["hp"])
	require.Equal(t, w.Letter, wf["letter"])
	// điểm chữ mục tiêu
	mf, ok, err := r.ad.MinFinal(r.ctx, r.tc(r.a), "A")
	require.NoError(t, err)
	require.True(t, ok)
	wt, err := r.svc.MeWhatIf(r.ctx, r.course, r.a, grade.WhatIfInput{TargetLetter: new("A")})
	require.NoError(t, err)
	if wt.MinFinalExam != nil {
		require.Equal(t, *wt.MinFinalExam, mf["min_final_exam_score"])
	}
	// chưa đủ điểm quá trình đã công bố → không bịa
	_, err = r.pool.Exec(r.ctx, `update grade_items set published_at = null`)
	require.NoError(t, err)
	_, ok, err = r.ad.WhatIf(r.ctx, r.tc(r.a), map[string]decimal.Decimal{"final": decimal.NewFromInt(8)})
	require.NoError(t, err)
	require.False(t, ok)
}

// TestToolGradeNotInPublicRegistry: hai tool điểm chỉ ở kênh chat riêng.
func TestToolGradeNotInPublicRegistry(t *testing.T) {
	pub := NewPublicRegistry(NewLibraryTool(nil))
	for _, n := range []string{"get_my_grade_summary", "what_if_final_grade"} {
		require.NotContains(t, pub.Names(), n)
		_, err := pub.Run(t.Context(), n, TrustedContext{}, nil)
		require.Error(t, err)
	}
	require.Contains(t, DefaultPrivateRegistry(nil, nil, nil, nil, nil).Names(), "get_my_grade_summary")
	require.Contains(t, DefaultPrivateRegistry(nil, nil, nil, nil, nil).Names(), "what_if_final_grade")
}

// TestGradeFormulaFromConfirmedScheme (AC6): khối cấu trúc từ công thức ĐÃ XÁC NHẬN: thành phần + trọng số + điểm cộng / trừ / làm tròn / thang chữ + nguồn "Quy chế học phần, tr. N".
func TestGradeFormulaFromConfirmedScheme(t *testing.T) {
	r := newGradeRig(t)
	_, err := r.pool.Exec(r.ctx, `update grade_components set citations = '[{"page":3,"quote":"Điểm quá trình chiếm 40%."}]' where course_id = $1 and key = 'weekly'`, r.course)
	require.NoError(t, err)
	f, ok, err := r.ad.Scheme(r.ctx, r.tc(r.a))
	require.NoError(t, err)
	require.True(t, ok)
	comps := f["components"].([]map[string]any)
	require.Len(t, comps, 2)
	require.Equal(t, "40", comps[0]["weight_percent"])
	require.Equal(t, "60", comps[1]["weight_percent"])
	require.Equal(t, map[string]any{"per_speak": "0.25", "cap": "1"}, f["bonus"])
	require.Equal(t, map[string]any{"free_absences": 2, "per_absence": "0.5"}, f["penalty"])
	require.Equal(t, map[string]any{"step": "0.5", "mode": "HALF_UP"}, f["rounding"])
	require.Len(t, f["letter_scale"], 3)
	require.Equal(t, "Quy chế học phần, tr. 3", f["source"])
}
