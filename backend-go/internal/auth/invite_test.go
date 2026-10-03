package auth_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *sessRig) accept(token, pw string) resp {
	return r.do(req{path: "/auth/accept-invite", body: map[string]string{"token": token, "password": pw}})
}

// invited tạo giảng viên INVITED (chưa mật khẩu) kèm token mời 72 giờ.
func (r *sessRig) invited(role store.UserRole) (store.User, string) {
	r.t.Helper()
	u, err := store.New(r.pool).InsertUser(r.t.Context(), store.InsertUserParams{Email: uniq("moi"), FullName: "Thầy Được Mời", Role: role, Status: store.UserStatusINVITED})
	require.NoError(r.t, err)
	return u, r.inviteToken(u.ID)
}

const invitePW = "Mat-khau-nhan-moi-2026"

// US-P2-06 AC2.
func TestAcceptInvite(t *testing.T) {
	r := newSessRig(t)
	u, tok := r.invited(store.UserRoleTEACHER)
	res := r.accept(tok, invitePW)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	user := res.json()["user"].(map[string]any)
	require.Equal(t, "TEACHER", user["role"])
	require.Equal(t, "ACTIVE", user["status"])
	require.Equal(t, true, user["email_verified"])
	c := res.cookie("ep_rt")
	require.NotNil(t, c, "phiên mới + cookie như đăng nhập")
	require.Equal(t, http.StatusNotFound, r.api(res.json()["access_token"].(string)).code, "access token dùng được")
	require.Equal(t, "ACTIVE", r.scalar(`select status::text from users where id = $1`, u.ID))
	require.NotEmpty(t, r.scalar(`select email_verified_at::text from users where id = $1`, u.ID))
	require.NotEmpty(t, r.scalar(`select used_at::text from auth_tokens where token_hash = $1`, hashOf(tok)))
	require.Equal(t, http.StatusOK, r.login(u.Email, invitePW).code, "đăng nhập được bằng mật khẩu vừa đặt")
}

func TestAcceptInviteReuse(t *testing.T) {
	r := newSessRig(t)
	_, tok := r.invited(store.UserRoleTA)
	require.Equal(t, http.StatusOK, r.accept(tok, invitePW).code)
	again := r.accept(tok, invitePW)
	require.Equal(t, http.StatusGone, again.code)
	require.Equal(t, "LINK_INVALID", again.errCode())
	require.Equal(t, "used", again.json()["details"].(map[string]any)["reason"])
}

func TestAcceptInviteExpired72h(t *testing.T) {
	r := newSessRig(t)
	u, tok := r.invited(store.UserRoleTEACHER)
	r.clk.Advance(72*time.Hour + time.Second)
	res := r.accept(tok, invitePW)
	require.Equal(t, http.StatusGone, res.code)
	require.Equal(t, "expired", res.json()["details"].(map[string]any)["reason"])
	require.Equal(t, "INVITED", r.scalar(`select status::text from users where id = $1`, u.ID))
	unknown := r.accept("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", invitePW)
	require.Equal(t, http.StatusGone, unknown.code)
	require.Equal(t, "invalid", unknown.json()["details"].(map[string]any)["reason"])
}

func TestAcceptInviteRace(t *testing.T) {
	r := newSessRig(t)
	_, tok := r.invited(store.UserRoleTEACHER)
	var wg sync.WaitGroup
	codes := make([]int, 30)
	for i := range codes {
		wg.Add(1)
		go func() { defer wg.Done(); codes[i] = r.accept(tok, invitePW).code }()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == http.StatusOK {
			ok++
		} else {
			require.Equal(t, http.StatusGone, c)
		}
	}
	require.Equal(t, 1, ok)
}

func TestAcceptInviteWeakPassword(t *testing.T) {
	r := newSessRig(t)
	u, tok := r.invited(store.UserRoleTEACHER)
	for pw, code := range map[string]string{"ngan": "PASSWORD_TOO_SHORT", "matkhau12345": "PASSWORD_COMMON"} {
		res := r.accept(tok, pw)
		require.Equal(t, http.StatusUnprocessableEntity, res.code)
		require.Equal(t, code, firstDetailCode(res))
		require.NotContains(t, string(res.body), pw)
	}
	require.Equal(t, "INVITED", r.scalar(`select status::text from users where id = $1`, u.ID), "không đổi gì khi lỗi")
	require.Equal(t, http.StatusOK, r.accept(tok, invitePW).code, "token chưa bị tiêu")
}

func TestAcceptInviteOnlyWhileInvited(t *testing.T) {
	r := newSessRig(t)
	u, tok := r.invited(store.UserRoleTA)
	_, err := r.pool.Exec(t.Context(), `update users set status = 'DISABLED' where id = $1`, u.ID)
	require.NoError(t, err)
	res := r.accept(tok, invitePW)
	require.Equal(t, http.StatusGone, res.code, "tài khoản đã bị khoá không nhận lời mời")
	require.Equal(t, "DISABLED", r.scalar(`select status::text from users where id = $1`, u.ID))
	prev := r.preview("INVITE", tok)
	require.Equal(t, http.StatusGone, prev.code)
}
