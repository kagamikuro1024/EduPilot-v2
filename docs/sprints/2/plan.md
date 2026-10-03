# Sprint 2 — PG Nền Go

Trạng thái: **PM tự duyệt 2026-10-01** (chủ dự án giao toàn quyền: "không dừng, triển khai luôn sprint 2, blocking tự quyết") · Nhánh: `sprint/2-pg` từ `main` (`b1a726f`), làm trong worktree `../TA_Agent_v2-s2` để không đụng nhánh `sprint/1.5-mock-ui` đang sửa UI.

## Mục tiêu
Xong trọn `docs/phases/PG.md` (L1–L7): gateway Go **không trạng thái** với khung dịch vụ, tầng dữ liệu, chuẩn HTTP, auth nền, SSE, hợp đồng OpenAPI, hạ tầng chạy (Caddy, PgBouncer, 2 bản gateway). **Không một endpoint nghiệp vụ nào.** Cuối sprint chạy cổng PG.

## Story (thứ tự thi công)

| # | Story | Lát PG | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 1 | US-PG-01 Khung dịch vụ: config, slog + otel `trace_id`, redis, graceful shutdown, giới hạn tầng, pool pgx + log chậm, `cmd/worker`, Makefile | L1 | M | – |
| 2 | US-PG-02 Dữ liệu: goose `00001 pg_platform` (users dạng cuối, audit_log, outbox, jobs, idempotency_keys, vector), sqlc, `platform/blob`, `platform/outbox` + consumer | L2 | L | 01 |
| 3 | US-PG-03 Chuẩn HTTP: middleware, mã lỗi thống nhất, cursor, Idempotency-Key, version/ETag, jobs 202 | L3 | L | 01, 02 |
| 4 | US-PG-04 Auth nền: JWT, bcrypt, RBAC, khung CourseAccessGuard | L4 | M | 03 |
| 5 | US-PG-05 SSE: id, heartbeat, Redis pub/sub fan-out, Last-Event-ID, giới hạn 2 kết nối / 120 s | L5 | M | 03 |
| 6 | US-PG-06 Hợp đồng API: `api/openapi.yaml` nguồn sự thật, `internal/contract` | L6 | S | 03–05 |
| 7 | US-PG-07 Hạ tầng chạy: Caddy, PgBouncer, Dockerfile < 40 MB, `--scale gateway=2`, CI thêm sqlc diff + race, k6 smoke | L7 | M | 01–06 |

Spec: `docs/specs/FEAT-pg-foundation/` (US.md, SRS.md, QUESTIONS.md) — BA viết, PM duyệt. Test case: `docs/sprints/2/qc/`.

## Quy trình
BA spec → PM duyệt → dev thi công từng story (commit `US-PG-0N: …`) ∥ QC viết TC từ AC → dev handoff `docs/sprints/2/handoff/` → QC chạy TC + cổng PG → PM báo cáo, push, mở PR vào `main`.

## Quyết định PM tự chốt
- **D52**: thêm `getkin/kin-openapi` (chỉ dùng trong test `internal/contract`) để kiểm response khớp `openapi.yaml`; tự viết validator là việc thừa. Thư viện OTel exporter: `go.opentelemetry.io/otel` + stdout/otlp exporter (đã có dòng "OpenTelemetry Go" trong ARCHITECTURE).
- Sprint 2 không đụng `frontend/` ngoài `pnpm -C frontend build` của cổng; prototype 1.5 ở nhánh riêng.
- Hai nhánh mở PR riêng vào `main`; ai merge sau thì PM gộp `main` vào nhánh đó và giải xung đột tài liệu (`PROGRESS.md`, `README.md`).
- k6 smoke: SLO lấy ở `SYSTEM_DESIGN.md` mục 5; nếu máy dev (colima 4 CPU) không đạt, ghi số đo + lý do vào báo cáo, không nới SLO.

## Rủi ro
- PgBouncer transaction mode + pgx prepared statement → phải dùng `QueryExecModeExec`/`SimpleProtocol`; test qua PgBouncer thật.
- SSE qua Caddy bị buffer → `flush_interval -1`.
- testcontainers trong colima cần `DOCKER_HOST`/`TESTCONTAINERS_RYUK_DISABLED`; ghi vào Makefile.
