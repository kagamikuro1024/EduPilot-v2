# QC sprint 10.5 — báo cáo L4–L5 (và kiểm lại L1–L3 sau sửa)

Stack `ep-qc` dựng từ `6f6ca1f6` (+ `f6f5246b` chỉ đổi test), bản `testroutes`. Provider decision `decision-fake`; sinh chữ qua stub QC (`scripts/stub_llm.py`) hoặc `fake`; `GEMINI_API_KEY` trống, `LLM_DECISION_ALLOW_REAL=false`. Log gateway/worker: 0 dòng `googleapis`; `llm_audit` 24 giờ chỉ có provider «Quyết định» (giả), «QC stub», `env:fake`.

## 1. Kiểm lại sau sửa (4ffffbcd, 3b902c37, b879d1f5)

| Mục | Kết quả |
| --- | --- |
| DM-BUG-1…4 (type=decision bị từ chối, TEACHER thấy provider/tác vụ DECIDE, test kết nối decision-fake, env không tới worker) | **Đã sửa**, kiểm lại PASS (`dm_a_provider.py`, `dm_c.py`) |
| L2 Threads, ma trận P × stub | PASS đủ: P=0,9 chặn FAST (0 CLASSIFY); P=0,01 cho qua FAST; P=0,2 / lạ → FALLBACK + CLASSIFY; stub `block` → 422 `PII_DETECTED`; `error`/`invalid` → FAIL_CLOSED 422; `slow` → 422 sau ≈ 6,4 s. Ghi chú: lượt trước các ca `block` trả 201 do **stub QC** (lược đồ mới là enum `no/yes`, stub chọn `no`); đã sửa stub, không phải lỗi sản phẩm |
| L3 chèn lệnh | P=0,9 → 14 cờ mới, điểm 16/16 không đổi; P=0,01 → 0 cờ mới; `MODE=error` → chấm vẫn xong, điểm y hệt; bài trống không gọi decision |
| DM-BUG-5 (tên test trong spec) | Các tên trong **AC2** (`TestInjectionOR`, `TestCallBudgetPerMessage`, …) nay có thật và PASS; tên thêm ở SRS §9 (`TestInjectionEmptySkipped`, `TestInjectionScoreUnchanged`, `TestInjectionSet6of6`, `TestRouteFallbackToRules`, `TestDecisionRBAC`, `TestDecisionGateVersionConflict`, `TestDecisionAudit`) vẫn không tồn tại (có test tương đương tên khác). **Thấp, tài liệu** |
| P-105-3 (+ `f6f5246b`) | **Xác nhận**: test cũ chỉ đổi số/danh sách theo lược đồ mới (migration 19→20, contract 304→310 thao tác, `TestClientSurface` 4→5, providers 20→21, `settings-llm.spec.ts` 4→5 phần / Test kết nối 2→3; `llm_test.go`: thêm cột `cascade_gate/cascade_path`, thêm `llm_audit_decide_idx`, đếm `llm_models` loại trừ `kind='decision'`). Không bỏ assertion, không nới. `student_draft_test.go` đổi sang `leaksNumber` có ranh giới theo P-105-2, vẫn kiểm |

## 2. L4 — chat `CHAT_ROUTING` (US-DM-05), `dm_d_chat.py`
Cổng bật ở lớp 761987; mỗi hàng 1 tin.

| Nhãn | specific (luật khớp) | vague | crisis |
| --- | --- | --- | --- |
| cổng tắt | 0 DECIDE | 0 DECIDE, 2 EMBEDDING | 0 DECIDE |
| P=0,97 | 0 DECIDE | **1** DECIDE `FAST` | 0 DECIDE, 0 gọi mô hình |
| P=0,5 | 0 DECIDE | 1 DECIDE `FALLBACK` | 0 DECIDE |
| decision `error` | 0 DECIDE | 1 DECIDE `FALLBACK` (`error`), vẫn trả lời | 0 DECIDE |
| câu lạ (0,25) | 0 DECIDE | 1 DECIDE `FALLBACK` | 0 DECIDE |

- ≤ 1 DECIDE / tin, làn `INTERACTIVE`; khủng hoảng không bao giờ tới decision; TTFB sự kiện SSE đầu 6–19 ms (≤ 300).
- **Quan sát (không phải lỗi)**: thời gian trả lời trọn câu `vague` tăng 41 ms (tắt) → 360–657 ms (bật) vì decision giả có p50 ≈ 400 ms và embedding giả tức thì. Điều kiện «TTFT p95 ≤ +20 %» của hướng dẫn sẽ do `benchmarks/decision/run.sh` đo ở lượt chung; trên bản giả không chắc đạt → cổng chat có thể ra «Giữ tắt».

## 3. L5 — quản trị, quan sát, cảnh báo
**API (`dm_e_api.py`)**: RBAC 403/401 đủ 6 endpoint × 3 vai; ngưỡng biên 200/422 cả ba cổng (ε 0,01–0,20; τ chat 0,70–0,99; τ chèn lệnh 0,30–0,90; timeout 200–5000 ms); gate lạ 404; `VERSION_CONFLICT` 409; `audit_log(entity='decision_gate')`; `effective=false` khi chưa bật hệ thống; `limit=500` → 422; `window` lạ → 422, `Cache-Control: no-store`.
- **DM-BUG-6 (Thấp)**: `PUT` với `threshold:"0,5"` (chuỗi) trả `400 BAD_REQUEST`, TC-D02-12 kỳ vọng `422`. Thân `{}` → 422; `enabled:"yes"` → 422; gate lạ → 404; không 500. Thân chỉ `{version}` → 200 không đổi gì.

**Số liệu `/admin/observability/decision` (`dm_f_alert_obs.py obs`)**: với 154 / 70 / 44 dòng `llm_audit`, calls, FAST %, FALLBACK %, FAIL_CLOSED % của cả 3 cổng **khớp SQL**; p50/p95/lỗi %/chi phí của THREADS khớp SQL (300 / 701,45 / 12,9 % / 0). Cổng tắt hiện «—».

**Cảnh báo `DECISION_FAIL_CLOSED`** (`DECISION_ALERT_TICK=10s`, `DEDUPE=6h`): chèn dữ liệu 1 giờ → THREADS 16/70 = 23 % và INJECTION 19/154 = 12 % sinh **đúng 2 chuông cho ADMIN** («Kiểm duyệt Threads đang chặn nhiều», «Cờ chèn lệnh khi chấm đang chặn nhiều», link `/observability`); CHAT 4,5 % (≤ 5 %) không báo; sau 50 s / 5 tick vẫn 2 (khử trùng); teacher / TA / SV không thấy chuông.

**Không lộ ra ngoài (TC-D02-23)**: 27 phản hồi `/threads`, `/chat`, `/assignments`, `/notifications` của SV/GV/TA — 0 trường `cascade_*`, tên cổng, `FAST/FALLBACK`.

**Đường nóng không đọc DB (TC-D02-14)**: `log_statement=all`, 8 bài đăng + 8 phiên chat → 665 câu lệnh, **0** chạm `decision_gates` / `course_decision_gates`. **Lan truyền cấu hình (TC-D02-16, một gateway)**: tắt/bật cổng qua API có hiệu lực ≤ 1,2 s; `TestConfigSnapshotRefresh` PASS. Hai bản gateway song song: **chưa kiểm** (cần scale).

**UI thật (`qc-dm-ui-real.spec.ts`, ảnh ở `shots/`)** 1440 + 375: khối «Quyết định nhanh» ở `/settings/llm` nằm sau «Chuỗi dự phòng», trước «Mô hình tìm kiếm tài liệu»; đủ 3 cổng, không tràn ngang, công tắc 36×20 (1440) / 44×44 (375), không từ cấm; hàng «Quyết định nhanh» trong bảng mức dùng. Bảng «Quyết định nhanh» ở `/observability` có cột Cổng · Trạng thái · Lời gọi · Nhanh · Dự phòng · Chặn an toàn · p50 · p95 · Lỗi · Chi phí, tab 24 giờ / 7 ngày, không tràn ngang. TEACHER mở `/settings/llm`, `/observability` không thấy khối. Ghi chú: phần trên của `/observability` (50 yêu cầu gần nhất, «Thư chưa gửi được») vẫn là dữ liệu mẫu — thuộc sprint 11 (US-P10-01), không tính vào 10.5; điều kiện «bảng nằm sau bảng 7 ngày» chưa kiểm được vì bảng 7 ngày chưa có.
Playwright của dev (`decision-settings`, `decision-observability`, `cascade`): 15 passed, 15 skipped (mobile tự bỏ theo thiết kế).

## 4. Chưa kiểm
Hai gateway song song (TC-D02-16); 1 triệu `llm_audit` EXPLAIN; PII trong payload tới provider thật (máy chủ giả `LLM_DECISION_BASE_URL`; có `TestDecisionProviderHTTP` của dev PASS); biên 19/20 từ ở stack; bình luận / sửa bài Threads, GV/TA đăng bài qua cổng; 6 biến thể chèn lệnh chấm trên stack (chạy ở `benchmarks/decision/run.sh`). Đo thật / lượt đo chung: **hoãn** theo PM.
