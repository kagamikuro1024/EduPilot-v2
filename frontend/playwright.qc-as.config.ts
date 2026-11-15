import { defineConfig } from "@playwright/test";

// QC 10.8: chạy qc-as-real.spec.ts trên stack thật ep-qc (không webServer).
export default defineConfig({
  testDir: "./e2e",
  testMatch: /qc-as-(real|android)\.spec\.ts/,
  timeout: 120_000,
  expect: { timeout: 10_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: { baseURL: process.env.QC_BASE ?? "http://localhost:580", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
});
