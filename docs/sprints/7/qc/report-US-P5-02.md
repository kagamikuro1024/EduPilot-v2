# QC report — US-P5-02 (điểm danh ≤ 60 s, phát biểu, 409, mất mạng, AI hỏi đáp) — **vòng 2 (AC14)**
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `2ad5764` (sửa vòng 2: `803393b`). Bộ TC: `tc-US-P5-02.md` (không sửa chữ TC). Handoff: `dev-US-P5-02.md` mục "Vòng 2". Stack `edupilot100` dựng lại từ code mới (DB trống, migration `14`, `--scale gateway=2`, provider `fake`). Dữ liệu sinh viên B (`sv.kha`): vắng 2 buổi (buổi 8, 10), 2 phát biểu = +0,50; ghi chú `CANARY-NOTE-9` ở hàng điểm danh.

## Kết luận: **PASS** (BUG-1 vòng 1 đã sửa; 46 TC + 2 nhánh không FAIL; TC-37 không áp dụng)
## 1. Kết quả chạy lại ở vòng 2
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 38 AI hỏi đáp khớp DB | **PASS** | đúng câu AC14 "Em đã nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu rồi ạ?": `blocks` = `[attendance {recorded 2, present 0, late 0, excused 0, absent 2, recent_absent[{buổi 10, 2026-10-14},{buổi 8, 2026-09-30}]}, participation {speak_count 2, points_total "0.50", recent[2 × SPEAK 0.25]}]` — khớp từng số với `v_attendance_summary` (2/0/2) và `v_student_360` (2 phát biểu, 0,50). Dữ kiện gửi LLM chứa cả `absent: 2` và `points_total: 0.50`; thứ tự khối cố định điểm danh → điểm cộng |
| 39 cập nhật | PASS | ghi thêm 1 phát biểu: cùng câu ghép → `points_total 0.75`; hỏi riêng từng ý đúng (`PERSONAL_ATTENDANCE` chỉ khối `attendance`, `PERSONAL_PARTICIPATION` chỉ `participation`) |
| 40 chưa có dữ liệu | PASS | sinh viên lớp 2 hỏi câu ghép: "Hệ thống chưa có dữ liệu điểm danh, điểm cộng của bạn.", `blocks []` |
| 41 hỏi hộ, ghi chú | PASS | `CANARY-NOTE-9` = 0 trong phản hồi, trong `chat_messages` và trong payload LLM; hỏi hộ "bạn Bùi Thanh Khải" → `OTHER_PERSON` "Mình chỉ xem được dữ liệu của chính bạn." |
| 01–37, 42–46, E1, E2 | PASS (giữ từ vòng 1) | không chạy lại: diff vòng 2 chỉ chạm `internal/agent` (phân loại / tool nhiều ý) |

## 2. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -race -tags testroutes ./internal/agent ./internal/chat` | ok (chat 43 s) |
| `go test -race -tags testroutes ./internal/crm` | ok |
| `playwright attendance.spec.ts private-chat.spec.ts` (desktop) | 35 passed, 0 failed |

## 3. Ghi nhận
- Số lời gọi sinh chữ của câu ghép ("đúng một lần") không đếm được trực tiếp: vòng đệm `_test/llm/payloads` là của từng bản gateway (`--scale gateway=2`).
- Mục 4 là báo cáo vòng 1 (đã đóng).

## 4. Phụ lục: vòng 1 (đã đóng)
### Kết luận: **FAIL** (1 lỗi: TC-38, mức TRUNG BÌNH)
Điểm danh (lưới, `PUT` một giao dịch, xung đột, hoàn tất / sửa sau, phát biểu, hàng đợi ngoại tuyến, đăng xuất, phân quyền) đạt toàn bộ. Hỏng ở AC14: câu hỏi mẫu của AC ghép "nghỉ mấy buổi" và "điểm phát biểu" chỉ trả lời nửa sau.

### 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 đọc lưới | PASS | `{session, students[30]}`; mỗi sinh viên `{id, full_name, student_code, record, speak_count}`; `record:null` khi chưa có; sắp theo tên không dấu (kiểm lại bằng gập dấu trong Python: đúng) và ổn định (3 lần cùng `md5`) |
| 02 không PENDING / REMOVED | PASS (một phần) | sinh viên `PENDING` (sv52) không có trong `students` (30 / 30). `REMOVED`: không dựng được trong DB (CHECK `enrollments`) nên không kiểm |
| 03 vai, lớp khác | PASS | `$SVA` 403, `$ADM` 403, buổi lớp 2 qua URL lớp 1 → 404; TA 200 |
| 04 ghi cả lưới | PASS | `200 {records:[{student_id,status,version}],done_at:null}`; 30 hàng `version=1`; 1 dòng `audit_log` (`save`, `entity_id=<session>`, `before {"counts":{}}` / `after` có `counts` và `changes[{student_id,from,to}]`, 0 tên); 1 outbox `crm.attendance_saved` |
| 05 lặp y hệt | PASS | `200`; `audit_log` +0; `version` giữ |
| 06 hàng giống hệt bị bỏ qua | PASS | gửi kèm `version` của 30 hàng, đổi 1: `200`; chỉ hàng đổi `version` 2; audit 1 `{student_id, from:PRESENT, to:LATE}` |
| 07 hiệu năng | PASS | 20 lần `PUT` 30 hàng, 3 hàng đổi mỗi lần: p50 8 ms, p95 13 ms, max 14 ms (≤ 500 ms) |
| 08 `409` xung đột | PASS | teacher đổi X (v2); TA `PUT` X (v1, EXCUSED) + hàng Y: `409 VERSION_CONFLICT`, `details.conflicts[{student_id, mine:{status,note}, current:{status,note,version,marked_by:{full_name},marked_at}}]`; hàng Y không ghi (`md5` toàn buổi không đổi) |
| 09 hàng không đổi không xung đột | PASS | TA gửi X trùng DB hiện tại (version cũ) → `200` |
| 10 UI xung đột | PASS | `scripts/p5-02-ui.mjs conflict`: "TS. Lê Thu Hà vừa đổi 1 dòng", `Giữ của tôi`, `Lấy của TS. Lê Thu Hà`; thay đổi riêng của TA giữ; chọn xong gửi lại "Đã lưu 03:37" |
| 11 dữ liệu sai | PASS | ngoài lớp / `PENDING` → `422 STUDENT_NOT_IN_COURSE` (`details.student_ids`); trùng `student_id` / `status:"X"` / `note` 201 ký tự / 1.001 hàng → `422 VALIDATION_FAILED` (`DUPLICATE`, `INVALID`, `TOO_LONG`, `TOO_MANY`); buổi lớp khác `404`; lớp `ARCHIVED` `409 COURSE_ARCHIVED` (cả `PUT` và `POST participation`; `GET` 200) |
| 12 DB chết | PASS | dừng PgBouncer: `503 SERVICE_UNAVAILABLE`; `md5` các hàng không đổi |
| 13 hoàn tất | PASS | `complete:true` với 2 hàng: `done_at`, `done_by` đặt; `audit_log.action='complete'`; 28 hàng `PRESENT` điền thêm (tổng 30) |
| 14 sửa sau | PASS | `edit_after_complete`, `changes[{from:PRESENT,to:ABSENT}]`; `done_at` không đổi (cả khi `complete:true`) |
| 15 autosave không hoàn tất | PASS | `complete:false` → `attendance_done_at` vẫn `NULL` |
| 16 375 px | PASS | danh sách 30 hàng; không tràn ngang; bấm hàng mở 4 nút `Có mặt / Muộn / Vắng phép / Vắng` cao 44 px (rộng 74), ≥ 44 px; không `Dialog`. Không kiểm zoom 200 % và màu chữ "Vắng" |
| 17 a11y | PASS | `a11y.spec.ts` (desktop) xanh (lượt 35 passed bên dưới) |
| 18 lưới ≥ 1100 px | PASS | 4 cột radio + "Phát biểu"; tiêu đề cột dính (`position: sticky`); `1`–`4` đặt rồi xuống hàng (hàng 0–2 Vắng, hàng 3 Muộn) |
| 19 30 SV bằng phím | PASS | gõ 30 phím trong 83 ms; lưu: DB 26 `PRESENT`, 1 `LATE`, 3 `ABSENT`; cả kịch bản + lưu ≈ 3 s |
| 20 lạc quan, hoàn tác | PASS | UI đổi sau 72 ms (gồm click); dòng "Đã đánh Vắng · {tên} · Hoàn tác"; 0 hộp thoại thêm; `Hoàn tác` trả "Có mặt" |
| 21 gom 2 giây | PASS | 3 thay đổi trong 1 s → đúng 1 `PUT` sau 2.003 ms, `complete:false`; "Đang lưu…" → "Đã lưu 03:34" |
| 22 5xx | PASS | `PUT` 500: "Chưa lưu được. Thử lại", lựa chọn giữ nguyên; bỏ chặn + `Thử lại` → "Đã lưu 03:35" |
| 23 `Lưu điểm danh` | PASS | 1 `PUT` `complete:true` cả lưới; "Đã lưu điểm danh · 30 sinh viên · vắng 3"; nút → "Đã lưu" khoá |
| 24 bộ chọn buổi | PASS | mặc định buổi 9 (buổi gần nhất đã qua; buổi 10 chưa bắt đầu); nhãn "Đã điểm danh" / "Chưa"; `?session=` chọn đúng buổi |
| 25 rỗng, đổi buổi | PASS | lớp mới: "Lớp chưa có lịch buổi học." + `Tạo lịch buổi học` (→ `/calendar`); đổi buổi khi còn thay đổi → `PUT` gửi trước khi đổi |
| 26 `+ phát biểu` | PASS | nhãn "+ phát biểu · +0,25"; bấm 1 lần rồi 3 lần = 4 `POST` với 4 `client_event_id` khác nhau, `type:"SPEAK"` |
| 27 idempotent, hoàn tác | PASS | cùng `client_event_id` → `200`, cùng `id`, 1 sự kiện; `DELETE` `204` |
| 28 lỗi | PASS | ngoài lớp, `PENDING` → `422 STUDENT_NOT_IN_COURSE`; `$SVA`, `$ADM` 403 |
| 29 60 s điện thoại | PASS (giả lập) | 375 px: Vắng 3, Muộn 1, 1 phát biểu, `Lưu điểm danh`: 4,4 s (≈ 8 chạm); 1 `PUT` `complete:true` với 30 hàng; 0 hộp thoại. **Giới hạn:** không có điện thoại thật (Q-QC-P5-03) — phần "Bạn tự kiểm" của chủ dự án |
| 30 kịch bản tự động | PASS | nằm trong `attendance.spec.ts` (xanh) |
| 31 mất mạng → hàng đợi | PASS | `setOffline`: "Chưa có mạng · 3 thay đổi chờ lưu"; có mạng → tự gửi, "Đã lưu"; DB 3 `ABSENT` |
| 32 tải lại giữ hàng đợi | PASS | khoá `ep:draft:<userId>:attendance:<course>:<session>` trong `localStorage`; tải lại: 3 thay đổi còn, banner còn |
| 33 `Lưu điểm danh` offline | PASS | xếp hàng cùng `complete:true`; có mạng → `done_at` đặt (DB) |
| 34 `409` là lỗi fatal | PASS | offline đánh 2 hàng; TA đổi cùng hàng qua API; có mạng → luồng xung đột ("vừa đổi", `Giữ của tôi`), 1 `PUT`, không thử lại; hàng riêng giữ |
| 35 đăng xuất hỏi trước | PASS | "Còn điểm danh chưa lưu · 2 thay đổi chưa lên máy chủ sẽ bị xoá khỏi máy này." + `Ở lại` / `Đăng xuất`; `Ở lại` giữ phiên và khoá |
| 36 máy dùng chung | PASS (một phần) | A xác nhận đăng xuất: kho không còn khoá `attendance:`; B đăng nhập: kho không có khoá của A; DB: 0 hàng của buổi đó (dữ liệu A không bị gửi bằng phiên B). Không kiểm `marked_by`=B của hàng mới |
| 37 AC13b | KHÔNG ÁP DỤNG | AC13 không bị cắt |
| 38 AI hỏi đáp khớp DB | **FAIL** | xem mục 2 |
| 39 cập nhật | PASS | thêm 1 phát biểu cho sinh viên B: hỏi "Em được cộng bao nhiêu điểm phát biểu rồi ạ?" → `points_total 0.75` (không cache cũ) |
| 40 chưa có dữ liệu | PASS | sinh viên lớp 2: "Hệ thống chưa có dữ liệu điểm danh của bạn.", `_test/llm/stats` `fake_calls` 8 → 8 (0 lời gọi LLM) |
| 41 hỏi hộ, ghi chú | PASS | ghi chú `CANARY-NOTE-9` ở hàng: 0 trong phản hồi, DB tin nhắn, payload LLM; hỏi hộ "bạn Bùi Thanh Khải" → `OTHER_PERSON` "Mình chỉ xem được dữ liệu của chính bạn."; "bạn Kha" (biệt danh không khớp roster) trả dữ liệu của chính người hỏi, không lộ người khác |
| 42 phân quyền | PASS | `GET`/`PUT` `attendance`, `POST participation`: STUDENT, ADMIN, `PENDING`, ngoài lớp → 403; TA, TEACHER 200 |
| 43 xoá phát biểu | PASS | TA xoá của teacher `403`; teacher xoá của TA `204`; người tạo xoá của mình `204` |
| 44 trạng thái | PASS | tải chậm: `aria-busy`, khung xương; `GET` 500: "Có lỗi xảy ra. Dữ liệu của bạn vẫn an toàn…" + `Thử lại` |
| 45 mock, build, cổng | PASS | `@/mock` trong `features/attendance` = 0; `pnpm lint` sạch; `pnpm build` rc=0; `ui-antipatterns.sh` rc=0 |
| 46 e2e | PASS | `attendance.spec.ts`, `exam-save-queue.spec.ts`, `a11y.spec.ts` (desktop): 35 passed, 0 failed |
| E1 mạng chậm hai người | PASS | như TC-10 / 34 |
| E2 `PUT` lặp không audit trùng | PASS | TC-05 |

Tổng 46 TC + 2 nhánh: PASS 45 (gồm TC-02, 36 một phần), FAIL 1 (TC-38), KHÔNG ÁP DỤNG 1 (TC-37).

### 2. Lỗi
**BUG-1 (TRUNG BÌNH) TC-38 / AC14 — câu hỏi ghép chỉ ra nửa sau.** Câu của AC14 "Em đã nghỉ mấy buổi, được cộng bao nhiêu điểm phát biểu rồi ạ?" (sinh viên B: vắng 2, 2 phát biểu = +0,50) được phân loại `PERSONAL_PARTICIPATION`: khối `blocks` chỉ có `participation` (`speak_count 2`, `points_total 0.50`, đúng), **không** có khối `attendance`, nên không trả "vắng 2 buổi". Hỏi riêng từng ý thì đúng và khớp DB: "Em đã nghỉ mấy buổi?" → `PERSONAL_ATTENDANCE` `{recorded 3, present 1, absent 2, recent_absent[buổi 10, buổi 8]}` = `v_attendance_summary`; "Em được cộng bao nhiêu điểm phát biểu rồi ạ?" → `{speak_count 2, points_total "0.50"}`. Tái hiện: sinh viên có điểm danh và phát biểu, mở `/chat`, gửi đúng câu AC14, đọc `chat_messages.blocks` của câu trả lời. AC14 yêu cầu cả hai khối cho một câu hỏi này; luật tốc độ D47 (phân loại một lần mỗi tin) có thể buộc chọn một intent — nhờ PM / BA quyết (sửa định tuyến hoặc sửa chữ AC14).

### 3. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -race -tags testroutes ./internal/crm` | ok (234 s) |
| `go test -race -tags testroutes ./internal/contract/... ./internal/httpapi/... ./internal/agent` | ok ở lượt chạy riêng (contract 118 s, httpapi, sse, agent). Hai lượt trước đỏ rải rác (`TestChatThreadsContract` 121 s, `TestRouteSpecParity`, `TestContract_ResponsesMatchSpec`, `TestSpec_EveryDocumentedStatusExercised` …) khi máy QC đang chạy stack 13 container và các lệnh khác; chạy riêng hai test đầu `ok`. Ghi nhận như nhiễu tải, không chấm |
| `sqlc diff` | rc=0 |
| `playwright attendance.spec.ts exam-save-queue.spec.ts` (+ `a11y.spec.ts`) desktop | 35 passed |
| `ui-antipatterns.sh` | rc=0 |

### 4. Ghi nhận (không làm FAIL)
- `TestSchemaAttendance` đỏ ngẫu nhiên (xem `report-BUG-S7-01.md`); dev cũng ghi trong handoff.
- Lưới: "MSSV rút gọn" của TC-16 — giao diện hiện đủ 8 chữ số.
- Số `PUT` khi tự lưu: mỗi thay đổi ngoại tuyến ra một `PUT` riêng khi có mạng (3 thay đổi = 3 `PUT`, theo thứ tự).
- `ATTENDANCE_SEED` còn dùng bởi màn khác (dev đã khai).
