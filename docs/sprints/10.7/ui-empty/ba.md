# ui-empty — BA: câu trong trạng thái rỗng có trong SRS / US (2026-11-14)

Nguồn: `docs/sprints/10.7/ui-empty/plan.md` (chủ dự án) + luật chữ UI (CLAUDE.md 09/10/2026). Luật chốt: **rỗng = tiêu đề ngắn + tối đa 1 hành động; không câu giải thích**. BA đã vá spec cùng commit này (không cần proposal riêng — theo plan; PM ghi số nếu cần). Dev bỏ câu ở mã đúng theo cột «Sau».

## A. Đã vá

| # | Spec | Trước | Sau |
|---|---|---|---|
| 1 | FEAT-uiux-91 US-UX91-05 (tiêu đề + AC1) `/inbox` | tiêu đề + «Khi AI không đủ chắc chắn…»; nút `ghost` «Xem tất cả» | chỉ tiêu đề; bỏ «Xem tất cả» (trùng tab «Tất cả»); căn giữa vùng danh sách |
| 2 | uiux-91 AC3 `/grading` | «Chưa có bài nộp.» + «Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên.» + `Tải bài lên` | «Chưa có bài nộp.» + `Tải bài lên` |
| 3 | uiux-91 AC6 điểm công bố (sinh viên) | «Chưa có điểm nào được công bố.» + nêu nơi sẽ xuất hiện | chỉ tiêu đề |
| 4 | uiux-91 bảng #56 `StudentToday` | «Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp…» | «Bạn chưa vào lớp nào.» + `Nhập mã` (cú pháp mã ở `placeholder`) |
| 5 | FEAT-ui-foundation SRS 7.2 (ô `empty`, `rỗng` L/E/R) + US AC8 chuông | «nói vì sao rỗng + đúng 1 hành động»; «Chưa có thông báo. Khi có việc cần bạn, nó sẽ hiện ở đây.» | «tiêu đề ngắn + ≤ 1 hành động, không thân chữ» (đếm `button, a` ≤ 1); «Chưa có thông báo.» |
| 6 | ui-foundation SRS 108 + US AC11 `no-backend` | «Tính năng này đang được xây ở giai đoạn P3 — Chat riêng. Khi xong…» (+ test đòi tên phase) | «Tính năng chưa mở.» + `Về Hôm nay`; test: không chứa tên phase |
| 7 | ui-foundation US AC10 + llm-gateway US AC10 màn chặn quyền | tiêu đề + câu lý do («Trang này dành cho …») | chỉ «Bạn không có quyền xem màn này» + `Về Hôm nay` |
| 8 | FEAT-course-foundation SRS/US (thông báo, danh sách lớp) | «Chưa có thông báo. Khi có việc…»; «Chưa có lớp nào. Mở lớp đầu tiên.» | «Chưa có thông báo.»; «Chưa có lớp nào.» + `Mở lớp` |
| 9 | FEAT-private-chat-pii SRS/US Threads rỗng | «Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên.» + `Đặt câu hỏi` | «Chưa có câu hỏi nào.» + `Đặt câu hỏi` |
| 10 | FEAT-weekly-exam SRS 7.2 + US (ngân hàng câu hỏi, `/exams`) | «Chưa có câu hỏi nào. Tạo câu đầu tiên hoặc nhờ AI gợi ý nháp.»; «Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi.» | «Chưa có câu hỏi nào.» + `Tạo câu hỏi`; «Chưa có bài thi nào.» + `Tạo bài thi` (gợi ý nháp AI không còn ở trạng thái rỗng) |
| 11 | FEAT-llm-gateway US `/settings/llm` bảng rỗng | «Đang dùng cấu hình mặc định của máy chủ. Thêm nhà cung cấp để thay đổi.» | «Đang dùng cấu hình mặc định.» + `Thêm nhà cung cấp` |

## B. Giữ nguyên (đã đúng luật, hoặc không phải EmptyState)

- Tiêu đề ngắn có sẵn: gradebook AC9 (a)(b), crm-360 «Không có sinh viên nào khớp.» + `Bỏ lọc`, gradebook «Lớp chưa có quy chế.» + `Tải quy chế`, p10-eval / decision-model «`EmptyState` một dòng», Insights «`EmptyState` + một nút», `/analytics` «Chưa có bài thi công bố».
- «Việc nên làm tiếp» (một hành động + **lý do**) là hợp đồng DESIGN §14.1, không phải EmptyState; câu lý do `JOIN_CODE` / `VERIFY_EMAIL` ở FEAT-course-foundation US giữ — chủ dự án muốn bỏ thì nói.
- Chat trả lời «Hệ thống chưa có dữ liệu điểm danh của bạn.» là lời của trợ lý, không phải trạng thái rỗng.
- `PageHeader meta` lặp mã lớp: không spec nào bắt; FEAT-uiux-91 SRS 147 đã nói «meta chỉ dữ liệu» — chỉ sửa mã.

## C. Việc của vai khác (ngoài vùng BA)

- Tech Lead / PO: `DESIGN.md` §15 + `UX.md` («empty = câu dạy bước kế») và mô tả mặc định `PageState` — sửa theo luật mới.
- QC: TC theo spec trên (đặc biệt `shell.spec` `no-backend` / `forbidden`, ô `empty` của ma trận `/dev/ui`, `inbox` AC1, ảnh mốc); thêm TC «không có thân chữ trong EmptyState».
- Dev: `scripts/ui-antipatterns.sh` chặn `<EmptyState …>chữ</EmptyState>`.
