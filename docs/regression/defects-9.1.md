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


## Lượt chạy lại 2026-11-18 (HEAD 88ab83e4) — qc2
- Đã sửa, xác nhận P: DEF-R9.1-01 (TC-UX04-3, TC-UX08-5: 85 luật :hover, 0 ngoài @media), DEF-R9.1-02 (TC-UX07-8: 0 chuỗi), DEF-R9.1-03 (TC-UX08-50, P-11-40: axe landing 1440 và 375, 3 lượt mỗi cỡ, 0 serious/critical; `qc2-r91/qc-uiux91-r91axe-real.spec.ts`, logs/…r91axe….rerun4.log).
- **Nguyên nhân môi trường đã xác lập (không phải F):** (1) ep-r: 761987 đã Chốt v1, có điểm QUIZ/MANUAL do ca khác ⇒ TC-UX16-24 lệch 7 SV, TC-UX01-10 không thấy «Chốt điểm»; trên stack sạch smk91 (seed lại) TC-UX16-24 = 30/30 P. (2) Lỗi kịch bản QC: TC-UX01-1 teacher /class/settings đọc cả tooltip `role=tooltip` (US-01 #34 chuyển câu vào tooltip) — đã sửa spec (`addStyleTag display:none` cho tooltip), chưa chạy lại. (3) Đăng nhập `waitForURL` 30 s hết giờ khi máy tải/bộ đếm auth; postgres tự reinit (xem hạ tầng 9.5). (4) UX12-2 (số chấm đỏ): TC-UX12-2/3/5/14/15 của w4b XANH trên cả ep-r và smk91; ca đỏ ở spec w4 do spec đếm mọi hàng khác Có mặt trong DB đã có sẵn bản ghi khác Có mặt (dữ liệu, không phải lỗi sản phẩm: CSS `.radio[data-status=PRESENT]:checked` ink, còn lại đỏ). Chưa chạy lại w4 trên smk91 sạch.
- **Ứng viên lỗi sản phẩm cần xác nhận khi chạy sạch:** TC-UX16-10 (Trung bình-Thấp): lớp không có SV thiếu điểm vẫn hiện cột «Trạng thái» (đỏ ở cả ep-r và smk91 sạch, `inc=0`); US-UX91-16 AC4 đòi ẩn cả cột; có thể là hồi quy do sprint 10 (cột dùng cho «Đã chốt/Khác bản chốt/LIỆT») — chưa có nguồn LT nên chưa kết luận LT.
- TC-UX14-3/5/7/11, UX15-8/9/13: đỏ trên smk91 sạch (spec cần bài chưa chấm có bản nháp / «Test ended» 5 phút) nhưng trùng thời điểm postgres reinit ⇒ N, chạy lại.
