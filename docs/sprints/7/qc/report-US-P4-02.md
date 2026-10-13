# QC report — US-P4-02 (chuông tức thời, gộp, hai bản gateway, một cửa)
Ngày 2026-10-13. Commit kiểm: `sprint/7-p4-p5` @ `24ef239` (mã `e2a7e68`; handoff `dev-US-P4-02.md`). Bộ TC: `tc-US-P4-02.md` (không sửa chữ TC). Stack như `report-US-P4-01.md` (`edupilot100`, gateway ×2 khi cần). **#12 ACCEPTED:** AC1 chấm theo thứ tự đã quyết (Threads giữ ghi đồng bộ qua `notify.SendTx`, chuyển outbox ở US-P4-06), không tính 5 assertion `thread_test.go` là lỗi.

## Kết luận: **FAIL**
AC10 (chuông trên giao diện) chưa làm đủ: không hiện số chưa đọc, không "9+", không chống nhảy vị trí khi chèn mục mới. Dev đã tự ghi là nợ #3 trong handoff; theo luật QC vẫn FAIL. Phần hạ tầng (một cửa, SSE tức thì, hai gateway, đọc bù, gộp, cô lập, giới hạn kết nối) PASS.

## 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 một cửa | PASS (#12) | `grep -rn 'InsertNotification\|insert into notifications' backend-go --include=*.go --include=*.sql` loại `internal/notify` và `store/queries/notify.sql` (và `*.sql.go` sinh): 0 dòng trong mã không phải test; chỉ còn các tệp `_test.go` chèn dữ liệu mồi (không phải lời gọi sản xuất) |
| 02 `SendMany` một câu SQL | PASS (N=30) | `log_statement='mod'`: lập lịch bài thi cho lớp 30 sinh viên → **+30** dòng `notifications` bằng đúng **1** câu `insert into notifications …` (`log` Postgres). Chưa đo N = 1.000 (cần 1.000 sinh viên thật, không dựng được trên seed 30) |
| 03 test cũ không sửa | PASS | `go test -tags testroutes ./internal/course ./internal/exam ./internal/thread ./internal/calendar ./cmd/worker` → **540 passed**; `git diff 1518f83..HEAD` không đụng test của bốn gói |
| 04 đẩy tức thì ≤ 5 s | PASS | SSE giảng viên (`scripts/sse.mjs`): `POST escalate` → khung `notification {id,type:"TICKET_NEW",title,body,link,course_id,read:false,created_at,unread_count}` sau 93–535 ms |
| 05 p95 | PASS (n=18) | 5 + 13 lượt (không đủ 20): lớn nhất 779 ms (đo ở UI gồm thời gian chạy lệnh); p95 ≪ 5000 ms |
| 06 giao diện không reload | PASS | giảng viên mở danh sách chuông; sinh viên nhờ → mục "N câu hỏi đang chờ giảng viên" tăng sau 779 ms, 0 lần `load` |
| 07 hai bản gateway | PASS | `docker compose up --scale gateway=2`; trong mạng compose (`scripts/two-gateways.mjs`): nối SSE ở A, `POST escalate` ở B → nhận 1 khung, 507 ms; đảo lại → 473 ms |
| 08 dừng một bản | PASS | nối SSE ghi `LAST`; dừng gateway-1; sinh 2 ticket; nối lại với `Last-Event-ID=LAST` qua gateway còn lại → đúng 2 khung thiếu, đúng thứ tự (id tăng), không trùng |
| 09 đọc bù | PASS | `LAST`; ngắt; sinh 3 ticket (+1 ticket do thử Redis chết): nối lại → 4 khung (đủ phần thiếu), tăng dần, không trùng |
| 10 bộ đệm quá hạn | PASS | `DEL ep:sse:buf:<uid>`; nối với id cũ → `resync {"reason":"buffer_exceeded"}`; `GET /notifications` `unread_count`=14 = `count(read_at is null)` trong DB |
| 11 id sai | PASS | `Last-Event-ID: abc` → HTTP 200 + `resync {"reason":"invalid_last_event_id"}` |
| 12 giao diện offline | PASS | `setOffline(true)` 10 s, 4 ticket sinh trong lúc đó; online → chuông hiện 46 = số `OPEN` trong DB (một dòng gộp, đủ số) |
| 13 gộp 5 ticket | PASS | 5 sinh viên nhờ trong 1 phút: teacher có **một** dòng `TICKET_NEW` (id giữ nguyên `…0640da`), tiêu đề 20 → 24; số = số `OPEN` của lớp; TA có dòng gộp riêng; lớp 2 có dòng riêng |
| 14 đọc xong, ticket mới | PASS (diễn giải SRS) | giảng viên `POST /notifications/{id}/read` (`204`); ticket mới → **cùng dòng** (cùng `id`), tiêu đề 24→25, `read_at=NULL` (chưa đọc lại). TC viết "dòng mới (id khác)" theo AC5; SRS 4.4.3 ("`INSERT … ON CONFLICT … DO UPDATE`; chỉ đặt `read_at=NULL` khi số tăng") cho phép cùng dòng — hành vi người dùng đúng (chưa đọc, số mới). Không sửa TC; đề nghị BA thống nhất chữ AC5 / SRS |
| 15 số = 0 tự đọc | BLOCKED | không có đường làm số `OPEN` giảm (nhận / trả lời / đóng thuộc US-P4-03/05); `HandleCreated` chỉ chạy khi có ticket mới. Handoff không nêu cách kích hoạt |
| 16 số giảm giữ `read_at` | BLOCKED | như TC-15 |
| 17 song song | PASS (n=13) | 13 `escalate` đồng thời từ 13 sinh viên: số cuối trong tiêu đề = 38 = số `OPEN` thật; 1 dòng gộp (không nhân đôi). Không đủ 50 lượt (mỗi sinh viên chỉ được 5 ticket / lớp) |
| 18 loại khác / cá nhân | BLOCKED | `AI_PENDING`, `POST_REPORTED` thuộc US-P4-06; `TICKET_ANSWERED` thuộc US-P4-03. Đã thấy: thông báo cá nhân `EXAM_*` của một sinh viên là các dòng riêng, không gộp |
| 19 không lộ nội dung riêng | PASS (một phần) | `TICKET_NEW`: `title`/`body` chỉ số + mã lớp + `link`; `notifications` không chứa canary / tên (0 dòng); `outbox.payload` chỉ `{kind,course_id,ticket_id}`. `TICKET_ANSWERED`, `TICKET_STALE` chưa có |
| 20 giao lại outbox | PASS | đẩy lại dòng `escalation.created` (`dispatched_at=NULL`): worker xử lý lại; `notifications` không tăng (3 → 3); không khung SSE mới vì số không đổi |
| 21 chỉ của mình | PASS | sinh viên A nối SSE, ticket của sinh viên khác sinh ra: A và ADMIN **0** khung; `POST /notifications/{id-của-teacher}/read` bằng A, ADMIN, TA → `404`; `GET /notifications/{id}` → `404`; `/events?user_id=<teacher>` không đổi chủ kênh (khung `ready` của chính A) |
| 22 tab thứ ba | PASS | A có 2 SSE: kết nối thứ ba → `429 SSE_LIMIT_REACHED` `retry_after:5` |
| 23 giao diện rơi về làm mới | PASS (một phần) | chặn `/api/v1/events`: chuông tự làm mới (3 lượt `GET /notifications` trong 36 s), không báo lỗi lên giao diện. Chưa đo đúng nhịp 30 s |
| 24 Redis chết | BLOCKED | Redis dừng 12 s: `escalate` vẫn `201`; dòng `TICKET_NEW` **không** cập nhật tới khi Redis lên lại, rồi tự cập nhật (không mất). Redis chết hẳn làm dừng relay outbox (Redis Streams), nên không dựng được riêng ca "lỗi lúc phát, dòng DB vẫn có" bằng hộp đen; `TestPublishFailureKeepsRow` của dev không thay thế được. Cần cách làm chỉ `Publish` lỗi hoặc PM chấp nhận hành vi trên |
| 25 chuông UI | **FAIL** | `scripts/p4-02-ui.mjs`: nhãn `aria-label="Thông báo, 15 chưa đọc"` nhưng chữ hiển thị rỗng (chỉ chấm `data-part="bell-dot"`): không có số, không "9+". Bấm mục → mở đúng `link` (`/class/members`), đánh dấu đã đọc lạc quan (15 → 14), không toast: PASS; `link` rỗng → `/`: không thử |
| 26 375 px | PASS | chuông 44 × 44; mục danh sách 330 × 65 px; `scrollWidth ≤ innerWidth`; `aria-live="polite"` có ("Có 15 thông báo mới") |
| 27 không nhảy khi chèn | FAIL | chưa làm (handoff nợ #3: "chưa làm"); không có thử nghiệm đối chứng |
| 28 làm mới lỗi | PASS | chặn `GET /notifications`: danh sách cũ còn nguyên; dòng nhạt "Chưa cập nhật được" khi mở; sau nhịp kế thử lại |
| 29 UI | PASS (một phần) | `bash scripts/ui-antipatterns.sh` → `rc=0`; `a11y.spec.ts -g bell` chưa chạy (đợi cổng G4-05) |
| E1 `POST /notifications/read` | KHÔNG KIỂM ĐƯỢC | route thuộc US-P4-05 (`/notifications/read {link}`) |
| P1 quyền chuông | PASS | sinh viên không có `TICKET_NEW`; ADMIN không nhận khung `TICKET_NEW` (0 khung) |

Đối chiếu AC: AC1, AC2, AC3, AC4, AC6 (phần có thể), AC7, AC8, AC11 PASS; AC5 PASS một phần (TC-15, 16, 18 BLOCKED); AC9 PASS một phần (TC-24 BLOCKED); **AC10 FAIL**. Tổng 29 TC chính + 2 nhánh: PASS 21, FAIL 2 (25, 27), BLOCKED 4 (15, 16, 18, 24), KHÔNG KIỂM ĐƯỢC 1 (E1), 1 PASS có điều kiện diễn giải (14).

## 2. Lỗi
**BUG-1 (TRUNG BÌNH): AC10 chưa đủ** — thiếu số chưa đọc (tối đa "9+") hiển thị trên chuông và chống nhảy vị trí khi chèn mục mới lúc danh sách đang mở. Tái hiện: `node ../docs/sprints/7/qc/scripts/p4-02-ui.mjs` (từ `frontend`, cần stack). Handoff đã nêu "chưa làm (cần chốt chấm hay số)". Spec chốt: số tối đa "9+" và không nhảy → cần làm.
**BUG-2 (THẤP, chữ spec): AC5 "dòng mới" so với SRS 4.4.3 "cùng dòng, `read_at=NULL` khi số tăng"** (TC-14). Hành vi hiện tại đúng SRS; đề nghị BA sửa AC5 cho khớp.

## 3. Ghi chú
- Tên tệp kiểm dùng lại: `scripts/sse.mjs` (ghi khung SSE có dấu thời gian), `scripts/two-gateways.mjs` (chạy trong mạng compose, cần `docker run … edupilot-frontend`), `scripts/p4-02-ui.mjs`.
- Handoff nợ 2 (AC4/AC8 không có test mới trong `notify`): QC đã kiểm hộp đen cả hai (TC-09…11, 21) nên không cần thêm test.
- Handoff nợ 4 (`SendMany` phát tuần tự): trên 30 người không đo được khác biệt.
- Dữ liệu QC thêm bằng SQL: 12 dòng `notifications` `COURSE_ASSIGNED` cho giảng viên (để thử "9+"), đặt lại giờ bài thi.
