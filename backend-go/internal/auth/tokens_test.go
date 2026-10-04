package auth_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

var b64url43 = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func newDB(t *testing.T) (*pgxpool.Pool, uuid.UUID) {
	t.Helper()
	testutil.RequireContainers(t)
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	u, err := store.New(pool).InsertUser(t.Context(), store.InsertUserParams{Email: "tok@example.test", FullName: "Tok", Role: store.UserRoleSTUDENT, Status: store.UserStatusINVITED})
	require.NoError(t, err)
	return pool, u.ID
}

func issue(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, kind auth.TokenKind) string {
	t.Helper()
	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)
	plain, err := auth.Tokens{Clock: clock.Real{}}.Issue(t.Context(), tx, user, kind, time.Hour, nil)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
	return plain
}

// US-P2-01 AC3: bản rõ 43 ký tự base64url trả đúng một lần, DB chỉ có sha256 hex.
func TestTokenIssueHashed(t *testing.T) {
	pool, uid := newDB(t)
	seen := map[string]bool{}
	for range 20 {
		plain := issue(t, pool, uid, auth.TokenVerifyEmail)
		require.Regexp(t, b64url43, plain)
		require.False(t, seen[plain], "token phải khác nhau")
		seen[plain] = true
		var hash string
		require.NoError(t, pool.QueryRow(t.Context(), `select token_hash from auth_tokens where token_hash = $1`, auth.HashToken(plain)).Scan(&hash))
		require.Len(t, hash, 64)
		require.NotEqual(t, plain, hash)
	}
	var bad int
	require.NoError(t, pool.QueryRow(t.Context(), `select count(*) from auth_tokens where length(token_hash) <> 64`).Scan(&bad))
	require.Zero(t, bad)

	// Hết hạn đặt đúng theo ttl.
	var exp time.Time
	require.NoError(t, pool.QueryRow(t.Context(), `select expires_at from auth_tokens order by created_at desc limit 1`).Scan(&exp))
	require.WithinDuration(t, time.Now().Add(time.Hour), exp, 10*time.Second)

	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(t.Context()) }()
	_, err = auth.Tokens{Clock: clock.Real{}}.Issue(t.Context(), tx, uid, "NOPE", time.Hour, nil)
	require.Error(t, err, "loại lạ bị từ chối")
	_, err = auth.Tokens{Clock: clock.Real{}}.Issue(t.Context(), tx, uid, auth.TokenInvite, 0, nil)
	require.Error(t, err, "ttl ≤ 0 bị từ chối")
}

// US-P2-01 AC3: phát token mới cùng (user, kind) thu hồi token cũ chưa dùng; loại khác không bị đụng.
func TestTokenIssueRevokesPrevious(t *testing.T) {
	pool, uid := newDB(t)
	first := issue(t, pool, uid, auth.TokenResetPassword)
	other := issue(t, pool, uid, auth.TokenVerifyEmail)
	second := issue(t, pool, uid, auth.TokenResetPassword)

	revoked := func(plain string) bool {
		var at *time.Time
		require.NoError(t, pool.QueryRow(t.Context(), `select revoked_at from auth_tokens where token_hash = $1`, auth.HashToken(plain)).Scan(&at))
		return at != nil
	}
	require.True(t, revoked(first))
	require.False(t, revoked(second))
	require.False(t, revoked(other), "VERIFY_EMAIL không bị RESET_PASSWORD thu hồi")
}

// US-P2-01 AC3: bản rõ không nằm ở cột text/jsonb nào của DB và không vào log của consumer thư (cả đường thành công lẫn lỗi).
func TestTokenNeverLogged(t *testing.T) {
	pool, uid := newDB(t)
	smtp := testutil.NewFakeSMTP(t)
	host, port := smtp.Addr()
	var logs bytes.Buffer
	cfg := config.Config{
		AppPublicURL: "https://localhost", SMTPHost: host, SMTPPort: port, SMTPTLS: "none",
		MailFrom: "EduPilot <no-reply@edupilot.local>", MailSendTimeout: 3 * time.Second,
		OutboxRetryBackoff: []time.Duration{time.Second, time.Second, time.Second}, ResetTokenTTL: 30 * time.Minute,
	}
	h := &mail.Handler{Pool: pool, Clock: clock.Real{}, Sender: mail.SMTP{Cfg: cfg}, Cfg: cfg, Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}

	enqueue := func() outbox.Message {
		tx, err := pool.Begin(t.Context())
		require.NoError(t, err)
		id, _, err := mail.Enqueue(t.Context(), tx, mail.Message{To: "tok@example.test", Template: "reset_password", Payload: map[string]any{"user_id": uid.String(), "full_name": "Tok"}})
		require.NoError(t, err)
		require.NoError(t, tx.Commit(t.Context()))
		return outbox.Message{Topic: mail.Topic, Payload: []byte(fmt.Sprintf(`{"mail_id":%q}`, id))}
	}

	smtp.Stop() // đường lỗi trước
	require.Error(t, h.Handle(t.Context(), enqueue()))
	smtp.Start()
	require.NoError(t, h.Handle(t.Context(), enqueue()))

	got := smtp.Delivered()
	require.Len(t, got, 1)
	m := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(strings.ReplaceAll(got[0].Data, "=\r\n", ""))
	require.NotNil(t, m, "thư có token")
	plain := m[1]

	require.NotContains(t, logs.String(), plain)
	require.NotContains(t, logs.String(), "tok@example.test", "log không chứa email")

	rows, err := pool.Query(t.Context(), `select table_name, column_name from information_schema.columns
		where table_schema = 'public' and data_type in ('text','jsonb','json','character','character varying')`)
	require.NoError(t, err)
	type tc struct{ table, col string }
	cols, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (tc, error) { var c tc; return c, r.Scan(&c.table, &c.col) })
	require.NoError(t, err)
	require.NotEmpty(t, cols)
	for _, c := range cols {
		var n int
		q := fmt.Sprintf(`select count(*) from %q where %q::text like '%%'||$1||'%%'`, c.table, c.col)
		require.NoError(t, pool.QueryRow(t.Context(), q, plain).Scan(&n))
		require.Zero(t, n, "bản rõ nằm ở %s.%s", c.table, c.col)
	}
}

// US-P2-01 AC12 + AC11: chỉ gói auth (và store sinh ra) chạm bảng token/phiên; không SQL thô trong auth/mail.
func TestOnlyAuthPackageTouchesTokenTables(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..") // internal/
	tokenRE := regexp.MustCompile(`auth_tokens|auth_sessions|AuthToken|AuthSession`)
	sqlRE := regexp.MustCompile(`(?i)\b(select\s+.+\s+from|insert\s+into|update\s+\w+\s+set|delete\s+from)\b`)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		pkg := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(raw)
		if pkg != "auth" && pkg != "store" {
			require.False(t, tokenRE.MatchString(src), "%s chạm bảng token/phiên ngoài gói auth", rel)
		}
		if pkg == "auth" || pkg == "mail" {
			require.False(t, sqlRE.MatchString(src), "%s có SQL thô (SQL chỉ ở internal/store/queries)", rel)
		}
		return nil
	})
	require.NoError(t, err)
}
