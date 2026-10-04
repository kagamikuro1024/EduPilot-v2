# Báo cáo QC — US-P2-05 (chờ tăng dần, khoá 15 phút, giới hạn IP, chính sách mật khẩu, `login_attempts`, màn bị chờ)
**Kết luận: PASS có điều kiện** — không lỗ hổng. 2 lệch nhỏ (L1, L2) và 1 ghi chú đo (L3). Bản chấm `c946461` (`sprint/4-p2`); stack riêng của QC (Postgres, Redis, Mailpit, 2 gateway `testroutes` :8080/:8081, worker, Next `build` :3400, Chrome for Testing); thời gian thật. `go vet` rc=0, `golangci-lint` 0 issues, `go test -race -count=1 -tags integration ./internal/auth/...` ok (140 `--- PASS`, 0 FAIL), `internal/contract` + `internal/httpapi` ok. Playwright `account.spec.ts -g 'login throttled'`: 4 pass. Q-QC-P205-1/2/3: BA đã trả lời, QC làm theo.

## Lệch / ghi chú
- **L1 (TC-31).** Nợ "dọn `login_attempts` quá 90 ngày" (AC9) dev ghi ở handoff nhưng **chưa có** ở `docs/PROGRESS.md` (tệp của PM) — `grep "90 ngày\|login_attempts" docs/PROGRESS.md` = 0. PM thêm vào mục Nợ.
- **L2 (TC-29).** Test `TestDummyCompareRuns` trong chữ TC **không tồn tại**; phần tương đương là `TestLoginTimingEqualized` (PASS, 26 s). QC sửa tên trong TC khi chấm lại.
- **L3 (TC-09, đo thật).** Khoá lúc t0=19:52:50Z (lần sai thứ 10): DB `locked_until` = t0+14:59,8 (đúng ±1 s); T+1:00 → `429 LOGIN_THROTTLED retry_after=840` (đúng 900−60). Ở T+14:30 và T+15:05 vẫn `429` nhưng **do QC**: tải đo thời gian (TC-28) và 5 lần sai trên giao diện từ cùng `127.0.0.1` làm IP bị chặn 15 phút (khoá `ep:auth:blockip:127.0.0.1`, `retry_after=262` ≠ 30 là dấu hiệu). Sau khi xoá khoá IP, đăng nhập tài khoản đã hết khoá (≈ T+15:13) trả `401` (mật khẩu thử sai, không `429`) và `failed_logins=1`, `locked_until=null` — hết khoá đúng, bộ đếm bắt đầu lại từ 1. Mốc 14:30/15:05 sạch cần chạy lại trên IP riêng: ghi **không kiểm được sạch**; phần rút gọn (TC-07) PASS.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | thời gian thật: lần 1–5 sai `401`; chờ sau lần sai 5,6,7,8,9 = **2, 4, 8, 16, 32 s** (`retry_after` + `Retry-After` header); hết chờ → kiểm lại; cuối `failed_logins=9` |
| 02 | PASS | trong chờ: mật khẩu **đúng** và **sai** đều `429 LOGIN_THROTTLED` cùng `retry_after` (đo 2–3 ms, không bcrypt), thân giống hệt; đúng sau hết chờ → `200`, `failed_logins=0`, khoá Redis xoá |
| 04 | PASS | lần 10: `failed_logins=10`, `locked_until−now=900 s` |
| 05 | PASS | khoá: mật khẩu đúng `429 LOGIN_THROTTLED retry_after=900`, cùng mã; thân không chứa "khoá"/`LOCKED` |
| 06 | PASS | đúng **1** thư "Tài khoản EduPilot bị khoá tạm thời" dù thử thêm 6 lần; `dedupe_key=account_locked:<user>:<locked_unix>` khớp `locked_until` |
| 07 | PASS (rút gọn) | `locked_until` chỉnh về quá khứ (ghi rõ) → đăng nhập đúng `200`, `failed_logins=0` |
| 09 | xem L3 | một phần: mốc đầu PASS; mốc 14:30 / 15:05 bị nhiễu bởi IP-block của QC |
| 10 | PASS | 12 lần sai song song, email thật vs không tồn tại, mỗi bước chờ cùng lịch: **0/12 bước khác** (mã, khoá JSON, `retry_after`, header, `Set-Cookie`); lần 11–12 cùng `429 LOGIN_THROTTLED retry_after=900`; không `user_id` |
| 11 | PASS | email không tồn tại: 0 thư, không tạo user; chủ thật: 1 thư |
| 13 | PASS | 4 sai + 1 đúng → `200`, khoá Redis xoá, DB 0; 4 sai nữa rồi lần 5 → `401` (đếm lại từ 1, không cộng dồn); TTL khoá đếm 1800 s; hết TTL (rút gọn, chỉnh Redis, ghi rõ) → lần sai kế `n=1` |
| 14 | PASS | khoá (`failed_logins=10`, đúng → `429`) → `forgot` + `reset-password` `200` → mật khẩu mới đăng nhập ngay `200`, `locked_until=NULL`, `failed_logins=0` |
| 16 | PASS | Redis `docker stop`: 10 lần sai → DB `failed_logins=10`, `locked_until` có; mật khẩu đúng `429 retry_after=899`; email không tồn tại 12 lần `401` (không chờ — chấp nhận); tài khoản khác đăng nhập `200`; log `ERROR` "Redis không với tới" 2 dòng cách 30 s; `start redis` → hoạt động, khoá DB còn |
| 18 | PASS | bảng đo (429 đầu): login **11** (`401`×10 rồi `429`), register **6**, forgot/IP **6**, forgot/email **4** (mỗi lần IP khác), resend-verification/email **2**, verify-email **21**, reset-password **21**, tokens/preview **21** (3 endpoint **chung** bộ đếm: 7+7+7 → `429` ở lần 21), refresh **61**; mọi `429` mang `RATE_LIMITED`, header `Retry-After` và `retry_after` |
| 19 | PASS | 30 lần sai, 30 email khác nhau, một IP, 190 s (≤ 10/phút): đều `401`; lần 31 `429 LOGIN_THROTTLED retry_after=900`; **đúng** từ IP đó cũng `429`; IP khác `200`; `blockip` TTL 900 |
| 20 | PASS | 12 lần xen kẽ :8080 / :8081 cùng IP → `429` ở lần 11 (bộ đếm chung qua Redis) |
| 21 | PASS | Redis tắt: 15 lần sai nhanh, 15×`401` (IP-limit mở cửa, fail-open); khoá tài khoản vẫn theo DB (TC-16) |
| 23 | PASS | 20 mật khẩu × 3 đường (`register`, `reset-password`, `POST /me/password`): kết quả **y hệt** — `1234567890`, `matkhau12345`, `abcdefghij`, `aaaaaaaaaa`, `1234567890a`, `PASSWORD1234`, `Password1234`, `matkhau123`, `mậtkhẩu123`, `MatKhau12345`, `Mậtkhẩu12345`, `0123456789`, `qwertyuiop` → `PASSWORD_COMMON`; 9 ký tự và `Mậtkhẩu12` (9 ký tự) `PASSWORD_TOO_SHORT`; 73 byte và 1.000 byte `PASSWORD_TOO_LONG`; `<phần-email>pass2026` `PASSWORD_CONTAINS_EMAIL`; `y`×72 `PASSWORD_LOW_ENTROPY` (lặp); `Passw0rd!!!`, `Cây-cầu-Hà-Nội-2026` hợp lệ; `Mậtkhẩu123456` bị COMMON (có trong danh sách, không phải lỗi). `accept-invite` (đường thứ 4) chưa có — **chờ US-P2-06** |
| 24 | PASS | `common_passwords.txt`: 22.344 dòng, 0 dòng < 10 ký tự; 20 mục ngẫu nhiên × (gốc, HOA, bỏ dấu) = 60 thử: 0 mật khẩu lọt |
| 25 | PASS | thân `422` không lặp mật khẩu; log 2 gateway + worker chứa 6 mật khẩu thử: 0; `pg_dump`: 0 |
| 27 | PASS | hash `$2a$12$`; 1 lần đăng nhập sai ≈ 232 ms; 72 byte đầu của mật khẩu khác → `401`; 73 byte → `401` (không `500`) |
| 28 | PASS | trung vị (4–5 mẫu mỗi nhóm; phần còn lại bị `429` của giới hạn IP 10/phút): email lạ 240 ms, INVITED 232 ms, ACTIVE sai mật khẩu 231 ms → 104 % / 100 % (≥ 65 %) |
| 30 | PASS | `outcome` có `SUCCESS`, `BAD_PASSWORD`, `UNKNOWN_EMAIL`, `THROTTLED`, `LOCKED`; `email_hash like '%@%'`=0; 3 dòng gần nhất 64 hex và sha256 khớp email đã thử; `user_agent` tối đa 10 ký tự (UA 300 ký tự → cắt 200 do test Go); cột chỉ `ip,outcome,created_at,user_id,id,email_hash,user_agent` |
| 31 | PASS có L1 | `GET /admin/login-attempts`, `/me/login-attempts`, `/login-attempts` → `404`; 0 handler đọc `login_attempts` |
| 33 | PASS | `/login` (build thật): sau chờ — "Bạn đã thử quá nhiều lần. Thử lại sau 00:04." đếm 00:04→03→02→01 mỗi ~1 s, `role="status"`, nút `Đăng nhập` `disabled` với `aria-describedby=login-wait` trỏ vào dòng đó, mở lại ở ~3,4 s; email + mật khẩu (14 ký tự) **giữ**; `Quên mật khẩu?` bấm được; `innerText` không có `LOGIN_THROTTLED|RATE_LIMITED`. Ca `page.route` chặn: thay bằng ca thật + spec |
| 35 | PASS | gateway với `TRUSTED_PROXY_CIDRS=10.0.0.0/8` (nguồn QC `127.0.0.1` không tin cậy): 12 lần đăng nhập mỗi biến thể `X-Forwarded-For: 1.2.3.n`, `X-Real-IP`, `Forwarded: for=`, `X-Forwarded-For: 8.8.8.n, 10.0.0.1`, `X-Forwarded-For: 10.0.0.n` (email mỗi lần khác): **`429` ở lần 11**; khoá Redis chỉ theo `127.0.0.1`. Nguồn tin cậy: `TestForgedXForwardedForIgnored` PASS; **chưa qua Caddy** |
| 36 | PASS | Admin 10 lần sai (chờ đúng) → mật khẩu đúng `429 retry_after=900`, `failed_logins=10` |
| 37 | PASS | `POST /admin/users/<id>/unlock`, `PATCH`/`PUT /admin/users/<id>`, `POST /auth/unlock` → `404`; `?bypass=1` và header `X-RateLimit-Bypass` không đổi kết quả (`429`); Admin vẫn khoá |
| 39 | ghi rủi ro | DoS khoá (đã duyệt, Q3): lối thoát là đặt lại mật khẩu mở ngay (TC-14 PASS) |
| 40 | PASS | xem đầu báo cáo |
| 03, 08, 12, 15, 17, 22, 26, 29, 32, 34, 38 | PASS | test Go của dev (tên đủ ở đầu báo cáo, trừ L2) ok; TC-34 Playwright 4 pass |

## Việc sau
PM: L1. QC chấm lại: TC-09 mốc 14:30 / 15:05 trên IP riêng (khi cần), TC-23 `accept-invite` ở P2-06, TC-35 qua Caddy ở cổng P2.
