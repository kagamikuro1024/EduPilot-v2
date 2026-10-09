import AxeBuilder from "@axe-core/playwright";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { expect, test, type Page } from "@playwright/test";
import { PU_ROUTES, ROLES, routesFor } from "./support/routes";
import { settleGoto } from "./support/hydrate";
import { asDemo, type DemoRole } from "./support/session";
import { mockStudentExams, mockStudentToday } from "./support/screens";
import { mockTake, TAKE_COURSE, TAKE_EXAM } from "./support/takeMock";
import { BASE_URL } from "./support/env";
import { mockAdminApi, mockLlmApi, mockStaffApi, STAFF_DRAFT, STAFF_EXAM } from "./support/staffMock";

// US-PU-05 AC4: axe (WCAG 2.2 AA) trên mọi route × vai được phép mở, ở 1440 và 390; chặn `critical` và `serious`,
// `moderate` / `minor` ghi vào báo cáo (test-results/axe-report.json). Ngoại lệ ở axe-allow.json (≤ 3).
type Allow = { rule: string; route: string; reason: string; approvedBy: string };
const allow: Allow[] = JSON.parse(readFileSync("e2e/axe-allow.json", "utf8"));
const TAGS = ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"];
const WIDTHS = [1440, 390];
const report: Array<{ role: string; route: string; width: number; rule: string; impact: string; nodes: number; allowed: boolean }> = [];
let scanned = 0;
const incompleteContrast: Array<{ route: string; width: number; nodes: string[] }> = []; // US-UI-07 AC3: mục `incomplete` của color-contrast để QC xem tay

test.describe.configure({ mode: "serial" }); // một worker ⇒ một báo cáo tổng
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});
test.beforeEach(({ page }) => {
  settleGoto(page);
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
  for (const v of res.incomplete) if (v.id === "color-contrast") incompleteContrast.push({ route, width, nodes: v.nodes.slice(0, 5).map((n) => n.target.join(" ")) });
  for (const v of res.violations) {
    const allowed = allow.some((a) => a.rule === v.id && a.route === route);
    report.push({ role, route, width, rule: v.id, impact: v.impact ?? "minor", nodes: v.nodes.length, allowed });
    if (v.id === "color-contrast" && !allowed) bad.push(`color-contrast phải 0 vi phạm: ×${v.nodes.length} ${v.nodes.slice(0, 2).map((n) => n.target.join(" ")).join(" | ")}`);
    else if ((v.impact === "critical" || v.impact === "serious") && !allowed) bad.push(`${v.id} (${v.impact}) ×${v.nodes.length}: ${v.nodes.slice(0, 2).map((n) => n.target.join(" ")).join(" | ")}`);
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

// US-UI-05 / US-UI-06: các màn dùng API thật ở trạng thái CÓ DỮ LIỆU (ca ở trên chạy khi gateway không có nên chỉ thấy trạng thái lỗi).
test("axe: admin có dữ liệu (/admin/users, /admin/courses, /settings/llm)", async ({ page, context }) => {
  test.setTimeout(120_000);
  await asDemo(context, "admin");
  await mockAdminApi(page);
  await mockLlmApi(page);
  const bad: string[] = [];
  for (const route of ["/admin/users", "/admin/courses", "/settings/llm"]) for (const w of WIDTHS) for (const b of await scan(page, "admin+data", route, w)) bad.push(`${route} @${w}: ${b}`);
  expect(bad).toEqual([]);
});

test("axe: giảng viên có dữ liệu (bài thi, soạn, kết quả, giống nhau, câu hỏi, thành viên, cài đặt lớp)", async ({ page, context }) => {
  test.setTimeout(180_000);
  await asDemo(context, "teacher");
  await mockStaffApi(page);
  const bad: string[] = [];
  const routes = ["/exams", `/exams/${STAFF_DRAFT}`, `/exams/${STAFF_EXAM}/results`, `/exams/${STAFF_EXAM}/similarity`, "/questions", "/class/members", "/class/settings"];
  for (const route of routes) for (const w of WIDTHS) for (const b of await scan(page, "teacher+data", route, w)) bad.push(`${route} @${w}: ${b}`);
  expect(bad).toEqual([]);
});

test("axe: sinh viên có dữ liệu (Hôm nay, bài thi, làm bài trước giờ / đang làm / đã công bố)", async ({ page, context }) => {
  test.setTimeout(180_000);
  await asDemo(context, "student");
  await mockStudentToday(page);
  await mockStudentExams(page);
  const bad: string[] = [];
  for (const route of ["/", "/exams"]) for (const w of WIDTHS) for (const b of await scan(page, "student+data", route, w)) bad.push(`${route} @${w}: ${b}`);
  for (const kind of ["intro", "running", "published"] as const) {
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await mockTake(page, kind);
    for (const w of WIDTHS) for (const b of await scan(page, `student+take-${kind}`, `/exams/${TAKE_EXAM}/take?course=${TAKE_COURSE}`, w)) bad.push(`take-${kind} @${w}: ${b}`);
  }
  expect(bad).toEqual([]);
});

// QC US-UI-07 B1: bảng ảo hoá có nhiều dòng hơn cửa sổ — aria-activedescendant phải ở phần tử có role hợp lệ và chỉ trỏ tới dòng đang render.
test("axe: bảng ảo hoá > cửa sổ (kết quả bài thi 1.000 dòng): không critical, activedescendant hợp lệ khi dòng ở trong / ngoài cửa sổ", async ({ page, context }) => {
  test.setTimeout(120_000);
  await asDemo(context, "teacher");
  await mockStaffApi(page);
  const many = Array.from({ length: 1000 }, (_, i) => ({ attempt_id: `att-${i}`, student: { id: `s-${i}`, full_name: `Sinh viên ${String(i).padStart(4, "0")}`, student_code: `B20DC${String(i).padStart(4, "0")}` }, status: "GRADED", auto_score: "7.50", score: "7.50", adjusted: false, submitted_at: "2026-12-01T01:00:00Z", submit_reason: "MANUAL", flags: { similarity: 0, tab_hidden: 0, paste: 0 } }));
  const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
  await page.route(/\/api\/v1\/courses\/[^/]+\/exams\/[^/]+\/results(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ progress: { not_started: 0, in_progress: 0, grading: 0, graded: 1000, absent: 0 }, items: many, next_cursor: null }) }));
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(`/exams/${STAFF_EXAM}/results`);
  const table = page.locator("[data-part=virtual-scroll] table");
  await table.waitFor();
  expect(await table.locator("tbody tr[data-index]").count(), "chỉ dựng dòng trong cửa sổ ảo").toBeLessThan(120);
  const problems = async (label: string) => {
    const res = await new AxeBuilder({ page }).withTags(TAGS).analyze();
    const dangling = await page.evaluate(() => [...document.querySelectorAll("[aria-activedescendant]")].filter((e) => !document.getElementById(e.getAttribute("aria-activedescendant")!)).length);
    return [...res.violations.filter((v) => v.impact === "critical" || v.impact === "serious").map((v) => `${label}: ${v.id} (${v.impact})`), ...(dangling ? [`${label}: aria-activedescendant trỏ tới id không tồn tại ×${dangling}`] : [])];
  };
  const bad = await problems("ban đầu");
  await table.focus();
  for (let i = 0; i < 3; i++) await page.keyboard.press("ArrowDown");
  await expect(table.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "2");
  expect(await table.getAttribute("aria-activedescendant"), "dòng đang chọn đang render ⇒ có activedescendant").toMatch(/^vt-/);
  bad.push(...(await problems("dòng chọn trong cửa sổ")));
  await page.locator("[data-part=virtual-scroll]").evaluate((e) => { e.scrollTop = e.scrollHeight; }); // dòng chọn bị đẩy ra ngoài cửa sổ ảo
  await expect(table.locator('tbody tr[data-index="999"]')).toBeVisible();
  expect(await table.locator('tr[aria-current="true"]').count()).toBe(0);
  expect(await table.getAttribute("aria-activedescendant"), "dòng chọn ngoài cửa sổ ⇒ không đặt activedescendant").toBeNull();
  bad.push(...(await problems("dòng chọn ngoài cửa sổ")));
  await page.keyboard.press("End");
  await expect(table.locator('tr[aria-current="true"]')).toHaveAttribute("data-index", "999");
  bad.push(...(await problems("sau End")));
  expect(bad).toEqual([]);
});

test("axe: route PU không cần vai (/login, /dev/ui, /dev/data)", async ({ page }) => {
  test.setTimeout(120_000);
  const bad: string[] = [];
  for (const route of PU_ROUTES) for (const w of WIDTHS) for (const b of await scan(page, "-", route, w)) bad.push(`${route} @${w}: ${b}`);
  expect(bad).toEqual([]);
});

test.afterAll(() => {
  mkdirSync("test-results", { recursive: true });
  writeFileSync("test-results/axe-report.json", JSON.stringify({ scanned, routes: ROLES.reduce((n, r) => n + routesFor(r).length, 0) + PU_ROUTES.length, violations: report }, null, 1));
  writeFileSync("test-results/axe-incomplete-contrast.json", JSON.stringify(incompleteContrast, null, 1));
  console.log(`AXE incomplete color-contrast: ${incompleteContrast.length} lượt có mục cần xem tay (chi tiết test-results/axe-incomplete-contrast.json)`);
  const by = (imp: string) => report.filter((r) => r.impact === imp).length;
  console.log(`AXE: ${scanned} lượt quét (route × vai × bề rộng); critical ${by("critical")}, serious ${by("serious")}, moderate ${by("moderate")}, minor ${by("minor")}`);
});
