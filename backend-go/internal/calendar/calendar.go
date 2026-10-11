// Package calendar: lịch gộp lúc đọc (buổi học + bài thi PE + sự kiện của Staff), feed ICS, nhắc 24 giờ, nguồn cho tool lịch (SRS FEAT-docs-calendar 4.7–4.9).
// Không nhân bản: buổi học và bài thi KHÔNG có dòng nào trong calendar_events.
package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// TopicChanged: sự kiện của Staff đổi → vô hiệu cache "Hôm nay" của thành viên lớp (≤ 5 s).
const TopicChanged = "calendar.changed"

// MaxRange: khoảng `from..to` tối đa của GET calendar.
const MaxRange = 62 * 24 * time.Hour

var vietnam = time.FixedZone("ICT", 7*60*60) //nolint:gochecknoglobals // múi giờ cố định Asia/Ho_Chi_Minh (không DST)

// Service là nghiệp vụ lịch.
type Service struct {
	Pool      *pgxpool.Pool
	Redis     *appredis.Client
	Clock     clock.Clock
	PublicURL string // APP_PUBLIC_URL, dựng URL feed
	Log       *slog.Logger
}

func (s *Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now()
	}
	return time.Now()
}

// Actor là người gọi: danh tính từ JWT, vai TRONG LỚP từ CourseAccessGuard.
type Actor struct {
	UserID  uuid.UUID
	Student bool
	Staff   bool
	TraceID string
}

// Item là một dòng của lịch gộp.
type Item struct {
	ID            string     `json:"id"` // "<source>:<uuid>"
	Source        string     `json:"source"`
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	StartsAt      time.Time  `json:"starts_at"`
	EndsAt        *time.Time `json:"ends_at"`
	Location      *string    `json:"location"`
	Href          *string    `json:"href"`
	Editable      bool       `json:"editable"`
	PersonalState *string    `json:"personal_state"`
	Status        *string    `json:"status"` // bài thi: trạng thái HIỆU LỰC theo giờ (SCHEDULED / OPEN / CLOSED / PUBLISHED); nơi khác null
	Version       *int32     `json:"version,omitempty"`
}

// Page là một trang lịch.
type Page struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func ptr[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// List: hợp ba nguồn ở thời điểm truy vấn trong [from, to); sắp (starts_at, id); phân trang con trỏ.
func (s *Service) List(ctx context.Context, a Actor, course uuid.UUID, from, to time.Time, p httpx.PageParams) (Page, error) {
	if to.Sub(from) > MaxRange || !to.After(from) {
		return Page{}, apierr.New(http.StatusUnprocessableEntity, apierr.RangeTooLarge)
	}
	rows, err := s.rows(ctx, a, course, from, to, false, p.Fetch(), p.Cursor)
	if err != nil {
		return Page{}, err
	}
	out := Page{Items: make([]Item, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].StartsAt, rows[len(rows)-1].raw)
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, r.Item)
	}
	return out, nil
}

type row struct {
	Item
	raw string
}

func (s *Service) rows(ctx context.Context, a Actor, course uuid.UUID, from, to time.Time, onlyExam bool, limit int, cur *httpx.Cursor) ([]row, error) {
	arg := store.CalListParams{CourseID: course, ViewerID: a.UserID, IsStudent: a.Student, FromAt: from, ToAt: to, OnlyExam: onlyExam, PageLimit: int32(limit)} //nolint:gosec // ≤ 101
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.CreatedAt, &cur.ID
	}
	rs, err := store.New(s.Pool).CalList(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("calendar: danh sách: %w", err)
	}
	now := s.now()
	out := make([]row, 0, len(rs))
	for _, r := range rs {
		var ends *time.Time
		if r.HasEnd {
			ends = &r.EndsAt
		}
		it := Item{ID: r.Src + ":" + r.ID.String(), Source: r.Src, Type: r.Typ, Title: r.Title, StartsAt: r.StartsAt, EndsAt: ends, Location: r.Location, PersonalState: ptr(r.PersonalState)}
		switch r.Src {
		case "weekly_exam":
			st := exam.EffectiveStatus(r.ExamStatus, &r.StartsAt, ends, now)
			it.Status = ptr(st)
			href := "/exams/" + r.ID.String()
			if a.Student {
				href += "/take"
			}
			it.Href = &href
		case "calendar_event":
			it.Editable = a.Staff
			if a.Staff {
				it.Version = &r.Ver
			}
		}
		out = append(out, row{Item: it, raw: r.ID.String()})
	}
	return out, nil
}

// EventIn là thân tạo / sửa sự kiện.
type EventIn struct {
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	Location    *string    `json:"location"`
	Description *string    `json:"description"`
	Version     *int       `json:"version"`
}

// EventOut là một sự kiện của Staff.
type EventOut struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	Location    *string    `json:"location"`
	Description *string    `json:"description"`
	Version     int32      `json:"version"`
}

func eventOf(e store.CalendarEvent) EventOut {
	return EventOut{ID: e.ID, Type: string(e.Type), Title: e.Title, StartsAt: e.StartsAt, EndsAt: e.EndsAt, Location: e.Location, Description: e.Description, Version: e.Version}
}

func field(f, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: f, Code: code, Message: msg})
}

func (s *Service) validate(in *EventIn) error {
	in.Title = strings.TrimSpace(in.Title)
	switch {
	case in.Type != "EXAM" && in.Type != "OTHER":
		return field("type", "enum", "Loại sự kiện là EXAM hoặc OTHER.")
	case utf8.RuneCountInString(in.Title) < 1 || utf8.RuneCountInString(in.Title) > 120:
		return field("title", "length", "Tên sự kiện dài 1–120 ký tự.")
	case in.Location != nil && utf8.RuneCountInString(*in.Location) > 80:
		return field("location", "length", "Địa điểm tối đa 80 ký tự.")
	case in.Description != nil && utf8.RuneCountInString(*in.Description) > 1000:
		return field("description", "length", "Mô tả tối đa 1.000 ký tự.")
	case in.StartsAt.IsZero():
		return field("starts_at", "required", "Thiếu giờ bắt đầu.")
	}
	now := s.now()
	if in.StartsAt.Before(now.AddDate(-2, 0, 0)) || in.StartsAt.After(now.AddDate(2, 0, 0)) || (in.EndsAt != nil && !in.EndsAt.After(in.StartsAt)) {
		return apierr.New(http.StatusUnprocessableEntity, apierr.EventTimeInvalid)
	}
	return nil
}

func (s *Service) requireActive(ctx context.Context, q *store.Queries, course uuid.UUID) error {
	st, err := q.CalCourseStatus(ctx, course)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return fmt.Errorf("calendar: đọc lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

func (s *Service) write(ctx context.Context, a Actor, course uuid.UUID, action, entity string, before, after any, fn func(q *store.Queries) (uuid.UUID, error)) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("calendar: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := s.requireActive(ctx, q, course); err != nil {
		return err
	}
	id, err := fn(q)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(before)
	af, _ := json.Marshal(after)
	var trace *string
	if a.TraceID != "" {
		trace = &a.TraceID
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &a.UserID, Entity: entity, EntityID: id.String(), Action: action, Before: b, After: af, TraceID: trace}); err != nil {
		return fmt.Errorf("calendar: audit: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicChanged, map[string]string{"course_id": course.String(), "event_id": id.String()}); err != nil {
		return fmt.Errorf("calendar: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("calendar: commit: %w", err)
	}
	return nil
}

// Create tạo sự kiện (Staff); lớp lưu trữ → 409.
func (s *Service) Create(ctx context.Context, a Actor, course uuid.UUID, in EventIn) (EventOut, error) {
	if err := s.validate(&in); err != nil {
		return EventOut{}, err
	}
	var out EventOut
	err := s.write(ctx, a, course, "calendar.event.create", "calendar_event", nil, in, func(q *store.Queries) (uuid.UUID, error) {
		e, err := q.CalEventInsert(ctx, store.CalEventInsertParams{CourseID: course, Type: store.CalendarEventType(in.Type), Title: in.Title, StartsAt: in.StartsAt, EndsAt: in.EndsAt, Location: in.Location, Description: in.Description, CreatedBy: a.UserID})
		out = eventOf(e)
		return e.ID, err
	})
	return out, err
}

// Update sửa sự kiện của lớp (khoá lạc quan `version`).
func (s *Service) Update(ctx context.Context, a Actor, course, id uuid.UUID, version int, in EventIn) (EventOut, error) {
	if err := s.validate(&in); err != nil {
		return EventOut{}, err
	}
	var out EventOut
	err := s.write(ctx, a, course, "calendar.event.update", "calendar_event", nil, in, func(q *store.Queries) (uuid.UUID, error) {
		cur, err := q.CalEventGet(ctx, store.CalEventGetParams{ID: id, CourseID: course})
		if errors.Is(err, pgx.ErrNoRows) {
			return id, apierr.New(http.StatusNotFound, apierr.NotFound) // sự kiện lớp khác = không có
		}
		if err != nil {
			return id, fmt.Errorf("calendar: đọc sự kiện: %w", err)
		}
		e, err := q.CalEventUpdate(ctx, store.CalEventUpdateParams{ID: id, CourseID: course, Version: int32(version), Type: store.CalendarEventType(in.Type), Title: in.Title, StartsAt: in.StartsAt, EndsAt: in.EndsAt, Location: in.Location, Description: in.Description}) //nolint:gosec // version nhỏ
		if errors.Is(err, pgx.ErrNoRows) {
			return id, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current": eventOf(cur)})
		}
		if err != nil {
			return id, fmt.Errorf("calendar: sửa sự kiện: %w", err)
		}
		out = eventOf(e)
		return id, nil
	})
	return out, err
}

// Delete xoá sự kiện của lớp.
func (s *Service) Delete(ctx context.Context, a Actor, course, id uuid.UUID) error {
	return s.write(ctx, a, course, "calendar.event.delete", "calendar_event", nil, nil, func(q *store.Queries) (uuid.UUID, error) {
		n, err := q.CalEventDelete(ctx, store.CalEventDeleteParams{ID: id, CourseID: course})
		if err != nil {
			return id, fmt.Errorf("calendar: xoá sự kiện: %w", err)
		}
		if n == 0 {
			return id, apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		return id, nil
	})
}

// ---- nguồn cho tool (agent.ScheduleSource): đọc DB trực tiếp, không cache; chỉ lớp của trusted_context --------------------------------------------

var weekdays = [...]string{"Chủ Nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"} //nolint:gochecknoglobals // bảng hằng

// Format là "Thứ Hai, 21/09 · 14:00" theo Asia/Ho_Chi_Minh (lưu UTC).
func Format(t time.Time) string {
	t = t.In(vietnam)
	return fmt.Sprintf("%s, %s · %s", weekdays[t.Weekday()], t.Format("02/01"), t.Format("15:04"))
}

func (s *Service) facts(rs []row) agent.Facts {
	evs := make([]map[string]any, 0, len(rs))
	for _, r := range rs {
		m := map[string]any{"title": r.Title, "type": r.Type, "starts": Format(r.StartsAt)}
		if r.EndsAt != nil {
			m["ends"] = Format(*r.EndsAt)
		}
		if r.Location != nil {
			m["location"] = *r.Location
		}
		evs = append(evs, m)
	}
	return agent.Facts{"events": evs}
}

func (s *Service) viewer(tc agent.TrustedContext) Actor {
	return Actor{UserID: tc.UserID, Student: tc.Role == "STUDENT", Staff: tc.Role == "TEACHER" || tc.Role == "TA"}
}

// ExamSchedule: bài thi PE đã lên lịch + sự kiện EXAM từ bây giờ, tối đa 10.
func (s *Service) ExamSchedule(ctx context.Context, tc agent.TrustedContext) (agent.Facts, bool, error) {
	now := s.now()
	rs, err := s.rows(ctx, s.viewer(tc), tc.CourseID, now, now.AddDate(1, 0, 0), true, 10, nil)
	if err != nil || len(rs) == 0 {
		return nil, false, err
	}
	return s.facts(rs), true, nil
}

// Upcoming: mọi nguồn trong `days` ngày tới (kẹp 1–30; ≤ 0 → 7), tối đa 20.
func (s *Service) Upcoming(ctx context.Context, tc agent.TrustedContext, days int) (agent.Facts, bool, error) {
	if days <= 0 {
		days = 7
	}
	days = min(days, 30)
	now := s.now()
	rs, err := s.rows(ctx, s.viewer(tc), tc.CourseID, now, now.AddDate(0, 0, days), false, 20, nil)
	if err != nil || len(rs) == 0 {
		return nil, false, err
	}
	f := s.facts(rs)
	f["days"] = days
	return f, true, nil
}
