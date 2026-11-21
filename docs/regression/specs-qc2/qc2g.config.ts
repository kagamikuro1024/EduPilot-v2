import { defineConfig } from "@playwright/test";
// QC2 nhóm G: chạy bản sao spec trong docs/regression/evidence-*/specs trên ep-r (không webServer)
export default defineConfig({
  testDir: require("node:path").resolve(process.cwd(), process.env.QC2G_DIR ?? "../docs/regression/evidence-10.6/specs"),
  testMatch: /.*\.spec\.ts/,
  timeout: 120_000,
  expect: { timeout: 8_000 },
  workers: 1, retries: 0, reporter: [["list"]],
  use: { baseURL: process.env.QC_BASE ?? "https://localhost:1143", actionTimeout: 15_000, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }],
});
