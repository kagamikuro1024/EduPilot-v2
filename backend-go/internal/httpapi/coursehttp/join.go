package coursehttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

type codeBody struct {
	Code string `json:"code"`
}

func (h *Handler) clientIP(r *http.Request) string {
	if h.ClientIP == nil {
		return ""
	}
	return h.ClientIP(r)
}

type teacherName struct {
	FullName string `json:"full_name"`
}

type previewOut struct {
	Name      string        `json:"name"`
	ClassCode string        `json:"class_code"`
	Semester  string        `json:"semester"`
	Teachers  []teacherName `json:"teachers"`
	State     string        `json:"state"`
}

func (h *Handler) joinPreview(w http.ResponseWriter, r *http.Request) {
	var b codeBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	pv, err := h.Courses.Preview(r.Context(), uid, h.clientIP(r), b.Code)
	if h.fail(w, r, "join-preview", err) {
		return
	}
	out := previewOut{Name: pv.Name, ClassCode: pv.ClassCode, Semester: pv.Semester, State: string(pv.State), Teachers: make([]teacherName, len(pv.Teachers))}
	for i, n := range pv.Teachers {
		out.Teachers[i] = teacherName{FullName: n}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	var b codeBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	res, err := h.Courses.Join(r.Context(), uid, h.clientIP(r), b.Code)
	if h.fail(w, r, "join", err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"course_id": res.CourseID, "status": res.Status, "already_member": res.AlreadyMember})
}

type joinInfoOut struct {
	JoinCode           string     `json:"join_code"`
	JoinURL            string     `json:"join_url"`
	Enabled            bool       `json:"enabled"`
	ExpiresAt          *time.Time `json:"expires_at"`
	RequireApproval    bool       `json:"require_approval"`
	AllowedEmailDomain *string    `json:"allowed_email_domain"`
	Capacity           *int       `json:"capacity"`
	ActiveStudents     int        `json:"active_students"`
	Pending            int        `json:"pending"`
	Version            int        `json:"version"`
}

func joinInfoOf(i course.JoinInfo) joinInfoOut {
	return joinInfoOut{JoinCode: i.JoinCode, JoinURL: i.JoinURL, Enabled: i.Enabled, ExpiresAt: i.ExpiresAt, RequireApproval: i.RequireApproval, AllowedEmailDomain: i.AllowedEmailDomain,
		Capacity: i.Capacity, ActiveStudents: i.ActiveStudents, Pending: i.Pending, Version: i.Version}
}

func (h *Handler) joinCode(w http.ResponseWriter, r *http.Request) {
	id, ok := courseIDOf(w, r)
	if !ok {
		return
	}
	info, err := h.Courses.JoinCode(r.Context(), id)
	if h.fail(w, r, "join-code", err) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("ETag", httpx.ETagVersion(info.Version))
	httpx.WriteJSON(w, http.StatusOK, joinInfoOf(info))
}

func (h *Handler) regenerate(w http.ResponseWriter, r *http.Request) {
	id, ok := courseIDOf(w, r)
	if !ok {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	info, err := h.Courses.Regenerate(r.Context(), actor, id)
	if h.fail(w, r, "regenerate", err) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"join_code": info.JoinCode, "join_url": info.JoinURL, "version": info.Version})
}

// optField phân biệt khoá vắng (giữ nguyên) với null / chuỗi rỗng (bỏ).
type optField[T any] struct {
	Set bool
	Val *T
}

func (o *optField[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Val = &v
	return nil
}

type settingsBody struct {
	Version            *int                `json:"version"`
	Enabled            *bool               `json:"enabled"`
	RequireApproval    *bool               `json:"require_approval"`
	ExpiresAt          optField[time.Time] `json:"expires_at"`
	AllowedEmailDomain optField[string]    `json:"allowed_email_domain"`
	Capacity           optField[int]       `json:"capacity"`
}

func (h *Handler) putJoinSettings(w http.ResponseWriter, r *http.Request) {
	id, ok := courseIDOf(w, r)
	if !ok {
		return
	}
	var b settingsBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	version, aerr := httpx.WantedVersion(r, b.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	in := course.SettingsInput{Version: version, Enabled: b.Enabled, RequireApproval: b.RequireApproval,
		SetExpires: b.ExpiresAt.Set, ExpiresAt: b.ExpiresAt.Val, SetDomain: b.AllowedEmailDomain.Set, Domain: b.AllowedEmailDomain.Val, SetCapacity: b.Capacity.Set, Capacity: b.Capacity.Val}
	info, err := h.Courses.PutJoinSettings(r.Context(), actor, id, in)
	if h.fail(w, r, "join-settings", err) {
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(info.Version))
	httpx.WriteJSON(w, http.StatusOK, joinInfoOf(info))
}

type memberOut struct {
	UserID          uuid.UUID `json:"user_id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	StudentCode     string    `json:"student_code"`
	RoleInCourse    string    `json:"role_in_course"`
	Status          string    `json:"status"`
	JoinedVia       string    `json:"joined_via"`
	Warning         *string   `json:"warning"`
	StatusChangedAt time.Time `json:"status_changed_at"`
}

type memberPage struct {
	Items      []memberOut `json:"items"`
	NextCursor *string     `json:"next_cursor"`
	Counts     struct {
		Active  int `json:"active"`
		Pending int `json:"pending"`
	} `json:"counts"`
}

func (h *Handler) members(w http.ResponseWriter, r *http.Request) {
	id, ok := courseIDOf(w, r)
	if !ok {
		return
	}
	p, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	f := course.MemberFilter{Status: q.Get("status"), Role: q.Get("role"), Q: q.Get("q"), Limit: p.Limit}
	switch f.Status {
	case "", "ACTIVE", "PENDING", "REMOVED":
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "status", Code: "INVALID_STATUS", Message: "Trạng thái không hợp lệ."}))
		return
	}
	switch f.Role {
	case "", "STUDENT", "TA", "TEACHER":
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "role", Code: "INVALID_ROLE", Message: "Vai không hợp lệ."}))
		return
	}
	if len(f.Q) > 100 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "q", Code: "TOO_LONG", Message: "Từ khoá tối đa 100 ký tự."}))
		return
	}
	if access, ok := auth.CourseFromContext(r.Context()); ok {
		f.ByStudentCode = access.Status == "ACTIVE" && (access.CourseRole == auth.RoleTeacher || access.CourseRole == auth.RoleTA)
	}
	if p.Cursor != nil {
		f.Cursor = &course.Cursor{At: p.Cursor.CreatedAt, ID: p.Cursor.ID}
	}
	res, err := h.Courses.Members(r.Context(), id, f)
	if h.fail(w, r, "members", err) {
		return
	}
	page := httpx.Paginate(res.Rows, p, func(m course.Member) (time.Time, string) { return m.StatusChangedAt, m.UserID.String() })
	out := memberPage{NextCursor: page.NextCursor, Items: make([]memberOut, len(page.Items))}
	out.Counts.Active, out.Counts.Pending = res.Active, res.Pending
	for i, m := range page.Items {
		out.Items[i] = memberOut{UserID: m.UserID, FullName: m.FullName, Email: m.Email, StudentCode: m.StudentCode, RoleInCourse: m.RoleInCourse, Status: m.Status, JoinedVia: m.JoinedVia, Warning: m.Warning, StatusChangedAt: m.StatusChangedAt}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

type transitionOut struct {
	UserID         uuid.UUID `json:"user_id"`
	Status         string    `json:"status"`
	PreviousStatus *string   `json:"previous_status"`
	Warning        *string   `json:"warning"`
}

// callerActor: giảng viên của lớp hoặc Admin (chế độ Manage) khác TA.
func callerActor(w http.ResponseWriter, r *http.Request) (course.Actor, bool) {
	id, ok := actorOf(w, r)
	if !ok {
		return course.Actor{}, false
	}
	p := auth.MustFromContext(r.Context())
	a, _ := auth.CourseFromContext(r.Context())
	return course.Actor{ID: id, Teacher: p.Role == auth.RoleAdmin || (a.Status == "ACTIVE" && a.CourseRole == auth.RoleTeacher)}, true
}

type memberRoute func(actor course.Actor, courseID, uid uuid.UUID, r *http.Request) (course.Transition, error)

func (h *Handler) memberOp(op string, run memberRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cid, ok := courseIDOf(w, r)
		if !ok {
			return
		}
		uid, err := uuid.Parse(chi.URLParam(r, "uid"))
		if err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return
		}
		actor, ok := callerActor(w, r)
		if !ok {
			return
		}
		t, err := run(actor, cid, uid, r)
		if h.fail(w, r, op, err) {
			return
		}
		httpx.WriteJSON(w, http.StatusOK, transitionOut{UserID: t.UserID, Status: t.Status, PreviousStatus: t.PreviousStatus, Warning: t.Warning})
	}
}

type approveBody struct {
	ConfirmMismatch bool `json:"confirm_mismatch"`
}

func (h *Handler) approve() http.HandlerFunc {
	return h.memberOp("approve", func(a course.Actor, cid, uid uuid.UUID, r *http.Request) (course.Transition, error) {
		var b approveBody
		if r.ContentLength != 0 {
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&b); err != nil && !errors.Is(err, io.EOF) {
				return course.Transition{}, &course.InvalidError{Field: "body", Code: "INVALID_BODY", Message: "Thân yêu cầu không đọc được."}
			}
		}
		return h.Courses.Approve(r.Context(), a, cid, uid, b.ConfirmMismatch)
	})
}

func (h *Handler) reject() http.HandlerFunc {
	return h.memberOp("reject", func(a course.Actor, cid, uid uuid.UUID, r *http.Request) (course.Transition, error) {
		return h.Courses.Reject(r.Context(), a, cid, uid)
	})
}

func (h *Handler) removeMember() http.HandlerFunc {
	return h.memberOp("remove", func(a course.Actor, cid, uid uuid.UUID, r *http.Request) (course.Transition, error) {
		return h.Courses.Remove(r.Context(), a, cid, uid)
	})
}

func (h *Handler) undo() http.HandlerFunc {
	return h.memberOp("undo", func(a course.Actor, cid, uid uuid.UUID, r *http.Request) (course.Transition, error) {
		return h.Courses.Undo(r.Context(), a, cid, uid)
	})
}

func courseIDOf(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return uuid.Nil, false
	}
	return id, true
}
