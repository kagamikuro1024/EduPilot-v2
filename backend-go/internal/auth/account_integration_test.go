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
