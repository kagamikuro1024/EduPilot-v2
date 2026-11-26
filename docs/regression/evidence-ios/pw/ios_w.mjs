// qc3 iOS — ca W chạy trên VPS bằng Playwright (Chromium + WebKit iPhone 15). Chạy: node ios_w.mjs <stage>
import { createRequire } from "node:module";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
const require = createRequire(process.env.PWREQ ?? "/srv/work/TA_Agent_dev3/frontend/package.json");
const pwmod = (() => { try { return require("@playwright/test"); } catch { return require("playwright"); } })();
const { chromium, webkit, devices } = pwmod;
let AxeBuilder; try { AxeBuilder = require("@axe-core/playwright").default; } catch {}
const BASE = "https://localhost"; // gateway nội bộ có thể khác; ưu tiên tên miền công khai
const HOST = process.env.HOST ?? "https://v2202609419695521098.happysrv.de";
const PW = process.env.PW; if (!PW) throw new Error("thiếu PW (đọc từ stdin qua ssh)");
const OUT = process.env.OUT ?? "/srv/work/.qc3-out/ios"; mkdirSync(OUT, { recursive: true });
const stage = process.argv[2] ?? "all";
const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 300)); };
const CID = "01a2136e-0560-7de4-9cda-c937a98e0916"; // QCMOB1
const SID = process.env.SID ?? "01a2150b-f05f-7b38-964b-b9ecac97b4a4";

async function api(path, { tok, method = "GET", body, headers = {} } = {}) {
  const r = await fetch(HOST + "/api/v1" + path, { method, headers: { "content-type": "application/json", ...(tok ? { authorization: "Bearer " + tok } : {}), ...headers }, body: body ? JSON.stringify(body) : undefined });
  const t = await r.text(); let j = {}; try { j = JSON.parse(t); } catch {}
  return { s: r.status, j };
}
const tokOf = async (u) => (await api("/auth/login", { method: "POST", body: { email: `${u}@edupilot.local`, password: PW } })).j.access_token;

async function loginPage(ctx, user) {
  const p = await ctx.newPage();
  await p.goto(HOST + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await p.locator("input[type=password]").fill(PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 20000 });
  return p;
}

if (stage === "axe" || stage === "all") {
  const b = await chromium.launch({ args: ["--no-sandbox"] });
  const ctx = await b.newContext({ viewport: { width: 375, height: 812 }, ignoreHTTPSErrors: true, locale: "vi-VN" });
  const p = await loginPage(ctx, "sv.gioi");
  await p.goto(HOST + "/checkin?c=" + CID, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(3000);
  const res = await new AxeBuilder({ page: p }).withTags(["wcag2a", "wcag2aa"]).analyze();
  const bad = res.violations.filter((v) => ["critical", "serious"].includes(v.impact));
  rec("TC-MOB-DAT-06", bad.length === 0 ? "P" : "F", `axe /checkin 375: ${res.violations.length} loại, critical/serious=${bad.length}${bad.length ? " " + bad.map((v) => v.id + ":" + v.nodes.length).join("|") : ""}`);
  // A11-14: reduced motion
  const ctx2 = await b.newContext({ viewport: { width: 375, height: 812 }, ignoreHTTPSErrors: true, locale: "vi-VN", reducedMotion: "reduce" });
  const p2 = await loginPage(ctx2, "sv.gioi");
  const states = [];
  for (const url of ["/checkin?c=" + CID, "/checkin"]) {
    await p2.goto(HOST + url, { waitUntil: "domcontentloaded" });
    for (const t of [200, 1200, 2500]) {
      await p2.waitForTimeout(t);
      states.push(await p2.evaluate(() => ({
        running: document.getAnimations().filter((a) => a.playState === "running" && (a.effect?.getComputedTiming().duration ?? 0) > 50 && (a.effect?.getComputedTiming().iterations ?? 1) === Infinity).length,
        longTransitions: [...document.querySelectorAll("*")].filter((e) => { const c = getComputedStyle(e); return parseFloat(c.transitionDuration) > 0.2 || (parseFloat(c.animationDuration) > 0.2 && c.animationName !== "none"); }).length,
        mq: matchMedia("(prefers-reduced-motion: reduce)").matches,
      })));
    }
  }
  const worst = states.reduce((a, s) => ({ running: Math.max(a.running, s.running), long: Math.max(a.long, s.longTransitions) }), { running: 0, long: 0 });
  rec("TC-MOB-A11-14", states.every((s) => s.mq) && worst.running === 0 && worst.long === 0 ? "P" : "F", `reduced-motion: mq=${states.every((s) => s.mq)}; vô hạn đang chạy=${worst.running}; chuyển/hoạt ảnh >0,2 s=${worst.long} (6 lần đo, 2 trạng thái)`);
  await b.close();
}

if (stage === "persist" || stage === "all") {
  const t = await tokOf("teacher");
  await api(`/courses/${CID}/sessions/${SID}/checkin-windows`, { tok: t, method: "POST", body: { minutes: 30 }, headers: { "Idempotency-Key": crypto.randomUUID() } }); // 409 nếu đang mở: bỏ qua
  const b = await webkit.launch();
  const ctx = await b.newContext({ ...devices["iPhone 15"], ignoreHTTPSErrors: true, locale: "vi-VN" });
  await ctx.addInitScript(() => { const proto = navigator.storage && Object.getPrototypeOf(navigator.storage); if (proto && proto.persist) { const o = proto.persist; proto.persist = function () { try { sessionStorage.setItem("__p", String(+(sessionStorage.getItem("__p") ?? 0) + 1)); } catch {} return o.call(this); }; } });
  await ctx.addInitScript(([k, v]) => { try { localStorage.setItem(k, v); } catch {} }, ["ep:ui:course", CID]);
  const p = await loginPage(ctx, "sv.gioi");
  const w = (await api(`/courses/${CID}/sessions/${SID}/checkin-windows`, { tok: t, method: "POST", body: { minutes: 30 }, headers: { "Idempotency-Key": crypto.randomUUID() } }));
  const wid = w.j.id ?? w.j.details?.window_id;
  const d = (await api(`/courses/${CID}/sessions/${SID}/checkin-windows/${wid}/display`, { tok: t })).j;
  await p.goto(HOST + "/checkin?c=" + CID, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(3000);
  const inp = p.locator("input").first(); await inp.waitFor({ timeout: 15000 });
  const d2 = (await api(`/courses/${CID}/sessions/${SID}/checkin-windows/${wid}/display`, { tok: t })).j;
  await inp.fill(d2.short_code ?? d.short_code); await p.keyboard.press("Enter");
  await p.waitForTimeout(6000);
  const n = await p.evaluate(async () => ({ calls: +(sessionStorage.getItem("__p") ?? 0), dbs: (await indexedDB.databases()).map((d) => d.name).join(","), recs: await new Promise((res) => { const q = indexedDB.open("edupilot.device.v1"); q.onsuccess = () => { try { const r = q.result.transaction("keys").objectStore("keys").count(); r.onsuccess = () => res(r.result); r.onerror = () => res("err"); } catch (e) { res("noStore:" + e.name); } }; q.onerror = () => res("openErr"); }) }));
  const txt = (await p.locator("main").innerText()).slice(0, 120).replace(/\s+/g, " ");
  await p.screenshot({ path: OUT + "/PWA-22.png" });
  rec("TC-MOB-PWA-22", n.calls === 1 ? "P" : "F", `WebKit iPhone 15: persist() gọi ${n.calls} lần (CSDL: ${n.dbs}, bản ghi khoá: ${n.recs}) sau khi tạo khoá + điểm danh; màn: «${txt}»`);
  await b.close();
}
writeFileSync(OUT + `/results-${stage}.json`, JSON.stringify(R, null, 1));
