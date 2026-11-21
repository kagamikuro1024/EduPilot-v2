# Smoke hồi quy sprint 2 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15

Tổng TC cũ (cập nhật 22/11, bỏ D/X/K theo luật chủ dự án; mọi ca chưa có P/F là N): xem số P/F/N ở dòng tổng của env/qc2-progress.py
Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».
Cách chạy:
- `docs/regression/evidence-2/scripts/{pg01..pg07,gate-pg}.sh` (bản sao của `docs/sprints/2/qc/scripts`, chỉ chỉnh cổng/project sang ep-r: 6133/7080/9700/1143, `-p ep-r`, thêm APP_ENCRYPTION_KEY vào env nhị phân trần — bắt buộc từ sprint 3, proposals#4; docker bọc chặn lệnh phá huỷ). Log: `evidence-2/run-logs/<story>.log`.
- Lượt 1: 7 story chạy tuần tự/song song nhẹ trên ep-r (1 gateway, bản test). Ca cần dừng/scale/recreate (gateway=2, kill, stop postgres/redis/minio, chế độ default) chạy ở các cửa sổ khoá global (`scripts-lock/`, ≤ 400 s mỗi cửa sổ, khôi phục baseline sau mỗi cửa sổ): `run-logs/<story>-lock.log`.
- Ca trượt lượt 1 chạy lại lần 2 riêng (`*.rerun2.log`, `*.rerun3.log`).
- Go contract: `go test -race -tags testroutes ./internal/contract` (qua `gt`), thêm chạy riêng TestStatusParity.
- 02-39/40/41 (migrate down/up) chạy trên Postgres TẠM `qc2-j-pg` (docker run, cổng 6199) — `scripts-j/pg02-j.sh`, log `run-logs/02-j.log`; KHÔNG chạm ep-r. Cổng Go: `go vet` + `go test -race` TỪNG GÓI (-timeout 280 s, env CI: TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1; 12 test đo thời gian chạy lại riêng không đặt EP_SKIP_TIMING). 02-44 và GATE-20 (`pnpm dev` từ volume trống) = X: cần stack `edupilot` mặc định (đang chạy, của nhóm khác) dừng/xoá volume — chỉ làm khi mọi nhóm dừng; bằng chứng một phần: `migrate up` trên PG trống rc 0 (`run-logs/02-44-j.log`). Ca scripts-lock sửa cho ep-r (tên container `ep-r-gateway`, BASE k6 = $GW, ensure_mode ≥2 gateway, bỏ JWT khỏi env baseline): xem chú thích «QC2 (SmkJ)» trong `scripts-lock/*.sh`.

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-PG-01 | 85 | 80 | 0 | 0 | 5 | 0 | 0 | khung dịch vụ |
| US-PG-02 | 70 | 60 | 0 | 0 | 9 | 1 | 0 | schema/blob/outbox |
| US-PG-03 | 108 | 106 | 0 | 0 | 2 | 0 | 0 | idempotency/REST |
| US-PG-04 | 58 | 55 | 0 | 0 | 3 | 0 | 0 | JWT/auth |
| US-PG-05 | 74 | 74 | 0 | 0 | 0 | 0 | 0 | SSE |
| US-PG-06 | 38 | 31 | 0 | 0 | 7 | 0 | 0 | OpenAPI/contract |
| US-PG-07 | 69 | 52 | 1 | 0 | 16 | 0 | 0 | compose/CI |
| GATE-PG | 25 | 18 | 2 | 1 | 1 | 3 | 0 | cổng PG |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| **US-PG-01** | | |
| TC-PG01-01 | P | (stack tạm ep-r2) |
| TC-PG01-02 | P | (stack tạm ep-r2) |
| TC-PG01-03 | P | (stack tạm ep-r2) |
| TC-PG01-04 | P |  |
| TC-PG01-05 | P | (stack tạm ep-r2) |
| TC-PG01-06 | P | (stack tạm ep-r2) |
| TC-PG01-07 | P | (stack tạm ep-r2) |
| TC-PG01-08 | P | (stack tạm ep-r2) |
| TC-PG01-09 | P | (stack tạm ep-r2) |
| TC-PG01-10 | P | (stack tạm ep-r2) |
| TC-PG01-11 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG01-11: biến JWT_EXPIRATION không còn (đổi thành ACCESS_TOKEN_TTL, sprint 3) [ms khi JWT_EXPIRATION sai: thực tế [2457] phải ≤ 1000] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-12 | P | (stack tạm ep-r2) |
| TC-PG01-13 | P | (stack tạm ep-r2) |
| TC-PG01-14 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-15 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-16 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG01-16: APP_ENV=production nay kiểm khoá/CORS bắt buộc nên thoát <100 ms (sprint sau) [APP_ENV=production: đi tới bước chờ phụ thuộc (ms): thực tế [48] phải ≥ 1500] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-17 | P | (stack tạm ep-r2) |
| TC-PG01-18 | P | (stack tạm ep-r2) |
| TC-PG01-19 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-20 | P | (stack tạm ep-r2) |
| TC-PG01-21 | P | (stack tạm ep-r2) |
| TC-PG01-22 | P | (stack tạm ep-r2) |
| TC-PG01-23 | P | (stack tạm ep-r2) |
| TC-PG01-24 | P | (stack tạm ep-r2) |
| TC-PG01-25 | P | (stack tạm ep-r2) |
| TC-PG01-26 | P |  |
| TC-PG01-27 | P |  |
| TC-PG01-28 | P |  |
| TC-PG01-29 | P |  |
| TC-PG01-30 | P |  |
| TC-PG01-31 | P |  |
| TC-PG01-32 | P |  |
| TC-PG01-33 | P |  |
| TC-PG01-34 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG01-34: Redis ep-r dùng chung 7 nhóm khoá (ep:auth:*, ep:llm:* … thêm sau sprint 2) [số khoá Redis ngoài quy ước: thực tế [4] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-35 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-36 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-37 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-38 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-39 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-40 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-41 | P |  |
| TC-PG01-42 | P |  |
| TC-PG01-43 | P |  |
| TC-PG01-44 | P |  |
| TC-PG01-45 | P |  |
| TC-PG01-46 | P |  |
| TC-PG01-47 | P |  |
| TC-PG01-48 | P |  |
| TC-PG01-49 | P |  |
| TC-PG01-50 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-51 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-52 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-53 | P |  |
| TC-PG01-54 | P |  |
| TC-PG01-55 | P |  |
| TC-PG01-56 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-57 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-58 | P |  |
| TC-PG01-59 | P |  |
| TC-PG01-60 | P |  |
| TC-PG01-61 | P |  |
| TC-PG01-62 | P |  |
| TC-PG01-63 | P |  |
| TC-PG01-64 | P |  |
| TC-PG01-65 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-66 | P |  |
| TC-PG01-67 | P |  |
| TC-PG01-68 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG01-68: Makefile test nay có 2 lần RYUK_DISABLED=true [số lần RYUK_DISABLED=true: thực tế [2] ≠ mong đợi [1]] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-69 | P |  |
| TC-PG01-70 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-71 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-72 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-73 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-74 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG01-75 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-76 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-77 | P |  |
| TC-PG01-78 | P |  |
| TC-PG01-79 | P |  |
| TC-PG01-80 | P |  |
| TC-PG01-81 | P |  |
| TC-PG01-82 | P |  |
| TC-PG01-83 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-84 | P |  |
| TC-PG01-85 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG01-85: Makefile lint nay gọi golangci-lint 3 lần (không tag, testroutes, +1) [số lần gọi golangci-lint: thực tế [3] ≠ mong đợi [2]] (log evidence-2/run-logs-22nov/01-n.log) |
| **US-PG-02** | | |
| TC-PG02-01…TC-PG02-02 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-01: schema 00001→00029: nay 60+ bảng (sprint 3…11); TC đếm theo 00001 [danh sách bảng public: thực tế [assignments attendance_checkins attendance_records audit_log auth_sessions auth_tokens calendar_events chat_]; PG02-02: enum nay nhiều hơn (sprint 3…11) [danh sách enum: thực tế [appeal_status attempt_status attempt_submit_reason attendance_status auth_token_kind calendar_event_type chat_chann] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-03 | P |  |
| TC-PG02-04 | P |  |
| TC-PG02-05 | P |  |
| TC-PG02-06 | P |  |
| TC-PG02-07 | P |  |
| TC-PG02-08 | P |  |
| TC-PG02-09 | P |  |
| TC-PG02-10 | P |  |
| TC-PG02-11 | P |  |
| TC-PG02-12 | P |  |
| TC-PG02-13 | P |  |
| TC-PG02-14 | P |  |
| TC-PG02-15 | P |  |
| TC-PG02-16 | P |  |
| TC-PG02-17 | P |  |
| TC-PG02-18 | P |  |
| TC-PG02-19 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-19: index 308 (TC đòi 17 theo 00001) [số index: thực tế [308] ≠ mong đợi [17]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-20 | P |  |
| TC-PG02-21 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-21: 66 cột id uuidv7() (TC đòi 5) [cột id mặc định uuidv7(): thực tế [66] ≠ mong đợi [5]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-22 | P |  |
| TC-PG02-23 | P |  |
| TC-PG02-24 | P |  |
| TC-PG02-25 | P |  |
| TC-PG02-26 | P |  |
| TC-PG02-27 | P |  |
| TC-PG02-28 | P |  |
| TC-PG02-29 | P |  |
| TC-PG02-30 | P |  |
| TC-PG02-31 | P |  |
| TC-PG02-32 | P |  |
| TC-PG02-33 | P |  |
| TC-PG02-34 | P |  |
| TC-PG02-35 | P |  |
| TC-PG02-36 | P |  |
| TC-PG02-37 | P |  |
| TC-PG02-38 | P |  |
| TC-PG02-39 | P | chạy trên Postgres tạm qc2-j-pg (không chạm ep-r): down→up, pg_dump -s trước/sau giống hệt (7333 dòng); run-logs/02-j.log |
| TC-PG02-40 | P | Postgres tạm qc2-j-pg: sau down 0/5 bảng nền, extension vector giữ, up khôi phục 5 bảng; run-logs/02-j.log |
| TC-PG02-41 | F | RERUN 22/11 @aba738ef trên Postgres tạm (evidence-2/scripts-j/pg02-j.sh, log run-logs-22nov/02-j.log): migrate up lần hai rc=0 và goose_db_version không thêm dòng (đạt) nhưng «vẫn đúng 5 bảng + goose = 6» thực tế 83 bảng public (schema 00001→00029) — TC lệch sau sprint 3…11, chờ PM quyết sửa TC/LT (proposal) |
| TC-PG02-42 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-42: goose version mới nhất 29 (TC đòi 1) [goose_db_version.version_id mới nhất: thực tế [29] ≠ mong đợi [1]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-43 | P |  |
| TC-PG02-44 | N | chưa chạy (trước đây X: cần `pnpm dev` từ volume trống = dừng + xoá volume của stack `edupilot` MẶC ĐỊNH đang chạy (13 service healthy, của nhóm khác) — c) |
| TC-PG02-45 | P |  |
| TC-PG02-46 | P |  |
| TC-PG02-47 | P |  |
| TC-PG02-48 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-48: float64 trong mã sqlc sinh từ cột double precision (sprint sau; spec v1.5 miễn _test.go) [không float64 trong mã sản xuất internal/store (spec v1.5, #12: _test.go miễn): thực tế [19] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-49 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-49: 1 dòng «OFFSET» nằm trong chú thích «không OFFSET» (observability.sql:2), không có OFFSET trong câu SQL — grep quá chặt [OFFSET trong internal/store/queries: thực tế [1] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-50 | P |  |
| TC-PG02-51 | P |  |
| TC-PG02-52 | P |  |
| TC-PG02-53 | P |  |
| TC-PG02-54 | P |  |
| TC-PG02-55 | P |  |
| TC-PG02-56 | P |  |
| TC-PG02-57 | P |  |
| TC-PG02-58 | P |  |
| TC-PG02-59 | P |  |
| TC-PG02-60 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-61 | P |  |
| TC-PG02-62 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-63 | P |  |
| TC-PG02-64 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-65 | P |  |
| TC-PG02-66 | P |  |
| TC-PG02-67 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-68 | P |  |
| TC-PG02-69 | P |  |
| TC-PG02-70 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG02-70: cùng lý do PG02-01/19/21 (schema tăng) [bảng public sau khi loại _test_%: thực tế [assignments attendance_checkins attendance_records audit_log auth_sessions auth_tokens calendar_e] (log evidence-2/run-logs-22nov/02-n.log) |
| **US-PG-03** | | |
| TC-PG03-01 | P |  |
| TC-PG03-02 | P |  |
| TC-PG03-03 | P |  |
| TC-PG03-04 | P |  |
| TC-PG03-05 | P |  |
| TC-PG03-06 | P |  |
| TC-PG03-07 | P |  |
| TC-PG03-08 | P |  |
| TC-PG03-09 | P |  |
| TC-PG03-10 | P |  |
| TC-PG03-11 | P |  |
| TC-PG03-12 | P |  |
| TC-PG03-13 | P |  |
| TC-PG03-14 | P |  |
| TC-PG03-15 | P |  |
| TC-PG03-16 | P |  |
| TC-PG03-17 | P |  |
| TC-PG03-18 | P |  |
| TC-PG03-19 | P |  |
| TC-PG03-20 | P |  |
| TC-PG03-21 | P |  |
| TC-PG03-22 | P |  |
| TC-PG03-23 | P |  |
| TC-PG03-24 | P |  |
| TC-PG03-25 | P |  |
| TC-PG03-26 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-27 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-28 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-29 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-30 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-31 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-32 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-33 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-34 | P |  |
| TC-PG03-35 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-36 | P |  |
| TC-PG03-37 | P |  |
| TC-PG03-38 | P |  |
| TC-PG03-39 | P |  |
| TC-PG03-40 | P |  |
| TC-PG03-41 | P |  |
| TC-PG03-42 | P |  |
| TC-PG03-43 | P |  |
| TC-PG03-44 | P |  |
| TC-PG03-45 | P |  |
| TC-PG03-46 | P |  |
| TC-PG03-47 | P |  |
| TC-PG03-48 | P |  |
| TC-PG03-49 | P |  |
| TC-PG03-50 | P |  |
| TC-PG03-51 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG03-51: 11 dòng OFFSET trong mã/chú thích ngoài _test.go (sprint sau thêm truy vấn OFFSET ở bảng nhỏ hoặc chú thích) [số dòng có OFFSET (ngoài _test.go): thực tế [11] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/03-n.log) |
| TC-PG03-52 | P |  |
| TC-PG03-53 | P |  |
| TC-PG03-54 | P |  |
| TC-PG03-55 | P |  |
| TC-PG03-56 | P |  |
| TC-PG03-57 | P |  |
| TC-PG03-58 | P |  |
| TC-PG03-59 | P |  |
| TC-PG03-60 | P |  |
| TC-PG03-61 | P |  |
| TC-PG03-62 | P |  |
| TC-PG03-63 | P |  |
| TC-PG03-64 | P |  |
| TC-PG03-65 | P |  |
| TC-PG03-66 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-67 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-68 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-69 | P |  |
| TC-PG03-70 | P |  |
| TC-PG03-71 | P |  |
| TC-PG03-72 | P |  |
| TC-PG03-73 | P |  |
| TC-PG03-74 | P |  |
| TC-PG03-75 | P |  |
| TC-PG03-76 | P |  |
| TC-PG03-77 | P |  |
| TC-PG03-78 | P |  |
| TC-PG03-79 | P |  |
| TC-PG03-80 | P |  |
| TC-PG03-81 | P |  |
| TC-PG03-82 | P |  |
| TC-PG03-83 | P |  |
| TC-PG03-84 | P |  |
| TC-PG03-85 | P |  |
| TC-PG03-86 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-87 | P |  |
| TC-PG03-88 | P |  |
| TC-PG03-89 | P |  |
| TC-PG03-90 | P |  |
| TC-PG03-91 | P |  |
| TC-PG03-92 | P |  |
| TC-PG03-93 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-94 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-95 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-96 | P |  |
| TC-PG03-97 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-98 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG03-98: APP_ENV=production: log lỗi nêu biến production khác (kiểm tra production sprint sau) [log nêu APP_ENV: thực tế [0] phải ≥ 1] (log evidence-2/run-logs-22nov/03-n.log) |
| TC-PG03-99 | P |  |
| TC-PG03-100 | P |  |
| TC-PG03-101 | P |  |
| TC-PG03-102 | P |  |
| TC-PG03-103 | P |  |
| TC-PG03-104 | P |  |
| TC-PG03-105 | P |  |
| TC-PG03-106 | P |  |
| TC-PG03-107 | P |  |
| TC-PG03-108 | P |  |
| **US-PG-04** | | |
| TC-PG04-01 | P |  |
| TC-PG04-02 | P |  |
| TC-PG04-03 | P |  |
| TC-PG04-04 | P |  |
| TC-PG04-05 | P |  |
| TC-PG04-06 | P |  |
| TC-PG04-07 | P |  |
| TC-PG04-08 | P |  |
| TC-PG04-09 | P |  |
| TC-PG04-10 | P |  |
| TC-PG04-11 | P |  |
| TC-PG04-12 | P |  |
| TC-PG04-13 | P |  |
| TC-PG04-14 | P |  |
| TC-PG04-15 | P |  |
| TC-PG04-16 | P |  |
| TC-PG04-17 | P |  |
| TC-PG04-18 | P |  |
| TC-PG04-19 | P |  |
| TC-PG04-20 | P |  |
| TC-PG04-21 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG04-22 | P |  |
| TC-PG04-23 | P |  |
| TC-PG04-24 | P |  |
| TC-PG04-25 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG04-26 | P |  |
| TC-PG04-27 | P |  |
| TC-PG04-28 | P |  |
| TC-PG04-29 | P |  |
| TC-PG04-30 | P |  |
| TC-PG04-31 | P |  |
| TC-PG04-32 | P |  |
| TC-PG04-33 | P |  |
| TC-PG04-34 | P |  |
| TC-PG04-35 | P |  |
| TC-PG04-36 | P |  |
| TC-PG04-37 | P |  |
| TC-PG04-38 | P |  |
| TC-PG04-39 | P |  |
| TC-PG04-40 | P |  |
| TC-PG04-41 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG04-42 | P |  |
| TC-PG04-43 | P |  |
| TC-PG04-44 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG04-44: openapi nay có /auth/ /me/ /admin/ (sprint 3…11), TC đòi 0 [grep -c '/auth/' backend-go/api/openapi.yaml: thực tế [11] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/04-n.log) |
| TC-PG04-45 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG04-45: route auth/me/admin nay tồn tại (405/415/200), TC đòi 404 theo sprint 2 [GET /api/v1/auth/login (chế độ test) · status: thực tế [405] ≠ mong đợi [404]] (log evidence-2/run-logs-22nov/04-n.log) |
| TC-PG04-46 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG04-46: route auth/me/admin nay tồn tại (401/405/415), TC đòi 404 theo sprint 2 [không vào được chế độ default] (log evidence-2/run-logs-22nov/04-n.log) |
| TC-PG04-47 | P |  |
| TC-PG04-48 | P |  |
| TC-PG04-49 | P |  |
| TC-PG04-50 | P |  |
| TC-PG04-51 | P |  |
| TC-PG04-52 | P |  |
| TC-PG04-53 | P |  |
| TC-PG04-54 | P |  |
| TC-PG04-55 | P |  |
| TC-PG04-56 | P |  |
| TC-PG04-57 | P |  |
| TC-PG04-58 | P |  |
| **US-PG-05** | | |
| TC-PG05-01 | P |  |
| TC-PG05-02 | P |  |
| TC-PG05-03 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-04 | P |  |
| TC-PG05-05 | P |  |
| TC-PG05-06 | P |  |
| TC-PG05-07 | P |  |
| TC-PG05-08 | P |  |
| TC-PG05-09 | P |  |
| TC-PG05-10 | P |  |
| TC-PG05-11 | P |  |
| TC-PG05-12 | P |  |
| TC-PG05-13 | P |  |
| TC-PG05-14 | P |  |
| TC-PG05-15 | P |  |
| TC-PG05-16 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-17 | P |  |
| TC-PG05-18 | P |  |
| TC-PG05-19 | P |  |
| TC-PG05-20 | P |  |
| TC-PG05-21 | P |  |
| TC-PG05-22 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-23 | P |  |
| TC-PG05-24 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-25 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-26 | P |  |
| TC-PG05-27 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-28 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-29 | P |  |
| TC-PG05-30 | P |  |
| TC-PG05-31 | P |  |
| TC-PG05-32 | P |  |
| TC-PG05-33 | P |  |
| TC-PG05-34 | P |  |
| TC-PG05-35 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-36 | P |  |
| TC-PG05-37 | P |  |
| TC-PG05-38 | P |  |
| TC-PG05-39 | P |  |
| TC-PG05-40 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-41 | P |  |
| TC-PG05-42 | P |  |
| TC-PG05-43 | P |  |
| TC-PG05-44 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-45 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-46 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-47 | P |  |
| TC-PG05-48 | P |  |
| TC-PG05-49 | P |  |
| TC-PG05-50 | P |  |
| TC-PG05-51 | P |  |
| TC-PG05-52 | P |  |
| TC-PG05-53 | P |  |
| TC-PG05-54 | P |  |
| TC-PG05-55 | P |  |
| TC-PG05-56 | P |  |
| TC-PG05-57 | P |  |
| TC-PG05-58 | P |  |
| TC-PG05-59 | P |  |
| TC-PG05-60 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-61 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-62 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-63 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-64 | P |  |
| TC-PG05-65 | P |  |
| TC-PG05-66 | P |  |
| TC-PG05-67 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-68 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-69 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG05-70 | P |  |
| TC-PG05-71 | P |  |
| TC-PG05-72 | P |  |
| TC-PG05-73 | P |  |
| TC-PG05-74 | P |  |
| **US-PG-06** | | |
| TC-PG06-01 | P | RERUN 2026-11-16 sau 816498c6 (DEF-R2-01 đã sửa): `redocly lint backend-go/api/openapi.yaml` rc 0, 0 lỗi (21 cảnh báo) |
| TC-PG06-02 | P |  |
| TC-PG06-03 | P |  |
| TC-PG06-04…TC-PG06-07 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG06-04: openapi.yaml 299 đường dẫn (TC đòi 5) [grep -cE '^  /' api/openapi.yaml: thực tế [299] ≠ mong đợi [5]]; PG06-05: 2 chữ «_test» nằm trong mô tả (sprint sau) [grep -c '_test' api/openapi.yaml: thực tế [2] ≠ mong đợi [0]]; PG06-06: openapi.test.yaml 19 đường dẫn thử (TC đòi 13) [grep -cE '^  /api/v1/_test' api/openapi.test.yaml: thực tế [19] ≠ mong đợi [13]]; PG06-07: 345 thao tác (TC đòi 5), 21 thao tác thử (TC đòi 15) [số thao tác openapi.yaml: thực tế [345] ≠ mong đợi [5]] (log evidence-2/run-logs-22nov/06-n.log) |
| TC-PG06-08 | P |  |
| TC-PG06-09 | P |  |
| TC-PG06-10 | P | DEF-R2-02 = do tải, đóng: `go test -count=1 -tags testroutes ./internal/contract/...` ok 37 s lúc load ≈ 6 (bản dbf48ed0); thêm TestStatusParity riêng PASS 2/3 lượt (110 s, 46 s); evidence-2/logs-contract-rerun2.log |
| TC-PG06-11 | P | như TC-PG06-10 (gồm TestStatusParity + TestSpec_EveryDocumentedStatusExercised trong gói contract xanh) |
| TC-PG06-12 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG06-13 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG06-14 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG06-15 | P |  |
| TC-PG06-16 | P |  |
| TC-PG06-17 | P |  |
| TC-PG06-18 | P |  |
| TC-PG06-19 | P |  |
| TC-PG06-20 | P |  |
| TC-PG06-21 | P |  |
| TC-PG06-22 | P |  |
| TC-PG06-23 | P |  |
| TC-PG06-24 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG06-24:  (log evidence-2/run-logs-22nov/06-n.log) |
| TC-PG06-25 | P |  |
| TC-PG06-26 | P |  |
| TC-PG06-27 | P |  |
| TC-PG06-28 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG06-28:  (log evidence-2/run-logs-22nov/06-n.log) |
| TC-PG06-29 | P |  |
| TC-PG06-30 | P |  |
| TC-PG06-31 | P |  |
| TC-PG06-32 | P |  |
| TC-PG06-33 | P |  |
| TC-PG06-34 | P |  |
| TC-PG06-35 | P |  |
| TC-PG06-36 | P |  |
| TC-PG06-37 | P |  |
| TC-PG06-38 | P | TC tay: exempt.go (đếm operation/status/reason = 0/0/0, reason rỗng 0) khớp handoff dev-US-PG-06.md «exempt.go miễn trừ — rỗng»: hai danh sách cùng 0 mục; TC-PG06-28 PASS cùng lượt |
| **US-PG-07** | | |
| TC-PG07-01…TC-PG07-02 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-01: compose nay 13 service (thêm judge, mock-graph, mock-imap…) [danh sách service của docker-compose.local.yml: thực tế [caddy frontend gateway judge mailpit migrate minio mock-graph mock-imap pgbouncer p]; PG07-02: compose nay có docling (11 chuỗi) [chuỗi docling trong compose: thực tế [11] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-03 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-04 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-05 | P |  |
| TC-PG07-06 | P |  |
| TC-PG07-07 | P |  |
| TC-PG07-08 | P |  |
| TC-PG07-09 | P | RERUN 22/11 @aba738ef trên ep-r (cổng đổi 80→780): curl -sI http://localhost:780/ → HTTP/1.1 308 Permanent Redirect, Location: https://localhost/ |
| TC-PG07-10 | P |  |
| TC-PG07-11 | P |  |
| TC-PG07-12 | P |  |
| TC-PG07-13 | P |  |
| TC-PG07-14 | P | RERUN 22/11 @aba738ef trên ep-r (cổng 443→1143): openssl s_client -connect localhost:1143 → issuer CN=Caddy Local Authority - ECC Intermediate; SAN DNS:localhost; notAfter=Nov 22 01:07:54 2026 GMT |
| TC-PG07-15 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-16 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-17 | P |  |
| TC-PG07-18 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-19 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-20 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-21 | P |  |
| TC-PG07-22 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-22:  (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-23 | P |  |
| TC-PG07-24 | P |  |
| TC-PG07-25 | P |  |
| TC-PG07-26 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-27 | P |  |
| TC-PG07-28 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-29 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-30 | P |  |
| TC-PG07-31 | P |  |
| TC-PG07-32 | P |  |
| TC-PG07-33 | P |  |
| TC-PG07-34 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-35 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-35: 4 chỗ ghi đĩa: internal/llm/fake/fake.go (ghi bản ghi replay của provider giả, chỉ dùng test/demo) và cmd/regress/main.go (CLI offline) — không nằm trên đường request [số chỗ ghi đĩa trong mã sản xuất: thực tế [4] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-36 | P |  |
| TC-PG07-37 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-37: mỗi container có 2 thay đổi FS: A /seed, A /seed/demo (thư mục mount seed của ep-r) [số thay đổi hệ thống file của 325f91ee45ade8bf9413ae65629e6695dfe9051dd6f40ccb0885db34462815a0: thực tế [2] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-38…TC-PG07-41 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-38: gh không có run CI cho sha 06f164c7 (commit cũ của sprint 2 đã bị thay bằng rebase/merge) [headSha của run mới nhất: thực tế [KHÔNG CÓ] ≠ mong đợi [06f164c780e58475cbdcc3dd74228ff32635a344]]; PG07-39: không tìm được run CI của sprint 2 [tìm được run id: [] không khớp /^[0-9]+$/]; PG07-40: không có run CI sprint 2 để đếm bước [số bước có 'vet': thực tế [0] phải ≥ 2]; PG07-41: không có run CI sprint 2 để đếm job [số job tên chứa Frontend: thực tế [0] phải ≥ 1] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-42 | P |  |
| TC-PG07-43…TC-PG07-44 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-43: nhánh ci/sqlc-drift đã xoá (TC ghi: xoá sớm = FAIL) [không tìm thấy run ci.yml của nhánh ci/sqlc-drift (chạy lại với RUN_ID=<id> lấy từ handoff; nhánh bị xoá sớm hơn lúc QC chấm = FAIL)]; PG07-44: không có run ci/sqlc-drift để đối chiếu [không có run ci/sqlc-drift để đối chiếu] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-45 | P |  |
| TC-PG07-46 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-47 | P |  |
| TC-PG07-48 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-49 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-50 | P |  |
| TC-PG07-51 | P |  |
| TC-PG07-52 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-53 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-54 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-55 | P |  |
| TC-PG07-56 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-56: compose nay công bố thêm cổng mock-graph, mock-imap; trên ep-r caddy ánh xạ 780/1143, frontend không công bố 3000 (ep-r dùng cổng riêng) [tập service có cổng công bố: thực tế [caddy frontend mailpit minio mock-graph mock-imap postgres redis] ≠ mong đợi [caddy frontend mailpit m] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-57 | P |  |
| TC-PG07-58 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-58: .env.example có SEED_DEFAULT_PASSWORD không chứa «-dev» (mật khẩu seed theo thiết kế) [số giá trị bí mật của .env.example KHÔNG chứa '-dev': thực tế [1] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-59 | P |  |
| TC-PG07-60 | P |  |
| TC-PG07-61 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-62 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-63 | F | DEF-R2-04: cổng GATE-PG chạy trọn còn 2 ca đỏ (GATE-03, GATE-06: contract không tag); 01–02, 04–24, 26, 28–29 đạt; GATE-25/27 (trình duyệt), GATE-20 (pnpm dev) chưa chạy |
| TC-PG07-64 | P |  |
| TC-PG07-65 | P |  |
| TC-PG07-66 | F | RERUN 22/11 @aba738ef FAIL — TC lệch spec/sản phẩm sau sprint 2, chờ PM quyết sửa TC/LT (proposal): PG07-66: compose có override nay 13 service (thêm judge, mock-graph, mock-imap) [danh sách service khi có override: thực tế [caddy frontend gateway judge mailpit migrate minio mock-graph mock-imap pgbouncer postgres redis] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-67 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-68 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG07-69 | P |  |
| **GATE-PG** | | |
| TC-GATE-01 | P |  |
| TC-GATE-02 | P |  |
| TC-GATE-03 | F | DEF-R2-04: `go test -race ./...` KHÔNG tag: 98 gói xanh (3360 PASS) nhưng `internal/contract` đỏ cố định (TestChatThreadsContract: POST /_test/llm/fake → 404; TestContract_ResponsesMatchSpec «0 quan sát»; TestSpec_EveryDocumentedStatusExercised) — kịch bản contract từ sprint 7 cần route thử (`-tags testroutes`); Makefile/CI chỉ chạy bản có tag |
| TC-GATE-04 | P | `go vet -tags testroutes ./...` rc 0; `go test -race -count=1 -tags testroutes` TỪNG GÓI (100 gói, -timeout 280 s, nhiều tay ngoài cổng): 3424 test PASS, 0 FAIL; 14 skip = 12 test đo thời gian (EP_SKIP_TIMING=1 của CI) đã chạy lại riêng không đặt biến: cả 12 PASS, + 2 test llm «BLOCKED: cần khoá API thật» (sprint 3+, K(b)); sửa DEF-R2-03 (TestLLMSchema, c272d3d3) rồi store/privacy/grading chạy lại xanh; /tmp/qc2-j-gate-testroutes/ |
| TC-GATE-05 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-06 | F | DEF-R2-04: contract KHÔNG tag rc 1 (cố định, lặp 3 lượt + go test trực tiếp); có tag PASS |
| TC-GATE-07 | P | 10/10 test nêu tên PASS + mọi gói `internal/httpapi/...` (-race, testroutes) PASS khi chạy từng gói (không gộp 1 lệnh 20 phút) |
| TC-GATE-08 | P | 6/6 test SSE + 5/5 test auth nêu tên PASS; gói internal/httpapi/sse và internal/auth (-race, testroutes) PASS từng gói |
| TC-GATE-09 | P | `pnpm -C frontend build` rc 0 (35 s; chạy ngoài cửa sổ khoá); run-logs/gate-09-j.log |
| TC-GATE-10 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-11 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-12 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-13 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-20 | N | chưa chạy (trước đây X: cần `pnpm dev` từ volume trống = dừng + xoá volume của stack `edupilot` MẶC ĐỊNH đang chạy (13 service healthy, của nhóm khác) — c) |
| TC-GATE-21 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-22 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-23 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-24 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-25 | N | chưa chạy (trước đây X: cần trình duyệt thật (Playwright qua eval `sse-browser-cut.mjs` / `idem-two-tabs.mjs`, ≥ 1 cửa sổ khoá) — chưa chạy; phần tương đư) |
| TC-GATE-26 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-27 | N | chưa chạy (trước đây X: cần trình duyệt thật (Playwright qua eval `sse-browser-cut.mjs` / `idem-two-tabs.mjs`, ≥ 1 cửa sổ khoá) — chưa chạy; phần tương đư) |
| TC-GATE-28 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-29 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-GATE-30 | N | chưa chạy (trước đây Kb: đọc tay toàn bộ diff internal/auth + internal/httpapi theo checklist — cần người đọc (script gom ứng viên đã chạy rc 0: run-logs/g) |
| TC-GATE-99 | N | chưa chạy (đã ghi lỗi thời nhưng KHÔNG có nguồn quyết định → phải chạy): dọn cuối của cổng cũ (trả stack về image mặc định, 2 gateway): nay baseline ep-r do PM đặt là 1 gateway + image edupilot-*-test; mọi cửa sổ  |

## Trượt: bước tái hiện
### DEF-R2-01 — ĐÃ SỬA (816498c6): `redocly lint` hợp lệ, TC-PG06-01 và P0-02/TC-21 đạt — chi tiết `defects-2.md`
### DEF-R2-02 — ĐÓNG, do tải (rig contract REQUEST_TIMEOUT=1s): gói contract xanh khi máy rảnh, TC-PG06-10, -11 đạt — chi tiết `defects-2.md`
### DEF-R2-03 — ĐÃ SỬA (c272d3d3): TestLLMSchema 6→9 bảng `llm_%` (migration 00027, sprint 11) làm go test đỏ ở GATE-03/04, P0-02/TC-20, P0-03/TC-15 — chi tiết `defects-2.md`
