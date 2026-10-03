# DEV handoff — US-PG-05 — SSE hạ tầng
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-05'`)

## Đã làm (theo thứ tự lát dọc)
`internal/httpapi/sse`: Publisher (Redis Stream theo người, `MAXLEN ~ SSE_BUFFER_MAXLEN`, TTL 1 h) → Handler (`GET /api/v1/events`: `ready`, `heartbeat` 25 s, `shutdown`, `reconnect`, `resync`, `max duration` 120 s; `Last-Event-ID` đọc bù, **subscribe trước rồi đọc bù, loại trùng theo id**) → giới hạn 2 stream/người bằng slot Lua (429 `SSE_LIMIT_REACHED`) → tiến độ job (`job.progress`) → test.

## File đổi
`backend-go/internal/httpapi/sse/*`, `backend-go/internal/testroutes/events.go`.

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg05.sh`.

## Test đã chạy và kết quả
Gói chính: `./internal/httpapi/sse` (25 test, tên khớp bộ QC: TestSSE_Headers, TestSSE_LastEventID_NoLossNoDup, TestSSE_ReconnectRace, TestSSE_CrossInstance, TestSSE_GatewayShutdownFailover, TestSSE_MaxTwoPerUser…) — 46.9 s với `-race`.

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

## Nợ / chưa làm / cần hỏi
- AC14 (qua Caddy), AC10/AC13 (hai gateway thật) kiểm ở GATE.
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.
