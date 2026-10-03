import { defineConfig } from "@playwright/test";

// Gate giao diện (FEAT-ui-foundation SRS 8): hai dự án desktop 1440 × 900 và mobile 390 × 844, chạy trên bản build (`next start -p 3300`).
// Ca gắn @real cần gateway thật và KHÔNG chạy trong CI (CI không dựng stack).
const PORT = 3300;

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: true,
  reporter: [["list"]],
  use: {
    baseURL: `http://localhost:${PORT}`,
    locale: "vi-VN",
    timezoneId: "Asia/Ho_Chi_Minh",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "desktop", use: { viewport: { width: 1440, height: 900 } } },
    { name: "mobile", use: { viewport: { width: 390, height: 844 }, hasTouch: true } },
  ],
  webServer: {
    command: `pnpm exec next start -p ${PORT}`,
    url: `http://localhost:${PORT}/login`,
    reuseExistingServer: true,
    timeout: 60_000,
  },
});
