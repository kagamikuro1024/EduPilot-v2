# QC report — US-P4-01 (migration `00012`, ticket, nút "Nhờ giảng viên", tự mở, trần 5) — **vòng 2**
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `438c649`. Bộ TC: `tc-US-P4-01.md` (không sửa chữ TC). Handoff: `dev-US-P4-01.md` mục "Vòng 2", `dev-QC-answers.md`. Stack dựng lại từ code mới: project `edupilot100` (cổng HTTPS 543, Mailpit 8125), `docker-compose.test.yml` (build `gateway-test`, `worker-test`, có `_test/*`), `--scale gateway=2`, provider `fake`; DB trống + seed (dừng ở bước 11 do thiếu docling như vòng 1, bước 1–10 xong). PM đã nhận chữ spec #15 (PENDING / ngoài lớp → 404), #16 (Redis chết vẫn tạo ticket), #17 (giờ thi dùng `EXAM_IN_PROGRESS`); chấm theo chữ mới.

## Kết luận: **PASS** (42/42 TC; không FAIL, không BLOCKED)
Ba lỗi vòng 1 đã xử lý: BUG-1 (hợp đồng) sửa trong code; BUG-2, BUG-3 là chữ spec, hành vi đã khớp chữ mới. Hai TC BLOCKED vòng 1 (TC-14, ca `degraded` của TC-15, ca `STREAMING`/`CANCELLED` của TC-18, TC-28) nay chạy được nhờ `dev-QC-answers.md` và route staff của US-P4-03.

## 1. Kết quả TC
| TC | Kết quả | Bằng chứng vòng 2 |
| --- | --- | --- |
| 01–06 migration, lược đồ, mặc định | PASS (không đổi) | `git diff 24ef239..HEAD -- backend-go/db/migrations` rỗng; `gateway migrate up` trên DB trống chạy sạch tới `version 12`, `00012` Applied; kết quả vòng 1 giữ (CHECK / UNIQUE / FK, `down` một bước bằng khối `Down`, UPDATE dữ liệu cũ `0.60→0.80`, `true→false`) |
| 07–09 tạo, lặp, 10 song song | PASS | `201 already:false`; 3 lần lặp `200 already:true`; 10 song song: 1 × `201` + 9 × `200`, cùng `ticket.id`, `count`=1 |
| 10 `CLOSED` không mở lại | PASS | `escalate` lại → `200 already:true`, trạng thái `CLOSED` |
| 11 đủ 5 bấm lại tin đã nhờ | PASS | `200 already:true` |
| 12 tự mở dưới ngưỡng | PASS | `done` có `ticket`; `origin=AUTO_LOW_CONFIDENCE`; outbox `escalation.created` |
| 13 nguyên tử | PASS | trigger tạm lỗi outbox: tin `FAILED`, `ticket:null`, `escalation_tickets` 11→11; gỡ trigger |
| 14 intent tự mở | PASS (theo thiết kế, handoff) | `COURSE_QA` ra `AUTO_LOW_CONFIDENCE`; `GRADE_FORMULA` ra `AUTO_NO_FORMULA`; `LIBRARY_SEARCH`, `PERSONAL_*`, `WHAT_IF_GRADE`, `EXAM_SCHEDULE`, `UPCOMING_EVENTS` → câu mẫu `low_confidence=false` hoặc tool 1,000 nên 0 ticket tự động — đúng bảng ở `dev-US-P4-01.md` ("không có cách, đúng thiết kế"); `TestShouldEscalateTable` 18 dòng |
| 15 nhánh không tự mở | PASS | `CRISIS`, `OTHER_PERSON` (họ tên trong roster), `SMALLTALK`, `low_confidence=false` → 0 ticket; `degraded=true`: `POST /_test/llm/fake {"error_rate":1}` → câu "AI đang gián đoạn", `degraded:true`, `ticket:null`, `can_escalate:true`, 0 ticket (trả lại `error_rate:0`) |
| 16 `AUTO_NO_FORMULA` | PASS | "Công thức tính điểm của môn là gì?" → câu "Lớp chưa có công thức điểm chính thức…", ticket `AUTO_NO_FORMULA` |
| 17 `TestShouldEscalateTable` | PASS | vòng 1: `ok`, 18 dòng (code không đổi) |
| 18 nút (UI) | PASS | `scripts/p4-01-ui.mjs`: nút ở 2 tin thường; không ở `CRISIS` / `OTHER_PERSON` / `SMALLTALK` / `FAILED`. `STREAMING`: `stream_delay_ms:3000`, `can_escalate:false`, `escalate` → `422 MESSAGE_NOT_ESCALATABLE`; sau `POST …/cancel` (`204`) trạng thái `CANCELLED`, `can_escalate:false`, `escalate` → `422` |
| 19–24 giao diện | PASS | dòng "Đang chờ giảng viên · đã gửi N phút trước", không nút; tooltip đúng "Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI"; nút `disabled` + "Bạn đang có 5 câu hỏi chờ giảng viên"; bấm đúp = 1 POST (trạng thái hiện sau 3 ms đo qua Playwright); offline → "Đang gửi…" rồi "Đang chờ giảng viên · vừa gửi", 1 ticket; 5xx → "Chưa gửi được. Thử lại" |
| 25–27 trần | PASS | tin thứ 6 → `409 TICKET_LIMIT_REACHED`; tự mở bị chặn (`ticket:null`, log `chat_tickets_suppressed_total`); lớp 2 và sinh viên khác không bị chặn; `ANSWERED` không tính (vòng 1) |
| 28 giảng viên chỉ thấy cặp hỏi–đáp | PASS | tin liền trước / liền sau chứa `CANARY-9K3Z`, cùng phiên khác và phiên xoá mềm: `GET …/tickets/{id}` bằng giảng viên → `canary=0` (1.066 byte, chỉ `ticket`, `question`, `ai_answer`, `transcript`) |
| 29 vai | PASS | SV khác 404; TA, TEACHER, ADMIN 403; không token 401; `mid` không tồn tại 404 |
| 30 `PENDING` / `REMOVED` / ngoài lớp | PASS (#15) | `PENDING` (sv52) 404; ngoài lớp (`sv.moi`) 404; `REMOVED` (tin của chính họ) 403 |
| 31 `ARCHIVED` | PASS | `409 COURSE_ARCHIVED` |
| 32 giờ thi | PASS (#17) | vòng 1: `409 EXAM_IN_PROGRESS` cùng mã với chat riêng; code không đổi |
| 33 không lộ nội bộ | PASS | `ticket` chỉ `{id,status,created_at[,answered_at]}`; 0 khoá `confidence`, `claimed_by`, `origin`, `sla_stage`, `answered_by` |
| 34 không đủ điều kiện | PASS | `USER`, `CRISIS`, `OTHER_PERSON`, `SMALLTALK`, `FAILED` → `422 MESSAGE_NOT_ESCALATABLE` |
| 35 DB / Redis chết | PASS (#16) | PgBouncer dừng → `503 SERVICE_UNAVAILABLE` `retry_after:5`, 0 dòng nửa vời; Redis dừng → `201`, ticket tạo đúng 1 dòng |
| 36 `Idempotency-Key` tuỳ chọn | PASS | có / không có key đều chạy; cùng key → `201` rồi `200 already:true` |
| 37 hợp đồng | PASS | `go test -count=1 -tags testroutes ./internal/contract/... ./db ./cmd/worker` → **42 passed** (vòng 1: 3 fail) |
| 38 chỉ mục | PASS (không đổi) | lược đồ không đổi: `escalation_tickets_course_status_idx`, `escalation_tickets_student_idx`, `escalation_tickets_sla_idx`; không `Seq Scan` |
| E1, E3, P1 | PASS | hai tab = TC-09/22; log 0 canary; SV khác đọc phiên → 404 |
| E2 phiên xoá mềm | PASS | `TC-P4-03-23`: ticket giữ `session_id`, chi tiết vẫn đọc, trả lời vào "Tin nhắn từ giảng viên" |

Phần chạy lại hoàn toàn ở vòng 2 (DB mới): TC-07…16, 18…27, 28…37 và các ca nhánh lỗi. Tổng: 38 TC chính + 4 TC nhánh lỗi / quyền = 42; PASS 42.

## 2. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -tags testroutes ./internal/contract/... ./db ./cmd/worker` | 42 passed |
| `go test -count=1 -tags testroutes ./internal/course ./internal/exam ./internal/thread ./internal/calendar ./cmd/worker` | 540 passed (vòng 1; không có sửa test của các gói này trong `git diff 1518f83..HEAD`) |
| `playwright test escalation.spec.ts private-chat.spec.ts … --project=desktop` | xem mục 3 (lỗi test không phải lỗi của US-P4-01) |
| `bash scripts/ui-antipatterns.sh` | **rc=1** — xem `report-US-P4-02.md` / `-03.md` (2 vi phạm nằm ngoài mã của story này) |

## 3. Ghi chú không làm FAIL story
- `private-chat.spec.ts:235` "auto ticket…" đỏ 2/2 lượt ở 00:21 (giờ máy). Nguyên nhân quan sát được: ticket mô phỏng tạo ở `Date.now() − 3 giờ` và giao diện in "đã gửi **hôm qua 21:21**" khi ngày đổi (qua nửa đêm), trong khi test mong "đã gửi 3 giờ trước". Test phụ thuộc giờ trong ngày; không phải lỗi của nút / ticket. Đề xuất dev cố định đồng hồ trong test.
- `GET /api/v1/_test/llm/payloads` là bộ nhớ của từng bản gateway: khi `--scale gateway=2` mỗi bản có vòng đệm riêng; khi kiểm payload cần dừng một bản. Ghi nhận để handoff nêu.
- Cache câu trả lời lớp (`ep:*cache*`) làm lặp lại câu hỏi y hệt trả kết quả cũ (kể cả câu "AI đang gián đoạn" từ một lần lỗi trước); muốn thử lại với provider khác phải xoá khoá cache. Không phải lỗi của story này.

## 4. Phụ lục: vòng 1 (đã đóng)
### Kết luận: **FAIL**
Một lỗi trung bình (BUG-1: test hợp đồng đỏ), hai lệch chữ AC mức thấp (BUG-2, BUG-3) cần PM quyết, hai TC BLOCKED. Phần nghiệp vụ lõi (tạo, idempotent, 10 song song, trần 5, tự mở cùng giao dịch, phân quyền, giao diện) PASS.

### 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 migration up/down/up | PASS | DB riêng: `gateway migrate up` → `00012 Applied`; chạy phần `Down` của `00012` (CLI `gateway migrate down` lùi hết, không lùi một bước nên cắt khối `-- +goose Down`): ba bảng mất, `escalation_threshold` về `0.60`, `notify_ticket_by_mail` về `true`, cột `hidden_at` mất; xoá dòng `goose_db_version` 12 rồi `up` lại sạch |
| 02 grep `00012` | PASS | `CREATE TABLE mail_inbound` = 0; đúng 4 `ALTER TABLE` (`forum_threads`, `forum_posts`, `user_settings`, `courses`); 2 `CREATE INDEX` (`documents_knowledge_key`, `mail_outbox_dead_idx`) |
| 03 `PROGRESS.md` | PASS | dòng `00012 | escalation_notify` |
| 04 CHECK / UNIQUE / FK `escalation_tickets` | PASS (ghi chú) | `OPEN` có `claimed_by` → `23514 open_chk`; `CLAIMED` thiếu → `claimed_chk`; `ANSWERED` thiếu → `answered_chk`; `QUESTION` không `message_id` → `question_chk`; `GRADE_APPEAL` không payload / `QUESTION` có payload → `appeal_chk`; trùng `message_id` → `23505`; lệch lớp → `23503 message_fkey`. `kind='X'` bị từ chối bằng `22P02` (enum), không phải `23514` — vẫn chặn, chỉ khác mã |
| 05 `post_reports`, `knowledge_items` | PASS (cấu trúc) | `\d`: `target_chk` đúng một trong `thread_id`/`post_id`; `UNIQUE (reporter_id,target_id)`; `reason` là enum; `note` chỉ khi `OTHER`; `knowledge_items` `UNIQUE (source_type,source_id)`, `chunk_id ON DELETE CASCADE`. Chưa chèn thử vì chưa có thread (US-P4-06) |
| 06 mặc định + UPDATE dữ liệu cũ | PASS | bản sao DB đưa về `00011`, đặt lớp A (`version=3`) `0.60`, lớp B `0.70`, `user_settings` `version=2` `true`; `up`: `0.80` / `0.70` / `false` (version giữ); mặc định cột `0.80` / `false` |
| 07 `escalate` tạo | PASS | `201 {already:false,ticket:{id,status:"OPEN",created_at}}` |
| 08 gọi lại tuần tự | PASS | 3 lần → `200 already:true` cùng `id`; `count(*)`=1 |
| 09 10 song song | PASS | 1 × `201`, 9 × `200`; cùng `ticket.id`; `count`=1 |
| 10 ticket `CLOSED` không mở lại | PASS | đặt `CLOSED` bằng SQL (route đóng thuộc US-P4-03/05); `escalate` → `200 already:true`, trạng thái vẫn `CLOSED` |
| 11 đủ 5 vẫn bấm lại tin đã nhờ | PASS | `$SVA` có 5 mở: tin đã nhờ → `200 already:true` (không `409`) |
| 12 tự mở dưới ngưỡng | PASS | khung `done` có `ticket:{id,status,created_at}`; `origin=AUTO_LOW_CONFIDENCE`, `confidence` có; 1 dòng outbox `escalation.created` (payload chỉ `ticket_id`, `course_id`, `kind`); `GET …/messages` có `ticket` |
| 13 nguyên tử với câu trả lời | PASS | trigger tạm làm outbox `escalation.created` lỗi: tin `stream_status=FAILED`, `ticket:null`, `escalation_tickets` không tăng; gỡ trigger |
| 14 các intent tự mở | BLOCKED (một phần) | `COURSE_QA` ra ticket. Các intent có dữ liệu (`LIBRARY_SEARCH`, `PERSONAL_*`, `WHAT_IF_GRADE`, `EXAM_SCHEDULE`, `UPCOMING_EVENTS`) trả câu `low_confidence=false` ("Hệ thống chưa có dữ liệu…") nên không ticket; không ép được `low_confidence=true` cho từng intent bằng provider giả → `Q-QC-P4-02` chưa được trả lời. Bù: `TestShouldEscalateTable` 18 dòng xanh (TC-17), nhưng đó là test của dev |
| 15 nhánh không tự mở | PASS (thiếu một ca) | `CRISIS`, `OTHER_PERSON` (hỏi bằng họ tên có trong roster; câu chỉ nêu "Kha" bị phân loại `COURSE_QA`, không phải lỗi sản phẩm), `SMALLTALK`, câu `low_confidence=false` → 0 ticket. `degraded=true` KHÔNG KIỂM ĐƯỢC: không có cách làm provider chết trên stack (không có route `_test`, handoff không nêu) |
| 16 `AUTO_NO_FORMULA` | PASS | "Công thức tính điểm của môn là gì?" → câu "Lớp chưa có công thức điểm chính thức…" + ticket `AUTO_NO_FORMULA`. "Điểm cuối kỳ tính thế nào?" ra `COURSE_QA` + `AUTO_LOW_CONFIDENCE` (phân loại intent, không thuộc story) |
| 17 `TestShouldEscalateTable` | PASS | `ok`, 18 dòng con |
| 18 nút hiện / không hiện (UI) | PASS (thiếu hai ca) | `scripts/p4-01-ui.mjs`: có nút ở hai tin thường; không nút ở `CRISIS`, `OTHER_PERSON`, `SMALLTALK`, tin `FAILED`. `STREAMING`/`CANCELLED`: không dựng được (huỷ giữa chừng quá nhanh) — KHÔNG KIỂM ĐƯỢC hai ca này |
| 19 dòng trạng thái sau khi tự mở | PASS | "Đang chờ giảng viên · đã gửi 3 phút trước", không nút |
| 20 tooltip duy nhất | PASS | `title`="Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI" |
| 21 khoá khi đủ 5 | PASS | nút `disabled`, tooltip "Bạn đang có 5 câu hỏi chờ giảng viên" |
| 22 bấm đúp | PASS | đúng 1 `POST /escalate`; trạng thái hiện ngay (đo qua Playwright: 3 ms); `count`=1 |
| 23 mất mạng | PASS | offline → "Đang gửi…"; online → "Đang chờ giảng viên · vừa gửi"; 1 request, 1 ticket |
| 24 lỗi 5xx | PASS | nút hoàn lại + "Chưa gửi được. Thử lại" |
| 25 trần 5 | PASS | tin thứ 6 → `409 TICKET_LIMIT_REACHED` |
| 26 tự mở bị chặn | PASS | `ticket:null`, câu vẫn `low_confidence:true`; log `chat_tickets_suppressed_total` (WARN, có `course_id`); bộ đếm là log + `atomic` (handoff nợ 4), không đọc được qua `/metrics` |
| 27 trần theo lớp / sinh viên / `ANSWERED` | PASS | cùng sinh viên lớp 2 → `201`; sinh viên khác lớp 1 → `201`; đặt 1 ticket `ANSWERED` (SQL) → tin thứ 6 `201` |
| 28 giảng viên chỉ thấy cặp hỏi–đáp | BLOCKED | route nhân viên (`GET …/tickets/{id}`, `/me/tickets`) chưa có trong US-P4-01 (→ `404`); chuyển sang `TC-P4-03-09`. Handoff nói AC10 ✓ ở mức dịch vụ (`TestStaffCannot…`), không kiểm hộp đen được |
| 29 phân quyền (vai) | PASS | sinh viên khác và `mid` không tồn tại → `404 NOT_FOUND`; TA, TEACHER, ADMIN → `403`; không token → `401` |
| 30 `PENDING` / `REMOVED` / ngoài lớp | FAIL (thấp) | `REMOVED` (tin của chính họ) → `403`. `PENDING` (`sv52`) và ngoài lớp (`sv.moi`) gọi tin của người khác → `404`, spec AC11 (b) ghi `403`. Hai người này không thể có tin của chính mình nên chỉ chạm nhánh "tin của người khác → 404" (AC11a): hai điều khoản mâu thuẫn nhau. Xem BUG-2 |
| 31 `ARCHIVED` | PASS | `409 COURSE_ARCHIVED` |
| 32 giờ thi | PASS (ghi chú) | sinh viên đang làm bài (`attempts` 201): `escalate` → `409 EXAM_IN_PROGRESS` cùng mã và HTTP với chat riêng (`POST …/messages`). Spec ghi `EXAM_LOCKED`; handoff nợ 1 đã nêu dùng mã chat — khớp ý định "cùng mã với chat riêng". Cần sửa chữ spec |
| 33 sinh viên không thấy nội bộ | PASS | duyệt `chat/sessions`, `messages`, SSE `done`, `me/today`, `notifications`, `threads`: 0 khoá `confidence`, `claimed_by`, `origin`, `sla_stage`, `answered_by`; mọi `ticket` chỉ `{id,status,created_at[,answered_at]}` |
| 34 không đủ điều kiện | PASS | tin `USER`, `CRISIS`, `OTHER_PERSON`, `SMALLTALK`, `FAILED` → `422 MESSAGE_NOT_ESCALATABLE`; không ticket |
| 35 DB / Redis chết | FAIL (thấp, một nửa) | PgBouncer dừng: `503 SERVICE_UNAVAILABLE` `retry_after:5`, không dòng nửa vời. Redis dừng: `escalate` vẫn `201`, ticket tạo đúng 1 dòng (không dùng Redis ở đường này). AC12 ghi "DB / Redis không dùng được → 503". Xem BUG-3 |
| 36 `Idempotency-Key` tuỳ chọn | PASS | có và không có key đều chạy; cùng key hai lần → `201` rồi `200 already:true`, 1 ticket |
| 37 hợp đồng | FAIL | `go test -tags testroutes ./internal/contract/...` → 18 pass, **3 fail**: `TestChatThreadsContract` "21 thao tác (cần 20)", `TestSpec_LoadsAndValidates` "171 thao tác (cần 170)", `TestSpec_SecurityDeclared` "thao tác lạ … POST /api/v1/chat/messages/{mid}/escalate". `openapi.yaml` có route `escalate` và trường `ticket`, `can_escalate`. Xem BUG-1 |
| 38 chỉ mục | PASS (ghi chú) | 10.000 ticket / 50 lớp trên DB riêng, `ANALYZE`: trang đầu theo lớp + trạng thái → `Index Scan escalation_tickets_course_status_idx`; theo sinh viên → `escalation_tickets_student_idx (course_id,student_id,status,created_at,id)`; SLA → `escalation_tickets_sla_idx` (partial, `waiting_since`, `id`, không `course_id` vì job quét mọi lớp). Không `Seq Scan`. Câu SQL do QC dựng vì handoff không liệt kê câu của job SLA (`Q-QC-P4-04`) |
| E1 hai tab | PASS | phủ bởi TC-09 và TC-22 |
| E2 phiên xoá mềm | KHÔNG ÁP DỤNG | cần route nhân viên (US-P4-03) |
| E3 log không có canary | PASS | `CANARY-9K3Z` = 0 ở log gateway, worker; = 0 ở `notifications` |
| P1 sinh viên khác đọc phiên | PASS | `GET /chat/sessions/$SA/messages` bằng `$SVB` → `404` |

Tổng: 38 TC chính + 4 TC nhánh lỗi / quyền = 42. PASS 36, FAIL 3 (30, 35, 37), BLOCKED 2 (14, 28; TC-15 và TC-18 PASS nhưng thiếu một ca mỗi cái), KHÔNG ÁP DỤNG 1 (E2). Đối chiếu AC: AC1–AC9, AC14 PASS; AC10 BLOCKED; AC11 FAIL (b); AC12 FAIL (Redis); AC13 FAIL.

### 2. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -race ./internal/escalation ./internal/agent …` | không chạy lại (trùng bằng chứng hộp đen); chỉ chạy `TestShouldEscalateTable` ok |
| `go test -tags testroutes ./internal/contract/...` | **3 FAIL** như TC-37 (dev đã ghi "chờ #11") |
| `go test -tags testroutes ./internal/course ./internal/exam ./internal/thread ./internal/calendar ./cmd/worker` | **540 passed**, 5 gói; `git diff 1518f83..HEAD` không sửa test của course / exam / thread / calendar |
| `playwright test escalation.spec.ts private-chat.spec.ts --project=desktop` (gate `build:gate`) | lượt đầu 16 pass / **4 fail** (`page.goto` quá hạn: cold start), lượt hai ngay sau **20 pass**; ghi nhận chập chờn khởi động, không phải lỗi hành vi |
| `bash scripts/ui-antipatterns.sh` | `rc=0` |

### 3. Lỗi
**BUG-1 (TRUNG BÌNH): test hợp đồng đỏ sau khi thêm route.** Tái hiện: `cd backend-go && go test -tags testroutes ./internal/contract/...`. `proposals.md` #11 ACCEPTED yêu cầu mỗi story cập nhật **đúng** con số ở cùng commit thêm route, commit ghi `#11`; commit `863c02e` chưa làm (handoff ghi "chờ #11"). Cần dev cập nhật ba con số (`20 → 21`, `170 → 171`, danh sách bảo mật) và `TestMigrations_RoundTrip` (đếm bảng / version 12). Nếu không story không qua cổng `go test ./...`.
**BUG-2 (THẤP): AC11 (a) và (b) mâu thuẫn cho PENDING / ngoài lớp.** Sinh viên `PENDING` / ngoài lớp gọi `escalate` tin của người khác nhận `404`; AC11 (b) viết `403`. Đề xuất cho PM: giữ `404` (không lộ sự tồn tại, nhất quán với AC11 a) và sửa chữ AC11 (b) thành "sinh viên `REMOVED` → 403; `PENDING` / ngoài lớp: không có chat riêng nên mọi tin là của người khác → 404"; hoặc dev đổi thứ tự kiểm tra.
**BUG-3 (THẤP): AC12 nói Redis chết → `503`.** Thực tế ticket vẫn tạo được (không cần Redis ở đường tạo; outbox ghi Postgres). PM quyết: sửa chữ AC12 (chỉ DB → 503) hay buộc `503` khi Redis chết.

### 4. Câu hỏi / ghi chú
- **Q-QC-P4-02** (trả lời của PM trong prompt: chi tiết thi công vào handoff): vẫn thiếu cách ép `low_confidence=true` cho từng intent và cách làm provider chết trên stack; TC-14 và ca `degraded` của TC-15 BLOCKED. `_test/llm/payloads` không có ở stack dựng bằng `scripts/dev.mjs` (image gateway không có `-tags testroutes`): `GET /api/v1/_test/llm/stats` → `404`. Cần handoff nêu cách dựng stack có route test, nếu muốn QC kiểm payload LLM ở pha sau.
- `docker`: `scripts/dev.mjs` với `EP_PORT_OFFSET=100` va cổng MinIO `9100` của stack khác (`llm-eval-backend`); QC đặt thêm `EP_PORT_MINIO=9300 EP_PORT_MINIO_CONSOLE=9301`. Seed dừng ở bước 11 (`fetch failed` — tài liệu cần `COMPOSE_PROFILES=ingest`, không bật), bước 1–10 xong (tài khoản, lớp, buổi học, bài thi); đủ cho các TC này.
- Dữ liệu QC thêm bằng SQL trên DB test (chỉ để dựng tiền điều kiện): đặt ticket `CLOSED` / `ANSWERED`, lùi `closes_at`, `ARCHIVED` lớp 2, gỡ / khôi phục `enrollments` của `sv30`, DB phụ (đã xoá). Stack `edupilot100` còn chạy cho `report-US-P4-02` và các story sau.
- Script chạy lại: `docs/sprints/7/qc/scripts/p4-01-ui.mjs` (TC-18…24).
