package course

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

// DismissSetup ghi `settings.setup.dismissed_at`: mục "Thiết lập lớp mới" ở "Hôm nay" ẩn đi. Idempotent (gọi lại không đổi mốc);
// guard Teacher đã kiểm quyền. Đổi trạng thái ⇒ `course.changed` để xoá cache "Hôm nay".
func (s Service) DismissSetup(ctx context.Context, actor, courseID uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("course: khoá lớp: %w", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return ErrArchived
	}
	n, err := q.DismissCourseSetup(ctx, store.DismissCourseSetupParams{ID: courseID, At: s.now().Format("2006-01-02T15:04:05Z07:00")})
	if err != nil {
		return fmt.Errorf("course: bỏ qua thiết lập: %w", err)
	}
	if n > 0 {
		if err := s.audit(ctx, q, courseID, actor, courseID.String(), "setup_dismissed", nil, nil); err != nil {
			return err
		}
		if err := emitChanged(ctx, tx, courseID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("course: commit: %w", err)
	}
	return nil
}
