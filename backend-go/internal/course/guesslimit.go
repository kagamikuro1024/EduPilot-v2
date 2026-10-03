package course

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// Giới hạn đoán mã (SRS 4.2): chỉ đếm THẤT BẠI `JOIN_CODE_INVALID`; 5 lần / 10 phút / người dùng và 20 / 10 phút / IP, cửa sổ trượt.
const (
	guessWindow  = 10 * time.Minute
	guessPerUser = 5
	guessPerIP   = 20
	guessWarn    = 30 * time.Second
	// guessRedisTimeout chặn việc Redis chết kéo mỗi yêu cầu join chờ hạn dial mặc định (≈ 5 s): quá hạn ⇒ mở cửa như lỗi Redis.
	guessRedisTimeout = 250 * time.Millisecond
)

// ZSET: member = mã ngẫu nhiên của lần thử, score = ms. Khoá chỉ chứa định danh và số — không mã tham gia, email, MSSV.
const guessCheckLua = `
local now = tonumber(ARGV[1]); local win = tonumber(ARGV[2])
local worst = 0
for i = 1, #KEYS do
  redis.call('ZREMRANGEBYSCORE', KEYS[i], '-inf', now - win)
  local lim = tonumber(ARGV[2 + i])
  if redis.call('ZCARD', KEYS[i]) >= lim then
    local first = redis.call('ZRANGE', KEYS[i], 0, 0, 'WITHSCORES')
    local left = tonumber(first[2]) + win - now
    if left > worst then worst = left end
  end
end
return worst
`

const guessAddLua = `
local now = tonumber(ARGV[1]); local win = tonumber(ARGV[2])
for i = 1, #KEYS do
  redis.call('ZADD', KEYS[i], now, ARGV[2 + i])
  redis.call('PEXPIRE', KEYS[i], win)
end
return 1
`

type guessLimiter struct {
	rdb   *appredis.Client
	log   *slog.Logger
	check *goredis.Script
	add   *goredis.Script
	// lastWarn dùng CHUNG cho mọi yêu cầu của một Service (con trỏ do NewService cấp); mỗi bộ giới hạn là của MỘT yêu cầu.
	lastWarn *atomic.Int64
}

func newGuessLimiter(rdb *appredis.Client, log *slog.Logger, lastWarn *atomic.Int64) *guessLimiter {
	if lastWarn == nil {
		lastWarn = new(atomic.Int64) // Service dựng bằng literal (test): giới hạn log theo từng bộ
	}
	return &guessLimiter{rdb: rdb, log: log, check: goredis.NewScript(guessCheckLua), add: goredis.NewScript(guessAddLua), lastWarn: lastWarn}
}

func guessKeys(userID uuid.UUID, ip string) []string {
	keys := []string{appredis.Key("join", "fail", "user", userID.String())}
	if ip != "" {
		keys = append(keys, appredis.Key("join", "fail", "ip", ip))
	}
	return keys
}

func (g *guessLimiter) warn(ctx context.Context, op string, err error) {
	now := time.Now().UnixMilli()
	last := g.lastWarn.Load()
	if now-last < guessWarn.Milliseconds() || !g.lastWarn.CompareAndSwap(last, now) {
		return
	}
	g.log.ErrorContext(ctx, "join: Redis lỗi, giới hạn đoán mã mở cửa (fail-open)", "op", op, "error", err.Error())
}

// Blocked trả số giây còn phải chờ (> 0) nếu người dùng hoặc IP đã đủ số lần thất bại trong cửa sổ trượt. Redis lỗi ⇒ 0 (mở cửa), log ≤ 1 lần / 30 s.
func (g *guessLimiter) Blocked(ctx context.Context, userID uuid.UUID, ip string, now time.Time) int {
	if g == nil || g.rdb == nil {
		return 0
	}
	keys := guessKeys(userID, ip)
	args := []any{now.UnixMilli(), guessWindow.Milliseconds(), guessPerUser}
	if len(keys) > 1 {
		args = append(args, guessPerIP)
	}
	ctx, cancel := context.WithTimeout(ctx, guessRedisTimeout)
	defer cancel()
	ms, err := g.check.Run(ctx, g.rdb, keys, args...).Int64()
	if err != nil {
		g.warn(ctx, "check", err)
		return 0
	}
	if ms <= 0 {
		return 0
	}
	return int((ms + 999) / 1000)
}

// Fail ghi một lần thất bại.
func (g *guessLimiter) Fail(ctx context.Context, userID uuid.UUID, ip string, now time.Time) {
	if g == nil || g.rdb == nil {
		return
	}
	keys := guessKeys(userID, ip)
	args := []any{now.UnixMilli(), guessWindow.Milliseconds()}
	for range keys {
		args = append(args, fmt.Sprintf("%s-%s", strconv.FormatInt(now.UnixNano(), 36), uuid.NewString()[:8]))
	}
	ctx, cancel := context.WithTimeout(ctx, guessRedisTimeout)
	defer cancel()
	if err := g.add.Run(ctx, g.rdb, keys, args...).Err(); err != nil {
		g.warn(ctx, "add", err)
	}
}
