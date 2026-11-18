import { test } from "@playwright/test";
import { login, go, FX, shot, log, psql, newUser } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p404", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const sw = (p: any, n: string) => p.getByRole("switch", { name: n });
test("26-27", async ({ browser }) => {
  const out: any = {};
  for (const [u, k] of [["qc3t", "GV"], ["qc3ta", "TA"], ["qc3s30", "SV"]] as const) {
    const { ctx, page } = await newUser(browser, u); await go(page, "/settings"); await page.waitForTimeout(1200);
    out[k] = { ticket: await sw(page, "Mail báo ticket mới").count(), answer: await sw(page, "Mail khi giảng viên trả lời").count(), tkOn: await sw(page, "Mail báo ticket mới").first().getAttribute("aria-checked").catch(() => null), anOn: await sw(page, "Mail khi giảng viên trả lời").first().getAttribute("aria-checked").catch(() => null), mailGroup: (await page.locator("main").innerText()).includes("Thư") };
    await shot(page, `p404-26-${k}`); await ctx.close();
  }
  R("TC-P4-04-26", out.GV.ticket === 1 && out.GV.tkOn === "false" && out.GV.answer === 0 && out.TA.ticket === 1 && out.TA.tkOn === "false" && out.TA.answer === 0 && out.SV.answer === 1 && out.SV.anOn === "true" && out.SV.ticket === 0, `/settings: GV ${JSON.stringify(out.GV)}; TA ${JSON.stringify(out.TA)}; SV ${JSON.stringify(out.SV)} (GV/TA: «Mail báo ticket mới» tắt mặc định, không có công tắc của SV; SV: «Mail khi giảng viên trả lời» bật mặc định, không có công tắc ticket)`);
  // 27: lưu lạc quan
  const { ctx, page } = await newUser(browser, "qc3t"); await go(page, "/settings"); await page.waitForTimeout(1200);
  const reqs: string[] = []; page.on("request", (r) => { if (r.method() === "PUT" && /\/me\/settings/.test(r.url())) reqs.push(r.postData() ?? ""); });
  const s = sw(page, "Mail báo ticket mới");
  const t0 = await page.evaluate(() => performance.now()); await s.click(); const flipped = await page.waitForFunction(() => (document.querySelector('[role=switch][aria-label="Mail báo ticket mới"], [role=switch]') as HTMLElement) !== null, null, { timeout: 2000 }).catch(() => null);
  const v1 = await s.getAttribute("aria-checked"); const t1 = await page.evaluate(() => performance.now());
  await page.waitForTimeout(1500);
  const toast = await page.locator("[role=status],[role=alert],[class*=oast]").evaluateAll((els) => els.map((e) => (e as HTMLElement).innerText.trim()).filter((x) => x && !/EduPilot/.test(x)));
  await s.click(); await page.waitForTimeout(1500); const v2 = await s.getAttribute("aria-checked");
  const hasVer = reqs.length >= 1 && reqs.every((b) => /"version":\d+/.test(b));
  R("TC-P4-04-27", v1 === "true" && v2 === "false" && hasVer && toast.length === 0, `bật «Mail báo ticket mới»: aria-checked đổi sau ${(t1 - t0).toFixed(0)} ms (lạc quan, trước khi PUT xong); tắt lại → ${v2}; ${reqs.length} PUT /me/settings, mọi thân có version: ${hasVer} («${reqs[0]}»); toast/thông báo: ${JSON.stringify(toast)}`);
  await ctx.close();
});
test("33", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "admin"); 
  await page.route("**/admin/observability/failed-mail**", (r) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], next_cursor: null }) }));
  await go(page, "/observability"); await page.waitForTimeout(2500);
  const txt = (await page.locator("main").innerText()).replace(/\n+/g, " | ");
  const idx = txt.indexOf("Không có thư lỗi"); await shot(page, "p404-33");
  const btns = await page.locator("main", { hasText: "Không có thư lỗi" }).locator("button").allInnerTexts();
  R("TC-P4-04-33", idx >= 0, `ADMIN /observability, GET failed-mail trả danh sách rỗng (page.route; stack có thư DEAD thật từ các ca khác nên không thể rỗng thật): «${txt.slice(Math.max(0, idx - 30), idx + 60)}»; nút trong khối=${JSON.stringify(btns)}`);
  await ctx.close();
});
