import { writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { SEED_PW, apiLogin, loadAuditSrc } from "./r4ui-lib";

// S4X: soi chi tiết các FAIL của audit P212-21 (cut/touch ở 375/390, ma trận SV chặn quyền) trước khi phân loại F / lỗi kịch bản.
const OUT = "../docs/regression/evidence-4/logs/s4x/probe.json";
const res: Record<string, unknown> = {};
const CASES: Array<[string, string, string, number, number]> = [
  ["sv.gioi", "/", "SV /@375", 375, 812],
  ["ta", "/students", "TA /students@390", 390, 844],
  ["ta", "/gradebook", "TA /gradebook@390", 390, 844],
  ["teacher", "/students", "GV /students@390", 390, 844],
];
test.setTimeout(240_000);
test("[S4X] probe audit", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  const { TOUCH_SRC } = await loadAuditSrc();
  let who = "";
  for (const [u, route, tag, w, h] of CASES) {
    if (who !== u) { await page.context().clearCookies(); await apiLogin(page, `${u}@edupilot.local`, SEED_PW); who = u; }
    await page.setViewportSize({ width: w, height: h });
    await page.goto(route);
    await page.waitForTimeout(2500);
    res[tag] = await page.evaluate((touchSrc) => {
      const out: Array<Record<string, unknown>> = [];
      for (const e of Array.from(document.querySelectorAll("main *, header *"))) {
        if (e.children.length || !e.textContent?.trim() || e.closest("[data-scroll-x]") || e.closest(".ep-sr-only")) continue;
        if (getComputedStyle(e).textOverflow === "ellipsis" && e.scrollWidth > e.clientWidth && !(e as HTMLElement).title) continue;
        let p = e.parentElement; while (p && getComputedStyle(p).overflowX === "visible") p = p.parentElement;
        if (!p) continue;
        const a = e.getBoundingClientRect(), b = p.getBoundingClientRect();
        if (a.right > b.right + 1 || a.left < b.left - 1) out.push({ tag: e.tagName, text: e.textContent!.trim().slice(0, 40), cls: String((e as HTMLElement).className).slice(0, 60), a: [Math.round(a.left), Math.round(a.right)], parent: p.tagName + "." + String((p as HTMLElement).className).slice(0, 50), pdata: Array.from(p.attributes).map((x) => x.name).join(","), b: [Math.round(b.left), Math.round(b.right)], ovx: getComputedStyle(p).overflowX });
      }
      // eslint-disable-next-line no-new-func
      const touch = (new Function(`return ${touchSrc}`))();
      return { cut: out.slice(0, 6), cutN: out.length, touch };
    }, TOUCH_SRC);
  }
  // SV chặn quyền: văn bản thật của vài route nhân viên
  await page.context().clearCookies(); await apiLogin(page, "sv.gioi@edupilot.local", SEED_PW); await page.setViewportSize({ width: 1440, height: 900 });
  const blocked: Record<string, unknown> = {};
  for (const r of ["/inbox", "/students", "/gradebook", "/documents", "/insights", "/class/members"]) {
    await page.goto(r); await page.waitForTimeout(2500);
    blocked[r] = { url: new URL(page.url()).pathname, h1: (await page.locator("main h1").allInnerTexts()).join("|"), text: (await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 160) };
  }
  res["SV chặn quyền"] = blocked;
  writeFileSync(OUT, JSON.stringify(res, null, 1));
});
