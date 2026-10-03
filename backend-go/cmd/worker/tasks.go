package main

import (
	"context"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

// Task là một việc nền chạy tới khi ctx bị huỷ (relay outbox, consumer Stream…).
// Task tự quản vòng lặp và span gốc của mình; trả về khi ctx huỷ để worker tắt trong ≤ 10 s.
type Task interface {
	Name() string
	Run(ctx context.Context) error
}

// newTasks trả relay + consumer outbox (US-PG-02): relay xếp hàng dòng tới hạn vào Stream
// `outbox.dispatch`, consumer chạy handler đăng ký trong newRegistry.
func newTasks(d Deps) []Task {
	od := outbox.Deps{Pool: d.DB, Redis: d.Redis, Log: d.Log, Clock: clock.Real{}, Cfg: d.Cfg}
	return []Task{outbox.NewRelay(od), outbox.NewConsumer(od, newRegistry(d))}
}
