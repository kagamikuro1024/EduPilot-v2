import { expect, type Page } from "@playwright/test";
import { appendFileSync, mkdirSync } from "node:fs";
export const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
export const EV = process.env.EV ?? "../docs/regression/evidence-8";
mkdirSync(`${EV}/shots`, { recursive: true }); mkdirSync(`${EV}/logs`, { recursive: true });
let LAST = "teacher";
export async function login(page: Page, user: string) {
  LAST = user;
  for (let t = 0; t < 4; t++) {
    await page.goto("/login"); await page.waitForLoadState("networkidle").catch(() => undefined); await page.waitForTimeout(1500);
    const em = page.locator("input[type=email]"), pw = page.locator("input[type=password]");
    for (let k = 0; k < 4; k++) { await em.fill(`${user}@edupilot.local`); await pw.fill(PW); await page.waitForTimeout(400); if ((await pw.inputValue()) === PW && (await em.inputValue()) === `${user}@edupilot.local`) break; }
    await page.keyboard.press("Enter");
    const ok = await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 40_000 }).then(() => true).catch(() => false);
    if (ok) { await page.locator("[data-part=topbar]").waitFor({ timeout: 20_000 }); await page.waitForTimeout(1500); return; }
    await page.waitForTimeout(10_000);
  }
  throw new Error("login failed " + user);
}
export async function go(page: Page, url: string) {
  for (let t = 0; t < 3; t++) {
    const pushed = !page.url().includes("/login") && page.url().startsWith("http") && !process.env.HARD && await page.evaluate((u) => { const r = (window as any).next?.router; if (!r) return false; r.push(u); return true; }, url).catch(() => false);
    if (!pushed) await page.goto(url);
    if (page.url().includes("/login")) { await page.waitForTimeout(8000); await login(page, LAST); continue; }
    await page.locator("main").first().waitFor({ state: "attached", timeout: 20_000 }).catch(() => undefined);
    let prev = -1, same = 0;
    for (let i = 0; i < 45 && same < 3; i++) { const n = await page.evaluate(() => { const m = document.querySelector("main"); return (m?.innerText ?? "").length * 1000 + (m?.querySelectorAll("*").length ?? 0) - (document.querySelectorAll("[class*=skeleton],[data-skeleton],[aria-busy=true]").length ? 1e9 : 0); }); same = n === prev ? same + 1 : 0; prev = n; await page.waitForTimeout(700); }
    return;
  }
}
export const log = (name: string, line: string) => { appendFileSync(`${EV}/logs/s8r-ui-${name}.log`, line + "\n"); console.log(line); };
export const shot = (page: Page, name: string) => page.screenshot({ path: `${EV}/shots/s8r-${name}.png` });
export const IDS: Record<string, any> = JSON.parse(require("node:fs").readFileSync("/tmp/qc2-s8r/ids.json", "utf8"));
import { execFileSync } from "node:child_process";
export const cid = (code: string) => (IDS[code]?.id as string) ?? execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", `select id from courses where class_code='${code}'`], { encoding: "utf8" }).trim();
