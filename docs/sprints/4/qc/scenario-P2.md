# Kịch bản tay P2 (TC-P212-24 / -25) — luồng F1 + F2 từ DB trống

QC chạy trên stack riêng (DB trống, không seed): Postgres + Redis + Mailpit + 2 gateway + worker thật + `next start`, Chrome for Testing. Ảnh ở `shots/p212/` (1440 px cho Admin / Giảng viên, 375 × 812 cho Sinh viên).

| Bước | Việc | Kỳ vọng (AC) | Ảnh |
| --- | --- | --- | --- |
| 0 | `printf 'MẬT-KHẨU\n' \| gateway admin create --email admin@edupilot.local --name "Quản trị viên EduPilot"` | `Đã tạo quản trị viên.` (US-P2-06 AC9) | — |
| 1 | Admin đăng nhập, `/admin/users` | một nút chính `Mời giảng viên` (US-P2-06 AC10) | `01-admin-users` |
| 2 | Mời `teacher@…` "TS. Lê Thu Hà" | "Đã gửi link mời, hạn 72 giờ." tại chỗ | `02-admin-invited` |
| 3 | Mailpit → `/invite/<token>` → đặt mật khẩu → vào `/` | token bị thay `·` trong URL; vào "Hôm nay" | `03-gv-invite`, `04-gv-home-empty` |
| 4 | Admin `/admin/courses` → `Mở lớp` 761987 + chọn giảng viên | "Đã mở lớp. Đã gửi thông báo phân công cho …" (US-P2-08 AC10) | `05-admin-open-course`, `06-admin-course-opened` |
| 5 | GV đăng nhập: chuông + khung thông báo | "Thông báo, 1 chưa đọc" ≤ 60 s; "Bạn được phân công lớp An ninh mạng – 761987" | `07-gv-bell` |
| 6 | Bấm thông báo → `/class/settings` | mã tham gia hiện đúng `courses.join_code` | `08-gv-class-settings` |
| 7 | SV D (375 px) `/register` → thư → `/verify-email` | màn "Kiểm tra email"; xác minh; token rời URL | `09-d-register-375`, `10-d-check-mail-375` |
| 8 | D đăng nhập `/` | ô nhập mã ngay trên trang (US-P2-11 AC11) | `11-d-today-375` |
| 9 | D mở `/join/<mã>` → xem trước → `Tham gia lớp` | "An ninh mạng · 761987 · TS. Lê Thu Hà · HK1 2026–2027"; "Bạn đã vào lớp." | `12-d-join-preview-375`, `13-d-joined-375` |
| 10 | GV `/class/members` | hàng của D (MSSV, "Bằng mã") | `14-gv-members` |

Bạn tự kiểm P2 (8 mục, TC-P212-25): (1) mạo danh MSSV → TC-P210-14 / TC-P209-30; (2) quên mật khẩu máy A → máy B đăng xuất → TC-P204-14; (3) storage không token (đo lại: `localStorage` chỉ có `ep_demo_state` mock, `sessionStorage` `ep_auto_heal`; không `eyJ`); (4) import 2 dòng lỗi → TC-P210-06; (5) bấm đúp tạo → 1 bản ghi → TC-P208-31; (6) tạo lại mã, mã cũ bị từ chối (đo lại); (7) `/` có một hành động chính ≈ 3 s (đo lại); (8) đăng nhập 7 tài khoản mẫu (TC-P212-05).
