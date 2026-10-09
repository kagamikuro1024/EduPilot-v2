// Package exam: ngân hàng câu hỏi, bài code và test, duyệt, khoá sửa (US-PE-03); bài thi, lên lịch, bộ lập lịch (US-PE-04); lượt làm… ở các story sau.
// Lỗi nghiệp vụ trả `*apierr.Error` (handler ghi nguyên văn); sai `version` trả `*VersionConflict`.
package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Blob là phần của `platform/blob` mà gói này dùng (test lớn > 64 KiB nằm ở kho đối tượng).
type Blob interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// Service là nghiệp vụ của PE phía gateway.
type Service struct {
	Pool  *pgxpool.Pool
	Blob  Blob // nil ⇒ test > 64 KiB bị từ chối (503), phần còn lại chạy bình thường
	Clock clock.Clock
	Jobs  *jobs.Service
	// ZipMaxUncompressed: EXAM_TESTZIP_MAX_UNCOMPRESSED (0 = mặc định 50 MiB).
	ZipMaxUncompressed int
	Limits             Limits           // giới hạn bài thi (US-PE-04)
	Attempt            AttemptConfig    // lượt làm (US-PE-05)
	Code               CodeConfig       // bài code trong lượt làm (US-PE-06)
	Integrity          IntegrityConfig  // liêm chính (US-PE-07)
	Results            ResultsConfig    // kết quả / công bố (US-PE-08)
	Log                *slog.Logger     // nil = slog.Default()
	Redis              *appredis.Client // giới hạn lưu theo lượt (nil = không giới hạn)
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

// VersionConflict: `version` gửi lên không khớp (409 VERSION_CONFLICT kèm bản hiện tại — chỉ trường người gọi được đọc).
type VersionConflict struct {
	Version int
	Current any
}

func (e *VersionConflict) Error() string { return "exam: sai version" }

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func conflict(msg string) *apierr.Error {
	return apierr.New(http.StatusConflict, apierr.Conflict).WithMessage(msg)
}

func fieldErr(field, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: field, Code: code, Message: msg})
}

// tx chạy fn trong một transaction; lỗi (kể cả *apierr.Error) → rollback.
func (s *Service) tx(ctx context.Context, fn func(q *store.Queries, tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("exam: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(store.New(tx), tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("exam: commit: %w", err)
	}
	return nil
}

// writable chặn ghi vào lớp đã lưu trữ (409 COURSE_ARCHIVED; đọc vẫn được).
func writable(ctx context.Context, q *store.Queries, courseID uuid.UUID) error {
	st, err := q.QuestionCourseStatus(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return fmt.Errorf("exam: đọc trạng thái lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

// audit ghi MỘT dòng audit_log cùng transaction; before / after chỉ gồm trường nhỏ (trạng thái, version, số lượng) — không nội dung câu hỏi.
func audit(ctx context.Context, q *store.Queries, courseID, actor uuid.UUID, entity string, entityID uuid.UUID, action string, before, after map[string]any) error {
	arg := store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: entity, EntityID: entityID.String(), Action: action}
	if before != nil {
		arg.Before, _ = json.Marshal(before)
	}
	if after != nil {
		arg.After, _ = json.Marshal(after)
	}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		t := sc.TraceID().String()
		arg.TraceID = &t
	}
	if _, err := q.InsertAuditLog(ctx, arg); err != nil {
		return fmt.Errorf("exam: ghi audit_log: %w", err)
	}
	return nil
}
