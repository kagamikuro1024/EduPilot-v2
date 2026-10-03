# Prompt cho `dev` — Thi công sprint 5 (PE Thi hằng tuần)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s5`, nhánh `sprint/5-pe`** (xếp chồng trên sprint 4). Bắt đầu khi sprint 4 đã xong US-P2-12 + cổng P2; trước đó `git -C /Users/kuro/Documents/TA_Agent_v2-s5 merge --no-ff sprint/4-p2` (xung đột tài liệu → báo PM).

## Đọc
`docs/team/CONTEXT.md`, `docs/team/DEV.md`, `AGENTS.md` (luật 5 điểm `decimal`, 10–14; "Cấm tuyệt đối": đáp án khi bài còn mở, chạy code sinh viên ngoài sandbox), `docs/sprints/5/plan.md`, `docs/sprints/5/proposals.md`, spec **APPROVED** `docs/specs/FEAT-weekly-exam/` (US, SRS, QUESTIONS — mọi câu theo cột trả lời), `docs/phases/PE.md`, `docs/DECISIONS.md` D54–D58, `docs/FLOWS.md` F19, research `docs/research/2026-10-03-code-judge.md` (lệnh chạy go-judge, PoC trong `/tmp` của research đã có thể mất — dựng lại theo tài liệu).

## Thứ tự
US-PE-01 → 02 (sandbox: compose service `judge` ở mạng Docker riêng, không thấy DB / Redis / blob, không secret; `TestSandboxAttacks` đủ 15 ca) → 03 → 04 (kèm góp ý #1: `nav.ts` 8 / 13 / 16) → 05 → 06 → 07 → 08 (`TestNoAnswerLeak` bắt buộc) → 09 (seed + k6 `judge_burst` + cổng PE). Ghi ánh xạ số migration vào `docs/PROGRESS.md`.

## Mỗi story
Code + test (Go table-driven, testcontainers, judge thật cho `-tags integration`; Playwright cho màn, 375 px cho trắc nghiệm) → `make -C backend-go lint test sqlc-check`, `pnpm -C frontend lint build`, `bash scripts/ui-antipatterns.sh` xanh → commit `US-PE-NN: …` đúng đường dẫn, push → handoff `docs/sprints/5/handoff/dev-US-PE-NN.md` (AC, lệnh tự kiểm + kết quả thật, nợ).

## Luật
- Không thêm thư viện (editor, markdown) ngoài bảng `ARCHITECTURE.md`; cần → góp ý `docs/sprints/5/proposals.md`.
- Không tắt seccomp ở amd64; trên colima arm64 dùng `JUDGE_EXTRA_ARGS=-no-seccomp` (D58).
- Phân vân kỹ thuật (judge, hàng đợi, đồng hồ máy chủ, winnowing) → Tech Lead qua `docs/sprints/5/techlead.md`. Spec sai → góp ý, không tự đổi, không hỏi QC / BA.
- Không mở subagent. Dọn docker / server khi xong; không đụng cổng 3100. Báo PM ngắn sau mỗi story rồi làm tiếp.
