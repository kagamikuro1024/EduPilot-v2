// Cổng của Next trong lượt Playwright: một nguồn duy nhất (đặt E2E_PORT để đổi). Không cứng cổng ở nơi khác.
export const PORT = Number(process.env.E2E_PORT ?? 3310);
export const BASE_URL = `http://localhost:${PORT}`;
