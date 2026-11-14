# QC test case — US-DM-01 (`Decide()` + provider), US-DM-02 (cascade + cấu hình)
Nguồn: US.md + SRS §3.1, 3.5, 3.6, 4.1–4.6, 5, 6; Q-D-1…6, 15. Hộp đen; quy ước: `tc-00-chung.md` (**provider giả, không Gemini thật**). Ký hiệu: `GATE` ∈ {THREADS_PRECHECK, INJECTION_FLAG, CHAT_ROUTING}; `DEC=/admin/llm/decision`.

## US-DM-01 `Decide()` và provider
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D01-01 | AC1 | Repo | Đọc `internal/llm` (giao diện `Client`, `DecideRequest`, `DecideResponse`) | `Client` có **5** hàm gồm `Decide(ctx, DecideRequest)`; `Questions` là kiểu đóng `noul` / `choice` / `score` (không `map`/JSON tự do/chuỗi chỉ thị từ người dùng); response có `probabilities`, `confidence`, `Usage{TokensIn, TokensOut=0}` | DIFF |
| TC-D01-02 | AC2 | Repo | `go list -deps ./... \| grep -E 'google.golang.org/genai\|generative-ai-go'`; `git diff <gốc>... -- backend-go/go.mod go.sum` | 0 SDK; không module mới (chỉ `net/http`) | DIFF |
| TC-D01-03 | AC2 | Máy chủ giả cục bộ làm base URL (Q-S105-1), `ALLOW_REAL=true` **chỉ trong test cục bộ**, khoá giả `test-key` | Một `Decide` thật-đường-mã → bắt yêu cầu | `POST /v1beta/decisions`, header `x-goog-api-key: test-key`, thân `{model:"gemini-decision-1", input, questions}`; không header khác chứa khoá; không gọi ra ngoài | GO/API |
| TC-D01-04 | AC2 | Cùng | Khoá lấy từ bản ghi `llm_providers` (mã hoá) rồi xoá bản ghi → rơi về `GEMINI_API_KEY` (giả) | Thứ tự đúng; khoá không xuất hiện ở log, `llm_audit`, phản hồi API (`grep`) | API+DB |
| TC-D01-05 | AC3 | `decision-fake`, đầu vào có trong bản phát lại | Gọi `Decide` 5 lần cùng đầu vào | Kết quả **y hệt** 5 lần (xác suất, nhãn); không mạng (lsof / không DNS) | GO |
| TC-D01-06 | AC3 | Đầu vào **không** có trong bản phát lại | `Decide` `noul` / `choice` | `noul = 0,5`; `choice` đều nhau (confidence thấp) → ép fallback ở cascade | GO |
| TC-D01-07 | AC3 | `LLM_DECISION_FAKE_P50_MS=100`, `P95=150`; rồi `ERROR_RATE=0,5` | 200 lời gọi | p50 / p95 `latency_ms` ở `llm_audit` ≈ cấu hình (±20 %); tỉ lệ lỗi ≈ 50 % (±10 điểm); mặc định 426 / 682, lỗi 0 | API+DB |
| TC-D01-08 | AC3 | `FAKE_MODE=error|timeout|always_yes|always_no` (mỗi chế độ) | 1 lời gọi mỗi chế độ | `error` → lỗi, `timeout` → quá hạn cổng, `always_yes` → P(có)=1, `always_no` → 0; đúng như tên | API |
| TC-D01-09 | AC4 | Bật cổng THREADS ở lớp 1 | Đăng bài ở lớp 1; đọc `llm_audit` | Có đúng **một** dòng `task='DECIDE'`: `provider`, `model`, `tokens_in`, `latency_ms`, `status`, `error_kind`, `cost_est`, `cascade_gate`, `cascade_path`; **không** cột / trường chứa nội dung | API+DB |
| TC-D01-10 | AC4 | Đăng bài chứa tên + MSSV thật của SV seed | Bắt thân tới provider (máy chủ giả) hoặc đọc `masked` qua log test | Thân yêu cầu **không** có tên / MSSV (chỉ placeholder); `llm_audit.pii_masked_count` > 0 | API+DB |
| TC-D01-11 | AC4 | – | `llm_audit.lane`/làn: Threads, chat → `INTERACTIVE`; chèn lệnh → `BATCH` | Đúng làn từng cổng | DB |
| TC-D01-12 | AC4 | `LLM_DECISION_RPM=10` | 30 lời gọi trong 10 s | Token bucket: ≥ ~20 lời gọi nhận `rate_limited` → đi fallback ngay (không xếp hàng ở `INTERACTIVE`, không chờ); độ trễ chat không tăng | API+DB |
| TC-D01-13 | AC5 | `FAKE_P50_MS=2000` (hơn hạn 800 ms) | Chat 1 câu / đăng 1 bài | Lời gọi hết hạn ở ~800 ms (`status=timeout`/`error`), **không thử lại** (đúng 1 dòng `DECIDE`); cổng đi fallback; chat không chậm hơn ~800 ms + tối đa | API+DB |
| TC-D01-14 | AC5 | Request gốc còn < 800 ms hạn | Gọi qua đường có hạn ngắn | `Decide` không vượt phần hạn còn lại | GO |
| TC-D01-15 | AC6 | Máy chủ giả trả lần lượt `429`, `500`, treo, `400 safety`/`prohibited_content`, lỗi mạng | `Decide` mỗi ca | `rate_limited` / `error` / `timeout` / `error_kind=SAFETY_BLOCK` / `error`; `status` là giá trị đã có của `llm_audit` (không thêm giá trị lạ: `SELECT DISTINCT status`) | GO+DB |
| TC-D01-16 | AC6 | Cầu dao `decision` mở (chuỗi lỗi) | `Decide` | `circuit_open` ngay (không mạng) | API |
| TC-D01-17 | AC6 | `llm_budgets` hết | `Decide` | `budget_blocked`; cascade đi fallback | API |
| TC-D01-18 | AC6, AC10 | Khoá trống | `Decide` | `not_configured` trong **< 1 ms** (đo ở test), không treo; `llm_audit` một dòng / lần gọi, không lặp | GO+DB |
| TC-D01-19 | AC7 | `ALLOW_REAL=false` (mặc định) | `PUT /admin/llm/providers/{id}` chọn model `gemini-decision-1` | `409 DECISION_REAL_DISABLED`; không đổi; `audit_log` không ghi thành công giả | API |
| TC-D01-20 | AC7 | Bản ghi provider đã `gemini-decision-1` sẵn (ghi DB, ghi rõ) + `ALLOW_REAL=false` | Gọi cổng | `Decide` trả `not_configured`, **không gọi mạng**; cổng đi fallback | API+DB |
| TC-D01-21 | AC7 | `ALLOW_REAL=true`, `REAL_DAILY_CAP=3`, máy chủ giả | 5 lời gọi cùng ngày (UTC+7) | 3 đầu qua; lời 4, 5 → `budget_blocked` → fallback; sang ngày (đổi giờ giả lập / `created_at`) → đếm lại | GO |
| TC-D01-22 | AC7 | – | Đọc cấu hình mặc định `.env.example`, compose, seed | `LLM_DECISION_ALLOW_REAL` mặc định `false`; không file nào đặt `true`; `REAL_DAILY_CAP=300` | DIFF |
| TC-D01-23 | AC8 | – | `SELECT model, price_in, price_out FROM llm_models WHERE kind='decision'` | `gemini-decision-1`: `price_in=2600`, `price_out=0` VND/triệu token; `decision-fake`: 0; `cost_est` của lời gọi fake = 0; bản thật (máy chủ giả, 330 token) ≈ 330×2600/1e6 ≈ 0,86 VND | DB |
| TC-D01-24 | AC9 | – | `go test ./internal/llm/... -run 'TestDecide' -v` | Đủ 6 test (`TestDecideAudit`, `NoSDK`, `Masked`, `ErrorMap`, `RealGuard`, `FakeDeterministic`) chạy và PASS (không «no tests to run»); QC đọc từng test: không assert rỗng | GO |
| TC-D01-25 | AC10 | Provider `decision` bị tắt | Chat / đăng bài / chấm | Mọi cổng như trước sprint 10.5; `llm_audit task='DECIDE'` = 0 | API+DB |
| TC-D01-26 | RBAC | – | Teacher / TA / STUDENT: `GET /admin/llm/providers`, `POST …/providers/{id}/test` | `403` | API |
| TC-D01-27 | AC6 | – | `POST /admin/llm/providers/{id}/test` (ADMIN) với provider `decision-fake` | `200`; thử bằng một câu `noul` vô hại; không PII; **không gọi Gemini** (fake) | API |
| TC-D01-28 | SRS §5.1 | DB trống | `goose up`; thử tạo hai provider `type='decision'` cùng `enabled` | Chỉ mục `llm_providers_one_decision` chặn cái thứ hai; `type` `decision` và `kind` `decision` hợp lệ; seed có `decision-fake` + `gemini-decision-1`; `goose down` rồi `up` sạch | DB |

## US-DM-02 Cascade + cấu hình
Bảng chính sách dùng `FAKE_MODE` / bản phát lại có nhãn P(có) cho từng ô (Q-S105-2). `AddOnly{ε=0,05}`: P ≥ 0,5 → có (FAST); P ≤ ε → không (FAST); giữa → fallback. `Symmetric{τ=0,95}`: confidence ≥ τ → FAST; dưới → fallback.
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D02-01 | AC1, AC8 | – | `go test ./internal/llm/... -run 'TestCascadeTable\|TestCascadeFailClosed' -v` | PASS; log nêu các ô policy × xác suất × lỗi; QC đọc bảng test: biên 0,05 (≤ ε nhận), 0,5 (≥ nhận), 0,949 / 0,95 (τ), 0,0499 | GO |
| TC-D02-02 | AC1 | Threads bật; đầu vào P=0,5 / 0,51 / 0,06 / 0,05 / 0,049 | Quan sát `Verdict` + `cascade_path` | 0,5 và 0,51 → có, FAST; 0,05 và 0,049 → không, FAST; 0,06 → FALLBACK (vùng giữa) | GO/API |
| TC-D02-03 | AC2 | Bốn tình huống: decision OK; decision lỗi + fallback OK; lưỡng lự + fallback OK; decision lỗi + fallback lỗi | Từng cổng | `Path` = FAST / FALLBACK / FALLBACK / FAIL_CLOSED; `llm_audit.cascade_path` khớp; giá trị an toàn: Threads **chặn**, chèn lệnh **không thêm cờ từ tầng này**, chat **giữ luật** | API+DB |
| TC-D02-04 | AC2 | – | Mỗi lần cascade | `llm_audit`: **một** dòng `DECIDE`; lời gọi fallback là dòng riêng với task gốc (`CLASSIFY` / `UTILITY`) | DB |
| TC-D02-05 | AC2 | `400 safety` từ provider (máy chủ giả) | Threads | Coi là lỗi decision → fallback, **không** thành kết luận «có» (không chặn do SAFETY_BLOCK) | GO |
| TC-D02-06 | AC3 | Seed sạch | Gọi từng cổng ở lớp 1 | `decision_gates.enabled=false` và không hàng `course_decision_gates` → **0** lời gọi `DECIDE`; hành xử như trước 10.5 | API+DB |
| TC-D02-07 | AC3 | Bật toàn hệ thống, **chưa** bật lớp 1 | Đăng bài / chat / chấm ở lớp 1 | 0 lời gọi `DECIDE` | API+DB |
| TC-D02-08 | AC3 | Bật lớp 1, **tắt** toàn hệ thống | Như trên | 0 lời gọi | API+DB |
| TC-D02-09 | AC3 | Bật cả hai (lớp 1) | Như trên; rồi cùng việc ở lớp 2 | Lớp 1 có `DECIDE`; lớp 2 **không** | API+DB |
| TC-D02-10 | AC3 | Bật cả hai, provider tắt / khoá trống / `ALLOW_REAL` chặn | Như trên | Hiệu lực = false (SRS 4.4): 0 lời gọi, như trước | API+DB |
| TC-D02-11 | AC4 | – | `PUT $DEC/gates/{gate}` với `threshold` và `timeout_ms` ở biên: THREADS ε 0,01 / 0,20 / 0,009 / 0,201; CHAT τ 0,70 / 0,99 / 0,69 / 0,991; INJECTION τ 0,30 / 0,90 / 0,29 / 0,91; `timeout_ms` 200 / 5000 / 199 / 5001 | Trong khoảng `200`; ngoài khoảng `422 VALIDATION_FAILED` kèm `field` | API |
| TC-D02-12 | AC4 | – | `PUT` với `gate` lạ; thân rỗng; kiểu sai (`"0,5"` chuỗi) | `404` gate lạ; `422` còn lại; không `500` | API |
| TC-D02-13 | AC4 | – | Đọc mặc định sau seed: `GET $DEC` | THREADS ε 0,05 / 800; INJECTION τ 0,5 / 3000; CHAT τ 0,95 / 800; `enabled=false` cả ba | API |
| TC-D02-14 | AC5 | Bật cổng; bật `log_statement=all` ở DB thử (hoặc `pg_stat_statements`) | 50 tin chat / bài đăng | **0 truy vấn** tới `decision_gates` / `course_decision_gates` trên đường request (chỉ lúc nạp / làm mới); không lỗi khi DB cấu hình chậm | DB |
| TC-D02-15 | AC5 | Không nạp được cấu hình (chặn truy cập bảng, khởi động gateway) | Chat / đăng bài | Coi như tắt: 0 `DECIDE`; không 5xx | API |
| TC-D02-16 | AC5, AC6 | Hai bản gateway (hoặc Go test) | ADMIN bật cổng; đo thời gian tới khi bản kia có hiệu lực | ≤ 5 s qua `ep:decision:cfg`; chặn Redis pub/sub → ≤ 60 s (TTL) | API/GO |
| TC-D02-17 | AC6 | – | `PUT` đổi `enabled` / ngưỡng; `audit_log` | `version` +1; `audit_log(entity='decision_gate', before, after, actor_id)`; cùng giao dịch | API+DB |
| TC-D02-18 | AC6 | – | Hai admin cùng `PUT` với `version` cũ | Người sau `409 VERSION_CONFLICT`; không ghi đè; không dòng `audit_log` | API |
| TC-D02-19 | AC6 | – | `PUT /courses/{id}/decision-gates/{gate}` đổi lớp; lớp không tồn tại / UUID rác | `200` + `audit_log(entity='course_decision_gate')`; `404`; `409` khi `version` sai | API+DB |
| TC-D02-20 | AC7 | Cầu dao `decision` mở; cổng bật | Mọi cổng | Đi thẳng fallback (`FALLBACK`), không mạng, không chờ (đo độ trễ ≈ lúc tắt cổng) | API |
| TC-D02-21 | AC8 | – | `AddOnly`: bài mà luật đã chặn nhưng decision nói «không» P=0,01 | Vẫn **chặn**; không đường nào gỡ chặn | GO/API |
| TC-D02-22 | AC9 | – | Teacher / TA / STUDENT / không token: `GET/PUT $DEC…`, `GET/PUT /courses/{id}/decision-gates…` | `403` / `401`; lớp khác với ADMIN: `404` khi lớp lạ | API |
| TC-D02-23 | AC9 | Cổng bật ở lớp 1 | Mọi phản hồi của `/threads`, `/chat`, `/grading` (SV, GV, TA) | **Không** có trường / khoá `cascade_*`, tên cổng, ngưỡng, `FAST/FALLBACK` (`grep` trên ≥ 30 phản hồi); chỉ ADMIN thấy ở API cấu hình | API |
| TC-D02-24 | AC3 | – | `GET /courses/{id}/decision-gates` (ADMIN) | Ba cổng: `enabled`, `effective`, `version`; `effective` = bật hệ thống ∧ lớp ∧ provider sẵn sàng | API |
| TC-D02-25 | AC6 | – | `GET $DEC/courses?gate&q&cursor&limit` | Cursor duyệt hết không trùng / sót; `limit=500` → kẹp 100 hoặc `422`; tìm `q` theo tên / mã | API |

## Nhánh lỗi
TC-D01-13, 15…18, 20, 21; TC-D02-05, 12, 15, 18, 20.

## Phân quyền
TC-D01-26; TC-D02-22, 23.

## Script chạy được
Pha 2: `docs/sprints/10.5/qc/scripts/dm_core.py`.

## Lịch sử sửa TC
(chưa có)
