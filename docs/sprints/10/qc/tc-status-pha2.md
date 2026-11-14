# Trạng thái pha 2 theo từng TC — sprint 10 (trên stack thật)
Cập nhật 2026-11-14 ở `57f4cb25`. Chạy theo thứ tự ưu tiên của PM; TC không có trong bảng «đã chạy» = **chưa chạy** (hết thời gian pha 2), tính FAIL theo `QC.md` nhưng không phải lỗi sản phẩm. Chi tiết lỗi: `report-stack.md`.

Tổng: CHƯA CHẠY 90, FAIL 26, NGOÀI 1, PASS 133 (tổng 250).

| TC | Trạng thái | Ghi chú |
| --- | --- | --- |
| TC-01-01 | PASS | bảng, enum `practice_mode` / `practice_status`, không có `practice_exams` (DB sạch) |
| TC-01-02 | PASS | FK phức hợp, CHECK `(mode='MOCK')=(deadline_at IS NOT NULL)`, UNIQUE từng phần, chỉ mục `(course_id,student_id,started_at DESC,id DESC)` và `(status,deadline_at)` |
| TC-01-03 | PASS | SV không có `enrollments` → `23503`; phần `practice_items` lớp khác chưa chạy (ngân hàng lớp 2 trống) |
| TC-01-04 | CHƯA CHẠY |  |
| TC-01-05 | PASS | Go (pha không stack) |
| TC-01-06 | PASS | trùng lượt dở cùng chủ đề → `23505`; khác chủ đề được |
| TC-01-07 | PASS | Go (pha không stack) |
| TC-01-08 | PASS | API stack |
| TC-01-09 | PASS | API stack |
| TC-01-10 | PASS | Go (pha không stack) |
| TC-01-11 | PASS | Go (pha không stack) |
| TC-01-12 | CHƯA CHẠY |  |
| TC-01-13 | PASS | API stack |
| TC-01-14 | PASS | Go + quét API câu chưa trả lời / đã trả lời / MOCK |
| TC-01-15 | PASS | Go (pha không stack) |
| TC-01-16 | PASS | API stack |
| TC-01-P1 | PASS | API stack |
| TC-01-P2 | PASS | API stack |
| TC-01-P3 | PASS | GV/TA/ADMIN → 403 trên lượt SV |
| TC-01-P4 | PASS | sv.gioi → 404 |
| TC-02-01 | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-02 | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-02b | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-03 | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-04 | PASS | API stack |
| TC-02-05 | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-06 | CHƯA CHẠY |  |
| TC-02-07 | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-08 | PASS | API stack |
| TC-02-09 | FAIL | L1: trích đề docling thật → 0/24 câu (report-stack mục 2) |
| TC-02-10 | CHƯA CHẠY |  |
| TC-02-11 | PASS | API stack |
| TC-02-12 | PASS | tài liệu riêng của lớp khác → 404 |
| TC-02-13 | PASS | bảng đáp án riêng của lớp khác → 404; sai loại → 422 |
| TC-02-14 | FAIL | KHÔNG KIỂM ĐƯỢC: công thức OVERLOADED của dev không làm đầy hàng cho việc nền |
| TC-02-15 | CHƯA CHẠY |  |
| TC-02-16 | CHƯA CHẠY |  |
| TC-02-17 | CHƯA CHẠY |  |
| TC-02-18 | FAIL | HOÃN theo chủ đề dự án (k6 / Lighthouse / axe) |
| TC-02-19 | PASS | API stack |
| TC-02-20 | PASS | Go TestExtractPageCap (chưa thử PDF >60 trang trên stack) |
| TC-02-P1 | PASS | API stack |
| TC-02-P2 | PASS | API stack |
| TC-02-P3 | PASS | API stack |
| TC-02-P4 | PASS | API stack |
| TC-02-P5 | PASS | API stack |
| TC-03-01 | PASS | API stack |
| TC-03-02 | PASS | API stack |
| TC-03-03 | CHƯA CHẠY |  |
| TC-03-04 | PASS | API stack |
| TC-03-05 | CHƯA CHẠY |  |
| TC-03-06 | CHƯA CHẠY |  |
| TC-03-07 | FAIL | L9: types=[] → 202 |
| TC-03-08 | FAIL | KHÔNG KIỂM ĐƯỢC (như 02-14) |
| TC-03-09 | PASS | sinh lại: 4/5 câu `DUPLICATE`; câu thứ 5 được tạo vì bản gốc đã bị QC sửa stem |
| TC-03-10 | PASS | API stack |
| TC-03-11 | PASS | `{unchanged:2, edited:1, rejected:2, pending:0}` |
| TC-03-12 | PASS | API stack |
| TC-03-13 | PASS | API stack |
| TC-04-01 | CHƯA CHẠY |  |
| TC-04-02 | FAIL | L11: menu không có «Soạn tay» |
| TC-04-03 | CHƯA CHẠY |  |
| TC-04-04 | CHƯA CHẠY |  |
| TC-04-05 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-06 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-07 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-08 | CHƯA CHẠY |  |
| TC-04-09 | CHƯA CHẠY |  |
| TC-04-10 | CHƯA CHẠY |  |
| TC-04-11 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-12 | CHƯA CHẠY |  |
| TC-04-13 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-14 | CHƯA CHẠY |  |
| TC-04-15 | CHƯA CHẠY |  |
| TC-04-16 | CHƯA CHẠY |  |
| TC-04-17 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-P1 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-P2 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-P3 | CHƯA CHẠY |  |
| TC-04-P4 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-P5 | PASS | API stack (04-05: L10 mã bulk) |
| TC-04-06b | CHƯA CHẠY |  |
| TC-05-01 | PASS | API stack |
| TC-05-02 | PASS | API stack |
| TC-05-03 | PASS | API stack |
| TC-05-04 | FAIL | L8 |
| TC-05-05 | PASS | 0 trùng với câu vừa làm đúng (lùi 10 ngày bằng DB) |
| TC-05-06 | CHƯA CHẠY |  |
| TC-05-07 | CHƯA CHẠY |  |
| TC-05-08 | PASS | API stack |
| TC-05-09 | CHƯA CHẠY |  |
| TC-05-10 | PASS | API stack |
| TC-05-11 | PASS | API stack |
| TC-05-12 | PASS | API stack |
| TC-05-13 | PASS | API stack |
| TC-05-14 | PASS | API stack |
| TC-05-15 | PASS | API stack |
| TC-05-16 | FAIL | L6 count=0 → 201; L7 chủ đề 0 câu đủ điều kiện → 422 |
| TC-05-17 | PASS | API stack |
| TC-05-18 | CHƯA CHẠY |  |
| TC-06-01 | CHƯA CHẠY |  |
| TC-06-02 | CHƯA CHẠY |  |
| TC-06-03 | PASS | API stack |
| TC-06-04 | PASS | API stack |
| TC-06-05 | PASS | API stack |
| TC-06-06 | PASS | API stack |
| TC-06-07 | CHƯA CHẠY |  |
| TC-06-08 | PASS | API stack |
| TC-06-09 | PASS | API stack |
| TC-06-10 | CHƯA CHẠY |  |
| TC-06-11 | PASS | API stack |
| TC-06-12 | PASS | API stack |
| TC-06-13 | CHƯA CHẠY |  |
| TC-06-14 | CHƯA CHẠY |  |
| TC-06-15 | CHƯA CHẠY |  |
| TC-06-16 | PASS | API stack |
| TC-06-17 | PASS | API stack |
| TC-06-18 | PASS | API stack |
| TC-06-19 | PASS | API stack |
| TC-06-20 | PASS | API stack |
| TC-06-21 | CHƯA CHẠY |  |
| TC-06-22 | PASS | API stack |
| TC-06-23 | FAIL | L15: chữ khác TC (hai dòng + liên kết lịch sử) |
| TC-06-24 | CHƯA CHẠY |  |
| TC-06-25 | CHƯA CHẠY |  |
| TC-06-25b | PASS | API stack |
| TC-06-26 | CHƯA CHẠY |  |
| TC-07-01 | PASS | API stack |
| TC-07-02 | PASS | API stack |
| TC-07-03 | CHƯA CHẠY |  |
| TC-07-04 | CHƯA CHẠY |  |
| TC-07-05 | CHƯA CHẠY |  |
| TC-07-06 | PASS | API stack |
| TC-07-07 | PASS | API stack |
| TC-07-08 | PASS | API stack |
| TC-07-09 | CHƯA CHẠY |  |
| TC-07-10 | CHƯA CHẠY |  |
| TC-07-11 | PASS | API stack |
| TC-07-11b | PASS | API stack |
| TC-07-11c | PASS | API stack |
| TC-07-12 | CHƯA CHẠY |  |
| TC-07-13 | PASS | API stack |
| TC-07-14 | PASS | API stack |
| TC-07-15 | CHƯA CHẠY |  |
| TC-07-16 | PASS | API stack |
| TC-07-16b | PASS | API stack |
| TC-07-17 | PASS | API stack |
| TC-07-18 | PASS | API stack |
| TC-07-19 | PASS | API stack |
| TC-07-19b | PASS | API stack |
| TC-07-20 | PASS | API stack |
| TC-07-21 | CHƯA CHẠY |  |
| TC-07-22 | CHƯA CHẠY |  |
| TC-07-23 | CHƯA CHẠY |  |
| TC-07-24 | PASS | API stack |
| TC-07-25 | FAIL | HOÃN theo chủ đề dự án (k6 / Lighthouse / axe) |
| TC-07-26 | CHƯA CHẠY |  |
| TC-08-01 | PASS | API stack |
| TC-08-02 | PASS | API stack |
| TC-08-03 | PASS | 10 lần feedback = 10 dòng llm_audit task GRADING lane BATCH (quan sát: lane BATCH cho việc SV chờ) |
| TC-08-04 | PASS | API stack |
| TC-08-05 | PASS | API stack |
| TC-08-06 | PASS | API stack |
| TC-08-07 | PASS | API stack |
| TC-08-08 | NGOÀI PHẠM VI | P-10-7 Q16; dev xác nhận không có cách bật (handoff) |
| TC-08-09 | PASS | API stack |
| TC-08-10 | PASS | API stack |
| TC-08-11 | CHƯA CHẠY |  |
| TC-08-12 | PASS | API stack |
| TC-08-13 | CHƯA CHẠY |  |
| TC-08-14 | PASS | API stack |
| TC-09-01 | PASS | API stack |
| TC-09-02 | CHƯA CHẠY |  |
| TC-09-03 | PASS | API stack |
| TC-09-04 | PASS | API stack |
| TC-09-05 | CHƯA CHẠY |  |
| TC-09-06 | FAIL | L3: cache Hôm nay theo lớp không bị huỷ |
| TC-09-07 | FAIL | L3: cache Hôm nay theo lớp không bị huỷ |
| TC-09-08 | FAIL | L3: cache Hôm nay theo lớp không bị huỷ |
| TC-09-09 | PASS | API stack |
| TC-09-10 | PASS (không stack) | `TestWrongAnswerSignalsNoIdentity` PASS |
| TC-09-11 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestWrongAnswerSignalsNoIdentity` PASS (k ≥ 3, không định danh) |
| TC-09-12 | CHƯA CHẠY |  |
| TC-09-13 | CHƯA CHẠY |  |
| TC-09-14 | PASS | API stack |
| TC-09-15 | CHƯA CHẠY |  |
| TC-09-16 | CHƯA CHẠY |  |
| TC-09-17 | PASS | API stack |
| TC-09-18 | CHƯA CHẠY |  |
| TC-09-19 | CHƯA CHẠY |  |
| TC-09-20 | CHƯA CHẠY |  |
| TC-09-21 | PASS | API stack |
| TC-10-01 | PASS | Go |
| TC-10-02 | PASS | API stack |
| TC-10-03 | PASS | Go |
| TC-10-04 | PASS | API stack |
| TC-10-05 | PASS | API stack |
| TC-10-06 | CHƯA CHẠY |  |
| TC-10-07 | PASS | API stack |
| TC-10-08 | PASS | API stack |
| TC-10-09 | CHƯA CHẠY |  |
| TC-10-10 | CHƯA CHẠY |  |
| TC-10-11 | PASS | Go |
| TC-10-12 | PASS | Go |
| TC-10-13 | PASS | API stack |
| TC-10-14 | PASS | API stack |
| TC-10-15 | PASS | API stack |
| TC-10-16 | CHƯA CHẠY |  |
| TC-10-17 | PASS | API stack |
| TC-11-01 | PASS | API stack |
| TC-11-02 | FAIL | L12: ánh xạ theo tên cột (ví dụ trong openapi) bị từ chối |
| TC-11-03 | PASS | API stack |
| TC-11-04 | CHƯA CHẠY |  |
| TC-11-05 | PASS | API stack |
| TC-11-06 | PASS | API stack |
| TC-11-07 | PASS | API stack |
| TC-11-08 | PASS | API stack |
| TC-11-09 | PASS | API stack |
| TC-11-10 | CHƯA CHẠY |  |
| TC-11-11 | PASS | API stack |
| TC-11-12 | PASS | API stack |
| TC-11-13 | PASS | log 0 PII; `audit_log` chỉ có hash / số khớp, chưa khớp |
| TC-11-14 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestFormsImportNoLLM`, `TestFormsImportScoreDecimal` PASS; `llm_audit` trên stack chờ stack |
| TC-11-15 | CHƯA CHẠY |  |
| TC-11-16 | CHƯA CHẠY |  |
| TC-11-17 | PASS | `audit_log forms_imports/import` |
| TC-11-18 | CHƯA CHẠY |  |
| TC-11-19 | CHƯA CHẠY |  |
| TC-12-01 | PASS | `seed.mjs` trên stack trống: mã 0, 334 s (kể cả docling) |
| TC-12-02 | FAIL | 104 APPROVED, 14 chủ đề, đủ độ khó; nhưng 0 câu có citations, không có EXTRACTED trong PENDING (L4) |
| TC-12-03 | FAIL | 42 lượt của 13 SV (đạt); Quiz `SCHEDULED` thay vì `PUBLISHED` (L4c) |
| TC-12-04 | FAIL | L4d: lớp 761988 có 0 câu trong ngân hàng (spec: ngân hàng nhỏ khác) |
| TC-12-05 | PASS | script đã sửa ở `5d9c54e3`: `OK: 107 APPROVED, 26 PENDING…` (bản `57f4cb25` lỗi, L5) |
| TC-12-06 | PASS | chạy lại: mã 0, 21 s; `question_bank` 129, `practice_attempts` 42, `exams` 3 không đổi |
| TC-12-07 | PASS | Go |
| TC-12-08 | PASS | Go |
| TC-12-09 | PASS | 52 passed (e2e dev, gateway giả) |
| TC-12-10 | CHƯA CHẠY |  |
| TC-12-11 | CHƯA CHẠY |  |
| TC-12-12 | PASS | mock/practice.ts, mock/questions.ts không còn; antipatterns sạch |
| TC-12-13 | CHƯA CHẠY |  |
| TC-12-14 | PASS | contract test xanh (4831150a); 429 ở explain/extract/generate có trong openapi |
| TC-12-15 | CHƯA CHẠY |  |
| TC-12-16 | CHƯA CHẠY |  |
| TC-12-17 | FAIL | HOÃN theo chủ đề dự án (k6 / Lighthouse / axe) |
| TC-12-18 | CHƯA CHẠY |  |
| TC-12-19 | FAIL | HOÃN theo chủ đề dự án (k6 / Lighthouse / axe) |
| TC-12-20 | CHƯA CHẠY |  |
| TC-12-21 | CHƯA CHẠY |  |
| TC-12-22 | PASS có điều kiện; chờ dev | F1 chấp nhận (P-10-8): ngưỡng Hôm nay SV 6 → 7 (1 truy vấn gộp cố định, không N+1). Còn lại sạch (không SKIP / secret / float64 / fetch trần / thư viện mới). **Điều kiện:** commit test phải ghi số P-10-8 — QC kiểm lại khi dev sửa |
