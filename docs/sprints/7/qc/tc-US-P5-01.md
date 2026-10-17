# QC test case — US-P5-01 (lịch buổi học: xem trước, tạo hàng loạt, sửa, huỷ, buổi bù, migration `00013`)
Nguồn: `docs/specs/FEAT-crm-360/US.md` US-P5-01 AC1…AC11 + `SRS.md` 2, 3.3, 4.2, 5, 6.1, 7.1, 9.1; `proposals.md` #5, #6; `docs/phases/P5.md` L1; FLOWS F7 bước 1–2. Hộp đen, không đọc code dev.
**Biến shell** (SRS 9.1): `GW`; `C1` = lớp 1 (`761987`); `Q=$GW/api/v1/courses/$C1`; `SVA`/`SVB`/`SVC` = `Authorization` của `sv.gioi@`/`sv.kha@`/`sv.nguyco@edupilot.local`; `TA_`, `TCH`, `ADM`; `j='curl -sk -H "Content-Type: application/json"'`; `idem`; `PSQL`, `RDS`; `PW='pnpm -C frontend exec playwright test'`; `S10` = buổi 10 lớp 1. Mật khẩu / token lấy từ `scripts/seed.mjs`. Giao diện nghiệm thu tay bằng `playwright-cli`. Đồng hồ giả không có trên stack thật: dùng `date` thật và dữ liệu tương đối ("hôm nay", "hôm qua"). Lớp mới để thử không ảnh hưởng seed: tạo bằng `POST /admin/courses` (như `scripts/seed.mjs`) → `$CN`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P5-01-01 | AC1 | DB trống | `cd backend-go && goose -dir db/migrations postgres "$DATABASE_URL" up && goose … down && goose … up && goose … status \| grep 00013` | `up`/`down`(một bước)/`up` sạch; `00013` `Applied` |
| TC-P5-01-02 | AC1 | repo | `grep -c '^ALTER TABLE' db/migrations/00013_crm.sql`; `grep -n '00013' docs/PROGRESS.md` | `1` (chỉ `class_sessions`: `UNIQUE (course_id,id)`, `attendance_done_at`, `attendance_done_by`); `PROGRESS.md` ghi `00013 crm` |
| TC-P5-01-03 | AC2 | DB sau `00013` | `\d attendance_records`, `\d participation_events`; chèn: `status='X'`; `points` dương với `type='PENALTY'`; `points` âm với `type='SPEAK'`; `client_event_id` trùng; trùng `(course_id,session_id,student_id)`; `(course_id,session_id)` của lớp khác; `student_id` không ghi danh lớp | `23514` (CHECK) / `23505` / `23503` (FK phức hợp) tương ứng; `UNIQUE (course_id,session_id,student_id)` có; `version` có trong `CREATE`; `type` ∈ {SPEAK,BONUS,PENALTY} |
| TC-P5-01-04 | AC3 | DB sau `00013` | `\d student_observations`, `\d student_risk`; `select column_name from information_schema.columns where table_name in ('v_attendance_summary','v_student_360')`; so với SRS 5.6; mọi chỉ mục bảng mới | hai view có đúng danh sách cột ghim ở SRS 5.6; mọi bảng mới có chỉ mục phức hợp bắt đầu bằng `course_id` (trừ `UNIQUE(client_event_id)` một cột); `sqlc diff` sạch |
| TC-P5-01-05 | AC4 (xem trước) | `$CN` đã có giảng viên; chưa buổi nào | `j -X POST "$Q/sessions/generate?dry_run=true" -H "$TCH" -d '{"weekdays":[1],"start_time":"09:00","end_time":"11:30","from":"2026-10-12","to":"2026-12-28","exclude_dates":["2026-11-02"]}'` (không `Idempotency-Key`) | `200 {would_create,skipped}`; `would_create` 11 phần tử (`session_no`,`starts_at`,`ends_at`), `skipped[0].reason="EXCLUDED"` |
| TC-P5-01-06 | AC4 (không ghi gì) | trước/sau TC-05 | đếm `select count(*) from class_sessions, audit_log, outbox where course_id=…` trước và sau | không đổi (giao dịch chỉ đọc); không `audit_log`, không `outbox` |
| TC-P5-01-07 | AC4 (bằng tạo thật) | sau TC-05 | cùng tham số nhưng không `dry_run`, `Idempotency-Key` mới | tạo đúng tập `would_create` (cùng `session_no`, cùng giờ) |
| TC-P5-01-08 | AC4 (skipped) | đã có buổi trùng `starts_at` | `dry_run` lại | buổi trùng ở `skipped` với `reason:"DUPLICATE"`; ngày loại trừ `EXCLUDED` |
| TC-P5-01-09 | AC4 (tham số sai) | – | `from>to`; `end_time<=start_time`; `weekdays:[]`; `weekdays:[0]`; `weekdays:[8]`; `room` 41 ký tự; `from:"2026-02-30"` | `422` + `details[].field` mỗi ca |
| TC-P5-01-10 | AC4 (> 60 buổi) | – | khoảng ngày cho > 60 buổi (`weekdays:[1..5]`, 4 tháng) | `422 TOO_MANY_SESSIONS` nêu số buổi và giới hạn 60 |
| TC-P5-01-11 | AC5 (tạo hàng loạt) | xem trước hợp lệ | `generate` thật có `idem`; `select * from audit_log where entity='class_sessions' and action='generate'`; `select count(*) from outbox where topic='course.changed'` | `201 {created,first_session_no,last_session_no,skipped}`; một giao dịch (tất cả buổi + 1 dòng `audit_log` có số buổi + 1 outbox `course.changed`) |
| TC-P5-01-12 | AC5 (idempotent) | – | gửi lại cùng `Idempotency-Key`; rồi khác khoá cùng tham số | cùng khoá → cùng phản hồi, không tạo thêm; khác khoá → `created=0`, mọi buổi ở `skipped` `DUPLICATE` |
| TC-P5-01-13 | AC5 (không có key) | – | `generate` thật không `Idempotency-Key` | lỗi (kỳ vọng `400`/`422`; ghi mã thực tế); không tạo |
| TC-P5-01-14 | AC5 (lỗi giữa chừng) | cài trigger tạm lỗi ở `class_sessions` insert thứ 5 (chỉ DB test) | `generate` | lỗi; **không** buổi nào được tạo (`count=0`); gỡ trigger |
| TC-P5-01-15 | AC6 (sửa một buổi) | buổi `S` (version N) | `PUT $Q/sessions/$S {starts_at,ends_at,room,topic,version:N} -H "$TCH"` | `200`, `version=N+1`, `session_no` không đổi; có `audit_log` |
| TC-P5-01-16 | AC6 (lỗi) | – | sai `version`; `ends_at<=starts_at`; `starts_at` trùng buổi khác; sửa `session_no` trong thân | `409 VERSION_CONFLICT` kèm bản hiện tại; `422`; `409 SESSION_DUPLICATE`; `session_no` không đổi |
| TC-P5-01-17 | AC6 (lịch phản ánh ≤ 5 s) | `/calendar` mở | sửa giờ buổi; đo thời gian tới khi `GET /calendar` (hoặc `/me/today`) đổi (ETag đổi) | ≤ 5 s |
| TC-P5-01-18 | AC7 (xoá) | buổi chưa điểm danh | `DELETE $Q/sessions/$S -H "$TCH"` | `204`; `audit_log` |
| TC-P5-01-19 | AC7 (đã điểm danh) | buổi 1 lớp 1 (có `attendance_records`) | `DELETE` | `409 SESSION_HAS_ATTENDANCE`; không xoá; `PUT` sửa giờ / chủ đề vẫn được |
| TC-P5-01-20 | AC7 (buổi bù) | lớp có buổi 1…10 | `generate` một ngày (`weekdays=[thứ của ngày đó]`, `from=to`) rồi `PUT` `topic:"Học bù"` | `session_no`=max+1 (11); chủ đề "Học bù" |
| TC-P5-01-21 | AC7 (khoảng trống không lấp) | lớp có buổi 1…5; xoá buổi 3 (chưa điểm danh); tạo buổi mới | `generate` | buổi mới = 6 (không lấp 3) |
| TC-P5-01-22 | AC7 (xoá buổi cuối) | lớp có buổi 1…5; xoá buổi 5; tạo buổi mới | `generate` | buổi mới = 5 (cấp lại số vì buổi 5 chưa từng có điểm danh) |
| TC-P5-01-23 | AC8 (phân quyền) | – | `generate` (kể cả `dry_run`), `PUT`, `DELETE` bằng `$SVA`, `$ADM`; sinh viên ngoài lớp; `$TA_`, `$TCH`; `GET $Q/sessions` bằng `$SVA` | STUDENT/ADMIN/ngoài lớp → `403`; TA, TEACHER ✓; `GET sessions` Member ✓ |
| TC-P5-01-24 | AC8 (lớp khác, lưu trữ) | buổi của lớp 2; lớp `ARCHIVED` | `PUT $Q/sessions/<buổi lớp 2>`; mọi ghi trên lớp lưu trữ | `404`; `409 COURSE_ARCHIVED` |
| TC-P5-01-25 | AC9 (UI desktop) | `playwright-cli`, giảng viên 1280 | `/calendar`: nút `Tạo lịch buổi học`; mở ngăn kéo; chọn Thứ nhiều, Giờ, Phòng (placeholder "P.302"), Từ–Đến (`input type=date`), Ngày nghỉ; `Xem trước`; `Tạo 11 buổi` | nút chỉ Staff; bảng ≤ 60 dòng (Buổi, Ngày, Giờ) + "Sẽ tạo 11 buổi · bỏ 1 ngày nghỉ"; `Tạo 11 buổi` chỉ hoạt động sau xem trước; không câu mô tả dưới tiêu đề; danh sách buổi có menu `…` (`Sửa`, `Xoá`); lỗi 422 hiện tại ô lỗi |
| TC-P5-01-26 | AC9 (sinh khoá mới mỗi lần) | – | theo dõi `network`: `Xem trước` rồi `Tạo N buổi` | `Idempotency-Key` của lần tạo khác (không dùng lại) khoá nào của xem trước (xem trước không có key) |
| TC-P5-01-27 | AC9 (việc "Thiết lập lớp mới") | `$CN` chưa có buổi | `/me/today` bằng giảng viên → tạo buổi | việc "Tạo lịch buổi học" (`Href=/calendar?course=…`) tự tick khi có buổi |
| TC-P5-01-28 | AC9 (375 px) | `resize 375 812` | mở ngăn kéo | toàn màn hình; không tràn ngang; vùng chạm ≥ 44 px |
| TC-P5-01-29 | AC10 (Hôm nay) | buổi `S` hôm nay `starts_at ≤ now ≤ ends_at+24h`, `attendance_done_at IS NULL` (dùng buổi seed hôm nay hoặc `generate` ngày hôm nay) | `GET /me/today -H "$TCH"` | mục `SESSION_UNMARKED`: "Điểm danh buổi {no} đang diễn ra · {mã lớp}" (đang diễn ra) hoặc "Buổi {no} đã kết thúc, chưa điểm danh · {mã lớp}" (`Overdue=true`), `Href=/attendance?course=…&session=…`; ≤ 3 mục mỗi lớp mới nhất trước |
| TC-P5-01-30 | AC10 (vị trí bậc, biến) | cùng lúc có `EXAM_GRADE_ERROR` (nếu có) | quan sát thứ tự; sau `Lưu điểm danh` (US-P5-02) | `SESSION_UNMARKED` (bậc 12) đứng trước/sau đúng bảng bậc, không trùng bậc 15; biến ≤ 60 s sau khi lưu điểm danh |
| TC-P5-01-31 | AC10 (vai, lớp lưu trữ) | – | `GET /me/today` bằng `$SVA`, `$ADM`; lớp `ARCHIVED` | sinh viên/ADMIN không thấy; lớp lưu trữ không có |
| TC-P5-01-32 | AC11 (hai khoá, cùng tuần) | hai giảng viên (teacher và TA) cùng tạo cho cùng tuần | `&`+`wait` hai `generate` khác khoá | người sau `skipped DUPLICATE`; không `session_no` trùng; không `500` |
| TC-P5-01-33 | AC11 (mất mạng) | `playwright-cli` `setOffline` khi bấm `Tạo N buổi` | bấm; online; thử lại | nút trở lại; thử lại cùng `Idempotency-Key` không tạo đôi (`count` đúng) |
| TC-P5-01-34 | UI | frontend | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW calendar.spec.ts -g 'generate sessions preview and create\|edit delete session'`; `$PW today.spec.ts -g 'course setup sessions step ticks'` | `0`; xanh |

## Nhánh lỗi
TC-P5-01-09, 10, 13, 14, 16, 19, 24, 32, 33. Thêm TC-P5-01-E1: `dry_run` với lớp `ARCHIVED` → `409` hoặc `200` (SRS 6.1: `409 COURSE_ARCHIVED` — ghi thực tế); TC-P5-01-E2: Redis/DB chết khi `generate` → `503`, không buổi nửa vời.

## Phân quyền
TC-P5-01-23, 24, 31.

## Câu hỏi (Q-QC)
- **Q-QC-P5-01** Mã lỗi khi thiếu `Idempotency-Key` ở `generate` thật (TC-13) không nêu. Ghi mã thực tế; PM/BA xác nhận mã.
- **Q-QC-P5-02** AC10 yêu cầu `starts_at ≤ now ≤ ends_at + 24 giờ`; stack thật không có đồng hồ giả. Dùng buổi hôm nay của seed (09:00–11:30) và các buổi tạo thêm theo giờ hiện tại. Nếu cần kiểm "đã kết thúc, chưa điểm danh" ngoài giờ seed: tạo buổi có `ends_at` trong quá khứ vài phút (bằng `generate` + `PUT`). Xác nhận.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu (spec FEAT-crm-360 v1.1 APPROVED).
