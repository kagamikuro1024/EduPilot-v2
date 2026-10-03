// QC probe: gọi llm.Gateway.Structured / Embed (không có đường HTTP cho hai hàm này ở US-P1-02). In kết quả từng ca.
// Chạy: copy vào backend-go/cmd/qcp102b/ của worktree QC; cần máy chủ OpenAI giả của QC (Bun) ở :9701 và cấu hình DB đã dựng bằng p102-setup.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load(os.Getenv, config.Gateway)
	if err != nil {
		panic(err)
	}
	pool, _ := pgxpool.New(ctx, cfg.DatabaseURL)
	rdb := goredis.NewClient(&goredis.Options{Addr: "localhost:46379"})
	lg := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	rt, err := llm.NewRuntime(ctx, cfg, pool, rdb, lg)
	if err != nil {
		panic(err)
	}
	g := rt.Gateway
	schema := json.RawMessage(`{"type":"object","properties":{"category":{"type":"string"},"confidence":{"type":"number"}},"required":["category"],"additionalProperties":false}`)
	st := func(name, prompt string) {
		ctx2, c := context.WithTimeout(ctx, 20*time.Second)
		defer c()
		out, err := g.Structured(ctx2, llm.Request{Task: llm.TaskClassify, Messages: []llm.Message{{Role: "user", Content: prompt}}}, schema)
		fmt.Printf("STRUCT %s out=%s err=%v\n", name, string(out), err)
	}
	st("valid", "phan loai cau hoi")
	st("badschema", "BADSCHEMA")
	st("notjson", "NOTJSON")
	st("json400", "JSONFMT400")
	em := func(name string, in []string) {
		v, err := g.Embed(ctx, llm.EmbedRequest{Inputs: in})
		d := 0
		if len(v) > 0 {
			d = len(v[0])
		}
		fmt.Printf("EMBED %s n=%d dim=%d err=%v\n", name, len(v), d, err)
	}
	em("ok", []string{"a", "b"})
	em("dim768", []string{"DIM768"})
	em("empty", []string{""})
	big := make([]string, 250)
	for i := range big {
		big[i] = fmt.Sprint("s", i)
	}
	em("250", big)
	em("none", nil)
	_ = strings.Join
	rt.Close(ctx)
}
