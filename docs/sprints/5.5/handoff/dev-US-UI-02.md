# DEV handoff — US-UI-02 (token + `Panel` / `PanelSection`)
Nhánh `sprint/5.5-ui-panels`; D59 phương án (a). Spec `FEAT-ui-panels` v1.1. Chưa đổi màn nào ngoài những chỗ cần để bỏ `--ep-paper` (UI-03…06 làm khung và màn).

## Làm gì
- `frontend/src/shared/styles/tokens.css`: thêm `--ep-canvas` (`oklch(95% 0.008 60)`), `--ep-surface-strong`, `--ep-panel-border` (= `--ep-rule`), `--ep-radius-panel` (14 px), `--ep-elevation-1` (bóng hai lớp cột (a)); **xoá `--ep-paper`**; `--ep-focus` vòng trong = `--ep-surface`; nền `html` = `--ep-canvas`. Chuyển mọi chỗ dùng: `base.css` (viền thanh cuộn → canvas), `AppShell.module.css` (thanh trên → `--ep-surface`, chấm chưa đọc → `--ep-surface`), `AuthShell.module.css` (canvas), `DataTable.module.css` (đầu bảng `--ep-surface-subtle`, cột dính `--ep-surface`), `features/{chat (composer dính → surface), calendar (ngoài tháng → surface-subtle), grading + gradebook (thanh dính cuối trang → canvas)}`, `docs/design/DESIGN_TOKENS.css` (đã ở UI-01 AC7).
- `frontend/src/shared/ui/Panel.tsx` + `Panel.module.css` (Server Component thuần CSS, không context, không `className`), xuất ở `shared/ui/index.ts`. `PanelSection` (`title` → `h3`, `action`, `tone="strong"`). Bán kính / bóng chỉ ở `Panel.module.css`.
- Primitive trong panel: `DataTable` / `ActionList` bỏ `border-top` của gốc (hàng ngăn bằng đường kẻ; đầu bảng nền `--ep-surface-subtle`); `Composer` trong `[data-ep-panel]` đường kẻ trên 1 px, không `--ep-shadow-composer`; `Tabs` / `DefinitionList` đã đạt.
- `/dev/ui`: mục **Panel** (`id="panel"`, `PanelMatrix.tsx`, 14 ô: {md, lg, none} × {nội dung, hai `PanelSection`}, ô nhấn 1 và 3, `DataTable` / `ActionList` / `Tabs` + `DefinitionList` / `Composer`, `PageState` tải / rỗng / lỗi) và ca gieo vi phạm `?fixture=strong4` / `?fixture=wall`. Không nằm trong `REGISTRY` (ma trận 25 khối × trạng thái của `FEAT-ui-foundation` giữ nguyên).
- Test: `e2e/panels.spec.ts` (tokens, dom contract, nest, strong cap / kpi wall, primitives in panel, token only — 11 ca ở dự án desktop), `e2e/contrast.spec.ts`, `e2e/support/colors.ts`.
- `docs/design/DESIGN.md`: §10.14 `Panel` / `PanelSection`, §19 liệt kê `ui/Panel.tsx`.

## AC (kết quả thật)
| AC | Kết quả |
| --- | --- |
| 1 | `grep -rn 'ep-paper' frontend/src frontend/e2e docs/design | wc -l` = **0**; 5 token ở `tokens.css` = **5**; `panels.spec.ts -g tokens` pass (canvas / surface-strong / panel-border / elevation / radius đúng cột (a), nền `html` = canvas) |
| 2 | chỉ `shared/ui/Panel.module.css` dùng `ep-radius-panel` / `ep-elevation-1` ngoài `shared/styles/`; `filter` / `backdrop-filter` = **0**; ui-antipatterns phép 20, 21 ✓ |
| 3 | `-g 'dom contract'` pass: nền, viền 1 px, bán kính 14 px, bóng, padding 24 / 32 / 0, đường kẻ giữa hai `PanelSection`, `h3`, ô nhấn (nền strong, 8 px, không viền / bóng); `grep -c '"use client"' Panel.tsx` = 0 |
| 4 | phép 22 ✓ trên mã thật; `--selftest` `22 / 22` (hạt giống `nest.tsx` nhiều dòng); `-g nest` pass: `/dev/ui` kể cả mở mọi lớp nổi, và NEST / TITLE = 0 ở mọi route × vai hiện có (chưa có `Panel` nào ở màn thật — UI-04…06 tăng dần); `grep -c 'use client\|createContext' Panel.tsx` = 0 |
| 5 | `-g 'strong cap\|kpi wall'` pass: ca gieo 4 ô nhấn và 3 panel cùng hàng **bị bắt**; `/dev/ui` và mọi route thật 0 vi phạm |
| 6 | `contrast.spec.ts` pass (bảng dưới); không phải chỉnh giá trị khởi điểm. amber: 3,59 trên `--ep-surface` (≥ 3). `color:…var(--ep-amber)` còn ở: `OfflineBanner` (biểu tượng `svg`), `VerificationState` (`.dot`, `.icon` — chấm / biểu tượng); các chỗ khác chỉ là **viền** (`Calendar` `.event.deadline` / `.chip.deadline`, `Feedback` `.n_warning`) — không có chữ màu amber |
| 7 | `-g 'primitives in panel'` pass (computed `border-top-width` / `box-shadow` / `border-radius` của gốc `DataTable`, `ActionList`, `Tabs`, `DefinitionList` = 0 / none / 0; đầu bảng = `--ep-surface-subtle`; `Composer` 1 px rule + `none`); `/dev/ui` 1440 / 390: AUDIT `ox = 0`, không ô tràn. `InlineNotice` giữ viền nhạt (không đổi) |
| 8 | `-g 'token only'` pass (ghi đè năm token → nền html, panel, ô nhấn, viền, bóng đổi theo); `grep -rnE 'prefers-color-scheme\|data-theme' frontend/src | wc -l` = 0 |
| 9 | ui-antipatterns 22 ✓ rc 0, `--selftest` 22 / 22, `lint-selftest` 7 / 7 + 22 / 22 (số `19` → `22` ở chữ báo của script, SRS 4.3); `ui-allow:` ở `frontend/src`: **9 → 9**; `package.json` / `pnpm-lock.yaml` không đổi |
| 10 | Đo bằng CDP (tổng `encodedDataLength` của request `Script`, trung vị 3 lượt, bản dựng cổng, khung `AuthGate` + `asDemo` Giảng viên) **cạnh nhau trước / sau trên cùng máy** — không dùng `lhci` (Lighthouse không chạy ở worktree này; chưa chạy lại bảng PU-06). Δ ≤ **0,24 KB** mỗi route, `/dev/ui` +1,27 KB. `PreShell.tsx` không có `Panel` (0). Ghi chú: số tuyệt đối khác bảng PU-06 vì cách đo khác (bản dựng cổng có `/dev/*`, không gzip lại), chỉ Δ có ý nghĩa |
| 11 | `DESIGN.md` §10.14 + §19 ✓; ảnh mốc 14 / 14 sinh lại (bảng dưới), `visual.spec.ts` **14 passed ba lần liên tiếp** trong `mcr.microsoft.com/playwright:v1.63.0-noble` (lần 1 `--update-snapshots=all`, hai lần kiểm không cập nhật). Đề xuất thêm khối `Panel` vào `FEAT-ui-foundation` 7.2: **PM ghi proposal** (BA không tự sửa) — chưa ghi vì `proposals.md` của 5.5 là việc PM |

### Tương phản (`contrast.spec.ts`; màu đã giải)
| Chữ \ Nền | canvas | panel | ô nhấn | surface-subtle | red-soft |
| --- | --- | --- | --- | --- | --- |
| ink | 16,01 | 18,58 | 16,50 | 16,96 | 16,30 |
| ink-2 | 8,02 | 9,30 | 8,26 | 8,49 | 8,16 |
| ink-3 | **4,57** | 5,30 | 4,70 | 4,84 | 4,65 |
| red | 4,90 | 5,69 | 5,05 | 5,19 | 4,99 |
| green | 5,08 | 5,89 | 5,23 | 5,38 | 5,17 |
| blue | 4,91 | 5,70 | 5,06 | 5,20 | 5,00 |

### JS truyền (KB, trung vị 3 lượt)
| Route | trước | sau | Δ |
| --- | --- | --- | --- |
| `/`, `/chat`, `/threads`, `/inbox`, `/settings/llm` | 339,8 | 340,1 | 0,24 |
| `/gradebook` | 345,3 | 345,5 | 0,24 |
| `/dev/ui` | 197,8 | 199,1 | 1,27 |

### Ảnh mốc sinh lại (14)
| Ảnh | Lý do | Trước / sau |
| --- | --- | --- |
| `frontend/e2e/visual.spec.ts-snapshots/chat-1440.png` | nền trang canvas; thanh trên trắng; ô nhập (composer) nền `--ep-surface` | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/chat-390.png` | nền trang canvas; thanh trên trắng; ô nhập (composer) nền `--ep-surface` | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/dev-ui-1440.png` | nền trang canvas; thêm mục `Panel` (14 ô) cuối ma trận; đầu bảng `--ep-surface-subtle` | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/dev-ui-390.png` | nền trang canvas; thêm mục `Panel` (14 ô) cuối ma trận; đầu bảng `--ep-surface-subtle` | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/gradebook-1440.png` | nền trang canvas; thanh trên trắng; đầu bảng `--ep-surface-subtle`, cột dính nền `--ep-surface` | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/gradebook-390.png` | nền trang canvas; thanh trên trắng; đầu bảng `--ep-surface-subtle`, cột dính nền `--ep-surface` | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/home-1440.png` | nền trang `--ep-canvas` (xám ấm) thay nền gần trắng; thanh trên `--ep-surface` trắng | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/home-390.png` | nền trang `--ep-canvas` (xám ấm) thay nền gần trắng; thanh trên `--ep-surface` trắng | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/inbox-1440.png` | nền trang canvas; thanh trên trắng; đầu bảng `--ep-surface-subtle`, bảng không còn đường kẻ trên | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/inbox-390.png` | nền trang canvas; thanh trên trắng; đầu bảng `--ep-surface-subtle`, bảng không còn đường kẻ trên | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/settings-llm-1440.png` | nền trang canvas; thanh trên trắng | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/settings-llm-390.png` | nền trang canvas; thanh trên trắng | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/threads-1440.png` | nền trang canvas; thanh trên trắng | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/threads-390.png` | nền trang canvas; thanh trên trắng | trước: `ace622a`/`1abe61e` (git); sau: ảnh trong commit này |

Ngoài ra `--update-snapshots` mặc định chỉ ghi ảnh lệch quá ngưỡng (`threshold 0.2`), nên lần đầu chỉ 2 / 14 ảnh đổi dù nền mọi trang đổi; đã dùng `--update-snapshots=all` để ảnh phản ánh đúng giao diện hiện tại.

## Gates
`pnpm lint`, `tsc --noEmit` sạch; Playwright không `@real`/`visual`: **455 passed / 115 skipped / 0 failed** (5,1 phút, 2 worker); `panels.spec.ts` + `contrast.spec.ts` 12 passed; `ui-antipatterns` 22 ✓. **Chưa chạy**: Lighthouse (`lhci`), GitHub CI của nhánh 5.5 sau commit này (xem handoff kế tiếp).
