# DEV handoff — US-PG-04 — Auth nền: JWT, bcrypt, RBAC, CourseAccessGuard
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-04'`)

## Đã làm (theo thứ tự lát dọc)
`internal/auth` (JWT HS256 iss=edupilot, aud=edupilot-api, leeway 5 s; `Principal` chỉ đi trong context) → middleware xác thực → `RequireRole` → `CourseAccessGuard` (resolver mặc định "từ chối tất cả", không cache) → bcrypt cost 12 (4–14, sai → thoát 1) → `gateway token` (dev only, thoát 1 khi `APP_ENV=production`) → test.
- Không có route `/auth/*`, `/me/*`, `/admin/*` (AC10).
- Không log mật khẩu / hash / token đầy đủ (AC9).

## File đổi
`backend-go/internal/auth/*`, `backend-go/cmd/gateway/token.go`.

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg04.sh`.

## Test đã chạy và kết quả
Gói chính: `./internal/auth` (20 hàm test, 98 ca con trong lần chạy riêng), `./cmd/gateway` (TestToken_*).

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

## Nợ / chưa làm / cần hỏi
- Không.
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.
