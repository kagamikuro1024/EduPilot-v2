# Báo cáo QC — US-UI-03 (khung ứng dụng trên canvas)
**Kết luận: PASS có điều kiện** — AC1…AC8 đạt theo số đo của QC; điều kiện: AC6 (CLS bằng `lhci`) không đo được (Chrome cho `lhci` không chạy ở máy QC) — QC thay bằng `shell.spec.ts -g 'preshell parity'` của dev + đo nền ba vùng; ảnh mốc 14 / 14 hai lần (TC-08). Bản chấm `8ba5ec9`, so với `1abe61e`. QC tự đo bằng `scripts/qc-measure.spec.ts` (nền body / sidebar / thanh trên / main, viền, `backdrop-filter`, nav, thanh dưới, `AuthPanel`); chạy thêm `shell.spec.ts`, `panels.spec.ts`, `contrast.spec.ts` của dev: **40 pass**, 0 fail (dự án mobile bỏ qua theo thiết kế). Ảnh trước / sau: `docs/sprints/5.5/qc/shots/ui03-before|ui03-after/` (`/` bốn vai + sáu màn đăng nhập × 1440 / 1024 / 375); số đo `measure/ui03-*.json`.

## Lỗi / lệch
- **L1 (TOUCH).** Ở 375 px, phần tử tương tác < 44 px duy nhất là `a[Bỏ qua điều hướng] 159 × 40` (liên kết ẩn ngoài màn hình tới khi focus). Không tính FAIL.
- **L2 (đo màn thật).** `main` của màn ứng dụng trong suốt (đúng spec) — nền trang là `body` canvas.
- Không có bug.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| TC-UI03-01 (AC1) | PASS | 4 vai × `/`: `body` = canvas `lab(94.19 …)`; **sidebar** từ `lab(96.45 …)` (surface-subtle) → **`lab(100 0 0)`** (surface trắng), `border-right` **1 px** `lab(87.13 …)` (`--ep-rule`); **thanh trên** `lab(100 0 0)` + `border-bottom` 1 px; `main` `rgba(0,0,0,0)`; `backdrop-filter: none` ở cả hai; dev `frame layers` pass |
| TC-UI03-02 (AC2) | PASS | mục nav đang chọn (`aria-current`) nền `lab(96.45 2.44 1.31)` = `--ep-surface-subtle` ở 1440 / 1024; vạch đỏ 2 px, focus vòng: dev `nav states` pass; tương phản chữ nav ≥ 4,5: `contrast.spec.ts -g sidebar` pass |
| TC-UI03-03 (AC3) | PASS | `ox = 0` ở 1440 / 1024 / 375 mọi vai và màn đăng nhập; `git diff 1abe61e..8ba5ec9 -- frontend/src/shared/ui/Layout.module.css` = không đổi (padding `.page` giữ) |
| TC-UI03-04 (AC4) | PASS | 375 px: thanh dưới `lab(100 0 0)`, `border-top` **1 px**, `backdrop-filter: none`, cao 57 px; dev `bottom nav + more sheet` pass (mục ≥ 48 px, bảng "Thêm" không có `Panel`) |
| TC-UI03-05 (AC5) | PASS | 6 màn (`/login`, `/register`, `/forgot-password`, `/reset-password`, `/verify-email`, `/invite/…`): **một** `[data-ep-panel]`, `h1` đứng **trước** panel (ngoài panel, TITLE = 0), NEST 0, panel **440 px** ở 1440 / 1024 và **351 px** (lề **12 px**) ở 375; nền `main` = canvas; ≤ 1 nút primary; `ox = 0`; dev `auth shell` + `account.spec.ts`, `class-join.spec.ts` pass |
| TC-UI03-06 (AC6) | PASS (một phần) | `preshell parity` (dev) pass: nền body / sidebar / thanh trên của `PreShell` = `AppShell`; QC đã đo nền ba vùng ở màn thật như TC-01; `PreShell.tsx` không import `Panel`. CLS `lhci`: không đo được |
| TC-UI03-07 (AC7) | PASS | `git diff 1abe61e..8ba5ec9 -- nav.ts` không đổi; dev `nav per role\|route access` pass (số mục nav theo vai 8 / 13 / 16 / 6); không `backdrop-filter` (TC-01) |
| TC-UI03-08 (AC8) | PASS | 12 ảnh khung của dev có ở `handoff/ui-03/`; `visual.spec.ts` (image `v1.63.0-noble`) ở `8ba5ec9`: **14 passed** hai lần liên tiếp (QC tự chạy) |

## Việc sau
- Màn có panel thật (UI-04…06): đo NEST / WALL / lề 12 px ở từng route.
