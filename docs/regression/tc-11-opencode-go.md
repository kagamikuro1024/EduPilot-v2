# QC test case — OpenCode Go (P-11-106 · FEAT-llm-gateway US-P1-02 AC18 · SRS 4.2 · FR-33)
Nguồn: US-P1-02 AC18 (+ AC5 bảng lỗi, AC14 nạp lại nóng, AC16 khoá không lộ), SRS 4.2 đoạn «OpenCode Go», FR-33, P-11-106 (PM `446578fc`, BA `bf62e697`, dev `0e4bd078`). Người viết: qc3 · 2026-11-24.
Bản ghi mẫu: `type=openai_compatible`, `base_url=https://opencode.ai/inference/go/v1`, model chat `deepseek-v4.1-flash`. Khoá đọc từ `~/.config/edupilot-team/secrets/opencode-go.key` (KHÔNG in ra, KHÔNG ghi vào bằng chứng; bằng chứng chỉ ghi «khoá đã dùng», độ dài, 6 ký tự đầu `oc_sk_` khi cần).

Nhóm chạy:
- **LOCAL** — không cần stack: `go test`, `git grep`, chương trình Go dùng một lần (đặt tạm ở worktree chỉ đọc, xoá sau khi chạy), gọi thẳng OpenCode Go bằng `provider.NewOpenAI` với khoá thật.
- **STACK** — cần gateway có `0e4bd078` + Postgres + Redis: chạy trên `ep-qc` **sau khi hỏi QC** (không đụng lớp 761987/761988; lớp thử dùng mã `QCOC*`). `ep-r` hiện là bản cũ hơn `0e4bd078` nên không dùng cho nhóm này.

Ký hiệu API: `A=/api/v1/admin/llm`. Vai: ADMIN `admin@edupilot.local`, TEACHER `teacher@…`, TA `ta@…`, SV `sv.gioi@…` (mật khẩu seed).

## 1. Tiêu đề / tham số theo host (AC18 mệnh đề 1–2)

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Nhóm |
| --- | --- | --- | --- | --- | --- |
| TC-OC-01 | AC18 | `0e4bd078` | `cd backend-go && go test ./internal/llm/provider -run OpenCode -v -count=1` | `ok`, 5 test PASS (bốn trường hợp host × Chat và Stream) | LOCAL |
| TC-OC-02 | AC18 | Chương trình Go một lần: `NewOpenAI{openai_compatible, base_url=https://opencode.ai/inference/go/v1}` + Transport đổi đích về máy chủ `httptest` ghi header + thân | `Chat` với `Task=GRADING` | Header `User-Agent: edupilot/2.0`; `x-opencode-session: edupilot-grading` (chữ thường); thân có `"reasoning_effort":"low"` | LOCAL |
| TC-OC-03 | AC18 | Như TC-OC-02 | `Stream` với `Task=CHAT` | Cả ba có đủ: UA, `x-opencode-session: edupilot-chat`, `reasoning_effort=low`; luồng đọc hết, có usage cuối | LOCAL |
| TC-OC-04 | AC18 | Như TC-OC-02 | `Chat` với `Task=""` (rỗng) | `x-opencode-session: edupilot` (không có dấu `-` thừa) | LOCAL |
| TC-OC-05 | AC18 | Như TC-OC-02 | `Structured` (schema JSON) | Có UA; có `x-opencode-session` (đi qua `Chat`) và `reasoning_effort=low`; thân dùng `response_format: json_object` và schema nằm trong lời nhắc hệ thống (không dùng `json_schema`) | LOCAL |
| TC-OC-06 | AC18 (nhánh khác host) | Chương trình Go; `base_url=http://127.0.0.1:20128/v1` (9router) | `Chat` và `Stream` | KHÔNG có `x-opencode-session`; UA không phải `edupilot/2.0`; thân KHÔNG có `reasoning_effort` | LOCAL |
| TC-OC-07 | AC18 (host giả — an ninh) | `base_url` ∈ {`https://opencode.ai.evil.example/v1`, `https://api.opencode.ai/v1`, `https://opencode.ai@evil.example/v1`, `https://evil.example/opencode.ai/v1`, `http://localhost:8080/v1`} | `Chat` + `Stream` từng URL | Cả 5 URL: không UA `edupilot/2.0`, không `x-opencode-session`, không `reasoning_effort` (so `Hostname()` chính xác; khoá không bị gửi cho host lạ kèm cờ OpenCode) | LOCAL |
| TC-OC-08 | AC18 (cổng) | `https://opencode.ai:443/inference/go/v1` | `Chat` | Vẫn khớp (Hostname bỏ cổng): đủ ba | LOCAL |
| TC-OC-09 | AC18 | `provider` package | `go test ./internal/llm/provider -count=1` | Toàn bộ 63 test xanh (không phá 9router/OpenAI/Gemini) | LOCAL |
| TC-OC-10 | AC18 («không migration, không đổi enum») | `0e4bd078` | `git show --stat 0e4bd078`; `git diff 0e4bd078~1 0e4bd078 -- backend-go/db backend-go/api` | Chỉ `provider/openai.go` + `opencode_test.go`; không có file migration, không đổi `openapi.yaml`, enum `type` giữ nguyên 6 giá trị | LOCAL |

## 2. Cấu hình qua /settings/llm và phân quyền (AC18 + AC1/AC4)

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Nhóm |
| --- | --- | --- | --- | --- | --- |
| TC-OC-11 | AC18 | ADMIN có JWT; `ep-qc` | `POST A/providers` (Idempotency-Key) `{type:openai_compatible, name:"OpenCode Go", base_url:"https://opencode.ai/inference/go/v1", api_key:<khoá>, models:[{model:"deepseek-v4.1-flash",…}]}` | 201, `type=openai_compatible`, `base_url` đúng, `has_key=true`, `key_status=ok`, `version=1`; không có trường khoá | STACK |
| TC-OC-12 | AC18 + AC1 | Bản ghi TC-OC-11 | `POST A/providers/{id}/test` | 200 `{ok:true, latency_ms}`; `last_test.ok=true` ở GET kế tiếp | STACK |
| TC-OC-13 | AC18 (UI) | ADMIN đăng nhập `/settings/llm` | Thêm nhà: loại «Tương thích OpenAI», URL, khoá, nhấn «Test kết nối» rồi Lưu (Playwright, ảnh chụp) | Test báo OK ngay dưới hàng; sau Lưu hàng «OpenCode Go» xuất hiện, khoá không hiện lại; không có dòng giải thích thừa (CLAUDE.md luật chữ UI) | STACK |
| TC-OC-14 | AC18 (phân quyền) | TEACHER, TA, SV có JWT | Mỗi vai: `POST A/providers`, `PUT A/providers/{id}` (đúng `version`), `DELETE A/providers/{id}`, `POST A/providers/test`, `POST A/providers/{id}/test` | 403 `FORBIDDEN` cho cả 5 × 3 vai; DB không đổi (`version` còn 1, số nhà không tăng) | STACK |
| TC-OC-15 | AC18 (phân quyền) | Không token | `POST A/providers`, `GET A/providers` | 401 | STACK |
| TC-OC-16 | AC18 (phân quyền, UI) | TA và SV đăng nhập | Mở `/settings/llm`; TEACHER mở `/settings/llm` | TA, SV bị chặn (không vào được/không có mục menu); TEACHER xem được danh sách nhưng không có nút Thêm/Sửa/Xoá hay nút đó không thực hiện được (ma trận SRS 7) | STACK |
| TC-OC-17 | AC18 (nhánh lỗi) | ADMIN | Tạo lần hai cùng `name` «OpenCode Go»; tạo với `base_url` không hợp lệ (`ftp://…`, `https://169.254.169.254/…`); thiếu `Idempotency-Key` | Trùng tên → 409 `CONFLICT`; URL sai → 422 `INVALID_BASE_URL`; thiếu khoá idempotency → 4xx; không tạo bản ghi | STACK |
| TC-OC-18 | AC18 (idempotency) | ADMIN | Gửi cùng `POST A/providers` hai lần với CÙNG `Idempotency-Key` | Lần hai trả lại đúng kết quả lần một; DB chỉ có MỘT bản ghi | STACK |
| TC-OC-19 | AC18 (đồng thời) | ADMIN | `PUT A/providers/{id}` với `version` cũ | 409 `VERSION_CONFLICT`; bản ghi không đổi | STACK |

## 3. Khoá không lộ (AC18 cuối + AC16)

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Nhóm |
| --- | --- | --- | --- | --- | --- |
| TC-OC-20 | AC16 | `ep-qc`, bản ghi TC-OC-11 | `GET A/providers`, `GET A/providers/{id}`, thân 201 của POST, thân PUT; `grep -c oc_sk_` + kiểm không có khoá `api_key`/đuôi khoá | 0 lần `oc_sk_`; không có trường `api_key`, `key_last4`; chỉ `has_key`, `key_status` | STACK |
| TC-OC-21 | AC16 | Sau TC-OC-11…TC-OC-33 chạy | `docker logs` gateway + worker (toàn bộ, kể cả log lỗi của lời gọi) → `grep -c oc_sk_` | 0 | STACK |
| TC-OC-22 | AC16 | `ep-qc` Postgres | `SELECT api_key_enc FROM llm_providers WHERE name='OpenCode Go'`; tìm `oc_sk_` trong cột, `llm_audit`, `audit_log`, `idempotency` (dump văn bản) | Cột khoá là bản mã (không chứa `oc_sk_`); 0 chỗ khác chứa khoá | STACK |
| TC-OC-23 | AC16 | `git grep` | `git grep -n 'oc_sk_' -- .` và `git grep -nE 'opencode\.ai' -- ':!docs' ':!*_test.go'` trên `sprint/11-p10-eval`; kiểm `.env.example`, compose | 0 khoá; chỉ có hằng `openCodeUA`/host so khớp, không có khoá trong repo, `.env` mẫu, compose | LOCAL |
| TC-OC-24 | AC16 | Chương trình Go: máy chủ giả trả 401 kèm thân chứa đúng khoá gửi lên; và 500 kèm khoá | `Chat` qua `base_url` host opencode.ai (Transport đổi đích) | `Error.Detail` đã bôi khoá (`Redact`), không còn khoá nguyên văn; `Kind=AUTH` / `SERVER` | LOCAL |
| TC-OC-25 | AC16 | Chương trình Go | Gọi thật OpenCode Go bằng khoá đúng, in `err`/log của chương trình | Đầu ra chương trình không chứa khoá (dò `oc_sk_` trong stdout/stderr = 0) | LOCAL |

## 4. Chuyển tuyến, nạp lại nóng (AC18 + AC14)

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Nhóm |
| --- | --- | --- | --- | --- | --- |
| TC-OC-26 | AC14 + AC18 | `ep-qc` có nhà 9router + OpenCode Go; ghi `StartedAt` container gateway + worker | ADMIN `PUT A/routes {task:CHAT, chain:[deepseek-v4.1-flash của OpenCode Go], version}`; ngay sau đó SV gửi một tin chat; đọc `llm_audit` mới nhất | `llm_audit.provider='OpenCode Go'`, `model='deepseek-v4.1-flash'`, `status='OK'`; tin trả lời có chữ; `StartedAt` gateway/worker KHÔNG đổi (không dựng lại) | STACK |
| TC-OC-27 | AC14 + AC18 | Sau TC-OC-26 | ADMIN `PUT A/routes` đưa CHAT về 9router (đúng `version`); đo từ lúc PUT 200 tới lời gọi chat kế tiếp | Lời gọi kế tiếp ghi `provider=9router`; đổi trong ≤ 1 s (AC14); không dựng lại; không còn lời gọi nào đi OpenCode sau thời điểm đổi | STACK |
| TC-OC-28 | AC14 | Gateway + worker cùng chạy | Đặt `GRADING` sang OpenCode Go bằng PUT; kích hoạt chấm từ worker (hàng đợi) | Cả tiến trình worker lẫn gateway dùng tuyến mới sau ≤ 1 s (pub/sub `ep:llm:reload`); `llm_audit` của lời gọi chấm ghi OpenCode Go | STACK |
| TC-OC-29 | AC18 (phân quyền) | TEACHER, TA, SV | `PUT A/routes` (CHAT → OpenCode Go) | 403; `GET A/routes` của TEACHER vẫn 200, tuyến không đổi | STACK |
| TC-OC-30 | AC18 (nhánh lỗi) | ADMIN | `PUT A/routes` với `model_id` không tồn tại; với `chain` rỗng; với `version` cũ | 422 / 422 / 409; tuyến cũ vẫn chạy (gọi chat OK) | STACK |
| TC-OC-31 | AC18 | Tuyến CHAT = [OpenCode Go] | Xoá nhà OpenCode Go khi đang được tuyến dùng | 409 `PROVIDER_IN_USE`; bản ghi còn nguyên | STACK |

## 5. Gọi thật (AC18 — nhà cung cấp thật)

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Nhóm |
| --- | --- | --- | --- | --- | --- |
| TC-OC-32 | AC18 | Khoá thật; chương trình Go gọi `provider.NewOpenAI` tới `https://opencode.ai/inference/go/v1` | `Chat` «Trả lời đúng một từ: xin chào», model `deepseek-v4.1-flash` | Không lỗi; có nội dung; `usage` tokens_in/out > 0; thời gian < 30 s | LOCAL |
| TC-OC-33 | AC18 | Như TC-OC-32 | `Stream` cùng câu hỏi | Có ≥ 1 delta chữ trước khi kết thúc; usage cuối > 0; kênh đóng | LOCAL |
| TC-OC-34 | AC18 | Như TC-OC-32 | `Structured` với schema `{score:number(0–10), comment:string}` cho một bài làm mẫu | Đầu ra là JSON hợp lệ khớp schema (`score` số trong khoảng, `comment` chuỗi không rỗng); thử 3 lần liên tiếp đều hợp lệ | LOCAL |
| TC-OC-35 | AC18 (luồng chấm) | `ep-qc`; tuyến GRADING = OpenCode Go; lớp thử `QCOC1` có bài tập có rubric và một bài nộp | Giảng viên chạy «Chấm AI» cho 1 bài nộp; đợi job xong | Bài ở trạng thái điểm nháp (KHÔNG tự công bố); `rubric_scores` là JSON hợp lệ khớp schema; `llm_audit` có dòng `task=GRADING`, `provider='OpenCode Go'`, `model='deepseek-v4.1-flash'`, `status='OK'`; điểm tính bằng code (không nằm trong lời LLM) | STACK |
| TC-OC-36 | AC18 + luật 4 | `ep-qc`; nhà «CapOC» kiểu `openai_compatible` trỏ về máy chủ bắt thân do QC dựng; tuyến CHAT = CapOC | SV gửi tin chứa họ tên và MSSV của chính mình | Thân gửi ra ngoài KHÔNG chứa tên/MSSV thật (chỉ placeholder); `llm_audit.pii_masked_count` ≥ 1 | STACK |

## 6. Nhánh lỗi nhà cung cấp (AC18 nhánh lỗi + AC5)

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Nhóm |
| --- | --- | --- | --- | --- | --- |
| TC-OC-37 | AC18 lỗi | Chương trình Go; khoá SAI (`oc_sk_` + chuỗi ngẫu nhiên) tới OpenCode Go thật | `Chat` với hạn 20 s | Trả về trong < 20 s (không treo), `Kind=AUTH`, `Detail` không chứa khoá | LOCAL |
| TC-OC-38 | AC18 lỗi + AC1 | `ep-qc` | `POST A/providers/test` với khoá sai; rồi `POST A/providers` với khoá sai | Test: 200 `{ok:false,error_kind:"AUTH"}` (không lưu); POST: 422 `PROVIDER_AUTH_FAILED`, không ghi bản ghi; UI: «Chưa lưu gì» | STACK |
| TC-OC-39 | AC18 lỗi + AC5 | Chương trình Go + `httptest` (host opencode.ai qua Transport): lần lượt đáp 401, 403, 404, 400, 429 (`Retry-After: 2`), 500 | `Chat` từng mã | Kind theo bảng: 401/403→`AUTH`; 404→`MODEL_NOT_FOUND`; 400→`BAD_REQUEST`; 429→`RATE_LIMIT` (có `RetryAfter`); 500→`SERVER`; đều trả về < 1 s, không tự thử lại bên trong SDK (đúng 1 request mỗi lần gọi) | LOCAL |
| TC-OC-40 | AC18 lỗi | Chương trình Go; model không có ở vùng (`deepseek-v9-khong-co`) gọi OpenCode Go thật | `Chat` | Lỗi nhà cung cấp (`MODEL_NOT_FOUND` / `AUTH` / `BAD_REQUEST` tuỳ phản hồi thật; ghi mã nhận được), trả về < 30 s, không treo | LOCAL |
| TC-OC-41 | AC18 lỗi + AC5 | `ep-qc`; tuyến CHAT = [OpenCode Go với khoá hỏng (bản ghi `skip_verify`, ADMIN), 9router] | SV gửi chat | Lời gọi vẫn thành công qua 9router; `llm_audit`: `fallback_index=1`, `provider=9router`, dòng lỗi trước đó `error_kind=AUTH`; không treo | STACK |
| TC-OC-42 | AC18 lỗi + AC5 | `ep-qc`; nhà «Stub429» (máy chủ QC luôn trả 429 `Retry-After: 1`) cùng tuyến [Stub429] rồi [Stub429, 9router] | Gửi 6 chat liên tiếp | `RATE_LIMIT` được thử lại 1 lần theo `Retry-After` rồi chuyển nhà tiếp; sau 5 lỗi liên tiếp mạch `circuit=open` (hiện ở `GET A/providers`) và lời gọi kế tiếp bỏ qua nhà đó ngay; sau `LLM_BREAKER_OPEN` (30 s) về `half_open`/`closed` | STACK |
| TC-OC-43 | AC18 lỗi | Chương trình Go; máy chủ giả giữ kết nối không trả lời | `Chat` với hạn 2 s | Trả về ≤ 3 s với `Kind=TIMEOUT`; không treo | LOCAL |
| TC-OC-44 | AC18 lỗi (an ninh) | Chương trình Go; máy chủ giả đáp 302 sang `http://127.0.0.1:1/steal` | `Chat` (HTTP client mặc định của `NewOpenAI`) | Không theo chuyển hướng; lỗi `BAD_RESPONSE`; máy chủ đích không nhận được request/khoá | LOCAL |

## Phân quyền (tổng)
`POST/PUT/DELETE A/providers*`, `POST A/providers/test`, `PUT A/routes`: chỉ ADMIN (TC-OC-14, 15, 29). `GET A/providers`, `GET A/routes`: ADMIN + TEACHER. TA, SV: 403 mọi thao tác. Khoá không bao giờ có trong phản hồi (TC-OC-20).

## Truy vết AC18
Mệnh đề header/effort → TC-OC-01…05; host khác → TC-OC-06…08; không migration → TC-OC-10; đổi tuyến → TC-OC-26…28; lỗi model/vùng, 429/cầu dao → TC-OC-39…42; khoá không lộ (AC16) → TC-OC-20…25; gọi thật → TC-OC-32…36.
