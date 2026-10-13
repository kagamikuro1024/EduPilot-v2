# Prompt cho `qc`: test case P5 + seed (hộp đen)

Worktree QC `/Users/kuro/Documents/TA_Agent_qc5`: `git fetch && git switch --detach origin/sprint/7-p4-p5`.

## Đọc
- `docs/team/QC.md`, `docs/team/CONTEXT.md`, `docs/sprints/7/plan.md`.
- Spec **APPROVED** `docs/specs/FEAT-crm-360/` (US, SRS, QUESTIONS cột Trả lời, `TL-REVIEW.md` mục "Quyết định PM").
- `docs/sprints/7/proposals.md` (#5, #6, #7).
- `docs/phases/P5.md` (cổng, "Bạn tự kiểm"), `docs/FLOWS.md` F7, F8 (nhánh lỗi).
- **Không đọc code của dev.**

## Việc
Viết `docs/sprints/7/qc/tc-US-P5-01…06.md`, `tc-US-S7-SEED.md` và `gate-P5.md` (cột `TC-id → AC → tiền điều kiện → bước/lệnh → kết quả mong đợi`). Mỗi AC ≥ 1 TC; có TC nhánh lỗi và phân quyền cho mỗi story.

Phải có:
- **Dữ liệu nhạy cảm:** STUDENT (kể cả qua API trực tiếp, qua `/me`, qua câu hỏi AI) không bao giờ thấy ghi chú quan sát, nhãn / câu nhận định rủi ro, `note` trên dòng điểm danh của mình → 403 / không có trường. ADMIN và TA/TEACHER lớp khác không mở được hồ sơ.
- **Audit:** mỗi lần mở hồ sơ (đếm request, cả khi quay lại trang) = 1 dòng `audit_log`; đọc ghi chú cũng ghi.
- **Điểm danh:** 30 SV ≤ 60 s (bấm giờ, 375 px và bàn phím 1–4); hai người cùng lưu → 409 rõ ràng; ngắt mạng đánh vắng 3 người rồi nối lại → tự đồng bộ, không mất; đăng xuất khi còn thay đổi chờ thì được hỏi; sửa sau buổi ghi audit; `get_my_attendance` / `get_my_participation` trả đúng số của chính mình, hỏi hộ người khác bị từ chối.
- **Lịch buổi học:** sinh hàng loạt có xem trước (không cần `Idempotency-Key` khi `dry_run`), ngày loại trừ, xoá buổi chưa điểm danh, buổi đã điểm danh không xoá được, học bù.
- **Thời gian học:** idle 2 phút ngừng đếm; tab ẩn không đếm; hai tab không đếm đôi; heartbeat không ghi Postgres; lớp cho tắt / không cho tắt / SV học 2 lớp khác cấu hình; chỉ TEACHER đổi `allow_study_time_opt_out` và chỉ khi lớp chưa có sinh viên (409 `COURSE_HAS_STUDENTS`); không đo `/exams/**`; `internal/grade` không phụ thuộc `activity` / `crm` (kể cả bắc cầu).
- **At-risk + `/students`:** đúng lý do bằng lời cho sinh viên C của seed; `EXPLAIN` lớp 100 SV không `Seq Scan`, lớp 1.000 SV trang đầu p95 ≤ 300 ms; phân trang con trỏ; "Lớp cần chú ý" ≤ 5 dòng, ẩn khi rỗng.
- **Không gamify** ở `/me` (không streak, xếp hạng, vòng tiến độ); 375 px cho `/attendance`, `/me`.

Nghiệm thu tay bằng `playwright-cli` (global). Xong: commit `sprint 7: QC test case P5`, push, báo PM ≤ 5 dòng (số TC từng story). Câu hỏi về AC ghi `Q-QC-…` trong file TC.
