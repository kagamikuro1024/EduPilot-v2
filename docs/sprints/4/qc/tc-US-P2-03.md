# QC test case — US-P2-03 (tự đăng ký STUDENT, xác minh email, gửi lại 60 s, chống mạo danh MSSV)
Nguồn: `docs/specs/FEAT-account-security/US.md` US-P2-03 AC1–AC12 + `SRS.md` 4.2 (danh sách trắng truy vấn dùng MSSV ≤ 3), 6.5 (mẫu `verify_email`, `email_exists`), FLOWS F1 ("QUY TẮC AN TOÀN SỐ 1"). **Trọng tâm tấn công:** mạo danh MSSV, nâng quyền qua trường lạ, liệt kê email, link dùng hai lần / đua, đoán mã, token ở DB.

Tiền điều kiện chung: stack test (Mailpit, 2 gateway, worker); biến của `US.md`; `PW`; SV mẫu `sv.gioi@edupilot.local` (có MSSV trong danh sách lớp mẫu), lớp mẫu `INT1006` (có danh sách MSSV `20229002`…). QC dùng email `qc-<n>@example.test` mới mỗi TC. Công cụ: **S** `scripts/p203.sh`, **D** (DB/Mailpit), **A** (Chrome thật), **G** (`go test`, `$PW account.spec.ts`). Thiếu route → FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P203-01 | AC1 | – | **S** `j -X POST $GW/api/v1/auth/register -d '{"email":"qc-1@example.test","password":"'$PW'","full_name":"QC Một"}'`; `$PSQL -c "select role,status,email_verified_at is null,left(password_hash,4) from users where email='qc-1@example.test'"` | `202`; `STUDENT\|PENDING_VERIFICATION\|t\|$2a$`/`$2b$` (bcrypt); không `user_id` trong phản hồi |
| TC-P203-02 | AC1 (**nâng quyền**) | – | **S** đăng ký với từng thân: `+"role":"TEACHER"`, `"role":"ADMIN"`, `"status":"ACTIVE"`, `"email_verified_at":"2020-01-01T00:00:00Z"`, `"is_admin":true`, `"id":"<uuid>"`, `"password_hash":"x"`, trường lồng `{"profile":{"role":"ADMIN"}}`, khoá trùng `"role":"STUDENT","role":"ADMIN"`; đếm `users` | Mỗi cái `422` (trường lạ **bị từ chối**, không bị bỏ qua); **0** dòng `users` mới; không có đường công khai tạo TEACHER/TA/ADMIN (`select count(*) from users where role<>'STUDENT' and created_at>…` = 0) |
| TC-P203-03 | AC1 | – | **G** `-run 'TestRegisterCreatesStudentOnly\|TestRegisterRejectsRoleField\|TestRegisterRejectsUnknownFields'` | `ok` |
| TC-P203-04 | AC2 (**liệt kê email**) | các email ở 5 trạng thái: mới, ACTIVE, PENDING_VERIFICATION, INVITED, DISABLED | **S** đăng ký lần lượt; so `jq -S` thân + mã + header + thời gian | Cả 5 trả **y hệt** `202 {"message":"Nếu email này dùng được, chúng tôi đã gửi thư xác nhận. Kiểm tra hộp thư của bạn."}`; cùng khoá JSON; trung vị thời gian 20 mẫu lệch ≤ **35 %** giữa "mới" và "đã có"; không `user_id` |
| TC-P203-05 | AC2 | – | **D** `mail_count` + chủ đề thư mỗi trạng thái; `select count(*) from users where email=…` | Mới → thư `verify_email`; đã có (ACTIVE / PENDING) → thư `email_exists`; INVITED → gửi lại `invite_student`/`invite_staff` (không phải thư thường); DISABLED → **không thư**; không bản ghi `users` thứ hai (count=1) |
| TC-P203-06 | AC2 | – | **G** `-run 'TestRegisterUniformResponse\|TestRegisterExistingSendsEmailExists\|TestRegisterInvitedResendsInvite\|TestRegisterDisabledSendsNothing\|TestRegisterTimingEqualized'` | `ok` |
| TC-P203-07 | AC3 (**mạo danh MSSV — tấn công chính**) | `sv.gioi` đã vào lớp INT1006 với MSSV `20229002` (hoặc MSSV nằm trong danh sách lớp chưa gắn tài khoản) | **S** tài khoản mới `qc-attacker@example.test` đăng ký với `"student_code":"20229002"`; xác minh email (`mail_token`); đăng nhập; `GET /api/v1/me/courses`; `GET /api/v1/courses/<id lớp INT1006>`; `GET /api/v1/courses/<id>/members`; `POST /courses/join/preview` không mã | `enrollments` **không có dòng nào** cho attacker (`select count(*) from enrollments where user_id=<attacker>` = 0); `me/courses` **rỗng**; các route lớp `403`; **không** thấy điểm / tên / dữ liệu của `sv.gioi` ở bất kỳ API nào; `users.student_code` của attacker lưu `20229002` nhưng không có tác dụng |
| TC-P203-08 | AC3 (tấn công biến thể) | – | **S** thử: MSSV viết thường / có khoảng trắng / chữ hoa-thường lẫn; MSSV của người đã **vào lớp** (đã có tài khoản); đăng ký **trước** khi roster được nạp rồi nạp roster chứa MSSV đó (US-P2-10) | Vẫn **không** nối tài khoản vào lớp qua MSSV; roster nạp sau chỉ tạo bản ghi `PENDING` theo **email đã xác minh** (không theo MSSV); attacker không thành thành viên |
| TC-P203-09 | AC3 (đóng cổng ở mã) | repo | **S** `grep -rnE "student_code\s*=\s*\\\$\|student_code_snapshot\s*=\s*\\\$" backend-go/internal --include=*.sql --include=*.go \| grep -v _test.go`; đối chiếu với danh sách trắng ≤ 3 ở SRS 4.2; `grep -rn 'WHERE.*student_code' backend-go/internal/store/queries` | Các dòng chỉ thuộc danh sách SRS 4.2 (≤ 3, dùng cho chống trùng MSSV); **không** truy vấn nào dùng MSSV để mở dữ liệu / nối tài khoản |
| TC-P203-10 | AC3 | – | **S** gieo một truy vấn dùng MSSV để nối (tệp `.sql` tạm) → `go test -run TestNoQueryLinksByStudentCode`; xoá | Test **đỏ**; xoá → xanh; `git status` sạch |
| TC-P203-11 | AC3 (định dạng) | – | **S** `student_code`: `"AB12"` (ngắn), `"ABCDEFGHIJKLMNOP"` (16), `"2022 9002"`, `"2022-9002"`, `"<script>"`, `"abc123456"` (hợp lệ), `"　"` | Sai → `422 VALIDATION_FAILED`; hợp lệ → lưu **chữ hoa** (`ABC123456`); `^[A-Za-z0-9]{6,15}$` |
| TC-P203-12 | AC3 | – | **G** `-run 'TestSelfDeclaredStudentCodeLinksNothing\|TestStudentCodeFormat\|TestNoQueryLinksByStudentCode'` | `ok` |
| TC-P203-13 | AC4 | thư `verify_email` | **S** `t=$(mail_token qc-1@example.test); j -X POST $GW/api/v1/auth/verify-email -d "{\"token\":\"$t\"}"`; lần hai | `200`; `email_verified_at` có, `status=ACTIVE`; `auth_tokens.used_at` có; lần hai → **`410 LINK_INVALID`** `details.reason="used"` (**link dùng hai lần**) |
| TC-P203-14 | AC4 (đua) | token chưa dùng | **S** 50 `curl` song song cùng token (`xargs -P50`) | **Đúng 1** `200`, 49 `410`; `users` chỉ chuyển trạng thái một lần |
| TC-P203-15 | AC4 (băm) | – | **D** `select count(*) from auth_tokens where token_hash = '$t'` và `pg_dump \| grep -c "$t"` | `0`; `0` (chỉ băm) |
| TC-P203-16 | AC4 | – | **G** `-run 'TestVerifyEmail\|TestVerifyEmailSingleUseRace'` | `ok` |
| TC-P203-17 | AC5 | – | **S** token `age_token` (hết 24 h); token ngẫu nhiên 43 ký tự; token cũ sau khi gửi lại; token sai độ dài / ký tự lạ / rỗng / JSON thiếu trường | `410 LINK_INVALID` với `reason` lần lượt `expired`, `invalid`, `invalid`; thông báo người dùng giống nhau; sai độ dài / rỗng → `422`/`410` nhất quán; **không** đổi trạng thái tài khoản; mọi 410 cùng khoá JSON |
| TC-P203-18 | AC5 (**đoán mã**) | – | **S** 1.000 token ngẫu nhiên 43 ký tự gửi `verify-email` liên tiếp từ một IP; rồi 1.000 từ nhiều IP giả (`X-Forwarded-For`) | Tất cả `410 invalid` (không trúng); **giới hạn tốc độ** theo IP kích hoạt (`429 RATE_LIMITED` `retry_after`) — xem US-P2-05; không `X-Forwarded-For` giả vượt giới hạn (Caddy đặt IP thật); entropy: 32 byte → xác suất đoán bỏ qua |
| TC-P203-19 | AC5 | – | **G** `-run 'TestVerifyEmailExpired24h\|TestVerifyEmailUnknown\|TestVerifyEmailReplaced'` | `ok` |
| TC-P203-20 | AC6 | tài khoản PENDING | **S** `for i in 1 2; do j -X POST $GW/api/v1/auth/resend-verification -d '{"email":"qc-1@example.test"}' \| tail -1; done`; `mail_count`; token cũ | `202`, `429` (`retry_after` ∈ [1,60]); `mail_count` tăng đúng **1**; token cũ **thu hồi** (dùng cũ → `410`); sau 60 s gửi lại được |
| TC-P203-21 | AC6 (không lộ tồn tại) | – | **S** gửi lại cho email **không tồn tại** hai lần; cho email đã xác minh | Mẫu 202 / 429 **giống** email có thật (giới hạn theo băm email bất kể tồn tại); đã xác minh → `202` nhưng **không** gửi thư; không tạo `users` |
| TC-P203-22 | AC6 | – | **G** `-run 'TestResendThrottle60s\|TestResendRevokesOld\|TestResendVerifiedNoMail\|TestResendUniform'` | `ok` |
| TC-P203-23 | AC7 | tài khoản PENDING | **S** đăng nhập (`user.email_verified=false`); `GET /me/courses`; `POST /courses/join/preview`; `POST /courses/join` | Đăng nhập `200`; `me/courses` `200` rỗng; join/preview `403 EMAIL_NOT_VERIFIED`; sau xác minh dùng được |
| TC-P203-24 | AC7 | roster có email này `PENDING` + `warning=EMAIL_UNVERIFIED` | **D** xác minh email rồi `select status from enrollments` | `PENDING` → `ACTIVE` (đẩy enrollment); chỉ cho **email đã xác minh** trùng |
| TC-P203-25 | AC7 | – | **G** `-run 'TestUnverifiedCanLogin\|TestUnverifiedCannotJoin\|TestVerifyPromotesRosterPending'` | `ok` |
| TC-P203-26 | AC8 | – | **S** bảng ≥ 20 đầu vào: email `a@b`, `a b@c.d`, `a@@b.c`, 255 ký tự, `a+tag@x.vn` (hợp lệ), email có `\n`, `\u0000`, chữ hoa (`A@B.VN` → lưu chữ thường), tên `"Nguyễn Văn Ặ"` (hợp lệ), tên khoảng trắng, tên 101 ký tự, tên có điều khiển, thân 2 MB | Sai → `422 VALIDATION_FAILED` liệt kê `{field, code, message}`; thân > `MAX_BODY_BYTES` → `413`; chữ Việt có dấu **giữ nguyên**; email chuẩn hoá (cắt, chữ thường) |
| TC-P203-27 | AC8 (injection) | – | **S** tên `<script>alert(1)</script>`, `'; drop table users;--`, `{{7*7}}`, `${jndi:ldap://x}`; sau đó xem thư và `users.full_name` | Lưu nguyên văn (không thực thi); trong thư HTML bị thoát; không lỗi 500; bảng `users` còn |
| TC-P203-28 | AC8 | – | **G** `-run TestRegisterValidationTable -v` | `ok`; ≥ 20 ca |
| TC-P203-29 | AC9 | – | **D** đăng ký; đọc `$MP/search?query=to:...`; tiêu đề; liên kết; `grep -cE "$PW\|20229002"` | Đúng **1** thư; tiêu đề "Xác minh email EduPilot của bạn"; liên kết `${APP_PUBLIC_URL}/verify-email?token=<43 ký tự>` (đúng một tham số); có chữ thuần + HTML; **không** mật khẩu / MSSV |
| TC-P203-30 | AC9 | – | **G** `-tags integration -run TestRegisterSendsVerifyMail -v` | `ok` |
| TC-P203-31 | AC10 | `/register` | **A** đọc form: nhãn, `autocomplete="new-password"`, gợi ý chính sách, chú thích MSSV "Chỉ để giảng viên đối chiếu; không dùng để vào lớp.", số nút `primary`; điền thiếu / sai; gửi | Đủ ô; **một** nút chính `Tạo tài khoản`; lỗi dưới từng ô (`aria-invalid`); gửi xong: màn "Kiểm tra email" (đúng câu AC2 **bất kể** email) + `Gửi lại thư` khoá đếm ngược 60 s |
| TC-P203-32 | AC10 | – | **A** mở `/verify-email?token=<hợp lệ>`; đọc `location.search` sau; `curl -sk -I $GW/verify-email \| grep -i referrer-policy` | Xác minh tự động; token **bị xoá khỏi URL** (`history.replaceState`); `Referrer-Policy: no-referrer`; token không rò qua `Referer` khi tải tài nguyên ngoài (xem Network) |
| TC-P203-33 | AC10 | 375 | **A** `AUDIT_SRC`, `TOUCH_SRC`, axe ở `/register`, `/verify-email` | Sạch; 0 `serious`; không từ kỹ thuật |
| TC-P203-34 | AC10 | – | **G** `$PW account.spec.ts -g 'register page\|verify page'` | `rc=0` |
| TC-P203-35 | AC11 | – | **S** `register` / `verify-email` / `resend` kèm `Authorization: Bearer <JWT ADMIN>` với `role` trong thân; `PATCH /me/profile {"role":"ADMIN"}` bằng token SV | JWT **không** nâng quyền (vẫn chỉ STUDENT); `PATCH` `422`; không đường PATCH vai trò cho chính mình |
| TC-P203-36 | AC11 | – | **G** `-run 'TestPublicEndpointsIgnoreJWTRole\|TestNoSelfRolePatch'` | `ok` |
| TC-P203-37 | AC12 | `https://localhost` | **A** luồng đầu cuối: đăng ký → Mailpit → xác minh → đăng nhập → `/` (ô nhập mã lớp); nhánh lỗi: dùng lại liên kết, hết hạn, gửi lại sau 60 s | Đi trọn; từng bước ghi `test.step`; `$PW account.spec.ts -g 'register verify login'` `rc=0` |
| TC-P203-38 | tổng (log/PII) | – | **D** `docker compose logs gateway worker \| grep -c 'qc-1@example.test'` và MSSV / mật khẩu `PW` | `0` (không log email / MSSV / mật khẩu); `grep -c "$PW"` = `0` ở DB (chỉ bcrypt) |
| TC-P203-39 | tổng | – | **S** `go vet ./... && golangci-lint run && go test -race -count=1 ./internal/auth/... ./internal/mail/...` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Mạo danh MSSV để vào lớp / thấy dữ liệu người khác | 07–10 |
| Nâng quyền qua trường lạ / JWT trong thân | 02, 35 |
| Liệt kê email đã đăng ký (thư, mã, thời gian) | 04, 05, 21 |
| Link xác minh dùng hai lần / đua 50 luồng | 13, 14 |
| Đoán token | 18 |
| Token / mật khẩu / MSSV ở DB / log | 15, 38 |
| Chưa xác minh mà vào lớp | 23 |
| XSS / injection ở tên | 27 |
| Token rò qua `Referer` | 32 |

## Câu hỏi cho BA / PM
- **Q-QC-P203-1** — TC-P203-08: kịch bản "đăng ký trước, roster nạp sau, MSSV trùng" — SRS nối theo **email đã xác minh** (US-P2-10). QC giả định không nối theo MSSV; nêu rõ nếu SRS khác. — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Xác nhận: nối chỉ bằng **email đã xác minh** (US-P2-10 AC3); MSSV không bao giờ là khoá nối. Kịch bản "đăng ký trước, roster nạp sau, MSSV trùng, email khác" → không nối; chỉ nối khi email trùng và đã xác minh.
- **Q-QC-P203-2** — TC-P203-18 đo giới hạn IP: cần Caddy đặt `X-Forwarded-For` thật; QC giả định gateway tin header chỉ từ Caddy. — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Xác nhận: Caddy đặt `X-Forwarded-For`; gateway chỉ tin header từ `TRUSTED_PROXY_CIDRS` (mặc định gồm dải mạng compose) — ghi ở SRS 8.1 (v1.1).

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-account-security, APPROVED 2026-10-03).

Tổng: 39 TC.
