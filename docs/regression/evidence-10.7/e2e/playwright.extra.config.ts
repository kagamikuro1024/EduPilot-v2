import { defineConfig } from "@playwright/test";
export default defineConfig({ testDir: ".", testMatch: /qc-r107-landing-extra\.spec\.ts/, timeout: 300_000, workers: 1, retries: 0, reporter: [["list"]], use: { ignoreHTTPSErrors: true, locale: "vi-VN" } });
