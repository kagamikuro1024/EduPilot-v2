package today_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

// teaching: giảng viên dạy các lớp cs; trả người + phiên.
func (r *rig) teaching(cs ...cls) (store.User, session) {
	r.t.Helper()
	u, s := r.person(store.UserRoleTEACHER, true)
	for _, c := range cs {
		r.enrollAt(c, u, "TEACHER", "ACTIVE", "", 24*time.Hour)
	}
	return u, s
}

func (r *rig) pending(c cls, n int, age time.Duration, warning string) {
	r.t.Helper()
	for range n {
		p := r.addUser(uniq("p"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
		r.enrollAt(c, p, "STUDENT", "PENDING", warning, age)
	}
}

func classOfAction(a any) string {
	c, _ := a.(map[string]any)["course"].(map[string]any)
	s, _ := c["class_code"].(string)
	return s
}

// ===== Giảng viên / TA (AC4) =====

func TestStaffTodaySingleCourse(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761987"), r.klass("761988")
	_, g := r.teaching(c1, c2)
	r.pending(c1, 2, time.Hour, "")
	r.pending(c2, 1, time.Hour, "")
	res := r.tget(g, "/courses/"+c1.sid()+"/today")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	acts := actionsOf(res)
	var codes []string
	for _, a := range acts {
		codes = append(codes, classOfAction(a))
	}
	require.NotEmpty(t, acts)
	for _, c := range codes {
		require.Equal(t, "761987", c, "một lớp: chỉ việc của lớp đó")
	}
	require.Equal(t, []any{}, res.json()["attention"], "attention rỗng ở P2")
}

func TestStaffTodayAllCourses(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761987"), r.klass("761988")
	_, g := r.teaching(c1, c2)
	r.pending(c1, 2, time.Hour, "")
	r.pending(c2, 1, time.Hour, "")
	j := r.tget(g, "/me/today").json()
	seen := map[string]bool{}
	for _, a := range j["actions"].([]any) {
		seen[classOfAction(a)] = true
		require.NotEmpty(t, a.(map[string]any)["course"], "mỗi mục ghi rõ lớp nào")
	}
	require.True(t, seen["761987"] && seen["761988"], "việc của cả hai lớp")
}

func TestStaffTodayReasonsFromData(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	r.pending(c, 3, 49*time.Hour, "") // cũ nhất 2 ngày
	acts := actionsOf(r.tget(g, "/me/today"))
	var req map[string]any
	for _, a := range acts {
		if a.(map[string]any)["kind"] == "JOIN_REQUEST" {
			req = a.(map[string]any)
		}
	}
	require.NotNil(t, req)
	require.Equal(t, "3 yêu cầu vào lớp 761988 đang chờ duyệt", req["title"])
	require.Equal(t, "Cũ nhất đã chờ 2 ngày.", req["reason"])
	require.Equal(t, fmt.Sprintf("/class/members?course=%s&tab=pending", c.sid()), req["href"])
	require.Equal(t, float64(49*60), req["age_minutes"])
	require.Equal(t, "overdue", req["urgency"])
}

func TestStaffTodayCountAndCap(t *testing.T) {
	r := newRig(t)
	_, g := r.person(store.UserRoleTEACHER, true)
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	var gv uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id from users where role = 'TEACHER' order by created_at desc limit 1`).Scan(&gv))
	for i := range 55 {
		c := cls{id: r.course(adm.ID, fmt.Sprintf("CAP%02d", i)), code: fmt.Sprintf("CAP%02d", i)}
		_, err := r.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, c.id, gv)
		require.NoError(t, err)
		r.pending(c, 1, time.Hour, "")
	}
	j := r.tget(g, "/me/today").json()
	require.Equal(t, float64(55+55), j["count"], "55 yêu cầu + 55 mục thiết lập")
	require.Len(t, j["actions"], 50, "tối đa 50 mục")
}

func TestStaffTodayUpcoming(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	now := r.clk.Now()
	r.session(c, 1, now.Add(-3*time.Hour), time.Hour, "P.1") // đã xong: không hiện
	for i := 2; i <= 14; i++ {
		r.session(c, i, now.Add(time.Duration(i)*7*time.Hour), time.Hour, fmt.Sprintf("P.%d", i))
	}
	up := r.tget(g, "/me/today").json()["upcoming"].([]any)
	require.Len(t, up, 10, "tối đa 10")
	require.Equal(t, "Buổi 2 · An ninh mạng", up[0].(map[string]any)["title"])
	var last string
	for _, u := range up {
		at := u.(map[string]any)["at"].(string)
		require.GreaterOrEqual(t, at, last, "theo starts_at")
		last = at
		tm, err := time.Parse(time.RFC3339, at)
		require.NoError(t, err)
		require.True(t, tm.Before(now.AddDate(0, 0, 7)), "trong 7 ngày")
	}
}

// ===== Provider P2 (AC5) =====

func TestProviderJoinRequest(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	ta, st := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", time.Hour)
	r.pending(c, 2, 90*time.Minute, "")
	r.pending(c, 1, time.Hour, "EMAIL_UNVERIFIED") // chờ xác minh email của roster: không phải yêu cầu cần duyệt
	for who, s := range map[string]session{"giảng viên": g, "trợ giảng": st} {
		var req map[string]any
		for _, a := range actionsOf(r.tget(s, "/me/today")) {
			if a.(map[string]any)["kind"] == "JOIN_REQUEST" {
				req = a.(map[string]any)
			}
		}
		require.NotNil(t, req, who)
		require.Equal(t, "2 yêu cầu vào lớp 761988 đang chờ duyệt", req["title"], who)
		require.Equal(t, "Cũ nhất đã chờ 1 giờ.", req["reason"], who)
	}
}

func TestProviderEmailMismatchTeacherOnly(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	ta, st := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", time.Hour)
	r.pending(c, 2, time.Hour, "EMAIL_MISMATCH")
	gk := kinds(actionsOf(r.tget(g, "/me/today")))
	require.Contains(t, gk, "EMAIL_MISMATCH")
	require.NotContains(t, gk, "JOIN_REQUEST", "hàng email chưa khớp không tính vào yêu cầu thường")
	for _, a := range actionsOf(r.tget(g, "/me/today")) {
		if a.(map[string]any)["kind"] == "EMAIL_MISMATCH" {
			require.Equal(t, "2 yêu cầu có email chưa khớp MSSV · lớp 761988", a.(map[string]any)["title"])
			require.Equal(t, "Cần bạn xác nhận: email đăng ký khác email trong danh sách lớp.", a.(map[string]any)["reason"])
		}
	}
	for _, k := range kinds(actionsOf(r.tget(st, "/me/today"))) {
		require.NotContains(t, []string{"EMAIL_MISMATCH", "COURSE_SETUP"}, k, "TA không thấy việc của giảng viên")
	}
}

func TestProviderCourseSetupSteps(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	setup := func() map[string]any {
		for _, a := range actionsOf(r.tget(g, "/courses/"+c.sid()+"/today")) {
			if a.(map[string]any)["kind"] == "COURSE_SETUP" {
				return a.(map[string]any)
			}
		}
		return nil
	}
	done := func(m map[string]any) map[string]bool {
		out := map[string]bool{}
		for _, s := range m["steps"].([]any) {
			out[s.(map[string]any)["key"].(string)] = s.(map[string]any)["done"].(bool)
		}
		return out
	}
	m := setup()
	require.Equal(t, "Thiết lập lớp mới · 761988", m["title"])
	require.Equal(t, "0/4 bước xong: chia sẻ mã lớp → tải quy chế môn học → tạo lịch buổi học → tải tài liệu.", m["reason"])
	require.Equal(t, "/class/settings?course="+c.sid(), m["href"])
	require.Equal(t, map[string]bool{"share_code": false, "policy": false, "sessions": false, "documents": false}, done(m))
	u, _ := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "PENDING", "", time.Hour) // ≥ 1 sinh viên ACTIVE / PENDING
	require.True(t, done(setup())["share_code"])
	r.session(c, 1, r.clk.Now().Add(time.Hour), time.Hour, "")
	require.True(t, done(setup())["sessions"])
	r.document(c, "COURSE_POLICY", "READY")
	require.True(t, done(setup())["policy"])
	r.document(c, "LECTURE", "FAILED") // FAILED không tính
	require.False(t, done(setup())["documents"])
	r.document(c, "LECTURE", "READY")
	m = setup()
	require.Nil(t, m, "xong cả bốn ⇒ mục biến mất")
}

func (r *rig) document(c cls, typ, status string) string {
	r.t.Helper()
	return r.scalar(`insert into documents (course_id, title, type, status) values ($1, 'T', $2::document_type, $3::document_status) returning id::text`, c.id, typ, status)
}

func TestProviderCourseSetupDisappears(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	u, _ := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "ACTIVE", "", time.Hour)
	r.session(c, 1, r.clk.Now().Add(time.Hour), time.Hour, "")
	r.document(c, "COURSE_POLICY", "READY")
	// Tài liệu CHIA SẺ vào lớp cũng tính cho bước "tải tài liệu".
	other := r.klass("761989")
	d := r.document(other, "LECTURE", "READY")
	_, err := r.pool.Exec(t.Context(), `insert into document_courses (document_id, course_id) values ($1, $2)`, d, c.id)
	require.NoError(t, err)
	require.NotContains(t, kinds(actionsOf(r.tget(g, "/me/today"))), "COURSE_SETUP")
}

func TestProviderCourseSetupDismiss(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	ta, st := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", time.Hour)
	_, sv := r.student("")
	require.Contains(t, kinds(actionsOf(r.tget(g, "/me/today"))), "COURSE_SETUP")
	for name, s := range map[string]session{"TA": st, "sinh viên": sv, "không JWT": {}} {
		want := http.StatusForbidden
		if name == "không JWT" {
			want = http.StatusUnauthorized
		}
		require.Equal(t, want, r.do(req{method: http.MethodPost, path: "/courses/" + c.sid() + "/setup/dismiss", bearer: s.access}).code, name)
	}
	require.Equal(t, http.StatusNoContent, r.do(req{method: http.MethodPost, path: "/courses/" + c.sid() + "/setup/dismiss", bearer: g.access}).code)
	require.Equal(t, http.StatusNoContent, r.do(req{method: http.MethodPost, path: "/courses/" + c.sid() + "/setup/dismiss", bearer: g.access}).code, "idempotent")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where course_id = $1 and action = 'setup_dismissed'`, c.id), "chỉ ghi một lần")
	// Cache cũ có thể còn đến khi sự kiện course.changed được xử lý; xoá tay như worker sẽ làm.
	deliver(t, r, today.TopicChanged, c)
	require.NotContains(t, kinds(actionsOf(r.tget(g, "/me/today"))), "COURSE_SETUP")
}

func TestProviderOverdueAfter48h(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	r.pending(c, 1, 3*time.Hour, "EMAIL_MISMATCH") // bậc 40
	r.pending(c, 1, 49*time.Hour, "")              // bậc 45 nhưng quá hạn
	acts := actionsOf(r.tget(g, "/me/today"))
	require.Equal(t, []string{"JOIN_REQUEST", "EMAIL_MISMATCH", "COURSE_SETUP"}, kinds(acts), "quá hạn nổi lên trước mọi bậc")
	r2 := newRig(t)
	c2 := r2.klass("761987")
	_, g2 := r2.teaching(c2)
	r2.pending(c2, 1, 3*time.Hour, "EMAIL_MISMATCH")
	r2.pending(c2, 1, 47*time.Hour, "")
	require.Equal(t, []string{"EMAIL_MISMATCH", "JOIN_REQUEST", "COURSE_SETUP"}, kinds(actionsOf(r2.tget(g2, "/me/today"))), "47 giờ chưa quá hạn")
}

// deliver xử lý một sự kiện outbox đồng bộ bằng Invalidator (như worker sẽ làm).
func deliver(t *testing.T, r *rig, topic string, c cls, users ...uuid.UUID) {
	t.Helper()
	payload := fmt.Sprintf(`{"course_id":%q`, c.sid())
	if len(users) > 0 {
		payload += fmt.Sprintf(`,"user_id":%q`, users[0])
	}
	payload += "}"
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
	require.NoError(t, inv.Handle(context.Background(), outbox.Message{ID: uuid.New(), Topic: topic, Payload: []byte(payload)}))
}

// ===== Admin (AC6) =====

type fakeLLM struct {
	pct  int
	ok   bool
	open map[string]bool
}

func (f fakeLLM) BudgetPercent(context.Context) (int, bool) { return f.pct, f.ok }
func (f fakeLLM) OpenCircuit(_ context.Context, id string) bool {
	return f.open[id]
}

func (r *rig) adminItems(sig today.LLMSignals) []today.Item {
	r.t.Helper()
	items, err := today.AdminProvider{Pool: r.pool, LLM: sig}.Items(t(r), today.Viewer{Role: today.RoleAdmin, Now: r.clk.Now()}, today.Scope{})
	require.NoError(r.t, err)
	return items
}

func t(r *rig) context.Context { return r.t.Context() }

func (r *rig) llmProvider(name string, enabled bool, ok *bool) string {
	r.t.Helper()
	return r.scalar(`insert into llm_providers (type, name, enabled, last_test_ok) values ('fake', $1, $2, $3) returning id::text`, name, enabled, ok)
}

func find(items []today.Item, k today.Kind) []today.Item {
	var out []today.Item
	for _, it := range items {
		if it.Kind == k {
			out = append(out, it)
		}
	}
	return out
}

func TestAdminTodayLLMProviderError(t *testing.T) {
	r := newRig(t)
	f, tr := false, true
	bad := r.llmProvider("OpenAI chính", true, &f)
	r.llmProvider("Dự phòng", true, &tr)
	r.llmProvider("Đã tắt", false, &f)
	open := r.llmProvider("Mạch mở", true, &tr)
	items := find(r.adminItems(fakeLLM{open: map[string]bool{open: true}}), today.KindLLMProviderError)
	require.Len(t, items, 2)
	reasons := items[0].Reason + "|" + items[1].Reason
	require.Contains(t, reasons, "OpenAI chính không phản hồi. Chat của sinh viên có thể dùng dự phòng.")
	require.Contains(t, reasons, "Mạch mở không phản hồi.")
	require.NotContains(t, reasons, "Đã tắt")
	_ = bad
	require.Equal(t, "/settings/llm", items[0].Href)
	require.Equal(t, today.TierLLMProviderError, items[0].Tier)
}

func TestAdminTodayBudget(t *testing.T) {
	r := newRig(t)
	require.Empty(t, find(r.adminItems(fakeLLM{pct: 79, ok: true}), today.KindLLMBudgetWarn))
	w := find(r.adminItems(fakeLLM{pct: 85, ok: true}), today.KindLLMBudgetWarn)
	require.Len(t, w, 1)
	require.Equal(t, "Chi phí AI hôm nay đã dùng 85 % ngân sách.", w[0].Reason)
	o := find(r.adminItems(fakeLLM{pct: 100, ok: true}), today.KindLLMBudgetOut)
	require.Len(t, o, 1)
	require.Less(t, o[0].Tier, w[0].Tier, "cạn gấp hơn cảnh báo")
	require.Empty(t, find(r.adminItems(fakeLLM{pct: 99, ok: false}), today.KindLLMBudgetWarn), "không có hạn mức: không báo")
}

func TestAdminTodayCourseNoTeacher(t *testing.T) {
	r := newRig(t)
	c := r.klass("761999")
	got := find(r.adminItems(nil), today.KindCourseNoTeacher)
	var mine *today.Item
	for i := range got {
		if got[i].Course.ClassCode == "761999" {
			mine = &got[i]
		}
	}
	require.NotNil(t, mine)
	require.Equal(t, "Lớp 761999 chưa có giảng viên", mine.Title)
	require.Equal(t, "/admin/courses", mine.Href)
	r.teaching(c)
	for _, it := range find(r.adminItems(nil), today.KindCourseNoTeacher) {
		require.NotEqual(t, "761999", it.Course.ClassCode, "có giảng viên ACTIVE thì hết")
	}
}

func TestAdminTodayExpiredInvites(t *testing.T) {
	r := newRig(t)
	before := len(find(r.adminItems(nil), today.KindInviteExpired))
	for range 2 {
		u := r.addUser(uniq("moi"), store.UserRoleTEACHER, store.UserStatusINVITED)
		_, err := r.pool.Exec(t.Context(), `insert into auth_tokens (user_id, kind, token_hash, expires_at) values ($1, 'INVITE', $2, $3)`, u.ID, strings.Repeat("a", 60)+uuid.NewString()[:4], r.clk.Now().Add(-time.Hour))
		require.NoError(t, err)
	}
	live := r.addUser(uniq("conhan"), store.UserRoleTEACHER, store.UserStatusINVITED)
	_, err := r.pool.Exec(t.Context(), `insert into auth_tokens (user_id, kind, token_hash, expires_at) values ($1, 'INVITE', $2, $3), ($1, 'INVITE', $4, $5)`, live.ID, strings.Repeat("b", 60)+"0001", r.clk.Now().Add(-time.Hour), strings.Repeat("b", 60)+"0002", r.clk.Now().Add(time.Hour))
	require.NoError(t, err)
	items := find(r.adminItems(nil), today.KindInviteExpired)
	require.Len(t, items, 1)
	require.Equal(t, fmt.Sprintf("%d lời mời giảng viên đã hết hạn", 2+before), items[0].Title, "còn một lời mời sống thì không tính")
	require.Equal(t, "/admin/users?status=INVITED", items[0].Href)
}

func TestAdminCannotCourseToday(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, a := r.person(store.UserRoleADMIN, true)
	require.Equal(t, http.StatusForbidden, r.tget(a, "/courses/"+c.sid()+"/today").code)
	j := r.tget(a, "/me/today").json()
	require.Contains(t, j, "actions")
	require.NotContains(t, j, "upcoming")
	require.NotContains(t, j, "timeline")
}

// ===== Phân quyền theo vai (AC8) =====

func TestTodayRBACMatrix(t *testing.T) {
	r := newRig(t)
	c, other := r.klass("761988"), r.klass("761999")
	_, g := r.teaching(c)
	_, g2 := r.teaching(other)
	ta, st := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", time.Hour)
	sv, ss := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, sv, "STUDENT", "ACTIVE", "", time.Hour)
	pe, sp := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, pe, "STUDENT", "PENDING", "", time.Hour)
	re, sr := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, re, "STUDENT", "REMOVED", "", time.Hour)
	_, so := r.person(store.UserRoleSTUDENT, true)
	_, sa := r.person(store.UserRoleADMIN, true)

	shape := func(res resp) string {
		j := res.json()
		switch {
		case j["recommended"] != nil || j["timeline"] != nil:
			return "student"
		case j["upcoming"] != nil:
			return "staff"
		case j["actions"] != nil:
			return "admin"
		}
		return "?"
	}
	me := []struct {
		name  string
		s     session
		code  int
		shape string
	}{
		{"không JWT", session{}, 401, ""}, {"sinh viên", ss, 200, "student"}, {"sinh viên ngoài lớp", so, 200, "student"},
		{"sinh viên chờ duyệt", sp, 200, "student"}, {"TA", st, 200, "staff"}, {"giảng viên", g, 200, "staff"}, {"admin", sa, 200, "admin"},
	}
	for _, tc := range me {
		for _, q := range []string{"", "?role=ADMIN", "?scope=all&shape=staff"} { // tham số của client không đổi dạng
			res := r.tget(tc.s, "/me/today"+q)
			require.Equal(t, tc.code, res.code, tc.name+q)
			if tc.code == 200 {
				require.Equal(t, tc.shape, shape(res), tc.name+q)
			}
		}
	}
	co := []struct {
		name  string
		s     session
		code  int
		shape string
	}{
		{"không JWT", session{}, 401, ""}, {"sinh viên của lớp", ss, 200, "student"}, {"TA của lớp", st, 200, "staff"}, {"giảng viên của lớp", g, 200, "staff"},
		{"giảng viên lớp khác", g2, 403, ""}, {"sinh viên ngoài lớp", so, 403, ""}, {"PENDING", sp, 403, ""}, {"REMOVED", sr, 403, ""}, {"admin", sa, 403, ""},
	}
	for _, tc := range co {
		res := r.tget(tc.s, "/courses/"+c.sid()+"/today")
		require.Equal(t, tc.code, res.code, tc.name+": "+string(res.body))
		if tc.code == 200 {
			require.Equal(t, tc.shape, shape(res), tc.name)
		}
	}
	require.Equal(t, http.StatusNotFound, r.tget(g, "/courses/khong-phai-uuid/today").code)
}

// ===== Cache (AC9) và truy vấn (AC10) =====

func TestTodayCacheHit(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, g := r.teaching(c)
	r.pending(c, 1, time.Hour, "")
	first := r.tgetC(g, "/me/today")
	before := r.qc.n.Load()
	second := r.tgetC(g, "/me/today")
	require.Equal(t, first.body, second.body)
	require.Zero(t, r.qc.n.Load()-before, "lần hai trong 60 s: 0 truy vấn DB")
	ttl := r.rdb.TTL(t.Context(), today.CacheKey(u.ID, "all")).Val()
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, 60*time.Second)
}

func TestTodayTTLSafetyNet(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, g := r.teaching(c)
	r.pending(c, 1, time.Hour, "")
	r.tgetC(g, "/me/today")
	require.Contains(t, kinds(actionsOf(r.tgetC(g, "/me/today"))), "JOIN_REQUEST")
	// Mất sự kiện: dữ liệu đổi mà không ai xoá cache ⇒ vẫn là bản cũ, nhưng khoá có hạn (lưới an toàn).
	_, err := r.pool.Exec(t.Context(), `update enrollments set status = 'REMOVED', removed_at = now() where course_id = $1 and status = 'PENDING'`, c.id)
	require.NoError(t, err)
	require.Contains(t, kinds(actionsOf(r.tgetC(g, "/me/today"))), "JOIN_REQUEST", "còn cache")
	ttl := r.rdb.TTL(t.Context(), today.CacheKey(u.ID, "all")).Val()
	require.True(t, ttl > 0 && ttl <= 60*time.Second, "TTL %s", ttl)
	require.True(t, r.rdb.PExpire(t.Context(), today.CacheKey(u.ID, "all"), 5*time.Millisecond).Val())
	time.Sleep(50 * time.Millisecond)
	require.NotContains(t, kinds(actionsOf(r.tgetC(g, "/me/today"))), "JOIN_REQUEST", "hết TTL ⇒ tính lại")
}

// Cả 7 topic xoá đúng khoá của người bị ảnh hưởng.
func TestTodayInvalidatedByOutbox(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	gu, g := r.teaching(c)
	ta, st := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", time.Hour)
	su, ss := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	bystander, sb := r.person(store.UserRoleSTUDENT, true)
	adm, sa := r.person(store.UserRoleADMIN, true)
	warm := func() {
		for _, s := range []session{g, st, ss, sb, sa} {
			r.tgetC(s, "/me/today")
			r.tgetC(s, "/courses/"+c.sid()+"/today")
		}
	}
	exists := func(u uuid.UUID, scope string) bool {
		return r.rdb.Exists(t.Context(), today.CacheKey(u, scope)).Val() == 1
	}
	for _, tc := range []struct {
		topic   string
		gone    []uuid.UUID // người có khoá `all` bị xoá
		kept    []uuid.UUID
		payload string
	}{
		{today.TopicJoinRequested, []uuid.UUID{gu.ID, ta.ID, su.ID}, []uuid.UUID{bystander.ID}, fmt.Sprintf(`{"course_id":%q,"user_id":%q}`, c.sid(), su.ID)},
		{today.TopicJoinDecided, []uuid.UUID{gu.ID, ta.ID, su.ID}, []uuid.UUID{bystander.ID}, fmt.Sprintf(`{"course_id":%q,"user_id":%q}`, c.sid(), su.ID)},
		{today.TopicMemberChanged, []uuid.UUID{gu.ID, ta.ID, su.ID}, []uuid.UUID{bystander.ID}, fmt.Sprintf(`{"course_id":%q,"user_id":%q}`, c.sid(), su.ID)},
		{today.TopicAssigned, []uuid.UUID{gu.ID, ta.ID, su.ID, adm.ID}, []uuid.UUID{bystander.ID}, fmt.Sprintf(`{"course_id":%q,"user_id":%q}`, c.sid(), su.ID)},
		{today.TopicChanged, []uuid.UUID{gu.ID, ta.ID, su.ID, adm.ID}, []uuid.UUID{bystander.ID}, fmt.Sprintf(`{"course_id":%q}`, c.sid())},
		{today.TopicUserVerified, []uuid.UUID{su.ID}, []uuid.UUID{bystander.ID, gu.ID}, fmt.Sprintf(`{"user_id":%q}`, su.ID)},
		{today.TopicRosterImport, []uuid.UUID{gu.ID, ta.ID, su.ID}, []uuid.UUID{bystander.ID}, fmt.Sprintf(`{"course_id":%q,"user_ids":[%q]}`, c.sid(), su.ID)},
	} {
		warm()
		inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
		require.NoError(t, inv.Handle(t.Context(), outbox.Message{ID: uuid.New(), Topic: tc.topic, Payload: []byte(tc.payload)}), tc.topic)
		for _, u := range tc.gone {
			require.False(t, exists(u, "all"), "%s: khoá all của %s phải bị xoá", tc.topic, u)
		}
		for _, u := range tc.kept {
			require.True(t, exists(u, "all"), "%s: khoá của người không liên quan phải còn", tc.topic)
		}
	}
	require.Equal(t, 21, len(today.Topics()), "7 topic của P2 + 4 topic bài thi (US-PE-04) + 2 topic lượt làm (US-PE-05) + 2 topic so độ giống (US-PE-07: exam.similarity_done / similarity_reviewed)")
	// user.verified xoá cả khoá theo từng lớp của người đó.
	warm()
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{ID: uuid.New(), Topic: today.TopicUserVerified, Payload: []byte(fmt.Sprintf(`{"user_id":%q}`, su.ID))}))
	require.False(t, exists(su.ID, c.sid()))
}

// Đầu-cuối: duyệt hết yêu cầu ⇒ JOIN_REQUEST biến khỏi "Hôm nay" của giảng viên ≤ 2 s nhờ sự kiện, không đợi 60 s.
func TestTodayInvalidatedByApprove(t *testing.T) {
	r := newRig(t)
	r.worker()
	c := r.klass("761988")
	_, g := r.teaching(c)
	r.pending(c, 2, time.Hour, "")
	require.Contains(t, kinds(actionsOf(r.tgetC(g, "/me/today"))), "JOIN_REQUEST")
	rows, err := r.pool.Query(t.Context(), `select user_id::text from enrollments where course_id = $1 and status = 'PENDING'`, c.id)
	require.NoError(t, err)
	var uids []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		uids = append(uids, s)
	}
	rows.Close()
	for _, uid := range uids {
		res := r.do(req{method: http.MethodPost, path: "/courses/" + c.sid() + "/members/" + uid + "/approve", bearer: g.access, body: map[string]any{}})
		require.Equal(t, http.StatusOK, res.code, string(res.body))
	}
	took := waitFor(t, 2*time.Second, "JOIN_REQUEST biến khỏi Hôm nay", func() bool {
		return !strings.Contains(strings.Join(kinds(actionsOf(r.tgetC(g, "/me/today"))), ","), "JOIN_REQUEST")
	})
	require.Less(t, took, 2*time.Second)
}

func TestTodayRedisDownFailsOpen(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, _ := r.teaching(c)
	r.pending(c, 1, time.Hour, "")
	logs := &bytes.Buffer{}
	svc := today.NewService(r.pool, r.rdb, r.clk, slog.New(slog.NewTextHandler(logs, nil)), nil)
	require.NoError(t, r.rdb.Close()) // mọi lệnh Redis lỗi
	for range 5 {
		body, err := svc.Get(t.Context(), u.ID, today.RoleTeacher, today.Scope{})
		require.NoError(t, err, "Redis chết: tính trực tiếp")
		require.Contains(t, string(body), "JOIN_REQUEST")
	}
	require.Equal(t, 1, strings.Count(logs.String(), "Redis lỗi"), "log ≤ 1 dòng / 30 s")
}

func TestTodayQueryBudget(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761987"), r.klass("761988")
	_, g := r.teaching(c1, c2)
	for _, c := range []cls{c1, c2} {
		for range 30 {
			p := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
			r.enrollAt(c, p, "STUDENT", "ACTIVE", "", time.Hour)
		}
		r.pending(c, 2, time.Hour, "")
	}
	su, ss := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c1, su, "STUDENT", "ACTIVE", "", time.Hour)
	for name, tc := range map[string]struct {
		s    session
		path string
	}{
		"giảng viên tất cả lớp": {g, "/me/today"}, "giảng viên một lớp": {g, "/courses/" + c1.sid() + "/today"}, "sinh viên": {ss, "/courses/" + c1.sid() + "/today"},
	} {
		before := r.qc.n.Load()
		require.Equal(t, http.StatusOK, r.tgetC(tc.s, tc.path).code, name)
		require.LessOrEqual(t, r.qc.n.Load()-before, int64(5), name+": ≤ 5 truy vấn kể cả guard")
	}
}

func TestTodayETag(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	first := r.tgetC(g, "/me/today")
	etag := first.hdr.Get("ETag")
	require.NotEmpty(t, etag)
	require.Equal(t, "private, no-cache", first.hdr.Get("Cache-Control"))
	second := r.do(req{method: http.MethodGet, path: "/me/today", bearer: g.access, hdr: map[string]string{"If-None-Match": etag}})
	require.Equal(t, http.StatusNotModified, second.code)
	require.Empty(t, second.body)
}

// ===== Nhánh lỗi (AC14) =====

type boomProvider struct{}

func (boomProvider) Name() string { return "boom" }
func (boomProvider) Items(context.Context, today.Viewer, today.Scope) ([]today.Item, error) {
	return nil, errors.New("hỏng")
}

func TestTodayProviderErrorPartial(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	u, _ := r.teaching(c)
	r.pending(c, 1, time.Hour, "")
	logs := &bytes.Buffer{}
	svc := today.NewService(r.pool, nil, r.clk, slog.New(slog.NewTextHandler(logs, nil)), nil)
	svc.Agg.Register(boomProvider{})
	body, err := svc.Get(t.Context(), u.ID, today.RoleTeacher, today.Scope{})
	require.NoError(t, err)
	require.Contains(t, string(body), "JOIN_REQUEST", "phần còn lại vẫn trả")
	require.Contains(t, logs.String(), "provider=boom")
}

func TestTodayDeadline504(t *testing.T) {
	r := newRig(t)
	c := r.klass("761988")
	_, g := r.teaching(c)
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	hr := httptest.NewRequest(http.MethodGet, "/api/v1/me/today", nil).WithContext(ctx)
	hr.RemoteAddr = r.ip + ":4444"
	hr.Header.Set("Authorization", "Bearer "+g.access)
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, hr)
	require.Equal(t, http.StatusGatewayTimeout, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "DEADLINE_EXCEEDED")
}

type hangLLM struct{}

func (hangLLM) BudgetPercent(ctx context.Context) (int, bool)  { <-ctx.Done(); return 0, false }
func (hangLLM) OpenCircuit(ctx context.Context, _ string) bool { <-ctx.Done(); return false }

// QC L3 (US-P2-11): Redis chết ⇒ tín hiệu cổng AI treo; Admin vẫn phải thấy việc đọc từ DB (lớp không giảng viên, lời mời hết hạn).
func TestAdminTodaySurvivesHangingLLMSignals(t *testing.T) {
	r := newRig(t)
	r.klass("761995")
	r.llmProvider("Treo", true, new(bool))
	start := time.Now()
	items, err := (today.AdminProvider{Pool: r.pool, LLM: hangLLM{}}).Items(t.Context(), today.Viewer{Role: today.RoleAdmin, Now: r.clk.Now()}, today.Scope{})
	require.NoError(t, err)
	require.Less(t, time.Since(start), 140*time.Millisecond, "tín hiệu treo bị cắt ở hạn riêng, trước hạn 150 ms của Provider")
	var seen bool
	for _, it := range items {
		seen = seen || (it.Kind == today.KindCourseNoTeacher && it.Course.ClassCode == "761995")
	}
	require.True(t, seen, "việc từ DB vẫn còn")
	require.NotEmpty(t, find(items, today.KindLLMProviderError), "nhà cung cấp có last_test_ok=false vẫn báo lỗi")
}
