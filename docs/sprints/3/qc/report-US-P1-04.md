# Báo cáo QC — US-P1-04 (API cấu hình LLM: nhà cung cấp, Test, tuyến, mức dùng, ngân sách)
**Kết luận: FAIL** — 1 lỗi (BUG-P104-1: số quá lớn làm `PUT` ngân sách / `POST` nhà cung cấp trả `500 INTERNAL` thay vì `422`; TC-P104-34 FAIL). 47/48 TC PASS; mọi AC còn lại đạt, gồm RBAC 78 ca, canary khoá API (0 ở mọi nơi), nạp nóng 11–16 ms, contract/golden, SLO đọc.

- Bản chấm: `216ad45` (chứa `2234558`) trong worktree QC `../TA_Agent_qcp1` (đã gỡ). 2 gateway `-tags testroutes` native, Postgres/Redis/MinIO riêng, **`FAKE_LLM_VALID_KEY=good-key` đặt bằng env của stack test** (góp ý #10: `fake` chạy trong gateway nên khoá đúng nằm ở env, không phải DB). Máy chủ OpenAI giả của QC (`scripts/p1-run/qserver.mjs`, thêm chế độ trả `200` thân thô cho ca `BAD_RESPONSE`).
- Q-QC-P104-1 (SSRF): chờ BA; QC ghi hành vi thực (TC-14). Q-QC-P104-2 (giới hạn Test): SRS 6.2 / `ep:llm:test:{user}` = **10 lần/phút**; đo đúng 10. Q-QC-P104-3: BA đã trả lời (QC viết `scenario-P1.md`) → đã viết.

## Lỗi
### BUG-P104-1 — số vượt biên của cột `numeric` trả `500 INTERNAL` (AC11 "mã lỗi chuẩn", TC-P104-34)
Tái hiện (ADMIN, bất kỳ gateway):
1. `GET /admin/llm/budget` lấy `version`.
2. `PUT /admin/llm/budget {"daily_limit":"99999999999999999999999","monthly_limit":"99999999999999999999999","version":<v>}`.
Thực tế: `500 {"code":"INTERNAL","message":"Đã xảy ra lỗi. Hãy thử lại sau."}`; log gateway `llmconfig: sửa ngân sách: ERROR: numeric field overflow (SQLSTATE 22003)`. Mong đợi: `422 VALIDATION_FAILED` `details[].field=daily_limit`, `code=OUT_OF_RANGE` như ca số âm / `daily > monthly`.
Cùng nguyên nhân (đã xác nhận): `PUT /courses/{id}/llm-budget` với `99999999999999999999`; `POST /admin/llm/providers` có `models[].price_in:"99999999999999999999"` → `500`.
Ca lành cùng nhóm đạt: số âm → `422`, giá âm → `422`, `rpm_limit` 0 / âm → `422`, tên 1000 ký tự → `422`, 101 mô hình → `422`.
Việc cần dev: chặn biên trên (đúng độ rộng cột) ở tầng kiểm đầu vào; thêm một dòng vào `TestBudgetPutRules` / `TestProviderCRUD`.

## Ghi chú không thành lỗi (BA/PM đọc)
- **TC-14 (SSRF, chờ BA):** theo AC13 `base_url` nội bộ **không bị chặn**. Đo: `Test`/`POST` tới `http://169.254.169.254/`, `http://localhost:45432/v1` (Postgres), `http://127.0.0.1:46379/v1` (Redis) → gateway **có kết nối** (`NETWORK`, 4–11 ms), `POST` trả `422 PROVIDER_UNREACHABLE` và không lưu; không lộ nội dung phản hồi nội bộ. Rủi ro còn lại: ADMIN phân biệt được `ok` / `NETWORK` / `BAD_RESPONSE` / `TIMEOUT` ⇒ quét cổng nội bộ mù. Chấp nhận theo AC13; ghi để PM biết.
- Ngân sách: JSON số (`100000`, không phải chuỗi) và `"1e3"` được nhận; `"100.12345"` bị làm tròn im lặng thành `100.12`; `"abc"` → `400 BAD_REQUEST` (không `422` kèm `field`). SRS nói tiền là **chuỗi** ở phản hồi (đúng) — đầu vào không bị cấm; để BA quyết có nên cấm.
- JWT ký đúng secret nhưng `sub` không phải UUID (`abc`): `GET` và `PUT budget` vẫn `200`, `POST providers` → `401 UNAUTHENTICATED` (không nhất quán; cần khoá ký của gateway nên không khai thác được từ ngoài). Để dev xem xét.
- Tên nhà cung cấp: `idm` sau `IDM` được tạo (`201`); chỉ trùng **y hệt** mới `409 CONFLICT`. Spec không nói phân biệt hoa/thường.
- `GET routes.embedding.reindex_required` luôn `false` sau khi đổi mô hình (không có chunk; `indexed_chunks:null`); chỉ phản hồi `PUT` mang `true`. Đúng với SRS 6.3 (`PUT` trả cờ), `GET` chờ P8.
- Đọc ngay sau `PUT` ở cùng gateway có thể còn thấy cấu hình cũ ≈ 10 ms (1 yêu cầu).
- `fallback_index` đếm trong chuỗi đang bật (xem `scenario-P1.md`).
- `redocly lint`: hợp lệ, 0 lỗi, **8 cảnh báo** (`no-unused-components`: `DeadlineExceeded`, …) trên `openapi.yaml`, 9 trên `openapi.test.yaml` — không lỗi.
- Caddy: không dựng trong stack QC; kiểm tĩnh `deploy/caddy/Caddyfile` không có chỉ thị `log` (không ghi log truy cập / thân). Log gateway: `0` dòng chứa `authorization|api_key|Bearer`.
- Chưa chạy `scripts/canary-scan.sh` của dev (cần compose); QC tự quét độc lập (TC-16).

## Kết quả từng TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | 13 thao tác × {ADMIN, TEACHER, TA, STUDENT, không JWT, JWT hết hạn}: ADMIN `200`/`422` hợp lệ; TEACHER `200` ở 4 `GET` đọc (`providers`, `routes`, `usage`, `budget`), mọi thao tác khác `403 FORBIDDEN details.reason=role`; TA, STUDENT `403` mọi thao tác; không JWT `401 UNAUTHENTICATED`; hết hạn `401 TOKEN_EXPIRED`; DB không đổi sau các ca bị từ chối |
| 02 | PASS | 15 lần gọi bị `403` (thân rác, thân dài, sai `Content-Type`): `fake_calls` không đổi; số lần quét/ghi các bảng `llm_*`, `audit_log` **+0**; thân rác không bị đọc (vẫn `403`, không `400/415`) |
| 03 | PASS | JWT `alg=none` (có/không chữ ký), ký sai secret, sửa `role`→ADMIN giữ chữ ký cũ, `a.b.c`, rác, `Bearer ` rỗng: tất cả `401` (`TOKEN_INVALID` / `UNAUTHENTICATED`) ở `GET providers`, `PUT routes`, `PUT budget`; DB không đổi. Ký **đúng secret** nhưng `role` = `SUPERUSER` / `admin` thường / rỗng / thiếu, thiếu `sub`, `nbf` tương lai → `401 TOKEN_INVALID`; `exp` quá hạn → `401 TOKEN_EXPIRED`; token đối chứng hợp lệ → `200` |
| 04 | PASS | STUDENT thêm `X-Role: ADMIN`, `X-Forwarded-Role`, `X-User-Id` giả, `?role=ADMIN`, thân `{"role":"ADMIN"}` → `403` ở mọi ca |
| 05 | PASS | `TestRBACMatrix` ok (13 × 4 vai + 2 = 78 ca theo mã nguồn) |
| 06 | PASS | khoá `bad-key`: `200 ok:false error_kind=AUTH` + câu "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."; `llm_providers` không thêm dòng; `audit_log` +1 |
| 07 | PASS | `good-key`: `200 ok:true latency_ms=0`; không lưu bản ghi |
| 08 | PASS | 7 loại đo bằng máy chủ Q: `AUTH` (401, 403), `NETWORK` (cổng đóng; 500 → `NETWORK`), `RATE_LIMIT` (429), `MODEL_NOT_FOUND` (404), `BAD_RESPONSE` (thân `<html>`, `{"foo":1}`, `choices:[]`, rỗng, SSE cho chat không-stream, embed không có `data`; **307** không theo: đích 0 request), `TIMEOUT` (treo 12 s → **10,006 s**), `DIMS_MISMATCH` (768 chiều). **7/7 câu tiếng Việt khác nhau**; không thân lỗi nhà cung cấp trong phản hồi |
| 09 | PASS | mỗi Test +1 dòng `audit_log` (`{"ok","host","type","model","error_kind"}`, không khoá); Test bản đã lưu cập nhật `last_test`; Test có ghi đè khoá **không** ghi đè `last_test`; **10 lần/phút/người**: lần 11–13 → `429 RATE_LIMITED` `retry_after=60` + header `Retry-After: 60`; admin khác không bị; cùng người sang gateway thứ hai và `/{id}/test` vẫn `429` (Redis dùng chung) |
| 10 | PASS | `TestProviderTestEndpoint`, `TestTestRateLimit` |
| 11 | PASS | `POST` khoá sai → `422 VALIDATION_FAILED` `details=[{field:api_key,code:PROVIDER_AUTH_FAILED,…}]`; `GET` không có "F"; `llm_providers` 1→1 |
| 12 | PASS | `PUT` khoá sai → `422`; `md5(api_key_enc)` và `version` y nguyên; chat qua nhà đó vẫn `200` |
| 13 | PASS | `skip_verify` ADMIN → `201`, `last_test={ok:null,at:null}`, `audit_log.after.skip_verify=true`; TEACHER → `403`; `PUT` đổi giá/`rpm` không khoá → `200`, `fake_calls` không đổi |
| 14 | PASS* | xem ghi chú SSRF: `169.254.169.254`, `localhost:45432`, `127.0.0.1:46379`, `redis:6379` → `422 PROVIDER_UNREACHABLE`, 0 dòng lưu, không lộ nội dung; 8 `base_url` sai cú pháp (`ftp:`, `javascript:`, `user:pw@`, `#frag`, rỗng host, > 300 ký tự…) → `422 INVALID_BASE_URL` |
| 15 | PASS | `TestSaveVerifiesFirst`, `TestSaveSkipVerify`, `TestUpdateWithoutKeyNoVerify` |
| 16 | PASS | canary `sk-LEAK-CANARY-7f3a9c1e` qua tạo (kèm máy chủ phản chiếu khoá trong thân `401`), `skip_verify`, đổi khoá, `Test` ứng viên / đã lưu, kiểu sai (`type` lạ, `api_key` số), JSON hỏng, `Content-Type` sai, `413` mang khoá, xoá: **0** lần xuất hiện ở 14 phản hồi, log 2 gateway (`7f3a9c1e`, `LEAK-CANARY`, `Bearer sk-`), `pg_dump` toàn DB, `audit_log`, `outbox`, `llm_audit`, `jobs`, khoá Redis; `llm_providers.api_key_enc` không chứa khoá rõ |
| 17 | PASS | thân phản hồi `GET providers` không có `api_key` / `sk-`; chỉ `has_key`, `key_status`; 14 phản hồi canary không có đuôi khoá |
| 18 | PASS* | log gateway `0` dòng `authorization|api_key|Bearer`; log truy cập không có thân; Caddy kiểm tĩnh (không `log`) — không chạy Caddy |
| 19 | PASS | `TestCanaryNeverEchoed` |
| 20 | PASS | thiếu key → `422 IDEMPOTENCY_KEY_REQUIRED`; đúng → `201` (13 trường SRS 6.3, không khoá); lặp cùng key + thân → cùng thân + `Idempotent-Replayed: true`, **1** bản ghi; thân khác → `422 IDEMPOTENCY_KEY_REUSED` |
| 21 | PASS | 10 `POST` đồng thời cùng key: 10 × `201`, 9 × `Idempotent-Replayed`, **1** dòng |
| 22 | PASS | `PUT` đúng version: `200`, `v1→v2`; sai: `409 VERSION_CONFLICT` `details.current_version`, `ETag: W/"v2"`; đua 2 `PUT`: `200` + `409`; thiếu version `422`; `If-Match: W/"v3"` nhận được |
| 23 | PASS | trùng tên y hệt `409 CONFLICT`; xoá nhà đang dùng `409 PROVIDER_IN_USE` `details.tasks=["CHAT"]`; xoá nhà không dùng `204`, mô hình mồ côi 0; nhà thứ 21 `422` "Chỉ cấu hình tối đa 20 nhà cung cấp."; 101 mô hình `422` "tối đa 100…", 100 mô hình `201` |
| 24 | PASS | `TestProviderCRUD TestProviderVersionConflict TestProviderInUse TestProviderIdempotent TestProviderLimit` |
| 25 | PASS | `GET routes`: 7 tác vụ đúng thứ tự + `lane`, `chain[]` theo `fallback_order`, `params`, `version`, `embedding{model_id,provider_name,model,dims:1536,reindex_required,indexed_chunks:null}`, `ETag` |
| 26 | PASS | `chain_empty`, `chain_too_long` (5), `kind_mismatch` ×2, `embedding_single`, `provider_disabled`, `duplicate_model`, `params_out_of_range` ×3 → `422 ROUTE_INVALID` + `details.{rule,field}`; dims 768 → `422 MODEL_DIMS_MISMATCH {actual:768,expected:1536}`; uuid hỏng / tác vụ lạ / mô hình không tồn tại → `422 VALIDATION_FAILED`; sai version `409`; `llm_task_routes` không đổi |
| 27 | PASS | `PUT CHAT` rồi thăm dò cả 2 gateway mỗi 2 ms: 20 lần đổi × 2 gateway: nhỏ nhất **11 ms**, trung vị **13 ms**, lớn nhất **16 ms** (≤ 1 s); `ETag: W/"v<n+1>"`; không khởi động lại |
| 28 | PASS | đổi mô hình `EMBEDDING` → `reindex_required:true`; đổi `CHAT` → `false` |
| 29 | PASS | `TestRoutesGetShape TestRoutesPutRules TestRoutesHotReload TestEmbeddingChangeFlagsReindex` |
| 30 | PASS | gieo 1.000 dòng: khớp `psql` với `percentile_cont` làm tròn: mỗi tác vụ `calls=250`, `tokens_in/out`, `cost_est` (vd. CHAT `"62737.4500"`), p50 **508**, p95 **956**, `errors=10`, `degraded=10`; `group=day` khớp theo ngày **ICT** (5 ngày, `cost_est` đến 4 số lẻ); `from/to` tổng 600 = 600 |
| 31 | PASS | mặc định 7 ngày; 92 ngày `200`, 93 ngày `422`; `from>to` `422`; `group` ∈ {provider, model, course, foo} `422`; `from`/`course_id` hỏng `422`; TEACHER `?course_id=` `403`, TEACHER tổng `200`, ADMIN `?course_id=` `200` |
| 32 | PASS | 21.000 dòng: `group=task` 7 ngày p50 **2 ms**, p95 **3 ms**; `group=day` 92 ngày tối đa **24 ms** (≤ 300); `EXPLAIN`: `Bitmap Index Scan on llm_audit_created_idx` |
| 33 | PASS | `TestUsageAggregates TestUsageWindowLimit TestUsageTeacherNoCourseFilter` |
| 34 | **FAIL** | `GET budget` đúng hình (`scope,daily_limit,monthly_limit,spent_today,spent_month,pct_today,pct_month,state,version`, tiền chuỗi); `100000/2000000` ok; `daily>monthly`, âm → `422 OUT_OF_RANGE`; `null` = không giới hạn; sai version `409`; giới hạn 0,0001 → `GRADING` `503 budget_exhausted`, nâng lên → `200` ở lời gọi kế; **nhưng số vượt biên → `500`: BUG-P104-1** |
| 35 | PASS | `GET/PUT courses/{id}/llm-budget`: ADMIN `200`; TEACHER/TA/STUDENT `403`; `course_id` = `abc` / `123` / hex hỏng → `422` |
| 36 | PASS | `TestBudgetGetShape TestBudgetPutRules TestBudgetTakesEffect TestCourseBudgetAdminOnly`; `grep float32|float64` ở `llmconfig` (không `_test`) = 0 |
| 37 | PASS | sửa ở A, gọi ở B: 11–16 ms; chặn `PUBLISH` (`ACL SETUSER default -publish`): `PUT` vẫn `200` + log WARN "không báo được nạp lại…"; gateway **không** nhận `PUBLISH` tự nạp sau **8,5 s** (≤ 60 s); gateway nhận trực tiếp: 17 ms |
| 38 | PASS | `-tags integration TestReloadAcrossProcesses` |
| 39 | PASS | `go test -count=1 ./internal/contract/...` rc=0 (default và `-tags testroutes`) |
| 40 | PASS | golden PG đổi: **0** (`git diff --stat ef56cf0 2234558` ngoài `/llm/`); `operationId` `openapi.yaml` = **18**, `openapi.test.yaml` = **18**; `contract_test.go` chỉ đổi hằng số 5→18 và danh sách thao tác; `redocly lint` 0 lỗi |
| 41 | PASS | thêm `QCExtra string` vào `budgetView` (worktree QC): `TestContract_ResponsesMatchSpec` **đỏ** (4 thao tác báo `additional properties 'qc_extra' not allowed`); hoàn tác → xanh |
| 42 | PASS | PG: `healthz`, `readyz`, `jobs/{id}`, `events` còn y nguyên (golden không đổi); `golangci-lint` (+tag) 0 vấn đề, `go vet` ok |
| 43 | PASS | mọi nhánh lỗi `application/json` + `{code,message,trace_id}`: JSON hỏng `400 BAD_REQUEST`, sai `Content-Type` `415`, trường lạ `422`, `2 MB` `413 PAYLOAD_TOO_LARGE` (cả khi thiếu `Idempotency-Key`), `404`, `405`; CORS: `Origin: https://evil.example` → không `Access-Control-Allow-Origin` (cả preflight `204`); `http://localhost:3000` được cho; `GET providers` có `ETag`, `If-None-Match` → `304` |
| 44 | PASS | 20 nhà × ~9 mô hình: `GET providers` = `ListLLMProviders` + `ListLLMModels` (**2 truy vấn** dữ liệu + ping) bằng `log_statement=all` |
| 45 | PASS | `TestErrorFormat TestBodyLimit TestNoNPlusOne` |
| 46 | PASS | `scenario-P1.md` (QC tự viết và tự chạy): 13 bước, `fallback_index=1` đo khi nhà chính hỏng |
| 47 | PASS | `git grep` khoá bí mật: chỉ `provider_test.go` (chuỗi kiểm `Redact`); `.env.local` không bị theo dõi; mã mới có `message` tiếng Việt, status đúng SRS 6.1 (`ROUTE_INVALID` 422, `PROVIDER_IN_USE` 409, `MODEL_DIMS_MISMATCH` 422, `IDEMPOTENCY_KEY_*` 422, `VERSION_CONFLICT` 409); `OVERLOADED`, `LLM_UNAVAILABLE`, `LLM_NOT_CONFIGURED` đã đo ở US-P1-03 |
| 48 | PASS | `go vet` (+`testroutes`), `golangci-lint` (+tag), `sqlc diff` rc=0; `go test -race -tags testroutes ./...` 0 FAIL; `-tags "integration testroutes" ./internal/llm/... ./internal/llmconfig/...` ok |

SLO đọc (2 gateway, 20 nhà × 10 mô hình): `providers` p95 **8 ms**, `routes` 1 ms, `budget` 2 ms, `usage` 2 ms (≤ 300).

## Việc sau
Dev sửa BUG-P104-1 → QC chạy lại TC-P104-34/43 và 3 ca tràn số.
