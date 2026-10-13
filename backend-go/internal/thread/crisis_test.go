package thread_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/thread"
)

// US-P4-07 AC4 (SRS 4.9.4): bài đăng / bình luận / sửa có tín hiệu khủng hoảng không được lưu, không AI, không pii_events; precheck không bao giờ chặn.
const crisisText = "Em không muốn sống nữa, học hành áp lực quá"

func (r *rig) noSideEffects() {
	r.t.Helper()
	require.Zero(r.t, r.count(`select count(*) from pii_events`))
	require.Zero(r.t, r.count(`select count(*) from outbox`))
	require.Zero(r.t, r.count(`select count(*) from jobs`))
	require.Zero(r.t, r.count(`select count(*) from notifications`))
}

func TestThreadCrisisNotStored(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.svc.Support = "Phòng Công tác sinh viên, phòng 101 nhà A1"
	_, err := r.create(r.sv, "Hỏi gì đó", crisisText, false)
	require.Equal(t, apierr.CrisisSupport, code(err))
	require.Equal(t, 422, status(err))
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Contains(t, ae.Message, "Mình rất tiếc")
	require.Contains(t, ae.Message, r.svc.Support)
	require.Equal(t, map[string]any{"actions": []string{"OPEN_PRIVATE_CHAT"}}, ae.Details)
	require.Zero(t, r.count(`select count(*) from forum_threads`))
	r.noSideEffects()
	// tiêu đề cũng được kiểm; cả khi `redact` bật
	_, err = r.svc.Create(t.Context(), r.sv, r.course, thread.CreateIn{Title: "em muốn chết", Body: clean, Redact: true})
	require.Equal(t, apierr.CrisisSupport, code(err))
	require.Zero(t, r.count(`select count(*) from forum_threads`))
}

func TestThreadCrisisNoAIAnswer(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.create(r.sv, "Câu hỏi", "Cho em hỏi điểm giữa kỳ. Mà em muốn chết thật rồi", false)
	require.Equal(t, apierr.CrisisSupport, code(err))
	require.Zero(t, r.count(`select count(*) from jobs where kind = 'thread.answer'`))
	require.Zero(t, r.count(`select count(*) from outbox where topic like 'thread.%'`))
}

func TestCommentCrisisNotStored(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	before := r.count(`select count(*) from outbox`)
	_, err := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: crisisText})
	require.Equal(t, apierr.CrisisSupport, code(err))
	require.Zero(t, r.count(`select count(*) from forum_posts where kind = 'HUMAN'`))
	require.Equal(t, before, r.count(`select count(*) from outbox`), "không thông báo cho ai")
	require.Zero(t, r.count(`select count(*) from notifications where type = 'THREAD_REPLY'`))
	// sửa bình luận / bài của mình cũng không lưu
	pid := r.comment(r.other, id)
	_, err = r.svc.EditPost(t.Context(), r.other, r.course, pid, thread.EditPostIn{Body: crisisText, Version: 1})
	require.Equal(t, apierr.CrisisSupport, code(err))
	_, err = r.edit(r.sv, id, "Hỏi", crisisText, 1, false)
	require.Equal(t, apierr.CrisisSupport, code(err))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id = $1 and edited_at is null`, id))
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where id = $1 and body = 'Bình luận thử'`, pid))
}

func TestPrecheckNeverReturnsCrisisSupport(t *testing.T) {
	t.Parallel()
	h := newHTTPRig(t)
	out, err := h.svc.Precheck(t.Context(), h.sv, h.course, "", crisisText)
	require.NoError(t, err, "gõ phím và bấm Đăng dùng cùng một precheck")
	require.True(t, out.Allowed)
	res := h.do(h.sv.UserID, auth.RoleStudent, http.MethodPost, "/courses/"+h.course.String()+"/threads/precheck", `{"title":"","body":"`+crisisText+`"}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	require.NotContains(t, body, "code")
	require.Zero(t, h.count(`select count(*) from pii_events`))
}
