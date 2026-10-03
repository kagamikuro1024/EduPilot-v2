package llmconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EmbedDims là số chiều vectơ cố định của hệ thống (pgvector vector(1536)).
const EmbedDims = 1536

// Params là tham số của một tuyến: khoá ∈ {temperature, max_tokens, timeout_s, retries}; số giữ nguyên dạng chữ (json.Number), không float.
type Params map[string]json.Number

type paramRange struct {
	lo, hi  decimal.Decimal
	integer bool
}

func paramRanges() map[string]paramRange {
	return map[string]paramRange{
		"temperature": {lo: decimal.NewFromInt(0), hi: decimal.NewFromInt(2)},
		"max_tokens":  {lo: decimal.NewFromInt(1), hi: decimal.NewFromInt(32768), integer: true},
		"timeout_s":   {lo: decimal.NewFromInt(1), hi: decimal.NewFromInt(300), integer: true},
		"retries":     {lo: decimal.NewFromInt(0), hi: decimal.NewFromInt(5), integer: true},
	}
}

func validateParams(p Params) error {
	ranges := paramRanges()
	for k, v := range p {
		rg, ok := ranges[k]
		field := "params." + k
		if !ok {
			return &ErrRouteInvalid{Rule: RuleParamsOutOfRange, Field: field}
		}
		d, err := decimal.NewFromString(v.String())
		if err != nil || d.LessThan(rg.lo) || d.GreaterThan(rg.hi) || (rg.integer && !d.IsInteger()) {
			return &ErrRouteInvalid{Rule: RuleParamsOutOfRange, Field: field}
		}
	}
	return nil
}

// TaskLane là làn mặc định của tác vụ (SRS 4.3).
func TaskLane(task string) string {
	switch task {
	case "CHAT":
		return "INTERACTIVE"
	case "CLASSIFY", "UTILITY":
		return "NEAR_REALTIME"
	default:
		return "BATCH"
	}
}

// ChainItem là một mô hình trong chuỗi tuyến đã lưu.
type ChainItem struct {
	ModelID      uuid.UUID
	ProviderID   uuid.UUID
	ProviderName string
	Model        string
}

// RouteView là tuyến của một tác vụ để hiển thị; tác vụ chưa cấu hình có Chain rỗng và Version 0.
type RouteView struct {
	Task    string
	Lane    string
	Chain   []ChainItem
	Params  Params
	Version int
}

// EmbeddingView là khối mô hình nhúng của GET routes.
type EmbeddingView struct {
	ModelID      uuid.UUID
	ProviderName string
	Model        string
	Dims         int
}

// Routes là kết quả ListRoutes: 7 tác vụ theo thứ tự cố định.
type Routes struct {
	Items     []RouteView
	Embedding *EmbeddingView
}

func viewOf(task string, rt Route, ok bool) RouteView {
	v := RouteView{Task: task, Lane: TaskLane(task), Chain: []ChainItem{}, Params: Params{}}
	if !ok {
		return v
	}
	v.Version, v.Params = rt.Version, rt.Params
	if v.Params == nil {
		v.Params = Params{}
	}
	for _, e := range rt.Chain {
		v.Chain = append(v.Chain, ChainItem{ModelID: e.ModelID, ProviderID: e.ProviderID, ProviderName: e.ProviderName, Model: e.Model})
	}
	return v
}

// ListRoutes trả tuyến của cả 7 tác vụ (ADMIN, TEACHER).
func (s *Service) ListRoutes(ctx context.Context) (Routes, error) {
	if _, err := actorFrom(ctx, false); err != nil {
		return Routes{}, err
	}
	rts, err := s.res.routes(ctx)
	if err != nil {
		return Routes{}, err
	}
	out := Routes{}
	for _, task := range Tasks() {
		rt, ok := rts[task]
		out.Items = append(out.Items, viewOf(task, rt, ok))
		if task == "EMBEDDING" && ok && len(rt.Chain) == 1 {
			e := rt.Chain[0]
			out.Embedding = &EmbeddingView{ModelID: e.ModelID, ProviderName: e.ProviderName, Model: e.Model, Dims: EmbedDims}
		}
	}
	return out, nil
}

// SetRouteResult là kết quả SetRoute; ReindexRequired=true khi đổi mô hình EMBEDDING so với giá trị đang lưu.
type SetRouteResult struct {
	Route           RouteView
	ReindexRequired bool
}

// SetRoute đặt chuỗi mô hình của một tác vụ (ADMIN). version: 0 khi tác vụ chưa có tuyến, ngược lại phải khớp.
func (s *Service) SetRoute(ctx context.Context, task string, chain []uuid.UUID, params Params, version int) (SetRouteResult, error) {
	a, err := actorFrom(ctx, true)
	if err != nil {
		return SetRouteResult{}, err
	}
	if !slices.Contains(Tasks(), task) {
		return SetRouteResult{}, invalid("task", "INVALID_TASK", "tác vụ không hợp lệ")
	}
	switch {
	case len(chain) == 0:
		return SetRouteResult{}, &ErrRouteInvalid{Rule: RuleChainEmpty, Field: "chain"}
	case len(chain) > maxChain:
		return SetRouteResult{}, &ErrRouteInvalid{Rule: RuleChainTooLong, Field: "chain"}
	case task == "EMBEDDING" && len(chain) != 1:
		return SetRouteResult{}, &ErrRouteInvalid{Rule: RuleEmbeddingSingle, Field: "chain"}
	}
	if err := validateParams(params); err != nil {
		return SetRouteResult{}, err
	}
	for i, id := range chain {
		if slices.Index(chain, id) != i {
			return SetRouteResult{}, &ErrRouteInvalid{Rule: RuleDuplicateModel, Field: fmt.Sprintf("chain[%d]", i)}
		}
	}
	if params == nil {
		params = Params{}
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return SetRouteResult{}, fmt.Errorf("llmconfig: mã hoá params: %w", err)
	}

	var res SetRouteResult
	err = s.inTx(ctx, func(q *store.Queries) error {
		if err := q.LockLLMTask(ctx, task); err != nil {
			return fmt.Errorf("llmconfig: khoá tác vụ: %w", err)
		}
		models, err := s.checkChain(ctx, q, task, chain)
		if err != nil {
			return err
		}
		old, err := q.ListLLMRoutesByTask(ctx, task)
		if err != nil {
			return fmt.Errorf("llmconfig: đọc tuyến cũ: %w", err)
		}
		cur := 0
		if len(old) > 0 {
			cur = int(old[0].Version)
		}
		if cur != version {
			return &ErrVersionConflict{Current: cur}
		}
		before := routeSnapshot(old)
		if err := q.DeleteLLMRoutesByTask(ctx, task); err != nil {
			return fmt.Errorf("llmconfig: xoá tuyến cũ: %w", err)
		}
		for i, id := range chain {
			if _, err := q.InsertLLMRoute(ctx, store.InsertLLMRouteParams{
				Task: task, ModelID: id, FallbackOrder: int32(i), Params: raw, Version: int32(cur + 1), //nolint:gosec // i ≤ 3
			}); err != nil {
				return fmt.Errorf("llmconfig: chèn tuyến: %w", err)
			}
		}
		res.ReindexRequired = task == "EMBEDDING" && len(old) > 0 && old[0].ModelID != chain[0]
		rv := RouteView{Task: task, Lane: TaskLane(task), Params: params, Version: cur + 1}
		for _, id := range chain {
			m := models[id]
			rv.Chain = append(rv.Chain, ChainItem{ModelID: id, ProviderID: m.ProviderID, ProviderName: m.ProviderName, Model: m.Model})
		}
		res.Route = rv
		after := map[string]any{"models": routeNames(rv.Chain), "params": params, "reindex_required": res.ReindexRequired}
		return s.writeAudit(ctx, q, a, "llm_route", task, "set", before, after)
	})
	if err != nil {
		return SetRouteResult{}, err
	}
	s.changed(ctx)
	return res, nil
}

type chainModel struct {
	ProviderID   uuid.UUID
	ProviderName string
	Model        string
}

// checkChain áp quy tắc tuyến lên từng mô hình (loại, số chiều, nhà cung cấp bật).
func (s *Service) checkChain(ctx context.Context, q *store.Queries, task string, chain []uuid.UUID) (map[uuid.UUID]chainModel, error) {
	ids := make([]string, len(chain)) // text[]: PgBouncer (QueryExecModeExec) không mã hoá được []uuid.UUID
	for i, c := range chain {
		ids[i] = c.String()
	}
	ms, err := q.ListLLMModelsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("llmconfig: đọc mô hình: %w", err)
	}
	byID := make(map[uuid.UUID]store.LlmModel, len(ms))
	for _, m := range ms {
		byID[m.ID] = m
	}
	wantKind := "chat"
	if task == "EMBEDDING" {
		wantKind = "embedding"
	}
	out := make(map[uuid.UUID]chainModel, len(chain))
	for i, id := range chain {
		field := fmt.Sprintf("chain[%d]", i)
		m, ok := byID[id]
		if !ok {
			return nil, invalid(field, "MODEL_NOT_FOUND", "mô hình không tồn tại")
		}
		if m.Kind != wantKind {
			return nil, &ErrRouteInvalid{Rule: RuleKindMismatch, Field: field}
		}
		if wantKind == "embedding" && (m.Dims == nil || int(*m.Dims) != EmbedDims) {
			actual := 0
			if m.Dims != nil {
				actual = int(*m.Dims)
			}
			return nil, &ErrDimsMismatch{Expected: EmbedDims, Actual: actual}
		}
		p, err := q.GetLLMProvider(ctx, m.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("llmconfig: đọc nhà cung cấp của mô hình: %w", err)
		}
		if !p.Enabled {
			return nil, &ErrRouteInvalid{Rule: RuleProviderDisabled, Field: field}
		}
		out[id] = chainModel{ProviderID: p.ID, ProviderName: p.Name, Model: m.Model}
	}
	return out, nil
}

func routeSnapshot(rows []store.LlmTaskRoute) any {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ModelID.String())
	}
	return map[string]any{"model_ids": ids, "version": rows[0].Version}
}

func routeNames(c []ChainItem) []string {
	out := make([]string, 0, len(c))
	for _, it := range c {
		out = append(out, it.ProviderName+"/"+it.Model)
	}
	return out
}
