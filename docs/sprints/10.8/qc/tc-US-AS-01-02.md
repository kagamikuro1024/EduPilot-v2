# QC test case — US-AS-01 (chính sách lớp), US-AS-02 (mở / đóng cửa sổ)
Nguồn: US.md + SRS §3.1, 3.4, 4.1, 4.2, 5, 6 #1–6. Hộp đen; quy ước: `tc-00-chung.md`. `C`=lớp QC-AS, `S`=một buổi của C, `W`=cửa sổ. Tiền tố route `…` = `/courses/{C}`. Loại: API, DB, UI, E2E, SQL.

## US-AS-01 Chính sách điểm danh theo lớp
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS01-01 | AC1 | Lớp mới, chưa đụng `settings.attendance`; lưu bản chụp phản hồi `PUT …/sessions/{S}/attendance` | `GET /courses/{C}` ; `POST …/sessions/{S}/checkin-windows`; `POST …/checkins`; rồi `PUT …/attendance` (cùng thân như bản chụp) | `attendance` thiếu hoặc `self_checkin=false`; hai route đầu `409 SELF_CHECKIN_DISABLED`; điểm danh tay **cùng byte** (so với bản chụp, bỏ trường thời gian) | API |
| TC-AS01-02 | AC2 | TEACHER | `PUT …/settings {attendance:{self_checkin:true, rotate_seconds:5, late_after_min:10, require:"WEB"}, version}` | `200 CourseDetail`; `version`+1; `audit_log` (`course_settings`/`attendance_policy`, trước/sau); outbox `course.settings_changed` (đọc bảng outbox) | API+DB |
| TC-AS01-03 | AC2 | Bật `geofence {lat,lng,radius_m:200}` + `campus_cidrs` | `PUT …/settings`; đọc `audit_log` | `audit_log` chỉ ghi «geofence: bật», bán kính; `grep` toạ độ (lat/lng đã gửi) trong `audit_log` = 0 | API+DB |
| TC-AS01-04 | AC3 | – | Lần lượt `rotate_seconds` 4 / 61, `late_after_min` −1 / 121, `require:"XX"`, `geofence.radius_m` 19 / 2001, lat 91, lng −181, 11 CIDR, CIDR `10.0.0.0/0`, `1.2.3.4/33`, chuỗi rác | Mỗi ca `422 VALIDATION_FAILED` kèm tên trường; `settings` không đổi, `version` không đổi, 0 dòng `audit_log` mới | API+DB |
| TC-AS01-05 | AC3 | – | `PUT` với `attendance:{}` (chỉ `self_checkin:true`) | Mặc định: `rotate_seconds=10`, `late_after_min=15`, `require=WEB` (đọc `GET /courses/{C}` Staff) | API |
| TC-AS01-06 | AC3 | – | Biên hợp lệ: `rotate_seconds` 5 và 60; `late_after_min` 0 và 120; `radius_m` 20 và 2000; đúng 10 CIDR | Tất cả `200` | API |
| TC-AS01-07 | AC4 | – | `PUT` với `version` cũ | `409 VERSION_CONFLICT`, không ghi | API |
| TC-AS01-08 | AC4 | – | Hai `PUT` song song cùng `version` | Đúng một `200`, một `409`; `version` chỉ +1 | API |
| TC-AS01-09 | AC4 | Lớp lưu trữ | `PUT …/settings {attendance…}` | `409 COURSE_ARCHIVED` | API |
| TC-AS01-10 | AC5 | Cửa sổ `W` đang mở với `rotate_seconds=5` | TEACHER đổi `rotate_seconds` thành 30 rồi tắt `self_checkin`; `GET …/display` hai lần cách 12 s; SV `POST …/checkins` | `display` vẫn xoay mỗi 5 s (đã ghim); SV vẫn điểm danh được tới khi đóng; mở cửa sổ mới → `409 SELF_CHECKIN_DISABLED` | API |
| TC-AS01-11 | AC6 | – | `PUT …/settings {attendance…}` bằng TA, STUDENT, ADMIN, `sv.moi` (ngoài lớp) | Đều `403`; `sv.moi` có `details.reason="course"` | API |
| TC-AS01-12 | AC6 | Lớp bật `self_checkin` + geofence + CIDR | `GET /courses/{C}` bằng TEACHER / TA / STUDENT | Staff: `self_checkin, require, geo_check, rotate_seconds, late_after_min, geofence, campus_cidrs`. STUDENT: **chỉ** `self_checkin, require, geo_check` (bool); `geofence`, `campus_cidrs`, `rotate_seconds`, `late_after_min` vắng | API |
| TC-AS01-13 | AC7 | TEACHER mở `/courses/{C}` (cài đặt) 1440 + 375 | Bật công tắc «Vị trí» | Đúng **một** dòng «Lớp này kiểm tra vị trí khi điểm danh.» cạnh công tắc; khối «Điểm danh bằng mã»: công tắc, `Chu kỳ mã`, `Tính muộn sau`, `Yêu cầu`; mục «Vị trí», «Mạng trường» mở dần; `422` hiện tại ô; một cột ở 375; không câu giải thích thêm | UI |
| TC-AS01-14 | AC7 | Lớp bật geofence; SV mở `/checkin` | Quan sát trước khi trình duyệt hỏi quyền (Playwright `permissions: []`, bắt `geolocation`) | Dòng «Lớp này kiểm tra vị trí khi điểm danh.» hiện **trước** lần gọi `navigator.geolocation` đầu tiên | E2E |
| TC-AS01-15 | – | Lớp bật `self_checkin` | `PUT …/settings` thân KHÔNG có `attendance` nhưng đổi trường khác (`name`) | `attendance` **không bị xoá** (không có trong thân ⇒ không đổi); ngược lại thân có `attendance:{self_checkin:false}` thay **toàn bộ** chính sách (trường vắng = mặc định) | API |

## US-AS-02 Mở / đóng / liệt kê cửa sổ
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS02-01 | AC1 | `self_checkin` bật, `rotate_seconds=5` | TEACHER `POST …/sessions/{S}/checkin-windows` (không `minutes`) kèm `Idempotency-Key` K | `201 {id, round:1, rotate_seconds:5, closes_at}`; `closes_at ≈ now+10 phút` (±5 s); `audit_log` `checkin_window.open` | API+DB |
| TC-AS02-02 | AC1 | – | Lặp lại cùng K | `201`/`200` cùng `id`; `SELECT count(*) FROM checkin_windows WHERE session_id=S` = 1 | API+DB |
| TC-AS02-03 | AC1 | Vòng 1 đã đóng | Mở vòng 2 không `minutes`; mở vòng 3 `minutes=1` | Vòng 2 `closes_at ≈ now+3 phút`; vòng 3 ≈ 1 phút; `round` = 2, 3 | API |
| TC-AS02-04 | AC1, AC8 | – | `minutes` = 0, 61, −1, "x" | `422`; không cửa sổ mới | API |
| TC-AS02-05 | AC1 | – | `minutes` = 1 và 60 | `201` | API |
| TC-AS02-06 | AC2 | Cửa sổ đang mở | Mở lại (khác `Idempotency-Key`) | `409 WINDOW_ALREADY_OPEN {window_id}` đúng id đang mở | API |
| TC-AS02-07 | AC2 | Buổi chưa có cửa sổ | Hai `POST` song song khác khoá (20 vòng lặp / buổi mới) | Mỗi vòng đúng một `201` + một `409`; `SELECT count(*) … closed_at IS NULL` ≤ 1 | API+DB |
| TC-AS02-08 | AC3 | Buổi có 3 vòng đã đóng | Mở vòng 4 | `409 ROUND_LIMIT`; hàng `checkin_windows` của buổi = 3 | API+DB |
| TC-AS02-09 | AC3 | Vòng 1 đã đóng; SV có chuông / `Hôm nay` trống | Mở vòng 2; đọc outbox + `notifications` + `GET /me/today` của SV | Không có sự kiện tới SV nào; không mục `Hôm nay` mới cho SV | API+DB |
| TC-AS02-10 | AC4 | Cửa sổ mở | Staff `GET …/display` hai lần cách 1 s (cùng bước) | `200 {qr_url, short_code, step_expires_at, server_time, rotate_seconds, round, checked_in, headcount, closes_at}`; `qr_url` = `{APP_PUBLIC_URL}/checkin?c={C}#<token>`; `short_code` 6 ký tự Crockford (không `I L O U`); hai lần cùng bước → **cùng** `qr_url` và `short_code`; `Cache-Control: no-store` | API |
| TC-AS02-11 | AC4 | – | Chờ `step_expires_at`+1 s, gọi lại | `qr_url` **và** `short_code` đổi cùng lúc (cùng bước) | API |
| TC-AS02-12 | AC4 | – | Đọc `docker logs gateway`/access log của Caddy sau vài lần `display` + quét | Không dòng nào chứa phần sau `#` của `qr_url` (token không vào access log); response `display` không có `Referer` policy dẫn token (kiểm header `Referrer-Policy` của trang `/checkin` = `no-referrer`/`same-origin`) | API |
| TC-AS02-13 | AC5 | Buổi có 3 cửa sổ | `GET …/checkin-windows` | ≤ 3 phần tử, mỗi cái `{id, round, opened_at, closes_at, closed_at, headcount, checked_in, over_headcount}`; **không** có `token`/`short_code`/mã nào | API |
| TC-AS02-14 | AC5 | – | Đọc schema `openapi.yaml` của route #3 + DB | `maxItems: 3`; `CHECK (round BETWEEN 1 AND 3)` + `UNIQUE (course_id,session_id,round)` (đọc `\d checkin_windows`) | DIFF+DB |
| TC-AS02-15 | AC6 | Cửa sổ có 12 lượt | `PATCH …/checkin-windows/{W} {headcount:10}` rồi 12 lượt; kế đó `{headcount:12}` | `headcount=10`: `over_headcount=true` khi số SV khác nhau > 10 + max(2, ⌈0,5⌉=1 → 2)=12 ⇒ với 12 lượt **false**, 13 lượt **true**; `headcount=2000` OK; `audit_log` `checkin_window.headcount` | API+DB |
| TC-AS02-16 | AC6, AC8 | – | `headcount` = −1, 2001, "a" | `422` | API |
| TC-AS02-17 | AC6 | Cửa sổ đã đóng | `PATCH headcount` | `200` (cho phép khi đã đóng) | API |
| TC-AS02-18 | AC7 | `minutes=1`, không ai đóng | Chờ 125 s; đọc DB + `audit_log` | `closed_at` đặt, `closed_by IS NULL`, `audit_log` `checkin_window.auto_close`; `attendance_records` ghi theo thủ tục US-AS-05 (có mặt/muộn cho SV đã điểm danh); không `ABSENT` | API+DB |
| TC-AS02-19 | AC7 | Cửa sổ đã tự đóng | Chờ thêm 2 nhịp quét (120 s) | Không đổi gì, không audit mới | DB |
| TC-AS02-20 | AC8 | – | `sid` của lớp khác; `wid` không tồn tại; UUID hỏng | `404` (UUID hỏng `422`) | API |
| TC-AS02-21 | AC8 | Cửa sổ đã đóng / quá `closes_at` | `GET …/display` | `409 WINDOW_CLOSED` | API |
| TC-AS02-22 | AC8 | – | 31 lần `GET …/display` trong 60 s (một người) | Từ lần thứ 31 `429` + `Retry-After` | API |
| TC-AS02-23 | AC8 | Lớp lưu trữ | `POST …/checkin-windows` | `409 COURSE_ARCHIVED` | API |
| TC-AS02-24 | AC9 | – | Mọi route #2–#7 bằng STUDENT, ADMIN, `sv.moi`; TA và TEACHER | STUDENT/ADMIN `403`; `sv.moi` `403` `reason=course`; TA/TEACHER thành công; `audit_log` ghi đúng `actor` | API+DB |
| TC-AS02-25 | – | Lớp bật; hai cửa sổ ở hai buổi khác nhau của lớp | Mở cửa sổ ở buổi S1 khi buổi S2 đang mở | Cho phép (mỗi buổi một cửa sổ mở) — một cửa sổ mở **mỗi buổi**, không mỗi lớp | API |
