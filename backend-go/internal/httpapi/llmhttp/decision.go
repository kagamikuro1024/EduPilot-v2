package llmhttp

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/decision"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// WithDecision gắn dịch vụ cấu hình decision (SRS FEAT-decision-model 6); nil → các route decision không được gắn.
func (a *Admin) WithDecision(s *decision.Service) *Admin {
	a.Decision = s
	return a
}

// mountDecision đăng ký SRS 6 #1–#5 và #7 (ADMIN). #6 (provider type=decision) dùng API nhà cung cấp sẵn có.
func (a *Admin) mountDecision(r chi.Router) {
	if a.Decision == nil {
		return
	}
	admin := adminOnly
	r.With(admin).Get("/admin/llm/decision", a.decisionOverview)
	r.With(admin).Put("/admin/llm/decision/gates/{gate}", a.putDecisionGate)
	r.With(admin).Get("/admin/llm/decision/courses", a.decisionCourses)
	r.With(admin).Put("/courses/{id}/decision-gates/{gate}", a.putCourseDecisionGate)
	r.With(admin).Get("/courses/{id}/decision-gates", a.courseDecisionGates)
	r.With(admin).Get("/admin/observability/decision", a.decisionObservability)
}

func (a *Admin) decisionOverview(w http.ResponseWriter, r *http.Request) {
	o, err := a.Decision.Overview(r.Context())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, o)
}

func (a *Admin) putDecisionGate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled   *bool            `json:"enabled"`
		Threshold *decimal.Decimal `json:"threshold"`
		TimeoutMS *int             `json:"timeout_ms"`
		Version   *int             `json:"version" validate:"omitnil,min=0"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	want, e := httpx.WantedVersion(r, in.Version)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	g, err := a.Decision.PutGate(r.Context(), chi.URLParam(r, "gate"), decision.GateIn{Enabled: in.Enabled, Threshold: in.Threshold, TimeoutMS: in.TimeoutMS}, want)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, g)
}

func (a *Admin) decisionCourses(w http.ResponseWriter, r *http.Request) {
	p, e := httpx.ParsePageParams(r)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	q := r.URL.Query()
	var curT *time.Time
	var curID *uuid.UUID
	if p.Cursor != nil {
		curT, curID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := a.Decision.Courses(r.Context(), q.Get("gate"), q.Get("q"), curT, curID, p.Fetch())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, httpx.Paginate(rows, p, func(c decision.CourseRow) (time.Time, string) { return c.CreatedAt, c.CourseID.String() }))
}

func (a *Admin) putCourseDecisionGate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "id", Code: "uuid", Message: "Mã lớp không hợp lệ."}))
		return
	}
	var in struct {
		Enabled *bool `json:"enabled" validate:"required"`
		Version *int  `json:"version" validate:"omitnil,min=0"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	want, e := httpx.WantedVersion(r, in.Version)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	g, err := a.Decision.PutCourseGate(r.Context(), id, chi.URLParam(r, "gate"), *in.Enabled, want)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, g)
}

func (a *Admin) courseDecisionGates(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "id", Code: "uuid", Message: "Mã lớp không hợp lệ."}))
		return
	}
	gs, err := a.Decision.CourseGates(r.Context(), id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": gs})
}

func (a *Admin) decisionObservability(w http.ResponseWriter, r *http.Request) {
	gs, err := a.Decision.Observability(r.Context(), strings.TrimSpace(r.URL.Query().Get("window")))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"gates": gs})
}
