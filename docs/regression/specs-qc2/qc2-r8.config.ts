import { defineConfig } from "@playwright/test";
// QC2 hồi quy sprint 8: spec UI thật trên stack ep-r (https://localhost:1143), e2e/qc2-r8/*.spec.ts, --workers=1.
export default defineConfig({
  testDir: "./qc2-r8", testMatch: /.*\.spec\.ts/, timeout: 240_000, expect: { timeout: 10_000 }, workers: 1, retries: 0,
  reporter: [["list"], ["json", { outputFile: process.env.PW_JSON ?? "/tmp/qc2-s8r/pw.json" }]],
  use: { baseURL: process.env.QC_BASE ?? "https://localhost:1143", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1440, height: 900 } } }],
});
