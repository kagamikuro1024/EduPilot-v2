package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/clock"
)

// newJudgeQueue dựng hàng chấm: bản `JUDGE_CONSUMER=true` có client sandbox + consumer; bản khác chỉ có phần đưa tín hiệu (outbox `judge.enqueue`).
func newJudgeQueue(d Deps, s judge.Settings) (*judge.Queue, error) {
	host, _ := os.Hostname()
	q := &judge.Queue{Pool: d.DB, Redis: d.Redis, Log: d.Log, Clock: clock.Real{}, P: s.Parallelism, Name: "judge-" + host,
		Lease: s.Lease, Renew: s.LeaseRenew, Idle: s.ClaimIdle, TickEvery: 5 * time.Second, Probe: 10 * time.Second}
	if !s.Consumer {
		return q, nil
	}
	c, err := judge.NewClient(judge.Config{URL: s.URL, Token: s.Token, Slack: s.HTTPSlack})
	if err != nil {
		return nil, fmt.Errorf("JUDGE_TOKEN / JUDGE_URL: %w", err)
	}
	q.Judge = c
	return q, nil
}

// attachExamDeps nối các phụ thuộc của việc nền thi hằng tuần. Thiếu một phần (không token judge, không khoá AI, không kho đối tượng) KHÔNG làm worker thoát:
// việc cần phần đó báo FAILED bằng câu tiếng Việt, phần còn lại chạy bình thường.
func (d *Deps) attachExamDeps(ctx context.Context, s judge.Settings) {
	if len(s.Token) >= judge.MinTokenLen {
		if c, err := judge.NewClient(judge.Config{URL: s.URL, Token: s.Token, Slack: s.HTTPSlack}); err == nil {
			d.Sandbox = c
		}
	}
	if st, err := blob.New(ctx, blob.Config{Endpoint: d.Cfg.BlobEndpoint, PublicEndpoint: d.Cfg.BlobPublicEndpoint, Bucket: d.Cfg.BlobBucket, AccessKey: d.Cfg.BlobAccessKey,
		SecretKey: d.Cfg.BlobSecretKey, Region: d.Cfg.BlobRegion, UseSSL: d.Cfg.BlobUseSSL}); err == nil {
		d.Blob = st
	} else {
		d.Log.WarnContext(ctx, "chưa dùng được kho đối tượng", "error", err.Error())
	}
	rt, err := llmrt.New(ctx, d.Cfg, d.DB, d.Redis.Client, d.Log)
	if err != nil {
		d.Log.WarnContext(ctx, "chưa dựng được cổng LLM cho việc nền", "error", err.Error())
		return
	}
	d.llmRT, d.LLM = rt, rt.Gateway
}

func (d *Deps) closeExamDeps(ctx context.Context) {
	if d.llmRT != nil {
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), probeTimeout)
		defer cancel()
		d.llmRT.Close(cctx)
	}
}
