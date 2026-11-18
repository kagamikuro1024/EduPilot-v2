import { expect, type Page } from "@playwright/test";
import { appendFileSync, mkdirSync, readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
export const PW = "Edupilot#Seed-2026";
export const EV = "/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-7/r3";
mkdirSync(`${EV}/shots`, { recursive: true });
let LAST = "teacher";
export async function login(page: Page, user: string) {
  LAST = user;
  for (let t = 0; t < 8; t++) {
    await page.goto("/login"); await page.waitForLoadState("networkidle").catch(() => undefined); await page.waitForTimeout(1200);
    const em = page.locator("input[type=email]"), pw = page.locator("input[type=password]");
    for (let k = 0; k < 4; k++) { await em.fill(`${user}@edupilot.local`); await pw.fill(PW); await page.waitForTimeout(300); if ((await pw.inputValue()) === PW && (await em.inputValue()) === `${user}@edupilot.local`) break; }
    await page.keyboard.press("Enter");
    const ok = await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 40_000 }).then(() => true).catch(() => false);
    if (ok) { await page.locator("[data-part=topbar]").waitFor({ timeout: 20_000 }); await page.waitForTimeout(1200); return; }
    await page.waitForTimeout(20_000);
  }
  throw new Error("login failed " + user);
}
export async function settle(page: Page) {
  let prev = -1, same = 0;
  for (let i = 0; i < 40 && same < 3; i++) { const n = await page.evaluate(() => { const m = document.querySelector("main"); return (m?.innerText ?? "").length * 1000 + (m?.querySelectorAll("*").length ?? 0) - (document.querySelectorAll("[class*=skeleton],[data-skeleton],[aria-busy=true]").length ? 1e9 : 0); }); same = n === prev ? same + 1 : 0; prev = n; await page.waitForTimeout(600); }
}
export async function go(page: Page, url: string) {
  await page.goto(url);
  for (let k = 0; k < 4 && page.url().includes("/login"); k++) { await page.waitForTimeout(8000 + k * 8000); await login(page, LAST); await page.goto(url); await page.waitForTimeout(1500); }
  await settle(page);
}
export const log = (name: string, line: string) => { appendFileSync(`${EV}/ui-${name}.log`, line + "\n"); console.log(line); };
export const shot = (page: Page, name: string) => page.screenshot({ path: `${EV}/shots/${name}.png` });
export const psql = (sql: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
export const FX = (n: string) => JSON.parse(readFileSync(`/tmp/qc3e/${n}.json`, "utf8"));
export { expect };

import type { Browser } from "@playwright/test";
export async function newUser(browser: Browser, user: string, vp = { width: 1280, height: 800 }) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: "https://localhost:1143", viewport: vp, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  const page = await ctx.newPage(); await login(page, user); return { ctx, page };
}
