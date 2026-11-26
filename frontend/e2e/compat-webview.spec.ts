import { expect, test } from "@playwright/test";
import { checkinApi } from "./support/checkin-fixtures";
import { asDemo } from "./support/session";

// DEF-MOB-12b (P-11-149): WebView Android cũ = Chrome 64–66 (AVD API 28). Bundle đã parse (browserslist chrome>=64) nhưng app chết im lặng:
// React #418 khi hydrate và KHÔNG gửi /api/v1 nào (kẹt khung mô phỏng, nút Điểm danh khoá).
// Cách kiểm trên Chromium hiện đại: XOÁ đúng những API mà Chrome 64–66 không có, rồi đòi app chạy như thường.
const CHROME66_GAPS = `(() => {
  delete AbortSignal.any;                 // Chrome 116
  delete Promise.allSettled;              // 76
  delete Array.prototype.flat;            // 69
  delete Array.prototype.flatMap;         // 69
  delete String.prototype.replaceAll;     // 85
  delete Object.hasOwn;                   // 93
  delete Array.prototype.findLast;        // 97
  delete window.structuredClone;          // 98
  delete Intl.RelativeTimeFormat;         // 71
  delete Intl.ListFormat;                 // 72
  delete Crypto.prototype.randomUUID;     // 92
})()`;

test("webview cũ: khung app dựng và có gọi /me/courses + /me/today (không kẹt mô phỏng)", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "một ca là đủ");
  await page.addInitScript(CHROME66_GAPS);
  const reqs: string[] = [];
  page.on("request", (r) => {
    const u = r.url();
    if (u.includes("/api/v1/")) reqs.push(u.slice(u.indexOf("/api/v1") + 8));
  });
  await asDemo(context, "student");
  await page.goto("/");
  await expect(page.locator("[data-part=sidebar], [data-part=bottom-nav]").first()).toBeVisible();
  await expect(page.getByText("Bản mô phỏng · dữ liệu giả")).toHaveCount(0); // phiên thật, không rơi về mô phỏng
  // truy vấn chạy sau khi khung dựng: chờ tới 5 s thay vì đọc ngay
  await expect.poll(() => reqs.some((u) => u.startsWith("me/courses")), { timeout: 5000 }).toBe(true);
  await expect.poll(() => reqs.some((u) => u.endsWith("/today")), { timeout: 5000 }).toBe(true); // «Hôm nay» của lớp đang chọn
});

test("webview cũ: điểm danh bằng mã vẫn gửi POST và báo Đã điểm danh", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "một ca là đủ");
  await page.addInitScript(CHROME66_GAPS);
  const log = await checkinApi(page, context);
  await page.goto("/checkin");
  const field = page.getByLabel("Mã điểm danh");
  await expect(field).toBeVisible();
  await field.fill("ABC123");
  await page.getByRole("button", { name: "Điểm danh", exact: true }).first().click();
  await expect(page.locator("[data-part=done]")).toHaveText("Đã điểm danh 07:03");
  expect(log.posts).toHaveLength(1);
});
