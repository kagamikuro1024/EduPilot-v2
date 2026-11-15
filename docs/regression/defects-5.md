### DEF-R5-01 — Nút đầu trang AppShell vi phạm axe `label-content-name-mismatch` (serious) trên mọi màn đăng nhập
- Mức: Trung bình (truy cập; ảnh hưởng người dùng đọc màn hình/điều khiển bằng giọng nói; rộng)
- Sprint gốc: 5 · TC gốc: TC-PE03-52, TC-PE04-45, TC-PE05-48, TC-PE06-08 (axe 0 `serious`/`critical`) · Hồi quy do sprint sau?: CÓ (khung AppShell mới sau sprint 5: 10.7 §23 / sprint 9.1–10.x; `git log frontend/src/shared/shell/AppShell.tsx`)
- Môi trường: ep-r, be0c0d96, vai GV r5t / TA / SV r5s11–12, Chrome 1440 và 390; frontend http://localhost:3700 qua https://localhost:1143
- Bước tái hiện: 1. Đăng nhập (UI) bằng bất kỳ tài khoản. 2. Mở `/questions` (GV) hoặc `/exams`, `/exams/<id>/take` (SV). 3. Chạy `@axe-core/playwright` (wcag2a/aa/21a/21aa) — script: `docs/regression/evidence-5/ui/ui_pe.cjs` (in node vi phạm).
- Kỳ vọng (theo AC/TC): axe 0 `serious`/`critical`. · Thực tế: `label-content-name-mismatch` ×2–3: (1) `button.search aria-label="Tìm nhanh hoặc đi đến"` khác chữ nhìn thấy "Tìm nhanh hoặc đi đến…"; (2) nút chuông `aria-label="Thông báo, 58 chưa đọc"` khác chữ hiển thị "9+"; (3) nút hồ sơ `aria-label="Tài khoản: R5 SV 12"` khác chữ "R5 SV 12".
- Bằng chứng: `evidence-5/logs/ui_pe.log`, `evidence-5/ui/shots/*.png` · Nghi ngờ mã: `frontend/src/shared/shell/AppShell.tsx:245-266` (aria-label không chứa nguyên văn chữ hiển thị — WCAG 2.5.3)
- Chạy lại lần 2: trượt/ổn định (3 lượt chạy)

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
