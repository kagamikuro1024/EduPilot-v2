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


## Ca lỗi thời đã XOÁ khỏi smoke (lệnh chủ dự án; mỗi mã một dòng + nguồn)
- TC-UX05-1: P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý»
- TC-UX05-2: P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý»
- TC-UX05-3: P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý»
- TC-UX05-5: P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý»
- TC-UX05-11: P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý»
- TC-UX08-1: AC1: h1 đổi còn 8 từ «Vận hành lớp học có trợ lý AI», sáu mục mới (P-107-9, P-107-11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng land
- TC-UX08-2: AC1: danh sách h2 đổi theo TC-UX08-1 (P-107-9, P-107-11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devsp
- TC-UX08-3: AC2: màn đầu đúng hai nút «Đăng nhập» + «Xem video» (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs
- TC-UX08-4: AC2: như UX08-3 (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-6: AC3 + US-09 AC2: clip/poster màn đầu bị bỏ, thay bằng đoạn sản phẩm HTML (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landin
- TC-UX08-7: AC3 + US-09 AC2: không còn clip để đo (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-8: AC3 + US-09 AC2: không còn clip (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-9: AC3/AC14: không còn clip, reduce theo AC14 mới (P-107-12, P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 
- TC-UX08-10: AC3/AC14: không còn clip (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-11: AC3: không còn clip (tab ẩn) (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-12: AC3 + AC19: không còn clip tải lỗi (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-13: AC3: màn đầu không còn clip/poster; đoạn sản phẩm HTML (P-107-13)
- TC-UX08-14: AC4 mới: nút Tạm dừng/Phát 32 px, vùng chạm ≥ 44 px hoặc vùng đệm (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.
- TC-UX08-15: AC4 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-20: AC5/AC13 sửa 10.7: ngưỡng «chênh ≤ 1,5 KB so với v1 216 KB» bị thay bằng tổng ≤ 256 KB và script riêng ≤ 5 KB (P-107-12/13); số đo hiện hành ở UX08-44
- TC-UX08-22: AC6: khung ghim đổi: ba tên màn ngang + ảnh rộng hết khung (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/2
- TC-UX08-23: AC6 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-24: AC6: ảnh khung ghim rộng 1280 px, không còn giới hạn 92.160 B (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 2
- TC-UX08-25: AC6/AC12: hoạt cảnh khung ghim theo techlead 8.3 (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/de
- TC-UX08-26: AC7 đổi theo khung ghim mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-27: AC7 đổi theo khung ghim mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-29: AC8: ba bước dùng IntersectionObserver thay view-timeline (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21
- TC-UX08-30: AC8 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-31: AC8/AC9: số đếm do script (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-32: AC8/AC9 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-33: AC9: tiêu đề khối «Ba số đo trên dữ liệu mô phỏng», bỏ dòng chú thích (P-107-11; BA 14/11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng
- TC-UX08-34: AC9 mới: bỏ chú thích riêng (P-107-11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-35: AC10: hai cột vai thành hai tab (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-36: AC10 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-38: AC11 sửa: chữ thân ≥ 16 px, dẫn ≥ 18 px (P-107-9). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log
- TC-UX08-39: AC11: bỏ điều kiện «một h2 mỗi khung nhìn» (P-91-8 (b)) và khoảng cách đổi theo sáu mục 100svh (P-107-9). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới 
- TC-UX08-40: AC12: vi mô khi rê đổi (nhích 1 px, mũi tên 3 px; P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/de
- TC-UX08-41: AC12 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-42: AC12 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-46: AC14 mới: reduce → không gắn m-on (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX08-47: AC13/AC15: không còn animation-timeline trong Landing.module.css; thay bằng IntersectionObserver (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản m
- TC-UX08-49: AC15: nay có đúng MỘT IntersectionObserver (LandingMotion.tsx:143) thay P-91-2 «không IntersectionObserver» (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/1
- TC-UX08-53: AC19: lỗi clip không còn; ảnh khung giữ kích thước theo AC19 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.t
- TC-UX08-55: AC21 sửa 10.7: landing.spec.ts nay kiểm đoạn sản phẩm/Tạm dừng/Xem video/ghim/hai tab; bản mới 21/21 xanh (logs/devspecs.log); TC cũ ghi «hai cột vai (không tab)», «clip không tải khi reduce» (P-107-1
- TC-UX08-56: AC21 mới: ảnh mốc landing-1440/390 anonymous reduce (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs
- TC-UX09-4: US-09 AC2 «clip màn đầu — bỏ»: gioi-thieu-clip-v1.* và poster đã xoá khỏi public/landing (P-107-13); ls public/landing không còn tệp clip
- TC-UX09-5: US-09 AC2: không còn clip (P-107-13)
- TC-UX09-6: US-09 AC2: không còn clip để kiểm faststart (P-107-13); video đầy đủ vẫn có moov trước mdat (ftyp,moov,free,mdat), h264
- TC-UX09-17: US-09 AC2/AC5: không còn clip màn đầu; video đầy đủ chỉ mở bằng «Xem video» (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng lan
- TC-UX09-18: US-09 AC2/AC5 (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log)
- TC-UX09-20: US-09 AC2/AC13: không còn clip lặp để xem (P-107-13)
