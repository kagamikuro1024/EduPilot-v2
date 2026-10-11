# DEV handoff — US-P3-01
Nhánh: `sprint/6-p3-p8`. Commit: `US-P3-01: …` (xem `git log`).

## Đã làm (theo thứ tự lát dọc)
1. Migration `00007_chat_threads.sql`: 8 enum, hàm `forum_tags_valid(text[])` (IMMUTABLE, cho CHECK thẻ ≤ 5 phần tử × 1–30 ký tự vì CHECK không dùng được subquery), `chat_sessions`, `chat_messages`, `forum_threads`, `forum_posts`, chỉ mục SRS 5.6, 4 trigger `set_updated_at`. Không `ALTER` (FK phức hợp khai trong `CREATE TABLE`, UNIQUE khai dạng constraint nội tuyến).
2. Migration `00008_privacy.sql`: `pii_kind`, `pii_action`, `pii_events` (không cột văn bản), trigger chỉ-thêm dùng lại `audit_log_block_mutation()` (UPDATE/DELETE/TRUNCATE → `42501 … append-only`).
3. sqlc: `queries/chat.sql` (phiên, tin, danh sách con trỏ, `SetChatPartial`, `FinishChatMessage` ghi có điều kiện, reaper), `forum.sql`, `privacy.sql`.
4. Test `internal/store/chat_schema_test.go` (13 test).
5. `docs/PROGRESS.md` "Ánh xạ migration": `00007`, `00008`, `00009 calendar`, `00010 chunk_search`.

Chủ ý: `pii_events (course_id, session_id)` có FK phức hợp tới `chat_sessions` (NULL được → luồng Threads không có phiên). Cột `confidence` chat/post có thêm CHECK 0–1; `forum_posts.confidence` chỉ cho bài `AI`. Cả hai nằm ngoài SRS 5.3/5.4 nhưng chỉ siết dữ liệu sai.

## File đổi
`backend-go/db/migrations/00007_chat_threads.sql`, `00008_privacy.sql`; `backend-go/internal/store/queries/{chat,forum,privacy}.sql`; `backend-go/internal/store/{chat,forum,privacy}.sql.go`, `models.go` (sqlc sinh); `backend-go/internal/store/chat_schema_test.go`; `docs/PROGRESS.md`.

## Lệnh QC chạy để kiểm
```bash
cd backend-go
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
sqlc generate && sqlc diff
go test -count=1 ./internal/store -run 'TestSchemaChat|TestSchemaForum|TestCrossCourseFK|TestChatOwnerFK|TestReplyToFK|TestDocumentDeleteSetsNull|TestUpdatedAtTriggers|TestPIIEvents|TestEnumRejectsUnknown|TestMigrateIdempotentUp|TestChatMigrationDownUp|TestChatForumIndexesUsed' -v
grep -c 'ALTER TABLE' db/migrations/00007_chat_threads.sql db/migrations/00008_privacy.sql   # 0 và 0
```
Kiểm tay append-only: `$PSQL -c "update pii_events set count=2"` → `ERROR: audit_log is append-only` (SQLSTATE 42501; thông điệp lấy từ hàm chung với `audit_log`).

## Test đã chạy và kết quả
- 13 test mới: xanh. `go vet`, `golangci-lint run`: sạch.
- `go test -race -tags testroutes ./...` (EP_SKIP_TIMING=1): xanh, trừ `internal/today` đỏ 2 test (`TestExamResultTodayItems`, `TestQuestionReviewProvider`) khi cả cây chạy song song; chạy riêng `./internal/today/` → `ok`. Là nhiễu tải của máy, không đụng schema mới.

## AC tự đánh giá
AC1 ✓ (up / down / up qua `TestChatMigrationDownUp`, `TestMigrateIdempotentUp`; 0 `ALTER`) · AC2 ✓ · AC3 ✓ · AC4 ✓ · AC5 ✓ · AC6 ✓ (`TestChatForumIndexesUsed` chạy cả khi không có tag `integration`) · AC7 ✓ (`schema` ghim cột sáu bảng trong `TestSchemaChat`) · AC8 ✓ · AC9 chưa (xem Nợ).

## Nợ / chưa làm / cần hỏi
- **AC9 `TestChatContentOnlyInMessages`** ở `internal/chat` cần dịch vụ chat nên làm cùng US-P3-05; QC kiểm AC9 ở story đó.
- `goose status` bằng CLI chưa chạy tay (không cài `goose`); `db.Migrate` (cùng thư viện) được test phủ.
