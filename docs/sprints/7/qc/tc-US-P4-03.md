# QC test case — US-P4-03 (hộp thư hỗ trợ `/inbox`: danh sách, chi tiết, nhận, trả lời, bỏ nhận)
Nguồn: `US.md` US-P4-03 AC1…AC16 + `SRS.md` 2, 3.3, 4.2, 4.3, 6.2, 7.1; `DESIGN.md` §14.5; `UX.md` mục 5, 6. Hộp đen, không đọc code dev.
Biến và quy ước như `tc-US-P4-01.md`. Dữ liệu seed: lớp 1 có 5 ticket (`OPEN` mới, `OPEN` quá hạn, `CLAIMED` bởi teacher, `ANSWERED`, `CLOSED`, SRS 9.4); lớp 2 rỗng ticket. `$T` = id ticket `OPEN` của `$SVA`. `idem` = `Idempotency-Key: k-$RANDOM`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P4-03-01 | AC1 | ≥ 120 ticket lớp 1 (gieo bằng `$PSQL generate_series` hoặc API) | `j "$Q/tickets?status=OPEN&limit=50" -H "$TCH"`, đi hết `next_cursor` | `{items,next_cursor}` sắp `(created_at,id)`; không trùng, không sót (so `select count(*)`); mỗi dòng có `id,status,kind,student{id,full_name},question_preview(≤140),age_minutes,claimed_by,sla_stage,course{id,class_code}`; không có nội dung đầy đủ |
| TC-P4-03-02 | AC1 | như trên | `limit=101`; `limit=0`; không `status` | `limit` bị chặn ≤ 100 (hoặc `422`, ghi lại); mặc định `OPEN` |
| TC-P4-03-03 | AC1 | có ticket `ANSWERED` và `CLOSED` | `status=ANSWERED`; `status=ALL`; `status=CLAIMED` | `ANSWERED` chỉ `ANSWERED` (không `CLOSED`); `ALL` gồm cả `CLOSED`; `CLAIMED` đúng |
| TC-P4-03-04 | AC1 (không N+1, phân trang con trỏ) | 1.000 ticket | `EXPLAIN (ANALYZE)` hoặc đếm câu SQL (log) khi gọi `limit=100` | tối đa vài câu SQL (không 100+); không `OFFSET` (`grep -i offset` trong log) |
| TC-P4-03-05 | AC2 | giảng viên T phụ trách lớp 1 và 2 (teacher ở cả hai) | `j "$GW/api/v1/me/tickets" -H "$TCH"` | ticket của cả hai lớp, mỗi dòng `course.class_code`; không có ticket của lớp mình không thuộc |
| TC-P4-03-06 | AC2 | TA chỉ thuộc lớp 1 | `GET /me/tickets -H "$TA_"` | chỉ lớp 1 |
| TC-P4-03-07 | AC2 | – | `GET /me/tickets -H "$ADM"`; `-H "$SVA"` | `403` cả hai |
| TC-P4-03-08 | AC2 (UI) | `playwright-cli`, giảng viên 2 lớp | `/inbox` → chọn "Tất cả lớp của tôi" | mỗi dòng ghi mã lớp; bộ chọn lớp riêng của `/inbox` |
| TC-P4-03-09 | AC3 | ticket `$T` | `GET $Q/tickets/$T -H "$TCH"` | `{ticket, question, ai_answer:{content,citations,blocks,confidence}, transcript}` đúng phạm vi US-P4-01 AC10 (grep canary = 0) |
| TC-P4-03-10 | AC3 | ticket của lớp 2; teacher đang ở lớp 1 | `GET $GW/api/v1/courses/$C2/tickets/$T2 -H "$TCH"` nơi teacher không thuộc lớp 2; và `GET $Q/tickets/$T2` (lớp lệch) | `404` |
| TC-P4-03-11 | AC3 | – | `GET $Q/tickets/$T -H "$SVA"`; `-H "$ADM"` | `403` |
| TC-P4-03-12 | AC3 (audit) | `select count(*) from audit_log where entity='escalation_ticket'` trước | mở chi tiết 3 lần; `claim`; `answer`; `dismiss` (ticket khác) | mở chi tiết **không** thêm dòng; `claim`, `answer`, `dismiss` mỗi hành động 1 dòng kèm `ticket_id`, **không** kèm nội dung (grep canary = 0) |
| TC-P4-03-13 | AC4 | `$T` `OPEN` | `$TCH` `POST $Q/tickets/$T/claim` | `200`, `status=CLAIMED`, `claimed_by`=teacher, có `claimed_at` |
| TC-P4-03-14 | AC4 (đua) | `$T2` `OPEN` | `$TCH` và `$TA_` gọi `claim` đồng thời (`&` + `wait`) | đúng 1 `200`; người kia `409 TICKET_ALREADY_CLAIMED` kèm `details.claimed_by{id,full_name,role}` và `claimed_at` |
| TC-P4-03-15 | AC4 (idempotent) | `$T` do teacher đã nhận | teacher `claim` lại | `200`, không đổi `claimed_at` |
| TC-P4-03-16 | AC4 (UI hai ngữ cảnh) | `playwright-cli`: ngữ cảnh 1 teacher, ngữ cảnh 2 TA cùng mở `$T3` | bấm `Nhận` gần như đồng thời (script) | người thua thấy nút `Nhận` hoàn lại + `InlineNotice` một dòng "Cô … đã nhận câu này lúc HH:MM" |
| TC-P4-03-17 | AC5 | `$T` `CLAIMED` bởi teacher | `$TCH` `POST $Q/tickets/$T/answer {"body":"Chào em, …"} -H "$idem"` | `200 {ticket:{status:ANSWERED}, message}`; `select role,intent,blocks from chat_messages where id=…` → `ASSISTANT`, `STAFF_ANSWER`, `blocks` có `kind:"ticket"`, `ticket_id`, `answered_by{full_name,role}`; `answered_by`, `answer`, `answered_at` đã ghi; phiên nổi lên đầu `GET /chat/sessions`; có dòng outbox `escalation.answered` |
| TC-P4-03-18 | AC5 (giao diện sinh viên) | `$SVA` mở `/chat` | quan sát tin | nhãn "Giảng viên · {tên}" (hoặc "Trợ giảng · {tên}"), **không** nhãn AI, không nút Hữu ích / Không hữu ích |
| TC-P4-03-19 | AC5 (trả lời tự nhận) | `$T4` `OPEN` (chưa nhận) | `answer` | `200`, `ANSWERED`, `claimed_by` = người trả lời (tự nhận) |
| TC-P4-03-20 | AC5 (idempotent) | `$T5` | `answer` hai lần cùng `Idempotency-Key` | cùng phản hồi; đúng 1 tin `STAFF_ANSWER`; không thêm dòng outbox |
| TC-P4-03-21 | AC5 (thiếu key) | – | `answer` không `Idempotency-Key` | lỗi (route bắt buộc key — ghi mã thực tế, kỳ vọng `400/422`); không tạo tin |
| TC-P4-03-22 | AC5 (độ dài) | – | `body:""`; `body` 4.001 ký tự; 4.000 ký tự | `422`; `422`; `200` |
| TC-P4-03-23 | AC5 (phiên đã xoá mềm) | `$SVA` xoá mềm phiên của ticket | giảng viên `answer` | tin vào phiên "Tin nhắn từ giảng viên" (tạo-hoặc-lấy), `ticket.session_id` không đổi; giảng viên vẫn đọc ngữ cảnh; `$SVA` thấy tin, `Hỏi tiếp` / `Đã rõ` bấm được ở phiên đó |
| TC-P4-03-24 | AC5 (không vào lịch sử LLM) | phiên có `STAFF_ANSWER`; `LLM_PROVIDER=fake` ghi payload (`GET $GW/api/v1/_test/llm/stats`) | `$SVA` hỏi tiếp một câu thường; kiểm payload gửi LLM | không có nội dung `STAFF_ANSWER` / `ESCALATION_FOLLOWUP` trong lịch sử gửi |
| TC-P4-03-25 | AC6 | `$T6` `CLAIMED` bởi TA | `$TCH` `answer` | `409 TICKET_ALREADY_CLAIMED` kèm người đã nhận |
| TC-P4-03-26 | AC6 (bỏ nhận) | `$T6` do TA nhận | `$TCH` `release` → `409`; `$TA_` `release` → `200 OPEN`; `select waiting_since` trước/sau | teacher `409`; TA `200`; `waiting_since` **không đổi** |
| TC-P4-03-27 | AC7 | `$T` `ANSWERED` bởi teacher | teacher `answer` thêm; TA `answer` thêm | thêm tin `STAFF_ANSWER` mới, ticket giữ `ANSWERED`, `answered_at` đổi; TA `409`; sinh viên nhận thông báo mới (không gộp, `TICKET_ANSWERED` thêm 1 dòng) |
| TC-P4-03-28 | AC8 (tên SV khác) | lớp có SV `Trần Thị Kha` (MSSV `M2`) và SV hỏi `Nguyễn Gioi` | `answer` với body chứa "bạn Trần Thị Kha" | `422 PII_IN_ANSWER` + `details:[{start,end,kind}]` (không lặp giá trị); ticket không đổi; không có lối ghi đè (thử `redact`/`force:true` → vẫn `422`) |
| TC-P4-03-29 | AC8 | – | body chứa MSSV `M2`; email SV khác `sv.kha@edupilot.local`; CCCD 12 số hợp lệ | `422 PII_IN_ANSWER` (3 ca) |
| TC-P4-03-30 | AC8 (được phép) | – | body chứa tên của chính SV hỏi; email giảng viên `teacher@edupilot.local`; `phongdaotao@edupilot.local`; từ thường trùng tên (ví dụ "Hà", "Anh" trong câu thường) | `200` cả bốn |
| TC-P4-03-31 | AC9 | `playwright-cli` giảng viên | gõ dở ở ticket T1 → chuyển T2 → quay T1; tải lại; đóng trình duyệt rồi mở; chờ ≥ 2 s giữa lần gõ | nháp T1 còn nguyên ở mọi bước; nháp T2 riêng |
| TC-P4-03-32 | AC9 (gửi thành công / lỗi) | – | gửi thành công; rồi `page.route` trả 500 | thành công → nháp xoá; lỗi → nháp giữ, nút `Gửi trả lời` hoàn lại + `Thử lại`; nháp không ra khỏi máy (không request chứa nháp trước khi gửi: theo dõi `network`) |
| TC-P4-03-33 | AC10 (375 px) | `playwright-cli resize 375 812` | `/inbox` danh sách → bấm một dòng → chi tiết → Quay lại | danh sách toàn màn hình; chi tiết toàn màn hình (`?ticket=` trong URL); Quay lại về danh sách giữ bộ lọc và vị trí cuộn; ô trả lời dính đáy |
| TC-P4-03-34 | AC10 (đo) | như trên | `eval` đo mọi nút/dòng `getBoundingClientRect()`; `document.documentElement.scrollWidth` | vùng chạm ≥ 44 px; `scrollWidth ≤ innerWidth` |
| TC-P4-03-35 | AC10 (≥ 1100 px) | `resize 1280 800` | mở `/inbox` | chia đôi: danh sách rộng 380 px + chi tiết |
| TC-P4-03-36 | AC11 (cắt được) | `/inbox` chia đôi | `J`/`K`; `R`; `⌘↵`/`Ctrl+↵`; gõ `J` khi đang ở ô trả lời | `J`/`K` chuyển ticket; `R` focus ô trả lời; `Ctrl+↵` gửi rồi sang ticket kế; gõ trong ô nhập không kích hoạt phím tắt; tooltip phím trên `Gửi trả lời`. **Nếu dev ghi "Nợ" cắt AC11 trong `report.md`: TC này "KHÔNG ÁP DỤNG", không tính FAIL** |
| TC-P4-03-37 | AC12 | có 2 ticket `OPEN` (1 do teacher nhận) | `GET $GW/api/v1/me/today -H "$TCH"` (hoặc `$Q/today`) | mục `TICKET` mỗi lớp "N câu hỏi đang chờ · M câu bạn đã nhận · lâu nhất {Age}", `Href` `/inbox?course=…`; xuất hiện ≤ 5 s sau khi ticket tạo; biến ≤ 60 s sau khi xử lý hết; `CLAIMED` bởi người khác không tính vào "đang chờ" của mình; `$SVA` không thấy mục này |
| TC-P4-03-38 | AC13 | `playwright-cli` | `/inbox` lúc tải (chậm mạng 3G); rỗng (lớp 2: "Không có câu hỏi nào đang chờ." + nút `Xem tất cả`); lỗi (`page.route` 500: nói chuyện gì xảy ra + nút `Thử lại`); chuyển `OPEN` → `ANSWERED` → `ALL` | khung xương khi tải; rỗng đúng chữ; lỗi đúng; chuyển trạng thái không nhảy layout (đo CLS ≈ 0); trạng thái có chữ, không chỉ màu |
| TC-P4-03-39 | AC14 | – | ma trận: mỗi vai `$SVA`, `$SVB`, `$ADM`, `$TA_`, `$TCH` × route `list`, `get`, `claim`, `release`, `answer`, `dismiss`, `knowledge-preview`, `knowledge`, `/me/tickets` | STUDENT, ADMIN → `403` mọi route; TA có `list,get,claim,answer,release,dismiss,knowledge`; chi tiết ticket của lớp mình không phụ thuộc người nhận |
| TC-P4-03-40 | AC15 | repo | `grep -rn "@/mock" frontend/src/features/inbox \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint` | `0`; xanh. Chế độ demo của `/chat`, `/threads` vẫn chạy (mở `/chat`, `/threads`) |
| TC-P4-03-41 | AC16 | `STAFF_ANSWER` mid `$SM` | `POST /chat/sessions/$S/messages/$SM/feedback -H "$SVA"`; sinh lại/thử lại (`attempt`) trên `$SM` | `422 MESSAGE_NOT_RATEABLE`, không ghi gì |
| TC-P4-03-42 | UI | frontend | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW a11y.spec.ts -g inbox` | `0`; 0 vi phạm nghiêm trọng |

## Nhánh lỗi (SRS 3.3)
TC-P4-03-14 (hai người nhận), 21, 22, 23 (phiên xoá), 28–30 (PII), 32 (gửi lỗi mạng), 41. Thêm: TC-P4-03-E1 `claim` ticket đã `ANSWERED`/`CLOSED` → `409 TICKET_STATE_INVALID`; `answer` ticket `CLOSED` (kỳ vọng `409`, ghi mã thực tế); TC-P4-03-E2 `answer` lỗi mạng giữa chừng rồi gửi lại cùng key → không tin đôi.

## Phân quyền
TC-P4-03-07, 10, 11, 39 và TC-P4-03-P1: teacher của lớp 1 mở ticket lớp 2 → `404`; TA lớp 1 `answer` ticket lớp 2 → `404`; `$SVB` gọi mọi route staff → `403`.

## Câu hỏi (Q-QC)
- **Q-QC-P4-07** AC5: `Idempotency-Key` bắt buộc (SRS 6.2 dòng 9) nhưng mã lỗi khi thiếu không nêu. TC-21 ghi nhận mã thật; PM/BA có muốn chốt (`400 IDEMPOTENCY_KEY_REQUIRED`?).
- **Q-QC-P4-08** AC4 trả `409` khi cùng người ở hai tab: đã `200` idempotent. OK.
- **Q-QC-P4-09** AC1: `limit` vượt 100 chặn (`min`) hay `422`? Chấp nhận cả hai, ghi hành vi.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
