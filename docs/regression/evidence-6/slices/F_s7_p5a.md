# Slice F_s7_p5a: 93 ca còn N

### TC-P5-01-01 [AC1] pre: DB trống
STEP: `cd backend-go && goose -dir db/migrations postgres "$DATABASE_URL" up && goose … down && goose … up && goose … status \| grep 00013`
EXPECT: `up`/`down`(một bước)/`up` sạch; `00013` `Applied`

### TC-P5-01-02 [AC1] pre: repo
STEP: `grep -c '^ALTER TABLE' db/migrations/00013_crm.sql`; `grep -n '00013' docs/PROGRESS.md`
EXPECT: `1` (chỉ `class_sessions`: `UNIQUE (course_id,id)`, `attendance_done_at`, `attendance_done_by`); `PROGRESS.md` ghi `00013 crm`

### TC-P5-01-03 [AC2] pre: DB sau `00013`
STEP: `\d attendance_records`, `\d participation_events`; chèn: `status='X'`; `points` dương với `type='PENALTY'`; `points` âm với `type='SPEAK'`; `client_event_id` trùng; trùng `(course_id,session_id,student_id)`; `(course_id,session_id)` của lớp khác; `student_id` không ghi danh lớp
EXPECT: `23514` (CHECK) / `23505` / `23503` (FK phức hợp) tương ứng; `UNIQUE (course_id,session_id,student_id)` có; `version` có trong `CREATE`; `type` ∈ {SPEAK,BONUS,PENALTY}

### TC-P5-01-04 [AC3] pre: DB sau `00013`
STEP: `\d student_observations`, `\d student_risk`; `select column_name from information_schema.columns where table_name in ('v_attendance_summary','v_student_360')`; so với SRS 5.6; mọi chỉ mục bảng mới
EXPECT: hai view có đúng danh sách cột ghim ở SRS 5.6; mọi bảng mới có chỉ mục phức hợp bắt đầu bằng `course_id` (trừ `UNIQUE(client_event_id)` một cột); `sqlc diff` sạch

### TC-P5-01-06 [AC4 (không ghi gì)] pre: trước/sau TC-05
STEP: đếm `select count(*) from class_sessions, audit_log, outbox where course_id=…` trước và sau
EXPECT: không đổi (giao dịch chỉ đọc); không `audit_log`, không `outbox`

### TC-P5-01-11 [AC5 (tạo hàng loạt)] pre: xem trước hợp lệ
STEP: `generate` thật có `idem`; `select * from audit_log where entity='class_sessions' and action='generate'`; `select count(*) from outbox where topic='course.changed'`
EXPECT: `201 {created,first_session_no,last_session_no,skipped}`; một giao dịch (tất cả buổi + 1 dòng `audit_log` có số buổi + 1 outbox `course.changed`)

### TC-P5-01-13 [AC5 (không có key)] pre: –
STEP: `generate` thật không `Idempotency-Key`
EXPECT: lỗi (kỳ vọng `400`/`422`; ghi mã thực tế); không tạo

### TC-P5-01-14 [AC5 (lỗi giữa chừng)] pre: cài trigger tạm lỗi ở `class_sessions` insert thứ 5 (chỉ DB test)
STEP: `generate`
EXPECT: lỗi; **không** buổi nào được tạo (`count=0`); gỡ trigger

### TC-P5-01-17 [AC6 (lịch phản ánh ≤ 5 s)] pre: `/calendar` mở
STEP: sửa giờ buổi; đo thời gian tới khi `GET /calendar` (hoặc `/me/today`) đổi (ETag đổi)
EXPECT: ≤ 5 s

### TC-P5-01-19 [AC7 (đã điểm danh)] pre: buổi 1 lớp 1 (có `attendance_records`)
STEP: `DELETE`
EXPECT: `409 SESSION_HAS_ATTENDANCE`; không xoá; `PUT` sửa giờ / chủ đề vẫn được

### TC-P5-01-20 [AC7 (buổi bù)] pre: lớp có buổi 1…10
STEP: `generate` một ngày (`weekdays=[thứ của ngày đó]`, `from=to`) rồi `PUT` `topic:"Học bù"`
EXPECT: `session_no`=max+1 (11); chủ đề "Học bù"

### TC-P5-01-21 [AC7 (khoảng trống không lấp)] pre: lớp có buổi 1…5; xoá buổi 3 (chưa điểm danh); tạo buổi mới
STEP: `generate`
EXPECT: buổi mới = 6 (không lấp 3)

### TC-P5-01-22 [AC7 (xoá buổi cuối)] pre: lớp có buổi 1…5; xoá buổi 5; tạo buổi mới
STEP: `generate`
EXPECT: buổi mới = 5 (cấp lại số vì buổi 5 chưa từng có điểm danh)

### TC-P5-01-25 [AC9 (UI desktop)] pre: `playwright-cli`, giảng viên 1280
STEP: `/calendar`: nút `Tạo lịch buổi học`; mở ngăn kéo; chọn Thứ nhiều, Giờ, Phòng (placeholder "P.302"), Từ–Đến (`input type=date`), Ngày nghỉ; `Xem trước`; `Tạo 11 buổi`
EXPECT: nút chỉ Staff; bảng ≤ 60 dòng (Buổi, Ngày, Giờ) + "Sẽ tạo 11 buổi · bỏ 1 ngày nghỉ"; `Tạo 11 buổi` chỉ hoạt động sau xem trước; không câu mô tả dưới tiêu đề; danh sách buổi có menu `…` (`Sửa`, `Xoá`); lỗi 422 hiện tại ô lỗi

### TC-P5-01-26 [AC9 (sinh khoá mới mỗi lần)] pre: –
STEP: theo dõi `network`: `Xem trước` rồi `Tạo N buổi`
EXPECT: `Idempotency-Key` của lần tạo khác (không dùng lại) khoá nào của xem trước (xem trước không có key)

### TC-P5-01-27 [AC9 (việc "Thiết lập lớp mới")] pre: `$CN` chưa có buổi
STEP: `/me/today` bằng giảng viên → tạo buổi
EXPECT: việc "Tạo lịch buổi học" (`Href=/calendar?course=…`) tự tick khi có buổi

### TC-P5-01-28 [AC9 (375 px)] pre: `resize 375 812`
STEP: mở ngăn kéo
EXPECT: toàn màn hình; không tràn ngang; vùng chạm ≥ 44 px

### TC-P5-01-29 [AC10 (Hôm nay)] pre: buổi `S` hôm nay `starts_at ≤ now ≤ ends_at+24h`, `attendance_done_at IS NULL` (dùng buổi seed hôm nay hoặc `generate` ngày hôm nay)
STEP: `GET /me/today -H "$TCH"`
EXPECT: mục `SESSION_UNMARKED`: "Điểm danh buổi {no} đang diễn ra · {mã lớp}" (đang diễn ra) hoặc "Buổi {no} đã kết thúc, chưa điểm danh · {mã lớp}" (`Overdue=true`), `Href=/attendance?course=…&session=…`; ≤ 3 mục mỗi lớp mới nhất trước

### TC-P5-01-30 [AC10 (vị trí bậc, biến)] pre: cùng lúc có `EXAM_GRADE_ERROR` (nếu có)
STEP: quan sát thứ tự; sau `Lưu điểm danh` (US-P5-02)
EXPECT: `SESSION_UNMARKED` (bậc 12) đứng trước/sau đúng bảng bậc, không trùng bậc 15; biến ≤ 60 s sau khi lưu điểm danh

### TC-P5-01-31 [AC10 (vai, lớp lưu trữ)] pre: –
STEP: `GET /me/today` bằng `$SVA`, `$ADM`; lớp `ARCHIVED`
EXPECT: sinh viên/ADMIN không thấy; lớp lưu trữ không có

### TC-P5-01-32 [AC11 (hai khoá, cùng tuần)] pre: hai giảng viên (teacher và TA) cùng tạo cho cùng tuần
STEP: `&`+`wait` hai `generate` khác khoá
EXPECT: người sau `skipped DUPLICATE`; không `session_no` trùng; không `500`

### TC-P5-01-33 [AC11 (mất mạng)] pre: `playwright-cli` `setOffline` khi bấm `Tạo N buổi`
STEP: bấm; online; thử lại
EXPECT: nút trở lại; thử lại cùng `Idempotency-Key` không tạo đôi (`count` đúng)

### TC-P5-01-34 [UI] pre: frontend
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW calendar.spec.ts -g 'generate sessions preview and create\|edit delete session'`; `$PW today.spec.ts -g 'course setup sessions step ticks'`
EXPECT: `0`; xanh

### TC-P5-02-02 [AC1 (không PENDING/REMOVED)] pre: 1 sinh viên `PENDING`, 1 `REMOVED`
STEP: gọi lưới
EXPECT: không có trong `students`

### TC-P5-02-06 [AC2 (hàng giống hệt bị bỏ qua)] pre: –
STEP: `PUT` với 29 hàng giữ nguyên + 1 hàng đổi
EXPECT: chỉ hàng đổi `version`+1; `audit_log` chỉ nêu 1 `{student_id,from,to}`

### TC-P5-02-07 [AC2 (hiệu năng)] pre: 30 hàng
STEP: đo `PUT` 20 lần (`curl -w '%{time_total}'`)
EXPECT: p95 ≤ 500 ms

### TC-P5-02-09 [AC3 (hàng không đổi không xung đột)] pre: –
STEP: T2 gửi hàng X trùng DB hiện tại
EXPECT: không `409`

### TC-P5-02-10 [AC3 (UI)] pre: `playwright-cli` hai ngữ cảnh: teacher và TA cùng `/attendance?session=$S10`
STEP: cả hai đổi cùng sinh viên; teacher lưu trước; TA lưu
EXPECT: TA thấy `InlineNotice` "Thầy/Cô … vừa đổi N dòng. Giữ của tôi / Lấy của …" với chọn theo từng dòng; các thay đổi còn lại của TA giữ; chọn xong gửi lại với `version` mới thành công

### TC-P5-02-12 [AC4 (DB lỗi)] pre: dừng postgres giữa `PUT` (script: `docker compose stop postgres` ngay khi gửi)
STEP: `PUT`
EXPECT: `503`; không ghi nửa chừng (`count` không đổi sau khi bật lại)

### TC-P5-02-14 [AC5 (sửa sau)] pre: buổi đã `done_at`
STEP: `PUT` đổi 1 hàng (`complete:false` hoặc `true`)
EXPECT: `200`; `action='edit_after_complete'` có `before`/`after` đủ dựng lại hàng nào đổi từ gì sang gì; không giới hạn thời gian; `done_at` không bị đặt lại

### TC-P5-02-15 [AC5 (autosave không hoàn tất)] pre: buổi chưa hoàn tất
STEP: `PUT` `complete:false`
EXPECT: `done_at` vẫn `NULL`

### TC-P5-02-16 [AC6 (375 px)] pre: `resize 375 812`, `/attendance?session=$S10`
STEP: xem danh sách; bấm một hàng; chọn `Vắng`
EXPECT: danh sách hàng ngăn đường kẻ 1 px (không card); mỗi hàng: tên + MSSV rút gọn + trạng thái bằng chữ ("Có mặt" nhạt; "Vắng" đỏ); bấm hàng mở điều khiển 4 lựa chọn ngay dưới hàng (`Có mặt`·`Muộn`·`Vắng phép`·`Vắng`, mỗi nút cao ≥ 44 px); đổi một người ≤ 2 chạm; người không chạm giữ "Có mặt"; không `Dialog`; không tràn ngang (`scrollWidth ≤ innerWidth`); zoom 200 % vẫn thao tác được

### TC-P5-02-17 [AC6 (a11y)] pre: –
STEP: `$PW a11y.spec.ts -g attendance`
EXPECT: 0 vi phạm nghiêm trọng

### TC-P5-02-18 [AC7 (≥ 1100 px)] pre: `resize 1280 800`
STEP: xem lưới; cuộn; bấm `1`–`4`, `↑`/`↓`, `Shift+1…4`
EXPECT: `AttendanceGrid` 4 cột radio + cột `Phát biểu`; cột tên và header dính khi cuộn; vòng focus nhìn thấy; `1`–`4` đặt trạng thái hàng focus rồi xuống hàng kế; `Shift+1…4` không chuyển hàng

### TC-P5-02-19 [AC7 (30 SV bằng phím ≤ 60 s)] pre: lớp 30 sinh viên
STEP: `playwright-cli`: gõ chuỗi phím cho 30 hàng (3 vắng, 1 muộn), `Lưu điểm danh`; đo thời gian
EXPECT: tổng ≤ 60 s; kết quả DB đúng

### TC-P5-02-20 [AC8 (lạc quan ≤ 100 ms, hoàn tác)] pre: `/attendance`
STEP: đổi một trạng thái; đo thời gian tới khi UI đổi; xem dòng tĩnh
EXPECT: UI đổi ≤ 100 ms; dòng "Đã đánh Vắng · Hoàn tác" 5 giây; không toast, không hộp thoại; Hoàn tác trả trạng thái trước

### TC-P5-02-21 [AC8 (gom 2 giây, chỉ báo lưu)] pre: –
STEP: đổi 3 hàng liên tiếp trong 1 s; theo dõi `network`; xem đầu trang
EXPECT: đúng 1 `PUT` sau 2 s yên lặng (`complete:false`); "Đang lưu…" → "Đã lưu HH:MM"

### TC-P5-02-22 [AC8 (5xx)] pre: `page.route` trả 500 cho `PUT`
STEP: đổi trạng thái
EXPECT: trạng thái hiển thị hoàn lại + nút `Thử lại`; chữ đã chọn không mất; `Thử lại` thành công khi bỏ chặn

### TC-P5-02-23 [AC9 (`Lưu điểm danh`)] pre: –
STEP: bấm nút chính
EXPECT: gửi ngay thay đổi chờ cùng `complete:true`; dòng tĩnh "Đã lưu điểm danh · 30 sinh viên · vắng 3" (số theo dữ liệu); không toast, không hộp thoại; nút → `Đã lưu` (khoá) tới khi có thay đổi mới; nhãn "Đã điểm danh" ở bộ chọn buổi

### TC-P5-02-24 [AC10 (bộ chọn buổi)] pre: lớp có buổi
STEP: mở `/attendance` không `?session=`; đổi buổi; `?session=` trong URL
EXPECT: mặc định buổi đang diễn ra (hoặc gần nhất đã qua); gần nhất trước, nhãn "Đã điểm danh"/"Chưa"; `?session=` chọn đúng

### TC-P5-02-25 [AC10 (rỗng, đổi buổi giữ thay đổi)] pre: lớp mới không buổi; buổi có thay đổi chưa lưu
STEP: mở; đổi buổi
EXPECT: "Lớp chưa có lịch buổi học." + nút `Tạo lịch buổi học` (về `/calendar`); đổi buổi khi còn thay đổi → tự lưu trước khi đổi (không mất)

### TC-P5-02-26 [AC11 (`+ phát biểu`)] pre: `/attendance`
STEP: bấm `+ phát biểu · +0,25` 1 lần; theo dõi `network`; sau đó bấm 3 lần liên tiếp
EXPECT: bộ đếm +1 ngay; `POST …/participation {student_id,session_id,type:"SPEAK",client_event_id}`; 3 lần = 3 sự kiện (mỗi lần `client_event_id` khác); nhãn đúng "+ phát biểu · +0,25"

### TC-P5-02-29 [AC12 (đo 60 s, điện thoại)] pre: lớp 30 SV, buổi chưa điểm danh; `resize 375 812` (hoặc điện thoại thật)
STEP: tay: mở `/attendance`, đánh `Vắng` 3 người, `Muộn` 1, ghi 1 phát biểu, `Lưu điểm danh`; bấm giờ
EXPECT: ≤ 60 s; 0 hộp thoại; `PUT` cuối đúng **1** lời gọi ghi cả lưới (kiểm `network`)

### TC-P5-02-30 [AC12 (kịch bản tự động)] pre: –
STEP: `$PW attendance.spec.ts -g 'attendance 375 scripted under 15s'`
EXPECT: xanh; ≤ 15 s; độ trễ mỗi chạm ≤ 100 ms

### TC-P5-02-31 [AC13 (mất mạng → hàng đợi)] pre: **nếu AC13 không bị cắt**; `playwright-cli`
STEP: `context.setOffline(true)`; đánh `Vắng` 3 người; xem banner; `setOffline(false)`; kiểm DB
EXPECT: `OfflineBanner` "Chưa có mạng · 3 thay đổi chờ lưu" hiện ngay; sau khi có mạng tự gửi theo thứ tự; banner "Đã lưu"; `select count(*) from attendance_records where status='ABSENT' and session_id=…` đủ 3

### TC-P5-02-32 [AC13 (tải lại giữ hàng đợi)] pre: offline, đã đánh 3 vắng
STEP: tải lại trang; đóng/mở trình duyệt (`browser.newContext` cùng storage)
EXPECT: thay đổi còn nguyên; khoá `ep:draft:<userId>:attendance:<course>:<session>` có ở kho (`localStorage`/IndexedDB)

### TC-P5-02-33 [AC13 (`Lưu điểm danh` offline)] pre: offline
STEP: bấm `Lưu điểm danh`
EXPECT: xếp hàng cùng `complete:true`; khi có mạng → `done_at` đặt

### TC-P5-02-34 [AC13 (409 là lỗi fatal)] pre: offline; trong lúc đó người khác đổi cùng hàng
STEP: có mạng lại
EXPECT: hàng đợi dừng (không thử lại), chuyển sang luồng xung đột TC-10; không mất dữ liệu

### TC-P5-02-35 [AC13 (đăng xuất hỏi trước)] pre: còn thay đổi chờ
STEP: bấm `Đăng xuất`
EXPECT: giao diện **hỏi trước** (việc có thể mất dữ liệu); huỷ → ở lại

### TC-P5-02-36 [AC13 (máy dùng chung)] pre: A còn thay đổi, A đăng xuất (xác nhận), B đăng nhập cùng trình duyệt
STEP: kiểm kho và `marked_by`
EXPECT: kho không còn khoá `attendance:` của A; B không thấy `note`/dữ liệu của A; mọi hàng mới `marked_by`=B; dữ liệu A không bị gửi bằng phiên B

### TC-P5-02-37 [AC13b (nếu **cắt** AC13)] pre: PM ghi cắt sau 2 vòng FAIL AC13
STEP: offline
EXPECT: nút `Lưu điểm danh` khoá; banner "Chưa có mạng. Điểm danh chưa được lưu."; lựa chọn giữ ở máy tới khi có mạng (tải lại vẫn thấy); nút `Lưu lại` hoạt động khi có mạng; **không** hiện "Đã lưu" khi chưa lưu. **Chỉ chạy khi PM xác nhận cắt AC13; nếu không, "KHÔNG ÁP DỤNG"**

### TC-P5-02-40 [AC14 (chưa có dữ liệu)] pre: sinh viên mới chưa có điểm danh (`sv.moi@`/lớp mới)
STEP: hỏi
EXPECT: "Hệ thống chưa có dữ liệu điểm danh của bạn." và 0 lời gọi LLM (`/_test/llm/stats` không tăng)

### TC-P5-02-41 [AC14 (hỏi hộ người khác, ghi chú)] pre: `$SVA`; ghi chú điểm danh `CANARY-NOTE-9` ở hàng của A
STEP: hỏi "Em hỏi hộ bạn Kha nghỉ mấy buổi"; hỏi điểm danh của mình; grep `CANARY-NOTE-9` trong phản hồi
EXPECT: hỏi hộ → từ chối; ghi chú **không** đi vào kết quả tool (grep = 0)

### TC-P5-02-44 [AC16 (trạng thái)] pre: `/attendance`
STEP: tải chậm (3G): khung xương; `page.route` 500: nói chuyện gì xảy ra + nút `Thử lại` (dữ liệu đang chọn còn nguyên)
EXPECT: đúng

### TC-P5-02-45 [AC16 (mock)] pre: repo
STEP: `grep -rn "@/mock" frontend/src/features/attendance \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint`; `bash scripts/ui-antipatterns.sh`
EXPECT: `0`; xanh; `rc=0`. `ATTENDANCE_SEED`, `mock/roster.ts` phần điểm danh bị xoá chỉ **sau khi** cả US-P5-04, 06 xong (ghi, không FAIL nếu còn dùng)

### TC-P5-02-46 [UI] pre: –
STEP: `$PW attendance.spec.ts` (toàn bộ)
EXPECT: xanh; ghi số test, flaky

### TC-P5-03-01 [AC1 (migration)] pre: DB sau `00013`
STEP: `goose up/down/up`; `grep -c '^ALTER TABLE' db/migrations/00014_observation.sql`; `$PSQL -c "select table_name,column_name,column_default from information_schema.columns where (table_name='user_settings' and column_name='track_study_time') or (table_name='courses' and column_name='allow_study_time_opt_out')"`; `grep -n '00014' docs/PROGRESS.md`
EXPECT: sạch; `2`; hai dòng mặc định `true`; `PROGRESS.md` ghi `00014 observation`

### TC-P5-03-02 [AC1 (lược đồ)] pre: –
STEP: `\d study_time_daily`; chèn `area='X'`; `active_seconds=-1`; `active_seconds=86401`; trùng `(course_id,student_id,date,area)`; `\d insight_reports`
EXPECT: `23514` ×3, `23505`; `UNIQUE` có `course_id`; `area` ∈ {CHAT,THREADS,PRACTICE,LIBRARY,CALENDAR}; `insight_reports` đúng SRS 5.8

### TC-P5-03-03 [AC2 (chỉ Redis)] pre: `$SVA` đã `Đã hiểu`, ghi danh lớp 1
STEP: bật `log_statement='all'` Postgres; gửi 1 nhịp; `grep -ciE 'insert\|update' pg.log` quanh thời điểm
EXPECT: `204`; khoá Redis `ep:study:{course}:{student}:{yyyymmdd}:CHAT` += 30 (`$RDS GET` trước/sau); **0** câu `INSERT`/`UPDATE` từ request này tới Postgres

### TC-P5-03-04 [AC2 (độ trễ, tải)] pre: –
STEP: `k6 run benchmarks/load/heartbeat.js` (10 req/s, 60 s)
EXPECT: p95 < 20 ms; 0 lỗi 5xx

### TC-P5-03-05 [AC3 (hai tab)] pre: `$SVA`
STEP: gửi nhịp `tab_id:t1`, rồi `tab_id:t2` trong cùng khung 30 s; rồi nhịp ở khung kế
EXPECT: nhịp thứ hai cùng khung → `204` **không** cộng (+30 tổng); khung khác đều tính (+30 mỗi khung)

### TC-P5-03-06 [AC3 (10 phút hai tab, đồng hồ giả)] pre: `playwright-cli`, hai tab `/chat` cùng người dùng, `page.clock`
STEP: chạy 10 phút giả lập, thao tác ở cả hai
EXPECT: tổng ≤ 10 phút + 30 s (kiểm Redis); không gấp đôi

### TC-P5-03-07 [AC4 (tab ẩn)] pre: `/chat` mở, đồng hồ giả
STEP: `document.visibilityState='hidden'` (mô phỏng `Page.setWebLifecycleState`/`visibilitychange`); đếm `POST /activity/heartbeat`
EXPECT: không nhịp nào khi ẩn; hiện lại + thao tác → gửi tiếp ≤ 1 s

### TC-P5-03-08 [AC4 (idle 2 phút)] pre: `/chat` mở, thao tác lúc t0, rồi dừng
STEP: chạy 5 phút giả lập
EXPECT: nhịp ngừng sau mốc 2 phút (nhịp cuối trước mốc idle); thao tác lại → tiếp tục

### TC-P5-03-09 [AC4 (dọn khi đổi trang)] pre: –
STEP: `/chat` → `/library` → `/` (không đo)
EXPECT: không rò bộ hẹn giờ (không nhịp thừa sau khi rời; đổi `area` ghi đúng khu vực mới)

### TC-P5-03-10 [AC4 (mạng đứt, 403/422)] pre: –
STEP: `setOffline` rồi online; `page.route` trả 403/422 cho nhịp
EXPECT: bỏ nhịp, không xếp hàng; ngừng im lặng (không báo người dùng)

### TC-P5-03-11 [AC5 (mở 10 phút, thao tác 3 phút)] pre: tab hiện suốt 10 phút (đồng hồ giả), thao tác 3 phút đầu rồi dừng
STEP: chạy; đọc `study_time_daily` sau rollup (hoặc Redis)
EXPECT: **300 ± 30 s** cho ngày/khu vực đó

### TC-P5-03-12 [AC5 (tab ẩn 4/10 phút)] pre: tab ẩn 4 phút
STEP: chạy
EXPECT: 4 phút đó không cộng

### TC-P5-03-13 [AC5 (chỉ khu vực)] pre: –
STEP: `select * from study_time_daily` + khoá Redis
EXPECT: chỉ `area` ∈ 5 khu vực; không nội dung, đường dẫn, tiêu đề, tên; trang `/`, `/me`, `/exams/**`, trang giảng viên **không** gửi nhịp (đếm request khi duyệt các trang đó)

### TC-P5-03-14 [AC6 (rollup tuyệt đối)] pre: khoá Redis có giá trị V
STEP: chờ job `heartbeat.rollup` (mỗi phút); `select active_seconds from study_time_daily …`
EXPECT: `= GREATEST(cũ, V)` (ghi tuyệt đối, không cộng dồn); chạy lại → không đổi

### TC-P5-03-15 [AC6 (giao đôi, worker chết)] pre: –
STEP: `docker compose kill -s KILL worker` giữa rollup (vòng lặp theo log) rồi `up`; chạy thêm 2 phút
EXPECT: kết quả không đổi (không đôi, không âm)

### TC-P5-03-16 [AC6 (Redis mất khoá)] pre: giá trị DB `X`; `$RDS DEL` khoá
STEP: rollup
EXPECT: DB giữ `X`, không giảm, không âm

### TC-P5-03-17 [AC6 (nửa đêm, TTL)] pre: nhịp 23:59 và 00:01 giờ VN (đặt khoá tay `$RDS` theo ngày)
STEP: rollup
EXPECT: tách đúng hai hàng theo ngày `Asia/Ho_Chi_Minh`; `$RDS TTL` khoá ngày cũ ≤ 8 ngày

### TC-P5-03-18 [AC7 (hộp thoại lần đầu)] pre: `users.tracking_notice_ack_at IS NULL`
STEP: `playwright-cli`: đăng nhập sinh viên, vào trang bất kỳ
EXPECT: `Dialog` một lần: nội dung (SRS 7.5) nói EduPilot ghi **thời gian** dùng Chat, Threads, Luyện đề, Thư viện, Lịch khi tab mở và đang thao tác; không ghi nội dung; giảng viên xem được; không dùng để tính điểm; "Bạn có thể tắt ở Cài đặt." khi sinh viên có lớp cho phép tắt; nút `Đã hiểu`

### TC-P5-03-20 [AC7 (chưa ack)] pre: sinh viên chưa ack
STEP: gửi nhịp trực tiếp bằng `curl`
EXPECT: `204` **không** cộng; hook cũng không gửi

### TC-P5-03-21 [AC7 (staff không đo)] pre: TA/TEACHER/ADMIN
STEP: đăng nhập; gửi nhịp
EXPECT: không thấy hộp thoại; không cộng (nhịp `204` không cộng hoặc `403` — ghi)

### TC-P5-03-22 [AC8 (luật cứng, bắc cầu)] pre: repo
STEP: `cd backend-go && go list -deps ./internal/grade/... 2>&1 \| grep -E '/internal/(activity\|crm)'; go test ./internal/activity -run 'TestGradeNeverImportsActivity\|TestGradeQueriesNeverReadStudyTime' -v`
EXPECT: không dòng nào (hoặc "no packages" khi `grade` chưa có); `ok`

### TC-P5-03-23 [AC8 (tự thử đỏ — đường 1)] pre: nhánh tạm `qc/tmp-grade-import`
STEP: thêm `backend-go/internal/grade/zz/zz.go` `package zz; import _ ".../internal/today"` (khi `today` import `crm`) rồi chạy bài kiểm
EXPECT: `FAIL` (bắt phụ thuộc bắc cầu). Xoá nhánh tạm

### TC-P5-03-24 [AC8 (tự thử đỏ — đường 2)] pre: nhánh tạm
STEP: thêm vào `grade/zz` một lời gọi `q.CRMListStudents(...)` (hàm sinh từ `crm.sql.go`)
EXPECT: `FAIL` (bắt gọi `store.Queries`). Xoá nhánh tạm. (Nếu tên hàm khác: lấy từ `store/crm.sql.go`; ghi lại)

### TC-P5-03-25 [AC8 (quét SQL)] pre: repo
STEP: `grep -rln 'study_time_daily\|student_risk\|heartbeat\|v_student_360' backend-go/internal/grade backend-go/internal/store/queries/grade*.sql 2>/dev/null`
EXPECT: 0 tệp

### TC-P5-03-28 [AC9 (rỗng, guard)] pre: sinh viên mới
STEP: gọi `/me/study-time`
EXPECT: mảng rỗng, không lỗi; `grep` route trong `TestAllCourseRoutesGuarded` (`go test ./internal/httpapi/... -run TestAllCourseRoutesGuarded -v` → `ok`)

### TC-P5-03-29 [AC10 (Redis lỗi)] pre: `docker compose stop redis`
STEP: gửi nhịp
EXPECT: `204`; `activity_heartbeat_dropped_total` tăng (`curl $GW/metrics`); log không PII; không `5xx`; bật lại redis

### TC-P5-03-31 [AC10 (trần ngày)] pre: khoá tổng `ep:study:total:{uid}:{yyyymmdd}` = 43.170 (đặt bằng `$RDS`)
STEP: gửi 2 nhịp
EXPECT: nhịp 1 → +30 (43.200); nhịp 2 → bị bỏ (`204`, không cộng); trần tính cả hai lớp × hai khu vực (đặt khoá tổng từ nhiều lớp)

### TC-P5-03-32 [AC10 (burst 4 nhịp/phút)] pre: –
STEP: gửi 6 nhịp trong 60 s cho 6 khung khác nhau (chờ đủ 30 s mỗi khung là quá dài → thay: gửi 6 nhịp liên tiếp nhanh, mỗi nhịp `tab_id` khác)
EXPECT: ≥ 2 nhịp bị bỏ theo cách `204` (không `429`)

### TC-P5-03-33 [AC10 (`flushed` không phình)] pre: sau rollup nhiều chu kỳ
STEP: `$RDS HLEN ep:study:flushed`; so với `SCARD ep:study:dirty`
EXPECT: không tăng vô hạn; khoá đã `SREM` khỏi `dirty` thì `HDEL` ở `flushed`

### TC-P5-03-34 [AC11 (lớp cho tắt: tắt/bật)] pre: lớp 1 (`allow_study_time_opt_out=true`), `$SVA` đã `Đã hiểu`
STEP: `PUT /me/settings {"track_study_time":false,"version":N}`; gửi nhịp ở lớp 1; chờ rollup; `PUT` bật lại; gửi nhịp
EXPECT: `200`; nhịp → `204` **không đo** (không tạo khoá `ep:study:{X}:…` mới; `study_time_daily` không đổi), hiệu lực ngay; dữ liệu cũ giữ và đọc được; bật lại → nhịp kế đo tiếp

### TC-P5-03-36 [AC12a (lớp không cho tắt)] pre: lớp 2 (`allow_study_time_opt_out=false`), `$SVA` ghi danh cả hai
STEP: `PUT /me/settings {track_study_time:false}`; gửi nhịp ở lớp 2; `/settings`
EXPECT: `200` (lưu, không tác dụng ở lớp 2); nhịp vẫn đo; công tắc lớp 2 **bật và khoá**, tooltip lý do ở chính công tắc, không dòng mô tả

### TC-P5-03-37 [AC12b (lớp mới, Admin không chọn)] pre: –
STEP: `POST /admin/courses` có trường `allow_study_time_opt_out`; form Mở lớp; lớp mới
EXPECT: trường bị từ chối/không nhận (`422` hoặc bị bỏ qua — ghi); form không có trường; lớp mới mặc định `true`

### TC-P5-03-40 [AC12c (đua)] pre: `$CN` chưa sinh viên
STEP: 50 lượt song song: một luồng ghi danh sinh viên đầu tiên, một luồng `PUT` đổi cờ (`&`+`wait`, lặp 50 lần với lớp mới mỗi lần)
EXPECT: mỗi lượt chỉ một trong hai kết cục: (1) `PUT` thắng rồi sinh viên vào lớp dưới luật mới, hoặc (2) sinh viên vào trước rồi `PUT` `409`; không lượt nào có sinh viên ghi danh dưới giá trị cũ mà cờ bị đổi

### TC-P5-03-41 [AC12d (UI)] pre: `playwright-cli`
STEP: `/class/settings` bằng teacher (chưa/đã có sinh viên), TA
EXPECT: teacher thấy công tắc đổi được khi chưa có sinh viên, khi đã khoá chỉ hiện giá trị + tooltip lý do ở chính công tắc; TA chỉ đọc; `GET $Q` (Member) trả cờ

### TC-P5-03-42 [AC12e] pre: –
STEP: `GET /me/settings` bằng `$ADM`; `PUT /me/settings` có tham số danh tính
EXPECT: ADMIN chỉ thấy cài đặt của mình; không đọc `track_study_time` của sinh viên

### TC-P5-03-43 [AC13 (hai lớp, hai cấu hình)] pre: `$SVA` ghi danh lớp X=lớp 1 (cho tắt), Y=lớp 2 (không cho tắt), đã `Đã hiểu`, `track_study_time=false`
STEP: gửi nhịp `CHAT` cho X rồi cho Y (khung 30 s khác nhau, cùng ngày); rollup; `GET /courses/{X}/me/study-time`; `GET /courses/{Y}/me/study-time`
EXPECT: X: `204` không đo, **không chiếm** khung 30 s của `S`; Y: đo (+30); `study_time_daily` có hàng của Y, **không có** hàng của X; X `tracking:{enabled:false,locked:false}`, Y `tracking:{enabled:true,locked:true}`; `/settings` hai dòng (X Tắt đổi được; Y Bật khoá); bật lại → X đo từ nhịp kế

### TC-P5-03-44 [UI] pre: –
STEP: `$PW activity.spec.ts`; `bash scripts/ui-antipatterns.sh`
EXPECT: xanh; `rc=0`; ghi số test
