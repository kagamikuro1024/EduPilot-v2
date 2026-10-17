# QC report — US-P5-04 (`/students`, at-risk theo tuần, ngưỡng theo lớp, "Lớp cần chú ý")
Ngày 2026-10-15. Commit kiểm: `sprint/7-p4-p5` @ `aff08c2c` (code dev `7e7ae643`; CI xanh `b23b647`). Bộ TC: `tc-US-P5-04.md` (không sửa chữ TC). Handoff: `dev-US-P5-04.md` (#28 #30 áp dụng). Stack `edupilot100` dựng lại từ mã mới.

## Kết luận: **PASS** (35 TC + E1 + P1; 0 lỗi; TC-08/09/10 và E1 chỉ kiểm gián tiếp — PM xem mục 3)
Dữ liệu thử: **không có** `gateway admin crm-risk-run` / `seed-study-time` (thuộc US-S7-SEED, chưa làm; `dev-QC-answers.md`). QC dựng lớp `QC risk` (mã `108503`, giảng viên `teacher@`) bằng API + SQL: 12 sinh viên ACTIVE (+ 1 PENDING, 1 REMOVED), 8 buổi ở hai tuần trọn gần nhất (T2 28/9 và 5/10, 4 buổi/tuần), vắng 3/4 mỗi tuần cho `sv04`–`sv11`, `sv12` vắng chỉ tuần gần nhất, `sv13` đủ, `sv14` đủ nhưng học ít, `sv15` vắng có phép. Chạy job bằng cách đổi ngưỡng (`PUT /courses/{id}/settings {risk}` → outbox `course.settings_changed` → tính lại lớp đó), không dùng lệnh `crm-risk-run`.

## 1. Kết quả TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `{items,next_cursor,total}`; mỗi dòng đúng khoá `student{id,full_name,student_code}`, `attendance`, `participation{speak_count,points_total}`, `study{minutes_7d,tracked}`, `activity{asks_last_week,posts_last_week}`, `risk{level,computed_at}`; không cột điểm, không câu lý do; 12 dòng = 12 ACTIVE |
| 02 | PASS | đi hết bằng `limit=5`: 3 trang, 12 id không trùng; thứ tự `AT_RISK`×8, `WATCH`, `NONE`×3; `limit=500` → `422`; `cursor=xxx` → `422 INVALID_CURSOR` |
| 03 | PASS | PENDING và REMOVED không có trong danh sách / `total` |
| 04 | PASS | `log_statement='all'`: gọi danh sách, `filter=attention`, hồ sơ → 0 lần `chat_messages` |
| 05 | PASS | `attention`=8, `absent`=9 (8 + sv12 vắng 3), `inactive`=12; giao hai bộ lọc ở **cả hai** dạng `filter=a&filter=b` và `filter=a,b` (Q-QC-P5-08: cả hai chạy); `filter=xyz` → `422` |
| 06 | PASS | `q=NgUyEn` → "Nguyễn Anh Linh"; `q=2022` (MSSV tiền tố); `q=zzzzqq` → `[]`, `total 0` |
| 07 | PASS | `422` |
| 08 / 09 / 10 | **PASS gián tiếp** | QC không gieo `seed-perf.sql` (đĩa colima 91%, hộp đen không có câu SQL — Q-QC-P5-07 chưa có hồi đáp). Chạy lại bài của dev trong `go test -race ./internal/crm/...`: `TestStudents360ExplainNoSeqScan` (EXPLAIN đúng câu SQL tracer ghi, lớp 100), `TestStudentsListP95Under300ms` (lớp 1.000), `TestStudents360LargeCoursePerf`, `TestProfileUnder3s` — xanh; không `SET enable_seqscan` trong mã chạy |
| 11 | PASS | `TestRiskTable` 16 ca (`--- PASS` ×16); kiểm tay ca 10/14/15 qua dữ liệu lớp thử: `sv15` (vắng có phép) `NONE`; `sv14` (chỉ học ít, không neo) `NONE`; `STUDY` chưa ack trước đầu tuần → `null` (không bị coi là thấp) |
| 12 | PASS | `sv14` (chỉ thời gian học thấp 2 tuần) `NONE`; tuần không có dữ liệu chuyên cần (`weeks[2..3]`) `weak:false` |
| 13 | PASS | hai lần tính lại: `entered_at` giữ `15:18:16`, `computed_at` đổi; `signals_required=3 + activity=0` → `NONE`, `entered_at` rỗng; trả ngưỡng → `AT_RISK`, `entered_at` = `now()` mới |
| 14 | PASS | `student_risk` của PENDING / REMOVED: 0 dòng |
| 15 | PASS (một phần, Q-QC-P5-09) | `ep:crm.risk:done:20261015` có, `TTL` 172.563 s ≈ 48 h; chạy bù hai bản worker: `TestRiskJobRunsOncePerDay`, `…CatchUpAfterWorkerDown`, `…LeaderOnce` xanh |
| 16 | PASS | `weeks` JSON 4 tuần T2–CN (`2026-10-05`, `09-28`, `09-21`, `09-14`), mỗi tuần `attendance{ok,ratio,recorded}`, `activity{count}`, `study`, `low[]`, `weak` |
| 17 | PASS | hồ sơ AT_RISK: "Cần chú ý: vắng 6/8 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 20 phút mỗi tuần; không hỏi AI hay đăng bài ở Threads trong 2 tuần." — thứ tự chuyên cần, thời gian học, hoạt động; số khớp danh sách ("Vắng 6/8"); `reasons[{code,text}]`; không tên người khác / điểm / chat |
| 18 | PASS | WATCH mở đầu "Đang theo dõi:"; NONE `sentence:null`, `reasons:[]` |
| 19 | PASS | `PUT {risk:{…},version}` → `200`; `audit_log` `course_risk_updated` cũ/mới; danh sách đổi trong ≤ 40 s. Lưu ý: `attendance_ok_min` là **chuỗi** ("0.8") ở cả `GET` và `PUT`; số `0.8` bị `422 type` (TC viết số) |
| 20 | PASS | 9 ca (`0,45`, `1,05`, `-1`, `21`, `601`, `signals 0`, `4`, `absent 0`, `21`) → `422` đúng trường `risk.<tên>` |
| 21 | PASS | TA / STUDENT / ADMIN `403`×3; sai version `409`; mặc định `0.70 / 1 / 60 / 2 / 3` |
| 22 | PASS | nhóm "Ngưỡng cần chú ý": năm dòng nhãn + giá trị (Tỷ lệ có mặt tối thiểu 0,70 · Hoạt động tối thiểu mỗi tuần 1 · Phút học tối thiểu mỗi tuần 60 · Số tín hiệu thấp để tính 2 · Vắng nhiều từ 3 buổi); không câu mô tả |
| 23 | PASS | mục `STUDENT_ATTENTION` "8 sinh viên mới vào diện cần chú ý · 108503", `href=/students?filter=attention&course=…`; khối `attention` 5 dòng ("vắng 6/8 buổi; ít hoạt động"), `attention_total` 8; biến mất ≤ 50 s khi ngưỡng đưa mọi người về `NONE`, hiện lại sau khi trả ngưỡng. Lưu ý môi trường: `actions` cắt ở 50 mục; 43 mục `COURSE_SETUP` từ lớp thử đẩy `STUDENT_ATTENTION` ra ngoài tới khi tôi lưu trữ lớp thử và xoá `ep:today:*` |
| 24 | PASS | tất cả `NONE` → `attention:[]`, `attention_total 0`, không mục; STUDENT, ADMIN: không có khoá `attention` |
| 25 | PASS | 8 `AT_RISK` → 5 dòng + "Xem tất cả (8)" (`attention_total`) |
| 26 | PASS | 1280 px: ô tìm + chip `Cần chú ý` · `Vắng nhiều` · `Ít hoạt động` (không `Điểm giảm`); cột Sinh viên · Chuyên cần · Phát biểu · Hoạt động học · Rủi ro; dòng "Vắng 6/8 · 0 · 0,00 · 0 phút · 0 câu hỏi · Cần chú ý"; bấm dòng → `/students/{id}` |
| 27 | PASS | tìm + chip vào URL (`?q=nguyen&filter=attention`); quay lại từ hồ sơ giữ chip (`aria-pressed=true`); `?filter=attention` bật đúng chip |
| 28 | PASS | `500` → "Không tải được danh sách sinh viên." + `Thử lại`; không khớp → "0 sinh viên" + `Bỏ lọc`; 720 và 600 px không tràn trang; ảo hoá + 1.000 dòng: e2e `students virtual cursor` |
| 29 | PASS | STUDENT (kể cả uid mình) và ADMIN → `403` cho danh sách và hồ sơ |
| 30 | PASS | crawl `sv04` (AT_RISK, có ghi chú canary): `me/today`, `me/settings`, `notifications`, `me/courses`, `…/me/study-time`, `…/threads`: không khoá `risk*`/`at_risk`/`observation*`/`watch`, 0 lần `CANARY-OBS`/"rủi ro"/"cần chú ý" |
| 31 | PASS | chat "Em có bị coi là cần chú ý không?" / "Giảng viên ghi gì về em?" → "Mình chưa tìm thấy nội dung này trong tài liệu của lớp."; payload LLM (bộ đệm `_test/llm/payloads`) 0 `AT_RISK` / canary |
| 32 | PASS | `TestSensitiveQueriesOnlyFromCRM`, `TestStudentNeverSeesRisk` (trong `./internal/crm`), `TestAgentHasNoRiskOrObservationAccess` xanh |
| 33 | PASS | lớp mới + 1 sinh viên: `total 1`, mọi số 0, `risk NONE` / `computed_at null`, không lỗi chia 0 |
| 34 | PASS | job chưa chạy: `risk {level NONE, computed_at null}`; lớp `ARCHIVED` `GET …/students` → `200` |
| 35 | PASS | `ui-antipatterns.sh` rc=0; Playwright desktop (`students.spec`, `a11y.spec`, `class-join.spec`, `today.spec`): 163 passed, 7 skipped (375 px / `@real`, project desktop), 0 fail |
| E1 | PASS gián tiếp | `TestRiskJobFailureRetries` + `TestRiskJobDoesNotTouchRemoved` xanh; không tắt worker giữa lô (lô một lớp nhỏ < 1 s) |
| P1 | PASS | TA lớp 1 gọi `…/students` lớp thử → `403` |

## 2. Lệnh của handoff
`go test -race ./internal/crm -run 'TestRisk|TestSettingsRisk|TestStudents|TestStudent|TestSensitive'` ok (63 s); `./internal/today -run Attention` 4 passed; `./internal/agent -run TestAgentHasNoRiskOrObservationAccess` ok; `./internal/contract/... ./internal/course ./internal/store ./db/...` ok (cả gói rộng 1.010 test / 28 gói, không đỏ); `sqlc generate && sqlc diff` rc=0; `pnpm build:gate` + playwright như trên; `ui-antipatterns.sh` rc=0; `pnpm lint` sạch.

## 3. Ghi nhận / cần PM
- TC-08/09/10 và E1 gián tiếp: nếu PM muốn QC tự gieo `seed-perf.sql` (1.000 + 19×100 SV, ≥ 280.000 dòng thời gian học) thì cần thêm đĩa cho colima (volume `edupilot` Postgres đã 91%).
- `gateway admin crm-risk-run`, `seed-study-time`, `seed-age` chưa có (US-S7-SEED); TC viết theo lệnh đó (TC-04…15 mô tả "sau `crm-risk-run`") chạy được bằng cách thay lệnh bằng đổi ngưỡng.
- `/students` có dòng phụ dưới tiêu đề "Chuyên cần và nhãn rủi ro chỉ giảng viên/TA thấy" — trái luật cứng của PO về câu mô tả dưới tiêu đề (`~/.claude/CLAUDE.md`), không có trong TC; xin PM xác nhận cố ý (spec) hay xoá.
- `attendance_ok_min` kiểu chuỗi trong API (như `escalation_threshold`); số → `422 type`.
