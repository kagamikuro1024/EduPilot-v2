# QC report — US-P4-02 (chuông tức thời, gộp, hai bản gateway, một cửa) — **vòng 3 (cuối)**
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `79a230d` (code US-P4-02 vòng 3: `a606a29`, `1381a79`). Bộ TC: `tc-US-P4-02.md` (không sửa chữ TC). Handoff: `dev-US-P4-02.md` mục "Vòng 3". Stack dựng lại từ code mới: `edupilot100`, `docker-compose.test.yml`, `--scale gateway=2`, provider `fake`, DB trống.

## Kết luận: **PASS** (BUG-1 vòng 2 đã sửa; không FAIL; 2 TC còn BLOCKED vì phụ thuộc US-P4-06 và quyết định PM)
## 1. Kết quả TC chạy lại ở vòng 3
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 29 antipatterns | **PASS** | `bash scripts/ui-antipatterns.sh` → mọi dòng `✓`, `rc=0` (vòng 2 `rc=1`: `font-size: 10px` ở `AppShell.module.css:124`, nay dùng token) |
| 25 chuông UI | PASS | `scripts/p4-02-ui.mjs`: 13 chưa đọc → chữ "9+"; `aria-label="Thông báo, 13 chưa đọc"`; bấm mục → `/class/members`, 375 px cũng "9+" (12 chưa đọc) |
| 26 375 px | PASS | chuông 44×44; mục 330×65; không tràn ngang (chấm 18 px không phá khung) |
| 27 không nhảy | PASS | `scripts/p4-02-nojump.mjs`: Δ = 1,5 px; dòng gộp lên đầu |
| 07 hai gateway | PASS | `two-gateways.mjs` chạy trong container frontend: A→B 252 ms, B→A 229 ms, mỗi hướng 1 khung |
| 13 gộp | PASS | 5 ticket 5 sinh viên → 1 dòng `TICKET_NEW` "8 câu hỏi đang chờ giảng viên", số = `OPEN` thật (8) |
| 16 số giảm giữ `read_at` | PASS | `claim`, `dismiss` → tiêu đề 8 → 6; `read_at` giữ khi đã đọc |
| 15 số về 0 | PASS | hết `OPEN` → dòng không bị xoá |
| 01…12, 14, 17, 19…23, 28 | PASS (giữ) | không đổi mã (vòng 3 chỉ đổi CSS chấm, `useSSE({enabled})`, `session.ts` của test) |
| e2e (desktop) `escalation, shell, class-join, private-chat, data-layer, a11y` | **PASS** | 141 passed, 5 skipped, **0 failed** (vòng 2: 4 đỏ). Gồm `shell.spec` "forbidden" (#18) và `data-layer` "sse parse + singleton" |
| Chuông chỉ nghe `/events` khi có phiên | PASS | `data-layer.spec` xanh; trang chưa đăng nhập không mở `/events` (e2e) |

## 2. Còn BLOCKED (không tính FAIL)
- **TC-18** `AI_PENDING`, `POST_REPORTED`: thuộc US-P4-06 (chưa thi công).
- **TC-24** Redis chết: dev xin PM chấp nhận hành vi quan sát (hàng cập nhật khi Redis lên lại); chờ quyết định PM.

## 3. Phụ lục: vòng 1 và 2 (đã đóng)
### Kết luận vòng 2: **FAIL** (1 lỗi thấp: TC-29)
AC10 (chuông) nay đủ: số chưa đọc "9+", chống nhảy vị trí. Còn một lỗi: `bash scripts/ui-antipatterns.sh` đỏ (rc=1) vì huy hiệu số của chuông đặt `font-size: 10px` ngoài thang vai trò (`AppShell.module.css:124`). Luật `CLAUDE.md`: script phải sạch trước khi báo xong. Ba TC vẫn BLOCKED (TC-18 phần Threads, TC-24; xem dưới) nhưng không tính FAIL theo yêu cầu PM nếu handoff chưa cho đường kiểm.

### 1. Kết quả TC
| TC | Kết quả | Bằng chứng vòng 2 |
| --- | --- | --- |
| 01 một cửa | PASS (#12) | không còn `InsertNotification` / `insert into notifications` ngoài `internal/notify` và `store/queries/notify.sql` trong mã không phải test |
| 02 `SendMany` | PASS (N=30) | `log_statement='mod'`: lập lịch bài thi (lớp 30 SV) → +30 dòng bằng 1 câu `insert into notifications`; chưa đo N=1.000 |
| 03 test cũ không sửa | PASS | vòng 1: 540 test pass; `git diff 1518f83..HEAD` không sửa test của course/exam/thread/calendar. Vòng 2 chạy lại `./cmd/worker ./db ./internal/contract` = 42 pass |
| 04–06 tức thì, p95, UI | PASS | khung `notification` tới sau 93–535 ms; UI không reload (vòng 1; code SSE không đổi, kiểm lại tới khung ở TC-13/17: tới ≤ 1 s) |
| 07 hai gateway | PASS | `--scale gateway=2` (vòng 2 dựng lại): nối A / sinh ở B và ngược lại, mỗi hướng 1 khung, 473–507 ms (`scripts/two-gateways.mjs`) |
| 08 dừng một bản | PASS | nối `LAST`; dừng gateway-1; sinh 2 ticket; nối lại `Last-Event-ID=LAST` qua bản còn lại → đúng 2 khung thiếu, tăng dần, không trùng |
| 09–11 đọc bù | PASS (vòng 1) | 4 khung đúng thứ tự không trùng; `resync buffer_exceeded` khi xoá bộ đệm; `resync invalid_last_event_id` khi id sai |
| 12 UI offline | PASS (vòng 1) | online lại chuông đủ số = số `OPEN` |
| 13 gộp 5 ticket | PASS | 1 dòng `TICKET_NEW`, `id` giữ nguyên, số = số `OPEN` thật (3 → 18 → 227 trong các lượt); TA và lớp 2 có dòng riêng |
| 14 đọc xong, ticket mới | PASS (diễn giải SRS 4.4.3) | cùng dòng, `read_at=NULL` khi số tăng; AC5 chữ "dòng mới" lệch SRS — PM ghi BA sửa chữ (BUG-2 vòng 1) |
| 15 số = 0 tự đọc | **PASS** (mới) | đóng hết (`dismiss`) → dòng `TICKET_NEW` `read_at` đặt, dòng không bị xoá (1 dòng cho teacher/lớp); `OPEN`=0 |
| 16 số giảm giữ `read_at` | **PASS** (mới) | 3 `OPEN`, giảng viên đọc (`read_at` có), `claim` → tiêu đề "2 câu hỏi…", `dismiss` → "1 câu hỏi…", `read_at` giữ nguyên cả hai lần |
| 17 song song | PASS | 13 `escalate` đồng thời; số cuối = số `OPEN` |
| 18 loại khác / cá nhân | PASS một phần / BLOCKED một phần | `TICKET_ANSWERED` là dòng riêng mỗi lần trả lời (sv04: 2 lần `answer` → 2 dòng, không gộp, link `/chat?session=…`). `AI_PENDING`, `POST_REPORTED` thuộc US-P4-06 (chưa thi công) → BLOCKED |
| 19 không lộ nội dung | PASS | `TICKET_NEW`, `TICKET_ANSWERED` (title "Giảng viên đã trả lời câu hỏi của bạn", không nội dung); `outbox.payload` chỉ `{kind,course_id,ticket_id}`; audit không chứa canary. `TICKET_STALE` thuộc US-P4-05 |
| 20 giao lại outbox | PASS (vòng 1) | không thêm thông báo, không khung SSE mới |
| 21 chỉ của mình | PASS | A và ADMIN 0 khung; đọc / đánh dấu thông báo người khác → 404 |
| 22 tab thứ ba | PASS | `429 SSE_LIMIT_REACHED` |
| 23 rơi về làm mới | PASS (một phần) | chặn `/events`: chuông tự làm mới, không lỗi |
| 24 Redis chết | BLOCKED | dev đề nghị PM chấp nhận (`dev-US-P4-02.md`: không có khoảnh khắc "chỉ `Publish` lỗi" khi Redis dừng hẳn; `TestPublishFailureKeepsRow` phủ). Hành vi quan sát vẫn như vòng 1: không mất, cập nhật khi Redis lên lại. PM quyết |
| 25 chuông UI | **PASS** (mới) | `scripts/p4-02-ui.mjs`: giảng viên 14 chưa đọc → chữ hiển thị **"9+"**; `aria-label="Thông báo, 14 chưa đọc"`; `aria-live="polite"` "Có N thông báo mới"; bấm mục → mở đúng `link` (`/class/members`), đánh dấu đã đọc lạc quan (14 → 13), không toast |
| 26 375 px | PASS | chuông 44×44; mục 330×65; không tràn ngang; hiện "9+" ở 375 px |
| 27 không nhảy khi chèn | **PASS** (mới) | `scripts/p4-02-nojump.mjs`: danh sách mở, ticket mới → dòng gộp lên đầu; mục "QC thông báo 6" lệch y = 1,5 px (< 2) |
| 28 làm mới lỗi | PASS | danh sách cũ còn, dòng "Chưa cập nhật được", tự thử lại |
| 29 antipatterns | **FAIL** | `bash scripts/ui-antipatterns.sh` → `rc=1`: `✗ Cỡ chữ tuỳ ý ngoài thang vai trò` tại `frontend/src/shared/shell/AppShell.module.css:124` (`.unreadDot … font-size: 10px`). Vòng 1 `rc=0`; do thay đổi chuông vòng 2. (Vi phạm thứ hai `RealChat.tsx:80` thuộc US-P4-03) |
| E1 `POST /notifications/read` | PASS | route của US-P4-03: `{link}` rỗng → 422; link của người khác → không đổi dòng người kia |
| P1 | PASS | sinh viên không có `TICKET_NEW`; ADMIN không nhận khung ticket |

Tổng: 29 TC chính + 2 TC nhánh / quyền = 31: PASS 28 (gồm TC-14, 15, 16, 25, 27 mới đạt), FAIL 1 (TC-29), BLOCKED 2 (TC-18 phần Threads, TC-24).

### 2. Lỗi
**BUG-1 (THẤP): `ui-antipatterns.sh` đỏ ở `AppShell.module.css:124`** — `font-size: 10px` của `.unreadDot`. Tái hiện: `bash scripts/ui-antipatterns.sh; echo $?` → `1`. Sửa: dùng token cỡ chữ trong thang vai trò (`--ep-text-*`). Không đụng hành vi.

### 3. Ghi chú
- Kiểm giao diện chuông ở stack có số thật: hai người đọc cùng dòng, số "9+" ở 375 px, khung chèn không nhảy — xem script trong `docs/sprints/7/qc/scripts/` (`p4-02-ui.mjs`, `p4-02-nojump.mjs`, `sse.mjs`, `two-gateways.mjs`).
- `shell.spec.ts:312` "forbidden… 0 request /api/v1" vẫn đỏ vì chuông mở SSE `/api/v1/events` (proposals #18 ACCEPTED, dev chưa áp dụng ở `HEAD`): ghi ở `report-US-P4-03.md` (cổng G4-05).

### 4. Phụ lục: vòng 1 (đã đóng)
### Kết luận vòng 2: **FAIL**
AC10 (chuông trên giao diện) chưa làm đủ: không hiện số chưa đọc, không "9+", không chống nhảy vị trí khi chèn mục mới. Dev đã tự ghi là nợ #3 trong handoff; theo luật QC vẫn FAIL. Phần hạ tầng (một cửa, SSE tức thì, hai gateway, đọc bù, gộp, cô lập, giới hạn kết nối) PASS.

### 1. Kết quả TC
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

Đối chiếu AC: AC1, AC2, AC3, AC4, AC6 (phần có thể), AC7, AC8, AC11 PASS; AC5 PASS một phần (TC-15, 16, 18 BLOCKED); AC9 PASS một phần (TC-24 BLOCKED); **AC10 FAIL**. Tổng 29 TC chính + 2 nhánh = 31: PASS 24 (gồm TC-14 theo diễn giải SRS), FAIL 2 (25, 27), BLOCKED 4 (15, 16, 18, 24), KHÔNG KIỂM ĐƯỢC 1 (E1).

### 2. Lỗi
**BUG-1 (TRUNG BÌNH): AC10 chưa đủ** — thiếu số chưa đọc (tối đa "9+") hiển thị trên chuông và chống nhảy vị trí khi chèn mục mới lúc danh sách đang mở. Tái hiện: `node ../docs/sprints/7/qc/scripts/p4-02-ui.mjs` (từ `frontend`, cần stack). Handoff đã nêu "chưa làm (cần chốt chấm hay số)". Spec chốt: số tối đa "9+" và không nhảy → cần làm.
**BUG-2 (THẤP, chữ spec): AC5 "dòng mới" so với SRS 4.4.3 "cùng dòng, `read_at=NULL` khi số tăng"** (TC-14). Hành vi hiện tại đúng SRS; đề nghị BA sửa AC5 cho khớp.

### 3. Ghi chú
- Tên tệp kiểm dùng lại: `scripts/sse.mjs` (ghi khung SSE có dấu thời gian), `scripts/two-gateways.mjs` (chạy trong mạng compose, cần `docker run … edupilot-frontend`), `scripts/p4-02-ui.mjs`.
- Handoff nợ 2 (AC4/AC8 không có test mới trong `notify`): QC đã kiểm hộp đen cả hai (TC-09…11, 21) nên không cần thêm test.
- Handoff nợ 4 (`SendMany` phát tuần tự): trên 30 người không đo được khác biệt.
- Dữ liệu QC thêm bằng SQL: 12 dòng `notifications` `COURSE_ASSIGNED` cho giảng viên (để thử "9+"), đặt lại giờ bài thi.
