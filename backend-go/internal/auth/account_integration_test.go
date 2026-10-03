//go:build integration

package auth_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/testutil"
)

func mpJSON(t *testing.T, api, path string, out any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api+path, http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
}

// US-P2-03 AC9: đăng ký ⇒ đúng MỘT thư thật tới Mailpit, đúng tiêu đề / liên kết, không chứa mật khẩu hay MSSV.
func TestRegisterSendsVerifyMail(t *testing.T) {
	r := newSessRig(t)
	smtpAddr, api := testutil.Mailpit(t)
	u, err := url.Parse("tcp://" + smtpAddr)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	e := uniq("mp")
	require.Equal(t, http.StatusAccepted, r.register(map[string]string{"email": e, "password": rigPassword, "full_name": "Trần Thu Uyên", "student_code": "20229002"}).code)

	cfg := config.Config{
		AppPublicURL: "https://localhost", SMTPHost: u.Hostname(), SMTPPort: port, SMTPTLS: "none", MailFrom: "EduPilot <no-reply@edupilot.local>",
		MailSendTimeout: 3 * time.Second, VerifyTokenTTL: 24 * time.Hour, ResetTokenTTL: 30 * time.Minute, InviteTokenTTL: 72 * time.Hour,
		OutboxRetryBackoff: []time.Duration{time.Second, time.Second, time.Second},
	}
	h := &mail.Handler{Pool: r.pool, Clock: r.clk, Sender: mail.SMTP{Cfg: cfg}, Cfg: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	var id string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id::text from mail_outbox where to_addr = $1`, e).Scan(&id))
	require.NoError(t, h.Handle(t.Context(), outbox.Message{Topic: mail.Topic, Payload: []byte(`{"mail_id":"` + id + `"}`)}))

	var list struct {
		Messages []struct{ ID string } `json:"messages"`
	}
	mpJSON(t, api, "/api/v1/search?query="+url.QueryEscape("to:"+e), &list)
	require.Len(t, list.Messages, 1)
	var m struct {
		Subject string
		Text    string
		HTML    string
	}
	mpJSON(t, api, "/api/v1/message/"+list.Messages[0].ID, &m)
	require.Equal(t, "Xác minh email EduPilot của bạn", m.Subject)
	require.Regexp(t, regexp.MustCompile(`https://localhost/verify-email\?token=[A-Za-z0-9_-]{43}\b`), m.Text)
	require.Equal(t, 1, strings.Count(m.Text, "token="), "đúng một tham số token")
	require.Contains(t, m.HTML, "<html")
	for _, secret := range []string{rigPassword, "20229002"} {
		require.NotContains(t, m.Text, secret)
		require.NotContains(t, m.HTML, secret)
	}
	for _, bad := range []string{"{{", "<no value>", "%!"} {
		require.NotContains(t, m.Text, bad)
	}
}

// mailpitMail gửi qua SMTP thật mọi thư QUEUED của `to` rồi trả thư (duy nhất) tìm được ở Mailpit theo tiêu đề.
func mailpitMail(t *testing.T, r *sessRig, to, subject string) (text, html string) {
	t.Helper()
	smtpAddr, api := testutil.Mailpit(t)
	u, err := url.Parse("tcp://" + smtpAddr)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	cfg := config.Config{
		AppPublicURL: "https://localhost", SMTPHost: u.Hostname(), SMTPPort: port, SMTPTLS: "none", MailFrom: "EduPilot <no-reply@edupilot.local>",
		MailSendTimeout: 3 * time.Second, VerifyTokenTTL: 24 * time.Hour, ResetTokenTTL: 30 * time.Minute, InviteTokenTTL: 72 * time.Hour,
		OutboxRetryBackoff: []time.Duration{time.Second, time.Second, time.Second},
	}
	h := &mail.Handler{Pool: r.pool, Clock: r.clk, Sender: mail.SMTP{Cfg: cfg}, Cfg: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rows, err := r.pool.Query(t.Context(), `select id::text from mail_outbox where to_addr = $1 and status = 'QUEUED'`, to)
	require.NoError(t, err)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	for _, id := range ids {
		require.NoError(t, h.Handle(t.Context(), outbox.Message{Topic: mail.Topic, Payload: []byte(`{"mail_id":"` + id + `"}`)}))
	}
	var list struct {
		Messages []struct{ ID, Subject string } `json:"messages"`
	}
	mpJSON(t, api, "/api/v1/search?query="+url.QueryEscape("to:"+to+" subject:\""+subject+"\""), &list)
	require.Len(t, list.Messages, 1, subject)
	var m struct{ Text, HTML string }
	mpJSON(t, api, "/api/v1/message/"+list.Messages[0].ID, &m)
	return m.Text, m.HTML
}

// US-P2-04 AC12.
func TestResetMail(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("rm")
	require.Equal(t, http.StatusAccepted, r.forgot(e).code)
	text, html := mailpitMail(t, r, e, "Đặt lại mật khẩu EduPilot")
	require.Regexp(t, regexp.MustCompile(`https://localhost/reset-password\?token=[A-Za-z0-9_-]{43}\b`), text)
	require.Equal(t, 1, strings.Count(text, "token="))
	require.Contains(t, text, "30 phút")
	require.Contains(t, html, "<html")
	for _, secret := range []string{rigPassword, newPassword} {
		require.NotContains(t, text, secret)
		require.NotContains(t, html, secret)
	}
}

func TestPasswordChangedMail(t *testing.T) {
	r := newSessRig(t)
	e := r.activeUser("pm")
	s := r.mustLogin(e)
	require.Equal(t, http.StatusNoContent, r.changePw(s, rigPassword, newPassword).code)
	text, html := mailpitMail(t, r, e, "Mật khẩu EduPilot của bạn đã được đổi")
	require.Regexp(t, `lúc \d{2}:\d{2} \d{2}/\d{2}/\d{4} \(giờ Việt Nam\)`, text)
	require.Contains(t, text, "https://localhost/forgot-password")
	for _, secret := range []string{rigPassword, newPassword} {
		require.NotContains(t, text, secret)
		require.NotContains(t, html, secret)
	}
}
