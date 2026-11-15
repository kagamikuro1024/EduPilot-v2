import { test, expect, type Browser, type Page, type BrowserContext } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { mkdirSync, appendFileSync } from "fs";
import { execSync } from "child_process";

// Smoke hồi quy 10.8 (qc2/SmkH): các TC giao diện còn lại của US-AS-08 / 09 / 10 / 01-14. Chạy trên ep-r, lớp QC-AS (GV r108t).
test.skip(!process.env.QC_REAL, "cần stack thật");
const BASE = process.env.QC_BASE ?? "https://localhost:1143";
const PW = process.env.QC_PW ?? "";
const C = process.env.QC_CID ?? "";
const SIDS = (process.env.QC_SIDS ?? "").split(",").filter(Boolean);
const OUT = process.env.QC_OUT ?? "/tmp/qc-r108";
const GOLD = "/tmp/qc2-h";
mkdirSync(OUT, { recursive: true });
const rec = (tc: string, st: "PASS" | "FAIL" | "NOTE", note = "") => { console.log(`${st} ${tc} ${note}`.slice(0, 300)); appendFileSync(`${OUT}/ui.jsonl`, JSON.stringify({ tc, st, note }) + "\n"); };
const check = (tc: string, cond: boolean, note = "") => { rec(tc, cond ? "PASS" : "FAIL", note); return cond; };
const psql = (q: string) => execSync(`docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -At -F '|' -c "${q.replace(/"/g, '\\"')}"`, { encoding: "utf8" }).trim();
const T: Record<string, string> = {};
async function token(u: string) {
  if (T[u]) return T[u];
  for (let i = 0; i < 5; i++) {
    const r = await fetch(`${BASE}/api/v1/auth/login`, { method: "POST", headers: { "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) });
    if (r.status === 200) return (T[u] = ((await r.json()) as { access_token: string }).access_token);
    await new Promise((x) => setTimeout(x, 5000));
  }
  throw new Error("login " + u);
}
async function api(u: string, method: string, path: string, body?: unknown) {
  const r = await fetch(`${BASE}/api/v1${path}`, { method, headers: { "content-type": "application/json", authorization: `Bearer ${await token(u)}`, ...(method === "POST" || method === "PUT" ? { "Idempotency-Key": crypto.randomUUID() } : {}) }, body: body === undefined ? undefined : JSON.stringify(body) });
  const t = await r.text();
  return { s: r.status, j: t ? JSON.parse(t) : null };
}
import { webcrypto } from "crypto";
async function codeDev(user: string, code: string) {
  const kp = await webcrypto.subtle.generateKey({ name: "ECDSA", namedCurve: "P-256" }, true, ["sign"]);
  const spki = Buffer.from(await webcrypto.subtle.exportKey("spki", kp.publicKey)).toString("base64");
  const id = psql(`select id from users where email='${user}@edupilot.local'`);
  const sig = Buffer.from(await webcrypto.subtle.sign({ name: "ECDSA", hash: "SHA-256" }, kp.privateKey, new TextEncoder().encode(`edupilot-checkin-v1|code:${code}|${id}`))).toString("base64");
  return { kind: "WEB_KEY", public_key: spki, signature: sig };
}
let sidIdx = 0;
const nextSid = () => SIDS[sidIdx++];
const base = (n: number) => { sidIdx = n; };
async function openWin(sid: string, minutes?: number) { let o = await api("r108t", "POST", `/courses/${C}/sessions/${sid}/checkin-windows`, minutes ? { minutes } : {}); for (let i = 0; i < 4 && o.s >= 500; i++) { await new Promise((x) => setTimeout(x, 8000)); o = await api("r108t", "POST", `/courses/${C}/sessions/${sid}/checkin-windows`, minutes ? { minutes } : {}); } if (o.s !== 201) throw new Error("open " + o.s + JSON.stringify(o.j)); return o.j.id as string; }
async function tokOf(sid: string, wid: string) { const d = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows/${wid}/display`)).j; return { t: d.qr_url.split("#")[1] as string, code: (d.short_code as string).replace(/\s/g, ""), d }; }
const sv = async (u: string, sid: string, wid: string, extra: Record<string, unknown> = {}) => { const { t } = await tokOf(sid, wid); return api(u, "POST", `/courses/${C}/checkins`, { token: t, ...extra }); };
async function closeAll(sid: string) { const w = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items; for (const x of w) if (!x.closed_at) await api("r108t", "POST", `/courses/${C}/checkin-windows/${x.id}/close`); }
async function login(browser: Browser, user: string, w: number, h: number, mobile = false, opts: Record<string, unknown> = {}) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: mobile, isMobile: mobile, baseURL: BASE, ignoreHTTPSErrors: true, ...opts });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"), { timeout: 60_000 });
  return { ctx, page };
}
async function pickClass(page: Page, code: string) {
  const btn = page.locator("button[aria-label^='Chọn lớp']").first(); await btn.waitFor({ timeout: 20_000 }).catch(() => {});
  if (await btn.count() === 0) return;
  if (!((await btn.getAttribute("aria-label")) ?? "").includes(`${code} ·`)) { await btn.click(); await page.getByRole("menuitem", { name: new RegExp(`^${code}\\b`) }).or(page.getByText(code, { exact: true })).first().click(); await page.waitForTimeout(2500); }
}
async function more(page: Page, name: RegExp) {
  const trig = page.getByRole("button", { name: /^Thêm( hành động)?$/ }).first(); await trig.waitFor(); await page.waitForTimeout(400); await trig.click(); await page.waitForTimeout(300);
  await page.getByRole("menuitem", { name }).first().click();
}
const guard = async (tc: string, fn: () => Promise<void>) => { try { await fn(); } catch (e) { rec(tc, "FAIL", "ngoại lệ: " + String(e).replace(/\s+/g, " ").slice(0, 220)); } };
const staff = async (browser: Browser, w = 1440, h = 900, mobile = false) => { const x = await login(browser, "r108t", w, h, mobile); await pickClass(x.page, "QC-AS"); return x; };

test("DEF-03 rerun: TC-AS08-20", async ({ browser }) => {
  base(0);
  const { ctx, page } = await staff(browser);
  await guard("TC-AS08-20", async () => {
    const sid6 = nextSid(); const w6 = await openWin(sid6); await sv("sv.gioi", sid6, w6);
    await page.goto(`${BASE}/attendance?session=${sid6}`); await page.waitForTimeout(5000); console.log("URL", page.url(), "TOP", (await page.locator("[data-part=topbar]").innerText()).replace(/\n/g," | ").slice(0,80), "WL", await page.locator("[data-part=window-line]").count(), "MAIN", (await page.locator("main").innerText()).replace(/\n/g," | ").slice(0,260)); await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 });
    const posts: string[] = []; const resp: string[] = []; page.on("request", (q) => { if (q.url().endsWith("/close")) posts.push(q.headers()["idempotency-key"] ?? ""); }); page.on("requestfailed", (q) => { if (q.url().endsWith("/close")) resp.push("fail:" + (q.headers()["idempotency-key"] ?? "").slice(0, 8)); }); page.on("response", (r) => { if (r.url().endsWith("/close")) resp.push(r.status() + ":" + (r.request().headers()["idempotency-key"] ?? "").slice(0, 8)); });
    await ctx.setOffline(true); await page.getByRole("button", { name: "Đóng", exact: true }).click(); await page.waitForTimeout(2500);
    const q = await page.locator("[data-part=close-queued]").count();
    await ctx.setOffline(false); await page.waitForTimeout(12_000);
    const wins = (await api("r108t", "GET", `/courses/${C}/sessions/${sid6}/checkin-windows`)).j.items; const closedN = wins.filter((x: { closed_at: string | null }) => x.closed_at).length;
    const keys = [...new Set(posts.filter(Boolean))];
    check("TC-AS08-20 mất mạng khi «Đóng»: dòng «Chưa đóng»; có mạng → tự thử lại cùng Idempotency-Key, một cửa sổ đóng", q >= 1 && closedN === 1 && keys.length <= 1, `queued=${q} closed=${closedN} posts=${posts.length} keys=${keys.length} resp=${resp.join(",")}`);
  });
  await ctx.close();
  await ctx.close();
});
