# QC test case — US-P2-06 (Admin mời giảng viên / TA, khoá / mở khoá, đổi vai, `/admin/users`, `/invite/[token]`, `gateway admin create`)
Nguồn: `docs/specs/FEAT-account-security/US.md` US-P2-06 AC1–AC13 + `SRS.md` 6.5 (`invite_staff`), 7.4 (mock `/admin/users` để đối chiếu), PRD M0 ("Admin KHÔNG thêm sinh viên vào lớp"). **Trọng tâm tấn công:** leo thang vai (tạo ADMIN / STUDENT), Admin biết / đặt mật khẩu, link mời dùng hai lần / đua / hết hạn, đoán mã, token băm, IDOR / phân quyền `/admin/*`, khoá / mở khoá.

Tiền điều kiện chung: stack test; biến của `US.md`; `A=$(bearer admin@edupilot.local)`; email thử `gv.moi-<n>@example.test`; Mailpit; `PW`. Công cụ: **S** `scripts/p206.sh`, **D**, **A** (Chrome thật), **G**. Thiếu route → FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P206-01 | AC1 | Admin | **S** `j -X POST $GW/api/v1/admin/users -H "$A" -H "Idempotency-Key: $(idem)" -d '{"email":"gv.moi-1@example.test","full_name":"Giảng Viên Mới","role":"TEACHER"}'`; DB; thư | `201 {id,email,full_name,role,status:"INVITED",version}` — **không** có token, mật khẩu, hash; `users.password_hash IS NULL`; **1** token `INVITE` hạn **72 giờ** (`expires_at-created_at=72h`, chỉ băm); `mail_count`=1 thư `invite_staff` (người mời, vai bằng chữ Việt, hạn, liên kết `${APP_PUBLIC_URL}/invite/<43 ký tự>`); `audit_log` +1 dòng **không** chứa token |
| TC-P206-02 | AC1 (**leo thang vai**) | Admin | **S** `role` = `STUDENT`, `ADMIN`, `admin`, `Teacher`, `""`, `null`, `SUPERUSER`, mảng `["ADMIN"]`; thêm trường `status:"ACTIVE"`, `password:"x"`, `email_verified_at` | Mỗi cái `422` (chỉ TEACHER/TA); **0** dòng `users` mới; trường lạ bị từ chối |
| TC-P206-03 | AC1 | – | **S** email trùng; email trùng khác hoa/thường; lặp cùng `Idempotency-Key` + cùng thân; thiếu `Idempotency-Key` | `409 CONFLICT` `details.field="email"`; lặp key → **1** người dùng, **1** thư, `Idempotent-Replayed: true`; thiếu key → `422 IDEMPOTENCY_KEY_REQUIRED` |
| TC-P206-04 | AC1 | – | **G** `-run 'TestInviteCreate\|TestInviteRejectsStudentAndAdmin\|TestInviteDuplicateEmail\|TestInviteIdempotent\|TestInviteAudit'` | `ok` |
| TC-P206-05 | AC2 | thư mời | **S** `t=$(mail_token gv.moi-1@example.test); j -X POST $GW/api/v1/auth/accept-invite -d "{\"token\":\"$t\",\"password\":\"$PW\"}" \| head -1 \| jq -r '.user.role,.user.status'`; cookie; DB | `TEACHER`, `ACTIVE`; trả như đăng nhập (access token + cookie `ep_rt`, **phiên mới**); `email_verified_at=now()`; `used_at` có; mật khẩu qua chính sách (yếu → 422, **không tiêu** token) |
| TC-P206-06 | AC2 (**link dùng hai lần**) | – | **S** dùng lại cùng token | **`410 LINK_INVALID`** `reason="used"`; tài khoản không đổi |
| TC-P206-07 | AC2 (**đua**) | token mới | **S** 50 `curl` song song cùng token | **Đúng 1** `200`; 49 `410`; **1** phiên được tạo (`count(*)` `auth_sessions`) |
| TC-P206-08 | AC2 (hết hạn / sai / **đoán mã**) | – | **S** `age_token`+72 h; 1.000 token ngẫu nhiên 43 ký tự | `410 expired`; `410 invalid` (không trúng), giới hạn IP 20/phút (`429`); phản hồi cùng khoá JSON |
| TC-P206-09 | AC2 (băm) | – | **D** `select count(*) from auth_tokens where token_hash='$t'`; `pg_dump \| grep -c "$t"`; log | `0` (chỉ băm); `0` ở log / `audit_log` / `mail_outbox.payload` |
| TC-P206-10 | AC2 | – | **G** `-run 'TestAcceptInvite\|TestAcceptInviteReuse\|TestAcceptInviteExpired72h\|TestAcceptInviteRace\|TestAcceptInviteWeakPassword'` | `ok` |
| TC-P206-11 | AC3 (**Admin không biết mật khẩu**) | – | **S** `grep -n -iE 'password\|token' backend-go/openapi.yaml \| grep -E '/admin/' \| wc -l`; quét **mọi** phản hồi thật của 6 thao tác `/admin/users*` bằng `grep -ciE 'password\|hash\|token'`; log `/admin/users` có thân? | `0` (trừ khai báo `Idempotency-Key`); phản hồi không có `password`, `password_hash`, `token`; log truy cập không chứa thân; **không** đường để Admin đặt / "đặt lại" mật khẩu hộ (thử `PATCH {"password":"x"}`, `POST /admin/users/{id}/reset-password` → 422/404) |
| TC-P206-12 | AC3 | – | **G** `-run 'TestAdminAPINeverCarriesSecrets\|TestNoPasswordHashInJSON'` | `ok` |
| TC-P206-13 | AC4 | `status=INVITED` | **S** `POST /admin/users/{id}/resend-invite` hai lần liền; liên kết cũ; trên người `ACTIVE`; bằng TEACHER | `200`, `429` (≤ 60 s); token cũ chưa dùng **bị thu hồi** (`410`), token mới 72 h + thư mới; `ACTIVE` → `409 CONFLICT`; TEACHER → `403` |
| TC-P206-14 | AC4 | – | **G** `-run 'TestResendInvite\|TestResendInviteOnlyInvited\|TestResendInviteThrottle'` | `ok` |
| TC-P206-15 | AC5 (khoá) | `gv.moi-1` đang đăng nhập (access `$AC`) | **S** `PATCH /admin/users/{id} {"status":"DISABLED","version":N}`; `curl -H "Authorization: Bearer $AC" …/me/courses`; đo thời gian; refresh | `200`; mọi phiên `revoked_reason=ACCOUNT_DISABLED`; access cũ **`401 SESSION_REVOKED` ≤ 1 s**; refresh `401`; `enrollments` **giữ nguyên** (count trước-sau) |
| TC-P206-16 | AC5 | `DISABLED` | **S** đăng nhập **mật khẩu đúng**; đăng nhập **mật khẩu sai** | Đúng → `403 ACCOUNT_DISABLED` "Tài khoản đã bị khoá. Hãy liên hệ quản trị viên."; sai → `401 INVALID_CREDENTIALS` (**không lộ trạng thái**) |
| TC-P206-17 | AC5 | – | **S** mở khoá `{"status":"ACTIVE"}` (có và không có `password_hash`); sai `version` | Có hash → `ACTIVE`, đăng nhập lại được; trống hash → `INVITED`; sai version → `409 VERSION_CONFLICT`; `audit_log` ghi người làm |
| TC-P206-18 | AC5 | – | **G** `-run 'TestDisableRevokesSessions\|TestDisabledLoginUniform\|TestDisabledCorrectPassword403\|TestEnableRestores\|TestDisableVersionConflict\|TestDisableKeepsEnrollments'` | `ok` |
| TC-P206-19 | AC6 (vai) | TEACHER `T1`, TA `A1` | **S** `PATCH {role:"TA"}` (TEACHER→TA) và ngược lại; `PATCH` tới/từ `ADMIN`, `STUDENT`; tự đổi vai mình; tự khoá mình; khoá / hạ ADMIN **cuối cùng** | TEACHER↔TA: `200` + mọi phiên thu hồi (token mới mang vai mới); tới/từ `ADMIN`/`STUDENT` → `422`; tự đổi/khoá → `409 CONFLICT` `details.reason="self"`; ADMIN cuối → `409` `details.reason="last_admin"`; `enrollments` không đổi |
| TC-P206-20 | AC6 | – | **G** `-run 'TestRoleChangeAllowedOnlyStaff\|TestRoleChangeRevokesSessions\|TestCannotChangeSelf\|TestCannotDisableLastAdmin\|TestCannotTouchAdminOrStudentRole'` | `ok` |
| TC-P206-21 | AC7 (**ma trận quyền**) | 4 vai + không JWT + JWT hết hạn | **S** 4 thao tác (`GET/POST /admin/users`, `PATCH /{id}`, `POST /{id}/resend-invite`) × {ADMIN, TEACHER, TA, STUDENT, không JWT, hết hạn}; kể cả `GET` | **Chỉ ADMIN**; TEACHER/TA/STUDENT → `403 FORBIDDEN` `details.reason="role"` **kể cả `GET`**; không JWT → `401`; hết hạn → `401 TOKEN_EXPIRED`; không ca lệch (≥ 24 ca) |
| TC-P206-22 | AC7 (vai trong JWT thắng DB) | – | **S** JWT STUDENT hợp lệ rồi `update users set role='ADMIN'` trong DB, gọi `/admin/users` bằng JWT cũ; ngược lại | Vai **trong JWT** thắng (FR-40): vẫn `403`; token mới sau đăng nhập mới mang vai DB |
| TC-P206-23 | AC7 (token giả) | – | **S** JWT `alg=none`, sửa `role=ADMIN` payload, secret sai | `401`; không tạo / sửa được người dùng; DB không đổi |
| TC-P206-24 | AC7 (IDOR) | – | **S** TEACHER `PATCH /admin/users/<id ADMIN>`; STUDENT `GET /admin/users?q=…` ; header `X-User-Id`/`X-Role` giả | `403`; không lộ dữ liệu; header giả vô tác dụng |
| TC-P206-25 | AC7 | – | **G** `-run TestAdminUsersRBACMatrix -v` | `ok`; ≥ 32 ca |
| TC-P206-26 | AC8 | ≥ 57 SV | **S** `GET /admin/users?limit=30`, theo `next_cursor` đi hết; `limit=101`; `?role=`, `?status=`, `?q=nguyen` (không dấu) và `q=Nguyễn`, `q=sv.g`; `?cursor=rác` | Phân trang con trỏ: mặc định 30, tối đa 100 (`limit=101` → `422`); `{items,next_cursor}`; đi hết **đủ mọi người, không trùng / không thiếu** (đếm với `select count(*)`); `q` không phân biệt dấu ở tên, tiền tố ở email; cursor rác → `400/422 INVALID_CURSOR` |
| TC-P206-27 | AC8 (dữ liệu tối thiểu) | – | **S** `jq '.items[0] \| keys'` ; `grep -ciE 'student_code\|password\|ics_token\|failed'` | Khoá đúng `{id,email,full_name,role,status,last_login_at,version}`; **không** `student_code`, `password_hash`, `ics_token`, số lần sai |
| TC-P206-28 | AC8 (N+1, chỉ mục) | – | **D** đếm truy vấn / `EXPLAIN` | 1 truy vấn; dùng chỉ mục (không Seq Scan trên bảng lớn) |
| TC-P206-29 | AC8 | – | **G** `-run 'TestAdminListCursor\|TestAdminListFilters\|TestAdminListMinimalFields\|TestAdminListNoNPlusOne\|TestAdminListLimitCap'` | `ok` |
| TC-P206-30 | AC9 (Admin đầu tiên) | DB trống | **S** `ADMIN_PASSWORD="$PW" ./bin/gateway admin create --email admin@edupilot.local --name "Quản trị"; echo rc=$?`; lần hai; `ADMIN_PASSWORD=123 …`; mật khẩu qua tham số dòng lệnh (`--password x`); `ps` khi chạy | `rc=0`, ADMIN `ACTIVE`, đã xác minh; lần hai `rc=0` "đã tồn tại", **không đổi** mật khẩu (`md5(password_hash)` không đổi), `count(role='ADMIN')=1`; yếu → `rc=1`; **không** nhận mật khẩu từ tham số dòng lệnh (không lộ ở `ps`); không route HTTP tạo ADMIN; `audit_log` (`actor_id=NULL`, `admin_bootstrap`) |
| TC-P206-31 | AC9 | – | **S** `grep -rn '"ADMIN"' backend-go/internal --include=*.go \| grep -v _test.go` tìm đường tạo ADMIN trừ CLI; thử `POST /auth/register` `role:ADMIN`, `POST /admin/users role:ADMIN` | Chỉ CLI tạo được ADMIN |
| TC-P206-32 | AC10 | Admin, `/admin/users` | **A** đọc bảng: cột, chip lọc, ô tìm, số nút `primary`; mở `Mời giảng viên` (Drawer) | Cột Tên, Email, Vai trò, Trạng thái, "Lần cuối"; **một** nút chính `Mời giảng viên`; Drawer tại chỗ, **không** `role=dialog`; trường Email, Họ và tên, Vai (Giảng viên / Trợ giảng); **không có** "Tạo sinh viên" |
| TC-P206-33 | AC10 | – | **A** gửi lời mời; `Gửi lại lời mời` ở hàng `INVITED`; `Khoá tài khoản` → `UndoLine` → `Hoàn tác` trong 5 s | Dòng tĩnh "Đã gửi link mời, hạn 72 giờ." tại chỗ (không toast); khoá lạc quan + "Đã khoá … · Hoàn tác" 5 s, `Hoàn tác` = `PATCH` mở khoá; rỗng "Chưa có người dùng khớp…" |
| TC-P206-34 | AC10 | 375 | **A** `AUDIT_SRC`, `TOUCH_SRC`, axe; < 720 px thành danh sách | Sạch; 0 `serious`; không từ kỹ thuật; thay màn mock (không `useDemoSlice` ở `/admin/users`: `grep`) |
| TC-P206-35 | AC10 | – | **G** `$PW account.spec.ts -g 'admin users page'` | `rc=0` |
| TC-P206-36 | AC11 | `/invite/<token>` | **A** mở liên kết mời; đọc lời chào, form, số nút primary; đặt mật khẩu; kiểm URL | "Chào {Họ tên}, bạn được mời làm Giảng viên trên EduPilot." (gọi `tokens/preview`); hai ô mật khẩu `autocomplete="new-password"` + chính sách; **một** nút `Đặt mật khẩu và vào`; thành công → `/` đã đăng nhập; token **bị thay** trong URL (`/invite/·`) |
| TC-P206-37 | AC11 | – | **A** liên kết hết hạn / đã dùng / token rác; `Referrer-Policy`; có hiện email? | "Lời mời đã hết hạn hoặc đã được dùng. Hãy nhờ quản trị viên gửi lại." (không nút tự gửi lại, **không lộ email**); `Referrer-Policy: no-referrer`; log Caddy `grep -c "$t"` = 0 |
| TC-P206-38 | AC11 | – | **G** `$PW account.spec.ts -g 'invite page'` | `rc=0` |
| TC-P206-39 | AC12 | Admin | **A** ép lỗi: `409 CONFLICT` email trùng; `422`; `403`; offline; `409 VERSION_CONFLICT` | Lỗi **tại ô / hàng** tiếng Việt ("Email này đã có tài khoản."); nội dung Drawer **giữ**; `Gửi lại` dùng **cùng** `Idempotency-Key` (so header); xung đột phiên bản → "Tài khoản này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?" |
| TC-P206-40 | AC12 | – | **G** `$PW account.spec.ts -g 'admin users errors'` | `rc=0` |
| TC-P206-41 | AC13 | `https://localhost` | **A** luồng: Admin mời → Mailpit → GV đặt mật khẩu → `/`; Admin khoá → GV (đang mở tab) bị đăng xuất; GV đăng nhập → "Tài khoản đã bị khoá"; Admin mở khoá → đăng nhập lại được | Đi trọn; `$PW account.spec.ts -g 'invite flow'` (`@real`) `rc=0` |
| TC-P206-42 | tổng (log/PII) | – | **D** `docker compose logs gateway worker caddy \| grep -c 'gv.moi-1@example.test\|$t\|$PW'` | `0` |
| TC-P206-43 | tổng | – | **S** `go vet && golangci-lint run && go test -race -count=1 ./internal/user/... ./internal/auth/...` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Tạo ADMIN / STUDENT bằng API; trường lạ | 02, 31 |
| Admin biết / đặt mật khẩu hộ | 11 |
| Link mời dùng hai lần / đua 50 luồng / hết hạn / đoán mã | 06–08 |
| Token ở DB / log / Referer | 09, 37, 42 |
| Phân quyền `/admin/*` (kể cả `GET`), JWT giả, vai DB ≠ JWT | 21–24 |
| Khoá không cắt phiên tức thì; lộ trạng thái khoá | 15, 16 |
| Khoá / hạ ADMIN cuối cùng; tự khoá | 19 |
| Lộ `student_code` / hash trong danh sách | 27 |
| Mật khẩu qua tham số CLI | 30 |

## Câu hỏi cho BA / PM
- **Q-QC-P206-1** — TC-P206-22 giả định "vai trong JWT thắng DB" tới khi JWT hết hạn (≤ 15 phút) *trừ khi phiên bị thu hồi*: khi đổi vai bằng API phiên bị thu hồi (AC6); khi sửa DB trực tiếp thì không. QC chỉ kiểm sửa DB để xác nhận FR-40. — *chờ xác nhận*.
- **Q-QC-P206-2** — TC-P206-30: `--password` trên dòng lệnh không tồn tại theo AC9; QC chỉ kiểm cờ **không được hỗ trợ** (lỗi rõ), không kiểm `ps`. — *chờ xác nhận*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-account-security, APPROVED 2026-10-03).

Tổng: 43 TC.
