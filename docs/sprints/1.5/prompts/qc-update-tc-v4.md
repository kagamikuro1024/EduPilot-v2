# Prompt cho `qc` — Viết thêm test case theo spec v4 (song song với dev)

BA vừa cập nhật spec lên Phiên bản 4 (`docs/sprints/1.5/spec/US.md` và `SRS.md` đã APPROVED) theo yêu cầu của chủ dự án:
- `01-AC4`: Form tạo thread đầy đủ (Tiêu đề, Chủ đề, Nội dung chi tiết, checkbox hỏi AI), quét PII (Dialog 2 lối), đăng xong chuyển hướng ngay sang `/threads/${id}`.
- `01-AC10` (MỚI): Reply Composer ở cuối trang chi tiết thread cho mọi vai trò, nút "Hỏi trợ lý AI" sinh câu trả lời Socratic có nguồn trích dẫn, quét PII khi gửi phản hồi.
- `02-AC8` (MỚI): Quyền kiểm duyệt Threads của GV/TA: Xác nhận, Chỉnh sửa inline câu trả lời AI (lưu thành `Đã được giảng viên sửa & xác nhận` (CORRECTED) kèm nút xem lại bản AI gốc), Loại khỏi tri thức (ẩn kèm Undo).
- `02-AC5`: Hộp thư `/inbox` phân định rõ 2 panel (cột danh sách và cột chi tiết) có viền, nền surface/surface-subtle, padding riêng biệt.
- `03-AC1`: Chấm bài `/grading/[submissionId]` phân định rõ 2 panel (cột bài nộp sinh viên và cột rubric chấm điểm) độc lập.

## Việc
1. Đọc kỹ `docs/sprints/1.5/spec/US.md` và `SRS.md` v4.
2. Cập nhật và bổ sung test case (hộp đen từ AC, không đọc code dev) vào các file:
   - `docs/sprints/1.5/qc/tc-US-PROTO-01.md` (thêm TC cho 01-AC4 và 01-AC10)
   - `docs/sprints/1.5/qc/tc-US-PROTO-02.md` (thêm TC cho 02-AC8 và 02-AC5 phân tách panel)
   - `docs/sprints/1.5/qc/tc-US-PROTO-03.md` (thêm TC cho 03-AC1 phân tách panel)
3. Cập nhật script kiểm tra tự động nếu cần trong `docs/sprints/1.5/qc/scripts/`.

Commit chỉ `docs/sprints/1.5/qc/**`.
Xong tóm tắt số TC mới bổ sung và dừng, chờ Dev xong handoff thì chạy kiểm thử toàn bộ.
