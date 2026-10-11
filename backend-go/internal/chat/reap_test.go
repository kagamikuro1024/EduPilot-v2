package chat_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/llm"
)

func (r *rig) backdate(mid uuid.UUID, age time.Duration) {
	r.t.Helper()
	tx, err := r.pool.Begin(r.t.Context())
	require.NoError(r.t, err)
	defer func() { _ = tx.Rollback(r.t.Context()) }()
	_, err = tx.Exec(r.t.Context(), `alter table chat_messages disable trigger chat_messages_set_updated_at`)
	require.NoError(r.t, err)
	_, err = tx.Exec(r.t.Context(), `update chat_messages set updated_at = now() - $2::interval where id=$1`, mid, age.String())
	require.NoError(r.t, err)
	_, err = tx.Exec(r.t.Context(), `alter table chat_messages enable trigger chat_messages_set_updated_at`)
	require.NoError(r.t, err)
	require.NoError(r.t, tx.Commit(r.t.Context()))
}

// TestReapInterrupted — AC19: quá 150 s không cập nhật → FAILED INTERRUPTED, giữ partial, nhả CHAT_BUSY.
func TestReapInterrupted(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	user, mid := r.orphan(r.student)
	var cm uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select client_msg_id from chat_messages where id=$1`, user).Scan(&cm))
	require.NoError(t, r.rdb.Set(t.Context(), "ep:chat:active:"+r.student.UserID.String(), cm.String(), time.Minute).Err())
	r.backdate(mid, 200*time.Second)
	n, err := r.svc.Reap(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	st, _, partial, code := r.row(mid)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "INTERRUPTED", *code)
	require.Equal(t, "phần đầu", *partial)
	require.Zero(t, r.count(`select count(*) from chat_messages where id=$1 and completed_at is null`, mid))
	_, err = r.rdb.Get(t.Context(), "ep:chat:active:"+r.student.UserID.String()).Result()
	require.Error(t, err, "khoá nhả")
	// chạy lại không đụng gì
	n, _ = r.svc.Reap(t.Context())
	require.Zero(t, n)
}

// TestReapSkipsLive — AC19: tin còn cập nhật trong hạn không bị đánh dấu.
func TestReapSkipsLive(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid := r.orphan(r.student)
	r.backdate(mid, 100*time.Second)
	n, err := r.svc.Reap(t.Context())
	require.NoError(t, err)
	require.Zero(t, n)
	st, _, _, _ := r.row(mid)
	require.Equal(t, "STREAMING", st)
}

// TestReapDoesNotOverwriteLive — AC19: G còn sống mà bị reaper ghi trước thấy 0 dòng ở lệnh ghi cuối và dừng; không ai ghi đè ai.
func TestReapDoesNotOverwriteLive(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "muộn")}, nil
	}
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "chào")
	require.NoError(t, err)
	r.backdate(m.ID, 200*time.Second)
	n, err := r.svc.Reap(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	close(gate)
	r.svc.Wait()
	st, content, _, code := r.row(m.ID)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "INTERRUPTED", *code)
	require.Empty(t, content, "ghi cuối của G không ghi đè")
}

// TestDrainMarksInterruptedAndReleasesBusy — AC19: tắt gateway có chủ đích → G tự ghi FAILED INTERRUPTED và nhả khoá ngay.
func TestDrainMarksInterruptedAndReleasesBusy(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	drain := make(chan struct{})
	r.svc = r.service(chat.Config{StreamMax: 20 * time.Second})
	r.svc.Drain = drain
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "x")}, nil
	}
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "chào")
	require.NoError(t, err)
	close(drain)
	r.svc.Wait()
	st, _, _, code := r.row(m.ID)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "INTERRUPTED", *code)
	_, err = r.rdb.Get(t.Context(), "ep:chat:active:"+r.student.UserID.String()).Result()
	require.Error(t, err)
	close(gate)
}

// TestUpdatedAtTrigger — AC19: updated_at do trigger (reaper dựa vào nó).
func TestUpdatedAtTrigger(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid := r.orphan(r.student)
	r.backdate(mid, time.Hour)
	_, err := r.pool.Exec(t.Context(), `update chat_messages set partial_content='mới' where id=$1`, mid)
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id=$1 and updated_at > now() - interval '1 minute'`, mid))
}
