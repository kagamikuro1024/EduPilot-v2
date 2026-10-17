package crmhttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

var ict = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) //nolint:gochecknoglobals // múi giờ cố định, bất biến

// mountMe: route của CHÍNH sinh viên (SRS 6.2 #7). Guard `Member` + vai lớp STUDENT; danh tính chỉ từ JWT, không tham số student.
func (h *Handler) mountMe(r chi.Router) {
	member := h.Guard(auth.Member)
	r.With(member).Get("/courses/{id}/me/attendance", h.myAttendance)
	r.With(member).Get("/courses/{id}/me/participation", h.myParticipation)
}

// meIDs trả (lớp, sinh viên) hoặc ghi lỗi: nhân viên / không phải STUDENT của lớp → 403.
func (h *Handler) meIDs(w http.ResponseWriter, r *http.Request) (course, student uuid.UUID, ok bool) {
	if course, ok = idParam(w, r, "id"); !ok {
		return
	}
	if student, ok = actorOf(w, r); !ok {
		return
	}
	if ca, _ := auth.CourseFromContext(r.Context()); ca.CourseRole != auth.RoleStudent {
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden))
		return course, student, false
	}
	return course, student, true
}

type myAttendanceSession struct {
	SessionNo int       `json:"session_no"`
	StartsAt  time.Time `json:"starts_at"`
	Status    string    `json:"status"`
}

type myAttendanceOut struct {
	Attendance struct {
		Recorded int                   `json:"recorded"`
		Present  int                   `json:"present"`
		Late     int                   `json:"late"`
		Excused  int                   `json:"excused"`
		Absent   int                   `json:"absent"`
		Sessions []myAttendanceSession `json:"sessions"`
	} `json:"attendance"`
	NextCursor *string `json:"next_cursor"`
}

func (h *Handler) myAttendance(w http.ResponseWriter, r *http.Request) {
	cid, uid, ok := h.meIDs(w, r)
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
	res, err := h.Svc.MyAttendance(r.Context(), cid, uid, at, after, pg.Limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	page := httpx.Paginate(res.Sessions, pg, func(x crm.AttendanceItem) (time.Time, string) { return x.StartsAt, x.SessionID.String() })
	var out myAttendanceOut
	a := &out.Attendance
	a.Recorded, a.Present, a.Late, a.Excused, a.Absent = res.Recorded, res.Present, res.Late, res.Excused, res.Absent
	a.Sessions = make([]myAttendanceSession, len(page.Items))
	for i, x := range page.Items {
		a.Sessions[i] = myAttendanceSession{SessionNo: x.SessionNo, StartsAt: x.StartsAt, Status: x.Status}
	}
	out.NextCursor = page.NextCursor
	w.Header().Set("Cache-Control", "no-store") // luật 15: điểm danh không cache
	httpx.WriteJSON(w, http.StatusOK, out)
}

type myEvent struct {
	Date   string `json:"date"`
	Type   string `json:"type"`
	Points string `json:"points"`
}

type myParticipationOut struct {
	Participation struct {
		SpeakCount  int       `json:"speak_count"`
		PointsTotal string    `json:"points_total"`
		Events      []myEvent `json:"events"`
	} `json:"participation"`
	NextCursor *string `json:"next_cursor"`
}

func (h *Handler) myParticipation(w http.ResponseWriter, r *http.Request) {
	cid, uid, ok := h.meIDs(w, r)
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
	res, err := h.Svc.MyParticipation(r.Context(), cid, uid, at, after, pg.Limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	page := httpx.Paginate(res.Events, pg, func(x crm.ParticipationRow) (time.Time, string) { return x.CreatedAt, x.ID.String() })
	var out myParticipationOut
	p := &out.Participation
	p.SpeakCount, p.PointsTotal = res.SpeakCount, res.PointsTotal.StringFixed(2)
	p.Events = make([]myEvent, len(page.Items))
	for i, x := range page.Items {
		p.Events[i] = myEvent{Date: x.CreatedAt.In(ict).Format(time.DateOnly), Type: x.Type, Points: x.Points.StringFixed(2)}
	}
	out.NextCursor = page.NextCursor
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}
