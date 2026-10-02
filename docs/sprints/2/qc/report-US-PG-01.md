# QC report — US-PG-01 (gateway nền) · sprint 2 (PG Nền Go) · Kết luận: **FAIL**

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `bash docs/sprints/2/qc/scripts/pg01.sh` (lượt đầu → sau triage + sửa script: 82/85 → sau sửa script 84/85). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 84/85 TC PASS, **1 FAIL** — lỗi sản phẩm / TC: BUG-PG-4.

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG01-01 | PASS | AC1 | rc khi thiếu DATABASE_URL = [1] ; ms khi thiếu DATABASE_URL = 33 (≤ 1000) (+32 dòng ok) |
| TC-PG01-02 | PASS | AC1 | số lần cổng 127.0.0.1:18080 trả lời = [0] ; rc = [1] |
| TC-PG01-03 | PASS | AC1 | rc = [1] ; ms = 19 (≤ 1000) (+3 dòng ok) |
| TC-PG01-04 | PASS | AC1 | rc = [1] ; ms = 549 (≤ 1000) (+1 dòng ok) |
| TC-PG01-05 | PASS | AC1 | rc = [1] ; ms = 17 (≤ 1000) (+1 dòng ok) |
| TC-PG01-06 | PASS | AC1 | rc = [1] ; ms = 17 (≤ 1000) (+2 dòng ok) |
| TC-PG01-07 | PASS | AC1 / SRS 8.1 | rc (JWT_SECRET_KEY ba dấu cách) = [1] ; missing nêu JWT_SECRET_KEY = [true] (+2 dòng ok) |
| TC-PG01-08 | PASS | AC1 | go test ./internal/platform/config -run 'TestLoad_MissingEnv\|TestLoad_Worker' rc = [0] ; --- PASS: TestLoad_MissingEnv (+1 dòng ok) |
| TC-PG01-09 | PASS | AC2 | rc khi JWT_SECRET_KEY sai = [1] ; ms khi JWT_SECRET_KEY sai = 18 (≤ 1000) (+2 dòng ok) — _sửa script QC: #4/#11: literal 31 byte thật_ |
| TC-PG01-10 | PASS | AC2 | rc khi DB_MAX_CONNS sai = [1] ; ms khi DB_MAX_CONNS sai = 19 (≤ 1000) (+1 dòng ok) |
| TC-PG01-11 | PASS | AC2 | rc khi JWT_EXPIRATION sai = [1] ; ms khi JWT_EXPIRATION sai = 18 (≤ 1000) (+2 dòng ok) |
| TC-PG01-12 | PASS | AC2 / SRS 6.7, 8.1 | rc khi CORS_ORIGINS sai = [1] ; ms khi CORS_ORIGINS sai = 17 (≤ 1000) (+1 dòng ok) |
| TC-PG01-13 | PASS | AC2 | rc khi APP_ENV sai = [1] ; ms khi APP_ENV sai = 19 (≤ 1000) (+2 dòng ok) |
| TC-PG01-14 | PASS | AC2 (biên) | độ dài khoá hợp lệ = [32] ; độ dài khoá lỗi = [31] (+5 dòng ok) |
| TC-PG01-15 | PASS | AC2 (biên) | DB_MAX_CONNS=1: số dòng error nêu biến = [0] ; DB_MAX_CONNS=1: đi tới bước chờ phụ thuộc (ms) = 3036 (≥ 1500) (+5 dòng ok) |
| TC-PG01-16 | PASS | AC2 (biên) | APP_ENV=dev: số dòng error nêu APP_ENV = [0] ; APP_ENV=dev: đi tới bước chờ phụ thuộc (ms) = 3041 (≥ 1500) (+4 dòng ok) |
| TC-PG01-17 | PASS | AC2 | go test ./internal/platform/config -run 'TestLoad_Invalid' rc = [0] ; --- PASS: TestLoad_Invalid |
| TC-PG01-18 | PASS | AC3 | RestartCount của 363a6b896e45 (tiền điều kiện: không restart) = [0] ; số dòng 'config loaded' của 363a6b896e45 = [1] (+4 dòng ok) |
| TC-PG01-19 | PASS | AC3 | số dòng 'config loaded' = [1] ; db_max_conns (kiểu JSON) = [number] (+10 dòng ok) |
| TC-PG01-20 | PASS | AC3 | số dòng log chứa bí mật = [0] |
| TC-PG01-21 | PASS | AC3 | go test ./internal/platform/config -run 'TestLoad_Defaults\|TestConfig_NoSecretInLog' rc = [0] ; --- PASS: TestLoad_Defaults (+1 dòng ok) |
| TC-PG01-22 | PASS | AC4 | mọi dòng log đạt (JSON + 6 trường + trace_id 32 hex ≠ 0) = [true] ; số dòng log đã kiểm = 91 (≥ 10) |
| TC-PG01-23 | PASS | AC4 | service trong log gateway = [gateway] ; service trong log worker = [worker] (+1 dòng ok) |
| TC-PG01-24 | PASS | AC4 | mọi dòng log quanh lúc dừng đạt = [true] ; số dòng log quanh lúc dừng = 1 (≥ 1) |
| TC-PG01-25 | PASS | AC4 | số dòng log chứa email mồi = [0] ; số dòng log chứa chữ ký token = [0] |
| TC-PG01-26 | PASS | AC4 | go test ./internal/platform/log ./internal/platform/otel -run 'TestHandler_AlwaysTraceID\|TestStartupHasTrace'  ; --- PASS: TestHandler_AlwaysTraceID (+1 dòng ok) |
| TC-PG01-27 | PASS | AC5 | trace_id trong thân lỗi = [4bf92f3577b34da6a3ce929d0e0e4736] ; mã = [404] (+2 dòng ok) |
| TC-PG01-28 | PASS | AC5 | số dòng log mang trace_id của request = 3 (≥ 1) |
| TC-PG01-29 | PASS | AC5 / SRS 6.5 | trace_id tự sinh 32 hex ~ /^[0-9a-f]{32}$/ ; trace_id khác toàn 0 = [31191a921cacb46f0ffdf49ef3e2cb00] (≠ [00000000000000000000000000000000]) (+2 dòng ok) |
| TC-PG01-30 | PASS | AC5 | go test ./internal/httpapi -run 'TestTraceID_InErrorBodyAndLogs' rc = [0] ; --- PASS: TestTraceID_InErrorBodyAndLogs |
| TC-PG01-31 | PASS | AC6 | go test ./internal/platform/redis -run 'TestKeys\|TestTTLConstants' rc = [0] ; --- PASS: TestKeys (+1 dòng ok) |
| TC-PG01-32 | PASS | AC6 | có khoá ep:idem:* = 316 (≥ 1) ; có khoá ep:rl:* = 2 (≥ 1) (+3 dòng ok) |
| TC-PG01-33 | PASS | AC6 / SRS 5.6 | TTL ep:idem ≤ 86400 = 85562 (≤ 86400) ; TTL ep:rl ≤ 120 = 119 (≤ 120) (+4 dòng ok) — _sửa script QC: đo TTL lớn nhất (khoá mới); khoá cũ đã già_ |
| TC-PG01-34 | PASS | AC6 | số khoá Redis ngoài quy ước = [0] |
| TC-PG01-35 | PASS | AC7 (2) | mã của request chậm = [200] |
| TC-PG01-36 | PASS | AC7 (5) | mã thoát của mọi bản gateway = [0] ; thời gian tắt (ms) = 3198 (≤ 25000) |
| TC-PG01-37 | PASS | AC7 (1) | mã readyz khi đang tắt = [503] ; code = [NOT_READY] (+1 dòng ok) |
| TC-PG01-38 | PASS | AC7 (3) | số dòng 'event: shutdown' trong stream = [1] ; thời gian đóng kết nối sau SIGTERM (ms; AC 1000 + 200 sai số đo) = 28 (≤ 1200) |
| TC-PG01-39 | PASS | AC7 (4) | kết nối mới KHÔNG được 2xx !~ /^2/ ; mã của kết nối mới (503 / từ chối) ~ /^(000\|502\|503)$/ |
| TC-PG01-40 | PASS | AC7 (nhánh lỗi) | mã thoát khi cắt request quá hạn = [1] ; thời gian tắt (ms) ≈ SHUTDOWN_TIMEOUT 2 s = 2220 (≤ 8000) (+4 dòng ok) |
| TC-PG01-41 | PASS | AC7 | go test ./cmd/gateway ./internal/httpapi -run 'TestGracefulShutdown\|TestShutdown_ForcedAfterTimeout' rc = [0] ; --- PASS: TestGracefulShutdown (+1 dòng ok) — _sửa script QC: `gt` bỏ cờ "no tests to run" (#10(2))_ |
| TC-PG01-42 | PASS | AC8 | mã = [413] ; code = [PAYLOAD_TOO_LARGE] (+1 dòng ok) |
| TC-PG01-43 | PASS | AC8 (biên dưới) | kích thước thân = [1048576] ; mã KHÔNG phải 413 (thân hợp lệ JSON nên có thể 2xx hoặc 422) !~ /^413$/ (+1 dòng ok) |
| TC-PG01-44 | PASS | AC8 (biên trên) | kích thước thân = [1048577] ; mã = [413] (+1 dòng ok) |
| TC-PG01-45 | PASS | AC8 | không nhận được trả lời 2xx !~ /^HTTP/1\.1 2/ ; thời gian tới lúc đóng (ms) = 5103 (≥ 3000) (+1 dòng ok) |
| TC-PG01-46 | PASS | AC8 | dòng trạng thái ~ /^HTTP/1\.1 431/ |
| TC-PG01-47 | PASS | AC8 | không nhận được trả lời 2xx !~ /^HTTP/1\.1 2/ ; thời gian tới lúc đóng (ms) = 15082 (≥ 12000) (+1 dòng ok) |
| TC-PG01-48 | PASS | AC8 / SRS 6.1 | số trường hợp lệ = tổng số trường = [4] ; trace_id 32 hex ~ /^[0-9a-f]{32}$/ (+2 dòng ok) |
| TC-PG01-49 | PASS | AC8 | go test ./internal/httpapi -run 'TestServerLimits\|TestServerLimits_SlowHeader\|TestServerLimits_HugeHeader' rc  ; --- PASS: TestServerLimits (+2 dòng ok) |
| TC-PG01-50 | PASS | AC9 | mã = [504] ; code = [DEADLINE_EXCEEDED] (+1 dòng ok) |
| TC-PG01-51 | PASS | AC9 | số truy vấn pg_sleep còn chạy = [0] |
| TC-PG01-52 | PASS | AC9 | mã = [504] ; code = [DEADLINE_EXCEEDED] (+1 dòng ok) |
| TC-PG01-53 | PASS | AC9 | số truy vấn pg_sleep trước khi gọi = [0] ; số truy vấn pg_sleep 1,5 s sau khi client ngắt = [0] |
| TC-PG01-54 | PASS | AC9 | go test ./internal/httpapi -run 'TestDeadline_DB\|TestDeadline_Redis\|TestDeadline_ClientCancel' rc = [0] ; --- PASS: TestDeadline_DB (+2 dòng ok) |
| TC-PG01-55 | PASS | AC10 | go test ./internal/platform/db -run 'TestPool_MaxConns\|TestPool_AcquireHonorsDeadline' rc = [0] ; --- PASS: TestPool_MaxConns (+1 dòng ok) |
| TC-PG01-56 | PASS | AC10 | số bản gateway lúc đo = [1] ; số client PgBouncer của gateway lớn nhất (≤ DB_MAX_CONNS × số bản) = 3 (≤ 3) (+1 dòng ok) |
| TC-PG01-57 | PASS | AC10 | số request trả 504 = [5] ; tổng thời gian 5 request song song (ms; không treo) = 2043 (≤ 6000) |
| TC-PG01-58 | PASS | AC11 | số dòng 'slow query' của request = [1] ; mức log = [warn] (+4 dòng ok) |
| TC-PG01-59 | PASS | AC11 | số dòng 'slow query' của truy vấn 50 ms = [0] |
| TC-PG01-60 | PASS | AC11 | có dòng slow query để kiểm ~ /slow query/ ; số lần xuất hiện giá trị tham số 0.37 = [0] |
| TC-PG01-61 | PASS | AC11 | go test ./internal/platform/db -run 'TestSlowQueryLog\|TestSlowQueryLog_NoArgs' rc = [0] ; --- PASS: TestSlowQueryLog (+1 dòng ok) |
| TC-PG01-62 | PASS | AC12 | dòng trạng thái ~ /^HTTP/1\.1 200/ |
| TC-PG01-63 | PASS | AC12 | healthcheck gọi -healthcheck ~ /healthcheck/ ; Health của worker = [healthy] (+1 dòng ok) |
| TC-PG01-64 | PASS | AC12 | số dòng outbox đã được worker nhặt trong ≤ 5 s = 1 (≥ 1) |
| TC-PG01-65 | PASS | AC12 | mã thoát của worker = [0] ; thời gian tắt worker (ms) = 299 (≤ 10000) |
| TC-PG01-66 | PASS | AC12 | go test ./cmd/worker -run 'TestWorker_Health\|TestWorker_Shutdown' rc = [0] ; --- PASS: TestWorker_Health (+1 dòng ok) |
| TC-PG01-67 | PASS | AC13 / SRS 9.4 | mục tiêu Makefile thiếu = [] |
| TC-PG01-68 | PASS | AC13 / SRS 9.4 | số lần RYUK_DISABLED=true = [1] ; có cờ -race = 1 (≥ 1) (+1 dòng ok) |
| TC-PG01-69 | PASS | AC13 | có DOCKER_HOST = 1 (≥ 1) ; trỏ tới socket colima = 1 (≥ 1) |
| TC-PG01-70 | PASS | AC14 | mã readyz = [503] ; code = [NOT_READY] (+3 dòng ok) |
| TC-PG01-71 | PASS | AC14 | mã readyz sau khi Redis trở lại = [200] ; số giây tới khi readyz 200 = 0 (≤ 10) |
| TC-PG01-72 | PASS | AC14 | RestartCount\|StartedAt\|Pid\|Running trước = sau = [0\|2026-10-02T21:33:45.60146561Z\|1888345\|true] ; tiến trình vẫn chạy ~ /true$/ |
| TC-PG01-73 | PASS | AC14 | mã readyz = [503] ; code = [NOT_READY] (+3 dòng ok) |
| TC-PG01-74 | PASS | AC14 | mã readyz = [503] ; details.db = [down] (+2 dòng ok) |
| TC-PG01-75 | PASS | AC14 / SRS 3.4 | rc = [1] ; ms ≈ STARTUP_TIMEOUT 3 s = 3051 (≥ 2500) (+6 dòng ok) |
| TC-PG01-76 | **FAIL** | AC14 / SRS 3.4 | **FAIL — sản phẩm (thấp)**: STARTUP_TIMEOUT=3s, Redis đóng: chỉ 1 dòng warn `dependency not ready` (t=+1,0 s) rồi error (t=+3,0 s); TC/SRS 3.4 đòi ≥ 2 warn "mỗi giây". Log cho thấy go-redis thử dial 5 lần (~1 s/lần) kéo vòng chờ giãn ra. → BUG-PG-4 |
| TC-PG01-77 | PASS | AC14 | go test ./internal/httpapi -run 'TestReadyz_DependencyDown\|TestStartup_WaitsForDeps' rc = [0] ; --- PASS: TestReadyz_DependencyDown (+1 dòng ok) |
| TC-PG01-78 | PASS | AC15 | số dòng khớp bí mật trong repo = [0] — _sửa script QC: #9(2)/#11 `:!legacy`_ |
| TC-PG01-79 | PASS | AC15 | .env.example tồn tại ; có khai báo DATABASE_URL = 1 (≥ 1) (+7 dòng ok) — _sửa script QC: #9(1)/#11 đo `-dev` thay so với .env.local_ |
| TC-PG01-80 | PASS | AC15 | số dòng log chứa giá trị JWT_SECRET_KEY = [0] ; số dòng log chứa giá trị BLOB_SECRET_KEY = [0] (+2 dòng ok) |
| TC-PG01-81 | PASS | AC1 | worker: số dòng log có .missing (JWT/BLOB_* không bắt buộc) = [0] ; worker: đã qua kiểm cấu hình, chờ phụ thuộc tới STARTUP_TIMEOUT=2s (ms) = 2070 (≥ 1500) (+5 dòng ok) |
| TC-PG01-82 | PASS | AC2 | rc khi BCRYPT_COST sai = [1] ; ms khi BCRYPT_COST sai = 17 (≤ 1000) (+7 dòng ok) |
| TC-PG01-83 | PASS | AC2 (biên) | BCRYPT_COST=4: số dòng error nêu biến = [0] ; BCRYPT_COST=4: đi tới bước chờ phụ thuộc (ms) = 3040 (≥ 1500) (+2 dòng ok) |
| TC-PG01-84 | PASS | AC8 | kích thước thân = [1048577] ; mã = [413] (+1 dòng ok) |
| TC-PG01-85 | PASS | AC13 / SRS 9.4 | số lần gọi golangci-lint = [2] ; số lần golangci-lint chạy với tag testroutes = [1] |

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
