import { defineConfig } from "@playwright/test";

// QC sprint 10: chạy spec `qc-s10-*.spec.ts` trên stack thật ep-qc (https://localhost:943), không webServer.
export default defineConfig({
  testDir: "./e2e",
  testMatch: /qc-s10-.*\.spec\.ts/,
  timeout: 90_000,
  expect: { timeout: 8_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: { actionTimeout: 10_000, baseURL: process.env.QC_BASE ?? "https://localhost:943", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }],
});
