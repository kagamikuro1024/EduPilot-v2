package main

import "context"

// Task là một việc nền chạy tới khi ctx bị huỷ (relay outbox, consumer Stream…).
// Task tự quản vòng lặp và span gốc của mình; trả về khi ctx huỷ để worker tắt trong ≤ 10 s.
type Task interface {
	Name() string
	Run(ctx context.Context) error
}

// newTasks là ĐIỂM CẮM của US-PG-02: trả về relay + consumer outbox, ví dụ
//
//	return []Task{outbox.NewRelay(d.DB, d.Redis, d.Log, d.Cfg), outbox.NewConsumer(d.DB, d.Redis, d.Log, d.Cfg)}
//
// US-PG-01 chưa có việc nghiệp vụ nào: worker chỉ chạy nhịp + `/healthz`.
func newTasks(_ Deps) []Task { return nil }
