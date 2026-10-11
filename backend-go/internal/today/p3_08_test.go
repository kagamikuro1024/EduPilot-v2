package today_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

// ---- "Tiếp tục học" (US-P3-08 AC3) -----------------------------------------------------------------------------------------

func (r *rig) chat(c cls, user uuid.UUID, title string, lastMsg time.Duration, withMessage bool) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	var t any
	if title != "" {
		t = title
	}
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into chat_sessions (course_id, user_id, title, last_message_at) values ($1, $2, $3, now() - $4::interval) returning id`,
		c.id, user, t, fmt.Sprintf("%d seconds", int(lastMsg.Seconds()))).Scan(&id))
	if withMessage {
		_, err := r.pool.Exec(r.t.Context(), `insert into chat_messages (course_id, session_id, user_id, role, content) values ($1, $2, $3, 'USER', 'xin chào')`, c.id, id, user)
		require.NoError(r.t, err)
	}
	return id
}

func continueOf(res resp) []map[string]any {
	var out []map[string]any
	list, _ := res.json()["continue"].([]any)
	for _, x := range list {
		out = append(out, x.(map[string]any))
	}
	return out
}

func (r *rig) enrolledStudent(c cls) (store.User, session) {
	r.t.Helper()
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(c, u, "STUDENT", "ACTIVE", "", time.Hour)
	return u, s
}

// TestContinueChatSessions — AC3: tối đa 3 phiên có tin trong 7 ngày, mới nhất trước, đúng hình dạng; phiên rỗng / cũ / đã xoá không có.
func TestContinueChatSessions(t *testing.T) {
	r := newRig(t)
	c := r.klass("761971")
	u, s := r.enrolledStudent(c)
	old := r.chat(c, u.ID, "Quá cũ", 8*24*time.Hour, true)
	empty := r.chat(c, u.ID, "Chưa có tin", time.Minute, false)
	deleted := r.chat(c, u.ID, "Đã xoá", time.Minute, true)
	_, err := r.pool.Exec(t.Context(), `update chat_sessions set deleted_at = now() where id = $1`, deleted)
	require.NoError(t, err)
	s1 := r.chat(c, u.ID, "AES là gì", 5*time.Hour, true)
	s2 := r.chat(c, u.ID, "", 2*time.Hour, true) // chưa có tiêu đề
	s3 := r.chat(c, u.ID, "Quy chế thi", time.Hour, true)
	s4 := r.chat(c, u.ID, "Tuần 5", 30*time.Minute, true)
	res := r.tget(s, "/me/today")
	got := continueOf(res)
	require.Len(t, got, 3, "tối đa 3")
	require.Equal(t, []string{s4.String(), s3.String(), s2.String()}, []string{got[0]["id"].(string), got[1]["id"].(string), got[2]["id"].(string)}, "mới nhất trước")
	require.Equal(t, "CHAT", got[0]["kind"])
	require.Equal(t, "Tuần 5", got[0]["title"])
	require.Equal(t, "/chat?session="+s4.String(), got[0]["href"])
	require.Equal(t, "Cuộc trò chuyện với trợ lý", got[2]["title"])
	require.Equal(t, c.sid(), got[0]["course"].(map[string]any)["id"])
	require.Equal(t, "761971", got[0]["course"].(map[string]any)["class_code"])
	require.NotEmpty(t, got[0]["at"])
	for _, banned := range []string{old.String(), empty.String(), deleted.String(), s1.String()} {
		require.NotContains(t, string(res.body), banned)
	}
}

// TestContinueOnlyOwn — AC3: không bao giờ có phiên của người khác (cùng lớp hay khác lớp); trong phạm vi một lớp chỉ phiên của lớp đó.
func TestContinueOnlyOwn(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761972"), r.klass("761973")
	a, sa := r.enrolledStudent(c1)
	b, sb := r.enrolledStudent(c1)
	r.enrollAt(c2, a, "STUDENT", "ACTIVE", "", time.Hour)
	mine := r.chat(c1, a.ID, "Của A", time.Minute, true)
	theirs := r.chat(c1, b.ID, "Của B", time.Minute, true)
	other := r.chat(c2, a.ID, "Lớp 2 của A", 2*time.Minute, true)
	resA, resB := r.tget(sa, "/me/today"), r.tget(sb, "/me/today")
	require.Contains(t, string(resA.body), mine.String())
	require.Contains(t, string(resA.body), other.String())
	require.NotContains(t, string(resA.body), theirs.String())
	require.NotContains(t, string(resB.body), mine.String())
	require.NotContains(t, string(resB.body), other.String())
	require.Len(t, continueOf(resB), 1)
	one := r.tget(sa, "/courses/"+c1.sid()+"/today")
	require.Len(t, continueOf(one), 1, "một lớp: chỉ phiên của lớp đó")
	require.NotContains(t, string(one.body), other.String())
}

// TestContinueEmptyHidden — AC3: chưa chat → `continue` là `[]` (không null) để giao diện ẩn hẳn vùng "Tiếp tục học".
func TestContinueEmptyHidden(t *testing.T) {
	r := newRig(t)
	c := r.klass("761974")
	_, s := r.enrolledStudent(c)
	res := r.tget(s, "/me/today")
	require.Contains(t, string(res.body), `"continue":[]`)
	require.Empty(t, continueOf(res))
}

// ---- AI_CONFIRM (US-P3-08 AC4) ---------------------------------------------------------------------------------------------

type thr struct{ id uuid.UUID }

// thread chèn một thread của sinh viên `author`; aiState PENDING / ANSWERED / SKIPPED; post != "" thêm bài AI với verification_state đó.
func (r *rig) thread(c cls, author uuid.UUID, title, aiState, post string, age time.Duration) thr {
	r.t.Helper()
	var reason any
	if aiState == "SKIPPED" {
		reason = "NO_CONTEXT"
	}
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into forum_threads (course_id, author_id, title, body, ai_state, ai_skip_reason, created_at) values ($1, $2, $3, 'nội dung', $4::thread_ai_state, $5, now() - $6::interval) returning id`,
		c.id, author, title, aiState, reason, fmt.Sprintf("%d seconds", int(age.Seconds()))).Scan(&id))
	if post != "" {
		_, err := r.pool.Exec(r.t.Context(), `insert into forum_posts (course_id, thread_id, kind, body, verification_state, confidence) values ($1, $2, 'AI', 'trả lời', $3::post_verification, 0.9)`, c.id, id, post)
		require.NoError(r.t, err)
	}
	return thr{id}
}

func aiConfirm(res resp) []map[string]any {
	var out []map[string]any
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "AI_CONFIRM" {
			out = append(out, m)
		}
	}
	return out
}

// TestAIConfirmProviderStaffOnly — AC4: một mục "{a} câu trả lời AI chờ xác nhận · {b} câu hỏi AI chưa trả lời được" (chỉ số khác 0) cho Giảng viên và TA của lớp;
// bài AI bị ẩn / REJECTED / VERIFIED không tính; thread SKIPPED đã có bình luận Staff không tính; Sinh viên, Admin, Staff lớp khác không bao giờ nhận.
func TestAIConfirmProviderStaffOnly(t *testing.T) {
	r := newRig(t)
	c1, c2 := r.klass("761975"), r.klass("761976")
	_, gv := r.teaching(c1)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c1, ta, "TA", "ACTIVE", "", 24*time.Hour)
	_, other := r.teaching(c2)
	sv, svSess := r.enrolledStudent(c1)
	for i := range 4 {
		r.thread(c1, sv.ID, fmt.Sprintf("Chờ %d", i), "ANSWERED", "PENDING", time.Duration(i+1)*time.Hour)
	}
	hidden := r.thread(c1, sv.ID, "Ẩn", "ANSWERED", "PENDING", time.Hour)
	_, err := r.pool.Exec(t.Context(), `update forum_posts set hidden_at = now(), hidden_reason = 'x' where thread_id = $1`, hidden.id)
	require.NoError(t, err)
	r.thread(c1, sv.ID, "Đã xác nhận", "ANSWERED", "VERIFIED", time.Hour)
	r.thread(c1, sv.ID, "Bị loại", "ANSWERED", "REJECTED", time.Hour)
	r.thread(c1, sv.ID, "Chưa trả lời 1", "SKIPPED", "", 6*time.Hour+5*time.Minute) // cũ nhất
	r.thread(c1, sv.ID, "Chưa trả lời 2", "SKIPPED", "", 2*time.Hour)
	answered := r.thread(c1, sv.ID, "Đã có Staff trả lời", "SKIPPED", "", 3*time.Hour)
	_, err = r.pool.Exec(t.Context(), `insert into forum_posts (course_id, thread_id, author_id, kind, body) values ($1, $2, $3, 'HUMAN', 'em xem lại bài 3')`, c1.id, answered.id, ta.ID)
	require.NoError(t, err)
	for name, s := range map[string]session{"Giảng viên": gv, "TA": taSess} {
		items := aiConfirm(r.tget(s, "/me/today"))
		require.Len(t, items, 1, name)
		require.Equal(t, "4 câu trả lời AI chờ xác nhận · 2 câu hỏi AI chưa trả lời được · lớp 761975", items[0]["title"], name)
		require.Equal(t, "/threads?state=pending&course="+c1.sid(), items[0]["href"], name)
		require.Equal(t, c1.sid(), items[0]["course"].(map[string]any)["id"], name)
		require.Contains(t, items[0]["reason"], "6 giờ", name+": nêu thread cũ nhất")
	}
	require.Empty(t, aiConfirm(r.tget(other, "/me/today")), "Staff lớp khác không thấy")
	res := r.tget(svSess, "/me/today")
	require.NotContains(t, string(res.body), "AI_CONFIRM")
	adm, admSess := r.person(store.UserRoleADMIN, true)
	_ = adm
	require.NotContains(t, string(r.tget(admSess, "/me/today").body), "AI_CONFIRM")
	// chỉ nêu số khác 0
	_, err = r.pool.Exec(t.Context(), `update forum_threads set ai_state = 'ANSWERED', ai_skip_reason = null where ai_state = 'SKIPPED'`)
	require.NoError(t, err)
	items := aiConfirm(r.tget(gv, "/me/today"))
	require.Equal(t, "4 câu trả lời AI chờ xác nhận · lớp 761975", items[0]["title"])
	_, err = r.pool.Exec(t.Context(), `update forum_posts set verification_state = 'VERIFIED', verified_by = $1, verified_at = now() where kind = 'AI' and verification_state = 'PENDING'`, ta.ID)
	require.NoError(t, err)
	require.Empty(t, aiConfirm(r.tget(gv, "/me/today")), "hết việc thì mục biến mất")
}

// TestAIConfirmInvalidatedOnDecision — AC4: sự kiện `thread.post_decided` xoá cache "Hôm nay" của Staff ngay (không đợi TTL 60 s); bài AI quyết xong thì mục biến mất.
func TestAIConfirmInvalidatedOnDecision(t *testing.T) {
	r := newRig(t)
	c := r.klass("761977")
	g, gv := r.teaching(c)
	sv, _ := r.enrolledStudent(c)
	th := r.thread(c, sv.ID, "Chờ xác nhận", "ANSWERED", "PENDING", time.Hour)
	require.Len(t, aiConfirm(r.tgetC(gv, "/me/today")), 1)
	_, err := r.pool.Exec(t.Context(), `update forum_posts set verification_state = 'VERIFIED', verified_by = $1, verified_at = now() where thread_id = $2`, g.ID, th.id)
	require.NoError(t, err)
	require.Len(t, aiConfirm(r.tgetC(gv, "/me/today")), 1, "còn trong cache tới khi có sự kiện")
	deliver(t, r, today.TopicThreadPostDecided, c)
	require.Empty(t, aiConfirm(r.tgetC(gv, "/me/today")))
	require.True(t, strings.Contains(strings.Join(today.Topics(), ","), "thread.post_decided") && strings.Contains(strings.Join(today.Topics(), ","), "thread.created"))
}
