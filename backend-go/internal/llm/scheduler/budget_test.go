package scheduler_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/budget"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llm/scheduler"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/testutil"
)

type limits map[string]llmconfig.Budget

func (l limits) Budget(_ context.Context, scope string, course *uuid.UUID) (llmconfig.Budget, error) {
	k := scope
	if course != nil {
		k += ":" + course.String()
	}
	return l[k], nil
}

func dec(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }

type budgetRig struct {
	m    *budget.Manager
	rdb  *goredis.Client
	pool *pgxpool.Pool
	clk  *clock.Fake
	lim  limits
}

func newBudgetRig(t *testing.T, lim limits) *budgetRig {
	t.Helper()
	opt, err := goredis.ParseURL(testutil.RedisURL(t))
	require.NoError(t, err)
	rdb := goredis.NewClient(opt)
	t.Cleanup(func() { _ = rdb.Close() })
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	clk := clock.NewFake(time.Date(2026, 10, 3, 3, 0, 0, 0, time.UTC)) // 10:00 ICT
	m := budget.New(rdb, lim, pool, clk, discard()).WithPrefix(uuid.NewString() + ":")
	return &budgetRig{m: m, rdb: rdb, pool: pool, clk: clk, lim: lim}
}

func (b *budgetRig) warnRows(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, b.pool.QueryRow(t.Context(), `select count(*) from outbox where topic = 'llm.budget.warn'`).Scan(&n))
	return n
}

func TestBudgetWarn80Once(t *testing.T) {
	t.Parallel()
	r := newBudgetRig(t, limits{"system": {Daily: dec("100000"), Monthly: dec("2000000")}})
	ctx := t.Context()
	require.Equal(t, budget.OK, r.m.State(ctx, nil))
	r.m.Charge(ctx, nil, decimal.RequireFromString("79999"))
	require.Equal(t, budget.OK, r.m.State(ctx, nil))
	require.Zero(t, r.warnRows(t))

	r.m.Charge(ctx, nil, decimal.RequireFromString("1")) // đúng 80 %
	require.Equal(t, budget.Warn, r.m.State(ctx, nil))
	require.Equal(t, 1, r.warnRows(t))
	var payload string
	require.NoError(t, r.pool.QueryRow(ctx, `select payload::text from outbox where topic = 'llm.budget.warn'`).Scan(&payload))
	require.JSONEq(t, `{"scope":"system","period":"day","pct":80}`, payload)

	r.m.Charge(ctx, nil, decimal.RequireFromString("5000")) // vẫn warn: KHÔNG lặp sự kiện
	r.m.Charge(ctx, nil, decimal.RequireFromString("5000"))
	require.Equal(t, budget.Warn, r.m.State(ctx, nil))
	require.Equal(t, 1, r.warnRows(t), "mỗi phạm vi mỗi kỳ chỉ một sự kiện")
}

func TestBudgetTwoScopes(t *testing.T) {
	t.Parallel()
	c1, c2 := uuid.New(), uuid.New()
	r := newBudgetRig(t, limits{
		"system":                {Daily: dec("1000000")},
		"course:" + c1.String(): {Daily: dec("1000")},
	})
	ctx := t.Context()
	r.m.Charge(ctx, &c1, decimal.RequireFromString("1000")) // lớp c1 cạn, hệ thống 0,1 %
	require.Equal(t, budget.Exhausted, r.m.State(ctx, &c1))
	require.Equal(t, budget.OK, r.m.State(ctx, &c2), "lớp khác không bị ảnh hưởng")
	require.Equal(t, budget.OK, r.m.State(ctx, nil))

	r.lim["system"] = llmconfig.Budget{Daily: dec("1200")} // hệ thống nay 1.200 đ, đã chi 1.000 (83 %) → warn cho mọi lớp
	r.m.Invalidate()
	require.Equal(t, budget.Warn, r.m.State(ctx, &c2))
	r.m.Charge(ctx, &c2, decimal.RequireFromString("600")) // 1.600 ≥ 1.200
	require.Equal(t, budget.Exhausted, r.m.State(ctx, &c2), "hệ thống cạn → mọi lớp cạn")
	require.GreaterOrEqual(t, r.warnRows(t), 2, "warn riêng cho mỗi phạm vi")
}

func TestBudgetDayRollover(t *testing.T) {
	t.Parallel()
	r := newBudgetRig(t, limits{"system": {Daily: dec("1000"), Monthly: dec("100000")}})
	ctx := t.Context()
	r.clk.Advance(13*time.Hour + 59*time.Minute) // 23:59 ICT ngày 3/10
	r.m.Charge(ctx, nil, decimal.RequireFromString("1000"))
	require.Equal(t, budget.Exhausted, r.m.State(ctx, nil))
	r.clk.Advance(2 * time.Minute) // 00:01 ICT ngày 4/10: sang ngày mới (theo Asia/Ho_Chi_Minh, không phải UTC)
	require.Equal(t, budget.OK, r.m.State(ctx, nil), "bộ đếm ngày mới bắt đầu từ 0")
	r.m.Charge(ctx, nil, decimal.RequireFromString("1000"))
	require.Equal(t, budget.Exhausted, r.m.State(ctx, nil))

	// sang tháng: 31/10 → 1/11 ICT
	r.clk.Advance(28 * 24 * time.Hour)
	require.Equal(t, budget.OK, r.m.State(ctx, nil))
}

func TestBudgetReconcileFromAudit(t *testing.T) {
	t.Parallel()
	cid := uuid.New()
	r := newBudgetRig(t, limits{"system": {Daily: dec("1000")}})
	_, err := r.pool.Exec(t.Context(), `insert into llm_audit (task, lane, status, trace_id, cost_est, course_id, created_at)
		values ('CHAT','INTERACTIVE','ok','t1', 600.5, $1, $2), ('CHAT','INTERACTIVE','ok','t2', 500, null, $2)`, cid, r.clk.Now())
	require.NoError(t, err)
	require.Equal(t, budget.OK, r.m.State(t.Context(), nil), "Redis chưa có số liệu (mất dữ liệu)")
	require.NoError(t, r.m.Reconcile(t.Context()))
	require.Equal(t, budget.Exhausted, r.m.State(t.Context(), nil), "đối soát: 1.100,5 đ ≥ 1.000 đ")
}

// adminSvc dựng dịch vụ cấu hình + Registry thật trên Postgres để kiểm "mô hình chat rẻ nhất".
func TestBudgetExhaustedInteractiveCheapest(t *testing.T) {
	t.Parallel()
	br := newBudgetRig(t, limits{"system": {Daily: dec("100")}})
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	c, err := crypto.NewCipher(key)
	require.NoError(t, err)
	svc := llmconfig.New(br.pool, c)
	ctx := auth.WithPrincipal(t.Context(), auth.Principal{Sub: "00000000-0000-7000-8000-0000000000a0", Role: auth.RoleAdmin})
	price := func(in, out string) (decimal.Decimal, decimal.Decimal) {
		return decimal.RequireFromString(in), decimal.RequireFromString(out)
	}
	mk := func(name string, in, out string, models ...string) llmconfig.Provider {
		pi, po := price(in, out)
		var ms []llmconfig.ModelInput
		for _, m := range models {
			ms = append(ms, llmconfig.ModelInput{Model: m, Kind: "chat", PriceIn: pi, PriceOut: po})
		}
		p, err := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: name, Models: ms})
		require.NoError(t, err)
		return p
	}
	big := mk("BIG", "4000", "16000", "big-1")
	mid := mk("MID", "1000", "1000", "mid-1")
	cheap := mk("CHEAP", "100", "200", "cheap-1")
	_ = mk("CHEAP2", "100", "200", "cheap-2") // hoà giá → theo thứ tự tạo: CHEAP thắng
	_, err = svc.SetRoute(ctx, "CHAT", []uuid.UUID{big.Models[0].ID, mid.Models[0].ID}, nil, 0)
	require.NoError(t, err)
	_, err = svc.SetRoute(ctx, "INSIGHT", []uuid.UUID{big.Models[0].ID}, nil, 0)
	require.NoError(t, err)

	stubs := map[string]*stubProv{"BIG": {name: "BIG"}, "MID": {name: "MID"}, "CHEAP": {name: "CHEAP"}, "CHEAP2": {name: "CHEAP2"}}
	reg := llm.NewRegistry(llmconfig.NewResolver(br.pool, c), llm.EnvConfig{}, func(s llm.ProviderSpec) (provider.Provider, error) { return stubs[s.Name], nil }, discard())
	require.NoError(t, reg.Load(t.Context()))
	reg.OnReload(br.m.Invalidate)
	tgt, ok := reg.Cheapest()
	require.True(t, ok)
	require.Equal(t, "cheap-1", tgt.Model, "rẻ nhất = price_in + price_out nhỏ nhất; hoà thì theo created_at")

	rg := finish(t, reg, scheduler.New(baseCfg(), nil, discard(), scheduler.WithBudget(br.m)), nil)
	// chưa cạn: dùng tuyến bình thường
	resp, err := rg.g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("x")})
	require.NoError(t, err)
	require.Equal(t, "BIG", resp.Provider)

	br.m.Charge(t.Context(), nil, decimal.RequireFromString("100")) // hệ thống cạn
	resp, err = rg.g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("x")})
	require.NoError(t, err, "INTERACTIVE không bao giờ bị từ chối vì ngân sách")
	require.Equal(t, "CHEAP", resp.Provider)
	require.Equal(t, "cheap-1", resp.Model)
	lane := llm.LaneNearRealtime
	resp, err = rg.g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Lane: &lane, Messages: msg("x")})
	require.NoError(t, err)
	require.Equal(t, "CHEAP", resp.Provider, "NEAR_REALTIME cũng chuyển sang mô hình rẻ")
	_ = cheap
}

func TestBudgetExhaustedBatchStops(t *testing.T) {
	t.Parallel()
	br := newBudgetRig(t, limits{"system": {Daily: dec("100")}})
	p := &stubProv{name: "P"}
	reg := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	for _, task := range llm.ChatTasks() {
		reg.SetRoute(task, llm.Route{Targets: []llm.Target{{ProviderID: "P", ProviderName: "P", Model: "m", RPM: 1e6, TPM: 1e9, P: p}}})
	}
	rg := finish(t, reg, scheduler.New(baseCfg(), nil, discard(), scheduler.WithBudget(br.m)), nil)
	br.m.Charge(t.Context(), nil, decimal.RequireFromString("100"))
	before := p.calls.Load()
	for _, task := range []llm.Task{llm.TaskGrading, llm.TaskInsight, llm.TaskQuestionGen} {
		_, err := rg.g.Chat(t.Context(), llm.Request{Task: task, Messages: msg("x")})
		var un *llm.ErrUnavailable
		if !errors.As(err, &un) || un.Reason != llm.ReasonBudgetExhausted {
			t.Errorf("%s: err = %v, muốn LLM_UNAVAILABLE budget_exhausted", task, err)
		}
	}
	if p.calls.Load() != before {
		t.Error("BATCH bị chặn không được gọi nhà cung cấp")
	}
	rows := rg.cap.all(rg.g)
	if len(rows) != 3 || rows[0].Status != "budget_blocked" {
		t.Errorf("audit = %+v", rows)
	}
	// chat sinh viên vẫn trả lời
	if resp, err := rg.g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("x")}); err != nil || resp.Text == "" {
		t.Errorf("chat khi ngân sách cạn: %v %v", resp, err)
	}
}
