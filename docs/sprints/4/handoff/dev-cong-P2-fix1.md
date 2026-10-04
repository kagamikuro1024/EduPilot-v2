# DEV handoff — Cổng P2, vòng sửa 1 (report-GATE-P2.md)

| Mục | Kết quả |
| --- | --- |
| B1 US-P2-12 | đã sửa ở `616cd98`; chờ QC chấm lại TC-08 (trên DB trống `unread_count` GV = 1 ở hai lần seed đầu) |
| G1 Lighthouse TBT | **sửa gốc**, không đổi ngưỡng — xem dưới |
| G3 stack riêng cho QC | `EP_PORT_OFFSET` — xem dưới; hai stack dựng song song cùng lúc, cả hai seed xong, không đụng nhau |
| L6 | sửa (≈ 4 dòng): mặc định vào lớp ACTIVE đầu tiên, không phải lớp đã lưu trữ; test mới trong `class-join.spec.ts` |

## G1 — TBT trên CI
- Log CI (`37163914028`): TBT 243–569 ms ở cả 6 route người dùng, **benchmarkIndex 2100–2400**; lần xanh duy nhất (`37163000672`) benchmarkIndex **3500–4200** và TBT vẫn 71–217 ms (xanh sát ngưỡng). Tức TBT thật cỡ 150 ms trên máy trung bình và CI có runner chậm hơn ≈ 40 % ⇒ đỏ. Không phải do một route.
- Hồ sơ (Playwright + CDP, CPU 4× / 6,7×, `/inbox`, phiên giả như Lighthouse): hai tác vụ dài — **~107 ms lúc nạp và chạy tập lệnh** (chunk `react-dom`, runtime turbopack) và **~170 ms lúc đăng nhập xong**: `POST /auth/refresh` xong ⇒ `useSyncExternalStore` (luôn đồng bộ) ⇒ AuthGate dựng **cả khung + trang trong một tác vụ**. Tác vụ thứ hai là phần P2 mới thêm (AuthGate/refresh).
- Sửa: `AuthProvider` đọc `useDeferredValue(useSyncExternalStore(authStore…))` ⇒ React dựng cây sau khi có phiên **theo lát thời gian** (ngắt được) thay vì một khối. Kết quả hồ sơ: tác vụ ~170 ms **biến mất** (chỉ còn tác vụ nạp 107 ms). Token vẫn đọc đồng bộ ở `tokenStore`; đăng xuất / thu hồi chỉ trễ một lát dựng.
- Kiểm: `lhci autorun` cục bộ với **CPU 6,7×** (mô phỏng runner chậm: 4 × 3800/2250) — TBT 6 route người dùng **41–151 ms** (trung vị mỗi route 50–97), `/dev/ui` 198–219 (route `warn`). Playwright không-visual: 322 passed (sau đổi); `class-join` + `today` 85 passed (sau L6).
- Kèm: `upload-artifact` không tải thư mục ẩn nên báo cáo Lighthouse (`frontend/.lighthouseci`) chưa bao giờ lên artifact `frontend-reports` — thêm `include-hidden-files: true`.
- Chưa chắc: số CI thật chỉ biết sau lần chạy này; nếu vẫn sát ngưỡng, bước kế tiếp là tách mock khỏi bundle layout (#14) — cần Tech Lead.

## G3 — dựng stack riêng
```bash
# stack của QC cạnh stack dev (project edupilot100, mọi cổng host + 100)
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock   # Colima
EP_PORT_OFFSET=100 SEED_ON_EMPTY_DB=true pnpm dev     # https://localhost:543 · Mailpit :8125 · Postgres :5533 · Redis :6480
EP_PORT_OFFSET=100 pnpm dev:status | dev:logs
EP_PORT_OFFSET=100 pnpm dev:down -v                    # chỉ gỡ project edupilot100
```
- `scripts/dev.mjs` nhận `EP_PORT_OFFSET` (hoặc `COMPOSE_PROJECT_NAME` + `EP_PORT_*` riêng lẻ); `pnpm dev:down|dev:status|dev:logs` đi qua cùng script nên dùng đúng project; seed của `dev.mjs` nhận `API_URL` / `MAILPIT_URL` / `COMPOSE_PROJECT_NAME` theo stack. `docker-compose.local.yml`: cổng host đọc `${EP_PORT_*:-mặc định}`.
- **Kiểm thật:** `pnpm dev` (mặc định) và `EP_PORT_OFFSET=100 pnpm dev` chạy **cùng lúc** (hai `docker compose up --wait` song song, cùng build): cả hai xong, seed 36 s / 37 s, mỗi DB 60 `users`; đăng nhập trình duyệt thật vào `https://localhost:543` (GV thấy bộ chọn lớp "761987"); `EP_PORT_OFFSET=100 pnpm dev:down -v` chỉ gỡ stack B (stack A còn 9 container), rồi gỡ A. Đã dọn sạch (`docker volume prune -f`).
- `docker compose up --wait` **không gãy** khi chạy song song với project khác. Lỗi "No such container" lúc QC thử là do `down -v --remove-orphans` trên project `edupilot` của dev (xem `dev-US-P2-12.md`).
- Lưu ý cho QC: tên volume / network theo project (`edupilot100_postgres_data`, `edupilot100_default`); `gate-pg.sh` hard-code `edupilot_…` và `pnpm dev:down` ⇒ cần `EP_PORT_OFFSET=100` khi chạy và thay tên (script của QC, dev không sửa). Truy cập `http://localhost:180` (HTTP) chuyển hướng về cổng 443 chuẩn của Caddy — dùng HTTPS với cổng đầy đủ.

## L6
Bộ chọn lớp **đã** nhớ lựa chọn qua tải lại (`localStorage ep:ui:course`; ca `?course= … được nhớ ở localStorage` + ca mới "chọn tay … tải lại vẫn giữ" xanh) — không tái hiện được B27; có thể là ngữ cảnh trình duyệt mới của QC (localStorage trống). Phần B26 (mặc định vào lớp đã lưu trữ) đã sửa.
