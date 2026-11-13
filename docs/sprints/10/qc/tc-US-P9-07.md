# QC test case — US-P9-07 Thi thử có giờ, tự lưu, ngoại tuyến
Nguồn: US-P9-07 + SRS FR-12–17, §3, §6 #6–#10. Giờ: `deadline_at` do máy chủ. Để thử hết giờ nhanh, dùng `duration_minutes=10` (tối thiểu theo SRS) rồi **ép** `deadline_at` về quá khứ (ghi DB, stack kiểm; ghi rõ là cách ép) hoặc chờ 10 phút.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-07-01 | AC1 | `sv.kha` | `start {"mode":"MOCK","count":20,"duration_minutes":30}` (mặc định thử: bỏ `count`, `duration_minutes`) | `201`; `deadline_at ≈ now()+30 phút` do máy chủ (sai lệch ≤ 2 s so với giờ máy chủ); mặc định `count=20`, `duration=30` | API |
| TC-07-02 | AC1 | Còn lượt MOCK dở | `start` MOCK lần nữa (khoá mới) | `200` cùng `attempt_id` (không lượt thứ hai); DB đếm 1 lượt `IN_PROGRESS` | API + DB |
| TC-07-03 | AC1 | – | `duration_minutes=9` và `181`; `count=4`, `51` | `422 VALIDATION_FAILED` | API |
| TC-07-04 | AC2 | Lượt MOCK | Mở màn làm bài 1440 và 375; đặt đồng hồ trình duyệt lệch +5 phút (`page.clock` / chỉnh `Date`) | Đồng hồ hiển thị theo máy chủ (`server_now` bù lệch; Q-S10-12): chênh với `deadline_at − server_now` ≤ 2 s dù máy lệch; chữ số cố định bề rộng (đo bề rộng ô đồng hồ không đổi khi giây đổi `0↔9`); **không** phản hồi đúng/sai, **không** nút `Giải thích thêm` | E2E |
| TC-07-05 | AC3 | Lượt MOCK | Chọn đáp án, chuyển câu; hoặc chọn rồi đợi 1 s | Có `PUT …/answers/{itemId}` kèm `version` và `X-Practice-Tab`; lưu lần theo thay đổi; hiển thị «Đã lưu» 1–2 từ | E2E |
| TC-07-06 | AC3 | Lượt mở ở tab 1 | Mở cùng lượt ở tab 2 (cùng phiên đăng nhập); ở tab 1 chọn đáp án | Tab 1 nhận `409 TAB_TAKEN_OVER`, chuyển **chỉ đọc**, có nút `Làm tiếp ở tab này`; bấm nút ở tab 1 → tab 2 bị chiếm lại tương tự; không mất câu đã lưu | E2E |
| TC-07-07 | AC3 | – | `PUT answers` với `version` cũ | `409 VERSION_CONFLICT` | API |
| TC-07-08 | AC3 | MOCK | `PUT` `option_id` không thuộc câu; `question_id` lớp khác | `422 INVALID_OPTION_ID`; `422 INVALID_QUESTION` | API |
| TC-07-09 | AC4 | Lượt MOCK | `setOffline(true)`; chọn 5 câu; | Thanh dưới hiện «Chưa gửi được 5 câu» (đếm đúng, không chặn làm bài, vẫn chuyển câu được); lưu cục bộ ở IndexedDB (kiểm `indexedDB` có 5 mục) | E2E |
| TC-07-10 | AC4 | TC-07-09 | `setOffline(false)` | Gửi lại theo **thứ tự** chọn; sau thành công thanh dưới biến mất; DB `practice_answers` đủ 5; không mất câu | E2E + DB |
| TC-07-11 | AC4 | Mất mạng khi hết giờ (đặt `deadline_at` quá khứ **nhưng trong 60 s dung sai** hoặc thiết kế E2E với đồng hồ giả) | Chọn câu lúc còn giờ (offline, `client_ts ≤ deadline_at`); hết giờ; bật mạng | Bài tự động nộp khi có mạng; lựa chọn có `client_ts ≤ deadline_at + 60 s` được tính; lựa chọn có `client_ts > deadline_at+60 s` bị bỏ, `notes` ghi; (dung sai/`≤ deadline_at`: Q-S10-13) | E2E |
| TC-07-12 | AC5 | Còn 5 phút | Đặt đồng hồ chạm 5:00 | Cảnh báo **một lần**, tại chỗ (không modal), đỏ; tải lại khi còn 4:00 → không hiện lại | E2E |
| TC-07-13 | AC6 | Đang câu 8 | `PATCH …/attempts/{aid} {"current_index":7}` (qua UI: chuyển câu); tải lại | Quay đúng câu 8; đồng hồ **không đặt lại** (đếm tiếp theo `deadline_at`); mọi lựa chọn đã lưu còn | E2E + API |
| TC-07-14 | AC6 | – | `PATCH` `current_index` ngoài `[0,count-1]` | `422`/`409` (ghi mã) | API |
| TC-07-15 | AC7 | Lượt MOCK đã chọn hết | Bấm `Nộp bài` | Hộp xác nhận **một lần** (`ConfirmIrreversible`, nêu không đảo ngược); xác nhận → `POST …/submit` | E2E |
| TC-07-16 | AC7 | Lượt MOCK 20 câu: QC tính tay trước đáp án (chọn đúng X câu) | `submit` | Kết quả chính `{đúng}/{tổng}` = giá trị tính tay; điểm thang 10 **một chữ số thập phân** phụ cạnh bên (`round(đúng/tổng×10, 1)`; ví dụ 13/20 → `6,5`; 7/20 → `3,5`; 1/3 → `3,3` — `decimal`, không lỗi dấu phẩy động, kiểm `0.1+0.2` kiểu: 3/7 → `4,3`); từng câu đúng/sai + đáp án + giải thích + trích dẫn; phân tích theo chủ đề; chữ «điểm» chỉ ở giá trị phụ; không xếp hạng / so sánh | API + E2E |
| TC-07-16b | AC7 | `count=8` (hoặc ma trận 8 câu), QC chọn đúng 5 | `submit` | `5/8`; điểm phụ **6,3** nếu làm tròn nửa lên, **6,2** nếu half-even (Q-S10-14: spec chỉ nói «một chữ số thập phân bằng `decimal`»; chạy theo nửa lên, kết quả khác ghi vào câu hỏi) | API |
| TC-07-17 | AC7 | – | Gửi lại `submit` (cùng khoá; khoá khác) | Cùng kết quả; `correct_count`, `score`, `submitted_at` không đổi (DB) | API + DB |
| TC-07-18 | AC7 | – | `DB`: so `grade_entries` / `grade_items` trước và sau `submit` | Không đổi (điểm luyện không vào sổ điểm); `GET /gradebook` không có cột/dòng luyện đề | DB |
| TC-07-19 | AC8 | Lượt MOCK, ép `deadline_at` về quá khứ | `PUT answers` mới | `409 ATTEMPT_CLOSED`; nếu `deadline_at` quá nhưng chưa quá +60 s xem Q-S10-13 | API |
| TC-07-20 | AC8 | Lượt MOCK quá `deadline_at + 60 s`, SV không nộp | Đợi worker `practice.close` (tick 1 phút) | Lượt `EXPIRED`, có `correct_count` chấm trên câu đã lưu; mở lại màn → thấy kết quả; chạy tick lần nữa (hoặc 2 worker) không chấm lại/đổi số liệu (idempotent) | API + DB |
| TC-07-21 | AC9 | Nộp khi mạng đứt | `setOffline(true)` → `Nộp bài` → xác nhận | Thanh dưới «Chưa nộp được. Đang thử lại.»; bật mạng → tự gửi lại với **cùng** `Idempotency-Key` (bắt request); kết quả đúng; không lượt chấm hai lần | E2E |
| TC-07-22 | AC9 | – | Trong lúc TC-07-21, đóng tab và mở lại | Nội dung bài không mất (IndexedDB); gửi lại được | E2E |
| TC-07-23 | AC10 | – | Rời tab (`visibilitychange`), dán (`paste`) trong lúc thi thử | Không có `POST` ghi nhật ký; `exam_events` không có dòng mới; **chat riêng không khoá** (`GET /me/exam-lock` không đổi, gửi chat nội dung môn học được trả lời) | E2E + DB |
| TC-07-24 | AC11 | Lượt của A | `sv.gioi` mọi API của lượt; SV lớp khác; staff gọi `PUT answers` | `404` / `404` / `403` | API |
| TC-07-25 | AC12 | `benchmarks/load/practice-mock.js` có trong repo | 100 SV mô phỏng, provider `fake` (chỉ khi stack được phép đo tải; Q-S10-8) | `PUT answers` p95 ≤ 500 ms; `submit` p95 ≤ 1 s; 0 lượt mất câu đã lưu (đối chiếu số `PUT` thành công = số `practice_answers`) | GO/k6 |
| TC-07-26 | UX | 375×812 | Làm lượt MOCK 20 câu | Không tràn ngang; nút ≥ 44 px; thanh dưới cố định; đồng hồ không nhảy layout | E2E |

## Phân quyền
TC-07-24; staff `start` MOCK → `403`; `sv.moi` → `403`/`404`.

## Lịch sử sửa TC
(trống)
