# Smoke hồi quy sprint 1 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15

Tổng TC cũ: 68 · Đạt (P): 27 · Trượt (F): 2 · Không chạy K (a/b/c): 0 · Lỗi thời (D): 32 · Chưa chạy — cần PM quyết (X): 5 · Chưa chạy — ngoài mẫu (N): 2   (P+F+K+D+X+N = 68)
Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».
Cách chạy:
- `bash docs/sprints/1/qc/scripts/tc-US-P0-01.sh` (14 TC), `tc-US-P0-02.sh` (tĩnh: 01–09 19 23–27 31 34–38; riêng 18, 21), `tc-US-P0-03.sh TC-04 05 13 14 16` — log ở `evidence-1/run-*.log`.
- Kiểm tương đương tay (curl/docker inspect trên ep-r) cho TC-12, 13, 14, 15, 32, 33: cổng gốc 8080/3000/9000 không còn công bố, thay bằng Caddy 1143 / :3700 / :9700.
- `pnpm -C frontend lint` + `build` (rc 0) cho TC-22; `pnpm install --frozen-lockfile` rc 0.
- Không dựng stack `edupilot` thứ hai (luật RAM PM) → `pnpm dev`/`dev:down` = X.

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-P0-01 | 14 | 7 | 1 | 0 | 6 | 0 | 0 | tài liệu DEMO_SCRIPT đã viết lại v2 → 6 ca D |
| US-P0-02 | 38 | 18 | 1 | 0 | 13 | 5 | 1 | scaffold sprint 1: nhiều ca D vì cây đã đổi; X = `pnpm dev` stack thứ hai |
| US-P0-03 | 16 | 2 | 0 | 0 | 13 | 0 | 1 | CI sprint 1: bằng chứng run đã đóng, ci.yml tái cấu trúc |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| **US-P0-01** | | |
| P0-01/TC-01 | D | DEMO_SCRIPT v2 (bd032bf2): 8 bước, thêm F7 điểm danh QR + bước nhật ký — TC kiểm cấu trúc v1 (7 bước) |
| P0-01/TC-02 | D | DEMO_SCRIPT v2: 8 bảng mốc — TC kiểm 7 bảng |
| P0-01/TC-03 | D | DEMO_SCRIPT v2: tổng thời lượng/mốc «12:15» thay «13:45» |
| P0-01/TC-04 | D | DEMO_SCRIPT v2: route /checkin?c=… + vai «2 sinh viên» ngoài danh sách TC |
| P0-01/TC-05 | P | không từ khoá cấm ở lời SV |
| P0-01/TC-06 | D | DEMO_SCRIPT v2 viết lại mục 5 (dự phòng): khoá DEMO_MODE/«Ai bật»/«lỗi 2 lần» không còn |
| P0-01/TC-07 | F | DEF-R1-01: DEMO_SCRIPT trích attendance-checkin/-display/checkin-code/audit/insights.spec.ts nhưng FLOWS.md không có mục tương ứng |
| P0-01/TC-08 | P | 4 cụm phân quyền nằm đúng hàng bảng |
| P0-01/TC-09 | P |  |
| P0-01/TC-10 | P | email @edupilot.local, không secret |
| P0-01/TC-11 | P | đường dẫn trích dẫn tồn tại |
| P0-01/TC-12 | P |  |
| P0-01/TC-13 | D | script đòi dòng «Phiên bản 1»; tài liệu nay «Phiên bản 2» |
| P0-01/TC-14 | P |  |
| **US-P0-02** | | |
| P0-02/TC-01 | D | gốc repo nay có benchmarks/ deploy/ infra/ mobile/ … |
| P0-02/TC-02 | P |  |
| P0-02/TC-03 | P |  |
| P0-02/TC-04 | P |  |
| P0-02/TC-05 | D | tokens.css + brand: 6 tệp, tokens đã đổi qua nhiều sprint UI |
| P0-02/TC-06 | P |  |
| P0-02/TC-07 | D | legacy/ có tệp mới (sprint sau) |
| P0-02/TC-08 | D | script đòi commit dời thuần rename — cây đã đổi ≥ 1.700 commit |
| P0-02/TC-09 | D | compose nay 13 service (thêm caddy, pgbouncer, judge, mock-*, worker…) — TC đòi 6 |
| P0-02/TC-10 | X | `pnpm dev` dựng stack project edupilot cổng mặc định — cấm dựng stack thứ hai (luật RAM PM) |
| P0-02/TC-11 | D | `dev:status` 6 service — nay 13 |
| P0-02/TC-12 | P | kiểm tương đương: GET /api/v1/healthz qua Caddy 1143 = 200 {"status":"ok"} JSON (cổng 8080 không còn công bố: US-PG-07 AC17) |
| P0-02/TC-13 | P | kiểm tương đương trên cổng ep-r: Mailpit info 200, MinIO live+console 200, PG 18.6 + vector, Redis 8.10 PONG, SMTP mở |
| P0-02/TC-14 | P | lang="vi", logo tham chiếu và phục vụ 200 (frontend :3700) |
| P0-02/TC-15 | P | gateway user nonroot; healthcheck `/gateway -healthcheck`; image không có sh/curl/wget |
| P0-02/TC-16 | X | xoá .env.local rồi `pnpm dev` — chạm .env.local dùng chung + dựng stack mặc định (cấm) |
| P0-02/TC-17 | X | như TC-16 (sửa .env.local + `pnpm dev`) |
| P0-02/TC-18 | P | PATH không docker → thoát 1, thông báo Docker, không stack trace |
| P0-02/TC-19 | D | có phụ thuộc mới (uqr, react-dom 19.3…) ngoài phạm vi TC |
| P0-02/TC-20 | N | = `go vet` + `go test -race ./...` toàn bộ — thuộc GATE-03/04 sprint 2 (chưa chạy do tải máy) |
| P0-02/TC-21 | F | DEF-R2-01: `redocly lint` openapi.yaml 1 lỗi (analytics/exams/compare: parameters null) |
| P0-02/TC-22 | P | `pnpm -C frontend lint` rc 0, `build` rc 0 |
| P0-02/TC-23 | P |  |
| P0-02/TC-24 | P |  |
| P0-02/TC-25 | P |  |
| P0-02/TC-26 | P |  |
| P0-02/TC-27 | D | gateway nay thoát 1 khi thiếu DB/Redis (US-PG-01 AC1) — TC đòi /healthz sống |
| P0-02/TC-28 | X | `pnpm dev:down` project edupilot — cấm chạm/dựng stack thứ hai |
| P0-02/TC-29 | P | container gateway thật (image edupilot-gateway) trên mạng tạm qc2-j: `docker stop` → ExitCode 0, 4,7 s (<8 s), log «gateway stopped»; run-logs/s1-tc29-j.log |
| P0-02/TC-30 | X | `dev:down` khi đã dừng — như TC-28 |
| P0-02/TC-31 | D | grep «legacy» trúng testdata/lib `kin-openapi/routers/legacy`, test mask — không phải tham chiếu thư mục legacy |
| P0-02/TC-32 | P | không mount legacy (chỉ ./seed/demo); `go list ./...` không gói legacy; history image không «legacy» |
| P0-02/TC-33 | P | kiểm tương đương /api/v1/healthz: body {"status":"ok"}, POST 405, đường dẫn lạ 404 JSON, có/không Authorization đều 200 |
| P0-02/TC-34 | D | .env.example nay > 10 nhóm biến; «8640…» là số TTL trùng, không phải secret |
| P0-02/TC-35 | P |  |
| P0-02/TC-36 | D | phạm vi diff từ sprint 1 → HEAD: trúng dữ liệu giả trong test (token=AAAA…, sk-ABCDEFGH) |
| P0-02/TC-37 | D | luật «gateway không ghi đĩa»: nay có cmd/regress, fake llm ghi tệp (đã duyệt) |
| P0-02/TC-38 | D | benchmarks/ và 20 tệp .py ngoài legacy/docs (judge/benchmark — sprint sau) |
| **US-P0-03** | | |
| P0-03/TC-01 | D | bằng chứng là run GitHub Actions của nhánh sprint/1-p0-prep — nhánh/run đã đóng |
| P0-03/TC-02…P0-03/TC-03 | D | như TC-01 |
| P0-03/TC-04 | D | ci.yml đã tái cấu trúc (US-CI-02/TL-CI: matrix, job tách, Node 24, cache tự quản) |
| P0-03/TC-05 | P |  |
| P0-03/TC-06 | D | như TC-01 (log run cũ) |
| P0-03/TC-07 | D | cần RUN_RED_GO; nhánh ci/red-check đã xoá |
| P0-03/TC-08…P0-03/TC-09 | D | như TC-07 |
| P0-03/TC-10 | D | cần RUN_RED_UI; nhánh ci/red-check đã xoá |
| P0-03/TC-11 | D | như TC-10 |
| P0-03/TC-12 | D | nhánh ci/red-check đã xoá — TC chỉ chấm sau khi dev xoá |
| P0-03/TC-13 | P | permissions: contents: read |
| P0-03/TC-14 | D | diff .github từ 1b72f54: thêm actions/fe-setup, xoá keep-huggingface (sprint sau) |
| P0-03/TC-15 | N | Go: GATE-01/02 PASS (vet, golangci-lint ×2); Go test -race ./… chưa chạy; FE: install --frozen-lockfile/lint/build/antipatterns rc 0 — thiếu `pnpm test` & go test |
| P0-03/TC-16 | D | HEAD detached/1.740 commit chưa so với origin — không còn nghĩa |

## Trượt: bước tái hiện
### DEF-R1-01 — DEMO_SCRIPT trích spec E2E không có trong FLOWS (TC-07)
### DEF-R2-01 — openapi.yaml 1 lỗi redocly (TC-21) — chi tiết `defects-2.md`
