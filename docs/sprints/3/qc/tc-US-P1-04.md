# QC test case — US-P1-04 (API cấu hình LLM `/api/v1/admin/llm/...`, ma trận quyền, khoá chỉ ghi, `openapi.yaml` + contract)
Nguồn: `docs/specs/FEAT-llm-gateway/US.md` US-P1-04 AC1–AC12 + `SRS.md` 6.1 (mã lỗi mới `OVERLOADED`, `LLM_NOT_CONFIGURED`, `LLM_UNAVAILABLE`, `PROVIDER_IN_USE`, `MODEL_DIMS_MISMATCH`, `ROUTE_INVALID`), 6.2 (13 thao tác / 8 đường dẫn), 6.3 (thân phản hồi), 8.3 (`FAKE_LLM_VALID_KEY`). Hộp đen: QC chỉ dùng HTTP (curl) + `psql` + log container; `go test` của dev chạy thêm. **Phần tấn công (khoá rò, phân quyền, idempotency) là trọng tâm.**

Tiền điều kiện chung: stack test (2 gateway, Postgres, Redis, Caddy) chạy với `FAKE_LLM_VALID_KEY=good-key`; `GW=https://localhost`; `tok`, `api`, `idem`, `$A/$T/$TA_/$S`, `PSQL`, `RDS`, `CANARY="sk-LEAK-CANARY-7f3a9c1e"` như `FEAT-llm-gateway/US.md`; QC sinh thêm **token giả mạo**: JWT ký bằng secret khác (`tok` với `JWT_SECRET_KEY=wrong`), JWT `exp` quá khứ (`--ttl -1m`), JWT `alg=none`, JWT `role=ADMIN` sửa tay phần payload (chữ ký sai). Công cụ: **S** `scripts/p104.sh` (hàm `tc_p104_NN`), **G** = `go test`, **T** = tay. Chưa có route → TC FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P104-01 | AC1 | stack | **S** ma trận **13 thao tác × {ADMIN, TEACHER, TA, STUDENT, không JWT, JWT hết hạn}** (vòng bash, 78 ca): `GET providers`, `POST providers`, `PUT providers/{id}`, `DELETE providers/{id}`, `POST providers/test`, `POST providers/{id}/test`, `GET routes`, `PUT routes`, `GET usage`, `GET budget`, `PUT budget`, `GET courses/{id}/llm-budget`, `PUT courses/{id}/llm-budget` | **ADMIN** mọi thao tác thành công (2xx / 4xx nghiệp vụ, không 401/403); **TEACHER** chỉ 4 `GET` (providers, routes, usage, budget hệ thống) = 200; mọi thao tác ghi, `…/test`, `GET courses/{id}/llm-budget` → **403 `FORBIDDEN`** `details.reason="role"`; **TA**, **STUDENT** → 403 trên **cả `GET`**; không JWT → **401 `UNAUTHENTICATED`**; hết hạn → **401 `TOKEN_EXPIRED`**; không ca nào lệch |
| TC-P104-02 | AC1 (chặn trước DB) | – | **S** TEACHER `POST providers` với thân rác + `$PSQL` bật `log_statement` hoặc đếm `pg_stat_statements`/`xact_commit` trước-sau; máy chủ nhà cung cấp Q 0 request | 403 trả **trước** khi chạm DB (không truy vấn bảng `llm_*`) và trước nhà cung cấp; thân lỗi chuẩn `{code,message,trace_id}` |
| TC-P104-03 | AC1 (**tấn công** token giả) | – | **S** gọi `GET providers` và `PUT routes` với: JWT ký secret sai; JWT `alg=none`; JWT sửa `role` thành ADMIN giữ chữ ký cũ; JWT `role` lạ `SUPERUSER`; JWT thiếu `sub`; `Authorization: Bearer` rỗng; `Bearer a.b.c` | Tất cả **401** (`TOKEN_INVALID`/`UNAUTHENTICATED`), **không** 200/403 lẫn lộn; không lộ lý do nội bộ; 0 thay đổi DB |
| TC-P104-04 | AC1 (**tấn công** leo thang) | JWT STUDENT hợp lệ | **S** STUDENT thử: header `X-Role: ADMIN`, `?role=ADMIN`, thân `{"role":"ADMIN"}` ở mọi thao tác ghi; `X-User-Id` giả | Vai chỉ lấy từ JWT; mọi cách đều **403**; DB không đổi |
| TC-P104-05 | AC1 | – | **G** `go test ./internal/llmconfig/... -run TestRBACMatrix -v` | `ok`; bảng ≥ 60 ca |
| TC-P104-06 | AC2 | `FAKE_LLM_VALID_KEY=good-key` | **S** `api "$A" -X POST $GW/api/v1/admin/llm/providers/test -d '{"type":"fake","api_key":"bad-key","model":"fake-chat"}'`; đếm `llm_providers` trước-sau; `docker compose logs gateway \| grep -c bad-key` | `200`, `.ok==false`, `.error_kind=="AUTH"`, `.message == "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."`, có `latency_ms`; `count(*)` **không đổi**; log `0` lần `bad-key` |
| TC-P104-07 | AC2 | – | **S** khoá đúng `good-key` | `200`, `.ok==true`, `latency_ms` số; **không** lưu bản ghi |
| TC-P104-08 | AC2 (7 `error_kind`) | máy chủ Q | **S** gây `NETWORK` (cổng đóng), `TIMEOUT` (treo), `RATE_LIMIT` (429), `MODEL_NOT_FOUND` (404), `BAD_RESPONSE` (200 thân rác), `DIMS_MISMATCH` (vectơ 768) | Mỗi `error_kind` đúng, mỗi loại có **câu tiếng Việt riêng** (không trùng nhau, không mã trần, không thân nhà cung cấp) |
| TC-P104-09 | AC2 | – | **S** mỗi lần Test: `select count(*) from audit_log` trước-sau; `details::text` có khoá? ; thử 30 lần/phút (giới hạn tần suất ghi ở US) | +1 dòng `audit_log` mỗi lần Test, **không** có khoá; vượt giới hạn tần suất → `429 RATE_LIMITED` có `retry_after` (đối chiếu giá trị US ghi ở phần bị cắt: QC đọc `SRS 6.2`) |
| TC-P104-10 | AC2 | – | **G** `-run 'TestProviderTestEndpoint\|TestTestRateLimit'` | `ok` |
| TC-P104-11 | AC3 | – | **S** `api "$A" -H "Idempotency-Key: $(idem)" -X POST $GW/api/v1/admin/llm/providers -d '{"type":"fake","name":"F","api_key":"bad-key","models":[{"model":"fake-chat","kind":"chat","price_in":"0","price_out":"0"}]}'`; rồi `GET providers` | `422 VALIDATION_FAILED`, `details=[{"field":"api_key","code":"PROVIDER_AUTH_FAILED","message":…}]`; `GET` **không** có "F"; `llm_providers` không dòng mới |
| TC-P104-12 | AC3 | – | **S** `PUT` đổi `api_key` sai cho nhà đã có (đã có `good-key`); rồi dùng nhà đó gọi chat | 422; khoá cũ **giữ nguyên** (`md5(api_key_enc)` không đổi; gọi chat vẫn thành công) |
| TC-P104-13 | AC3 | – | **S** `POST` với `skip_verify:true` bằng ADMIN, rồi bằng TEACHER; `PUT` chỉ đổi `name`/`enabled`/giá (không `api_key`/`base_url`) | ADMIN: lưu được, `last_test.ok=null`; TEACHER: 403; `PUT` không đổi khoá/URL **không** gọi kiểm kết nối (máy chủ Q 0 request) |
| TC-P104-14 | AC3 | – | **S** đổi `base_url` sang máy chủ không với tới / sang địa chỉ **nội bộ** (`http://169.254.169.254/`, `http://localhost:5432`, `http://redis:6379`) | `422` `PROVIDER_UNREACHABLE`; **kiểm SSRF**: không lộ nội dung phản hồi nội bộ trong lỗi; ghi lại hành vi thực (nếu gateway có gọi tới metadata cloud → báo lỗi bảo mật) |
| TC-P104-15 | AC3 | – | **G** `-run 'TestSaveVerifiesFirst\|TestSaveSkipVerify\|TestUpdateWithoutKeyNoVerify'` | `ok` |
| TC-P104-16 | AC4 (**canary**) | – | **S** `scripts/canary-scan.sh`: tạo nhà cung cấp với `$CANARY` (skip_verify nếu cần), đổi khoá, gửi sai định dạng (thiếu `type`, `api_key` số, mảng), kích hoạt 500 giả lập (thân quá `MAX_BODY_BYTES`, JSON hỏng), thực hiện đủ 13 thao tác; sau đó `docker compose logs \| grep -c "$CANARY"`; quét `audit_log`, `outbox`, `llm_audit`, `jobs`, `pg_dump` | **`0`** ở mọi nơi; chỉ có đuôi? `grep -c "7f3a9c1e"` cũng `0` (không cả đuôi khoá) |
| TC-P104-17 | AC4 (phản hồi) | – | **S** quét **mọi** thân phản hồi (GET/POST/PUT/lỗi/test) bằng `grep -ci 'api_key\|sk-'`; nhớ chính trường tên `api_key` ở **phản hồi** | Phản hồi **không** có khoá hay đuôi; chỉ `has_key`, `key_status`; trường `api_key` không xuất hiện trong thân phản hồi |
| TC-P104-18 | AC4 (header/log truy cập) | – | **S** `docker compose logs caddy gateway \| grep -ci "authorization\|api_key"`; log truy cập có thân yêu cầu? | Log truy cập không ghi thân yêu cầu / header `Authorization`; Caddy không log thân |
| TC-P104-19 | AC4 | – | **G** `-run TestCanaryNeverEchoed -v` | `ok` |
| TC-P104-20 | AC5 | – | **S** `POST` thiếu `Idempotency-Key`; `POST` đúng; lặp **cùng key + cùng thân**; lặp **cùng key + thân khác** | Thiếu → `422 IDEMPOTENCY_KEY_REQUIRED`; đúng → `201` thân theo SRS 6.3 (`has_key`, `key_status`, `last_test`, `circuit`, `models[]`, `version`; **không** có khoá); lặp → cùng phản hồi + `Idempotent-Replayed: true`, **1** bản ghi (`count(*)`); thân khác → `422/409 IDEMPOTENCY_KEY_REUSED` (đúng mã SRS 6.1 PG) |
| TC-P104-21 | AC5 (idempotency đua) | – | **S** 2 tab / `xargs -P2` `POST` cùng key đồng thời | 1 bản ghi; một phản hồi `Idempotent-Replayed: true` |
| TC-P104-22 | AC5 | – | **S** `PUT` đúng `version`; `PUT` sai `version`; `PUT` hai người cùng `version` (đua) | Đúng → `200`, `version` tăng 1; sai → **409 `VERSION_CONFLICT`** `details.current_version`; đua: 1 thắng, 1 `409` |
| TC-P104-23 | AC5 | – | **S** tạo trùng `name`; `DELETE` nhà đang được tuyến dùng; `DELETE` nhà không dùng; tạo nhà thứ 21 | `409 CONFLICT`; `409 PROVIDER_IN_USE` `details.tasks=["CHAT",…]`; `204` và mô hình của nó biến mất; thứ 21 → `422 VALIDATION_FAILED` ("tối đa 20") |
| TC-P104-24 | AC5 | – | **G** `-run 'TestProviderCRUD\|TestProviderVersionConflict\|TestProviderInUse\|TestProviderIdempotent\|TestProviderLimit'` | `ok` |
| TC-P104-25 | AC6 | 2 mô hình `fake-chat`, `fake-chat-2`, 1 mô hình embedding 1.536 | **S** `GET routes`; kiểm 7 tác vụ, `chain[]`, `params`, `version`, khối `embedding` (`model_id`, `dims:1536`, `reindex_required`, `indexed_chunks: null`) | Đúng hình dạng SRS 6.3; `chain` theo `fallback_order` |
| TC-P104-26 | AC6 | – | **S** `PUT routes` vi phạm từng quy tắc P1-01 AC10: chuỗi rỗng, 5 mô hình, mô hình `embedding` cho `CHAT`, `EMBEDDING` 2 mô hình, `dims=768`, nhà cung cấp tắt, trùng, `temperature=3` | `422 ROUTE_INVALID` + `details.rule` (hoặc `422 MODEL_DIMS_MISMATCH` cho dims); **không** đổi DB (`version` cũ còn) |
| TC-P104-27 | AC6 (nóng) | 2 gateway | **S** `PUT routes {"task":"CHAT","chain":["<id fake-chat-2>"],"version":N}` rồi gọi ngay `POST /api/v1/_test/llm/chat` ở **cả hai** gateway (cổng riêng); đo thời gian | `model:"fake-chat-2"` trong **≤ 1 s** ở cả hai; `ETag: W/"v<n+1>"`; không restart; lời gọi đang chạy không bị đứt |
| TC-P104-28 | AC6 | – | **S** đổi mô hình `EMBEDDING` | Phản hồi `reindex_required:true` |
| TC-P104-29 | AC6 | – | **G** `-run 'TestRoutesGetShape\|TestRoutesPutRules\|TestRoutesHotReload\|TestEmbeddingChangeFlagsReindex'` | `ok` |
| TC-P104-30 | AC7 | gieo `llm_audit` (1.000 dòng với số liệu biết trước, tính tay bằng `psql`) | **S** `GET usage?group=task`, `group=day`, `from/to`; so từng số với `select … group by` của QC | `calls`, `tokens_in`, `tokens_out`, `cost_est` (chuỗi thập phân VND, **chính xác từng đồng**), `latency_p50_ms`, `latency_p95_ms`, `errors`, `degraded` khớp số tay |
| TC-P104-31 | AC7 | – | **S** khoảng mặc định (7 ngày); khoảng 93 ngày; `from > to`; `group=lạ`; TEACHER `?course_id=<uuid>` | 7 ngày mặc định đúng; 93 ngày → `422 VALIDATION_FAILED`; `from>to` → 422; `group` lạ → 422; TEACHER với `course_id` → **403**; TEACHER tổng hệ thống → 200 |
| TC-P104-32 | AC7 | – | **S** `EXPLAIN` truy vấn tương ứng với 20.000 dòng (như TC-P101-11) / thời gian phản hồi 20.000 dòng | Dùng chỉ mục; phản hồi ≤ 300 ms (ghi số) |
| TC-P104-33 | AC7 | – | **G** `-run 'TestUsageAggregates\|TestUsageWindowLimit\|TestUsageTeacherNoCourseFilter'` | `ok` |
| TC-P104-34 | AC8 | – | **S** `GET /admin/llm/budget`; `PUT` các giá trị: `daily=100000, monthly=2000000`; `daily>monthly`; âm; `null`; `"abc"`; sai `version` | Thân `{daily_limit, monthly_limit, spent_today, spent_month, pct_today, pct_month, state, version}` (tiền là **chuỗi** thập phân); `daily>monthly` → 422; âm → 422; `null` = không giới hạn; sai version → 409; hiệu lực ngay cho lời gọi kế tiếp |
| TC-P104-35 | AC8 (phân quyền) | – | **S** `GET/PUT courses/{id}/llm-budget` bằng ADMIN / TEACHER / TA / STUDENT; `course_id` không phải UUID | Chỉ ADMIN; còn lại 403; `course_id` rác → `422` (tồn tại của lớp chưa kiểm — P2) |
| TC-P104-36 | AC8 | – | **G** `-run 'TestBudgetGetShape\|TestBudgetPutRules\|TestBudgetTakesEffect\|TestCourseBudgetAdminOnly'`; `grep -rnE 'float(32\|64)' backend-go/internal/llmconfig` (không `_test`) | `ok`; `0` |
| TC-P104-37 | AC9 | 2 gateway | **S** sửa ở gateway A (`PUT providers`/`routes`), gọi `_test/llm/chat` ở gateway B ngay; rồi chặn `PUBLISH` (dừng Redis tạm hoặc `redis-cli CLIENT PAUSE`) và sửa tiếp | B dùng cấu hình mới ≤ 1 s; khi PUBLISH hỏng: thăm dò **60 s** vẫn nạp kịp |
| TC-P104-38 | AC9 | – | **G** `-tags integration -run TestReloadAcrossProcesses -v` | `ok` |
| TC-P104-39 | AC10 | – | **S** `cd backend-go && go test -count=1 ./internal/contract/...; echo rc=$?` | `rc=0` |
| TC-P104-40 | AC10 | – | **S** `git diff --stat origin/main -- backend-go/internal/contract/testdata/golden/ \| grep -v '/llm/' \| grep -c golden`; `grep -c 'operationId' backend-go/openapi.yaml` so với bản sprint 2 + 13; `redocly lint` | `0` golden PG bị sửa; `operationId` ≥ cũ + 13; 8 đường dẫn `/api/v1/admin/llm/...` / `/courses/{id}/llm-budget`; `redocly lint` rc=0 |
| TC-P104-41 | AC10 (đối chứng âm) | – | **S** thêm khoá lạ vào một phản hồi (tệp tạm ở handler, không commit) → `go test ./internal/contract/...`; hoàn tác | Contract test **đỏ**; hoàn tác → xanh; `git status` sạch |
| TC-P104-42 | AC10 | – | **S** `make -C backend-go lint` + `go vet`; PG routes (`/api/v1/events`, `healthz`, `readyz`, `jobs/{id}`) còn y nguyên (so golden) | Không đường dẫn PG nào đổi (nguyên tắc 7) |
| TC-P104-43 | AC11 | – | **S** thân lỗi mọi nhánh: `Content-Type: application/json`, `{code,message,trace_id}`; thân > `MAX_BODY_BYTES`; JSON hỏng; `Content-Type` sai; CORS với `Origin` không có trong `CORS_ORIGINS` | Định dạng lỗi chuẩn PG; `413`/`400` đúng mã SRS 6.1; CORS không cho origin lạ (không `Access-Control-Allow-Origin` cho evil.example); `GET` một tài nguyên có `ETag` |
| TC-P104-44 | AC11 (N+1) | 20 nhà × 10 mô hình | **S** đếm truy vấn của `GET providers` (`pg_stat_statements` hoặc log) | ≤ 3 truy vấn (không N+1) |
| TC-P104-45 | AC11 | – | **G** `-run 'TestErrorFormat\|TestBodyLimit\|TestNoNPlusOne'` | `ok` |
| TC-P104-46 | AC12 | stack sạch | **T** làm theo `docs/sprints/3/qc/scenario-P1.md`: thêm 2 nhà `fake`, đổi `CHAT` sang nhà thứ hai, nhập khoá sai → Test báo lỗi + khoá không hiện lại, tắt nhà chính → fallback `fallback_index:1`; mỗi bước ghi lệnh + kết quả | Mỗi bước đúng AC2–AC6; QC **lặp lại** độc lập (không chép từ handoff); bảng bước ghi vào `report-US-P1-04.md` |
| TC-P104-47 | tổng (an toàn) | – | **S** `git grep -nE 'sk-[A-Za-z0-9]{10,}\|AKIA\|BEGIN (RSA\|PRIVATE)'`; `.env.local` không bị commit; mã lỗi mới có `message` tiếng Việt: lặp 6 mã SRS 6.1 mới (`OVERLOADED`…`ROUTE_INVALID`) | Không secret trong repo; mỗi mã mới có thông điệp tiếng Việt, HTTP status đúng bảng SRS 6.1 |
| TC-P104-48 | tổng | – | **S** `cd backend-go && go vet ./... && golangci-lint run && sqlc diff && go test -race ./...; echo rc=$?` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Ma trận quyền lệch (TA đọc được, TEACHER ghi được) | 01 |
| Token giả / `alg=none` / sửa vai / leo thang bằng header | 03, 04 |
| 403 sau khi đã chạm DB / nhà cung cấp | 02 |
| Lưu khoá sai làm mất khoá cũ | 11, 12 |
| SSRF qua `base_url` | 14 |
| Khoá lộ qua phản hồi / log / DB / audit | 16–18 |
| Idempotency: lặp, đua, thân khác | 20, 21 |
| Sửa đồng thời (version) | 22 |
| Xoá nhà đang dùng | 23 |
| Hot-reload không tới tiến trình kia | 27, 37 |
| Số `usage` sai một đồng | 30 |
| Golden PG bị sửa để xanh | 40 |

## Câu hỏi cho BA / PM
- **Q-QC-P104-1** — TC-P104-14 (SSRF): US không nói `base_url` có bị chặn địa chỉ nội bộ (`localhost`, `169.254.*`, tên dịch vụ compose) không, nhưng Admin có thể trỏ `openai_compatible` vào máy chủ trong trường (cần cho trường hợp thật). QC ghi hành vi thực và báo như **rủi ro** (không tự FAIL) trừ khi SRS yêu cầu chặn. SRS có quy định? — *chờ trả lời*.
- **Q-QC-P104-2** — AC2 giới hạn tần suất của `…/test`: giá trị nằm ở phần bị cắt trong US/SRS; QC lấy từ SRS 6.2 khi chạy. Nếu SRS không nêu con số, TC-P104-09 ghi N/A. — *chờ xác nhận*.
- **Q-QC-P104-3** — `scenario-P1.md` do dev viết hay QC? AC12 nói "Dev tự kiểm, QC lặp lại": QC dùng bản của dev nếu có, nếu không QC tự viết từ kịch bản AC12. — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-llm-gateway, APPROVED 2026-10-03).

Tổng: 48 TC.
