// Package crm: điểm danh, sự kiện tham gia và (về sau) hồ sơ 360 của sinh viên (FEAT-crm-360). Dữ liệu điểm / dữ liệu cá nhân:
// mọi truy vấn có course_id, mọi ghi có audit_log cùng giao dịch, không cache phía máy chủ (luật 15), điểm cộng là decimal (luật 5).
package crm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/store"
)

// Service là nghiệp vụ điểm danh / tham gia. Không giữ trạng thái giữa các yêu cầu.
type Service struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
	Log   *slog.Logger
	// MaxRows = ATTENDANCE_MAX_ROWS; 0 = 1000.
	MaxRows int
	// DefaultSpeakPoints = PARTICIPATION_DEFAULT_POINTS; zero = 0,25.
	DefaultSpeakPoints decimal.Decimal
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

func (s *Service) maxRows() int {
	if s.MaxRows > 0 {
		return s.MaxRows
	}
	return 1000
}

func (s *Service) speakPoints() decimal.Decimal {
	if s.DefaultSpeakPoints.IsZero() {
		return decimal.RequireFromString("0.25")
	}
	return s.DefaultSpeakPoints
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func unavailable() *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable).WithRetryAfter(5)
}

// dbErr: lỗi từ Postgres là lỗi code (500, người gọi bọc `%w`); lỗi khác (mất kết nối, pool đóng, ctx) là DB không dùng được ⇒ 503, không ghi nửa chừng.
func dbErr(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return err
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	return unavailable()
}

func wrap(op string, err error) error { return fmt.Errorf("crm: %s: %w", op, dbErr(err)) }

// audit ghi audit_log của điểm danh trong giao dịch của người gọi. before / after chỉ có số đếm và {student_id, from, to}: KHÔNG tên, KHÔNG ghi chú.
func audit(ctx context.Context, q *store.Queries, entity string, courseID, actor uuid.UUID, entityID, action string, before, after any) error {
	enc := func(v any) json.RawMessage {
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v) // map / số / chuỗi: không lỗi
		return b
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: entity, EntityID: entityID, Action: action, Before: enc(before), After: enc(after)}); err != nil {
		return wrap("audit_log", err)
	}
	return nil
}
