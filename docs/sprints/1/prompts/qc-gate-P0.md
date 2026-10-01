# Prompt cho `qc` — đóng sprint 1, cổng P0

1. `ci/red-check` đã xoá. Chạy lại TC-12 (US-P0-03), cập nhật `report-US-P0-03.md` (đóng AC4).
2. BA đã sửa BUG-1 của US-P0-01 (commit `sprint 1: DEMO_SCRIPT BUG-1`): chạy lại TC liên quan AC7, cập nhật `report-US-P0-01.md`.
3. Story cuối của P0 đã xong → chạy `/gate P0` đúng theo `.claude/skills/gate/SKILL.md` và mục "Cổng nghiệm thu" của `docs/phases/P0.md`, từng lệnh đúng như viết. CI phải xanh trên HEAD `origin/sprint/1-p0-prep` tại lúc chạy (chờ run xong). Kết quả `docs/sprints/1/qc/gate-P0.md`. Lệnh nào trong cổng lệch với thực tế (tên nhánh, cổng…) → không tự sửa lệnh, ghi FAIL + đề xuất vào `proposals.md`.
4. Xong `pnpm dev:down`. Commit chỉ `docs/sprints/1/qc/**` (+ `proposals.md` nếu có). Trả lời ≤ 8 dòng, dừng.
