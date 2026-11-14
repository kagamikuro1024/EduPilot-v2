import { test, type Browser, type Page } from "@playwright/test";
import { mkdirSync, appendFileSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 (UI-R) v2 — chụp MỌI route × vai × 1440/375 × (dữ liệu | rỗng | đang tải | lỗi) và đo DOM theo móc `data-ep-*` (P-107-7).
// Tiêu chí theo techlead.md 1.2 (W1–W5, H1–H8, P1–P5, E1–E7, L1–L3, X1–X3, D1–D3, T1–T2, C3). CHỈ ĐỌC.
// Env: QC_STATE = data | empty | loading | error ; QC_CLASS = 761987 | QCEMPTY ; QC_PHASE = sau (thư mục audit/<phase>) ; QC_DYN = JSON id route động.
const PW = process.env.QC_PW ?? "";
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-audit";
const CLASS = process.env.QC_CLASS ?? "761987";
const STATE = process.env.QC_STATE ?? "data";
const PHASE = process.env.QC_PHASE ?? "sau";
const ONLY = process.env.QC_ROLE ?? "";
const DYN = JSON.parse(process.env.QC_DYN ?? "{}") as Record<string, string>;
const DIR = `${OUT}/${PHASE}`;
mkdirSync(DIR, { recursive: true });
const LOG = `${DIR}/audit-${STATE}.jsonl`;

const dyn = (p: string, k: string) => (DYN[k] ? [p.replace("{id}", DYN[k])] : []);
const ROUTES: Record<string, string[]> = {
  student: ["/", "/chat", "/threads", "/practice", "/practice/history", "/exams", "/library", "/calendar", "/me", "/settings", ...dyn("/assignments/{id}", "assignment"), ...dyn("/threads/{id}", "thread"), ...dyn("/library/{id}", "doc")],
  ta: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/settings", ...dyn("/students/{id}", "student"), ...dyn("/exams/{id}", "exam"), ...dyn("/grading/{id}", "submission"), ...dyn("/threads/{id}", "thread")],
  teacher: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/observability", "/settings/llm", "/settings/integrations", "/settings", ...dyn("/students/{id}", "student"), ...dyn("/exams/{id}", "exam"), ...dyn("/exams/{id}/results", "exam"), ...dyn("/grading/{id}", "submission"), ...dyn("/threads/{id}", "thread")],
  admin: ["/", "/observability", "/admin/courses", "/admin/users", "/settings/llm", "/settings/integrations", "/settings"],
};
const USER: Record<string, string> = { student: "sv.kha", ta: "ta", teacher: "teacher", admin: "admin" };
// đang tải / lỗi: bố cục không phụ thuộc vai → một vai mỗi route (TL-1 1.1)
const STUDENT_ONLY = (r: string) => /^\/(chat|practice|library|me|assignments)(\/|$)/.test(r);
const ADMIN_ONLY = (r: string) => r.startsWith("/admin/");
const roleFor = (r: string) => (ADMIN_ONLY(r) ? "admin" : STUDENT_ONLY(r) ? "student" : "teacher");

async function login(browser: Browser, role: string, w: number, h: number) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript(() => {
    (window as unknown as { __cls: number }).__cls = 0;
    try { new PerformanceObserver((l) => { for (const e of l.getEntries() as unknown as { value: number; hadRecentInput: boolean }[]) if (!e.hadRecentInput) (window as unknown as { __cls: number }).__cls += e.value; }).observe({ type: "layout-shift", buffered: true }); } catch { /* không hỗ trợ */ }
  });
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
  return page.evaluate(([cls, vw]) => {
    const txt = (e: Element | null) => (e?.textContent ?? "").replace(/\s+/g, " ").trim();
    const R = (e: Element) => { const r = e.getBoundingClientRect(); return { x: +r.left.toFixed(1), y: +r.top.toFixed(1), w: +r.width.toFixed(1), h: +r.height.toFixed(1), r: +r.right.toFixed(1), b: +r.bottom.toFixed(1) }; };
    const vis = (e: Element) => !!(e as HTMLElement).offsetParent || getComputedStyle(e).position === "fixed";
    const lh = (e: Element) => parseFloat(getComputedStyle(e).lineHeight) || 20;
    const main = document.querySelector("main");
    if (!main) return { noMain: true, url: location.pathname };
    const pages = Array.from(main.querySelectorAll("[data-ep-page]"));
    const pg = pages[0] ?? null;
    let col: { x: number; r: number; w: number } | null = null;
    if (pg) { const r = R(pg); const cs = getComputedStyle(pg); const pl = parseFloat(cs.paddingLeft), pr = parseFloat(cs.paddingRight); col = { x: +(r.x + pl).toFixed(1), r: +(r.r - pr).toFixed(1), w: +(r.w - pl - pr).toFixed(1) }; }
    const topPanels = Array.from(main.querySelectorAll("[data-ep-panel]")).filter((p) => !p.parentElement?.closest("[data-ep-panel]"));
    const panels = topPanels.slice(0, 8).map((p) => {
      const r = R(p); const cs = getComputedStyle(p);
      const kids = Array.from(p.children).filter((c) => (c as HTMLElement).offsetHeight > 0);
      const contentH = kids.length ? Math.max(...kids.map((k) => k.getBoundingClientRect().bottom)) - Math.min(...kids.map((k) => k.getBoundingClientRect().top)) + parseFloat(cs.paddingTop) + parseFloat(cs.paddingBottom) : r.h;
      return { x: r.x, w: r.w, h: r.h, gapL: col ? +(r.x - col.x).toFixed(1) : null, gapR: col ? +(col.r - r.r).toFixed(1) : null, pad: [cs.paddingTop, cs.paddingLeft].map((v) => parseFloat(v)), extra: +(r.h - contentH).toFixed(1) };
    });
    const nested = main.querySelectorAll("[data-ep-panel] [data-ep-panel]").length;
    const h1s = Array.from(document.querySelectorAll("main h1"));
    const hdr = main.querySelector("[data-ep-page-header]");
    const h1 = hdr?.querySelector("h1") ?? h1s[0] ?? null;
    const actions = hdr?.querySelector("[data-ep-header-actions]") ?? null;
    const meta = hdr?.querySelector("[data-ep-header-meta]") ?? null;
    const hdrKids = hdr ? Array.from(hdr.querySelectorAll("*")).filter((e) => e.tagName === "P" && !meta?.contains(e) && !actions?.contains(e)).length : 0;
    const topbar = txt(document.querySelector("[data-part=topbar]"));
    const classTokens = [cls, "QC lớp rỗng", "An ninh mạng"].filter(Boolean) as string[];
    const h1r = h1 ? R(h1) : null; const ar = actions ? R(actions) : null; const hr = hdr ? R(hdr) : null;
    const hbtns = actions ? Array.from(actions.querySelectorAll("button,a")).filter(vis) : [];
    const variantOf = (b: Element) => b.getAttribute("data-variant") ?? "link";
    const primOut = Array.from(main.querySelectorAll("[data-variant=primary]")).filter((b) => vis(b) && !actions?.contains(b));
    const sections = Array.from(main.querySelectorAll("[data-ep-section]"));
    const secGaps = sections.slice(1).map((s, i) => +(R(s).y - R(sections[i]).b).toFixed(1));
    const headToPanel = sections.slice(0, 4).map((s) => { const hd = s.querySelector("[data-ep-section-head]"); const pn = s.querySelector("[data-ep-panel]"); return hd && pn ? +(R(pn).y - R(hd).b).toFixed(1) : null; });
    const hdrToBody = hdr && hdr.nextElementSibling ? +(R(hdr.nextElementSibling).y - R(hdr).b).toFixed(1) : null;
    const h2s = Array.from(main.querySelectorAll("h2"));
    const tabs = Array.from(main.querySelectorAll("[role=tab]")).map(txt);
    const toolbarBtns = Array.from(main.querySelectorAll("[data-ep-toolbar] button, [data-ep-toolbar] a")).filter(vis).map((b) => txt(b) || b.getAttribute("aria-label") || "");
    const empties = Array.from(main.querySelectorAll("[data-ep-empty]")).map((e) => {
      const r = R(e); const par = e.closest("[data-ep-panel]") ?? pg ?? main; const pr = R(par);
      const pEls = Array.from(e.querySelectorAll("p")); const titleEl = e.querySelector(".ep-item-title") ?? pEls[0] ?? null;
      const btns = Array.from(e.querySelectorAll("button,a"));
      const svgs = Array.from(e.querySelectorAll("svg")).filter((s) => !s.closest("button,a"));
      const cxc = par === pg && col ? (col.x + col.r) / 2 : pr.x + pr.w / 2;
      return { title: txt(titleEl), titleLen: txt(titleEl).length, titleLines: titleEl ? Math.max(1, Math.round(R(titleEl).h / lh(titleEl))) : 0, pCount: pEls.length, svg: svgs.length, extraText: Math.max(0, pEls.length - 1),
        actions: btns.map((b) => ({ t: txt(b), v: variantOf(b) })), cxOff: +((r.x + r.w / 2) - cxc).toFixed(1), parentH: pr.h, inPanel: !!e.closest("[data-ep-panel]") };
    });
    const hdrLabels = hbtns.map((b) => txt(b) || b.getAttribute("aria-label") || "");
    const emptyLabels = empties.flatMap((e) => e.actions.map((a) => a.t));
    const dupEmpty = emptyLabels.filter((t) => hdrLabels.includes(t) || tabs.includes(t) || toolbarBtns.includes(t));
    const alerts = Array.from(main.querySelectorAll("[role=alert]")).map((a) => ({ w: R(a).w, text: txt(a).slice(0, 160), retry: !!Array.from(a.querySelectorAll("button")).find((b) => /Thử lại/.test(txt(b))), bare: /\b[45]\d\d\b|undefined|Error:|\[object/.test(txt(a)) }));
    // D: cỡ chữ, cao điều khiển, cao hàng bảng
    const fonts = new Set<number>();
    const walker = document.createTreeWalker(main, NodeFilter.SHOW_TEXT);
    for (let n = walker.nextNode(); n; n = walker.nextNode()) { const el = n.parentElement; if (el && (n.nodeValue ?? "").trim() && vis(el)) fonts.add(Math.round(parseFloat(getComputedStyle(el).fontSize) * 10) / 10); }
    const ctrlH = Array.from(main.querySelectorAll("button, input:not([type=checkbox]):not([type=radio]):not([type=hidden]), select")).filter(vis).map((c) => Math.round(R(c).h));
    const rowH = Array.from(main.querySelectorAll("tbody tr")).filter(vis).map((c) => Math.round(R(c).h));
    const icons = Array.from(main.querySelectorAll("svg.lucide-info, svg.lucide-circle-help")).filter((s) => !s.closest("[role=status],[role=alert]")).length;
    const long: Record<string, number> = {}; Array.from(main.querySelectorAll("p,span,li,td,div")).filter((e) => e.children.length === 0).forEach((e) => { const t = txt(e); if (t.length > 30) long[t] = (long[t] || 0) + 1; });
    const repeats = Object.entries(long).filter(([, n]) => n >= 3).map(([t, n]) => `${t.slice(0, 50)}×${n}`);
    const yellow = (rgb: string) => /rgb\(243, 193, 8\)|rgba\(243, 193, 8/.test(rgb);
    let yBg = 0, yFg = 0; Array.from(main.querySelectorAll("*")).slice(0, 1500).forEach((e) => { const cs = getComputedStyle(e); if (yellow(cs.backgroundColor)) yBg++; if (yellow(cs.color)) yFg++; });
    const metaText = txt(meta);
    return {
      url: location.pathname, title: txt(h1), vw,
      pageCount: pages.length, kind: pg?.getAttribute("data-ep-page") ?? "none", col, split: !!main.querySelector("[data-ep-split]"), splitFill: !!main.querySelector("[data-ep-split-fill]"),
      panels: { n: topPanels.length, nested, list: panels },
      header: { h1Count: h1s.length, h1InHeader: !!hdr && !!hdr.querySelector("h1"), pTags: hdrKids, h1: h1r, h1x: h1r?.x ?? null, hdrH: hr?.h ?? null, hasMeta: !!meta, metaText, metaLines: meta ? Math.max(1, Math.round(R(meta).h / lh(meta))) : 0,
        metaDupClass: !!meta && classTokens.some((t) => metaText.includes(t) && topbar.includes(t)),
        dy: h1r && ar ? +((ar.y + ar.h / 2) - (h1r.y + h1r.h / 2)).toFixed(1) : null, actionsRight: ar && col ? +(col.r - ar.r).toFixed(1) : null, actionsH: ar?.h ?? null, btns: hbtns.map((b) => ({ t: txt(b) || b.getAttribute("aria-label") || "", v: variantOf(b) })) },
      primaryTotal: primOut.length + hbtns.filter((b) => variantOf(b) === "primary").length, primaryOutside: primOut.map(txt),
      sections: sections.length, h2: h2s.length, h2NoHook: h2s.filter((h) => !h.hasAttribute("data-ep-section-title")).length, sectionP: main.querySelectorAll("[data-ep-section-head] p").length,
      secGaps, headToPanel, hdrToBody, tabs, toolbarBtns,
      empties, dupEmpty, alerts, alertCount: main.querySelectorAll("[role=alert]").length, busy: main.querySelectorAll("[aria-busy=true]").length,
      fonts: Array.from(fonts).sort((a, b) => a - b), ctrlH: Array.from(new Set(ctrlH)).sort((a, b) => a - b), rowH: Array.from(new Set(rowH)), icons, repeats, yellowBg: yBg, yellowFg: yFg,
      overflowX: document.documentElement.scrollWidth - window.innerWidth, docH: document.documentElement.scrollHeight,
      forbidden: /Không tìm thấy trang|không có quyền/i.test(document.body.innerText), cls: (window as unknown as { __cls?: number }).__cls ?? 0,
    };
  }, [CLASS, page.viewportSize()?.width ?? 0] as const);
}

async function guides(page: Page, kind: string, w: number) {
  const [l, r] = w < 500 ? [12, 363] : kind === "read" ? [428, 1228] : [240, 1416];
  await page.evaluate(([a, b]) => { for (const x of [a, b]) { const d = document.createElement("div"); d.setAttribute("data-qc-guide", ""); d.style.cssText = `position:fixed;top:0;left:${x}px;width:1px;height:100vh;background:rgba(0,110,255,.75);z-index:2147483647;pointer-events:none`; document.body.appendChild(d); } }, [l, r]);
}

async function shoot(page: Page, rec: Record<string, unknown>, route: string, w: number) {
  const kind = String(rec.kind ?? "work");
  await guides(page, kind, w);
  const f = `${rec.role}-${STATE}-${(route.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "id").replace(/\W+/g, "_") || "home")}-${w}.jpg`;
  await page.screenshot({ path: `${DIR}/${f}`, type: "jpeg", quality: 55, fullPage: w < 500 && STATE === "data" });
  rec.img = f;
}

async function settle(page: Page, extra = 600) {
  for (let i = 0; i < 14 && (await page.locator("main [aria-busy=true]").count()) > 0; i++) await page.waitForTimeout(500);
  await page.waitForTimeout(extra);
}

if (STATE === "data" || STATE === "empty") {
  for (const role of Object.keys(ROUTES)) {
    if (role === "admin" && STATE === "empty") continue;
    for (const [w, h] of [[1440, 900], [375, 812]] as const) {
      test(`chụp ${role} ${STATE} @${w}`, async ({ browser }, ti) => {
        test.skip(ti.project.name !== "desktop" || (!!ONLY && ONLY !== role));
        test.setTimeout(900_000);
        const { ctx, page } = await login(browser, role, w, h);
        for (const route of ROUTES[role]) {
          let rec: Record<string, unknown> = { role, state: STATE, w, route };
          try {
            await page.goto(`${BASE}${route}`, { waitUntil: "domcontentloaded" });
            await page.waitForTimeout(1300); await settle(page);
            rec = { ...rec, ...(await probe(page)) };
            await shoot(page, rec, route, w);
          } catch (e) { rec.err = String(e).slice(0, 160); }
          appendFileSync(LOG, JSON.stringify(rec) + "\n");
        }
        await ctx.close();
      });
    }
  }
} else {
  // loading | error: một vai mỗi route
  const all = Array.from(new Set(Object.values(ROUTES).flat()));
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    for (const role of ["teacher", "student", "admin"]) {
      test(`chụp ${role} ${STATE} @${w}`, async ({ browser }, ti) => {
        test.skip(ti.project.name !== "desktop" || (!!ONLY && ONLY !== role));
        test.setTimeout(900_000);
        const { ctx, page } = await login(browser, role, w, h);
        let mode: "pass" | "loading" | "error" = "pass";
        await page.route("**/api/v1/**", async (route) => {
          const p = new URL(route.request().url()).pathname;
          if (mode === "pass" || p.startsWith("/api/v1/auth/") || p === "/api/v1/me" || p.startsWith("/api/v1/me/")) return route.continue();
          if (mode === "error") return route.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi máy chủ." }) });
          await new Promise((r) => setTimeout(r, 5000));
          return route.continue();
        });
        for (const route of all.filter((r) => roleFor(r) === role)) {
          let rec: Record<string, unknown> = { role, state: STATE, w, route };
          try {
            mode = "pass"; await page.goto(`${BASE}/`, { waitUntil: "domcontentloaded" }); await page.waitForTimeout(800);
            mode = STATE as "loading" | "error";
            await page.goto(`${BASE}${route}`, { waitUntil: "domcontentloaded" });
            await page.waitForTimeout(STATE === "loading" ? 800 : 2000);
            rec = { ...rec, ...(await probe(page)) };
            await shoot(page, rec, route, w);
          } catch (e) { rec.err = String(e).slice(0, 160); }
          appendFileSync(LOG, JSON.stringify(rec) + "\n");
        }
        await ctx.close();
      });
    }
  }
}
