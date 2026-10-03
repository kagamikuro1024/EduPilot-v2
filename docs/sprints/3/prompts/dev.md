# Prompt cho `dev` — Thi công sprint 3 (PU + P1)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s3`, nhánh `sprint/3-pu-p1`** (xếp chồng trên sprint 2 + 1.5). Git: `git -C /Users/kuro/Documents/TA_Agent_v2-s3`.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/DEV.md`, `AGENTS.md`, `docs/sprints/3/plan.md`, spec **APPROVED** `docs/specs/FEAT-llm-gateway/` và `docs/specs/FEAT-ui-foundation/` (kèm `QUESTIONS.md` — mọi câu đã chốt theo mặc định của BA), `docs/phases/P1.md`, `docs/phases/PU.md`, research `docs/research/2026-10-02-*.md`.

## Thứ tự (khác bảng plan — PM đổi để tránh đụng 1.5 đang sửa giao diện)
1. **P1 trước** (chỉ backend): US-P1-01 → 02 → 03 → 04.
2. Rồi PU: US-PU-01 → 02 → 03 → 04 → 05 (trước khi gộp PU-01, QC phải có `audit-baseline.md` — FEAT-ui-foundation Q10; chưa có thì làm nhưng chưa push story đó, báo PM).
3. Cuối: US-P1-05 (`/settings/llm` thật) → cổng PU + P1.

Trước story PU đầu tiên: PM sẽ gộp lại `sprint/1.5-mock-ui` mới nhất vào nhánh này; hỏi PM nếu thấy `frontend/` lệch.

## Mỗi story
Code + test table-driven → `go vet`, `golangci-lint run`, `go test -race ./...` (+ `sqlc diff`; frontend: lint, build, `ui-antipatterns.sh`) xanh → commit `US-XX-0N: …` đúng đường dẫn, push → handoff `docs/sprints/3/handoff/dev-US-XX-0N.md` (AC nào, lệnh tự kiểm + kết quả thật, nợ).

## Luật
- Chỉ thư viện trong plan (openai-go, TanStack Query/Virtual, Playwright, axe, LHCI); cần khác → góp ý `docs/sprints/3/proposals.md`.
- Khoá provider thật chưa có (Q11): mọi test dùng `fake` + replay; AC ghi âm thật để BLOCKED, ghi rõ trong handoff.
- Không mở subagent. Spec sai → góp ý, không tự đổi, không hỏi QC/BA trực tiếp.
- `make -C backend-go test-clean` nếu container test cũ; dọn docker khi xong.
Trả lời ngắn sau mỗi story (story, commit, kết quả test) rồi làm tiếp story kế, không cần chờ PM. Dừng khi xong US-P1-04 để PM gộp 1.5 trước PU.
