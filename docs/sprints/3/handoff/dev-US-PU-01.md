# DEV handoff — US-PU-01 (token, font, chuẩn hoá nền, lint chặn giá trị viết cứng)
Nhánh `sprint/3-pu-p1`. Góp ý #11, #12 ở `docs/sprints/3/proposals.md`. Điều kiện trước story đầu: `docs/sprints/3/qc/audit-baseline.md` đã có.

## Làm gì
- `scripts/ui-antipatterns.sh` viết lại thành **19 phép** (SRS 4.1 + 4.3) + `--selftest` (gieo từng vi phạm vào bản sao `frontend/src`, mong `✗ <tên>`). Ba vi phạm thật bị bắt và đã sửa: `AppShell.module.css` bo `0 2px 2px 0` → `0`; `font-size` số cứng ở `SubmissionReview` / `Field` / `Composer` → `--ep-text-page` / `--ep-text-item`; `Chart.tsx` bảng sr-only → vai ARIA (góp ý #12). `ui-allow:` vẫn **10** chỗ.
- `frontend/eslint-rules/ep.mjs` + `eslint.config.mjs`: 7 luật `ep/*` (góp ý #11), ngoại lệ đường dẫn mỗi luật một khối.
- `scripts/lint-selftest.sh`: gieo từng luật vào `frontend/src/__selftest__/` → `7 / 7 luật ESLint bắt được`, rồi gọi `--selftest` → `19 / 19 phép bắt được`; dọn tệp kể cả khi lỗi.
- `layout.tsx`: bỏ import `base.css` trùng; font qua `variable` (không còn `className` đè `font-family`, nên `--ep-font` với "Noto Sans" dự phòng là chuỗi font duy nhất).
- `base.css`: `color-scheme: light`. `AppShell`: thanh trên mobile cũng có `data-part="brand"`.
- Playwright (`@playwright/test` 1.63, thư viện thuộc kế hoạch): `frontend/playwright.config.ts` (desktop 1440×900, mobile 390×844, `webServer` = `next start -p 3300`, `reuseExistingServer`), `e2e/support/session.ts` (phiên mô phỏng bằng cookie `ep_demo_*`), `e2e/tokens.spec.ts`.

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `comm -23` = **0**, `comm -13` = **4** (đúng 4 token SRS 5.2), khai báo `--ep-*` ngoài `tokens.css` = **0** |
| 2 | `pnpm -C frontend lint` rc=0; `bash scripts/ui-antipatterns.sh` rc=0, 19 dòng `✓`; `grep -rn 'ui-allow:' frontend/src \| wc -l` = **10** |
| 3 | `bash scripts/lint-selftest.sh` rc=0: `7 / 7 luật ESLint bắt được`, `19 / 19 phép bắt được`; `git status --short frontend` không còn tệp `__selftest__` |
| 4 | Lệnh grep của AC: `fetch(` ngoài `shared/data` 0; `confirm(`/`alert(` 0; `tailwind` trong `package.json` 0; `@tailwind`/`@theme` 0; 7 luật chạy thật trên `frontend/src` không báo lỗi |
| 5 | `tokens.spec.ts › font` (desktop + mobile): `document.fonts.check` đúng cho 400/500/600/700, `body` bắt đầu `"Be Vietnam Pro"`, 0 request tới `fonts.googleapis/gstatic`, `weight: [` 1 dòng chỉ 4 độ đậm, `display: "swap"`, tập con có `vietnamese` |
| 6 | `tabular` (`/gradebook`, GV): mọi `td`/`th` có `tabular-nums` (DataTable); `diacritics` ("Ặ Ế Ộ Ử Ữ Ầ") ở `h1`, `h2`, nút, ô nhập: `scrollHeight ≤ clientHeight + 1`; `h1` cao ≥ 1,25 × cỡ chữ. (Ô nhập/nút lấy từ `/students` vì `/dev/ui` chưa có — chuyển sang `/dev/ui` ở US-PU-02) |
| 7 | `focus` (Tab → `box-shadow` = `--ep-focus`; bấm chuột → không còn vòng), `reduced` (≤ 0,001 s), `selection` (`::selection` = `--ep-red-soft`), `color-scheme: light` — pass ở cả hai dự án |
| 8 | `brand`: `[data-part=brand] img` `src` kết thúc `logo-edupilot(-mark).svg`, cha `border-radius:0px`, nền `rgba(0, 0, 0, 0)`, `/brand/favicon.svg` 200, `<link rel=icon>` đúng |
| 9 | `pnpm -C frontend build` rc=0, không dòng `Failed to load font`. **Chưa chạy `audit.mjs` của QC 1.5**: ở máy dev `Page.captureScreenshot` hết hạn với trình duyệt dùng chung (baseline của QC dùng Chrome for Testing riêng + `caffeinate`). Việc nới / sửa chỉ ở 5 tệp CSS/TSX nêu trên, không đổi bố cục — QC chạy lượt `student/teacher/ta/admin` theo `audit-baseline.md` |
| 10 | `grep -rnE "(localStorage\|sessionStorage)\.setItem\(…(token\|jwt…)" frontend/src \| wc -l` = **0** |

## Nợ / ghi chú
- `next start` in cảnh báo "does not work with output: standalone" nhưng chạy đúng (Playwright `webServer`); Dockerfile dùng `standalone/server.js`.
- Ảnh mốc 14 cảnh (US-PU-05) chưa nộp ⇒ AC9 so mắt với `docs/sprints/3/qc/shots/before/` (Q-QC-PU01-2).
