import { test } from "@playwright/test";
import { go, FX, shot, log, newUser } from "./lib";
test("explore threads", async ({ browser }) => {
  const f = FX("fx_ui7");
  for (const [u, path] of [["qc3s24", `/threads/${f.th}?course=${f.A}`], ["qc3s26", `/threads/${f.th2}?course=${f.A}`], ["qc3t", `/threads/${f.th2}?course=${f.A}`], ["qc3s26", `/threads?course=${f.A}`]] as const) {
    const { ctx, page } = await newUser(browser, u); await go(page, path); await page.waitForTimeout(1500);
    const txt = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 1200);
    const menus = await page.locator('button[aria-label="Tuỳ chọn bài"]').count();
    log("explore", `== ${u} ${path.slice(0, 60)}\n${txt}\nMENUS=${menus}`);
    await shot(page, `explore-thr-${u}`); await ctx.close();
  }
});
