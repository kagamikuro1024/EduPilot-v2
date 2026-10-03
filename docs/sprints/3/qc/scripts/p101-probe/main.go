// QC probe cho US-P1-01 (hộp đen qua API Go công khai của llmconfig; không sửa mã dev).
// Chạy: copy vào backend-go/cmd/qcp101/ của worktree QC, `go run ./cmd/qcp101` với env của gateway.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const canary = "sk-LEAK-CANARY-7f3a9c1e"

func as(r auth.Role) context.Context {
	return auth.WithPrincipal(context.Background(), auth.Principal{Sub: "00000000-0000-7000-8000-0000000000a0", Role: r})
}
func p(k string, v any)              { fmt.Printf("%s %v\n", k, v) }
func sec(s string) *llmconfig.Secret { x := llmconfig.NewSecret(s); return &x }
func chat(m string) llmconfig.ModelInput {
	return llmconfig.ModelInput{Model: m, Kind: "chat", PriceIn: decimal.NewFromInt(1), PriceOut: decimal.NewFromInt(2)}
}

func main() {
	ctx := as(auth.RoleAdmin)
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	c, err := crypto.ParseKey(os.Getenv("APP_ENCRYPTION_KEY"))
	if err != nil {
		panic(err)
	}
	svc := llmconfig.New(pool, c)
	cnt := func(q string) int { var n int; _ = pool.QueryRow(ctx, q).Scan(&n); return n }
	md5 := func(id uuid.UUID) string {
		var s string
		_ = pool.QueryRow(ctx, "select md5(api_key_enc) from llm_providers where id=$1", id).Scan(&s)
		return s
	}
	if hx := os.Getenv("QC_PY_CT"); hx != "" { // chế độ 2: nhét bản mã do công cụ ngoài (node:crypto) tạo cho QC-C rồi giải bằng Go
		var cid uuid.UUID
		_ = pool.QueryRow(ctx, "select id from llm_providers where name='QC-C'").Scan(&cid)
		_, err := pool.Exec(ctx, "update llm_providers set api_key_enc=decode($2,'hex') where id=$1", cid, hx)
		r := llmconfig.NewResolver(pool, c)
		k, e2 := r.DecryptKey(ctx, cid)
		p("ext_ct_decrypt", fmt.Sprint(err, " plain=", k, " err=", e2))
		_, _ = pool.Exec(ctx, "update llm_providers set api_key_enc=set_byte(api_key_enc,0,2) where id=$1", cid)
		_, e3 := r.DecryptKey(ctx, cid)
		p("verflip_decrypt_err", e3)
		return
	}
	// --- tạo
	a, err := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "QC-A", APIKey: sec(canary), Models: []llmconfig.ModelInput{chat("m-a1"), chat("m-a2")}})
	p("create_A_err", err)
	b, _ := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "QC-B", APIKey: sec("other-key-123456"), Models: []llmconfig.ModelInput{chat("m-b1")}})
	d := 1536
	cc, _ := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "QC-C", APIKey: sec("third-key-abcdef"), Models: []llmconfig.ModelInput{chat("m-c1"), {Model: "emb-1536", Kind: "embedding", Dims: &d}, {Model: "emb-1536b", Kind: "embedding", Dims: &d}}})
	p("ID_A", a.ID)
	p("ID_B", b.ID)
	p("ID_C", cc.ID)
	p("A_hasKey_status", fmt.Sprint(a.HasKey, " ", a.KeyStatus))
	// --- đầu ra không lộ khoá (TC-24)
	in := llmconfig.ProviderInput{Type: "fake", Name: "x", APIKey: sec(canary)}
	var buf strings.Builder
	h := slog.New(slog.NewJSONHandler(&buf, nil))
	h.Info("probe", "in", in, "key", *in.APIKey, "provider", a)
	j1, _ := json.Marshal(in)
	j2, _ := json.Marshal(a)
	all := fmt.Sprintf("%v %+v %#v", in, in, in) + fmt.Sprintf("%v %+v %#v", a, a, a) + string(j1) + string(j2) + buf.String()
	p("leak_canary_in_outputs", strings.Contains(all, canary))
	p("leak_tail4_in_outputs", strings.Contains(all, "9c1e"))
	p("redacted_present", strings.Contains(all, "[REDACTED]"))
	// --- giữ / thay khoá (TC-27)
	m0 := md5(a.ID)
	a2, err := svc.UpdateProvider(ctx, a.ID, llmconfig.ProviderInput{Type: "fake", Name: "QC-A", Version: a.Version})
	p("update_nokey_err", err)
	p("key_kept", md5(a.ID) == m0)
	_, err = svc.UpdateProvider(ctx, a.ID, llmconfig.ProviderInput{Type: "fake", Name: "QC-A", APIKey: sec(""), Version: a2.Version})
	p("update_emptykey_err", err)
	p("key_unchanged_after_empty", md5(a.ID) == m0)
	a3, err := svc.UpdateProvider(ctx, a.ID, llmconfig.ProviderInput{Type: "fake", Name: "QC-A", APIKey: sec(canary), Version: a2.Version})
	p("update_samekey_err", err)
	p("blob_changed_new_nonce", md5(a.ID) != m0)
	_, err = svc.UpdateProvider(ctx, a.ID, llmconfig.ProviderInput{Type: "fake", Name: "QC-A", Version: 1})
	var vc *llmconfig.ErrVersionConflict
	p("stale_version_conflict", errors.As(err, &vc))
	if vc != nil {
		p("stale_current", vc.Current)
	}
	_ = a3
	// --- audit_log (TC-26)
	n0 := cnt("select count(*) from audit_log")
	_, _ = svc.UpdateProvider(ctx, b.ID, llmconfig.ProviderInput{Type: "fake", Name: "QC-B2", Version: b.Version})
	p("audit_delta_update", cnt("select count(*) from audit_log")-n0)
	p("audit_has_canary", cnt("select count(*) from audit_log where (before::text||after::text||coalesce(entity_id,'')) like '%7f3a9c1e%'"))
	// --- route rules (TC-29/30)
	dims768 := 768
	bad, _ := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "QC-D", APIKey: sec("k-dddddddd"), Models: []llmconfig.ModelInput{{Model: "emb-768", Kind: "embedding", Dims: &dims768}, chat("m-d1")}})
	off := false
	dis, _ := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "QC-OFF", APIKey: sec("k-eeeeeeee"), Enabled: &off, Models: []llmconfig.ModelInput{chat("m-off")}})
	mid := func(pv llmconfig.Provider, name string) uuid.UUID {
		x, _ := svc.GetProvider(ctx, pv.ID)
		for _, m := range x.Models {
			if m.Model == name {
				return m.ID
			}
		}
		return uuid.Nil
	}
	mA1, mA2, mB1, mC1 := mid(a, "m-a1"), mid(a, "m-a2"), mid(b, "m-b1"), mid(cc, "m-c1")
	e1, e2, e768, mD1, mOff := mid(cc, "emb-1536"), mid(cc, "emb-1536b"), mid(bad, "emb-768"), mid(bad, "m-d1"), mid(dis, "m-off")
	try := func(name, task string, ch []uuid.UUID, pr llmconfig.Params, ver int) {
		before := cnt("select count(*) from llm_task_routes")
		_, err := svc.SetRoute(ctx, task, ch, pr, ver)
		p("route_"+name, fmt.Sprintf("err=%v rows_changed=%v", err, cnt("select count(*) from llm_task_routes")-before))
	}
	jn := func(s string) json.Number { return json.Number(s) }
	try("empty_chain", "CHAT", nil, nil, 0)
	try("5_models", "CHAT", []uuid.UUID{mA1, mA2, mB1, mC1, mD1}, nil, 0)
	try("embedding_for_chat", "CHAT", []uuid.UUID{e1}, nil, 0)
	try("chat_for_embedding", "EMBEDDING", []uuid.UUID{mA1}, nil, 0)
	try("embedding_2_models", "EMBEDDING", []uuid.UUID{e1, e2}, nil, 0)
	try("dims_768", "EMBEDDING", []uuid.UUID{e768}, nil, 0)
	try("provider_off", "CHAT", []uuid.UUID{mOff}, nil, 0)
	try("dup_in_chain", "CHAT", []uuid.UUID{mA1, mA1}, nil, 0)
	try("temp_2_1", "CHAT", []uuid.UUID{mA1}, llmconfig.Params{"temperature": jn("2.1")}, 0)
	try("maxtok_40000", "CHAT", []uuid.UUID{mA1}, llmconfig.Params{"max_tokens": jn("40000")}, 0)
	try("timeout_0", "CHAT", []uuid.UUID{mA1}, llmconfig.Params{"timeout_s": jn("0")}, 0)
	try("retries_6", "CHAT", []uuid.UUID{mA1}, llmconfig.Params{"retries": jn("6")}, 0)
	try("unknown_param", "CHAT", []uuid.UUID{mA1}, llmconfig.Params{"top_p": jn("1")}, 0)
	try("bad_task", "NOPE", []uuid.UUID{mA1}, nil, 0)
	try("OK_edge_low", "CLASSIFY", []uuid.UUID{mA1}, llmconfig.Params{"temperature": jn("0"), "retries": jn("0"), "max_tokens": jn("1"), "timeout_s": jn("1")}, 0)
	try("OK_edge_high", "UTILITY", []uuid.UUID{mA1}, llmconfig.Params{"temperature": jn("2"), "retries": jn("5"), "max_tokens": jn("32768"), "timeout_s": jn("300")}, 0)
	try("OK_chain4", "CHAT", []uuid.UUID{mA1, mA2, mB1, mC1}, nil, 0)
	res, err := svc.SetRoute(ctx, "EMBEDDING", []uuid.UUID{e1}, nil, 0)
	p("embed_first_set", fmt.Sprint("err=", err, " reindex=", res.ReindexRequired))
	res, err = svc.SetRoute(ctx, "EMBEDDING", []uuid.UUID{e2}, nil, 1)
	p("embed_change", fmt.Sprint("err=", err, " reindex=", res.ReindexRequired))
	// --- xoá đang dùng / khỏi hạn mức (TC-27/28)
	err = svc.DeleteProvider(ctx, a.ID)
	var iu *llmconfig.ErrProviderInUse
	p("delete_in_use", fmt.Sprint(errors.As(err, &iu), " ", func() any {
		if iu != nil {
			return iu.Tasks
		}
		return nil
	}()))
	nm := cnt("select count(*) from llm_models")
	err = svc.DeleteProvider(ctx, dis.ID)
	p("delete_unused", fmt.Sprint(err, " models_removed=", nm-cnt("select count(*) from llm_models"), " orphan_models=", cnt("select count(*) from llm_models m where not exists(select 1 from llm_providers p where p.id=m.provider_id)")))
	// --- RBAC (TC-32)
	for _, r := range []auth.Role{auth.RoleAdmin, auth.RoleTeacher, auth.RoleTA, auth.RoleStudent, ""} {
		cx := as(r)
		_, e1 := svc.ListProviders(cx)
		_, e2 := svc.GetProvider(cx, a.ID)
		_, e3 := svc.ListRoutes(cx)
		_, e4 := svc.GetBudget(cx, "system", nil)
		_, e5 := svc.CreateProvider(cx, llmconfig.ProviderInput{Type: "fake", Name: "rbac-" + string(r), APIKey: sec("kkkkkkkk")})
		e6 := svc.DeleteProvider(cx, uuid.New())
		_, e7 := svc.SetRoute(cx, "CHAT", []uuid.UUID{mA1}, nil, 4)
		_, e8 := svc.SetBudget(cx, "system", nil, nil, nil, 0)
		f := func(e error) string {
			if errors.Is(e, llmconfig.ErrForbidden) {
				return "F"
			}
			return "ok"
		}
		p("rbac_"+string(r)+"_(list,get,routes,budget,create,delete,setroute,setbudget)", strings.Join([]string{f(e1), f(e2), f(e3), f(e4), f(e5), f(e6), f(e7), f(e8)}, ","))
	}
	// --- hoán đổi khoá giữa bản ghi (AAD, TC-21)
	_, _ = pool.Exec(ctx, "create temp table _k as select 1")
	_, _ = pool.Exec(ctx, "update llm_providers set api_key_enc = (select api_key_enc from llm_providers where id=$2) where id=$1", a.ID, b.ID)
	lst, _ := svc.ListProviders(ctx)
	for _, x := range lst {
		p("aad_swap_"+x.Name, fmt.Sprint(x.HasKey, " ", x.KeyStatus))
	}
	// --- giới hạn (TC-28)
	n := cnt("select count(*) from llm_providers")
	var lerr error
	made := 0
	for i := 0; i < 25; i++ {
		_, lerr = svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: fmt.Sprintf("lim-%d", i), APIKey: sec("kkkkkkkk")})
		if lerr != nil {
			break
		}
		made++
	}
	p("provider_limit", fmt.Sprintf("existing=%d created_more=%d stop_err=%v", n, made, lerr))
	// --- giải mã chéo (TC-16): in bản mã + id
	rows, _ := pool.Query(ctx, "select id, name, encode(api_key_enc,'hex') from llm_providers where name in ('QC-B2','QC-C')")
	for rows.Next() {
		var id, nm, hx string
		_ = rows.Scan(&id, &nm, &hx)
		p("CT", id+" "+nm+" "+hx)
	}
	// DecryptKey đúng
	r := llmconfig.NewResolver(pool, c)
	k, err := r.DecryptKey(ctx, cc.ID)
	p("decrypt_C", fmt.Sprint(k == "third-key-abcdef", " ", err))
}
