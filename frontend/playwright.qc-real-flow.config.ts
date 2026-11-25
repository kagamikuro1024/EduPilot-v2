import { defineConfig } from "@playwright/test";

// QC @real (P-11-29): spec `qc-real-*.spec.ts` trên stack thật ep-qc (không webServer, không gateway giả).
// QC_BASE = origin HTTPS của Caddy ep-qc (cùng origin với /api/v1), mặc định https://localhost:943.
export default defineConfig({
  testDir: "./e2e",
  testMatch: /qc-real-.*\.spec\.ts/,
  timeout: 120_000,
  expect: { timeout: 10_000 },
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: process.env.QC_BASE ?? "https://localhost:943",
    ignoreHTTPSErrors: true,
    locale: "vi-VN",
    timezoneId: "Asia/Ho_Chi_Minh",
  },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }],
});
