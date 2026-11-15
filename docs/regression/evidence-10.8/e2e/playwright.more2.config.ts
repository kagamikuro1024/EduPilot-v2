import { defineConfig } from "@playwright/test";
export default defineConfig({ testDir: ".", testMatch: /qc-r108-more2\.spec\.ts/, timeout: 300_000, expect: { timeout: 10_000 }, workers: 1, retries: 0, reporter: [["list"]], use: { baseURL: process.env.QC_BASE ?? "https://localhost:1143", ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" } });
