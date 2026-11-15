import { defineConfig } from "@playwright/test";

// QC sprint 10.6: spec `qc-gd-*-real.spec.ts` trên stack thật `ep-qc` (offset 500 → https://localhost:943). Không webServer.
export default defineConfig({
  testDir: "./e2e",
  testMatch: /qc-gd-.*-real\.spec\.ts/,
  timeout: 90_000,
  expect: { timeout: 8_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: { baseURL: process.env.QC_BASE ?? "https://localhost:943", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }],
});
