# QC test case — US-P9-05 Dựng đề theo ma trận chủ đề × độ khó
Nguồn: US-P9-05 + SRS FR-2, 3, 21, §6 #6 (dựng đề qua `POST practice/start`; QC quan sát đề qua `GET practice/attempts/{aid}`). Seed: lớp 1 có 80 `APPROVED` (≥ 6 chủ đề, 3 độ khó), 20 `PENDING`; bài QUIZ = bài thi PE `PUBLISHED`.
Trợ giúp: `ITEMS(aid)` = `GET /courses/$C1/practice/attempts/$aid | jq '.items[]'` (id câu, `position`; tên trường theo openapi). Muốn biết nhãn thật của câu: `GET …/questions/{id}` bằng GV.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-05-01 | AC1 | Seed | `go test ./internal/quiz -run TestBuildMatrix -v` | PASS | GO |
| TC-05-02 | AC1 | SV `sv.kha`, chủ đề đủ ≥ 20 câu `APPROVED` (hoặc ma trận trộn nhiều chủ đề, `mode=MOCK`) | `start {mode:MOCK,count:20,matrix:{…đủ câu…}}` | `201`; đề đúng 20 câu, không câu trùng (`question_id` phân biệt), số câu mỗi ô khớp ma trận (± 0) | API |
| TC-05-03 | AC1 | Seed | Dựng 100 lần (mỗi lần dùng SV hoặc kết thúc lượt trước bằng `submit` rồi dựng lại), đo `time_total` | p95 ≤ 3 s (ghi p50/p95/max; «MacBook Apple Silicon, Colima 6 CPU / 10 GB») | API |
| TC-05-04 | AC1 | Ma trận ô thiếu (ô X xin 6 câu chỉ có 3) | `start` | Bù từ ô gần nhất theo độ khó rồi chủ đề; tổng vẫn đủ nếu bù được; `notes` mô tả việc bù | API |
| TC-05-05 | AC2 | `sv.kha` làm lượt A chủ đề T: đúng 5 câu, sai 5 câu; ép `started_at` các lượt này về 10 ngày trước (ghi DB, stack kiểm) | `start TOPIC T count 10` (sau khi lượt A `submit`) | Ưu tiên câu chưa làm hoặc làm sai; câu vừa làm **đúng** không xuất hiện nếu còn câu khác (T có ≥ 20 câu: 0 trùng với 5 câu đúng) | API + DB |
| TC-05-06 | AC2 | Cùng, nhưng lượt A cách đây 40 ngày (ghi DB) | `start` lại | Cửa sổ 30 ngày: câu làm đúng cách 40 ngày không còn bị ưu tiên loại (có thể xuất hiện); ghi quan sát | API + DB |
| TC-05-07 | AC3 | Bài thi E1 `DRAFT`, E2 `SCHEDULED`, E3 `OPEN`, E4 `CLOSED` chưa công bố, E5 `PUBLISHED`, E6 đã xoá mềm; mỗi bài chứa câu riêng X1…X6 (GV tạo qua API; đi đúng quy trình thật bằng API, chỉ ép trạng thái bằng DB khi cần và ghi rõ — mặc định QC được PM chấp nhận, P-10-7 Q9) | Dựng đề luyện bất kỳ chủ đề chứa X1…X6, nhiều lần (≥ 30 lần/chủ đề, SV khác nhau) | X1, X2, X3, X4, X6 **không bao giờ** xuất hiện; X5 (`PUBLISHED`) được phép xuất hiện; kiểm cả `explain` / phản hồi sau khi trả lời không nhắc X1…X4 | API |
| TC-05-08 | AC3 | Câu ở `PENDING`, `DRAFT`, `REJECTED`, `archived`, `CODE`, `ESSAY` (chế độ không bật tự luận) trong cùng chủ đề | Dựng đề 30 lần | 0 lần xuất hiện | API |
| TC-05-09 | AC3 | `exam_items` đổi trạng thái | Đưa E3 `OPEN` → `CLOSED` → `PUBLISHED` (GV) rồi dựng lại | Câu X3 chỉ xuất hiện sau `PUBLISHED` (theo `TestPracticeNoExamLeak` 6 trạng thái; chạy `go test … -run TestPracticeNoExamLeak -v` ở US-P9-10) | API |
| TC-05-10 | AC4 | Chủ đề «Ít» có đúng 4 câu đủ điều kiện + đề xin 10 | Chủ đề «Ít» có đúng 6 câu, `count=10` | `201`, đề **6 câu**, `notes` chứa «thiếu câu chủ đề Ít» (hoặc tương đương), không 5xx | API |
| TC-05-11 | AC4 | Chủ đề «Rất ít» 4 câu đủ điều kiện | `start TOPIC «Rất ít»` | `409 NOT_ENOUGH_QUESTIONS` với `{available:4}`; UI «Chưa đủ câu để luyện chủ đề này» | API + E2E |
| TC-05-12 | AC5 | `sv.kha`, `sv.gioi` cùng chủ đề T | Mỗi người `start`; so thứ tự câu (position→question_id) và thứ tự phương án | Thứ tự **khác nhau** giữa hai lượt (xác suất; thử 3 cặp SV, ít nhất 2 cặp khác); id phương án gốc giữ nguyên (`option_id` trùng giữa hai SV cho cùng câu) | API |
| TC-05-13 | AC5 | Lượt đang dở | `GET attempt` 3 lần + tải lại trang (E2E) | Thứ tự câu và phương án **không đổi** | API + E2E |
| TC-05-14 | AC6 | SV lớp 1 | `POST /courses/$C2/practice/start` | `403` hoặc `404` | API |
| TC-05-15 | AC6 | SV có `enrollments.status != ACTIVE` (rời lớp; dev tạo qua `DELETE …/enrollments` hoặc DB) | `start` ở lớp đó | `403` hoặc `404` | API |
| TC-05-16 | AC7 | – | `count=4`; `count=51`; `count=0`; `count="abc"`; `topic` hoàn toàn lạ (chủ đề có trong ngân hàng nhưng 0 câu đủ điều kiện → `409 NOT_ENOUGH_QUESTIONS` `{available:0}`, P-10-7 Q10) | `422 VALIDATION_FAILED` mỗi lần (chủ đề có trong ngân hàng nhưng không còn câu đủ điều kiện → `409`, P-10-7 Q10) | API |
| TC-05-17 | AC1 | – | Biên: `count=5`, `count=50` (cần ≥ 50 câu: dùng ma trận nhiều chủ đề của lớp 1) | `201` cho cả hai (hoặc đề ngắn + `notes` nếu thiếu) | API |
| TC-05-18 | FR-21 | – | (DIFF pha 2) đọc truy vấn dựng đề | Phép loại câu thuộc bài thi chưa `PUBLISHED` nằm ở **truy vấn SQL** (không lọc ở giao diện hay sau khi lấy hết) | DIFF |

## Phân quyền
TC-05-14, TC-05-15; staff `POST …/practice/start` → `403` (`reason: role`); SV không thuộc lớp (sv.moi) → `403`/`404`.

## Lịch sử sửa TC
- 2026-11-14 — spec FEAT-practice v1.1 + PM trả lời P-10-7 (Q-S10-1…26): TC-05-07, TC-05-16 (ghi mặc định QC được PM chấp nhận: Q9, Q10).
- 2026-11-14 — P-10-9 (PM chấp nhận): TC-05-04 và TC-05-10 theo AC1/AC4 mới: ô thiếu câu → bù từ chủ đề khác cùng độ khó, rồi mới cắt ngắn kèm `notes`; chỉ cắt ngắn khi cả ngân hàng hết câu đủ điều kiện. Ma trận không có `difficulty` cũng phải bù (L8).
