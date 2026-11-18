import { defineConfig } from "@playwright/test";
// QC2 s8flow: stack thật ep-r (https://localhost:1143), spec ở e2e/qc2-s8flow/real-*.spec.ts, Playwright --workers=1.
export default defineConfig({
  testDir: "./e2e/qc2-s8flow", testMatch: /real-.*\.spec\.ts/, timeout: 180_000, expect: { timeout: 10_000 }, workers: 1, retries: 0,
  reporter: [["list"], ["json", { outputFile: process.env.PW_JSON ?? "/tmp/qc2-s8f/pw.json" }]],
  use: { baseURL: process.env.QC_BASE ?? "https://localhost:1143", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }],
});
