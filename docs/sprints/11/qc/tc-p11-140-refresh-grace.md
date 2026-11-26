# TC — P-11-140 phương án A: ân hạn 10 s khi xoay refresh token (qc, cặp 3)

Nguồn: `docs/sprints/11/proposals.md` P-11-140 (chủ dự án chọn A) + lệnh PM 27/11. Spec đang được BA sửa (SRS 4.2, US-P2-02 AC4, Q-QC-P202-1, Q9) — **mọi giá trị dưới đây theo lời PM; khi SRS ra, đối chiếu từng dòng rồi mới chạy**. Tệp này của qc; sửa sau khi SRS APPROVED chỉ qua proposal.

Phạm vi: gateway `/api/v1/auth/refresh` + cookie `ep_rt` + `apiClient` (tải lại / nhiều tab). Không đổi: `SET_EXPIRE`, đăng xuất, CSRF, trần tuyệt đối 30 ngày.
Môi trường: VPS (`ep-vps-test`, chạy sau khi PM báo VPS lên bản có sửa). Tài khoản **`sv.qc`, `sv.qc-b`** (không dùng sv.gioi/sv.kha, sv.qc2*). SQL chỉ đọc (`audit_log`, `auth_sessions`). Hạn mức đăng nhập 10/phút: mỗi ca đăng nhập 1 lần, gộp tối đa.
Ký hiệu: **A** = Playwright (Chromium, Pixel 7 + desktop), **S** = `fetch`/curl với `Origin` hợp lệ, **D** = SQL đọc, **G** = `go test`. «Ân hạn» = 10 s tính **từ lúc xoay** (không trượt theo lần trình lại). Ca có chờ thời gian dùng mốc ≤ 6 s (trong) và ≥ 13 s (ngoài), **không** đo biên 10 s.

| TC-id | Ý | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-RG-01 | **Tải lại nhanh 2 lần liên tiếp** (A) | Đăng nhập `sv.qc`; ép access hết hạn (xoá bộ nhớ token) để lần tải sinh `POST /auth/refresh`; `page.reload()` rồi `reload()` ngay, không chờ `networkidle`; lặp 5 vòng | Mọi vòng: URL không về `/login`; không `401 SESSION_REVOKED`; thấy tên tài khoản. D: `audit_log action='refresh_reuse'` của user = **0**; `auth_sessions` của user không tăng thêm hàng |
| TC-RG-02 | **Trình duyệt huỷ response giữa lúc xoay** (A) | `page.route('**/auth/refresh')`: `route.fetch()` (máy chủ đã xoay) rồi `route.abort()` — trình duyệt **không nhận cookie mới**; tải lại | Trang tải lại dùng token cũ **trong ân hạn** → vào được app, **không** `/login`; cookie `ep_rt` cuối cùng = token mới của máy chủ |
| TC-RG-03 | **Ân hạn trả lại đúng cặp, không xoay thêm** (S, D) | Login → `T1`; refresh bằng `T1` → `T2`; trình `T1` lần nữa sau ≤ 5 s | `200`; `Set-Cookie ep_rt` = **đúng `T2`** (cùng giá trị); `sid` giữ nguyên; access trả về hợp lệ gọi được `/me/courses`. D: `auth_sessions` của user **không thêm hàng**; hash hiện hành không đổi sau lần trình lại; `refresh_reuse` = 0 |
| TC-RG-04 | **Nhiều lần trình lại trong hạn, hạn không trượt** (S) | Như 03; trình `T1` lúc ~3 s (200) và ~6 s (200); rồi **chờ tới ≥ 13 s kể từ lúc xoay** (không phải kể từ lần trình cuối) và trình `T1` | Hai lần đầu `200` cùng `T2`; lần ở ≥ 13 s → `401 SESSION_REVOKED` (hạn đếm từ lúc xoay) |
| TC-RG-05 | **2 tab cùng làm mới trong 10 s** (A) | Một context, 2 trang cùng đăng nhập `sv.qc`; ép access hết hạn cả hai; hai trang gọi API cùng lúc (`Promise.all`); ghi mọi request `/auth/refresh` | Cả hai tab ở lại phiên, không `/login`, 0 `SESSION_REVOKED`; mọi `/auth/refresh` đều `200`. D: `refresh_reuse` = 0; **một** phiên, không hàng mới |
| TC-RG-06 | **Đua 2 yêu cầu cùng token cũ** (S) | Hai `fetch` song song cùng `T1` (`Promise.all`) | Cả hai `200` và nhận **cùng cặp** (một xoay, một ân hạn) hoặc một `200` + một ân hạn cùng giá trị; **không** có hai cookie hợp lệ khác nhau; `refresh_reuse` = 0 (thay cho kỳ vọng TC-P202-14 cũ) |
| TC-RG-07 | **Ngoài 10 s = dùng lại → thu hồi** (S, D) | Login → `T1`; refresh → `T2`; chờ ≥ 13 s; trình `T1` | `401 SESSION_REVOKED` (reason `refresh_reuse`); sau đó `T2` → `401`; access cũ → `401 SESSION_REVOKED` ≤ 1 s. D: `audit_log` đúng **1** dòng `entity=auth_session, action=refresh_reuse`, `details` không chứa token/email/tên |
| TC-RG-08 | **Token cũ hơn 2 đời → thu hồi** (S, D) | Login → `T0`; refresh → `T1`; refresh bằng `T1` → `T2` (ngay); trình `T0` (vẫn trong 10 s) | `401 SESSION_REVOKED` + thu hồi cả phiên (`T2` → `401`); `refresh_reuse` +1. Đối chứng: trình `T1` (1 đời trước, trong ân hạn) **trước** khi trình `T0` trả `200` = `T2` |
| TC-RG-09 | **Token hiện hành luôn xoay bình thường** (S) | Sau 03 (đã ở `T2` trong ân hạn của `T1`), refresh bằng `T2` → `T3`; rồi trình `T1` | `T2→T3` `200`, `T3 ≠ T2`; `T1` giờ là 2 đời → `401` + thu hồi (như 08) |
| TC-RG-10 | **Đăng xuất / thu hồi huỷ ân hạn** (S) | Login → `T1`; refresh → `T2`; `POST /auth/logout` bằng `T2`; trình `T1` ngay (trong 10 s); phiên B của cùng tài khoản refresh | `T1` → `401` (phiên đã chết **không** hồi sinh bằng ân hạn); phiên B không ảnh hưởng (`200`) |
| TC-RG-11 | **Ân hạn không vượt qua kiểm tra khác** (S, D) | Trong ân hạn: (a) `Origin: https://evil.example` → ; (b) user chuyển `LOCKED` (**không** sửa DB tay — dùng tài khoản đã khoá sẵn nếu có, nếu không ghi N); (c) `expires_at`/trần tuyệt đối đã qua (dùng phiên cũ nếu có, không thì N) | (a) `403`, **không** lộ `T2`; (b)/(c) `401` — ân hạn không cứu phiên hết hạn/khoá |
| TC-RG-12 | **Không lưu bản rõ** (D, S) | Sau 03: `select count(*) from auth_sessions a where a::text like '%'\|\|'<T2>'\|\|'%'` (và `T1`); `docker logs` gateway grep `T1`/`T2`; xem cột mới (`grace_until`, bản mã hoá) | Bản rõ `T1`/`T2` **0** lần ở DB (mọi cột) và log; `grace_until` ≈ lúc xoay + 10 s (±2 s) |
| TC-RG-13 | **Phát hiện dùng lại còn nguyên** (S) | Kẻ trộm giữ `T1`: trình `T1` **sau** 13 s kể từ khi nạn nhân xoay; thứ tự ngược: kẻ trộm xoay trước, nạn nhân mang `T0` cũ sau 13 s | Cả hai thứ tự → thu hồi cả phiên + 1 dòng `refresh_reuse` (AC4 giữ nghĩa ngoài ân hạn). **Ghi nhận rủi ro đã chấp nhận:** trong 10 s kẻ trộm cũng nhận `T2` (không chấm P/F, chỉ ghi) |
| TC-RG-14 | **Contract test** (G) | `cd backend-go && go test -count=1 ./internal/auth/... -run 'TestRefresh' -v` | `ok`; `TestRefreshReuseRevokesSession` và `TestRefreshReuseThiefFirst` đã sửa theo ân hạn; có ca mới cho ân hạn trong/ngoài hạn và 2 đời. **Soi: test không bị nới** (assertion «ngoài hạn → thu hồi» vẫn có) |
| TC-RG-15 | **Hồi quy TC cũ** | Chạy lại TC-P202-07 (xoay, `sid` giữ), 11, 12, 14, 18 (đăng xuất), 25 (≤ 1 s) trên bản mới | TC-P202-11/12/14 kỳ vọng đổi theo SRS mới (ngoài ân hạn mới thu hồi) — ghi rõ phần đổi; 07/18/25 không đổi |
| TC-RG-16 | **Điện thoại** (A, Pixel 7 và iPhone 15 giả lập) | Như 01 ở 375 px | Không `/login`; ngang 0 px tràn |

## Câu hỏi cho BA (chưa có trong SRS — cần trả lời trước khi chạy)

1. Trong ân hạn, **access token** trả về là cái mới (jti mới) hay cùng cái đã cấp? TC-RG-03 chỉ đòi «hợp lệ»; nếu SRS ghi «cùng cặp», TC sẽ đòi giống hệt.
2. Hết 10 s: lần trình `T1` → `SESSION_REVOKED` **reason `refresh_reuse`** và `audit_log.action` viết thường `refresh_reuse` (hiện có trong `session.go`) hay `REFRESH_REUSE` như lệnh PM? TC dùng chữ thường theo mã; cần SRS chốt một.
3. Có cấu hình được độ dài ân hạn (env) không? TC-RG-04/07 cố định 10 s; nếu có env thì VPS phải ghim 10.
4. «Hơn 2 đời» trong lúc đang ân hạn của `T1` (TC-RG-08) → thu hồi: xác nhận đúng ý (chỉ giữ **một** token trước).

## Số ca: 16 (A 4: 01, 02, 05, 16 · S 9: 03, 04, 06–11, 13 · D 1: 12 · G 1: 14 · hồi quy 1: 15). Chưa chạy.
