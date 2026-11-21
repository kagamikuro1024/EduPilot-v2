import { writeFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
test.skip(!process.env.QC_REAL, "cần stack thật");
const PW = process.env.QC_PW ?? "";
const OUT = process.env.QC_OUT ?? "/tmp/qc10-shots";
const BAD = /\b(AI|RAG|citation|model|LLM|prompt|embedding|token|streak|huy hiệu|xếp hạng)\b/i;
async function login(browser: Browser, user: string, w: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((id) => { try { if (!localStorage.getItem("ep:ui:course")) localStorage.setItem("ep:ui:course", id); } catch {} }, process.env.QC_C1 ?? "");
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page };
}
const rows: string[] = [];
for (const [user, routes] of [["sv.kha", ["/practice", "/practice/history", "/"]], ["teacher", ["/questions", "/gradebook", "/"]]] as const) {
  for (const w of [1440, 375]) {
    test(`QCR10 ${user} ${w}`, async ({ browser }) => {
      const { ctx, page } = await login(browser, user, w);
      for (const r of routes) {
        await page.goto(r); await page.waitForTimeout(2000);
        const m = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth, t: document.body.innerText, h: document.querySelector("main h1")?.textContent ?? "" }));
        const ok = m.sw <= m.cw && (user.startsWith("teacher") || !BAD.test(m.t)) && m.t.length > 50;
        console.log(`QCR10 ${ok ? "PASS" : "FAIL"} ${user} ${w} ${r} sw=${m.sw}/${m.cw} h1="${m.h}" cấm=${(m.t.match(BAD) ?? ["không"])[0]} len=${m.t.length}`);
        await page.screenshot({ path: `${OUT}/z-${user}-${w}-${r.replace(/\W+/g, "_")}.png` });
      }
      await ctx.close();
    });
  }
}
