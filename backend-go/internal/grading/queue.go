package grading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/notify"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Tên Stream / nhóm / loại việc (SRS 4.7).
const (
	StreamGrading     = "ep:grading"
	StreamGradingDead = "ep:grading.dead"
	GroupGrading      = "grading"
	KindBatch         = "grading.batch"

	// ErrSuperseded: bản nộp đã có bản mới hơn — việc kết thúc không gọi LLM; không hiện ở hàng chờ.
	ErrSuperseded = "SUPERSEDED"
	// TypeBudget: một thông báo gộp cho nhân viên khi cả lô hết ngân sách (không một thông báo mỗi bài).
	TypeBudget = "GRADING_BUDGET_EXHAUSTED"

	batchTTL = 24 * time.Hour
)

// QueueConfig là các hằng của SRS 4.7 (env `GRADING_*`).
type QueueConfig struct {
	Workers      int             // GRADING_WORKERS (2): goroutine mỗi worker
	Lease        time.Duration   // GRADING_LEASE (60 s)
	Heartbeat    time.Duration   // GRADING_HEARTBEAT (30 s)
	RequeueAfter time.Duration   // GRADING_REQUEUE_AFTER (60 s)
	Attempts     int             // GRADING_ATTEMPTS (3)
	Backoff      []time.Duration // GRADING_BACKOFF (5 s, 20 s)
	Poll         time.Duration   // chờ đọc Stream mỗi vòng
	Tick         time.Duration   // chu kỳ máy quét
}

// DefaultQueueConfig là giá trị mặc định của SRS 4.7.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{Workers: 2, Lease: 60 * time.Second, Heartbeat: 30 * time.Second, RequeueAfter: 60 * time.Second, Attempts: 3,
		Backoff: []time.Duration{5 * time.Second, 20 * time.Second}, Poll: 500 * time.Millisecond, Tick: 5 * time.Second}
}

// Metrics là bộ đếm KHÔNG PII (SRS 4.7 AC10): chỉ số đếm theo trạng thái, độ sâu, thời lượng. `/observability` (P10) đọc Snapshot.
// ponytail: bộ đếm trong tiến trình (mỗi worker một bản); xuất Prometheus khi P10 dựng đường thu thập.
type Metrics struct {
	mu        sync.Mutex
	states    map[string]int64
	dead      atomic.Int64
	durations []float64 // giây; giữ tối đa 1.000 mẫu gần nhất
	Depth     atomic.Int64
}

// MetricsSnapshot là ảnh chụp của Metrics.
type MetricsSnapshot struct {
	JobsTotal map[string]int64 `json:"grading_jobs_total"`
	DeadTotal int64            `json:"grading_dead_total"`
	QueueSize int64            `json:"grading_queue_depth"`
	P50, P95  float64
}

func (m *Metrics) inc(state string) {
	m.mu.Lock()
	if m.states == nil {
		m.states = map[string]int64{}
	}
	m.states[state]++
	m.mu.Unlock()
}

func (m *Metrics) observe(d time.Duration) {
	m.mu.Lock()
	m.durations = append(m.durations, d.Seconds())
	if len(m.durations) > 1000 {
		m.durations = m.durations[len(m.durations)-1000:]
	}
	m.mu.Unlock()
}

// Snapshot chụp bộ đếm.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := MetricsSnapshot{JobsTotal: map[string]int64{}, DeadTotal: m.dead.Load(), QueueSize: m.Depth.Load()}
	for k, v := range m.states {
		s.JobsTotal[k] = v
	}
	if n := len(m.durations); n > 0 {
		d := append([]float64(nil), m.durations...)
		sort.Float64s(d)
		s.P50, s.P95 = d[n*50/100], d[min(n-1, n*95/100)]
	}
	return s
}

// Queue là consumer chấm bài: tín hiệu ở Redis Stream `ep:grading`, trạng thái nhận việc / thuê / thử lại ở DB (SRS 4.7.3). Theo mẫu `judge/queue.go`;
// KHÔNG chạy trong consumer outbox (một bài chậm không được chặn `today.invalidate`, thư, thông báo).
type Queue struct {
	Pool   *pgxpool.Pool
	Redis  *appredis.Client
	Grader *Grader
	Jobs   *jobs.Runner
	Notify *notify.Service
	Log    *slog.Logger
	Cfg    QueueConfig
	M      *Metrics
	Prefix string // chỉ test dùng để cô lập Stream
	Name   string // tên consumer ổn định giữa các lần khởi động

	wg sync.WaitGroup
}

// NewQueue đặt mặc định MỘT lần trước khi Run / Tick chạy ở các goroutine khác nhau.
func NewQueue(q *Queue) *Queue {
	d := DefaultQueueConfig()
	c := &q.Cfg
	c.Workers = cmpOr(c.Workers, d.Workers)
	c.Lease = cmpOr(c.Lease, d.Lease)
	c.Heartbeat = cmpOr(c.Heartbeat, d.Heartbeat)
	c.RequeueAfter = cmpOr(c.RequeueAfter, d.RequeueAfter)
	c.Attempts = cmpOr(c.Attempts, d.Attempts)
	c.Poll = cmpOr(c.Poll, d.Poll)
	c.Tick = cmpOr(c.Tick, d.Tick)
	if len(c.Backoff) == 0 {
		c.Backoff = d.Backoff
	}
	if q.Name == "" {
		q.Name = "grading-1"
	}
	if q.M == nil {
		q.M = &Metrics{}
	}
	if q.Log == nil {
		q.Log = slog.Default()
	}
	return q
}

func cmpOr[T comparable](v, d T) T {
	var zero T
	if v == zero {
		return d
	}
	return v
}

func (q *Queue) stream() string { return q.Prefix + StreamGrading }

// Signal XADD một tín hiệu (sau COMMIT; idempotent vì bước nhận việc ở DB chặn việc đôi).
func (q *Queue) Signal(ctx context.Context, course, submission, result uuid.UUID, round int16) error {
	return q.Redis.XAdd(ctx, &goredis.XAddArgs{Stream: q.stream(), Values: map[string]any{
		"course_id": course.String(), "submission_id": submission.String(), "round": strconv.Itoa(int(round)), "result_id": result.String()}}).Err()
}

// BatchPayload là payload của việc `grading.batch`.
type BatchPayload struct {
	CourseID  uuid.UUID   `json:"course_id"`
	ResultIDs []uuid.UUID `json:"result_ids"`
}

func batchKey(prefix string, job uuid.UUID) string { return prefix + "ep:gradebatch:" + job.String() }
func resultKey(prefix string, r uuid.UUID) string  { return prefix + "ep:gradejob:" + r.String() }

// RegisterBatch đăng ký `grading.batch`: handler (chạy trong consumer outbox SAU COMMIT) chỉ XADD rồi ErrDeferred — không chấm trong consumer outbox.
// Tiến độ + kết quả do Queue ghi (jobs.Runner.Report / Complete) khi từng bài xong.
func (q *Queue) RegisterBatch(r *jobs.Runner) {
	r.Register(KindBatch, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		var p BatchPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil || len(p.ResultIDs) == 0 {
			return nil, fmt.Errorf("grading: payload %s hỏng", KindBatch)
		}
		for _, id := range p.ResultIDs {
			if err := q.Redis.SAdd(ctx, batchKey(q.Prefix, j.ID), id.String()).Err(); err != nil {
				return nil, fmt.Errorf("grading: SADD: %w", err)
			}
			if err := q.Redis.Set(ctx, resultKey(q.Prefix, id), j.ID.String(), batchTTL).Err(); err != nil {
				return nil, fmt.Errorf("grading: SET: %w", err)
			}
		}
		q.Redis.Expire(ctx, batchKey(q.Prefix, j.ID), batchTTL)
		rows, err := store.New(q.Pool).GradingBatchRows(ctx, uuidStrings(p.ResultIDs))
		if err != nil {
			return nil, fmt.Errorf("grading: đọc lô: %w", err)
		}
		for _, r := range rows {
			if err := q.Signal(ctx, r.CourseID, r.SubmissionID, r.ID, r.Round); err != nil {
				return nil, fmt.Errorf("grading: XADD: %w", err)
			}
		}
		return nil, jobs.ErrDeferred
	})
}

func uuidStrings(ids []uuid.UUID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.String()
	}
	return s
}

// Run chạy tới khi ctx huỷ: tạo nhóm, máy quét, `Workers` goroutine đọc nhóm.
func (q *Queue) Run(ctx context.Context) error {
	if err := q.Redis.XGroupCreateMkStream(ctx, q.stream(), GroupGrading, "0").Err(); err != nil && !isBusyGroup(err) {
		return fmt.Errorf("grading: tạo nhóm: %w", err)
	}
	var bg sync.WaitGroup
	bg.Add(1)
	go func() { defer bg.Done(); q.tickLoop(ctx) }()
	for i := 0; i < q.Cfg.Workers; i++ {
		name := fmt.Sprintf("%s-%d", q.Name, i)
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			q.drainPending(ctx, name)
			for ctx.Err() == nil {
				if m, ok := q.next(ctx, name); ok {
					q.process(context.WithoutCancel(ctx), m)
				}
			}
		}()
	}
	q.wg.Wait()
	bg.Wait()
	return ctx.Err()
}

func isBusyGroup(err error) bool {
	return err != nil && len(err.Error()) >= 9 && err.Error()[:9] == "BUSYGROUP"
}

// next: ưu tiên tin treo quá thuê (worker chết), rồi tin mới.
func (q *Queue) next(ctx context.Context, consumer string) (goredis.XMessage, bool) {
	if msgs, _, err := q.Redis.XAutoClaim(ctx, &goredis.XAutoClaimArgs{Stream: q.stream(), Group: GroupGrading, Consumer: consumer, MinIdle: q.Cfg.Lease, Start: "0-0", Count: 1}).Result(); err == nil && len(msgs) > 0 {
		return msgs[0], true
	}
	rs, err := q.Redis.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: GroupGrading, Consumer: consumer, Streams: []string{q.stream(), ">"}, Count: 1, Block: q.Cfg.Poll}).Result()
	if err != nil || len(rs) == 0 || len(rs[0].Messages) == 0 {
		if err != nil && !errors.Is(err, goredis.Nil) && ctx.Err() == nil {
			q.Log.WarnContext(ctx, "grading: đọc Stream lỗi", "error", err.Error())
			select {
			case <-ctx.Done():
			case <-time.After(q.Cfg.Poll):
			}
		}
		return goredis.XMessage{}, false
	}
	return rs[0].Messages[0], true
}

// drainPending xử lý tin còn treo của CHÍNH consumer này (nhận trước khi khởi động lại): bước nhận việc ở DB quyết định có chấm hay không.
func (q *Queue) drainPending(ctx context.Context, consumer string) {
	for ctx.Err() == nil {
		rs, err := q.Redis.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: GroupGrading, Consumer: consumer, Streams: []string{q.stream(), "0"}, Count: 10, Block: -1}).Result()
		if err != nil || len(rs) == 0 || len(rs[0].Messages) == 0 {
			return
		}
		for _, m := range rs[0].Messages {
			q.process(ctx, m)
		}
	}
}

func (q *Queue) tickLoop(ctx context.Context) {
	t := time.NewTicker(q.Cfg.Tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			q.Tick(ctx)
		}
	}
}

// Tick là máy quét: nạp lại QUEUED đã tới giờ (mất tin Redis, tới giờ thử lại) và RUNNING hết thuê; cập nhật độ sâu hàng.
func (q *Queue) Tick(ctx context.Context) {
	s := store.New(q.Pool)
	rows, err := s.GradingSweep(ctx, q.Cfg.RequeueAfter.Seconds())
	if err != nil {
		q.Log.WarnContext(ctx, "grading: quét QUEUED lỗi", "error", err.Error())
	}
	for _, r := range rows {
		if err := q.Signal(ctx, r.CourseID, r.SubmissionID, r.ID, r.Round); err != nil {
			q.Log.WarnContext(ctx, "grading: nạp lại lỗi", "error", err.Error())
			break
		}
	}
	exp, err := s.GradingExpiredLeases(ctx)
	if err != nil {
		q.Log.WarnContext(ctx, "grading: quét thuê lỗi", "error", err.Error())
	}
	for _, r := range exp {
		if err := q.Signal(ctx, r.CourseID, r.SubmissionID, r.ID, r.Round); err != nil {
			break
		}
	}
	if n, err := s.GradingQueueDepth(ctx); err == nil {
		q.M.Depth.Store(int64(n))
	}
}

func (q *Queue) ack(ctx context.Context, m goredis.XMessage) {
	if err := q.Redis.XAck(context.WithoutCancel(ctx), q.stream(), GroupGrading, m.ID).Err(); err != nil {
		q.Log.WarnContext(ctx, "grading: XACK lỗi", "error", err.Error())
	}
}

// process xử lý một tin (mọi UPDATE chỉ ảnh hưởng đúng dòng nếu điều kiện còn đúng).
func (q *Queue) process(ctx context.Context, m goredis.XMessage) {
	defer q.ack(ctx, m)
	rid, err := uuid.Parse(fmt.Sprint(m.Values["result_id"]))
	if err != nil {
		return
	}
	s := store.New(q.Pool)
	// Bản cũ bị thay: không gọi LLM.
	if sup, err := s.GradingIsSuperseded(ctx, rid); err == nil && sup {
		if n, _ := s.GradingFailPermanent(ctx, store.GradingFailPermanentParams{ID: rid, Code: new(ErrSuperseded), Owner: ""}); n > 0 {
			q.M.inc("superseded")
			q.progress(ctx, rid)
		}
		return
	}
	owner := uuid.NewString()
	row, err := s.GradingClaim(ctx, store.GradingClaimParams{ID: rid, Owner: &owner, LeaseSecs: q.Cfg.Lease.Seconds()})
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		q.Log.ErrorContext(ctx, "grading: nhận việc lỗi", "error", err.Error())
		return
	}
	started := time.Now()
	jctx, cancel := context.WithCancel(ctx)
	var lost atomic.Bool
	hb := make(chan struct{})
	go func() {
		defer close(hb)
		t := time.NewTicker(q.Cfg.Heartbeat)
		defer t.Stop()
		for {
			select {
			case <-jctx.Done():
				return
			case <-t.C:
				n, err := s.GradingRenew(jctx, store.GradingRenewParams{ID: rid, Owner: &owner, LeaseSecs: q.Cfg.Lease.Seconds()})
				if err == nil && n == 0 {
					lost.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	out, gerr := q.Grader.Run(WithLease(jctx, owner), row.CourseID, rid)
	cancel()
	<-hb
	q.M.observe(time.Since(started))
	if lost.Load() || errors.Is(gerr, ErrLeaseLost) {
		q.Log.WarnContext(ctx, "grading: mất thuê — bỏ kết quả", "result_id", rid)
		q.M.inc("lease_lost")
		return
	}
	if gerr != nil {
		q.retryOrFail(ctx, s, rid, owner, row.Attempts, gerr)
		return
	}
	q.M.inc(map[string]string{"DONE": "done", "FAILED": "failed"}[out.State])
	q.progress(ctx, rid)
}

// retryOrFail: lỗi tạm còn lượt → QUEUED + not_before; hết lượt → FAILED + `.dead` (chỉ để đếm).
func (q *Queue) retryOrFail(ctx context.Context, s *store.Queries, rid uuid.UUID, owner string, attempts int16, gerr error) {
	code := "INTERNAL"
	var te *TransientError
	if errors.As(gerr, &te) {
		code = te.Code
	}
	if int(attempts) < q.Cfg.Attempts {
		d := q.Cfg.Backoff[min(int(attempts)-1, len(q.Cfg.Backoff)-1)]
		if n, _ := s.GradingRequeue(ctx, store.GradingRequeueParams{ID: rid, Owner: &owner, DelaySecs: d.Seconds()}); n > 0 {
			q.M.inc("retry")
		}
		return
	}
	n, err := s.GradingFailPermanent(ctx, store.GradingFailPermanentParams{ID: rid, Code: &code, Owner: owner})
	if err != nil || n == 0 {
		return
	}
	q.M.inc("failed")
	q.M.dead.Add(1)
	q.Redis.XAdd(ctx, &goredis.XAddArgs{Stream: q.Prefix + StreamGradingDead, MaxLen: 10000, Approx: true, Values: map[string]any{"result_id": rid.String(), "code": code}})
	q.progress(ctx, rid)
}

// progress cập nhật tiến độ của lô chứa `rid` (nếu có); lô xong thì Complete + (nếu hết ngân sách) MỘT thông báo gộp.
func (q *Queue) progress(ctx context.Context, rid uuid.UUID) {
	if q.Jobs == nil {
		return
	}
	jid, err := q.Redis.Get(ctx, resultKey(q.Prefix, rid)).Result()
	if err != nil {
		return
	}
	job, err := uuid.Parse(jid)
	if err != nil {
		return
	}
	ids, err := q.Redis.SMembers(ctx, batchKey(q.Prefix, job)).Result()
	if err != nil || len(ids) == 0 {
		return
	}
	c, err := store.New(q.Pool).GradingBatchCounts(ctx, ids)
	if err != nil {
		return
	}
	fin := int(c.Done + c.Failed)
	if fin < int(c.Total) {
		q.Jobs.Report(ctx, job, fin*100/int(c.Total))
		return
	}
	if q.Redis.SetNX(ctx, batchKey(q.Prefix, job)+":done", "1", batchTTL).Val() {
		if c.Budget > 0 {
			q.budgetNotice(ctx, ids[0])
		}
		if err := q.Jobs.Complete(ctx, job, map[string]int32{"graded": c.Done, "failed": c.Failed, "flagged": c.Flagged}); err != nil {
			q.Log.WarnContext(ctx, "grading: không ghi kết quả lô", "error", err.Error())
		}
	}
}

func (q *Queue) budgetNotice(ctx context.Context, anyResult string) {
	if q.Notify == nil {
		return
	}
	id, err := uuid.Parse(anyResult)
	if err != nil {
		return
	}
	s := store.New(q.Pool)
	st, err := s.GradingBatchRows(ctx, []string{anyResult})
	if err != nil || len(st) == 0 {
		return
	}
	staff, err := s.GradingStaffIDs(ctx, st[0].CourseID)
	if err != nil {
		return
	}
	course := st[0].CourseID
	_, _ = q.Notify.SendMany(ctx, staff, notify.Spec{CourseID: &course, Type: TypeBudget, Title: "Hết hạn mức chấm hôm nay, các bài còn lại chưa được chấm",
		Link: new("/grading"), DedupeKey: "grading.budget:" + id.String()})
}

// JobScope cho nhân viên (TEACHER / TA) cùng lớp đọc việc `grading.batch` của đồng nghiệp (AC9); ADMIN và người ngoài lớp không đọc được.
// Lớp suy ra từ lô lưu ở Redis (24 giờ); hết hạn thì chỉ người tạo đọc.
type JobScope struct {
	Pool   *pgxpool.Pool
	Redis  *appredis.Client
	Prefix string
}

func (JobScope) Kind() string { return KindBatch }

func (s JobScope) Allowed(ctx context.Context, p auth.Principal, jobID uuid.UUID) bool {
	uid, err := uuid.Parse(p.Sub)
	if err != nil || p.Role == auth.RoleAdmin {
		return false
	}
	ids, err := s.Redis.SRandMemberN(ctx, batchKey(s.Prefix, jobID), 1).Result()
	if err != nil || len(ids) == 0 {
		return false
	}
	q := store.New(s.Pool)
	rows, err := q.GradingBatchRows(ctx, ids)
	if err != nil || len(rows) == 0 {
		return false
	}
	ok, err := q.GradingIsStaff(ctx, store.GradingIsStaffParams{CourseID: rows[0].CourseID, UserID: uid})
	return err == nil && ok
}
