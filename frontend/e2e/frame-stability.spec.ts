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

