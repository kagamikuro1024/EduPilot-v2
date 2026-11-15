package today_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/today"
)

// TestProviderGuidance — 10.6: «Lưu ý chấm chờ duyệt · <bài>» (chỉ TEACHER) và «N bài chấm theo bản cũ · <bài>» (nhân viên) chỉ ở lớp bật GRADING_GUIDANCE; sinh viên không thấy;
// bài đã sửa dở không tính; sự kiện `grading_guidance.changed` xoá cache nên mục đổi ngay.
func TestProviderGuidance(t *testing.T) {
	r := newRig(t)
	c := r.klass("764203")
	_, gv := r.teaching(c)
	_, sv := r.student("")
	a := r.asg(c, "BT03", 72*time.Hour, true, "")
	r.exec(`insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support) values ($1, $2, 'c1', 'Chấm chặt hơn.', 'STRICTER', 3)`, c.id, a)
	r.exec(`update assignments set guidance_version = 1 where id = $1`, a)
	r.gres(c, a, 1, "DONE", "DRAFT", "{}")
	r.gres(c, a, 1, "DONE", "DRAFT", "{}")
	r.gres(c, a, 1, "DONE", "REVIEWED", "{}") // đã duyệt: không tính
	changed := func() {
		payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
		require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(t.Context(), outbox.Message{Topic: today.TopicGuidanceChanged, Payload: payload}))
	}

	require.Empty(t, kindItems(r.tget(gv, "/me/today"), "GUIDANCE_PENDING"), "cờ lớp tắt")
	require.Empty(t, kindItems(r.tgetC(gv, "/me/today"), "GUIDANCE_STALE"))
	r.exec(`insert into course_guidance_flags (course_id, feature, enabled) values ($1, 'GRADING_GUIDANCE', true)`, c.id)
	changed()
	pend := kindItems(r.tgetC(gv, "/me/today"), "GUIDANCE_PENDING")
	require.Len(t, pend, 1)
	require.Equal(t, "Lưu ý chấm chờ duyệt · Bài BT03", pend[0]["title"])
	stale := kindItems(r.tgetC(gv, "/me/today"), "GUIDANCE_STALE")
	require.Len(t, stale, 1)
	require.Equal(t, "2 bài chấm theo bản cũ · Bài BT03", stale[0]["title"])
	require.Empty(t, kindItems(r.tgetC(sv, "/me/today"), "GUIDANCE_STALE"), "sinh viên không thấy")
	require.Empty(t, kindItems(r.tgetC(sv, "/me/today"), "GUIDANCE_PENDING"))

	r.exec(`update grading_guidance_items set status = 'APPROVED', decided_by = (select created_by from assignments where id = $1) where assignment_id = $1`, a)
	r.exec(`update grading_results set criteria = '[{"id":"c1","score":"1"}]'::jsonb, ai_scores = '{"c1":"2"}'::jsonb where course_id = $1 and status = 'DRAFT'`, c.id) // đã sửa dở
	changed()
	require.Empty(t, kindItems(r.tgetC(gv, "/me/today"), "GUIDANCE_PENDING"), "hết lưu ý chờ duyệt")
	require.Empty(t, kindItems(r.tgetC(gv, "/me/today"), "GUIDANCE_STALE"), "bài đã sửa dở không phải «bản cũ chưa chạm»")
}
