# QC report — Kịch bản demo 15 phút (US-PROTO-DEMO)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit toàn diện `1915734` và nền chung.
Tài liệu nguồn: `docs/DEMO_SCRIPT.md` (kim chỉ nam), `docs/sprints/1.5/spec/SRS.md` mục 3 (đường đi) và 4.1 (con số theo v3).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-DEMO.md` (38 bước kịch bản + 4 mốc điểm).

**Tóm tắt:** Kịch bản demo 15 phút (F2→F3→F5→F7→F9→F10→F17) đi trọn vẹn từ đầu đến cuối trên một trình duyệt bằng cách đổi vai tại menu hồ sơ. Toàn bộ các mốc điểm và phép tính giải trình tuyến tính ăn khớp 100% với SRS 4.1 v3 (proposals #13: 8,3 → 8,5 → 8,7; What-if 8,3).

## Cổng nghiệm thu kịch bản
| Bước | Luồng | Thao tác chính | Kết quả | Ghi chú |
| --- | --- | --- | --- | --- |
| Bước 1 | F2 | Mở lớp → Vào lớp | **PASS** | Admin thấy 2 lớp 761987 & 761988; GV thấy chuông phân công và mã `BX4P9TW`; SV D mở `/join/BX4P9TW` xem trước lớp 761988 và gửi yêu cầu; GV duyệt D tại `/class/members` lớp 2; SV D tải lại có lớp mới |
| Bước 2 | F3 | Hỏi riêng tư | **PASS** | SV B gõ D1 được trả lời vắng 2 buổi, +0,75 phát biểu, 2 nguồn tham khảo, ẩn 2 thông tin cá nhân; D2 từ chối lịch sự không lộ thông tin SV C; nút `Hữu ích` đổi trạng thái tại chỗ không toast |
| Bước 3 | F5 | Cần GV hỗ trợ | **PASS** | SV B gõ D3 nhận thông báo AI không chắc chắn và tự chuyển GV; GV nhận ticket D3 tại `/inbox` và trả lời D4; SV B thấy câu trả lời có nhãn GV và đóng câu hỏi |
| Bước 4 | F7 | Điểm danh & phát biểu | **PASS** | GV mở `/attendance` buổi 10 lớp 1 (29/10), đánh vắng 2, muộn 1, ghi phát biểu +0,25 cho SV B; lưu hoàn tất buổi 10; SV B mở `/me` thấy **QT tăng từ 8,3 lên 8,5** |
| Bước 5 | F9 | Duyệt & công bố bài tập | **PASS** | SV B thấy BT03 "Đang chấm" không lộ điểm nháp; GV mở `/grading` thấy bài B bị cờ lệch 1,5 điểm; sửa tiêu chí 2 lên 2,5, duyệt và công bố; SV B thấy điểm 8,0 (đã trừ nộp muộn 0,5); **QT tăng từ 8,5 lên 8,7**; What-if CK 8,0 ra **8,3** |
| Bước 6 | F10 | Công thức điểm & Sổ điểm | **PASS** | Lớp 2 có banner chưa xác nhận công thức và khoá `Chốt điểm`; GV mở `/gradebook/scheme` điền D5 "Làm tròn đến 0,1", xác nhận công thức; về lại `/gradebook` lớp 2 gỡ banner và mở khoá `Chốt điểm`; lớp 1 giải trình hàng B đủ chi tiết |
| Bước 7 | F17 | Báo cáo lỗ hổng kiến thức | **PASS** | GV mở `/insights` lớp 1 thấy chủ đề A, B đứng đầu với các câu mẫu đã ẩn danh; bấm ghim tạo thread mới tại `/threads`; đổi sang lớp 2 thấy chủ đề C đứng đầu |

## Kiểm tra đối chiếu số liệu (SRS 4.1 v3)
| Mốc | Kỳ vọng | Thực tế | Đánh giá |
| --- | --- | --- | --- |
| 1. QT SV B lúc 09:20 ban đầu | **8,3** | 8,3 (TB 7,5 + phát biểu 0,75) | **PASS** |
| 2. QT SV B sau điểm danh (+0,25 phát biểu) | **8,5** | 8,5 (TB 7,5 + phát biểu 1,00) | **PASS** |
| 3. Điểm công bố BT03 của SV B | **8,0** | 8,0 (8,5 tiêu chí − 0,5 muộn) | **PASS** |
| 4. QT SV B sau công bố BT03 | **8,7** | 8,7 (TB 7,67 + phát biểu 1,00) | **PASS** |
| 5. What-if `/me` (nhập CK 8,0 sau công bố) | **8,3** | 8,3 (0,4 × 8,7 + 0,6 × 8,0 = 8,28) | **PASS** |

## Đánh giá trải nghiệm & luật giao diện
- **Chuyển vai trơn tru:** Đổi vai tức thì qua menu hồ sơ, không rò rỉ cookie hoặc lỗi văng trang.
- **Ranh giới dữ liệu:** Dữ liệu demo sau khi `Đặt lại dữ liệu demo` hoàn nguyên chính xác về trạng thái ban đầu để người thuyết trình có thể diễn lại nhiều lần.
- **Không có phản mẫu UI:** Không toast "Thành công", không spinner toàn trang, không thẻ KPI trang trí; màu đỏ chỉ đóng vai trò tín hiệu điểm dừng / cảnh báo.

## Đề nghị
Kịch bản demo 15 phút đạt trạng thái **PASS**, sẵn sàng tuyệt đối cho buổi thuyết trình với thầy hướng dẫn cuối tuần.
