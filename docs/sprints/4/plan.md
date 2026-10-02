# Sprint 4 — P2 Lớp học (tài khoản an toàn, mở lớp, mã tham gia, Hôm nay)

Trạng thái: **PM tự duyệt 2026-10-03** (chạy cuốn chiếu theo `docs/team/PM.md`; chủ dự án: "tôi tin bạn, đừng để thời gian trống") · Nhánh `sprint/4-p2`, worktree `../TA_Agent_v2-s4`, **xếp chồng** trên `sprint/3-pu-p1`. Dev chỉ bắt đầu khi sprint 3 xong PU (US-PU-03 `apiClient`) — trước đó BA viết spec, QC viết TC.

## Mục tiêu
Trọn `docs/phases/P2.md`, luồng **F1** (tài khoản) và **F2** (mở lớp → phân công → mã tham gia → vào lớp) đi được từ đầu đến cuối trên dữ liệu thật; seed bằng API thật; thay các màn mock `/login`, `/admin/*`, `/join`, `/class/*`, `/` bằng màn thật (D51). Cuối sprint: `gate P2`.

## Feature và story (lát dọc: API + màn + test trong cùng story)

**FEAT-account-security (F1)**

| # | Story | Lát P2 | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 1 | US-P2-01 Migration `00004 auth_hardening` + sqlc; `internal/mail` (go-mail + `html/template`) + consumer `mail_outbox` (retry 3, dead-letter) qua Mailpit | L1b | M | PG |
| 2 | US-P2-02 Đăng nhập thật: access 15 phút + refresh xoay vòng cookie `httpOnly; Secure; SameSite=Lax`, phát hiện dùng lại → thu hồi chuỗi, `auth_sessions`, đăng xuất; `apiClient` tự làm mới một lần khi 401; bỏ cổng token dev của sprint 3; `/login` thật | L1b | L | 01, PU-03 |
| 3 | US-P2-03 Đăng ký (chỉ STUDENT) + xác minh email (token băm, 24 h, gửi lại sau 60 s) + `/register`, `/verify-email` | L1b | M | 01, 02 |
| 4 | US-P2-04 Quên / đặt lại mật khẩu (30 phút, phản hồi đồng nhất, thu hồi mọi phiên), đổi mật khẩu, `/settings` thiết bị + thu hồi; `/forgot-password`, `/reset-password` | L1b | M | 02, 03 |
| 5 | US-P2-05 Chống dò: chờ tăng dần sau 5 lần, khoá 15 phút sau 10 lần + mail, rate limit IP, mật khẩu ≥ 10 ký tự + danh sách phổ biến | L1b | S | 02 |
| 6 | US-P2-06 Admin tạo GV/TA → link mời 72 h → `/invite/[token]`; khoá tài khoản; `/admin/users` (bảng + Drawer) | L1b, L2 | M | 01, 02 |

**FEAT-course-foundation (F2, M0, M14)**

| # | Story | Lát P2 | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 7 | US-P2-07 Migration `00003 course_foundation` (dạng cuối, gồm `documents`, `content_chunks`, `document_courses`); `CourseAccessGuard` thật trên `enrollments`; `/me/courses` + bộ chọn lớp thật; hồ sơ `/me/profile` | L1, L2 | M | 02 |
| 8 | US-P2-08 Admin mở / sửa / lưu trữ lớp, `join_code` (crypto/rand, bảng 31 ký tự), gán TEACHER/TA → outbox `course.assigned` → `notifications` → chuông thật (`GET /notifications`, làm mới 30 s); `/admin/courses` | L2 | L | 06, 07 |
| 9 | US-P2-09 Mã tham gia: preview + join (idempotent, 5 lần / 10 phút, lỗi đồng nhất `JOIN_CODE_INVALID`), duyệt, `/join`, `/join/[code]` (quay lại đúng link sau đăng nhập), `/class/settings` (tạo lại mã qua `<ConfirmIrreversible>`), `/class/members` (duyệt hàng loạt, mời ra + Hoàn tác 5 s) | L2a | L | 07, 08 |
| 10 | US-P2-10 Import roster CSV/XLSX (báo dòng lỗi) + **quy tắc nối chỉ bằng email đã xác minh** (MSSV trùng email khác → PENDING + cảnh báo), test mạo danh; `share-from` tài liệu / câu hỏi / nháp công thức giữa lớp cùng `subject_code` | L2a | M | 03, 09 |
| 11 | US-P2-11 "Hôm nay": `internal/today` (Provider + xếp hạng luật cứng, cache Redis 60 s xoá theo outbox), `GET /me/today`, `GET /courses/{id}/today`, "Tất cả lớp của tôi", Provider: thiết lập lớp mới, yêu cầu chờ duyệt, email lệch MSSV, hạn bài, buổi kế; trang `/` thật | L2c | L | 08–10 |
| 12 | US-P2-12 `scripts/seed.mjs` đi luồng API thật (2 lớp × 30 SV, mã `AN7K2MQ` / `BX4P9TW`, 3 SV học hai lớp, SV D chưa vào lớp), tự chạy khi `SEED_ON_EMPTY_DB=true` | L3 | M | 06–11 |

Spec: `docs/specs/FEAT-account-security/`, `docs/specs/FEAT-course-foundation/` — BA viết, PM duyệt. TC: `docs/sprints/4/qc/`.

## Quyết định PM tự chốt
- **L2b (cursor, Idempotency-Key, ETag, 409 version, jobs, outbox) đã xong ở PG** → không làm lại; P2 chỉ áp dụng cho API mới.
- "Thêm `TEACHER` vào `user_role`": `00001` đã có đủ 4 vai → bỏ việc này, chỉ kiểm điều hướng theo vai.
- Mail dev đi qua Mailpit (đã có trong compose); test tích hợp đọc REST API Mailpit.
- Màn thật thay mock theo D51: `/login`, `/register`, `/verify-email`, `/forgot-password`, `/reset-password`, `/invite/[token]`, `/settings`, `/admin/courses`, `/admin/users`, `/join*`, `/class/*`, `/`. Các màn mock còn lại vẫn chạy bằng dữ liệu mock nhưng **đổi vai bằng đăng nhập thật** thay cookie `ep_demo_*` — BA chốt cách hai chế độ cùng tồn tại (tài khoản seed ↔ người mock A/B/C/D).
- Câu hỏi đụng quyền / dữ liệu cá nhân / chính sách mật khẩu: BA ghi kèm mặc định an toàn; PM chốt theo mặc định và **báo chủ dự án trong báo cáo** để chủ dự án có thể đảo.

## Rủi ro
- Auth là phần mọi phase sau kế thừa: QC có TC tấn công (mạo danh MSSV, dùng lại refresh, đoán mã tham gia, CSRF với cookie `SameSite=Lax`).
- Nhánh xếp chồng hai tầng (4 trên 3 trên 2 + 1.5) → PM gộp lại sau mỗi lần nhánh dưới đổi; dev không bắt đầu P2 trước khi PU-03 xong.
- Seed qua API thật phụ thuộc mọi story → làm cuối, dùng làm kiểm luồng F1 + F2.
