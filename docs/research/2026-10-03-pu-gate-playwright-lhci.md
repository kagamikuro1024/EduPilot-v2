# Cổng PU trên máy dev và CI: Playwright + axe + Lighthouse CI với Next 16 `output: standalone` (SRS FEAT-ui-foundation 8.3–8.6)

**Câu hỏi.** Cấu hình nào cho Playwright, `@axe-core/playwright` và Lighthouse CI chạy ổn định với Next 16 `output: "standalone"` trên cả macOS (máy dev, Chrome headless từng treo khi màn hình ngủ) lẫn GitHub Actions?

**Kết luận.**
1. **Chạy server standalone đúng như Dockerfile, và phải chép `.next/static` + `public` vào thư mục standalone.** Không chép thì HTML trả 200 nhưng CSS/JS trả **307 → /login**, trang mất style; LHCI và ảnh mốc vẫn "chạy" nhưng đo sai. `next start` vẫn chạy được nhưng Next 16 cảnh báo: `"next start" does not work with "output: standalone"`.
2. **Một trình duyệt cho cả Playwright lẫn LHCI: `chromium-headless-shell` của Playwright** (mặc định khi không đặt `channel`). LHCI trỏ tới nó qua `CHROME_PATH`; PoC chạy được 3/3 lần. Lợi ích: máy dev và CI dùng cùng một bản, và CI chỉ cần `playwright install --with-deps --only-shell chromium`.
3. **Màn hình ngủ:** **không tái hiện được** trên máy này — `pmset displaysleepnow` bị bật sáng lại sau 1 s (log `pmset` ghi rõ, hai lần thử). Trong cửa sổ đó cả 3 cấu hình (headless shell, `--headless=new`, `--headless=new` + swiftshader) đều cho rAF và chụp ảnh bình thường. Khuyến nghị tránh từ gốc: chạy cổng trên máy bằng `caffeinate -dims`; cờ PM đã thử (`--disable-gpu --use-angle=swiftshader`) giữ trong `chromeFlags` của LHCI làm lưới an toàn. Nguyên nhân treo ghi là `[SUY LUẬN]`.
4. **LHCI mặc định chấm theo lần chạy *tốt nhất*** (`aggregationMethod: "optimistic"`), không phải trung vị như SRS 8.5 viết. PoC: cùng 3 bản ghi LCP 2789 / 2413 / 3326 ms → mặc định **qua**, `median` **trượt** (2789 > 2500). Phải ghi `"aggregationMethod": "median"` trong từng assertion.
5. **`puppeteerScript` đòi cài thêm `puppeteer`** — thư viện này không có trong bảng ARCHITECTURE. Thay bằng `settings.extraHeaders` chứa cookie `ep_demo_role=…` (PoC: `/inbox` với cookie GV → 200, không chuyển hướng), mỗi vai một lần `lhci collect --additive`.
6. **Ảnh mốc phụ thuộc nền tảng:** tên tệp kèm `-darwin` / `-linux`, và Playwright ghi rõ ảnh khác nhau giữa các nền tảng. Ảnh mốc phải tạo và so trên Linux — trên Mac thì chạy trong `mcr.microsoft.com/playwright:v1.63.0-noble` (colima), trùng với CI.

Độ chắc chắn: **cao** cho mục 1, 2, 4, 5, 6 (PoC + tài liệu); **thấp** cho nguyên nhân treo khi màn hình ngủ (không tái hiện được).

## Phương án — trình duyệt

| Tiêu chí | A. `chromium-headless-shell` (mặc định Playwright) cho cả hai | B. Chromium đầy đủ `--headless=new` (`channel: "chromium"`) | C. Google Chrome hệ thống cho LHCI |
| --- | --- | --- | --- |
| Playwright + axe trên Mac (PoC) | Đạt | Đạt | — |
| LHCI trên Mac (PoC, 3 lần) | Đạt; LCP 3339 / 2340 / 3305 ms | Đạt; LCP 2789 / 2413 / 3326 ms | Máy dev **không có** Chrome |
| Cùng bản trên Mac và CI | Có (ghim theo Playwright) | Có | Không (runner tự cập nhật Chrome) |
| Phụ thuộc màn hình / GPU của macOS | Thấp [SUY LUẬN: bản "old headless" tách khỏi trình duyệt có giao diện] | Có thể — PM từng gặp treo; [SUY LUẬN] dựng hình theo màn hình thật | Như B |
| Dung lượng tải ở CI | `--only-shell` (nhỏ nhất) | Bản đầy đủ | 0 (có sẵn) |
| Tài liệu Lighthouse khuyến nghị | Không nói rõ [SUY LUẬN] | Chrome "thật" | Có |

Chọn **A**; nếu Lighthouse báo lỗi lạ với headless shell thì lùi sang **B** + `--disable-gpu --use-angle=swiftshader`.

## Bằng chứng

Phiên bản (npm, 2026-10-03): `@playwright/test` 1.63.0, `@axe-core/playwright` 4.13.0 (`axe-core` 4.13.0), `@lhci/cli` 0.15.1 (2025-06-25; ghim `lighthouse` **12.6.1** trong khi bản mới nhất là 13.5.0), `next` 16.3.8; Node 24.21.0; pnpm 12.8.1. Ảnh Docker `mcr.microsoft.com/playwright:v1.63.0-noble` có sẵn (danh sách tag MCR).

Nguồn tài liệu:
- Playwright [Browsers — Chromium: headless shell / new headless mode](https://github.com/microsoft/playwright/blob/v1.63.0/docs/src/browsers.md): "Playwright ships a regular Chromium build for headed operations and a separate chromium headless shell for headless mode"; `--only-shell` khi không đặt `channel`; `channel: 'chromium'` để dùng new headless.
- Playwright [Visual comparisons](https://github.com/microsoft/playwright/blob/v1.63.0/docs/src/test-snapshots-js.md): "`chromium-darwin` - the browser name and the platform. Screenshots differ between browsers and platforms due to different rendering, fonts and more, so you will need different snapshots for them."
- LHCI 0.15.1 [configuration.md](https://github.com/GoogleChrome/lighthouse-ci/blob/v0.15.1/docs/configuration.md): `chromePath` → `CHROME_PATH` → puppeteer → chrome-launcher; "In order to use puppeteer scripts, you need to install `puppeteer` yourself"; mục "Page Behind Authentication" dùng `settings.extraHeaders` chứa `Cookie`; "When no options are set, the default options of `{"aggregationMethod": "optimistic", "minScore": 1}` are used"; `median` = "Use the median value from all runs".
- `frontend/Dockerfile` của repo: chép `.next/standalone` rồi riêng `.next/static` → `frontend/.next/static` và `public` → `frontend/public`, chạy `node frontend/server.js` (monorepo pnpm nên standalone lồng thêm thư mục `frontend/`).

PoC (`/tmp/research-pu/fe` = bản chép `frontend/` của `sprint/3-pu-p1` @ `a714ee2`, cài riêng `--ignore-workspace`; repo không bị đụng):

```
# next start trên bản standalone (cổng 3301)
⚠ "next start" does not work with "output: standalone" configuration. Use "node .next/standalone/server.js" instead.
# so sánh phục vụ tài nguyên của /login
port 3301 (next start):                     html=200 css=200 js=200
port 3302 (standalone, KHÔNG chép static):  html=200 css=307 js=307   (Location: /login)
port 3303 (standalone, đã chép static+public): html=200 css=200 js=200
# vai qua cookie (cách của extraHeaders)
no cookie / → 307 /login ;  Cookie: ep_demo_role=teacher /inbox → 200
```

Playwright (`playwright.config.ts` với `webServer` = chép static + `node .next/standalone/server.js`, hai project `shell` / `new-headless`; `e2e/smoke.spec.ts`: cookie GV, `page.clock.setFixedTime`, chờ `document.fonts.ready` + 2 rAF, axe với tag `wcag2a, wcag2aa, wcag21aa, wcag22aa`, chụp ảnh):

```
Running 2 tests using 2 workers
font="Be Vietnam Pro" axe: 1 vi phạm, chặn=color-contrast
  ✓  1 [shell] › e2e/smoke.spec.ts:8:5 › inbox … (1.2s)
font="Be Vietnam Pro" axe: 1 vi phạm, chặn=color-contrast
  ✓  2 [new-headless] › e2e/smoke.spec.ts:8:5 › inbox … (2.2s)
  2 passed (4.7s)
```

`/inbox` của prototype hiện có vi phạm `color-contrast` mức serious → khi bật cổng (SRS 8.4 chặn critical + serious) cổng sẽ **đỏ đúng**. Ảnh của hai project khác nhau (151,4 KB và 154,0 KB): ảnh mốc phải gắn với một cấu hình trình duyệt.

LHCI (`lighthouserc.json`: URL `/inbox`, 3 lần, preset mobile mặc định, `extraHeaders` cookie GV, `chromeFlags "--headless=new --disable-gpu --use-angle=swiftshader"`, 4 assertion của SRS 8.5):

```
CHROME_PATH=…/chromium-1243/…/Google Chrome for Testing   → autorun 35 s, "All results processed!" (mặc định optimistic: QUA)
  LCP/TBT/CLS/script: 2789/24/0/196245  2413/20/0/196245  3325/16/0/196245   (finalDisplayedUrl = /inbox)
cùng 3 bản ghi, thêm "aggregationMethod": "median":
  ✘ largest-contentful-paint … expected: <=2500 found: 2789.1345
    all values: 2789.1345, 2413.3621999999996, 3325.808525      → exit 1
CHROME_PATH=…/chromium_headless_shell-1243/…/chrome-headless-shell → collect 31 s
  LCP/TBT/CLS: 3339/27/0  2340/14/0  3305/10/0
```

Thử màn hình ngủ (`probe.mjs`: ba cấu hình chạy song song, mỗi 3 s đo 2 rAF và một lần chụp ảnh, hạn 3 s / 5 s): gọi `pmset displaysleepnow` lúc 12:35:39 rồi `caffeinate -u` lúc 12:36:22. Cả ba cấu hình đều có rAF 2–133 ms và chụp ảnh 22–571 ms suốt thời gian thử, không lần nào quá hạn. Nhưng `pmset -g log` cho thấy màn hình chỉ tắt **1 giây**: `12:35:39 Display is turned off` → `12:35:40 Display is turned on`; lần thử lại không có probe chạy cũng vậy (`12:37:17` off → `12:37:17` on). Vì vậy phép thử **không** chứng minh được cấu hình nào miễn nhiễm. Log cũng cho thấy Chrome tự tạo `NoDisplaySleepAssertion "Capturing"` mỗi lần chụp ảnh.

## Đề xuất cấu hình

`frontend/playwright.config.ts` (phần chính; đường dẫn theo monorepo như Dockerfile):

```ts
webServer: {
  command:
    "rm -rf .next/standalone/frontend/.next/static .next/standalone/frontend/public" +
    " && cp -r .next/static .next/standalone/frontend/.next/static" +
    " && cp -r public .next/standalone/frontend/public" +
    " && PORT=3300 HOSTNAME=127.0.0.1 node .next/standalone/frontend/server.js",
  url: "http://127.0.0.1:3300/login",
  reuseExistingServer: !process.env.CI,
},
use: { baseURL: "http://127.0.0.1:3300", locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", reducedMotion: "reduce" },
// không đặt `channel` → chromium-headless-shell
```

Chờ khung hình trong test (nếu cần), theo quy ước của repo dùng `Promise.withResolvers`:

```ts
await page.evaluate(() => {
  const { promise, resolve } = Promise.withResolvers<void>();
  requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
  return promise;
});
```

`frontend/lighthouserc.json`: giữ 4 assertion của SRS 8.5 và thêm `"aggregationMethod": "median"` vào mỗi cái; `collect.settings.chromeFlags = "--disable-gpu --use-angle=swiftshader"`; **không** dùng `puppeteerScript`. Chạy mỗi vai một lần rồi assert một lần:

```bash
export CHROME_PATH=$(ls -d "${PLAYWRIGHT_BROWSERS_PATH:-$HOME/$([ "$(uname)" = Darwin ] && echo Library/Caches || echo .cache)/ms-playwright}"/chromium_headless_shell-*/chrome-headless-shell-*/chrome-headless-shell | tail -1)
lhci collect --collect.settings.extraHeaders='{"Cookie":"ep_demo_role=student; ep_demo_person=sv-2"}' --url=http://127.0.0.1:3300/ …
lhci collect --additive --collect.settings.extraHeaders='{"Cookie":"ep_demo_role=teacher"}' --url=…/inbox --url=…/gradebook
lhci collect --additive --collect.settings.extraHeaders='{"Cookie":"ep_demo_role=admin"}'   --url=…/settings/llm
lhci assert && lhci upload
```

Ghi chú `CHROME_PATH`: Playwright chỉ có hàm trả đường dẫn của Chromium đầy đủ (`chromium.executablePath()`), không có hàm cho headless shell; dòng `ls` ở trên tìm headless shell ở thư mục mặc định của Playwright trên cả macOS (`~/Library/Caches/ms-playwright`) lẫn Linux (`~/.cache/ms-playwright`). Kiểm thật trên máy dev: ra `…/chromium_headless_shell-1243/chrome-headless-shell-mac-arm64/chrome-headless-shell`, `--version` → `Google Chrome for Testing 153.0.8010.12`.

GitHub Actions (job Frontend, SRS 8.6): cache `~/.cache/ms-playwright` theo khoá phiên bản `@playwright/test`; `pnpm -C frontend exec playwright install --with-deps --only-shell chromium`; `pnpm -C frontend build`; `playwright test`; rồi các lệnh LHCI ở trên. Ảnh mốc tạo bằng cùng môi trường Linux: trên Mac chạy trong `mcr.microsoft.com/playwright:v1.63.0-noble` qua colima (worktree nằm trong `$HOME` nên colima mount được — xem `2026-10-02-testcontainers-colima.md`). [SUY LUẬN, chưa chạy thử] `node_modules` cài trên macOS có gói nhị phân riêng nền tảng (`@next/swc-darwin-*`), nên trong container phải `pnpm install --frozen-lockfile` và `pnpm -C frontend build` lại vào một volume riêng cho `node_modules`, không dùng chung với máy. Muốn CI trùng tuyệt đối thì job chạy trong `container: mcr.microsoft.com/playwright:v1.63.0-noble`.

Máy dev: chạy cổng bằng `caffeinate -dims pnpm -C frontend <lệnh cổng>` để màn hình không ngủ trong lúc chạy. Đây là chặn ở gốc, công cụ có sẵn trên macOS, không cần cờ trình duyệt.

## Ảnh hưởng

- SRS 8.3 ghi "Nền: `next start` trên bản `gbuild`" → đổi thành server standalone + chép static/public (cùng sản phẩm với Dockerfile); `next start` vẫn chạy được nếu PM muốn giữ, nhưng Next 16 cảnh báo và không kiểm đúng thứ được đóng gói.
- SRS 8.5: thay `puppeteerScript` bằng `extraHeaders` (tránh thêm `puppeteer` ngoài bảng ARCHITECTURE §3); ghi rõ `aggregationMethod: "median"` cho khớp chữ "trung vị 3 lần".
- Ngân sách LCP 2,5 s (mobile, CPU 4×) trên `/inbox`: PoC đo trung vị 2,8–3,3 s trên máy này → cổng LHCI có thể đỏ ngay. Đây là số đo, không phải lý do nới ngưỡng; dev cần tối ưu, hoặc PM quyết như Q14 của sprint 2 (ghi số đo + cấu hình máy).
- axe: `/inbox` của prototype đang vi phạm `color-contrast` (serious) → dev sửa token màu hoặc đưa vào `axe-allow.json` (≤ 3, SRS 8.4).
- `@lhci/cli` 0.15.1 chưa có bản mới từ 2025-06 và kéo Lighthouse 12.6.1; chấp nhận được (cùng thư viện trong bảng), ghi rủi ro "LHCI chậm cập nhật" vào PROGRESS.
- Không đổi `ARCHITECTURE.md` / `DECISIONS.md`; không thêm thư viện.

## Đề xuất cho PM

`PU gate: chạy server standalone (chép .next/static + public; không chép → CSS/JS 307) thay next start; dùng chromium-headless-shell của Playwright cho cả Playwright và LHCI (CHROME_PATH, CI --only-shell); LHCI thêm aggregationMethod "median" (mặc định optimistic: PoC LCP 2789/2413/3326 ms qua với mặc định, trượt với median) và dùng extraHeaders cookie thay puppeteerScript (puppeteer ngoài bảng thư viện); ảnh mốc tạo/so trên Linux (ảnh Docker playwright:v1.63.0-noble); máy dev chạy cổng dưới caffeinate -dims — không tái hiện được treo khi màn hình ngủ (màn hình tự bật lại sau 1 s) — docs/research/2026-10-03-pu-gate-playwright-lhci.md.`
