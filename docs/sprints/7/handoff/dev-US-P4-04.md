# DEV handoff — US-P4-04 (thư: đúng lúc, không lộ người khác, lỗi không hỏng ứng dụng)
Nhánh: `sprint/7-p4-p5` (commit `US-CI-02 vòng 2 … US-P4-04` và commit sau nó).

## Đã làm
- **`internal/mail`**: `Mail.ID`; `Sender` đặt `Message-ID` tất định `<mail_outbox.id@domain MAIL_FROM>` (`mail.MessageID`); Mailpit chạy `MP_IGNORE_DUPLICATE_IDS=true` (`docker-compose.local.yml`, container của `testutil` — **container cũ phải xoá: `docker rm -f edupilot-test-mailpit`**). `Resolver` + `Handler.Resolvers` (nạp biến hiển thị LÚC GỬI), `mail.Permanent(class)`, `mail.Clip`. Năm mẫu mới trong `specs()`: `escalation_answered`, `outreach_message`, `ticket_reminder`, `ticket_new`, `thread_reply` (tiêu đề không chứa tên người; thiếu biến → `template_missing_var` → DEAD ngay).
- **`internal/escalation/mail.go`**: `MailResolver` cho bốn mẫu (ticket / tin trả lời phải thuộc đúng sinh viên + lớp, không thì `payload_invalid`; ticket không còn → `ticket_missing`; câu hỏi cắt 300, trả lời cắt 2000 + "… Xem đầy đủ trong EduPilot"; liên kết `APP_PUBLIC_URL/chat?session=…&m=…`, `/inbox?course=…` dựng lúc gửi). `Answer` xếp `escalation_answered` **trong cùng giao dịch** (chỉ tới email đã xác minh của chính chủ ticket, `notify_answer_by_mail` mặc định bật; `dedupe_key = escalation_answered:<answer_message_id>`). `Notifier.HandleCreated` xếp `ticket_new` cho TA / TEACHER ĐÃ bật `notify_ticket_by_mail` (mặc định tắt), gộp bằng cửa sổ trượt `MailGate` (Lua atomic trên Redis, đồng hồ truyền vào nên test dùng `clock.Fake`; Redis lỗi → vẫn gửi; `MAIL_TICKET_NEW_COALESCE_SECONDS`, mặc định 900); `EnqueueReminder` (cho job SLA của US-P4-05): TA + TEACHER ACTIVE đúng lớp, không ADMIN, `dedupe_key … :<yyyymmddHH>`.
- **`internal/thread/mail.go`**: `MailResolver` cho `thread_reply`; `Comment` (người khác bình luận) và `Decide` (VERIFIED / CORRECTED) xếp thư cho người đăng thread trong giao dịch của mình.
- **Worker** (`cmd/worker/registry.go`): `mail.Handler` đăng ký resolver của escalation + thread; `Notifier` có `Gate`.
- **`GET /admin/observability/failed-mail`** (`internal/httpapi/mailhttp`, ADMIN): `mail_outbox WHERE status='DEAD'`, cursor `(updated_at, id)` bằng `mail_outbox_dead_idx`; trường `{id, template, to_masked, attempts, last_error, updated_at}`; không thân thư, không payload. Có trong `openapi.yaml` + kịch bản hợp đồng; số thao tác 179 → 180 và `want` bảo mật + 1 (proposals #11, cùng commit).
- **Frontend**: `/settings` có công tắc "Mail báo ticket mới" (TA / giảng viên, tắt mặc định) hoặc "Mail khi giảng viên trả lời" (sinh viên, bật mặc định) — lạc quan, không toast (`MailSection`); `/observability` có mục "Thư chưa gửi được" (đầu trang, chỉ Admin; rỗng → "Không có thư lỗi.") (`FailedMail`).

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
docker rm -f edupilot-test-mailpit 2>/dev/null   # để container dựng lại với MP_IGNORE_DUPLICATE_IDS
go test -count=1 -race ./internal/escalation -run 'TestEscalationToMail|TestOutboxSurvivesWorkerKill|TestMailSendIdempotentRedelivery|TestMailNoCrossStudentLeak|TestMailPayloadHasNoContent|TestMailRetryThenDead|TestMailPermanentFailureNoRetry|TestMailFailureDoesNotAffectAnswer|TestMailRespectsSettings|TestMailRecipientIsTicketOwner|TestReminderMailRecipients|TestTicketNewMailCoalesced|TestTicketNewMailSlidingWindow|TestMailLogsHaveNoAddress|TestEscalationMailMissingTicket|TestEscalationMailBadPayload|TestFailedMail' -v
go test -count=1 ./internal/mail -run 'TestTemplate|TestSenderDeterministicMessageID|TestClip|TestNoHTTPReadsMailOutbox' -v
go test -count=1 ./internal/thread -run TestThreadReplyMail -v
go test -count=1 ./internal/user -run 'TestSettingsMailFlagsByRole|TestSettingsVersionConflict|TestSettingsStudentTicketFlagStoredNoEffect' -v
go test -count=1 -tags testroutes ./internal/contract/... ./cmd/worker
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test escalation.spec.ts -g 'mail settings|failed mail' --project=desktop
```
Trên stack thật: `docker compose … exec -T gateway /gateway …` không cần; Mailpit `http://localhost:8025`; thư DEAD xem ở `/observability` (Admin) hoặc `select template, status, attempts, last_error from mail_outbox where status='DEAD'`. Ép thư lỗi: dừng `mailpit` (`docker compose … stop mailpit`) rồi trả lời một ticket — thử 4 lần (1 s, 5 s, 30 s) rồi `DEAD`.

## AC tự đánh giá
AC1 ✓ (`TestEscalationToMail`: ticket → trả lời → thư trong Mailpit; đo một lượt, không đo p95 trên 10 lần) · AC2 ✓ · AC3 ✓ (giả lập giết bằng `panic` ngay sau khi Mailpit nhận `DATA` rồi giao lại: chỉ một thư mang `Message-ID` của dòng; **không** có ca `SIGKILL` tiến trình thật và không có điểm giết "trước SMTP" riêng — điểm đó hiển nhiên vì chưa gửi gì) · AC4 ✓ · AC5 ✓ · AC6 ✓ · AC7 ✓ · AC8 một phần: không đường HTTP đọc `mail_outbox` ngoài danh sách đã che, log không có địa chỉ (test); "SMTP luôn là Mailpit" là cấu hình compose, chưa có test · AC9 ✓ (EXPLAIN trên 100.000 dòng / 500 DEAD: `mail_outbox_dead_idx`, không `Seq Scan`) · AC10 ✓.

## Nợ / cần PM
1. Test của AC nằm ở `internal/escalation` (có rig + Redis + Mailpit) thay vì `./internal/integration` / `./internal/mail` như dòng `Kiểm:`; chạy bằng lệnh trên (không cần tag `integration`).
2. `outreach_message` có mẫu + resolver nhưng chưa có chỗ xếp thư (thuộc US-P5 "nhắn tin chủ động").
3. `ticket_reminder`: hàm `EnqueueReminder` đã có và có test; chỗ gọi (job `ticket.sla`) làm ở US-P4-05.
4. `docker-compose.local.yml` đổi (thêm env cho Mailpit): stack đang chạy phải `up -d` lại.
