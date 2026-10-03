# QC test case — US-P2-05 (chống dò mật khẩu: chờ tăng dần, khoá 15 phút, giới hạn IP, chính sách mật khẩu, `login_attempts`)
Nguồn: `docs/specs/FEAT-account-security/US.md` US-P2-05 AC1–AC11 + `SRS.md` 4.2 (ngưỡng), 5.7 (khoá Redis), 8.1 (bảng giới hạn, `TRUSTED_PROXY_CIDRS`, `BCRYPT_COST`). **Trọng tâm tấn công:** dò mật khẩu (brute-force) trên một email và trên nhiều email từ một IP, mạo `X-Forwarded-For`, liệt kê tài khoản qua throttle, khoá để gây DoS chủ tài khoản, Redis chết.

Tiền điều kiện chung: stack test (2 gateway sau Caddy; Redis; Mailpit); biến của `US.md`; `TRUSTED_PROXY_CIDRS` = dải mạng của Caddy trong compose (QC đọc từ `.env.test`); tài khoản dùng riêng cho từng TC: `sv.kha@edupilot.local` (đã đăng nhập đúng) và email giả `khong-co-<n>@example.test`. Mỗi TC **đặt lại trạng thái** (`redis-cli --scan --pattern 'ep:*login*' \| xargs redis-cli del`, `update users set failed_logins=0, locked_until=null`) trước khi chạy. **Đồng hồ:** thời gian thật (đợi thật, ghi thời điểm); ca dài (15 phút) QC rút ngắn bằng cách chỉnh `locked_until` trong DB (ghi rõ) và **đo thêm** một lần thật cho TC-P205-09. Công cụ: **S** `scripts/p205.sh`, **D**, **A**, **G**.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P205-01 | AC1 (chờ tăng dần) | email thật | **S** gửi 9 lần mật khẩu sai liên tiếp **chờ đúng thời gian cho phép** (đọc `retry_after`); ghi mã + `retry_after` mỗi lần | Lần 1–4: `401 INVALID_CREDENTIALS`; lần 5 trở đi: sau mỗi lần sai bắt buộc chờ 2, 4, 8, 16, 32 s (sau lần sai 5, 6, 7, 8, 9) |
| TC-P205-02 | AC1 | – | **S** thử **trong lúc chờ** với mật khẩu **sai** và **đúng**; kiểm `retry_after` ±1 s | `429 LOGIN_THROTTLED` + `retry_after` (giây còn lại, ±1) cho cả mật khẩu đúng (**không kiểm mật khẩu**: đo thời gian nhanh hơn một phép bcrypt ≈ vài ms); hết chờ → kiểm lại bình thường (mật khẩu đúng → `200`) |
| TC-P205-03 | AC1 | – | **G** `-run 'TestLockoutBackoffSchedule\|TestLockoutBackoffBlocksCorrectPassword\|TestLockoutBackoffRetryAfterAccurate'` | `ok` |
| TC-P205-04 | AC2 (khoá 15 phút) | – | **S** 10 lần sai liên tiếp (chờ đúng giữa các lần, hoặc chỉnh Redis để rút gọn **ghi rõ**); `select failed_logins, locked_until > now(), locked_until - now() from users where email=…` | `failed_logins=10`; `locked_until ≈ now()+15 phút` (±5 s) |
| TC-P205-05 | AC2 | đang khoá | **S** đăng nhập với mật khẩu **đúng**; so thân với lúc chờ AC1 | `429 LOGIN_THROTTLED` `retry_after` = giây còn lại (cùng mã với AC1, **không** lộ chữ "khoá"/`LOCKED`) |
| TC-P205-06 | AC2 (mail) | – | **D** `mail_count sv.kha@edupilot.local` trước-sau; chủ đề; thử khoá lần hai trong cùng `locked_until` | Đúng **1** thư `account_locked` mỗi lần khoá (`dedupe_key = account_locked:<user>:<locked_until>`); không thư mới khi thử tiếp trong lúc khoá |
| TC-P205-07 | AC2 (hết khoá) | – | **S** đặt `locked_until=now()-1s` (ghi rõ); đăng nhập đúng; đọc `failed_logins` | `200`; `failed_logins=0` |
| TC-P205-08 | AC2 | – | **G** `-run 'TestLockoutAfter10\|TestLockoutBlocksCorrectPassword\|TestLockoutMailOnce\|TestLockoutExpires'` | `ok` |
| TC-P205-09 | AC2 (đo thật 15 phút) | – | **S** 1 lần chạy **không rút gọn**: khoá, rồi đăng nhập đúng ở phút 14:30 (`429`) và 15:05 (`200`) | Mốc khoá đúng ±10 s (dài, chạy nền; nếu không đủ thời gian ghi "KHÔNG KIỂM ĐƯỢC" cho phần đo thật, không FAIL phần rút gọn) |
| TC-P205-10 | AC3 (**liệt kê tài khoản qua throttle**) | email không tồn tại và email thật | **S** chạy **song song** cùng một lịch 12 lần sai cho email thật và email không tồn tại; so từng bước: mã, khoá JSON, `retry_after`, header | **Khuôn giống hệt** ở mọi bước (lần 5+ cùng `429 LOGIN_THROTTLED` + cùng `retry_after`); chỉ khác ở thư `account_locked` (chỉ chủ thật nhận); không `user_id` |
| TC-P205-11 | AC3 | – | **D** `mail_count khong-co-1@example.test` sau 10 lần sai | `0` thư (không có chủ) |
| TC-P205-12 | AC3 | – | **G** `-run TestThrottleUniformForUnknownEmail` | `ok` |
| TC-P205-13 | AC4 (đặt lại) | – | **S** 4 lần sai rồi 1 lần đúng; rồi 4 lần sai nữa; 3 lần sai, đợi (rút gọn bằng chỉnh TTL **ghi rõ**) > 30 phút, 1 lần sai | Thành công đặt bộ đếm về 0 (lần sai kế là "lần 1"); lần sai > 30 phút trước **không cộng dồn**; `failed_logins`, khoá Redis đều về 0 |
| TC-P205-14 | AC4 (reset mở khoá) | đang khoá | **S** đặt lại mật khẩu qua thư (US-P2-04) | Khoá **mở ngay**: đăng nhập mật khẩu mới `200`; `locked_until=NULL`; (đánh đổi đã ghi ở `QUESTIONS.md` Q3) |
| TC-P205-15 | AC4 | – | **G** `-run 'TestCounterResetsOnSuccess\|TestCounterDecays30m\|TestResetUnlocks'` | `ok` |
| TC-P205-16 | AC5 (Redis chết) | `$C stop redis` | **S** 10 lần sai cho tài khoản thật; sau đó đăng nhập đúng; `select failed_logins, locked_until`; log | **Vẫn bị khoá** nhờ DB (`failed_logins=10`, `locked_until` có); email không tồn tại **không** bị chờ (chấp nhận); log `error` ≤ 1 lần / 30 s; đăng nhập đúng (tài khoản chưa khoá) vẫn chạy; sau `start redis` hoạt động lại |
| TC-P205-17 | AC5 | – | **G** `-tags integration -run TestLockoutWithoutRedis -v` | `ok` |
| TC-P205-18 | AC6 (bảng giới hạn — đo từng hành động) | Redis sạch | **S** cho mỗi hành động gửi N+1 lần từ **một IP**: `login` 10/phút (11 lần: `for i in $(seq 1 11); do curl -sk -o /dev/null -w '%{http_code} ' … /auth/login; done`); `register` 5/giờ; `forgot-password` 5/giờ/IP **và** 3/giờ/email; `resend-verification` 1/60 s/email; `verify-email`, `reset-password`, `accept-invite`, `tokens/preview` 20/phút/IP; `refresh` 60/phút/IP | Login: mười lần `401` rồi `429`; mỗi hành động chặn **đúng** ở lần vượt (`429 RATE_LIMITED`) với header `Retry-After` và `retry_after`; `forgot-password` chặn theo email ở lần 4 dù IP khác; ghi bảng thực đo |
| TC-P205-19 | AC6 (30 sai / 15 phút / IP) | – | **S** từ **một IP** gửi 31 lần đăng nhập sai cho **31 email khác nhau** (phân tán, tránh khoá theo email) trải qua nhiều phút (≤ 10/phút) | Sau 30 lần sai IP bị chặn 15 phút: `429 LOGIN_THROTTLED`; kể cả một **đăng nhập đúng** từ IP đó cũng `429`; từ IP khác vẫn `200` |
| TC-P205-20 | AC6 (hai gateway) | 2 gateway | **S** xen kẽ yêu cầu qua hai cổng gateway (không qua Caddy hoặc qua Caddy round-robin) | Bộ đếm **chia sẻ** (Redis): tổng 11 lần ở hai gateway vẫn `429` ở lần 11 |
| TC-P205-21 | AC6 (Redis chết, IP) | `stop redis` | **S** 15 lần đăng nhập sai nhanh | Giới hạn theo IP **mở cửa** (fail-open như PG); khoá theo tài khoản vẫn dựa DB (TC-16); ghi hành vi |
| TC-P205-22 | AC6 | – | **G** `-tags integration -run 'TestRateLimitTable\|TestIPBlockAfter30Failures\|TestRateLimitSharedAcrossInstances'` | `ok`; ≥ 8 hành động |
| TC-P205-23 | AC7 (chính sách — mọi đường) | – | **S** thử 30 mật khẩu qua **4 đường** (`register`, `reset-password`, `accept-invite`, `POST /me/password`): `1234567890`, `matkhau12345`, `Passw0rd!!!`, `abcdefghij`, `aaaaaaaaaa`, `1234567890a`, chuỗi 9 ký tự, 10 ký tự tiếng Việt có dấu `Mậtkhẩu12`, 73 byte, 72 byte, 1.000 byte, `qc-1@example.test`-chứa `qc-1pass2026`, `PASSWORD1234`, `Password1234` (biến thể hoa/thường), `matkhau123` (bỏ dấu), `mậtkhẩu123` | Cùng kết quả ở **cả bốn đường**: `422 VALIDATION_FAILED` với `PASSWORD_TOO_SHORT`/`PASSWORD_COMMON`/`PASSWORD_CONTAINS_EMAIL`/`PASSWORD_TOO_LONG`/(lặp, dãy tăng); hợp lệ: ≥ 10 ký tự Unicode, ≤ 72 byte, không phổ biến, không chứa phần trước `@` (≥ 4 ký tự); **không** bắt buộc hoa/số/ký hiệu (`Mậtkhẩu12` hợp lệ nếu không nằm trong danh sách) |
| TC-P205-24 | AC7 (danh sách phổ biến) | repo | **S** `wc -l < backend-go/internal/auth/common_passwords.txt`; `awk 'length($0)<10' … \| wc -l`; thử 20 mục ngẫu nhiên của danh sách qua `register` | `≥ 1000`; `0` mục ngắn hơn 10; cả 20 bị `PASSWORD_COMMON` (không phân biệt hoa thường, cả bản bỏ dấu) |
| TC-P205-25 | AC7 | – | **S** mật khẩu không xuất hiện ở log / thân lỗi: `docker compose logs … \| grep -c "<mật khẩu thử>"`; thân `422` có lặp lại mật khẩu? | `0`; thân lỗi **không** chứa mật khẩu |
| TC-P205-26 | AC7 | – | **G** `-run 'TestPasswordPolicyTable\|TestPasswordPolicyAllEntryPoints\|TestPasswordNeverLogged'` | `ok`; ≥ 25 ca |
| TC-P205-27 | AC8 | – | **S** kiểm `BCRYPT_COST`: `select password_hash from users limit 3` → tiền tố `$2a$12$`; đo thời gian bcrypt đăng nhập (≈ 150–400 ms); mật khẩu > 72 byte ở mọi đường | Cost = `BCRYPT_COST` (mặc định 12); > 72 byte **bị từ chối** (`PASSWORD_TOO_LONG`) trước khi băm — **không cắt im lặng** (thử đăng nhập với 72 byte đầu của mật khẩu 80 byte: phải `401`) |
| TC-P205-28 | AC8 (thời gian giả) | – | **S** đo trung vị 30 mẫu: email không tồn tại / INVITED / mật khẩu sai | Có phép so bcrypt giả (thời gian ≥ 65 % của mật khẩu sai) — xem TC-P202-05 |
| TC-P205-29 | AC8 | – | **G** `-run 'TestBcryptCost\|TestDummyCompareRuns\|TestRejectOver72Bytes'` | `ok` |
| TC-P205-30 | AC9 | sau các ca trên | **D** `select outcome, count(*) from login_attempts group by 1`; `select count(*) from login_attempts where email_hash like '%@%'`; cột `user_agent` dài tối đa | Mỗi lần thử đúng **một** dòng; `outcome` ∈ {`SUCCESS`,`BAD_PASSWORD`,`UNKNOWN_EMAIL`,`THROTTLED`,`LOCKED`,`DISABLED`}; `email_hash` = sha256 chữ thường (QC tự tính và so 3 dòng); `0` email rõ; `length(user_agent) ≤ 200`; không cột mật khẩu / kết quả so sánh |
| TC-P205-31 | AC9 | – | **S** `GET /api/v1/admin/login-attempts`, `/me/login-attempts`, … ; `grep -rn login_attempts backend-go/internal --include=*.go \| grep -E 'handler\|http'` | `404` (không API đọc); `0` handler; bản ghi nợ "dọn 90 ngày" có ở `docs/PROGRESS.md` (đọc tệp) |
| TC-P205-32 | AC9 | – | **G** `-run 'TestLoginAttemptsRecorded\|TestLoginAttemptsNoPlainEmail'` | `ok` |
| TC-P205-33 | AC10 | `/login` | **A** chặn `page.route` trả `429 LOGIN_THROTTLED` `retry_after:3`; sau đó thật (5 lần sai) | "Bạn đã thử quá nhiều lần. Thử lại sau 00:03." đếm ngược mỗi giây (`role="status"`, cập nhật ≤ 1 lần/giây); nút `Đăng nhập` `disabled` kèm lý do tới hết giờ rồi `enabled` (≈ 3,1 s); ô email và mật khẩu **giữ**; `Quên mật khẩu?` dùng được; `innerText` không khớp `/LOGIN_THROTTLED\|RATE_LIMITED/`; không gợi ý email có tồn tại |
| TC-P205-34 | AC10 | – | **G** `$PW account.spec.ts -g 'login throttled'` | `rc=0` |
| TC-P205-35 | AC11 (**mạo `X-Forwarded-For`**) | – | **S** từ ngoài (không qua Caddy hoặc qua Caddy), gửi 11 lần đăng nhập sai, mỗi lần `X-Forwarded-For` khác (`1.2.3.4`, `5.6.7.8`, …, và `X-Real-IP`, `Forwarded: for=`) | Vẫn `429` ở lần 11 (IP thật bị tính, header giả **không** đổi IP khi nguồn ngoài `TRUSTED_PROXY_CIDRS`); thử trực tiếp cổng gateway từ host (không phải IP Caddy) cũng không bị đổi IP |
| TC-P205-36 | AC11 (Admin) | – | **S** 10 lần sai cho `admin@edupilot.local` | Admin cũng bị chờ / khoá như người khác (không ngoại lệ) |
| TC-P205-37 | AC11 | – | **S** tìm đường mở khoá thủ công: `POST /admin/users/{id}/unlock`, `PATCH … {"locked_until":null}`; tham số bỏ qua giới hạn (`?bypass=1`, header `X-RateLimit-Bypass`) | `404`/`422`; **không** có endpoint mở khoá ngoài đặt lại mật khẩu; không tham số / header nào bỏ qua |
| TC-P205-38 | AC11 | – | **G** `-run 'TestForgedXForwardedForIgnored\|TestAdminSubjectToLockout'` | `ok` |
| TC-P205-39 | **DoS khoá** (rủi ro) | – | **S** kẻ xấu cố tình sai 10 lần với email của nạn nhân; nạn nhân đăng nhập | Nạn nhân bị `429` tới hết khoá (đã chấp nhận ở `QUESTIONS.md` Q3); lối thoát: đặt lại mật khẩu mở ngay (TC-14). QC **ghi** như rủi ro đã duyệt, không FAIL |
| TC-P205-40 | tổng | – | **S** `go vet && golangci-lint run && go test -race -count=1 ./internal/auth/...` + `-tags integration` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Dò mật khẩu một email (chờ tăng dần, khoá) | 01–09 |
| Mật khẩu đúng vẫn qua được trong lúc khoá | 02, 05 |
| Lộ "có tài khoản không" qua throttle | 10, 11 |
| Dò nhiều email từ một IP | 19 |
| Giả IP bằng header | 35 |
| Redis chết = mất khoá / chặn toàn hệ thống | 16, 21 |
| Mật khẩu yếu / phổ biến / bị cắt 72 byte | 23, 24, 27 |
| Email / mật khẩu ở `login_attempts` / log | 25, 30 |
| Admin ngoại lệ; endpoint mở khoá lén | 36, 37 |

## Câu hỏi cho BA / PM
- **Q-QC-P205-1** — Giới hạn "30 sai / 15 phút / IP" đo bằng 31 email khác nhau từ một IP: do `login` 10/phút/IP cũng chạm, TC-P205-19 phải rải ≤ 10 lần/phút (≥ 4 phút). QC chấp nhận chạy nền ≈ 5 phút. — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Chấp nhận chạy nền ≈ 5 phút (rải ≤ 10 lần / phút để không chạm giới hạn `login`).
- **Q-QC-P205-2** — Mốc TTL Redis rút gọn: QC chỉnh `locked_until`/khoá Redis để mô phỏng thời gian; xác nhận cách này hợp lệ cho FAIL/PASS (kèm 1 lần đo thật TC-09). — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Hợp lệ: QC chỉnh `locked_until` / khoá Redis hoặc dùng biến rút gọn ở stack riêng để mô phỏng thời gian, kèm **một lần đo thật** TC-09 (SRS 8.1 ghi chú cho QC).
- **Q-QC-P205-3** — Danh sách mật khẩu phổ biến có "biến thể bỏ dấu" (`matkhau123`): QC thử 5 biến thể; nếu SRS không liệt kê tập biến thể cụ thể QC chỉ ghi kết quả. — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Đã định nghĩa (v1.1, AC7 + SRS 4.2.1): `fold` = NFD, bỏ dấu, `đ`→`d`, chữ thường, giữ khoảng trắng / ký tự khác, **không** leetspeak, không cắt số; ví dụ `MatKhau12345` và `Mậtkhẩu12345` bị chặn nếu `matkhau12345` có trong danh sách. QC thử 5 biến thể và so với định nghĩa này.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-account-security, APPROVED 2026-10-03).

Tổng: 40 TC.
