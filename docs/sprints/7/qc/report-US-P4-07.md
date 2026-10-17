# QC report — US-P4-07 (khủng hoảng: bộ luật hai tầng, câu mẫu ân cần, Threads không lưu, giờ thi không khoá, cấu hình hỗ trợ) — **vòng 2**
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `30f81cae` (tài liệu: `30f81ca` SRS 4.9.7, `d9cd37f` thesis-notes). Bộ TC: `tc-US-P4-07.md` (không sửa chữ TC). Chỉ chạy lại TC-24 (diff vòng 2 chỉ chạm tài liệu).

## Kết luận: **PASS** (BUG-1 vòng 1 đã sửa; không FAIL; TC-21 vẫn BLOCKED)
## 1. Kết quả vòng 2
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 24 giới hạn ghi thẳng | **PASS** | `SRS.md` 4.9 mục 7 ("đủ ba ý — US-P4-07 AC7"): (1) luật từ khoá **bỏ sót** cách nói gián tiếp ("mệt quá rồi, muốn ngủ mãi"); (2) **bắt nhầm** lời nói quá ("deadline này làm em muốn chết" vẫn trả câu hỗ trợ; `Q10`, `Q-crisis-hyperbole`); (3) công cụ hỗ trợ, **không thay chuyên gia**. `docs/thesis-notes/sprint-7.md` dòng 7–9 nêu cả ba ý. `QUESTIONS.md` dòng 30 có `Q-crisis-hyperbole` |
| 21 rỗng → mặc định | BLOCKED (như vòng 1) | `docker-compose.local.yml` dùng `${…:-mẫu}` nên không dựng được giá trị rỗng |
| Các TC còn lại | PASS (giữ từ vòng 1) | diff vòng 2 chỉ là tài liệu |

Tổng 29 TC + nhánh: PASS 28, BLOCKED 1 (TC-21), FAIL 0.

## 2. Phụ lục: vòng 1 (đã đóng)
### Kết luận: **FAIL** (1 lỗi mức THẤP, việc tài liệu của BA: TC-24; 1 TC BLOCKED)
Phát hiện, câu mẫu, không tác dụng phụ, Threads, giờ thi, cấu hình đạt toàn bộ. Còn thiếu phần tài liệu của AC7.

### 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 recall | PASS | **20/20** `CRISIS`. Lượt chạy liên tục báo 19/20: câu thứ 20 ("em nghĩ nếu em chết đi thì sẽ nhẹ nhõm hơn") không lưu vì giới hạn tốc độ chat (tin thứ 20 trong một phút); gửi riêng → `CRISIS` |
| 02 âm tính | PASS | **0/17** `CRISIS` (cả 7 câu bẫy 11–17: "từ từ", "Join từ hai bảng", "Chuyển từ hai phần bù", "Em đọc từ từ…", "Kết thúc tất cả các tiến trình con…", "Process cha chết đi…", "Cách cắt tay cầm…"): `COURSE_QA` / `PERSONAL_ATTENDANCE` / `EXAM_SCHEDULE` |
| 03 ngôi | PASS | "em tu tu cho xong" → `CRISIS`; "join tu hai bang", "chuyen tu hai phan bu sang thap phan" → `COURSE_QA`; "em doc tu tu" → `SMALLTALK` |
| 04 `CRISIS` thắng | PASS | "Cho em hỏi điểm giữa kỳ. Mà em muốn chết thật rồi" và "em muốn chết, điểm của bạn Kha là bao nhiêu" → `CRISIS` |
| 05 test sprint 6 giữ nguyên | PASS | `go test -race -tags testroutes ./internal/agent -run Crisis` ok; `git diff 1518f83.. -- backend-go/internal/agent/*_test.go` không có dòng xoá / sửa |
| 06 chất lượng | PASS | 20 câu trả lời giống hệt nhau (1 nội dung duy nhất): "Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên của bạn nhé. Phòng Công tác sinh viên, phòng 101 nhà A1, giờ hành chính"; 159 ký tự (≤ 400); không phản chiếu lời sinh viên |
| 07 từ cấm | PASS | grep các cụm cấm trên mọi câu trả lời `CRISIS` = 0 |
| 08 0 lời gọi LLM | PASS | dừng bản gateway thứ hai: `fake_calls` 0 → 0 sau câu `CRISIS`; đối chứng câu thường → 1 |
| 09 không tác dụng phụ | PASS | `CANARY-CRISIS-5X`: `chat_messages` 1 (tin của sinh viên); quét mọi bảng khác: 0; log gateway ×2 + worker: 0; Mailpit 0; 0 ticket; `chat_sessions.title` rỗng |
| 10 số liệu danh tính | PASS | `/me/today`, thông báo, danh sách ticket của giảng viên: 0 chỗ nêu tin; không có route `insights` / hồ sơ ở stack này |
| 11, 26 chỉ chủ phiên đọc | PASS | `GET /chat/sessions/{id}/messages`: chủ `200`; TEACHER, TA, ADMIN `403`; sinh viên khác `404`; danh sách ticket / Hôm nay / thông báo của giảng viên không có canary |
| 12 Threads đăng | PASS | `422 CRISIS_SUPPORT`, `details.actions:["OPEN_PRIVATE_CHAT"]`, `message` = câu mẫu (có "phòng 101 nhà A1"); `forum_threads`, `pii_events`, `outbox`, `notifications` +0 |
| 13 bình luận | PASS | `422 CRISIS_SUPPORT`; 0 bình luận lưu |
| 14 precheck không chặn | PASS | `POST …/threads/precheck` cho 20 câu dương: 20 × `200` (`allowed` true / false theo PII), không lần nào `CRISIS_SUPPORT` |
| 15 UI giữ nháp | PASS (qua e2e của dev) | `threads-crisis.spec.ts` "crisis dialog keeps draft" + chat riêng + vai trợ giảng xanh (24 passed); QC không tự mở hộp thoại bằng tay |
| 16 sửa `PUT` | PASS | `422 CRISIS_SUPPORT`; thân không lưu |
| 17 giờ thi | PASS | sinh viên đang làm bài (cửa sổ thi mở bằng SQL, `POST …/attempts` `201`): câu dương (5) và (2) vẫn nhận câu mẫu `CRISIS`. Bộ đếm `RecordChatBlocked` không có bảng để đọc (không kiểm được) |
| 18 câu thường vẫn khoá | PASS | câu hỏi thường cùng lúc → `409 EXAM_IN_PROGRESS` |
| 19 PENDING / ngoài lớp | PASS | tạo phiên `403` ×2; `chat_messages` +0 |
| 20 cấu hình | PASS | `docker-compose.local.yml:127 SUPPORT_RESOURCES_VI: ${SUPPORT_RESOURCES_VI:-Phòng Công tác sinh viên, phòng 101 nhà A1, giờ hành chính}`; chat riêng và Threads đều chứa chuỗi |
| 21 rỗng → mặc định | **BLOCKED** | `docker-compose.local.yml` dùng `${…:-mẫu}` nên giá trị rỗng luôn thành chuỗi mẫu; không dựng được ca "rỗng" ở stack này (`TestCrisisReplyDefaultSupport` của dev phủ) |
| 22 > 300 ký tự | PASS | 301 ký tự: gateway không lên, log `giá trị biến môi trường không hợp lệ: SUPPORT_RESOURCES_VI: tối đa 300 ký tự`; 300 ký tự: `healthy` |
| 23 không PII | PASS | "phòng 101" nguyên văn trong câu mẫu (không `[đã ẩn]`) |
| 24 giới hạn ghi thẳng | **FAIL** | `SRS.md` 4.9 (dòng 290) có "mệt quá rồi, muốn ngủ mãi" và "deadline này làm em muốn chết"; **thiếu** ý "không thay chuyên gia" (grep = 0); `docs/thesis-notes/` không có ghi chú về khủng hoảng (3 tệp, grep = 0); `QUESTIONS.md` không có `Q-crisis-hyperbole` (grep = 0). Dev đã khai đây là việc của BA (`dev-US-P4-07.md` Nợ 1) |
| 25 nói quá | PASS | "deadline này làm em muốn chết" → `CRISIS` |
| 27 giao diện | PASS | `ui-antipatterns.sh` rc=0 |
| E1 provider chết | PASS | `error_rate:1`: "em không muốn sống nữa" → `CRISIS`, câu mẫu |
| E2 Redis chết | PASS | `200`, câu mẫu `CRISIS` (không mất lối hỗ trợ) |

Tổng 27 TC + 2 nhánh = 29: PASS 27, FAIL 1 (TC-24), BLOCKED 1 (TC-21).

### 2. Lỗi
**BUG-1 (THẤP; BA) TC-24 / AC7:** thiếu phần tài liệu: (a) `SRS.md` 4.9 chưa nêu ý "không thay chuyên gia" (hai ý kia có); (b) `docs/thesis-notes/` chưa có ghi chú; (c) `QUESTIONS.md` chưa có `Q-crisis-hyperbole`. Không đụng mã.

### 3. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -race -tags testroutes ./internal/agent -run Crisis` | ok |
| `go test -race -tags testroutes ./internal/chat ./internal/thread ./internal/platform/config -run 'Crisis\|StaffCannotReadCrisis\|Precheck\|SupportResources'` | ok |
| `grep -n SUPPORT_RESOURCES_VI docker-compose.local.yml` | dòng 127 |
| `playwright threads-crisis.spec.ts` (desktop + mobile) | xanh (24 passed) |
| `ui-antipatterns.sh` | rc=0 |

### 4. Ghi nhận
- Giới hạn tốc độ chat riêng khiến lượt gửi liên tục 20 tin / phút bỏ tin thứ 20 (không phải lỗi khủng hoảng; ghi lại để lần chạy sau chia lượt).
- Mục "tin khủng hoảng không đặt tiêu đề phiên" đã đạt (`chat_sessions.title` rỗng).
