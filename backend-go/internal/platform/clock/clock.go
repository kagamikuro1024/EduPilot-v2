// Package clock: đồng hồ và nguồn ngẫu nhiên có thể thay trong test (hết hạn, TTL, backoff không cần sleep thật).
package clock

import (
	"sync"
	"time"
)

// Clock cung cấp thời gian hiện tại.
type Clock interface{ Now() time.Time }

// Real là đồng hồ hệ thống.
type Real struct{}

// Now trả thời gian hệ thống (UTC).
func (Real) Now() time.Time { return time.Now().UTC() }

// Fake là đồng hồ chỉnh tay cho test; an toàn khi dùng song song.
type Fake struct {
	mu sync.Mutex
	t  time.Time
}

// NewFake tạo đồng hồ giả bắt đầu tại t.
func NewFake(t time.Time) *Fake { return &Fake{t: t.UTC()} }

// Now trả thời điểm hiện tại của đồng hồ giả.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Advance đẩy đồng hồ tiến d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

// Set đặt đồng hồ về t.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	f.t = t.UTC()
	f.mu.Unlock()
}
