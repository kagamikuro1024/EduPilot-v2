# QC report — US-P4-04 (thư: đúng lúc, không lộ người khác, lỗi không hỏng ứng dụng) — vòng 1
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `79a230d`. Bộ TC: `tc-US-P4-04.md` (không sửa chữ TC). Handoff: `dev-US-P4-04.md`. Stack `edupilot100` dựng lại từ code mới (`docker-compose.test.yml`, `--scale gateway=2`, provider `fake`, Mailpit `MP_IGNORE_DUPLICATE_IDS=true`). Mailpit: `http://localhost:8125`. Giả lập "SMTP 550" bằng một container Python mang bí danh mạng `mailpit` (tạm, đã gỡ).

## Kết luận: **PASS** (36 TC + 3 nhánh; 1 TC BLOCKED: TC-12)
## 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 thư trả lời ≤ 60 s | PASS | 0,22 s sau `answer`; tiêu đề "Giảng viên đã trả lời câu hỏi của bạn" (không tên); thân có "Câu hỏi của bạn ở lớp 761987: …", "Giảng viên TS. Lê Thu Hà trả lời: …", liên kết `https://localhost:543/chat?session=…&m=…` |
| 02 p95 trên 10 lần | PASS | 10 sinh viên: 0,49 0,65 0,52 0,55 0,49 0,61 0,49 0,20 0,86 0,33 s → p95 ≈ 0,86 s |
| 03 cắt | PASS | câu hỏi 350 ký tự → đúng 300 + "…"; trả lời 2.500 → đúng 2.000 + "… Xem đầy đủ trong EduPilot" |
| 04 không đôi | PASS | đặt lại dòng `outbox` `mail.send` (`dispatched_at=null`) với `mail_outbox` `QUEUED`: sau 12 s `SENT`, Mailpit đúng 1 thư (cùng `Message-ID`), `dedupe_key` 1 dòng |
| 05, 07 payload | PASS | `{answer_message_id, ticket_id}`; grep `CANARY|tên|MSSV` = 0 |
| 06 không rò chéo | PASS | thư `sv.gioi` chỉ `CANARY-A1`; thư `sv.kha` chỉ `CANARY-B2`; chéo = 0 |
| 08 thư nhắc không tên | PASS | "Lớp 761987 có 8 câu hỏi đang chờ." + "Mở hộp thư hỗ trợ: …/inbox?course=…"; 0 tên, 0 nội dung |
| 09 giết worker trước SMTP | PASS | trạng thái `mail_outbox` chỉ có `QUEUED / SENT / DEAD` (không có `PROCESSING`) nên dừng worker trước khi trả lời: dòng `QUEUED`, Mailpit 0; bật lại → `SENT`, đúng 1 thư |
| 10 giết worker sau SMTP, trước `SENT` | PASS | trigger tạm `pg_sleep(10)` khi `SENT`; Mailpit nhận (1), DB `QUEUED`; `kill -9` worker; bật lại → sau ≈ 60 s giao lại, `SENT`, Mailpit vẫn **1** thư |
| 11 `Message-ID` | PASS | `<mail_outbox.id>@edupilot.local` |
| 12 gateway chết giữa tạo ticket | **BLOCKED** | 3 lượt `kill -9` gateway khi 15–18 `escalate` đang bay: không bắt được yêu cầu giữa giao dịch (0 ticket tạo hoặc tạo trọn vẹn). Bất biến giữ trên 53 ticket: 0 ticket thiếu `escalation.created`, 0 dòng outbox mồ côi. Cần điểm dừng trong giao dịch (đồng hồ / hook) để kiểm đúng |
| 13 `ticket_reminder` | PASS | ticket 25 giờ → `run-job ticket.sla` → thư tới TA + TEACHER, tiêu đề "Có câu hỏi chờ giảng viên quá 24 giờ làm việc" |
| 14 `thread_reply` | PASS | bình luận người khác, `verify`, `correct` → thư tới người đăng, tiêu đề "Câu hỏi của bạn trên Threads có trả lời mới", thân có tiêu đề thread + `/threads/{id}` (câu trả lời AI chèn bằng SQL vì seed không có tài liệu) |
| 15 `ticket_new` gộp | PASS | cờ giảng viên bật; 5 ticket → 1 thư "Lớp 761987 có 18 câu hỏi đang chờ."; không nội dung |
| 16 cửa sổ 15 phút | PASS (đo gián tiếp) | khoá `ep:mail:ticket_new:{uid}:{course}` giữ mốc thư cuối (mili giây, TTL ≈ 15 phút); đặt mốc cũ 60 / 600 / 840 s → 0 thư; 899 / 901 s → 1 thư; ticket bị gộp **không** dời mốc (cố định từ thư cuối). Không có đồng hồ giả ở stack thật |
| 17 Redis chết | PASS | `docker compose stop redis`, tạo ticket → `201`; Redis lên → thư `ticket_new` `SENT` (≈ 25 s, không mất) |
| 18 tiêu đề không tên | PASS | 8 loại thư trong Mailpit: 0 tên seed trong tiêu đề |
| 19 DEAD sau 4 lần | PASS | Mailpit dừng: `attempts` 1 → 2 → 3 → 4 tại 1 s, 3 s, 8 s, 38 s (≈ 1 s / 5 s / 30 s); `DEAD`, `last_error='smtp_unavailable'` (không địa chỉ); dòng `outbox` `attempts=4`, `dead_at` đặt |
| 20 ứng dụng không bị ảnh hưởng | PASS | trong lúc thử: ticket `ANSWERED`, tin trả lời có, `TICKET_ANSWERED` (chuông) có |
| 21 lỗi vĩnh viễn 550 | PASS | SMTP giả trả `550` ở `RCPT` → `DEAD` ngay, `attempts=1`, `last_error='smtp_permanent_550'`, không thử lại |
| 22 cờ sinh viên tắt | PASS | `notify_answer_by_mail=false` → 0 thư (`escalation_answered`, `thread_reply`); chuông vẫn có |
| 23 mặc định `ticket_new` tắt | PASS | teacher mới: ticket → 0 thư |
| 24 địa chỉ nhận | PASS | `To` = email đã xác minh của chủ ticket; `answer` có `"to":"evil@x.vn"` → bỏ qua (`to_addr` = sinh viên, 0 thư cho `evil@x.vn`) |
| 25 `ticket_reminder` đúng người | PASS (một phần) | chỉ TA + TEACHER của lớp 1; ADMIN không nhận. Không có "giảng viên lớp khác" trong seed để đối chứng |
| 26 cài đặt UI | PASS | `scripts/p4-04-ui.mjs`: TEACHER, TA thấy "Mail báo ticket mới" (tắt); STUDENT thấy "Mail khi giảng viên trả lời" (bật), không thấy công tắc ticket |
| 27 lưu lạc quan | PASS | công tắc đổi sau 100 ms; một `PUT /me/settings {"notify_ticket_by_mail":true,"version":1}`; không toast |
| 28 `version` | PASS | `version` cũ → `409`; STUDENT gửi `notify_ticket_by_mail:true` → `200` |
| 29 không route nào trả `mail_outbox` | PASS | `openapi.yaml` không có từ `outbox` ở route trả thư (chỉ `failed-mail`, đã che); `SMTP_HOST=mailpit` ở `.env.local` |
| 30 log không email | PASS | `docker logs` gateway ×2 và worker: 0 dòng chứa `edupilot.local`; chỉ `mail_id`, `template`, `class` |
| 31 thư lỗi ở `/observability` | PASS | `GET /admin/observability/failed-mail` (ADMIN): `{id,template,to_masked,attempts,last_error,updated_at}`; UI: mục "Thư chưa gửi được" ở đầu trang (vị trí 124 ký tự), bảng Thư / Người nhận (`s***@edupilot.local`) / Số lần thử / Lỗi / Lúc; không thân thư |
| 32 phân quyền | PASS | TEACHER, TA, SV `403`; không token `401`; giảng viên mở `/observability` không thấy mục |
| 33 rỗng | PASS | "Không có thư lỗi." không nút |
| 34 chỉ mục | PASS | 100.000 dòng / 502 `DEAD`: trang đầu `n=50`, có `next_cursor`; `EXPLAIN` tương đương dùng `mail_outbox_dead_idx`, không `Seq Scan` |
| 35 payload hỏng | PASS | thiếu `ticket_id` → `DEAD` ngay, `payload_invalid`, `attempts=1`, 0 thư |
| 36 ticket đã xoá | PASS | `ticket_missing`, `DEAD` ngay, không thử lại |
| E1 Mailpit chết rồi bật sau 10 s | PASS | thư đến đúng 1 lần (lúc 39 s, `attempts=3`) |
| E2 người nhận chưa xác minh email | PASS (ghi chú) | không xếp thư (0 dòng `mail_outbox`, không `DEAD`); chuông vẫn có. TC viết "DEAD (ghi lớp lỗi thực tế)": hành vi thực chặn từ lúc xếp thư — đúng AC6 (chỉ gửi tới email đã xác minh) |
| P1 | PASS | 0 thư `escalation_answered` tới TEACHER / TA / ADMIN; ADMIN không nhận `ticket_reminder` |

## 2. Lệnh của handoff
Chạy trên máy QC: xem mục 4. e2e `escalation.spec.ts -g 'mail settings|failed mail'` nằm trong lượt 141 passed (desktop) của `report-US-P4-02.md`.

## 3. Ghi nhận (không làm FAIL)
- Mục "Thư chưa gửi được" có dòng phụ "Thư chưa gửi được, mới nhất trước" (luật UI: không câu giải thích dưới tiêu đề khối) và cột "Lỗi" in nhãn kỹ thuật (`smtp_unavailable`); chỉ Admin thấy.
- `mail_outbox` có ba trạng thái `QUEUED / SENT / DEAD`; TC-09 "đợi `PROCESSING`" không áp dụng được (đã dùng cách tương đương).
- TC-12 và `outreach_message` (chưa có chỗ xếp thư, thuộc US-P5) là nợ của dev đã khai.

## 4. Lệnh Go của handoff
Chạy trên máy QC (`DOCKER_HOST=unix://…/colima`, `-race`): `internal/escalation` (17 test mail) 19 passed; `internal/mail` 48 passed; `internal/thread` `TestThreadReplyMail` 1 passed; `internal/user` 3 passed; `-tags testroutes ./internal/contract/... ./cmd/worker` 41 passed.
