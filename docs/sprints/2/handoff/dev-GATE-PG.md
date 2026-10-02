# DEV handoff — Cổng PG (sprint 2)
Nhánh: `sprint/2-pg`. Commit cuối của cổng: xem `git log -1` (ghi ở cuối file). Đo ngày 2026-10-02, colima 4 CPU / 8 GiB, Apple Silicon.

## Lệnh đã chạy và kết quả thật

| Lệnh | Kết quả |
| --- | --- |
| `cd backend-go && go vet ./... && go vet -tags testroutes ./...` | không in gì |
| `golangci-lint run` / `golangci-lint run --build-tags testroutes` | `0 issues.` / `0 issues.` |
| `sqlc diff` | rc=0, không in gì |
| `go test -race -count=1 -tags testroutes ./...` (HEAD trước commit sửa timeout, `-json`) | 356 test PASS, 15 gói có test, 0 SKIP; 1 FAIL (`TestDeadline_Redis`) → **bug thật đã sửa** (timeout middleware, xem dưới), rồi `TestDeadline*` ×60 dưới tải CPU: PASS |
| `go test -race -count=1 ./internal/contract/...` (không tag) và `-tags testroutes` | PASS cả hai |
| `make lint-depguard-negative` | rc=0 (depguard chặn kin-openapi) |
| `go list -deps ./cmd/gateway ./cmd/worker \| grep -c kin-openapi` | `0` |
| `pnpm install --frozen-lockfile && pnpm -C frontend build` | build OK (rc=0) |
| `pnpm exec redocly lint backend-go/api/openapi.yaml` | rc=0 (chỉ cảnh báo) |
| `docker compose … up -d --build --scale gateway=2 --wait` | rc=0; 11 container chạy (10 service, gateway ×2), `migrate` exited 0 |

## Hạ tầng (US-PG-07 và liên quan) — số đo thật
- AC3 `curl -fsSk https://localhost/api/v1/healthz` → `{"status":"ok"}`; `/` trả HTML của frontend; `http://localhost/` chuyển hướng HTTPS; HSTS + `nosniff` có mặt; `Content-Encoding` có ở `/`.
- AC4 chứng chỉ do `Caddy Local Authority` cấp.
- AC5 40 lần `healthz` qua Caddy: 19 lần bản A, 21 lần bản B (hai `X-Instance-Id` khác nhau, chia gần đều).
- AC6 tắt một gateway giữa 200 GET (10/s): `199 × 200, 1 × 503`, 60 request cuối: 0 lỗi — 3 lần chạy liên tiếp cùng kết quả. Trước khi thêm `unhealthy_status 503` là 9 × 503.
- AC7 `SHOW CONFIG`: `pool_mode|transaction`, `max_client_conn|200`, `default_pool_size|20`.
- AC8 log `config loaded` → `db_via:"pgbouncer"`; `SHOW CLIENTS` có 3 client `edupilot` (2 gateway + worker), không có `migrate`.
- AC10 image `edupilot-gateway` 9 424 483 B, `edupilot-worker` 8 917 258 B (< 40 000 000), user `nonroot`, `docker run --entrypoint sh` thất bại (không có shell).
- AC11 `ReadonlyRootfs=true` cho cả hai gateway và worker; `grep os.(Create|WriteFile|OpenFile|Mkdir…)` ngoài `_test.go` → 0 dòng.
- AC16 xoá `JWT_SECRET_KEY` khỏi `.env.local` → gateway thoát mã 1, log JSON `missing:["JWT_SECRET_KEY"]`, restart ≤ 3 lần (`RestartCount=3`); `migrate` vẫn `exited 0`; `.env.local` đã khôi phục.
- AC17 chỉ `caddy` công bố 80/443 trong các service ứng dụng (`gateway`/`worker`/`pgbouncer` không có cổng publish; cổng 8080 từ máy chủ → `000`); không bí mật viết cứng trong `docker-compose.local.yml` (grep 0).
- AC18 Redis `appendonly yes`, `maxmemory-policy noeviction`.
- Chế độ thử: stack mặc định `_test/whoami` → 404; stack `docker-compose.test.yml` → 200; chuyển qua lại test ↔ default bằng `up -d --build --force-recreate` đúng cả hai chiều.
- US-PG-01 AC7 (tắt êm, qua Caddy): `slow=200`, mọi gateway `exit 0`. AC14: tắt Redis → `readyz` 503 `details.redis:"down"`, `healthz` 200, bật lại → 200; tắt Postgres → `details.db:"down"`; **sau khi bật lại hồi phục trong 1 s (3/3 lần)**.
- US-PG-02 AC7: `down -v` rồi `up --scale gateway=2 --wait` (23 s): `migrate exited 0`, `goose_db_version` 2 dòng đã áp dụng, 6 bảng (`audit_log goose_db_version idempotency_keys jobs outbox users`).
- k6 smoke (stack mặc định, `-e TOKEN=…`): rc=0, p95 healthz 5,75 ms, readyz 6,92 ms, jobs404 8,57 ms, `http_req_failed` 0,00 %, `checks` 100 %.
- k6 biến thể ghi (stack test, `-e TEST_ROUTES=1`): rc=0, p95 write 52,09 ms, 1 351/1 351 check đạt. Trên stack mặc định `-e TEST_ROUTES=1` → thoát ≠ 0 (`TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml`).
- Số liệu nền: `benchmarks/reports/pg-baseline.md` (RAM nghỉ gateway 3,98 / 3,96 MiB, worker 5,23 MiB, khởi động gateway 9 và 69 ms).

## Lỗi thật tìm ra ở cổng và đã sửa (commit `dd7462b`, `aa44994`, `2d2834b`)
1. **Hồi phục sau khi Postgres khởi động lại chậm 11–19 s (AC14 ≤ 10 s)**: PgBouncer giữ `server_login_retry=15` và nhớ NXDOMAIN của Docker DNS 15 s. Đặt `server_login_retry=1`, `dns_nxdomain_ttl=1`, `dns_max_ttl=5` → 1 s (3/3).
2. **Caddy `unhealthy_status 503`** (cho AC6) làm `readyz` 503 hợp lệ (Redis/Postgres đang tắt) cũng bị coi là lỗi thụ động → loại cả hai bản thêm `fail_duration` 10 s. Tách `handle /api/v1/readyz` không đánh dấu thụ động (snippet `gateway_upstream` dùng chung).
3. **`timeoutMiddleware` trả 200 rỗng** khi client Redis/DB báo lỗi i/o-timeout theo deadline *trước* khi `ctx.Err()` khác nil (tái hiện dưới tải CPU: `status=200, Content-Length:0`); nay coi mốc deadline đã qua là hết hạn → 504 `DEADLINE_EXCEEDED`.
4. `TestVectorConventions` đỏ ngẫu nhiên ở CI (so id tuyệt đối khi `halfvec` làm tròn khác nhau theo CPU) → so khoảng cosine (≈ 0) và có mặt trong top 5; ép `hnsw.ef_search=1000`, `enable_seqscan=off`.
5. `TestRateLimit_ForwardedFor` đỏ khi máy quá tải (rate limiter fail-open khi Redis chậm > 50 ms) → thêm lượt thử.
6. `smoke.js` `jobs404` ở đúng ngưỡng 300/phút/IP → 1/301 nhận 429; check chấp nhận 404 hoặc 429.
7. Compose: `${JWT_SECRET_KEY:-}` (không in cảnh báo biến thiếu; gateway tự từ chối chuỗi rỗng, AC16 vẫn thoát 1).

## Chạy thử script QC (`pg07.sh`, `gate-pg.sh`) — kết quả và lỗi nằm ở script
- `pg07.sh` (66 TC): PASS=56, MANUAL=2, FAIL=11 trước khi sửa; các FAIL còn lại do script / môi trường đã ghi ở `docs/sprints/2/proposals.md` #5–#10 (HTTP/2 làm hỏng `awk` đọc mã trạng thái; `rl_reset` bị `exec -T` nuốt stdin; `up --force-recreate gateway` tạo lại cả `migrate`; cổng MinIO in dạng `9000-9001->`; so `.env.local` với `.env.example`; `gt ./...`; `paste -sd' '` trên bash macOS; `docker ps --filter name=` với id; đếm gói không có test như SKIP). Hai TC `k6` ✗ nay đạt vì sửa `smoke.js`.
- TC-PG07-43/44 (CI nhánh `ci/sqlc-drift`): xem mục CI.
- TC-GATE-13 (k6 `TEST_ROUTES=1` ở bản mặc định phải dừng): trong lần chạy script, k6 chạy được kịch bản `write` nghĩa là stack đang ở chế độ test lúc đó; chạy tay cùng lệnh trên stack mặc định thoát 107 với thông báo đúng. Không tái hiện được nguyên nhân chuyển chế độ trong script; QC xin chạy lại sau khi sửa các lỗi script ở #10.

## CI (AC12/AC13)
CI_PLACEHOLDER

## Nợ chuyển tiếp
- Caddy chỉ có health check bị động (spec v1.3); nợ P10/PR: upstream tĩnh + `health_uri /api/v1/healthz`.
- Nhánh `ci/sqlc-drift` (cố ý đỏ) **chưa xoá**: chỉ xoá sau khi QC chấm TC-PG07-43/44.
- Bộ container test dùng chung toàn máy (`edupilot-test-*`); dọn bằng `make -C backend-go test-clean`.
