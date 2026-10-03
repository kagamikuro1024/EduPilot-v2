# QC test case — US-P2-04 (quên / đặt lại / đổi mật khẩu, danh sách và thu hồi thiết bị `/me/sessions`, xem trước token)
Nguồn: `docs/specs/FEAT-account-security/US.md` US-P2-04 AC1–AC12 + `SRS.md` 6.2, 6.4, 6.5 (thư `reset_password`, `password_changed`), 4.2. "Bạn tự kiểm" của P2: quên mật khẩu ở máy A làm máy B đăng xuất. **Trọng tâm tấn công:** IDOR `/me/sessions`, link đặt lại dùng hai lần / đua / hết hạn, đoán mã, token băm, mật khẩu ở thư / log, liệt kê tài khoản.

Tiền điều kiện chung: stack test; biến của `US.md`; hai cookie jar cho cùng một người (máy A, máy B) và một người thứ hai (`sv.kha`) để thử IDOR; `PW`, `NEWPW='Mat-khau-moi-2026'`. Công cụ: **S** `scripts/p204.sh`, **D** (DB/Mailpit/Redis), **A** (Chrome thật, hai `context`), **G** (`go test`, `$PW account.spec.ts`). Thiếu route → FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P204-01 | AC1 | 4 email: ACTIVE, PENDING_VERIFICATION, INVITED, DISABLED, + 1 không tồn tại | **S** `forgot-password` cho 5 email; `jq -S 'del(.trace_id)'`; đo thời gian 20 mẫu mỗi loại | **Cả 5** `202 {"message":"Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu."}` y hệt (cùng khoá, header); trung vị thời gian lệch ≤ 35 % |
| TC-P204-02 | AC1 | – | **D** `mail_count` cho từng email | Chỉ ACTIVE và PENDING_VERIFICATION nhận `reset_password`; INVITED / DISABLED / không tồn tại **0 thư** |
| TC-P204-03 | AC1 | – | **G** `-run 'TestForgotUniform\|TestForgotMailOnlyEligible\|TestForgotTimingEqualized'` | `ok` |
| TC-P204-04 | AC2 | thư `reset_password` | **S** `t=$(mail_token sv.kha@edupilot.local)`; `j -X POST $GW/api/v1/auth/reset-password -d "{\"token\":\"$t\",\"new_password\":\"$NEWPW\"}"`; đăng nhập bằng mật khẩu mới và cũ; kiểm phản hồi có access token? | `200`; mật khẩu **mới** đăng nhập được, **cũ** `401`; phản hồi **không** tự đăng nhập (không `access_token`, không `Set-Cookie ep_rt`); `used_at` có |
| TC-P204-05 | AC2 (**link dùng hai lần**) | – | **S** dùng lại cùng token | **`410 LINK_INVALID`** `details.reason="used"`; mật khẩu không đổi lần hai |
| TC-P204-06 | AC2 (đua) | token mới | **S** 50 `curl` song song cùng token với 50 mật khẩu khác nhau | **Đúng 1** `200`; 49 `410`; mật khẩu cuối cùng = của yêu cầu thắng (đăng nhập thử xác nhận) |
| TC-P204-07 | AC2 (thu hồi) | 2 phiên (máy A, B) đang đăng nhập | **S** reset; access cũ của A và B; `refresh` của A và B; `select revoked_reason from auth_sessions` | Mọi phiên `revoked_reason=PASSWORD_RESET`; access cũ `401 SESSION_REVOKED` **≤ 1 s** ở cả hai gateway; refresh `401 SESSION_REVOKED` |
| TC-P204-08 | AC2 | – | **D** `failed_logins`, `locked_until` trước-sau (đã khoá bằng 10 lần sai); `email_verified_at` khi trống | `failed_logins=0`, `locked_until=NULL`; nếu `email_verified_at` trống → `now()` và `status=ACTIVE`; thư `password_changed` đến |
| TC-P204-09 | AC2 (băm) | – | **D** `select count(*) from auth_tokens where token_hash='$t'`; `pg_dump \| grep -c "$t"`; `grep -c "$NEWPW"` ở DB, log, thư | `0`; `0`; mật khẩu mới chỉ ở dạng bcrypt (`grep` `0`); thư **không** chứa mật khẩu |
| TC-P204-10 | AC2 | – | **G** `-run 'TestResetSuccess\|TestResetRevokesAllSessions\|TestResetClearsLockout\|TestResetVerifiesEmail\|TestResetNoAutoLogin\|TestResetSingleUseRace'` | `ok` |
| TC-P204-11 | AC3 | – | **S** token `age_token`+30 phút; ngẫu nhiên 43 ký tự; mật khẩu yếu: `"abc"`, `"password1234"`, `"Edupilot2026"` chứa email, 129 ký tự | `410 LINK_INVALID` (`expired`/`invalid`) cùng khoá JSON; yếu → `422 VALIDATION_FAILED` `details` có `PASSWORD_TOO_SHORT`/`PASSWORD_COMMON`/`PASSWORD_CONTAINS_EMAIL`/`PASSWORD_TOO_LONG`; **token KHÔNG bị tiêu** (nhập lại mật khẩu mạnh với **cùng** token → 200); lỗi không đổi gì (`password_hash` không đổi) |
| TC-P204-12 | AC3 (**đoán mã**) | – | **S** 1.000 token ngẫu nhiên vào `reset-password` và `tokens/preview` từ một IP | Toàn `410`/`429`; **không** trúng; giới hạn 20 lần/phút/IP ở `preview` (`429 RATE_LIMITED` + `retry_after`); thời gian xử lý của token "đúng định dạng nhưng không có" ≈ token "sai định dạng" (không lộ qua thời gian) |
| TC-P204-13 | AC3 | – | **G** `-run 'TestResetExpired30m\|TestResetUnknownToken\|TestResetWeakPasswordKeepsToken'` | `ok` |
| TC-P204-14 | AC4 (máy A / máy B) | 2 `context` Chrome đăng nhập cùng tài khoản | **A** máy B mở `/threads`; ở máy A thực hiện quên mật khẩu (Mailpit) → đặt lại; máy B `page.reload()` | Máy B chuyển `/login` kèm "Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi." (`role="status"`); máy A cũng phải đăng nhập lại; cookie `ep_rt` hai máy vô hiệu |
| TC-P204-15 | AC4 | – | **S** hai jar curl: reset → cả hai `refresh` | Cả hai `401` |
| TC-P204-16 | AC4 | – | **G** `$PW account.spec.ts -g 'reset logs out other device'` (`@real`) | `rc=0` |
| TC-P204-17 | AC5 | đã đăng nhập 2 phiên (P1, P2) | **S** `POST /me/password {"current_password":"$PW","new_password":"$NEWPW"}` ở P1; sau đó `refresh` P1, P2; `audit_log`; thư | `200`; P1 `refresh` `200` (phiên hiện tại **giữ**); P2 `refresh` `401 SESSION_REVOKED` (`revoked_reason=PASSWORD_CHANGED`); thư `password_changed`; `audit_log` +1 dòng **không** chứa mật khẩu |
| TC-P204-18 | AC5 | – | **S** sai mật khẩu hiện tại; sai 5 lần / 10 phút; mật khẩu mới = cũ; yếu; không JWT | Sai → `422` `{field:"current_password",code:"WRONG_PASSWORD"}`; lần thứ 6 → `429 RATE_LIMITED` `retry_after`; bằng cũ → `422 PASSWORD_SAME_AS_OLD`; yếu → 422; không JWT → `401` |
| TC-P204-19 | AC5 | – | **G** `-run 'TestChangePassword\|TestChangePasswordWrongCurrent\|TestChangePasswordRateLimit\|TestChangePasswordKeepsCurrentSession\|TestChangePasswordSameAsOld'` | `ok` |
| TC-P204-20 | AC6 | 3 phiên của `sv.gioi` | **S** `curl -sk -H "$h" $GW/api/v1/me/sessions \| jq '.items[0] \| keys'`; đọc toàn thân; `grep -c 'ep_rt\|refresh_hash\|Mozilla'` | Đúng 6 khoá `{id, current, device_label, ip_masked, created_at, last_used_at}`; phiên hiện tại `current=true` đứng đầu; `device_label` kiểu "Chrome trên macOS"; `ip_masked` dạng `203.0.*.*` — **IP đầy đủ không trả ra**; **không** `refresh_hash`, **không** `user_agent` nguyên văn; sắp theo `last_used_at` giảm dần; ≤ 50 |
| TC-P204-21 | AC6 (chỉ của mình) | `sv.gioi` và `sv.kha` đều có phiên | **S** gọi bằng token `sv.gioi`; so `id` với `select id from auth_sessions where user_id=<sv.kha>` | Danh sách **không** chứa phiên của `sv.kha` |
| TC-P204-22 | AC6 | – | **G** `-run 'TestSessionsListShape\|TestSessionsListOnlyMine\|TestSessionsIPMasked'` | `ok` |
| TC-P204-23 | AC7 (**IDOR /me/sessions — tấn công chính**) | `SID_KHAC` = phiên của `sv.kha`; `SID_LA` = uuid ngẫu nhiên | **S** bằng token `sv.gioi`: `DELETE /me/sessions/$SID_KHAC`; `DELETE /me/sessions/$SID_LA`; `DELETE /me/sessions/abc`; rồi kiểm phiên `sv.kha` | `404 NOT_FOUND` **giống hệt** cho `SID_KHAC` và `SID_LA` (không lộ tồn tại; so thân + thời gian); `abc` → `400/422`/`404` nhất quán; phiên `sv.kha` **còn nguyên** (`revoked_at IS NULL`; refresh của `sv.kha` vẫn 200) |
| TC-P204-24 | AC7 (leo vai) | – | **S** Admin `DELETE /me/sessions/$SID_KHAC`; `DELETE /me/sessions?user_id=<sv.kha>`; thân `{"user_id":…}` | Admin cũng `404` (đường `/me` chỉ tác động phiên của chính JWT); không tham số `user_id` nào có tác dụng; phiên `sv.kha` còn |
| TC-P204-25 | AC7 | – | **S** `DELETE /me/sessions/{id của máy B}` bằng máy A; `DELETE /me/sessions` (mọi thiết bị khác); xoá phiên hiện tại | Máy B bị đăng xuất **≤ 1 s** (`REVOKED_BY_USER`); `DELETE /me/sessions` → `{revoked: n}` giữ phiên hiện tại; xoá phiên hiện tại = đăng xuất |
| TC-P204-26 | AC7 | – | **G** `-run 'TestRevokeOwnSession\|TestRevokeOthersSessionIs404\|TestRevokeAllOthers\|TestAdminCannotRevokeViaMe'` | `ok` |
| TC-P204-27 | AC8 | `/forgot-password`, `/reset-password` | **A** forgot: số ô, số nút primary, câu sau gửi (cho email có / không), nút gửi lại khoá 60 s; reset với token hợp lệ / không hợp lệ / sau thành công | Một ô email + **một** nút `Gửi hướng dẫn`; cùng câu AC1 bất kể email; reset hợp lệ: `Mật khẩu mới`, `Nhập lại mật khẩu` (`autocomplete="new-password"`, `type=password`), nêu chính sách; không hợp lệ: "Liên kết đã hết hạn hoặc đã được dùng." + `Yêu cầu liên kết mới`; thành công: "Mật khẩu đã được đổi. Hãy đăng nhập lại." + `Đăng nhập`; **token bị xoá khỏi URL** |
| TC-P204-28 | AC8 | – | **A** `Referrer-Policy` ở `/reset-password`; token trong `Referer` khi tải tài nguyên | `no-referrer`; token không rò qua Referer / log truy cập Caddy (`docker compose logs caddy \| grep -c "$t"` = `0` — nếu có, ghi lỗi bảo mật) |
| TC-P204-29 | AC8 | – | **G** `$PW account.spec.ts -g 'forgot page\|reset page'` | `rc=0` |
| TC-P204-30 | AC9 | `/settings` đã đăng nhập, 3 phiên | **A** phần "Mật khẩu" và "Thiết bị đang đăng nhập": nút primary; hàng "Thiết bị này"; menu `Đăng xuất thiết bị này`; `Đăng xuất mọi thiết bị khác` | `Đổi mật khẩu` là `primary`, `Đăng xuất mọi thiết bị khác` **không** primary; hàng hiện tại ghi "Thiết bị này"; `Đăng xuất thiết bị này` thực hiện ngay (không hộp thoại), dòng tĩnh "Đã đăng xuất thiết bị này" tại chỗ; "mọi thiết bị khác" mở `ConfirmIrreversible` nêu số ("Đăng xuất 2 thiết bị khác. Họ sẽ phải đăng nhập lại."); rỗng: "Không có thiết bị nào khác." |
| TC-P204-31 | AC9 | – | **A** chỉ bàn phím; 375 px; `bash scripts/ui-antipatterns.sh`; axe | Làm được chỉ bằng phím; `AUDIT`/`TOUCH` sạch; 0 `serious`; không từ kỹ thuật (không "session", "IP" trần: dùng "thiết bị") |
| TC-P204-32 | AC9 | – | **G** `$PW account.spec.ts -g 'settings security'` | `rc=0` |
| TC-P204-33 | AC10 | – | **S** `curl -sk -o /dev/null -w '%{http_code}\n'` không JWT: `GET /me/sessions`, `DELETE /me/sessions`, `POST /me/password`; công khai: `forgot-password`, `reset-password`, `tokens/preview`; mọi vai (STUDENT, TA, TEACHER, ADMIN) gọi `/me/sessions` | Không JWT → `401 UNAUTHENTICATED`; công khai không cần JWT; **4 vai** đều dùng được cho **chính mình**; không tham số `user_id` ở bất kỳ đường nào (thử `?user_id=`/thân → bỏ qua / 422) |
| TC-P204-34 | AC10 | – | **G** `-run 'TestMeEndpointsRequireJWT\|TestMeEndpointsNoUserIDParam\|TestAllRolesCanManageOwnSessions'` | `ok` |
| TC-P204-35 | AC11 | token `INVITE`, `RESET_PASSWORD` | **S** `POST /auth/tokens/preview` `{kind,token}`: hợp lệ cả hai; `VERIFY_EMAIL`; token đã dùng; sai `kind`; 21 lần / phút | `200 {valid:true, kind, full_name?, role?, expires_at}` — `full_name`,`role` **chỉ** `INVITE`; **không email**; `VERIFY_EMAIL` → `422`; đã dùng/hết hạn → `410 LINK_INVALID`; **không tiêu token** (dùng sau preview vẫn được); lần 21 → `429` |
| TC-P204-36 | AC11 | – | **G** `-run 'TestTokenPreviewValid\|TestTokenPreviewDoesNotConsume\|TestTokenPreviewNoEmail\|TestTokenPreviewRateLimit'` | `ok` |
| TC-P204-37 | AC12 | – | **D** đọc thư `reset_password` và `password_changed` (giờ VN, hạn 30 phút, hướng dẫn nếu không phải mình); `mail_text … \| grep -c "$NEWPW"` | Đúng lời văn SRS 6.5; liên kết `${APP_PUBLIC_URL}/reset-password?token=…`; **không** mật khẩu; giờ `Asia/Ho_Chi_Minh` |
| TC-P204-38 | AC12 | – | **G** `-tags integration -run 'TestResetMail\|TestPasswordChangedMail'` | `ok` |
| TC-P204-39 | tổng (log/PII) | – | **D** `docker compose logs gateway worker caddy \| grep -c "$NEWPW\|$PW"`; email trong log | `0` |
| TC-P204-40 | tổng | – | **S** `go vet && golangci-lint run && go test -race -count=1 ./internal/auth/...`; `$PW account.spec.ts` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Liệt kê tài khoản qua quên mật khẩu | 01, 02 |
| Link đặt lại dùng hai lần / đua / hết hạn / đoán mã | 05, 06, 11, 12 |
| Mật khẩu yếu làm cháy token | 11 |
| Phiên cũ sống sau khi đổi mật khẩu | 07, 14, 17 |
| IDOR xoá phiên người khác (cả Admin) | 23, 24 |
| Lộ IP đầy đủ / user-agent / refresh hash qua `/me/sessions` | 20 |
| Mật khẩu / token ở thư / log / DB | 09, 37, 39 |
| Token rò qua `Referer` / log Caddy | 28 |
| Đổi mật khẩu dò mật khẩu hiện tại | 18 |

## Câu hỏi cho BA / PM
- **Q-QC-P204-1** — TC-P204-12: "thời gian của token đúng định dạng nhưng không tồn tại ≈ token sai định dạng": US không yêu cầu cân bằng thời gian cho `reset-password`; QC chỉ **ghi** chênh lệch (không FAIL) trừ khi SRS nói. — *chờ xác nhận*.
- **Q-QC-P204-2** — TC-P204-28: log truy cập Caddy có thể ghi query-string `?token=` của GET `/reset-password?token=…` (trang frontend). QC coi đây là rủi ro bảo mật nếu có; cần cấu hình Caddy không log query. — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-account-security, APPROVED 2026-10-03).

Tổng: 40 TC.
