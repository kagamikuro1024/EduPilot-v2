# Prompt cho `ba`: spec sprint 7 (P4 + P5)

Làm ở worktree `/Users/kuro/Documents/TA_Agent_v2-s7`, nhánh `sprint/7-p4-p5` (tách từ `main` `628a79b`).

## Đọc
- `docs/team/CONTEXT.md`, `docs/sprints/7/plan.md`: 15 story, thứ tự, quyết định PM đã chốt, nợ đem vào.
- `docs/phases/P4.md`, `docs/phases/P5.md`.
- `docs/PRD.md`: M2, M3, M5, M6, M13-C, mục quyền, mục phi chức năng.
- `docs/FLOWS.md`: F4, F5, F7, F8 kèm nhánh lỗi; bảng nguồn việc "Hôm nay" ở F14.
- `docs/ARCHITECTURE.md`: bảng, API, gói `escalation`, `notify`, `mail`, `crm`, `activity`, `thread`, `today`.
- `docs/SYSTEM_DESIGN.md`: outbox, SSE nhiều bản gateway, heartbeat write-behind, SLO.
- `docs/DECISIONS.md`: D6 (không nhận reply qua hộp thư), D44, D47.
- `docs/design/DESIGN.md`: §13 lời văn; §14.5 `/inbox`, §14.7 `/students/[id]`, §14.8 `/attendance`, §14.9 `/me`, `/students`; quy tắc panel D59.
- `docs/UX.md` mục 1, 5, 6.
- Spec đã có, chỉ đọc: `FEAT-course-foundation` (thông báo, mail outbox, `class_sessions`, "Hôm nay"), `FEAT-private-chat-pii` (chat, Threads, confidence, tool cá nhân, preflight an toàn), `FEAT-docs-calendar` (ingest, RAG).
- Code hiện có, chỉ để biết cái đã làm: `backend-go/internal/agent` (preflight an toàn, `should_escalate`, tool `get_my_attendance` / `get_my_participation` đang trả "chưa có dữ liệu"), `internal/thread`, `internal/today`, migration `00003`, `00004`, `00007`.

## Việc
Viết hai feature, **FEAT-escalation-notify trước** (commit, push và báo PM ngay khi xong để Tech Lead thẩm định trong lúc bạn viết feature thứ hai):
- `docs/specs/FEAT-escalation-notify/{US,SRS,QUESTIONS}.md`: US-P4-01…07.
- `docs/specs/FEAT-crm-360/{US,SRS,QUESTIONS}.md`: US-P5-01…06 và US-S7-SEED.

Mỗi story có ít nhất một AC nhánh lỗi và một AC phân quyền. AC phải kiểm được bằng máy hoặc bằng tay, có số đo khi có thể.

Bắt buộc có:
1. **Ticket.** Tạo idempotent theo `message_id`; trạng thái OPEN → CLAIMED → ANSWERED → CLOSED (mở lại được); hai người cùng `Nhận` → người sau 409 và thấy ai đã nhận; tự đóng sau 7 ngày không phản hồi; `kind` QUESTION / GRADE_APPEAL / TEACHER_OUTREACH có sẵn trong CREATE (GRADE_APPEAL chưa dùng ở sprint này).
2. **Quyền đọc chat riêng.** TEACHER / TA chỉ thấy đúng phần hội thoại đã escalate, không thấy chat riêng khác của sinh viên. Ticket chỉ người trong nhóm giảng dạy của lớp đó thấy.
3. **Outbox.** Ticket + dòng outbox ghi cùng transaction; mail đi đúng một lần kể cả khi worker chết giữa chừng; mail lỗi thử 3 lần rồi hiện ở `/observability`, câu trả lời trong app không bị ảnh hưởng. Mail không chứa thông tin của sinh viên khác.
4. **Thông báo tức thời.** SSE + Redis pub/sub chạy đúng với 2 bản gateway; `Last-Event-ID` đọc bù; gộp thông báo cùng loại; chuông ≤ 5 s sau sự kiện.
5. **SLA.** Định nghĩa "24 giờ làm việc" (đề xuất mặc định 8:00–17:00 thứ Hai–thứ Sáu, giờ Việt Nam); 72 giờ → đầu "Hôm nay" + báo Admin khi lớp không ai hoạt động (định nghĩa "không ai hoạt động"); sinh viên luôn thấy trạng thái và thời gian đã chờ, bằng lời (`DESIGN.md` §13).
6. **Nút "Nhờ giảng viên".** Hiện trong `/chat` (sprint 6 đã ẩn). Nêu rõ khi nào hiện (dưới ngưỡng / sinh viên tự bấm), chống bấm nhiều lần, và ngữ cảnh gửi kèm.
7. **Kiểm duyệt Threads.** Báo cáo (lý do chọn sẵn, mỗi người một lần mỗi bài), 3 báo cáo → tự ẩn chờ duyệt, ẩn kèm lý do, chủ bài sửa (chạy lại tường lửa PII) / xoá, người đăng được báo; bài `Loại` biến mất với sinh viên và khỏi RAG; tri thức đã xác nhận / đã sửa nạp lại vector DB (nợ sprint 6).
8. **"Lưu thành tri thức".** Che thông tin cá nhân trước khi nạp; ai được bấm; sửa / gỡ được.
9. **Tin nhắn khủng hoảng.** Đối chiếu cái đã có ở sprint 6, chỉ viết AC cho phần còn thiếu; bộ 20 câu mẫu; không tự báo cho ai.
10. **Điểm danh.** Tạo lịch hàng loạt có xem trước; mặc định Có mặt; ghi cả lưới một transaction + audit; khoá lạc quan (`version`) → 409 hiển thị rõ; mất mạng xếp hàng cục bộ rồi đồng bộ (mục này cắt được nếu trễ, ghi AC tách riêng để cắt gọn); sửa sau buổi học được, ghi audit; 30 sinh viên ≤ 60 s trên điện thoại.
11. **Thời gian học.** Heartbeat chỉ ghi Redis; idle 2 phút ngừng đếm; tab ẩn không đếm; hai tab không đếm đôi; hộp thoại minh bạch lần đầu; `internal/grade` không bao giờ dùng thời gian học (luật cứng).
12. **Hồ sơ 360 + at-risk.** Mỗi lần mở hồ sơ ghi `audit_log`; sinh viên không bao giờ thấy ghi chú, nhãn rủi ro, câu nhận định rủi ro của mình (kể cả qua API, qua AI); "Nhận xét tổng hợp" AI không gửi tên / MSSV; at-risk = dưới ngưỡng 2 tuần liền, có lý do bằng lời; ngưỡng trong cài đặt lớp; `EXPLAIN` 1.000 SV không seq scan bảng lớn; danh sách phân trang con trỏ.
13. **`/me`.** Không streak, không xếp hạng, không vòng tiến độ; thời gian học là xu hướng nhỏ cuối trang.
14. **"Hôm nay".** Đăng ký nguồn việc: `TICKET` (10), bài bị báo cáo, buổi đang diễn ra chưa điểm danh, `STUDENT_ATTENTION` (95), "Lớp cần chú ý" (nợ 5.5 #2). Ghi bậc ưu tiên theo bảng SRS 4.7 của `FEAT-course-foundation`.
15. **Thay mock (D51).** `/inbox`, `/attendance`, `/students`, `/students/[id]`, `/me` nối API thật; dữ liệu `frontend/src/mock/` của các màn đó bị xoá trong sprint này.
16. **Migration.** `00012_escalation_notify`, `00013_crm`, `00014_observation`. Phase file ghi số cũ; ghi chú trong SRS. `mail_inbound` để P7 (plan).
17. **`QUESTIONS.md`.** Câu nào đụng hành vi sản phẩm / quyền / dữ liệu cá nhân thì đánh **[CHỦ DỰ ÁN]**, kèm phương án mặc định và hệ quả. Câu nên hỏi ít nhất: sinh viên có được tắt theo dõi thời gian học không; ai thấy ticket (chỉ GV + TA của lớp hay cả Admin); mail báo cho sinh viên mặc định bật hay tắt.

Không mở subagent. Chỉ sửa `docs/specs/FEAT-escalation-notify/**`, `docs/specs/FEAT-crm-360/**` và dòng `docs/sprints/7/proposals.md` của bạn.

Mỗi feature xong: commit `sprint 7: spec <FEAT>`, push, báo PM ≤ 5 dòng (số AC mỗi story, số câu [CHỦ DỰ ÁN]). Sau đó Tech Lead thẩm định (`TL-REVIEW.md`).
