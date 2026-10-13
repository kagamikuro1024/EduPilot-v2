# Sprint 7: P4 Escalation + Mail, P5 CRM + Hồ sơ 360

**Trạng thái:** **chủ dự án duyệt 2026-10-11.** Từ 2026-10-13 chủ dự án giao PM toàn quyền chốt các câu [CHỦ DỰ ÁN] của sprint này; PM chọn phương án an toàn, ghi lý do trong `QUESTIONS.md` / `proposals.md` và liệt kê lại trong `report.md` để chủ dự án xem.
**Nhánh và nơi làm:** `sprint/7-p4-p5` tách từ `main` (`628a79b`), worktree `../TA_Agent_v2-s7`.

## Mục tiêu
- **P4 (M2, M3):** AI chưa chắc thì giảng viên được báo ngay. Giảng viên trả lời trong app; sinh viên nhận câu trả lời trong chat riêng, qua chuông và qua mail. Ticket không nằm im quá 72 giờ. Threads có kiểm duyệt.
- **P5 (M5, M6, M13-C):**
  - giảng viên điểm danh 30 sinh viên ≤ 60 s trên điện thoại;
  - giảng viên xem được hồ sơ 360 của từng sinh viên;
  - sinh viên hỏi AI ra đúng số chuyên cần, phát biểu của mình.
- **Luồng đi trọn:** F4 (Threads + kiểm duyệt), F5 (escalation), F7 (điểm danh), F8 (theo dõi sinh viên).
- **Cổng:** `gate P4`, `gate P5`.
- **Quyển đồ án:** sau khi merge, `writer` viết phần sprint 7; PM + BA review.

P4 và P5 độc lập nhau (ROADMAP "Vì sao ghép"), trừ một chỗ: "Nhắn sinh viên" từ hồ sơ 360 cần ticket `TEACHER_OUTREACH` của P4. Vì vậy P4 làm trước, P5 theo sau.

## Story (lát dọc)

| # | Story | Lát | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 0 | **US-CI-02** TL-CI #3 (shard Playwright 2 phần + chia Lighthouse) và #5 (matrix Go theo nhóm gói). Không bỏ test, `-race`, cổng hay lượt Lighthouse nào; đo lại thời gian CI trước / sau | nợ sprint 6 | S | – |
| 1 | **US-P4-01** Migration `00012_escalation_notify`: `escalation_tickets` (`kind` QUESTION / GRADE_APPEAL / TEACHER_OUTREACH, `appeal_payload`, `version`), `post_reports`. `internal/escalation`: tạo idempotent theo `message_id`, trạng thái OPEN → CLAIMED → ANSWERED → CLOSED, tự đóng sau 7 ngày. Kích hoạt khi agent đặt `should_escalate` hoặc `POST /chat/messages/{mid}/escalate`. **Hiện nút "Nhờ giảng viên"** trong `/chat`; sinh viên thấy trạng thái + thời gian đã chờ. Ghi ticket + outbox cùng transaction | P4 L1 | M | – |
| 2 | **US-P4-02** `internal/notify`: SSE `/notifications/stream`, phát qua Redis `PUBLISH notify:<user_id>`, `Last-Event-ID` đọc bù từ bảng, gộp thông báo cùng loại; chuông ở header chuyển từ làm mới 30 s sang đẩy tức thời. `TestNotifyAcrossTwoGateways` | P4 L1 | M | 1 |
| 3 | **US-P4-03** `/inbox` (thay mock, `DESIGN.md` §14.5): danh sách 380 px + hội thoại; lọc Open / Claimed / Answered / All; "Tất cả lớp của tôi"; `Nhận` (người sau nhận 409 và thấy ai đã nhận); trả lời có `Idempotency-Key` → vào chat riêng + chuông; nháp tự lưu theo ticket; 375 px (danh sách → chi tiết). Provider `TICKET` cho "Hôm nay". Phím `J/K/R/⌘↵` | P4 L3 | L | 1, 2 |
| 4 | **US-P4-04** Mail: template trả lời escalation (câu hỏi gốc + trả lời + link), nhắc ticket chưa ai nhận, có trả lời trong thread của bạn; cài đặt "mail báo ticket mới" của giảng viên (mặc định tắt). Lỗi mail thử 3 lần rồi hiện ở `/observability`. `TestEscalationToMail`, `TestOutboxSurvivesWorkerKill` | P4 L2 | M | 3 |
| 5 | **US-P4-05** Vòng đời ticket phía sinh viên + SLA: sinh viên hỏi tiếp (mở lại) hoặc xác nhận đã rõ → đóng. Job SLA: OPEN quá 24 giờ làm việc → nhắc cả nhóm; quá 72 giờ → lên đầu "Hôm nay" kèm số giờ chờ + báo Admin khi lớp không ai hoạt động. "Lưu thành tri thức" (che PII rồi nạp vector DB + semantic cache). Thanh trượt ngưỡng escalation trong cài đặt lớp | P4 L2b, L3 | M | 3, 4 |
| 6 | **US-P4-06** Kiểm duyệt Threads: `Báo cáo` (lý do chọn sẵn; 3 báo cáo → tự ẩn chờ duyệt), `Ẩn bài` kèm lý do, chủ bài sửa / xoá (sửa chạy lại tường lửa PII); người đăng được báo khi có trả lời và khi được xác nhận; thông báo câu trả lời AI chờ duyệt; ghim; lọc. Provider "bài bị báo cáo". **Nạp lại tri thức đã xác nhận / đã sửa** vào vector DB, bài `Loại` ra khỏi RAG (nợ sprint 6) | P4 L2b, L3 | L | 2 |
| 7 | **US-P4-07** Tin nhắn khủng hoảng (F3): kiểm lại bộ phân loại an toàn ở preflight đã có từ sprint 6 → câu trả lời ngắn, ân cần + `SUPPORT_RESOURCES_VI` + gợi ý nói chuyện với giảng viên; KHÔNG tự báo cho ai. Test 20 câu mẫu: không từ chối lạnh lùng, không tư vấn chuyên môn. Phần nào sprint 6 đã làm thì story chỉ bổ sung chỗ thiếu | P4 L2b | S | – |
| 8 | **US-P5-01** Migration `00013_crm`: `attendance_records` (`version`), `participation_events`, `student_observations`, view `v_student_360`, `v_attendance_summary`. Tạo lịch buổi học: `POST …/sessions/generate` (thứ, giờ, phòng, khoảng ngày, ngày loại trừ) → xem trước → tạo hàng loạt; sửa / huỷ / buổi bù; một bước của "Thiết lập lớp mới". Provider "buổi đang diễn ra chưa điểm danh" | P5 L1 | M | – |
| 9 | **US-P5-02** `/attendance` (thay mock, `UX.md` 1 + 5, `DESIGN.md` §14.8): mặc định Có mặt, chạm người vắng; điện thoại = hàng ngăn đường kẻ, vùng chạm ≥ 44 px; máy tính = `AttendanceGrid` phím 1–4, header dính; tự lưu lạc quan + Hoàn tác tĩnh lặng; `Lưu điểm danh`; "+ phát biểu"; `PUT …/attendance` một transaction + audit; 409 khi xung đột; mất mạng xếp hàng cục bộ rồi tự đồng bộ. Nối dây thật `get_my_attendance`, `get_my_participation` | P5 L1 | L | 8 |
| 10 | **US-P5-03** Thời gian học: migration `00015_observation` (#25) (`study_time_daily`, `insight_reports`); `POST /activity/heartbeat` chỉ `INCRBY` Redis, worker `heartbeat.rollup` ghi DB mỗi phút, chống đếm đôi theo `user_id` + khung 30 s; hook `useActivityHeartbeat(area)` (Page Visibility + idle 2 phút) + hộp thoại minh bạch lần đầu. Test kiến trúc: `internal/grade` không import `activity` | P5 L2 | M | – |
| 11 | **US-P5-04** `/students` (thay mock): `<DataTable>` phân trang con trỏ, cột rủi ro + thời gian học, lọc "Cần chú ý"; `EXPLAIN` trên 1.000 SV không seq scan bảng lớn. At-risk: chuyên cần + điểm + hoạt động + thời gian học dưới ngưỡng 2 tuần liền, job đêm, ngưỡng trong cài đặt lớp. Provider `STUDENT_ATTENTION` (95) + "Lớp cần chú ý" (nợ 5.5 #2) | P5 L3 | M | 8, 10 |
| 12 | **US-P5-05** `/students/[id]` (thay mock, `DESIGN.md` §14.7): câu nhận định rủi ro bằng lời, tab Tổng quan / Chuyên cần / Điểm / Hoạt động học / Ghi chú; mở hồ sơ ghi `audit_log`; ghi chú riêng tư (STUDENT → 403); "Nhận xét tổng hợp" AI (không gửi tên / MSSV), sửa rồi lưu; `Nhắn sinh viên` mở ticket `TEACHER_OUTREACH` | P5 L3 | L | 1, 3, 11 |
| 13 | **US-P5-06** `/me` của sinh viên (thay mock, `DESIGN.md` §14.9): chuyên cần + điểm cộng phát biểu; thời gian học là xu hướng nhỏ cuối trang; không streak, không xếp hạng, không ghi chú / nhãn rủi ro | P5 L3 | S | 9, 10 |
| 14 | **US-S7-SEED** Seed: điểm danh, phát biểu, 10 tuần thời gian học, ticket ở đủ trạng thái, bài bị báo cáo. `escalation.spec.ts`, `attendance.spec.ts` (có bước ngắt mạng), `gate-p4.sh`, `gate-p5.sh` | – | M | 1–13 |

**Thứ tự cho `dev`:** 0 → 1 → 2 → 3 → 4 → 5 → 8 → 9 → 6 → 7 → 10 → 11 → 12 → 13 → 14. Story 8, 9 (điểm danh) đưa lên sớm vì có bước "bạn tự kiểm" trên điện thoại thật cần chủ dự án.

**Spec do BA viết:**
- `docs/specs/FEAT-escalation-notify/`: P4, gồm cả kiểm duyệt Threads và tin nhắn khủng hoảng.
- `docs/specs/FEAT-crm-360/`: P5.

BA viết FEAT-escalation-notify trước để dev bắt đầu sớm. Tech Lead thẩm định từng feature ngay khi BA xong. Story 0 không cần spec sản phẩm: TC lấy từ `techlead.md` TL-CI. TC ở `docs/sprints/7/qc/`.

## Quyết định PM tự chốt (đổi được qua `proposals.md`)
- **Số migration:** phase file ghi `00007` / `00008` / `00009` nhưng các số đó đã dùng hoặc bỏ trống. Sprint này dùng `00012_escalation_notify`, `00013_crm`, `00014_observation`.
- **`mail_inbound` không tạo ở sprint 7:** D6 chốt không nhận reply qua hộp thư. Bảng này phục vụ nhận bài qua mail ở P7, nên P7 tạo.
- **"Giờ làm việc" cho SLA 24 giờ:** BA đề xuất định nghĩa trong SRS (mặc định gợi ý: 8:00–17:00 thứ Hai–thứ Sáu, giờ Việt Nam); câu nào đụng hành vi sản phẩm thì vào `QUESTIONS.md`.
- **Mặc định mọi test dùng provider giả**; không gọi LLM thật trừ khi chủ dự án cho phép.

## Nợ đem vào
- **Sprint 6:** TL-CI #3, #5 (story 0); nút "Nhờ giảng viên" (story 1); nạp lại tri thức đã xác nhận (story 6).
- **5.5 #2:** "Lớp cần chú ý" trên "Hôm nay" (story 11).
- **PROGRESS, nguồn việc "Hôm nay":** P4 `TICKET` (10), bài bị báo cáo; P5 buổi chưa điểm danh, `STUDENT_ATTENTION` (95).

## Cắt được nếu trễ (ROADMAP, WORKFLOW §6)
#5 hàng đợi ghi cục bộ của điểm danh (mất mạng thì chặn lưu + báo rõ thay vì xếp hàng); #6 phím tắt `/inbox`. Không cắt gì khác khi chưa hỏi chủ dự án.

## Rủi ro
- **Dữ liệu cá nhân:** hồ sơ 360, ghi chú quan sát, thời gian học là vùng nhạy cảm nhất từ đầu dự án. Luật cứng: sinh viên không thấy ghi chú / nhãn rủi ro của mình; mọi lần mở hồ sơ ghi `audit_log`; giảng viên chỉ đọc chat riêng khi đã escalate; thời gian học không vào `internal/grade`.
- **Mail:** chỉ Mailpit ở local; không gửi mail thật.
- **SSE nhiều bản gateway:** thông báo tức thời phải chạy với `--scale gateway=2`; QC chạy `TestNotifyAcrossTwoGateways`.
- **RAM:** một stack mỗi lúc; docling chỉ bật khi test tài liệu (story 5, 6 cần nạp tri thức).
- **Điện thoại thật:** bước "bạn tự kiểm" của P5 cần chủ dự án bấm giờ trên điện thoại thật.
