package auth_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

const wrongPassword = "sai-mat-khau-1"

func (r *sessRig) bad(email string) resp { return r.login(email, wrongPassword) }

func retryAfterOf(res resp) int {
	f, _ := res.json()["retry_after"].(float64)
	return int(f)
}

// backoff là lịch chờ sau lần sai thứ n (5…9): 2, 4, 8, 16, 32 giây.
func backoff(n int) int { return 1 << (n - 4) }

// lock gây 10 lần sai liên tiếp, chờ đủ lịch giữa các lần (đồng hồ giả).
func (r *sessRig) lock(email string) {
	r.t.Helper()
	for n := 1; n <= 10; n++ {
		require.Equal(r.t, http.StatusUnauthorized, r.bad(email).code, "lần sai %d", n)
		if n >= 5 && n <= 9 {
			r.clk.Advance(time.Duration(backoff(n)) * time.Second)
		}
	}
}

func (r *sessRig) failKeyExists(email string) bool {
	r.t.Helper()
	n, err := r.rdb.Exists(r.t.Context(), appredis.Key("auth", "fail", auth.HashToken(email)[:32])).Result()
	require.NoError(r.t, err)
	return n == 1
}

// AC1.
func TestLockoutBackoffSchedule(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("bo")
	for n := 1; n <= 9; n++ {
		require.Equal(t, http.StatusUnauthorized, r.bad(e).code, "lần sai %d trả 401 bình thường", n)
		if n < 5 {
			continue
		}
		res := r.bad(e) // thử ngay trong lúc chờ
		require.Equal(t, http.StatusTooManyRequests, res.code, "sau lần sai %d", n)
		require.Equal(t, "LOGIN_THROTTLED", res.errCode())
		require.InDelta(t, backoff(n), retryAfterOf(res), 1, "chờ sau lần sai %d", n)
		r.clk.Advance(time.Duration(backoff(n)) * time.Second) // hết chờ ⇒ kiểm lại bình thường (vòng sau trả 401)
	}
}

func TestLockoutBackoffBlocksCorrectPassword(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("bc")
	for range 5 {
		require.Equal(t, http.StatusUnauthorized, r.bad(e).code)
		r.clk.Advance(0)
	}
	res := r.login(e, rigPassword)
	require.Equal(t, http.StatusTooManyRequests, res.code, "mật khẩu ĐÚNG vẫn bị chặn trong lúc chờ")
	require.Equal(t, "LOGIN_THROTTLED", res.errCode())
	require.Equal(t, "THROTTLED", r.scalar(`select outcome::text from login_attempts order by created_at desc, id desc limit 1`), "chờ ghi THROTTLED, không kiểm mật khẩu")
	r.clk.Advance(2 * time.Second)
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code)
}

func TestLockoutBackoffRetryAfterAccurate(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("ra")
	for range 5 {
		r.bad(e)
	}
	require.InDelta(t, 2, retryAfterOf(r.bad(e)), 1)
	r.clk.Advance(time.Second)
	res := r.login(e, rigPassword)
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.InDelta(t, 1, retryAfterOf(res), 1)
	require.NotEmpty(t, res.hdr.Get("Retry-After"))
}

// AC2.
func TestLockoutAfter10(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("l10")
	r.lock(e)
	require.Equal(t, "10", r.scalar(`select failed_logins::text from users where email = $1`, e))
	until := r.scalar(`select extract(epoch from locked_until)::bigint::text from users where email = $1`, e)
	require.Equal(t, fmt.Sprint(r.clk.Now().Add(15*time.Minute).Unix()), until)
}

func TestLockoutBlocksCorrectPassword(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("lb")
	r.lock(e)
	for _, pw := range []string{rigPassword, wrongPassword} {
		res := r.login(e, pw)
		require.Equal(t, http.StatusTooManyRequests, res.code)
		require.Equal(t, "LOGIN_THROTTLED", res.errCode(), "cùng mã với chờ tăng dần: không lộ trạng thái khoá")
		require.InDelta(t, 900, retryAfterOf(res), 1)
	}
	r.clk.Advance(5 * time.Minute)
	require.InDelta(t, 600, retryAfterOf(r.login(e, rigPassword)), 1)
	require.Equal(t, "LOCKED", r.scalar(`select outcome::text from login_attempts order by created_at desc, id desc limit 1`))
}

func TestLockoutMailOnce(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("lm")
	r.lock(e)
	for range 6 {
		r.bad(e)
	}
	require.Equal(t, 1, r.mails(e, "account_locked"), "một thư cho mỗi lần khoá, không thêm khi thử trong lúc khoá")
	m, _ := r.deliver(e)
	require.Equal(t, "Tài khoản EduPilot bị khoá tạm thời", m.Subject)
	require.Contains(t, m.Text, "15 phút")
	r.clk.Advance(15*time.Minute + time.Second)
	r.lock(e)
	require.Equal(t, 2, r.mails(e, "account_locked"), "khoá lần hai ⇒ thư thứ hai")
}

func TestLockoutExpires(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("lx")
	r.lock(e)
	r.clk.Advance(15*time.Minute + time.Second)
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code)
	require.Equal(t, "0", r.scalar(`select failed_logins::text from users where email = $1`, e))
	require.Empty(t, r.scalar(`select locked_until::text from users where email = $1`, e))
	require.False(t, r.failKeyExists(e))
}

// AC3.
func TestThrottleUniformForUnknownEmail(t *testing.T) {
	r := newSessRig(t)
	known, unknown := r.activeUser("known"), uniq("khong.co")
	shape := func(res resp) string {
		keys := []string{}
		for k := range res.json() {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		return fmt.Sprintf("%d %s %v retry=%d", res.code, res.errCode(), keys, retryAfterOf(res))
	}
	for step := 1; step <= 12; step++ {
		a, b := shape(r.bad(known)), shape(r.bad(unknown))
		require.Equal(t, a, b, "lần %d", step)
		if step >= 5 {
			r.clk.Advance(40 * time.Second) // vượt mọi lịch chờ (≤ 32 s) nhưng không vượt khoá 15 phút
		}
	}
	require.Equal(t, 1, r.mails(known, "account_locked"))
	require.Zero(t, r.mails(unknown, ""), "email lạ không có hộp thư để gửi")
}

// AC4.
func TestCounterResetsOnSuccess(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("cs")
	for range 3 {
		r.bad(e)
	}
	require.Equal(t, "3", r.scalar(`select failed_logins::text from users where email = $1`, e))
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code)
	require.Equal(t, "0", r.scalar(`select failed_logins::text from users where email = $1`, e))
	require.False(t, r.failKeyExists(e))
	for i := 1; i <= 4; i++ {
		require.Equal(t, http.StatusUnauthorized, r.bad(e).code, "sau đăng nhập đúng, lần sai %d không bị chờ", i)
	}
}

func TestCounterDecays30m(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("cd")
	for range 4 {
		r.bad(e)
	}
	r.clk.Advance(31 * time.Minute)
	require.Equal(t, http.StatusUnauthorized, r.bad(e).code)
	require.Equal(t, http.StatusUnauthorized, r.bad(e).code, "lần sai cũ hơn 30 phút không cộng dồn: đây mới là lần thứ 2")
	require.Equal(t, "2", r.scalar(`select failed_logins::text from users where email = $1`, e))
}

func TestResetUnlocks(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("ru")
	r.lock(e)
	require.Equal(t, http.StatusTooManyRequests, r.login(e, rigPassword).code)
	require.Equal(t, http.StatusOK, r.reset(r.resetToken(e), newPassword).code)
	require.Equal(t, http.StatusOK, r.login(e, newPassword).code, "đặt lại mật khẩu mở khoá ngay")
	require.Equal(t, "0", r.scalar(`select failed_logins::text from users where email = $1`, e))
}

// AC5: Redis không với tới (client đã đóng ⇒ mọi lệnh lỗi).
func TestLockoutWithoutRedis(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("nr")
	require.NoError(t, r.rdb.Close())
	for n := 1; n <= 10; n++ {
		require.Equal(t, http.StatusUnauthorized, r.bad(e).code, "lần sai %d", n)
	}
	require.Equal(t, "10", r.scalar(`select failed_logins::text from users where email = $1`, e))
	res := r.login(e, rigPassword)
	require.Equal(t, http.StatusTooManyRequests, res.code, "khoá bền ở DB dù Redis mất")
	require.Equal(t, "LOGIN_THROTTLED", res.errCode())
	require.InDelta(t, 900, retryAfterOf(res), 1)
	other := r.activeUser("ok")
	require.Equal(t, http.StatusOK, r.login(other, rigPassword).code, "đăng nhập đúng vẫn chạy khi Redis mất")
	for range 12 {
		require.Equal(t, http.StatusUnauthorized, r.bad(uniq("khong.co")).code, "email lạ không bị chờ (chỉ Redis giữ được)")
	}
	require.GreaterOrEqual(t, strings.Count(r.logs.String(), "level=ERROR"), 1)
	require.LessOrEqual(t, strings.Count(r.logs.String(), "Redis không với tới: bỏ qua kiểm thu hồi"), 1)
}

// AC9.
func TestLoginAttemptsRecorded(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("la")
	r.bad(e)
	r.bad(uniq("khong.co"))
	off := uniq("off")
	r.user(off, store.UserRoleSTUDENT, store.UserStatusDISABLED)
	r.login(off, rigPassword)
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code)
	locked := r.activeUser("lk")
	r.lock(locked)
	r.bad(locked) // LOCKED
	for range 5 { // chờ tăng dần ⇒ THROTTLED
		r.bad(e)
	}
	r.bad(e)
	for _, o := range []string{"SUCCESS", "BAD_PASSWORD", "UNKNOWN_EMAIL", "THROTTLED", "LOCKED", "DISABLED"} {
		require.NotEqual(t, "0", r.scalar(`select count(*)::text from login_attempts where outcome = $1::login_outcome`, o), o)
	}
	require.Equal(t, "0", r.scalar(`select count(*)::text from login_attempts where ip is null or created_at is null or length(email_hash) <> 64`))
}

func TestLoginAttemptsNoPlainEmail(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("pe")
	r.bad(e)
	require.Equal(t, http.StatusOK, r.login(e, rigPassword).code)
	require.Equal(t, "0", r.scalar(`select count(*)::text from login_attempts where email_hash like '%@%'`))
	require.Equal(t, "2", r.scalar(`select count(*)::text from login_attempts where email_hash = $1`, auth.HashToken(e)))
	// UA dài 300 ký tự bị cắt ≤ 200
	long := strings.Repeat("U", 300)
	r.do(req{path: "/auth/login", body: map[string]string{"email": e, "password": rigPassword}, hdr: map[string]string{"Origin": rigOrigin, "User-Agent": long}})
	require.Equal(t, "200", r.scalar(`select max(length(user_agent))::text from login_attempts`))
	for _, secret := range []string{wrongPassword, rigPassword, e} {
		require.Equal(t, "0", r.scalar(`select count(*)::text from login_attempts where user_agent like '%'||$1||'%'`, secret))
	}
}

// AC8.
func TestBcryptCost(t *testing.T) {
	r := newSessRig(t) // BCRYPT_COST=4
	e := r.activeUser("bcost")
	require.Contains(t, r.scalar(`select left(password_hash, 7) from users where email = $1`, e), "$2a$04$")
	require.Equal(t, http.StatusAccepted, r.register(regBody(uniq("bc"))).code)
}

func TestRejectOver72Bytes(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("ov")
	long := strings.Repeat("x", 73)
	require.Equal(t, http.StatusUnauthorized, r.login(e, long).code, "đăng nhập: 73 byte bị từ chối, không cắt im lặng, không lỗi 500")
	res := r.register(map[string]string{"email": uniq("ov"), "password": long, "full_name": "X"})
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Equal(t, "PASSWORD_TOO_LONG", firstDetailCode(res))
}

// AC6.
func TestRateLimitTable(t *testing.T) {
	const limit = 3
	cases := []struct {
		name, path string
		body       func() any
		hdr        map[string]string
	}{
		{"login", "/auth/login", func() any { return map[string]string{"email": uniq("x"), "password": wrongPassword} }, nil},
		{"register", "/auth/register", func() any { return regBody(uniq("rg")) }, nil},
		{"forgot-password", "/auth/forgot-password", func() any { return map[string]string{"email": uniq("f")} }, nil},
		{"verify-email", "/auth/verify-email", func() any { return map[string]string{"token": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} }, nil},
		{"reset-password", "/auth/reset-password", func() any {
			return map[string]string{"token": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "new_password": "Mat-khau-moi-2026"}
		}, nil},
		{"tokens/preview", "/auth/tokens/preview", func() any {
			return map[string]string{"kind": "INVITE", "token": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}
		}, nil},
		{"refresh", "/auth/refresh", func() any { return nil }, map[string]string{"Origin": rigOrigin}},
		{"resend-verification", "/auth/resend-verification", func() any { return map[string]string{"email": "cung.mot.email@a.bc"} }, nil},
	}
	require.GreaterOrEqual(t, len(cases), 8)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newSessRig(t, func(e map[string]string) {
				for _, k := range []string{"AUTH_LOGIN_IP_PER_MIN", "AUTH_REGISTER_IP_PER_HOUR", "AUTH_FORGOT_IP_PER_HOUR", "AUTH_TOKEN_IP_PER_MIN", "AUTH_REFRESH_IP_PER_MIN"} {
					e[k] = fmt.Sprint(limit)
				}
				e["AUTH_RESEND_SECONDS"] = "60"
			})
			var last resp
			for i := 1; i <= limit+1; i++ {
				last = r.do(req{path: c.path, body: c.body(), hdr: c.hdr})
				if c.name == "resend-verification" { // giới hạn theo băm email: lần 2 đã 429
					if i == 1 {
						require.Equal(t, http.StatusAccepted, last.code)
						continue
					}
					require.Equal(t, http.StatusTooManyRequests, last.code)
					require.Equal(t, "RATE_LIMITED", last.errCode())
					return
				}
				if i <= limit {
					require.NotEqual(t, http.StatusTooManyRequests, last.code, "lần %d chưa vượt", i)
				}
			}
			require.Equal(t, http.StatusTooManyRequests, last.code)
			require.Equal(t, "RATE_LIMITED", last.errCode())
			require.NotEmpty(t, last.hdr.Get("Retry-After"))
			require.GreaterOrEqual(t, retryAfterOf(last), 1)
		})
	}
}

func TestIPBlockAfter30Failures(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["AUTH_LOGIN_IP_FAIL_PER_15M"] = "30" })
	e := r.activeUser("ipb")
	for i := 1; i <= 30; i++ {
		require.Equal(t, http.StatusUnauthorized, r.bad(uniq("khong.co")).code, "lần sai %d", i)
	}
	res := r.login(e, rigPassword) // mật khẩu đúng của tài khoản khác, cùng IP
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "LOGIN_THROTTLED", res.errCode())
	require.InDelta(t, 900, retryAfterOf(res), 2)
	other := newSessRig(t)
	other.ip = "198.51.100.9"
	require.Equal(t, http.StatusOK, other.login(other.activeUser("vn"), rigPassword).code, "IP khác không bị chặn")
}

func TestRateLimitSharedAcrossInstances(t *testing.T) {
	opt := func(e map[string]string) { e["AUTH_LOGIN_IP_PER_MIN"] = "3" }
	a, b := newSessRig(t, opt), newSessRig(t, opt)
	b.ip = a.ip
	b.clk.Set(a.clk.Now()) // cùng cửa sổ phút
	for i, r := range []*sessRig{a, b, a} {
		require.NotEqual(t, http.StatusTooManyRequests, r.bad(uniq("x")).code, "lần %d", i+1)
	}
	require.Equal(t, http.StatusTooManyRequests, b.bad(uniq("x")).code, "bộ đếm chia sẻ qua Redis giữa hai bản gateway")
}

// AC11.
func TestForgedXForwardedForIgnored(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["AUTH_LOGIN_IP_PER_MIN"] = "10" })
	var last resp
	for i := 1; i <= 11; i++ {
		last = r.do(req{path: "/auth/login", body: map[string]string{"email": uniq("x"), "password": wrongPassword}, hdr: map[string]string{"X-Forwarded-For": fmt.Sprintf("1.2.3.%d", i)}})
	}
	require.Equal(t, http.StatusTooManyRequests, last.code, "X-Forwarded-For từ ngoài dải tin cậy không đổi IP bị tính")
	require.Equal(t, "RATE_LIMITED", last.errCode())
}

func TestAdminSubjectToLockout(t *testing.T) {
	r := newSessRig(t)
	admin := r.user(uniq("ad"), store.UserRoleADMIN, store.UserStatusACTIVE)
	r.lock(admin.Email)
	res := r.login(admin.Email, rigPassword)
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "LOGIN_THROTTLED", res.errCode())
}
