// Package scheduler: ba làn ưu tiên, đồng thời toàn cục (ZSET Redis), token bucket RPM/TPM (Lua), mạch ngắt dùng chung,
// ngân sách; cài llm.Gate. Redis mất → giới hạn cục bộ theo tiến trình, vẫn gọi (FEAT-llm-gateway SRS 3.4, 4.3).
package scheduler

import (
	"context"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// Config là hằng số của Scheduler (biến môi trường LLM_*).
type Config struct {
	MaxConcurrency int
	BatchShare     float64
	QueueMax       int
	QueueWaitMax   time.Duration
	BreakerFails   int
	BreakerOpen    time.Duration
	AvgLatency     time.Duration // trễ trung bình mặc định khi chưa có số liệu (5 s)
}

func (c Config) withDefaults() Config {
	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = 10
	}
	if c.BatchShare <= 0 || c.BatchShare > 1 {
		c.BatchShare = 0.5
	}
	if c.QueueMax <= 0 {
		c.QueueMax = 200
	}
	if c.QueueWaitMax <= 0 {
		c.QueueWaitMax = 10 * time.Second
	}
	if c.BreakerFails <= 0 {
		c.BreakerFails = 5
	}
	if c.BreakerOpen <= 0 {
		c.BreakerOpen = 30 * time.Second
	}
	if c.AvgLatency <= 0 {
		c.AvgLatency = 5 * time.Second
	}
	return c
}

// Scheduler cài llm.Gate.
type Scheduler struct {
	cfg    Config
	log    *slog.Logger
	clk    clock.Clock
	remote *redisBackend // nil = chỉ cục bộ
	local  *localBackend
	budget *budget.Manager

	redisDown  atomic.Bool
	downAt     atomic.Int64 // unix nano lúc ghi nhận mất
	lastLogged atomic.Int64
	onRecover  func(context.Context)

	q      *queues
	latMu  sync.Mutex
	lat    map[string]*ring
	provMu sync.Mutex
	seen   map[string]struct{}
	pfx    string
}

var _ llm.Gate = (*Scheduler)(nil)

// Option tuỳ chỉnh.
type Option func(*Scheduler)

// WithBudget gắn Manager ngân sách.
func WithBudget(m *budget.Manager) Option { return func(s *Scheduler) { s.budget = m } }

// WithKeyPrefix đặt tiền tố cho khoá Redis toàn cục (wait, lastint) — dùng trong test để các test không đạp nhau.
func WithKeyPrefix(p string) Option { return func(s *Scheduler) { s.pfx = p } }

// WithClock thay đồng hồ của backend cục bộ (test).
func WithClock(c clock.Clock) Option { return func(s *Scheduler) { s.clk = c } }

// New dựng Scheduler. rdb nil = chỉ cục bộ.
func New(cfg Config, rdb *goredis.Client, log *slog.Logger, opts ...Option) *Scheduler {
	cfg = cfg.withDefaults()
	s := &Scheduler{cfg: cfg, log: log, clk: clock.Real{}, q: newQueues(), lat: map[string]*ring{}, seen: map[string]struct{}{}}
	for _, o := range opts {
		o(s)
	}
	s.local = newLocalBackend(s.clk, cfg.BreakerFails, cfg.BreakerOpen)
	if rdb != nil {
		s.remote = newRedisBackend(rdb, cfg.BreakerFails, cfg.BreakerOpen)
		s.remote.pfx = s.pfx
	}
	return s
}

// OnRedisRecover đăng ký hàm gọi khi Redis trở lại sau khi mất (đối soát ngân sách).
func (s *Scheduler) OnRedisRecover(f func(context.Context)) { s.onRecover = f }

// RedisDown cho biết Scheduler đang ở chế độ cục bộ.
func (s *Scheduler) RedisDown() bool { return s.redisDown.Load() }

// be chọn backend cho một thao tác: Redis nếu đang tốt (hoặc đã tới lúc thử lại), không thì cục bộ.
func (s *Scheduler) be(ctx context.Context) backend {
	if s.remote == nil {
		return s.local
	}
	if !s.redisDown.Load() {
		return s.remote
	}
	if time.Since(time.Unix(0, s.downAt.Load())) >= time.Second { // thử lại tối đa mỗi giây
		pctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		err := s.remote.rdb.Ping(pctx).Err()
		cancel()
		if err == nil {
			s.redisDown.Store(false)
			s.log.InfoContext(ctx, "Redis đã trở lại, quay về giới hạn toàn cục")
			if s.onRecover != nil {
				go s.onRecover(context.WithoutCancel(ctx)) //nolint:contextcheck // đối soát chạy nền, không gắn với request
			}
			return s.remote
		}
		s.downAt.Store(time.Now().UnixNano())
	}
	return s.local
}

// fail ghi nhận Redis lỗi: chuyển cục bộ, log error một lần mỗi 30 s.
func (s *Scheduler) fail(ctx context.Context, err error) {
	if ctx.Err() != nil {
		return // lỗi do ctx huỷ, không phải Redis
	}
	s.redisDown.Store(true)
	s.downAt.Store(time.Now().UnixNano())
	now := time.Now().Unix()
	if last := s.lastLogged.Load(); now-last >= 30 && s.lastLogged.CompareAndSwap(last, now) {
		s.log.ErrorContext(ctx, "Redis không dùng được, Scheduler chuyển sang giới hạn cục bộ", "error", err.Error())
	}
}

// op chạy một thao tác trên backend tốt nhất; lỗi Redis → ghi nhận và chạy lại trên cục bộ.
func op[T any](ctx context.Context, s *Scheduler, f func(b backend) (T, error)) T {
	b := s.be(ctx)
	v, err := f(b)
	if err != nil && b != backend(s.local) {
		s.fail(ctx, err)
		v, _ = f(s.local)
	}
	return v
}

// ---- hàng đợi cục bộ ----

type waiter struct {
	lane llm.Lane
	prov string
}

type queues struct {
	mu      sync.Mutex
	byProv  map[string]*[3][]*waiter
	count   [3]int
	changed chan struct{}
}

func newQueues() *queues {
	return &queues{byProv: map[string]*[3][]*waiter{}, changed: make(chan struct{})}
}

func (q *queues) lists(p string) *[3][]*waiter {
	l := q.byProv[p]
	if l == nil {
		l = &[3][]*waiter{}
		q.byProv[p] = l
	}
	return l
}

// push thêm vào cuối làn; false nếu làn đã đầy.
func (q *queues) push(w *waiter, max int) (bool, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.count[w.lane] >= max {
		return false, q.count[w.lane]
	}
	l := q.lists(w.prov)
	l[w.lane] = append(l[w.lane], w)
	q.count[w.lane]++
	return true, q.count[w.lane]
}

func (q *queues) remove(w *waiter) {
	q.mu.Lock()
	l := q.lists(w.prov)
	for i, x := range l[w.lane] {
		if x == w {
			l[w.lane] = append(l[w.lane][:i], l[w.lane][i+1:]...)
			q.count[w.lane]--
			break
		}
	}
	close(q.changed)
	q.changed = make(chan struct{})
	q.mu.Unlock()
}

// eligible: đầu hàng của làn mình và không còn ai ở làn ưu tiên cao hơn cùng nhà cung cấp. Trả kênh báo thay đổi để chờ.
func (q *queues) eligible(w *waiter) (bool, <-chan struct{}) {
	q.mu.Lock()
	defer q.mu.Unlock()
	l := q.lists(w.prov)
	for lane := range w.lane {
		if len(l[lane]) > 0 {
			return false, q.changed
		}
	}
	return len(l[w.lane]) > 0 && l[w.lane][0] == w, q.changed
}

func (q *queues) depth() [3]int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

// ---- trễ trung bình ----

type ring struct {
	v   [100]time.Duration
	n   int
	pos int
	sum time.Duration
}

func (r *ring) add(d time.Duration) {
	if r.n == len(r.v) {
		r.sum -= r.v[r.pos]
	} else {
		r.n++
	}
	r.v[r.pos] = d
	r.sum += d
	r.pos = (r.pos + 1) % len(r.v)
}

func (s *Scheduler) avgLatency(provider string) time.Duration {
	s.latMu.Lock()
	defer s.latMu.Unlock()
	if r := s.lat[provider]; r != nil && r.n > 0 {
		return r.sum / time.Duration(r.n)
	}
	return s.cfg.AvgLatency
}

func (s *Scheduler) recordLatency(provider string, d time.Duration) {
	s.latMu.Lock()
	r := s.lat[provider]
	if r == nil {
		r = &ring{}
		s.lat[provider] = r
	}
	r.add(d)
	s.latMu.Unlock()
}

// RetryAfter = clamp(ceil(độ_dài_hàng ÷ LLM_MAX_CONCURRENCY × trễ_trung_bình_s), 1, 30) giây.
func RetryAfter(queueLen, maxConc int, avg time.Duration) time.Duration {
	if maxConc < 1 {
		maxConc = 1
	}
	sec := math.Ceil(float64(queueLen) / float64(maxConc) * avg.Seconds())
	return time.Duration(min(30, max(1, sec))) * time.Second
}

// ---- Admit ----

type permit struct {
	s       *Scheduler
	prov    string
	member  string
	est     int
	tpm     int
	wait    time.Duration
	granted time.Time
	done    atomic.Bool
}

func (p *permit) Wait() time.Duration { return p.wait }

// Done trả chỗ, ghi trễ, đối soát TPM (hoàn / trừ phần chênh giữa ước tính và thật). Gọi nhiều lần chỉ có tác dụng lần đầu.
func (p *permit) Done(actual int) {
	if !p.done.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	p.s.recordLatency(p.prov, time.Since(p.granted))
	op(ctx, p.s, func(b backend) (struct{}, error) { return struct{}{}, b.release(ctx, p.prov, p.member) })
	// Cũng giải phóng bản cục bộ nếu chỗ được cấp khi Redis mất (member khác backend không sao: delete trên map rỗng).
	if p.s.remote != nil {
		_ = p.s.local.release(ctx, p.prov, p.member)
	}
	if actual >= 0 && actual != p.est && p.tpm > 0 {
		op(ctx, p.s, func(b backend) (struct{}, error) { return struct{}{}, b.reconcile(ctx, p.prov, p.tpm, p.est-actual) })
	}
}

func laneCode(l llm.Lane) string {
	switch l {
	case llm.LaneInteractive:
		return laneI
	case llm.LaneNearRealtime:
		return laneN
	}
	return laneB
}

// Admit xếp hàng theo làn rồi cấp chỗ + token. Lỗi: *llm.ErrOverloaded (hàng đầy / INTERACTIVE chờ quá LLM_QUEUE_WAIT_MAX),
// llm.ErrDeadline, hoặc lỗi ctx.
func (s *Scheduler) Admit(ctx context.Context, w llm.Work) (llm.Permit, error) {
	s.provMu.Lock()
	s.seen[w.ProviderID] = struct{}{}
	s.provMu.Unlock()

	wt := &waiter{lane: w.Lane, prov: w.ProviderID}
	ok, depth := s.q.push(wt, s.cfg.QueueMax)
	if !ok {
		return nil, &llm.ErrOverloaded{RetryAfter: RetryAfter(depth, s.cfg.MaxConcurrency, s.avgLatency(w.ProviderID))}
	}
	start := time.Now()
	lane := w.Lane.String()
	if w.Lane == llm.LaneInteractive {
		op(ctx, s, func(b backend) (struct{}, error) { return struct{}{}, b.noteWaiting(ctx, lane, 1) })
	}
	defer func() {
		s.q.remove(wt)
		if w.Lane == llm.LaneInteractive {
			op(context.WithoutCancel(ctx), s, func(b backend) (struct{}, error) {
				return struct{}{}, b.noteWaiting(context.WithoutCancel(ctx), lane, -1)
			})
		}
	}()

	member := laneCode(w.Lane) + ":" + uuid.NewString()
	batchCap := int(math.Ceil(float64(s.cfg.MaxConcurrency) * s.cfg.BatchShare))
	rpm, tpm := max(w.RPM, 1), max(w.TPM, 1)
	for {
		if err := ctx.Err(); err != nil {
			return nil, ctxErr(err)
		}
		if w.Lane == llm.LaneInteractive && time.Since(start) > s.cfg.QueueWaitMax {
			return nil, &llm.ErrOverloaded{RetryAfter: RetryAfter(s.q.depth()[w.Lane], s.cfg.MaxConcurrency, s.avgLatency(w.ProviderID))}
		}
		elig, changed := s.q.eligible(wt)
		if !elig {
			select {
			case <-changed:
			case <-ctx.Done():
				return nil, ctxErr(ctx.Err())
			case <-time.After(s.pollEvery()):
			}
			continue
		}
		granted, hint := s.tryGrant(ctx, w, member, batchCap, rpm, tpm)
		if granted {
			return &permit{s: s, prov: w.ProviderID, member: member, est: w.EstTokens, tpm: tpm, wait: time.Since(start), granted: time.Now()}, nil
		}
		d := s.pollEvery()
		if hint > d {
			d = min(hint, time.Second) // thiếu token: ngủ theo gợi ý của bucket
		}
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return nil, ctxErr(ctx.Err())
		}
	}
}

// pollEvery: 5–15 ms có jitter (thăm dò chỗ trống).
func (s *Scheduler) pollEvery() time.Duration {
	return 5*time.Millisecond + time.Duration(rand.Int64N(int64(10*time.Millisecond))) //nolint:gosec // jitter
}

func ctxErr(err error) error {
	if err == context.DeadlineExceeded {
		return llm.ErrDeadline
	}
	return err
}

// tryGrant lấy chỗ rồi token; token thiếu → trả chỗ ngay và báo thời gian chờ.
func (s *Scheduler) tryGrant(ctx context.Context, w llm.Work, member string, batchCap, rpm, tpm int) (bool, time.Duration) {
	lease := 130 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		lease = time.Until(dl) + 10*time.Second
	}
	req := leaseReq{provider: w.ProviderID, lane: laneCode(w.Lane), member: member, max: s.cfg.MaxConcurrency, batchCap: batchCap, lease: lease}
	b := s.be(ctx)
	got, err := b.tryLease(ctx, req)
	if err != nil && b != backend(s.local) {
		s.fail(ctx, err)
		b = s.local
		got, _ = b.tryLease(ctx, req)
	}
	if !got {
		return false, 0
	}
	okTok, wait, err := b.tryBucket(ctx, bucketReq{provider: w.ProviderID, rpm: rpm, tpm: tpm, cost: max(w.EstTokens, 1)})
	if err != nil && b != backend(s.local) {
		s.fail(ctx, err)
		_ = b.release(ctx, w.ProviderID, member)
		b = s.local
		_, _ = b.tryLease(ctx, req)
		okTok, wait, _ = b.tryBucket(ctx, bucketReq{provider: w.ProviderID, rpm: rpm, tpm: tpm, cost: max(w.EstTokens, 1)})
	}
	if !okTok {
		_ = b.release(context.WithoutCancel(ctx), w.ProviderID, member)
		return false, wait
	}
	return true, 0
}

// ---- mạch ngắt ----

// BreakerAllow: mạch đóng / bán mở lấy được lượt thử → true.
func (s *Scheduler) BreakerAllow(ctx context.Context, providerID string) bool {
	return op(ctx, s, func(b backend) (bool, error) { return b.cbAllow(ctx, providerID) })
}

// BreakerReport báo kết quả: k rỗng = thành công; các loại lỗi còn lại tính vào mạch (Gateway chỉ gọi với loại có CountsToBreaker).
func (s *Scheduler) BreakerReport(ctx context.Context, providerID string, k provider.Kind) {
	op(context.WithoutCancel(ctx), s, func(b backend) (struct{}, error) {
		return struct{}{}, b.cbReport(context.WithoutCancel(ctx), providerID, k != "")
	})
}

// CircuitState: closed | open | half_open.
func (s *Scheduler) CircuitState(ctx context.Context, providerID string) string {
	return op(ctx, s, func(b backend) (string, error) { return b.cbState(ctx, providerID) })
}

// ---- ngân sách ----

// BudgetState ánh xạ trạng thái của Manager.
func (s *Scheduler) BudgetState(ctx context.Context, courseID *uuid.UUID) llm.BudgetState {
	if s.budget == nil || s.redisDown.Load() {
		return llm.BudgetOK
	}
	switch s.budget.State(ctx, courseID) {
	case budget.Exhausted:
		return llm.BudgetExhausted
	case budget.Warn:
		return llm.BudgetWarn
	}
	return llm.BudgetOK
}

// BudgetCharge cộng chi phí (bỏ qua khi Redis mất — đối soát sau).
func (s *Scheduler) BudgetCharge(ctx context.Context, courseID *uuid.UUID, c decimal.Decimal) {
	if s.budget == nil || s.redisDown.Load() {
		return
	}
	s.budget.Charge(context.WithoutCancel(ctx), courseID, c)
}

// ---- thống kê ----

// Stats là số đếm cho route thử `_test/llm/stats` (không nội dung).
type Stats struct {
	QueueDepth       map[string]int    `json:"queue_depth"`
	Inflight         map[string]int    `json:"inflight"`
	InflightBatch    map[string]int    `json:"inflight_batch"`
	ProviderInflight int               `json:"provider_inflight"`
	Circuit          map[string]string `json:"circuit"`
	RedisDown        bool              `json:"redis_down"`
}

// Stats đọc số đếm hiện hành.
func (s *Scheduler) Stats(ctx context.Context) Stats {
	d := s.q.depth()
	st := Stats{QueueDepth: map[string]int{"INTERACTIVE": d[0], "NEAR_REALTIME": d[1], "BATCH": d[2]},
		Inflight: map[string]int{}, InflightBatch: map[string]int{}, Circuit: map[string]string{}, RedisDown: s.redisDown.Load()}
	s.provMu.Lock()
	provs := make([]string, 0, len(s.seen))
	for p := range s.seen {
		provs = append(provs, p)
	}
	s.provMu.Unlock()
	for _, p := range provs {
		type cnt struct{ total, batch int }
		c := op(ctx, s, func(b backend) (cnt, error) {
			t, bt, err := b.counts(ctx, p)
			return cnt{t, bt}, err
		})
		st.Inflight[p], st.InflightBatch[p] = c.total, c.batch
		st.ProviderInflight += c.total
		st.Circuit[p] = s.CircuitState(ctx, p)
	}
	return st
}
