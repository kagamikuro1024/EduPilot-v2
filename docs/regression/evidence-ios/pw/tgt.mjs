import { createRequire } from "node:module"; const require = createRequire("/tmp/qc3pw/package.json");
const { webkit, devices } = require("playwright");
import { writeFileSync } from "node:fs";
const HOST = "https://v2202609419695521098.happysrv.de", PW = process.env.PW, OUT = process.env.OUT;
const CID = "01a2136e-0560-7de4-9cda-c937a98e0916";
const R = []; const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 700)); };
const b = await webkit.launch();
async function login(ctx, user) {
  await ctx.addInitScript(([k, v]) => { try { localStorage.setItem(k, v); } catch {} }, ["ep:ui:course", CID]);
  const p = await ctx.newPage(); await p.goto(HOST + "/login", { waitUntil: "domcontentloaded" });
  await p.locator("input[type=email]").fill(user + "@edupilot.local"); await p.locator("input[type=password]").fill(PW); await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 20000 }); return p;
}
const measure = (p) => p.evaluate(() => {
  const vis = (e) => { const r = e.getBoundingClientRect(); const s = getComputedStyle(e); return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0 && r.top < innerHeight && r.left < innerWidth && s.visibility !== "hidden" && s.display !== "none"; };
  const els = [...document.querySelectorAll("button, a[href], input, select, textarea, [role=button], [role=tab]")].filter(vis);
  return {
    coarse: matchMedia("(pointer: coarse)").matches, w: innerWidth, sw: document.documentElement.scrollWidth,
    items: els.map((e) => { const r = e.getBoundingClientRect(); return { t: (e.innerText || e.getAttribute("aria-label") || e.getAttribute("placeholder") || e.tagName).trim().replace(/\s+/g, " ").slice(0, 28), w: Math.round(r.width), h: Math.round(r.height), nav: !!e.closest("nav, [data-part=bottom-nav], [data-part=sidebar]") }; }),
  };
});
const small = (m) => m.items.filter((i) => i.w < 44 || i.h < 44);
// --- TGT-06: ngang ≥ 720 px, cảm ứng ---
{
  const ctx = await b.newContext({ ...devices["iPhone 15 landscape"], viewport: { width: 844, height: 390 }, ignoreHTTPSErrors: true, locale: "vi-VN" });
  const p = await login(ctx, "sv.gioi");
  await p.goto(HOST + "/checkin?c=" + CID, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(3500);
  const idle = await measure(p); await p.screenshot({ path: OUT + "/TGT-06-idle.png" });
  const inp = p.locator("input").first(); await inp.fill("ZZZ ZZZ"); await p.keyboard.press("Enter"); await p.waitForTimeout(3500);
  const err = await measure(p); await p.screenshot({ path: OUT + "/TGT-06-error.png" });
  const named = [...idle.items, ...err.items].filter((i) => /Thoát|Nhập mã|Thử lại/i.test(i.t));
  const side = idle.items.filter((i) => i.nav);
  const bad = [...small(idle), ...small(err)];
  rec("TC-MOB-TGT-06a", bad.length === 0 && idle.coarse ? "P" : "F", `/checkin ngang 844×390 pointer:coarse=${idle.coarse}: ${idle.items.length + err.items.length} điều khiển, <44: ${JSON.stringify(bad.slice(0, 6))}; mục thanh bên/dưới: ${side.length} (${side.slice(0, 5).map((s) => s.t + " " + s.w + "×" + s.h).join("; ")}); gặp «Thoát/Nhập mã/Thử lại»: ${JSON.stringify(named)}`);
  await ctx.close();
  // «Thoát» ở màn chiếu (GV)
  const ctx2 = await b.newContext({ ...devices["iPhone 15 landscape"], viewport: { width: 844, height: 390 }, ignoreHTTPSErrors: true, locale: "vi-VN" });
  const t = await login(ctx2, "teacher");
  await t.goto(HOST + "/attendance/display", { waitUntil: "domcontentloaded" }); await t.waitForTimeout(3500);
  const d = await measure(t); await t.screenshot({ path: OUT + "/TGT-06-display.png" });
  rec("TC-MOB-TGT-06b", small(d).length === 0 && d.coarse ? "P" : "F", `/attendance/display ngang: coarse=${d.coarse}, điều khiển: ${d.items.map((i) => i.t + " " + i.w + "×" + i.h).join("; ")}`);
  await ctx2.close();
}
// --- UIW-22: cỡ nhỏ nhất / lớn nhất của iPhone ---
for (const [name, vp] of [["iPhone SE 3", { ...devices["iPhone 13 mini"], viewport: { width: 375, height: 667 }, screen: { width: 375, height: 667 } }], ["iPhone 15 Pro Max", devices["iPhone 15 Pro Max"]]]) {
  const ctx = await b.newContext({ ...vp, ignoreHTTPSErrors: true, locale: "vi-VN" });
  const p = await login(ctx, "sv.gioi");
  await p.goto(HOST + "/checkin?c=" + CID, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(3500);
  const m = await measure(p); await p.screenshot({ path: OUT + `/UIW-22-${name.replace(/ /g, "")}.png` });
  const overlap = await p.evaluate(() => { const ts = [...document.querySelectorAll("main h1, main h2, main p, main label, main button, main input")].filter((e) => e.getBoundingClientRect().width > 0).map((e) => e.getBoundingClientRect()); let c = 0; for (let i = 0; i < ts.length; i++) for (let j = i + 1; j < ts.length; j++) { const a = ts[i], b = ts[j]; const ox = Math.min(a.right, b.right) - Math.max(a.left, b.left), oy = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top); if (ox > 2 && oy > 2 && !(a.left <= b.left && a.right >= b.right && a.top <= b.top && a.bottom >= b.bottom) && !(b.left <= a.left && b.right >= a.right && b.top <= a.top && b.bottom >= a.bottom)) c++; } return c; });
  rec("TC-MOB-UIW-22-" + name.replace(/ /g, ""), m.sw <= m.w && small(m).length === 0 && overlap === 0 ? "P" : "F", `${name} ${m.w}px: scrollWidth=${m.sw}, điều khiển<44: ${JSON.stringify(small(m).slice(0, 6))}, chồng khung chữ=${overlap}`);
  await ctx.close();
}
writeFileSync(OUT + "/results-tgt.json", JSON.stringify(R, null, 1)); await b.close();
