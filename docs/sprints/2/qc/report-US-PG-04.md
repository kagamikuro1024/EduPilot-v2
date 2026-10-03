# QC report — US-PG-04 (auth) · sprint 2 (PG Nền Go) · Kết luận: **PASS**

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `pg04.sh` (lượt đầu → sau triage + sửa script: 55/58 → 58/58). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 58/58 TC PASS, **0 FAIL**.

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG04-01 | PASS | AC1 | token có 3 đoạn = [3] ; header.alg = [HS256] (+2 dòng ok) |
| TC-PG04-02 | PASS | AC1 | keys của payload = [["aud","email","exp","iat","iss","jti","nbf","role","sub"]] — _sửa script QC: dùng `jwt_part` (base64 macOS đòi đệm `=`)_ |
| TC-PG04-03 | PASS | AC1 | exp-iat mặc định = [900] ; exp-iat với --ttl 10m = [600] (+2 dòng ok) |
| TC-PG04-04 | PASS | AC1 | role=ADMIN = [ADMIN] ; iss (vai ADMIN) = [edupilot] (+19 dòng ok) |
| TC-PG04-05 | PASS | AC1 | nbf ≤ iat = 1790973449 (≤ 1790973449) ; \|iat - now\| ≤ 120 (iat=1790973449 now=1790973449) = 0 (≤ 120) (+2 dòng ok) |
| TC-PG04-06 | PASS | AC1 | số token cấp = [200] ; số jti duy nhất = [200] (+1 dòng ok) |
| TC-PG04-07 | PASS | AC1 | go test ./internal/auth -run 'TestJWT_Claims\|TestJWT_JTIUnique' rc = [0] ; --- PASS: TestJWT_Claims (+1 dòng ok) |
| TC-PG04-08 | PASS | AC2 | hết hạn (--ttl -1m) · status = [401] ; hết hạn (--ttl -1m) · code = [TOKEN_EXPIRED] (+12 dòng ok) |
| TC-PG04-09 | PASS | AC2 | chữ ký đổi 1 ký tự · status = [401] ; chữ ký đổi 1 ký tự · code = [TOKEN_INVALID] (+19 dòng ok) |
| TC-PG04-10 | PASS | AC2 | alg=none · status = [401] ; alg=none · code = [TOKEN_INVALID] (+5 dòng ok) |
| TC-PG04-11 | PASS | AC2 | alg=HS512 (chữ ký hợp lệ theo HS512) · status = [401] ; alg=HS512 (chữ ký hợp lệ theo HS512) · code = [TOKEN_INVALID] (+5 dòng ok) |
| TC-PG04-12 | PASS | AC2 | alg=RS256 (chữ ký rác) · status = [401] ; alg=RS256 (chữ ký rác) · code = [TOKEN_INVALID] (+5 dòng ok) |
| TC-PG04-13 | PASS | AC2 | thiếu claim exp · status = [401] ; thiếu claim exp · code = [TOKEN_INVALID] (+5 dòng ok) |
| TC-PG04-14 | PASS | AC2 | role=SUPERUSER · status = [401] ; role=SUPERUSER · code = [TOKEN_INVALID] (+19 dòng ok) |
| TC-PG04-15 | PASS | AC2 | iss="khac" · status = [401] ; iss="khac" · code = [TOKEN_INVALID] (+12 dòng ok) |
| TC-PG04-16 | PASS | AC2 | aud="edupilot-web" · status = [401] ; aud="edupilot-web" · code = [TOKEN_INVALID] (+12 dòng ok) |
| TC-PG04-17 | PASS | AC2 | nbf = iat + 600 · status = [401] ; nbf = iat + 600 · code = [TOKEN_INVALID] (+5 dòng ok) |
| TC-PG04-18 | PASS | AC2 | chỉ 2 đoạn · status = [401] ; chỉ 2 đoạn · code = [TOKEN_INVALID] (+26 dòng ok) |
| TC-PG04-19 | PASS | AC2 | không gửi header Authorization · status = [401] ; không gửi header Authorization · code = [UNAUTHENTICATED] (+26 dòng ok) |
| TC-PG04-20 | PASS | AC2 | mã lỗi = [TOKEN_INVALID] ; mã lỗi = [TOKEN_INVALID] (+8 dòng ok) |
| TC-PG04-21 | PASS | AC2 / SRS 3.4 "không truy DB" | DB dừng · token hết hạn = [TOKEN_EXPIRED\|Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.] ; DB dừng · token hỏng = [TOKEN_INVALID\|Phiên đăng nhập không hợp lệ.] (+1 dòng ok) |
| TC-PG04-22 | PASS | AC2 | go test ./internal/auth -run 'TestVerify_Table' rc = [0] ; --- PASS: TestVerify_Table (+1 dòng ok) |
| TC-PG04-23 | PASS | AC3 | go test ./internal/auth -run 'TestVerify_Leeway' rc = [0] ; --- PASS: TestVerify_Leeway |
| TC-PG04-24 | PASS | AC3 (biên) | token mkjwt còn hạn (đối chứng dương) = [200] ; hết hạn 3 s (trong leeway) = [200] (+15 dòng ok) |
| TC-PG04-25 | PASS | AC4 | DB dừng · whoami vai ADMIN · status = [200] ; DB dừng · whoami ADMIN · [sub,role] = [["00000000-0000-7000-8000-000000000001","ADMIN"]] (+8 dòng ok) |
| TC-PG04-26 | PASS | AC4 ("0 truy vấn DB") | giao dịch DB trong 20 request whoami = 2 (≤ 5) |
| TC-PG04-27 | PASS | AC4 | go test ./internal/auth -run 'TestAuth_NoDBQueryPerRequest' rc = [0] ; --- PASS: TestAuth_NoDBQueryPerRequest |
| TC-PG04-28 | PASS | AC5 (phân quyền) | rbac/admin vai ADMIN = [200] ; rbac/staff vai ADMIN = [403] (+6 dòng ok) |
| TC-PG04-29 | PASS | AC5 | ẩn danh · rbac/admin · status = [401] ; ẩn danh · rbac/admin · code = [UNAUTHENTICATED] (+12 dòng ok) |
| TC-PG04-30 | PASS | AC5 | STUDENT → rbac/admin · status = [403] ; STUDENT → rbac/admin · code = [FORBIDDEN] (+8 dòng ok) |
| TC-PG04-31 | PASS | AC5 | go test ./internal/auth -run 'TestRBAC_Matrix' rc = [0] ; --- PASS: TestRBAC_Matrix |
| TC-PG04-32 | PASS | AC6 (phân quyền) | users.role trong DB = [ADMIN] ; DB=ADMIN, claim=STUDENT → rbac/admin · status = [403] (+4 dòng ok) |
| TC-PG04-33 | PASS | AC6 (chiều ngược) | DB=STUDENT, claim=ADMIN → rbac/admin = [200] ; đã dọn user tạm = [0] |
| TC-PG04-34 | PASS | AC6 | go test ./internal/auth -run 'TestRBAC_ClaimWinsOverDB' rc = [0] ; --- PASS: TestRBAC_ClaimWinsOverDB |
| TC-PG04-35 | PASS | AC7 (phân quyền) | guard · vai ADMIN → 403 = [403] ; guard · vai TEACHER → 403 = [403] (+3 dòng ok) |
| TC-PG04-36 | PASS | AC7 | guard ADMIN · status = [403] ; guard ADMIN · code = [FORBIDDEN] (+6 dòng ok) |
| TC-PG04-37 | PASS | AC7 (biên) | courseId='abc' · status = [404] ; courseId='abc' · code = [NOT_FOUND] (+14 dòng ok) |
| TC-PG04-38 | PASS | AC7 / AC2 | ẩn danh · guard ping · status = [401] ; ẩn danh · guard ping · code = [UNAUTHENTICATED] (+12 dòng ok) |
| TC-PG04-39 | PASS | AC7 | go test ./internal/auth -run 'TestCourseAccessGuard_DefaultDenyAll\|TestCourseAccessGuard_Membership\|TestCourse ; --- PASS: TestCourseAccessGuard_DefaultDenyAll (+4 dòng ok) |
| TC-PG04-40 | PASS | AC8 | go test ./internal/auth -run 'TestBcrypt_Format\|TestBcrypt_DefaultCost\|TestBcrypt_TooLong\|TestBcrypt_Check' rc ; --- PASS: TestBcrypt_Format (+6 dòng ok) |
| TC-PG04-41 | PASS | AC8 (biên env) | BCRYPT_COST=3 · rc = [1] ; BCRYPT_COST=3 · log nêu tên biến (+13 dòng ok) — _sửa script QC: `xargs` đưa VAR=… sau `serve` → dùng `env`_ |
| TC-PG04-42 | PASS | AC9 | go test ./internal/auth ./internal/httpapi -run 'TestAuth_NoSecretsInLogs' rc = [0] ; --- PASS: TestAuth_NoSecretsInLogs — _sửa script QC: `gt` nhận một chuỗi gói_ |
| TC-PG04-43 | PASS | AC9 | log có nội dung để soi = 973 (≥ 1) ; số dòng log chứa token đầy đủ = [0] (+6 dòng ok) |
| TC-PG04-44 | PASS | AC10 | có tệp backend-go/api/openapi.yaml ; grep -c '/auth/' backend-go/api/openapi.yaml = [0] (+2 dòng ok) |
| TC-PG04-45 | PASS | AC10 | GET /api/v1/auth/login (chế độ test) · status = [404] ; GET /api/v1/auth/login · code = [NOT_FOUND] (+15 dòng ok) |
| TC-PG04-46 | PASS | AC10 | POST /api/v1/auth/login = [404] ; GET /api/v1/auth/login · code = [NOT_FOUND] (+4 dòng ok) |
| TC-PG04-47 | PASS | AC11 | rc = [0] ; số dòng stdout = [1] (+3 dòng ok) |
| TC-PG04-48 | PASS | AC11 (nhánh lỗi) | rc = [1] ; số dòng stdout khớp dạng JWT = [0] (+2 dòng ok) |
| TC-PG04-49 | PASS | AC11 (nhánh lỗi) | rc = [1] ; thông báo nêu tên biến JWT_SECRET_KEY (+1 dòng ok) |
| TC-PG04-50 | PASS | AC11 (nhánh lỗi) | --role ROOT · rc = [1] ; --role ROOT · không in token = [0] (+3 dòng ok) |
| TC-PG04-51 | PASS | AC11 (nhánh lỗi) | --ttl 'abc' · rc = [1] ; --ttl 'abc' · thông báo nêu --ttl (+6 dòng ok) |
| TC-PG04-52 | PASS | AC11 / AC1 | bỏ --sub → claim sub là uuid v7 ~ /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a ; hai lần cấp cho hai jti khác nhau = [OAX-VH2xwkETiPGuztVmQw] (≠ [3UnQJBSX-y3oFbFRrA6tUg]) (+2 dòng ok) |
| TC-PG04-53 | PASS | AC11 | go test ./cmd/gateway -run 'TestTokenCommand' rc = [0] ; --- PASS: TestTokenCommand |
| TC-PG04-54 | PASS | AC12 | go test ./internal/auth -run 'TestPrincipal_ContextOnly\|TestAuth_ParallelRequests' rc = [0] ; --- PASS: TestPrincipal_ContextOnly (+1 dòng ok) |
| TC-PG04-55 | PASS | AC12 (phân quyền chéo) | ADMIN · số phản hồi = [50] ; ADMIN · giá trị duy nhất = [00000000-0000-7000-8000-000000000001\|ADMIN] (+6 dòng ok) |
| TC-PG04-56 | PASS | AC2 (v1.2) | scheme 'bearer' · status = [200] ; scheme 'bearer' · danh tính = [00000000-0000-7000-8000-000000000001\|STUDENT] (+4 dòng ok) |
| TC-PG04-57 | PASS | AC11 (nhánh lỗi, v1.2) | --sub 'khong-phai-uuid' · rc = [1] ; --sub 'khong-phai-uuid' · thông báo nêu --sub (+6 dòng ok) |
| TC-PG04-58 | PASS | AC11 / AC1 (v1.2) | lần 1 · sub là uuid v7 ~ /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12 ; lần 2 · sub là uuid v7 ~ /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12 (+2 dòng ok) |

## Lỗi
| Mã | Mức | Nơi | Bước tái hiện | Thấy | Mong đợi | AC / TC |
| --- | --- | --- | --- | --- | --- | --- |
| BUG-PG-1 | **trung bình** | Caddy (`Caddyfile`, `unhealthy_status 503`) | Tắt Redis; gọi `GET /api/v1/events` (hoặc `_test/error/503`) 6 lần liên tiếp qua `https://localhost` | Lần 1–2 `503` JSON `SERVICE_UNAVAILABLE`; lần 3 và 6 là `503` rỗng (không Content-Type, không `X-Instance-Id`, không `Retry-After`) vì Caddy coi cả hai gateway là hỏng | Mọi 503 của ứng dụng (trừ readyz) vẫn là JSON có `Retry-After`; Caddy không loại upstream vì 503 hợp lệ của ứng dụng | US-PG-05 AC15 · TC-PG05-68, US-PG-06 AC5 · TC-PG06-18 |
| BUG-PG-2 | trung bình-thấp | `internal/httpapi/sse/handler.go` | `Last-Event-ID: 99999999999999-0` rồi phát `test.future` | Có `ready`, không nhận sự kiện mới (id live nhỏ hơn Last-Event-ID nên bị loại) | Không `resync`, nhận sự kiện mới | US-PG-05 AC9 · TC-PG05-41 |
| BUG-PG-3 | thấp | `cmd/gateway/serve_test.go:130` (`TestServe_InvalidEnvNamesVariable/JWT_EXPIRATION`) | `go test -race ./...` nhiều lần | Đỏ chập chờn: "log lộ giá trị \"abc\"" vì `trace_id`/hostname ngẫu nhiên chứa "abc" (thấy 1/6 lần ở GATE-03) | Chỉ so giá trị trong trường `msg`, không so toàn dòng | US-PG-01 AC2 |
| BUG-PG-4 | thấp | `cmd/gateway` (chờ phụ thuộc lúc khởi động) | `REDIS_URL` cổng đóng, `STARTUP_TIMEOUT=3s` | 1 dòng warn `dependency not ready` (Redis) thay vì ≥ 2 | "Mỗi giây một dòng" (SRS 3.4) | US-PG-01 AC14 · TC-PG01-76 |
| BUG-PG-5 | thấp | `internal/store/vector_test.go:74,77` | `grep -rn float64 backend-go/internal/store` | 2 dòng `float64` (khoảng cách cosine, không phải điểm) | 0 dòng (TC) — hoặc BA chốt loại tệp `_test` | US-PG-02 AC8 · TC-PG02-48 |
| BUG-PG-6 | thấp (PM chốt) | `backend-go/api/openapi.yaml` | `grep -c _test`; tìm `bearerAuth` trong khối `jobs/{id}`, `events` | Chú thích dòng 7 có chữ `_test`; `bearerAuth` chỉ khai toàn cục (dòng 14) | 0 lần `_test`; khai tường minh theo AC7 (hoặc PM chấp nhận mặc định toàn cục) | US-PG-06 AC2, AC7 · TC-PG06-05, 30 |

Quan sát (không FAIL TC): worker xử lý việc **tuần tự** — việc 100 bước chặn việc mặc định sau nó ~17 s (TC-PG03-104 đã chỉnh thời gian chờ); `Vary` của gateway gửi thành hai dòng riêng.

## Sửa công cụ QC (theo góp ý #4–#10, PM #11) và lỗi script tìm thấy lúc chạy
Mỗi sửa ghi ở cột bằng chứng của TC tương ứng (_sửa script QC: …_). Tóm tắt: #4 literal 31 byte (`pg01.sh` 09); #5 `-w '%{http_code}'` (`pg07.sh` 15); #6 `</dev/null` (`lib.sh` `rl_reset`); #7 `--no-deps` (`pg07.sh` 54); #8 regex `9000(-[0-9]+)?->` (`pg07.sh` 56); #9 bỏ so gián tiếp + `':!legacy'` (`pg01.sh` 78/79, `pg07.sh` 58); #10 `paste -sd' ' -`, gói `httpapi` riêng, `--filter id=` (`gate-pg.sh`). Lỗi script mới (đo tay xác nhận, gateway đúng): `psql` in thêm "INSERT 0 1" (`pg02.sh` 64/65); `pg_dump` token `\restrict` ngẫu nhiên (39); `pg_get_triggerdef` đảo thứ tự sự kiện (37); `xargs` đưa biến môi trường sau `serve` (`pg04.sh` 41); `wait` trần chặn `sse_burst` (`sse-reconnect.sh`; 05/33/35/39); curl 8.7 in `200` khi hết `-m` giữa thân SSE (18/20/21/61); `Connection` bị Caddy bỏ (05-02); slog in level `ERROR` hoa (03-07); `Vary` hai dòng (03-76); mã dồn dòng (03-74); `wait` pid của subshell, `sse_ns` thiếu bỏ "data: ", log `compose run` có dòng không phải JSON (gate); `tab.run` truyền args sai (`sse-browser-cut.mjs`, `idem-two-tabs.mjs`). **Môi trường:** `~/.testcontainers.properties` (README), `brew install sqlc` (1.31.1 = CI); `run-all.sh` chết sau story 01 khi chạy nền — chạy từng script ở foreground.
