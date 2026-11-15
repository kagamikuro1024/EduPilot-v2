# Smoke hồi quy sprint 2 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15

Tổng TC cũ: 527 · Đạt (P): 408 · Trượt (F): 8 · Không chạy K (a/b/c): 0 · Lỗi thời (D): 40 · Chưa chạy — cần PM quyết (X): 71 · Chưa chạy — ngoài mẫu (N): 0   (P+F+K+D+X+N = 527)
Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».
Cách chạy:
- `docs/regression/evidence-2/scripts/{pg01..pg07,gate-pg}.sh` (bản sao của `docs/sprints/2/qc/scripts`, chỉ chỉnh cổng/project sang ep-r: 6133/7080/9700/1143, `-p ep-r`, thêm APP_ENCRYPTION_KEY vào env nhị phân trần — bắt buộc từ sprint 3, proposals#4; docker bọc chặn lệnh phá huỷ). Log: `evidence-2/run-logs/<story>.log`.
- Lượt 1: 7 story chạy tuần tự/song song nhẹ trên ep-r (1 gateway, bản test). Ca cần dừng/scale/recreate (gateway=2, kill, stop postgres/redis/minio, chế độ default) chạy ở các cửa sổ khoá global (`scripts-lock/`, ≤ 400 s mỗi cửa sổ, khôi phục baseline sau mỗi cửa sổ): `run-logs/<story>-lock.log`.
- Ca trượt lượt 1 chạy lại lần 2 riêng (`*.rerun2.log`, `*.rerun3.log`).
- Go contract: `go test -race -tags testroutes ./internal/contract` (qua `gt`), thêm chạy riêng TestStatusParity.
- X (chờ chạy): ca còn trong chuỗi cửa sổ khoá hoặc cần xoá schema/volume ep-r dùng chung (02-39/40/41/44, gate-20) — xin PM quyết (chạy được khi không còn nhóm khác dùng ep-r).

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-PG-01 | 85 | 80 | 0 | 0 | 5 | 0 | 0 | khung dịch vụ |
| US-PG-02 | 70 | 59 | 1 | 0 | 9 | 1 | 0 | schema/blob/outbox |
| US-PG-03 | 108 | 102 | 4 | 0 | 2 | 0 | 0 | idempotency/REST |
| US-PG-04 | 58 | 54 | 0 | 0 | 2 | 2 | 0 | JWT/auth |
| US-PG-05 | 74 | 55 | 0 | 0 | 0 | 19 | 0 | SSE |
| US-PG-06 | 38 | 25 | 3 | 0 | 6 | 4 | 0 | OpenAPI/contract |
| US-PG-07 | 69 | 31 | 0 | 0 | 16 | 22 | 0 | compose/CI |
| GATE-PG | 25 | 2 | 0 | 0 | 0 | 23 | 0 | cổng PG |

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
| TC-PG01-11 | D | biến JWT_EXPIRATION không còn (thay bằng ACCESS_TOKEN_TTL, sprint 3) — giá trị abc không còn bị báo |
| TC-PG01-12 | P | (stack tạm ep-r2) |
| TC-PG01-13 | P | (stack tạm ep-r2) |
| TC-PG01-14 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-15 | P | chạy lại lần 2 (đã thêm APP_ENCRYPTION_KEY vào env TC — bắt buộc từ sprint 3, proposals#4) |
| TC-PG01-16 | D | APP_ENV=production thoát ngay (<100 ms): sprint sau thêm kiểm tra bắt buộc cho production (khoá/CORS) — không còn "đợi phụ thuộc" |
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
| TC-PG01-34 | D | khoá Redis ngoài quy ước: Redis ep-r dùng chung với 7 nhóm khác (ep:auth:*, ep:llm:* … thêm sau sprint 2) |
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
| TC-PG01-68 | D | test Ryuk: 2 lần RYUK_DISABLED=true (Makefile sprint sau) |
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
| TC-PG01-85 | D | golangci-lint gọi 3 lần thay vì 2 (Makefile/CI sprint sau) |
| **US-PG-02** | | |
| TC-PG02-01…TC-PG02-02 | D | schema 00001 → 00025: nay 60+ bảng/enum/index (sprint 3…11) — TC đếm cố định theo 00001 |
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
| TC-PG02-19 | D | schema 00001 → 00025: nay 60+ bảng/enum/index (sprint 3…11) — TC đếm cố định theo 00001 |
| TC-PG02-20 | P |  |
| TC-PG02-21 | D | schema 00001 → 00025: nay 60+ bảng/enum/index (sprint 3…11) — TC đếm cố định theo 00001 |
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
| TC-PG02-41 | D | Postgres tạm: `migrate up` lần hai rc 0 và goose_db_version không thêm dòng (AC6 đạt); chỉ ý «đúng 5 bảng+goose» lỗi thời: nay 80 bảng (schema 00001→00025, sprint 3…11); run-logs/02-j.log |
| TC-PG02-42 | D | goose version mới nhất 25 (TC đòi 1) |
| TC-PG02-43 | P |  |
| TC-PG02-44 | X | `pnpm dev:down` + xoá volume + `pnpm dev` project edupilot — cấm (stack thứ hai / xoá dữ liệu chung) |
| TC-PG02-45 | P |  |
| TC-PG02-46 | P |  |
| TC-PG02-47 | P |  |
| TC-PG02-48 | D | float64 trong mã sqlc sinh ra từ cột double precision (sprint sau) |
| TC-PG02-49 | D | chuỗi «OFFSET» chỉ trong dòng chú thích «không OFFSET» (observability.sql:2); không có OFFSET trong câu SQL — grep quá chặt |
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
| TC-PG02-60 | F | CẦN TRIAGE: dòng outbox job.enqueue cùng xmin với dòng jobs (cùng transaction): thực tế [0] phải ≥ 1 |
| TC-PG02-61 | P |  |
| TC-PG02-62 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-63 | P |  |
| TC-PG02-64 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-65 | P |  |
| TC-PG02-66 | P |  |
| TC-PG02-67 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG02-68 | P |  |
| TC-PG02-69 | P |  |
| TC-PG02-70 | D | schema 00001 → 00025: nay 60+ bảng/enum/index (sprint 3…11) — TC đếm cố định theo 00001 |
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
| TC-PG03-32 | F | CẦN TRIAGE: có hai bản gateway: [] không khớp /^.+$/ |
| TC-PG03-33 | F | CẦN TRIAGE: XFF 203.0.113.7 request thứ 6 bị 429: [] không khớp / 429/ |
| TC-PG03-34 | P |  |
| TC-PG03-35 | F | CẦN TRIAGE: độ trễ thêm so với baseline (ms): thực tế [236] phải ≤ 50 |
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
| TC-PG03-51 | D | OFFSET nằm trong câu chú thích/SQL truy vấn sau sprint 2 (xem `grep`): TC đòi 0 |
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
| TC-PG03-95 | F | CẦN TRIAGE: số dòng _test_items tên qc95 (bảng thử không bị ghi): thực tế [1] ≠ mong đợi [0] |
| TC-PG03-96 | P |  |
| TC-PG03-97 | P | cửa sổ khoá global (gateway=2/chaos) |
| TC-PG03-98 | D | APP_ENV=production: log lỗi nêu biến production khác, không nêu APP_ENV (kiểm tra production sprint sau) |
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
| TC-PG04-25 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
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
| TC-PG04-44 | D | TC đòi 0 đường dẫn /auth/ /me/ /admin/ trong openapi — nay có (US-PU-01…) |
| TC-PG04-45 | D | GET /api/v1/auth/login → 405 (route auth có thật từ sprint 3), /admin/users → 200 — TC đòi 404 |
| TC-PG04-46 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
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
| TC-PG05-03 | X | lần 1 trượt khi 6 luồng + nhóm khác cùng tải stack; chưa chạy lại riêng để xác nhận |
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
| TC-PG05-16 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG05-17 | P |  |
| TC-PG05-18 | P |  |
| TC-PG05-19 | P |  |
| TC-PG05-20 | P |  |
| TC-PG05-21 | P |  |
| TC-PG05-22 | X | cần gateway=2 (ep-r chạy 1 gateway) — chờ cửa sổ khoá global |
| TC-PG05-23 | P |  |
| TC-PG05-24 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG05-25 | X | bước tay (chờ 150 s/đo thủ công) — chưa chạy  |
| TC-PG05-26 | P |  |
| TC-PG05-27 | X | bước tay (chờ 150 s/đo thủ công) — chưa chạy  |
| TC-PG05-28 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG05-29 | P |  |
| TC-PG05-30 | P |  |
| TC-PG05-31 | P |  |
| TC-PG05-32 | P |  |
| TC-PG05-33 | P |  |
| TC-PG05-34 | P |  |
| TC-PG05-35 | X | lần 1 trượt khi 6 luồng + nhóm khác cùng tải stack; chưa chạy lại riêng để xác nhận |
| TC-PG05-36 | P |  |
| TC-PG05-37 | P |  |
| TC-PG05-38 | P |  |
| TC-PG05-39 | P |  |
| TC-PG05-40 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG05-41 | P |  |
| TC-PG05-42 | P |  |
| TC-PG05-43 | P |  |
| TC-PG05-44…TC-PG05-46 | X | cần gateway=2 (ep-r chạy 1 gateway) — chờ cửa sổ khoá global |
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
| TC-PG05-60…TC-PG05-63 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG05-64 | P |  |
| TC-PG05-65 | P |  |
| TC-PG05-66 | P |  |
| TC-PG05-67…TC-PG05-69 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG05-70 | P |  |
| TC-PG05-71 | P |  |
| TC-PG05-72 | P |  |
| TC-PG05-73 | P |  |
| TC-PG05-74 | P |  |
| **US-PG-06** | | |
| TC-PG06-01 | F | DEF-R2-01: redocly lint openapi.yaml 1 lỗi (analytics/exams/compare: parameters null) |
| TC-PG06-02 | P |  |
| TC-PG06-03 | P |  |
| TC-PG06-04…TC-PG06-07 | D | openapi.yaml 5 đường dẫn (sprint 2) → 298 đường dẫn / 344 thao tác (sprint 3…11); 2 chữ «_test» trong mô tả |
| TC-PG06-08 | P |  |
| TC-PG06-09 | P |  |
| TC-PG06-10 | F | DEF-R2-02: `go test ./internal/contract` rc 1 (TestStatusParity trượt 2/2 lần) |
| TC-PG06-11 | F | DEF-R2-02: gồm TestStatusParity |
| TC-PG06-12…TC-PG06-14 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG06-15 | P |  |
| TC-PG06-16 | P |  |
| TC-PG06-17 | P |  |
| TC-PG06-18 | P |  |
| TC-PG06-19 | P |  |
| TC-PG06-20 | P |  |
| TC-PG06-21 | P |  |
| TC-PG06-22 | P |  |
| TC-PG06-23 | P |  |
| TC-PG06-24 | D | thông báo lỗi hợp đồng nêu GET /api/v1/jobs/{id} — spec nay 298 route, thông báo nêu route khác |
| TC-PG06-25 | P |  |
| TC-PG06-26 | P |  |
| TC-PG06-27 | P |  |
| TC-PG06-28 | D | -run chỉ 2 test: TestSpec_EveryDocumentedStatusExercised cần chạy cả bộ kịch bản — TC lỗi thời theo cách chọn test |
| TC-PG06-29 | P |  |
| TC-PG06-30 | P |  |
| TC-PG06-31 | P |  |
| TC-PG06-32 | P |  |
| TC-PG06-33 | P |  |
| TC-PG06-34 | P |  |
| TC-PG06-35 | P |  |
| TC-PG06-36 | P |  |
| TC-PG06-37 | P |  |
| TC-PG06-38 | X | bước tay (chờ 150 s/đo thủ công) — chưa chạy  |
| **US-PG-07** | | |
| TC-PG07-01…TC-PG07-02 | D | compose nay 13 service (thêm judge, docling, mock-*, migrate…) — TC đòi 6–8 service, không python/docling |
| TC-PG07-03 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-04 | X | cần gateway=2 (ep-r chạy 1 gateway) — chờ cửa sổ khoá global |
| TC-PG07-05 | P |  |
| TC-PG07-06 | P |  |
| TC-PG07-07 | P |  |
| TC-PG07-08 | P |  |
| TC-PG07-09 | D | cổng/chứng chỉ chuẩn 80/443/5433 — ep-r chạy cổng offset 780/1143/6133 (khác biệt cấu hình stack, không phải mã) |
| TC-PG07-10 | P |  |
| TC-PG07-11 | P |  |
| TC-PG07-12 | P |  |
| TC-PG07-13 | P |  |
| TC-PG07-14 | D | cổng/chứng chỉ chuẩn 80/443/5433 — ep-r chạy cổng offset 780/1143/6133 (khác biệt cấu hình stack, không phải mã) |
| TC-PG07-15…TC-PG07-16 | X | cần gateway=2 (ep-r chạy 1 gateway) — chờ cửa sổ khoá global |
| TC-PG07-17 | P |  |
| TC-PG07-18…TC-PG07-20 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-21 | P |  |
| TC-PG07-22 | D | cổng/chứng chỉ chuẩn 80/443/5433 — ep-r chạy cổng offset 780/1143/6133 (khác biệt cấu hình stack, không phải mã) |
| TC-PG07-23 | P |  |
| TC-PG07-24 | P |  |
| TC-PG07-25 | P |  |
| TC-PG07-26 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-27 | P |  |
| TC-PG07-28…TC-PG07-29 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-30 | P |  |
| TC-PG07-31 | P |  |
| TC-PG07-32 | P |  |
| TC-PG07-33 | P |  |
| TC-PG07-34 | X | cần gateway=2 (ep-r chạy 1 gateway) — chờ cửa sổ khoá global |
| TC-PG07-35 | D | mã sản xuất nay có ghi đĩa (regress, fake llm) / thay đổi FS container worker |
| TC-PG07-36 | P |  |
| TC-PG07-37 | D | mã sản xuất nay có ghi đĩa (regress, fake llm) / thay đổi FS container worker |
| TC-PG07-38…TC-PG07-41 | D | bằng chứng là run GitHub Actions của nhánh ci/sqlc-drift (nhánh đã xoá) và ci.yml tái cấu trúc (US-CI-02) |
| TC-PG07-42 | P |  |
| TC-PG07-43…TC-PG07-44 | D | bằng chứng là run GitHub Actions của nhánh ci/sqlc-drift (nhánh đã xoá) và ci.yml tái cấu trúc (US-CI-02) |
| TC-PG07-45 | P |  |
| TC-PG07-46 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-47 | P |  |
| TC-PG07-48…TC-PG07-49 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-50 | P |  |
| TC-PG07-51 | P |  |
| TC-PG07-52…TC-PG07-54 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-55 | P |  |
| TC-PG07-56 | D | nay thêm mock-graph/mock-imap công bố cổng (sprint 7/10.8) |
| TC-PG07-57 | P |  |
| TC-PG07-58 | D | .env.example có giá trị bí mật không chứa «-dev» (khoá demo sprint sau) |
| TC-PG07-59 | P |  |
| TC-PG07-60 | P |  |
| TC-PG07-61…TC-PG07-62 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-63 | X | bước tay (chờ 150 s/đo thủ công) — chưa chạy  |
| TC-PG07-64 | P |  |
| TC-PG07-65 | P |  |
| TC-PG07-66 | D | compose nay 13 service (thêm judge, docling, mock-*, migrate…) — TC đòi 6–8 service, không python/docling |
| TC-PG07-67…TC-PG07-68 | X | cần lệnh phá huỷ/scale trên ep-r (dừng postgres/redis, gateway=2, chế độ default) — chưa chạy trong cửa sổ khoá global |
| TC-PG07-69 | P |  |
| **GATE-PG** | | |
| TC-GATE-01 | P |  |
| TC-GATE-02 | P |  |
| TC-GATE-03…TC-GATE-99 | X | chưa chạy (gate: go test toàn bộ/xoá volume — chưa tới lượt) |

## Trượt: bước tái hiện
### DEF-R2-01 — openapi.yaml không qua `redocly lint` (TC-PG06-01; cũng P0-02/TC-21) — chi tiết `defects-2.md`
### DEF-R2-02 — `go test ./internal/contract` đỏ (TC-PG06-10, -11) — chi tiết `defects-2.md`
