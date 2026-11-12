// Package practice: luyện đề theo chủ đề, thi thử có giờ, lịch sử / điểm yếu, tín hiệu tổng hợp (SRS FEAT-practice, sprint 10 / P9).
// Chấm là CODE THUẦN (`internal/quiz`); gói này chỉ gọi LLM ở các việc nền / giải thích (file riêng). Lượt luyện là dữ liệu học tập RIÊNG của sinh viên:
// không hàm nào ở đây trả câu trả lời của một sinh viên cho người khác; `Signals` chỉ trả tổng hợp k ≥ 3.
package practice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn của luyện đề (SRS 4.3, US-P9-05 AC7).
const (
	MinCount        = 5
	MaxCount        = 50
	DefaultTopicN   = 10
	DefaultMockN    = 20
	MinDuration     = 10
	MaxDuration     = 180
	DefaultDuration = 30
	MaxEssayChars   = 4000
	MaxShortChars   = 1000
	// KAnonymity: chủ đề có ít sinh viên sai hơn mức này bị loại khỏi tín hiệu tổng hợp (FR-20).
	KAnonymity = 3
)

// Service là nghiệp vụ luyện đề phía gateway.
type Service struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
	Log   *slog.Logger
	// Grace: dung sai ghi ngoại tuyến và độ trễ đóng lượt hết giờ (Q-P9-5; 0 = 60 s).
	Grace time.Duration
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

func (s *Service) grace() time.Duration {
	if s.Grace <= 0 {
		return 60 * time.Second
	}
	return s.Grace
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func fieldErr(field, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: field, Code: code, Message: msg})
}

// tx chạy fn trong một transaction; lỗi → rollback.
func (s *Service) tx(ctx context.Context, fn func(q *store.Queries, tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("practice: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(store.New(tx), tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("practice: commit: %w", err)
	}
	return nil
}

func isUnique(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

// writable chặn ghi vào lớp đã lưu trữ (409 COURSE_ARCHIVED).
func writable(ctx context.Context, q *store.Queries, courseID uuid.UUID) error {
	st, err := q.QuestionCourseStatus(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return fmt.Errorf("practice: đọc trạng thái lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

// owned đọc lượt và kiểm chủ lượt: không phải của người gọi → 404 (không lộ tồn tại).
func owned(ctx context.Context, q *store.Queries, userID, courseID, attemptID uuid.UUID, lock bool) (store.PracticeAttempt, error) {
	var a store.PracticeAttempt
	var err error
	if lock {
		a, err = q.PracticeAttemptLock(ctx, store.PracticeAttemptLockParams{CourseID: courseID, ID: attemptID})
	} else {
		a, err = q.PracticeAttemptGet(ctx, store.PracticeAttemptGetParams{CourseID: courseID, ID: attemptID})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return a, notFound()
	}
	if err != nil {
		return a, fmt.Errorf("practice: đọc lượt: %w", err)
	}
	if a.StudentID != userID {
		return a, notFound()
	}
	return a, nil
}

// ---- bản chụp câu hỏi --------------------------------------------------------------------------------------------------

// snapOption / snapshot là `practice_items.question` (JSON): bản chụp LÚC DỰNG ĐỀ — sửa câu giữa chừng không làm lệch lượt.
type snapOption struct {
	ID   uuid.UUID `json:"id"`
	Body string    `json:"body"`
}

type snapshot struct {
	Type        string          `json:"type"`
	Topic       string          `json:"topic"`
	Difficulty  string          `json:"difficulty"`
	Stem        string          `json:"stem"`
	Options     []snapOption    `json:"options"`
	Explanation *string         `json:"explanation"`
	Citations   json.RawMessage `json:"citations"`
}

func parseSnapshot(raw []byte) (snapshot, error) {
	var sn snapshot
	if err := json.Unmarshal(raw, &sn); err != nil {
		return sn, fmt.Errorf("practice: đọc bản chụp câu: %w", err)
	}
	return sn, nil
}

// TopicUpdated: sinh viên vừa bắt đầu / nộp một lượt → "Hôm nay" của chính họ làm mới (PRACTICE_WEAK_TOPIC biến mất ≤ 5 s). Payload {user_id, student_id, course_id}
// theo khoá của `today.Invalidator`; `course_id` không đưa vào để không làm mới cache của staff mỗi lần có người luyện (lệch SRS FR-19, ghi ở handoff).
const TopicUpdated = "practice.updated"

func emitUpdated(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	if _, err := outbox.Write(ctx, tx, TopicUpdated, map[string]string{"user_id": userID.String(), "student_id": userID.String()}); err != nil {
		return fmt.Errorf("practice: outbox %s: %w", TopicUpdated, err)
	}
	return nil
}
