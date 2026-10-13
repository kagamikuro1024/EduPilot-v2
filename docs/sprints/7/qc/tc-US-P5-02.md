# QC test case — US-P5-02 (điểm danh ≤ 60 s, phát biểu, 409, mất mạng, AI hỏi đáp)
Nguồn: `US.md` US-P5-02 AC1…AC16 (AC13 có điều kiện cắt, AC13b) + `SRS.md` 2, 3.1, 3.3, 4.3–4.5, 6.2, 7.2, 9.4; `DESIGN.md` §14.8; `UX.md` mục 1, 2, 5; `P5.md` L1 "Bạn tự kiểm". Hộp đen, không đọc code dev.
Biến như `tc-US-P5-01.md`. Điểm danh: `GET/PUT $Q/sessions/$S10/attendance`. Giao diện bằng `playwright-cli` (375 × 812 và 1280 × 800). Bấm giờ: `performance.now()` trong `playwright-cli eval` từ lúc `goto /attendance` đến khi hiện "Đã lưu điểm danh".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P5-02-01 | AC1 (đọc lưới) | lớp 1 có 30 sinh viên `ACTIVE` | `j "$Q/sessions/$S10/attendance" -H "$TCH"` | `{session:{id,session_no,starts_at,ends_at,room,topic,attendance_done_at}, students:[{id,full_name,student_code,record:{…}\|null,speak_count}]}` cho mọi sinh viên `ACTIVE`; sắp theo tên không dấu, ổn định (gọi 3 lần cùng thứ tự); `record:null` khi chưa có |
| TC-P5-02-02 | AC1 (không PENDING/REMOVED) | 1 sinh viên `PENDING`, 1 `REMOVED` | gọi lưới | không có trong `students` |
| TC-P5-02-03 | AC1 (vai, lớp khác) | – | `$SVA`, `$ADM`; buổi lớp 2 qua URL lớp 1 | `403`; `404` |
| TC-P5-02-04 | AC2 (ghi cả lưới 1 giao dịch) | lưới 30 hàng | `PUT … {"records":[…30…],"complete":false}`; `select count(*), max(version) from attendance_records where session_id=…`; `select * from audit_log where entity='attendance'`; `outbox` `crm.attendance_saved` | `200 {records:[{student_id,status,version}],done_at}`; 1 dòng `audit_log` (`action` = `save`, `entity_id=<session_id>`, `before`/`after` là số đếm + `{student_id,from,to}`, **không** tên); outbox `crm.attendance_saved`; hàng giống hệt không đổi `version` |
| TC-P5-02-05 | AC2 (gửi lại cùng nội dung) | sau TC-04 | `PUT` lại y hệt | `200`; không thêm `audit_log` mới; `version` giữ |
| TC-P5-02-06 | AC2 (hàng giống hệt bị bỏ qua) | – | `PUT` với 29 hàng giữ nguyên + 1 hàng đổi | chỉ hàng đổi `version`+1; `audit_log` chỉ nêu 1 `{student_id,from,to}` |
| TC-P5-02-07 | AC2 (hiệu năng) | 30 hàng | đo `PUT` 20 lần (`curl -w '%{time_total}'`) | p95 ≤ 500 ms |
| TC-P5-02-08 | AC3 (409 xung đột) | `T1` (`$TCH`) và `T2` (`$TA_`) cùng đọc lưới (hàng X version 1); T1 đổi X (version 2) | T2 `PUT` X với `version:1` và trạng thái khác DB | `409 VERSION_CONFLICT` với `details.conflicts:[{student_id,mine:{status},current:{status,version,marked_by:{full_name},marked_at}}]`; **không** hàng nào được ghi (kể cả hàng khác của T2) |
| TC-P5-02-09 | AC3 (hàng không đổi không xung đột) | – | T2 gửi hàng X trùng DB hiện tại | không `409` |
| TC-P5-02-10 | AC3 (UI) | `playwright-cli` hai ngữ cảnh: teacher và TA cùng `/attendance?session=$S10` | cả hai đổi cùng sinh viên; teacher lưu trước; TA lưu | TA thấy `InlineNotice` "Thầy/Cô … vừa đổi N dòng. Giữ của tôi / Lấy của …" với chọn theo từng dòng; các thay đổi còn lại của TA giữ; chọn xong gửi lại với `version` mới thành công |
| TC-P5-02-11 | AC4 (dữ liệu sai) | – | `student_id` ngoài lớp / không `ACTIVE`; trùng `student_id` trong thân; `status:"X"`; `note` 201 ký tự; 1.001 hàng; buổi lớp khác; lớp `ARCHIVED` | `422 STUDENT_NOT_IN_COURSE` (kèm `details.student_ids`); `422`; `422`; `422`; `422`; `404`; `409 COURSE_ARCHIVED` |
| TC-P5-02-12 | AC4 (DB lỗi) | dừng postgres giữa `PUT` (script: `docker compose stop postgres` ngay khi gửi) | `PUT` | `503`; không ghi nửa chừng (`count` không đổi sau khi bật lại) |
| TC-P5-02-13 | AC5 (hoàn tất) | buổi chưa hoàn tất | `PUT … "complete":true`; `select attendance_done_at, attendance_done_by`; `audit_log.action` | `done_at`/`done_by` đặt lần đầu; `action='complete'`; máy chủ điền `PRESENT` cho sinh viên không có hàng khi `complete:true` |
| TC-P5-02-14 | AC5 (sửa sau) | buổi đã `done_at` | `PUT` đổi 1 hàng (`complete:false` hoặc `true`) | `200`; `action='edit_after_complete'` có `before`/`after` đủ dựng lại hàng nào đổi từ gì sang gì; không giới hạn thời gian; `done_at` không bị đặt lại |
| TC-P5-02-15 | AC5 (autosave không hoàn tất) | buổi chưa hoàn tất | `PUT` `complete:false` | `done_at` vẫn `NULL` |
| TC-P5-02-16 | AC6 (375 px) | `resize 375 812`, `/attendance?session=$S10` | xem danh sách; bấm một hàng; chọn `Vắng` | danh sách hàng ngăn đường kẻ 1 px (không card); mỗi hàng: tên + MSSV rút gọn + trạng thái bằng chữ ("Có mặt" nhạt; "Vắng" đỏ); bấm hàng mở điều khiển 4 lựa chọn ngay dưới hàng (`Có mặt`·`Muộn`·`Vắng phép`·`Vắng`, mỗi nút cao ≥ 44 px); đổi một người ≤ 2 chạm; người không chạm giữ "Có mặt"; không `Dialog`; không tràn ngang (`scrollWidth ≤ innerWidth`); zoom 200 % vẫn thao tác được |
| TC-P5-02-17 | AC6 (a11y) | – | `$PW a11y.spec.ts -g attendance` | 0 vi phạm nghiêm trọng |
| TC-P5-02-18 | AC7 (≥ 1100 px) | `resize 1280 800` | xem lưới; cuộn; bấm `1`–`4`, `↑`/`↓`, `Shift+1…4` | `AttendanceGrid` 4 cột radio + cột `Phát biểu`; cột tên và header dính khi cuộn; vòng focus nhìn thấy; `1`–`4` đặt trạng thái hàng focus rồi xuống hàng kế; `Shift+1…4` không chuyển hàng |
| TC-P5-02-19 | AC7 (30 SV bằng phím ≤ 60 s) | lớp 30 sinh viên | `playwright-cli`: gõ chuỗi phím cho 30 hàng (3 vắng, 1 muộn), `Lưu điểm danh`; đo thời gian | tổng ≤ 60 s; kết quả DB đúng |
| TC-P5-02-20 | AC8 (lạc quan ≤ 100 ms, hoàn tác) | `/attendance` | đổi một trạng thái; đo thời gian tới khi UI đổi; xem dòng tĩnh | UI đổi ≤ 100 ms; dòng "Đã đánh Vắng · Hoàn tác" 5 giây; không toast, không hộp thoại; Hoàn tác trả trạng thái trước |
| TC-P5-02-21 | AC8 (gom 2 giây, chỉ báo lưu) | – | đổi 3 hàng liên tiếp trong 1 s; theo dõi `network`; xem đầu trang | đúng 1 `PUT` sau 2 s yên lặng (`complete:false`); "Đang lưu…" → "Đã lưu HH:MM" |
| TC-P5-02-22 | AC8 (5xx) | `page.route` trả 500 cho `PUT` | đổi trạng thái | trạng thái hiển thị hoàn lại + nút `Thử lại`; chữ đã chọn không mất; `Thử lại` thành công khi bỏ chặn |
| TC-P5-02-23 | AC9 (`Lưu điểm danh`) | – | bấm nút chính | gửi ngay thay đổi chờ cùng `complete:true`; dòng tĩnh "Đã lưu điểm danh · 30 sinh viên · vắng 3" (số theo dữ liệu); không toast, không hộp thoại; nút → `Đã lưu` (khoá) tới khi có thay đổi mới; nhãn "Đã điểm danh" ở bộ chọn buổi |
| TC-P5-02-24 | AC10 (bộ chọn buổi) | lớp có buổi | mở `/attendance` không `?session=`; đổi buổi; `?session=` trong URL | mặc định buổi đang diễn ra (hoặc gần nhất đã qua); gần nhất trước, nhãn "Đã điểm danh"/"Chưa"; `?session=` chọn đúng |
| TC-P5-02-25 | AC10 (rỗng, đổi buổi giữ thay đổi) | lớp mới không buổi; buổi có thay đổi chưa lưu | mở; đổi buổi | "Lớp chưa có lịch buổi học." + nút `Tạo lịch buổi học` (về `/calendar`); đổi buổi khi còn thay đổi → tự lưu trước khi đổi (không mất) |
| TC-P5-02-26 | AC11 (`+ phát biểu`) | `/attendance` | bấm `+ phát biểu · +0,25` 1 lần; theo dõi `network`; sau đó bấm 3 lần liên tiếp | bộ đếm +1 ngay; `POST …/participation {student_id,session_id,type:"SPEAK",client_event_id}`; 3 lần = 3 sự kiện (mỗi lần `client_event_id` khác); nhãn đúng "+ phát biểu · +0,25" |
| TC-P5-02-27 | AC11 (idempotent, hoàn tác) | – | gửi lại cùng `client_event_id`; `DELETE …/participation/{id}` qua "Hoàn tác" 5 giây | không thêm sự kiện; xoá được |
| TC-P5-02-28 | AC11 (lỗi) | – | sinh viên `PENDING`/ngoài lớp; `$SVA`, `$ADM` gọi `POST` | `422 STUDENT_NOT_IN_COURSE`; `403` |
| TC-P5-02-29 | AC12 (đo 60 s, điện thoại) | lớp 30 SV, buổi chưa điểm danh; `resize 375 812` (hoặc điện thoại thật) | tay: mở `/attendance`, đánh `Vắng` 3 người, `Muộn` 1, ghi 1 phát biểu, `Lưu điểm danh`; bấm giờ | ≤ 60 s; 0 hộp thoại; `PUT` cuối đúng **1** lời gọi ghi cả lưới (kiểm `network`) |
| TC-P5-02-30 | AC12 (kịch bản tự động) | – | `$PW attendance.spec.ts -g 'attendance 375 scripted under 15s'` | xanh; ≤ 15 s; độ trễ mỗi chạm ≤ 100 ms |
| TC-P5-02-31 | AC13 (mất mạng → hàng đợi) | **nếu AC13 không bị cắt**; `playwright-cli` | `context.setOffline(true)`; đánh `Vắng` 3 người; xem banner; `setOffline(false)`; kiểm DB | `OfflineBanner` "Chưa có mạng · 3 thay đổi chờ lưu" hiện ngay; sau khi có mạng tự gửi theo thứ tự; banner "Đã lưu"; `select count(*) from attendance_records where status='ABSENT' and session_id=…` đủ 3 |
| TC-P5-02-32 | AC13 (tải lại giữ hàng đợi) | offline, đã đánh 3 vắng | tải lại trang; đóng/mở trình duyệt (`browser.newContext` cùng storage) | thay đổi còn nguyên; khoá `ep:draft:<userId>:attendance:<course>:<session>` có ở kho (`localStorage`/IndexedDB) |
| TC-P5-02-33 | AC13 (`Lưu điểm danh` offline) | offline | bấm `Lưu điểm danh` | xếp hàng cùng `complete:true`; khi có mạng → `done_at` đặt |
| TC-P5-02-34 | AC13 (409 là lỗi fatal) | offline; trong lúc đó người khác đổi cùng hàng | có mạng lại | hàng đợi dừng (không thử lại), chuyển sang luồng xung đột TC-10; không mất dữ liệu |
| TC-P5-02-35 | AC13 (đăng xuất hỏi trước) | còn thay đổi chờ | bấm `Đăng xuất` | giao diện **hỏi trước** (việc có thể mất dữ liệu); huỷ → ở lại |
| TC-P5-02-36 | AC13 (máy dùng chung) | A còn thay đổi, A đăng xuất (xác nhận), B đăng nhập cùng trình duyệt | kiểm kho và `marked_by` | kho không còn khoá `attendance:` của A; B không thấy `note`/dữ liệu của A; mọi hàng mới `marked_by`=B; dữ liệu A không bị gửi bằng phiên B |
| TC-P5-02-37 | AC13b (nếu **cắt** AC13) | PM ghi cắt sau 2 vòng FAIL AC13 | offline | nút `Lưu điểm danh` khoá; banner "Chưa có mạng. Điểm danh chưa được lưu."; lựa chọn giữ ở máy tới khi có mạng (tải lại vẫn thấy); nút `Lưu lại` hoạt động khi có mạng; **không** hiện "Đã lưu" khi chưa lưu. **Chỉ chạy khi PM xác nhận cắt AC13; nếu không, "KHÔNG ÁP DỤNG"** |
| TC-P5-02-38 | AC14 (AI hỏi đáp khớp DB) | `$SVB` (seed: vắng 2, 2 phát biểu = +0,50) | `/chat` hỏi "Em đã nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu rồi ạ?"; so với `select * from v_attendance_summary` | trả lời đúng 2 buổi vắng, +0,50; khối `attendance` {recorded,present,late,excused,absent,recent_absent[{session_no,date}]} và `participation` {speak_count,points_total,recent[]} khớp từng số với DB; không `[[…]]` |
| TC-P5-02-39 | AC14 (cập nhật) | giảng viên ghi thêm 1 phát biểu cho B | `$SVB` hỏi lại | +0,75; số không bị cache cũ |
| TC-P5-02-40 | AC14 (chưa có dữ liệu) | sinh viên mới chưa có điểm danh (`sv.moi@`/lớp mới) | hỏi | "Hệ thống chưa có dữ liệu điểm danh của bạn." và 0 lời gọi LLM (`/_test/llm/stats` không tăng) |
| TC-P5-02-41 | AC14 (hỏi hộ người khác, ghi chú) | `$SVA`; ghi chú điểm danh `CANARY-NOTE-9` ở hàng của A | hỏi "Em hỏi hộ bạn Kha nghỉ mấy buổi"; hỏi điểm danh của mình; grep `CANARY-NOTE-9` trong phản hồi | hỏi hộ → từ chối; ghi chú **không** đi vào kết quả tool (grep = 0) |
| TC-P5-02-42 | AC15 (phân quyền) | – | ma trận: `GET/PUT …/attendance`, `POST/DELETE …/participation` bằng `$SVA`, `$ADM`, TA, TEACHER, sinh viên `PENDING`/`REMOVED`/ngoài lớp | STUDENT/ADMIN/PENDING/REMOVED/ngoài lớp → `403`; TA, TEACHER ✓ |
| TC-P5-02-43 | AC15 (xoá phát biểu) | sự kiện do teacher tạo | TA `DELETE` sự kiện của teacher; teacher `DELETE` sự kiện của TA | TA xoá của người khác → `403`; TEACHER xoá được mọi sự kiện; người tạo xoá được của mình |
| TC-P5-02-44 | AC16 (trạng thái) | `/attendance` | tải chậm (3G): khung xương; `page.route` 500: nói chuyện gì xảy ra + nút `Thử lại` (dữ liệu đang chọn còn nguyên) | đúng |
| TC-P5-02-45 | AC16 (mock) | repo | `grep -rn "@/mock" frontend/src/features/attendance \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint`; `bash scripts/ui-antipatterns.sh` | `0`; xanh; `rc=0`. `ATTENDANCE_SEED`, `mock/roster.ts` phần điểm danh bị xoá chỉ **sau khi** cả US-P5-04, 06 xong (ghi, không FAIL nếu còn dùng) |
| TC-P5-02-46 | UI | – | `$PW attendance.spec.ts` (toàn bộ) | xanh; ghi số test, flaky |

## Nhánh lỗi (SRS 3.3)
TC-P5-02-08, 11, 12, 22, 31–36, 40. Thêm TC-P5-02-E1: hai người cùng điểm danh một buổi qua mạng chậm (3G) → luồng xung đột đúng; TC-P5-02-E2: `PUT` lặp lại sau mất mạng không tạo `audit_log` trùng.

## Phân quyền
TC-P5-02-03, 28, 42, 43, TC-P5-02-36 (phiên khác nhau không rò).

## Câu hỏi (Q-QC)
- **Q-QC-P5-03** AC12 "điện thoại thật": QC chỉ có giả lập 375 px (`playwright-cli`); phần điện thoại thật là "Bạn tự kiểm" của chủ dự án. Báo cáo ghi rõ giới hạn.
- **Q-QC-P5-04** AC13 chỉ cắt khi QC FAIL sau **2 vòng**; QC ghi vòng nào đang chấm trong report.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
