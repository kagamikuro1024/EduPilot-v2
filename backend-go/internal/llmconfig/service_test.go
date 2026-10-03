package llmconfig_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/testutil"
)

const canary = "sk-CANARY-7f3a9c1e-do-not-leak"

var adminID = uuid.MustParse("00000000-0000-7000-8000-0000000000a0")

type rig struct {
	pool *pgxpool.Pool
	svc  *llmconfig.Service
	res  *llmconfig.Resolver
	c    *crypto.Cipher
	ctx  context.Context // ADMIN
}

func asRole(ctx context.Context, r auth.Role) context.Context {
	return auth.WithPrincipal(ctx, auth.Principal{Sub: adminID.String(), Role: r})
}

func newRig(t *testing.T) *rig {
	t.Helper()
	url := testutil.MigratedPostgresURL(t)
	pool, err := pgxpool.New(t.Context(), url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	key := make([]byte, 32)
	_, err = rand.Read(key)
	require.NoError(t, err)
	c, err := crypto.NewCipher(key)
	require.NoError(t, err)
	return &rig{pool: pool, svc: llmconfig.New(pool, c), res: llmconfig.NewResolver(pool, c), c: c, ctx: asRole(t.Context(), auth.RoleAdmin)}
}

func chatModel(name string) llmconfig.ModelInput {
	return llmconfig.ModelInput{Model: name, Kind: "chat", PriceIn: decimal.RequireFromString("4000"), PriceOut: decimal.RequireFromString("16000")}
}

func embedModel(name string, dims int) llmconfig.ModelInput {
	return llmconfig.ModelInput{Model: name, Kind: "embedding", Dims: &dims}
}

func key(s string) *llmconfig.Secret { k := llmconfig.NewSecret(s); return &k }

func (r *rig) provider(t *testing.T, name string, k string, models ...llmconfig.ModelInput) llmconfig.Provider {
	t.Helper()
	in := llmconfig.ProviderInput{Type: "fake", Name: name, Models: models}
	if k != "" {
		in.APIKey = key(k)
	}
	p, err := r.svc.CreateProvider(r.ctx, in)
	require.NoError(t, err)
	return p
}

func (r *rig) auditCount(t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from audit_log where entity like 'llm\_%'`).Scan(&n))
	return n
}

func TestKeyAtRest(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	p := r.provider(t, "A", canary, chatModel("m1"))
	other := r.provider(t, "B", "sk-other", chatModel("m2"))
	require.True(t, p.HasKey)
	require.Equal(t, llmconfig.KeyOK, p.KeyStatus)

	var enc []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select api_key_enc from llm_providers where id=$1`, p.ID).Scan(&enc))
	require.NotContains(t, string(enc), canary)
	require.Equal(t, byte(0x01), enc[0])
	var hits int
	require.NoError(t, r.pool.QueryRow(t.Context(),
		`select count(*) from llm_providers where position(convert_to($1,'UTF8') in api_key_enc) > 0`, canary).Scan(&hits))
	require.Zero(t, hits)

	plain, err := r.res.DecryptKey(t.Context(), p.ID)
	require.NoError(t, err)
	require.Equal(t, canary, plain)

	// Đổi id bản ghi → AAD sai → unreadable; nhà khác không ảnh hưởng.
	_, err = r.pool.Exec(t.Context(), `update llm_providers set id = $2 where id = $1`, p.ID, uuid.New())
	require.Error(t, err, "llm_models.provider_id chặn đổi id khi còn mô hình") // FK không ON UPDATE
	_, err = r.pool.Exec(t.Context(), `delete from llm_models where provider_id=$1`, p.ID)
	require.NoError(t, err)
	newID := uuid.New()
	_, err = r.pool.Exec(t.Context(), `update llm_providers set id = $2 where id = $1`, p.ID, newID)
	require.NoError(t, err)
	_, err = r.res.DecryptKey(t.Context(), newID)
	require.ErrorIs(t, err, llmconfig.ErrKeyUnreadable)

	list, err := r.svc.ListProviders(r.ctx)
	require.NoError(t, err)
	status := map[string]llmconfig.KeyStatus{}
	for _, q := range list {
		status[q.Name] = q.KeyStatus
	}
	require.Equal(t, llmconfig.KeyUnreadable, status["A"])
	require.Equal(t, llmconfig.KeyOK, status[other.Name])

	// Không có khoá → missing.
	noKey := r.provider(t, "C", "")
	require.False(t, noKey.HasKey)
	require.Equal(t, llmconfig.KeyMissing, noKey.KeyStatus)
	_, err = r.res.DecryptKey(t.Context(), noKey.ID)
	require.ErrorIs(t, err, llmconfig.ErrKeyMissing)
}

func TestRedacted(t *testing.T) {
	t.Parallel()
	s := llmconfig.NewSecret(canary)
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("x", "k", s, slog.Any("k2", s))
	js, err := json.Marshal(struct{ K llmconfig.Secret }{s})
	require.NoError(t, err)
	for _, out := range []string{
		fmt.Sprintf("%v %+v %#v %s %q", s, s, s, s, s), s.String(), s.GoString(), buf.String(), string(js), fmt.Sprint(&s),
	} {
		require.NotContains(t, out, canary)
		require.Contains(t, out, "[REDACTED]")
	}
	var back llmconfig.Secret
	require.NoError(t, json.Unmarshal([]byte(`"`+canary+`"`), &back))
	require.False(t, back.IsEmpty())
	require.NotContains(t, fmt.Sprintf("%v", back), canary)
}

func TestNoKeyInOutputs(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	p := r.provider(t, "A", canary, chatModel("m1"), embedModel("e1", 1536))
	got, err := r.svc.GetProvider(r.ctx, p.ID)
	require.NoError(t, err)
	list, err := r.svc.ListProviders(r.ctx)
	require.NoError(t, err)
	snap, err := r.res.Snapshot(t.Context())
	require.NoError(t, err)

	var logbuf bytes.Buffer
	slog.New(slog.NewJSONHandler(&logbuf, nil)).Info("p", "provider", got, "list", list)
	for _, v := range []any{p, got, list, snap} {
		js, err := json.Marshal(v)
		require.NoError(t, err)
		out := strings.Join([]string{fmt.Sprintf("%v", v), fmt.Sprintf("%+v", v), fmt.Sprintf("%#v", v), string(js), logbuf.String()}, "\n")
		require.NotContains(t, out, canary)
	}
	// Lỗi cũng không mang khoá.
	_, err = r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", APIKey: key("  "), Version: 1})
	require.Error(t, err)
	require.NotContains(t, err.Error(), canary)
}

func TestVersion(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	p := r.provider(t, "A", "k1", chatModel("m1"))
	require.Equal(t, 1, p.Version)

	upd, err := r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A2", Version: 1})
	require.NoError(t, err)
	require.Equal(t, 2, upd.Version)
	require.Equal(t, "A2", upd.Name)

	_, err = r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A3", Version: 1})
	var vc *llmconfig.ErrVersionConflict
	require.ErrorAs(t, err, &vc)
	require.Equal(t, 2, vc.Current)

	_, err = r.svc.UpdateProvider(r.ctx, uuid.New(), llmconfig.ProviderInput{Type: "fake", Name: "Z", Version: 1})
	require.ErrorIs(t, err, llmconfig.ErrNotFound)

	_, err = r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "fake", Name: "A2"})
	require.ErrorIs(t, err, llmconfig.ErrDuplicateName)

	// ngân sách và tuyến cũng khoá lạc quan
	b, err := r.svc.SetBudget(r.ctx, llmconfig.ScopeSystem, nil, ptr(decimal.RequireFromString("100")), ptr(decimal.RequireFromString("1000")), 0)
	require.NoError(t, err)
	require.Equal(t, 1, b.Version)
	_, err = r.svc.SetBudget(r.ctx, llmconfig.ScopeSystem, nil, nil, nil, 0)
	require.ErrorAs(t, err, &vc)
	require.Equal(t, 1, vc.Current)
	b, err = r.svc.SetBudget(r.ctx, llmconfig.ScopeSystem, nil, ptr(decimal.RequireFromString("200")), nil, 1)
	require.NoError(t, err)
	require.Equal(t, 2, b.Version)
	require.Nil(t, b.Monthly)
	require.True(t, b.Daily.Equal(decimal.RequireFromString("200")))
}

func ptr[T any](v T) *T { return &v }

func TestKeyKeepReplace(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	p := r.provider(t, "A", "key-one", chatModel("m1"))
	enc := func() []byte {
		var b []byte
		require.NoError(t, r.pool.QueryRow(t.Context(), `select api_key_enc from llm_providers where id=$1`, p.ID).Scan(&b))
		return b
	}
	first := enc()

	// không gửi api_key → giữ nguyên từng byte
	p2, err := r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", Version: 1})
	require.NoError(t, err)
	require.Equal(t, first, enc())
	require.True(t, p2.HasKey)

	// gửi cùng giá trị → bản mã mới (nonce mới) nhưng giải mã vẫn đúng
	_, err = r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", APIKey: key("key-one"), Version: 2})
	require.NoError(t, err)
	require.NotEqual(t, first, enc())
	// thay khoá
	_, err = r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", APIKey: key("key-two"), Version: 3})
	require.NoError(t, err)
	plain, err := r.res.DecryptKey(t.Context(), p.ID)
	require.NoError(t, err)
	require.Equal(t, "key-two", plain)

	// chuỗi rỗng → lỗi dữ liệu, khoá không đổi
	_, err = r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", APIKey: key(""), Version: 4})
	var inv *llmconfig.ErrInvalid
	require.ErrorAs(t, err, &inv)
	require.Equal(t, "api_key", inv.Field)
	plain, err = r.res.DecryptKey(t.Context(), p.ID)
	require.NoError(t, err)
	require.Equal(t, "key-two", plain)
}

func TestDeleteInUse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	used := r.provider(t, "Used", "k", chatModel("m1"))
	free := r.provider(t, "Free", "k", chatModel("f1"), chatModel("f2"))
	_, err := r.svc.SetRoute(r.ctx, "CHAT", []uuid.UUID{used.Models[0].ID}, nil, 0)
	require.NoError(t, err)

	err = r.svc.DeleteProvider(r.ctx, used.ID)
	var inUse *llmconfig.ErrProviderInUse
	require.ErrorAs(t, err, &inUse)
	require.Equal(t, []string{"CHAT"}, inUse.Tasks)

	// bỏ mô hình đang dùng khỏi danh sách cũng bị chặn
	_, err = r.svc.UpdateProvider(r.ctx, used.ID, llmconfig.ProviderInput{Type: "fake", Name: "Used", Models: []llmconfig.ModelInput{chatModel("other")}, Version: 1})
	require.ErrorAs(t, err, &inUse)

	require.NoError(t, r.svc.DeleteProvider(r.ctx, free.ID))
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from llm_models where provider_id=$1`, free.ID).Scan(&n))
	require.Zero(t, n)
	require.ErrorIs(t, r.svc.DeleteProvider(r.ctx, free.ID), llmconfig.ErrNotFound)
}

func TestAuditLogRows(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	base := r.auditCount(t)
	step := func(want int, what string) {
		t.Helper()
		require.Equal(t, base+want, r.auditCount(t), what)
	}
	p := r.provider(t, "A", canary, chatModel("m1"), embedModel("e1", 1536))
	step(1, "create")
	_, err := r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", APIKey: key(canary + "2"), Version: 1})
	require.NoError(t, err)
	step(2, "update")
	_, err = r.svc.SetRoute(r.ctx, "CHAT", []uuid.UUID{p.Models[0].ID}, llmconfig.Params{"temperature": "0.2"}, 0)
	require.NoError(t, err)
	step(3, "set route")
	_, err = r.svc.SetBudget(r.ctx, llmconfig.ScopeSystem, nil, ptr(decimal.RequireFromString("5")), nil, 0)
	require.NoError(t, err)
	step(4, "set budget")
	// thao tác lỗi không ghi audit
	_, err = r.svc.SetRoute(r.ctx, "CHAT", nil, nil, 1)
	require.Error(t, err)
	step(4, "lỗi không ghi")
	require.NoError(t, r.svc.RecordTest(r.ctx, p.ID, true, ""))
	step(4, "test không ghi audit_log")

	rows, err := r.pool.Query(t.Context(), `select actor_id, action, entity, coalesce(before::text,'') || coalesce(after::text,'') from audit_log where entity like 'llm\_%' order by id`)
	require.NoError(t, err)
	defer rows.Close()
	var n int
	for rows.Next() {
		var actor *uuid.UUID
		var action, entity, body string
		require.NoError(t, rows.Scan(&actor, &action, &entity, &body))
		require.NotNil(t, actor)
		require.Equal(t, adminID, *actor)
		require.NotContains(t, body, canary)
		require.NotContains(t, strings.ToLower(body), "api_key_enc")
		n++
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 4, n)
	tmp := r.provider(t, "Tmp", "k")
	step(5, "create Tmp")
	require.NoError(t, r.svc.DeleteProvider(r.ctx, tmp.ID))
	step(6, "delete")
}

func TestLimits(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	// 100 mô hình / nhà
	many := make([]llmconfig.ModelInput, 0, 101)
	for i := range 101 {
		many = append(many, chatModel(fmt.Sprintf("m%03d", i)))
	}
	_, err := r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "fake", Name: "Big", Models: many})
	require.ErrorIs(t, err, llmconfig.ErrLimit)
	p, err := r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "fake", Name: "Big", Models: many[:100]})
	require.NoError(t, err)
	require.Len(t, p.Models, 100)
	_, err = r.svc.UpdateProvider(r.ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "Big", Models: many, Version: 1})
	require.ErrorIs(t, err, llmconfig.ErrLimit)

	// 20 nhà cung cấp
	for i := 1; i < 20; i++ {
		_, err := r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "fake", Name: fmt.Sprintf("P%02d", i)})
		require.NoError(t, err)
	}
	_, err = r.svc.CreateProvider(r.ctx, llmconfig.ProviderInput{Type: "fake", Name: "P21"})
	require.ErrorIs(t, err, llmconfig.ErrLimit)
}

func TestProviderValidation(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	u := func(s string) *string { return &s }
	cases := []struct {
		name  string
		in    llmconfig.ProviderInput
		field string
	}{
		{"type lạ", llmconfig.ProviderInput{Type: "mistral", Name: "x"}, "type"},
		{"tên rỗng", llmconfig.ProviderInput{Type: "fake", Name: "  "}, "name"},
		{"tên quá dài", llmconfig.ProviderInput{Type: "fake", Name: strings.Repeat("a", 61)}, "name"},
		{"openai_compatible thiếu base_url", llmconfig.ProviderInput{Type: "openai_compatible", Name: "x"}, "base_url"},
		{"base_url sai", llmconfig.ProviderInput{Type: "fake", Name: "x", BaseURL: u("ftp://x")}, "base_url"},
		{"rpm âm", llmconfig.ProviderInput{Type: "fake", Name: "x", RPMLimit: ptr(-1)}, "rpm_limit"},
		{"khoá rỗng", llmconfig.ProviderInput{Type: "fake", Name: "x", APIKey: key("")}, "api_key"},
		{"mô hình lạ", llmconfig.ProviderInput{Type: "fake", Name: "x", Models: []llmconfig.ModelInput{{Model: "m", Kind: "image"}}}, "models[0].kind"},
		{"embedding thiếu dims", llmconfig.ProviderInput{Type: "fake", Name: "x", Models: []llmconfig.ModelInput{{Model: "m", Kind: "embedding"}}}, "models[0].dims"},
		{"trùng mô hình", llmconfig.ProviderInput{Type: "fake", Name: "x", Models: []llmconfig.ModelInput{chatModel("m"), chatModel("m")}}, "models[1].model"},
		{"giá âm", llmconfig.ProviderInput{Type: "fake", Name: "x", Models: []llmconfig.ModelInput{{Model: "m", Kind: "chat", PriceIn: decimal.RequireFromString("-1")}}}, "models[0].price"},
	}
	for _, tc := range cases {
		_, err := r.svc.CreateProvider(r.ctx, tc.in)
		var inv *llmconfig.ErrInvalid
		require.ErrorAs(t, err, &inv, tc.name)
		require.Equal(t, tc.field, inv.Field, tc.name)
	}
}

func TestRouteRules(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	on := r.provider(t, "On", "k", chatModel("c1"), chatModel("c2"), chatModel("c3"), chatModel("c4"), chatModel("c5"),
		embedModel("e1536", 1536), embedModel("e768", 768), embedModel("e1536b", 1536))
	off := r.provider(t, "Off", "k", chatModel("o1"))
	disabled := false
	_, err := r.svc.UpdateProvider(r.ctx, off.ID, llmconfig.ProviderInput{Type: "fake", Name: "Off", Enabled: &disabled, Version: 1})
	require.NoError(t, err)
	off, err = r.svc.GetProvider(r.ctx, off.ID)
	require.NoError(t, err)

	id := map[string]uuid.UUID{}
	for _, m := range on.Models {
		id[m.Model] = m.ID
	}
	id["o1"] = off.Models[0].ID
	ids := func(names ...string) []uuid.UUID {
		out := make([]uuid.UUID, 0, len(names))
		for _, n := range names {
			out = append(out, id[n])
		}
		return out
	}

	routeInvalid := func(rule string) func(*testing.T, error) {
		return func(t *testing.T, err error) {
			var ri *llmconfig.ErrRouteInvalid
			require.ErrorAs(t, err, &ri)
			require.Equal(t, rule, ri.Rule)
		}
	}
	cases := []struct {
		name   string
		task   string
		chain  []uuid.UUID
		params llmconfig.Params
		check  func(*testing.T, error)
	}{
		{"chuỗi rỗng", "CHAT", nil, nil, routeInvalid(llmconfig.RuleChainEmpty)},
		{"chuỗi 5 mô hình", "CHAT", ids("c1", "c2", "c3", "c4", "c5"), nil, routeInvalid(llmconfig.RuleChainTooLong)},
		{"chat dùng mô hình embedding", "CHAT", ids("e1536"), nil, routeInvalid(llmconfig.RuleKindMismatch)},
		{"EMBEDDING dùng mô hình chat", "EMBEDDING", ids("c1"), nil, routeInvalid(llmconfig.RuleKindMismatch)},
		{"EMBEDDING 2 mô hình", "EMBEDDING", ids("e1536", "e1536b"), nil, routeInvalid(llmconfig.RuleEmbeddingSingle)},
		{"embedding 768 chiều", "EMBEDDING", ids("e768"), nil, func(t *testing.T, err error) {
			var dm *llmconfig.ErrDimsMismatch
			require.ErrorAs(t, err, &dm)
			require.Equal(t, 768, dm.Actual)
			require.Equal(t, 1536, dm.Expected)
		}},
		{"nhà cung cấp đang tắt", "CHAT", ids("o1"), nil, routeInvalid(llmconfig.RuleProviderDisabled)},
		{"trùng mô hình", "CHAT", ids("c1", "c2", "c1"), nil, routeInvalid(llmconfig.RuleDuplicateModel)},
		{"khoá params lạ", "CHAT", ids("c1"), llmconfig.Params{"top_p": "0.5"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"temperature 2.5", "CHAT", ids("c1"), llmconfig.Params{"temperature": "2.5"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"temperature âm", "CHAT", ids("c1"), llmconfig.Params{"temperature": "-0.1"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"max_tokens 0", "CHAT", ids("c1"), llmconfig.Params{"max_tokens": "0"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"max_tokens 32769", "CHAT", ids("c1"), llmconfig.Params{"max_tokens": "32769"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"max_tokens không nguyên", "CHAT", ids("c1"), llmconfig.Params{"max_tokens": "10.5"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"timeout_s 301", "CHAT", ids("c1"), llmconfig.Params{"timeout_s": "301"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"retries 6", "CHAT", ids("c1"), llmconfig.Params{"retries": "6"}, routeInvalid(llmconfig.RuleParamsOutOfRange)},
		{"mô hình không tồn tại", "CHAT", []uuid.UUID{uuid.New()}, nil, func(t *testing.T, err error) {
			var inv *llmconfig.ErrInvalid
			require.ErrorAs(t, err, &inv)
		}},
		{"tác vụ lạ", "SING", ids("c1"), nil, func(t *testing.T, err error) {
			var inv *llmconfig.ErrInvalid
			require.ErrorAs(t, err, &inv)
		}},
	}
	require.GreaterOrEqual(t, len(cases), 12)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := r.svc.SetRoute(r.ctx, tc.task, tc.chain, tc.params, 0)
			require.Error(t, err)
			tc.check(t, err)
		})
	}
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from llm_task_routes`).Scan(&n))
	require.Zero(t, n, "tuyến sai không được ghi gì")

	// ca đúng + biên
	res, err := r.svc.SetRoute(r.ctx, "CHAT", ids("c1", "c2", "c3", "c4"),
		llmconfig.Params{"temperature": "2", "max_tokens": "32768", "timeout_s": "300", "retries": "5"}, 0)
	require.NoError(t, err)
	require.Equal(t, 1, res.Route.Version)
	require.Len(t, res.Route.Chain, 4)
	require.False(t, res.ReindexRequired)
	// sai version
	_, err = r.svc.SetRoute(r.ctx, "CHAT", ids("c1"), nil, 0)
	var vc *llmconfig.ErrVersionConflict
	require.ErrorAs(t, err, &vc)
	require.Equal(t, 1, vc.Current)
	res, err = r.svc.SetRoute(r.ctx, "CHAT", ids("c2", "c1"), nil, 1)
	require.NoError(t, err)
	require.Equal(t, 2, res.Route.Version)
	require.Equal(t, "c2", res.Route.Chain[0].Model)

	// EMBEDDING: đặt lần đầu không cần dựng lại; đổi mô hình → ReindexRequired; đặt lại cùng mô hình → không
	res, err = r.svc.SetRoute(r.ctx, "EMBEDDING", ids("e1536"), nil, 0)
	require.NoError(t, err)
	require.False(t, res.ReindexRequired)
	res, err = r.svc.SetRoute(r.ctx, "EMBEDDING", ids("e1536"), nil, 1)
	require.NoError(t, err)
	require.False(t, res.ReindexRequired)
	res, err = r.svc.SetRoute(r.ctx, "EMBEDDING", ids("e1536b"), nil, 2)
	require.NoError(t, err)
	require.True(t, res.ReindexRequired)

	routes, err := r.svc.ListRoutes(r.ctx)
	require.NoError(t, err)
	require.Len(t, routes.Items, 7)
	require.Equal(t, "INTERACTIVE", routes.Items[0].Lane)
	require.Equal(t, "CHAT", routes.Items[0].Task)
	require.Len(t, routes.Items[0].Chain, 2)
	require.Empty(t, routes.Items[1].Chain)
	require.Equal(t, 0, routes.Items[1].Version)
	require.NotNil(t, routes.Embedding)
	require.Equal(t, "e1536b", routes.Embedding.Model)

	snap, err := r.res.Snapshot(t.Context())
	require.NoError(t, err)
	require.Equal(t, "c2", snap.Routes["CHAT"].Chain[0].Model)
	require.True(t, snap.Routes["CHAT"].Chain[0].PriceIn.Equal(decimal.RequireFromString("4000")))
}

func TestServiceRBAC(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	p := r.provider(t, "A", "k", chatModel("m1"))
	cid := uuid.New()
	now := time.Now()
	type op struct {
		name  string
		write bool
		call  func(ctx context.Context) error
	}
	ops := []op{
		{"ListProviders", false, func(ctx context.Context) error { _, err := r.svc.ListProviders(ctx); return err }},
		{"GetProvider", false, func(ctx context.Context) error { _, err := r.svc.GetProvider(ctx, p.ID); return err }},
		{"ListRoutes", false, func(ctx context.Context) error { _, err := r.svc.ListRoutes(ctx); return err }},
		{"GetBudget", false, func(ctx context.Context) error {
			_, err := r.svc.GetBudget(ctx, llmconfig.ScopeSystem, nil)
			return err
		}},
		{"Usage", false, func(ctx context.Context) error {
			_, err := r.svc.Usage(ctx, "task", now.Add(-time.Hour), now, nil)
			return err
		}},
		{"CreateProvider", true, func(ctx context.Context) error {
			_, err := r.svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "n-" + uuid.NewString()[:8]})
			return err
		}},
		{"UpdateProvider", true, func(ctx context.Context) error {
			_, err := r.svc.UpdateProvider(ctx, p.ID, llmconfig.ProviderInput{Type: "fake", Name: "A", Version: 99})
			return err
		}},
		{"DeleteProvider", true, func(ctx context.Context) error { return r.svc.DeleteProvider(ctx, uuid.New()) }},
		{"SetRoute", true, func(ctx context.Context) error {
			_, err := r.svc.SetRoute(ctx, "CHAT", []uuid.UUID{p.Models[0].ID}, nil, 99)
			return err
		}},
		{"SetBudget", true, func(ctx context.Context) error {
			_, err := r.svc.SetBudget(ctx, llmconfig.ScopeCourse, &cid, nil, nil, 99)
			return err
		}},
		{"RecordTest", true, func(ctx context.Context) error { return r.svc.RecordTest(ctx, uuid.New(), true, "") }},
	}
	roles := []struct {
		role      auth.Role
		canRead   bool
		canWrite  bool
		anonymous bool
	}{
		{auth.RoleAdmin, true, true, false},
		{auth.RoleTeacher, true, false, false},
		{auth.RoleTA, false, false, false},
		{auth.RoleStudent, false, false, false},
		{"", false, false, true},
	}
	for _, ro := range roles {
		for _, o := range ops {
			ctx := t.Context()
			if !ro.anonymous {
				ctx = asRole(ctx, ro.role)
			}
			err := o.call(ctx)
			allowed := ro.canRead
			if o.write {
				allowed = ro.canWrite
			}
			if allowed {
				require.False(t, errors.Is(err, llmconfig.ErrForbidden), "%s/%s bị chặn nhầm: %v", ro.role, o.name, err)
			} else {
				require.ErrorIs(t, err, llmconfig.ErrForbidden, "%s/%s phải bị chặn", ro.role, o.name)
			}
		}
	}
	// Chặn xảy ra TRƯỚC mọi tác dụng phụ: không có dòng audit nào từ vai bị chặn.
	require.Equal(t, 2, r.auditCount(t), "chỉ nhà cung cấp ban đầu và lần CreateProvider của ADMIN")
}

func TestUsageAndCostMoney(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.pool.Exec(t.Context(), `
		insert into llm_audit (task, lane, status, trace_id, latency_ms, tokens_in, tokens_out, cost_est, degraded)
		select 'CHAT', 'INTERACTIVE', case when i % 10 = 0 then 'error' else 'ok' end, md5(i::text), i * 10, 100, 50, 1.2345, i % 20 = 0
		  from generate_series(1, 100) i`)
	require.NoError(t, err)
	rows, err := r.svc.Usage(r.ctx, "task", time.Now().Add(-time.Hour), time.Now().Add(time.Minute), nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	u := rows[0]
	require.Equal(t, "CHAT", u.Key)
	require.EqualValues(t, 100, u.Calls)
	require.EqualValues(t, 10000, u.TokensIn)
	require.True(t, u.CostEst.Equal(decimal.RequireFromString("123.45")), u.CostEst.String())
	require.EqualValues(t, 505, u.LatencyP50Ms) // trung vị của 10..1000 bước 10
	require.EqualValues(t, 10, u.Errors)
	require.EqualValues(t, 5, u.Degraded)
	byDay, err := r.svc.Usage(r.ctx, "day", time.Now().Add(-time.Hour), time.Now().Add(time.Minute), nil)
	require.NoError(t, err)
	require.NotEmpty(t, byDay)

	_, err = r.svc.Usage(r.ctx, "task", time.Now().Add(-93*24*time.Hour), time.Now(), nil)
	var inv *llmconfig.ErrInvalid
	require.ErrorAs(t, err, &inv)
	_, err = r.svc.Usage(r.ctx, "week", time.Now().Add(-time.Hour), time.Now(), nil)
	require.ErrorAs(t, err, &inv)
}
