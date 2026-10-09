# Prompt cho `qc` — Sprint 5.5 (UI panel nổi)

**Worktree riêng của QC** từ `origin/sprint/5.5-ui-panels`. Cổng PE của sprint 5 làm trước.

## Việc
1. Viết `docs/sprints/5.5/qc/tc-US-UI-01.md` … `tc-US-UI-07.md` + `gate-UI.md` từ spec **APPROVED** `docs/specs/FEAT-ui-panels/`.
   - Đọc cả `TL-REVIEW.md` mục "Quyết định PM".
   - Mỗi AC ≥ 1 TC.
   - Đo bằng máy: CSS tính toán, `getBoundingClientRect`, axe, tương phản, `ui-antipatterns.sh`, Lighthouse `resource-summary:script`.
2. Khi có handoff: chạy TC, viết `report-<story>.md`. Dùng **`playwright-cli`** chụp mọi màn chính ở 1440 / 1024 / 375 px để so trước / sau (ảnh vào `docs/sprints/5.5/qc/shots/`).
3. US-UI-07 xong: chạy `gate-UI.md`.

## Luật
Như `docs/sprints/5/prompts/qc-run.md`: tự đo, không tin số dev; một stack; Playwright ≤ 2 worker; dọn sau mỗi lượt.
