# Báo cáo QC — US-P2-04 (quên / đặt lại / đổi mật khẩu, thiết bị `/me/sessions`, xem trước token)
**Kết luận: PASS có điều kiện** — không lỗ hổng; chờ US-P2-05 cho `PASSWORD_COMMON` / `PASSWORD_CONTAINS_EMAIL` và giới hạn IP; log Caddy mới kiểm ở cấu hình. Bản chấm `8575345` (`sprint/4-p2`); stack như P2-03. IDOR `/me/sessions` (kể cả Admin), link đặt lại dùng hai lần / đua, thu hồi phiên đều đạt.

| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | 5 email (ACTIVE, PENDING_VERIFICATION, INVITED, DISABLED, không tồn tại): thân `202 {"message":"Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu."}` + header **y hệt**; trung vị 8 mẫu: tồn tại 115 ms / không tồn tại 92 ms (lệch ~20 % ≤ 35 %; sàn 80 ms); giới hạn 3 lần/giờ/email: `202,202,202,429,429` |
| 02 | PASS | chỉ ACTIVE và PENDING_VERIFICATION nhận `reset_password`; INVITED / DISABLED / không tồn tại 0 thư |
| 03, 10, 13, 19, 22, 26, 34, 36, 38 | PASS | test Go của dev ok (`TestForgotUniform`, `TestResetSingleUseRace`, `TestChangePasswordRateLimit`, `TestRevokeOthersSessionIs404`, `TestTokenPreviewRateLimit`, `TestResetMail`, `TestPasswordChangedMail`…; 191 `--- PASS`) |
| 04 | PASS | `reset-password` `200 {"status":"password_reset"}`: **không** `access_token`, **không** `Set-Cookie`; mật khẩu mới đăng nhập `200`, cũ `401` |
| 05 | PASS | dùng lại → `410 LINK_INVALID` `reason="used"` |
| 06 | PASS | 50 `reset` song song cùng token với 50 mật khẩu khác: **1×200, 49×410**; đăng nhập bằng mật khẩu của yêu cầu thắng `200`, bằng mật khẩu khác `401` |
| 07 | PASS | 2 phiên A, B: access cũ `401` ở cả hai gateway sau **68 ms**; refresh cả hai `401 SESSION_REVOKED`; `revoked_reason=PASSWORD_RESET` |
| 08 | PASS | `failed_logins` 7 → 0, `locked_until` → null; tài khoản PENDING qua đặt lại → `ACTIVE`, `email_verified_at` có; đúng 1 thư `password_changed` |
| 09 | PASS | token rõ ở DB 0; mật khẩu mới ở `pg_dump` 0 và ở thư 0 |
| 11 | PASS một phần | token hết hạn → `410 expired`; `abc` → `422 PASSWORD_TOO_SHORT`; 73 ký tự → `422 PASSWORD_TOO_LONG`; **token không bị tiêu** sau 2 lần lỗi (nhập mật khẩu mạnh với cùng token → `200`); token **xác minh** dùng để đặt lại → `410 invalid` và token xác minh còn dùng được (`200`). **Chờ US-P2-05:** `password1234` (phổ biến) và mật khẩu chứa email vẫn được chấp nhận ở bản này — dev đã ghi, không FAIL |
| 12 | PASS một phần | 300 token ngẫu nhiên vào `reset-password`: 0 trúng, 300×`410`; `tokens/preview` giới hạn 20/phút/IP: 16 lần `200` rồi `429` + `Retry-After: 3` (QC đã dùng 6 lần trước đó trong cùng phút). Giới hạn IP của `reset-password` → US-P2-05; chênh thời gian token không tồn tại: chỉ ghi (Q-QC-P204-1) |
| 14 | PASS | máy B (Chrome) ở `/threads`; máy A đặt lại mật khẩu; B tải lại → `/login?next=%2Fthreads&revoked=password_reset` + "Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi." |
| 15 | PASS | refresh của hai phiên sau reset đều `401` |
| 16 | không kiểm được | ca `@real` hai-context của dev là `test.skip`; QC thay bằng TC-14 (đã PASS) |
| 17 | PASS | `POST /me/password` ở P1: `204`; P1 `refresh` `200` (giữ); P2 `401 SESSION_REVOKED`, `revoked_reason=PASSWORD_CHANGED`; thư `password_changed` 1; `audit_log` `password_changed {"revoked_sessions":1}` (không mật khẩu / băm) |
| 18 | PASS | sai mật khẩu hiện tại `422` `{field:current_password,code:WRONG_PASSWORD}`; mới = cũ `422 PASSWORD_SAME_AS_OLD`; yếu `422`; không JWT `401`; sau 5 lần sai, lần đúng kế → **`429 RATE_LIMITED`** `retry_after=326` |
| 20 | PASS | `/me/sessions`: 3 phiên, đúng 6 khoá `{id,current,device_label,ip_masked,created_at,last_used_at}`, phiên hiện tại `current=true` đứng đầu, không `ep_rt` / `refresh_hash` / `Mozilla` / `user_agent`; `ip_masked` dạng `a:b:*:*` (loopback `::1` hiện `0:0:*:*`) — IP đầy đủ không trả ra |
| 21 | PASS | phiên `sv.kha` không nằm trong danh sách của người khác; `?user_id=<uuid lạ>` bị bỏ qua (chỉ phiên của mình) |
| 23 | PASS (**IDOR chính**) | token `sv.gioi`: `DELETE /me/sessions/<phiên sv.kha>` → `404`, uuid lạ → `404` (**thân giống hệt** sau bỏ `trace_id`), `abc` → `404`; phiên `sv.kha` còn nguyên, refresh của `sv.kha` `200` |
| 24 | PASS | Admin `DELETE /me/sessions/<phiên sv.kha>` → `404`; `DELETE /me/sessions?user_id=<sv.kha>` → `200 {"revoked":1}` nhưng chỉ phiên **của chính Admin** (phiên `sv.kha` còn nguyên) — tham số `user_id` vô tác dụng |
| 25 | PASS | A xoá phiên B `204`; access cũ B `401 SESSION_REVOKED` sau 10 ms, `REVOKED_BY_USER`; `DELETE /me/sessions` `{revoked:3}` giữ phiên hiện tại; xoá phiên hiện tại `204`, cookie `ep_rt` bị xoá, access sau đó `401` |
| 27 | PASS | `/forgot-password`: 1 ô email + **1** nút `Gửi hướng dẫn`, sau gửi cùng câu AC1 bất kể email + `Gửi lại hướng dẫn (59 giây)`; `/reset-password?token=…`: `Mật khẩu mới`, `Nhập lại mật khẩu` (`password`/`new-password`), nêu chính sách, URL về `/reset-password` (token xoá), mật khẩu yếu → lỗi tại ô, thành công "Mật khẩu đã được đổi. Hãy đăng nhập lại." |
| 28 | PASS cấu hình / chưa kiểm log Caddy thật | `Referrer-Policy: no-referrer` + `no-store` ở `/reset-password`; `deploy/caddy/Caddyfile` có bộ lọc `request>uri regexp (token=\|/invite/)[A-Za-z0-9_-]+ ${1}REDACTED`. Stack QC không có Caddy: `docker compose logs caddy \| grep` chưa chạy — kiểm ở cổng P2 |
| 29 | PASS | `forgot page`, `reset page` trong Playwright pass |
| 30 | PASS | `/settings`: "Tài khoản và bảo mật" › Mật khẩu (3 ô, `Đổi mật khẩu` là **primary duy nhất**) › "Thiết bị đang đăng nhập"; hàng hiện tại "Chrome trên macOS · Thiết bị này" **không có menu**; hàng khác có menu `Đăng xuất thiết bị này`; `Đăng xuất mọi thiết bị khác` mở hộp thoại "Đăng xuất 2 thiết bị khác. Họ sẽ phải đăng nhập lại." (`Để sau` / `Đăng xuất`), sau xác nhận: "Không có thiết bị nào khác." |
| 31 | PASS | `/forgot-password`, `/reset-password`, `/register`, `/verify-email`, `/login` ở 375: `AUDIT` sạch, `TOUCH` `[]`, 0 từ kỹ thuật |
| 33 | PASS | không JWT: `GET /me/sessions`, `DELETE /me/sessions`, `POST /me/password` → `401`; 4 vai (STUDENT, TA, TEACHER, ADMIN) gọi `/me/sessions` `200` cho chính mình |
| 35 | PASS | `tokens/preview`: RESET hợp lệ `200 {valid,kind,expires_at}` (không email, không `full_name`/`role`); INVITE `200` kèm `full_name`, `role` (không email); `VERIFY_EMAIL` và kind rỗng `422`; sai loại / đã dùng → `410` (`invalid` / `used`); **preview không tiêu token** (đặt lại sau preview `200`) |
| 37 | PASS | thư `reset_password` và `password_changed` đúng mẫu SRS 6.5, liên kết `APP_PUBLIC_URL/reset-password?token=<43>`, không chứa mật khẩu |
| 39 | PASS | log gateway / worker: mật khẩu 0, email 0; `pg_dump` mật khẩu rõ 0 |
| 40 | PASS | `go vet`, `golangci-lint` 0 issues, `go test -race` (auth) ok; Playwright 221 pass (ghi chú flake ở `report-US-P2-03.md`) |

## Việc sau
QC chấm lại: TC-11 (`PASSWORD_COMMON`, `PASSWORD_CONTAINS_EMAIL`) và giới hạn IP ở P2-05; TC-28 (log Caddy) ở cổng P2 khi có Caddy. Gợi ý nhỏ: `ip_masked` của IPv6 loopback hiển thị `0:0:*:*` (nên `::*`?) — không FAIL.
