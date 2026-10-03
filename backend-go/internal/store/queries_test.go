package store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// TestQueries_Generated kiểm mã sqlc sinh ra chạy thật trên schema 00001: ánh xạ enum, uuid, time.Time,
// jsonb; outbox lấy dòng tới hạn + backoff + dead; jobs tiến độ chỉ tăng + phân trang con trỏ;
// idempotency_keys ON CONFLICT DO NOTHING. Đây là hợp đồng cho các gói dùng lại (outbox, jobs, httpapi).
func TestQueries_Generated(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	q := store.New(conn)

	user, err := q.InsertUser(ctx, store.InsertUserParams{
		Email: "qa@x.com", FullName: "QA", Role: store.UserRoleSTUDENT, Status: store.UserStatusINVITED,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, store.UserStatusINVITED, user.Status)
	require.Nil(t, user.PasswordHash)
	require.WithinDuration(t, time.Now(), user.CreatedAt, time.Minute)

	got, err := q.GetUserByEmail(ctx, "qa@x.com")
	require.NoError(t, err)
	require.Equal(t, user.ID, got.ID)

	audit, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{
		ActorID: &user.ID, Entity: "user", EntityID: user.ID.String(), Action: "create",
		After: []byte(`{"role":"STUDENT"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "create", audit.Action)

	// --- outbox ---
	row, err := q.InsertOutbox(ctx, store.InsertOutboxParams{Topic: "test.ok", Payload: []byte(`{"a":1}`)})
	require.NoError(t, err)
	due, err := q.SelectDueOutbox(ctx, 10)
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.Equal(t, row.ID, due[0].ID)

	n, err := q.MarkOutboxEnqueued(ctx, row.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	due, err = q.SelectDueOutbox(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, due, "dòng đã xếp hàng không được lấy lại")

	stale, err := q.RequeueStaleOutbox(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{row.ID}, stale)

	attempts, err := q.MarkOutboxFailed(ctx, store.MarkOutboxFailedParams{
		ID: row.ID, LastError: "boom", BackoffMs: 50,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, attempts)

	n, err = q.MarkOutboxDispatched(ctx, row.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	after, err := q.GetOutbox(ctx, row.ID)
	require.NoError(t, err)
	require.NotNil(t, after.DispatchedAt)
	require.Nil(t, after.DeadAt)

	dead, err := q.InsertOutbox(ctx, store.InsertOutboxParams{Topic: "test.fail", Payload: []byte(`{}`)})
	require.NoError(t, err)
	n, err = q.MarkOutboxDead(ctx, store.MarkOutboxDeadParams{ID: dead.ID, LastError: "cháy"})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// --- jobs ---
	job, err := q.InsertJob(ctx, store.InsertJobParams{Kind: "test.progress", OwnerID: user.ID})
	require.NoError(t, err)
	require.Equal(t, store.JobStatusQUEUED, job.Status)

	job, err = q.UpdateJobProgress(ctx, store.UpdateJobProgressParams{ID: job.ID, Progress: 40})
	require.NoError(t, err)
	require.EqualValues(t, 40, job.Progress)
	require.Equal(t, store.JobStatusRUNNING, job.Status)

	job, err = q.UpdateJobProgress(ctx, store.UpdateJobProgressParams{ID: job.ID, Progress: 10})
	require.NoError(t, err)
	require.EqualValues(t, 40, job.Progress, "tiến độ chỉ được tăng")

	job, err = q.MarkJobSucceeded(ctx, store.MarkJobSucceededParams{ID: job.ID, Result: []byte(`{"ok":true}`)})
	require.NoError(t, err)
	require.Equal(t, store.JobStatusSUCCEEDED, job.Status)
	require.EqualValues(t, 100, job.Progress)
	require.NotNil(t, job.FinishedAt)

	failing, err := q.InsertJob(ctx, store.InsertJobParams{Kind: "test.fail", OwnerID: user.ID})
	require.NoError(t, err)
	failed, err := q.MarkJobFailed(ctx, store.MarkJobFailedParams{ID: failing.ID, Error: []byte(`{"code":"X"}`)})
	require.NoError(t, err)
	require.Equal(t, store.JobStatusFAILED, failed.Status)

	page, err := q.ListJobsByOwner(ctx, store.ListJobsByOwnerParams{OwnerID: user.ID, RowLimit: 1})
	require.NoError(t, err)
	require.Len(t, page, 1)
	next, err := q.ListJobsByOwner(ctx, store.ListJobsByOwnerParams{
		OwnerID: user.ID, CursorCreatedAt: &page[0].CreatedAt, CursorID: &page[0].ID, RowLimit: 10,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	require.NotEqual(t, page[0].ID, next[0].ID)

	// Việc của người khác không thấy được.
	_, err = q.GetJobForOwner(ctx, store.GetJobForOwnerParams{ID: job.ID, OwnerID: uuid.New()})
	require.Error(t, err)

	// --- idempotency_keys ---
	args := store.InsertIdempotencyKeyParams{
		UserID: user.ID, Endpoint: "POST /api/v1/_test/items", Key: "k-12345678",
		RequestHash: "deadbeef", StatusCode: 201, Response: []byte(`{"id":1}`),
	}
	n, err = q.InsertIdempotencyKey(ctx, args)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	n, err = q.InsertIdempotencyKey(ctx, args)
	require.NoError(t, err, "chèn trùng không được lỗi")
	require.EqualValues(t, 0, n)

	saved, err := q.GetIdempotencyKey(ctx, store.GetIdempotencyKeyParams{
		UserID: args.UserID, Endpoint: args.Endpoint, Key: args.Key,
	})
	require.NoError(t, err)
	require.EqualValues(t, 201, saved.StatusCode)
	require.JSONEq(t, `{"id":1}`, string(saved.Response))
}
