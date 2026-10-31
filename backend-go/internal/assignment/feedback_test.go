package assignment_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

type capSender struct {
	sent []mail.Mail
	err  error
}

func (c *capSender) Send(_ context.Context, m mail.Mail) error {
	if c.err != nil {
		return c.err
	}
	c.sent = append(c.sent, m)
	return nil
}

func (e *env) mailHandler(s mail.Sender) *mail.Handler {
	vals := map[string]string{"DATABASE_URL": "postgres://x", "REDIS_URL": "redis://x", "JWT_SECRET_KEY": asgSecret, "BLOB_ENDPOINT": "x:1", "BLOB_BUCKET": "x", "BLOB_ACCESS_KEY": "x", "BLOB_SECRET_KEY": "x",
		"APP_ENCRYPTION_KEY": "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=", "APP_ENV": "test", "APP_PUBLIC_URL": "https://edupilot.test"}
	cfg, err := config.Load(func(k string) string { return vals[k] }, config.Worker)
	require.NoError(e.t, err)
	h := &mail.Handler{Pool: e.pool, Clock: clock.Real{}, Sender: s, Cfg: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Resolvers: map[string]mail.Resolver{}}
	assignment.MailResolver{PublicURL: cfg.AppPublicURL}.Register(h.Resolvers)
	return h
}

// published tạo kết quả đã công bố của một sinh viên (email xác minh tuỳ chọn) và chạy handler outbox.
func (e *env) publishedFor(a assignment.Assignment, verified bool, mut func(res uuid.UUID)) (rrow, uuid.UUID) {
	e.t.Helper()
	r := e.reviewed(a)
	stu := e.studentOf(r)
	if verified {
		e.exec(`update users set email_verified_at = now() where id = $1`, stu)
	}
	if mut != nil {
		mut(r.res)
	}
	_, err := e.publish1(r.res)
	require.NoError(e.t, err)
	require.NoError(e.t, e.svc.HandleResultPublished(e.ctx, e.publishedMsg(r.res)))
	return r, stu
}

func (e *env) mailsTo(addr string) (n int) {
	e.one(&n, `select count(*)::int from mail_outbox where to_addr = $1 and template = 'assignment_feedback'`, strings.ToLower(addr))
	return n
}

func (e *env) email(u uuid.UUID) (s string) {
	e.one(&s, `select email from users where id = $1`, u)
	return s
}

func (e *env) sendAll(h *mail.Handler) {
	e.t.Helper()
	rows, err := e.pool.Query(e.ctx, `select id from mail_outbox where template = 'assignment_feedback' and status = 'QUEUED'`)
	require.NoError(e.t, err)
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		require.NoError(e.t, rows.Scan(&id))
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		raw, _ := json.Marshal(map[string]string{"mail_id": id.String()})
		require.NoError(e.t, h.Handle(e.ctx, outbox.Message{ID: uuid.New(), Topic: mail.Topic, Payload: raw}))
	}
}

// TestPublishNotifiesOnlyOwner — AC4: chuông + thư chỉ tới người nộp bài đó.
func TestPublishNotifiesOnlyOwner(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r, owner := e.publishedFor(a, true, nil)
	other := e.student()
	e.exec(`update users set email_verified_at = now() where id = $1`, other)
	var n int
	e.one(&n, `select count(*)::int from notifications where type = 'GRADE_PUBLISHED' and user_id = $1`, owner)
	require.Equal(t, 1, n)
	e.one(&n, `select count(*)::int from notifications where type = 'GRADE_PUBLISHED' and user_id = $1`, other)
	require.Zero(t, n)
	require.Equal(t, 1, e.mailsTo(e.email(owner)))
	require.Zero(t, e.mailsTo(e.email(other)))
	var link, title string
	e.one(&link, `select link from notifications where type = 'GRADE_PUBLISHED' and user_id = $1`, owner)
	e.one(&title, `select title from notifications where type = 'GRADE_PUBLISHED' and user_id = $1`, owner)
	require.Equal(t, "/assignments/"+a.ID.String(), link)
	require.Contains(t, title, "đã được công bố")
	_ = r
}

// TestFeedbackMailContent — AC4: thư có điểm tổng / tối đa, từng tiêu chí (tên, điểm / max, nhận xét), nhận xét chung, dòng Trừ nộp muộn; đường dẫn dựng từ cấu hình lúc gửi.
func TestFeedbackMailContent(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	snd := &capSender{}
	_, owner := e.publishedFor(a, true, func(res uuid.UUID) {
		e.exec(`update grading_results set overall_feedback = 'Bài làm rõ ràng.', late_penalty = 0.5, total = 9.5, late_penalty_manual = true,
			criteria = '[{"id":"c1","score":"4.00","comment":"Đúng định nghĩa."},{"id":"c2","score":"5.50","comment":"Thiếu ví dụ thứ hai."}]'::jsonb where id = $1`, res)
	})
	e.sendAll(e.mailHandler(snd))
	require.Len(t, snd.sent, 1)
	m := snd.sent[0]
	require.Equal(t, strings.ToLower(e.email(owner)), m.To)
	for _, want := range []string{"9,50/10,00", "Khái niệm: 4,00/4,00 — Đúng định nghĩa.", "Ví dụ: 5,50/6,00 — Thiếu ví dụ thứ hai.", "Trừ nộp muộn: −0,50 điểm", "Nhận xét chung: Bài làm rõ ràng.", "https://edupilot.test/assignments/" + a.ID.String()} {
		require.Contains(t, m.Text, want)
	}
	require.NotContains(t, m.Subject, "Người Thử", "tiêu đề không chứa tên người")
}

// TestFeedbackMailNoDraftNoFlags — AC4: payload và thư không có điểm nháp, cờ, độ tin cậy, lượt chấm, chữ AI; payload không có khoá link / token.
func TestFeedbackMailNoDraftNoFlags(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	snd := &capSender{}
	r, _ := e.publishedFor(a, true, func(res uuid.UUID) {
		e.exec(`update grading_results set flags = '{NEEDS_REVIEW,INJECTION_SUSPECTED}', confidence = 0.31,
			pass_detail = '{"c1":{"pass1":"CANARY-PASS1","pass2":"CANARY-PASS2"}}'::jsonb, overall_feedback = 'Nhận xét CANARY-DRAFT-7K3 là của giảng viên.' where id = $1`, res)
	})
	var payload []byte
	e.one(&payload, `select payload from mail_outbox where template = 'assignment_feedback' and dedupe_key = $1`, "assignment_feedback:"+r.res.String())
	for _, bad := range []string{"flags", "confidence", "pass", "ai_", "NEEDS_REVIEW", "INJECTION", "link", "url", "token", "CANARY-PASS", "0.31"} {
		require.NotContains(t, strings.ToLower(string(payload)), strings.ToLower(bad), bad)
	}
	e.sendAll(e.mailHandler(snd))
	require.Len(t, snd.sent, 1)
	full := snd.sent[0].Text + snd.sent[0].HTML + snd.sent[0].Subject
	for _, bad := range []string{"NEEDS_REVIEW", "INJECTION", "CANARY-PASS", "AI ", " AI", "độ tin cậy", "lượt chấm"} {
		require.NotContains(t, full, bad, bad)
	}
}

// TestFeedbackMailToVerifiedEmail — AC4: chỉ gửi tới email ĐÃ XÁC MINH của tài khoản; chưa xác minh → chuông nhưng không thư.
func TestFeedbackMailToVerifiedEmail(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	_, unverified := e.publishedFor(a, false, nil)
	require.Zero(t, e.mailsTo(e.email(unverified)))
	var n int
	e.one(&n, `select count(*)::int from notifications where type = 'GRADE_PUBLISHED' and user_id = $1`, unverified)
	require.Equal(t, 1, n, "chuông vẫn có")
	_, verified := e.publishedFor(a, true, nil)
	require.Equal(t, 1, e.mailsTo(e.email(verified)))
	var to string
	e.one(&to, `select to_addr from mail_outbox where template = 'assignment_feedback' order by created_at desc limit 1`)
	require.Equal(t, strings.ToLower(e.email(verified)), to)
}

// TestFeedbackMailSMTPDownDoesNotBlock — AC4/AC8: SMTP chết → thư thử lại rồi DEAD; điểm vẫn PUBLISHED, chuông vẫn có.
func TestFeedbackMailSMTPDownDoesNotBlock(t *testing.T) {
	e := newEnv(t)
	a := e.create("BT03")
	r, owner := e.publishedFor(a, true, nil)
	h := e.mailHandler(&capSender{err: &mail.SendFailure{Class: "smtp_unreachable"}})
	var id uuid.UUID
	e.one(&id, `select id from mail_outbox where template = 'assignment_feedback' and dedupe_key = $1`, "assignment_feedback:"+r.res.String())
	raw, _ := json.Marshal(map[string]string{"mail_id": id.String()})
	last := len(h.Cfg.OutboxRetryBackoff)
	require.Error(t, h.Handle(e.ctx, outbox.Message{ID: uuid.New(), Payload: raw, Attempts: 0}), "thử lại")
	var st string
	e.one(&st, `select status::text from mail_outbox where id = $1`, id)
	require.Equal(t, "QUEUED", st)
	require.Error(t, h.Handle(e.ctx, outbox.Message{ID: uuid.New(), Payload: raw, Attempts: last}), "lần cuối")
	e.one(&st, `select status::text from mail_outbox where id = $1`, id)
	require.Equal(t, "DEAD", st)
	require.Equal(t, "PUBLISHED", e.status(r.res))
	var n int
	e.one(&n, `select count(*)::int from notifications where type = 'GRADE_PUBLISHED' and user_id = $1`, owner)
	require.Equal(t, 1, n)
}

var _ = time.Now
var _ = store.MailStatusDEAD
