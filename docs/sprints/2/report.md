# Sprint 2 — báo cáo (PG Nền Go)

Mục tiêu: trọn `docs/phases/PG.md` L1–L7 — gateway Go không trạng thái làm nền cho mọi phase sau, không endpoint nghiệp vụ · Kết quả: **7/7 story PASS · cổng PG PASS** sau 1 vòng sửa · Nhánh `sprint/2-pg` (từ `main` `b1a726f`)

| Story | Lát PG | Trạng thái | QC (TC) |
| --- | --- | --- | --- |
| US-PG-01 Khung dịch vụ: config chết sớm kèm tên biến, slog + OTel `trace_id`, Redis, tắt êm, giới hạn tầng, pool pgx + log chậm, worker, Makefile | L1 | PASS | `qc/report-US-PG-01.md` (85) |
| US-PG-02 Dữ liệu: goose `00001 pg_platform`, sqlc, `platform/blob` (MinIO), `platform/outbox` + consumer retry 3 / dead-letter | L2 | PASS | `qc/report-US-PG-02.md` (70) |
| US-PG-03 Chuẩn HTTP: middleware, lỗi thống nhất, cursor, Idempotency-Key, `version`/ETag, jobs 202 | L3 | PASS | `qc/report-US-PG-03.md` (108) |
| US-PG-04 Auth nền: JWT HS256, bcrypt, RBAC, khung `CourseAccessGuard` | L4 | PASS | `qc/report-US-PG-04.md` (58) |
| US-PG-05 SSE: id, heartbeat, fan-out Redis pub/sub, `Last-Event-ID` + `resync`, 2 kết nối / 120 s | L5 | PASS | `qc/report-US-PG-05.md` (74) |
| US-PG-06 Hợp đồng: `api/openapi.yaml` 3.1 + `internal/contract` (kin-openapi chỉ trong test) | L6 | PASS | `qc/report-US-PG-06.md` (38) |
| US-PG-07 Hạ tầng: Caddy 2.11, PgBouncer 1.26 transaction mode, image < 40 MB, `--scale gateway=2`, CI + sqlc diff, k6 smoke | L7 | PASS | `qc/report-US-PG-07.md` (69) |

Cổng: `qc/report-GATE-PG.md` — 25/25 PASS (gồm rút mạng SSE 10 s trên Chrome thật, tắt gateway giữa stream, gửi đôi Idempotency-Key hai tab, `down -v` rồi `up`). Spec: `docs/specs/FEAT-pg-foundation/` v1.6.

## Số liệu
- Test case: **527** (01: 85 · 02: 70 · 03: 108 · 04: 58 · 05: 74 · 06: 38 · 07: 69 · GATE: 25). Vòng 1: 8 FAIL → vòng sửa 1 → 0 FAIL. (Sửa 2026-10-10: bản trước ghi 536 vì đếm cả dòng tiêu đề bảng `TC-id`.)
- `go test -race` ≥ 356 test, 15 gói, 0 skip; `golangci-lint` 0 issues; `sqlc diff` sạch; CI xanh.
- Số nền (`benchmarks/reports/pg-baseline.md`): RAM nghỉ gateway ≈ 4 MiB; image gateway 9,4 MB / worker 8,9 MB; k6 smoke p95 `healthz` 5,75 ms, `jobs/{id}` qua PgBouncer 8,57 ms, 0 % lỗi; tắt một gateway giữa 200 request: 1 lỗi.
- 42 commit, `backend-go` +19.531 dòng. Góp ý #1–#13, PM chấp nhận 13.

## Lỗi thật tìm ra và đã sửa
- Hồi phục sau khi Postgres khởi động lại chậm 11–19 s (PgBouncer cache NXDOMAIN) → `server_login_retry=1`, DNS TTL.
- Caddy `unhealthy_status 503` nuốt 503 hợp lệ của ứng dụng (BUG-PG-1) → header `X-EP-Draining` khi tắt êm.
- `Last-Event-ID` vượt bộ đệm không nhận sự kiện live (BUG-PG-2) → `resync` rồi nhận live (#13).
- Middleware timeout với Redis (`TestDeadline_Redis`), test chập chờn so chuỗi log (BUG-PG-3), vòng chờ phụ thuộc lúc khởi động (BUG-PG-4).

## Quyết định trong sprint
- **D52** `kin-openapi` chỉ trong test. Route thử khoá bằng build tag `testroutes`, image mặc định trả 404 (#1). `CORS_ORIGINS` không bao giờ `*` (#2). Giữ Caddy `dynamic a` + health check bị động; upstream tĩnh + `health_uri` là nợ P10/PR (#3). Luật cấm `float64` áp cho điểm / tiền (#12).

## Nợ
- Caddy upstream tĩnh + health check chủ động khi số bản gateway cố định (P10/PR).
- Mỗi máy dev/QC cần `~/.testcontainers.properties` (README backend) để `go test` trần chạy trên colima.
- `run-all.sh` của QC chết khi chạy nền — chạy từng script ở foreground.

## Chủ dự án tự kiểm (từ `PG.md`)
- `pnpm dev:down -v && pnpm dev`: migration chạy từ `00001`, không thao tác tay.
- Xoá `JWT_SECRET_KEY` khỏi `.env.local`: gateway thoát ngay, log nêu đúng tên biến.
- `docker compose … up -d --scale gateway=2` rồi `curl -fsSk https://localhost/api/v1/healthz` nhiều lần: `X-Instance-Id` đổi giữa hai bản.
- Đọc diff `internal/auth` và `internal/httpapi/` — phần mọi phase sau kế thừa.
