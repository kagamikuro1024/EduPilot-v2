import { defineConfig } from "@playwright/test";

// QC 10.8 mobile pha 1: Playwright thiết bị giả (Chromium) chạy trên VPS, trỏ HTTPS thật của VPS (không webServer).
export default defineConfig({
  testDir: "./e2e",
  testMatch: /qc-mob-pha1.*\.spec\.ts/,
  timeout: 240_000,
  expect: { timeout: 10_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: { ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
});
