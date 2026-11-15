import { defineConfig } from "@playwright/test";
export default defineConfig({ testDir: ".", testMatch: /dbg-r108\.spec\.ts/, timeout: 60000, reporter: [["list"]], use: { ignoreHTTPSErrors: true } });
