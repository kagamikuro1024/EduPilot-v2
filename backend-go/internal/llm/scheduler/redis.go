package scheduler

import (
	"context"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Mọi quyết định là MỘT script Lua; thời gian lấy bằng redis.call('TIME') trong script (client chỉ gửi khoảng thời gian) —
// đồng hồ client lệch 3 s làm bucket cấp 472.596 lượt thay vì 53 (docs/research/2026-10-03-scheduler-redis.md).
// Script.Run tự lùi về EVAL khi gặp NOSCRIPT; không gọi trong pipeline.
const leaseLua = `
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[1]) then return 0 end
if ARGV[5] == 'B' then
  local waiting = tonumber(redis.call('GET', KEYS[2]) or '0') or 0
  local last = tonumber(redis.call('GET', KEYS[3]) or '0') or 0
  local active = waiting > 0 or (now - last) < tonumber(ARGV[6])
  local b = 0
  for _, m in ipairs(redis.call('ZRANGE', KEYS[1], 0, -1)) do
    local c = string.sub(m, 1, 1)
    if c == 'B' then b = b + 1 end
    if c == 'I' then active = true end
  end
  if active and b >= tonumber(ARGV[2]) then return 0 end
end
redis.call('ZADD', KEYS[1], now + tonumber(ARGV[4]), ARGV[3])
redis.call('PEXPIRE', KEYS[1], 300000)
if ARGV[5] == 'I' then redis.call('SET', KEYS[3], tostring(now), 'PX', 5000) end
return 1`

const bucketLua = `
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local function refill(key, limit, cap)
  local d = redis.call('HMGET', key, 'tokens', 'ts')
  local tokens = tonumber(d[1])
  local ts = tonumber(d[2])
  if not tokens then tokens = cap; ts = now end
  if now > ts then tokens = math.min(cap, tokens + (now - ts) * limit / 60000); ts = now end
  return tokens, ts
end
local rl, rc, tl, tc, cost = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3]), tonumber(ARGV[4]), tonumber(ARGV[5])
local rt, rts = refill(KEYS[1], rl, rc)
local tt, tts = refill(KEYS[2], tl, tc)
local needR = math.min(1, rc)
local needT = math.min(cost, tc)
local wait = 0
if rt < needR then wait = math.max(wait, math.ceil((needR - rt) * 60000 / rl)) end
if tt < needT then wait = math.max(wait, math.ceil((needT - tt) * 60000 / tl)) end
if wait == 0 then rt = rt - 1; tt = tt - cost end
redis.call('HSET', KEYS[1], 'tokens', tostring(rt), 'ts', tostring(rts))
redis.call('HSET', KEYS[2], 'tokens', tostring(tt), 'ts', tostring(tts))
redis.call('PEXPIRE', KEYS[1], 120000)
redis.call('PEXPIRE', KEYS[2], 120000)
if wait == 0 then return {1, 0} end
return {0, wait}`

const reconcileLua = `
local d = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(d[1])
if not tokens then return 0 end
tokens = math.min(tonumber(ARGV[2]), tokens + tonumber(ARGV[1]))
redis.call('HSET', KEYS[1], 'tokens', tostring(tokens))
return 1`

const cbLua = `
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local op = ARGV[1]
local st = redis.call('HGET', KEYS[1], 'state') or 'closed'
if op == 'allow' then
  if st == 'closed' then return 1 end
  if st == 'open' then
    local o = tonumber(redis.call('HGET', KEYS[1], 'opened_at') or '0')
    if now - o >= tonumber(ARGV[3]) then
      redis.call('HSET', KEYS[1], 'state', 'half_open', 'probe_at', tostring(now))
      redis.call('PEXPIRE', KEYS[1], 600000)
      return 1
    end
    return 0
  end
  local pa = tonumber(redis.call('HGET', KEYS[1], 'probe_at') or '0')
  if now - pa >= 30000 then redis.call('HSET', KEYS[1], 'probe_at', tostring(now)); return 1 end
  return 0
elseif op == 'ok' then
  if st == 'closed' or st == 'half_open' then
    redis.call('HSET', KEYS[1], 'state', 'closed', 'fails', '0')
    redis.call('PEXPIRE', KEYS[1], 600000)
  end
  return 1
else
  if st == 'half_open' then
    redis.call('HSET', KEYS[1], 'state', 'open', 'opened_at', tostring(now))
  elseif st == 'closed' then
    local f = redis.call('HINCRBY', KEYS[1], 'fails', 1)
    if f >= tonumber(ARGV[2]) then redis.call('HSET', KEYS[1], 'state', 'open', 'opened_at', tostring(now)) end
  end
  redis.call('PEXPIRE', KEYS[1], 600000)
  return 1
end`

type redisBackend struct {
	rdb       *goredis.Client
	lease     *goredis.Script
	bucket    *goredis.Script
	reconc    *goredis.Script
	cb        *goredis.Script
	failsMax  int
	openFor   time.Duration
	intWindow time.Duration
	pfx       string // tiền tố cho khoá TOÀN CỤC (wait, lastint) — test cô lập
}

func newRedisBackend(rdb *goredis.Client, failsMax int, openFor time.Duration) *redisBackend {
	return &redisBackend{rdb: rdb, lease: goredis.NewScript(leaseLua), bucket: goredis.NewScript(bucketLua), reconc: goredis.NewScript(reconcileLua),
		cb: goredis.NewScript(cbLua), failsMax: failsMax, openFor: openFor, intWindow: 2 * time.Second}
}

func keyInflight(p string) string     { return "ep:llm:inflight:" + p }
func keyWait(pfx, lane string) string { return pfx + "ep:llm:wait:" + lane }
func keyRL(kind, p string) string     { return "ep:llm:rl:" + kind + ":" + p }
func keyCB(p string) string           { return "ep:llm:cb:" + p }

func (b *redisBackend) tryLease(ctx context.Context, r leaseReq) (bool, error) {
	n, err := b.lease.Run(ctx, b.rdb, []string{keyInflight(r.provider), keyWait(b.pfx, "INTERACTIVE"), b.pfx + "ep:llm:lastint"},
		r.max, r.batchCap, r.member, r.lease.Milliseconds(), r.lane, b.intWindow.Milliseconds()).Int()
	return n == 1, err
}

func (b *redisBackend) release(ctx context.Context, provider, member string) error {
	return b.rdb.ZRem(ctx, keyInflight(provider), member).Err()
}

func (b *redisBackend) tryBucket(ctx context.Context, r bucketReq) (bool, time.Duration, error) {
	res, err := b.bucket.Run(ctx, b.rdb, []string{keyRL("rpm", r.provider), keyRL("tpm", r.provider)},
		r.rpm, burst(r.rpm), r.tpm, burst(r.tpm), r.cost).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	return res[0] == 1, time.Duration(res[1]) * time.Millisecond, nil
}

func (b *redisBackend) reconcile(ctx context.Context, provider string, tpm, diff int) error {
	return b.reconc.Run(ctx, b.rdb, []string{keyRL("tpm", provider)}, diff, burst(tpm)).Err()
}

func (b *redisBackend) noteWaiting(ctx context.Context, lane string, delta int) error {
	k := keyWait(b.pfx, lane)
	pipe := b.rdb.Pipeline()
	incr := pipe.IncrBy(ctx, k, int64(delta))
	pipe.Expire(ctx, k, 60*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	if incr.Val() < 0 { // đếm gợi ý, không để âm
		return b.rdb.Set(ctx, k, 0, 60*time.Second).Err()
	}
	return nil
}

func (b *redisBackend) cbAllow(ctx context.Context, provider string) (bool, error) {
	n, err := b.cb.Run(ctx, b.rdb, []string{keyCB(provider)}, "allow", b.failsMax, b.openFor.Milliseconds()).Int()
	return n == 1, err
}

func (b *redisBackend) cbReport(ctx context.Context, provider string, failure bool) error {
	op := "ok"
	if failure {
		op = "fail"
	}
	return b.cb.Run(ctx, b.rdb, []string{keyCB(provider)}, op, b.failsMax, b.openFor.Milliseconds()).Err()
}

func (b *redisBackend) cbState(ctx context.Context, provider string) (string, error) {
	vals, err := b.rdb.HMGet(ctx, keyCB(provider), "state", "opened_at").Result()
	if err != nil {
		return "", err
	}
	st, _ := vals[0].(string)
	if st == "" {
		return "closed", nil
	}
	if st == "open" {
		if s, ok := vals[1].(string); ok {
			if o, perr := strconv.ParseInt(s, 10, 64); perr == nil {
				t, terr := b.rdb.Time(ctx).Result()
				if terr == nil && t.UnixMilli()-o >= b.openFor.Milliseconds() {
					return "half_open", nil
				}
			}
		}
	}
	return st, nil
}

func (b *redisBackend) counts(ctx context.Context, provider string) (int, int, error) {
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)
	// chỉ đếm phần tử còn hạn; đồng hồ client chỉ dùng cho thống kê (không quyết định cấp chỗ)
	ms, err := b.rdb.ZRangeArgs(ctx, goredis.ZRangeArgs{Key: keyInflight(provider), ByScore: true, Start: now, Stop: "+inf"}).Result()
	if err != nil {
		return 0, 0, err
	}
	batch := 0
	for _, m := range ms {
		if len(m) > 0 && m[:1] == laneB {
			batch++
		}
	}
	return len(ms), batch, nil
}
