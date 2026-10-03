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

func (r *sessRig) scalar(sql string, args ...any) string {
	r.t.Helper()
	var s *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&s))
	if s == nil {
		return ""
	}
	return *s
}

// US-P2-02 AC1.
func TestLoginSuccess(t *testing.T) {
	r := newSessRig(t)
	email := uniq("sv.gioi")
	u := r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)

	res := r.do(req{path: "/auth/login", body: map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": rigPassword}, hdr: map[string]string{"Origin": rigOrigin}})
	require.Equal(t, http.StatusOK, res.code, string(res.body))

	c := res.cookie("ep_rt")
	require.NotNil(t, c)
	require.Regexp(t, `^[A-Za-z0-9_-]{43}$`, c.Value)
	require.True(t, c.HttpOnly)
	require.True(t, c.Secure)
	require.Equal(t, http.SameSiteLaxMode, c.SameSite)
	require.Equal(t, "/api/v1/auth", c.Path)
	require.Equal(t, 1209600, c.MaxAge)
	require.Empty(t, c.Domain)
	require.Equal(t, "no-store", res.hdr.Get("Cache-Control"))

	body := res.json()
	require.Equal(t, "Bearer", body["token_type"])
	require.EqualValues(t, 900, body["expires_in"])
	require.NotContains(t, string(res.body), c.Value, "thân phản hồi không chứa refresh token")
	user := body["user"].(map[string]any)
	require.Equal(t, u.ID.String(), user["id"])
	require.Equal(t, email, user["email"])
	require.Equal(t, "STUDENT", user["role"])
	require.Equal(t, "ACTIVE", user["status"])
	require.Equal(t, false, user["email_verified"])

	cl := claims(t, body["access_token"].(string))
	for _, k := range []string{"sub", "role", "email", "jti", "iat", "nbf", "exp", "iss", "aud", "sid"} {
		require.Contains(t, cl, k)
	}
	require.EqualValues(t, 900, cl["exp"].(float64)-cl["iat"].(float64))
	require.Equal(t, "edupilot", cl["iss"])
	require.Equal(t, "edupilot-api", cl["aud"])

	require.Equal(t, "1", r.scalar(`select count(*)::text from auth_sessions where id = $1 and user_id = $2 and device_label = 'Chrome trên macOS' and host(ip) = '203.0.113.7'`, cl["sid"], u.ID))
	require.NotEmpty(t, r.scalar(`select last_login_at::text from users where id = $1`, u.ID))
	require.Equal(t, "1", r.scalar(`select count(*)::text from login_attempts where user_id = $1 and outcome = 'SUCCESS'`, u.ID))
}

// US-P2-02 AC2: ba tình huống thất bại trả y hệt nhau.
func TestLoginUniformFailure(t *testing.T) {
	r := newSessRig(t)
	known := uniq("known")
	r.user(known, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	invited := uniq("invited")
	_, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: invited, FullName: "Mời", Role: store.UserRoleTEACHER, Status: store.UserStatusINVITED})
	require.NoError(t, err)

	var shapes []string
	for _, c := range []struct{ email, pw string }{{known, "sai-mat-khau-1"}, {uniq("khong.co"), "sai-mat-khau-1"}, {invited, rigPassword}} {
		res := r.login(c.email, c.pw)
		require.Equal(t, http.StatusUnauthorized, res.code)
		m := res.json()
		require.Equal(t, "INVALID_CREDENTIALS", m["code"])
		require.Equal(t, "Email hoặc mật khẩu không đúng.", m["message"])
		require.NotContains(t, m, "details")
		delete(m, "trace_id")
		keys := make([]string, 0, len(m))
		for k, v := range m {
			keys = append(keys, k+"="+v.(string))
		}
		sort.Strings(keys)
		shapes = append(shapes, strings.Join(keys, "|"))
	}
	require.Equal(t, shapes[0], shapes[1])
	require.Equal(t, shapes[0], shapes[2])
	require.Empty(t, r.cookieAfterFailure(known), "thất bại không phát cookie")
}

func (r *sessRig) cookieAfterFailure(email string) string {
	if c := r.login(email, "sai-mat-khau-1").cookie("ep_rt"); c != nil {
		return c.Value
	}
	return ""
}

// US-P2-02 AC2: tài khoản DISABLED: mật khẩu đúng → 403; sai → vẫn INVALID_CREDENTIALS.
func TestLoginDisabledAccount(t *testing.T) {
	r := newSessRig(t)
	email := uniq("off")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusDISABLED)
	res := r.login(email, rigPassword)
	require.Equal(t, http.StatusForbidden, res.code)
	require.Equal(t, "ACCOUNT_DISABLED", res.errCode())
	require.Nil(t, res.cookie("ep_rt"))
	require.Equal(t, "INVALID_CREDENTIALS", r.login(email, "sai-mat-khau-1").errCode())
}

// US-P2-02 AC2: email không tồn tại không nhanh hơn mật khẩu sai quá 35% (luôn một phép bcrypt).
func TestLoginTimingEqualized(t *testing.T) {
	if testing.Short() {
		t.Skip("đo thời gian")
	}
	r := newSessRig(t, func(e map[string]string) { e["BCRYPT_COST"] = "10" })
	hash, err := authHash(10)
	require.NoError(t, err)
	email := uniq("timing")
	_, err = store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: email, FullName: "T", Role: store.UserRoleSTUDENT, Status: store.UserStatusACTIVE, PasswordHash: &hash})
	require.NoError(t, err)

	median := func(e string) time.Duration {
		r.login(e, "sai-mat-khau-1") // làm nóng
		ds := make([]time.Duration, 20)
		for i := range ds {
			t0 := time.Now()
			require.Equal(t, http.StatusUnauthorized, r.login(e, "sai-mat-khau-1").code)
			ds[i] = time.Since(t0)
		}
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		return ds[len(ds)/2]
	}
	wrong, unknown := median(email), median(uniq("khong.co"))
	require.GreaterOrEqual(t, float64(unknown), 0.65*float64(wrong), "unknown=%v wrong=%v", unknown, wrong)
}

// US-P2-02 AC3.
func TestRefreshRotates(t *testing.T) {
	r := newSessRig(t)
	email := uniq("rot")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s1 := r.mustLogin(email)

	res := r.refresh(s1.rt)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	s2 := sessionOf(t, res)
	require.Equal(t, s1.sid, s2.sid)
	require.NotEqual(t, s1.rt, s2.rt)
	c1, c2 := claims(t, s1.access), claims(t, s2.access)
	require.NotEqual(t, c1["jti"], c2["jti"])
	require.Equal(t, "1", r.scalar(`select count(*)::text from auth_sessions where id = $1 and rotated_at is not null and prev_refresh_hash is not null`, s1.sid))
	require.Equal(t, http.StatusNotFound, r.api(s2.access).code, "access mới dùng được")
}

// US-P2-02 AC3: hạn trượt thêm REFRESH_TOKEN_TTL mỗi lần dùng.
func TestRefreshSliding(t *testing.T) {
	r := newSessRig(t)
	email := uniq("slide")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	r.clk.Advance(24 * time.Hour)
	res := r.refresh(s.rt)
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, 1209600, res.cookie("ep_rt").MaxAge, "mỗi lần xoay Max-Age đặt lại 14 ngày")
	var exp time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select expires_at from auth_sessions where id = $1`, s.sid).Scan(&exp))
	require.WithinDuration(t, r.clk.Now().Add(336*time.Hour), exp, time.Second)
}

// US-P2-02 AC3/AC5: không vượt hạn tuyệt đối 30 ngày; quá hạn đó thì hết.
func TestRefreshAbsoluteCap(t *testing.T) {
	r := newSessRig(t)
	email := uniq("abs")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	start := r.clk.Now()
	s := r.mustLogin(email)
	for range 3 { // 13 ngày mỗi bước → ngày 39 nếu không chặn
		r.clk.Advance(13 * 24 * time.Hour)
		if r.clk.Now().After(start.Add(30 * 24 * time.Hour)) {
			break
		}
		res := r.refresh(s.rt)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		s = sessionOf(t, res)
		remaining := start.Add(30 * 24 * time.Hour).Sub(r.clk.Now())
		require.LessOrEqual(t, time.Duration(res.cookie("ep_rt").MaxAge)*time.Second, remaining+time.Second)
	}
	var exp, abs time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select expires_at, absolute_expires_at from auth_sessions where id = $1`, s.sid).Scan(&exp, &abs))
	require.False(t, exp.After(abs))
	require.WithinDuration(t, start.Add(30*24*time.Hour), abs, time.Second)
}

func TestRefreshAbsoluteExpired(t *testing.T) {
	r := newSessRig(t)
	email := uniq("absx")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	r.clk.Advance(13 * 24 * time.Hour)
	s = sessionOf(t, r.refresh(s.rt))
	r.clk.Advance(13 * 24 * time.Hour)
	s = sessionOf(t, r.refresh(s.rt))
	r.clk.Advance(5 * 24 * time.Hour) // ngày 31 > 30
	res := r.refresh(s.rt)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "TOKEN_INVALID", res.errCode())
	require.EqualValues(t, -1, res.cookie("ep_rt").MaxAge)
}

// US-P2-02 AC3: DB chỉ có băm, không có bản rõ ở cột hay log.
func TestRefreshStoresHashOnly(t *testing.T) {
	r := newSessRig(t)
	email := uniq("hash")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s1 := r.mustLogin(email)
	res := r.refresh(s1.rt)
	s2 := sessionOf(t, res)
	for _, plain := range []string{s1.rt, s2.rt, rigPassword} {
		require.Zero(t, countRows(t, r, `select count(*) from auth_sessions where refresh_hash = $1 or prev_refresh_hash = $1 or user_agent like '%'||$1||'%'`, plain))
		require.NotContains(t, r.logs.String(), plain)
	}
	require.Equal(t, "1", r.scalar(`select count(*)::text from auth_sessions where refresh_hash = $1 and prev_refresh_hash = $2`, hashOf(s2.rt), hashOf(s1.rt)))
}

// US-P2-02 AC4: kẻ trộm dùng lại token cũ sau khi nạn nhân đã xoay.
func TestRefreshReuseRevokesSession(t *testing.T) {
	r := newSessRig(t)
	email := uniq("reuse")
	u := r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	t1 := r.mustLogin(email)
	t2 := sessionOf(t, r.refresh(t1.rt)) // nạn nhân xoay trước

	res := r.refresh(t1.rt) // kẻ trộm trình T1
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	require.EqualValues(t, -1, res.cookie("ep_rt").MaxAge, "xoá cookie")
	require.Equal(t, "refresh_reuse", res.json()["details"].(map[string]any)["reason"])
	require.Equal(t, "REFRESH_REUSE", r.scalar(`select revoked_reason from auth_sessions where id = $1`, t1.sid))

	require.Equal(t, "SESSION_REVOKED", r.refresh(t2.rt).errCode(), "T2 cũng vô hiệu")
	api := r.api(t2.access)
	require.Equal(t, http.StatusUnauthorized, api.code, "access còn hạn của phiên bị từ chối")
	require.Equal(t, "SESSION_REVOKED", api.errCode())
	require.Equal(t, 1, countRows(t, r, `select count(*) from audit_log where entity = 'auth_session' and action = 'refresh_reuse' and entity_id = $1 and actor_id = $2`, t1.sid, u.ID))
}

// Nạn nhân giữ T1, kẻ trộm xoay trước rồi nạn nhân trình T1.
func TestRefreshReuseThiefFirst(t *testing.T) {
	r := newSessRig(t)
	email := uniq("thief")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	t1 := r.mustLogin(email)
	thief := sessionOf(t, r.refresh(t1.rt))
	res := r.refresh(t1.rt)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	require.Equal(t, "SESSION_REVOKED", r.refresh(thief.rt).errCode(), "token vừa phát cho kẻ trộm cũng chết")
}

func TestRefreshReuseVictimFirst(t *testing.T) {
	r := newSessRig(t)
	email := uniq("victim")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	t1 := r.mustLogin(email)
	victim := sessionOf(t, r.refresh(t1.rt))
	require.Equal(t, http.StatusUnauthorized, r.refresh(t1.rt).code)
	require.Equal(t, http.StatusUnauthorized, r.refresh(victim.rt).code)
	require.Equal(t, http.StatusUnauthorized, r.api(victim.access).code)
}

func TestRefreshReuseAudit(t *testing.T) {
	r := newSessRig(t)
	email := uniq("audit")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	t1 := r.mustLogin(email)
	t2 := sessionOf(t, r.refresh(t1.rt))
	r.refresh(t1.rt)
	var before, after *string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select before::text, after::text from audit_log where entity = 'auth_session' and entity_id = $1`, t1.sid).Scan(&before, &after))
	require.Nil(t, before)
	require.Nil(t, after)
	for _, plain := range []string{t1.rt, t2.rt} {
		require.Zero(t, countRows(t, r, `select count(*) from audit_log where entity_id like '%'||$1||'%' or coalesce(trace_id,'') like '%'||$1||'%'`, plain))
	}
}

// AC4 (làm rõ Q-QC-P202-1): hai refresh song song cùng cookie → một 200, một 401, rồi cả phiên chết.
func TestRefreshConcurrentSameToken(t *testing.T) {
	r := newSessRig(t)
	email := uniq("race")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	var wg sync.WaitGroup
	out := make([]resp, 2)
	for i := range out {
		wg.Add(1)
		go func() { defer wg.Done(); out[i] = r.refresh(s.rt) }()
	}
	wg.Wait()
	codes := []int{out[0].code, out[1].code}
	sort.Ints(codes)
	require.Equal(t, []int{http.StatusOK, http.StatusUnauthorized}, codes)
	for _, o := range out {
		if o.code == http.StatusOK {
			require.Equal(t, "SESSION_REVOKED", r.refresh(sessionOf(t, o).rt).errCode(), "token vừa phát cũng chết")
		}
	}
	require.Equal(t, "REFRESH_REUSE", r.scalar(`select revoked_reason from auth_sessions where id = $1`, s.sid))
}

// AC5.
func TestRefreshMissing(t *testing.T) {
	r := newSessRig(t)
	res := r.refresh("")
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "UNAUTHENTICATED", res.errCode())
	require.EqualValues(t, -1, res.cookie("ep_rt").MaxAge)
}

func TestRefreshUnknown(t *testing.T) {
	r := newSessRig(t)
	email := uniq("unk")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	res := r.refresh("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	require.Equal(t, "TOKEN_INVALID", res.errCode())
	require.EqualValues(t, -1, res.cookie("ep_rt").MaxAge)
	require.Equal(t, http.StatusNotFound, r.api(s.access).code, "token lạ không thu hồi gì")
	require.Equal(t, 1, countRows(t, r, `select count(*) from auth_sessions where id = $1 and revoked_at is null`, s.sid))
}

func TestRefreshExpired(t *testing.T) {
	r := newSessRig(t)
	email := uniq("exp")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	r.clk.Advance(15 * 24 * time.Hour)
	res := r.refresh(s.rt)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "TOKEN_INVALID", res.errCode())
	require.Equal(t, 1, countRows(t, r, `select count(*) from auth_sessions where user_id = (select id from users where email = $1)`, email), "không tạo phiên mới")
}

func TestRefreshDisabledUser(t *testing.T) {
	r := newSessRig(t)
	email := uniq("dis")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	_, err := r.pool.Exec(t.Context(), `update users set status = 'DISABLED' where email = $1`, email)
	require.NoError(t, err)
	res := r.refresh(s.rt)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	require.Equal(t, "ACCOUNT_DISABLED", r.scalar(`select revoked_reason from auth_sessions where id = $1`, s.sid))
}

// AC6.
func TestLogout(t *testing.T) {
	r := newSessRig(t)
	email := uniq("out")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	require.Equal(t, http.StatusNotFound, r.api(s.access).code)
	res := r.logout(s.rt)
	require.Equal(t, http.StatusNoContent, res.code)
	require.EqualValues(t, -1, res.cookie("ep_rt").MaxAge)
	require.Equal(t, "LOGOUT", r.scalar(`select revoked_reason from auth_sessions where id = $1`, s.sid))
	api := r.api(s.access)
	require.Equal(t, http.StatusUnauthorized, api.code)
	require.Equal(t, "SESSION_REVOKED", api.errCode())
	require.Equal(t, "logout", api.json()["details"].(map[string]any)["reason"])
	require.Equal(t, "SESSION_REVOKED", r.refresh(s.rt).errCode())
}

func TestLogoutIdempotent(t *testing.T) {
	r := newSessRig(t)
	email := uniq("out2")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	require.Equal(t, http.StatusNoContent, r.logout(s.rt).code)
	require.Equal(t, http.StatusNoContent, r.logout(s.rt).code)
	require.Equal(t, http.StatusNoContent, r.logout("").code, "không còn cookie vẫn 204")
	require.Equal(t, http.StatusNoContent, r.logout("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA").code)
}

func TestLogoutOtherDeviceUnaffected(t *testing.T) {
	r := newSessRig(t)
	email := uniq("two")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	a, b := r.mustLogin(email), r.mustLogin(email)
	require.NotEqual(t, a.sid, b.sid)
	require.Equal(t, http.StatusNoContent, r.logout(a.rt).code)
	require.Equal(t, http.StatusNotFound, r.api(b.access).code)
	require.Equal(t, http.StatusOK, r.refresh(b.rt).code)
}

// AC7.
func TestCSRFOriginRejected(t *testing.T) {
	r := newSessRig(t)
	email := uniq("csrf")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	for _, path := range []string{"/auth/refresh", "/auth/logout"} {
		for _, rt := range []string{s.rt, ""} {
			res := r.do(req{path: path, rt: rt, hdr: map[string]string{"Origin": "https://evil.example"}})
			require.Equal(t, http.StatusForbidden, res.code, path)
			require.Equal(t, "FORBIDDEN", res.errCode())
			require.Equal(t, "origin", res.json()["details"].(map[string]any)["reason"])
			require.Nil(t, res.cookie("ep_rt"), "không xoay / xoá cookie")
		}
	}
	require.Equal(t, http.StatusOK, r.refresh(s.rt).code, "token chưa bị xoay bởi lần giả")
}

func TestCSRFSecFetchSite(t *testing.T) {
	r := newSessRig(t)
	email := uniq("sfs")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	for _, site := range []string{"cross-site", "same-site"} {
		res := r.do(req{path: "/auth/refresh", rt: s.rt, hdr: map[string]string{"Sec-Fetch-Site": site}})
		require.Equal(t, http.StatusForbidden, res.code, site)
	}
	for _, site := range []string{"same-origin", "none", ""} {
		res := r.do(req{path: "/auth/refresh", rt: s.rt, hdr: map[string]string{"Sec-Fetch-Site": site}})
		require.Equal(t, http.StatusOK, res.code, site)
		s = sessionOf(t, res)
	}
}

func TestCSRFAllowedOrigin(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["CORS_ORIGINS"] = "https://localhost,https://app.edupilot.example" })
	email := uniq("ok")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	for _, o := range []string{"https://app.edupilot.example", "https://localhost"} {
		res := r.do(req{path: "/auth/refresh", rt: s.rt, hdr: map[string]string{"Origin": o}})
		require.Equal(t, http.StatusOK, res.code, o)
		s = sessionOf(t, res)
	}
}

func TestCookieEndpointsPostJSONOnly(t *testing.T) {
	r := newSessRig(t)
	email := uniq("ct")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	for _, path := range []string{"/auth/refresh", "/auth/logout"} {
		for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			require.Equal(t, http.StatusMethodNotAllowed, r.do(req{method: m, path: path, rt: s.rt, hdr: map[string]string{"Origin": rigOrigin}}).code, m+" "+path)
		}
		for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", "multipart/form-data"} {
			res := r.do(req{path: path, rt: s.rt, hdr: map[string]string{"Origin": rigOrigin, "Content-Type": ct}})
			require.Equal(t, http.StatusUnsupportedMediaType, res.code, ct)
		}
	}
	require.Equal(t, http.StatusOK, r.refresh(s.rt).code, "không lần nào ở trên xoay token")
}

// AC8.
func TestRevokedSidRejected(t *testing.T) {
	r := newSessRig(t)
	email := uniq("rev")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	a, b := r.mustLogin(email), r.mustLogin(email)
	_, err := r.pool.Exec(t.Context(), `update auth_sessions set revoked_at = now(), revoked_reason = 'ADMIN' where id = $1`, a.sid)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.api(a.access).code, "chưa có khoá Redis → chưa bị chặn")
	require.NoError(t, r.rdb.Set(t.Context(), "ep:auth:rev:sid:"+a.sid, "1", time.Minute).Err())
	res := r.api(a.access)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	require.Equal(t, "admin", res.json()["details"].(map[string]any)["reason"])
	require.Equal(t, http.StatusNotFound, r.api(b.access).code, "phiên khác không ảnh hưởng")
}

func TestUserCutoffRejected(t *testing.T) {
	r := newSessRig(t)
	email := uniq("cut")
	u := r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	old := r.mustLogin(email)
	r.clk.Advance(10 * time.Second)
	require.NoError(t, r.rdb.Set(t.Context(), "ep:auth:rev:user:"+u.ID.String(), r.clk.Now().UnixMilli(), time.Minute).Err())
	res := r.api(old.access)
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "SESSION_REVOKED", res.errCode())
	fresh := r.mustLogin(email) // đăng nhập SAU mốc vẫn dùng được (kể cả trong cùng giây)
	require.Equal(t, http.StatusNotFound, r.api(fresh.access).code)
}

func TestRedisDownFailsOpen(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["REDIS_URL"] = "redis://127.0.0.1:1/0" })
	email := uniq("down")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	require.Equal(t, http.StatusNotFound, r.api(s.access).code, "Redis chết → chấp nhận token")
	require.Equal(t, http.StatusNoContent, r.logout(s.rt).code)
	require.Contains(t, r.logs.String(), "level=ERROR")
	require.Equal(t, "LOGOUT", r.scalar(`select revoked_reason from auth_sessions where id = $1`, s.sid), "DB vẫn là nguồn sự thật")
}

func TestDevTokenNoSid(t *testing.T) {
	r := newSessRig(t)
	tok := r.devToken(uuid.NewString())
	require.Empty(t, claims(t, tok)["sid"])
	require.Equal(t, http.StatusNotFound, r.api(tok).code, "token dev dùng được ở APP_ENV ≠ production")
}

func TestDevTokenRejectedInProduction(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["APP_ENV"] = "production" })
	res := r.api(r.devToken(uuid.NewString()))
	require.Equal(t, http.StatusUnauthorized, res.code)
	require.Equal(t, "TOKEN_INVALID", res.errCode())
	email := uniq("prod")
	r.user(email, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(email)
	require.Equal(t, http.StatusNotFound, r.api(s.access).code, "token phiên thật vẫn dùng được")
}

func countRows(t *testing.T, r *sessRig, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), sql, args...).Scan(&n))
	return n
}
