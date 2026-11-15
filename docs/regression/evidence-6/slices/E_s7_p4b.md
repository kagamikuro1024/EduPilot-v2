# Slice E_s7_p4b: 80 ca còn N

### TC-P4-05-02 [AC1 (giảng viên thấy đúng tin này)] pre: sau TC-01
STEP: `GET $Q/tickets/$T -H "$TCH"`
EXPECT: `transcript` có tin `ESCALATION_FOLLOWUP`; không tin khác của phiên (grep canary tin lân cận = 0)

### TC-P4-05-05 [AC1 (trạng thái)] pre: ticket `OPEN`/`CLAIMED`
STEP: `followup`
EXPECT: `409 TICKET_STATE_INVALID`

### TC-P4-05-06 [AC1 (quá 30 ngày)] pre: `CLOSED` với `closed_at` 31 ngày trước (`update`)
STEP: `followup`
EXPECT: `409 TICKET_REOPEN_EXPIRED`; ở 29 ngày → `200`

### TC-P4-05-07 [AC1 (không quá hạn tức thì)] pre: ticket tạo 10 ngày trước, `ANSWERED`; `followup`
STEP: chờ một tick `ticket.sla`; `select sla_stage`
EXPECT: vẫn `0` (đồng hồ từ `waiting_since`, không `created_at`); `TICKET` ở Hôm nay `Overdue=false`

### TC-P4-05-11 [AC2 (UI)] pre: `playwright-cli`, `/chat` của `$SVA`
STEP: bấm `Đã rõ`
EXPECT: cập nhật lạc quan, dòng tĩnh "Đã đóng · Hoàn tác" 5 giây; **không** hộp thoại xác nhận, không toast

### TC-P4-05-13 [AC3 (tự đóng 7 ngày)] pre: `ANSWERED` có `answered_at` 8 ngày trước (`update`)
STEP: chờ job `ticket.autoclose`
EXPECT: `CLOSED`, `closed_reason='AUTO_7D'`; chạy hai lần = một lần (không đổi `closed_at` lần hai); `OPEN`/`CLAIMED` quá 7 ngày không đóng

### TC-P4-05-14 [AC3 (câu trả lời mới đặt lại mốc)] pre: `ANSWERED` cách 6 ngày; teacher trả lời thêm
STEP: chờ job
EXPECT: không đóng (mốc 7 ngày tính từ lần trả lời mới)

### TC-P4-05-15 [AC3 (sau khi đóng)] pre: `CLOSED` 'AUTO_7D'
STEP: `$SVA` mở `/chat`
EXPECT: thấy "Đã đóng"; `Hỏi tiếp` được trong 30 ngày

### TC-P4-05-16 [AC4 (hàm giờ làm việc — hộp đen bằng hành vi)] pre: không có API trực tiếp; bảng SRS 9.3
STEP: chạy `cd backend-go && go test ./internal/escalation -run 'TestWorkingElapsedTable\|TestReminderAtTable\|TestEscalateAtTable\|TestWorkingHoursParametrised\|TestWorkingHoursProperty' -v` và đếm bảng; **QC tự kiểm thêm bằng stack**: ticket `waiting_since` = T2 08:00 (giờ VN), xem lúc job nhắc (TC-18); T6 12:00; T7 10:00
EXPECT: `ok`; ≥ 14 dòng bảng khung mặc định + 5 dòng khung 08:00–17:00; kết quả thực khớp bảng: T2 08:00 → nhắc T3 08:00; T6 12:00 → nhắc T2 12:00 (không nhắc T7/CN); `from>to` = 0

### TC-P4-05-17 [AC4 (dòng bảng)] pre: –
STEP: in từng dòng SRS 9.3: T2 09→T2 17 = 8 h; T6 16→T2 09 = 17 h; T7 10→CN 10 = 0; T6 23→T2 01 = 2 h; T2 07→T2 08 = 1 h; T2 08→T3 08 = 24 h; T2 17→T3 08 = 15 h; T4 12→T4 12 = 0; T5 08→T3 (tuần sau) 17 = 81 h
EXPECT: đúng 9 dòng khớp

### TC-P4-05-18 [AC5 (nhắc ở 24 giờ làm việc)] pre: ticket `OPEN` có `waiting_since` = 25 giờ làm việc trước (T3 09:00 nếu giờ hiện tại T4 10:00, tránh cuối tuần); `sla_stage=0`
STEP: chờ tick; `select sla_stage`; thông báo `TICKET_STALE` của teacher và TA; Mailpit
EXPECT: `sla_stage=1`; một thông báo `TICKET_STALE` gộp theo lớp "N câu hỏi chờ quá 24 giờ làm việc"; một thư `ticket_reminder` tới mỗi TA / TEACHER `ACTIVE` của lớp; chạy lại job → không thêm

### TC-P4-05-19 [AC5 (cuối tuần)] pre: ticket `waiting_since` = T6 12:00; giờ hiện tại CN
STEP: chờ tick
EXPECT: không nhắc ở T7/CN; sẽ nhắc T2 12:00 (nếu đang T2 sau 12:00: nhắc)

### TC-P4-05-20 [AC5 (hai bản worker song song)] pre: `docker compose up --scale worker=2`
STEP: chờ tick
EXPECT: `sla_stage` tăng một lần; thư `ticket_reminder` đúng 1 cho mỗi người nhận

### TC-P4-05-21 [AC5 (CLAIMED quá hạn)] pre: ticket `CLAIMED` bởi teacher, quá hạn
STEP: tick
EXPECT: chỉ **teacher** nhận `TICKET_STALE` cá nhân, không thư, không nhóm; ticket `Bỏ nhận` rồi nhận lại → nhắc lại lần nữa (`dedupe_key` theo `claimed_at`)

### TC-P4-05-22 [AC5 (lọc dùng `waiting_since`)] pre: hai ticket: `created_at` cũ nhưng `waiting_since` mới (sau followup); và ngược lại
STEP: tick
EXPECT: chỉ ticket có `waiting_since` quá hạn được nhắc

### TC-P4-05-23 [AC6 (72 giờ)] pre: ticket `OPEN` với 73 giờ làm việc
STEP: tick; `GET $GW/api/v1/me/today -H "$TCH"`
EXPECT: `sla_stage=2`; mục `TICKET` của lớp `Overdue=true` lên **trước mọi bậc** ở "Hôm nay"; `Reason` ghi số giờ ("… · lâu nhất 73 giờ")

### TC-P4-05-24 [AC6 (báo Admin khi không ai hoạt động)] pre: lớp có ticket `sla_stage=2`; không TA/TEACHER nào `auth_sessions.last_used_at` trong 72 giờ lịch (`update auth_sessions`) và không ai thao tác ticket
STEP: tick
EXPECT: ADMIN nhận **một** `COURSE_TICKETS_STALLED` "Lớp 761987 có N câu hỏi chờ quá 72 giờ làm việc" + mục `Hôm nay` của ADMIN (bậc 45); chỉ mã lớp và số, **không** nội dung ticket (grep canary = 0)

### TC-P4-05-25 [AC6 (có người hoạt động)] pre: một teacher có `last_used_at` trong 72 giờ
STEP: tick
EXPECT: **không** báo ADMIN

### TC-P4-05-26 [AC6 (mỗi ngày một lần)] pre: lớp không hoạt động
STEP: chạy tick nhiều lần trong ngày; rồi lùi `now` một ngày (`seed-age` hoặc `update notifications.created_at` + chờ)
EXPECT: 1 thông báo/ngày (`dedupe_key = stalled:<course>:<yyyymmdd>`); ba ngày liên tiếp → 3 thông báo, mỗi ngày 1

### TC-P4-05-27 [AC7 (chữ trạng thái)] pre: 4 ticket ở 4 trạng thái của `$SVA`
STEP: `playwright-cli` `/chat`, đọc chữ từng trạng thái; `eval innerText`
EXPECT: `OPEN` "Đang chờ giảng viên · đã gửi {Age} trước" ("vừa gửi" < 1 phút); `CLAIMED` "Giảng viên đã nhận · đã chờ {Age}"; `ANSWERED` nhãn người trả lời + `Hỏi tiếp`, `Đã rõ`; `CLOSED` "Đã đóng" + `Hỏi tiếp` (≤ 30 ngày); **không** chữ "ticket", "SLA", tên người nhận, số độ tin cậy

### TC-P4-05-28 [AC7 (cập nhật mỗi phút, không nhảy layout)] pre: –
STEP: giữ trang mở 2 phút; đo `getBoundingClientRect` trước/sau
EXPECT: `Age` đổi; vị trí khối không đổi (CLS ≈ 0)

### TC-P4-05-30 [AC8 (đọc khi mở)] pre: –
STEP: `$SVA` mở `/chat?session=…` có tin `STAFF_ANSWER`; `network` thấy `POST /notifications/read {link}`; `GET /chat/sessions/$S/messages` đơn lẻ
EXPECT: mục biến ≤ 60 s; `GET …/messages` không có tác dụng ghi (`select updated_at` thông báo không đổi nếu chỉ GET)

### TC-P4-05-33 [AC9 (hỏi tiếp sau dismiss)] pre: ba ticket `CLOSED`
STEP: `$SVA` `Hỏi tiếp` mỗi ticket
EXPECT: `200` (cả ba, ≤ 30 ngày)

### TC-P4-05-34 [AC9 (người khác đã nhận)] pre: ticket `CLAIMED` bởi TA
STEP: teacher `dismiss`
EXPECT: `409` (kèm người nhận); `reason` ngoài enum → `422`

### TC-P4-05-37 [AC10 (câu hỏi cá nhân)] pre: ticket từ câu "Điểm giữa kỳ của em là bao nhiêu?" (`PERSONAL_QUESTION`)
STEP: preview / `POST knowledge`
EXPECT: `422 KNOWLEDGE_PERSONAL`

### TC-P4-05-40 [AC10 (embed bản đã sửa)] pre: provider giả ghi chuỗi `Embed`
STEP: sửa câu trả lời khác câu gốc rồi lưu
EXPECT: chuỗi được `Embed` là bản đã sửa (không phải câu gốc)

### TC-P4-05-41 [AC11 (dùng được ngay)] pre: ticket đã lưu tri thức
STEP: ≤ 10 s sau; `$SVB` hỏi câu tương tự (provider giả)
EXPECT: `content_chunks.embedding` có; câu trả lời của SV B có nguồn "Hỏi đáp đã được giảng viên xác nhận" (nguồn TICKET: không liên kết; POST: liên kết `/threads/{id}`); `ep:rag:ver` của lớp tăng (`$RDS GET`); cache câu trả lời vô hiệu

### TC-P4-05-42 [AC11 (chưa nạp)] pre: provider embedding chết (dừng giả)
STEP: lưu tri thức
EXPECT: xác nhận vẫn thành công; giảng viên thấy "Chưa nạp vào tri thức" + `Thử lại`; đoạn **không** lọt ra tìm kiếm (`rag` trả 0); `Thử lại` ghi lại outbox `knowledge.sync`; khi provider sống, nạp xong

### TC-P4-05-44 [AC11 (không lọt vào tài liệu)] pre: đoạn tri thức tồn tại
STEP: `GET /library`, danh sách tài liệu của giảng viên, thống kê tài liệu, `share-from`, chọn tài liệu cho "Hỏi AI về tài liệu", `/today`, ingest; `select * from documents where category='THREAD_KNOWLEDGE' …`
EXPECT: không route nào liệt kê tài liệu `THREAD_K…`; tạo tài liệu với `category` đó → `422`

### TC-P4-05-45 [AC11 (cô lập lớp)] pre: tri thức ở lớp 1
STEP: `$SVx` ở lớp 2 hỏi câu tương tự
EXPECT: không trả đoạn của lớp 1

### TC-P4-05-46 [AC12 (ngưỡng)] pre: `/class/settings` bằng `$TCH`
STEP: kéo thanh (0,50–0,95, bước 0,05); `PUT $Q/settings {"escalation_threshold":0.85,"version":N}`
EXPECT: `200`; ở tin kế tiếp của sinh viên có hiệu lực (đọc mỗi lần): ngưỡng 0,95 → câu mặc định tự mở ticket

### TC-P4-05-49 [AC12 (audit, UI)] pre: –
STEP: đổi ngưỡng; `select old,new from audit_log where …`; `playwright-cli` xem thanh
EXPECT: audit có giá trị cũ / mới; thanh có đầu "Ít chuyển", cuối "Chuyển nhiều", không câu mô tả khác

### TC-P4-05-50 [AC13 (worker chết giữa `ticket.sla`)] pre: ticket quá hạn
STEP: `docker compose kill -s KILL worker` ngay lúc job chạy (script theo `sla_stage`/log); khởi động lại
EXPECT: lần chạy sau hoàn tất; mỗi ticket nhắc đúng một lần (thư/thông báo đếm bằng `select count(*)`); không có "đã gửi mà chưa ghi": không dòng outbox/thông báo nào sinh khi `sla_stage` còn 0

### TC-P4-06-08 [AC1 (hạn mức)] pre: `$SVB` báo 20 bài trong 1 giờ (20 thread khác nhau)
STEP: báo thêm bài thứ 21
EXPECT: `429 RATE_LIMITED` (`REPORT_RATE_PER_HOUR=20`)

### TC-P4-06-09 [AC1 (bình luận)] pre: `$P`
STEP: `$SVA` `POST $Q/threads/$TH/posts/$P/report {"reason":"OFFENSIVE"}`
EXPECT: `201`

### TC-P4-06-12 [AC2 (chủ bài, giảng viên)] pre: –
STEP: `$SVA` mở bài; `$TCH` mở hàng chờ
EXPECT: chủ bài thấy bài với dải "Bài của bạn đang được giảng viên xem xét"; giảng viên thấy "Đã ẩn · chờ duyệt" trong hàng chờ; `POST_REPORTED` gộp cho TA + TEACHER

### TC-P4-06-13 [AC2 (đếm người khác nhau)] pre: `$SVB` báo 3 bài khác nhau; `$TH2` có 3 báo cáo từ **cùng** `$SVB` (không thể: 409) → thử ba báo cáo ở ba bài không đủ
STEP: xem `hidden_at`
EXPECT: `$TH2` có 1 báo cáo; không tự ẩn

### TC-P4-06-14 [AC2 (chỉ đếm `OPEN`)] pre: `$TH3` có 2 báo cáo `OPEN` + 1 `DISMISSED` (qua `dismiss-reports`)
STEP: người thứ 3 báo
EXPECT: **không** tự ẩn (2 `OPEN`)

### TC-P4-06-15 [AC2 (ngưỡng cấu hình)] pre: –
STEP: `grep -rn REPORT_AUTOHIDE_THRESHOLD docker-compose*.yml .env.example backend-go/internal/platform/config`
EXPECT: hằng cấu hình = 3

### TC-P4-06-20 [AC3 (audit)] pre: –
STEP: `select entity,action from audit_log` sau mỗi hành động
EXPECT: mỗi `hide`/`unhide`/`dismiss-reports` 1 dòng

### TC-P4-06-22 [AC4 (chủ bài được báo)] pre: `$TH` tự ẩn rồi giảng viên `Ẩn bài` với lý do "Lạc đề"
STEP: `GET /notifications -H "$SVA"`; mở bài
EXPECT: tự ẩn: `POST_HIDDEN` "Bài của bạn đang được giảng viên xem xét"; giảng viên ẩn: "Bài của bạn đã bị ẩn: Lạc đề"; mở bài thấy dải lý do; **không** lộ ai báo cáo; `Bỏ ẩn`/`Giữ bài` → "Bài của bạn đã hiện lại"; Mailpit **không** có thư cho việc này

### TC-P4-06-25 [AC5 (canary PII không lưu)] pre: canary PII `CANARY-PII-7Q` (số điện thoại tổng hợp) trong lần sửa bị chặn
STEP: quét mọi bảng `forum_*`, `pii_events`, `audit_log`, `outbox` (`pg_dump \| grep -c`)
EXPECT: 0 chỗ chứa canary thô (có thể có bản đã che trong `pii_events` nếu spec cho; không bản thô)

### TC-P4-06-26 [AC5 (khoá khi đã `VERIFIED`/`CORRECTED`)] pre: thread có bài AI `VERIFIED`
STEP: chủ sửa
EXPECT: `409 EDIT_LOCKED_VERIFIED`

### TC-P4-06-28 [AC5 (sửa bài đang ẩn)] pre: thread ẩn
STEP: chủ sửa
EXPECT: `200`; `hidden_at` vẫn có (không tự bỏ ẩn)

### TC-P4-06-29 [AC5 (AI `PENDING`)] pre: thread có bài AI `PENDING`
STEP: chủ sửa câu hỏi
EXPECT: bài AI cũ `REJECTED`, `ai_state='PENDING'` (AI sinh lại), **không** thông báo cho chủ bài; sinh viên không thấy bài AI cũ; `AI_PENDING` đếm lại

### TC-P4-06-30 [AC5 (khủng hoảng ở `PUT`)] pre: –
STEP: sửa thân thành "em không muốn sống nữa"
EXPECT: `422 CRISIS_SUPPORT`; không lưu

### TC-P4-06-33 [AC6 (gỡ tri thức)] pre: thread có tri thức đã nạp
STEP: xoá thread
EXPECT: đoạn `content_chunks` tương ứng bị gỡ ≤ 5 s

### TC-P4-06-34 [AC6 (UI xác nhận)] pre: `playwright-cli`
STEP: bấm Xoá
EXPECT: `<ConfirmIrreversible>` ghi "Câu hỏi và N bình luận sẽ bị xoá"

### TC-P4-06-36 [AC7 (danh sách)] pre: 3 ghim
STEP: `GET $Q/threads` trang đầu và trang kế
EXPECT: `pinned:[…]` ≤ 3, `items` không lặp ghim; lọc `week`, `tag`, `status=pending\|verified\|none` giữ nguyên

### TC-P4-06-38 [AC8 (người đăng được báo)] pre: `$SVA` đăng thread; `$SVB` bình luận
STEP: thông báo + Mailpit
EXPECT: `THREAD_REPLY` qua chuông tức thì và thư `thread_reply` (nếu `notify_answer_by_mail`); `$SVA` tự bình luận không tự báo; `VERIFY`/`CORRECT` → `THREAD_VERIFIED`; `REJECT` **không** báo chủ bài

### TC-P4-06-39 [AC9 (gộp `AI_PENDING`)] pre: 3 bài AI `PENDING`
STEP: `GET /notifications -H "$TCH"`
EXPECT: 1 dòng "3 câu trả lời AI chờ duyệt" (N = số bài `PENDING` chưa ẩn), cập nhật tại chỗ; N=0 → tự đọc

### TC-P4-06-40 [AC10 (`Hôm nay`)] pre: có báo cáo `OPEN`
STEP: `GET /me/today -H "$TCH"`; `-H "$SVA"`; `-H "$ADM"`
EXPECT: giảng viên mỗi lớp 1 mục "N bài bị báo cáo chờ xem", `Href` `/threads?view=moderation&course=…`; biến ≤ 60 s sau khi xử lý hết; sinh viên / ADMIN không thấy

### TC-P4-06-41 [AC11 (`Loại` biến mất, gỡ khỏi RAG)] pre: bài AI `VERIFIED` đã nạp tri thức → `REJECT` (hoặc ẩn / xoá)
STEP: `$SVB` hỏi câu tương tự ≤ 5 s sau; `select count(*) from content_chunks where …`
EXPECT: sinh viên không thấy ở endpoint nào; đoạn gỡ ≤ 5 s; `rag` không trả đoạn đó; cache câu trả lời cũ vô hiệu (`ep:rag:ver` tăng)

### TC-P4-06-42 [AC12 (nạp lại)] pre: giảng viên `VERIFY` bài AI `PENDING`
STEP: `select * from knowledge_items, content_chunks`; chờ ≤ 10 s
EXPECT: `knowledge.sync` `Embed` ≤ 10 s; đoạn gồm câu hỏi (đã `Redact`) + câu trả lời; `rag.SearchStudent` trả đoạn với nguồn "Hỏi đáp đã được …" (liên kết `/threads/{id}`)

### TC-P4-06-43 [AC12 (`CORRECT` hai lần thay đoạn)] pre: –
STEP: `CORRECT` hai lần
EXPECT: `knowledge_items` vẫn 1; đoạn thay, không đôi; bản sửa là chuỗi được `Embed`

### TC-P4-06-44 [AC12 (embedding chết)] pre: dừng provider embedding
STEP: `VERIFY`
EXPECT: `VERIFY` vẫn thành công; `AI_PENDING` đếm lại; `today.invalidate` đã chạy (Hôm nay đổi ≤ 60 s)

### TC-P4-06-45 [AC12 (cô lập lớp)] pre: –
STEP: SV lớp 2 hỏi câu tương tự
EXPECT: không nhận đoạn của lớp 1

### TC-P4-06-47 [AC14 (UI)] pre: `playwright-cli` 1280 và 375
STEP: menu `…` của bài: `Báo cáo` là mục menu (không phải nút phụ thứ ba); chọn lý do bằng danh sách; sau báo cáo hiện "Đã báo cáo · Hoàn tác" 5 s; giảng viên thấy `Ẩn bài`/`Ghim`/`Giữ bài`; chủ bài thấy `Sửa`, `Xoá`; `<PageState>`; chữ "Loại khỏi tri thức" (giảng viên) / "Không hiển thị" (sinh viên)
EXPECT: đúng; dùng được ở 375 px (không tràn ngang, vùng chạm ≥ 44 px)

### TC-P4-06-48 [AC14 (hoàn tác báo cáo)] pre: vừa báo cáo
STEP: bấm Hoàn tác < 5 s (`DELETE …/report`); sau 5 s; sau khi tự ẩn
EXPECT: trong 5 s → rút báo cáo; sau 5 s hoặc đã tự ẩn → không hoàn tác

### TC-P4-06-49 [AC15 (đua báo cáo thứ 3)] pre: 2 báo cáo; hai người cùng báo cùng lúc
STEP: `&`+`wait`
EXPECT: đúng **một** lần tự ẩn; đúng một thông báo cho chủ bài (`dedupe_key`)

### TC-P4-06-50 [AC15 (gửi lại sau mất mạng)] pre: –
STEP: `setOffline`, báo, online, gửi lại cùng người
EXPECT: `409 ALREADY_REPORTED` hiện "Bạn đã báo cáo bài này" (không phải lỗi)

### TC-P4-06-51 [AC15 (xoá giữa chừng)] pre: tác giả xoá bài vừa trước khi giảng viên `Ẩn bài`
STEP: `hide`
EXPECT: `404` "Bài này không còn nữa"

### TC-P4-06-52 [AC16 (bài ẩn không rò)] pre: thread ẩn chứa canary `CANARY-HID-1`
STEP: `$SVB`: `GET $Q/threads?q=CANARY-HID-1`, `…/similar`, `/me/today`, số liệu lớp (`GET $Q/insights` nếu có), `AI_PENDING`; grep canary
EXPECT: 0 chỗ lộ; chỉ chủ bài và hàng chờ Staff thấy

### TC-P4-06-53 [UI] pre: frontend
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW threads.spec.ts -g 'report undo\|owner edit delete\|staff hide pin'`; `$PW a11y.spec.ts -g threads`
EXPECT: `0`; xanh

### TC-P4-07-03 [AC1 (ngôi)] pre: –
STEP: gửi "em tu tu cho xong", "em doc tu tu", "join tu hai bang", "chuyen tu hai phan bu sang thap phan" (không dấu)
EXPECT: câu 1 → `CRISIS`; ba câu sau → không

### TC-P4-07-04 [AC1 (CRISIS thắng)] pre: –
STEP: gửi (18) "Cho em hỏi điểm giữa kỳ. Mà em muốn chết thật rồi" và "em muốn chết, điểm của bạn Kha là bao nhiêu"
EXPECT: `CRISIS` (thắng `PERSONAL_*`, `OTHER_PERSON`)

### TC-P4-07-05 [AC1 (test sprint 6 giữ nguyên)] pre: repo
STEP: `cd backend-go && go test ./internal/agent -run 'TestQCCrisisSet\|TestCrisisCannedReply' -v`
EXPECT: `ok`; `git diff 1518f83.. -- backend-go/internal/agent/*_test.go` không sửa hai test này

### TC-P4-07-10 [AC3 (không số liệu danh tính)] pre: –
STEP: mọi route số liệu hướng giảng viên (`GET $Q/insights…`, Hôm nay, hồ sơ nếu có)
EXPECT: không nêu từng sinh viên có tin khủng hoảng; (đếm không danh tính nếu có)

### TC-P4-07-11 [AC3 (intent chỉ chủ phiên đọc)] pre: tin khủng hoảng của `$SVA`
STEP: `GET /chat/sessions/$S/messages` bằng `$SVA` (thấy); `$TCH`, `$TA_`, `$ADM`, `$SVB`
EXPECT: `$SVA` thấy; những người còn lại `403`/`404`

### TC-P4-07-15 [AC4 (UI giữ nháp)] pre: `playwright-cli` `/threads/new`
STEP: gõ câu khủng hoảng, bấm Đăng
EXPECT: hộp thoại bảo vệ: lối chính `Mở chat riêng` (mang bản nháp sang chat riêng), `Quay lại sửa`; chữ đang gõ không mất; debounce 800 ms (đếm request `precheck`); bản nháp **không** có ở máy chủ (`grep` DB = 0)

### TC-P4-07-16 [AC4 (sửa `PUT`)] pre: thread của `$SVA`
STEP: `PUT` thân khủng hoảng
EXPECT: `422 CRISIS_SUPPORT` (xem TC-P4-06-30)

### TC-P4-07-17 [AC5 (giờ thi)] pre: `$SVA` đang trong giờ thi (khoá chat)
STEP: gửi câu dương (5), (2)
EXPECT: vẫn nhận câu mẫu hỗ trợ (không `EXAM_LOCKED`); `RecordChatBlocked` không đếm (`select count(*) from integrity_events …` hoặc nơi liêm chính ghi, không tăng)

### TC-P4-07-18 [AC5 (câu thường vẫn khoá)] pre: cùng lúc
STEP: gửi câu hỏi thường
EXPECT: `EXAM_LOCKED` (đường vòng không mở)

### TC-P4-07-19 [AC5 (PENDING/ngoài lớp)] pre: sinh viên `PENDING` / ngoài lớp
STEP: gửi câu dương
EXPECT: không có chat (403) → không phát sinh gì (không dòng mới ở mọi bảng)

### TC-P4-07-20 [AC6 (cấu hình)] pre: `docker-compose.local.yml`
STEP: `grep -n SUPPORT_RESOURCES_VI docker-compose.local.yml`; gửi câu dương; rồi Threads `CRISIS_SUPPORT`
EXPECT: có biến; câu mẫu của chat riêng và Threads chứa đúng chuỗi

### TC-P4-07-21 [AC6 (rỗng)] pre: đặt `SUPPORT_RESOURCES_VI=""` rồi `up` lại
STEP: gửi câu dương
EXPECT: câu mặc định `DefaultSupport`

### TC-P4-07-22 [AC6 (> 300 ký tự)] pre: `SUPPORT_RESOURCES_VI` 301 ký tự
STEP: `docker compose up gateway`
EXPECT: gateway **không khởi động**, lỗi nêu tên biến `SUPPORT_RESOURCES_VI`; 300 ký tự → khởi động

### TC-P4-07-23 [AC6 (không PII)] pre: `SUPPORT_RESOURCES_VI` có số phòng "101"
STEP: gửi câu dương
EXPECT: câu mẫu nguyên chuỗi (không `[đã ẩn]`, không bị `Mask`)

### TC-P4-07-24 [AC7 (giới hạn ghi thẳng)] pre: repo
STEP: `grep -n -i 'mệt quá rồi, muốn ngủ mãi\|deadline này làm em muốn chết\|không thay chuyên gia\|thay thế chuyên gia' docs/specs/FEAT-escalation-notify/SRS.md docs/thesis-notes/*`; `grep -n Q-crisis-hyperbole docs/specs/FEAT-escalation-notify/QUESTIONS.md`
EXPECT: có đủ ba ý ở SRS 4.9 và `docs/thesis-notes/`; `Q-crisis-hyperbole` có

### TC-P4-07-27 [UI] pre: frontend
STEP: `bash scripts/ui-antipatterns.sh; echo $?`
EXPECT: `0`
