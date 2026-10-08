package exam

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

const leaderTTL = 15 * time.Second

// Ticker là bộ lập lịch `exam.tick` (SRS 4.2.6): mỗi nhịp chỉ MỘT bản (khoá leader Redis) mở bài đến giờ rồi đóng bài đến hạn.
// Mọi đường đọc / ghi dùng EffectiveStatus nên bộ lập lịch chậm hay tắt không cho ai làm bài ngoài giờ; Ticker chỉ ghi lại trạng thái và phát sự kiện.
type Ticker struct {
	Svc   *Service
	Redis *appredis.Client // nil = luôn là leader (test một tiến trình)
	Log   *slog.Logger
	// Name: định danh của tiến trình; các bộ lập lịch cùng tiến trình dùng chung để nhận ra khoá của mình.
	Name  string
	Every time.Duration // EXAM_TICK_INTERVAL (5 s)
}

// Run chạy tới khi ctx huỷ.
func (t *Ticker) Run(ctx context.Context) error {
	every := t.Every
	if every <= 0 {
		every = 5 * time.Second
	}
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tk.C:
			if t.Redis != nil && !t.Redis.Leader(ctx, judge.LeaderKey(), t.Name, leaderTTL) {
				continue
			}
			if err := t.Tick(ctx); err != nil && t.Log != nil { // một vòng lỗi không dừng vòng sau
				t.Log.ErrorContext(ctx, "exam.tick lỗi", "error", err.Error())
			}
		}
	}
}

// Tick chạy MỘT vòng: (1) mở bài đến giờ; (2) đóng bài đến hạn (kể cả SCHEDULED đã quá giờ đóng — chuyển thẳng CLOSED); (3) tự nộp lượt quá hạn (US-PE-05). Mỗi bước idempotent nhờ
// `UPDATE … WHERE status = <cũ>` và ghi outbox cùng transaction nên hai bản chạy cùng lúc cũng chỉ chuyển một lần, một sự kiện. Bước hoàn tất chấm / công bố: US-PE-06, 08.
func (t *Ticker) Tick(ctx context.Context) error {
	return errors.Join(
		t.step(ctx, TopicExamOpened, func(q *store.Queries, now time.Time) ([]store.ExamTickOpenRow, error) {
			return q.ExamTickOpen(ctx, now)
		}),
		t.step(ctx, TopicExamClosed, func(q *store.Queries, now time.Time) ([]store.ExamTickOpenRow, error) {
			rows, err := q.ExamTickClose(ctx, now)
			out := make([]store.ExamTickOpenRow, len(rows))
			for i, r := range rows {
				out[i] = store.ExamTickOpenRow(r)
			}
			return out, err
		}),
		func() error { _, err := t.Svc.AutoSubmitDue(ctx); return err }(),
	)
}

func (t *Ticker) step(ctx context.Context, topic string, move func(*store.Queries, time.Time) ([]store.ExamTickOpenRow, error)) error {
	return t.Svc.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		rows, err := move(q, t.Svc.now())
		if err != nil {
			return fmt.Errorf("exam.tick %s: %w", topic, err)
		}
		for _, r := range rows {
			if _, err := outbox.Write(ctx, tx, topic, map[string]any{"exam_id": r.ID, "course_id": r.CourseID}); err != nil {
				return fmt.Errorf("exam.tick outbox %s: %w", topic, err)
			}
		}
		return nil
	})
}
