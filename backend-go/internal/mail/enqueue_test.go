package mail_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/testutil"
)

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(t.Context(), sql, args...).Scan(&n))
	return n
}

func inTx(t *testing.T, pool *pgxpool.Pool, commit bool, fn func(tx pgx.Tx)) {
	t.Helper()
	tx, err := pool.Begin(t.Context())
	require.NoError(t, err)
	fn(tx)
	if commit {
		require.NoError(t, tx.Commit(context.WithoutCancel(t.Context())))
		return
	}
	require.NoError(t, tx.Rollback(context.WithoutCancel(t.Context())))
}

// US-P2-01 AC4: rollback → không còn dòng nào; commit → đúng một dòng mỗi loại.
func TestEnqueueRollback(t *testing.T) {
	testutil.RequireContainers(t)
	pool := newPool(t)
	inTx(t, pool, false, func(tx pgx.Tx) {
		_, queued, err := mail.Enqueue(t.Context(), tx, mail.Message{To: "a@example.test", Template: "email_exists", Payload: map[string]any{"full_name": "A"}})
		require.NoError(t, err)
		require.True(t, queued)
	})
	require.Zero(t, count(t, pool, `select count(*) from mail_outbox`))
	require.Zero(t, count(t, pool, `select count(*) from outbox where topic = 'mail.send'`))
}

func TestEnqueueCommit(t *testing.T) {
	testutil.RequireContainers(t)
	pool := newPool(t)
	var id string
	inTx(t, pool, true, func(tx pgx.Tx) {
		got, queued, err := mail.Enqueue(t.Context(), tx, mail.Message{To: " A@Example.Test ", Template: "email_exists", Payload: map[string]any{"full_name": "A"}})
		require.NoError(t, err)
		require.True(t, queued)
		id = got.String()
	})
	require.Equal(t, 1, count(t, pool, `select count(*) from mail_outbox where to_addr = 'a@example.test' and status = 'QUEUED'`), "địa chỉ được chuẩn hoá chữ thường")
	require.Equal(t, 1, count(t, pool, `select count(*) from outbox where topic = 'mail.send' and payload = jsonb_build_object('mail_id', $1::text)`, id))
}

// US-P2-01 AC7: dedupe_key chặn xếp hai thư cùng loại cho cùng sự kiện.
func TestDedupeKey(t *testing.T) {
	testutil.RequireContainers(t)
	pool := newPool(t)
	m := mail.Message{To: "a@example.test", Template: "account_locked", DedupeKey: "account_locked:u1:2026-10-03T10:00:00Z", Payload: map[string]any{"full_name": "A"}}
	for i, want := range []bool{true, false} {
		inTx(t, pool, true, func(tx pgx.Tx) {
			_, queued, err := mail.Enqueue(t.Context(), tx, m)
			require.NoError(t, err)
			require.Equal(t, want, queued, "lần %d", i+1)
		})
	}
	require.Equal(t, 1, count(t, pool, `select count(*) from mail_outbox`))
	require.Equal(t, 1, count(t, pool, `select count(*) from outbox where topic = 'mail.send'`), "lần trùng không ghi outbox")
}

// US-P2-01 AC9: bí mật không được nằm ở hàng đợi — Enqueue từ chối khoá nghi là bí mật, và không dòng nào chứa chúng.
func TestPayloadHasNoSecrets(t *testing.T) {
	testutil.RequireContainers(t)
	pool := newPool(t)
	for _, k := range []string{"token", "Token", "reset_token", "password", "link", "verify_url", "secret"} {
		inTx(t, pool, false, func(tx pgx.Tx) {
			_, _, err := mail.Enqueue(t.Context(), tx, mail.Message{To: "a@example.test", Template: "verify_email", Payload: map[string]any{k: "x"}})
			require.ErrorIs(t, err, mail.ErrSecretInPayload, k)
		})
	}
	inTx(t, pool, true, func(tx pgx.Tx) {
		_, _, err := mail.Enqueue(t.Context(), tx, mail.Message{To: "a@example.test", Template: "verify_email", Payload: map[string]any{"user_id": "u", "full_name": "A"}})
		require.NoError(t, err)
	})
	require.Zero(t, count(t, pool, `select count(*) from mail_outbox where payload::text ~* '"[^"]*(token|password|link|url)[^"]*"\s*:'`))
}
