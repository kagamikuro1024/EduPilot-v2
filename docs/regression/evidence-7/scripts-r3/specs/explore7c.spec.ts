import { test } from "@playwright/test";
import { go, FX, shot, log, newUser } from "./lib";
const F = FX("fx_ui8") as { C: string; T: string; th: Record<string, string> };
const L = (s: string) => log("p7c-explore", s);
const top = (p: any) => p.locator("[data-part=topbar]").innerText().then((t: string) => t.replace(/\n+/g, " | ").slice(0, 120));
test("explore", async ({ browser }) => {
  for (const [u, th] of [["qc3s12", "rep"], ["qc3s15", "rep"], [F.T, "rep"], ["qc3s11", "del"]] as const) {
    const { ctx, page } = await newUser(browser, u); await go(page, `/threads/${F.th[th]}?course=${F.C}`); await page.waitForTimeout(1500);
    L(`== ${u} ${th} topbar: ${await top(page)}`);
    const btn = page.locator('button[aria-label="Tuỳ chọn bài"]'); L(`menu buttons: ${await btn.count()}`);
    await btn.first().click().catch(() => L("click menu failed")); await page.waitForTimeout(600);
    L(`menuitems: ${JSON.stringify(await page.getByRole("menuitem").allInnerTexts())}`);
    await shot(page, `p7c-menu-${u}`);
    await page.keyboard.press("Escape"); await ctx.close();
  }
  const { ctx, page } = await newUser(browser, "qc3s12"); await go(page, `/threads/${F.th.rep}?course=${F.C}`); await page.waitForTimeout(1500);
  await page.locator('button[aria-label="Tuỳ chọn bài"]').first().click(); await page.getByRole("menuitem", { name: /Báo cáo/ }).click(); await page.waitForTimeout(800);
  L(`dialog báo cáo: ${(await page.getByRole("dialog").innerText().catch(() => "—")).replace(/\n+/g, " | ")}`); await shot(page, "p7c-report-dialog");
  await ctx.close();
  const o = await newUser(browser, "qc3s11"); await go(o.page, `/threads/${F.th.del}?course=${F.C}`); await o.page.waitForTimeout(1500);
  await o.page.locator('button[aria-label="Tuỳ chọn bài"]').first().click(); await o.page.getByRole("menuitem", { name: /Xoá/ }).first().click(); await o.page.waitForTimeout(800);
  L(`dialog xoá: ${(await o.page.getByRole("dialog").innerText().catch(() => "—")).replace(/\n+/g, " | ")}`); await shot(o.page, "p7c-delete-dialog");
  await o.ctx.close();
  const c = await newUser(browser, "qc3s13"); await go(c.page, `/threads/new?course=${F.C}`); await c.page.waitForTimeout(1500);
  L(`compose: ${(await c.page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 600)}`); await shot(c.page, "p7c-compose");
  L(`fields: ${JSON.stringify(await c.page.locator("input,textarea").evaluateAll((e: any[]) => e.map((x) => `${x.tagName}:${x.name || x.getAttribute("aria-label") || x.placeholder}`)))}`);
  await c.ctx.close();
});
