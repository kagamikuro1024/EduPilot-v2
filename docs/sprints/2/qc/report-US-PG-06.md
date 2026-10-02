# QC report — US-PG-06 (OpenAPI + contract) · sprint 2 (PG Nền Go) · Kết luận: **PASS** (vòng sửa 1, 2026-10-03)

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `pg06.sh` (lượt đầu → sau triage + sửa script: 34/38 (+1 MANUAL) → 35/38). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 38/38 TC PASS, **0 FAIL** sau vòng sửa 1 (lỗi sản phẩm BUG-PG-1/2/3/4/6 dev đã sửa; BUG-PG-5, bearerAuth, `time.Sleep` drain: PM chốt #12 — spec v1.5; TC-PG05-41: #13).

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG06-01 | PASS | AC1 | redocly lint rc = [0] ; đầu ra redocly không có lỗi !~ /[1-9][0-9]* error/ |
| TC-PG06-02 | PASS | AC1 | go test ./internal/contract -run 'TestSpec_LoadsAndValidates' rc = [0] ; --- PASS: TestSpec_LoadsAndValidates |
| TC-PG06-03 | PASS | AC1 (Q5) | openapi: của openapi.yaml ~ /^(3\.1\.0\|3\.0\.3)$/ ; openapi: của openapi.test.yaml ~ /^(3\.1\.0\|3\.0\.3)$/ (+1 dòng ok) |
| TC-PG06-04 | PASS | AC2 | grep -cE '^  /' api/openapi.yaml = [5] ; danh sách đường dẫn = [/api/v1/events /api/v1/healthz /api/v1/jobs/{id} /api/v1/readyz /healthz] |
| TC-PG06-05 | PASS | AC2 | Vòng sửa 1 (BUG-PG-6): `grep -c _test api/openapi.yaml` = 0; `/healthz` giữ 1. |
| TC-PG06-06 | PASS | AC2 / SRS 6.3 | grep -cE '^  /api/v1/_test' api/openapi.test.yaml = [13] ; 13 đường dẫn thử (SRS 6.3, chuẩn hoá {x}→{}) = [/api/v1/_test/courses/{}/ping /api/v1/_test/db-sleep /api/v1/_ (+13 dòng ok) |
| TC-PG06-07 | PASS | AC2 | số thao tác openapi.yaml = [5] ; số operationId openapi.yaml = [5] (+3 dòng ok) |
| TC-PG06-08 | PASS | AC3 | gotest ./internal/contract/... rc = [0] ; có gói contract chạy thật ~ /ok[ \t]+.*internal/contract/ (+1 dòng ok) |
| TC-PG06-09 | PASS | AC3 | go test ./internal/contract -run 'TestSpec_LoadsAndValidates\|TestRouteSpecParity\|TestStatusParity\|TestSpec_Err ; --- PASS: TestSpec_LoadsAndValidates (+7 dòng ok) |
| TC-PG06-10 | PASS | AC3 | go test (không tag) ./internal/contract/... rc = [0] ; có gói contract chạy thật ~ /ok[ \t]+.*internal/contract/ |
| TC-PG06-11 | PASS | AC3 | go test ./internal/contract -run 'TestSpec_LoadsAndValidates\|TestRouteSpecParity\|TestStatusParity\|TestSpec_Err ; --- PASS: TestSpec_LoadsAndValidates (+7 dòng ok) |
| TC-PG06-12 | PASS | AC3 (đối chứng 404) | số (method, đường dẫn) thử đã gọi = 15 (≥ 15) ; số phản hồi khác 404 ở image mặc định = [0] |
| TC-PG06-13 | PASS | AC7 (thật) | GET /api/v1/jobs/{id} không token = [401] ; code của thân lỗi = [UNAUTHENTICATED] (+3 dòng ok) |
| TC-PG06-14 | PASS | AC7 (thật) | GET /healthz ẩn danh = [200] ; GET /api/v1/healthz ẩn danh = [200] (+1 dòng ok) |
| TC-PG06-15 | PASS | AC3 (thật) | GET /healthz = [{"status":"ok"}] ; GET /api/v1/healthz = [ok] (+7 dòng ok) |
| TC-PG06-16 | PASS | AC3 (header bắt buộc) | spec khai ETag = 7 (≥ 1) ; spec khai Location = 2 (≥ 1) (+5 dòng ok) |
| TC-PG06-17 | PASS | AC3 (phân trang) | kiểu của items = [array] ; số phần tử (limit=1) = [1] (+3 dòng ok) |
| TC-PG06-18 | PASS | AC5 (thật) | Vòng sửa 1 (BUG-PG-1): 8 status gọi thật qua Caddy, đúng `.code`; 429 và 503 có `.retry_after`=7 và header `Retry-After` — hết 503 rỗng ở lần gọi thứ 3. |
| TC-PG06-19 | PASS | AC7 (thật, RBAC) | rbac/admin không token = [401] ; WWW-Authenticate ~ /^Bearer realm="edupilot"/ (+4 dòng ok) |
| TC-PG06-20 | PASS | AC4 (đối chứng dương) | rc (spec nguyên vẹn) = [0] ; có --- PASS: TestRouteSpecParity = 1 (≥ 1) |
| TC-PG06-21 | PASS | AC4 (lệch: thiếu ở spec) | đường dẫn còn lại sau khi xoá = [0] ; rc (spec thiếu readyz) = [1] (≠ [0]) (+2 dòng ok) |
| TC-PG06-22 | PASS | AC4 (không tag) | rc (không tag, spec thiếu readyz) = [1] (≠ [0]) ; có --- FAIL: TestRouteSpecParity = 1 (≥ 1) (+1 dòng ok) |
| TC-PG06-23 | PASS | AC4 (lệch: thừa ở spec) | spec thử có đường dẫn thừa = [1] ; rc (spec thừa đường dẫn) = [1] (≠ [0]) (+2 dòng ok) |
| TC-PG06-24 | PASS | AC4 (lệch: status thật không có trong spec) | spec gốc khai 404 cho jobs/{id} = 1 (≥ 1) ; bản sửa đã xoá 404 = [0] (+4 dòng ok) |
| TC-PG06-25 | PASS | AC5 | components.responses.Unauthorized = [1] ; components.responses.Forbidden = [1] (+6 dòng ok) |
| TC-PG06-26 | PASS | AC5 (ánh xạ status) | status 401 → responses/Unauthorized = 13 (≥ 1) ; status 403 → responses/Forbidden = 5 (≥ 1) (+6 dòng ok) |
| TC-PG06-27 | PASS | AC5 | số $ref schemas/Error trong components.responses = 8 (≥ 8) ; định nghĩa schema Error = [1] |
| TC-PG06-28 | PASS | AC5 | go test ./internal/contract -run 'TestSpec_ErrorResponsesDeclared\|TestSpec_EveryDocumentedStatusExercised' rc  ; --- PASS: TestSpec_ErrorResponsesDeclared (+4 dòng ok) |
| TC-PG06-29 | PASS | AC6 (đối chứng âm) | go test ./internal/contract -run 'TestValidator_RejectsBadResponses' rc = [0] ; --- PASS: TestValidator_RejectsBadResponses |
| TC-PG06-30 | PASS | AC7 (spec) | Spec v1.5 / #12: `bearerAuth` hiệu lực (khai tường minh hoặc kế thừa khoá gốc tài liệu) và có `401` cho `/api/v1/jobs/{id}` và `/api/v1/events`. TC và `pg06.sh` sửa theo #12. |
| TC-PG06-31 | PASS | AC7 (spec) | /healthz khai security: [] = [1] ; /healthz không khai bearerAuth = [0] (+4 dòng ok) |
| TC-PG06-32 | PASS | AC7 (spec) | /api/v1/_test/rbac/admin khai 403 = 2 (≥ 1) ; /api/v1/_test/rbac/admin khai 401 = 2 (≥ 1) (+2 dòng ok) |
| TC-PG06-33 | PASS | AC7 | go test ./internal/contract -run 'TestSpec_SecurityDeclared\|TestContract_UnauthenticatedMatchesSpec' rc = [0] ; --- PASS: TestSpec_SecurityDeclared (+1 dòng ok) |
| TC-PG06-34 | PASS | AC8 (D52) | số gói phụ thuộc (không tag) = 505 (≥ 50) ; số gói phụ thuộc (-tags testroutes) = 506 (≥ 50) (+2 dòng ok) |
| TC-PG06-35 | PASS | AC8 | kin-openapi trong go list -deps -test ./internal/contract = 5 (≥ 1) |
| TC-PG06-36 | PASS | AC8 (lint + đối chứng âm depguard) | golangci-lint run rc = [0] ; không vi phạm depguard !~ /depguard/ (+6 dòng ok) |
| TC-PG06-37 | PASS | AC8 (đối chứng trên binary) | số dòng dep của gateway (notag) = 43 (≥ 5) ; getkin/kin-openapi trong gateway (notag) = [0] (+6 dòng ok) |
| TC-PG06-38 | PASS | AC5 (đối chiếu handoff) | Danh sách miễn trừ `exempt.go` rỗng (0 mục) = handoff "miễn trừ — rỗng"; mọi status gọi thật ở TC-PG06-18/19. |

## Lỗi
_Vòng sửa 1: BUG-PG-1, 2, 3, 4, 6 dev đã sửa và QC chạy lại — **đóng**; BUG-PG-5 không phải lỗi (PM chốt #12, spec v1.5). Bảng dưới là hồ sơ vòng 1._

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
