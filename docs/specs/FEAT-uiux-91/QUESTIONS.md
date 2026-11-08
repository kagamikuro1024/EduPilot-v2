# QUESTIONS FEAT-uiux-91
Trạng thái spec: **DRAFT**. Q-UX91-1…5 đã chốt (chủ dự án, 2026-11-08, "Đồng ý hết", ghi ở `docs/sprints/9.1/plan.md`). Các câu từ 6 trở đi do BA nêu khi viết spec.

| # | Câu hỏi | Phương án BA đề xuất | Trả lời của chủ dự án | Ngày |
| --- | --- | --- | --- | --- |
| Q-UX91-1 | Nhãn điều hướng `Threads`, `Insights`, `Analytics` giữ tiếng Anh hay đổi? | Nhãn nav = tiêu đề trang, tiếng Việt: Hỏi đáp công khai / Lỗ hổng kiến thức / Số liệu lớp học | Đồng ý theo đề xuất | 2026-11-08 |
| Q-UX91-2 | Hôm nay giảng viên có chia nhóm không, ngưỡng "Gấp" là gì? | Nhóm Gấp / Hôm nay / Khi rảnh; Gấp = quá hạn hoặc cờ rủi ro; Khi rảnh = phát hiện giống bài, bài chưa khớp | Đồng ý. PM làm rõ (TL-2): Gấp dùng `urgency == overdue` của backend, không đặt ngưỡng 72 giờ mới ở client | 2026-11-08 |
| Q-UX91-3 | Cột "Trạng thái" sổ điểm ("Đủ") nghĩa là gì, gọi gì? | "Đủ" = đã có đủ điểm thành phần → **Đủ điểm / Thiếu điểm**; BA đề xuất ẩn cột khi không có hàng thiếu | Đồng ý; việc ẩn cột giao BA đề xuất trong spec (US-UX91-16 AC4) | 2026-11-08 |
| Q-UX91-4 | "Có mặt" mặc định ở điểm danh dùng chấm mực thay chấm đỏ? | Có: đỏ chỉ cho Vắng / Vắng phép / Muộn | Đồng ý | 2026-11-08 |
| Q-UX91-5 | Trang giới thiệu được nêu số đo từ môi trường mô phỏng? | Có, kèm dòng "đo trên dữ liệu mô phỏng" | Đồng ý | 2026-11-08 |
| Q-UX91-6 | Có thêm `meta` "Chỉ bạn xem được" dưới tiêu đề chat riêng không? | **Đóng, không cần.** Chat thật (`RealChat`) không có câu mô tả; lời giới thiệu phiên trống (`intro.ts` `body`) đã nói "Phiên chat này chỉ bạn đọc được". Giữ nguyên, không thêm `meta` | — (BA tự đóng, không đụng hành vi) | 2026-11-08 |
| Q-UX91-7 | Hiển thị điểm có làm tròn tới 2 chữ số thập phân không? | **Đóng.** PM chấp nhận P-91-1: không làm tròn khi hiển thị; đệm `0` tới số chữ số lẻ lớn nhất của cột | — (PM, P-91-1) | 2026-11-08 |
| Q-UX91-8 | "Cờ rủi ro" của Q-UX91-2 là gì trong dữ liệu `GET /…/today`? | `urgency = high` (bài AI "cần xem kỹ", email chưa khớp MSSV, yêu cầu vào lớp) cùng vào Gấp với `overdue`; mọi `normal` còn lại vào Hôm nay, trừ `EXAM_SIMILARITY` và `UNMATCHED_SUBMISSION` vào Khi rảnh. Nếu PM hiểu "cờ rủi ro" là thứ khác (vd. sinh viên AT_RISK ở khối "Lớp cần chú ý"), BA đổi AC1 của US-UX91-10 | | |
| Q-UX91-9 | "Lớp đầu tiên có dữ liệu" (US-UX91-06 AC2) định nghĩa thế nào để chọn bằng cách S mà không thêm API? | Lớp đầu tiên theo thứ tự `GET /me/courses` mà `GET /me/today` có ≥ 1 mục thuộc lớp đó; không có lớp nào → lớp đầu danh sách. Nếu Tech Lead thấy tốn thêm một lời gọi ở lúc đăng nhập thì giữ nguyên hành vi hiện tại ("lớp đầu danh sách") và chỉ làm AC1 (nhớ lớp) | | |
| Q-UX91-10 | Người còn cookie phiên nhưng mất cờ "đã từng đăng nhập" (xoá `localStorage`) sẽ thấy trang giới thiệu thay vì Hôm nay ở `/` (US-UX91-21). Chấp nhận? | Chấp nhận (TL-2: chỉ phải bấm `Đăng nhập`, `/login` vẫn thử cookie); đổi lại hết lỗi 401 ở console mọi lần khách vào trang công khai | | |
| Q-UX91-11 | Chữ trang giới thiệu hiện nói "Tên và MSSV được che trước **mọi** lời gọi AI". Báo cáo sprint 6 ghi recall phát hiện 0,960 trên 200 mẫu tự soạn và "tên người ngoài danh sách lớp chưa bắt được". Sửa chữ cho khớp nguồn? | Sửa: "146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp" + dòng "Đo trên dữ liệu mô phỏng" (US-UX91-08 AC6). Chữ cũ là khẳng định mạnh hơn bằng chứng, rủi ro khi vào quyển đồ án | | |
| Q-UX91-12 | Số ảnh mốc mới (landing, attendance, hom-nay-gv, grading-queue, grading-review, insights, observability, documents, me…) tăng thời gian CI. Thêm hết hay chọn? | Thêm theo đợt khi story đó làm xong; ưu tiên `landing`, `attendance`, `hom-nay-gv`, `grading-review`. QC quyết số lượng cuối | | |

## Đính chính findings (không phải câu hỏi, ghi để PM biết)
- UX-5: khối "Lớp cần chú ý" **đã có** trong `StaffToday.tsx` nhưng nằm dưới danh sách; story chỉ chuyển nó lên cột phải ≥ 1440 px. Dữ liệu `attention` là **sinh viên** AT_RISK, không phải "5 câu chờ quá 72 giờ" như mẫu 3.B ghi (TL-2).
- UX-18 (chiều rộng nội dung): DESIGN §2 đã quy định "đọc / form 960 px", "bảng fluid"; findings coi là chưa nhất quán là **sai**. Spec chỉ dùng khoảng trống bên phải của Hôm nay (≥ 1440 px). UX-18 ở các màn khác bỏ.
- UX-25(b), (c): bộ chọn lớp có dòng buổi học tiếp theo cần dữ liệu theo lớp (ngoài S); seed lớp 2 đồng nhất là việc của dev / QC. Ghi Nợ, không có story.
