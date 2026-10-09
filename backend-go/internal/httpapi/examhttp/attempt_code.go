package examhttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// codeBodyMax: thân `draft` / `run` / `submit` ≤ 64 KiB mã + phần bọc JSON (escape có thể nở tới 6 lần với ký tự điều khiển; chặn cứng ở 512 KiB, kiểm 64 KiB ở service).
const codeBodyMax = 512 << 10

// MountAttemptCode đăng ký thao tác 32–37 của SRS 6.2 (bài code của sinh viên trong lượt làm). Quyền: Member + STUDENT.
func (h *Handler) MountAttemptCode(r chi.Router) {
	student := h.Guard(auth.StudentRole)
	const a = "/courses/{id}/exams/{eid}/attempts/{aid}"
	r.With(student).Put(a+"/code/{itemId}/draft", h.saveDraft)
	r.With(student, h.Idem).Post(a+"/code/{itemId}/run", h.runCode)
	r.With(student).Get(a+"/runs/{runId}", h.getRun)
	r.With(student, h.Idem).Post(a+"/code/{itemId}/submit", h.submitCode)
	r.With(student).Get(a+"/code/{itemId}/submissions", h.listSubmissions)
	r.With(student).Get(a+"/submissions/{sid}", h.getSubmission)
}

func notFoundErr(w http.ResponseWriter, r *http.Request) {
	apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
}

// pathUUID đọc một id từ đường dẫn; sai định dạng → 404 (không lộ có / không).
func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		notFoundErr(w, r)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) saveDraft(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	item, ok := pathUUID(w, r, "itemId")
	if !ok {
		return
	}
	tab, ok := tabID(w, r, true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, codeBodyMax)
	var in exam.DraftIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.SaveDraft(r.Context(), c.actor, c.course, eid, aid, item, tab, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) runCode(w http.ResponseWriter, r *http.Request) {
	h.codeAction(w, r, func(c reqCtx, eid, aid, item, tab uuid.UUID, in exam.CodeRunIn) (any, error) {
		return h.Svc.RunCode(r.Context(), c.actor, c.course, eid, aid, item, tab, in)
	})
}

func (h *Handler) submitCode(w http.ResponseWriter, r *http.Request) {
	h.codeAction(w, r, func(c reqCtx, eid, aid, item, tab uuid.UUID, in exam.CodeRunIn) (any, error) {
		return h.Svc.SubmitCode(r.Context(), c.actor, c.course, eid, aid, item, tab, in)
	})
}

// codeAction là phần chung của `run` / `submit`: đọc id, `X-Exam-Tab`, thân; trả 202.
func (h *Handler) codeAction(w http.ResponseWriter, r *http.Request, do func(c reqCtx, eid, aid, item, tab uuid.UUID, in exam.CodeRunIn) (any, error)) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	item, ok := pathUUID(w, r, "itemId")
	if !ok {
		return
	}
	tab, ok := tabID(w, r, true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, codeBodyMax)
	var in exam.CodeRunIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := do(c, eid, aid, item, tab, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusAccepted, v)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "runId")
	if !ok {
		return
	}
	v, err := h.Svc.GetRun(r.Context(), c.actor, c.course, eid, aid, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) getSubmission(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "sid")
	if !ok {
		return
	}
	v, err := h.Svc.GetSubmission(r.Context(), c.actor, c.course, eid, aid, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	item, ok := pathUUID(w, r, "itemId")
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var cur *exam.Cursor
	if pp.Cursor != nil {
		cur = &exam.Cursor{At: pp.Cursor.CreatedAt, ID: pp.Cursor.ID}
	}
	rows, err := h.Svc.ListSubmissions(r.Context(), c.actor, c.course, eid, aid, item, cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pp, func(x exam.SubmissionView) (time.Time, string) { return x.CreatedAt, x.ID.String() }))
}
