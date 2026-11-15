import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asJwt } from "./support/session";

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (status: number, body: unknown) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const row = (i: number, over: object = {}) => ({
  id: `00000000-0000-7000-8000-${String(i).padStart(12, "0")}`, at: `2026-11-14T0${i % 10}:00:00Z`, actor_id: "00000000-0000-7000-8000-0000000000aa", actor_name: "Quản Trị Viên", actor_role: "ADMIN",
  entity: "llm_request", entity_id: "e1", action: "VIEW_DETAIL", course_id: null, trace_id: null, changed_fields: ["reason"], ...over,
});

async function open(page: Page, handler: Parameters<Page["route"]>[1]) {
  await asJwt(page, "ADMIN");
  await page.route("**/api/v1/admin/audit**", handler);
  await page.goto("/admin/audit");
}

test("bảng nhật ký: cột, người / hệ thống, khoá đổi, mục «Nhật ký» ở menu, không dòng giải thích (AC10)", async ({ page }, info) => {
  await open(page, (r) => r.fulfill(json(200, { items: [row(1), row(2, { actor_id: null, actor_name: null, actor_role: null, entity: "mail_outbox", action: "DEAD", changed_fields: [] })], next_cursor: null })));
  await expect(page.getByRole("heading", { name: "Nhật ký", level: 1 })).toBeVisible();
  const table = page.getByRole("table", { name: /Nhật ký kiểm toán/ });
  for (const h of ["Thời điểm", "Người", "Thực thể", "Hành động", "Khoá đổi"]) await expect(table.getByRole("columnheader", { name: h })).toBeVisible();
  await expect(table.getByText("Quản Trị Viên · Quản trị")).toBeVisible();
  await expect(table.getByText("Hệ thống")).toBeVisible();
  await expect(table.getByText("reason")).toBeVisible();
  if (info.project.name === "desktop") await expect(page.locator("[data-part=sidebar]").getByRole("link", { name: "Nhật ký" })).toBeVisible();
});

test("lọc gửi đúng tham số; entity sai mẫu báo tại ô và không gọi API (AC1, AC10)", async ({ page }) => {
  const urls: string[] = [];
  await open(page, (r) => {
    urls.push(r.request().url());
    return r.fulfill(json(200, { items: [row(1)], next_cursor: null }));
  });
  await expect(page.getByRole("table")).toBeVisible();
  await page.getByLabel("Thực thể").fill("llm_request");
  await expect.poll(() => urls.some((u) => u.includes("entity=llm_request"))).toBe(true);
  const n = urls.length;
  await page.getByLabel("Thực thể").fill("Bad-Entity");
  await expect(page.getByText("Chữ thường, số, gạch dưới")).toBeVisible();
  await page.waitForTimeout(700);
  expect(urls.length).toBe(n);
  await page.getByLabel("Thực thể").fill("");
  // lọc theo người: bấm tên ở một dòng (không bắt gõ mã); chip «Người: …» + nút bỏ lọc (TC-14-82)
  await expect(page.getByLabel("Người", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: /^Lọc theo Quản Trị Viên/ }).first().click();
  await expect.poll(() => urls.some((u) => u.includes("actor_id=00000000-0000-7000-8000-0000000000aa"))).toBe(true);
  await expect(page.getByRole("button", { name: "Bỏ lọc người" })).toContainText("Quản Trị Viên");
  await page.getByRole("button", { name: "Bỏ lọc người" }).click();
  await expect(page.getByRole("button", { name: "Bỏ lọc người" })).toHaveCount(0);
  expect(urls.every((u) => /[?&]from=/.test(u))).toBe(true);
});

test("Tải thêm theo con trỏ; rỗng / lỗi có chữ riêng (AC10)", async ({ page }) => {
  let calls = 0;
  await open(page, (r) => {
    calls += 1;
    const cursor = new URL(r.request().url()).searchParams.get("cursor");
    return r.fulfill(json(200, cursor ? { items: [row(3, { action: "SECOND_PAGE" })], next_cursor: null } : { items: [row(1), row(2)], next_cursor: "c1" }));
  });
  await page.getByRole("button", { name: /Tải thêm|Xem thêm/ }).click();
  await expect(page.getByText("SECOND_PAGE")).toBeVisible();
  expect(calls).toBe(2);
});

test("khoảng không có dòng → EmptyState một dòng", async ({ page }) => {
  await open(page, (r) => r.fulfill(json(200, { items: [], next_cursor: null })));
  await expect(page.getByText("Không có dòng nào trong khoảng này")).toBeVisible();
});

test("API lỗi → một thông báo + Thử lại", async ({ page }) => {
  await open(page, (r) => r.fulfill(json(500, { code: "INTERNAL", message: "Lỗi", trace_id: "t" })));
  await expect(page.getByRole("alert")).toHaveCount(1);
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
});

test("TEACHER mở thẳng URL → trang chặn chung, không gọi API (AC10)", async ({ page }) => {
  let asked = false;
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/admin/audit**", (r) => {
    asked = true;
    return r.fulfill(json(403, { code: "FORBIDDEN", message: "x", trace_id: "t" }));
  });
  await page.goto("/admin/audit");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
  expect(asked).toBe(false);
});

test("375 px: không tràn ngang (AC10)", async ({ page }, info) => {
  test.skip(info.project.name !== "mobile", "chỉ bề rộng di động");
  await open(page, (r) => r.fulfill(json(200, { items: [row(1), row(2)], next_cursor: null })));
  await expect(page.getByRole("table")).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
