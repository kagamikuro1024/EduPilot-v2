# DEV handoff — US-P4-05 (ticket không nằm im: hỏi tiếp, Đã rõ, tự đóng, SLA, tri thức, ngưỡng)
Nhánh: `sprint/7-p4-p5`. Commit: `US-P4-05 (1/3)`, `(2/3)`, `(3/3)` + commit "ui-antipatterns" của US-P4-02 và các commit `fix(test) … (#19)` / `(#18)` / `(#14)`.

## Đã làm
- **Giờ làm việc** (`internal/workhours`, hàm thuần): `Elapsed`, `At`, `SLA{ReminderAt, EscalateAt}`; khung từ `TICKET_WORKDAY_START/END`, `TICKET_WORKDAYS` (mặc định 00:00–24:00, T2–T6, giờ ICT).
- **Vòng đời** (`internal/escalation/lifecycle.go`): `Followup`, `Confirm`, `ConfirmUndo`, `AutoClose`. Route sinh viên `POST /chat/tickets/{id}/followup | confirm | confirm/undo`. `Hỏi tiếp` đặt `waiting_since = now` (không nhảy thẳng giai đoạn 2); tin `ESCALATION_FOLLOWUP` cùng `client_msg_id` → cùng tin.
- **Job `ticket.sla`** (`internal/escalation/sla.go`, khoá leader `ep:lock:ticket.sla`; worker `slaTask`): giai đoạn 1 (24 giờ làm việc → nhắc cả nhóm + thư `ticket_reminder`), giai đoạn 2 (72 giờ → lên đầu Hôm nay), ticket đã nhận mà im → `TICKET_STALE` cho người nhận, lớp có ticket mà không ai hoạt động → báo Admin mỗi ngày (`COURSE_TICKETS_STALLED`, không kèm nội dung). Mỗi giai đoạn ghi **có điều kiện cùng giao dịch với outbox** nên worker chết giữa chừng / hai bản song song không gửi trùng (`TestSLAJobCrashSafe`).
- **Hôm nay**: `TICKET.Overdue` tính ở provider từ `SLA.EscalateAt(waiting_since)` (không phụ thuộc job); `COURSE_TICKETS_STALLED` (Admin, bậc 45); `STAFF_ANSWER_UNREAD` (sinh viên, bậc 40).
- **Ngưỡng**: `PUT /courses/{id}/settings` (TEACHER; `escalation_threshold` 0,50–0,95 bước 0,05, `version`, audit cũ/mới), `GET /courses/{id}` thêm `settings` cho Staff; đọc mỗi lần ở tin kế tiếp (`TestThresholdTakesEffectNextMessage`).
- **Kho tri thức** (`internal/knowledge`, `internal/escalation/knowledge.go`, `queries/knowledge.sql`): `GET …/tickets/{tid}/knowledge-preview` (đã `Redact`, `personal` khi câu hỏi gốc là câu hỏi riêng), `POST …/knowledge {question, answer}` (201 lần đầu / 200 cập nhật; còn PII → `422 PII_DETECTED`; câu hỏi riêng → `422 KNOWLEDGE_PERSONAL`; OPEN/CLAIMED → `409`), `DELETE …/knowledge` (204, idempotent). Một giao dịch: `content_chunks` (`embedding = NULL`) + `knowledge_items` + outbox `knowledge.sync` + `save_as_knowledge` + `audit_log`. Handler `knowledge.sync` (`knowledge.Syncer`): **một** `Embed` (`KNOWLEDGE_EMBED_TIMEOUT`, mặc định 10 s, ngoài `outbox.Chain`), ghi `WHERE id AND md5(text)`, rồi `document.changed` (tăng `ep:rag:ver`). "Chưa nạp" = embedding NULL; `Thử lại` = lưu lại đúng bản đã lưu → ghi lại outbox. Tài liệu ảo `category='THREAD_KNOWLEDGE'` (một dòng mỗi lớp, `documents_knowledge_key`): **mọi** `from/join documents` ngoài `rag.sql` / `knowledge.sql` có `category IS DISTINCT FROM 'THREAD_KNOWLEDGE'` (≈ 27 truy vấn / 7 tệp; `TestKnowledgeDocExcludedEverywhere` quét tự động); giáo viên tạo / sửa category này → `422`. Trích dẫn tri thức có `knowledge:true` → giao diện không liên kết.
- **Tin của giảng viên mang ticket**: `GET /chat/sessions/{id}/messages` gắn `ticket` (4 trường) cho tin giảng viên (qua khối `ticket`) để giao diện hiện `Hỏi tiếp` / `Đã rõ`.
- **Lệnh quản trị**: `gateway admin run-job ticket.sla|ticket.autoclose [--now RFC3339]` (chỉ `APP_ENV` dev | test; xem `dev-QC-answers.md`).
- **Frontend**: `/chat` — dòng trạng thái 4 trạng thái + `Hỏi tiếp` (ô nhập tự lưu nháp, gửi lại cùng mã tin, lỗi giữ chữ) / `Đã rõ` (lạc quan, "Đã đóng · Hoàn tác" 5 s, không hộp thoại) ở tin giảng viên mới nhất của mỗi ticket (`StaffReplyActions`); `/inbox` — `KnowledgeBox` (`Lưu thành tri thức` → xem trước sửa được → `Lưu`; "Đã lưu" + `Gỡ`; "Chưa nạp vào tri thức" + `Thử lại`); `/class/settings` — thanh `Ngưỡng chuyển cho giảng viên` (đầu "Ít chuyển", cuối "Chuyển nhiều", số có dấu phẩy, lưu sau 0,4 s dừng kéo, 409 → nạp lại giá trị thật; TA chỉ xem).

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race -tags testroutes ./internal/escalation -run 'TestFollowup|TestReopened|TestConfirm|TestAutoClose|TestSLA|TestAdminNotified|TestAdminNeverSeesTicketContent|TestWorking|TestReminderAt|TestEscalateAt|TestStudentStatusLabels|TestKnowledge' -v
go test -count=1 -race ./internal/rag ./internal/library ./internal/document ./internal/course ./internal/chat -run 'TestKnowledge|TestDocumentRejectsReservedCategory|TestThreshold|TestSettingsThreshold' -v
go test -count=1 -race -tags testroutes ./internal/contract/... ./internal/today ./cmd/worker ./cmd/gateway
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test escalation.spec.ts class-join.spec.ts -g 'student sees ticket status|student followup|student confirm|knowledge save|threshold slider' --project=desktop
```
Chạy `ticket.sla` một lần trên stack thật (đồng hồ giả; xem `dev-QC-answers.md`):
```bash
docker compose --env-file .env.local -f docker-compose.local.yml -f docker-compose.test.yml -p edupilot \
  exec -T gateway /gateway admin run-job ticket.sla --now 2026-10-20T09:30:00+07:00
```
Xem tri thức đã lưu: `select ki.source_id, c.text, c.embedding is not null as loaded from knowledge_items ki join content_chunks c on c.id = ki.chunk_id;` — `loaded=false` quá 10 s nghĩa là `knowledge.sync` lỗi (`select status,last_error from outbox where topic='knowledge.sync'`).

## AC tự đánh giá
AC1 ✓ · AC2 ✓ · AC3 ✓ · AC4 ✓ (hàm thuần, bảng + thuộc tính) · AC5 ✓ · AC6 ✓ (báo Admin không kèm nội dung ticket) · AC7 ✓ (`student sees ticket status`: 4 trạng thái; không có chữ "ticket" / "SLA" / số tin cậy; `TestStudentStatusLabels`: sinh viên chỉ thấy 4 trường) · AC8 ✓ · AC9 ✓ (đã làm ở US-P4-03) · AC10 ✓ · AC11 ✓ (đo "≤ 10 s" bằng một lần chạy handler trong test, không bằng đồng hồ tường thật; `≤ 5 s` sau khi gỡ đo bằng truy vấn ngay sau commit) · AC12 ✓ · AC13 ✓.

## Nợ / cần PM
1. **Đề xuất #20 chờ ACCEPTED**: `TestNoUnscopedChunkQuery` ghim danh sách đường đọc `content_chunks`; kho tri thức thêm đúng ba tên (`KnowledgeNextOrd`, `KnowledgeChunkForSync`, `KnowledgeForTicket`, đều lọc `course_ids`). Test đang đỏ **đúng ba tên đó**; dev chưa sửa test.
2. Nguồn POST của tri thức (nhãn có liên kết `/threads/{id}`): thuộc US-P4-06 (AC12); `knowledge.Upsert` đã nhận `source_type='POST'`, UI hiện chỉ xử lý `knowledge:true` (không liên kết).
3. `Hỏi tiếp` ở ticket `CLOSED` quá 30 ngày: nút vẫn hiện, bấm nhận `409 TICKET_REOPEN_EXPIRED` và nút biến mất (StudentView giữ bốn trường theo SRS nên giao diện không biết `closed_at`).
4. Ba lệnh seed (`seed-age`, `seed-study-time`, `crm-risk-run`) thuộc US-S7-SEED, chưa làm; ca "quá hạn" thử bằng `run-job ticket.sla --now` (đồng hồ giả) thay vì lùi dữ liệu.
5. `visual.spec` `inbox-1440/390` sinh lại trong `mcr.microsoft.com/playwright:v1.63.0-noble` (#14b); 14/14 xanh trong image, không đổi ngưỡng.
6. Test #14(a): hai ca `mock course mapping` đổi màn mẫu `/inbox` → `/students` (còn mô phỏng tới US-P5-04).
