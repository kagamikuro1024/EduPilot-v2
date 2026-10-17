package today_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// atRisk ghi danh một sinh viên tên `name` vào lớp và đặt mức rủi ro (AT_RISK vào `enteredAgo` trước).
func (r *rig) atRisk(c cls, name string, enteredAgo time.Duration) {
	r.t.Helper()
	u, _ := r.person(store.UserRoleSTUDENT, true)
	r.exec(`update users set full_name = $1 where id = $2`, name, u.ID)
	r.enrollAt(c, u, "STUDENT", "ACTIVE", "", 24*time.Hour)
	r.exec(`insert into student_risk (course_id, student_id, level, reasons, entered_at, computed_at) values ($1, $2, 'AT_RISK', '[{"code":"ATTENDANCE_LOW","values":{"absent":3,"recorded":4,"weeks":2}},{"code":"STUDY_LOW","values":{"minutes":20}}]', $3, now())`,
		c.id, u.ID, r.clk.Now().Add(-enteredAgo))
}

func attentionItems(res resp) []map[string]any {
	var out []map[string]any
	if rec, ok := res.json()["recommended"].(map[string]any); ok && rec["kind"] == "STUDENT_ATTENTION" {
		out = append(out, rec)
	}
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "STUDENT_ATTENTION" {
			out = append(out, m)
		}
	}
	return out
}

// TestStudentAttentionProvider — US-P5-04 AC7: sinh viên mới vào AT_RISK ≤ 7 ngày ⇒ mục bậc 95 cho Giảng viên và TA; vào quá 7 ngày thì không.
func TestStudentAttentionProvider(t *testing.T) {
	r := newRig(t)
	c := r.klass("762001")
	_, gv := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	require.Empty(t, attentionItems(r.tget(gv, "/me/today")))
	r.atRisk(c, "Sv A", 2*24*time.Hour)
	r.atRisk(c, "Sv B", 6*24*time.Hour)
	r.atRisk(c, "Sv C", 9*24*time.Hour) // cũ: không tính vào "mới"
	for name, s := range map[string]session{"giảng viên": gv, "trợ giảng": taSess} {
		items := attentionItems(r.tget(s, "/me/today"))
		require.Len(t, items, 1, name)
		require.Equal(t, "2 sinh viên mới vào diện cần chú ý · 762001", items[0]["title"], name)
		require.Equal(t, fmt.Sprintf("/students?filter=attention&course=%s", c.id), items[0]["href"], name)
	}
}

// TestAttentionBlockMax5 — khối `attention`: tối đa 5 sinh viên, mới vào trước, kèm tổng và cụm lý do ngắn; không điểm, không nội dung chat.
func TestAttentionBlockMax5(t *testing.T) {
	r := newRig(t)
	c := r.klass("762002")
	_, gv := r.teaching(c)
	for i := 1; i <= 7; i++ {
		r.atRisk(c, fmt.Sprintf("Sv %d", i), time.Duration(i)*time.Hour) // Sv 1 vào gần nhất
	}
	res := r.tget(gv, "/me/today")
	att := res.json()["attention"].([]any)
	require.Len(t, att, 5)
	require.EqualValues(t, 7, res.json()["attention_total"])
	first := att[0].(map[string]any)
	require.Equal(t, "Sv 1", first["student"].(map[string]any)["full_name"])
	require.Equal(t, "vắng 3/4 buổi; học ít", first["reason"])
	require.Equal(t, "762002", first["course"].(map[string]any)["class_code"])
}

// TestAttentionHiddenWhenNone — không có AT_RISK ⇒ `attention: []`, tổng 0 và không có mục (không rỗng giả).
func TestAttentionHiddenWhenNone(t *testing.T) {
	r := newRig(t)
	c := r.klass("762003")
	_, gv := r.teaching(c)
	res := r.tget(gv, "/me/today")
	require.Empty(t, res.json()["attention"].([]any))
	require.EqualValues(t, 0, res.json()["attention_total"])
	require.Empty(t, attentionItems(res))
}

// TestAttentionStaffOnly — sinh viên và Admin không có mục lẫn khối, kể cả khi lớp có AT_RISK (kể cả chính họ).
func TestAttentionStaffOnly(t *testing.T) {
	r := newRig(t)
	c := r.klass("762004")
	r.teaching(c)
	su, ss := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", 24*time.Hour)
	r.exec(`insert into student_risk (course_id, student_id, level, entered_at, computed_at) values ($1, $2, 'AT_RISK', now(), now())`, c.id, su.ID)
	for name, s := range map[string]session{"sinh viên": ss} {
		res := r.tget(s, "/me/today")
		require.Empty(t, attentionItems(res), name)
		for _, bad := range []string{"attention", "STUDENT_ATTENTION", "at_risk", "AT_RISK"} {
			require.NotContains(t, string(res.body), bad, name)
		}
	}
}
