# QC report — US-PG-07 (hạ tầng + CI) · sprint 2 (PG Nền Go) · Kết luận: **FAIL**

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `pg07.sh` (lượt đầu → sau triage + sửa script: 61/69 → 68/69). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 68/69 TC PASS, **1 FAIL**.

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG07-01 | PASS | AC1 | danh sách service của docker-compose.local.yml = [caddy frontend gateway mailpit migrate minio pgbouncer postg |
| TC-PG07-02 | PASS | AC1 | service có tên python\|docling = [0] ; chuỗi docling trong compose = [0] |
| TC-PG07-03 | PASS | AC2 | rc của up -d --scale gateway=2 --wait = [0] |
| TC-PG07-04 | PASS | AC2 | số dòng 'gateway running healthy 0' = [2] ; postgres running healthy 0 = [1] (+8 dòng ok) |
| TC-PG07-05 | PASS | AC2 | migrate có FinishedAt thật (không phải mốc rỗng 0001-01-01) ~ /^20[0-9]{12}/ ; d2f561107141e1aca60cdd09eeaf019312e9042579485cb2f2dc0f273317ecad StartedAt (202610022055531012287790) > migrat (+2 dòng ok) |
| TC-PG07-06 | PASS | AC2 | gateway depends_on.migrate.condition = [service_completed_successfully] ; worker depends_on.migrate.condition = [service_completed_successfully] (+2 dòng ok) |
| TC-PG07-07 | PASS | AC3 | mã HTTP = [200] ; status trong thân = [ok] |
| TC-PG07-08 | PASS | AC3 | số lần '<html' ở trang gốc = 1 (≥ 1) ; content-type của / ~ /text/html/ |
| TC-PG07-09 | PASS | AC3 | dòng trạng thái ~ /^HTTP/[0-9.]+ (301\|308)/ ; header location ~ /^https://localhost/?$/ |
| TC-PG07-10 | PASS | AC3 | GET: số header (HSTS\|nosniff) = [2] ; max-age của HSTS ~ /max-age=31536000/ (+1 dòng ok) |
| TC-PG07-11 | PASS | AC3 | content-encoding của / ~ /^(zstd\|gzip)$/ |
| TC-PG07-12 | PASS | AC3 | số header content-encoding của JSON nhỏ = [0] ; content-type của SSE ~ /text/event-stream/ (+1 dòng ok) |
| TC-PG07-13 | PASS | AC3 / SRS 8.2 | header Server ở /api/v1/healthz = [0] ; header Server ở / = [0] |
| TC-PG07-14 | PASS | AC4 | số dòng chứa 'Caddy Local Authority' = 2 (≥ 1) ; SAN DNS:localhost = 1 (≥ 1) (+1 dòng ok) |
| TC-PG07-15 | PASS | AC5 | số lần mã 200 = [40] ; số lần mã khác 200 = [0] (+2 dòng ok) — _sửa script QC: #5/#11 `-w %{http_code}` (HTTP/2)_ |
| TC-PG07-16 | PASS | AC5 | số bản gateway đang chạy = 2 (≥ 2) ; X-Instance-Id [d2f561107141] khớp container (+1 dòng ok) |
| TC-PG07-17 | PASS | AC5 | số lần 404 = [20] ; số lần 401 = [0] |
| TC-PG07-18 | PASS | AC6 (nhánh lỗi) | vòng 1: số dòng không-200 (≤ 2 % của 200) = 1 (≤ 4) ; vòng 1: số dòng không-200 ở 60 dòng cuối (sau ~6 s) = [0] (+4 dòng ok) |
| TC-PG07-19 | PASS | AC6 | trước khi tắt = [404] ; sau khi tắt, cùng token (không 401) = [404] |
| TC-PG07-20 | PASS | AC6 | số container gateway 'running healthy 0' = [2] |
| TC-PG07-21 | PASS | AC7 / SRS 8.3 | pool_mode = [transaction] ; listen_port = [6432] (+10 dòng ok) |
| TC-PG07-22 | PASS | AC7 / AC17 | số cổng công bố của pgbouncer = [0] ; đối chứng: cổng 5433 (postgres) mở được từ host (+1 dòng ok) |
| TC-PG07-23 | PASS | AC7 | psql "$PGB" -Atc 'SHOW VERSION' từ container postgres ; chuỗi phiên bản ~ /pgbouncer/ |
| TC-PG07-24 | PASS | AC8 | gateway db_via = [pgbouncer] ; worker db_via = [pgbouncer] |
| TC-PG07-25 | PASS | AC8 | số client edupilot = 3 (≥ 3) ; số client migrate = [0] |
| TC-PG07-26 | PASS | AC8 | env của migrate dùng Postgres trực tiếp ~ /postgres:5432/ ; env của migrate không trỏ PgBouncer !~ /pgbouncer:6432/ (+2 dòng ok) |
| TC-PG07-27 | PASS | AC9 | go test ./internal/platform/db -run 'TestPgBouncer_TransactionMode\|TestPgBouncer_NoPreparedStatements\|TestVect ; --- PASS: TestPgBouncer_TransactionMode (+2 dòng ok) |
| TC-PG07-28 | PASS | AC9 (hộp đen) | số request trả 404 (có truy vấn DB) = [1000] ; số dòng log gateway có lỗi prepared statement / bind message = [0] — _sửa script QC: #6/#11 `</dev/null`_ |
| TC-PG07-29 | PASS | AC10 | rc của docker build --target gateway = [0] ; rc của docker build --target worker = [0] (+1 dòng ok) |
| TC-PG07-30 | PASS | AC10 | kích thước edupilot-gateway (byte) = 9424356 (≤ 39999999) ; kích thước edupilot-worker (byte) = 8917132 (≤ 39999999) |
| TC-PG07-31 | PASS | AC10 | Config.User của edupilot-gateway ~ /^(nonroot\|65532)(:(nonroot\|65532))?$/ ; Config.User của edupilot-worker ~ /^(nonroot\|65532)(:(nonroot\|65532))?$/ |
| TC-PG07-32 | PASS | AC10 | edupilot-gateway không chạy được sh ; edupilot-gateway không chạy được /bin/sh (+2 dòng ok) |
| TC-PG07-33 | PASS | AC10 / SRS 8.4 | image của service migrate = [edupilot-gateway] ; entrypoint của migrate = [["/gateway","migrate"]] (+1 dòng ok) |
| TC-PG07-34 | PASS | AC11a | số container gateway+worker đã kiểm = 3 (≥ 3) ; số container KHÔNG read_only = [0] |
| TC-PG07-35 | PASS | AC11b | số chỗ ghi đĩa trong mã sản xuất = [0] |
| TC-PG07-36 | PASS | AC11c | rc của golangci-lint run = [0] ; số dòng báo lỗi lint = [0] |
| TC-PG07-37 | PASS | AC11b (động) | số thay đổi hệ thống file của d2f561107141e1aca60cdd09eeaf019312e9042579485cb2f2dc0f273317ecad = [0] ; số thay đổi hệ thống file của e64784be07be9b29fb5d579e2a366af3f6f23f969bd87e8b0d4bf827a519efd6 = [0] (+1 dòng ok) |
| TC-PG07-38 | PASS | AC12 | headSha của run mới nhất = [4bf829cae811a5a19bfe5a6618b63d383219a026] ; conclusion = [success] |
| TC-PG07-39 | PASS | AC12 | tìm được run id ~ /^[0-9]+$/ ; số bước khớp sqlc\|race\|vet\|golangci = 9 (≥ 6) |
| TC-PG07-40 | PASS | AC12 | số bước có 'vet' = 2 (≥ 2) ; số bước có 'golangci' = 4 (≥ 2) (+1 dòng ok) |
| TC-PG07-41 | PASS | AC12 | số job tên chứa Frontend = 1 (≥ 1) ; job Frontend không success = [0] (+1 dòng ok) |
| TC-PG07-42 | PASS | AC12 | số lần 'testroutes' trong ci.yml = 6 (≥ 3) ; số dòng khớp 'legacy\|secrets\.' = [0] |
| TC-PG07-43 | PASS | AC13 (nhánh lỗi) | conclusion của run = [failure] ; headBranch = [ci/sqlc-drift] (+3 dòng ok) |
| TC-PG07-44 | PASS | AC13 (xác thực) | headSha của run = SHA đầu nhánh trên origin = [12532f776fab74512b9bdbb0ab099acee3eeb3ea] ; số tệp internal/store/queries/*.sql bị đổi = 1 (≥ 1) (+1 dòng ok) |
| TC-PG07-45 | **FAIL** | AC13 | **FAIL — KHÔNG KIỂM ĐƯỢC (dev chưa xoá)**: `git ls-remote --heads origin ci/sqlc-drift` vẫn in `12532f7… refs/heads/ci/sqlc-drift`. TC-43/44 đã PASS (run `37024642343` failure đúng step `sqlc diff`; headSha khớp) → dev xoá nhánh rồi chạy lại TC này. |
| TC-PG07-46 | PASS | AC14 | rc của k6 run = [0] ; số ngưỡng ✗ (không đạt) = [0] (+1 dòng ok) |
| TC-PG07-47 | PASS | AC14 | số lần 'p(95)<300' = 3 (≥ 3) ; tập ngưỡng p(95) có trong tệp = [p(95)<300,p(95)<500] (+2 dòng ok) |
| TC-PG07-48 | PASS | AC14 | rc của k6 run (TEST_ROUTES=1, tmode) = [0] ; số ngưỡng ✗ = [0] (+1 dòng ok) |
| TC-PG07-49 | PASS | AC14 | rc của k6 run (TEST_ROUTES=1, dmode) = [107] (≠ [0]) ; đầu ra chứa đúng chuỗi 'TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml' = 1 (≥ 1) |
| TC-PG07-50 | PASS | AC15 | tệp benchmarks/reports/pg-baseline.md tồn tại ; số dòng khớp RAM\|khởi động\|image\|p95 = 9 (≥ 4) (+2 dòng ok) |
| TC-PG07-51 | PASS | AC15 | RAM nghỉ gateway đo được ~ /^[0-9]/ ; RAM nghỉ worker đo được ~ /^[0-9]/ (+2 dòng ok) |
| TC-PG07-52 | PASS | AC16 (nhánh lỗi) | .env.local giống hệt bản gốc ; danh sách missing trong log ~ /JWT_SECRET_KEY/ (+1 dòng ok) |
| TC-PG07-53 | PASS | AC16 | RestartCount lớn nhất sau 60 s = 3 (≤ 3) ; số container gateway còn đang chạy = [0] |
| TC-PG07-54 | PASS | AC16 | migrate FinishedAt không đổi = [2026-10-02T21:05:28.60450107Z] ; trạng thái migrate = [exited 0] — _sửa script QC: #7/#11 `--no-deps`_ |
| TC-PG07-55 | PASS | AC17 (phân quyền / mạng) | số dòng gateway\|worker\|pgbouncer\|migrate có '->' = [0] |
| TC-PG07-56 | PASS | AC17 / SRS 8.4 | tập service có cổng công bố = [caddy frontend mailpit minio postgres redis] ; caddy công bố 80 = 1 (≥ 1) (+6 dòng ok) — _sửa script QC: #8/#11 regex 9000_ |
| TC-PG07-57 | PASS | AC17 | số dòng bí mật trong docker-compose.test.yml = [0] — _sửa script QC: neo regex ở đầu dòng (`${JWT_SECRET_KEY:-}`)_ |
| TC-PG07-58 | PASS | AC17 / 01-AC15 | số biến bí mật trong .env.example (không đo rỗng) = 5 (≥ 1) ; số giá trị bí mật của .env.example KHÔNG chứa '-dev' = [0] (+3 dòng ok) — _sửa script QC: #9/#11_ |
| TC-PG07-59 | PASS | AC17 | GET http://localhost:8080/healthz từ host = [000] ; GET http://localhost:8081/healthz từ host = [000] (+1 dòng ok) |
| TC-PG07-60 | PASS | AC18 | appendonly = [yes] ; appendfsync = [everysec] (+1 dòng ok) |
| TC-PG07-61 | PASS | AC18 | volume postgres = 1 (≥ 1) ; volume redis = 1 (≥ 1) (+3 dòng ok) |
| TC-PG07-62 | PASS | AC18 | giá trị khoá sau khi restart redis = [v1] |
| TC-PG07-63 | PASS | AC19 | PASS — AC19 = TC-GATE-01…13 đều PASS (xem report-GATE-PG). |
| TC-PG07-64 | PASS | AC20 | số 'FROM … AS (gateway-test\|worker-test)' = [2] ; số dòng chứa 'testroutes' = [2] (+1 dòng ok) |
| TC-PG07-65 | PASS | AC20 | số lần 'gateway-test\|worker-test' trong docker-compose.local.yml = [0] |
| TC-PG07-66 | PASS | AC20 | danh sách service khi có override = [caddy frontend gateway mailpit migrate minio pgbouncer postgres redis wor ; số dòng 'image: edupilot-(gateway\|worker)-test' = [2] (+3 dòng ok) |
| TC-PG07-67 | PASS | AC20 | whoami ở image *-test = [200] ; whoami ở image mặc định = [404] |
| TC-PG07-68 | PASS | AC20 | kích thước edupilot-gateway-test (byte) = 9643785 (≤ 39999999) ; kích thước edupilot-worker-test (byte) = 8919488 (≤ 39999999) |
| TC-PG07-69 | PASS | AC20 | số dòng đẩy image (docker push / --push / build-push-action) = [0] ; số dòng vừa nhắc '-test' vừa nhắc push = [0] |

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
