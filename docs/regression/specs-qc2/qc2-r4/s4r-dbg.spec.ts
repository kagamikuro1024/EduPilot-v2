import { test } from "@playwright/test";
import { apiLogin, SEED_PW } from "./r4ui-lib";
import { appendFileSync } from "node:fs";
test.setTimeout(200_000);
test("[dbg] SV mở trang của GV/TA", async ({ page }) => {
  await apiLogin(page, "sv.gioi@edupilot.local", SEED_PW);
  const apis: string[] = []; page.on("response", (r) => { if (/\/api\/v1\//.test(r.url())) apis.push(`${r.status()} ${new URL(r.url()).pathname}`); });
  for (const r of ["/inbox", "/students", "/attendance", "/class/members", "/gradebook", "/grading", "/questions", "/documents", "/insights"]) {
    apis.length = 0; await page.goto(r); await page.waitForTimeout(2500);
    const t = (await page.locator("main").first().innerText().catch(() => "(no main)")).replace(/\s+/g, " ").slice(0, 220);
    appendFileSync("../docs/regression/evidence-4/logs/s4r-dbg-sv.txt", `${r} → ${new URL(page.url()).pathname} | ${t} | API: ${[...new Set(apis)].join(",")}\n`);
  }
});
