import { defineConfig } from "@playwright/test";

// Gate giao diện (FEAT-ui-foundation SRS 8): hai dự án desktop 1440 × 900 và mobile 390 × 844, chạy trên bản build (`next start -p 3310`).
// Ca gắn @real cần gateway thật và KHÔNG chạy trong CI (CI không dựng stack).
const PORT = 3310;

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
  webServer: [
    {
      command: `pnpm exec next start -p ${PORT}`,
      url: `http://localhost:${PORT}/login`,
      reuseExistingServer: true,
      timeout: 60_000,
    },
    // máy chủ gateway GIẢ cho test lớp dữ liệu (bản dựng cổng đặt NEXT_PUBLIC_API_URL tới đây)
    { command: "node e2e/support/api-server.mjs", url: "http://localhost:3311/__ctl/ping", reuseExistingServer: true, timeout: 20_000 },
  ],
});
