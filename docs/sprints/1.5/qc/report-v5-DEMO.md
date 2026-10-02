# QC report v5 — DEMO (kịch bản 15 phút, `docs/DEMO_SCRIPT.md` bước 1–7)  · Kết luận: **FAIL**

Nhánh `sprint/1.5-mock-ui`, HEAD `8bb6861`, ngày 2026-10-01. Server `http://localhost:3400` (build sẵn, không build lại).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-DEMO.md` — **39 TC** (TC-DEMO-00 … TC-DEMO-38; đếm lại số hàng bảng TC: 39 hàng, không phải 40).
Công cụ: **B** — Chrome riêng (`--user-data-dir=/tmp/qcv5-v5qdemo`, headless, 1440×900 cho GV/Admin, 390×844 cho SV B/SV D), đi tay trọn kịch bản một lượt bằng **menu hồ sơ** (không đặt cookie tay), rồi dựng và chạy lại bằng script `docs/sprints/1.5/qc/scripts/demo-run.mjs`. Ảnh: `docs/sprints/1.5/shots/qc-v5/v5qdemo/` (35 ảnh tay + `run/` 20 ảnh script).

**Tóm tắt:** 35/39 TC PASS; **4 FAIL**: TC-DEMO-20, TC-DEMO-26, TC-DEMO-29, TC-DEMO-38.
Ba lỗi chặn demo: (1) "Giả lập mất mạng" ở điểm danh làm **mất trạng thái đã chốt buổi 10** → điểm SV B tụt 8,5 → 8,3 và cả chuỗi số sau đó sai (8,7 thành 8,4); (2) ô "Quy tắc làm tròn" (D5) **chỉ nhận 1 ký tự** rồi biến mất, không sửa lại được → công thức lớp 2 xác nhận với giá trị `L`; (3) việc "Trả lời" ở màn Hôm nay mở `/inbox` theo lớp đang chọn → **ngõ cụt "Không còn câu hỏi đang chờ"** ngay giữa bước 3.

## Cổng đã chạy

| Lệnh / script | Kết quả |
| --- | --- |
| Đi tay trọn kịch bản bước 1–7 (một trình duyệt, đổi vai bằng menu hồ sơ) | Đi hết, **2 bước ngoài kịch bản** (đổi lớp ở bước 3; lưu điểm danh lần hai trước bước 5) |
| `demo-run.mjs` (lượt dựng lại, bỏ bước tuỳ chọn TC-DEMO-20) | 19/19 nhóm bước PASS, tổng **170,9 s** (2:51) — xem bảng dưới |
| `demo-run.mjs` lượt 2 sau `Đặt lại dữ liệu demo` | Mọi giá trị về gốc: inbox 5 chờ / 1 đã nhận, `/grading` 9/28 + 4 bài khớp lọc, buổi 10 "30 có mặt · Chưa có thay đổi", `/me` QT **8,3**, phát biểu 3 lần +0,75, BT03 "Đang chấm" |
| Console + mạng ngoài | 0 lỗi console / 0 `pageerror` suốt lượt đi tay; `performance.getEntriesByType('resource')` không có tài nguyên ngoài `localhost:3400` |

Kết quả `demo-run.mjs` (máy, không có bước tuỳ chọn mất mạng):

| Nhóm bước | ms | Kết quả |
| --- | --- | --- |
| TC-DEMO-01 admin `/admin/courses` | 4.564 | PASS |
| TC-DEMO-02 GV chuông | 3.773 | PASS |
| TC-DEMO-03 SV D `/join/BX4P9TW` | 5.718 | PASS |
| TC-DEMO-04 GV duyệt D | 9.772 | PASS |
| TC-DEMO-05 SV D tải lại | 5.180 | PASS |
| TC-DEMO-06..09 chat D1/D2 + nguồn | 26.691 | PASS |
| TC-DEMO-10 D3 chuyển GV | 7.345 | PASS |
| TC-DEMO-11..13 inbox: nhận + D4 | 13.156 | PASS (có bước đổi lớp bù, xem BUG-v5-DEMO-3) |
| TC-DEMO-14..15 SV B nhận trả lời, đóng | 6.969 | PASS |
| TC-DEMO-16..18 điểm danh buổi 10 | 7.368 | PASS (đánh 4 ô: 1.460 ms) |
| TC-DEMO-19 `/me` 8,5 | 4.598 | PASS |
| TC-DEMO-21 BT03 "Đang chấm" | 1.367 | PASS |
| TC-DEMO-22..25 chấm + công bố | 14.889 | PASS |
| TC-DEMO-26 SV B thấy 8,0 · QT 8,7 | 6.650 | PASS **chỉ khi bỏ bước mất mạng** |
| TC-DEMO-27..30 công thức lớp 2 | 18.811 | PASS (giá trị D5 sai, xem TC-DEMO-29) |
| TC-DEMO-31..32 sổ điểm lớp 1 + XLSX | 8.015 | PASS |
| TC-DEMO-33 What-if 8,3 | 5.842 | PASS |
| TC-DEMO-34..36 insights lớp 1 + ghim | 12.712 | PASS |
| TC-DEMO-37 insights lớp 2 | 3.807 | PASS |
| **Tổng** | **170.907 (2:51)** | dưới ngưỡng 13:45 |

Giờ đo khi đi tay (chỉ thời gian máy phản hồi, chưa tính lời dẫn): bước 1 ≈ 19 s · bước 2 ≈ 17 s · bước 3 ≈ 24 s · bước 4 ≈ 16 s · bước 5 ≈ 23 s · bước 6 ≈ 29 s · bước 7 ≈ 20 s → **≈ 2 phút 27 giây** thao tác thực.

## TC

| TC-id | PASS/FAIL | Cách kiểm | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-DEMO-00 | PASS | B | `/login` + `/admin/courses` + Hôm nay GV: dải "Bản mô phỏng · dữ liệu giả"; "An ninh mạng"; 761987 + 761988, sĩ số 30 + 30; "Thứ Năm, 29 tháng 10 · tuần 10"; HK1 2026–2027 · ảnh `000`,`010`,`020` |
| TC-DEMO-01 | PASS | B | `/admin/courses`: 2 lớp, cả hai "TS. Lê Thu Hà", 761987 "Đang học hôm nay 09:20", 761988 "Mới mở hôm qua 16:40" · ảnh `010` |
| TC-DEMO-02 | PASS | B | Chuông GV: "Bạn được phân công lớp An ninh mạng – 761988. Mã tham gia: BX4P9TW"; việc "Thiết lập lớp mới — 761988" 4/4 bước · ảnh `021` |
| TC-DEMO-03 | PASS | B | SV D 390 `/join/BX4P9TW`: xem trước có "An ninh mạng (INT1006)", mã lớp 761988, TS. Lê Thu Hà, HK1 2026–2027; sau bấm: "Đã gửi yêu cầu, chờ giảng viên duyệt"; không cuộn ngang · ảnh `030`,`031` |
| TC-DEMO-04 | PASS | B | Bộ chọn lớp → "Quản lý lớp này" → `Duyệt` D: "Yêu cầu chờ duyệt (4)" → **(3)**, "Thành viên (24)" → **(25)**, 3 yêu cầu seed còn nguyên · ảnh `040`,`041`,`042` |
| TC-DEMO-05 | PASS | B | SV D 390 `/` tải lại: bộ chọn lớp có "An ninh mạng – 761988", không còn ô nhập mã trên trang (chỉ còn mục "Tham gia lớp bằng mã" trong bộ chọn), `scrollWidth−innerWidth = 0` · ảnh `050` · xem BUG-v5-DEMO-7 |
| TC-DEMO-06 | PASS | B | SV B 390 `/chat` + D1: "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI"; trả lời hiện dần (độ dài trang 460 → 663 ký tự trong 3,4 s); "vắng 2 buổi (10/09, 08/10)", "0,75 điểm cho 3 lần phát biểu"; không `[[SV_` · ảnh `061`,`062` |
| TC-DEMO-07 | PASS | B | `Hữu ích` → "Đã ghi nhận, cảm ơn bạn." ngay tại khối; `[role=status]/[role=alert]/[class*=toast]` rỗng · ảnh `070` |
| TC-DEMO-08 | PASS | B | D2 → "Mình chỉ trả lời được thông tin của chính bạn. Nếu cần trao đổi về bạn khác, hãy hỏi giảng viên."; không có con số nào về SV C · ảnh `080` |
| TC-DEMO-09 | PASS | B | `Nguồn tham khảo (1)` mở ngay dưới câu trả lời D1: "Quy chế môn học An ninh mạng – 761987 · trang 2 · mục Điểm quá trình" · ảnh `090` |
| TC-DEMO-10 | PASS | B | D3 → "AI chưa đủ chắc chắn về câu này… đã chuyển câu hỏi cho giảng viên" + "Đang chờ giảng viên · vừa gửi" (tự chuyển, không phải bấm) · ảnh `100` |
| TC-DEMO-11 | PASS | B | `/inbox` lớp 1, lọc "Đang chờ (6)": ticket "Trần Thu Uyên · 3 phút trước · Thi cuối kỳ…"; chi tiết có câu hỏi gốc, "Đã chờ 4 phút", "Lý do chuyển: Độ tin cậy 0,42 < 0,80 — không tài liệu nào trả lời được" · ảnh `113`,`114` · **phải đổi lớp bằng tay** (BUG-v5-DEMO-3) |
| TC-DEMO-12 | PASS | B | `Nhận` → "Trạng thái: Đã nhận" + "Bạn đã nhận lúc 09:20"; ticket seed do người khác nhận ghi đúng tên "Phạm Quốc Bảo đã nhận lúc 09:12" · ảnh `120`,`115` |
| TC-DEMO-13 | PASS | B | D4 → `Gửi trả lời`: "Trạng thái: Đã trả lời", "Lê Thu Hà · 09:20", "Đã gửi thư thông báo tới email của sinh viên (mô phỏng)" (thay bước Mailpit 05:55, FR-X7) · ảnh `130` |
| TC-DEMO-14 | PASS | B | SV B 390 chuông → "Giảng viên đã trả lời câu hỏi của bạn" → `/chat`: khối "Giảng viên Lê Thu Hà trả lời" + nguyên văn D4 trong chính phiên chat · ảnh `140`,`141` |
| TC-DEMO-15 | PASS | B | `Đã rõ` → "Câu hỏi đã đóng" · ảnh `150`,`151` · nhưng câu trả lời của GV biến mất khỏi phiên (BUG-v5-DEMO-4) |
| TC-DEMO-16 | PASS | B | GV 390 `/` chạm việc "Điểm danh buổi 10 đang diễn ra" → `/attendance?session=10`, tiêu đề "buổi 10 · Thứ Năm, 29 tháng 10 09:00–11:30 · P.302 – G2", nhãn "Buổi 10 · 29/10 · đang diễn ra" · ảnh `170` |
| TC-DEMO-17 | PASS | B | Vắng ×2 (Lê Quang Huy, Vũ Khánh Huy), Muộn ×1 (Lý Gia Thảo), `Phát biểu` cho Trần Thu Uyên → "1 lần · +0,25"; dải đếm "27 có mặt · 1 muộn · 2 vắng" + "Đã lưu"; **thời gian 4 thao tác: 1,3 s** (script: 1,46 s) « 60 s; vùng chạm 85×44 / 91×44 · ảnh `171` |
| TC-DEMO-18 | PASS | B | `Lưu điểm danh` → "Đã lưu 09:27" + "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; khối là `[role=status]` **position: static** nằm trong luồng trang (không phải toast nổi), không có chữ "Thành công" · ảnh `180` |
| TC-DEMO-19 | PASS | B | SV B 390 `/me`: "Điểm cộng phát biểu 4 lần × 0,25 (trần 1,00) +1,00"; "7,50 + 1,00 − 0,00 = 8,50 → làm tròn 0,1" → **QT 8,5** · ảnh `190` |
| TC-DEMO-20 | **FAIL** | B | "Giả lập mất mạng" → đổi ô (Đỗ Đức Hà = Vắng phép) → "Đang chờ mạng · 1 thay đổi" → tắt → "Đã lưu". Ô mới giữ được, **nhưng trạng thái đã chốt của buổi 10 bị mất**: dải "Đã hoàn tất buổi 10" biến mất, nút về "Lưu điểm danh / Chưa có thay đổi", và `/me` của SV B tụt **8,5 → 8,3** (phát biểu 4 lần → 3 lần, "Buổi đã ghi" 10 → 9). Repro sạch: A ngay sau lưu 8,5 → B chỉ rời trang & quay lại vẫn 8,5 → C sau giả lập mất mạng còn 8,3 · ảnh `200`,`201`,`262`,`264` → BUG-v5-DEMO-1 |
| TC-DEMO-21 | PASS | B | SV B 390 `/assignments/bt03` (mở từ `/me`): "Nộp lúc 08:10 Thứ Sáu, 23 tháng 10", "Nộp muộn 1 ngày · trừ 0,5 điểm", "Đang chấm"; không có điểm nháp / từ kỹ thuật · ảnh `210` |
| TC-DEMO-22 | PASS | B | `/grading` lọc mặc định (Cần xem kỹ 4 + Chưa duyệt 19 → "4 bài khớp bộ lọc"): bài Trần Thu Uyên ở đầu, lý do "Hai lượt chấm lệch 1,5 điểm" · ảnh `220` |
| TC-DEMO-23 | PASS | B | Mở `grading/sub-bt03-sv-2`: thông báo vàng ở tiêu chí 2 "Hai lượt chấm lệch 1,5 điểm · Lượt 1 chấm 1,00, lượt 2 chấm 2,50"; cả 4 tiêu chí đều có "Đoạn trích từ bài làm"; tăng 1,00 → 2,50 thì tổng đổi ngay theo từng nấc (6,8 → 7,0 → 7,3 → 7,5 → 7,8 → **8,0**), dòng chốt "Duyệt bài sẽ chốt 8,0 điểm"; sửa nhận xét tiêu chí 2 giữ nguyên · ảnh `230`,`231` |
| TC-DEMO-24 | PASS | B | `Duyệt bài` → quay về hàng chờ, "10/28 bài đã duyệt", bài B rời lọc "Chưa duyệt" · ảnh `240` |
| TC-DEMO-25 | PASS | B | Bỏ lọc "Chưa duyệt" → chọn hàng B → `Công bố` → hộp thoại "Công bố cho 1 sinh viên… Bài của Trần Thu Uyên công bố ở mức 8,0 (đã trừ 0,5 nộp muộn)" → "Đã công bố"; vai TA cùng màn hiện "Chỉ giảng viên công bố điểm", không có nút · ảnh `250`,`252`,`253`,`254` |
| TC-DEMO-26 | **FAIL** | B | Trên đường đi kịch bản (có bước tuỳ chọn TC-DEMO-20): `/me` ra **QT 8,4** ("7,67 + 0,75 − 0,00 = 8,42"), không phải 8,7 — do lỗi BUG-v5-DEMO-1; chỉ sau khi bấm `Lưu điểm danh` lần hai (bước ngoài kịch bản) mới ra 8,7. Thêm: màn SV hiện **đoạn trích không phải bài của SV B** ("Nhóm tấn công vào hệ thống bán lẻ qua tài khoản VPN của một nhà thầu bảo trì…", "240 cửa hàng") trong khi bài B là SolarWinds Orion → BUG-v5-DEMO-2. Điểm 8,0 và nhận xét theo tiêu chí thì đúng · ảnh `260`,`262`,`265` |
| TC-DEMO-27 | PASS | B | `/gradebook` lớp 2: banner "Công thức điểm chưa xác nhận… Chưa chốt được điểm cho tới khi công thức được xác nhận"; `Chốt điểm` `aria-disabled=true` kèm lý do hiện sẵn "Chưa chốt được: công thức điểm của lớp này chưa được xác nhận…" (hiện thường trực, không cần rê) · ảnh `270` |
| TC-DEMO-28 | PASS | B | `/gradebook/scheme` lớp 2 → `Tải quy chế`: "Đang đọc nội dung quy chế…" rồi bản nháp 5 mục, mỗi mục có đoạn trích + "Trang 2/3/5"; mục "Quy tắc làm tròn — Chưa rõ" chặn: "Chưa xác nhận được: mục “Quy tắc làm tròn” còn trống" · ảnh `280`,`281`,`282` · (không có hộp chọn tệp thật — tải giả lập, đúng phạm vi prototype) |
| TC-DEMO-29 | **FAIL** | B | Gõ D5 "Làm tròn đến 0,1" (120 ms/ký tự, như gõ tay): ô **chỉ nhận ký tự đầu "L"** rồi tự biến mất, không bấm lại để sửa được; mục hiển thị "Quy tắc làm tròn · Trang 5 · **L**", hộp thoại xác nhận đọc thành "…trừ 0,5 mỗi buổi vắng từ buổi thứ 3, **l**. Sinh viên sẽ thấy…". Trạng thái cuối "Đã xác nhận" thì đúng, nhưng công thức xác nhận với giá trị sai · ảnh `290`,`291`,`293` → BUG-v5-DEMO-5 |
| TC-DEMO-30 | PASS | B | `/gradebook` lớp 2 sau xác nhận: banner biến mất, "Công thức đã xác nhận", `Chốt điểm` hết `aria-disabled` · ảnh `300` |
| TC-DEMO-31 | PASS | B | `/gradebook` lớp 1, menu hàng B → "Xem giải trình điểm": "(7,0 + 8,0 + 8,0) ÷ 3 = 7,67", "Điểm cộng phát biểu 4 lần × 0,25 +1,00" (có +0,25 của bước 4), "7,67 + 1,00 − 0,00 = 8,67 → **8,7**", dòng "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" · ảnh `310`,`311` |
| TC-DEMO-32 | PASS | B | Menu "Thêm hành động với sổ điểm" → `Xuất XLSX` → "Đã tạo file sổ điểm (mô phỏng) · gồm cột điểm thành phần và dòng ghi chú điểm chính thức" · ảnh `320`,`321` |
| TC-DEMO-33 | PASS | B | SV B `/me`, ô cuối kỳ = 8,0 → "điểm học phần sẽ là **8,3** · 40% × 8,7 + 60% × 8,0" · ảnh `330` |
| TC-DEMO-34 | PASS | B | `/insights` lớp 1 → `Tạo báo cáo mới`: "Đang tạo báo cáo" ~2 s → báo cáo 29/10 09:20, 80 câu hỏi; #1 "Mật mã đối xứng (AES, CBC)" (14 SV · 31 câu · +6), #2 "Hàm băm và chữ ký số"; mỗi chủ đề có lý do + 5 tín hiệu + "Nên làm" · ảnh `340`,`341` |
| TC-DEMO-35 | PASS | B | "Xem 5 câu hỏi đã ẩn danh" của chủ đề A: 5 câu, không tên/MSSV; chủ đề "Bắt tay TLS 1.3" (2 SV): "Dưới 3 sinh viên hỏi — không hiện câu mẫu để không suy ngược ra được là ai hỏi" · ảnh `350` |
| TC-DEMO-36 | PASS | B | `Tạo thread ghim` → "Đã ghim thread “Mật mã đối xứng (AES, CBC)” vào Threads của lớp 761987" (có Hoàn tác); kiểm bằng vai SV B: `/threads` lớp 1 có thread ghim "Chủ đề đang vướng: Mật mã đối xứng (AES, CBC)" ở đầu · ảnh `360`,`361`,`362` |
| TC-DEMO-37 | PASS | B | `/insights` lớp 2: báo cáo 28/10, #1 "Tường lửa và phân đoạn mạng" (chủ đề C, SRS 4.1), không có chủ đề nào của lớp 1 · ảnh `370` |
| TC-DEMO-38 | **FAIL** | B | Thời gian đạt (thao tác thực ≈ 2:27 tay, 2:51 script « 13:45) và không phải sửa URL tay ngoài lần mở đầu; **nhưng có ngõ cụt**: ở mốc 04:45 việc "Trả lời" (ghi "An ninh mạng – 761987") mở `/inbox?ticket=tk-5` nhưng lọc theo lớp đang chọn 761988 → "Không còn câu hỏi đang chờ" (ảnh `112`); phải thêm 2 bước ngoài kịch bản (đổi lớp; lưu điểm danh lần hai) để kịch bản ra đúng số |

## Lỗi

| ID | Route | Vai | Bước tái hiện | Thấy | Mong đợi | Mức | AC / TC | Ảnh |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BUG-v5-DEMO-1 | `/attendance` → `/me` | GV 390 → SV B 390 | Đặt lại dữ liệu → buổi 10: ghi `Phát biểu` cho Trần Thu Uyên → `Lưu điểm danh` ("Đã hoàn tất buổi 10") → bật "Giả lập mất mạng" → đổi 1 ô → tắt | Dải "Đã hoàn tất buổi 10" biến mất, nút về "Lưu điểm danh · Chưa có thay đổi"; SV B: QT **8,3**, "Phát biểu 3 lần · +0,75", "Buổi đã ghi 9 buổi" | Giữ nguyên trạng thái đã chốt và QT 8,5; đồng bộ chỉ thêm thay đổi mới | **cao** | TC-DEMO-20, 19, 26 | `200`,`201`,`262`,`264` |
| BUG-v5-DEMO-2 | `/assignments/bt03` (SV) | SV B 390 | Công bố BT03 → SV mở bài | 4 "đoạn trích bằng chứng" nói về vụ tấn công hệ thống bán lẻ qua VPN nhà thầu / 240 cửa hàng — **không phải bài của B** (bài B: SolarWinds Orion, SUNBURST, Golden SAML) | Trích đúng từ bài làm của chính SV, khớp với đoạn trích GV thấy ở `/grading` | **cao** | TC-DEMO-26 | `230`,`260` |
| BUG-v5-DEMO-3 | `/` → `/inbox` (GV) | GV 1440 | Sau bước 1 (bộ chọn lớp đang ở 761988 vì vừa quản lý lớp 2) → Hôm nay → bấm `Trả lời` của việc ghi "An ninh mạng – 761987" | Mở `/inbox?ticket=tk-5` nhưng danh sách lọc theo lớp 761988 → "Không còn câu hỏi đang chờ"; phải tự đổi lớp mới thấy ticket | Bấm việc của lớp nào thì mở đúng lớp đó và đúng ticket | **cao** | TC-DEMO-11, 38 | `110`,`112`,`113` |
| BUG-v5-DEMO-4 | `/chat` (SV) | SV B 390 | Sau khi GV trả lời → bấm `Đã rõ` (hoặc tải lại) | Câu trả lời của GV và nhãn "Giảng viên Lê Thu Hà trả lời" **biến mất**, chỉ còn "Câu hỏi đã đóng" | Giữ câu trả lời trong phiên chat sau khi đóng (bước 3 của kịch bản dựa vào đó) | **cao** | TC-DEMO-14, 15 | `141`,`151`,`152` |
| BUG-v5-DEMO-5 | `/gradebook/scheme` lớp 2 | GV 1440 | Tải quy chế → bấm ô "Quy tắc làm tròn" → gõ "Làm tròn đến 0,1" (120 ms/ký tự) | Ô chỉ nhận ký tự đầu rồi tự đóng; mục ghi **"L"**, không bấm lại để sửa được; hộp thoại xác nhận in ra "…từ buổi thứ 3, l." | Nhận đủ chuỗi D5, cho sửa lại trước khi xác nhận | **cao** | TC-DEMO-29 | `291`,`293` |
| BUG-v5-DEMO-6 | `/` (GV) | GV 1440/390 | Sau khi lưu điểm danh buổi 10 → quay lại Hôm nay | Vẫn "6 việc cần xử lý hôm nay" và vẫn còn việc "Điểm danh buổi 10 đang diễn ra · 30 sinh viên chưa được chốt điểm danh" dù trang tự ghi "việc đã xử lý sẽ rời khỏi danh sách" | Việc rời danh sách hoặc đổi trạng thái sau khi chốt | vừa | TC-DEMO-18 | `221` |
| BUG-v5-DEMO-7 | `/` (SV D) | SV D 390 | Được duyệt vào 761988 (lớp tuần 3, học Thứ Ba) → mở Hôm nay (hôm nay là Thứ Năm) | "Hôm nay · P.207 – G3 · Thứ Ba 07:00–08:30" và "**Buổi 10** · An ninh mạng · Đang diễn ra"; khối "Học dở" có "bạn đọc dở 2 giờ trước", "Chat riêng · phiên trước hôm qua 20:10" cho SV vừa vào lớp | Buổi đúng của lớp 2 (tuần 3), không có buổi "đang diễn ra" vào ngày lớp không học, không có lịch sử học giả | vừa | TC-DEMO-05 | `050` |
| BUG-v5-DEMO-8 | `/gradebook` lớp 2 | GV 1440 | Duyệt SV D ở bước 1 (Thành viên 25) → bước 6 mở sổ điểm lớp 2 | "**24 sinh viên**" | 25 (khớp màn Thành viên lớp) | vừa | TC-DEMO-04, 27 | `270`,`300` |
| BUG-v5-DEMO-9 | `/gradebook` lớp 1 → "Lịch sử sửa điểm" hàng B | GV 1440 | Sau khi sửa tiêu chí 2 và công bố BT03 → mở "Lịch sử sửa điểm" của SV B | Chỉ có 4 dòng cũ (22/10 AI chấm nháp ×2, 05/10 BT02, 20/09 BT01); **không có dòng nào cho việc GV sửa điểm tiêu chí 2 và công bố BT03 hôm nay** | Ghi lại thao tác vừa làm ("mỗi thay đổi điểm đều ghi lại người sửa và thời điểm") | vừa | TC-DEMO-23, 25, 31 | `312` |
| BUG-v5-DEMO-10 | `/inbox` (rỗng) | GV 1440 | Mở `/inbox` của lớp chưa có ticket | Khối rỗng: tiêu đề "Không còn câu hỏi đang chờ" rồi **~270 px trắng** mới tới câu giải thích, không biểu tượng/khung — nhìn như trang vỡ | Khối rỗng gọn một cụm (DESIGN §21) | thấp | — | `112` |
| BUG-v5-DEMO-11 | `/chat` (SV) | SV B 390 | Sau khi GV ghi +0,25 ở bước 4, cuộn lại câu trả lời D1 | Thân câu trả lời vẫn ghi "được cộng 0,75 điểm cho 3 lần phát biểu" trong khi chip ngay dưới cùng tin nhắn đã là "Phát biểu · 4 lần · +1,00" | Nhất quán trong một tin nhắn (hoặc giữ nguyên cả hai theo thời điểm hỏi) | thấp | — | `152` |
| BUG-v5-DEMO-12 | menu hồ sơ | mọi vai | "Đặt lại dữ liệu demo" khi đang xem lớp 2 | Dữ liệu về gốc nhưng **lớp đang chọn vẫn là 761988**; diễn lại kịch bản phải tự đổi về 761987 | Về lớp mặc định của vai (761987) để chạy lại từ đầu | thấp | Quy tắc "lần chạy thứ hai" | `380` |

## Phản mẫu UI / phân quyền / ngưỡng 375–390 (DESIGN §21, §22)

- **Không toast "Thành công"**: xác nhận lưu điểm danh là `[role=status]` `position: static` nằm trong luồng trang ("Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"); `Hữu ích` đổi tại chỗ; công bố điểm và xác nhận công thức đều qua hộp thoại nêu hậu quả, không có thông báo nổi.
- **Không spinner giữa trang**: tiến độ nền ở `/insights` ("Đang tạo báo cáo") và `/gradebook/scheme` ("Đang đọc nội dung quy chế…") là chữ tại chỗ.
- **Đỏ đúng ba nghĩa**: nút chính một cái mỗi vùng (`Làm bài`, `Trả lời`, `Công bố điểm`, `Xác nhận công thức`), còn lại là nhãn trạng thái chấm tròn hổ phách — không thấy đỏ trang trí.
- **390 px**: `/join`, `/`, `/chat`, `/me`, `/assignments/bt03`, `/threads`, `/attendance` đều `scrollWidth − innerWidth = 0`; ô trạng thái điểm danh 85×44, nút `Phát biểu` 91×44 — đạt ngưỡng chạm.
- **Phân quyền**: TA ở `/grading` thấy "Chỉ giảng viên công bố điểm" và không có nút `Công bố`; SV không thấy điểm nháp ở BT03 trước công bố, không thấy từ kỹ thuật ("độ tin cậy", "AI chấm nháp") ở bất kỳ màn SV nào đã đi.
- Phản mẫu còn lại: khối rỗng `/inbox` giãn ~270 px (BUG-10) và chuỗi bị cắt trong hộp thoại xác nhận công thức (BUG-5).

## Cảm nhận khi dùng như chủ dự án

- **Chỗ gãy to nhất là bước 3**: vừa khoe xong việc duyệt sinh viên vào lớp 2, bấm đúng cái việc đang ghi "An ninh mạng – 761987" trên màn Hôm nay thì rơi vào hộp thư trống trơn của lớp 2. Trên sân khấu đây là 10–15 giây lúng túng, lại đúng lúc đang nói "AI không chắc thì chuyển cho người thật".
- **Nhịp bước 4 → 5 rất tốt** cho tới khi đụng "Giả lập mất mạng": thao tác điểm danh nhanh thật (4 ô trong hơn 1 giây máy), câu chốt "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng" rất đáng khoe. Nhưng nếu diễn nốt nhánh mất mạng thì điểm của sinh viên B lặng lẽ tụt về 8,3 — hội đồng nhìn `/me` sau đó sẽ thấy con số không khớp lời dẫn, mà không có gì báo là đã mất.
- **Bài chấm là chỗ "giả" lộ nhất**: ở `/grading` bài của B viết về SolarWinds, rất thuyết phục; nhưng khi công bố, sinh viên lại đọc "đoạn trích từ bài của chính mình" nói về siêu thị bán lẻ và 240 cửa hàng. Ai đọc kỹ hai màn liền nhau sẽ thấy ngay đây là dữ liệu dán ghép.
- **Ô "Quy tắc làm tròn"** là chi tiết đắt nhất của bước 6 (AI thú nhận chỗ chưa rõ, người quyết) nhưng gõ vào chỉ còn chữ "L", và hộp thoại xác nhận in ra "…từ buổi thứ 3, l." — đúng khoảnh khắc cần tin cậy nhất thì chữ bị cụt.
- **Màn Hôm nay của GV không "tiêu" việc**: lưu điểm danh xong vẫn còn "30 sinh viên chưa được chốt điểm danh" ngay trên trang nói "việc đã xử lý sẽ rời khỏi danh sách". Nhìn như màn hình tĩnh.
- **Sinh viên D vừa vào lớp đã có "học dở" và một "Buổi 10 đang diễn ra"** cho lớp học Thứ Ba, trong khi hôm nay là Thứ Năm — người xem tinh ý sẽ hỏi ngay, và đó là câu hỏi mình không muốn trả lời giữa demo.
- Mặt sáng: chuỗi số 8,3 → 8,5 → 8,7 → What-if 8,3 khớp SRS 4.1 từng bước, giải trình điểm có đủ phép tính và nguồn; báo cáo lỗ hổng kiến thức đọc như do người viết (lý do + tín hiệu + "Nên làm"), tách lớp sạch; `Đặt lại dữ liệu demo` đưa mọi thứ về gốc nên diễn lại được nhiều lần.

## Đề nghị

1. Sửa trước buổi bảo vệ (chặn): BUG-v5-DEMO-1 (mất trạng thái chốt buổi sau đồng bộ), BUG-v5-DEMO-5 (ô làm tròn chỉ nhận 1 ký tự), BUG-v5-DEMO-3 (việc Hôm nay không mở đúng lớp), BUG-v5-DEMO-2 (đoạn trích không phải bài của SV), BUG-v5-DEMO-4 (mất câu trả lời GV trong chat sau khi đóng).
2. Nếu không kịp sửa BUG-1: bỏ hàng "(tuỳ chọn) Giả lập mất mạng" khỏi kịch bản demo, vì chạy nó là hỏng toàn bộ con số của bước 5–6.
3. Sửa tiếp theo (vừa): việc điểm danh còn lại ở Hôm nay, sĩ số lớp 2 lệch 24/25, nhật ký sửa điểm không ghi thao tác hôm nay, "Buổi 10 đang diễn ra" và "học dở" của SV D ở lớp 2.
4. Câu hỏi cho PM: TC-DEMO-20 nói "không mất" — xác nhận giúp là trạng thái "đã chốt buổi" cũng thuộc phạm vi "không mất" (QC đang chấm theo hướng đó, nên FAIL).
5. Dùng lại `docs/sprints/1.5/qc/scripts/demo-run.mjs` làm cổng hồi quy trước mỗi lần diễn: chạy 2 lượt (một lượt ngay sau `Đặt lại dữ liệu demo`) mất ~3 phút và bắt được đúng các con số SRS 4.1.
