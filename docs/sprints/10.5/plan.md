# Sprint 10.5: Decision model + cascade (D60)

Chủ dự án 2026-11-14: tách decision model ra sprint riêng, làm NGAY (trước khi chạy tiếp bench 9.5 và các đợt còn lại của sprint 11), **viết tài liệu kỹ**. Nhánh `sprint/10.5-decision`, tách từ `sprint/11-p10-eval` (đã có D60, `cascade.md`, research, FLOWS / ARCHITECTURE / SYSTEM_DESIGN / PRODUCTION_READINESS đợt 6).

## Phạm vi (theo P-11-10, P-11-11, D60)
- `internal/llm`: `Decide()` + provider loại `decision` (chỉ `gemini-decision-1`, `POST /v1beta/decisions`, `net/http`, không SDK); làn + token bucket riêng trong Scheduler; PII placeholder như mọi lời gọi; `llm_audit` ghi lời gọi decision (không nội dung); provider giả `decision-fake` tất định cho test / CI / demo.
- Cascade 3 cổng, mỗi cổng có cờ bật / tắt theo lớp (mặc định tắt) và ngưỡng cấu hình:
  1. Tường lửa Threads: decision chỉ được THÊM chặn (ε = 0,05); lỗi / hết hạn → LLM Structured; cả hai lỗi → chặn (fail-closed).
  2. Cờ chèn lệnh khi chấm: regex OR Structured (P-11-9) OR decision (τ = 0,5); chỉ gắn cờ, không đổi điểm.
  3. Định tuyến chat: decision ≥ τ (0,95) thì dùng; dưới ngưỡng / lỗi → định tuyến bằng luật hiện có (không thêm lời gọi LLM, giữ D47(1)).
- Cấm: khủng hoảng (US-P3-04 AC13, `TestCrisisNeverReachesDecide`), điểm, mức rubric, escalate.
- Màn: `/settings/llm` thêm khối provider decision (chỉ ADMIN), công tắc cổng theo lớp; `/observability` hiện số lời gọi decision, tỉ lệ đi đường nhanh, tỉ lệ fallback.
- Đo: trước / sau (TTFT chat, trễ kiểm duyệt Threads, chi phí) bằng provider giả có độ trễ + một lượt thật NHỎ (≤ 300 lời gọi) chỉ khi PM cho phép.

## Tài liệu (bắt buộc, «viết kỹ»)
| Tài liệu | Nội dung | Ai |
| --- | --- | --- |
| `docs/specs/FEAT-decision-model/` US + SRS + QUESTIONS | Tách từ US-P10-13; đủ 11 mục SRS, sơ đồ tuần tự từng cổng, bảng lỗi, cấu hình, quyền | ba |
| `docs/DECISIONS.md` D60, D47(2) | đã có; rà lại khớp mã | techlead |
| `docs/ARCHITECTURE.md`, `SYSTEM_DESIGN.md`, `PRODUCTION_READINESS.md` | đã có; bổ sung bảng cấu hình / env / làn Scheduler cuối cùng | techlead |
| `docs/FLOWS.md` F3, F4, F9 + mục Decision cascade | đã có; rà lại sau khi có mã | ba |
| `docs/research/2026-11-14-jev-decision-model.md`, `cascade.md` | giữ làm nguồn số liệu | techlead |
| `docs/guides/decision-model.md` (mới) | hướng dẫn vận hành: bật / tắt cổng, chọn ngưỡng, đọc số trên `/observability`, xử lý sự cố, chi phí | dev + techlead |
| `backend-go/api/openapi.yaml` | endpoint / trường mới | dev |
| Quyển đồ án | chương kiến trúc + luồng + đánh giá | writer sau merge |

## Thứ tự
BA spec (APPROVED) → QC TC → dev theo lát: L1 `Decide()` + provider + fake + Scheduler → L2 cổng Threads → L3 cổng chèn lệnh (gộp P-11-9) → L4 cổng định tuyến chat → L5 màn + observability → L6 đo trước / sau + guide → QC kiểm → merge `main`.
Sau đó: chạy tiếp bench 9.5 (1.000 bài, R1–R6 qua 9router) có decision bật, rồi sprint 11 đợt 1–5 + smoke hồi quy.

## Luật chạy
LLM sinh chữ chỉ qua 9router. Decision thật và embedding thật: chỉ khi PM cho phép (tốn khoá Gemini). Tối đa demo + 1 stack.
