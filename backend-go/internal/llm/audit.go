package llm

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Tham số của bộ ghi llm_audit (US-P1-02 AC8).
const (
	auditBuffer = 1000
	auditBatch  = 100
	auditEvery  = time.Second
)

// AuditRow là MỘT dòng llm_audit; không có trường nội dung (prompt / câu trả lời).
type AuditRow struct {
	Task, Lane, Provider, Model string
	TokensIn, TokensOut         int
	LatencyMS, QueueWaitMS      int
	Attempts, FallbackIndex     int
	CostEst                     decimal.Decimal
	Status, ErrorKind           string
	Degraded                    bool
	PIIMaskedCount              int
	UserID, CourseID            *uuid.UUID
	TraceID                     string
	At                          time.Time
}

// AuditWriter ghi một lô; mặc định là COPY vào Postgres.
type AuditWriter func(ctx context.Context, rows []AuditRow) error

// Auditor đệm dòng audit trong bộ nhớ và ghi bất đồng bộ theo lô: lỗi ghi không bao giờ làm hỏng lời gọi LLM.
type Auditor struct {
	write   AuditWriter
	log     *slog.Logger
	mu      sync.Mutex
	buf     []AuditRow
	wake    chan struct{}
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
	dropped atomic.Int64
	flushed atomic.Int64
	lastLog atomic.Int64 // unix giây của lần log cảnh báo bỏ dòng gần nhất
}

// NewAuditor dựng và chạy vòng ghi nền. Gọi Close để đẩy hết trước khi thoát.
func NewAuditor(ctx context.Context, w AuditWriter, log *slog.Logger) *Auditor {
	a := &Auditor{write: w, log: log, wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	go a.loop(context.WithoutCancel(ctx))
	return a
}

// Record thêm một dòng; đệm đầy → bỏ dòng cũ nhất, tăng bộ đếm `llm_audit_dropped` và log warn (tối đa mỗi 10 s).
func (a *Auditor) Record(r AuditRow) {
	if a == nil {
		return
	}
	a.mu.Lock()
	if len(a.buf) >= auditBuffer {
		a.buf = a.buf[1:]
		n := a.dropped.Add(1)
		if now := time.Now().Unix(); now-a.lastLog.Load() >= 10 {
			a.lastLog.Store(now)
			a.log.Warn("llm_audit đầy, bỏ dòng cũ nhất", "llm_audit_dropped", n)
		}
	}
	a.buf = append(a.buf, r)
	full := len(a.buf) >= auditBatch
	a.mu.Unlock()
	if full {
		select {
		case a.wake <- struct{}{}:
		default:
		}
	}
}

// Dropped là bộ đếm `llm_audit_dropped`.
func (a *Auditor) Dropped() int64 { return a.dropped.Load() }

// Flushed là số dòng đã ghi thành công.
func (a *Auditor) Flushed() int64 { return a.flushed.Load() }

// Pending là số dòng chưa ghi.
func (a *Auditor) Pending() int { a.mu.Lock(); defer a.mu.Unlock(); return len(a.buf) }

func (a *Auditor) loop(ctx context.Context) {
	defer close(a.done)
	t := time.NewTicker(auditEvery)
	defer t.Stop()
	for {
		select {
		case <-t.C:
		case <-a.wake:
		case <-a.stop:
			return
		}
		a.flush(ctx)
	}
}

func (a *Auditor) take(n int) []AuditRow {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n > len(a.buf) {
		n = len(a.buf)
	}
	out := append([]AuditRow(nil), a.buf[:n]...)
	a.buf = a.buf[n:]
	return out
}

func (a *Auditor) flush(ctx context.Context) {
	for {
		rows := a.take(auditBatch)
		if len(rows) == 0 {
			return
		}
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		err := a.write(wctx, rows)
		cancel()
		if err == nil {
			a.flushed.Add(int64(len(rows)))
		}
		if err != nil {
			a.log.Warn("không ghi được llm_audit", "rows", len(rows), "error", err.Error())
		}
	}
}

// Close dừng vòng nền rồi đẩy hết dòng còn lại (đang tắt gateway). An toàn khi gọi nhiều lần.
func (a *Auditor) Close(ctx context.Context) {
	if a == nil {
		return
	}
	a.once.Do(func() { close(a.stop) })
	<-a.done
	a.flush(ctx)
}

// PGWriter ghi lô bằng COPY qua sqlc (`InsertLLMAudit :copyfrom`).
func PGWriter(db store.DBTX) AuditWriter {
	q := store.New(db)
	return func(ctx context.Context, rows []AuditRow) error {
		ps := make([]store.InsertLLMAuditParams, 0, len(rows))
		for _, r := range rows {
			p := store.InsertLLMAuditParams{
				Task: r.Task, Lane: r.Lane, TokensIn: int32(r.TokensIn), TokensOut: int32(r.TokensOut), //nolint:gosec // đếm token không tràn int32
				LatencyMs: int32(r.LatencyMS), QueueWaitMs: int32(r.QueueWaitMS), Attempts: int32(r.Attempts), //nolint:gosec // idem
				FallbackIndex: int32(r.FallbackIndex), CostEst: r.CostEst, Status: r.Status, Degraded: r.Degraded, //nolint:gosec // idem
				PiiMaskedCount: int32(r.PIIMaskedCount), UserID: r.UserID, CourseID: r.CourseID, TraceID: r.TraceID, CreatedAt: r.At, //nolint:gosec // idem
			}
			if r.Provider != "" {
				p.Provider = &r.Provider
			}
			if r.Model != "" {
				p.Model = &r.Model
			}
			if r.ErrorKind != "" {
				p.ErrorKind = &r.ErrorKind
			}
			ps = append(ps, p)
		}
		_, err := q.InsertLLMAudit(ctx, ps)
		return err
	}
}
