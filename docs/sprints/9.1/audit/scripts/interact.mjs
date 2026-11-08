// Sprint 9.1 pha 1 — hộp thoại / drawer / menu / focus bàn phím. CHỈ MỞ rồi đóng; không bấm nút có tác dụng ghi.
// Mọi request không phải GET (trừ /auth/) được ghi lại để chứng minh demo không bị đổi: data/interact.json → writes.
//   QC_PW_FILE=... node interact.mjs
import { chromium } from "/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright-core/index.mjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SHOTS = path.join(HERE, "../shots");
const DATA = path.join(HERE, "../data");
const B = process.env.DEMO_URL ?? "https://v2202609419695521098.happysrv.de";
const PW = fs.readFileSync(process.env.QC_PW_FILE ?? "/tmp/qcaudit/pw.txt", "utf8").trim();
const AXE = fs.readFileSync(process.env.AXE_JS ?? "/tmp/qcaudit/axe.min.js", "utf8");
const TAGS = ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"];
const C1 = "01a19813-a273-7dc9-99a5-a518f46f98ea";
const SUB = "01a19817-4b65-7416-a4ac-55a834f6662f";
const USERS = { teacher: "teacher", admin: "admin", student: "sv.gioi" };
const OV = () => { const vis = (e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && getComputedStyle(e).visibility !== "hidden"; }; return { dialog: [...document.querySelectorAll("[role=dialog],[role=alertdialog],dialog[open]")].filter(vis).length, menu: [...document.querySelectorAll("[role=menu],[role=listbox],[data-radix-popper-content-wrapper]")].filter(vis).length, details: document.querySelectorAll("details[open]").length, focus: document.activeElement ? `${document.activeElement.tagName.toLowerCase()}${document.activeElement.getAttribute("aria-label") ? "[" + document.activeElement.getAttribute("aria-label").slice(0, 30) + "]" : ""}` : null }; };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const btn = (page, name) => page.getByRole("button", { name }).first();
const first = (sel) => (page) => page.locator(sel).first();

// [vai, route, rộng, tên, hàm mở (page) → locator hoặc thực thi]
const ACTIONS = [
  ["teacher", "/", 1440, "bell", (p) => p.getByRole("button", { name: /Thông báo/ }).first().click()],
  ["teacher", "/", 1440, "usermenu", (p) => p.getByRole("button", { name: /^Tài khoản/ }).first().click()],
  ["teacher", "/", 1440, "courseswitch", (p) => p.locator("header button").first().click()],
  ["teacher", "/", 1440, "cmdk", (p) => p.keyboard.press("Meta+k")],
  ["teacher", "/inbox", 1440, "ticket", (p) => p.locator("main button").filter({ hasText: /ngày/ }).first().click()],
  ["teacher", "/inbox", 375, "ticket", (p) => p.locator("main button").filter({ hasText: /ngày/ }).first().click()],
  ["teacher", "/gradebook", 1440, "chotdiem", (p) => btn(p, "Chốt điểm").click()],
  ["teacher", "/gradebook", 1440, "themhanhdong", (p) => btn(p, "Thêm hành động").click()],
  ["teacher", "/gradebook", 1440, "hanghd", (p) => p.getByRole("button", { name: /^Hành động — / }).first().click()],
  ["teacher", "/grading", 1440, "chamtatca-khoa", (p) => p.getByRole("button", { name: "Chấm tất cả" }).hover({ force: true })],
  ["teacher", "/questions", 1440, "taocauhoi", (p) => btn(p, "Tạo câu hỏi").click()],
  ["teacher", "/threads", 1440, "datcauhoi", (p) => btn(p, "Đặt câu hỏi").click()],
  ["teacher", "/documents", 1440, "taitailieu", (p) => btn(p, "Tải tài liệu lên").click()],
  ["teacher", "/documents", 1440, "thaotac", (p) => p.getByRole("button", { name: /^Thao tác với / }).first().click()],
  ["teacher", "/calendar", 1440, "themsukien", (p) => btn(p, "Thêm sự kiện").click()],
  ["teacher", "/calendar", 1440, "taolichbuoihoc", (p) => btn(p, "Tạo lịch buổi học").click()],
  ["teacher", "/class/members", 1440, "nhapds", (p) => p.getByRole("tab", { name: "Nhập danh sách" }).click()],
  ["teacher", "/class/members", 1440, "thaotac", (p) => p.getByRole("button", { name: /^Thao tác cho / }).first().click()],
  ["teacher", "/class/settings", 1440, "taolaima", (p) => btn(p, "Tạo lại mã").click()],
  ["teacher", "/insights", 1440, "cauhoiandanh", (p) => p.locator("summary").first().click()],
  ["teacher", "/settings/llm", 1440, "nangcao", (p) => btn(p, "Cài đặt nâng cao").click()],
  ["teacher", "/settings", 1440, "thietbi", (p) => p.getByRole("button", { name: /^Thao tác cho / }).first().click()],
  ["teacher", `/grading/${SUB}`, 1440, "trichdoan", (p) => p.locator("main button").filter({ hasText: /Đoạn trích/ }).first().click()],
  ["admin", "/admin/users", 1440, "moigv", (p) => btn(p, "Mời giảng viên").click()],
  ["admin", "/admin/users", 1440, "thaotac", (p) => p.getByRole("button", { name: /^Thao tác cho / }).first().click()],
  ["admin", "/admin/courses", 1440, "molop", (p) => btn(p, "Mở lớp").click()],
  ["admin", "/admin/courses", 1440, "hanhdong", (p) => p.getByRole("button", { name: /^Hành động cho lớp/ }).first().click()],
  ["admin", "/settings/llm", 1440, "themncc", (p) => btn(p, "Thêm nhà cung cấp").click()],
  ["admin", "/settings/llm", 1440, "nangcao", (p) => btn(p, "Cài đặt nâng cao").click()],
  ["student", "/", 1440, "bell", (p) => p.getByRole("button", { name: /Thông báo/ }).first().click()],
  ["student", "/", 1440, "usermenu", (p) => p.getByRole("button", { name: /^Tài khoản/ }).first().click()],
  ["student", "/", 1440, "courseswitch", (p) => p.locator("header button").first().click()],
  ["student", "/", 375, "them", (p) => p.getByRole("button", { name: /^Thêm$/ }).first().click()],
  ["student", "/threads", 1440, "datcauhoi", (p) => btn(p, "Đặt câu hỏi").click()],
  ["student", "/threads", 375, "datcauhoi", (p) => btn(p, "Đặt câu hỏi").click()],
  ["student", "/library", 1440, "them", (p) => p.getByRole("button", { name: /^Thêm với / }).first().click()],
  ["student", "/calendar", 1440, "themvaolich", (p) => btn(p, "Thêm vào lịch").click()],
  ["student", "/chat", 1440, "soanthao", (p) => p.locator("textarea").first().focus()],
  ["student", "/chat", 375, "soanthao", (p) => p.locator("textarea").first().focus()],
];
// Focus bàn phím: nhấn Tab N lần, ghi từng điểm dừng, chụp khung nhìn.
const FOCUS = [["teacher", "/", 1440], ["teacher", "/inbox", 1440], ["teacher", "/gradebook", 1440], ["student", "/chat", 1440], ["student", "/", 375]];

const slug = (p) => (p === "/" ? "home" : p.replace(new RegExp(`${C1}|${SUB}`, "g"), "ID").replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-|-$/g, ""));
const out = { writes: [], actions: [], focus: [] };
const browser = await chromium.launch();
const ctxs = {};
async function ctxFor(role, width) {
  const key = role + width;
  if (ctxs[key]) return ctxs[key];
  const ctx = await browser.newContext({ viewport: { width, height: width > 700 ? 900 : 812 }, bypassCSP: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  ctx.setDefaultTimeout(8000);
  ctx.setDefaultNavigationTimeout(30000);
  ctx.on("request", (r) => { const m = r.method(); if (!["GET", "HEAD", "OPTIONS"].includes(m) && !/\/auth\//.test(r.url())) out.writes.push(`${role} ${m} ${r.url().replace(B, "")}`); });
  const p = await ctx.newPage();
  await p.goto(B + "/login");
  await p.fill("input[type=email]", USERS[role] + "@edupilot.local");
  await p.fill("input[type=password]", PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 });
  await sleep(1500);
  await p.close();
  return (ctxs[key] = ctx);
}
async function open(ctx, route, width) {
  const page = await ctx.newPage();
  const consoleLog = [];
  page.on("console", (m) => { if (["error", "warning"].includes(m.type())) consoleLog.push(`${m.type()}: ${m.text().slice(0, 200)}`); });
  page.on("pageerror", (e) => consoleLog.push(`pageerror: ${String(e.message).slice(0, 200)}`));
  await page.goto(B + route + (route.includes("?") ? "&" : "?") + "course=" + C1, { waitUntil: "domcontentloaded" });
  await page.waitForFunction(() => ((document.querySelector("main") ?? document.body).innerText ?? "").trim().length > 20, null, { timeout: 25000 }).catch(() => {});
  await page.waitForLoadState("networkidle", { timeout: 3000 }).catch(() => {});
  await page.evaluate(() => document.fonts.ready);
  await sleep(800);
  return { page, consoleLog };
}

for (const [role, route, width, name, act] of ACTIONS) {
  const rec = { role, route, width, name, opened: false, file: `${slug(route)}@dlg-${name}__${role}__${width}.png` };
  let page;
  try {
    const ctx = await ctxFor(role, width);
    const o = await open(ctx, route, width);
    page = o.page;
    const before = await page.evaluate(OV);
    await act(page);
    await sleep(60);
    // chuyển động lúc mở (luật mới: ứng dụng chỉ chuyển động chức năng ≤ 200 ms)
    rec.motion = await page.evaluate(() => document.getAnimations().map((a) => { const t = a.effect?.getTiming?.(); return `${a.transitionProperty ?? a.animationName ?? "?"}:${Math.round((t?.duration ?? 0) + (t?.delay ?? 0))}ms`; }).slice(0, 8));
    await sleep(840);
    rec.overlay = await page.evaluate(OV);
    const tot = (o) => o.dialog + o.menu + o.details;
    rec.opened = tot(rec.overlay) > tot(before) || (name === "soanthao" && /textarea/.test(rec.overlay.focus ?? "")) || name.endsWith("-khoa");
    await page.evaluate(AXE);
    const ax = await page.evaluate((tags) => axe.run(document, { runOnly: { type: "tag", values: tags }, resultTypes: ["violations"] }), TAGS);
    rec.axe = { critical: 0, serious: 0, moderate: 0, minor: 0, items: ax.violations.map((v) => { return `${v.id} (${v.impact}) ×${v.nodes.length}: ${(v.nodes[0]?.target ?? []).join(" ").slice(0, 80)}`; }) };
    for (const v of ax.violations) rec.axe[v.impact] += v.nodes.length;
    await page.screenshot({ path: path.join(SHOTS, rec.file) });
    // Escape đóng được không (đồng thời là kiểm tra bẫy focus / đóng bằng bàn phím)
    await page.keyboard.press("Escape");
    await sleep(500);
    rec.closedByEscape = tot(await page.evaluate(OV)) <= tot(before);
    rec.console = o.consoleLog;
  } catch (e) {
    rec.error = String(e.message).split("\n")[0].slice(0, 160);
  }
  out.actions.push(rec);
  console.log(`${role} ${width} ${route} ${name}: ${rec.opened ? "mở" : "KHÔNG mở"}${rec.error ? " ERR " + rec.error : ""} axe=${rec.axe ? rec.axe.critical + "/" + rec.axe.serious + "/" + rec.axe.moderate + "/" + rec.axe.minor : "-"} esc=${rec.closedByEscape}`);
  await page?.close();
  fs.writeFileSync(path.join(DATA, "interact.json"), JSON.stringify(out, null, 1));
}

for (const [role, route, width] of FOCUS) {
  const rec = { role, route, width, stops: [], file: `${slug(route)}@focus__${role}__${width}.png` };
  try {
    const ctx = await ctxFor(role, width);
    const { page } = await open(ctx, route, width);
    await page.evaluate(() => document.body.focus());
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press("Tab");
      await sleep(120);
      rec.stops.push(await page.evaluate(() => {
        const e = document.activeElement;
        if (!e || e === document.body) return "(body)";
        const cs = getComputedStyle(e);
        const ring = cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0 ? "outline" : cs.boxShadow !== "none" ? "shadow" : "KHÔNG";
        const name = (e.getAttribute("aria-label") || e.innerText || e.getAttribute("placeholder") || "").trim().replace(/\s+/g, " ").slice(0, 30);
        const r = e.getBoundingClientRect();
        return `${e.tagName.toLowerCase()} «${name}» vòng=${ring} ${Math.round(r.top)}${r.top < 0 || r.top > innerHeight ? " NGOÀI KHUNG" : ""}`;
      }));
      if (i === 5) await page.screenshot({ path: path.join(SHOTS, rec.file) });
    }
    await page.close();
  } catch (e) { rec.error = String(e.message).split("\n")[0].slice(0, 160); }
  out.focus.push(rec);
  console.log(`focus ${role} ${route} ${width}: ${rec.stops.length} điểm dừng${rec.error ? " ERR " + rec.error : ""}`);
  fs.writeFileSync(path.join(DATA, "interact.json"), JSON.stringify(out, null, 1));
}
await browser.close();
console.log("ghi (không phải GET):", out.writes.length ? out.writes.join(" | ") : "không có");
