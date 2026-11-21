# Smoke hồi quy sprint 1 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15

Tổng TC cũ: 68 · Đạt (P): 28 · Trượt (F): 3 · Không chạy K (a/b/c): 0 · Lỗi thời (D): 32 · Chưa chạy — cần PM quyết (X): 5 · Chưa chạy — ngoài mẫu (N): 0   (P+F+K+D+X+N = 68)
Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».
Cách chạy:
- `bash docs/sprints/1/qc/scripts/tc-US-P0-01.sh` (14 TC), `tc-US-P0-02.sh` (tĩnh: 01–09 19 23–27 31 34–38; riêng 18, 21), `tc-US-P0-03.sh TC-04 05 13 14 16` — log ở `evidence-1/run-*.log`.
- Kiểm tương đương tay (curl/docker inspect trên ep-r) cho TC-12, 13, 14, 15, 32, 33: cổng gốc 8080/3000/9000 không còn công bố, thay bằng Caddy 1143 / :3700 / :9700.
- `pnpm -C frontend lint` + `build` (rc 0) cho TC-22; `pnpm install --frozen-lockfile` rc 0.
- `pnpm dev`/`dev:down` (TC-10, 16, 17, 28, 30) cần stack `edupilot` mặc định đang chạy của nhóm khác (13 service healthy) — dừng/dựng lại sẽ phá stack đó; kiểm tĩnh `docker compose config` rc 0 (run-logs/s1-second-stack-static.log). TC-29: container gateway thật trên mạng tạm qc2-j (run-logs/s1-tc29-j.log). TC-20/15: go vet + go test -race từng gói (evidence-2, gate-03/04).

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-P0-01 | 14 | 7 | 1 | 0 | 6 | 0 | 0 | tài liệu DEMO_SCRIPT đã viết lại v2 → 6 ca D |
| US-P0-02 | 38 | 19 | 1 | 0 | 13 | 5 | 0 | scaffold sprint 1: nhiều ca D vì cây đã đổi; X = `pnpm dev` stack thứ hai |
| US-P0-03 | 16 | 2 | 1 | 0 | 13 | 0 | 0 | CI sprint 1: bằng chứng run đã đóng, ci.yml tái cấu trúc |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| **US-P0-01** | | |
| P0-01/TC-05 | P | không từ khoá cấm ở lời SV |
| P0-01/TC-07 | F | DEF-R1-01: DEMO_SCRIPT trích attendance-checkin/-display/checkin-code/audit/insights.spec.ts nhưng FLOWS.md không có mục tương ứng |
| P0-01/TC-08 | P | 4 cụm phân quyền nằm đúng hàng bảng |
| P0-01/TC-09 | P |  |
| P0-01/TC-10 | P | email @edupilot.local, không secret |
| P0-01/TC-11 | P | đường dẫn trích dẫn tồn tại |
| P0-01/TC-12 | P |  |
| P0-01/TC-14 | P |  |
| **US-P0-02** | | |
| P0-02/TC-02 | P |  |
| P0-02/TC-03 | P |  |
| P0-02/TC-04 | P |  |
| P0-02/TC-06 | P |  |
| P0-02/TC-07 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): legacy/ không có commit từ 2026-10-01 (git log -- legacy: 0 commit mới; commit cuối 78bfeb6a) |
| P0-02/TC-10 | N | chưa chạy (trước đây X: `pnpm dev`/`dev:down` trên project `edupilot` mặc định đang chạy (13 service healthy, của nhóm khác): chạy khi mọi nhóm dừng; `doc) |
| P0-02/TC-11 | N | chưa chạy (đã ghi lỗi thời nhưng KHÔNG có nguồn quyết định → phải chạy): `dev:status` 6 service — nay 13 |
| P0-02/TC-12 | P | kiểm tương đương: GET /api/v1/healthz qua Caddy 1143 = 200 {"status":"ok"} JSON (cổng 8080 không còn công bố: US-PG-07 AC17) |
| P0-02/TC-13 | P | kiểm tương đương trên cổng ep-r: Mailpit info 200, MinIO live+console 200, PG 18.6 + vector, Redis 8.10 PONG, SMTP mở |
| P0-02/TC-14 | P | lang="vi", logo tham chiếu và phục vụ 200 (frontend :3700) |
| P0-02/TC-15 | P | gateway user nonroot; healthcheck `/gateway -healthcheck`; image không có sh/curl/wget |
| P0-02/TC-16…P0-02/TC-17 | N | chưa chạy (trước đây X: `pnpm dev`/`dev:down` trên project `edupilot` mặc định đang chạy (13 service healthy, của nhóm khác): chạy khi mọi nhóm dừng; `doc) |
| P0-02/TC-18 | P | PATH không docker → thoát 1, thông báo Docker, không stack trace |
| P0-02/TC-19 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): go 1.27, next 16, react 19, pnpm, node 24 (script PASS); mọi phụ thuộc frontend (react-query, react-virtual, lucide-react, uqr, next…) và Go trực tiếp (validator, OpenTelemetry/otelpgx/otelhttp, kin-openapi cho test theo D52) có trong docs/ARCHITECTURE.md hoặc DECISIONS.md |
| P0-02/TC-20 | F | DEF-R2-04 (xem defects-2.md): `go vet ./...` rc 0; `go test -race ./...` không tag: contract đỏ cố định; bản `-tags testroutes` xanh (3424 PASS, 0 FAIL) |
| P0-02/TC-21 | P | RERUN 2026-11-16 sau 816498c6 (DEF-R2-01 đã sửa): `redocly lint backend-go/api/openapi.yaml` → «valid», 0 lỗi (21 cảnh báo) |
| P0-02/TC-22 | P | `pnpm -C frontend lint` rc 0, `build` rc 0 |
| P0-02/TC-23 | P |  |
| P0-02/TC-24 | P |  |
| P0-02/TC-25 | P |  |
| P0-02/TC-26 | P |  |
| P0-02/TC-27 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): gateway đủ phụ thuộc: docker stop qc2s-gw → ExitCode 0 (US-PG-07 tắt êm; TC-29 4,7 s); thiếu DB/Redis thoát mã 1 ngay, không ghi tệp, không panic (script tc-US-P0-02 TC-27: 3 kiểm PASS) — luật mới US-PG-01 AC1 |
| P0-02/TC-28 | N | chưa chạy (trước đây X: `pnpm dev`/`dev:down` trên project `edupilot` mặc định đang chạy (13 service healthy, của nhóm khác): chạy khi mọi nhóm dừng; `doc) |
| P0-02/TC-29 | P | container gateway thật (image edupilot-gateway) trên mạng tạm qc2-j: `docker stop` → ExitCode 0, 4,7 s (<8 s), log «gateway stopped»; run-logs/s1-tc29-j.log |
| P0-02/TC-30 | N | chưa chạy (trước đây X: `pnpm dev`/`dev:down` trên project `edupilot` mặc định đang chạy (13 service healthy, của nhóm khác): chạy khi mọi nhóm dừng; `doc) |
| P0-02/TC-31 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): mã backend-go/frontend/scripts/compose không import/tham chiếu thư mục legacy/ (chỉ thư viện kin-openapi/routers/legacy trong internal/contract/spec.go) |
| P0-02/TC-32 | P | không mount legacy (chỉ ./seed/demo); `go list ./...` không gói legacy; history image không «legacy» |
| P0-02/TC-33 | P | kiểm tương đương /api/v1/healthz: body {"status":"ok"}, POST 405, đường dẫn lạ 404 JSON, có/không Authorization đều 200 |
| P0-02/TC-34 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): .env.example không có khoá thật, đủ nhóm biến, .env/.env.local không bị track (script TC-34: 3 PASS) |
| P0-02/TC-35 | P |  |
| P0-02/TC-36 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): git grep mẫu khoá (AKIA…, sk-…, AIza…, BEGIN PRIVATE KEY, ghp_…, xox…) trên tệp theo dõi ngoài test/fixtures/docs = 0 dòng |
| P0-02/TC-37 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): tần ghi đĩa trong mã sản xuất chỉ ở internal/llm/fake/fake.go (bản ghi replay của provider giả) và cmd/regress (CLI offline), ngoài đường request (luật 10); không fetch trần, không màu cứng ngoài tokens (script PASS), không float64 trong internal/grade; docker diff mỗi container chỉ 2 thư mục mount /seed |
| P0-02/TC-38 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): git ls-files *.py: 0 tệp ngoài benchmarks/, legacy/ và docs/ (docs/ chứa PoC nghiên cứu, fixture, script QC; không có service Python — D46) |
| **US-P0-03** | | |
| P0-03/TC-05 | P |  |
| P0-03/TC-12 | P | RERUN 22/11 @aba738ef: script PASS (2 kiểm; evidence-1/run-22nov/) |
| P0-03/TC-13 | P | permissions: contents: read |
| P0-03/TC-15 | F | DEF-R2-04: lệnh Go `go test -race ./...` không tag đỏ ở internal/contract; FE: install --frozen-lockfile/lint/build/antipatterns rc 0; bản -tags testroutes xanh |

## Trượt: bước tái hiện
### DEF-R1-01 — DEMO_SCRIPT trích spec E2E không có trong FLOWS (TC-07)
### DEF-R2-01 — ĐÃ SỬA (816498c6): redocly lint hợp lệ, TC-21 đạt — chi tiết `defects-2.md`


## TC đã xoá khỏi tổng (nhóm A, lỗi thời — nguồn P-11-92 PM 2026-11-22)
P0-01/TC-02, P0-01/TC-03, P0-01/TC-04, P0-01/TC-06, P0-01/TC-13, P0-02/TC-01, P0-02/TC-05, P0-02/TC-08, P0-02/TC-09, P0-03/TC-01, P0-03/TC-02, P0-03/TC-04, P0-03/TC-06, P0-03/TC-07, P0-03/TC-08, P0-03/TC-10, P0-03/TC-11, P0-03/TC-16