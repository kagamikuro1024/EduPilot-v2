import { test } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { apiLogin } from "./r4ui-lib";
// SmkT4N: soi DOM "Hôm nay" của SV C và GV (nút/liên kết trong main: class, data-variant, chữ) để chốt cách đếm «hành động chính»
test("[dbg] home DOM", async ({ browser }, info) => {
  if (info.project.name !== "desktop") test.skip();
  for (const [u, tag] of [["sv.nguyco@edupilot.local", "SVC"], ["sv.gioi@edupilot.local", "SVA"], ["teacher@edupilot.local", "GV"]]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN", viewport: { width: 1440, height: 900 } });
    const page = await ctx.newPage(); await apiLogin(page, u, "Edupilot#Seed-2026"); await page.goto("/"); await page.waitForTimeout(6000);
    const rows = await page.locator("main a, main button").evaluateAll((els) => els.map((e) => `${e.tagName} cls=${(e.getAttribute("class") ?? "").slice(0, 60)} var=${e.getAttribute("data-variant")} part=${e.getAttribute("data-part")} «${(e.textContent ?? "").replace(/\s+/g, " ").trim().slice(0, 40)}»`));
    const h = (await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 300);
    appendFileSync("../docs/regression/evidence-4/logs/s4n-dbg.txt", `## ${tag}\n${h}\n${rows.join("\n")}\n`);
    await page.screenshot({ path: `../docs/regression/evidence-4/shots/s4-dbg-home-${tag}.png` }); await ctx.close();
  }
});
