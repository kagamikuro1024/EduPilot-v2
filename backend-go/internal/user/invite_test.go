package user_test

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/store"
)

// AC1.
func TestInviteCreate(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	email := uniq("gv.moi")
	res := r.invite(a, "  "+strings.ToUpper(email)+" ", "Giảng Viên Mới", "TEACHER")
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	m := res.json()
	require.Equal(t, email, m["email"], "email được chuẩn hoá")
	require.Equal(t, "INVITED", m["status"])
	require.Equal(t, "TEACHER", m["role"])
	require.EqualValues(t, 1, m["version"])
	for _, banned := range []string{"token", "password", "hash"} {
		require.NotContains(t, string(res.body), banned)
	}
	require.Equal(t, "", r.scalar(`select password_hash from users where email = $1`, email), "chưa có mật khẩu")

	require.Equal(t, 1, r.mails(email, "invite_staff"))
	mm, tok := r.deliver(email)
	require.Equal(t, "Lời mời tham gia EduPilot", mm.Subject)
	require.Contains(t, mm.Text, "giảng viên")
	require.Contains(t, mm.Text, "72 giờ")
	require.Contains(t, mm.Text, "https://localhost/invite/"+tok)
	require.Regexp(t, `^[A-Za-z0-9_-]{43}$`, tok)
	var exp time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select expires_at from auth_tokens where token_hash = $1 and kind = 'INVITE'`, auth.HashToken(tok)).Scan(&exp))
	require.WithinDuration(t, r.clk.Now().Add(72*time.Hour), exp, 2*time.Second)
	require.Equal(t, "0", r.scalar(`select count(*)::text from auth_tokens where token_hash = $1`, tok), "DB chỉ giữ băm, không giữ bản rõ")
	ta := r.invite(a, uniq("ta.moi"), "Trợ Giảng", "TA")
	require.Equal(t, http.StatusCreated, ta.code)
	require.Equal(t, "TA", ta.json()["role"])
}

func TestInviteRejectsStudentAndAdmin(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	for _, role := range []string{"STUDENT", "ADMIN", "", "teacher", "SUPERUSER"} {
		e := uniq("bad")
		res := r.invite(a, e, "X", role)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, role)
		require.Equal(t, "VALIDATION_FAILED", res.errCode())
		require.Equal(t, "0", r.scalar(`select count(*)::text from users where email = $1`, e), "không tạo gì")
	}
	require.Equal(t, http.StatusUnprocessableEntity, r.invite(a, "khong-phai-email", "X", "TA").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.invite(a, uniq("n"), "   ", "TA").code)
	// trường lạ bị từ chối, không bị bỏ qua
	res := r.do(req{path: "/admin/users", bearer: a.access, hdr: idem(), body: map[string]string{"email": uniq("x"), "full_name": "X", "role": "TA", "password": "Mat-khau-hộ-2026"}})
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
}

func TestInviteDuplicateEmail(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	existing := r.addUser(uniq("co.san"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	res := r.invite(a, existing.Email, "Trùng", "TEACHER")
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "CONFLICT", res.errCode())
	require.Equal(t, "email", res.details()["field"])
	require.Zero(t, r.mails(existing.Email, ""))
	e := uniq("hai.lan")
	require.Equal(t, http.StatusCreated, r.invite(a, e, "Một", "TA").code)
	require.Equal(t, http.StatusConflict, r.invite(a, strings.ToUpper(e), "Hai", "TEACHER").code, "so khớp sau chuẩn hoá")
}

func TestInviteIdempotent(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	e := uniq("idem")
	h := idem()
	body := map[string]string{"email": e, "full_name": "Một Lần", "role": "TEACHER"}
	first := r.do(req{path: "/admin/users", bearer: a.access, hdr: h, body: body})
	second := r.do(req{path: "/admin/users", bearer: a.access, hdr: h, body: body})
	require.Equal(t, http.StatusCreated, first.code)
	require.Equal(t, http.StatusCreated, second.code)
	require.JSONEq(t, string(first.body), string(second.body), "phát lại nguyên phản hồi")
	require.Equal(t, "1", r.scalar(`select count(*)::text from users where email = $1`, e))
	require.Equal(t, 1, r.mails(e, "invite_staff"))
	missing := r.do(req{path: "/admin/users", bearer: a.access, body: body})
	require.Equal(t, http.StatusUnprocessableEntity, missing.code)
	require.Equal(t, "IDEMPOTENCY_KEY_REQUIRED", missing.errCode())
}

func TestInviteAudit(t *testing.T) {
	r := newRig(t)
	adm, a := r.admin()
	e := uniq("audit")
	res := r.invite(a, e, "Có Audit", "TA")
	require.Equal(t, http.StatusCreated, res.code)
	id := res.json()["id"].(string)
	_, tok := r.deliver(e)
	require.Equal(t, "1", r.scalar(`select count(*)::text from audit_log where entity = 'user' and entity_id = $1 and action = 'user_invited' and actor_id = $2`, id, adm.ID))
	require.Equal(t, "0", r.scalar(`select count(*)::text from audit_log where coalesce(after::text,'') like '%'||$1||'%' or coalesce(before::text,'') like '%'||$1||'%'`, tok), "audit_log không chứa token")
}

// AC3.
func TestAdminAPINeverCarriesSecrets(t *testing.T) {
	raw, err := os.ReadFile("../../api/openapi.yaml")
	require.NoError(t, err)
	spec := string(raw)
	// mọi khối thuộc /admin/users (4 thao tác) không nhắc password / token / hash, ngoại trừ khai báo Idempotency-Key
	for _, part := range regexp.MustCompile(`(?m)^  /api/v1/admin/users[^\n]*:\n(?:    .*\n|\n)+`).FindAllString(spec, -1) {
		cleaned := strings.ReplaceAll(part, "Idempotency-Key", "")
		require.NotRegexp(t, `(?i)password|token|hash`, cleaned, part[:60])
	}
	r := newRig(t)
	adm, a := r.admin()
	e := uniq("sach")
	created := r.invite(a, e, "Sạch Bí Mật", "TEACHER")
	id := created.json()["id"].(string)
	responses := []resp{
		created,
		r.do(req{method: http.MethodGet, path: "/admin/users?limit=100", bearer: a.access}),
		r.patch(a, id, map[string]any{"full_name": "Đã Đổi Tên", "version": 1}),
		r.do(req{path: "/admin/users/" + id + "/resend-invite", bearer: a.access}),
	}
	for i, res := range responses {
		require.Less(t, res.code, 300, "thao tác %d: %s", i, res.body)
		require.NotRegexp(t, `(?i)password|token|hash|student_code|ics_token|failed`, string(res.body), "thao tác %d", i)
	}
	// log yêu cầu không chứa thân: tên người được mời không nằm trong log
	require.NotContains(t, r.logs.String(), "Sạch Bí Mật")
	require.NotContains(t, r.logs.String(), e)
	_ = adm
}

func TestNoPasswordHashInJSON(t *testing.T) {
	t.Parallel()
	// View (thứ duy nhất handler đọc) không có trường nào mang mật khẩu / hash; nhật ký admin không có route đặt mật khẩu.
	raw, err := os.ReadFile("service.go")
	require.NoError(t, err)
	src := string(raw)
	require.NotRegexp(t, `(?i)type View struct[^}]*(password|hash|token)`, src)
	require.NotContains(t, src, "SetPassword")
	require.NotContains(t, src, "ResetUserPassword")
	require.NotContains(t, src, "ChangeUserPassword")
}

// AC4.
func TestResendInvite(t *testing.T) {
	r := newRig(t, func(e map[string]string) { e["AUTH_RESEND_SECONDS"] = "1" })
	_, a := r.admin()
	e := uniq("gui.lai")
	id := r.invite(a, e, "Gửi Lại", "TA").json()["id"].(string)
	_, old := r.deliver(e)
	time.Sleep(1100 * time.Millisecond)
	res := r.do(req{path: "/admin/users/" + id + "/resend-invite", bearer: a.access})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	exp, err := time.Parse(time.RFC3339, res.json()["expires_at"].(string))
	require.NoError(t, err)
	require.WithinDuration(t, r.clk.Now().Add(72*time.Hour), exp, 5*time.Second)
	// liên kết cũ chết ngay khi yêu cầu gửi lại được nhận (chưa cần thư mới được gửi)
	var revoked *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select revoked_at from auth_tokens where token_hash = $1`, auth.HashToken(old)).Scan(&revoked))
	require.NotNil(t, revoked)
	require.Equal(t, 2, r.mails(e, "invite_staff"))
	_, fresh := r.deliver(e)
	require.NotEqual(t, old, fresh)
	prev := r.do(req{path: "/auth/tokens/preview", body: map[string]string{"kind": "INVITE", "token": old}})
	require.Equal(t, http.StatusGone, prev.code)
	require.Equal(t, http.StatusOK, r.do(req{path: "/auth/tokens/preview", body: map[string]string{"kind": "INVITE", "token": fresh}}).code)
}

func TestResendInviteOnlyInvited(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	active := r.addUser(uniq("act"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	res := r.do(req{path: "/admin/users/" + active.ID.String() + "/resend-invite", bearer: a.access})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "CONFLICT", res.errCode())
	require.Zero(t, r.mails(active.Email, ""))
	require.Equal(t, http.StatusNotFound, r.do(req{path: "/admin/users/" + uuid.NewString() + "/resend-invite", bearer: a.access}).code)
	require.Equal(t, http.StatusNotFound, r.do(req{path: "/admin/users/khong-phai-uuid/resend-invite", bearer: a.access}).code)
}

func TestResendInviteThrottle(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	id := r.invite(a, uniq("thr"), "Chặn Gửi", "TEACHER").json()["id"].(string)
	require.Equal(t, http.StatusOK, r.do(req{path: "/admin/users/" + id + "/resend-invite", bearer: a.access}).code)
	res := r.do(req{path: "/admin/users/" + id + "/resend-invite", bearer: a.access})
	require.Equal(t, http.StatusTooManyRequests, res.code)
	require.Equal(t, "RATE_LIMITED", res.errCode())
	ra := int(res.json()["retry_after"].(float64))
	require.GreaterOrEqual(t, ra, 1)
	require.LessOrEqual(t, ra, 60)
}
