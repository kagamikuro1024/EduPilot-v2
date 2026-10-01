# FEAT-pg-foundation Nền Go: gateway không trạng thái cho mọi phase sau
Nguồn: PRD §5 (Triển khai), FLOWS — (hạ tầng, chưa thuộc luồng nào; mọi luồng F1–F18 về sau đứng trên nền này), phase PG lát L1–L7 (`docs/phases/PG.md`); quyết định D22, D45, D46, D47, D48, D52; `ARCHITECTURE.md` §2, §3, §4, §5, §8; `SYSTEM_DESIGN.md` §2, §3.2, §3.3, §3.4, §5; luật 10–15 `AGENTS.md`. Chi tiết kỹ thuật: `SRS.md`; câu hỏi mở: `QUESTIONS.md`.

Phiên bản 1.1 · 2026-10-02 · Trạng thái: APPROVED (PM 2026-10-01; Q4 theo góp ý #1). **Phạm vi:** 7 story US-PG-01…07, không có endpoint nghiệp vụ. Endpoint được phép: `/healthz`, `/api/v1/healthz`, `/api/v1/readyz`, `/api/v1/jobs/{id}`, `/api/v1/events` (SSE) và các endpoint **chỉ có trong binary / image dựng bằng build tag `testroutes`** (target Dockerfile `gateway-test`, `worker-test`; compose override `docker-compose.test.yml`) dưới `/api/v1/_test/…` để kiểm helper (`SRS.md` mục 6.3). Binary và image mặc định **không chứa** các route này.

Lịch sử phiên bản: v1 (2026-10-02, BA viết). **v1.1 (2026-10-02)** — góp ý #1 `docs/sprints/2/proposals.md` (PM chốt, `ACCEPTED` 01/10; trích: "Khoá bằng build tag `testroutes`: file đăng ký route thử có `//go:build testroutes`; Dockerfile có target riêng `gateway-test` (`go build -tags testroutes`); compose dùng target đó qua override `docker-compose.test.yml` cho QC. Binary/image mặc định không chứa route thử. AC18 của US-PG-03 đổi thành: image mặc định → mọi `/api/v1/_test/*` 404; `go tool nm` của binary mặc định không có symbol của gói route thử"): route thử khoá bằng **build tag `testroutes`** thay cho `APP_ENV=test`. Đổi: header, "Quy ước kiểm chung" (`CT`, `tmode`, `dmode`, `gotest`), US-PG-01 AC9 (lệnh kiểm), US-PG-03 phần Truy vết và AC18 (nội dung và lệnh kiểm), US-PG-06 AC3–AC4, US-PG-07 AC12, AC14, AC19, và **thêm US-PG-07 AC20** (target `gateway-test`, `worker-test`, compose override). Tổng AC: 104 → 105. Hệ quả BA bổ sung để góp ý #1 chạy được (PM xem lại nếu không đồng ý): thêm target `worker-test` vì job `test.progress` / `test.fail` do worker xử lý; `gateway-test` từ chối khởi động khi `APP_ENV=production`.

## Quy ước kiểm chung

```bash
# Chạy ở gốc repo, sau `pnpm dev` (hoặc `$C up -d --wait`). Docker chạy qua colima.
C="docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot"
CT="$C -f docker-compose.test.yml"       # + override dựng gateway/worker bằng build tag testroutes (chỉ cho QC / test)
GW=https://localhost                      # qua Caddy; luôn thêm -k vì TLS nội bộ của Caddy
PSQL="$C exec -T postgres psql -U edupilot -d edupilot -v ON_ERROR_STOP=1 -At"
RDS="$C exec -T redis redis-cli"
SECRET=$(grep '^JWT_SECRET_KEY=' .env.local | cut -d= -f2-)
# JWT dev: tok <ROLE> [sub-uuid] [các cờ của lệnh token, ví dụ --ttl -1m]
tok() { (cd backend-go && JWT_SECRET_KEY="$SECRET" go run ./cmd/gateway token --role "$1" ${2:+--sub "$2"} "${@:3}"); }
# bật các endpoint kiểm tra: dựng gateway + worker bằng build tag `testroutes` (image edupilot-gateway-test / edupilot-worker-test, SRS 8.4); hai bản gateway
tmode() { $CT up -d --build --force-recreate --scale gateway=2 --wait gateway worker; }
# về chế độ thường: image mặc định, không có route thử
dmode() { $C up -d --build --force-recreate --scale gateway=2 --wait gateway worker; }
# test Go (container thật: Postgres 18 + pgvector, Redis 8, MinIO, PgBouncer qua testcontainers)
gotest() { (cd backend-go && go test -race -count=1 -tags testroutes "$@"); }
envv() { grep "^$1=" .env.local | cut -d= -f2-; }            # đọc một biến từ .env.local
PGB="postgres://$(envv PGBOUNCER_STATS_USER):$(envv PGBOUNCER_STATS_PASSWORD)@pgbouncer:6432/pgbouncer"   # quản trị PgBouncer, chạy từ container postgres
U1=00000000-0000-7000-8000-000000000001; U2=00000000-0000-7000-8000-000000000002
H="Authorization: Bearer $(tok STUDENT $U1)"                 # token STUDENT mặc định
```

Test Go cần Docker; không có Docker thì **fail** (không skip), trừ khi `-short`. `make -C backend-go test` tự đặt `TESTCONTAINERS_RYUK_DISABLED=true` và `DOCKER_HOST` của colima nếu chưa có (`SRS.md` mục 9) và chạy `go test -race -count=1 -tags testroutes ./...` (`gotest` bên dưới cũng vậy, vì test route thử, test SSE chéo bản, test auth… cần route thử). Bản **không tag** được kiểm riêng bằng US-PG-03 AC18 và `go build ./...`. **Lưu ý:** `tmode` làm gateway-test tự tạo bảng thử `_test_items`; các lệnh đếm bảng / dump schema (US-PG-02 AC1, AC4, AC6) phải chạy trên DB **chưa qua `tmode`** (volume sạch hoặc chỉ `dmode`).

Các ID cố định để lệnh kiểm chạy được: người dùng mẫu `U1 = 00000000-0000-7000-8000-000000000001`, `U2 = 00000000-0000-7000-8000-000000000002`; mật khẩu thử `Correct-Horse-9`; khoá idempotency mẫu `idem-0001-aaaa`.

---

## US-PG-01: Lập trình viên (dev các phase sau) muốn có khung dịch vụ Go với cấu hình chặt, log có `trace_id`, giới hạn ở mọi tầng và tắt máy êm để mọi module sau cắm vào mà không phải nghĩ lại chuyện vận hành
Ưu tiên: Must · Ước lượng: M · Sprint: 2

Truy vết: PG L1 (8 checkbox).

### Tiêu chí nghiệm thu
- AC1 (nhánh lỗi — thiếu env). Given gateway thiếu một biến bắt buộc trong `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET_KEY`, `BLOB_ENDPOINT`, `BLOB_BUCKET`, `BLOB_ACCESS_KEY`, `BLOB_SECRET_KEY` (từng biến một) When chạy `gateway serve` Then tiến trình thoát **mã 1 trong ≤ 1 s**, trước khi mở cổng; stdout có đúng một dòng JSON mức `error` có mảng `missing` chứa **tên** biến thiếu (không in giá trị); cổng 8080 không mở. Thiếu nhiều biến → `missing` liệt kê đủ cùng lúc. Worker thiếu `DATABASE_URL` hoặc `REDIS_URL` cũng thế.
  Kiểm:
  ```bash
  cd backend-go && go build -o /tmp/gw ./cmd/gateway && go build -o /tmp/wk ./cmd/worker
  full() { printf '%s\n' DATABASE_URL=postgres://u:p@127.0.0.1:1/db REDIS_URL=redis://127.0.0.1:1/0 JWT_SECRET_KEY=0123456789abcdef0123456789abcdef BLOB_ENDPOINT=127.0.0.1:1 BLOB_BUCKET=b BLOB_ACCESS_KEY=ak BLOB_SECRET_KEY=sk; }
  for v in DATABASE_URL REDIS_URL JWT_SECRET_KEY BLOB_ENDPOINT BLOB_BUCKET BLOB_ACCESS_KEY BLOB_SECRET_KEY; do
    s=$(date +%s%N); full | grep -v "^$v=" | xargs env -i PATH="$PATH" /tmp/gw serve >/tmp/o 2>&1; rc=$?
    echo "$v rc=$rc ms=$(( ($(date +%s%N)-s)/1000000 )) named=$(jq -r 'select(.missing).missing|index("'$v'")!=null' /tmp/o | head -1)"; done
  # mỗi dòng: rc=1, ms ≤ 1000, named=true
  env -i PATH="$PATH" /tmp/gw serve 2>&1 | jq -c '.missing'   # đủ 7 tên
  env -i PATH="$PATH" REDIS_URL=redis://x /tmp/wk 2>&1 | jq -c '.missing'   # ["DATABASE_URL"]
  ```
  Test: `gotest ./internal/platform/config -run 'TestLoad_MissingEnv|TestLoad_Worker'`.
- AC2 (nhánh lỗi — giá trị sai). Given `JWT_SECRET_KEY` dài 31 byte, hoặc `DB_MAX_CONNS=0`, hoặc `JWT_EXPIRATION=abc`, hoặc `APP_CORS_ALLOWED_ORIGINS=*`, hoặc `APP_ENV=staging` When chạy `gateway serve` Then thoát mã 1 ngay, dòng log nêu **tên biến và lý do** (ví dụ "JWT_SECRET_KEY: cần ≥ 32 byte") và **không in giá trị** của nó.
  Kiểm: `full | sed 's/^JWT_SECRET_KEY=.*/JWT_SECRET_KEY=short-secret-31-bytes-xxxxxxxxxx/' | xargs env -i PATH="$PATH" /tmp/gw serve 2>&1 | tee /tmp/o | jq -r .msg; grep -c 'short-secret' /tmp/o` → có chữ `JWT_SECRET_KEY`; `0`. Lặp cho 4 biến còn lại. Test: `gotest ./internal/platform/config -run TestLoad_Invalid`.
- AC3. Given đủ 7 biến bắt buộc, các biến còn lại bỏ trống When khởi động Then chạy với mặc định dev ở `SRS.md` mục 8 (ví dụ `DB_MAX_CONNS=10`, `REQUEST_TIMEOUT=30s`, `SSE_HEARTBEAT=25s`), ghi **một** dòng `config loaded` liệt kê giá trị hiệu lực của biến không bí mật và `"secrets":"[redacted]"`; không dòng log nào chứa `JWT_SECRET_KEY`, `BLOB_SECRET_KEY`, mật khẩu trong URL.
  Kiểm: `$C logs gateway | jq -c 'select(.msg=="config loaded")'` → đúng 1 dòng; `$C logs gateway worker | grep -cE "$SECRET|edupilot-dev-secret|edupilot-dev@"` → `0`. Test: `gotest ./internal/platform/config -run 'TestLoad_Defaults|TestConfig_NoSecretInLog'`.
- AC4. Given gateway và worker chạy When đọc log Then **mọi** dòng là JSON hợp lệ, có `time`, `level`, `msg`, `service` (`gateway`/`worker`), `instance`, và `trace_id` 32 hex khác toàn số 0 — kể cả dòng khởi động, dừng, vòng lặp nền của worker. Không dòng nào chứa tên, MSSV, email, token hay thân request.
  Kiểm: `$C logs --no-log-prefix gateway worker | jq -s 'all(.[]; (.trace_id|test("^[0-9a-f]{32}$")) and (.trace_id!="00000000000000000000000000000000") and .service and .instance)'` → `true`. Test: `gotest ./internal/platform/log ./internal/platform/otel -run 'TestHandler_AlwaysTraceID|TestStartupHasTrace'`.
- AC5. Given request gửi header `traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01` When gateway trả **bất kỳ** lỗi (ví dụ 404) Then thân lỗi có `trace_id` = `4bf92f3577b34da6a3ce929d0e0e4736`, header `X-Request-Id` có mặt, và mọi dòng log của request đó mang cùng `trace_id`. Không có `traceparent` → gateway tự sinh trace mới.
  Kiểm: `curl -sk -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' -D- $GW/api/v1/khong-co | grep -iE 'x-request-id|"trace_id"'`; `$C logs --no-log-prefix gateway | jq -c 'select(.trace_id=="4bf92f3577b34da6a3ce929d0e0e4736")' | wc -l` ≥ 1. Test: `gotest ./internal/httpapi -run TestTraceID_InErrorBodyAndLogs`.
- AC6. Given `redis` client dùng chung When đọc `internal/platform/redis` Then mọi khoá do code tạo bắt đầu bằng `ep:` qua một hàm dựng khoá duy nhất, mọi khoá ngoài Stream đều có TTL; hằng TTL mặc định đúng `SRS.md` mục 5.6 (idempotency 24 h, khoá khoá-chạy 30 s, rate limit 120 s, kết nối SSE 150 s, bộ đệm SSE 1 h, cache 5 phút).
  Kiểm: `gotest ./internal/platform/redis -run 'TestKeys|TestTTLConstants'`; sau khi chạy xong cả bộ test của story 03 và 05 trên Redis dev: `$RDS --scan --pattern 'ep:*' | while read k; do echo "$($RDS ttl "$k") $k"; done | grep -c '^-1 '` → `0`; `$RDS --scan | grep -vcE '^(ep:|outbox\.dispatch|jobs\.)'` → `0`.
- AC7 (nhánh lỗi — tắt máy êm). Given gateway đang có một request chậm 3 s (`GET /api/v1/_test/slow?ms=3000`) và một stream SSE mở When gửi SIGTERM Then (1) `/api/v1/readyz` trả 503 `NOT_READY` ngay; (2) request đang chạy **hoàn tất 200**, không bị cắt; (3) client SSE nhận `event: shutdown` rồi kết nối đóng trong ≤ 1 s; (4) kết nối mới sau SIGTERM bị từ chối hoặc nhận 503; (5) tiến trình thoát **mã 0** trước `SHUTDOWN_TIMEOUT` (25 s). Request chạy quá hạn → bị cắt, log mức `warn` ghi số request bị cắt, thoát mã 1.
  Kiểm: `tmode; (curl -sk -o /dev/null -w 'slow=%{http_code}\n' "$GW/api/v1/_test/slow?ms=3000" &) ; sleep 0.5; $C stop -t 30 gateway; $C ps -a --format '{{.Service}} {{.ExitCode}}' | grep ^gateway` → `slow=200`, mọi `gateway` exit 0. Test: `gotest ./cmd/gateway ./internal/httpapi -run 'TestGracefulShutdown|TestShutdown_ForcedAfterTimeout'`.
- AC8. Given giới hạn tầng HTTP When thử vượt Then: thân POST lớn hơn `MAX_BODY_BYTES` (1 MiB) → **413** `PAYLOAD_TOO_LARGE`; gửi header chậm hơn 5 s → server đóng kết nối (`ReadHeaderTimeout`); header lớn hơn 64 KiB → 431; request đọc thân quá `ReadTimeout` (15 s) → đóng; mọi trả lời lỗi theo định dạng thống nhất.
  Kiểm: `head -c 2000000 /dev/zero | tr '\0' a | curl -sk -X POST -H 'Content-Type: application/json' -H "$H" --data-binary @- -w '\n%{http_code}\n' $GW/api/v1/_test/items | tail -2` → `PAYLOAD_TOO_LARGE`, `413`; header chậm và header quá lớn kiểm bằng test (cổng 8080 không công bố ra máy chủ). Test: `gotest ./internal/httpapi -run 'TestServerLimits|TestServerLimits_SlowHeader|TestServerLimits_HugeHeader'`.
- AC9 (nhánh lỗi — deadline xuống tận DB/Redis). Given `REQUEST_TIMEOUT=1s` When gọi `GET /api/v1/_test/db-sleep?seconds=5` Then trả **504** `DEADLINE_EXCEEDED` trong ≤ 1,5 s và truy vấn `pg_sleep` **bị huỷ ở Postgres** (không còn dòng nào trong `pg_stat_activity` sau 1 s); `GET /api/v1/_test/redis-block?seconds=5` (BLPOP) cũng 504 và không còn client blocked; client ngắt giữa chừng cũng huỷ truy vấn.
  Kiểm: `REQUEST_TIMEOUT=1s $CT up -d --build --force-recreate --no-deps gateway; time curl -sk -o /dev/null -w '%{http_code}\n' "$GW/api/v1/_test/db-sleep?seconds=5"` → `504`, `real` < 1,6 s; `sleep 1; $PSQL -c "select count(*) from pg_stat_activity where query ilike '%pg_sleep%' and state='active' and pid<>pg_backend_pid()"` → `0`. Test: `gotest ./internal/httpapi -run 'TestDeadline_DB|TestDeadline_Redis|TestDeadline_ClientCancel'`.
- AC10. Given `DB_MAX_CONNS=3` When 20 request đồng thời mỗi cái giữ một truy vấn 1 s Then số kết nối Postgres do pool này mở (`application_name='edupilot-gateway'` trên kết nối **trực tiếp**, không qua PgBouncer) không bao giờ vượt 3; request chờ lấy kết nối vẫn tuân thủ deadline (hết deadline → 504, không treo).
  Test: `gotest ./internal/platform/db -run 'TestPool_MaxConns|TestPool_AcquireHonorsDeadline'`.
- AC11. Given một truy vấn chạy ≥ 200 ms When hoàn tất Then log đúng **một** dòng `msg:"slow query"` mức `warn` có `duration_ms ≥ 200`, `trace_id` của request gọi, tên truy vấn (sqlc `-- name:`) và **không** chứa giá trị tham số; truy vấn 50 ms không sinh dòng nào.
  Kiểm: `$C logs --no-log-prefix gateway | jq -c 'select(.msg=="slow query")'` sau khi gọi `GET /api/v1/_test/db-sleep?seconds=0.25`. Test: `gotest ./internal/platform/db -run 'TestSlowQueryLog|TestSlowQueryLog_NoArgs'`.
- AC12. Given `cmd/worker` When chạy Then có vòng lặp consumer (chưa có handler nghiệp vụ), `/healthz` nội bộ ở `WORKER_HEALTH_ADDR` (`:8081`) trả 200 khi vòng lặp còn nhịp và DB + Redis ping được, `worker -healthcheck` thoát 0; SIGTERM → thoát mã 0 trong ≤ 10 s.
  Kiểm: `$C ps --format '{{.Service}} {{.Health}}' | grep ^worker` → `healthy`; `$C stop -t 15 worker; $C ps -a --format '{{.Service}} {{.ExitCode}}' | grep ^worker` → `0`. Test: `gotest ./cmd/worker -run 'TestWorker_Health|TestWorker_Shutdown'`.
- AC13. Given Makefile ở `backend-go/` When đọc Then có mục tiêu `run`, `test`, `lint`, `sqlc`, `migrate`; `make test` = `go test -race ./...` với `TESTCONTAINERS_RYUK_DISABLED=true` và tự lấy `DOCKER_HOST` của colima khi chưa đặt.
  Kiểm: `for t in run test lint sqlc migrate; do make -C backend-go -n $t >/dev/null 2>&1 || echo "THIẾU $t"; done` → không in gì; `make -C backend-go -n test | grep -c 'RYUK_DISABLED=true'` → `1`.
- AC14 (nhánh lỗi — phụ thuộc chết lúc chạy). Given gateway đang chạy When tắt Redis (`$C stop redis`) Then `/api/v1/healthz` vẫn 200, `/api/v1/readyz` trả **503** `NOT_READY` với `details.redis:"down"` trong ≤ 10 s, tiến trình **không thoát**; bật Redis lại → `readyz` 200 trong ≤ 10 s không cần khởi động lại. Tương tự khi tắt Postgres (`details.db:"down"`). Khi khởi động mà DB/Redis chưa lên: chờ tối đa `STARTUP_TIMEOUT` (30 s), mỗi giây log `warn` tên phụ thuộc; quá hạn thoát mã 1 nêu tên phụ thuộc.
  Kiểm: `$C stop redis; sleep 10; curl -sk -w '\n%{http_code}\n' $GW/api/v1/readyz; curl -sk -o /dev/null -w '%{http_code}\n' $GW/api/v1/healthz; $C start redis; sleep 10; curl -sk -o /dev/null -w '%{http_code}\n' $GW/api/v1/readyz` → 503 kèm `"redis":"down"`; `200`; `200`. Test: `gotest ./internal/httpapi -run 'TestReadyz_DependencyDown|TestStartup_WaitsForDeps'`.
- AC15 (phân quyền). **Không áp dụng — story hạ tầng, chưa có endpoint cần vai trò.** Ràng buộc an toàn thay thế: cấu hình và log không lộ bí mật (AC3, AC4); `.env.example` chỉ có giá trị dev giả; secret không nằm trong `docker-compose.local.yml` (xem US-PG-07 AC17).
  Kiểm: `git grep -nE 'JWT_SECRET_KEY=[^ ]{20,}' -- ':!.env.example' ':!docs'` → không in gì.

### Ngoài phạm vi của story này
- Mã hoá AES-GCM (`APP_ENCRYPTION_KEY`) — P1 dùng lần đầu. Bảng, SQL, blob, outbox: US-PG-02. Middleware HTTP: US-PG-03. JWT: US-PG-04. SSE: US-PG-05.
- Metrics Prometheus, dashboard, cảnh báo (phase PR).

### Phụ thuộc
- Quyết định D48 (Go 1.27, Postgres 18, Redis 8), D22; `FEAT-scaffold` (`cmd/gateway`, `/healthz`, `platform.LoadConfig`, `ErrMissingEnv`) — nâng cấp, không viết lại.

---

## US-PG-02: Lập trình viên (dev các phase sau) muốn có schema nền `00001`, sqlc, kho file và outbox chạy được để mọi phase sau chỉ thêm bảng nghiệp vụ và nhà sản xuất, không dựng lại hạ tầng dữ liệu
Ưu tiên: Must · Ước lượng: L · Sprint: 2

Truy vết: PG L2 (8 checkbox).

### Tiêu chí nghiệm thu
- AC1. Given Postgres 18 trống When chạy `gateway migrate up` Then DB có đúng 5 bảng nghiệp vụ nền `users`, `audit_log`, `outbox`, `jobs`, `idempotency_keys` (+ `goose_db_version`), 3 kiểu enum `user_role`, `user_status`, `job_status`, extension `vector`; **không** bảng nghiệp vụ nào khác (không `courses`, `enrollments`, `documents`…).
  Kiểm:
  ```bash
  $PSQL -c "select tablename from pg_tables where schemaname='public' order by 1" | paste -sd' '
  # audit_log goose_db_version idempotency_keys jobs outbox users
  $PSQL -c "select typname from pg_type t join pg_namespace n on n.oid=typnamespace where nspname='public' and typtype='e' order by 1" | paste -sd' '   # job_status user_role user_status
  $PSQL -c "select extname from pg_extension where extname='vector'"        # vector
  ```
- AC2. Given bảng `users` When liệt kê cột Then có đúng 16 cột theo `SRS.md` mục 5.1 (tên, kiểu, nullable, mặc định), trong đó 6 cột dành cho P2/P5/P8 có sẵn: `email_verified_at`, `failed_logins`, `locked_until`, `status`, `ics_token`, `tracking_notice_ack_at`.
  Kiểm: `$PSQL -c "select column_name||':'||data_type||':'||is_nullable from information_schema.columns where table_name='users' and table_schema='public' order by ordinal_position" | wc -l` → `16`; `… | grep -cE '^(email_verified_at|failed_logins|locked_until|status|ics_token|tracking_notice_ack_at):'` → `6`; so từng dòng với bảng ở `SRS.md` 5.1. Test: `gotest ./internal/store -run TestSchema_UsersColumns`.
- AC3 (nhánh lỗi — ràng buộc). Given bảng `users` When thử ghi dữ liệu sai Then Postgres từ chối với đúng mã: email có chữ hoa (`A@X.com`) → `23514` (`users_email_lower_chk`); email trùng (`a@x.com` hai lần) → `23505` (`users_email_key`); `role='SUPERUSER'` → `22P02`; `failed_logins=-1` → `23514`; `status='ACTIVE'` mà `password_hash` rỗng → `23514` (`users_active_password_chk`); `student_code` cho vai khác STUDENT → `23514`; `status='INVITED'` không mật khẩu → hợp lệ.
  Kiểm: `gotest ./internal/store -run TestSchema_UsersConstraints` (bảng test: mỗi dòng một câu INSERT + SQLSTATE mong đợi).
- AC4. Given 4 bảng còn lại When kiểm Then cột, khoá, ràng buộc và index đúng `SRS.md` mục 5.2–5.5; tổng đúng 17 index (kể cả khoá chính) với đúng tên; mọi khoá chính là `uuid` mặc định `uuidv7()`.
  Kiểm:
  ```bash
  $PSQL -c "select indexname from pg_indexes where schemaname='public' and tablename<>'goose_db_version' order by 1" | paste -sd' '
  # audit_log_actor_created_idx audit_log_course_created_idx audit_log_entity_idx audit_log_pkey idempotency_keys_created_idx idempotency_keys_pkey idempotency_keys_user_endpoint_key_key jobs_active_idx jobs_owner_created_idx jobs_pkey outbox_pending_idx outbox_pkey outbox_stale_idx users_email_key users_ics_token_key users_pkey users_student_code_idx
  $PSQL -c "select count(*) from information_schema.columns where table_schema='public' and column_name='id' and column_default='uuidv7()'"   # 5
  ```
- AC5 (nhánh lỗi — append-only). Given `audit_log` có một dòng When chạy `UPDATE` hoặc `DELETE` hoặc `TRUNCATE` trên nó Then bị từ chối (SQLSTATE `42501`, thông báo "audit_log is append-only"); `INSERT` vẫn được.
  Kiểm: `$PSQL -c "insert into audit_log(entity,entity_id,action) values('t','1','x')"; $PSQL -c "update audit_log set action='y'" 2>&1 | grep -c 'append-only'` → `1`; tương tự DELETE, TRUNCATE. Test: `gotest ./internal/store -run TestSchema_AuditAppendOnly`.
- AC6 (nhánh lỗi — goose hai chiều). Given DB đã `up` When chạy `gateway migrate down` rồi `gateway migrate up` Then không lỗi, lược đồ **giống hệt** lúc đầu; `gateway migrate up` lần hai là no-op (mã 0); `down` không làm hỏng DB khác ngoài objects của 00001 (extension `vector` giữ lại).
  Kiểm:
  ```bash
  $C exec -T postgres pg_dump -U edupilot -d edupilot -s > /tmp/a.sql
  $C run --rm migrate down && $C run --rm migrate up
  $C exec -T postgres pg_dump -U edupilot -d edupilot -s > /tmp/b.sql; diff /tmp/a.sql /tmp/b.sql && echo SAME
  $PSQL -c "select version_id from goose_db_version where is_applied order by id desc limit 1"   # 1
  ```
  Test: `gotest ./db -run TestMigrations_RoundTrip`.
- AC7. Given xoá sạch volume Postgres When chạy `pnpm dev` Then service `migrate` chạy một lần, thoát mã 0, gateway và worker chỉ khởi động sau đó; không cần thao tác tay; migration dùng kết nối **trực tiếp** tới Postgres (không qua PgBouncer).
  Kiểm: `pnpm dev:down; docker volume rm edupilot_postgres_data; pnpm dev; $C ps -a --format '{{.Service}} {{.State}} {{.ExitCode}}' | grep ^migrate` → `migrate exited 0`; `$PSQL -c "select count(*) from goose_db_version where is_applied"` ≥ `2` (dòng 0 + 00001).
- AC8. Given `sqlc.yaml` When chạy `sqlc generate` Then mã sinh ra khớp bản đã commit (`sqlc diff` thoát 0, không in gì); enum Postgres thành kiểu Go (`UserRole`, `UserStatus`, `JobStatus`); cột `timestamptz` thành `time.Time` (`*time.Time` khi null); vector thành `pgvector.Vector`/`pgvector.HalfVector`; mọi truy vấn có `-- name:`; không `OFFSET` trong `queries/`.
  Kiểm: `cd backend-go && sqlc diff; echo rc=$?` → `rc=0` không in diff; `grep -cE '^type (UserRole|UserStatus|JobStatus) string' internal/store/models.go` → `3`; `grep -rniE '\boffset\b' internal/store/queries` → không in gì; `grep -rn 'float64' internal/store` → không in gì.
- AC9 (nhánh lỗi — sqlc lệch). Given sửa một file `internal/store/queries/*.sql` mà không chạy `sqlc generate` When chạy `sqlc diff` Then thoát khác 0 và nêu file lệch (đây là cổng CI, US-PG-07 AC12).
  Kiểm: `cd backend-go && echo '-- name: Tmp :one
  select 1;' >> internal/store/queries/users.sql; sqlc diff; echo rc=$?; git checkout -- internal/store/queries/users.sql` → `rc` khác 0.
- AC10 (quy ước vector). Given pgvector-go và PostgreSQL 18 When chạy test quy ước Then trong một bảng tạm có cột `vector(1536)`, index HNSW trên biểu thức `halfvec(1536)` (`halfvec_cosine_ops`, `m=16`, `ef_construction=64`) và truy vấn `ORDER BY embedding::halfvec(1536) <=> $1::halfvec(1536) LIMIT 5` có lọc `course_id` trong cùng truy vấn: plan dùng `Index Scan using … hnsw`, kết quả đầu là chính vector truy vấn; chạy được qua cả kết nối trực tiếp lẫn PgBouncer; bảng tạm bị xoá cuối test. Migration `00001` chứa khối **chú thích** SQL mẫu đúng các câu trên cho P1/P2 dùng lại.
  Kiểm: `gotest ./internal/store -run 'TestVectorConventions' ./internal/platform/db -run TestVector_ThroughPgBouncer`; `grep -c 'halfvec_cosine_ops' backend-go/db/migrations/00001_pg_platform.sql` ≥ `1` và dòng đó bắt đầu bằng `--`.
- AC11. Given `platform/blob` (MinIO) When chạy bộ test Then: `Put` rồi `Get` trả đúng byte (0 B, 1 B, 5 MiB, 20 MiB — 20 MiB đi bằng stream, không đọc hết vào RAM); `PresignGet` cho URL tải được bằng `http.Get` thường (200, đúng byte) và **hết hạn đúng TTL** (TTL 2 s → sau 3 s trả 403); `PresignPut` cho URL tải **lên** được rồi `Stat` thấy đúng kích thước; `Delete` rồi `Get` → `blob.ErrNotFound`; khoá sai (`..`, bắt đầu `/`, > 512 byte, ký tự lạ) → `blob.ErrInvalidKey` **không gọi mạng**; ctx bị huỷ giữa chừng → `Put` trả lỗi ctx và không để object nửa vời.
  Kiểm: `gotest ./internal/platform/blob -run 'TestBlob_RoundTrip|TestBlob_Presign|TestBlob_InvalidKey|TestBlob_Cancel'`.
- AC12. Given `BLOB_PUBLIC_ENDPOINT=localhost:9000` và gateway chỉ thấy MinIO ở `minio:9000` When sinh URL ký sẵn Then URL có host `localhost:9000` (trình duyệt dùng được), được ký **không cần gọi mạng** (cấu hình `BLOB_REGION`, mặc định `us-east-1`); URL host `minio:9000` không bao giờ trả cho client.
  Kiểm: `gotest ./internal/platform/blob -run 'TestPresign_PublicHost|TestPresign_NoNetwork'` (test cố ý trỏ endpoint công khai tới cổng đóng mà vẫn ký được).
- AC13 (outbox — cùng transaction). Given một nghiệp vụ thử ghi một dòng `outbox` trong **cùng** transaction với một dòng dữ liệu When commit Then có đúng 1 dòng `outbox` + 1 dòng dữ liệu; When rollback Then **0** dòng cả hai và không có tin nào trong Stream `outbox.dispatch`.
  Kiểm: `gotest ./internal/platform/outbox -run 'TestOutbox_SameTransaction_Commit|TestOutbox_SameTransaction_Rollback'`.
- AC14 (outbox — chạy thường). Given 100 dòng outbox topic `test.ok` và **hai** worker chạy đồng thời When đợi ≤ 10 s Then cả 100 có `dispatched_at`, handler được gọi đúng **1 lần cho mỗi `outbox.id`** (khử trùng), `XPENDING` của nhóm = 0, `XLEN outbox.dispatch.dead` = 0.
  Kiểm: `gotest ./internal/platform/outbox -run TestOutbox_TwoWorkers_ExactlyOnce`.
- AC15 (nhánh lỗi — retry rồi dead-letter). Given handler của topic `test.fail` luôn trả lỗi When có một dòng outbox topic đó Then handler được gọi **đúng 4 lần** (1 lần đầu + 3 lần thử lại, có backoff), sau đó dòng có `dead_at`, `attempts = 4`, `last_error` ≠ rỗng, `dispatched_at` NULL, và Stream `outbox.dispatch.dead` có đúng 1 tin chứa `outbox_id`, `topic`, `error`; tin gốc đã `XACK`. Handler **panic** hoặc topic **chưa đăng ký** cũng đi cùng đường (consumer không chết).
  Kiểm: `gotest ./internal/platform/outbox -run 'TestOutbox_RetryThenDead|TestOutbox_PanicIsFailure|TestOutbox_UnknownTopicDead'` (backoff rút còn 20 ms qua cấu hình test).
- AC16 (nhánh lỗi — consumer chết giữa chừng). Given consumer đọc một tin rồi bị giết trước khi `XACK` When quá `OUTBOX_CLAIM_IDLE` Then tin được consumer khác nhận lại (`XAUTOCLAIM`) và xử lý; không tin nào mất; dòng outbox không bao giờ kẹt ở trạng thái "đã xếp hàng" quá `OUTBOX_STALE_AFTER` mà không được đẩy lại.
  Kiểm: `gotest ./internal/platform/outbox -run 'TestOutbox_ConsumerCrash_Reclaimed|TestOutbox_StaleEnqueuedRequeued'`.
- AC17 (phân quyền). **Không áp dụng — chưa có API nghiệp vụ.** Ràng buộc an toàn thay thế ở tầng dữ liệu: `audit_log` chỉ-thêm (AC5); mật khẩu chỉ ở dạng bcrypt (US-PG-04 AC8); `idempotency_keys` khoá theo `user_id` nên người này không đọc được phản hồi của người kia (US-PG-03 AC12).

### Ngoài phạm vi của story này
- Mọi bảng nghiệp vụ (`courses`, `enrollments`, `documents`, `notifications`, … — phase đầu tiên dùng bảng là phase tạo bảng, D45); `llm_*` (P1).
- Vai trò Postgres riêng cho gateway / worker (SYSTEM_DESIGN §3.6) — ghi nợ cho phase PR (`QUESTIONS.md` Q9).
- Nhà sản xuất outbox thật (P4 là người dùng đầu tiên); handler nghiệp vụ.
- Dọn dẹp `idempotency_keys` quá 24 h bằng cron (phase sau; cột `created_at` có index sẵn).

### Phụ thuộc
- US-PG-01 (config, redis, db pool, log). Migration `00001`. Quyết định D45, D48, D52.

---

## US-PG-03: Lập trình viên (dev các phase sau) muốn có chuẩn HTTP dùng chung — lỗi thống nhất, phân trang con trỏ, Idempotency-Key, khoá lạc quan, ETag, việc dài 202 — để mọi API về sau theo đúng một quy ước mà không tự chế
Ưu tiên: Must · Ước lượng: L · Sprint: 2

Truy vết: PG L3 (7 checkbox). Các helper được kiểm qua endpoint **chỉ có khi dựng bằng build tag `testroutes`** (`SRS.md` 6.3) trên bảng thử `_test_items`: test Go do fixture tạo; khi chạy trong compose thì gói `internal/testroutes` tự `CREATE TABLE IF NOT EXISTS _test_items` lúc khởi động (không nằm trong migration; gói không được biên dịch vào binary mặc định).

### Tiêu chí nghiệm thu
- AC1. Given mọi response (kể cả lỗi, 304, SSE) When đọc header Then có `X-Request-Id` (nếu client gửi `X-Request-Id` hợp lệ — 1–64 ký tự `[A-Za-z0-9._-]` — thì giữ nguyên, ngược lại gateway sinh) và `X-Instance-Id` (tên container); `Content-Type` của lỗi là `application/json; charset=utf-8`.
  Kiểm: `curl -sk -D- -o /dev/null -H 'X-Request-Id: abc-123' $GW/api/v1/healthz | grep -iE '^x-request-id: abc-123|^x-instance-id'`; `curl -sk -D- -o /dev/null -H 'X-Request-Id: bad id!' $GW/api/v1/healthz | grep -i '^x-request-id'` ≠ `bad id!`. Test: `gotest ./internal/httpapi -run TestRequestID`.
- AC2 (nhánh lỗi — panic). Given một handler panic (`GET /api/v1/_test/panic`) When gọi Then trả **500** `INTERNAL` với thông điệp chung (không chứa chữ "panic", đường dẫn file hay SQL), log mức `error` có stack và `trace_id`, tiến trình sống, request kế tiếp 200; panic `http.ErrAbortHandler` không bị nuốt.
  Kiểm: `tmode; curl -sk -w '\n%{http_code}\n' $GW/api/v1/_test/panic | tail -3; curl -sk -o /dev/null -w '%{http_code}\n' $GW/api/v1/healthz` → `INTERNAL`, `500`, `200`. Test: `gotest ./internal/httpapi -run TestRecover`.
- AC3. Given `APP_CORS_ALLOWED_ORIGINS=http://localhost:3000` When preflight `OPTIONS` từ `Origin: http://localhost:3000` Then **204** với `Access-Control-Allow-Origin` = đúng origin đó, `Allow-Methods`, `Allow-Headers` (gồm `Authorization, Content-Type, Idempotency-Key, If-Match, If-None-Match, Last-Event-ID, X-Request-Id`), `Allow-Credentials: true`, `Max-Age`, `Vary: Origin`; `Expose-Headers` gồm `ETag, X-Request-Id, Retry-After, Idempotent-Replayed`. Origin lạ → không có header `Access-Control-Allow-Origin` (và không bao giờ `*`).
  Kiểm: `curl -sk -X OPTIONS -D- -o /dev/null -H 'Origin: http://localhost:3000' -H 'Access-Control-Request-Method: POST' $GW/api/v1/jobs/x | grep -i '^access-control'`; lặp với `Origin: https://evil.example` → không có dòng `access-control-allow-origin`. Test: `gotest ./internal/httpapi -run TestCORS`.
- AC4. Given định dạng lỗi When gọi các tình huống gây 400, 401, 403, 404, 405, 409, 413, 415, 422, 429, 500, 503, 504 Then **mọi** thân lỗi là JSON `{code, message, trace_id}` + `details?`, `retry_after?` (không trường lạ), `code` nằm trong bảng mã ở `SRS.md` 6.1 đúng status; `message` tiếng Việt, không lộ chi tiết nội bộ; route không có → 404 JSON (không phải văn bản `404 page not found`); sai method → 405 JSON kèm header `Allow`; 429 và 503 có `retry_after` (giây, số nguyên ≥ 1) trùng header `Retry-After`.
  Kiểm: `tmode; for s in 400 401 403 404 405 409 413 415 422 429 500 503 504; do curl -sk "$GW/api/v1/_test/error/$s" | jq -c '[.code,.message!=null,.trace_id!=null]'; done`; `curl -sk -X DELETE -D- $GW/api/v1/healthz | grep -iE '^allow|code'`. Test: `gotest ./internal/httpapi -run 'TestErrorFormat_AllStatuses|TestNotFoundAndMethodNotAllowed'`.
- AC5 (nhánh lỗi — validation). Given `POST /api/v1/_test/items` When thân JSON hỏng → **400** `BAD_REQUEST`; thiếu `name` / `name` rỗng / dài hơn 200 ký tự / có trường lạ → **422** `VALIDATION_FAILED` với `details` là mảng `{field, code, message}` liệt kê **mọi** lỗi cùng lúc; `Content-Type` khác `application/json` → **415** `UNSUPPORTED_MEDIA_TYPE`.
  Kiểm: `curl -sk -X POST -H 'Content-Type: application/json' -H "Authorization: Bearer $(tok STUDENT)" -d '{"name":"","extra":1}' $GW/api/v1/_test/items | jq -c '.code,(.details|map(.field))'` → `"VALIDATION_FAILED"`, `["name","extra"]`. Test: `gotest ./internal/httpapi -run TestValidation`.
- AC6. Given rate limit trên Redis (`RATE_LIMIT_IP_PER_MIN`, `RATE_LIMIT_USER_PER_MIN`) When đặt giới hạn IP = 5 Then request thứ 6 trong cùng phút nhận **429** `RATE_LIMITED` có `Retry-After` (1–60) = `retry_after`; `X-RateLimit-Limit` / `X-RateLimit-Remaining` có mặt; `/healthz`, `/api/v1/healthz`, `/api/v1/readyz` **miễn giới hạn**; sang cửa sổ mới lại 200; bộ đếm theo người dùng tách khỏi theo IP; **hai bản gateway dùng chung bộ đếm** (3 request vào A + 3 vào B với giới hạn 5 → đúng 1 request 429); IP lấy từ `X-Forwarded-For` chỉ khi nguồn kết nối thuộc `TRUSTED_PROXY_CIDRS`.
  Kiểm: `gotest ./internal/httpapi -run 'TestRateLimit_IP|TestRateLimit_User|TestRateLimit_SharedAcrossInstances|TestRateLimit_ExemptHealth|TestRateLimit_ForwardedFor'`; tay: `RATE_LIMIT_IP_PER_MIN=5 $C up -d --force-recreate gateway; for i in $(seq 7); do curl -sk -o /dev/null -w '%{http_code} ' $GW/api/v1/jobs/x; done` → 5 lần ≠ 429 rồi `429 429`.
- AC7 (nhánh lỗi — Redis chết khi rate limit). Given Redis không phản hồi When request thường (không cần Idempotency-Key) đến Then **cho qua** (fail-open) kèm log `warn` một lần mỗi 10 s, không treo quá 50 ms.
  Kiểm: `gotest ./internal/httpapi -run TestRateLimit_RedisDown_FailOpen`.
- AC8. Given helper phân trang con trỏ `?cursor=&limit=` When gọi `GET /api/v1/_test/items` Then mặc định 30 mục; `limit=100` được; `limit=101`, `limit=0`, `limit=abc` → **422** (`details[0].field = "limit"`); trả `{"items":[…],"next_cursor":"…"}` sắp theo `(created_at DESC, id DESC)`; trang cuối `next_cursor` là `null` (có mặt, không bỏ trường); danh sách rỗng là `"items":[]` (không `null`); cursor rác / sai phiên bản / sai kiểu → **422** `INVALID_CURSOR`.
  Kiểm: `curl -sk -H "Authorization: Bearer $(tok STUDENT)" "$GW/api/v1/_test/items?limit=101" | jq -r '.code,.details[0].field'` → `VALIDATION_FAILED`, `limit`; `…?cursor=@@@` → `INVALID_CURSOR`. Test: `gotest ./internal/httpapi -run 'TestPagination_Defaults|TestPagination_LimitBounds|TestPagination_InvalidCursor|TestPagination_Shape'`.
- AC9 (nhánh lỗi — không sót, không lặp). Given 250 bản ghi, `limit=100` When đi hết các trang Then hợp các trang = đúng 250 id duy nhất. Given 300 bản ghi có **cùng** `created_at` Then vẫn không sót, không lặp (phân định bằng `id`). Given trong lúc đi từ trang 1 sang trang 2 có 5 bản ghi **mới hơn** và 5 bản ghi **cũ hơn con trỏ** chen vào, và 3 bản ghi đã đọc bị xoá Then mọi bản ghi gốc chưa xoá xuất hiện đúng 1 lần, 5 bản ghi chen vào vùng chưa đọc xuất hiện đúng 1 lần, không id nào lặp, không lỗi.
  Kiểm: `gotest ./internal/httpapi -run 'TestCursor_NoSkipNoDup|TestCursor_SameTimestamp|TestCursor_InsertAndDeleteDuringScan'`.
- AC10. Given cursor When giải mã Then là base64url **không đệm** của JSON `{"v":1,"t":<created_at tính bằng micro-giây UTC>,"i":"<uuid>"}`, dài ≤ 120 ký tự; truy vấn trang dùng so sánh hàng `(created_at, id) < ($1, $2)` và index `(created_at, id)`: plan truy vấn trên 10.000 dòng là `Index Scan`/`Index Only Scan`, **không** `Sort` và **không** `OFFSET` ở bất kỳ file `.sql` / `.go` nào ngoài test.
  Kiểm: `grep -rniE '\boffset\b' backend-go/internal --include=*.sql --include=*.go | grep -v '_test.go'` → không in gì; `gotest ./internal/httpapi -run 'TestCursor_Format|TestCursor_UsesIndex'`.
- AC11. Given `Idempotency-Key` When gửi cùng một `POST /api/v1/_test/items` hai lần với cùng khoá và cùng thân Then **đúng 1 bản ghi** trong `_test_items`; lần hai trả **status, thân, `Content-Type` y hệt** lần một kèm header `Idempotent-Replayed: true` (lần một không có header này); khoá khác → bản ghi mới.
  Kiểm: `tmode; H="Authorization: Bearer $(tok STUDENT U1)"; for i in 1 2; do curl -sk -X POST -H "$H" -H 'Content-Type: application/json' -H 'Idempotency-Key: idem-0001-aaaa' -d '{"name":"x"}' -D- $GW/api/v1/_test/items | grep -iE '^HTTP|idempotent-replayed|"id"'; done; $PSQL -c "select count(*) from _test_items where name='x'"` → `1`, hai thân giống nhau. Test: `gotest ./internal/httpapi -run 'TestIdempotency_DoubleSend_OneRecord|TestIdempotency_ReplayIdentical'`.
- AC12 (nhánh lỗi — đua và biên). Given 50 goroutine cùng gửi một khoá When chạy Then đúng 1 bản ghi; các request còn lại nhận **cùng thân** (replay) hoặc **409** `IDEMPOTENCY_IN_PROGRESS` kèm `Retry-After` — không bao giờ 2 bản ghi, không 5xx. Given cùng khoá nhưng thân khác → **422** `IDEMPOTENCY_KEY_REUSED`. Given endpoint **bắt buộc** khoá mà thiếu header → **422** `IDEMPOTENCY_KEY_REQUIRED`; khoá sai định dạng (ngắn hơn 8, dài hơn 128, ký tự ngoài `[A-Za-z0-9._:-]`) → **422** `VALIDATION_FAILED`. Cùng khoá của **người dùng khác** hoặc **endpoint khác** là độc lập. Phản hồi 5xx / 429 **không** được lưu (gửi lại thì xử lý lại). Khoá và thân không bao giờ vào log.
  Kiểm: `gotest ./internal/httpapi -run 'TestIdempotency_Concurrent50|TestIdempotency_KeyReused|TestIdempotency_Required|TestIdempotency_BadKey|TestIdempotency_ScopedByUserAndEndpoint|TestIdempotency_5xxNotStored|TestIdempotency_NotLogged'`.
- AC13. Given phản hồi đã lưu When đo TTL Then khoá `ep:idem:…` có TTL 86.300–86.400 s (24 h); khoá chạy-dở `…:lock` TTL ≤ 30 s. Given Redis bị xoá sạch (`FLUSHALL`) rồi gửi lại cùng khoá Then phản hồi được khôi phục từ bảng `idempotency_keys` (cùng khoá `(user_id, endpoint, key)`), vẫn **1 bản ghi**, header `Idempotent-Replayed: true`. Given Redis chết khi gọi endpoint bắt buộc khoá Then **503** `SERVICE_UNAVAILABLE` (fail-closed) và **không** tạo bản ghi.
  Kiểm: `$RDS ttl "$($RDS --scan --pattern 'ep:idem:*' | grep -v ':lock$' | head -1)"` ∈ [86300, 86400]; test: `gotest ./internal/httpapi -run 'TestIdempotency_TTL|TestIdempotency_RedisFlushedFallsBackToTable|TestIdempotency_RedisDown_FailClosed'`.
- AC14. Given tài nguyên có cột `version` (`_test_items`) When `PUT /api/v1/_test/items/{id}` với `version` đang giữ Then 200, `version` +1; với `version` cũ → **409** `VERSION_CONFLICT` có `details.current_version` = giá trị hiện tại, `details.current` = biểu diễn hiện tại của tài nguyên và header `ETag` hiện tại; thiếu `version` → 422; id không có → 404. 20 request song song cùng `version` = n → **đúng 1** thành công, 19 lần 409, `version` cuối = n+1, không mất cập nhật. `If-Match: W/"v<n>"` tương đương trường `version`; có cả hai mà khác nhau → 422.
  Kiểm: `gotest ./internal/httpapi -run 'TestOptimisticLock_Stale409|TestOptimisticLock_Concurrent20|TestOptimisticLock_IfMatch|TestOptimisticLock_Missing'`.
- AC15. Given `ETag` When `GET /api/v1/_test/items/{id}` hai lần Then phản hồi đầu có `ETag: W/"v<version>"`, `Cache-Control: private, no-cache`, `Vary: Authorization`; gửi `If-None-Match` khớp (một giá trị, danh sách, hoặc `*`) → **304** thân rỗng + `ETag`; không khớp → 200; sau khi `PUT` thì ETag đổi; danh sách `GET …/items` có ETag băm thân (`W/"<base64url 16 ký tự>"`); `If-None-Match` trên POST / PUT bị bỏ qua.
  Kiểm: `E=$(curl -sk -D- -o /dev/null -H "$H" $GW/api/v1/_test/items/$ID | awk -F': ' 'tolower($1)=="etag"{print $2}' | tr -d '\r'); curl -sk -o /dev/null -w '%{http_code}\n' -H "$H" -H "If-None-Match: $E" $GW/api/v1/_test/items/$ID` → `304`. Test: `gotest ./internal/httpapi -run 'TestETag_NotModified|TestETag_ChangesAfterUpdate|TestETag_MultipleValues'`.
- AC16. Given việc dài When `POST /api/v1/_test/jobs {"steps":4}` (kind `test.progress`) Then **202** `{"job_id":"<uuid>"}` + `Location: /api/v1/jobs/<id>`; dòng `jobs` ở `QUEUED` và dòng `outbox` `job.enqueue` được ghi cùng transaction; `GET /api/v1/jobs/{id}` của chủ job trả `{id, kind, status, progress, result?, error?, created_at, updated_at, finished_at?}`; worker chuyển `QUEUED → RUNNING → SUCCEEDED`, `progress` tăng không giảm tới 100, `finished_at` có khi xong.
  Kiểm: `tmode; ID=$(curl -sk -X POST -H "$H" -H 'Content-Type: application/json' -d '{"steps":4}' $GW/api/v1/_test/jobs | jq -r .job_id); sleep 2; curl -sk -H "$H" $GW/api/v1/jobs/$ID | jq -c '[.status,.progress,.finished_at!=null]'` → `["SUCCEEDED",100,true]`. Test: `gotest ./internal/jobs ./internal/httpapi -run 'TestJobs_Lifecycle|TestJobs_ProgressMonotonic|TestJobs_EnqueueAtomic'`.
- AC17 (nhánh lỗi). Given job kind `test.fail` When worker xử lý Then `status = FAILED`, `error.code` có giá trị, `progress` giữ nguyên, `finished_at` có; handler panic cũng thành `FAILED` (không treo `RUNNING`); `GET /api/v1/jobs/{id}` với id không tồn tại hoặc không phải uuid → **404** `NOT_FOUND`.
  Kiểm: `gotest ./internal/jobs -run 'TestJobs_Failed|TestJobs_PanicFailed'`; `curl -sk -o /dev/null -w '%{http_code}\n' -H "$H" $GW/api/v1/jobs/not-a-uuid` → `404`.
- AC18 (phân quyền + mặc định an toàn). Given job của `U1` When `U2` (STUDENT) gọi `GET /api/v1/jobs/{id}` Then **404** `NOT_FOUND` (không lộ sự tồn tại); ADMIN → 200; không có token → **401**. Given **image mặc định** (`gateway`, không có build tag `testroutes`) When gọi **mọi** đường dẫn thử của `SRS.md` 6.3 (kể cả khi `APP_ENV=test`, với token ADMIN hợp lệ) Then **404** `NOT_FOUND` (route không tồn tại, không phải 403 / 401); binary mặc định **không có** gói `internal/testroutes`: `go tool nm` không in symbol nào của gói, và chuỗi `/api/v1/_test/` không có trong file thực thi trong image. Given binary dựng với tag nhưng `APP_ENV=production` Then thoát mã 1 ngay, log nêu `APP_ENV` (chặn nhầm image thử ra production).
  Kiểm: `curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $(tok STUDENT $U2)" $GW/api/v1/jobs/$ID` → `404`; `curl -sk -o /dev/null -w '%{http_code}\n' $GW/api/v1/jobs/$ID` → `401`; bản mặc định: `dmode; for p in items items/$U1 slow db-sleep redis-block panic error/500 whoami rbac/admin rbac/staff courses/$U1/ping jobs events; do curl -sk -o /dev/null -w '%{http_code} ' -H "Authorization: Bearer $(tok ADMIN $U1)" $GW/api/v1/_test/$p; done` → 13 lần `404`; `cd backend-go && go build -o /tmp/gw-default ./cmd/gateway && go tool nm /tmp/gw-default | grep -c 'internal/testroutes'` → `0`, còn `go build -tags testroutes -o /tmp/gw-test ./cmd/gateway && go tool nm /tmp/gw-test | grep -c 'internal/testroutes'` ≥ `1`; `docker create --name gwchk edupilot-gateway >/dev/null && docker cp gwchk:/gateway /tmp/gw-img && docker rm gwchk >/dev/null; grep -c '/api/v1/_test/' /tmp/gw-img` → `0`. Test (không tag): `cd backend-go && go test -count=1 ./cmd/gateway -run 'TestDefaultBinary_NoTestRoutes'` — tự `go build` bản mặc định rồi kiểm `go tool nm` và 404 cho cả 13 đường dẫn; test (có tag): `gotest ./internal/httpapi -run 'TestJobs_Ownership'` và `gotest ./cmd/gateway -run 'TestTestBinary_RefusesProduction'` (cấu hình đủ biến, chỉ `APP_ENV=production` → thoát 1, log nêu `APP_ENV`).

### Ngoài phạm vi của story này
- Mọi endpoint nghiệp vụ; danh sách thật có phân trang (mỗi phase dùng helper). Huỷ job (`CANCELLED` chỉ dành chỗ). Dọn khoá idempotency bằng cron.
- Auth và RBAC (US-PG-04); SSE (US-PG-05).

### Phụ thuộc
- US-PG-01, US-PG-02 (bảng `jobs`, `outbox`, `idempotency_keys`, consumer `outbox.dispatch`); US-PG-04 cho token trong AC.

---

## US-PG-04: Lập trình viên (dev các phase sau) muốn có JWT, bcrypt, RBAC và khung `CourseAccessGuard` chạy được để mọi phase sau lấy danh tính từ claim và chỉ việc nối `enrollments`
Ưu tiên: Must · Ước lượng: M · Sprint: 2

Truy vết: PG L4 (3 checkbox, trong đó mục 3 là ranh giới "không làm trùng P2").

### Tiêu chí nghiệm thu
- AC1. Given token do `auth.Issue` hoặc `gateway token` tạo When giải mã Then header `alg=HS256`, `typ=JWT`; claims gồm `sub` (uuid), `role` (một trong `ADMIN|TEACHER|TA|STUDENT`), `email`, `jti` (ngẫu nhiên 128 bit, base64url, duy nhất trong 10.000 lần cấp), `iat`, `nbf`, `exp`, `iss="edupilot"`, `aud="edupilot-api"`; `exp − iat` = `JWT_EXPIRATION` (mặc định 15 phút).
  Kiểm: `tok STUDENT $U1 | cut -d. -f2 | tr '_-' '/+' | base64 -d 2>/dev/null | jq -c 'keys'` → `["aud","email","exp","iat","iss","jti","nbf","role","sub"]`; `… | jq '.exp-.iat'` → `900`. Test: `gotest ./internal/auth -run 'TestJWT_Claims|TestJWT_JTIUnique'`.
- AC2 (nhánh lỗi — token sai). Given request tới endpoint cần đăng nhập (`GET /api/v1/jobs/{id}`) When token: hết hạn → **401** `TOKEN_EXPIRED`; sai chữ ký, `alg=none`, `alg=HS512`, `alg=RS256`, thiếu `exp`, `role` ngoài danh sách (`SUPERUSER`), sai `iss`/`aud`, `nbf` ở tương lai, chuỗi hỏng (2 đoạn, rác) → **401** `TOKEN_INVALID`; không có header, `Authorization: Basic …`, `Bearer ` rỗng → **401** `UNAUTHENTICATED`. Mọi 401 có `WWW-Authenticate: Bearer realm="edupilot"` (+ `error="invalid_token"` với hai mã đầu), thân đúng định dạng lỗi có `trace_id`, và **không** nói thêm lý do chi tiết ngoài `code`.
  Kiểm: `U=$GW/api/v1/jobs/00000000-0000-7000-8000-000000000009; for t in "$(tok STUDENT $U1 --ttl -1m)" "$(tok STUDENT $U1)x" garbage ""; do curl -sk -D- -H "Authorization: Bearer $t" $U | grep -iE '^HTTP|www-authenticate|"code"'; done` → `TOKEN_EXPIRED` rồi `TOKEN_INVALID` rồi `TOKEN_INVALID` rồi `UNAUTHENTICATED`. Test: `gotest ./internal/auth -run 'TestVerify_Table'` (≥ 14 dòng bảng cho các trường hợp trên).
- AC3. Given sai lệch đồng hồ When token hết hạn cách đây 3 s Then còn được chấp nhận (leeway 5 s); hết hạn cách đây 10 s → `TOKEN_EXPIRED`; đồng hồ lấy từ `platform/clock` (test dùng đồng hồ giả, không `time.Sleep`).
  Kiểm: `gotest ./internal/auth -run TestVerify_Leeway`.
- AC4. Given danh tính lấy từ claim (D47 mục 6) When Postgres bị tạm dừng (`docker pause`) Then `GET /api/v1/_test/whoami` với token hợp lệ vẫn **200** và trả đúng `sub`, `role`; số truy vấn DB trong một request đã xác thực tới route chỉ-đọc-claim là **0**.
  Kiểm: `tmode; docker pause $($C ps -q postgres); curl -sk -H "Authorization: Bearer $(tok TA $U1)" $GW/api/v1/_test/whoami | jq -c '[.sub,.role]'; docker unpause $($C ps -q postgres)` → `["00000000-0000-7000-8000-000000000001","TA"]`. Test: `gotest ./internal/auth -run TestAuth_NoDBQueryPerRequest` (bộ đếm tracer = 0).
- AC5 (phân quyền). Given middleware `auth.RequireRole` When gọi `GET /api/v1/_test/rbac/admin` (chỉ ADMIN) và `GET /api/v1/_test/rbac/staff` (TEACHER, TA) Then ma trận: `admin` → ADMIN 200, TEACHER/TA/STUDENT **403** `FORBIDDEN`, ẩn danh **401**; `staff` → TEACHER/TA 200, ADMIN/STUDENT **403**, ẩn danh 401. 403 có thân đúng định dạng và `details.reason = "role"`.
  Kiểm: `tmode; for r in ADMIN TEACHER TA STUDENT; do for p in admin staff; do printf '%s %s ' $r $p; curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $(tok $r $U1)" $GW/api/v1/_test/rbac/$p; done; done` → ADMIN: 200/403, TEACHER: 403/200, TA: 403/200, STUDENT: 403/403. Test: `gotest ./internal/auth -run TestRBAC_Matrix`.
- AC6 (phân quyền — claim thắng DB). Given user `U1` có `role=ADMIN` trong bảng `users` nhưng token mang `role=STUDENT` When gọi route ADMIN Then **403** (tin claim, không tra DB); hệ quả được ghi: đổi vai trò có hiệu lực khi token hết hạn (≤ 15 phút) — phase P2 xử lý thu hồi.
  Kiểm: `gotest ./internal/auth -run TestRBAC_ClaimWinsOverDB`.
- AC7 (khung CourseAccessGuard). Given route có `{courseId}` (`GET /api/v1/_test/courses/{courseId}/ping`) When dùng guard mặc định (resolver "từ chối tất cả") Then **403** `FORBIDDEN` cho mọi vai, **kể cả ADMIN** (ADMIN không thấy nội dung lớp, PRD §3); Given resolver giả cho STUDENT thuộc lớp A: lớp A → 200 và context có `{CourseID, CourseRole}`, lớp B → 403; `courseId` không phải uuid → **404**; resolver trả lỗi → **503** `SERVICE_UNAVAILABLE` (không cho qua); resolver được gọi đúng 1 lần mỗi request và guard **không** cache kết quả giữa các request.
  Kiểm: `gotest ./internal/auth -run 'TestCourseAccessGuard_DefaultDenyAll|TestCourseAccessGuard_Membership|TestCourseAccessGuard_BadID|TestCourseAccessGuard_ResolverError|TestCourseAccessGuard_NoCache'`; tay: `curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $(tok ADMIN $U1)" $GW/api/v1/_test/courses/00000000-0000-7000-8000-0000000000aa/ping` → `403`.
- AC8. Given `x/crypto/bcrypt` When băm Then chuỗi khớp `^\$2[aby]\$12\$.{53}$` (cost 12 mặc định, `BCRYPT_COST` cho phép 4–14 và test dùng 4; production không dưới 10); hai lần băm cùng mật khẩu cho hai chuỗi khác nhau; `CheckPassword` đúng / sai / rỗng; mật khẩu > 72 byte → `auth.ErrPasswordTooLong` (không cắt im lặng); so sánh tốn thời gian như nhau cho mật khẩu sai/đúng (dùng `bcrypt.CompareHashAndPassword`, không so chuỗi).
  Kiểm: `gotest ./internal/auth -run 'TestBcrypt_Format|TestBcrypt_DefaultCost|TestBcrypt_TooLong|TestBcrypt_Check'`.
- AC9. Given mật khẩu rõ hoặc hash When chạy toàn bộ test auth với bắt log Then không dòng log nào và không thân response nào chứa mật khẩu rõ, hash bcrypt hay token đầy đủ (chỉ `jti` rút gọn 8 ký tự được phép trong log).
  Kiểm: `gotest ./internal/auth ./internal/httpapi -run 'TestAuth_NoSecretsInLogs'`.
- AC10 (đúng phạm vi — không làm trùng P2). Given PG When liệt kê route Then **không** có `/auth/*` (đăng ký, đăng nhập, refresh, xác minh…), `/me/*`, `/admin/*`; `openapi.yaml` không có đường dẫn nào chứa `/auth/`.
  Kiểm: `grep -c '/auth/' backend-go/api/openapi.yaml` → `0`; `curl -sk -o /dev/null -w '%{http_code}\n' -X POST $GW/api/v1/auth/login` → `404`.
- AC11 (nhánh lỗi — công cụ dev). Given `gateway token` When chạy với `APP_ENV=production` Then thoát mã 1 và không in token; thiếu `JWT_SECRET_KEY` → thoát mã 1 nêu tên biến; `--role` ngoài danh sách → thoát mã 1; chạy bình thường in đúng **một** dòng là JWT.
  Kiểm: `APP_ENV=production JWT_SECRET_KEY="$SECRET" go run ./cmd/gateway token --role ADMIN; echo rc=$?` → `rc=1`; `go run ./cmd/gateway token --role ROOT; echo rc=$?` → `rc=1`. Test: `gotest ./cmd/gateway -run TestTokenCommand`.
- AC12. Given đồng thời Then không biến toàn cục giữ trạng thái người dùng: `Principal` chỉ đi qua `context.Context`; chạy `-race -count=20` các test auth song song không báo race.
  Kiểm: `gotest ./internal/auth -count=20 -run 'TestPrincipal_ContextOnly|TestAuth_ParallelRequests'`.

### Ngoài phạm vi của story này
- Đăng ký, xác minh email, link mời, quên mật khẩu, khoá khi dò, refresh xoay vòng, `auth_sessions`, thu hồi phiên (P2, D36). Nối `CourseAccessGuard` vào `enrollments` (P2). OIDC (PR).
- Danh sách thu hồi `jti` (P2 dùng `jti` đã có sẵn trong claim).

### Phụ thuộc
- US-PG-03 (định dạng lỗi, middleware); cột `users.role` (US-PG-02); quyết định D36, D47.

---

## US-PG-05: Lập trình viên (dev các phase sau) muốn có hạ tầng SSE chạy đúng khi có nhiều bản gateway — id sự kiện, heartbeat, nối lại bằng `Last-Event-ID`, giới hạn kết nối — để thông báo, tiến độ việc và stream chat chỉ việc phát sự kiện
Ưu tiên: Must · Ước lượng: M · Sprint: 2

Truy vết: PG L5 (4 checkbox). Endpoint nền tảng: `GET /api/v1/events` (xác thực bằng header `Authorization: Bearer`, **không** nhận token qua query string). Phát sự kiện bằng `sse.Publisher.Publish(ctx, userID, Event)`; endpoint thử `POST /api/v1/_test/events` chỉ có ở binary dựng bằng build tag `testroutes` (`gateway-test`).

### Tiêu chí nghiệm thu
- AC1. Given người dùng đã xác thực When `GET /api/v1/events` Then **200**, `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-transform`, `X-Accel-Buffering: no`; byte đầu tiên là `retry: 3000`, rồi `event: ready` với `data: {"connection_id":"…","server_time":"…"}` **không có `id:`** (để không ghi đè `Last-Event-ID`); sự kiện trạng thái đầu tiên đến trong **≤ 300 ms** (p95 trên 50 kết nối).
  Kiểm: `curl -sk -N --max-time 2 -H "Authorization: Bearer $(tok STUDENT $U1)" -w '\nttfb=%{time_starttransfer}\n' $GW/api/v1/events | head -8` → có `retry: 3000`, `event: ready`, `ttfb` ≤ 0,3. Test: `gotest ./internal/httpapi/sse -run 'TestSSE_Headers|TestSSE_FirstEventLatency'`.
- AC2. Given sự kiện phát cho người dùng When client đọc Then khung đúng `id: <ms>-<seq>\nevent: <type>\ndata: <một dòng JSON>\n\n`; `id` là id của Redis Stream, **tăng nghiêm ngặt**; `type` khớp `^[a-z][a-z0-9_.]{0,63}$`; dữ liệu > 64 KiB bị từ chối ở `Publish` (`ErrEventTooLarge`), type sai → `ErrInvalidEventType`.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_Framing|TestPublish_Validation|TestSSE_IDsMonotonic'`.
- AC3. Given kết nối nhàn rỗi When qua `SSE_HEARTBEAT` (mặc định **25 s**) Then server gửi dòng chú thích `: hb` (không phải sự kiện, không `id`); heartbeat đều đặn, không bị gộp lô.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_Heartbeat|TestConfig_SSEDefaults'` (heartbeat 200 ms trong test; test cấu hình khẳng định mặc định 25 s); tay: `timeout 30 curl -sk -N -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/events | grep -c '^: hb'` → `1` (qua Caddy).
- AC4 (nhánh lỗi — client ngắt). Given 100 chu kỳ mở / ngắt kết nối ngẫu nhiên (ngắt khi đang chờ, khi đang nhận, khi vừa gửi heartbeat) When kết thúc Then số goroutine về mức ban đầu (±5), `ZCARD ep:sse:conn:<uid>` = 0, `PUBSUB NUMSUB ep:sse:ch:<uid>` = 0 trong ≤ 2 s; không dòng log lỗi.
  Kiểm: `gotest ./internal/httpapi/sse -race -run TestSSE_NoLeakOnClientDisconnect`.
- AC5 (nhánh lỗi — giới hạn kết nối). Given `U1` đã mở **2** stream When mở stream thứ **3** Then **429** `SSE_LIMIT_REACHED` (JSON, `Retry-After`, **trước** khi mở stream); `U2` không bị ảnh hưởng; đóng một stream của `U1` thì stream mới được nhận; **đếm chung giữa hai bản gateway** (A giữ 2, B nhận 429). Gateway chết đột ngột (`kill -9`) → chỗ của nó tự hết sau `SSE_CONN_TTL` (150 s; test 1 s).
  Kiểm: `tmode; for i in 1 2 3; do curl -sk -N -o /dev/null -m 5 -w '%{http_code}\n' -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/events & sleep 0.5; done; wait` → hai `000` (hết `-m`, vẫn đang stream) rồi một `429`. Test: `gotest ./internal/httpapi/sse -run 'TestSSE_MaxTwoPerUser|TestSSE_LimitSharedAcrossInstances|TestSSE_SlotExpiresAfterCrash'`.
- AC6. Given `SSE_MAX_DURATION` (mặc định **120 s**) When stream mở đủ thời gian Then server gửi `event: reconnect` `data: {"reason":"max_duration"}` rồi đóng; client nối lại bằng `Last-Event-ID` không mất sự kiện (AC7). Stream cũng kết thúc khi token hết hạn (`event: reconnect` `{"reason":"token_expired"}`) — tối đa bằng min(120 s, `exp` còn lại); nối lại cần token mới.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_MaxDuration|TestSSE_TokenExpiryMidStream|TestConfig_SSEDefaults'` (`SSE_MAX_DURATION=1s` trong test; test cấu hình khẳng định mặc định 120 s); tay (2 phút): `curl -sk -N --max-time 130 -H "Authorization: Bearer $(tok STUDENT $U1 --ttl 10m)" $GW/api/v1/events | grep -c 'reason":"max_duration'` → `1`.
- AC7 (nhánh lỗi — nối lại không mất, không trùng). Given phát 10 sự kiện e1…e10, client đọc e1…e5 rồi ngắt, trong lúc ngắt phát e11…e13 When nối lại với `Last-Event-ID: <id của e5>` Then nhận đúng e6…e13 theo thứ tự, **không** nhận lại e5, rồi tiếp tục nhận sự kiện mới; mỗi sự kiện đúng 1 lần.
  Kiểm: `gotest ./internal/httpapi/sse -run TestSSE_LastEventID_NoLossNoDup`.
- AC8 (nhánh lỗi — đua khi nối). Given phát liên tục 1.000 sự kiện (≈ 1 kHz) trong khi client ngắt và nối lại 20 lần ở các thời điểm ngẫu nhiên Then hợp sự kiện client nhận = đúng 1.000 id duy nhất theo thứ tự, không sót, không lặp (subscribe **trước** khi đọc bù, loại trùng theo id).
  Kiểm: `gotest ./internal/httpapi/sse -race -run TestSSE_ReconnectRace`.
- AC9 (nhánh lỗi — bộ đệm). Given bộ đệm giữ `SSE_BUFFER_MAXLEN` (1.000) sự kiện / người, TTL 1 giờ When `Last-Event-ID` cũ hơn sự kiện còn lại, hoặc sai định dạng (không khớp `^\d+-\d+$`) Then server gửi `event: resync` `data: {"reason":"buffer_exceeded"}` đầu tiên (client phải tải lại dữ liệu), rồi chuyển sang sự kiện mới; không lỗi 4xx/5xx.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_ResyncWhenTooOld|TestSSE_ResyncOnMalformedLastEventID'`.
- AC10. Given hai bản gateway A và B dùng chung Redis When client nối A và sự kiện được phát từ B Then client nhận trong ≤ 1 s; hai kết nối của cùng người dùng ở hai bản đều nhận. Qua Caddy với `--scale gateway=2`: mở stream rồi gửi 5 `POST /api/v1/_test/events` (mỗi cái có thể trúng bản nào) → stream nhận đủ 5.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_CrossInstance|TestSSE_TwoConnectionsTwoInstances'`; tay: `tmode; (curl -sk -N -m 8 -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/events | grep -c '^event: test.ping' &); sleep 1; for i in 1 2 3 4 5; do curl -sk -o /dev/null -X POST -H "Authorization: Bearer $(tok STUDENT $U1)" -H 'Content-Type: application/json' -d '{"type":"test.ping","data":{"n":'$i'}}' $GW/api/v1/_test/events; done; wait` → `5`.
- AC11 (phân quyền). Given `U1` và `U2` cùng có stream When sự kiện được phát cho `U1` Then `U2` **không** nhận gì (kể cả khi gửi `?user_id=<U1>`; tham số đó bị bỏ qua); không có cách chọn stream người khác. Không token → **401** `UNAUTHENTICATED`; token hết hạn → **401** `TOKEN_EXPIRED` — trả JSON **trước** khi mở stream.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_IsolationBetweenUsers|TestSSE_UserIDQueryIgnored|TestSSE_Unauthenticated'`; `curl -sk -o /dev/null -w '%{http_code}\n' $GW/api/v1/events` → `401`.
- AC12. Given job `test.progress` 4 bước của `U1` When chạy Then `U1` nhận các sự kiện `event: job.progress` có `data: {"job_id","status","progress","result"?}` với `progress` 25, 50, 75, 100 (không giảm) và sự kiện cuối `status:"SUCCEEDED"`; `U2` không nhận; trạng thái trong DB khớp sự kiện cuối.
  Kiểm: `gotest ./internal/jobs ./internal/httpapi/sse -run 'TestJobProgress_SSE|TestJobProgress_OnlyOwner'`.
- AC13 (nhánh lỗi — tắt một gateway giữa stream). Given hai bản gateway, client đang nhận stream ở A When tắt A (SIGTERM) Then client nhận `event: shutdown` `{"reason":"server_shutdown"}` rồi kết nối đóng trong ≤ 1 s; client nối lại **bản còn lại** với `Last-Event-ID` và **cùng token** (không đăng nhập lại, không 401) → nhận đủ sự kiện phát trong lúc chuyển, không mất, không trùng. Phiên kế tiếp qua Caddy vẫn chạy.
  Kiểm: `gotest ./internal/httpapi/sse -run TestSSE_GatewayShutdownFailover`; tay: `tmode; curl -sk -N -H "Authorization: Bearer $T" $GW/api/v1/events > /tmp/s1.log & sleep 2; docker stop $($C ps -q gateway | head -1); sleep 3; curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $T" $GW/api/v1/jobs/00000000-0000-7000-8000-000000000009` → `404` (không `401`), `/tmp/s1.log` có `event: shutdown` nếu stream thuộc bản bị tắt (nếu không, lặp bước).
- AC14. Given Caddy đứng trước When mở stream qua `https://localhost` Then sự kiện `ready` đến ngay (không bị đệm), heartbeat đến từng nhịp, và **không** có `Content-Encoding` kể cả khi client gửi `Accept-Encoding: gzip`.
  Kiểm: `curl -sk -N -m 3 -D- -H 'Accept-Encoding: gzip' -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/events | grep -ciE '^content-encoding|^event: ready'` → `1` (chỉ dòng `event: ready`).
- AC15 (nhánh lỗi — Redis chết). Given stream đang mở When Redis mất Then server gửi `event: reconnect` `{"reason":"upstream_unavailable"}` và đóng (không treo); kết nối mới khi Redis còn chết → **503** `SERVICE_UNAVAILABLE`; Redis về → nối được lại.
  Kiểm: `gotest ./internal/httpapi/sse -run 'TestSSE_RedisDownMidStream|TestSSE_RedisDownOnConnect'`.

### Ngoài phạm vi của story này
- Sự kiện nghiệp vụ (thông báo `notifications` — P4; stream chat — P3; đọc bù từ bảng `notifications` thay cho bộ đệm Redis — P4 quyết). Nhận token qua cookie (P2).
- Đo SSE bằng k6 (k6 không có SSE sẵn; đo bằng test Go và `curl` ở AC1).

### Phụ thuộc
- US-PG-01 (redis, shutdown), US-PG-03 (định dạng lỗi, rate limit, jobs), US-PG-04 (JWT). Quyết định D22, D47 mục 4.

---

## US-PG-06: Lập trình viên (dev các phase sau) muốn `api/openapi.yaml` là nguồn sự thật và có contract test khoá hai chiều để API và tài liệu không bao giờ lệch nhau
Ưu tiên: Must · Ước lượng: S · Sprint: 2

Truy vết: PG L6 (3 checkbox).

### Tiêu chí nghiệm thu
- AC1. Given `backend-go/api/openapi.yaml` và `backend-go/api/openapi.test.yaml` When kiểm Then cả hai là OpenAPI hợp lệ (`redocly lint` 0 lỗi; `kin-openapi` nạp và `Validate` được). Phiên bản `openapi:` của file là phiên bản mà `kin-openapi` đã ghim nạp được (`QUESTIONS.md` Q5).
  Kiểm: `pnpm exec redocly lint backend-go/api/openapi.yaml backend-go/api/openapi.test.yaml; echo rc=$?` → `rc=0`. Test: `gotest ./internal/contract -run TestSpec_LoadsAndValidates`.
- AC2. Given `openapi.yaml` When liệt kê đường dẫn Then đúng 5: `/healthz`, `/api/v1/healthz`, `/api/v1/readyz`, `/api/v1/jobs/{id}`, `/api/v1/events`; **không** có `/_test/` (các route thử nằm ở `openapi.test.yaml`); mỗi thao tác có `operationId`.
  Kiểm: `cd backend-go && grep -cE '^  /' api/openapi.yaml` → `5`; `grep -c '_test' api/openapi.yaml` → `0`; `grep -cE '^  /api/v1/_test' api/openapi.test.yaml` ≥ `10`.
- AC3. Given gateway dựng thật bằng `httptest` (Postgres, Redis, MinIO container) When chạy contract test Then với mỗi thao tác trong spec, response thật khớp schema: **status** nằm trong spec, **trường bắt buộc** có mặt và đúng kiểu, **hình dạng lỗi** `Error`, **hình dạng phân trang** `{items, next_cursor}` (route thử), header bắt buộc (`ETag`, `Retry-After`, `Location`…). Chạy ở **hai chế độ**: có tag `testroutes` (kiểm cả `openapi.yaml` lẫn `openapi.test.yaml`) và **không tag** (chỉ `openapi.yaml`; mọi đường dẫn trong `openapi.test.yaml` phải trả 404 — khớp US-PG-03 AC18).
  Kiểm: `gotest ./internal/contract/... ; echo rc=$?` → `rc=0`; `(cd backend-go && go test -count=1 ./internal/contract/... ; echo rc=$?)` → `rc=0` (không tag).
- AC4 (nhánh lỗi — lệch hai chiều thì đỏ). Given route có trong code mà thiếu trong spec, hoặc có trong spec mà không có trong code, hoặc status thật không có trong spec Then test **đỏ** và nêu `METHOD /đường-dẫn` lệch. So khớp: `openapi.yaml` với các route **không** nằm dưới `/api/v1/_test/`; `openapi.test.yaml` với các route dưới `/api/v1/_test/` (chỉ ở chế độ có tag). QC kiểm bằng bản spec sửa tạm (không sửa repo): `OPENAPI_PATH` trỏ tới file sao chép đã xoá `/api/v1/readyz`, rồi thêm `/api/v1/khong-co`.
  Kiểm: `cp backend-go/api/openapi.yaml /tmp/m.yaml; (xoá khối /api/v1/readyz trong /tmp/m.yaml); OPENAPI_PATH=/tmp/m.yaml gotest ./internal/contract -run TestRouteSpecParity; echo rc=$?` → `rc≠0`, thông báo chứa `GET /api/v1/readyz`; thêm đường dẫn thừa → `rc≠0`, thông báo chứa `/api/v1/khong-co`; cùng phép thử **không tag** (`cd backend-go && OPENAPI_PATH=/tmp/m.yaml go test -count=1 ./internal/contract -run TestRouteSpecParity`) cũng `rc≠0`.
- AC5. Given spec khai báo các response lỗi Then `components.responses` có `Unauthorized`(401), `Forbidden`(403), `NotFound`(404), `Conflict`(409), `ValidationFailed`(422), `RateLimited`(429), `Unavailable`(503), `DeadlineExceeded`(504), mọi lỗi `$ref` schema `Error`; contract test **ghi nhận** cặp (thao tác, status) đã được gọi thật và đỏ nếu một status được khai báo mà không test nào sinh ra (ngoại trừ danh sách miễn trừ có lý do trong `internal/contract/exempt.go`).
  Kiểm: `gotest ./internal/contract -run 'TestSpec_ErrorResponsesDeclared|TestSpec_EveryDocumentedStatusExercised'`.
- AC6 (đối chứng âm). Given validator Then có test chứng minh nó **bắt được** lỗi: handler giả trả `{"items":null}` / thiếu `next_cursor` / `code` ngoài danh sách / status không khai báo → `ValidateResponse` báo lỗi.
  Kiểm: `gotest ./internal/contract -run TestValidator_RejectsBadResponses`.
- AC7 (phân quyền). Given spec Then `/api/v1/jobs/{id}` và `/api/v1/events` khai `security: [bearerAuth]` và có response 401; route chỉ vai trò (route thử `rbac/*`) khai thêm 403; `/healthz`, `/api/v1/healthz`, `/api/v1/readyz` khai `security: []`. Thật sự: gọi không token vào thao tác có `bearerAuth` → 401 đúng như spec.
  Kiểm: `gotest ./internal/contract -run 'TestSpec_SecurityDeclared|TestContract_UnauthenticatedMatchesSpec'`.
- AC8 (D52 — kin-openapi chỉ cho test). Given thư viện `getkin/kin-openapi` When kiểm phụ thuộc Then **không** nằm trong binary gateway/worker, chỉ trong `internal/contract` và các `_test.go`; lint chặn import ở nơi khác.
  Kiểm: `cd backend-go && go list -deps ./cmd/gateway ./cmd/worker | grep -c kin-openapi` → `0`; `go list -deps -test ./internal/contract | grep -c kin-openapi` ≥ `1`; `golangci-lint run` sạch (luật `depguard`).

### Ngoài phạm vi của story này
- Sinh mã từ spec (`oapi-codegen`) — PG chỉ dùng spec làm nguồn sự thật cho contract test. Spec cho endpoint nghiệp vụ (mỗi phase thêm).
- Phát lại bản ghi của hệ cũ / golden Java (D45).

### Phụ thuộc
- US-PG-03, 04, 05 (route cần mô tả); quyết định D45, D52.

---

## US-PG-07: Lập trình viên (dev, chủ dự án) muốn hạ tầng chạy đủ — Caddy, PgBouncer, migrate, worker, hai bản gateway, image nhỏ, CI chặn lệch, k6 smoke — để chứng minh gateway thật sự không trạng thái và mở rộng ngang không sửa mã
Ưu tiên: Must · Ước lượng: M · Sprint: 2

Truy vết: PG L7 (4 checkbox) + cổng nghiệm thu PG. Cập nhật `docker-compose.local.yml`, `backend-go/Dockerfile`, `.github/workflows/ci.yml`, `benchmarks/load/smoke.js`, `.env.example`.

### Tiêu chí nghiệm thu
- AC1. Given `docker-compose.local.yml` When liệt kê service Then đúng 10: `caddy`, `frontend`, `gateway`, `mailpit`, `migrate`, `minio`, `pgbouncer`, `postgres`, `redis`, `worker`; không service Python.
  Kiểm: `$C config --services | sort | paste -sd' '` → `caddy frontend gateway mailpit migrate minio pgbouncer postgres redis worker`; `$C config --services | grep -ciE 'python|docling'` → `0`.
- AC2. Given máy sạch When chạy `$C up -d --scale gateway=2 --wait` Then thoát mã 0; 2 container `gateway` healthy; `postgres`, `redis`, `minio`, `mailpit`, `pgbouncer`, `caddy`, `worker`, `frontend` healthy; `migrate` đã thoát mã 0 trước khi gateway/worker khởi động (`depends_on: condition: service_completed_successfully`).
  Kiểm: `$C up -d --scale gateway=2 --wait; echo rc=$?; $C ps -a --format '{{.Service}} {{.State}} {{.Health}} {{.ExitCode}}' | sort` → `rc=0`; hai dòng `gateway running healthy 0`; `migrate exited  0`; các dòng khác `running healthy 0`.
- AC3. Given Caddy When gọi qua `https://localhost` Then `curl -fsSk $GW/api/v1/healthz` → **200** `{"status":"ok"}`; `/` trả HTML của frontend; `http://localhost` chuyển hướng sang https; có `Strict-Transport-Security` và `X-Content-Type-Options: nosniff`; thân HTML lớn được nén (`Content-Encoding: zstd` hoặc `gzip`), JSON nhỏ và `text/event-stream` thì không.
  Kiểm: `curl -fsSk $GW/api/v1/healthz; curl -sk $GW/ | grep -c '<html'; curl -sI http://localhost/ | grep -ic '^location: https://'; curl -skI $GW/api/v1/healthz | grep -ciE '^(strict-transport-security|x-content-type-options: nosniff)'; curl -sk -D- -o /dev/null -H 'Accept-Encoding: gzip, zstd' $GW/ | grep -ci '^content-encoding'` → `{"status":"ok"}`, `1`, `1`, `2`, `1`.
- AC4. Given TLS nội bộ Then chứng chỉ do `Caddy Local Authority` cấp cho `localhost`.
  Kiểm: `echo | openssl s_client -connect localhost:443 -servername localhost 2>/dev/null | openssl x509 -noout -issuer | grep -c 'Caddy Local Authority'` → `1`.
- AC5 (hai bản gateway). Given `--scale gateway=2` When gọi 40 lần `GET /api/v1/healthz` qua Caddy Then **cả hai** bản trả 200 (200 ở mọi lần), thấy đúng 2 giá trị `X-Instance-Id` khác nhau; cùng **một** token (ký một lần) dùng được ở cả hai bản — `GET /api/v1/jobs/<uuid lạ>` trả **404** `NOT_FOUND` ở cả hai (không bao giờ 401) — chứng tỏ không trạng thái phiên.
  Kiểm: `for i in $(seq 40); do curl -sk -D- -o /dev/null $GW/api/v1/healthz | tr -d '\r' | awk -F': ' 'tolower($1)=="x-instance-id"{print $2} /^HTTP/{print $2}'; done | sort | uniq -c` → `40 200` và 2 dòng instance mỗi dòng > 0; `T=$(tok STUDENT $U1); for i in $(seq 20); do curl -sk -o /dev/null -w '%{http_code} ' -H "Authorization: Bearer $T" $GW/api/v1/jobs/00000000-0000-7000-8000-000000000009; done` → toàn `404`.
- AC6 (nhánh lỗi — tắt một bản gateway). Given 200 request `GET` đều đặn (10/s) qua Caddy When tắt một container gateway giữa chừng Then tổng lỗi 5xx ≤ 2 % trong cửa sổ tắt và **0** lỗi sau 6 s (Caddy loại upstream hỏng bằng health check chủ động + thử lại sang bản khác); bản còn lại phục vụ bình thường, không ai mất đăng nhập.
  Kiểm: `( for i in $(seq 200); do curl -sk -o /dev/null -w '%{http_code}\n' $GW/api/v1/healthz; sleep 0.1; done > /tmp/codes ) & sleep 5; docker stop $($C ps -q gateway | head -1); wait; sort /tmp/codes | uniq -c; tail -60 /tmp/codes | grep -vc '^200$'` → số dòng không-200 ≤ 4; phần đuôi `0`.
- AC7. Given PgBouncer Then chạy ở chế độ `transaction`, lắng nghe `6432` trong mạng compose, không mở cổng ra máy chủ; mở được từ `postgres` bằng tài khoản thống kê; `MAX_CLIENT_CONN`, `DEFAULT_POOL_SIZE` đúng `SRS.md` 8.3; kết nối tới Postgres dùng `scram-sha-256`.
  Kiểm: `$C exec -T postgres psql "$PGB" -Atc 'SHOW CONFIG' | grep -E '^(pool_mode|max_client_conn|default_pool_size)\|'` → `pool_mode|transaction`, `max_client_conn|200`, `default_pool_size|20`.
- AC8. Given gateway và worker When chạy Then truy vấn runtime đi **qua PgBouncer** (`PGBOUNCER_URL`), còn `migrate` đi thẳng Postgres; log `config loaded` có `"db_via":"pgbouncer"`; `SHOW CLIENTS` trên PgBouncer có ≥ 3 client (2 gateway + worker) và không có client của `migrate`.
  Kiểm: `$C logs --no-log-prefix gateway | jq -r 'select(.msg=="config loaded").db_via' | sort -u` → `pgbouncer`; `$C exec -T postgres psql "$PGB" -Atc 'SHOW CLIENTS' | grep -c edupilot` ≥ `3`.
- AC9 (nhánh lỗi — pgx hợp transaction mode). Given PgBouncer thật ở transaction mode trước Postgres When 50 goroutine × 20 vòng chạy hỗn hợp truy vấn sqlc, transaction nhiều câu, truy vấn vector và `pg_sleep` ngắn Then không lỗi `prepared statement "…" already exists` / `does not exist`, không lỗi giao thức; kết nối của pool không dùng prepared statement ngầm (dev chọn `QueryExecModeExec` hoặc `SimpleProtocol` và ghi lý do vào handoff; `CacheStatement`/`CacheDescribe` bị cấm).
  Kiểm: `gotest ./internal/platform/db -run 'TestPgBouncer_TransactionMode|TestPgBouncer_NoPreparedStatements|TestVector_ThroughPgBouncer'`.
- AC10. Given Dockerfile nhiều tầng Then build ra 2 image từ cùng `backend-go/Dockerfile` (`--target gateway`, `--target worker`), mỗi image **< 40 MB** (byte `.Size` < 40.000.000), chạy bằng người dùng không phải root, không có shell; `migrate` dùng lại image gateway (`entrypoint ["/gateway","migrate"]`).
  Kiểm: `for t in gateway worker; do docker build -q --target $t -t edupilot-$t backend-go >/dev/null; docker image inspect -f '{{.Config.User}} {{.Size}}' edupilot-$t; done` → `nonroot|65532 <số>` với số < 40000000 cho cả hai; `docker run --rm --entrypoint sh edupilot-gateway -c true; echo rc=$?` → lỗi (không có `sh`).
- AC11 (không trạng thái). Given luật 10 `AGENTS.md` Then (a) gateway và worker chạy với **hệ thống file chỉ-đọc** (`read_only: true`) mà mọi AC khác vẫn đạt; (b) không mã sản xuất nào ghi đĩa cục bộ; (c) không biến toàn cục giữ trạng thái (lint `gochecknoglobals`, trừ `Err*` và hằng).
  Kiểm: `docker inspect -f '{{.HostConfig.ReadonlyRootfs}}' $($C ps -q gateway worker)` → `true` mỗi dòng; `cd backend-go && grep -rnE 'os\.(Create|CreateTemp|WriteFile|OpenFile|Mkdir|MkdirAll|MkdirTemp)|ioutil\.(WriteFile|TempFile)' internal cmd --include=*.go | grep -v '_test.go'` → không in gì; `golangci-lint run` sạch.
- AC12. Given `.github/workflows/ci.yml` Then job **Go** chạy `go vet` và `golangci-lint` **hai lần** (không tag và `-tags testroutes` / `--build-tags testroutes`, để cả hai cấu hình biên dịch đều sạch), `go test -race -tags testroutes ./...` (gồm `TestDefaultBinary_NoTestRoutes` tự dựng bản không tag), `sqlc diff` (cài bằng `sqlc-dev/setup-sqlc`), job **Frontend** giữ nguyên (lint, build, phản mẫu UI); CI xanh ở HEAD nhánh; `ci.yml` vẫn không đọc `legacy/`, không dùng secret.
  Kiểm: `gh run list --workflow ci.yml --branch sprint/2-pg --limit 1 --json databaseId,headSha,conclusion` → `headSha` = `git rev-parse origin/sprint/2-pg`, `conclusion=success`; `gh run view <ID> --json jobs --jq '.jobs[].steps[].name' | grep -ciE 'sqlc|race|vet|golangci'` ≥ `6`; `grep -c 'testroutes' .github/workflows/ci.yml` ≥ `3`; `grep -nE 'legacy|secrets\.' .github/workflows/ci.yml` → không in gì.
- AC13 (nhánh lỗi — CI chặn lệch sqlc). Given nhánh tạm `ci/sqlc-drift` có một thay đổi `internal/store/queries/*.sql` không chạy `sqlc generate` When CI chạy Then run `failure`, job Go `failure` ở bước `sqlc diff`, job Frontend `success`. Dev **chỉ xoá** `ci/sqlc-drift` sau khi QC chấm; handoff ghi `<ID>`, `headSha`, tên nhánh (cùng cơ chế `FEAT-ci` AC3).
  Kiểm: `gh run view <ID> --json conclusion,headSha,headBranch,jobs --jq '.conclusion, .headSha, .headBranch, (.jobs[] | "\(.name) \(.conclusion)")'` → `failure`, khớp handoff, `ci/sqlc-drift`, `Go failure`, `Frontend success`; sau khi xoá: `git ls-remote --heads origin ci/sqlc-drift` → không in gì.
- AC14. Given `benchmarks/load/smoke.js` When chạy `k6 run benchmarks/load/smoke.js` (gateway ×2 qua Caddy) Then thoát mã 0 với ngưỡng đạt: p95 của `healthz`, `readyz`, và `jobs-404` (có token + truy vấn DB qua PgBouncer) mỗi cái **≤ 300 ms**; tỉ lệ 5xx **< 0,5 %**; mọi `check` > 99 %. Có cờ `-e TEST_ROUTES=1` chạy thêm kịch bản ghi (`POST /_test/items` có `Idempotency-Key`) với p95 **≤ 500 ms** (chỉ chạy được trên stack dựng bằng `tmode`, tức image `gateway-test`; trên stack mặc định route này 404 nên k6 dừng với thông báo rõ). Số đo không đạt trên máy dev (colima 4 CPU) thì ghi số + lý do vào báo cáo, **không nới SLO** (plan sprint 2).
  Kiểm: `k6 run -e BASE=https://localhost -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js; echo rc=$?` → `rc=0`, các dòng `✓` ở mục thresholds; ngưỡng nằm trong `options.thresholds` của file (`grep -c 'p(95)<300' benchmarks/load/smoke.js` ≥ `3`).
- AC15. Given số liệu nền Then `benchmarks/reports/pg-baseline.md` có đủ 4 số của bản Go: RAM nghỉ gateway và worker (MiB, đo bằng `docker stats --no-stream` sau 60 s rảnh), thời gian khởi động (ms, từ khi tiến trình chạy tới `/readyz` 200, lấy từ trường `startup_ms` của log), kích thước hai image (MB), p95 `/api/v1/healthz` (ms từ k6).
  Kiểm: `grep -ciE 'RAM|khởi động|image|p95' benchmarks/reports/pg-baseline.md` ≥ `4`; `$C logs --no-log-prefix gateway | jq -r 'select(.msg=="gateway ready").startup_ms' | head -1` là một số.
- AC16 (nhánh lỗi — thiếu env trong compose). Given xoá `JWT_SECRET_KEY` khỏi `.env.local` When `$C up -d gateway` Then container gateway **thoát mã 1** và log nêu `JWT_SECRET_KEY` (restart tối đa 3 lần rồi dừng, không lặp vô hạn); service `migrate` không bị ảnh hưởng.
  Kiểm: `sed -i.bak '/^JWT_SECRET_KEY=/d' .env.local; $C up -d --force-recreate gateway; sleep 8; $C logs --no-log-prefix gateway | jq -r 'select(.missing).missing[]' | sort -u; mv .env.local.bak .env.local` → có `JWT_SECRET_KEY`.
- AC17 (phân quyền / bề mặt mạng). Given cổng mạng Then chỉ `caddy` công bố 80/443 ra máy chủ trong số các service ứng dụng; `gateway`, `worker`, `pgbouncer`, `migrate` **không** có cổng công bố (nên `--scale gateway=2` không tranh cổng); không bí mật nào nằm trong `docker-compose.local.yml` (mọi giá trị nhạy cảm là `${BIEN}` lấy từ `.env.local`); `.env.example` chỉ chứa giá trị dev giả.
  Kiểm: `$C ps --format '{{.Service}} {{.Ports}}' | grep -E '^(gateway|worker|pgbouncer|migrate) ' | grep -c -- '->'` → `0`; `grep -nE '(PASSWORD|SECRET|KEY)[A-Z_]*: *[^$ ]' docker-compose.local.yml` → không in gì; `curl -sk -o /dev/null -w '%{http_code}\n' http://localhost:8080/healthz` → lỗi kết nối (`000`).
- AC18. Given Redis và Postgres cho dữ liệu hàng đợi Then Redis chạy `appendonly yes`, `appendfsync everysec`, `maxmemory-policy noeviction` (Streams và khoá idempotency không bị đẩy ra khi đầy); volume tên cố định cho postgres, redis, minio, caddy.
  Kiểm: `$RDS config get appendonly | tail -1; $RDS config get maxmemory-policy | tail -1` → `yes`, `noeviction`.
- AC19 (cổng PG chạy trọn). Given nhánh `sprint/2-pg` When chạy toàn bộ cổng nghiệm thu PG (`PG.md` + các lệnh của US này) Then tất cả đạt, kết quả dán vào handoff: `go vet` và `golangci-lint run` (cả hai cấu hình tag), `go test -race -tags testroutes ./...`, `sqlc diff`, `go test ./internal/contract/...` (có và không tag), `go test -race -tags testroutes ./internal/httpapi/... ./internal/httpapi/sse/... ./internal/auth/...`, `pnpm -C frontend build`, `up -d --scale gateway=2` (bản mặc định), `curl -fsSk https://localhost/api/v1/healthz`, `k6 run`.
  Kiểm: `cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes && go test -race -tags testroutes ./... && sqlc diff && go test -tags testroutes ./internal/contract/... && go test ./internal/contract/... && go test -race -tags testroutes ./internal/httpapi/... ./internal/auth/...; echo rc=$?` → `rc=0`; rồi `pnpm -C frontend build && dmode && curl -fsSk $GW/api/v1/healthz && k6 run -e BASE=https://localhost -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js`.
- AC20 (route thử chỉ ở target thử). Given `backend-go/Dockerfile` và `docker-compose.test.yml` Then Dockerfile có thêm hai target `gateway-test`, `worker-test` (cùng tầng chạy, binary dựng bằng `go build -tags testroutes`); các target mặc định `gateway`, `worker` **không** dùng tag; `docker-compose.test.yml` là **override** (không thêm service mới): chỉ đổi `build.target` và `image` của `gateway` và `worker` sang `edupilot-gateway-test` / `edupilot-worker-test` (khác tên image mặc định để hai bản không ghi đè nhau) và `APP_ENV=test`; `docker-compose.local.yml` **không** tham chiếu target `-test`; CI không đẩy image `-test` đi đâu.
  Kiểm: `grep -cE '^FROM .* AS (gateway-test|worker-test)$' backend-go/Dockerfile` → `2`; `grep -c 'testroutes' backend-go/Dockerfile` → `2` (đúng hai dòng `go build`, của hai target thử); `grep -c 'gateway-test\|worker-test' docker-compose.local.yml` → `0`; `$CT config --services | sort | paste -sd' '` → cùng danh sách 10 service như AC1; `$CT config | grep -E 'image: edupilot-(gateway|worker)-test' | wc -l` → `2`; `$CT config | grep -cE 'target: (gateway|worker)-test'` → `2`; `tmode; curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/_test/whoami` → `200`; `dmode; curl -sk -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $(tok STUDENT $U1)" $GW/api/v1/_test/whoami` → `404`; `docker image inspect -f '{{.Size}}' edupilot-gateway-test` < `40000000`.

### Ngoài phạm vi của story này
- Triển khai thật (VPS), tên miền và chứng chỉ công khai, sao lưu, Kubernetes; Prometheus / Grafana; test hỗn loạn và SLO T1 đầy đủ (P10, khung k6 dựng từ đây).
- `docling-serve`, `mock-graph` (phase sau); kịch bản k6 nghiệp vụ.

### Phụ thuộc
- US-PG-01…06; `FEAT-scaffold` (compose, `.env.example`), `FEAT-ci` (workflow); quyết định D22, D48, D52; `SYSTEM_DESIGN.md` mục 5 (SLO).
