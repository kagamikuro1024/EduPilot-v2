// Sprint 9.1 pha 1 — thu bằng chứng UI/UX trên bản demo. Chỉ ĐỌC dữ liệu demo (không bấm Lưu / Công bố / Chốt / Xoá).
// Chạy trên Mac, chỉ trình duyệt (CONTEXT.md mục 4). Dùng playwright-core đi kèm playwright-cli.
//   QC_PW_FILE=<tệp chứa mật khẩu seed của demo> node shoot.mjs <vai> [<vai>...]     vai: anon teacher ta admin student student2 newbie
// Kết quả: ../shots/*.png, ../data/<vai>.json
import { chromium } from "/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright-core/index.mjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SHOTS = path.join(HERE, "../shots");
const DATA = path.join(HERE, "../data");
fs.mkdirSync(SHOTS, { recursive: true });
fs.mkdirSync(DATA, { recursive: true });
const B = process.env.DEMO_URL ?? "https://v2202609419695521098.happysrv.de";
const PW = fs.readFileSync(process.env.QC_PW_FILE ?? "/tmp/qcaudit/pw.txt", "utf8").trim();
const AXE = fs.readFileSync(process.env.AXE_JS ?? "/tmp/qcaudit/axe.min.js", "utf8");
const TAGS = ["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"];

// Id của dữ liệu seed trên demo (lấy bằng API lúc khảo sát 2026-11-08).
const C1 = "01a19813-a273-7dc9-99a5-a518f46f98ea"; // 761987 — lớp đầy dữ liệu
const C2 = "01a19813-a284-74ae-817d-0543343cf0ab"; // 761988 — lớp ít dữ liệu
const STU = "01a19813-a4c6-79c2-9d8b-378897e75558";
const EX_A = "01a19814-4fc8-7766-97aa-634dba51f72f";
const EX_B = "01a19814-4fa6-7f52-8bdb-d8cd4c788342";
const THREAD = "01a19817-063f-746d-8b89-ca8f4bccdf4b";
const ASSIGN = "01a19817-470f-73c3-8f6a-911f3a4dc51e";
const LIB = "01a19817-2ea3-7496-9077-0423f2abcbbe";
const SUB = "01a19817-4b65-7416-a4ac-55a834f6662f";

const USERS = { teacher: "teacher", ta: "ta", admin: "admin", student: "sv.gioi", student2: "sv.kha", newbie: "sv.moi" };

const STAFF = ["/", "/inbox", "/students", "/attendance", "/gradebook", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/gradebook/scheme", "/settings"];
const STAFF_DETAIL = [`/students/${STU}`, `/exams/${EX_A}`, `/exams/${EX_A}/results`, `/exams/${EX_A}/similarity`, `/exams/${EX_B}`, `/grading/${SUB}`, `/threads/${THREAD}`, `/assignments/${ASSIGN}`, "/library", `/library/${LIB}`, "/inbox?tab=resolved", "/class/members?tab=pending", "/threads?view=moderation", "/questions?review_status=PENDING", "/khong-ton-tai"];
const SYSTEM = ["/observability", "/settings/llm", "/settings/integrations"];
const STUDENT = ["/", "/chat", "/threads", "/practice", "/exams", "/library", "/calendar", "/me", "/practice/history"];
const STUDENT_DETAIL = [`/threads/${THREAD}`, `/library/${LIB}`, `/assignments/${ASSIGN}`, "/practice/at-symmetric", "/join", "/join/AN7K2MQ", `/exams/${EX_A}`, `/exams/${EX_A}/results`, "/gradebook", "/inbox", "/admin/users", "/khong-ton-tai"]; // 3 mục cuối = vai sai → màn chặn quyền

const PLAN = {
  anon: { login: null, items: ["/", "/login", "/register", "/forgot-password", "/reset-password", "/verify-email", "/invite/bogus-token-000", "/khong-ton-tai"].map((p) => ({ p, w: [1440, 375] })) },
  teacher: {
    login: "teacher",
    items: [...STAFF, ...STAFF_DETAIL, ...SYSTEM].map((p) => ({ p, c: C1, w: ["/attendance", "/inbox", "/"].includes(p) ? [1440, 375] : [1440] })),
    empty: [...STAFF, ...SYSTEM],
  },
  ta: { login: "ta", items: [...STAFF, ...STAFF_DETAIL].map((p) => ({ p, c: C1, w: ["/attendance", "/inbox", "/"].includes(p) ? [1440, 375] : [1440] })) },
  admin: { login: "admin", items: ["/", "/observability", "/admin/courses", "/admin/users", "/settings/llm", "/settings/integrations", "/settings", "/class/members", "/gradebook"].map((p) => ({ p, w: [1440] })) },
  student: { login: "student", items: [...STUDENT, ...STUDENT_DETAIL].map((p) => ({ p, c: C1, w: [1440, 375] })), empty: STUDENT },
  student2: { login: "student2", items: ["/", "/chat", "/me", "/exams", "/threads", "/practice"].map((p) => ({ p, c: C1, w: [1440] })) },
  newbie: { login: "newbie", items: ["/", "/join", "/chat", "/library", "/me", "/exams", "/calendar"].map((p) => ({ p, w: [1440, 375] })) },
};

const slugOf = (p) => (p === "/" ? "home" : p.replace(/\?/g, "-").replace(/=/g, "-").replace(new RegExp(`${C1}|${STU}|${EX_A}|${EX_B}|${THREAD}|${ASSIGN}|${LIB}|${SUB}`, "g"), "ID").replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-|-$/g, ""));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const withCourse = (p, c) => (c ? p + (p.includes("?") ? "&" : "?") + "course=" + c : p);

const INIT = () => {
  window.__m = { lcp: 0, fcp: 0 };
  try {
    new PerformanceObserver((l) => { for (const e of l.getEntries()) window.__m.lcp = Math.round(e.startTime); }).observe({ type: "largest-contentful-paint", buffered: true });
    new PerformanceObserver((l) => { for (const e of l.getEntries()) if (e.name === "first-contentful-paint") window.__m.fcp = Math.round(e.startTime); }).observe({ type: "paint", buffered: true });
  } catch {}
};

const INSPECT = (techWords) => {
  const de = document.documentElement;
  const W = de.clientWidth;
  const main = document.querySelector("main") ?? document.body;
  const text = main.innerText ?? "";
  const bad = ["undefined", "null", "NaN", "[object", "Invalid Date"].filter((w) => new RegExp(`(^|[^A-Za-z])${w.replace(/[[\]]/g, "\\$&")}([^A-Za-z]|$)`).test(text));
  const tech = techWords ? techWords.filter((w) => new RegExp(`\\b${w}\\b`, "i").test(text)) : [];
  const off = [];
  if (de.scrollWidth > W + 1) {
    for (const e of document.body.querySelectorAll("*")) {
      const r = e.getBoundingClientRect();
      if (r.width > 0 && r.height > 0 && r.right > W + 1 && getComputedStyle(e).position !== "fixed") {
        let clipped = false;
        for (let a = e.parentElement; a && a !== document.body; a = a.parentElement) {
          const o = getComputedStyle(a).overflowX;
          if (o === "auto" || o === "scroll" || o === "hidden") { clipped = true; break; }
        }
        if (!clipped) { off.push(`${e.tagName.toLowerCase()}.${String(e.className).split(" ")[0].slice(0, 30)} phải=${Math.round(r.right)}`); if (off.length >= 4) break; }
      }
    }
  }
  const innerScroll = [...document.querySelectorAll("*")].filter((e) => { const o = getComputedStyle(e).overflowX; return (o === "auto" || o === "scroll") && e.scrollWidth > e.clientWidth + 2 && e.clientWidth > 0; }).length;
  const ellipsis = [...main.querySelectorAll("*")].filter((e) => { const s = getComputedStyle(e); return s.textOverflow === "ellipsis" && e.scrollWidth > e.clientWidth + 1; }).map((e) => (e.innerText ?? "").slice(0, 40)).slice(0, 3);
  const vis = (e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && getComputedStyle(e).visibility !== "hidden"; };
  const noName = [...document.querySelectorAll("button, a[href], [role=button], input, select, textarea")].filter((e) => vis(e) && !(e.getAttribute("aria-label") || e.getAttribute("aria-labelledby") || e.getAttribute("title") || (e.innerText ?? "").trim() || (e.labels && e.labels.length) || e.getAttribute("placeholder") || e.type === "hidden")).map((e) => `${e.tagName.toLowerCase()}.${String(e.className).split(" ")[0].slice(0, 30)}`).slice(0, 4);
  const small = [...document.querySelectorAll("button, a[href], [role=button], [role=tab], input:not([type=hidden]), select")].filter((e) => { if (!vis(e)) return false; const r = e.getBoundingClientRect(); return r.height < 32 || r.width < 32; });
  const smallTxt = small.filter((e) => { const r = e.getBoundingClientRect(); return r.height < 24 || r.width < 24; }).length;
  return {
    title: document.title, lang: de.lang, h1: document.querySelectorAll("h1").length, mainChars: text.trim().length,
    bad, tech, off, innerScroll, ellipsis, noName, small32: small.length, small24: smallTxt,
    sw: de.scrollWidth, W, docH: de.scrollHeight, anims: document.getAnimations().length,
    lcp: window.__m?.lcp ?? 0, fcp: window.__m?.fcp ?? 0, longest: null,
  };
};

async function uiLogin(ctx, key) {
  const p = await ctx.newPage();
  await p.goto(B + "/login");
  await p.fill("input[type=email]", USERS[key] + "@edupilot.local");
  await p.fill("input[type=password]", PW);
  await p.keyboard.press("Enter");
  await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 });
  await p.waitForTimeout(2000);
  await p.close();
}
async function login(browser, key) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, bypassCSP: true, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  if (key) await uiLogin(ctx, key);
  return ctx;
}

let results = []; // gộp với lần chạy trước (chụp lại một phần)
async function visit(ctx, role, it, width, state, retry = 1) {
  const url = withCourse(it.p, state === "empty" ? C2 : it.c);
  const slug = slugOf(it.p) + (state === "full" ? "" : "@" + state);
  const file = `${slug}__${role}__${width}.png`;
  if (!process.env.REDO && results.some((r) => r.role === role && r.file === file && !r.fail)) return; // chạy tiếp sau khi trình duyệt chết
  const page = await ctx.newPage();
  const rec = { role, route: it.p, course: state === "empty" ? "761988" : it.c ? "761987" : "-", width, state, file, console: [], http: [], slowApi: [], url: null };
  await page.addInitScript(INIT);
  await page.setViewportSize({ width, height: width > 700 ? 900 : 812 });
  page.on("console", (m) => { if (["error", "warning"].includes(m.type())) rec.console.push(`${m.type()}: ${m.text().slice(0, 220)}`); });
  page.on("pageerror", (e) => rec.console.push(`pageerror: ${String(e.message).slice(0, 220)}`));
  page.on("response", (r) => { if (r.status() >= 400) rec.http.push(`${r.status()} ${r.request().method()} ${r.url().replace(B, "").slice(0, 110)}`); });
  page.on("requestfinished", (r) => { const t = r.timing(); const d = t.responseEnd - t.startTime; if (r.url().includes("/api/") && d > 800) rec.slowApi.push(`${Math.round(d)}ms ${r.method()} ${r.url().replace(B, "").slice(0, 90)}`); });
  if (state === "loading") await page.route("**/api/v1/**", async (r) => { if (/\/auth\//.test(r.request().url())) return r.continue(); await sleep(8000); r.continue().catch(() => {}); });
  if (state === "error") await page.route("**/api/v1/**", (r) => (/\/auth\//.test(r.request().url()) ? r.continue() : r.abort("internetdisconnected")));
  const t0 = Date.now();
  try {
    await page.goto(B + url, { waitUntil: "domcontentloaded", timeout: 45000 });
    if (state === "loading") {
      await sleep(1800);
      rec.url = page.url().replace(B, "");
      await page.screenshot({ path: path.join(SHOTS, file) });
    } else {
      await page.waitForFunction(() => {
        const m = document.querySelector("main") ?? document.body;
        if (m.querySelector('[aria-busy="true"], [data-skeleton], [class*="keleton"]')) return false;
        return (m.innerText ?? "").trim().length > 20;
      }, null, { timeout: 25000, polling: 50 }).then(() => { rec.contentMs = Date.now() - t0; }).catch(() => { rec.contentMs = null; });
      await page.waitForLoadState("networkidle", { timeout: 3000 }).catch(() => {});
      await page.evaluate(() => document.fonts.ready);
      await sleep(700);
      rec.url = page.url().replace(B, "");
      Object.assign(rec, await page.evaluate(INSPECT, role.startsWith("student") || role === "newbie" ? ["RAG", "PII", "provider", "fallback", "trace", "confidence", "token", "LLM", "embedding", "prompt", "model", "AI"] : null));
      try {
        await page.evaluate(AXE);
        const ax = await page.evaluate((tags) => axe.run(document, { runOnly: { type: "tag", values: tags }, resultTypes: ["violations"] }), TAGS);
        rec.axe = { critical: 0, serious: 0, moderate: 0, minor: 0, items: [] };
        for (const v of ax.violations) { rec.axe[v.impact] += v.nodes.length; rec.axe.items.push(`${v.id} (${v.impact}) ×${v.nodes.length}: ${(v.nodes[0]?.target ?? []).join(" ").slice(0, 80)}`); }
      } catch (e) { rec.axe = { error: String(e.message).slice(0, 100) }; }
      await page.evaluate(async () => { for (let y = 0; y < Math.min(document.documentElement.scrollHeight, 9000); y += 500) { window.scrollTo(0, y); await new Promise((r) => setTimeout(r, 120)); } window.scrollTo(0, 0); }); // kích ảnh lazy / hiệu ứng khi cuộn
      await sleep(500);
      // Không dùng fullPage của Playwright: nó đôi khi vẽ lệch nội dung ~24 px sang trái (DOM vẫn đúng, đã đối chiếu). Thay bằng kéo khung nhìn tới hết chiều cao trang.
      const dh = await page.evaluate(() => document.documentElement.scrollHeight);
      if (dh > (width > 700 ? 900 : 812)) { await page.setViewportSize({ width, height: Math.min(dh, 9000) }); await sleep(600); }
      await page.screenshot({ path: path.join(SHOTS, file) });
    }
  } catch (e) {
    rec.fail = String(e.message).slice(0, 200);
  }
  rec.totalMs = Date.now() - t0;
  const lost = role !== "anon" && /^\/login/.test(rec.url ?? "") && !/^\/(login|khong-ton-tai)/.test(it.p) && state !== "error";
  if (lost && retry > 0) { // phiên bị đá ra (làm mới token đụng nhau): đăng nhập lại rồi chụp lại
    console.log(`${role} ${it.p} bị đá ra /login — đăng nhập lại`);
    await page.close();
    await uiLogin(ctx, PLAN[role].login);
    return visit(ctx, role, it, width, state, retry - 1);
  }
  const i = results.findIndex((r) => r.role === role && r.file === file);
  if (i >= 0) results[i] = rec; else results.push(rec);
  console.log(`${role} ${width} ${state} ${it.p} → ${rec.url ?? "?"} ${rec.fail ? "FAIL " + rec.fail : ""} content=${rec.contentMs}ms axe=${rec.axe ? JSON.stringify({ c: rec.axe.critical, s: rec.axe.serious, m: rec.axe.moderate, n: rec.axe.minor }) : "-"}`);
  await page.close();
  fs.writeFileSync(path.join(DATA, `${role}.json`), JSON.stringify(results, null, 1));
}

const browser = await chromium.launch();
const ONLY = process.env.ONLY ? new RegExp(process.env.ONLY) : null; // chụp lại một phần: ONLY='^/$' STATES=full
const STATES = (process.env.STATES ?? "full,loading,error,empty").split(",");
for (const role of process.argv.slice(2)) {
  const plan = PLAN[role];
  try { results = JSON.parse(fs.readFileSync(path.join(DATA, `${role}.json`), "utf8")); } catch { results = []; }
  const ctx = await login(browser, plan.login);
  const items = plan.items.filter((it) => !ONLY || ONLY.test(it.p));
  if (STATES.includes("full")) for (const it of items) for (const w of it.w) await visit(ctx, role, it, w, "full");
  if (STATES.includes("loading")) for (const it of items) await visit(ctx, role, it, 1440, "loading");
  if (STATES.includes("error")) for (const it of items) await visit(ctx, role, it, 1440, "error");
  if (STATES.includes("empty")) for (const p of (plan.empty ?? []).filter((p) => !ONLY || ONLY.test(p))) await visit(ctx, role, { p, c: C2, w: [1440] }, 1440, "empty");
  await ctx.close();
}
await browser.close();
