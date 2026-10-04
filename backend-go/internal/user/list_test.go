package user_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) seedMany(n int) {
	r.t.Helper()
	for i := 0; i < n; i++ {
		r.addUser(uniq(fmt.Sprintf("sv%02d", i)), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	}
}

func (r *rig) list(s session, query string) resp {
	return r.do(req{method: http.MethodGet, path: "/admin/users" + query, bearer: s.access})
}

func items(res resp) []map[string]any {
	var out []map[string]any
	for _, it := range res.json()["items"].([]any) {
		out = append(out, it.(map[string]any))
	}
	return out
}

// AC8.
func TestAdminListCursor(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	r.seedMany(57)
	seen := map[string]bool{}
	cursor, pages := "", 0
	for {
		q := "?limit=25"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		res := r.list(a, q)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		for _, it := range items(res) {
			id := it["id"].(string)
			require.False(t, seen[id], "trùng %s giữa các trang", id)
			seen[id] = true
		}
		pages++
		next, _ := res.json()["next_cursor"].(string)
		if next == "" {
			break
		}
		cursor = next
	}
	require.Len(t, seen, 58, "57 sinh viên + 1 admin")
	require.Equal(t, 3, pages)
	def := r.list(a, "")
	require.Len(t, items(def), 30, "mặc định 30")
	require.NotNil(t, def.json()["next_cursor"])
	bad := r.list(a, "?cursor=khong-phai-cursor")
	require.Equal(t, http.StatusUnprocessableEntity, bad.code)
	require.Equal(t, "INVALID_CURSOR", bad.errCode())
}

func TestAdminListLimitCap(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	require.Equal(t, http.StatusUnprocessableEntity, r.list(a, "?limit=101").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.list(a, "?limit=0").code)
	require.Equal(t, http.StatusOK, r.list(a, "?limit=100").code)
}

func TestAdminListFilters(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	r.addUser("nguyen.van.an@example.test", store.UserRoleSTUDENT, store.UserStatusACTIVE)
	_, err := r.pool.Exec(t.Context(), `update users set full_name = 'Nguyễn Văn Ân' where email = 'nguyen.van.an@example.test'`)
	require.NoError(t, err)
	gv := r.addUser(uniq("giang.vien"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusDISABLED)
	inv := r.invite(a, "gv.chua.nhan@example.test", "Lê Thị Ngọc", "TEACHER")
	require.Equal(t, http.StatusCreated, inv.code)

	emails := func(query string) []string {
		res := r.list(a, query)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		var out []string
		for _, it := range items(res) {
			out = append(out, it["email"].(string))
		}
		return out
	}
	require.ElementsMatch(t, []string{gv.Email, "gv.chua.nhan@example.test"}, emails("?role=TEACHER"))
	require.Equal(t, []string{"gv.chua.nhan@example.test"}, emails("?status=INVITED"))
	require.Equal(t, []string{"nguyen.van.an@example.test"}, emails("?q=nguyen"), "tìm theo tên")
	require.Equal(t, []string{"nguyen.van.an@example.test"}, emails("?q="+url.QueryEscape("Nguyễn Văn Ân")), "có dấu")
	require.Equal(t, []string{"nguyen.van.an@example.test"}, emails("?q="+url.QueryEscape("van an")), "không dấu, chứa")
	require.Equal(t, []string{"gv.chua.nhan@example.test"}, emails("?q="+url.QueryEscape("ngoc")), "không phân biệt dấu: ngoc ⇒ Ngọc")
	require.Equal(t, []string{"gv.chua.nhan@example.test"}, emails("?q=gv.chua"), "tiền tố email")
	require.Empty(t, emails("?q=chua.nhan"), "email chỉ khớp theo tiền tố")
	require.Empty(t, emails("?q=%25"), "% là chữ, không phải mẫu")
	require.Equal(t, http.StatusUnprocessableEntity, r.list(a, "?role=ROOT").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.list(a, "?status=NOPE").code)
	both := emails("?role=TA&status=DISABLED")
	require.Len(t, both, 1)
}

func TestAdminListMinimalFields(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	sv := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	_, err := r.pool.Exec(t.Context(), `update users set student_code = 'B20DCCN001', ics_token = 'tok-ics', failed_logins = 3 where id = $1`, sv.ID)
	require.NoError(t, err)
	res := r.list(a, "")
	for _, it := range items(res) {
		require.Len(t, it, 7)
		for _, k := range []string{"id", "email", "full_name", "role", "status", "last_login_at", "version"} {
			require.Contains(t, it, k)
		}
	}
	for _, banned := range []string{"student_code", "B20DCCN001", "ics", "tok-ics", "password", "failed"} {
		require.NotContains(t, string(res.body), banned)
	}
}

func TestAdminListNoNPlusOne(t *testing.T) {
	r := newRig(t)
	_, a := r.admin()
	r.seedMany(40)
	before := r.qc.n.Load()
	res := r.list(a, "?limit=100")
	require.Equal(t, http.StatusOK, res.code)
	require.Len(t, items(res), 41)
	require.EqualValues(t, 1, r.qc.n.Load()-before, "đúng một câu SQL cho cả trang (không N+1)")
}
