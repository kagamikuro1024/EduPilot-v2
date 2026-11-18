import { test } from "@playwright/test";
import { go, FX, shot, log, psql, newUser } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p405", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
test("kb42", async ({ browser }) => {
  const f = FX("fx_kb"); const { ctx, page } = await newUser(browser, "qc3t");
  await go(page, `/inbox?course=${f.A}`); await page.getByRole("radio", { name: "Đã trả lời" }).click(); await page.waitForTimeout(1200);
  await page.locator(`[data-ticket-id="${f.tid}"]`).click(); await page.waitForTimeout(1500);
  const box = page.locator("[data-part=knowledge]"); const t1 = (await box.innerText()).replace(/\n+/g, " | ");
  const retry = await box.getByRole("button", { name: "Thử lại" }).count(); await shot(page, "p405-42-a");
  const nOut0 = psql(`select count(*) from outbox where topic='knowledge.sync' and created_at>now()-interval '10 minutes'`);
  if (retry) { await box.getByRole("button", { name: "Thử lại" }).click(); await page.waitForTimeout(2000); }
  const nOut1 = psql(`select count(*) from outbox where topic='knowledge.sync' and created_at>now()-interval '10 minutes'`);
  log("p405", `INFO kb42 phase A: «${t1}», nút Thử lại=${retry}, outbox knowledge.sync gần đây ${nOut0}→${nOut1}`);
  await ctx.close();
});
test("kb42b", async ({ browser }) => {
  const f = FX("fx_kb"); const { ctx, page } = await newUser(browser, "qc3t");
  await go(page, `/inbox?course=${f.A}`); await page.getByRole("radio", { name: "Đã trả lời" }).click(); await page.waitForTimeout(1200);
  await page.locator(`[data-ticket-id="${f.tid}"]`).click(); await page.waitForTimeout(1500);
  const t = (await page.locator("[data-part=knowledge]").innerText()).replace(/\n+/g, " | ");
  log("p405", `INFO kb42 phase B (worker đã lên lại): «${t}»`); await ctx.close();
});
