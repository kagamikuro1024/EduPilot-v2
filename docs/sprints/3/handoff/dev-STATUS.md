# DEV — trạng thái khi dừng (chủ dự án chuyển chỗ)
Nhánh `sprint/3-pu-p1`, đã push tới `ac85ab0`. Không có việc dở dang trong cây làm việc (chỉ còn `docs/sprints/3/qc/**` là của QC).

## Đã xong (có handoff)
US-P1-01…04 (+ bản sửa lỗi QC: BUG-P101-1, P102-1/3, P103-1/2, P104-1, góp ý #6), US-PU-01 (+ BUG-PU01-1), US-PU-02, US-PU-03.

## Chưa làm
- **US-PU-04** (shell, hai nguồn phiên `jwt`/`demo`, cổng dán token `NEXT_PUBLIC_DEV_AUTH`, điều hướng theo vai: GV 15 mục), **US-PU-05** (ảnh mốc 14 cảnh, axe, LHCI, 19 phép ui-antipatterns đã có), **US-P1-05** (`/settings/llm` thật), cổng PU + P1.
- Việc PM đã xếp hàng: không bắt đầu.

## Nợ / lưu ý cho người nhận
- `audit.mjs` đầy đủ chưa chạy ở máy dev (chụp màn hình quá hạn) — QC chạy theo `docs/sprints/3/qc/audit-baseline.md`.
- Ca `@real` của US-PU-03 (AC6, 8, 11, 13, 16, 17) chưa chạy; cần stack Go + bản dựng `NEXT_PUBLIC_API_URL=https://localhost`.
- AC23 của PU-03 (`grep DEV_AUTH` trong bản production) thuộc cổng dán token của PU-04.
- Proposals #5–#21 ở `docs/sprints/3/proposals.md`: #13–#21 chờ PM.
- Playwright: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test` (cổng 3310 + gateway giả 3311). `next start` có thể để lại tiến trình `next-server`; kiểm `lsof -i :3310 -i :3311`.
- Go: `DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true`; dọn container test bằng `make -C backend-go test-clean`.
