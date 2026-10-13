// Package crmhttp: điểm danh và sự kiện tham gia của lớp (FEAT-crm-360 SRS 6.2). Handler mỏng, nghiệp vụ ở internal/crm.
package crmhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler là route điểm danh / tham gia. Guard `Staff`: TA + TEACHER của lớp; STUDENT, ADMIN, người ngoài lớp → 403.
type Handler struct {
	Svc   *crm.Service
	Guard func(auth.GuardMode) func(http.Handler) http.Handler
	Log   *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	staff := h.Guard(auth.Staff)
	r.With(staff).Get("/courses/{id}/sessions/{sid}/attendance", h.getGrid)
	r.With(staff).Put("/courses/{id}/sessions/{sid}/attendance", h.putGrid)
	r.With(staff).Post("/courses/{id}/participation", h.addParticipation)
	r.With(staff).Get("/courses/{id}/participation", h.listParticipation)
	r.With(staff).Delete("/courses/{id}/participation/{pid}", h.deleteParticipation)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	h.Log.ErrorContext(r.Context(), "crm: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
}

func actorOf(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	p, ok := auth.FromContext(r.Context())
	id, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return uuid.Nil, false
	}
	return id, true
}

func idParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		notFound(w, r)
		return uuid.Nil, false
	}
	return id, true
}

// ---- lưới điểm danh ----

type markedBy struct {
	FullName string `json:"full_name"`
}

type recordOut struct {
	Status   string    `json:"status"`
	Note     *string   `json:"note"`
	Version  int       `json:"version"`
	MarkedBy markedBy  `json:"marked_by"`
	MarkedAt time.Time `json:"marked_at"`
}

type studentOut struct {
	ID          uuid.UUID  `json:"id"`
	FullName    string     `json:"full_name"`
	StudentCode *string    `json:"student_code"`
	Record      *recordOut `json:"record"`
	SpeakCount  int        `json:"speak_count"`
}

type gridSessionOut struct {
	ID               uuid.UUID  `json:"id"`
	SessionNo        int        `json:"session_no"`
	StartsAt         time.Time  `json:"starts_at"`
	EndsAt           time.Time  `json:"ends_at"`
	Room             *string    `json:"room"`
	Topic            *string    `json:"topic"`
	AttendanceDoneAt *time.Time `json:"attendance_done_at"`
}

type gridOut struct {
	Session  gridSessionOut `json:"session"`
	Students []studentOut   `json:"students"`
}

func (h *Handler) getGrid(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "sid")
	if !ok {
		return
	}
	g, err := h.Svc.GetGrid(r.Context(), cid, sid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := gridOut{
		Session:  gridSessionOut{ID: g.Session.ID, SessionNo: g.Session.SessionNo, StartsAt: g.Session.StartsAt, EndsAt: g.Session.EndsAt, Room: g.Session.Room, Topic: g.Session.Topic, AttendanceDoneAt: g.Session.AttendanceDoneAt},
		Students: make([]studentOut, len(g.Students)),
	}
	for i, s := range g.Students {
		st := studentOut{ID: s.ID, FullName: s.FullName, StudentCode: s.StudentCode, SpeakCount: s.SpeakCount}
		if s.Record != nil {
			st.Record = &recordOut{Status: s.Record.Status, Note: s.Record.Note, Version: s.Record.Version, MarkedBy: markedBy{FullName: s.Record.MarkedBy}, MarkedAt: s.Record.MarkedAt}
		}
		out.Students[i] = st
	}
	w.Header().Set("Cache-Control", "no-store") // dữ liệu điểm danh không cache (luật 15)
	httpx.WriteJSON(w, http.StatusOK, out)
}

type putRecord struct {
	StudentID uuid.UUID `json:"student_id"`
	Status    string    `json:"status"`
	Note      *string   `json:"note"`
	Version   *int      `json:"version"`
}

type putBody struct {
	Records  []putRecord `json:"records"`
	Complete bool        `json:"complete"`
}

type putRecordOut struct {
	StudentID uuid.UUID `json:"student_id"`
	Status    string    `json:"status"`
	Version   int       `json:"version"`
}

type putOut struct {
	Records []putRecordOut `json:"records"`
	DoneAt  *time.Time     `json:"done_at"`
}

func (h *Handler) putGrid(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "sid")
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var b putBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	in := crm.PutGridInput{Complete: b.Complete, Records: make([]crm.RecordIn, len(b.Records))}
	for i, x := range b.Records {
		in.Records[i] = crm.RecordIn{StudentID: x.StudentID, Status: x.Status, Note: x.Note, Version: x.Version}
	}
	res, err := h.Svc.PutGrid(r.Context(), actor, cid, sid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := putOut{Records: make([]putRecordOut, len(res.Records)), DoneAt: res.DoneAt}
	for i, x := range res.Records {
		out.Records[i] = putRecordOut{StudentID: x.StudentID, Status: x.Status, Version: x.Version}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ---- sự kiện tham gia ----

type participationBody struct {
	StudentID     uuid.UUID        `json:"student_id"`
	SessionID     *uuid.UUID       `json:"session_id"`
	Type          string           `json:"type"`
	Points        *decimal.Decimal `json:"points"`
	Note          *string          `json:"note"`
	ClientEventID uuid.UUID        `json:"client_event_id"`
}

type participationOut struct {
	ID            uuid.UUID  `json:"id"`
	StudentID     uuid.UUID  `json:"student_id"`
	SessionID     *uuid.UUID `json:"session_id"`
	Type          string     `json:"type"`
	Points        string     `json:"points"`
	Note          *string    `json:"note"`
	ClientEventID uuid.UUID  `json:"client_event_id"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (h *Handler) addParticipation(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var b participationBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	p, created, err := h.Svc.AddParticipation(r.Context(), actor, cid, crm.ParticipationIn{StudentID: b.StudentID, SessionID: b.SessionID, Type: b.Type, Points: b.Points, Note: b.Note, ClientEventID: b.ClientEventID})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, participationOut{ID: p.ID, StudentID: p.StudentID, SessionID: p.SessionID, Type: p.Type, Points: p.Points.StringFixed(2), Note: p.Note, ClientEventID: p.ClientEventID, CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt})
}

type partStudent struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type partItem struct {
	ID        uuid.UUID   `json:"id"`
	Student   partStudent `json:"student"`
	SessionID *uuid.UUID  `json:"session_id"`
	Type      string      `json:"type"`
	Points    string      `json:"points"`
	Note      *string     `json:"note"`
	CreatedBy uuid.UUID   `json:"created_by"`
	CreatedAt time.Time   `json:"created_at"`
}

func (h *Handler) listParticipation(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var student, session *uuid.UUID
	for name, dst := range map[string]**uuid.UUID{"student_id": &student, "session_id": &session} {
		if v := r.URL.Query().Get(name); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: name, Code: "INVALID", Message: "Mã không hợp lệ."}))
				return
			}
			*dst = &id
		}
	}
	var at *time.Time
	var after *uuid.UUID
	if pg.Cursor != nil {
		at, after = &pg.Cursor.CreatedAt, &pg.Cursor.ID
	}
	rows, err := h.Svc.ListParticipation(r.Context(), cid, student, session, at, after, pg.Limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	page := httpx.Paginate(rows, pg, func(x crm.ParticipationRow) (time.Time, string) { return x.CreatedAt, x.ID.String() })
	items := make([]partItem, len(page.Items))
	for i, x := range page.Items {
		items[i] = partItem{ID: x.ID, Student: partStudent{ID: x.StudentID, FullName: x.FullName}, SessionID: x.SessionID, Type: x.Type, Points: x.Points.StringFixed(2), Note: x.Note, CreatedBy: x.CreatedBy, CreatedAt: x.CreatedAt}
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[partItem]{Items: items, NextCursor: page.NextCursor})
}

func (h *Handler) deleteParticipation(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	pid, ok := idParam(w, r, "pid")
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	ca, _ := auth.CourseFromContext(r.Context())
	if err := h.Svc.DeleteParticipation(r.Context(), actor, ca.CourseRole == auth.RoleTeacher, cid, pid); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
