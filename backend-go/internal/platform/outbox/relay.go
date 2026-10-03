package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appotel "github.com/edupilot/backend-go/internal/platform/otel"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// GroupName là nhóm consumer của Stream outbox.dispatch (SRS 3.2).
const GroupName = "outbox"

// Streams đặt tên Stream / nhóm / consumer. Trường rỗng = tên sản phẩm (SRS 5.6);
// test đặt tên riêng để chạy song song trên cùng một Redis.
type Streams struct {
	Dispatch string
	Dead     string
	Group    string
	Consumer string
}

func (s Streams) withDefaults(instance string) Streams {
	if s.Dispatch == "" {
		s.Dispatch = appredis.StreamOutboxDispatch
	}
	if s.Dead == "" {
		s.Dead = appredis.StreamOutboxDead
	}
	if s.Group == "" {
		s.Group = GroupName
	}
	if s.Consumer == "" {
		s.Consumer = instance
	}
	return s
}

// Deps là phụ thuộc chung của relay và consumer trong worker.
type Deps struct {
	Pool    *pgxpool.Pool
	Redis   *appredis.Client
	Log     *slog.Logger
	Clock   clock.Clock
	Cfg     config.Config
	Streams Streams
}

// Relay quét outbox mỗi OUTBOX_POLL_INTERVAL: lấy dòng tới hạn bằng FOR UPDATE SKIP LOCKED,
// XADD vào Stream rồi đánh dấu enqueued_at trong CÙNG transaction — hai relay không bao giờ
// xếp hàng trùng một dòng. Dòng "đã xếp hàng" quá OUTBOX_STALE_AFTER được đặt lại.
type Relay struct {
	pool  *pgxpool.Pool
	rdb   *appredis.Client
	log   *slog.Logger
	cfg   config.Config
	names Streams
}

// NewRelay dựng relay từ phụ thuộc của worker.
func NewRelay(d Deps) *Relay {
	return &Relay{pool: d.Pool, rdb: d.Redis, log: d.Log, cfg: d.Cfg, names: d.Streams.withDefaults(d.Cfg.InstanceID)}
}

// Name là tên việc nền (log của worker).
func (*Relay) Name() string { return "outbox.relay" }

// Run chạy vòng quét tới khi ctx huỷ.
func (r *Relay) Run(ctx context.Context) error {
	tick := time.NewTicker(r.cfg.OutboxPollInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if err := r.poll(ctx); err != nil && ctx.Err() == nil {
				r.log.ErrorContext(ctx, "relay outbox lỗi", "error", err.Error())
			}
		}
	}
}

func (r *Relay) poll(ctx context.Context) error {
	if err := r.requeueStale(ctx); err != nil {
		return err
	}
	return r.enqueueDue(ctx)
}

// requeueStale đặt lại enqueued_at = NULL cho dòng kẹt quá OUTBOX_STALE_AFTER (tin bị mất hẳn khỏi Stream).
func (r *Relay) requeueStale(ctx context.Context) error {
	ids, err := store.New(r.pool).RequeueStaleOutbox(ctx, r.cfg.OutboxStaleAfter.Milliseconds())
	if err != nil {
		return fmt.Errorf("requeue stale outbox: %w", err)
	}
	if len(ids) > 0 {
		r.log.WarnContext(ctx, "đặt lại dòng outbox kẹt", "count", len(ids))
	}
	return nil
}

func (r *Relay) enqueueDue(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin relay tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := store.New(tx)
	rows, err := q.SelectDueOutbox(ctx, int32(r.cfg.OutboxBatch))
	if err != nil {
		return fmt.Errorf("select due outbox: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	// Mỗi lượt xếp hàng là một span GỐC: dòng log có trace_id riêng.
	ctx, span := appotel.StartRoot(ctx, "outbox.relay")
	defer span.End()
	for _, row := range rows {
		add := r.rdb.XAdd(ctx, &goredis.XAddArgs{
			Stream: r.names.Dispatch,
			Values: []any{"outbox_id", row.ID.String(), "topic", row.Topic},
		})
		if err := add.Err(); err != nil {
			return fmt.Errorf("xadd %s: %w", r.names.Dispatch, err)
		}
		if _, err := q.MarkOutboxEnqueued(ctx, row.ID); err != nil {
			return fmt.Errorf("mark enqueued %s: %w", row.ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit relay tx: %w", err)
	}
	r.log.DebugContext(ctx, "outbox đã xếp hàng", "count", len(rows))
	return nil
}
