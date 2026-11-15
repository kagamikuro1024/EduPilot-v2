import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { gradebookApi, type Opts } from "./support/gradebook-fixtures";
import { REPORT } from "./support/insights-fixtures";
import { asDemo, asJwt } from "./support/session";

// P-11-51 (qc2 audit-sau4 / sau5): khung trang giữ nguyên số đo và bộ nút ở mọi trạng thái (TL 14.2, 14.5, 14.6, 14.8).
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const headerButtons = (page: Page) =>
  page.locator("[data-ep-page-header] [data-ep-header-actions] :is(button, a)").evaluateAll((els) =>
    els.filter((e) => (e as HTMLElement).offsetParent !== null).map((e) => (e.textContent ?? "").trim() || e.getAttribute("aria-label") || ""),
  );

for (const [name, opts] of [["đã chốt", { locked: true, drift: true }], ["chưa chốt", {}]] as const) {
  test(`gradebook ${name}: dòng meta cao đúng một dòng (H7)`, async ({ page, context }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await asDemo(context, "teacher");
    await gradebookApi(page, { students: 30, ...opts });
    await page.goto("/gradebook");
    await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
    const m = await page.locator("[data-ep-page-header]").evaluate((el) => {
      const meta = el.querySelector<HTMLElement>("[class*=meta]");
      return meta ? { h: meta.getBoundingClientRect().height, line: parseFloat(getComputedStyle(meta).lineHeight) } : null;
    });
    if (opts.locked) expect(m).not.toBeNull();
    if (m) expect(Math.abs(m.h - m.line)).toBeLessThan(0.6);
  });

  test(`gradebook ${name}: mọi hàng bảng cùng một chiều cao (D3)`, async ({ page, context }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await asDemo(context, "teacher");
    await gradebookApi(page, { students: 30, ...opts });
    await page.goto("/gradebook");
    await expect(page.locator("[data-part=grade-row]").first()).toBeVisible();
    const rows = await page.locator("[data-part=grade-row]").evaluateAll((els) => [...new Set(els.map((e) => Math.round(e.getBoundingClientRect().height)))]);
    expect(rows).toHaveLength(1);
  });
}

for (const [name, opts] of [["chưa chốt", {}], ["chưa có công thức", { scheme: false }], ["đã chốt", { locked: true }]] as const satisfies ReadonlyArray<readonly [string, Opts]>) {
  test(`gradebook ${name}: đang tải và lỗi giữ đúng bộ nút của lúc có dữ liệu (X3)`, async ({ page, context }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await asDemo(context, "teacher");
    let slow = true;
    let broken = false;
    await gradebookApi(page, {
      students: 5,
      ...opts,
      h: { "GET gradebook": async (r, next) => { if (broken) return r.fulfill(json({ code: "INTERNAL", message: "x", trace_id: "t" }, 500)); if (slow) await new Promise((ok) => setTimeout(ok, 2500)); return next(r); } },
    });
    const statusIn = page.waitForResponse((r) => r.url().includes("grade-scheme/status"));
    await page.goto("/gradebook");
    await statusIn;
    await expect(page.locator("[aria-busy=true]").first()).toBeVisible();
    await expect.poll(() => headerButtons(page).then((l) => l.length)).toBeGreaterThan(0);
    const loading = await headerButtons(page);
    await expect(page.locator("[aria-busy=true]")).toHaveCount(0, { timeout: 10_000 });
    slow = false;
    const data = await headerButtons(page);
    expect(loading).toEqual(data);
    broken = true;
    const statusAgain = page.waitForResponse((r) => r.url().includes("grade-scheme/status"));
    await page.reload();
    await statusAgain;
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect.poll(() => headerButtons(page)).toEqual(data);
  });
}

test("insights: đang tải và lỗi giữ đúng bộ nút đầu trang của lúc có báo cáo (X3)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  const KC = { id: "22222222-2222-7222-8222-222222222987", class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
  let mode: "slow" | "ok" | "fail" = "slow";
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: [{ course: KC, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null })));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill(json({ count: 0, actions: [] })));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill(json({ items: [], next_cursor: null, unread_count: 0 })));
  await page.route("**/api/v1/courses/*/insights/knowledge-gap/latest", async (r) => {
    if (mode === "fail") return r.fulfill(json({ code: "INTERNAL", message: "x", trace_id: "t" }, 500));
    if (mode === "slow") await new Promise((ok) => setTimeout(ok, 2500));
    return r.fulfill(json(REPORT));
  });
  await page.goto("/insights");
  await expect(page.locator("[aria-busy=true]").first()).toBeVisible();
  const loading = await headerButtons(page);
  await expect(page.getByText("Chủ đề nên dạy lại")).toBeVisible();
  const data = await headerButtons(page);
  expect(data.length).toBeGreaterThan(0);
  expect(loading).toEqual(data);
  mode = "fail";
  await page.reload();
  await expect(page.getByRole("alert").first()).toBeVisible();
  expect(await headerButtons(page)).toEqual(data);
});

test("observability (giảng viên) lớp chưa có yêu cầu: khối rỗng nằm trong một Panel cao ≤ 320 px (E6)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  const KC = { id: "22222222-2222-7222-8222-222222222987", class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: [{ course: KC, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null })));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill(json({ count: 0, actions: [] })));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill(json({ items: [], next_cursor: null, unread_count: 0 })));
  await page.route("**/api/v1/courses/*/observability/summary", (r) =>
    r.fulfill(json({ requests: 0, error_rate: 0, escalations: 0, pii_masked: 0, helpful: 0, not_helpful: 0, thread_correct: 0, thread_reject: 0, graded_total: 0, graded_changed: 0, change_rate: 0, mean_abs_deviation: 0 })),
  );
  await page.goto("/observability");
  const empty = page.locator("main [data-ep-empty]");
  await expect(empty).toBeVisible();
  const box = await empty.evaluate((e) => {
    const p = e.closest("[data-ep-panel]");
    return p ? p.getBoundingClientRect().height : null;
  });
  expect(box).not.toBeNull();
  expect(box!).toBeLessThanOrEqual(320);
});
