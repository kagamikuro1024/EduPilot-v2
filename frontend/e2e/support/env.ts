// Cổng của Next trong lượt Playwright: một nguồn duy nhất (đặt E2E_PORT để đổi). Không cứng cổng ở nơi khác.
export const PORT = Number(process.env.E2E_PORT ?? 3310);
export const BASE_URL = `http://localhost:${PORT}`;
// Cổng gateway GIẢ (api-server.mjs) — build:gate đặt NEXT_PUBLIC_API_URL tới đây; đổi bằng E2E_API_PORT khi chạy song song nhiều worktree.
export const API_PORT = Number(process.env.E2E_API_PORT ?? 3312);
export const API_URL = `http://localhost:${API_PORT}`;
