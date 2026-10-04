# DEV handoff — US-P2-02 (đăng nhập thật, làm mới xoay vòng, đăng xuất, `/login`)
Nhánh `sprint/4-p2`. **Một phần chưa kiểm được trên stack thật** (xem "Chưa làm / nợ"): AC14 curl `/admin/courses` chờ US-P2-08; chưa chạy tay trên `https://localhost` với stack compose mới.

## Làm gì
**Backend**
- `internal/auth/session.go`: `Sessions` — `Login` (luôn một phép bcrypt, kể cả email lạ; INVITED/không mật khẩu ≡ sai; DISABLED + đúng mật khẩu → 403), `Refresh` (khoá hàng `FOR UPDATE`, xoay `prev_refresh_hash`, trượt `REFRESH_TOKEN_TTL` kẹp ở `absolute_expires_at`, dùng lại ⇒ thu hồi cả phiên + `audit_log refresh_reuse`, user bị khoá ⇒ `SESSION_REVOKED`), `Logout` (idempotent), `Revoked` (Redis `MGET ep:auth:rev:{sid,user}`; Redis chết ⇒ chấp nhận token + log error ≤ 1 lần / 30 s). Không có ân hạn (Q-QC-P202-1): 2 refresh song song ⇒ 1×200 + 1×401 và cả phiên chết (`TestRefreshConcurrentSameToken`).
- `internal/auth/jwt.go`: claim `sid` (`Issuer.IssueSession`; `Issue` dev không có `sid`), `Principal.SessionID/IssuedAt`; `middleware.go`: `WithRevocation`, `RequireSession` (production từ chối token dev).
- `internal/httpapi/authhttp`: `POST /auth/login|refresh|logout`, cookie `ep_rt` (`HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth`, `Max-Age` = hạn còn lại), `Cache-Control: no-store`; lớp CSRF `cookieGuard` (Origin ∈ `CORS_ORIGINS`/`APP_PUBLIC_URL`; không Origin mà `Sec-Fetch-Site` cross/same-site ⇒ 403 `reason=origin`; Content-Type có thì phải JSON).
- Config: `ACCESS_TOKEN_TTL` (thay `JWT_EXPIRATION`; 1 phút–1 giờ), `REFRESH_TOKEN_TTL`, `SESSION_ABSOLUTE_TTL`, `COOKIE_DOMAIN`; `gateway token --ttl` mặc định = `ACCESS_TOKEN_TTL`; `.env.example`, compose, `docs/ARCHITECTURE.md` §8 cập nhật. 6 mã lỗi mới ở `apierr` + `openapi.yaml` (3 thao tác, `SessionResponse`, enum mã) — `internal/contract` golden PG không sửa; thêm `golden/auth/*.json`.
- Migration không đổi (bảng ở US-P2-01). Thêm `InsertAuthSession`, `LockAuthSessionByRefresh`, `RotateAuthSession`, `RevokeAuthSession`, `InsertLoginAttempt`, `TouchLastLogin`.

**Frontend**
- `shared/data/authSession.ts` (máy trạng thái `initializing → authenticated | anonymous | revoked`; `refreshSession()` gộp trong tab + `navigator.locks` `ep-refresh`), `apiClient`: 401 `TOKEN_EXPIRED` ⇒ làm mới đúng 1 lần (gộp) rồi phát lại mỗi request tối đa 1 lần (POST cùng `Idempotency-Key`); `SESSION_REVOKED` ⇒ không làm mới; làm mới hỏng ⇒ xoá token + `auth:expired`.
- `shared/session/AuthProvider` (refresh một lần lúc tải trang), `AuthGate` (khung xương → `/login?next=` hoặc `SessionProvider`), `safeNext`, `session.tsx` lấy vai từ JWT + người mock theo `mock/identity.ts` (SRS 7.4); xoá `TokenGate*`, alias `@ep/token-gate`, `NEXT_PUBLIC_DEV_AUTH`, cổng "Dán token".
- `/login` thật (`AuthShell`, `PasswordInput` mới ở `shared/ui`): lỗi `role=alert`, giữ email / xoá mật khẩu, đếm ngược `LOGIN_THROTTLED`, thông báo bị thu hồi; "Tài khoản mẫu" (cookie `ep_demo_*`) chỉ ở build `NEXT_PUBLIC_DEV_TOOLS=1` (alias `@ep/login-choices`, build thường 0 chuỗi).
- `README`: `http://localhost:3000` không đăng nhập được, dùng `https://localhost` (Caddy + service `frontend` đã có sẵn từ PG).

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `TestLoginSuccess` PASS (cookie đủ cờ, 10 claim, `exp−iat=900`, thân không chứa refresh, email chuẩn hoá, `last_login_at`, `login_attempts`, nhãn thiết bị "Chrome trên macOS") |
| 2 | `TestLoginUniformFailure` (3 tình huống y hệt), `TestLoginDisabledAccount`, `TestLoginTimingEqualized` (bcrypt cost 10, trung vị 20 mẫu, unknown ≥ 65 % wrong) PASS |
| 3 | `TestRefreshRotates`, `TestRefreshSliding`, `TestRefreshAbsoluteCap`, `TestRefreshStoresHashOnly` PASS (đồng hồ giả) |
| 4 | `TestRefreshReuseRevokesSession`, `…ThiefFirst`, `…VictimFirst`, `…Audit`, `TestRefreshConcurrentSameToken` PASS |
| 5 | `TestRefreshMissing/Unknown/Expired/AbsoluteExpired/DisabledUser` PASS |
| 6 | `TestLogout`, `TestLogoutIdempotent`, `TestLogoutOtherDeviceUnaffected` PASS |
| 7 | `TestCSRFOriginRejected`, `TestCSRFSecFetchSite`, `TestCSRFAllowedOrigin`, `TestCookieEndpointsPostJSONOnly` PASS |
| 8 | `TestRevokedSidRejected`, `TestUserCutoffRejected`, `TestRedisDownFailsOpen`, `TestDevTokenNoSid`, `TestDevTokenRejectedInProduction` PASS |
| 9 | `account.spec.ts` "two tabs refresh": 2 tab tải cùng lúc ⇒ ≤ 2 refresh, đỉnh đồng thời = 1 PASS (thật `auth_sessions.revoked_at` chưa kiểm — cần stack) |
| 10 | `data-layer.spec.ts` "auto refresh" (5 GET cùng 401 ⇒ 1 refresh, 5 phát lại; POST cùng Idempotency-Key; refresh hỏng ⇒ `auth:expired`, không phát lại; `SESSION_REVOKED` không làm mới) + `account.spec.ts` "refresh hỏng giữa chừng ⇒ /login?next=" PASS |
| 11 | `account.spec.ts` "no token in storage" (quét storage/cookie; tải lại ⇒ đúng 1 refresh, không đi qua `/login`) PASS; `grep -rn 'DEV_AUTH\|Dán token' frontend/src` = 0; `pnpm build`: 0 chuỗi `Dán token`/`token-gate`/`Tài khoản mẫu` ở `.next/static`. `ep_demo_role` còn trong bundle (mock, xoá ở US-P2-12 theo AC11) |
| 12 | `account.spec.ts` "login page" ×4 (thuộc tính nhập, 1 nút primary, logo 40/32 px, 375 px: `AUDIT_SRC`/`TOUCH_SRC` sạch, lỗi/đang gửi/chờ/thu hồi) PASS |
| 13 | `e2e/safe-next.spec.ts` (20 ca: 6 hợp lệ, 14 độc hại) + "open redirect" PASS — **không dùng vitest** (không có trong bảng thư viện); xem proposals #3 |
| 14 | "nav per role after login" 7/12/15/6 PASS; curl `/admin/courses` 403/403/403/200: **chờ US-P2-08** (route chưa có) — RBAC `RequireRole` đã có sẵn từ PG |
| 15 | `TestLoad…` (3 biến + ngoài khoảng), `gateway serve` với `ACCESS_TOKEN_TTL=2h` thoát 1 nêu tên biến (`TestServe_InvalidEnvNamesVariable`); `grep JWT_EXPIRATION` ngoài `legacy/` và `docs/HUGGINGFACE_DEPLOY.md` (tài liệu Java cũ) = 0; `internal/contract` PASS |
| 16 | "mock identity from session": `sv.kha@…` ⇒ "Trần Thu Uyên"; email lạ ⇒ tên thật; cookie `ep_demo_person` bị bỏ qua PASS. `audit.mjs` của QC 1.5: chưa chạy với đăng nhập thật (cần seed US-P2-12) |
| 17 | Caddy + service `frontend` đã có từ PG (`/api/*` → gateway, còn lại → frontend); `NEXT_PUBLIC_API_URL` trống ở Dockerfile; README ghi rõ. Chưa quan sát tay `POST /auth/refresh` qua `https://localhost` |

Cổng: xem cuối tệp (kết quả chạy).

## Quyết định / lệch spec
- **Content-Type của `refresh`/`logout`:** chỉ từ chối khi header CÓ mà không phải `application/json` (form/text ⇒ 415); cho qua khi không có header (curl không thân ở TC-P202-07). CSRF vẫn chặn bằng Origin / `Sec-Fetch-Site` nên `fetch` no-cors chéo site không lọt.
- **Build `NEXT_PUBLIC_DEV_TOOLS=1`:** nếu có cookie `ep_demo_role` thì dùng phiên mô phỏng và KHÔNG gọi `refresh` lúc tải (giữ nguyên 155 ca e2e cũ dùng `asDemo`); đăng nhập thật xoá cookie đó nên phiên thật thắng.
- Giá trị khoá thu hồi theo phiên là `1`; `details.reason` của `SESSION_REVOKED` ở access token lấy từ `auth_sessions.revoked_reason` (một truy vấn, chỉ trên đường 401).
- Cập nhật test cũ do hợp đồng đổi (không nới lỏng): `serve_test`/`config_test` (`JWT_EXPIRATION`→`ACCESS_TOKEN_TTL`), `contract_test` (18→21 thao tác; `refresh` miễn kiểm "công khai không trả 401" vì thiếu cookie ⇒ 401 là đúng hợp đồng), `shell.spec`/`settings-llm.spec`/`visual.spec`/`data-layer.spec` (cổng dán token → phiên giả lập bằng `page.route`; ca "auth expired" cũ vẫn xanh). 2 ảnh mốc `settings-llm-{1440,390}.png` đổi vì tên người trên thanh trên (trước: email) — cần PM/QC lưu ý; `visual` các route khác không đổi.
- SSE (`/events`) chưa kiểm thu hồi `sid` (handler tự xác thực bằng Verifier): token bị thu hồi vẫn mở được stream tới hết hạn access (≤ 15 phút). Nợ nhỏ; sẽ làm ở story có SSE thật (P4).

## Chưa làm / nợ
- Ca `@real` (đăng nhập qua Caddy, 2 tab thật, tấn công CSRF từ trang giả) chưa chạy — cần stack compose; QC chạy tay.
- Throttle / khoá (US-P2-05), đăng ký / xác minh (US-P2-03), quên mật khẩu / thiết bị (US-P2-04), mời (US-P2-06) còn lại theo kế hoạch; `LOGIN_THROTTLED` mới chỉ có mã lỗi + giao diện đếm ngược.

## Kết quả cổng (đã chạy)
- `make -C backend-go lint test sqlc-check`: 0 issues ×3 (mặc định, `testroutes`, `integration`), toàn bộ gói ok (`-race`), `sqlc diff` rc=0.
- `pnpm -C frontend exec eslint .` + `tsc --noEmit` sạch; `bash scripts/ui-antipatterns.sh` 0 ✗; `bash scripts/lint-selftest.sh` 7/7, 19/19.
- `rm -rf frontend/.next && E2E_API_PORT=3322 pnpm -C frontend build:gate` rồi `E2E_PORT=3320 E2E_API_PORT=3322 env -u CI playwright test`: **222 passed / 90 skipped / 0 failed**. Cổng 3320/3322 là của worktree s4 (đặt bằng `E2E_PORT`/`E2E_API_PORT`, mặc định cũ 3310/3312 không đổi).
