# SRS FEAT-escalation-notify Escalation, thông báo tức thời, thư qua mail, kiểm duyệt Threads, an toàn con người
Phiên bản 1.0 · 2026-10-13 · Trạng thái: **DRAFT** (chờ Tech Lead thẩm định `TL-REVIEW.md`, PM duyệt).

Nguồn: `docs/phases/P4.md` (L1, L2, L2b, L3); `docs/sprints/7/plan.md` story 1–7; PRD M2, M3, §3, §5; FLOWS F3, F4, F5, F14; `ARCHITECTURE.md` §4 (`00012`), §5 (Escalation, Thông báo, Kiểm duyệt Threads), §7 (`/inbox`); `SYSTEM_DESIGN.md` 3.2–3.5, §5 (SLO); `DECISIONS.md` D6, D39, D44, D47, D49, D51, D56, D59; `design/DESIGN.md` §13, §14.3–14.5; `UX.md` mục 2, 5, 6; `CLAUDE.md` (luật chữ UI); spec đã có `FEAT-course-foundation` v1.8 (`notifications`, mail outbox, `Hôm nay` 4.7, Guard), `FEAT-private-chat-pii` v1.7 (chat, Threads, ngưỡng tự tin, khoá giờ thi, an toàn con người), `FEAT-docs-calendar` (RAG, documents, ingest); mã `backend-go/internal/{agent,chat,thread,today,mail,httpapi/sse,platform/outbox}` ở `main` + `sprint/7-p4-p5` (đọc để biết cái đã làm, không sửa).

## 1. Mục đích và phạm vi

AI chưa chắc thì giảng viên được báo ngay; giảng viên trả lời trong app; sinh viên nhận câu trả lời trong chat riêng, qua chuông và qua mail; ticket không nằm im; Threads có kiểm duyệt; tri thức đã xác nhận quay lại làm nguồn cho AI; tin nhắn khủng hoảng được đón bằng lời ân cần. Bảy story (`US.md`): ticket + nút (US-P4-01), thông báo tức thì (-02), `/inbox` (-03), thư (-04), vòng đời + SLA + tri thức + ngưỡng (-05), kiểm duyệt Threads (-06), an toàn con người (-07).

**Hiện trạng đã đối chiếu (đọc mã) — spec chỉ viết phần còn thiếu:**

| Hạng mục | Đã có ở `main` | Thiếu → làm ở sprint 7 |
| --- | --- | --- |
| Hạ tầng SSE | `GET /api/v1/events` (`internal/httpapi/sse`): một luồng ghép kênh theo người dùng, bộ đệm Redis Stream `ep:sse:buf:{uid}`, Pub/Sub `ep:sse:ch:{uid}`, `Last-Event-ID` = id Redis Stream, `resync`, ≤ `SSE_MAX_PER_USER` = 2 kết nối, đóng sau `SSE_MAX_DURATION` = 120 s (`reconnect`) | Phát sự kiện `notification`; chuông dùng luồng này; **không** dựng `/notifications/stream` thứ hai (4.4) |
| Thông báo | bảng `notifications` (`dedupe_key` duy nhất theo người), `GET /notifications`, chuông làm mới 30 s; bốn nơi ghi trực tiếp (`course`, `exam`, `thread`, nhắc lịch) | `internal/notify` một cửa + đẩy SSE + gộp; chuyển bốn nơi ghi sang cửa này |
| Mail | `internal/mail` (`Enqueue` trong giao dịch, consumer `mail.send`, thử lại 1 s / 5 s / 30 s rồi `DEAD`), bảng mẫu `specs()`; `user_settings.notify_*_by_mail` | 4 mẫu mới; **resolver** đọc ticket lúc gửi (payload chỉ có id); mặc định `notify_ticket_by_mail` sai ở `00003` |
| Ngưỡng tự tin | `courses.escalation_threshold`; `agent.ResponseMetadata.LowConfidence`; `chat.Service.finish` ghi câu trả lời trong **một** giao dịch | tự mở ticket trong chính giao dịch đó; `ShouldEscalate`; thanh trượt + `PUT …/settings` |
| Nút "Nhờ giảng viên" | **ẩn** (US-P3-07 AC4, US-P3-05 AC16; chủ dự án 2026-10-10) | hiện lại; đảo ngược hai AC đó (10) |
| Threads | `Decide` (verify / correct / reject), thông báo trả lời / xác nhận, lọc tuần / thẻ / trạng thái, `pinned_at` (cột), `hidden_*` ở `forum_posts` | báo cáo, tự ẩn, ẩn / bỏ ẩn, sửa / xoá của chủ bài, ghim, nạp tri thức; `forum_threads` **chưa có** cột ẩn |
| Hôm nay | `Item`, 14 nguồn P2/P3; `AI_CONFIRM` (bậc 50); `attention: []` | `TICKET`, `POST_REPORTED`, mục sinh viên "câu trả lời chưa đọc", mục Admin "lớp có ticket kẹt" |
| An toàn con người | `IntentCrisis` thắng mọi intent, câu mẫu + `SUPPORT_RESOURCES_VI`, 0 LLM, không log (US-P3-04 AC13) | bộ 20 câu mẫu + 10 âm; kiểm lời văn; Threads; cấu hình compose |
| `should_escalate` | **không có** cờ này ở mã (chỉ có `LowConfidence`) | định nghĩa `ShouldEscalate(intent)` (4.2) — prompt BA nhắc nhưng mã không có |

**Ngoài phạm vi:** xem `US.md` (đầu tệp). **Migration:** `00012_escalation_notify` (số trong phase file `00007` đã bị `chat_threads` dùng; số kế hoạch `00007…00009` của `P4.md` / `P5.md` thay bằng `00012`, `00013`, `00014` — `plan.md`).

## 2. Người dùng và quyền

Quyền theo guard lớp (`FEAT-course-foundation` 4.1): `Member` = thành viên `ACTIVE` (STUDENT, TA, TEACHER); `Staff` = TA + TEACHER `ACTIVE`; `Teacher` = TEACHER; **ADMIN bị từ chối ở mọi route dưới đây** (D39: Admin không đọc nội dung lớp), trừ chỗ ghi rõ.

| # | Hành động | STUDENT (chủ) | STUDENT (khác) | TA | TEACHER | ADMIN |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `POST /chat/messages/{mid}/escalate` | ✓ (tin của phiên mình) | 404 | 403 | 403 | 403 |
| 2 | `POST /chat/tickets/{id}/followup`, `/confirm`, `/confirm/undo` | ✓ (ticket của mình) | 404 | 403 | 403 | 403 |
| 3 | Thấy trạng thái ticket của mình (`ticket` trong tin chat) | ✓ (chỉ `{id,status,created_at,answered_at}`) | – | – | – | – |
| 4 | `GET …/tickets`, `GET …/tickets/{id}`, `GET /me/tickets` | 403 | 403 | ✓ | ✓ | 403 |
| 5 | `claim`, `release` (của mình), `answer`, `dismiss` | 403 | 403 | ✓ | ✓ | 403 |
| 6 | `knowledge-preview`, `POST` / `DELETE …/tickets/{id}/knowledge` | 403 | 403 | ✓ | ✓ | 403 |
| 7 | Đọc **toàn bộ** chat riêng của sinh viên | chỉ chủ | 403 | **không** | **không** | **không** |
| 8 | `PUT /courses/{id}/settings` (ngưỡng) | 403 | 403 | 403 | ✓ | 403 |
| 9 | `PUT /me/settings` `notify_ticket_by_mail` | 422 | – | ✓ | ✓ | 422 |
| 10 | `PUT /me/settings` `notify_answer_by_mail` | ✓ | – | ✓ | ✓ | – |
| 11 | `POST …/threads/{tid}/report`, `…/posts/{pid}/report` | ✓ (không bài của mình) | ✓ | ✓ | ✓ | 403 |
| 12 | `hide`, `unhide`, `dismiss-reports`, `pin`, `unpin`, `GET …/moderation/queue` | 403 | 403 | ✓ | ✓ | 403 |
| 13 | `PUT` / `DELETE` thread / bình luận | ✓ (chủ bài) | 403 | chỉ bài của mình | chỉ bài của mình | 403 |
| 14 | `Verify` / `Correct` / `Reject` (đã có, sprint 6) | 403 | 403 | ✓ | ✓ | 403 |
| 15 | `GET /admin/observability/failed-mail` | 403 | 403 | 403 | 403 | ✓ |
| 16 | Nhận thông báo `COURSE_TICKETS_STALLED` (mã lớp + số lượng, **không** nội dung) | – | – | – | – | ✓ |
| 17 | `GET /notifications`, `POST /notifications/{id}/read` | chỉ của mình | | chỉ của mình | chỉ của mình | chỉ của mình |

**Nguyên tắc dữ liệu (luật cứng, kiểm bằng test):**
- Giảng viên / TA chỉ thấy **đúng cặp hỏi–đáp đã nhờ** + các tin của chính ticket (4.3); không có đường nào liệt kê phiên, tin khác, hay `confidence` của tin chưa nhờ.
- Sinh viên **không** thấy `confidence`, người nhận, `origin`, `sla_stage` (chỉ lời: "Đang chờ giảng viên · …"); ADMIN **không** thấy nội dung ticket (chỉ mã lớp + số).
- Thông báo (chuông) và thư cho giảng viên **không** chứa nội dung câu hỏi / trả lời, tên hay MSSV sinh viên.
- Câu trả lời của giảng viên **không** được chứa định danh sinh viên khác (4.2.6).
- Tin khủng hoảng: không ticket, không báo ai, không nội dung ở bất kỳ bảng nào ngoài `chat_messages` (4.9).

## 3. Luồng chính và các nhánh lỗi

### 3.1 Vòng đời ticket

```mermaid
stateDiagram-v2
  [*] --> OPEN: tự mở (dưới ngưỡng) / sinh viên bấm / giảng viên nhắn (OUTREACH → ANSWERED)
  OPEN --> CLAIMED: Nhận (409 nếu người khác đã nhận)
  OPEN --> ANSWERED: Trả lời (tự nhận)
  CLAIMED --> OPEN: Bỏ nhận (chỉ người nhận)
  CLAIMED --> ANSWERED: Trả lời (chỉ người nhận)
  OPEN --> CLOSED: Đóng không trả lời (giảng viên)
  CLAIMED --> CLOSED: Đóng không trả lời (người nhận)
  ANSWERED --> ANSWERED: Trả lời thêm (người đã trả lời)
  ANSWERED --> OPEN: Hỏi tiếp (sinh viên)
  ANSWERED --> CLOSED: Đã rõ (sinh viên) / 7 ngày không phản hồi
  CLOSED --> OPEN: Hỏi tiếp trong 30 ngày (sinh viên)
```

### 3.2 Escalation → trả lời → thư

```mermaid
sequenceDiagram
  participant SV as Sinh viên (chat)
  participant G as Gateway
  participant W as Worker
  participant R as Redis (SSE)
  participant GV as Giảng viên (/inbox, chuông)
  participant M as Mailpit
  SV->>G: tin hỏi → finish(): câu trả lời DONE, low_confidence
  G->>G: tx: FinishChatMessage + escalation_tickets (OPEN) + outbox escalation.created
  G-->>SV: SSE done {ticket:{status:OPEN}} → "Đang chờ giảng viên · vừa gửi"
  W->>W: handler: notify TICKET_NEW (gộp) + ticket_new mail (nếu bật) + today.invalidate
  W->>R: Publish notification
  R-->>GV: chuông ≤ 5 s
  GV->>G: claim (200 / 409) → answer (Idempotency-Key)
  G->>G: tx: ticket ANSWERED + chat_messages STAFF_ANSWER + outbox escalation.answered
  W->>W: notify TICKET_ANSWERED (SSE) + mail escalation_answered (nếu bật)
  W->>M: SMTP (thử lại 1 s / 5 s / 30 s → DEAD)
  R-->>SV: chuông; mail ≤ 60 s
```

### 3.3 Bảng nhánh lỗi

| Tình huống | Hệ thống phản ứng | Người dùng thấy |
| --- | --- | --- |
| Bấm `Nhờ giảng viên` hai lần / hai tab | khử trùng `message_id`; 1 ticket | một dòng trạng thái |
| Mất mạng khi bấm | gửi lại khi có mạng (cùng `message_id`) | "Đang gửi…" → trạng thái |
| 5 ticket đang mở | `409 TICKET_LIMIT_REACHED`; tự mở bị chặn, đếm `chat_tickets_suppressed_total` | nút khoá + tooltip; câu "AI chưa đủ chắc chắn về câu này" |
| Đang bị khoá giờ thi | `EXAM_LOCKED` | như chat riêng khi đang thi |
| Lớp đã lưu trữ | `409 COURSE_ARCHIVED` | thông báo lớp chỉ đọc |
| Hai người cùng `Nhận` | người sau `409 TICKET_ALREADY_CLAIMED` + `claimed_by` | "Cô Lê Thu Hà đã nhận câu này lúc 14:02" |
| Trả lời chứa tên / MSSV sinh viên khác | `422 PII_IN_ANSWER` + vị trí | vùng cần sửa được tô |
| Trả lời gửi lỗi mạng | giữ nháp; `Idempotency-Key` khử trùng khi thử lại | `Thử lại`, chữ không mất |
| SMTP lỗi tạm | thử lại 3 lần, rồi `DEAD`; câu trả lời trong app không đổi | sinh viên vẫn thấy trả lời + chuông; Admin thấy ở `/observability` |
| SMTP lỗi vĩnh viễn | `DEAD` ngay | như trên |
| Worker chết giữa chừng | outbox giao lại; thư idempotent theo `mail_outbox.id` | đúng một thư |
| Redis phát SSE lỗi | dòng `notifications` vẫn có; chuông rơi về làm mới 30 s | trễ ≤ 30 s |
| Tab thứ 3 mở SSE | `429 SSE_LIMIT_REACHED` | chuông làm mới 30 s, không báo lỗi |
| Ticket quá 24 giờ làm việc | nhắc cả nhóm một lần (`sla_stage=1`) | chuông + thư cho giảng viên / TA |
| Ticket quá 72 giờ làm việc | `Overdue` ở `Hôm nay`; Admin báo nếu lớp không ai hoạt động | mục lên đầu, ghi số giờ chờ |
| Embedding tri thức lỗi | thử lại 3 lần, `DEAD`; xác nhận của giảng viên vẫn thành công | nhãn "Chưa nạp vào tri thức" + `Thử lại` |
| 3 người khác nhau báo cáo | tự ẩn chờ duyệt, một thông báo | chủ bài: "đang được giảng viên xem xét" |
| Sửa bài chứa PII | `422 PII_DETECTED`, không lưu | hộp thoại hai lối |
| Đăng / bình luận khủng hoảng ở Threads | `422 CRISIS_SUPPORT`, không lưu, không AI | hộp thoại hỗ trợ + `Mở chat riêng`; chữ không mất |

## 4. Yêu cầu chức năng

### 4.1 Hằng số và cấu hình (mọi giá trị có tên, nạp lúc khởi động, sai → gateway không chạy và nêu tên biến)

| Tên | Mặc định | Miền | Dùng ở |
| --- | --- | --- | --- |
| `TICKET_OPEN_CAP` | `5` | 1–50 | 4.2.4 |
| `TICKET_REOPEN_DAYS` | `30` | 1–365 | 4.2.7 |
| `TICKET_AUTOCLOSE_DAYS` | `7` | 1–60 | 4.6.3 |
| `TICKET_WORKDAY_START` / `END` | `08:00` / `17:00` | `HH:MM`, START < END | 4.6.1 |
| `TICKET_WORKDAYS` | `1-5` (T2–T6) | tập con 1–7 | 4.6.1 |
| `TICKET_REMIND_AFTER` | `24h` (giờ làm việc) | 1h–720h | 4.6.2 |
| `TICKET_ESCALATE_AFTER` | `72h` (giờ làm việc) | > `TICKET_REMIND_AFTER` | 4.6.2 |
| `TICKET_INACTIVE_AFTER` | `72h` (giờ lịch) | 1h–720h | 4.6.2 |
| `TICKET_SLA_TICK` | `15m` | 1m–1h | 4.6.2 |
| `ANSWER_MAX_CHARS` / `FOLLOWUP_MAX_CHARS` | `4000` / `2000` | | 4.2 |
| `MAIL_TICKET_NEW_COALESCE` | `15m` | 1m–6h | 4.5 |
| `REPORT_AUTOHIDE_THRESHOLD` | `3` | 2–20 | 4.10 |
| `REPORT_RATE_PER_HOUR` | `20` | 1–200 | 4.10 |
| `THREAD_PIN_MAX` | `3` | 0–20 | 4.10 |
| `ESCALATION_THRESHOLD_MIN` / `MAX` / `STEP` | `0.50` / `0.95` / `0.05` | | 4.2.8 |
| `KNOWLEDGE_EMBED_TIMEOUT` | `10s` | 1s–60s | 4.7 |
| `SUPPORT_RESOURCES_VI` | (rỗng → câu mặc định) | ≤ 300 ký tự | 4.9 |

`TICKET_WORKDAY_*`, `TICKET_REMIND_AFTER`, `TICKET_ESCALATE_AFTER` là **tham số**: đổi ý nghĩa "24 / 72 giờ" (Q4) chỉ cần đổi giá trị, không đổi mã.

### 4.2 `internal/escalation` — ticket

**4.2.1 Điều kiện của tin có thể nhờ (`CreateFromMessage`).** Tin `m` phải: `role='ASSISTANT'`, `stream_status='DONE'`, thuộc phiên của người gọi (hoặc của sinh viên trong ngữ cảnh tự mở), phiên chưa xoá mềm, sinh viên `ACTIVE` ở lớp, lớp `ACTIVE`, `m.intent ∉ {CRISIS, OTHER_PERSON, SMALLTALK, STAFF_ANSWER, ESCALATION_FOLLOWUP}`. Ngược lại `422 MESSAGE_NOT_ESCALATABLE`. Chủ phiên ≠ người gọi → `404`.

**4.2.2 Tạo idempotent.** Một giao dịch: khoá `pg_advisory_xact_lock(hashtextextended(student_id::text || course_id::text, 0))` (chặn vượt trần khi song song), `INSERT … ON CONFLICT (message_id) DO NOTHING RETURNING *`; không trả hàng → `SELECT` ticket sẵn có và trả `already:true` (kể cả `CLOSED`). Thành công ghi `outbox` `escalation.created` và `audit_log` (`escalation_ticket.created`, không nội dung) trong cùng giao dịch. Lỗi ở bất kỳ bước nào → rollback toàn bộ.

**4.2.3 Tự mở trong `chat.Service.finish`.** Sau `FinishChatMessage` (cùng `tx`), nếu `meta.ShouldEscalate(intent)` thì gọi `escalation.CreateTx(ctx, tx, …)` (không mở giao dịch mới; `origin='AUTO_LOW_CONFIDENCE'` hoặc `'AUTO_NO_FORMULA'`). Lỗi ghi ticket làm lỗi cả `finish` (câu trả lời **không** `DONE` mà không có ticket — chấp nhận: một `INSERT` đơn giản không thất bại nếu `FinishChatMessage` thành công; việc ngược lại — mất ticket thầm lặng — mới nguy hiểm). Khung SSE `done` và hàng tin thêm trường `ticket` (nullable).

**4.2.4 `ShouldEscalate(intent)` (hàm thuần, `internal/agent`).**

| Điều kiện | Kết quả |
| --- | --- |
| `intent ∈ {CRISIS, OTHER_PERSON, SMALLTALK}` | `false` |
| `degraded = true` | `false` (nút vẫn hiện — sinh viên tự nhờ) |
| `intent = GRADE_FORMULA` và câu trả lời là câu mẫu "chưa có công thức" | `true`, `origin = AUTO_NO_FORMULA` |
| `LowConfidence = true` (kể cả `no_context`) | `true`, `origin = AUTO_LOW_CONFIDENCE` |
| còn lại | `false` |

Trần: `count(*) WHERE course_id AND student_id AND status IN ('OPEN','CLAIMED') ≥ TICKET_OPEN_CAP` → thủ công `409 TICKET_LIMIT_REACHED`; tự động **bỏ qua** (không lỗi, tăng bộ đếm `chat_tickets_suppressed_total{course}`, log không nội dung).

**4.2.5 Chuyển trạng thái (cập nhật có điều kiện, mỗi lần tăng `version`, ghi `audit_log`).**

| Thao tác | Điều kiện SQL | Hiệu ứng |
| --- | --- | --- |
| `claim` | `status='OPEN'` (hoặc đã `CLAIMED` bởi chính mình → 200 không đổi) | `CLAIMED`, `claimed_by`, `claimed_at`. 0 hàng và ticket `CLAIMED` bởi người khác → `409 TICKET_ALREADY_CLAIMED` + `details.claimed_by{id,full_name,role}`, `claimed_at` |
| `release` | `status='CLAIMED' AND claimed_by = người gọi` | `OPEN`, `claimed_by=NULL` (`sla_stage` giữ) |
| `answer` | (`OPEN`) hoặc (`CLAIMED` và `claimed_by` = người gọi) hoặc (`ANSWERED` và `answered_by` = người gọi) | tx: `ANSWERED` (`answer`, `answered_by`, `answered_at`, `claimed_by` = người gọi nếu từ `OPEN`) + tin `STAFF_ANSWER` + outbox `escalation.answered`; khác → `409 TICKET_ALREADY_CLAIMED` / `TICKET_STATE_INVALID` |
| `dismiss {reason}` | `OPEN` hoặc (`CLAIMED` bởi mình) | `CLOSED` (`closed_reason='STAFF_DISMISSED'`, `dismiss_reason`), tin `STAFF_ANSWER` ngắn theo lý do, outbox `escalation.closed` |
| `followup` | `ANSWERED`, hoặc `CLOSED` và `now() - closed_at ≤ TICKET_REOPEN_DAYS` | `OPEN`, `claimed_by=NULL`, `reopened_count+1`, `sla_stage=0`; tin `ESCALATION_FOLLOWUP`; outbox `escalation.followup` |
| `confirm` | `ANSWERED` | `CLOSED` (`STUDENT_CONFIRMED`) |
| `confirm/undo` | `CLOSED AND closed_reason='STUDENT_CONFIRMED' AND closed_at > now()-5s` | `ANSWERED`, `closed_at=NULL` |
| `autoclose` (job) | `ANSWERED AND answered_at < now()-TICKET_AUTOCLOSE_DAYS` | `CLOSED` (`AUTO_7D`) |
| `knowledge` (4.7) | `ANSWERED` / `CLOSED` | `save_as_knowledge` |

Mọi lệnh nhận `version` tuỳ chọn; có và sai → `409 VERSION_CONFLICT` kèm bản hiện tại. Người nhận / trả lời khác nhau cho ticket tái mở: `answered_by` giữ để hiển thị "{tên} đã trả lời trước đó".

**4.2.6 Câu trả lời không chứa định danh người khác.** Trước khi ghi: `privacy.Detect(courseID, body)` (từ điển roster lớp + regex). Mọi khoảng khớp **mà không thuộc chính sinh viên hỏi** (tên, MSSV, email của họ cho phép) → `422 PII_IN_ANSWER`, `details:[{start,end,kind}]` (không lặp giá trị). Dùng đúng quy tắc từ thường / biên từ của US-P3-02 (không bắt "anh", "minh chứng"). Không có nút ghi đè (đề xuất; Q21). Câu hỏi của sinh viên được lưu nguyên như họ gõ (đã ở `chat_messages`).

**4.2.7 `TEACHER_OUTREACH` (khung dùng chung, route ở P5).** `escalation.CreateOutreach(ctx, tx, {course, student, sender, body})`: tạo (hoặc dùng lại) phiên chat riêng "Tin nhắn từ giảng viên" của sinh viên (`chat_sessions` `PRIVATE`, `title` cố định), tin `STAFF_ANSWER` (`blocks.ticket`), ticket `kind=TEACHER_OUTREACH`, `origin=STAFF_OUTREACH`, `source_type='STUDENT'`, `source_id=student_id`, `message_id=NULL`, **trạng thái `ANSWERED`** (`claimed_by=answered_by=sender`), outbox `escalation.answered` (thông báo + thư `outreach_message`). Sinh viên `Hỏi tiếp` → `OPEN` như thường. Trần 5 không áp dụng cho `OUTREACH` (giảng viên chủ động). Sinh viên `PENDING` / `REMOVED` / ngoài lớp → từ chối.

**4.2.8 Ngưỡng (`PUT /courses/{id}/settings`).** Chấp nhận `{escalation_threshold?, version}` (`Teacher`); giá trị ∈ [0,50; 0,95], bội của 0,05 (so sánh bằng `decimal`); sai → `422`; ghi `audit_log` (cũ / mới); `courses.version + 1`; đọc mỗi lần khi trả lời (đã có ở `chat.threshold`). Hàm dự phòng `0.80` của `chat.threshold` giữ nguyên (khớp mặc định mới của `00012`).

### 4.3 Quyền đọc chat riêng của giảng viên (phạm vi hẹp, có kiểm)

`internal/escalation` được đọc `chat_messages` **chỉ** qua hai truy vấn sqlc, cả hai bắt buộc `course_id`:
1. `GetEscalatedExchange(course_id, message_id)`: tin trợ lý `m` + tin `m.reply_to` (câu hỏi `U`) — đúng hai hàng.
2. `ListTicketTranscript(course_id, session_id, ticket_id)`: `intent IN ('STAFF_ANSWER','ESCALATION_FOLLOWUP') AND blocks @> '[{"kind":"ticket","data":{"ticket_id":"<id>"}}]'` sắp theo `(created_at, id)`.

Không có truy vấn liệt kê phiên / tin của sinh viên cho `Staff`. Test kiến trúc `TestEscalationReadsChatOnlyViaTicketQueries` quét mã gói `escalation` (và tệp `store/queries/escalation.sql`) để chắc chỉ có hai truy vấn trên chạm `chat_messages` (cùng tinh thần `TestNoUnscopedChunkQuery`). Phiên đã xoá mềm **không** chặn ticket (Q7, mặc định: giảng viên vẫn thấy cặp hỏi–đáp đã nhờ). Lịch sử gửi cho LLM của phiên **loại** các tin `STAFF_ANSWER` / `ESCALATION_FOLLOWUP` (người viết là giảng viên, không phải trợ lý).

### 4.4 `internal/notify` — một cửa, SSE, gộp

**4.4.1 Một cửa.** `notify.Service.Send(ctx, Spec) (Result, error)`: `Spec{UserID, CourseID?, Type, Title, Body?, Link?, DedupeKey, Aggregate bool}`. Ghi `notifications` bằng pool (người gọi là handler outbox hoặc job, không phải request), rồi `Publisher.Publish(ctx, userID, "notification", frame)` **chỉ khi** dòng được tạo hoặc nội dung gộp đổi. Lỗi `Publish` → log + `notify_publish_errors_total`, trả `nil` (dòng đã có; chuông rơi về làm mới 30 s). Khung: `{id, type, title, body, link, course_id, read:false, created_at, unread_count}` (≤ 64 KiB). Bốn nơi ghi trực tiếp hiện có (`course/notify.go`, `exam/notify.go`, `thread/service.go: notify`, nhắc lịch) chuyển sang gọi `Send`; nơi ở trong request (thread: `THREAD_REPLY`, `THREAD_VERIFIED`, `THREAD_ANSWERED`) đổi thành ghi **outbox** (`thread.reply_created`, `thread.post_decided` — đã có — , `thread.ai_answered`) để handler `notify.*` gọi `Send` (luật "thông báo đi qua outbox"). `dedupe_key` hiện có giữ nguyên.

**4.4.2 Đường SSE: dùng `GET /api/v1/events` (không có `/notifications/stream`).** Lý do: PG đã dựng luồng ghép kênh với `Last-Event-ID`, `resync`, giới hạn 2 kết nối, nhiều gateway qua Pub/Sub (`ep:sse:ch:{uid}`); thêm đường thứ hai làm đôi số kết nối tính vào hạn 2. Đổi so với `ARCHITECTURE.md` §5 / `P4.md` L1 (đề xuất — `proposals.md` #1). `Last-Event-ID` = id Redis Stream (bộ đệm `SSE_BUFFER_MAXLEN`=1000, TTL 1 giờ). "Đọc bù từ bảng" thực hiện bằng `resync`: sự kiện `resync` (`buffer_exceeded`, `invalid_last_event_id`) → khách hàng gọi `GET /notifications?cursor=` và làm mới `unread_count`.

**4.4.3 Gộp.** Loại gộp (`Aggregate=true`), một dòng chưa đọc cho mỗi (người nhận, lớp, loại), `dedupe_key = "agg:<TYPE>:<course_id>"`:

| Type | Người nhận | Tiêu đề (đếm hiện tại, không đếm sự kiện) | Nguồn đếm |
| --- | --- | --- | --- |
| `TICKET_NEW` | TA + TEACHER `ACTIVE` của lớp | "N câu hỏi đang chờ giảng viên" | `count(escalation_tickets) status='OPEN'` |
| `AI_PENDING` | TA + TEACHER | "N câu trả lời AI chờ duyệt" | `forum_posts` `kind='AI' AND verification_state='PENDING' AND hidden_at IS NULL AND deleted_at IS NULL` |
| `POST_REPORTED` | TA + TEACHER | "N bài bị báo cáo chờ xem" | bài có báo cáo `OPEN` hoặc tự ẩn chờ duyệt |
| `TICKET_STALE` | TA + TEACHER (nhóm) / người nhận (cá nhân, `dedupe_key = ticket_stale:<ticket_id>`) | "N câu hỏi chờ quá 24 giờ làm việc" | `count(status='OPEN' AND sla_stage ≥ 1)` |

Cập nhật gộp: `INSERT … ON CONFLICT (user_id, dedupe_key) DO UPDATE SET title=…, body=…, read_at=NULL, created_at=now()` khi số đổi; số = 0 → `UPDATE … SET read_at = now()`. Mỗi lần ticket / bài đổi trạng thái, handler đếm lại và gọi `Send` cho từng người nhận. **Không** gộp: `TICKET_ANSWERED`, `THREAD_REPLY`, `THREAD_VERIFIED`, `THREAD_ANSWERED`, `POST_HIDDEN`, `COURSE_TICKETS_STALLED`, `OUTREACH`.

**4.4.4 Chuông (`GET /notifications`).** Hợp đồng sẵn có `{items[{id,type,title,body,link,course_id,read_at,created_at}], next_cursor, unread_count}`; không đổi. `link` rỗng → `/`. Nội dung `title` / `body`: xem AC6 của US-P4-02 (không riêng tư).

### 4.5 Thư (`internal/mail`)

**4.5.1 Quy tắc payload.** `mail.Enqueue` từ chối khoá `token|password|secret|link|url`; payload chỉ có **định danh**: `escalation_answered` → `{ticket_id, answer_message_id}`; `outreach_message` → `{ticket_id, answer_message_id}`; `ticket_reminder` / `ticket_new` → `{course_id}`; `thread_reply` → `{thread_id, event, post_id?}` (`event` ∈ `REPLY`, `VERIFIED`, `CORRECTED`). Biến hiển thị (câu hỏi, trả lời, tiêu đề thread, liên kết) do **resolver** của mẫu nạp lúc gửi: `mail.Resolver interface{ Resolve(ctx, tx pgx.Tx, row MailOutbox) (map[string]string, error) }` đăng ký theo tên mẫu (`escalation` đăng ký ba, `thread` đăng ký `thread_reply`); `build` gọi resolver thay vì bảng khoá cố định. Lý do: nội dung riêng tư không nằm ở `mail_outbox.payload` / `outbox.payload`; thư lấy đúng dữ liệu của đúng ticket (không lẫn sinh viên khác). Liên kết dựng ở resolver từ `APP_PUBLIC_URL`.

**4.5.2 Mẫu (bổ sung `specs()`).**

| Mẫu | Tới | Tiêu đề (không tên người) | Thân (đoạn) |
| --- | --- | --- | --- |
| `escalation_answered` | sinh viên chủ ticket (`notify_answer_by_mail`) | "Giảng viên đã trả lời câu hỏi của bạn" | "Chào {FullName}," · "Câu hỏi của bạn ở lớp {ClassCode}: {QuestionText}" · "{AnswererRole} {AnswererName} trả lời: {AnswerText}" · "Xem và hỏi tiếp: {ChatURL}" |
| `outreach_message` | sinh viên | "Giảng viên vừa nhắn cho bạn" | "Chào {FullName}," · "{AnswererRole} {AnswererName} (lớp {ClassCode}): {AnswerText}" · "Trả lời trong EduPilot: {ChatURL}" |
| `ticket_reminder` | TA + TEACHER `ACTIVE` | "Có câu hỏi chờ giảng viên quá 24 giờ làm việc" | "Lớp {ClassCode} có {Count} câu hỏi đang chờ." · "Mở hộp thư hỗ trợ: {InboxURL}" |
| `ticket_new` | TA / TEACHER đã bật cờ | "Có câu hỏi mới cần giảng viên" | "Lớp {ClassCode} có {Count} câu hỏi đang chờ." · "{InboxURL}" |
| `thread_reply` | chủ thread (`notify_answer_by_mail`) | "Câu hỏi của bạn trên Threads có trả lời mới" | "{Event} trong câu hỏi «{ThreadTitle}»." · "Xem: {ThreadURL}" |

`QuestionText` cắt 300 ký tự, `AnswerText` cắt 2000 (thêm "… Xem đầy đủ trong EduPilot"). Mọi giá trị qua `html/template` (thoát tự động). Không đính kèm, không ảnh nhúng.

**4.5.3 Khử trùng và gộp.** `dedupe_key`: `escalation_answered:<answer_message_id>`; `outreach_message:<answer_message_id>`; `ticket_new:<user_id>:<course_id>:<floor(unix/900)>` (tối đa một thư mỗi `MAIL_TICKET_NEW_COALESCE`; số trong thư đếm lúc gửi); `ticket_reminder:<user_id>:<course_id>:<yyyymmddHH>` (tối đa một thư mỗi giờ mỗi lớp); `thread_reply:<post_id|thread_id>:<event>:<user_id>`.

**4.5.4 Lỗi và quan sát.** Hành vi sẵn có: lỗi tạm thời → outbox thử lại theo `OUTBOX_RETRY_BACKOFF` = 1 s, 5 s, 30 s (**3 lần thử lại**, tổng 4 lần gọi; `mail_outbox.attempts` ≤ 4), hết → `DEAD` + dead-letter; lỗi vĩnh viễn (`template_unknown`, `payload_invalid`, `ticket_missing`, hộp thư không tồn tại) → `DEAD` ngay. Resolver không tìm thấy ticket → `permanentErr{"ticket_missing"}`. `GET /admin/observability/failed-mail` (ADMIN): `mail_outbox WHERE status='DEAD'` phân trang con trỏ `(updated_at, id)`, trường `{id, template, to_masked, attempts, last_error, updated_at}` (`to_masked` = chữ đầu + `***` + `@domain`); không thân thư, không payload.

### 4.6 SLA và giờ làm việc

**4.6.1 Giờ làm việc.** `WorkingElapsed(from, to) = Σ độ dài giao của [from, to) với khung [START, END) của mọi ngày thuộc `TICKET_WORKDAYS`, múi `Asia/Ho_Chi_Minh` (UTC+7, không DST)`. Không có lịch nghỉ lễ (Nợ). `ReminderAt(created)` = thời điểm đầu tiên `WorkingElapsed(created, t) ≥ TICKET_REMIND_AFTER`; `EscalateAt(created)` tương tự với `TICKET_ESCALATE_AFTER`. Mặc định: 9 giờ / ngày làm việc → 24 giờ làm việc ≈ 2,67 ngày làm việc; 72 giờ làm việc = 8 ngày làm việc (Q4: đối chiếu mục tiêu "không nằm im quá 72 giờ").

**4.6.2 Job `ticket.sla`** (đăng ký `cmd/worker/tasks.go` như `reminderTask`; một bản nhờ khoá leader `ep:lock:ticket.sla`; mỗi `TICKET_SLA_TICK`):
1. Ứng viên giai đoạn 1: `status='OPEN' AND sla_stage=0 AND created_at < now() - 24h(lịch)` (lọc rẻ; chỉ mục `escalation_tickets_sla_idx`), tính `ReminderAt` ở Go; đến hạn → `UPDATE … SET sla_stage=1, sla_stage_at=now() WHERE id=$1 AND sla_stage=0 AND status='OPEN' RETURNING` + `outbox` `escalation.sla_reminder{course_id}` **trong cùng giao dịch** (mỗi lớp mỗi lượt chạy ≤ 1 sự kiện).
2. Giai đoạn 2 tương tự với `EscalateAt` → `sla_stage=2` + `escalation.sla_escalated{course_id}`.
3. Ticket `CLAIMED` quá `ReminderAt(claimed_at)` mà chưa trả lời → nhắc **người nhận** một lần (`TICKET_STALE` cá nhân; không thư).
4. "Lớp không ai hoạt động": không TA / TEACHER `ACTIVE` của lớp nào có `auth_sessions.last_used_at ≥ now() - TICKET_INACTIVE_AFTER` (phiên chưa thu hồi) **và** không có dòng `audit_log` `entity='escalation_ticket'` do họ ghi trong khoảng đó. Chỉ khi đó handler `escalation.sla_escalated` gửi `COURSE_TICKETS_STALLED` cho mọi ADMIN `ACTIVE` (`dedupe_key = stalled:<course_id>:<yyyymmdd>`, một lần mỗi ngày).
5. Giai đoạn đặt về 0 khi ticket tái mở. `Overdue` của `Hôm nay` **không** phụ thuộc job: provider tự tính `EscalateAt` (nên đúng cả khi job chậm).

**4.6.3 Job `ticket.autoclose`** (cùng cơ chế, mỗi giờ): đóng ticket `ANSWERED` quá hạn (4.2.5), ghi `audit_log` `escalation_ticket.autoclosed`, outbox `escalation.closed` (không thông báo cho sinh viên — họ thấy "Đã đóng" khi mở chat).

### 4.7 Kho tri thức của lớp ("Lưu thành tri thức" và nạp lại Threads)

**Quyết định kiến trúc (đề xuất — `proposals.md` #2, Q14).** Tri thức đã xác nhận dùng lại **toàn bộ đường RAG của P8** bằng một dòng `documents` ảo cho mỗi lớp, thay vì thêm nguồn thứ hai vào truy vấn RRF:

- `documents`: `title='Hỏi đáp đã được giảng viên xác nhận'`, `type='OTHER'`, `category='THREAD_KNOWLEDGE'`, `status='READY'`, `visible_to_students=true`, `use_for_rag=true`, `uploaded_by=NULL`, `blob_key/sha256 = NULL`; tạo lười (`knowledge.EnsureDocument(course)` dưới `pg_advisory_xact_lock(hashtextextended('knowledge:'||course_id,0))`, tra theo `(course_id, category)` bằng chỉ mục `documents_course_type_idx`; không cần chỉ mục mới).
- Mỗi mục tri thức = một `content_chunks` (`course_ids=[course]`, `audience='ALL'`, `ord = max+1`, `heading` = 80 ký tự đầu của câu hỏi đã che, `text = "Câu hỏi: …\nTrả lời: …"`, `embedding` từ `llm.Embed` — làn NEAR_REALTIME, hạn `KNOWLEDGE_EMBED_TIMEOUT`) + một `knowledge_items(course_id, source_type POST|TICKET, source_id, chunk_id, created_by)`.
- Sau mọi thêm / sửa / gỡ: outbox `document.changed{course_id, document_id}` (handler `rag.BumpVersion` sẵn có tăng `ep:rag:ver` → cache câu trả lời của lớp vô hiệu). Đây là toàn bộ "semantic cache" của sprint 7 (cache khoá chính xác; **không** có cache theo độ tương đồng — SRS 10).
- **Loại khỏi mọi bề mặt tài liệu:** mọi truy vấn liệt kê / tải / thống kê / chia sẻ `documents` thêm `category IS DISTINCT FROM 'THREAD_KNOWLEDGE'` (`library.sql`, `document.sql`, `courses.sql` `share-from`, công cụ `search_library`); `RagSearch` **không** thêm (cần đoạn này). Test `TestKnowledgeDocHiddenFromLibraryAndDocuments` + test kiến trúc quét `store/queries/*.sql` tìm `FROM documents` thiếu vị từ (trừ danh sách cho phép: `rag.sql`, `knowledge.sql`).
- `knowledge.Upsert(source)` / `knowledge.Remove(source)`: đường duy nhất ghi `content_chunks` của tài liệu này; idempotent theo `(source_type, source_id)`.
- Nguồn **POST**: câu trả lời AI `VERIFIED` / `CORRECTED` (nội dung đã sửa nếu `CORRECTED`) + câu hỏi thread; gỡ khi `REJECT`, ẩn, xoá (bài hoặc thread), hoặc `Loại` sau khi đã xác nhận. Nguồn **TICKET**: giảng viên bấm `Lưu thành tri thức`.
- **Che thông tin cá nhân:** văn bản đi vào chunk luôn qua `privacy.Redact` rồi `Detect` lại; còn khoảng khớp → từ chối (`PII_DETECTED`). Ticket: câu hỏi bị phân loại `PERSONAL_QUESTION` → `422 KNOWLEDGE_PERSONAL`; giảng viên xem trước và sửa bản đã che; `POST …/knowledge {question, answer}` nhận **bản đã sửa** (không đọc lại `chat_messages`).
- Việc nạp chạy ở handler outbox `knowledge.sync` (một lần `Embed`, ≤ `KNOWLEDGE_EMBED_TIMEOUT`); lỗi → thử lại theo outbox rồi `DEAD`; trạng thái "Chưa nạp" hiển thị cho giảng viên suy từ việc `knowledge_items` chưa có chunk; nút `Thử lại` phát lại sự kiện.

### 4.8 `Hôm nay` — nguồn việc mới (bảng bậc; số nhỏ = gấp hơn)

Khoá xếp và khuôn `Item` theo `FEAT-course-foundation` 4.7. Mỗi nguồn gộp **một mục mỗi lớp**; mọi mục đăng ký `today.invalidate` ở sự kiện của nó (≤ 5 s sau khi đổi, ≤ 60 s biến mất).

| Vai | Bậc | Kind | Lý do (tiếng Việt, sinh từ dữ liệu) | Overdue | Href |
| --- | --- | --- | --- | --- | --- |
| TA / TEACHER | 10 | `TICKET` | "{n} câu hỏi đang chờ · {m} câu bạn đã nhận · lâu nhất {Age}"; khi `Overdue`: "{n} câu hỏi chờ quá 72 giờ làm việc · lâu nhất {giờ} giờ" | có `OPEN` với `now ≥ EscalateAt(created)` | `/inbox?course=<id>` |
| TA / TEACHER | 47 | `POST_REPORTED` | "{n} bài bị báo cáo chờ xem" | – | `/threads?view=moderation&course=<id>` |
| Sinh viên | 40 | `STAFF_ANSWER_UNREAD` | "Giảng viên đã trả lời câu hỏi của bạn · {Age} trước" | – | `/chat?session=<id>` |
| ADMIN | 45 | `COURSE_TICKETS_STALLED` | "Lớp {class_code} có {n} câu hỏi chờ quá 72 giờ làm việc" | – | `/` (không nội dung) |

(P5 thêm bậc 15 `SESSION_NOW_UNMARKED` và 95 `STUDENT_ATTENTION` — `FEAT-crm-360` SRS 4.9.) Cập nhật `FEAT-course-foundation` 4.7 bảng bậc theo cột này khi PM duyệt (10).

### 4.9 An toàn con người (đối chiếu sprint 6, phần thiếu)

1. **Bộ luật** trong `internal/agent/intent.go` (`reCrisis`): giữ 10 cụm hiện có; mở rộng theo ba lớp, **cùng một biểu thức chính quy trên văn bản đã bỏ dấu + chữ thường** (phủ có dấu lẫn không dấu): (A) trực tiếp — `tu tu`, `tu sat`, `tu hai`, `tu lam dau`, `ket lieu`; (B) ý định — `muon chet`, `khong muon song`, `chan song`, `khong con muon song`, `khong muon ton tai`, `ket thuc (cuoc song|tat ca|moi thu)`, `bien mat khoi the gioi`, `tha chet`, `chet di`; (C) phương tiện — `nhay lau`, `treo co`, `cat tay`, `uong thuoc ngu qua lieu`; (D) gián tiếp — `khong con ly do de song`, `song (chang|khong) con y nghia`, `(moi nguoi|ai do) (se )?tot hon neu khong co (em|minh|toi)`, `neu (em|minh|toi) chet`; (E) viết tắt — `k muon song`, `ko muon song`, `hok muon song`. Ranh giới từ như hiện có (`\b…\b`). Danh sách chính xác do dev chốt theo bộ test (9.5); luật chỉ **thêm**, không bỏ cụm cũ.
2. **Ưu tiên** giữ nguyên: `CRISIS` thắng mọi intent; không gọi LLM; không embed; không `pii_events` / log nội dung / `audit_log` / `outbox` / `jobs` / `llm_audit`; không ticket (4.2.4); không báo ai.
3. **Câu mẫu** (`replies_vi.go`): giữ `ReplyCrisis` + `SUPPORT_RESOURCES_VI` (hoặc `DefaultSupport`); thêm hằng kiểm tra từ cấm / bắt buộc ở test (AC2). Văn bản chốt: "Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên của bạn nhé. {SUPPORT_RESOURCES_VI}".
4. **Threads:** `thread.Service` (`precheck` chỉ khi bấm Đăng, `Create`, `Comment`) gọi `agent.DetectIntent(text) == IntentCrisis` ở `gate()` **trước** firewall PII; khớp → `422 CRISIS_SUPPORT` (`message` = câu mẫu; `details.actions=["OPEN_PRIVATE_CHAT"]`), không ghi `forum_*`, không `pii_events`, không `thread.created` (không AI trả lời). Giao diện: một hộp thoại bảo vệ (`Dialog`), lối chính `Mở chat riêng` (mang nháp theo cơ chế `Chuyển sang chat riêng` sẵn có, nháp không lưu ở máy chủ), lối phụ `Quay lại sửa`. `precheck` khi đang gõ **không** chặn theo tín hiệu khủng hoảng.
5. **Giờ thi:** khoá chat giờ thi **không** chặn đường hỗ trợ: kiểm `CRISIS` chạy **trước** `exam.Locker` ở `chat.Send` (tin trả câu mẫu, không đếm `RecordChatBlocked`).
6. **Cấu hình:** `SUPPORT_RESOURCES_VI` (≤ 300 ký tự; dài hơn → gateway không chạy) đặt ở `docker-compose.local.yml` bằng giá trị **mô phỏng** (D44); rỗng → `DefaultSupport`.
7. **Giới hạn đã biết (ghi thẳng):** luật từ khoá bỏ sót cách nói rất gián tiếp ("mệt quá rồi, muốn ngủ mãi"), có thể bắt nhầm lời nói quá ("deadline này làm em muốn chết" **vẫn** trả câu hỗ trợ — chọn an toàn; `Q10`); đây là lời đón ân cần và thông tin hỗ trợ chứ không thay chuyên gia; không tự báo cho ai khi chưa có chính sách ghi trong thông báo quyền riêng tư (PR).

### 4.10 Kiểm duyệt Threads

**4.10.1 Báo cáo.** `POST …/threads/{tid}/report` | `…/posts/{pid}/report` `{reason, note?}`; `reason ∈ {SPAM, OFF_TOPIC, PERSONAL_INFO, OFFENSIVE, WRONG_ANSWER, OTHER}`, `note` ≤ 200 chỉ với `OTHER`; mỗi người một lần mỗi bài (`UNIQUE (reporter_id, target_id)`; trùng → `409 ALREADY_REPORTED`); không báo bài của mình (`422 CANNOT_REPORT_OWN`; bài AI không có tác giả nên ai cũng báo được); `≤ REPORT_RATE_PER_HOUR` mỗi giờ mỗi người (`429 RATE_LIMITED`); `DELETE …/report` rút báo cáo trong 5 giây và khi chưa tự ẩn (cho "Hoàn tác"). Sinh viên không thấy số / người báo cáo.

**4.10.2 Tự ẩn.** Trong giao dịch báo cáo, dưới `pg_advisory_xact_lock(hashtextextended(target_id::text,0))`: nếu `count(DISTINCT reporter_id) WHERE target_id AND status='OPEN' ≥ REPORT_AUTOHIDE_THRESHOLD` và bài chưa ẩn → `UPDATE … SET hidden_at=now(), hidden_reason='Nhiều báo cáo, chờ duyệt', hidden_by=NULL WHERE hidden_at IS NULL RETURNING` + outbox `thread.hidden`. `hidden_by IS NULL` = tự động.

**4.10.3 Xử lý.** `hide {reason (1–200), version}` (`hidden_by` = người ẩn; báo cáo `OPEN` → `ACTIONED`); `unhide {version}` (xoá `hidden_*`); `dismiss-reports {version}` (`OPEN` → `DISMISSED`; nếu bài đang tự ẩn thì hiện lại). Mọi lệnh có `audit_log` (`forum_thread|forum_post`, id, hành động, lý do — **không** nội dung bài). `GET …/moderation/queue?status=OPEN|HIDDEN&cursor=` trả `{items:[{kind,thread_id,post_id,title_or_excerpt(≤140),reports:{total,by_reason{}},hidden_at,hidden_by_auto,created_at}]}` phân trang `(created_at, id)`.

**4.10.4 Hiển thị bài ẩn.** Staff thấy bài ẩn với nhãn "Đã ẩn" + lý do; chủ bài thấy bài của mình + dải lý do (không lộ người báo cáo); mọi người khác: bài biến khỏi danh sách / tìm kiếm / "tương tự" / `similar`, chi tiết `404`. Bình luận ẩn trong thread: không hiện (Staff thấy với nhãn). Thread ẩn: bình luận của nó cũng không hiện với người ngoài.

**4.10.5 Sửa / xoá của chủ bài.** `PUT …/threads/{tid} {title, body, tags, version}`, `PUT …/posts/{pid} {body, version}` chạy **cùng `gate()`** như tạo (PII → `422 PII_DETECTED` + hai lối; `redact=true` = "Ẩn thông tin rồi lưu"; khủng hoảng → 4.9.4); câu hỏi đã có bài AI `VERIFIED` / `CORRECTED` → `409 EDIT_LOCKED_VERIFIED`; lưu `edited_at`; không tự bỏ ẩn. `DELETE` = xoá mềm (`deleted_at`), outbox `thread.deleted{thread_id|post_id}` → gỡ tri thức (4.7). Chỉ chủ bài (Staff không sửa / xoá bài người khác; họ `Ẩn`).

**4.10.6 Ghim.** `pin` / `unpin` (Staff), `pinned_at`; ≤ `THREAD_PIN_MAX` ghim mỗi lớp (`409 PIN_LIMIT`, so sánh dưới khoá lớp); `GET …/threads` trả thêm `pinned:[ThreadRow]` ở trang đầu và `items` **loại** thread đang ghim (không lặp); mọi bộ lọc áp lên cả hai.

**4.10.7 Thông báo Threads.** Qua `notify.Service` (4.4): chủ thread — `THREAD_REPLY` (đã có), `THREAD_VERIFIED` (đã có), `THREAD_ANSWERED` (đã có), mới `POST_HIDDEN` / `POST_UNHIDDEN`; Staff — `AI_PENDING` (gộp), `POST_REPORTED` (gộp). Thư `thread_reply` cho chủ thread theo cờ. Không báo chủ thread khi chính họ bình luận hoặc khi bài AI bị `REJECT`.

### 4.11 Sự kiện outbox và handler

Mỗi topic nhiều handler độc lập, idempotent (khử trùng bằng khoá nghiệp vụ); lỗi → thử lại 1 s, 5 s, 30 s rồi dead-letter (như `FEAT-course-foundation` 4.9).

| Topic | Payload | Handler → việc |
| --- | --- | --- |
| `escalation.created` | `{ticket_id,course_id,kind}` | `notify.ticket_new` (đếm lại, gộp, SSE); `mail.ticket_new` (người đã bật cờ); `today.invalidate` |
| `escalation.claimed` / `released` | `{ticket_id,course_id}` | `notify.ticket_new` (đếm lại); `today.invalidate` |
| `escalation.answered` | `{ticket_id,course_id,answer_message_id}` | `notify.ticket_answered` (sinh viên); `mail.escalation_answered` / `outreach_message`; `notify.ticket_new` (đếm lại); `today.invalidate` |
| `escalation.followup` | `{ticket_id,course_id}` | `notify.ticket_new` (đếm lại); `today.invalidate` |
| `escalation.closed` | `{ticket_id,course_id}` | `notify.ticket_new` (đếm lại); `today.invalidate` |
| `escalation.sla_reminder` | `{course_id}` | `notify.ticket_stale` (nhóm); `mail.ticket_reminder` |
| `escalation.sla_escalated` | `{course_id}` | `today.invalidate`; `notify.course_stalled` (nếu lớp không ai hoạt động) |
| `thread.reply_created` | `{thread_id,post_id,course_id}` | `notify.thread_reply` (chủ thread); `mail.thread_reply` |
| `thread.post_decided` (có) | `{post_id,thread_id,course_id,decision}` | `notify.thread_verified`; `mail.thread_reply`; `knowledge.sync` (VERIFY / CORRECT → upsert; REJECT → remove); `notify.ai_pending` (đếm lại) |
| `thread.ai_answered` | `{thread_id,course_id}` | `notify.ai_pending` (đếm lại); `notify.thread_answered` (chủ thread) |
| `thread.reported` | `{target_id,course_id}` | `notify.post_reported` (đếm lại); `today.invalidate` |
| `thread.hidden` / `unhidden` | `{target_id,course_id,auto}` | `notify.post_hidden` (chủ bài); `notify.post_reported` (đếm lại); `knowledge.sync` (gỡ / nạp lại); `today.invalidate` |
| `thread.deleted` | `{thread_id|post_id,course_id}` | `knowledge.sync` (gỡ); `notify.*` đếm lại |
| `knowledge.sync` | `{course_id,source_type,source_id,op}` | `knowledge.Upsert` / `Remove`; phát `document.changed` |
| `document.changed` (có) | `{course_id,document_id}` | `rag.BumpVersion` |

### 4.12 Danh sách FR

| FR | Hệ thống phải… | AC |
| --- | --- | --- |
| FR-1 | `00012` đúng danh sách `ALTER` / `CREATE`; số migration; không `mail_inbound` | P4-01-AC1 |
| FR-2 | Lược đồ `escalation_tickets`, `post_reports`, `knowledge_items`, CHECK, FK phức hợp, mặc định đã sửa, chỉ mục | P4-01-AC2, AC3, AC14 |
| FR-3 | Tạo ticket thủ công idempotent theo `message_id`, kể cả song song | P4-01-AC4, AC12 |
| FR-4 | Tự mở trong cùng giao dịch `finish`; `ShouldEscalate`; không tự mở ở CRISIS / OTHER_PERSON / SMALLTALK / degraded | P4-01-AC5, AC6 |
| FR-5 | Trần ticket mở theo lớp và sinh viên | P4-01-AC9 |
| FR-6 | Điều kiện và quyền tạo (chủ phiên, lớp, khoá giờ thi, lớp lưu trữ) | P4-01-AC11, AC12 |
| FR-7 | Nút `Nhờ giảng viên hỗ trợ`: khi nào hiện, chống bấm đúp, dòng trạng thái, tooltip duy nhất | P4-01-AC7, AC8 |
| FR-8 | Giảng viên chỉ đọc đúng cặp hỏi–đáp + tin của ticket; sinh viên không thấy trường nội bộ | P4-01-AC10, AC11; P4-03-AC3 |
| FR-9 | Hợp đồng API chat cộng thêm `ticket`; golden | P4-01-AC13 |
| FR-10 | `notify.Service` là cửa duy nhất tạo thông báo; idempotent | P4-02-AC1, AC7 |
| FR-11 | Đẩy SSE qua `/events` ≤ 5 s, hai gateway, đọc bù / `resync` | P4-02-AC2, AC3, AC4 |
| FR-12 | Gộp thông báo cùng loại (số hiện tại); thông báo cá nhân không gộp | P4-02-AC5 |
| FR-13 | Thông báo không chứa nội dung riêng tư; chỉ của mình | P4-02-AC6, AC8 |
| FR-14 | Chuông: giới hạn 2 kết nối → làm mới 30 s; UI; lỗi làm mới | P4-02-AC9, AC10, AC11 |
| FR-15 | Danh sách ticket, lọc, phân trang, "tất cả lớp của tôi" | P4-03-AC1, AC2 |
| FR-16 | `claim` 409 hiện người đã nhận; `release` | P4-03-AC4, AC6 |
| FR-17 | `answer` → tin trong chat riêng + outbox, idempotent, trả lời thêm | P4-03-AC5, AC7 |
| FR-18 | Câu trả lời không chứa định danh sinh viên khác | P4-03-AC8 |
| FR-19 | `/inbox`: nháp theo ticket, 375 px, phím tắt (cắt được), trạng thái, thay mock | P4-03-AC9, AC10, AC11, AC13, AC15 |
| FR-20 | `Hôm nay`: `TICKET` (GV/TA), `STAFF_ANSWER_UNREAD` (SV), `COURSE_TICKETS_STALLED` (Admin) | P4-03-AC12; P4-05-AC6, AC8 |
| FR-21 | RBAC mọi route ticket; route lớp đi qua guard | P4-03-AC14 |
| FR-22 | Thư trả lời tới sinh viên ≤ 60 s, idempotent; không mất / không đôi khi worker chết | P4-04-AC1, AC3 |
| FR-23 | Thư không lộ thông tin sinh viên khác; payload chỉ id; payload hỏng → `DEAD` | P4-04-AC2, AC10 |
| FR-24 | Bốn mẫu thư còn lại, gộp `ticket_new`, tiêu đề không tên | P4-04-AC4 |
| FR-25 | Thử lại 3 lần rồi `DEAD`; câu trả lời trong app không bị ảnh hưởng; thư lỗi ở `/observability` | P4-04-AC5, AC9 |
| FR-26 | Cờ thư theo vai; `/settings`; không đường HTTP đọc `mail_outbox`; chỉ Mailpit | P4-04-AC6, AC7, AC8 |
| FR-27 | Hỏi tiếp, `Đã rõ` (+ hoàn tác 5 s), tự đóng 7 ngày | P4-05-AC1, AC2, AC3 |
| FR-28 | `WorkingElapsed`; job SLA 24 / 72 giờ làm việc; báo Admin khi không ai hoạt động; crash-safe | P4-05-AC4, AC5, AC6, AC13 |
| FR-29 | Sinh viên thấy trạng thái và thời gian đã chờ bằng lời | P4-05-AC7 |
| FR-30 | `Đóng không trả lời` | P4-05-AC9 |
| FR-31 | Lưu thành tri thức: xem trước đã che, PII / câu hỏi cá nhân bị từ chối, dùng được ≤ 10 s, gỡ / sửa được, không lọt thư viện | P4-05-AC10, AC11 |
| FR-32 | Thanh trượt ngưỡng + `PUT …/settings` | P4-05-AC12 |
| FR-33 | Báo cáo (lý do chọn sẵn, một lần mỗi bài, tự ẩn ở 3 người khác nhau, cuộc đua) | P4-06-AC1, AC2, AC15 |
| FR-34 | Hàng chờ, `Ẩn bài` có lý do, `Bỏ ẩn`, `Giữ bài`, người đăng được báo và thấy lý do | P4-06-AC3, AC4 |
| FR-35 | Chủ bài sửa (chạy lại tường lửa, khoá khi đã xác nhận) / xoá | P4-06-AC5, AC6 |
| FR-36 | Ghim, lọc | P4-06-AC7 |
| FR-37 | Thông báo / thư Threads qua `notify`; `AI_PENDING` gộp | P4-06-AC8, AC9 |
| FR-38 | `Hôm nay`: bài bị báo cáo | P4-06-AC10 |
| FR-39 | `Loại` / ẩn / xoá gỡ khỏi RAG; `Verify` / `Correct` nạp lại tri thức; lỗi embedding không ảnh hưởng xác nhận | P4-06-AC11, AC12 |
| FR-40 | Ma trận quyền kiểm duyệt; UI Threads | P4-06-AC13, AC14 |
| FR-41 | Bộ luật khủng hoảng mở rộng: ≥ 19/20 dương, 0/10 âm | P4-07-AC1 |
| FR-42 | Câu trả lời khủng hoảng ân cần, không từ chối, không tư vấn chuyên môn | P4-07-AC2 |
| FR-43 | Không tác dụng phụ; staff không đọc tin khủng hoảng | P4-07-AC3, AC8 |
| FR-44 | Khủng hoảng ở Threads: không lưu, không AI, hộp thoại hỗ trợ | P4-07-AC4 |
| FR-45 | Hỗ trợ không bị khoá giờ thi; cấu hình `SUPPORT_RESOURCES_VI` | P4-07-AC5, AC6 |
| FR-46 | Giới hạn của luật từ khoá được ghi thẳng | P4-07-AC7 |

## 5. Dữ liệu

### 5.1 Migration `00012_escalation_notify.sql` (một tệp, một giao dịch)

`CREATE`: enum `ticket_status`, `ticket_kind`, `ticket_origin`, `post_report_reason`, `post_report_status`; bảng `escalation_tickets`, `post_reports`, `knowledge_items`; chỉ mục; trigger `set_updated_at`. **`ALTER` có chủ ý (bốn câu; PM / chủ dự án duyệt — `proposals.md` #2, cùng tiền lệ `00010`/`00011`):**
1. `ALTER TABLE forum_threads ADD COLUMN hidden_at timestamptz, ADD COLUMN hidden_reason text, ADD COLUMN hidden_by uuid REFERENCES users (id), ADD COLUMN edited_at timestamptz` + `CHECK ((hidden_at IS NULL) = (hidden_reason IS NULL) AND char_length(hidden_reason) <= 200)` + chỉ mục `forum_threads_hidden_idx (course_id, hidden_at) WHERE hidden_at IS NOT NULL`. Lý do: `forum_posts` đã có `hidden_*` nhưng câu hỏi gốc (thread) thì chưa; ẩn bằng `deleted_at` mất lý do và không đảo ngược được.
2. `ALTER TABLE forum_posts ADD COLUMN edited_at timestamptz` (nhãn "đã sửa").
3. `ALTER TABLE user_settings ALTER COLUMN notify_ticket_by_mail SET DEFAULT false` + `UPDATE user_settings SET notify_ticket_by_mail=false WHERE version=1` (`00003` đặt `true`, PRD M3 / `P4.md` L2 ghi mặc định **tắt**).
4. `ALTER TABLE courses ALTER COLUMN escalation_threshold SET DEFAULT 0.80` + `UPDATE courses SET escalation_threshold=0.80 WHERE version=1 AND escalation_threshold=0.60` (`00003` đặt `0.60`; PRD M3, US-P3-07 AC2 ghi mặc định `0,80`; `chat.threshold` dự phòng `0.80`) — Q6.

`Down` hoàn tác các `ALTER` (bỏ cột, đặt lại mặc định cũ) rồi `DROP` bảng / enum. Không `mail_inbound`.

### 5.2 `escalation_tickets`

| Cột | Kiểu | Null | Mặc định | Ghi chú |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | `uuidv7()` | PK; `UNIQUE (course_id, id)` |
| `course_id` | `uuid` | NOT NULL | – | FK `courses` |
| `kind` | `ticket_kind` | NOT NULL | `'QUESTION'` | QUESTION / GRADE_APPEAL / TEACHER_OUTREACH |
| `origin` | `ticket_origin` | NOT NULL | – | AUTO_LOW_CONFIDENCE / AUTO_NO_FORMULA / STUDENT_REQUEST / STAFF_OUTREACH / STUDENT_APPEAL |
| `status` | `ticket_status` | NOT NULL | `'OPEN'` | OPEN / CLAIMED / ANSWERED / CLOSED |
| `source_type` | `text` | NOT NULL | – | `CHAT_MESSAGE` \| `STUDENT` \| `SUBMISSION` (P7); `CHECK (source_type ~ '^[A-Z_]{3,30}$')` |
| `source_id` | `uuid` | NOT NULL | – | không FK (đa hình) |
| `message_id` | `uuid` | NULL | – | **UNIQUE**; FK phức hợp `(course_id, message_id) → chat_messages (course_id, id)`; `kind='QUESTION'` ⇒ NOT NULL |
| `session_id` | `uuid` | NULL | – | FK phức hợp `(course_id, session_id) → chat_sessions (course_id, id)` (khử chuẩn để đọc transcript) |
| `student_id` | `uuid` | NOT NULL | – | FK `users` |
| `confidence` | `numeric(4,3)` | NULL | – | chỉ nhân viên đọc; `CHECK BETWEEN 0 AND 1` |
| `appeal_payload` | `jsonb` | NULL | – | `kind='GRADE_APPEAL'` ⇔ NOT NULL (`jsonb_typeof='object'`) |
| `claimed_by` | `uuid` | NULL | – | FK `users`; `OPEN` ⇒ NULL; `CLAIMED` ⇒ NOT NULL |
| `claimed_at` | `timestamptz` | NULL | – | |
| `answer` | `text` | NULL | – | trả lời gần nhất (≤ 4000) |
| `answered_by` | `uuid` | NULL | – | FK `users`; `ANSWERED` ⇒ NOT NULL |
| `answered_at` | `timestamptz` | NULL | – | mốc tự đóng 7 ngày |
| `closed_at` | `timestamptz` | NULL | – | `CLOSED` ⇒ NOT NULL |
| `closed_reason` | `text` | NULL | – | `STUDENT_CONFIRMED` / `AUTO_7D` / `STAFF_DISMISSED` |
| `dismiss_reason` | `text` | NULL | – | `OUT_OF_SCOPE` / `DUPLICATE` / `NEED_MORE_INFO` |
| `reopened_count` | `smallint` | NOT NULL | `0` | |
| `sla_stage` | `smallint` | NOT NULL | `0` | 0 / 1 (nhắc) / 2 (lên đầu); `CHECK BETWEEN 0 AND 2` |
| `sla_stage_at` | `timestamptz` | NULL | – | |
| `save_as_knowledge` | `boolean` | NOT NULL | `false` | |
| `version` | `integer` | NOT NULL | `1` | `CHECK >= 1` |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | `now()` | trigger |

Chỉ mục: PK · `UNIQUE (message_id)` · `escalation_tickets_course_status_idx (course_id, status, created_at, id)` · `escalation_tickets_student_idx (course_id, student_id, status, created_at DESC, id DESC)` · `escalation_tickets_sla_idx (created_at, id) WHERE status IN ('OPEN','CLAIMED')` · `escalation_tickets_autoclose_idx (answered_at) WHERE status = 'ANSWERED'` · `escalation_tickets_claimed_idx (course_id, claimed_by) WHERE status = 'CLAIMED'`. (Cột bổ sung so với `ARCHITECTURE.md` §4: `origin`, `session_id`, `answered_by`, `closed_*`, `dismiss_reason`, `reopened_count`, `sla_stage*`, `version`, `created/updated` — D45: mọi cột nằm trong `CREATE`.)

### 5.3 `post_reports`

| Cột | Kiểu | Null | Ghi chú |
| --- | --- | --- | --- |
| `id` | `uuid` | NOT NULL | PK `uuidv7()` |
| `course_id` | `uuid` | NOT NULL | FK `courses` |
| `thread_id` | `uuid` | NULL | FK phức hợp `(course_id, thread_id) → forum_threads (course_id, id)` |
| `post_id` | `uuid` | NULL | FK phức hợp `(course_id, post_id) → forum_posts (course_id, id)` |
| `target_id` | `uuid` | NOT NULL | `GENERATED ALWAYS AS (COALESCE(thread_id, post_id)) STORED`; `CHECK ((thread_id IS NULL) <> (post_id IS NULL))` |
| `reporter_id` | `uuid` | NOT NULL | FK `users` |
| `reason` | `post_report_reason` | NOT NULL | SPAM / OFF_TOPIC / PERSONAL_INFO / OFFENSIVE / WRONG_ANSWER / OTHER |
| `note` | `text` | NULL | ≤ 200; chỉ `OTHER` |
| `status` | `post_report_status` | NOT NULL `'OPEN'` | OPEN / DISMISSED / ACTIONED |
| `handled_by`, `handled_at` | `uuid`, `timestamptz` | NULL | cặp cùng có / cùng không |
| `created_at`, `updated_at` | `timestamptz` | NOT NULL | trigger |

Chỉ mục: `UNIQUE (reporter_id, target_id)` · `post_reports_open_idx (course_id, status, created_at, id)` · `post_reports_target_idx (target_id) WHERE status = 'OPEN'`. (Cột `target_id`, `status`, `course_id`, `note`, hai khoá phức hợp so với `ARCHITECTURE.md` `post_reports(post_id, reporter_id, reason)`.)

### 5.4 `knowledge_items`

`id uuid PK`; `course_id uuid NOT NULL` FK `courses`; `source_type text NOT NULL CHECK (IN ('POST','TICKET'))`; `source_id uuid NOT NULL`; `chunk_id uuid NOT NULL REFERENCES content_chunks (id) ON DELETE CASCADE`; `created_by uuid` FK `users`; `created_at`, `updated_at`. `UNIQUE (source_type, source_id)`; `knowledge_items_course_idx (course_id, created_at DESC)`. (Bảng mới, **không** có trong `ARCHITECTURE.md` — đề xuất, `proposals.md` #2.)

### 5.5 Redis

| Khoá | Dùng | TTL |
| --- | --- | --- |
| `ep:sse:buf:{uid}`, `ep:sse:ch:{uid}`, `ep:sse:conn:{uid}` | SSE (có từ PG) | theo PG |
| `ep:lock:ticket.sla`, `ep:lock:ticket.autoclose` | leader job | 2 × tick |
| `ep:rl:report:{uid}` | đếm báo cáo mỗi giờ | 1 giờ |
| `ep:rag:ver:{course}` | phiên bản tri thức (có từ P3) | – |

### 5.6 Hợp đồng chat (cộng thêm)

`chat_messages` (đã có `intent`, `blocks`): tin `STAFF_ANSWER` = `role=ASSISTANT`, `user_id` = chủ phiên (FK phức hợp chủ phiên không cho đặt người khác — người viết nằm ở `blocks`), `stream_status='DONE'`, `confidence=NULL`, `blocks=[{"kind":"ticket","data":{"ticket_id":"…","answered_by":{"full_name":"…","role":"TEACHER|TA"}}}]`; tin `ESCALATION_FOLLOWUP` = `role=USER`, `client_msg_id` = khoá của lời gọi. **Không** thay đổi bảng `chat_messages`.

## 6. API

Thân lỗi `{code, message, details?, retry_after?}` (`ARCHITECTURE.md` §5). Mọi route lớp đi qua `CourseAccessGuard` (kiểm bằng `TestAllCourseRoutesGuarded`). Phân trang con trỏ `?cursor=&limit=` (≤ 100). **Route mới ngoài `ARCHITECTURE.md` §5** (đề xuất PM cập nhật): `GET …/tickets/{id}`, `release`, `dismiss`, `knowledge-preview`, `knowledge` (POST / DELETE), `GET /me/tickets`, `POST /chat/tickets/{id}/followup|confirm|confirm/undo`, `PUT …/settings` (mục tiêu P4), `GET …/moderation/queue`, `…/dismiss-reports`, `…/unhide`, `…/pin|unpin`, `…/report` (DELETE), `PUT/DELETE …/threads/{tid}`, `GET /admin/observability/failed-mail`.

### 6.1 Ticket (sinh viên)

| # | Route | Quyền | `Idempotency-Key` | Thành công | Lỗi chính |
| --- | --- | --- | --- | --- | --- |
| 1 | `POST /chat/messages/{mid}/escalate` | STUDENT chủ | không bắt buộc | `201 {ticket,already:false}` / `200 {…,already:true}` | `404`, `403`, `409 TICKET_LIMIT_REACHED`, `409 COURSE_ARCHIVED`, `EXAM_LOCKED`, `422 MESSAGE_NOT_ESCALATABLE`, `503` |
| 2 | `POST /chat/tickets/{id}/followup {content, client_msg_id}` | STUDENT chủ | `client_msg_id` (DB) | `200 {ticket, message}` | `404`, `409 TICKET_STATE_INVALID`, `409 TICKET_REOPEN_EXPIRED`, `422` |
| 3 | `POST /chat/tickets/{id}/confirm` · `/confirm/undo` | STUDENT chủ | – | `200 {ticket}` | `404`, `409 TICKET_STATE_INVALID` |

`ticket` (sinh viên) = `{id, status, created_at, answered_at?}`; thêm vào tin trợ lý ở `GET /chat/sessions/{sid}/messages` và khung SSE `done` (`ticket: null` khi không có).

### 6.2 Ticket (giảng viên / TA)

| # | Route | Quyền | `Idempotency-Key` | Thành công | Lỗi chính |
| --- | --- | --- | --- | --- | --- |
| 4 | `GET /courses/{id}/tickets?status=&cursor=&limit=` | `Staff` | – | `200 {items,next_cursor}` | `403` |
| 5 | `GET /me/tickets?status=&cursor=` | TA / TEACHER (mọi lớp mình) | – | `200 {items,next_cursor}` (kèm `course`) | `403` (ADMIN / STUDENT) |
| 6 | `GET /courses/{id}/tickets/{tid}` | `Staff` | – | `200 {ticket, question, ai_answer, transcript}` | `404` |
| 7 | `POST …/tickets/{tid}/claim` | `Staff` | – | `200 {ticket}` | `409 TICKET_ALREADY_CLAIMED` |
| 8 | `POST …/tickets/{tid}/release` | `Staff` (người nhận) | – | `200 {ticket}` | `409` |
| 9 | `POST …/tickets/{tid}/answer {body, version?}` | `Staff` | **bắt buộc** | `200 {ticket, message}` | `409`, `422 PII_IN_ANSWER`, `422` (độ dài) |
| 10 | `POST …/tickets/{tid}/dismiss {reason, version?}` | `Staff` | – | `200 {ticket}` | `409`, `422` |
| 11 | `GET …/tickets/{tid}/knowledge-preview` | `Staff` | – | `200 {question, answer}` (đã `Redact`) | `409`, `422 KNOWLEDGE_PERSONAL` |
| 12 | `POST …/tickets/{tid}/knowledge {question, answer}` · `DELETE …/knowledge` | `Staff` | – | `201`/`200` · `204` | `422 PII_DETECTED`, `422 KNOWLEDGE_PERSONAL`, `409` |
| 13 | `PUT /courses/{id}/settings {escalation_threshold?, version}` | `Teacher` | – | `200 {course}` | `409 VERSION_CONFLICT`, `422` |

`ticket` (nhân viên) = `{id, kind, status, student{id,full_name}, claimed_by?, answered_by?, age_minutes, sla_stage, reopened_count, save_as_knowledge, course{id,class_code}, version, created_at}`. `ai_answer` = `{content, citations, blocks, confidence}`; `transcript` = `[{id, role, author:{full_name,role}|"student", content, created_at}]`.

### 6.3 Thông báo, cài đặt, quan sát

| # | Route | Quyền | Ghi chú |
| --- | --- | --- | --- |
| 14 | `GET /api/v1/events` (SSE, **có từ PG**) | JWT | sự kiện `notification` (4.4.1); `Last-Event-ID`; `resync` |
| 15 | `GET /notifications`, `POST /notifications/{id}/read` | JWT (chủ) | không đổi hợp đồng |
| 16 | `GET/PUT /me/settings` | JWT | thêm kiểm vai: `notify_ticket_by_mail` chỉ TA / TEACHER (STUDENT → `422`); `notify_answer_by_mail` mọi vai |
| 17 | `GET /admin/observability/failed-mail?cursor=` | ADMIN | 4.5.4 |

### 6.4 Kiểm duyệt Threads

| # | Route | Quyền | Ghi chú |
| --- | --- | --- | --- |
| 18 | `POST …/threads/{tid}/report`, `…/posts/{pid}/report` `{reason, note?}` · `DELETE` cùng đường (rút) | `Member` | `201`; `409 ALREADY_REPORTED`; `422 CANNOT_REPORT_OWN`; `429 RATE_LIMITED` |
| 19 | `POST …/threads/{tid}/hide`, `…/posts/{pid}/hide` `{reason, version}` · `…/unhide {version}` · `…/dismiss-reports {version}` | `Staff` | `200`; `409 VERSION_CONFLICT`; `404` |
| 20 | `GET …/moderation/queue?status=&cursor=` | `Staff` | 4.10.3 |
| 21 | `POST …/threads/{tid}/pin`, `…/unpin` | `Staff` | `409 PIN_LIMIT` |
| 22 | `PUT/DELETE …/threads/{tid}` `{title,body,tags,version,redact?}`; `PUT/DELETE …/posts/{pid}` `{body,version,redact?}` | chủ bài | `422 PII_DETECTED`, `422 CRISIS_SUPPORT`, `409 EDIT_LOCKED_VERIFIED`, `409 VERSION_CONFLICT`, `403` |
| 23 | `GET …/threads` (đã có) | `Member` | thêm `pinned[]`; `items` loại thread ghim; bài ẩn lọc theo 4.10.4 |
| 24 | `POST …/threads`, `POST …/threads/{tid}/posts`, `POST …/threads/precheck` (đã có) | `Member` | thêm `422 CRISIS_SUPPORT` ở `Create` / `Comment` |

### 6.5 Mã lỗi mới (thêm vào `apierr`)

`TICKET_ALREADY_CLAIMED` (409), `TICKET_STATE_INVALID` (409), `TICKET_LIMIT_REACHED` (409), `TICKET_REOPEN_EXPIRED` (409), `MESSAGE_NOT_ESCALATABLE` (422), `PII_IN_ANSWER` (422), `KNOWLEDGE_PERSONAL` (422), `ALREADY_REPORTED` (409), `CANNOT_REPORT_OWN` (422), `EDIT_LOCKED_VERIFIED` (409), `PIN_LIMIT` (409), `CRISIS_SUPPORT` (422). Dùng lại: `VERSION_CONFLICT`, `PII_DETECTED`, `COURSE_ARCHIVED`, `EXAM_LOCKED`, `RATE_LIMITED`, `SSE_LIMIT_REACHED`.

## 7. Giao diện

Chữ theo `CLAUDE.md` (luật chữ UI): **không** câu mô tả dưới tiêu đề / dòng; tooltip chỉ khi thiếu nó người dùng làm sai (hệ quả của lựa chọn, lý do khoá nút); mỗi dòng: nhãn + điều khiển + giá trị; trạng thái 1–2 từ; ví dụ cú pháp ở `placeholder`. Chỉ token `--ep-*` và primitive `frontend/src/shared/ui`; thành phần miền mới `EscalationRow` ở `shared/domain` (ARCH §7). `bash scripts/ui-antipatterns.sh` sạch. Khung hợp đồng: `DESIGN.md` §14.2, §14.3, §14.4, §14.5, D59 (mỗi vùng làm việc một `Panel`).

### 7.1 `/inbox` (thay mock — `DESIGN.md` §14.5)

- **Khung nhìn đầu (≥ 1100 px):** `PageHeader` "Hộp thư hỗ trợ" + bộ chọn lớp riêng (một lớp / "Tất cả lớp của tôi"); trái: `Panel` danh sách 380 px; phải: `Panel` chi tiết. Bộ lọc `SegmentedControl`: `Đang chờ` (mặc định) · `Đã nhận` · `Đã trả lời` · `Tất cả`.
- **`EscalationRow`:** (1) tên sinh viên + mã lớp (chỉ khi "Tất cả lớp"), (2) câu hỏi 1 dòng cắt 80 ký tự, (3) tuổi ("3 giờ") và `StatusText` (`Đang chờ` đỏ / `Đã nhận` hổ phách / `Đã trả lời` xanh / `Đã đóng` xám); khi `sla_stage ≥ 1` tuổi tô đỏ (tín hiệu "cần hành động"), **không** thêm câu giải thích. Không nút trên dòng.
- **Chi tiết:** đầu: tên sinh viên · lớp · tuổi · trạng thái; khối "Câu hỏi", khối "Câu trả lời của AI" (trích nguồn mở tại chỗ, khối kết quả tool) với "Độ tin cậy 0,52" nhạt (thuật ngữ cho giảng viên, `DESIGN.md` §13); `transcript` (tin của giảng viên / sinh viên theo thứ tự). **Một** hành động chính theo trạng thái: `OPEN` → `Nhận`; `CLAIMED` (mình) → ô trả lời + `Gửi trả lời`; `CLAIMED` (người khác) → dòng "Cô … đang xử lý" không có nút; `ANSWERED` → "Trả lời thêm". Hành động phụ (menu `…`): `Bỏ nhận`, `Đóng không trả lời`. Sau khi đã trả lời: dòng `Lưu thành tri thức` (mở xem trước đã che, sửa được, `Lưu`; đã lưu → "Đã lưu thành tri thức · Gỡ").
- **Lạc quan + hoàn tác:** `Nhận` đổi trạng thái ngay, 409 → hoàn lại + `InlineNotice`; `Gửi trả lời` **không** hoàn tác được (thư đã xếp hàng) nhưng **không** hộp thoại xác nhận (không thuộc danh sách việc không đảo ngược của `DESIGN.md` §10.11); thành công → dòng tĩnh "Đã gửi" và sang ticket kế khi dùng `⌘↵`.
- **Nháp:** `useAutosaveDraft('ticket:'+id)`.
- **Điện thoại (< 720 px):** danh sách toàn màn hình → chi tiết toàn màn hình (`?ticket=`), ô trả lời dính đáy, vùng chạm ≥ 44 px, không tràn ngang; zoom 200 % vẫn thao tác được.
- **Phím tắt** (cắt được): `J`/`K`, `R`, `⌘↵`; bản đồ trong `CommandPalette`.
- **Trạng thái:** tải = khung xương 6 dòng; rỗng = "Không có câu hỏi nào đang chờ." + `Xem tất cả`; lỗi = `ApiErrorNotice` + `Thử lại`; PII trong trả lời: tô vùng, nút `Gửi trả lời` khoá tới khi sửa.

### 7.2 `/chat` (sinh viên)

- Nút `Nhờ giảng viên hỗ trợ` (variant phụ, cùng hàng với Hữu ích / Không hữu ích) dưới mọi câu trả lời `DONE` đủ điều kiện (4.2.1); tooltip duy nhất "Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI".
- Câu dưới ngưỡng đã tự mở: dòng chữ nhạt "AI chưa đủ chắc chắn về câu này" (đã có) **thay** nút bằng một dòng trạng thái (ba trạng thái: `Đang chờ giảng viên · vừa gửi` / `· đã gửi 3 giờ trước`; `Giảng viên đã nhận · đã chờ 5 giờ`; `Đã đóng`). Không icon cảnh báo đỏ; màu đỏ chỉ khi cần hành động của chính sinh viên (không có ở đây).
- Tin `STAFF_ANSWER`: nhãn `VerificationState`-kiểu "Giảng viên · {tên}" / "Trợ giảng · {tên}" (viền trái 1 px xanh như tin đã xác nhận), không nút phản hồi; dưới tin cuối của ticket `ANSWERED`: hai nút `Hỏi tiếp` (mở ô nhập nhỏ ngay dưới tin, nháp tự lưu) và `Đã rõ` (hoàn tác 5 giây tại chỗ).
- Sinh viên đã có 5 ticket mở: nút khoá, tooltip "Bạn đang có 5 câu hỏi chờ giảng viên".
- Hộp nhập của chat không đổi; mọi thứ dùng được ở 375 px.

### 7.3 Chuông (header)

Nhận khung SSE `notification` qua `useSSE` (một kết nối dùng chung cả ứng dụng); số chưa đọc "9+"; danh sách mới nhất trước; mục gộp cập nhật tại chỗ; không toast; `aria-live="polite"`; 429 / lỗi SSE → `refetchInterval: 30_000` như hiện tại; `resync` → làm mới truy vấn.

### 7.4 `/threads`, `/threads/[id]`

- Menu `…` của bài: `Báo cáo` (chọn lý do từ danh sách chọn sẵn; "Lý do khác" có ô ≤ 200 ký tự `placeholder` "Mô tả ngắn"), `Sửa`, `Xoá` (chủ bài), `Ẩn bài` / `Bỏ ẩn` / `Giữ bài` / `Ghim` (Staff). Không quá hai hành động hiện sẵn.
- Sau `Báo cáo`: dòng tĩnh "Đã báo cáo · Hoàn tác" 5 giây tại chỗ (`useUndoableAction`).
- `Xoá`: `<ConfirmIrreversible>` với hậu quả bằng số ("Câu hỏi và 4 bình luận sẽ bị xoá").
- Bài của mình bị ẩn: dải `InlineNotice` một dòng với lý do; Staff thấy "Đã ẩn" + lý do + nút `Bỏ ẩn`.
- `view=moderation` (Staff): danh sách hàng chờ (`DataTable`, không card): bài, số báo cáo, lý do gộp theo loại, thời gian; hành động ở chi tiết.
- Ghim: icon nhỏ + thứ tự (ghim lên đầu trang đầu), không thẻ lớn.
- Hộp thoại khủng hoảng (`Dialog`): lời ân cần + thông tin hỗ trợ + `Mở chat riêng` (chính), `Quay lại sửa`.

### 7.5 `/class/settings`, `/settings`

- `/class/settings`: một dòng "Ngưỡng chuyển cho giảng viên" + thanh trượt `Slider` 0,50–0,95 (bước 0,05), "Ít chuyển" ở đầu, "Chuyển nhiều" ở cuối, giá trị hiện bằng dấu phẩy ("0,80"); lưu lạc quan, 409 hoàn lại + `InlineNotice`; chỉ TEACHER thấy điều khiển hoạt động (TA chỉ đọc giá trị).
- `/settings`: công tắc "Mail báo ticket mới" (TA / TEACHER; tắt mặc định), "Mail khi giảng viên trả lời" (mọi vai; bật mặc định); lưu lạc quan, không toast.

### 7.6 `/observability` (ADMIN) — một mục

Đầu trang: mục "Thư chưa gửi được" (`DataTable` cột: Mẫu · Người nhận (đã che) · Số lần thử · Lỗi · Lúc); rỗng "Không có thư lỗi." Phần còn lại giữ mock tới P10.

## 8. Phi chức năng áp dụng

| Hạng mục | Yêu cầu | Đo |
| --- | --- | --- |
| Chuông | ≤ 5 s từ commit ticket tới khung SSE (p95, 20 lần) | `TestEscalationToMail` (`bell_latency_p95_ms`) |
| Thư | ≤ 60 s sau khi gửi trả lời (p95, 10 lần); đúng một thư khi worker chết | `TestEscalationToMail`, `TestOutboxSurvivesWorkerKill` |
| Đọc / ghi API | đọc p95 ≤ 300 ms, ghi p95 ≤ 500 ms (SLO) kể cả `GET …/tickets` trên 10.000 ticket | k6 `benchmarks/load/tickets.js` (khung PG); `TestEscalationIndexesUsed` |
| Phân trang | con trỏ mọi danh sách (`tickets`, `moderation/queue`, `failed-mail`, `notifications`); `OFFSET` cấm | review + test |
| Idempotency | `answer` bắt buộc `Idempotency-Key`; `escalate` khử trùng bằng `message_id`; `followup` bằng `client_msg_id`; thư bằng `dedupe_key`; thông báo bằng `dedupe_key` | các test AC |
| Khoá lạc quan | `version` ở ticket, thread, post, course; sai → `409 VERSION_CONFLICT` | `TestModerationVersionConflict`, `TestSettingsThresholdVersion` |
| Không trạng thái | không giữ gì trong tiến trình; job bằng khoá leader Redis; giao dịch cho mọi chuyển trạng thái | review |
| Hạn mức LLM (D47) | Không có lời gọi LLM sinh chữ ở sprint này; chỉ `Embed` (tri thức, một lần mỗi mục) ở làn NEAR_REALTIME | `TestKnowledgeOneEmbedPerItem` |
| Riêng tư | bảng quyền 2; AC10/11 P4-01, AC6 P4-02, AC2 P4-04, AC3 P4-07; **không** nội dung ở `outbox.payload`, `mail_outbox.payload`, `notifications`, `audit_log`, log | quét canary |
| An toàn tải | ≤ 5 ticket mở / sinh viên / lớp; ≤ 20 báo cáo / giờ / người; ≤ 3 ghim / lớp; `answer` ≤ 4000 ký tự | test AC |
| Cache | tri thức đổi → `ep:rag:ver` tăng (vô hiệu cache câu trả lời); dữ liệu ticket không cache phía máy chủ; `Hôm nay` cache 60 s + `today.invalidate` | `TestKnowledgeBumpsRagVersion`, `today` tests |
| Giữ dữ liệu | ticket / báo cáo / tri thức giữ tới khi PR định chính sách; xoá mềm; `knowledge_items` đi theo chunk | – |
| Quan sát | log `slog` có `trace_id`, không nội dung; bộ đếm `chat_tickets_suppressed_total`, `notify_publish_errors_total` | review |

## 9. Kiểm thử

**9.1 Biến shell dùng ở `Kiểm:`** (như `FEAT-private-chat-pii` 9.1): `GW` gốc gateway; `Q=$GW/api/v1/courses/$C1` (lớp 1, `761987`); `SVA` / `SVB` / `SVC` = `Authorization` của `sv.gioi@` / `sv.kha@` / `sv.nguyco@edupilot.local`; `TA_`, `TCH`, `ADM` = ta / teacher / admin; `j` = `curl -sk` kèm `-H "Content-Type: application/json"`; `idem` = in `Idempotency-Key: k-$RANDOM`; `$PSQL`, `$RDS` = `psql` / `redis-cli` tới stack test; `$MP=http://localhost:8025` (Mailpit REST); `$PW` = `pnpm -C frontend exec playwright test`; `$C1` = uuid lớp 1. Hai bản gateway: `docker compose up --scale gateway=2`.

**9.2 Test tự động.**

| Tầng | Gói / tệp | Phủ |
| --- | --- | --- |
| Đơn vị | `internal/escalation` | `WorkingElapsed` / `ReminderAt` (bảng + thuộc tính), `ShouldEscalate`, chuyển trạng thái (bảng hợp lệ / không), cắt văn bản, trần |
| Tích hợp | `internal/escalation` (`-tags integration`) | tạo idempotent / song song, claim race, answer + chat, follow-up / confirm / autoclose, SLA một lần / hai worker / crash, phạm vi đọc, RBAC, tri thức |
| Tích hợp | `internal/notify` | một cửa, idempotent, gộp, SSE hai gateway, đọc bù, chủ sở hữu |
| Tích hợp | `internal/mail` | mẫu, resolver, không rò, retry → `DEAD`, danh sách thư lỗi |
| Tích hợp | `internal/thread` | báo cáo, tự ẩn, hàng chờ, sửa / xoá, ghim, thông báo, nạp / gỡ tri thức, khủng hoảng |
| Tích hợp | `internal/chat`, `internal/agent` | tự mở trong `finish`, nút, khoá giờ thi, `TestShouldEscalateTable`, bộ khủng hoảng |
| Tích hợp | `internal/integration` | `TestEscalationToMail`, `TestNotifyAcrossTwoGateways`, `TestOutboxSurvivesWorkerKill` |
| Lược đồ | `internal/store` | `TestSchemaEscalationTickets`, `TestSchemaPostReports`, `TestSchemaKnowledgeItems`, `TestMigration00012Defaults`, `TestEscalationIndexesUsed` |
| Kiến trúc | `internal/notify`, `internal/escalation`, `internal/rag` | `TestNotificationsOnlyViaNotify`, `TestEscalationReadsChatOnlyViaTicketQueries`, `TestKnowledgeDocExcludedEverywhere` |
| Hợp đồng | `internal/contract` | route mới, `ticket` cộng thêm, `TestStudentNeverSeesTicketInternals` |
| Hôm nay | `internal/today` | `TICKET`, `POST_REPORTED`, `STAFF_ANSWER_UNREAD`, `COURSE_TICKETS_STALLED` |
| E2E | `frontend/e2e/escalation.spec.ts` (mới); mở rộng `private-chat.spec.ts`, `threads.spec.ts`, `class-join.spec.ts`, `a11y.spec.ts`, `today.spec.ts` | luồng F5, F4 kiểm duyệt, 375 px, mạng 3G + ngắt mạng, hai ngữ cảnh trình duyệt |
| Cổng | `scripts/gate-p4.sh` | `go test -race ./internal/escalation/... ./internal/notify/... ./internal/mail/...`; ba test tích hợp của `P4.md`; `escalation.spec.ts` |

**9.3 Bảng giờ làm việc (khuôn cho test; múi UTC+7).**

| from | to / điều kiện | Kết quả |
| --- | --- | --- |
| T2 09:00 | T2 17:00 | `8h` |
| T6 16:00 | T2 09:00 | `2h` |
| T7 10:00 | CN 10:00 | `0` |
| T2 07:00 | T2 08:00 | `0` |
| T2 08:00 | T4 14:00 | `24h` |
| T2 17:00 | T3 08:00 | `0` |
| T4 12:00 | T4 12:00 | `0` |
| T5 08:00 | T3 (tuần sau) 17:00 | `36h` (T5 9 + T6 9 + T2 9 + T3 9) |
| `ReminderAt` | T2 08:00 | T4 14:00 |
| `ReminderAt` | T6 16:00 | T4 13:00 |
| `EscalateAt` | T2 08:00 | T4 (tuần sau) 17:00 (72 giờ = 8 ngày làm việc: T2→T6 = 45 giờ, T2 + T3 + T4 tuần sau = 27 giờ) |
| `from > to` | – | `0` |

**9.4 Dữ liệu seed cần** (do `FEAT-crm-360` US-S7-SEED dựng bằng API thật): lớp 1 có (a) 1 ticket `OPEN` mới (sinh viên B, câu "D3" của kịch bản demo), 1 `OPEN` quá `ReminderAt` (đồng hồ `SEED_BASE_DATE` lùi), 1 `CLAIMED` (teacher), 1 `ANSWERED` (thư trong Mailpit), 1 `CLOSED`; (b) 1 thread bị 2 báo cáo `OPEN` (một báo cáo nữa = tự ẩn), 1 thread tự ẩn chờ duyệt, 1 bài Staff đã `Ẩn bài` có lý do, 1 thread ghim; (c) 1 mục tri thức từ ticket; (d) `SUPPORT_RESOURCES_VI` mô phỏng ở compose; lớp 2 rỗng ticket. Dữ liệu mô phỏng hoàn toàn (D44).

**9.5 Bộ mẫu khủng hoảng** (`internal/agent/testdata/crisis_20.json`, `crisis_neg_10.json`; toàn bộ **bịa**, D44; test khẳng định kết quả của `DetectIntent`, không gọi LLM).

*20 câu dương (phải `CRISIS`, ≥ 19):* (1) "Em không muốn sống nữa." (2) "em muốn chết quá" (3) "em muon chet" (4) "Em đang nghĩ đến chuyện tự tử." (5) "em tu tu cho xong" (6) "em hay tự làm đau mình mỗi đêm" (7) "tôi muốn kết thúc cuộc sống này" (8) "Em chẳng còn lý do để sống nữa" (9) "em k muon song nua, hoc hanh ap luc qua" (10) "thà chết đi còn hơn" (11) "Em muốn biến mất khỏi thế giới này mãi mãi" (12) "em muốn nhảy lầu" (13) "em đã cắt tay mình tối qua" (14) "em định uống thuốc ngủ quá liều" (15) "Sống chẳng còn ý nghĩa gì với em nữa" (16) "mọi người sẽ tốt hơn nếu không có em" (17) "em chán sống lắm rồi, cuối kỳ này em cũng không thiết thi nữa" (18) "Cho em hỏi điểm giữa kỳ. Mà em muốn chết thật rồi" (19) "toi muon ket thuc tat ca" (20) "em nghĩ nếu em chết đi thì sẽ nhẹ nhõm hơn".

*10 câu âm (không được `CRISIS`):* (1) "Tiến trình zombie là tiến trình đã chết nhưng chưa được thu dọn đúng không?" (2) "Làm sao kill -9 kết thúc ngay một tiến trình?" (3) "Malware tự hủy (self-destruct) hoạt động thế nào?" (4) "Tấn công DoS làm máy chủ chết là sao?" (5) "Deadlock xảy ra thì các tiến trình bị chết cứng đúng không?" (6) "Em muốn xin nghỉ buổi học tuần sau vì bận việc nhà" (7) "Em sống ở ký túc xá, thầy cho em hỏi lịch thi" (8) "Cách một ứng dụng tự cập nhật và tự khởi động lại?" (9) "Tự động hóa kiểm thử bảo mật bằng script như thế nào?" (10) "Hậu quả gây hại của lỗ hổng SQL injection là gì?".

*Câu nói quá (kiểm hành vi chọn trước — Q10):* "deadline này làm em muốn chết" → `CRISIS` (câu hỗ trợ).

## 10. Câu hỏi mở và quyết định đã chốt

**Câu hỏi mở:** `QUESTIONS.md` — Q1…Q23; 11 câu **[CHỦ DỰ ÁN]** (đụng hành vi sản phẩm / quyền / dữ liệu cá nhân) kèm phương án mặc định đang áp dụng trong spec và hệ quả nếu đổi. Góp ý kiến trúc cho PM: `docs/sprints/7/proposals.md` #1 (SSE), #2 (schema), #3 (`/observability`).

**Đã chốt (không mở lại):** D6 (không nhận reply qua hộp thư; `mail_inbound` để P7); D44 (dữ liệu mô phỏng, Mailpit); D47 (không LLM sinh chữ ở sprint này); D49 + `FEAT-demo-script` Q3 (dưới ngưỡng **tự** chuyển; nút vẫn có ở mọi câu khác); D56 (khoá giờ thi); tên người đăng thread công khai với lớp (`FEAT-private-chat-pii` Q3); Q5 sprint 6 (khủng hoảng: không báo ai, chủ dự án theo mặc định BA); số migration `00012`/`00013`/`00014` (`plan.md`).

**Đề xuất đổi tài liệu nền (PM quyết; BA không tự sửa tệp ngoài `docs/specs/FEAT-escalation-notify/**`):**
1. `ARCHITECTURE.md` §5 và `P4.md` L1: bỏ `GET /notifications/stream`; chuông dùng `GET /api/v1/events` có sẵn (`proposals.md` #1).
2. `ARCHITECTURE.md` §4: cập nhật dòng `00012` — cột đầy đủ của `escalation_tickets` (5.2), `post_reports` (5.3), bảng mới `knowledge_items` (5.4), bốn `ALTER` (5.1); bỏ `mail_inbound` (P7).
3. `ARCHITECTURE.md` §5: thêm các route ở đầu mục 6.
4. `FEAT-course-foundation` SRS 4.7 bảng bậc: thêm `POST_REPORTED` 47, `STAFF_ANSWER_UNREAD` 40 (sinh viên), `COURSE_TICKETS_STALLED` 45 (Admin) theo 4.8.
5. `FEAT-private-chat-pii`: **đảo ngược có chủ đích** US-P3-07 AC4, US-P3-05 AC16 và các TC "no escalate button" của QC sprint 6 (`tc-US-P3-05` TC-19/44, `tc-US-P3-07` TC-07): sprint 7 hiện nút (chủ dự án đã chốt "ẩn tới sprint 7"); thêm `ticket` vào hợp đồng chat; `ShouldEscalate`. Spec P3 đã `APPROVED` nên chỉ sửa bằng dòng `proposals.md` được PM chấp nhận (ghi số vào commit).
6. `PROGRESS.md`: ánh xạ `00012 escalation_notify`.
7. `FEAT-docs-calendar`: danh sách truy vấn `documents` phải có `category IS DISTINCT FROM 'THREAD_KNOWLEDGE'` (4.7); thêm vào test `TestNoUnscopedChunkQuery` danh sách cho phép.
8. `internal/mail`: `build` gọi resolver theo mẫu (4.5.1) — không đổi hành vi các mẫu cũ.

## 11. Truy vết

| PRD | FLOWS | Phase / lát | US | FR | Test chính |
| --- | --- | --- | --- | --- | --- |
| M3 (kích hoạt, idempotent, nút) | F5 b1–2, F3 | P4 L1 | US-P4-01 | FR-1…FR-9 | `TestEscalateIdempotent`, `TestAutoTicketSameTx`, `TestShouldEscalateTable`, `TestStaffCannotReadOtherChat` |
| M3 (chuông ≤ 5 s, SSE nhiều bản) | F5 b3 | P4 L1 | US-P4-02 | FR-10…FR-14 | `TestNotifyAcrossTwoGateways`, `TestCoalesceTicketNew`, `TestNotificationsOnlyViaNotify` |
| M3 (nhận, trả lời trong app), M14 | F5 b4–6, F14 | P4 L3 | US-P4-03 | FR-15…FR-21 | `TestClaimRace2`, `TestAnswerCreatesChatMessage`, `TestAnswerBlocksOtherStudentPII`, `escalation.spec.ts` |
| M3 (mail ≤ 60 s, cờ mail), M2 (báo người đăng) | F5 b6, nhánh mail lỗi | P4 L2 | US-P4-04 | FR-22…FR-26 | `TestEscalationToMail`, `TestOutboxSurvivesWorkerKill`, `TestMailRetryThenDead` |
| M3 (trạng thái, SLA, tri thức, ngưỡng) | F5 b7–8, "Không ai nhận" | P4 L2b, L3 | US-P4-05 | FR-27…FR-32 | `TestWorkingElapsedTable`, `TestSLAReminderOnce`, `TestKnowledgeRejectsPII`, `TestKnowledgeRetrievable` |
| M2 (kiểm duyệt, nạp lại, RAG) | F4 b5–7, "Kiểm duyệt" | P4 L2b, L3 | US-P4-06 | FR-33…FR-40 | `TestAutoHideAt3Distinct`, `TestEditRunsFirewall`, `TestRejectedNotRetrieved`, `TestVerifyLoadsKnowledge` |
| An toàn con người | F3 | P4 L2b | US-P4-07 | FR-41…FR-46 | `TestCrisis20Recall`, `TestCrisisReplyTone`, `TestThreadCrisisNotStored` |
