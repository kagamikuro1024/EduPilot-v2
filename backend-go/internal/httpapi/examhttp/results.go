package examhttp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// MountResults đăng ký thao tác 26, 42, 44–50, 55, 56 của SRS 6.2 (công bố, kết quả cho Staff, sửa điểm, chấm lại, phúc khảo). Quyền theo `exam.Routes()`.
func (h *Handler) MountResults(r chi.Router) {
	student, staff, teacher := h.Guard(auth.StudentRole), h.Guard(auth.StaffRole), h.Guard(auth.TeacherRole)
	const e = "/courses/{id}/exams/{eid}"
	r.With(teacher).Put(e+"/publish-hold", h.publishHold)
	r.With(student, h.Idem).Post(e+"/attempts/{aid}/appeal", h.createAppeal)
	r.With(staff).Get(e+"/results", h.listResults)
	r.With(staff).Get(e+"/results.csv", h.resultsCSV) // đứng trước `/results/{aid}`: đường tĩnh
	r.With(staff).Get(e+"/results/{aid}", h.resultDetail)
	r.With(teacher).Put(e+"/results/{aid}/score", h.adjustScore)
	r.With(teacher).Put(e+"/items/{itemId}/override", h.overrideItem)
	r.With(teacher, h.Idem).Post(e+"/regrade", h.regrade)
	r.With(staff).Get(e+"/stats", h.examStats)
	r.With(staff).Get(e+"/appeals", h.listAppeals)
	r.With(teacher, h.OptIdem).Post(e+"/appeals/{pid}/answer", h.answerAppeal)
}

type holdIn struct {
	Hold    bool `json:"hold"`
	Version int  `json:"version"`
}

func (h *Handler) publishHold(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	var in holdIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	d, err := h.Svc.SetHold(r.Context(), c.actor, c.course, eid, in.Hold, in.Version)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(d.Version))
	httpx.WriteJSON(w, http.StatusOK, d)
}

type appealIn struct {
	Reason string `json:"reason"`
}

func (h *Handler) createAppeal(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	var in appealIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.CreateAppeal(r.Context(), c.actor, c.course, eid, aid, in.Reason)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusCreated, v)
}

// resultsBody là phản hồi `GET …/results`.
type resultsBody struct {
	Progress   exam.ResultsProgress `json:"progress"`
	Items      []exam.ResultRow     `json:"items"`
	NextCursor *string              `json:"next_cursor"`
}

func (h *Handler) listResults(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	q := r.URL.Query()
	f := exam.ResultsFilter{Status: q.Get("status"), Sort: q.Get("sort"), Q: q.Get("q"), Teacher: c.teacher}
	switch f.Status {
	case "", "ABSENT", "IN_PROGRESS", "GRADING", "GRADED":
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "status", Code: "invalid", Message: "Trạng thái không hợp lệ."}))
		return
	}
	switch f.Sort {
	case "":
		f.Sort = "score"
	case "score", "name":
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "sort", Code: "invalid", Message: "Cách sắp xếp không hợp lệ."}))
		return
	}
	var after *uuid.UUID
	if pp.Cursor != nil {
		after = &pp.Cursor.ID
	}
	page, rows, err := h.Svc.ResultsList(r.Context(), c.course, eid, f, after, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	pg := httpx.Paginate(rows, pp, func(x exam.ResultRow) (time.Time, string) { return time.Time{}, x.Student.ID.String() })
	body, err := json.Marshal(resultsBody{Progress: page.Progress, Items: pg.Items, NextCursor: pg.NextCursor})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	sum := sha256.Sum256(body)
	etag := `W/"` + hex.EncodeToString(sum[:12]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *Handler) resultDetail(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.ResultDetail(r.Context(), c.course, eid, aid, c.teacher)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) adjustScore(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	var in exam.AdjustIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.AdjustScore(r.Context(), c.actor, c.course, eid, aid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) overrideItem(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	iid, ok := pathUUID(w, r, "itemId")
	if !ok {
		return
	}
	var in exam.OverrideIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	out, err := h.Svc.OverrideItem(r.Context(), c.actor, c.course, eid, iid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	if out.JobID != nil {
		httpx.WriteJSON(w, http.StatusAccepted, out)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) regrade(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	var in exam.RegradeIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	id, err := h.Svc.Regrade(r.Context(), c.actor, c.course, eid, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"job_id": id})
}

func (h *Handler) examStats(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	st, err := h.Svc.ExamStats(r.Context(), c.course, eid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) resultsCSV(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	err := h.Svc.ResultsCSV(r.Context(), c.course, eid, w, func() {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ket-qua-bai-thi.csv"`)
		noStore(w)
	})
	if err != nil {
		// Trước byte đầu: JSON lỗi như mọi route; sau byte đầu: kết nối bị ngắt (khách thấy tệp thiếu, không có nửa dòng).
		if w.Header().Get("Content-Type") == "text/csv; charset=utf-8" {
			h.Log.WarnContext(r.Context(), "exam: xuất CSV dừng giữa chừng", "error", err.Error())
			return
		}
		h.fail(w, r, err)
	}
}

func (h *Handler) listAppeals(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "", "OPEN", "UPHELD", "ADJUSTED":
	default:
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "status", Code: "invalid", Message: "Trạng thái không hợp lệ."}))
		return
	}
	var cur *exam.Cursor
	if pp.Cursor != nil {
		cur = &exam.Cursor{At: pp.Cursor.CreatedAt, ID: pp.Cursor.ID}
	}
	rows, err := h.Svc.ListAppeals(r.Context(), c.course, eid, status, cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pp, func(x exam.AppealView) (time.Time, string) { return x.CreatedAt, x.ID.String() }))
}

func (h *Handler) answerAppeal(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var in exam.AppealAnswerIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.AnswerAppeal(r.Context(), c.actor, c.course, eid, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}
