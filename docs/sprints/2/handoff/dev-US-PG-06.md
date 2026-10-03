# DEV handoff — US-PG-06 — OpenAPI + contract test
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-06'`)

## Đã làm (theo thứ tự lát dọc)
`api/openapi.yaml` (3.1.0, 5 đường dẫn) + `api/openapi.test.yaml` (13 đường dẫn / 15 thao tác, route thử) → `internal/contract` (`spec.go` nạp + router kin-openapi, `routes.go` liệt kê route thật, `exempt.go` miễn trừ — rỗng) → test hai chiều spec↔code, validator request/response, ca âm.
- kin-openapi **chỉ** trong `internal/contract` + `_test` (depguard); `go list -deps ./cmd/gateway ./cmd/worker | grep -c kin-openapi` = 0.
- **Góp ý #3(a) (PM ACCEPTED):** đăng ký `format: uuid` mẫu RFC 9562 (kin mặc định không kiểm uuid; RFC 4122 từ chối nhầm UUIDv7) — `formatOnce` trong `spec.go`; test `TestValidator_UUIDFormat` (v4, v7 đúng; `x`, rác, thiếu ký tự, rỗng bị bắt). Thêm cấm `nullable:` trong tệp 3.1 vào `TestSpec_LoadsAndValidates`.

## File đổi
`backend-go/api/openapi{,.test}.yaml`, `backend-go/internal/contract/*`, `backend-go/Makefile` (mục `lint-depguard-negative`).

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg06.sh`.

## Test đã chạy và kết quả
Gói chính: `./internal/contract` (11 hàm test; chạy cả không tag và `-tags testroutes`, đều PASS). `make lint-depguard-negative` rc=0. Ca âm kịch bản m1/m2/m3 qua `OPENAPI_PATH` đều đỏ đúng ở hai chế độ. `pnpm exec redocly lint` rc=0 (chỉ cảnh báo).

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

## Nợ / chưa làm / cần hỏi
- Spec dùng `servers: [{url: /}]`; contract bỏ `Doc.Servers` trước khi dựng router (router của kin so cả host). Không ảnh hưởng spec.
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.
