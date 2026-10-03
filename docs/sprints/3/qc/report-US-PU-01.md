# Báo cáo QC — US-PU-01 (token, font, chuẩn hoá nền, lint chặn giá trị viết cứng)
**Kết luận: FAIL** — 2 TC không đạt theo chữ của AC, cả hai nhẹ và không phải lỗi hiển thị: TC-PU01-11 (BUG-PU01-1: chuỗi in của `lint-selftest.sh` thiếu "ui-antipatterns") và TC-PU01-33 (phép kiểm cũ của `proto-curl.sh` chặn `@playwright/test`, **lỗi công cụ của QC**). Còn lại 33/36 TC PASS, 1 TC chờ US-PU-05 (TC-34). Không có hồi quy giao diện: `audit.mjs` 683 hàng, `sweep` 42 hàng sạch.

- Bản chấm: `216ad45` (chứa `d0d041a`) trong worktree `../TA_Agent_qcp1`; `pnpm install --frozen-lockfile` + `pnpm -C frontend build` rc=0, 0 dòng `Failed to load font`; chạy `next start -p 3400` (cổng 3300 là của Playwright `webServer` của dev — QC dùng 3400 để không giẫm nhau). Trình duyệt: Chrome for Testing 150 riêng (`--remote-debugging-port`), `caffeinate -d -i`.
- Q-QC-PU01-1: QC lấy tên luật từ SRS 4.2 (chọn (a)); khớp 7/7 — chờ BA xác nhận. Q-QC-PU01-2: **chờ BA**; QC không so điểm ảnh được (xem TC-34).

## Lỗi / phát hiện
### BUG-PU01-1 (thấp) — chuỗi tự kiểm không đúng chữ AC3
`bash scripts/lint-selftest.sh` in `7 / 7 luật ESLint bắt được` và `19 / 19 phép bắt được`; AC3 (US dòng 40) đòi `19 / 19 phép ui-antipatterns bắt được`. Tái hiện: `bash scripts/lint-selftest.sh | tail -2`. Hành vi đúng (7/7, 19/19, rc=0, cây sạch). Việc cần: dev sửa chuỗi **hoặc** BA sửa AC3 (US dòng 238 đã dùng dạng ngắn — spec tự mâu thuẫn).

### Phát hiện công cụ QC (không phải lỗi dev)
- `proto-curl.sh all` ra **496 PASS / 1 FAIL**: phép `tc_00_static` "package.json không thêm phụ thuộc ngoài lucide-react" so `git diff main -- frontend/package.json` và thấy `@playwright/test` (devDependency mà dev thêm đúng kế hoạch: `ARCHITECTURE.md` mục E2E liệt kê Playwright). Với phép đó nới thêm `@playwright/test` (bản sao tạm, không commit): **497 PASS / 0 FAIL**. QC chưa sửa công cụ khi chưa được duyệt → TC-33 giữ FAIL; PM cho phép thì QC sửa phép kiểm cũ (`docs/sprints/1.5/qc/scripts/proto-curl.sh`) và chạy lại.
- `audit.mjs` báo 7 hàng `AUDIT` FAIL (`cut=[…]`) ở `/me` (SV, ×3 bề rộng), `/analytics` (GV ×2, TA ×2). Đo tay: các phần tử bị coi là "cắt" (`T5`, `120 phút`, `T6 23/10`, `40 câu`…) đều nằm trong `.ep-sr-only` (1 × 1 px, `clip: rect(0,0,0,0)`, `overflow: hidden`) — bảng dữ liệu dành cho trình đọc màn hình của `Chart.tsx`, mà dev vừa đổi từ `<table>` sang `div role="table"` (góp ý #12). Trước đây `<table>` không nhận `overflow` nên `audit.mjs` bỏ qua; nay `div` nhận `overflow:hidden`. Không có gì bị cắt với người dùng (ảnh `shots/pu01-audit/student_me-1440-audit.png`, `teacher_analytics-1440-audit.png`). **nghi lỗi công cụ** → TC-32 PASS (đo tay); QC sẽ loại `.ep-sr-only` khỏi phép đo ở lần cập nhật công cụ kế tiếp.

## Ghi chú
- Điểm hở của `ui-antipatterns.sh` (không vi phạm AC, để dev biết): `RGB(`/`HSL(` viết hoa lọt (CSS không phân biệt hoa thường); `hwb(`, `border-top-left-radius:20px`, `font-size:calc(…)` lọt; `box-shadow:none` hoặc `var(--ep-focus)` ở **câu lệnh cuối không có `;`** bị báo nhầm (SRS 4.1 cho phép). ESLint `ep/*`: lọt khi viết vòng (`const f = fetch`, khoá động `localStorage[k]`, `localStorage['setItem']`) — đúng giới hạn phân tích cú pháp, SRS 4.2 không đòi hơn.
- `body` có `font-family` tính ra `…, -apple-system, "system-ui", "Segoe UI", sans-serif` còn `--ep-font` là `…, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`: bốn mục đầu đúng AC5; phần đuôi khác nhau (Chrome chuẩn hoá). Không ảnh hưởng.
- Cookie `ep_demo_course/role/person` đọc được bằng JS (phiên **mô phỏng**, không phải token) — không khoá nào chứa `token|jwt|access|refresh|secret|password`.

## Kết quả từng TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `comm -23 /tmp/kit /tmp/own \| wc -l` = `0` |
| 02 | PASS | `comm -13` = `4`: `--ep-mask-clear --ep-mask-solid --ep-scrim --ep-shadow-composer` |
| 03 | PASS | `DESIGN_TOKENS.css` 51 khai báo; `git log` thấy 1 commit (bản gốc không bị sửa) |
| 04 | PASS | khai báo `--ep-*` ngoài `tokens.css` = `0` |
| 05 | PASS | `pnpm -C frontend lint` rc=0, 0 cảnh báo |
| 06 | PASS | `ui-antipatterns.sh` rc=0, **19** dòng `✓` (≥ 11), 0 dòng khác |
| 07 | PASS | `ui-allow:` = `10`, trùng 10 tệp trong `audit-baseline.md` |
| 08 | PASS | `.a{color:#ff0000}`, `.b{background:rgb(1,2,3)}` → `✗ Màu viết cứng`; `.c{border-radius:14px}` → `✗ Bo góc`; gỡ → rc=0, `git status frontend` rỗng |
| 09 | PASS | `box-shadow:0 2px 4px red`, `font-size:15px`, `z-index:99` bị bắt; `border-radius:0`, `50%`, `z-index:1`, `font-size:inherit` **không** bị báo; thêm: `var(--ep-radius-md)`, `var(--ep-text-item)`, `box-shadow: var(--ep-shadow-menu);` lành |
| 10 | PASS | `className="text-[13px] rounded-2xl shadow-lg bg-gray-100"` → 4 phép ✗ (xám, bo góc, bóng, cỡ chữ); `style={{ color:'#fff', borderRadius:12 }}` → ✗ (antipatterns) và `ep/no-literal-color-in-style` (ESLint); xoá → sạch |
| 11 | **FAIL** | BUG-PU01-1 (chuỗi); còn lại: `7 / 7`, `19 / 19`, rc=0, `git status --short frontend` = 0 |
| 12 | PASS | ngắt (`SIGINT`) sau 2 s và 4 s: `frontend/src/__selftest__` và `features/_selftest` không còn, `git status frontend` rỗng, không tiến trình sót |
| 13 | PASS | QC tự gieo 33 ca bằng `eslint <tệp>` (một tệp mỗi ca), tên luật đúng SRS 4.2: **E1** `ep/no-raw-fetch` (`fetch`, `window.fetch`, `globalThis.fetch`, `XMLHttpRequest`, `axios`); **E2** `ep/no-native-dialogs` (`confirm`, `window.confirm`, `alert`, `prompt`, `globalThis.alert`); **E3** `ep/no-custom-spinner` (`Spinner`, `FullPageSpinner`, `LoadingSpinner`); **E4** `ep/no-raw-table`; **E5** `ep/no-token-in-storage` (`access_token`, `jwt`, `document.cookie='refresh=1'`, `password=`, template `` `my_Token` ``); **E6** `ep/no-tailwind` (`tailwindcss`, `@tailwindcss/vite`); **E7** `ep/no-literal-color-in-style` (`#fff`, `rgb(`, `oklch(`, `hsl(`); ca lành `ep_demo_state`, `var(--ep-red)` rc=0 |
| 14 | PASS | `fetch(` ngoài `shared/data` = 0; ca âm bị chặn |
| 15 | PASS | `confirm(`/`alert(` = 0; ca âm bị chặn |
| 16 | PASS | `<Spinner/>` và `<table>` thô bị chặn; `<table` ngoài `DataTable` = 0 |
| 17 | PASS | 3 ca âm bị chặn; `localStorage.setItem('ep_demo_state','{}')` không bị chặn |
| 18 | PASS | `tailwind` trong `package.json` = 0; `@tailwind`/`@theme` = 0; `import 'tailwindcss'` và `style={{color:'#fff'}}` bị chặn |
| 19 | PASS | Playwright `tokens.spec.ts` 16/16 (desktop + mobile); tay: `document.fonts.check` 400/500/600/700 = `true`×4; `body` bắt đầu `"Be Vietnam Pro", "Noto Sans"` |
| 20 | PASS | trình duyệt thật mở `/login`: 25 request, host duy nhất `localhost:3400`, **0** tới `fonts.googleapis.com` / `fonts.gstatic.com` |
| 21 | PASS | `tokens.spec.ts › font` (đọc `layout.tsx`: 1 dòng `weight`, 4 độ đậm, `display: "swap"`, `vietnamese`) |
| 22 | PASS | `--ep-font` = `"Be Vietnam Pro", "Noto Sans", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif` |
| 23 | PASS | `/gradebook` (GV): 310 ô, **0** thiếu `tabular-nums` |
| 24 | PASS | `tokens.spec.ts › diacritics` ok ở cả hai dự án |
| 25 | PASS | cỡ chữ tính ra ở `/`, `/chat`, `/threads` (SV), `/gradebook` (GV) chỉ gồm 12, 14, 15, 16, 20, 28 px = đúng sáu `--ep-text-*` (meta, label, body, item, section, page) |
| 26 | PASS | `tokens.spec.ts › focus` ok |
| 27 | PASS | `color-scheme: light`; `::selection { background: var(--ep-red-soft) }`; `scrollbar-width: thin`; `prefers-color-scheme: dark` trong CSS đã build = 0 |
| 28 | PASS | `emulateMedia(reducedMotion: reduce)`: 238 phần tử, thời lượng chuyển/animation tối đa **0,001 s**, 0 phần tử vượt |
| 29 | PASS | `/brand/favicon.svg`, `logo-edupilot.svg`, `logo-edupilot-mark.svg` đều `200`; `<link rel="icon" href="/brand/favicon.svg"/>` |
| 30 | PASS | `tokens.spec.ts › brand` ok (cha `border-radius:0px`, nền trong suốt) |
| 31 | PASS | build rc=0, 0 dòng `Failed to load font` |
| 32 | PASS (nghi lỗi công cụ) | `audit.mjs`: SV **165**, GV **170**, TA **106**, Admin **54**, spec **188** = **683** hàng (nền 680, 0 hàng nền bị mất); **7 hàng FAIL đều là `.ep-sr-only` (xem trên)**, 676 hàng còn lại PASS |
| 33 | **FAIL** | `sweep.mjs only:'student'`: 42 hàng, `FORBIDDEN` 0, 0 cuộn ngang, 0 HTTP lỗi, 1 `h1`/trang. `proto-curl.sh all` **496 PASS / 1 FAIL** (phép kiểm cũ, xem trên); nới phép kiểm: 497/0 |
| 34 | chờ US-PU-05 | điều kiện trước (ảnh mốc `visual.spec.ts-snapshots/`) chưa có; thử so điểm ảnh với `shots/before/` không đối chứng được (cửa sổ CfT kết nối CDP không áp dụng viewport / DPR của ảnh nền) nên QC không chấm TC này; thay bằng `audit.mjs` 683 hàng không đổi bố cục |
| 35 | PASS | `grep … setItem(…token\|jwt…)` = `0` |
| 36 | PASS | `/login`, `/`: `localStorage` = `["ep_demo_state"]`, `sessionStorage` rỗng, cookie JS-đọc-được chỉ `ep_demo_*` |

## Việc sau
Dev: BUG-PU01-1 (sửa chuỗi hoặc BA sửa AC3). PM: cho phép QC nới phép kiểm `tc_00_static` (cho `@playwright/test`) và loại `.ep-sr-only` khỏi `audit.mjs`; sau đó QC chấm lại TC-33 (đồng thời TC-34 khi US-PU-05 có ảnh mốc).

## Vòng sửa 1 (dev `00425f0`; công cụ QC sửa theo góp ý #13) — DỪNG GIỮA CHỪNG theo lệnh PM
- **TC-11 PASS:** `lint-selftest.sh` in `7 / 7 luật ESLint bắt được` và `19 / 19 phép ui-antipatterns bắt được`, rc=0, `git status frontend`=0, không còn `__selftest__`. BUG-PU01-1 **đóng**.
- **TC-32 PASS:** `audit.mjs` (đã bỏ `.ep-sr-only`) trên `a0ecef6`: SV 165, GV 170, TA 106, Admin 54, spec 188 = **683 hàng, FAIL 0**.
- **TC-33:** `sweep` SV 42 hàng, `FORBIDDEN` 0, 0 cuộn ngang, 1 `h1`/trang. `proto-curl.sh all` = 496 PASS / **1 FAIL**: `tc_00_static: token ở storage` do phép `grep` tìm chữ "localStorage" + "token" bắt **dòng chú thích** đầu `frontend/src/shared/data/tokenStore.ts` (US-PU-03, nói rằng token KHÔNG nằm ở storage) — dương tính giả của công cụ QC 1.5, không phải lỗi mã. Chưa sửa công cụ (đã dừng); đề nghị phiên sau cho phép phép kiểm bỏ qua dòng chú thích rồi chốt TC-33.
