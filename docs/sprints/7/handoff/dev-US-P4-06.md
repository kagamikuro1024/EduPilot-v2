# DEV handoff — US-P4-06 (kiểm duyệt Threads: báo cáo, tự ẩn, hàng chờ, sửa / xoá, ghim, tri thức, thông báo gộp)
Nhánh: `sprint/7-p4-p5`. Commit: `US-P4-06: …` (một commit; cũng chứa việc BUG-S7-01 BUG-2 ở `ci.yml` và đề xuất #26 — xem cuối). Số ghim: `#11`, `#24`, `#25`.

## Đã làm
- **Store / sqlc**: `queries/moderation.sql` (mới: khoá + chèn + đếm + huỷ báo cáo, hàng chờ, ẩn / bỏ ẩn / sửa / xoá mềm, ghim, đếm AI chờ, đếm báo cáo mở); `forum.sql` viết lại — mọi truy vấn đọc `forum_threads` / `forum_posts` đều có `hidden_at is null` hoặc `-- hidden-ok: <lý do>`; `today.sql` thêm `reported`, `reported_oldest` vào `TodayStaffPending` (vẫn ≤ 5 truy vấn).
- **`internal/thread`**: `moderation.go` (`Report`, `Unreport`, `Hide`, `Unhide`, `DismissReports`, `Queue`, `Pin`, `syncKnowledge`), `edit.go` (`EditThread`, `EditPost`, `DeleteThread`, `DeletePost`), `notifier.go` (`Notifier`: gộp `AI_PENDING`, `POST_HIDDEN`, `POST_UNHIDDEN`, đếm lại báo cáo). `Create` / `Get` / `List` / `Similar` biết người xem (ẩn thì chỉ Staff + chủ bài thấy; `pinned[]` tách khỏi `items`, chỉ trang đầu).
- **Báo cáo / tự ẩn**: một người một lần (`ALREADY_REPORTED`), không báo bài mình (`CANNOT_REPORT_OWN`), hạn `REPORT_RATE_PER_HOUR` (20) đếm từ `post_reports`, ba người KHÁC NHAU còn `OPEN` → tự ẩn (`hidden_by NULL`, "Nhiều báo cáo, chờ duyệt"), khoá hàng nên hai báo cáo song song chỉ ẩn một lần. Admin 403, ngoài lớp 404.
- **Staff**: `Ẩn bài` (lý do ≤ 200) / `Bỏ ẩn` / `Giữ bài` (`dismiss-reports`, chỉ bỏ ẩn bài TỰ ẩn) / `Ghim` (≤ `THREAD_PIN_MAX` = 3). Mọi lệnh cần `version` hiện tại → `409 VERSION_CONFLICT` kèm `current_version`. Ẩn / bỏ ẩn ghi `audit_log` không chứa nội dung bài.
- **Chủ bài**: sửa (qua tường lửa PII, `422 PII_DETECTED` + `redact`), khoá `EDIT_LOCKED_VERIFIED` khi bài AI đã `VERIFIED` / `CORRECTED`; xoá mềm; Staff (kể cả GV) 403 trên bài người khác. Sửa khi bài AI còn `PENDING` → bài AI cũ `REJECTED`, `ai_state='PENDING'`, xếp lại `thread.answer`, **không** báo chủ bài.
- **Tri thức (AC11–12)**: `syncKnowledge` suy trạng thái từ bài AI duy nhất của thread, cùng giao dịch với `Decide` / ẩn / bỏ ẩn / xoá; nhúng vẫn qua `knowledge.sync` (outbox) nên lỗi nhúng không làm hỏng xác nhận.
- **Outbox (AC8–9, SRS 4.11)**: `thread.ai_answered`, `edited`, `reported`, `hidden`, `unhidden`, `deleted` + `thread.post_decided` đều đi `outbox.Chain(handler, inv.Handle)` đăng ký ở `cmd/worker/registry.go`. Bốn đường thông báo cũ giữ đồng bộ — **đề xuất #24**.
- **Today**: `POST_REPORTED`, bậc 47, Staff của lớp, `/threads?view=moderation&course=…`; không thêm vào `today.Topics()` (test ghim 24).
- **OpenAPI**: 17 thao tác (tag `threads`), schema `ReportInput`, `ReportCreated`, `HideInput`, `ModResult`, `EditThreadInput`, `EditPostInput`, `ModerationReports`, `ModerationQueueItem/Page`, `ThreadHidden`, mở rộng `ThreadRow`, `ThreadPage.pinned`, `ThreadPost`, `ThreadDetail`. Số ghim (#11): `contract_test.go` 194 → 211, `chat_threads_contract_test.go` 21 → 38, bảng `want` bảo mật; miễn 429 cho hai POST báo cáo (`exempt.go`, cùng lý do `ReportRateLimit` có test riêng).
- **Cấu hình**: `REPORT_RATE_PER_HOUR`, `REPORT_AUTOHIDE_THRESHOLD`, `THREAD_PIN_MAX`.
- **Frontend**: `ThreadModeration.tsx` (`PostMenu` "Tuỳ chọn bài": Báo cáo / Sửa / Xoá / Ẩn bài / Bỏ ẩn / Giữ bài / Ghim; biểu mẫu báo cáo + ẩn nằm tại chỗ, KHÔNG dùng `Dialog` vì `ui-antipatterns` chỉ cho Dialog việc cần bảo vệ; xoá dùng `ConfirmIrreversible` "Câu hỏi và N bình luận sẽ bị xoá"; `EditForm` dùng lại `usePostGate` với `method: "PUT"` + nhãn `Ẩn thông tin rồi lưu`; `HiddenNotice` một dòng). `RealThreads`: `pinned[]` trên `items`, tab `Câu hỏi | Cần xem` cho Staff (`DataTable`, `?view=moderation`). `ALREADY_REPORTED` hiện "Bạn đã báo cáo bài này" (không là lỗi). Mã lỗi / chữ Việt đã có sẵn trong `ApiError.ts` / `vi.ts`.

## Test
- Go mới (tên khớp US): `internal/thread/moderation_test.go` (≈ 60 ca: báo cáo, tự ẩn, race, hàng chờ, RBAC, version, thông báo, sửa / xoá, ghim, tri thức, similar), `internal/today/post_reported_test.go`, `internal/store/forum_hidden_test.go` (`TestForumQueriesFilterHidden`: quét `queries/*.sql`, truy vấn đọc forum_* mà thiếu `hidden_at is null` / `-- hidden-ok:` thì đỏ). Kịch bản contract `moderationScenarios`.
- e2e mới: `frontend/e2e/threads-moderation.spec.ts` (báo cáo + hoàn tác + báo lại, sửa PII hai lối + xoá, ẩn / bỏ ẩn / ghim, bài mình bị ẩn, ghim trên items + hàng chờ + axe, sinh viên không có tab, 375 px). `threads.spec.ts` cũ xanh nguyên (menu mới đặt nhãn "Tuỳ chọn bài" để không trùng "Thêm hành động" của Loại).
- Không sửa test cũ nào (ngoài số version `db/migrations_test.go`, #11).

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race -tags testroutes ./internal/thread ./internal/today ./internal/store ./internal/contract/... ./internal/httpapi/...
sqlc generate && sqlc diff
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate
E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test threads.spec.ts threads-moderation.spec.ts --project=desktop --project=mobile
bash ../scripts/ui-antipatterns.sh                                   # rc=0
```

## Quyết định cần nêu
1. Ẩn tay lên bài đang tự ẩn → chuyển thành ẩn tay (đổi người ẩn + lý do); ẩn tay lên bài đã ẩn tay → `409 POST_STATE_CONFLICT`.
2. Báo cáo đến sau khi bài đã tự ẩn → 404 (bài không còn hiện với người đó, đúng SRS 4.10.4); test race chấp nhận 201 hoặc 404, và đúng MỘT dòng outbox `thread.hidden`.
3. Hạn tần suất đếm bằng `post_reports.created_at` trong DB, không dùng Redis (chịu được Redis chết, cùng tinh thần #16).

## Cần PM / nợ
1. ~~#25~~ **ACCEPTED, xong ở commit vòng 2**: migration `00014_forum_ai_regen` (DROP + CREATE `forum_posts_one_ai_key WHERE kind='AI' AND verification_state <> 'REJECTED'`, `down` trả lại, báo rõ `down` chỉ được khi chưa thread nào sinh lại). Ba truy vấn từng giả định "một bài AI mỗi thread" đã sửa: `ThreadList.ai_verification` (ưu tiên bài chưa REJECTED, mới nhất), `ThreadSimilar` (chỉ loại thread có bài AI MỚI NHẤT bị loại / ẩn), `ThreadKnowledgeSource` (bài hiệu lực mới nhất). Số ghim (#11): `db/migrations_test.go` 13 → 14. Test `TestEditRegeneratesAIAnswer`. Migration của P5-03 là `00015_observation` (theo PM).
2. ~~#24~~ **ACCEPTED**: bốn đường cũ giữ đồng bộ; không sửa tám test sprint 6.
3. `TestSchemaAttendance` đã sửa theo chỉ đạo PM (commit BUG-S7-01 vòng 2).
4. **BUG-S7-01 BUG-2 (thời gian CI)**: tách `Go test (rest)` thành `rest-1` / `rest-2` (gói lẻ / chẵn của `go list`) trong `.github/workflows/ci.yml` — chỉ đổi cấu hình, đúng chỉ dẫn #19. Nếu nhánh có "required check" tên `Go test (rest)` thì PM đổi tên check.
5. `docs/ARCHITECTURE.md` bảng env: thêm `REPORT_RATE_PER_HOUR`, `REPORT_AUTOHIDE_THRESHOLD`, `THREAD_PIN_MAX` (BA).
6. Chạy `go test ./...` cả cây trên máy 8 GB làm `TestChatThreadsContract` chậm (131 s so với 62 s chạy riêng) và đỏ rải rác ở kịch bản khác nhau mỗi lần (`POST /exams` 500, `PATCH …/chunks` 503); chạy riêng `./internal/contract/...` xanh. Một lần ổ đĩa Colima đầy (`No space left on device`) — đã dọn. QC đã ghi nhận hiện tượng tải tương tự ở report P5-02. CI (tách `rest-1` / `rest-2`) là nguồn đúng.
