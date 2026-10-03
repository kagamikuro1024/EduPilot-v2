package llmhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// Admin là API cấu hình LLM (US-P1-04; SRS FEAT-llm-gateway 6.2–6.3): 8 đường dẫn, 13 thao tác.
// Handler mỏng: logic và SQL ở internal/llmconfig; kiểm tra kết nối ở verify.go.
type Admin struct {
	RT    *llmrt.Runtime
	Redis *goredis.Client
	Log   *slog.Logger
	Clock clock.Clock
}

// NewAdmin dựng Admin; clk nil → đồng hồ hệ thống.
func NewAdmin(rt *llmrt.Runtime, rdb *goredis.Client, log *slog.Logger, clk clock.Clock) *Admin {
	if clk == nil {
		clk = clock.Real{}
	}
	return &Admin{RT: rt, Redis: rdb, Log: log, Clock: clk}
}

// Mount đăng ký 13 thao tác trong nhóm đã xác thực (auth.Middleware). RBAC chạy TRƯỚC mọi việc chạm DB hay nhà cung cấp;
// idem là middleware Idempotency-Key (đặt sau RBAC để 403 đến trước 422 thiếu khoá). Đường tĩnh /providers/test đứng trước /{id}.
func (a *Admin) Mount(r chi.Router, idem func(http.Handler) http.Handler) {
	readers := auth.RequireRole(auth.RoleAdmin, auth.RoleTeacher)
	admin := auth.RequireRole(auth.RoleAdmin)
	r.Route("/admin/llm", func(r chi.Router) {
		r.With(readers).Get("/providers", a.listProviders)
		r.With(admin, idem).Post("/providers", a.createProvider)
		r.With(admin).Post("/providers/test", a.testCandidate)
		r.With(admin).Put("/providers/{id}", a.updateProvider)
		r.With(admin).Delete("/providers/{id}", a.deleteProvider)
		r.With(admin).Post("/providers/{id}/test", a.testSaved)
		r.With(readers).Get("/routes", a.listRoutes)
		r.With(admin).Put("/routes", a.putRoute)
		r.With(readers).Get("/usage", a.usage)
		r.With(readers).Get("/budget", a.getSystemBudget)
		r.With(admin).Put("/budget", a.putSystemBudget)
	})
	r.With(admin).Get("/courses/{id}/llm-budget", a.getCourseBudget)
	r.With(admin).Put("/courses/{id}/llm-budget", a.putCourseBudget)
}

// fail đổi lỗi của dịch vụ cấu hình sang lỗi API; lỗi lạ → 500 INTERNAL (chi tiết chỉ vào log, không có khoá / thân nhà cung cấp).
func (a *Admin) fail(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := FromConfig(err); ok {
		apierr.Write(w, r, e)
		return
	}
	if r.Context().Err() != nil {
		apierr.Write(w, r, apierr.New(http.StatusGatewayTimeout, apierr.DeadlineExceeded))
		return
	}
	a.Log.ErrorContext(r.Context(), "API cấu hình LLM lỗi", "error", err.Error(), "path", r.URL.Path)
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func pathID(r *http.Request) (uuid.UUID, error) { return uuid.Parse(chi.URLParam(r, "id")) }

func notFound(w http.ResponseWriter, r *http.Request) {
	apierr.Write(w, r, apierr.ByStatus(http.StatusNotFound))
}

func (a *Admin) circuit(ctx context.Context, id uuid.UUID) string {
	return a.RT.Scheduler.CircuitState(ctx, id.String())
}

func (a *Admin) providerView(ctx context.Context, p llmconfig.Provider) providerView {
	return toProviderView(p, a.circuit(ctx, p.ID))
}

// ---- nhà cung cấp ----

func (a *Admin) listProviders(w http.ResponseWriter, r *http.Request) {
	ps, err := a.RT.Config.ListProviders(r.Context())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	out := providerListView{Items: make([]providerView, 0, len(ps)),
		EnvFallback: envFallbackView{Active: a.RT.Registry.EnvActive(), Providers: a.RT.Registry.EnvProviders()}}
	for _, p := range ps {
		out.Items = append(out.Items, a.providerView(r.Context(), p))
	}
	httpx.WriteJSONETag(w, r, out, "")
}

type modelReq struct {
	Model    string           `json:"model" validate:"required"`
	Kind     string           `json:"kind" validate:"required,oneof=chat embedding"`
	Dims     *int             `json:"dims" validate:"omitnil,min=1"`
	PriceIn  *decimal.Decimal `json:"price_in" validate:"required"`
	PriceOut *decimal.Decimal `json:"price_out" validate:"required"`
	Enabled  *bool            `json:"enabled"`
}

type providerReq struct {
	Type       string            `json:"type" validate:"required,oneof=openai anthropic gemini openai_compatible fake"`
	Name       string            `json:"name" validate:"required"`
	BaseURL    *string           `json:"base_url"`
	APIKey     *llmconfig.Secret `json:"api_key"`
	Enabled    *bool             `json:"enabled"`
	RPMLimit   *int              `json:"rpm_limit"`
	TPMLimit   *int              `json:"tpm_limit"`
	Models     []modelReq        `json:"models" validate:"omitempty,dive"`
	SkipVerify bool              `json:"skip_verify"`
	Version    *int              `json:"version" validate:"omitnil,min=1"`
}

func (q providerReq) input() llmconfig.ProviderInput {
	in := llmconfig.ProviderInput{Type: q.Type, Name: q.Name, BaseURL: q.BaseURL, APIKey: q.APIKey, Enabled: q.Enabled,
		RPMLimit: q.RPMLimit, TPMLimit: q.TPMLimit, SkipVerify: q.SkipVerify}
	if q.Models != nil {
		in.Models = make([]llmconfig.ModelInput, 0, len(q.Models))
		for _, m := range q.Models {
			in.Models = append(in.Models, llmconfig.ModelInput{Model: m.Model, Kind: m.Kind, Dims: m.Dims, PriceIn: *m.PriceIn, PriceOut: *m.PriceOut, Enabled: m.Enabled})
		}
	}
	return in
}

func (a *Admin) createProvider(w http.ResponseWriter, r *http.Request) {
	var req providerReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	in := req.input()
	if err := llmconfig.ValidateInput(in); err != nil {
		a.fail(w, r, err)
		return
	}
	verified := false
	if !in.SkipVerify {
		key := ""
		if in.APIKey != nil {
			key = in.APIKey.Reveal()
		}
		if !a.verifyBeforeSave(w, r, in.Type, in.BaseURL, key, pickInput(in.Models)) {
			return
		}
		verified = true
	}
	p, err := a.RT.Config.CreateProvider(ctx, in)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	p = a.recordVerified(ctx, p, verified)
	w.Header().Set("ETag", httpx.ETagVersion(p.Version))
	httpx.WriteJSON(w, http.StatusCreated, a.providerView(ctx, p))
}

// recordVerified lưu last_test.ok=true sau khi lưu thành công một bản ghi đã qua kiểm tra; lỗi ghi chỉ là log (bản ghi đã lưu).
func (a *Admin) recordVerified(ctx context.Context, p llmconfig.Provider, verified bool) llmconfig.Provider {
	if !verified {
		return p
	}
	if err := a.RT.Config.RecordTest(ctx, p.ID, true, ""); err != nil {
		a.Log.WarnContext(ctx, "không lưu được kết quả kiểm tra kết nối", "error", err.Error())
		return p
	}
	if q, err := a.RT.Config.GetProvider(ctx, p.ID); err == nil {
		return q
	}
	return p
}

func (a *Admin) updateProvider(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		notFound(w, r)
		return
	}
	var req providerReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	want, e := httpx.WantedVersion(r, req.Version)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	ctx := r.Context()
	cur, err := a.RT.Config.GetProvider(ctx, id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if cur.Version != want { // báo xung đột trước khi gọi mạng
		httpx.WriteVersionConflict(w, r, cur.Version, a.providerView(ctx, cur))
		return
	}
	in := req.input()
	in.Version = want
	if err := llmconfig.ValidateInput(in); err != nil {
		a.fail(w, r, err)
		return
	}
	verified := false
	changed := in.APIKey != nil || in.Type != cur.Type || !sameURL(in.BaseURL, cur.BaseURL)
	if changed && !in.SkipVerify {
		key := ""
		if in.APIKey != nil {
			key = in.APIKey.Reveal()
		} else if k, kerr := a.RT.Resolver.DecryptKey(ctx, id); kerr == nil {
			key = k
		}
		cand := pickInput(in.Models)
		if in.Models == nil {
			cand = pickStored(cur.Models)
		}
		if !a.verifyBeforeSave(w, r, in.Type, in.BaseURL, key, cand) {
			return
		}
		verified = true
	}
	p, err := a.RT.Config.UpdateProvider(ctx, id, in)
	var vc *llmconfig.ErrVersionConflict
	if errors.As(err, &vc) { // đổi tay giữa lúc kiểm tra kết nối
		if now, gerr := a.RT.Config.GetProvider(ctx, id); gerr == nil {
			httpx.WriteVersionConflict(w, r, now.Version, a.providerView(ctx, now))
			return
		}
	}
	if err != nil {
		a.fail(w, r, err)
		return
	}
	p = a.recordVerified(ctx, p, verified)
	w.Header().Set("ETag", httpx.ETagVersion(p.Version))
	httpx.WriteJSON(w, http.StatusOK, a.providerView(ctx, p))
}

func sameURL(a, b *string) bool {
	norm := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}
	return norm(a) == norm(b)
}

func (a *Admin) deleteProvider(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		notFound(w, r)
		return
	}
	if err := a.RT.Config.DeleteProvider(r.Context(), id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- tuyến ----

func (a *Admin) listRoutes(w http.ResponseWriter, r *http.Request) {
	rs, err := a.RT.Config.ListRoutes(r.Context())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	out := routesView{Items: make([]routeView, 0, len(rs.Items))}
	for _, it := range rs.Items {
		out.Items = append(out.Items, toRouteView(it))
	}
	if e := rs.Embedding; e != nil {
		out.Embedding = &embeddingView{ModelID: e.ModelID.String(), ProviderName: e.ProviderName, Model: e.Model, Dims: e.Dims}
	}
	httpx.WriteJSONETag(w, r, out, "")
}

func (a *Admin) putRoute(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Task    string           `json:"task" validate:"required,oneof=CHAT CLASSIFY UTILITY GRADING QUESTION_GEN INSIGHT EMBEDDING"`
		Chain   []string         `json:"chain"`
		Params  llmconfig.Params `json:"params"`
		Version *int             `json:"version" validate:"omitnil,min=0"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	want, e := httpx.WantedVersion(r, in.Version)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	chain := make([]uuid.UUID, 0, len(in.Chain))
	for i, s := range in.Chain {
		id, err := uuid.Parse(s)
		if err != nil {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "chain[" + strconv.Itoa(i) + "]", Code: "uuid", Message: "Mã mô hình không hợp lệ."}))
			return
		}
		chain = append(chain, id)
	}
	res, err := a.RT.Config.SetRoute(r.Context(), in.Task, chain, in.Params, want)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETagVersion(res.Route.Version))
	httpx.WriteJSON(w, http.StatusOK, routePutView{routeView: toRouteView(res.Route), ReindexRequired: res.ReindexRequired})
}

// ---- mức dùng ----

const (
	usageDefaultWindow = 7 * 24 * time.Hour
	dateLayout         = "2006-01-02"
)

func parseTime(s string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(dateLayout, s); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func (a *Admin) usage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	to := a.Clock.Now().UTC()
	from := to.Add(-usageDefaultWindow)
	for _, p := range []struct {
		name string
		dst  *time.Time
	}{{"from", &from}, {"to", &to}} {
		if s := q.Get(p.name); s != "" {
			t, ok := parseTime(s)
			if !ok {
				apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: p.name, Code: "format", Message: "Thời điểm phải có dạng RFC 3339 hoặc YYYY-MM-DD."}))
				return
			}
			*p.dst = t
		}
	}
	if q.Get("from") != "" && q.Get("to") == "" {
		to = from.Add(usageDefaultWindow)
	}
	group := q.Get("group")
	if group == "" {
		group = "task"
	}
	var course *uuid.UUID
	if s := q.Get("course_id"); s != "" {
		if p, _ := auth.FromContext(r.Context()); p.Role != auth.RoleAdmin {
			apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "role"}))
			return
		}
		id, err := uuid.Parse(s)
		if err != nil {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "course_id", Code: "uuid", Message: "Mã lớp không hợp lệ."}))
			return
		}
		course = &id
	}
	rows, err := a.RT.Config.Usage(r.Context(), group, from, to, course)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	out := usageView{From: from, To: to, Group: group, Items: make([]usageItemView, 0, len(rows))}
	for _, u := range rows {
		out.Items = append(out.Items, usageItemView{Key: u.Key, Calls: u.Calls, TokensIn: u.TokensIn, TokensOut: u.TokensOut, CostEst: money(u.CostEst),
			LatencyP50Ms: u.LatencyP50Ms, LatencyP95Ms: u.LatencyP95Ms, Errors: u.Errors, Degraded: u.Degraded})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ---- ngân sách ----

func (a *Admin) budgetView(ctx context.Context, b llmconfig.Budget) budgetView {
	v := budgetView{Scope: b.Scope, DailyLimit: limitStr(b.Daily), MonthlyLimit: limitStr(b.Monthly), Version: b.Version,
		SpentToday: money(decimal.Zero), SpentMonth: money(decimal.Zero), State: "ok"}
	if b.CourseID != nil {
		s := b.CourseID.String()
		v.CourseID = &s
	}
	st, err := a.RT.Budget.Status(ctx, b.Scope, b.CourseID, budget.Limit{Daily: b.Daily, Monthly: b.Monthly})
	if err != nil { // Redis mất: hiển thị 0 thay vì lỗi (cấu hình vẫn đọc / sửa được)
		a.Log.WarnContext(ctx, "không đọc được chi phí ngân sách LLM", "error", err.Error())
		return v
	}
	v.SpentToday, v.SpentMonth = money(st.SpentToday), money(st.SpentMonth)
	v.PctToday, v.PctMonth = pctNum(st.PctToday), pctNum(st.PctMonth)
	v.State = stateName(st.State)
	return v
}

func (a *Admin) getSystemBudget(w http.ResponseWriter, r *http.Request) {
	a.getBudget(w, r, llmconfig.ScopeSystem, nil)
}

func (a *Admin) getCourseBudget(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "id", Code: "uuid", Message: "Mã lớp không hợp lệ."}))
		return
	}
	a.getBudget(w, r, llmconfig.ScopeCourse, &id)
}

func (a *Admin) getBudget(w http.ResponseWriter, r *http.Request, scope string, course *uuid.UUID) {
	b, err := a.RT.Config.GetBudget(r.Context(), scope, course)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	httpx.WriteJSONETag(w, r, a.budgetView(r.Context(), b), httpx.ETagVersion(b.Version))
}

func (a *Admin) putSystemBudget(w http.ResponseWriter, r *http.Request) {
	a.putBudget(w, r, llmconfig.ScopeSystem, nil)
}

func (a *Admin) putCourseBudget(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "id", Code: "uuid", Message: "Mã lớp không hợp lệ."}))
		return
	}
	a.putBudget(w, r, llmconfig.ScopeCourse, &id)
}

func (a *Admin) putBudget(w http.ResponseWriter, r *http.Request, scope string, course *uuid.UUID) {
	var in struct {
		Daily   *decimal.Decimal `json:"daily_limit"`
		Monthly *decimal.Decimal `json:"monthly_limit"`
		Version *int             `json:"version" validate:"omitnil,min=0"`
	}
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	want, e := httpx.WantedVersion(r, in.Version)
	if e != nil {
		apierr.Write(w, r, e)
		return
	}
	b, err := a.RT.Config.SetBudget(r.Context(), scope, course, in.Daily, in.Monthly, want)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.RT.Budget.Invalidate() // có hiệu lực ngay cho lời gọi kế tiếp ở tiến trình này; tiến trình khác nhận qua ep:llm:reload
	w.Header().Set("ETag", httpx.ETagVersion(b.Version))
	httpx.WriteJSON(w, http.StatusOK, a.budgetView(r.Context(), b))
}
