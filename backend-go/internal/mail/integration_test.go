//go:build integration

package mail_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const beat = 20 * time.Millisecond

type rig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	names   outbox.Streams
	handler *mail.Handler
}

// newRig dựng relay + consumer thật (Postgres + Redis) với handler thư gửi tới SMTP smtpHost:smtpPort.
func newRig(t *testing.T, smtpHost string, smtpPort int, backoff time.Duration) *rig {
	t.Helper()
	ctx := t.Context()
	pool := newPool(t)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	p := testutil.TestPrefix(t)
	names := outbox.Streams{Dispatch: p + ".dispatch", Dead: p + ".dispatch.dead", Consumer: p + "-c1"}
	t.Cleanup(func() { _ = rdb.Del(context.Background(), names.Dispatch, names.Dead).Err() })

	cfg := config.Config{
		InstanceID: p, OutboxPollInterval: beat, OutboxBatch: 100,
		OutboxRetryBackoff: []time.Duration{backoff, backoff, backoff},
		OutboxClaimIdle:    5 * time.Second, OutboxStaleAfter: time.Minute,
		AppPublicURL: "https://localhost", SMTPHost: smtpHost, SMTPPort: smtpPort, SMTPTLS: "none",
		MailFrom: "EduPilot <no-reply@edupilot.local>", MailSendTimeout: 3 * time.Second,
		VerifyTokenTTL: 24 * time.Hour, ResetTokenTTL: 30 * time.Minute, InviteTokenTTL: 72 * time.Hour,
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := &mail.Handler{Pool: pool, Clock: clock.Real{}, Sender: mail.SMTP{Cfg: cfg}, Cfg: cfg, Log: log}
	reg := outbox.NewRegistry()
	reg.Register(mail.Topic, h.Handle)
	d := outbox.Deps{Pool: pool, Redis: rdb, Log: log, Clock: clock.Real{}, Cfg: cfg, Streams: names}

	runCtx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, task := range []interface{ Run(context.Context) error }{outbox.NewRelay(d), outbox.NewConsumer(d, reg)} {
		wg.Add(1)
		go func() { defer wg.Done(); _ = task.Run(runCtx) }()
	}
	t.Cleanup(func() { cancel(); wg.Wait() })
	return &rig{t: t, pool: pool, rdb: rdb, names: names, handler: h}
}

func (r *rig) user(email, name string) uuid.UUID {
	r.t.Helper()
	u, err := store.New(r.pool).InsertUser(r.t.Context(), store.InsertUserParams{Email: email, FullName: name, Role: store.UserRoleSTUDENT, Status: store.UserStatusINVITED})
	require.NoError(r.t, err)
	return u.ID
}

func (r *rig) enqueue(m mail.Message) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	inTx(r.t, r.pool, true, func(tx pgx.Tx) {
		var err error
		id, _, err = mail.Enqueue(r.t.Context(), tx, m)
		require.NoError(r.t, err)
	})
	return id
}

func (r *rig) status(id uuid.UUID) (status string, attempts int, lastErr string) {
	r.t.Helper()
	var le *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text, attempts, last_error from mail_outbox where id = $1`, id).Scan(&status, &attempts, &le))
	if le != nil {
		lastErr = *le
	}
	return
}

func (r *rig) waitStatus(id uuid.UUID, want string) {
	r.t.Helper()
	require.Eventually(r.t, func() bool { s, _, _ := r.status(id); return s == want }, 15*time.Second, 25*time.Millisecond, "mail_outbox → %s", want)
}

func (r *rig) waitAttempts(id uuid.UUID, min int) {
	r.t.Helper()
	require.Eventually(r.t, func() bool { _, a, _ := r.status(id); return a >= min }, 15*time.Second, 10*time.Millisecond)
}

func uniqueAddr(t *testing.T) string { return testutil.TestPrefix(t) + "@example.test" }

// --- Mailpit REST ---

type mpMessage struct {
	Subject string
	Text    string
	HTML    string
	From    struct{ Address string }
	To      []struct{ Address string }
}

func mpGet(t *testing.T, api, path string, out any) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, api+path, http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
}

// mpMessages trả mọi thư gửi tới addr.
func mpMessages(t *testing.T, api, addr string) []mpMessage {
	t.Helper()
	var list struct {
		Messages []struct{ ID string } `json:"messages"`
	}
	mpGet(t, api, "/api/v1/search?query="+url.QueryEscape("to:"+addr), &list)
	out := make([]mpMessage, 0, len(list.Messages))
	for _, m := range list.Messages {
		var full mpMessage
		mpGet(t, api, "/api/v1/message/"+m.ID, &full)
		out = append(out, full)
	}
	return out
}

func mpWait(t *testing.T, api, addr string, n int) []mpMessage {
	t.Helper()
	var got []mpMessage
	require.Eventually(t, func() bool { got = mpMessages(t, api, addr); return len(got) >= n }, 10*time.Second, 50*time.Millisecond)
	return got
}

var tokenRE = regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`)

// US-P2-01 AC5 + AC3: thư thật tới Mailpit; token chỉ nằm trong thư, DB chỉ có băm.
func TestSendViaMailpit(t *testing.T) {
	testutil.RequireContainers(t)
	smtpAddr, api := testutil.Mailpit(t)
	host, portStr, _ := splitHostPort(smtpAddr)
	r := newRig(t, host, portStr, beat)

	to := uniqueAddr(t)
	uid := r.user(to, "Nguyễn Văn An")
	id := r.enqueue(mail.Message{To: to, Template: "verify_email", Payload: map[string]any{"user_id": uid.String(), "full_name": "Nguyễn Văn An"}})
	r.waitStatus(id, "SENT")

	msgs := mpWait(t, api, to, 1)
	require.Len(t, msgs, 1)
	m := msgs[0]
	require.Equal(t, "Xác minh email EduPilot của bạn", m.Subject)
	require.Equal(t, "no-reply@edupilot.local", m.From.Address)
	require.Equal(t, to, m.To[0].Address)
	require.Contains(t, m.Text, "Chào Nguyễn Văn An,")
	require.Contains(t, m.HTML, "<html")
	for _, bad := range []string{"{{", "<no value>", "%!"} {
		require.NotContains(t, m.Text, bad)
		require.NotContains(t, m.HTML, bad)
	}
	mt := tokenRE.FindStringSubmatch(m.Text)
	require.NotNil(t, mt, "liên kết có token 43 ký tự")
	require.Contains(t, m.Text, "https://localhost/verify-email?token="+mt[1])

	var hash string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select token_hash from auth_tokens where user_id = $1 and kind = 'VERIFY_EMAIL' and revoked_at is null`, uid).Scan(&hash))
	require.Equal(t, auth.HashToken(mt[1]), hash)
	var sentAt *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select sent_at from mail_outbox where id = $1`, id).Scan(&sentAt))
	require.NotNil(t, sentAt)
	// Bản rõ không nằm ở mail_outbox / outbox.
	require.Zero(t, count(t, r.pool, `select count(*) from mail_outbox where payload::text like '%'||$1||'%' or coalesce(last_error,'') like '%'||$1||'%'`, mt[1]))
	require.Zero(t, count(t, r.pool, `select count(*) from outbox where payload::text like '%'||$1||'%' or coalesce(last_error,'') like '%'||$1||'%'`, mt[1]))
}

// US-P2-01 AC7: cùng tin giao hai lần (đua) → một thư.
func TestRedeliverOnce(t *testing.T) {
	testutil.RequireContainers(t)
	smtpAddr, api := testutil.Mailpit(t)
	host, port, _ := splitHostPort(smtpAddr)
	r := newRig(t, host, port, beat)

	to := uniqueAddr(t)
	id := r.enqueue(mail.Message{To: to, Template: "email_exists", Payload: map[string]any{"full_name": "Bình"}})
	r.waitStatus(id, "SENT")
	raw, _ := json.Marshal(map[string]string{"mail_id": id.String()})
	var wg sync.WaitGroup
	for range 4 { // giao lại đua nhau sau khi đã SENT (và cả lúc consumer thật còn chạy)
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, r.handler.Handle(t.Context(), outbox.Message{Topic: mail.Topic, Payload: raw}))
		}()
	}
	wg.Wait()
	time.Sleep(300 * time.Millisecond)
	require.Len(t, mpMessages(t, api, to), 1)
}

// US-P2-01 AC6: SMTP chết → thử lại 3 lần rồi DEAD + dead-letter; last_error không có email/token.
func TestRetryThenDead(t *testing.T) {
	testutil.RequireContainers(t)
	smtp := testutil.NewFakeSMTP(t)
	smtp.Stop()
	host, port := smtp.Addr()
	r := newRig(t, host, port, beat)

	to := uniqueAddr(t)
	uid := r.user(to, "Cường")
	id := r.enqueue(mail.Message{To: to, Template: "reset_password", Payload: map[string]any{"user_id": uid.String(), "full_name": "Cường"}})
	r.waitStatus(id, "DEAD")

	_, attempts, lastErr := r.status(id)
	require.Equal(t, 4, attempts)
	require.Equal(t, "smtp_unavailable", lastErr)
	require.NotContains(t, lastErr, "@")
	require.Equal(t, 1, count(t, r.pool, `select count(*) from outbox where topic = 'mail.send' and dead_at is not null and payload->>'mail_id' = $1`, id.String()))
	require.Eventually(t, func() bool { n, _ := r.rdb.XLen(t.Context(), r.names.Dead).Result(); return n == 1 }, 5*time.Second, 20*time.Millisecond)
	require.Empty(t, smtp.Delivered())
	require.Zero(t, count(t, r.pool, `select count(*) from auth_tokens where user_id = $1 and revoked_at is null`, uid), "token phát lúc gửi bị huỷ khi gửi lỗi")
	require.Zero(t, count(t, r.pool, `select count(*) from mail_outbox where last_error ~* '@|token='`))
	require.Zero(t, count(t, r.pool, `select count(*) from outbox where last_error ~* '@|token='`))
}

// US-P2-01 AC6 (nửa sau): bật lại SMTP trước lần thử cuối → thư tới đúng một lần.
func TestRecoverMidRetry(t *testing.T) {
	testutil.RequireContainers(t)
	smtp := testutil.NewFakeSMTP(t)
	smtp.Stop()
	host, port := smtp.Addr()
	r := newRig(t, host, port, 400*time.Millisecond)

	to := uniqueAddr(t)
	id := r.enqueue(mail.Message{To: to, Template: "email_exists", Payload: map[string]any{"full_name": "Dũng"}})
	r.waitAttempts(id, 1)
	smtp.Start()
	r.waitStatus(id, "SENT")
	require.Len(t, smtp.Delivered(), 1)
	_, attempts, lastErr := r.status(id)
	require.Equal(t, 1, attempts, "đã thất bại đúng một lần trước khi hồi phục")
	require.Empty(t, lastErr, "SENT xoá last_error")
}

// US-P2-01 AC10: 550 → DEAD ngay, không thử lại, lỗi không chứa email.
func TestPermanentFailureNoRetry(t *testing.T) {
	testutil.RequireContainers(t)
	smtp := testutil.NewFakeSMTP(t)
	smtp.SetRcptCode(550)
	host, port := smtp.Addr()
	r := newRig(t, host, port, beat)

	to := uniqueAddr(t)
	id := r.enqueue(mail.Message{To: to, Template: "email_exists", Payload: map[string]any{"full_name": "Em"}})
	r.waitStatus(id, "DEAD")
	_, attempts, lastErr := r.status(id)
	require.Equal(t, 1, attempts)
	require.Equal(t, "smtp_permanent_550", lastErr)
	require.NotContains(t, lastErr, to)
	// outbox coi là đã xử lý (không thử lại, không dead-letter).
	require.Eventually(t, func() bool {
		return count(t, r.pool, `select count(*) from outbox where topic='mail.send' and dispatched_at is not null and payload->>'mail_id' = $1`, id.String()) == 1
	}, 5*time.Second, 20*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	require.Empty(t, smtp.Delivered())

	// Mẫu không tồn tại, thiếu biến → DEAD ngay, không chạm SMTP.
	for name, m := range map[string]mail.Message{
		"mẫu lạ":    {To: uniqueAddr(t), Template: "khong_co", Payload: map[string]any{"full_name": "X"}},
		"thiếu biến": {To: uniqueAddr(t), Template: "email_exists", Payload: map[string]any{}},
	} {
		mid := r.enqueue(m)
		r.waitStatus(mid, "DEAD")
		_, a, le := r.status(mid)
		require.Equal(t, 1, a, name)
		require.NotEmpty(t, le, name)
	}
	require.Empty(t, smtp.Delivered())
}

// US-P2-01 AC10: 451 là tạm thời → được thử lại và cuối cùng gửi được.
func TestTemporaryFailureRetries(t *testing.T) {
	testutil.RequireContainers(t)
	smtp := testutil.NewFakeSMTP(t)
	smtp.SetRcptCode(451)
	host, port := smtp.Addr()
	r := newRig(t, host, port, 300*time.Millisecond)

	id := r.enqueue(mail.Message{To: uniqueAddr(t), Template: "email_exists", Payload: map[string]any{"full_name": "Giang"}})
	r.waitAttempts(id, 1)
	s, _, lastErr := r.status(id)
	require.Equal(t, "QUEUED", s, "4xx không phải lỗi vĩnh viễn")
	require.Equal(t, "smtp_temporary_451", lastErr)
	smtp.SetRcptCode(250)
	r.waitStatus(id, "SENT")
	require.Len(t, smtp.Delivered(), 1)
}

func splitHostPort(hp string) (string, int, error) {
	u, err := url.Parse("tcp://" + hp)
	if err != nil {
		return "", 0, fmt.Errorf("parse %q: %w", hp, err)
	}
	var port int
	_, err = fmt.Sscanf(u.Port(), "%d", &port)
	return u.Hostname(), port, err
}
