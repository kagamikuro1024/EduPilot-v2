package coursehttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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

// generateSessions: giảng viên / TA tạo buổi học theo tuần (201). Chạy lại không tạo trùng.
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
	})
	if h.fail(w, r, "sessions-generate", err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, generateOut{Created: res.Created, FirstSessionNo: res.FirstSessionNo, LastSessionNo: res.LastSessionNo, Skipped: res.Skipped})
}

type sessionOut struct {
	ID        uuid.UUID `json:"id"`
	SessionNo int       `json:"session_no"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Room      *string   `json:"room"`
	Topic     *string   `json:"topic"`
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
	items := make([]sessionOut, len(page.Items))
	for i, s := range page.Items {
		items[i] = sessionOut{ID: s.ID, SessionNo: s.SessionNo, StartsAt: s.StartsAt, EndsAt: s.EndsAt, Room: s.Room, Topic: s.Topic}
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[sessionOut]{Items: items, NextCursor: page.NextCursor})
}
