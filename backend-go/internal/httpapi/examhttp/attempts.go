package examhttp

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// answersMaxBytes: thân `PUT …/answers` ≤ 64 KiB (SRS 4.3.3).
const answersMaxBytes = 64 << 10

// MountAttempts đăng ký thao tác 29, 30, 31, 39, 40, 41 của SRS 6.2 (lượt làm của sinh viên). Quyền theo `exam.Routes()`: Member + STUDENT (Staff → 403 reason=role).
// #32–#37 (bài code) ở `attempt_code.go` (US-PE-06); #38, #42 (sự kiện, phúc khảo) thuộc US-PE-07…08.
func (h *Handler) MountAttempts(r chi.Router) {
	student := h.Guard(auth.StudentRole)
	const e = "/courses/{id}/exams/{eid}"
	r.With(student, h.Idem).Post(e+"/attempts", h.startAttempt)
	r.With(student).Get(e+"/attempts/mine", h.myAttempt)
	r.With(student).Put(e+"/attempts/{aid}/answers", h.saveAnswers)
	r.With(student).Post(e+"/attempts/{aid}/takeover", h.takeover)
	r.With(student, h.Idem).Post(e+"/attempts/{aid}/submit", h.submitAttempt)
	r.With(student).Get(e+"/attempts/{aid}/result", h.attemptResult)
}

// tabID đọc `X-Exam-Tab` (uuid sinh mỗi lần tải trang). Thiếu khi bắt buộc → 422; sai định dạng → 422.
func tabID(w http.ResponseWriter, r *http.Request, required bool) (uuid.UUID, bool) {
	raw := r.Header.Get("X-Exam-Tab")
	if raw == "" && !required {
		return uuid.Nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "X-Exam-Tab", Code: "VALUE_REQUIRED", Message: "Thiếu mã tab (X-Exam-Tab, uuid)."}))
		return uuid.Nil, false
	}
	return id, true
}

// attemptIDs đọc id lớp, bài, lượt từ đường dẫn.
func (h *Handler) attemptIDs(w http.ResponseWriter, r *http.Request) (reqCtx, uuid.UUID, uuid.UUID, bool) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return c, uuid.Nil, uuid.Nil, false
	}
	aid, err := uuid.Parse(chi.URLParam(r, "aid"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return c, uuid.Nil, uuid.Nil, false
	}
	return c, eid, aid, true
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

func (h *Handler) startAttempt(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	tab, ok := tabID(w, r, false)
	if !ok {
		return
	}
	v, created, err := h.Svc.StartAttempt(r.Context(), c.actor, c.course, eid, tab)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, v)
}

func (h *Handler) myAttempt(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	tab, ok := tabID(w, r, false)
	if !ok {
		return
	}
	v, err := h.Svc.MyAttempt(r.Context(), c.actor, c.course, eid, tab)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) saveAnswers(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	tab, ok := tabID(w, r, true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, answersMaxBytes)
	var in exam.AnswersIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.SaveAnswers(r.Context(), c.actor, c.course, eid, aid, tab, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

type takeoverIn struct {
	Reload bool `json:"reload"`
}

func (h *Handler) takeover(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	tab, ok := tabID(w, r, true)
	if !ok {
		return
	}
	var in takeoverIn
	if r.ContentLength != 0 { // thân tuỳ chọn: {"reload":true} sau khi tải lại trang
		if b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<10)); len(b) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(b))
			if !httpx.DecodeJSON(w, r, &in) {
				return
			}
		}
	}
	v, err := h.Svc.Takeover(r.Context(), c.actor, c.course, eid, aid, tab, in.Reload)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) submitAttempt(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	tab, ok := tabID(w, r, true)
	if !ok {
		return
	}
	v, err := h.Svc.SubmitAttempt(r.Context(), c.actor, c.course, eid, aid, tab)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) attemptResult(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	v, err := h.Svc.AttemptResult(r.Context(), c.actor, c.course, eid, aid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}
