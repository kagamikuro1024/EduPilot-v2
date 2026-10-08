// Package examhttp: các đường REST của thi hằng tuần. Handler mỏng; nghiệp vụ ở internal/exam. US-PE-03: ngân hàng câu hỏi (thao tác 1–16 của SRS 6.2).
package examhttp

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler là nhóm đường ngân hàng câu hỏi. Quyền theo `exam.Routes()` (mọi thao tác 1–16: Staff — ADMIN và sinh viên bị chặn ở guard).
type Handler struct {
	Svc     *exam.Service
	Guard   func(auth.GuardMode) func(http.Handler) http.Handler
	Idem    func(http.Handler) http.Handler
	OptIdem func(http.Handler) http.Handler
	// ZipMaxBytes: EXAM_TESTZIP_MAX_BYTES.
	ZipMaxBytes int
	Log         *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware (cả bài thi, MountExams). Đường tĩnh `/questions/suggest` đứng trước `/questions/{qid}`.
func (h *Handler) Mount(r chi.Router) {
	staff := h.Guard(auth.StaffRole)
	const q = "/courses/{id}/questions"
	r.With(staff).Get(q, h.list)
	r.With(staff, h.OptIdem).Post(q, h.create)
	r.With(staff, h.Idem).Post(q+"/suggest", h.suggest)
	r.With(staff).Get(q+"/{qid}", h.get)
	r.With(staff).Put(q+"/{qid}", h.update)
	r.With(staff).Post(q+"/{qid}/archive", h.archive)
	r.With(staff).Post(q+"/{qid}/duplicate", h.duplicate)
	r.With(staff).Put(q+"/{qid}/review", h.review)
	r.With(staff).Put(q+"/{qid}/code", h.putCode)
	r.With(staff).Get(q+"/{qid}/testcases", h.listTests)
	r.With(staff).Post(q+"/{qid}/testcases", h.addTest)
	r.With(staff).Post(q+"/{qid}/testcases/import", h.importTests)
	r.With(staff).Post(q+"/{qid}/testcases/approve", h.approveTests)
	r.With(staff).Put(q+"/{qid}/testcases/{tid}", h.updateTest)
	r.With(staff).Delete(q+"/{qid}/testcases/{tid}", h.deleteTest)
	r.With(staff, h.Idem).Post(q+"/{qid}/reference/verify", h.verify)
	h.MountExams(r)
}

type reqCtx struct {
	course, question, actor uuid.UUID
	teacher                 bool
}

// ids đọc id lớp / câu từ đường dẫn và người gọi từ phiên; id sai định dạng → 404 (không lộ tồn tại).
func (h *Handler) ids(w http.ResponseWriter, r *http.Request, withQuestion bool) (reqCtx, bool) {
	var c reqCtx
	p := auth.MustFromContext(r.Context())
	var err error
	if c.actor, err = uuid.Parse(p.Sub); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return c, false
	}
	if c.course, err = uuid.Parse(chi.URLParam(r, "id")); err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return c, false
	}
	if withQuestion {
		if c.question, err = uuid.Parse(chi.URLParam(r, "qid")); err != nil {
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return c, false
		}
	}
	access, _ := auth.CourseFromContext(r.Context())
	c.teacher = access.CourseRole == auth.RoleTeacher
	return c, true
}

// fail ghi lỗi của service: *apierr.Error nguyên văn, sai version → 409 kèm bản hiện tại, còn lại 500 (không lộ chi tiết).
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	var vc *exam.VersionConflict
	switch {
	case errors.As(err, &ae):
		apierr.Write(w, r, ae)
	case errors.As(err, &vc):
		httpx.WriteVersionConflict(w, r, vc.Version, vc.Current)
	default:
		h.Log.ErrorContext(r.Context(), "exam: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
		apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
	}
}

// questionOut = chi tiết câu + `details.warnings` (cảnh báo không chặn).
type questionOut struct {
	exam.QuestionDetail
	Details *struct {
		Warnings []apierr.FieldError `json:"warnings"`
	} `json:"details,omitempty"`
}

func out(d exam.QuestionDetail) questionOut {
	o := questionOut{QuestionDetail: d}
	if len(d.Warnings) > 0 {
		o.Details = &struct {
			Warnings []apierr.FieldError `json:"warnings"`
		}{Warnings: d.Warnings}
	}
	return o
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	qs := r.URL.Query()
	f := exam.ListFilter{ReviewStatus: qs.Get("review_status"), Topic: qs.Get("topic"), Difficulty: qs.Get("difficulty"), Type: qs.Get("type"), Origin: qs.Get("origin"), Q: qs.Get("q")}
	var errs []apierr.FieldError
	enum := func(field, v string, allowed ...string) {
		if v == "" {
			return
		}
		for _, a := range allowed {
			if v == a {
				return
			}
		}
		errs = append(errs, apierr.FieldError{Field: field, Code: "enum", Message: "Giá trị lọc không hợp lệ."})
	}
	enum("review_status", f.ReviewStatus, "DRAFT", "PENDING", "APPROVED", "REJECTED")
	enum("difficulty", f.Difficulty, "EASY", "MEDIUM", "HARD")
	enum("type", f.Type, "MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE", "CODE", "SHORT", "ESSAY")
	enum("origin", f.Origin, "MANUAL", "AI_DRAFT", "EXTRACTED", "GENERATED")
	if utf8.RuneCountInString(f.Q) > 100 {
		errs = append(errs, apierr.FieldError{Field: "q", Code: "max", Message: "Từ khoá tối đa 100 ký tự."})
	}
	if v := qs.Get("archived"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, apierr.FieldError{Field: "archived", Code: "bool", Message: "archived là true hoặc false."})
		}
		f.Archived = b
	}
	if len(errs) > 0 {
		apierr.Write(w, r, apierr.Validation(errs...))
		return
	}
	var cur *exam.Cursor
	if pp.Cursor != nil {
		cur = &exam.Cursor{At: pp.Cursor.CreatedAt, ID: pp.Cursor.ID}
	}
	rows, err := h.Svc.List(r.Context(), c.course, f, cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pp, func(x exam.ListItem) (time.Time, string) { return x.CreatedAt, x.ID.String() }))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	var in exam.QuestionIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.Svc.Create(r.Context(), c.actor, c.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out(d))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	d, err := h.Svc.Get(r.Context(), c.course, c.question)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	var in exam.QuestionIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	ver, aerr := httpx.WantedVersion(r, in.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	d, err := h.Svc.Update(r.Context(), c.course, c.question, in, ver)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, out(d))
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	d, err := h.Svc.Archive(r.Context(), c.actor, c.course, c.question)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) duplicate(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	d, err := h.Svc.Duplicate(r.Context(), c.actor, c.course, c.question)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

type reviewIn struct {
	Decision string `json:"decision" validate:"required"`
	Version  *int   `json:"version"`
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	var in reviewIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	ver, aerr := httpx.WantedVersion(r, in.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	d, err := h.Svc.Review(r.Context(), c.actor, c.course, c.question, in.Decision, ver)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) putCode(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	var in exam.CodeIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	ver, aerr := httpx.WantedVersion(r, in.Version)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	d, err := h.Svc.PutCode(r.Context(), c.course, c.question, in, ver)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) listTests(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var after *int
	if pp.Cursor != nil { // con trỏ của danh sách test mang `position` (giây của mốc thời gian) thay cho created_at
		p := int(pp.Cursor.CreatedAt.Unix())
		after = &p
	}
	rows, err := h.Svc.ListTests(r.Context(), c.course, c.question, after, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pp, func(t exam.Testcase) (time.Time, string) { return time.Unix(int64(t.Position), 0).UTC(), t.ID.String() }))
}

func (h *Handler) addTest(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	var in exam.TestcaseIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	t, err := h.Svc.AddTest(r.Context(), c.actor, c.course, c.question, in, c.teacher)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTest(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	tid, err := uuid.Parse(chi.URLParam(r, "tid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var in exam.TestcaseIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	t, err := h.Svc.UpdateTest(r.Context(), c.actor, c.course, c.question, tid, in, c.teacher)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTest(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	tid, err := uuid.Parse(chi.URLParam(r, "tid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	if err := h.Svc.DeleteTest(r.Context(), c.actor, c.course, c.question, tid, c.teacher); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type approveIn struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100"`
}

func (h *Handler) approveTests(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	var in approveIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	n, err := h.Svc.ApproveTests(r.Context(), c.actor, c.course, c.question, in.IDs, c.teacher)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"approved": n})
}

// importTests: `multipart/form-data` trường `file`, `?dry_run=true`, `?mode=append|replace`. Zip quá EXAM_TESTZIP_MAX_BYTES → 413.
func (h *Handler) importTests(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	qs := r.URL.Query()
	dry, err := strconv.ParseBool(orDefault(qs.Get("dry_run"), "false"))
	mode := orDefault(qs.Get("mode"), "append")
	if err != nil || (mode != "append" && mode != "replace") {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "mode", Code: "enum", Message: "dry_run là true/false; mode là append hoặc replace."}))
		return
	}
	limit := h.ZipMaxBytes
	if limit <= 0 {
		limit = 10 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(limit)+(1<<20)) // chừa chỗ cho phần mô tả multipart
	file, _, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			apierr.Write(w, r, exam.ZipTooLarge(limit))
			return
		}
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "file", Code: "required", Message: "Cần tải lên tệp zip ở trường file."}))
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		return
	}
	if len(data) > limit {
		apierr.Write(w, r, exam.ZipTooLarge(limit))
		return
	}
	rep, err := h.Svc.ImportZip(r.Context(), c.actor, c.course, c.question, data, dry, mode == "replace", c.teacher)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	status := http.StatusCreated
	if dry {
		status = http.StatusOK
	}
	httpx.WriteJSON(w, status, rep)
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, true)
	if !ok {
		return
	}
	id, err := h.Svc.EnqueueVerify(r.Context(), c.actor, c.course, c.question)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]uuid.UUID{"job_id": id})
}

func (h *Handler) suggest(w http.ResponseWriter, r *http.Request) {
	c, ok := h.ids(w, r, false)
	if !ok {
		return
	}
	var in exam.SuggestIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	id, err := h.Svc.EnqueueSuggest(r.Context(), c.actor, c.course, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]uuid.UUID{"job_id": id})
}
