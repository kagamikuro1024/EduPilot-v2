package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// Chờ tăng dần, khoá tạm và giới hạn theo IP/hành động (US-P2-05; SRS FEAT-account-security 4.2.2, 5.7).

// Limits là các ngưỡng (mặc định theo SRS 8.1; cấu hình bằng biến môi trường).
type Limits struct {
	LoginIPPerMin      int
	LoginIPFailPer15m  int
	RegisterIPPerHour  int
	ForgotIPPerHour    int
	ForgotEmailPerHour int
	TokenIPPerMin      int
	RefreshIPPerMin    int
	ChangePWFailPer10m int
	BackoffFrom        int // lần sai thứ n ≥ BackoffFrom bắt đầu phải chờ 2^(n-BackoffFrom+1) giây
	LockAt             int // lần sai thứ LockAt → khoá LockDuration
	LockDuration       time.Duration
}

// DefaultLimits trả ngưỡng mặc định của SRS 8.1.
func DefaultLimits() Limits {
	return Limits{
		LoginIPPerMin: 10, LoginIPFailPer15m: 30, RegisterIPPerHour: 5, ForgotIPPerHour: 5, ForgotEmailPerHour: 3,
		TokenIPPerMin: 20, RefreshIPPerMin: 60, ChangePWFailPer10m: 5, BackoffFrom: 5, LockAt: 10, LockDuration: 15 * time.Minute,
	}
}

const (
	failDecay   = 30 * time.Minute // bộ đếm sai hết hiệu lực nếu không sai thêm trong khoảng này
	ipFailWin   = 15 * time.Minute
	ipBlockFor  = 15 * time.Minute
	warnEvery   = 30 * time.Second
	failKeyHash = 32
)

// failState là ảnh chụp bộ đếm sai của một email.
type failState struct {
	N        int
	Last     time.Time // lần sai gần nhất
	LockedTo time.Time // zero = không khoá
}

// wait trả số giây còn phải chờ ở thời điểm now và lý do (khoá hay chờ tăng dần); 0 = được thử.
func (s failState) wait(now time.Time, lim Limits) (secs int, locked bool) {
	if !s.LockedTo.IsZero() {
		if now.Before(s.LockedTo) {
			return ceilSecs(s.LockedTo.Sub(now)), true
		}
		return 0, false // hết khoá: bộ đếm coi như về 0
	}
	if s.N < lim.BackoffFrom || now.Sub(s.Last) > failDecay {
		return 0, false
	}
	until := s.Last.Add(backoffFor(s.N, lim.BackoffFrom))
	if now.Before(until) {
		return ceilSecs(until.Sub(now)), false
	}
	return 0, false
}

// backoffFor: lần sai thứ n (≥ from) ⇒ 2^(n-from+1) giây: from=5 ⇒ 2, 4, 8, 16, 32.
func backoffFor(n, from int) time.Duration {
	return time.Duration(1<<min(n-from+1, 20)) * time.Second
}

func ceilSecs(d time.Duration) int { return max(int((d+time.Second-1)/time.Second), 1) }

// Throttle giữ bộ đếm sai theo băm email và theo IP trên Redis (dùng chung các bản gateway).
// Mọi lỗi Redis đều trả cho người gọi quyết định (đăng nhập: rơi về cột users.failed_logins / locked_until).
type Throttle struct {
	rdb    *appredis.Client
	lim    Limits
	log    *slog.Logger
	script *goredis.Script
}

func newThrottle(rdb *appredis.Client, lim Limits, log *slog.Logger) *Throttle {
	return &Throttle{rdb: rdb, lim: lim, log: log, script: goredis.NewScript(failLua)}
}

// failLua: tăng bộ đếm sai nguyên tử. Trả {n, lock_ms, vừa_khoá}. Khoá hết hạn hoặc 30 phút không sai ⇒ về 0 trước khi cộng.
const failLua = `
local k = KEYS[1]
local now, decay, lockAt, lockMs = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3]), tonumber(ARGV[4])
local n = tonumber(redis.call('HGET', k, 'n') or '0')
local last = tonumber(redis.call('HGET', k, 'last') or '0')
local lock = tonumber(redis.call('HGET', k, 'lock') or '0')
if lock > 0 and lock <= now then n = 0; lock = 0 end
if n > 0 and now - last > decay then n = 0; lock = 0 end
n = n + 1
local fresh = 0
if n >= lockAt then
  if lock <= now then fresh = 1 end
  lock = now + lockMs
end
redis.call('HSET', k, 'n', n, 'last', now, 'lock', lock)
local ttl = decay
if lock > now then ttl = math.max(decay, lock - now + 60000) end
redis.call('PEXPIRE', k, ttl)
return {n, lock, fresh}
`

func failKey(emailHash string) string {
	return appredis.Key("auth", "fail", emailHash[:failKeyHash])
}

// State đọc bộ đếm của emailHash. Chưa có/hết hạn ⇒ zero.
func (t *Throttle) State(ctx context.Context, emailHash string) (failState, error) {
	vals, err := t.rdb.HMGet(ctx, failKey(emailHash), "n", "last", "lock").Result()
	if err != nil {
		return failState{}, fmt.Errorf("auth: đọc bộ đếm sai: %w", err)
	}
	var st failState
	if s, ok := vals[0].(string); ok {
		st.N, _ = strconv.Atoi(s)
	}
	if s, ok := vals[1].(string); ok {
		if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
			st.Last = time.UnixMilli(ms)
		}
	}
	if s, ok := vals[2].(string); ok {
		if ms, err := strconv.ParseInt(s, 10, 64); err == nil && ms > 0 {
			st.LockedTo = time.UnixMilli(ms)
		}
	}
	return st, nil
}

// Fail ghi một lần sai; trả bộ đếm mới và newlyLocked = đúng lần này mới khoá (mỗi lần khoá chỉ một goroutine thấy true).
func (t *Throttle) Fail(ctx context.Context, emailHash string, now time.Time) (n int, lockedTo time.Time, newlyLocked bool, err error) {
	res, err := t.script.Run(ctx, t.rdb, []string{failKey(emailHash)}, now.UnixMilli(), failDecay.Milliseconds(), t.lim.LockAt, t.lim.LockDuration.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, time.Time{}, false, fmt.Errorf("auth: ghi lần sai: %w", err)
	}
	if len(res) != 3 {
		return 0, time.Time{}, false, errors.New("auth: kịch bản Redis trả sai dạng")
	}
	if res[1] > 0 {
		lockedTo = time.UnixMilli(res[1])
	}
	return int(res[0]), lockedTo, res[2] == 1, nil
}

// Clear xoá bộ đếm (đăng nhập đúng, đặt lại mật khẩu).
func (t *Throttle) Clear(ctx context.Context, emailHash string) error {
	if err := t.rdb.Del(ctx, failKey(emailHash)).Err(); err != nil {
		return fmt.Errorf("auth: xoá bộ đếm sai: %w", err)
	}
	return nil
}

func ipFailKey(ip string) string  { return appredis.Key("auth", "ipfail", ip) }
func ipBlockKey(ip string) string { return appredis.Key("auth", "blockip", ip) }

// IPBlocked trả số giây IP còn bị chặn (0 = không).
func (t *Throttle) IPBlocked(ctx context.Context, ip string) (int, error) {
	ttl, err := t.rdb.PTTL(ctx, ipBlockKey(ip)).Result()
	if err != nil {
		return 0, fmt.Errorf("auth: đọc chặn IP: %w", err)
	}
	if ttl <= 0 {
		return 0, nil
	}
	return ceilSecs(ttl), nil
}

// IPFail đếm một lần đăng nhập sai của IP; đủ ngưỡng thì chặn IP ipBlockFor.
func (t *Throttle) IPFail(ctx context.Context, ip string) error {
	pipe := t.rdb.Pipeline()
	n := pipe.Incr(ctx, ipFailKey(ip))
	pipe.ExpireNX(ctx, ipFailKey(ip), ipFailWin)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("auth: đếm sai theo IP: %w", err)
	}
	if int(n.Val()) >= t.lim.LoginIPFailPer15m {
		if err := t.rdb.Set(ctx, ipBlockKey(ip), "1", ipBlockFor).Err(); err != nil {
			return fmt.Errorf("auth: chặn IP: %w", err)
		}
	}
	return nil
}

// Limiter: cửa sổ cố định theo (hành động, IP) hoặc khoá tuỳ ý; dùng chung các bản gateway qua Redis.
type Limiter struct {
	rdb *appredis.Client
	log *slog.Logger
	now func() time.Time
}

// NewLimiter dựng Limiter; rdb nil = không giới hạn (test).
func NewLimiter(rdb *appredis.Client, now func() time.Time, log *slog.Logger) *Limiter {
	if log == nil {
		log = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Limiter{rdb: rdb, log: log, now: now}
}

// Window trả (khoá, giây tới hết cửa sổ) của cửa sổ cố định `window` chứa bây giờ: ep:rl:auth:{action}:ip:{ip}:{số cửa sổ}.
func (l *Limiter) window(action, subject string, window time.Duration) (string, int) {
	w := int64(window / time.Second)
	now := l.now().Unix()
	return appredis.Key("rl", "auth", action, "ip", subject, strconv.FormatInt(now/w, 10)), int(w - now%w)
}

// HitWindow đếm một lần trong cửa sổ cố định; over = vượt limit (retryAfter = giây tới cửa sổ sau). Redis lỗi ⇒ cho qua, log.
func (l *Limiter) HitWindow(ctx context.Context, action, subject string, window time.Duration, limit int) (retryAfter int, over bool) {
	if l == nil || l.rdb == nil {
		return 0, false
	}
	key, left := l.window(action, subject, window)
	pipe := l.rdb.Pipeline()
	n := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*window)
	if _, err := pipe.Exec(ctx); err != nil {
		l.log.WarnContext(ctx, "auth: không kiểm được giới hạn (Redis), cho qua (fail-open)", "action", action, "error", err.Error())
		return 0, false
	}
	if int(n.Val()) > limit {
		return left, true
	}
	return 0, false
}
