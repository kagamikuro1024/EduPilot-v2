import { test, type Browser } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
const BASE = process.env.QC_BASE ?? "https://localhost:1143";
const ac = (n: string, ok: boolean, d = "") => console.log(`QCAC ${n} ${ok ? "PASS" : "FAIL"} ${d}`);
async function open(browser: Browser, w: number, h: number, extra: Record<string, unknown> = {}) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", ignoreHTTPSErrors: true, ...extra }); const page = await ctx.newPage(); return { ctx, page };
}
test("landing bổ sung: AC12, AC16, AC20, AC22", async ({ browser }) => {
  test.setTimeout(180_000);
  const { ctx, page } = await open(browser, 1440, 900); await page.goto(`${BASE}/`); await page.waitForTimeout(2500);
  // AC12 rê / bấm
  const btn = page.locator("[data-part=landing]").getByRole("link", { name: "Đăng nhập" }).first();
  const t0 = await btn.evaluate((e) => getComputedStyle(e).transform); await btn.hover(); await page.waitForTimeout(400);
  const t1 = await btn.evaluate((e) => getComputedStyle(e).transform);
  await page.mouse.down(); await page.waitForTimeout(250); const t2 = await btn.evaluate((e) => getComputedStyle(e).transform); await page.mouse.up();
  const hoverMedia = await page.evaluate(() => [...document.styleSheets].some((s) => { try { return [...s.cssRules].some((r) => /hover: hover/.test(r.cssText) && /pointer: fine/.test(r.cssText)); } catch { return false; } }));
  ac("AC12", /matrix\(1, 0, 0, 1, 0, -1\)|translateY\(-1px\)/.test(t1) && t1 !== t0 && t2 !== t1 && hoverMedia, `nghỉ=${t0} rê=${t1} bấm=${t2} hover:hover+pointer:fine=${hoverMedia}`);
  // AC22 màn đầu một dữ kiện cứng
  const first = await page.evaluate(() => { const h = innerHeight; const out: string[] = []; document.querySelectorAll("[data-part=landing] *").forEach((e) => { const r = e.getBoundingClientRect(); if (r.top < h && r.bottom > 0 && e.children.length === 0 && e.textContent?.trim()) out.push(e.textContent.trim()); }); return out.join(" | "); });
  const nums = (first.match(/\d[\d.,]*/g) ?? []);
  ac("AC22", nums.filter((n) => n === "146").length === 1 && !/1\.000 sinh viên/.test(first), `màn đầu: ${first.slice(0, 700)}`);
  // AC20 CTA
  const logins = await page.locator("[data-part=landing]").getByRole("link", { name: "Đăng nhập" }).evaluateAll((a) => a.map((x) => (x as HTMLAnchorElement).getAttribute("href"))); const regs = await page.locator("[data-part=landing]").getByRole("link", { name: "Tạo tài khoản" }).evaluateAll((a) => a.map((x) => (x as HTMLAnchorElement).getAttribute("href")));
  ac("AC20", logins.length >= 1 && logins.every((h) => h === "/login") && regs.length >= 1 && regs.every((h) => h === "/register"), `login=${logins} register=${regs}`);
  await ctx.close();
  // AC16 axe + bàn phím
  for (const [w, h] of [[1440, 900], [390, 844]] as const) {
    const o = await open(browser, w, h); await o.page.goto(`${BASE}/`); await o.page.waitForTimeout(2500);
    for (let y = 0; y < 6000; y += 700) { await o.page.mouse.wheel(0, 700); await o.page.waitForTimeout(250); }
    const ax = await new AxeBuilder({ page: o.page }).analyze(); const bad = ax.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? ""));
    const heads = await o.page.evaluate(() => [...document.querySelectorAll("h1,h2,h3")].map((e) => e.tagName)); const order = heads.every((t, i) => i === 0 || Number(t[1]) - Number(heads[i - 1][1]) <= 1);
    const noAlt = await o.page.evaluate(() => [...document.images].filter((i) => !i.hasAttribute("alt")).length);
    ac(`AC16-${w}`, bad.length === 0 && order && noAlt === 0, `${bad.map((v) => v.id + ":" + v.nodes.length).join(",") || "axe 0"}; thứ bậc ${order}; ảnh thiếu alt=${noAlt}`);
    await o.ctx.close();
  }
  const k = await open(browser, 1440, 900); await k.page.goto(`${BASE}/`); await k.page.waitForTimeout(2000);
  await k.page.keyboard.press("Tab"); await k.page.keyboard.press("Tab"); await k.page.keyboard.press("Tab");
  const names: string[] = []; for (let i = 0; i < 8; i++) { names.push(await k.page.evaluate(() => (document.activeElement as HTMLElement)?.textContent?.trim().slice(0, 30) ?? "")); await k.page.keyboard.press("Tab"); }
  const xv = k.page.getByRole("button", { name: /Xem video/ }).or(k.page.getByRole("link", { name: /Xem video/ })).first(); await xv.focus(); await k.page.keyboard.press("Enter"); await k.page.waitForTimeout(800); const open1 = await k.page.locator("dialog[open]").count(); await k.page.keyboard.press("Escape"); await k.page.waitForTimeout(500); const open2 = await k.page.locator("dialog[open]").count();
  const pb = k.page.locator("button[aria-pressed]").first(); await pb.focus(); const before = await pb.getAttribute("aria-pressed"); await k.page.keyboard.press("Enter"); await k.page.waitForTimeout(300); const after = await pb.getAttribute("aria-pressed");
  const ring = await xv.evaluate((e) => { (e as HTMLElement).focus(); const s = getComputedStyle(e); return `${s.outlineStyle} ${s.outlineWidth} ${s.outlineColor}`; });
  ac("AC16-kb", open1 === 1 && open2 === 0 && before !== after && /solid|auto/.test(ring), `dialog mở ${open1}→${open2}; tạm dừng ${before}→${after}; focus ring ${ring}; tab: ${names.join(" > ")}`);
  await k.ctx.close();
});
