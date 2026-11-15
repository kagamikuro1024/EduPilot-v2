# QC test case — US-AS-05 (đóng, đánh vắng), US-AS-06 (danh sách cờ, Hôm nay), US-AS-07 (đối chiếu người thật)
Nguồn: US.md + SRS §3.1, 3.2, 4.2, 4.3, 6 #6–#10, #15–#16. Hộp đen; quy ước: `tc-00-chung.md`. Lớp QC-AS (30 SV ghi danh để đủ số liệu; ≥ 3 SV chưa điểm danh). `S`=buổi quá khứ 1 giờ (`starts_at`), `W`=cửa sổ. `late_after_min=10`.

## US-AS-05 Đóng cửa sổ, đánh vắng
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS05-01 | AC1, AC2 | `W` mở; 5 SV điểm danh (2 SV trong 10 phút đầu sau `starts_at`, 3 SV sau 10 phút — dùng buổi có `starts_at` giả lập quá khứ, xem Q-S108-1) | Staff `POST …/checkin-windows/{W}/close` | `200 {window, written:{present, late}, unchecked, skipped_existing}`; `closed_at`, `closed_by`; `audit_log` `checkin_window.close` (số đếm, **không** tên) | API+DB |
| TC-AS05-02 | AC2 | – | Đọc `attendance_records` của buổi | Mỗi SV có lượt: `PRESENT`, hoặc `LATE` nếu **lượt sớm nhất** của SV sau `starts_at + late_after_min`; `marked_by` = người đóng; `version=1` | DB |
| TC-AS05-03 | AC2 | Một SV đã có dòng GV sửa tay (`ABSENT`, `version=3`) trước khi đóng; SV đó cũng có lượt | Đóng | Dòng của SV đó **không đổi** (`ABSENT`, `version=3`); tính vào `skipped_existing` | API+DB |
| TC-AS05-04 | AC3 | SV ACTIVE không có lượt nào | Đóng; đọc DB | **Không** hàng `ABSENT` nào được tạo; `unchecked` = số SV ACTIVE chưa có dòng điểm danh của buổi | API+DB |
| TC-AS05-05 | AC1 | Cửa sổ tự đóng (US-AS-02 AC7) | Đọc DB | `closed_by IS NULL`, `marked_by` = người mở, cùng quy tắc | DB |
| TC-AS05-06 | AC4 | Cửa sổ đã đóng | `close` lại hai lần (một lần cùng, một lần khác `Idempotency-Key`) | `200` cùng `{window, written, unchecked, skipped_existing}` đã lưu; `attendance_records`, `audit_log`, outbox **không** thêm hàng | API+DB |
| TC-AS05-07 | AC5 | Buổi có cửa sổ, không cửa sổ nào mở; 3 SV chưa có dòng | `POST …/sessions/{S}/absent-unchecked` (`Idempotency-Key`) | `200 {marked:3}`; ba dòng `ABSENT`, `marked_by`=người gọi; `audit_log` `attendance`/`absent_unchecked` (số đếm + `[{student_id,to:"ABSENT"}]`, không tên) | API+DB |
| TC-AS05-08 | AC5 | – | Lặp lại cùng khoá; rồi gọi lại khác khoá | `200` cùng kết quả / `{marked:0}`; không dòng đôi (`ON CONFLICT DO NOTHING`) | API+DB |
| TC-AS05-09 | AC5 | Sau đó | `PUT …/attendance` sửa một dòng `ABSENT` → `PRESENT` bằng `version` | `200`; dòng sửa được bằng lưới như thường (`version`+1, `audit_log`) | API |
| TC-AS05-10 | AC6 | Buổi có cửa sổ, còn 3 SV chưa có dòng | `PUT …/sessions/{S}/attendance {complete:true, records:[…chỉ SV đã có dòng…]}` | `422 CHECKIN_UNRESOLVED {count:3}`; không ghi gì | API+DB |
| TC-AS05-11 | AC6 | Cùng buổi | Thân phủ hết 3 SV (đánh vắng tay) rồi `complete:true` | `200` | API |
| TC-AS05-12 | AC6 | Buổi **không** có cửa sổ (buổi khác) | `PUT … {complete:true}` như trước sprint | Hành vi cũ: điền `PRESENT` cho người thiếu; kết quả **cùng byte** với bản chụp trước sprint | API |
| TC-AS05-13 | AC7 | – | `absent-unchecked` khi còn cửa sổ mở; khi buổi chưa từng có cửa sổ | `409 WINDOW_OPEN`; `409 NO_WINDOW` | API |
| TC-AS05-14 | AC7 | – | `close` cửa sổ của lớp khác; lớp lưu trữ | `404`; `409 COURSE_ARCHIVED` | API |
| TC-AS05-15 | AC8 | Cửa sổ mở, 10 SV đã điểm danh | Chạy song song `close` và `PUT …/attendance` (GV sửa tay 1 SV) × 20 vòng (buổi mới mỗi vòng) | Không mất ghi, không dòng trùng, không deadlock; dòng GV vừa lưu thắng | API+DB |
| TC-AS05-16 | AC9 | – | Mọi route #6, #8 bằng STUDENT, ADMIN, `sv.moi`; TA và TEACHER | `403`; `sv.moi` `reason=course`; TA / TEACHER ✓ | API |
| TC-AS05-17 | – | Giao diện | Đóng bằng lưới `/attendance` | «Đã ghi {a} có mặt · {b} muộn · {u} chưa điểm danh» tại chỗ (không toast) + `Đánh vắng {u} SV chưa điểm danh` → hộp xác nhận chuẩn nêu số người | UI |

## US-AS-06 Danh sách lượt, bỏ cờ, Hôm nay
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS06-01 | AC1 | Buổi có ≥ 120 lượt (dùng seed hoặc 120 SV giả qua `INSERT` fixture QC) | `GET …/sessions/{S}/checkins?limit=50`, đi hết `cursor` | Thứ tự `(created_at,id)`; không trùng / thiếu; mỗi phần tử `{id, window_id, round, student:{id,full_name,student_code}, created_at, channel, assurance, flags:[{code,label}], cleared, geo_distance_m?, in_campus?}` | API |
| TC-AS06-02 | AC1 | – | `?flagged=true`; `?round=2`; `limit=100`; `limit=101`; `cursor` hỏng | `flagged` chỉ lượt còn cờ chưa bỏ; `round` lọc đúng; `101` → `422`; `cursor` hỏng → `422` | API |
| TC-AS06-03 | AC1 | – | `EXPLAIN` cho truy vấn danh sách (hoặc đọc kế hoạch qua `pg_stat_statements`) | Dùng chỉ mục `(course_id, session_id, created_at)`; không `OFFSET` | DB |
| TC-AS06-04 | AC2 | Lượt có cờ | `POST …/checkins/{cid}/clear-flags` | `200`; `flags_cleared_by/at` đặt; `audit_log` `checkin.clear_flags` (mã cờ, `student_id`, không tên); `flags` **không bị xoá** khỏi hàng; `cleared=true` | API+DB |
| TC-AS06-05 | AC2 | – | Gọi lại | `200`, không đổi, không audit mới | API+DB |
| TC-AS06-06 | AC3 | Cửa sổ đóng ≤ 7 ngày, k=3 lượt còn cờ chưa bỏ | TA / TEACHER `GET /me/today` | Có mục `CHECKIN_REVIEW`: «Buổi {no}: 3 lượt điểm danh cần xem · {mã lớp}», `Href=/attendance?course=…&session=…&filter=review`; thứ tự bậc 13 (sau các bậc cao hơn) | API |
| TC-AS06-07 | AC3 | Bỏ cờ hết (k=0) | `GET /me/today` | Mục biến mất | API |
| TC-AS06-08 | AC3 | Cửa sổ đóng > 7 ngày (cập nhật `closed_at` fixture) | `GET /me/today` | Mục biến mất | API+DB |
| TC-AS06-09 | AC3, AC5 | – | STUDENT `GET /me/today` | Không có `CHECKIN_REVIEW`; không cờ nào | API |
| TC-AS06-10 | AC4 | – | `cid` của lớp khác; lớp lưu trữ `GET` và `clear-flags` | `404`; đọc được; `409 COURSE_ARCHIVED` | API |
| TC-AS06-11 | AC5 | – | STUDENT, ADMIN, `sv.moi` gọi `GET …/checkins` / `clear-flags` | `403`; `sv.moi` `reason=course` | API |
| TC-AS06-12 | AC6 | Cửa sổ đóng > 30 ngày (fixture `closed_at` cũ) + chạy purge (US-AS-13) | `GET …/checkins` | Lượt quá 30 ngày không còn | API+DB |

## US-AS-07 Đối chiếu bằng người thật
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS07-01 | AC1 | `headcount=5`, 12 lượt khác nhau | `GET …/checkin-windows`, giao diện `/attendance` | `over_headcount=true`; giao diện nhãn «Số điểm danh nhiều hơn số người đếm»; không chặn gì (vẫn lưu được) | API+UI |
| TC-AS07-02 | AC2 | Vòng 1: 5 SV; vòng 1 đóng; mở vòng 2; chỉ 3 trong 5 điểm danh lại; đóng vòng 2 | Đọc `flags` của 5 lượt vòng 1 | Hai SV vắng vòng 2 có `RECHECK_MISS` ở lượt **vòng 1**; `attendance_records` **không đổi** | API+DB |
| TC-AS07-03 | AC2 | – | Đóng vòng 2 lần hai (idempotent) | `RECHECK_MISS` không nhân đôi | DB |
| TC-AS07-04 | AC2 | – | Đóng vòng 1 (đầu tiên) | **Không** `RECHECK_MISS` (chỉ khi đóng vòng ≥ 2) | DB |
| TC-AS07-05 | AC3 | Buổi 12 lượt, 3 lượt còn cờ chưa bỏ | `POST …/checkin-windows/{W}/spot-check {n:5}` | `200 {students:[…5…]}`: 3 người có cờ trước (`reason=FLAGGED`), 2 ngẫu nhiên (`reason=RANDOM`); không trùng; mỗi phần tử `{student_id, full_name, student_code, reason}`; DB không đổi, `audit_log` không dòng mới | API+DB |
| TC-AS07-06 | AC3 | – | `n` mặc định (không gửi) / `n=10` / `n` > số lượt | Mặc định 5; 10 ok; `n` lớn hơn số lượt → trả hết | API |
| TC-AS07-07 | AC3 | – | Chạy 20 lần cùng buổi | Phần ngẫu nhiên thay đổi (phân bố, không cố định; `crypto/rand`) — không cần chứng minh mật mã, kiểm ≥ 3 tập khác nhau | API |
| TC-AS07-08 | AC4 | – | `n=0`, `n=11`, `n="a"`; buổi chưa có lượt; `wid` lớp khác | `422`; `200 {students:[]}`; `404` | API |
| TC-AS07-09 | AC5 | – | STUDENT / ADMIN / `sv.moi`; TA / TEACHER | `403` / `403` `reason=course` / ✓ | API |
| TC-AS07-10 | AC1–AC3 | Giao diện `/attendance` | Menu `⋯` | `Mở vòng 2`, `Nhập số đầu người` (ô số tự lưu), `Gọi ngẫu nhiên 5 người` → `Drawer` 5 tên + MSSV + lý do «Có cờ» / «Ngẫu nhiên» | UI |
