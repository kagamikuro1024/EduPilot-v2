package today_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

// unmarked: các mục SESSION_UNMARKED của một phản hồi `/me/today` (theo id mục, giữ thứ tự).
func unmarked(res resp) []map[string]any {
	var out []map[string]any
	if rec, ok := res.json()["recommended"].(map[string]any); ok && rec["kind"] == "SESSION_UNMARKED" {
		out = append(out, rec)
	}
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "SESSION_UNMARKED" {
			out = append(out, m)
		}
	}
	return out
}

// TestSessionUnmarkedProvider — US-P5-01 AC10: buổi đang diễn ra chưa điểm danh ⇒ mục bậc 12 cho Giảng viên và TA, có đường dẫn tới lưới.
func TestSessionUnmarkedProvider(t *testing.T) {
	r := newRig(t)
	c := r.klass("761995")
	_, gv := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	require.Empty(t, unmarked(r.tget(gv, "/me/today")), "chưa có buổi nào")
	r.session(c, 3, r.clk.Now().Add(-30*time.Minute), 90*time.Minute, "P.302")
	sid := r.scalar(`select id::text from class_sessions where course_id = $1 and session_no = 3`, c.id)

	for name, s := range map[string]session{"giảng viên": gv, "trợ giảng": taSess} {
		items := unmarked(r.tget(s, "/me/today"))
		require.Len(t, items, 1, name)
		it := items[0]
		require.Equal(t, "Điểm danh buổi 3 đang diễn ra · 761995", it["title"], name)
		require.Equal(t, fmt.Sprintf("/attendance?course=%s&session=%s", c.id, sid), it["href"], name)
		require.NotEqual(t, "overdue", it["urgency"], "đang diễn ra chưa quá hạn")
		require.Contains(t, it["reason"], "P.302")
	}
}

// TestSessionUnmarkedOverdueAfterEnd — AC10: sau ends_at là quá hạn ("Buổi n đã kết thúc, chưa điểm danh"); quá ends_at + 24 giờ thì hết.
func TestSessionUnmarkedOverdueAfterEnd(t *testing.T) {
	r := newRig(t)
	c := r.klass("761996")
	_, gv := r.teaching(c)
	r.session(c, 1, r.clk.Now().Add(-3*time.Hour), 2*time.Hour, "")            // kết thúc 1 giờ trước
	r.session(c, 2, r.clk.Now().Add(-27*time.Hour), 2*time.Hour, "")           // kết thúc 25 giờ trước: ngoài ân hạn
	r.session(c, 3, r.clk.Now().Add(2*time.Hour), 2*time.Hour, "")             // chưa bắt đầu
	r.session(c, 4, r.clk.Now().Add(-26*time.Hour), 2*time.Hour+time.Hour, "") // kết thúc 23 giờ trước: còn trong ân hạn
	items := unmarked(r.tget(gv, "/me/today"))
	var titles []string
	for _, it := range items {
		titles = append(titles, it["title"].(string))
		require.Equal(t, "overdue", it["urgency"])
	}
	require.ElementsMatch(t, []string{"Buổi 1 đã kết thúc, chưa điểm danh · 761996", "Buổi 4 đã kết thúc, chưa điểm danh · 761996"}, titles)
}

// TestSessionUnmarkedClearsOnComplete — AC10: hoàn tất điểm danh (attendance_done_at) ⇒ mục biến mất.
func TestSessionUnmarkedClearsOnComplete(t *testing.T) {
	r := newRig(t)
	c := r.klass("761997")
	gvUser, gv := r.teaching(c)
	r.session(c, 1, r.clk.Now().Add(-30*time.Minute), 90*time.Minute, "")
	require.Len(t, unmarked(r.tget(gv, "/me/today")), 1)
	r.exec(`update class_sessions set attendance_done_at = now(), attendance_done_by = $1 where course_id = $2`, gvUser.ID, c.id)
	require.Empty(t, unmarked(r.tget(gv, "/me/today")))
}

// TestSessionUnmarkedStaffOnly — AC10: sinh viên và Admin không thấy; lớp lưu trữ không có; mỗi lớp tối đa 3 mục, mới nhất trước.
func TestSessionUnmarkedStaffOnly(t *testing.T) {
	r := newRig(t)
	c := r.klass("761998")
	_, gv := r.teaching(c)
	svUser, sv := r.student("B20DC000098")
	r.enrollAt(c, svUser, "STUDENT", "ACTIVE", "", 24*time.Hour)
	_, adm := r.person(store.UserRoleADMIN, true)
	for i := 1; i <= 5; i++ { // 5 buổi chưa điểm danh, cách nhau 1 giờ, buổi 5 mới nhất, buổi nào cũng đã/đang trong ân hạn
		r.session(c, i, r.clk.Now().Add(-time.Duration(6-i)*time.Hour), 30*time.Minute, "")
	}
	require.Empty(t, unmarked(r.tget(sv, "/me/today")), "sinh viên không thấy")
	require.Empty(t, unmarked(r.tget(adm, "/me/today")), "Admin không thấy")
	items := unmarked(r.tget(gv, "/me/today"))
	require.Len(t, items, 3, "tối đa 3 mục mỗi lớp")
	var titles []string
	for _, it := range items {
		titles = append(titles, it["title"].(string))
	}
	require.ElementsMatch(t, []string{"Buổi 5 đã kết thúc, chưa điểm danh · 761998", "Buổi 4 đã kết thúc, chưa điểm danh · 761998", "Buổi 3 đã kết thúc, chưa điểm danh · 761998"}, titles, "mới nhất trước")
	r.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, c.id)
	require.Empty(t, unmarked(r.tget(gv, "/me/today")), "lớp lưu trữ không có")
}

// TestSessionUnmarkedTier — TLR-10: bậc 12, không trùng bậc nào đang chạy (15 đã là EXAM_GRADE_ERROR); xếp trước nó khi cùng không quá hạn.
func TestSessionUnmarkedTier(t *testing.T) {
	require.Equal(t, 12, today.TierSessionUnmarked)
	for name, tier := range map[string]int{
		"EXAM_GRADE_ERROR": today.TierExamGradeError, "EXAM_APPEAL": today.TierExamAppeal, "EXAM_PUBLISH_HOLD": today.TierExamPublishHold, "TICKET": today.TierTicket,
		"EXAM_OPEN": 8, "EXAM_IN_PROGRESS": today.TierExamInProgress, "JOIN_REQUEST": today.TierJoinRequest, "AI_CONFIRM": today.TierAIConfirm,
		"EXAM_SIMILARITY": today.TierExamSimilarity, "QUESTION_REVIEW": today.TierQuestionReview, "COURSE_SETUP": today.TierCourseSetup, "EMAIL_MISMATCH": today.TierEmailMismatch,
	} {
		require.NotEqual(t, today.TierSessionUnmarked, tier, name)
	}
	items := []today.Item{
		{ID: "a", Kind: today.KindExamGradeError, Tier: today.TierExamGradeError},
		{ID: "b", Kind: today.KindSessionUnmarked, Tier: today.TierSessionUnmarked},
	}
	today.Rank(items)
	require.Equal(t, "b", items[0].ID)
}

// TestCourseSetupSessionsStepHref — AC9: bước "Tạo lịch buổi học" mở /calendar với ngăn kéo đã mở (`new=sessions`) và tự tick khi có buổi.
func TestCourseSetupSessionsStepHref(t *testing.T) {
	r := newRig(t)
	c := r.klass("761994")
	_, gv := r.teaching(c)
	step := func() map[string]any {
		for _, a := range actionsOf(r.tget(gv, "/courses/"+c.sid()+"/today")) {
			if m := a.(map[string]any); m["kind"] == "COURSE_SETUP" {
				for _, s := range m["steps"].([]any) {
					if st := s.(map[string]any); st["key"] == "sessions" {
						return st
					}
				}
			}
		}
		return nil
	}
	require.Equal(t, "/calendar?course="+c.sid()+"&new=sessions", step()["href"])
	require.Equal(t, false, step()["done"])
	r.session(c, 1, r.clk.Now().Add(time.Hour), time.Hour, "")
	require.Equal(t, true, step()["done"])
}
