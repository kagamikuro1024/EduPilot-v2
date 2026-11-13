# QC test case — US-P9-02 Trích câu hỏi từ đề cũ (+ ghép đáp án)
Nguồn: US-P9-02 + SRS §3 (nhánh trích), §4.2 FR-5, 7, 8, §6 #1. Cần docling (`--profile ingest`), tài liệu `de-mau-trac-nghiem.pdf` (24 câu A–D), `dap-an-mau.pdf` do dev cung cấp ở `backend-go/internal/practice/testdata/` (dev bàn giao fixture, P-10-7 Q4; thiếu → «chờ handoff»). Tải lên qua `uploads/presign` + `complete` loại `EXAM_PAPER` / `ANSWER_KEY`, chờ `READY`.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-02-01 | AC1 | `EXAM_PAPER` `READY` (đề mẫu) | `IDEM=$(uuidgen) api teacher POST /courses/$C1/questions/extract '{"document_id":"<D>"}'`; mở SSE `job.progress` (`GET /events` hoặc kênh theo openapi) | `202 {job_id}`; nhận ≥ 2 sự kiện `job.progress` với tiến độ tăng, kết thúc ở hoàn tất; nội dung không có PII | API |
| TC-02-02 | AC2 | TC-02-01 xong, provider `fake` | `GET /courses/$C1/questions?origin=EXTRACTED&review_status=PENDING&limit=100` | ≥ 20 câu; mỗi câu `stem` không rỗng, ≥ 2 phương án, đúng 1 hoặc nhiều đáp án đúng đã đánh dấu (không câu nào thiếu `answer_key`), `citations[]` có `document_id=<D>` và `page` số nguyên ≥ 1 | API |
| TC-02-03 | AC2 | – | Lấy danh sách «bỏ qua» của việc (API đọc danh sách: **chờ handoff + openapi**, P-10-7 Q5) | Có mục từng câu bỏ qua kèm trang và lý do «không tìm thấy đáp án»; số lưu + số bỏ qua = 24 (hoặc số câu đề mẫu; ghi số thật) | API |
| TC-02-04 | AC2 | – | `go test ./internal/practice -run TestExtractQuestions -v` | PASS | GO |
| TC-02-05 | AC3 | Thêm `ANSWER_KEY` `dap-an-mau.pdf` `READY` | Trích với `{"document_id":"<D>","answer_key_document_id":"<K>"}` (cùng đề, DB sạch câu trích) | Đáp án đúng của từng câu = đáp án trong `dap-an-mau.pdf` theo **số câu** (đối chiếu tay 24 cặp: kết quả khớp 100% với bảng đáp án); câu số không có trong đáp án → vào «bỏ qua» có lý do, không lưu | API |
| TC-02-06 | AC3 | TC-02-05 | `llm_audit` của `batch_id` việc trích: đọc payload đã che (ADMIN, ghi `audit_log`) | `ANSWER_KEY` chỉ vào prompt trích vì người gọi là staff; 0 lời gọi LLM **kênh sinh viên** chứa chuỗi đáp án (`grep` một đoạn lạ của `dap-an-mau.pdf` trong mọi `llm_audit` task chat / practice explain của SV: 0 dòng) | DB |
| TC-02-07 | AC4 | Provider `fake` có bản ghi lỗi: đầu ra thiếu `stem`; có 1 phương án; `answer_label` trỏ nhãn không tồn tại; sai schema (thiếu `questions`) | Chạy trích (dev bàn giao `fake_llm.json` có bản ghi lỗi schema, P-10-7 Q6) | Mỗi mục lỗi **không** vào `question_bank` (đếm), vào «bỏ qua» với lý do; việc vẫn hoàn tất cho các câu hợp lệ | API |
| TC-02-08 | AC5 | TC-02-01 đã có | Gửi lại **cùng** `Idempotency-Key` và `document_id` | Cùng `job_id`; số câu `EXTRACTED` không đổi | API |
| TC-02-09 | AC5 | TC-02-01 đã có | Gửi lại **khoá mới** cùng tài liệu | `202` job mới; xong: số câu `EXTRACTED` không tăng (trùng `stem` chuẩn hoá bị bỏ qua), «bỏ qua» ghi «đã có» cho từng câu | API |
| TC-02-10 | AC6 | Tài liệu `PROCESSING` (tải mới, chưa READY) | Trích | `409 DOCUMENT_NOT_READY` | API |
| TC-02-11 | AC6 | Tài liệu loại `LECTURE` | Trích | `422 WRONG_DOCUMENT_TYPE` | API |
| TC-02-12 | AC6 | `EXAM_PAPER` của lớp 2 | GV lớp 1 trích với `document_id` đó | `404` (không lộ tồn tại, không `403`) | API |
| TC-02-13 | AC6 | `answer_key_document_id` thuộc loại khác / lớp khác | Trích | `422 WRONG_DOCUMENT_TYPE` hoặc `404` tương ứng (mặc định QC theo P-10-7 Q7: sai loại `422 WRONG_DOCUMENT_TYPE`, lớp khác `404`; mã thật **chờ handoff + openapi**) | API |
| TC-02-14 | AC6 | `fake_llm.json` trả `OVERLOADED` | Trích | Retry đúng 3 lần (đếm trong `llm_audit` / log), rồi dead-letter (thấy ở `/admin` hoặc bảng hàng đợi); GV thấy «Chưa trích được. Thử lại.» ở luồng `Tạo câu hỏi` (E2E); `question_bank` không có câu nửa vời | API + E2E |
| TC-02-15 | AC6 | Đề nhiều trang, provider `fake` | Dừng mạng giữa việc: `docker compose -p ep-qc restart worker` giữa lúc chạy | Câu đã lưu giữ nguyên; việc chạy tiếp từ trang chưa xong khi worker lên; không câu trùng sau khi xong | API |
| TC-02-16 | AC8 | Đề mẫu có dòng «Người ra đề: Nguyễn Văn Test» (dev chèn) | Trích; đọc `llm_audit` payload (ADMIN) và `stem` đã lưu | Payload không có tên; `stem` đã lưu **không** chứa placeholder rò, chỉ chữ của đề (không thêm tên người) | DB + API |
| TC-02-17 | (SRS §3, ngoài phạm vi AC) | Đề ảnh quét, docling trả rỗng | Trích | Thông báo «Không đọc được văn bản», không câu nào lưu (spec: ngoài phạm vi story; kiểm như thông tin) | API |
| TC-02-18 | AC1 | k6 chat 1 QPS + extract (chỉ khi PM báo dev hết đo tải 9.5; chạy cuối pha 2, P-10-7 Q8) | `benchmarks/load/…` kịch bản chat + extract | TTFT chat p95 xấu đi ≤ 20% so với nền | GO/k6 |
| TC-02-19 | SRS Q-P9-12 | 5 việc `extract` / lớp / giờ | Gửi 6 việc trong 1 giờ | Việc thứ 6 → `429 RATE_LIMITED` + `Retry-After` | API |
| TC-02-20 | SRS Q-P9-12 | Tài liệu > 60 trang | Trích | Giới hạn ≤ 60 trang (chỉ trích 60 hoặc báo rõ; ghi hành vi) | API |

## Phân quyền
| TC | Bước | Kết quả |
| --- | --- | --- |
| TC-02-P1 | STUDENT `POST …/questions/extract` | `403` (`reason: role`) |
| TC-02-P2 | ADMIN không thuộc lớp | `403` (`reason: role` hoặc `course`) |
| TC-02-P3 | TA `POST …/questions/extract` | `202` |
| TC-02-P4 | Thiếu `Idempotency-Key` | `400`/`422` (ghi mã thật) |
| TC-02-P5 | GV lớp 2 gọi trên `C1` | `403` |

## Lịch sử sửa TC
- 2026-11-14 — spec FEAT-practice v1.1 + PM trả lời P-10-7 (Q-S10-1…26): TC-02 ghi «chờ handoff» cho Q4, Q5, Q6, Q7; Q8 (k6 cuối pha 2).
