package chat_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/llm"
)

// TestChatLockedDuringExam / TestLockCheckBeforeAnything / TestLockedWritesChatBlockedEvent / TestLockerErrorDeniesChat / TestUnlockedAfterSubmit — AC9.
func TestChatLockedDuringExam(t *testing.T) {
	t.Parallel()
	r := newRig(t, func(c *chat.Config) { c.RatePerMin = 1 }) // giới hạn tốc độ 1: khoá phải chạy TRƯỚC nó
	att := uuid.New()
	until := time.Now().Add(30 * time.Minute).UTC()
	r.lock.lock = &exam.Lock{AttemptID: att, Until: until}
	for range 3 {
		_, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "hi")
		require.Equal(t, 409, httpStatus(err))
		require.Equal(t, apierr.ExamInProgress, apiCode(err))
		var ae *apierr.Error
		require.True(t, errors.As(err, &ae))
		require.Equal(t, until.Format(time.RFC3339), ae.Details.(map[string]any)["until"])
	}
	require.Zero(t, r.count(`select count(*) from chat_messages where session_id=$1`, r.sess), "không tạo hàng chat_messages")
	require.Zero(t, r.ag.calls.Load(), "không gọi provider / embed")
	require.Len(t, r.lock.blocked, 3)
	require.Equal(t, att, r.lock.blocked[0])

	// retry cũng bị khoá
	_, mid := r.orphan(r.student)
	_, err := r.pool.Exec(t.Context(), `update chat_messages set stream_status='FAILED', error_code='X', completed_at=now() where id=$1`, mid)
	require.NoError(t, err)
	_, err = r.svc.Retry(t.Context(), r.student, mid)
	require.Equal(t, apierr.ExamInProgress, apiCode(err))

	// nộp bài → mở lại
	r.lock.lock = nil
	_, _, err = r.send(r.student, r.sess, "tiếp")
	require.NoError(t, err)
}

func TestLockerErrorDeniesChat(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.lock.err = errors.New("db chết")
	_, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "hi")
	require.Equal(t, 503, httpStatus(err))
	require.Equal(t, apierr.ChatUnavailable, apiCode(err))
	require.Zero(t, r.ag.calls.Load())
}

// TestSendIdempotentReplay / TestSendBusy409 / TestSend20ParallelSameKey / TestBusyKeyReleasedOnEveryTerminalState — AC11.
func TestSendIdempotentReplay(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	key := uuid.New()
	m1, err := r.svc.Send(t.Context(), r.student, r.sess, key, "xin chào")
	require.NoError(t, err)
	r.svc.Wait()
	m2, err := r.svc.Send(t.Context(), r.student, r.sess, key, "xin chào")
	require.NoError(t, err)
	require.Equal(t, m1.ID, m2.ID)
	require.Equal(t, int32(1), r.ag.calls.Load(), "không sinh lại")
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where session_id=$1 and role='USER'`, r.sess))
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where session_id=$1 and role='ASSISTANT'`, r.sess))
	c := newCol()
	require.NoError(t, r.svc.Tail(t.Context(), c, m2, ""))
	require.Equal(t, "Xin chào bạn.", c.text(), "phát lại cùng luồng")
}

func TestSendBusy409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "x")}, nil
	}
	first := uuid.New()
	_, err := r.svc.Send(t.Context(), r.student, r.sess, first, "một")
	require.NoError(t, err)
	_, err = r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "hai")
	require.Equal(t, 409, httpStatus(err))
	require.Equal(t, apierr.ChatBusy, apiCode(err))
	v, err := r.rdb.Get(t.Context(), "ep:chat:active:"+r.student.UserID.String()).Result()
	require.NoError(t, err)
	require.Equal(t, first.String(), v, "giá trị khoá = client_msg_id")
	// gửi lặp CÙNG khoá khi bản đầu còn chạy → phát lại, không 409
	_, err = r.svc.Send(t.Context(), r.student, r.sess, first, "một")
	require.NoError(t, err)
	close(gate)
	r.svc.Wait()
	_, err = r.rdb.Get(t.Context(), "ep:chat:active:"+r.student.UserID.String()).Result()
	require.Error(t, err, "khoá nhả khi DONE")
}

func TestSend20ParallelSameKey(t *testing.T) {
	t.Parallel()
	r := newRig(t, func(c *chat.Config) { c.RatePerMin = 100 })
	key := uuid.New()
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, 20)
	errs := make([]error, 20)
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := r.svc.Send(t.Context(), r.student, r.sess, key, "song song")
			ids[i], errs[i] = m.ID, err
		}()
	}
	wg.Wait()
	r.svc.Wait()
	for i := range 20 {
		require.NoError(t, errs[i])
		require.Equal(t, ids[0], ids[i])
	}
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where session_id=$1 and role='USER'`, r.sess))
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where session_id=$1 and role='ASSISTANT'`, r.sess))
	require.Equal(t, int32(1), r.ag.calls.Load())
}

func TestBusyKeyReleasedOnEveryTerminalState(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	busy := func() bool {
		_, err := r.rdb.Get(t.Context(), "ep:chat:active:"+r.student.UserID.String()).Result()
		return err == nil
	}
	// DONE
	_, _, err := r.send(r.student, r.sess, "một")
	require.NoError(t, err)
	r.svc.Wait()
	require.False(t, busy())
	// FAILED
	r.ag.fn = func(context.Context, agent.TrustedContext, agent.Input) (agent.Outcome, error) {
		return agent.Outcome{}, errors.New("hỏng")
	}
	_, _, err = r.send(r.student, r.sess, "hai")
	require.NoError(t, err)
	r.svc.Wait()
	require.False(t, busy())
	// CANCELLED (G sống)
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "x")}, nil
	}
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "ba")
	require.NoError(t, err)
	require.True(t, busy())
	require.NoError(t, r.svc.Cancel(t.Context(), r.student, m.ID))
	r.svc.Wait()
	require.False(t, busy())
}

// TestSendValidation / TestSendRateLimit / TestSendRateLimitRetryAfter / TestSendArchivedCourse — AC12.
func TestSendValidation(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, s := range []string{"", "   \n\t "} {
		_, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), s)
		require.Equal(t, 422, httpStatus(err), s)
	}
	_, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), strings.Repeat("a", 4001))
	require.Equal(t, 422, httpStatus(err))
	require.Equal(t, apierr.MessageTooLong, apiCode(err))
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, 4000, ae.Details.(map[string]any)["limit"])
	_, err = r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), strings.Repeat("a", 4000))
	require.NoError(t, err)
	// phiên của người khác / đã xoá → 404
	_, err = r.svc.Send(t.Context(), r.other, r.sess, uuid.New(), "hi")
	require.Equal(t, 404, httpStatus(err))
	r.svc.Wait()
	require.NoError(t, r.svc.DeleteSession(t.Context(), r.student, r.sess))
	_, err = r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "hi")
	require.Equal(t, 404, httpStatus(err))
}

func TestSendRateLimit(t *testing.T) {
	t.Parallel()
	r := newRig(t, func(c *chat.Config) { c.RatePerMin = 3 })
	for range 3 {
		_, _, err := r.send(r.student, r.sess, "x")
		require.NoError(t, err)
		r.svc.Wait()
	}
	_, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "x")
	require.Equal(t, 429, httpStatus(err))
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, apierr.RateLimited, ae.Code)
	require.GreaterOrEqual(t, ae.RetryAfter, 1)
	require.LessOrEqual(t, ae.RetryAfter, 60)
}

func TestSendArchivedCourse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.pool.Exec(t.Context(), `update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, r.course)
	require.NoError(t, err)
	_, err = r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "hi")
	require.Equal(t, apierr.CourseArchived, apiCode(err))
	_, err = r.svc.CreateSession(t.Context(), r.student, chat.CreateSessionIn{CourseID: r.course})
	require.Equal(t, apierr.CourseArchived, apiCode(err))
}

// TestNoticeMasked / TestNoticeMaskedCurrentOnly / TestPIIEventsMaskedWritten — AC13.
func TestNoticeMasked(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, nil, time.Millisecond, llm.Response{MaskedCurrent: 2}, "ok")}, nil
	}
	c, mid, err := r.send(r.student, r.sess, "mail em a.b@sv.edu.vn, số 0912345678")
	require.NoError(t, err)
	var notices []frame
	for _, f := range c.all() {
		if f.Ev == chat.EvNotice {
			notices = append(notices, f)
		}
	}
	require.Len(t, notices, 1)
	require.EqualValues(t, 2, notices[0].D["masked"])
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id=$1 and masked_count=2`, mid))
	// pii_events: một dòng MASKED mỗi loại, chỉ số đếm
	require.Equal(t, 2, r.count(`select count(*) from pii_events where user_id=$1 and action='MASKED' and session_id=$2`, r.student.UserID, r.sess))
	require.Equal(t, 1, r.count(`select count(*) from pii_events where user_id=$1 and pii_type='EMAIL' and count=1`, r.student.UserID))
	require.Equal(t, 1, r.count(`select count(*) from pii_events where user_id=$1 and pii_type='PHONE' and count=1`, r.student.UserID))
}

func TestNoticeMaskedCurrentOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, nil, time.Millisecond, llm.Response{MaskedCurrent: 0}, "ok")}, nil
	}
	c, _, err := r.send(r.student, r.sess, "tiếp tục nhé")
	require.NoError(t, err)
	require.NotContains(t, c.events(), chat.EvNotice, "n = 0 thì không có dòng")
	require.Zero(t, r.count(`select count(*) from pii_events where user_id=$1`, r.student.UserID))
}

// TestCitationHistoryOnlyMine: lịch sử đưa vào agent chỉ gồm tin ĐÃ XONG của chính phiên, theo thứ tự thời gian.
func TestHistoryPassedToAgent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, _, err := r.send(r.student, r.sess, "câu một")
	require.NoError(t, err)
	r.svc.Wait()
	var got []llm.Message
	r.ag.fn = func(ctx context.Context, tc agent.TrustedContext, in agent.Input) (agent.Outcome, error) {
		got = in.History
		require.Equal(t, r.student.UserID, tc.UserID, "danh tính từ Actor")
		require.Equal(t, r.course, tc.CourseID, "lớp lấy từ phiên")
		require.Equal(t, r.sess, tc.SessionID)
		return agent.Outcome{Plan: agent.IntentSmalltalk, Canned: "ok"}, nil
	}
	_, _, err = r.send(r.student, r.sess, "câu hai")
	require.NoError(t, err)
	require.Equal(t, []llm.Message{{Role: "user", Content: "câu một"}, {Role: "assistant", Content: "Xin chào bạn."}}, got)
}

// TestFeedbackOwnerOnly — AC16.
func TestFeedbackOwnerOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid, err := r.send(r.student, r.sess, "chào")
	require.NoError(t, err)
	v := "HELPFUL"
	require.Equal(t, 404, httpStatus(r.svc.Feedback(t.Context(), r.other, mid, &v)))
	require.NoError(t, r.svc.Feedback(t.Context(), r.student, mid, &v))
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id=$1 and feedback='HELPFUL'`, mid))
	require.NoError(t, r.svc.Feedback(t.Context(), r.student, mid, &v), "bấm lại cùng giá trị = bỏ")
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id=$1 and feedback is null`, mid))
	bad := "MAYBE"
	require.Equal(t, 422, httpStatus(r.svc.Feedback(t.Context(), r.student, mid, &bad)))
}

// TestSessionsList / TestSessionDocumentScope / TestSessionSoftDeleteRestore — AC17.
func TestSessionsList(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for range 4 {
		r.newSession(r.student, r.course)
	}
	r.newSession(r.other, r.course)
	p1, err := r.svc.ListSessions(t.Context(), r.student, r.course, pageOf(3))
	require.NoError(t, err)
	require.Len(t, p1.Items, 3)
	require.NotNil(t, p1.NextCursor)
	seen := map[uuid.UUID]bool{}
	for _, s := range p1.Items {
		seen[s.ID] = true
	}
	pg := pageOf(3)
	cur := cursorOf(t, *p1.NextCursor)
	pg.Cursor = &cur
	p2, err := r.svc.ListSessions(t.Context(), r.student, r.course, pg)
	require.NoError(t, err)
	require.Len(t, p2.Items, 2, "5 phiên của mình (1 từ rig + 4), không có phiên của người khác")
	for _, s := range p2.Items {
		require.False(t, seen[s.ID])
	}
	require.Nil(t, p2.NextCursor)
}

func TestSessionDocumentScope(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	doc := func(status string, visible bool, typ string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, r.pool.QueryRow(t.Context(), `insert into documents (course_id, title, type, status, visible_to_students) values ($1, 'T', $2::document_type, $3::document_status, $4) returning id`, r.course, typ, status, visible).Scan(&id))
		return id
	}
	ok := doc("READY", true, "LECTURE")
	s, err := r.svc.CreateSession(t.Context(), r.student, chat.CreateSessionIn{CourseID: r.course, DocumentID: &ok})
	require.NoError(t, err)
	require.Equal(t, ok, *s.DocumentID)
	for name, id := range map[string]uuid.UUID{"chưa READY": doc("QUEUED", true, "LECTURE"), "ẩn với sinh viên": doc("READY", false, "LECTURE"), "đáp án": doc("READY", false, "ANSWER_KEY"), "không tồn tại": uuid.New()} {
		_, err := r.svc.CreateSession(t.Context(), r.student, chat.CreateSessionIn{CourseID: r.course, DocumentID: &id})
		require.Equal(t, 404, httpStatus(err), name)
	}
	// tài liệu của lớp khác
	other := r.newCourse(r.user("GV2", "TEACHER"), "ACTIVE")
	var foreign uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into documents (course_id, title, status) values ($1, 'T', 'READY') returning id`, other).Scan(&foreign))
	_, err = r.svc.CreateSession(t.Context(), r.student, chat.CreateSessionIn{CourseID: r.course, DocumentID: &foreign})
	require.Equal(t, 404, httpStatus(err))
}

func TestSessionSoftDeleteRestore(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	require.NoError(t, r.svc.DeleteSession(t.Context(), r.student, r.sess))
	l, err := r.svc.ListSessions(t.Context(), r.student, r.course, pageOf(30))
	require.NoError(t, err)
	require.Empty(t, l.Items)
	_, err = r.svc.Messages(t.Context(), r.student, r.sess, pageOf(30))
	require.Equal(t, 404, httpStatus(err))
	_, err = r.svc.RestoreSession(t.Context(), r.other, r.sess)
	require.Equal(t, 404, httpStatus(err))
	_, err = r.svc.RestoreSession(t.Context(), r.student, r.sess)
	require.NoError(t, err)
	l, _ = r.svc.ListSessions(t.Context(), r.student, r.course, pageOf(30))
	require.Len(t, l.Items, 1)
}

// TestChatRoleMatrix — AC18 (tầng service): vai sai / ngoài lớp / PENDING / REMOVED → 403; sinh viên khác → 404.
func TestChatRoleMatrix(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	staff := func(role auth.Role, global string) chat.Actor {
		return chat.Actor{UserID: r.user("X", global), Role: role}
	}
	ta, teacher, admin := staff(auth.RoleTA, "TA"), staff(auth.RoleTeacher, "TEACHER"), staff(auth.RoleAdmin, "ADMIN")
	r.enroll(r.course, ta.UserID, "TA", "ACTIVE")
	pending := chat.Actor{UserID: r.user("P", "STUDENT"), Role: auth.RoleStudent}
	r.enroll(r.course, pending.UserID, "STUDENT", "PENDING")
	removed := chat.Actor{UserID: r.user("R", "STUDENT"), Role: auth.RoleStudent}
	r.enroll(r.course, removed.UserID, "STUDENT", "REMOVED")
	outside := chat.Actor{UserID: r.user("O", "STUDENT"), Role: auth.RoleStudent}
	_, _, err := r.send(r.student, r.sess, "chào")
	require.NoError(t, err)
	var mid uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id from chat_messages where session_id=$1 and role='ASSISTANT'`, r.sess).Scan(&mid))

	for name, a := range map[string]chat.Actor{"TA": ta, "TEACHER": teacher, "ADMIN": admin, "PENDING": pending, "REMOVED": removed, "ngoài lớp": outside} {
		want := 403
		_, e1 := r.svc.ListSessions(t.Context(), a, r.course, pageOf(10))
		_, e2 := r.svc.Messages(t.Context(), a, r.sess, pageOf(10))
		_, e3 := r.svc.CreateSession(t.Context(), a, chat.CreateSessionIn{CourseID: r.course})
		_, e4 := r.svc.Send(t.Context(), a, r.sess, uuid.New(), "hi")
		e5 := r.svc.Cancel(t.Context(), a, mid)
		e6 := r.svc.Feedback(t.Context(), a, mid, nil)
		e7 := r.svc.DeleteSession(t.Context(), a, r.sess)
		for i, e := range []error{e1, e2, e3, e4, e5, e6, e7} {
			if a.Role == auth.RoleStudent && (i == 1 || i == 3 || i == 4 || i == 5 || i == 6) {
				require.Equal(t, 404, httpStatus(e), "%s thao tác %d: không lộ tồn tại", name, i)
				continue
			}
			require.Equal(t, want, httpStatus(e), "%s thao tác %d", name, i)
		}
	}
	require.Zero(t, r.ag.calls.Load()-1, "mọi lời gọi trái phép không tới agent")
}

// TestFromDraftNoMessageStored — US-P3-06 AC5: phiên mới không có tin nhắn; nháp không được lưu ở bất kỳ đâu phía máy chủ (không forum_*, chat_messages, tiêu đề phiên, outbox, jobs).
func TestFromDraftNoMessageStored(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	body := "MSSV 20229001 chữ-nháp-duy-nhất-9f3"
	out, err := r.svc.FromDraft(t.Context(), r.student, chat.FromDraftIn{CourseID: r.course, Title: "tiêu-đề-nháp-9f3", Body: body})
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from chat_messages where session_id=$1`, out.SessionID))
	require.Zero(t, r.count(`select count(*) from forum_threads`)+r.count(`select count(*) from forum_posts`))
	for _, tbl := range []string{"chat_sessions", "outbox", "jobs", "audit_log", "pii_events"} {
		require.Zero(t, r.count(`select count(*) from `+tbl+` t where t::text like '%9f3%'`), tbl)
	}
}

// TestFromDraftKeepsText / OnSwitched — US-P3-06 AC5: phiên PRIVATE không có tin nhắn; bản nháp trả lại ĐÚNG TỪNG BYTE (kể cả xuống dòng); không lưu nháp ở máy chủ; hook SWITCHED được gọi một lần.
func TestFromDraftKeepsText(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	var got []string
	r.svc.OnSwitched = func(_ context.Context, uid, cid uuid.UUID, title, body string) {
		require.Equal(t, r.student.UserID, uid)
		require.Equal(t, r.course, cid)
		got = append(got, title, body)
	}
	body := "Dòng một có MSSV 20229001\r\n\r\n  Dòng ba, thụt lề\t và ký tự lạ: ủ̉ ☃ [[x]] \n"
	out, err := r.svc.FromDraft(t.Context(), r.student, chat.FromDraftIn{CourseID: r.course, Title: " Tiêu đề  ", Body: body})
	require.NoError(t, err)
	require.Equal(t, body, out.Draft.Body, "đúng từng byte")
	require.Equal(t, " Tiêu đề  ", out.Draft.Title)
	require.Equal(t, 1, r.count(`select count(*) from chat_sessions where id=$1 and user_id=$2 and channel='PRIVATE'`, out.SessionID, r.student.UserID))
	require.Equal(t, []string{" Tiêu đề  ", body}, got)
	// phân quyền: người ngoài lớp / vai sai
	_, err = r.svc.FromDraft(t.Context(), chat.Actor{UserID: r.user("X", "STUDENT"), Role: auth.RoleStudent}, chat.FromDraftIn{CourseID: r.course, Body: "x"})
	require.Equal(t, 403, httpStatus(err))
	_, err = r.svc.FromDraft(t.Context(), r.student, chat.FromDraftIn{CourseID: r.course, Body: "   "})
	require.Equal(t, 422, httpStatus(err))
}
