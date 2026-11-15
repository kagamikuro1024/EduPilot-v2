import AxeBuilder from "@axe-core/playwright";
import { expect, type Page, test } from "@playwright/test";
import { call, login, needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — US-P10-14 giao diện /admin/audit trên ep-qc thật (TC-14-69…82, 86, 91). Cần au_fixture.sql đã nạp (entity qc_audit).
needReal();

const table = (p: Page) => p.getByRole("table", { name: /Nhật ký kiểm toán/ });

test("@real AUD-1 cột, bộ lọc, Tải thêm, khớp API (TC-14-69/70/71)", async ({ page, request }) => {
  const api: string[] = [];
  page.on("response", (r) => { if (/\/api\/v1\/admin\/audit\?/.test(r.url()) && r.request().method() === "GET") api.push(`${r.status()} ${decodeURIComponent(r.url().split("/api/v1")[1])}`); });
  await uiLogin(page, "admin");
  await page.goto("/admin/audit");
  await expect(table(page)).toBeVisible();
  const heads = await table(page).getByRole("columnheader").allInnerTexts();
  expect(heads.map((h) => h.trim())).toEqual(["Thời điểm", "Người", "Thực thể", "Hành động", "Khoá đổi"]);
  for (const l of ["Thực thể", "Người"]) await expect(page.getByLabel(l, { exact: true })).toBeVisible();
  await expect(page.getByRole("radiogroup", { name: "Khoảng thời gian" }).or(page.getByText("Khoảng thời gian"))).toBeVisible();
  await shot(page, "audit-1440");
  await page.getByLabel("Thực thể", { exact: true }).fill("qc_audit");
  await expect(table(page).getByRole("row").nth(1)).toContainText("qc_audit");
  const tk = await login(request, "admin");
  const ref = await call(request, tk, "GET", "/admin/audit?entity=qc_audit&limit=50");
  const first = ((ref.json as { items: { action: string }[] }).items)[0];
  await expect(table(page).getByRole("row").nth(1)).toContainText(first.action);
  const more = page.getByRole("button", { name: "Tải thêm" });
  let clicks = 0, calls0 = api.length;
  while (await more.isVisible().catch(() => false) && clicks < 20) {
    const before = api.length;
    await more.click();
    await expect.poll(() => api.length, { timeout: 8000 }).toBeGreaterThan(before);
    expect(api.length - before, "mỗi lần bấm một lần gọi").toBe(1);
    clicks++;
    await page.waitForTimeout(300);
  }
  const rows = await table(page).getByRole("row").count();
  const total = Number(sql(`select count(*) from audit_log where entity='qc_audit' and created_at >= now() - interval '7 days'`)[0][0]);
  expect(rows - 1, `số dòng sau «Tải thêm» tới hết (${clicks} lần) = DB ${total}`).toBe(total);
  expect(await more.isVisible().catch(() => false), "hết trang: nút ẩn").toBe(false);
  const callsEnd = api.length;
  await page.waitForTimeout(1500);
  expect(api.length, "không còn gọi API khi đã hết").toBe(callsEnd);
  expect(calls0).toBeGreaterThan(0);
  await shot(page, "audit-all-loaded");
});

test("@real AUD-2 rỗng, lỗi + Thử lại, chữ giao diện, hàng ≤ 48 px (TC-14-72…76)", async ({ page }) => {
  await uiLogin(page, "admin");
  await page.goto("/admin/audit");
  await expect(table(page)).toBeVisible();
  await page.getByLabel("Thực thể", { exact: true }).fill("khong_co_dong_nao");
  await expect(page.getByText("Không có dòng nào trong khoảng này")).toBeVisible();
  await shot(page, "audit-empty");
  await page.getByLabel("Thực thể", { exact: true }).fill("qc_audit");
  await expect(table(page)).toBeVisible();
  // chữ: không giải thích dưới tiêu đề, không từ kỹ thuật
  const body = await page.locator("main").innerText();
  expect(body, "không từ kỹ thuật").not.toMatch(/\b(before|after|JSON|payload)\b/i);
  const h1 = page.getByRole("heading", { name: "Nhật ký", level: 1 });
  const next = await h1.evaluate((el) => (el.parentElement?.nextElementSibling as HTMLElement | null)?.tagName ?? "");
  expect(next, "ngay dưới tiêu đề không phải đoạn giải thích").not.toBe("P");
  // hàng nhiều khoá: f5-25 có 20 khoá
  await page.getByLabel("Thực thể", { exact: true }).fill("qc_audit");
  const heights = await table(page).getByRole("row").evaluateAll((rs) => rs.map((r) => Math.round(r.getBoundingClientRect().height)));
  expect(Math.max(...heights), `chiều cao hàng lớn nhất (${heights.slice(0, 8)})`).toBeLessThanOrEqual(48);
  // lỗi
  let fail = true;
  await page.route("**/api/v1/admin/audit**", (r) => (fail ? r.fulfill({ status: 500, json: { code: "INTERNAL", message: "x" } }) : r.continue()));
  await page.getByLabel("Thực thể", { exact: true }).fill("qc_audit2");
  await expect(page.getByRole("button", { name: /Thử lại/ })).toBeVisible();
  await shot(page, "audit-error");
  fail = false;
  await page.getByRole("button", { name: /Thử lại/ }).click();
  await expect(page.getByText("Không có dòng nào trong khoảng này").or(table(page))).toBeVisible();
});

test("@real AUD-3 bộ lọc «Người» nhập tên/email, không bắt dán uuid (TC-14-82)", async ({ page }) => {
  await uiLogin(page, "admin");
  await page.goto("/admin/audit");
  const actor = page.getByLabel("Người", { exact: true });
  await expect(actor).toBeVisible();
  const ph = (await actor.getAttribute("placeholder")) ?? "";
  expect(ph, "ô «Người» không đòi mã người dùng (uuid)").not.toMatch(/mã|uuid|id/i);
  await actor.fill("Quản trị");
  await expect(page.getByText("Mã người dùng không đúng"), "gõ tên không bị báo sai mã").toHaveCount(0);
});

test("@real AUD-4 phân quyền giao diện + chưa đăng nhập (TC-14-78/79)", async ({ page, browser }) => {
  await page.goto("/admin/audit");
  await expect(page).toHaveURL(/\/login\?next=/);
  await page.getByLabel(/Email/i).fill("admin@edupilot.local");
  await uiLogin(page, "admin");
  await page.goto("/admin/audit");
  await expect(table(page)).toBeVisible();
  for (const u of ["teacher", "ta", "sv.kha"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: process.env.QC_BASE ?? "https://localhost:943" });
    const p = await ctx.newPage();
    const ok200: string[] = [];
    p.on("response", (r) => { if (/\/api\/v1\/admin\/audit(\?|$)/.test(r.url()) && r.status() === 200) ok200.push(r.url()); });
    await uiLogin(p, u);
    await p.goto("/admin/audit");
    await p.waitForTimeout(1500);
    expect(ok200, `${u}: không có GET /admin/audit thành công`).toEqual([]);
    await expect(table(p)).toHaveCount(0);
    await expect(p.getByRole("link", { name: "Nhật ký" }), `${u}: menu không có mục Nhật ký`).toHaveCount(0);
    await shot(p, `audit-403-${u}`);
    await ctx.close();
  }
});

test("@real AUD-5 375 px, axe 1440 + 375, không before/after ở mạng (TC-14-77/81/91)", async ({ page }) => {
  const bodies: string[] = [];
  page.on("response", async (r) => { if (r.url().includes("/api/v1/admin/audit")) bodies.push(await r.text().catch(() => "")); });
  await uiLogin(page, "admin");
  await page.goto("/admin/audit");
  await expect(table(page)).toBeVisible();
  const ax1 = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag22aa"]).analyze();
  expect(ax1.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? "")).map((v) => `${v.id}:${v.nodes.length}`), "axe 1440").toEqual([]);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.reload();
  await expect(table(page)).toBeVisible();
  await noOverflow(page);
  const small = await page.locator("button, input, a[href]").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && !/Bỏ qua điều hướng/.test(e.textContent ?? "") && (r.height < 44) && getComputedStyle(e).visibility !== "hidden"; }).map((e) => `${e.tagName}:${(e.textContent ?? "").trim().slice(0, 20)}:${Math.round(e.getBoundingClientRect().height)}`));
  await shot(page, "audit-375");
  const ax2 = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag22aa"]).analyze();
  expect(ax2.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? "")).map((v) => `${v.id}:${v.nodes.length}`), "axe 375").toEqual([]);
  expect(small, "vùng chạm < 44 px ở 375").toEqual([]);
  expect(bodies.join(" "), "không khoá before/after ở mạng").not.toMatch(/"(before|after)"\s*:/);
});
