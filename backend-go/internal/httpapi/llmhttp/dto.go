package llmhttp

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llmconfig"
)

// Thân phản hồi theo SRS FEAT-llm-gateway 6.3. Tiền là CHUỖI thập phân (không float); khoá API không bao giờ xuất hiện.

type modelView struct {
	ID       string `json:"id"`
	Model    string `json:"model"`
	Kind     string `json:"kind"`
	Dims     *int   `json:"dims"`
	PriceIn  string `json:"price_in"`
	PriceOut string `json:"price_out"`
	Enabled  bool   `json:"enabled"`
}

type lastTestView struct {
	OK        *bool      `json:"ok"`
	At        *time.Time `json:"at"`
	ErrorKind *string    `json:"error_kind"`
}

type providerView struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	BaseURL   *string      `json:"base_url"`
	Enabled   bool         `json:"enabled"`
	HasKey    bool         `json:"has_key"`
	KeyStatus string       `json:"key_status"`
	RPMLimit  *int         `json:"rpm_limit"`
	TPMLimit  *int         `json:"tpm_limit"`
	LastTest  lastTestView `json:"last_test"`
	Circuit   string       `json:"circuit"`
	Models    []modelView  `json:"models"`
	Version   int          `json:"version"`
}

type envFallbackView struct {
	Active    bool     `json:"active"`
	Providers []string `json:"providers"`
}

type providerListView struct {
	Items       []providerView  `json:"items"`
	EnvFallback envFallbackView `json:"env_fallback"`
}

// money: giá / chi phí giữ 4 chữ số thập phân (đ / 1 triệu token, 1/10.000 đ).
func money(d decimal.Decimal) string { return d.StringFixed(4) }

func toProviderView(p llmconfig.Provider, circuit string) providerView {
	v := providerView{
		ID: p.ID.String(), Type: p.Type, Name: p.Name, BaseURL: p.BaseURL, Enabled: p.Enabled, HasKey: p.HasKey,
		KeyStatus: string(p.KeyStatus), RPMLimit: p.RPMLimit, TPMLimit: p.TPMLimit, Circuit: circuit, Version: p.Version,
		Models: make([]modelView, 0, len(p.Models)),
	}
	if t := p.LastTest; t != nil {
		ok, at := t.OK, t.At
		v.LastTest = lastTestView{OK: &ok, At: &at}
		if t.ErrorKind != "" {
			k := t.ErrorKind
			v.LastTest.ErrorKind = &k
		}
	}
	for _, m := range p.Models {
		v.Models = append(v.Models, modelView{ID: m.ID.String(), Model: m.Model, Kind: m.Kind, Dims: m.Dims,
			PriceIn: money(m.PriceIn), PriceOut: money(m.PriceOut), Enabled: m.Enabled})
	}
	return v
}

type chainItemView struct {
	ModelID      string `json:"model_id"`
	ProviderID   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	Model        string `json:"model"`
}

type routeView struct {
	Task    string           `json:"task"`
	Lane    string           `json:"lane"`
	Chain   []chainItemView  `json:"chain"`
	Params  llmconfig.Params `json:"params"`
	Version int              `json:"version"`
}

type routePutView struct {
	routeView
	ReindexRequired bool `json:"reindex_required"`
}

type embeddingView struct {
	ModelID         string `json:"model_id"`
	ProviderName    string `json:"provider_name"`
	Model           string `json:"model"`
	Dims            int    `json:"dims"`
	ReindexRequired bool   `json:"reindex_required"`
	IndexedChunks   *int   `json:"indexed_chunks"`
}

type routesView struct {
	Items     []routeView    `json:"items"`
	Embedding *embeddingView `json:"embedding"`
}

func toRouteView(r llmconfig.RouteView) routeView {
	v := routeView{Task: r.Task, Lane: r.Lane, Params: r.Params, Version: r.Version, Chain: make([]chainItemView, 0, len(r.Chain))}
	if v.Params == nil {
		v.Params = llmconfig.Params{}
	}
	for _, c := range r.Chain {
		v.Chain = append(v.Chain, chainItemView{ModelID: c.ModelID.String(), ProviderID: c.ProviderID.String(), ProviderName: c.ProviderName, Model: c.Model})
	}
	return v
}

type usageItemView struct {
	Key          string `json:"key"`
	Calls        int64  `json:"calls"`
	TokensIn     int64  `json:"tokens_in"`
	TokensOut    int64  `json:"tokens_out"`
	CostEst      string `json:"cost_est"`
	LatencyP50Ms int64  `json:"latency_p50_ms"`
	LatencyP95Ms int64  `json:"latency_p95_ms"`
	Errors       int64  `json:"errors"`
	Degraded     int64  `json:"degraded"`
}

type usageView struct {
	From  time.Time       `json:"from"`
	To    time.Time       `json:"to"`
	Group string          `json:"group"`
	Items []usageItemView `json:"items"`
}

type budgetView struct {
	Scope        string       `json:"scope"`
	CourseID     *string      `json:"course_id,omitempty"`
	DailyLimit   *string      `json:"daily_limit"`
	MonthlyLimit *string      `json:"monthly_limit"`
	SpentToday   string       `json:"spent_today"`
	SpentMonth   string       `json:"spent_month"`
	PctToday     *json.Number `json:"pct_today"` // chỉ để hiển thị (SRS 6.3); không dùng để tính tiền
	PctMonth     *json.Number `json:"pct_month"`
	State        string       `json:"state"`
	Version      int          `json:"version"`
}

func limitStr(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.StringFixed(2)
	return &s
}

func pctNum(d *decimal.Decimal) *json.Number {
	if d == nil {
		return nil
	}
	n := json.Number(d.StringFixed(1))
	return &n
}

func stateName(s budget.State) string {
	switch s {
	case budget.Warn:
		return "warn"
	case budget.Exhausted:
		return "exhausted"
	}
	return "ok"
}

type testResultView struct {
	OK        bool   `json:"ok"`
	ErrorKind string `json:"error_kind,omitempty"`
	Message   string `json:"message,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}
