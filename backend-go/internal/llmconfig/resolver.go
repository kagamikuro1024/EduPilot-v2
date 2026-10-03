package llmconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Resolver đọc cấu hình cho cổng `internal/llm` (không kiểm vai: người gọi là mã hệ thống, không phải người dùng).
// DecryptKey là HÀM DUY NHẤT trả khoá API ở dạng rõ; chỉ `internal/llm` (và gói này) được gọi — kiểm bằng grep ở US-P1-01 AC8.
type Resolver struct {
	q      *store.Queries
	cipher *crypto.Cipher
}

// NewResolver dựng Resolver trên DBTX của store (pool hoặc tx).
func NewResolver(db store.DBTX, c *crypto.Cipher) *Resolver {
	return &Resolver{q: store.New(db), cipher: c}
}

// DecryptKey trả khoá rõ của nhà cung cấp; ErrKeyMissing khi chưa có, ErrKeyUnreadable khi AAD/khoá không khớp.
func (r *Resolver) DecryptKey(ctx context.Context, providerID uuid.UUID) (string, error) {
	p, err := r.q.GetLLMProvider(ctx, providerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("llmconfig: đọc nhà cung cấp: %w", err)
	}
	if len(p.ApiKeyEnc) == 0 {
		return "", ErrKeyMissing
	}
	plain, err := r.cipher.Decrypt(p.ApiKeyEnc, aadFor(p.ID))
	if err != nil {
		return "", ErrKeyUnreadable
	}
	return string(plain), nil
}

// RouteEntry là một mô hình trong chuỗi của tác vụ, kèm đủ thông tin cho cổng llm (giá, nhà cung cấp).
type RouteEntry struct {
	ModelID         uuid.UUID
	ProviderID      uuid.UUID
	ProviderName    string
	ProviderEnabled bool
	Model           string
	Kind            string
	Dims            *int
	PriceIn         decimal.Decimal
	PriceOut        decimal.Decimal
	ModelEnabled    bool
}

// Route là chuỗi mô hình của một tác vụ (phần tử 0 = chính) và tham số.
type Route struct {
	Task    string
	Chain   []RouteEntry
	Params  Params
	Version int
}

// Snapshot là toàn bộ cấu hình cần cho Registry: nhà cung cấp (không khoá) và tuyến.
type Snapshot struct {
	Providers []Provider
	Routes    map[string]Route
}

// Snapshot đọc cấu hình hiện hành (3 truy vấn).
func (r *Resolver) Snapshot(ctx context.Context) (Snapshot, error) {
	ps, err := r.q.ListLLMProviders(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("llmconfig: liệt kê nhà cung cấp: %w", err)
	}
	ms, err := r.q.ListLLMModels(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("llmconfig: liệt kê mô hình: %w", err)
	}
	byProvider := map[uuid.UUID][]Model{}
	for _, m := range ms {
		byProvider[m.ProviderID] = append(byProvider[m.ProviderID], toModel(m))
	}
	snap := Snapshot{Providers: make([]Provider, 0, len(ps)), Routes: map[string]Route{}}
	for _, p := range ps {
		snap.Providers = append(snap.Providers, toProvider(p, byProvider[p.ID], r.cipher))
	}
	routes, err := r.routes(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	snap.Routes = routes
	return snap, nil
}

func (r *Resolver) routes(ctx context.Context) (map[string]Route, error) {
	rows, err := r.q.ListLLMRoutes(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmconfig: liệt kê tuyến: %w", err)
	}
	out := map[string]Route{}
	for _, row := range rows {
		rt := out[row.Task]
		rt.Task, rt.Version = row.Task, int(row.Version)
		if len(rt.Chain) == 0 {
			if err := json.Unmarshal(row.Params, &rt.Params); err != nil {
				return nil, fmt.Errorf("llmconfig: params của %s hỏng: %w", row.Task, err)
			}
		}
		rt.Chain = append(rt.Chain, RouteEntry{
			ModelID: row.ModelID, ProviderID: row.ProviderID, ProviderName: row.ProviderName, ProviderEnabled: row.ProviderEnabled,
			Model: row.Model, Kind: row.Kind, Dims: intPtr(row.Dims), PriceIn: row.PriceIn, PriceOut: row.PriceOut, ModelEnabled: row.ModelEnabled,
		})
		out[row.Task] = rt
	}
	return out, nil
}

// Budget đọc ngân sách (scope "system" hoặc "course"); chưa có dòng → Version 0, không hạn mức.
func (r *Resolver) Budget(ctx context.Context, scope string, courseID *uuid.UUID) (Budget, error) {
	return readBudget(ctx, r.q, scope, courseID)
}
