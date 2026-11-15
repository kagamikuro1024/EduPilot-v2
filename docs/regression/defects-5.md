### DEF-R5-01 — Nút đầu trang AppShell: axe `label-content-name-mismatch` (chỉ khi bật thẻ wcag21a)
- Mức: Thấp (ghi nhận, KHÔNG phải hồi quy của sprint sau; ca TC liên quan được chấm lại)
- Sprint gốc: 5 · TC gốc: TC-PE03-52 (axe wcag2a/2aa/21aa/22aa 0 serious) · Hồi quy do sprint sau?: KHÔNG — `aria-label` của nút tìm nhanh / chuông / hồ sơ có từ `f7a15c7d` (US-PROTO-00, 2026-10-02) và `c8c3b70b` (2026-10-02), TRƯỚC sprint 5 (10-03…10-10); `axe-allow.json` không có mục `label-content-name-mismatch`. Quy tắc này thuộc thẻ `wcag21a`, không nằm trong bộ thẻ của TC/dev (`a11y.spec.ts` TAGS = wcag2a, wcag2aa, wcag21aa, wcag22aa) nên lượt đầu của QC (thêm wcag21a) báo thừa.
- Môi trường: ep-r, be0c0d96, GV r5t 1440; script `evidence-5/ui/ui_pe.cjs`
- Kỳ vọng: 0 serious với bộ thẻ TC. · Thực tế: với bộ thẻ TC → 0 (`/questions`); thêm wcag21a → `button.search` aria-label "Tìm nhanh hoặc đi đến" ≠ chữ "…" nhìn thấy, chuông "Thông báo, N chưa đọc" ≠ "9+", hồ sơ "Tài khoản: Tên" ≠ "Tên" (WCAG 2.5.3). Nghi mã: `frontend/src/shared/shell/AppShell.tsx:245-266`.
- Chạy lại lần 2: ổn định. Khuyến nghị (PM quyết): đưa chữ nhìn thấy vào đầu aria-label.

### DEF-R5-02 — Vùng cuộn ở màn làm bài code không focus được bằng bàn phím
- Mức: Thấp
- Sprint gốc: 5 · TC gốc: TC-PE06-08 (axe 1024/1440 0 `serious`) · Hồi quy do sprint sau?: KHÔNG RÕ (gốc PE-06 chỉ chạy axe qua spec dev có `axe-allow`)
- Môi trường: ep-r, be0c0d96, SV r5s12, 1440, màn `/exams/<id>/take` câu code sau khi `Chạy thử` có kết quả
- Bước tái hiện: 1. SV mở bài MIXED, chuyển tới câu code, bấm `Chạy thử`. 2. axe → `scrollable-region-focusable` ×1 (vùng cuộn của khối kết quả / soạn mã không có `tabindex=0`).
- Kỳ vọng: 0 serious · Thực tế: 1 vùng cuộn không tới được bằng Tab. · Bằng chứng: `evidence-5/logs` (ui_pe, lượt SV 1440) · Nghi ngờ mã: khối kết quả chạy thử / editor ở `frontend/src/features/exam` (chưa định vị dòng)
- Chạy lại lần 2: ổn định

## Ghi nhận không tính lỗi
- Go `TestRunningNotCancelled` đỏ 1 lần khi máy tải nặng (lease 30 s hết hạn), PASS 2/2 khi chạy riêng — chập chờn do tải.
- TC-PE02-37: fork bomb A8 ra `WA` (FORKED=0) thay vì `TLE` như gốc; bất biến `procPeak==1`/0 tiến trình mồ côi thoả.
- Today bỏ provider `staff`/`assignment` khi stack quá tải (timeout) — degrade có chủ đích.
