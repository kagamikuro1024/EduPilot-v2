import { chromium, newCtx, text, BASE } from "./lib.mjs";
const b = await chromium.launch();
for (const u of ["admin", "teacher", "ta", "sv.gioi"]) {
  const { ctx, page } = await newCtx(b, u);
  await page.goto(`${BASE}/settings/llm`); await page.waitForTimeout(4000);
  const t = await text(page);
  console.log("==", u, page.url(), "\n", t.slice(0, 700).replace(/\n+/g, " | "));
  await page.screenshot({ path: `/tmp/qc3oc/ui/probe-${u}.png` });
  await ctx.close();
}
await b.close();
