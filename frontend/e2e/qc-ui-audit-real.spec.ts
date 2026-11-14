import { test, type Browser, type Page } from "@playwright/test";
import { mkdirSync, appendFileSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 (UI-R, bước 1): chụp MỌI route × vai × 1440/375 × (có dữ liệu | rỗng) và đo DOM để so chéo khung trang. CHỈ ĐỌC.
// QC_CLASS = 761987 (có dữ liệu) | QCEMPTY (rỗng thật); QC_STATE = data | empty; QC_DYN = JSON {assignment,exam,thread,doc,student,submission}
const PW = process.env.QC_PW ?? "";
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-audit";
const CLASS = process.env.QC_CLASS ?? "761987";
const STATE = process.env.QC_STATE ?? "data";
const ONLY = process.env.QC_ROLE ?? "";
const DYN = JSON.parse(process.env.QC_DYN ?? "{}") as Record<string, string>;
mkdirSync(OUT, { recursive: true });
const LOG = `${OUT}/audit-${STATE}.jsonl`;

const dyn = (p: string, k: string) => (DYN[k] ? [p.replace("{id}", DYN[k])] : []);
const ROUTES: Record<string, string[]> = {
  student: ["/", "/chat", "/threads", "/practice", "/practice/history", "/exams", "/library", "/calendar", "/me", "/settings", ...dyn("/assignments/{id}", "assignment"), ...dyn("/threads/{id}", "thread"), ...dyn("/library/{id}", "doc")],
  ta: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/settings", ...dyn("/students/{id}", "student"), ...dyn("/exams/{id}", "exam"), ...dyn("/grading/{id}", "submission"), ...dyn("/threads/{id}", "thread")],
  teacher: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/observability", "/settings/llm", "/settings/integrations", "/settings", ...dyn("/students/{id}", "student"), ...dyn("/exams/{id}", "exam"), ...dyn("/exams/{id}/results", "exam"), ...dyn("/grading/{id}", "submission"), ...dyn("/threads/{id}", "thread")],
  admin: ["/", "/observability", "/admin/courses", "/admin/users", "/settings/llm", "/settings/integrations", "/settings"],
};
const USER: Record<string, string> = { student: "sv.kha", ta: "ta", teacher: "teacher", admin: "admin" };

async function login(browser: Browser, role: string, w: number, h: number) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${role === "student" && STATE === "empty" ? "sv31" : USER[role]}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  if (role !== "admin") {
    const tb = page.locator("[data-part=topbar]");
    if (!(await tb.innerText()).includes(CLASS)) {
      await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click();
      await page.getByText(CLASS, { exact: false }).first().click();
      await page.waitForTimeout(1500);
    }
  }
  return { ctx, page };
}

async function probe(page: Page) {
  return page.evaluate((cls) => {
    const txt = (e: Element | null) => (e?.textContent ?? "").replace(/\s+/g, " ").trim();
    const vh = window.innerHeight;
    const tok = (e: Element, suffix: string) => e.className.toString().split(/\s+/).some((t) => t === suffix || t.endsWith("__" + suffix));
    const all = (root: ParentNode, p: (e: Element) => boolean) => Array.from(root.querySelectorAll("*")).filter(p);
    const R = (e: Element) => { const r = e.getBoundingClientRect(); return { x: Math.round(r.left), y: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height), r: Math.round(r.right), b: Math.round(r.bottom) }; };
    const main = document.querySelector("main");
    if (!main) return { noMain: true, url: location.pathname };
    const mr = R(main);
    const pageEl = all(main, (e) => tok(e, "page"))[0] ?? null;
    const kind = pageEl ? (["reading", "wide", "full"].find((k) => tok(pageEl, k)) ?? "?") : "none";
    const pr = pageEl ? R(pageEl) : null;
    // panel cấp cao nhất
    const panels = Array.from(main.querySelectorAll("[data-ep-panel]")).filter((p) => !p.parentElement?.closest("[data-ep-panel]")).map((p) => {
      const r = R(p); return { w: r.w, h: r.h, gapL: r.x - mr.x, gapR: mr.r - r.r, hVsVp: +(r.h / vh).toFixed(2) };
    });
    const h1 = document.querySelector("h1[data-part=page-title]");
    const headText = h1?.parentElement ?? null;
    const headRow = headText?.parentElement ?? null;
    const actions = headRow ? Array.from(headRow.children).find((c) => c !== headText) ?? null : null;
    const h1r = h1 ? R(h1) : null;
    const ar = actions ? R(actions) : null;
    const lineH = (e: Element) => parseFloat(getComputedStyle(e).lineHeight) || 20;
    const metaEl = headText ? all(headText, (e) => tok(e, "meta"))[0] ?? null : null;
    const descEl = headText ? all(headText, (e) => tok(e, "desc"))[0] ?? null : null;
    const headerBtns = actions ? Array.from(actions.querySelectorAll("button,a")).filter((b) => (b as HTMLElement).offsetParent) : [];
    const primaries = Array.from(main.querySelectorAll("[data-variant=primary]")).filter((b) => (b as HTMLElement).offsetParent);
    const secDesc = all(main, (e) => tok(e, "sectionDesc")).map((e) => txt(e));
    const emptyBlocks = all(main, (e) => tok(e, "empty") && !!e.querySelector(".ep-item-title"));
    const empties = emptyBlocks.map((e) => {
      const r = R(e); const par = e.closest("[data-ep-panel]") ?? pageEl ?? main; const q = R(par);
      return {
        title: txt(e.querySelector(".ep-item-title")), icon: !!all(e, (x) => tok(x, "emptyIcon")).length, bodies: all(e, (x) => tok(x, "emptyText")).map((x) => txt(x)),
        actions: Array.from(e.querySelectorAll("button,a")).map((b) => ({ t: txt(b), v: (b as HTMLElement).getAttribute("data-variant") ?? "link" })),
        cxOff: Math.round(r.x + r.w / 2 - (q.x + q.w / 2)), topInParent: r.y - q.y, w: r.w, parentH: q.h, parentVsVp: +(q.h / vh).toFixed(2), align: getComputedStyle(e).justifyItems,
      };
    });
    const label = (b: Element) => txt(b) || (b.getAttribute("aria-label") ?? "");
    const mbtns = Array.from(main.querySelectorAll("button,a")).filter((b) => (b as HTMLElement).offsetParent).map(label).filter((t) => t && t.length < 40 && !/^\d+$/.test(t));
    const cnt: Record<string, number> = {};
    mbtns.forEach((t) => (cnt[t] = (cnt[t] || 0) + 1));
    const dup = Object.entries(cnt).filter(([, n]) => n > 1).map(([t, n]) => `${t}×${n}`);
    const emptyActTexts = empties.flatMap((e) => e.actions.map((a) => a.t));
    const hdrTexts = headerBtns.map(label);
    return {
      url: location.pathname, title: txt(h1), kind, pageW: pr?.w ?? null, pageX: pr?.x ?? null, mainW: mr.w, mainX: mr.x,
      panels: { n: panels.length, list: panels.slice(0, 6) },
      header: {
        h1: h1r, actionsPos: ar && h1r ? (ar.y >= h1r.b - 2 ? "dưới tiêu đề" : "cùng hàng") : "không có", actionsRightGap: ar ? mr.r - ar.r : null,
        btns: headerBtns.map((b) => ({ t: label(b), v: (b as HTMLElement).getAttribute("data-variant") ?? "link" })),
        metaText: txt(metaEl), metaLines: metaEl ? Math.max(1, Math.round(R(metaEl).h / lineH(metaEl))) : 0, metaDupClass: !!metaEl && new RegExp(cls + "|QC lớp rỗng").test(txt(metaEl)),
        descText: txt(descEl), descLines: descEl ? Math.max(1, Math.round(R(descEl).h / lineH(descEl))) : 0,
      },
      primaryTotal: primaries.length, primaryOutsideHeader: primaries.filter((b) => !actions?.contains(b)).map(label),
      sectionDescs: secDesc, empties, dupButtons: dup, dupHeaderEmpty: hdrTexts.filter((t) => emptyActTexts.includes(t)),
      h2: Array.from(main.querySelectorAll("h2")).length, tabs: Array.from(main.querySelectorAll("[role=tab]")).map(txt),
      overflowX: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1, docH: document.documentElement.scrollHeight,
      forbidden: /Không tìm thấy trang|không có quyền/i.test(document.body.innerText), busy: main.querySelectorAll("[aria-busy=true]").length,
    };
  }, CLASS);
}

for (const role of Object.keys(ROUTES)) {
  if (role === "admin" && STATE === "empty") continue; // admin không gắn lớp: một trạng thái
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    test(`chụp ${role} ${STATE} @${w}`, async ({ browser }, ti) => {
      test.skip(ti.project.name !== "desktop" || (!!ONLY && ONLY !== role));
      test.setTimeout(900_000);
      const { ctx, page } = await login(browser, role, w, h);
      for (const route of ROUTES[role]) {
        let rec: Record<string, unknown> = { role, state: STATE, w, route };
        try {
          await page.goto(`${BASE}${route}`, { waitUntil: "domcontentloaded" });
          await page.waitForTimeout(1300);
          for (let i = 0; i < 14 && (await page.locator("main [aria-busy=true]").count()) > 0; i++) await page.waitForTimeout(500);
          await page.waitForTimeout(600);
          rec = { ...rec, ...(await probe(page)) };
          const f = `${role}-${STATE}-${(route.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "id").replace(/\W+/g, "_") || "home")}-${w}.jpg`;
          await page.screenshot({ path: `${OUT}/${f}`, type: "jpeg", quality: 55 });
          rec.img = f;
        } catch (e) { rec.err = String(e).slice(0, 160); }
        appendFileSync(LOG, JSON.stringify(rec) + "\n");
      }
      await ctx.close();
    });
  }
}
