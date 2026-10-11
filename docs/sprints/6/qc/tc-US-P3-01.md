# QC test case — US-P3-01 (migration `00007_chat_threads`, `00008_privacy`, store sqlc)
Nguồn: `docs/specs/FEAT-private-chat-pii/US.md` US-P3-01 AC1–AC9 (v1.2 APPROVED) + `SRS.md` 3.3 (nhánh lỗi), 5.1–5.8 (enum, DDL, chỉ mục, Redis, outbox), 9.2; `TL-REVIEW.md` TLR-4, TLR-11 và mục "Quyết định PM" (TLR-11 (1) = `ON DELETE SET NULL`); `docs/sprints/6/plan.md` story 1; `docs/phases/P3.md` L0 + cổng nghiệm thu. Viết trước khi có code (pha 1), hộp đen: QC tự dựng Postgres riêng, đo bằng `goose`, `psql`, `pg_dump`, `EXPLAIN`; test Go của dev chỉ chạy **thêm**, không thay cho phép đo của QC.

**Tiền điều kiện chung.** Worktree `/Users/kuro/Documents/TA_Agent_qc5` (nhánh `sprint/6-p3-p8`). Stack riêng của QC: Postgres 18 `pgvector/pgvector:pg18` **trống** (chưa chạy migration nào), Redis 8. Biến: `DB` = `postgres://…` tới DB trống đó; `PSQL='psql "$DB" -v ON_ERROR_STOP=0 -At'`; `GOOSE='goose -dir db/migrations postgres "$DB"'`; `C1` = lớp `761987`, `C2` = lớp `761988` (sau khi gieo seed); `SVA`/`SVB` = `sv.gioi@edupilot.local` (Vũ Hoàng Giang, 20229001) / `sv.kha@edupilot.local` (Bùi Thanh Khải, 20229002) theo `SRS.md` 9.1. Công cụ ghi trước mỗi bước: **S** = shell, **D** = SQL qua `psql`, **G** = `go test`, **P** = Python (`docs/sprints/6/qc/scripts/p601-schema.py`). Sau mỗi lượt dựng stack: `docker volume prune -f` (`docs/team/CONTEXT.md` mục 4).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P3-01-01 | AC1 | DB trống | **S** `cd backend-go && $GOOSE up && $GOOSE status` | `rc=0`; `status` có **8** dòng `Applied`, hai dòng mới là `00007_chat_threads.sql` và `00008_privacy.sql`; không dòng `Pending` |
| TC-P3-01-02 | AC1 | sau 01 | **S** `pg_dump "$DB" --schema-only > /tmp/a.sql`; `$GOOSE down && $GOOSE down`; **D** `select count(*) from pg_tables where tablename in ('chat_sessions','chat_messages','forum_threads','forum_posts','pii_events')`; `select count(*) from pg_type where typname in ('chat_channel','chat_role','chat_stream_status','chat_feedback','thread_state','thread_ai_state','post_kind','post_verification','pii_kind','pii_action')`; **S** `$GOOSE up`; `pg_dump "$DB" --schema-only > /tmp/b.sql`; `diff /tmp/a.sql /tmp/b.sql` | mỗi `down` `rc=0`; sau hai `down`: `0` bảng **và** `0` enum (không để kiểu mồ côi); `up` lại `rc=0`; `diff` **rỗng** |
| TC-P3-01-03 | AC1 | – | **S** `grep -ciE '^[[:space:]]*alter table' db/migrations/00007_chat_threads.sql db/migrations/00008_privacy.sql`; với mỗi dòng khớp (nếu có) đọc ngữ cảnh ±5 dòng | `0` ở mỗi tệp; ngoại lệ duy nhất được chấp nhận: `ADD CONSTRAINT … DEFERRABLE` đứng ngay sau `CREATE TABLE` **của chính tệp đó** — mọi `ALTER` chạm bảng của `00001`–`00006` là FAIL |
| TC-P3-01-04 | AC1 | – | **S** `git diff origin/main -- backend-go/db/migrations/0000[1-6]*` ; `for f in db/migrations/0000[1-6]*; do git show origin/main:backend-go/$f \| shasum -a 256; shasum -a 256 $f; done` | `diff` rỗng; từng cặp hash khớp (migration đã merge không bị sửa — nguyên tắc bất biến 6) |
| TC-P3-01-05 | AC1 | – | **S** `grep -nE '00007 +chat_threads\|00008 +privacy\|00009 +calendar' docs/PROGRESS.md` và xác nhận ba dòng nằm trong mục "Ánh xạ migration" | đủ **3** dòng trong đúng mục đó |
| TC-P3-01-06 | AC2 | sau 01 | **P** `python3 docs/sprints/6/qc/scripts/p601-schema.py --tables chat_sessions,chat_messages` (bảng kỳ vọng chép tay từ SRS 5.2–5.3: tên cột, kiểu, NULL, mặc định) | `0 lệch`; không cột thừa, không cột thiếu; `chat_messages.confidence` là `numeric(4,3)`, `attempt` là `smallint`, `citations`/`blocks` là `jsonb` mặc định `'[]'` |
| TC-P3-01-07 | AC2 | – | **S** `awk '/CREATE TABLE .*chat_messages/,/^\);/' db/migrations/00007_chat_threads.sql \| grep -cE 'partial_content\|stream_status'` | ≥ `2`: cả hai cột nằm **trong** `CREATE TABLE` (không phải `ALTER` thêm sau — L3b chỉ việc ghi vào) |
| TC-P3-01-08 | AC2 | – | **D** `select enum_range(null::chat_stream_status)`; `select enum_range(null::chat_role)`; `select enum_range(null::chat_channel)` | `{STREAMING,DONE,FAILED,CANCELLED}`; `{USER,ASSISTANT}`; `{PRIVATE,PUBLIC}` — đúng giá trị và đúng thứ tự SRS 5.1 |
| TC-P3-01-09 | AC2 | phiên `S1` của SVA ở C1 | **D** chèn hai `chat_messages` cùng `(session_id, client_msg_id)` (UUID giống nhau); rồi chèn hai hàng có `client_msg_id IS NULL` | lần hai → `23505`; hai hàng NULL chèn được (UNIQUE là partial `WHERE client_msg_id IS NOT NULL`) |
| TC-P3-01-10 | AC2 | – | **D** `insert … role='USER', stream_status='STREAMING'` | `23514` |
| TC-P3-01-11 | AC2 | – | **D** `insert … role='USER', confidence=0.500` | `23514` |
| TC-P3-01-12 | AC2 | – | **D** `insert … stream_status='DONE', partial_content='abc'` | `23514` |
| TC-P3-01-13 | AC2 | – | **D** `insert … stream_status='STREAMING', completed_at=now()` | `23514` |
| TC-P3-01-14 | AC2 | – | **D** `insert … content='x'` rồi `select updated_at` ; `update chat_messages set content='y' where id=…` ; `select updated_at` lần hai | `updated_at` lần hai **lớn hơn** lần đầu (trigger `set_updated_at`; reaper 4.7.5 dựa vào cột này — TLR-4) |
| TC-P3-01-15 | AC2 | – | **D** `insert … content=repeat('a',20001)` ; `insert … attempt=0` ; `insert … intent='abc'` (chữ thường) | `23514` cả ba (`content ≤ 20000`, `attempt ≥ 1`, `intent ~ '^[A-Z_]{3,40}$'`) |
| TC-P3-01-16 | AC2 | – | **G** `cd backend-go && go test ./internal/store -run 'TestSchemaChat' -v` | `ok` |
| TC-P3-01-17 | AC3 | sau 01 | **P** `p601-schema.py --tables forum_threads,forum_posts`; **D** `select format_type(atttypid, atttypmod) from pg_attribute where attrelid='forum_threads'::regclass and attname='embedding'` | khớp SRS 5.4 từng cột; `embedding` = `vector(1536)`; có sẵn `hidden_at`, `hidden_reason`, `verification_state`, `similar_of`, `pinned_at`, `ai_skip_reason`, `version` |
| TC-P3-01-18 | AC3 | thread `T1` ở C1 | **D** chèn hai `forum_posts` `kind='AI'` cùng `thread_id=T1` | lần hai → `23505` (`UNIQUE (thread_id) WHERE kind='AI'`) |
| TC-P3-01-19 | AC3 | – | **D** `insert … kind='AI', author_id=<uuid SVA>` ; `insert … kind='HUMAN', author_id=null` | `23514` cả hai (`(kind='AI') = (author_id IS NULL)`) |
| TC-P3-01-20 | AC3 | – | **D** `insert … hidden_at=now(), hidden_reason=null` ; `insert … hidden_at=null, hidden_reason='spam'` | `23514` cả hai (cặp cùng có hoặc cùng không) |
| TC-P3-01-21 | AC3 | – | **D** `insert … kind='HUMAN', verification_state='PENDING'` ; `insert … kind='AI', verification_state='NONE'` | `23514` cả hai (`(kind='HUMAN') = (verification_state='NONE')`) |
| TC-P3-01-22 | AC3 | – | **D** `insert forum_threads … tags=array_fill('x',array[6])` ; `tags='{"<31 ký tự>"}'` ; `week_no=0` ; `week_no=21` ; `title=''` ; `body=repeat('a',8001)` | `23514` mỗi ca |
| TC-P3-01-23 | AC3 | – | **G** `go test ./internal/store -run 'TestSchemaForum' -v` | `ok` |
| TC-P3-01-24 | AC4 | C1, C2 có thread/phiên riêng | **D** chèn `forum_posts` có `course_id=C2` trỏ `thread_id` của C1 | `23503` (FK phức hợp `(course_id, thread_id)`) |
| TC-P3-01-25 | AC4 | – | **D** chèn `chat_messages` có `course_id=C2` trỏ `session_id` của phiên thuộc C1 | `23503` |
| TC-P3-01-26 | AC4 | phiên `S1` của SVA | **D** chèn `chat_messages` có `user_id` = id của SVB, `session_id=S1` | `23503` (FK `(session_id, user_id)` → `chat_sessions (id, user_id)`; ghi sai ở service là lộ chat riêng — TLR-11 (2)) |
| TC-P3-01-27 | AC4 | tin `M2` ở C2 | **D** chèn `chat_messages` ở C1 có `reply_to=M2` | `23503` (FK `(course_id, reply_to)`) |
| TC-P3-01-28 | AC4 | tài liệu `D1` của C1, phiên `S2` có `document_id=D1` | **D** `delete from documents where id=D1`; rồi `select document_id from chat_sessions where id=S2` | `delete` **thành công** (không `23503`); `document_id` → `NULL` (`ON DELETE SET NULL`, quyết định PM TLR-11 (1)) |
| TC-P3-01-29 | AC4 | – | **D** `insert into chat_sessions (…) values (…)` không nêu `last_message_at` → đọc lại; rồi `insert … last_message_at=null` | lần đầu: cột có giá trị `now()`; lần hai: `23502` (NOT NULL — khoá con trỏ không được NULL, TLR-11 (4)) |
| TC-P3-01-30 | AC4 | – | **D** `update forum_threads set title=…` và `update forum_posts set body=…`; so `updated_at` trước/sau | `updated_at` tăng ở cả hai bảng (trigger `set_updated_at` — TLR-4) |
| TC-P3-01-31 | AC4 | – | **G** `go test ./internal/store -run 'TestCrossCourseFK\|TestChatOwnerFK\|TestReplyToFK\|TestDocumentDeleteSetsNull\|TestUpdatedAtTriggers' -v` | `ok` |
| TC-P3-01-32 | AC5 | sau 01 | **D** `select column_name, data_type from information_schema.columns where table_name='pii_events' order by ordinal_position` | đúng **9** cột `id, course_id, session_id, user_id, channel, pii_type, count, action, created_at`; **không** có `updated_at`; `session_id` nullable |
| TC-P3-01-33 | AC5 | – | **D** `select count(*) from information_schema.columns where table_name='pii_events' and data_type in ('text','character varying','character')` | `0` — không cột văn bản tự do nào (bất biến "không nội dung") |
| TC-P3-01-34 | AC5 | một dòng `pii_events` | **D** `update pii_events set count=2`; `delete from pii_events` | cả hai → `42501`, thông điệp chứa `append-only` (cùng hàm trigger với `audit_log`) |
| TC-P3-01-35 | AC5 | – | **D** `insert … count=0`; `insert … count=-1` | `23514` cả hai (`count >= 1`) |
| TC-P3-01-36 | AC5 | – | **D** `select enum_range(null::pii_kind)`; `select enum_range(null::pii_action)` | `{MSSV,EMAIL,PHONE,CCCD,NAME,PERSONAL_QUESTION,OTHER_PERSON}`; `{BLOCKED,REDACTED,SWITCHED,MASKED}` |
| TC-P3-01-37 | AC5 | – | **G** `go test ./internal/store -run 'TestPIIEventsAppendOnly\|TestPIIEventsNoFreeText' -v` | `ok` |
| TC-P3-01-38 | AC6 | – | **D** `select indexname, indexdef from pg_indexes where tablename in ('chat_sessions','chat_messages','forum_threads','forum_posts','pii_events')` so với danh sách 11 chỉ mục ở SRS 5.6 | đủ **11** chỉ mục đúng tên, đúng cột, đúng thứ tự `DESC`, đúng mệnh đề `WHERE` của chỉ mục từng phần (`chat_messages_streaming_idx` có `WHERE stream_status='STREAMING'`; `forum_posts_pending_idx` có đủ 4 điều kiện) |
| TC-P3-01-39 | AC6 | 10.000 dòng gieo ngẫu nhiên cho mỗi bảng (`generate_series`, rải đều 2 lớp, 30 người) + `ANALYZE` | **D** `EXPLAIN (ANALYZE, FORMAT JSON)` bốn truy vấn trang đầu: danh sách phiên theo `(course_id,user_id,last_message_at DESC,id DESC)`; tin theo `(session_id, created_at DESC, id DESC)`; thread theo `(course_id,last_activity_at DESC,id DESC)`; bài theo `(thread_id, created_at, id)` — `limit 30` | `0` nút `Seq Scan` trên bảng > 1.000 dòng ở cả bốn kế hoạch; mỗi kế hoạch dùng đúng chỉ mục tên ở SRS 5.6; ghi thời gian thực thi vào report |
| TC-P3-01-40 | AC6 | – | **G** `go test -tags integration ./internal/store -run 'TestChatForumIndexesUsed' -v` | `ok` |
| TC-P3-01-41 | AC7 | – | **S** `cd backend-go && sqlc generate && sqlc diff && git status --porcelain internal/store` | `sqlc diff` `rc=0`; `git status` **rỗng** (mã sinh đã commit đúng bản) |
| TC-P3-01-42 | AC7 | – | **S** `grep -c . internal/store/queries/chat.sql internal/store/queries/forum.sql internal/store/queries/privacy.sql`; **G** `go test ./internal/store -run 'TestSchema' -v`; đọc `schema_test.go` | ba tệp truy vấn tồn tại và khác rỗng; `ok`; `schema_test.go` ghim danh sách cột của **sáu** bảng mới (`chat_sessions`, `chat_messages`, `forum_threads`, `forum_posts`, `pii_events` + bảng thứ sáu dev khai — xem Q-QC-P3-01-2) |
| TC-P3-01-43 | AC8 | đã `up` xong | **S** `$GOOSE up; echo "rc=$?"` lần hai | Ghi đúng mã trả về và thông điệp. Chấm theo Q-QC-P3-01-1: nếu BA xác nhận "không có gì để chạy, `rc=0`" thì `rc=0` + thông điệp đọc được là PASS; DB **không** đổi (`pg_dump --schema-only` giống trước) |
| TC-P3-01-44 | AC8 | bản chép `00008` sửa lỗi cố ý (QC chép cả thư mục ra `/tmp/mig-broken`, thêm câu SQL sai ở **giữa** tệp) | **S** `goose -dir /tmp/mig-broken postgres "$DB2" up; echo "rc=$?"`; **D** kiểm bảng của tệp lỗi | `rc≠0`, thông điệp đọc được (có tên tệp + lỗi SQL); DB **không nửa vời**: không bảng/enum nào của tệp lỗi tồn tại (mỗi tệp trong một transaction); `goose status` không đánh dấu tệp đó `Applied` |
| TC-P3-01-45 | AC8 | – | **D** `update chat_messages set stream_status='X' where id=…` ; `insert … stream_status='x'` (chữ thường) | `22P02` cả hai; không panic ở tầng Go khi đọc lại hàng đó qua `go test -run TestEnumRejectsUnknown` |
| TC-P3-01-46 | AC8 | – | **G** `go test -tags integration ./internal/store -run 'TestMigrateIdempotentUp\|TestEnumRejectsUnknown' -v` | `ok` |
| TC-P3-01-47 | AC9 | gateway + worker + seed chạy được (phụ thuộc US-P3-05; nếu chưa có, QC chèn thẳng tin nhắn bằng `psql` và chạy một lượt AI Threads) | **S** gửi một tin chat chứa canary `CANARY-7Q2X`; **D** quét **mọi** cột `text`/`varchar`/`jsonb` của **mọi** bảng trừ `chat_messages` bằng truy vấn sinh từ `information_schema` tìm chuỗi `CANARY-7Q2X` | **0** lần xuất hiện ngoài `chat_messages` (`audit_log`, `outbox.payload`, `jobs`, `llm_audit`, `pii_events` đều sạch) |
| TC-P3-01-48 | AC9 | – | **G** `go test -tags integration ./internal/chat -run 'TestChatContentOnlyInMessages' -v` | `ok` |
| TC-P3-01-49 | AC9 (phân quyền DB) | – | **D** `select rolsuper, rolbypassrls from pg_roles where rolname=current_user` khi nối bằng đúng chuỗi kết nối của gateway; rồi thử `update pii_events set count=9` bằng vai đó | vai ứng dụng **không** `rolsuper`; `update` vẫn `42501` (trigger không bị vai ứng dụng vượt qua) |
| TC-P3-01-50 | AC9 (phân quyền đọc chéo) | phiên `S1` của SVA (phụ thuộc US-P3-05 cho đường đọc) | **S** `curl -s -o /dev/null -w '%{http_code}' -H "$SVB" $GW/api/v1/chat/sessions/$S1/messages`; `-H "$TA_"`; `-H "$TCH"`; `-H "$ADM"` | `404` với SVB (không lộ tồn tại); `403`/`404` với TA / TEACHER / ADMIN — **không** `200`, không dòng nội dung nào trả về (`SRS.md` 2: không có route đọc chat riêng của người khác) |

## Nhánh lỗi (SRS 3.3 + AC8 — phần thuộc tầng dữ liệu)
| Tình huống | TC |
| --- | --- |
| `goose up` lần hai khi đã `up` | 43 |
| `down` giữa chừng lỗi, DB nửa vời | 02, 44 |
| Giá trị enum lạ qua SQL thô (`22P02`), panic ở tầng Go | 45, 46 |
| Dữ liệu sai lọt DB (CHECK: `USER` + `STREAMING`, `DONE` + `partial_content`, `STREAMING` + `completed_at`, `count=0`, cặp `hidden_*`, `kind`/`author_id`, `kind`/`verification_state`) | 10–13, 15, 19–22, 35 |
| Trùng khoá (`client_msg_id` lặp, hai bài AI một thread) | 09, 18 |
| Trộn lớp / trộn người qua FK | 24–27 |
| Xoá tài liệu đang được phiên chat dùng | 28 |
| `pii_events` bị sửa / xoá / chứa nội dung | 32–34, 47 |
| Danh sách quét toàn bảng (`Seq Scan`) ở 10.000 dòng | 39 |
| Nội dung tin nhắn rò sang bảng khác | 47, 48 |

## Phân quyền
| Ca | TC |
| --- | --- |
| Vai DB của ứng dụng không vượt được trigger append-only, không `superuser` | 49 |
| SVB đọc tin của SVA → `404` (không lộ tồn tại) | 50 |
| TA / TEACHER / ADMIN đọc chat riêng → không có đường (`403`/`404`) | 50 |
| DB chặn ghi chéo lớp kể cả khi service ghi sai (`23503`) | 24–27 |
| `chat_messages.user_id` buộc bằng chủ phiên (chống lộ chat riêng do lỗi service) | 26 |

## Script chạy được
- `docs/sprints/6/qc/scripts/p601-schema.py` — bảng kỳ vọng **chép tay từ SRS 5.1–5.6** (enum + 5 bảng + 11 chỉ mục), so với `information_schema` / `pg_indexes` / `pg_constraint`; in từng dòng lệch. Dùng ở TC 06, 17, 32, 38. Chạy: `python3 docs/sprints/6/qc/scripts/p601-schema.py --dsn "$DB" --tables all`.
- Test Go của dev (`./internal/store`, `./internal/chat`) chỉ chạy **thêm** ở TC 16, 23, 31, 37, 40, 42, 46, 48; TC chính vẫn là phép đo bằng `psql` của QC.
- Cổng phase: `cd backend-go && go test -race ./internal/privacy/...` (`docs/phases/P3.md`) không thuộc story này; `scripts/gate-p3.sh` ở US-P3-08.

## Câu hỏi cho BA (Q-QC-…)
- **Q-QC-P3-01-1** — AC8 viết "Given `goose up` chạy lần hai khi đã `up` … Then lệnh trả **mã khác 0**". `goose` chuẩn trả `rc=0` và in "no migrations to run" khi không còn gì để chạy. AC đang yêu cầu dev bọc lại `goose` để trả mã lỗi, hay chỉ yêu cầu "không hỏng DB"? QC chưa viết được kết quả mong đợi dứt khoát cho TC 43 — *chờ BA*.
- **Q-QC-P3-01-2** — AC7 nói `schema_test.go` "ghim danh sách cột của **sáu** bảng mới", nhưng `00007` + `00008` chỉ tạo **năm** bảng (`chat_sessions`, `chat_messages`, `forum_threads`, `forum_posts`, `pii_events`); SRS 5 cũng chỉ mô tả năm. Bảng thứ sáu là bảng nào (tag của thread nằm ở cột `tags text[]` chứ không phải bảng riêng)? — *chờ BA*.
- **Q-QC-P3-01-3** — AC1 cấm `ALTER` nhưng `docs/sprints/6/proposals.md` #6 (PM **ACCEPTED**) yêu cầu `ALTER TABLE users ADD CONSTRAINT … CHECK (ics_token ~ '^[0-9a-f]{64}$')` trong `00009`. Ngoại lệ đó chỉ áp cho `00009_calendar` hay cũng áp cho `00007`/`00008`? QC đang chấm `00007`/`00008` là **0 `ALTER` chạm bảng cũ** (TC 03) — *chờ BA xác nhận*.
- **Q-QC-P3-01-4** — AC6 yêu cầu `EXPLAIN` "không có `Seq Scan`" trên 10.000 dòng. Với bảng nhỏ (`pii_events` chỉ vài trăm dòng ở seed) Postgres chọn `Seq Scan` là đúng. QC chỉ chấm bốn danh sách nêu trong AC và bỏ qua bảng < 1.000 dòng (TC 39). Đúng ý AC? — *chờ BA*.

## Lịch sử sửa TC
(chỉ sửa khi SPEC đổi: ghi ngày, TC nào, lý do, số proposal PM đã chấp nhận)
- 2026-10-10 — tạo mới theo `US.md` v1.2 / `SRS.md` v1.2 (APPROVED).

Tổng TC: 50
