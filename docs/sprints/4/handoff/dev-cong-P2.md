# DEV handoff — Cổng P2 (sprint 4)
CI `sprint/4-p2` run `37163000672`: **Go ✓, Frontend ✓** (lint, antipatterns, selftest, build, Playwright gồm `visual` 14 ảnh, Lighthouse CI).

- Ảnh mốc `visual.spec.ts`: 6 ảnh (`home`, `inbox`, `gradebook` × 2 bề rộng) sinh lại trong `mcr.microsoft.com/playwright:v1.63.0-noble` (`/` thật thay màn mô phỏng; chuông / menu hồ sơ bỏ phần mô phỏng). 14/14 pass lần chạy thứ hai không `--update-snapshots`.
- Lighthouse CI: cookie `ep_demo_*` đã bị xoá ⇒ `lighthouse-auth.cjs` đặt `lh_role`, `lighthouse-api.mjs` là gateway giả (proposal #13). Đo local: script 237–251 KB (< 256 KB), TBT 29–76 ms. **LCP vẫn 3,2–4,1 s ⇒ giữ `warn`** (proposal #14, chờ PM).
- Sửa kèm: QC L3 của US-P2-11 — `AdminProvider` đọc DB trước, tín hiệu cổng AI sau với hạn riêng 60 ms; `Service` không ghi lại cache sau khi `Get` đã lỗi (test `TestAdminTodaySurvivesHangingLLMSignals`). `TestLoginTimingEqualized` đo xen kẽ từng cặp (CI đỏ một lần vì tải máy làm lệch tỉ lệ; không đổi ngưỡng 0,65).
- Còn nợ: Bun không chạy được seed; `pnpm dev:down && SEED_ON_EMPTY_DB=true pnpm dev` chưa đo; k6 `today.js` do QC đã chạy; `@real` chưa chạy; QC `qc/scripts` dùng đăng nhập thật nên không cần sửa.
