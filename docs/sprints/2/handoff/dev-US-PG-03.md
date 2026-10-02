# DEV handoff — US-PG-03 — Chuẩn HTTP: lỗi, CORS, rate limit, phân trang, idempotency, version, job
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-03'`)

## Đã làm (theo thứ tự lát dọc)
apierr → httpx (JSON, cursor, ETag, version) → middleware (CORS, rate limit Redis fail-open, Idempotency-Key) → `internal/jobs` (enqueue cùng transaction với outbox) → route thử `internal/testroutes` (**chỉ có khi build tag `testroutes`**) → test.
- Mọi lỗi theo `{code,message,trace_id,details?}`; panic → 500 `INTERNAL`.
- Phân trang con trỏ base64url JSON `{"v":1,"t":µs,"i":uuid}`, `(created_at,id) < ($1,$2)` không OFFSET.
- Idempotency: Redis, TTL 24 h, 50 goroutine → đúng 1 bản ghi; 409/422 đúng mã.
- Khoá lạc quan `version` → 409 `VERSION_CONFLICT`; ETag `W/"v<version>"` + 304.
- `POST /_test/jobs` → 202 `job_id`; `GET /api/v1/jobs/{id}` chỉ chủ job (người khác 404).
- Image mặc định không chứa route thử (`go tool nm` không có symbol gói testroutes).

## File đổi
`backend-go/internal/httpapi/{apierr,httpx}/*`, `internal/httpapi/{cors,ratelimit,idempotency,middleware,routes,router}.go`, `internal/httpapi/*_test.go`, `internal/testroutes/*`, `internal/jobs/*`, `cmd/gateway/testbuild_{on,off}.go`.

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg03.sh`.

## Test đã chạy và kết quả
Gói chính: `./internal/httpapi` (57 test), `./internal/jobs` (7) — gồm bộ tên QC (TestCursor_*, TestIdempotency_*, TestOptimisticLock_*, TestETag_*, TestRateLimit_*, TestPagination_*).

Kết quả thật (chạy ở HEAD):
- `go vet ./...` và `go vet -tags testroutes ./...`: không in gì.
- `golangci-lint run` và `golangci-lint run --build-tags testroutes`: `0 issues.` (cả hai).
- `go test -race -count=1 -tags testroutes ./...` ở HEAD `2d2834b`: 15 gói `ok`, 0 FAIL, 0 SKIP (cmd/gateway 5.1 s, cmd/worker 7.4 s, db 4.6 s, auth 23.6 s, contract 12.7 s, httpapi 14.9 s, httpapi/sse 46.9 s, jobs 9.0 s, blob 8.5 s, config 1.4 s, platform/db 21.1 s, log 1.4 s, otel 1.5 s, outbox 9.5 s, redis 3.8 s, store 11.0 s).
- `sqlc diff` rc=0, không in gì.

## AC tự đánh giá
- AC1: ✓
- AC2: ✓
- AC3: ✓
- AC4: ✓
- AC5: ✓
- AC6: ✓
- AC7: ✓
- AC8: ✓
- AC9: ✓
- AC10: ✓
- AC11: ✓
- AC12: ✓
- AC13: ✓
- AC14: ✓
- AC15: ✓
- AC16: ✓
- AC17: ✓
- AC18: ✓

## Nợ / chưa làm / cần hỏi
- Không. Helper ở gói lá `internal/httpapi/httpx` (không phải `httpapi`) để tránh vòng import với `testroutes` (build tag).
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.
