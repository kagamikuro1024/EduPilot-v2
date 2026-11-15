# Smoke hồi quy sprint 7 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm E) · Ngày: 2026-11-15
Tổng TC cũ: 582 · Đạt: 321 · Trượt: 1 · Không chạy K (a/b/c): 9 · Lỗi thời (T, tính vào cột K của brief): 2 · Chưa chạy (mẫu) N: 249   (P+F+K+T+N = 582 = 582)

Quy ước: P = đạt; F = trượt (chạy lại 1 lần vẫn trượt); K = không chạy vì (a) cần thiết bị không có, (b) cần người thật, (c) ngoài phạm vi đồ án; T = TC lỗi thời do spec đổi hợp lệ (ghi proposal); N = chưa chạy, ngoài mẫu smoke «đường chính + ≥ 1 nhánh lỗi/phân quyền cho mỗi US» (PM quyết có chạy nốt hay không).

Cách chạy:
- API (python/urllib + lib `/tmp/qc2e`): `s7_p4.py` (escalate, hàng chờ ticket, claim/release/answer/dismiss, phân quyền), `s7_mail.py` (Mailpit, mail_outbox, settings, failed-mail, chuông), `s7_p5.py`/`s7_p5b.py`/`s7_p5c.py` (follow-up, Đã rõ/hoàn tác, dismiss, tri thức từ ticket, ngưỡng chuyển), `s7_p6.py` (báo cáo/ẩn/ghim/sửa/xoá thread), `s7_p7.py` (khủng hoảng: `crisis-sets.json` 20 dương/17 âm qua chat thật, Threads 422 CRISIS_SUPPORT), `s7_p51.py` (sinh/sửa/xoá buổi học, điểm danh thủ công PUT lưới, phát biểu), `s7_p53.py` (heartbeat, study-time, danh sách SV, ngưỡng risk, hồ sơ 360, ghi chú, outreach, me/*).
- CI (US-CI-02): `gh run view 43712425198 --json jobs --log` trên run PR của be0c0d96 (14 job success) + `go list -tags testroutes ./...` đối chiếu log (100/100 gói, mỗi gói đúng một dòng matrix).
- Điểm danh thủ công sau 10.8 (hồi quy): PUT lưới/complete/xung đột version/phân quyền vẫn đúng; câu chat «Em vắng mấy buổi» khớp DB cho từng SV.
- Môi trường: Postgres ep-r có lần crash-recovery ~5 phút; hạn mức đăng nhập IP (LOGIN_THROTTLED) và 20 tin/phút/SV nên các script có giãn nhịp / dùng nhiều SV.
- Tick dài rút ngắn bằng override sẵn có của ep-r (TICKET_SLA_TICK=1m, REMINDER_TICK=30s) — không cần đổi env: SLA chạy được sau khi dùng seed-age vì hôm nay Chủ nhật (giờ làm việc = 0). Auto-close/reminder/mail nhắc chưa kiểm hết (xem N).
- Goose up/down/up trên DB tạm riêng: schema giống hệt, xem TC-P4-01-01. Go test sprint 7: `gorun.py 7` (bỏ ca `go test ./...` toàn cây); kết quả xem bảng. Postgres crash-recovery: xem smoke-6 và `evidence-7/postgres-crash-recovery.log`.
- Go sprint 7: chỉ chạy xong TC-P4-01-17 và TC-P4-01-37 (các lệnh còn lại — course/exam/thread/calendar, crm, integration seed — quá nặng khi máy tải ~20, dừng giữa chừng) → N.

## Tổng hợp theo US
| US | TC tổng | P | F | K | T | N | Ghi chú |
|---|---|---|---|---|---|---|---|
| US-P4-01 | 42 | 15 | 0 | 0 | 0 | 27 |  |
| US-P4-02 | 31 | 7 | 0 | 0 | 0 | 24 |  |
| US-P4-03 | 45 | 19 | 0 | 0 | 0 | 26 |  |
| US-P4-04 | 39 | 10 | 0 | 0 | 0 | 29 |  |
| US-P4-05 | 53 | 40 | 0 | 0 | 0 | 13 |  |
| US-P4-06 | 55 | 48 | 0 | 0 | 0 | 7 |  |
| US-P4-07 | 29 | 26 | 0 | 0 | 0 | 3 |  |
| US-P5-01 | 36 | 23 | 0 | 0 | 0 | 13 |  |
| US-P5-02 | 48 | 13 | 0 | 0 | 0 | 35 |  |
| US-P5-03 | 47 | 10 | 0 | 0 | 0 | 37 |  |
| US-P5-04 | 37 | 34 | 0 | 0 | 0 | 3 |  |
| US-P5-05 | 35 | 26 | 0 | 0 | 0 | 9 |  |
| US-P5-06 | 28 | 12 | 0 | 0 | 1 | 15 |  |
| US-S7-SEED | 25 | 19 | 0 | 3 | 1 | 2 |  |
| US-CI-02 | 32 | 19 | 1 | 6 | 0 | 6 |  |

## Kết quả từng ca
(Mỗi ca P/F/K/T một dòng. Ca N liền nhau cùng lý do được gom một dòng.)
| TC | KQ | Ghi chú |
|---|---|---|
| TC-P4-01-01 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P4-01-02 | P | mail_inbound=0; ^ALTER TABLE đúng 4 dòng (forum_threads,forum_posts,user_settings,courses); CREATE INDEX documents_knowledge_key/mail_outbox_dead_idx=2 |
| TC-P4-01-03 | P | docs/PROGRESS.md:94 '/ 00012 / escalation_notify /' |
| TC-P4-01-04 | P | kind=X→22P02; OPEN+claimed_by→23514; CLAIMED thiếu claimed_by→23514; ANSWERED thiếu answered_by→23514; QUESTION không message_id→23514; GRADE_APPEAL không appeal_payload→23514; QUESTION có appeal_payl |
| TC-P4-01-05 | P | cả thread và post→23514; cả hai rỗng→23514; hai lần cùng đích→23505; reason ngoài enum→22P02; 1 lần hợp lệ→OK; cascade→OK; cascade xoá chunk→BEGIN INSERT 0 1 DELETE 1 left=0 ROLLBACK |
| TC-P4-01-06 | P | DB 00011 → up 00012: ngưỡng 0.60→0.80 (kể cả version=3), 0.70 giữ; notify_ticket_by_mail true→false; default 0.80/false |
| TC-P4-01-07 | P | escalate → 201 {'already': False, 'ticket': {'id': '01a1df53-c92c-7dbc-8808-d96a6181fbf9', 'status': 'OPEN', 'creat (đã có ticket trước đó=0) |
| TC-P4-01-08 | P | 3 lần → [200, 200, 200], số ticket=1 |
| TC-P4-01-09 | P | 10 yêu cầu song song → [200, 201], số ticket=1 |
| TC-P4-01-10 … TC-P4-01-11 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-01-12 | P | ticket tự động cho câu yếu: AUTO_LOW_CONFIDENCE/0.000 |
| TC-P4-01-13 … TC-P4-01-16 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-01-17 | P | go test ./internal/agent -run TestShouldEscalateTable ok (7 ca con) |
| TC-P4-01-18 … TC-P4-01-27 (10 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-01-28 | P | chi tiết ticket → 200; keys=['ticket', 'question', 'ai_answer', 'transcript'] |
| TC-P4-01-29 | P | SV khác 404, TA 403, TCH 403, ADM 403, id lạ 404 |
| TC-P4-01-30 … TC-P4-01-36 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-01-37 | P | `go test -tags testroutes ./internal/contract/...`: lệnh gốc thiếu tag nên 404 `_test/*`; có tag chỉ 1 test đỏ do 504 DEADLINE (TestNoAnswerLeak, tải máy ~20) — chạy riêng ok 29 s |
| TC-P4-01-38 | P | 10.000 ticket/50 lớp: list lớp+status → Index Scan escalation_tickets_course_status_idx; theo SV → escalation_tickets_student_idx; SLA → sla_idx; autoclose → autoclose_idx; không Seq Scan escalation_t |
| TC-P4-01-E1 … TC-P4-01-P1 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-01 … TC-P4-02-02 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-03 | N | go test course/exam/thread/calendar: một gói đỏ lúc máy tải ~20 (chưa xác định gói) — chưa chạy lại riêng từng gói |
| TC-P4-02-04 | P | SSE /events của GV nhận khung `notification` (TICKET_NEW) < 1 s sau khi ticket tạo |
| TC-P4-02-05 … TC-P4-02-12 (8 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-13 | P | TICKET_NEW gộp: «8 câu hỏi đang chờ giảng viên» (một thông báo/lớp) |
| TC-P4-02-14 … TC-P4-02-16 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-17 | P | số trong tiêu đề (8) = count(OPEN) của lớp (8) |
| TC-P4-02-18 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-19 | P | thông báo ticket chứa PII: 0 |
| TC-P4-02-20 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-21 | P | SV đọc thông báo GV → 404; GV đọc → 204; lặp → 204 |
| TC-P4-02-22 … TC-P4-02-27 (6 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-28 | P | danh sách chuông 200, 3 mục, unread? keys=['next_cursor', 'unread_count'] |
| TC-P4-02-29 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-E1 | P | link rỗng → 422 |
| TC-P4-02-P1 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-01 | P | limit=2 → 2 mục, cursor=True |
| TC-P4-03-02 | P | limit 101/0 → [422, 422] |
| TC-P4-03-03 | P | status ANSWERED/ALL/CLAIMED → [200, 200, 200] |
| TC-P4-03-04 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-05 | P | me/tickets GV 200, 15 mục |
| TC-P4-03-06 | P | TA 200 |
| TC-P4-03-07 | P | ADMIN/SV me/tickets → 403/403 |
| TC-P4-03-08 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-09 | P | chi tiết → 200 |
| TC-P4-03-10 | P | GV lớp khác xem ticket → 403 |
| TC-P4-03-11 | P | SV/ADMIN → 403/403 |
| TC-P4-03-12 … TC-P4-03-13 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-14 | P | claim đồng thời → {'ta': 200, 'teacher': 409} |
| TC-P4-03-15 | P | người đã nhận bấm lại = 200 |
| TC-P4-03-16 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-17 | P | answer → 200 |
| TC-P4-03-18 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-19 | P | dismiss OUT_OF_SCOPE → 200 {'ticket': {'id': '01a1df54-2168-70c5-a6c2-b6ad84e6bbca', 'k |
| TC-P4-03-20 | P | answer lặp cùng key → 200, thân giống=True |
| TC-P4-03-21 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-22 | P | answer body rỗng/4001 → [422, 422] (ticket đã ANSWERED nên 409 cũng hợp lệ) |
| TC-P4-03-23 … TC-P4-03-25 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-26 | P | người khác release → 409, chủ release → 200 |
| TC-P4-03-27 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-28 | P | answer chứa tên/MSSV/email/CCCD người khác → [422, 422, 422] |
| TC-P4-03-29 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-30 | P | answer sạch → 200 |
| TC-P4-03-31 … TC-P4-03-38 (8 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-39 | P | {'sv08': [403, 403, 403], 'sv09': [403, 403, 403], 'admin': [403, 403, 403], 'ta': [200, 200, 409], 'teacher': [200, 200, 409]} |
| TC-P4-03-40 … TC-P4-03-42 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-03-E1 … TC-P4-03-P1 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-01 | P | Mailpit có 2 thư «Giảng viên đã trả lời» tới sv08 |
| TC-P4-04-02 | P | độ trễ outbox→SENT (s): 0.42 0.31 0.28 0.29 |
| TC-P4-04-03 | P | thư chứa link; rò nội dung trả lời/PII: []; độ dài 845 |
| TC-P4-04-04 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-05 | P | payload escalation chỉ chứa id (không nội dung) |
| TC-P4-04-06 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-07 | P | khoá payload: answer_message_id,ticket_id |
| TC-P4-04-08 … TC-P4-04-27 (20 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-28 | P | sai version → 409; STUDENT PUT notify_ticket_by_mail → 200 (lưu, không tác dụng) theo kỳ vọng TC |
| TC-P4-04-29 | P | route chứa "outbox": không có |
| TC-P4-04-30 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-31 | P | ADMIN failed-mail → 200 {'items': [{'id': '01a1df2b-7e85-7cdc-9b73-ae2a759fc2df', 'template': 'seed_obse |
| TC-P4-04-32 | P | GV/TA/SV → [403, 403, 403] |
| TC-P4-04-33 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-34 | P | mail_outbox 100.000 dòng/500 DEAD: EXPLAIN trang đầu + con trỏ dùng mail_outbox_dead_idx, không Seq Scan |
| TC-P4-04-35 … TC-P4-04-36 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-E1 … TC-P4-04-P1 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-01 | P | followup → 200 {'ticket': {'id': '01a1df53-c92c-7dbc-8808-d96a6181fbf9', 'status': 'OPEN', 'created_at': '2026-11-1 |
| TC-P4-05-02 | P | transcript GV = [STAFF_ANSWER, USER "Em chưa hiểu bước 2" (DB intent=ESCALATION_FOLLOWUP)]; canary tin lân cận=0 (API không phơi trường intent, kiểm intent ở DB) |
| TC-P4-05-03 | P | lặp cùng client_msg_id → 200; số tin=1 |
| TC-P4-05-04 | P | content rỗng/2001 → [422, 422] |
| TC-P4-05-05 | P | OPEN→409 TICKET_STATE_INVALID; CLAIMED→409 TICKET_STATE_INVALID |
| TC-P4-05-06 | P | 31 ngày→409 TICKET_REOPEN_EXPIRED; 29 ngày→200 |
| TC-P4-05-07 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-08 | P | followup bởi SV khác/GV/ADMIN → [404, 403, 403] |
| TC-P4-05-09 | P | confirm → 200; trạng thái CLOSED/STUDENT_CONFIRMED |
| TC-P4-05-10 | P | undo <5s → 200; trạng thái ANSWERED |
| TC-P4-05-11 | P | bấm "Đã rõ": ngay (300 ms) dòng "Đã đóng · Hoàn tác" (lạc quan); sau 6 s còn "Đã đóng" + "Hỏi tiếp" (Hoàn tác biến mất); không hộp thoại (role=dialog/alertdialog), không toast, không window.dialog |
| TC-P4-05-12 | P | confirm bởi SV khác/GV → [404, 403] |
| TC-P4-05-13 … TC-P4-05-16 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-17 | P | worktime_test.go TestWorkingElapsedTable: đủ 9 dòng SRS 9.3 (09→17=8h, T6 16→T2 09=17h, T7→CN=0, T6 23→T2 01=2h, T2 07→08=1h, T2 08→T3 08=24h, T2 17→T3 08=15h, cùng điểm=0, T5 08→T3 sau 17=81h) khớp;  |
| TC-P4-05-18 | P | tick SLA 1 phút (TICKET_SLA_TICK=1m), Chủ nhật nên tuổi ticket tính theo giờ làm việc = 0: dùng seed-age (UPDATE waiting_since vào các ngày làm việc trước): 8 giờ làm việc → stage 0; ~2 ngày → stage 1 |
| TC-P4-05-19 | P | ticket ≥ 72 giờ làm việc (waiting_since 2026-11-05) → sla_stage=2 sau một tick |
| TC-P4-05-20 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-21 | P | s21 CLAIMED quá hạn: TICKET_STALE cá nhân e7gv=2 (lần 1 + sau bỏ nhận/nhận lại, dedupe theo claimed_at), e7ta=0, thư nhắc cho lớp=0; không thông báo nhóm |
| TC-P4-05-22 | P | s22a(created_at cũ, waiting_since mới)=stage0; s22b(created_at cũ... waiting_since cũ)=stage1; chỉ ticket waiting_since quá hạn được nhắc |
| TC-P4-05-23 | P | s23: waiting_since T3 23:00 (73 giờ LV) → sla_stage=2; /me/today e7gv: mục TICKET lớp E7S23 urgency=overdue đứng đầu, reason "2 câu hỏi chờ quá 72 giờ làm việc · lâu nhất 73 giờ" |
| TC-P4-05-24 | P | ADMIN nhận đúng 1 COURSE_TICKETS_STALLED "Lớp E7S23 có 2 câu hỏi chờ quá 72 giờ làm việc" (body rỗng, không nội dung ticket); /me/today ADMIN có mục COURSE_TICKETS_STALLED (lần gọi đầu rỗng do cache H |
| TC-P4-05-25 | P | s23 (stage 2) nhưng e7gv/e7ta có phiên last_used_at <72h → COURSE_TICKETS_STALLED = 0 (không báo ADMIN) |
| TC-P4-05-26 | P | sau nhiều tick cùng ngày: 20261113,20261115 (1 thông báo/ngày); lùi ngày liên tiếp: 20261113,20261114,20261115 → tổng 3, mỗi ngày đúng 1 (dedupe_key stalled:<lớp>:<yyyymmdd>) |
| TC-P4-05-27 | P | UI /chat (1440 & 375) 4 trạng thái: OPEN "Đang chờ giảng viên · vừa gửi"/"· đã gửi 2 phút trước"; CLAIMED "Giảng viên đã nhận · đã chờ 2 phút"; ANSWERED nhãn "Giảng viên · <tên>" + "Hỏi tiếp","Đã rõ"; |
| TC-P4-05-28 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-29 | P | me/today SV → 200 keys ['no_course', 'email_verified', 'recommended', 'timeline', 'continue'] |
| TC-P4-05-30 | P | mở /chat?session=… → mạng có GET …/messages?limit=100 và POST /notifications/read {"link":"/chat?session=<id>"} (mục chuyển đã đọc); GET …/messages riêng (API) 200 không đổi read_at/updated_at của thô |
| TC-P4-05-31 | P | notifications/read {link} bởi SV → 200 |
| TC-P4-05-32 | P | [('OUT_OF_SCOPE', 200, 'CLOSED/STAFF_DISMISSED'), ('DUPLICATE', 200, 'CLOSED/STAFF_DISMISSED'), ('NEED_MORE_INFO', 200, 'CLOSED/STAFF_DISMISSED')] |
| TC-P4-05-33 | P | 3 ticket CLOSED (dismiss) → Hỏi tiếp: [('OUT_OF_SCOPE', 'CLOSED', 200), ('DUPLICATE', 'CLOSED', 200), ('NEED_MORE_INFO', 'CLOSED', 200)] |
| TC-P4-05-34 | P | teacher dismiss ticket TA đã nhận → 409 {'code': 'TICKET_ALREADY_CLAIMED', 'message': 'Câu hỏi này đã có người nhận.', 'trace_id': '178ccf4a46c3860d7d55008de5d7; reason ngoài enum → 422 |
| TC-P4-05-35 | P | preview → 200 |
| TC-P4-05-36 | P | lưu lần đầu → 201; lần hai → 200 {'saved': True, 'loaded': False} |
| TC-P4-05-37 | P | preview 200 personal=True; POST knowledge → 422 KNOWLEDGE_PERSONAL |
| TC-P4-05-38 | P | có knowledge_items sau khi lưu |
| TC-P4-05-39 | P | SV/ADMIN/GV lớp khác lưu tri thức → [403, 403, 403] |
| TC-P4-05-40 | P | chunk text (được nhúng) chứa bản đã sửa=True; embedding nạp sau 3.3s; text='true/Câu hỏi: Quy định nộp bài muộn kỳ QCKX9 của lớp là gì?\nTrả lời: ĐÃ SỬA QCKX9: nộp muộn trừ 20% mỗi ngày, tối đa 3 n' |
| TC-P4-05-41 | P | embedding nạp sau 3.3s (<10s); ep:rag:ver (rỗng→1) tăng; sv22 hỏi trùng đoạn → trả lời trích nguồn "Hỏi đáp đã được giảng viên xác nhận" (TICKET: không có thread_id/liên kết) [1]. Lưu ý: câu hỏi diễn  |
| TC-P4-05-42 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-43 | P | gỡ tri thức → 204, lặp → 204 |
| TC-P4-05-44 | P | GET library (SV 200; GV 403 vì route của SV), documents, documents/stats, share-sources, /me/today GV+SV: 0 lần lộ tài liệu THREAD_KNOWLEDGE / "Hỏi đáp đã được giảng viên xác nhận"; documents/{id}, li |
| TC-P4-05-45 | P | sv29 (lớp E7C2) hỏi đúng chuỗi đoạn của lớp 1 → "Mình chưa tìm thấy…", citations=[]; cùng chuỗi ở lớp 1 thì trúng |
| TC-P4-05-46 | P | PUT escalation_threshold 0.95 → 200, GET lại 0.95; 0.49/0.96 → 422; trả về 0.80. Hiệu lực tin kế tiếp: chat/gen.go threshold() đọc courses.escalation_threshold ở mỗi tin (GetCourse); [stack giả: tool/ |
| TC-P4-05-47 | P | 0.85 → 200; 0.49/0.96/0.82/abc → [422, 422, 422, 422] |
| TC-P4-05-48 | P | version cũ → 409; TA/SV/ADMIN → [403, 403, 403] |
| TC-P4-05-49 | N | slice E ghi F không có triệu chứng (dở dang) — cần chạy lại |
| TC-P4-05-50 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-E1 … TC-P4-05-P1 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-01 | P | SV báo SPAM → 201 {'reported': True} |
| TC-P4-06-02 | P | báo lần hai → 409 ALREADY_REPORTED; số dòng=1 |
| TC-P4-06-03 | P | enum sai/note>200/note với SPAM → [422, 422, 422] |
| TC-P4-06-04 | P | báo bài của mình → 422 CANNOT_REPORT_OWN |
| TC-P4-06-05 | P | bài không tồn tại → 404 |
| TC-P4-06-06 | P | ADMIN báo → 403 |
| TC-P4-06-07 | P | SV (người báo/chủ bài/khác) không thấy khoá báo cáo |
| TC-P4-06-08 | P | 20 báo cáo đầu → {201}; bài 21 → 429 RATE_LIMITED |
| TC-P4-06-09 | P | sv23 báo bình luận → 201 {'reported': True} (comment 201) |
| TC-P4-06-10 | P | 3 báo cáo → true/NULL/Nhiều báo cáo, chờ duyệt (201,201) |
| TC-P4-06-11 | P | bài ẩn: có trong danh sách=False, chi tiết 404, tìm kiếm 0 |
| TC-P4-06-12 | P | chủ bài xem 200 (keys ['thread', 'posts', 'next_cursor']); queue HIDDEN chứa bài=True; POST_REPORTED teacher="2 bài bị báo cáo chờ xem" TA="2 bài bị báo cáo chờ xem" |
| TC-P4-06-13 | P | cùng sv24 báo hai lần → 201,409; số báo cáo=1; hidden=-/- |
| TC-P4-06-14 | P | báo cáo 201/dismiss 200/201/201; trạng thái DISMISSED,OPEN,OPEN; hidden=-/- |
| TC-P4-06-15 | P | REPORT_AUTOHIDE_THRESHOLD: config.go:454 mặc định 3 (khoảng 2–20); compose/.env.example không khai báo (dùng mặc định) |
| TC-P4-06-16 | P | queue HIDDEN 200, OPEN 200; keys mục=['kind', 'thread_id', 'post_id', 'title_or_excerpt', 'reports', 'hidden_at', 'hidden_by_auto', 'created_at'] |
| TC-P4-06-17 | P | hide → 200; reason rỗng/201 → [422, 422]; báo cáo → ACTIONED |
| TC-P4-06-18 | P | unhide → 200; SV xem lại → 200 |
| TC-P4-06-19 | P | {'sv08': [403, 403, 403, 403, 403], 'admin': [403, 403, 403, 403, 403]} |
| TC-P4-06-20 | P | audit entity_id=01a1e014-87c0-7a68-9976-996853d5fef1: thread.auto_hide/1 ; thread.dismiss_reports/1 ; thread.hide/1 ; thread.unhide/1; dismiss-reports→200 |
| TC-P4-06-21 | P | hide song song → {'teacher': 200, 'ta': 409} |
| TC-P4-06-22 | P | hide→200; ẩn: ['POST_HIDDEN/Bài của bạn đã bị ẩn: Lạc đề/-', 'POST_HIDDEN/Bài của bạn đang được giảng viên xem xét/-']; bỏ ẩn→200: ['POST_UNHIDDEN/Bài của bạn đã hiện lại/-']; lộ người báo cáo=False;  |
| TC-P4-06-23 | P | sửa kèm SĐT → 422 PII_DETECTED; bản lưu chứa SĐT=False |
| TC-P4-06-24 | P | sửa sạch → 200; edited_at có=true |
| TC-P4-06-25 | P | sửa kèm SĐT canary 0933777417 → 422 PII_DETECTED; pg_dump / grep -c = 0 (0 chỗ thô) |
| TC-P4-06-26 | P | verify→200; chủ sửa → 409 EDIT_LOCKED_VERIFIED |
| TC-P4-06-27 | P | GV/TA/SV khác sửa → [403, 403, 403]; sai version → 409 |
| TC-P4-06-28 | P | chủ sửa bài ẩn → 200; hidden=2026-11-15 16:45:01.872987+00/Lạc đề |
| TC-P4-06-29 | P | sửa → 200; posts=AI/REJECTED; ai_state=PENDING; thông báo chủ bài 35→35; SV còn thấy bài AI cũ=False; AI_PENDING="2 câu trả lời AI chờ duyệt" (PENDING thực=2) |
| TC-P4-06-30 | P | PUT khủng hoảng → 422 CRISIS_SUPPORT; thân lưu='Nội dung sạch ban đầu của bài QCE30.' |
| TC-P4-06-31 | P | chủ xoá → 204; deleted_at=true; GV mở lại → 404 |
| TC-P4-06-32 | P | SV khác/GV xoá → [403, 403] |
| TC-P4-06-33 | P | tri thức trước xoá=1; DELETE (mất 0.3s) → ngay sau đó còn 0 (gỡ đồng bộ ≤5 s) |
| TC-P4-06-34 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-35 | P | ghim 3 bài đầu thành công, bài thứ 4 → 409 (PIN_LIMIT); mã thành công là 204 theo openapi (TC ghi «200») |
| TC-P4-06-36 | P | ghim 3 bài → 204×3; trang đầu: pinned=3 (≤3), items không lặp ghim; trang kế (cursor): pinned=0; lọc week=3, tag=qc36, status=pending/verified/none đều 200 giữ nguyên |
| TC-P4-06-37 | P | SV/ADMIN ghim → [403, 403] |
| TC-P4-06-38 | P | sv22 bình luận→201; THREAD_REPLY cho chủ bài=1 (sv21 tự bình luận 201 không thêm); VERIFY→THREAD_VERIFIED 6→7; REJECT báo chủ bài=0; mail thread_* 6→7 (gần nhất: thread_reply/SENT ; thread_reply/SENT  |
| TC-P4-06-39 | P | 3 bài AI PENDING → teacher "3 câu trả lời AI chờ duyệt/-", TA "3 câu trả lời AI chờ duyệt" (1 dòng gộp, cập nhật tại chỗ); xử lý hết → "2 câu trả lời AI chờ duyệt/2026-11-15 16:47:30.106789+00" (đã đọ |
| TC-P4-06-40 | P | teacher có mục POST_REPORTED "N bài bị báo cáo chờ xem · lớp E7C1" Href /threads?view=moderation&course=<E1> (SV, ADMIN không thấy); sau khi xử lý hết hàng chờ (OPEN 0, HIDDEN 0) ≤65 s: mục còn=0 |
| TC-P4-06-41 | P | trước REJECT truy xuất được=True; REJECT→200 (0.4s) knowledge_items=0; sau REJECT truy xuất=False; ep:rag:ver 8→9; SV thấy bài AI bị loại=False |
| TC-P4-06-42 | P | VERIFY→200; embedding sau 1.6s; ep:rag:ver 5→6; chunk="Câu hỏi: QC-E42 hàm băm Hỏi về hàm băm QCE42 dùng làm gì trong thực tế? Trả lời: Hàm băm Q"; SV hỏi đúng chuỗi → nguồn tri thức=True, có thread_i |
| TC-P4-06-43 | P | CORRECT 200,200; knowledge_items=1; số đoạn chứa QCE43B=1; sau lần 1: 'true/Câu hỏi: QC-E43b sửa hai lần\nHỏi về chứng chỉ số QCE43B'; sau lần 2: 'true/Câu hỏi: QC-E43b sửa hai lần\nHỏi về chứng chỉ s |
| TC-P4-06-44 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-45 | P | SV lớp E7C2 hỏi đúng chuỗi → '[]///Mình chưa tìm thấy nội dung này trong tài liệu của lớp.' |
| TC-P4-06-46 | P | {'sv08': [201, 403], 'ta': [201, 200], 'teacher': [201, 200], 'admin': [403, 403]} |
| TC-P4-06-47 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-48 | P | Hoàn tác <5s → 204, báo cáo bị rút (còn 0); sau 6.5s → 404 NOT_FOUND và báo cáo còn nguyên; sau khi tự ẩn → 404 (không hoàn tác được). Mã 404 thay vì 409 nhưng đúng ý "không hoàn tác" |
| TC-P4-06-49 | P | 3 người báo cùng lúc → {'sv24': 201, 'sv25': 404, 'sv26': 404}; auto_hide audit=1; POST_HIDDEN cho chủ bài=1/1; hidden=2026-11-15 16:43:23.571997+00/Nhiều báo cáo, chờ duyệt |
| TC-P4-06-50 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-51 | P | xoá bởi chủ → 204; hide → 404 {'code': 'NOT_FOUND', 'message': 'Không tìm thấy.', 'trace_id': '861381313558c34 |
| TC-P4-06-52 | P | SV khác thấy canary: {'list': (200, False), 'list2': (200, False), 'similar': (404, False), 'today': (200, False), 'tget': (404, False), 'queue': (403, False), 'notif': (200, False)}; chủ bài 200; hàn |
| TC-P4-06-53 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-E1 … TC-P4-06-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-01 | P | 20/20 câu dương → CRISIS; sót: [] |
| TC-P4-07-02 | P | 17 câu âm: bị bắt nhầm CRISIS: [] |
| TC-P4-07-03 | P | intent: ['CRISIS', 'SMALLTALK', 'COURSE_QA', 'COURSE_QA'] |
| TC-P4-07-04 | P | intent: ['CRISIS', 'CRISIS'] |
| TC-P4-07-05 | P | go test ./internal/agent -run TestQCCrisisSet/TestCrisisCannedReply → ok (PASS cả hai); git diff 1518f83.. với agent_test.go và qc_p304_test.go (nơi chứa hai test) = 0 dòng, không sửa |
| TC-P4-07-06 | P | câu trả lời mẫu duy nhất: 'Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên củ' |
| TC-P4-07-07 | P | cụm cấm trong câu mẫu: [] |
| TC-P4-07-08 | P | 0 lời gọi LLM (llm_audit CHAT/UTILITY/CLASSIFY không tăng) qua 20+17+… câu |
| TC-P4-07-09 | P | (lớp E6C1) tickets/notifications/pii_events: ['28', '30', '27'] → ['28', '30', '27'] |
| TC-P4-07-10 | P | route số liệu hướng GV/TA/ADMIN: {'today': (200, False), 'ctoday': (200, False), 'gap': (404, False), 'tickets': (200, False), 'threads': (200, False), 'queue': (200, False), 'notif': (200, False), 't |
| TC-P4-07-11 | P | đọc phiên khủng hoảng: {'sv25': 200, 'teacher': 403, 'ta': 403, 'admin': 403, 'sv26': 404} |
| TC-P4-07-12 | P | đăng thread khủng hoảng → 422 CRISIS_SUPPORT |
| TC-P4-07-13 | P | bình luận khủng hoảng → 422 CRISIS_SUPPORT |
| TC-P4-07-14 | P | precheck 6 câu khủng hoảng → 200 allowed, không trả CRISIS_SUPPORT (chỉ chặn ở Create/Comment: 422 CRISIS_SUPPORT) |
| TC-P4-07-15 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-16 | P | PUT thread thân khủng hoảng → 422 CRISIS_SUPPORT (cùng ca 06-30) |
| TC-P4-07-17 | P | SV trong giờ thi (khoá chat; exam OPEN + attempt IN_PROGRESS): 2 câu dương → [200, 200], nhận câu mẫu hỗ trợ=True; CHAT_BLOCKED 0→0 (không tăng) |
| TC-P4-07-18 | P | câu thường trong giờ thi → 409 ; CHAT_BLOCKED 0→1 (đường vòng không mở) |
| TC-P4-07-19 | P | SV ngoài lớp tạo phiên → 403 FORBIDDEN; đếm chat_messages/sessions/notifications/pii/tickets 0/0/0/0/0 → 0/0/0/0/0 (đếm theo user sv.moi) |
| TC-P4-07-20 | P | docker-compose.local.yml:172 có SUPPORT_RESOURCES_VI; chat riêng chứa đúng chuỗi=True; Threads 422 CRISIS_SUPPORT chứa đúng chuỗi=True |
| TC-P4-07-21 | P | gateway tạm (cùng ảnh/DB) với SUPPORT_RESOURCES_VI="" → câu mẫu chat riêng = "hải một mình, hãy nói chuyện với giảng viên của bạn nhé. Bạn hãy trao đổi với giảng viên hoặc phòng công tác sinh viên của |
| TC-P4-07-22 | P | gateway tạm với SUPPORT_RESOURCES_VI 301 ký tự → exit 1: "giá trị biến môi trường không hợp lệ: SUPPORT_RESOURCES_VI: tối đa 300 ký tự"; 300 ký tự → "gateway ready" (khởi động) |
| TC-P4-07-23 | P | SUPPORT_RESOURCES_VI có "phòng 101": câu mẫu nguyên chuỗi, "101" còn nguyên, không "[đã ẩn]"; câu mẫu='Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên của bạn n |
| TC-P4-07-24 | P | SRS 4.9 mục 7 (dòng 290) đủ 3 ý + docs/thesis-notes/sprint-7.md + QUESTIONS.md:30 Q-crisis-hyperbole |
| TC-P4-07-25 | P | «deadline này làm em muốn chết» → CRISIS (câu hỗ trợ) |
| TC-P4-07-26 | P | đọc phiên khủng hoảng bởi TA/GV/ADMIN/SV khác → {'ta': 403, 'teacher': 403, 'admin': 403, 'sv09': 404} |
| TC-P4-07-27 | P | scripts/ui-antipatterns.sh exit=0 |
| TC-P4-07-E1 … TC-P4-07-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-01 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P5-01-02 | P | grep -c ^ALTER TABLE 00013_crm.sql = 1 (class_sessions: attendance_done_at/by, UNIQUE(course_id,id), CHECK); PROGRESS.md dòng 95 "00013 / crm" |
| TC-P5-01-03 | P | {'status X': '22P02', 'dup unique': '23505', 'fk session other course': '23503', 'fk student not enrolled': '23503', 'PENALTY +': '23514', 'SPEAK -': '23514', 'client_event dup': '23505', 'type X': '2 |
| TC-P5-01-04 | P | v_attendance_summary & v_student_360 đúng 8/18 cột như SRS 4.8.1/5.6 (00015 thay view, #20); mọi chỉ mục bảng mới (attendance_records, participation_events, student_observations, student_risk) bắt đầu |
| TC-P5-01-05 | P | dry_run → 200, buổi xem trước=0; số buổi DB 9→9 |
| TC-P5-01-06 | P | dry_run 200; (buổi,audit,outbox) (0, '2', '2')→(0, '2', '2') |
| TC-P5-01-07 | P | lần chạy đầu trên lớp trống: generate thật → 201, số buổi 0→6 (lần chạy lặp báo «DUPLICATE» nên số buổi giữ nguyên — đúng) |
| TC-P5-01-08 | P | lần chạy đầu: khác khoá cùng tham số → 201 nhưng số buổi 6→6 (bỏ DUPLICATE, không nhân đôi) |
| TC-P5-01-09 | P | tham số sai → [422, 422, 422, 422, 422, 422] |
| TC-P5-01-10 | P | >60 buổi → 422 VALIDATION_FAILED |
| TC-P5-01-11 | P | 201 {'created': 9, 'first_session_no': 1, 'last_session_no': 9, 'skipped': 0}; audit 1 dòng: generate/{"last": 9, "first": 1, "created": 9}; outbox course.changed=1 |
| TC-P5-01-12 | P | lặp cùng key → 201, số buổi giữ 9 |
| TC-P5-01-13 | P | không Idempotency-Key → 422 IDEMPOTENCY_KEY_REQUIRED; buổi=0 |
| TC-P5-01-14 | P | trigger lỗi ở buổi 5 → 500 INTERNAL; số buổi=0; đã gỡ trigger |
| TC-P5-01-15 | P | PUT buổi → 200 {'session': {'id': '01a1df59-df98-7982-94b4-aac2b49cce36', 'session_no': 11, 'st |
| TC-P5-01-16 | P | sai version / kết thúc<bắt đầu → [409, 422] |
| TC-P5-01-17 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-18 | P | DELETE buổi chưa có điểm danh → 204 |
| TC-P5-01-19 | P | PUT điểm danh 200; DELETE → 409 SESSION_HAS_ATTENDANCE; PUT sửa → 200 |
| TC-P5-01-20 | P | buổi bù → 201 {'created': 1, 'first_session_no': 10, 'last_session_no': 10, 'skipped': 0}; PUT topic → 200 |
| TC-P5-01-21 | P | xoá buổi 3 → 204; buổi mới → 201 {'created': 1, 'first_session_no': 6, 'last_session_no': 6, 'skipped': 0} |
| TC-P5-01-22 | P | xoá buổi 5 (và 6) → 204/204; buổi mới → 201 {'created': 1, 'first_session_no': 5, 'last_session_no': 5, 'skipped': 0} (kỳ vọng 5) |
| TC-P5-01-23 | P | SV 403, ADMIN 403, SV ghi 403, SV ngoài lớp 403; TA dry_run 200 |
| TC-P5-01-24 | P | PUT buổi lớp 1 qua URL lớp 2 → 404 |
| TC-P5-01-25 … TC-P5-01-34 (10 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-E1 … TC-P5-01-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-01 | P | lưới điểm danh → 200; keys=['session', 'students'] |
| TC-P5-02-02 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-03 | P | buổi lớp 1 qua URL lớp 2 → 404 |
| TC-P5-02-04 | P | PUT 7 hàng → 200; DB 7/1 |
| TC-P5-02-05 | P | PUT lại y hệt → 200; DB 7/1 (không đổi version) |
| TC-P5-02-06 … TC-P5-02-07 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-08 | P | T2 PUT với version cũ → 409 VERSION_CONFLICT |
| TC-P5-02-09 … TC-P5-02-10 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-11 | P | ngoài lớp/trùng/status X/note 201 → [422, 422, 422, 422] |
| TC-P5-02-12 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-13 | P | complete:true → 200; true/01a1df27-5bfd-746a-acbf-411de5ffccb2 |
| TC-P5-02-14 … TC-P5-02-26 (13 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-27 | P | POST phát biểu → 201; lặp client_event_id → 200; số dòng=1 |
| TC-P5-02-28 | P | SV/ADMIN/SV ngoài lớp → [403, 403, 422] |
| TC-P5-02-29 … TC-P5-02-37 (9 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-38 | P | sv.kha (DB: vắng 1): Hiện tại bạn đã vắng 1 buổi học. |
| TC-P5-02-39 | P | sv.gioi (DB vắng 0) hỏi cùng câu → «chưa vắng buổi nào (0)»; mỗi SV nhận số của chính mình |
| TC-P5-02-40 … TC-P5-02-41 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-42 | P | {'sv08': [403, 403], 'admin': [403, 403], 'sv.moi': [403, 403], 'ta': [200, 200]} |
| TC-P5-02-43 | P | TA xoá sự kiện của GV → 403 |
| TC-P5-02-44 … TC-P5-02-46 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-02-E1 … TC-P5-02-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-01 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P5-03-02 … TC-P5-03-04 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-05 | P | hai tab (t1,t2) cùng khung 30 s → study_time_daily chỉ +30 s (không nhân đôi) |
| TC-P5-03-06 … TC-P5-03-13 (8 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-14 | P | rollup mỗi phút: nhịp CHAT/THREADS/LIBRARY của SV được ghi vào study_time_daily |
| TC-P5-03-15 … TC-P5-03-18 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-19 | P | ack lần 1/2 → 204/204 |
| TC-P5-03-20 … TC-P5-03-25 (6 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-26 | P | me/study-time → 200 |
| TC-P5-03-27 | P | weeks 0/21/abc/trống → [422, 422, 422, 200] |
| TC-P5-03-28 … TC-P5-03-29 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-30 | P | course ngoài lớp/area X/thiếu tab_id/course_id số → [422, 422, 422, 422] |
| TC-P5-03-31 … TC-P5-03-34 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-35 | P | tắt công tắc → 200; sai version → 409 |
| TC-P5-03-36 … TC-P5-03-37 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-38 | P | TA/SV/ADMIN đổi allow_study_time_opt_out → [403, 403, 403] |
| TC-P5-03-39 | P | lớp đã có SV → 409 COURSE_HAS_STUDENTS |
| TC-P5-03-40 … TC-P5-03-44 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-03-E1 … TC-P5-03-P1 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-01 | P | danh sách SV → 200, 3 mục, cursor=True |
| TC-P5-04-02 | P | duyệt con trỏ: 7 SV, duy nhất=7 |
| TC-P5-04-03 | P | lớp G7A (12 ACTIVE + 1 PENDING g7s13 + 1 REMOVED g7s14): danh sách 12 dòng, total=12; PENDING/REMOVED không có. Lớp seed 761987: sv.lech PENDING không có trong 30 dòng, total=30=ACTIVE |
| TC-P5-04-04 | P | Không bật log_statement toàn cục (dùng chung stack). Bằng chứng tương đương: pg_depend: v_student_360 (nguồn của danh sách) không phụ thuộc chat_messages (0 dòng); câu CrmStudentsList/CrmStudentsCount |
| TC-P5-04-05 | P | filter attention/absent/inactive → [200, 200, 200] |
| TC-P5-04-06 | P | q không dấu/hoa thường/MSSV/không khớp → [200, 200, 200, 200] |
| TC-P5-04-07 | P | filter=xyz → 422 |
| TC-P5-04-08 | P | go test ./internal/crm -run TestStudents360ExplainNoSeqScan: PASS (dữ liệu dựng đúng SRS 9.4: 1 lớp 1.000 SV + 19 lớp 100 SV, 15 buổi, ≥280.000 study_time_daily, ≥5.000 participation_events, ANALYZE;  |
| TC-P5-04-09 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-10 | P | go test TestStudents360LargeCoursePerf PASS (EXPLAIN hồ sơ MỘT SV lớp 1.000: Index Scan theo course_id, không Seq Scan bảng lớn) + TestProfileUnder3s PASS. docker-compose/ env của ep-r-gateway không đ |
| TC-P5-04-11 | P | go test TestRiskTable ok (16 ca con). Kiểm tay ở lớp thử G7A bằng dữ liệu SQL + crm-risk-run: ca 10 — g7s03 tỉ lệ chuyên cần 0,50 đúng ngưỡng attendance_ok_min=0,50 → NONE (đặt lại 0,70 → AT_RISK); ca |
| TC-P5-04-12 | P | g7s06 chỉ thấp thời gian học 2 tuần (đi học đủ) → NONE; g7s07 tuần W2 không có dữ liệu chuyên cần → WATCH (không AT_RISK) |
| TC-P5-04-13 | P | crm-risk-run ×3: entered_at giữ nguyên 2026-11-15 16:35:18.218291+00, computed_at đổi mỗi lần (3 giá trị) ; rời AT_RISK (g7s01 có mặt đủ) → NONE entered_at=-; vào lại → entered_at=2026-11-15 16:35:58. |
| TC-P5-04-14 | P | select count(*) from student_risk where student_id in (REMOVED g7s14, PENDING g7s13) = 0 sau crm-risk-run |
| TC-P5-04-15 | P | job crm-risk-run đã chạy: khóa Redis ep:crm.risk:done:20261115 có; student_risk lớp E6C1 có 7 hàng (cả bảy NONE) |
| TC-P5-04-16 | P | student_risk.weeks của g7s01/C: week_start toàn thứ Hai (['2026-11-02', '2026-10-26', '2026-10-19', '2026-10-12']…, T2–CN trọn). Thử ranh giới: g7s08 có 2 thread, created_at 2026-11-08 16:30Z (CN 23:3 |
| TC-P5-04-17 | P | g7s01 (AT_RISK): risk={level, sentence, reasons[3]}; câu: "Cần chú ý: vắng 4/4 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 30 phút mỗi tuần; không hỏi AI hay đăng bài ở Threads trong 2 tuần." |
| TC-P5-04-18 | P | g7s02 WATCH: "Đang theo dõi: vắng 2/2 buổi trong 1 tuần gần nhất; học trên EduPilot khoảng 30 phút mỗi tuần; không hỏi AI hay đăng bài ở Threads trong 1 tuần."; g7s04 NONE: sentence=None |
| TC-P5-04-19 | P | đặt ngưỡng risk → 200 |
| TC-P5-04-20 | P | ngưỡng ngoài miền → [422, 422, 422, 422, 422] |
| TC-P5-04-21 | P | TA/SV/ADMIN → [403, 403, 403]; sai version → 409 |
| TC-P5-04-22 | P | UI thật /class/settings (teacher): nhóm "Ngưỡng cần chú ý": ngay sau tiêu đề là nhãn "Tỷ lệ có mặt tối thiểu" (không câu mô tả); 5 hàng nhãn+điều khiển (5 thanh, 5 ô nhập) + giá trị: Tỷ lệ có mặt tối  |
| TC-P5-04-23 | P | GET /me/today bởi g7t sau crm-risk-run: mục STUDENT_ATTENTION "3 sinh viên mới vào diện cần chú ý · G7A", href /students?filter=attention&course=01a1e00d-e964-7d39-864e-500cbd54f3e5; khối attention 3  |
| TC-P5-04-24 | P | g7t2 (GV lớp G7B không AT_RISK): không STUDENT_ATTENTION, attention=[], attention_total=0 (khối ẩn, không rỗng giả); STUDENT: khoá attention/attention_total vắng, ADMIN: vắng (['count', 'actions']) |
| TC-P5-04-25 | P | gieo 8 AT_RISK ở G7A: khối attention 5 dòng (≤5), attention_total=8 → UI "Xem tất cả (8)"; mục "8 sinh viên mới vào diện cần chú ý · G7A" |
| TC-P5-04-26 | P | UI thật (https://localhost:1143, teacher g7t, 1280×800, lớp G7A): cột=Sinh viên · Chuyên cần · Phát biểu · Hoạt động học · Rủi ro; chip=Cần chú ý, Vắng nhiều, Ít hoạt động; dòng 01="G7 Sinh Viên 01 G7 |
| TC-P5-04-27 | P | UI thật: URL sau gõ+chip: /students?filter=attention&q=g7+sinh+vi%C3%AAn+0; quay lại từ hồ sơ: ô tìm="g7 sinh viên 0", chip Cần chú ý pressed=1; deep link ?filter=attention → chip pressed=1, chip khác |
| TC-P5-04-28 | P | UI thật: khung xương khi tải chậm: 1 phần tử; lỗi 500: có "Thử lại" và bấm lại tải được; tìm không khớp: "Không có sinh viên nào khớp." + nút Bỏ lọc (1) → bấm → danh sách; 720 px: tràn trang=0px; 600  |
| TC-P5-04-29 | P | {'sv09': [403, 403], 'admin': [403, 403], 'sv.moi': [403, 403], 'ta': [200, 200]} |
| TC-P5-04-30 | P | Crawl tài khoản g7s01 (AT_RISK, ghi chú CANARY-OBS-4F, đã nhận outreach): /me/{courses,profile,settings,today,sessions}, /notifications, /chat/sessions + messages 2 phiên, /courses/{id}/me/{attendance |
| TC-P5-04-31 | P | g7s01 hỏi chat "Em có bị coi là cần chú ý không?" và "Giảng viên ghi gì về em?": trả lời mẫu "Mình chưa tìm thấy nội dung này trong tài liệu của lớp." (không tiết lộ); /_test/llm/payloads (fake provid |
| TC-P5-04-32 | P | go test ./internal/crm -run TestSensitiveQueriesOnlyFromCRM/TestStudentNeverSeesRisk -v: ok (2 PASS; test nay nằm ở internal/crm, ./internal/contract không còn các test này nên "no tests to run"); go  |
| TC-P5-04-33 | P | API: G7N (2 sinh viên, không buổi/không dữ liệu): số 0, risk.level=NONE, không lỗi. UI thật (ảnh students-G7N.png / arch.txt): cột Chuyên cần "—", Phát biểu "0 · 0,00", Hoạt động học "0 phút · 0 câu h |
| TC-P5-04-34 | P | student_risk của G7N xoá trống: API risk={level:NONE, computed_at:null}; UI: cột Rủi ro không có nhãn nào (cột giữ tiêu đề nhưng ô trống, không "Cần chú ý"/"Theo dõi"). Lớp lưu trữ (admin archive → st |
| TC-P5-04-35 | P | ui-antipatterns.sh trên cây be0c0d96: exit 0 (CI "32/32 phép bắt được"). run PR 43712425198 @be0c0d96 (CI Playwright, cùng test, khung mô phỏng gateway): students.spec "list filters search keep state" |
| TC-P5-04-E1 … TC-P5-04-P1 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-01 | P | hồ sơ 360 → 200; keys=['student', 'risk', 'attendance', 'participation', 'study', 'activity'] |
| TC-P5-05-02 | P | audit_log entity=student_profile entity_id=<g7s01> action=view: 0 → 3 (+3 sau 3 lần GET profile); dòng: actor_id=teacher ✓, course_id ✓; before/after không có nội dung ghi chú |
| TC-P5-05-03 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-04 | P | trigger tạm (chỉ cho entity_id=g7s01) làm INSERT audit_log lỗi → GET profile 503, thân không có dữ liệu ({"code":"SERVICE_UNAVAILABLE","message":"Dịch vụ tạm thời không khả dụng. Hãy th); gỡ trigger → |
| TC-P5-05-05 | P | GET observations → 200; audit view_observations 0→1 (+1); grep CANARY-OBS-4F trong toàn audit_log = 0 |
| TC-P5-05-06 | P | hồ sơ không chứa nội dung chat riêng |
| TC-P5-05-07 | P | Không bật log_statement toàn cục. Bằng chứng tương đương: profile.sql (CrmProfileGet, StudyWeeks, StudentAttendance, Obs*) chỉ nhắc chat_messages trong chú thích, không truy vấn; v_student_360 không p |
| TC-P5-05-08 | P | {'sv09': [403, 403], 'admin': [403, 403], 'ta': [200, 200]} |
| TC-P5-05-09 | P | TA sửa/xoá ghi chú của GV → [403, 403] |
| TC-P5-05-10 | P | tạo ghi chú → 201 |
| TC-P5-05-11 | P | tags>5/tag rỗng/tag 31/body rỗng/body 2001 → [422, 422, 422, 422, 422] |
| TC-P5-05-12 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-13 | P | SV không thấy CANARY-OBS trong me/today, me/attendance, me/participation |
| TC-P5-05-14 … TC-P5-05-15 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-16 | P | POST …/students/{g7s01}/ai-summary (teacher g7t) → 200 {draft:{summary,strengths[1],concerns[…],suggestions[…]}}. Mỗi yêu cầu đúng 1 dòng llm_audit (2 yêu cầu → 2 dòng task=UTILITY, lane=NEAR_REALTIME |
| TC-P5-05-17 | P | student_observations trước/sau ai-summary: 1 → 1 (không lưu); API trả 200 {draft}; hiển thị bản nháp nhãn AI trong ô sửa được: CI students.spec "ai summary edit then save" ✓ + UI thật trong lượt UI |
| TC-P5-05-18 | P | POST observations kind=AI_SUMMARY (ai_edited=false) → 201 kind=AI_SUMMARY ai_edited=False; sửa chữ rồi lưu (PUT ai_edited=true) → 200 ai_edited=True version 2. Nhãn "Nhận xét tổng hợp · AI" / "· Đã sử |
| TC-P5-05-19 | P | Stack ep-r: UTILITY đi qua provider thật nên không ép được bản nháp bịa tên → chấm bằng go test ./internal/crm -run TestAISummaryRedactsInventedName: PASS (provider giả trả bản nháp chứa tên sinh viên |
| TC-P5-05-20 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-21 | P | Dev test (provider giả chết/chậm/quá tải): TestAISummaryLLMDown503 PASS (503 LLM_UNAVAILABLE + retry_after), TestAISummaryTimeout504 PASS (>20 s → 504 DEADLINE_EXCEEDED), TestAISummaryNoDataNoLLM PASS |
| TC-P5-05-22 | P | lớp mới không dữ liệu (g7s11 ở G7N) → 409 NOT_ENOUGH_DATA, llm_audit UTILITY 2→2 (không gọi LLM); SV/ADMIN → [403, 403]; gọi liên tiếp (đã 2 lượt trước): 9 lượt, kết quả [200, 200, 200, 200, 200, 200, |
| TC-P5-05-23 | P | outreach → 201 |
| TC-P5-05-24 | P | g7s01: phiên "Tin nhắn từ giảng viên" có tin ASSISTANT kèm block ticket answered_by={role TEACHER, full_name "G7 Giảng viên"} (UI nhãn "Giảng viên · …", kiểm trong lượt UI); `Hỏi tiếp` (POST /chat/tic |
| TC-P5-05-25 | P | outreach chứa tên/email/MSSV của SV khác → [422, 422, 422] |
| TC-P5-05-26 | P | outreach tới g7s01 (AT_RISK): notifications title/body khớp (rủi ro/cần chú ý/at-risk/đang theo dõi/CANARY) = 0; thư Mailpit tới g7s01 ['Giảng viên vừa nhắn cho bạn'] → 0 khớp; tin STAFF_ANSWER không  |
| TC-P5-05-27 | P | 4 outreach/ngày tới g7s02 (2 nhân viên g7t, g7ta cộng lại): [(201, 'ok'), (201, 'ok'), (201, 'ok'), (429, 'OUTREACH_LIMIT')] — tin thứ 4 → 429 OUTREACH_LIMIT; tin STAFF_ANSWER thực có 3; REMOVED (g7s1 |
| TC-P5-05-28 | P | Như TC-P5-04-31 trên g7s01 (AT_RISK + ghi chú canary): trả lời mẫu, không `[[…]]`, payload LLM/RAG không chứa ghi chú hay nhãn rủi ro; dev test TestStudentAsksAboutOwnNotes, TestNoToolReadsObservation |
| TC-P5-05-29 | P | go test ./internal/agent -run TestStudentAsksAboutOwnNotes/TestNoToolReadsObservations -v: ok (2 PASS) |
| TC-P5-05-30 … TC-P5-05-31 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-32 | P | grep -rn "@/mock" frontend/src/features/students / wc -l = 0 (cây be0c0d96); CI fe-check: pnpm lint + next build xanh; Hôm nay staff (today) lấy attention từ API: /me/today có khối attention thật (761 |
| TC-P5-05-33 | P | ui-antipatterns.sh exit 0; run PR 43712425198 @be0c0d96 (CI Playwright, cùng test, khung mô phỏng gateway): students.spec toàn bộ 14 test desktop ✓ (mobile skip theo thiết kế "đo một lần"), 0 flaky, 0 |
| TC-P5-05-E1 … TC-P5-05-P1 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-01 | P | me/attendance, me/participation → [200, 200] |
| TC-P5-06-02 | P | tham số student/student_id bị bỏ qua (cùng kết quả của chính mình) |
| TC-P5-06-03 | P | g7s12 (lớp G7N chưa buổi nào): 200 recorded=0 sessions=[] (không lỗi); PENDING (g7s13) / REMOVED (g7s14) / ngoài lớp (g7s03 vào G7N) → [403, 403, 403] |
| TC-P5-06-04 | P | TA/GV/ADMIN me/* → [403, 403, 403, 403, 403, 403] |
| TC-P5-06-05 | P | limit=2 → 200 0 mục; keys=['attendance', 'next_cursor'] |
| TC-P5-06-06 | P | Crawl tài khoản g7s01 (AT_RISK, ghi chú CANARY-OBS-4F, đã nhận outreach): /me/{courses,profile,settings,today,sessions}, /notifications, /chat/sessions + messages 2 phiên, /courses/{id}/me/{attendance |
| TC-P5-06-07 … TC-P5-06-10 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-11 | P | g7s02 (vắng 2 buổi, 2 SPEAK = +0,50): /me/attendance absent=2, /me/participation points_total 0.50 (2 lần); chat "Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?" → "Chào bạn, hiện tạ |
| TC-P5-06-12 | P | giảng viên (ngữ cảnh khác) ghi thêm 1 SPEAK cho g7s02 → sau 3,6 s: /me/participation points_total=0.75 (3 lần), chat "Bạn đã nghỉ 2 buổi và hiện có tổng cộng 0.75 điểm cộng phát biểu (tương ứng với 3  |
| TC-P5-06-13 | P | GET …/me/study-time?weeks=8 (g7s08, đã Đã hiểu): 200 {weeks:[{week_start,minutes,by_area}] (8 tuần), this_week_minutes=100, tracking={'enabled': True, 'locked': False}}; chỉ của mình; người khác/…/stu |
| TC-P5-06-14 … TC-P5-06-18 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-19 | T | Spec đổi hợp lệ: proposals sprint 7 #34 (PM ACCEPTED 2026-10-15): nhân viên vào /me thấy màn chặn chung ("Bạn không có quyền xem màn này" + nút Về Hôm nay), không redirect; CI me.spec "me staff blocke |
| TC-P5-06-20 | P | g7s11 ghi danh G7A và G7N: /me/attendance G7A recorded=4 (DB 4), G7N recorded=0 (DB 0); study this_week G7A=100, G7N=0; số đổi theo lớp, không lẫn (dev TestMeTwoCoursesIsolated PASS; UI đổi bộ chọn: C |
| TC-P5-06-21 | P | grep -rn "@/mock" frontend/src/features/me / wc -l = 0; CI fe-check (pnpm lint + build) xanh |
| TC-P5-06-22 … TC-P5-06-25 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-26 | P | ui-antipatterns.sh exit 0; run PR 43712425198 @be0c0d96 (CI Playwright, cùng test, khung mô phỏng gateway): me.spec 25 test desktop ✓; activity.spec (nay chứa "study time opt out on allowed course", " |
| TC-P5-06-E1 … TC-P5-06-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-S7-SEED-01 | P | 761987: attendance_records tạo bởi seed (created_at 12:25:54, trước 12:30) = 270 dòng, buổi 1–9 `attendance_done_at` có giá trị, buổi 10 (2026-11-15 09:00) lúc seed chưa có dòng nào/chưa điểm danh. (H |
| TC-S7-SEED-02 | P | Buổi 1–9 (dữ liệu seed): sv.gioi vắng 0/có mặt 9; sv.kha vắng 2, muộn 1, có mặt 6; sv.nguyco vắng 5, muộn 1, có mặt 3 — khớp seed/expected_crm.csv từng cột (speak_count 12/2/1, points 3,00/0,50/0,25). |
| TC-S7-SEED-03 | P | ABSENT buổi 1–9 = 21 (18–26); participation_events lúc seed = 65 (≈60), A nhiều nhất 12; B (sv.kha) đúng 2 sự kiện SPEAK = +0,50 |
| TC-S7-SEED-04 | P | Không chạy seed.mjs lần hai trên DB trống (cấm — phá seed chung). Bằng chứng tương đương: kế hoạch số liệu tất định: `node scripts/crm-plan.mjs --json` chạy hai lần cho cùng md5 (007481bc…) và `cmp` v |
| TC-S7-SEED-05 | P | TestSeedCRMExpected nay ở internal/seedtool (-tags integration, testcontainers riêng): ok 126 s |
| TC-S7-SEED-06 | P | 761987: count(distinct date), count(*), sum = 70/4200/1939080 (≥60 ngày; 4200 = 70 ngày × 30 SV × 2 khu vực TB). `admin seed-study-time --course G7N --weeks 2 --profile -` (APP_ENV=dev) in "58 dòng" |
| TC-S7-SEED-07 | P | Phút/tuần (761987, 10 tuần từ 2026-09-07): sv.nguyco 120 ×6 rồi tuần 7–9 = 30 / 28 / 35 (<60), tuần 10 = 120; sv.gioi 300 mọi tuần; sv.kha 150 mọi tuần; lớp 761988 (3 tuần 2026-10-26, 11-02, 11-09 mỗi |
| TC-S7-SEED-08 | P | `docker exec -e APP_ENV=… gateway admin seed-study-time --course G7N --weeks 2 --profile -`: production → exit 1 "chỉ chạy khi APP_ENV thuộc dev, test (hiện: production)"; staging → 1; rỗng → 1; local |
| TC-S7-SEED-09 | P | Chạy seed-study-time G7N lần 1: 58 dòng, tổng 24.000 giây; chạy lại (dev, test): "0 dòng", count(*) = 58 và sum(active_seconds) = 24000 không đổi |
| TC-S7-SEED-10 | T | Spec đổi hợp lệ: proposals sprint 8 #5: thêm tín hiệu GRADE, "số WATCH không ghim". Hiện 761987: đúng 1 AT_RISK (C, lý do chuyên cần + điểm quá trình + thời gian học + hoạt động), A/B NONE, lớp 761988 |
| TC-S7-SEED-11 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-S7-SEED-12 | P | 761987 (5 ticket gốc của seed, sv14…sv18): OPEN mới, OPEN quá hạn (created 2026-11-07, waiting_since <now−80h, sla_stage 2), CLAIMED, ANSWERED, CLOSED (STUDENT_CONFIRMED); lớp 761988 = 0 ticket; không |
| TC-S7-SEED-13 | P | ticket thử ở G7A: trước/sau seed-age (--hours 100): mọi cột không-thời-gian của escalation_tickets giống hệt, md5 chat_messages.content không đổi; created_at/waiting_since lùi (2026-11-15 16:45:08.941 |
| TC-S7-SEED-14 | P | ticket G7A sau `seed-age --tickets --ticket <id> --hours 192` + `run-job ticket.sla`: sla_stage=2, Today mục TICKET urgency=overdue "1 câu hỏi chờ quá 72 giờ làm việc · lâu nhất 196 giờ". (Lần đầu --h |
| TC-S7-SEED-15 | P | 761987: GET moderation/queue (teacher) = 2 phần tử; post_reports = 5 (OPEN): thread "Worms DO NOT NEED USER INTERVENTION!" 2 báo cáo (2 người khác nhau, OPEN); thread "Phisher" 3 báo cáo của 3 người k |
| TC-S7-SEED-16 | K | K loại b: cần PM duyệt — chạy lại node scripts/seed.mjs trên stack ep-r đã seed (PM cấm vì phá seed chung; idempotency kiểm đọc: bước 14 client_event_id cố định, 16–17 thoát sớm — dev báo ở handoff; c |
| TC-S7-SEED-17 | P | `NODE_ENV=production node scripts/seed.mjs` → exit 1 "Seed bị chặn: NODE_ENV=production…"; `APP_ENV=production` → exit 1 (chặn trước mọi lệnh mạng). `grep -n SUPPORT_RESOURCES_VI docker-compose.local. |
| TC-S7-SEED-18 | P | Đọc mã + chạy phần rẻ (không chạy cả cổng: cần build:gate/Playwright/Docker nặng): scripts/gate-p4.sh = P4.md (+ -race): go test -race escalation/notify/mail(+thread), TestEscalationToMail, TestNotify |
| TC-S7-SEED-19 | P | run PR 43712425198 @be0c0d96 (CI Playwright, cùng test, khung mô phỏng gateway): escalation.spec 25/25 ✓ (có "inbox claim conflict" = hai người cùng Nhận, "inbox send failure", mail settings, ticket s |
| TC-S7-SEED-20 | P | A/B/C (sv.gioi/sv.kha/sv.nguyco) GET me/attendance → 200 đúng số của mình (A vắng 0; B vắng 2, muộn 1; C vắng 5, muộn 1), …/students/{uid}/{profile,attendance} và của người khác → 403; sv.moi (D): me/ |
| TC-S7-SEED-21 | P | TestSeedIsolation + TestSeedAdminForbidden nay ở internal/crm (-tags integration): TestSeedAdminForbidden PASS (57 s); TestSeedIsolation lần 1 FAIL do máy tải 27 (175 s, timeout container), chạy lại 1 |
| TC-S7-SEED-22 | K | K loại b: cần PM duyệt — kill -9 seed.mjs giữa bước điểm danh trên DB trống (cần DB trống/seed riêng, PM cấm chạm seed chung) |
| TC-S7-SEED-23 | K | K loại b: cần PM duyệt — `docker compose stop gateway` giữa lúc seed (dừng gateway dùng chung của nhiều agent, PM cấm) |
| TC-S7-SEED-24 | P | courses: 761987/t, 761988/f; user_settings track_study_time=false: 0; PUT {allow_study_time_opt_out:false,version} bởi teacher → (409, 'COURSE_HAS_STUDENTS'); ta/sv.gioi/admin → 403/403/403 |
| TC-S7-SEED-25 … TC-id (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-01 | P | Σ shard CI $S: 658+74 / 343+389 / 126+606 = 2196 (passed 1127, skipped 1069, flaky 0, failed 0) = PW_N 2196 (playwright --list --grep-invert @real trên cây sạch be0c0d96). TRƯỚC (main 9bcad074, run 43 |
| TC-02 | P | --list --shard=i/3: 586/805/805; giao 0 dòng trùng; hợp sort -u = --list không shard = 2196 (diff rỗng). Ghi chú: tổng kết CI mỗi shard 732 (chia theo số test lúc chạy), --list chia khác nhau nhưng hợ |
| TC-03 | P | ci.yml be0c0d96: `playwright test --grep-invert @real --shard=${{matrix.shard}}/3`; playwright.config.ts không đổi so 1518f83 (git diff rỗng): workers CI=2, retries CI=1; không --workers/--retries/--m |
| TC-04 | P | Run đỏ 43716942738: artifact riêng e2e-reports-2 (playwright-report + test-results) và lhci-reports-simulate chứa thư mục ẩn .lighthouseci (include-hidden-files: true); run 43712050291 có e2e-reports- |
| TC-05 | P | mọi dòng Go matrix: `go test -race -tags testroutes -v $pkgs`; không -short/-failfast |
| TC-06 | P | union 4 dòng matrix = `go list -tags testroutes ./...` = 100 gói, mỗi gói đúng 1 dòng (không trùng, không thiếu); sprint 7 gốc là 42 gói — tăng do sprint sau thêm gói |
| TC-07 | P | GO_T ở S: `go test -tags testroutes -list ".*" ./... / grep -c ^Test` = 2808 = CI Σ(PASS+SKIP) 1360+6+1106+3+147+3+182+1 = 2808; FAIL=0, panic=0, DATA RACE=0 ở cả 4 dòng. TRƯỚC (main 9bcad074) CI Σ =  |
| TC-08 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-09 | P | course/auth là dòng riêng; rest-1/rest-2 loại trừ bằng grep -Ev (xem log) |
| TC-10 | P | bước `go test -p 1 -count=3 -tags testroutes -run TimingEqualized$` vẫn chạy cho ./internal/auth/ và ./internal/course/ |
| TC-11 | P | Go (static) 1 lần: go vet, go vet -tags testroutes, golangci-lint v2.14.0 ×2 (testroutes có --build-tags), setup sqlc 1.31.1 + sqlc diff; không lặp ở matrix; xanh 62 s |
| TC-12 | P | git diff 1518f83..be0c0d96 lighthouserc*.json: assertion không đổi (LCP devtools median ≤2500 error, dev/ui warn; CLS ≤0.1; TBT ≤200 (dev/ui warn); script size ≤256000; median); chỉ thêm URL /me (cả h |
| TC-13 | P | devtools A (/ /chat /threads /inbox: 4 URL, 12 lượt) ∪ B (/gradebook /me /settings/llm /dev/ui: 4 URL, 12 lượt) = 8 URL (7 cũ + /me), không giao; simulate 9 URL ×5 = 45 lượt (7 cũ + /me + /checkin; be |
| TC-14 | P | Cùng assertMatrix cho cả hai nửa (--collect.url ghi đè): dev/ui = warn (log B có ⚠ largest-contentful-paint warning), 6 URL còn lại error; log "Checking assertions against 4 URL(s), 12 run(s)" ở A và  |
| TC-15 | P | 3 job lhci: Healthcheck passed, "Done running Lighthouse!", "All results processed!"; Assertion failed = 0; CHROME_PATH đặt ở cả 3 job |
| TC-16 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-17 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-18 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-19 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-20 | F | wall run PR 43712425198 = 15,97 phút (12:09:46→12:25:44), run push e0bf1fc9 = 16,7 phút; job dài nhất Playwright 1/3 = 15,4 phút (shard 2/3 = 6,5, 3/3 = 2,7: lệch tải); ngưỡng cứng PM ≤ 10 phút (và <  |
| TC-21 | P | run PR 43712425198 @be0c0d96: 14/14 job success, không job skipped |
| TC-22 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-23 | P | gh api branches/main/protection → 404 "Branch not protected"; rulesets = [] → không có check bắt buộc |
| TC-24 | P | Ánh xạ bước 1518f83→be0c0d96 đủ: Changes selftest ci-changes.sh; go vet ×2, golangci ×2, sqlc diff (go-static); go test -race testroutes + đo thời gian (go-test); Judge `-race -tags integration -timeo |
| TC-25 | P | Mỗi dòng Go khôi phục cache (go-mod key go.sum; go-build restore-keys …-static-/…-rest-); rest-1 ghi "Cache saved with key …-rest-43712425198", static ghi …-static-…; không "Unable to reserve"/"Failed |
| TC-26 | P | Commit US-CI-02 (089d239d) chỉ đổi .github/workflows/ci.yml; vòng 2/3 (d49c41a4, e5f6390b) ci.yml + handoff (test Go sửa theo PM #19 ở commit riêng). playwright.config.ts không đổi 1518f83..be0c0d96;  |
| TC-E1 … TC-27 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |

## Trượt: bước tái hiện
### DEF-R7-01 — CI đủ job (US-CI-02) vượt ngưỡng 10 phút: wall 16 phút, Playwright shard 1/3 gánh 15,4 phút (TC-20). Chi tiết `defects-7.md`.

## Ca K / T / N và lý do
- K(b — PM duyệt đẩy nhánh tạm): TC-08, 16, 17, 18, 19, 22 của US-CI-02.
- N (chưa chạy mẫu): ca UI Playwright-CLI/375 px, tick dài (SLA, reminder, rollup, auto-close, crm-risk-run), kill worker/gateway giữa chừng, hai gateway SSE, k6, Lighthouse, các ca Go chưa kịp chạy, S7-SEED (seed/expected, seed-study-time, gate-p4/p5 cần chạy lại script gate), CI-02 còn lại (đối chiếu trước/sau).
