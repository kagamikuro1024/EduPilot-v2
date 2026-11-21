import { defineConfig } from "@playwright/test";
// QC2 hồi quy sprint 3: spec trong e2e/qc2-r3 chạy trên ep-r (không webServer)
export default defineConfig({
  testDir: "./qc2-r3",
  testMatch: /.*\.spec\.ts/,
  timeout: 120_000,
  expect: { timeout: 8_000 },
  workers: 1, retries: 0, reporter: [["list"]],
  use: { baseURL: process.env.QC_BASE ?? "https://localhost:1143", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }, { name: "mobile", use: { viewport: { width: 375, height: 812 }, hasTouch: true } }],
});
