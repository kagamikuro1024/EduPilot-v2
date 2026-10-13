# QC test case — US-P4-04 (thư qua mail, outbox, thư lỗi ở `/observability`)
Nguồn: `US.md` US-P4-04 AC1…AC10 + `SRS.md` 2, 3.3, 4.5, 6.3 (route 17), 7.5, 7.6, 9.1; PRD M3 ("mail ≤ 60 s"). Hộp đen, không đọc code dev.
Biến như `tc-US-P4-01.md`; `MP=http://localhost:8025` (Mailpit REST). `mp_count '<Message-ID>'` = `curl -s $MP/api/v1/messages | jq '[.messages[]|select(.MessageID=="<id>")]|length'`. `mail_outbox` đọc bằng `$PSQL`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P4-04-01 | AC1 | `$SVA` có `notify_answer_by_mail=true`; ticket `$T` `CLAIMED` | giảng viên `answer` (ghi `t0`); đợi thư: `curl -s $MP/api/v1/messages \| jq '.messages[0]'` | thư `escalation_answered` ≤ 60 s sau `t0`; tiêu đề đúng "Giảng viên đã trả lời câu hỏi của bạn" (không tên); thân có câu hỏi gốc (cắt 300 ký tự), trả lời (cắt 2.000 + "Xem đầy đủ trong EduPilot" nếu cắt), liên kết `APP_PUBLIC_URL/chat?session=…&m=…` |
| TC-P4-04-02 | AC1 (p95 trên 10 lần) | 10 ticket / 10 sinh viên (hoặc `$SVA` lặp 10 lần) | đo `t_mail − t0` | p95 ≤ 60 s (in bảng 10 số) |
| TC-P4-04-03 | AC1 (cắt) | câu hỏi 350 ký tự, trả lời 2.500 ký tự | gửi, đọc thân thư | câu hỏi cắt ở 300; trả lời cắt ở 2.000 kèm "Xem đầy đủ trong EduPilot" |
| TC-P4-04-04 | AC1 (không đôi) | – | đẩy lại dòng outbox (`update outbox set status='PENDING'`) | **1** thư (`dedupe_key = escalation_answered:<answer_message_id>`); `mp_count` = 1 |
| TC-P4-04-05 | AC1 (liên kết không nằm trong payload) | – | `select payload from mail_outbox where …` | `payload` chỉ `{ticket_id, answer_message_id}`; không có URL, không nội dung |
| TC-P4-04-06 | AC2 (không rò chéo) | `$SVA` (`CANARY-A1` ở câu hỏi), `$SVB` (`CANARY-B2`); cả hai ticket được trả lời | lấy thư của A và B từ Mailpit (`/api/v1/message/{id}`); `grep -c CANARY-B2` thư A và ngược lại | thư A chỉ có `CANARY-A1`; thư B chỉ có `CANARY-B2`; 0 chéo |
| TC-P4-04-07 | AC2 | – | `select payload from mail_outbox` mọi dòng `escalation_answered` | chỉ `ticket_id`, `answer_message_id`; grep canary / tên / MSSV = 0 |
| TC-P4-04-08 | AC2 (thư nhắc nhóm không có tên / nội dung) | ticket quá 24 giờ (xem TC-US-P4-05) | đọc thư `ticket_reminder` | chỉ số lượng, mã lớp, liên kết `/inbox?course=…`; 0 tên, 0 nội dung, 0 MSSV |
| TC-P4-04-09 | AC3 (giết worker, điểm 1: trước SMTP) | ticket có dòng `mail.send` `PENDING` | `docker compose kill -s KILL worker` ngay khi dòng ở `PROCESSING` (script: vòng lặp `psql` tới khi `status='PROCESSING'` rồi kill); `docker compose up -d worker` | sau khi worker lên: `mp_count '<id>@domain'` = 1; `status='SENT'`; không ticket thiếu thư |
| TC-P4-04-10 | AC3 (điểm 2: sau SMTP, trước `SENT`) | như trên; làm chậm bước ghi `SENT` (chặn bằng `pg_sleep` trigger tạm trên `mail_outbox` update) | kill giữa lúc Mailpit đã nhận | sau khởi động lại: Mailpit có đúng **1** thư mang `Message-ID` của dòng (`MP_IGNORE_DUPLICATE_IDS=true`); `status='SENT'`. Gỡ trigger |
| TC-P4-04-11 | AC3 (`Message-ID` tất định) | – | `curl $MP/api/v1/messages \| jq -r '.messages[].MessageID'` so `mail_outbox.id` | dạng `<{mail_outbox.id}@{domain của MAIL_FROM}>` |
| TC-P4-04-12 | AC3 (ticket + outbox cùng giao dịch) | – | tạo ticket rồi kill gateway giữa chừng (nhiều lần, 20) | không có ticket nào thiếu dòng outbox `escalation.created`; không dòng outbox mồ côi |
| TC-P4-04-13 | AC4 (`ticket_reminder`) | ticket `OPEN` quá `TICKET_REMIND_AFTER` | chờ job SLA (TC-US-P4-05) | thư tới TA + TEACHER `ACTIVE` của lớp; tiêu đề "Có câu hỏi chờ giảng viên quá 24 giờ làm việc"; thân: số câu, mã lớp, `/inbox?course=…` |
| TC-P4-04-14 | AC4 (`thread_reply`) | `$SVA` đăng thread; `$SVB` bình luận | xem Mailpit | thư tới `$SVA`, tiêu đề "Câu hỏi của bạn trên Threads có trả lời mới"; thân: tiêu đề thread + liên kết; cũng khi câu trả lời AI `VERIFY` / `CORRECT` |
| TC-P4-04-15 | AC4 (`ticket_new`, cờ bật) | teacher bật `notify_ticket_by_mail` (`PUT /me/settings`) | 3 ticket trong 5 phút ở lớp 1 | **1** thư `ticket_new` (gộp); ghi số ticket đang chờ; không nội dung |
| TC-P4-04-16 | AC4 (cửa sổ trượt 15 phút) | đồng hồ giả / chờ thật | ticket T0, T0+14 m 59 s, T0+15 m 01 s (đồng hồ giả hoặc đặt TTL khoá `ep:mail:ticket_new:{uid}:{course}` bằng `$RDS`) | 14 m 59 s → **1** thư tổng; ticket ở 15 m 01 s → thư thứ hai; ticket 10:00 và 10:16 → 2 thư |
| TC-P4-04-17 | AC4 (Redis lỗi thì vẫn gửi) | `docker compose stop redis` rồi tạo ticket (nếu outbox còn chạy) hoặc chặn riêng khoá | ticket mới | thư vẫn gửi (không bỏ) |
| TC-P4-04-18 | AC4 (tiêu đề không có tên) | – | mọi thư test | tiêu đề không chứa họ tên (grep tên seed = 0) |
| TC-P4-04-19 | AC5 (thử 3 lần rồi DEAD) | Mailpit dừng (`docker compose stop mailpit`) hoặc `SMTP_HOST` sai | trả lời ticket; theo dõi `select attempts,status,last_error,updated_at from mail_outbox` | thử lại theo 1 s, 5 s, 30 s (3 lần thử lại, tổng 4 lần gọi); sau đó `status='DEAD'`; `last_error` là nhãn lớp lỗi (grep địa chỉ email / nội dung = 0); tin outbox vào dead-letter |
| TC-P4-04-20 | AC5 (ứng dụng không bị ảnh hưởng) | như trên | trong khi thử: `$SVA` mở `/chat`; chuông | ticket vẫn `ANSWERED`, tin trả lời vẫn hiện, chuông vẫn đến |
| TC-P4-04-21 | AC5 (lỗi vĩnh viễn) | hộp thư người nhận không tồn tại (SMTP giả trả `550`; dùng cổng tạm trả 550) hoặc địa chỉ bị từ chối | gửi | `DEAD` ngay, `attempts=1`, không thử lại |
| TC-P4-04-22 | AC6 (cờ sinh viên) | `$SVA` `notify_answer_by_mail=false` | trả lời ticket; thread có bình luận | **không** thư `escalation_answered` / `thread_reply`; chuông vẫn có |
| TC-P4-04-23 | AC6 (mặc định ticket_new tắt) | teacher mới chưa đổi cài đặt | tạo ticket | không thư `ticket_new` |
| TC-P4-04-24 | AC6 (địa chỉ nhận) | ticket của `$SVA` | xem `To` của thư | đúng email **đã xác minh** của chủ ticket; thử client truyền `to` trong body `answer` → bỏ qua |
| TC-P4-04-25 | AC6 (`ticket_reminder` đúng người) | lớp 1 có teacher, TA, ADMIN, teacher lớp khác | job nhắc | thư chỉ TA + TEACHER `ACTIVE` của lớp 1; không ADMIN, không teacher lớp khác |
| TC-P4-04-26 | AC7 (cài đặt UI) | `playwright-cli` | `/settings` bằng teacher; bằng TA; bằng STUDENT | TA/TEACHER thấy "Mail báo ticket mới" (tắt mặc định); STUDENT thấy "Mail khi giảng viên trả lời" (bật mặc định) và không thấy công tắc ticket; không hiện công tắc không áp dụng |
| TC-P4-04-27 | AC7 (lưu lạc quan) | – | bật/tắt công tắc; theo dõi `network` | giao diện đổi ngay, không toast; `PUT /me/settings` có `version` |
| TC-P4-04-28 | AC7 (version, STUDENT) | – | `PUT /me/settings {notify_ticket_by_mail:true, version:<cũ>}`; STUDENT gửi `notify_ticket_by_mail` | sai `version` → `409`; STUDENT → `200` (lưu, không tác dụng) |
| TC-P4-04-29 | AC8 | repo | `curl $GW/api/v1/...mail_outbox`; liệt kê route (`api/openapi.yaml`) có `outbox` | không route nào trả `mail_outbox` (trừ route 17 đã che); `SMTP_HOST=mailpit` trong compose test |
| TC-P4-04-30 | AC8 (log) | sau mọi TC mail | `docker compose logs gateway worker \| grep -ci '@edupilot.local'`; so tên template | log chỉ có `mail_id`, `template`, lớp lỗi; 0 email người nhận (trừ chỗ cấu hình khởi động, ghi lại nếu có) |
| TC-P4-04-31 | AC9 | có thư `DEAD` (TC-19/21), ADMIN | `GET /admin/observability/failed-mail -H "$ADM"`; mở `/observability` bằng `playwright-cli` | mục "Thư chưa gửi được" ở **đầu** trang; dòng: mẫu thư, người nhận đã che (`a***@domain`), số lần thử, lớp lỗi, thời điểm; không thân thư, không payload; phần còn lại giữ mock |
| TC-P4-04-32 | AC9 (phân quyền) | – | route bằng `$TCH`, `$TA_`, `$SVA` | `403`; teacher mở `/observability` không thấy mục (hoặc cả trang như cũ — ghi) |
| TC-P4-04-33 | AC9 (rỗng) | không có thư lỗi | mở mục | "Không có thư lỗi." không nút |
| TC-P4-04-34 | AC9 (chỉ mục) | `mail_outbox` 100.000 dòng, 500 `DEAD` (gieo `generate_series`) | `EXPLAIN` truy vấn trang đầu (hoặc `j …/failed-mail` với `log_statement`) | dùng `mail_outbox_dead_idx`; không `Seq Scan on mail_outbox`; trang theo con trỏ |
| TC-P4-04-35 | AC10 (payload hỏng) | chèn dòng `mail_outbox` thiếu `ticket_id` (`$PSQL insert`) | chờ worker | `DEAD` ngay, lớp lỗi `payload_invalid`; không thư rỗng (Mailpit không thêm) |
| TC-P4-04-36 | AC10 (ticket đã xoá) | dòng `mail_outbox` trỏ ticket đã xoá | chờ worker | `DEAD` ngay, `ticket_missing`; không thử lại |

## Nhánh lỗi
TC-P4-04-17, 19, 20, 21, 35, 36. Thêm TC-P4-04-E1: Mailpit chết lúc trả lời, bật lại sau 10 s → thư đến đúng 1 lần (thử lại thành công); TC-P4-04-E2: người nhận không có email đã xác minh → `DEAD` (ghi lớp lỗi thực tế).

## Phân quyền
TC-P4-04-24, 25, 32. Thêm TC-P4-04-P1: thư `escalation_answered` không bao giờ tới giảng viên / TA; `COURSE_TICKETS_STALLED` chỉ thông báo ADMIN, không thư (kiểm Mailpit 0 thư cho admin ở mọi loại ngoài `ticket_reminder` không tới ADMIN).

## Câu hỏi (Q-QC)
- **Q-QC-P4-10** AC3 yêu cầu "giết worker" ở hai điểm; hộp đen cần cách dừng đúng điểm. Đề xuất: dùng script `docs/sprints/7/qc/scripts/kill-worker-at.sh` theo trạng thái `PROCESSING` của dòng; điểm sau SMTP cần cài trigger tạm hoặc `iptables`/`docker pause mailpit` sau `DATA`. Chấp nhận kết quả bằng chứng "Mailpit đúng 1 thư" ở cả hai kịch bản.
- **Q-QC-P4-11** AC4 `ticket_new` cần đồng hồ giả cho 14 m 59 s / 15 m 01 s; `clock.Fake` chỉ có trong test Go. Trên stack chạy thật không có đồng hồ giả. Đề xuất: kiểm đúng/sai bằng `$RDS TTL ep:mail:ticket_new:…` (xem khoá còn hạn ≈ 15 phút sau thư đầu; xoá khoá = ra khỏi cửa sổ) thay vì đợi thật; kiểm `clock.Fake` do dev chạy. PM xác nhận.
- **Q-QC-P4-12** TC-21 (lỗi vĩnh viễn 550) cần SMTP giả khác Mailpit; spec không nêu cách tạo. Đề xuất: cổng `nc -l` trả `550` tạm thời.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
