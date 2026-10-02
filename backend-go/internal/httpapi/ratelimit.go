package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	goredis "github.com/redis/go-redis/v9"
)

// rateLimitTimeout là thời gian request CHỜ bộ đếm Redis; quá hạn thì cho qua (fail-open) để không
// treo quá 50 ms khi Redis chết (03-AC7).
const rateLimitTimeout = 30 * time.Millisecond

// rateLimitMaxWait là hạn sống của chính lệnh INCR (chạy tiếp sau khi request đã được cho qua):
// đủ dài để bộ đếm không hụt, đủ ngắn để Redis treo không giữ goroutine mãi.
const rateLimitMaxWait = 5 * time.Second

// rateLimitWarnEvery: tần suất tối đa của dòng log warn khi Redis chết (03-AC7).
const rateLimitWarnEvery = 10 * time.Second

// errRateLimitSlow: Redis chưa trả lời trong rateLimitTimeout — request được cho qua.
var errRateLimitSlow = errors.New("rate limit: Redis không trả lời trong hạn cho phép")

// rateLimitResult là kết quả của lần tăng bộ đếm chạy nền.
type rateLimitResult struct {
	counts []int64
	err    error
}

// rateLimitMiddleware (M6): bộ đếm cửa sổ phút cố định trên Redis, theo IP và theo người dùng,
// dùng chung giữa mọi bản gateway (SRS 5.6, 6.5). Redis chết → cho qua.
func rateLimitMiddleware(d Deps) func(http.Handler) http.Handler {
	warn := &throttledWarn{every: rateLimitWarnEvery}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit, remaining, err := rateLimitCount(r, d)
			if err != nil {
				warn.log(r.Context(), d.Log, "rate limit: Redis không phản hồi, cho qua (fail-open)", err)
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("X-RateLimit-Limit", strconv.Itoa(limit))
			h.Set("X-RateLimit-Remaining", strconv.Itoa(max(remaining, 0)))
			if remaining < 0 {
				apierr.Write(w, r, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).
					WithRetryAfter(secondsToNextWindow(d.now())))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitCount tăng bộ đếm IP (và bộ đếm người dùng nếu request có token hợp lệ) rồi trả về
// hạn mức chặt nhất: `remaining` < 0 nghĩa là đã vượt.
//
// Lệnh INCR chạy trên ctx KHÔNG huỷ theo request và chỉ việc CHỜ bị giới hạn bởi rateLimitTimeout:
// huỷ giữa chừng thì lệnh có thể đã tới Redis (đếm) hoặc bị thử lại (đếm hai lần) — cả hai đều làm
// bộ đếm sai. Quá hạn chờ ⇒ cho qua (fail-open) nhưng lần tăng vẫn được ghi nhận đúng một lần.
func rateLimitCount(r *http.Request, d Deps) (limit, remaining int, err error) {
	if d.Redis == nil {
		return 0, 0, errNoDependency
	}
	minute := strconv.FormatInt(d.now().Unix()/60, 10)
	keys := []string{appredis.Key("rl", "ip", clientIP(r, d), minute)}
	limits := []int{d.Cfg.RateLimitIPPerMin}
	if sub := tokenSubject(r, d); sub != "" {
		keys = append(keys, appredis.Key("rl", "user", sub, minute))
		limits = append(limits, d.Cfg.RateLimitUserPerMin)
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), rateLimitMaxWait)
	done := make(chan rateLimitResult, 1)
	go func() {
		defer cancel()
		counts, err := incrWindow(ctx, d, keys)
		done <- rateLimitResult{counts: counts, err: err}
	}()

	timer := time.NewTimer(rateLimitTimeout)
	defer timer.Stop()
	var counts []int64
	select {
	case res := <-done:
		if res.err != nil {
			return 0, 0, res.err
		}
		counts = res.counts
	case <-timer.C:
		return 0, 0, errRateLimitSlow
	}

	limit, remaining = limits[0], limits[0]-int(counts[0])
	for i := 1; i < len(keys); i++ {
		if rest := limits[i] - int(counts[i]); rest < remaining {
			limit, remaining = limits[i], rest
		}
	}
	return limit, remaining, nil
}

// incrWindow tăng mọi bộ đếm của cửa sổ phút hiện tại trong một chuyến đi và gia hạn TTL (SRS 5.6).
func incrWindow(ctx context.Context, d Deps, keys []string) ([]int64, error) {
	pipe := d.Redis.Pipeline()
	cmds := make([]*goredis.IntCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.Incr(ctx, k)
		pipe.Expire(ctx, k, appredis.TTLRateLimit)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	counts := make([]int64, len(cmds))
	for i, c := range cmds {
		counts[i] = c.Val()
	}
	return counts, nil
}

// secondsToNextWindow là số giây còn lại của cửa sổ phút hiện tại (1..60) — giá trị của `Retry-After`.
func secondsToNextWindow(now time.Time) int {
	if s := 60 - now.Second(); s > 0 {
		return s
	}
	return 1
}

// clientIP lấy IP để đếm: `X-Forwarded-For` CHỈ khi kết nối đến từ TRUSTED_PROXY_CIDRS (SRS 6.5).
func clientIP(r *http.Request, d Deps) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	addr = addr.Unmap()
	if !trustedProxy(addr, d.Cfg.TrustedProxyCIDRs) {
		return addr.String()
	}
	first, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")
	if fwd, err := netip.ParseAddr(strings.TrimSpace(first)); err == nil {
		return fwd.Unmap().String()
	}
	return addr.String()
}

func trustedProxy(addr netip.Addr, cidrs []netip.Prefix) bool {
	for _, c := range cidrs {
		if c.Contains(addr) {
			return true
		}
	}
	return false
}

// tokenSubject đọc `sub` từ Bearer token để đếm theo người dùng. Rate limit chạy TRƯỚC auth (SRS 3.1)
// nên tự kiểm chữ ký ở đây; token sai/thiếu → chỉ đếm theo IP.
func tokenSubject(r *http.Request, d Deps) string {
	if d.Verifier == nil {
		return ""
	}
	const scheme = "bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(scheme) || !strings.EqualFold(h[:len(scheme)], scheme) {
		return ""
	}
	p, err := d.Verifier.Verify(strings.TrimSpace(h[len(scheme):]))
	if err != nil {
		return ""
	}
	return p.Sub
}

// throttledWarn giới hạn tần suất một dòng log warn (không có biến toàn cục: sống trong closure middleware).
type throttledWarn struct {
	every time.Duration
	mu    sync.Mutex
	last  time.Time
}

func (t *throttledWarn) log(ctx context.Context, l *slog.Logger, msg string, err error) {
	if l == nil {
		return
	}
	now := time.Now()
	t.mu.Lock()
	if !t.last.IsZero() && now.Sub(t.last) < t.every {
		t.mu.Unlock()
		return
	}
	t.last = now
	t.mu.Unlock()
	l.WarnContext(ctx, msg, "error", err.Error())
}
