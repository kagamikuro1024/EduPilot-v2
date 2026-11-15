# Smoke hồi quy sprint 7 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm E) · Ngày: 2026-11-15
Tổng TC cũ: 582 · Đạt: 159 · Trượt: 1 · Không chạy K (a/b/c): 6 · Lỗi thời (T, tính vào cột K của brief): 0 · Chưa chạy (mẫu) N: 416   (P+F+K+T+N = 582 = 582)

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
| US-P4-01 | 42 | 9 | 0 | 0 | 0 | 33 |  |
| US-P4-02 | 31 | 7 | 0 | 0 | 0 | 24 |  |
| US-P4-03 | 45 | 19 | 0 | 0 | 0 | 26 |  |
| US-P4-04 | 39 | 9 | 0 | 0 | 0 | 30 |  |
| US-P4-05 | 53 | 19 | 0 | 0 | 0 | 34 |  |
| US-P4-06 | 55 | 22 | 0 | 0 | 0 | 33 |  |
| US-P4-07 | 29 | 11 | 0 | 0 | 0 | 18 |  |
| US-P5-01 | 36 | 12 | 0 | 0 | 0 | 24 |  |
| US-P5-02 | 48 | 13 | 0 | 0 | 0 | 35 |  |
| US-P5-03 | 47 | 10 | 0 | 0 | 0 | 37 |  |
| US-P5-04 | 37 | 10 | 0 | 0 | 0 | 27 |  |
| US-P5-05 | 35 | 9 | 0 | 0 | 0 | 26 |  |
| US-P5-06 | 28 | 4 | 0 | 0 | 0 | 24 |  |
| US-S7-SEED | 25 | 0 | 0 | 0 | 0 | 25 |  |
| US-CI-02 | 32 | 5 | 1 | 6 | 0 | 20 |  |

## Kết quả từng ca
(Mỗi ca P/F/K/T một dòng. Ca N liền nhau cùng lý do được gom một dòng.)
| TC | KQ | Ghi chú |
|---|---|---|
| TC-P4-01-01 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P4-01-02 … TC-P4-01-06 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
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
| TC-P4-01-38 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-01-E1 … TC-P4-01-P1 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-02-01 … TC-P4-02-03 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
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
| TC-P4-04-33 … TC-P4-04-36 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-04-E1 … TC-P4-04-P1 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-01 | P | followup → 200 {'ticket': {'id': '01a1df53-c92c-7dbc-8808-d96a6181fbf9', 'status': 'OPEN', 'created_at': '2026-11-1 |
| TC-P4-05-02 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-03 | P | lặp cùng client_msg_id → 200; số tin=1 |
| TC-P4-05-04 | P | content rỗng/2001 → [422, 422] |
| TC-P4-05-05 … TC-P4-05-07 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-08 | P | followup bởi SV khác/GV/ADMIN → [404, 403, 403] |
| TC-P4-05-09 | P | confirm → 200; trạng thái CLOSED/STUDENT_CONFIRMED |
| TC-P4-05-10 | P | undo <5s → 200; trạng thái ANSWERED |
| TC-P4-05-11 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-12 | P | confirm bởi SV khác/GV → [404, 403] |
| TC-P4-05-13 … TC-P4-05-17 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-18 | P | tick SLA 1 phút (TICKET_SLA_TICK=1m), Chủ nhật nên tuổi ticket tính theo giờ làm việc = 0: dùng seed-age (UPDATE waiting_since vào các ngày làm việc trước): 8 giờ làm việc → stage 0; ~2 ngày → stage 1 |
| TC-P4-05-19 | P | ticket ≥ 72 giờ làm việc (waiting_since 2026-11-05) → sla_stage=2 sau một tick |
| TC-P4-05-20 … TC-P4-05-28 (9 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-29 | P | me/today SV → 200 keys ['no_course', 'email_verified', 'recommended', 'timeline', 'continue'] |
| TC-P4-05-30 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-31 | P | notifications/read {link} bởi SV → 200 |
| TC-P4-05-32 | P | [('OUT_OF_SCOPE', 200, 'CLOSED/STAFF_DISMISSED'), ('DUPLICATE', 200, 'CLOSED/STAFF_DISMISSED'), ('NEED_MORE_INFO', 200, 'CLOSED/STAFF_DISMISSED')] |
| TC-P4-05-33 … TC-P4-05-34 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-35 | P | preview → 200 |
| TC-P4-05-36 | P | lưu lần đầu → 201; lần hai → 200 {'saved': True, 'loaded': False} |
| TC-P4-05-37 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-38 | P | có knowledge_items sau khi lưu |
| TC-P4-05-39 | P | SV/ADMIN/GV lớp khác lưu tri thức → [403, 403, 403] |
| TC-P4-05-40 … TC-P4-05-42 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-43 | P | gỡ tri thức → 204, lặp → 204 |
| TC-P4-05-44 … TC-P4-05-46 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-47 | P | 0.85 → 200; 0.49/0.96/0.82/abc → [422, 422, 422, 422] |
| TC-P4-05-48 | P | version cũ → 409; TA/SV/ADMIN → [403, 403, 403] |
| TC-P4-05-49 … TC-P4-05-50 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-05-E1 … TC-P4-05-P1 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-01 | P | SV báo SPAM → 201 {'reported': True} |
| TC-P4-06-02 | P | báo lần hai → 409 ALREADY_REPORTED; số dòng=1 |
| TC-P4-06-03 | P | enum sai/note>200/note với SPAM → [422, 422, 422] |
| TC-P4-06-04 | P | báo bài của mình → 422 CANNOT_REPORT_OWN |
| TC-P4-06-05 | P | bài không tồn tại → 404 |
| TC-P4-06-06 | P | ADMIN báo → 403 |
| TC-P4-06-07 | P | SV (người báo/chủ bài/khác) không thấy khoá báo cáo |
| TC-P4-06-08 … TC-P4-06-09 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-10 | P | 3 báo cáo → true/NULL/Nhiều báo cáo, chờ duyệt (201,201) |
| TC-P4-06-11 | P | bài ẩn: có trong danh sách=False, chi tiết 404, tìm kiếm 0 |
| TC-P4-06-12 … TC-P4-06-15 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-16 | P | queue HIDDEN 200, OPEN 200; keys mục=['kind', 'thread_id', 'post_id', 'title_or_excerpt', 'reports', 'hidden_at', 'hidden_by_auto', 'created_at'] |
| TC-P4-06-17 | P | hide → 200; reason rỗng/201 → [422, 422]; báo cáo → ACTIONED |
| TC-P4-06-18 | P | unhide → 200; SV xem lại → 200 |
| TC-P4-06-19 | P | {'sv08': [403, 403, 403, 403, 403], 'admin': [403, 403, 403, 403, 403]} |
| TC-P4-06-20 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-21 | P | hide song song → {'teacher': 200, 'ta': 409} |
| TC-P4-06-22 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-23 | P | sửa kèm SĐT → 422 PII_DETECTED; bản lưu chứa SĐT=False |
| TC-P4-06-24 | P | sửa sạch → 200; edited_at có=true |
| TC-P4-06-25 … TC-P4-06-26 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-27 | P | GV/TA/SV khác sửa → [403, 403, 403]; sai version → 409 |
| TC-P4-06-28 … TC-P4-06-30 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-31 | P | chủ xoá → 204; deleted_at=true; GV mở lại → 404 |
| TC-P4-06-32 | P | SV khác/GV xoá → [403, 403] |
| TC-P4-06-33 … TC-P4-06-34 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-35 | P | ghim 3 bài đầu thành công, bài thứ 4 → 409 (PIN_LIMIT); mã thành công là 204 theo openapi (TC ghi «200») |
| TC-P4-06-36 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-37 | P | SV/ADMIN ghim → [403, 403] |
| TC-P4-06-38 … TC-P4-06-45 (8 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-46 | P | {'sv08': [201, 403], 'ta': [201, 200], 'teacher': [201, 200], 'admin': [403, 403]} |
| TC-P4-06-47 … TC-P4-06-53 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-06-E1 … TC-P4-06-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-01 | P | 20/20 câu dương → CRISIS; sót: [] |
| TC-P4-07-02 | P | 17 câu âm: bị bắt nhầm CRISIS: [] |
| TC-P4-07-03 … TC-P4-07-05 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-06 | P | câu trả lời mẫu duy nhất: 'Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên củ' |
| TC-P4-07-07 | P | cụm cấm trong câu mẫu: [] |
| TC-P4-07-08 | P | 0 lời gọi LLM (llm_audit CHAT/UTILITY/CLASSIFY không tăng) qua 20+17+… câu |
| TC-P4-07-09 | P | (lớp E6C1) tickets/notifications/pii_events: ['28', '30', '27'] → ['28', '30', '27'] |
| TC-P4-07-10 … TC-P4-07-11 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-12 | P | đăng thread khủng hoảng → 422 CRISIS_SUPPORT |
| TC-P4-07-13 | P | bình luận khủng hoảng → 422 CRISIS_SUPPORT |
| TC-P4-07-14 | P | precheck 6 câu khủng hoảng → 200 allowed, không trả CRISIS_SUPPORT (chỉ chặn ở Create/Comment: 422 CRISIS_SUPPORT) |
| TC-P4-07-15 … TC-P4-07-24 (10 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-25 | P | «deadline này làm em muốn chết» → CRISIS (câu hỗ trợ) |
| TC-P4-07-26 | P | đọc phiên khủng hoảng bởi TA/GV/ADMIN/SV khác → {'ta': 403, 'teacher': 403, 'admin': 403, 'sv09': 404} |
| TC-P4-07-27 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P4-07-E1 … TC-P4-07-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-01 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P5-01-02 … TC-P5-01-04 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-05 | P | dry_run → 200, buổi xem trước=0; số buổi DB 9→9 |
| TC-P5-01-06 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-07 | P | lần chạy đầu trên lớp trống: generate thật → 201, số buổi 0→6 (lần chạy lặp báo «DUPLICATE» nên số buổi giữ nguyên — đúng) |
| TC-P5-01-08 | P | lần chạy đầu: khác khoá cùng tham số → 201 nhưng số buổi 6→6 (bỏ DUPLICATE, không nhân đôi) |
| TC-P5-01-09 | P | tham số sai → [422, 422, 422, 422, 422, 422] |
| TC-P5-01-10 | P | >60 buổi → 422 VALIDATION_FAILED |
| TC-P5-01-11 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-12 | P | lặp cùng key → 201, số buổi giữ 9 |
| TC-P5-01-13 … TC-P5-01-14 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-15 | P | PUT buổi → 200 {'session': {'id': '01a1df59-df98-7982-94b4-aac2b49cce36', 'session_no': 11, 'st |
| TC-P5-01-16 | P | sai version / kết thúc<bắt đầu → [409, 422] |
| TC-P5-01-17 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-01-18 | P | DELETE buổi chưa có điểm danh → 204 |
| TC-P5-01-19 … TC-P5-01-22 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
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
| TC-P5-04-03 … TC-P5-04-04 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-05 | P | filter attention/absent/inactive → [200, 200, 200] |
| TC-P5-04-06 | P | q không dấu/hoa thường/MSSV/không khớp → [200, 200, 200, 200] |
| TC-P5-04-07 | P | filter=xyz → 422 |
| TC-P5-04-08 … TC-P5-04-14 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-15 | P | job crm-risk-run đã chạy: khóa Redis ep:crm.risk:done:20261115 có; student_risk lớp E6C1 có 7 hàng (cả bảy NONE) |
| TC-P5-04-16 … TC-P5-04-18 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-19 | P | đặt ngưỡng risk → 200 |
| TC-P5-04-20 | P | ngưỡng ngoài miền → [422, 422, 422, 422, 422] |
| TC-P5-04-21 | P | TA/SV/ADMIN → [403, 403, 403]; sai version → 409 |
| TC-P5-04-22 … TC-P5-04-28 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-29 | P | {'sv09': [403, 403], 'admin': [403, 403], 'sv.moi': [403, 403], 'ta': [200, 200]} |
| TC-P5-04-30 … TC-P5-04-35 (6 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-04-E1 … TC-P5-04-P1 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-01 | P | hồ sơ 360 → 200; keys=['student', 'risk', 'attendance', 'participation', 'study', 'activity'] |
| TC-P5-05-02 … TC-P5-05-05 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-06 | P | hồ sơ không chứa nội dung chat riêng |
| TC-P5-05-07 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-08 | P | {'sv09': [403, 403], 'admin': [403, 403], 'ta': [200, 200]} |
| TC-P5-05-09 | P | TA sửa/xoá ghi chú của GV → [403, 403] |
| TC-P5-05-10 | P | tạo ghi chú → 201 |
| TC-P5-05-11 | P | tags>5/tag rỗng/tag 31/body rỗng/body 2001 → [422, 422, 422, 422, 422] |
| TC-P5-05-12 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-13 | P | SV không thấy CANARY-OBS trong me/today, me/attendance, me/participation |
| TC-P5-05-14 … TC-P5-05-22 (9 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-23 | P | outreach → 201 |
| TC-P5-05-24 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-25 | P | outreach chứa tên/email/MSSV của SV khác → [422, 422, 422] |
| TC-P5-05-26 … TC-P5-05-33 (8 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-05-E1 … TC-P5-05-P1 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-01 | P | me/attendance, me/participation → [200, 200] |
| TC-P5-06-02 | P | tham số student/student_id bị bỏ qua (cùng kết quả của chính mình) |
| TC-P5-06-03 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-04 | P | TA/GV/ADMIN me/* → [403, 403, 403, 403, 403, 403] |
| TC-P5-06-05 | P | limit=2 → 200 0 mục; keys=['attendance', 'next_cursor'] |
| TC-P5-06-06 … TC-P5-06-26 (21 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P5-06-E1 … TC-P5-06-E2 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-S7-SEED-01 … TC-id (26 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-01 … TC-04 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-05 | P | mọi dòng Go matrix: `go test -race -tags testroutes -v $pkgs`; không -short/-failfast |
| TC-06 | P | union 4 dòng matrix = `go list -tags testroutes ./...` = 100 gói, mỗi gói đúng 1 dòng (không trùng, không thiếu); sprint 7 gốc là 42 gói — tăng do sprint sau thêm gói |
| TC-07 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-08 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-09 | P | course/auth là dòng riêng; rest-1/rest-2 loại trừ bằng grep -Ev (xem log) |
| TC-10 | P | bước `go test -p 1 -count=3 -tags testroutes -run TimingEqualized$` vẫn chạy cho ./internal/auth/ và ./internal/course/ |
| TC-11 … TC-15 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-16 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-17 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-18 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-19 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-20 | F | wall run PR 43712425198 = 15,97 phút (12:09:46→12:25:44), run push e0bf1fc9 = 16,7 phút; job dài nhất Playwright 1/3 = 15,4 phút (shard 2/3 = 6,5, 3/3 = 2,7: lệch tải); ngưỡng cứng PM ≤ 10 phút (và <  |
| TC-21 | P | run PR 43712425198 @be0c0d96: 14/14 job success, không job skipped |
| TC-22 | K | loại b / cần PM duyệt đẩy nhánh tạm lên GitHub (brief cấm push) |
| TC-23 … TC-26 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-E1 … TC-27 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |

## Trượt: bước tái hiện
### DEF-R7-01 — CI đủ job (US-CI-02) vượt ngưỡng 10 phút: wall 16 phút, Playwright shard 1/3 gánh 15,4 phút (TC-20). Chi tiết `defects-7.md`.

## Ca K / T / N và lý do
- K(b — PM duyệt đẩy nhánh tạm): TC-08, 16, 17, 18, 19, 22 của US-CI-02.
- N (chưa chạy mẫu): ca UI Playwright-CLI/375 px, tick dài (SLA, reminder, rollup, auto-close, crm-risk-run), kill worker/gateway giữa chừng, hai gateway SSE, k6, Lighthouse, các ca Go chưa kịp chạy, S7-SEED (seed/expected, seed-study-time, gate-p4/p5 cần chạy lại script gate), CI-02 còn lại (đối chiếu trước/sau).

## Độ phủ P/F theo US (lượt cuối)
P4-01 21% · P4-02 23% · P4-03 42% · P4-04 23% · P4-05 36% · P4-06 40% · P4-07 38% · P5-01 33% · P5-02 27% · P5-03 21% · P5-04 27% · P5-05 26% · P5-06 14% · S7-SEED 0% · CI-02 19%. Chưa đạt 40%: các US còn lại; S7-SEED chưa chạy (cần seed.mjs/seed-study-time + gate-p4/p5).
