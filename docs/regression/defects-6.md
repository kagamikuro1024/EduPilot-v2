### DEF-R6-01 — Vai DB của ứng dụng là siêu quyền
- Mức: Thấp
- Sprint gốc: 6 · TC gốc: TC-P3-01-49 · Hồi quy do sprint sau?: KHÔNG (có sẵn — chính là BUG-1 trong `docs/sprints/6/qc/report-US-P3-01.md`)
- Môi trường: ep-r, be0c0d96; gateway nối Postgres bằng vai `edupilot`
- Bước tái hiện: 1. `docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -Atc "select rolsuper,rolbypassrls from pg_roles where rolname='edupilot'"` → `t|t`.
- Kỳ vọng: vai ứng dụng không `rolsuper`; Thực tế: `rolsuper=t`. Trigger append-only (`pii_events`, `audit_log`) vẫn chặn UPDATE/DELETE với 42501 nên không mất bất biến.
- Bằng chứng: lệnh trên; `docs/regression/smoke-6.md` · Nghi ngờ mã: cấu hình compose (user Postgres = user ứng dụng), không phải mã Go
- Chạy lại lần 2: ổn định

### DEF-R6-02 — Bộ nhớ đệm roster PII không bị vô hiệu khi sinh viên vào lớp (duyệt hoặc vào bằng mã)
- Mức: Cao (tên SV mới vào lớp không được che trong Threads/chat tới 1 giờ ⇒ lộ họ tên người khác ở kênh công khai)
- Sprint gốc: 6 · TC gốc: TC-P3-02-21 · Hồi quy do sprint sau?: KHÔNG RÕ (có vẻ có sẵn: `members.go` ở db7c187b đã chỉ phát `course.member_changed` ở mời ra / hoàn tác; duyệt phát `course.join_decided`, vào tự động không phát gì; worker chỉ gắn `roster.Invalidate` vào `course.member_changed`, `cmd/worker/registry.go:129`)
- Môi trường: ep-r, be0c0d96; GV `teacher`, SV; lớp E6C1 (`class_code=E6C1`)
- Bước tái hiện: 1. Bật `require_approval` (PUT `/courses/{id}/join-settings`). 2. `redis-cli DEL ep:roster:<course>`; SV A `POST …/threads/precheck` để dựng cache (không có SV B). 3. SV B `POST /courses/join` (PENDING) rồi GV `POST /courses/{id}/members/{B}/approve` (200). 4. `redis-cli EXISTS ep:roster:<course>` sau 10 s → vẫn `1`; `precheck` với «Đỗ Bảo Hải …» (tên SV B) → `allowed:true`. Tương tự khi lớp không cần duyệt (vào mã). Mời ra thì cache xoá sau 0,5 s (đúng).
- Kỳ vọng (SRS P3-02 AC3 / TC-P3-02-21): cache roster xoá ≤ 5 s sau khi thành viên đổi, tên SV mới bị chặn; Thực tế: cache sống đến hết TTL 1 giờ.
- Bằng chứng: lệnh trên (chạy hai lần, cùng kết quả); outbox chỉ có `course.member_changed` cho thao tác mời ra · Nghi ngờ mã: `backend-go/internal/course/members.go:222` (approve chỉ ghi `TopicJoinDecided`), `backend-go/internal/course/join.go:185-269` (không ghi sự kiện khi ACTIVE ngay), `cmd/worker/registry.go:129`
- Chạy lại lần 2: ổn định
- **ĐÃ SỬA (afbd433e), RERUN 2026-11-16 ĐẠT:** vào bằng mã (không duyệt) → outbox `course.member_changed` +1, `ep:roster` xoá ≤ 3 s; duyệt 1,0 s; mời ra 0,5 s. TC-P3-02-21 đạt. (Lượt trước còn đỏ vì gateway ep-r chạy ảnh cũ do cửa sổ chaos khôi phục nhầm ảnh `edupilot-gateway`.)
### DEF-R6-C-01 — Dòng khoá ở /threads dùng chữ của chat thay vì "Đăng bài tạm khóa…"
- Mức: Thấp
- TC: TC-P3-06-42
- Hồi quy do sprint sau?: KHÔNG (có từ commit 3f1e376a, US-P3-06)
- Bước tái hiện: 1. SV có lượt thi IN_PROGRESS. 2. Mở /threads?course=<lớp>, bấm "Đặt câu hỏi". 3. Đọc dòng báo trạng thái trên ô soạn.
- Kỳ vọng / Thực tế: SRS FEAT-private-chat-pii 472 "Đăng bài tạm khóa trong lúc bạn làm bài thi. Dùng lại được sau {HH:mm}." / hiện "Chat tạm khóa trong lúc bạn làm bài thi. Dùng lại được sau 00:08." (e2e threads.spec.ts:180 cũng khẳng định chữ chat).
- Nghi ngờ mã: frontend/src/features/threads/RealThreads.tsx:186

- LT-đã-xoá TC-P3-01-05: proposals #13 (số 00009 bỏ trống, calendar là 00011)
- LT-đã-xoá TC-P3-03-31: GET /admin/llm/usage mở cho TEACHER: openapi «ADMIN, TEACHER» (sprint 9/10)
- LT-đã-xoá TC-P3-04-45: US-P4-07 2026-10-14: khủng hoảng hai tầng, câu mẫu đổi
