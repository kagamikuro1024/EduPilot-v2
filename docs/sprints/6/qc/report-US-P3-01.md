# QC report — US-P3-01 (migration `00007_chat_threads`, `00008_privacy`, sqlc)  · Kết luận: PASS (sau quyết định PM Q1–Q3, 2026-10-10)

Handoff: `docs/sprints/6/handoff/dev-US-P3-01.md` (commit `f1a5b9c`, kiểm trên `31fe6f0`). Bộ TC: `tc-US-P3-01.md` (50 TC, không sửa).
Môi trường: DB trống `qc_p301` / `qc_p301b` tạo **trong** Postgres 18 của stack dev đang chạy (cổng 5433), không down stack, đã `DROP` sau khi chạy. `goose v3.28.0` build ngoài repo (`/tmp/goose`). Không có UI → không cần `playwright-cli`; `ui-antipatterns.sh` không áp dụng.

**Cập nhật theo PM (Q1–Q3, `proposals.md`):**
- **Q1:** TC thiếu bề mặt chuyển sang story giao bề mặt; story này đóng khi phần còn lại PASS.
- **Q2:** TC-49 (vai DB siêu quyền) ghi nợ PR, không tính vào story này.
- **#8 (spec v1.3):** AC7 "năm bảng" → TC-42 PASS (`chat_schema_test.go` ghim đủ 5 bảng); AC8 tách hai ca → TC-43 PASS (`up` lần hai `rc=0`, "no migrations to run", DB không đổi).

| TC chuyển | Đích |
| --- | --- |
| TC-47, 48, 50 (AC9 nội dung chat chỉ ở `chat_messages`, đọc chéo) | `report-US-P3-05` (AC9 của US-P3-01 kiểm cùng `TestChatContentOnlyInMessages`) |
| TC-49 | nợ PR (Q2) |

Kết quả còn lại: **PASS 46 / 46** (TC-01…46 trừ không có TC FAIL). AC1–AC8 PASS; AC9 chờ P3-05 (không phải FAIL của story này theo Q1).

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `goose up` trên DB trống → `goose status` | PASS — 8 `Applied`, `00007`/`00008` mới: `goose: successfully migrated database to version: 8` |
| `down`×2 → 0 bảng, 0 enum → `up` → `pg_dump` diff | PASS — diff chỉ 2 dòng `\restrict`/`\unrestrict` (token ngẫu nhiên của pg_dump), schema giống hệt |
| `grep -ciE '^\s*alter table'` hai tệp | PASS — 0 và 0 |
| `sqlc generate && sqlc diff`; `git status internal/store` | PASS — rc=0, rỗng |
| `go test ./internal/store -run 'TestSchema…TestChatForumIndexesUsed…'` | PASS — `ok … 3.19s` |
| `python3 scripts/p601-schema.py --tables all` | PASS — 10 enum, 10 chỉ mục có tên, 35 cột (5.2+5.3) lệch 0. (Script QC có lỗi regex + đường dẫn SRS, đã sửa trong lượt này; không phải lỗi dev.) |
| `EXPLAIN (ANALYZE)` 4 truy vấn trang đầu, 10.000 dòng/bảng | PASS — đều Index Scan đúng chỉ mục (`chat_sessions_user_idx` 0,041 ms; `chat_messages_session_idx` 0,017 ms; `forum_threads_course_idx` 0,017 ms; `forum_posts_thread_idx` 0,013 ms); 0 Seq Scan trên bảng chính |

## TC
| TC | Kết quả | Ghi chú |
| --- | --- | --- |
| 01–06 | PASS | `pg_dump` diff chỉ khác token `\restrict`; `git diff origin/main` migration 00001–00006 rỗng; PROGRESS có 00007/00008/00009 |
| 07–08 | PASS | `partial_content`, `stream_status` nằm trong `CREATE TABLE`; enum đúng giá trị + thứ tự |
| 09–15 | PASS | `23505` (trùng `client_msg_id`), hai NULL chèn được; `23514` cho USER+STREAMING, USER+confidence, DONE+partial, STREAMING+completed_at, content 20001, attempt 0, intent chữ thường; `updated_at` tăng |
| 16 | PASS | `TestSchemaChat` ok |
| 17–22 | PASS | `vector(1536)`; `23505` hai bài AI; `23514` cho AI+author, HUMAN không author, cặp `hidden_*` (cả 2 chiều), HUMAN+PENDING, AI+NONE, tag 6 phần tử / 31 ký tự, week 0 / 21, title rỗng, body 8001 |
| 23 | PASS | `TestSchemaForum` ok |
| 24–27 | PASS | `23503` cho bài chéo lớp, tin chéo lớp, `user_id` khác chủ phiên, `reply_to` chéo lớp |
| 28 | PASS | xoá `documents` thành công; `chat_sessions.document_id` → NULL |
| 29–30 | PASS | `last_message_at` mặc định `now()`, NULL → `23502`; `forum_threads.updated_at` tăng. `forum_posts` có trigger `set_updated_at` (thấy trong `\d`), chưa chạy UPDATE riêng cho bảng này |
| 31 | PASS | test Go ok |
| 32–37 | PASS | `pii_events` đúng 9 cột, 0 cột text; `UPDATE`/`DELETE` → `42501` (kể cả khi vai là superuser); `count` 0 / −1 → `23514`; enum đúng |
| 38–40 | PASS | 10 chỉ mục có tên + `chat_messages_idem_key` = 11; `EXPLAIN` xem trên; `TestChatForumIndexesUsed` ok (chạy không kèm `-tags integration`) |
| 41 | PASS | sqlc sạch |
| 42 | PASS (v1.3 #8) | `queries/{chat,forum,privacy}.sql` có; `TestSchema*` ok; "sáu bảng" ↔ 5 bảng (Q-QC-P3-01-2) |
| 43 | PASS (v1.3 #8) | `goose up` lần hai: `rc=0`, "no migrations to run. current version: 8", DB không đổi. AC8 viết "mã khác 0" (Q-QC-P3-01-1) |
| 44 | PASS | tệp 00008 hỏng giữa `CREATE TABLE`: `rc=1`, thông điệp có tên tệp + SQL; 0 bảng `pii_events`, 0 enum `pii_*`; `goose status` ghi `00008 Pending` |
| 45–46 | PASS | `22P02` cho enum lạ; test ok |
| 47 | CHUYỂN → P3-05 | chưa có đường chat (US-P3-05) |
| 48 | CHUYỂN → P3-05 | `internal/chat` chưa tồn tại; dev xác nhận hoãn `TestChatContentOnlyInMessages` |
| 49 | NỢ PR (Q2) | xem BUG-1. Phần trigger: PASS |
| 50 | CHUYỂN → P3-05 | chưa có route đọc tin (US-P3-05) |

Tổng (sau Q1–Q3): PASS 46 · FAIL 0 · chuyển P3-05: 3 · nợ PR: 1.

## AC
| AC | Kết quả |
| --- | --- |
| AC1 up/down/up, 0 ALTER, không sửa 00001–06 | PASS |
| AC2 `chat_sessions`/`chat_messages` | PASS |
| AC3 `forum_*` | PASS |
| AC4 FK phức hợp, `SET NULL`, trigger | PASS |
| AC5 `pii_events` append-only, không văn bản | PASS |
| AC6 chỉ mục + `EXPLAIN` | PASS |
| AC7 sqlc + `schema_test.go` ghim cột | PASS (năm bảng, v1.3) |
| AC8 migration lỗi / chạy lại | PASS |
| AC9 nội dung chat chỉ ở `chat_messages` | chuyển P3-05 |

## Lỗi
- **BUG-1 (Thấp, hạ tầng có từ trước, không do story này)** — TC-49. Vai DB của stack dev (`edupilot`) có `rolsuper=t`. Tái hiện: `docker exec edupilot-postgres-1 psql -U edupilot -d qc_p301 -c "select rolsuper,rolbypassrls from pg_roles where rolname=current_user"` → `t | t`. Kỳ vọng: vai ứng dụng không siêu quyền. Chuỗi kết nối của gateway (qua pgbouncer) không đọc được từ container (env rỗng) nên chưa xác nhận gateway dùng vai này. Trigger append-only vẫn chặn cả superuser (`42501`). Đề nghị PM quyết (đã ghi `proposals.md`).
- **BUG-2 (AC9 chưa giao)** — không phải lỗi mã, là AC chưa làm; xem trên.

## Kiểm phản mẫu / luật mở rộng / phân quyền
Luật 6 (không sửa migration cũ): PASS. Luật 13 (chỉ mục bắt đầu `course_id`, cursor `id DESC`): PASS. Phân quyền đọc chéo (TC-50) chờ P3-05. Không `fetch`, không thư viện mới (story chỉ đụng SQL/sqlc/test).

## Đề nghị
Đóng story. BUG-1 (vai DB siêu quyền) thành nợ PR; AC9 kiểm ở P3-05.
