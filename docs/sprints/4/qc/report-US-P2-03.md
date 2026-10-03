# Báo cáo QC — US-P2-03 (đăng ký STUDENT, xác minh email, gửi lại 60 s, chống mạo danh MSSV)
**Kết luận: PASS có điều kiện** — 1 lệch nhỏ (L1), 4 TC chờ story / hạ tầng khác. **Không có lỗ hổng**: tấn công mạo danh MSSV, nâng quyền, liệt kê email, dùng lại / đua link, đoán mã đều bị chặn. Bản chấm `a7e264d`/`8575345` (`sprint/4-p2`), stack riêng của QC (Postgres, Redis, Mailpit, 2 gateway `testroutes`, worker thật; Next `build` ở 3400; Chrome for Testing). `go test -race -tags integration ./internal/auth/... ./internal/mail/...` ok (191 `--- PASS`, 0 FAIL); `go vet`, `golangci-lint` 0 issues; `internal/contract` ok; Playwright 221 pass (xem Ghi chú). Q-QC-P203-1/-2: BA đã trả lời, QC làm theo.

## Lệch
- **L1 (TC-11).** `student_code` là khoảng trắng toàn chiều rộng `"　"` (U+3000): `202`, lưu **rỗng** (coi như bỏ trống) thay vì `422`. Không rủi ro (không dùng để nối lớp), nhưng khác chữ TC ("sai định dạng → 422"). Chuỗi có khoảng trắng giữa (`"2022 9002"`) và các ca sai khác đều `422` đúng. Đề nghị dev cắt khoảng trắng Unicode rồi kiểm; BA chốt "toàn khoảng trắng = bỏ trống" có chấp nhận không.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `register` → `202` `{message:"Nếu email này dùng được…"}`; DB `STUDENT|PENDING_VERIFICATION|email_verified_at null|$2a$` (bcrypt); không `user_id` |
| 02 | PASS | 9 thân thêm trường (`role` TEACHER / ADMIN, `status`, `email_verified_at`, `is_admin`, `id`, `password_hash`, `profile.role`, khoá `role` trùng): **cả 9 `422 VALIDATION_FAILED`**; 0 dòng `users` mới; 0 vai khác STUDENT được tạo |
| 03, 06, 10, 12, 16, 19, 22, 25, 28, 30, 36 | PASS | test Go của dev (`-race`, `-tags integration`) — gồm `TestRegisterCreatesStudentOnly`, `…RejectsRoleField`, `…UniformResponse`, `…TimingEqualized`, `TestSelfDeclaredStudentCodeLinksNothing`, `TestVerifyEmailSingleUseRace`, `TestResendThrottle60s`, `TestPublicEndpointsIgnoreJWTRole`, `TestRegisterSendsVerifyMail`. `TestUnverifiedCannotJoin` **không có** (chờ US-P2-09, dev đã ghi) |
| 04 | PASS | 5 trạng thái (mới, ACTIVE, PENDING, INVITED, DISABLED): thân + mã + header **y hệt** (0 `Set-Cookie`); trung vị 20 mẫu: mới 221 ms / đã có 222 ms (lệch ~0 %) |
| 05 | PASS | mới → `verify_email`; ACTIVE và PENDING → `email_exists`; INVITED → "Lời mời tham gia EduPilot" (`invite_staff`); DISABLED → **0 thư**; mỗi email đúng 1 dòng `users` |
| 07 | PASS (**tấn công chính**) | `sv.gioi` ACTIVE trong lớp INT1006 với MSSV `20229002`. Kẻ mạo danh đăng ký `student_code:"20229002"`, xác minh email, đăng nhập: **0 enrollment**, tổng `enrollments` 1 → 1, bản ghi `sv.gioi` nguyên vẹn (đúng `user_id`); `me/courses`, `courses/{id}`, `members`, `gradebook` đều `404` (route lớp chưa có ở bản này — bằng chứng chính là DB không có dòng nối). `users.student_code` của kẻ tấn công lưu `20229002` nhưng vô tác dụng |
| 08 | PASS | MSSV có khoảng trắng / chữ lẫn / trùng: không nối; bản ghi roster `PENDING` (`EMAIL_UNVERIFIED`) của người khác với MSSV `20229099`, kẻ mạo danh đăng ký + xác minh với MSSV đó: 0 enrollment mới, bản ghi `PENDING` vẫn của đúng người |
| 09 | PASS | `grep student_code` ở `queries/*.sql`: chỉ 2 chỗ **GHI** (`InsertPendingStudent`, `users.sql`) — không truy vấn nào dùng MSSV để nối / lọc |
| 10 | PASS | gieo `zz_qc.sql` (`join … on u.student_code = e.student_code_snapshot`) → `TestNoQueryLinksByStudentCode` **đỏ**; xoá → xanh; `git status` sạch |
| 11 | PASS có L1 | `AB12`, 16 ký tự, `2022 9002`, `2022-9002`, `<script>` → `422`; `abc123456` → `202`, lưu `ABC123456` (chữ hoa); `"　"` → xem L1 |
| 13 | PASS | xác minh `200` → `ACTIVE`, `email_verified_at`, `used_at`; lần hai `410 LINK_INVALID` `details.reason="used"` |
| 14 | PASS | 50 `verify-email` song song cùng token: **1×200, 49×410** |
| 15 | PASS | `auth_tokens.token_hash = <bản rõ>` 0 dòng; `pg_dump` chứa token 0 lần |
| 17 | PASS | hết hạn → `410 expired`; ngẫu nhiên / sai độ dài / ký tự lạ → `410 invalid`; rỗng và thiếu trường → `422`; mọi 410 cùng khoá JSON `code,details,message,trace_id`; tài khoản không đổi |
| 18 | PASS một phần | 1.000 token ngẫu nhiên: 0 trúng, 1.000×`410` (2,3 s). Giới hạn IP **chưa có** (0 `429`) — thuộc **US-P2-05** (dev đã ghi); QC chấm lại ở P2-05 |
| 20 | PASS | `resend` `202` rồi `429` `retry_after=60`; đúng 2 thư (đăng ký + gửi lại); token cũ → `410 invalid` (đã thu hồi); sau 61 s gửi lại `202` |
| 21 | PASS | email không tồn tại: `202` rồi `429` (giống email thật), 0 user mới; đã xác minh: `202`, **0 thư**; thân 202 giống hệt |
| 23 | **chờ US-P2-09** | `courses/join*` chưa có; đăng nhập chưa xác minh: `200`, `email_verified=false` (TC-P203-23 phần đăng nhập PASS) |
| 24 | PASS | roster `PENDING` + `EMAIL_UNVERIFIED` của `pending@`; xác minh email → enrollment `ACTIVE`, user `ACTIVE verified`; chỉ email đã xác minh trùng được đẩy |
| 26 | PASS | email: `a@b`, `a b@c.d`, `a@@b.c`, 255 ký tự, xuống dòng → `422`; `a+tag@x.vn` `202`; `A@B.VN` → lưu chữ thường (0 email có chữ hoa); tên `Nguyễn Văn Ặ` `202`, toàn khoảng trắng / 101 ký tự / ký tự điều khiển → `422`, 100 ký tự `202`; mật khẩu 9 byte `422`, 10 và 72 byte `202`, 73 byte `422`; thân 2 MB → `413` |
| 27 | PASS | tên `<script>alert(1)</script>`, `'; drop table users;--`, `{{7*7}}`, `${jndi:ldap://x}`: `202`, lưu nguyên văn, bảng `users` còn; HTML thư thoát `<script>`; `{{7*7}}` không được thực thi (không "49") |
| 29 | PASS | 1 thư "Xác minh email EduPilot của bạn"; liên kết `APP_PUBLIC_URL/verify-email?token=<43>` đúng 1 tham số; có Text + HTML; không mật khẩu / MSSV / `{{` |
| 31 | PASS | `/register`: 4 ô (Họ và tên `name`, Email `username`, MSSV `off` kèm chú thích "Chỉ để giảng viên đối chiếu; không dùng để vào lớp.", Mật khẩu `new-password`), **1** nút primary `Tạo tài khoản`; điền sai → lỗi dưới từng ô (`aria-invalid`, chữ đã gõ giữ); gửi xong: "Kiểm tra email của bạn…" (đúng câu AC2) + `Gửi lại thư (58 giây)` khoá đếm ngược; mật khẩu không còn trong DOM |
| 32 | PASS | `/verify-email?token=…`: xác minh tự động, URL về `/verify-email` (token bị xoá), HTML / storage không chứa token; dùng lại: "Liên kết đã được dùng…"; thiếu token: ô email "Gửi lại thư". Header `Referrer-Policy: no-referrer` + `Cache-Control: no-store` cho `/verify-email`, `/reset-password`, `/invite/*` (trên `next start`; **chưa qua Caddy**) |
| 33 | PASS | 375 px `/register`, `/verify-email`, `/forgot-password`, `/reset-password`, `/login`: `AUDIT` `ox:0 cut:0 ell:0`, `TOUCH` `[]`, 0 từ kỹ thuật |
| 34 | PASS | trong Playwright 221 pass (`account.spec.ts` các ca "register page", "verify page") |
| 35 | PASS | `register` kèm JWT ADMIN: vai tạo ra `STUDENT`; `PATCH /me/profile` → `404` (route chưa có, chờ US-P2-07 — dev đã ghi) |
| 37 | một phần | QC đã đi tay từng khúc (đăng ký → Mailpit → xác minh → đăng nhập, dùng lại liên kết, gửi lại sau 61 s); ca `@real` trọn luồng qua `https://localhost`/Caddy **chưa chạy** (stack QC không có Caddy) |
| 38 | PASS | log 2 gateway + worker: email đăng ký 0, MSSV 0, mật khẩu 0; `pg_dump` chứa mật khẩu rõ 0 |
| 39 | PASS | `go vet`, `golangci-lint` 0 issues, `go test -race` (auth, mail) ok |

## Ghi chú
- Chạy toàn bộ trong một lượt tải nặng có 2 test Go (`TestRateLimit_IP`, `TestRateLimit_SharedAcrossInstances`) và 4 ca Playwright (`settings-llm` ×1, `visual settings-llm` ×2, `dev-ui` mobile ×1) đỏ **một lần**; chạy riêng từng nhóm đều xanh (Go 3,4 s ok; Playwright 76 pass / 30 skip). Nghi nhạy với tải máy (nhiều tiến trình song song), không lặp lại khi chạy riêng; ghi nhận, không tính lỗi.
- `invited@` là TEACHER/INVITED nên nhận `invite_staff`; nhánh INVITED **sinh viên** (roster) chưa gửi thư (chờ US-P2-10, dev đã ghi).

## Việc sau
Dev: L1. BA: chốt L1. QC chấm lại: TC-18 (giới hạn IP) ở P2-05, TC-23 ở P2-09, TC-37 (Caddy) ở cổng P2.
