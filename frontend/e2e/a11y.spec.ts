import AxeBuilder from "@axe-core/playwright";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { expect, test, type Page } from "@playwright/test";
import { PU_ROUTES, ROLES, routesFor } from "./support/routes";
import { asDemo, type DemoRole } from "./support/session";

// US-PU-05 AC4: axe (WCAG 2.2 AA) trên mọi route × vai được phép mở, ở 1440 và 390; chặn `critical` và `serious`,
// `moderate` / `minor` ghi vào báo cáo (test-results/axe-report.json). Ngoại lệ ở axe-allow.json (≤ 3).
type Allow = { rule: string; route: string; reason: string; approvedBy: string };
const allow: Allow[] = JSON.parse(readFileSync("e2e/axe-allow.json", "utf8"));
const TAGS = ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"];
const WIDTHS = [1440, 390];
const report: Array<{ role: string; route: string; width: number; rule: string; impact: string; nodes: number; allowed: boolean }> = [];
let scanned = 0;

test.describe.configure({ mode: "serial" }); // một worker ⇒ một báo cáo tổng
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});
test("axe-allow.json: tối đa 3 ngoại lệ, mỗi mục đủ luật / route / lý do / người duyệt", () => {
  expect(allow.length).toBeLessThanOrEqual(3);
  for (const a of allow) expect(a.rule && a.route && a.reason.trim() && a.approvedBy).toBeTruthy();
});

async function scan(page: Page, role: string, route: string, width: number) {
  await page.setViewportSize({ width, height: width > 700 ? 900 : 844 });
  await page.goto(route);
  await page.locator("main, body").first().waitFor();
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(250);
  const res = await new AxeBuilder({ page }).withTags(TAGS).analyze();
  scanned++;
  const bad: string[] = [];
  for (const v of res.violations) {
    const allowed = allow.some((a) => a.rule === v.id && a.route === route);
    report.push({ role, route, width, rule: v.id, impact: v.impact ?? "minor", nodes: v.nodes.length, allowed });
    if ((v.impact === "critical" || v.impact === "serious") && !allowed) bad.push(`${v.id} (${v.impact}) ×${v.nodes.length}: ${v.nodes.slice(0, 2).map((n) => n.target.join(" ")).join(" | ")}`);
  }
  return bad;
}

for (const role of ROLES) {
  test(`axe: ${role}`, async ({ page, context }) => {
    test.setTimeout(180_000);
    await asDemo(context, role as DemoRole);
    const bad: string[] = [];
    for (const route of routesFor(role)) for (const w of WIDTHS) for (const b of await scan(page, role, route, w)) bad.push(`${role} ${route} @${w}: ${b}`);
    expect(bad).toEqual([]);
  });
}

test("axe: route PU không cần vai (/login, /dev/ui, /dev/data)", async ({ page }) => {
  test.setTimeout(120_000);
  const bad: string[] = [];
  for (const route of PU_ROUTES) for (const w of WIDTHS) for (const b of await scan(page, "-", route, w)) bad.push(`${route} @${w}: ${b}`);
  expect(bad).toEqual([]);
});

test.afterAll(() => {
  mkdirSync("test-results", { recursive: true });
  writeFileSync("test-results/axe-report.json", JSON.stringify({ scanned, routes: ROLES.reduce((n, r) => n + routesFor(r).length, 0) + PU_ROUTES.length, violations: report }, null, 1));
  const by = (imp: string) => report.filter((r) => r.impact === imp).length;
  console.log(`AXE: ${scanned} lượt quét (route × vai × bề rộng); critical ${by("critical")}, serious ${by("serious")}, moderate ${by("moderate")}, minor ${by("minor")}`);
});
