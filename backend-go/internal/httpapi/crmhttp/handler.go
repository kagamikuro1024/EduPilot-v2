// Package crmhttp: điểm danh và sự kiện tham gia của lớp (FEAT-crm-360 SRS 6.2). Handler mỏng, nghiệp vụ ở internal/crm.
package crmhttp

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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
	Idem  func(http.Handler) http.Handler // `Idempotency-Key` bắt buộc (nhắn sinh viên)
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
	r.With(staff).Get("/courses/{id}/students", h.listStudents)
	r.With(staff).Get("/courses/{id}/students/{uid}/profile", h.getProfile)
	r.With(staff).Get("/courses/{id}/students/{uid}/attendance", h.studentAttendance)
	r.With(staff).Get("/courses/{id}/students/{uid}/observations", h.listObservations)
	r.With(staff).Post("/courses/{id}/students/{uid}/observations", h.createObservation)
	r.With(staff).Put("/courses/{id}/students/{uid}/observations/{oid}", h.updateObservation)
	r.With(staff).Delete("/courses/{id}/students/{uid}/observations/{oid}", h.deleteObservation)
	r.With(staff).Post("/courses/{id}/students/{uid}/ai-summary", h.aiSummary)
	r.With(staff, h.Idem).Post("/courses/{id}/students/{uid}/outreach", h.outreach)
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

// ---- danh sách sinh viên (US-P5-04) ----

type studentRowOut struct {
	Student struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		StudentCode *string   `json:"student_code"`
	} `json:"student"`
	Attendance struct {
		Recorded int `json:"recorded"`
		Present  int `json:"present"`
		Late     int `json:"late"`
		Excused  int `json:"excused"`
		Absent   int `json:"absent"`
	} `json:"attendance"`
	Participation struct {
		SpeakCount  int    `json:"speak_count"`
		PointsTotal string `json:"points_total"`
	} `json:"participation"`
	Study struct {
		Minutes7d int  `json:"minutes_7d"`
		Tracked   bool `json:"tracked"`
	} `json:"study"`
	Activity struct {
		AsksLastWeek  int `json:"asks_last_week"`
		PostsLastWeek int `json:"posts_last_week"`
	} `json:"activity"`
	Risk struct {
		Level      string     `json:"level"`
		ComputedAt *time.Time `json:"computed_at"`
	} `json:"risk"`
}

type studentsPageOut struct {
	Items      []studentRowOut `json:"items"`
	NextCursor *string         `json:"next_cursor"`
	Total      int             `json:"total"`
}

const studentsDefaultLimit, studentsMaxLimit = 50, 100

// listStudents là `GET /courses/{id}/students?q=&filter=&cursor=&limit=` (Staff).
func (h *Handler) listStudents(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	in := crm.StudentsQuery{Q: strings.TrimSpace(r.URL.Query().Get("q")), Limit: studentsDefaultLimit}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > studentsMaxLimit {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "limit", Code: "range", Message: "limit phải là số nguyên từ 1 đến 100."}))
			return
		}
		in.Limit = n
	}
	for _, raw := range r.URL.Query()["filter"] {
		for _, f := range strings.Split(raw, ",") {
			switch strings.TrimSpace(f) {
			case "attention":
				in.Attention = true
			case "absent":
				in.Absent = true
			case "inactive":
				in.Inactive = true
			case "":
			default:
				apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "filter", Code: "invalid", Message: "Bộ lọc không hợp lệ."}))
				return
			}
		}
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		c, err := crm.ParseStudentCursor(raw)
		if err != nil {
			apierr.Write(w, r, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor))
			return
		}
		in.Cursor = &c
	}
	page, err := h.Svc.ListStudents(r.Context(), cid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := studentsPageOut{Items: make([]studentRowOut, len(page.Items)), NextCursor: page.NextCursor, Total: page.Total}
	for i, x := range page.Items {
		o := &out.Items[i]
		o.Student.ID, o.Student.FullName, o.Student.StudentCode = x.ID, x.FullName, x.StudentCode
		o.Attendance.Recorded, o.Attendance.Present, o.Attendance.Late, o.Attendance.Excused, o.Attendance.Absent = x.Recorded, x.Present, x.Late, x.Excused, x.Absent
		o.Participation.SpeakCount, o.Participation.PointsTotal = x.SpeakCount, x.PointsTotal.StringFixed(2)
		o.Study.Minutes7d, o.Study.Tracked = x.StudyMinutes7d, x.Tracked
		o.Activity.AsksLastWeek, o.Activity.PostsLastWeek = x.AsksLastWeek, x.PostsLastWeek
		o.Risk.Level, o.Risk.ComputedAt = x.RiskLevel, x.RiskComputedAt
	}
	w.Header().Set("Cache-Control", "no-store") // dữ liệu cá nhân (luật 15)
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ---- hồ sơ 360, ghi chú, nhận xét AI, nhắn sinh viên (US-P5-05) ----

func (h *Handler) ids(w http.ResponseWriter, r *http.Request) (course, student, actor uuid.UUID, ok bool) {
	if course, ok = idParam(w, r, "id"); !ok {
		return
	}
	if student, ok = idParam(w, r, "uid"); !ok {
		return
	}
	actor, ok = actorOf(w, r)
	return
}

type profileOut struct {
	Student struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		StudentCode *string   `json:"student_code"`
		JoinedAt    time.Time `json:"joined_at"`
	} `json:"student"`
	Risk struct {
		Level      string          `json:"level"`
		Sentence   *string         `json:"sentence"`
		Reasons    []crm.ReasonOut `json:"reasons"`
		ComputedAt *time.Time      `json:"computed_at"`
	} `json:"risk"`
	Attendance struct {
		Recorded int `json:"recorded"`
		Present  int `json:"present"`
		Late     int `json:"late"`
		Excused  int `json:"excused"`
		Absent   int `json:"absent"`
	} `json:"attendance"`
	Participation struct {
		SpeakCount  int    `json:"speak_count"`
		PointsTotal string `json:"points_total"`
	} `json:"participation"`
	Study struct {
		Minutes7d int             `json:"minutes_7d"`
		Weeks     []crm.StudyWeek `json:"weeks"`
		Tracked   bool            `json:"tracked"`
	} `json:"study"`
	Activity struct {
		AsksLastWeek  int `json:"asks_last_week"`
		PostsLastWeek int `json:"posts_last_week"`
	} `json:"activity"`
}

// getProfile: `GET …/students/{uid}/profile` — mỗi LẦN GỌI một dòng audit; no-store, không ETag (TLR-7).
func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	p, err := h.Svc.Profile(r.Context(), actor, cid, uid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var o profileOut
	o.Student.ID, o.Student.FullName, o.Student.StudentCode, o.Student.JoinedAt = p.StudentID, p.FullName, p.StudentCode, p.JoinedAt
	o.Risk.Level, o.Risk.Reasons, o.Risk.ComputedAt = p.Level, p.Reasons, p.ComputedAt
	if p.Sentence != "" {
		o.Risk.Sentence = &p.Sentence
	}
	o.Attendance.Recorded, o.Attendance.Present, o.Attendance.Late, o.Attendance.Excused, o.Attendance.Absent = p.Recorded, p.Present, p.Late, p.Excused, p.Absent
	o.Participation.SpeakCount, o.Participation.PointsTotal = p.SpeakCount, p.PointsTotal.StringFixed(2)
	o.Study.Minutes7d, o.Study.Weeks, o.Study.Tracked = p.Minutes7d, p.StudyWeeks, p.Tracked
	o.Activity.AsksLastWeek, o.Activity.PostsLastWeek = p.Asks, p.Posts
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, o)
}

type attendanceItemOut struct {
	Session struct {
		ID       uuid.UUID `json:"id"`
		No       int       `json:"session_no"`
		StartsAt time.Time `json:"starts_at"`
	} `json:"session"`
	Status string  `json:"status"`
	Note   *string `json:"note"`
}

func (h *Handler) studentAttendance(w http.ResponseWriter, r *http.Request) {
	cid, uid, _, ok := h.ids(w, r)
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var at *time.Time
	var after *uuid.UUID
	if pg.Cursor != nil {
		at, after = &pg.Cursor.CreatedAt, &pg.Cursor.ID
	}
	rows, err := h.Svc.StudentAttendance(r.Context(), cid, uid, at, after, pg.Limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	page := httpx.Paginate(rows, pg, func(x crm.AttendanceItem) (time.Time, string) { return x.StartsAt, x.SessionID.String() })
	items := make([]attendanceItemOut, len(page.Items))
	for i, x := range page.Items {
		items[i].Session.ID, items[i].Session.No, items[i].Session.StartsAt = x.SessionID, x.SessionNo, x.StartsAt
		items[i].Status, items[i].Note = x.Status, x.Note
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[attendanceItemOut]{Items: items, NextCursor: page.NextCursor})
}

type observationOut struct {
	ID     uuid.UUID `json:"id"`
	Author struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
	} `json:"author"`
	Kind      string    `json:"kind"`
	Tags      []string  `json:"tags"`
	Body      string    `json:"body"`
	AIEdited  bool      `json:"ai_edited"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func obsOut(o crm.Observation) observationOut {
	x := observationOut{ID: o.ID, Kind: o.Kind, Tags: o.Tags, Body: o.Body, AIEdited: o.AIEdited, Version: o.Version, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
	x.Author.ID, x.Author.FullName = o.AuthorID, o.AuthorName
	return x
}

func (h *Handler) listObservations(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var at *time.Time
	var after *uuid.UUID
	if pg.Cursor != nil {
		at, after = &pg.Cursor.CreatedAt, &pg.Cursor.ID
	}
	rows, err := h.Svc.ListObservations(r.Context(), actor, cid, uid, at, after, pg.Limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	page := httpx.Paginate(rows, pg, func(x crm.Observation) (time.Time, string) { return x.CreatedAt, x.ID.String() })
	items := make([]observationOut, len(page.Items))
	for i, x := range page.Items {
		items[i] = obsOut(x)
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[observationOut]{Items: items, NextCursor: page.NextCursor})
}

type observationBody struct {
	Kind     string   `json:"kind"`
	Tags     []string `json:"tags"`
	Body     string   `json:"body"`
	AIEdited bool     `json:"ai_edited"`
	Version  *int     `json:"version"`
}

func (h *Handler) createObservation(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	var b observationBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	o, err := h.Svc.CreateObservation(r.Context(), actor, cid, uid, crm.ObservationIn{Kind: b.Kind, Tags: b.Tags, Body: b.Body, AIEdited: b.AIEdited})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusCreated, obsOut(o))
}

func (h *Handler) updateObservation(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	oid, ok := idParam(w, r, "oid")
	if !ok {
		return
	}
	var b observationBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	version, aerr := httpx.WantedVersion(r, b.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	o, err := h.Svc.UpdateObservation(r.Context(), actor, cid, uid, oid, crm.ObservationIn{Tags: b.Tags, Body: b.Body, Version: version})
	var conflict *crm.ObservationConflict
	if errors.As(err, &conflict) {
		httpx.WriteVersionConflict(w, r, conflict.Current.Version, obsOut(conflict.Current))
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, obsOut(o))
}

func (h *Handler) deleteObservation(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	oid, ok := idParam(w, r, "oid")
	if !ok {
		return
	}
	if err := h.Svc.DeleteObservation(r.Context(), actor, cid, uid, oid); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) aiSummary(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.AISummary(r.Context(), actor, cid, uid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"draft": d})
}

func (h *Handler) outreach(w http.ResponseWriter, r *http.Request) {
	cid, uid, actor, ok := h.ids(w, r)
	if !ok {
		return
	}
	var b struct {
		Body string `json:"body"`
	}
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	res, err := h.Svc.Outreach(r.Context(), actor, cid, uid, b.Body)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"ticket": map[string]any{"id": res.TicketID, "kind": "TEACHER_OUTREACH", "status": res.Status, "created_at": res.CreatedAt}})
}
