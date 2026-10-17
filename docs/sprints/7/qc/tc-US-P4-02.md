# QC test case — US-P4-02 (chuông tức thời, gộp, hai bản gateway, đọc bù)
Nguồn: `US.md` US-P4-02 AC1…AC11 + `SRS.md` 2, 3.3, 4.4, 6.3, 7.3; `proposals.md` #1 (dùng lại `GET /api/v1/events`, không dựng đường SSE thứ hai); `SYSTEM_DESIGN.md` SLO "Chuông ≤ 5 s". Hộp đen, không đọc code dev.
Biến shell và quy ước như `tc-US-P4-01.md`. Thêm: `EV='curl -sNk -H "Accept: text/event-stream"'`; `sse <token> [Last-Event-ID]` = `$EV -H "$tok" [-H "Last-Event-ID: $id"] $GW/api/v1/events` ghi ra tệp có dấu thời gian mỗi khung (`ts` bằng `date +%s%3N`).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P4-02-01 | AC1 | repo | `grep -rn 'InsertNotification\|insert into notifications' backend-go --include=*.go --include=*.sql \| grep -v 'internal/notify\|store/queries/notify.sql'` | 0 dòng (không nơi nào ngoài `notify` và `notify.sql`). Mọi gói `course`, `exam`, `thread`, `escalation`, `calendar` tạo thông báo qua `notify.Service` |
| TC-P4-02-02 | AC1 (gửi hàng loạt = 1 câu SQL) | lớp có 1.000 sinh viên (seed tạm bằng `generate_series`) | bật `log_statement='all'` của Postgres test; giáo viên đặt lịch thi (kích hoạt `ExamNotifyScheduled`); `grep -c 'INSERT INTO notifications' pg.log` | đúng **1** câu `INSERT … unnest` ghi cho 1.000 người; 1.000 dòng `notifications`; không 1.000 lượt upsert |
| TC-P4-02-03 | AC1 (test cũ giữ nguyên) | repo | `go test ./internal/course ./internal/exam ./internal/thread ./internal/calendar` | `ok`, không test cũ nào bị sửa (`git diff 1518f83.. -- '*_test.go'` chỉ liệt kê các test #8/#10) |
| TC-P4-02-04 | AC2 (đẩy tức thì) | `$TCH` mở `sse` (khung SSE đang chạy) và `$TA_` cũng | `$SVA` tạo ticket (`escalate`), ghi `t0` ngay khi `201` nhận; đọc khung `notification` ở tệp SSE của teacher và TA | khung `notification` có `{id,type:"TICKET_NEW",title,body,link,course_id,read:false,created_at,unread_count}`, đến ≤ 5 s sau `t0`; số chưa đọc đổi không cần tải lại trang |
| TC-P4-02-05 | AC2 (p95 trên 20 lần) | như trên, 20 sinh viên/ticket (seed hoặc `SVA`…) | lặp 20 lần, ghi `t_frame − t_commit` | p95 ≤ 5000 ms (in bảng 20 số) |
| TC-P4-02-06 | AC2 (giao diện) | `playwright-cli` hai ngữ cảnh: giảng viên mở `/`; sinh viên ở ngữ cảnh khác | sinh viên bấm `Nhờ giảng viên` | chuông của giảng viên nhảy số ≤ 5 s, không reload (`page.on('load')` đếm 0) |
| TC-P4-02-07 | AC3 (hai bản gateway) | `docker compose up --scale gateway=2` (hoặc hai cổng) | nối SSE vào bản A (cổng a), `POST escalate` trúng bản B (cổng b); sau đó đảo (nối B, sinh ở A) | cả hai chiều: người dùng nhận khung `notification` ≤ 5 s |
| TC-P4-02-08 | AC3 (gateway không giữ trạng thái) | như trên | dừng bản A khi SSE đang mở ở A; nối lại vào B với `Last-Event-ID` cuối | nhận đủ khung còn thiếu, không trùng |
| TC-P4-02-09 | AC4 (đọc bù bằng `Last-Event-ID`) | `$TCH` nối SSE, nhận khung id `E0`; ngắt kết nối 10 s; trong lúc đó sinh 3 ticket (3 sinh viên) | nối lại `Last-Event-ID: E0` | nhận đúng 3 sự kiện còn thiếu, đúng thứ tự, **không** trùng (so `id`) |
| TC-P4-02-10 | AC4 (bộ đệm quá hạn) | xoá bộ đệm Redis của người dùng (`$RDS DEL` khoá buffer `BufKey` — lấy tên từ `docker exec redis redis-cli --scan --pattern '*events*'`) | nối với `Last-Event-ID` cũ | khung `resync` với lý do `buffer_exceeded`; khách hàng gọi `GET /notifications` đọc bù **từ bảng**; số chưa đọc đúng so `select count(*) … read_at is null` |
| TC-P4-02-11 | AC4 (id sai) | – | nối với `Last-Event-ID: abc` | khung `resync invalid_last_event_id`, kết nối không lỗi (HTTP 200 stream) |
| TC-P4-02-12 | AC4 (giao diện) | `playwright-cli`, `context.setOffline(true)` 10 s rồi `false`, 3 ticket sinh trong lúc offline | quan sát chuông | sau nối lại, chuông hiện đủ 3 thông báo mới, không nhân đôi |
| TC-P4-02-13 | AC5 (gộp 5 ticket) | giảng viên chưa đọc; lớp 1 | 5 sinh viên khác nhau tạo ticket trong 1 phút; `GET /notifications -H "$TCH" \| jq '[.[]\|select(.type=="TICKET_NEW")]'` | đúng **một** dòng chưa đọc "5 câu hỏi đang chờ giảng viên"; `id` giữ nguyên qua 5 lần cập nhật; số trong tiêu đề = số ticket `OPEN` hiện tại của lớp |
| TC-P4-02-14 | AC5 (đọc xong, ticket mới) | sau TC-13 giảng viên đọc dòng | tạo ticket mới | dòng **mới** chưa đọc (id khác) |
| TC-P4-02-15 | AC5 (số = 0 tự đọc) | tất cả ticket của lớp đã xử lý | kiểm dòng | tự đánh dấu đã đọc (`read_at` có) |
| TC-P4-02-16 | AC5 (số giảm không đặt lại `read_at`) | dòng đã đọc, số 3; một ticket được nhận → 2 | đọc dòng + SSE | `read_at` giữ nguyên; không khung SSE mới nếu số không tăng |
| TC-P4-02-17 | AC5 (song song 50 lượt) | 2 handler song song (hai ticket cùng lúc, lặp 50 lần: tạo 100 ticket bằng 2 luồng `curl`) | so số trong tiêu đề cuối với `select count(*) from escalation_tickets where course_id and status='OPEN'` | bằng nhau |
| TC-P4-02-18 | AC5 (loại khác, cá nhân không gộp) | `AI_PENDING`, `POST_REPORTED`, thông báo cá nhân (`TICKET_ANSWERED`) | tạo 3 bài AI chờ duyệt, 3 báo cáo, 3 câu trả lời cho 3 sinh viên | `AI_PENDING` "3 câu trả lời AI chờ duyệt" một dòng; `POST_REPORTED` "3 bài bị báo cáo chờ xem" một dòng; thông báo cá nhân 3 dòng riêng |
| TC-P4-02-19 | AC6 (không lộ nội dung) | câu hỏi `CANARY-Q1` (SV `Nguyễn Văn Gioi`/MSSV), trả lời `CANARY-A1`; ticket mới, trả lời, và `TICKET_STALE` | `select title,body from notifications`; khung SSE; `select payload from outbox`; `grep -c 'CANARY-\|Gioi\|<mssv>'` | 0 ở `title`, `body`, khung SSE và `outbox.payload`; chỉ có số, mã lớp, liên kết |
| TC-P4-02-20 | AC7 (idempotent theo sự kiện) | sự kiện `escalation.created` | `kill -STOP` worker giữa xử lý rồi `restart`; hoặc đẩy lại dòng outbox (`update outbox set status='PENDING'`) | đúng 1 thông báo cho mỗi người nhận (`dedupe_key`), đúng 1 khung SSE; cập nhật gộp không phát lại khi số không đổi |
| TC-P4-02-21 | AC8 (chỉ của mình) | hai người dùng A (`$SVA`), B (`$SVB`) | A nối SSE; B có thông báo; `GET /notifications -H "$SVA"`; `POST /notifications/{idB}/read -H "$SVA"`; cũng bằng `$ADM`; thử `GET /events?user_id=…` | A không nhận khung của B (không có tham số danh tính); `404` ở thông báo của B (cả A và ADMIN) |
| TC-P4-02-22 | AC9 (tab thứ ba) | `$SVA` đã có 2 SSE (`SSE_MAX_PER_USER=2`) | mở kết nối thứ ba (curl) | `429 SSE_LIMIT_REACHED` |
| TC-P4-02-23 | AC9 (giao diện rơi về 30 s) | `playwright-cli` mở 3 tab của cùng người dùng | tab thứ ba: tạo ticket mới ở ngữ cảnh khác; quan sát | chuông tab 3 không báo lỗi; cập nhật trong ≤ 30 s (làm mới) |
| TC-P4-02-24 | AC9 (Redis chết) | `docker compose stop redis` | tạo thông báo (ticket / thread) | `select count(*) from notifications` có dòng mới; luồng nghiệp vụ không lỗi (HTTP 201/200); SSE `reconnect upstream_unavailable`; chuông ≤ 30 s; bật lại redis |
| TC-P4-02-25 | AC10 (chuông UI) | `playwright-cli` 1280 × 800, giảng viên có 12 chưa đọc | xem chuông | hiện "9+"; danh sách mới nhất trước; bấm một mục → mở đúng `link`; `link` rỗng → `/` ("Hôm nay"); `?course=` được tiêu thụ; đánh dấu đã đọc lạc quan; không toast |
| TC-P4-02-26 | AC10 (375 px, a11y) | `playwright-cli resize 375 812` | đo `getBoundingClientRect` của mục chuông và nút chuông; kiểm `aria-live="polite"` ở số chưa đọc | vùng chạm ≥ 44 px; không tràn ngang (`scrollWidth ≤ innerWidth`) |
| TC-P4-02-27 | AC10 (mở danh sách khi có thông báo mới) | mở danh sách, đang đọc mục thứ 3 | đẩy thông báo mới | chèn lên đầu; mục đang đọc không nhảy vị trí (đo `getBoundingClientRect().top` của mục trước/sau, trừ chiều cao mục mới nếu cuộn) |
| TC-P4-02-28 | AC11 (làm mới lỗi) | `page.route('**/notifications', abort)` | mở danh sách chuông | giữ danh sách cũ; không lỗi chặn; dòng "Chưa cập nhật được" nhạt chỉ khi mở; tự thử lại ở nhịp kế |
| TC-P4-02-29 | UI | frontend | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW a11y.spec.ts -g bell` | `0`; 0 vi phạm nghiêm trọng |

## Nhánh lỗi
TC-P4-02-10, 11, 12, 22, 23, 24, 28 (đọc bù, id sai, mất mạng, tab 3, Redis chết, làm mới lỗi). Thêm TC-P4-02-E1: gửi `POST /notifications/read {link:""}` → `422`; `{link}` của người khác → `200 {updated:0}` và thông báo người kia không đổi.

## Phân quyền
TC-P4-02-21 (A không thấy khung B, ADMIN không đọc của người khác). TC-P4-02-P1: sinh viên không nhận `TICKET_NEW`; ADMIN không nhận `TICKET_NEW` (chỉ `COURSE_TICKETS_STALLED`).

## Câu hỏi (Q-QC)
- **Q-QC-P4-05** Tên khoá Redis bộ đệm SSE và cách bật `log_statement` chưa ghi ở spec (hộp đen). Cần handoff của dev; nếu thiếu, TC-02 và TC-10 chạy bằng quan sát hành vi (đếm `notifications`, nối `Last-Event-ID` rất cũ).
- **Q-QC-P4-06** Thông báo cho 1.000 sinh viên qua `exam` (AC1): spec không nêu cách kích hoạt trên stack. Giả định: đặt lịch thi qua API exam có sẵn (sprint 5). Xác nhận.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
