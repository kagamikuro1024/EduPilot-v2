// QC probe US-P1-03: single-flight qua llm.Gateway (HTTP route thử không có cờ Shareable). Máy chủ OpenAI giả của QC ở :9701.
// Chạy: copy vào backend-go/cmd/qcp103/ của worktree QC. In số lời gọi do 50 yêu cầu giống hệt gây ra theo từng cấu hình.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
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
	rt, err := llmrt.New(ctx, cfg, pool, rdb, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	if err != nil {
		panic(err)
	}
	g := rt.Gateway
	run := func(name string, task llm.Task, lane *llm.Lane, share bool, prompt string, cancelOne bool) {
		var wg sync.WaitGroup
		res := make([]string, 50)
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c := ctx
				if cancelOne && i == 0 {
					var cc context.CancelFunc
					c, cc = context.WithTimeout(ctx, 300*time.Millisecond)
					defer cc()
				}
				r, err := g.Chat(c, llm.Request{Task: task, Lane: lane, Shareable: share, Messages: []llm.Message{{Role: "user", Content: prompt}}})
				if err != nil {
					res[i] = "ERR:" + err.Error()
					return
				}
				res[i] = r.Text
			}(i)
		}
		wg.Wait()
		same, errs := 0, 0
		for _, s := range res {
			if len(s) > 3 && s[:4] == "ERR:" {
				errs++
			} else if s == res[1] {
				same++
			}
		}
		fmt.Printf("SF %s: results identical=%d errs=%d\n", name, same, errs)
	}
	nr := llm.LaneNearRealtime
	run("NEAR_REALTIME shareable", llm.TaskClassify, &nr, true, "SF1", false)
	time.Sleep(200 * time.Millisecond)
	fmt.Println("SF-marker-after-1")
	time.Sleep(3 * time.Second)
	run("NEAR_REALTIME NOT shareable", llm.TaskClassify, &nr, false, "SF2", false)
	time.Sleep(3 * time.Second)
	run("INTERACTIVE shareable", llm.TaskChat, nil, true, "SF3", false)
	time.Sleep(3 * time.Second)
	run("GRADING shareable", llm.TaskGrading, nil, true, "SF4", false)
	time.Sleep(3 * time.Second)
	run("NEAR_REALTIME shareable, 1 waiter cancels", llm.TaskClassify, &nr, true, "SF5", true)
	rt.Close(ctx)
}
