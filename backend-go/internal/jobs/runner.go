package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// EventProgress là tên sự kiện SSE báo tiến độ việc (SRS 6.8, US-PG-05 AC12).
const EventProgress = "job.progress"

// Mã lỗi ghi vào cột `jobs.error` ({code, message}, không PII — SRS 5.4).
const (
	codeFailed      = "JOB_FAILED"
	codePanic       = "JOB_PANIC"
	codeUnknownKind = "UNKNOWN_KIND"
)

// KindFunc chạy một loại việc. Trả `result` (ghi vào `jobs.result`) hoặc lỗi → việc FAILED.
// Panic cũng thành FAILED (không bao giờ treo ở RUNNING).
type KindFunc func(ctx context.Context, j JobCtx) (any, error)

// Runner là phía worker: nhận tin `job.enqueue` từ outbox và chạy việc.
type Runner struct {
	db  *pgxpool.Pool
	pub sse.Publisher
	clk clock.Clock
	log *slog.Logger

	mu    sync.RWMutex
	kinds map[string]KindFunc
}

// NewRunner dựng Runner rỗng; gọi Register cho từng loại việc trước khi chạy.
func NewRunner(db *pgxpool.Pool, pub sse.Publisher, clk clock.Clock, log *slog.Logger) *Runner {
	return &Runner{db: db, pub: pub, clk: clk, log: log, kinds: map[string]KindFunc{}}
}

// Register gắn hàm chạy cho một loại việc; đăng ký trùng = lỗi lập trình lúc khởi động → panic.
func (r *Runner) Register(kind string, fn KindFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.kinds[kind]; dup {
		panic(fmt.Sprintf("jobs: kind %q đã đăng ký", kind))
	}
	r.kinds[kind] = fn
}

func (r *Runner) lookup(kind string) (KindFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.kinds[kind]
	return fn, ok
}

// HandleMessage chạy một tin `job.enqueue` (đăng ký vào outbox.Registry). Idempotent: việc đã SUCCEEDED/FAILED
// thì bỏ qua. Lỗi của chính việc KHÔNG trả về lỗi outbox (việc đã FAILED, chạy lại cũng thế); chỉ lỗi hạ tầng
// mới trả lỗi để outbox retry.
func (r *Runner) HandleMessage(ctx context.Context, m outbox.Message) error {
	var p enqueuePayload
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("jobs: tin %s sai định dạng: %w", m.ID, err)
	}
	q := store.New(r.db)
	j, err := q.GetJob(ctx, p.JobID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil // việc đã bị xoá — không có gì để chạy
	case err != nil:
		return fmt.Errorf("jobs: đọc việc: %w", err)
	case j.Status == store.JobStatusSUCCEEDED || j.Status == store.JobStatusFAILED:
		return nil // đã xong ở lần giao trước (at-least-once)
	}

	fn, ok := r.lookup(j.Kind)
	if !ok {
		r.log.ErrorContext(ctx, "job kind không có handler", "job_id", j.ID.String(), "kind", j.Kind)
		return r.fail(ctx, j, codeUnknownKind, "Loại việc không được hỗ trợ.")
	}

	// QUEUED → RUNNING (UpdateJobProgress chỉ đổi khi việc còn QUEUED/RUNNING).
	if _, err := q.UpdateJobProgress(ctx, store.UpdateJobProgressParams{ID: j.ID, Progress: j.Progress}); err != nil {
		return fmt.Errorf("jobs: chuyển việc sang RUNNING: %w", err)
	}

	start := r.clk.Now()
	tr := &progressTracker{runner: r, job: j.ID, owner: j.OwnerID, last: int(j.Progress)}
	result, err := r.runKind(ctx, fn, JobCtx{ID: j.ID, OwnerID: j.OwnerID, Payload: p.Payload, ctx: ctx, tr: tr})
	dur := r.clk.Now().Sub(start).Milliseconds()
	if err != nil {
		code, msg := codeFailed, "Việc chạy thất bại."
		var ue *UserError
		switch {
		case errors.Is(err, errPanic):
			code, msg = codePanic, "Việc dừng do lỗi không mong đợi."
		case errors.As(err, &ue):
			code, msg = ue.Code, ue.Message
		}
		r.log.ErrorContext(ctx, "job thất bại", "job_id", j.ID.String(), "kind", j.Kind, "duration_ms", dur, "error", err.Error())
		return r.fail(ctx, j, code, msg)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return r.fail(ctx, j, codeFailed, "Kết quả việc không mã hoá được.")
	}
	done, err := q.MarkJobSucceeded(ctx, store.MarkJobSucceededParams{ID: j.ID, Result: raw})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // đã kết thúc ở nơi khác
	}
	if err != nil {
		return fmt.Errorf("jobs: ghi kết quả: %w", err)
	}
	r.log.InfoContext(ctx, "job xong", "job_id", j.ID.String(), "kind", j.Kind, "duration_ms", dur)
	r.publish(ctx, done)
	return nil
}

// UserError là lỗi của việc kèm mã và câu tiếng Việt hiển thị được cho chủ việc (không PII, không chi tiết nội bộ).
// Lỗi khác UserError chỉ ra "Việc chạy thất bại." chung.
type UserError struct{ Code, Message string }

func (e *UserError) Error() string { return e.Code + ": " + e.Message }

// errPanic đánh dấu việc chết vì panic (phân biệt mã lỗi ghi vào jobs.error).
var errPanic = errors.New("panic trong job")

// runKind chạy hàm của loại việc và biến panic thành lỗi.
func (r *Runner) runKind(ctx context.Context, fn KindFunc, jc JobCtx) (result any, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%w: %v", errPanic, rec)
		}
	}()
	return fn(ctx, jc)
}

// fail đánh dấu việc FAILED với {code, message} (không PII) và phát sự kiện cuối.
func (r *Runner) fail(ctx context.Context, j store.Job, code, msg string) error {
	raw, err := json.Marshal(map[string]string{"code": code, "message": msg})
	if err != nil {
		return fmt.Errorf("jobs: mã hoá lỗi việc: %w", err)
	}
	done, err := store.New(r.db).MarkJobFailed(ctx, store.MarkJobFailedParams{ID: j.ID, Error: raw})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("jobs: ghi lỗi việc: %w", err)
	}
	r.publish(ctx, done)
	return nil
}

// progressEvent là `data` của sự kiện SSE `job.progress`.
type progressEvent struct {
	JobID    string          `json:"job_id"`
	Status   string          `json:"status"`
	Progress int             `json:"progress"`
	Result   json.RawMessage `json:"result,omitempty"`
}

// publish phát trạng thái việc tới stream của CHỦ việc (chỉ chủ việc — US-PG-05 AC12).
func (r *Runner) publish(ctx context.Context, j store.Job) {
	if r.pub == nil {
		return
	}
	ev := progressEvent{JobID: j.ID.String(), Status: string(j.Status), Progress: int(j.Progress), Result: j.Result}
	if _, err := r.pub.Publish(ctx, j.OwnerID.String(), EventProgress, ev); err != nil {
		r.log.WarnContext(ctx, "không phát được job.progress", "job_id", j.ID.String(), "error", err.Error())
	}
}

// JobCtx là những gì một loại việc thấy. ctx của tin được giữ ở đây để `Progress(pct)` giữ đúng chữ ký ngắn
// (hàm việc vẫn nhận ctx riêng ở tham số đầu của KindFunc).
type JobCtx struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Payload json.RawMessage

	ctx context.Context //nolint:containedctx // ctx của tin, dùng cho Progress()
	tr  *progressTracker
}

// Progress báo tiến độ 0–100. Chỉ tăng: giá trị bằng hoặc nhỏ hơn lần trước bị bỏ qua.
// Mỗi mốc được ghi vào `jobs.progress` và phát SSE cho chủ việc.
func (j JobCtx) Progress(pct int) { j.tr.set(j.ctx, pct) }

type progressTracker struct {
	runner *Runner
	job    uuid.UUID
	owner  uuid.UUID

	mu   sync.Mutex
	last int
}

func (p *progressTracker) set(ctx context.Context, pct int) {
	switch {
	case pct > 100:
		pct = 100
	case pct < 0:
		return
	}
	p.mu.Lock()
	if pct <= p.last {
		p.mu.Unlock()
		return
	}
	p.last = pct
	p.mu.Unlock()

	r := p.runner
	row, err := store.New(r.db).UpdateJobProgress(ctx, store.UpdateJobProgressParams{ID: p.job, Progress: int16(pct)})
	if err != nil {
		r.log.WarnContext(ctx, "không ghi được tiến độ việc", "job_id", p.job.String(), "error", err.Error())
		return
	}
	// 100 % không phát riêng: sự kiện kết thúc (SUCCEEDED) đã mang progress 100 — tránh sự kiện thừa.
	if pct < 100 {
		r.publish(ctx, row)
	}
}
