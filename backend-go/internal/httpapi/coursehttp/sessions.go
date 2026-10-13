package coursehttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

type generateBody struct {
	Weekdays       []int    `json:"weekdays"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	Room           *string  `json:"room"`
	From           string   `json:"from"`
	To             string   `json:"to"`
	ExcludeDates   []string `json:"exclude_dates"`
	FirstSessionNo *int     `json:"first_session_no"`
}

type generateOut struct {
	Created        int  `json:"created"`
	FirstSessionNo *int `json:"first_session_no"`
	LastSessionNo  *int `json:"last_session_no"`
	Skipped        int  `json:"skipped"`
}

type plannedOut struct {
	SessionNo int       `json:"session_no"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
}

type skipOut struct {
	Date   string `json:"date"`
	Reason string `json:"reason"`
}

type sessionPreviewOut struct {
	WouldCreate []plannedOut `json:"would_create"`
	Skipped     []skipOut    `json:"skipped"`
}

// isDryRun: `?dry_run=true` = xem trước (chỉ đọc, không đòi Idempotency-Key — TLR-12).
func isDryRun(r *http.Request) bool { return r.URL.Query().Get("dry_run") == "true" }

// idemUnlessDryRun đòi `Idempotency-Key` cho lần chạy thật; xem trước không ghi gì nên bỏ qua middleware (băm thân + query sẽ từ chối dùng lại khoá).
func (h *Handler) idemUnlessDryRun(next http.Handler) http.Handler {
	withKey := h.Idem(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isDryRun(r) {
			next.ServeHTTP(w, r)
			return
		}
		withKey.ServeHTTP(w, r)
	})
}

// generateSessions: giảng viên / TA tạo buổi học theo tuần (201), hoặc xem trước (`?dry_run=true`, 200). Chạy lại không tạo trùng.
func (h *Handler) generateSessions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var b generateBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	res, err := h.Courses.GenerateSessions(r.Context(), actor, id, course.GenerateInput{
		Weekdays: b.Weekdays, StartTime: b.StartTime, EndTime: b.EndTime, Room: b.Room, From: b.From, To: b.To, ExcludeDates: b.ExcludeDates, FirstSessionNo: b.FirstSessionNo,
		DryRun: isDryRun(r),
	})
	if h.fail(w, r, "sessions-generate", err) {
		return
	}
	if isDryRun(r) {
		out := sessionPreviewOut{WouldCreate: make([]plannedOut, len(res.WouldCreate)), Skipped: make([]skipOut, len(res.SkippedItems))}
		for i, p := range res.WouldCreate {
			out.WouldCreate[i] = plannedOut{SessionNo: p.SessionNo, StartsAt: p.StartsAt, EndsAt: p.EndsAt}
		}
		for i, k := range res.SkippedItems {
			out.Skipped[i] = skipOut{Date: k.Date, Reason: k.Reason}
		}
		httpx.WriteJSON(w, http.StatusOK, out)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, generateOut{Created: res.Created, FirstSessionNo: res.FirstSessionNo, LastSessionNo: res.LastSessionNo, Skipped: res.Skipped})
}

// sessionAttendance chỉ nhân viên thấy: buổi đã hoàn tất điểm danh chưa, đã có dòng điểm danh nào chưa (khoá nút Xoá).
type sessionAttendance struct {
	DoneAt     *time.Time `json:"done_at"`
	HasRecords bool       `json:"has_records"`
}

type sessionOut struct {
	ID         uuid.UUID          `json:"id"`
	SessionNo  int                `json:"session_no"`
	StartsAt   time.Time          `json:"starts_at"`
	EndsAt     time.Time          `json:"ends_at"`
	Room       *string            `json:"room"`
	Topic      *string            `json:"topic"`
	Version    int                `json:"version"`
	Attendance *sessionAttendance `json:"attendance,omitempty"`
}

func sessionOutOf(s course.SessionRow, staff bool) sessionOut {
	o := sessionOut{ID: s.ID, SessionNo: s.SessionNo, StartsAt: s.StartsAt, EndsAt: s.EndsAt, Room: s.Room, Topic: s.Topic, Version: s.Version}
	if staff {
		o.Attendance = &sessionAttendance{DoneAt: s.AttendanceDoneAt, HasRecords: s.HasAttendance}
	}
	return o
}

type sessionPutBody struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Room     *string   `json:"room"`
	Topic    *string   `json:"topic"`
	Version  int       `json:"version"`
}

// updateSession: giảng viên / TA sửa giờ / phòng / chủ đề một buổi (`session_no` không đổi). 200 {session}.
func (h *Handler) updateSession(w http.ResponseWriter, r *http.Request) {
	id, sid, ok := sessionIDs(w, r)
	if !ok {
		return
	}
	var b sessionPutBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	row, err := h.Courses.UpdateSession(r.Context(), actor, id, sid, course.SessionInput{StartsAt: b.StartsAt, EndsAt: b.EndsAt, Room: b.Room, Topic: b.Topic, Version: b.Version})
	if h.fail(w, r, "session-update", err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]sessionOut{"session": sessionOutOf(row, true)})
}

// deleteSession: huỷ một buổi chưa có điểm danh. 204.
func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	id, sid, ok := sessionIDs(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	if h.fail(w, r, "session-delete", h.Courses.DeleteSession(r.Context(), actor, id, sid)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sessionIDs(w http.ResponseWriter, r *http.Request) (course, session uuid.UUID, ok bool) {
	course, err := uuid.Parse(chi.URLParam(r, "id"))
	if err == nil {
		session, err = uuid.Parse(chi.URLParam(r, "sid"))
	}
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return uuid.Nil, uuid.Nil, false
	}
	return course, session, true
}

// listSessions: thành viên ACTIVE của lớp xem lịch, theo starts_at, phân trang con trỏ.
func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	p, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var after *time.Time
	if p.Cursor != nil {
		after = &p.Cursor.CreatedAt
	}
	rows, err := h.Courses.ListSessions(r.Context(), id, after, p.Limit)
	if h.fail(w, r, "sessions-list", err) {
		return
	}
	page := httpx.Paginate(rows, p, func(s course.SessionRow) (time.Time, string) { return s.StartsAt, s.ID.String() })
	access, _ := auth.CourseFromContext(r.Context())
	staff := access.Status == "ACTIVE" && (access.CourseRole == auth.RoleTeacher || access.CourseRole == auth.RoleTA)
	items := make([]sessionOut, len(page.Items))
	for i, s := range page.Items {
		items[i] = sessionOutOf(s, staff)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[sessionOut]{Items: items, NextCursor: page.NextCursor})
}
