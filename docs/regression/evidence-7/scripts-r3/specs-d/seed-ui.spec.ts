import { test, type BrowserContext } from "@playwright/test";
import { log, shot, expect } from "../lib";
// Giao diện thật của ep-r (https://localhost:1143) nhưng mọi /api/v1/** được chuyển tiếp sang gateway phụ của stack seed (http://127.0.0.1:18095, DB qc3seed).
const UP = "http://127.0.0.1:18095";
const L = (tc: string, ok: boolean, note: string) => log("p7seed", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
async function proxy(ctx: BrowserContext) {
  await ctx.route(/\/api\/v1\//, async (route) => {
    const u = new URL(route.request().url());
    if (u.pathname.endsWith("/events")) return route.fulfill({ status: 204 });
    try { const r = await route.fetch({ url: UP + u.pathname + u.search, maxRedirects: 0 }); await route.fulfill({ response: r }); } catch { await route.abort(); }
  });
}
async function open(browser: any, user: string) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: "https://localhost:1143", viewport: { width: 1280, height: 800 }, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  await proxy(ctx); const page = await ctx.newPage(); await page.goto("/login"); await page.waitForTimeout(1500);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill("Edupilot#Seed-2026"); await page.keyboard.press("Enter");
  await page.waitForURL((x) => !x.pathname.startsWith("/login"), { timeout: 40_000 }); await page.locator("[data-part=topbar]").waitFor({ timeout: 30_000 }); await page.waitForTimeout(2500);
  return { ctx, page };
}
test("SEED-11 Hôm nay (giảng viên)", async ({ browser }) => {
  const { ctx, page } = await open(browser, "teacher"); await page.goto("/"); await page.waitForTimeout(3000);
  const top0 = (await page.locator("[data-part=topbar]").innerText()).replace(/\n+/g, " | ").slice(0, 80);
  if (!/761987/.test(top0)) { await page.locator("[data-part=topbar]").getByRole("button", { name: /7619/ }).first().click(); await page.waitForTimeout(800); await page.getByRole("option", { name: /761987/ }).first().click().catch(async () => { await page.getByText("761987", { exact: false }).last().click(); }); await page.waitForTimeout(4000); }
  const top1 = (await page.locator("[data-part=topbar]").innerText()).replace(/\n+/g, " | ").slice(0, 80);
  const t = (await page.locator("main").innerText()).replace(/\n+/g, " | ");
  const item = /1 sinh viên mới vào diện cần chú ý · 761987/.test(t); const name = /Ngô Ngọc Cẩm/.test(t);
  await shot(page, "p7seed-11"); L("TC-S7-SEED-11-ui", item && name, `/ của teacher@: mục «1 sinh viên mới vào diện cần chú ý · 761987»=${item}; khối chú ý có «Ngô Ngọc Cẩm» (sv.nguyco, Sinh viên C)=${name}; lớp trên thanh trên: «${top0}» → «${top1}»; main: ${t.slice(0, 1800)}`);
  await ctx.close();
});
test("SEED-25 /settings của sv.gioi", async ({ browser }) => {
  const { ctx, page } = await open(browser, "sv.gioi");
  const dlg = page.getByRole("dialog"); if (await dlg.count()) { await dlg.getByRole("button", { name: "Đã hiểu" }).click(); await page.waitForTimeout(1500); }
  await page.goto("/settings"); await page.waitForTimeout(4000);
  const sws = await page.getByRole("switch", { name: /·/ }).evaluateAll((els) => els.map((e) => ({ name: ((e.closest("label,li,div[class]") as HTMLElement | null)?.innerText ?? "").replace(/\n+/g, " | ").slice(0, 120), disabled: (e as HTMLInputElement).disabled || e.getAttribute("aria-disabled") === "true", checked: (e as HTMLInputElement).checked || e.getAttribute("aria-checked") === "true", tip: (() => { const id = e.getAttribute("aria-describedby"); return id ? document.getElementById(id)?.innerText ?? "" : (e.getAttribute("title") ?? ""); })() })));
  await shot(page, "p7seed-25");
  const l1 = sws.find((s) => /761987/.test(s.name)), l2 = sws.find((s) => /761988/.test(s.name));
  L("TC-S7-SEED-25-ui", sws.length === 2 && !!l1 && !l1.disabled && l1.checked && !!l2 && l2.disabled && l2.checked, `/settings của sv.gioi: ${sws.length} dòng công tắc: ${JSON.stringify(sws)}`);
  await ctx.close();
});
