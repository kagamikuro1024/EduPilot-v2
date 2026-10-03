# Prompt cho `dev` — Thi công sprint 4 (P2 Lớp học)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s4`, nhánh `sprint/4-p2`** (xếp chồng trên sprint 3). Bắt đầu khi sprint 3 đã xong cổng PU + P1; trước khi bắt đầu: `git -C /Users/kuro/Documents/TA_Agent_v2-s4 merge --no-ff sprint/3-pu-p1` để có mã sprint 3 mới nhất (xung đột tài liệu → báo PM).

## Đọc
`docs/team/CONTEXT.md`, `docs/team/DEV.md`, `AGENTS.md` ("Cấm tuyệt đối": MSSV tự khai, token băm, không localStorage), `docs/sprints/4/plan.md`, spec **APPROVED** `docs/specs/FEAT-account-security/` (US-P2-01…06) và `docs/specs/FEAT-course-foundation/` (US-P2-07…12) kèm `QUESTIONS.md` (mọi câu theo mặc định của BA), `docs/phases/P2.md`, `docs/FLOWS.md` F1, F2, F14.

## Thứ tự
US-P2-01 (nộp `00003` + `00004` cùng commit — Q1) → 02 → 03 → 04 → 05 → 06 → 07 → 08 → 09 → 10 → 11 → 12 (seed) → cổng P2 (`docs/phases/P2.md`).

## Mỗi story
Code + test (Go table-driven, testcontainers, Mailpit REST cho mail; Playwright cho màn) → `make -C backend-go lint test sqlc-check`, `pnpm -C frontend lint build`, `bash scripts/ui-antipatterns.sh` xanh → commit `US-P2-NN: …` đúng đường dẫn, push → handoff `docs/sprints/4/handoff/dev-US-P2-NN.md` (AC, lệnh tự kiểm + kết quả thật, nợ).

## Luật
- Thư viện mới chỉ theo bảng `ARCHITECTURE.md` (go-mail, excelize). Ngoài bảng → góp ý `docs/sprints/4/proposals.md`.
- Không mở subagent. Spec sai → góp ý, không tự đổi, không hỏi QC/BA trực tiếp.
- Dọn docker khi xong. Báo PM ngắn sau mỗi story rồi làm tiếp, không chờ.
