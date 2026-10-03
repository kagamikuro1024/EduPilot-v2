# QC test case — US-PROTO-02 (Giảng viên / trợ giảng: vận hành lớp)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` 4.1, 4.4. Hộp đen. Công cụ: **C** = `proto-curl.sh`, **B** = trình duyệt. Đặt lại dữ liệu demo trước chuỗi TC.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-02-01 | AC1 | – | **C** `tc_02_01` (GV: `/ /inbox /students /students/sv-3 /attendance /class/members`) | Đều `MO` |
| TC-02-02 | AC1 / SRS 4.4 `/` | GV, lớp "Tất cả lớp của tôi" | **B** Mở `/` | "6 việc cần xử lý hôm nay" đúng thứ tự: (1) ticket chờ 3 ngày 4 giờ, (2) điểm danh buổi 10 lớp 761987 đang diễn ra, (3) BT03 có 4 bài "Cần xem kỹ", (4) 1 câu AI chờ xác nhận, (5) "Thiết lập lớp mới — 761988" (chia sẻ mã → tải quy chế → tạo lịch buổi học → tải tài liệu), (6) 3 yêu cầu vào lớp chờ duyệt; mỗi việc ghi tên lớp |
| TC-02-03 | AC1 (sửa theo v5.1) | – | **B** `/` | Chuông có thông báo "Bạn được phân công lớp An ninh mạng – 761988" kèm `BX4P9TW`; **không thẻ số liệu, không biểu đồ** ở khung nhìn đầu (DESIGN §21: ≥ 3 KPI cùng cỡ là phản mẫu); "Lớp cần chú ý · 8 sinh viên" (N3) với 3 người đầu và `Xem cả 8` (v5.1 thay "C + 2 SV"); dải lịch sắp tới |
| TC-02-04 | AC1 | – | **B** Bấm từng việc; xử lý một việc | Đúng màn đích; việc đã xử lý biến khỏi danh sách; hết việc → "Không còn việc cần bạn quyết định" |
| TC-02-05 | AC2 | B đã gửi D3 (US-PROTO-01 TC-01-07) | **B** GV → `/inbox` → mở ticket D3 → `Nhận` | Trạng thái Claimed, tên giảng viên |
| TC-02-06 | AC2 | Sau TC-02-05 | **B** Gõ D4 (nguyên văn) → `Gửi trả lời` | Chuyển Answered; dòng "Đã gửi thư thông báo tới email của sinh viên (mô phỏng)" hiện tại chỗ (FR-X7); tuỳ chọn `Lưu thành tri thức` có |
| TC-02-07 | AC2 / SRS mục 3 | – | **B** Ticket mẫu "đã có người nhận" | Hiện "Phạm Quốc Bảo đã nhận lúc 09:12" thay nút `Nhận` (409 giả lập) |
| TC-02-08 | AC2 (bổ sung v5.1) | – | **B** Danh sách ticket | 5 ticket mở (26 phút, 3 giờ, 1 ngày, 2 ngày, 3 ngày 4 giờ) + 1 đã có người nhận; hàng có tên SV, câu ngắn, tuổi, trạng thái, lý do ("Độ tin cậy 0,42 < 0,80" — GV được thấy); 3 hàng quá 24 giờ (1 ngày, 2 ngày, 3 ngày 4 giờ) có nhãn "Quá 24 giờ" (N1) |
| TC-02-09 | AC3 | `/attendance` buổi 10 lớp 1 (29/10) | **B** Bàn phím: ↑/↓ chọn hàng, `4` đánh vắng 2 SV, `2` một SV muộn, `P` cho B → `Lưu điểm danh`; bấm giờ | Mỗi thao tác hiện "Đã đánh vắng · Hoàn tác" 5 s + "Đã lưu 09:2x"; kết thúc "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; toàn bộ ≤ 60 s |
| TC-02-10 | AC3 | Sau TC-02-09 | **B** Đổi vai B → `/me` | QT = 8,5 |
| TC-02-11 | AC3 | – | **B** `Hoàn tác` trong 5 s; phím `3`; `P` hai lần | Hoàn tác khôi phục trạng thái; `3` = Vắng phép; `P` hai lần = +0,5 phát biểu (trần +1,0 tính ở `/me`) |
| TC-02-12 | AC3 / SRS 4.4 | – | **B** Bộ chọn buổi 1–15 | Buổi 1–9 đã điểm danh (xem lại), buổi 11+ → "Buổi này chưa diễn ra" |
| TC-02-13 | AC4 | D đã gửi yêu cầu (US-PROTO-01 TC-01-23) | **B** GV → `/class/members` lớp 2 | 24 thành viên + 3 chờ duyệt + D; mã `BX4P9TW`, bật duyệt |
| TC-02-14 | AC4 | Sau TC-02-13 | **B** `Duyệt` D | D thành thành viên; có Hoàn tác; đổi vai về D → bộ chọn lớp có 761988, "Hôm nay" bình thường |
| TC-02-15 | AC4 / SRS 4.4 | GV | **B** `Từ chối` một yêu cầu; `Tạo lại mã` | `Từ chối` có Hoàn tác; `Tạo lại mã` qua xác nhận "mã cũ vô hiệu ngay" → mã mới + `Sao chép link`; mã cũ ở `/join` của D → câu chung "Mã không hợp lệ…" |
| TC-02-16 | AC5 | GV trên desktop (1440 px) | **B** Mở `/inbox` | Bố cục split view 2 panel độc lập (FR-X11, Proposal #16): Cột danh sách ticket (bên trái, max 380px) và Panel chi tiết ticket (bên phải), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc (`var(--radius-sm)` hoặc `var(--radius-md)`), padding rõ ràng (`var(--space-4)` hoặc `var(--space-5)`), cuộn độc lập giữa 2 cột |
| TC-02-17 | AC5 | 375 × 812 | **B** `/inbox` trên mobile | Chuyển thành danh sách → chi tiết (hai bước, nút quay lại); chữ không tràn; vùng chạm ≥ 44 px |
| TC-02-18 | AC5 | 375 × 812 | **B** `/attendance` trên mobile | Hàng có kẻ, **không card**, vùng chạm ≥ 44 px, thao tác bằng chạm; không cuộn ngang |
| TC-02-19 | AC6 (nhánh lỗi) | `/attendance` | **B** Bật "Giả lập mất mạng", đổi 2 ô | "Đang chờ mạng · 2 thay đổi"; tắt công tắc → "Đã lưu 09:2x", không mất thay đổi, không trùng |
| TC-02-20 | AC6 | – | **B** `/students?state=error` | Lỗi có câu khắc phục + `Thử lại` |
| TC-02-21 | AC7 (phân quyền) | – | **C** `tc_02_07` | SV, Admin bị chặn mọi route của US; TA không có `Tạo lại mã`, không có `Mời ra khỏi lớp` |
| TC-02-22 | AC7 | TA (chỉ lớp 761987, SRS 4.1) | **B** `/class/members` lớp 1; `?course=int1006-2` | TA xem được danh sách lớp 761987, không có `Tạo lại mã` / `Mời ra khỏi lớp` / `Mời trợ giảng`; `Duyệt` được khi lớp có yêu cầu chờ (lớp 1 không bật duyệt → N/A nếu không có yêu cầu); lớp 2 không vào được (`?course=` bị bỏ qua) |
| TC-02-23 | SRS 4.4 `/students` (sửa theo v5.1) | GV | **B** Tìm "Huy"/"20229003"/"zzzz"; chip `Cần chú ý`, `Vắng nhiều`, `Điểm giảm`, `Ít hoạt động` | 30 SV lớp 1; chip `Cần chú ý` = **8** (N3, v5.1 thay "C + 2"); tìm theo tên/MSSV; không khớp → rỗng có hướng; bấm hàng → `/students/[id]`, quay lại giữ vị trí cuộn |
| TC-02-24 | SRS 4.4 `/students/[id]` | GV | **B** `/students/sv-3` (C) | Câu rủi ro "Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu BT02"; tab Tổng quan / Chuyên cần / Điểm / Hoạt động học / Ghi chú; ghi chú "Chỉ giảng viên/TA thấy" |
| TC-02-25 | SRS 4.4 | GV | **B** `Thêm ghi chú`; `Nhắn riêng` | Ghi chú thêm tại chỗ + Hoàn tác (không modal); `Nhắn riêng` → dòng xác nhận mô phỏng |
| TC-02-26 | **Quyền chat riêng** (CLAUDE.md) | Sau D1–D3 của B | **B** GV ở mọi route (`/inbox`, `/students/sv-2`, `/`, `/analytics`) | Không thấy nội dung D1/D2 (chat riêng chưa escalate); chỉ thấy ticket D3 (đã escalate) với câu hỏi D3 |
| TC-02-27 | SRS 4.4 trạng thái | – | **B** `?state=` trên 6 route; Lọc Open rỗng ở `/inbox` | Rỗng "Không còn câu hỏi đang chờ"; loading skeleton đúng hình |
| TC-02-28 | AC8 | GV / TA ở `/threads/[id]` có câu trả lời AI `Chờ xác nhận` | **B** Bấm `Xác nhận` | Trạng thái câu trả lời chuyển thành `Đã được giảng viên xác nhận` (verified) |
| TC-02-29 | AC8 | GV / TA ở `/threads/[id]` có câu trả lời AI `Chờ xác nhận` | **B** Bấm `Chỉnh sửa` | Mở ô sửa inline chứa sẵn nội dung câu trả lời AI |
| TC-02-30 | AC8 | Sau TC-02-29 | **B** Sửa nội dung và bấm `Lưu và xác nhận` | Trạng thái chuyển thành `Đã được giảng viên sửa & xác nhận` (CORRECTED), hiển thị nội dung đã sửa kèm nút/chi tiết "Xem câu trả lời AI gốc" để đối chiếu (Proposal #15) |
| TC-02-31 | AC8 | GV / TA ở `/threads/[id]` có câu trả lời AI | **B** Bấm `Loại khỏi tri thức` | Câu trả lời AI bị ẩn/loại bỏ khỏi thread và hiển thị dòng Hoàn tác |
| TC-02-32 | AC9 (#17c) | GV, 1440 × 900, `/inbox` | **A** "02-AC9 chip", AUDIT `/inbox` 1440; **B** | Hàng tab / chip lọc cuộn ngang (`data-scroll-x`), đọc đủ "Tất cả 6" (và Đang chờ, Đã nhận, Đã trả lời); `cut: []`; không chữ nào bị cắt ngang |
| TC-02-33 | AC9 | GV, 1440 | **A** đo cột danh sách; **B** ảnh | Cột danh sách ≤ 380 px, không phần tử vượt cột; mỗi ticket: dòng 1 tên + thời gian, dòng 2 câu hỏi tối đa 2 dòng (ticket dài của Lý Gia Thảo bị cắt đúng 2 dòng, không tràn), dòng 3 meta xuống dòng được (không cắt giữa chữ như v4 "…bấm Nhờ giảng viên hỗ trợ") |
| TC-02-34 | AC9 | GV, 375 và 390 px | **A** `/inbox` | Chỉ thấy danh sách; bấm ticket (Lý Gia Thảo / D3) → chi tiết toàn bề rộng, URL có `?ticket=<id>`, có `← Hộp thư` cao và rộng ≥ 44 px, danh sách ẩn; AUDIT `cut: []` ở 375 |
| TC-02-35 | AC9 | GV, 375 và 390 px | **A** cuộn danh sách 250 px → mở ticket → `← Hộp thư`; lặp lại với Back của trình duyệt | Về danh sách **đúng vị trí cuộn** (±2 px) cả hai cách; URL bỏ `?ticket=` |
| TC-02-36 | AC9 (biên) | GV | **A** 1100 → 1099 px; **B** mở thẳng `/inbox?ticket=<id>` ở 1440 và 375, tải lại ở 375 khi đang xem chi tiết | 1100: hai panel; 1099: chỉ danh sách rồi chi tiết (ngưỡng SRS 4.7 c2). 1440 + `?ticket=`: ticket đó được chọn sẵn; 375: tải lại vẫn ở chi tiết đúng ticket |
| TC-02-37 | AC9 (nhánh lỗi) | GV | **B** `/inbox?ticket=khong-co`; `/inbox?state=error` ở 375 | `?ticket=` sai → quay về danh sách hoặc thông báo rỗng có hướng, không trắng trang, không lỗi console; `?state=error` có `Thử lại`, không vỡ bố cục |
| TC-02-38 | AC9 | GV, 375 | **B** Chip lọc: Đang chờ → Đã nhận → Đã trả lời | Hàng chip cuộn ngang, chip đang chọn tự cuộn vào khung; số trên chip đủ chữ; không cắt |
| TC-02-39 | AC10 (#17d) | GV, 1440 | **A** "02-AC10" cho 5 ticket: Nguyễn Minh Trung, Đặng Gia An, Lê Quang Huy, Đỗ Thanh Long, Lý Gia Thảo | `[data-part=inbox-reply]` cách phần tử liền trước ≤ 24 px (v4: 100 px); `Gửi trả lời` cách ô soạn ≤ 24 px; ảnh ticket Lý Gia Thảo |
| TC-02-40 | AC10 | GV, 1440 | **B** Ticket ngắn (D3) so với ticket có bản nháp AI dài | Panel chi tiết cao theo nội dung, không kéo giãn khối thông tin để lấp chiều cao; không khoảng trống lớn trước hoặc sau ô trả lời |
| TC-02-41 | AC10 | GV | **A** sau `Nhận`; sau `Gửi trả lời` (Answered) | Khoảng cách ≤ 24 px được giữ ở trạng thái Claimed; ở Answered dòng "Đã gửi thư thông báo… (mô phỏng)" nằm sát phía dưới, không khoảng trống |
| TC-02-42 | AC10 (biên) | GV | **B** Ticket "đã có người nhận" (Phạm Quốc Bảo, 09:12) | Thay nút `Nhận` bằng dòng "đã nhận lúc 09:12"; ô "Trả lời của bạn" vẫn cách khối trên ≤ 24 px |
| TC-02-43 | AC11 (#18) | GV, sau `Đặt lại dữ liệu`; B vừa tạo thread khớp mẫu | **T** `T1j`, `T1k`, `T1l` (hoặc **B**: B tạo "Dùng lại IV trong CTR có sao không?" → đổi vai GV) | Hôm nay: "7 việc cần xử lý hôm nay"; có việc "Câu hỏi mới: «Dùng lại IV trong CTR có sao không?» · Mật mã đối xứng · vừa xong" nhãn `Chờ xác nhận`; chuông: "Câu hỏi mới trong Threads: «…»" |
| TC-02-44 | AC11 | Sau TC-02-43 | **T** `T1m`, `T1n` | Bấm việc → `/threads/<id>`; `Xác nhận` → việc rời Hôm nay, tiêu đề "6 việc cần xử lý hôm nay" |
| TC-02-45 | AC11 (nhánh không khớp) | B vừa tạo "WPA3 chặn được KRACK không?" | **T** `T4e`, `T4f` | Hôm nay "7 việc", nhãn `Cần giảng viên trả lời`; GV gửi phản hồi trong thread → việc rời, "6 việc" |
| TC-02-46 | AC11 / H (TA) | TA, cùng trạng thái TC-02-43 và TC-02-45 | **T** `T4g`; **B** | Trợ giảng thấy giống GV: việc, nhãn, đếm, chuông; thao tác của TA làm việc rời |
| TC-02-47 | AC11 / H | GV, sau TC-02-43 | **B** `Chỉnh sửa` → `Lưu và xác nhận`; làm lại ở thread khác với `Loại khỏi tri thức` | Mỗi thao tác (Xác nhận / Chỉnh sửa / Loại / gửi phản hồi) đều làm việc rời Hôm nay và đếm giảm 1 |
| TC-02-48 | AC11 (biên) | GV, sau `Đặt lại dữ liệu demo` | **B** B tạo 2 thread khớp → đếm; Xác nhận một thread mới; Xác nhận `t-cbc` của seed; `Đặt lại dữ liệu demo` | 8 → 7 → 6 (việc seed "1 câu trả lời của AI chờ bạn xác nhận" rời độc lập với việc mới); sau Đặt lại: 6, thread mới biến mất khỏi `/threads` |
| TC-02-49 | AC11 / phân quyền | SV B, D | **T** `T1o`; **B** | SV không thấy việc "Câu hỏi mới" của GV ở `/`; GV không thấy việc của SV; việc dành cho GV/TA không lộ nội dung chat riêng |
| TC-02-50 | AC12 (#19 E3, FR-X17) | GV lớp 761987, sau `Đặt lại dữ liệu` | **B** Hôm nay → thẻ "3 yêu cầu vào lớp chờ duyệt · 761988" → `Duyệt` | URL `/class/members?tab=pending` (đã bỏ `course=`); thanh trên và bộ chọn lớp chuyển 761988; tab "Chờ duyệt" mở sẵn với 3 yêu cầu (+ D nếu đã gửi); Console `document.cookie.includes('ep_demo_course=int1006-2')` → `true`, `location.search` không còn `course=` (v4: mở thành viên 761987 "30/30") |
| TC-02-51 | AC12 | Như TC-02-50 | **B** Thẻ "Thiết lập lớp mới — 761988": `Mở lớp`, rồi từng bước 1–4 | Mở `/class/members` · `/gradebook/scheme` · `/calendar` · `/documents` **của lớp 761988** (thanh trên 761988; lớp 2 chưa xác nhận công thức, lịch Thứ Ba 07:00–08:30, 9 tài liệu) |
| TC-02-52 | AC12 | GV ở "Tất cả lớp của tôi"; GV đang ở 761988 | **B** Lặp TC-02-50, 51 | Cùng kết quả; từ 761988 bấm thẻ của 761987 (ví dụ `Điểm danh`) → chuyển lại 761987 đúng; `Hoàn tác` / quay lại bằng Back không để lại `?course=` |
| TC-02-53 | AC12 (biên) | GV, SV B, SV A | **B** Mở `/class/members?course=int1006-9`; GV `?course=int1006-2`; B `/?course=int1006-2`; A `/?course=int1006-2` | Id lạ hoặc ngoài quyền (B không thuộc lớp 2) → bỏ qua (giữ lớp hiện tại, không lỗi); GV, A → đổi sang lớp 2 rồi bỏ tham số khỏi URL |
| TC-02-54 | AC12 / SRS 4.9 liên kết sâu | GV | **B** Bấm: "N bài Bài tập 03 cần xem kỹ"; điểm danh buổi 10; "1 câu AI chờ xác nhận"; phiếu cũ nhất | `/grading?filter=review` (chip `Cần xem kỹ` bật sẵn); `/attendance?session=10` (ô Buổi = Buổi 10); `/threads/t-cbc`; `/inbox?ticket=<id phiếu>` (đúng bảng 4.9, mang đủ lớp + đối tượng) |
| TC-02-55 | AC12 / SRS 4.9 | D gửi yêu cầu vào 761988; GV ở lớp 761987 | **B** Mở chuông GV, bấm mục | Mục "Phạm Ngọc Linh xin vào lớp 761988" (Lớp học · vừa gửi) → `/class/members?course=int1006-2&tab=pending` đúng lớp + tab Chờ duyệt, thấy D |
| TC-02-56 | AC12 / SRS 4.9 | GV ở `/class/members` lớp 2 | **B** `Duyệt` D; `Từ chối` một yêu cầu khác | Tạo thông báo cho người nhận (xem US-PROTO-01 TC-01-137, 107); `Hoàn tác` hoạt động (phần TA: xem TC-02-22, chỉ lớp 761987) |
| TC-02-57 | AC13 (#19 E9) | GV ở `/`, sau `Đặt lại dữ liệu` | **B** Thẻ "Câu hỏi của Lý Gia Thảo đã chờ 3 ngày 4 giờ" → `Trả lời` | URL `/inbox?ticket=tk-5`; phiếu Thảo được chọn sẵn, hàng nằm trong khung nhìn, panel chi tiết có "Nhóm em muốn phân tích vụ lộ dữ liệu…" (Console `[data-part=inbox-detail]` `innerText.includes(…)` → `true`); không phải phiếu Nguyễn Minh Trung (v4) |
| TC-02-58 | AC13 | GV, 375 px | **B** Như TC-02-57 | Mở thẳng chi tiết, có `← Hộp thư` ≥ 44 px; `←` về danh sách |
| TC-02-59 | AC13 (biên) | GV | **B** Trả lời / `Nhận` phiếu Thảo rồi quay lại `/`; mở `/inbox?ticket=tk-d3` (D3 sau khi B gửi) | Thẻ phiếu cũ nhất chuyển sang phiếu kế (Đỗ Thanh Long 2 ngày) và liên kết đúng phiếu đó; `?ticket=tk-d3` chọn đúng D3; phiếu "bất kỳ" đều cuộn tới hàng |
| TC-02-60 | AC14 (#19 E19, FR-X13) | GV, mọi route | **A** (SPEC_V51 "02-AC14") 1440 / 390; **C** `tc_02_14` | Sidebar và thanh dưới: `nav-badge-inbox:5`, `nav-badge-grading:4` ở `/`, `/inbox`, `/gradebook`, `/grading`; `nav.ts` không còn `badge: <số>` cứng |
| TC-02-61 | AC14 | Sau `Đặt lại dữ liệu` | **B** `Nhận` một phiếu (không tải lại); sau đó `Duyệt bài` của B | Hộp thư 5 → **4** ngay; Chấm bài 4 → **3** ngay; Hôm nay "3 bài Bài tập 03 cần xem kỹ"; chip lọc cùng số 3 |
| TC-02-62 | AC14 (biên) | GV | **B** `Nhận` hết 5 phiếu `open`; `Duyệt` hết 4 bài cần xem kỹ; "Tất cả lớp của tôi" so với từng lớp | Badge ẩn khi về 0 (không "0"), cả sidebar và thanh dưới; "Tất cả lớp của tôi" = tổng các lớp (Hộp thư 5 + lớp 2; Chấm bài 4 + lớp 2) |
| TC-02-63 | AC14 / N1, N9 | B gửi D3 | **B** GV và TA xem badge | Hộp thư thành 6 (phiếu mới là `open`); TA thấy cùng số; sau GV `Nhận` D3 giảm lại 5 |
| TC-02-64 | AC15 (#19 E7, E8) | GV, `/attendance` buổi 10, 375 và 390 px | **A** (SPEC_V51 "02-AC15"); AUDIT + `TOUCH` | Ô "Buổi" rộng ≥ 160 px hiện "Buổi 10 · 29/10" (`[data-part=att-session-select]`); công tắc "Giả lập mất mạng" ở hàng riêng, không đè ô nào (hình chữ nhật không giao nhau); `{ ox: 0, cut: [], ell: [] }` và `TOUCH` → `[]` |
| TC-02-65 | AC15 | Như TC-02-64 | **A** hàng `[data-part=att-row]`; **B** ảnh | Mỗi SV là hàng 2 dòng: dòng 1 tên (≥ 140 px, tối đa 2 dòng) + `Phát biểu` ≥ 44 × 44; dòng 2 bốn nút "Có mặt · Muộn · Vắng phép · Vắng" mỗi nút ≥ 44 × 44, nằm trọn bề rộng màn, không cuộn ngang; nút đang chọn có dấu tick + chữ đậm (không chỉ màu); đủ 4 trạng thái + Phát biểu của một SV trong **một** màn hình |
| TC-02-66 | AC15 | Như TC-02-64 | **B** Đọc dòng tóm tắt | "30 có mặt · 0 muộn · …" xuống dòng **giữa các cụm**, không ngắt trong cụm ("30 có mặt" không rớt 3 dòng như v4) |
| TC-02-67 | AC15 / 02-AC3 | 375 px, cảm ứng | **B** Bấm giờ: 2 vắng, 1 muộn, +phát biểu B, `Lưu điểm danh` bằng chạm | ≤ 60 s; kết "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; `/me` của B 8,5 |
| TC-02-68 | AC15 (biên) | 375 và 390 | **B** Chọn buổi 9 (đã điểm danh) và buổi 11 (chưa diễn ra) | Ô Buổi vẫn hiện đủ chữ ("Buổi 9 · 22/10", "Buổi 11 · 05/11"); "Buổi này chưa diễn ra" không tràn; layout hàng SV không đổi |
| TC-02-69 | AC16 (#19 E18) | `/attendance` 1440 | **B** Bật "Giả lập mất mạng", đánh vắng 1 SV, tắt công tắc | "Đã lưu hh:mm" và hết dòng "Đang chờ mạng"; Console ở mọi bước `document.body.innerText.includes('· 0 thay đổi')` → `false`; công tắc bật + 0 thay đổi → "Đang giả lập mất mạng" |
| TC-02-70 | AC16 | Sau TC-02-69 | **B** `Lưu điểm danh`; sau đó đổi một ô | "Đã hoàn tất buổi 10 · …" với số đúng thực tế (tổng 30); nút chuyển "Đã lưu" **vô hiệu, trung tính (không đỏ)** tới khi có thay đổi mới; thay đổi mới → nút bật lại, đỏ như cũ |
| TC-02-71 | AC16 (biên) | GV | **B** Lưu với 3 vắng 0 muộn; lưu với 0 vắng; `Hoàn tác` ngay sau đánh dấu | Số trong dòng hoàn tất đúng và cộng thành 30; khi n = 0 bỏ cụm "· 0 thay đổi"; Hoàn tác không để lại "Đang chờ mạng" |
| TC-02-72 | AC17 (#19 E16) | GV, `/threads/t-cbc` | **B** `Chỉnh sửa` → xoá hết; chỉ khoảng trắng; để nguyên | Rỗng / chỉ trắng: `Lưu và xác nhận` khoá (Console `disabled` → `true`) + "Nội dung không được để trống"; không đổi: khoá + "Chưa có thay đổi — dùng Xác nhận nếu nội dung đã đúng" |
| TC-02-73 | AC17 | Sau TC-02-72 | **B** Sửa một chữ → `Lưu và xác nhận`; sửa lại đúng bản gốc | Nút bật khi khác rỗng **và** khác bản đang hiển thị; sau lưu: "Đã được giảng viên sửa & xác nhận" + "Xem câu trả lời AI gốc"; hai nhãn này **chỉ** có khi nội dung khác bản AI gốc (sửa về đúng bản gốc → không "đã sửa") |
| TC-02-74 | AC18 (#19 E12, FR-X13) | GV lớp 761987, `/students` | **A** (SPEC_V51 "02-AC18"); **B** | Chip `Cần chú ý` = **8** = số hàng sau lọc = số hàng cột "Rủi ro" ghi "Cần chú ý" (Console `[data-part=student-row]`); cột Rủi ro không còn "Theo dõi" / "Không" (dùng "–") |
| TC-02-75 | AC18 | GV, `/` | **A**/**B** | "Lớp cần chú ý · 8 sinh viên"; 3 người đầu theo N3 (`high` trước, rồi số buổi vắng giảm dần, rồi N4), mỗi người cũng "Cần chú ý" ở `/students`; `Xem cả 8` → `/students?filter=watch` (chip bật sẵn, 8 hàng) |
| TC-02-76 | AC18 (nhất quán) | GV | **B** So lý do mỗi người ở Hôm nay, `/students` và `/students/sv-3` | Cùng một câu `riskSentence` ("Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu Bài tập 02" cho C); không người nào ở Hôm nay mà ở `/students` ghi "–" |
| TC-02-77 | AC18 | – | **C** `tc_02_18` | `/students` không có "Theo dõi" |
| TC-02-78 | AC19 (#19 E28, FR-X13) | GV lớp 761987 | **A** (SPEC_V51 "02-AC19"); **B** Console của US | `[data-part=student-row]` → `data-student-id` giống hệt nhau ở `/students`, `/attendance`, `/gradebook`, `/class/members`, tăng dần theo `n`; ba người đầu Nguyễn Minh Trung, Trần Thu Uyên, Lê Quang Huy |
| TC-02-79 | AC19 / N4 | GV | **B** `/grading` hàng chờ; sau đó đổi bộ lọc, `Duyệt` một bài | Thứ tự N4 (sau bài đang được lọc ưu tiên); không đổi thứ tự lớp khi lọc / sắp lại bằng tên (`localeCompare`) ở bất kỳ màn nào |
| TC-02-80 | AC20 (#19 E29) | GV, `/` lúc mới vào | **C** `tc_02_20`; **B** | Thẻ "4 bài Bài tập 03 cần xem kỹ" có dòng phụ đúng: "1 lệch hai lượt chấm · 1 bài ngắn bất thường · 1 trùng đoạn với bài khác · 1 AI không chắc ở một tiêu chí" (không còn "Hai lượt chấm lệch…" áp cho cả 4) |
| TC-02-81 | AC20 | GV, `/grading` | **B** Đọc cột lý do từng hàng | B: "Hai lượt chấm lệch 1,5 điểm"; sv-9 bài ngắn; sv-14 trùng đoạn; sv-21 AI không chắc — khớp dòng phụ ở TC-02-80 |
| TC-02-82 | AC20 (nhánh) | Sau TC-02-81 | **B** `Duyệt bài` B; xem Hôm nay, chip lọc, badge | Thẻ "3 bài Bài tập 03 cần xem kỹ"; dòng phụ không còn "lệch hai lượt chấm"; chip và badge Chấm bài cùng 3 (N8: chỉ đếm bài **chưa duyệt** thuộc `Cần xem kỹ`) |
| TC-02-83 | AC21 (PM chốt J3) | GV, sau `Đặt lại dữ liệu`; B tạo thread thứ nhất khớp mẫu | **B** (+ **T** `T1j…o`) GV mở `/` | "7 việc cần xử lý hôm nay"; thẻ "Câu hỏi mới: «<tiêu đề>»" nằm **ngay trên** thẻ "1 câu trả lời của AI chờ bạn xác nhận" (vẫn ghi 1, của `t-cbc`, không đếm thread mới) và **dưới** thẻ "4 bài Bài tập 03 cần xem kỹ"; thứ tự `data-task-id`: ticket, điểm danh, review, `thread-new-1`, `ai-pending`, setup, members |
| TC-02-84 | AC21 | Sau TC-02-83, B tạo thread thứ hai | **B** GV mở `/`; `Xác nhận` câu AI của thread thứ hai | "8 việc" với thread thứ hai đứng **trên** thread thứ nhất (`thread-new-2`, `thread-new-1`); sau Xác nhận thread hai: "7 việc", thẻ thread hai biến mất, thread một còn nguyên |
| TC-02-85 | AC21 / J3 (nhánh không khớp) | B tạo thread WPA3 | **B** GV mở `/`; GV gửi phản hồi | Thẻ ghi `Cần giảng viên trả lời`; chỉ rời khi GV / TA **gửi phản hồi** (Xác nhận / Loại không áp dụng vì không có câu AI chính) |
| TC-02-86 | AC21 / N10 | B bấm `Hỏi trợ lý AI` có chữ ở `t-salt` (tạo bài AI `Chờ xác nhận`) | **B** GV mở `/` | Thẻ "N câu trả lời của AI chờ bạn xác nhận" N = 2 (loại thread đã có việc "Câu hỏi mới"); bấm → `/threads?filter=pending`; N = 1 → `/threads/t-cbc`; TA thấy giống GV |
| TC-02-87 | AC21 / N10 | SV B / GV | **B** `/threads` chip "chờ xác nhận" | Chip đếm **mọi** thread có câu AI chờ (kể cả thread mới của B); `t-rsa-key` không còn ở trạng thái chờ (đã xác nhận, v5.1) |
| TC-02-88 | 4.9 / FR-X17 | B gửi D3 (ticket mới) | **B** Đổi vai GV, rồi TA; mở chuông | Cả hai thấy "1 câu hỏi mới cần xử lý" (Hộp thư hỗ trợ · vừa gửi) có chấm; bấm → `/inbox?ticket=tk-d3` chọn sẵn D3; đã đọc thì chấm giảm; GV `Gửi trả lời` → B nhận thông báo (US-PROTO-01 TC-01-138) |
| TC-02-89 | N1 (SRS 4.8) | GV | **B** `/inbox`: đếm nhãn "Quá 24 giờ" trước và sau `Nhận` phiếu 1 ngày | 3 hàng có nhãn (1 ngày, 2 ngày, 3 ngày 4 giờ); sau `Nhận` phiếu quá 24 giờ còn 2 (khớp "Câu chờ quá 24 giờ" ở `/analytics`, US-PROTO-04 TC-04-35); nhãn tính từ tuổi ≥ 1440 phút, không ghi cứng |
| TC-02-90 | AC9 (#24a, biên) | GV; `/inbox` 1440, 1280, 1100, 1099, 900, 720, 390, 375 | **A** `regress-v24.mjs` (TC-02-90) + **B** ảnh | Hàng tab "Đang chờ / Đã nhận / Đã trả lời / Tất cả" trong panel danh sách hiện **đủ chữ ở mọi bề rộng**: phần nhìn thấy (sau khi trừ vùng bị cha cắt) cao ≥ 28 px, không co thành vạch mỏng; chữ không bị cắt |
| TC-02-91 | AC15 (#24d, biên) | GV; `/attendance` 390, 375 | **A** `regress-v24.mjs` (TC-02-91) + **B** ảnh | Ô chọn buổi hiện nhãn ngắn "Buổi 10 · 29/10" **không bị cắt** (độ rộng chữ ≤ chỗ trống của ô); trạng thái ("đang diễn ra") ở dòng phụ ngoài ô; không cuộn ngang |



## Công cụ bổ sung (spec v5, v5.1)
**A** = `scripts/audit.mjs` (phép đo "02-AC9", "02-AC10", v5.1: "02-AC14", "02-AC15", "02-AC18", "02-AC19", `TOUCH` ở `/attendance`, `/inbox`); **T** = `scripts/threads-timeline.mjs` (các bước `T1j`…`T1o`, `T4e`…`T4g`); **C** v5.1: `tc_02_14`, `tc_02_18`, `tc_02_20`.

## Nhánh lỗi
TC-02-07 (409 giả lập), TC-02-18 (mất mạng), TC-02-19, TC-02-26 (`?state=`), TC-02-11 (hoàn tác); spec v5: TC-02-36, 37 (`?ticket=` sai / tải lại), 45 (nhánh không khớp), 48 (đếm khi nhiều thread).

## Phân quyền
TC-02-20, 21 (SV/Admin/TA); TC-02-25 (GV không đọc chat riêng chưa escalate — luật CLAUDE.md); TC-02-23 (ghi chú chỉ GV/TA).

## Kiểm chéo
- Bàn phím và focus: TC-02-09 chỉ dùng bàn phím; kiểm Tab/Shift+Tab, focus thấy rõ trên `/attendance` và Dialog.
- Phản mẫu §21: `/` không tường KPI (TC-02-03); `/attendance` không card (TC-02-17); ≤ 1 nút chính đỏ mỗi vùng.
- 375 px: `/inbox`, `/attendance` bắt buộc; các route khác 390 px không vỡ.

## Điểm khó kiểm
- "Toàn bộ ≤ 60 s" phụ thuộc người bấm: ghi thời gian thực tế trong report (bấm giờ bằng script `demo-run.mjs` khi có).
- Phụ thuộc chéo: TC-02-05 cần D3 (US-PROTO-01); TC-02-13/14 cần yêu cầu của D; TC-02-43…49 cần thread do B tạo (US-PROTO-01 TC-01-39 / TC-01-79).
- AC9 / AC10 đo ở trình duyệt thật (Console); `data-part` `inbox-list`, `inbox-detail`, `inbox-reply` do dev thêm — thiếu móc = FAIL.
- Thứ tự việc "Câu hỏi mới" trong Hôm nay: **đã chốt ở v5.1 (SRS 4.3.1 J3, 02-AC21)** — TC-02-83…86 kiểm đúng thứ tự và cách đếm; TC-02-43…49 (v5) vẫn đúng.
- 02-AC12: spec ghi `?tab=pending` sau khi bỏ `course=`; TC-02-50 chỉ đòi tab Chờ duyệt mở sẵn, không đòi URL giữ nguyên chuỗi `tab=pending` nếu dev đổi nơi lưu tab.
- 02-AC14: "Tất cả lớp của tôi = tổng" không nêu số lớp 2 (seed không có phiếu / bài chấm lớp 2 theo 4.1) — TC-02-62 so với tổng các lớp thật, không so số cứng.
- 02-AC16: spec không nêu mẫu chữ khi 3 vắng / 0 muộn (có bỏ cụm "0 muộn" hay không); TC-02-71 chỉ đòi tổng = 30 và không "· 0 thay đổi".

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-01 · TC-02-16..18: Cập nhật kiểm tra cấu trúc 2 panel độc lập (viền, nền surface, bo góc, cuộn độc lập) của `/inbox` (02-AC5); thêm TC-02-28..31 kiểm duyệt câu trả lời AI ở Threads (Xác nhận, Chỉnh sửa inline lưu CORRECTED, Loại bỏ có Hoàn tác) theo spec v4 (02-AC8, proposals #15, #16).
- 2026-10-02 · Thêm TC-02-32…49 (không sửa TC cũ): spec v5 — 02-AC9 (#17c `/inbox` danh sách + mobile `?ticket=`), 02-AC10 (#17d khoảng trống chi tiết), 02-AC11 (#18 việc "Câu hỏi mới" ở Hôm nay, 6 ↔ 7).
- 2026-10-02 · Thêm TC-02-50…89: spec v5.1 (#19) — 02-AC12 (E3, FR-X17), 02-AC13 (E9), 02-AC14 (E19, FR-X13), 02-AC15 (E7, E8), 02-AC16 (E18), 02-AC17 (E16), 02-AC18 (E12), 02-AC19 (E28), 02-AC20 (E29), 02-AC21 (J3), cộng SRS 4.8 N1, N3, N4, N8, N9, N10 và 4.9 (bảng liên kết sâu + thông báo).
- 2026-10-02 · **Sửa TC-02-03, TC-02-08, TC-02-23:** "Lớp cần chú ý (C + 2 SV)" / chip `Cần chú ý` = C + 2 → N = 8 (SRS 4.8 N3); thêm nhãn "Quá 24 giờ" (N1). Lý do: spec v5.1 đổi.
- 2026-10-02 · **Thêm TC-02-90, 91 (không sửa TC cũ):** góp ý #24a (`/inbox` 1440 tab co thành vạch, vi phạm 02-AC9) và #24d (`/attendance` 390 ô chọn buổi cắt "đang diễn…", biên 02-AC15) — PM, ACCEPTED 01/10. `scripts/regress-v24.mjs`.
- 2026-10-02 · **Sửa TC-02-22, TC-02-56 (proposals #25, ACCEPTED):** TA chỉ phụ trách lớp 761987; bỏ yêu cầu TA duyệt ở lớp 761988.

