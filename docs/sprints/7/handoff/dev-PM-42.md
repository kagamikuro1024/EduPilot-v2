# DEV handoff — #42 (luật chữ trên UI: không câu mô tả dưới tiêu đề)
Nhánh: `sprint/7-p4-p5`. Commit ghi `#42`. Luật: `~/.claude/CLAUDE.md` "Luật cứng — chữ trên UI" (xoá hẳn, không dời vào tooltip).

## Câu đã xoá (6)
| # | Màn | Câu | Cách |
|---|---|---|---|
| 1 | `/students` | "Chuyên cần và nhãn rủi ro chỉ giảng viên/TA thấy" (`PrivateMark` ở `meta` tiêu đề) | bỏ `meta` |
| 2 | `/class/settings` | "Lớp {mã}. Chia sẻ mã hoặc liên kết cho sinh viên; bạn quyết định ai được vào." | còn `Lớp {mã}` (dữ liệu) |
| 3 | `/settings` (khối Mật khẩu) | "Đổi mật khẩu sẽ đăng xuất mọi thiết bị khác; thiết bị này vẫn đăng nhập." | bỏ `description` |
| 4 | `/students/[id]` | nhãn "Chỉ giảng viên/TA thấy" lặp trong khung nhận định rủi ro (đã có một nhãn ở khối Ghi chú) | bỏ nhãn thứ hai |
| 5 | `/students/[id]` (Ghi chú) | placeholder "…— sinh viên không thấy nội dung này." (lặp nhãn riêng tư ngay cạnh) | placeholder còn "Điều bạn quan sát được và việc đã làm" |
| 6 | `/students/[id]` (bản nháp AI) | "Nhận xét tổng hợp — sửa rồi mới lưu" | còn "Nhận xét tổng hợp" |

## Rà bốn màn mới
- `/inbox`, `/attendance`, `/me`: không có câu mô tả dưới tiêu đề; chỉ có dữ liệu ở dòng phụ (`mã lớp`, `buổi · ngày · phòng`) và thông báo trạng thái / lỗi (không phải mô tả). Không đổi.
- `/students/[id]`: mục 4–6 ở trên. Giữ MỘT nhãn riêng tư ở khối Ghi chú (hệ quả của việc gõ vào đó).
- Tooltip còn lại trên màn mới: `title` của cột "Hoạt động học" ở `/students` ("Phút: 7 ngày gần nhất. Câu hỏi và bài đăng: tuần trước (T2–CN).") — đặt trên chính nhãn, nói đơn vị / khoảng thời gian mà thiếu nó số liệu bị đọc sai. Giữ.

## Ảnh mốc, SRS, test
- Ảnh mốc `visual.spec.ts` chỉ có home, chat, threads, inbox, gradebook, settings-llm, dev-ui: không màn nào ở trên → **không đổi ảnh nào**.
- Không test nào khẳng định các câu đã xoá (chỉ còn tên test `ai summary edit then save: … sửa rồi mới lưu` — nói về hành vi, không phải chữ trên màn). `students.spec`, `account.spec`, `class-join.spec`, `panels.spec`, `a11y.spec`, `tokens.spec`, `shell.spec`, `me.spec`, `attendance.spec`, `escalation.spec` chạy lại (xem cuối).
- BA vá SRS / US (nếu có chữ tương ứng): FEAT-crm-360 SRS 7.4 / US AC5 nhắc "sửa rồi mới lưu" như HÀNH VI của bản nháp (vẫn đúng: bản nháp sửa được rồi bấm Lưu), chỉ dòng chữ trên màn bị bỏ; nhãn `PrivateMark` ở SRS 7.4 / US `/students/[id]` còn một lần ở Ghi chú.

## Ngoài phạm vi #42 — câu mô tả dưới tiêu đề vẫn còn (chưa đổi, PM quyết)
`/settings/llm` (tiêu đề trang + bốn khối), `/settings/integrations` (tiêu đề + dòng mô tả từng tích hợp), `/settings` bảo mật (tiêu đề trang "Đổi mật khẩu và xem nơi tài khoản…"), `/class/members` ("Lớp … Duyệt yêu cầu vào lớp và quản lý người học."), `Switch` "Cần giảng viên duyệt" ("Sinh viên vào bằng mã sẽ chờ bạn duyệt."), `QuestionReview` / `ResultDrawer` ("chỉ giảng viên/TA thấy"), `EmptyState` của `MembersView`. Đều có trong SRS / ảnh mốc cũ (`settings-llm-*.png`).

## Kiểm
`pnpm lint`, `tsc`, `ui-antipatterns.sh` sạch. Playwright (desktop + mobile, `--retries=0`) `students`, `account`, `class-join`, `panels`, `a11y`, `tokens`, `shell`, `me`, `attendance`, `escalation`: 291 passed, 193 skipped (ca chỉ chạy một dự án), 0 failed.
