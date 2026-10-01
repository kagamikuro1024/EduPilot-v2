# QC test case xuyên suốt — đi trọn `docs/DEMO_SCRIPT.md` bước 1–7 trên prototype
Nguồn: `docs/DEMO_SCRIPT.md` (kim chỉ nam), `docs/sprints/1/prototype/spec/SRS.md` mục 3 (đường đi), 4.1 (con số), FR-X7 (bỏ bước Mailpit 05:55). Một trình duyệt, đổi vai bằng menu hồ sơ; **không** tải lại trang bằng tay giữa chừng ngoài chỗ kịch bản ghi "Tải lại". Chạy sau khi cả US-PROTO-01…04 có handoff. Đặt lại dữ liệu demo trước khi bắt đầu; ghi giờ thực mỗi bước (≤ khoảng của kịch bản).
Công cụ: **B** trình duyệt (Eval `browser`, `scripts/demo-run.mjs` khi dựng xong; ảnh `qc/shots/demo-*.png`, 1440 px cho GV/Admin, 390 px cho SV B / SV D như cột "Vai trò" của kịch bản).

| TC-id | Bước | Vai / viewport | Thao tác (nguyên văn kịch bản) | Kết quả phải thấy |
| --- | --- | --- | --- | --- |
| TC-DEMO-00 | Mở đầu | – | Mở `/login` rồi chọn vai | Dải "Bản mô phỏng · dữ liệu giả"; học phần "An ninh mạng", hai lớp 761987 / 761988, mỗi lớp 30 SV, lớp 1 tuần 10 (29/10/2026 09:20) |
| TC-DEMO-01 | 1 · 00:30 | Admin 1440 | `/admin/courses` | Hai lớp 761987, 761988, cả hai đã gán giảng viên |
| TC-DEMO-02 | 1 · 00:50 | Giảng viên | `/` → chuông | "Bạn được phân công lớp An ninh mạng – 761988" kèm `BX4P9TW`; việc "Thiết lập lớp mới" ghi tên lớp 2 |
| TC-DEMO-03 | 1 · 01:10 | Sinh viên D 390 | `/join/BX4P9TW` → xem trước → `Tham gia lớp` | Xem trước: tên lớp, mã lớp 761988, giảng viên, học kỳ; sau bấm: "chờ giảng viên duyệt" (lớp 2 bật duyệt) |
| TC-DEMO-04 | 1 · 01:35 | Giảng viên | `/class/members` (bộ chọn lớp → "Quản lý lớp này") → `Duyệt` D | D thành thành viên; **3 yêu cầu seed khác vẫn chờ** |
| TC-DEMO-05 | 1 · 01:55 | Sinh viên D 390 | `/` tải lại | Bộ chọn lớp có lớp 761988; không còn ô nhập mã |
| TC-DEMO-06 | 2 · 02:15 | Sinh viên B 390 | `/chat` lớp 1, gõ D1 nguyên văn | "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI"; trả lời chảy; vắng 2, +0,75; không `[[SV_1]]` |
| TC-DEMO-07 | 2 · 03:05 | B 390 | `Hữu ích` | Đổi trạng thái tại chỗ, **không thông báo nổi / toast** |
| TC-DEMO-08 | 2 · 03:15 | B 390 | Gõ D2 | Chỉ trả lời thông tin của chính bạn; không con số nào về C |
| TC-DEMO-09 | 2 · 03:45 | B 390 | (tuỳ chọn) `Nguồn tham khảo` | Mở ngay dưới câu trả lời |
| TC-DEMO-10 | 3 · 04:15 | B 390 | Gõ D3 | "AI chưa đủ chắc chắn về câu này"; "Đang chờ giảng viên · vừa gửi" (Q1: tự chuyển) |
| TC-DEMO-11 | 3 · 04:45 | Giảng viên | `/` → `/inbox` | Ticket D3 ở lọc Open: câu hỏi gốc, thời gian chờ, lý do |
| TC-DEMO-12 | 3 · 05:00 | Giảng viên | `Nhận` | Claimed + tên người nhận |
| TC-DEMO-13 | 3 · 05:10 | Giảng viên | Gõ D4 → `Gửi trả lời` | Answered; dòng "Đã gửi thư thông báo… (mô phỏng)" thay bước Mailpit 05:55 (FR-X7) |
| TC-DEMO-14 | 3 · 05:35 | B 390 | `/chat` → chuông | Câu trả lời giảng viên trong phiên chat, nhãn giảng viên |
| TC-DEMO-15 | 3 · 06:10 | B 390 | Xác nhận đã rõ (`Đã rõ`) | Câu hỏi đóng |
| TC-DEMO-16 | 4 · 06:30 | Giảng viên 390 | `/` chạm "Điểm danh buổi đang diễn ra – 761987" | Mở `/attendance` đúng buổi 10 hôm nay |
| TC-DEMO-17 | 4 · 06:40 | Giảng viên 390 | Chạm `Vắng` cho 2 SV, `Muộn` cho 1, +0,25 phát biểu cho B | Trạng thái lưu hiển thị; **toàn bộ < 60 s** (ghi giờ) |
| TC-DEMO-18 | 4 · 07:10 | Giảng viên 390 | `Lưu điểm danh` | "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; không toast "Thành công" |
| TC-DEMO-19 | 4 · 07:25 | B 390 | `/me` | Chuyên cần + điểm cộng có thêm +0,25; **QT = 8,5** (SRS 4.1: 8,3 → 8,5) |
| TC-DEMO-20 | 4 · 07:45 | Giảng viên 390 | (tuỳ chọn) "Giả lập mất mạng" → đổi ô → bật lại | "Đang chờ mạng · n thay đổi" → "Đã lưu", không mất |
| TC-DEMO-21 | 5 · 08:00 | B 390 | `/assignments/bt03` (mở từ `/me`) | Đã nộp, dấu thời gian, nhãn nộp muộn; "Đang chấm"; **không có điểm nháp** |
| TC-DEMO-22 | 5 · 08:15 | Giảng viên | `/grading` giữ lọc mặc định | Bài B ở đầu; lý do cờ: hai lượt chấm lệch hơn 1 điểm |
| TC-DEMO-23 | 5 · 08:35 | Giảng viên | Mở bài B; sửa điểm tiêu chí lệch; sửa một nhận xét | Thông báo vàng tại tiêu chí 2; mỗi tiêu chí có đoạn trích; tổng tính lại ngay |
| TC-DEMO-24 | 5 · 09:05 | Giảng viên | `Duyệt bài` | Đã duyệt |
| TC-DEMO-25 | 5 · 09:15 | Giảng viên | `/grading` → bài vừa duyệt → `Công bố` | Đã công bố; chỉ TEACHER thấy hành động này |
| TC-DEMO-26 | 5 · 09:35 | B 390 | `/assignments/bt03` tải lại | Điểm 8,5 (theo spec; proposals #13), nhận xét theo tiêu chí, đoạn trích từ bài của B; `/me` QT **8,8** (8,5 → 8,8) |
| TC-DEMO-27 | 6 · 10:00 | Giảng viên | `/gradebook` lớp 2 | Banner "công thức điểm chưa xác nhận"; `Chốt điểm` khoá, lý do khi rê / focus |
| TC-DEMO-28 | 6 · 10:15 | Giảng viên | `/gradebook/scheme` lớp 2, tải `seed/demo/quy-che-lop2.pdf` | Tiến độ xử lý nền; bản nháp: thành phần, trọng số, điểm cộng, mỗi mục cạnh đoạn trích có số trang; mục "chưa rõ: quy tắc làm tròn" chặn nút xác nhận |
| TC-DEMO-29 | 6 · 10:55 | Giảng viên | Điền D5 → `Xác nhận công thức` → xác nhận trong hộp thoại | Đã xác nhận |
| TC-DEMO-30 | 6 · 11:15 | Giảng viên | `/gradebook` lớp 2 | Banner biến mất; `Chốt điểm` mở |
| TC-DEMO-31 | 6 · 11:25 | Giảng viên | `/gradebook` lớp 1, mở giải trình hàng B | Có +0,25 từ bước 4 và điểm BT03 từ bước 5; dòng "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường"; QT 8,8 |
| TC-DEMO-32 | 6 · 11:45 | Giảng viên | `Xuất XLSX` (menu) | "Đã tạo file sổ điểm (mô phỏng)" có dòng ghi chú điểm chính thức |
| TC-DEMO-33 | 6 · What-if | B 390 | `/me`: What-if CK = 8,0 | **8,3** (0,4 × 8,8 + 0,6 × 8,0; SRS 4.1) |
| TC-DEMO-34 | 7 · 12:00 | Giảng viên | `/insights` lớp 1 → `Tạo báo cáo mới` | Tiến độ; chủ đề A, B đứng đầu, mỗi chủ đề lý do + tín hiệu |
| TC-DEMO-35 | 7 · 12:35 | Giảng viên | Mở câu mẫu chủ đề A | 3–5 câu đã ẩn danh, không tên / MSSV; chủ đề < 3 SV không có câu mẫu |
| TC-DEMO-36 | 7 · 12:50 | Giảng viên | `Tạo thread ghim` | Thread ghim mới ở `/threads` lớp 1 (kiểm bằng vai B) |
| TC-DEMO-37 | 7 · 13:05 | Giảng viên | `/insights` lớp 2 | Chủ đề C đứng đầu — dữ liệu hai lớp không lẫn |
| TC-DEMO-38 | Kết | – | Tổng thời gian | Cả kịch bản đi được trong ≤ 13:45 thao tác thực (ghi giờ chạy tay; bấm có thể nhanh hơn); không ngõ cụt, không phải sửa URL tay (trừ mở lần đầu), không tải lại bắt buộc ngoài "Tải lại" ở TC-DEMO-05, 26 |

## Kiểm các con số (SRS 4.1)
| Mốc | Giá trị | TC |
| --- | --- | --- |
| QT B lúc 09:20 | **8,3** | TC-01-16, TC-03-15 |
| Sau điểm danh | **8,5** | TC-DEMO-19, TC-02-10 |
| Sau công bố BT03 | **8,8** (8,5 = 9,0 − 0,5 theo spec; mâu thuẫn rubric → proposals #13) | TC-DEMO-26, TC-03-08 |
| What-if CK 8,0 | **8,3** | TC-DEMO-33, TC-01-17 |

## Quy tắc kiểm áp dụng ở mọi bước
- Vai: trước mỗi bước đổi vai bằng `Đổi vai` (không sửa cookie tay); trạng thái của vai trước thấy được ở vai sau (FR-X3).
- 390 px: mọi bước ghi "(375 px)" trong kịch bản được chạy ở 390 (và 375 cho SV B, `/attendance`, `/inbox` bằng `sweep.mjs`): không cuộn ngang, vùng chạm ≥ 44 px.
- Ảnh chụp: ít nhất một ảnh mỗi bước (`shots/demo-NN-*.png`); kiểm DESIGN §21 (phản mẫu) và §22 (10 điều kiện) bằng mắt.
- Console không lỗi; không `fetch`/mạng ngoài ngoài font.
- Không toast "Thành công"; không spinner giữa trang; một nút chính đỏ mỗi vùng; đỏ chỉ mang ba nghĩa (CLAUDE.md luật giao diện).
- Sinh viên không thấy từ kỹ thuật AI, điểm nháp, ghi chú / nhãn rủi ro của mình ở **mọi** bước SV.
- Lần chạy thứ hai sau `Đặt lại dữ liệu demo`: kịch bản đi lại từ đầu, mọi giá trị về dữ liệu gốc (lặp được cho buổi thuyết trình).

## Nhánh lỗi / chưa đi trong kịch bản chính
Mạng/AI hỏng (tầng 2/3 của DEMO_SCRIPT mục 5) không có trong prototype (không backend). Ba nhánh lỗi mô phỏng được chấm riêng ở các TC story: 409 nhận ticket (TC-02-07), mất mạng điểm danh (TC-02-18), lệch điểm chấm (TC-03-04).

## Điểm khó kiểm
- Thời gian ≤ 60 s điểm danh và mốc thời gian kịch bản phụ thuộc thao tác: ghi giờ thực, không FAIL nếu chỉ lệch do người bấm; FAIL nếu cần > 3 bước ngoài kịch bản.
- Bỏ bước Mailpit 05:55 (FR-X7) và các nhánh "Nếu hội đồng hỏi" (không có trong prototype).

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có)
