import { expect, test } from "@playwright/test";
import { asDemo, sessionBody } from "./support/session";
import { API_URL, BASE_URL } from "./support/env";

// US-P10-08 U1: `/auth/refresh` bắt đầu từ <script> trong <head> (song song với tải JS) nhưng vẫn ĐÚNG MỘT lần mỗi lần tải trang, và tuần tự giữa các tab
// (cookie xoay vòng: dùng lại token cũ là bị thu hồi phiên).
test.describe.configure({ retries: 0 });
test.beforeEach(({}, info) => test.skip(info.project.name !== "desktop", "đo một lần"));

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };

test("early refresh: một tải trang = đúng một POST /auth/refresh; lời gọi sớm được dùng một lần rồi xoá khỏi window", async ({ page, context }) => {
  await asDemo(context, "student");
  let calls = 0;
  const urls: string[] = [];
  page.on("request", (r) => { if (r.url().includes("/auth/refresh")) urls.push(r.url()); });
  await page.route("**/api/v1/auth/refresh", (route) => {
    calls++;
    return route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(sessionBody("STUDENT", {})) });
  });
  await page.goto("/me");
  await expect(page.locator("[data-part=bottom-nav] a, [data-part=sidebar] a").first()).toBeVisible({ timeout: 30_000 });
  await page.waitForTimeout(1500);
  expect(calls).toBe(1);
  expect(urls).toEqual([`${API_URL}/api/v1/auth/refresh`]); // lời gọi sớm đi tới API_BASE (NEXT_PUBLIC_API_URL), không phải đường dẫn tương đối của Next
  expect(await page.evaluate(() => "__epRefresh" in window)).toBe(false);
});

test("early refresh: `/` chưa từng đăng nhập không gọi làm mới (như trước)", async ({ page }) => {
  let calls = 0;
  await page.route("**/api/v1/auth/refresh", (route) => {
    calls++;
    return route.fulfill({ status: 401, contentType: "application/json", headers: cors, body: JSON.stringify({ code: "UNAUTHENTICATED", message: "x", trace_id: "a".repeat(32) }) });
  });
  await page.goto("/");
  await page.waitForTimeout(1500);
  expect(calls).toBe(0);
});

test("early refresh: hai tab tải cùng lúc không bao giờ có hai làm mới chạy chồng lên nhau (khoá ep-refresh)", async ({ context }) => {
  await asDemo(context, "student");
  let active = 0;
  let maxActive = 0;
  let calls = 0;
  await context.route("**/api/v1/auth/refresh", async (route) => {
    calls++;
    active++;
    maxActive = Math.max(maxActive, active);
    await new Promise((r) => setTimeout(r, 400));
    active--;
    await route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(sessionBody("STUDENT", {})) });
  });
  const a = await context.newPage();
  const b = await context.newPage();
  await Promise.all([a.goto("/me"), b.goto("/me")]);
  for (const p of [a, b]) await expect(p.locator("[data-part=bottom-nav] a, [data-part=sidebar] a").first()).toBeVisible({ timeout: 30_000 });
  expect(maxActive).toBe(1);
  expect(calls).toBe(2); // mỗi tab một lần
});
