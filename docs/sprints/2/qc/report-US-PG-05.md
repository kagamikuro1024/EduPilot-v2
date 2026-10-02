# QC report — US-PG-05 (SSE) · sprint 2 (PG Nền Go) · Kết luận: **FAIL**

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `pg05.sh` (+ `QC_SLOW=1 pg05.sh 25 27`) (lượt đầu → sau triage + sửa script: 59/74 (+2 MANUAL) → 72/74). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 72/74 TC PASS, **2 FAIL** — lỗi sản phẩm / TC: BUG-PG-1, BUG-PG-2.

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG05-01 | PASS | AC1 | 13 byte đầu (retry + dòng trống) = [retry: 3000@@] ; dòng 3 = sự kiện trạng thái đầu = [event: ready] (+4 dòng ok) |
| TC-PG05-02 | PASS | AC1 / SRS 6.8 | mã HTTP = [200] ; Content-Type ~ /^text/event-stream/ (+5 dòng ok) — _sửa script QC: `Connection` do gateway phát, Caddy bỏ (hop-by-hop) → đo thẳng gateway_ |
| TC-PG05-03 | PASS | AC1 | số kết nối đo được = [50] ; p95 time_starttransfer (ms) = 34 (≤ 300) |
| TC-PG05-04 | PASS | AC1 | connection_id kết nối 1 không rỗng = [deab2980-bc10-45e0-b112-83a30ebff080] (≠ []) ; connection_id khác nhau giữa 2 kết nối = [deab2980-bc10-45e0-b112-83a30ebff080] (≠ [ee491acd-4a5f-4879-b98a-fb (+2 dòng ok) |
| TC-PG05-05 | PASS | AC2 | số sự kiện nhận được = [20] ; số khối đúng định dạng (id/event/data/dòng trống) = [20] (+2 dòng ok) — _sửa script QC: `sse_burst`: `wait` trần bị chặn bởi curl SSE nền_ |
| TC-PG05-06 | PASS | AC2 | số id sai định dạng ^\d+-\d+$ = [0] ; id tăng nghiêm ngặt = [ok] (+1 dòng ok) |
| TC-PG05-07 | PASS | AC2 (biên) | mã khi type 1 ký tự 'a' ~ /^2[0-9][0-9]$/ ; mã khi type 64 ký tự ~ /^2[0-9][0-9]$/ (+3 dòng ok) |
| TC-PG05-08 | PASS | AC2 (nhánh lỗi, biên) | type [Bad Type] → mã HTTP = [422] ; type [Bad Type] → code = [VALIDATION_FAILED] (+18 dòng ok) |
| TC-PG05-09 | PASS | AC2 (biên) | data 60.000 byte → chấp nhận ~ /^2[0-9][0-9]$/ ; data 66.000 byte → mã HTTP = [422] (+3 dòng ok) |
| TC-PG05-10 | PASS | AC2 | go test ./internal/httpapi/sse -run 'TestSSE_Framing\|TestPublish_Validation\|TestSSE_IDsMonotonic' rc = [0] ; --- PASS: TestSSE_Framing (+2 dòng ok) |
| TC-PG05-11 | PASS | AC3 | số dòng ': hb' trong 30 s (SSE_HEARTBEAT = 25 s) = [1] |
| TC-PG05-12 | PASS | AC3 | số nhịp ': hb' trong 55 s = 2 (≥ 2) ; nhịp 1 sau ready (ms) = 25006 (≥ 23000) (+4 dòng ok) |
| TC-PG05-13 | PASS | AC3 | go test ./internal/httpapi/sse -run 'TestSSE_Heartbeat\|TestConfig_SSEDefaults' rc = [0] ; --- PASS: TestSSE_Heartbeat (+1 dòng ok) |
| TC-PG05-14 | PASS | AC4 (nhánh lỗi) | dọn sạch trong ≤ 2 s sau chu kỳ cuối ; ZCARD ep:sse:conn:<uid> = [0] (+1 dòng ok) |
| TC-PG05-15 | PASS | AC4 (nhánh lỗi) | ZCARD + NUMSUB về 0 trong ≤ 2 s ; ZCARD ep:sse:conn:<uid> = [0] |
| TC-PG05-16 | PASS | AC4 | số dòng log mức error trong 60 s qua = [0] |
| TC-PG05-17 | PASS | AC4 | go test ./internal/httpapi/sse -run 'TestSSE_NoLeakOnClientDisconnect' rc = [0] ; --- PASS: TestSSE_NoLeakOnClientDisconnect |
| TC-PG05-18 | PASS | AC5 | số mã 000/200 (hai stream được giữ; curl 8.7 in 200 khi hết -m giữa thân) = [2] ; số mã 429 (stream thứ 3) = [1] — _sửa script QC: curl 8.7 in `200` khi hết `-m` giữa thân_ |
| TC-PG05-19 | PASS | AC5 (nhánh lỗi) | mã HTTP stream thứ 3 = [429] ; code trong thân = [SSE_LIMIT_REACHED] (+6 dòng ok) |
| TC-PG05-20 | PASS | AC5 | mã của U2 (đang stream → 000/200) ~ /^(000\|200)$/ ; mã của U1 (stream thứ 3 → 429) = [429] — _sửa script QC: như 18_ |
| TC-PG05-21 | PASS | AC5 | trước khi đóng: stream thứ 3 = [429] ; sau khi đóng 1 stream: mở được trong ≤ 2 s = [1] — _sửa script QC: như 18_ |
| TC-PG05-22 | PASS | AC5 | có hai bản gateway = [a18bc0121dc8e0d6f010f2afffbb98a6e0bccf84627a09090f29f18e8a3a2415] (≠ []) ; ZCARD sau khi A giữ 2 kết nối = [2] (+1 dòng ok) |
| TC-PG05-23 | PASS | AC5 / SRS 5.6 | số member (1 kết nối) = [1] ; score − now (ms) ≈ SSE_CONN_TTL = 149772 (≥ 140000) (+1 dòng ok) — _sửa script QC: đếm member trước khi cắt stream_ |
| TC-PG05-24 | PASS | AC5 | ZCARD khi bản A giữ 2 kết nối = [2] ; ZCARD về 0 trong ≤ 12 s với SSE_CONN_TTL=5 (1 = đúng) = [1] |
| TC-PG05-25 | PASS | AC5 | thời gian chỗ hết hạn (ms) = 148059 (≥ 140000) ; thời gian chỗ hết hạn (ms) = 148059 (≤ 160000) — _sửa script QC: chạy `QC_SLOW=1`_ |
| TC-PG05-26 | PASS | AC5 | go test ./internal/httpapi/sse -run 'TestSSE_MaxTwoPerUser\|TestSSE_LimitSharedAcrossInstances\|TestSSE_SlotExpi ; --- PASS: TestSSE_MaxTwoPerUser (+2 dòng ok) |
| TC-PG05-27 | PASS | AC6 | số lần 'reason":"max_duration' = [1] ; ready → reconnect (ms) = 120013 (≥ 115000) (+2 dòng ok) — _sửa script QC: chạy `QC_SLOW=1`_ |
| TC-PG05-28 | PASS | AC6 (rút gọn) | reason của event: reconnect = [max_duration] ; reconnect không mang id: = [0] (+3 dòng ok) |
| TC-PG05-29 | PASS | AC6 | reason = [token_expired] ; ready → reconnect (ms), exp = 8 s = 7953 (≥ 6000) (+2 dòng ok) |
| TC-PG05-30 | PASS | AC6 / AC11 | mã HTTP = [401] ; code = [TOKEN_EXPIRED] (+2 dòng ok) |
| TC-PG05-31 | PASS | AC6 | go test ./internal/httpapi/sse -run 'TestSSE_MaxDuration\|TestSSE_TokenExpiryMidStream\|TestConfig_SSEDefaults'  ; --- PASS: TestSSE_MaxDuration (+2 dòng ok) |
| TC-PG05-32 | PASS | AC7 (nhánh lỗi) | phiên 1 nhận 5 sự kiện = [5] ; phiên 2 nhận đúng e6…e14 theo thứ tự = [6 7 8 9 10 11 12 13 14 ] (+3 dòng ok) |
| TC-PG05-33 | PASS | AC7 (biên) | số sự kiện đọc bù (phải 0) = [0] ; nhận sự kiện mới sau khi nối lại = [1] (+1 dòng ok) — _sửa script QC: như 05_ |
| TC-PG05-34 | PASS | AC7 | go test ./internal/httpapi/sse -run 'TestSSE_LastEventID_NoLossNoDup' rc = [0] ; --- PASS: TestSSE_LastEventID_NoLossNoDup |
| TC-PG05-35 | PASS | AC8 (nhánh lỗi) | số sự kiện KHÁC NHAU client nhận = [1000] ; số id duy nhất = [1000] (+3 dòng ok) — _sửa script QC: như 05_ |
| TC-PG05-36 | PASS | AC8 | go test ./internal/httpapi/sse -run 'TestSSE_ReconnectRace' rc = [0] ; --- PASS: TestSSE_ReconnectRace |
| TC-PG05-37 | PASS | AC9 (nhánh lỗi, biên) | Last-Event-ID [abc] → không 4xx/5xx ~ /^(000\|200)$/ ; Last-Event-ID [abc] → sự kiện đầu sau ready là resync = [resync] (+16 dòng ok) |
| TC-PG05-38 | PASS | AC9 (nhánh lỗi) | sự kiện đầu sau ready = [resync] ; reason = [buffer_exceeded] (+2 dòng ok) |
| TC-PG05-39 | PASS | AC9 / SRS 5.6 | XLEN ep:sse:buf (MAXLEN ~ 1000) = 1000 (≥ 1000) ; XLEN ep:sse:buf (MAXLEN ~ 1000, cho phép xấp xỉ) = 1000 (≤ 1100) (+2 dòng ok) — _sửa script QC: như 05_ |
| TC-PG05-40 | PASS | AC9 / SRS 5.6 | TTL ep:sse:conn (giây) > 0 = 150 (≥ 1) ; TTL ep:sse:conn (giây) ≤ SSE_CONN_TTL 150 = 150 (≤ 150) |
| TC-PG05-41 | **FAIL** | AC9 (biên) | **FAIL — sản phẩm (trung bình-thấp)**: `Last-Event-ID: 99999999999999-0` (mới hơn mọi id): `ready` đến nhưng sự kiện live phát sau (id nhỏ hơn) bị loại do so id → client không nhận `n=99` (tay: 0 sự kiện sau 3 s). → BUG-PG-2 |
| TC-PG05-42 | PASS | AC9 | go test ./internal/httpapi/sse -run 'TestSSE_ResyncWhenTooOld\|TestSSE_ResyncOnMalformedLastEventID' rc = [0] ; --- PASS: TestSSE_ResyncWhenTooOld (+1 dòng ok) |
| TC-PG05-43 | PASS | AC10 | số 'event: test.ping' nhận được = [5] |
| TC-PG05-44 | PASS | AC10 | X-Instance-Id của stream = [35881e44e194] (≠ []) ; số bản gateway nhận POST (round_robin của Caddy) = 2 (≥ 2) (+2 dòng ok) |
| TC-PG05-45 | PASS | AC10 | tìm được bản KHÁC bản giữ stream = [35881e44e1944ae8434464988d0d9dbf135d6b785c83c4b99a52905a8b555053] (≠ []) ; stream (ở bản A) nhận sự kiện phát từ bản B = [1] (+2 dòng ok) |
| TC-PG05-46 | PASS | AC10 | kết nối ở bản A nhận = [1] ; kết nối ở bản B nhận = [1] |
| TC-PG05-47 | PASS | AC10 | go test ./internal/httpapi/sse -run 'TestSSE_CrossInstance\|TestSSE_TwoConnectionsTwoInstances' rc = [0] ; --- PASS: TestSSE_CrossInstance (+1 dòng ok) |
| TC-PG05-48 | PASS | AC11 (phân quyền) | U1 nhận = [1] ; U2 nhận (phải 0) = [0] (+1 dòng ok) |
| TC-PG05-49 | PASS | AC11 (phân quyền) | số sự kiện nhận được = [1] ; chỉ nhận sự kiện của chính U2 (n=2) = [2 ] |
| TC-PG05-50 | PASS | AC11 (nhánh lỗi) | mã HTTP = [401] ; code = [UNAUTHENTICATED] (+3 dòng ok) |
| TC-PG05-51 | PASS | AC11 (nhánh lỗi) | mã HTTP = [401] ; code = [TOKEN_EXPIRED] |
| TC-PG05-52 | PASS | AC11 / SRS 6.8 | ?token= → mã HTTP = [401] ; ?access_token= → mã HTTP = [401] (+1 dòng ok) |
| TC-PG05-53 | PASS | AC11 / SRS 6.1 | mã HTTP = [401] ; code = [TOKEN_INVALID] |
| TC-PG05-54 | PASS | AC11 / SRS 6.3 | ADMIN phát user_id=U2 → 2xx ~ /^2[0-9][0-9]$/ ; U2 nhận sự kiện do ADMIN phát = [1] (+4 dòng ok) |
| TC-PG05-55 | PASS | AC11 | go test ./internal/httpapi/sse -run 'TestSSE_IsolationBetweenUsers\|TestSSE_UserIDQueryIgnored\|TestSSE_Unauthen ; --- PASS: TestSSE_IsolationBetweenUsers (+2 dòng ok) |
| TC-PG05-56 | PASS | AC12 | job_id là uuid ~ /^[0-9a-f-]{36}$/ ; dãy progress = [25 50 75 100 ] (+3 dòng ok) |
| TC-PG05-57 | PASS | AC12 (phân quyền) | số job.progress U2 nhận = [0] ; U2 không nhận sự kiện nào = [0] |
| TC-PG05-58 | PASS | AC12 | dòng jobs trong DB (status\|progress) = [SUCCEEDED\|100] ; GET /api/v1/jobs/{id} = [SUCCEEDED\|100] |
| TC-PG05-59 | PASS | AC12 | go test ./internal/jobs ./internal/httpapi/sse -run 'TestJobProgress_SSE\|TestJobProgress_OnlyOwner' rc = [0] ; --- PASS: TestJobProgress_SSE (+1 dòng ok) |
| TC-PG05-60 | PASS | AC13 (nhánh lỗi) | xác định được bản giữ stream = [35881e44e1944ae8434464988d0d9dbf135d6b785c83c4b99a52905a8b555053] (≠ []) ; có event: shutdown = [1] (+4 dòng ok) |
| TC-PG05-61 | PASS | AC13 | mã HTTP khi nối lại bằng token cũ (000/200 = đang stream, không 401) ~ /^(000\|200)$/ ; phiên 2 nhận đúng n=3,4,5 theo thứ tự = [3 4 5 ] (+2 dòng ok) — _sửa script QC: như 18_ |
| TC-PG05-62 | PASS | AC13 | GET /api/v1/jobs/<uuid lạ> sau khi tắt một bản = [404] |
| TC-PG05-63 | PASS | AC13 (chiều còn lại) | tìm được bản KHÔNG giữ stream = [4b77fedce711db5a7a263bcedaf022e71bdd1f317016ef0cf7d7f68fa4298d19] (≠ []) ; stream không nhận shutdown = [0] (+1 dòng ok) |
| TC-PG05-64 | PASS | AC13 | go test ./internal/httpapi/sse -run 'TestSSE_GatewayShutdownFailover' rc = [0] ; --- PASS: TestSSE_GatewayShutdownFailover |
| TC-PG05-65 | PASS | AC14 | số dòng khớp '^content-encoding\|^event: ready' (chỉ còn event: ready) = [1] |
| TC-PG05-66 | PASS | AC14 | ttfb (ms) khi xin gzip = 5 (≤ 300) ; dòng đầu đọc được dạng văn bản = [retry: 3000] (+2 dòng ok) |
| TC-PG05-67 | PASS | AC15 (nhánh lỗi) | có event: reconnect = [1] ; reason = [upstream_unavailable] (+3 dòng ok) |
| TC-PG05-68 | **FAIL** | AC15 (nhánh lỗi) | **FAIL — sản phẩm (trung bình)**: Redis tắt, mở `/api/v1/events` 3 lần liên tiếp qua Caddy: lần 1–2 `503 SERVICE_UNAVAILABLE` JSON, lần 3 là `503` rỗng (không `Content-Type`, không `X-Instance-Id`) — do Caddy tự trả "không có upstream" sau khi coi hai gateway là hỏng (`unhealthy_status 503`). Lặp lại 6 lần: lần 3 và 6 rỗng. → BUG-PG-1 |
| TC-PG05-69 | PASS | AC15 | có event: ready sau khi Redis về = [1] ; nhận được sự kiện mới = [1] |
| TC-PG05-70 | PASS | AC15 | go test ./internal/httpapi/sse -run 'TestSSE_RedisDownMidStream\|TestSSE_RedisDownOnConnect' rc = [0] ; --- PASS: TestSSE_RedisDownMidStream (+1 dòng ok) |
| TC-PG05-71 | PASS | AC2 (biên chính xác) | độ dài JSON gọn của data (biên dưới) = [65536] ; độ dài JSON gọn của data (biên trên) = [65537] (+5 dòng ok) |
| TC-PG05-72 | PASS | AC2 / SRS 6.1 | thân gửi đi (byte) > MAX_BODY_BYTES = 1100036 (≥ 1048577) ; mã HTTP = [413] (+3 dòng ok) |
| TC-PG05-73 | PASS | AC9 (biên) | [rong] dòng trạng thái 200 ~ /^HTTP/1\.1 200 / ; [rong] có event: ready = [1] (+6 dòng ok) |
| TC-PG05-74 | PASS | AC11 (phân quyền) | STUDENT phát user_id = chính mình → 2xx ~ /^2[0-9][0-9]$/ ; nhận đúng 1 sự kiện test.self = [1] (+1 dòng ok) |

## Lỗi
| Mã | Mức | Nơi | Bước tái hiện | Thấy | Mong đợi | AC / TC |
| --- | --- | --- | --- | --- | --- | --- |
| BUG-PG-1 | **trung bình** | Caddy (`Caddyfile`, `unhealthy_status 503`) | Tắt Redis; gọi `GET /api/v1/events` (hoặc `_test/error/503`) 6 lần liên tiếp qua `https://localhost` | Lần 1–2 `503` JSON `SERVICE_UNAVAILABLE`; lần 3 và 6 là `503` rỗng (không Content-Type, không `X-Instance-Id`, không `Retry-After`) vì Caddy coi cả hai gateway là hỏng | Mọi 503 của ứng dụng (trừ readyz) vẫn là JSON có `Retry-After`; Caddy không loại upstream vì 503 hợp lệ của ứng dụng | US-PG-05 AC15 · TC-PG05-68, US-PG-06 AC5 · TC-PG06-18 |
| BUG-PG-2 | trung bình-thấp | `internal/httpapi/sse/handler.go` | `Last-Event-ID: 99999999999999-0` rồi phát `test.future` | Có `ready`, không nhận sự kiện mới (id live nhỏ hơn Last-Event-ID nên bị loại) | Không `resync`, nhận sự kiện mới | US-PG-05 AC9 · TC-PG05-41 |
| BUG-PG-3 | thấp | `cmd/gateway/serve_test.go:130` (`TestServe_InvalidEnvNamesVariable/JWT_EXPIRATION`) | `go test -race ./...` nhiều lần | Đỏ chập chờn: "log lộ giá trị \"abc\"" vì `trace_id`/hostname ngẫu nhiên chứa "abc" (thấy 1/6 lần ở GATE-03) | Chỉ so giá trị trong trường `msg`, không so toàn dòng | US-PG-01 AC2 |
| BUG-PG-4 | thấp | `cmd/gateway` (chờ phụ thuộc lúc khởi động) | `REDIS_URL` cổng đóng, `STARTUP_TIMEOUT=3s` | 1 dòng warn `dependency not ready` (Redis) thay vì ≥ 2 | "Mỗi giây một dòng" (SRS 3.4) | US-PG-01 AC14 · TC-PG01-76 |
| BUG-PG-5 | thấp | `internal/store/vector_test.go:74,77` | `grep -rn float64 backend-go/internal/store` | 2 dòng `float64` (khoảng cách cosine, không phải điểm) | 0 dòng (TC) — hoặc BA chốt loại tệp `_test` | US-PG-02 AC8 · TC-PG02-48 |
| BUG-PG-6 | thấp (PM chốt) | `backend-go/api/openapi.yaml` | `grep -c _test`; tìm `bearerAuth` trong khối `jobs/{id}`, `events` | Chú thích dòng 7 có chữ `_test`; `bearerAuth` chỉ khai toàn cục (dòng 14) | 0 lần `_test`; khai tường minh theo AC7 (hoặc PM chấp nhận mặc định toàn cục) | US-PG-06 AC2, AC7 · TC-PG06-05, 30 |

Quan sát (không FAIL TC): worker xử lý việc **tuần tự** — việc 100 bước chặn việc mặc định sau nó ~17 s (TC-PG03-104 đã chỉnh thời gian chờ); `Vary` của gateway gửi thành hai dòng riêng.

## Sửa công cụ QC (theo góp ý #4–#10, PM #11) và lỗi script tìm thấy lúc chạy
Mỗi sửa ghi ở cột bằng chứng của TC tương ứng (_sửa script QC: …_). Tóm tắt: #4 literal 31 byte (`pg01.sh` 09); #5 `-w '%{http_code}'` (`pg07.sh` 15); #6 `</dev/null` (`lib.sh` `rl_reset`); #7 `--no-deps` (`pg07.sh` 54); #8 regex `9000(-[0-9]+)?->` (`pg07.sh` 56); #9 bỏ so gián tiếp + `':!legacy'` (`pg01.sh` 78/79, `pg07.sh` 58); #10 `paste -sd' ' -`, gói `httpapi` riêng, `--filter id=` (`gate-pg.sh`). Lỗi script mới (đo tay xác nhận, gateway đúng): `psql` in thêm "INSERT 0 1" (`pg02.sh` 64/65); `pg_dump` token `\restrict` ngẫu nhiên (39); `pg_get_triggerdef` đảo thứ tự sự kiện (37); `xargs` đưa biến môi trường sau `serve` (`pg04.sh` 41); `wait` trần chặn `sse_burst` (`sse-reconnect.sh`; 05/33/35/39); curl 8.7 in `200` khi hết `-m` giữa thân SSE (18/20/21/61); `Connection` bị Caddy bỏ (05-02); slog in level `ERROR` hoa (03-07); `Vary` hai dòng (03-76); mã dồn dòng (03-74); `wait` pid của subshell, `sse_ns` thiếu bỏ "data: ", log `compose run` có dòng không phải JSON (gate); `tab.run` truyền args sai (`sse-browser-cut.mjs`, `idem-two-tabs.mjs`). **Môi trường:** `~/.testcontainers.properties` (README), `brew install sqlc` (1.31.1 = CI); `run-all.sh` chết sau story 01 khi chạy nền — chạy từng script ở foreground.
