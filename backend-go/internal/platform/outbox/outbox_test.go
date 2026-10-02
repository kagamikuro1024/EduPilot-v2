// Test của US-PG-02 AC13–AC16: outbox cùng transaction, hai worker không trùng, retry rồi dead-letter,
// consumer chết giữa chừng. Dùng Postgres + Redis THẬT (testcontainers); tên Stream riêng theo test nên
// các test chạy song song trên cùng một Redis.
package outbox_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

const (
	wait = 10 * time.Second // AC14: "đợi ≤ 10 s"
	beat = 20 * time.Millisecond
)

type env struct {
	t     *testing.T
	ctx   context.Context
	pool  *pgxpool.Pool
	rdb   *appredis.Client
	q     *store.Queries
	cfg   config.Config
	names outbox.Streams
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	p := testutil.TestPrefix(t)
	names := outbox.Streams{Dispatch: p + ".dispatch", Dead: p + ".dispatch.dead", Consumer: p + "-c1"}
	t.Cleanup(func() {
		_ = rdb.Del(context.Background(), names.Dispatch, names.Dead).Err()
	})

	return &env{
		t: t, ctx: ctx, pool: pool, rdb: rdb, q: store.New(pool), names: names,
		cfg: config.Config{
			InstanceID:         p,
			OutboxPollInterval: beat,
			OutboxBatch:        100,
			OutboxRetryBackoff: []time.Duration{beat, beat, beat},
			OutboxClaimIdle:    5 * time.Second,
			OutboxStaleAfter:   time.Minute,
		},
	}
}

func (e *env) deps(consumer string) outbox.Deps {
	names := e.names
	if consumer != "" {
		names.Consumer = consumer
	}
	return outbox.Deps{
		Pool:    e.pool,
		Redis:   e.rdb,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Clock:   clock.Real{},
		Cfg:     e.cfg,
		Streams: names,
	}
}

// start chạy các việc nền tới khi test kết thúc.
func (e *env) start(tasks ...interface{ Run(context.Context) error }) {
	e.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(task interface{ Run(context.Context) error }) {
			defer wg.Done()
			_ = task.Run(ctx)
		}(task)
	}
	e.t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
}

func (e *env) insert(topic, payload string) uuid.UUID {
	e.t.Helper()
	row, err := e.q.InsertOutbox(e.ctx, store.InsertOutboxParams{Topic: topic, Payload: []byte(payload)})
	require.NoError(e.t, err)
	return row.ID
}

func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.pool.QueryRow(e.ctx, sql, args...).Scan(&n))
	return n
}

func (e *env) xlen(stream string) int64 {
	n, err := e.rdb.XLen(e.ctx, stream).Result()
	require.NoError(e.t, err)
	return n
}

// pending trả số tin chưa XACK của nhóm (0 khi nhóm chưa tồn tại).
func (e *env) pending() int64 {
	res, err := e.rdb.XPending(e.ctx, e.names.Dispatch, outbox.GroupName).Result()
	if err != nil {
		return 0
	}
	return res.Count
}

// counter đếm số lần handler chạy theo outbox.id.
type counter struct {
	mu sync.Mutex
	n  map[uuid.UUID]int
}

func newCounter() *counter { return &counter{n: map[uuid.UUID]int{}} }

func (c *counter) hit(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[id]++
}

func (c *counter) get(id uuid.UUID) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n[id]
}

func (c *counter) all() map[uuid.UUID]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[uuid.UUID]int, len(c.n))
	for k, v := range c.n {
		out[k] = v
	}
	return out
}

// TestOutbox_SameTransaction_Commit — AC13: commit ⇒ đúng 1 dòng dữ liệu + 1 dòng outbox.
func TestOutbox_SameTransaction_Commit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	tx, err := e.pool.Begin(e.ctx)
	require.NoError(t, err)
	user, err := store.New(tx).InsertUser(e.ctx, store.InsertUserParams{
		Email: "commit@x.com", FullName: "QA", Role: store.UserRoleSTUDENT, Status: store.UserStatusINVITED,
	})
	require.NoError(t, err)
	id, err := outbox.Write(e.ctx, tx, "test.ok", map[string]string{"user_id": user.ID.String()})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(e.ctx))

	require.Equal(t, 1, e.count("select count(*) from users"))
	require.Equal(t, 1, e.count("select count(*) from outbox"))
	row, err := e.q.GetOutbox(e.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "test.ok", row.Topic)
	require.JSONEq(t, fmt.Sprintf(`{"user_id":%q}`, user.ID), string(row.Payload))
	require.Nil(t, row.EnqueuedAt)
	require.Nil(t, row.DispatchedAt)
}

// TestOutbox_SameTransaction_Rollback — AC13: rollback ⇒ 0 dòng cả hai, không tin nào vào Stream.
func TestOutbox_SameTransaction_Rollback(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.start(outbox.NewRelay(e.deps("")))

	tx, err := e.pool.Begin(e.ctx)
	require.NoError(t, err)
	_, err = store.New(tx).InsertUser(e.ctx, store.InsertUserParams{
		Email: "rollback@x.com", FullName: "QA", Role: store.UserRoleSTUDENT, Status: store.UserStatusINVITED,
	})
	require.NoError(t, err)
	_, err = outbox.Write(e.ctx, tx, "test.ok", map[string]string{"a": "1"})
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(e.ctx))

	time.Sleep(10 * beat) // relay quét vài nhịp
	require.Equal(t, 0, e.count("select count(*) from users"))
	require.Equal(t, 0, e.count("select count(*) from outbox"))
	require.EqualValues(t, 0, e.xlen(e.names.Dispatch))
	require.EqualValues(t, 0, e.xlen(e.names.Dead))
}

// TestOutbox_TwoWorkers_ExactlyOnce — AC14: 100 dòng, hai relay + hai consumer ⇒ mỗi outbox.id
// chạy handler đúng 1 lần, tất cả dispatched, XPENDING = 0, dead-letter rỗng.
func TestOutbox_TwoWorkers_ExactlyOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	calls := newCounter()
	reg := outbox.NewRegistry()
	reg.Register("test.ok", func(_ context.Context, m outbox.Message) error {
		calls.hit(m.ID)
		return nil
	})

	const n = 100
	ids := make([]uuid.UUID, 0, n)
	for i := range n {
		ids = append(ids, e.insert("test.ok", fmt.Sprintf(`{"i":%d}`, i)))
	}

	e.start(
		outbox.NewRelay(e.deps("")), outbox.NewRelay(e.deps("")),
		outbox.NewConsumer(e.deps(e.names.Consumer+"-a"), reg),
		outbox.NewConsumer(e.deps(e.names.Consumer+"-b"), reg),
	)

	require.Eventually(t, func() bool {
		return e.count("select count(*) from outbox where dispatched_at is not null") == n
	}, wait, beat, "100 dòng phải dispatched trong 10 s")

	got := calls.all()
	require.Len(t, got, n)
	for _, id := range ids {
		require.Equal(t, 1, got[id], "handler phải chạy đúng 1 lần cho %s", id)
	}
	require.Equal(t, 0, e.count("select count(*) from outbox where attempts <> 0 or dead_at is not null"))
	require.Eventually(t, func() bool { return e.xlen(e.names.Dispatch) == 0 && e.pending() == 0 },
		wait, beat, "tin phải được XACK rồi XDEL")
	require.EqualValues(t, 0, e.xlen(e.names.Dead))
}

// TestOutbox_RetryThenDead — AC15: handler luôn lỗi ⇒ 4 lần gọi, dead_at, attempts = 4, dead-letter 1 tin.
func TestOutbox_RetryThenDead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	calls := newCounter()
	reg := outbox.NewRegistry()
	reg.Register("test.fail", func(_ context.Context, m outbox.Message) error {
		calls.hit(m.ID)
		return fmt.Errorf("luôn hỏng (attempts=%d)", m.Attempts)
	})

	id := e.insert("test.fail", `{"a":1}`)
	e.start(outbox.NewRelay(e.deps("")), outbox.NewConsumer(e.deps(""), reg))

	row := e.waitDead(id)
	require.Equal(t, 4, calls.get(id), "1 lần đầu + 3 lần thử lại")
	require.EqualValues(t, 4, row.Attempts)
	require.Nil(t, row.DispatchedAt)
	require.NotNil(t, row.LastError)
	require.NotEmpty(t, *row.LastError)

	msgs, err := e.rdb.XRange(e.ctx, e.names.Dead, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, id.String(), msgs[0].Values["outbox_id"])
	require.Equal(t, "test.fail", msgs[0].Values["topic"])
	require.NotEmpty(t, msgs[0].Values["error"])

	require.Eventually(t, func() bool { return e.pending() == 0 && e.xlen(e.names.Dispatch) == 0 },
		wait, beat, "tin gốc phải XACK + XDEL")
}

// TestOutbox_PanicIsFailure — AC15: handler panic đi cùng đường lỗi và consumer vẫn sống.
func TestOutbox_PanicIsFailure(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	calls := newCounter()
	reg := outbox.NewRegistry()
	reg.Register("test.panic", func(_ context.Context, m outbox.Message) error {
		calls.hit(m.ID)
		panic("nổ trong handler")
	})
	reg.Register("test.ok", func(_ context.Context, m outbox.Message) error {
		calls.hit(m.ID)
		return nil
	})

	id := e.insert("test.panic", `{"a":1}`)
	e.start(outbox.NewRelay(e.deps("")), outbox.NewConsumer(e.deps(""), reg))

	row := e.waitDead(id)
	require.Equal(t, 4, calls.get(id))
	require.EqualValues(t, 4, row.Attempts)
	require.Contains(t, *row.LastError, "panic")

	// Consumer không chết: dòng tiếp theo vẫn chạy.
	next := e.insert("test.ok", `{"a":2}`)
	require.Eventually(t, func() bool { return calls.get(next) == 1 }, wait, beat, "consumer phải sống sau panic")
	require.Eventually(t, func() bool {
		return e.count("select count(*) from outbox where id = $1 and dispatched_at is not null", next) == 1
	}, wait, beat)
}

// TestOutbox_UnknownTopicDead — AC15: topic chưa đăng ký ⇒ dead-letter; last_error ≤ 1000 ký tự, không lộ PII.
func TestOutbox_UnknownTopicDead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	id := e.insert("test.unknown", `{"email":"pii-canary@example.com"}`)
	e.start(outbox.NewRelay(e.deps("")), outbox.NewConsumer(e.deps(""), outbox.NewRegistry()))

	row := e.waitDead(id)
	require.EqualValues(t, 4, row.Attempts)
	require.Nil(t, row.DispatchedAt)
	require.NotNil(t, row.LastError)
	require.LessOrEqual(t, len([]rune(*row.LastError)), 1000)
	require.NotContains(t, *row.LastError, "pii-canary", "last_error không được chứa payload")
	require.Contains(t, *row.LastError, "test.unknown")

	msgs, err := e.rdb.XRange(e.ctx, e.names.Dead, "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "test.unknown", msgs[0].Values["topic"])
	require.Eventually(t, func() bool { return e.pending() == 0 }, wait, beat)
}

// TestOutbox_ConsumerCrash_Reclaimed — AC16: tin đã đọc mà chưa XACK, quá OUTBOX_CLAIM_IDLE thì
// consumer khác XAUTOCLAIM và xử lý; không mất tin.
func TestOutbox_ConsumerCrash_Reclaimed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.cfg.OutboxClaimIdle = 50 * time.Millisecond
	calls := newCounter()
	reg := outbox.NewRegistry()
	reg.Register("test.ok", func(_ context.Context, m outbox.Message) error {
		calls.hit(m.ID)
		return nil
	})

	require.NoError(t, e.rdb.XGroupCreateMkStream(e.ctx, e.names.Dispatch, outbox.GroupName, "0").Err())
	id := e.insert("test.ok", `{"a":1}`)
	e.start(outbox.NewRelay(e.deps("")))
	require.Eventually(t, func() bool { return e.xlen(e.names.Dispatch) == 1 }, wait, beat, "relay phải XADD")

	// Consumer "chết": đọc tin rồi không XACK.
	read, err := e.rdb.XReadGroup(e.ctx, &goredis.XReadGroupArgs{
		Group: outbox.GroupName, Consumer: "crashed", Streams: []string{e.names.Dispatch, ">"}, Count: 10,
	}).Result()
	require.NoError(t, err)
	require.Len(t, read[0].Messages, 1)
	require.EqualValues(t, 1, e.pending())
	require.Equal(t, 0, calls.get(id))

	e.start(outbox.NewConsumer(e.deps(e.names.Consumer+"-new"), reg))
	require.Eventually(t, func() bool {
		return e.count("select count(*) from outbox where id = $1 and dispatched_at is not null", id) == 1
	}, wait, beat, "consumer mới phải nhận lại tin (XAUTOCLAIM)")
	require.Equal(t, 1, calls.get(id))
	require.Eventually(t, func() bool { return e.pending() == 0 && e.xlen(e.names.Dispatch) == 0 }, wait, beat)
	require.EqualValues(t, 0, e.xlen(e.names.Dead))
}

// TestOutbox_StaleEnqueuedRequeued — AC16: dòng kẹt "đã xếp hàng" quá OUTBOX_STALE_AFTER (tin mất
// hẳn khỏi Stream) được relay đặt lại enqueued_at = NULL và đẩy lại.
func TestOutbox_StaleEnqueuedRequeued(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.cfg.OutboxStaleAfter = 100 * time.Millisecond
	calls := newCounter()
	reg := outbox.NewRegistry()
	reg.Register("test.ok", func(_ context.Context, m outbox.Message) error {
		calls.hit(m.ID)
		return nil
	})

	id := e.insert("test.ok", `{"a":1}`)
	// Giả lập tin mất: dòng đã đánh dấu xếp hàng nhưng KHÔNG có trong Stream.
	n, err := e.q.MarkOutboxEnqueued(e.ctx, id)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	require.EqualValues(t, 0, e.xlen(e.names.Dispatch))

	e.start(outbox.NewRelay(e.deps("")), outbox.NewConsumer(e.deps(""), reg))
	require.Eventually(t, func() bool {
		return e.count("select count(*) from outbox where id = $1 and dispatched_at is not null", id) == 1
	}, wait, beat, "dòng kẹt phải được đẩy lại rồi dispatched")
	require.GreaterOrEqual(t, calls.get(id), 1)
}

// waitDead đợi dòng outbox đi hết đường dead-letter rồi trả về dòng đó.
func (e *env) waitDead(id uuid.UUID) store.Outbox {
	e.t.Helper()
	var row store.Outbox
	require.Eventually(e.t, func() bool {
		var err error
		row, err = e.q.GetOutbox(e.ctx, id)
		return err == nil && row.DeadAt != nil
	}, wait, beat, "dòng phải dead-letter sau 4 lần thử")
	return row
}

// TestOutbox_LastErrorTruncated — SRS 5.3: thông điệp lỗi dài bị cắt còn 1000 ký tự.
func TestOutbox_LastErrorTruncated(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	reg := outbox.NewRegistry()
	reg.Register("test.long", func(_ context.Context, _ outbox.Message) error {
		return fmt.Errorf("%s", strings.Repeat("x", 5000))
	})

	id := e.insert("test.long", `{"a":1}`)
	e.start(outbox.NewRelay(e.deps("")), outbox.NewConsumer(e.deps(""), reg))

	row := e.waitDead(id)
	require.NotNil(t, row.LastError)
	require.Len(t, []rune(*row.LastError), 1000)
}
