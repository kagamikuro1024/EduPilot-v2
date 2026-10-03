import { defineConfig } from "@playwright/test";
import { API_PORT, API_URL, PORT } from "./e2e/support/env";

// Gate giao diện (FEAT-ui-foundation SRS 8): hai dự án desktop 1440 × 900 và mobile 390 × 844, chạy trên bản build (`next start -p 3310`).
// Ca gắn @real cần gateway thật và KHÔNG chạy trong CI (CI không dựng stack).

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: true,
  // CI: 2 worker, thử lại 1 lần (ảnh mốc đặt retries: 0 riêng trong visual.spec.ts để không che lỗi lệch ảnh)
  workers: process.env.CI ? 2 : undefined,
  retries: process.env.CI ? 1 : 0,
  // ảnh mốc một tệp cho mỗi (route, bề rộng), không đuôi dự án / hệ điều hành: `e2e/visual.spec.ts-snapshots/<tên>.png` (US-PU-05 AC1)
  snapshotPathTemplate: "{testDir}/{testFilePath}-snapshots/{arg}{ext}",
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : [["list"]],
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
    { command: `API_PORT=${API_PORT} node e2e/support/api-server.mjs`, url: `${API_URL}/__ctl/ping`, reuseExistingServer: true, timeout: 20_000 },
  ],
});
