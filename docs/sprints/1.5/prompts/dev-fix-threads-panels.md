# Prompt cho `dev` — Hoàn thiện luồng Threads và Phân định rõ các Panel giao diện

Chủ dự án vừa xem qua giao diện và yêu cầu chỉnh sửa 2 điểm mấu chốt (xem `docs/sprints/1.5/proposals.md` #15 và #16, PM đã CHẤP NHẬN):

## 1. Hoàn thiện luồng Threads (Proposal #15)
Tham khảo mã cũ trong `legacy/frontend/src/app/threads/` và `legacy/backend-java/.../Forum*.java`, cùng `DESIGN.md` §14.3, §14.4:
- **Tạo thread mới (`ThreadsScreen.tsx`)**:
  - Không chỉ dùng 1 ô Composer cắt 80 ký tự. Cần form tạo thread rõ ràng gồm:
    - Tiêu đề câu hỏi (Input rõ ràng, có label + placeholder)
    - Chủ đề (Select chọn trong `THREAD_TOPICS`)
    - Nội dung chi tiết câu hỏi (Textarea)
    - Tùy chọn "Nhờ AI trả lời gợi ý ngay" (mặc định bật)
  - Vẫn chạy kiểm tra PII trước khi đăng (hiện `PIIChannelDialog` nếu có dữ liệu cá nhân).
  - Khi đăng thành công: lưu vào state (`NEW_THREADS_KEY`) và **chuyển hướng ngay sang `/threads/${id}`**!
- **Chi tiết thread (`ThreadDetail.tsx`)**:
  - Khối câu hỏi gốc của tác giả rõ ràng (tác giả, vai trò, thời gian, chủ đề, nội dung).
  - Khối câu trả lời AI:
    - Nếu còn chờ duyệt: nhãn `Chờ xác nhận`.
    - Các nút cho Giảng viên / Trợ giảng:
      - `Xác nhận`: chuyển trạng thái thành `Đã được giảng viên xác nhận`.
      - `Chỉnh sửa`: **phải hoạt động!** Bấm vào mở ô textarea inline chứa nội dung câu trả lời AI, giảng viên chỉnh sửa rồi bấm `Lưu và xác nhận` -> trạng thái thành `Đã được giảng viên sửa & xác nhận`, hiển thị nội dung đã sửa kèm nút/chi tiết "Xem câu trả lời AI gốc" để người xem đối chiếu!
      - `Loại khỏi tri thức`: ẩn câu trả lời, có thanh Hoàn tác.
    - Trích dẫn nguồn (citations) mở rộng xem chi tiết được.
  - Khối thảo luận / các câu trả lời của sinh viên / giảng viên khác.
  - **Ô trả lời thảo luận (Reply Composer) ở cuối trang**:
    - Bất kỳ ai (sinh viên, giảng viên) đều có thể nhập phản hồi và gửi vào thread.
    - Có nút "Hỏi trợ lý AI" để AI tạo thêm câu trả lời gợi ý Socratic kèm nguồn trích dẫn nếu muốn.
    - Kiểm tra PII khi gửi phản hồi.

## 2. Phân định rõ các Panel giao diện (Proposal #16)
Chủ dự án phản ánh: "UI này nó không chia rõ các panel nên cảm giác view hơi khó phân biệt các phần".
Hiện tại do áp dụng "không card" quá cứng nhắc nên mọi thứ đều là chữ trên nền phẳng chỉ có đường kẻ 1px, khiến các phần dính vào nhau và khó định vị thị giác.
Yêu cầu:
- Các màn hình chia cột (Split views) như:
  - Hộp thư hỗ trợ (`/inbox`): cột danh sách bên trái và cột chi tiết bên phải phải là 2 panel riêng biệt, có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)` hoặc `var(--ep-surface-subtle)`, bo góc `var(--ep-radius-md)`, padding phù hợp.
  - Chấm bài (`/grading/[submissionId]`): cột bài làm của sinh viên bên trái và cột rubric chấm điểm bên phải phải là 2 panel/khung làm việc phân định rõ ràng.
  - Chi tiết Threads (`/threads/[id]`): khối câu hỏi, khối trả lời AI, và khối soạn thảo trả lời phải là các panel có nền/viền phân tách rành mạch.
  - Cài đặt (`/settings/llm`, `/settings/integrations`): các khối cấu hình phải nằm trong các panel/card sạch sẽ, có viền phân định.
- Sử dụng đúng bảng token màu:
  - Nền canvas trang: `var(--ep-paper)`
  - Nền panel / vùng làm việc: `var(--ep-surface)` (trắng)
  - Nền vùng phụ / danh sách bên: `var(--ep-surface-subtle)`
  - Viền phân định: `1px solid var(--ep-rule)`
  - Bo góc: `var(--ep-radius-md)` (8px)

## Yêu cầu kiểm tra
- Chạy `pnpm -C frontend build`, `pnpm -C frontend lint` và `bash scripts/ui-antipatterns.sh` đảm bảo XANH.
- Chụp ảnh cập nhật vào `docs/sprints/1.5/shots/`.
- Commit và cập nhật handoff.
