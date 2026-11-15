import { test, type Browser, type Page } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 — P-107-14: nút chính LÀ thanh dính đáy dưới 720 px (`data-ep-header-dock`), đầu trang không vẽ nút chính. /attendance, /grading/[id].
const PW = process.env.QC_PW ?? "";
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-dock";
mkdirSync(OUT, { recursive: true });
const ok = (n: string, v: boolean, d = "") => console.log(`QCAC ${n} ${v ? "PASS" : "FAIL"} ${d}`);

async function login(browser: Browser, w: number, h: number) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500, ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  const tb = page.locator("[data-part=topbar]");
  if (!(await tb.innerText()).includes("761987")) { await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click(); await page.getByText("761987").first().click(); await page.waitForTimeout(1500); }
  return { ctx, page };
}

const measure = (page: Page) => page.evaluate(() => {
  const vis = (e: Element) => (e as HTMLElement).offsetParent !== null || getComputedStyle(e).position === "fixed";
  const dock = [...document.querySelectorAll<HTMLElement>("[data-ep-header-dock]")].filter(vis)[0];
  const hdr = document.querySelector("[data-ep-page-header]");
  const hp = [...(hdr?.querySelectorAll<HTMLElement>("[data-variant=primary]") ?? [])].filter(vis);
  const dp = [...(dock?.querySelectorAll<HTMLElement>("[data-variant=primary]") ?? [])].filter(vis);
  const dr = dock?.getBoundingClientRect();
  const last = [...document.querySelectorAll<HTMLElement>("main *")].filter((e) => e.children.length === 0 && e.textContent?.trim() && !dock?.contains(e)).map((e) => e.getBoundingClientRect().bottom + scrollY).sort((a, b) => b - a)[0];
  return { dock: !!dock, pos: dock ? getComputedStyle(dock).position : null, bottomGap: dr ? Math.round(innerHeight - dr.bottom) : null, navH: (() => { const e = document.elementFromPoint(innerWidth / 2, innerHeight - 4); let n: Element | null = e; while (n && getComputedStyle(n).position !== "fixed") n = n.parentElement; return n ? Math.round(innerHeight - n.getBoundingClientRect().top) : 0; })(), h: dr ? Math.round(dr.height) : null, dockBtn: dp.map((b) => ({ t: b.textContent?.trim().slice(0, 24), h: Math.round(b.getBoundingClientRect().height), w: Math.round(b.getBoundingClientRect().width) })), headerPrimary: hp.map((b) => b.textContent?.trim().slice(0, 24)), lastTextBottom: Math.round(last ?? 0), docH: document.documentElement.scrollHeight, vh: innerHeight, overflowX: document.documentElement.scrollWidth > innerWidth };
});

test("thanh dính đáy 375 / 1440", async ({ browser }) => {
  test.setTimeout(240_000);
  const { ctx, page } = await login(browser, 375, 812);
  await page.goto(`${BASE}/attendance`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(1500);
  const a = await measure(page);
  console.log("QCINFO attendance@375 " + JSON.stringify(a));
  ok("P14-att375-dock", a.dock && a.dockBtn.length === 1 && a.headerPrimary.length === 0, `dock=${a.dock}; nút chính trong dock=${JSON.stringify(a.dockBtn)}; nút primary ở đầu trang=${JSON.stringify(a.headerPrimary)}`);
  ok("P14-att375-sticky", (a.pos === "sticky" || a.pos === "fixed") && a.bottomGap !== null && Math.abs(a.bottomGap - a.navH) <= 1, `position=${a.pos}; cách đáy khung nhìn=${a.bottomGap}px = cao thanh điều hướng dưới (${a.navH}px); cao ${a.h}`);
  ok("P14-att375-touch", a.dockBtn.every((b) => b.h >= 44), `cao nút ${a.dockBtn.map((b) => b.h)}`);
  await page.evaluate(() => scrollTo(0, 400)); await page.waitForTimeout(300);
  const a2 = await measure(page);
  ok("P14-att375-scroll", a2.dock && a2.bottomGap !== null && Math.abs(a2.bottomGap - a2.navH) <= 1, `sau cuộn cách đáy ${a2.bottomGap}px (thanh dưới ${a2.navH}px)`);
  await page.evaluate(() => scrollTo(0, document.documentElement.scrollHeight)); await page.waitForTimeout(300);
  const a3 = await page.evaluate(() => { const d = [...document.querySelectorAll<HTMLElement>("[data-ep-header-dock]")][0]; const dt = d.getBoundingClientRect().top; const rows = [...document.querySelectorAll<HTMLElement>("main *")].filter((e) => e.children.length === 0 && e.textContent?.trim() && !d.contains(e) && getComputedStyle(e).visibility !== "hidden").map((e) => e.getBoundingClientRect().bottom); return { dockTop: Math.round(dt), maxTextBottom: Math.round(Math.max(...rows.filter((b) => b < innerHeight + 5))) , covered: rows.filter((b) => b > dt + 2 && b <= innerHeight + 2).length }; });
  ok("P14-att375-nocover", a3.covered === 0 || a3.maxTextBottom <= a3.dockTop + 2, `cuối trang: đỉnh dock ${a3.dockTop}; số dòng chữ nằm sau dock ${a3.covered}`);
  await page.screenshot({ path: `${OUT}/attendance-375-bottom.png` });
  await page.evaluate(() => scrollTo(0, 0)); await page.waitForTimeout(200);
  await page.screenshot({ path: `${OUT}/attendance-375-top.png` });
  // /grading/[id] 375
  await page.goto(`${BASE}/grading`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(1500);
  const href = await page.locator("a[href^='/grading/']").first().getAttribute("href").catch(() => null);
  if (href) {
    await page.goto(`${BASE}${href}`, { waitUntil: "domcontentloaded" }); await page.waitForTimeout(2000);
    const g = await measure(page);
    console.log("QCINFO grading-detail@375 " + JSON.stringify(g));
    ok("P14-rev375-dock", g.dock && g.headerPrimary.length === 0, `dock=${g.dock}; nút trong dock=${JSON.stringify(g.dockBtn)}; primary ở đầu trang=${JSON.stringify(g.headerPrimary)}`);
    ok("P14-rev375-sticky", (g.pos === "sticky" || g.pos === "fixed") && Math.abs((g.bottomGap ?? 99) - g.navH) <= 1, `position=${g.pos}; cách đáy=${g.bottomGap}px (thanh dưới ${g.navH}px)`);
    ok("P14-rev375-touch", g.dockBtn.every((b) => b.h >= 44), `cao ${g.dockBtn.map((b) => b.h)}`);
    await page.evaluate(() => scrollTo(0, 600)); await page.waitForTimeout(300);
    const g2 = await measure(page); ok("P14-rev375-scroll", g2.dock && Math.abs((g2.bottomGap ?? 99) - g2.navH) <= 1, `sau cuộn ${g2.bottomGap}px (thanh dưới ${g2.navH}px)`);
    await page.screenshot({ path: `${OUT}/review-375.png` });
  } else ok("P14-rev375-dock", false, "không có bài để mở /grading/[id]");
  await ctx.close();
  const d = await login(browser, 1440, 900);
  await d.page.goto(`${BASE}/attendance`, { waitUntil: "domcontentloaded" }); await d.page.waitForTimeout(1500);
  const x = await measure(d.page);
  console.log("QCINFO attendance@1440 " + JSON.stringify(x));
  ok("P14-att1440", x.headerPrimary.length <= 1 && x.dockBtn.length <= 1 && !(x.headerPrimary.length === 1 && x.dockBtn.length === 1 && x.dockBtn[0].t === x.headerPrimary[0]), `đầu trang primary=${JSON.stringify(x.headerPrimary)}; dock=${JSON.stringify(x.dockBtn)} (không hai nút cùng việc)`);
  await d.page.screenshot({ path: `${OUT}/attendance-1440.png` });
  await d.ctx.close();
});
