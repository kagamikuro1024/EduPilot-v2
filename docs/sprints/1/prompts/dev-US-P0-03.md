# Prompt cho `dev` — US-P0-03

Thi công **US-P0-03**, spec `docs/specs/FEAT-ci/` (v2, APPROVED; đã gồm proposals #5). Nhánh `sprint/1-p0-prep`. Cùng luật môi trường và git như `docs/sprints/1/prompts/dev-US-P0-02.md`.

- Phiên bản trong CI khớp máy dev và D48: Go theo `backend-go/go.mod`, Node 24, pnpm theo lockfile (thêm trường `packageManager` ở `package.json` gốc nếu cần để CI và Dockerfile cùng một bản — gỡ luôn nợ "pnpm ghim tay" của US-P0-02 nếu làm được trong vài dòng).
- AC3/AC4: nhánh tạm `ci/red-check`; handoff ghi `<ID>`, `headSha`, nhánh cho từng run. **Không xoá** `ci/red-check` cho tới khi PM báo QC đã chấm AC3/AC4.
- Handoff `docs/sprints/1/handoff/dev-US-P0-03.md`. Trả lời ≤ 10 dòng rồi dừng.
