package today_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// dead chèn một dòng outbox chết `age` trước đồng hồ giả. payload / last_error cố ý chứa dữ liệu cá nhân: mục «Hôm nay» không được đọc chúng.
func (r *rig) dead(topic string, age time.Duration) {
	r.t.Helper()
	r.exec(`insert into outbox (topic, payload, attempts, last_error, dead_at) values ($1, '{"to":"sv.bi.mat@example.test","full_name":"Nguyễn Bí Mật"}', 4, 'smtp: 550 sv.bi.mat@example.test không tồn tại', $2)`, topic, r.clk.Now().Add(-age))
}

// TestDeadLetterProvider — US-P2-11 AC16: một mục DEAD_LETTER cho Admin khi có dòng outbox chết trong 7 ngày: «Hàng đợi có N tin lỗi hẳn», lý do «{chủ đề nhiều nhất} ({k}) · mới nhất {tuổi} trước.», không href;
// dòng chết quá 7 ngày không đếm; hết dòng chết trong 7 ngày thì mục biến mất.
func TestDeadLetterProvider(t *testing.T) {
	r := newRig(t)
	_, as := r.person(store.UserRoleADMIN, true)
	require.Empty(t, kindItems(r.tget(as, "/me/today"), "DEAD_LETTER"), "không có dòng chết")
	r.dead("mail.send", 3*time.Hour)
	r.dead("mail.send", 26*time.Hour)
	r.dead("outbox.test", 5*24*time.Hour)
	r.dead("mail.send", 8*24*time.Hour) // quá 7 ngày: không đếm
	items := kindItems(r.tget(as, "/me/today"), "DEAD_LETTER")
	require.Len(t, items, 1)
	require.Equal(t, "Hàng đợi có 3 tin lỗi hẳn", items[0]["title"])
	require.Equal(t, "mail.send (2) · mới nhất 3 giờ trước.", items[0]["reason"])
	require.Equal(t, "", items[0]["href"], "chưa có trang dead-letter")
	r.exec(`update outbox set dead_at = dead_at - interval '10 days'`)
	require.Empty(t, kindItems(r.tget(as, "/me/today"), "DEAD_LETTER"), "không còn dòng chết trong 7 ngày")
}

// TestDeadLetterNoPII — AC16: thân phản hồi không chứa payload, last_error, địa chỉ thư, tên.
func TestDeadLetterNoPII(t *testing.T) {
	r := newRig(t)
	_, as := r.person(store.UserRoleADMIN, true)
	r.dead("mail.send", time.Hour)
	body := string(r.tget(as, "/me/today").body)
	require.Contains(t, body, "DEAD_LETTER")
	for _, banned := range []string{"payload", "last_error", "sv.bi.mat", "example.test", "Nguyễn Bí Mật", "smtp"} {
		require.NotContains(t, body, banned)
	}
}

// TestDeadLetterRBAC — AC16: giảng viên, TA, sinh viên không có mục này.
func TestDeadLetterRBAC(t *testing.T) {
	r := newRig(t)
	c := r.klass("762101")
	r.dead("mail.send", time.Hour)
	gv, gs := r.person(store.UserRoleTEACHER, true)
	r.enrollAt(c, gv, "TEACHER", "ACTIVE", "", time.Hour)
	ta, ts := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", time.Hour)
	sv, ss := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, sv, "STUDENT", "ACTIVE", "", time.Hour)
	for name, s := range map[string]session{"giảng viên": gs, "TA": ts, "sinh viên": ss} {
		require.False(t, strings.Contains(string(r.tget(s, "/me/today").body), "DEAD_LETTER"), name)
	}
}

// TestDeadLetterProviderError — AC16: Provider này lỗi thì các mục khác của Admin vẫn trả 200.
func TestDeadLetterProviderError(t *testing.T) {
	r := newRig(t)
	r.klass("762111") // lớp không giảng viên: COURSE_NO_TEACHER
	_, as := r.person(store.UserRoleADMIN, true)
	r.dead("mail.send", time.Hour)
	r.exec(`alter table outbox rename column dead_at to dead_at_x`) // truy vấn của Provider hỏng
	res := r.tget(as, "/me/today")
	require.Equal(t, 200, res.code)
	require.NotEmpty(t, kindItems(res, "COURSE_NO_TEACHER"), "mục khác của Admin vẫn có")
	require.Empty(t, kindItems(res, "DEAD_LETTER"))
}
