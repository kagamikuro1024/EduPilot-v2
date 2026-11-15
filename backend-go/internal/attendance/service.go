package attendance

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

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Config là cấu hình điểm danh (SRS 4.1), đọc ở internal/platform/config.
type Config = config.Attendance

// TopicAttendanceSaved trùng crm.TopicAttendanceSaved (attendance không import crm: crm gọi attendance).
const TopicAttendanceSaved = "crm.attendance_saved"

// Service là nghiệp vụ điểm danh tự phục vụ. Redis nil ⇒ không giới hạn lượt (như middleware hạn mức); Attest nil ⇒ đăng ký thiết bị trả 503.
type Service struct {
	Pool   *pgxpool.Pool
	Redis  *appredis.Client
	Clock  clock.Clock
	Log    *slog.Logger
	Cfg    Config
	Attest Registrar // L5: kiểm Key Attestation; nil thì POST /me/devices trả ATTESTATION_UNAVAILABLE
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func unavailable() *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable).WithRetryAfter(5)
}

func invalid(field, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: field, Code: code, Message: msg})
}

func conflict(code string) *apierr.Error { return apierr.New(http.StatusConflict, code) }

// dbErr: lỗi Postgres là lỗi code (500); lỗi khác (mất kết nối, pool đóng) là DB không dùng được ⇒ 503, không ghi nửa chừng.
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

func wrap(op string, err error) error { return fmt.Errorf("attendance: %s: %w", op, dbErr(err)) }

// audit ghi audit_log trong giao dịch của người gọi. before / after chỉ có số đếm và id: KHÔNG tên, toạ độ, IP, token. actor nil = việc nền.
func audit(ctx context.Context, q *store.Queries, entity string, courseID uuid.UUID, actor *uuid.UUID, entityID, action string, before, after any) error {
	enc := func(v any) json.RawMessage {
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v) // map / số / chuỗi: không lỗi
		return b
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: actor, Entity: entity, EntityID: entityID, Action: action, Before: enc(before), After: enc(after)}); err != nil {
		return wrap("audit_log", err)
	}
	return nil
}

// requireActive: lớp đã lưu trữ → 409 COURSE_ARCHIVED (mọi thao tác ghi của Staff).
func (s *Service) requireActive(ctx context.Context, q *store.Queries, courseID uuid.UUID) error {
	c, _, err := s.course(ctx, q, courseID)
	if err != nil {
		return err
	}
	if c.Status != store.CourseStatusACTIVE {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

// course đọc lớp (không khoá) và chính sách điểm danh; lớp không có → 404.
func (s *Service) course(ctx context.Context, q *store.Queries, courseID uuid.UUID) (store.Course, Policy, error) {
	c, err := q.GetCourse(ctx, courseID)
	if err != nil {
		if isNoRows(err) {
			return store.Course{}, Policy{}, notFound()
		}
		return store.Course{}, Policy{}, wrap("đọc lớp", err)
	}
	return c, PolicyOf(s.Cfg, c.Settings), nil
}
