# QC report v5 — US-PROTO-02 (Giảng viên + TA)  · Kết luận: **FAIL**

Nhánh `sprint/1.5-mock-ui` · HEAD `8bb6861` · ngày 2026-10-01 · server `http://localhost:3400` · Chrome for Testing headless riêng (profile `/tmp/qcv5-*`)
TC: `docs/sprints/1.5/qc/tc-US-PROTO-02.md` — **91 TC**. Đối chiếu spec v5 + v5.1 + #20–#24.

**Tóm tắt:** 78/91 TC PASS; **13 FAIL** — TC-02-04, TC-02-22, TC-02-38, TC-02-52, TC-02-55, TC-02-56, TC-02-57, TC-02-61, TC-02-62, TC-02-68, TC-02-76, TC-02-90, TC-02-91.

## Cổng đã chạy

| Lệnh / script | Kết quả |
| --- | --- |
| `proto-curl.sh` | 497 PASS / 0 FAIL; phần US-02: `tc_02_01`, `_07`, `_14`, `_18`, `_20` PASS |
| `audit.mjs` | 02-AC9, AC10, AC14, AC15, AC18, AC19, AC21: 0 FAIL; `AUDIT-biên` FAIL: `/students` 720, `/attendance` 720 (đã xác nhận tay, BUG-v5-02-9) |
| `threads-timeline.mjs` (T1j–T4g) | PASS |
| `regress-v24.mjs` | TC-02-90 FAIL (1440 / 1280 / 1100); TC-02-91 FAIL (390 / 375) |
| `pnpm -C frontend lint` · `bash scripts/ui-antipatterns.sh` | rc=0 · sạch |
| Chạy tay | Chrome riêng + chuột thật + bàn phím; ảnh `shots/qc-v5/owner/`, `v5q02/`, `v24/` |

## TC

| TC-id | PASS/FAIL | Cách kiểm | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-02-01 | PASS | C | proto-curl.log tc_02_01 (6 dòng) PASS |
| TC-02-02 | PASS | B | inbox,attendance,review,ai-pending,setup,members · 6 việc cần xử lý hôm nay Xếp t |
| TC-02-03 | PASS | B | Bạn được phân công lớp An ninh mạng – 761988 |
| TC-02-04 | FAIL | B | Xử lý Nhận 5 phiếu, lưu điểm danh, duyệt cả 4 bài, xác nhận t-cbc: Hôm nay còn "3 việc" gồm thẻ review "cần xem kỹ" (dù chip lọc 0 hàng), setup, members; không bao giờ ra "Không còn việc cần bạn quyết định" (BUG-v5-02-1); ảnh owner/gv-today-cleared.webp |
| TC-02-05 | PASS | B | sau Nhận: Trạng thái Đã nhận AI đã tra Quy chế môn học (tr · đã nhận lúc 09:20 |
| TC-02-06 | PASS | B | Trạng thái Đã trả lời AI đã tra Quy chế môn học (tr · Đã gửi thư thông báo tới email của sinh viên (mô phỏng) |
| TC-02-07 | PASS | B | Phạm Quốc Bảo đã nhận lúc 09:12 Lưu thành |
| TC-02-08 | PASS | B | 5 phiếu mở (26 phút, 3 giờ, hôm qua 09:20 = 1 ngày, 2 ngày, 3 ngày trước) + tab Đã nhận 1; chi tiết "Đã chờ 1 ngày"; Tất cả 6 |
| TC-02-09 | PASS | B (DEMO) | TC-DEMO-17 PASS (điểm danh bàn phím → "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng") |
| TC-02-10 | PASS | B (DEMO) | TC-DEMO-19 PASS (/me QT 8,5) |
| TC-02-11 | PASS | B | Click hàng để lấy tiêu điểm, ↓ chọn hàng, `4` → "29 có mặt · 1 vắng" + Hoàn tác (5 s) → 30 có mặt; `3` → Vắng phép; `P` ×2 → "2 lần · +0,50" (+0,25/lần). Quan sát: bấm chuột vào hàng không đổi hàng đang chọn của bàn phím |
| TC-02-12 | PASS | B | options=15 buổi11: true buổi9:  10 09:00–11:30 · P.302 – G2 Mặc định mọi sinh viên có mặt — chỉ đánh người vắng |
| TC-02-13 | PASS | B | 24/30 chỗ đã dùng Mời trợ · nút Duyệt=4 · có Linh=true · mã true |
| TC-02-14 | PASS | B | D chooser=761988; Hôm nay: Chào Linh Thứ Năm, 29 tháng 10 · tuần 3 · An ninh mạng – 761988 Việc n; chuông: Bạn đã được duyệt vào lớp An ninh mạng – 761988 Lớp học · vừa  |
| TC-02-15 | PASS | B | Từ chối → Hoàn tác; Tạo lại mã → hộp xác nhận "Mã cũ BX4P9TW vô hiệu ngay…" → mã mới 5LZLV8U, rồi XKY4BTW; còn Sao chép link/mã |
| TC-02-16 | PASS | B | {"l":[380,646],"d":[772,560]} |
| TC-02-17 | PASS | B | chỉ danh sách=true; mở → /inbox?ticket=tk-5; nút ← Hộp thư [89,44]; tràn ngang=0 |
| TC-02-18 | PASS | B | 375/390: hàng 2 dòng có kẻ trên mỗi hàng, không nền/viền/bóng, 4 nhãn 81–85×44, Phát biểu 91×44, tràn ngang 0; ảnh owner/att-375.webp |
| TC-02-19 | PASS | B | Đang chờ mạng · 2 thay đổi Điểm d → Đã lưu 09:20; marks=[[2,"0001"],[3,"0100"]]; còn chờ mạng=false |
| TC-02-20 | PASS | B | Sinh viên 30 sinh viên · An ninh mạng – 761987 Chuyên cần, điểm và nhãn rủi ro chỉ giảng viên/TA thấy Không tải được dan |
| TC-02-21 | PASS | C | proto-curl.log tc_02_07 (12 dòng) PASS |
| TC-02-22 | FAIL | B | KHÔNG KIỂM ĐƯỢC: TA chỉ có lớp 761987 trong bộ chọn lớp (cookie/`?course=int1006-2` bị bỏ qua); không có đường vào /class/members của 761988 để thử `Duyệt`. Ở 761987 TA thấy danh sách + Sao chép link/mã, không có `Tạo lại mã`/`Mời ra khỏi lớp`/`Mời trợ giảng`. Chuông TA lại có "Bạn được phân công lớp 761988" |
| TC-02-23 | PASS | B | 30=30 Huy=2 MSSV=1 zzzz="iểm và nhãn rủi ro chỉ giảng viên/TA thấy Tìm theo tên hoặc mã số sinh viên Cần " chips=Cần chú ý 8\|Vắng nhiều 5\|Điểm giảm 2\|Ít hoạt động 4 |
| TC-02-24 | PASS | B | Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu Bài tập 02 |
| TC-02-25 | PASS | B | tab Ghi chú; save=true; thấy=true; hoàn tác=true; Sinh viên Lê Quang Huy 20229003 · An ninh mạng – 761987 Thêm ghi chú Nhắn riêng Vắng 5/9 buổi, đã bị |
| TC-02-26 | PASS | B | lộ D1/D2 ở []; /inbox có ticket D3: Trần Thu Uyên vừa xong Thi cuối kỳ có được mang một tờ A4 gh |
| TC-02-27 | PASS | B | [["/inbox",true,true],["/students",true,true],["/attendance",true,true],["/class/members",true,true],["/gradebook",true,true],["/",true,true]] |
| TC-02-28 | PASS | B | ["Chờ xác nhận"] → ["Đã được giảng viên xác nhận"] |
| TC-02-29 | PASS | B | textarea inline độ dài=261,0 |
| TC-02-30 | PASS | B | ["Đã được giảng viên sửa & xác nhận"] nội dung mới=true |
| TC-02-31 | PASS | B | menu=Loại khỏi tri thức rej=true hoàn tác=true |
| TC-02-32 | PASS | A | audit 02-AC9 chip (1 dòng) → 0 FAIL |
| TC-02-33 | PASS | A | audit 02-AC9 cột danh sách (1 dòng) → 0 FAIL |
| TC-02-34 | PASS | A | audit 02-AC9 mobile (2 dòng) → 0 FAIL |
| TC-02-35 | PASS | A | audit 02-AC9 (Back\|"← Hộp thư") (4 dòng) → 0 FAIL |
| TC-02-36 | PASS | A | audit 02-AC9 biên (2 dòng) → 0 FAIL |
| TC-02-37 | PASS | B | ?state=error 375: Thử lại=true tràn=0; ticket lạ → danh sách + chú giải (TC-02-37a PASS) |
| TC-02-38 | FAIL | B | Hàng chip/tab ở 1440 co thành vạch 3 px (TC-02-90, bug #24a): không đọc được chip Đang chờ/Đã nhận/Đã trả lời/Tất cả → chưa thể kiểm "đủ chữ, tự cuộn vào khung" |
| TC-02-39 | PASS | A | audit 02-AC10 ticket (5 dòng) → 0 FAIL |
| TC-02-40 | PASS | B | Trung {"h":560,"slack":25} Thảo {"h":585,"slack":25} |
| TC-02-41 | PASS | B | gap sau Nhận=24px; Answered: Đã gửi thư thông báo tới email của sinh viên (mô phỏng) |
| TC-02-42 | PASS | B | gap=24 |
| TC-02-43 | PASS | T | timeline.json T1j+T1k+T1l PASS |
| TC-02-44 | PASS | T | timeline.json T1m+T1n PASS |
| TC-02-45 | PASS | T | timeline.json T4e+T4f PASS |
| TC-02-46 | PASS | T | timeline.json T4g PASS |
| TC-02-47 | PASS | B | Xác nhận / Chỉnh sửa→Lưu và xác nhận / Loại khỏi tri thức đều làm thẻ rời Hôm nay, đếm giảm 1 (7→6) |
| TC-02-48 | PASS | B | 8→7→6→5 chuỗi đủ (Xác nhận thread mới → 7; Chỉnh sửa+Lưu → 6; Xác nhận t-cbc → 5, ai-pending rời độc lập); Đặt lại dữ liệu demo (menu hồ sơ, không hộp thoại) → GV 6 việc, B thấy 12 thread |
| TC-02-49 | PASS | T | timeline.json T1o PASS |
| TC-02-50 | PASS | B | → /class/members?tab=pending · chooser=761988 · An ninh mạng · Hôm nay Thành viên lớp An ninh mạng – 761988 · 24/30 chỗ đã dùng Mời t |
| TC-02-51 | PASS | B | thẻ: Thiết lập lớp mới — 761988 Còn 4/4 bước trước buổi đầu tiên Chia sẻ mã tham gia BX4P9TW Tải quy chế để lập công thức điể · nút ["Chia sẻ mã tham gia BX4P9TW→/class/members?course=int1006-2","Tải quy chế để lập công thức điểm→/gradebook/scheme?course=int1006-2","Tạo lịch buổi học→/calendar?course=int1006-2","Tải tài liệu bài |
| TC-02-52 | FAIL | B | 761988 chooser=761988; từ lớp 2 bấm Điểm danh (lớp 1) → /attendance?session=10 chooser=761988 |
| TC-02-53 | PASS | B | B=761987/761987 lỗi=false; GV=761988; A=761988; D(sv-4)= |
| TC-02-54 | PASS | B | {"review":"/grading?filter=review \| Chấm bài An ninh mạng – 761987 · AI chấm nháp, giảng viên duyệt và côn","attendance":"/attendance?session=10 \| Điểm danh An ninh mạng – 761987 · buổi 10 · Thứ Năm, 29 tháng 10 09:00","ai-pending":"/threads/t-cbc \| Threads CBC khác ECB ở điểm nào? Đặng Gia An Mật mã đối xứng · Tuần 10"} |
| TC-02-55 | FAIL | B | GV ở 761987 mở chuông: KHÔNG có mục "Phạm Ngọc Linh xin vào lớp 761988" (state có note courseId int1006-2); chỉ hiện khi chọn lớp 761988; TA ở 761987 cũng không thấy |
| TC-02-56 | FAIL | B | GV: Duyệt D → chuông D "Bạn đã được duyệt…" + Hoàn tác (PASS); Từ chối → chuông D (01-107 PASS); phần "TA Duyệt được + thông báo giống" KHÔNG KIỂM ĐƯỢC (TA không vào được lớp 761988, xem TC-02-22) |
| TC-02-57 | FAIL | B | Thẻ Trả lời → /inbox?ticket=tk-5, chi tiết đúng "Nhóm em muốn phân tích…", nhưng hàng Thảo (788–912 px) bị khung danh sách cắt đáy (876) và vượt khung nhìn 900 → hàng không nằm trọn trong khung nhìn; ảnh owner/inbox-tk5.webp |
| TC-02-58 | PASS | B | trực tiếp tk-5: {"back":44,"det":true}; ← về /inbox |
| TC-02-59 | PASS | B | Nhận phiếu Thảo → thẻ Hôm nay chuyển sang Đỗ Thanh Long 2 ngày, href /inbox?ticket=tk-4; ?ticket=tk-5 chọn đúng; (tk-d3: xem TC-02-88) |
| TC-02-60 | PASS | A | audit 02-AC14 (8 dòng) → 0 FAIL |
| TC-02-61 | FAIL | B | Nhận một phiếu → nav-badge-inbox 5→4; nav-badge-grading 4 ‖ Nhận phiếu: Hộp thư 5→4 và Chấm bài 4 (PASS); Duyệt bài B: Chấm bài 4→3, Hôm nay "3 bài" (PASS); nhưng duyệt tiếp 3 bài còn lại không làm số giảm (kẹt 3) → xem TC-02-62 |
| TC-02-62 | FAIL | B | Nhận hết 5 phiếu: badge Hộp thư 5→4→3→2→1→ẩn (không "0") PASS; Duyệt hết 4 bài trong chip "Cần xem kỹ": hàng 4→0 nhưng badge Chấm bài kẹt ở 3 và thẻ Hôm nay vẫn "3 bài Bài tập 03 cần xem kỹ" → không bao giờ về 0 (BUG-v5-02-1) |
| TC-02-63 | PASS | B | GV inbox:6,grading:4 · TA inbox:6,grading:4 · sau Nhận D3: inbox:5,grading:4 |
| TC-02-64 | PASS | A | audit 02-AC15 ô Buổi (2 dòng) → 0 FAIL |
| TC-02-65 | PASS | B | 375: tên 236px, Phát biểu 91,44 · 390: tên 251px, Phát biểu 91,44 |
| TC-02-66 | PASS | B | 375 sau 2 vắng+1 muộn: "27 có mặt · 1 muộn · 2 vắng" mỗi cụm một dòng, không ngắt giữa cụm; ảnh owner/att-375-summary.webp. Quan sát: ô Buổi rộng hết cỡ khi dòng tóm tắt xuống hàng riêng (bố cục nhảy khi đánh dấu) |
| TC-02-67 | PASS | B (DEMO) | TC-DEMO-17 + 18 PASS ở 390 px (cảm ứng): 4 thao tác 1,3 s ≪ 60 s; "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; /me B 8,5 (TC-DEMO-19). Chưa chạy riêng ở 375 |
| TC-02-68 | FAIL | B | 375: "Buổi 9 · 22/10" vừa (106≤113 px) nhưng "Buổi 11 · 05/11 · chưa diễn ra" 211 px > chỗ 166 → hiện "chưa diễn" bị cắt; "Buổi này chưa diễn ra" không tràn; ảnh owner/att-375-Buổi11.webp (cùng gốc bug #24d) |
| TC-02-69 | PASS | B | [["đầu",false,"Chưa có thay đổi"],["bật",false,"Đang giả lập mất mạng"],["1 vắng khi mất mạng",false,"Đang chờ mạng · 1 thay đổi Điểm d"],["tắt",false,"Đã lưu 09:20"]] |
| TC-02-70 | PASS | B | Đã hoàn tất buổi 10 · 27 có mặt, 3 vắng · nút sau lưu={"dis":true,"t":"Đã lưu"} · đổi thêm ô → nút=["Lưu điểm danh\|false"] · dòng="không còn" |
| TC-02-71 | PASS | B | 3 vắng 0 muộn → "Đã hoàn tất buổi 10 · 27 có mặt, 3 vắng" (27+3=30, bỏ cụm 0 muộn); 0 vắng → "· 30 có mặt"; Hoàn tác khôi phục 30, không để "Đang chờ mạng" |
| TC-02-72 | PASS | B | rỗng disabled=true trắng=true msg=true |
| TC-02-73 | PASS | B | nguyên văn disabled=true; +1 chữ disabled=false; về bản gốc disabled=true ‖ sau lưu: sửa&xác nhận=true; xem bản gốc=true |
| TC-02-74 | PASS | A | audit 02-AC18 chip (1 dòng) → 0 FAIL |
| TC-02-75 | PASS | A | audit 02-AC18 Hôm nay (1 dòng) → 0 FAIL |
| TC-02-76 | FAIL | B | Hôm nay "Lớp cần chú ý · 8 sinh viên" và /students/sv-3 cùng câu "Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu Bài tập 02"; nhưng /students (cột Rủi ro) chỉ ghi nhãn "Cần chú ý", không có câu lý do → không so được; ảnh owner/students-1440.webp |
| TC-02-77 | PASS | C | proto-curl.log tc_02_18 (1 dòng) PASS |
| TC-02-78 | PASS | A | audit 02-AC19 (1 dòng) → 0 FAIL |
| TC-02-79 | PASS | B | /grading mặc định bật chip "Cần xem kỹ" (B, Vũ Khánh Huy, Phạm Minh Dũng, Trịnh Tuấn Mai); bỏ chip / chuyển "Chưa duyệt" → thứ tự tương đối giữ nguyên (B trước, rồi Trung, Thảo…); khớp audit 02-AC19 (data-student-id đồng nhất) |
| TC-02-80 | PASS | C | proto-curl.log tc_02_20 (4 dòng) PASS |
| TC-02-81 | PASS | B | /grading?filter=review: B "Hai lượt chấm lệch 1,5 điểm"; Vũ Khánh Huy "Bài ngắn bất thường…"; Phạm Minh Dũng "Trùng đoạn với bài của một sinh viên khác"; Trịnh Tuấn Mai "AI không đủ chắc chắn ở tiêu chí…" |
| TC-02-82 | PASS | B | Duyệt bài B: Hôm nay "3 bài Bài tập 03 cần xem kỹ", dòng phụ không còn "lệch hai lượt chấm" (còn: ngắn bất thường · trùng đoạn · AI không chắc); chip "Cần xem kỹ 3" = badge Chấm bài 3 |
| TC-02-83 | PASS | B | thứ tự: inbox > attendance > review > thread-new-2 > thread-new-1 > ai-pending > setup > members |
| TC-02-84 | PASS | B | 8 việc cần xử lý hôm nay · inbox,attendance,review,thread-new-2,thread-new-1,ai-pending,setup,members |
| TC-02-85 | PASS | T | timeline.json T4e+T4f PASS |
| TC-02-86 | PASS | B | ai-pending: 1 câu trả lời của AI chờ bạn xác nhận Sinh viên chỉ thấy nhãn chờ xác nhận cho tới khi bạn duyệt câu |
| TC-02-87 | PASS | B | chip=Chờ xác nhận 2; lọc → 2 hàng: Dùng lại IV trong CTR có sao không? Em thấy C \| CBC khác ECB ở điểm nào? Em đọc slide chương  |
| TC-02-88 | PASS | B | chuông GV có "1 câu hỏi mới cần xử lý"=true → /inbox?ticket=tk-d3 · chi tiết: Trần Thu Uyên  20229002 · An ninh mạng – 761987  Thi cuối kỳ |
| TC-02-89 | PASS | B | 3 nhãn "Quá 24 giờ" → sau Nhận phiếu Huy còn 2; Đang chờ 5→4, badge 5→4, tab Đã nhận 2 |
| TC-02-90 | FAIL | A + B | `regress-v24.mjs`: FAIL 1440 / 1280 / 1100 — 4 tab (Đang chờ, Đã nhận, Đã trả lời, Tất cả) cao 30 px nhưng phần nhìn thấy **3 px** (co thành vạch mỏng); đạt 1099 / 900 / 720 / 390 / 375 (visH 30–44). Ảnh `owner/inbox-tk5.webp` (vạch đen trên đầu cột danh sách), `shots/qc-v5/v24/inbox-tabs-*` |
| TC-02-91 | FAIL | A + B | `regress-v24.mjs`: FAIL 390 và 375 — ô Buổi hiện "Buổi 10 · 29/10 · đang diễn ra" 222 px > chỗ 183 / 168 px, bị cắt ("đang diễ…"); trạng thái nằm trong ô, không có dòng phụ riêng; tay: buổi 11 "chưa diễn" cắt ở 375. Khi dòng tóm tắt xuống hàng riêng (sau khi đánh dấu) ô rộng đủ chữ → bố cục nhảy. Ảnh `owner/att-375.webp`, `owner/att-375-Buổi11.webp`, `owner/att-375-summary.webp` |

## Lỗi

| Mã | Route | Vai | Bước | Thấy | Mong đợi | Mức | AC | Ảnh |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BUG-v5-02-1 | `/grading`, thanh bên, `/` | GV | `/grading?filter=review` → `Duyệt bài` lần lượt 4 bài | Chip "Cần xem kỹ" 3→2→1→0 nhưng badge Chấm bài **kẹt 3**, thẻ Hôm nay vẫn "3 bài Bài tập 03 cần xem kỹ"; Hôm nay không bao giờ về "Không còn việc cần bạn quyết định" | Badge và thẻ đếm bài **chưa duyệt** thuộc Cần xem kỹ, ẩn khi 0 (N8) | **cao** | 02-AC14, AC20 · TC-02-04, 61, 62 | `owner/gv-today-after4.webp`, `owner/gv-today-cleared.webp` |
| BUG-v5-02-2 | `/` → `/attendance` | GV | Chọn lớp 761988 → bấm thẻ "Điểm danh buổi 10" của 761987 | Mở `/attendance` của **761988** ("Lớp 761988 chưa có lịch buổi học"), bộ chọn lớp vẫn 761988; link thẻ thiếu `course=int1006-1` (cùng gốc BUG-v5-DEMO-3 với `/inbox?ticket`) | Chuyển lại 761987 đúng (TC-02-52) | **cao** | AC12 · TC-02-52 | `owner/att-from-761988.webp`, `owner/gv-today-761988.webp` |
| BUG-v5-02-3 | `/inbox` ≥ 1100 | GV | Mở `/inbox` ở 1440 | Hàng tab Đang chờ / Đã nhận / Đã trả lời / Tất cả cao 30 px nhưng nhìn thấy **3 px** (co thành vạch) — không đổi lọc được | Đủ chữ, ≥ 28 px (#24a) | **cao** | #24a · TC-02-90, 38 | `owner/inbox-tk5.webp`, `v24/inbox-tabs-*` |
| BUG-v5-02-4 | `/attendance` 390 / 375 | GV | Mở buổi 10 / chọn buổi 11 | Ô Buổi cắt "Buổi 10 · 29/10 · đang diễ…" và "Buổi 11 · 05/11 · chưa diễn" | Nhãn ngắn không cắt, trạng thái ở dòng phụ (#24d) | trung bình | #24d · TC-02-91, 68 | `owner/att-375.webp`, `owner/att-375-Buổi11.webp` |
| BUG-v5-02-5 | chuông | GV, TA đang ở 761987 | D gửi yêu cầu vào 761988 → mở chuông | Không có mục "Phạm Ngọc Linh xin vào lớp 761988" (state có note `courseId=int1006-2`); chỉ hiện khi chọn 761988 | Thấy mục, bấm → `/class/members?course=int1006-2&tab=pending` (TC-02-55) | trung bình | AC12 · TC-02-55 | — |
| BUG-v5-02-6 | bộ chọn lớp | TA | Mở bộ chọn lớp / `?course=int1006-2` | TA chỉ có 761987; không tới được `/class/members` của 761988 để thử `Duyệt`; chuông TA lại ghi "Bạn được phân công lớp 761988" | TA duyệt được (TC-02-22 / 56) hoặc sửa TC + bỏ thông báo | trung bình | TC-02-22, 56 | `owner/ta-members.webp` |
| BUG-v5-02-7 | `/inbox?ticket=tk-5` | GV 1440 | Hôm nay → `Trả lời` | Hàng Lý Gia Thảo (788–912 px) cắt đáy khung danh sách (876) và vượt khung nhìn 900 | Hàng nằm trọn trong khung nhìn | thấp | AC13 · TC-02-57 | `owner/inbox-tk5.webp` |
| BUG-v5-02-8 | `/students` | GV | Đọc cột Rủi ro | Chỉ nhãn "Cần chú ý", không câu lý do (Hôm nay và `/students/sv-3` có "Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu Bài tập 02") | Cùng một `riskSentence` (TC-02-76) | thấp | AC18 · TC-02-76 | `owner/students-1440.webp` |
| BUG-v5-02-9 | `/students`, `/attendance` 720 | GV | Mở ở 720 px | Bảng rộng 825 / 816 px trong khung 720 → cột Rủi ro / Phát biểu cắt; `/attendance` không có `data-scroll-x` | Cuộn được hoặc gọn lại | thấp | `AUDIT-biên` | `owner/720-*.webp` |

Quan sát (không FAIL TC): (1) bấm chuột vào một hàng điểm danh không đổi hàng đang chọn của bàn phím (phải ↑/↓ từ hàng 1); (2) ô Buổi đổi bề rộng khi dòng tóm tắt xuống hàng → bố cục nhảy; (3) `Duyệt bài` nằm dưới màn hình 1440×900, phải cuộn mới thấy; (4) TA ở Hôm nay có nền 4 việc (không có `setup`, `members`) — khớp việc TA không được vào lớp 761988.

## Phản mẫu · phân quyền · 375–390

- `ui-antipatterns.sh` sạch; `lint` sạch. Hôm nay đúng thứ tự `inbox, attendance, review, thread-new…, ai-pending, setup, members`; đếm 8→7→6→5 đúng; SV không thấy việc của GV (T1o).
- Phân quyền: SV, Admin bị chặn mọi route US-02; TA không có `Tạo lại mã` / `Mời ra khỏi lớp` / `Mời trợ giảng` (TC-02-21); GV không thấy chat riêng D1/D2 ở `/inbox`, `/students/sv-2`, `/`, `/analytics`, `/students`, `/observability`, `/grading` — chỉ ticket D3 (TC-02-26).
- 375–390: `/inbox` danh sách → chi tiết, `← Hộp thư` 89×44; `/attendance` hàng 2 dòng có kẻ, nút ≥ 44, không cuộn ngang; lỗi: ô Buổi cắt chữ (BUG-v5-02-4), tab hộp thư co ở 1440 (BUG-v5-02-3).

## Cảm nhận khi dùng như chủ dự án

Hộp thư tách hai khung dễ đọc, điểm danh bằng phím nhanh (Hoàn tác 5 s, Giả lập mất mạng đúng), AI duyệt Xác nhận / Chỉnh sửa / Loại gọn. Hai thứ làm mất niềm tin: số "cần xem kỹ" đứng im sau khi duyệt hết (Hôm nay không bao giờ sạch), và thẻ Hôm nay của lớp 1 mở nhầm sang lớp 2 khi đang xem lớp 2. Tab hộp thư ở màn rộng gần như biến mất — rất dễ nhận ra khi demo.

## Đề nghị

1. Dev sửa BUG-v5-02-1 (đếm N8 theo trạng thái duyệt) và BUG-v5-02-2 (thêm `course=` vào link thẻ Hôm nay) trước demo.
2. Dev sửa #24a / #24d (BUG-v5-02-3, 4).
3. PM chốt: TA có thuộc 761988 không (BUG-v5-02-6); chuông có hiện thông báo lớp khác lớp đang chọn không (BUG-v5-02-5); `/students` có hiện câu lý do không (BUG-v5-02-8).

## Sửa công cụ QC v5 (theo proposals #21, #23)

| Công cụ | Sửa |
| --- | --- |
| `proto-curl.sh` | `${s}»` / `${k}»` đóng ngoặc đúng; `tc_04_08` đếm bằng `grep -o 'Quá 24 giờ' \| wc -l` |
| `audit.mjs` | `TOUCH` bỏ nhãn không bọc/không trỏ tới checkbox-radio; `HEADER đặc` nhận màu `lab()/oklab()`; bấm ưu tiên nút trong; cột thu gọn lấy bề rộng `aside`; kịch bản form thread mở `Đặt câu hỏi` trước; mới `wide` (viewport nở: `max(innerWidth, scrollWidth) − w`) — **bắt được lỗi /threads 390/375 mà `scrollWidth − innerWidth` bỏ sót**; `routesOnly`, `skipScenarios`, `skipEdge`, `specWhich` |
| `pii-matrix.mjs` | chỉ lấy hộp thoại `[role=dialog]` đang hiện có chữ "thông tin cá nhân" |
| `threads-timeline.mjs` | mở form qua `Đặt câu hỏi`; mở "Nguồn tham khảo" trước khi đọc; T4g chờ nền 4 việc của TA; T13 chờ 0,9 s trước khi rời trang; T15e cho +2/+3 rồi +2 |
| `regress-v24.mjs` (mới) | TC-02-90, TC-00-69, TC-01-153, TC-02-91 |
| Môi trường | Chrome riêng `--headless=new --no-first-run --user-data-dir` (cờ GPU/swiftshader làm treo ảnh trên máy này); `caffeinate -d -i` giữ màn hình |
| Nghi lỗi công cụ | audit "bộ chọn lớp mở" GV 1440 FAIL là dương tính giả (`display:none` bị tính là cắt); `/threads` 390/375 lúc đầu tôi nghi lỗi công cụ vì trang không cuộn — **đo tay xác nhận lỗi thật** (innerWidth 772) |
