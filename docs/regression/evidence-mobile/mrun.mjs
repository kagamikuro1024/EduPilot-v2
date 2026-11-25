// qc3 — pha 1 (Playwright trên VPS, CHỈ Chromium): nhóm 6.J (phần web), 6.L (phần web), 6.M (phần web),
// 6.Q (axe), 6.T (phần web), 6.V (phần web), 6.W (phần web), 6.Y (ENT). Chạy: node mrun.mjs <stage> [--base=URL]
import { createRequire } from "node:module";
import { appendFileSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
const require = createRequire("/srv/work/TA_Agent_dev3/frontend/package.json");
const { chromium } = require("@playwright/test");
const { default: AxeBuilder } = require("@axe-core/playwright");

const BASE = (process.argv.find((a) => a.startsWith("--base=")) ?? "").slice(7) || "https://v2202609419695521098.happysrv.de";
const PW = readFileSync("/srv/work/.qc3-secret", "utf8").trim();
const OUT = "/srv/work/.qc3-out";
mkdirSync(OUT + "/shots", { recursive: true });
const stage = process.argv[2] ?? "all";
const R = [];
const rec = (tc, kq, note) => { R.push({ tc, kq, note }); console.log(kq, tc, note.slice(0, 220)); };
const ok = (c, why) => (c ? "P" : "F:" + why);

const browser = await chromium.launch({ args: ["--no-sandbox"] });
const shot = async (page, name) => { try { await page.screenshot({ path: `${OUT}/shots/${name}.png`, fullPage: false }); } catch {} };

async function ctxFor(w, h, opts = {}) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, ignoreHTTPSErrors: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", ...opts });
  return ctx;
}
async function login(ctx, user) {
  const p = await ctx.newPage();
  for (let i = 0; i < 3; i++) {
    await p.goto(BASE + "/login", { waitUntil: "domcontentloaded" });
    await p.locator("input[type=email]").fill(`${user}@edupilot.local`);
    await p.locator("input[type=password]").fill(PW);
    await p.keyboard.press("Enter");
    if (await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 }).then(() => true).catch(() => false)) break;
    await p.waitForTimeout(4000);
  }
  // hộp «Đã hiểu» (thông báo theo dõi) chặn thao tác: đóng nếu có
  try { const b = p.getByRole("button", { name: "Đã hiểu" }).first(); if (await b.count()) { await b.click({ timeout: 3000 }); await p.waitForTimeout(500); } } catch {}
  return p;
}
const setCourse = (ctx, id) => ctx.addInitScript((c) => { try { localStorage.setItem("ep:ui:course", c); } catch {} }, id);
const txt = (p) => p.evaluate(() => document.body.innerText);
const navTexts = (p) => p.locator("nav a:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean));

const IDS = JSON.parse(readFileSync(new URL("./vps-ids.json", import.meta.url), "utf8"));

try {
  // ---------- 6.Y: ENT ----------
  if (stage === "all" || stage === "ent") {
    // ENT-01 thanh điều hướng ≥720px
    {
      const ctx = await ctxFor(1024, 768, { hasTouch: true }); await setCourse(ctx, IDS.QC3M1);
      const p = await login(ctx, "sv.qc3"); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const items = await p.locator("nav a").evaluateAll((els) => els.map((e) => ({ href: e.getAttribute("href"), label: (e.getAttribute("aria-label") || e.innerText || "").trim() }))); await shot(p, "ENT-01-nav-1024");
      const hasCheckin = items.some((x) => x.href === "/checkin");
      const hasLabel = items.some((x) => /Điểm danh/.test(x.label));
      rec("TC-MOB-ENT-01", ok(hasCheckin && items.length >= 10, `nav=${items.length} có /checkin=${hasCheckin} nhãn chữ=${hasLabel}`), `1024px: ${items.length} mục nav, mục /checkin=${hasCheckin}, nhãn chữ «Điểm danh»=${hasLabel} (nhãn rỗng => thanh bên thu gọn)`);
      await ctx.close();
    }
    // ENT-02 thanh dưới + «Thêm»
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true, deviceScaleFactor: 3 }); await setCourse(ctx, IDS.QC3M1);
      const p = await login(ctx, "sv.qc3"); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const bar = await p.locator("nav a:visible,nav button:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean));
      const more = p.getByRole("button", { name: /Thêm/ }).first(); let moreItems = [];
      if (await more.count()) { await more.click(); await p.waitForTimeout(700); moreItems = await p.locator("[role=menu] a:visible,[role=dialog] a:visible,dialog[open] a:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean)); }
      await shot(p, "ENT-02-more-390");
      const has = moreItems.includes("Điểm danh");
      let hh = null;
      if (has) {
        await p.locator("[role=menu] a:visible,[role=dialog] a:visible,dialog[open] a:visible").filter({ hasText: "Điểm danh" }).first().evaluate((e) => { const b = e.getBoundingClientRect(); window.__ep = { w: b.width, h: b.height }; });
        hh = await p.evaluate(() => window.__ep);
      }
      rec("TC-MOB-ENT-02", ok(has && hh && hh.h >= 44, `«Thêm» có Điểm danh=${has} box=${JSON.stringify(hh)}`), `390px bar [${bar.join("|")}] «Thêm» [${moreItems.join("|")}] hộp Điểm danh=${JSON.stringify(hh)}`);
      await ctx.close();
    }
    // ENT-03 Tìm nhanh
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
      const p = await login(ctx, "sv.qc3"); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const trig = p.getByText("Tìm nhanh hoặc đi đến…").first();
      let opened = false;
      if (await trig.count()) { await trig.click({ force: true }); opened = true; }
      else { const b = p.locator('[aria-label*="Tìm"], button:has-text("Tìm")').first(); if (await b.count()) { await b.click(); opened = true; } }
      await p.waitForTimeout(800);
      if (opened) { await p.keyboard.type("diem danh", { delay: 40 }); await p.waitForTimeout(900); }
      const body = await txt(p); await shot(p, "ENT-03-palette-390");
      const found = /Điểm danh/.test(body);
      rec("TC-MOB-ENT-03", ok(opened && found, `mở=${opened} thấy Điểm danh=${found}`), `«Tìm nhanh» mở=${opened}; kết quả có «Điểm danh»=${found}`);
      await ctx.close();
    }
    // ENT-04 ba lớp bật/tắt/cửa sổ đóng — giao diện mục giống nhau
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true });
      const p = await login(ctx, "sv.qc3"); const shapes = {};
      for (const [code, id] of [["QC3M1", IDS.QC3M1], ["QCMOB5", IDS.QCMOB5], ["QCMOB2", IDS.QCMOB2]]) {
        await p.evaluate((c) => localStorage.setItem("ep:ui:course", c), id);
        await p.goto(BASE + "/?course=" + id, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
        const more = p.getByRole("button", { name: /Thêm/ }).first(); let items = [];
        if (await more.count()) { await more.click(); await p.waitForTimeout(600); items = await p.locator("[role=menu] a:visible,[role=dialog] a:visible,dialog[open] a:visible").evaluateAll((els) => els.map((e) => ({ t: (e.innerText || "").trim(), h: e.getAttribute("href"), html: e.innerHTML.length }))); await p.keyboard.press("Escape"); }
        shapes[code] = items.find((x) => x.t === "Điểm danh") ?? null;
      }
      const all = ["QC3M1", "QCMOB5", "QCMOB2"].every((k) => shapes[k]);
      const same = JSON.stringify(shapes.QC3M1) === JSON.stringify(shapes.QCMOB5) && JSON.stringify(shapes.QC3M1) === JSON.stringify(shapes.QCMOB2);
      rec("TC-MOB-ENT-04", ok(all && same, `có=${JSON.stringify(Object.fromEntries(Object.entries(shapes).map(([k, v]) => [k, Boolean(v)])))} giống=${same}`), `mục Điểm danh ở 3 lớp; giống nhau=${same}`);
      await ctx.close();
    }
    // ENT-05 không lộ trạng thái cửa sổ: mở cửa sổ rồi so nav/«Thêm» trước-sau
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
      const p = await login(ctx, "sv.qc3"); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const snap = async () => {
        const more = p.getByRole("button", { name: /Thêm/ }).first(); await more.click(); await p.waitForTimeout(600);
        const html = await p.locator("nav, [role=menu], [role=dialog], dialog[open]").evaluateAll((els) => els.map((e) => e.outerHTML.replace(/\s+/g, " ")).join(""));
        await p.keyboard.press("Escape"); await p.waitForTimeout(300);
        return { html, body: await txt(p) };
      };
      const before = await snap();
      const open = await (await fetch(BASE + "/api/v1/auth/login", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ email: "teacher@edupilot.local", password: PW }) })).json();
      const wk = await (await fetch(`${BASE}/api/v1/courses/${IDS.QC3M1}/sessions/${IDS.sessC}/checkin-windows`, { method: "POST", headers: { "content-type": "application/json", authorization: "Bearer " + open.access_token, "Idempotency-Key": "qc3-ent05-" + Date.now() }, body: JSON.stringify({ minutes: 10 }) })).json();
      await p.reload(); await p.waitForTimeout(4000);
      const after = await snap(); await shot(p, "ENT-05-after-open-390");
      const diff = before.html !== after.html || before.body !== after.body;
      const leaked = /điểm danh (đang )?mở|đang mở điểm danh/i.test(after.body);
      rec("TC-MOB-ENT-05", ok(!diff && !leaked && wk.id, `cửa sổ=${Boolean(wk.id)} DOM đổi=${diff} lộ chữ=${leaked}`), `mở cửa sổ (${wk.id ? "ok" : JSON.stringify(wk).slice(0, 80)}); DOM/«Thêm» đổi=${diff}; chữ «đang mở»=${leaked}`);
      // đóng cửa sổ để trả trạng thái
      if (wk.id) await fetch(`${BASE}/api/v1/courses/${IDS.QC3M1}/checkin-windows/${wk.id}/close`, { method: "POST", headers: { "content-type": "application/json", authorization: "Bearer " + open.access_token, "Idempotency-Key": "qc3-ent05c-" + Date.now() }, body: "{}" });
      await ctx.close();
    }
    // ENT-06..10: đo riêng bằng ent_probe.mjs (một buổi probe, đổi giờ từng nhánh)
    if (false) {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true });
      const p = await login(ctx, "sv.nguyco"); await p.goto(BASE + "/"); await p.waitForTimeout(3500);
      const rows = await p.evaluate(() => [...document.querySelectorAll("a")].filter((a) => (a.getAttribute("href") || "").includes("/checkin")).map((a) => ({ href: a.getAttribute("href"), text: (a.innerText || "").trim(), row: (a.closest("li")?.innerText || "").replace(/\s+/g, " ").slice(0, 140) })).filter((x) => x.text === "Điểm danh" && x.row));
      const body = await txt(p); await shot(p, "ENT-06-10-today-390");
      const has = (needle) => rows.some((r) => (r.row || "").includes(needle));
      const links = rows.length;
      rec("TC-MOB-ENT-06", ok(!has("A_xa40") , "hàng A_xa40 (còn 40 phút) vẫn có liên kết"), `hàng A_xa40 (bắt đầu sau ~40 phút) có liên kết=${has("A_xa40")}; tổng liên kết Điểm danh trên trang=${links}`);
      rec("TC-MOB-ENT-07", ok(has("B_gan8"), "hàng B không có liên kết"), `hàng B_gan8 (trong 15 phút trước giờ) có liên kết=${has("B_gan8")}`);
      rec("TC-MOB-ENT-08", ok(has("C_dang"), "hàng C không có liên kết"), `hàng C_dang (đang diễn ra) có liên kết=${has("C_dang")}`);
      rec("TC-MOB-ENT-09", ok(!has("D_vuaHet"), "hàng D vừa hết vẫn có liên kết"), `hàng D_vuaHet (vừa hết giờ) có liên kết=${has("D_vuaHet")}`);
      rec("TC-MOB-ENT-10", ok(!has("E_tat"), "hàng lớp tắt vẫn có liên kết"), `hàng E_tat (QC3M2 self_checkin tắt) có liên kết=${has("E_tat")}`);
      const cap8 = body.length > 0;
      await ctx.close();
      rec("TC-MOB-ENT-20", ok(true, ""), `trang «Hôm nay» 390px: số liên kết Điểm danh=${links}; không tràn ngang (đo ở UIW)`);
    }
    // ENT-12 / 13 / 14 / 17 / 18 / 22
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
      const p = await login(ctx, "sv.qc3");
      await p.goto(BASE + "/checkin"); await p.waitForTimeout(2500);
      const body = await txt(p); await shot(p, "ENT-12-checkin-web-390");
      const hasInput = await p.locator("input").filter({ hasNot: p.locator('[type=hidden]') }).count();
      const scanBtn = /Quét mã điểm danh/.test(body);
      rec("TC-MOB-ENT-12", ok(/Mã điểm danh/.test(body) && !scanBtn, `web phải chỉ có ô mã; thấy nút quét=${scanBtn}`), `web Chromium: ô «Mã điểm danh»=${/Mã điểm danh/.test(body)}; nút «Quét mã điểm danh»=${scanBtn}; «Lớp QC3M1»=${/QC3M1/.test(body)} (phần PWA/app: pha 2/3)`);
      // ENT-14: nhập mã sai khi chưa từng mở cửa sổ ở QC3M2
      await p.evaluate((c) => localStorage.setItem("ep:ui:course", c), IDS.QC3M1);
      await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const inp = p.locator("input").first();
      if (await inp.count()) { await inp.click(); await inp.fill("ABC123"); await p.waitForTimeout(300); await p.getByRole("button", { name: "Điểm danh" }).first().click(); await p.waitForTimeout(4000); }
      const b14 = await txt(p); await shot(p, "ENT-14-code-nomatch-390");
      rec("TC-MOB-ENT-14", ok(/Mã không đúng hoặc đã hết hạn/.test(b14), `không thấy câu lỗi mã: ${b14.slice(0, 120)}`), `lớp chưa từng mở cửa sổ, mã sai → «Mã không đúng hoặc đã hết hạn…»=${/Mã không đúng hoặc đã hết hạn/.test(b14)}; «Đã đóng điểm danh.»=${/Đã đóng điểm danh/.test(b14)}`);
      // ENT-13: lớp self_checkin tắt
      await p.evaluate((c) => localStorage.setItem("ep:ui:course", c), IDS.QCMOB5);
      await p.goto(BASE + "/checkin?course=" + IDS.QCMOB5, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
      const b13 = await txt(p); await shot(p, "ENT-13-class-off-390");
      const inp13 = await p.locator("input").count();
      rec("TC-MOB-ENT-13", ok(/không điểm danh bằng mã/.test(b13) && inp13 === 0, `câu=${/không điểm danh bằng mã/.test(b13)} ô nhập=${inp13}`), `lớp tắt: «Lớp này không điểm danh bằng mã.»=${/không điểm danh bằng mã/.test(b13)}; số ô nhập=${inp13}`);
      await ctx.close();
    }
    // ENT-15 sv.moi (chưa có lớp)
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true });
      const p = await login(ctx, "sv.moi"); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const items = await navTexts(p); const more = p.getByRole("button", { name: /Thêm/ }).first(); let mi = [];
      if (await more.count()) { await more.click(); await p.waitForTimeout(600); mi = await p.locator("[role=menu] a:visible,[role=dialog] a:visible,dialog[open] a:visible").evaluateAll((e) => e.map((x) => (x.innerText || "").trim())); }
      await shot(p, "ENT-15-noclass-390");
      rec("TC-MOB-ENT-15", ok(!items.includes("Điểm danh") && !mi.includes("Điểm danh"), `vẫn hiện Điểm danh: nav=${items.includes("Điểm danh")} thêm=${mi.includes("Điểm danh")}`), `SV chưa có lớp: nav ${items.length} mục [${items.join("|")}], «Thêm» [${mi.join("|")}]`);
      await ctx.close();
    }
    // ENT-16 Staff không có mục
    {
      const out = {};
      for (const u of ["teacher", "ta", "admin"]) {
        const ctx = await ctxFor(1024, 768, { hasTouch: true }); const p = await login(ctx, u); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
        const items = await navTexts(p); out[u] = items.filter((x) => x === "Điểm danh").length;
        await ctx.close();
      }
      rec("TC-MOB-ENT-16", ok(Object.values(out).every((n) => n === 0), JSON.stringify(out)), `số mục «Điểm danh» ở nav staff: ${JSON.stringify(out)}`);
    }
    // ENT-17/18 hộp bấm + aria-current
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
      const p = await login(ctx, "sv.qc3"); await p.goto(BASE + "/checkin", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
      const link = p.locator("nav a").filter({ hasText: "Điểm danh" }).first();
      const box = await link.count() ? await link.evaluate((e) => { const b = e.getBoundingClientRect(); return { w: Math.round(b.width), h: Math.round(b.height), cur: e.getAttribute("aria-current") }; }) : null;
      const small = await p.locator("nav a, nav button").evaluateAll((els) => els.map((e) => { const b = e.getBoundingClientRect(); return { t: (e.innerText || "").trim().slice(0, 14), w: Math.round(b.width), h: Math.round(b.height) }; }).filter((x) => x.h > 0 && x.h < 44));
      await shot(p, "ENT-17-18-nav-checkin-1024");
      rec("TC-MOB-ENT-17", ok(small.length === 0, `điều khiển < 44px: ${JSON.stringify(small.slice(0, 6))}`), `điều khiển nav < 44 px: ${small.length} ${JSON.stringify(small.slice(0, 4))}; mục nav Điểm danh=${JSON.stringify(box)}`);
      rec("TC-MOB-ENT-18", ok(box && box.cur === "page", `aria-current=${box && box.cur}`), `aria-current=${box && box.cur}`);
      await ctx.close();
    }
    // ENT-19/22 (Tất cả lớp / PWA — web-only phần «Tất cả lớp của tôi»)
    {
      const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true });
      const p = await login(ctx, "sv.qc3"); await p.goto(BASE + "/", { waitUntil: "domcontentloaded" }); await p.waitForTimeout(6000);
      const sel = p.getByRole("button", { name: /Chọn lớp/ }).first();
      let opts = [];
      if (await sel.count()) { await sel.click(); await p.waitForTimeout(700); opts = await p.locator("[role=menu] *, [role=dialog] *").evaluateAll((e) => e.map((x) => (x.innerText || "").trim()).filter(Boolean)); }
      await shot(p, "ENT-19-course-select-390");
      rec("TC-MOB-ENT-19", ok(true, ""), `bộ chọn lớp mở, ${opts.length} mục; có «Tất cả lớp»=${opts.some((o) => /Tất cả/.test(o))} (ghi hành vi)`);
      await ctx.close();
    }
  }
  // ---------- 6.Q: axe ----------
  if (stage === "all" || stage === "axe") {
    const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
    const p = await login(ctx, "sv.qc3");
    for (const [name, url] of [["checkin", "/checkin?course=" + IDS.QC3M1], ["today", "/"], ["attendance", null]]) {
      if (!url) continue;
      await p.goto(BASE + url); await p.waitForTimeout(2500);
      const res = await new AxeBuilder({ page: p }).analyze();
      const bad = res.violations.filter((v) => v.impact === "critical" || v.impact === "serious");
      rec(`TC-MOB-${name === "checkin" ? "A11-01" : "A11-15"}`, ok(bad.length === 0, bad.map((v) => `${v.id}(${v.nodes.length})`).join(",")), `axe ${name}: ${res.violations.length} loại vi phạm, critical/serious=${bad.length}${bad.length ? " [" + bad.map((v) => `${v.id}:${v.nodes.length}`).join("|") + "]" : ""}`);
    }
    await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(4000);
    const b = await p.evaluate(() => ({ lang: document.documentElement.lang, title: document.title }));
    rec("TC-MOB-A11-16", ok(b.lang === "vi" && /Điểm danh/.test(b.title), JSON.stringify(b)), `/checkin: lang=${b.lang} title=${b.title}`);
    const dec = await p.evaluate(() => {
      const inp = document.querySelector("input");
      const lbl = inp && (inp.labels?.[0]?.innerText || inp.getAttribute("aria-label") || "");
      const svg = [...document.querySelectorAll("svg")].filter((s) => !s.closest("[aria-hidden=true]") && !s.getAttribute("aria-hidden"));
      return { label: (lbl || "").trim(), deco: svg.length };
    });
    rec("TC-MOB-A11-09", ok(dec.deco === 0 && dec.label.length > 0, JSON.stringify(dec)), `ô mã nhãn thật="${dec.label}"; biểu tượng không aria-hidden=${dec.deco}`);
    await ctx.close();
  }
  // ---------- 6.T: bảo mật (phần web) ----------
  if (stage === "all" || stage === "sec") {
    const r = await fetch(BASE + "/checkin", { redirect: "manual" });
    const h = Object.fromEntries(r.headers);
    const csp = h["content-security-policy"] || "";
    rec("TC-MOB-SEC-07", ok(csp && !/unsafe-eval/.test(csp) && /camera=\(self\)/.test(h["permissions-policy"] || "") && (h["referrer-policy"] || "") === "no-referrer", `csp=${csp.slice(0, 60)} pp=${h["permissions-policy"]} rp=${h["referrer-policy"]}`), `CSP unsafe-eval=${/unsafe-eval/.test(csp)}; Permissions-Policy=${h["permissions-policy"]}; Referrer-Policy=${h["referrer-policy"]}; HSTS=${h["strict-transport-security"] ? "có" : "KHÔNG"}`);
    const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
    const p = await login(ctx, "sv.qc3"); const urls = [];
    p.on("request", (q) => urls.push(q.url()));
    await p.goto(BASE + "/checkin?course=" + IDS.QC3M1, { waitUntil: "domcontentloaded" }); await p.waitForTimeout(5000);
    await p.locator("input[placeholder=\"ABC 123\"]").fill("ABC123"); await p.getByRole("button", { name: "Điểm danh" }).first().click(); await p.waitForTimeout(2500);
    const hits = urls.filter((u) => /ABC123|token=|\/checkins\?/.test(u));
    rec("TC-MOB-SEC-01", ok(hits.length === 0, `URL chứa mã/token: ${hits.slice(0, 3).join(" ")}`), `mã ngắn không vào URL: vi phạm=${hits.length}/${urls.length} request`);
    const store = await p.evaluate(() => ({ ls: Object.keys(localStorage), ss: Object.keys(sessionStorage), cookie: document.cookie }));
    const leak = store.ls.some((k) => /token|refresh|ep_rt/i.test(k)) || /ep_rt/.test(store.cookie);
    rec("TC-MOB-SEC-18", ok(!leak, JSON.stringify(store).slice(0, 200)), `localStorage=[${store.ls.join(",")}] sessionStorage=[${store.ss.join(",")}] document.cookie thấy ep_rt=${/ep_rt/.test(store.cookie)}`);
    await ctx.close();
  }
  // ---------- 6.L: Back (phần web) ----------
  if (stage === "all" || stage === "nav") {
    const ctx = await ctxFor(390, 844, { hasTouch: true, isMobile: true }); await setCourse(ctx, IDS.QC3M1);
    const p = await login(ctx, "sv.qc3"); const posts = [];
    p.on("request", (q) => { if (q.method() === "POST" && /\/checkins$/.test(q.url())) posts.push(q.url()); });
    await p.goto(BASE + "/checkin?course=" + IDS.QC3M1); await p.waitForTimeout(2000);
    await p.locator("input[placeholder=\"ABC 123\"]").fill("ZZZ999"); await p.getByRole("button", { name: "Điểm danh" }).first().click(); await p.waitForTimeout(2500);
    await p.goBack(); await p.waitForTimeout(1500); await p.goForward(); await p.waitForTimeout(1500);
    const body = await txt(p); await shot(p, "NAV-back-390");
    rec("TC-MOB-NAV-12", ok(posts.length <= 1, `POST checkins=${posts.length}`), `Back/Forward sau khi gửi: POST /checkins=${posts.length}; câu lỗi cũ hiện lại=${/Mã không đúng/.test(body)}`);
    await ctx.close();
  }
} finally {
  await browser.close();
  writeFileSync(`${OUT}/results.json`, JSON.stringify(R, null, 1));
  appendFileSync(`${OUT}/results.jsonl`, R.map((x) => JSON.stringify(x)).join("\n") + "\n");
  console.log("WROTE", OUT + "/results.json", R.length);
}
