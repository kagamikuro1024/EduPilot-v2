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
  const btn = page.locator("button[aria-label^='Chọn lớp']").first();
  if (await btn.count() === 0) return;
  if (!((await btn.getAttribute("aria-label")) ?? "").includes(`${code} ·`)) { await btn.click(); await page.getByRole("menuitem", { name: new RegExp(`^${code}\\b`) }).or(page.getByText(code, { exact: true })).first().click(); await page.waitForTimeout(800); }
}
async function more(page: Page, name: RegExp) {
  const trig = page.getByRole("button", { name: /^Thêm( hành động)?$/ }).first(); await trig.waitFor(); await page.waitForTimeout(400); await trig.click(); await page.waitForTimeout(300);
  await page.getByRole("menuitem", { name }).first().click();
}
const guard = async (tc: string, fn: () => Promise<void>) => { try { await fn(); } catch (e) { rec(tc, "FAIL", "ngoại lệ: " + String(e).replace(/\s+/g, " ").slice(0, 220)); } };
const staff = async (browser: Browser, w = 1440, h = 900, mobile = false) => { const x = await login(browser, "r108t", w, h, mobile); await pickClass(x.page, "QC-AS"); return x; };
const requests = (page: Page) => { const l: string[] = []; page.on("request", (q) => { if (q.url().includes("/api/v1/")) l.push(new URL(q.url()).pathname); }); return l; };

test.setTimeout(480_000);

test("G /checkin: hết hạn, thông báo theo mã, mất mạng, chọn lớp, quét giả (AS09-06,11,13,23, AS11-05…11,13)", async ({ browser }) => {
  base(0);
  const pol = async (rot: number) => api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: rot, late_after_min: 10, require: "WEB" }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version });
  await guard("TC-AS09-06", async () => {
    await pol(5); const sid = nextSid(); const w = await openWin(sid); const x = await tokOf(sid, w); await new Promise((r) => setTimeout(r, 13_000));
    const { ctx, page } = await login(browser, "sv04", 375, 812, true);
    await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.waitForTimeout(6000); const tx = await page.locator("body").innerText();
    check("TC-AS09-06 token > 12 s: POST → 422 CHECKIN_TOKEN_EXPIRED → «Mã đã hết hạn. Quét lại mã trên màn chiếu.»", /Mã đã hết hạn\. Quét lại mã trên màn chiếu\./.test(tx), tx.replace(/\n/g, " ").slice(-90));
    await ctx.close(); await closeAll(sid);
  });
  await guard("TC-AS09-11/15", async () => {
    await pol(30); const sid = nextSid(); const w = await openWin(sid); const x = await tokOf(sid, w);
    const { ctx, page } = await login(browser, "sv05", 375, 812, true); const out: Array<[string, boolean, string]> = [];
    const probe = async (name: string, url: string, re: RegExp) => { await page.goto(url); await page.waitForTimeout(4500); const tx = await page.locator("body").innerText(); out.push([name, re.test(tx), tx.replace(/\n/g, " ").slice(-80)]); };
    await probe("sai", `${BASE}/checkin?c=${C}#${x.t.slice(0, -1)}${x.t.endsWith("A") ? "B" : "A"}`, /Mã không đúng\. Quét lại mã trên màn chiếu\./);
    await probe("ngoài lớp", `${BASE}/checkin?c=00000000-0000-4000-8000-000000000000#${x.t}`, /Bạn không thuộc lớp này\./);
    await probe("c không UUID", `${BASE}/checkin?c=xyz#${x.t}`, /Liên kết không đúng\./);
    await closeAll(sid); await probe("đã đóng", `${BASE}/checkin?c=${C}#${x.t}`, /Đã đóng điểm danh\./);
    await pol(30); const off = await api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: false }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version });
    void off; await probe("tắt", `${BASE}/checkin?c=${C}#${x.t}`, /Lớp này không điểm danh bằng mã\.|Đã đóng điểm danh\./);
    await pol(30);
    check("TC-AS09-11 thông báo đúng một câu theo mã: sai / ngoài lớp / đã đóng / tắt (hết hạn: AS09-06; 429: AS09-22)", out.every((o) => o[1]), out.map((o) => o[0] + ":" + o[1] + (o[1] ? "" : "[" + o[2] + "]")).join(", "));
    check("TC-AS09-15 c không phải UUID → «Liên kết không đúng.»; lớp lạ → «Bạn không thuộc lớp này.»", out[1][1] && out[2][1], "");
    await ctx.close();
  });
  await guard("TC-AS09-13", async () => {
    await pol(30); const sid = nextSid(); const w = await openWin(sid); const x = await tokOf(sid, w);
    const { ctx, page } = await login(browser, "sv06", 375, 812, true);
    await page.goto(`${BASE}/checkin`); await page.waitForTimeout(2000); await pickClass(page, "QC-AS"); const input = page.getByLabel("Mã điểm danh"); await input.waitFor({ timeout: 20_000 }); await input.fill(x.code);
    await ctx.setOffline(true); await page.locator("[data-part=checkin-dock] button").click({ timeout: 5000 }); await page.waitForTimeout(2500);
    const off = await page.locator("body").innerText(); const lab = await page.locator("[data-part=checkin-dock] button").innerText();
    await ctx.setOffline(false); await page.waitForTimeout(500); await page.locator("[data-part=checkin-dock] button").click({ timeout: 5000 }); await page.waitForTimeout(6000);
    const rows = psql(`select count(*) from attendance_checkins where window_id='${w}' and student_id=(select id from users where email='sv06@edupilot.local')`);
    check("TC-AS09-13 mất mạng («Chưa có mạng.») rồi «Thử lại»: gửi lại an toàn, đúng một lượt", /Chưa có mạng\./.test(off) && rows === "1", `nút='${lab}' rows=${rows}`); await ctx.close(); await closeAll(sid);
  });
  await guard("TC-AS09-23", async () => {
    await pol(30); const sid = nextSid(); const w = await openWin(sid); const x = await tokOf(sid, w);
    const { ctx, page } = await login(browser, "sv06", 375, 812, true); await page.goto(`${BASE}/checkin`); await page.waitForTimeout(2500);
    await pickClass(page, "QC-ASr108"); const input = page.getByLabel("Mã điểm danh"); await input.waitFor({ timeout: 20_000 }); await input.fill(x.code); await page.locator("[data-part=checkin-dock] button").click(); await page.waitForTimeout(3500);
    const bad = await page.locator("body").innerText();
    await pickClass(page, "QC-AS"); const in2 = page.getByLabel("Mã điểm danh"); await in2.waitFor(); await in2.fill(x.code); await page.locator("[data-part=checkin-dock] button").click(); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 20_000 });
    check("TC-AS09-23 nhập mã lớp A khi đang chọn lớp B (bật điểm danh): «Mã không đúng…»; đổi sang lớp A → thành công", /Mã không đúng/.test(bad), bad.replace(/\n/g, " ").slice(-90)); await ctx.close(); await closeAll(sid);
  });
  const nat = `(() => { window.__stopped = 0; window.Capacitor = { isNativePlatform: () => true, isPluginAvailable: () => false }; navigator.mediaDevices.getUserMedia = async () => { const c = document.createElement("canvas"); c.width = c.height = 16; const s = c.captureStream(5); c.getContext("2d").fillRect(0, 0, 16, 16); s.getTracks().forEach((t) => { const st = t.stop.bind(t); t.stop = () => { window.__stopped++; st(); }; }); return s; }; })()`;
  await guard("TC-AS11-05/06/09", async () => {
    await pol(30); const sid = nextSid(); const w = await openWin(sid); const x = await tokOf(sid, w);
    const { ctx, page } = await login(browser, "sv07", 375, 812, true); const posts: string[] = []; page.on("request", (q) => q.method() === "POST" && q.url().endsWith("/checkins") && posts.push(q.postData() ?? ""));
    const urls = [`https://evil.example/checkin?c=${C}#${x.t}`, "không phải URL", `${BASE}/other?c=${C}#${x.t}`, `${BASE}/checkin?c=not-a-uuid#${x.t}`, `${BASE}/checkin?c=${C}`, `${BASE}/checkin?c=${C}#${x.t}`];
    await page.addInitScript(nat + `; (() => { const q = ${JSON.stringify(urls)}; window.BarcodeDetector = class { static async getSupportedFormats() { return ["qr_code"]; } async detect() { const v = q.shift(); return v ? [{ rawValue: v }] : []; } }; })()`);
    await page.goto(`${BASE}/checkin?c=${C}`); await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click();
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
    const stopped = await page.evaluate(() => (window as unknown as { __stopped: number }).__stopped);
    check("TC-AS11-05 quét giả: mã đúng → xin camera, gửi MỘT POST với token, track.stop()", posts.length === 1 && JSON.parse(posts[0]).token === x.t && stopped > 0, `posts=${posts.length} stopped=${stopped}`);
    check("TC-AS11-06 QR lạ (origin khác / path khác / c không UUID / không #token) bị bỏ qua im lặng, không POST thừa", posts.length === 1, `posts=${posts.length}`);
    await ctx.close(); await closeAll(sid);
    // 09: ẩn tab khi đang quét → track ended
    const sid2 = nextSid(); const w2 = await openWin(sid2); await tokOf(sid2, w2);
    const b = await login(browser, "sv08", 375, 812, true);
    await b.page.addInitScript(nat + `; window.BarcodeDetector = class { static async getSupportedFormats() { return ["qr_code"]; } async detect() { return []; } }; window.__tr = []; const g = navigator.mediaDevices.getUserMedia; navigator.mediaDevices.getUserMedia = async (c) => { const s = await g(c); s.getTracks().forEach((t) => window.__tr.push(t)); return s; };`);
    await b.page.goto(`${BASE}/checkin?c=${C}`); await b.page.getByRole("button", { name: "Quét mã điểm danh" }).first().click(); await b.page.waitForTimeout(1500);
    const live = await b.page.evaluate(() => (window as unknown as { __tr: Array<{ readyState: string }> }).__tr.map((t) => t.readyState));
    await b.page.evaluate(() => { Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true }); document.dispatchEvent(new Event("visibilitychange")); }); await b.page.waitForTimeout(1500);
    const after = await b.page.evaluate(() => (window as unknown as { __tr: Array<{ readyState: string }> }).__tr.map((t) => t.readyState));
    check("TC-AS11-09 đang quét: visibilitychange→hidden → MediaStreamTrack ended", live.includes("live") && after.length > 0 && after.every((x) => x === "ended"), `trước=${live} sau=${after}`); await b.ctx.close(); await closeAll(sid2);
  });
  await guard("TC-AS11-07/08", async () => {
    const { ctx, page } = await login(browser, "sv09", 375, 812, true);
    await page.addInitScript(nat.replace("navigator.mediaDevices.getUserMedia = async", "navigator.mediaDevices.getUserMedia = async () => { throw new DOMException(\"denied\", \"NotAllowedError\"); }; const _x = async") + `; window.BarcodeDetector = class { static async getSupportedFormats() { return ["qr_code"]; } async detect() { return []; } };`);
    await page.goto(`${BASE}/checkin?c=${C}`); await page.getByRole("button", { name: "Quét mã điểm danh" }).first().click(); await page.waitForTimeout(2500);
    const tx = await page.locator("body").innerText();
    check("TC-AS11-08 từ chối camera: ô nhập mã + «Chưa cấp quyền camera.»", /Chưa cấp quyền camera\./.test(tx) && (await page.getByLabel("Mã điểm danh").count()) === 1, tx.replace(/\n/g, " ").slice(-90)); await ctx.close();
    const c2 = await login(browser, "sv10", 375, 812, true); await c2.page.addInitScript(`window.Capacitor = { isNativePlatform: () => true, isPluginAvailable: () => false }; window.BarcodeDetector = class { static async getSupportedFormats() { return []; } async detect() { return []; } };`);
    await c2.page.goto(`${BASE}/checkin?c=${C}`); await c2.page.waitForTimeout(2500);
    check("TC-AS11-07 qr_code không hỗ trợ: không nút quét; ô nhập mã hiện ngay", (await c2.page.getByRole("button", { name: "Quét mã điểm danh" }).count()) === 0 && (await c2.page.getByLabel("Mã điểm danh").count()) === 1, ""); await c2.ctx.close();
  });
});
