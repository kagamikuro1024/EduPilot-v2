# QC test case — US-P4-05 (vòng đời phía sinh viên, SLA, tri thức, ngưỡng)
Nguồn: `US.md` US-P4-05 AC1…AC13 + `SRS.md` 2, 3.1, 3.3, 4.6, 4.7, 4.8, 6.1–6.3, 9.3, 9.4; `QUESTIONS.md` Q1, Q2, Q3, Q4 (cả ngày T2–T6), Q20 (không cache theo độ tương đồng); `proposals.md` #7 (`gateway admin seed-age` chỉ ngoài production). Hộp đen, không đọc code dev.
Biến và quy ước như `tc-US-P4-01.md`. **Đồng hồ:** stack chạy thật không có đồng hồ giả (chỉ có trong test Go). QC lùi thời gian bằng lệnh `gateway admin seed-age` (proposals #7; cú pháp ghi `Q-QC-P4-13`) hoặc, nếu chưa có, bằng `$PSQL update escalation_tickets set created_at=…, waiting_since=…` (chỉ trên DB test), rồi chờ tick job `ticket.sla` (`TICKET_SLA_TICK` = 15 phút; QC đặt `TICKET_SLA_TICK=10s` ở stack test nếu cấu hình cho phép, `Q-QC-P4-14`). Giờ Việt Nam (UTC+7). Bảng thời gian (SRS 9.3): T2 = thứ Hai.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P4-05-01 | AC1 (hỏi tiếp mở lại) | `$T` `ANSWERED` của `$SVA` | `j -X POST $GW/api/v1/chat/tickets/$T/followup -H "$SVA" -d '{"content":"Em chưa hiểu bước 2","client_msg_id":"c1"}'`; `select status,claimed_by,reopened_count,sla_stage,waiting_since from escalation_tickets where id='$T'` | `200 {ticket,message}`; `OPEN`, `claimed_by` rỗng, `reopened_count`+1, `sla_stage=0`, `waiting_since≈now()`; tin `role=USER`, `intent='ESCALATION_FOLLOWUP'`, `blocks` có `kind:"ticket"` |
| TC-P4-05-02 | AC1 (giảng viên thấy đúng tin này) | sau TC-01 | `GET $Q/tickets/$T -H "$TCH"` | `transcript` có tin `ESCALATION_FOLLOWUP`; không tin khác của phiên (grep canary tin lân cận = 0) |
| TC-P4-05-03 | AC1 (idempotent) | – | lặp đúng `client_msg_id:"c1"` | không thêm tin (`select count(*) from chat_messages where intent='ESCALATION_FOLLOWUP' and …` = 1) |
| TC-P4-05-04 | AC1 (độ dài) | – | `content:""`; `content` 2.001 ký tự; 2.000 | `422`; `422`; `200` |
| TC-P4-05-05 | AC1 (trạng thái) | ticket `OPEN`/`CLAIMED` | `followup` | `409 TICKET_STATE_INVALID` |
| TC-P4-05-06 | AC1 (quá 30 ngày) | `CLOSED` với `closed_at` 31 ngày trước (`update`) | `followup` | `409 TICKET_REOPEN_EXPIRED`; ở 29 ngày → `200` |
| TC-P4-05-07 | AC1 (không quá hạn tức thì) | ticket tạo 10 ngày trước, `ANSWERED`; `followup` | chờ một tick `ticket.sla`; `select sla_stage` | vẫn `0` (đồng hồ từ `waiting_since`, không `created_at`); `TICKET` ở Hôm nay `Overdue=false` |
| TC-P4-05-08 | AC1 (chủ ticket) | – | `followup` bằng `$SVB`; `$TCH`; `$ADM` | `404`; `403`; `403` |
| TC-P4-05-09 | AC2 (`Đã rõ`) | `$T` `ANSWERED` | `POST /chat/tickets/$T/confirm -H "$SVA"`; `select status,closed_reason,closed_at` | `CLOSED`, `closed_reason='STUDENT_CONFIRMED'`, `closed_at` có |
| TC-P4-05-10 | AC2 (hoàn tác 5 s) | sau TC-09 | `POST …/confirm/undo` trong < 5 s; rồi tạo ticket khác, `confirm`, chờ 6 s, `undo` | trong 5 s → `ANSWERED`; sau 5 s → lỗi (kỳ vọng `409`, ghi mã thực tế); undo sau hành động khác → lỗi |
| TC-P4-05-11 | AC2 (UI) | `playwright-cli`, `/chat` của `$SVA` | bấm `Đã rõ` | cập nhật lạc quan, dòng tĩnh "Đã đóng · Hoàn tác" 5 giây; **không** hộp thoại xác nhận, không toast |
| TC-P4-05-12 | AC2 (vai) | – | `confirm` bằng `$SVB`; `$TCH` | `404`; `403` |
| TC-P4-05-13 | AC3 (tự đóng 7 ngày) | `ANSWERED` có `answered_at` 8 ngày trước (`update`) | chờ job `ticket.autoclose` | `CLOSED`, `closed_reason='AUTO_7D'`; chạy hai lần = một lần (không đổi `closed_at` lần hai); `OPEN`/`CLAIMED` quá 7 ngày không đóng |
| TC-P4-05-14 | AC3 (câu trả lời mới đặt lại mốc) | `ANSWERED` cách 6 ngày; teacher trả lời thêm | chờ job | không đóng (mốc 7 ngày tính từ lần trả lời mới) |
| TC-P4-05-15 | AC3 (sau khi đóng) | `CLOSED` 'AUTO_7D' | `$SVA` mở `/chat` | thấy "Đã đóng"; `Hỏi tiếp` được trong 30 ngày |
| TC-P4-05-16 | AC4 (hàm giờ làm việc — hộp đen bằng hành vi) | không có API trực tiếp; bảng SRS 9.3 | chạy `cd backend-go && go test ./internal/escalation -run 'TestWorkingElapsedTable\|TestReminderAtTable\|TestEscalateAtTable\|TestWorkingHoursParametrised\|TestWorkingHoursProperty' -v` và đếm bảng; **QC tự kiểm thêm bằng stack**: ticket `waiting_since` = T2 08:00 (giờ VN), xem lúc job nhắc (TC-18); T6 12:00; T7 10:00 | `ok`; ≥ 14 dòng bảng khung mặc định + 5 dòng khung 08:00–17:00; kết quả thực khớp bảng: T2 08:00 → nhắc T3 08:00; T6 12:00 → nhắc T2 12:00 (không nhắc T7/CN); `from>to` = 0 |
| TC-P4-05-17 | AC4 (dòng bảng) | – | in từng dòng SRS 9.3: T2 09→T2 17 = 8 h; T6 16→T2 09 = 17 h; T7 10→CN 10 = 0; T6 23→T2 01 = 2 h; T2 07→T2 08 = 1 h; T2 08→T3 08 = 24 h; T2 17→T3 08 = 15 h; T4 12→T4 12 = 0; T5 08→T3 (tuần sau) 17 = 81 h | đúng 9 dòng khớp |
| TC-P4-05-18 | AC5 (nhắc ở 24 giờ làm việc) | ticket `OPEN` có `waiting_since` = 25 giờ làm việc trước (T3 09:00 nếu giờ hiện tại T4 10:00, tránh cuối tuần); `sla_stage=0` | chờ tick; `select sla_stage`; thông báo `TICKET_STALE` của teacher và TA; Mailpit | `sla_stage=1`; một thông báo `TICKET_STALE` gộp theo lớp "N câu hỏi chờ quá 24 giờ làm việc"; một thư `ticket_reminder` tới mỗi TA / TEACHER `ACTIVE` của lớp; chạy lại job → không thêm |
| TC-P4-05-19 | AC5 (cuối tuần) | ticket `waiting_since` = T6 12:00; giờ hiện tại CN | chờ tick | không nhắc ở T7/CN; sẽ nhắc T2 12:00 (nếu đang T2 sau 12:00: nhắc) |
| TC-P4-05-20 | AC5 (hai bản worker song song) | `docker compose up --scale worker=2` | chờ tick | `sla_stage` tăng một lần; thư `ticket_reminder` đúng 1 cho mỗi người nhận |
| TC-P4-05-21 | AC5 (CLAIMED quá hạn) | ticket `CLAIMED` bởi teacher, quá hạn | tick | chỉ **teacher** nhận `TICKET_STALE` cá nhân, không thư, không nhóm; ticket `Bỏ nhận` rồi nhận lại → nhắc lại lần nữa (`dedupe_key` theo `claimed_at`) |
| TC-P4-05-22 | AC5 (lọc dùng `waiting_since`) | hai ticket: `created_at` cũ nhưng `waiting_since` mới (sau followup); và ngược lại | tick | chỉ ticket có `waiting_since` quá hạn được nhắc |
| TC-P4-05-23 | AC6 (72 giờ) | ticket `OPEN` với 73 giờ làm việc | tick; `GET $GW/api/v1/me/today -H "$TCH"` | `sla_stage=2`; mục `TICKET` của lớp `Overdue=true` lên **trước mọi bậc** ở "Hôm nay"; `Reason` ghi số giờ ("… · lâu nhất 73 giờ") |
| TC-P4-05-24 | AC6 (báo Admin khi không ai hoạt động) | lớp có ticket `sla_stage=2`; không TA/TEACHER nào `auth_sessions.last_used_at` trong 72 giờ lịch (`update auth_sessions`) và không ai thao tác ticket | tick | ADMIN nhận **một** `COURSE_TICKETS_STALLED` "Lớp 761987 có N câu hỏi chờ quá 72 giờ làm việc" + mục `Hôm nay` của ADMIN (bậc 45); chỉ mã lớp và số, **không** nội dung ticket (grep canary = 0) |
| TC-P4-05-25 | AC6 (có người hoạt động) | một teacher có `last_used_at` trong 72 giờ | tick | **không** báo ADMIN |
| TC-P4-05-26 | AC6 (mỗi ngày một lần) | lớp không hoạt động | chạy tick nhiều lần trong ngày; rồi lùi `now` một ngày (`seed-age` hoặc `update notifications.created_at` + chờ) | 1 thông báo/ngày (`dedupe_key = stalled:<course>:<yyyymmdd>`); ba ngày liên tiếp → 3 thông báo, mỗi ngày 1 |
| TC-P4-05-27 | AC7 (chữ trạng thái) | 4 ticket ở 4 trạng thái của `$SVA` | `playwright-cli` `/chat`, đọc chữ từng trạng thái; `eval innerText` | `OPEN` "Đang chờ giảng viên · đã gửi {Age} trước" ("vừa gửi" < 1 phút); `CLAIMED` "Giảng viên đã nhận · đã chờ {Age}"; `ANSWERED` nhãn người trả lời + `Hỏi tiếp`, `Đã rõ`; `CLOSED` "Đã đóng" + `Hỏi tiếp` (≤ 30 ngày); **không** chữ "ticket", "SLA", tên người nhận, số độ tin cậy |
| TC-P4-05-28 | AC7 (cập nhật mỗi phút, không nhảy layout) | – | giữ trang mở 2 phút; đo `getBoundingClientRect` trước/sau | `Age` đổi; vị trí khối không đổi (CLS ≈ 0) |
| TC-P4-05-29 | AC8 (`Hôm nay` sinh viên) | `$SVA` có `TICKET_ANSWERED` chưa đọc | `GET $GW/api/v1/me/today -H "$SVA"` | mục "Giảng viên đã trả lời câu hỏi của bạn · {Age} trước", `Href` `/chat?session=…` (bậc 40) |
| TC-P4-05-30 | AC8 (đọc khi mở) | – | `$SVA` mở `/chat?session=…` có tin `STAFF_ANSWER`; `network` thấy `POST /notifications/read {link}`; `GET /chat/sessions/$S/messages` đơn lẻ | mục biến ≤ 60 s; `GET …/messages` không có tác dụng ghi (`select updated_at` thông báo không đổi nếu chỉ GET) |
| TC-P4-05-31 | AC8 (chỉ của mình) | thông báo của `$SVB` cùng `link` | `POST /notifications/read {link}` bằng `$SVA` | chỉ thông báo của A đổi; B không đổi; `$SVB` không thấy mục của A |
| TC-P4-05-32 | AC9 (`Đóng không trả lời`) | ticket `OPEN` | `POST …/tickets/$T/dismiss {"reason":"OUT_OF_SCOPE"}`, rồi `DUPLICATE`, `NEED_MORE_INFO` (ba ticket) | `CLOSED`, `closed_reason='STAFF_DISMISSED'`; một tin `STAFF_ANSWER` ngắn "Giảng viên đã đóng câu hỏi này: ngoài phạm vi môn học." / "…trùng với câu hỏi khác." / "…cần bạn nói rõ hơn."; chuông cho sinh viên |
| TC-P4-05-33 | AC9 (hỏi tiếp sau dismiss) | ba ticket `CLOSED` | `$SVA` `Hỏi tiếp` mỗi ticket | `200` (cả ba, ≤ 30 ngày) |
| TC-P4-05-34 | AC9 (người khác đã nhận) | ticket `CLAIMED` bởi TA | teacher `dismiss` | `409` (kèm người nhận); `reason` ngoài enum → `422` |
| TC-P4-05-35 | AC10 (xem trước che PII) | ticket `ANSWERED` có câu hỏi chứa tên `Nguyễn Văn Gioi`, MSSV, email, SĐT, CCCD; câu trả lời cũng | `GET $Q/tickets/$T/knowledge-preview -H "$TCH"` | `{question,answer}` đã `Redact` ([đã ẩn]); grep tên / MSSV / email / SĐT / CCCD = 0 |
| TC-P4-05-36 | AC10 (còn PII) | – | `POST $Q/tickets/$T/knowledge {question,answer}` giữ nguyên tên | `422 PII_DETECTED` |
| TC-P4-05-37 | AC10 (câu hỏi cá nhân) | ticket từ câu "Điểm giữa kỳ của em là bao nhiêu?" (`PERSONAL_QUESTION`) | preview / `POST knowledge` | `422 KNOWLEDGE_PERSONAL` |
| TC-P4-05-38 | AC10 (lưu) | ticket sạch | `POST …/knowledge` đã sửa; `select * from content_chunks, knowledge_items, audit_log` | `201`; `content_chunks.text` = bản đã sửa, đã che, `embedding IS NULL` ngay sau; một dòng `knowledge_items` mỗi ticket; outbox `knowledge.sync`; `save_as_knowledge=true`; `audit_log`; canary tên/MSSV ở `content_chunks.text` = 0 |
| TC-P4-05-39 | AC10 (lưu hai lần, vai, trạng thái) | – | `POST knowledge` lần hai; `$SVA`, `$ADM`, TA lớp khác; ticket `OPEN`/`CLAIMED` | cập nhật, không đôi (`knowledge_items` vẫn 1); `403` ×3; `409 TICKET_STATE_INVALID` |
| TC-P4-05-40 | AC10 (embed bản đã sửa) | provider giả ghi chuỗi `Embed` | sửa câu trả lời khác câu gốc rồi lưu | chuỗi được `Embed` là bản đã sửa (không phải câu gốc) |
| TC-P4-05-41 | AC11 (dùng được ngay) | ticket đã lưu tri thức | ≤ 10 s sau; `$SVB` hỏi câu tương tự (provider giả) | `content_chunks.embedding` có; câu trả lời của SV B có nguồn "Hỏi đáp đã được giảng viên xác nhận" (nguồn TICKET: không liên kết; POST: liên kết `/threads/{id}`); `ep:rag:ver` của lớp tăng (`$RDS GET`); cache câu trả lời vô hiệu |
| TC-P4-05-42 | AC11 (chưa nạp) | provider embedding chết (dừng giả) | lưu tri thức | xác nhận vẫn thành công; giảng viên thấy "Chưa nạp vào tri thức" + `Thử lại`; đoạn **không** lọt ra tìm kiếm (`rag` trả 0); `Thử lại` ghi lại outbox `knowledge.sync`; khi provider sống, nạp xong |
| TC-P4-05-43 | AC11 (gỡ) | đã nạp | `DELETE …/tickets/$T/knowledge` | `204`; ≤ 5 s đoạn không còn ở kết quả tìm; `save_as_knowledge=false`; lưu lại bản sửa **thay** đoạn cũ (không đôi) |
| TC-P4-05-44 | AC11 (không lọt vào tài liệu) | đoạn tri thức tồn tại | `GET /library`, danh sách tài liệu của giảng viên, thống kê tài liệu, `share-from`, chọn tài liệu cho "Hỏi AI về tài liệu", `/today`, ingest; `select * from documents where category='THREAD_KNOWLEDGE' …` | không route nào liệt kê tài liệu `THREAD_K…`; tạo tài liệu với `category` đó → `422` |
| TC-P4-05-45 | AC11 (cô lập lớp) | tri thức ở lớp 1 | `$SVx` ở lớp 2 hỏi câu tương tự | không trả đoạn của lớp 1 |
| TC-P4-05-46 | AC12 (ngưỡng) | `/class/settings` bằng `$TCH` | kéo thanh (0,50–0,95, bước 0,05); `PUT $Q/settings {"escalation_threshold":0.85,"version":N}` | `200`; ở tin kế tiếp của sinh viên có hiệu lực (đọc mỗi lần): ngưỡng 0,95 → câu mặc định tự mở ticket |
| TC-P4-05-47 | AC12 (ngoài khoảng / bước) | – | `0.49`, `0.96`, `0.82`, `"abc"` | `422` mỗi ca |
| TC-P4-05-48 | AC12 (version, vai) | – | `version` cũ; `$TA_`, `$SVA`, `$ADM` | `409 VERSION_CONFLICT` kèm giá trị hiện tại; `403` ×3 |
| TC-P4-05-49 | AC12 (audit, UI) | – | đổi ngưỡng; `select old,new from audit_log where …`; `playwright-cli` xem thanh | audit có giá trị cũ / mới; thanh có đầu "Ít chuyển", cuối "Chuyển nhiều", không câu mô tả khác |
| TC-P4-05-50 | AC13 (worker chết giữa `ticket.sla`) | ticket quá hạn | `docker compose kill -s KILL worker` ngay lúc job chạy (script theo `sla_stage`/log); khởi động lại | lần chạy sau hoàn tất; mỗi ticket nhắc đúng một lần (thư/thông báo đếm bằng `select count(*)`); không có "đã gửi mà chưa ghi": không dòng outbox/thông báo nào sinh khi `sla_stage` còn 0 |

## Nhánh lỗi
TC-P4-05-04, 05, 06, 34, 36, 37, 42, 47, 48, 50. Thêm TC-P4-05-E1: Redis chết lúc `confirm` → `503`; TC-P4-05-E2: `ticket.autoclose` chạy khi DB chậm — không đóng nhầm `OPEN`.

## Phân quyền
TC-P4-05-08, 12, 31, 39, 48. Thêm TC-P4-05-P1: TA lớp 1 gọi `knowledge`/`dismiss` ticket lớp 2 → `404`/`403`; `COURSE_TICKETS_STALLED` chỉ ADMIN nhận, teacher không.

## Câu hỏi (Q-QC)
- **Q-QC-P4-13** Cú pháp `gateway admin seed-age` (proposals #7) chưa ghi ở US/SRS/handoff. Cần ví dụ lệnh để QC lùi `created_at`/`waiting_since`. Nếu thiếu, QC dùng `UPDATE` trực tiếp trên DB test.
- **Q-QC-P4-14** Trigger job `ticket.sla` / `ticket.autoclose` trên stack thật: có biến cấu hình rút ngắn nhịp (`TICKET_SLA_TICK`) hay lệnh chạy một lần (`gateway admin run-job ticket.sla`)? Không có → chờ 15 phút/lần, TC-18, 19, 20, 23–26 dài.
- **Q-QC-P4-15** AC10: bản "chưa lưu" dùng nhãn "Chưa nạp vào tri thức"; tên route kiểm "embedding chết" ở stack thật chưa rõ (dừng provider giả bằng cách nào?). Hỏi PM/dev: cờ `LLM_PROVIDER=fake` có `FAKE_EMBED_FAIL` không?

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
