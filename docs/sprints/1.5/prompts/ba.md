# Prompt cho `ba` — spec prototype giao diện

1. Đọc `docs/team/CONTEXT.md` (mới — bối cảnh cả đội), rồi `docs/sprints/1.5/plan.md`.
2. Viết `docs/specs/FEAT-prototype-ui/` gồm `US.md`, `SRS.md`, `QUESTIONS.md`. Nhánh `sprint/1.5-mock-ui` (đã có, đừng đổi nhánh).

## Nội dung
- Một feature, **US-PROTO-00 … US-PROTO-04** đúng nhóm route trong `plan.md`.
- Mỗi route: trỏ mục hợp đồng (`DESIGN.md` §14.x hoặc `INTEGRATION.md` mục 2 cho route ngoài §14) — **không chép lại** §14; chỉ ghi phần §14 chưa nói mà prototype cần: dữ liệu mô phỏng cụ thể (đủ để màn trông thật: số lượng, tên, trạng thái, mốc thời gian quanh "bây giờ" = Thứ Năm 01/10/2026 09:20, tuần 5), **tương tác giả lập** (bấm gì → trạng thái đổi ra sao; ví dụ chat stream + nguồn tham khảo + dòng "Đã ẩn 2 thông tin cá nhân"; Threads gửi câu riêng tư → hộp thoại hai lối; điểm danh bàn phím + Hoàn tác; duyệt bài sửa điểm + lệch > 1 điểm; công thức điểm chặn xác nhận tới khi gỡ chỗ mơ hồ; What-if ở `/me`…), và trạng thái rỗng / lỗi nào phải xem được (có công tắc minh hoạ nếu cần).
- Dữ liệu phải **khớp `docs/DEMO_SCRIPT.md`** (cùng câu hỏi, cùng sinh viên A–D, cùng bài tập, cùng lớp) để chủ dự án đi trọn kịch bản trên prototype.
- AC kiểm được bằng tay trên trình duyệt hoặc lệnh: route mở được với vai đúng; vai sai thấy màn chặn quyền; một hành động chính mỗi vùng; 375 px dùng được cho màn SV + `/attendance` + `/inbox`; sinh viên không thấy từ kỹ thuật AI; `ui-antipatterns.sh` sạch. Có AC nhánh lỗi + AC phân quyền mỗi US.
- SRS rút gọn: mục 1, 2 (ma trận vai trò × route), 3, 4, 7, 9, 11.
- Quyền / hành vi sản phẩm: lấy từ PRD; chỗ PRD không nói → QUESTIONS kèm đề xuất. Ưu tiên tốc độ: chủ dự án thuyết trình cuối tuần, chỉ hỏi điều thật sự chặn.

## Khi xong
Commit `sprint 1: spec FEAT-prototype-ui` (chỉ `docs/specs/**`), tóm tắt ≤ 8 dòng, dừng.
