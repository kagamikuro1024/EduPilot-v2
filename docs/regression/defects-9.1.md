# Lỗi hồi quy sprint 9.1 — qc2, be0c0d96

### DEF-R9.1-01 — luật `:hover` ngoài `@media (hover: hover) and (pointer: fine)` (TC-UX04-3, TC-UX08-5)
- Mức: Thấp · Sprint gốc: 9.1 (US-04 AC2) · Hồi quy do sprint sau?: CÓ (Gradebook.module.css:7; Landing.module.css:40,73,97,144,226 — landing viết lại ở 10.7)
- Bước tái hiện: `node docs/sprints/9.1/qc/scripts/hover-check.mjs` → liệt kê tệp:dòng. Kỳ vọng 0 vi phạm · Thực tế: các dòng trên.
- Bằng chứng: evidence-9.1/logs/grep-batch.log · Chạy lại lần 2: ổn định

### DEF-R9.1-02 — chuỗi «Kiểm duyệt Threads» hiển thị (TC-UX07-8)
- Mức: Thấp · Sprint gốc: 9.1 (US-07 AC4) · Hồi quy do sprint sau?: CÓ (sprint 10.5, `frontend/src/features/.../decisionApi.ts:17`)
- Bước tái hiện: `rg -n -w "Threads" frontend/src --glob '!**/*.test.*'` rồi loại tên biến/đường dẫn → còn chuỗi hiển thị «Kiểm duyệt Threads». Kỳ vọng 0 · Thực tế 1.
- Chạy lại lần 2: ổn định

### DEF-R9.1-03 — axe color-contrast trên `/` ngay sau tải (TC-UX08-50)
- Mức: Thấp · Hồi quy do sprint sau?: CÓ (landing 10.7, Landing.module.css) — nghi artefact hiện-theo-cuộn (opacity đang chuyển tiếp)
- Môi trường: ep-r, 375 px (ổn định 2/2, 12 nút) và 1440 px (1/2 lượt, 2 nút), axe +700 ms sau load.
- Phần tử/màu (375): `.step:nth-child(2) > .card > h3` #aeadad/#b9b8b8 trên #fff = 2,23/1,97; `.step:nth-child(2) .spec[data-ui] p > span|b` #adaaaa/#b8b5b5 trên #fdfbfa = 2,23/1,97; `b > s` #cbc5c4 trên #fdfbfa = 1,65; `.tag--done` #aeccb1 trên #fff = 1,73; `.stat:nth-child(3)` (.stat__num/h3/p) #f9f9f9 trên #fff = 1,05. (1440): `.step:nth-child(2) .card h3`, `> p` #e0e0e0 trên #fff = 1,32.
- Kỳ vọng AC16: 0 serious/critical · Thực tế: như trên; cuộn hết trang rồi chạy axe = 0 vi phạm. Đề xuất: (a) QC cuộn trước khi axe, hoặc (b) giữ opacity 1 (chỉ dịch transform).
- Bằng chứng: frontend/e2e/qc2-r91/qc-uiux91-r91axe-real.spec.ts

### Ghi nhận chưa thành lỗi
Phần tử LCP của landing ở 375 px là `p.lead` (không phải h1). Nhóm F nghi môi trường (không phải lỗi mã): G3 54 ca, G4 18 ca, G5 4 ca — xem triage.md.
