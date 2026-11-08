// Sprint 9.1 pha 1 — hai đo bổ sung (chỉ đọc):
//  (1) lề trái của tiêu đề trang so với mép sidebar ở mọi route staff (nghi ngờ vài trang sát mép sidebar)
//  (2) trạng thái lỗi SAU 15 s (lúc chụp @error ở sweep mới ~2 s, TanStack còn đang thử lại)
//   QC_PW_FILE=... node extra.mjs
import { chromium } from "/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright-core/index.mjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SHOTS = path.join(HERE, "../shots");
const DATA = path.join(HERE, "../data");
const B = process.env.DEMO_URL ?? "https://v2202609419695521098.happysrv.de";
const PW = fs.readFileSync(process.env.QC_PW_FILE ?? "/tmp/qcaudit/pw.txt", "utf8").trim();
const C1 = "01a19813-a273-7dc9-99a5-a518f46f98ea";
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const browser = await chromium.launch();
async function ctxFor(user) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, bypassCSP: true, locale: "vi-VN" });
  const p = await ctx.newPage();
  await p.goto(B + "/login");
  await p.fill("input[type=email]", user + "@edupilot.local");
  await p.fill("input[type=password]", PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 });
  await sleep(1500);
  await p.close();
  return ctx;
}
const out = { layout: [], lateError: [] };
const ROUTES = {
  teacher: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/observability", "/settings/llm", "/settings/integrations", "/settings"],
  student: ["/", "/chat", "/threads", "/practice", "/exams", "/library", "/calendar", "/me"],
  admin: ["/", "/observability", "/admin/courses", "/admin/users", "/settings/llm", "/settings/integrations"],
};
for (const [role, user] of [["teacher", "teacher"], ["student", "sv.gioi"], ["admin", "admin"]]) {
  const ctx = await ctxFor(user);
  for (const r of ROUTES[role]) {
    for (let rep = 0; rep < 2; rep++) {
      const p = await ctx.newPage();
      await p.goto(B + r + (r.includes("?") ? "&" : "?") + "course=" + C1, { waitUntil: "domcontentloaded" });
      await p.waitForFunction(() => ((document.querySelector("main") ?? document.body).innerText ?? "").trim().length > 20, null, { timeout: 25000 }).catch(() => {});
      await p.waitForLoadState("networkidle", { timeout: 3000 }).catch(() => {});
      await sleep(900);
      const m = await p.evaluate(() => {
        const nav = document.querySelector("nav")?.getBoundingClientRect();
        const main = document.querySelector("main");
        const h = main?.querySelector("h1, h2")?.getBoundingClientRect();
        const cs = main ? getComputedStyle(main) : null;
        return { navRight: nav ? Math.round(nav.right) : null, mainLeft: main ? Math.round(main.getBoundingClientRect().left) : null, headingLeft: h ? Math.round(h.left) : null, mainPadLeft: cs?.paddingLeft, mainWidth: main ? Math.round(main.getBoundingClientRect().width) : null };
      });
      out.layout.push({ role, route: r, rep, ...m });
      await p.close();
    }
    const l = out.layout.filter((x) => x.role === role && x.route === r);
    console.log(role, r, l.map((x) => `h=${x.headingLeft} navR=${x.navRight} pad=${x.mainPadLeft}`).join(" | "));
    fs.writeFileSync(path.join(DATA, "extra-layout.json"), JSON.stringify(out, null, 1));
  }
  await ctx.close();
}
// (2) lỗi muộn
for (const [role, user, route] of [["student", "sv.gioi", "/"], ["teacher", "teacher", "/"], ["teacher", "teacher", "/gradebook"], ["student", "sv.gioi", "/me"]]) {
  const ctx = await ctxFor(user);
  const p = await ctx.newPage();
  await p.route("**/api/v1/**", (r) => (/\/auth\//.test(r.request().url()) ? r.continue() : r.abort("internetdisconnected")));
  await p.goto(B + route + "?course=" + C1, { waitUntil: "domcontentloaded" });
  await sleep(15000);
  const slug = route === "/" ? "home" : route.replace(/[^a-z]+/gi, "-").replace(/^-|-$/g, "");
  const file = `${slug}@error-late__${role}__1440.png`;
  await p.screenshot({ path: path.join(SHOTS, file), fullPage: true });
  const txt = await p.evaluate(() => (document.querySelector("main") ?? document.body).innerText.replace(/\s+/g, " ").slice(0, 300));
  const hasRetry = await p.evaluate(() => [...document.querySelectorAll("button")].some((b) => /thử lại|tải lại/i.test(b.innerText)));
  out.lateError.push({ role, route, file, text: txt, hasRetryButton: hasRetry });
  console.log("lỗi muộn", role, route, "nút thử lại:", hasRetry, "|", txt.slice(0, 120));
  await ctx.close();
  fs.writeFileSync(path.join(DATA, "extra-layout.json"), JSON.stringify(out, null, 1));
}
await browser.close();
