# Báo cáo QC — US-P1-02 (cổng `internal/llm`)
**Kết luận: PASS (sau vòng sửa 1)** — vòng 1: 3 lỗi; vòng 2 (commit `ef56cf0` + PM duyệt góp ý #1): BUG-P102-1 và -3 đã sửa, BUG-P102-2 đóng theo quyết định PM → 49/50 PASS, TC-41 BLOCKED có chủ đích (cần khoá thật). Xem "Vòng sửa 1".

- Bản chấm: commit `1b71421` trong worktree QC riêng (như `report-US-P1-01.md`); không chạm worktree của dev.
- Môi trường: gateway `-tags testroutes` native :8080/:8081 (2 bản, Redis dùng chung), Postgres + Redis riêng. **Máy chủ OpenAI giả của QC** (Bun, :9701-9703; mã / thân / `Retry-After` / độ trễ chọn được, ghi lại từng request) nối qua nhà cung cấp `openai_compatible`; `fake` của dev dùng cho ca không cần mạng.
- Không có đường HTTP cho `Structured` và `Embed` → QC gọi `llm.Gateway` bằng chương trình Go riêng (`scripts/p102-gwprobe`) trỏ về máy chủ giả.
- Q-QC: P102-1 (base_url nội bộ) và P102-2 (bộ đếm `stats`) — SRS v1.2 đã trả lời; QC chấm `stats` theo SRS 6.4 v1.2 (BUG-P102-3).

## Lỗi
### BUG-P102-1 — HTTP 504 bị xếp `TIMEOUT`, spec bắt `SERVER`
SRS 4.2 + US AC5: `SERVER` = 500, 502, 503, **504**.
Tái hiện: máy chủ giả trả 504 `{}` cho `CLASSIFY`; `select status, error_kind from llm_audit order by created_at desc limit 1`.
Thực tế: `timeout | TIMEOUT | 3 lần thử`. Mong đợi: `error | SERVER`. (500 / 502 / 503 đúng `SERVER`.) Test của dev (`provider_test.go:88`, ca "504") đang **ghim** hành vi sai.

### BUG-P102-2 — `Structured` không thử `json_schema` trước (lệch AC10) — **chờ PM, `proposals.md` #1**
Tái hiện: `scripts/p102-gwprobe` gọi `Structured` bốn lần vào nhà `openai_compatible`; máy chủ giả ghi `response_format`.
Thực tế: cả 4 request tới nhà đó chỉ có `{"type":"json_object"}`; không bao giờ `json_schema`; không có đường "lùi" (`TestStructuredDowngrade` không tồn tại). Dev đã nêu lệch này ở góp ý #1 (cột "Quyết định PM" còn trống). QC chấm theo spec hiện hành → TC-28 / TC-30 FAIL; nếu PM duyệt #1, QC sẽ sửa TC rồi chạy lại. Phần còn lại của AC10 đạt: JSON sai schema (`{"x":1}`) và không phải JSON bị loại, **không bao giờ trả JSON sai schema**, chuyển fallback rồi `ErrAllProvidersFailed`.

### BUG-P102-3 — `GET /_test/llm/stats` sai hình dạng so với SRS 6.4 v1.2
SRS 6.4 (v1.2, Q-QC-P102-2): `{queue_depth, inflight, provider_inflight, circuit, fake_calls:{<provider>:n}, audit:{buffer_len, flushed, dropped}}`.
Thực tế (ADMIN): `audit_dropped, circuit, inflight, provider_calls, provider_inflight, queue_depth`. Thiếu `fake_calls`, `audit.{buffer_len,flushed,dropped}`; `inflight` / `provider_inflight` chỉ đếm `fake`, không đếm nhà `openai_compatible`. QC đã đo gián tiếp được (`llm_audit`, log, máy chủ giả) nên TC-07/24 vẫn PASS về hành vi, nhưng khoá spec thiếu → ghi lỗi riêng. Spec v1.2 commit 13:22 sau bàn giao.

## Kết quả từng TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | grep SDK ngoài `internal/llm/` = 0 dòng |
| 02 | PASS | tên miền nhà cung cấp ngoài `internal/llm/` = 0; `go.mod`: anthropic/genai = 0, thêm `openai-go/v3 v3.71.1` (theo góp ý #1, chờ PM) và `shopspring/decimal` |
| 03 | PASS | phần AC1 (grep). **Ghi chú:** QC đã viết thừa phép "lint đỏ khi import SDK": thực tế `.golangci.yml` chỉ có depguard cho `kin-openapi`; AC1 chỉ đòi quét, không đòi cổng lint — QC sửa TC (xem Lịch sử), đề xuất thêm depguard làm lớp phòng thủ (không bắt buộc) |
| 04 | PASS | `TestClientSurface` |
| 05 | PASS | grep `UserID|CourseID|StudentCode` ở `types.go` = 0 |
| 06 | PASS | `TestStreamOneGeneration`, `TestStreamCancel` `-race` |
| 07 | PASS* | một câu hỏi = 1 dòng `llm_audit` (CHAT/INTERACTIVE/ok), Q ghi đúng 1 request chat; stream phát `event: token` rồi `done`. *`fake_calls` chưa có — BUG-P102-3 |
| 08 | PASS | `fake` 5–15 s, `curl` ngắt sau 1 s: `provider_inflight` về 0 ≤ ~116 ms sau khi curl thoát; audit `cancelled / CANCELLED` |
| 09 | PASS | `TestRegistry` (5 loại × URL, tắt, unreadable); QC thêm `openai_compatible` (URL của bản ghi) và `fake` qua HTTP; nhà tắt / unreadable không làm sập |
| 10 | PASS | như 09 |
| 11 | **FAIL** | BUG-P102-1. Các mã còn lại đúng: 401/403→`AUTH`, 404 & `model_not_found`→`MODEL_NOT_FOUND`, 429→`RATE_LIMIT`, 500/502/503→`SERVER`, 400/422/`content_filter`→`BAD_REQUEST`, đứt kết nối→`NETWORK`; `AUTH`/`BAD_REQUEST`/404 không thử lại (Q thấy đúng 1 request). Không đo `TIMEOUT` thật (hang 30 s) — dev có `TestErrorMapping`; không đo DNS |
| 12 | PASS | thân 5.000 ký tự HTML + "stack trace": phản hồi chỉ là câu chung 174 byte, không HTML / stack |
| 13 | PASS | `TestErrorMapping` |
| 14 | PASS | INTERACTIVE: `503,ok` → 2 request; 503 mãi → Q1 đúng 2 request rồi chuyển Q2. NEAR_REALTIME / BATCH: `503,503,ok` → 3 request, `503×3` → 3 request rồi chuyển. Jitter: 20 lần INTERACTIVE khoảng cách 44–452 ms (19 giá trị khác nhau, ≤ 500); NEAR_REALTIME lần 2 tối đa 882 ms (≤ 1.000), 10 giá trị khác nhau |
| 15 | PASS | `Retry-After: 2` → chờ 2.002 ms rồi 200; `Retry-After: 30` → không đợi, lỗi sau 3 ms, 1 request |
| 16 | PASS | `TestRetryBackoff TestRetryAfterHeader TestRetryRespectsDeadline` |
| 17 | PASS | Q1 503 → Q2: `fallback_index:1`, audit `fallback_index=1, attempts=3`; log `WARN` có `trace_id, task, from, to, error_kind`, không khoá / nội dung. QC thêm `fake` AUTH (`F2` khoá sai → `F1`) → `fallback_index:1` |
| 18 | PASS | 400 → Q2, Q3 0 request, 422; cả ba 503 → `LLM_UNAVAILABLE` 503, 2 request mỗi nhà, **1** dòng audit |
| 19 | PASS | nhánh tay ("tắt nhà chính") tương đương TC-17 (AUTH fallback) |
| 20 | PASS | `TestFallbackOrder TestNoFallbackOnBadRequest TestAllFailed` |
| 21 | PASS | đủ cột (task, lane, provider, model, tokens, latency, queue_wait, attempts, fallback_index, cost_est, status, error_kind, degraded, pii, user_id, trace_id 32 hex); thành công / lỗi / huỷ mỗi loại 1 dòng |
| 22 | PASS | `llm_audit` không cột `prompt|content|message|text|response`; `grep "xin chao"` trong dòng audit = 0 |
| 23 | PASS | `docker pause` Postgres, 50 lời gọi: **50/50 thành công**; sau `unpause` 50 dòng audit đến |
| 24 | PASS* | tạm dừng DB, 1.500 lời gọi: không lỗi; đệm đầy → bỏ dòng cũ nhất, `audit_dropped=400`, log `WARN llm_audit đầy, bỏ dòng cũ nhất`; 1.100 dòng vào DB sau khi bật lại (đệm 1.000 + lô đang bay). *tên khoá stats lệch spec (BUG-P102-3) |
| 25 | PASS | 50 lời gọi rồi `SIGTERM` gateway: tiến trình thoát, đủ 50 dòng |
| 26 | PASS | `TestAuditRow TestAuditAsync TestAuditDropOldest TestAuditFlushOnShutdown` |
| 27 | PASS | `traceparent` 32 hex `0af765…319c`: `llm_audit` 1 dòng, log gateway khớp |
| 28 | **FAIL** | BUG-P102-2 |
| 29 | PASS | `{"x":1}` sai schema và văn bản không phải JSON: `Structured` trả lỗi (`mọi nhà cung cấp đều lỗi`), không trả JSON sai |
| 30 | **FAIL** | BUG-P102-2 (`TestStructuredDowngrade` vắng; `TestStructuredJSONSchema`, `TestStructuredInvalidFallsBack` PASS) |
| 31 | PASS | vectơ 768 → `vectơ 768 chiều, cần 1536`, chỉ 1 request (không thử lại); 1536 OK, mọi request có `dimensions=1536`; 250 chuỗi → 100/100/50; chuỗi rỗng và danh sách rỗng → `yêu cầu không hợp lệ`, 0 request |
| 32 | PASS | `LLM_EMBED_DIMS=768` → rc=1, nêu tên biến, không mở cổng |
| 33 | PASS | `TestEmbedDims TestEmbedBatchSplit TestEmbedEmptyInput` |
| 34 | PASS | `Chat` giống nhau 2 lần (md5 trùng); `Stream` cách ~20 ms/token (16–21 ms; 2 token đầu gộp); vectơ L2 / `Structured` xác định: test `fake` của dev |
| 35 | PASS | `valid_key` đặt rồi nhà không khoá → `AUTH` (1 lần thử); độ trễ 200–400 ms: 10 mẫu 212–375 ms; `error_rate=1` với `SERVER`, `RATE_LIMIT`, `TIMEOUT`, `NETWORK`, `AUTH`, `BAD_REQUEST` ra đúng `error_kind`; `error_rate=1.5` → 422 |
| 36 | PASS | phát lại: `TestProviderContract` đọc `fake-replay`; sai khoá → so hình dạng lỗi (TC-39); `REPLAY_MISS`: test dev |
| 37 | PASS | `TestFakeNoNetwork` |
| 38 | PASS | `fake`, `fake-replay` PASS; `openai/anthropic/gemini` SKIP "BLOCKED: cần khoá thật để ghi" |
| 39 | PASS | đổi `category`→`categoryX` ở `testdata/replay/fake-replay/classify.json` (trong worktree QC): FAIL nêu "khoá thừa $.categoryX; thiếu khoá $.category"; hoàn tác → PASS |
| 40 | PASS | `env -i go test -run TestProviderContract` ok (không cần khoá) |
| 41 | BLOCKED | ghi âm cần khoá nhà cung cấp thật — chờ chủ dự án (Q11) |
| 42 | PASS | đổi `CHAT` bằng SQL + `PUBLISH ep:llm:reload`, tải 20 req/s vào **cả hai** gateway: lời gọi mới dùng mô hình mới sau **88 ms** ở cả hai; 0 lỗi / 72 lời gọi mỗi bên |
| 43 | PASS | đổi tên bảng `llm_models` (nạp lỗi) + publish: chat vẫn chạy cấu hình cũ; log `ERROR nạp lại cấu hình LLM thất bại, giữ cấu hình cũ` |
| 44 | PASS | sửa DB không publish: gateway thứ hai nhận sau 28,6 s (≤ 60 s) |
| 45 | PASS | `TestReloadAtomic TestReloadKeepsOldOnError`; `-tags integration TestReloadTwoProcesses` PASS |
| 46 | PASS | `LLM_PROVIDER=fake` → provider `env:fake`; không cấu hình nào (DB trống, `LLM_PROVIDER=` rỗng) → `LLM_NOT_CONFIGURED` 503 (cả stream), không panic, audit `not_configured`, không kết nối ra ngoài; nhánh khoá OpenAI/Anthropic/Gemini: `TestEnvFallback` (QC không gọi nhà thật) |
| 47 | PASS | máy chủ giả trả lỗi 401/403/429/500/400 có thân phản chiếu canary + `Bearer`: phản hồi, log gateway (hai bản), `llm_audit`, `audit_log` không chứa canary / `sk-…` / `Bearer …`; `pg_dump` = 0 |
| 48 | PASS | `TestKeyNeverLeaksViaErrors` (hai gói) |
| 49 | PASS | binary thường: 3 route 404; `testroutes`: ADMIN 200, TEACHER / TA / STUDENT 403, không token 401 |
| 50 | PASS | `go vet` (+`testroutes`), `golangci-lint` (+tag) 0 issue, `go test -race -tags testroutes ./...` ok (gồm contract) |

## Ghi chú
- Giới hạn tần suất PG (300 req/phút/IP) chặn tải lớn → QC đặt `RATE_LIMIT_*_PER_MIN=100000` cho các ca tải (ghi rõ; không phải lỗi).
- Không có `stats` cho nhà `openai_compatible` (`inflight` chỉ `fake`) — nằm trong BUG-P102-3; US-P1-03 sẽ cần.
- Bước tay `curl` của AC7 / AC9 / AC17 trong handoff nay đã được QC chạy (TC-17, 27, 49).
- Tệp: `scripts/p102-setup`, `scripts/p102-gwprobe`, `scripts/p1-run/{burst.py,reload.py,gw.sh,env.sh,spec-*.json,README.md}`.

## Việc sau
Dev sửa BUG-P102-1 và -3; PM quyết góp ý #1 (BUG-P102-2). QC chạy lại TC-11, 28, 30 và đo lại `stats`.

## Lịch sử sửa TC
- 2026-10-03 — TC-P102-03: bỏ vế "lint đỏ khi import SDK" (không thuộc AC1); giữ phép grep.

## Vòng sửa 1 (đo trên `ef56cf0`)
| Lỗi | Kết quả | Bằng chứng |
| --- | --- | --- |
| BUG-P102-1 (504→`SERVER`) | **ĐÃ SỬA** | máy chủ giả trả 500, 502, 503, **504**: audit `error / SERVER / 3 lần thử` cả bốn; 429 → `rate_limited / RATE_LIMIT` |
| BUG-P102-2 (Structured) | **ĐÓNG** (PM duyệt góp ý #1) | TC-28/30 sửa: nhà `openai_compatible` nhận **đúng 1** lời gọi mỗi `Structured`, `response_format={"type":"json_object"}` và schema trong lời nhắc; JSON sai schema (`{"x":1}`) → `ErrAllProvidersFailed`, không trả JSON sai; `TestStructuredRequestShape` (4 loại), `TestStructuredJSONSchema`, `TestStructuredInvalidFallsBack` PASS |
| BUG-P102-3 (stats) | **ĐÃ SỬA** | `GET _test/llm/stats` (ADMIN) có đủ `audit{buffer_len,flushed,dropped}`, `circuit`, `fake_calls`, `inflight` (khoá theo `provider_id`, gồm cả nhà `openai_compatible`), `provider_inflight`, `queue_depth`; không `prompt`/`text` |
Chạy lại TC-11 (mã lỗi), TC-31 (Embed 768 → `vectơ 768 chiều, cần 1536`, 250 chuỗi → 100/100/50, rỗng → 0 request), TC-29 (JSON sai), v1.3 "không theo chuyển hướng" (nhà giả trả 307 sang nhà khác: **0** request tới nơi chuyển, khoá không bị chuyển tiếp, audit `BAD_RESPONSE`): đều PASS. `go test -race -tags testroutes ./...` 0 FAIL.

## TC-41 (AC13) chạy lại — replay đã ghi (dev `5143d3e`; đo trên `fc7c920`)
- **TC-P102-41: PASS một phần / BLOCKED một phần.** `openai` (`gpt-4o-mini` 3 schema + `text-embedding-3-small` 1536 chiều) và `gemini` (3 schema + `gemini-embedding-001` `dimensions=1536`) **đã có bản ghi thật**; QC không có khoá nên **không** tự ghi lại, chỉ kiểm bản ghi:
  - `env -i go test ./internal/llm -run 'TestProviderContract|TestReplayHasNoSecrets|TestEmbedReplay' -v`: `TestProviderContract` **fake, fake-replay, openai, gemini PASS**; `anthropic` **SKIP** ("cần khoá thật để ghi" — **BLOCKED**, không FAIL); `TestEmbedReplay` openai, gemini PASS (1536 chiều, chuẩn L2 ≈ 1); `TestReplayHasNoSecrets` PASS; chạy không cần biến khoá (TC-40 cũng PASS).
  - Quét độc lập `testdata/replay/**`: `Authorization|Bearer |sk-…|AIza|org-|x-api-key|api_key` = **0**; bản ghi chỉ có nội dung JSON đã kiểm schema và `{model,dims,l2_norm}` (73–75 byte); `git grep` mẫu khoá ở toàn repo (ngoài `_test.go`) = **0**; `.env`/`.env.local` không bị theo dõi.
- **Verdict US-P1-02: PASS** (anthropic replay BLOCKED — chờ khoá thật của chủ dự án).
