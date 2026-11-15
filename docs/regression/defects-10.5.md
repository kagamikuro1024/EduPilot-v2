### DEF-R10.5-01 — Bảng «Quyết định nhanh» ở /observability nằm trên bảng «7 ngày qua»
- Mức: Thấp
- Sprint gốc: 10.5 · TC gốc: TC-D07-02 (US-DM-07 AC1: «một bảng … dưới bảng 7 ngày») · Hồi quy do sprint sau?: CÓ — sprint 11 viết lại màn «Quan sát AI» (`AdminView.tsx`, các commit 7993ff72 merge wip/11-wave1, 78b0a2be «Section/Page/EmptyState»); ở 10.5 (`ObservabilityScreen.tsx:159`) bảng nằm cuối trang
- Môi trường: ep-r, be0c0d96, ADMIN, 1440 và 375
- Bước tái hiện: 1. Đăng nhập `admin@edupilot.local`. 2. Mở `/observability`. 3. Thứ tự các khối: «Thư chưa gửi được» → «Quyết định nhanh» → «7 ngày qua» → «Chi phí theo tác vụ» → «Yêu cầu». (`frontend/e2e/qc-dm-ui-real.spec.ts`, ảnh `evidence-10.5/shots/observability-admin-1440.png`)
- Kỳ vọng (theo AC): bảng «Quyết định nhanh» dưới bảng 7 ngày  · Thực tế: trên bảng 7 ngày (`frontend/src/features/observability/AdminView.tsx:58` trước `:65`)
- Bằng chứng: `evidence-10.5/pw-dm.log`, `evidence-10.5/shots/observability-admin-1440.png`  · Nghi ngờ mã: `AdminView.tsx:58-65`
- Chạy lại lần 2: ổn định
