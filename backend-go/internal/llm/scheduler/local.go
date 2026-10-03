package scheduler

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/edupilot/backend-go/internal/platform/clock"
)

// localBackend: trạng thái trong tiến trình — dùng khi Redis mất (mỗi tiến trình áp nguyên giá trị cấu hình) và trong test đơn vị.
type localBackend struct {
	mu       sync.Mutex
	clk      clock.Clock
	failsMax int
	openFor  time.Duration
	leases   map[string]map[string]time.Time // provider → member → hạn
	lastInt  time.Time
	waitingI int
	buckets  map[string]*bucket
	cb       map[string]*cbState
}

type bucket struct {
	tokens float64
	ts     time.Time
	init   bool
}

type cbState struct {
	state    string
	fails    int
	openedAt time.Time
	probeAt  time.Time
}

func newLocalBackend(clk clock.Clock, failsMax int, openFor time.Duration) *localBackend {
	return &localBackend{clk: clk, failsMax: failsMax, openFor: openFor, leases: map[string]map[string]time.Time{},
		buckets: map[string]*bucket{}, cb: map[string]*cbState{}}
}

func (l *localBackend) prune(provider string, now time.Time) map[string]time.Time {
	m := l.leases[provider]
	if m == nil {
		m = map[string]time.Time{}
		l.leases[provider] = m
	}
	for k, exp := range m {
		if !exp.After(now) {
			delete(m, k)
		}
	}
	return m
}

func (l *localBackend) tryLease(_ context.Context, r leaseReq) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clk.Now()
	m := l.prune(r.provider, now)
	if len(m) >= r.max {
		return false, nil
	}
	if r.lane == laneB {
		active := l.waitingI > 0 || now.Sub(l.lastInt) < 2*time.Second
		b := 0
		for k := range m {
			switch k[:1] {
			case laneB:
				b++
			case laneI:
				active = true
			}
		}
		if active && b >= r.batchCap {
			return false, nil
		}
	}
	m[r.member] = now.Add(r.lease)
	if r.lane == laneI {
		l.lastInt = now
	}
	return true, nil
}

func (l *localBackend) release(_ context.Context, provider, member string) error {
	l.mu.Lock()
	delete(l.leases[provider], member)
	l.mu.Unlock()
	return nil
}

func (l *localBackend) refill(key string, limit, cap int, now time.Time) *bucket {
	b := l.buckets[key]
	if b == nil {
		b = &bucket{}
		l.buckets[key] = b
	}
	if !b.init {
		b.tokens, b.ts, b.init = float64(cap), now, true
	}
	if now.After(b.ts) {
		b.tokens = math.Min(float64(cap), b.tokens+float64(now.Sub(b.ts).Milliseconds())*float64(limit)/60000)
		b.ts = now
	}
	return b
}

func (l *localBackend) tryBucket(_ context.Context, r bucketReq) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clk.Now()
	rb := l.refill("rpm:"+r.provider, r.rpm, burst(r.rpm), now)
	tb := l.refill("tpm:"+r.provider, r.tpm, burst(r.tpm), now)
	needR := math.Min(1, float64(burst(r.rpm)))
	needT := math.Min(float64(r.cost), float64(burst(r.tpm)))
	var wait float64
	if rb.tokens < needR {
		wait = math.Max(wait, math.Ceil((needR-rb.tokens)*60000/float64(r.rpm)))
	}
	if tb.tokens < needT {
		wait = math.Max(wait, math.Ceil((needT-tb.tokens)*60000/float64(r.tpm)))
	}
	if wait > 0 {
		return false, time.Duration(wait) * time.Millisecond, nil
	}
	rb.tokens--
	tb.tokens -= float64(r.cost)
	return true, 0, nil
}

func (l *localBackend) reconcile(_ context.Context, provider string, tpm, diff int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if b := l.buckets["tpm:"+provider]; b != nil {
		b.tokens = math.Min(float64(burst(tpm)), b.tokens+float64(diff))
	}
	return nil
}

func (l *localBackend) noteWaiting(_ context.Context, lane string, delta int) error {
	if lane == "INTERACTIVE" {
		l.mu.Lock()
		l.waitingI = max(0, l.waitingI+delta)
		l.mu.Unlock()
	}
	return nil
}

func (l *localBackend) state(provider string) *cbState {
	s := l.cb[provider]
	if s == nil {
		s = &cbState{state: "closed"}
		l.cb[provider] = s
	}
	return s
}

func (l *localBackend) cbAllow(_ context.Context, provider string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, now := l.state(provider), l.clk.Now()
	switch s.state {
	case "closed":
		return true, nil
	case "open":
		if now.Sub(s.openedAt) >= l.openFor {
			s.state, s.probeAt = "half_open", now
			return true, nil
		}
		return false, nil
	}
	if now.Sub(s.probeAt) >= 30*time.Second {
		s.probeAt = now
		return true, nil
	}
	return false, nil
}

func (l *localBackend) cbReport(_ context.Context, provider string, failure bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, now := l.state(provider), l.clk.Now()
	switch {
	case !failure && (s.state == "closed" || s.state == "half_open"):
		s.state, s.fails = "closed", 0
	case failure && s.state == "half_open":
		s.state, s.openedAt = "open", now
	case failure && s.state == "closed":
		s.fails++
		if s.fails >= l.failsMax {
			s.state, s.openedAt = "open", now
		}
	}
	return nil
}

func (l *localBackend) cbState(_ context.Context, provider string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.state(provider)
	if s.state == "open" && l.clk.Now().Sub(s.openedAt) >= l.openFor {
		return "half_open", nil
	}
	return s.state, nil
}

func (l *localBackend) counts(_ context.Context, provider string) (int, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := l.prune(provider, l.clk.Now())
	b := 0
	for k := range m {
		if k[:1] == laneB {
			b++
		}
	}
	return len(m), b, nil
}
