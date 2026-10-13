# Prompt cho `qc`: test case P4 (hộp đen)

Worktree QC `/Users/kuro/Documents/TA_Agent_qc5`: `git fetch && git switch --detach origin/sprint/7-p4-p5`.

## Đọc
- `docs/team/QC.md`, `docs/team/CONTEXT.md`, `docs/sprints/7/plan.md`.
- Spec **APPROVED** `docs/specs/FEAT-escalation-notify/` (US, SRS v1.1, QUESTIONS cột Trả lời, `TL-REVIEW.md` mục "Quyết định PM").
- `docs/sprints/7/proposals.md` (#1, #2, #3, #8, #9).
- `docs/phases/P4.md` (cổng, "Bạn tự kiểm"), `docs/FLOWS.md` F4, F5 (nhánh lỗi).
- **Không đọc code của dev.**

## Việc
Viết `docs/sprints/7/qc/tc-US-P4-01…07.md` và `gate-P4.md` (cột `TC-id → AC → tiền điều kiện → bước/lệnh → kết quả mong đợi`). Mỗi AC ≥ 1 TC; có TC nhánh lỗi và phân quyền cho mỗi story.

Phải có:
- **Quyền đọc:** TEACHER / TA lớp khác, ADMIN, STUDENT khác không đọc được ticket; giảng viên chỉ thấy đúng cặp hỏi–đáp đã nhờ, không thấy tin khác của phiên.
- **Idempotent:** bấm "Nhờ giảng viên" 2 lần / gửi lại cùng `message_id` → 1 ticket; hai người `Nhận` cùng lúc → người sau 409 thấy ai đã nhận; `answer` lặp `Idempotency-Key`.
- **Outbox + mail:** giết worker giữa chừng → Mailpit có đúng 1 thư mang `Message-ID` của dòng; mail không chứa dữ liệu sinh viên khác; mail lỗi 3 lần → mục "Thư chưa gửi được" ở `/observability` (ADMIN).
- **Chuông qua 2 bản gateway** (`--scale gateway=2`), ≤ 5 s; `Last-Event-ID` / `resync`.
- **SLA** bằng đồng hồ giả / lệnh seed: 24 h (cả ngày T2–T6) nhắc, 72 h lên đầu "Hôm nay" + Admin; hỏi tiếp đặt lại đồng hồ (`waiting_since`); cuối tuần không tính.
- **Threads kiểm duyệt:** 3 báo cáo → tự ẩn; mỗi người 1 báo cáo / bài; sửa chạy lại tường lửa PII; bài ẩn không lọt ở tìm kiếm / "tương tự" / Hôm nay; bài `Loại` ra khỏi RAG.
- **Khủng hoảng:** 20 câu dương + 17 câu âm (gồm "từ từ", "từ hai", "kết thúc tất cả các tiến trình con"…); không báo ai; giờ thi vẫn có đường hỗ trợ.
- **Giờ thi:** "Nhờ giảng viên" bị khoá (`EXAM_LOCKED`).
- **PII trong câu trả lời của giảng viên:** tên / MSSV / email sinh viên khác → `422 PII_IN_ANSWER`; email của chính giảng viên thì cho.
- **375 px** cho `/inbox` (danh sách → chi tiết) và `/chat`.

Nghiệm thu tay bằng `playwright-cli` (global). Xong: commit `sprint 7: QC test case P4`, push, báo PM ≤ 5 dòng (số TC từng story). Câu hỏi về AC ghi `Q-QC-…` trong file TC.
