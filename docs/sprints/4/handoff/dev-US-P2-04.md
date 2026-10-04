# DEV handoff — US-P2-04 (quên / đặt lại / đổi mật khẩu, thiết bị đăng nhập)
Nhánh `sprint/4-p2`. Chưa kiểm trên stack compose thật (xem "Chưa làm").

## Làm gì
**Backend** (`internal/auth/passwords.go`, `policy.go`, mở rộng `Accounts`; handler ở `internal/httpapi/authhttp`)
- `POST /auth/forgot-password`: mọi email trả cùng 202; chỉ ACTIVE / PENDING_VERIFICATION xếp thư `reset_password` (token phát lúc gửi, thu hồi token cũ); mọi nhánh mất ≥ 80 ms (`forgotFloor`) để thời gian không lộ; 3 lần / giờ / băm email (Redis `ep:auth:forgot:*`, tính cả email không tồn tại).
- `POST /auth/reset-password`: một giao dịch = tiêu token (UPDATE nguyên tử) → chính sách → bcrypt → đổi mật khẩu + mở khoá + `email_verified_at`/ACTIVE nếu còn chờ → thu hồi MỌI phiên (`PASSWORD_RESET`) → xếp `password_changed`. Mật khẩu yếu ⇒ 422 và **giao dịch lùi** nên token chưa bị tiêu; không `Set-Cookie`, không access token. Sau commit đặt khoá thu hồi Redis theo từng `sid` (≤ 1 s).
- `POST /auth/tokens/preview`: RESET_PASSWORD | INVITE, không tiêu; `full_name`/`role` chỉ INVITE; không email; `VERIFY_EMAIL` hoặc kind lạ → 422; 20 lần/phút/IP (`ep:rl:auth:token:ip:{ip}:{phút}`).
- `POST /me/password` (JWT): sai mật khẩu hiện tại → 422 `WRONG_PASSWORD` (đếm Redis `ep:rl:auth:chgpw:{uid}:{10 phút}`; ≥ 5 lần sai ⇒ 429 kể cả khi nhập đúng); `PASSWORD_SAME_AS_OLD`; đổi + thu hồi phiên KHÁC (`PASSWORD_CHANGED`, giữ `sid` hiện tại) + `audit_log` (`after = {"revoked_sessions":n}`, không mật khẩu / băm) + thư `password_changed`, cùng một giao dịch.
- `GET /me/sessions` (≤ 50, hiện tại đứng đầu, `ip_masked` `a.b.*.*` / IPv6 `x:y:*:*`, không refresh hash / UA), `DELETE /me/sessions/{id}` (UPDATE `… AND user_id = $jwt` ⇒ phiên người khác, đã chết, id sai đều 404; xoá phiên hiện tại = đăng xuất + xoá cookie `ep_rt`), `DELETE /me/sessions` (`{revoked:n}`, giữ phiên hiện tại). Không có tham số `user_id` ở đâu.
- sqlc: `ResetUserPassword`, `ChangeUserPassword`, `RevokeUserSessions` (:many ⇒ id để đặt khoá Redis), `RevokeOwnSession`, `ListOwnSessions`. `authhttp.Handler.MountMe` nằm trong nhóm đã qua `auth.Middleware` (+ `WithRevocation`).
- `openapi.yaml` 0.6.0: +7 thao tác (tổng 31), schema mới; golden mới `auth/forgot-password`, `reset-password.410`, `tokens-preview.410/.422` (golden cũ KHÔNG đổi); kịch bản hợp đồng `passwordScenarios` phủ mọi status khai báo.
- Caddy: `log { format filter { request>uri regexp (token=|/invite/)[A-Za-z0-9_-]+ ${1}REDACTED } }` (AC13). `Referrer-Policy`/`no-store` đã có từ US-P2-03 (`next.config.ts`).

**Frontend**: `/forgot-password`, `/reset-password` (preview khi mở, `history.replaceState` xoá token, token chỉ còn trong `useRef`), `/settings` (mọi vai; `SecuritySettings` = `PasswordSection` 3 ô + `DevicesSection`), mục "Tài khoản và bảo mật" trong menu tài khoản (chỉ phiên thật), `nav.ts` ACCESS: `/settings` mọi vai, `/settings/llm` và `/settings/integrations` giữ teacher/admin.

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `TestForgotUniform` (5 trạng thái ⇒ thân giống hệt, không Set-Cookie), `TestForgotMailOnlyEligible`, `TestForgotTimingEqualized` (trung vị 15 lượt, lệch ≤ 35 %), `TestForgotPerEmailLimit` PASS |
| 2 | `TestResetSuccess` (mật khẩu cũ hết dùng, `used_at`, dùng lại ⇒ 410 `used`, thư `password_changed` không chứa mật khẩu), `TestResetRevokesAllSessions` (access cũ ⇒ 401 `SESSION_REVOKED` + `details.reason=password_reset`, refresh 401), `TestResetClearsLockout`, `TestResetVerifiesEmail`, `TestResetNoAutoLogin`, `TestResetSingleUseRace` (30 goroutine ⇒ 1×200) PASS |
| 3 | `TestResetExpired30m`, `TestResetUnknownToken` (kể cả dùng token XÁC MINH để đặt lại ⇒ `invalid`, token xác minh còn nguyên), `TestResetWeakPasswordKeepsToken` PASS. **Chỉ `PASSWORD_TOO_SHORT` / `PASSWORD_TOO_LONG` có mã ở đây**: `PASSWORD_COMMON` / `PASSWORD_CONTAINS_EMAIL` thuộc **US-P2-05** (cùng hàm `ValidatePasswordPolicy`, ngay đây) |
| 4 | Phần máy chủ: `TestResetRevokesAllSessions` (hai phiên A/B). Giao diện máy B chuyển `/login` + dòng thông báo: đã có từ US-P2-02 (`LoginForm` khoá `password_reset`/`password_changed`, e2e "refresh hỏng … phiên bị thu hồi"). `@real` hai-context: `test.skip` (cần stack) |
| 5 | `TestChangePassword` (+ audit_log), `…WrongCurrent` (`details[0]={field:current_password,code:WRONG_PASSWORD}`), `…RateLimit` (5 sai ⇒ 429 kể cả đúng; `retry_after` ≤ 600; sau 10 phút (đồng hồ giả) đổi được), `…KeepsCurrentSession`, `…SameAsOld`, 401 không JWT (`TestMeEndpointsRequireJWT`) PASS |
| 6 | `TestSessionsListShape` (đúng 6 khoá, hiện tại đứng đầu dù mới hơn, "Chrome trên macOS", không UA/refresh), `TestSessionsListOnlyMine`, `TestSessionsIPMasked` PASS. Giới hạn 50: bằng `LIMIT 50` trong SQL, chưa có test 51 phiên (nợ nhỏ) |
| 7 | `TestRevokeOwnSession` (≤ 1 s: access ⇒ 401 `revoked_by_user`; xoá phiên hiện tại ⇒ cookie `Max-Age=-1`), `TestRevokeOthersSessionIs404` (phiên B, uuid lạ, id sai định dạng ⇒ 404), `TestRevokeAllOthers`, `TestAdminCannotRevokeViaMe` PASS |
| 8 | `account.spec.ts` "forgot page", "reset page" (preview 1 lần, URL sạch token, hai ô `type=password` + `autocomplete=new-password`, lỗi tại ô giữ chữ, thành công, liên kết xấu / thiếu token), "reset page: Referrer-Policy no-referrer, no-store" PASS |
| 9 | `account.spec.ts` "settings security" ×2 (Thiết bị này, không menu ở hàng hiện tại, đăng xuất ngay không dialog + dòng tĩnh `Đã đăng xuất thiết bị này`, xác nhận "Đăng xuất 1 thiết bị khác. Họ sẽ phải đăng nhập lại." , rỗng, đổi mật khẩu 3 ô + lỗi tại ô, nút `Đổi mật khẩu` primary còn `Đăng xuất mọi thiết bị khác` thì không; khung xương / lỗi chuẩn + Thử lại; 375 px `AUDIT`/`TOUCH` sạch; không từ `session/token/refresh/jwt/bcrypt`) PASS. Hàng thiết bị là `<ul>` (không `DataTable`) vì spec cho phép "danh sách / hàng" |
| 10 | `TestMeEndpointsRequireJWT` (4 đường ⇒ 401), `TestMeEndpointsNoUserIDParam` (`?user_id=` bị bỏ qua, thân có `user_id` ⇒ 422), `TestAllRolesCanManageOwnSessions` (4 vai) PASS |
| 11 | `TestTokenPreviewValid` (RESET: không `full_name`/`role`; INVITE: có; VERIFY_EMAIL/kind rỗng 422; sai loại ⇒ `invalid`; hết hạn ⇒ `expired`), `…DoesNotConsume`, `…NoEmail`, `…RateLimit` (20 rồi 429 + `Retry-After`; sau 61 s lại được) PASS |
| 12 | `go test -tags integration ./internal/auth -run 'TestResetMail|TestPasswordChangedMail'` PASS với Mailpit thật (liên kết 43 ký tự, một tham số, "30 phút", giờ Việt Nam `HH:mm dd/MM/yyyy`, liên kết `/forgot-password`, không chứa mật khẩu) |
| 13 | Caddy: `caddy validate` Valid; kiểm riêng bằng Caddy 2.11 thật: 3 URL có token 43 ký tự ⇒ log `"uri":"/verify-email?token=REDACTED&x=1"`, `"/invite/REDACTED"`, `"/reset-password?a=1&token=REDACTED"`; `grep -cE 'token=[A-Za-z0-9_-]{43}\|/invite/[A-Za-z0-9_-]{43}'` = 0. `grep -n 'REDACTED\|filter' deploy/caddy/Caddyfile \| wc -l` = 3 (≥ 1). Log của frontend (Next) không ghi request. `Referrer-Policy` kiểm trên `next start` (e2e), chưa qua Caddy thật |

## Lệch spec / nợ (cho PM)
- **Hợp đồng `password_changed.At`** mã hoá RFC3339 trong payload thư rồi consumer đổi sang giờ Việt Nam (đã có từ US-P2-01).
- **Giới hạn IP** của `forgot-password` (5/giờ/IP), `reset-password`, `register`… vẫn là **US-P2-05**; ở đây chỉ có các giới hạn mà AC của story này nêu (preview 20/phút/IP, đổi mật khẩu 5 sai/10 phút, forgot 3/giờ/email).
- `TestForgotTimingEqualized` dựa vào sàn 80 ms thật (sleep); không dùng đồng hồ giả.
- Hàng thiết bị "Lần cuối 09:20": hiển thị giờ Việt Nam; qua ngày thêm " · dd/MM".
- `DELETE /me/sessions` với token dev (không `sid`): thu hồi mọi phiên (không có phiên "hiện tại" để giữ).
- Chưa làm: `@real` (hai context, Mailpit); không chạy tay trên `pnpm dev` / Caddy thật ngoài phép thử log ở trên; `PASSWORD_COMMON` / `PASSWORD_CONTAINS_EMAIL` (US-P2-05).
