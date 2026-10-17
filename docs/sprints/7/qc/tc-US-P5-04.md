# QC test case — US-P5-04 (`/students`, at-risk, "Lớp cần chú ý")
Nguồn: `US.md` US-P5-04 AC1…AC10 + `SRS.md` 2, 4.7, 4.9, 5.4–5.6, 6.4, 7.3, 7.7, 9.3, 9.4; `DESIGN.md` §14.1, §14.6; `SYSTEM_DESIGN.md` §5 (SLO). Hộp đen, không đọc code dev.
Biến như `tc-US-P5-01.md`. Seed (US-S7-SEED): lớp 1 có 30 sinh viên; A (`sv.gioi`) vắng 0; B (`sv.kha`) vắng 2, muộn 1; C (`sv.nguyco`) vắng 5, muộn 1, phút học tuần 7–9 < 60; sau `gateway admin crm-risk-run`: C `AT_RISK`, 2 sinh viên khác `WATCH`, A, B `NONE`. `uidC` = id của C. Chạy job at-risk: `gateway admin crm-risk-run` (proposals #7).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P5-04-01 | AC1 | seed đã chạy; `crm-risk-run` | `j "$Q/students?limit=50" -H "$TCH"` | `{items:[{student:{id,full_name,student_code}, attendance:{recorded,present,late,excused,absent}, participation:{speak_count,points_total}, study:{minutes_7d,tracked}, activity:{asks_last_week,posts_last_week}, risk:{level}}], next_cursor, total}`; chỉ sinh viên `ACTIVE`; không có cột điểm; `risk.level` ∈ {NONE,WATCH,AT_RISK}; **không** có câu lý do ở danh sách; trường `activity` không tên `_7d` |
| TC-P5-04-02 | AC1 (sắp xếp, phân trang) | lớp 1.000 SV (xem TC-09) hoặc 30 SV với `limit=7` | đi hết `next_cursor`; so thứ tự: `risk_rank` giảm, tên không dấu, id | không trùng, không sót (đối chiếu `select count(*)`); `limit` mặc định 50, tối đa 100 (`limit=500` bị chặn); `cursor` sai → `INVALID_CURSOR` (`422/400`, ghi) |
| TC-P5-04-03 | AC1 (chỉ ACTIVE) | 1 `PENDING`, 1 `REMOVED` | liệt kê | không có trong danh sách và `total` |
| TC-P5-04-04 | AC1 (không đọc chat) | – | bật `log_statement='all'`, gọi danh sách; `grep -c chat_messages` | 0 truy vấn chạm `chat_messages` |
| TC-P5-04-05 | AC2 (lọc) | seed | `filter=attention`; `filter=absent`; `filter=inactive`; `filter=attention&filter=absent` (hoặc `filter=attention,absent` — theo spec "hai lọc cùng lúc = giao"; thử cả hai dạng, ghi dạng đúng) | `attention` → chỉ `AT_RISK` (C); `absent` → sinh viên có ABSENT ≥ `absent_alert` (mặc định 3; gồm C); `inactive` → `asks+posts=0` và `minutes_7d < study_min_minutes`; hai lọc = giao |
| TC-P5-04-06 | AC2 (tìm) | – | `q=nguyen` (không dấu, hoa/thường lẫn lộn `NgUyEn`); `q=<MSSV 4 ký tự đầu>`; `q=` không khớp | khớp tên không dấu, không phân biệt hoa thường, hoặc MSSV tiền tố; không khớp → `items:[]` |
| TC-P5-04-07 | AC2 (filter lạ) | – | `filter=xyz` | `422` |
| TC-P5-04-08 | AC3a (EXPLAIN lớp 100) | gieo: **1 lớp 1.000 SV + 19 lớp 100 SV**, ≥ 15 buổi (15.000 + 28.500 `attendance_records`), ≥ 280.000 `study_time_daily`, ≥ 5.000 `participation_events` bằng `generate_series` (script `docs/sprints/7/qc/scripts/seed-perf.sql`), `ANALYZE` | lấy câu SQL của trang đầu, trang giữa, `filter=attention`, `q=`, hồ sơ một SV cho **lớp 100 SV** (từ `log_statement` hoặc `auto_explain` khi gọi API); `EXPLAIN (FORMAT JSON)` | không `Seq Scan` trên `attendance_records`, `study_time_daily`, `participation_events`, `student_risk`, `student_observations`; mọi truy cập bằng chỉ mục bắt đầu `course_id` |
| TC-P5-04-09 | AC3b (lớp 1.000 SV) | dữ liệu TC-08 | gọi `students?limit=50` 30 lần, đo (`curl -w '%{time_total}'`); đo hồ sơ 360 | trang đầu p95 ≤ 300 ms; hồ sơ ≤ 3 s |
| TC-P5-04-10 | AC3b (hồ sơ dùng chỉ mục) | – | `EXPLAIN` truy vấn hồ sơ **một** SV lớp 1.000 | `Index Scan` theo `(course_id, student_id…)`; không `SET enable_seqscan=off` (QC grep log `SET enable_seqscan` = 0, và `docker compose` không đặt biến) |
| TC-P5-04-11 | AC4 (at-risk — bảng ca) | – | chạy `cd backend-go && go test ./internal/crm -run 'TestRiskTable' -v` và đếm 16 ca ở SRS 9.3 (1: 0,5/0/30 hai tuần → AT_RISK; 2: W2 không có neo thấp → WATCH; 3: chỉ 1 tín hiệu → NONE; 4: neo + hoạt động → AT_RISK; 5: không neo → NONE; 6 → WATCH; 7: tuần nghỉ → NONE; 8: chưa ack → AT_RISK; 9 → NONE; 10: đúng ngưỡng → NONE; 11: `signals_required=1` → AT_RISK; 12: `=3` → NONE; 13: vào lớp giữa kỳ → WATCH; 14: EXCUSED → NONE; 15: `STUDY=null` → AT_RISK; 16 → NONE) | `ok`; đủ 16 ca; QC cũng kiểm tay ca 10, 14, 15 qua dữ liệu seed riêng ở lớp thử (đặt dữ liệu bằng API + `seed-study-time` rồi `crm-risk-run`) |
| TC-P5-04-12 | AC4 (neo bắt buộc; thiếu dữ liệu) | lớp thử | sinh viên chỉ thấp thời gian học (không chuyên cần) 2 tuần; sinh viên có tuần giữa không đủ dữ liệu | cả hai `NONE`/không `AT_RISK` |
| TC-P5-04-13 | AC4 (idempotent, `entered_at` giữ) | C `AT_RISK` | `crm-risk-run` hai lần liên tiếp (hai "đêm"); `select entered_at, computed_at from student_risk where student_id=<C>` | cùng kết quả; `entered_at` **không đổi** (chỉ `computed_at` đổi); rời `AT_RISK` → `entered_at` xoá; vào lại → `now()` |
| TC-P5-04-14 | AC4 (REMOVED/PENDING) | – | `select count(*) from student_risk where student_id in (<removed>,<pending>)` | 0 |
| TC-P5-04-15 | AC4 (đúng một lần mỗi ngày, chạy bù) | worker | `$RDS GET ep:crm.risk:done:<yyyymmdd>`; chạy `crm-risk-run`; kiểm `$RDS TTL`; dừng worker qua 02:00 (đặt đồng hồ máy chạy thử không khả thi → dùng `$RDS DEL ep:crm.risk:done:<hôm nay>` rồi bật worker sau 02:00) | khoá có TTL ≈ 48 h; job chạy bù khi worker lên lại, **một** lần (hai bản worker → một lần); lỗi giữa chừng → `DEL` khoá, tick sau thử lại |
| TC-P5-04-16 | AC4 (tuần/ranh giới, đếm theo `created_at`) | dữ liệu hoạt động | kiểm `student_risk.weeks` JSON của C | mỗi tuần T2–CN trọn; hoạt động đếm theo `created_at` (không theo khoảng `id`) |
| TC-P5-04-17 | AC5 (lý do bằng lời) | C `AT_RISK`; seed | `j "$Q/students/$uidC/profile" -H "$TCH" \| jq '.risk'` | `risk:{level:"AT_RISK",sentence,reasons:[{code,text}]}`; câu dạng "Cần chú ý: vắng 3/4 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 20 phút mỗi tuần." (chuyên cần + thời gian học); số khớp danh sách và tab; không nhắc tên khác, không nêu điểm AI hay nội dung chat |
| TC-P5-04-18 | AC5 (WATCH, NONE) | một sinh viên `WATCH`, A `NONE` | hồ sơ | `WATCH` mở đầu "Đang theo dõi:"; `NONE` không có `sentence` |
| TC-P5-04-19 | AC6 (ngưỡng) | `/class/settings` bằng teacher | `PUT $Q/settings {"risk":{"attendance_ok_min":0.8,"activity_min_per_week":2,"study_min_minutes":90,"signals_required":2,"absent_alert":3},"version":N}` | `200`; lưu vào `courses.settings.risk`; `audit_log` cũ/mới; danh sách phản ánh ≤ 60 s (job tính lại riêng lớp qua outbox `course.settings_changed`) |
| TC-P5-04-20 | AC6 (ngoài khoảng) | – | `attendance_ok_min` 0,49 / 1,01; `activity_min_per_week` -1 / 21; `study_min_minutes` 601; `signals_required` 0 / 4; `absent_alert` 0 / 21 | `422` mỗi ca, chỉ rõ trường |
| TC-P5-04-21 | AC6 (vai, version, mặc định) | – | TA/STUDENT/ADMIN `PUT`; sai `version`; lớp chưa đặt | `403` ×3; `409 VERSION_CONFLICT`; mặc định `0,70 / 1 / 60 / 2 / 3` (đọc `GET $Q`) |
| TC-P5-04-22 | AC6 (UI) | `playwright-cli` `/class/settings` bằng teacher | nhóm "Ngưỡng cần chú ý" | thanh/ô có nhãn + giá trị; không câu mô tả dưới tiêu đề |
| TC-P5-04-23 | AC7 (Hôm nay) | C `AT_RISK` mới (`entered_at` ≤ 7 ngày) | `GET /me/today -H "$TCH"` | mục `STUDENT_ATTENTION` (bậc 95) "1 sinh viên mới vào diện cần chú ý · 761987", `Href` `/students?filter=attention&course=…`; khối `attention` liệt kê ≤ 5 sinh viên `AT_RISK` (tên + mã lớp + cụm lý do ngắn) + "Xem tất cả (n)"; biến ≤ 60 s sau khi job tính lại |
| TC-P5-04-24 | AC7 (ẩn khi rỗng, vai) | lớp 2 không `AT_RISK`; `$SVA`, `$ADM` | `GET /me/today` | khối `attention` **không** hiện (không rỗng giả); sinh viên/ADMIN không thấy cả hai |
| TC-P5-04-25 | AC7 (tối đa 5) | gieo 8 `AT_RISK` ở lớp thử | xem khối | ≤ 5 dòng + "Xem tất cả (8)" |
| TC-P5-04-26 | AC8 (UI) | `playwright-cli` 1280 | `/students` | ô tìm + chip `Cần chú ý` · `Vắng nhiều` · `Ít hoạt động` (**không** chip `Điểm giảm`); `DataTable` cột: Sinh viên · Chuyên cần ("Vắng 2/9") · Phát biểu · Hoạt động học ("45 phút · 3 câu hỏi") · Rủi ro ("Cần chú ý" đỏ / "Theo dõi" hổ phách / trống); không nút trên dòng; bấm dòng → `/students/[id]` |
| TC-P5-04-27 | AC8 (giữ trạng thái, deep link) | – | gõ tìm + chọn chip → vào hồ sơ → quay lại; mở `/students?filter=attention` từ "Hôm nay" | ô tìm và chip giữ; `?filter=` bật đúng chip |
| TC-P5-04-28 | AC8 (trạng thái, ≥ 720 px) | – | tải chậm; `page.route` 500; tìm không có kết quả; `resize 720 900` và `600 900`; cuộn lớp 1.000 SV | khung xương; lỗi + `Thử lại`; "Không có sinh viên nào khớp." + nút `Bỏ lọc`; ≥ 720 dùng được, dưới đó bảng cuộn ngang trong khung; cuộn 1.000 dòng mượt (ảo hoá: số `<tr>` trong DOM ≪ 1.000) |
| TC-P5-04-29 | AC9 (sinh viên 403) | – | `$SVA` (kể cả `uid` của mình), `$ADM` gọi `…/students`, `…/students/{uid}/profile`… | `403` |
| TC-P5-04-30 | AC9 (không lộ qua API sinh viên) | C có `AT_RISK` và ghi chú canary | crawl mọi route của STUDENT bằng `$SVC` (C): `me/*`, `chat/*`, `threads`, `me/today`, `notifications`, `calendar`, `library`; `jq '[..\|objects\|keys[]]\|unique'` | không có khoá `risk`, `risk_level`, `at_risk`, `observation`, `watch`; không câu nhận định |
| TC-P5-04-31 | AC9 (không lộ qua AI) | `$SVC` | hỏi chat "Em có bị coi là cần chú ý không?", "Giảng viên ghi gì về em?"; xem payload LLM (`/_test/llm/stats` hoặc log provider giả) | câu trả lời thông thường, không tiết lộ; payload không có `risk`/`AT_RISK`/ghi chú |
| TC-P5-04-32 | AC9 (kiến trúc) | repo | `cd backend-go && go test ./internal/crm ./internal/contract -run 'TestSensitiveQueriesOnlyFromCRM\|TestStudentNeverSeesRisk' -v; go test ./internal/agent -run TestAgentHasNoRiskOrObservationAccess -v` | `ok` |
| TC-P5-04-33 | AC10 (lớp mới) | `$CN` không buổi/không dữ liệu | `GET $CNQ/students -H "$TCH"` | đủ sinh viên, số 0, `risk.level='NONE'`; chuyên cần "—" khi chưa ghi buổi nào (UI); không chia cho 0; không lỗi |
| TC-P5-04-34 | AC10 (job chưa chạy; lớp lưu trữ) | `student_risk` trống; lớp `ARCHIVED` | danh sách; UI | `NONE`, `computed_at` rỗng, giao diện không hiện "Rủi ro"; lớp lưu trữ đọc được (chỉ đọc) |
| TC-P5-04-35 | UI | frontend | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW students.spec.ts -g 'students list filters search keep state\|students deep link filter\|students states'`; `$PW a11y.spec.ts -g students`; `$PW class-join.spec.ts -g 'risk thresholds'`; `$PW today.spec.ts -g 'staff attention block'` | `0`; xanh |

## Nhánh lỗi
TC-P5-04-02 (cursor sai), 07, 12, 14, 15, 20, 28, 33, 34. Thêm TC-P5-04-E1: `crm.risk` lỗi giữa chừng (tắt worker giữa lúc chạy) → tick sau thử lại, không dữ liệu nửa vời (`computed_at` chỉ đổi cho hàng đã xong; không hàng trùng).

## Phân quyền
TC-P5-04-21, 24, 29, 30, 31, 32; TC-P5-04-P1: TA lớp 1 gọi `…/students` lớp 2 → `403`; teacher lớp 3 → `403`.

## Script chạy được: `docs/sprints/7/qc/scripts/`
`seed-perf.sql` (TC-08/09): `generate_series` 1 lớp 1.000 SV + 19 lớp 100 SV, 15 buổi, ≥ 280.000 `study_time_daily`, ≥ 5.000 `participation_events`; `students-cursor-walk.sh` (TC-02): đi hết `next_cursor` và đếm. Viết ở pha 2 khi có schema thật (không đọc code, chỉ `\d` từ DB).

## Câu hỏi (Q-QC)
- **Q-QC-P5-07** TC-08: lấy câu SQL của danh sách từ đâu (hộp đen)? Đề xuất bật `auto_explain.log_min_duration=0` + `log_statement` ở Postgres test, hoặc dev ghi câu vào handoff. Nếu thiếu, kiểm gián tiếp bằng thời gian (TC-09) và hỏi PM.
- **Q-QC-P5-08** TC-05: cú pháp "hai lọc cùng lúc" (`filter=a&filter=b` hay `filter=a,b`) không nêu; ghi dạng thật.
- **Q-QC-P5-09** TC-15: không có đồng hồ giả để kiểm "02:00 chạy bù"; kiểm bằng khoá Redis. PM xác nhận hoặc yêu cầu lệnh chạy một lần (đã có `gateway admin crm-risk-run`).

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
