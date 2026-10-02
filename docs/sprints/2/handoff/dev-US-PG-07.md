# DEV handoff — US-PG-07 — Hạ tầng: Caddy, PgBouncer, compose, Dockerfile, CI, k6
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-07'`)

## Đã làm (theo thứ tự lát dọc)
compose 10 service → Caddy (TLS nội bộ, `dynamic a gateway 8080`, `flush_interval -1`) → PgBouncer (transaction, `max_prepared_statements=0`) → `migrate` (trực tiếp Postgres) → Dockerfile 4 target (gateway, worker, gateway-test, worker-test) → `docker-compose.test.yml` → CI → k6 smoke → test PgBouncer thật.
- **Góp ý #3(c) (PM ACCEPTED):** `caddy:2.11` (đã chạy: v2.11.4). (b) giữ `dynamic a` — health check **bị động**: `max_fails 1`, `fail_duration`, `lb_try_*` và **`unhealthy_status 503`** (gateway đang tắt trả 503 `NOT_READY` ⇒ Caddy ngừng gửi tới bản đó ngay). Nợ P10/PR: upstream tĩnh + `health_uri /api/v1/healthz`.
- **Góp ý #3(d):** `backend-go/README.md` hướng dẫn `~/.testcontainers.properties` (`docker.host`, `ryuk.disabled=true`) và `make test-clean` sau khi kéo image.
- PgBouncer AC9: dùng **`QueryExecModeExec`** (lý do: mọi lệnh gửi kèm tham số trong một message, không Parse/Describe tách ⇒ không dựa vào prepared statement phía server; `SimpleProtocol` kém an toàn kiểu dữ liệu với jsonb/vector). `testutil.PgBouncerHostPort/…` dựng PgBouncer THẬT bằng testcontainers trước container Postgres dùng chung.

## File đổi
`docker-compose.local.yml`, `docker-compose.test.yml`, `deploy/caddy/Caddyfile`, `deploy/pgbouncer/pgbouncer.ini`, `backend-go/Dockerfile`, `.github/workflows/ci.yml`, `benchmarks/load/smoke.js`, `benchmarks/reports/pg-baseline.md`, `.env.example`, `scripts/dev.mjs`, `backend-go/internal/testutil/pgbouncer.go`, `backend-go/internal/platform/db/pgbouncer_test.go`.

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg07.sh`.

## Test đã chạy và kết quả
Gói chính: `./internal/platform/db` — `TestPgBouncer_TransactionMode` (50×20, SHOW CONFIG pool_mode=transaction), `TestPgBouncer_NoPreparedStatements`, `TestVector_ThroughPgBouncer`: 3 PASS, không SKIP. Các AC hạ tầng đo ở GATE — xem `dev-GATE-PG.md` (số liệu thật).

Kết quả thật (chạy ở HEAD):
- `go vet ./...` và `go vet -tags testroutes ./...`: không in gì.
- `golangci-lint run` và `golangci-lint run --build-tags testroutes`: `0 issues.` (cả hai).
- `go test -race -count=1 -tags testroutes ./...` ở HEAD `2d2834b`: 15 gói `ok`, 0 FAIL, 0 SKIP (cmd/gateway 5.1 s, cmd/worker 7.4 s, db 4.6 s, auth 23.6 s, contract 12.7 s, httpapi 14.9 s, httpapi/sse 46.9 s, jobs 9.0 s, blob 8.5 s, config 1.4 s, platform/db 21.1 s, log 1.4 s, otel 1.5 s, outbox 9.5 s, redis 3.8 s, store 11.0 s).
- `sqlc diff` rc=0, không in gì.

## AC tự đánh giá
- AC1: ✓ xem GATE
- AC2: ✓ xem GATE
- AC3: ✓ xem GATE
- AC4: ✓ xem GATE
- AC5: ✓ xem GATE
- AC6: ✓ xem GATE
- AC7: ✓ xem GATE
- AC8: ✓ xem GATE
- AC9: ✓ xem GATE
- AC10: ✓ xem GATE
- AC11: ✓ xem GATE
- AC12: ✓ xem GATE
- AC13: ✓ xem GATE
- AC14: ✓ xem GATE
- AC15: ✓ xem GATE
- AC16: ✓ xem GATE
- AC17: ✓ xem GATE
- AC18: ✓ xem GATE
- AC19: ✓ xem GATE
- AC20: ✓ xem GATE

## Sửa ở cổng PG (số đo thật ở `dev-GATE-PG.md`)
- PgBouncer: `server_login_retry=1`, `dns_nxdomain_ttl=1`, `dns_max_ttl=5` (mặc định 15 s làm `readyz` hồi phục 11–19 s sau khi Postgres khởi động lại, vượt AC14 ≤ 10 s của US-PG-01; nay 1 s, 3/3 lần).
- Caddyfile: snippet `gateway_upstream`; `readyz` đi riêng không bị loại thụ động (503 NOT_READY hợp lệ khi phụ thuộc chết không được loại gateway), các đường còn lại có `fail_duration 10s`, `max_fails 1`, `unhealthy_status 503`. AC6: 199 × 200 + 1 × 503 (≤ 4), 0 lỗi ở 60 request cuối, 3/3 lần.
- `smoke.js` `jobs404`: 429 ở biên 300/phút/IP được chấp nhận trong check.
- Compose: `JWT_SECRET_KEY: ${JWT_SECRET_KEY:-}` để compose không in cảnh báo; gateway tự thoát 1 khi rỗng (AC16).

## Nợ / chưa làm / cần hỏi
- Caddy health check bị động (spec v1.3). Ảnh PgBouncer `edoburu/pgbouncer:v1.26.0-p0`; `pgb_stats` cố định trong ini. CI AC12/AC13: xem GATE.
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.
