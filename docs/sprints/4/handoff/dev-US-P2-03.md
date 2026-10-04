# DEV handoff — US-P2-03 (đăng ký STUDENT, xác minh email, gửi lại)
Nhánh `sprint/4-p2`. Chưa kiểm trên stack compose thật (xem "Chưa làm").

## Làm gì
**Backend**
- `internal/auth/accounts.go`: `Accounts.Register` (bcrypt LUÔN chạy trước mọi nhánh ⇒ cân bằng thời gian; `InsertPendingStudent … ON CONFLICT (email) DO NOTHING` nên đăng ký đua nhau không huỷ giao dịch; email mới ⇒ `verify_email`; ACTIVE/PENDING ⇒ `email_exists`; INVITED TEACHER/TA ⇒ `invite_staff` (tên người mời lấy từ `auth_tokens.created_by` gần nhất); DISABLED ⇒ không gửi), `VerifyEmail` (`ConsumeAuthToken` nguyên tử, `MarkEmailVerified`, `PromoteUnverifiedRosterEnrollments`; lỗi `LinkError{expired|used|invalid}` phân loại trong CÙNG transaction), `ResendVerification` (khoá Redis `ep:auth:resend:verify:{emailhash32}` SET NX 60 s theo băm email **kể cả email không tồn tại**; đã xác minh / không có / không chờ ⇒ 202 không thư), `ValidatePasswordPolicy` (mới có độ dài ≥ 10 ký tự, ≤ 72 byte — US-P2-05 mở rộng ngay trong hàm này).
- `auth` không import `mail` (consumer thư gọi `auth.Tokens` ⇒ vòng): `Accounts` nhận `MailQueue` (hàm) do `httpapi.queueMail` nối tới `mail.Enqueue`, cùng transaction.
- `authhttp`: `POST /auth/register` (thân dùng `httpx.DecodeJSON` ⇒ trường lạ `role/status/email_verified_at/...` ⇒ 422), `/auth/verify-email` (410 `LINK_INVALID` + `details.reason`), `/auth/resend-verification` (thân tuỳ chọn; có Bearer hợp lệ thì dùng email của chính người đó, chỉ để biết gửi tới đâu — không cấp quyền).
- Config `AUTH_RESEND_SECONDS` (60). `openapi.yaml` 0.5.0: +3 thao tác (tổng 24), `components.responses.Gone`; golden `auth/register*.json`, `verify-email.410.json`, `resend-verification*.json` (golden login / refresh KHÔNG đổi).
- Query mới trong `internal/store/queries/auth.sql`; MSSV chỉ được GHI (`InsertPendingStudent`), không nằm trong điều kiện nào.

**Frontend**
- `/register` (`RegisterForm`: 4 ô, chú thích MSSV, `autocomplete="new-password"`, lỗi từng ô `aria-invalid`, màn "Kiểm tra email của bạn" + nút `Gửi lại thư` khoá đếm ngược 60 s, `RATE_LIMITED` ⇒ đếm ngược theo `retry_after`), `/verify-email` (`VerifyEmail`: gọi MỘT lần — ref chặn StrictMode/tải lại —, `history.replaceState` xoá token khỏi URL ngay, 4 trạng thái; hết hạn / không dùng được có ô email để nhận thư mới). `next.config.ts` `headers()`: `Referrer-Policy: no-referrer` + `Cache-Control: no-store` cho `/verify-email`, `/reset-password`, `/invite/*`.

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `TestRegisterCreatesStudentOnly` (STUDENT/PENDING_VERIFICATION/chưa xác minh/băm bcrypt), `TestRegisterRejectsRoleField` (5 trường), `TestRegisterRejectsUnknownFields` PASS |
| 2 | `TestRegisterUniformResponse` (mới / ACTIVE / DISABLED / PENDING / INVITED ⇒ 5 thân giống hệt, không bản ghi thứ hai), `…ExistingSendsEmailExists`, `…InvitedResendsInvite` (có liên kết `/invite/<43>` qua consumer thật), `…DisabledSendsNothing`, `…TimingEqualized` (bcrypt cost 10, trung vị 20: lệch ≤ 35 %) PASS |
| 3 | `TestSelfDeclaredStudentCodeLinksNothing` (B ACTIVE trong lớp với MSSV X; kẻ giả đăng ký "x" chữ thường, xác minh, đăng nhập ⇒ 0 enrollment, bản ghi của B nguyên vẹn), `TestStudentCodeFormat` (9 ca), `TestNoQueryLinksByStudentCode` (quét `queries/*.sql`, danh sách trắng SRS 4.2.5) PASS |
| 4 | `TestVerifyEmail` (200, ACTIVE, `used_at`, DB không có bản rõ, lần hai 410 `used`), `TestVerifyEmailSingleUseRace` (50 goroutine ⇒ 1×200 + 49×410) PASS. Lỗi tìm được và sửa: tra lý do trong lúc giữ transaction bằng `pool` làm cạn pool (deadlock 30 s) ⇒ giờ dùng cùng transaction |
| 5 | `TestVerifyEmailExpired24h` (đồng hồ giả), `…Unknown` (3 token lạ), `…Replaced` (token bị thay ⇒ `invalid`, tài khoản chưa đổi) PASS |
| 6 | `TestResendThrottle60s` (202 rồi 429, `retry_after` ∈ [1,60]), `TestResendRevokesOld`, `TestResendVerifiedNoMail`, `TestResendUniform` (+ email không tồn tại cũng bị giới hạn), `TestResendWithBearerNoBody` PASS (cửa sổ rút gọn `AUTH_RESEND_SECONDS=1` cho ca "sau 60 giây gửi lại được") |
| 7 | `TestUnverifiedCanLogin`, `TestVerifyPromotesRosterPending` PASS. **`TestUnverifiedCannotJoin` (403 `EMAIL_NOT_VERIFIED` ở `/courses/join*`) chờ US-P2-09** — endpoint chưa tồn tại; mã `EMAIL_NOT_VERIFIED` đã có sẵn ở `apierr` / openapi enum |
| 8 | `TestRegisterValidationTable` 22 ca (`"Nguyễn Văn Ặ"`, email 255 ký tự, `a@b`, `a b@c.d`, hai `@`, điều khiển, tên 101 ký tự, mật khẩu 9/73 byte…) + 413 khi thân > `MAX_BODY_BYTES` PASS |
| 9 | `go test -tags integration ./internal/auth -run TestRegisterSendsVerifyMail` PASS (Mailpit thật: 1 thư, tiêu đề đúng, 1 tham số `token`, không chứa mật khẩu / MSSV, không còn `{{`) |
| 10 | `account.spec.ts` "register page" (nhãn, chú thích, autocomplete, 1 primary, lỗi từng ô giữ chữ đã gõ, màn Kiểm tra email, đếm ngược), "register 375 px" (`AUDIT`/`TOUCH` sạch), "verify page" (1 lần gọi, URL sạch token, ok / used / expired + gửi lại / thiếu token, `Referrer-Policy: no-referrer` + `no-store` qua `request.get`) PASS |
| 11 | `TestPublicEndpointsIgnoreJWTRole`, `TestNoSelfRolePatch` PASS. `PATCH /me/profile` chưa có (US-P2-07): test hiện khẳng định mọi đường vai trò thử đều ≥ 400; US-P2-07 phải giữ nó đỏ nếu thêm trường `role` |
| 12 | `@real` "register verify login": `test.skip` (cần stack Go + Mailpit). Phần tương đương chạy được bằng Go: đăng ký → consumer thật → xác minh → đăng nhập (`TestSelfDeclaredStudentCodeLinksNothing`, `TestUnverifiedCanLogin`) |

## Lệch spec / nợ
- **INVITED sinh viên** (từ import roster, US-P2-10) đăng ký lại: hiện KHÔNG gửi thư (cần tên lớp / giảng viên cho `invite_student`); US-P2-10 nối vào `mailExisting`. Ghi trong mã (`ponytail:`).
- Giới hạn IP của `register` / `verify-email` (5/giờ, 20/phút) là **US-P2-05**; mới có giới hạn gửi lại 60 s.
- Cổng nhập ở `/verify-email` khi hết hạn bắt người dùng nhập email (trang công khai không biết email); nếu đã đăng nhập thì `resend-verification` không thân dùng được qua API nhưng giao diện chưa dùng đường đó.
- Chưa chạy tay trên stack compose (`pnpm dev`) và chưa kiểm `curl -I https://localhost/verify-email` qua Caddy; header kiểm trên `next start`.

## Kết quả cổng (đã chạy)
- `make -C backend-go lint sqlc-check`: 0 issues ×3, `sqlc diff` rc=0. `make test`: mọi gói ok (`-race`; `internal/auth` 88 s). `internal/contract` đỏ lần đầu vì `BCRYPT_COST` mặc định 12 + `REQUEST_TIMEOUT=1s` dưới `-race` làm `register` 500 → đặt `BCRYPT_COST=4` cho rig hợp đồng (cấu hình test, không nới assertion) → xanh cả hai bản dựng (mặc định / `testroutes`).
- Frontend: eslint + tsc sạch; `ui-antipatterns` 0 ✗; Playwright (`build:gate`, cổng 3320/3322) **228 passed / 0 failed**.
