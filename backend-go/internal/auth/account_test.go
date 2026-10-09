package auth_test

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const registerMsg = "Nếu email này dùng được, chúng tôi đã gửi thư xác nhận. Kiểm tra hộp thư của bạn."

func (r *sessRig) userRow(email string) (role, status string, verified bool, code string, n int) {
	r.t.Helper()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select count(*) from users where email = $1`, email).Scan(&n))
	if n == 0 {
		return
	}
	var c *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select role::text, status::text, email_verified_at is not null, student_code from users where email = $1`, email).Scan(&role, &status, &verified, &c))
	if c != nil {
		code = *c
	}
	return
}

// AC1.
func TestRegisterCreatesStudentOnly(t *testing.T) {
	r := newSessRig(t)
	email := uniq("dk")
	res := r.register(regBody(email))
	require.Equal(t, http.StatusAccepted, res.code, string(res.body))
	require.Equal(t, registerMsg, res.json()["message"])
	role, status, verified, _, n := r.userRow(email)
	require.Equal(t, 1, n)
	require.Equal(t, "STUDENT", role)
	require.Equal(t, "PENDING_VERIFICATION", status)
	require.False(t, verified)
	var hash string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select password_hash from users where email = $1`, email).Scan(&hash))
	require.True(t, strings.HasPrefix(hash, "$2"), "bcrypt")
	require.NotContains(t, hash, rigPassword)
}

func TestRegisterRejectsRoleField(t *testing.T) {
	r := newSessRig(t)
	for _, f := range []string{"role", "status", "email_verified_at", "id", "version"} {
		email := uniq("role")
		b := map[string]any{"email": email, "password": rigPassword, "full_name": "X", f: "TEACHER"}
		res := r.register(b)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, f)
		require.Equal(t, "VALIDATION_FAILED", res.errCode())
		_, _, _, _, n := r.userRow(email)
		require.Zero(t, n, "%s: không tạo bản ghi", f)
	}
}

func TestRegisterRejectsUnknownFields(t *testing.T) {
	r := newSessRig(t)
	res := r.register(map[string]any{"email": uniq("u"), "password": rigPassword, "full_name": "X", "foo": 1})
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
}

// AC2: mọi trường hợp cùng thân / mã / khoá JSON.
func TestRegisterUniformResponse(t *testing.T) {
	r := newSessRig(t)
	active := uniq("act")
	r.user(active, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	disabled := uniq("dis")
	r.user(disabled, store.UserRoleSTUDENT, store.UserStatusDISABLED)
	pending := uniq("pen")
	require.Equal(t, http.StatusAccepted, r.register(regBody(pending)).code)
	invited := uniq("inv")
	_, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: invited, FullName: "Mời", Role: store.UserRoleTEACHER, Status: store.UserStatusINVITED})
	require.NoError(t, err)

	var bodies []string
	for _, e := range []string{uniq("moi"), active, disabled, pending, invited} {
		res := r.register(regBody(e))
		require.Equal(t, http.StatusAccepted, res.code, e)
		m := res.json()
		require.Equal(t, map[string]any{"message": registerMsg}, m)
		bodies = append(bodies, string(res.body))
	}
	for _, b := range bodies[1:] {
		require.Equal(t, bodies[0], b)
	}
	for _, e := range []string{active, disabled, pending, invited} {
		_, _, _, _, n := r.userRow(e)
		require.Equal(t, 1, n, "không tạo bản ghi thứ hai: %s", e)
	}
}

func TestRegisterExistingSendsEmailExists(t *testing.T) {
	r := newSessRig(t)
	for _, st := range []store.UserStatus{store.UserStatusACTIVE} {
		e := uniq("ex")
		r.user(e, store.UserRoleSTUDENT, st)
		require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
		require.Equal(t, 1, r.mails(e, "email_exists"))
		require.Zero(t, r.mails(e, "verify_email"))
	}
	p := uniq("pen")
	require.Equal(t, http.StatusAccepted, r.register(regBody(p)).code)
	require.Equal(t, http.StatusAccepted, r.register(regBody(p)).code)
	require.Equal(t, 1, r.mails(p, "verify_email"))
	require.Equal(t, 1, r.mails(p, "email_exists"))
}

func TestRegisterInvitedResendsInvite(t *testing.T) {
	r := newSessRig(t)
	e := uniq("inv")
	_, err := store.New(r.pool).InsertUser(t.Context(), store.InsertUserParams{Email: e, FullName: "Thầy Mời", Role: store.UserRoleTA, Status: store.UserStatusINVITED})
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	require.Equal(t, 1, r.mails(e, "invite_staff"))
	require.Zero(t, r.mails(e, "email_exists"))
	m, tok := r.deliver(e)
	require.Equal(t, "Lời mời tham gia EduPilot", m.Subject)
	require.Contains(t, m.Text, "trợ giảng")
	require.Regexp(t, `^[A-Za-z0-9_-]{43}$`, tok)
	require.Contains(t, m.Text, "https://localhost/invite/"+tok)
}

func TestRegisterDisabledSendsNothing(t *testing.T) {
	r := newSessRig(t)
	e := uniq("dis")
	r.user(e, store.UserRoleSTUDENT, store.UserStatusDISABLED)
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	require.Zero(t, r.mails(e, ""))
}

func TestRegisterTimingEqualized(t *testing.T) {
	testutil.SkipTiming(t)
	r := newSessRig(t, func(e map[string]string) { e["BCRYPT_COST"] = "10" })
	exist := uniq("tm")
	r.user(exist, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	median := func(newEmail bool) time.Duration {
		ds := make([]time.Duration, 20)
		for i := range ds {
			e := exist
			if newEmail {
				e = uniq("tmn")
			}
			t0 := time.Now()
			require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
			ds[i] = time.Since(t0)
		}
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		return ds[len(ds)/2]
	}
	fresh, dup := median(true), median(false)
	lo, hi := min(fresh, dup), max(fresh, dup)
	require.GreaterOrEqual(t, float64(lo), 0.65*float64(hi), "mới=%v đã có=%v", fresh, dup)
}

// AC3.
func TestSelfDeclaredStudentCodeLinksNothing(t *testing.T) {
	r := newSessRig(t)
	victimCode := "B20DCCN001"
	// "danh sách lớp": B đã ACTIVE trong một lớp, MSSV của B = victimCode.
	b := r.user(uniq("b"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	var course uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		values ('INT1006', 'LOP-'||left(gen_random_uuid()::text, 6), 'Lớp thử', '2026-2027-HK1', 'AN7K2MQ', $1) returning id`, b.ID).Scan(&course))
	_, err := r.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ROSTER', $3)`, course, b.ID, victimCode)
	require.NoError(t, err)

	mallory := uniq("mal")
	require.Equal(t, http.StatusAccepted, r.register(map[string]string{"email": mallory, "password": rigPassword, "full_name": "Kẻ Giả", "student_code": strings.ToLower(victimCode)}).code)
	_, tok := r.deliver(mallory)
	require.Equal(t, http.StatusOK, r.verify(tok).code)
	_, status, verified, code, _ := r.userRow(mallory)
	require.Equal(t, "ACTIVE", status)
	require.True(t, verified)
	require.Equal(t, victimCode, code, "chỉ lưu thông tin khai báo, đã chuẩn hoá chữ hoa")
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from enrollments e join users u on u.id = e.user_id where u.email = $1`, mallory).Scan(&n))
	require.Zero(t, n, "MSSV tự khai không nối được vào lớp")
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from enrollments where course_id = $1 and student_code_snapshot = $2`, course, victimCode).Scan(&n))
	require.Equal(t, 1, n, "bản ghi của B không bị đụng")
}

func TestStudentCodeFormat(t *testing.T) {
	r := newSessRig(t)
	for code, want := range map[string]int{"B20DCCN001": 202, "b20dccn001": 202, "123456": 202, "": 202, "12345": 422, "B20DCCN-001": 422, "1234567890123456": 422, "20 22900": 422, "Ặ12345": 422} {
		res := r.register(map[string]string{"email": uniq("mssv"), "password": rigPassword, "full_name": "X", "student_code": code})
		require.Equal(t, want, res.code, "%q", code)
		if want == 422 {
			require.Equal(t, "STUDENT_CODE_FORMAT", firstDetailCode(res))
		}
	}
}

func firstDetailCode(res resp) string {
	d, _ := res.json()["details"].([]any)
	if len(d) == 0 {
		return ""
	}
	m, _ := d[0].(map[string]any)
	s, _ := m["code"].(string)
	return s
}

// SRS 4.2.5: không truy vấn sqlc nào dùng MSSV trong điều kiện để nối / mở dữ liệu, ngoài danh sách trắng.
func TestNoQueryLinksByStudentCode(t *testing.T) {
	t.Parallel()
	// ListMembers: ô tìm của GIẢNG VIÊN / TA trong lớp của họ (US-P2-09 AC13: `q` tìm MSSV) — chỉ lọc hiển thị, không nối, không mở dữ liệu (đề xuất #10).
	allowed := map[string]bool{"EnrollmentConflictByStudentCode": true, "RosterStudentCodeConflict": true, "UpdateProfile": true, "ListMembers": true}
	dir := filepath.Join("..", "store", "queries")
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	cond := regexp.MustCompile(`(?is)\b(where|and|or|on|having)\b[^;]*student_code`)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, block := range strings.Split(string(raw), "-- name: ")[1:] {
			name := strings.Fields(block)[0]
			if cond.MatchString(block) && !allowed[name] {
				t.Errorf("%s: truy vấn %s dùng student_code trong điều kiện (không có trong danh sách trắng SRS 4.2.5)", filepath.Base(f), name)
			}
		}
	}
}

// AC4.
func TestVerifyEmail(t *testing.T) {
	r := newSessRig(t)
	e := uniq("vf")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	_, tok := r.deliver(e)
	res := r.verify(tok)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, map[string]any{"status": "verified"}, res.json())
	_, status, verified, _, _ := r.userRow(e)
	require.Equal(t, "ACTIVE", status)
	require.True(t, verified)
	var used *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select used_at from auth_tokens where token_hash = $1`, hashOf(tok)).Scan(&used))
	require.NotNil(t, used)
	require.Zero(t, countRows(t, r, `select count(*) from auth_tokens where token_hash = $1`, tok), "DB chỉ có băm")

	again := r.verify(tok)
	require.Equal(t, http.StatusGone, again.code)
	require.Equal(t, "LINK_INVALID", again.errCode())
	require.Equal(t, "used", again.json()["details"].(map[string]any)["reason"])
}

func TestVerifyEmailSingleUseRace(t *testing.T) {
	r := newSessRig(t)
	e := uniq("race")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	_, tok := r.deliver(e)
	var wg sync.WaitGroup
	codes := make([]int, 50)
	for i := range codes {
		wg.Add(1)
		go func() { defer wg.Done(); codes[i] = r.verify(tok).code }()
	}
	wg.Wait()
	ok, gone := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusGone:
			gone++
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 49, gone)
}

// AC5.
func TestVerifyEmailExpired24h(t *testing.T) {
	r := newSessRig(t)
	e := uniq("exp")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	_, tok := r.deliver(e)
	r.clk.Advance(24*time.Hour + time.Second)
	res := r.verify(tok)
	require.Equal(t, http.StatusGone, res.code)
	require.Equal(t, "expired", res.json()["details"].(map[string]any)["reason"])
	_, status, verified, _, _ := r.userRow(e)
	require.Equal(t, "PENDING_VERIFICATION", status)
	require.False(t, verified)
}

func TestVerifyEmailUnknown(t *testing.T) {
	r := newSessRig(t)
	for _, tok := range []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "x", strings.Repeat("z", 100)} {
		res := r.verify(tok)
		require.Equal(t, http.StatusGone, res.code, tok)
		require.Equal(t, "invalid", res.json()["details"].(map[string]any)["reason"])
	}
}

func TestVerifyEmailReplaced(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["AUTH_RESEND_SECONDS"] = "1" })
	e := uniq("rep")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	_, t1 := r.deliver(e)
	time.Sleep(1100 * time.Millisecond)
	require.Equal(t, http.StatusAccepted, r.resend(e).code)
	_, t2 := r.deliver(e)
	require.NotEqual(t, t1, t2)
	old := r.verify(t1)
	require.Equal(t, http.StatusGone, old.code)
	require.Equal(t, "invalid", old.json()["details"].(map[string]any)["reason"], "token bị thay là invalid")
	_, status, _, _, _ := r.userRow(e)
	require.Equal(t, "PENDING_VERIFICATION", status)
	require.Equal(t, http.StatusOK, r.verify(t2).code)
}

// AC6.
func TestResendThrottle60s(t *testing.T) {
	r := newSessRig(t)
	e := uniq("rs")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	first := r.resend(e)
	require.Equal(t, http.StatusAccepted, first.code)
	second := r.resend(e)
	require.Equal(t, http.StatusTooManyRequests, second.code)
	require.Equal(t, "RATE_LIMITED", second.errCode())
	ra := int(second.json()["retry_after"].(float64))
	require.GreaterOrEqual(t, ra, 1)
	require.LessOrEqual(t, ra, 60)
	require.Equal(t, 2, r.mails(e, "verify_email"), "đăng ký + một lần gửi lại; lần hai bị chặn")
}

func TestResendRevokesOld(t *testing.T) {
	r := newSessRig(t, func(e map[string]string) { e["AUTH_RESEND_SECONDS"] = "1" })
	e := uniq("rr")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	_, t1 := r.deliver(e)
	time.Sleep(1100 * time.Millisecond)
	require.Equal(t, http.StatusAccepted, r.resend(e).code)
	r.deliver(e)
	var revoked *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select revoked_at from auth_tokens where token_hash = $1`, hashOf(t1)).Scan(&revoked))
	require.NotNil(t, revoked)
	time.Sleep(1100 * time.Millisecond)
	require.Equal(t, http.StatusAccepted, r.resend(e).code, "sau cửa sổ gửi lại được")
}

func TestResendVerifiedNoMail(t *testing.T) {
	r := newSessRig(t)
	e := uniq("rv")
	r.user(e, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	_, err := r.pool.Exec(t.Context(), `update users set email_verified_at = now() where email = $1`, e)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, r.resend(e).code)
	require.Zero(t, r.mails(e, ""))
}

func TestResendUniform(t *testing.T) {
	r := newSessRig(t)
	pending := uniq("p")
	require.Equal(t, http.StatusAccepted, r.register(regBody(pending)).code)
	verified := uniq("v")
	r.user(verified, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	var bodies []string
	for _, e := range []string{pending, verified, uniq("khong.co")} {
		res := r.resend(e)
		require.Equal(t, http.StatusAccepted, res.code)
		bodies = append(bodies, string(res.body))
	}
	require.Equal(t, bodies[0], bodies[1])
	require.Equal(t, bodies[0], bodies[2])
	// giới hạn tính theo băm email KỂ CẢ email không tồn tại
	ghost := uniq("ghost")
	require.Equal(t, http.StatusAccepted, r.resend(ghost).code)
	require.Equal(t, http.StatusTooManyRequests, r.resend(ghost).code)
}

// resend khi đã đăng nhập, không kèm thân.
func TestResendWithBearerNoBody(t *testing.T) {
	r := newSessRig(t)
	e := uniq("bb")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	s := r.mustLogin(e)
	res := r.do(req{path: "/auth/resend-verification", bearer: s.access})
	require.Equal(t, http.StatusAccepted, res.code, string(res.body))
	require.Equal(t, 2, r.mails(e, "verify_email"))
	require.Equal(t, http.StatusUnprocessableEntity, r.do(req{path: "/auth/resend-verification"}).code, "không Bearer, không thân ⇒ 422")
}

// AC7.
func TestUnverifiedCanLogin(t *testing.T) {
	r := newSessRig(t)
	e := uniq("ul")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	res := r.login(e, rigPassword)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	u := res.json()["user"].(map[string]any)
	require.Equal(t, false, u["email_verified"])
	require.Equal(t, "PENDING_VERIFICATION", u["status"])
}

func TestVerifyPromotesRosterPending(t *testing.T) {
	r := newSessRig(t)
	e := uniq("ro")
	require.Equal(t, http.StatusAccepted, r.register(regBody(e)).code)
	var uid, owner, course uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id from users where email = $1`, e).Scan(&uid))
	owner = r.user(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE).ID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		values ('INT1006', 'LOP-'||left(gen_random_uuid()::text, 6), 'Lớp thử', '2026-2027-HK1', 'BX4P9TW', $1) returning id`, owner).Scan(&course))
	_, err := r.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, warning) values ($1, $2, 'STUDENT', 'PENDING', 'ROSTER', 'EMAIL_UNVERIFIED')`, course, uid)
	require.NoError(t, err)
	_, tok := r.deliver(e)
	require.Equal(t, http.StatusOK, r.verify(tok).code)
	var status string
	var warning *string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select status::text, warning from enrollments where user_id = $1`, uid).Scan(&status, &warning))
	require.Equal(t, "ACTIVE", status)
	require.Nil(t, warning)
}

// AC8.
func TestRegisterValidationTable(t *testing.T) {
	r := newSessRig(t)
	long := strings.Repeat("a", 243) + "@example.com" // 255 ký tự
	ok := func(over map[string]string) map[string]string {
		m := map[string]string{"email": uniq("v"), "password": rigPassword, "full_name": "Nguyễn Văn Ặ"}
		for k, v := range over {
			m[k] = v
		}
		return m
	}
	cases := []struct {
		name  string
		over  map[string]string
		code  int
		field string
	}{
		{"hợp lệ có dấu", nil, 202, ""},
		{"email có +", map[string]string{"email": "an+lop1." + uuid.NewString()[:6] + "@example.test"}, 202, ""},
		{"email viết hoa + khoảng trắng", map[string]string{"email": "  AN." + uuid.NewString()[:6] + "@Example.Test "}, 202, ""},
		{"tên 100 ký tự", map[string]string{"full_name": strings.Repeat("Ặ", 100)}, 202, ""},
		{"tên toàn khoảng trắng quanh", map[string]string{"full_name": "  Trần Thu Uyên  "}, 202, ""},
		{"thiếu email", map[string]string{"email": ""}, 422, "email"},
		{"a@b", map[string]string{"email": "a@b"}, 422, "email"},
		{"a b@c.d", map[string]string{"email": "a b@c.d"}, 422, "email"},
		{"hai @", map[string]string{"email": "a@b@c.de"}, 422, "email"},
		{"thiếu @", map[string]string{"email": "abc.example.test"}, 422, "email"},
		{"điều khiển", map[string]string{"email": "a\u0007b@c.de"}, 422, "email"},
		{"255 ký tự", map[string]string{"email": long}, 422, "email"},
		{"đuôi chấm", map[string]string{"email": "a@b."}, 422, "email"},
		{"tên rỗng", map[string]string{"full_name": ""}, 422, "full_name"},
		{"tên chỉ khoảng trắng", map[string]string{"full_name": "   "}, 422, "full_name"},
		{"tên 101 ký tự", map[string]string{"full_name": strings.Repeat("a", 101)}, 422, "full_name"},
		{"tên có điều khiển", map[string]string{"full_name": "An\u0000Bình"}, 422, "full_name"},
		{"mật khẩu 9 ký tự", map[string]string{"password": "abcdefgh9"}, 422, "password"},
		{"mật khẩu rỗng", map[string]string{"password": ""}, 422, "password"},
		{"mật khẩu 73 byte", map[string]string{"password": strings.Repeat("x", 73)}, 422, "password"},
		{"mật khẩu 10 ký tự Việt", map[string]string{"password": "ặếộửữầẳẵẫẩ"}, 202, ""},
		{"MSSV sai", map[string]string{"student_code": "ab"}, 422, "student_code"},
	}
	require.GreaterOrEqual(t, len(cases), 20)
	for _, tc := range cases {
		res := r.register(ok(tc.over))
		require.Equal(t, tc.code, res.code, "%s: %s", tc.name, res.body)
		if tc.code == 422 {
			require.Equal(t, "VALIDATION_FAILED", res.errCode())
			found := false
			for _, d := range res.json()["details"].([]any) {
				m := d.(map[string]any)
				require.NotEmpty(t, m["code"])
				require.NotEmpty(t, m["message"])
				found = found || m["field"] == tc.field
			}
			require.True(t, found, "%s: thiếu lỗi cho trường %s", tc.name, tc.field)
		}
	}
	// 413: thân vượt MAX_BODY_BYTES
	r2 := newSessRig(t, func(e map[string]string) { e["MAX_BODY_BYTES"] = "200" })
	big := regBody(uniq("big"))
	big["full_name"] = strings.Repeat("a", 500)
	require.Equal(t, http.StatusRequestEntityTooLarge, r2.register(big).code)
	// chuẩn hoá: email lưu chữ thường, đã cắt
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from users where email ~ '[A-Z ]'`).Scan(&n))
	require.Zero(t, n)
}

// AC11.
func TestPublicEndpointsIgnoreJWTRole(t *testing.T) {
	r := newSessRig(t)
	admin := r.user(uniq("ad"), store.UserRoleADMIN, store.UserStatusACTIVE)
	s := r.mustLogin(admin.Email)
	e := uniq("byadmin")
	res := r.do(req{path: "/auth/register", body: regBody(e), bearer: s.access})
	require.Equal(t, http.StatusAccepted, res.code)
	role, _, _, _, _ := r.userRow(e)
	require.Equal(t, "STUDENT", role, "đã đăng nhập vẫn chỉ tạo STUDENT")
}

func TestNoSelfRolePatch(t *testing.T) {
	r := newSessRig(t)
	e := uniq("sv")
	u := r.user(e, store.UserRoleSTUDENT, store.UserStatusACTIVE)
	s := r.mustLogin(e)
	for _, m := range []string{http.MethodPatch, http.MethodPut, http.MethodPost} {
		for _, p := range []string{"/me", "/me/profile", "/me/role", "/users/" + u.ID.String()} {
			res := r.do(req{method: m, path: p, body: map[string]string{"role": "ADMIN"}, bearer: s.access})
			require.GreaterOrEqual(t, res.code, 400, "%s %s", m, p)
		}
	}
	role, _, _, _, _ := r.userRow(e)
	require.Equal(t, "STUDENT", role)
}
