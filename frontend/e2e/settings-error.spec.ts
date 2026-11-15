import { expect, test } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asJwt } from "./support/session";

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const KC = { id: "11111111-1111-7111-8111-111111111106", class_code: "KC106", subject_code: "KC", name: "Kiểm chứng", semester: "2026-2027-HK1", status: "ACTIVE" };

// TL 14.7: một alert cho mỗi TRUY VẤN hỏng. Thư, Nhắc 24 giờ, Thời gian học cùng đọc /me/settings → đúng một alert.
test("settings (SV): /me/settings hỏng → đúng một thông báo lỗi, mật khẩu và thiết bị vẫn dùng được", async ({ page }) => {
  await asJwt(page, "STUDENT");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: [{ course: KC, role_in_course: "STUDENT", enrollment_status: "ACTIVE" }], next_cursor: null })));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill(json({ count: 0, actions: [] })));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill(json({ items: [], next_cursor: null, unread_count: 0 })));
  await page.route("**/api/v1/me/sessions", (r) => r.fulfill(json({ items: [] })));
  await page.route("**/api/v1/me/settings", (r) => r.fulfill(json({ code: "INTERNAL", message: "m", trace_id: "b".repeat(32) }, 500)));
  await page.goto("/settings");
  await expect(page.locator("main [role=alert]").first()).toBeVisible();
  await page.waitForTimeout(1500); // đủ cho mọi lần thử lại
  await expect(page.locator("main [role=alert]")).toHaveCount(1);
  await expect(page.getByRole("heading", { name: "Thư" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Mật khẩu" })).toBeVisible();
});
