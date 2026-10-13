# QC test case — US-P4-01 (migration `00012`, ticket, nút "Nhờ giảng viên", tự mở, trần 5)
Nguồn: `docs/specs/FEAT-escalation-notify/US.md` US-P4-01 AC1…AC14 + `SRS.md` 2, 3.3, 5, 6.1, 9.4; `proposals.md` #2, #8, #10; `docs/phases/P4.md` L1. Viết trước khi có handoff, hộp đen: không đọc code của dev.

**Biến shell** (SRS 9.1): `GW`; `C1` = lớp 1 (`761987`); `Q=$GW/api/v1/courses/$C1`; `SVA`/`SVB`/`SVC` = `Authorization: Bearer …` của `sv.gioi@` / `sv.kha@` / `sv.nguyco@edupilot.local`; `TA_`, `TCH`, `ADM` = `ta@`, `teacher@`, `admin@`; `j='curl -sk -H "Content-Type: application/json"'`; `idem`; `PSQL`, `RDS`, `MP=http://localhost:8025`; `PW='pnpm -C frontend exec playwright test'`. Mật khẩu và cách lấy token lấy từ `scripts/seed.mjs` (`POST /auth/login`). Provider giả (`LLM_PROVIDER=fake`). Nghiệm thu giao diện bằng `playwright-cli`. "Tin AI thường" `$MA` = câu trả lời `DONE`, `low_confidence=false` của `$SVA` trong phiên `$SA`; "tin AI yếu" `$MW` = câu `low_confidence=true` (ép bằng `update courses set escalation_threshold=0.95 where id='$C1'` rồi hỏi câu truy xuất yếu; đặt lại `0.80` sau TC). Câu hỏi chung: `Q-QC-P4-01` (cuối file).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P4-01-01 | AC1 | DB trống, repo ở HEAD | `cd backend-go && goose -dir db/migrations postgres "$DATABASE_URL" up && goose … down && goose … up && goose … status \| grep 00012` | `up`, `down` (một bước), `up` đều sạch, `00012` `Applied` |
| TC-P4-01-02 | AC1 | như trên | `grep -c 'CREATE TABLE mail_inbound' db/migrations/00012_escalation_notify.sql`; `grep -n '^ALTER TABLE' …`; `grep -c 'CREATE.* INDEX .*\(documents_knowledge_key\|mail_outbox_dead_idx\)' …` | `0`; đúng 4 dòng (`forum_threads`, `forum_posts`, `user_settings`, `courses`); `2` |
| TC-P4-01-03 | AC1 | repo | `grep -n '00012' docs/PROGRESS.md` | Mục "Ánh xạ migration" ghi `00012 escalation_notify` |
| TC-P4-01-04 | AC2 | DB sau `00012` | `$PSQL "\d escalation_tickets"`; thử chèn vi phạm: `kind='X'`; `status='OPEN'` kèm `claimed_by`; `status='CLAIMED'` thiếu `claimed_by`; `status='ANSWERED'` thiếu `answered_by`; `kind='QUESTION'` không `message_id`; `kind='GRADE_APPEAL'` không `appeal_payload`; `kind='QUESTION'` có `appeal_payload`; `message_id` trùng; `(course_id,message_id)` lệch lớp | Mỗi ca chèn vi phạm → `23514` (CHECK) / `23505` (trùng) / `23503` (FK lệch lớp); bảng có `message_id` UNIQUE, FK phức hợp tới `chat_messages (course_id,id)` và `chat_sessions (course_id,id)` |
| TC-P4-01-05 | AC3 | DB sau `00012` | `\d post_reports`, `\d knowledge_items`; chèn `post_reports` cả `thread_id` và `post_id`; cả hai rỗng; cùng `(reporter_id,target_id)` hai lần; `reason` ngoài enum; xoá `content_chunks` có `knowledge_items.chunk_id` | `23514`; `23514`; `23505`; lỗi enum; `knowledge_items` bị xoá theo (`ON DELETE CASCADE`); `UNIQUE (source_type, source_id)` có |
| TC-P4-01-06 | AC3 (mặc định + UPDATE dữ liệu cũ) | DB ở `00011` có: lớp `version=3` `escalation_threshold=0.60`, lớp `0.70`, `user_settings` `version=2` `notify_ticket_by_mail=true` | `goose up` đến `00012`; `select escalation_threshold from courses`; `select notify_ticket_by_mail from user_settings` | `0.60` → `0.80` (kể cả `version≥2`); `0.70` giữ; mọi `notify_ticket_by_mail=false`; cột mới có mặc định `0.80` / `false` |
| TC-P4-01-07 | AC4 (tạo thủ công) | `$MA` của `$SVA` | `j -X POST $GW/api/v1/chat/messages/$MA/escalate -H "$SVA"` | `201 {ticket:{id,status:"OPEN",created_at},already:false}` |
| TC-P4-01-08 | AC4 (gọi lại tuần tự) | sau TC-07 | lặp cùng lệnh 3 lần; `$PSQL "select count(*) from escalation_tickets where message_id='$MA'"` | `200 {already:true}` cùng `id`; count = 1 |
| TC-P4-01-09 | AC4 (10 lời gọi song song) | `$MA2` chưa có ticket | `for i in $(seq 10); do j -X POST …/$MA2/escalate -H "$SVA" -o /tmp/r$i -w '%{http_code}\n' & done; wait`; đếm | Đúng 1 mã `201`, 9 mã `200`; count = 1; mọi thân cùng `ticket.id` |
| TC-P4-01-10 | AC4 (ticket đã `CLOSED` không mở lại) | ticket của `$MA` ở `CLOSED` (qua dismiss) | gọi lại `escalate` | `200 already:true`, `status` vẫn `CLOSED` (không `OPEN`) |
| TC-P4-01-11 | AC4 (đã đủ 5 mà bấm lại tin đã nhờ) | `$SVA` đã có 5 ticket mở, trong đó `$MA` | bấm lại `escalate` ở `$MA` | `200 already:true` (tra `message_id` trước khi đếm trần), không `409` |
| TC-P4-01-12 | AC5 (tự mở dưới ngưỡng) | ngưỡng 0,95; intent `COURSE_QA` | hỏi câu truy xuất yếu qua SSE `POST /chat/…/messages`; đọc khung `done`; `select origin,confidence from escalation_tickets where message_id=…`; `select count(*) from outbox where topic='escalation.created' and …` | `done` có `ticket:{id,status,created_at}`; `origin='AUTO_LOW_CONFIDENCE'`, `confidence` có; đúng 1 dòng outbox `escalation.created`; `GET /chat/sessions/$S/messages` cũng có `ticket` |
| TC-P4-01-13 | AC5 (nguyên tử với câu trả lời) | có cách làm outbox lỗi (QC chèn trigger tạm: `create or replace function … raise exception` trên `outbox` insert, chỉ ở DB test) | hỏi câu yếu | Câu trả lời không `DONE` (hoặc `FAILED`), **không** có ticket mồ côi (`escalation_tickets` không thêm); gỡ trigger sau TC |
| TC-P4-01-14 | AC5 (các intent được tự mở) | ngưỡng 0,95 | lần lượt hỏi `LIBRARY_SEARCH`, `PERSONAL_*` (điểm của mình), `WHAT_IF_GRADE`, `EXAM_SCHEDULE`, `UPCOMING_EVENTS` có `low_confidence` | mỗi intent có đủ điều kiện (SRS) → có ticket `AUTO_LOW_CONFIDENCE`; ghi intent nào không (→ `Q-QC-P4-02`) |
| TC-P4-01-15 | AC6 (nhánh không tự mở) | ngưỡng 0,95 | gửi tin `CRISIS` ("Em không muốn sống nữa."), `OTHER_PERSON` (hỏi điểm bạn khác), `SMALLTALK` ("chào bạn"); câu `degraded=true` (tắt provider giả: `LLM_PROVIDER` lỗi); câu `low_confidence=false` | không có ticket mới cho cả 5 trường hợp |
| TC-P4-01-16 | AC6 (`AUTO_NO_FORMULA`) | lớp không có công thức điểm `CONFIRMED` (lớp 2 hoặc lớp mới) | `SVx` hỏi "Điểm cuối kỳ tính thế nào?" | Câu mẫu "Lớp chưa có công thức điểm chính thức…"; ticket `origin='AUTO_NO_FORMULA'`; đổi lời văn câu mẫu không đổi hành vi (kiểm bằng đọc SRS: nhận biết bằng cờ, không so chuỗi — xem `Q-QC-P4-03`) |
| TC-P4-01-17 | AC6 (bảng `ShouldEscalate`) | repo | `cd backend-go && go test ./internal/agent -run TestShouldEscalateTable -v` | `ok`, ≥ 14 dòng bảng (QC đếm `=== RUN`/dòng con) |
| TC-P4-01-18 | AC7 (nút hiện / không hiện) | `/chat` của `$SVA` (`playwright-cli`, 1280 × 800) | `goto $GW/chat`, mở phiên có: tin AI thường, câu `CRISIS`, câu từ chối `OTHER_PERSON`, câu `SMALLTALK`, tin `FAILED`, tin `CANCELLED`, tin đang `STREAMING`; `find "Nhờ giảng viên"` dưới từng tin | Nút có ở tin AI thường `DONE` chưa có ticket; **không** ở `CRISIS` / `OTHER_PERSON` / `SMALLTALK` / `FAILED` / `CANCELLED` / `STREAMING` |
| TC-P4-01-19 | AC7 (dòng trạng thái sau khi nhờ) | tin `low_confidence` đã tự mở ticket | quan sát tin | Không có nút; có dòng "Đang chờ giảng viên · vừa gửi" (đúng chữ) |
| TC-P4-01-20 | AC7 (tooltip duy nhất; không câu mô tả thêm) | tin AI thường | rê chuột lên nút; `eval document.body.innerText` quanh nút | Tooltip đúng "Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI"; không câu mô tả nào khác dưới nút |
| TC-P4-01-21 | AC7 (khoá khi đủ 5) | `$SVA` có 5 ticket mở | mở tin AI thường thứ 6 | Nút hiện nhưng khoá (`disabled`), tooltip nêu lý do |
| TC-P4-01-22 | AC8 (bấm đúp ≤ 100 ms, 1 request) | `playwright-cli` + theo dõi `network` | bấm đúp / ba lần liên tiếp vào nút | đúng 1 `POST …/escalate`; dòng "Đang chờ giảng viên · vừa gửi" hiện ≤ 100 ms (đo `performance.now`) |
| TC-P4-01-23 | AC8 (mất mạng) | `playwright-cli` `context.setOffline(true)` | bấm nút khi offline → `setOffline(false)` | giữ "Đang gửi…"; sau khi có mạng tạo đúng 1 ticket (`count=1`) |
| TC-P4-01-24 | AC8 (lỗi 4xx/5xx) | chặn route bằng `page.route` trả 500 | bấm nút | nút hoàn lại + `InlineNotice` "Chưa gửi được. Thử lại" một dòng có nút `Thử lại` |
| TC-P4-01-25 | AC9 (trần 5) | `$SVA` có 5 ticket `OPEN`/`CLAIMED` ở lớp 1 | bấm nút ở câu thứ 6 | `409 TICKET_LIMIT_REACHED`, không tạo |
| TC-P4-01-26 | AC9 (tự mở bị chặn) | như trên, ngưỡng 0,95 | hỏi câu yếu; đọc metrics `chat_tickets_suppressed_total` trước/sau (`curl $GW/metrics` hoặc nơi spec chỉ) | câu trả lời vẫn hiện "AI chưa đủ chắc chắn về câu này", không dòng trạng thái ticket, số đếm tăng 1 |
| TC-P4-01-27 | AC9 (theo lớp, theo sinh viên, `ANSWERED`/`CLOSED` không tính) | `$SVA` 5 ticket mở ở lớp 1 | (a) `$SVA` nhờ ở lớp 2; (b) `$SVB` nhờ ở lớp 1; (c) trả lời 1 ticket của `$SVA` rồi nhờ tin thứ 6 | (a) `201`; (b) `201`; (c) `201` (`ANSWERED` không tính) |
| TC-P4-01-28 | AC10 (giảng viên chỉ thấy cặp hỏi–đáp) | phiên `$SA` có tin liền trước `U-1` chứa `CANARY-9K3Z`, `U` + `M`, và tin liền sau `U+1` chứa `CANARY-9K3Z`; một phiên khác và một phiên xoá mềm cũng chứa canary; ticket tạo từ `M` | `j $Q/tickets/$T -H "$TCH" \| tee r.json`; `grep -c CANARY-9K3Z r.json`; thử mọi route `GET` staff đã biết | `0`; chỉ có nội dung `U`, `M` (+ nguồn, khối tool), `confidence` có số, tin của ticket; không tên phiên khác, không danh sách phiên |
| TC-P4-01-29 | AC11a (chủ phiên) | `$MA` của `$SVA` | `escalate` bằng `$SVB` (SV khác); bằng `$TA_`, `$TCH`, `$ADM`; `mid` không tồn tại | `404 NOT_FOUND` (cả hai ca đầu, không lộ tồn tại); `403 FORBIDDEN` ×3; `404` |
| TC-P4-01-30 | AC11b | sinh viên `PENDING`, `REMOVED`, ngoài lớp | `escalate` | `403` mỗi ca |
| TC-P4-01-31 | AC11c | lớp `ARCHIVED` | `escalate` | `409 COURSE_ARCHIVED` |
| TC-P4-01-32 | AC11d (giờ thi) | `$SVA` đang trong giờ thi (khoá) | `escalate` | `EXAM_LOCKED` cùng mã và cùng trạng thái HTTP với chat riêng khi thi (đối chiếu: gửi tin chat riêng lúc đó) |
| TC-P4-01-33 | AC11e (sinh viên không thấy nội bộ) | có ticket (tự mở + tay) | `GET /chat/sessions/$S/messages`, SSE `done`, `GET /chat/sessions`, mọi route `chat/*` của `$SVA`; `jq '[..\|objects\|keys[]]\|unique'` | không có khoá `confidence`, `claimed_by`, `origin`, `sla_stage`, `answered_by` ngoài `ticket:{id,status,created_at,answered_at}` |
| TC-P4-01-34 | AC12 (không đủ điều kiện) | tin vai `USER`; tin `stream_status≠DONE`; câu mẫu `CRISIS` / `OTHER_PERSON` | `escalate` mỗi tin | `422 MESSAGE_NOT_ESCALATABLE`; không tạo ticket |
| TC-P4-01-35 | AC12 (Redis/DB chết) | dừng `redis` (`docker compose stop redis`) rồi dừng `postgres` ở lần khác | `escalate` | `503 SERVICE_UNAVAILABLE` kèm `retry_after`; không dòng nửa vời (`select count(*)` không đổi); bật lại dịch vụ |
| TC-P4-01-36 | AC12 (Idempotency-Key tuỳ chọn) | `$MA3` | `escalate` có và không `Idempotency-Key`; lặp cùng key | cả hai đều chạy; cùng key → cùng kết quả, 1 ticket |
| TC-P4-01-37 | AC13 (hợp đồng) | repo | `cd backend-go && go test ./internal/contract/...`; `grep -n 'escalate' api/openapi.yaml`; kiểm `ticket` nullable ở tin trợ lý và khung `done` | `ok`; có `POST /chat/messages/{mid}/escalate`; thay đổi chỉ cộng; tin không có ticket → `ticket: null` |
| TC-P4-01-38 | AC14 (chỉ mục) | 10.000 ticket gieo ngẫu nhiên, 50 lớp (`generate_series` bằng `$PSQL`) | `EXPLAIN` trang đầu: danh sách theo lớp + trạng thái; theo sinh viên; truy vấn job SLA (lấy từ log truy vấn hoặc hỏi dev → `Q-QC-P4-04`) | không có `Seq Scan on escalation_tickets`; chỉ mục phức hợp bắt đầu bằng `course_id` |

## Nhánh lỗi (SRS 3.3)
Phủ ở TC-P4-01-10, 11, 13, 22–26, 31, 32, 34, 35. Thêm:
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P4-01-E1 | Hai tab cùng bấm nút | mở `/chat` hai tab cùng phiên, bấm đồng thời | 1 ticket; cả hai tab hiện dòng trạng thái sau khi làm mới |
| TC-P4-01-E2 | Phiên bị xoá mềm sau khi nhờ | xoá mềm phiên → giảng viên mở ticket | chi tiết vẫn mở (xem US-P4-03); `session_id` giữ nguyên |
| TC-P4-01-E3 | Log không chứa nội dung | `docker compose logs gateway worker \| grep -c CANARY-9K3Z` sau TC-28 | `0` |

## Phân quyền
TC-P4-01-29, 30, 31, 32, 33 (ma trận SRS 2 dòng 1, 3). Thêm TC-P4-01-P1: `$SVB` gọi `GET /chat/sessions/$SA/messages` → `404`; không thấy ticket của `$SVA`.

## Script chạy được: `docs/sprints/7/qc/scripts/`
`tc-p4-01-concurrent.sh` (TC-09): mười `curl` song song, in phân bố mã HTTP. Viết ở pha 2 khi có stack.

## Câu hỏi (Q-QC)
- **Q-QC-P4-01** Cách lấy token / mật khẩu seed chưa nêu ở US/SRS; tôi dùng `scripts/seed.mjs` (`POST /auth/login`). Xác nhận không đổi.
- **Q-QC-P4-02** AC5 liệt kê `PERSONAL_*` và `WHAT_IF_GRADE`; TC-14 cần ép `low_confidence` bằng provider giả cho từng intent. Spec không nói cách ép cho intent có tool (độ tin cậy = 1,000 theo US-P3-07 AC1). Hỏi PM: intent có tool có bao giờ `low_confidence=true`? Nếu không, TC-14 chỉ kiểm `COURSE_QA`, `LIBRARY_SEARCH`.
- **Q-QC-P4-03** AC6 yêu cầu nhận biết "câu mẫu chưa có công thức" bằng cờ, không so chuỗi. Hộp đen chỉ kiểm hành vi; không thể chứng minh "không so chuỗi" nếu không đọc code. Đề xuất: phần đó chỉ kiểm bằng `TestShouldEscalateTable` do dev chạy. PM xác nhận.
- **Q-QC-P4-04** AC14 "truy vấn của job SLA": hộp đen không biết câu SQL. Đề xuất dev ghi câu vào handoff để QC chạy `EXPLAIN`.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu (spec v1.1 APPROVED).
