# Hướng dẫn vận hành: decision model và cascade (D60)

Dành cho người vận hành EduPilot (ADMIN). Trả lời: decision model làm gì, bật / tắt thế nào, chọn ngưỡng ra sao, đọc số ở đâu, xử lý sự cố, chi phí.

**Trước khi dùng dữ liệu sinh viên thật.** Gói miễn phí (Free Tier) của Gemini API dùng dữ liệu gửi vào để cải thiện sản phẩm của Google (`PRODUCTION_READINESS.md` §1). Chỉ dữ liệu mô phỏng đã che được phép gửi thật; triển khai thật **bắt buộc dùng khoá gói trả phí**. `LLM_DECISION_ALLOW_REAL` mặc định `false`: máy chủ không gọi Gemini thật cho tới khi bật biến này (và PM cho phép). Dev / test / CI / seed / demo / eval luôn dùng provider giả `decision-fake`.

## 1. Decision model làm gì
`gemini-decision-1` **chỉ trả lời câu hỏi đóng** (có / không, chọn một trong danh sách) kèm xác suất; không viết chữ. EduPilot dùng nó ở đúng ba **cổng**. Cổng chạy ở một lớp khi bật ở **cả hai** nơi: toàn hệ thống (`decision_gates`) và lớp đó (`course_decision_gates`), và provider decision sẵn sàng. Mặc định mọi cổng **tắt**; tắt hoặc lỗi hết thì hệ thống chạy như trước sprint 10.5 (riêng Threads còn có LLM `Structured` làm fallback).

| Cổng | Ở đâu | Decision quyết gì | Phân vân / decision lỗi | Cả hai lỗi |
| --- | --- | --- | --- | --- |
| `THREADS_PRECHECK` | Đăng bài Hỏi đáp công khai | Bài có hỏi chuyện riêng / nêu tên người cụ thể không. **Chỉ thêm chặn**, không mở lại bài luật đã chặn | Hỏi LLM `Structured` (hạn 6 s, `THREADS_STRUCTURED_TIMEOUT`); người dùng thấy «Đang kiểm tra» | **Chặn** (`FAIL_CLOSED`); ADMIN nhận cảnh báo `DECISION_FAIL_CLOSED` |
| `INJECTION_FLAG` | Chấm bài tự luận (việc nền) | Bài có câu nhắm vào người / máy chấm không. Chỉ **gắn cờ**, không đổi điểm | Cờ = regex OR `Structured` OR decision | Tầng decision vắng, chấm vẫn xong |
| `CHAT_ROUTING` | Chat riêng, bước phân loại | Câu hỏi thuộc loại nào (điểm, điểm danh, lịch thi…); chỉ gọi khi luật rơi về «hỏi đáp môn» | Giữ kết quả luật | Giữ kết quả luật |

Decision **không bao giờ** dùng cho: khủng hoảng (luôn bằng luật), điểm, mức rubric, công thức điểm, escalate (`TestDecideCallersAllowlist` giữ ranh giới này).

## 2. Bật / tắt
Mọi thao tác ở `/settings/llm` → khối **Quyết định nhanh** (chỉ ADMIN thấy). Cùng API (ADMIN): `GET /admin/llm/decision`, `PUT /admin/llm/decision/gates/{gate}`, `GET /admin/llm/decision/courses`, `GET|PUT /courses/{id}/decision-gates[/{gate}]`.

1. **Provider.** Một provider loại `decision` đang bật (seed có sẵn «Quyết định», model `decision-fake`). Chọn `gemini-decision-1` chỉ được khi `LLM_DECISION_ALLOW_REAL=true`; trần `LLM_DECISION_REAL_DAILY_CAP` (300 lời gọi / ngày), hết trần thì đi fallback. `Test kết nối` với bản giả chạy cục bộ, không ra mạng.
2. **Toàn hệ thống.** Công tắc từng cổng.
3. **Theo lớp.** Công tắc từng cổng của từng lớp (bảng «Các lớp»).
4. **Tắt khẩn.** Một công tắc: tắt cổng toàn hệ thống → mọi lớp về hành vi cũ ngay (snapshot làm mới trong ≈ 1 s qua Redis; không tải được thì cổng coi như tắt).

### Thứ tự bật thử đề xuất
1. Một lớp (lớp thử), theo thứ tự rủi ro tăng dần: `INJECTION_FLAG` → `THREADS_PRECHECK` → `CHAT_ROUTING`.
2. Mỗi cổng theo dõi 1–2 ngày ở `/observability` (mục 4) trước khi bật cổng kế.
3. Vài lớp, rồi toàn bộ, chỉ khi báo cáo đo trước / sau (mục 6) ghi **Bật được** cho cổng đó.

## 3. Ngưỡng (ε, τ)
Mỗi cổng có `threshold` và `timeout_ms` riêng (`decision_gates`), áp chung cho mọi lớp. Mặc định theo PoC 14/11/2026 (`docs/research/poc/jev-decision/cascade.md`).

| Cổng | Mặc định | `timeout_ms` | Ý nghĩa | Chỉnh |
| --- | --- | --- | --- | --- |
| `THREADS_PRECHECK` | ε 0,05 | 800 | P(có) ≤ ε: cho qua ngay; ε < P < 0,5: hỏi LLM; P ≥ 0,5: chặn | Giảm ε: hỏi LLM nhiều hơn (chậm, tốn hơn, ít sót hơn). Không đặt < 0,01 |
| `INJECTION_FLAG` | τ 0,5 | 3.000 | P(có) ≥ τ: gắn cờ | Giảm τ: nhiều cờ hơn, nhiều cờ nhầm hơn |
| `CHAT_ROUTING` | τ 0,95 | 800 | Độ tự tin ≥ τ: dùng nhãn decision; dưới τ: dùng luật | Ở PoC, 3 câu decision sai có độ tự tin 0,63–0,91: **không đặt dưới 0,92** khi chưa đo lại |

Đổi ngưỡng là đổi hành vi: đo lại (`benchmarks/decision/run.sh`) trước khi áp cho lớp thật.

## 4. Đọc số ở `/observability`
Bảng «Quyết định nhanh» (ADMIN), cửa sổ 24 giờ hoặc 7 ngày (`GET /admin/observability/decision?window=24h|7d`). Mỗi cổng: trạng thái, số lớp bật / tổng, số lời gọi, `FAST` / `FALLBACK` / `FAIL_CLOSED` (%), p50 / p95, tỉ lệ lỗi, chi phí. Không có nội dung câu hỏi (chỉ `llm_audit`).

| Số | Bình thường | Đáng lo |
| --- | --- | --- |
| `FAST` | THREADS ≈ 85–95 %, ROUTING ≈ 80–90 % (PoC) | Tụt mạnh: model đổi hành vi hoặc dữ liệu lớp khác dữ liệu đo |
| `FALLBACK` do lỗi | ≈ 0 | > 5 %: hạn mức, mạng hoặc khoá hỏng (xem `error_kind` ở mức dùng) |
| `FAIL_CLOSED` | 0 | Bất kỳ giá trị cao: ADMIN nhận chuông `DECISION_FAIL_CLOSED` (một việc nền mỗi `DECISION_ALERT_TICK`, không lặp trong `DECISION_ALERT_DEDUPE`) |
| p95 trễ decision | ≤ `timeout_ms` | Sát `timeout_ms`: nhiều ca hết giờ, chat chậm |

## 5. Sự cố
| Triệu chứng | Nguyên nhân thường gặp | Làm gì |
| --- | --- | --- |
| Sinh viên không đăng được bài sạch; chuông `DECISION_FAIL_CLOSED` | Decision **và** 9router cùng lỗi → `THREADS_PRECHECK` chặn | Tắt `THREADS_PRECHECK` của lớp (hoặc toàn hệ thống). Kiểm 9router (`curl 127.0.0.1:20128/v1/models`) và khoá |
| `RATE_LIMIT` nhiều / hết hạn mức | Vượt RPM / TPM gói, hoặc `LLM_DECISION_RPM` cao hơn gói | Giảm số lớp bật cổng hoặc nâng gói; `LLM_DECISION_RPM` (mặc định 150) phải thấp hơn RPM của gói. Hết hạn mức thì cổng tự đi fallback |
| Cầu dao mở (`circuit open`) | Lỗi liên tiếp | Cổng tự đi fallback; sửa nguyên nhân rồi `Test kết nối` |
| `AUTH` | Khoá sai / thu hồi | Dán khoá mới ở `/settings/llm` |
| Lọc an toàn của Google (`SAFETY_BLOCK`) | Nội dung nhạy cảm | Bình thường: cascade tự đi fallback |
| Chat chậm sau khi bật `CHAT_ROUTING` | Decision chậm hơn lần nhúng | Xem p95; tắt cổng nếu TTFT p95 tăng > 20 % |
| Bài bị chặn nhầm ở Threads | Decision / LLM quá nhạy | Ghi mẫu (đã che) vào bộ E1 `dev`, đo lại, cân nhắc đổi ε. Không sửa từng bài trong DB |

## 6. Đo trước / sau
`bash benchmarks/decision/run.sh` (cần stack + seed + `SEED_DEFAULT_PASSWORD`; provider decision giả; thoát mã 2 «Gateway chưa chạy» nếu không có gateway). Ba kịch bản bật / tắt cùng seed: chat 30 câu × 3 lượt (TTFT), 40 bài Threads (độ trễ đăng, chặn đúng / nhầm, `FAST` / `FALLBACK` / `FAIL_CLOSED`), chấm 12 bài (6 biến thể chèn lệnh + 6 đối chứng). Kết quả: `benchmarks/reports/decision-before-after.{json,md,svg}`, mỗi cổng một dòng **Bật được / Giữ tắt**:

- chat: TTFT p95 ≤ +20 % **và** độ chính xác ≥ luật;
- Threads: chặn nhầm ≤ 0,05 **và** không lọt thêm so với luật;
- chèn lệnh: 6/6 **và** 0 cờ nhầm trên 6 đối chứng.

Cổng không đạt giữ **tắt**. Lượt thật (≤ 300 lời gọi) chỉ chạy khi PM cho phép bằng văn bản và `LLM_DECISION_ALLOW_REAL=true`.

## 7. Chi phí `[ƯỚC TÍNH]`
- $0,10 / 1 triệu token vào; ra không tính tiền. Mỗi lời gọi ≈ 170–460 token tuỳ cổng (PoC).
- Chat T1 (1.000 sinh viên) ≈ 29.000 lời gọi / ngày ≈ **$1 / ngày**. Công thức: `lời gọi × token vào / 10⁶ × 0,10`.
- Ghi vào `llm_audit.cost_est` theo giá đặt ở model. Free Tier 200 RPM không đủ đỉnh 5 QPS; Tier 1: 1.500 RPM.
- Dev / test / demo / eval: `decision-fake`, không tốn tiền.

## 8. Câu lệnh thường dùng
```bash
# Seed: provider decision-fake + 3 cổng tắt; bật cả ba cho lớp 761987 để demo / E2E
SEED_DECISION_DEMO=1 node scripts/seed.mjs && SEED_DECISION_DEMO=1 node scripts/check-decision-seed.mjs
# Bộ test của decision
cd backend-go && go test ./internal/llm/... -run 'TestDecide|TestCascade' -v
cd backend-go && go test ./internal/thread ./internal/agent ./internal/grading -run 'TestPrecheckAddOnly|TestPlaceholderOnlyBlockedByRules|TestCrisisNeverReachesDecide|TestRouteDecisionNoOverride|TestDecideCallersAllowlist|TestInjectionOR|TestOneEmbedPerMessage|TestCallBudgetPerMessage' -v
cd backend-go && go test -tags testroutes ./internal/contract -run 'TestCascadeThreadsHTTP|TestDecisionProviderHTTP|TestDecisionReplayFile'
pnpm -C frontend exec playwright test decision-settings.spec.ts decision-observability.spec.ts cascade.spec.ts
# Dựng lại bản phát lại của provider giả sau khi đổi bộ dữ liệu benchmarks/decision/data
cd backend-go && UPDATE_DECISION_REPLAY=1 go test ./internal/contract -run TestDecisionReplayFile
```
Bản phát lại (`internal/llm/testdata/decision-replay.json`, 61 ca) khoá theo `sha256(đầu vào đã che):id câu hỏi`; câu không có trong bản phát lại trả 0,25 (lưỡng lự), hoặc `FAKE_DECISION_PROB` nếu đặt. Biến giả: `LLM_DECISION_FAKE_P50_MS` / `_P95_MS` / `_ERROR_RATE` / `_MODE` (`error`, `timeout`, `always_yes`, `always_no`). Cổng chèn lệnh chạy ở **worker**: các biến `LLM_DECISION_*` phải tới cả gateway và worker (compose đã chuyển).

## 9. Tham chiếu
`docs/DECISIONS.md` D60, D47(2) · `docs/specs/FEAT-decision-model/` · `docs/sprints/10.5/techlead.md` TL-1 · `docs/research/2026-11-14-jev-decision-model.md` · `docs/research/poc/jev-decision/cascade.md` · Google: `ai.google.dev/gemini-api/docs/decisions-api`, `…/pricing#gemini-decision`, `…/rate-limits#gemini-decision`.
