package course

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Ngưỡng chuyển cho giảng viên (SRS FEAT-escalation-notify 4.2.8): [0,50; 0,95], bước 0,05, so sánh bằng decimal.
var (
	thresholdMin  = decimal.New(50, -2) //nolint:gochecknoglobals // hằng số tra cứu bất biến
	thresholdMax  = decimal.New(95, -2) //nolint:gochecknoglobals
	thresholdStep = decimal.New(5, -2)  //nolint:gochecknoglobals
)

// ThresholdConflictError: sai version khi đặt ngưỡng; Current là bản hiện hành của lớp (để giao diện hoàn lại).
type ThresholdConflictError struct{ Current Detail }

func (e *ThresholdConflictError) Error() string { return "course: sai version cài đặt lớp" }

// PutEscalationThreshold đặt `courses.escalation_threshold` (chỉ giảng viên của lớp; guard ở route). Có hiệu lực ở tin kế tiếp của sinh viên (chat đọc mỗi lần).
// Giá trị ngoài khoảng / không đúng bước → 422; sai version → ThresholdConflictError; ghi audit_log (cũ / mới). `courses.version + 1`.
func (s Service) PutEscalationThreshold(ctx context.Context, actor, courseID uuid.UUID, version int, raw string) (Detail, error) {
	v, err := decimal.NewFromString(raw)
	if err != nil || v.LessThan(thresholdMin) || v.GreaterThan(thresholdMax) || !v.Mod(thresholdStep).IsZero() {
		return Detail{}, invalid("escalation_threshold", "range", "Ngưỡng từ 0,50 đến 0,95, mỗi bước 0,05.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if cur.Status == store.CourseStatusARCHIVED {
		return Detail{}, ErrArchived
	}
	if int(cur.Version) != version {
		now, gerr := s.Get(ctx, courseID, "TEACHER", true)
		if gerr != nil {
			return Detail{}, gerr
		}
		return Detail{}, &ThresholdConflictError{Current: now}
	}
	upd, err := q.UpdateCourseThreshold(ctx, store.UpdateCourseThresholdParams{ID: courseID, Threshold: v})
	if err != nil {
		return Detail{}, fmt.Errorf("course: đặt ngưỡng: %w", err)
	}
	if err := s.audit(ctx, q, courseID, actor, courseID.String(), "course_threshold_updated",
		map[string]any{"escalation_threshold": cur.EscalationThreshold.StringFixed(2), "version": cur.Version},
		map[string]any{"escalation_threshold": upd.EscalationThreshold.StringFixed(2), "version": upd.Version}); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("course: commit ngưỡng: %w", err)
	}
	return s.Get(ctx, courseID, "TEACHER", true)
}
