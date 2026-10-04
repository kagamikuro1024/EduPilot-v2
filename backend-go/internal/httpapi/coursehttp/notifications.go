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

type notificationItem struct {
	ID        uuid.UUID  `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	Link      *string    `json:"link"`
	CourseID  *uuid.UUID `json:"course_id"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type notificationPage struct {
	Items       []notificationItem `json:"items"`
	NextCursor  *string            `json:"next_cursor"`
	UnreadCount int                `json:"unread_count"`
}

func (h *Handler) notifications(w http.ResponseWriter, r *http.Request) {
	uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	p, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	unreadOnly := false
	switch r.URL.Query().Get("unread_only") {
	case "", "false", "0":
	case "true", "1":
		unreadOnly = true
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "unread_only", Code: "BOOL", Message: "unread_only là true hoặc false."}))
		return
	}
	var cur *course.Cursor
	if p.Cursor != nil {
		cur = &course.Cursor{At: p.Cursor.CreatedAt, ID: p.Cursor.ID}
	}
	res, err := h.Courses.Notifications(r.Context(), uid, unreadOnly, cur, p.Limit)
	if h.fail(w, r, "notifications", err) {
		return
	}
	page := httpx.Paginate(res.Rows, p, func(n course.Notification) (time.Time, string) { return n.CreatedAt, n.ID.String() })
	items := make([]notificationItem, len(page.Items))
	for i, n := range page.Items {
		items[i] = notificationItem{ID: n.ID, Type: n.Type, Title: n.Title, Body: n.Body, Link: n.Link, CourseID: n.CourseID, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(w, http.StatusOK, notificationPage{Items: items, NextCursor: page.NextCursor, UnreadCount: res.UnreadCount})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	if h.fail(w, r, "notification-read", h.Courses.MarkRead(r.Context(), uid, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
