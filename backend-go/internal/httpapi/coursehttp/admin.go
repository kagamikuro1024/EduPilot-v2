package coursehttp

import (
	"encoding/json"
	"errors"
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

type person struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

func personOf(p *course.Person) *person {
	if p == nil {
		return nil
	}
	return &person{ID: p.ID, FullName: p.FullName}
}

type adminItem struct {
	ID              uuid.UUID `json:"id"`
	ClassCode       string    `json:"class_code"`
	SubjectCode     string    `json:"subject_code"`
	Name            string    `json:"name"`
	Semester        string    `json:"semester"`
	Status          string    `json:"status"`
	Teacher         *person   `json:"teacher"`
	AssistantsCount int       `json:"assistants_count"`
	StudentsActive  int       `json:"students_active"`
	StudentsPending int       `json:"students_pending"`
	Capacity        *int      `json:"capacity"`
	Version         int       `json:"version"`
}

func adminItemOf(v course.AdminView) adminItem {
	return adminItem{ID: v.ID, ClassCode: v.ClassCode, SubjectCode: v.SubjectCode, Name: v.Name, Semester: v.Semester, Status: v.Status, Teacher: personOf(v.Teacher),
		AssistantsCount: v.AssistantsCount, StudentsActive: v.StudentsActive, StudentsPending: v.StudentsPending, Capacity: v.Capacity, Version: v.Version}
}

func (h *Handler) adminList(w http.ResponseWriter, r *http.Request) {
	p, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	f := course.AdminFilter{Status: q.Get("status"), Semester: q.Get("semester"), Q: q.Get("q"), Limit: p.Limit}
	if f.Status != "" && f.Status != "ACTIVE" && f.Status != "ARCHIVED" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "status", Code: "INVALID_STATUS", Message: "Trạng thái không hợp lệ."}))
		return
	}
	if len(f.Q) > 100 || len(f.Semester) > 20 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "q", Code: "TOO_LONG", Message: "Từ khoá quá dài."}))
		return
	}
	if p.Cursor != nil {
		f.Cursor = &course.Cursor{At: p.Cursor.CreatedAt, ID: p.Cursor.ID}
	}
	rows, err := h.Courses.AdminList(r.Context(), f)
	if h.fail(w, r, "admin-list", err) {
		return
	}
	page := httpx.Paginate(rows, p, func(v course.AdminView) (time.Time, string) { return v.CreatedAt, v.ID.String() })
	items := make([]adminItem, len(page.Items))
	for i, v := range page.Items {
		items[i] = adminItemOf(v)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Page[adminItem]{Items: items, NextCursor: page.NextCursor})
}

// optInt phân biệt khoá vắng (giữ nguyên) với `null` (bỏ giới hạn) và số.
type optInt struct {
	Set bool
	Val *int
}

func (o *optInt) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		return nil
	}
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Val = &v
	return nil
}

type createBody struct {
	SubjectCode string    `json:"subject_code"`
	ClassCode   string    `json:"class_code"`
	Name        string    `json:"name"`
	Semester    string    `json:"semester"`
	Capacity    *int      `json:"capacity"`
	TeacherID   *string   `json:"teacher_id"`
	TAIDs       *[]string `json:"ta_ids"`
	JoinCode    string    `json:"join_code"`
}

type createOut struct {
	ID          uuid.UUID `json:"id"`
	ClassCode   string    `json:"class_code"`
	SubjectCode string    `json:"subject_code"`
	Name        string    `json:"name"`
	Semester    string    `json:"semester"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	Teacher     *person   `json:"teacher"`
}

func parseID(field, s string) (uuid.UUID, *apierr.Error) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return uuid.Nil, apierr.Validation(apierr.FieldError{Field: field, Code: "INVALID_ID", Message: "Mã người dùng không hợp lệ."})
	}
	return id, nil
}

func parseIDs(field string, in *[]string) (*[]uuid.UUID, *apierr.Error) {
	if in == nil {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, len(*in))
	for _, s := range *in {
		id, aerr := parseID(field, s)
		if aerr != nil {
			return nil, aerr
		}
		out = append(out, id)
	}
	return &out, nil
}

func (h *Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	var b createBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	in := course.CreateInput{SubjectCode: b.SubjectCode, ClassCode: b.ClassCode, Name: b.Name, Semester: b.Semester, Capacity: b.Capacity, JoinCode: b.JoinCode}
	if b.TeacherID != nil {
		id, aerr := parseID("teacher_id", *b.TeacherID)
		if aerr != nil {
			apierr.Write(w, r, aerr)
			return
		}
		in.TeacherID = &id
	}
	tas, aerr := parseIDs("ta_ids", b.TAIDs)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	if tas != nil {
		in.TAIDs = *tas
	}
	v, err := h.Courses.Create(r.Context(), actor, in)
	if h.fail(w, r, "admin-create", err) {
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, createOut{ID: v.ID, ClassCode: v.ClassCode, SubjectCode: v.SubjectCode, Name: v.Name, Semester: v.Semester, Status: v.Status, Version: v.Version, Teacher: personOf(v.Teacher)})
}

type updateBody struct {
	Version     *int    `json:"version"`
	SubjectCode *string `json:"subject_code"`
	ClassCode   *string `json:"class_code"`
	Name        *string `json:"name"`
	Semester    *string `json:"semester"`
	Capacity    optInt  `json:"capacity"`
}

func (h *Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var b updateBody
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
	v, err := h.Courses.Update(r.Context(), actor, id, course.UpdateInput{Version: version, SubjectCode: b.SubjectCode, ClassCode: b.ClassCode, Name: b.Name, Semester: b.Semester, SetCapacity: b.Capacity.Set, Capacity: b.Capacity.Val})
	if h.fail(w, r, "admin-update", err) {
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(v.Version))
	httpx.WriteJSON(w, http.StatusOK, adminItemOf(v))
}

func (h *Handler) adminArchive(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	v, err := h.Courses.Archive(r.Context(), actor, id)
	if h.fail(w, r, "admin-archive", err) {
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(v.Version))
	httpx.WriteJSON(w, http.StatusOK, adminItemOf(v))
}

type assignBody struct {
	TeacherID *string   `json:"teacher_id"`
	TAIDs     *[]string `json:"ta_ids"`
}

type assignOut struct {
	Teacher    *person  `json:"teacher"`
	Assistants []person `json:"assistants"`
	Changed    struct {
		Teacher   bool        `json:"teacher"`
		AddedTA   []uuid.UUID `json:"added_ta"`
		RemovedTA []uuid.UUID `json:"removed_ta"`
	} `json:"changed"`
}

func (h *Handler) doAssign(w http.ResponseWriter, r *http.Request, op string, withTeacher bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var b assignBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	var teacher *uuid.UUID
	if withTeacher && b.TeacherID != nil {
		t, aerr := parseID("teacher_id", *b.TeacherID)
		if aerr != nil {
			apierr.Write(w, r, aerr)
			return
		}
		teacher = &t
	} else if !withTeacher && b.TeacherID != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "teacher_id", Code: "UNKNOWN_FIELD", Message: "Trường không được hỗ trợ."}))
		return
	}
	tas, aerr := parseIDs("ta_ids", b.TAIDs)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	if !withTeacher && tas == nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "ta_ids", Code: "REQUIRED", Message: "Cần danh sách trợ giảng."}))
		return
	}
	res, err := h.Courses.Assign(r.Context(), actor, id, teacher, tas)
	if h.fail(w, r, op, err) {
		return
	}
	out := assignOut{Teacher: personOf(res.Teacher), Assistants: make([]person, len(res.Assistants))}
	for i, a := range res.Assistants {
		out.Assistants[i] = person{ID: a.ID, FullName: a.FullName}
	}
	out.Changed.Teacher = res.Changed.Teacher
	out.Changed.AddedTA = nonNil(res.Changed.AddedTA)
	out.Changed.RemovedTA = nonNil(res.Changed.RemovedTA)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func nonNil(in []uuid.UUID) []uuid.UUID {
	if in == nil {
		return []uuid.UUID{}
	}
	return in
}

func (h *Handler) adminAssign(w http.ResponseWriter, r *http.Request) {
	h.doAssign(w, r, "admin-assign", true)
}
func (h *Handler) putAssistants(w http.ResponseWriter, r *http.Request) {
	h.doAssign(w, r, "assistants", false)
}

type candidate struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

func (h *Handler) candidates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 100 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "q", Code: "TOO_LONG", Message: "Từ khoá tối đa 100 ký tự."}))
		return
	}
	rows, err := h.Courses.AssistantCandidates(r.Context(), q)
	if h.fail(w, r, "assistant-candidates", err) {
		return
	}
	items := make([]candidate, len(rows))
	for i, c := range rows {
		items[i] = candidate{ID: c.ID, FullName: c.FullName, Email: c.Email}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
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

// fail ánh xạ lỗi nghiệp vụ sang mã API; true = đã ghi phản hồi lỗi.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, op string, err error) bool {
	if err == nil {
		return false
	}
	var inv *course.InvalidError
	var vc *course.VersionConflictError
	var jv *course.JoinVersionConflictError
	var rl *course.RateLimitedError
	var ce course.ConfirmError
	switch {
	case errors.As(err, &inv):
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: inv.Field, Code: inv.Code, Message: inv.Message}))
	case errors.Is(err, course.ErrClassCodeTaken):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"field": "class_code"}).WithMessage("Mã lớp này đã có."))
	case errors.Is(err, course.ErrArchived):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.CourseArchived))
	case errors.Is(err, course.ErrNotFound):
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
	case errors.As(err, &vc):
		httpx.WriteVersionConflict(w, r, vc.Current.Version, adminItemOf(vc.Current))
	case errors.As(err, &jv):
		httpx.WriteVersionConflict(w, r, jv.Current.Version, joinInfoOf(jv.Current))
	case errors.Is(err, course.ErrJoinInvalid):
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.JoinCodeInvalid))
	case errors.Is(err, course.ErrCourseFull):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.CourseFull).WithMessage("Lớp đã đủ sĩ số. Hãy báo giảng viên."))
	case errors.Is(err, course.ErrNotVerified):
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.EmailNotVerified))
	case errors.As(err, &rl):
		apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(rl.RetryAfter))
	case errors.Is(err, course.ErrMismatchNeedsTeacher):
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "mismatch_needs_teacher"}))
	case errors.As(err, &ce):
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "confirm_mismatch", Code: "REQUIRED", Message: "Hãy xác nhận email chưa khớp trước khi duyệt."}))
	case errors.Is(err, course.ErrUndoExpired):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"reason": "undo_expired"}).WithMessage("Đã quá thời gian hoàn tác."))
	case errors.Is(err, course.ErrNotPending):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"reason": "not_pending"}).WithMessage("Yêu cầu này không còn chờ duyệt."))
	case errors.Is(err, course.ErrNotActive):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]string{"reason": "not_active"}).WithMessage("Sinh viên này không còn trong lớp."))
	case errors.Is(err, course.ErrStaffMember):
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "user_id", Code: "STAFF_MEMBER", Message: "Giảng viên và trợ giảng đổi bằng cách gán lớp."}))
	case errors.Is(err, course.ErrSourceCourse):
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "source_course"}))
	case errors.Is(err, course.ErrRosterRace):
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.Conflict).WithMessage("Có thay đổi đồng thời. Hãy thử lại."))
	case errors.Is(err, course.ErrForbiddenAction):
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "role"}))
	default:
		h.Log.ErrorContext(r.Context(), "courses lỗi", "op", op, "error", err.Error())
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
	}
	return true
}
