# Sprint 1.5 — Báo cáo nghiệm thu Prototype Giao diện

Mục tiêu: Hoàn thành prototype giao diện tương tác toàn bộ tính năng phục vụ thuyết trình với thầy hướng dẫn cuối tuần · Kết quả: **5/5 story PASS · DEMO 15 phút PASS 100%** · Nhánh `sprint/1.5-mock-ui`

| Story | Phạm vi | Trạng thái | QC | Ghi chú |
| --- | --- | --- | --- | --- |
| US-PROTO-00 | Nền tảng prototype, layout, /login, phân quyền | PASS | `report-US-PROTO-00.md` (17 TC) | 4 vai, chọn SV A–D, cookie phiên, ma trận 128/128 quyền, sửa BUG-1..3 |
| US-PROTO-01 | Giao diện Sinh viên (Chat, Threads, Luyện đề, Thư viện, Lịch, Kết quả, Bài tập, Join) | PASS | `report-US-PROTO-01.md` (38 TC) | Form tạo thread đầy đủ, chuyển hướng sang `/threads/${id}`, reply composer, hỏi AI, quét PII 2 lối, stream chữ, What-if |
| US-PROTO-02 | Giao diện GV/TA vận hành lớp (Hôm nay, Hộp thư, Sinh viên, Điểm danh, Thành viên) | PASS | `report-US-PROTO-02.md` (31 TC) | Hộp thư 2 panel độc lập, điểm danh phím tắt + giả lập mất mạng, kiểm duyệt Threads GV/TA (sửa inline CORRECTED, xác nhận, loại bỏ) |
| US-PROTO-03 | Giao diện Đánh giá (Sổ điểm, Công thức điểm, Chấm bài, Ngân hàng câu hỏi, Tài liệu) | PASS | `report-US-PROTO-03.md` (29 TC) | Màn chấm bài 2 panel độc lập (doc 55% & rubric 45%), lệch điểm > 1 cảnh báo, duyệt & công bố BT03 = 8,0, xác nhận công thức lớp 2 |
| US-PROTO-04 | Giao diện Hiểu lớp + Hệ thống (Insights, Analytics, Quan sát AI, Cấu hình, Quản trị) | PASS | `report-US-PROTO-04.md` (23 TC) | Phân định panel /settings, /insights, bảo vệ thông tin cá nhân, audit log Drawer |
| US-PROTO-DEMO | Đi trọn kịch bản demo 15 phút (F2→F3→F5→F7→F9→F10→F17) | PASS | `report-US-PROTO-DEMO.md` | Đi trọn 7 luồng trong một trình duyệt bằng cách đổi vai; điểm quá trình SV B khớp: 8,3 → 8,5 → 8,7; What-if 8,3 |

## Thay đổi lớn theo góp ý của chủ dự án trong sprint
1. **Luồng Threads hoàn chỉnh (Proposal #15)**: Bám sát mã cũ `legacy/` và `DESIGN.md` §14.3–14.4: form tạo thread gồm Tiêu đề, Chủ đề, Nội dung chi tiết, checkbox hỏi AI; quét PII 2 lối; đăng xong chuyển hướng ngay sang `/threads/${id}`; chi tiết thread có Reply Composer ở cuối trang cho mọi vai trò; GV/TA có nút Chỉnh sửa inline câu trả lời AI (lưu thành `CORRECTED` kèm xem bản gốc), Xác nhận, Loại khỏi tri thức.
2. **Phân định rõ các Panel giao diện (Proposal #16, FR-X11)**: Khắc phục cảm giác phẳng lỳ, khó phân biệt khu vực. Các màn chia đôi (Hộp thư `/inbox`, Chấm bài `/grading/[submissionId]`, Threads `/threads/[id]`, Cài đặt `/settings/*`) và các khối lớn trên trang được đóng gói thành các panel độc lập có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)` hoặc `var(--ep-surface-subtle)`, bo góc `var(--ep-radius-md)`, padding phù hợp.
3. **Mốc thời gian và điểm số chuẩn hóa**: Tuần 10, ngày 29/10/2026 09:20; điểm BT03 công bố = 8,0 (trừ muộn 0,5), QT SV B = 8,7, What-if CK 8,0 = 8,3 (Proposals #12, #13).

## Số liệu nghiệm thu
- Tổng test case: **138 TC**, 100% PASS, 0 FAIL.
- Kiểm tra tĩnh: `tsc --noEmit` xanh, `eslint .` (0 lỗi, 0 cảnh báo), `next build` biên dịch sạch 31 route.
- Phản mẫu UI (`scripts/ui-antipatterns.sh`): **10/10 check XANH**.
- Ảnh chụp nghiệm thu: 24 ảnh tại `docs/sprints/1.5/shots/`.

## Trạng thái bàn giao
Prototype đã sẵn sàng 100% để khởi động và trình chiếu trước thầy hướng dẫn.
Mọi dữ liệu mô phỏng được lưu tại `localStorage` (`ep_demo_state`), có nút `Đặt lại dữ liệu demo` trong menu hồ sơ để reset trạng thái sạch bất cứ lúc nào.
