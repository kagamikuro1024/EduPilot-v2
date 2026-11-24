import { chromium, newCtx, text, BASE } from "./lib.mjs";
const b = await chromium.launch();
for (const u of ["teacher", "admin"]) {
  const { ctx, page } = await newCtx(b, u);
  await page.goto(`${BASE}/settings/llm`); await page.waitForTimeout(4000);
  const btns = await page.locator("button:visible").evaluateAll(els => els.map(e => (e.innerText || e.getAttribute("aria-label") || "").trim().replace(/\s+/g," ")+(e.disabled?" [disabled]":"")).filter(Boolean));
  console.log("==", u, JSON.stringify(btns.slice(0, 60)));
  const sw = await page.locator("[role=switch]:visible").evaluateAll(els => els.map(e => (e.getAttribute("aria-label")||"")+(e.disabled||e.getAttribute("aria-disabled")==="true"?" [disabled]":"")));
  console.log("switches", JSON.stringify(sw.slice(0,10)));
  await ctx.close();
}
await b.close();
