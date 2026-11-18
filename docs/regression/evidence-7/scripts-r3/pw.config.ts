import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: process.env.SPECDIR ?? "/tmp/qc3e/specs", testMatch: /.*\.spec\.ts/, timeout: 300_000, expect: { timeout: 8_000 },
  outputDir: "/tmp/qc3e/pw-out", workers: 1, retries: 0, reporter: [["list"]],
  use: { baseURL: "https://localhost:1143", actionTimeout: 15_000, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" },
  projects: [{ name: "desktop", use: { viewport: { width: 1280, height: 800 } } }],
});
