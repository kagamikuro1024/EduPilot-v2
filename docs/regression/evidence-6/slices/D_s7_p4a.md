# Slice D_s7_p4a: 108 ca còn N

### TC-P4-01-01 [AC1] pre: DB trống, repo ở HEAD
STEP: `cd backend-go && goose -dir db/migrations postgres "$DATABASE_URL" up && goose … down && goose … up && goose … status \| grep 00012`
EXPECT: `up`, `down` (một bước), `up` đều sạch, `00012` `Applied`

### TC-P4-01-02 [AC1] pre: như trên
STEP: `grep -c 'CREATE TABLE mail_inbound' db/migrations/00012_escalation_notify.sql`; `grep -n '^ALTER TABLE' …`; `grep -c 'CREATE.* INDEX .*\(documents_knowledge_key\|mail_outbox_dead_idx\)' …`
EXPECT: `0`; đúng 4 dòng (`forum_threads`, `forum_posts`, `user_settings`, `courses`); `2`

### TC-P4-01-03 [AC1] pre: repo
STEP: `grep -n '00012' docs/PROGRESS.md`
EXPECT: Mục "Ánh xạ migration" ghi `00012 escalation_notify`

### TC-P4-01-04 [AC2] pre: DB sau `00012`
STEP: `$PSQL "\d escalation_tickets"`; thử chèn vi phạm: `kind='X'`; `status='OPEN'` kèm `claimed_by`; `status='CLAIMED'` thiếu `claimed_by`; `status='ANSWERED'` thiếu `answered_by`; `kind='QUESTION'` không `message_id`; `kind='GRADE_APPEAL'` không `appeal_payload`; `kind='QUESTION'` có `appeal_payload`; `message_id` trùng; `(course_id,message_id)` lệch lớp
EXPECT: Mỗi ca chèn vi phạm → `23514` (CHECK) / `23505` (trùng) / `23503` (FK lệch lớp); bảng có `message_id` UNIQUE, FK phức hợp tới `chat_messages (course_id,id)` và `chat_sessions (course_id,id)`

### TC-P4-01-05 [AC3] pre: DB sau `00012`
STEP: `\d post_reports`, `\d knowledge_items`; chèn `post_reports` cả `thread_id` và `post_id`; cả hai rỗng; cùng `(reporter_id,target_id)` hai lần; `reason` ngoài enum; xoá `content_chunks` có `knowledge_items.chunk_id`
EXPECT: `23514`; `23514`; `23505`; lỗi enum; `knowledge_items` bị xoá theo (`ON DELETE CASCADE`); `UNIQUE (source_type, source_id)` có

### TC-P4-01-06 [AC3 (mặc định + UPDATE dữ liệu cũ)] pre: DB ở `00011` có: lớp `version=3` `escalation_threshold=0.60`, lớp `0.70`, `user_settings` `version=2` `notify_ticket_by_mail=true`
STEP: `goose up` đến `00012`; `select escalation_threshold from courses`; `select notify_ticket_by_mail from user_settings`
EXPECT: `0.60` → `0.80` (kể cả `version≥2`); `0.70` giữ; mọi `notify_ticket_by_mail=false`; cột mới có mặc định `0.80` / `false`

### TC-P4-01-10 [AC4 (ticket đã `CLOSED` không mở lại)] pre: ticket của `$MA` ở `CLOSED` (qua dismiss)
STEP: gọi lại `escalate`
EXPECT: `200 already:true`, `status` vẫn `CLOSED` (không `OPEN`)

### TC-P4-01-11 [AC4 (đã đủ 5 mà bấm lại tin đã nhờ)] pre: `$SVA` đã có 5 ticket mở, trong đó `$MA`
STEP: bấm lại `escalate` ở `$MA`
EXPECT: `200 already:true` (tra `message_id` trước khi đếm trần), không `409`

### TC-P4-01-13 [AC5 (nguyên tử với câu trả lời)] pre: có cách làm outbox lỗi (QC chèn trigger tạm: `create or replace function … raise exception` trên `outbox` insert, chỉ ở DB test)
STEP: hỏi câu yếu
EXPECT: Câu trả lời không `DONE` (hoặc `FAILED`), **không** có ticket mồ côi (`escalation_tickets` không thêm); gỡ trigger sau TC

### TC-P4-01-14 [AC5 (các intent được tự mở)] pre: ngưỡng 0,95
STEP: lần lượt hỏi `LIBRARY_SEARCH`, `PERSONAL_*` (điểm của mình), `WHAT_IF_GRADE`, `EXAM_SCHEDULE`, `UPCOMING_EVENTS` có `low_confidence`
EXPECT: mỗi intent có đủ điều kiện (SRS) → có ticket `AUTO_LOW_CONFIDENCE`; ghi intent nào không (→ `Q-QC-P4-02`)

### TC-P4-01-15 [AC6 (nhánh không tự mở)] pre: ngưỡng 0,95
STEP: gửi tin `CRISIS` ("Em không muốn sống nữa."), `OTHER_PERSON` (hỏi điểm bạn khác), `SMALLTALK` ("chào bạn"); câu `degraded=true` (tắt provider giả: `LLM_PROVIDER` lỗi); câu `low_confidence=false`
EXPECT: không có ticket mới cho cả 5 trường hợp

### TC-P4-01-16 [AC6 (`AUTO_NO_FORMULA`)] pre: lớp không có công thức điểm `CONFIRMED` (lớp 2 hoặc lớp mới)
STEP: `SVx` hỏi "Điểm cuối kỳ tính thế nào?"
EXPECT: Câu mẫu "Lớp chưa có công thức điểm chính thức…"; ticket `origin='AUTO_NO_FORMULA'`; đổi lời văn câu mẫu không đổi hành vi (kiểm bằng đọc SRS: nhận biết bằng cờ, không so chuỗi — xem `Q-QC-P4-03`)

### TC-P4-01-17 [AC6 (bảng `ShouldEscalate`)] pre: repo
STEP: `cd backend-go && go test ./internal/agent -run TestShouldEscalateTable -v`
EXPECT: `ok`, ≥ 14 dòng bảng (QC đếm `=== RUN`/dòng con)

### TC-P4-01-18 [AC7 (nút hiện / không hiện)] pre: `/chat` của `$SVA` (`playwright-cli`, 1280 × 800)
STEP: `goto $GW/chat`, mở phiên có: tin AI thường, câu `CRISIS`, câu từ chối `OTHER_PERSON`, câu `SMALLTALK`, tin `FAILED`, tin `CANCELLED`, tin đang `STREAMING`; `find "Nhờ giảng viên"` dưới từng tin
EXPECT: Nút có ở tin AI thường `DONE` chưa có ticket; **không** ở `CRISIS` / `OTHER_PERSON` / `SMALLTALK` / `FAILED` / `CANCELLED` / `STREAMING`

### TC-P4-01-19 [AC7 (dòng trạng thái sau khi nhờ)] pre: tin `low_confidence` đã tự mở ticket
STEP: quan sát tin
EXPECT: Không có nút; có dòng "Đang chờ giảng viên · vừa gửi" (đúng chữ)

### TC-P4-01-20 [AC7 (tooltip duy nhất; không câu mô tả thêm)] pre: tin AI thường
STEP: rê chuột lên nút; `eval document.body.innerText` quanh nút
EXPECT: Tooltip đúng "Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI"; không câu mô tả nào khác dưới nút

### TC-P4-01-21 [AC7 (khoá khi đủ 5)] pre: `$SVA` có 5 ticket mở
STEP: mở tin AI thường thứ 6
EXPECT: Nút hiện nhưng khoá (`disabled`), tooltip nêu lý do

### TC-P4-01-22 [AC8 (bấm đúp ≤ 100 ms, 1 request)] pre: `playwright-cli` + theo dõi `network`
STEP: bấm đúp / ba lần liên tiếp vào nút
EXPECT: đúng 1 `POST …/escalate`; dòng "Đang chờ giảng viên · vừa gửi" hiện ≤ 100 ms (đo `performance.now`)

### TC-P4-01-23 [AC8 (mất mạng)] pre: `playwright-cli` `context.setOffline(true)`
STEP: bấm nút khi offline → `setOffline(false)`
EXPECT: giữ "Đang gửi…"; sau khi có mạng tạo đúng 1 ticket (`count=1`)

### TC-P4-01-24 [AC8 (lỗi 4xx/5xx)] pre: chặn route bằng `page.route` trả 500
STEP: bấm nút
EXPECT: nút hoàn lại + `InlineNotice` "Chưa gửi được. Thử lại" một dòng có nút `Thử lại`

### TC-P4-01-25 [AC9 (trần 5)] pre: `$SVA` có 5 ticket `OPEN`/`CLAIMED` ở lớp 1
STEP: bấm nút ở câu thứ 6
EXPECT: `409 TICKET_LIMIT_REACHED`, không tạo

### TC-P4-01-26 [AC9 (tự mở bị chặn)] pre: như trên, ngưỡng 0,95
STEP: hỏi câu yếu; đọc metrics `chat_tickets_suppressed_total` trước/sau (`curl $GW/metrics` hoặc nơi spec chỉ)
EXPECT: câu trả lời vẫn hiện "AI chưa đủ chắc chắn về câu này", không dòng trạng thái ticket, số đếm tăng 1

### TC-P4-01-27 [AC9 (theo lớp, theo sinh viên, `ANSWERED`/`CLOSED` không tính)] pre: `$SVA` 5 ticket mở ở lớp 1
STEP: (a) `$SVA` nhờ ở lớp 2; (b) `$SVB` nhờ ở lớp 1; (c) trả lời 1 ticket của `$SVA` rồi nhờ tin thứ 6
EXPECT: (a) `201`; (b) `201`; (c) `201` (`ANSWERED` không tính)

### TC-P4-01-30 [AC11b] pre: sinh viên `PENDING`, `REMOVED`, ngoài lớp
STEP: `escalate`
EXPECT: `403` mỗi ca

### TC-P4-01-31 [AC11c] pre: lớp `ARCHIVED`
STEP: `escalate`
EXPECT: `409 COURSE_ARCHIVED`

### TC-P4-01-32 [AC11d (giờ thi)] pre: `$SVA` đang trong giờ thi (khoá)
STEP: `escalate`
EXPECT: `EXAM_LOCKED` cùng mã và cùng trạng thái HTTP với chat riêng khi thi (đối chiếu: gửi tin chat riêng lúc đó)

### TC-P4-01-33 [AC11e (sinh viên không thấy nội bộ)] pre: có ticket (tự mở + tay)
STEP: `GET /chat/sessions/$S/messages`, SSE `done`, `GET /chat/sessions`, mọi route `chat/*` của `$SVA`; `jq '[..\|objects\|keys[]]\|unique'`
EXPECT: không có khoá `confidence`, `claimed_by`, `origin`, `sla_stage`, `answered_by` ngoài `ticket:{id,status,created_at,answered_at}`

### TC-P4-01-34 [AC12 (không đủ điều kiện)] pre: tin vai `USER`; tin `stream_status≠DONE`; câu mẫu `CRISIS` / `OTHER_PERSON`
STEP: `escalate` mỗi tin
EXPECT: `422 MESSAGE_NOT_ESCALATABLE`; không tạo ticket

### TC-P4-01-35 [AC12 (Redis/DB chết)] pre: dừng `redis` (`docker compose stop redis`) rồi dừng `postgres` ở lần khác
STEP: `escalate`
EXPECT: `503 SERVICE_UNAVAILABLE` kèm `retry_after`; không dòng nửa vời (`select count(*)` không đổi); bật lại dịch vụ

### TC-P4-01-36 [AC12 (Idempotency-Key tuỳ chọn)] pre: `$MA3`
STEP: `escalate` có và không `Idempotency-Key`; lặp cùng key
EXPECT: cả hai đều chạy; cùng key → cùng kết quả, 1 ticket

### TC-P4-01-37 [AC13 (hợp đồng)] pre: repo
STEP: `cd backend-go && go test ./internal/contract/...`; `grep -n 'escalate' api/openapi.yaml`; kiểm `ticket` nullable ở tin trợ lý và khung `done`
EXPECT: `ok`; có `POST /chat/messages/{mid}/escalate`; thay đổi chỉ cộng; tin không có ticket → `ticket: null`

### TC-P4-01-38 [AC14 (chỉ mục)] pre: 10.000 ticket gieo ngẫu nhiên, 50 lớp (`generate_series` bằng `$PSQL`)
STEP: `EXPLAIN` trang đầu: danh sách theo lớp + trạng thái; theo sinh viên; truy vấn job SLA (lấy từ log truy vấn hoặc hỏi dev → `Q-QC-P4-04`)
EXPECT: không có `Seq Scan on escalation_tickets`; chỉ mục phức hợp bắt đầu bằng `course_id`

### TC-P4-02-01 [AC1] pre: repo
STEP: `grep -rn 'InsertNotification\|insert into notifications' backend-go --include=*.go --include=*.sql \| grep -v 'internal/notify\|store/queries/notify.sql'`
EXPECT: 0 dòng (không nơi nào ngoài `notify` và `notify.sql`). Mọi gói `course`, `exam`, `thread`, `escalation`, `calendar` tạo thông báo qua `notify.Service`

### TC-P4-02-02 [AC1 (gửi hàng loạt = 1 câu SQL)] pre: lớp có 1.000 sinh viên (seed tạm bằng `generate_series`)
STEP: bật `log_statement='all'` của Postgres test; giáo viên đặt lịch thi (kích hoạt `ExamNotifyScheduled`); `grep -c 'INSERT INTO notifications' pg.log`
EXPECT: đúng **1** câu `INSERT … unnest` ghi cho 1.000 người; 1.000 dòng `notifications`; không 1.000 lượt upsert

### TC-P4-02-03 [AC1 (test cũ giữ nguyên)] pre: repo
STEP: `go test ./internal/course ./internal/exam ./internal/thread ./internal/calendar`
EXPECT: `ok`, không test cũ nào bị sửa (`git diff 1518f83.. -- '*_test.go'` chỉ liệt kê các test #8/#10)

### TC-P4-02-04 [AC2 (đẩy tức thì)] pre: `$TCH` mở `sse` (khung SSE đang chạy) và `$TA_` cũng
STEP: `$SVA` tạo ticket (`escalate`), ghi `t0` ngay khi `201` nhận; đọc khung `notification` ở tệp SSE của teacher và TA
EXPECT: khung `notification` có `{id,type:"TICKET_NEW",title,body,link,course_id,read:false,created_at,unread_count}`, đến ≤ 5 s sau `t0`; số chưa đọc đổi không cần tải lại trang

### TC-P4-02-05 [AC2 (p95 trên 20 lần)] pre: như trên, 20 sinh viên/ticket (seed hoặc `SVA`…)
STEP: lặp 20 lần, ghi `t_frame − t_commit`
EXPECT: p95 ≤ 5000 ms (in bảng 20 số)

### TC-P4-02-06 [AC2 (giao diện)] pre: `playwright-cli` hai ngữ cảnh: giảng viên mở `/`; sinh viên ở ngữ cảnh khác
STEP: sinh viên bấm `Nhờ giảng viên`
EXPECT: chuông của giảng viên nhảy số ≤ 5 s, không reload (`page.on('load')` đếm 0)

### TC-P4-02-07 [AC3 (hai bản gateway)] pre: `docker compose up --scale gateway=2` (hoặc hai cổng)
STEP: nối SSE vào bản A (cổng a), `POST escalate` trúng bản B (cổng b); sau đó đảo (nối B, sinh ở A)
EXPECT: cả hai chiều: người dùng nhận khung `notification` ≤ 5 s

### TC-P4-02-08 [AC3 (gateway không giữ trạng thái)] pre: như trên
STEP: dừng bản A khi SSE đang mở ở A; nối lại vào B với `Last-Event-ID` cuối
EXPECT: nhận đủ khung còn thiếu, không trùng

### TC-P4-02-09 [AC4 (đọc bù bằng `Last-Event-ID`)] pre: `$TCH` nối SSE, nhận khung id `E0`; ngắt kết nối 10 s; trong lúc đó sinh 3 ticket (3 sinh viên)
STEP: nối lại `Last-Event-ID: E0`
EXPECT: nhận đúng 3 sự kiện còn thiếu, đúng thứ tự, **không** trùng (so `id`)

### TC-P4-02-10 [AC4 (bộ đệm quá hạn)] pre: xoá bộ đệm Redis của người dùng (`$RDS DEL` khoá buffer `BufKey` — lấy tên từ `docker exec redis redis-cli --scan --pattern '*events*'`)
STEP: nối với `Last-Event-ID` cũ
EXPECT: khung `resync` với lý do `buffer_exceeded`; khách hàng gọi `GET /notifications` đọc bù **từ bảng**; số chưa đọc đúng so `select count(*) … read_at is null`

### TC-P4-02-11 [AC4 (id sai)] pre: –
STEP: nối với `Last-Event-ID: abc`
EXPECT: khung `resync invalid_last_event_id`, kết nối không lỗi (HTTP 200 stream)

### TC-P4-02-12 [AC4 (giao diện)] pre: `playwright-cli`, `context.setOffline(true)` 10 s rồi `false`, 3 ticket sinh trong lúc offline
STEP: quan sát chuông
EXPECT: sau nối lại, chuông hiện đủ 3 thông báo mới, không nhân đôi

### TC-P4-02-13 [AC5 (gộp 5 ticket)] pre: giảng viên chưa đọc; lớp 1
STEP: 5 sinh viên khác nhau tạo ticket trong 1 phút; `GET /notifications -H "$TCH" \| jq '[.[]\|select(.type=="TICKET_NEW")]'`
EXPECT: đúng **một** dòng chưa đọc "5 câu hỏi đang chờ giảng viên"; `id` giữ nguyên qua 5 lần cập nhật; số trong tiêu đề = số ticket `OPEN` hiện tại của lớp

### TC-P4-02-14 [AC5 (đọc xong, ticket mới)] pre: sau TC-13 giảng viên đọc dòng
STEP: tạo ticket mới
EXPECT: dòng **mới** chưa đọc (id khác)

### TC-P4-02-15 [AC5 (số = 0 tự đọc)] pre: tất cả ticket của lớp đã xử lý
STEP: kiểm dòng
EXPECT: tự đánh dấu đã đọc (`read_at` có)

### TC-P4-02-16 [AC5 (số giảm không đặt lại `read_at`)] pre: dòng đã đọc, số 3; một ticket được nhận → 2
STEP: đọc dòng + SSE
EXPECT: `read_at` giữ nguyên; không khung SSE mới nếu số không tăng

### TC-P4-02-17 [AC5 (song song 50 lượt)] pre: 2 handler song song (hai ticket cùng lúc, lặp 50 lần: tạo 100 ticket bằng 2 luồng `curl`)
STEP: so số trong tiêu đề cuối với `select count(*) from escalation_tickets where course_id and status='OPEN'`
EXPECT: bằng nhau

### TC-P4-02-18 [AC5 (loại khác, cá nhân không gộp)] pre: `AI_PENDING`, `POST_REPORTED`, thông báo cá nhân (`TICKET_ANSWERED`)
STEP: tạo 3 bài AI chờ duyệt, 3 báo cáo, 3 câu trả lời cho 3 sinh viên
EXPECT: `AI_PENDING` "3 câu trả lời AI chờ duyệt" một dòng; `POST_REPORTED` "3 bài bị báo cáo chờ xem" một dòng; thông báo cá nhân 3 dòng riêng

### TC-P4-02-20 [AC7 (idempotent theo sự kiện)] pre: sự kiện `escalation.created`
STEP: `kill -STOP` worker giữa xử lý rồi `restart`; hoặc đẩy lại dòng outbox (`update outbox set status='PENDING'`)
EXPECT: đúng 1 thông báo cho mỗi người nhận (`dedupe_key`), đúng 1 khung SSE; cập nhật gộp không phát lại khi số không đổi

### TC-P4-02-22 [AC9 (tab thứ ba)] pre: `$SVA` đã có 2 SSE (`SSE_MAX_PER_USER=2`)
STEP: mở kết nối thứ ba (curl)
EXPECT: `429 SSE_LIMIT_REACHED`

### TC-P4-02-23 [AC9 (giao diện rơi về 30 s)] pre: `playwright-cli` mở 3 tab của cùng người dùng
STEP: tab thứ ba: tạo ticket mới ở ngữ cảnh khác; quan sát
EXPECT: chuông tab 3 không báo lỗi; cập nhật trong ≤ 30 s (làm mới)

### TC-P4-02-24 [AC9 (Redis chết)] pre: `docker compose stop redis`
STEP: tạo thông báo (ticket / thread)
EXPECT: `select count(*) from notifications` có dòng mới; luồng nghiệp vụ không lỗi (HTTP 201/200); SSE `reconnect upstream_unavailable`; chuông ≤ 30 s; bật lại redis

### TC-P4-02-25 [AC10 (chuông UI)] pre: `playwright-cli` 1280 × 800, giảng viên có 12 chưa đọc
STEP: xem chuông
EXPECT: hiện "9+"; danh sách mới nhất trước; bấm một mục → mở đúng `link`; `link` rỗng → `/` ("Hôm nay"); `?course=` được tiêu thụ; đánh dấu đã đọc lạc quan; không toast

### TC-P4-02-26 [AC10 (375 px, a11y)] pre: `playwright-cli resize 375 812`
STEP: đo `getBoundingClientRect` của mục chuông và nút chuông; kiểm `aria-live="polite"` ở số chưa đọc
EXPECT: vùng chạm ≥ 44 px; không tràn ngang (`scrollWidth ≤ innerWidth`)

### TC-P4-02-27 [AC10 (mở danh sách khi có thông báo mới)] pre: mở danh sách, đang đọc mục thứ 3
STEP: đẩy thông báo mới
EXPECT: chèn lên đầu; mục đang đọc không nhảy vị trí (đo `getBoundingClientRect().top` của mục trước/sau, trừ chiều cao mục mới nếu cuộn)

### TC-P4-02-29 [UI] pre: frontend
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW a11y.spec.ts -g bell`
EXPECT: `0`; 0 vi phạm nghiêm trọng

### TC-P4-03-04 [AC1 (không N+1, phân trang con trỏ)] pre: 1.000 ticket
STEP: `EXPLAIN (ANALYZE)` hoặc đếm câu SQL (log) khi gọi `limit=100`
EXPECT: tối đa vài câu SQL (không 100+); không `OFFSET` (`grep -i offset` trong log)

### TC-P4-03-08 [AC2 (UI)] pre: `playwright-cli`, giảng viên 2 lớp
STEP: `/inbox` → chọn "Tất cả lớp của tôi"
EXPECT: mỗi dòng ghi mã lớp; bộ chọn lớp riêng của `/inbox`

### TC-P4-03-12 [AC3 (audit)] pre: `select count(*) from audit_log where entity='escalation_ticket'` trước
STEP: mở chi tiết 3 lần; `claim`; `answer`; `dismiss` (ticket khác)
EXPECT: mở chi tiết **không** thêm dòng; `claim`, `answer`, `dismiss` mỗi hành động 1 dòng kèm `ticket_id`, **không** kèm nội dung (grep canary = 0)

### TC-P4-03-13 [AC4] pre: `$T` `OPEN`
STEP: `$TCH` `POST $Q/tickets/$T/claim`
EXPECT: `200`, `status=CLAIMED`, `claimed_by`=teacher, có `claimed_at`

### TC-P4-03-16 [AC4 (UI hai ngữ cảnh)] pre: `playwright-cli`: ngữ cảnh 1 teacher, ngữ cảnh 2 TA cùng mở `$T3`
STEP: bấm `Nhận` gần như đồng thời (script)
EXPECT: người thua thấy nút `Nhận` hoàn lại + `InlineNotice` một dòng "Cô … đã nhận câu này lúc HH:MM"

### TC-P4-03-18 [AC5 (giao diện sinh viên)] pre: `$SVA` mở `/chat`
STEP: quan sát tin
EXPECT: nhãn "Giảng viên · {tên}" (hoặc "Trợ giảng · {tên}"), **không** nhãn AI, không nút Hữu ích / Không hữu ích

### TC-P4-03-21 [AC5 (thiếu key)] pre: –
STEP: `answer` không `Idempotency-Key`
EXPECT: lỗi (route bắt buộc key — ghi mã thực tế, kỳ vọng `400/422`); không tạo tin

### TC-P4-03-23 [AC5 (phiên đã xoá mềm)] pre: `$SVA` xoá mềm phiên của ticket
STEP: giảng viên `answer`
EXPECT: tin vào phiên "Tin nhắn từ giảng viên" (tạo-hoặc-lấy), `ticket.session_id` không đổi; giảng viên vẫn đọc ngữ cảnh; `$SVA` thấy tin, `Hỏi tiếp` / `Đã rõ` bấm được ở phiên đó

### TC-P4-03-24 [AC5 (không vào lịch sử LLM)] pre: phiên có `STAFF_ANSWER`; `LLM_PROVIDER=fake` ghi payload (`GET $GW/api/v1/_test/llm/stats`)
STEP: `$SVA` hỏi tiếp một câu thường; kiểm payload gửi LLM
EXPECT: không có nội dung `STAFF_ANSWER` / `ESCALATION_FOLLOWUP` trong lịch sử gửi

### TC-P4-03-25 [AC6] pre: `$T6` `CLAIMED` bởi TA
STEP: `$TCH` `answer`
EXPECT: `409 TICKET_ALREADY_CLAIMED` kèm người đã nhận

### TC-P4-03-27 [AC7] pre: `$T` `ANSWERED` bởi teacher
STEP: teacher `answer` thêm; TA `answer` thêm
EXPECT: thêm tin `STAFF_ANSWER` mới, ticket giữ `ANSWERED`, `answered_at` đổi; TA `409`; sinh viên nhận thông báo mới (không gộp, `TICKET_ANSWERED` thêm 1 dòng)

### TC-P4-03-29 [AC8] pre: –
STEP: body chứa MSSV `M2`; email SV khác `sv.kha@edupilot.local`; CCCD 12 số hợp lệ
EXPECT: `422 PII_IN_ANSWER` (3 ca)

### TC-P4-03-31 [AC9] pre: `playwright-cli` giảng viên
STEP: gõ dở ở ticket T1 → chuyển T2 → quay T1; tải lại; đóng trình duyệt rồi mở; chờ ≥ 2 s giữa lần gõ
EXPECT: nháp T1 còn nguyên ở mọi bước; nháp T2 riêng

### TC-P4-03-32 [AC9 (gửi thành công / lỗi)] pre: –
STEP: gửi thành công; rồi `page.route` trả 500
EXPECT: thành công → nháp xoá; lỗi → nháp giữ, nút `Gửi trả lời` hoàn lại + `Thử lại`; nháp không ra khỏi máy (không request chứa nháp trước khi gửi: theo dõi `network`)

### TC-P4-03-33 [AC10 (375 px)] pre: `playwright-cli resize 375 812`
STEP: `/inbox` danh sách → bấm một dòng → chi tiết → Quay lại
EXPECT: danh sách toàn màn hình; chi tiết toàn màn hình (`?ticket=` trong URL); Quay lại về danh sách giữ bộ lọc và vị trí cuộn; ô trả lời dính đáy

### TC-P4-03-34 [AC10 (đo)] pre: như trên
STEP: `eval` đo mọi nút/dòng `getBoundingClientRect()`; `document.documentElement.scrollWidth`
EXPECT: vùng chạm ≥ 44 px; `scrollWidth ≤ innerWidth`

### TC-P4-03-35 [AC10 (≥ 1100 px)] pre: `resize 1280 800`
STEP: mở `/inbox`
EXPECT: chia đôi: danh sách rộng 380 px + chi tiết

### TC-P4-03-36 [AC11 (cắt được)] pre: `/inbox` chia đôi
STEP: `J`/`K`; `R`; `⌘↵`/`Ctrl+↵`; gõ `J` khi đang ở ô trả lời
EXPECT: `J`/`K` chuyển ticket; `R` focus ô trả lời; `Ctrl+↵` gửi rồi sang ticket kế; gõ trong ô nhập không kích hoạt phím tắt; tooltip phím trên `Gửi trả lời`. **Nếu dev ghi "Nợ" cắt AC11 trong `report.md`: TC này "KHÔNG ÁP DỤNG", không tính FAIL**

### TC-P4-03-37 [AC12] pre: có 2 ticket `OPEN` (1 do teacher nhận)
STEP: `GET $GW/api/v1/me/today -H "$TCH"` (hoặc `$Q/today`)
EXPECT: mục `TICKET` mỗi lớp "N câu hỏi đang chờ · M câu bạn đã nhận · lâu nhất {Age}", `Href` `/inbox?course=…`; xuất hiện ≤ 5 s sau khi ticket tạo; biến ≤ 60 s sau khi xử lý hết; `CLAIMED` bởi người khác không tính vào "đang chờ" của mình; `$SVA` không thấy mục này

### TC-P4-03-38 [AC13] pre: `playwright-cli`
STEP: `/inbox` lúc tải (chậm mạng 3G); rỗng (lớp 2: "Không có câu hỏi nào đang chờ." + nút `Xem tất cả`); lỗi (`page.route` 500: nói chuyện gì xảy ra + nút `Thử lại`); chuyển `OPEN` → `ANSWERED` → `ALL`
EXPECT: khung xương khi tải; rỗng đúng chữ; lỗi đúng; chuyển trạng thái không nhảy layout (đo CLS ≈ 0); trạng thái có chữ, không chỉ màu

### TC-P4-03-40 [AC15] pre: repo
STEP: `grep -rn "@/mock" frontend/src/features/inbox \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint`
EXPECT: `0`; xanh. Chế độ demo của `/chat`, `/threads` vẫn chạy (mở `/chat`, `/threads`)

### TC-P4-03-41 [AC16] pre: `STAFF_ANSWER` mid `$SM`
STEP: `POST /chat/sessions/$S/messages/$SM/feedback -H "$SVA"`; sinh lại/thử lại (`attempt`) trên `$SM`
EXPECT: `422 MESSAGE_NOT_RATEABLE`, không ghi gì

### TC-P4-03-42 [UI] pre: frontend
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW a11y.spec.ts -g inbox`
EXPECT: `0`; 0 vi phạm nghiêm trọng

### TC-P4-04-04 [AC1 (không đôi)] pre: –
STEP: đẩy lại dòng outbox (`update outbox set status='PENDING'`)
EXPECT: **1** thư (`dedupe_key = escalation_answered:<answer_message_id>`); `mp_count` = 1

### TC-P4-04-06 [AC2 (không rò chéo)] pre: `$SVA` (`CANARY-A1` ở câu hỏi), `$SVB` (`CANARY-B2`); cả hai ticket được trả lời
STEP: lấy thư của A và B từ Mailpit (`/api/v1/message/{id}`); `grep -c CANARY-B2` thư A và ngược lại
EXPECT: thư A chỉ có `CANARY-A1`; thư B chỉ có `CANARY-B2`; 0 chéo

### TC-P4-04-08 [AC2 (thư nhắc nhóm không có tên / nội dung)] pre: ticket quá 24 giờ (xem TC-US-P4-05)
STEP: đọc thư `ticket_reminder`
EXPECT: chỉ số lượng, mã lớp, liên kết `/inbox?course=…`; 0 tên, 0 nội dung, 0 MSSV

### TC-P4-04-09 [AC3 (giết worker, điểm 1: trước SMTP)] pre: ticket có dòng `mail.send` `PENDING`
STEP: `docker compose kill -s KILL worker` ngay khi dòng ở `PROCESSING` (script: vòng lặp `psql` tới khi `status='PROCESSING'` rồi kill); `docker compose up -d worker`
EXPECT: sau khi worker lên: `mp_count '<id>@domain'` = 1; `status='SENT'`; không ticket thiếu thư

### TC-P4-04-10 [AC3 (điểm 2: sau SMTP, trước `SENT`)] pre: như trên; làm chậm bước ghi `SENT` (chặn bằng `pg_sleep` trigger tạm trên `mail_outbox` update)
STEP: kill giữa lúc Mailpit đã nhận
EXPECT: sau khởi động lại: Mailpit có đúng **1** thư mang `Message-ID` của dòng (`MP_IGNORE_DUPLICATE_IDS=true`); `status='SENT'`. Gỡ trigger

### TC-P4-04-11 [AC3 (`Message-ID` tất định)] pre: –
STEP: `curl $MP/api/v1/messages \| jq -r '.messages[].MessageID'` so `mail_outbox.id`
EXPECT: dạng `<{mail_outbox.id}@{domain của MAIL_FROM}>`

### TC-P4-04-12 [AC3 (ticket + outbox cùng giao dịch)] pre: –
STEP: tạo ticket rồi kill gateway giữa chừng (nhiều lần, 20)
EXPECT: không có ticket nào thiếu dòng outbox `escalation.created`; không dòng outbox mồ côi

### TC-P4-04-13 [AC4 (`ticket_reminder`)] pre: ticket `OPEN` quá `TICKET_REMIND_AFTER`
STEP: chờ job SLA (TC-US-P4-05)
EXPECT: thư tới TA + TEACHER `ACTIVE` của lớp; tiêu đề "Có câu hỏi chờ giảng viên quá 24 giờ làm việc"; thân: số câu, mã lớp, `/inbox?course=…`

### TC-P4-04-14 [AC4 (`thread_reply`)] pre: `$SVA` đăng thread; `$SVB` bình luận
STEP: xem Mailpit
EXPECT: thư tới `$SVA`, tiêu đề "Câu hỏi của bạn trên Threads có trả lời mới"; thân: tiêu đề thread + liên kết; cũng khi câu trả lời AI `VERIFY` / `CORRECT`

### TC-P4-04-15 [AC4 (`ticket_new`, cờ bật)] pre: teacher bật `notify_ticket_by_mail` (`PUT /me/settings`)
STEP: 3 ticket trong 5 phút ở lớp 1
EXPECT: **1** thư `ticket_new` (gộp); ghi số ticket đang chờ; không nội dung

### TC-P4-04-16 [AC4 (cửa sổ trượt 15 phút)] pre: đồng hồ giả / chờ thật
STEP: ticket T0, T0+14 m 59 s, T0+15 m 01 s (đồng hồ giả hoặc đặt TTL khoá `ep:mail:ticket_new:{uid}:{course}` bằng `$RDS`)
EXPECT: 14 m 59 s → **1** thư tổng; ticket ở 15 m 01 s → thư thứ hai; ticket 10:00 và 10:16 → 2 thư

### TC-P4-04-17 [AC4 (Redis lỗi thì vẫn gửi)] pre: `docker compose stop redis` rồi tạo ticket (nếu outbox còn chạy) hoặc chặn riêng khoá
STEP: ticket mới
EXPECT: thư vẫn gửi (không bỏ)

### TC-P4-04-18 [AC4 (tiêu đề không có tên)] pre: –
STEP: mọi thư test
EXPECT: tiêu đề không chứa họ tên (grep tên seed = 0)

### TC-P4-04-19 [AC5 (thử 3 lần rồi DEAD)] pre: Mailpit dừng (`docker compose stop mailpit`) hoặc `SMTP_HOST` sai
STEP: trả lời ticket; theo dõi `select attempts,status,last_error,updated_at from mail_outbox`
EXPECT: thử lại theo 1 s, 5 s, 30 s (3 lần thử lại, tổng 4 lần gọi); sau đó `status='DEAD'`; `last_error` là nhãn lớp lỗi (grep địa chỉ email / nội dung = 0); tin outbox vào dead-letter

### TC-P4-04-20 [AC5 (ứng dụng không bị ảnh hưởng)] pre: như trên
STEP: trong khi thử: `$SVA` mở `/chat`; chuông
EXPECT: ticket vẫn `ANSWERED`, tin trả lời vẫn hiện, chuông vẫn đến

### TC-P4-04-21 [AC5 (lỗi vĩnh viễn)] pre: hộp thư người nhận không tồn tại (SMTP giả trả `550`; dùng cổng tạm trả 550) hoặc địa chỉ bị từ chối
STEP: gửi
EXPECT: `DEAD` ngay, `attempts=1`, không thử lại

### TC-P4-04-22 [AC6 (cờ sinh viên)] pre: `$SVA` `notify_answer_by_mail=false`
STEP: trả lời ticket; thread có bình luận
EXPECT: **không** thư `escalation_answered` / `thread_reply`; chuông vẫn có

### TC-P4-04-23 [AC6 (mặc định ticket_new tắt)] pre: teacher mới chưa đổi cài đặt
STEP: tạo ticket
EXPECT: không thư `ticket_new`

### TC-P4-04-24 [AC6 (địa chỉ nhận)] pre: ticket của `$SVA`
STEP: xem `To` của thư
EXPECT: đúng email **đã xác minh** của chủ ticket; thử client truyền `to` trong body `answer` → bỏ qua

### TC-P4-04-25 [AC6 (`ticket_reminder` đúng người)] pre: lớp 1 có teacher, TA, ADMIN, teacher lớp khác
STEP: job nhắc
EXPECT: thư chỉ TA + TEACHER `ACTIVE` của lớp 1; không ADMIN, không teacher lớp khác

### TC-P4-04-26 [AC7 (cài đặt UI)] pre: `playwright-cli`
STEP: `/settings` bằng teacher; bằng TA; bằng STUDENT
EXPECT: TA/TEACHER thấy "Mail báo ticket mới" (tắt mặc định); STUDENT thấy "Mail khi giảng viên trả lời" (bật mặc định) và không thấy công tắc ticket; không hiện công tắc không áp dụng

### TC-P4-04-27 [AC7 (lưu lạc quan)] pre: –
STEP: bật/tắt công tắc; theo dõi `network`
EXPECT: giao diện đổi ngay, không toast; `PUT /me/settings` có `version`

### TC-P4-04-30 [AC8 (log)] pre: sau mọi TC mail
STEP: `docker compose logs gateway worker \| grep -ci '@edupilot.local'`; so tên template
EXPECT: log chỉ có `mail_id`, `template`, lớp lỗi; 0 email người nhận (trừ chỗ cấu hình khởi động, ghi lại nếu có)

### TC-P4-04-33 [AC9 (rỗng)] pre: không có thư lỗi
STEP: mở mục
EXPECT: "Không có thư lỗi." không nút

### TC-P4-04-34 [AC9 (chỉ mục)] pre: `mail_outbox` 100.000 dòng, 500 `DEAD` (gieo `generate_series`)
STEP: `EXPLAIN` truy vấn trang đầu (hoặc `j …/failed-mail` với `log_statement`)
EXPECT: dùng `mail_outbox_dead_idx`; không `Seq Scan on mail_outbox`; trang theo con trỏ

### TC-P4-04-35 [AC10 (payload hỏng)] pre: chèn dòng `mail_outbox` thiếu `ticket_id` (`$PSQL insert`)
STEP: chờ worker
EXPECT: `DEAD` ngay, lớp lỗi `payload_invalid`; không thư rỗng (Mailpit không thêm)

### TC-P4-04-36 [AC10 (ticket đã xoá)] pre: dòng `mail_outbox` trỏ ticket đã xoá
STEP: chờ worker
EXPECT: `DEAD` ngay, `ticket_missing`; không thử lại
