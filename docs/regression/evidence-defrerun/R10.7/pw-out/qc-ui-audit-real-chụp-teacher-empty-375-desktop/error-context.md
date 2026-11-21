# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: qc-ui-audit-real.spec.ts >> chụp teacher empty @375
- Location: e2e/qc-ui-audit-real.spec.ts:170:7

# Error details

```
Error: locator.click: Target page, context or browser has been closed
Call log:
  - waiting for getByText('QCEMPTYr107').first()

```

# Test source

```ts
  1   | import { test, type Browser, type Page } from "@playwright/test";
  2   | import { mkdirSync, appendFileSync } from "fs";
  3   | 
  4   | test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
  5   | // QC 10.7 (UI-R) v2 — chụp MỌI route × vai × 1440/375 × (dữ liệu | rỗng | đang tải | lỗi) và đo DOM theo móc `data-ep-*` (P-107-7).
  6   | // Tiêu chí theo techlead.md 1.2 (W1–W5, H1–H8, P1–P5, E1–E7, L1–L3, X1–X3, D1–D3, T1–T2, C3). CHỈ ĐỌC.
  7   | // Env: QC_STATE = data | empty | loading | error ; QC_CLASS = 761987 | QCEMPTY ; QC_PHASE = sau (thư mục audit/<phase>) ; QC_DYN = JSON id route động.
  8   | const PW = process.env.QC_PW ?? "";
  9   | const BASE = process.env.QC_BASE ?? "http://localhost:580";
  10  | const OUT = process.env.QC_OUT ?? "/tmp/qc-audit";
  11  | const CLASS = process.env.QC_CLASS ?? "761987";
  12  | const STATE = process.env.QC_STATE ?? "data";
  13  | const PHASE = process.env.QC_PHASE ?? "sau";
  14  | const ONLY = process.env.QC_ROLE ?? "";
  15  | const DYN = JSON.parse(process.env.QC_DYN ?? "{}") as Record<string, string>;
  16  | const DIR = `${OUT}/${PHASE}`;
  17  | mkdirSync(DIR, { recursive: true });
  18  | const LOG = `${DIR}/audit-${STATE}.jsonl`;
  19  | 
  20  | const dyn = (p: string, k: string) => (DYN[k] ? [p.replace("{id}", DYN[k])] : []);
  21  | const RF = process.env.QC_ROUTES ? new RegExp(process.env.QC_ROUTES) : null; // lọc route theo lô, vd. ^/(insights|questions|inbox)$
  22  | const ROUTES: Record<string, string[]> = {
  23  |   student: ["/", "/chat", "/threads", "/practice", "/practice/history", "/exams", "/library", "/calendar", "/me", "/settings", ...dyn("/assignments/{id}", "assignment"), ...dyn("/threads/{id}", "thread"), ...dyn("/library/{id}", "doc")],
  24  |   ta: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/settings", ...dyn("/students/{id}", "student"), ...dyn("/exams/{id}", "exam"), ...dyn("/grading/{id}", "submission"), ...dyn("/threads/{id}", "thread")],
  25  |   teacher: ["/", "/inbox", "/students", "/attendance", "/gradebook", "/gradebook/scheme", "/grading", "/questions", "/exams", "/threads", "/documents", "/calendar", "/insights", "/analytics", "/class/members", "/class/settings", "/observability", "/settings/llm", "/settings/integrations", "/settings", ...dyn("/students/{id}", "student"), ...dyn("/exams/{id}", "exam"), ...dyn("/exams/{id}/results", "exam"), ...dyn("/grading/{id}", "submission"), ...dyn("/threads/{id}", "thread")],
  26  |   admin: ["/", "/observability", "/admin/courses", "/admin/users", "/settings/llm", "/settings/integrations", "/settings"],
  27  | };
  28  | for (const k of Object.keys(ROUTES)) if (RF) ROUTES[k] = ROUTES[k].filter((r) => RF.test(r.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "id")));
  29  | const USER: Record<string, string> = { student: "sv.kha", ta: "ta", teacher: "teacher", admin: "admin" };
  30  | // đang tải / lỗi: bố cục không phụ thuộc vai → một vai mỗi route (TL-1 1.1)
  31  | const STUDENT_ONLY = (r: string) => /^\/(chat|practice|library|me|assignments)(\/|$)/.test(r);
  32  | const ADMIN_ONLY = (r: string) => r.startsWith("/admin/");
  33  | const roleFor = (r: string) => (ADMIN_ONLY(r) ? "admin" : STUDENT_ONLY(r) ? "student" : "teacher");
  34  | 
  35  | async function login(browser: Browser, role: string, w: number, h: number) {
  36  |   const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  37  |   await ctx.addInitScript(() => {
  38  |     (window as unknown as { __cls: number }).__cls = 0;
  39  |     try { new PerformanceObserver((l) => { for (const e of l.getEntries() as unknown as { value: number; hadRecentInput: boolean }[]) if (!e.hadRecentInput) (window as unknown as { __cls: number }).__cls += e.value; }).observe({ type: "layout-shift", buffered: true }); } catch { /* không hỗ trợ */ }
  40  |   });
  41  |   const page = await ctx.newPage();
  42  |   await page.goto(`${BASE}/login`);
  43  |   await page.locator("input[type=email]").fill(`${role === "student" && STATE === "empty" ? "sv31" : USER[role]}@edupilot.local`);
  44  |   await page.locator("input[type=password]").fill(PW);
  45  |   await page.keyboard.press("Enter");
  46  |   await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  47  |   await page.locator("[data-part=topbar]").waitFor();
  48  |   if (role !== "admin") {
  49  |     const tb = page.locator("[data-part=topbar]");
  50  |     if (!(await tb.innerText()).includes(CLASS)) {
  51  |       await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click();
> 52  |       await page.getByText(CLASS, { exact: false }).first().click();
      |                                                             ^ Error: locator.click: Target page, context or browser has been closed
  53  |       await page.waitForTimeout(1500);
  54  |     }
  55  |   }
  56  |   return { ctx, page };
  57  | }
  58  | 
  59  | async function probe(page: Page) {
  60  |   return page.evaluate(([cls, vw]) => {
  61  |     const txt = (e: Element | null) => (e?.textContent ?? "").replace(/\s+/g, " ").trim();
  62  |     const R = (e: Element) => { const r = e.getBoundingClientRect(); return { x: +r.left.toFixed(1), y: +r.top.toFixed(1), w: +r.width.toFixed(1), h: +r.height.toFixed(1), r: +r.right.toFixed(1), b: +r.bottom.toFixed(1) }; };
  63  |     const vis = (e: Element) => !!(e as HTMLElement).offsetParent || getComputedStyle(e).position === "fixed";
  64  |     const lh = (e: Element) => parseFloat(getComputedStyle(e).lineHeight) || 20;
  65  |     const main = document.querySelector("main");
  66  |     if (!main) return { noMain: true, url: location.pathname };
  67  |     const pages = Array.from(main.querySelectorAll("[data-ep-page]"));
  68  |     const pg = pages[0] ?? null;
  69  |     let col: { x: number; r: number; w: number } | null = null;
  70  |     if (pg) { const r = R(pg); const cs = getComputedStyle(pg); const pl = parseFloat(cs.paddingLeft), pr = parseFloat(cs.paddingRight); col = { x: +(r.x + pl).toFixed(1), r: +(r.r - pr).toFixed(1), w: +(r.w - pl - pr).toFixed(1) }; }
  71  |     const topPanels = Array.from(main.querySelectorAll("[data-ep-panel]")).filter((p) => !p.parentElement?.closest("[data-ep-panel]"));
  72  |     const panels = topPanels.slice(0, 8).map((p) => {
  73  |       const r = R(p); const cs = getComputedStyle(p);
  74  |       const kids = Array.from(p.children).filter((c) => (c as HTMLElement).offsetHeight > 0);
  75  |       const contentH = kids.length ? Math.max(...kids.map((k) => k.getBoundingClientRect().bottom)) - Math.min(...kids.map((k) => k.getBoundingClientRect().top)) + parseFloat(cs.paddingTop) + parseFloat(cs.paddingBottom) : r.h;
  76  |       return { x: r.x, w: r.w, h: r.h, gapL: col ? +(r.x - col.x).toFixed(1) : null, gapR: col ? +(col.r - r.r).toFixed(1) : null, pad: [cs.paddingTop, cs.paddingLeft].map((v) => parseFloat(v)), extra: +(r.h - contentH).toFixed(1) };
  77  |     });
  78  |     const nested = main.querySelectorAll("[data-ep-panel] [data-ep-panel]").length;
  79  |     const h1s = Array.from(document.querySelectorAll("main h1"));
  80  |     const hdr = main.querySelector("[data-ep-page-header]");
  81  |     const h1 = hdr?.querySelector("h1") ?? h1s[0] ?? null;
  82  |     const actions = hdr?.querySelector("[data-ep-header-actions]") ?? null;
  83  |     const meta = hdr?.querySelector("[data-ep-header-meta]") ?? null;
  84  |     const hdrKids = hdr ? Array.from(hdr.querySelectorAll("*")).filter((e) => e.tagName === "P" && !meta?.contains(e) && !actions?.contains(e)).length : 0;
  85  |     const topbar = txt(document.querySelector("[data-part=topbar]"));
  86  |     const classTokens = [cls, "QC lớp rỗng", "An ninh mạng"].filter(Boolean) as string[];
  87  |     const h1r = h1 ? R(h1) : null; const ar = actions ? R(actions) : null; const hr = hdr ? R(hdr) : null;
  88  |     const hasBack = !!hdr && Array.from(hdr.querySelectorAll("a")).some((a) => vis(a) && !actions?.contains(a) && !a.closest("[data-ep-header-meta]") && h1 !== undefined && R(a).y + R(a).h <= (h1 ? R(h1).y + 2 : 0)); // hàng quay lại (TL 14.5)
  89  |     const hbtns = actions ? Array.from(actions.querySelectorAll("button,a")).filter(vis) : [];
  90  |     const variantOf = (b: Element) => b.getAttribute("data-variant") ?? "link";
  91  |     const primOut = Array.from(main.querySelectorAll("[data-variant=primary]")).filter((b) => vis(b) && !actions?.contains(b) && !b.closest("[data-ep-header-dock]") && !b.closest("form") && !b.closest("[data-ep-action-list]")); // P-107-14 dock; TL 14.1: nút gửi của form và hàng đầu ActionList là ngoại lệ
  92  |     const formPrim = Math.max(0, ...Array.from(main.querySelectorAll("form")).map((f) => Array.from(f.querySelectorAll("[data-variant=primary]")).filter(vis).length));
  93  |     const alPrim = Array.from(main.querySelectorAll("[data-ep-action-list] [data-variant=primary]")).filter(vis).length;
  94  |     const alCount = main.querySelectorAll("[data-ep-action-list]").length;
  95  |     const sections = Array.from(main.querySelectorAll("[data-ep-section]"));
  96  |     const secGaps = sections.slice(1).map((s, i) => +(R(s).y - R(sections[i]).b).toFixed(1));
  97  |     const headToPanel = sections.slice(0, 4).map((s) => { const hd = s.querySelector("[data-ep-section-head]"); const pn = s.querySelector("[data-ep-panel]"); return hd && pn ? +(R(pn).y - R(hd).b).toFixed(1) : null; });
  98  |     let nxt: Element | null = hdr ? hdr.nextElementSibling : null; while (nxt && !((nxt as HTMLElement).offsetHeight > 0 && nxt.getBoundingClientRect().top >= (hdr ? R(hdr).b : 0) - 0.5)) nxt = nxt.nextElementSibling;
  99  |     const hdrToBody = hdr && nxt ? +(R(nxt).y - R(hdr).b).toFixed(1) : null;
  100 |     const h2s = Array.from(main.querySelectorAll("h2"));
  101 |     const tabs = Array.from(main.querySelectorAll("[role=tab]")).map(txt);
  102 |     const toolbarBtns = Array.from(main.querySelectorAll("[data-ep-toolbar] button, [data-ep-toolbar] a")).filter(vis).map((b) => txt(b) || b.getAttribute("aria-label") || "");
  103 |     const empties = Array.from(main.querySelectorAll("[data-ep-empty]")).map((e) => {
  104 |       // P-11-52: khối rỗng trong Split đo theo ngăn con chứa nó (như W3/P5), không theo cả thẻ Split (gồm rail «Phiên trước»)
  105 |       const sp = e.closest("[data-ep-split]"); const pane = sp ? Array.from(sp.children).find((c) => c.contains(e)) : null;
  106 |       const r = R(e); const par = pane ?? e.closest("[data-ep-panel]") ?? pg ?? main; const pr = R(par);
  107 |       const pEls = Array.from(e.querySelectorAll("p")); const titleEl = e.querySelector(".ep-item-title") ?? pEls[0] ?? null;
  108 |       const btns = Array.from(e.querySelectorAll("button,a"));
  109 |       const svgs = Array.from(e.querySelectorAll("svg")).filter((s) => !s.closest("button,a"));
  110 |       const cxc = par === pg && col ? (col.x + col.r) / 2 : pr.x + pr.w / 2;
  111 |       return { title: txt(titleEl), titleLen: txt(titleEl).length, titleLines: titleEl ? Math.max(1, Math.round(R(titleEl).h / lh(titleEl))) : 0, pCount: pEls.length, svg: svgs.length, extraText: Math.max(0, pEls.length - 1),
  112 |         actions: btns.map((b) => ({ t: txt(b), v: variantOf(b) })), cxOff: +((r.x + r.w / 2) - cxc).toFixed(1), parentH: pr.h, inPanel: !!e.closest("[data-ep-panel]") };
  113 |     });
  114 |     const hdrLabels = hbtns.map((b) => txt(b) || b.getAttribute("aria-label") || "");
  115 |     const emptyLabels = empties.flatMap((e) => e.actions.map((a) => a.t));
  116 |     const dupEmpty = emptyLabels.filter((t) => hdrLabels.includes(t) || tabs.includes(t) || toolbarBtns.includes(t));
  117 |     const alerts = Array.from(main.querySelectorAll("[role=alert]")).map((a) => ({ w: R(a).w, text: txt(a).slice(0, 160), retry: !!Array.from(a.querySelectorAll("button")).find((b) => /Thử lại/.test(txt(b))), bare: /\b[45]\d\d\b|undefined|Error:|\[object/.test(txt(a)) }));
  118 |     // D: cỡ chữ, cao điều khiển, cao hàng bảng
  119 |     const fonts = new Set<number>();
  120 |     const walker = document.createTreeWalker(main, NodeFilter.SHOW_TEXT);
  121 |     for (let n = walker.nextNode(); n; n = walker.nextNode()) { const el = n.parentElement; if (el && (n.nodeValue ?? "").trim() && vis(el)) fonts.add(Math.round(parseFloat(getComputedStyle(el).fontSize) * 10) / 10); }
  122 |     const ctrlH = Array.from(main.querySelectorAll("button[data-variant], input:not([type=checkbox]):not([type=radio]):not([type=hidden]):not([type=file]), select")).filter(vis).map((c) => Math.round(R(c).h));
  123 |     const rowH = Array.from(main.querySelectorAll("tbody tr")).filter(vis).map((c) => Math.round(R(c).h));
  124 |     const icons = Array.from(main.querySelectorAll("svg.lucide-info, svg.lucide-circle-help")).filter((s) => !s.closest("[role=status],[role=alert]")).length;
  125 |     const long: Record<string, number> = {}; Array.from(main.querySelectorAll("p,span,li,td,div")).filter((e) => e.children.length === 0 && !e.closest("[role=tooltip]")).forEach((e) => { const t = txt(e); /* P-107-15: chữ của [role=tooltip] không đếm T2 */ if (t.length > 30) long[t] = (long[t] || 0) + 1; });
  126 |     const repeats = Object.entries(long).filter(([, n]) => n >= 3).map(([t, n]) => `${t.slice(0, 50)}×${n}`);
  127 |     const yellow = (rgb: string) => /rgb\(243, 193, 8\)|rgba\(243, 193, 8/.test(rgb);
  128 |     let yBg = 0, yFg = 0; Array.from(main.querySelectorAll("*")).slice(0, 1500).forEach((e) => { const cs = getComputedStyle(e); if (yellow(cs.backgroundColor)) yBg++; if (yellow(cs.color)) yFg++; });
  129 |     const metaText = txt(meta);
  130 |     return {
  131 |       url: location.pathname, title: txt(h1), vw,
  132 |       pageCount: pages.length, kind: pg?.getAttribute("data-ep-page") ?? "none", col, split: !!main.querySelector("[data-ep-split]"), splitFill: !!main.querySelector("[data-ep-split-fill]"),
  133 |       panels: { n: topPanels.length, nested, list: panels },
  134 |       header: { h1Count: h1s.length, h1InHeader: !!hdr && !!hdr.querySelector("h1"), pTags: hdrKids, h1: h1r, h1x: h1r?.x ?? null, hdrH: hr?.h ?? null, hasBack, hasMeta: !!meta, metaText, metaLines: meta ? Math.max(1, Math.round(R(meta).h / lh(meta))) : 0,
  135 |         metaDupClass: !!meta && classTokens.some((t) => metaText.includes(t) && topbar.includes(t)),
  136 |         dy: h1r && ar ? +((ar.y + ar.h / 2) - (h1r.y + h1r.h / 2)).toFixed(1) : null, actionsRight: ar && ar.w > 0 && col ? +(col.r - ar.r).toFixed(1) : null, actionsH: ar?.h ?? null, btns: hbtns.map((b) => ({ t: txt(b) || b.getAttribute("aria-label") || "", v: variantOf(b) })) },
  137 |       primaryTotal: primOut.length + hbtns.filter((b) => variantOf(b) === "primary").length, alertSecMax: Math.max(0, ...Array.from(main.querySelectorAll("[data-ep-section]")).map((x) => x.querySelectorAll("[role=alert]").length)), primaryOutside: primOut.map(txt), dock: Array.from(document.querySelectorAll("[data-ep-header-dock]")).filter(vis).length, formPrim, alPrim, alCount,
  138 |       sections: sections.length, h2: h2s.length, h2NoHook: h2s.filter((h) => !h.hasAttribute("data-ep-section-title")).length, sectionP: main.querySelectorAll("[data-ep-section-head] p").length,
  139 |       secGaps, headToPanel, hdrToBody, tabs, toolbarBtns,
  140 |       empties, dupEmpty, alerts, alertCount: main.querySelectorAll("[role=alert]").length, busy: main.querySelectorAll("[aria-busy=true]").length,
  141 |       fonts: Array.from(fonts).sort((a, b) => a - b), ctrlH: Array.from(new Set(ctrlH)).sort((a, b) => a - b), rowH: Array.from(new Set(rowH)), icons, repeats, yellowBg: yBg, yellowFg: yFg,
  142 |       overflowX: document.documentElement.scrollWidth - window.innerWidth, docH: document.documentElement.scrollHeight,
  143 |       forbidden: /Không tìm thấy trang|không có quyền/i.test(document.body.innerText), cls: (window as unknown as { __cls?: number }).__cls ?? 0,
  144 |     };
  145 |   }, [CLASS, page.viewportSize()?.width ?? 0] as const);
  146 | }
  147 | 
  148 | async function guides(page: Page, kind: string, w: number) {
  149 |   const [l, r] = w < 500 ? [12, 363] : kind === "read" ? [428, 1228] : [240, 1416];
  150 |   await page.evaluate(([a, b]) => { for (const x of [a, b]) { const d = document.createElement("div"); d.setAttribute("data-qc-guide", ""); d.style.cssText = `position:fixed;top:0;left:${x}px;width:1px;height:100vh;background:rgba(0,110,255,.75);z-index:2147483647;pointer-events:none`; document.body.appendChild(d); } }, [l, r]);
  151 | }
  152 | 
```