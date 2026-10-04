# QC test case — US-P1-02 (`internal/llm`: Client 4 hàm, registry, lỗi, thử lại, fallback, `llm_audit`, `fake`, replay)
Nguồn: `docs/specs/FEAT-llm-gateway/US.md` US-P1-02 AC1–AC17 + `SRS.md` 4.2 (bảy loại lỗi, tuyến mặc định), 5.2 (`llm_audit`), 6.x (route thử `/api/v1/_test/llm/chat`), 8.3 (`fake`, `FAKE_LLM_VALID_KEY`). Hộp đen. **Giới hạn đã biết:** không có khoá nhà cung cấp thật → mọi TC "gọi nhà cung cấp thật" là **BLOCKED** (ghi, không FAIL; xem US AC13). QC dùng `fake` và máy chủ OpenAI giả **của QC** (`scripts/p102-fake-openai.mjs`, Bun.serve, trả mã / thân / header chọn được, ghi lại request).

Tiền điều kiện chung: stack test (binary `testroutes`, 2 gateway) chạy; `GW=https://localhost`; hàm `tok`, `api`, `$A/$T/$TA_/$S`, `PSQL`, `RDS`, `CANARY` như `FEAT-llm-gateway/US.md` "Quy ước kiểm chung"; `export TESTCONTAINERS_RYUK_DISABLED=true DOCKER_HOST=unix://$HOME/.colima/default/docker.sock`. Công cụ: **G** = `go test` của dev, **S** = shell/curl, **Q** = máy chủ OpenAI giả của QC (cấu hình nhà cung cấp `openai_compatible` trỏ `http://host.docker.internal:<cổng>`), **D** = QC tự đo qua DB, **T** = tay. Mỗi TC loại **G** có ≥ 1 TC **S/Q/D** cùng AC (QC không chỉ tin test dev).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P102-01 | AC1 | repo | **S** `grep -rn "openai\.\|anthropic\.\|genai\." backend-go --include=*.go \| grep -v "internal/llm/"; test $? -eq 1` | exit `0` (không dòng nào) |
| TC-P102-02 | AC1 | – | **S** `grep -rnE 'api\.openai\.com\|api\.anthropic\.com\|generativelanguage\.googleapis\.com' backend-go --include=*.go \| grep -v 'internal/llm/' \| wc -l`; `grep -E 'anthropic\|generative-ai\|genai' backend-go/go.mod \| wc -l`; `grep -c 'github.com/openai/openai-go' backend-go/go.mod` | `0`; `0`; `1` (chỉ thêm `openai-go`; `git diff origin/main -- backend-go/go.mod` chỉ thêm gói này + phụ thuộc gián tiếp của nó) |
| TC-P102-03 | AC1 | – | **S** (tuỳ chọn, không chặn) gieo tệp tạm `internal/zz/x.go` import `github.com/openai/openai-go/v3`; `golangci-lint run ./internal/zz/...`; xoá | AC1 chỉ yêu cầu quét bằng grep (TC-01/02); lint chỉ là lớp phòng thủ **khuyến nghị**: ghi kết quả thực tế, không FAIL nếu chưa có depguard |
| TC-P102-04 | AC2 | – | **G** `go test ./internal/llm/... -run TestClientSurface -v` | `ok` |
| TC-P102-05 | AC2 | – | **S** `grep -nE 'UserID\|CourseID\|StudentCode' backend-go/internal/llm/types.go \| wc -l`; `go doc ./internal/llm Client` | `0`; chỉ 4 hàm `Chat`, `Stream`, `Structured`, `Embed` với đúng chữ ký của US; `Request` có `Task`, `Lane`, `Messages`, `Params`, `Shareable`, `PIIMaskedCount`, `Passages` |
| TC-P102-06 | AC3 | `fake` | **G** `go test -race ./internal/llm/... -run 'TestStreamOneGeneration\|TestStreamCancel' -v` | `ok` |
| TC-P102-07 | AC3 | route thử + `fake` có đếm | **D** `POST /api/v1/_test/llm/chat` (stream) một câu hỏi rồi `GET` bộ đếm `fake.Calls` (hoặc đếm `llm_audit` theo `trace_id`) | Đúng **1** lần sinh văn bản / câu hỏi; **1** dòng `llm_audit`; token phát lần lượt; kênh đóng khi xong |
| TC-P102-08 | AC3 (huỷ) | `fake` độ trễ 5–15 s | **Q** mở stream bằng `curl -N`, ngắt sau 1 s; quan sát `fake` / máy chủ Q | Kết nối tới nhà cung cấp **dừng** (máy chủ Q thấy `close`) ≤ 200 ms sau khi client ngắt; dòng `llm_audit` `status='cancelled'` (hoặc tương đương SRS 5.2) |
| TC-P102-09 | AC4 | – | **G** `go test ./internal/llm/... -run TestRegistry -v` | `ok` |
| TC-P102-10 | AC4 | DB | **D** tạo qua dịch vụ 5 loại nhà cung cấp (5 `type`), 1 nhà **tắt**, 1 nhà **unreadable** (đổi `id`); `Registry.Load`; QC kiểm URL client dùng (log debug / request ở máy chủ Q) | `openai` → `https://api.openai.com/v1`; `anthropic` → `https://api.anthropic.com/v1/`; `gemini` → `https://generativelanguage.googleapis.com/v1beta/openai/`; `openai_compatible` → `base_url` bản ghi; `fake` → giả; tắt / unreadable bị bỏ qua, registry **không sập**; chuỗi tuyến theo `fallback_order` |
| TC-P102-11 | AC5 | **Q** | Cấu hình máy chủ Q trả lần lượt: 401, 403, 404, 404 `model_not_found`, 429, 500, 502, 503, 504, 400, 422, nội dung bị lọc, treo quá hạn, đóng kết nối, từ chối kết nối (cổng đóng), DNS sai; gọi `Chat` | Loại lỗi (`error_kind` ở `llm_audit`): 401/403 `AUTH`; 404 / `model_not_found` `MODEL_NOT_FOUND`; 429 `RATE_LIMIT`; 5xx `SERVER`; quá hạn `TIMEOUT`; đứt / DNS / từ chối `NETWORK`; 400/422/lọc `BAD_REQUEST`; `AUTH`, `BAD_REQUEST` **không** thử lại (máy chủ Q thấy đúng 1 request) |
| TC-P102-12 | AC5 | **Q** | Thân lỗi dài 5.000 ký tự chứa `<html>` / stack trace | Chuỗi lỗi trả ra **không** là thân nguyên văn (ngắn, không HTML, không stack) |
| TC-P102-13 | AC5 | – | **G** `go test ./internal/llm/... -run TestErrorMapping -v` | `ok`; ≥ 16 ca |
| TC-P102-14 | AC6 | **Q** | INTERACTIVE: 503, 503; BATCH: 503, 503, 503, 200; ghi thời điểm request ở máy chủ Q | INTERACTIVE: tối đa **1** lần thử lại (2 request); làn khác: tối đa **2** lần (3 request); trễ lần 1 ∈ [0, 500] ms, lần 2 ∈ [0, 1000] ms (jitter đầy đủ — chạy 20 lần: trễ **không** cố định), trần 4 s |
| TC-P102-15 | AC6 | **Q** | 429 `Retry-After: 2`; 429 `Retry-After: 30`; `ctx` còn < 1 s | `Retry-After ≤ 5 s` được tôn trọng (≈ 2 s); 30 → không đợi 30 s (không thử lại / trả lỗi); còn < 1 s → **không** thử lại; không lần thử nào vượt deadline gốc |
| TC-P102-16 | AC6 | – | **G** `go test ./internal/llm/... -run 'TestRetryBackoff\|TestRetryAfterHeader\|TestRetryRespectsDeadline' -v` | `ok` |
| TC-P102-17 | AC7 | tuyến CHAT `[A,B,C]` (3 nhà **Q** khác cổng) | **Q** A trả 503 liên tục; B 200 | Kết quả từ B, `fallback_index: 1` (JSON route thử); `llm_audit.fallback_index=1`; log `warn` có `trace_id`, `task`, `from`, `to`, `error_kind`, **không** khoá / nội dung |
| TC-P102-18 | AC7 | – | **Q** A 400 (`BAD_REQUEST`); A,B,C đều 503 | 400: **không** chuyển tiếp (máy chủ B 0 request); cả ba hỏng → `ErrAllProvidersFailed` (→ US-P1-03 xử lý suy giảm); `llm_audit.status` lỗi, đúng **1** dòng |
| TC-P102-19 | AC7 (tay) | `/settings/llm` hoặc API | **T** (a) tắt nhà chính (công tắc) rồi gọi `POST /api/v1/_test/llm/chat`; (b) nhà chính **lỗi nhưng còn bật** (ví dụ `openai_compatible` trỏ cổng đóng) rồi gọi | (a) trả lời bởi nhà kế, `fallback_index: 0` (chuỗi đang bật); (b) `fallback_index: 1`, `llm_audit.fallback_index=1`, log `warn` `from`/`to` (góp ý #33) |
| TC-P102-20 | AC7 | – | **G** `-run 'TestFallbackOrder\|TestNoFallbackOnBadRequest\|TestAllFailed'` | `ok` |
| TC-P102-21 | AC8 | `fake` | **D** gọi 1 lần thành công, 1 lần lỗi, 1 lần huỷ; `select * from llm_audit where trace_id in (…)` | **Đúng 1** dòng mỗi lời gọi logic; đủ cột: `task`, `lane`, `provider`, `model`, `tokens_in`, `tokens_out`, `latency_ms`, `queue_wait_ms`, `attempts`, `fallback_index`, `cost_est`, `status`, `error_kind`, `degraded`, `pii_masked_count`, `trace_id`, `user_id`, `course_id`; kể cả lỗi và huỷ |
| TC-P102-22 | AC8 | – | **S** `$PSQL -c "select column_name from information_schema.columns where table_name='llm_audit' and column_name ~* 'prompt\|content\|message\|text\|response'"`; `grep -c 'xin chao'` trên `llm_audit::text` sau khi gửi prompt chứa dấu | Rỗng; không nội dung prompt / câu trả lời ở bất kỳ cột nào |
| TC-P102-23 | AC8 (không đồng bộ) | – | **D** chặn ghi DB (tạm `lock table llm_audit` / dừng Postgres 3 s) khi gọi `fake` | Lời gọi LLM **vẫn thành công**; dòng ghi sau khi DB trở lại (lô); không lỗi 500 |
| TC-P102-24 | AC8 (đệm đầy) | – | **D** chặn DB, gửi > 1.000 lời gọi | Bỏ dòng **cũ nhất**, bộ đếm `llm_audit_dropped` tăng (log `warn`); không OOM |
| TC-P102-25 | AC8 (tắt) | – | **D** gửi 50 lời gọi rồi `SIGTERM` gateway ngay | Mọi 50 dòng có mặt trong DB trước khi tiến trình thoát (đẩy hết khi tắt) |
| TC-P102-26 | AC8 | – | **G** `-run 'TestAuditRow\|TestAuditAsync\|TestAuditDropOldest\|TestAuditFlushOnShutdown'` | `ok` |
| TC-P102-27 | AC9 | route thử | **S** `curl -sk -H "$A" -H 'traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01' -X POST $GW/api/v1/_test/llm/chat -d '{"task":"CHAT","prompt":"xin chao"}'`; `$PSQL -c "select count(*) from llm_audit where trace_id='0af7651916cd43dd8448eb211c80319c'"`; grep log gateway theo `trace_id` | `1`; mọi dòng log của lời gọi mang cùng 32 hex; không `traceparent` → `trace_id` do gateway sinh, vẫn khớp giữa `llm_audit` và log |
| TC-P102-28 | AC10 (góp ý #1, PM duyệt) | **Q** | `Structured` với schema 3 mẫu vào nhà `openai_compatible`; máy chủ Q ghi `response_format` và lời nhắc | **Một** lời gọi nhà cung cấp mỗi `Structured` (D47: không thử rồi lùi); request có `response_format={"type":"json_object"}` **và** schema nằm trong lời nhắc; `type` khác chọn cố định: openai/gemini `json_schema`, anthropic tool bắt buộc (kiểm bằng `TestStructuredRequestShape`, 4 loại); kết quả luôn được kiểm bằng schema |
| TC-P102-29 | AC10 | **Q** | Máy chủ Q trả JSON **sai schema** (thiếu khoá bắt buộc / thừa khoá khi `additionalProperties:false`) cả hai chế độ | Không bao giờ trả JSON sai schema; tính là lỗi nhà cung cấp, chuyển fallback; nếu hết chuỗi → `ErrAllProvidersFailed` |
| TC-P102-30 | AC10 (góp ý #1) | – | **G** `-run 'TestStructuredJSONSchema\|TestStructuredRequestShape\|TestStructuredInvalidFallsBack\|TestValidateJSON'` | `ok`; `TestStructuredDowngrade` không còn (bỏ theo #1) |
| TC-P102-31 | AC11 | **Q** | `Embed`: máy chủ Q trả vectơ 768 chiều; 1.536 chiều; 250 chuỗi; chuỗi rỗng | 768 → `MODEL_DIMS_MISMATCH`, **không** thử lại, không trả vectơ; 1.536 → ok, request có `dimensions=1536`; 250 chuỗi → chia 3 lô (100 / 100 / 50 ở máy chủ Q); rỗng → `BAD_REQUEST` cục bộ, máy chủ Q **0 request** |
| TC-P102-32 | AC11 | `bin/gateway` | **S** `LLM_EMBED_DIMS=768 timeout 10 ./bin/gateway serve; echo rc=$?` | `rc≠0` (từ chối khởi động), thông báo nêu `LLM_EMBED_DIMS` |
| TC-P102-33 | AC11 | – | **G** `-run 'TestEmbedDims\|TestEmbedBatchSplit\|TestEmbedEmptyInput'` | `ok` |
| TC-P102-34 | AC12 | `fake` | **D** `Chat` cùng đầu vào 2 lần; `Embed` 2 chuỗi; chuẩn L2 của vectơ (`‖v‖`); `Structured` hai lần; `Stream` đo khoảng cách token | Xác định (hai lần giống hệt); vectơ 1.536 chiều, `‖v‖ = 1 ± 1e-6`, khác nhau theo chuỗi vào; JSON hợp schema; token cách ≈ 20 ms (mặc định) |
| TC-P102-35 | AC12 | – | **D** `FAKE_LLM_VALID_KEY=good-key`; nhà `fake` khoá `bad-key` rồi `good-key`; độ trễ `5000-15000`; tỉ lệ lỗi 1.0 với từng loại lỗi | Khoá sai → `AUTH`; đúng → thành công; độ trễ nằm trong khoảng (đo 10 lần); tỉ lệ 1.0 → lỗi đúng loại chọn |
| TC-P102-36 | AC12 (phát lại) | – | **S** chế độ replay: yêu cầu có tệp `testdata/replay/*.json` khớp `sha256`; yêu cầu lạ | Có tệp → trả đúng nội dung; lạ → lỗi `REPLAY_MISS` (không âm thầm gọi nơi khác) |
| TC-P102-37 | AC12 (không mạng) | – | **G** `go test ./internal/llm/fake/... -v`; **D** chạy `fake` khi chặn mạng (`HTTP_PROXY=http://127.0.0.1:1`, hoặc chạy trong container `--network none`) | `ok`; `TestFakeNoNetwork` PASS; không kết nối ra ngoài (`lsof`/`tcpdump` 0 gói tới ngoài loopback) |
| TC-P102-38 | AC13 | – | **G** `cd backend-go && go test ./internal/llm -run TestProviderContract -v; echo rc=$?` | `PASS` cho `fake` và `fake-replay`; `rc=0`; với nhà cung cấp thật: `SKIP "BLOCKED: cần khoá thật để ghi"` → **ghi BLOCKED, không FAIL** (đúng US) |
| TC-P102-39 | AC13 (đối chứng âm) | – | **S** sửa tạm một tệp replay (đổi tên khoá `category`→`categoryX`), chạy `TestProviderContract`; hoàn tác | Test **FAIL** và nêu **rõ khoá lệch** (`categoryX`/`category`); hoàn tác → PASS; `git status` sạch |
| TC-P102-40 | AC13 | – | **S** chạy `TestProviderContract` **không biến môi trường khoá** (`env -i`) | Chạy được, không cần khoá |
| TC-P102-41 | AC13 (BLOCKED) | cần khoá thật | `LLM_RECORD=1 OPENAI_API_KEY=… go test ./internal/llm -run TestRecordReplay` | **BLOCKED — thiếu khoá nhà cung cấp thật** (ghi rõ trong report; không FAIL, không giả) |
| TC-P102-42 | AC14 | 2 gateway + Redis | **D** đổi tuyến CHAT sang model khác qua dịch vụ rồi `PUBLISH ep:llm:reload`; gọi liên tục 20 req/s trong lúc đổi | Lời gọi **đang chạy** hoàn tất với cấu hình cũ (không lỗi 5xx); lời gọi mới dùng cấu hình mới trong ≤ **1 s** (đo từ lúc PUBLISH); cả **hai** gateway nhận (`llm_audit.model` đổi ở cả hai) |
| TC-P102-43 | AC14 | – | **D** làm hỏng cấu hình (ví dụ `UPDATE llm_models` vi phạm) rồi reload | Giữ cấu hình **cũ** + log `error`; gọi vẫn thành công; không sập |
| TC-P102-44 | AC14 | – | **D** thăm dò 60 s: sửa DB trực tiếp (không PUBLISH) rồi đợi | Trong ≤ 60 s (+ dung sai) cấu hình mới được nạp |
| TC-P102-45 | AC14 | – | **G** `-race -run 'TestReloadAtomic\|TestReloadKeepsOldOnError'`; `go test -tags integration ./internal/llm/... -run TestReloadTwoProcesses` | `ok` |
| TC-P102-46 | AC15 | DB không dòng bật | **D** xoá / tắt hết nhà cung cấp; thử `LLM_PROVIDER=fake`; `OPENAI_API_KEY` giả; không env nào | `fake` → dùng `fake`; có khoá env → provider tương ứng với tuyến mặc định (SRS 4.2); không gì → `LLM_NOT_CONFIGURED` (503), **không panic**, **không** gọi mạng (máy chủ Q 0 request) |
| TC-P102-47 | AC16 | **Q** phản chiếu header | Máy chủ Q trả lỗi có thân chứa header `Authorization` và `$CANARY`; gọi lỗi | Chuỗi lỗi trả người gọi, log gateway (`docker logs`), `llm_audit.error_kind`, `audit_log` **không** chứa `$CANARY` hay `sk-[A-Za-z0-9_-]{8,}` / `Bearer `; thân lỗi bị cắt ≤ 200 ký tự |
| TC-P102-48 | AC16 | – | **G** `-run TestKeyNeverLeaksViaErrors` | `ok` |
| TC-P102-49 | AC17 | stack | **S** binary **không** `testroutes`: `curl -sk -o /dev/null -w '%{http_code}\n' -H "$A" -X POST $GW/api/v1/_test/llm/chat -d '{}'`; binary `testroutes` với `$S`, `$T`, `$TA_`, không token | `404`; `403` (SV/GV/TA — chỉ `ADMIN`), `401` không token |
| TC-P102-50 | tổng | – | **S** `cd backend-go && go vet ./... && golangci-lint run && go test -race ./...; echo rc=$?`; `go test ./internal/contract/...` | `rc=0`; contract test không đỏ (route thử mới có trong `openapi.test.yaml` nếu spec yêu cầu) |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| SDK nhà cung cấp lọt ra ngoài `internal/llm` | 01–03 |
| Gọi hai lần cho một câu hỏi (D47) | 07 |
| Client ngắt mà nhà cung cấp vẫn chạy | 08 |
| Retry bão, không jitter, vượt deadline | 14, 15 |
| Chuyển fallback cho lỗi của ta | 18 |
| JSON sai schema lọt ra | 29 |
| Vectơ sai chiều lọt vào kho | 31, 32 |
| Mất dòng audit / làm sập lời gọi vì audit | 23–25 |
| Khoá rò qua lỗi | 47 |
| Cấu hình hỏng sập hệ thống | 43 |
| Chưa cấu hình mà panic / gọi mạng | 46 |
| Nhà cung cấp thật không có khoá | 38, 41 (BLOCKED) |

## Câu hỏi cho BA / PM
- **Q-QC-P102-1** — Cách cho QC dựng nhà cung cấp `openai_compatible` trỏ về máy chủ giả trên host: gateway chạy trong container nên cần `host.docker.internal` (colima hỗ trợ). Nếu `base_url` bị chặn bởi quy tắc an toàn (chống SSRF vào mạng nội bộ) — SRS có quy định không? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Spec đã quy định (v1.2, US-P1-04 AC13, SRS 4.2): `base_url` chỉ cần scheme `http|https`, có host, không userinfo, không fragment, ≤ 300 ký tự; **không chặn** địa chỉ nội bộ (`host.docker.internal`, `localhost`, tên dịch vụ compose đều dùng được); không theo chuyển hướng. QC dùng `host.docker.internal` bình thường. Câu hỏi chính sách SSRF → Q15 **[CHỦ DỰ ÁN]** (mặc định: không chặn).
- **Q-QC-P102-2** — Bộ đếm `fake.Calls` / `llm_audit_dropped` phơi ra qua đâu cho hộp đen (route thử / `/metrics`)? Nếu không có, TC-P102-07 / 24 chỉ đo gián tiếp qua `llm_audit` / log. — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Đã thêm (v1.2, SRS 6.4): `GET /api/v1/_test/llm/stats` trả thêm `fake_calls:{<provider>:n}` và `audit:{buffer_len,flushed,dropped}` (chỉ số đếm, không nội dung; chỉ build `testroutes`, ADMIN). TC-P102-07 / 24 đọc qua đó.

## Lịch sử sửa TC
- 2026-10-03 — PM duyệt góp ý sprint 3 #1: TC-P102-28/30 đổi từ "json_schema rồi lùi json_object" sang "`Structured` cố định theo `type`, một lời gọi" (SDK `openai-go/v3`, TC-P102-02 chấp nhận `v3`). BUG-P102-2 đóng.
- 2026-10-03 — TC-P102-03: bỏ yêu cầu "lint đỏ" (QC viết thừa so với AC1); chỉ còn ghi nhận.
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-llm-gateway, APPROVED 2026-10-03).

Tổng: 50 TC (có 1 BLOCKED có chủ đích: TC-P102-41; TC-P102-38 phần provider thật).
