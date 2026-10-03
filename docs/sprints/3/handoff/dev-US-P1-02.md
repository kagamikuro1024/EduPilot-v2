# DEV handoff — US-P1-02 (cổng `internal/llm`)
Nhánh `sprint/3-pu-p1`. Đã làm theo `docs/research/2026-10-03-openai-go-compat.md` (SDK `openai-go/v3` v3.71.1, `WithMaxRetries(0)`, Structured theo loại) — xem `docs/sprints/3/proposals.md` #1–#3.

## Làm gì
- `internal/llm`: `Client` (Chat/Stream/Structured/Embed), `Request/Response/Chunk`, `ResolveLane`, lỗi gói (`ErrNotConfigured, ErrOverloaded, ErrUnavailable, ErrDeadline, ErrBadRequest, ErrAllProvidersFailed, ErrDimsMismatch, ErrBadLane, ErrStream`), danh tính + trace từ ctx (`identity.go`), `Registry` (dựng client từ DB, env dự phòng, hoán đổi nguyên tử, `Watch` pub/sub `ep:llm:reload` + 60 s), `Gateway` (hạn chót, thử lại jitter, fallback, một dòng `llm_audit`), `Auditor` (đệm 1.000, đẩy 1 s / 100 dòng, bỏ dòng cũ + `llm_audit_dropped`, đẩy hết khi đóng, ghi COPY), `jsonschema.go`, `gate.go` (điểm nối Scheduler cho US-P1-03; mặc định không giới hạn), `setup.go` (`Runtime`).
- `internal/llm/provider`: DUY NHẤT nơi import SDK; ánh xạ lỗi 7 loại + `BAD_RESPONSE/REPLAY_MISS`, `Retry-After(-Ms)`, `Redact`.
- `internal/llm/fake`: provider giả (độ trễ, tỉ lệ lỗi, khoá hợp lệ, `Stream` từng từ, `Embed` 1536 chiều L2 xác định, `Structured` sinh theo schema, phát lại theo sha256).
- `internal/llm/cost`: chi phí `decimal`.
- Route thử (`testroutes`): `POST _test/llm/chat` (JSON hoặc SSE), `GET _test/llm/stats`, `POST _test/llm/fake` (chỉ ADMIN); 6 mã lỗi mới trong `apierr`; `httpapi/llmhttp` ánh xạ lỗi → HTTP.
- Gateway nối vào `cmd/gateway/serve.go`; config `LLM_*`, `OPENAI/ANTHROPIC/GEMINI_API_KEY`, `FAKE_LLM_*`, `LLM_REPLAY_DIR` (`LLM_EMBED_DIMS≠1536` → từ chối khởi động); `.env.example` + `docker-compose.local.yml` đã thêm.

## AC tự đánh giá
| AC | Lệnh / kết quả thật |
| --- | --- |
| 1 | grep SDK ngoài `internal/llm/` = 0; grep tên miền = 0; `go.mod` chỉ thêm `openai-go/v3` (anthropic/genai = 0) |
| 2 | `TestClientSurface` PASS (so chữ ký bằng reflect); grep `UserID\|CourseID\|StudentCode` ở `types.go` = 0 |
| 3 | `TestStreamOneGeneration` (fake.Calls==1; Structured kèm là lời gọi riêng), `TestStreamCancel` (đóng ≤ 200 ms, `Active()==0`, audit `cancelled`) PASS `-race` |
| 4 | `TestRegistry` (5 loại × base URL, nhà tắt, nhà `unreadable` bị bỏ, chuỗi theo `fallback_order`) PASS trên Postgres thật |
| 5 | `TestErrorMapping` 16 ca (httptest: OpenAI/Anthropic/Gemini-mảng/HTML 502; Kind + Retryable + NextProvider + CountsToBreaker; đúng 1 lần gọi vì `MaxRetries(0)`; `Error()` không lộ thân) PASS |
| 6 | `TestRetryBackoff` (8 ca: INTERACTIVE 1, khác 2, 500 ms→1 s với jitter=trần, AUTH/404/BAD_REQUEST không thử lại), `TestRetryAfterHeader` (3 s chờ đúng 3 s; 30 s bỏ qua), `TestRetryRespectsDeadline` (còn < 1 s không thử lại) PASS |
| 7 | `TestFallbackOrder`, `TestNoFallbackOnBadRequest`, `TestAllFailed` PASS; log `warn` có `trace_id, task, from, to, error_kind` |
| 8 | `TestAuditRow`, `TestAuditAsync`, `TestAuditBatches`, `TestAuditDropOldest` (rơi đúng 400 dòng cũ nhất), `TestAuditFlushOnShutdown`, `TestAuditPGWriter` (COPY thật; không cột `prompt/content/message/text/response`) PASS |
| 9 | `TestTraceIDPropagation` PASS (audit + log cùng 32 hex của `traceparent`). Bước tay `curl … _test/llm/chat` + `select count(*) … trace_id=…` chưa chạy tay (cần stack compose); test Go thay thế |
| 10 | `TestValidateJSON`, `TestStructuredJSONSchema`, `TestStructuredInvalidFallsBack` + `TestStructuredRequestShape` (4 loại) PASS. Lưu ý: không "json_schema rồi lùi json_object" (xem góp ý #1) |
| 11 | `TestEmbedDims`, `TestEmbedBatchSplit` (250 → 100/100/50), `TestEmbedEmptyInput` PASS; `LLM_EMBED_DIMS=768 gateway serve` → rc=1, nêu tên biến |
| 12 | `go test ./internal/llm/fake` PASS, gồm `TestFakeNoNetwork` (bẫy `http.DefaultTransport`, `net.DefaultResolver.Dial`, `HTTP_PROXY` cổng đóng) |
| 13 | `TestProviderContract`: `fake` + `fake-replay` PASS; `openai/anthropic/gemini` = **BLOCKED** (SKIP "cần khoá thật để ghi"); `TestRecordReplay` chờ `LLM_RECORD=1` + khoá (Q11) |
| 14 | `TestReloadAtomic` (lời gọi đang chạy xong với cấu hình cũ, lời gọi mới dùng mới), `TestReloadKeepsOldOnError` PASS; `go test -tags integration -run TestReloadTwoProcesses` PASS (2 tiến trình, Redis thật, đổi sau ~20 ms) |
| 15 | `TestEnvFallback` (fake; 3 khoá → chuỗi OpenAI→Anthropic→Gemini + nhúng chỉ OpenAI; chỉ Anthropic → không nhúng), `TestNotConfigured` (4 hàm → `ErrNotConfigured`, không panic, không mạng) PASS |
| 16 | `TestKeyNeverLeaksViaErrors` ở `provider` (Detail ≤ 200, `[REDACTED]`) và ở `llm` (máy chủ phản chiếu `Authorization`: lỗi/log/audit không chứa canary, không khớp `sk-…`/`Bearer …`) PASS |
| 17 | contract `TestDefaultBuild_TestPathsAre404` (binary không tag: 404 cả 3 route mới) và kịch bản `-tags testroutes` (401 / TEACHER-STUDENT 403 / ADMIN 200) PASS |

Cổng: `go vet` (cả `testroutes`, `integration`) sạch; `golangci-lint run` và `--build-tags testroutes` 0 issues; `go test -race -count=1 -tags testroutes ./...` toàn bộ ok; `sqlc diff` rc=0.

## Nợ / ghi chú
- **BLOCKED:** ghi âm phản hồi thật (AC13) cần khoá thật — chủ dự án (Q11).
- Chưa làm tay bước `curl` của AC7/AC9/AC17 (cần stack compose); chạy ở cổng P1 hoặc US-P1-04.
- Hạn mức / hàng đợi / mạch / ngân sách / suy giảm / single-flight: US-P1-03 (đã chừa `Gate`).
- Contract: số thao tác của `openapi.test.yaml` 15→18 (góp ý #3). Enum `Error.code` của `openapi.test.yaml` thêm 6 mã; `openapi.yaml` sẽ thêm ở US-P1-04.

## Sửa lỗi QC (report-US-P1-02)
| BUG | Đã sửa | Tự kiểm |
| --- | --- | --- |
| BUG-P102-1 HTTP 504 bị xếp `TIMEOUT` | `provider.classify`: chỉ 408 → `TIMEOUT`; 504 rơi vào `>= 500` → `SERVER` (`llm_audit`: `error | SERVER`). `TIMEOUT` chỉ còn cho `DeadlineExceeded` / timeout mạng | `TestErrorMapping/504_là_SERVER` (đã sửa ca ghim sai); `go test -race ./internal/llm/provider` |
| BUG-P102-3 `GET /_test/llm/stats` sai hình dạng SRS 6.4 v1.2 | `llmrt.Runtime.Stats` trả đúng `{queue_depth, inflight, provider_inflight, circuit, fake_calls:{<provider>:n}, audit:{buffer_len, flushed, dropped}}`. `inflight` / `provider_inflight` / `circuit` lấy từ Scheduler nên đếm MỌI nhà đã qua `Admit` (kể cả `openai_compatible`), không chỉ fake. `fake.NewNamed` đếm lời gọi sinh văn bản theo từng provider; `Auditor.Flushed()` mới. `?lanes=1` thêm `inflight_batch` (chỉ cho `cmd/llmload`). `openapi.test.yaml` `LLMStats` đổi theo (`additionalProperties:false` nên hình dạng được contract test giữ). Lưu ý: số đếm là CỦA TIẾN TRÌNH — qua Caddy với 2 gateway mỗi lần gọi có thể ra bản khác | `TestCallsByProvider`, `TestStatsCountsAllProviders`, `TestAuditBatches` (flushed/pending/dropped), contract `stats` + `stats?lanes=1` validate theo spec; curl thật qua compose: khoá `['audit','circuit','fake_calls','inflight','provider_inflight','queue_depth']`, `fake_calls={'env:fake':1}` sau một lời gọi |
| BUG-P102-2 | Đóng theo góp ý #1 (PM ACCEPTED: Structured chọn cố định theo `type`, spec v1.3) — không đổi mã | – |

## Khớp spec v1.2/v1.3 (sau khi BA đổi spec)
- Thêm test theo tên mới của AC6/AC10/AC11: `TestSDKNoInternalRetry` (429 + `Retry-After: 3` → 1 lần gọi, không chờ), `TestRateLimitKeepsStatusOnDeadline` (`Retry-After: 30`, ctx 2 s → vẫn `RATE_LIMIT` 429), `TestStructuredDowngrade` (kiểm hành vi theo loại: nhà cung cấp trả 400 cho `json_schema` → đúng 1 yêu cầu, `BAD_REQUEST` thẳng, cả 4 loại), `TestEmbedGeminiNormalizesL2` (|v| = 1 ± 1e-6, `dimensions: 1536`).
- Provider không theo chuyển hướng (`CheckRedirect = ErrUseLastResponse`; `TestProviderNoRedirectFollow`: 302 → đúng 1 yêu cầu, đích không bị chạm). Lỗi SDK không giải mã được phản hồi → `BAD_RESPONSE` (trước là `NETWORK`).

## Ghi âm thật (AC13, AC11) — openai và gemini ĐÃ GHI; anthropic còn BLOCKED (chưa có khoá)
Chạy MỘT lần: `LLM_RECORD=1 OPENAI_API_KEY=… GEMINI_API_KEY=… go test ./internal/llm -run TestRecordReplay -v` (khoá lấy từ `.env.local` gitignored, không in ra log / handoff).
| Nhà | Mô hình | Lời gọi | Token (vào / ra) | Tệp |
| --- | --- | --- | --- | --- |
| openai | `gpt-4o-mini` (3 schema: `classify`, `formula`, `rubric`; `max_completion_tokens` 300) + `text-embedding-3-small` 1536 chiều | **4** (3 Structured + 1 nhúng; mỗi lời gọi 1 lần HTTP) | 335 / 334 (token vào gồm cả câu nhúng) | `testdata/replay/openai/{classify,formula,rubric,embed}.json` |
| gemini | `gemini-3.6-flash` (mặc định; cùng 3 schema; `max_tokens` 300) + `gemini-embedding-001` `dimensions=1536` | **4** ghi được + 1 lượt thất bại (3 lần thử của gateway, mô hình "suy nghĩ" ăn hết 300 token) + 1 lời gọi dò trực tiếp = **≤ 8** (< 10) | 48 / 301 ở lượt ghi được (lượt thất bại không có số liệu) | `testdata/replay/gemini/{classify,formula,rubric,embed}.json` |
- Bản ghi chỉ chứa NỘI DUNG JSON đã kiểm schema (+ `embed.json` = `{model, dims, l2_norm}`); không header, URL, thân yêu cầu, khoá hay mã tổ chức. `TestReplayHasNoSecrets` quét `testdata/replay` tìm `Authorization`, `Bearer `, `sk-`, `AIza`, `org-`, `x-api-key` ⇒ 0; `git check-ignore .env.local` ⇒ `.env.local`, không đưa vào commit.
- `TestProviderContract`: `fake`, `fake-replay`, **`openai`, `gemini` PASS** (cùng khoá, kiểu, bắt buộc với `fake`); `anthropic` SKIP "BLOCKED: cần khoá thật để ghi".
- AC11: `TestEmbedReplay` — openai: 1536 chiều, chuẩn L2 = 0,99982 (≤ 1 ± 1e-3); gemini: 1536 chiều, chuẩn L2 = 1,0000000238 (≤ 1 ± 1e-6, sau `Normalize`).
- Phát hiện (không đổi mã): với Gemini 3.x, `Structured` ở làn **không INTERACTIVE** (ví dụ `CLASSIFY` / `GRADING`) và `max_tokens` nhỏ (300) thất bại vì mô hình "suy nghĩ" dùng hết token (`Fast` chỉ bật ở làn INTERACTIVE); bản ghi dùng tác vụ `CHAT`. Khi đặt `max_tokens` cho tác vụ chạy nền, nên để ≥ 2.048 (US-P1-05 cho chỉnh `max_tokens` đến 32.768).
