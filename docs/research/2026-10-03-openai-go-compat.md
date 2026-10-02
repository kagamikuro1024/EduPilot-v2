# openai-go và lớp tương thích OpenAI của Anthropic, Gemini (P1, SRS FEAT-llm-gateway 4.2)

**Câu hỏi.** Với `openai-go` bản mới nhất, gọi OpenAI / Anthropic / Gemini qua `base_url` thì `response_format: json_schema`, `stream_options.include_usage`, mã lỗi + `Retry-After`, `max_tokens` / `max_completion_tokens`, và embedding 1536 chiều hành xử ra sao — chỗ nào spec / mã P1 phải để ý?

**Kết luận.**
1. **Dùng `github.com/openai/openai-go/v3` v3.71.1** (2026-10-02). Module không hậu tố dừng ở v1.12.0 (2025-07), `/v2` dừng ở v2.7.1 (2025-09).
2. **Bắt buộc `option.WithMaxRetries(0)`** cho mọi client. Mặc định SDK tự thử lại 2 lần với 408/409/429/5xx và lỗi mạng, tôn trọng `Retry-After` tới **2 phút**. PoC: một 429 với `Retry-After: 3` thành **3 lần gọi, 6 s**, chồng lên lần thử lại của Scheduler. Với `Retry-After: 30` và hạn 2 s thì lỗi trả về là `context.DeadlineExceeded`, **mất** status 429 → bị xếp nhầm là `TIMEOUT`.
3. **Anthropic bỏ qua `response_format` hoàn toàn**, kể cả `json_object` (tài liệu chính thức). SRS 4.2 ghi "`Structured` luôn dùng `json_object` + kiểm schema" thì với Anthropic `json_object` vô tác dụng. Nên ép một lời gọi tool (`tools` + `tool_choice` được hỗ trợ đầy đủ), lấy `arguments` rồi kiểm schema; vẫn chỉ một lời gọi (D47).
4. **Hai mô hình dự phòng mặc định trong SRS 4.2 đã bị gỡ khỏi API:** `claude-3-5-haiku-*` ngừng từ 2026-02-19, `gemini-2.0-flash` tắt từ 2026-06-01.
5. Mọi dạng thân lỗi thử được (OpenAI, Anthropic, mảng kiểu Google, HTML 502) đều ra `*openai.Error` có `StatusCode`. Nhưng `Code` **rỗng** với Anthropic / Gemini → `MODEL_NOT_FOUND` phải dựa vào status 404, không dựa vào mã `model_not_found`. Lỗi **giữa stream** là `*ssestream.StreamError`, **không có status**.
6. `include_usage`: cả ba đều có trong tài liệu; chunk cuối có `choices: []` — đọc `Choices[0]` sẽ panic.

Độ chắc chắn: **cao** cho hành vi SDK (mã nguồn + PoC). Hành vi thật của từng nhà cung cấp chỉ dựa trên tài liệu, chưa gọi API thật vì không có khoá; chỗ nào tài liệu không nói thì ghi `[SUY LUẬN]`.

## Bảng tổng hợp theo nhà cung cấp

| Hạng mục | OpenAI (`api.openai.com/v1`) | Anthropic (`api.anthropic.com/v1/`) | Gemini (`…/v1beta/openai/`) |
| --- | --- | --- | --- |
| `response_format: json_schema` | Có (strict) | **Bị bỏ qua** | Có (ví dụ `parse(response_format=…)` trong tài liệu) |
| `response_format: json_object` | Có | **Bị bỏ qua** | [SUY LUẬN] có — tài liệu compat không liệt kê |
| Cách nên dùng cho `Structured` | `json_schema` + `strict: true` | tool bắt buộc + kiểm schema | `json_schema` + kiểm schema |
| `stream_options.include_usage` | Có | "Fully supported" | Có (ví dụ trong tài liệu) |
| `max_completion_tokens` | Có (`max_tokens` đã lỗi thời, không dùng được với mô hình suy luận) | Có | [SUY LUẬN] dùng `max_tokens` — tài liệu compat không nêu |
| `temperature` | 0–2 | 0–1, >1 bị **kẹp** về 1 | 0–1 (trang troubleshooting) |
| Lỗi HTTP | thân `{"error":{code,type,…}}` | thân giữ dạng `{"error":{type,message}}`, `code` rỗng | [SUY LUẬN] không có tài liệu cho lớp compat; trang troubleshooting gợi ý khoá sai có thể ra **400** chứ không 401 |
| Header thử lại | `Retry-After-Ms` / `Retry-After` (SDK đọc cả hai) | `retry-after` "Fully supported" + `x-ratelimit-*` | [SUY LUẬN] không cam kết header; Google khuyên tự backoff |
| Embedding | `text-embedding-3-small` (1536) | **không có** endpoint | `gemini-embedding-001` / `-2` mặc định 3072; [SUY LUẬN] `dimensions: 1536` được chuyển thành `output_dimensionality` |
| Tình trạng lớp compat | — | "not considered a long-term or production-ready solution" | "still in beta" |

## Bằng chứng

**Phiên bản.** `proxy.golang.org`: `github.com/openai/openai-go/v3/@latest` → `v3.71.1` (2026-10-02T16:46Z); `/v2` → v2.7.1 (2025-09-30); gốc → v1.12.0 (2025-07-30); `/v4` không tồn tại. `go.mod` của v3.71.1: `go 1.25.0`, có require Azure/AWS SDK nhưng chỉ cho gói con `azure/`, `bedrock/` [SUY LUẬN: không import thì không vào binary — kiểm lại ngưỡng image 40 MB khi dev thêm thư viện].

**Mã nguồn openai-go v3.71.1** (`$GOMODCACHE/github.com/openai/openai-go/v3@v3.71.1`):
- `internal/requestconfig/requestconfig.go:285` `MaxRetries: 2`; `:388–418` `shouldRetry`: lỗi kết nối, `x-should-retry`, 408, 409, 429, ≥ 500; `:446–455` đọc `Retry-After-Ms` rồi `Retry-After` (giây hoặc ngày giờ HTTP); `:33` trần chờ theo header `defaultMaxRetryAfterDelay = 2 * time.Minute`; `:556–575` backoff 0,5 s × 2^n, trần 8 s, jitter 25 %; `:693, :722` ctx hết hạn trong lúc chờ → trả `ctx.Err()` (mất phản hồi 429).
- `:743–760` mọi status ≥ 400 → `apierror.Error{StatusCode, Request, Response}` (`:755`), đọc `error` trong thân bằng gjson; `Response.Header` dùng được để đọc `Retry-After`.
- `internal/apierror/apierror.go:36, 44–51`: `Error()` chỉ in "OpenAI API error: 429 Too Many Requests" — không lộ URL / khoá (hợp SRS 8.4); nội dung thô nằm ở `RawJSON()`.
- `packages/ssestream/ssestream.go:227–236`: dữ liệu SSE có khoá `error` → `*StreamError{Message: "received error while streaming: " + error}`, không có status; `Message` chứa **nguyên văn** lỗi của nhà cung cấp → phải bôi trước khi log.
- `chatcompletion.go:3822, 3830`: `MaxCompletionTokens` / `MaxTokens` (bản sau ghi "deprecated … not compatible with o-series"); `:3382` `IncludeUsage`; `:4281–4282` `OfJSONSchema` / `OfJSONObject`; `embedding.go:149` `Dimensions`.

**Tài liệu nhà cung cấp** (truy cập 2026-10-03):
- Anthropic, [OpenAI SDK compatibility](https://platform.claude.com/docs/en/cli-sdks-libraries/libraries/openai-sdk): "`response_format` — Ignored. For JSON output, use Structured Outputs with the native Claude API"; "`strict` … is ignored"; `tool_choice` "Fully supported"; `max_tokens`, `max_completion_tokens`, `stream_options` "Fully supported"; `temperature` "Values greater than 1 are capped at 1"; header `retry-after`, `x-ratelimit-*` "Fully supported"; "Most unsupported fields are silently ignored rather than producing errors"; ghi chú đầu trang: "not considered a long-term or production-ready solution".
- Anthropic, [Model deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations): `claude-3-5-haiku-20241022` Retired 2026-02-19 → thay bằng `claude-haiku-4-5-20251001`. Bản thay này đang Active nhưng có thể bị gỡ sớm nhất từ **2026-10-15**.
- Google, [OpenAI compatibility](https://ai.google.dev/gemini-api/docs/openai) (cập nhật 2026-09-02): ví dụ dùng `gemini-3.8-flash`; structured output qua `response_format`; embeddings `gemini-embedding-2-preview` / `gemini-embedding-001`; ví dụ có `stream_options={'include_usage': True}`; "Support for the OpenAI libraries is still in beta". Gemini 3.x mặc định bật "thinking" (trang troubleshooting) → tốn thêm thời gian và token; `reasoning_effort` ánh xạ sang `thinking_level`.
- Google, [Embeddings](https://ai.google.dev/gemini-api/docs/embeddings): MRL, `output_dimensionality`, mặc định 3072; "If you are using `gemini-embedding-001`, you must manually normalize non-3072 dimensions"; `gemini-embedding-2` tự chuẩn hoá. Trang compat **không** nói `dimensions` của OpenAI có được chuyển sang không.
- Google, [Deprecations](https://ai.google.dev/gemini-api/docs/deprecations): `gemini-2.0-flash` tắt 2026-06-01 → thay bằng `gemini-3.6-flash`.
- OpenAI, [Deprecations](https://developers.openai.com/api/docs/deprecations): `gpt-4o-mini` (chat) và `text-embedding-3-small` **không** có trong danh sách bị gỡ.

**PoC** `/tmp/research-oai/poc_test.go`: máy chủ giả `httptest` trả đúng dạng thân / header theo tài liệu từng nhà; dạng mảng `[{"error":…}]` và HTML 502 là trường hợp xấu tự đặt. Lệnh: `cd /tmp/research-oai && go test -count=1 -v ./...` (openai-go v3.71.1, Go 1.27.1). Kết quả thật:

```
mặc định (MaxRetries=2), Retry-After 3           lần gọi=3    6.0s  *openai.Error=true   status=429 Retry-After="3"
WithMaxRetries(0), Retry-After 3                 lần gọi=1    0.0s  *openai.Error=true   status=429 Retry-After="3"
mặc định, Retry-After 30, hạn ctx 2 s            lần gọi=1    2.0s  *openai.Error=false  status=0   DeadlineExceeded=true
WithMaxRetries(0), Retry-After 30, hạn ctx 2 s   lần gọi=1    0.0s  *openai.Error=true   status=429 Retry-After="30"
anth401    *openai.Error status=401 code="" type="authentication_error"
anth404    *openai.Error status=404 code="" type="not_found_error"
gem404arr  *openai.Error status=404 code="" type=""
html502    *openai.Error status=502 code="" type=""
json_schema + MaxTokens → {…,"max_tokens":256,"response_format":{"json_schema":{"name":"out","strict":true,"schema":{…}},"type":"json_schema"}}
json_object + MaxCompletionTokens → {…,"max_completion_tokens":256,"response_format":{"type":"json_object"}}
embed dimensions=1536 → body {"input":"x","model":"gemini-embedding-001","dimensions":1536}; len=1536
không include_usage    chunks=3 (choices rỗng=0) text="Xin chào" usage=0/0 err=<nil>
include_usage          chunks=4 (choices rỗng=1) text="Xin chào" usage=11/2 err=<nil>
lỗi giữa stream        chunks=1 text="Xin" err=received error while streaming: {"type":"overloaded_error","message":"Overloaded"} StreamError=true
```

## Ảnh hưởng (việc của dev / BA qua PM)

- `internal/llm/provider/`: `openai.NewClient(option.WithBaseURL(u), option.WithAPIKey(k), option.WithMaxRetries(0))`. Thử lại do Scheduler làm theo SRS 4.2. Có thể thêm test hợp đồng: máy chủ giả trả 429 → đúng **1** lần gọi.
- Ánh xạ lỗi (SRS 4.2): lấy `errors.As(err, &*openai.Error)` → `StatusCode`; `Retry-After` đọc từ `ae.Response.Header`, ưu tiên `Retry-After-Ms`. `MODEL_NOT_FOUND` = **404** (bỏ điều kiện "mã `model_not_found`" vì Anthropic / Gemini để `code` rỗng). `*ssestream.StreamError` → `SERVER` (thử lại / chuyển nhà **chỉ khi chưa phát token nào** cho client; đã phát rồi thì kết thúc stream có lỗi). Gemini: 400 có thông điệp về API key → [SUY LUẬN] nên xếp `AUTH` ở "Test kết nối", nếu không người dùng sẽ thấy "yêu cầu không hợp lệ" thay vì "khoá sai".
- `Structured` (FR-12, 02-AC10) theo `type`: `openai` → `json_schema` strict (schema phải đặt `additionalProperties: false`, mọi thuộc tính `required`); `gemini` → `json_schema`; `anthropic` → tool bắt buộc (`tool_choice` = hàm duy nhất), lấy `tool_calls[0].function.arguments`; `openai_compatible` → `json_object`. Mọi nhánh đều kiểm schema phía Go. Chọn theo `type` cố định, **không** thử `json_schema` rồi lùi khi gặp 400 (sẽ thành hai lời gọi, trái D47). Đây là sửa dòng `anthropic` của SRS 4.2.
- `Stream`: luôn bật `IncludeUsage`; bỏ qua chunk `len(Choices)==0` (chỉ lấy `Usage`); dùng `ChatCompletionAccumulator` hoặc tự cộng. Nhà nào không trả usage → token ra = ước tính (ghi `llm_audit` là ước tính) [SUY LUẬN cho `openai_compatible`].
- `max_tokens`: gửi `max_completion_tokens` cho `openai` / `anthropic`, `max_tokens` cho `gemini` / `openai_compatible` [SUY LUẬN cho hai loại sau].
- `temperature`: kẹp 0–1 cho `anthropic` / `gemini` khi gửi (SRS 5.4 cho phép 0–2).
- Embedding: giữ "chỉ OpenAI" mặc định. Nếu Admin cấu hình Gemini thì gửi `dimensions: 1536`, kiểm độ dài (đã có `DIMS_MISMATCH` / `MODEL_DIMS_MISMATCH`), và **chuẩn hoá L2** khi mô hình là `gemini-embedding-001`.
- Mặc định env (`internal/llm/defaults.go`, SRS 4.2): đổi `claude-3-5-haiku-latest` → `claude-haiku-4-5-20251001`, đổi `gemini-2.0-flash` → `gemini-3.6-flash`. Gemini 3.x có thinking: với làn INTERACTIVE gửi `reasoning_effort: "low"` (hoặc `minimal` với bản Flash), nếu không sẽ khó đạt SLO trễ của D47. Theo dõi lịch gỡ `claude-haiku-4-5-20251001` (sớm nhất 2026-10-15).
- Không mở lại D46: OpenAI vẫn là nhà chính; Anthropic / Gemini chỉ là dự phòng, nên nhãn "beta / không cho production" của họ chấp nhận được, ghi vào rủi ro P10.

## Đề xuất cho PM

`P1 openai-go: dùng openai-go/v3 v3.71.1 với WithMaxRetries(0) (mặc định SDK tự thử lại 2 lần, tới 2 phút, làm mất status 429 khi hết hạn — PoC); sửa SRS 4.2: Anthropic bỏ qua response_format nên Structured của anthropic = tool bắt buộc + kiểm schema, MODEL_NOT_FOUND = 404 (code rỗng ở Anthropic/Gemini), lỗi giữa stream không có status; đổi mặc định dự phòng claude-3-5-haiku-latest → claude-haiku-4-5-20251001 và gemini-2.0-flash → gemini-3.6-flash (hai mô hình cũ đã bị gỡ) — docs/research/2026-10-03-openai-go-compat.md.`
