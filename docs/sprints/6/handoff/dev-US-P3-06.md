# DEV handoff — US-P3-06 (Threads hai kênh + tường lửa PII) — **giao đủ (backend + UI)**
Nhánh `sprint/6-p3-p8`. Commit `US-P3-06: …` (backend, contract, UI, e2e).

## Đã làm
**Backend**
- `internal/thread`: `Firewall.CheckPost` (một hàm cho thread, bình luận, và P4 sau này; `Redacted="[đã ẩn]"`), `Service` (List, Get, Similar, Precheck, Create, Comment, Decide, `RecordSwitched`), `Answer` (việc AI), `RegisterKind` (kind `thread.answer`).
  - Thứ tự `Create`: khoá giờ thi (409 `EXAM_IN_PROGRESS` + `until`, ghi `exam.RecordChatBlocked`; lỗi đọc khoá → 503) → tường lửa → ghi thread + việc `thread.answer` + outbox `thread.created` **một giao dịch**.
  - PII: 422 `PII_DETECTED` kèm `reasons`, `redacted_text`, `personal_question`; `redact:true` máy chủ **tự chạy lại** tường lửa, chỉ lưu bản `[đã ẩn]`, bản máy khách gửi không được tin. Câu hỏi riêng tư không có lối ẩn (422 cả khi `redact:true`). `pii_events` BLOCKED / REDACTED / SWITCHED chỉ có loại + số đếm.
  - AI trả lời: handler `job.enqueue` chỉ XADD vào `ep:ingest` (`ingest.Queue.Extra` nhận kind `thread.answer`) → consumer worker nhúng một lần, truy xuất `audience=ALL`, **một** lời gọi `Chat` làn `NEAR_REALTIME`, một bài `AI` `PENDING` kèm nguồn. Không đủ tin cậy → `SKIPPED` (`NO_CONTEXT`/`LOW_SCORE`); lỗi tạm thời giao lại, lần thứ 3 tự `SKIPPED LLM_UNAVAILABLE`.
  - Phân quyền theo vai: sinh viên không thấy `confidence`, `ai_body`, bài `REJECTED`; Staff thấy. Tên người đăng công khai với cả lớp (Q3). Quyết định Staff: `verify` / `correct` (kèm `version`) / `reject` idempotent, `POST_STATE_CONFLICT` / `VERSION_CONFLICT`, audit, thông báo `THREAD_ANSWERED|REPLY|VERIFIED` có `dedupe_key`.
- `chat`: `POST /chat/sessions/from-draft` (phiên PRIVATE không có tin nhắn, trả lại đúng từng byte, không lưu nháp), hook `OnSwitched` → `thread.RecordSwitched`.
- `rag`: sửa `document_ids` → `text[]` (xem handoff P3-05 vòng sửa 1); `testutil.RuntimePool` dùng cho rig của `internal/thread`.
- 10 route mới (9 Threads + from-draft), `openapi.yaml` + tag `threads`, mã lỗi `PII_DETECTED`, `POST_STATE_CONFLICT`; contract op 141 → 151, `scenarios_thread_test.go` (cả nhánh 404 / 409 / 403).

**Giao diện (`frontend/src/features/threads`)**
- `ThreadsScreen` / `ThreadDetail`: phiên thật có lớp → `RealThreads` / `RealThreadDetail`; phiên mô phỏng → `DemoThreads*` (bản cũ).
- `usePostGate`: precheck sau khi ngừng gõ 800 ms (một dòng "Phát hiện MSSV, Email", không mở hộp thoại, xoá chữ thì biến mất); đăng với `Idempotency-Key` mới sau mọi 4xx / khi thân đổi, giữ khoá khi mất mạng / 5xx; 422 → `PIIChannelDialog` đúng hai nút (`Chuyển sang chat riêng`, `Ẩn thông tin rồi đăng`); câu hỏi riêng tư khoá nút ẩn (lý do ở `title`); chữ không bao giờ bị xoá khi lỗi.
- `Chuyển sang chat riêng`: `from-draft` → `seedDraft` (đặt nháp chat) → `/chat?session=…`; `RealChat` mở phiên theo query.
- Soạn tại chỗ (không modal), nháp `thread:{course}:…` tự lưu 2 s, khoá giờ thi (ô soạn khoá + giờ mở lại), lọc tiêu đề / chủ đề / tuần / trạng thái (dưới 1100 px thu sau nút `Bộ lọc`), phân trang con trỏ, 375 px.
- Chi tiết: bài AI có nhãn + nguồn mở tại chỗ + thread tương tự; Staff có `Xác nhận` / `Chỉnh sửa` / menu `Loại` cạnh bản nháp và `Độ tin cậy 0,72`.
- e2e: `threads.spec.ts` (13 ca), `support/thread-fixtures.ts`; `asDemo` giả `GET …/threads*` mặc định; ca `/threads` của `ui-foundation.spec.ts` và `panels.spec.ts` viết lại theo Threads thật; route mẫu `/threads/t-cbc` → `/threads/${TID}`.

## File đổi
`backend-go/internal/{thread,chat,rag,agent/cite.go,ingest/queue.go,httpapi/{threadhttp,chathttp,chatwire.go,router.go,routes.go,apierr},testutil,contract,store/queries/forum.sql}` + sinh, `cmd/worker/{registry,tasks,thread_job_test}.go`, `api/openapi.yaml`; `frontend/src/features/{threads,chat/RealChat.tsx}`, `shared/data/{ApiError.ts,index.ts,useAutosaveDraft.ts}`, `shared/i18n/vi.ts`, `frontend/e2e/{threads,ui-foundation,panels}.spec.ts`, `e2e/support/{thread-fixtures,session,routes}.ts`.

## Lệnh QC
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/thread ./internal/chat -v              # tên test theo đúng cột "Kiểm" của US (gồm TestThreadsMatrix, TestCreateIdempotent20Parallel qua router thật)
go test -count=1 -race ./cmd/worker -run TestRosterInvalidateNotBlockedByLongJob
go test -count=1 -race -tags testroutes ./internal/contract
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 npx playwright test e2e/threads.spec.ts e2e/privacy.spec.ts --workers=2
bash ../scripts/ui-antipatterns.sh   # rc=0
```
Lưu ý: `pnpm build:gate` **phải** có `E2E_API_PORT=3412` (không đặt thì bản dựng trỏ cổng 3312 và `data-layer`, `settings-llm` đỏ giả).

## AC tự đánh giá
AC1 ✓ · AC2 ✓ · AC3 ✓ (`precheck debounce`) · AC4 ✓ (`dialog two paths`) · AC5 ✓ (`switch keeps text`, `TestFromDraft*`) · AC6 ✓ · AC7 ✓ · AC8 ✓ (`personal question`) · AC9 ✓ · AC10 ✓ · AC11 ✓ · AC12 ✓ · AC13 ✓ · AC14 ✓ · AC15 ✓ · AC16 ✓ · AC17 ✓ · AC18 ◐ (xem nợ 2) · AC19 ✓ (`TestAllWritePathsUseFirewall`) · AC20 ✓.

## Nợ / cần hỏi
1. **Chưa chạy trên stack compose thật**: e2e dùng gateway giả theo hợp đồng thật; đo thật (k6, 3G) để QC / US-P3-08.
2. **AC18 lọc "chủ đề"**: thẻ (`tags`) nhập tay (chữ thường, gạch nối); chưa có danh sách chủ đề cố định ⇒ ô nhập chứ không phải chip. Dưới 1100 px lọc thu sau nút `Bộ lọc` (không phải Popover). Cần BA chốt nếu muốn chip theo danh mục.
3. **`TestRosterInvalidateNotBlockedByLongJob`** (AC3 của P3-02) nay có, ở `cmd/worker` — đóng nợ 1 của handoff P3-02.
4. `visual.spec.ts` ảnh mốc `threads` lệch (màn đã đổi, cộng lệch máy từ trước); cần dựng lại ảnh khi chủ chốt máy chuẩn.
5. Các test khác ngoài `rag`, `integration`, `thread` vẫn dùng `pgxpool.New` mặc định (xem nợ ở handoff P3-05).

## Vòng sửa 1 (PM, theo `qc/report-US-P3-06.md`)
- **BUG-1 — `thread.created` không có handler (dead-letter sau 4 lần).** Gốc: đăng thread ghi hai tin outbox (`job.enqueue` cho việc AI, `thread.created` cho P4 / P10) nhưng worker chỉ đăng ký `job.enqueue`. Sửa: `reg.Register(thread.TopicCreated, thread.OnCreated)` ở `cmd/worker/registry.go` (handler ghi nhận, trả ngay, idempotent; **không** chạy AI ở consumer outbox tuần tự — TLR-9 giữ nguyên: AI vẫn đi `job.enqueue` → XADD `ep:ingest` → consumer riêng). Lệch nhỏ so với chữ SRS 4.9.5 ("handler `thread.created` chỉ xếp hàng"): việc xếp hàng do `job.enqueue` làm, `thread.created` không xếp thêm để khỏi có hai tin XADD cho một thread (hai worker có thể cùng gọi LLM trước khi `ai_state` đổi). Test tích hợp mới `TestThreadPostToAIAnswer` (`cmd/worker`): `thread.Service.Create` → giao MỌI tin outbox của lần đăng qua bảng handler thật `newRegistry` (fail nếu có topic không có handler) → consumer `ep:ingest` → đúng một bài AI `PENDING`, `ai_state=ANSWERED`.
- **BUG-2 — `TestRosterInvalidateNotBlockedByLongJob` panic.** Gốc (khác chỗ nghi ban đầu): test dựng `ingest.Processor` không có `Pool` và dùng nhóm tiêu thụ `ingest` trên Redis **dùng chung** với các gói test khác — khi gói `internal/ingest` chạy trước, tin `document.ingest` tồn đọng bị consumer của test nhận và panic. Sửa: `Processor.Pool` có thật; `ingest.Queue.Group` (mới, mặc định `GroupName`) để test dùng nhóm riêng tạo từ `$` (không đọc tin tồn); `blockingLLM` đóng `started` bằng `sync.Once`. Đã chạy lại sau `internal/ingest` với `-p 1` trên cùng Redis: xanh; AC3 của P3-02 được chứng minh bằng test chạy được (`LLM treo` ở consumer `ep:ingest`, roster xoá ≤ 5 s).
- **BUG-3 — hộp thoại PII ghi "mssv".** `countReasons` viết `MSSV` / `CCCD` hoa, các loại khác thường ("1 email, 1 MSSV"); e2e `dialog two paths` khẳng định "Chúng tôi tìm thấy: 1 MSSV.".
