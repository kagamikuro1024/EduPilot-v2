# Scheduler trên Redis: token bucket Lua (RPM + TPM) và ZSET lease cho inflight (P1, SRS FEAT-llm-gateway 4.3, 5.5)

**Câu hỏi.** Mẫu đúng cho token bucket RPM + TPM và chỗ inflight (ZSET lease) dùng chung giữa nhiều tiến trình bằng Lua trên Redis 8 với go-redis v9 là gì, và cạm bẫy nào (đồng hồ, `EVALSHA` sau khi Redis khởi động lại, hai tiến trình)?

**Kết luận.**
1. **Mỗi quyết định = một script Lua; thời gian lấy bằng `redis.call('TIME')` bên trong script.** Hạn thuê tính trong script: `now + lease_ms`; client chỉ gửi **khoảng thời gian** (`time.Until(deadline) + 10s`), không gửi mốc tuyệt đối. PoC: đồng hồ client lệch +3 s làm bucket cấp **472.596** lượt trong 10 s so với **53** khi dùng `TIME` — bucket mất tác dụng hoàn toàn.
2. **Hai tiến trình × 50 goroutine, `LLM_MAX_CONCURRENCY=10`:** cấp 2.865 chỗ, **không lúc nào > 10**. Bucket dùng chung cấp đúng 53 lượt (lý thuyết 53,3 = dung lượng 20 + 33,3 nạp lại).
3. **`EVALSHA` sau khi Redis mất cache script:** `Script.Run` của go-redis tự lùi về `EVAL` khi gặp `NOSCRIPT`. PoC `SCRIPT FLUSH` mỗi 500 ms: **0 lỗi**. Đừng gọi script trong pipeline (ở đó không lùi được).
4. **Cạm bẫy spec:** nếu dung lượng bucket TPM = burst 10 % (10.000 token với TPM 100.000), một yêu cầu ước tính lớn hơn dung lượng (GRADING `max_tokens` tới 32.768) **không bao giờ được cấp** → chờ tới hết hạn. Cách vá: chỉ đòi `min(cost, cap)` rồi cho bucket âm (nợ). PoC: đúng 1 lượt mỗi 12 s = đúng tốc độ TPM.
5. **Redis khởi động lại:** lỗi `EOF` / connection refused / `LOADING …` trong lúc nạp AOF (~95 lỗi mỗi tiến trình trong PoC); có AOF thì lease còn nguyên, vẫn không vượt 10. Phải coi `LOADING` như "Redis mất" (SRS 3.4).

Độ chắc chắn: **cao** (PoC hai tiến trình thật trên Redis 8.10.2 + mã go-redis v9.22.0).

## Phương án — nguồn thời gian

| Tiêu chí | `TIME` của Redis trong script (đề xuất) | Đồng hồ client truyền vào (`ARGV now_ms`) |
| --- | --- | --- |
| Nhiều gateway / máy lệch giờ | Đúng (một đồng hồ) | **Hỏng**: lệch 3 s → 472.596 lượt / 10 s thay vì 53 |
| Cùng máy, không lệch | Đúng | Đúng (54 lượt) |
| Test giả lập thời gian (`platform/clock`) | Không giả được; test dùng hạn mức nhỏ + thời gian thật | Giả được |
| Sao chép / AOF | Redis 7+ chỉ sao chép hiệu ứng của script nên `TIME` + ghi là hợp lệ; PoC chạy trên 8.10.2 | Không vấn đề |
| Khớp prompt PM | Có | Không |

Mẫu SSE hiện có (`internal/httpapi/sse/handler.go:77–83`, `takeSlot` truyền `h.clk.Now()`) thuộc loại thứ hai; chấp nhận được vì 2 gateway cùng máy, nhưng Scheduler **không** nên chép nguyên mẫu đó.

## Bằng chứng

- go-redis v9.22.0, `script.go:34–43`: `NewScript` tính SHA-1 phía Go và giữ trạng thái (`mu`, `hash`). `:193–197`: `Run` gọi `EvalSha`, gặp `NOSCRIPT` (`isNoScriptErr`) thì gọi `Eval` với nguyên mã → Redis tự nạp lại cache. Trong pipeline, kết quả chỉ có sau `Exec`, nên `Run` không thấy `NOSCRIPT` để lùi [SUY LUẬN từ cấu trúc mã].
- `NewScript` có trạng thái → giữ trong struct như SSE (`handler.go:91, 96`: `acquire *goredis.Script`), không đặt ở biến gói (lint `gochecknoglobals`, PG 07-AC11).
- Redis, [Scripting with Lua](https://redis.io/docs/latest/develop/programmability/eval-intro/): "In Redis 5.0, effects replication became the default mode. As of Redis 7.0, verbatim replication is no longer supported" → gọi `TIME` (không tất định) rồi ghi là hợp lệ; "The Redis script cache is **always volatile** … may be cleared when the server restarts, during fail-over … or explicitly by `SCRIPT FLUSH`" → phải chịu được `NOSCRIPT`. Kiểm thật: script dưới đây gọi `TIME` rồi `HSET` / `ZADD` trên `redis:8` (8.10.2), không lỗi.

PoC (`/tmp/research-sched/`; Redis 8.10.2 trong container, `--appendonly yes --appendfsync everysec` như compose; go-redis v9.22.0; Go 1.27.1). Mã: `main.go`, gồm hai script:

```lua
-- bucketLua: KEYS[1]=rpm, KEYS[2]=tpm; ARGV: rpm_limit, rpm_cap, tpm_limit, tpm_cap, cost, now_ms ("" = TIME)
local now = tonumber(ARGV[6])
if not now then local t = redis.call('TIME'); now = tonumber(t[1])*1000 + math.floor(tonumber(t[2])/1000) end
-- nạp lại: tokens = min(cap, tokens + (now - ts) * limit / 60000), chỉ khi now > ts
local need = math.min(cost, tc)          -- yêu cầu lớn hơn dung lượng: đòi bucket đầy rồi để âm
-- đủ cả hai → trừ cả hai, trả {1,0}; thiếu → trả {0, wait_ms} = max((thiếu) * 60000 / limit)
-- HSET tokens, ts; PEXPIRE 120000

-- leaseLua: KEYS[1]=ep:llm:inflight:<provider>; ARGV: max, req_id, lease_ms, now_ms ("" = TIME)
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
if redis.call('ZCARD', KEYS[1]) >= max then return 0 end
redis.call('ZADD', KEYS[1], now + lease_ms, req_id); redis.call('PEXPIRE', KEYS[1], 300000)
```

Lệnh: `docker run -d --name rsch-redis -p 16379:6379 redis:8 redis-server --appendonly yes --appendfsync everysec; cd /tmp/research-sched && go build -o sched . && bash run.sh`. Mỗi kịch bản chạy **2 tiến trình OS** song song (A, B), mỗi tiến trình 50 goroutine, 10 s. Kết quả thật:

```
== 1. lease, TIME của Redis, max=10
A mode=lease skew=-1: grants=1482 errs=0 max=10
B mode=lease skew=-1: grants=1383 errs=0 max=10
== 2. bucket RPM 600 (cap 60), TPM 100k (cap 10k), cost 500, TIME, 10 s
B grants=34   A grants=19                       → tổng 53
== 3. như 2 nhưng đồng hồ client, B lệch +3000 ms
B grants=254391   A grants=218205               → tổng 472.596
== 3b. như 2 nhưng đồng hồ client, không lệch
A grants=29   B grants=25                       → tổng 54
== 4. cost 20000 > cap TPM 10000, TIME, 10 s
B grants=0 max_wait=11997ms   A grants=1 max_wait=12000ms
== 5. lease + SCRIPT FLUSH mỗi 500 ms
B grants=1352 errs=0 max=10   A grants=1406 errs=0 max=10
== 6. lease, docker restart redis ở giây 4 (AOF everysec)
A grants=600 errs=97 max=10   B grants=527 errs=95 max=10
   lỗi gặp: EOF | LOADING Redis is loading the dataset in memory
```

Đọc kết quả. (2) TPM là giới hạn chặt hơn: 10.000/500 = 20 lượt ban đầu + 100.000 × 10/60/500 = 33,3 lượt nạp lại = 53,3, khớp 53. (3) Khi B ghi `ts` lệch +3 s, lần sau A thấy `now < ts` nên không nạp; B gọi tiếp thì thấy `ts` của A cũ hơn 3 s và nạp thêm 3 s token — cứ lặp như vậy, bucket coi như không tồn tại. (4) Không có dòng `need = min(cost, cap)` thì điều kiện `t ≥ 20000` không bao giờ đúng vì `t ≤ cap = 10000`; có dòng đó thì 1 lượt / 12 s = 100.000 TPM / 20.000.

## Ảnh hưởng (việc của dev / BA qua PM)

- `internal/llm/scheduler`: hai script như trên (`TIME` trong script, `ARGV` chỉ có hạn mức, chi phí, thời lượng), giữ `*redis.Script` trong struct, gọi `Run` (không trong pipeline). Đối soát TPM sau lời gọi: `HINCRBYFLOAT tokens (ước_tính − thật)` là đủ, vì lần nạp sau đã kẹp `min(cap, …)` [SUY LUẬN].
- **Cần PM / BA làm rõ SRS 4.3:** "dung lượng = hạn mức, burst = max(1, 10 % hạn mức)" có thể hiểu hai cách — (a) dung lượng = burst (tối đa 10 % mỗi lúc), (b) dung lượng = hạn mức cả phút. Theo (a) thì bắt buộc có quy tắc "yêu cầu lớn hơn dung lượng: chờ bucket đầy rồi cho âm", nếu không mọi yêu cầu ước tính > 10 % TPM sẽ chờ tới hết hạn rồi `OVERLOADED`. Ghi quy tắc này vào SRS 4.3.
- Chỗ inflight: score = `TIME + lease_ms` trong script, `lease_ms = time.Until(deadline) + 10s`. TTL khoá 300 s (SRS 5.5) phải luôn > lease dài nhất (BATCH 120 s + 10 s, đang thoả); nếu sau này nâng `LLM_REQUEST_TIMEOUT` lên > 290 s thì khoá có thể hết hạn khi còn lease.
- Chờ: bucket trả `wait_ms` → ngủ `min(wait_ms, thời gian còn lại)` + jitter. Lease đầy không có gợi ý thời gian → thăm dò 5–15 ms có jitter như PoC, hoặc phát pub/sub khi trả chỗ (chỉ làm khi đo thấy tải Redis đáng kể — `ponytail:`).
- Lỗi Redis: `LOADING` (đang nạp AOF sau khởi động lại), `EOF`, connection refused → nhánh "Redis mất" của SRS 3.4 (giới hạn cục bộ), không phải lỗi người dùng. Test 2 tiến trình (03-AC2, AC3) nên dùng 2 client riêng như PoC.
- Redis Cluster (không dùng hiện nay): mọi khoá của một script phải cùng slot → khi đó đặt hash tag `{<provider_id>}` vào tên khoá. Hiện chưa cần.
- Không đụng `ARCHITECTURE.md` / `DECISIONS.md`; không thêm thư viện.

## Đề xuất cho PM

`P1 Scheduler: token bucket + lease bằng Lua với redis.call('TIME') trong script (client chỉ gửi thời lượng) — PoC 2 tiến trình: inflight không vượt 10, bucket đúng 53/53,3, đồng hồ client lệch 3 s thì cấp 472.596 lượt; go-redis Script.Run chịu được SCRIPT FLUSH (0 lỗi), không dùng trong pipeline; sửa SRS 4.3: yêu cầu lớn hơn dung lượng bucket TPM được cấp khi bucket đầy rồi cho âm (nếu không GRADING ước tính > 10 % TPM chờ mãi) và làm rõ "dung lượng" = burst hay = hạn mức — docs/research/2026-10-03-scheduler-redis.md.`
