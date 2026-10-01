---
version: 1
slug: "frontend-src-app"
primary_target: "frontend/src/app"
related_targets: []
---

## Scope
EduPilot prototype UI — toàn bộ route của app (DESIGN.md §14 + màn tài khoản/quản trị), dữ liệu mô phỏng, tương tác giả lập. Mode: Operate. Mở rộng trong thế giới đã có (Red Thread / Academic Instrument, docs/design/DESIGN.md) — không phát minh thế giới mới.

## Audience and task
Chủ dự án trình diễn cho thầy hướng dẫn: đổi vai Sinh viên / Trợ giảng / Giảng viên / Admin, đi trọn kịch bản docs/DEMO_SCRIPT.md. Mỗi màn: một việc chính nhận ra trong ~3 giây.

## Direction contract
THESIS: Lớp học là một văn bản đang được chấm: giấy trắng, mực đậm, một sợi chỉ đỏ đánh dấu chỗ đang đứng, việc cần làm, điều đã được sửa/xác nhận. Từ chối dashboard SaaS thẻ KPI và chatbot có thêm menu.
OWN-WORLD: nền giấy ấm oklch 99%, mực 19%, đường kẻ 1px, bo 6–8px, không bóng ngoài popover/dialog; Be Vietnam Pro một họ; đỏ < 8% diện tích; xanh lá = đã xác nhận, hổ phách = cần xem, chấm tròn trạng thái thay badge; hàng ngăn bằng đường kẻ, không card.
STORY: sinh viên thấy một việc nên làm ngay, hỏi riêng an toàn (thấy "Đã ẩn 2 thông tin cá nhân"), hiểu điểm của mình bằng phép tính tuyến tính; giảng viên thấy danh sách việc cần quyết, xử lý hộp thư, điểm danh 30 SV dưới 60 giây, duyệt bài AI chấm nháp rồi tự công bố.
FIRST VIEWPORT: "/" theo vai — sinh viên: lời chào + ngày, một khối gợi ý (tiêu đề 20px, lý do, thời lượng, nút đỏ duy nhất), dòng thời gian hôm nay; giảng viên: "5 việc cần xử lý" + ActionList hàng kẻ, mỗi hàng một hành động, nút đỏ chỉ ở việc khẩn nhất.
FORM: app shell sidebar 216px có nhãn + topbar 56px (lớp, tìm ⌘K, chuông, vai trò); mobile: bottom nav 5 mục + "Thêm".
SIGNATURE: Red Thread Transition — chọn việc ở "Hôm nay" thì một vạch đỏ mảnh chạy từ hàng tới tiêu đề trang đích (< 300 ms, tôn trọng reduced-motion). Phụ: chữ AI chảy từng đoạn với nguồn tham khảo mở tại chỗ.

## Constraints
Chỉ token --ep-* và primitive ở frontend/src/shared/; scripts/ui-antipatterns.sh sạch; sinh viên không thấy từ kỹ thuật AI; không gamify; dữ liệu ghi rõ là mô phỏng.
