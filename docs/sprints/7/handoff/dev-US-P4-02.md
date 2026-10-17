# DEV handoff — US-P4-02 (thông báo tức thì, gộp, một cửa)
Nhánh: `sprint/7-p4-p5`, commit `e2a7e68` (+ handoff).

## Đã làm
- **`internal/notify`**: `Service.Send` / `SendTx` (+ `Announce` sau commit) / `SendMany` (MỘT `INSERT … SELECT unnest` cho cả danh sách, 1.000 người = 1 câu), `Aggregate` (gộp: một giao dịch, `pg_advisory_xact_lock('notify_agg:…')`, đếm dưới khoá; tăng → nổi lên + chưa đọc + SSE; bằng → không đổi; giảm → đổi chữ, giữ `read_at`, không SSE; 0 → đã đọc). Khung SSE `notification {id,type,title,body,link,course_id,read,created_at,unread_count}` qua `Publisher` có sẵn của `/api/v1/events` (không dựng đường SSE thứ hai — #1). Lỗi phát chỉ log + đếm.
- **Một cửa**: `store/queries/notify.sql` là nơi DUY NHẤT `insert into notifications`. Đã chuyển `course/notify.go`, `exam/{notify,result_events}.go` (ba câu hàng loạt → `SendMany`), `thread` (ghi qua `notify.SendTx`, `Announce` sau commit), `calendar/reminder.go`. Bỏ `InsertNotification`, `ExamNotifyScheduled/Unscheduled/Users`. Worker/gateway truyền `notify.Service` (có `Publisher`); các struct vẫn chạy khi `Notify == nil` nên test cũ dựng chỉ bằng `Pool` không phải sửa.
- **`escalation.Notifier.HandleCreated`**: `escalation.created` → đếm lại ticket OPEN → dòng gộp `TICKET_NEW` "N câu hỏi đang chờ giảng viên" (mã lớp ở `body`, link `/inbox?course=`) cho TA + TEACHER, rồi `today.invalidate`. Payload `{ticket_id, course_id, kind}`.
- **Chuông** (`useRealNotifications`): nghe SSE `notification` (một kết nối dùng chung), chèn / cập nhật tại chỗ, `resync` → đọc bù từ `GET /notifications`; SSE mở → làm mới thưa 120 s, không mở (429, mất kết nối) → 30 s; dòng nhạt "Chưa cập nhật được" khi làm mới lỗi mà còn danh sách cũ.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/notify ./internal/escalation -run 'TestNotificationsOnlyViaNotify|TestSendManyOneStatement|TestCoalesce|TestPersonalNotNotCoalesced|TestNotifyIdempotentRedelivery|TestPublishFailureKeepsRow|TestNotificationNoPrivateText'
go test -count=1 -tags testroutes ./internal/integration -run TestNotifyAcrossTwoGateways -v   # in bell_latency_ms
go test -count=1 -tags testroutes ./internal/course ./internal/exam ./internal/thread ./internal/calendar ./cmd/worker   # test cũ không sửa
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test escalation.spec.ts --project=desktop
```

## AC tự đánh giá
AC1 ✓ (test kiến trúc quét Go + `*.sql`; `SendMany` 1 INSERT) · AC2 ✓ (độ trễ commit → khung SSE đo được ~12 ms ở `TestNotifyAcrossTwoGateways`; phần outbox relay `OUTBOX_POLL_INTERVAL=500ms` cộng thêm khi chạy thật) · AC3 ✓ (hai router, mỗi cái một Redis client; cả hai chiều) · AC4 ⚠ (hạ tầng `Last-Event-ID` / `resync` của PG đã có test ở `httpapi/sse`; chuông gọi lại `GET /notifications` ở `resync`; chưa có test riêng trong `notify` → tôi dựa vào `sse/replay_test.go`) · AC5 ✓ (6 test gộp) · AC6 ✓ (`TestNotificationNoPrivateText`) · AC7 ✓ · AC8 ⚠ (cô lập theo `user_id` là của `sse` + test `course` đã có; không thêm test mới) · AC9 ✓ (e2e `bell third tab`; `TestPublishFailureKeepsRow`) · AC10 ⚠ (chuông là chấm đỏ, chưa có số "9+" — giữ chấm theo `DESIGN.md` hiện hành; ghi để PM quyết) · AC11 ✓ (e2e `bell refresh error`).

## Nợ / cần PM
1. **proposals #12**: thông báo Threads đổi sang outbox đúng chữ SRS sẽ làm đỏ 5 assertion `thread_test.go`; giữ ghi đồng bộ qua `notify.SendTx`.
2. AC4/AC8: không viết lại test SSE đã có; nếu QC muốn test trong `notify` thì ghi TC.
3. AC10: số "9+" trên chuông và tránh nhảy vị trí khi chèn mục mới lúc đang mở — chưa làm (cần chốt chấm hay số).
4. `internal/notify` dùng `Publisher.Publish` tuần tự cho `SendMany` (1.000 lượt Redis ≈ vài trăm ms), không pipeline.

## Vòng 2 (sau QC FAIL vòng 1, `report-US-P4-02.md`)
- **BUG-1 / AC10 chuông**: làm đủ. Chuông hiện số chưa đọc (`data-part="bell-dot"`, tối đa `9+`, `aria-label="Thông báo, N chưa đọc"`, vùng `aria-live=polite` đọc "Có N thông báo mới" một lần). Khung danh sách cuộn được (cao tối đa 420 px) và **chốt chiều cao lúc mở**; thông báo mới chèn lên đầu lúc đang mở → bù `scrollTop` đúng độ lệch của mục đầu cũ nên mục đang đọc đứng yên (đo `getBoundingClientRect`, lệch < 1,5 px). Test: `escalation.spec.ts` "bell count", "bell count small", "bell no jump". File: `shared/shell/NotificationPopover.tsx`, `AppShell.module.css`.
- **BUG-2** chữ spec (AC5 "dòng mới" ↔ SRS 4.4.3 "cùng dòng"): BA / PM sửa chữ, code giữ nguyên.
- **TC-15, 16, 18** nay kích hoạt được bằng route của US-P4-03: `POST /api/v1/courses/{id}/tickets/{tid}/claim` (hoặc `answer` / `dismiss`) bằng TA / giảng viên → worker `escalation.claimed|answered|closed` → dòng gộp `TICKET_NEW` đếm lại theo số `OPEN` hiện tại; về 0 thì `read_at` được đặt (dòng KHÔNG bị xoá), tăng lại thì `read_at = NULL`. `TICKET_ANSWERED` (không gộp) tới sinh viên khi `answer`. Kiểm bằng `select type, title, read_at from notifications where course_id='…' order by created_at`.
- **TC-24 (Redis chết)**: dựng bằng hộp đen không được — Redis dừng thì relay outbox (Redis Streams) cũng dừng nên không có khoảnh khắc "chỉ `Publish` lỗi". Phần đó do `TestPublishFailureKeepsRow` (`internal/notify`) phủ; hành vi QC quan sát (dòng DB cập nhật khi Redis lên lại, không mất) là đúng thiết kế. Xin PM chấp nhận.
- **Test e2e `shell.spec.ts` "forbidden … 0 request /api/v1"** đỏ vì chuông nay mở kết nối SSE `/api/v1/events` ở mọi trang (kể cả màn bị chặn) — xem `proposals.md` #18.

## Vòng 3 (QC vòng 2: TC-29 `ui-antipatterns`)
- `AppShell.module.css:124` (`.unreadDot`) `font-size: 10px` đã sửa ở commit `a606a29` → `var(--ep-text-meta)` (chấm 18 px để vừa số). `bash scripts/ui-antipatterns.sh` → mọi dòng `✓`, exit 0 (đã chạy lại sau commit; QC kiểm trên `a606a29` trở đi — báo cáo vòng 2 đo trên bản trước nó).
- Thêm (gốc của hai ca e2e đỏ trên CI): `useSSE(…, {enabled})`; `AppShell` chỉ nghe `/events` khi đã có phiên thật (`identity`). Trước đó trang chưa đăng nhập (`/dev/data`) cũng mở kết nối `/events`, làm `data-layer.spec` "sse parse + singleton" thấy 2 kết nối. `support/session.ts` (`asDemo`, `asJwt`) mặc định giả `/events` "mở rồi im" để trang có phiên không chạm máy chủ giả dùng chung (đếm sai số kết nối của test lớp dữ liệu chạy song song).
- Visual `shell`/`home` không đổi ảnh (14/14 xanh trong image Playwright v1.63.0-noble).
