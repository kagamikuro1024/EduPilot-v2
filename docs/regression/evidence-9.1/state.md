# Trạng thái dở — smoke 9.1 (qc2, dừng theo lệnh PM)

Mã: be0c0d96 · stack ep-r · ngày 2026-11-15. **Chưa viết `smoke-9.1.md` / `defects-9.1.md`** (sẽ viết sau khi chạy nốt). Đã dừng mọi tiến trình nền (runner, bộ chạy đầy đủ, FE 3418, vòng xoá bộ đếm auth).

## Số hiện tại (`python3 build_smoke91.py`, tổng 414)
P 184 · F 82 · LT 51 · K 0 · M 97. Trong F có nhiều ca là lượt đỏ cũ chưa chạy lại (xem «Việc tiếp theo»).

## Đã xong
- Triage 118 F → `triage.md` (G1 landing LT 36, G2 rỗng LT 5, G3 dữ liệu ep-r 54, G4 phiên/timeout 18, G5 trình duyệt thiếu 4, G6 axe 1). PM đã chấp nhận G1/G2 là LT. G6 UX08-50 đã gửi dev2 (axe color-contrast chỉ ở trạng thái chưa cuộn tới; cuộn hết trang = 0 lỗi).
- `overrides91.json` (93 mục; sinh bằng `make_overrides.py`): GREP/FFP/UNIT/LH/TAY/landing đã chốt, kèm bằng chứng. Ca F đã ghi: UX04-3 (hover ngoài @media: Gradebook.module.css:7 + Landing.module.css:35,39,40,73,97,144,226), UX07-8 («Kiểm duyệt Threads» decisionApi.ts:17, sprint 10.5), UX08-5 (cùng UX04-3). Nghi LCP ở 375 là `p.lead` không phải h1 (Lighthouse + PerformanceObserver) → định ghi DEF-R9.1-05.
- Bằng chứng: `logs/` (log Playwright gốc + `mock-w1-w3-w4.log` 50/51 xanh trên bản dựng cổng, `devspecs.log` datetime/landing/shell/today xanh, `r91land.log`, `lh-*.json`, `antipatterns.log`, `grep-batch2.log`), `frames/` (ảnh ghép video + OCR). Log lượt chạy lại sai môi trường đã chuyển sang `invalid/`.
- `build_smoke91.py` đã sửa: đọc `*.rerun3.log`, hỗ trợ ký hiệu `1..4,12`, ưu tiên rerun3 > rerun > lượt 1.
- Spec mới `frontend/e2e/qc2-r91/`: `r91lib.ts`, `qc-uiux91-r91axe/land/shots/sv/probe-real.spec.ts`, `qc2-r91-mock-extra.spec.ts` (UX04-1/2 xanh), và bản sao đã sửa `me/courses?limit=100` của mọi `qc-uiux91-r<w>-real.spec.ts`.

## Phát hiện môi trường (cần biết khi chạy tiếp)
- Giảng viên `teacher` có > 20 lớp: `GET /me/courses` mặc định chỉ trả 20 → spec cũ không tìm thấy 761987 (đã sửa ở bản sao `r<w>`).
- Bộ đếm `ep:rl:auth*` làm đăng nhập `waitForURL` hết hạn: chạy vòng xoá mỗi 3 s (`/tmp/qc2-s91/rlclear.sh`, dừng bằng `touch /tmp/qc2-s91/stop-rl`).
- ep-r: postgres từng vào recovery (~23:2x), gateway bị agent khác restart; sổ điểm 761987 đã Chốt; 761987 có 125 bản nháp chấm, 33 SV, grade_entries có 6 QUIZ + 1 MANUAL v9 → đúng 7 SV lệch CSV ở UX16-24 (môi trường, chưa chạy lại được). Đã tạo user `teacher.r91@edupilot.local` (không gán lớp được vì mỗi lớp một giảng viên).
- Chat gọi 9router thật (route QC 9router, rpm dùng chung).

## Việc tiếp theo (thứ tự)
1. Chạy lại bản `r<w>` bằng `/tmp/qc2-s91/runner.sh w2 w12b w12c w4d w4b w4c w4g w4 w5r w5a w5b w5d w5e w5f w5i w5x` (log → `logs/qc-uiux91-r<w>-real.rerun3.log`; bật vòng xoá bộ đếm trước). Mới xong dở w2: UX07-15 đỏ = LT (nav 9/13/16/7 theo `shell.spec.ts:585`, sprint 10), UX01-1 class/settings và UX01-10 cần xem lại. Các spec w4/w5/w12 chưa chạy lượt 3.
2. Chạy `qc-uiux91-r91shots-real` (ảnh 1440/375 cho 16 ca MAC), rồi viết thêm assertion cho M còn lại: UX13-1,5-13,15,17 (spec `r91sv` có sẵn: UX20-1..4 đã xanh; UX20-5 sửa kỳ vọng: `/me` không có nút «Yêu cầu xem lại», QC 9.1 đã ghi «không đổi»; UX20-9 chưa chạy), UX17 (1-7,10-13,PQ), UX18 (1,2,3,5,7,9,ID), UX19 (1-5,8,9,13,PG), UX01-18/PQ, UX03-11/16, UX04-4/19/PQ, UX05-7, UX06-10, UX07-PQ, UX12-16/17/21, UX14-8, UX15-5/19/20/PQ, UX16-13/14/17/22, UX21-PQ, UX02-6, UX91-X2.
3. Ca chạy bằng dev spec (kết quả bộ đầy đủ `/tmp/qc2-s91/full.out`, chưa có tổng): UX03-5, UX04-22, UX06-10, UX10-26, UX12-21, UX13-27, UX15-20, UX16-22, UX20-10, UX01-7/8 (lcp.spec).
4. UX09-16 (`scripts/landing-video/run.sh` bằng docker, ảnh `mcr.microsoft.com/playwright:v1.63.0-noble` đã có), UX16-17 cần lớp thử (761987 đã chốt), UX16-24 cần dọn/nhân bản dữ liệu 7 SV.
5. Sau khi chạy: `python3 make_overrides.py && python3 build_smoke91.py`, viết `smoke-9.1.md` (định dạng smoke-9.5.md) + `defects-9.1.md` (DEF-R9.1-01.. gộp theo nguyên nhân: 02 hover ngoài @media, 03 «Kiểm duyệt Threads», 05 LCP p.lead ở 375, 04 axe chưa cuộn), kiểm tổng = 414.
