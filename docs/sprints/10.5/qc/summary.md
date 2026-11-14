# QC sprint 10.5 (D60, decision model) — kết luận

Nhánh `sprint/10.5-decision`, HEAD kiểm `f6f5246b` (mã L1–L6). Stack `ep-qc` + `decision-fake` + stub QC; không Gemini, không `gemini-decision-1`, không 9router trực tiếp cho decision. Chi tiết: `report-L1.md`, `report-L2-L3.md`, `report-L4-L5.md`.

| Story | Kết luận | Căn cứ |
| --- | --- | --- |
| US-DM-01 Provider decision | **PASS** | migration 00020 up/down/up sạch; `type=decision` lưu / bật; `gemini-decision-1` BẬT → 409 `DECISION_REAL_DISABLED`; TEACHER không thấy provider / tác vụ DECIDE; Test kết nối ok |
| US-DM-02 Cổng, cấu hình, hiệu lực | **FAIL (Thấp)** | DM-BUG-6: thân sai kiểu (`threshold:"0,5"`) trả 400, TC-D02-12 kỳ vọng 422. Còn lại PASS: RBAC, biên ngưỡng, 409, `audit_log`, hiệu lực hệ thống ∧ lớp, 0 truy vấn `decision_gates` trên đường nóng, tắt / bật ≤ 1,2 s |
| US-DM-03 Cổng Threads | **PASS** | P × stub ma trận đủ (chặn / cho qua FAST; FALLBACK + Structured; FAIL_CLOSED 422; hết giờ ≈ 6,4 s → 422); chỉ thêm chặn |
| US-DM-04 Cờ chèn lệnh | **PASS** | P=0,9 → 14 cờ mới, điểm không đổi; P=0,01 → 0; decision lỗi → chấm vẫn xong; bài trống không gọi; `TestInjectionOR` 7 nhánh PASS |
| US-DM-05 Cổng chat | **PASS** | specific / crisis → 0 DECIDE; vague → 1 DECIDE (FAST 0,97 / FALLBACK 0,5, lỗi, câu lạ); ≤ 1 DECIDE / tin; TTFB ≤ 19 ms |
| US-DM-06 UI cấu hình | **PASS** | `/settings/llm` 1440 + 375: khối đúng vị trí, 3 cổng, công tắc ≥ 44 px ở 375, không tràn; TEACHER không thấy; playwright dev 15 pass |
| US-DM-07 Quan sát + cảnh báo | **PASS** | số khớp SQL (calls, %, p50, p95, lỗi, chi phí); chuông `DECISION_FAIL_CLOSED` đúng 2 / 3 cổng vượt ngưỡng, khử trùng, chỉ ADMIN |
| US-DM-08 Seed, hướng dẫn, cổng nghiệm thu | **PASS** (đo trước / sau hoãn) | `check-decision-seed.mjs` mặc định PASS; `SEED_DECISION_DEMO=1` PASS (cả ba cổng bật ở lớp 761987, chạy lại không đổi); `cascade.spec.ts`, `decision-*.spec.ts` PASS; hướng dẫn đọc khớp hành vi; bộ test AC2 có tên thật |
| US-DM-09 Ranh giới an toàn | **PASS** | allowlist `Decide`, crisis không tới decision, `TestDecideMasked`, ≤ 1 DECIDE / 1 EMBED / tin PASS; không bảng nào lưu nội dung decision; `go.mod`/`package.json` không đổi; `.env.example` `LLM_DECISION_ALLOW_REAL=false`; 0 gói ra `googleapis` |

**Tổng: 8/9 PASS, 1 FAIL Thấp (DM-BUG-6).** P-105-3 và `f6f5246b` xác nhận là cập nhật theo lược đồ / số liệu mới, không nới assertion.

## Lỗi còn mở
| # | Mức | Mô tả |
| --- | --- | --- |
| DM-BUG-6 | Thấp | `PUT /admin/llm/decision/gates/{gate}` thân sai kiểu → 400 `BAD_REQUEST`, TC kỳ vọng 422. Đề xuất: sửa sang 422 hoặc proposal đổi TC-D02-12 thành 400 |
| Tài liệu | Thấp | `docs/ARCHITECTURE.md` ghi «thiếu khoá → `noul = 0,5`» nhưng chạy thật và guide là 0,25 (P-105-1); SRS §9 còn nêu tên test không tồn tại; seed `SEED_DECISION_DEMO=1` vẫn in «3 cổng tắt» |

## Quan sát cho lượt đo chung
Trên bản giả, bật `CHAT_ROUTING` làm thời gian trả lời trọn câu `vague` tăng 41 → 360–657 ms (decision giả p50 ≈ 400 ms, embedding giả tức thì), ngược tiêu chí «TTFT p95 ≤ +20 %» của hướng dẫn; kết luận «Bật được / Giữ tắt» thuộc `benchmarks/decision/run.sh`.

## Hoãn / chưa kiểm (không tính FAIL)
`benchmarks/decision/run.sh` (đo trước / sau 3 kịch bản), lượt thật `gemini-decision-1`, smoke hồi quy toàn bộ — chờ lượt đo chung / phép PM. Chưa kiểm: hai gateway song song, EXPLAIN 1 triệu `llm_audit`, PII trong payload tới máy chủ giả provider thật, biên 19/20 từ trên stack, bình luận / sửa bài Threads, GV / TA đăng bài qua cổng, 6 biến thể chèn lệnh trên stack.

## Trạng thái stack
`ep-qc` giữ ở `6f6ca1f6`; ba cổng đang BẬT cho lớp 761987 (từ bước seed demo). Stub QC (`scripts/stub_llm.py`) chạy cổng 8777.
