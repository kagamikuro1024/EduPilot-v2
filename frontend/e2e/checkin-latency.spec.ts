import { expect, test } from "@playwright/test";
import { COURSE, TOKEN, checkinApi } from "./support/checkin-fixtures";

// L3 nghiệm thu 2: đã đăng nhập, từ `goto` tới «Đã điểm danh» 0 thao tác, p95 ≤ 1,5 s trên 20 lần. Ở đây gateway giả trả ngay ⇒ đo phần FRONTEND của đường nóng
// (tải trang, làm mới phiên, đọc chính sách lớp, ký WebCrypto, POST). Số trên stack thật (mạng + máy chủ) đo ở L7.
test("checkin latency: goto → «Đã điểm danh» p95 ≤ 1,5 s trên 20 lần (gateway giả)", async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  test.setTimeout(120_000);
  const ms: number[] = [];
  for (let i = 0; i < 20; i++) {
    const page = await context.newPage();
    await checkinApi(page, context);
    const t0 = Date.now();
    await page.goto(`/checkin?c=${COURSE}#${TOKEN}`);
    await expect(page.locator("[data-part=done]")).toBeVisible({ timeout: 5000 });
    ms.push(Date.now() - t0);
    await page.close();
  }
  ms.sort((a, b) => a - b);
  const p95 = ms[Math.ceil(0.95 * ms.length) - 1];
  console.log(`checkin latency ms: ${ms.join(",")} p95=${p95}`);
  expect(p95).toBeLessThanOrEqual(1500);
});
