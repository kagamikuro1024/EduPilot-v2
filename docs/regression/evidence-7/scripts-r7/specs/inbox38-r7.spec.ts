import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, newUser } from "./lib";
// TC-P4-03-38 chạy lại sau debd2d05 (DEF-R7-11, P-11-81): 1440 và 375
const F = FX("fx_ui8") as { C: string; T: string }; const FA = FX("fx_att") as { C: string; teacher: string };
const R = (tc: string, ok: boolean, note: string) => log("r7-i38", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const main = async (p: Page) => (await p.locator("main").innerText()).replace(/\n+/g, " | ");
const cls = async (p: Page) => ((await p.locator("[data-part=topbar]").innerText()).match(/QC3[A-Z0-9]+/) ?? [""])[0];
for (const [vp, tag] of [[{ width: 1440, height: 900 }, "1440"], [{ width: 375, height: 812 }, "375"]] as const) test(`38 ${tag}`, async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.T, vp);
  await page.addInitScript(() => { (window as any).__cls = 0; try { new PerformanceObserver((l) => { for (const e of l.getEntries() as any[]) if (!e.hadRecentInput) (window as any).__cls += e.value; }).observe({ type: "layout-shift", buffered: true }); } catch { /* */ } });
  // tải chậm
  let slow = true; await page.route(/\/tickets\?/, async (r) => { if (slow) await new Promise((x) => setTimeout(x, 3500)); await r.continue().catch(() => undefined); });
  await page.goto(`/inbox?course=${F.C}`); await page.waitForTimeout(1200);
  const skel = await page.locator("[class*=skeleton],[data-skeleton],[aria-busy=true]").count(); slow = false; await page.waitForTimeout(4500);
  const c = await cls(page);
  // bộ lọc: rỗng khi lớp còn phiếu (a: không chọn phiếu; b: sau khi chọn phiếu ở bộ lọc khác)
  const names = await page.getByRole("radio").allInnerTexts(); const seen: Record<string, string> = {}; const c0 = await page.evaluate(() => (window as any).__cls as number);
  for (const n of names) { await page.getByRole("radio", { name: n }).click(); await page.waitForTimeout(1500); seen[n] = await main(page); }
  const c1 = await page.evaluate(() => (window as any).__cls as number);
  await shot(page, `r7-i38-${tag}-filter-empty`);
  const empty1 = seen["Đã nhận"] ?? "";
  await page.getByRole("radio", { name: "Đã trả lời" }).click(); await page.waitForTimeout(1200); await page.locator("[data-part=inbox-list] [data-ticket-id]").first().click(); await page.waitForTimeout(1000);
  const sel = page.url().includes("ticket="); if (!(await page.getByRole("radio", { name: "Đã nhận" }).isVisible())) { await page.getByRole("button", { name: "Hộp thư" }).first().click(); await page.waitForTimeout(800); } await page.getByRole("radio", { name: "Đã nhận" }).click(); await page.waitForTimeout(1500); const empty2 = await main(page); await shot(page, `r7-i38-${tag}-filter-empty-selected`);
  // lỗi 500
  await page.unrouteAll({ behavior: "ignoreErrors" }); await page.route(/\/tickets\?/, (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }));
  await page.goto(`/inbox?course=${F.C}`); await page.waitForTimeout(3500); const err = await main(page); const retry = await page.getByRole("button", { name: "Thử lại" }).count(); await page.unrouteAll({ behavior: "ignoreErrors" });
  // lớp không có phiếu
  const e = await newUser(browser, FA.teacher, vp); await go(e.page, `/inbox?course=${FA.C}`); await e.page.waitForTimeout(2500);
  const none = await main(e.page); const seeAll = (await e.page.getByRole("button", { name: "Xem tất cả" }).count()) + (await e.page.getByRole("link", { name: "Xem tất cả" }).count()); const noneBtns = await e.page.locator("main [data-part=empty-state] button, main [role=status] button").count(); const nc = await cls(e.page); await shot(e.page, `r7-i38-${tag}-class-empty`);
  const sw = await e.page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]); const sw2 = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
  const f1 = /Không có câu hỏi trong bộ lọc này/.test(empty1) && !/Chưa có câu hỏi cần xử lý/.test(empty1);
  const f2 = /Không có câu hỏi trong bộ lọc này/.test(empty2) && !/Chưa có câu hỏi cần xử lý/.test(empty2);
  const k = /Chưa có câu hỏi cần xử lý/.test(none) && !/Chưa có câu hỏi cần xử lý\./.test(none) && seeAll === 0 && !/Không có câu hỏi trong bộ lọc/.test(none);
  const rowsOk = Object.entries(seen).filter(([n]) => n !== "Đã nhận").every(([, t]) => /\S/.test(t));
  R(`TC-P4-03-38(${tag})`, c === "QC3T1" && skel >= 1 && f1 && f2 && k && /Thử lại/.test(err) && retry >= 1 && c1 - c0 <= 0.05 && rowsOk && sw[0] <= sw[1] && sw2[0] <= sw2[1], `[${tag}px, P-11-81] GV ${F.T} lớp ${c}: tải chậm 3,5 s → khung xương ${skel}; bộ lọc ${JSON.stringify(names)}; «Đã nhận» rỗng (lớp còn phiếu): «${empty1.slice(-90)}» (đúng=${f1}); chọn một phiếu rồi (url có ticket=${sel}) đổi sang «Đã nhận»: «${empty2.slice(-90)}» (đúng=${f2}); CLS khi đổi bộ lọc ${(c1 - c0).toFixed(4)}; 500: «${err.slice(-110)}» «Thử lại»=${retry}; lớp ${nc} không có phiếu: «${none.slice(-70)}», «Xem tất cả»=${seeAll}, nút trong trạng thái rỗng=${noneBtns}; scrollWidth ${sw}/${sw2}`);
  await e.ctx.close(); await ctx.close();
});
