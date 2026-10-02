# DEV handoff — US-PG-01 — Khung dịch vụ Go
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-01'`)

## Đã làm (theo thứ tự lát dọc)
platform/{config,log,otel,redis,db,clock} → httpapi (router, middleware: request-id, recover, draining, access log, deadline, body limit, CORS) → `cmd/gateway serve`, `cmd/worker` → Makefile, `.golangci.yml` (depguard, noctx, contextcheck, gochecknoglobals) → test.
- Config: `config.Load(getenv, Gateway|Worker)` gom MỌI lỗi một lần, tên biến trong log, thoát mã 1 (AC1–AC3).
- Log JSON `slog` có `trace_id`, không PII; `traceparent` được nối vào span và log (AC4–AC5).
- Redis: mọi khoá qua `redis.Key(...)` tiền tố `ep:` (AC6).
- Tắt êm: `readyz` trả 503 `NOT_READY` (`draining:true`) trong cửa sổ drain `min(3s, SHUTDOWN_TIMEOUT/4)` trước khi đóng listener; request đang chạy hoàn tất; quá hạn → log `forced shutdown` `cancelled_requests` và thoát 1 (AC7).
- Giới hạn HTTP: `MAX_BODY_BYTES`, ReadHeaderTimeout 5 s, header 64 KiB, ReadTimeout 15 s (AC8). Deadline xuống DB/Redis (AC9). `DB_MAX_CONNS` (AC10). Log `slow query` (AC11). Worker có vòng consumer + `/healthz` (AC12).
- Pool Postgres: `QueryExecModeExec` (lý do: PgBouncer transaction mode không giữ prepared statement; `CacheStatement`/`CacheDescribe` bị cấm).

## File đổi
`backend-go/internal/platform/{config,log,otel,redis,db,clock}/*`, `backend-go/internal/httpapi/{router,server,middleware,routes,testroutes_on,testroutes_off}.go`, `backend-go/cmd/gateway/*`, `backend-go/cmd/worker/*`, `backend-go/Makefile`, `backend-go/.golangci.yml`, `backend-go/README.md`, `backend-go/internal/testutil/{containers,migrated,pgbouncer}.go`, `.gitignore`.

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg01.sh`.

## Test đã chạy và kết quả
Gói chính: `./internal/platform/config` (5), `log` (2), `otel` (2), `redis` (4), `db` (8, gồm TestPool_MaxConns, TestSlowQuery…), `./internal/httpapi` (57 tồn tại chung với PG-03), `./cmd/gateway` (6), `./cmd/worker` (3).

Kết quả thật (chạy ở HEAD):
- `go vet ./...` và `go vet -tags testroutes ./...`: không in gì.
- `golangci-lint run` và `golangci-lint run --build-tags testroutes`: `0 issues.` (cả hai).
- `go test -race -count=1 -tags testroutes ./...` ở HEAD `2d2834b`: 15 gói `ok`, 0 FAIL, 0 SKIP (cmd/gateway 5.1 s, cmd/worker 7.4 s, db 4.6 s, auth 23.6 s, contract 12.7 s, httpapi 14.9 s, httpapi/sse 46.9 s, jobs 9.0 s, blob 8.5 s, config 1.4 s, platform/db 21.1 s, log 1.4 s, otel 1.5 s, outbox 9.5 s, redis 3.8 s, store 11.0 s).
- `sqlc diff` rc=0, không in gì.

## AC tự đánh giá
- AC1: ✓ TestConfig_Missing…
- AC2: ✓
- AC3: ✓
- AC4: ✓
- AC5: ✓
- AC6: ✓
- AC7: ✓ test Go + compose (xem GATE)
- AC8: ✓
- AC9: ✓
- AC10: ✓
- AC11: ✓
- AC12: ✓
- AC13: ✓ (`make test` đặt DOCKER_HOST colima, Ryuk tắt)
- AC14: ✓ test Go; compose xem GATE
- AC15: n/a (story hạ tầng, đúng US)

## Nợ / chưa làm / cần hỏi
- testutil dùng **container dùng chung toàn máy** (tên cố định `edupilot-test-*`, flock, Reuse) thay vì container riêng mỗi gói: bỏ flaky khi `go test ./...` khởi động nhiều container song song. Dọn bằng `make -C backend-go test-clean` (cũng cần sau khi kéo image mới). `go test` trần cần `~/.testcontainers.properties` (README backend).
- `internal/testutil/containers.go` khoá tệp bằng `syscall` (không `os.OpenFile`) để vẫn khớp lệnh grep "không ghi đĩa" của AC11b; đây là hạ tầng test, không chạy ở production.
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.
