### DEF-R10.5-01 — Bảng «Quyết định nhanh» ở /observability nằm trên bảng «7 ngày qua»
- Mức: Thấp
- Sprint gốc: 10.5 · TC gốc: TC-D07-02 (US-DM-07 AC1: «một bảng … dưới bảng 7 ngày») · Hồi quy do sprint sau?: CÓ — sprint 11 viết lại màn «Quan sát AI» (`AdminView.tsx`, các commit 7993ff72 merge wip/11-wave1, 78b0a2be «Section/Page/EmptyState»); ở 10.5 (`ObservabilityScreen.tsx:159`) bảng nằm cuối trang
- Môi trường: ep-r, be0c0d96, ADMIN, 1440 và 375
- Bước tái hiện: 1. Đăng nhập `admin@edupilot.local`. 2. Mở `/observability`. 3. Thứ tự các khối: «Thư chưa gửi được» → «Quyết định nhanh» → «7 ngày qua» → «Chi phí theo tác vụ» → «Yêu cầu». (`frontend/e2e/qc-dm-ui-real.spec.ts`, ảnh `evidence-10.5/shots/observability-admin-1440.png`)
- Kỳ vọng (theo AC): bảng «Quyết định nhanh» dưới bảng 7 ngày  · Thực tế: trên bảng 7 ngày (`frontend/src/features/observability/AdminView.tsx:58` trước `:65`)
- Bằng chứng: `evidence-10.5/pw-dm.log`, `evidence-10.5/shots/observability-admin-1440.png`  · Nghi ngờ mã: `AdminView.tsx:58-65`
- Chạy lại lần 2: ổn định

### Mục TC-D04 (cổng chèn lệnh) — không có lỗi sản phẩm
- TC-D04-01, D04-09 ghi `LỖI THỜI (P-11-26)`: `FAKE_DECISION_PROB` chỉ ghi đè đầu vào NGOÀI `decision-replay.json`; sau P-11-26 replay chứa mẫu chấm của seed (20 FALLBACK + 1 FAST ở P=0,9) nên kịch bản `dm_c.py` (14 cờ mới ở P=0,9) không còn dựng được. Cần dev/PM cho cách điều khiển mới nếu muốn kiểm OR ba tầng.
- TC-D04-02/03 PASS: lỗi decision (circuit_open) → chấm xong 22/22, điểm y hệt giữa cổng tắt / bật / lỗi cùng điều kiện. Chênh điểm so với round 1 trong `dm_c.py` KHÔNG do cổng (cổng tắt cũng chênh y hệt): round 1 chấm lúc route GRADING do nhóm khác đang đổi.
- Lưu ý môi trường: Postgres vào recovery giữa lượt B5a (nhóm khác chaos) làm hỏng một lượt; đã chạy lại ở B6.
