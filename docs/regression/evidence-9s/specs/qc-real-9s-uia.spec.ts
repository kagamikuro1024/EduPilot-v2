import { writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const out: Record<string, unknown> = {};
test("@real UIA 05-11 tích hợp lớp có Teams", async ({ page }) => {
  for (const w of [1440, 390]) {
    await page.setViewportSize({ width: w, height: 1000 });
    await uiLogin(page, "teacher"); await page.goto("/settings/integrations"); await page.locator("main").waitFor(); await page.waitForTimeout(2500);
    await page.evaluate((x) => localStorage.setItem("ep:ui:course", x), "01a1efdc-e3cf-7257-88aa-5d7ed0b39796"); await page.reload(); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
    const L = (await page.locator("main").innerText()).split("\n").map((s) => s.trim()).filter((s) => s && !s.startsWith("QD") && !s.startsWith("QC") && !/^76198/.test(s));
    out[`t${w}`] = L; out[`t${w}btn`] = await page.locator("main button").allInnerTexts();
    out[`t${w}tech`] = ["config_enc", "client_secret", "trace", "NEEDS"].filter((x) => L.join(" ").includes(x));
    out[`t${w}overflow`] = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
    out[`t${w}cards`] = await page.locator("main section, main [class*=card], main [class*=Card]").evaluateAll((els) => els.map((e) => { const cs = getComputedStyle(e); return { c: String((e as HTMLElement).className).slice(0, 40), border: cs.borderTopWidth, radius: cs.borderRadius, shadow: cs.boxShadow !== "none" }; }).slice(0, 8));
    await shot(page, `p705-integrations-teams-${w}`);
  }
  writeFileSync("/tmp/q9s/uia-t.json", JSON.stringify(out, null, 1));
});
