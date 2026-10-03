// QC: dựng cấu hình nhà cung cấp / tuyến từ JSON (stdin) bằng llmconfig.Service rồi PUBLISH ep:llm:reload.
// Chạy: copy vào backend-go/cmd/qcp102 của worktree QC. Xoá sạch cấu hình LLM trước khi dựng.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type spec struct {
	Providers []struct {
		Name, Type, BaseURL, Key string
		Enabled                  *bool
		Models                   []struct {
			Model, Kind string
			Dims        *int
		}
	}
	Routes []struct {
		Task   string
		Chain  []string // "provider/model"
		Params map[string]json.Number
	}
}

func main() {
	ctx := auth.WithPrincipal(context.Background(), auth.Principal{Sub: "00000000-0000-7000-8000-0000000000a0", Role: auth.RoleAdmin})
	pool, _ := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	c, _ := crypto.ParseKey(os.Getenv("APP_ENCRYPTION_KEY"))
	svc := llmconfig.New(pool, c)
	for _, q := range []string{"delete from llm_task_routes", "delete from llm_models", "delete from llm_providers"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			panic(err)
		}
	}
	var sp spec
	b, _ := io.ReadAll(os.Stdin)
	if err := json.Unmarshal(b, &sp); err != nil {
		panic(err)
	}
	ids := map[string]uuid.UUID{}
	for _, p := range sp.Providers {
		in := llmconfig.ProviderInput{Type: p.Type, Name: p.Name, Enabled: p.Enabled}
		if p.BaseURL != "" {
			u := p.BaseURL
			in.BaseURL = &u
		}
		if p.Key != "" {
			s := llmconfig.NewSecret(p.Key)
			in.APIKey = &s
		}
		for _, m := range p.Models {
			in.Models = append(in.Models, llmconfig.ModelInput{Model: m.Model, Kind: m.Kind, Dims: m.Dims, PriceIn: decimal.NewFromInt(1000), PriceOut: decimal.NewFromInt(2000)})
		}
		pv, err := svc.CreateProvider(ctx, in)
		if err != nil {
			panic(fmt.Sprint(p.Name, ": ", err))
		}
		for _, m := range pv.Models {
			ids[p.Name+"/"+m.Model] = m.ID
		}
	}
	for _, r := range sp.Routes {
		var ch []uuid.UUID
		for _, k := range r.Chain {
			ch = append(ch, ids[k])
		}
		if _, err := svc.SetRoute(ctx, r.Task, ch, llmconfig.Params(r.Params), 0); err != nil {
			panic(fmt.Sprint(r.Task, ": ", err))
		}
	}
	rc := redis.NewClient(&redis.Options{Addr: "localhost:46379"})
	fmt.Println("published", rc.Publish(ctx, "ep:llm:reload", "x").Val())
}
