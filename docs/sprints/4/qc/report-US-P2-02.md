# Báo cáo QC — US-P2-02 (đăng nhập thật, làm mới xoay vòng, đăng xuất, `/login`)
**Kết luận: PASS có điều kiện** — không lỗi chức năng. 2 lệch nhỏ (L1, L2), 3 nhóm TC chờ story sau / hạ tầng (TC-43, TC-49, TC-50/51). Bản chấm `876d695`. Stack riêng của QC: Postgres + Redis + Mailpit, 2 gateway `testroutes` (:8080, :8081) + worker, Chrome for Testing thật, Next `build` ở 3400 trỏ gateway :8080 (không Caddy: `Origin` = `http://localhost:3400`). Seed QC bằng SQL: 7 tài khoản (SV Giỏi, SV Khá, GV, TA, Admin, INVITED, DISABLED), mật khẩu `Edupilot#2026-demo` (bcrypt cost 12). `go test -race -tags testroutes,integration ./...` rc=0 (24 gói), `playwright test` (cổng 3520/3522): **222 passed / 90 skipped / 0 failed**.

## Lệch
- **L1 (TC-48, AC16).** Build `NEXT_PUBLIC_DEV_TOOLS=1` + đăng nhập thật `sv.kha@…`: đặt cookie `ep_demo_person=sv-9; ep_demo_role=teacher` → menu hồ sơ đổi thành "TS. Lê Thu Hà" và nav thành 15 mục (vai GV). Dev đã ghi quyết định này (giữ 155 ca e2e cũ). **Build thường (không DEV_TOOLS): cookie bị bỏ qua** (vẫn "Trần Thu Uyên", 7 mục). TC-48 FAIL ở build gate theo chữ AC16 ("không đọc cookie `ep_demo_*`"), PASS ở build thường. Không rủi ro production; chờ BA xác nhận chấp nhận (Q-QC-P202-1).
- **L2 (TC-46).** `COOKIE_DOMAIN="a b"` (có khoảng trắng): gateway **khởi động bình thường** (không thoát 1). Spec AC15 "giá trị sai → thoát 1 nêu tên biến"; SRS chỉ ghi "rỗng = host hiện tại". 4 biến TTL còn lại thoát `rc=1` nêu tên biến đúng. Đề nghị dev kiểm tên miền cookie hợp lệ.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `"  Sv.Gioi@Edupilot.Local "` → 200 `{access_token, token_type:"Bearer", expires_in:900, user{id,email,full_name,role,status,email_verified}}`; claims `aud, email, exp, iat, iss, jti, nbf, role, sid, sub`, `iss=edupilot`, `aud=edupilot-api`, `exp−iat=900`; `Set-Cookie: ep_rt=<43 ký tự>; Path=/api/v1/auth; Max-Age=1209600; HttpOnly; Secure; SameSite=Lax`; `Cache-Control: no-store`; thân không chứa refresh |
| 02 | PASS | +1 phiên; `last_login_at` null → có giá trị |
| 03 | PASS | HMAC-SHA256 tính lại với `JWT_SECRET_KEY` khớp; `alg=HS256` |
| 04 | PASS | email có thật + sai mật khẩu / email không có / INVITED: **cả ba y hệt** `401 INVALID_CREDENTIALS` "Email hoặc mật khẩu không đúng." (bỏ `trace_id`), cùng bộ header, 0 `Set-Cookie` |
| 05 | PASS | 40 mẫu mỗi loại, trung vị: không tồn tại **231,6 ms**, sai mật khẩu **233,1 ms**, INVITED **231,9 ms** → 99 % / 99 % (≥ 65 %) |
| 06, 10, 15, 17, 19, 24, 28 | PASS | test Go của dev (`-race`), nằm trong chạy toàn bộ rc=0 |
| 07 | PASS | refresh 200: `sid` giữ, `jti` mới, `ep_rt` mới ≠ cũ, thân không chứa refresh |
| 08 | PASS | bản rõ cũ ở `refresh_hash`/`prev_refresh_hash` = 0; `sha256(cũ)` ở `prev_refresh_hash` = 1, `refresh_hash = sha256(mới)` = 1; `rotated_at` có; `pg_dump` và log 2 gateway + worker chứa bản rõ cũ/mới = **0** |
| 09 | PASS | sau refresh `expires` = +336 h, `absolute` = +720 h, `expires ≤ absolute`; ép `absolute` còn 2 h → `expires = absolute` (kẹp đúng); qua trần → `401 TOKEN_INVALID`. (QC ép bằng SQL, không dịch đồng hồ) |
| 11 | PASS | `T1` dùng lại → `401 SESSION_REVOKED` + cookie xoá; `T2` cũng `401 SESSION_REVOKED`; access cũ → `401 SESSION_REVOKED` sau **2 ms** ở gateway 1 và `401` ở gateway 2; `revoked_reason=REFRESH_REUSE` |
| 12 | PASS | kẻ trộm xoay trước (200) → nạn nhân dùng `V1` `401 SESSION_REVOKED` → token của kẻ trộm cũng `401 SESSION_REVOKED` |
| 13 | PASS | mỗi phiên bị tái sử dụng có đúng **1** dòng `audit_log` (`entity=auth_session`, `action=refresh_reuse`); `before`/`after` không chứa token (0), không chứa `@` |
| 14 | PASS | hai refresh song song cùng token: `200` + `401 SESSION_REVOKED`, 0 lỗi 500; phiên bị thu hồi `REFRESH_REUSE` (kết quả "cả phiên thu hồi", theo SRS) — người thắng cũng mất phiên |
| 16 | PASS | không cookie `401 UNAUTHENTICATED`; cookie lạ `401 TOKEN_INVALID` + cookie xoá; hết hạn trượt `401 TOKEN_INVALID` (**phiên không bị thu hồi**); quá hạn tuyệt đối `401 TOKEN_INVALID`; user `DISABLED` `401 SESSION_REVOKED`; 0 phiên tạo thêm |
| 18 | PASS | `logout` hai lần `204`/`204`; refresh A `401 SESSION_REVOKED`; access cũ A `401` sau **5 ms**; B refresh `200`; `revoked_reason=LOGOUT`; cookie xoá |
| 20 | PASS | `Origin: https://evil.example` → refresh `403 {"reason":"origin"}`, logout `403`; refresh hợp lệ sau đó **cùng cookie** vẫn `200` |
| 21 | PASS | `Sec-Fetch-Site: cross-site` không Origin → 403; `same-site` không Origin → 403; không cả hai (curl) → 200; Origin hợp lệ → 200; `Origin: null` → 403; `http://localhost:3400.evil.example` → 403 (khớp chính xác); cổng khác → 403 |
| 22 | PASS | `GET`, `PUT` → 405; `POST text/plain`, `form-urlencoded` → 415; cookie còn dùng được sau các thử sai |
| 23 | PASS | trang giả `http://localhost:9999` (`fetch` credentials include, `no-cors`, form `POST` ẩn tới `/auth/logout`): form POST bị `403 origin`, phiên nạn nhân **không** bị thu hồi, vào `/` vẫn đăng nhập |
| 25 | PASS | thu hồi theo sự kiện (reuse, logout) có hiệu lực ≤ 5 ms ở **cả hai** gateway. Khoá user bằng SQL trực tiếp (không qua API) không phát sự kiện thu hồi: access còn dùng tới hết hạn (≤ 15 phút), nhưng refresh bị `401 SESSION_REVOKED` — đúng thiết kế (chưa có API khoá/đổi mật khẩu, thuộc P2-04/08) |
| 26 | PASS | dừng Redis: access hợp lệ vẫn được chấp nhận (403 = đúng vai không đủ) ở cả 2 gateway, `healthz` 200, đăng nhập mới 200; Redis về → `readyz` 200 |
| 27 | PASS | gateway **production** (không `testroutes`, `APP_ENV=production`): token dev (không `sid`) → `401 TOKEN_INVALID`; token đăng nhập thật → 200 |
| 29 | PASS | ký sai secret, `alg=none`, `iss`/`aud` sai, `exp` quá khứ (`TOKEN_EXPIRED`), `nbf` tương lai, sửa chữ ký: `401`; đổi `sub` sang admin / đổi `role` / đổi `sid` sang phiên người khác (giữ chữ ký cũ): **`401 TOKEN_INVALID`**. (Token ký đúng secret với `sid` lệch `sub` vẫn được nhận — chỉ kẻ có secret làm được; ghi nhận, không tính lỗi) |
| 30, 31, 33 | PASS (spec dev) | `account.spec.ts` "two tabs refresh", `data-layer.spec.ts` "auto refresh": nằm trong 222 passed. QC thử 2 tab bằng tay nhưng công cụ trình duyệt treo — không có số riêng |
| 32 | PASS | thu hồi phiên (`ADMIN`) rồi tải lại `/threads` → `/login?next=%2Fthreads&revoked=admin` + "Bạn đã bị đăng xuất. Hãy đăng nhập lại." |
| 34 | PASS | đăng nhập thật: `localStorage` chỉ có `ep_demo_state` (mock), 0 khoá chứa token/JWT/refresh, `sessionStorage` rỗng, IndexedDB rỗng, `document.cookie` rỗng (`ep_rt` HttpOnly), 0 biến toàn cục, 0 `data-*`/HTML/URL chứa JWT; tải lại → đúng **1** `POST /auth/refresh` 200 và ở lại đúng trang |
| 35 | PASS | `grep DEV_AUTH|Dán token` ở `frontend/src` = 0; build thường: 0 chuỗi `DEV_AUTH|Dán token|Tài khoản mẫu` ở `.next/static`; `ep_demo_role` còn 1 tệp (chờ US-P2-12, như dev ghi) |
| 36 | PASS | xem TC-34 (JWT chỉ trong bộ nhớ) |
| 37 | PASS | `/login`: `Email` (`autocomplete=username`, `inputmode=email`), `Mật khẩu` (`current-password`, nút "Hiện mật khẩu"), **1** nút primary "Đăng nhập", liên kết "Quên mật khẩu?" và "Chưa có tài khoản? Đăng ký"; logo cao **40 px** (1440) / **32 px** (390) |
| 38 | PASS | nhập sai: "Email hoặc mật khẩu không đúng." trong `role=alert`; email **giữ**, mật khẩu **xoá**; nhấp đúp `Đăng nhập` → **1** `POST /auth/login` |
| 39 | PASS | 375: `AUDIT` `ox:0 cut:[] ell:0`, `TOUCH` `[]`; axe (1440 và 390) 0 vi phạm; 0 từ kỹ thuật |
| 40, 42 | PASS (spec dev) | `account.spec.ts` "login page" ×4; TC-42 gốc dùng `vitest` — dev thay bằng `e2e/safe-next.spec.ts` (20 ca) vì vitest không có trong bảng thư viện (proposals #3): chấp nhận, ca đã pass |
| 41 | PASS | 12 `?next=` độc hại (`//evil.example`, `https://evil.example`, `/\evil.example`, `javascript:`, `/%2F%2Fevil.example`, `/%5Cevil.example`, `/foo%0d%0aSet-Cookie`, `/a:b`, `data:`, `///evil.example`, ` //evil.example`) → `/`, không rời origin; `/ok?x=//evil`, `/threads`, `/class/x?tab=a`, `/settings/llm` → giữ đúng |
| 43 | **chờ US-P2-08** | `/admin/courses`, `/admin/users` → **404** với cả 4 vai (route chưa có, dev đã báo). RBAC tương đương đo trên `/admin/llm/*` (TC-45) |
| 44 | PASS | chưa đăng nhập `/threads` → `/login?next=%2Fthreads`; `/settings/llm` → `/login?next=%2Fsettings%2Fllm`; sau đăng nhập nav **SV 7, TA 12, GV 15, Admin 6**; không bộ đổi vai ở build thường |
| 45 | PASS | STUDENT `GET /admin/llm/providers` 403 `{"reason":"role"}`; TEACHER `PUT /admin/llm/routes` 403; TA `GET /admin/users` 404 (route chưa có) |
| 46 | PASS có lệch L2 | `ACCESS_TOKEN_TTL=2h`, `30s`, `REFRESH_TOKEN_TTL=30m`, `SESSION_ABSOLUTE_TTL=1h` → `rc=1` nêu tên biến; mặc định `15m0s / 336h0m0s / 720h0m0s`; `COOKIE_DOMAIN` có khoảng trắng → **không** thoát |
| 47 | PASS | `JWT_EXPIRATION` ngoài tài liệu thiết kế của chính feature này = 0; golden PG không bị sửa (0 dòng xoá); `go test ./internal/contract/...` rc=0; `openapi.yaml` có `/auth/*`; `gateway token` mặc định `exp−iat=900` |
| 48 | FAIL ở build gate / PASS ở build thường | xem L1 |
| 49 | chờ US-P2-12 | cần 6 người mock có tài khoản thật (seed P2-12) để chạy `audit.mjs` bằng đăng nhập |
| 50, 51 | không kiểm được | stack QC không có Caddy (`https://localhost`); `docker-compose.test.yml` dùng chung với phiên khác. Chạy lại khi dựng stack đầy đủ ở cổng P2 |
| 52 | PASS | `go vet`, `golangci-lint` 0 issues, `go test -race` rc=0, contract rc=0; `lint` + `ui-antipatterns` + playwright 222/90/0 |

## Việc sau
Dev: L2. BA: Q-QC-P202-1 (L1 chấp nhận?). QC: chạy lại TC-43 ở US-P2-08, TC-49 ở US-P2-12, TC-50/51 ở cổng P2.

## Chấm lại sau góp ý #3, #4 (PM ACCEPTED, 2026-10-04; bản `bd1404a`)
- TC-22 (AC7): `POST /auth/refresh` không `Content-Type` + `Origin` hợp lệ → `200`; `text/plain` và `form-urlencoded` → `415`; `GET`/`PUT` → `405`; refresh hợp lệ sau các thử sai vẫn `200` → **PASS**. TC-20/21/23 (chéo site) giữ nguyên: không `Origin` + `Sec-Fetch-Site: cross-site` → `403`, `Origin` evil → `403`, `Origin: null` → `403`; không Origin và không Sec-Fetch-Site → `200`.
- TC-42 (AC13): `playwright test safe-next.spec.ts` → 42 passed (20 ca × 2 project + 2) → **PASS** theo lệnh mới.
