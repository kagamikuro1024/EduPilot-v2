package examhttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// eventsBodyMax: thân `POST …/events` ≤ 64 KiB (lô tối đa 50 sự kiện nhỏ).
const eventsBodyMax = 64 << 10

// MountIntegrity đăng ký thao tác 38 (sinh viên gửi sự kiện), 51 (Giảng viên đọc log), 52–54 (so độ giống) của SRS 6.2, thêm `GET …/similarity/{sid}` (hai mã cạnh nhau — đề xuất #17).
func (h *Handler) MountIntegrity(r chi.Router) {
	student, teacher := h.Guard(auth.StudentRole), h.Guard(auth.TeacherRole)
	const e = "/courses/{id}/exams/{eid}"
	r.With(student).Post(e+"/attempts/{aid}/events", h.postEvents)
	r.With(teacher).Get(e+"/events", h.listEvents)
	r.With(teacher).Get(e+"/similarity", h.listSimilarity)
	r.With(teacher, h.OptIdem).Post(e+"/similarity/run", h.runSimilarity)
	r.With(teacher).Get(e+"/similarity/{sid}", h.getSimilarity)
	r.With(teacher).Put(e+"/similarity/{sid}/review", h.reviewSimilarity)
}

func (h *Handler) postEvents(w http.ResponseWriter, r *http.Request) {
	c, eid, aid, ok := h.attemptIDs(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, eventsBodyMax)
	var in exam.EventsIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	if err := h.Svc.RecordEvents(r.Context(), c.actor, c.course, eid, aid, in); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	aid, err := uuid.Parse(r.URL.Query().Get("attempt"))
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "attempt", Code: "VALUE_REQUIRED", Message: "Cần mã lượt làm (attempt)."}))
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
	page, err := h.Svc.ListEvents(r.Context(), c.course, eid, aid, cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	pg := httpx.Paginate(page.Items, pp, func(x exam.EventView) (time.Time, string) { return x.OccurredAt, x.ID.String() })
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, struct {
		Summary    exam.IntegritySummary `json:"summary"`
		Items      []exam.EventView      `json:"items"`
		NextCursor *string               `json:"next_cursor"`
	}{page.Summary, pg.Items, pg.NextCursor})
}

// MyLock: `GET /me/exam-lock` (JWT, chỉ chính mình) → `{locked, until?}`. KHÔNG lộ tên bài thi / lớp. DB cũng lỗi → 503 (người gọi coi như bị khoá — an toàn khi nghi ngờ).
func MyLock(l *exam.Locker) http.HandlerFunc {
	type out struct {
		Locked bool       `json:"locked"`
		Until  *time.Time `json:"until,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		p := auth.MustFromContext(r.Context())
		uid, err := uuid.Parse(p.Sub)
		if err != nil {
			apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
			return
		}
		lk, locked, err := l.IsLocked(r.Context(), uid)
		if err != nil {
			apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
			return
		}
		noStore(w)
		if !locked {
			httpx.WriteJSON(w, http.StatusOK, out{})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out{Locked: true, Until: &lk.Until})
	}
}

func (h *Handler) listSimilarity(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	pp, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	var run *uuid.UUID
	if v := r.URL.Query().Get("run"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "run", Code: "type", Message: "Mã bản chạy không hợp lệ."}))
			return
		}
		run = &id
	}
	var cur *exam.SimilarityCursor
	if pp.Cursor != nil { // cursor mang điểm (‰) ở trường thời gian: sắp theo (điểm, id) chứ không theo thời gian
		cur = &exam.SimilarityCursor{Score: decimal.New(pp.Cursor.CreatedAt.UnixMicro(), -3), ID: pp.Cursor.ID}
	}
	rows, err := h.Svc.ListSimilarity(r.Context(), c.course, eid, run, r.URL.Query().Get("flagged") == "true", cur, pp.Fetch())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, pp, func(x exam.SimilarityView) (time.Time, string) {
		return time.UnixMicro(x.Score.Shift(3).IntPart()), x.ID.String()
	}))
}

func (h *Handler) runSimilarity(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	id, err := h.Svc.EnqueueSimilarity(r.Context(), c.actor, c.course, eid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"job_id": id})
}

func (h *Handler) getSimilarity(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "sid")
	if !ok {
		return
	}
	d, err := h.Svc.GetSimilarity(r.Context(), c.course, eid, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) reviewSimilarity(w http.ResponseWriter, r *http.Request) {
	c, eid, ok := h.examIDs(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "sid")
	if !ok {
		return
	}
	var in exam.SimilarityReviewIn
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	v, err := h.Svc.ReviewSimilarity(r.Context(), c.actor, c.course, eid, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, v)
}
