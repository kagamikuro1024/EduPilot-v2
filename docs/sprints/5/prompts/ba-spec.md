# Prompt cho `ba` — Spec sprint 5 (PE Thi hằng tuần)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s5`, nhánh `sprint/5-pe`.** Git: `git -C /Users/kuro/Documents/TA_Agent_v2-s5`.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/BA.md`, `docs/team/TEMPLATES.md`, **`docs/sprints/5/plan.md`**, **`docs/phases/PE.md`**, `docs/DECISIONS.md` D54–D57 (+ D45, D46, D47), `AGENTS.md` (luật 5 tính điểm, 10–14; "Cấm tuyệt đối": đáp án khi bài còn mở), `docs/ARCHITECTURE.md` §4, §5, §7, `docs/design/DESIGN.md` (§14 route `/questions`, `/practice` làm tham chiếu màn làm bài; §21), `docs/UX.md` (tự lưu, mất mạng, 375 px), `docs/phases/P9.md` (phần đã dời sang PE). Spec nền: `FEAT-pg-foundation`, `FEAT-ui-foundation`, `FEAT-llm-gateway`, `FEAT-course-foundation` (quyền theo lớp, "Hôm nay"). Research sandbox: `docs/research/2026-10-03-code-judge.md` (nếu chưa có: viết phần sandbox theo hợp đồng trừu tượng, ghi câu hỏi; PM sẽ báo khi research xong).

## Việc
1. Thêm **F19** vào `docs/FLOWS.md` (giảng viên soạn → lên lịch → SV làm trắc nghiệm / code → chấm → đóng → tự công bố → xem kết quả → phúc khảo) kèm nhánh lỗi: mất mạng giữa bài, hết giờ khi đang gõ, sandbox chết / quá tải, test sai do giảng viên → chấm lại, SV làm ở hai tab, nộp trùng (Idempotency-Key), lỗi biên dịch. Thêm **M15** vào `docs/PRD.md` (mục tiêu, AC đo được).
2. `docs/specs/FEAT-weekly-exam/{US,SRS,QUESTIONS}.md` — US-PE-01…09 đúng bảng plan. Bắt buộc:
   - DDL đủ cột / kiểu / CHECK / index (`course_id` đầu), trạng thái + chuyển trạng thái của bài thi và lượt làm, công thức điểm (decimal, trọng số test, làm tròn — giảng viên chọn?).
   - API theo `ARCHITECTURE.md` §5 (đường dẫn, quyền, mã lỗi, Idempotency-Key cho bắt đầu / nộp, 202 + job + SSE cho chấm), khoá Redis (`ep:exam_lock`, rate limit `Chạy thử`).
   - Hợp đồng sandbox (đầu vào / đầu ra, giới hạn mặc định, phân loại kết quả, checker), test tấn công.
   - Ma trận quyền: GV / TA / SV / Admin; **AC chống rò đáp án** cho mọi endpoint SV (trước và sau công bố).
   - Màn: `/questions`, `/exams`, `/exams/[id]/take` (trắc nghiệm 375 px; code ≥ 1024 px), `/exams/[id]/results`; trạng thái tải / rỗng / lỗi; chuỗi tiếng Việt.
3. `QUESTIONS.md`: mỗi câu có mặc định an toàn; đánh **[CHỦ DỰ ÁN]** cho câu đụng điểm số, liêm chính, quyền (vd: nộp code nhiều lần hay một lần — plan mặc định nhiều lần, lần cuối tính; giới hạn thời gian / bộ nhớ mặc định; có cho xem test ẩn sau công bố không).

Không mở subagent. Chỉ sửa `docs/specs/FEAT-weekly-exam/**`, `docs/FLOWS.md` (F19), `docs/PRD.md` (M15). Commit `sprint 5: spec FEAT-weekly-exam + F19 + M15`, push. Trả lời số AC từng story + câu hỏi mở (số [CHỦ DỰ ÁN]), dừng. Làm thật kỹ.
