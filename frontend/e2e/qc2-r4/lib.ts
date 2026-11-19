import { type Page } from "@playwright/test";
import { appendFileSync, mkdirSync, readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
export const EV = process.env.EV ?? "../docs/regression/evidence-4";
mkdirSync(`${EV}/shots`, { recursive: true }); mkdirSync(`${EV}/logs`, { recursive: true });
export const S: any = (() => { try { return JSON.parse(readFileSync("/tmp/qc2-s4-ui.json", "utf8")); } catch { return {}; } })();
export const SEEDPW = "Edupilot#Seed-2026";
export const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], { encoding: "utf8" }).trim();
export const OUT = `${EV}/results-smk34/ui-s4.tsv`;
export const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 900)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note.slice(0, 300)}`); };
export const shot = (page: Page, name: string) => page.screenshot({ path: `${EV}/shots/s4-${name}.png`, fullPage: false });
export async function login(page: Page, email: string, pw = SEEDPW) {
  const em = email.includes("@") ? email : `${email}@edupilot.local`;
  for (let t = 0; t < 4; t++) {
    execFileSync("docker", ["exec", "ep-r-redis-1", "sh", "-c", "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del"]);
    await page.goto("/login"); await page.waitForLoadState("networkidle").catch(() => undefined); await page.waitForTimeout(1200);
    const e = page.locator("input[type=email]"), p = page.locator("input[type=password]");
    for (let k = 0; k < 4; k++) { await e.fill(em); await p.fill(pw); await page.waitForTimeout(300); if ((await p.inputValue()) === pw && (await e.inputValue()) === em) break; }
    await page.keyboard.press("Enter");
    const ok = await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 }).then(() => true).catch(() => false);
    if (ok) { await page.locator("[data-part=topbar]").waitFor({ timeout: 20_000 }).catch(() => undefined); await page.waitForTimeout(1200); return; }
    await page.waitForTimeout(8000);
  }
  const bt = (await page.locator("body").innerText().catch(() => "")).replace(/\s+/g, " ").slice(0, 300);
  await page.screenshot({ path: `${EV}/shots/s4-loginfail-${Date.now()}.png` }).catch(() => undefined);
  throw new Error("login failed " + email + " url=" + page.url() + " body=" + bt);
}
export const settle = async (page: Page, ms = 2500) => { await page.waitForLoadState("networkidle").catch(() => undefined); await page.waitForTimeout(ms); };
export const text = async (page: Page, sel = "main") => ((await page.locator(sel).first().innerText().catch(() => "")) as string).replace(/\s+/g, " ");
export const TECH = /\b(RAG|PII|trace_id|provider|fallback|embedding|token|JWT|payload|enrollment|SQL)\b/i;
