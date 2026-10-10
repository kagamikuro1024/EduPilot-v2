// Package chat là chat riêng của sinh viên (SRS FEAT-private-chat-pii 4.7): phiên, gửi tin, SSE có thể nối lại, Dừng / Thử lại, reaper.
// Danh tính chỉ lấy từ JWT (Actor); mọi truy vấn lọc theo chủ phiên nên không có đường đọc chat của người khác.
package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

// Config là các hằng của SRS 8.1 (có giá trị mặc định; 0 = mặc định).
type Config struct {
	MaxInput     int           // CHAT_MAX_INPUT_CHARS
	RatePerMin   int           // CHAT_RATE_PER_MIN
	StreamMax    time.Duration // CHAT_STREAM_MAX_SECONDS
	FlushEvery   time.Duration // nhịp ghi partial_content
	ReapAfter    time.Duration // quá hạn này không cập nhật → INTERRUPTED
	History      int           // số tin gần nhất đưa vào ngữ cảnh
	ExtractHits  int           // CHAT_EXTRACT_HITS
	ExtractChars int           // CHAT_EXTRACT_CHARS
	BufTTL       time.Duration // hạn bộ đệm Redis Stream của một lượt
}

func (c Config) withDefaults() Config {
	def := func(v *int, d int) {
		if *v <= 0 {
			*v = d
		}
	}
	dur := func(v *time.Duration, d time.Duration) {
		if *v <= 0 {
			*v = d
		}
	}
	def(&c.MaxInput, 4000)
	def(&c.RatePerMin, 20)
	def(&c.History, 12)
	def(&c.ExtractHits, 3)
	def(&c.ExtractChars, 400)
	dur(&c.StreamMax, 120*time.Second)
	dur(&c.FlushEvery, time.Second)
	dur(&c.ReapAfter, 150*time.Second)
	dur(&c.BufTTL, 10*time.Minute)
	return c
}

// Responder là agent.Agent.
type Responder interface {
	Respond(ctx context.Context, tc agent.TrustedContext, in agent.Input) (agent.Outcome, error)
}

// Locker là exam.Locker: khoá chat trong giờ thi và dấu vết CHAT_BLOCKED.
type Locker interface {
	IsLocked(ctx context.Context, userID uuid.UUID) (exam.Lock, bool, error)
	RecordChatBlocked(ctx context.Context, userID, attemptID uuid.UUID) error
}

// Service là nghiệp vụ chat riêng.
type Service struct {
	Pool    *pgxpool.Pool
	Redis   *appredis.Client
	Agent   Responder
	Lock    Locker
	Members auth.CourseResolver
	// PII (tuỳ chọn) đếm số lần che theo loại ở tin của người dùng để ghi pii_events; nil = không ghi.
	PII   *privacy.Detector
	Clock clock.Clock
	Log   *slog.Logger
	Cfg   Config
	// Drain đóng khi gateway bắt đầu tắt: mọi lượt sinh còn sống ghi FAILED INTERRUPTED và nhả CHAT_BUSY ngay.
	Drain <-chan struct{}

	wg   sync.WaitGroup
	once sync.Once
	cfg  Config
}

func (s *Service) c() Config {
	s.once.Do(func() { s.cfg = s.Cfg.withDefaults() })
	return s.cfg
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

// Wait chờ mọi lượt sinh đang chạy kết thúc (test; tắt gateway).
func (s *Service) Wait() { s.wg.Wait() }

// Actor là người gọi: chỉ từ Principal.
type Actor struct {
	UserID  uuid.UUID
	Role    auth.Role
	TraceID string
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func forbidden() *apierr.Error {
	return apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]any{"reason": "course"})
}

func unavailable() *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.ChatUnavailable)
}

// requireStudent: chỉ sinh viên ACTIVE của lớp (TA / TEACHER / ADMIN, PENDING, REMOVED, ngoài lớp → 403).
func (s *Service) requireStudent(ctx context.Context, a Actor, courseID uuid.UUID) error {
	if a.Role != auth.RoleStudent {
		return forbidden()
	}
	ms, err := s.Members.Resolve(ctx, auth.Principal{Sub: a.UserID.String(), Role: a.Role}, courseID)
	if err != nil {
		return unavailable()
	}
	if !ms.Found || ms.Status != "ACTIVE" || ms.Role != auth.RoleStudent {
		return forbidden()
	}
	return nil
}

// session nạp phiên CỦA CHÍNH MÌNH (chưa xoá) rồi kiểm quyền lớp. Vai sai → 403 trước (không lộ phiên); phiên của người khác / không có → 404.
func (s *Service) session(ctx context.Context, a Actor, sid uuid.UUID) (store.ChatSession, error) {
	if a.Role != auth.RoleStudent {
		return store.ChatSession{}, forbidden()
	}
	row, err := store.New(s.Pool).GetChatSession(ctx, store.GetChatSessionParams{ID: sid, UserID: a.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ChatSession{}, notFound()
	}
	if err != nil {
		return store.ChatSession{}, fmt.Errorf("chat: nạp phiên: %w", err)
	}
	if err := s.requireStudent(ctx, a, row.CourseID); err != nil {
		return store.ChatSession{}, err
	}
	return row, nil
}

func (s *Service) archived(ctx context.Context, courseID uuid.UUID) error {
	st, err := store.New(s.Pool).ChatCourseStatus(ctx, courseID)
	if err != nil {
		return fmt.Errorf("chat: trạng thái lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}
