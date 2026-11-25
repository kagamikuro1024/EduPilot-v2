# Gate P10 — LƯỢT 2: chạy lại DEMO_SCRIPT.md trên demo dựng lại (main b12d70b9)

Người chạy: QC cặp 3 · 25/11/2026 06:48–07:12 (giờ VN) · stack demo `edupilot` dựng lại từ `main` b12d70b9 (gateway 06:46, frontend image a156331f91a1 06:52 — lúc đầu lượt FE vẫn là bản cũ 02:21, đã báo PM, PM dựng lại giữa lượt; mọi mục FE dưới đây chạy trên bản mới) · DB MỚI seed embedding thật (qwen3), chat/chấm LLM giả · kịch bản: `DEMO_SCRIPT.md` @origin/main (BA 8ab42c30, P-11-112/114).

Cách chạy: như lượt 1 — 5 Chromium headless riêng (Admin, Giảng viên, SV B `sv.kha` 375, SV D `sv.moi` 375, SV X 375 cho `sv.nguyco`/`sv04`/`sv05`); camera không dùng được ⇒ «quét» = mở đúng `qr_url` lấy từ API `/checkin-windows/{id}/display` (tuổi ≤ 10 s). Không seed / reset / lệnh admin ghi dữ liệu; DB chỉ đọc. Bằng chứng: `evidence-gate-p10-r2/` (ảnh + `rec.json`).

## L2.1 Kết quả

**51 mục chấm: 44 P · 3 F · 4 N.** Lượt 1: 39 P · 9 F · 4 N.

| Nhóm | P | F | N | Ghi chú |
| --- | --- | --- | --- | --- |
| Bước 1 | 4 | 0 | 0 | đạt |
| Bước 2 | 3 | 0 | 0 | đạt; D1 «Chuyên cần»/«Phát biểu» (GP10-01 ĐÓNG) |
| Bước 3 | 6 | 0 | 0 | đạt; chuông ở /chat (GP10-03 ĐÓNG); D3 chậm 25,6 s (embedding thật) |
| Bước 4 | 5 | 1 | 1 | 4.1 N: buổi 10 hôm nay 09:00 chưa bắt đầu; 4.6 F: /me không có hàng «Điểm danh lúc» cho buổi hôm nay có mặt; 4.2b GP10-02 5/5 khung ĐÓNG |
| Bước 5 | 5 | 1 | 0 | 5.1 F: «Đang chấm» không có dấu thời gian nộp |
| Bước 6 | 4 | 0 | 1 | 6.5 N: BT03 chưa gắn cột điểm (seed) |
| Bước 7 | 2 | 1 | 0 | ĐỎ→XANH một phần: chủ đề đúng tài liệu, GP10-05 đạt; 7.2 F: câu mẫu chỉ 1 (seed trùng) |
| Bước 8 | 2 | 0 | 0 | đạt |
| Hội đồng hỏi | 6 | 0 | 2 | H5 N (BT04 chưa quá hạn), H7 N (trần 3 báo cáo/giờ…) |
| Còn giờ | 7 | 0 | 0 | đạt |

**Dò lại các lỗi đã sửa (PM yêu cầu):**

| Lỗi | Kết quả | Chứng cứ |
| --- | --- | --- |
| GP10-01 (D1 khối «Chuyên cần»/«Phát biểu») | **ĐÓNG — P** | mục 2.1: Vắng 2 · Có mặt 6 · Muộn 1 · Có phép 0 · «Buổi vắng gần đây» 28/10, 07/10; «Số lần phát biểu» 2 · «Điểm cộng» 0,50 (khớp sổ); không khoá tiếng Anh/JSON, không tiêu đề «Kết quả»; 375 và 1440 px: `b2-1-d1-block-375.png`, `b2-1-d1-block-1440.png` |
| GP10-02 / TC-AS08-30 (P-11-113) | **ĐÓNG — P** | mục 4.2b: 1024/1366/1440/1920/375 đều có mã ngắn NGAY DƯỚI QR, trong bề ngang QR, scrollW ≤ innerWidth, scrollH ≤ innerHeight, 0 phần tử tràn; `b4-4-display-{1024,1366,1440,1920,375}.png` |
| GP10-03 (chuông ở /chat) | **ĐÓNG — P** | mục 3.5: bấm «Giảng viên đã trả lời…» khi đang ở /chat → `?session=…`, phiên hiện đủ trong 0,4 s, nhãn «Giảng viên · TS. Lê Thu Hà» |
| GP10-05 (Bước 7 kỳ 30 ngày) | **ĐÓNG — P** | mục 7.1: #1 «Cảnh báo học vụ và bảo lưu» (Quy chế học vụ), #2 «Cơ chế hoạt động của ransomware» (Network Security Threats · Tuần 3), honeypot/firewall cùng chương; «Thời tiết Hà Nội hôm nay» KHÔNG có chương/tuần |
| (GP10-04 đã rút) | — | một lần bấm «Mở điểm danh» vào thẳng màn chiếu CÙNG tab (AC1) |

## L2.2 Thời gian

Chỉ đo được độ trễ hệ thống; lời dẫn người trình bày không đo được ⇒ không kết luận được «≤ 15 phút». Toàn bộ lượt chạy tự động (gồm chuẩn bị #3, chụp ảnh, đọc DB) ≈ 24 phút đồng hồ — không phải thời gian trình bày.

| Việc | Đo được |
| --- | --- |
| Chuẩn bị #3 «Chấm tất cả» BT03 (14 bài chưa chấm; 8 bài đã có) | xong < 5,8 phút (lượt 1: 22 bài ≈ 12–13 phút) |
| D1 (có «Chuyên cần») | 1,0 s |
| D2 | 0,9 s |
| D3 → «AI chưa đủ chắc chắn» + «Đang chờ giảng viên» | **25,6 s** (lượt 1: 2,1 s) — embedding thật qua mạng ~20 s ở bước truy xuất |
| Chuông → phiên chat có câu trả lời GV | 0,4 s |
| Mở điểm danh → màn chiếu | ≈ 3 s |
| Đóng → Đánh vắng → Muộn → +0,25 (4.4) | ≈ 28–42 s (gồm độ trễ công cụ) < 60 s |
| Lưu điểm danh | 2,6 s |
| Tải quy chế → bản nháp công thức | 4,4 s |
| Chọn «30 ngày» → danh sách chủ đề | tức thì (báo cáo sinh sẵn) |

## L2.3 Chuẩn bị — trạng thái demo lúc bắt đầu

| # | Việc | Thực tế |
| --- | --- | --- |
| 3 | Bài chấm nháp BT03 | 8 bài đã duyệt (seed), 14 chưa chấm → QC bấm «Chấm tất cả» bằng giao diện (đã chọn Bài tập 03 trước; không chọn thì nút khoá «Chọn một bài tập trước.»); xong < 5,8 phút; 11 bài «Cần xem kỹ», SV B «Hai lượt chấm lệch hơn 1 điểm» ở vị trí 11/11 |
| 4 | Báo cáo lỗ hổng | PM đã sinh sẵn: 761987 (06:41, 30 ngày, 79 câu · 20 SV · 32 cụm) và 761988 (06:39, 43 câu · 20 SV); QC KHÔNG bấm «Tạo báo cáo mới» |
| 5 | Điểm danh bằng mã | công tắc 761987 đang TẮT → QC bật bằng UI (`/class/settings`). Có Buổi 10 = HÔM NAY 25/11 09:00–11:30 (seed theo ngày chạy) ⇒ khác lượt 1; «Mở điểm danh» dùng được từ 07:10 (trước giờ học), người quét ra «Có mặt», không «Muộn» |
| 8 | Mailpit hộp thư trống | còn 36 thư cũ: KHÔNG xoá (dữ liệu PM) |

## L2.4 Kết quả chi tiết từng mục

| Mục | Việc | Kết quả | Ghi chú | Ảnh (`evidence-gate-p10-r2/`) |
| --- | --- | --- | --- | --- |
| 1.1 | Admin /admin/courses: hai lớp, đã gán giảng viên | P | 2 lớp 761987, 761988 hiển thị: 761988	INT1006 / 	TS. Lê Thu Hà	24 / 30	 / 761987	INT1006 / 	TS. Lê Thu Hà	30	 | `b1-1-admin-courses.png` |
| 1.2 | Teacher chuông: phân công lớp + mã BX4P9TW; «Thiết lập lớp mới» ghi tên lớp 2 | P | Chuông có «Bạn được phân công lớp An ninh mạng – 761988» và «…761987»; Hôm nay (lớp 761988) có «Thiết lập lớp mới» 3/4 bước. LỆCH CHỮ nhỏ (như lượt 1): dòng chuông không kèm mã BX4P9TW; bộ chọn lớp GV mở ở lớp gần nhất (761988 lúc đầu), kịch bản ghi 761987 | `b1-2-teacher-bell.png` |
| 1.3 | SV D (375) /join/BX4P9TW: xem trước + Tham gia lớp → chờ duyệt | P | Xem trước: «An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027», «Lớp này cần giảng viên duyệt.»; sau bấm: Đã gửi yêu cầu, chờ giảng viên duyệt. | `b1-3-svd-after.png` |
| 1.4 | Teacher /class/members: duyệt SV D; 3 yêu cầu khác vẫn chờ | P | Tab «Chờ duyệt» 4 (sv.moi + sv52–54); sau «Duyệt» sv.moi: còn 3 yêu cầu chờ (sv54@edupilot.local, sv53@edupilot.local, sv52@edupilot.local) | `b1-4-after-approve.png` |
| 1.x | Còn giờ B1: SV D tải lại / → bộ chọn lớp có 761988; không còn ô nhập mã | P | Đầu trang bộ chọn lớp hiện «761988 ; Buổi 4 · An ninh mạng ; Buổi 5 · An ninh mạng» (lớp 761988 sau khi được duyệt); ô nhập mã tham gia: không còn; «Chưa có lớp» biến mất | `x-b1-svd-reload.png` |
| 2.1 | SV B (375 + 1440) D1 — dò GP10-01 trên FE mới | P | «Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI»; khối «Chuyên cần»: Vắng 2 · Có mặt 6 · Muộn 1 · Có phép 0 · «Buổi vắng gần đây» 28/10, 07/10; khối «Phát biểu»: Số lần phát biểu 2 · Điểm cộng 0,50 (số khớp sổ). Không còn khoá tiếng Anh/JSON/tiêu đề «Kết quả»; không có [[SV_1]]; D1 phản hồi 1,0 s | `b2-1-d1-block-375.png` |
| 2.2 | SV B bấm Hữu ích | P | Nút đổi tại chỗ → «Đã ghi nhận · Hoàn tác» (dòng tĩnh), không toast nổi | `b2-2-helpful.png` |
| 2.3 | SV B gõ D2 (hỏi về SV C) | P | «Mình chỉ xem được dữ liệu của chính bạn.» — không có con số nào về SV C; 0,9 s | `b2-3-d2.png` |
| 3.1 | SV B gõ D3 → «AI chưa đủ chắc chắn» + «Đang chờ giảng viên · vừa gửi» | P | Có «AI chưa đủ chắc chắn về câu này» và «Đang chờ giảng viên · vừa gửi». QUAN SÁT: từ gửi tới thấy kết quả 25,6 s (lượt 1: 2,1 s) — embedding thật qwen3 qua mạng ~20 s ở bước truy xuất; câu dẫn của LLM giả ghi «Mình đã tra sổ của bạn…» (giả, không tính) | `b3-1-d3.png` |
| 3.2 | GV mở ticket mới nhất ở lọc Open: câu hỏi gốc, thời gian chờ, lý do | P | Hàng chờ có 3 ticket; mới nhất «Bùi Thanh Khải · 0 phút · Thi cuối kỳ…» trạng thái «Đang chờ», độ tin cậy 0,16; có nút Nhận / Đóng không trả lời | `b3-3-ticket-open.png` |
| 3.3 | GV bấm Nhận | P | «Đã nhận · 0 phút»; không hiện tên người nhận (đúng P-11-112); có Gửi trả lời / Bỏ nhận / Đóng không trả lời. Hai người cùng Nhận → 409: N (chỉ 1 GV) | `b3-4-claimed.png` |
| 3.4 | GV gõ D4 → Gửi trả lời → Answered | P | Sau «Gửi trả lời» ticket chuyển «Đã trả lời»: Đã trả lời / Đã trả lời · 0 phút | `b3-5-answered.png` |
| 3.5 | SV B mở chuông ở /chat → câu trả lời GV trong phiên chat (dò GP10-03) | P | FE mới (a156331f): bấm «Giảng viên đã trả lời…» khi đang ở /chat → URL ?session=…, phiên hiện đủ trong 374 ms, có câu trả lời của GV; không còn khung trống. Nhãn người trả lời: xem dòng sau | `b3-6b-after-bell.png` |
| 3.6 | Mailpit: thư mới nhất tới sv.kha chứa câu trả lời + link về app | P | Thư «Giảng viên đã trả lời câu hỏi của bạn» tới sv.kha@edupilot.local: câu hỏi gốc, «Giảng viên TS. Lê Thu Hà trả lời: …», link https://localhost/chat?session=…&m=… . Lưu ý: Mailpit còn 36 thư cũ (chuẩn bị #8 «hộp thư trống» không làm: không xoá dữ liệu PM) | `b3-7-mailpit-open.png` |
| 4.1 | GV chạm việc «Điểm danh buổi đang diễn ra – 761987» ở Hôm nay | N | Chưa kiểm được: lúc 07:12 buổi 10 (25/11 09:00–11:30) chưa bắt đầu nên «Hôm nay» chưa có việc điểm danh, chỉ có «Sắp tới: Buổi 10 · Hôm nay 09:00 · P.302». Đã chạy Bước 4 sớm (07:10) nên không còn kiểm lại được trong buổi; cần chạy trong 09:00–11:30 trên DB chưa điểm danh buổi 10 | `b4-1-today.png` |
| 4.2 | GV /attendance Buổi 10 · Hôm nay: Mở điểm danh → màn chiếu /attendance/display | P | 1 lần bấm «Mở điểm danh» (Buổi 10 · 25/11 09:00–11:30; bấm lúc 07:10, trước giờ học vẫn mở được) → vào /attendance/display CÙNG tab (AC1, GP10-04 đã rút): QR, mã ngắn «MXA 1K2» NGAY DƯỚI QR, số đã điểm danh «0», «còn 09:45», Đóng điểm danh / Thoát. QUAN SÁT: số «0» không có nhãn chữ «đã điểm danh» | `b4-4-display-1440.png` |
| 4.2b | GP10-02 / TC-AS08-30 (P-11-113): mã ngắn NGAY DƯỚI QR, trong bề ngang QR, không cuộn — 1024/1366/1440/1920/375 | P | 1024×768: QR 32,43,576,587; mã 109,644,500,725 (dưới QR true, trong bề ngang true); scrollW=1024≤1024, scrollH=768≤768 / 1366×768: QR 32,43,576,587; mã 109,644,500,725 (dưới QR true, trong bề ngang true); scrollW=1366≤1366, scrollH=768≤768 / 1440×900: QR 32,43,684,696; mã 124,759,592,857 (dưới QR true, trong bề ngang true); scrollW=1440≤1440, scrollH=900≤900 / 1920×1080: QR 32,44,831,844; mã 145,916,719,1036 (dưới QR true, trong bề ngang true); scrollW=1920≤1920, scrollH=1080≤1080 / 375×812: QR 32,70,357,395; mã 78,433,311,482 (dưới QR true, trong bề ngang true); scrollW=375≤375, scrollH=812≤812. 5/5 khung đạt; nút Đóng điểm danh/Thoát trong khung, 0 phần tử tràn | `b4-4-display-375.png` |
| 4.3 | 2 SV quét QR (mở qr_url từ màn chiếu, ≤10 s tuổi): «Đã điểm danh hh:mm»; số trên màn chiếu tăng | P | Camera hệ thống không dùng được (headless): mở đúng qr_url «/checkin?c=…#<window>.<…>» lấy từ API display (tuổi 8 s và 1 s) cho sv.kha (B) và sv.nguyco: mỗi người thấy «Điểm danh · Đã điểm danh 07:05», không có nút nào; màn chiếu 0 → 2 sau ≤ 10 s | `b4-5-scan-sv.kha.png` |
| 4.4 | GV (375) /attendance: Đóng; Đánh vắng chưa quét; Muộn cho 1; phát biểu +0,25 cho SV B (<60 s) | P | Sau «Đóng»: «Đã ghi 4 có mặt · 0 muộn · 26 chưa điểm danh» (buổi 10, chưa tới 09:00 vẫn «Có mặt»); «Cần xem (3)» = 3 SV «Cùng máy với bạn khác» (sv.nguyco, sv04, sv05) + chip «Nhập mã» (sv04, sv05); nút «Đánh vắng 26 SV chưa điểm danh» → hộp «Đánh vắng 26 sinh viên?» → 26 dòng «Vắng»; chạm tên Bùi Khánh Việt → radio Có mặt/Muộn/Vắng phép/Vắng → «Muộn»; «Ghi phát biểu · Bùi Thanh Khải» → dòng «+1 · +0,25×1»; mỗi SV quét có «Điểm danh lúc hh:mm»; sinh viên chưa quét «Chưa điểm danh». Toàn bộ thao tác ≈ 42 s (<60 s ✔) | `b4-6-after-marks.png` |
| 4.5 | GV Lưu điểm danh: xác nhận tại chỗ, không toast «Thành công» | P | Đã lưu 07:08 / Đã lưu điểm danh · 30 sinh viên · vắng 25; Buổi 10 trong chọn buổi: Buổi 10 · 25/11 · Đã điểm danh; toast: «Đã lưu điểm danh · 30 sinh viên · vắng 25» | `b4-7-saved.png` |
| 4.6 | SV B (375) /me: hàng buổi hôm nay «Điểm danh lúc hh:mm»; chuyên cần + điểm cộng thêm +0,25; giải trình phép tính | F | ĐẠT: «Điểm cộng 0,75 · tối đa 1» (0,50 → 0,75), «Phát biểu 3 lần · +0,75 · 25/11 ×3», giải trình phép tính («Điểm trừ 0 · vắng 2 buổi, được miễn 2, mỗi buổi trừ 0,5», «7,75 → 8,5»), trang cao 1 516 px ≤ 1 600. KHÔNG ĐẠT: không có hàng «Buổi 10 · 25/11 … Điểm danh lúc 07:05»: RealMe.tsx chỉ liệt kê buổi KHÁC «Có mặt» (Buổi 6, 4, 3), hàng «Điểm danh lúc» chỉ hiện cho buổi Muộn/Vắng có lượt quét; buổi hôm nay «Có mặt» chỉ nằm trong câu «có mặt 8/10 buổi». Lệch kịch bản/SRS 411 vs UI — Thấp | `b4-8-me-375.png` |
| 5.1 | SV B (375) mở Bài tập 03: đã nộp, dấu thời gian, nhãn nộp muộn, đang chấm, không điểm nháp | F | Có «Đang chấm · Nộp muộn 1 ngày» và KHÔNG có điểm nháp (✔) nhưng KHÔNG có dấu thời gian nộp: khi bài ở trạng thái GRADING (sau chuẩn bị #3) StudentAssignment.tsx chỉ in «Đang chấm{lateTag}», dấu thời gian chỉ có ở trạng thái SUBMITTED («Đã nộp CN 08/11 · 20:04…», lượt 1). Lệch kịch bản vs UI — Thấp; PM chốt sửa chữ kịch bản hay UI | `b5-1-assignment-sv-submission.png` |
| 5.2 | GV /grading Hàng chờ, lọc mặc định «Cần xem kỹ»+«Chưa duyệt»: bài SV B + lý do cờ | P | 11 bài; dòng Bùi Thanh Khải: «Hai lượt chấm lệch hơn 1 điểm», 6,50, Chưa duyệt — ở vị trí 11/11 (xếp theo giờ nộp). Kịch bản mới (#3) đã ghi «không nhất thiết ở đầu danh sách» ⇒ đạt | `b5-2-queue.png` |
| 5.3 | GV /grading/[id]: đọc bài + 4 tiêu chí; sửa điểm lệch + 1 nhận xét; tổng tính lại ngay | P | Trái: bài nộp; phải: 4 tiêu chí mỗi tiêu chí có đoạn trích “…”; cảnh báo ở «Phân tích tấn công»: «Lệch 1,5 điểm so với lượt 2 (1,00 và 2,50)»; sửa điểm 1,00→2,00 + nhận xét: Tổng điểm 6,50 → 7,50 ngay (đã trừ nộp muộn 1 ngày) | `b5-3-edited.png` |
| 5.4 | GV bấm Duyệt bài → Đã duyệt | P | Đã duyệt · / Đã duyệt 1/11 / Đã duyệt | `b5-4-approved.png` |
| 5.5 | GV chọn bài vừa duyệt → Công bố | P | Hộp xác nhận «Công bố 1 bài · 1 sinh viên nhận thư · không tính vào sổ điểm (bài chưa gắn vào cột điểm)» → «Công bố»; lọc «Đã công bố» có dòng Bùi Thanh Khải. SEED: BT03 chưa gắn cột điểm ⇒ điểm không vào sổ (ảnh hưởng 6.5). Chỉ GV thấy hành động: kiểm riêng ở TA/SV: N | `b5-5-published.png` |
| 5.6 | SV B tải lại /assignments/[id]: điểm, nhận xét theo tiêu chí, đoạn trích | P | «Kết quả 7,50 / 10,00»; 4 tiêu chí «2,00 / 2,50» + nhận xét (kể cả câu GV sửa «Cần thêm bằng chứng về kỹ thuật khai thác khoá SAML.») + đoạn trích từ bài của chính SV | `b5-6-sv-result.png` |
| 6.1 | GV /gradebook lớp 2: banner công thức chưa xác nhận; Chốt điểm khoá + lý do | P | Banner «Công thức điểm chưa xác nhận»; «Chốt điểm» disabled, mô tả/tooltip «Chưa có công thức điểm đã xác nhận» | `b6-1-gradebook-lop2.png` |
| 6.2 | GV tải quy-che-lop2.pdf ở /gradebook/scheme lớp 2 | P | Tiến độ xử lý nền ≈ 4.353 s → «Bản nháp»: Giữa kỳ 25% / Bài thực hành 25% / Thi cuối kỳ 50% (Tổng 100%), điểm cộng +0,25/lần phát biểu (tối đa +1,5), điểm trừ −0,5/buổi vắng (miễn 1), mỗi mục kèm «Trang n»; «Còn 1 mục chưa rõ: làm tròn» (chặn xác nhận) | `b6-2-draft.png` |
| 6.3 | GV điền D5 (làm tròn 0,1) → Xác nhận công thức → hộp thoại hậu quả → xác nhận | P | Chọn «Làm tròn đến 0,1» → nút mở khoá; hộp thoại «Áp cho 25 sinh viên · điểm học phần của 0 sinh viên sẽ đổi … · 0 điểm đã nhập được giữ · … bị bỏ»; sau xác nhận: Đã xác nhận · Hôm nay, 06:57 | `b6-3-confirmed.png` |
| 6.4 | GV /gradebook lớp 2 sau xác nhận: banner biến mất, Chốt điểm mở | P | Banner «chưa xác nhận»: false; Chốt điểm states=[] | `b6-4-gradebook-lop2-after.png` |
| 6.5 | GV /gradebook lớp 1: giải trình SV B có +0,25 (bước 4) và điểm BT03 (bước 5); dòng «điểm chính thức…» | N | ĐẠT: «Điểm cộng min(0,75; 1) = 0,75» (đã gồm +0,25 từ Bước 4; trước Bước 4 là 0,5), «Điểm trừ», «Điểm quá trình 7,75 + 0,75 → 8,5», «Điểm học phần thô 7,3 → làm tròn 0,5 = 7,5», chữ B; có «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.». KHÔNG KIỂM ĐƯỢC: điểm Bài tập 03 (7,50) không là thành phần nào vì BT03 chưa gắn vào cột điểm (dữ liệu seed; hộp công bố đã báo «không tính vào sổ điểm») | `b6-5-explain-after4.png` |
| 7.1 | GV /insights 761987 kỳ «30 ngày»: danh sách «Chủ đề nên dạy lại» + GP10-05 | P | Báo cáo 25/11 06:41 · 79 câu · 20 SV · 32 cụm. Top: #1 «Cảnh báo học vụ và bảo lưu» (Quy chế học vụ; 6 SV · 7 câu), #2 «Cơ chế hoạt động của ransomware» (Network Security Threats · Tuần 3; 5 SV · 6 câu), #3 «Thời tiết Hà Nội hôm nay» KHÔNG có chương/tuần ✔ (GP10-05). Mỗi chủ đề có «Nên làm». QUAN SÁT (không chặn): #4 và #8 tên «Nhóm chưa có câu hỏi» (cụm 0 câu mẫu: #4 cong/gom/nau = nhóm lạc đề «công thức nấu…», 2 SV «Ít người hỏi»; #8 3 SV gắn «Quy chế học vụ») HIỆN TRÊN MÀN; từ hạng 9 trở đi tên là từ khoá («cross, scripting, site»); hạng 3 là chủ đề lạc đề «Thời tiết» đứng trên chủ đề ôn thật; 3 cụm 1-SV gắn chương «Forecasting (QMB ch. 6b)» | `b7-1-insights-30.png` |
| 7.2 | GV mở «Câu hỏi mẫu» của chủ đề ransomware: 3–5 câu ẩn danh, không tên/MSSV; chủ đề «Ít người hỏi» không có mẫu | F | Câu hỏi mẫu của «Cơ chế hoạt động của ransomware» chỉ có 1 câu («Ransomware hoạt động như thế nào?»), không phải 3–5: seed có 5 câu giống hệt (distinct = 1; chat_messages: «Ransomware hoạt động như thế nào?» ×5) + 1 câu khác không được chọn. Ẩn danh ✔ (không tên/MSSV); chủ đề «Ít người hỏi» (#4, <3 SV) không có mẫu và không «Câu hỏi mẫu» ✔. Nguyên nhân dữ liệu seed — Thấp; PM chốt sửa seed hay chữ «3–5 câu» | `b7-2-examples-ransomware.png` |
| 7.3 | GV «Tạo thread ghim» chủ đề ransomware → form tại chỗ → Đăng | P | Form mở TẠI CHỖ dưới chủ đề (Tiêu đề «Chủ đề đang vướng: Cơ chế hoạt động của ransomware», Nội dung điền sẵn, Đăng/Huỷ), không hộp thoại; sau «Đăng» thread ghim xuất hiện ở /threads: true | `b7-3-posted.png` |
| 8.1 | Admin /admin/audit (24 giờ): dòng vừa tạo ở Bước 4–6 với thời điểm, người làm kèm vai, thực thể, hành động, khoá đổi | P | Có: attendance (complete / save / absent_unchecked), participation create, checkin_window.open/close, grading_results (grading_result.edit/review/publish), grade_schemes (extract/edit/confirm) — mỗi dòng giờ, «TS. Lê Thu Hà · Giảng viên», thực thể, hành động, khoá đổi (changes, counts, points, …); không giá trị cũ/mới. Ngoài ra gradebook.export, escalation_ticket.*, audit_log VIEW_LIST (chính việc mở nhật ký) | `b8-1-audit-final.png` |
| 8.2 | Admin /admin/audit: ô «Thực thể» gõ grade_schemes → chỉ dòng công thức của Bước 6 | P | 6 dòng, toàn bộ grade_schemes extract/edit/confirm (06:57 25-11, giảng viên); kịch bản đã sửa tên thực thể (lượt 1 là F) | `b8-2-audit-filter.png` |
| H1 | Hội đồng hỏi B1: nhập sai mã tham gia | P | sv.nguyco vào /join/ZZ2K4MQ: 5 lần liên tiếp cùng một thông báo «Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.»; lần 6: «Bạn đã thử quá nhiều lần. Thử lại sau 09:46» (429 RATE_LIMITED, retry_after ≈ 580 s, Redis TTL 580 s = 10 phút; chữ «09:46» là đếm ngược mm:ss, dễ đọc nhầm thành giờ — Thấp). Mã tạo lại → mã cũ bị từ chối: không kiểm (không đổi mã lớp trên demo) | `h1-wrong-code.png` |
| H3 | Hội đồng hỏi B3: ticket chờ quá 72 giờ nổi lên Hôm nay | P | Hôm nay của GV (761987): «Gấp · 5 — Câu hỏi chờ giảng viên: 1 câu hỏi chờ quá 72 giờ làm việc · lâu nhất 145 giờ» ✔. Hai người cùng «Nhận» → 409: N (chỉ 1 GV); mail lỗi thử lại 3 lần: N (mail mock) | `b3-2-today-761987.png` |
| H4a | Hội đồng hỏi B4: mã QR cũ hết hạn | P | qr_url cũ (41 s, >2 chu kỳ 10 s) mở bằng sv04: «Mã đã hết hạn. Quét lại mã trên màn chiếu.» | `h4a-expired.png` |
| H4b | Hội đồng hỏi B4: SV không thuộc lớp quét QR hợp lệ → «Bạn không thuộc lớp này.» | P | sv.moi (chỉ thuộc 761988) quét QR hợp lệ của 761987 → Bạn không thuộc lớp này. (kịch bản mới P-11-112: câu «Bạn không thuộc lớp này.» 403, không nêu lớp khác) | `h4b-svd-outsider.png` |
| H4c | Hội đồng hỏi B4: hai SV cùng một điện thoại → dòng bị gắn cờ «Cần xem» | P | Cùng trình duyệt: sv.nguyco (quét), sv04 và sv05 (nhập mã) → 3 dòng ở «Cần xem (3)» với chip «Cùng máy với bạn khác» (sv04, sv05 thêm chip «Nhập mã»), điểm danh lúc 07:05/07:06/07:07; sv.kha (máy khác) không bị cờ. (Quan sát: SV thuộc 2 lớp phải chọn đúng lớp trước khi nhập mã: ở lớp khác báo «Mã không đúng hoặc đã hết hạn.») | `b4-6-need-review.png` |
| H5 | Hội đồng hỏi B5: nộp sau hạn khi không cho nộp muộn → chặn | N | Không kiểm được trên demo mới: Bài tập 04 có hạn 02/12 (chưa quá hạn) và nút «Nộp bài» hoạt động; lượt 1 hạn đã qua. (Nội dung BT04 vẫn là bản sao BT03: ghi «Nộp muộn: được, trừ 0,5 điểm/ngày», «[BT03] <MSSV>» — Thấp, seed.) Worker chết giữa chừng: N |  |
| H7 | Hội đồng hỏi B7: khoảng 7/14/30/toàn kỳ; chưa đủ dữ liệu 409; 3 báo cáo/giờ; hai người cùng bấm | N | Có 4 nút kỳ 7 / 14 / 30 ngày / Toàn kỳ nhưng chúng chỉ chọn kỳ cho lần «Tạo báo cáo mới»: danh sách luôn là báo cáo mới nhất (30 ngày, 06:41) bất kể chip, không ghi kỳ của báo cáo đang xem (QUAN SÁT Thấp). 409 «Chưa đủ dữ liệu», trần 3 báo cáo/giờ, hai người cùng bấm: không kiểm (PM dặn không bấm «Tạo báo cáo mới»; 1 GV)  | `h7-761988-toan-ky.png` |
| H8 | Hội đồng hỏi B8: chỉ Admin vào /admin/audit; chính việc mở nhật ký được ghi | P | GV và SV vào /admin/audit: «Bạn không có quyền xem màn này» (UI); Admin lọc «Thực thể» = audit_log → 4 dòng VIEW_LIST (việc mở nhật ký được ghi; ô lọc cần tên đầy đủ «audit_log», gõ «audit» → 0 dòng). API 403 cho TA: không kiểm | `h8-admin-audit-self.png` |
| x.B3 | Còn giờ B3: SV B bấm «Đã rõ» → câu hỏi được đóng | P | Đã đóng / Đã đóng / Đã đóng / Hoàn tác | `x-b3-da-ro.png` |
| x.B4a | Còn giờ B4: SV không quét được nhập mã ngắn dưới QR → Đã điểm danh | P | sv04 /checkin: ô «Mã điểm danh» + nút «Điểm danh»; nhập «6YN PH2» → Đã điểm danh 07:06 (chip «Nhập mã» ở /attendance: kiểm ở bước đóng) | `x-b4a-code-sv04.png` |
| x.B4b | Còn giờ B4: GV (375) tắt mạng → chạm 1 ô → bật mạng: xếp hàng rồi tự đồng bộ, không mất | P | Offline (emulate): chạm «Ghi phát biểu · Đỗ Bảo Hải» → «Mất kết nối mạng. Chữ bạn đã nhập vẫn được giữ; hệ thống sẽ gửi lại khi có mạng.»; bật mạng → sau ≤ 5 s «Đã lưu 07:10»; DB: participation_events +0,25 của Đỗ Bảo Hải lúc 07:10:11 (đúng 1 dòng, không trùng) | `x-b4b-online.png` |
| x.B6 | Còn giờ B6: Xuất XLSX sổ điểm lớp 1 | P | Menu «Thêm hành động» → «Xuất XLSX» tải so-diem.xlsx (8,4 KB): tiêu đề An ninh mạng / 761987 / «Bản xem trước — chưa chốt», cột MSSV, Họ tên, Giữa kỳ, Kiểm tra hằng tuần, Thi cuối kỳ, Điểm cộng, Điểm trừ, Điểm quá trình, …, Điểm học phần; có dòng «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» | `x-b6-xlsx.png` |
| x.B7 | Còn giờ B7: Menu ⋯ → xuất Markdown báo cáo lỗ hổng | P | Menu «Thêm hành động» có «Xuất Markdown» / «Xuất PDF»; tải lo-hong-kien-thuc.md (3,8 KB): bảng chủ đề 1., 2., 3.… số SV/câu, khuyến nghị, câu mẫu; không có tên/MSSV/email (kiểm regex). Tệp ghi «30 ngày gần nhất» | `x-b6-xlsx.png` |
| x.B7b | Còn giờ B7: /insights đổi lớp 761988 → chủ đề TCP/UDP đứng đầu; hai lớp không lẫn | P | 761988: báo cáo 25/11 06:39 · 43 câu · 20 SV; hạng 1 «So sánh khái niệm mạng» (16 SV; câu mẫu gồm «UDP khác TCP ở điểm nào?», HTTP/HTTPS, Router/switch, IP/MAC), hạng 3 «Quy trình giao thức mạng» (TCP bắt tay ba bước, DNS). Không có chủ đề mã độc / quy chế của 761987 → dữ liệu hai lớp không lẫn. Tên do LLM đặt nên không phải chữ «TCP / UDP» cứng (kịch bản mới không đọc cứng) | `x-b7b-insights-761988.png` |

## L2.5 Phát hiện còn lại (PM chốt mức)

| # | Mục | Mức đề xuất | Mô tả | Gợi ý |
| --- | --- | --- | --- | --- |
| G1 | 7.2 | Thấp (seed) | Câu mẫu của chủ đề ransomware chỉ có 1 câu: seed có 5 câu giống hệt («Ransomware hoạt động như thế nào?» ×5) nên distinct = 1; kịch bản ghi «3–5 câu» | sửa seed (thêm câu khác nhau) hoặc đổi chữ kịch bản |
| G2 | 7.1 (quan sát) | Thấp–TB | Top 4 của 761987 có «Nhóm chưa có câu hỏi» (#4: cụm lạc đề «công thức nấu…», 2 SV «Ít người hỏi»; #8 cũng tên đó, 3 SV, gắn «Quy chế học vụ», 0 mẫu) HIỆN TRÊN MÀN; hạng 9+ tên là từ khoá («cross, scripting, site»); #3 là chủ đề lạc đề «Thời tiết» đứng trên chủ đề ôn thật; 3 cụm 1-SV gắn chương «Forecasting (QMB ch. 6b)» | đặt tên dự phòng từ câu đại diện hoặc ẩn cụm 0 mẫu; người trình bày đừng cuộn quá hạng 3 |
| G3 | 4.6 | Thấp | /me không có hàng «Điểm danh lúc hh:mm» cho buổi hôm nay «Có mặt» (`RealMe.tsx` chỉ liệt kê buổi khác Có mặt) — lệch kịch bản/SRS dòng 411 | PM chốt: sửa kịch bản hay UI |
| G4 | 5.1 | Thấp | Khi bài ở trạng thái «Đang chấm» (đúng luồng sau chuẩn bị #3) trang SV không có dấu thời gian nộp, chỉ «Đang chấm · Nộp muộn 1 ngày» | như G3 |
| G5 | 3.1 (quan sát) | Thấp | D3 mất 25,6 s với embedding thật (lượt 1: 2,1 s); người trình bày có thể chờ lâu ở Bước 3; SLO D47 (sự kiện SSE đầu ≤ 300 ms) không đo | PM xem embedding cache / làm nóng |
| G6 | H1 | Thấp | «Thử lại sau 09:46» là đếm ngược mm:ss (retry_after ≈ 580 s), dễ đọc nhầm thành giờ | đổi thành «sau 10 phút» |
| G7 | H7 (quan sát) | Thấp | Chip 7/14/30 ngày/Toàn kỳ chỉ chọn kỳ cho lần «Tạo báo cáo mới»; danh sách luôn là báo cáo mới nhất và không ghi kỳ | ghi kỳ vào dòng «Báo cáo …» |
| G8 | 4.2 (quan sát) | Thấp | Màn chiếu: số «0»/«2» không có nhãn chữ «đã điểm danh» | – |
| G9 | H4c (quan sát) | Thấp | SV thuộc 2 lớp nhập mã ngắn khi đang xem lớp khác → «Mã không đúng hoặc đã hết hạn.» (không gợi ý đổi lớp) | tự chọn lớp theo cửa sổ đang mở hoặc gợi ý |
| G10 | seed (đã biết) | Thấp | BT04 hạn 02/12, nội dung vẫn là bản sao BT03 («[BT03] <MSSV>», «Nộp muộn: được…») ⇒ H5 không kiểm được; BT03 chưa gắn cột điểm ⇒ 6.5 N | seed |

Đã biết (PM dặn, KHÔNG tính lỗi): thread «· Phreaker» SKIPPED; 2 thread lạc đề (bóng đá, cà phê muối); 5 thread tri thức ẩn/bỏ ẩn trong nhật ký kiểm duyệt (thấy ở /admin/audit: `thread.hide/unhide`, `escalation_ticket.knowledge_saved`); «Nhóm chưa có câu hỏi» xem G2.

## L2.6 Dữ liệu QC TẠO thêm trên demo (PM quyết giữ / dọn)

| # | Dữ liệu | Chi tiết |
| --- | --- | --- |
| D1 | `sv.moi` vào lớp 761988 | gửi yêu cầu từ `/join/BX4P9TW`, GV duyệt (còn 3 yêu cầu chờ: sv52–54) |
| D2 | Chat riêng `sv.kha` (761987) | 1 phiên: D1, D2, D3; 1 ticket «Thi cuối kỳ… A4» đã nhận → trả lời → SV bấm «Đã rõ» (CLOSED); 1 thư tới sv.kha trong Mailpit; 1 «Hữu ích» |
| D3 | Công tắc «Điểm danh bằng mã» 761987 | BẬT |
| D4 | Buổi 10 (25/11) lớp 761987 | 1 cửa sổ check-in mở rồi đóng; 4 lượt (sv.kha, sv.nguyco QR; sv04, sv05 nhập mã); 25 Vắng, 1 Muộn (Bùi Khánh Việt), 5 Có mặt; «Đã điểm danh» (đã lưu) |
| D5 | Điểm phát biểu +0,25 | Bùi Thanh Khải (sv.kha, 07:08) và Đỗ Bảo Hải (07:10, từ thử offline) |
| D6 | BT03 lớp 761987 | 14 kết quả chấm nháp (DRAFT) mới; bài sv.kha sửa (6,50→7,50) → duyệt → CÔNG BỐ; SV nhận thư |
| D7 | Lớp 761988 | tài liệu quy-che-lop2.pdf, công thức ĐÃ XÁC NHẬN (làm tròn 0,1), 25 SV |
| D8 | Thread ghim 761987 | «Chủ đề đang vướng: Cơ chế hoạt động của ransomware» |
| D9 | Khoá nhập mã join | `sv.nguyco` bị khoá 10 phút (Redis `ep:join:fail:*`), tự hết ≈ 07:13 |
| D10 | `audit_log` | thêm các dòng thao tác ở trên + `VIEW_LIST` ×≈6 (mở nhật ký) + `gradebook.export` ×1 |
| D11 | KHÔNG tạo | báo cáo lỗ hổng mới, bài nộp mới, đổi mã tham gia, đổi mật khẩu, xoá thư Mailpit |

Hoàn tác (gợi ý PM): D3 tắt công tắc; D4 mở khoá lại buổi 10 hoặc seed lại nếu cần demo đúng «Hôm nay có buổi đang diễn ra»; D6 bài sv.kha công bố 7,50 — nếu demo cần trạng thái «chưa công bố» thì không hoàn tác bằng UI được ⇒ seed lại. **Lưu ý demo thật:** Bước 4 và Bước 5 đã bị QC chạy trên chính DB này; muốn trình bày đúng từ đầu cần DB seed lại (buổi 10 «Chưa», BT03 chưa công bố, 761988 chưa xác nhận công thức, sv.moi chưa vào lớp).

---

# Gate P10 — LƯỢT 1 (lưu để đối chiếu): chạy DEMO_SCRIPT.md trên stack demo (main 23c97510)

Người chạy: QC cặp 3 · ngày 25/11/2026 02:20–03:30 (giờ VN) · stack demo `edupilot` (https://localhost, API https://localhost/api/v1, LLM giả, migrate 29, dữ liệu demo của chủ dự án giữ nguyên, bản sao lưu ở `~/Documents/EduPilot-demo-backups/`) · kịch bản: `docs/DEMO_SCRIPT.md` @`origin/main` (v2, 245 dòng).

Cách chạy: 5 trình duyệt Chromium headless RIÊNG (mỗi vai một hồ sơ, không dùng chung phiên): Admin 1440×900, Giảng viên 1440×900, Sinh viên B (`sv.kha`), Sinh viên D (`sv.moi` → dùng lại cho các sinh viên quét QR) và Giảng viên 375 px (phiên thứ hai của `teacher`) ở 375×812 cảm ứng; mọi thao tác bằng giao diện (bấm, gõ, tải tệp); không seed, không reset, không lệnh admin ghi dữ liệu; chỉ đọc DB (`psql select`) để đối chiếu. Camera không có → «quét QR» = mở đúng `qr_url` mà màn chiếu phát (tuổi ≤ 10 s). Ảnh từng bước: `evidence-gate-p10/` (cột «Ảnh»), nhật ký thô: `evidence-gate-p10/rec.json`.

## 1. Kết quả

**52 mục chấm: 39 P · 9 F · 4 N** (8 bước chính + chuẩn bị #3/#4/#5 + «Nếu hội đồng hỏi» + «Nếu còn giờ» kiểm được).

| Nhóm | P | F | N | Ghi chú |
| --- | --- | --- | --- | --- |
| Bước 1 (F2) | 4 | 0 | 0 | đạt |
| Bước 2 (F3) | 2 | 1 | 0 | D1 đúng số liệu nhưng khối «Kết quả» hiện khoá tiếng Anh + JSON thô |
| Bước 3 (F5) | 4 | 2 | 0 | không thấy tên người nhận; bấm chuông tại /chat → khung chat trống |
| Bước 4 (F7) | 4 | 1 | 1 | không có buổi học HÔM NAY (thiếu seed); mã ngắn bị cắt ở 1440 px |
| Bước 5 (F9) | 7 | 0 | 0 | đạt (chuẩn bị #3 phải làm tay ≈ 12–13 phút) |
| Bước 6 (F10) | 4 | 0 | 1 | Bài tập 03 chưa gắn cột điểm → điểm 7,50 không vào sổ |
| Bước 7 (F17) | 1 | 2 | 0 | **ĐỎ với embedding/LLM giả**: không có chủ đề A/B đứng đầu |
| Bước 8 | 1 | 1 | 0 | tên thực thể trong kịch bản khác thực tế |
| Hội đồng hỏi | 6 | 1 | 1 | |
| Còn giờ | 6 | 1 | 1 | |

**Kết luận cho PM:** luồng chạy được từ đầu tới cuối trừ Bước 7 (nội dung báo cáo vô nghĩa với embedding giả) và hai thiếu hụt dữ liệu seed (không có buổi học hôm nay; 761988 không có câu hỏi). Chưa nên bảo vệ ở «tầng 2» cho Bước 7: cần embedding thật hoặc bỏ/đổi bước. 4 phát hiện về giao diện/chữ cần quyết (mục 5).

## 2. Thời gian

Chỉ đo được độ trễ hệ thống (cột «Hệ thống phản hồi» trong các bảng); lời dẫn và thời gian gõ của người trình bày KHÔNG đo được trong lượt tự động, nên **không kết luận được «≤ 15 phút»**. Các điểm đo được:

| Việc | Đo được |
| --- | --- |
| Chat D1 / D2 / D3 (LLM giả) | 1,1 s / 0,6 s / 2,1 s (SSE đầu tiên < 1 s) |
| Nhận / trả lời ticket | 1,8 s / 2,7 s; mail tới Mailpit ≤ 1 s sau «Gửi trả lời» |
| Mở điểm danh → màn chiếu | 2,5 s; mã đổi mỗi 10 s; số đã điểm danh cập nhật ≤ 10 s |
| Đóng + đánh vắng 27 + phát biểu + Lưu (4 thao tác) | ≈ 45 s thao tác máy (kịch bản: < 60 s) |
| Tải quy chế → bản nháp công thức | ≈ 5 s (docling bật) |
| Xác nhận công thức → sổ điểm | 3,2 s + 6,8 s |
| Tạo báo cáo lỗ hổng (Toàn kỳ, 162 câu, embedding giả) | ≈ 3,6 s |
| **Chuẩn bị #3 «Chấm tất cả» BT03 (22 bài, LLM giả)** | **≈ 12–13 phút** (xếp hàng, worker chấm lần lượt) — PHẢI làm trước, không làm được trong 15 phút |

## 3. Chuẩn bị (T−30) — trạng thái thật của demo lúc bắt đầu

| # | Việc | Thực tế |
| --- | --- | --- |
| #1 | Stack + `COMPOSE_PROFILES=ingest` | **Thiếu docling** (không có container `edupilot-docling-1`) và `BLOB_PUBLIC_ENDPOINT=localhost:9000` (CSP `connect-src 'self'` chặn PUT URL ký sẵn): mọi tải tệp lỗi «Không kết nối được tới máy chủ». PM đã sửa giữa lượt (blob → `localhost` + SSL; bật docling); sau đó 6.2 chạy được. |
| #2 | Tầng LLM | Không có nhà cung cấp nào ở `/settings/llm` (chỉ «Quyết định»/decision-fake) → env `fake` (tầng 2). |
| #3 | Chấm nháp BT03 | **Chưa làm** (0/22 đã chấm). QC bấm «Chấm tất cả» (UI): xong sau ≈ 12–13 phút; bài của SV B có cờ «Hai lượt chấm lệch hơn 1 điểm» ✔. |
| #4 | Báo cáo lỗ hổng 2 lớp | Tạo bằng UI «Tạo báo cáo mới»: 761987 chỉ chạy được với khoảng «Toàn kỳ» (dữ liệu seed cũ hơn 14 ngày → «Chưa đủ câu hỏi (cần ≥ 20)»); 761988 có 0 câu hỏi → không tạo được. Kết quả vô nghĩa (mục Bước 7). |
| #5 | Điểm danh bằng mã + buổi học hôm nay | Công tắc «Điểm danh bằng mã» của 761987 TẮT → QC bật trong `/class/settings`; **không có buổi học ngày 25/11** (buổi 11 = 19/11, buổi 12 = 26/11) → dùng Buổi 11 (đã qua) thay thế, nên cả 3 SV quét ra «Muộn». |
| #6–#9, #11 | Hồ sơ trình duyệt, Mailpit, PDF, video | Mailpit hộp trống ✔; PDF `seed/demo/quy-che-lop2.pdf` có trong `main` ✔; video: không kiểm. |
| #10 | Có sẵn dòng nhật ký | Có sau khi QC chạy Bước 4–6 (27 dòng). |

## 4. Kết quả chi tiết từng bước

### Bước 1 — Mở lớp → vào lớp (F2)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 1.1 | Admin /admin/courses | **P** | 2.2 s | Danh sách lớp: 761987, 761988; không có «Chưa gán» | [ảnh](evidence-gate-p10/b1-1-admin-courses.png) |
| 1.2 | GV Hôm nay + chuông | **P** | – | Chuông có «Bạn được phân công lớp An ninh mạng – 761988» (Phân công lớp, 12/11 19:48) và «…761987»; Hôm nay (lớp 761988) có «Thiết lập lớp mới – 3/4 bước xong». LỆCH CHỮ nhỏ: dòng chuông KHÔNG kèm mã tham gia BX4P9TW như kịch bản (mã có ở màn khác). Lưu ý: bộ chọn lớp của GV mở mặc định ở 761988, kịch bản ghi 761987 | [ảnh](evidence-gate-p10/b1-2-teacher-bell-761988.png) |
| 1.3 | SV D (375) /join/BX4P9TW | **P** | 2.6 s | Xem trước: «An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027», «Lớp này cần giảng viên duyệt.»; sau bấm:  / Bản mô phỏng · dữ liệu giả / Tham gia lớp / Đã gửi yêu cầu, chờ giảng viên duyệt. / Về Hôm nay / Hôm nay | [ảnh](evidence-gate-p10/b1-3-svd-after-join.png) |
| 1.4 | GV /class/members duyệt SV D | **P** | 2.7 s | Tab «Chờ duyệt» liệt kê 4 (sv.moi + 3 seed); sau «Duyệt» (hàng sv.moi): còn 3 yêu cầu chờ (sv52–54); sv.moi rời danh sách chờ: true | [ảnh](evidence-gate-p10/b1-4-after-approve.png) |

### Bước 2 — Hỏi riêng tư (F3)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 2.1 | SV B (375) D1 /chat lớp 1 | **F** | 1.1 s | Nội dung ĐÚNG: «Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI»; câu dẫn không nêu số; khối dữ liệu từ sổ: absent 2, present 6, late 1, excused 0, points_total 0.50 (speak_count 2, 2×SPEAK 0.25 ngày 12/11); không có [[SV_n]]; toàn bộ phản hồi 1,1 s. LỆCH TRÌNH BÀY: khối «Kết quả» hiện KHOÁ KỸ THUẬT TIẾNG ANH + JSON thô (late/absent/excused/present/recorded/recent_absent {"date":…}, recent, speak_count, points_total) cho sinh viên, trái DESIGN §13/luật chữ UI; kịch bản chỉ nói «khối gọn nêu số buổi vắng (2) và điểm cộng». Ghi nhận (Thấp–Trung bình, PM xếp) | [ảnh](evidence-gate-p10/b2-1-d1.png) |
| 2.2 | SV B bấm «Hữu ích» | **P** | 0.9 s | Nút đổi trạng thái tại chỗ; thông báo nổi 'Thành công/Cảm ơn': không | [ảnh](evidence-gate-p10/b2-2-huuich.png) |
| 2.3 | SV B D2 (hỏi hộ bạn C) | **P** | 0.6 s | Trả lời: «Trợ lý AI / Mình chỉ xem được dữ liệu của chính bạn. / Hữu ích / Không hữu ích / Gửi / Hôm nay / Chat riêng / Hỏi đáp / Luyện đề / Thêm»; không con số về C: true | [ảnh](evidence-gate-p10/b2-3-d2.png) |

### Bước 3 — Cần giảng viên hỗ trợ (F5)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 3.1 | SV B D3 /chat | **P** | 2.1 s | «AI chưa đủ chắc chắn về câu này» + «Mình chưa tìm thấy nội dung này trong tài liệu của lớp.» + «Đang chờ giảng viên · vừa gửi»; tự chuyển ticket không cần bấm «Nhờ giảng viên hỗ trợ» (Q3); 2,1 s | [ảnh](evidence-gate-p10/b3-1-d3.png) |
| 3.2 | GV / → /inbox, mở ticket mới nhất | **P** | – | Hôm nay (lớp 761987) có «Câu hỏi chờ giảng viên» (Gấp·2); /inbox lọc «Đang chờ» có ticket «Bùi Thanh Khải · 0 phút · Thi cuối kỳ có được mang một tờ A4…»; chi tiết: câu hỏi gốc, «Đang chờ · 0 phút», «Độ tin cậy 0,00». Lưu ý: ticket mới nhất nằm DƯỚI 2 ticket seed cũ (20 và 12 ngày) — người trình bày phải chọn đúng dòng | [ảnh](evidence-gate-p10/b3-2-ticket-detail.png) |
| 3.3 | GV /inbox «Nhận» | **F** | 1.8 s | Sau «Nhận»: trạng thái «Đã nhận · 0 phút», hiện «Gửi trả lời / Bỏ nhận / Đóng không trả lời». KHÔNG thấy TÊN người nhận trên chi tiết/dòng (kịch bản: «Claimed, ghi tên người nhận»). Chỉ có một giảng viên nên không phân biệt được; ghi nhận Thấp (PM/BA: kịch bản hay UI) | [ảnh](evidence-gate-p10/b3-3-claimed.png) |
| 3.4 | GV D4 «Gửi trả lời» | **P** | 2.7 s | Sau gửi: Hộp thư hỗ trợ / Đang chờ / Đã nhận / Đã trả lời / Tất cả / Phạm Mai Chi / 20 ngày / Tấn công chuỗi cung ứng phần mềm là gì? / Đang chờ / Ngô Văn Hải / 12 ngày / Giải thích giúp em: Đánh giá rủi ro an ninh gồm những bước nào? / Đang chờ / Bùi Thanh Khải / Đã t | [ảnh](evidence-gate-p10/b3-4-answered.png) |
| 3.5 | SV B (375) /chat mở chuông | **F** | 6 s | Chuông có «Giảng viên đã trả lời câu hỏi của bạn · vừa xong». ĐÚNG khi mở từ «/» (vào /chat?session=… thấy «Giảng viên đã trả lời · Giảng viên · TS. Lê Thu Hà» + câu D4 + nút «Hỏi tiếp / Đã rõ»). SAI khi bấm thông báo lúc ĐANG Ở /chat (đúng kịch bản 04:50): URL đổi sang ?session=… nhưng khung chat trống «Hỏi bất cứ điều gì về lớp này», chỉ hiện câu trả lời sau khi tải lại trang. Ghi nhận Thấp–Trung bình (1 lần tái hiện, ảnh b3-5-svb-answer.png) | [ảnh](evidence-gate-p10/b3-5-svb-answer.png) |
| 3.6 | Mailpit thư mới nhất | **P** | – | Mailpit :8025 (hộp trống trước đó, 1 thư mới): tới sv.kha@edupilot.local, tiêu đề «Giảng viên đã trả lời câu hỏi của bạn», nội dung câu hỏi + lời GV TS. Lê Thu Hà (D4) + link «Xem và hỏi tiếp: https://localhost/chat?session=…&m=…» | [ảnh](evidence-gate-p10/b3-6-mailpit.png) |

### Bước 4 — Điểm danh bằng QR và phát biểu (F7)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 4.1 | GV chạm việc «Điểm danh buổi đang diễn ra» | **N** | – | THIẾU DỮ LIỆU SEED: không có buổi học NGÀY HÔM NAY (25/11) ở 761987 (buổi 11 = 19/11, buổi 12 = 26/11 09:00 — «Ngày mai»), nên Hôm nay không có việc «Điểm danh buổi đang diễn ra – 761987» và /attendance mở mặc định ở Buổi 11 · 19/11 · Chưa. Thêm: công tắc «Điểm danh bằng mã» của 761987 đang TẮT (chuẩn bị #5 chưa làm) — QC bật trong /class/settings (GV, UI) để đi tiếp; dùng Buổi 11 (19/11) thay buổi hôm nay | [ảnh](evidence-gate-p10/b4-1-attendance-after-enable.png) |
| 4.2 | GV /attendance «Mở điểm danh» → màn chiếu | **F** | 2.5 s | Đã làm được: 1 lần bấm «Mở điểm danh» (POST checkin-windows 201) và trình duyệt vào thẳng /attendance/display (CÙNG tab, không có nút «Mở màn chiếu» riêng, không cửa sổ mới): QR, mã ngắn 6 ký tự dạng «EA2 86J» ở BÊN PHẢI QR (không «ngay dưới»), số đã điểm danh, «còn 09:50» đếm ngược, mã đổi mỗi 10 s (round/short_code đổi, qr_url có token đổi). LỆCH: ở viewport 1440×900 mã ngắn (126 px) tràn ra ngoài khung (right 1485 > 1440, bị cắt «VQ/»; scrollWidth 1485); ở 1920×1080 vừa. Quan trọng cho máy chiếu 1280/1440. Ghi nhận Thấp–Trung bình | [ảnh](evidence-gate-p10/b4-2-display.png) |
| 4.3 | 2 SV (375) quét QR | **P** | 10 s | Camera hệ thống KHÔNG dùng được (headless): QC mở đúng qr_url «/checkin?c=<course>#<window>.<…>» mà màn chiếu phát (lấy từ API display, ≤ 10 s tuổi) cho 3 sinh viên: sv.kha (B), sv.nguyco, sv.gioi → mỗi người thấy «Điểm danh · Đã điểm danh 02:37/02:38» và KHÔNG có nút nào; màn chiếu: số đã điểm danh 0 → 3 (cập nhật sau ≤ 10 s). Phần quét bằng camera: N (không kiểm được) | [ảnh](evidence-gate-p10/b4-3-display-count.png) |
| 4.4 | GV (375) «Đóng», đánh vắng SV chưa quét, phát biểu +0,25 cho B | **P** | 45 s | Sau «Đóng»: «Đã ghi 0 có mặt · 3 muộn · 27 chưa điểm danh» (cả 3 SV quét ra «Muộn» vì QC phải dùng Buổi 11 · 19/11 đã qua, không phải buổi hôm nay); nút «Đánh vắng 27 SV chưa điểm danh» → hộp xác nhận «Đánh vắng 27 sinh viên?» → 27 dòng «Vắng»; «Ghi phát biểu · Bùi Thanh Khải» → dòng ghi «+0,25×1»; mỗi SV quét có «Điểm danh lúc 02:37». LỆCH nhẹ so kịch bản: vắng là thao tác hàng loạt có hộp xác nhận (không chạm từng dòng); «Muộn cho 1» không cần vì cả 3 đã Muộn | [ảnh](evidence-gate-p10/b4-4-speak.png) |
| 4.5 | GV «Lưu điểm danh» | **P** | 2.8 s | Nút «Lưu điểm danh · 3 có mặt · 27 vắng» → dòng tại chỗ «Đã lưu · 3 có mặt · 27 vắng»; Buổi 11 đổi sang «Đã điểm danh»; không thông báo nổi «Thành công»; 2,8 s. Nhánh «Còn n SV chưa điểm danh (422)» không xuất hiện vì đã đánh vắng hết (chưa kiểm riêng) | [ảnh](evidence-gate-p10/b4-5-saved.png) |
| 4.6 | SV B (375) /me | **P** | – | Hàng «Buổi 11 · 19/11 · Muộn · Điểm danh lúc 02:37» (không phải «hôm nay» vì QC dùng buổi 11); «Điểm cộng 0,75 · tối đa 1» (0,50 trước + 0,25 mới); giải trình dạng phép tính («Điểm trừ 0 · vắng 2 buổi, được miễn 2, mỗi buổi trừ 0,5», «Điểm quá trình trước điểm cộng/trừ 7,75 → 8,5») | [ảnh](evidence-gate-p10/b4-6-svb-me.png) |

### Bước 5 — Duyệt và công bố bài tập (F9)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 5.0 | Chuẩn bị #3 «Chấm tất cả» BT03 (GV, UI) | **P** | – | Bấm «Chấm tất cả» (đã chọn Bài tập 03) → «Đã xếp hàng chấm 22 bài.» (Bài tập 03 trước đó 0/22 đã chấm: chuẩn bị #3 chưa làm). Với LLM giả worker chấm xong 22 bài sau ≈ 12–13 phút (kịch bản coi là việc chuẩn bị); kết quả: 17 bài ở lọc «Cần xem kỹ» (lý do: «Trích dẫn không có trong bài», «Tổng điểm chưa khớp», «Nghi chèn lệnh», «Bài lạc đề», «Thiếu nội dung»), bài của Bùi Thanh Khải: «Hai lượt chấm lệch hơn 1 điểm», 6,50, «muộn 1 ngày» | [ảnh](evidence-gate-p10/b5-0-after-grade-all.png) |
| 5.1 | SV B (375) /assignments/[id] Bài tập 03 | **P** | – | Trang chi tiết: «Đã nộp Chủ Nhật, 08/11 · 20:04 · Nộp muộn 1 ngày»; KHÔNG có điểm/nhận xét nháp (đúng); danh sách /assignments ghi «Đã nộp, chờ điểm». LỆCH nhỏ: trang chi tiết không có dòng chữ «đang chấm/chờ điểm» (chỉ danh sách có); phần «Đề bài» hiện markdown thô («# Bài tập 03…», «## Đề bài») thay vì định dạng — Thấp | [ảnh](evidence-gate-p10/b5-1-svb-assignment03.png) |
| 5.2 | GV /grading hàng chờ (lọc mặc định) | **P** | 3 s | Lọc mặc định «Cần xem kỹ» + «Chưa duyệt»: 17 bài; dòng Bùi Thanh Khải: «Hai lượt chấm lệch hơn 1 điểm» (6,50, muộn 1 ngày) ✔. LỆCH: KHÔNG ở đầu danh sách mà ở vị trí 16/17 (xếp theo giờ nộp, không theo mức ưu tiên); 17 dòng cờ khác (Trích dẫn không có trong bài ×11, Tổng điểm chưa khớp ×3, …) — người trình bày phải tìm dòng B | [ảnh](evidence-gate-p10/b5-0-after-grade-all.png) |
| 5.3 | GV /grading/[id]: đọc bài + sửa điểm tiêu chí lệch + sửa nhận xét | **P** | 3.2 s | Trái: bài nộp; phải: 4 tiêu chí, mỗi tiêu chí có đoạn trích «“…”»; dòng cảnh báo tại «Phân tích tấn công»: «Lệch 1,5 điểm so với lượt 2 (1,00 và 2,50)»; sửa điểm tiêu chí lệch 1,00 → 2,00 và một nhận xét: Tổng điểm 6,50 → 7,50 ngay lập tức (trừ nộp muộn 0,50) | [ảnh](evidence-gate-p10/b5-3-edited.png) |
| 5.4 | GV «Duyệt bài» | **P** | 2.7 s | Sau «Duyệt bài»: Đã duyệt bài trước  / Đã duyệt 1/17  | [ảnh](evidence-gate-p10/b5-4-approved.png) |
| 5.5 | GV /grading: chọn bài đã duyệt → «Công bố» | **P** | – | Lọc «Đã duyệt» → tích «Chọn bài của Bùi Thanh Khải» → «Công bố» → hộp xác nhận «Công bố 1 bài · 1 sinh viên nhận thư · không tính vào sổ điểm (bài chưa gắn vào cột điểm)» → «Công bố»; lọc «Đã công bố» có dòng B (7,50). LƯU Ý: bài tập 03 CHƯA gắn vào cột điểm nên điểm KHÔNG vào sổ (ảnh hưởng Bước 6 dòng 10:20); hành động chỉ-Teacher: không kiểm bằng TA | [ảnh](evidence-gate-p10/b5-5-published-list.png) |
| 5.6 | SV B tải lại /assignments/[id] | **P** | – | «Kết quả 7,50 / 10,00»; từng tiêu chí «2,00 / 2,50» + nhận xét (kể cả câu GV sửa «thầy cần thêm bằng chứng…») + đoạn trích «“Bề mặt tấn công bị khai thác…”» từ bài của chính SV; nhãn dưới cùng «Đã có điểm» | [ảnh](evidence-gate-p10/b5-6-svb-result.png) |

### Bước 6 — Công thức điểm và sổ điểm (F10)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 6.1 | GV /gradebook lớp 2 | **P** | 4 s | Đổi sang 761988: banner «Công thức điểm chưa xác nhận» (+ «Chưa có công thức điểm đã xác nhận», nút «Tải quy chế» / «Thiết lập công thức»); «Chốt điểm» disabled; tooltip khi rê: «Chưa có công thức điểm đã xác nhận» | [ảnh](evidence-gate-p10/b6-1-gradebook-hover.png) |
| 6.2 | GV /gradebook/scheme lớp 2: «Tải quy chế» quy-che-lop2.pdf | **P** | – | SAU KHI PM sửa BLOB_PUBLIC_ENDPOINT=localhost / USE_SSL=true và bật docling (profile ingest), xoá tài liệu lỗi do lần tải hỏng trước và tải lại: tiến độ «Đang đọc · quy-che-lop2.pdf» rồi bản nháp sau ≈ 5 s: «Bản nháp», thành phần Giữa kỳ 25% / Bài thực hành 25% / Thi cuối kỳ 50% (Tổng 100%), Điểm cộng «+0,25 mỗi lần phát biểu · tối đa +1,5», Điểm trừ «−0,5 mỗi buổi vắng · miễn 1 buổi…», mỗi mục kèm «Trang 1/2», «Còn 1 mục chưa rõ: làm tròn» và nút «Xác nhận công thức». LƯU Ý (Thấp): lần tải đầu (docling chưa bật) sinh tài liệu «Sẵn sàng» nhưng KHÔNG có bản nháp công thức, tải lại bị chặn «Tệp này đã có trong lớp.» → phải xoá tài liệu mới đi tiếp (đường tự phục hồi thiếu) | [ảnh](evidence-gate-p10/b6-2-scheme-draft.png) |
| 6.3 | GV điền D5 → «Xác nhận công thức» → hộp thoại hậu quả → xác nhận | **P** | 3.2 s | Chọn «Bước làm tròn» = «Làm tròn đến 0,1» → nút «Xác nhận công thức» mở khoá (trước đó disabled); hộp thoại «Áp cho 25 sinh viên · điểm học phần của 0 sinh viên sẽ đổi so với công thức hiện tại · 0 điểm đã nhập được giữ · 0 điểm đã nhập bị bỏ» (Để sau / Xác nhận công thức); sau xác nhận: Công thức điểm / Tải quy chế / Đã xác nhận · Hôm nay, 03:13 / Thành phần / Tổng 100% / Trang 1 / Giữa kỳ · 25% · Quá trình / Bài thực hành · 25% · Quá trình / T. Lưu ý: «Làm tròn đến 0,1» là LỰA CHỌN trong danh sách (0,01/0,1/0,25/0,5/1), không phải ô gõ chữ | [ảnh](evidence-gate-p10/b6-3-confirmed.png) |
| 6.4 | GV /gradebook lớp 2 sau xác nhận | **P** | 6.8 s | Banner «Công thức điểm chưa xác nhận» biến mất; «Chốt điểm» mở | [ảnh](evidence-gate-p10/b6-4-gradebook-after.png) |
| 6.5 | GV /gradebook lớp 1: giải trình dòng SV B | **N** | – | ĐẠT một phần: giải trình «Bùi Thanh Khải» có «Điểm cộng min(0,75; 1) = 0,75» (đã gồm +0,25 từ Bước 4 ✔), «Điểm trừ», «Điểm quá trình (20×7,5+20×8)÷40 → 8,5», «Điểm học phần thô 7,3 → làm tròn 0,5 …» và dòng «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» ✔. KHÔNG ĐẠT: không có thành phần/điểm «Bài tập 03» (7,50 của Bước 5) vì công thức lớp 1 chỉ có Giữa kỳ / Kiểm tra hằng tuần / Thi cuối kỳ và hộp xác nhận «Công bố» đã báo «không tính vào sổ điểm (bài chưa gắn vào cột điểm)». THIẾU CẤU HÌNH SEED (bài tập chưa gắn cột điểm), không phải lỗi mã | [ảnh](evidence-gate-p10/b6-5-explain-B.png) |

### Bước 7 — Báo cáo lỗ hổng kiến thức (F17)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 7.1 | GV /insights lớp 761987 | **F** | – | Báo cáo tạo bằng UI «Tạo báo cáo mới» (khoảng «Toàn kỳ»: 162 câu · 30 SV; khoảng mặc định «14 ngày»/«7»/«30 ngày» của lớp 761987 báo «Chưa đủ câu hỏi để gom (cần ≥ 20)» vì câu hỏi seed cuối cùng ngày 08/11 — QUÁ 14 NGÀY so với 25/11; 761988 = 0 câu hỏi → «Chưa đủ câu hỏi để gom (cần ≥ 20)», nghĩa là kịch bản «lớp 761988: TCP/UDP (C) đứng đầu» không có dữ liệu). KẾT QUẢ SAI KỲ VỌNG: 152 chủ đề, 142 chủ đề 1 SV + 10 chủ đề 2 SV, KHÔNG có chủ đề nào ≥3 SV; đầu bảng là «huong, man, middle», «bang, cach, chan», «diem, doc, khac», «botnet, dung»… tên tạm theo từ khoá + «Chưa đặt được tên chủ đề», mọi chủ đề đều «Ít người hỏi» và nhãn chương «Quy chế học vụ»; KHÔNG có chủ đề A (mã độc/tấn công) / B (quy chế) đứng đầu. Nguyên nhân: embedding GIẢ (hash) trên demo không gom theo nghĩa (kịch bản đã ghi [CHƯA KIỂM: gom cụm với embedding giả]) — bước này ĐỎ ở tầng LLM giả; cần embedding thật hoặc dữ liệu/ngưỡng khác | [ảnh](evidence-gate-p10/prep4-insights-761987-after.png) |
| 7.2 | GV mở «Câu hỏi mẫu» của chủ đề A | **F** | – | Không có chủ đề A để mở. Nhánh phụ đúng kịch bản: chủ đề «Ít người hỏi» (<3 SV) KHÔNG có «Câu hỏi mẫu» và KHÔNG có gợi ý «Nên làm» (✔); nhưng 3–5 câu mẫu ẩn danh của chủ đề A không kiểm được | [ảnh](evidence-gate-p10/prep4-insights-761987-full.png) |
| 7.3 | GV «Tạo thread ghim» (chủ đề hạng 1) | **P** | 5 s | Form mở TẠI CHỖ dưới chủ đề (Tiêu đề điền sẵn «Chủ đề đang vướng: huong, man, middle», Nội dung «Nhiều bạn đang hỏi về “…”.», nút «Đăng»/«Huỷ»), không hộp thoại; «Đăng» → «Đã đăng và ghim thread. Xem ở Hỏi đáp công khai»; /threads có thread ghim: CÓ. LƯU Ý: dùng chủ đề hạng 1 của báo cáo giả («huong, man, middle», tên từ khoá) vì không có chủ đề A; thread đã tạo (liệt kê cuối báo cáo) | [ảnh](evidence-gate-p10/b7-3-threads.png) |

### Bước 8 — Nhật ký quản trị

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 8.1 | Admin /admin/audit (24 giờ) | **P** | – | Admin thấy các dòng vừa tạo ở Bước 3–6 (27 dòng): attendance (complete, absent_unchecked, course_attendance_updated), checkin_window.open/close, participation create, grading_results (grading_result.edit/review/publish), grade_schemes (extract/edit/confirm), escalation_ticket (created/claimed/answered/confirmed), thread.pin, enrollment (member_joined/member_approved), document.delete, gradebook export; mỗi dòng có Thời điểm / Người (kèm vai) / Thực thể / Hành động / Khoá đổi, KHÔNG có giá trị cũ/mới ✔. LỆCH TÊN so kịch bản: «assignments (publish)» thực tế là «grading_results · grading_result.publish»; «grade_entries (set)» KHÔNG có (Bước 6 sinh «grade_schemes · extract/edit/confirm») | [ảnh](evidence-gate-p10/b8-1-audit-2.png) |
| 8.2 | Admin ô «Thực thể» gõ grade_entries | **F** | – | Gõ «grade_entries» → «Không có» (0 dòng); chỉ khi gõ «grade_schemes» mới ra 3 dòng sổ điểm của Bước 6 (extract/edit/confirm). Kịch bản (và chuẩn bị #10: «xác nhận công thức tạo dòng grade_entries / set») sai tên thực thể — sửa kịch bản thành «grade_schemes» hoặc nhập điểm tay trong sổ để có grade_entries | [ảnh](evidence-gate-p10/b8-2-filter-grade_entries.png) |

### «Nếu hội đồng hỏi» kiểm được

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| H1 | Hội đồng hỏi B1: nhập sai mã tham gia | **P** | – | SV (sv.nguyco) vào /join/<mã sai> 5 lần: cùng một thông báo «Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.»; lần 6 và 7: POST /courses/join/preview 429 → «Bạn đã thử quá nhiều lần. Thử lại sau 09:47.» (khoá 10 phút ✔). «Tạo lại mã → mã cũ bị từ chối» KHÔNG chạy (tránh đổi mã AN7K2MQ/BX4P9TW của demo): N | [ảnh](evidence-gate-p10/h1-wrong-code.png) |
| H3 | Hội đồng hỏi B3: ticket chờ quá 72 giờ nổi lên Hôm nay | **P** | – | Hôm nay của GV (761987) có «Gấp · 2 — Câu hỏi chờ giảng viên: 1 câu hỏi chờ quá 72 giờ làm việc · lâu nhất 342 giờ» ✔. Hai người cùng «Nhận» → 409: chỉ có 1 giảng viên nên N; mail lỗi thử lại 3 lần: N (mail mock) | [ảnh](evidence-gate-p10/b3-2-teacher-today.png) |
| H4a | Hội đồng hỏi B4: mã QR cũ hết hạn | **P** | – | Dùng qr_url cũ 28 s (>2 chu kỳ 10 s): «Mã đã hết hạn. Quét lại mã trên màn chiếu.» (Vòng 2 của Buổi 11 mở bằng «Thêm hành động → Mở vòng 2») | [ảnh](evidence-gate-p10/h4-expired.png) |
| H4b | Hội đồng hỏi B4: SV không thuộc lớp quét QR | **F** | – | sv.moi (không thuộc 761987) quét QR hợp lệ → bị từ chối nhưng LỘ LÝ DO: «Bạn không thuộc lớp này.» (kịch bản: «bị từ chối, không lộ lý do»). Ghi nhận Thấp, PM/BA chốt chữ | [ảnh](evidence-gate-p10/h4-nonmember.png) |
| H4c | Hội đồng hỏi B4: hai SV cùng một điện thoại | **P** | – | Trên cùng một trình duyệt, sv04 nhập mã ngắn (Đã điểm danh 03:09) rồi sv05 quét QR: hàng của sv04 ở /attendance ghi «Cùng máy với bạn khác» và bộ lọc «Cần xem» tăng (3 → 5) | [ảnh](evidence-gate-p10/h4-flagged-list.png) |
| H5 | Hội đồng hỏi B5: nộp sau hạn khi không cho nộp muộn | **P** | – | SV B nộp «late-test.txt» vào Bài tập 04 (quá hạn 5 ngày; mô tả cho phép muộn tối đa 2 ngày) → «Đã hết hạn nộp bài.» và KHÔNG tạo bài nộp (Chưa nộp). Lưu ý nội dung Bài tập 04 trong demo là bản sao «Bài tập 03» (đề/tiêu chí/ngày muộn) — Thấp (seed). «Worker chết giữa chừng → không chấm đôi»: N | [ảnh](evidence-gate-p10/h5-bt04-submit.png) |
| H7 | Hội đồng hỏi B7: khoảng 7/14/30/toàn kỳ; 3 báo cáo/giờ; hai người cùng bấm | **N** | – | Khoảng 7 / 14 / 30 ngày / Toàn kỳ có đủ 4 lựa chọn; «14 ngày» (và 7/30) báo «Chưa đủ câu hỏi để gom (cần ≥ 20)» vì dữ liệu seed cũ hơn 14 ngày; «Toàn kỳ» tạo được báo cáo (✔ cùng tầng embedding giả). Bấm «Tạo báo cáo mới» 4 lần liên tiếp: chỉ lần đầu tạo (DB insight_reports 761987: 19:30:50 và 20:14:25 UTC), 3 lần sau KHÔNG tạo thêm và KHÔNG báo gì (không thấy «3 báo cáo mỗi giờ» hay «đang chạy») → giới hạn 3/giờ và «hai người cùng bấm» không dựng được (N); lưu ý bấm lặp im lặng | [ảnh](evidence-gate-p10/h7-limit.png) |
| H8 | Hội đồng hỏi B8: GV/SV vào /admin/audit; mở nhật ký được ghi | **P** | – | UI GV và SV: «Bạn không có quyền xem màn này · Trang này dành cho quản trị viên.»; API GET /admin/audit: teacher 403, sv.kha 403; Admin mở /admin/audit → audit_log có dòng VIEW_LIST (entity audit_log) = việc mở nhật ký được ghi lại | [ảnh](evidence-gate-p10/h8-teacher-audit.png) |

### «Nếu còn giờ» (dự trữ)

| # | Bước / vai | KQ | Hệ thống phản hồi | Quan sát | Ảnh |
| --- | --- | --- | --- | --- | --- |
| 1.x | Còn giờ B1: SV D tải lại / | **P** | 3.1 s | Bộ chọn lớp: 761988 ; ô nhập mã: không còn | [ảnh](evidence-gate-p10/b1-x-svd-reload.png) |
| x.B2 | Còn giờ B2: SV B mở «Nguồn tham khảo» dưới câu trả lời D1 | **F** | – | Dưới câu trả lời D1 KHÔNG có nút/khối «Nguồn tham khảo» (D1 là câu tra sổ điểm danh, không trích tài liệu); chỉ có «Tìm hiểu» (giải thích che thông tin cá nhân: «Tên, mã số sinh viên, email và số điện thoại được thay bằng ký hiệu trước khi gửi cho AI, rồi hiện lại cho bạn.») và «Hữu ích / Không hữu ích / Nhờ giảng viên hỗ trợ». Kịch bản sai chỗ hoặc D1 cần nguồn — PM/BA chốt | [ảnh](evidence-gate-p10/x-b2-timhieu.png) |
| x.B3 | Còn giờ B3: SV B bấm «Đã rõ» | **P** | – | Dưới câu trả lời GV có «Hỏi tiếp» / «Đã rõ»; bấm «Đã rõ» → «Đã đóng · Hoàn tác» tại chỗ (câu hỏi đóng), không hộp thoại/thông báo nổi | [ảnh](evidence-gate-p10/x-b3-darou.png) |
| x.B4a | Còn giờ B4: SV nhập mã ngắn khi không quét được | **P** | – | /checkin: ô «Mã điểm danh» + nút «Điểm danh»; nhập mã ngắn từ màn chiếu (VD «3TC 1MV») → «Đã điểm danh 03:09»; dòng ở /attendance có chip «Nhập mã» ✔ | [ảnh](evidence-gate-p10/h4-short-code.png) |
| x.B4b | Còn giờ B4: GV (375) tắt mạng → chạm một ô → bật mạng | **P** | – | Offline: bấm «Ghi phát biểu · Đỗ Bảo Hải» → dòng hiện ngay «+1 · +0,25×1», DB participation_events vẫn 67 (chưa gửi); bật mạng, sau ≤ 9 s DB = 68 và UI giữ nguyên → xếp hàng rồi tự đồng bộ, không mất ✔ | [ảnh](evidence-gate-p10/x-b4-online2.png) |
| x.B6 | Còn giờ B6: Xuất XLSX sổ điểm lớp 1 | **P** | – | Menu «Thêm hành động» → «Xuất XLSX» tải so-diem.xlsx (8,4 KB): tiêu đề «An ninh mạng · 761987 · 25/11/2026 03:04 · Bản xem trước — chưa chốt», cột MSSV/Họ tên/Giữa kỳ/Kiểm tra hằng tuần/Thi cuối kỳ/Điểm cộng/Điểm trừ/Điểm quá trình/Điểm thi cuối kỳ/Điểm học phần/Điểm chữ/Ghi chú, 33 dòng, và dòng «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» | – |
| x.B7 | Còn giờ B7: xuất Markdown báo cáo lỗ hổng | **P** | – | Menu «⋯» → «Xuất Markdown» tải lo-hong-kien-thuc.md (8,7 KB): tiêu đề «Lỗ hổng kiến thức — 120 ngày gần nhất», «Tạo lúc 24/11/2026 19:30 · 162 câu hỏi», danh sách chủ đề + số SV/câu; không có tên hay MSSV (kiểm regex: không). LỆCH nhỏ: giờ «Tạo lúc 19:30» là UTC trong khi màn báo «25/11/2026, 02:30» (giờ VN) — xuất ra lệch 7 giờ (Thấp). Chủ đề là tên từ khoá của báo cáo giả | – |
| x.B7b | Còn giờ B7: /insights đổi lớp 761988 → TCP/UDP (C) đứng đầu | **N** | – | THIẾU DỮ LIỆU SEED: 761988 không có câu hỏi chat nào (0 tin nhắn người dùng) → «Chưa đủ câu hỏi để gom (cần ≥ 20)» ở mọi khoảng; không có chủ đề C để so với 761987 (dữ liệu hai lớp không lẫn: chưa kiểm được) | [ảnh](evidence-gate-p10/prep4-insights-761988-after.png) |

## 5. Phát hiện cần xử lý (xếp mức đề xuất — PM chốt)

| # | Mức đề xuất | Mục | Mô tả | Loại |
| --- | --- | --- | --- | --- |
| G1 | **Cao (cho demo)** | 7.1, 7.2 | Báo cáo lỗ hổng với embedding giả: 152 chủ đề, 142 chủ đề 1 SV + 10 chủ đề 2 SV, mọi chủ đề «Ít người hỏi», nhãn «Quy chế học vụ», tên là từ khoá («huong, man, middle»…), KHÔNG có chủ đề A (mã độc/tấn công) / B đứng đầu, không có câu mẫu. Mặc định «14 ngày» luôn «Chưa đủ câu hỏi» vì dữ liệu seed già. | Kịch bản `[CHƯA KIỂM: embedding giả]` nay xác nhận ĐỎ; cần embedding thật hoặc đổi kịch bản/ngưỡng; dữ liệu seed cần câu hỏi mới nhất |
| G2 | Trung bình | 2.1 | Khối «Kết quả» dưới câu trả lời D1 hiện khoá tiếng Anh + JSON thô (`late`, `absent`, `recent_absent {"date":…}`, `speak_count`, `points_total`) cho sinh viên | Giao diện (DESIGN §13) |
| G3 | Trung bình | 4.2 | `/attendance/display` ở 1440×900: mã ngắn (126 px) tràn khỏi khung (right 1485 > 1440, bị cắt); ở 1920×1080 vừa. Mã nằm bên phải QR, không «ngay dưới QR» | Giao diện (máy chiếu) |
| G4 | Thấp–Trung bình | 3.5 | SV đang ở `/chat` bấm thông báo «Giảng viên đã trả lời» → URL đổi `?session=…` nhưng khung chat trống; chỉ hiện câu trả lời sau khi tải lại (mở từ `/` thì đúng) | Lỗi giao diện |
| G5 | Thấp | 3.3 | Sau «Nhận» chỉ ghi «Đã nhận», không thấy TÊN người nhận | Kịch bản/UI |
| G6 | Thấp | H4b | SV không thuộc lớp quét QR → «Bạn không thuộc lớp này.» (kịch bản: không lộ lý do) | Chữ |
| G7 | Thấp | 8.1, 8.2 | Tên thực thể: kịch bản ghi `assignments (publish)` / `grade_entries (set)`; thực tế `grading_results · grading_result.publish` và `grade_schemes · extract/edit/confirm`; lọc `grade_entries` → «Không có» (cả chuẩn bị #10 sai) | Kịch bản |
| G8 | Thấp | x.B2 | D1 là câu tra sổ nên không có «Nguồn tham khảo» (chỉ có «Tìm hiểu») | Kịch bản |
| G9 | Thấp | x.B7 | Xuất Markdown báo cáo ghi «Tạo lúc 24/11/2026 19:30» (UTC) trong khi màn báo «25/11/2026, 02:30» (giờ VN) — lệch 7 giờ | Lỗi nhỏ |
| G10 | Thấp | 6.2 | Lần tải quy chế đầu hỏng (docling chưa bật) để lại tài liệu «Sẵn sàng» nhưng KHÔNG có bản nháp công thức; tải lại bị chặn «Tệp này đã có trong lớp.» → phải xoá tài liệu mới đi tiếp | Đường phục hồi |
| G11 | Thấp | 5.1 | Đề bài BT03/BT04 hiện markdown thô («# Bài tập 03…», «## Đề bài»); Bài tập 04 là bản sao nội dung BT03 | Giao diện + seed |
| G12 | Thấp | H7 | Bấm «Tạo báo cáo mới» lặp (không có dữ liệu mới) im lặng, không báo «đã có báo cáo mới nhất» | UX |
| G13 | Lưu ý | – | Chế độ «Bản mô phỏng · dữ liệu giả»: tên trên đầu trang (vd. `sv.kha` hiện «Trần Thu Uyên», `sv.moi` «Phạm Ngọc Linh», admin «Đỗ Hoàng Nam») KHÁC tên thật trong DB/danh sách lớp (Bùi Thanh Khải, Lý Thu Minh…). D1/D2 phải gõ tên thật theo seed («Bùi Thanh Khải» / «Ngô Ngọc Cẩm», MSSV 20229002); người trình bày đọc tên từ đầu trang sẽ gõ sai | Kịch bản/seed |

**Thiếu dữ liệu seed (không tính lỗi sản phẩm):** (a) không có buổi học ngày hôm nay ở 761987 (4.1); (b) 761988 không có câu hỏi chat (7.1, x.B7b); (c) Bài tập 03 chưa gắn cột điểm nên điểm không vào sổ (6.5); (d) công tắc «Điểm danh bằng mã» tắt (chuẩn bị #5); (e) chưa chấm nháp BT03 (chuẩn bị #3); (f) câu hỏi seed già hơn 14 ngày; (g) một số thread Hỏi đáp công khai bằng tiếng Anh lẫn (vd. «Seasonality without Trend») trong lớp An ninh mạng.

**Không chạy được (N), lý do:** camera quét QR thật; «tạo lại mã → mã cũ bị từ chối» (đổi mã demo AN7K2MQ/BX4P9TW); hai người cùng «Nhận» ticket (chỉ có 1 giảng viên); mail lỗi thử lại 3 lần; worker chết giữa chừng; mất mạng giữa lúc AI đang trả lời (LLM giả trả < 1 s); mọi dịch vụ AI hỏng → trích tài liệu; hai người sửa cùng ô sổ điểm (409) và mở khoá sau chốt; 3 báo cáo/giờ và hai người cùng bấm tạo báo cáo; video dự phòng; Bước 6 `Chốt điểm` lớp 1 (kịch bản không làm trong 15 phút).

## 6. Dữ liệu QC đã TẠO / ĐỔI trên demo (PM quyết giữ / dọn)

| # | Dữ liệu | Chi tiết |
| --- | --- | --- |
| D1 | Lớp 761988 — thành viên | `sv.moi` (Lý Thu Minh) gửi yêu cầu `/join/BX4P9TW` rồi được GV duyệt → thành viên ACTIVE (24 → 25); còn 3 yêu cầu seed (sv52–54) chờ duyệt |
| D2 | Chat riêng 761987 của `sv.kha` | 1 phiên mới (`01a20f0c-6c77-…`) gồm D1, D2, D3; phản hồi «Hữu ích» cho D1; ticket D3 (Bùi Thanh Khải) ĐÃ TRẢ LỜI (D4) rồi SV bấm «Đã rõ» → đã đóng; thông báo + 2 thư Mailpit tới `sv.kha` («Giảng viên đã trả lời…», «Điểm bài tập đã được công bố») |
| D3 | Cài đặt lớp 761987 | công tắc «Điểm danh bằng mã» đã BẬT (trước đó tắt) |
| D4 | Điểm danh 761987 Buổi 11 (19/11) | 2 cửa sổ điểm danh (vòng 1, vòng 2); 5 lượt check-in (`sv.kha`, `sv.nguyco`, `sv.gioi` quét QR; `sv04` nhập mã ngắn; `sv05` quét QR cùng máy → cờ «Cùng máy»); 27 SV bị ĐÁNH VẮNG, 3 «Muộn»; Buổi 11 → «Đã điểm danh» (trước: «Chưa»); vòng 2 gắn «Vắng ở lượt kiểm lại» cho sv.kha, sv.gioi |
| D5 | Điểm cộng phát biểu +0,25 | `sv.kha` (+0,25 → điểm cộng 0,75), `sv23` (Bùi Minh Yến) và `sv06` (Đỗ Bảo Hải) mỗi người +0,25 (kiểm nhánh mất mạng); 3 dòng `participation_events` |
| D6 | Bài tập 03 (761987) | «Chấm tất cả» → 22 kết quả chấm nháp (DRAFT 21 + 1 PUBLISHED); bài `sv.kha` đã SỬA (điểm tiêu chí «Phân tích tấn công» 1,00 → 2,00, nhận xét), DUYỆT và CÔNG BỐ: 7,50/10 (SV thấy điểm) |
| D7 | Báo cáo lỗ hổng 761987 | 2 bản ghi `insight_reports` (19:30:50 và 20:14:25 UTC, «Toàn kỳ»); thông báo «Báo cáo lỗ hổng kiến thức đã sẵn sàng»; 1 thread ghim «Chủ đề đang vướng: huong, man, middle» ở Hỏi đáp công khai |
| D8 | Lớp 761988 — công thức điểm | tài liệu `quy-che-lop2.pdf` (đã xoá bản lỗi, tải lại bản mới, ĐÃ XÁC NHẬN công thức, «Làm tròn đến 0,1») — banner «chưa xác nhận» của lớp 2 đã biến mất, «Chốt điểm» đã mở; nhật ký: `document.delete`, `grade_schemes extract/edit/confirm` |
| D9 | Khoá thử | `sv.nguyco` bị khoá tham gia lớp 10 phút (bài thử mã sai; hết tự động) |
| D10 | Phiên/thông báo/tệp | nhiều phiên đăng nhập của `teacher`, `admin`, `sv.kha`, `sv.moi`, `sv.nguyco`, `sv.gioi`, `sv04`, `sv05`, `sv.chuaxm`; tệp tải xuống (so-diem.xlsx, lo-hong-kien-thuc.md) chỉ ở máy QC; 1 lần «Xuất XLSX» (audit `gradebook export`) |
| D11 | Hạ tầng (PM làm giữa lượt) | gateway/worker khởi động lại với `BLOB_PUBLIC_ENDPOINT=localhost`, `BLOB_PUBLIC_USE_SSL=true`; container `edupilot-docling-1` được bật |

Không tạo / không đổi: mã tham gia lớp, mật khẩu, tài khoản, bài nộp của SV (BT04 thử nộp bị chặn nên không tạo), điểm sổ điểm lớp 1 (chưa «Chốt điểm»).

Để khôi phục nhanh trạng thái trước demo (nếu PM muốn): bản sao lưu `~/Documents/EduPilot-demo-backups/` (QC không đụng). Các thay đổi D3, D4, D6, D8 là phần có tác động thấy được khi chủ dự án chạy lại kịch bản (Bước 4 «Chưa điểm danh» và Bước 5/6 «chưa chấm / chưa xác nhận» đã được làm sẵn).

## 7. Nhật ký chạy lại (DEMO_SCRIPT.md mục 7) — dòng đề xuất

| Ngày | Phase vừa xong | Bước xa nhất chạy thật | Tổng thời gian | Tầng | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| 2026-11-25 | Gate P10 (main 23c97510) | Bước 8 (đủ 8 bước) | không đo được trọn kịch bản (xem mục 2); chuẩn bị #3 ≈ 12–13 phút | Tầng 2 (LLM giả) | Bước 7 đỏ với embedding giả; 4 phát hiện giao diện; 7 điều kiện seed chưa đủ (mục 5) |

## L2.7 Dò lại sau sửa (ep-qc, HEAD sprint/11-p10-eval 7ad0d062)

Rebuild ep-qc gateway + worker + frontend lên 7ad0d062 (từng service, RAM ≥ 2500 MB; migrate: 0 mới, version 29). Lớp QC `QD10Q` (không 761987/88; thêm 1 buổi hôm nay bằng SQL vào ep-qc, bật «Điểm danh bằng mã», sv36 quét QR, GV Đóng → Đánh vắng → Lưu).

| Lỗi | Kết quả | Chứng cứ |
| --- | --- | --- |
| GP10-06 (810c16fe): `/me` hàng «Điểm danh lúc hh:mm» cho buổi «Có mặt» | **P** | sv36 `/me` ở 1440 và 375 px: «Bạn có mặt 1/1 buổi, vắng 0.» + «Buổi 1 · 25/11 — Có mặt · Điểm danh lúc 07:20»; `evidence-gp10-06/me-1440.png`, `me-375.png` |
| GP10-08 (7ad0d062): cụm 0 câu mẫu không gửi LLM đặt tên | **P** | `go test ./internal/insight -run TestKnowledgeGap -count=1` xanh (cả `TestKnowledgeGapNameZeroSample` PASS) từ worktree 7ad0d062 |
