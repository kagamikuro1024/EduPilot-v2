package main

import (
	"fmt"
	"os"
	"time"

	"github.com/edupilot/backend-go/internal/judge"
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
