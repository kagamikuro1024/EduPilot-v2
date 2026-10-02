package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appotel "github.com/edupilot/backend-go/internal/platform/otel"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

const (
	// maxAttempts: 1 lần đầu + 3 lần thử lại rồi dead-letter (SRS 3.2, FR-24).
	maxAttempts = 4
	// maxErrLen là giới hạn của cột last_error (SRS 5.3).
	maxErrLen = 1000
	// deadMaxLen giữ Stream dead-letter ở ~10000 tin (SRS 5.6).
	deadMaxLen = 10000
)

// Consumer đọc Stream outbox.dispatch bằng XREADGROUP (nhóm `outbox`), chạy handler theo topic
// và khử trùng theo outbox.id: dòng đã dispatched/dead thì chỉ XACK + XDEL. Lỗi, panic hay topic
// lạ đều là thất bại: attempts++ với backoff, tới lần thứ 4 thì dead-letter. Tin kẹt trong PEL quá
// OUTBOX_CLAIM_IDLE được consumer khác XAUTOCLAIM.
type Consumer struct {
	pool  *pgxpool.Pool
	rdb   *appredis.Client
	log   *slog.Logger
	clk   clock.Clock
	cfg   config.Config
	reg   *Registry
	names Streams
}

// NewConsumer dựng consumer với bảng handler reg.
func NewConsumer(d Deps, reg *Registry) *Consumer {
	return &Consumer{
		pool: d.Pool, rdb: d.Redis, log: d.Log, clk: d.Clock, cfg: d.Cfg, reg: reg,
		names: d.Streams.withDefaults(d.Cfg.InstanceID),
	}
}

// Name là tên việc nền (log của worker).
func (*Consumer) Name() string { return "outbox.consumer" }

// Run đọc Stream tới khi ctx huỷ. Lỗi Redis/DB chỉ ghi log rồi thử lại — việc nền không được chết.
func (c *Consumer) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		if err := c.cycle(ctx); err != nil && ctx.Err() == nil {
			c.log.ErrorContext(ctx, "consumer outbox lỗi", "error", err.Error())
			select {
			case <-ctx.Done():
			case <-time.After(c.cfg.OutboxPollInterval):
			}
		}
	}
	return nil
}

func (c *Consumer) cycle(ctx context.Context) error {
	// Gọi mỗi vòng: rẻ và tự lành nếu Redis mất nhóm (NOGROUP).
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	if err := c.claimStale(ctx); err != nil {
		return err
	}
	return c.read(ctx)
}

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.names.Dispatch, c.names.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("tạo nhóm %s: %w", c.names.Group, err)
	}
	return nil
}

// claimStale nhận lại tin consumer khác đã đọc nhưng chưa XACK quá OUTBOX_CLAIM_IDLE.
func (c *Consumer) claimStale(ctx context.Context) error {
	msgs, _, err := c.rdb.XAutoClaim(ctx, &goredis.XAutoClaimArgs{
		Stream:   c.names.Dispatch,
		Group:    c.names.Group,
		Consumer: c.names.Consumer,
		MinIdle:  c.cfg.OutboxClaimIdle,
		Start:    "0-0",
		Count:    int64(c.cfg.OutboxBatch),
	}).Result()
	if err != nil && !errors.Is(err, goredis.Nil) {
		return fmt.Errorf("xautoclaim %s: %w", c.names.Dispatch, err)
	}
	for _, m := range msgs {
		c.handle(ctx, m)
	}
	return nil
}

func (c *Consumer) read(ctx context.Context) error {
	streams, err := c.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    c.names.Group,
		Consumer: c.names.Consumer,
		Streams:  []string{c.names.Dispatch, ">"},
		Count:    int64(c.cfg.OutboxBatch),
		Block:    c.cfg.OutboxPollInterval,
	}).Result()
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("xreadgroup %s: %w", c.names.Dispatch, err)
	}
	for _, s := range streams {
		for _, m := range s.Messages {
			c.handle(ctx, m)
		}
	}
	return nil
}

// handle xử lý một tin. Mỗi tin là một span GỐC (SRS 3.2).
func (c *Consumer) handle(ctx context.Context, m goredis.XMessage) {
	ctx, span := appotel.StartRoot(ctx, "outbox.consume")
	defer span.End()

	raw, _ := m.Values["outbox_id"].(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		c.log.WarnContext(ctx, "tin outbox không hợp lệ", "entry_id", m.ID)
		c.done(ctx, m.ID)
		return
	}

	q := store.New(c.pool)
	row, err := q.GetOutbox(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.done(ctx, m.ID)
		return
	}
	if err != nil {
		// Không XACK: tin ở lại PEL để XAUTOCLAIM thử lại.
		c.log.ErrorContext(ctx, "nạp dòng outbox lỗi", "outbox_id", id.String(), "error", err.Error())
		return
	}
	if row.DispatchedAt != nil || row.DeadAt != nil { // khử trùng theo outbox.id
		c.done(ctx, m.ID)
		return
	}

	if runErr := c.runHandler(ctx, row); runErr != nil {
		c.fail(ctx, row, runErr)
		c.done(ctx, m.ID)
		return
	}
	if _, err := q.MarkOutboxDispatched(ctx, row.ID); err != nil {
		// At-least-once: không XACK, handler idempotent nên chạy lại được.
		c.log.ErrorContext(ctx, "đánh dấu dispatched lỗi", "outbox_id", row.ID.String(), "error", err.Error())
		return
	}
	c.done(ctx, m.ID)
}

// runHandler chạy handler của topic; panic và topic lạ cũng là lỗi.
func (c *Consumer) runHandler(ctx context.Context, row store.Outbox) (err error) {
	h, ok := c.reg.Lookup(row.Topic)
	if !ok {
		return fmt.Errorf("topic %q chưa đăng ký handler", row.Topic)
	}
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("handler panic: %v", p)
		}
	}()
	return h(ctx, Message{ID: row.ID, Topic: row.Topic, Payload: row.Payload, Attempts: int(row.Attempts)})
}

// fail ghi lỗi + backoff; tới lần thứ maxAttempts thì dead-letter.
func (c *Consumer) fail(ctx context.Context, row store.Outbox, cause error) {
	// last_error: chỉ thông điệp lỗi, KHÔNG kèm payload (tránh lộ PII — SRS 5.3).
	msg := truncate(cause.Error(), maxErrLen)
	q := store.New(c.pool)
	attempts, err := q.MarkOutboxFailed(ctx, store.MarkOutboxFailedParams{
		ID: row.ID, LastError: msg, BackoffMs: c.backoffMS(int(row.Attempts)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return // dòng đã xong ở nơi khác
	}
	if err != nil {
		c.log.ErrorContext(ctx, "đánh dấu outbox thất bại lỗi", "outbox_id", row.ID.String(), "error", err.Error())
		return
	}
	c.log.WarnContext(ctx, "xử lý outbox thất bại",
		"outbox_id", row.ID.String(), "topic", row.Topic, "attempts", attempts, "error", msg)
	if int(attempts) < maxAttempts {
		return
	}
	if _, err := q.MarkOutboxDead(ctx, store.MarkOutboxDeadParams{ID: row.ID, LastError: msg}); err != nil {
		c.log.ErrorContext(ctx, "đánh dấu outbox dead lỗi", "outbox_id", row.ID.String(), "error", err.Error())
		return
	}
	add := c.rdb.XAdd(ctx, &goredis.XAddArgs{
		Stream: c.names.Dead,
		MaxLen: deadMaxLen,
		Approx: true,
		Values: []any{
			"outbox_id", row.ID.String(),
			"topic", row.Topic,
			"error", msg,
			"at", c.clk.Now().Format(time.RFC3339),
		},
	})
	if err := add.Err(); err != nil {
		c.log.ErrorContext(ctx, "xadd dead-letter lỗi", "outbox_id", row.ID.String(), "error", err.Error())
		return
	}
	c.log.ErrorContext(ctx, "outbox dead-letter", "outbox_id", row.ID.String(), "topic", row.Topic, "attempts", attempts)
}

// backoffMS trả độ trễ của lần thử lại sau khi đã thất bại `done` lần.
func (c *Consumer) backoffMS(done int) int64 {
	b := c.cfg.OutboxRetryBackoff
	if len(b) == 0 {
		return 0
	}
	if done >= len(b) {
		done = len(b) - 1
	}
	return b[done].Milliseconds()
}

// done xác nhận tin rồi xoá khỏi Stream (SRS 3.2: XACK + XDEL).
func (c *Consumer) done(ctx context.Context, entryID string) {
	if err := c.rdb.XAck(ctx, c.names.Dispatch, c.names.Group, entryID).Err(); err != nil {
		c.log.WarnContext(ctx, "xack lỗi", "entry_id", entryID, "error", err.Error())
		return
	}
	if err := c.rdb.XDel(ctx, c.names.Dispatch, entryID).Err(); err != nil {
		c.log.WarnContext(ctx, "xdel lỗi", "entry_id", entryID, "error", err.Error())
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
