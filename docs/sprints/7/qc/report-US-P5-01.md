# QC report — US-P5-01 (migration `00013_crm`, lịch buổi học: xem trước, sửa, huỷ, buổi bù, "buổi chưa điểm danh") — vòng 1
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `c9a2daa` (US-P5-01 vòng 2). CI của `c9a2daa`: run `38448247169` **xanh** (8 m 29 s). Bộ TC: `tc-US-P5-01.md` (không sửa chữ TC). Handoff: `dev-US-P5-01.md`. Stack `edupilot100` dựng lại từ code mới (DB trống, migration `13`, `--scale gateway=2`, provider `fake`). Lớp thử riêng: `900001` / `900002` / `900003` (tạo bằng `POST /admin/courses` + `assign`), không đụng seed.

## Kết luận: **PASS** (34 TC + 2 nhánh, không FAIL, không BLOCKED; PM đã nhận dev #20 b, c: giữ hợp đồng P2)
## 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 migration | PASS | `gateway migrate up` → `13`; `down` (về 0) → `up` lại sạch, `00013_crm.sql` có `applied_at`; `go test ./db/...` ok (vòng khứ hồi + version 13). Lệnh `down` của CLI lùi mọi bước (không có "một bước") nên không tự kiểm được riêng khối `Down` của `00013` ngoài `./db` |
| 02 một `ALTER` | PASS | `grep -c '^ALTER TABLE' 00013_crm.sql` = 1 (`class_sessions`); `PROGRESS.md` dòng 78 "00013 crm" |
| 03 `attendance_records`, `participation_events` | PASS | `status='X'` → `22P02` (enum `attendance_status`); trùng `(course,session,student)` → `23505`; buổi lớp khác / sinh viên không ghi danh → `23503` (FK phức hợp); `PENALTY` dương, `SPEAK` âm, `BONUS` âm, `points=6` → `23514`; `type='BAD'` → `22P02`; `client_event_id` trùng → `23505`; `version` có trong `attendance_records` |
| 04 `student_observations`, `student_risk`, view | PASS | `v_attendance_summary` và `v_student_360` có đúng danh sách cột ghim ở SRS 4.8.1 / 6; mọi chỉ mục phức hợp bắt đầu bằng `course_id` (chỉ `participation_events_client_event_key (client_event_id)` một cột, theo TC); `sqlc diff` rc=0 |
| 05 xem trước | PASS | `POST …/sessions/generate?dry_run=true` không `Idempotency-Key` → `200`, `would_create` 11 (buổi 1: `2026-10-12T09:00+07:00`, buổi 11: `2026-12-28`), `skipped:[{"date":"2026-11-02","reason":"EXCLUDED"}]` |
| 06 không ghi gì | PASS | `class_sessions`, `audit_log`, `outbox` của lớp: 0 / 2 / 2 trước và sau |
| 07 tạo thật = xem trước | PASS | `201 {created:11,first_session_no:1,last_session_no:11}`; số buổi và giờ khớp `would_create` (09:00 các thứ Hai trừ 02/11) |
| 08 `DUPLICATE` | PASS | dry_run lại: `would_create` 0, `skipped` = 11 `DUPLICATE` + 1 `EXCLUDED` |
| 09 tham số sai | PASS | `from>to` (`to`), `end<=start` (`end_time`), `weekdays` `[]` / `[0]` / `[8]` (`weekdays`), `room` 41 ký tự (`room`), `from:"2026-02-30"` (`from`) → `422` + `details[].field` |
| 10 > 60 buổi | PASS (theo #20 c) | `422 VALIDATION_FAILED`, `details:[{"field":"to","code":"TOO_MANY_SESSIONS","message":"Khoảng ngày này có 90 buổi, mỗi lần tạo tối đa 60…"}]`: nêu số buổi (90) và giới hạn 60. PM giữ hợp đồng P2 (mã ở `details[].code`) |
| 11 tạo hàng loạt | PASS | `201`; `audit_log` `class_sessions`/`generate` `{"first":1,"last":11,"created":11}`; 1 dòng outbox `course.changed` |
| 12 idempotent | PASS (theo #20 b) | cùng khoá: phản hồi giống hệt, 11 buổi; khác khoá cùng tham số → `created:0`, `skipped:11` (số, theo hợp đồng P2 đã giữ) |
| 13 không có khoá | PASS | `422 IDEMPOTENCY_KEY_REQUIRED`, không tạo (trả lời Q-QC-P5-01) |
| 14 lỗi giữa chừng | PASS | trigger tạm `RAISE` ở buổi thứ 5: `500`, `class_sessions` = 0, `audit_log` `generate` = 0; gỡ trigger, thử lại cùng khoá → `201 created 11` |
| 15 sửa | PASS | `PUT` → `200`, `version` 1 → 2, `session_no=3` giữ, `room`/`topic` ghi; `audit_log` `update` |
| 16 lỗi sửa | PASS | `version` cũ → `409 VERSION_CONFLICT` kèm bản hiện tại; `ends_at ≤ starts_at` → `422` (`ends_at`); trùng `starts_at` → `409 SESSION_DUPLICATE`; `session_no` trong thân → `422`, `session_no` giữ 3 |
| 17 ≤ 5 s | PASS | `PUT` giờ buổi → `GET /courses/{id}/calendar` đổi `ETag` và giờ ngay lập tức (0,07 s) |
| 18 xoá | PASS | `204`; `audit_log` `delete`; buổi mất |
| 19 đã điểm danh | PASS | (bản ghi `attendance_records` chèn bằng SQL — US-P5-02 chưa có) `GET` `attendance.has_records=true`; `DELETE` → `409 SESSION_HAS_ATTENDANCE`, buổi còn; `PUT` đổi chủ đề `200` |
| 20 buổi bù | PASS | lớp có 1…10 (buổi 11 vừa xoá): `generate` một ngày → `session_no=11` |
| 21 khoảng trống không lấp | PASS | còn 1,2,4,5; tạo → `6` |
| 22 xoá buổi cuối | PASS | xoá 6, tạo → `6` (cấp lại) |
| 23 phân quyền | PASS | `dry_run` / `PUT` / `DELETE`: STUDENT `403`, ADMIN `403`, TA `200`/`409` (qua guard), TEACHER ✓; TA `generate` thật `201`; không JWT `401`; sinh viên ngoài lớp (`$SVA` trên lớp khác) `403`; `GET sessions` Member `200` |
| 24 lớp khác, lưu trữ | PASS | `PUT` buổi lớp 2 qua URL lớp 1 → `404`; lớp `ARCHIVED` (qua `POST /admin/courses/{id}/archive`): `generate`, `PUT`, `DELETE` → `409 COURSE_ARCHIVED`; `GET` vẫn `200` |
| 25 UI desktop | PASS | `scripts/p5-01-ui.mjs`: nút "Tạo lịch buổi học" chỉ TEACHER / TA (STUDENT 0); ngăn kéo "Lịch buổi học": Thứ (T2…CN), Bắt đầu, Kết thúc, Phòng (placeholder `P.302`), Từ ngày, Đến ngày (`input type=date`), Ngày nghỉ + `Thêm`, `Xem trước`, `Tạo buổi` khoá tới khi xem trước; xem trước 11 dòng + "Sẽ tạo 11 buổi · bỏ 1 ngày nghỉ"; nút "Tạo 11 buổi"; danh sách buổi có menu `…` → `Sửa`, `Xoá`; `Xoá` → "Đã xoá buổi 1 · Hoàn tác", không hộp thoại; `?new=sessions` mở sẵn |
| 26 khoá mới mỗi lần | PASS (ghi chú) | khoá của lần tạo (`ep-78b70…`) khác khoá của xem trước (`ep-0016de86…`). Lưu ý: giao diện **có gửi** `Idempotency-Key` cả ở xem trước (TC viết "xem trước không có key"); máy chủ bỏ qua đúng AC4, nên không ảnh hưởng hành vi |
| 27 việc "Thiết lập lớp" | PASS | `/me/today`: bước `sessions` `done:false`, `href=/calendar?course=…&new=sessions`; sau `generate` → `done:true` ≤ 4 s |
| 28 375 px | PASS | ngăn kéo 375×812; không tràn ngang; 0 phần tử tương tác < 44 px |
| 29 `SESSION_UNMARKED` | PASS | buổi hôm nay đang diễn ra: "Điểm danh buổi 8 đang diễn ra · 900002", `urgency:high`, `href=/attendance?course=…&session=…`; sau `PUT` `ends_at` quá khứ: "Buổi 8 đã kết thúc, chưa điểm danh · 900002", `urgency:overdue`. 5 buổi chưa điểm danh trong 24 h → đúng 3 mục (buổi 8, 11, 12 — ba buổi mới nhất; thứ tự hiển thị 11, 12, 8) |
| 30 vị trí bậc, biến | PASS (một phần) | `SESSION_UNMARKED` đứng trước `EMAIL_MISMATCH`, `JOIN_REQUEST`, `EXAM_SIMILARITY`, `COURSE_SETUP`; không có `EXAM_GRADE_ERROR` để đối chiếu bậc 15. Biến mất ≤ 1 s sau khi `attendance_done_at` được ghi (giả bằng SQL: US-P5-02 chưa có `Lưu điểm danh`) |
| 31 vai, lớp lưu trữ | PASS | STUDENT, ADMIN: 0 mục; lớp `ARCHIVED`: 0 mục |
| 32 hai khoá cùng tuần | PASS | teacher và TA song song: một `created:4` (buổi 4…7), một `created:0 skipped:4`; không `session_no` trùng, không `500` |
| 33 mất mạng | PASS | `setOffline` khi bấm "Tạo 11 buổi": thông báo "Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ."; nút bật lại; online → thử lại cùng khoá (4 lần gửi, 1 khoá), "Đã tạo 11 buổi" |
| 34 giao diện | PASS | `ui-antipatterns.sh` rc=0; `playwright calendar.spec.ts today.spec.ts` (desktop + mobile): 40 passed, 18 skipped, 0 failed |
| E1 `dry_run` lớp lưu trữ | PASS | `409 COURSE_ARCHIVED` (SRS 6.1) |
| E2 DB / Redis chết | PASS | PgBouncer dừng → `503 SERVICE_UNAVAILABLE`, 0 buổi mới; Redis dừng → `503`, 0 buổi mới |

Tổng: 34 TC + 2 nhánh = 36: PASS 36 (TC-30 chỉ kiểm một phần: không có `EXAM_GRADE_ERROR` để đối chiếu bậc 15, và `Lưu điểm danh` của US-P5-02 chưa có).

## 2. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -race ./internal/store -run 'TestSchemaAttendance|…|TestMigration00013SingleAlter'` | 10 passed |
| `go test ./db/...` | ok |
| `go test -race ./internal/course -run 'TestGenerate|TestSession|TestMakeup'` | 28 passed |
| `go test -race ./internal/today -run 'TestSessionUnmarked|TestCourseSetupSessionsStepHref|TestTodayQueryBudget'` | 7 passed |
| `go test -race -tags testroutes ./internal/contract/... ./internal/httpapi/...` | 154 passed / 18 gói |
| `sqlc diff` | rc=0 |
| `playwright calendar.spec.ts today.spec.ts`; `ui-antipatterns.sh` | 40 passed / 0 failed; rc=0 |

## 3. Ghi nhận (không làm FAIL)
- `GET …/sessions` bằng STUDENT vẫn trả `version` (handoff nói `version` và `attendance` chỉ cho nhân viên; `attendance` đúng là ẩn, `version` còn).
- Ngăn kéo: dòng "Lớp chưa có buổi học." khi rỗng; mọi nút đều là động từ.
- Trigger tạm, bản ghi `attendance_records` và 3 lớp thử (`900001`…`900003`) chỉ tồn tại trong stack kiểm của QC.
- `proposals.md` #21 (`v_student_360` vào danh sách loại trừ của `TestRawPIINeverStored`) dev đã sửa trước khi PM quyết: QC không chấm.
