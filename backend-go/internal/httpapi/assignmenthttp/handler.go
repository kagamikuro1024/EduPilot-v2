// Package assignmenthttp: route bài tập, nộp bài, chấm, phúc khảo (FEAT-assignments SRS 6). Handler mỏng; nghiệp vụ ở internal/assignment, internal/submission, internal/grading.
package assignmenthttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/assignment"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/submission"
)

// Handler gắn các route. ADMIN không bao giờ qua (các chế độ *Role): quản trị viên không đọc bài / điểm.
type Handler struct {
	Assignments *assignment.Service
	Submit      *submission.Service
	Batches     *grading.Batches
	Guard       func(auth.GuardMode) func(http.Handler) http.Handler
	Idem        func(http.Handler) http.Handler // `Idempotency-Key` bắt buộc
	Log         *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	member, staff, teacher, student := h.Guard(auth.MemberRole), h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole), h.Guard(auth.StudentRole)
	r.With(member).Get("/courses/{id}/assignments", h.list)
	r.With(teacher, h.Idem).Post("/courses/{id}/assignments", h.create)
	r.With(member).Get("/courses/{id}/assignments/{aid}", h.get)
	r.With(teacher).Put("/courses/{id}/assignments/{aid}", h.update)
	r.With(teacher).Delete("/courses/{id}/assignments/{aid}", h.remove)
	r.With(teacher, h.Idem).Post("/courses/{id}/assignments/{aid}/publish", h.publish)
	r.With(teacher).Put("/courses/{id}/assignments/{aid}/rubric", h.putRubric)
	r.With(staff, h.Idem).Post("/courses/{id}/assignments/{aid}/rubric/suggest", h.suggest)
	// Route của nhân viên về bài nộp (US-P7-04): tải tệp lẻ / ZIP, tab `Chưa khớp`, gán tay, bỏ qua, tải tệp.
	r.With(staff, h.Idem).Post("/courses/{id}/assignments/{aid}/submissions/upload", h.upload)
	r.With(staff).Get("/courses/{id}/submissions", h.listSubmissions)
	r.With(staff).Put("/courses/{id}/submissions/{sid}/match", h.match)
	r.With(staff).Get("/courses/{id}/submissions/{sid}/file", h.file)
	r.With(staff).Post("/courses/{id}/submissions/{sid}/dismiss", h.dismiss)
	r.With(staff).Delete("/courses/{id}/submissions/{sid}/dismiss", h.undismiss)
	// Hàng đợi chấm (US-P7-07): nạp lô, chấm / chấm lại một bài, thử lại bài lỗi.
	r.With(staff, h.Idem).Post("/courses/{id}/assignments/{aid}/grade-all", h.gradeAll)
	r.With(staff, h.Idem).Post("/courses/{id}/submissions/{sid}/grade", h.gradeOne)
	r.With(staff, h.Idem).Post("/courses/{id}/grading-results/{rid}/retry", h.retryResult)
	// Route của sinh viên (US-P7-03): chỉ STUDENT của lớp (TA / TEACHER / ADMIN 403), mọi phản hồi `Cache-Control: no-store`.
	r.With(student).Get("/courses/{id}/assignments/{aid}/mine", h.mine)
	r.With(student).Post("/courses/{id}/assignments/{aid}/submit/presign", h.presign)
	r.With(student, h.Idem).Post("/courses/{id}/assignments/{aid}/submit", h.submit)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	if h.Log != nil {
		h.Log.ErrorContext(r.Context(), "assignment: lỗi nội bộ", "error", err.Error())
	}
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func ids(w http.ResponseWriter, r *http.Request) (course, id uuid.UUID, a assignment.Actor, ok bool) {
	course, e1 := uuid.Parse(chi.URLParam(r, "id"))
	p, hasP := auth.FromContext(r.Context())
	uid, e3 := uuid.Parse(p.Sub)
	if e1 != nil || !hasP || e3 != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return course, id, a, false
	}
	if raw := chi.URLParam(r, "aid"); raw != "" {
		var err error
		if id, err = uuid.Parse(raw); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return course, id, a, false
		}
	}
	ca, _ := auth.CourseFromContext(r.Context())
	a = assignment.Actor{UserID: uid, Student: ca.CourseRole == auth.RoleStudent, TraceID: llm.TraceID(r.Context())}
	return course, id, a, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	course, _, a, ok := ids(w, r)
	if !ok {
		return
	}
	pp, perr := httpx.ParsePageParams(r)
	if perr != nil {
		apierr.Write(w, r, perr)
		return
	}
	page, err := h.Assignments.List(r.Context(), a, course, pp)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if a.Student {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": page.StudentItems, "next_cursor": page.NextCursor})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	course, _, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in assignment.Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Assignments.Create(r.Context(), a, course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.Get(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if a.Student {
		httpx.WriteJSON(w, http.StatusOK, assignment.StudentOf(out))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in assignment.Input
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.Version < 1 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "version", Code: "required", Message: "Thiếu version."}))
		return
	}
	out, err := h.Assignments.Update(r.Context(), a, course, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	if err := h.Assignments.Delete(r.Context(), a, course, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.Publish(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) putRubric(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in struct {
		Version  int                   `json:"version"`
		Criteria []assignment.RubricIn `json:"criteria"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.Version < 1 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "version", Code: "required", Message: "Thiếu version."}))
		return
	}
	out, err := h.Assignments.SetRubric(r.Context(), a, course, id, in.Criteria, in.Version)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) suggest(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.SuggestRubric(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"criteria": out})
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store") // mọi phản hồi (kể cả lỗi) của route sinh viên: TLR-17
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	out, err := h.Assignments.Mine(r.Context(), a, course, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store") // mọi phản hồi (kể cả lỗi) của route sinh viên: TLR-17
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in submission.PresignIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Submit.Presign(r.Context(), a.UserID, course, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store") // mọi phản hồi (kể cả lỗi) của route sinh viên: TLR-17
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in struct {
		UploadID uuid.UUID `json:"upload_id"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.UploadID == uuid.Nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "upload_id", Code: "required", Message: "Thiếu mã tải lên."}))
		return
	}
	out, err := h.Submit.Complete(r.Context(), a.UserID, course, id, in.UploadID, a.TraceID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"submission": out})
}

func staffOf(w http.ResponseWriter, r *http.Request) (course, sid uuid.UUID, st submission.Staff, ok bool) {
	course, e1 := uuid.Parse(chi.URLParam(r, "id"))
	p, hasP := auth.FromContext(r.Context())
	uid, e2 := uuid.Parse(p.Sub)
	if e1 != nil || !hasP || e2 != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return course, sid, st, false
	}
	if raw := chi.URLParam(r, "sid"); raw != "" {
		var err error
		if sid, err = uuid.Parse(raw); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return course, sid, st, false
		}
	}
	ca, _ := auth.CourseFromContext(r.Context())
	return course, sid, submission.Staff{UserID: uid, IsTA: ca.CourseRole == auth.RoleTA, TraceID: llm.TraceID(r.Context())}, true
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in struct {
		UploadID uuid.UUID `json:"upload_id"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if in.UploadID == uuid.Nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "upload_id", Code: "required", Message: "Thiếu mã tải lên."}))
		return
	}
	out, job, err := h.Submit.UploadComplete(r.Context(), a.UserID, course, id, in.UploadID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if job != nil {
		httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"job_id": job})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	course, _, _, ok := staffOf(w, r)
	if !ok {
		return
	}
	if r.URL.Query().Get("match_status") != "UNMATCHED" {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "match_status", Code: "enum", Message: "match_status là UNMATCHED."}))
		return
	}
	pp, perr := httpx.ParsePageParams(r)
	if perr != nil {
		apierr.Write(w, r, perr)
		return
	}
	out, err := h.Submit.ListUnmatched(r.Context(), course, pp)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) match(w http.ResponseWriter, r *http.Request) {
	course, sid, st, ok := staffOf(w, r)
	if !ok {
		return
	}
	var in submission.MatchIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Submit.Match(r.Context(), st, course, sid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	course, sid, _, ok := staffOf(w, r)
	if !ok {
		return
	}
	u, err := h.Submit.FileURL(r.Context(), course, sid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"url": u, "expires_in": 300})
}

func (h *Handler) dismiss(w http.ResponseWriter, r *http.Request) {
	course, sid, st, ok := staffOf(w, r)
	if !ok {
		return
	}
	var in struct {
		Reason  string `json:"reason"`
		Version int64  `json:"version"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if err := h.Submit.Dismiss(r.Context(), st, course, sid, in.Reason, in.Version); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) undismiss(w http.ResponseWriter, r *http.Request) {
	course, sid, st, ok := staffOf(w, r)
	if !ok {
		return
	}
	if err := h.Submit.Undismiss(r.Context(), st, course, sid); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) enqueued(w http.ResponseWriter, e grading.Enqueued, err error, r *http.Request) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if e.JobID == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"queued": 0})
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"job_id": e.JobID, "queued": e.Queued})
}

func (h *Handler) gradeAll(w http.ResponseWriter, r *http.Request) {
	course, id, a, ok := ids(w, r)
	if !ok {
		return
	}
	var in struct {
		OnlyUnscored *bool `json:"only_unscored"`
		RetryFailed  bool  `json:"retry_failed"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&in); err != nil && !errors.Is(err, io.EOF) {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "body", Code: "invalid", Message: "Nội dung không hợp lệ."}))
		return
	}
	e, err := h.Batches.GradeAll(r.Context(), course, id, a.UserID, in.RetryFailed && in.OnlyUnscored != nil && !*in.OnlyUnscored)
	h.enqueued(w, e, err, r)
}

func (h *Handler) gradeOne(w http.ResponseWriter, r *http.Request) {
	course, sid, st, ok := staffOf(w, r)
	if !ok {
		return
	}
	e, err := h.Batches.GradeOne(r.Context(), course, sid, st.UserID)
	h.enqueued(w, e, err, r)
}

func (h *Handler) retryResult(w http.ResponseWriter, r *http.Request) {
	course, _, st, ok := staffOf(w, r)
	if !ok {
		return
	}
	rid, err := uuid.Parse(chi.URLParam(r, "rid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	e, err := h.Batches.Retry(r.Context(), course, rid, st.UserID)
	h.enqueued(w, e, err, r)
}
