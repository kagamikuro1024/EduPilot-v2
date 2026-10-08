package judge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Tên Stream / nhóm (SRS 5.15; Stream không tiền tố).
const (
	StreamSubmit     = "judge.submit"
	StreamRun        = "judge.run"
	StreamSubmitDead = "judge.submit.dead"
	StreamRunDead    = "judge.run.dead"
	Group            = "judge"
	// TopicEnqueue là topic outbox ghi CÙNG transaction với `code_submissions`; handler XADD tín hiệu rồi đặt `enqueued_at`.
	TopicEnqueue = "judge.enqueue"
	// TopicDone: sự kiện outbox sau khi một bản nộp xong (DONE) hoặc hết lần thử (ERROR) — US-PE-08 chấm lượt làm.
	TopicDone = "exam.submission_done"
	// UpKey: khoá Redis do worker đặt sau mỗi lần thăm dò thành công, xoá khi lỗi; gateway chỉ đọc (SRS 5.15).
	upKeySuffix = "up"
	deadMaxLen  = 10000
)

// Enqueue là payload của outbox `judge.enqueue`.
type Enqueue struct {
	SubmissionID uuid.UUID `json:"submission_id"`
	Kind         string    `json:"kind"`
}

// BlobReader đọc test lớn lưu ở object storage (`input_blob_key`); nil → chỉ chấm test lưu trong DB.
type BlobReader interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// Queue là consumer chấm bài: tín hiệu ở Redis Stream, trạng thái nhận việc / thuê / thử lại ở DB (SRS 4.5.7). Chạy ở ĐÚNG một bản worker (4.5.8).
type Queue struct {
	Pool      *pgxpool.Pool
	Redis     *appredis.Client
	Judge     *Client
	Clock     clock.Clock
	Log       *slog.Logger
	Blob      BlobReader
	P         int             // JUDGE_PARALLELISM: số chỗ chạy
	Prefix    string          // tiền tố tên Stream (chỉ test dùng để cô lập; sản xuất để trống)
	Name      string          // tên consumer (ổn định giữa các lần khởi động để đọc lại tin còn treo của chính mình)
	Lease     time.Duration   // JUDGE_LEASE (30 s)
	Renew     time.Duration   // JUDGE_LEASE_RENEW (10 s)
	Idle      time.Duration   // JUDGE_CLAIM_IDLE (10 phút)
	TickEvery time.Duration   // 5 s
	Probe     time.Duration   // 10 s
	Poll      time.Duration   // thời gian chờ đọc Stream mỗi vòng
	Backoff   []time.Duration // 1 s, 5 s, 30 s theo fail_count

	up        atomic.Bool
	downSince atomic.Int64
	warned    atomic.Bool
	running   atomic.Int64 // số bản đang chạy
	runningSb atomic.Int64 // trong đó SUBMIT
	wg        sync.WaitGroup
}

func (q *Queue) defaults() {
	if q.Lease <= 0 {
		q.Lease = 30 * time.Second
	}
	if q.Renew <= 0 {
		q.Renew = 10 * time.Second
	}
	if q.Idle <= 0 {
		q.Idle = 10 * time.Minute
	}
	if q.TickEvery <= 0 {
		q.TickEvery = 5 * time.Second
	}
	if q.Probe <= 0 {
		q.Probe = 10 * time.Second
	}
	if q.Poll <= 0 {
		q.Poll = 200 * time.Millisecond
	}
	if len(q.Backoff) == 0 {
		q.Backoff = []time.Duration{time.Second, 5 * time.Second, 30 * time.Second}
	}
	if q.P <= 0 {
		q.P = 2
	}
	if q.Name == "" {
		q.Name = "judge-1"
	}
	if q.Clock == nil {
		q.Clock = clock.Real{}
	}
}

// Up cho biết lần thăm dò gần nhất có thành công không (`/readyz` của worker, chỉ thông tin).
func (q *Queue) Up() bool { return q.up.Load() }

func (q *Queue) streamFor(kind string) (stream, dead string) {
	if kind == string(store.SubmissionKindRUN) {
		return q.Prefix + StreamRun, q.Prefix + StreamRunDead
	}
	return q.Prefix + StreamSubmit, q.Prefix + StreamSubmitDead
}

// HandleEnqueue là handler outbox của topic `judge.enqueue`: XADD tín hiệu `{submission_id}` rồi đặt `enqueued_at` (idempotent: tín hiệu thừa vô hại vì bước nhận việc chặn).
func (q *Queue) HandleEnqueue(ctx context.Context, m outbox.Message) error {
	q.defaults()
	var p Enqueue
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.SubmissionID == uuid.Nil {
		return fmt.Errorf("judge.enqueue: payload hỏng")
	}
	return q.signal(ctx, p.SubmissionID, p.Kind)
}

func (q *Queue) signal(ctx context.Context, id uuid.UUID, kind string) error {
	stream, _ := q.streamFor(kind)
	if err := q.Redis.XAdd(ctx, &goredis.XAddArgs{Stream: stream, Values: map[string]any{"submission_id": id.String()}}).Err(); err != nil {
		return fmt.Errorf("XADD %s: %w", stream, err)
	}
	if _, err := store.New(q.Pool).JudgeMarkEnqueued(ctx, id); err != nil {
		return fmt.Errorf("đặt enqueued_at: %w", err)
	}
	return nil
}

// Run chạy tới khi ctx huỷ: tạo nhóm, thăm dò judge, tick, rồi vòng đọc Stream chỉ khi có chỗ chạy.
func (q *Queue) Run(ctx context.Context) error {
	q.defaults()
	for _, s := range []string{q.Prefix + StreamSubmit, q.Prefix + StreamRun} {
		if err := q.Redis.XGroupCreateMkStream(ctx, s, Group, "0").Err(); err != nil && !isBusyGroup(err) {
			return fmt.Errorf("tạo nhóm %s: %w", s, err)
		}
	}
	q.probeOnce(ctx)
	var bg sync.WaitGroup
	bg.Add(2)
	go func() { defer bg.Done(); q.probeLoop(ctx) }()
	go func() { defer bg.Done(); q.tickLoop(ctx) }()

	q.drainPending(ctx)
	slots := make(chan struct{}, q.P)
	for ctx.Err() == nil {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
		kind, entry, ok := q.next(ctx)
		if !ok {
			<-slots
			continue
		}
		q.wg.Add(1)
		q.running.Add(1)
		if kind == string(store.SubmissionKindSUBMIT) {
			q.runningSb.Add(1)
		}
		go func() {
			defer func() {
				<-slots
				q.running.Add(-1)
				if kind == string(store.SubmissionKindSUBMIT) {
					q.runningSb.Add(-1)
				}
				q.wg.Done()
			}()
			q.process(context.WithoutCancel(ctx), kind, entry)
		}()
	}
	q.wg.Wait()
	bg.Wait()
	return ctx.Err()
}

func isBusyGroup(err error) bool {
	return err != nil && len(err.Error()) >= 9 && err.Error()[:9] == "BUSYGROUP"
}

// next đọc MỘT tin: `Chạy thử` dùng bất kỳ chỗ trống; `Nộp lời giải` tối đa max(1, P-1) chỗ KHI có `Chạy thử` đang chờ, ngược lại đủ P (4.5.8).
func (q *Queue) next(ctx context.Context) (kind string, e goredis.XMessage, ok bool) {
	if m, ok := q.read(ctx, q.Prefix+StreamRun, 0); ok {
		return string(store.SubmissionKindRUN), m, true
	}
	limit := int64(q.P)
	if q.runWaiting(ctx) {
		limit = int64(max(1, q.P-1))
	}
	if q.runningSb.Load() >= limit {
		select {
		case <-ctx.Done():
		case <-time.After(q.Poll):
		}
		return "", goredis.XMessage{}, false
	}
	if m, ok := q.read(ctx, q.Prefix+StreamSubmit, q.Poll); ok {
		return string(store.SubmissionKindSUBMIT), m, true
	}
	return "", goredis.XMessage{}, false
}

func (q *Queue) read(ctx context.Context, stream string, block time.Duration) (goredis.XMessage, bool) {
	if block <= 0 {
		block = -1 // go-redis: Block 0 = chờ vô hạn; -1 = không chờ
	}
	rs, err := q.Redis.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: Group, Consumer: q.Name, Streams: []string{stream, ">"}, Count: 1, Block: block}).Result()
	if err != nil || len(rs) == 0 || len(rs[0].Messages) == 0 {
		if err != nil && !errors.Is(err, goredis.Nil) && ctx.Err() == nil {
			q.Log.WarnContext(ctx, "đọc Stream lỗi", "stream", stream, "error", err.Error())
			time.Sleep(q.Poll)
		}
		return goredis.XMessage{}, false
	}
	return rs[0].Messages[0], true
}

// runWaiting: còn tin `Chạy thử` chưa giao (lag) hoặc đang treo trong PEL.
func (q *Queue) runWaiting(ctx context.Context) bool {
	gs, err := q.Redis.XInfoGroups(ctx, q.Prefix+StreamRun).Result()
	if err != nil {
		return false
	}
	for _, g := range gs {
		if g.Name == Group {
			return g.Lag > 0
		}
	}
	return false
}

// drainPending xử lý tin còn treo của CHÍNH consumer này (đã nhận trước khi khởi động lại): bước nhận việc ở DB quyết định có chấm hay không.
func (q *Queue) drainPending(ctx context.Context) {
	for _, s := range []string{q.Prefix + StreamRun, q.Prefix + StreamSubmit} {
		kind := string(store.SubmissionKindSUBMIT)
		if s == q.Prefix+StreamRun {
			kind = string(store.SubmissionKindRUN)
		}
		for ctx.Err() == nil {
			rs, err := q.Redis.XReadGroup(ctx, &goredis.XReadGroupArgs{Group: Group, Consumer: q.Name, Streams: []string{s, "0"}, Count: 10, Block: -1}).Result()
			if err != nil || len(rs) == 0 || len(rs[0].Messages) == 0 {
				break
			}
			for _, m := range rs[0].Messages {
				q.process(ctx, kind, m)
			}
		}
	}
}

func (q *Queue) ack(ctx context.Context, kind string, e goredis.XMessage) {
	stream, _ := q.streamFor(kind)
	if err := q.Redis.XAck(context.WithoutCancel(ctx), stream, Group, e.ID).Err(); err != nil {
		q.Log.WarnContext(ctx, "XACK lỗi", "stream", stream, "error", err.Error())
	}
}

// process chấm một tin (mọi bước idempotent; mọi UPDATE chỉ ảnh hưởng đúng dòng nếu điều kiện còn đúng).
func (q *Queue) process(ctx context.Context, kind string, e goredis.XMessage) {
	sid, _ := e.Values["submission_id"].(string)
	id, err := uuid.Parse(sid)
	if err != nil {
		q.ack(ctx, kind, e)
		return
	}
	defer q.ack(ctx, kind, e)
	s := store.New(q.Pool)
	// 3. Thay bản cũ: SUBMIT đang QUEUED mà đã có SUBMIT mới hơn của (lượt, mục) → SUPERSEDED, không chạy sandbox.
	if _, err := s.JudgeSupersedeQueued(ctx, id); err == nil {
		q.Log.InfoContext(ctx, "bản nộp bị thay bằng bản mới hơn", "submission_id", id)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		q.Log.ErrorContext(ctx, "kiểm bản thay lỗi", "submission_id", id, "error", err.Error())
		return
	}

	// 4. Nhận việc (thuê). 0 dòng → bỏ qua. Mọi so sánh thời gian dùng `now()` của DB (không lệch đồng hồ giữa worker và Postgres).
	row, err := s.JudgeClaim(ctx, store.JudgeClaimParams{ID: id, LeaseMs: int32(q.Lease.Milliseconds())})
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		q.Log.ErrorContext(ctx, "nhận việc lỗi", "submission_id", id, "error", err.Error())
		return
	}
	lease := *row.LeaseUntil

	// 5. Gia hạn thuê trong lúc chấm; mất thuê → dừng chấm, bỏ kết quả.
	jctx, cancel := context.WithCancel(ctx)
	var curLease atomic.Value
	curLease.Store(lease)
	var lost atomic.Bool
	renewDone := make(chan struct{})
	go func() {
		defer close(renewDone)
		t := time.NewTicker(q.Renew)
		defer t.Stop()
		for {
			select {
			case <-jctx.Done():
				return
			case <-t.C:
				old := curLease.Load().(time.Time)
				next, err := s.JudgeRenew(jctx, store.JudgeRenewParams{ID: id, OldLease: old, LeaseMs: int32(q.Lease.Milliseconds())})
				if errors.Is(err, pgx.ErrNoRows) {
					lost.Store(true)
					cancel()
					return
				}
				if err == nil {
					curLease.Store(*next)
				}
			}
		}
	}()
	res, jerr := q.grade(jctx, s, row)
	cancel()
	<-renewDone
	if lost.Load() {
		q.Log.WarnContext(ctx, "mất thuê — bỏ kết quả", "submission_id", id)
		return
	}
	final := curLease.Load().(time.Time)
	q.finish(ctx, s, row, final, res, jerr, kind)
}

// grade đọc bài + test approved rồi chấm.
func (q *Queue) grade(ctx context.Context, s *store.Queries, row store.JudgeClaimRow) (Result, error) {
	p, err := s.JudgeProblem(ctx, store.JudgeProblemParams{CourseID: row.CourseID, QuestionID: row.ProblemID})
	if err != nil {
		return Result{}, fmt.Errorf("%w: đọc đề: %v", ErrSandbox, err)
	}
	rows, err := s.JudgeApprovedTests(ctx, store.JudgeApprovedTestsParams{CourseID: row.CourseID, ProblemID: row.ProblemID})
	if err != nil {
		return Result{}, fmt.Errorf("%w: đọc test: %v", ErrSandbox, err)
	}
	tests := make([]Test, 0, len(rows))
	for _, r := range rows {
		in, err := q.text(ctx, r.Input, r.InputBlobKey)
		if err != nil {
			return Result{}, err
		}
		exp, err := q.text(ctx, r.Expected, r.ExpectedBlobKey)
		if err != nil {
			return Result{}, err
		}
		tests = append(tests, Test{ID: r.ID.String(), Position: int(r.Position), IsSample: r.IsSample, Weight: int(r.Weight), Input: in, Expected: exp, Approved: true})
	}
	eps := 0.0
	if p.FloatEps.Valid {
		eps = p.FloatEps.Decimal.InexactFloat64()
	}
	prob := Problem{
		Limits:       Limits{TimeMS: int(p.TimeLimitMs), MemoryMB: int(p.MemoryLimitMb), OutputLimitKB: int(p.OutputLimitKb)},
		Checker:      CheckerKind(p.Checker),
		FloatEps:     eps,
		TestsVersion: int(p.TestsVersion),
	}
	return q.Judge.Judge(ctx, Language(row.Language), row.Source, prob, tests)
}

func (q *Queue) text(ctx context.Context, inline, key *string) (string, error) {
	if inline != nil {
		return *inline, nil
	}
	if key == nil || q.Blob == nil {
		return "", fmt.Errorf("%w: test lưu ở object storage nhưng worker chưa cấu hình blob", ErrSandbox)
	}
	rc, err := q.Blob.Get(ctx, *key)
	if err != nil {
		return "", fmt.Errorf("%w: đọc test: %v", ErrSandbox, err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, 2<<20))
	if err != nil {
		return "", fmt.Errorf("%w: đọc test: %v", ErrSandbox, err)
	}
	return string(b), nil
}

// finish ghi kết quả (rào chắn bằng thuê của mình) hoặc xử lý lỗi thật (6, 7).
func (q *Queue) finish(ctx context.Context, s *store.Queries, row store.JudgeClaimRow, lease time.Time, res Result, jerr error, kind string) {
	switch {
	case jerr != nil && IsSystemError(jerr):
		q.fail(ctx, s, row, lease, kind, jerr)
	case jerr != nil:
		q.Log.ErrorContext(ctx, "chấm lỗi không rõ", "submission_id", row.ID, "error", jerr.Error())
		q.fail(ctx, s, row, lease, kind, jerr)
	case res.ConfigProblem != "":
		msg := res.ConfigProblem
		if _, err := s.JudgeConfigError(ctx, store.JudgeConfigErrorParams{ID: row.ID, LeaseUntil: lease, CompileLog: &msg}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			q.Log.ErrorContext(ctx, "ghi lỗi cấu hình lỗi", "submission_id", row.ID, "error", err.Error())
			return
		}
		q.Log.WarnContext(ctx, "bài lỗi cấu hình test", "submission_id", row.ID, "exam_id", row.ExamID)
	default:
		q.done(ctx, row, lease, res)
	}
}

func (q *Queue) done(ctx context.Context, row store.JudgeClaimRow, lease time.Time, res Result) {
	results, _ := json.Marshal(res.Results)
	tv, pw, tw := int32(res.TestsVersion), int32(res.PassedWeight), int32(res.TotalWeight)
	tm, mk := int32(res.TimeMSMax), int32(res.MemoryKBMax)
	var log *string
	if res.CompileLog != "" {
		log = &res.CompileLog
	}
	tx, err := q.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		q.Log.ErrorContext(ctx, "mở transaction lỗi", "submission_id", row.ID, "error", err.Error())
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	ok := res.CompileOK
	fin, err := store.New(tx).JudgeFinish(ctx, store.JudgeFinishParams{
		ID: row.ID, LeaseUntil: lease, Verdict: store.JudgeVerdict(res.Verdict), TestsVersion: &tv, CompileOk: &ok, CompileLog: log, Results: results,
		PassedWeight: &pw, TotalWeight: &tw, TimeMsMax: &tm, MemoryKbMax: &mk,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		q.Log.WarnContext(ctx, "bản khác đã sở hữu bản nộp — bỏ kết quả", "submission_id", row.ID)
		return
	}
	if err != nil {
		q.Log.ErrorContext(ctx, "ghi kết quả lỗi", "submission_id", row.ID, "error", err.Error())
		return
	}
	if _, err := outbox.Write(ctx, tx, TopicDone, map[string]any{"submission_id": fin.ID, "course_id": fin.CourseID, "exam_id": fin.ExamID, "attempt_id": fin.AttemptID, "item_id": fin.ItemID, "kind": fin.Kind, "status": "DONE"}); err != nil {
		q.Log.ErrorContext(ctx, "ghi outbox lỗi", "submission_id", row.ID, "error", err.Error())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		q.Log.ErrorContext(ctx, "commit lỗi", "submission_id", row.ID, "error", err.Error())
		return
	}
	q.Log.InfoContext(ctx, "đã chấm", "submission_id", row.ID, "attempt_id", row.AttemptID, "exam_id", row.ExamID, "verdict", string(res.Verdict))
}

// fail: fail_count + 1; < 4 → QUEUED với next_attempt_at theo backoff; = 4 → ERROR + IE + dead-letter (không điểm 0).
func (q *Queue) fail(ctx context.Context, s *store.Queries, row store.JudgeClaimRow, lease time.Time, kind string, cause error) {
	n := int(row.FailCount) // fail_count TRƯỚC lần này
	back := q.Backoff[min(n, len(q.Backoff)-1)]
	r, err := s.JudgeFail(ctx, store.JudgeFailParams{ID: row.ID, LeaseUntil: lease, BackoffMs: int32(back.Milliseconds())})
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		q.Log.ErrorContext(ctx, "ghi lỗi chấm lỗi", "submission_id", row.ID, "error", err.Error())
		return
	}
	q.Log.WarnContext(ctx, "chấm lỗi hệ thống", "submission_id", row.ID, "fail_count", r.FailCount, "status", string(r.Status), "error", scrub(cause.Error(), ""))
	if r.Status == store.SubmissionStatusERROR {
		_, dead := q.streamFor(kind)
		_ = q.Redis.XAdd(ctx, &goredis.XAddArgs{Stream: dead, MaxLen: deadMaxLen, Approx: true, Values: map[string]any{"submission_id": row.ID.String()}}).Err()
		q.notifyError(ctx, r)
	}
}

// notifyError báo `exam.submission_done` (status ERROR) để US-PE-08 biết bài ở lại GRADING vì IE — KHÔNG chấm 0 điểm.
func (q *Queue) notifyError(ctx context.Context, r store.JudgeFailRow) {
	tx, err := q.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := outbox.Write(ctx, tx, TopicDone, map[string]any{"course_id": r.CourseID, "exam_id": r.ExamID, "attempt_id": r.AttemptID, "kind": r.Kind, "status": "ERROR"}); err == nil {
		_ = tx.Commit(ctx)
	}
}

// ---- tick (leader), thăm dò ---------------------------------------------------------------------------------------------

func (q *Queue) tickLoop(ctx context.Context) {
	t := time.NewTicker(q.TickEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if q.leader(ctx) {
				q.tick(ctx)
			}
		}
	}
}

// leader giữ khoá `ep:exam:tick:leader` (SET NX PX 15 s): một bộ lập lịch (SRS 4.2.6).
func (q *Queue) leader(ctx context.Context) bool {
	return q.Redis.Leader(ctx, LeaderKey(), q.Name, 15*time.Second)
}

// LeaderKey là khoá leader chung của mọi bộ lập lịch nền của thi hằng tuần (`ep:exam:tick:leader`).
func LeaderKey() string { return appredis.Key("exam", "tick", "leader") }

// Tick chạy MỘT vòng tick (xuất ra để test gọi trực tiếp, không chờ nhịp 5 s): đưa lại Stream các dòng đến hạn / hết thuê / mất tín hiệu.
func (q *Queue) Tick(ctx context.Context) { q.defaults(); q.tick(ctx) }

func (q *Queue) tick(ctx context.Context) {
	s := store.New(q.Pool)
	if rows, err := s.JudgeRequeueExpired(ctx); err == nil && len(rows) > 0 {
		q.Log.WarnContext(ctx, "thuê hết hạn — đưa bản nộp về hàng đợi", "count", len(rows))
	} else if err != nil {
		q.Log.ErrorContext(ctx, "đưa lại thuê hết hạn lỗi", "error", err.Error())
	}
	if _, err := s.JudgeResetStaleSignal(ctx, int32(q.Idle.Milliseconds())); err != nil {
		q.Log.ErrorContext(ctx, "đặt lại tín hiệu cũ lỗi", "error", err.Error())
	}
	due, err := s.JudgeDueUnqueued(ctx, 200)
	if err != nil {
		q.Log.ErrorContext(ctx, "đọc hàng đến hạn lỗi", "error", err.Error())
		return
	}
	for _, d := range due {
		if err := q.signal(ctx, d.ID, string(d.Kind)); err != nil {
			q.Log.WarnContext(ctx, "đưa lại tín hiệu lỗi", "submission_id", d.ID, "error", err.Error())
		}
	}
}

func (q *Queue) probeLoop(ctx context.Context) {
	t := time.NewTicker(q.Probe)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			q.probeOnce(ctx)
		}
	}
}

// probeOnce thăm dò `GET /version`: thành công → đặt `ep:judge:up` (TTL 30 s); lỗi → xoá (gateway thấy 503 trong ≤ 10 s). Down > 60 s: log warn MỘT lần.
// ProbeNow chạy MỘT lần thăm dò (test gọi trực tiếp với đồng hồ giả, không chờ nhịp 10 s).
func (q *Queue) ProbeNow(ctx context.Context) { q.defaults(); q.probeOnce(ctx) }

func (q *Queue) probeOnce(ctx context.Context) {
	key := appredis.Key("judge", upKeySuffix)
	if err := q.Judge.Version(ctx, 3*time.Second); err != nil {
		if q.up.Swap(false) || q.downSince.Load() == 0 {
			q.downSince.Store(q.Clock.Now().Unix())
		}
		_ = q.Redis.Del(ctx, key).Err()
		if since := q.downSince.Load(); since > 0 && q.Clock.Now().Unix()-since > 60 && q.warned.CompareAndSwap(false, true) {
			q.Log.WarnContext(ctx, "máy chấm không phản hồi quá 60 giây", "error", err.Error())
		}
		return
	}
	q.up.Store(true)
	q.downSince.Store(0)
	q.warned.Store(false)
	_ = q.Redis.Set(ctx, key, "1", 30*time.Second).Err()
}

// ParseMemLimit đọc JUDGE_MEM_LIMIT (`2g`, `3584m`, `2147483648`) ra MiB.
func ParseMemLimit(s string) (int, error) {
	if s == "" {
		return 0, errors.New("JUDGE_MEM_LIMIT trống")
	}
	var mult float64
	num := s
	switch last := s[len(s)-1]; last {
	case 'g', 'G':
		mult, num = 1024, s[:len(s)-1]
	case 'm', 'M':
		mult, num = 1, s[:len(s)-1]
	case 'k', 'K':
		mult, num = 1.0/1024, s[:len(s)-1]
	case 'b', 'B':
		mult, num = 1.0/(1<<20), s[:len(s)-1]
	default:
		mult = 1.0 / (1 << 20) // byte thuần
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil || f <= 0 {
		return 0, fmt.Errorf("JUDGE_MEM_LIMIT %q không hợp lệ", s)
	}
	return int(f * mult), nil
}

// CheckMemLimit kiểm công thức `JUDGE_MEM_LIMIT ≥ P × (512 + 256) MiB + 512 MiB` (SRS 4.5.8). Sai → lỗi nêu tên biến để worker thoát 1.
func CheckMemLimit(parallelism int, memLimit string) error {
	if parallelism < 1 {
		return errors.New("JUDGE_PARALLELISM phải ≥ 1")
	}
	mib, err := ParseMemLimit(memLimit)
	if err != nil {
		return err
	}
	need := parallelism*(512+256) + 512
	if mib < need {
		return fmt.Errorf("JUDGE_MEM_LIMIT=%s (%d MiB) nhỏ hơn mức cần %d MiB cho JUDGE_PARALLELISM=%d (công thức P×(512+256)+512 MiB); hãy tăng JUDGE_MEM_LIMIT hoặc giảm JUDGE_PARALLELISM", memLimit, mib, need, parallelism)
	}
	return nil
}
