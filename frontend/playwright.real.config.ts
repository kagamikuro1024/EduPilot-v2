import { defineConfig } from "@playwright/test";

// Spec chạy API THẬT (`*.real.spec.ts`, DEF-R10S-13 / P-11-111): stack đã seed (`pnpm seed`), không webServer, không gateway giả. Không chạy trong CI (CI không dựng stack).
//   QC_REAL=1 QC_BASE=https://localhost:<cổng HTTPS> QC_PW=<SEED_DEFAULT_PASSWORD> pnpm exec playwright test -c playwright.real.config.ts
// Giờ mở bài thi: EXAM_MIN_LEAD_SECONDS ≤ 14 và EXAM_MIN_DURATION_MINUTES ≤ 2 (cùng giá trị mà `scripts/seed.mjs` cần cho hai bài mẫu).
export default defineConfig({
  testDir: "./e2e",
  testMatch: /\.real\.spec\.ts/,
  timeout: 300_000,
  expect: { timeout: 10_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: { baseURL: process.env.QC_BASE ?? "https://localhost:943", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [
    { name: "desktop", use: { viewport: { width: 1440, height: 900 } } },
    { name: "mobile", use: { viewport: { width: 375, height: 812 }, hasTouch: true } },
  ],
});
