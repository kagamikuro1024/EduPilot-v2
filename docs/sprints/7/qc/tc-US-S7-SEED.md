# QC test case — US-S7-SEED (dữ liệu seed sprint 7, lệnh quản trị, cổng)
Nguồn: `US.md` US-S7-SEED AC1…AC10 + `SRS.md` 4.10; `FEAT-escalation-notify` SRS 9.4; `proposals.md` #7 (ba lệnh `gateway admin seed-study-time`, `seed-age`, `crm-risk-run`, chỉ `APP_ENV ∈ {local, test}`); `ARCHITECTURE.md` §9; D44 (dữ liệu mô phỏng hoàn toàn). Hộp đen, không đọc code dev.
Biến như `tc-US-P5-01.md`. `seed/expected_crm.csv` (A, B, C: `recorded, present, late, excused, absent, speak_count, points_total`). Chạy seed: `node scripts/seed.mjs` (cùng `SEED_ON_EMPTY_DB`). DB trống = `pnpm dev:down -v && pnpm dev` hoặc `dropdb` rồi `goose up`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-S7-SEED-01 | AC1 (điểm danh) | DB trống | `node scripts/seed.mjs`; `$PSQL -c "select count(*) from attendance_records where course_id='$C1'"`; `select session_no, attendance_done_at is not null from class_sessions where course_id='$C1' order by session_no` | `270` dòng; buổi 1–9 `attendance_done_at` có giá trị; **buổi 10 (hôm nay 09:00–11:30) chưa điểm danh** |
| TC-S7-SEED-02 | AC1 (A, B, C) | – | `select … from v_attendance_summary` cho `sv.gioi`, `sv.kha`, `sv.nguyco`; so `seed/expected_crm.csv` | A vắng 0; B vắng **2**, muộn 1; C vắng **5**, muộn 1; khớp từng cột csv |
| TC-S7-SEED-03 | AC1 (tổng, phát biểu) | – | `select count(*) from attendance_records where status='ABSENT' and course_id='$C1'`; `select count(*), sum(points) from participation_events where course_id='$C1'`; B | tổng ABSENT trong 18–26; `participation_events` ≈ 60, lệch (A nhiều nhất); **B đúng 2 sự kiện `SPEAK` = +0,50** |
| TC-S7-SEED-04 | AC1 (tất định) | – | seed trên DB trống lần hai (cùng `SEED_RNG`); so số liệu | giống hệt lần một |
| TC-S7-SEED-05 | AC1 (test của dev) | repo | `cd backend-go && go test -tags integration ./internal/integration -run TestSeedCRMExpected -v` | `ok` |
| TC-S7-SEED-06 | AC2 (thời gian học 10 tuần) | sau TC-01 | `gateway admin seed-study-time` (cú pháp: xem `Q-QC-S7-01`); `select count(distinct date), count(*) from study_time_daily where course_id='$C1'` | in "N dòng"; `count(distinct date) ≥ 60`; ≈ 70 ngày × 30 SV × 1–3 khu vực |
| TC-S7-SEED-07 | AC2 (số mẫu) | – | phút học theo tuần của C, A, B, lớp 2 | C tuần 7–9 dưới 60 phút/tuần; A ≈ 300; B ≈ 150; lớp 2 có 3 tuần |
| TC-S7-SEED-08 | AC2 (chặn môi trường) | – | `APP_ENV=production gateway admin seed-study-time …; echo $?`; `APP_ENV=staging`; `APP_ENV=` (rỗng); `APP_ENV=local` | production/staging/rỗng → thoát **khác 0**, nêu lý do; local → `0`, in "N dòng" |
| TC-S7-SEED-09 | AC2 (idempotent) | – | chạy `seed-study-time` lần hai; so `count(*)` và tổng `active_seconds` | không đổi (không nhân đôi) |
| TC-S7-SEED-10 | AC3 (at-risk C) | seed + `gateway admin crm-risk-run` | `j "$Q/students?filter=attention" -H "$TCH"`; `select level, count(*) from student_risk where course_id='$C1' group by 1` | đúng **1** `AT_RISK` (C) với lý do chuyên cần + thời gian học (câu khớp SRS 4.7.5); đúng **2** `WATCH`; A, B `NONE`; lớp 2 không `AT_RISK` |
| TC-S7-SEED-11 | AC3 (Hôm nay) | – | `GET /me/today -H "$TCH"`; `playwright-cli` `/` | mục `STUDENT_ATTENTION` "1 sinh viên mới vào diện cần chú ý · 761987"; khối `attention` có Sinh viên C |
| TC-S7-SEED-12 | AC4 (ticket đủ trạng thái) | seed (luồng thật) + `gateway admin seed-age` | `j "$Q/tickets?status=ALL" -H "$TCH"`; `curl -s $MP/api/v1/messages \| jq '.messages\|length'`; `select count(*) from escalation_tickets where status='OPEN' and waiting_since < now() - interval '80 hours'` | 5 phần tử, đủ 4 trạng thái (OPEN mới, OPEN quá hạn, CLAIMED bởi teacher, ANSWERED, CLOSED `STUDENT_CONFIRMED`); Mailpit ≥ 1 thư; ≥ 1 ticket `waiting_since` quá 80 giờ; lớp 2 không ticket; **không** ticket của B |
| TC-S7-SEED-13 | AC4 (`seed-age` chỉ đổi thời gian) | – | chụp `escalation_tickets` (bỏ cột thời gian) và `chat_messages.content` trước/sau `seed-age`; chạy `APP_ENV=production gateway admin seed-age; echo $?` | nội dung không đổi (chỉ `created_at`, `answered_at`, `waiting_since` của ticket; `created_at` của tin chat/thread seed); in số dòng đã đổi; production → thoát khác 0 |
| TC-S7-SEED-14 | AC4 (quá hạn thật sự) | OPEN quá hạn | chờ job SLA (hoặc `gateway admin run-job ticket.sla`, cú pháp xem `tc-US-P4-05`) | `sla_stage=2`, mục `TICKET` `Overdue=true` (chứng minh `seed-age` lùi cả `waiting_since`, không chỉ `created_at`) |
| TC-S7-SEED-15 | AC5 (Threads) | seed | `j "$Q/moderation/queue" -H "$TCH"`; `select count(*) from post_reports where course_id='$C1'`; ghim; RAG | ≥ 2 phần tử trong hàng chờ; `post_reports` ≥ 5; có: 1 thread 2 báo cáo `OPEN`, 1 thread tự ẩn (3 người khác nhau), 1 bình luận đã `Ẩn bài` có lý do, 1 thread ghim, 1 bài AI `VERIFIED` đã nạp tri thức (hỏi câu tương tự → có nguồn "Hỏi đáp đã được …"); `Hôm nay` có `POST_REPORTED` |
| TC-S7-SEED-16 | AC6 (idempotent) | DB đã seed | chạy `node scripts/seed.mjs` lần hai; đếm: `attendance_records` (270), `participation_events`, ticket (5), `post_reports`, `knowledge_items`, thread | số **không đổi** |
| TC-S7-SEED-17 | AC6 (tự chạy, production) | – | `pnpm dev` trên DB trống; `NODE_ENV=production node scripts/seed.mjs; echo $?`; `grep -n SUPPORT_RESOURCES_VI docker-compose.local.yml` | `pnpm dev` seed đủ dữ liệu của story; production → thoát khác 0; có `SUPPORT_RESOURCES_VI` mô phỏng |
| TC-S7-SEED-18 | AC7 (cổng) | repo | `bash scripts/gate-p4.sh; echo $?`; `bash scripts/gate-p5.sh; echo $?` | `0`; mỗi lệnh in một dòng `PASS`/`FAIL`; `gate-p4.sh` chạy đúng lệnh `P4.md` (+ `-race`); `gate-p5.sh` chạy lệnh `P5.md` (`./internal/crm/... ./internal/activity/...`, `TestToolMyAttendance\|TestToolMyParticipation`, `attendance.spec.ts`); khi **một** lệnh đỏ → script thoát khác 0 (QC cố ý làm đỏ ở nhánh tạm, xoá sau) |
| TC-S7-SEED-19 | AC7 (E2E) | – | `$PW escalation.spec.ts attendance.spec.ts` | xanh; `escalation.spec.ts` phủ F5 (đường chính + hai người cùng `Nhận` + mail lỗi) và F4 kiểm duyệt; `attendance.spec.ts` phủ F7 (đường chính + **bước ngắt mạng** + hai người cùng điểm danh) ở 375 px, dùng tài khoản seed. Đọc tên test để đối chiếu (`--list`) |
| TC-S7-SEED-20 | AC8 (không rò) | seed | `$SVA`, `$SVB`, `$SVC`: chỉ `me/*` của mình (`GET $Q/me/attendance` mỗi người; thử `…/students/{uid}/…` → `403`); `sv.moi` (D, chưa vào lớp): `GET /me/today`, `GET $Q/me/attendance` | A/B/C thấy đúng của mình; D không thấy dữ liệu lớp nào (`403`); TA thấy lớp 1 và **không** thấy lớp 2 (`403`/`404`); ADMIN `403` ở ticket/hồ sơ/điểm danh; canary (ghi chú, ticket) không ở phản hồi nào dành cho sinh viên (crawl như TC-P5-06-06) |
| TC-S7-SEED-21 | AC8 (test của dev) | repo | `cd backend-go && go test -tags integration ./internal/integration -run 'TestSeedIsolation\|TestSeedAdminForbidden' -v` | `ok` |
| TC-S7-SEED-22 | AC9 (seed gãy giữa chừng) | DB trống | chạy `node scripts/seed.mjs`, `kill -9` tiến trình khi đang bước điểm danh (theo log); chạy lại; `select count(*) from attendance_records where course_id='$C1'` | `270`; tiếp tục từ chỗ dở; không tạo đôi, không sót; kết thúc cùng số liệu như chạy một lần (đối chiếu TC-01…03) |
| TC-S7-SEED-23 | AC9 (lỗi API) | chặn gateway giữa seed (`docker compose stop gateway` khi đang chạy) | quan sát | in bước, mã lỗi, **dừng với mã khác 0** (không nuốt lỗi) |
| TC-S7-SEED-24 | AC10 (cờ đo của hai lớp) | seed đã chạy | `$PSQL -c "select class_code, allow_study_time_opt_out from courses order by class_code"`; `select count(*) from user_settings where track_study_time=false`; `j -X PUT "$Q/settings" -H "$TCH" -d '{"allow_study_time_opt_out":false,"version":1}'`; cùng lệnh bằng `$TA_`, `$SVA`, `$ADM` | `761987 \| t`, `761988 \| f`; `0`; `409 COURSE_HAS_STUDENTS`; `403` ×3 |
| TC-S7-SEED-25 | AC10 (A hai dòng; chạy lại) | – | `/settings` của `sv.gioi`; seed chạy lại | hai dòng (lớp 1 đổi được, lớp 2 bật và khoá); seed chạy lại không đổi hai giá trị, không lỗi (409 được coi là đã áp dụng) |

## Nhánh lỗi
TC-S7-SEED-08, 13, 17, 22, 23.

## Phân quyền
TC-S7-SEED-20, 24.

## Câu hỏi (Q-QC)
- **Q-QC-S7-01** Cú pháp ba lệnh `gateway admin seed-study-time`, `seed-age`, `crm-risk-run` chưa ghi ở US/SRS (chỉ tên). Cần ví dụ chạy trong handoff (container nào, tham số). Nếu thiếu → "KHÔNG KIỂM ĐƯỢC" = FAIL ở pha 2.
- **Q-QC-S7-02** TC-18: `scripts/gate-p4.sh`, `gate-p5.sh` do dev tạo; QC làm đỏ một lệnh ở nhánh tạm để kiểm thoát khác 0. Xác nhận cho phép sửa script ở nhánh tạm (không commit).
- **Q-QC-S7-03** `seed/expected_crm.csv` có trong repo chưa? (spec nhắc). Nếu thiếu khi chấm → TC-02 "KHÔNG KIỂM ĐƯỢC".

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
