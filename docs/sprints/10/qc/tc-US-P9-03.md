# QC test case — US-P9-03 Sinh câu hỏi mới từ đoạn tài liệu (Could, cắt đầu tiên)
Nguồn: US-P9-03 + SRS FR-6, §6 #2, #4. **Chỉ chạy nếu handoff ghi đã làm.** Không làm → «NGOÀI PHẠM VI SPRINT» (spec: bỏ được). Khi đó TC-04-08 (nút `Sinh từ tài liệu` ẩn) và TC-12 xác nhận route/UX không lộ nửa vời.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-03-01 | AC1 | `LECTURE` `READY` L1, provider `fake` | `POST …/questions/generate {"document_ids":["<L1>"],"topic":"Mật mã","difficulty":"MEDIUM","count":5,"types":["MCQ_SINGLE"]}` | `202 {job_id}`; sau khi xong: câu `origin=GENERATED`, `review_status=PENDING` | API |
| TC-03-02 | AC2 | TC-03-01 xong | Mỗi câu: đọc `citations` | Mỗi câu có ≥ 1 `{document_id, chunk_id, page}` thật (chunk tồn tại, thuộc lớp, trang đúng); từ khoá của đáp án đúng / giải thích có trong chunk đó (QC kiểm bằng `grep` văn bản chunk) | API + DB |
| TC-03-03 | AC2 | `fake_llm.json` có câu không có căn cứ trong chunk | Sinh | Câu đó bị loại, đếm vào «bỏ qua» | API |
| TC-03-04 | AC3 | `ANSWER_KEY` | Sinh với `document_ids=[<K>]` | `422` | API |
| TC-03-05 | AC3 | `LECTURE` `visible_to_students=false` | Staff sinh từ tài liệu đó | `202`; câu sinh `PENDING` không hiện cho SV (`GET /practice/overview`); sau `APPROVED` hiện ở luyện đề; liên kết nguồn tới tài liệu ẩn **không bấm mở được** ở SV, chỉ chữ «Nguồn trong tài liệu» (Q-P9-9) | API + E2E |
| TC-03-06 | AC3 | – | Đọc `llm_audit` payload | Chỉ gồm chủ đề, độ khó, số câu, nội dung chunk đã chọn; qua lớp che | DB |
| TC-03-07 | AC4 | – | `count=11`; `types=[]`; `document_ids` của lớp 2 | `422 VALIDATION_FAILED` ×3 (lớp 2: `422`, không lộ tồn tại) | API |
| TC-03-08 | AC4 | `OVERLOADED` | Sinh | Như TC-02-14 | API |
| TC-03-09 | AC5 | Đã có câu `stem` X (ở `REJECTED`, `APPROVED`, `PENDING`) | Sinh ra câu trùng X sau chuẩn hoá (hoa/thường, dấu cách, số thứ tự đầu) | Bỏ qua «đã có» ở mọi trạng thái kể cả `REJECTED` | API |
| TC-03-10 | AC6 | – | STUDENT gọi `generate` | `403` | API |
| TC-03-11 | AC7 | Lô 5 câu: duyệt 2 nguyên (`APPROVE` không sửa), 1 sửa rồi `APPROVE`, 2 `REJECT` | `GET …/questions/gen-stats?job_id=` | `{unchanged:2, edited:1, rejected:2}` | API |
| TC-03-12 | AC7 | – | `gen-stats` với `job_id` lạ / lớp khác | `404` | API |
| TC-03-13 | SRS Q-P9-12 | – | 6 việc `generate` trong 1 giờ | Việc thứ 6 `429 RATE_LIMITED` + `Retry-After` | API |

## Phân quyền
STUDENT → `403` (TC-03-10); TA → `202`; GV lớp khác → `403`; thiếu `Idempotency-Key` → lỗi.

## Lịch sử sửa TC
(trống)
