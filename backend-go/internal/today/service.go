package today

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// DefaultTTL là TODAY_CACHE_TTL: lưới an toàn khi mất sự kiện xoá cache (SRS 4.7).
const DefaultTTL = 60 * time.Second

const (
	cacheOpTimeout = 250 * time.Millisecond
	warnEvery      = 30 * time.Second
)

// ictZone là Asia/Ho_Chi_Minh (không phụ thuộc tzdata của máy chạy).
func ictZone() *time.Location { return time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) }

// ErrUserGone: người dùng của JWT không còn trong DB.
var ErrUserGone = errors.New("today: người dùng không còn")

// Service dựng "Hôm nay" cho một người xem: Viewer (một truy vấn), Provider, xếp hạng, cache Redis.
type Service struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client // nil = không cache
	Clock clock.Clock
	Log   *slog.Logger
	Agg   *Aggregator
	TTL   time.Duration

	warned *atomic.Int64 // mốc log "Redis lỗi" gần nhất, chung cho mọi yêu cầu
}

// NewService dựng Service với bộ gộp mặc định: Provider sinh viên, staff và admin của P2. llm có thể nil.
func NewService(pool *pgxpool.Pool, rdb *appredis.Client, clk clock.Clock, log *slog.Logger, llm LLMSignals) *Service {
	agg := &Aggregator{Log: log}
	agg.Register(StudentProvider{})
	agg.Register(StaffProvider{Pool: pool})
	agg.Register(AdminProvider{Pool: pool, LLM: llm})
	return &Service{Pool: pool, Redis: rdb, Clock: clk, Log: log, Agg: agg, TTL: DefaultTTL, warned: new(atomic.Int64)}
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

func (s *Service) ttl() time.Duration {
	if s.TTL <= 0 {
		return DefaultTTL
	}
	return s.TTL
}

// CacheKey là `ep:today:{user_id}:{scope}`.
func CacheKey(userID uuid.UUID, scopeKey string) string {
	return appredis.Key("today", userID.String(), scopeKey)
}

func (s *Service) warn(ctx context.Context, op string, err error) {
	if s.Log == nil || s.warned == nil {
		return
	}
	now := time.Now().UnixMilli()
	last := s.warned.Load()
	if now-last < warnEvery.Milliseconds() || !s.warned.CompareAndSwap(last, now) {
		return
	}
	s.Log.ErrorContext(ctx, "today: Redis lỗi, tính trực tiếp (fail-open)", "op", op, "error", err.Error())
}

// Get trả thân JSON của "Hôm nay" cho (userID, role, scope): từ Redis nếu còn, nếu không thì tính và ghi lại.
// Dạng phản hồi chỉ phụ thuộc vai trong JWT, không phụ thuộc tham số nào của client.
func (s *Service) Get(ctx context.Context, userID uuid.UUID, role Role, scope Scope) ([]byte, error) {
	key := CacheKey(userID, scope.Key())
	if s.Redis != nil {
		cctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
		b, err := s.Redis.Get(cctx, key).Bytes()
		cancel()
		switch {
		case err == nil:
			return b, nil
		case !errors.Is(err, appredisNil()):
			s.warn(ctx, "get", err)
		}
	}
	body, err := s.compute(ctx, userID, role, scope)
	if err != nil {
		return nil, err
	}
	if s.Redis != nil {
		cctx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
		if err := s.Redis.Set(cctx, key, body, s.ttl()).Err(); err != nil {
			s.warn(ctx, "set", err)
		}
		cancel()
	}
	return body, nil
}

func (s *Service) compute(ctx context.Context, userID uuid.UUID, role Role, scope Scope) ([]byte, error) {
	v, err := s.viewer(ctx, userID, role)
	if err != nil {
		return nil, err
	}
	items, total, err := s.Agg.Collect(ctx, v, scope)
	if err != nil {
		return nil, err
	}
	switch role {
	case RoleStudent:
		return marshal(s.student(ctx, v, scope, items))
	case RoleAdmin:
		return marshal(AdminView{Count: total, Actions: nonNil(items)}, nil)
	case RoleTeacher, RoleTA:
		return marshal(s.staff(ctx, v, scope, items, total))
	}
	return nil, fmt.Errorf("today: vai %q không có dạng Hôm nay", role)
}

func marshal(v any, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func nonNil(items []Item) []Item {
	if items == nil {
		return []Item{}
	}
	return items
}

// viewer dựng Viewer bằng MỘT truy vấn.
func (s *Service) viewer(ctx context.Context, userID uuid.UUID, role Role) (Viewer, error) {
	rows, err := store.New(s.Pool).TodayViewer(ctx, userID)
	if err != nil {
		return Viewer{}, fmt.Errorf("today: dựng người xem: %w", err)
	}
	if len(rows) == 0 {
		return Viewer{}, ErrUserGone
	}
	v := Viewer{UserID: userID, Role: role, EmailVerified: rows[0].Verified, MaskedEmail: MaskEmail(rows[0].Email), Now: s.now(), Courses: []CourseRef{}}
	for _, r := range rows {
		if r.CourseID == nil || r.ClassCode == nil || r.Status == nil || r.RoleInCourse == nil {
			continue
		}
		ref := CourseRef{ID: *r.CourseID, ClassCode: *r.ClassCode, RoleInCourse: string(*r.RoleInCourse)}
		switch *r.Status {
		case store.EnrollmentStatusACTIVE:
			v.Courses = append(v.Courses, ref)
		case store.EnrollmentStatusPENDING:
			if ref.RoleInCourse == "STUDENT" && r.StatusChangedAt != nil {
				v.pending = append(v.pending, pendingRef{Course: ref, Since: *r.StatusChangedAt})
			}
		}
	}
	return v, nil
}

// MaskEmail che email: "a***@x.com" (giữ ký tự đầu và tên miền).
func MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

// ----- dạng phản hồi -----

// SessionOut là một buổi học trong dòng thời gian của sinh viên.
type SessionOut struct {
	At     time.Time `json:"at"`
	EndsAt time.Time `json:"ends_at"`
	Title  string    `json:"title"`
	Place  string    `json:"place"`
	State  string    `json:"state"`
	Course CourseRef `json:"course"`
}

// StudentView là phản hồi cho Sinh viên.
type StudentView struct {
	NoCourse      bool         `json:"no_course"`
	EmailVerified bool         `json:"email_verified"`
	Recommended   *Item        `json:"recommended"`
	Timeline      []SessionOut `json:"timeline"`
	Continue      []any        `json:"continue"`
}

// UpcomingOut là một buổi học sắp tới của giảng viên / TA.
type UpcomingOut struct {
	At     time.Time `json:"at"`
	Title  string    `json:"title"`
	Place  string    `json:"place"`
	Course CourseRef `json:"course"`
}

// StaffView là phản hồi cho Giảng viên / TA.
type StaffView struct {
	Count     int           `json:"count"`
	Actions   []Item        `json:"actions"`
	Attention []any         `json:"attention"`
	Upcoming  []UpcomingOut `json:"upcoming"`
}

// AdminView là phản hồi cho Admin.
type AdminView struct {
	Count   int    `json:"count"`
	Actions []Item `json:"actions"`
}

func (s *Service) student(ctx context.Context, v Viewer, scope Scope, items []Item) (StudentView, error) {
	out := StudentView{NoCourse: len(v.Courses) == 0, EmailVerified: v.EmailVerified, Timeline: []SessionOut{}, Continue: []any{}}
	if len(items) > 0 {
		first := items[0]
		out.Recommended = &first
	}
	courses := v.Scoped(scope, "")
	if len(courses) == 0 {
		return out, nil
	}
	day := startOfDay(v.Now)
	rows, err := store.New(s.Pool).TodaySessions(ctx, store.TodaySessionsParams{CourseIds: ids(courses), FromAt: day, ToAt: day.AddDate(0, 0, 8)})
	if err != nil {
		return StudentView{}, fmt.Errorf("today: buổi học: %w", err)
	}
	for _, r := range rows {
		if len(out.Timeline) == 8 {
			break
		}
		state := "NEXT"
		switch {
		case !r.EndsAt.After(v.Now):
			state = "DONE"
		case !r.StartsAt.After(v.Now):
			state = "NOW"
		}
		out.Timeline = append(out.Timeline, SessionOut{At: r.StartsAt, EndsAt: r.EndsAt, State: state, Place: deref(r.Room),
			Title: fmt.Sprintf("Buổi %d · %s", r.SessionNo, r.CourseName), Course: CourseRef{ID: r.CourseID, ClassCode: r.ClassCode}})
	}
	return out, nil
}

func (s *Service) staff(ctx context.Context, v Viewer, scope Scope, items []Item, total int) (StaffView, error) {
	out := StaffView{Count: total, Actions: nonNil(items), Attention: []any{}, Upcoming: []UpcomingOut{}}
	courses := v.Scoped(scope, "STAFF")
	if len(courses) == 0 {
		return out, nil
	}
	rows, err := store.New(s.Pool).TodaySessions(ctx, store.TodaySessionsParams{CourseIds: ids(courses), FromAt: v.Now, ToAt: v.Now.AddDate(0, 0, 7)})
	if err != nil {
		return StaffView{}, fmt.Errorf("today: buổi học sắp tới: %w", err)
	}
	for _, r := range rows {
		if len(out.Upcoming) == 10 {
			break
		}
		out.Upcoming = append(out.Upcoming, UpcomingOut{At: r.StartsAt, Place: deref(r.Room), Title: fmt.Sprintf("Buổi %d · %s", r.SessionNo, r.CourseName), Course: CourseRef{ID: r.CourseID, ClassCode: r.ClassCode}})
	}
	return out, nil
}

func startOfDay(t time.Time) time.Time {
	l := t.In(ictZone())
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, ictZone())
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
