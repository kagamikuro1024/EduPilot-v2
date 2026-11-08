import { defineConfig } from "@playwright/test";

// QC: chạy spec `qc-uiux91-*-real.spec.ts` trên stack thật `ep-qc` (không webServer). `QC_BASE` mặc định https://localhost:14443.
export default defineConfig({
  testDir: "./e2e",
  testMatch: /qc-uiux91-.*-real\.spec\.ts/,
  timeout: 90_000,
  expect: { timeout: 8_000 },
  workers: 2,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: process.env.QC_BASE ?? "https://localhost:14443",
    ignoreHTTPSErrors: true,
    locale: "vi-VN",
    timezoneId: "Asia/Ho_Chi_Minh",
  },
  projects: [
    { name: "desktop", use: { viewport: { width: 1440, height: 900 } } },
    { name: "mobile", use: { viewport: { width: 390, height: 844 }, hasTouch: true } },
  ],
});
