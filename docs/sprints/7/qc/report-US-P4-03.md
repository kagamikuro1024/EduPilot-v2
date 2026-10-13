# QC report — US-P4-03 (hộp thư giảng viên `/inbox`) — **vòng 2**
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `79a230d` (code vòng 2: `e017337`, `a606a29`, `4a7bae0`, `04a1bfb`, `5795075`). Bộ TC: `tc-US-P4-03.md` (không sửa chữ TC). Handoff: `dev-US-P4-03.md` mục "Vòng 2". Stack `edupilot100` dựng lại từ code mới (như `report-US-P4-02.md`).

## Kết luận: **FAIL** (1 lỗi: TC-16, mức THẤP–TRUNG BÌNH). BUG-2, BUG-3 vòng 1 đã sửa.
## 1. Kết quả vòng 2
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 16 người thua lượt nhận | **FAIL** (còn một phần) | Teacher và TA bấm `Nhận` cùng lúc (Playwright, `B:200 A:409`). Bên `409` nay có InlineNotice, nhưng chữ là **"Phạm Quốc Bảo đã nhận."** — thiếu "lúc HH:MM" dù body `409` có `details.claimed_at` (`2026-10-13T18:02:27Z`, `curl` xác nhận). Đồng thời hiện thêm một dòng chung **"Có người vừa sửa mục này. Chọn giữ bản nào. Chi tiết kỹ thuật"** (chữ kỹ thuật của khung xung đột chung, trùng nghĩa với thông báo đúng). Bên thắng: ô soạn hiện ngay (`textarea: 1`) — phần UX "phiếu không biến mất" đạt. AC4 / TC-16 đòi "đã nhận câu này lúc HH:MM" |
| 36 phím tắt | **PASS** | J/K, R, phím không kích hoạt khi gõ, Ctrl+Enter gửi rồi sang phiếu kế; tooltip nút `Gửi trả lời` = **"⌘↵ · Ctrl+↵"** (vòng 1: rỗng) |
| 42 cổng | **PASS** | `ui-antipatterns.sh` `rc=0`; e2e desktop `escalation, shell, class-join, private-chat, data-layer, a11y`: 141 passed, 5 skipped, 0 failed; `@/mock` trong `features/inbox` = 0 |
| 31, 32, 35, 38 | PASS (chạy lại) | nháp theo phiếu và sau tải lại; gửi lỗi 5xx giữ nháp / thành công xoá; cột danh sách 380 px; tải khung xương / lỗi "Thử lại" |
| 01…15, 17…30, 33, 34, 37, 39…41, E1, E2, P1 | PASS (giữ từ vòng 1) | không đổi mã backend của story; không chạy lại 33/34 (UI 375 px không đổi trong diff vòng 2 ngoài hành vi giữ phiếu) |

Tổng: 45 TC (42 + 3 nhánh): PASS 44, FAIL 1 (TC-16).

## 2. Lỗi
**BUG-1 (THẤP–TRUNG BÌNH) TC-16:** `InlineNotice` của người thua thiếu giờ nhận và đi kèm khung "Có người vừa sửa mục này… Chi tiết kỹ thuật". Tái hiện: hai tài khoản (teacher, ta) mở cùng phiếu `OPEN`, bấm `Nhận` cùng lúc; đọc `main`. Mong đợi: một dòng "{tên} đã nhận câu này lúc HH:MM" (giờ Việt Nam), không khung chung. Dev nói giờ lấy từ `details.claimed_at` — test mô phỏng của dev có trường đó nhưng chữ thật không có giờ; cần kiểm cách đọc `details` của `apiClient`.

## 3. Phụ lục: vòng 1 (đã đóng)
### Kết luận: **FAIL** (3 lỗi: TC-16, TC-36, TC-42)
Backend (danh sách, chi tiết, nhận, trả lời, đóng, PII, audit, idempotency, quyền) đạt toàn bộ. Hỏng ở giao diện: người thua lượt nhận không thấy thông báo, thiếu gợi ý phím trên nút Gửi, và cổng antipatterns / e2e đỏ.

### 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 danh sách phân trang | PASS | 227 `OPEN` (97 thật + 130 nhân bản SQL) → 5 trang × 50; không trùng; tập = DB; sắp theo `created_at`; `question_preview` ≤ 140 |
| 02 `limit` | PASS | `0` và `101` → 422; mặc định `status=OPEN` |
| 03 lọc trạng thái | PASS | `ANSWERED` chỉ 7 dòng `ANSWERED`; `ALL` có `OPEN` / `CLAIMED` / `ANSWERED` / `CLOSED` (7/8/7/256 qua phân trang) |
| 04 truy vấn | PASS | một câu `TicketListStaff`, keyset, không OFFSET, không N+1 (`log_statement`) |
| 05–07 `/me/tickets` | PASS | gộp 2 lớp, có `class_code`; TA chỉ lớp 1; ADMIN và STUDENT 403 |
| 08 UI "Tất cả lớp của tôi" | PASS | bộ chọn lớp có mục đó → gọi `/me/tickets`; dòng "Đang chờ · 761988" |
| 09 chi tiết | PASS | khoá `ticket`, `question`, `ai_answer`, `transcript`; tin liền trước / sau, phiên khác, phiên xoá mềm chứa `CANARY-9K3Z` → 0 trong chi tiết, danh sách, `/me/tickets` |
| 10–11 quyền | PASS | lớp lệch 404; STUDENT, ADMIN 403; TA ngoài lớp 403 |
| 12 audit | PASS | mở chi tiết 0 dòng; `claim`, `answer`, `dismiss` mỗi thao tác 1 dòng; không nội dung |
| 13 nhận | PASS | `200`, `CLAIMED`, `claimed_by` |
| 14 đua nhận | PASS | giảng viên và TA đồng thời: `200` + `409 TICKET_ALREADY_CLAIMED` (kèm `claimed_by`, `claimed_at`) |
| 15 nhận lặp | PASS | cùng người `200`, không đổi |
| 16 người thua trên UI | **FAIL** | hai trình duyệt cùng bấm `Nhận`: một `200`, một `409`. Bên `409`: không có InlineNotice "đã nhận lúc HH:MM"; chi tiết chỉ trở về "Chọn một câu hỏi để xem". Đo 150 ms và 2,5 s sau |
| 17 trả lời | PASS | `ANSWERED`; tin `ASSISTANT` / `STAFF_ANSWER` khối `kind:"ticket"`; outbox `escalation.answered`; phát lại cùng key trả y hệt |
| 18 sinh viên thấy | PASS | nhãn "Giảng viên · TS. Lê Thu Hà"; tin này không có nút Hữu ích |
| 19 trả lời phiếu `OPEN` | PASS | TA trả lời `OPEN` → tự nhận (`claimed_by` = TA), `ANSWERED`, nhãn `role:"TA"` |
| 20–22 key, kích thước | PASS | thiếu key → 422 `IDEMPOTENCY_KEY_REQUIRED`; nội dung rỗng / 4.001 ký tự → 422; 4.000 → 200 |
| 23 phiên xoá mềm | PASS | vào "Tin nhắn từ giảng viên"; `ticket.session_id` giữ nguyên |
| 24 LLM không thấy | PASS | payload `_test/llm/payloads` (dừng 1 gateway): lịch sử có, `CANARY-ANS-7` không |
| 25–27 xung đột | PASS | giảng viên trả lời phiếu TA đã nhận → 409 `claimed_by`; nhả của người khác 409; TA nhả 200, `waiting_since` giữ; trả lời lần hai → 2 tin, `answered_at` cập nhật, 2 thông báo; TA trên phiếu `ANSWERED` của giảng viên → 409 |
| 28–30 PII | PASS | tên / MSSV / email / CCCD của SV khác → 422 `PII_IN_ANSWER` chỉ có vị trí; không cờ bỏ qua; tên chính SV, email giảng viên, `phongdaotao` → 200. Ghi chú: "phần Hà" bị chặn vì trùng tên roster "Phan Thu Hà" (gập dấu) |
| 31 nháp theo phiếu | PASS | gõ T1, sang T2 trống, về T1 còn, tải lại còn; gửi lỗi 5xx → nháp còn, nút "Gửi lại" |
| 32 gửi | PASS | thành công → ô soạn đóng, nháp xoá |
| 33 375 px chuyển màn | PASS | danh sách toàn màn; chọn → chi tiết toàn màn (`?ticket=`); ô trả lời trong khung nhìn. Ghi nhận: không có nút "Quay lại" (`INFO`), dùng nút Back trình duyệt |
| 34 vùng chạm | PASS | không phần tử tương tác <44 px, không tràn ngang |
| 35 ≥1100 px chia đôi | PASS | grid `380px | 634px` |
| 36 phím tắt | **FAIL** | J / K đi lại, R vào ô soạn, phím không kích hoạt khi đang gõ, Ctrl+Enter gửi rồi sang phiếu kế: PASS. Tooltip phím trên nút "Gửi trả lời": rỗng (dev khai trong handoff là chưa làm; AC11 không có dòng "cắt" của PM) |
| 37 Hôm nay | PASS | `TICKET` mỗi lớp (chờ / bạn đã nhận / lâu nhất), `Href=/inbox?course=`; SV có `STAFF_ANSWER_UNREAD`; mục lớp mất ≤ 4 s sau khi xong hết |
| 38 trạng thái | PASS | tải → khung xương `aria-busy`; lỗi → "Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ." + "Thử lại"; rỗng → "Không có câu hỏi nào đang chờ." + "Xem tất cả" |
| 39 vai | PASS | STUDENT, ADMIN 403 mọi route; TA, TEACHER được |
| 40 gỡ mock | PASS | `grep "@/mock" frontend/src/features/inbox` = 0 |
| 41 phản hồi trên tin staff | PASS | `PUT feedback` → 422 `MESSAGE_NOT_RATEABLE`; cột feedback trống |
| 42 cổng | **FAIL** | xem mục 2 |
| E1 nhận phiếu `ANSWERED`/`CLOSED` | PASS | 409 |
| E2 gửi lại cùng key sau lỗi mạng | PASS | không tin đôi |
| P1 TA lớp khác | PASS | 403 |

Tổng 42 TC + 3 nhánh = 45: PASS 42, FAIL 3 (TC-16, 36, 42).

### 2. Lỗi
**BUG-1 (TRUNG BÌNH) TC-16:** người thua `409` không thấy "đã nhận lúc HH:MM" (AC4). Tái hiện: hai tài khoản (teacher và ta, cùng lớp) mở cùng phiếu `OPEN`, bấm `Nhận` cùng lúc; `scripts/p4-03-ui.mjs 16`.
**BUG-2 (THẤP) TC-36:** thiếu tooltip phím (AC11) trên nút "Gửi trả lời".
**BUG-3 (THẤP) TC-42:** `bash scripts/ui-antipatterns.sh` → `rc=1`: (a) `frontend/src/features/chat/RealChat.tsx:80` — chú thích chứa `#15a` bị bắt là "Màu viết cứng"; (b) `AppShell.module.css:124` (`font-size: 10px`, US-P4-02). e2e (`escalation, shell, class-join, private-chat, a11y`, desktop): 100 pass, 5 skip, 4 đỏ: `shell.spec.ts:312` (#18 đã ACCEPTED, test chưa sửa), `class-join.spec.ts:137,148` (#14 chưa quyết), `private-chat.spec.ts:235` (lệ thuộc giờ trong ngày: sau 00:00 giao diện in "hôm qua 21:21" thay "3 giờ trước"; không phải lỗi của nút).

### 3. Ghi nhận UX (không chấm)
Sau `Nhận` ở bộ lọc "Đang chờ", phiếu rời danh sách và chi tiết trở về trống; phải sang tab "Đã nhận" mới có ô soạn. `p4-03-ui.mjs` ghi lại luồng này.
