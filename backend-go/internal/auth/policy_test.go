package auth_test

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
)

// AC7.
func TestPasswordPolicyTable(t *testing.T) {
	const email = "nguyen.van@sv.example"
	cases := []struct{ name, pw, email, want string }{
		{"rỗng", "", email, auth.PasswordTooShort},
		{"ngắn", "abc", email, auth.PasswordTooShort},
		{"9 ký tự", "123456789", email, auth.PasswordTooShort},
		{"9 chữ Việt có dấu", "ặếộửữầẳẵẫ", email, auth.PasswordTooShort},
		{"10 chữ Việt có dấu", "ặếộửữầẳẵẫẩ", email, ""},
		{"72 byte", strings.Repeat("ab1", 24), email, ""},
		{"73 byte", strings.Repeat("ab1", 24) + "x", email, auth.PasswordTooLong},
		{"1234567890", "1234567890", email, auth.PasswordCommon},
		{"matkhau12345", "matkhau12345", email, auth.PasswordCommon},
		{"MatKhau12345 (hoa thường)", "MatKhau12345", email, auth.PasswordCommon},
		{"Mậtkhẩu12345 (bỏ dấu)", "Mậtkhẩu12345", email, auth.PasswordCommon},
		{"MẬTKHẨU12345", "MẬTKHẨU12345", email, auth.PasswordCommon},
		{"dạng NFD", "Ma\u0323\u0302tkha\u0309\u0302u12345", email, auth.PasswordCommon},
		{"đ → d, bỏ dấu", "đạiHọc2026", "", auth.PasswordCommon},
		{"qwertyuiop", "qwertyuiop", email, auth.PasswordCommon},
		{"p@ssw0rd123", "P@ssw0rd123", email, auth.PasswordCommon},
		{"Passw0rd!!! không trong danh sách", "Passw0rd!!!", email, ""},
		{"chứa phần đầu email", "Nguyen.Van-2026!", email, auth.PasswordContainsEmail},
		{"chứa phần đầu email, hoa thường", "xxNGUYEN.VANxx!", email, auth.PasswordContainsEmail},
		{"phần đầu email quá ngắn (<4) không tính", "kab-lop-9900", "kab@sv.example", ""},
		{"một ký tự lặp", "kkkkkkkkkk", email, auth.PasswordLowEntropy},
		{"dãy chữ tăng", "mnopqrstuv", email, auth.PasswordLowEntropy},
		{"dãy số tăng vòng qua 9→0", "3456789012", email, auth.PasswordLowEntropy},
		{"10 dấu cách", strings.Repeat(" ", 10), email, auth.PasswordLowEntropy},
		{"mật khẩu mẫu của dự án", "Edupilot#2026-demo", email, ""},
		{"có gạch nối", "Mat-khau-moi-2026", email, ""},
		{"cụm từ dài", "correct horse battery staple", email, ""},
		{"khoảng trắng được giữ (không cắt)", "  matkhau12345  ", email, ""},
		{"tiếng Việt dài", "trường học 2026 đẹp", email, ""},
		{"không bắt buộc hoa / số / ký hiệu", "chuthuongthoidaidai", email, ""},
	}
	require.GreaterOrEqual(t, len(cases), 25)
	for _, c := range cases {
		require.Equal(t, c.want, auth.ValidatePasswordPolicy(c.pw, c.email), c.name)
	}
}

func TestPasswordPolicyAllEntryPoints(t *testing.T) {
	r := newSessRig(t)
	reject := func(res resp, code, via string) {
		t.Helper()
		require.Equal(t, http.StatusUnprocessableEntity, res.code, via+": "+string(res.body))
		require.Equal(t, "VALIDATION_FAILED", res.errCode(), via)
		require.Equal(t, code, firstDetailCode(res), via)
	}
	// register
	reject(r.register(map[string]string{"email": uniq("ep"), "password": "matkhau12345", "full_name": "X"}), auth.PasswordCommon, "register")
	e := uniq("epl")
	local, _, _ := strings.Cut(e, "@")
	reject(r.register(map[string]string{"email": e, "password": local + "A9!", "full_name": "X"}), auth.PasswordContainsEmail, "register")
	// reset-password: token chưa bị tiêu
	u := r.activeUser("epr")
	tok := r.resetToken(u)
	reject(r.reset(tok, "Mậtkhẩu12345"), auth.PasswordCommon, "reset-password")
	require.Equal(t, http.StatusOK, r.reset(tok, newPassword).code)
	// POST /me/password
	s := r.mustLoginWith(u, newPassword)
	reject(r.changePw(s, newPassword, "MatKhau12345"), auth.PasswordCommon, "me/password")
	reject(r.changePw(s, newPassword, "kkkkkkkkkkkk"), auth.PasswordLowEntropy, "me/password")
}

func TestPasswordNeverLogged(t *testing.T) {
	r := newSessRig(t)
	secrets := []string{"matkhau12345", "Edupilot#Sieu-Bi-Mat-77", "Moi-Sieu-Bi-Mat-88"}
	e := uniq("nl")
	res := r.register(map[string]string{"email": e, "password": secrets[0], "full_name": "X"})
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.NotContains(t, string(res.body), secrets[0], "thông báo lỗi không nhắc lại mật khẩu")
	u := r.activeUser("nlu")
	require.Equal(t, http.StatusOK, r.reset(r.resetToken(u), secrets[1]).code)
	s := r.mustLoginWith(u, secrets[1])
	require.Equal(t, http.StatusNoContent, r.changePw(s, secrets[1], secrets[2]).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.changePw(s, "sai-mat-khau-hien-tai", secrets[2]).code)
	for _, sec := range secrets {
		require.NotContains(t, r.logs.String(), sec, "log không chứa mật khẩu")
		require.Equal(t, "0", r.scalar(`select count(*)::text from audit_log where coalesce(before::text,'') like '%'||$1||'%' or coalesce(after::text,'') like '%'||$1||'%'`, sec))
		require.Equal(t, "0", r.scalar(`select count(*)::text from mail_outbox where payload::text like '%'||$1||'%'`, sec))
	}
}

func (r *sessRig) mustLoginWith(email, pw string) session {
	r.t.Helper()
	res := r.login(email, pw)
	require.Equal(r.t, http.StatusOK, res.code, string(res.body))
	return sessionOf(r.t, res)
}

// AC7: tệp danh sách phổ biến đúng khuôn — mỗi dòng đã fold (chữ thường, không dấu), dài ≥ 10 ký tự, không trùng, đã sắp xếp.
func TestCommonPasswordsFileWellFormed(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("common_passwords.txt")
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 1000)
	require.True(t, sort.StringsAreSorted(lines), "danh sách phải sắp xếp và không trùng")
	for i, l := range lines {
		require.GreaterOrEqual(t, utf8.RuneCountInString(l), 10, "dòng %d: %q", i+1, l)
		require.Equal(t, auth.Fold(l), l, "dòng %d chưa fold: %q", i+1, l)
		if i > 0 {
			require.NotEqual(t, lines[i-1], l, "trùng %q", l)
		}
	}
}
