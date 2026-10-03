# QC report — US-PG-03 (HTTP: lỗi, idempotency, cursor, ETag) · sprint 2 (PG Nền Go) · Kết luận: **PASS**

Worktree `TA_Agent_v2-s2`, nhánh `sprint/2-pg` @ `4bf829c` (CI xanh run `37055504838`). Máy: colima, curl 8.7.1, sqlc 1.31.1. Đo 2026-10-03. Script: `pg03.sh` (lượt đầu → sau triage + sửa script: 103/108 → 108/108). Nhật ký thô: `run-logs/`.
**Tóm tắt:** 108/108 TC PASS, **0 FAIL** — lỗi sản phẩm / TC: BUG-PG-3 (test dev chập chờn — thấy ở GATE-03).

## TC
| TC-id | PASS/FAIL | AC | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-PG03-01 | PASS | AC1 | X-Request-Id giữ nguyên = [abc-123] ; X-Instance-Id có mặt ~ /^.+$/ (+1 dòng ok) |
| TC-PG03-02 | PASS | AC1 (biên) | 64 ký tự được giữ = [aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa] ; 65 ký tự bị thay = [58e8efa43383b1910795c5bc0056cac7] (≠ [aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa (+4 dòng ok) |
| TC-PG03-03 | PASS | AC1 | có item để thử 304 ~ /^[0-9a-f-]{36}$/ ; X-Request-Id trên 200 ~ /^.+$/ (+6 dòng ok) |
| TC-PG03-04 | PASS | AC1 | Content-Type 404 = [application/json; charset=utf-8] ; Content-Type 500 = [application/json; charset=utf-8] (+1 dòng ok) |
| TC-PG03-05 | PASS | AC1 (test Go) | go test ./internal/httpapi -run 'TestRequestID' rc = [0] ; --- PASS: TestRequestID |
| TC-PG03-06 | PASS | AC2 | status = [500] ; code = [INTERNAL] (+3 dòng ok) |
| TC-PG03-07 | PASS | AC2 | dòng log cùng trace_id bb90ad1501af5c3c87bd2fe9a808ce2c = 1 (≥ 1) ; dòng log mức error = 1 (≥ 1) (+1 dòng ok) — _sửa script QC: slog in level `ERROR` (hoa)_ |
| TC-PG03-08 | PASS | AC2 | request kế tiếp = [200] ; request thử kế tiếp = [200] (+2 dòng ok) |
| TC-PG03-09 | PASS | AC2 (test Go) | go test ./internal/httpapi -run 'TestRecover' rc = [0] ; --- PASS: TestRecover |
| TC-PG03-10 | PASS | AC3 | mã preflight = [204] ; Access-Control-Allow-Origin = [http://localhost:3000] (+19 dòng ok) |
| TC-PG03-11 | PASS | AC3 | Access-Control-Allow-Credentials (origin trong danh sách) = [true] |
| TC-PG03-12 | PASS | AC3 (nhánh lỗi) | origin lạ: không có ACAO = [] ; origin lạ: không có Allow-Credentials = [] (+5 dòng ok) |
| TC-PG03-13 | PASS | AC3 | ACAO trên GET = [http://localhost:3000] ; Vary có Origin trên GET ~ /Origin/ (+1 dòng ok) |
| TC-PG03-14 | PASS | AC3 (test Go) | go test ./internal/httpapi -run 'TestCORS' rc = [0] ; --- PASS: TestCORS |
| TC-PG03-15 | PASS | AC4 | status 400 = [400] ; code 400 = [BAD_REQUEST] (+76 dòng ok) |
| TC-PG03-16 | PASS | AC4 | retry_after trong thân 429 = [7] ; header Retry-After 429 = [7] (+4 dòng ok) |
| TC-PG03-17 | PASS | AC4 (nhánh lỗi) | status = [404] ; code = [NOT_FOUND] (+2 dòng ok) |
| TC-PG03-18 | PASS | AC4 (nhánh lỗi) | status = [405] ; code = [METHOD_NOT_ALLOWED] (+2 dòng ok) |
| TC-PG03-19 | PASS | AC4 | status không token = [401] ; code = [UNAUTHENTICATED] (+1 dòng ok) |
| TC-PG03-20 | PASS | AC4 (test Go) | go test ./internal/httpapi -run 'TestErrorFormat_AllStatuses\|TestNotFoundAndMethodNotAllowed' rc = [0] ; --- PASS: TestErrorFormat_AllStatuses (+1 dòng ok) |
| TC-PG03-21 | PASS | AC5 (nhánh lỗi) | status = [400] ; code = [BAD_REQUEST] (+1 dòng ok) |
| TC-PG03-22 | PASS | AC5 | status = [422] ; code = [VALIDATION_FAILED] (+2 dòng ok) |
| TC-PG03-23 | PASS | AC5 (biên) | thiếu name: status = [422] ; thiếu name: code = [VALIDATION_FAILED] (+4 dòng ok) |
| TC-PG03-24 | PASS | AC5 (nhánh lỗi) | text/plain: status = [415] ; text/plain: code = [UNSUPPORTED_MEDIA_TYPE] (+3 dòng ok) |
| TC-PG03-25 | PASS | AC5 (test Go) | go test ./internal/httpapi -run 'TestValidation' rc = [0] ; --- PASS: TestValidation |
| TC-PG03-26 | PASS | AC6 | X-RateLimit-Limit = 5 (compose có chuyển tiếp RATE_LIMIT_IP_PER_MIN?) = [5] ; code khi 429 = [RATE_LIMITED] (+3 dòng ok) |
| TC-PG03-27 | PASS | AC6 | có đạt 429 = [429] ; retry_after ≥ 1 = 23 (≥ 1) (+2 dòng ok) |
| TC-PG03-28 | PASS | AC6 | X-RateLimit-Limit là số ~ /^[0-9]+$/ ; X-RateLimit-Remaining request 1 là số ~ /^[0-9]+$/ (+2 dòng ok) |
| TC-PG03-29 | PASS | AC6 | đã vượt giới hạn trước khi thử health = [yes] ; /healthz = [200] (+2 dòng ok) |
| TC-PG03-30 | PASS | AC6 (chậm ≤ 62 s) | cuối cửa sổ cũ bị 429 = [429] ; cửa sổ mới không 429 = [404] (≠ [429]) |
| TC-PG03-31 | PASS | AC6 | U1 request thứ 4 (giới hạn user = 3) = [429] ; code của U1 = [RATE_LIMITED] (+1 dòng ok) |
| TC-PG03-32 | PASS | AC6 | có hai bản gateway ~ /^.+$/ ; số 429 trên 6 request chia hai bản = [1] |
| TC-PG03-33 | PASS | AC6 | XFF 203.0.113.7 request thứ 6 bị 429 ~ / 429/ ; XFF 203.0.113.8 (bộ đếm khác) không 429 !~ / 429/ |
| TC-PG03-34 | PASS | AC6 (test Go) | go test ./internal/httpapi -run 'TestRateLimit_IP\|TestRateLimit_User\|TestRateLimit_SharedAcrossInstances\|TestR ; --- PASS: TestRateLimit_IP (+4 dòng ok) |
| TC-PG03-35 | PASS | AC7 (nhánh lỗi) | request thường vẫn 200 khi Redis chết (fail-open) = [200] ; độ trễ thêm so với baseline (ms) = 31 (≤ 50) (+1 dòng ok) — _sửa script QC: đợi readyz hồi phục (~0,5 s) + đếm log đúng cửa sổ_ |
| TC-PG03-36 | PASS | AC7 (test Go) | go test ./internal/httpapi -run 'TestRateLimit_RedisDown_FailOpen' rc = [0] ; --- PASS: TestRateLimit_RedisDown_FailOpen |
| TC-PG03-37 | PASS | AC8 | số mục mặc định = [30] ; items là mảng = [array] (+3 dòng ok) |
| TC-PG03-38 | PASS | AC8 (biên) | limit=1 = [1] ; limit=100 = [100] |
| TC-PG03-39 | PASS | AC8 (biên, nhánh lỗi) | limit=0 status = [422] ; limit=0 code = [VALIDATION_FAILED] (+13 dòng ok) |
| TC-PG03-40 | PASS | AC8 (nhánh lỗi) | cursor rac status = [422] ; cursor rac code = [INVALID_CURSOR] (+8 dòng ok) |
| TC-PG03-41 | PASS | AC8 | đi được ít nhất 1 trang = 1 (≥ 1) ; trang cuối có trường next_cursor = [true] (+1 dòng ok) |
| TC-PG03-42 | PASS | AC8 | items là mảng = [array] ; items rỗng = [0] (+1 dòng ok) |
| TC-PG03-43 | PASS | AC8 | số id của mình đọc được = [40] ; thứ tự giống psql (created_at DESC, id DESC) = [giong] |
| TC-PG03-44 | PASS | AC8 (test Go) | go test ./internal/httpapi -run 'TestPagination_Defaults\|TestPagination_LimitBounds\|TestPagination_InvalidCurs ; --- PASS: TestPagination_Defaults (+3 dòng ok) |
| TC-PG03-45 | PASS | AC9 | số dòng đã dựng = [250] ; số id đọc được = [250] (+3 dòng ok) |
| TC-PG03-46 | PASS | AC9 | mọi dòng cùng created_at = [1] ; số id đọc được = [300] (+2 dòng ok) |
| TC-PG03-47 | PASS | AC9 (nhánh lỗi) | trang 1 đọc được 20 mục = [20] ; không id nào lặp trong cả lần đi = [0] (+2 dòng ok) |
| TC-PG03-48 | PASS | AC9 (test Go) | go test ./internal/httpapi -run 'TestCursor_NoSkipNoDup\|TestCursor_SameTimestamp\|TestCursor_InsertAndDeleteDur ; --- PASS: TestCursor_NoSkipNoDup (+2 dòng ok) |
| TC-PG03-49 | PASS | AC10 | cursor không rỗng ~ /^.+$/ ; độ dài cursor ≤ 120 = 95 (≤ 120) (+8 dòng ok) |
| TC-PG03-50 | PASS | AC10 | số dòng trong bảng = 10002 (≥ 10000) ; plan có Index Scan / Index Only Scan ~ /Index (Only )?Scan/ (+1 dòng ok) |
| TC-PG03-51 | PASS | AC10 | số dòng có OFFSET (ngoài _test.go) = [0] |
| TC-PG03-52 | PASS | AC10 (test Go) | go test ./internal/httpapi -run 'TestCursor_Format\|TestCursor_UsesIndex' rc = [0] ; --- PASS: TestCursor_Format (+1 dòng ok) |
| TC-PG03-53 | PASS | AC11 | status lần 1 = [201] ; status lần 2 (phát lại đúng 201) = [201] (+5 dòng ok) |
| TC-PG03-54 | PASS | AC11 | status lần 1 = [201] ; status lần 2 = [201] (+3 dòng ok) |
| TC-PG03-55 | PASS | AC11 (test Go) | go test ./internal/httpapi -run 'TestIdempotency_DoubleSend_OneRecord\|TestIdempotency_ReplayIdentical' rc = [0 ; --- PASS: TestIdempotency_DoubleSend_OneRecord (+1 dòng ok) |
| TC-PG03-56 | PASS | AC12 (đua) | số dòng _test_items = [1] ; số phản hồi hợp lệ (replay hoặc 409) = [50] (+3 dòng ok) |
| TC-PG03-57 | PASS | AC12 (đua — "hai tab" PG.md) | số dòng _test_items = [1] ; status tiến trình 1 ~ /^(201\|409)$/ (+3 dòng ok) |
| TC-PG03-58 | PASS | AC12 (nhánh lỗi) | status lần 1 = [201] ; status lần 2 = [422] (+2 dòng ok) |
| TC-PG03-59 | PASS | AC12 (nhánh lỗi) | status = [422] ; code = [IDEMPOTENCY_KEY_REQUIRED] (+1 dòng ok) |
| TC-PG03-60 | PASS | AC12 (biên) | khoá 7 ký tự: status = [422] ; khoá 7 ký tự: code = [VALIDATION_FAILED] (+5 dòng ok) |
| TC-PG03-61 | PASS | AC12 (nhánh lỗi) | khoá 'bad key!': status = [422] ; khoá 'bad key!': code = [VALIDATION_FAILED] (+3 dòng ok) |
| TC-PG03-62 | PASS | AC12 (phân quyền / phạm vi) | U1: status = [201] ; U2: status = [201] (+4 dòng ok) |
| TC-PG03-63 | PASS | AC12 | request được xử lý ~ /^(201\|422)$/ ; số dòng log chứa khoá idem = [0] (+1 dòng ok) |
| TC-PG03-64 | PASS | AC12 (test Go) | go test ./internal/httpapi -run 'TestIdempotency_Concurrent50\|TestIdempotency_KeyReused\|TestIdempotency_Requir ; --- PASS: TestIdempotency_Concurrent50 (+6 dòng ok) |
| TC-PG03-65 | PASS | AC13 | tìm thấy khoá phản hồi của qc65-1790973080083 ~ /^ep:idem:/ ; TTL ≥ 86300 = 86400 (≥ 86300) (+1 dòng ok) |
| TC-PG03-66 | PASS | AC13 (chậm ~45 s) | không bắt được khoá :lock khi đang chạy (handler quá nhanh) — chấm bằng điều kiện hết hạn dưới đây + TC-PG03-6 ; có ≥ 1 phản hồi 409 hoặc replay (chứng tỏ có khoá chạy-dở) = 1 (≥ 1) (+2 dòng ok) |
| TC-PG03-67 | PASS | AC13 | số dòng idempotency_keys sau lần 1 = [1] ; Redis đã sạch khoá ep:idem = [0] (+5 dòng ok) |
| TC-PG03-68 | PASS | AC13 (nhánh lỗi — fail-closed) | status = [503] ; code = [SERVICE_UNAVAILABLE] (+3 dòng ok) |
| TC-PG03-69 | PASS | AC13 (test Go) | go test ./internal/httpapi -run 'TestIdempotency_TTL\|TestIdempotency_RedisFlushedFallsBackToTable\|TestIdempote ; --- PASS: TestIdempotency_TTL (+2 dòng ok) |
| TC-PG03-70 | PASS | AC14 | dựng được item ~ /^[0-9a-f-]{36}$/ ; version ban đầu = [1] (+3 dòng ok) |
| TC-PG03-71 | PASS | AC14 (nhánh lỗi) | status = [409] ; code = [VERSION_CONFLICT] (+4 dòng ok) |
| TC-PG03-72 | PASS | AC14 (nhánh lỗi) | thiếu version: status = [422] ; thiếu version: code = [VALIDATION_FAILED] (+4 dòng ok) |
| TC-PG03-73 | PASS | AC14 | If-Match đúng: status = [200] ; version sau If-Match = [2] (+4 dòng ok) |
| TC-PG03-74 | PASS | AC14 (đua) | số 200 = [1] ; số 409 = [19] (+2 dòng ok) — _sửa script QC: `-w` thiếu `\n` làm dồn mã_ |
| TC-PG03-75 | PASS | AC14 (test Go) | go test ./internal/httpapi -run 'TestOptimisticLock_Stale409\|TestOptimisticLock_Concurrent20\|TestOptimisticLoc ; --- PASS: TestOptimisticLock_Stale409 (+3 dòng ok) |
| TC-PG03-76 | PASS | AC15 | status = [200] ; ETag = [W/"v1"] (+2 dòng ok) — _sửa script QC: `Vary` gửi hai dòng (Origin, Authorization)_ |
| TC-PG03-77 | PASS | AC15 | status = [304] ; thân rỗng (byte tải về) = [0] (+1 dòng ok) |
| TC-PG03-78 | PASS | AC15 | danh sách có ETag hiện tại = [304] ; danh sách W/"v1", W/"v9" (khớp v1) = [304] (+1 dòng ok) |
| TC-PG03-79 | PASS | AC15 | ETag không khớp → 200 = [200] ; ETag đổi sau PUT = [W/"v2"] (≠ [W/"v1"]) (+2 dòng ok) |
| TC-PG03-80 | PASS | AC15 | status = [200] ; định dạng ETag danh sách ~ /^W/"[A-Za-z0-9_-]{16}"$/ (+3 dòng ok) |
| TC-PG03-81 | PASS | AC15 | POST với If-None-Match: * vẫn tạo (201, không 304) = [201] ; PUT với If-None-Match: * vẫn ghi (không 304) = [200] (+1 dòng ok) |
| TC-PG03-82 | PASS | AC15 (test Go) | go test ./internal/httpapi -run 'TestETag_NotModified\|TestETag_ChangesAfterUpdate\|TestETag_MultipleValues' rc  ; --- PASS: TestETag_NotModified (+2 dòng ok) |
| TC-PG03-83 | PASS | AC16 | status = [202] ; job_id là uuid ~ /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/ (+3 dòng ok) |
| TC-PG03-84 | PASS | AC16 | trạng thái trong DB = [SUCCEEDED] ; khoá lạ trong thân = [] (+4 dòng ok) |
| TC-PG03-85 | PASS | AC16 | status POST = [202] ; số lần đo = 12 (≥ 2) (+4 dòng ok) |
| TC-PG03-86 | PASS | AC16 | status POST = [202] ; jobs.status ngay sau POST (worker dừng) = [QUEUED] (+4 dòng ok) |
| TC-PG03-87 | PASS | AC16 (test Go) | go test ./internal/jobs ./internal/httpapi -run 'TestJobs_Lifecycle\|TestJobs_ProgressMonotonic\|TestJobs_Enqueu ; --- PASS: TestJobs_Lifecycle (+2 dòng ok) |
| TC-PG03-88 | PASS | AC17 (nhánh lỗi) | status POST = [202] ; trạng thái = [FAILED] (+6 dòng ok) |
| TC-PG03-89 | PASS | AC17 (nhánh lỗi) | uuid không tồn tại: status = [404] ; uuid không tồn tại: code = [NOT_FOUND] (+2 dòng ok) |
| TC-PG03-90 | PASS | AC17 (test Go) | go test ./internal/jobs -run 'TestJobs_Failed\|TestJobs_PanicFailed' rc = [0] ; --- PASS: TestJobs_Failed (+1 dòng ok) |
| TC-PG03-91 | PASS | AC18 (phân quyền) | U2 (STUDENT khác): status = [404] ; U2: code = [NOT_FOUND] (+5 dòng ok) |
| TC-PG03-92 | PASS | AC18 (phân quyền) | không token: status = [401] ; không token: code = [UNAUTHENTICATED] (+3 dòng ok) |
| TC-PG03-93 | PASS | AC18 (mặc định an toàn) | số đường dẫn trả 404 / 13 = [13] ; đường dẫn không trả 404 = [] (+2 dòng ok) |
| TC-PG03-94 | PASS | AC18 | số đường dẫn trả 404 / 13 = [13] ; đường dẫn không trả 404 (401/403 là lỗi: lộ sự tồn tại) = [] |
| TC-PG03-95 | PASS | AC18 | POST _test/items = [404] ; POST _test/jobs = [404] (+3 dòng ok) |
| TC-PG03-96 | PASS | AC18 | symbol internal/testroutes trong binary mặc định = [0] ; symbol internal/testroutes trong binary -tags testroutes = 32 (≥ 1) |
| TC-PG03-97 | PASS | AC18 | lấy được binary từ image mặc định ; lấy được binary từ image test (+2 dòng ok) |
| TC-PG03-98 | PASS | AC18 (nhánh lỗi) | mã thoát = [1] ; log nêu APP_ENV = 1 (≥ 1) (+3 dòng ok) |
| TC-PG03-99 | PASS | AC18 (test Go, **không** tag) | go test ./cmd/gateway -run 'TestDefaultBinary_NoTestRoutes' rc = [0] ; --- PASS: TestDefaultBinary_NoTestRoutes |
| TC-PG03-100 | PASS | AC8 (phân quyền, #Q-QC-03-2) | dựng được item của U1 ~ /^[0-9a-f-]{36}$/ ; U2 GET item của U1: status = [404] (+2 dòng ok) |
| TC-PG03-101 | PASS | AC14 (phân quyền, #Q-QC-03-2) | U2 PUT item của U1: status = [404] ; U2 PUT item của U1: code = [NOT_FOUND] (+2 dòng ok) |
| TC-PG03-102 | PASS | AC8 (#Q-QC-03-2) | U1 thấy item của mình = [1] ; người dùng khác không thấy item của U1 = [0] (+1 dòng ok) |
| TC-PG03-103 | PASS | AC11 (#Q-QC-03-3) | status = [201] ; Location khớp /api/v1/_test/items/<uuid> ~ /^/api/v1/_test/items/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{ (+3 dòng ok) |
| TC-PG03-104 | PASS | AC16 (biên, #Q-QC-03-4) | steps=1: status = [202] ; steps=100: status = [202] (+4 dòng ok) — _sửa script QC: việc mặc định xếp sau việc 100 bước (worker tuần tự ~17 s) → thăm 600 lần_ |
| TC-PG03-105 | PASS | AC16 (nhánh lỗi, #Q-QC-03-4) | steps=0: status = [422] ; steps=0: code = [VALIDATION_FAILED] (+14 dòng ok) |
| TC-PG03-106 | PASS | AC16 (#Q-QC-03-4) | thiếu kind: status = [202] ; thiếu kind: kind trong DB = [test.progress] (+4 dòng ok) |
| TC-PG03-107 | PASS | AC14 (nhánh lỗi, #Q-QC-03-5) | version=0: status = [422] ; version=0: code = [VALIDATION_FAILED] (+11 dòng ok) |
| TC-PG03-108 | PASS | AC15 (#Q-QC-03-6) | dựng được 2 item ~ /^[0-9a-f-]{72}$/ ; next_cursor có mặt khi còn trang ~ /^[A-Za-z0-9_-]+$/ (+5 dòng ok) |

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
