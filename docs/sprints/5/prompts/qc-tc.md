# Prompt cho `qc` — Viết test case sprint 5 (PE Thi hằng tuần)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s5`, nhánh `sprint/5-pe`.** Bắt đầu sau khi xong cổng PU + P1 sprint 3; nếu dev đã giao story sprint 4 thì chạy TC sprint 4 trước, viết TC sprint 5 xen giữa.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/QC.md`, `docs/team/TEMPLATES.md`, spec **APPROVED** `docs/specs/FEAT-weekly-exam/` (US, SRS, QUESTIONS), `docs/FLOWS.md` F19, `docs/phases/PE.md` (cổng), `docs/sprints/5/proposals.md`.

## Việc
1. `docs/sprints/5/qc/tc-US-PE-01.md` … `tc-US-PE-09.md`: mỗi AC ≥ 1 TC; có TC nhánh lỗi, phân quyền (GV / TA / SV / Admin / người ngoài lớp) và 375 px cho màn trắc nghiệm.
2. Bắt buộc có nhóm TC:
   - **chống rò đáp án**: mọi endpoint SV, trước và sau công bố, `reveal_answers` bật / tắt, test ẩn chỉ hiện số đạt / tổng;
   - **sandbox tấn công** (theo bảng A1–A15 của SRS, QC tự viết mã tấn công, không dùng mã của dev);
   - **đồng hồ**: hết giờ khi đang gõ, grace 10 s, bắt đầu muộn, mất mạng, hai tab;
   - **điểm**: `PARTIAL` (mặc định) và `ALL_OR_NOTHING`, câu `void`, làm tròn một lần ở cuối, chấm lại làm giảm điểm có lý do;
   - **tự công bố đúng một lần** khi bài đóng.
3. Góp ý #1: sửa `docs/sprints/3/qc/tc-US-PU-04.md` AC3 thành 8 / 13 / 16 **trên nhánh `sprint/5-pe`**, commit ghi `#1`.
4. `docs/sprints/5/qc/gate-PE.md`: kịch bản cổng PE theo `PE.md`.

Không mở subagent. Câu hỏi về AC → ghi `docs/specs/FEAT-weekly-exam/QUESTIONS.md` mục Q-QC, báo PM. Commit `sprint 5: QC test case PE`, push, báo PM ≤ 5 dòng.
