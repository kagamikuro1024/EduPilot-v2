# Prompt cho `dev` — Thi công theo spec v4 (Luồng Threads + Phân định Panel)

BA đã cập nhật spec kỹ lưỡng tại `docs/sprints/1.5/spec/US.md` và `SRS.md` (Phiên bản 4, PM đã APPROVED).

Bạn thi công hoàn thiện theo đúng spec:

## 1. Luồng Threads (`ThreadsScreen.tsx`, `ThreadDetail.tsx`, `src/mock/threads.ts`):
- **Form tạo thread mới**:
  - Tiêu đề câu hỏi (Input rõ ràng, có label + placeholder)
  - Chủ đề (Select chọn trong `THREAD_TOPICS`)
  - Nội dung chi tiết câu hỏi (Textarea)
  - Checkbox "Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" (mặc định bật)
  - Nút "Đăng câu hỏi" (Primary red button)
  - Quét PII trên tiêu đề + nội dung trước khi đăng (hiện `PIIChannelDialog` 2 lối nếu có dữ liệu cá nhân)
  - Sau khi đăng thành công: **chuyển hướng ngay lập tức sang `/threads/${id}`**!
- **Chi tiết thread (`/threads/[id]`)**:
  - Khối câu hỏi gốc: người hỏi, vai trò, thời gian, chủ đề, nội dung câu hỏi.
  - Khối câu trả lời AI (kèm citations):
    - Nhãn `Chờ xác nhận` (pending) / `Đã được giảng viên xác nhận` (verified) / `Đã được giảng viên sửa & xác nhận` (corrected).
    - Quyền kiểm duyệt cho Giảng viên / Trợ giảng:
      - `Xác nhận`: chuyển trạng thái thành `Đã được giảng viên xác nhận`.
      - `Chỉnh sửa`: bấm vào mở ô textarea inline chứa nội dung câu trả lời AI, GV/TA sửa xong bấm `Lưu và xác nhận` -> trạng thái thành `Đã được giảng viên sửa & xác nhận` (CORRECTED), hiển thị nội dung đã sửa kèm nút/chi tiết "Xem câu trả lời AI gốc" để đối chiếu!
      - `Loại khỏi tri thức`: ẩn câu trả lời, có dòng Hoàn tác.
  - Khối thảo luận / các phản hồi của người dùng khác.
  - **Ô trả lời thảo luận (Reply Composer) ở cuối trang**:
    - Cho phép viết phản hồi và đăng vào thread.
    - Có nút "Hỏi trợ lý AI" để sinh câu trả lời AI Socratic nếu muốn.
    - Quét PII khi gửi phản hồi.

## 2. Phân định rõ các Panel giao diện (FR-X11 trong SRS v4):
- Trên các màn chia cột (Hộp thư `/inbox`, Chấm bài `/grading/[submissionId]`, Threads `/threads/[id]`, Cài đặt `/settings/*`, Insights `/insights`):
  - Cột danh sách và cột chi tiết / các khối lớn phải là các **panel/khung làm việc độc lập**.
  - Có đường viền phân định (`1px solid var(--ep-rule)`), nền thẻ (`var(--ep-surface)` trắng) hoặc nền vùng phụ (`var(--ep-surface-subtle)`), bo góc `var(--ep-radius-md)` (8px), padding phù hợp.
  - Tránh để mọi thứ phẳng lỳ trôi trên nền canvas chỉ có đường kẻ mỏng.

## 3. Kiểm tra trước khi báo xong:
- `pnpm -C frontend lint` và `pnpm -C frontend build` xanh (0 lỗi).
- `bash scripts/ui-antipatterns.sh` sạch (10/10 check xanh).
- Cập nhật ảnh chụp vào `docs/sprints/1.5/shots/` và cập nhật handoff.
- Commit theo quy ước. Báo xong để PM giao QC test.
