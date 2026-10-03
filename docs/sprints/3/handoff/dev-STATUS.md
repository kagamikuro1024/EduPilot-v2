# DEV — trạng thái sprint 3 (`sprint/3-pu-p1`)
Xong và có handoff: US-P1-01…05, US-PU-01…05, mọi bản sửa lỗi QC đến BUG-PU04-1, ghi replay openai + gemini (AC13, AC11), cổng PU + P1 (`dev-GATE-PU-P1.md`).

Còn mở:
- **LCP > 2,5 s** ở cả 7 route Lighthouse (2,71–3,39 s sau khi bỏ phông 500; CLS / TBT / JS đạt): PM đã quyết #26 — `warn` tạm, Nợ ở `PROGRESS.md`, cổng P2 (sprint 4) đưa mock ra khỏi bundle layout và đổi lại `error`.
- `make eval` hoãn theo plan (P3 / P10). Anthropic chưa có khoá ⇒ replay BLOCKED.
- Ca `@real`, bước tay của cổng (xem `dev-GATE-PU-P1.md`), lượt chạy CI trên GitHub.
- Góp ý #25–#32: PM đã quyết (xem `proposals.md`); #25 / #26 / #28 đã thực hiện.

Chạy lại: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test` (Next :3310, gateway giả :3312); Go: `DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true go test -race -tags testroutes ./...`.
