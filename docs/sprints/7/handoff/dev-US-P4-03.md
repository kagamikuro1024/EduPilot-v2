# DEV handoff — US-P4-03 (hộp thư hỗ trợ: danh sách, nhận, trả lời, đóng)
Nhánh: `sprint/7-p4-p5`, commit đầu `9e05ffb`; cập nhật test ghim theo **proposals #11** (số thao tác 170 → 179, chat+threads 20 → 21, `want` của `TestSpec_SecurityDeclared` +9 mục, version migration 11 → 12) nằm ở chính commit đó và commit của US-P4-01.

## Đã làm
- **`internal/escalation/staff.go`**: `ListStaff` (một lớp hoặc `MyCourseIDs`), `GetDetail`, `Claim` / `Release` / `Dismiss` / `Answer`. Mỗi thao tác một giao dịch: đổi trạng thái có điều kiện (`WHERE status … AND claimed_by …`, hai người cùng `Nhận` → đúng một người thắng), `audit_log`, `outbox` (`escalation.claimed|released|answered|closed`). `Answer` thêm tin `STAFF_ANSWER` (`blocks=[{kind:"ticket",data:{ticket_id,answered_by:{full_name,role}}}]`) vào phiên của ticket; phiên đã xoá mềm → phiên "Tin nhắn từ giảng viên" (tạo-hoặc-lấy). `checkAnswerPII` chặn cứng (`422 PII_IN_ANSWER`, `details:[{start,end,kind}]`, không lặp giá trị): tên / MSSV trong roster của sinh viên KHÁC, email của sinh viên khác, CCCD; tên và email của chính người hỏi, email giảng viên không bị chặn.
- **Route** (`internal/httpapi/escalationhttp/staff.go`, guard `StaffRole`, ADMIN / sinh viên 403): `GET /courses/{id}/tickets`, `GET /me/tickets`, `GET …/tickets/{tid}`, `POST …/claim|release|answer|dismiss` (`answer` bắt buộc `Idempotency-Key`), `POST /notifications/read {link}` (#15a). Có trong `openapi.yaml` (+ schema); mọi status đã khai báo có kịch bản trong `internal/contract/scenarios_chat_test.go`.
- **Worker**: `escalation.claimed|released|closed` → `Notifier.HandleRecount` (đếm lại dòng gộp `TICKET_NEW`); `escalation.answered` → `HandleAnswered` (thông báo `TICKET_ANSWERED` cho sinh viên, không gộp, khoá khử trùng theo id tin, link `/chat?session=<id phiên nhận tin>`; tiêu đề / thân không có nội dung) + đếm lại; cả bốn qua `today.Invalidator`.
- **Chat**: lịch sử gửi LLM bỏ intent `STAFF_ANSWER` / `ESCALATION_FOLLOWUP`; `feedback` và `retry` trên các tin đó → `422 MESSAGE_NOT_RATEABLE` (khai báo trong openapi cho `retry`).
- **Hôm nay**: `TICKET` (bậc 10, TA + TEACHER, "N câu hỏi đang chờ · M câu bạn đã nhận · lâu nhất {Age}", `/inbox?course=`) gộp vào `TodayStaffPending`; `STAFF_ANSWER_UNREAD` (bậc 40, sinh viên) gộp vào `TodayViewer` (hai truy vấn không tăng ngân sách ≤ 5, `TestTodayQueryBudget` giữ nguyên).
- **Frontend**: `features/inbox/{RealInbox,InboxScreen,inboxApi}`; `InboxView.tsx` (mock) xoá, `features/inbox` không còn `@/mock`; `noteTicketAnswered` xoá. Chia đôi 380 px | chi tiết, ≤ 1100 px danh sách → chi tiết qua `?ticket=`; bộ lọc `Đang chờ / Đã nhận / Đã trả lời / Tất cả`; "Tất cả lớp của tôi" → `/me/tickets` (dòng ghi mã lớp); nháp `useAutosaveDraft('ticket:'+id)`; gửi có khoá Idempotency, lỗi giữ nháp + `Gửi lại` (cùng khoá); `PII_IN_ANSWER` báo tại ô soạn; phím `J` `K` `R` `⌘↵`. `/chat`: tin giảng viên hiện "Giảng viên · {tên}" / "Trợ giảng · {tên}", không nút Hữu ích; mở phiên có tin giảng viên → `POST /notifications/read`.

## File đổi
`backend-go/internal/escalation/{staff,notifier,service}.go` (+ test `staff_test.go`, `notify_test.go`), `internal/httpapi/escalationhttp/staff.go`, `internal/httpapi/{routes,chatwire}.go`, `internal/chat/{sessions,send}.go`, `internal/notify/notify.go` (`MarkReadByLink`), `internal/today/{today,providers,service}.go`, `internal/store/queries/{escalation,chat,notify,today}.sql` (+ sinh lại), `cmd/worker/registry.go`, `api/openapi.yaml`, `internal/contract/*`, `db/migrations_test.go`; `frontend/src/features/{inbox,chat}/**`, `e2e/escalation.spec.ts`, `e2e/support/session.ts` (giả mặc định `/tickets` cho `asDemo`).

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/escalation ./internal/today ./internal/notify
go test -count=1 -race ./internal/chat -run 'TestChatHistoryExcludesStaffMessages|TestStaffMessageNotRateableNotRetryable'
go test -count=1 -tags testroutes ./internal/contract/... ./db ./cmd/worker
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate
E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test escalation.spec.ts --project=desktop --workers=2
```

## AC tự đánh giá
AC1 ✓ · AC2 ✓ (`TestMyTicketsAcrossCourses`; ADMIN 403 ở contract) · AC3 ✓ (chi tiết chỉ đúng cặp hỏi–đáp + tin của ticket; không audit khi mở) · AC4 ✓ · AC5 ✓ · AC6 ✓ · AC7 ✓ · AC8 ✓ · AC9 ✓ · AC10 ✓ (375 px, vùng chạm do `panels.spec` đo) · AC11 một phần: `J` `K` `R` `⌘↵` có, **tooltip phím trên nút `Gửi trả lời` chưa** (Composer chưa có prop) · AC12 ✓ · AC13 ✓ · AC14 ✓ (contract: mọi vai × route; `TestAllCourseRoutesGuarded` xanh) · AC15 ✓ (`grep` = 0; dữ liệu mock ticket còn lại do `DemoChat`, `AnalyticsScreen`, `AppShell` badge, Hôm nay mock dùng — ngoài `features/inbox`) · AC16 ✓.

## Nợ / cần PM
1. **proposals #13** (đã làm; test của chính dev) và **#14** (chờ ACCEPTED): hai ca `class-join.spec.ts` "mock course mapping…" và hai ảnh chuẩn `visual.spec.ts` `inbox-*`. Dev chưa sửa.
2. Tên test theo US khác tên trong repo ở một số chỗ (`TestAnswerBlocksOtherStudentPII` ↔ `TestAnswerPIIOfOtherStudentBlocked`…): xem mục "Test" ở cuối nếu QC chạy `-run` theo tên.
3. Huy hiệu số trên mục `Hộp thư hỗ trợ` ở thanh điều hướng vẫn từ dữ liệu mock (`AppShell`); chưa có API đếm — để US-P4-06.
4. `POST /notifications/read` chưa xoá cache Hôm nay của sinh viên ngay (chờ TTL ≤ 60 s).
