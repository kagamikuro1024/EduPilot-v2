# DEV handoff — US-P8-03 (lịch gộp, feed ICS, nhắc 24 giờ, tool lịch) — **giao đủ (backend + UI)**
Nhánh `sprint/6-p3-p8`. Commit `US-P8-03: …`.

## Đã làm
**Backend**
- Migration `00011_calendar.sql`: `calendar_events` (`UNIQUE (course_id,id)`, `ends_at > starts_at`, index `(course_id, starts_at, id)`), `reminder_log` (`UNIQUE (user_id, source_type, source_id, starts_at, kind)`), một `CHECK` `users_ics_token_hash_chk` (64 chữ hex). Không `ALTER` bảng cũ nào khác. Test `TestSchemaCalendar`, `TestReminderLogUnique`, `TestICSTokenHashCheck` (`internal/store`); `db/migrations_test.go` ghim 10 → 11; `internal/user/list_test.go` seed `ics_token` đổi thành 64 chữ `a` vì CHECK từ chối `'tok-ics'`.
- `internal/calendar` (`calendar.go`, `ics.go`, `reminder.go`):
  - `List`: UNION lúc đọc `class_sessions` + `exams` (không `DRAFT`) + `calendar_events` bằng một truy vấn `CalList`; `personal_state` chỉ của chính sinh viên (`GRADING`/`GRADED` → `SUBMITTED`); trạng thái bài thi là `exam.EffectiveStatus` theo giờ; khoảng ≤ 62 ngày (422 `RANGE_TOO_LARGE`); con trỏ `(starts_at,id)`; `ETag` = băm thân (`httpx.WriteJSONETag`).
  - Sự kiện của Staff: `Create/Update/Delete` — một giao dịch gồm kiểm lớp `ACTIVE` (409 `COURSE_ARCHIVED`), ghi, `audit_log`, outbox `calendar.changed`; `version` (409 kèm bản hiện hành); 422 `EVENT_TIME_INVALID` (±2 năm, `ends_at > starts_at`).
  - Token ICS: 32 byte `crypto/rand` → base64url (43); chỉ lưu `hex(sha256)`; URL chỉ có ở phản hồi POST (`Cache-Control: no-store`); POST lại = xoay; DELETE idempotent; GET chỉ `{exists}`.
  - Feed: tra theo băm bằng chỉ mục duy nhất; mọi lỗi (thiếu / sai / xoay / thu hồi / `DISABLED` / sai độ dài) cùng một 404; `ep:rl:ics:{ip}:{phút}` 60 / phút → 429 + `Retry-After`; `ETag`/304, `Cache-Control: private, max-age=300`; CRLF, gập 75 octet không cắt giữa UTF-8, thoát `\ ; , \n`; `DTSTAMP` = `updated_at` (ổn định nên ETag không đổi khi dữ liệu không đổi); tiền tố `{mã lớp} · ` khi ≥ 2 lớp; 2.000 sự kiện; không `description`, không tên người. Log truy cập chỉ ghi `r.URL.Path` nên không bao giờ có giá trị `token` (có test).
  - Tool `get_exam_schedule` / `get_upcoming_events` nối thật (`calendarService(d)` thay `nil` ở `chatwire.go`): đọc DB trực tiếp theo `trusted_context`, `days` kẹp 1–30, giờ `Asia/Ho_Chi_Minh` ("Thứ Hai, 21/09 · 14:00"). `agent.Result.Message` mới: câu NoData riêng ("Chưa có lịch thi nào được công bố." / "Bạn không có sự kiện nào trong {n} ngày tới.").
  - `reminder.tick` (`calendar.Reminder`, task `reminder.tick` ở worker, leader `ep:reminder:tick:leader`, `REMINDER_TICK=5m`, `REMINDER_LEAD=24h`): nguồn `(now, now+24h]` của lớp `ACTIVE`; người nhận = sinh viên `ACTIVE` (tài khoản `ACTIVE`), lô 200 mỗi giao dịch; `reminder_log … ON CONFLICT DO NOTHING`, chỉ dòng chèn được mới sinh chuông `REMINDER` (`dedupe_key remind:{src}:{id}:{epoch}`) và thư (`mail.Enqueue`, mẫu `reminder`, chỉ `event_title` + `at`) trong cùng giao dịch; thư chỉ khi `remind_deadline_by_mail` + email đã xác minh.
- Route `calendarhttp` (#16–#23): đọc `MemberRole`, ghi `StaffRole` (ADMIN không qua, đúng AC15), `/me/calendar/ics-token` theo JWT, feed công khai đăng ký ngoài `auth.Middleware`.
- `PUT /me/settings` nhận `reminders {exam, class_session, other}` (gộp từng khoá vào `user_settings.preferences`, khoá lạ → 422 `reminders.<khoá>`); `GET` trả `reminders` (mặc định `true/false/true`). **Đổi một pin của P2:** `TestSettingsDefaultsLazyCreate` (`internal/user/profile_test.go`) thêm khoá `reminders` vào thân mong đợi — thay đổi do SRS 4.9 yêu cầu mở rộng lược đồ, không nới lỏng assertion nào.
- `today.Topics()` + `Invalidator` nhận `calendar.changed` (người học cũng bị ảnh hưởng, như topic bài thi); worker đăng ký `calendar.TopicChanged` (`TestEveryEmittedTopicHasHandler` xanh).
- `openapi.yaml` +8 thao tác (tag `calendar`) → ghim 162 → 170; mã lỗi `RANGE_TOO_LARGE`, `EVENT_TIME_INVALID`; `scenarios_calendar_test.go` gọi mọi status đã khai báo (kể cả 304 / 429 của feed); contract bỏ kiểm thân cho `text/calendar` (không có bộ giải mã ICS; định dạng do `TestICS*` kiểm).
- `.env.example`: `REMINDER_TICK`, `REMINDER_LEAD`.

**Giao diện**
- `/calendar`: `CalendarScreen` → `RealCalendar` (phiên thật có lớp) hoặc `DemoCalendar` (bản cũ đổi tên). Tuần ≥ 720 px / Danh sách < 720 px / Tháng, kiểu đã chọn nhớ ở `localStorage`; một Panel; đỏ chỉ cho bài thi trong 48 giờ; `personal_state` bằng chữ cho sinh viên; Staff có `Thêm sự kiện` (form tại chỗ, không hộp thoại), ⋯ `Sửa` / `Xoá` (xoá: dòng biến ngay, `Hoàn tác` 5 giây, hết giờ hoặc rời trang mới gọi API); `Thêm vào lịch` mở vùng nhỏ: `Tạo liên kết` → URL hiện một lần + `Sao chép`; `Đặt lại liên kết` có xác nhận nêu hậu quả; có token mà mất thì chỉ có "Mất liên kết thì đặt lại" (đúng một chỗ).
- `/settings`: khối "Nhắc trước 24 giờ" (Bài thi / Buổi học / Sự kiện khác), chỉ hiện cho sinh viên có lớp thật; công tắc lạc quan.
- Chat: khối `exam_schedule` / `upcoming_events` hiện từng dòng "Thứ Hai, 21/09 · 14:00 Tên · Địa điểm" thay vì JSON thô.
- e2e `calendar.spec.ts` (`week default`, `list on mobile`, `month`, `staff add event` ×2, `ics panel`, `states`, `375`), `private-chat.spec.ts -g 'calendar tool block'`, `support/cal-fixtures.ts` (giờ cố định `NOW` để tuần không phụ thuộc ngày chạy), `asDemo` giả mặc định `GET …/calendar`.

## Lệnh QC
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/calendar ./internal/store ./db ./internal/user ./internal/today ./cmd/worker -v
go test -count=1 -race -tags testroutes ./internal/contract ./internal/agent
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 npx playwright test e2e/calendar.spec.ts e2e/private-chat.spec.ts --workers=2
bash ../scripts/ui-antipatterns.sh   # rc=0
```
Tên test theo AC: AC2 `TestCalendarUnionSources|NoDuplication|RangeLimit|Cursor|ExcludesDraftExam`; AC3 `TestEventCRUD|Validation|VersionConflict|Audited|ArchivedCourse`; AC4 `TestCalendarETag304|ETagChangesOnWrite|ETagChangesOnAttemptStart|ETagChangesWithEffectiveStatus`; AC5 `TestPersonalStateOwnOnly|TestCalendarNoExamContentLeak`; AC6 `TestChangeReflectedInCalendarAndTool|TestCalendarChangedInvalidatesToday`; AC7 `TestICSToken*`; AC8 `TestICS*`; AC9 `TestICSBadTokenUniform404|RateLimitPerIP|DisabledUser404`; AC10 `Test{ExamSchedule,UpcomingEvents,UpcomingEventsDaysClamped,CalendarToolsScopedToCourse,CalendarToolsNoData,CalendarToolsVietnamTime}`; AC11–13 `TestReminder*`; AC15 `TestCalendarMatrix|TestICSTokenOnlySelf`.

## AC tự đánh giá
AC1 ✓ · AC2 ✓ · AC3 ✓ (Hoàn tác 5 giây ở giao diện có e2e) · AC4 ✓ · AC5 ✓ · AC6 ✓ (phần "tay" hỏi chat sau khi đổi giờ chưa chạy — cần stack thật) · AC7 ✓ · AC8 ✓ (chưa đăng ký URL vào Google Calendar thật) · AC9 ✓ · AC10 ✓ · AC11 ✓ (20 tick song song) · AC12 ✓ · AC13 ✓ (1.000 sinh viên một lần tick; thư thật qua Mailpit chưa chạy) · AC14 ✓ (375 px: kiểm không tràn ngang + vùng chạm của nút `Thêm vào lịch`) · AC15 ✓ · AC16, AC17 chưa — thuộc P3-08 (`gate-p8.sh`, seed).

## Nợ / cần hỏi
1. Chưa chạy trên stack compose thật (chủ đang dùng stack s55): tick nhắc → thư Mailpit, đăng ký URL vào lịch thật, hỏi chat sau đổi giờ là việc của QC / `gate-p8.sh`.
2. `useCalendar` tải hết các trang (`limit=100`) của khoảng đang xem — tháng rất dày (> 100 sự kiện) sẽ nhiều lượt gọi; chưa dựng "Xem thêm".
3. Công tắc nhắc ở `/settings` dùng `version` của thời điểm tải: hai lần bật / tắt liên tiếp quá nhanh có thể gặp 409 một lần rồi tự lấy lại bản thật (không mất lựa chọn đã hiện).
4. Xoá sự kiện: nếu đóng tab giữa 5 giây `Hoàn tác`, lời gọi xoá có thể không kịp gửi (dòng vẫn còn ở lần mở sau). Chấp nhận được vì hoàn toàn đảo ngược.
5. `REMINDER_TICK` / `REMINDER_LEAD` chỉ đọc ở worker; gateway không dùng.
6. Mẫu thư `reminder` thêm vào `internal/mail/templates.go` (bảng mẫu hiện có) — không đụng SRS FEAT-account-security 6.5; nếu BA muốn liệt kê mẫu này ở đó thì ghi vào `proposals.md`.
