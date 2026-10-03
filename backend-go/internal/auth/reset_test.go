package auth_test

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

const (
	forgotMsg   = "Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu."
	newPassword = "Mat-khau-moi-2026"
)

func (r *sessRig) forgot(email string) resp {
	return r.do(req{path: "/auth/forgot-password", body: map[string]string{"email": email}})
}

func (r *sessRig) reset(token, pw string) resp {
	return r.do(req{path: "/auth/reset-password", body: map[string]string{"token": token, "new_password": pw}})
}

// resetToken chạy forgot ⇒ consumer thật ⇒ trả token trong thư.
func (r *sessRig) resetToken(email string) string {
	r.t.Helper()
	require.Equal(r.t, http.StatusAccepted, r.forgot(email).code)
	m, tok := r.deliver(email)
	require.Equal(r.t, "Đặt lại mật khẩu EduPilot", m.Subject)
	require.NotEmpty(r.t, tok)
	return tok
}

func (r *sessRig) activeUser(prefix string) string {
	e := uniq(prefix)
	r.user(e, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	return e
}

// AC1.
func TestForgotUniform(t *testing.T) {
	r := newSessRig(t)
	active, pending := r.activeUser("fa"), uniq("fp")
	require.Equal(t, http.StatusAccepted, r.register(regBody(pending)).code)
	disabled := uniq("fd")
	r.user(disabled, store.UserRoleSTUDENT, store.UserStatusDISABLED)
	invited := uniq("fi")
	_, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: invited, FullName: "Mời", Role: store.UserRoleTA, Status: store.UserStatusINVITED})
	require.NoError(t, err)

	var bodies []string
	for _, e := range []string{active, pending, disabled, invited, uniq("khong.co")} {
		res := r.forgot(e)
		require.Equal(t, http.StatusAccepted, res.code, e)
		require.Equal(t, map[string]any{"message": forgotMsg}, res.json())
		require.Empty(t, res.hdr.Get("Set-Cookie"))
		bodies = append(bodies, string(res.body))
	}
	for _, b := range bodies[1:] {
		require.Equal(t, bodies[0], b)
	}
}

func TestForgotMailOnlyEligible(t *testing.T) {
	r := newSessRig(t)
	active, pending := r.activeUser("ma"), uniq("mp")
	require.Equal(t, http.StatusAccepted, r.register(regBody(pending)).code)
	disabled, invited, ghost := uniq("md"), uniq("mi"), uniq("mg")
	r.user(disabled, store.UserRoleSTUDENT, store.UserStatusDISABLED)
	_, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: invited, FullName: "Mời", Role: store.UserRoleTA, Status: store.UserStatusINVITED})
	require.NoError(t, err)
	for _, e := range []string{active, pending, disabled, invited, ghost} {
		require.Equal(t, http.StatusAccepted, r.forgot(e).code)
	}
	require.Equal(t, 1, r.mails(active, "reset_password"))
	require.Equal(t, 1, r.mails(pending, "reset_password"))
	for _, e := range []string{disabled, invited, ghost} {
		require.Zero(t, r.mails(e, ""), e)
	}
}

func TestForgotTimingEqualized(t *testing.T) {
	if testing.Short() {
		t.Skip("đo thời gian")
	}
	r := newSessRig(t)
	median := func(eligible bool) time.Duration {
		ds := make([]time.Duration, 15)
		for i := range ds {
			e := uniq("tm")
			if eligible {
				r.user(e, store.UserRoleSTUDENT, store.UserStatusACTIVE)
			}
			t0 := time.Now()
			require.Equal(t, http.StatusAccepted, r.forgot(e).code)
			ds[i] = time.Since(t0)
		}
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		return ds[len(ds)/2]
	}
	has, none := median(true), median(false)
	lo, hi := min(has, none), max(has, none)
	require.GreaterOrEqual(t, float64(lo), 0.65*float64(hi), "có tài khoản=%v không có=%v", has, none)
}

func TestForgotPerEmailLimit(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("lim")
	for range 3 {
		require.Equal(t, http.StatusAccepted, r.forgot(e).code)
	}
	res := r.forgot(e)
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "RATE_LIMITED", res.errCode())
	ghost := uniq("ghost")
	for range 3 {
		require.Equal(t, http.StatusAccepted, r.forgot(ghost).code)
	}
	require.Equal(t, http.StatusTooManyRequests, r.forgot(ghost).code, "email không tồn tại cũng bị giới hạn: không lộ tồn tại")
}

// AC2.
func TestResetSuccess(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("rs")
	tok := r.resetToken(e)
	res := r.reset(tok, newPassword)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, map[string]any{"status": "password_reset"}, res.json())
	require.Equal(t, http.StatusUnauthorized, r.login(e, rigPassword).code, "mật khẩu cũ hết dùng")
	require.Equal(t, http.StatusOK, r.login(e, newPassword).code)
	var used *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select used_at from auth_tokens where token_hash = $1`, hashOf(tok)).Scan(&used))
	require.NotNil(t, used)
	var hash string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select password_hash from users where email = $1`, e).Scan(&hash))
	require.True(t, strings.HasPrefix(hash, "$2"))

	again := r.reset(tok, newPassword)
	require.Equal(t, http.StatusGone, again.code)
	require.Equal(t, "used", again.json()["details"].(map[string]any)["reason"])

	m, _ := r.deliver(e)
	require.Equal(t, "Mật khẩu EduPilot của bạn đã được đổi", m.Subject)
	require.NotContains(t, m.Text, newPassword)
}

func TestResetRevokesAllSessions(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("rv")
	a, b := r.mustLogin(e), r.mustLogin(e)
	require.Equal(t, http.StatusNotFound, r.api(a.access).code)
	tok := r.resetToken(e)
	require.Equal(t, http.StatusOK, r.reset(tok, newPassword).code)
	for _, s := range []session{a, b} {
		res := r.api(s.access)
		require.Equal(t, http.StatusUnauthorized, res.code)
		require.Equal(t, "SESSION_REVOKED", res.errCode())
		require.Equal(t, "password_reset", res.json()["details"].(map[string]any)["reason"])
		ref := r.refresh(s.rt)
		require.Equal(t, http.StatusUnauthorized, ref.code)
		require.Equal(t, "SESSION_REVOKED", ref.errCode())
	}
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from auth_sessions s join users u on u.id = s.user_id where u.email = $1 and s.revoked_reason = 'PASSWORD_RESET'`, e).Scan(&n))
	require.Equal(t, 2, n)
}

func TestResetClearsLockout(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("lk")
	_, err := r.pool.Exec(t.Context(), `update users set failed_logins = 7, locked_until = now() + interval '15 minutes' where email = $1`, e)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.reset(r.resetToken(e), newPassword).code)
	var f int
	var lu *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select failed_logins, locked_until from users where email = $1`, e).Scan(&f, &lu))
	require.Zero(t, f)
	require.Nil(t, lu)
}

func TestResetVerifiesEmail(t *testing.T) {
	r := newSessRig(t)
	e := uniq("rve")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	require.Equal(t, http.StatusOK, r.reset(r.resetToken(e), newPassword).code)
	_, status, verified, _, _ := r.userRow(e)
	require.Equal(t, "ACTIVE", status)
	require.True(t, verified)
}

func TestResetNoAutoLogin(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("na")
	res := r.reset(r.resetToken(e), newPassword)
	require.Equal(t, http.StatusOK, res.code)
	require.Empty(t, res.hdr.Get("Set-Cookie"))
	require.NotContains(t, string(res.body), "access_token")
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from auth_sessions s join users u on u.id = s.user_id where u.email = $1`, e).Scan(&n))
	require.Zero(t, n)
}

func TestResetSingleUseRace(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("rr")
	tok := r.resetToken(e)
	var wg sync.WaitGroup
	codes := make([]int, 30)
	for i := range codes {
		wg.Add(1)
		go func() { defer wg.Done(); codes[i] = r.reset(tok, newPassword).code }()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		default:
			require.Equal(t, http.StatusGone, c)
		}
	}
	require.Equal(t, 1, ok)
}

// AC3.
func TestResetExpired30m(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("re")
	tok := r.resetToken(e)
	r.clk.Advance(30*time.Minute + time.Second)
	res := r.reset(tok, newPassword)
	require.Equal(t, http.StatusGone, res.code)
	require.Equal(t, "expired", res.json()["details"].(map[string]any)["reason"])
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code, "mật khẩu cũ còn nguyên")
}

func TestResetUnknownToken(t *testing.T) {
	r := newSessRig(t)
	res := r.reset("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", newPassword)
	require.Equal(t, http.StatusGone, res.code)
	require.Equal(t, "invalid", res.json()["details"].(map[string]any)["reason"])
	// token xác minh email không dùng để đặt lại mật khẩu
	e := uniq("wk")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	_, vt := r.deliver(e)
	wrong := r.reset(vt, newPassword)
	require.Equal(t, http.StatusGone, wrong.code)
	require.Equal(t, "invalid", wrong.json()["details"].(map[string]any)["reason"])
	require.Equal(t, http.StatusOK, r.verify(vt).code, "token xác minh còn nguyên")
}

func TestResetWeakPasswordKeepsToken(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("wp")
	tok := r.resetToken(e)
	for pw, code := range map[string]string{"ngan": "PASSWORD_TOO_SHORT", strings.Repeat("x", 73): "PASSWORD_TOO_LONG"} {
		res := r.reset(tok, pw)
		require.Equal(t, http.StatusUnprocessableEntity, res.code)
		require.Equal(t, "VALIDATION_FAILED", res.errCode())
		require.Equal(t, code, firstDetailCode(res))
	}
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code, "không đổi gì khi lỗi")
	require.Equal(t, http.StatusOK, r.reset(tok, newPassword).code, "token chưa bị tiêu")
}

// AC5.
func (r *sessRig) changePw(s session, cur, next string) resp {
	return r.do(req{path: "/me/password", bearer: s.access, body: map[string]string{"current_password": cur, "new_password": next}})
}

func TestChangePassword(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("cp")
	s := r.mustLogin(e)
	res := r.changePw(s, rigPassword, newPassword)
	require.Equal(t, http.StatusNoContent, res.code, string(res.body))
	require.Equal(t, http.StatusOK, r.login(e, newPassword).code)
	require.Equal(t, http.StatusUnauthorized, r.login(e, rigPassword).code)
	var n int
	var after string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*), coalesce(max(after::text), '') from audit_log where action = 'password_changed' and entity_id = (select id::text from users where email = $1)`, e).Scan(&n, &after))
	require.Equal(t, 1, n)
	require.NotContains(t, after, newPassword)
	require.NotContains(t, after, "$2")
	m, _ := r.deliver(e)
	require.Equal(t, "Mật khẩu EduPilot của bạn đã được đổi", m.Subject)
	require.NotContains(t, m.Text, newPassword)
	require.NotContains(t, m.Text, rigPassword)
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	r := newSessRig(t)
	s := r.mustLogin(r.activeUser("wc"))
	res := r.changePw(s, "sai-mat-khau-hien-tai", newPassword)
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Equal(t, "VALIDATION_FAILED", res.errCode())
	d := res.json()["details"].([]any)[0].(map[string]any)
	require.Equal(t, "current_password", d["field"])
	require.Equal(t, "WRONG_PASSWORD", d["code"])
}

func TestChangePasswordRateLimit(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("rl")
	s := r.mustLogin(e)
	for range 5 {
		require.Equal(t, http.StatusUnprocessableEntity, r.changePw(s, "sai-mat-khau-hien-tai", newPassword).code)
	}
	res := r.changePw(s, rigPassword, newPassword) // đúng mật khẩu cũng bị chặn khi đã quá 5 lần sai
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "RATE_LIMITED", res.errCode())
	ra := int(res.json()["retry_after"].(float64))
	require.GreaterOrEqual(t, ra, 1)
	require.LessOrEqual(t, ra, 600)
	r.clk.Advance(10 * time.Minute)
	require.Equal(t, http.StatusNoContent, r.changePw(r.mustLogin(e), rigPassword, newPassword).code, "sau cửa sổ 10 phút thử lại được")
}

func TestChangePasswordKeepsCurrentSession(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("kc")
	s1, s2 := r.mustLogin(e), r.mustLogin(e)
	require.Equal(t, http.StatusNoContent, r.changePw(s1, rigPassword, newPassword).code)
	require.Equal(t, http.StatusNotFound, r.api(s1.access).code, "phiên hiện tại còn sống")
	require.Equal(t, http.StatusOK, r.refresh(s1.rt).code)
	res := r.api(s2.access)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	require.Equal(t, "password_changed", res.json()["details"].(map[string]any)["reason"])
	require.Equal(t, http.StatusUnauthorized, r.refresh(s2.rt).code)
}

func TestChangePasswordSameAsOld(t *testing.T) {
	r := newSessRig(t)
	s := r.mustLogin(r.activeUser("so"))
	res := r.changePw(s, rigPassword, rigPassword)
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Equal(t, "PASSWORD_SAME_AS_OLD", firstDetailCode(res))
	weak := r.changePw(s, rigPassword, "ngan")
	require.Equal(t, http.StatusUnprocessableEntity, weak.code)
	require.Equal(t, "PASSWORD_TOO_SHORT", firstDetailCode(weak))
	require.Equal(t, http.StatusOK, r.login(r.activeUserEmailOf(s), rigPassword).code)
}

func (r *sessRig) activeUserEmailOf(s session) string {
	r.t.Helper()
	var e string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select u.email from auth_sessions a join users u on u.id = a.user_id where a.id = $1`, s.sid).Scan(&e))
	return e
}

// AC6, AC7.
func (r *sessRig) sessions(s session) resp {
	return r.do(req{method: http.MethodGet, path: "/me/sessions", bearer: s.access})
}

func TestSessionsListShape(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("sl")
	s1 := r.mustLogin(e)
	r.clk.Advance(time.Minute)
	s2 := r.mustLogin(e)
	res := r.sessions(s1)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	items := res.json()["items"].([]any)
	require.Len(t, items, 2)
	first := items[0].(map[string]any)
	require.Len(t, first, 6)
	for _, k := range []string{"id", "current", "device_label", "ip_masked", "created_at", "last_used_at"} {
		require.Contains(t, first, k)
	}
	require.Equal(t, s1.sid, first["id"], "phiên hiện tại đứng đầu dù mới hơn")
	require.Equal(t, true, first["current"])
	require.Equal(t, false, items[1].(map[string]any)["current"])
	require.Equal(t, s2.sid, items[1].(map[string]any)["id"])
	require.Equal(t, "Chrome trên macOS", first["device_label"])
	for _, banned := range []string{"refresh", "user_agent", "Mozilla"} {
		require.NotContains(t, string(res.body), banned)
	}
	require.Equal(t, http.StatusOK, r.sessions(s2).code)
}

func TestSessionsListOnlyMine(t *testing.T) {
	r := newSessRig(t)
	a, b := r.mustLogin(r.activeUser("la")), r.mustLogin(r.activeUser("lb"))
	r.mustLogin(r.activeUserEmailOf(a))
	for _, s := range []session{a, b} {
		for _, it := range r.sessions(s).json()["items"].([]any) {
			m := it.(map[string]any)
			if s.sid != b.sid {
				require.NotEqual(t, b.sid, m["id"])
			} else {
				require.Equal(t, b.sid, m["id"])
			}
		}
	}
	require.Len(t, r.sessions(b).json()["items"].([]any), 1)
	// phiên đã thu hồi / hết hạn không hiện
	require.Equal(t, http.StatusNoContent, r.do(req{method: http.MethodDelete, path: "/me/sessions/" + b.sid, bearer: b.access}).code)
}

func TestSessionsIPMasked(t *testing.T) {
	r := newSessRig(t)
	s := r.mustLogin(r.activeUser("ip"))
	res := r.sessions(s)
	parts := strings.Split(r.ip, ".")
	require.Equal(t, parts[0]+"."+parts[1]+".*.*", res.json()["items"].([]any)[0].(map[string]any)["ip_masked"])
	require.NotContains(t, string(res.body), r.ip)
}

func (r *sessRig) del(s session, path string) resp {
	return r.do(req{method: http.MethodDelete, path: path, bearer: s.access})
}

func TestRevokeOwnSession(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("ro")
	s1, s2 := r.mustLogin(e), r.mustLogin(e)
	require.Equal(t, http.StatusNoContent, r.del(s1, "/me/sessions/"+s2.sid).code)
	res := r.api(s2.access)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	require.Equal(t, "revoked_by_user", res.json()["details"].(map[string]any)["reason"])
	require.Equal(t, http.StatusUnauthorized, r.refresh(s2.rt).code)
	require.Equal(t, http.StatusNotFound, r.del(s1, "/me/sessions/"+s2.sid).code, "đã thu hồi rồi ⇒ 404")
	// xoá phiên hiện tại = đăng xuất (và xoá cookie)
	out := r.del(s1, "/me/sessions/"+s1.sid)
	require.Equal(t, http.StatusNoContent, out.code)
	c := out.cookie("ep_rt")
	require.NotNil(t, c)
	require.Equal(t, -1, c.MaxAge)
	require.Equal(t, http.StatusUnauthorized, r.api(s1.access).code)
}

func TestRevokeOthersSessionIs404(t *testing.T) {
	r := newSessRig(t)
	a, b := r.mustLogin(r.activeUser("xa")), r.mustLogin(r.activeUser("xb"))
	for _, id := range []string{b.sid, uuid.NewString(), "khong-phai-uuid"} {
		res := r.del(a, "/me/sessions/"+id)
		require.Equal(t, http.StatusNotFound, res.code, id)
		require.Equal(t, "NOT_FOUND", res.errCode())
	}
	require.Equal(t, http.StatusNotFound, r.api(b.access).code, "phiên của B vẫn sống")
}

func TestRevokeAllOthers(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("ra")
	s1, s2, s3 := r.mustLogin(e), r.mustLogin(e), r.mustLogin(e)
	res := r.del(s1, "/me/sessions")
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, float64(2), res.json()["revoked"])
	require.Equal(t, http.StatusNotFound, r.api(s1.access).code)
	for _, s := range []session{s2, s3} {
		require.Equal(t, http.StatusUnauthorized, r.api(s.access).code)
		require.Equal(t, http.StatusUnauthorized, r.refresh(s.rt).code)
	}
	require.Equal(t, float64(0), r.del(s1, "/me/sessions").json()["revoked"])
}

func TestAdminCannotRevokeViaMe(t *testing.T) {
	r := newSessRig(t)
	admin := r.user(uniq("ad"), store.UserRoleADMIN, store.UserStatusACTIVE)
	victim := r.mustLogin(r.activeUser("vt"))
	a := r.mustLogin(admin.Email)
	require.Equal(t, http.StatusNotFound, r.del(a, "/me/sessions/"+victim.sid).code)
	require.Equal(t, http.StatusNotFound, r.api(victim.access).code)
	for _, it := range r.sessions(a).json()["items"].([]any) {
		require.NotEqual(t, victim.sid, it.(map[string]any)["id"])
	}
}

// AC10.
func TestMeEndpointsRequireJWT(t *testing.T) {
	r := newSessRig(t)
	for _, c := range []struct{ m, p string }{{"GET", "/me/sessions"}, {"DELETE", "/me/sessions"}, {"DELETE", "/me/sessions/" + uuid.NewString()}, {"POST", "/me/password"}} {
		res := r.do(req{method: c.m, path: c.p, body: map[string]string{"current_password": "x", "new_password": "y"}})
		require.Equal(t, http.StatusUnauthorized, res.code, c.m+" "+c.p)
		require.Equal(t, "UNAUTHENTICATED", res.errCode())
	}
}

func TestMeEndpointsNoUserIDParam(t *testing.T) {
	r := newSessRig(t)
	a, b := r.mustLogin(r.activeUser("ua")), r.mustLogin(r.activeUser("ub"))
	bUser := r.activeUserEmailOf(b)
	var bid string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id::text from users where email = $1`, bUser).Scan(&bid))
	res := r.do(req{method: http.MethodGet, path: "/me/sessions?user_id=" + bid, bearer: a.access})
	require.Equal(t, http.StatusOK, res.code)
	for _, it := range res.json()["items"].([]any) {
		require.NotEqual(t, b.sid, it.(map[string]any)["id"], "tham số user_id bị bỏ qua")
	}
	res = r.do(req{path: "/me/password", bearer: a.access, body: map[string]string{"user_id": bid, "current_password": rigPassword, "new_password": newPassword}})
	require.Equal(t, http.StatusUnprocessableEntity, res.code, "trường user_id bị từ chối")
	require.Equal(t, http.StatusOK, r.login(bUser, rigPassword).code, "mật khẩu của B không đổi")
}

func TestAllRolesCanManageOwnSessions(t *testing.T) {
	r := newSessRig(t)
	for _, role := range []store.UserRole{store.UserRoleSTUDENT, store.UserRoleTA, store.UserRoleTEACHER, store.UserRoleADMIN} {
		e := uniq("role")
		r.user(e, role, store.UserStatusACTIVE)
		s1, s2 := r.mustLogin(e), r.mustLogin(e)
		require.Len(t, r.sessions(s1).json()["items"].([]any), 2, string(role))
		require.Equal(t, http.StatusNoContent, r.del(s1, "/me/sessions/"+s2.sid).code, string(role))
		require.Equal(t, http.StatusNoContent, r.changePw(s1, rigPassword, newPassword).code, string(role))
	}
}

// AC11.
func (r *sessRig) preview(kind, token string) resp {
	return r.do(req{path: "/auth/tokens/preview", body: map[string]string{"kind": kind, "token": token}})
}

func TestTokenPreviewValid(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("pv")
	tok := r.resetToken(e)
	res := r.preview("RESET_PASSWORD", tok)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	m := res.json()
	require.Equal(t, true, m["valid"])
	require.Equal(t, "RESET_PASSWORD", m["kind"])
	require.NotEmpty(t, m["expires_at"])
	require.NotContains(t, m, "full_name", "chỉ INVITE mới có tên / vai")
	require.NotContains(t, m, "role")

	inv := uniq("inv")
	u, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: inv, FullName: "Thầy Mời", Role: store.UserRoleTEACHER, Status: store.UserStatusINVITED})
	require.NoError(t, err)
	itok := r.inviteToken(u.ID)
	res = r.preview("INVITE", itok)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "Thầy Mời", res.json()["full_name"])
	require.Equal(t, "TEACHER", res.json()["role"])

	require.Equal(t, http.StatusUnprocessableEntity, r.preview("VERIFY_EMAIL", tok).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.preview("", tok).code)
	wrongKind := r.preview("INVITE", tok)
	require.Equal(t, http.StatusGone, wrongKind.code)
	require.Equal(t, "invalid", wrongKind.json()["details"].(map[string]any)["reason"])
	r.clk.Advance(31 * time.Minute)
	exp := r.preview("RESET_PASSWORD", tok)
	require.Equal(t, http.StatusGone, exp.code)
	require.Equal(t, "expired", exp.json()["details"].(map[string]any)["reason"])
}

func (r *sessRig) inviteToken(uid uuid.UUID) string {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), `insert into auth_tokens (user_id, kind, token_hash, expires_at) values ($1, 'INVITE', $2, $3)`, uid, hashOf("invite-token-"+uid.String()), r.clk.Now().Add(72*time.Hour))
	require.NoError(r.t, err)
	return "invite-token-" + uid.String()
}

func TestTokenPreviewDoesNotConsume(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("pd")
	tok := r.resetToken(e)
	for range 3 {
		require.Equal(t, http.StatusOK, r.preview("RESET_PASSWORD", tok).code)
	}
	require.Equal(t, http.StatusOK, r.reset(tok, newPassword).code, "token còn nguyên sau khi xem trước")
	used := r.preview("RESET_PASSWORD", tok)
	require.Equal(t, http.StatusGone, used.code)
	require.Equal(t, "used", used.json()["details"].(map[string]any)["reason"])
}

func TestTokenPreviewNoEmail(t *testing.T) {
	r := newSessRig(t)
	inv := uniq("pn")
	u, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: inv, FullName: "Cô TA", Role: store.UserRoleTA, Status: store.UserStatusINVITED})
	require.NoError(t, err)
	for _, res := range []resp{r.preview("INVITE", r.inviteToken(u.ID)), r.preview("INVITE", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")} {
		require.NotContains(t, string(res.body), "@", "không lộ email")
		require.NotContains(t, string(res.body), inv)
	}
}

func TestTokenPreviewRateLimit(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["AUTH_TOKEN_IP_PER_MIN"] = "20" })
	for i := range 20 {
		require.Equal(t, http.StatusGone, r.preview("INVITE", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA").code, i)
	}
	res := r.preview("INVITE", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "RATE_LIMITED", res.errCode())
	require.NotEmpty(t, res.hdr.Get("Retry-After"))
	r.clk.Advance(61 * time.Second)
	require.Equal(t, http.StatusGone, r.preview("INVITE", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA").code)
}
