# QC test case — US-PG-04 (JWT, 401, RBAC, CourseAccessGuard, bcrypt)
Nguồn: `docs/specs/FEAT-pg-foundation/US.md` v1.1 (US-PG-04 AC1–AC12) + `SRS.md` mục 3.4 (nhánh lỗi), 4.4 (FR-37…FR-44), 6.1 (mã lỗi), 6.2–6.3 (route sản xuất / route thử), 8.1 (env). Hộp đen: chỉ hợp đồng trong spec, không đọc mã dev. Công cụ: **S** = `scripts/pg04.sh` (hàm `tc_pg04_MM` ↔ `TC-PG04-MM`), **T** = tay (story này **không có** TC tay). Chạy `bash docs/sprints/2/qc/scripts/pg04.sh [MM …|--list]` ở gốc worktree sau `pnpm dev`.

Tiền điều kiện chung của story:
- Stack compose `edupilot` đang chạy (`$C ps` đủ service), `https://localhost` trả `readyz` 200, `.env.local` có `JWT_SECRET_KEY` ≥ 32 byte.
- Hai bản gateway (`--scale gateway=2`) theo `tmode`/`dmode`; `lib.sh` tự dựng binary mặc định `$GWBIN` cho `tok` và các TC chạy tiến trình trần.
- **Chế độ test** (`ensure_mode test`, image `*-test` build tag `testroutes`) cho mọi TC chạm `/api/v1/_test/*` (whoami, rbac, courses). **Chế độ default** (`ensure_mode default`) chỉ cho TC-PG04-46.
- TC chạm `/api/v1/jobs/{id}` dùng `UX = 00000000-0000-7000-8000-000000000009` (không có job) — route tồn tại ở **mọi** chế độ nên chuỗi TC-PG04-08…22 không cần dựng lại stack.
- DB có thể có hoặc không có dòng `users`; TC-PG04-32/33 tự chèn và tự xoá user `qc-claim@example.test`.
- `main()` của `lib.sh` tự `rl_reset` + làm mới `$H` trước mỗi TC; mọi TC dừng/pause container đều khôi phục ngay sau lệnh đo (trước khi chấm).
- Token tuỳ ý (`alg` lạ, claim thiếu, `exp` quá khứ) dựng bằng hàm `mkjwt <header-json> <payload-json> [HS256|HS512|none|junk] [secret]` + `pay <giây tới exp> ['<bộ lọc jq>']` khai báo ở đầu `pg04.sh` (openssl + jq, base64url không đệm).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PG04-01 | AC1 | – | **S** `pg04.sh 01` — `tok STUDENT $U1`, giải mã đoạn 1 bằng `jwt_part <t> 1` | 3 đoạn; `alg=HS256`; `typ=JWT`; header đúng 2 khoá `alg,typ` |
| TC-PG04-02 | AC1 | – | **S** `pg04.sh 02` — `tok STUDENT $U1 \| cut -d. -f2 \| tr '_-' '/+' \| base64 -d \| jq -c 'keys'` | `["aud","email","exp","iat","iss","jti","nbf","role","sub"]` (đúng 9 khoá, không thừa) |
| TC-PG04-03 | AC1 | – | **S** `pg04.sh 03` — `jq '.exp-.iat'` với mặc định và `--ttl 10m` / `1m` / `1h` | `900`, `600`, `60`, `3600` |
| TC-PG04-04 | AC1 | – | **S** `pg04.sh 04` — cấp token 4 vai (`ADMIN TEACHER TA STUDENT`) với `--sub $U1`, thêm `TEACHER $U2` | Mỗi token: `role` = đúng vai; `iss="edupilot"`; `aud="edupilot-api"`; `sub`=`$U1` (và `$U2` ở token cuối); `email` khớp `.+@.+` |
| TC-PG04-05 | AC1 | – | **S** `pg04.sh 05` — đọc `iat`,`nbf`,`exp` của token mới cấp, so với `date +%s` | `nbf ≤ iat`; `\|iat − now\| ≤ 120`; `exp > iat`; `iat` là số nguyên |
| TC-PG04-06 | AC1 | – | **S** `pg04.sh 06` — cấp **200** token bằng CLI, trích `.jti` vào `$QC_OUT/tc-pg04-06.jti` | 200 dòng; `sort -u` = **200** (không trùng); **200** dòng khớp `^[A-Za-z0-9_-]{22}$` (128 bit base64url không đệm) |
| TC-PG04-07 | AC1 | Docker cho testcontainers | **S** `pg04.sh 07` — `gt ./internal/auth 'TestJWT_Claims\|TestJWT_JTIUnique'` (10.000 lần cấp) | rc=0 và có `--- PASS:` cho **cả hai** tên; không `--- SKIP`, không `no tests to run` |
| TC-PG04-08 | AC2 | – | **S** `pg04.sh 08` — `GET /api/v1/jobs/$UX` với `tok … --ttl -1m` và `mkjwt` `exp=now−3600` | 401 · `code=TOKEN_EXPIRED` · `WWW-Authenticate: Bearer realm="edupilot"` **có** `error="invalid_token"` · thân đúng 3 khoá `code,message,trace_id` · `trace_id` 32 hex |
| TC-PG04-09 | AC2 | – | **S** `pg04.sh 09` — sai chữ ký 3 cách: đổi 1 ký tự đoạn 3; ký bằng secret khác; đoạn 3 rỗng | Cả 3: 401 · `TOKEN_INVALID` · `WWW-Authenticate` có `error="invalid_token"` · `trace_id` 32 hex |
| TC-PG04-10 | AC2 | – | **S** `pg04.sh 10` — `mkjwt '{"alg":"none","typ":"JWT"}' <payload> none` (chữ ký rỗng) | 401 · `TOKEN_INVALID` (không bao giờ 200) |
| TC-PG04-11 | AC2 | – | **S** `pg04.sh 11` — `alg=HS512`, chữ ký **đúng** theo HS512 bằng chính `$SECRET` (`openssl dgst -sha512 -hmac`) | 401 · `TOKEN_INVALID` (chỉ chấp nhận HS256) |
| TC-PG04-12 | AC2 | – | **S** `pg04.sh 12` — `alg=RS256`, chữ ký rác | 401 · `TOKEN_INVALID` |
| TC-PG04-13 | AC2 | – | **S** `pg04.sh 13` — `pay 900 'del(.exp)'` (HS256 ký đúng, thiếu `exp`) | 401 · `TOKEN_INVALID` |
| TC-PG04-14 | AC2 | – | **S** `pg04.sh 14` — `role="SUPERUSER"`, `role="student"` (chữ thường), thiếu `role` | Cả 3: 401 · `TOKEN_INVALID` |
| TC-PG04-15 | AC2 | – | **S** `pg04.sh 15` — `iss="khac"`, thiếu `iss` | 401 · `TOKEN_INVALID` |
| TC-PG04-16 | AC2 | – | **S** `pg04.sh 16` — `aud="edupilot-web"`, thiếu `aud` | 401 · `TOKEN_INVALID` |
| TC-PG04-17 | AC2 | – | **S** `pg04.sh 17` — `nbf = iat + 600` (tương lai) | 401 · `TOKEN_INVALID` |
| TC-PG04-18 | AC2 | – | **S** `pg04.sh 18` — chuỗi hỏng: 2 đoạn; `garbage`; 4 đoạn; đoạn 2 = `!!!` | Cả 4: 401 · `TOKEN_INVALID` |
| TC-PG04-19 | AC2 | – | **S** `pg04.sh 19` — không gửi `Authorization`; `Basic dTpw`; `Bearer ` (rỗng); `Authorization:` rỗng | Cả 4: 401 · `UNAUTHENTICATED` · `WWW-Authenticate: Bearer realm="edupilot"` **không** có `error=` |
| TC-PG04-20 | AC2 | – | **S** `pg04.sh 20` — 6 token xấu khác nguyên nhân (secret khác, `alg=none`, sai `iss`, sai `aud`, thiếu `exp`, rác), gom `.message` | Cả 6 đều `TOKEN_INVALID`; `sort -u` của `message` = **1** giá trị; `message` không khớp `alg\|HS256\|HS512\|RS256\|signature\|chữ ký\|"iss"\|"aud"\|"nbf"\|"exp"\|base64\|JWT\|claim`; không chứa `eyJ` |
| TC-PG04-21 | AC2 / SRS 3.4 "không truy DB" | – | **S** `pg04.sh 21` — `docker pause $($C ps -q postgres)`; 3 request (hết hạn / rác / không token); `docker unpause` + `wait_ready` ngay | `TOKEN_EXPIRED`, `TOKEN_INVALID`, `UNAUTHENTICATED` (DB chết không đổi phân loại) |
| TC-PG04-22 | AC2 | Docker | **S** `pg04.sh 22` — `gt ./internal/auth 'TestVerify_Table'` rồi đếm `--- PASS: TestVerify_Table/` trong log của `gt` | `--- PASS: TestVerify_Table`; số subtest PASS **≥ 14** (AC yêu cầu ≥ 14 dòng bảng) |
| TC-PG04-23 | AC3 | Docker | **S** `pg04.sh 23` — `gt ./internal/auth 'TestVerify_Leeway'` | `--- PASS: TestVerify_Leeway` |
| TC-PG04-24 | AC3 (biên) | chế độ test | **S** `pg04.sh 24` — `GET /_test/whoami` với `mkjwt` `exp` = now+900 / now−3 / now−4 / now−7 / now−10 | `200` (đối chứng dương), `200`, `200`; rồi `401 TOKEN_EXPIRED`, `401 TOKEN_EXPIRED` (leeway 5 s) |
| TC-PG04-25 | AC4 | chế độ test | **S** `pg04.sh 25` — `docker pause postgres`; `whoami` với token 4 vai (`$U1`) + `STUDENT $U2`; `docker unpause` ngay | Cả 5: `200` và `[.sub,.role]` = `["$U1","<vai>"]`, riêng dòng cuối `["$U2","STUDENT"]` |
| TC-PG04-26 | AC4 ("0 truy vấn DB") | chế độ test | **S** `pg04.sh 26` — đo `pg_stat_database.xact_commit` nền trong 5 s để tính tốc độ nền, rồi đo delta quanh cửa sổ **20** request `whoami` | `delta ≤ nền×(thời gian cửa sổ) + 3` (3 = 2 lệnh psql của phép đo + 1 dung sai); dòng `info` in nền/cửa sổ/delta/ngưỡng |
| TC-PG04-27 | AC4 | Docker | **S** `pg04.sh 27` — `gt ./internal/auth 'TestAuth_NoDBQueryPerRequest'` | `--- PASS: TestAuth_NoDBQueryPerRequest` (bộ đếm tracer = 0) |
| TC-PG04-28 | AC5 (phân quyền) | chế độ test | **S** `pg04.sh 28` — 4 vai × 2 route (`/_test/rbac/admin`, `/_test/rbac/staff`) | ADMIN `200/403`; TEACHER `403/200`; TA `403/200`; STUDENT `403/403` |
| TC-PG04-29 | AC5 | chế độ test | **S** `pg04.sh 29` — không gửi `Authorization` tới cả hai route RBAC | 401 · `UNAUTHENTICATED` (không 403/404) · `WWW-Authenticate` không có `error=` |
| TC-PG04-30 | AC5 | chế độ test | **S** `pg04.sh 30` — thân 403 của STUDENT→`admin` và ADMIN→`staff` | 403 · `code=FORBIDDEN` · `details.reason="role"` · `trace_id` 32 hex · thân đúng 4 khoá `code,details,message,trace_id` · `message` không nêu tên vai |
| TC-PG04-31 | AC5 | Docker | **S** `pg04.sh 31` — `gt ./internal/auth 'TestRBAC_Matrix'` | `--- PASS: TestRBAC_Matrix` |
| TC-PG04-32 | AC6 (phân quyền) | chế độ test; DB chạy | **S** `pg04.sh 32` — `INSERT users(id=$U1, email=qc-claim@example.test, role=ADMIN)` (upsert), gọi `/_test/rbac/admin` bằng token **STUDENT** `$U1`, rồi `DELETE` | `users.role`=`ADMIN` trong DB; request → 403 `FORBIDDEN` `details.reason="role"`; sau dọn `count(*)`=0 |
| TC-PG04-33 | AC6 (chiều ngược) | chế độ test; DB chạy | **S** `pg04.sh 33` — DB `role=STUDENT`, token **ADMIN** `$U1` gọi `/_test/rbac/admin`, rồi dọn | `200` (claim thắng DB cả hai chiều); `count(*)`=0 sau dọn |
| TC-PG04-34 | AC6 | Docker | **S** `pg04.sh 34` — `gt ./internal/auth 'TestRBAC_ClaimWinsOverDB'` | `--- PASS: TestRBAC_ClaimWinsOverDB` |
| TC-PG04-35 | AC7 (phân quyền) | chế độ test | **S** `pg04.sh 35` — `GET /_test/courses/00000000-0000-7000-8000-0000000000aa/ping` với 4 vai + STUDENT `$U2` | Cả 5: `403` (resolver mặc định từ chối tất cả, **kể cả ADMIN**) |
| TC-PG04-36 | AC7 | chế độ test | **S** `pg04.sh 36` — thân 403 của guard (ADMIN và STUDENT) | 403 · `FORBIDDEN` · `details.reason="course"` (khác `"role"` của AC5) · `trace_id` 32 hex |
| TC-PG04-37 | AC7 (biên) | chế độ test | **S** `pg04.sh 37` — `courseId` ∈ {`abc`, `123`, uuid thiếu 1 ký tự, uuid thừa 1 ký tự, `not-a-uuid-at-all`}; rồi uuid hợp lệ | 5 giá trị sai: `404` · `NOT_FOUND` · `trace_id` 32 hex. uuid hợp lệ: `403` (vào tới guard, không 404) |
| TC-PG04-38 | AC7 / AC2 | chế độ test | **S** `pg04.sh 38` — route guard: ẩn danh và `Bearer garbage` | `401 UNAUTHENTICATED` và `401 TOKEN_INVALID` (xác thực chạy trước guard — chờ **Q-QC-04-3**) |
| TC-PG04-39 | AC7 | Docker | **S** `pg04.sh 39` — `gt ./internal/auth 'TestCourseAccessGuard_DefaultDenyAll\|…_Membership\|…_BadID\|…_ResolverError\|…_NoCache'` | 5 dòng `--- PASS:` (resolver giả lớp A/B, lỗi resolver → 503 `SERVICE_UNAVAILABLE`, gọi đúng 1 lần, không cache) |
| TC-PG04-40 | AC8 | Docker | **S** `pg04.sh 40` — `gt ./internal/auth 'TestBcrypt_Format\|TestBcrypt_DefaultCost\|TestBcrypt_TooLong\|TestBcrypt_Check'` + soi log `gt` | 4 dòng `--- PASS:`; tổng dòng `--- PASS: TestBcrypt_` ≥ 4; log **không** chứa `Correct-Horse-9`, không chứa `$2[aby]$NN$` |
| TC-PG04-41 | AC8 (biên env) | Không có tiến trình nào giữ `:8080` trên host | **S** `pg04.sh 41` — chạy `$GWBIN serve` (env tối thiểu 7 biến, DB/Redis không có thật) với `BCRYPT_COST` = 3, 15, 4, 14 | `3` và `15`: rc=**1**, log nêu `BCRYPT_COST`. `4` và `14`: tiến trình **còn sống sau 3 s** (không thoát vì cấu hình) — chờ **Q-QC-04-4** |
| TC-PG04-42 | AC9 | Docker | **S** `pg04.sh 42` — `gt ./internal/auth ./internal/httpapi 'TestAuth_NoSecretsInLogs'` | `--- PASS: TestAuth_NoSecretsInLogs` |
| TC-PG04-43 | AC9 | chế độ test | **S** `pg04.sh 43` — 20 `whoami` + 1 token hết hạn + 1 token rác, chờ 2 s, `$C logs --since 5m gateway worker` | Số dòng log chứa: token đầy đủ = **0**; đoạn chữ ký = **0**; `$SECRET` = **0**; `Correct-Horse-9` = **0**; `$2[aby]$NN$` = **0**; `jti` đầy đủ (22 ký tự) = **0**; `Bearer ` = **0** |
| TC-PG04-44 | AC10 | – | **S** `pg04.sh 44` — `grep -c '/auth/' backend-go/api/openapi.yaml`, tương tự `/me/`, `/admin/` | Tệp tồn tại; cả ba đếm = **0** |
| TC-PG04-45 | AC10 | chế độ test | **S** `pg04.sh 45` — `GET /api/v1/{auth/login,auth/register,auth/refresh,auth/verify,me,me/profile,admin/x,admin/users}` + `POST /api/v1/auth/login` (token ADMIN) | Mọi đường dẫn: `404` · thân JSON `code=NOT_FOUND`; `POST auth/login` = `404` (route thử **không** mở cửa hậu cho P2) |
| TC-PG04-46 | AC10 | **chế độ default** | **S** `pg04.sh 46` — `curl -X POST $GW/api/v1/auth/login` (lệnh Kiểm của AC) + `GET` `/auth/login`, `/me`, `/admin/users`, `/auth/refresh` | `404` ở mọi đường dẫn; thân JSON `NOT_FOUND` + `trace_id` 32 hex |
| TC-PG04-47 | AC11 | `$GWBIN` đã dựng (lib tự dựng) | **S** `pg04.sh 47` — `JWT_SECRET_KEY=$SECRET $GWBIN token --role ADMIN --sub $U1`, stdout/stderr tách riêng | rc=**0**; stdout đúng **1** dòng khớp `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*$`; stderr không chứa token; claim `role=ADMIN` |
| TC-PG04-48 | AC11 (nhánh lỗi) | – | **S** `pg04.sh 48` — `APP_ENV=production JWT_SECRET_KEY=$SECRET $GWBIN token --role ADMIN` | rc=**1**; **0** dòng dạng JWT ở stdout **và** stderr; có ≥ 1 dòng thông báo |
| TC-PG04-49 | AC11 (nhánh lỗi) | – | **S** `pg04.sh 49` — `env -i PATH=$PATH $GWBIN token --role ADMIN` | rc=**1**; thông báo chứa chuỗi `JWT_SECRET_KEY`; 0 dòng dạng JWT |
| TC-PG04-50 | AC11 (nhánh lỗi) | – | **S** `pg04.sh 50` — `--role ROOT`; `--role student`; bỏ hẳn `--role` | `ROOT` rc=**1**; `student` rc=**1**; thiếu `--role` rc≠0; cả ba: 0 dòng dạng JWT |
| TC-PG04-51 | AC11 (nhánh lỗi) | – | **S** `pg04.sh 51` — `--ttl abc`; `--ttl ''` | rc≠0 và không in token (chờ **Q-QC-04-2**) |
| TC-PG04-52 | AC11 / AC1 | – | **S** `pg04.sh 52` — bỏ `--sub`; cấp 2 lần; `--sub $U2 --email qc-tc52@example.test` | Bỏ `--sub` → `sub` khớp regex uuid; hai lần cấp cho hai `jti` khác nhau; `email` và `sub` bằng đúng giá trị cờ |
| TC-PG04-53 | AC11 | Docker | **S** `pg04.sh 53` — `gt ./cmd/gateway 'TestTokenCommand'` | `--- PASS: TestTokenCommand` |
| TC-PG04-54 | AC12 | Docker | **S** `pg04.sh 54` — `gt ./internal/auth 'TestPrincipal_ContextOnly\|TestAuth_ParallelRequests' -count=20` (chạy kèm `-race`) | rc=0; `--- PASS:` cho **cả hai** tên; không dòng `DATA RACE`/`--- FAIL` |
| TC-PG04-55 | AC12 (phân quyền chéo) | chế độ test | **S** `pg04.sh 55` — **200** request `whoami` (5 đợt × 40 song song) bằng 4 token: ADMIN/$U1, TEACHER/$U1, TA/$U2, STUDENT/$U2 | Mỗi nhóm đúng **50** phản hồi; `sort -u` của `sub\|role` mỗi nhóm = đúng **1** giá trị và bằng của chính token đó (không lẫn danh tính giữa request song song) |

## Nhánh lỗi (mỗi dòng SRS 3.4 thuộc story một TC)

| Tình huống SRS 3.4 | Hệ thống phản ứng (SRS) | TC-id |
| --- | --- | --- |
| Token hết hạn / sai — **không truy DB** | 401 `TOKEN_EXPIRED` / `TOKEN_INVALID` / `UNAUTHENTICATED` | Hết hạn: TC-PG04-08, 24. Sai: TC-PG04-09…18, 20, 22. Thiếu/sai kiểu header: TC-PG04-19, 29, 38. **Không truy DB**: TC-PG04-21 (401 đúng mã khi Postgres `pause`), TC-PG04-25, 26 (200 + 0 giao dịch) |
| Sai vai trò hoặc ngoài lớp | 403 `FORBIDDEN` | Sai vai trò: TC-PG04-28, 30, 32, 33 (`details.reason="role"`). Ngoài lớp: TC-PG04-35, 36, 39 (`details.reason="course"`) |

Mã lỗi SRS 6.1 do story này chịu trách nhiệm, TC kích hoạt:

| Status · `code` | TC kích hoạt | Header bắt buộc |
| --- | --- | --- |
| 401 `UNAUTHENTICATED` | TC-PG04-19, 21, 29, 38 | `WWW-Authenticate: Bearer realm="edupilot"` (**không** `error=`) |
| 401 `TOKEN_EXPIRED` | TC-PG04-08, 21, 24 | `… realm="edupilot", error="invalid_token"` |
| 401 `TOKEN_INVALID` | TC-PG04-09…18, 20, 21, 38 | `… realm="edupilot", error="invalid_token"` |
| 403 `FORBIDDEN` (`reason:"role"`) | TC-PG04-28, 30, 32 | — |
| 403 `FORBIDDEN` (`reason:"course"`) | TC-PG04-35, 36 | — |
| 404 `NOT_FOUND` | TC-PG04-37 (`courseId` không uuid), 45, 46 (`/auth/*`, `/me*`, `/admin/*`) | — |
| 503 `SERVICE_UNAVAILABLE` (resolver guard lỗi) | TC-PG04-39 (chỉ test Go — không có cách kích hoạt từ ngoài, xem Điểm khó kiểm) | — |

## Phân quyền
- **Ma trận vai × route** (AC5): TC-PG04-28 (8 ô), TC-PG04-29 (ẩn danh ở cả hai route), TC-PG04-30 (hình dạng 403), TC-PG04-31 (test Go).
- **Claim thắng DB** (AC6): TC-PG04-32 (DB ADMIN + claim STUDENT → 403), TC-PG04-33 (chiều ngược: DB STUDENT + claim ADMIN → 200), TC-PG04-34.
- **Ngoài lớp** (AC7): TC-PG04-35 (mọi vai bị từ chối, kể cả ADMIN — PRD §3), TC-PG04-36, TC-PG04-37, TC-PG04-38, TC-PG04-39.
- **Không lẫn danh tính giữa người dùng** (AC12): TC-PG04-55 (4 token, 2 `sub`, 200 request song song); TC-PG04-25 cũng đối chứng `$U2` ≠ `$U1`.
- **Không làm trùng P2** (AC10): TC-PG04-44, 45, 46 — không có bề mặt `/auth/*`, `/me/*`, `/admin/*` để leo quyền.

## Điểm khó kiểm
- **TC-PG04-24 (leeway) có nguy cơ flaky**: `exp` tính bằng `date +%s` của **host**, gateway so bằng đồng hồ **container**; colima có thể lệch vài trăm ms, cộng độ trễ request. Vì vậy chọn −3 s / −4 s (chấp nhận) và −7 s / −10 s (từ chối), chừa ≥ 1 s mỗi phía biên 5 s thay vì thử đúng −5 s / −6 s. Nếu máy lệch đồng hồ > 1 s, TC này FAIL giả — chạy lại `pg04.sh 24` sau khi đồng bộ giờ colima; biên đúng 5 s chỉ được chứng minh bằng TC-PG04-23 (đồng hồ giả).
- **TC-PG04-26 (0 truy vấn DB)**: `xact_commit` là bộ đếm **toàn DB**, worker poll outbox mỗi 500 ms nên nền ≠ 0. TC đo tốc độ nền 5 s trước rồi chuẩn hoá theo độ dài cửa sổ, cộng 3 (2 lệnh `psql` của chính phép đo + 1 dung sai). Đây là chặn trên, không phải phép đo "chính xác 0"; bằng chứng mạnh cho "0 truy vấn" là TC-PG04-25 (Postgres `pause` mà vẫn 200) + TC-PG04-27 (bộ đếm tracer).
- **`docker pause` (TC-PG04-21, 25)**: cần colima hỗ trợ `pause`/`unpause` (dùng `docker pause $($C ps -q postgres)` đúng như AC). Nếu thất bại → ghi FAIL "KHÔNG KIỂM ĐƯỢC". Cả hai TC gọi `unpause` + `wait_ready` **ngay sau** lệnh đo, trước mọi `chk`, nên stack luôn về trạng thái chuẩn kể cả khi chấm hỏng.
- **TC-PG04-41 chạy `$GWBIN serve` trên host**, không trong container: cần cổng `:8080` của host rảnh (spec không công bố cổng 8080 nên bình thường là rảnh) và dùng env giả (DB/Redis `127.0.0.1:1`) + `STARTUP_TIMEOUT=20s`; nhánh "hợp lệ" chỉ khẳng định *tiến trình không thoát trong 3 s*, vì nối DB thật không phải việc của AC8. Không dùng `gw_env` cho `BCRYPT_COST`/`REQUEST_TIMEOUT`/`RATE_LIMIT_*`: `docker-compose.local.yml` của sprint 1 **chưa có** service `gateway` (dev thêm ở US-PG-07), nên không thể khẳng định compose sẽ chuyển tiếp biến ghi đè — chạy binary trần tránh hẳn rủi ro này.
- **Đổi chế độ tốn thời gian**: `ensure_mode` chỉ dựng lại khi chế độ hiện tại khác, nhưng TC-PG04-46 (default) nằm giữa dãy test-mode → chạy đủ bộ sẽ dựng lại 2 lần (~2–6 phút mỗi lần, có `--build`). Muốn nhanh: chạy `bash pg04.sh 46` riêng ở cuối.
- **Test Go (`gt`)**: cần Docker/colima cho testcontainers; `gt` ép mỗi tên test phải có `--- PASS:` (vì `go test -run X` thoát 0 cả khi không khớp test nào hoặc SKIP). AC10 cho phép `grep` `backend-go/api/openapi.yaml` nên TC-PG04-44 chỉ **đếm**, không đọc nội dung tệp của dev.
- **`-count=20` (AC12)**: `lib._gt` dựng lệnh `go test -race -count=1 … "$@" <pkg>`, cờ thêm nằm **sau** `-count=1` và cờ sau cùng thắng → `gt … -count=20` hợp lệ. Nếu phiên bản Go báo lỗi trùng cờ, chạy tay `(cd backend-go && go test -race -tags testroutes -count=20 -run 'TestPrincipal_ContextOnly|TestAuth_ParallelRequests' ./internal/auth)` và chấm theo dòng `--- PASS:`.
- **AC7 resolver giả / lỗi 503 / NoCache / gọi đúng 1 lần**: route thử chỉ có guard **mặc định từ chối tất cả**; spec không cho cách thay resolver từ ngoài, nên 4 mệnh đề này chỉ đo được bằng TC-PG04-39 (test Go). Hộp đen phủ được: deny-all mọi vai (35), hình dạng 403 (36), 404 cho id sai (37), 401 trước guard (38).
- **AC8 bcrypt không có bề mặt HTTP** (PG không có `/auth/*`): chỉ TC-PG04-40 (test Go) + TC-PG04-41 (biên env) + chứng cứ gián tiếp TC-PG04-43 (không có `$2a$`/`$2b$` trong log). Không kiểm `go.mod` vì AC không nêu.
- **macOS / bash 3.2**: không có `timeout`, `base64 -w0`, `date +%s%N`, `${x,,}`, mảng kết hợp → script dùng `curl --max-time`, `openssl base64 -A`, `now_ms` của lib, `date +%s` (giây). `grep -E` của BSD không hiểu `(?i)` nên các mẫu không phân biệt hoa thường viết bằng lớp ký tự.
- **TC-PG04-55**: 200 request chia 5 đợt × 40 tiến trình song song để không chạm `ulimit -n` của macOS; mỗi đợt `wait`. Tổng 200 < 300 req/phút/IP (hạn mức mặc định qua Caddy) và `main()` đã `rl_reset` trước TC, nên không chạm 429. Các nhóm ghi thêm (`>>`) vào tệp riêng từng vai, dòng < 100 byte nên ghi nối là nguyên tử.
- **TC-PG04-06** cấp 200 token bằng 200 lần gọi binary (~10–20 s). Mốc 10.000 của AC1 nằm ở TC-PG04-07 (test Go) vì gọi CLI 10.000 lần là vô lý về thời gian.
- **`jti` 22 ký tự**: 128 bit mã base64url không đệm = đúng 22 ký tự. Nếu dev chọn base64url **có** đệm (24 ký tự) hay hex (32), TC-PG04-06 FAIL — đúng ý, vì AC1 ghi rõ "128 bit, base64url".

## Câu hỏi cho PM
- **Q-QC-04-1** — `Authorization: bearer <token>` (tên scheme chữ thường). AC2 chỉ liệt kê "không có header, `Basic …`, `Bearer ` rỗng". (a) Chấp nhận, vì RFC 7235 quy định tên scheme không phân biệt hoa thường → 200; (b) Từ chối → 401 `UNAUTHENTICATED`. **Chưa viết TC** cho trường hợp này; sẽ bổ sung một TC khi có trả lời (chờ trả lời).
- **Q-QC-04-2** — `gateway token --ttl abc` (giá trị thời lượng hỏng). (a) rc=1 nêu tên cờ, không in token; (b) bỏ qua cờ và dùng mặc định 15 phút. TC-PG04-51 viết theo (a) (nguyên tắc "cấu hình sai thì thoát 1" của US-PG-01 AC2); **chờ trả lời**.
- **Q-QC-04-3** — Thứ tự xác thực ↔ `CourseAccessGuard` ở `/api/v1/_test/courses/{courseId}/ping` khi **ẩn danh**. (a) 401 `UNAUTHENTICATED` (auth middleware chạy trước guard, giống AC5); (b) 403 `FORBIDDEN` vì guard từ chối tất cả. TC-PG04-38 viết theo (a) (AC2 nói mọi endpoint cần đăng nhập trả 401 khi thiếu header); **chờ trả lời**.
- **Q-QC-04-4** — `BCRYPT_COST` ngoài khoảng 4–14 (SRS 8.1). (a) Thoát mã 1 lúc khởi động, log nêu tên biến (như US-PG-01 AC2); (b) kẹp về mặc định 12 và chạy tiếp. TC-PG04-41 viết theo (a); **chờ trả lời**.
- **Q-QC-04-5** — `gateway token --sub khong-phai-uuid`. AC1 chỉ nói claim `sub` là uuid; AC11 không liệt kê trường hợp này. (a) rc=1; (b) nhận nguyên chuỗi rồi gateway trả `TOKEN_INVALID` khi dùng. TC-PG04-52 **chỉ** khẳng định phần spec nêu (bỏ `--sub` → `sub` vẫn là uuid hợp lệ); **chờ trả lời**.
- **Q-QC-04-6** — AC2 yêu cầu "không nói thêm lý do chi tiết ngoài `code`". `message` của ba mã 401 có được khác nhau theo **mã** không? (a) Mỗi `code` một `message` cố định (3 câu), giống nhau trong cùng mã; (b) một `message` chung cho cả ba. TC-PG04-20 chỉ khẳng định điều kiện yếu hơn và chắc chắn đúng ở cả hai phương án: **mọi `TOKEN_INVALID` có cùng một `message`** và `message` không chứa từ khoá kỹ thuật; **chờ trả lời** trước khi siết thêm.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-02 — viết lần đầu theo US.md v1.1 (đã gồm góp ý #1 route thử = build tag testroutes; tmode/dmode/CT); không có TC nào bị sửa vì chưa có TC cũ.

Tổng: 55 TC (55 tự động, 0 tay).
