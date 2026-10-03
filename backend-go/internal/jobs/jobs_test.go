package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appdb "github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

// env là bộ phụ thuộc thật (Postgres riêng mỗi test + Redis dùng chung) cho test việc dài.
type env struct {
	cfg  config.Config
	log  *slog.Logger
	pool *pgxpool.Pool
	rdb  *appredis.Client
	svc  *Service
}

func newEnv(t *testing.T) env {
	t.Helper()
	cfg := config.Config{
		Role: config.Worker, AppEnv: "test", InstanceID: "worker-test", LogLevel: "debug",
		DBMaxConns: 8, DBSlowQueryMS: 200, RequestTimeout: 30 * time.Second,
		DatabaseURL: testutil.MigratedPostgresURL(t), RedisURL: testutil.RedisURL(t),
		SSEBufferMaxLen: 1000, SSEBufferTTL: time.Hour,
		OutboxPollInterval: 50 * time.Millisecond, OutboxBatch: 20,
		OutboxRetryBackoff: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
		OutboxClaimIdle:    5 * time.Second, OutboxStaleAfter: 30 * time.Second,
	}
	log := applog.NewTo(testWriter{t}, cfg, "worker")
	pool, err := appdb.NewPool(t.Context(), cfg, log)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(t.Context(), cfg.RedisURL)
	if err != nil {
		t.Fatalf("redis.New: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return env{cfg: cfg, log: log, pool: pool, rdb: rdb, svc: NewService(pool)}
}

// testWriter đưa log của gói vào `go test -v` (không cần biến toàn cục).
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func (e env) runner(pub sse.Publisher) *Runner {
	return NewRunner(e.pool, pub, clock.Real{}, e.log)
}

func (e env) publisher() sse.Publisher {
	return sse.NewPublisher(e.rdb, e.cfg.SSEBufferMaxLen, e.cfg.SSEBufferTTL)
}

// message lấy đúng dòng outbox mà Enqueue đã ghi và dựng tin như consumer sẽ giao.
func (e env) message(t *testing.T, jobID uuid.UUID) outbox.Message {
	t.Helper()
	var id uuid.UUID
	var topic string
	var payload json.RawMessage
	err := e.pool.QueryRow(t.Context(),
		`select id, topic, payload from outbox where topic = $1 and payload->>'job_id' = $2`,
		TopicEnqueue, jobID.String()).Scan(&id, &topic, &payload)
	if err != nil {
		t.Fatalf("đọc dòng outbox của việc: %v", err)
	}
	return outbox.Message{ID: id, Topic: topic, Payload: payload}
}

func (e env) job(t *testing.T, id uuid.UUID) store.Job {
	t.Helper()
	j, err := store.New(e.pool).GetJob(t.Context(), id)
	if err != nil {
		t.Fatalf("đọc việc: %v", err)
	}
	return j
}

// progressKind là bản chạy thử của `test.progress`: n bước, mỗi bước báo i/n × 100.
func progressKind(t *testing.T) KindFunc {
	t.Helper()
	return func(_ context.Context, j JobCtx) (any, error) {
		var p struct {
			Steps int `json:"steps"`
		}
		if err := json.Unmarshal(j.Payload, &p); err != nil || p.Steps <= 0 {
			p.Steps = 4
		}
		for i := 1; i <= p.Steps; i++ {
			time.Sleep(5 * time.Millisecond)
			j.Progress(i * 100 / p.Steps)
		}
		return map[string]any{"steps": p.Steps}, nil
	}
}

// sseEvents đọc mọi sự kiện trong bộ đệm SSE của một người dùng.
func sseEvents(t *testing.T, e env, userID string, typ string) []progressEvent {
	t.Helper()
	msgs, err := e.rdb.XRange(t.Context(), sse.BufKey(userID), "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	out := []progressEvent{}
	for _, m := range msgs {
		if m.Values["type"] != typ {
			continue
		}
		var ev progressEvent
		if err := json.Unmarshal([]byte(fmt.Sprint(m.Values["data"])), &ev); err != nil {
			t.Fatalf("data không phải JSON: %v", err)
		}
		out = append(out, ev)
	}
	return out
}

func TestJobs_EnqueueAtomic(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := uuid.New()

	j, err := e.svc.Enqueue(t.Context(), owner, "test.progress", map[string]any{"steps": 4})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if j.Status != store.JobStatusQUEUED || j.Progress != 0 || j.OwnerID != owner {
		t.Fatalf("việc mới = %+v", j)
	}

	// Dòng jobs và dòng outbox phải cùng một transaction: now() trong một tx là một giá trị.
	var n int
	if err := e.pool.QueryRow(t.Context(), `select count(*) from jobs j, outbox o
		where j.id = $1 and o.topic = $2 and o.payload->>'job_id' = j.id::text and o.created_at = j.created_at`,
		j.ID, TopicEnqueue).Scan(&n); err != nil {
		t.Fatalf("đối chiếu jobs/outbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("có %d cặp (jobs, outbox) cùng created_at, muốn 1", n)
	}
	m := e.message(t, j.ID)
	var p enqueuePayload
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		t.Fatalf("payload outbox: %v", err)
	}
	if p.JobID != j.ID || p.Kind != "test.progress" || !jsonEq(p.Payload, `{"steps":4}`) {
		t.Fatalf("payload = %+v", p)
	}

	// Rollback: outbox.Write hỏng → KHÔNG dòng jobs nào được giữ lại.
	if _, err := e.pool.Exec(t.Context(), "alter table outbox rename to outbox_hidden"); err != nil {
		t.Fatalf("đổi tên outbox: %v", err)
	}
	before := e.count(t, "select count(*) from jobs")
	if _, err := e.svc.Enqueue(t.Context(), owner, "test.progress", nil); err == nil {
		t.Fatal("Enqueue phải lỗi khi không ghi được outbox")
	}
	if after := e.count(t, "select count(*) from jobs"); after != before {
		t.Fatalf("jobs tăng %d dòng sau khi rollback, muốn 0", after-before)
	}
	if _, err := e.pool.Exec(t.Context(), "alter table outbox_hidden rename to outbox"); err != nil {
		t.Fatalf("trả lại tên outbox: %v", err)
	}
}

// jsonEq so sánh theo nghĩa JSON (Postgres chuẩn hoá khoảng trắng của jsonb).
func jsonEq(got json.RawMessage, want string) bool {
	var a, b any
	if err := json.Unmarshal(got, &a); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		return false
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func (e env) count(t *testing.T, q string) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(t.Context(), q).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

// TestJobs_Lifecycle chạy ĐẦU-CUỐI: Enqueue (gateway) → relay → Redis Stream → consumer → runner.
func TestJobs_Lifecycle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := uuid.New()

	gate := make(chan struct{})
	r := e.runner(e.publisher())
	r.Register("test.progress", func(ctx context.Context, j JobCtx) (any, error) {
		j.Progress(25)
		<-gate
		j.Progress(100)
		return map[string]any{"ok": true}, nil
	})
	reg := outbox.NewRegistry()
	reg.Register(TopicEnqueue, r.HandleMessage)

	pre := testutil.TestPrefix(t)
	d := outbox.Deps{Pool: e.pool, Redis: e.rdb, Log: e.log, Clock: clock.Real{}, Cfg: e.cfg,
		Streams: outbox.Streams{Dispatch: pre + ".dispatch", Dead: pre + ".dead", Group: "outbox", Consumer: "c1"}}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = outbox.NewRelay(d).Run(ctx) }()
	go func() { _ = outbox.NewConsumer(d, reg).Run(ctx) }()

	j, err := e.svc.Enqueue(t.Context(), owner, "test.progress", map[string]any{"steps": 4})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	waitStatus(t, e, j.ID, store.JobStatusRUNNING, 30*time.Second)
	if got := e.job(t, j.ID); got.Progress != 25 || got.FinishedAt != nil {
		t.Fatalf("khi đang chạy: progress = %d, finished_at = %v", got.Progress, got.FinishedAt)
	}
	close(gate)
	waitStatus(t, e, j.ID, store.JobStatusSUCCEEDED, 30*time.Second)

	done := e.job(t, j.ID)
	if done.Progress != 100 || done.FinishedAt == nil || !jsonEq(done.Result, `{"ok":true}`) {
		t.Fatalf("việc xong = %+v (result %s)", done, done.Result)
	}
	if jsonOrNil(done.Error) != nil {
		t.Fatalf("việc thành công không được có error: %s", done.Error)
	}
}

// waitStatus chờ việc đạt trạng thái mong muốn.
func waitStatus(t *testing.T, e env, id uuid.UUID, want store.JobStatus, max time.Duration) {
	t.Helper()
	deadline := time.Now().Add(max)
	last := store.JobStatus("")
	for time.Now().Before(deadline) {
		j, err := store.New(e.pool).GetJob(t.Context(), id)
		if err == nil {
			last = j.Status
			if j.Status == want {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("việc %s ở %q sau %s, muốn %q", id, last, max, want)
}

func TestJobs_ProgressMonotonic(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := uuid.New()
	seen := []int{}

	r := e.runner(e.publisher())
	r.Register("mono", func(ctx context.Context, j JobCtx) (any, error) {
		for _, pct := range []int{50, 10, 50, 60, 0, 90} {
			j.Progress(pct)
			row, err := store.New(e.pool).GetJob(ctx, j.ID)
			if err != nil {
				return nil, err
			}
			seen = append(seen, int(row.Progress))
		}
		return nil, nil
	})
	j, err := e.svc.Enqueue(t.Context(), owner, "mono", nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := r.HandleMessage(t.Context(), e.message(t, j.ID)); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	want := []int{50, 50, 50, 60, 60, 90}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("progress trong DB = %v, muốn %v (chỉ tăng)", seen, want)
	}
	evs := sseEvents(t, e, owner.String(), EventProgress)
	prev := -1
	for _, ev := range evs {
		if ev.Progress < prev {
			t.Fatalf("sự kiện giảm tiến độ: %v", evs)
		}
		prev = ev.Progress
	}
	if got := e.job(t, j.ID); got.Status != store.JobStatusSUCCEEDED || got.Progress != 100 {
		t.Fatalf("việc cuối = %s/%d", got.Status, got.Progress)
	}
}

func TestJobs_Failed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := uuid.New()

	r := e.runner(e.publisher())
	r.Register("test.fail", func(ctx context.Context, j JobCtx) (any, error) {
		j.Progress(25)
		return nil, fmt.Errorf("chi tiết nội bộ: nguoi.dung@example.com")
	})
	j, err := e.svc.Enqueue(t.Context(), owner, "test.fail", nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := r.HandleMessage(t.Context(), e.message(t, j.ID)); err != nil {
		t.Fatalf("HandleMessage không được trả lỗi outbox: %v", err)
	}

	got := e.job(t, j.ID)
	if got.Status != store.JobStatusFAILED || got.FinishedAt == nil {
		t.Fatalf("việc = %s, finished_at = %v", got.Status, got.FinishedAt)
	}
	if got.Progress != 25 {
		t.Fatalf("progress = %d, muốn giữ nguyên 25 (< 100)", got.Progress)
	}
	var e2 struct{ Code, Message string }
	if err := json.Unmarshal(got.Error, &e2); err != nil {
		t.Fatalf("error không phải JSON: %s", got.Error)
	}
	if e2.Code == "" || e2.Message == "" {
		t.Fatalf("error = %+v, cần {code, message}", e2)
	}
	if strings.Contains(string(got.Error), "example.com") {
		t.Fatalf("error lộ chi tiết nội bộ: %s", got.Error)
	}

	// Giao lại cùng tin: việc đã FAILED → không chạy lại, không đổi trạng thái.
	if err := r.HandleMessage(t.Context(), e.message(t, j.ID)); err != nil {
		t.Fatalf("giao lại: %v", err)
	}
	if again := e.job(t, j.ID); again.UpdatedAt != got.UpdatedAt {
		t.Fatalf("việc đã kết thúc bị xử lý lại (updated_at đổi)")
	}
}

func TestJobs_PanicFailed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := uuid.New()
	var runs atomic.Int64

	r := e.runner(e.publisher())
	r.Register("boom", func(ctx context.Context, j JobCtx) (any, error) {
		runs.Add(1)
		panic("nổ giữa chừng")
	})
	j, err := e.svc.Enqueue(t.Context(), owner, "boom", nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := r.HandleMessage(t.Context(), e.message(t, j.ID)); err != nil {
		t.Fatalf("panic phải thành FAILED, không trả lỗi outbox: %v", err)
	}
	got := e.job(t, j.ID)
	if got.Status != store.JobStatusFAILED || got.FinishedAt == nil {
		t.Fatalf("việc = %s, finished_at = %v (không được treo ở RUNNING)", got.Status, got.FinishedAt)
	}
	if !strings.Contains(string(got.Error), codePanic) {
		t.Fatalf("error = %s, muốn code %s", got.Error, codePanic)
	}
	if runs.Load() != 1 {
		t.Fatalf("kind chạy %d lần, muốn 1", runs.Load())
	}

	// Loại việc không có handler cũng phải FAILED chứ không treo.
	j2, err := e.svc.Enqueue(t.Context(), owner, "khong.ton.tai", nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := r.HandleMessage(t.Context(), e.message(t, j2.ID)); err != nil {
		t.Fatalf("kind lạ: %v", err)
	}
	if got := e.job(t, j2.ID); got.Status != store.JobStatusFAILED ||
		!strings.Contains(string(got.Error), codeUnknownKind) {
		t.Fatalf("kind lạ → %s / %s", got.Status, got.Error)
	}
}

func TestJobProgress_SSE(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := uuid.New()

	r := e.runner(e.publisher())
	r.Register("test.progress", progressKind(t))
	j, err := e.svc.Enqueue(t.Context(), owner, "test.progress", map[string]any{"steps": 4})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := r.HandleMessage(t.Context(), e.message(t, j.ID)); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	evs := sseEvents(t, e, owner.String(), EventProgress)
	if len(evs) != 4 {
		t.Fatalf("có %d sự kiện job.progress, muốn 4: %+v", len(evs), evs)
	}
	for i, want := range []int{25, 50, 75, 100} {
		if evs[i].Progress != want {
			t.Fatalf("sự kiện #%d progress = %d, muốn %d (%+v)", i, evs[i].Progress, want, evs)
		}
		if evs[i].JobID != j.ID.String() {
			t.Fatalf("sự kiện #%d job_id = %s", i, evs[i].JobID)
		}
	}
	if evs[3].Status != string(store.JobStatusSUCCEEDED) || !jsonEq(evs[3].Result, `{"steps":4}`) {
		t.Fatalf("sự kiện cuối = %+v, muốn SUCCEEDED + result", evs[3])
	}
	for _, ev := range evs[:3] {
		if ev.Status != string(store.JobStatusRUNNING) {
			t.Fatalf("sự kiện giữa chừng = %+v, muốn RUNNING", ev)
		}
	}
}

func TestJobProgress_OnlyOwner(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner, other := uuid.New(), uuid.New()

	r := e.runner(e.publisher())
	r.Register("test.progress", progressKind(t))
	j, err := e.svc.Enqueue(t.Context(), owner, "test.progress", map[string]any{"steps": 4})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := r.HandleMessage(t.Context(), e.message(t, j.ID)); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	if n := len(sseEvents(t, e, owner.String(), EventProgress)); n == 0 {
		t.Fatal("chủ việc không nhận được sự kiện nào")
	}
	n, err := e.rdb.Exists(t.Context(), sse.BufKey(other.String())).Result()
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if n != 0 {
		t.Fatalf("người khác có bộ đệm sự kiện (exists = %d)", n)
	}
}
