# Handoff Dev — Sprint 10.5 (Decision model + cascade, D60)

Nhánh `sprint/10.5-decision`. Mỗi lát một commit `10.5: L<n> …`. Cập nhật theo từng lát.

## Bản decision giả (`decision-fake`) — bảng ca replay (P-105-1 Q2/Q3)

Bản giả không gọi mạng, tất định. Khoá phát lại = `sha256(đầu vào ĐÃ CHE) + ":" + id câu hỏi`, nạp từ `backend-go/internal/llm/testdata/decision-replay.json` (nhúng vào binary; Go test thêm bằng `Controller.AddDecision(maskedInput, qid, DecisionRecord)`).

| Đầu vào | Câu hỏi | Kết quả | Đường đi cascade |
| --- | --- | --- | --- |
| Có trong replay, `noul` ≥ 0,5 | `noul` | P(có) = giá trị replay (kẹp 0,01–0,99) | `AddOnly`: FAST (có) |
| Có trong replay, `noul` ≤ ε | `noul` | như trên | FAST (không) |
| Có trong replay, ε < `noul` < 0,5 | `noul` | như trên | FALLBACK |
| Có trong replay, `choice` conf ≥ τ | `choice` | nhãn + xác suất của replay | `Symmetric`: FAST |
| **Không có trong replay (đầu vào lạ)** | `noul` | **0,25** (trung tính, dưới ngưỡng "có", trên ε 0,01–0,20) | FALLBACK |
| Không có trong replay | `choice` | các nhãn chia đều, confidence = 1/n | FALLBACK |
| không có trong replay, `FAKE_DECISION_PROB=p` (replay thắng) | `noul` / `choice` | `noul` = p; nhãn đứng đầu (theo bảng chữ cái) có confidence p | theo p |
| bất kỳ, `LLM_DECISION_FAKE_MODE=always_yes` / `always_no` | `noul` | 0,99 / 0,01 | FAST |
| bất kỳ, `…MODE=error` | — | HTTP 503 giả | FALLBACK (hoặc FAIL_CLOSED nếu fallback cũng lỗi) |
| bất kỳ, `…MODE=timeout` | — | treo tới hết hạn ctx | FALLBACK |

Độ trễ log-chuẩn: `LLM_DECISION_FAKE_P50_MS` / `…_P95_MS` (mặc định 426 / 682); tỉ lệ lỗi `LLM_DECISION_FAKE_ERROR_RATE`.

## Base URL decision (P-105-1 Q1)

Bản thật gọi `POST {base}/decisions` (header `x-goog-api-key`). `{base}` lấy theo thứ tự: cột `base_url` của bản ghi provider `type=decision` → env `LLM_DECISION_BASE_URL` → Google. QC dựng máy chủ giả và trỏ vào đây. Mô hình thật chỉ chạy khi `LLM_DECISION_ALLOW_REAL=true` VÀ provider chọn mô hình `gemini-decision-1`; mặc định `409 DECISION_REAL_DISABLED`.

## Biến môi trường mới

`LLM_DECISION_TIMEOUT` (800ms), `LLM_DECISION_ALLOW_REAL` (false), `LLM_DECISION_REAL_DAILY_CAP` (300/ngày UTC+7), `LLM_DECISION_RPM` (150), `LLM_DECISION_BASE_URL`, `LLM_DECISION_FAKE_{P50_MS,P95_MS,ERROR_RATE,MODE}`, `FAKE_DECISION_PROB`.

## Điểm spec ≠ TL-1 (spec FEAT-decision-model thắng)

- Tên cổng `THREADS_PRECHECK` / `INJECTION_FLAG` / `CHAT_ROUTING`; `cascade_path` ∈ FAST | FALLBACK | FAIL_CLOSED.
- Không thêm `DECIDE` vào `llm_task_routes`: chọn provider bằng đúng MỘT hàng `llm_providers` `type='decision'` đang bật (unique index).
- Cấu hình cổng ở hai bảng `decision_gates` / `course_decision_gates` (không phải cột trên `courses`); ngưỡng ở `decision_gates`.
- `Client` có 5 hàm: `TestClientSurface` đổi từ 4 → 5 (đúng SRS 4.1). Tên kiểu cổng là `llm.DecideGate` (`llm.Gate` đã là giao diện Scheduler).
- Migration seed một nhà `decision` nên `ListProviders` trả thêm 1 hàng: `TestNoNPlusOne` đổi `Len 20 → 21`; hạn mức 20 nhà không tính nhà `decision` (`CountLLMProviders`).

## P-105-2 (đã làm)

- `TestStudentNeverSeesDraftGrade`: «6.37» khớp vào dấu thời gian `…46.379254Z` (dương tính giả theo giờ chạy). Không bề mặt nào lộ điểm nháp (kiểm lại cả 8 trạng thái × 7 bề mặt + tool tóm tắt). Phép kiểm số đổi từ chuỗi con sang `leaksNumber` (số đứng riêng, ranh giới chữ số / dấu chấm; thử cả dấu phẩy); chuỗi canary chữ và các khoá cấm giữ nguyên. Thêm `TestLeaksNumber` (dấu thời gian không báo nhầm; số thật, chuỗi `"6,37"`, `"được 6.37 điểm"` vẫn bị bắt, kể cả khi cùng body có dấu thời gian vô hại).
- Hai test sửa ở L1 theo spec: `TestClientSurface` 4→5 hàm; `TestNoNPlusOne` `Len 20→21` (seed nhà `decision`).
