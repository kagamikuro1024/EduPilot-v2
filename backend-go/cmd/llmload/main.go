// Command llmload: công cụ đo Scheduler của cổng LLM qua route thử `_test/llm/chat` (US-P1-03 AC4, "Bạn tự kiểm" của P1).
// KHÔNG nằm trong image (Dockerfile chỉ dựng gateway và worker). Cần gateway dựng bằng build tag testroutes và JWT ADMIN.
//
//	go run ./cmd/llmload --base https://localhost --secret "$JWT_SECRET_KEY" --batch 200 --chat 25
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

type result struct {
	status  int
	waitMS  int64
	elapsed time.Duration
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("llmload", flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "https://localhost", "địa chỉ gateway (bản dựng testroutes)")
	token := fs.String("token", "", "JWT ADMIN (hoặc dùng --secret)")
	secret := fs.String("secret", os.Getenv("JWT_SECRET_KEY"), "JWT_SECRET_KEY để ký token ADMIN")
	batch := fs.Int("batch", 200, "số việc BATCH (INSIGHT) bơm trước")
	chat := fs.Int("chat", 25, "số yêu cầu INTERACTIVE (CHAT)")
	conc := fs.Int("chat-concurrency", 5, "số CHAT tối đa cùng lúc (≤ 50 % công suất)")
	gap := fs.Duration("chat-gap", 100*time.Millisecond, "khoảng cách giữa hai lần phát CHAT")
	warm := fs.Duration("warmup", 500*time.Millisecond, "chờ sau khi bơm BATCH rồi mới phát CHAT")
	maxWait := fs.Int64("max-wait-p95-ms", 500, "ngưỡng interactive_wait_p95_ms")
	maxBatch := fs.Int("max-batch", 5, "ngưỡng batch_peak")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	tok := *token
	if tok == "" {
		if *secret == "" {
			_, _ = fmt.Fprintln(stderr, "cần --token hoặc --secret / JWT_SECRET_KEY")
			return 2
		}
		var err error
		tok, err = auth.NewIssuer(*secret, time.Hour, clock.Real{}).Issue("00000000-0000-7000-8000-0000000000a0", auth.RoleAdmin, "")
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "không ký được token:", err)
			return 2
		}
	}
	cl := &http.Client{Timeout: 10 * time.Minute, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, MaxIdleConnsPerHost: 512}} //nolint:gosec // công cụ đo, cert tự ký của Caddy dev
	post := func(ctx context.Context, task, prompt string) result {
		body, _ := json.Marshal(map[string]any{"task": task, "prompt": prompt})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, *base+"/api/v1/_test/llm/chat", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		start := time.Now()
		resp, err := cl.Do(req)
		if err != nil {
			return result{status: 0, elapsed: time.Since(start)}
		}
		defer func() { _ = resp.Body.Close() }()
		var out struct {
			QueueWaitMS int64 `json:"queue_wait_ms"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return result{status: resp.StatusCode, waitMS: out.QueueWaitMS, elapsed: time.Since(start)}
	}
	stats := func(ctx context.Context) (batchInflight int) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, *base+"/api/v1/_test/llm/stats?lanes=1", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := cl.Do(req)
		if err != nil {
			return 0
		}
		defer func() { _ = resp.Body.Close() }()
		var st struct {
			InflightBatch map[string]int `json:"inflight_batch"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&st)
		for _, n := range st.InflightBatch {
			batchInflight += n
		}
		return batchInflight
	}

	ctx := context.Background()
	var peak atomic.Int64
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				if n := int64(stats(ctx)); n > peak.Load() {
					peak.Store(n)
				}
				time.Sleep(500 * time.Millisecond) // thăm dò thưa: mỗi lần là một request tính vào RATE_LIMIT_USER_PER_MIN
			}
		}
	}()

	var bwg sync.WaitGroup
	var batchOK, batchRejected atomic.Int64
	for i := range *batch {
		bwg.Add(1)
		go func() {
			defer bwg.Done()
			switch post(ctx, "INSIGHT", fmt.Sprintf("batch %d", i)).status {
			case http.StatusOK:
				batchOK.Add(1)
			case http.StatusServiceUnavailable:
				batchRejected.Add(1)
			}
		}()
	}
	time.Sleep(*warm)

	var mu sync.Mutex
	var waits []int64
	var cwg sync.WaitGroup
	sem := make(chan struct{}, max(*conc, 1))
	var chatOK, chatFail atomic.Int64
	var stMu sync.Mutex
	statuses := map[int]int{}
	for i := range *chat {
		sem <- struct{}{}
		cwg.Add(1)
		go func() {
			defer cwg.Done()
			defer func() { <-sem }()
			r := post(ctx, "CHAT", fmt.Sprintf("hỏi %d", i))
			stMu.Lock()
			statuses[r.status]++
			stMu.Unlock()
			if r.status != 200 {
				chatFail.Add(1)
				return
			}
			chatOK.Add(1)
			mu.Lock()
			waits = append(waits, r.waitMS)
			mu.Unlock()
		}()
		time.Sleep(*gap)
	}
	cwg.Wait()
	close(stop)
	bwg.Wait()

	slices.Sort(waits)
	var p95 int64
	if len(waits) > 0 {
		p95 = waits[max(int(float64(len(waits))*0.95)-1, 0)]
	}
	lines := []string{
		fmt.Sprintf("interactive_wait_p95_ms=%d", p95),
		fmt.Sprintf("batch_peak=%d", peak.Load()),
		fmt.Sprintf("chat_ok=%d chat_fail=%d batch_ok=%d batch_rejected=%d", chatOK.Load(), chatFail.Load(), batchOK.Load(), batchRejected.Load()),
		fmt.Sprintf("chat_status=%v (429 = chạm RATE_LIMIT_USER_PER_MIN: nâng biến này khi đo)", statuses),
	}
	_, _ = fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	if p95 > *maxWait || peak.Load() > int64(*maxBatch) || chatFail.Load() > 0 {
		_, _ = fmt.Fprintln(stderr, "KHÔNG ĐẠT: vượt ngưỡng (interactive_wait_p95_ms ≤", *maxWait, ", batch_peak ≤", *maxBatch, ", chat_fail = 0)")
		return 1
	}
	return 0
}
