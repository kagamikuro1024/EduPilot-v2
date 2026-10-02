# Prompt cho `ba` — Cập nhật đặc tả chi tiết Luồng Threads và Phân định Panel giao diện

Chủ dự án vừa xem qua giao diện và yêu cầu chỉnh sửa 2 điểm quan trọng (đã ghi trong `docs/sprints/1.5/proposals.md` #15 và #16, PM đã CHẤP NHẬN). Bạn cần đặc tả kỹ lưỡng vào spec trước khi giao cho Dev thi công.

## Đọc
- `docs/team/CONTEXT.md`
- `docs/sprints/1.5/proposals.md` (Proposal #15 và #16)
- Mã nguồn cũ trong `legacy/frontend/src/app/threads/` và `legacy/backend-java/.../Forum*.java`
- `docs/design/DESIGN.md` §10, §14.3, §14.4

## Nhiệm vụ cập nhật `docs/sprints/1.5/spec/US.md` và `docs/sprints/1.5/spec/SRS.md`:

### 1. Đặc tả chi tiết Luồng Threads (Proposal #15)
- **Tạo thread mới**:
  - Không dùng 1 ô composer đơn lẻ cắt 80 ký tự làm tiêu đề.
  - Phải có form tạo thread gồm:
    - Tiêu đề câu hỏi (Input rõ ràng, bắt buộc)
    - Chủ đề (Select chọn trong danh sách chủ đề của môn)
    - Nội dung câu hỏi chi tiết (Textarea, bắt buộc)
    - Tùy chọn "Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" (mặc định bật)
  - Vẫn quét PII trước khi đăng. Nếu có dữ liệu cá nhân -> hiện `PIIChannelDialog` đúng 2 lối.
  - Sau khi đăng thành công: **chuyển hướng ngay lập tức sang `/threads/${id}`** của thread vừa tạo!
- **Chi tiết thread (`/threads/[id]`)**:
  - Khối câu hỏi gốc: người hỏi, vai trò, thời gian, chủ đề, nội dung câu hỏi.
  - Khối câu trả lời AI:
    - Trạng thái: `Chờ xác nhận` (pending) / `Đã được giảng viên xác nhận` (verified) / `Đã được giảng viên sửa & xác nhận` (corrected).
    - Giải thích Socratic kèm danh sách trích dẫn nguồn (citations) mở rộng xem được.
    - Quyền kiểm duyệt cho Giảng viên / Trợ giảng:
      - `Xác nhận`: chuyển trạng thái thành `Đã được giảng viên xác nhận`.
      - `Chỉnh sửa`: mở ô sửa inline chứa nội dung câu trả lời AI, giảng viên sửa xong bấm `Lưu và xác nhận` -> trạng thái thành `Đã được giảng viên sửa & xác nhận` (CORRECTED), hiển thị nội dung đã sửa kèm nút/chi tiết "Xem câu trả lời AI gốc" để đối chiếu!
      - `Loại khỏi tri thức`: ẩn câu trả lời, có dòng Hoàn tác.
  - Khối thảo luận / các phản hồi của người dùng khác (sinh viên, giảng viên).
  - **Ô trả lời thảo luận (Reply Composer) ở cuối trang**:
    - Mọi vai trò đều có thể viết phản hồi và đăng vào thread.
    - Có nút "Hỏi trợ lý AI" để sinh câu trả lời AI Socratic nếu muốn.
    - Quét PII khi gửi phản hồi.

### 2. Đặc tả Phân định rõ các Panel giao diện (Proposal #16)
- Quy định rõ cấu trúc giao diện: các màn chia cột (Hộp thư `/inbox`, Chấm bài `/grading/[submissionId]`, Threads `/threads/[id]`, Cài đặt) và các khối nội dung lớn trên trang phải là các **panel/khung làm việc độc lập**, có đường viền phân định (`1px solid var(--ep-rule)`), nền thẻ (`var(--ep-surface)`) hoặc nền vùng phụ (`var(--ep-surface-subtle)`), bo góc và padding rõ ràng, không để mọi thứ phẳng lỳ trôi trên nền giấy trắng chỉ có đường kẻ mỏng.

## Tiêu chí nghiệm thu (AC)
- Bổ sung / cập nhật các AC trong `US.md` tương ứng cho US-PROTO-01 (Threads SV), US-PROTO-02 (Inbox, Threads GV), US-PROTO-03 (Grading detail).
- Mỗi AC có bước Given/When/Then và lệnh/thao tác kiểm tra cụ thể.

Tăng `SRS.md` lên Phiên bản 4. Commit chỉ `docs/sprints/1.5/spec/**`.
Xong tóm tắt ≤ 8 dòng và dừng.
