# Dev — đợt 2 (chữ) · sprint 9.1 pha 2

Commit: `4b6389f1` (chung với đợt 1). Stack kiểm như `dev-wave-1.md`.

## Làm được
| Story | Nội dung |
| --- | --- |
| US-01 | Xoá câu mô tả tĩnh dưới tiêu đề ở AdminCourses, AdminUsers, CoursePanel, InvitePanel, Analytics, ExamItems, ExamsHome, Insights, PracticeHistory, PracticeScreen, QuestionBank, ClassSettings, MembersView, RealMe, RealStudents, RealProfile, JoinScreen, Observability, RouteStub, `shell/intro.ts`. Chỗ thật sự cần giải thích → `title` / `hint` trên nhãn hoặc nút. `RealAttendance` gộp ngày giờ buổi vào tiêu đề. Phép 23 mới ở `scripts/ui-antipatterns.sh` ("Câu mô tả tĩnh dưới tiêu đề") + ca selftest (23/23). |
| US-05 | Trạng thái rỗng có đường đi: Hộp thư ("Chưa có câu hỏi nào cần thầy/cô xử lý." + nút "Xem tất cả"; ẩn vùng chi tiết khi rỗng và chưa chọn phiếu), Chấm bài ("Chưa có bài nộp." + Tải bài lên), Sổ điểm (`OFFICIAL_NOTE` chỉ khi có công thức và dữ liệu). |
| US-06 | Lớp mặc định lần đầu của GV/TA ≥ 2 lớp = lớp đầu tiên có dữ liệu trong `GET /me/today` (`useFirstCourseWithData`); chỉ tính ở `/` để màn khác (kể cả màn bị chặn quyền) không gọi thêm API. |
| US-07 | Đổi nhãn: "Hỏi đáp công khai" (thay Threads; thanh dưới 375 px dùng "Hỏi đáp"), "Lỗ hổng kiến thức" (Insights), "Số liệu lớp học" (Analytics); nav, metadata trang, thông báo, `Landing`, `DemoThreads` / `DemoThreadDetail` (đổi chữ duy nhất). |

## Test đã sửa theo spec
- `shell.spec.ts` bảng NAV (nhãn mới), `activity.spec.ts` dòng nhãn.
- `escalation.spec.ts` ("Chưa có câu hỏi nào cần thầy/cô xử lý."), `assignment-lifecycle.spec.ts` (`Tải bài lên` có 2 nút: thanh công cụ + trạng thái rỗng → `.first()`), `exam.spec.ts` (tiêu đề ngăn kéo "Chọn câu đã duyệt từ ngân hàng").

## Lệch / nợ
- Bộ chọn lớp hiện "buổi tiếp theo" chưa làm (Nợ theo spec).
- Hai `ui-allow` mới ở `not-found.tsx`, `AppShell.tsx` (`NoCourse`) — tổng 10/10.
