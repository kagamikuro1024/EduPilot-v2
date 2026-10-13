# QC report — US-P4-05 (vòng đời phía sinh viên, SLA, tri thức, ngưỡng) — vòng 1
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `79a230d`. Bộ TC: `tc-US-P4-05.md` (không sửa chữ TC; hai chỗ TC viết chưa đúng hợp đồng ghi ở mục 4). Handoff: `dev-US-P4-05.md`, `dev-QC-answers.md` (`gateway admin run-job ticket.sla|ticket.autoclose [--now]`). Stack `edupilot100` dựng lại từ code mới (`docker-compose.test.yml`, `--scale gateway=2`, provider `fake`; seed dừng ở bước 11 vì thiếu docling, nên không có tài liệu). Đồng hồ giả: `--now` (chỉ ảnh hưởng một lần chạy) và UPDATE `waiting_since` / `answered_at` trên DB test. `audit_log` bất biến (trigger chặn UPDATE).

## Kết luận: **FAIL** (6 TC FAIL từ 4 lỗi; 3 TC BLOCKED)
| Lỗi | Mức | TC |
| --- | --- | --- |
| BUG-1 `knowledge.sync` thất bại thật ở worker: `MASK_FAILED: lời gọi không gắn lớp (llm.WithIdentity)` → tri thức không bao giờ được nạp | **CAO** | 40, 41, 42 |
| BUG-2 ticket `CLAIMED` quá hạn **không** nhắc người đã nhận (không `TICKET_STALE` cá nhân) | TRUNG BÌNH | 21 |
| BUG-3 `Đóng không trả lời`: chữ tin khác AC9 và không có chuông cho sinh viên | TRUNG BÌNH | 32 |
| BUG-4 dòng "Giảng viên đã nhận · đã chờ N phút" không tự cập nhật theo phút | THẤP | 28 |

## 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 hỏi tiếp mở lại | PASS | `200`; `OPEN`, `claimed_by` rỗng, `reopened_count=1`, `sla_stage=0`, `waiting_since≈now`; tin `USER` / `ESCALATION_FOLLOWUP`, `blocks` có `kind:"ticket"`. `client_msg_id` phải là UUID (TC ghi `"c1"`: `422 required`) |
| 02 giảng viên thấy đúng tin | PASS | `transcript` có tin hỏi tiếp, không tin khác của phiên |
| 03 idempotent | PASS | cùng `client_msg_id` → `200`, 1 tin |
| 04 độ dài | PASS | rỗng `422`; 2.001 `422`; 2.000 `200` |
| 05 trạng thái | PASS | `OPEN`, `CLAIMED` → `409 TICKET_STATE_INVALID` |
| 06 30 ngày | PASS | 31 ngày `409 TICKET_REOPEN_EXPIRED`; 29 ngày `200` |
| 07 không quá hạn tức thì | PASS | ticket tạo 10 ngày trước, hỏi tiếp, `run-job ticket.sla` → `sla_stage=0` |
| 08 chủ ticket | PASS | `$SVB` 404; `$TCH` 403; `$ADM` 403 |
| 09 `Đã rõ` | PASS | `CLOSED`, `STUDENT_CONFIRMED`, `closed_at` có |
| 10 hoàn tác | PASS | < 5 s → `ANSWERED`, `closed_reason`/`closed_at` rỗng; sau 6 s → `409 TICKET_STATE_INVALID`; sau hỏi tiếp → `409` |
| 11 UI `Đã rõ` | PASS | `scripts/p4-05-ui.mjs`: "Đã đóng · Hoàn tác", 0 hộp thoại, không toast riêng; sau 5,5 s mất "Hoàn tác" |
| 12 vai | PASS | `$SVB` 404; `$TCH` 403 |
| 13 tự đóng 7 ngày | PASS | `answered_at` 8 ngày → `AUTO_7D`; chạy lại giữ `closed_at`; `OPEN` 9 ngày không đóng |
| 14 mốc đặt lại | PASS | `answered_at` 6 ngày + trả lời thêm → không đóng |
| 15 sau khi đóng | PASS | `CLOSED` `AUTO_7D` → `Hỏi tiếp` `200` |
| 16 hàm giờ làm việc (bằng hành vi) | PASS | T6 12:00 chờ qua T7/CN: `sla_stage=0` ở T7 12:00, CN 23:00, T2 11:59; `=1` ở T2 12:01. T2 08:00 → `0` ở T3 07:59, `1` ở T3 08:01. `go test -race … -run 'TestWorking\|TestReminderAt\|TestEscalateAt…'` `ok` (5 hàm kiểm) |
| 17 chín dòng bảng | **BLOCKED** | không API trả `WorkingElapsed`; test Go không dùng `t.Run` nên không đếm được dòng bảng (≥ 14 + 5 theo TC-16); chỉ kiểm được qua TC-16 / 19 |
| 18 nhắc 24 giờ | PASS | `sla_stage=1`; một `TICKET_STALE` gộp "N câu hỏi chờ quá 24 giờ làm việc" cho TEACHER và TA; một thư `ticket_reminder` mỗi người ("Lớp 761987 có 8 câu hỏi đang chờ." + `/inbox?course=…`, 0 tên / nội dung); chạy lại không thêm; thư tối đa 1 mỗi giờ mỗi lớp (`dedupe_key …:2026101401`) |
| 19 cuối tuần | PASS | xem TC-16 |
| 20 song song | PASS | 3 lần `run-job` đồng thời: `sla_stage=1`, `outbox escalation.sla_reminder` +1 |
| 21 `CLAIMED` quá hạn | **FAIL** | ticket `CLAIMED` (bởi teacher, rồi bởi TA), `claimed_at` và `waiting_since` lùi 26–30 giờ, `run-job ticket.sla` (cả `--now`+30h): **0** dòng `TICKET_STALE` cá nhân (`dedupe_key ticket_stale:%`), `sla_stage` giữ 0; US AC5 đòi nhắc người đã nhận |
| 22 lọc theo `waiting_since` | PASS | TC-07 (cũ `created_at`, `waiting_since` mới → không nhắc); mọi ca nhắc đặt cả hai mốc cũ |
| 23 72 giờ | PASS | 125 giờ lịch (= 73+ giờ làm việc) → `sla_stage=2`; `GET /me/today` TEACHER: `TICKET` đứng **đầu** `actions`, "1 câu hỏi chờ quá 72 giờ làm việc · lâu nhất 77 giờ", `/inbox?course=…` |
| 24 báo Admin | PASS | không ai hoạt động (`--now` +100 h, `audit_log` bất biến không lùi được): ADMIN nhận một `COURSE_TICKETS_STALLED` "Lớp 761987 có 1 câu hỏi chờ quá 72 giờ làm việc"; Hôm nay ADMIN "Chưa có giảng viên hay trợ giảng nào xử lý."; 0 canary / nội dung |
| 25 có người hoạt động | PASS | teacher vừa đăng nhập → job: 0 thông báo; teacher / TA không nhận `STALLED` |
| 26 mỗi ngày một lần | PASS (một phần) | 3 lần cùng ngày → 1 thông báo (`stalled:<course>:20261014`). Qua ngày: `--now` không đổi ngày của khoá (khoá dùng giờ thật) nên **không** kiểm được 3 ngày liên tiếp — BLOCKED phần này |
| 27 chữ trạng thái | PASS | `OPEN` "Đang chờ giảng viên · vừa gửi"; `CLAIMED` "Giảng viên đã nhận · đã chờ 0 phút"; `ANSWERED` "Giảng viên · TS. Lê Thu Hà" + `Hỏi tiếp`, `Đã rõ`; `CLOSED` "Đã đóng" + `Hỏi tiếp`; không "ticket" / "SLA" / số tin cậy |
| 28 cập nhật mỗi phút, không nhảy | **FAIL** | "Giảng viên đã nhận · đã chờ 13 phút" giữ nguyên chữ sau 130 s (Δ vị trí 0 px nhưng không đổi); mong "15 phút" |
| 29 Hôm nay sinh viên | PASS | `STAFF_ANSWER_UNREAD` "Giảng viên đã trả lời câu hỏi của bạn", "Lớp 761987 · 0 phút trước", `/chat?session=…` (bậc `urgency high`) |
| 30 đọc khi mở | PASS | `GET …/messages` không đổi `updated_at`; `POST /notifications/read {link}` → `{"updated":1}`, `read_at` đặt |
| 31 chỉ của mình | PASS | `$SVA` đọc link của `sv.kha` → `{"updated":0}`, dòng của kha còn chưa đọc |
| 32 `Đóng không trả lời` | **FAIL** | `OUT_OF_SCOPE` / `DUPLICATE` / `NEED_MORE_INFO` → `CLOSED` `STAFF_DISMISSED`, nhưng chữ tin: "Câu hỏi này nằm ngoài phạm vi học phần nên giảng viên không trả lời." / "Câu hỏi này đã được trả lời ở nơi khác của lớp." / "Giảng viên cần thêm thông tin để trả lời. Bạn hãy đặt lại câu hỏi rõ hơn." — AC9: "Giảng viên đã đóng câu hỏi này: ngoài phạm vi môn học." / "…trùng với câu hỏi khác." / "…cần bạn nói rõ hơn.". Và **không có chuông** (0 `notifications` mới của sinh viên sau `dismiss`; AC9 "và chuông") |
| 33 hỏi tiếp sau dismiss | PASS | cả ba `200` |
| 34 người khác đã nhận | PASS | teacher `dismiss` ticket TA nhận → `409` kèm người nhận; `reason` ngoài enum → `422` |
| 35 xem trước che PII | PASS | tên, MSSV, email, SĐT → "[đã ẩn]"; grep = 0 |
| 36 còn PII | PASS | tên, MSSV → `422 PII_DETECTED` |
| 37 câu hỏi cá nhân | PASS | preview `personal:true`; `POST knowledge` → `422 KNOWLEDGE_PERSONAL` |
| 38 lưu | PASS | `201 {"saved":true,"loaded":false}`; `content_chunks.text` = bản đã sửa, `embedding IS NULL`; 1 dòng `knowledge_items`; outbox `knowledge.sync`; `save_as_knowledge=t`; audit `escalation_ticket.knowledge_saved`; canary tên / MSSV trong chunk = 0 |
| 39 lưu hai lần, vai, trạng thái | PASS | lần hai `200`, `knowledge_items` 1, chunk 1; `$SVA` 403, `$ADM` 403, TA lớp khác 403; `OPEN` / `CLAIMED` → `409 TICKET_STATE_INVALID` |
| 40 embed bản đã sửa | **FAIL** | worker không gọi được `Embed` (BUG-1) nên không có chuỗi nhúng nào để kiểm |
| 41 dùng được ngay ≤ 10 s | **FAIL** | sau 12 s và sau ≈ 1 phút `embedding IS NULL`; `outbox` `knowledge.sync` `attempts=3`, `last_error="knowledge: nhúng: agent: nhúng: MASK_FAILED: lời gọi không gắn lớp (llm.WithIdentity)"`; log worker `xử lý outbox thất bại … MASK_FAILED` |
| 42 chưa nạp | **FAIL** | giao diện đúng ("Chưa nạp vào tri thức" + `Thử lại` + `Gỡ`), nhưng provider **sống** mà vẫn không nạp được; `Thử lại` ghi lại outbox và lỗi y nguyên |
| 43 gỡ | PASS (một phần) | `DELETE` `204`, `save_as_knowledge=f`, chunk bị xoá, lặp lại `204`; lưu lại thay đoạn cũ (`knowledge_items` vẫn 1). "≤ 5 s đoạn không còn ở kết quả tìm" không kiểm được (không có đoạn nào được nạp, BUG-1) |
| 44 không lọt vào tài liệu | PASS | tài liệu `THREAD_KNOWLEDGE` (đoạn tri thức) không nằm trong `GET /documents`, `GET /documents/stats`, `GET /documents/{id}` → 404; `uploads/complete` với `category=THREAD_KNOWLEDGE` → `422 reserved` |
| 45 cô lập lớp | **BLOCKED** | không có đoạn nào nạp được (BUG-1) |
| 46 ngưỡng | PASS | `PUT /courses/{id}/settings {"escalation_threshold":"0.85","version":1}` → `200` (giá trị là **chuỗi**; TC viết số). Hiệu lực ở tin kế tiếp: không dựng được ca (chỉ `COURSE_QA` / `GRADE_FORMULA` ép được `low_confidence`, seed không có tài liệu); dev có `TestThresholdTakesEffectNextMessage` |
| 47 ngoài khoảng / bước | PASS | `"0.49"`, `"0.96"`, `"0.82"`, `"abc"` → `422`; số `0.85` (không phải chuỗi) → `422 type` |
| 48 version, vai | PASS | `version` cũ → `409 VERSION_CONFLICT` kèm `current_version` và `settings`; TA / SV / ADM `403` |
| 49 audit, UI | PASS | audit `course_threshold_updated` `{"escalation_threshold":"0.80"} → "0.85"`; thanh trượt "Ít chuyển / Chuyển nhiều" qua e2e `threshold slider` xanh |
| 50 worker chết giữa `ticket.sla` | **BLOCKED** | không dừng được đúng giữa job trên stack thật; `TestSLAJobCrashSafe` (dev) xanh; TC-20 (3 bản song song) cho thấy ghi có điều kiện |
| E1 Redis chết lúc `confirm` | PASS (theo #22) | `confirm` vẫn `200`; PM đã quyết #22: Redis chết lúc `confirm` → 200 |
| E2 autoclose không đóng nhầm `OPEN` | PASS | TC-13: `OPEN` 9 ngày giữ `OPEN` |
| P1 | PASS | TA lớp 1 gọi `knowledge` ticket lớp 2 → 403; `COURSE_TICKETS_STALLED` chỉ ADMIN |

Tổng 50 TC: PASS 40 (gồm 4 PASS một phần: 26, 43, 46, E-ngoài), FAIL 6 (21, 28, 32, 40, 41, 42), BLOCKED 3 (17, 45, 50); E1 PASS theo #22.

## 2. Lỗi
**BUG-1 (CAO) — kho tri thức không nạp được ở stack thật.** Tái hiện: ticket `ANSWERED`, giảng viên `POST …/tickets/{id}/knowledge {question, answer}` → `201 {"loaded":false}`; đợi 60 s: `select attempts,last_error from outbox where topic='knowledge.sync'` → `3 | knowledge: nhúng: agent: nhúng: MASK_FAILED: lời gọi không gắn lớp (llm.WithIdentity)`. `EXPLAIN`: không liên quan. Nguyên nhân quan sát: lời gọi `Embed` của handler `knowledge.sync` không gắn danh tính (`llm.WithIdentity`) nên tường lửa PII từ chối. Các test Go của dev (`TestKnowledge*`) xanh vì không đi qua đường này ở worker thật. Hệ quả: AC10/AC11 không đạt đầu-cuối ("Dùng được ngay").
**BUG-2 (TRUNG BÌNH):** AC5 — `CLAIMED` quá hạn không có `TICKET_STALE` cá nhân. Tái hiện: nhận ticket, `update escalation_tickets set claimed_at=now()-'30 hours', waiting_since=now()-'30 hours'`, `gateway admin run-job ticket.sla`.
**BUG-3 (TRUNG BÌNH):** AC9 — ba chữ tin và chuông (xem TC-32).
**BUG-4 (THẤP):** TC-28 — chữ "đã chờ N phút" không tăng khi để trang mở.

## 3. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -race -tags testroutes ./internal/escalation -run 'TestFollowup\|…\|TestKnowledge'` | `ok` (14,8 s) |
| `go test -race ./internal/rag ./internal/library ./internal/document ./internal/course ./internal/chat -run 'TestKnowledge\|…\|TestSettingsThreshold'` | 14 passed / 5 gói |
| `go test -race -tags testroutes ./internal/contract/... ./internal/today ./cmd/worker ./cmd/gateway` | 136 passed / 4 gói |
| e2e `student sees ticket status \| student followup \| student confirm \| knowledge save \| threshold slider` | xanh trong lượt desktop 141 passed / 0 failed |
| `run-job ticket.sla --now …` | chạy; kết quả ở TC-16…26 |

## 4. Ghi chú về TC (không sửa; xin PM nếu muốn đổi)
- `client_msg_id` là UUID (TC-01…05 dùng `"c1"`), `escalation_threshold` là chuỗi (TC-46…48 dùng số): lệch chữ TC, không phải lỗi sản phẩm.
- TC-E1 (Redis chết lúc `confirm`): PM quyết #22 → `200`; TC viết `503` là chữ cũ.
- `--now` của `run-job` không đổi ngày dùng trong `dedupe_key` của `COURSE_TICKETS_STALLED`; TC-26 qua ngày cần cách khác.
